package inter

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/rony4d/go-opera-asset/utils/cser"
)

/*
	This file implements Custom Serialization (CSER) for Ethereum Transactions.
	Even though Ethereum transactions are usually RLP-encoded(Recursive Length Prefix),
	this project wraps them in its own cser format when storing or transmitting them internally within the consensus layer.
	It supports 3 transaction types (EIP-2718):
	LegacyTx (Type 0x00): Standard pre-EIP-1559 transactions.
	AccessListTx (Type 0x01): EIP-2930 transactions with access lists.
	DynamicFeeTx (Type 0x02): EIP-1559 transactions (London hardfork) with GasTipCap and GasFeeCap.
*/

// ErrUnknownTxType is returned when deserializing a transaction with an unsupported type byte.
var ErrUnknownTxType = errors.New("unknown tx type: supported types are Legacy, AccessList, DynamicFee")

// encodeSig packs the ECDSA signature values 'R' and 'S' into a fixed 64-byte array.
// Format: [32 bytes R] [32 bytes S]
// This is a standard serialization format for signatures (often called 'RS' format).
// Note: 'V' (recovery ID) is serialized separately.
func encodeSig(r, s *big.Int) (sig [64]byte) {
	// PaddedBytes ensures we always have 32 bytes, even if the number is small.
	// (e.g., if R is 0x05, it becomes 0x00...05).
	copy(sig[0:], cser.PaddedBytes(r.Bytes(), 32)[:32])
	copy(sig[32:], cser.PaddedBytes(s.Bytes(), 32)[:32])
	return sig
}

// decodeSig unpacks the 64-byte signature array back into BigInts 'R' and 'S'.
func decodeSig(sig [64]byte) (r, s *big.Int) {
	r = new(big.Int).SetBytes(sig[:32])
	s = new(big.Int).SetBytes(sig[32:64])
	return
}

// TransactionMarshalCSER serializes an Ethereum Transaction into the custom CSER format.
// It handles polymorphism (Legacy vs EIP-2930 vs EIP-1559) using a type prefix.
func TransactionMarshalCSER(w *cser.Writer, tx *types.Transaction) error {
	// 1. Validation: Check if type is supported
	if tx.Type() != types.LegacyTxType && tx.Type() != types.AccessListTxType && tx.Type() != types.DynamicFeeTxType {
		return ErrUnknownTxType
	}

	// 2. Header / Type Prefix
	if tx.Type() != types.LegacyTxType {
		// New Transaction Types (EIP-2718)
		// We use a "marker" in the bit stream to signal this isn't a legacy tx.
		// The marker is 6 bits of zeros.
		// Why 6 bits? It likely corresponds to a specific layout assumption in the reader (see Unmarshal).
		w.BitsW.Write(6, 0)
		w.U8(tx.Type())
	} else if tx.Gas() <= 0xff {
		// Legacy Transaction specific check.
		// It seems there's a constraint that legacy gas limit must be > 255?
		// This might be to avoid ambiguity with the "0" bits marker above if gas was encoded compactly?
		// (Legacy format starts with Nonce/Gas which are U64s).
		return errors.New("cannot serialize legacy tx with gasLimit <= 256")
	}

	// 3. Common Fields (Nonce, Gas)
	w.U64(tx.Nonce())
	w.U64(tx.Gas())

	// 4. Fee Fields (Type Dependent)
	if tx.Type() == types.DynamicFeeTxType {
		w.BigInt(tx.GasTipCap()) // EIP-1559 Priority Fee
		w.BigInt(tx.GasFeeCap()) // EIP-1559 Max Fee
	} else {
		w.BigInt(tx.GasPrice()) // Legacy Gas Price
	}

	// 5. Payment Fields
	w.BigInt(tx.Value())

	// 6. Recipient (To)
	// Contract creation has To == nil. We use a boolean flag to indicate presence.
	w.Bool(tx.To() != nil)
	if tx.To() != nil {
		w.FixedBytes(tx.To().Bytes())
	}

	// 7. Input Data (Calldata)
	w.SliceBytes(tx.Data())

	// 8. Signature
	v, r, s := tx.RawSignatureValues()
	w.BigInt(v) // Recovery ID / ChainID replay protection
	sig := encodeSig(r, s)
	w.FixedBytes(sig[:])

	// 9. Extended Fields (AccessList / ChainID)
	if tx.Type() == types.AccessListTxType || tx.Type() == types.DynamicFeeTxType {
		w.BigInt(tx.ChainId()) // EIP-1559/2930 include ChainID explicitly in the payload

		// Serialize Access List: [Address, [StorageKey1, StorageKey2...]]
		w.U32(uint32(len(tx.AccessList())))
		for _, tuple := range tx.AccessList() {
			w.FixedBytes(tuple.Address.Bytes())
			w.U32(uint32(len(tuple.StorageKeys)))
			for _, h := range tuple.StorageKeys {
				w.FixedBytes(h.Bytes())
			}
		}
	}

	return nil
}

