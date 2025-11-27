package cser

import (
	"crypto/rand"
	"errors"
	"math"
	"math/big"
	"testing"

	"github.com/rony4d/go-opera-asset/utils/fast"
	"github.com/stretchr/testify/require"
)

// TestEmpty verifies behavior when marshaling/unmarshaling empty or nil structures.
func TestEmpty(t *testing.T) {
	var (
		buf []byte
		err error
	)

	// Verify that an empty writer produces a valid (likely empty or minimal header) output.
	t.Run("Write", func(t *testing.T) {
		buf, err = MarshalBinaryAdapter(func(w *Writer) error {
			// No operations performed
			return nil
		})
		require.NoError(t, err)
	})

	// Verify that the output from the empty writer can be unmarshaled successfully.
	t.Run("Read", func(t *testing.T) {
		err = UnmarshalBinaryAdapter(buf, func(r *Reader) error {
			// No operations expected
			return nil
		})
		require.NoError(t, err)
	})
}

// TestErr verifies error handling during marshaling and unmarshaling.
// It checks propagation of custom errors and detection of malformed/non-canonical encoding.
func TestErr(t *testing.T) {
	var (
		buf []byte
	)

	// Helper to safely copy the buffer for independent test cases
	bufCopy := func() []byte {
		bb := make([]byte, len(buf))
		copy(bb, buf)
		return bb
	}

	t.Run("Write", func(t *testing.T) {
		require := require.New(t)

		// 1. Successful write: Serialize MaxUint64
		bb, err := MarshalBinaryAdapter(func(w *Writer) error {
			w.U64(math.MaxUint64)
			return nil
		})
		require.NoError(err)
		buf = append(buf, bb...)

		// 2. Failed write: Serialize a Bool but return a custom error
		errExp := errors.New("custom")
		bb, err = MarshalBinaryAdapter(func(w *Writer) error {
			w.Bool(false)
			return errExp
		})
		require.Equal(errExp, err)
		// Even though it failed, we append to test 'garbage' handling or partial streams?
		// In this test structure, it seems to be building a specific corrupted scenario or just testing limits.
		buf = append(buf, bb...)
	})

	// Verify that unmarshaling nil input returns ErrMalformedEncoding
	t.Run("Read nil", func(t *testing.T) {
		require := require.New(t)
		err := UnmarshalBinaryAdapter(nil, func(r *Reader) error {
			return nil
		})
		require.Equal(ErrMalformedEncoding, err)
	})

	// Verify that a custom error returned from the unmarshal callback is propagated correctly
	t.Run("Read err", func(t *testing.T) {
		require := require.New(t)

		errExp := errors.New("custom")
		err := UnmarshalBinaryAdapter(buf, func(r *Reader) error {
			// Read the valid part (MaxUint64) first
			require.Equal(uint64(math.MaxUint64), r.U64())
			// Return custom error
			return errExp
		})
		require.Equal(errExp, err)
	})

	// Verify handling of corrupted size headers
	t.Run("Read corrupted size", func(t *testing.T) {
		require := require.New(t)
		// Deconstruct the valid buffer
		_, bbytes, err := binaryToCSER(bufCopy())
		require.NoError(err)

		// Reconstruct with a manipulated size header
		corrupted := fast.NewWriter(bbytes)
		sizeWriter := fast.NewWriter(make([]byte, 0, 4))
		// Intentionally report a size 1 byte larger than actual
		writeUint64Compact(sizeWriter, uint64(len(bbytes)+1))
		corrupted.Write(reversed(sizeWriter.Bytes()))

		// 1. Check binaryToCSER level error
		_, _, err = binaryToCSER(corrupted.Bytes())
		require.Equal(ErrMalformedEncoding, err)

		// 2. Check UnmarshalBinaryAdapter level error
		err = UnmarshalBinaryAdapter(corrupted.Bytes(), func(r *Reader) error {
			require.Equal(uint64(math.MaxUint64), r.U64())
			return nil
		})
		require.Equal(ErrMalformedEncoding, err)
	})

	// Helper to create defects in the binary structure
	repackWithDefect := func(
		defect func(bbits, bbytes *[]byte) (expected error),
	) func(t *testing.T) {
		return func(t *testing.T) {
			require := require.New(t)
			// Unpack valid buffer
			bbits, bbytes, err := binaryToCSER(bufCopy())
			require.NoError(err)

			// Apply defect
			errExp := defect(&bbits.Bytes, &bbytes)

			// Repack
			corrupted, err := binaryFromCSER(bbits, bbytes)
			require.NoError(err)

			// Attempt unmarshal
			err = UnmarshalBinaryAdapter(corrupted, func(r *Reader) error {
				_ = r.U64()
				return nil
			})
			require.Equal(errExp, err)
		}
	}

	// Case 1: No defect, should pass (nil error)
	t.Run("Read Valid", repackWithDefect(func(bbits, bbytes *[]byte) (expected error) {
		return nil
	}))

	// Case 2: Extra byte in 'bytes' section -> NonCanonical
	t.Run("Read Extra Bytes", repackWithDefect(func(bbits, bbytes *[]byte) (expected error) {
		*bbytes = append(*bbytes, 0xFF)
		return ErrNonCanonicalEncoding
	}))

	// Case 3: Extra byte in 'bits' section -> NonCanonical
	t.Run("Read Extra Bits", repackWithDefect(func(bbits, bbytes *[]byte) (expected error) {
		*bbits = append(*bbits, 0x0F)
		return ErrNonCanonicalEncoding
	}))

	// Case 4: Missing byte in 'bytes' section -> Malformed/NonCanonical (depending on impl)
	// Here it results in ErrNonCanonicalEncoding because it probably reads 0s or fails checks?
	// Actually, if we truncate bytes, the reader might panic or return incorrect data.
	// The original test expects ErrNonCanonicalEncoding.
	t.Run("Read Truncated Bytes", repackWithDefect(func(bbits, bbytes *[]byte) (expected error) {
		*bbytes = (*bbytes)[:len(*bbytes)-1]
		return ErrNonCanonicalEncoding
	}))
}

