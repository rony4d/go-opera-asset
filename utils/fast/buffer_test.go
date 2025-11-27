package fast

import (
	"bytes"
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestBuffer_Integration verifies the complete lifecycle of writing and reading.
// It ensures that data written via Writer is correctly retrieved via Reader.
func TestBuffer_Integration(t *testing.T) {
	const N = 100
	var (
		w *Writer
		r *Reader
		// Custom byte sequence to test bulk writing/reading
		extraData = []byte{0, 0, 0xFF, 9, 0}
	)

	// Phase 1: Verify Write Operations
	t.Run("Writer", func(t *testing.T) {
		require := require.New(t)

		// Initialize Writer with initial capacity but 0 length
		w = NewWriter(make([]byte, 0, N/2))

		// Write sequential bytes 0 to 99
		for i := byte(0); i < N; i++ {
			w.WriteByte(i)
		}

		// Verify length matches number of written bytes
		require.Equal(N, len(w.Bytes()), "Writer should contain N bytes")

		// Append a bulk slice of bytes
		w.Write(extraData)

		// Verify total length includes both sequential and bulk data
		require.Equal(N+len(extraData), len(w.Bytes()), "Writer should contain N + extra bytes")
	})

	// Phase 2: Verify Read Operations using the data written in Phase 1
	t.Run("Reader", func(t *testing.T) {
		require := require.New(t)

		// Initialize Reader with the buffer from the Writer
		r = NewReader(w.Bytes())

		// 1. Check initial state
		require.Equal(N+len(extraData), len(r.Bytes()), "Reader buffer size mismatch")
		require.False(r.Empty(), "New reader should not be empty")
		require.Equal(0, r.Position(), "New reader should start at position 0")

		// 2. Verify sequential single-byte reads match written values
		for exp := byte(0); exp < N; exp++ {
			got := r.ReadByte()
			require.Equal(exp, got, "ReadByte mismatch at index %d", exp)
		}

		// 3. Verify current position matches number of bytes read so far
		require.Equal(N, r.Position(), "Position should match number of bytes read")

		// 4. Verify bulk read matches the appended extraData
		got := r.Read(len(extraData))
		require.Equal(extraData, got, "Read() mismatch for bulk data")

		// 5. Verify final state
		require.True(r.Empty(), "Reader should be empty after reading all bytes")
		require.Equal(N+len(extraData), r.Position(), "Final position should match total length")
	})
}

// TestBuffer_Boundaries adds specific checks for edge cases like empty buffers,
// single-byte buffers, and partial reads.
func TestBuffer_Boundaries(t *testing.T) {
	t.Run("Empty Buffer", func(t *testing.T) {
		r := NewReader([]byte{})
		require.True(t, r.Empty(), "Reader initialized with empty slice should be empty")
		require.Equal(t, 0, r.Position())
	})

	t.Run("Partial Reads", func(t *testing.T) {
		data := []byte{1, 2, 3, 4, 5}
		r := NewReader(data)

		// Read first 2 bytes
		chunk1 := r.Read(2)
		require.Equal(t, []byte{1, 2}, chunk1)
		require.Equal(t, 2, r.Position())
		require.False(t, r.Empty())

		// Read next 1 byte
		b := r.ReadByte()
		require.Equal(t, byte(3), b)
		require.Equal(t, 3, r.Position())

		// Read remaining 2 bytes
		chunk2 := r.Read(2)
		require.Equal(t, []byte{4, 5}, chunk2)
		require.True(t, r.Empty())
	})

	t.Run("Write to nil buffer", func(t *testing.T) {
		// Verify Writer handles nil initialization gracefully (append works on nil slices)
		w := NewWriter(nil)
		w.WriteByte(0xAA)
		require.Equal(t, []byte{0xAA}, w.Bytes())
	})
}

// Benchmark compares the custom fast buffer implementation against standard library
// bytes.Buffer (for writes) and bytes.Reader (for reads).
func Benchmark(b *testing.B) {
	b.Run("Write", func(b *testing.B) {
		b.Run("Std", func(b *testing.B) {
			w := bytes.NewBuffer(make([]byte, 0, b.N))
			for i := 0; i < b.N; i++ {
				w.WriteByte(byte(i))
			}
			// Sanity check to ensure compiler doesn't optimize away the loop
			require.Equal(b, b.N, len(w.Bytes()))
		})
		b.Run("Fast", func(b *testing.B) {
			w := NewWriter(make([]byte, 0, b.N))
			for i := 0; i < b.N; i++ {
				w.WriteByte(byte(i))
			}
			require.Equal(b, b.N, len(w.Bytes()))
		})
	})

	b.Run("Read", func(b *testing.B) {
		src := make([]byte, 1000)
		rand.Read(src)

		b.Run("Std", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				r := bytes.NewReader(src)
				for j := 0; j < len(src); j++ {
					_, _ = r.ReadByte()
				}
			}
		})
		b.Run("Fast", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				r := NewReader(src)
				for j := 0; j < len(src); j++ {
					_ = r.ReadByte()
				}
			}
		})
	})
}
