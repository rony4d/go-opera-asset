package inter

import (
	"bytes"
	"encoding/json"
	"math"
	"math/big"
	"math/rand"
	"testing"

	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// emptyEvent creates a minimal EventPayload with the given version.
// This helps testing boundary conditions where fields are empty or zero.
func emptyEvent(ver uint8) EventPayload {
	empty := MutableEventPayload{}
	empty.SetVersion(ver)
	if ver == 0 {
		empty.SetEpoch(256) // Version 0 requires epoch >= 256
	}
	empty.SetParents(hash.Events{})
	empty.SetExtra([]byte{})
	empty.SetTxs(types.Transactions{})
	empty.SetPayloadHash(EmptyPayloadHash(ver))
	return *empty.Build()
}

// TestEventPayloadSerialization_RoundTrip verifies that EventPayloads can be successfully
// encoded to RLP and decoded back without data loss.

func TestEventPayloadSerialization_RoundTrip(t *testing.T) {
	// 1. Construct a "maximal" event with all fields populated with large values.
	max := MutableEventPayload{}
	max.SetEpoch(math.MaxUint32)
	max.SetSeq(idx.Event(math.MaxUint32))
	max.SetLamport(idx.Lamport(math.MaxUint32))
	h := hash.BytesToEvent(bytes.Repeat([]byte{math.MaxUint8}, 32))
	max.SetParents(hash.Events{hash.Event(h), hash.Event(h), hash.Event(h)})
	max.SetPayloadHash(hash.Hash(h))
	max.SetSig(BytesToSignature(bytes.Repeat([]byte{math.MaxUint8}, SigSize)))
	max.SetExtra(bytes.Repeat([]byte{math.MaxUint8}, 100))
	max.SetCreationTime(math.MaxUint64)
	max.SetMedianTime(math.MaxUint64)

	// Add various transaction types to test complex payload serialization
	tx1 := types.NewTx(&types.LegacyTx{
		Nonce:    math.MaxUint64,
		GasPrice: h.Big(),
		Gas:      math.MaxUint64,
		To:       nil, // Contract creation
		Value:    h.Big(),
		Data:     []byte{},
		V:        big.NewInt(0xff),
		R:        h.Big(),
		S:        h.Big(),
	})
	tx2 := types.NewTx(&types.LegacyTx{
		Nonce:    math.MaxUint64,
		GasPrice: h.Big(),
		Gas:      math.MaxUint64,
		To:       &common.Address{},
		Value:    h.Big(),
		Data:     max.extra,
		V:        big.NewInt(0xff),
		R:        h.Big(),
		S:        h.Big(),
	})
	txs := types.Transactions{}
	for i := 0; i < 200; i++ {
		txs = append(txs, tx1)
		txs = append(txs, tx2)
	}
	max.SetTxs(txs)

	// Define a suite of test cases
	cases := map[string]EventPayload{
		"empty_v0": emptyEvent(0),
		"empty_v1": emptyEvent(1),
		"max":      *max.Build(),
		"random":   *FakeEvent(12, 1, 1, true),
	}

	for name, original := range cases {
		t.Run(name, func(t *testing.T) {
			// Encode
			buf, err := rlp.EncodeToBytes(&original)
			require.NoError(t, err, "RLP encoding failed")

			// Decode
			var decoded EventPayload
			err = rlp.DecodeBytes(buf, &decoded)
			require.NoError(t, err, "RLP decoding failed")

			// Verify Fields
			assert.EqualValues(t, original.extEventData, decoded.extEventData, "External event data mismatch")
			assert.EqualValues(t, original.sigData, decoded.sigData, "Signature data mismatch")

			require.Equal(t, len(original.payloadData.txs), len(decoded.payloadData.txs), "Tx count mismatch")
			for i := range original.payloadData.txs {
				assert.EqualValues(t, original.payloadData.txs[i].Hash(), decoded.payloadData.txs[i].Hash(), "Tx hash mismatch at index %d", i)
			}

			assert.EqualValues(t, original.baseEvent, decoded.baseEvent, "Base event mismatch")
			assert.EqualValues(t, original.ID(), decoded.ID(), "ID mismatch")
			assert.EqualValues(t, original.HashToSign(), decoded.HashToSign(), "HashToSign mismatch")
			assert.EqualValues(t, original.Size(), decoded.Size(), "Size mismatch")
		})
	}
}
func TestEventPayloadSerialization(t *testing.T) {
	max := MutableEventPayload{}
	max.SetEpoch(math.MaxUint32)
	max.SetSeq(idx.Event(math.MaxUint32))
	max.SetLamport(idx.Lamport(math.MaxUint32))
	h := hash.BytesToEvent(bytes.Repeat([]byte{math.MaxUint8}, 32))
	max.SetParents(hash.Events{hash.Event(h), hash.Event(h), hash.Event(h)})
	max.SetPayloadHash(hash.Hash(h))
	max.SetSig(BytesToSignature(bytes.Repeat([]byte{math.MaxUint8}, SigSize)))
	max.SetExtra(bytes.Repeat([]byte{math.MaxUint8}, 100))
	max.SetCreationTime(math.MaxUint64)
	max.SetMedianTime(math.MaxUint64)
	tx1 := types.NewTx(&types.LegacyTx{
		Nonce:    math.MaxUint64,
		GasPrice: h.Big(),
		Gas:      math.MaxUint64,
		To:       nil,
		Value:    h.Big(),
		Data:     []byte{},
		V:        big.NewInt(0xff),
		R:        h.Big(),
		S:        h.Big(),
	})
	tx2 := types.NewTx(&types.LegacyTx{
		Nonce:    math.MaxUint64,
		GasPrice: h.Big(),
		Gas:      math.MaxUint64,
		To:       &common.Address{},
		Value:    h.Big(),
		Data:     max.extra,
		V:        big.NewInt(0xff),
		R:        h.Big(),
		S:        h.Big(),
	})
	txs := types.Transactions{}
	for i := 0; i < 200; i++ {
		txs = append(txs, tx1)
		txs = append(txs, tx2)
	}
	max.SetTxs(txs)

	ee := map[string]EventPayload{
		"empty0": emptyEvent(0),
		"empty1": emptyEvent(1),
		"max":    *max.Build(),
		"random": *FakeEvent(12, 1, 1, true),
	}

	t.Run("ok", func(t *testing.T) {
		require := require.New(t)

		for name, header0 := range ee {
			buf, err := rlp.EncodeToBytes(&header0)
			require.NoError(err)

			var header1 EventPayload
			err = rlp.DecodeBytes(buf, &header1)
			require.NoError(err, name)

			require.EqualValues(header0.extEventData, header1.extEventData, name)
			require.EqualValues(header0.sigData, header1.sigData, name)
			for i := range header0.payloadData.txs {
				require.EqualValues(header0.payloadData.txs[i].Hash(), header1.payloadData.txs[i].Hash(), name)
			}
			require.EqualValues(header0.baseEvent, header1.baseEvent, name)
			require.EqualValues(header0.ID(), header1.ID(), name)
			require.EqualValues(header0.HashToSign(), header1.HashToSign(), name)
			require.EqualValues(header0.Size(), header1.Size(), name)
		}
	})

	t.Run("err", func(t *testing.T) {
		require := require.New(t)

		for name, header0 := range ee {
			bin, err := header0.MarshalBinary()
			require.NoError(err, name)

			n := rand.Intn(len(bin) - len(header0.Extra()) - 1)
			bin = bin[0:n]

			buf, err := rlp.EncodeToBytes(bin)
			require.NoError(err, name)

			var header1 Event
			err = rlp.DecodeBytes(buf, &header1)
			require.Error(err, name)
		}
	})
}

// TestEventPayloadSerialization_Corrupted verifies that the decoder correctly rejects
// truncated or malformed binary data.
func TestEventPayloadSerialization_Corrupted(t *testing.T) {
	cases := map[string]EventPayload{
		"empty_v0": emptyEvent(0),
		"random":   *FakeEvent(12, 1, 1, true),
	}

	for name, original := range cases {
		t.Run(name, func(t *testing.T) {
			// Get valid binary encoding (inner CSER)
			bin, err := original.MarshalBinary()
			require.NoError(t, err)

			// Truncate the binary data
			// We ensure we cut off at least one byte, but don't cut into the fixed header if possible?
			// Actually, cutting randomly is fine for testing "corrupted" input.
			if len(bin) > len(original.Extra())+1 {
				n := rand.Intn(len(bin) - len(original.Extra()) - 1)
				bin = bin[0:n]
			} else {
				bin = bin[0 : len(bin)-1]
			}

			// Wrap the truncated binary in RLP
			buf, err := rlp.EncodeToBytes(bin)
			require.NoError(t, err)

			// Attempt decode
			var decoded Event
			err = rlp.DecodeBytes(buf, &decoded)
			require.Error(t, err, "Should fail to decode truncated data")
		})
	}
}

// TestEventRPCMarshaling verifies the JSON RPC marshaling logic for Events and EventPayloads.
// It ensures that fields are correctly mapped to their JSON representation and back.
func TestEventRPCMarshaling(t *testing.T) {
	t.Run("Event", func(t *testing.T) {
		for i := 0; i < 3; i++ {
			// Create a fake event (subset of payload)
			payload := FakeEvent(i, i, i, i != 0)
			event0 := &payload.Event

			// Marshal to RPC format (map[string]interface{})
			mapping := RPCMarshalEvent(event0)

			// Marshal to JSON bytes
			bb, err := json.Marshal(mapping)
			require.NoError(t, err)

			// Unmarshal JSON back to map
			mapping = make(map[string]interface{})
			err = json.Unmarshal(bb, &mapping)
			require.NoError(t, err)

			// Unmarshal map back to Event struct
			event1 := RPCUnmarshalEvent(mapping)

			assert.Equal(t, event0, event1, "Event mismatch after RPC roundtrip")
		}
	})

	t.Run("EventPayload", func(t *testing.T) {
		for i := 0; i < 3; i++ {
			event0 := FakeEvent(i, i, i, i != 0)

			// Marshal payload to RPC map
			mapping, err := RPCMarshalEventPayload(event0, true, false)
			require.NoError(t, err)

			// Marshal to JSON
			bb, err := json.Marshal(mapping)
			require.NoError(t, err)

			// Unmarshal JSON back to map
			mapping = make(map[string]interface{})
			err = json.Unmarshal(bb, &mapping)
			require.NoError(t, err)

			// Unmarshal map back to Event struct (RPCUnmarshalEvent only returns Event part)
			event1 := RPCUnmarshalEvent(mapping)

			// Verify the Base Event part matches
			assert.Equal(t, &event0.SignedEvent.Event, event1, "Event base mismatch after RPC roundtrip")
		}
	})
}

// --- Benchmarks ---

func BenchmarkEventPayload_EncodeRLP_empty(b *testing.B) {
	e := emptyEvent(0)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf, err := rlp.EncodeToBytes(&e)
		if err != nil {
			b.Fatal(err)
		}
		b.ReportMetric(float64(len(buf)), "size")
	}
}

