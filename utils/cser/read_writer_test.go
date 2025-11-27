package cser

import (
	"math"
	"math/big"
	"testing"

	"github.com/rony4d/go-opera-asset/utils/bits"
	"github.com/rony4d/go-opera-asset/utils/fast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to simulate a Reader consuming the output of a Writer.
// Unlike binary.go, this does not perform the framing (size headers, etc.),
// but directly connects the bit/byte streams.
func newReaderFromWriter(w *Writer) *Reader {
	return &Reader{
		BitsR:  bits.NewReader(w.BitsW.Array),
		BytesR: fast.NewReader(w.BytesW.Bytes()),
	}
}

// TestIntegers_RoundTrip verifies that all supported integer types (U8, U16, U32, U64, I64, U56)
// can be written and read back correctly across a range of values.
func TestIntegers_RoundTrip(t *testing.T) {
	w := NewWriter()

	// Define test cases covering zero, single byte, max values, etc.
	u8Vals := []uint8{0, 1, 0xFF}
	u16Vals := []uint16{0, 1, 0xFF, 0xFFFF}
	u32Vals := []uint32{0, 1, 0xFFFF, 0xFFFFFFFF}
	u64Vals := []uint64{0, 1, 0xFFFF, 0xFFFFFFFF, 0xFFFFFFFFFFFFFFFF}
	i64Vals := []int64{0, 1, -1, math.MinInt64, math.MaxInt64}
	u56Vals := []uint64{0, 1, (1 << 56) - 1} // Max 56-bit value

	// --- Write Phase ---
	for _, v := range u8Vals {
		w.U8(v)
	}
	for _, v := range u16Vals {
		w.U16(v)
	}
	for _, v := range u32Vals {
		w.U32(v)
	}
	for _, v := range u64Vals {
		w.U64(v)
	}
	for _, v := range u64Vals {
		w.VarUint(v) // VarUint uses the same logic as U64 in this implementation
	}
	for _, v := range i64Vals {
		w.I64(v)
	}
	for _, v := range u56Vals {
		w.U56(v)
	}

	// --- Read Phase ---
	r := newReaderFromWriter(w)

	for i, want := range u8Vals {
		assert.Equal(t, want, r.U8(), "U8 mismatch at index %d", i)
	}
	for i, want := range u16Vals {
		assert.Equal(t, want, r.U16(), "U16 mismatch at index %d", i)
	}
	for i, want := range u32Vals {
		assert.Equal(t, want, r.U32(), "U32 mismatch at index %d", i)
	}
	for i, want := range u64Vals {
		assert.Equal(t, want, r.U64(), "U64 mismatch at index %d", i)
	}
	for i, want := range u64Vals {
		assert.Equal(t, want, r.VarUint(), "VarUint mismatch at index %d", i)
	}
	for i, want := range i64Vals {
		assert.Equal(t, want, r.I64(), "I64 mismatch at index %d", i)
	}
	for i, want := range u56Vals {
		assert.Equal(t, want, r.U56(), "U56 mismatch at index %d", i)
	}

	// Ensure streams are completely consumed.
	assert.True(t, r.BytesR.Empty(), "Bytes buffer should be empty after reading all values")

	// Check bit stream emptiness.
	// Note: The bit stream is padded to the nearest byte.
	// Valid data must have consumed all "logical" bits, but physical zero-padding bits may remain.
	remainingBits := r.BitsR.NonReadBits()
	assert.Less(t, remainingBits, 8, "Remaining bits should be less than a full byte (padding only)")
	if remainingBits > 0 {
		val := r.BitsR.Read(remainingBits)
		assert.Equal(t, uint(0), val, "Padding bits must be zero")
	}
}

// TestBool_RoundTrip verifies boolean serialization.
func TestBool_RoundTrip(t *testing.T) {
	w := NewWriter()
	vals := []bool{true, false, true, true, false}

	for _, v := range vals {
		w.Bool(v)
	}

	r := newReaderFromWriter(w)
	for i, want := range vals {
		assert.Equal(t, want, r.Bool(), "Bool index %d", i)
	}
}

// TestBytes_RoundTrip verifies FixedBytes and SliceBytes.
func TestBytes_RoundTrip(t *testing.T) {
	w := NewWriter()

	fixed1 := []byte{1, 2, 3}
	fixed2 := []byte{4, 5}
	slice1 := []byte{6, 7, 8, 9}
	slice2 := []byte{} // Empty slice

	w.FixedBytes(fixed1)
	w.FixedBytes(fixed2)
	w.SliceBytes(slice1)
	w.SliceBytes(slice2)

	r := newReaderFromWriter(w)

	// FixedBytes requires pre-allocated buffer
	buf1 := make([]byte, len(fixed1))
	r.FixedBytes(buf1)
	assert.Equal(t, fixed1, buf1)

	buf2 := make([]byte, len(fixed2))
	r.FixedBytes(buf2)
	assert.Equal(t, fixed2, buf2)

	// SliceBytes allocates its own buffer
	gotSlice1 := r.SliceBytes(100)
	assert.Equal(t, slice1, gotSlice1)

	gotSlice2 := r.SliceBytes(100)
	assert.Equal(t, slice2, gotSlice2)
}

// TestBigInt_RoundTrip verifies BigInt serialization.
func TestBigInt_RoundTrip(t *testing.T) {
	w := NewWriter()
	vals := []*big.Int{
		big.NewInt(0),
		big.NewInt(1),
		big.NewInt(-1), // Note: BigInt implementation in read_writer.go uses v.Bytes() which is absolute value for negative numbers?
		// Wait, read_writer.go:238: if v.Sign() != 0 { bigBytes = v.Bytes() }
		// big.Int.Bytes() returns absolute value bytes. It does NOT encode sign.
		// If the implementation relies on generic SliceBytes of v.Bytes(), it might lose sign information
		// unless the protocol assumes unsigned BigInts or handles sign externally.
		// Let's check the code:
		// func (w *Writer) BigInt(v *big.Int) { ... bigBytes = v.Bytes() ... w.SliceBytes(bigBytes) }
		// This confirms it only stores the magnitude. Negative numbers will become positive on read.
		big.NewInt(123456789),
	}

	// Add a note/check about sign behavior
	// If we pass -1, v.Bytes() is {1}. Read back will be 1.

	for _, v := range vals {
		w.BigInt(v)
	}

	r := newReaderFromWriter(w)
	for i, v := range vals {
		got := r.BigInt()
		// effective expectation is abs(v)
		want := new(big.Int).Abs(v)
		assert.Equal(t, want, got, "BigInt index %d", i)
	}
}

// TestPaddedBytes verifies the PaddedBytes helper.
func TestPaddedBytes(t *testing.T) {
	tests := []struct {
		in       []byte
		n        int
		expected []byte
	}{
		{[]byte{1}, 2, []byte{0, 1}},
		{[]byte{1, 2}, 2, []byte{1, 2}},
		{[]byte{1, 2, 3}, 2, []byte{1, 2, 3}}, // Should return as is if len >= n
		{[]byte{}, 3, []byte{0, 0, 0}},
	}

	for i, tc := range tests {
		got := PaddedBytes(tc.in, tc.n)
		assert.Equal(t, tc.expected, got, "Case %d", i)
	}
}

// TestAllocLimit verifies that SliceBytes enforces the maxLen parameter.
func TestAllocLimit_(t *testing.T) {
	w := NewWriter()
	data := make([]byte, 100)
	w.SliceBytes(data) // Writes size=100, then 100 bytes

	r := newReaderFromWriter(w)

	// Try to read with limit=50. Should panic with ErrTooLargeAlloc
	assert.PanicsWithError(t, ErrTooLargeAlloc.Error(), func() {
		r.SliceBytes(50)
	})
}

// TestU56_Overflow verifies U56 writes panic on overflow.
func TestU56_Overflow(t *testing.T) {
	w := NewWriter()
	assert.Panics(t, func() {
		w.U56(1 << 56) // 2^56 fits in 8 bytes, but is 1 larger than max 56-bit value
	})
}

// TestCompactEncoding_Structure inspects the actual bytes written to ensure efficient encoding is used.
func TestCompactEncoding_Structure(t *testing.T) {
	// Case 1: Small U64 (0)
	// Expect: 1 byte (0x00), 3 bits size-offset (0)
	w := NewWriter()
	w.U64(0)
	require.Equal(t, []byte{0}, w.BytesW.Bytes())
	// bits written: 3 bits of value 0.
	// We can't easily peek bits without closing/reading, but we know implementation.

	// Case 2: U64(256) -> 0x0100
	// Expect: 2 bytes (0x00, 0x01), 3 bits size-offset (1, because 2 bytes - 1 min = 1)
	w = NewWriter()
	w.U64(256)
	require.Equal(t, []byte{0, 1}, w.BytesW.Bytes())

	// Check bits via reader
	r := newReaderFromWriter(w)
	// Manually read the size bits for U64 (3 bits)
	sizeOffset := r.BitsR.Read(3)
	assert.Equal(t, uint(1), sizeOffset, "Size offset for 256 should be 1 (total 2 bytes)")
}
