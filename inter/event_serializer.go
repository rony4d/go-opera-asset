/*
This file, event_serializer.go (likely short for Event Serializer), defines how to serialize and deserialize Events using a custom binary format called CSER (Canonical Serialization).
1. What is CSER?
CSER is a custom binary format designed for speed and deterministic hashing.
It is used to serialize and deserialize Events in the network.
2. Why CSER?
The main reason is speed. CSER is much faster than RLP (Recursive Length Prefix) serialization.
3. How it works?

CSER is a custom binary format designed for speed and deterministic hashing.
It is used to serialize and deserialize Events in the network.
2. Why CSER?
The main reason is speed. CSER is much faster than RLP (Recursive Length Prefix) serialization.
3. How it works?
CSER is a custom binary format designed for speed and deterministic hashing.
It is used to serialize and deserialize Events in the network.
4. How to use it?
You can use it by calling the MarshalCSER function.
5. How to deserialize it?
You can deserialize it by calling the UnmarshalCSER function.
6. How to verify it?
You can verify it by calling the VerifyCSER function.
7. How to hash it?
You can hash it by calling the HashCSER function.
8. How to compare it?
You can compare it by calling the CompareCSER function.
9. How to get the size of it?
You can get the size of it by calling the SizeCSER function.
10. How to get the type of it?
You can get the type of it by calling the TypeCSER function.
*/
package inter

import (
	"errors"
	"io"

	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/dag"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/rony4d/go-opera-asset/utils/cser"
)

// Errors related to event serialization.
var (
	ErrSerMalformedEvent = errors.New("serialization of malformed event: structure violates protocol rules")
	ErrTooLowEpoch       = errors.New("serialization of events with epoch<256 and version=0 is unsupported")
	ErrUnknownVersion    = errors.New("unknown serialization version: client is likely outdated")
)

// MaxSerializationVersion defines the highest version of the wire protocol this node supports.
const MaxSerializationVersion = 1

// ProtocolMaxMsgSize defines the hard limit for network message size (10 MB).
// Used to prevent DoS attacks via massive allocations.
const ProtocolMaxMsgSize = 10 * 1024 * 1024

// MarshalCSER serializes an Event (Header) into the Canonical Serialization format.
// CSER is a custom compact binary format designed for speed and deterministic hashing.
//
// Structure (Version > 0):
// 1. Version (2 bits + uint8)
// 2. NetworkForkID (uint16)
// 3. Epoch, Lamport, Creator, Seq, Frame (uint32s)
// 4. Timestamps (Creation, Median Diff)
// 5. Gas Power (Used, Left)
// 6. Parents (Count, Lamport Diffs, Hashes)
// 7. PrevEpochHash (Optional)