// TransactionUnmarshalCSER deserializes a CSER stream into an Ethereum Transaction.
func TransactionUnmarshalCSER(r *cser.Reader) (*types.Transaction, error) {
	// 1. Determine Type
	// Check the next 6 bits.
	// If they are 0, it's a marker for a Typed Tx.
	// If they are NOT 0, it's the start of a Legacy Tx (likely part of the Nonce or Gas?).
	txType := uint8(types.LegacyTxType)
	if r.BitsR.View(6) == 0 {
		r.BitsR.Read(6) // Consume the marker
		txType = r.U8() // Read the actual type byte
	}

	// 2. Read Common Fields
	nonce := r.U64()
	gasLimit := r.U64()

	// 3. Read Fees
	var gasPrice *big.Int
	var gasTipCap *big.Int
	var gasFeeCap *big.Int
	if txType == types.DynamicFeeTxType {
		gasTipCap = r.BigInt()
		gasFeeCap = r.BigInt()
	} else {
		gasPrice = r.BigInt()
	}

	// 4. Read Value & Recipient
	amount := r.BigInt()
	toExists := r.Bool()
	var to *common.Address
	if toExists {
		var _to common.Address
		r.FixedBytes(_to[:])
		to = &_to
	}

	// 5. Read Data & Sig
	data := r.SliceBytes(ProtocolMaxMsgSize)
	v := r.BigInt()
	var sig [64]byte
	r.FixedBytes(sig[:])
	_r, s := decodeSig(sig)

	// 6. Construct Legacy Tx
	if txType == types.LegacyTxType {
		return types.NewTx(&types.LegacyTx{
			Nonce:    nonce,
			GasPrice: gasPrice,
			Gas:      gasLimit,
			To:       to,
			Value:    amount,
			Data:     data,
			V:        v,
			R:        _r,
			S:        s,
		}), nil
	} else if txType == types.AccessListTxType || txType == types.DynamicFeeTxType {
		// 7. Read Extended Fields for Typed Txs
		chainID := r.BigInt()

		// Read Access List
		accessListLen := r.U32()
		if accessListLen > ProtocolMaxMsgSize/24 {
			return nil, cser.ErrTooLargeAlloc // prevent huge allocs
		}
		accessList := make(types.AccessList, accessListLen)
		for i := range accessList {
			r.FixedBytes(accessList[i].Address[:])
			keysLen := r.U32()
			if keysLen > ProtocolMaxMsgSize/32 {
				return nil, cser.ErrTooLargeAlloc
			}
			accessList[i].StorageKeys = make([]common.Hash, keysLen)
			for j := range accessList[i].StorageKeys {
				r.FixedBytes(accessList[i].StorageKeys[j][:])
			}
		}

		// 8. Construct Typed Tx
		if txType == types.AccessListTxType {
			return types.NewTx(&types.AccessListTx{
				ChainID:    chainID,
				Nonce:      nonce,
				GasPrice:   gasPrice,
				Gas:        gasLimit,
				To:         to,
				Value:      amount,
				Data:       data,
				AccessList: accessList,
				V:          v,
				R:          _r,
				S:          s,
			}), nil
		} else {
			return types.NewTx(&types.DynamicFeeTx{
				ChainID:    chainID,
				Nonce:      nonce,
				GasTipCap:  gasTipCap,
				GasFeeCap:  gasFeeCap,
				Gas:        gasLimit,
				To:         to,
				Value:      amount,
				Data:       data,
				AccessList: accessList,
				V:          v,
				R:          _r,
				S:          s,
			}), nil
		}
	}
	return nil, ErrUnknownTxType
}
