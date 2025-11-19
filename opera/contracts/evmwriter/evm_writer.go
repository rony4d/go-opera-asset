// Package evmwriter implements a precompiled contract that allows the Opera driver contract
// to directly modify EVM state (balances, code, storage, nonces) during event processing.
//
// Overview:
//
//	The EvmWriter contract is a critical bridge between Opera's consensus layer (Lachesis DAG)
//	and the EVM execution layer. It enables the driver contract to apply state changes from
//	events to the EVM state database without going through normal transaction execution.
//
// Security Model:
//   - Only the driver contract can call EvmWriter (strict caller validation)
//   - Protects transaction origin from balance/nonce manipulation during execution
//   - Enforces gas costs to prevent resource exhaustion attacks
//   - Validates input parameters to prevent invalid state transitions
//
// Use Cases:
//   - Applying validator rewards and penalties
//   - Updating validator code during upgrades
//   - Modifying contract storage for consensus-related state
//   - Adjusting account nonces for internal transactions
//
// Gas Costs:
//
//	Each operation charges appropriate gas costs based on the complexity and state changes:
//	- Balance operations: CallValueTransferGas
//	- Code operations: CreateGas + data-dependent costs
//	- Storage operations: SstoreSetGasEIP2200
//	- Nonce operations: CallValueTransferGas
package evmwriter

import (
	"bytes"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/rony4d/go-opera-asset/opera/contracts/driver"
)

var (
	// ContractAddress is the precompiled contract address for EvmWriter.
	// Address: 0xd100ec0000000000000000000000000000000000
	// This address is reserved in the EVM precompiled contract range.
	ContractAddress = common.HexToAddress("0xd100ec0000000000000000000000000000000000")

	// ContractABI is the JSON ABI definition for the EvmWriter contract.
	// This defines the function signatures that can be called:
	//   - setBalance(address acc, uint256 value): Set account balance to specific value
	//   - copyCode(address acc, address from): Copy code from one account to another
	//   - swapCode(address acc, address with): Swap code between two accounts
	//   - setStorage(address acc, bytes32 key, bytes32 value): Set storage slot value
	//   - incNonce(address acc, uint256 diff): Increment account nonce by specified amount
	ContractABI string = "[{\"constant\":false,\"inputs\":[{\"internalType\":\"address\",\"name\":\"acc\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"setBalance\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"internalType\":\"address\",\"name\":\"acc\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"}],\"name\":\"copyCode\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"internalType\":\"address\",\"name\":\"acc\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"with\",\"type\":\"address\"}],\"name\":\"swapCode\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"internalType\":\"address\",\"name\":\"acc\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"key\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"value\",\"type\":\"bytes32\"}],\"name\":\"setStorage\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"internalType\":\"address\",\"name\":\"acc\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"diff\",\"type\":\"uint256\"}],\"name\":\"incNonce\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]"
)

var (
	// Method IDs are the first 4 bytes of the keccak256 hash of the function signature.
	// These are computed at initialization time for efficient method dispatch.
	setBalanceMethodID []byte // setBalance(address,uint256)
	copyCodeMethodID   []byte // copyCode(address,address)
	swapCodeMethodID   []byte // swapCode(address,address)
	setStorageMethodID []byte // setStorage(address,bytes32,bytes32)
	incNonceMethodID   []byte // incNonce(address,uint256)
)

// init initializes the method IDs by parsing the contract ABI and extracting
// the method selector (first 4 bytes) for each function.
// This is called once at package initialization time.
func init() {
	// Parse the JSON ABI string into an ABI object
	abi, err := abi.JSON(strings.NewReader(ContractABI))
	if err != nil {
		panic(err)
	}

	// Map function names to their corresponding method ID variables
	for name, constID := range map[string]*[]byte{
		"setBalance": &setBalanceMethodID,
		"copyCode":   &copyCodeMethodID,
		"swapCode":   &swapCodeMethodID,
		"setStorage": &setStorageMethodID,
		"incNonce":   &incNonceMethodID,
	} {
		// Look up the method in the ABI
		method, exist := abi.Methods[name]
		if !exist {
			panic("unknown EvmWriter method")
		}

		// Copy the method ID (first 4 bytes of function selector)
		*constID = make([]byte, len(method.ID))
		copy(*constID, method.ID)
	}
}

// PreCompiledContract implements the vm.PrecompiledContract interface.
// This allows EvmWriter to be registered as a precompiled contract in the EVM.
type PreCompiledContract struct{}