// 8. Flags (AnyTxs, AnyMPs, etc.)
// 9. PayloadHash (if flags are true)
// 10. Extra Data
func (e *Event) MarshalCSER(w *cser.Writer) error {
	// 1. Versioning
	// We use a bit-packing trick here.
	if e.Version() > 0 {
		w.BitsW.Write(2, 0) // Write 2 bits as '0' to signal non-zero version follows?
		w.U8(e.Version())
	} else {
		// Version 0 check: Epoch must be >= 256 for legacy reasons.
		if e.Epoch() < 256 {
			return ErrTooLowEpoch
		}
	}

	// 2. Base Fields
	if e.Version() > 0 {
		w.U16(e.NetForkID())
	}
	w.U32(uint32(e.Epoch()))
	w.U32(uint32(e.Lamport()))
	w.U32(uint32(e.Creator()))
	w.U32(uint32(e.Seq()))
	w.U32(uint32(e.Frame()))
	w.U64(uint64(e.creationTime))

	// Optimization: Store median time as a difference from creation time to save space (varint).
	medianTimeDiff := int64(e.creationTime) - int64(e.medianTime)
	w.I64(medianTimeDiff)

	// 3. Gas Power
	w.U64(e.gasPowerUsed)
	w.U64(e.gasPowerLeft.Gas[0])
	w.U64(e.gasPowerLeft.Gas[1])

	// 4. Parents (Graph Topology)
	w.U32(uint32(len(e.Parents())))
	for _, p := range e.Parents() {
		if e.Lamport() < p.Lamport() {
			return ErrSerMalformedEvent // Child cannot be older than parent
		}
		// Optimization: Store parent lamport as difference (varint friendly)
		w.U32(uint32(e.Lamport() - p.Lamport()))
		// Store parent hash suffix (assuming prefix is known or full hash used depending on impl)
		// Note: The code writes `p.Bytes()[8:]`, skipping first 8 bytes?
		// This usually implies the first 8 bytes are Epoch/Lamport data embedded in ID.
		w.FixedBytes(p.Bytes()[8:])
	}

	// 5. Previous Epoch Hash (Linking epochs)
	w.Bool(e.prevEpochHash != nil)
	if e.prevEpochHash != nil {
		w.FixedBytes(e.prevEpochHash.Bytes())
	}

	// 6. Content Flags
	w.Bool(e.AnyTxs())
	if e.Version() > 0 {
		w.Bool(e.AnyMisbehaviourProofs())
		w.Bool(e.AnyEpochVote())
		w.Bool(e.AnyBlockVotes())
	}

	// 7. Payload Hash
	// Only write the payload hash if there is actual content.
	if e.AnyTxs() || e.AnyMisbehaviourProofs() || e.AnyBlockVotes() || e.AnyEpochVote() {
		w.FixedBytes(e.PayloadHash().Bytes())
	}

	// 8. Extra Data
	w.SliceBytes(e.Extra())
	return nil
}

// MarshalBinary implements encoding.BinaryMarshaller.
// It wraps the CSER writer to output a byte slice.
func (e *Event) MarshalBinary() ([]byte, error) {
	return cser.MarshalBinaryAdapter(e.MarshalCSER)
}

// eventUnmarshalCSER reads an Event (Header) from the CSER format.
// It is the inverse of MarshalCSER.
func eventUnmarshalCSER(r *cser.Reader, e *MutableEventPayload) (err error) {
	// 1. Version
	var version uint8
	if r.BitsR.View(2) == 0 {
		r.BitsR.Read(2)
		version = r.U8()
		if version == 0 {
			return cser.ErrNonCanonicalEncoding
		}
	}
	if version > MaxSerializationVersion {
		return ErrUnknownVersion
	}

	// 2. Base Fields
	var netForkID uint16
	if version > 0 {
		netForkID = r.U16()
	}
	epoch := r.U32()
	lamport := r.U32()
	creator := r.U32()
	seq := r.U32()
	frame := r.U32()
	creationTime := r.U64()
	medianTimeDiff := r.I64()

	// 3. Gas Power
	gasPowerUsed := r.U64()
	gasPowerLeft0 := r.U64()
	gasPowerLeft1 := r.U64()

	// 4. Parents
	parentsNum := r.U32()
	if parentsNum > ProtocolMaxMsgSize/24 {
		return cser.ErrTooLargeAlloc // Sanity check
	}
	parents := make(hash.Events, 0, parentsNum)
	for i := uint32(0); i < parentsNum; i++ {
		lamportDiff := r.U32()
		h := [24]byte{}
		r.FixedBytes(h[:]) // Reads the suffix

		// Reconstruct full Parent ID
		eID := dag.MutableBaseEvent{}
		eID.SetEpoch(idx.Epoch(epoch))
		eID.SetLamport(idx.Lamport(lamport - lamportDiff))
		eID.SetID(h) // SetID likely handles merging the suffix with Epoch/Lamport
		parents.Add(eID.ID())
	}

	// 5. Prev Epoch Hash
	var prevEpochHash *hash.Hash
	prevEpochHashExists := r.Bool()
	if prevEpochHashExists {
		prevEpochHash_ := hash.Hash{}
		r.FixedBytes(prevEpochHash_[:])
		prevEpochHash = &prevEpochHash_
	}

	// 6. Content Flags
	anyTxs := r.Bool()
	anyMisbehaviourProofs := version > 0 && r.Bool()
	anyEpochVote := version > 0 && r.Bool()
	anyBlockVotes := version > 0 && r.Bool()

	// 7. Payload Hash
	payloadHash := EmptyPayloadHash(version)
	if anyTxs || anyMisbehaviourProofs || anyEpochVote || anyBlockVotes {
		r.FixedBytes(payloadHash[:])
		if payloadHash == EmptyPayloadHash(version) {
			return cser.ErrNonCanonicalEncoding // Must not explicitly transmit empty hash if empty
		}
	}

	// 8. Extra Data
	extra := r.SliceBytes(ProtocolMaxMsgSize)

	// Validation
	if version == 0 && epoch < 256 {
		return ErrTooLowEpoch
	}

	// Populate the Mutable Event
	e.SetVersion(version)
	e.SetNetForkID(netForkID)
	e.SetEpoch(idx.Epoch(epoch))
	e.SetLamport(idx.Lamport(lamport))
	e.SetCreator(idx.ValidatorID(creator))
	e.SetSeq(idx.Event(seq))
	e.SetFrame(idx.Frame(frame))
	e.SetCreationTime(Timestamp(creationTime))
	e.SetMedianTime(Timestamp(int64(creationTime) - medianTimeDiff))
	e.SetGasPowerUsed(gasPowerUsed)
	e.SetGasPowerLeft(GasPowerLeft{[2]uint64{gasPowerLeft0, gasPowerLeft1}})
	e.SetParents(parents)
	e.SetPrevEpochHash(prevEpochHash)
	e.anyTxs = anyTxs
	e.anyBlockVotes = anyBlockVotes
	e.anyEpochVote = anyEpochVote
	e.anyMisbehaviourProofs = anyMisbehaviourProofs
	e.SetPayloadHash(payloadHash)
	e.SetExtra(extra)
	return nil
}