func BenchmarkEventPayload_EncodeRLP_NoPayload(b *testing.B) {
	e := FakeEvent(0, 0, 0, false)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf, err := rlp.EncodeToBytes(&e)
		if err != nil {
			b.Fatal(err)
		}
		b.ReportMetric(float64(len(buf)), "size")
	}
}

func BenchmarkEventPayload_EncodeRLP(b *testing.B) {
	e := FakeEvent(1000, 0, 0, false)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf, err := rlp.EncodeToBytes(&e)
		if err != nil {
			b.Fatal(err)
		}
		b.ReportMetric(float64(len(buf)), "size")
	}
}

func BenchmarkEventPayload_DecodeRLP_empty(b *testing.B) {
	e := emptyEvent(0)
	me := MutableEventPayload{}
	buf, err := rlp.EncodeToBytes(&e)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = rlp.DecodeBytes(buf, &me)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEventPayload_DecodeRLP_NoPayload(b *testing.B) {
	e := FakeEvent(0, 0, 0, false)
	me := MutableEventPayload{}
	buf, err := rlp.EncodeToBytes(&e)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = rlp.DecodeBytes(buf, &me)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEventPayload_DecodeRLP(b *testing.B) {
	e := FakeEvent(1000, 0, 0, false)
	me := MutableEventPayload{}
	buf, err := rlp.EncodeToBytes(&e)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = rlp.DecodeBytes(buf, &me)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// --- Helpers for Fake Data Generation ---

func randBig(r *rand.Rand) *big.Int {
	b := make([]byte, r.Intn(8))
	_, _ = r.Read(b)
	if len(b) == 0 {
		b = []byte{0}
	}
	return new(big.Int).SetBytes(b)
}

func randAddr(r *rand.Rand) common.Address {
	addr := common.Address{}
	r.Read(addr[:])
	return addr
}

func randBytes(r *rand.Rand, size int) []byte {
	b := make([]byte, size)
	r.Read(b)
	return b
}

func randHash(r *rand.Rand) hash.Hash {
	return hash.BytesToHash(randBytes(r, 32))
}

func randAddrPtr(r *rand.Rand) *common.Address {
	addr := randAddr(r)
	return &addr
}

func randAccessList(r *rand.Rand, maxAddrs, maxKeys int) types.AccessList {
	accessList := make(types.AccessList, r.Intn(maxAddrs))
	for i := range accessList {
		accessList[i].Address = randAddr(r)
		accessList[i].StorageKeys = make([]common.Hash, r.Intn(maxKeys))
		for j := range accessList[i].StorageKeys {
			r.Read(accessList[i].StorageKeys[j][:])
		}
	}
	return accessList
}

// FakeEvent generates random event for testing purpose.
// It populates the event with a configurable number of transactions, misbehavior proofs, and votes.
func FakeEvent(txsNum, mpsNum, bvsNum int, ersNum bool) *EventPayload {
	r := rand.New(rand.NewSource(int64(0)))
	random := &MutableEventPayload{}
	random.SetVersion(1)
	random.SetNetForkID(uint16(r.Uint32() >> 16))
	random.SetLamport(1000)
	random.SetExtra([]byte{byte(r.Uint32())})
	random.SetSeq(idx.Event(r.Uint32() >> 8))
	random.SetCreator(idx.ValidatorID(r.Uint32()))
	random.SetFrame(idx.Frame(r.Uint32() >> 16))
	random.SetCreationTime(Timestamp(r.Uint64()))
	random.SetMedianTime(Timestamp(r.Uint64()))
	random.SetGasPowerUsed(r.Uint64())
	random.SetGasPowerLeft(GasPowerLeft{[2]uint64{r.Uint64(), r.Uint64()}})

	// Generate Transactions
	txs := types.Transactions{}
	for i := 0; i < txsNum; i++ {
		h := hash.Hash{}
		r.Read(h[:])
		var tx *types.Transaction

		switch i % 3 {
		case 0: // LegacyTx
			tx = types.NewTx(&types.LegacyTx{
				Nonce:    r.Uint64(),
				GasPrice: randBig(r),
				Gas:      257 + r.Uint64(),
				To:       nil,
				Value:    randBig(r),
				Data:     randBytes(r, r.Intn(300)),
				V:        big.NewInt(int64(r.Intn(0xffffffff))),
				R:        h.Big(),
				S:        h.Big(),
			})
		case 1: // AccessListTx
			tx = types.NewTx(&types.AccessListTx{
				ChainID:    randBig(r),
				Nonce:      r.Uint64(),
				GasPrice:   randBig(r),
				Gas:        r.Uint64(),
				To:         randAddrPtr(r),
				Value:      randBig(r),
				Data:       randBytes(r, r.Intn(300)),
				AccessList: randAccessList(r, 300, 300),
				V:          big.NewInt(int64(r.Intn(0xffffffff))),
				R:          h.Big(),
				S:          h.Big(),
			})
		case 2: // DynamicFeeTx
			tx = types.NewTx(&types.DynamicFeeTx{
				ChainID:    randBig(r),
				Nonce:      r.Uint64(),
				GasTipCap:  randBig(r),
				GasFeeCap:  randBig(r),
				Gas:        r.Uint64(),
				To:         randAddrPtr(r),
				Value:      randBig(r),
				Data:       randBytes(r, r.Intn(300)),
				AccessList: randAccessList(r, 300, 300),
				V:          big.NewInt(int64(r.Intn(0xffffffff))),
				R:          h.Big(),
				S:          h.Big(),
			})
		}
		txs = append(txs, tx)
	}
	random.SetTxs(txs)

	// Generate Misbehaviour Proofs
	mps := []MisbehaviourProof{}
	for i := 0; i < mpsNum; i++ {
		mps = append(mps, MisbehaviourProof{
			EventsDoublesign: &EventsDoublesign{
				Pair: [2]SignedEventLocator{SignedEventLocator{}, SignedEventLocator{}},
			},
		})
	}
	random.SetMisbehaviourProofs(mps)

	// Generate Block Votes
	bvs := LlrBlockVotes{}
	if bvsNum > 0 {
		bvs.Start = 1 + idx.Block(rand.Intn(1000))
		bvs.Epoch = 1 + idx.Epoch(rand.Intn(1000))
	}
	for i := 0; i < bvsNum; i++ {
		bvs.Votes = append(bvs.Votes, randHash(r))
	}
	random.SetBlockVotes(bvs)

	// Generate Epoch Vote
	ers := LlrEpochVote{}
	if ersNum {
		ers.Epoch = 1 + idx.Epoch(rand.Intn(1000))
		ers.Vote = randHash(r)
	}
	random.SetEpochVote(ers)

	// Finalize
	random.SetPayloadHash(CalcPayloadHash(random))

	parent := MutableEventPayload{}
	parent.SetVersion(1)
	parent.SetLamport(random.Lamport() - 500)
	parent.SetEpoch(random.Epoch())
	random.SetParents(hash.Events{parent.Build().ID()})

	return random.Build()
}
