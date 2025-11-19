# Decoding Contract Bytecode and Generating Pre-Deployment Bytecode

## Decoding the NodeDriver Contract Bytecode

The bytecode in `driver_predeploy.go` contains the compiled Solidity contract. Here's how to decode it:

### Function Selectors Identified

From the bytecode analysis, the following **16 function selectors** are present:

1. `0x07690b2a`
2. `0x0aeeca00`
3. `0x18f628d4`
4. `0x1e702f83`
5. `0x242a6e3f`
6. `0x267ab446`
7. `0x39e503ab`
8. `0x485cc955`
9. `0x4feb92f3`
10. `0xa4066fbe`
11. `0xb9cc6b1c`
12. `0xd6a0c7af`
13. `0xda7fc24f`
14. `0xe08d7e66`
15. `0xe30443bc`
16. `0xebdf104c`

**Note**: These are the 4-byte function selectors embedded in the bytecode. To find the actual function signatures, use the methods below.

### How to Decode Function Signatures

Function selectors are the first 4 bytes of `keccak256(function_signature)`.

**Method 1: Using Online Tools**

1. **Ethervm.io Decompiler**: https://ethervm.io/decompile
   - Paste the bytecode
   - Get decompiled Solidity-like code

2. **EVMDecompiler**: https://evmdecompiler.com/
   - Supports multiple chains
   - Provides readable control flow

3. **4byte.directory**: https://www.4byte.directory/
   - Search function selectors to find signatures
   - Example: Search `0x07690b2a` to find matching function signatures

**Method 2: Using Command Line Tools**

```bash
# Using cast from Foundry (recommended)
# Install foundry first: curl -L https://foundry.paradigm.xyz | bash && foundryup

# Calculate selector from function signature
cast sig "functionName(uint256,address)"
# Output: 0x12345678

# Reverse lookup selector to find possible signatures
cast 4byte 0x07690b2a
# This searches 4byte.directory for matching signatures

# Or use web3.py to calculate
python3 -c "from web3 import Web3; print(Web3.keccak(text='functionName(uint256)')[:4].hex())"
```

**Method 3: Using Solidity Compiler**

If you have access to the source code (opera-sfc repository):
```bash
# Clone the source
git clone https://github.com/Fantom-foundation/opera-sfc.git
cd opera-sfc
git checkout c1d33c81f74abf82c0e22807f16e609578e10ad8

# Compile and get function selectors
solc --hashes NodeDriver.sol
```

### Expected Functions (Based on Comments)

Based on the contract purpose (NodeDriver for validator operations), likely functions include:

- Validator registration/management
- Epoch transitions
- State synchronization
- Reward/penalty handling
- Delegation logic

## Generating Your Own Pre-Deployment Bytecode

### Step 1: Write Your Solidity Contract

Create a contract file, e.g., `MyPredeploy.sol`:

```solidity
pragma solidity ^0.5.17;

contract MyPredeploy {
    address public backend;
    
    constructor(address _backend) public {
        backend = _backend;
    }
    
    function setValue(uint256 value) external {
        require(msg.sender == backend, "caller is not the backend");
        // Your logic here
    }
    
    function getValue() external view returns (uint256) {
        // Your logic here
        return 0;
    }
}
```

### Step 2: Compile the Contract

**Using Solidity Compiler (solc):**

```bash
# Install solc
npm install -g solc

# Compile with runtime bytecode output
solc --bin-runtime MyPredeploy.sol -o output/

# This generates:
# - output/MyPredeploy.bin-runtime (runtime bytecode for predeployment)
# - output/MyPredeploy.bin (creation bytecode, includes constructor)
```

**Using Hardhat:**

```bash
# Install hardhat
npm install --save-dev hardhat

# Create hardhat.config.js
npx hardhat init

# Compile
npx hardhat compile

# Bytecode is in artifacts/contracts/MyPredeploy.sol/MyPredeploy.json
# Look for "deployedBytecode" field
```

**Using Foundry:**

```bash
# Install foundry
curl -L https://foundry.paradigm.xyz | bash
foundryup

# Compile
forge build

# Get bytecode
forge inspect MyPredeploy bytecode
```

### Step 3: Extract Runtime Bytecode

For pre-deployment, you need **runtime bytecode** (not creation bytecode):

- **Runtime bytecode**: Code that's stored on-chain (what `GetContractBin()` returns)
- **Creation bytecode**: Includes constructor logic, used for deployment

**Important**: The bytecode in `driver_predeploy.go` is **runtime bytecode** because:
1. It's predeployed at genesis (no constructor execution needed)
2. The comment says "bin-runtime flag"
3. It excludes constructor initialization code

### Step 4: Convert to Go Format

Once you have the runtime bytecode:

```go
package mycontract

import (
    "github.com/ethereum/go-ethereum/common/hexutil"
)

func GetContractBin() []byte {
    // Paste your runtime bytecode here (without 0x prefix in hexutil.MustDecode)
    return hexutil.MustDecode("0x608060405234801561001057600080fd5b...")
}

var ContractAddress = common.HexToAddress("0xYourPredeployAddress")
```

### Step 5: Choose a Predeploy Address

Opera uses specific address ranges for system contracts:
- Format: `0xd100a01e00000000000000000000000000000000` (NodeDriver)
- Choose a unique address that won't conflict

### Complete Example Workflow

```bash
# 1. Write contract
cat > MyContract.sol << 'EOF'
pragma solidity ^0.5.17;
contract MyContract {
    function hello() public pure returns (string memory) {
        return "Hello, Opera!";
    }
}
EOF

# 2. Compile
solc --bin-runtime MyContract.sol -o output/

# 3. Get runtime bytecode
cat output/MyContract.bin-runtime
# Output: 608060405234801561001057600080fd5b50600436106100165760003560e01c80638381f58a1461001b575b600080fd5b610023610039565b60405161002f9190610042565b60405180910390f35b60005481565b600081905091905056fea265627a7a72315820...

# 4. Use in Go code
```

### Verification

To verify your bytecode matches the source:

```bash
# Get function selectors from bytecode
cast sig "hello()"  # Should match selector in bytecode

# Deploy and test locally
npx hardhat node
npx hardhat run scripts/deploy.js --network localhost
```

### Best Practices

1. **Use exact compiler version**: Match the Solidity version used (0.5.17 for NodeDriver)
2. **Optimization settings**: Use same optimization runs (10000 for NodeDriver)
3. **Runtime vs Creation**: Always use `--bin-runtime` for predeployment
4. **Test thoroughly**: Deploy to testnet first
5. **Document source**: Keep track of source code location and commit hash
6. **Verify addresses**: Ensure predeploy address doesn't conflict

### Tools Reference

- **Solc**: https://github.com/ethereum/solidity
- **Hardhat**: https://hardhat.org/
- **Foundry**: https://book.getfoundry.sh/
- **Etherscan Bytecode Verifier**: For mainnet verification
- **Tenderly**: For bytecode analysis and debugging