// TestVals verifies correct serialization and deserialization for all supported data types.
func TestVals(t *testing.T) {
	var (
		buf []byte
		err error
	)
	var (
		expBigInt     = []*big.Int{big.NewInt(0), big.NewInt(0xFFFFF)}
		expBool       = []bool{true, false}
		expFixedBytes = [][]byte{[]byte{}, randBytes(0xFF)}
		expSliceBytes = [][]byte{[]byte{}, randBytes(0xFF)}
		expU8         = []uint8{0, 1, 0xFF}
		expU16        = []uint16{0, 1, 0xFFFF}
		expU32        = []uint32{0, 1, 0xFFFFFFFF}
		expU64        = []uint64{0, 1, 0xFFFFFFFFFFFFFFFF}
		expVarUint    = []uint64{0, 1, 0xFFFFFFFFFFFFFFFF}
		expI64        = []int64{0, 1, math.MinInt64, math.MaxInt64}
		expU56        = []uint64{0, 1, 1<<(8*7) - 1}
	)

	// Phase 1: Write all values sequentially
	t.Run("Write", func(t *testing.T) {
		require := require.New(t)

		buf, err = MarshalBinaryAdapter(func(w *Writer) error {
			for _, v := range expBigInt {
				w.BigInt(v)
			}
			for _, v := range expBool {
				w.Bool(v)
			}
			for _, v := range expFixedBytes {
				w.FixedBytes(v)
			}
			for _, v := range expSliceBytes {
				w.SliceBytes(v)
			}
			for _, v := range expU8 {
				w.U8(v)
			}
			for _, v := range expU16 {
				w.U16(v)
			}
			for _, v := range expU32 {
				w.U32(v)
			}
			for _, v := range expU64 {
				w.U64(v)
			}
			for _, v := range expVarUint {
				w.VarUint(v)
			}
			for _, v := range expI64 {
				w.I64(v)
			}
			for _, v := range expU56 {
				w.U56(v)
			}
			return nil
		})
		require.NoError(err)
	})

	// Phase 2: Read all values back and verify equality
	t.Run("Read", func(t *testing.T) {
		require := require.New(t)

		err = UnmarshalBinaryAdapter(buf, func(r *Reader) error {
			for i, exp := range expBigInt {
				got := r.BigInt()
				require.Equal(exp, got, "BigInt index %d", i)
			}
			for i, exp := range expBool {
				got := r.Bool()
				require.Equal(exp, got, "Bool index %d", i)
			}
			for i, exp := range expFixedBytes {
				got := make([]byte, len(exp))
				r.FixedBytes(got)
				require.Equal(exp, got, "FixedBytes index %d", i)
			}
			for i, exp := range expSliceBytes {
				got := r.SliceBytes(255)
				require.Equal(exp, got, "SliceBytes index %d", i)
			}
			for i, exp := range expU8 {
				got := r.U8()
				require.Equal(exp, got, "U8 index %d", i)
			}
			for i, exp := range expU16 {
				got := r.U16()
				require.Equal(exp, got, "U16 index %d", i)
			}
			for i, exp := range expU32 {
				got := r.U32()
				require.Equal(exp, got, "U32 index %d", i)
			}
			for i, exp := range expU64 {
				got := r.U64()
				require.Equal(exp, got, "U64 index %d", i)
			}
			for i, exp := range expVarUint {
				got := r.VarUint()
				require.Equal(exp, got, "VarUint index %d", i)
			}
			for i, exp := range expI64 {
				got := r.I64()
				require.Equal(exp, got, "I64 index %d", i)
			}
			for i, exp := range expU56 {
				got := r.U56()
				require.Equal(exp, got, "U56 index %d", i)
			}
			return nil
		})
		require.NoError(err)
	})
}