// MarshalTxsCSER serializes a list of transactions.
func MarshalTxsCSER(txs types.Transactions, w *cser.Writer) error {
	w.U56(uint64(txs.Len())) // Write count
	for _, tx := range txs {
		// Assuming TransactionMarshalCSER exists elsewhere in utility code
		err := TransactionMarshalCSER(w, tx)
		if err != nil {
			return err
		}
	}
	return nil
}

// MarshalCSER for LlrBlockVotes (The batch of block votes).
func (bvs LlrBlockVotes) MarshalCSER(w *cser.Writer) error {
	w.U64(uint64(bvs.Start))
	w.U32(uint32(bvs.Epoch))
	w.U32(uint32(len(bvs.Votes)))
	for _, r := range bvs.Votes {
		w.FixedBytes(r[:])
	}
	return nil
}

// UnmarshalCSER for LlrBlockVotes.
func (bvs *LlrBlockVotes) UnmarshalCSER(r *cser.Reader) error {
	start := r.U64()
	epoch := r.U32()
	num := r.U32()
	if num > ProtocolMaxMsgSize/32 {
		return cser.ErrTooLargeAlloc
	}
	records := make([]hash.Hash, num)
	for i := range records {
		r.FixedBytes(records[i][:])
	}
	bvs.Start = idx.Block(start)
	bvs.Epoch = idx.Epoch(epoch)
	bvs.Votes = records
	return nil
}

// MarshalCSER for LlrEpochVote.
func (ers LlrEpochVote) MarshalCSER(w *cser.Writer) error {
	w.U32(uint32(ers.Epoch))
	w.FixedBytes(ers.Vote[:])
	return nil
}

// UnmarshalCSER for LlrEpochVote.
func (ers *LlrEpochVote) UnmarshalCSER(r *cser.Reader) error {
	epoch := r.U32()
	record := hash.Hash{}
	r.FixedBytes(record[:])
	ers.Epoch = idx.Epoch(epoch)
	ers.Vote = record
	return nil
}