// Run executes the precompiled contract logic.
// This is called by the EVM when a call is made to the ContractAddress.
//
// Security Checks:
//   1. Only the driver contract can call this (caller validation)
//   2. Input must contain at least 4 bytes (method selector)
//   3. Each method validates its specific input parameters
//   4. Gas costs are enforced for each operation
//
// Parameters:
//   - stateDB: The EVM state database interface for reading/writing state
//   - _: Block context (unused)
//   - txCtx: Transaction context containing origin address
//   - caller: Address of the contract calling this precompiled contract
//   - input: ABI-encoded function call data (method selector + parameters)
//   - suppliedGas: Gas available for this operation
//
// Returns:
//   - []byte: Return data (always nil for these operations)
//   - uint64: Remaining gas after execution
//   - error: Execution error (nil on success)

func (_ PreCompiledContract) Run(stateDB vm.StateDB, _ vm.BlockContext, txCtx vm.TxContext, caller common.Address, input []byte, suppliedGas uint64) ([]byte, uint64, error) {
	// SECURITY: Only the driver contract can call EvmWriter
	// This prevents arbitrary contracts from modifying EVM state
	if caller != driver.ContractAddress {
		return nil, 0, vm.ErrExecutionReverted
	}

	// Validate minimum input length (need at least 4 bytes for method selector)
	if len(input) < 4 {
		return nil, 0, vm.ErrExecutionReverted
	}

	// Dispatch to the appropriate method based on the first 4 bytes (method selector)
	if bytes.Equal(input[:4], setBalanceMethodID) {
		// Remove method selector from input
		input = input[4:]

		// setBalance(address acc, uint256 value)
		// Sets the balance of an account to a specific value.
		// This is used for applying validator rewards/penalties.

		// Charge base gas cost for value transfer operation
		if suppliedGas < params.CallValueTransferGas {
			return nil, 0, vm.ErrOutOfGas
		}
		suppliedGas -= params.CallValueTransferGas

		// Validate input length: 2 parameters * 32 bytes each = 64 bytes
		if len(input) != 64 {
			return nil, 0, vm.ErrExecutionReverted
		}

		// Decode address parameter (bytes 12-32, skipping 12 bytes of padding)
		acc := common.BytesToAddress(input[12:32])
		input = input[32:]

		// Decode uint256 value parameter (next 32 bytes)
		value := new(big.Int).SetBytes(input[:32])

		// SECURITY: Prevent modification of transaction origin's balance
		// This protects users from having their balance changed during their own transaction
		if acc == txCtx.Origin {
			return nil, 0, vm.ErrExecutionReverted
		}

		// Get current balance and adjust to target value
		balance := stateDB.GetBalance(acc)
		if balance.Cmp(value) >= 0 {
			// Current balance is higher than target, subtract the difference
			diff := new(big.Int).Sub(balance, value)
			stateDB.SubBalance(acc, diff)
		} else {
			// Current balance is lower than target, add the difference
			diff := new(big.Int).Sub(value, balance)
			stateDB.AddBalance(acc, diff)
		}

	} else if bytes.Equal(input[:4], copyCodeMethodID) {
		// Remove method selector from input
		input = input[4:]

		// copyCode(address acc, address from)
		// Copies contract code from one account to another.
		// Used for validator contract upgrades and code deployment.

		// Charge base gas cost for code creation operation
		if suppliedGas < params.CreateGas {
			return nil, 0, vm.ErrOutOfGas
		}
		suppliedGas -= params.CreateGas

		// Validate input length: 2 addresses * 32 bytes each = 64 bytes
		if len(input) != 64 {
			return nil, 0, vm.ErrExecutionReverted
		}

		// Decode destination address
		accTo := common.BytesToAddress(input[12:32])
		input = input[32:]

		// Decode source address
		accFrom := common.BytesToAddress(input[12:32])

		// Get code from source account (nil means empty code)
		code := stateDB.GetCode(accFrom)
		if code == nil {
			code = []byte{}
		}

		// Calculate gas cost based on code size
		// Each byte costs CreateDataGas + MemoryGas
		cost := uint64(len(code)) * (params.CreateDataGas + params.MemoryGas)
		if suppliedGas < cost {
			return nil, 0, vm.ErrOutOfGas
		}
		suppliedGas -= cost

		// Only set code if accounts are different (no-op if copying to self)
		if accTo != accFrom {
			stateDB.SetCode(accTo, code)
		}

	} else if bytes.Equal(input[:4], swapCodeMethodID) {
		// Remove method selector from input
		input = input[4:]

		// swapCode(address acc, address with)
		// Swaps contract code between two accounts atomically.
		// Used for validator contract migrations and upgrades.

		// Charge base gas cost for two code operations
		cost := 2 * params.CreateGas
		if suppliedGas < cost {
			return nil, 0, vm.ErrOutOfGas
		}
		suppliedGas -= cost

		// Validate input length: 2 addresses * 32 bytes each = 64 bytes
		if len(input) != 64 {
			return nil, 0, vm.ErrExecutionReverted
		}

		// Decode both addresses
		acc0 := common.BytesToAddress(input[12:32])
		input = input[32:]
		acc1 := common.BytesToAddress(input[12:32])

		// Get code from both accounts
		code0 := stateDB.GetCode(acc0)
		if code0 == nil {
			code0 = []byte{}
		}
		code1 := stateDB.GetCode(acc1)
		if code1 == nil {
			code1 = []byte{}
		}

		// Calculate gas cost for both code operations
		cost0 := uint64(len(code0)) * (params.CreateDataGas + params.MemoryGas)
		cost1 := uint64(len(code1)) * (params.CreateDataGas + params.MemoryGas)

		// Apply 50% discount because swapping code doesn't increase total trie size
		// (one account's code increases while the other decreases)
		cost = (cost0 + cost1) / 2
		if suppliedGas < cost {
			return nil, 0, vm.ErrOutOfGas
		}
		suppliedGas -= cost

		// Only swap if accounts are different
		if acc0 != acc1 {
			stateDB.SetCode(acc0, code1)
			stateDB.SetCode(acc1, code0)
		}

	} else if bytes.Equal(input[:4], setStorageMethodID) {
		// Remove method selector from input
		input = input[4:]

		// setStorage(address acc, bytes32 key, bytes32 value)
		// Sets a storage slot value for an account.
		// Used for updating consensus-related contract state.

		// Charge gas cost for storage write (EIP-2200: net gas metering)
		if suppliedGas < params.SstoreSetGasEIP2200 {
			return nil, 0, vm.ErrOutOfGas
		}
		suppliedGas -= params.SstoreSetGasEIP2200

		// Validate input length: address (32) + bytes32 key (32) + bytes32 value (32) = 96 bytes
		if len(input) != 96 {
			return nil, 0, vm.ErrExecutionReverted
		}

		// Decode account address
		acc := common.BytesToAddress(input[12:32])
		input = input[32:]

		// Decode storage key (bytes32)
		key := common.BytesToHash(input[:32])
		input = input[32:]

		// Decode storage value (bytes32)
		value := common.BytesToHash(input[:32])

		// Set the storage slot value
		stateDB.SetState(acc, key, value)

	} else if bytes.Equal(input[:4], incNonceMethodID) {
		// Remove method selector from input
		input = input[4:]

		// incNonce(address acc, uint256 diff)
		// Increments an account's nonce by a specified amount.
		// Used for internal transaction processing and nonce management.

		// Charge base gas cost for value transfer operation
		if suppliedGas < params.CallValueTransferGas {
			return nil, 0, vm.ErrOutOfGas
		}
		suppliedGas -= params.CallValueTransferGas

		// Validate input length: address (32) + uint256 (32) = 64 bytes
		if len(input) != 64 {
			return nil, 0, vm.ErrExecutionReverted
		}

		// Decode account address
		acc := common.BytesToAddress(input[12:32])
		input = input[32:]

		// Decode increment amount (uint256)
		value := new(big.Int).SetBytes(input[:32])

		// SECURITY: Prevent modification of transaction origin's nonce
		// This protects users from having their nonce changed during their own transaction
		if acc == txCtx.Origin {
			return nil, 0, vm.ErrExecutionReverted
		}

		// SECURITY: Prevent nonce overflow by limiting increment to 255
		// Nonces are uint64, but we limit to 255 to prevent edge cases
		if value.Cmp(common.Big256) >= 0 {
			return nil, 0, vm.ErrExecutionReverted
		}

		// Validate increment is positive
		if value.Sign() <= 0 {
			return nil, 0, vm.ErrExecutionReverted
		}

		// Increment the account's nonce
		stateDB.SetNonce(acc, stateDB.GetNonce(acc)+value.Uint64())

	} else {
		// Unknown method selector - revert
		return nil, 0, vm.ErrExecutionReverted
	}

	// Success: return nil data, remaining gas, and no error
	return nil, suppliedGas, nil
}