// TestBadVals ensures that invalid inputs panic or are handled correctly during writing,
// and that mismatched reads detect inconsistencies (though some are logic errors in the test consumer).
func TestBadVals(t *testing.T) {
	var (
		buf []byte
		err error
	)
	var (
		expBigInt     = []*big.Int{nil}
		expFixedBytes = [][]byte{nil}
		expSliceBytes = [][]byte{nil}
		expU56        = []uint64{1 << (8 * 7), math.MaxUint64} // Values too large for U56
	)

	// Phase 1: Write invalid values (should panic)
	t.Run("Write", func(t *testing.T) {
		require := require.New(t)

		buf, err = MarshalBinaryAdapter(func(w *Writer) error {
			// Nil BigInt panic check
			for _, v := range expBigInt {
				require.Panics(func() {
					w.BigInt(v)
				}, "Should panic on nil BigInt")
			}
			// Nil bytes are treated as empty slices, so no panic expected here
			for _, v := range expFixedBytes {
				w.FixedBytes(v)
			}
			for _, v := range expSliceBytes {
				w.SliceBytes(v)
			}
			// Oversized U56 panic check
			for _, v := range expU56 {
				require.Panics(func() {
					w.U56(v)
				}, "Should panic on oversize U56")
			}
			return nil
		})
		require.NoError(err)
	})

	// Phase 2: Read back the (valid) data written above
	t.Run("Read", func(t *testing.T) {
		require := require.New(t)

		err = UnmarshalBinaryAdapter(buf, func(r *Reader) error {
			// Skip BigInts (none were successfully written)
			for range expBigInt {
				// skip
			}
			// Verify that what we wrote as nil slices comes back as non-nil (empty) slices
			for i, exp := range expFixedBytes {
				got := make([]byte, len(exp))
				r.FixedBytes(got)
				// Original was nil, got is empty slice -> NotEqual (in terms of nil-ness vs empty)
				// But content length is 0 for both.
				require.NotEqual(exp, got, i)
				require.Equal(len(exp), len(got), i)
			}
			for i, exp := range expSliceBytes {
				got := r.SliceBytes(1)
				require.NotEqual(exp, got, i)
				require.Equal(len(exp), len(got), i)
			}
			// Skip U56s (none were successfully written)
			for range expU56 {
				// skip
			}
			return nil
		})
		require.NoError(err)
	})
}

// TestAllocLimit checks if SliceBytes correctly respects the max length limit.
func TestAllocLimit(t *testing.T) {
	require := require.New(t)

	// Create a valid slice of 100 bytes
	data := randBytes(100)
	buf, err := MarshalBinaryAdapter(func(w *Writer) error {
		w.SliceBytes(data)
		return nil
	})
	require.NoError(err)

	// Attempt to read it with a limit smaller than the data size (50 < 100).
	// The Reader.SliceBytes method should panic with ErrTooLargeAlloc.
	// UnmarshalBinaryAdapter recovers from panics and returns ErrMalformedEncoding.
	err = UnmarshalBinaryAdapter(buf, func(r *Reader) error {
		// This call should trigger the panic/error because 100 > 50
		_ = r.SliceBytes(50)
		return nil
	})

	// We expect an error here. Depending on implementation details, it might be
	// ErrMalformedEncoding (if panic recovered) or the panic itself if not caught.
	// Based on UnmarshalBinaryAdapter implementation:
	// if r := recover(); r != nil { err = ErrMalformedEncoding }
	require.Equal(ErrMalformedEncoding, err)
}

// randBytes generates a random byte slice of length n.
func randBytes(n int) []byte {
	bb := make([]byte, n)
	_, err := rand.Read(bb)
	if err != nil {
		panic(err)
	}
	return bb
}