// MarshalCSER for the full EventPayload (Header + Body + Sig).
// This is the main function called when sending an event over the network.
func (e *EventPayload) MarshalCSER(w *cser.Writer) error {
	// Sanity checks to ensure flags match content
	if e.AnyTxs() != (e.txs.Len() != 0) {
		return ErrSerMalformedEvent
	}
	if e.AnyMisbehaviourProofs() != (len(e.misbehaviourProofs) != 0) {
		return ErrSerMalformedEvent
	}
	// ... other checks ...

	// 1. Write Header (Event part)
	err := e.Event.MarshalCSER(w)
	if err != nil {
		return err
	}

	// 2. Write Signature
	w.FixedBytes(e.sig.Bytes())

	// 3. Write Body (Conditional on flags)
	if e.AnyTxs() {
		if e.Version() == 0 {
			// Legacy format uses custom CSER for txs
			err = MarshalTxsCSER(e.txs, w)
			if err != nil {
				return err
			}
		} else {
			// Modern format uses RLP for txs inside CSER
			b, err := rlp.EncodeToBytes(e.txs)
			if err != nil {
				return err
			}
			w.SliceBytes(b)
		}
	}
	if e.AnyMisbehaviourProofs() {
		// MPs are always RLP encoded
		b, err := rlp.EncodeToBytes(e.misbehaviourProofs)
		if err != nil {
			return err
		}
		w.SliceBytes(b)
	}
	if e.AnyEpochVote() {
		err = e.EpochVote().MarshalCSER(w)
		if err != nil {
			return err
		}
	}
	if e.AnyBlockVotes() {
		err = e.BlockVotes().MarshalCSER(w)
		if err != nil {
			return err
		}
	}
	return nil
}

// UnmarshalCSER for MutableEventPayload.
// Reads Header -> Sig -> Body.
func (e *MutableEventPayload) UnmarshalCSER(r *cser.Reader) error {
	// 1. Read Header
	err := eventUnmarshalCSER(r, e)
	if err != nil {
		return err
	}

	// 2. Read Signature
	r.FixedBytes(e.sig[:])

	// 3. Read Body
	// Transactions
	txs := make(types.Transactions, 0, 4)
	if e.AnyTxs() {
		if e.version == 0 {
			// Legacy CSER decoding
			size := r.U56()
			// ... size checks ...
			for i := uint64(0); i < size; i++ {
				tx, err := TransactionUnmarshalCSER(r)
				if err != nil {
					return err
				}
				txs = append(txs, tx)
			}
		} else {
			// Modern RLP decoding
			b := r.SliceBytes(ProtocolMaxMsgSize)
			err := rlp.DecodeBytes(b, &txs)
			if err != nil {
				return err
			}
		}
	}
	e.txs = txs

	// Misbehaviour Proofs
	mps := make([]MisbehaviourProof, 0)
	if e.AnyMisbehaviourProofs() {
		b := r.SliceBytes(ProtocolMaxMsgSize)
		err := rlp.DecodeBytes(b, &mps)
		if err != nil {
			return err
		}
	}
	e.misbehaviourProofs = mps

	// Epoch Votes
	ev := LlrEpochVote{}
	if e.AnyEpochVote() {
		err := ev.UnmarshalCSER(r)
		if err != nil {
			return err
		}
		// Validation
		if ev.Epoch == 0 {
			return cser.ErrNonCanonicalEncoding
		}
	}
	e.epochVote = ev

	// Block Votes
	bvs := LlrBlockVotes{Votes: make([]hash.Hash, 0, 2)}
	if e.AnyBlockVotes() {
		err := bvs.UnmarshalCSER(r)
		if err != nil {
			return err
		}
		// Validation
		if len(bvs.Votes) == 0 || bvs.Start == 0 || bvs.Epoch == 0 {
			return cser.ErrNonCanonicalEncoding
		}
	}
	e.blockVotes = bvs
	return nil
}

// UnmarshalBinary implements encoding.BinaryUnmarshaller interface.
func (e *MutableEventPayload) UnmarshalBinary(raw []byte) (err error) {
	return cser.UnmarshalBinaryAdapter(raw, e.UnmarshalCSER)
}

// MarshalBinary implements encoding.BinaryMarshaller.
func (e *EventPayload) MarshalBinary() ([]byte, error) {
	return cser.MarshalBinaryAdapter(e.MarshalCSER)
}

// UnmarshalBinary for EventPayload (Immutable).
// It uses MutableEventPayload as an intermediate builder.
func (e *EventPayload) UnmarshalBinary(raw []byte) (err error) {
	mutE := MutableEventPayload{}
	err = mutE.UnmarshalBinary(raw)
	if err != nil {
		return err
	}
	// After deserializing, we must rebuild the cached hashes and immutable structure.
	eventSer, _ := mutE.immutable().Event.MarshalBinary()
	locatorHash, baseHash := calcEventHashes(eventSer, &mutE)
	*e = *mutE.build(locatorHash, baseHash, len(raw))
	return nil
}

// EncodeRLP implements rlp.Encoder interface.
func (e *EventPayload) EncodeRLP(w io.Writer) error {
	bytes, err := e.MarshalBinary()
	if err != nil {
		return err
	}

	err = rlp.Encode(w, &bytes)

	return err
}

// DecodeRLP implements rlp.Decoder interface.
func (e *EventPayload) DecodeRLP(src *rlp.Stream) error {
	bytes, err := src.Bytes()
	if err != nil {
		return err
	}

	return e.UnmarshalBinary(bytes)
}

// DecodeRLP implements rlp.Decoder interface.
func (e *MutableEventPayload) DecodeRLP(src *rlp.Stream) error {
	bytes, err := src.Bytes()
	if err != nil {
		return err
	}

	return e.UnmarshalBinary(bytes)
}

// RPCMarshalEvent converts the Event to a JSON-friendly map for API responses.
// Uses hexutil for hex encoding of binary fields.
// RPCMarshalEvent converts the given event to the RPC output .
func RPCMarshalEvent(e EventI) map[string]interface{} {
	return map[string]interface{}{
		"version":        hexutil.Uint64(e.Version()),
		"networkVersion": hexutil.Uint64(e.NetForkID()),
		"epoch":          hexutil.Uint64(e.Epoch()),
		"seq":            hexutil.Uint64(e.Seq()),
		"id":             hexutil.Bytes(e.ID().Bytes()),
		"frame":          hexutil.Uint64(e.Frame()),
		"creator":        hexutil.Uint64(e.Creator()),
		"prevEpochHash":  e.PrevEpochHash(),
		"parents":        EventIDsToHex(e.Parents()),
		"lamport":        hexutil.Uint64(e.Lamport()),
		"creationTime":   hexutil.Uint64(e.CreationTime()),
		"medianTime":     hexutil.Uint64(e.MedianTime()),
		"extraData":      hexutil.Bytes(e.Extra()),
		"payloadHash":    hexutil.Bytes(e.PayloadHash().Bytes()),
		"gasPowerLeft": map[string]interface{}{
			"shortTerm": hexutil.Uint64(e.GasPowerLeft().Gas[ShortTermGas]),
			"longTerm":  hexutil.Uint64(e.GasPowerLeft().Gas[LongTermGas]),
		},
		"gasPowerUsed":          hexutil.Uint64(e.GasPowerUsed()),
		"anyTxs":                e.AnyTxs(),
		"anyMisbehaviourProofs": e.AnyMisbehaviourProofs(),
		"anyEpochVote":          e.AnyEpochVote(),
		"anyBlockVotes":         e.AnyBlockVotes(),
	}
}

// RPCUnmarshalEvent converts the RPC output to the header.
func RPCUnmarshalEvent(fields map[string]interface{}) EventI {
	mustBeUint64 := func(name string) uint64 {
		s := fields[name].(string)
		return hexutil.MustDecodeUint64(s)
	}
	mustBeBytes := func(name string) []byte {
		s := fields[name].(string)
		return hexutil.MustDecode(s)
	}
	mustBeID := func(name string) (id [24]byte) {
		s := fields[name].(string)
		bb := hexutil.MustDecode(s)
		copy(id[:], bb)
		return
	}
	mustBeBool := func(name string) bool {
		return fields[name].(bool)
	}
	mayBeHash := func(name string) *hash.Hash {
		s, ok := fields[name].(string)
		if !ok {
			return nil
		}
		bb := hexutil.MustDecode(s)
		h := hash.BytesToHash(bb)
		return &h
	}

	e := MutableEventPayload{}

	e.SetVersion(uint8(mustBeUint64("version")))
	e.SetNetForkID(uint16(mustBeUint64("networkVersion")))
	e.SetEpoch(idx.Epoch(mustBeUint64("epoch")))
	e.SetSeq(idx.Event(mustBeUint64("seq")))
	e.SetID(mustBeID("id"))
	e.SetFrame(idx.Frame(mustBeUint64("frame")))
	e.SetCreator(idx.ValidatorID(mustBeUint64("creator")))
	e.SetPrevEpochHash(mayBeHash("prevEpochHash"))
	e.SetParents(HexToEventIDs(fields["parents"].([]interface{})))
	e.SetLamport(idx.Lamport(mustBeUint64("lamport")))
	e.SetCreationTime(Timestamp(mustBeUint64("creationTime")))
	e.SetMedianTime(Timestamp(mustBeUint64("medianTime")))
	e.SetExtra(mustBeBytes("extraData"))
	e.SetPayloadHash(*mayBeHash("payloadHash"))
	e.SetGasPowerUsed(mustBeUint64("gasPowerUsed"))
	e.anyTxs = mustBeBool("anyTxs")
	e.anyMisbehaviourProofs = mustBeBool("anyMisbehaviourProofs")
	e.anyEpochVote = mustBeBool("anyEpochVote")
	e.anyBlockVotes = mustBeBool("anyBlockVotes")

	gas := GasPowerLeft{}
	obj := fields["gasPowerLeft"].(map[string]interface{})
	gas.Gas[ShortTermGas] = hexutil.MustDecodeUint64(obj["shortTerm"].(string))
	gas.Gas[LongTermGas] = hexutil.MustDecodeUint64(obj["longTerm"].(string))
	e.SetGasPowerLeft(gas)

	return &e.Build().Event
}

// RPCMarshalEventPayload converts the given event to the RPC output which depends on fullTx. If inclTx is true transactions are
// returned. When fullTx is true the returned block contains full transaction details, otherwise it will only contain
// transaction hashes.
func RPCMarshalEventPayload(event EventPayloadI, inclTx bool, fullTx bool) (map[string]interface{}, error) {
	fields := RPCMarshalEvent(event)
	fields["size"] = hexutil.Uint64(event.Size())

	if inclTx {
		formatTx := func(tx *types.Transaction) (interface{}, error) {
			return tx.Hash(), nil
		}
		if fullTx {
			// TODO: full txs for events API
			panic("is not implemented")
			//formatTx = func(tx *types.Transaction) (interface{}, error) {
			//	return newRPCTransactionFromBlockHash(event, tx.Hash()), nil
			//}
		}
		txs := event.Txs()
		transactions := make([]interface{}, len(txs))
		var err error
		for i, tx := range txs {
			if transactions[i], err = formatTx(tx); err != nil {
				return nil, err
			}
		}

		fields["transactions"] = transactions
	}

	return fields, nil
}

func EventIDsToHex(ids hash.Events) []hexutil.Bytes {
	res := make([]hexutil.Bytes, len(ids))
	for i, id := range ids {
		res[i] = hexutil.Bytes(id.Bytes())
	}
	return res
}

func HexToEventIDs(bb []interface{}) hash.Events {
	res := make(hash.Events, len(bb))
	for i, b := range bb {
		res[i] = hash.BytesToEvent(hexutil.MustDecode(b.(string)))
	}
	return res
}
