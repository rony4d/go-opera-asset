package bits

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

// testWord represents a single value to write and read from the bit array.
// 'bits' is the number of bits the value occupies (e.g., 5 bits).
// 'v' is the actual integer value.
type testWord struct {
	bits int
	v    uint
}

// bytesToFit calculates the minimum number of bytes required to store a given number of bits.
// For example, 9 bits require 2 bytes.
func bytesToFit(bits int) int {
	if bits%8 == 0 {
		return bits / 8 // If bits is divisible by 8, return the number of bytes required.
	}
	return bits/8 + 1 // If bits is not divisible by 8, return the number of bytes required plus 1.
}

// genTestWords generates a slice of random testWords for fuzz-like testing.
// maxCount: maximum number of words to generate.
// maxBits: maximum number of bits a single word can use.
func genTestWords(r *rand.Rand, maxCount int, maxBits int) []testWord {
	count := r.Intn(maxCount)
	words := make([]testWord, count)
	for i := range words {
		// Randomize bit length (1 to maxBits)
		if maxBits == 1 {
			words[i].bits = 1
		} else {
			words[i].bits = 1 + r.Intn(maxBits-1)
		}
		// Randomize value that fits within that bit length
		words[i].v = uint(r.Intn(1 << words[i].bits))
	}
	return words
}

// testBitArray is the core assertion logic used by all tests.
// It performs a full cycle:
// 1. Writes all words to a new BitArray.
// 2. Verifies the underlying byte array length.
// 3. Reads all words back and verifies they match the originals.
// 4. Verifies internal state (NonReadBits, NonReadBytes).
// 5. Verifies EOF behavior.
func testBitArray(t *testing.T, words []testWord, name string) {
	arr := Array{make([]byte, 0, 100)}
	writer := NewWriter(&arr)
	reader := NewReader(&arr)

	// --- WRITE PHASE ---
	totalBitsWritten := 0
	for _, w := range words {
		writer.Write(w.bits, w.v)
		totalBitsWritten += w.bits
	}

	// Verify the underlying byte slice grew to the correct size
	expectedBytes := bytesToFit(totalBitsWritten)
	assert.EqualValuesf(t, expectedBytes, len(arr.Bytes), "%s: byte length mismatch", name)

	// --- READ PHASE ---
	totalBitsRead := 0
	for _, w := range words {
		// Check remaining unread bits/bytes before reading
		remainingBits := bytesToFit(totalBitsWritten)*8 - totalBitsRead
		assert.EqualValuesf(t, remainingBits, reader.NonReadBits(), "%s: NonReadBits mismatch before read", name)
		assert.EqualValuesf(t, bytesToFit(reader.NonReadBits()), reader.NonReadBytes(), "%s: NonReadBytes mismatch before read", name)

		// Perform the read
		v := reader.Read(w.bits)
		assert.EqualValuesf(t, w.v, v, "%s: read value mismatch", name)
		totalBitsRead += w.bits

		// Check remaining unread bits/bytes after reading
		remainingBitsAfter := bytesToFit(totalBitsWritten)*8 - totalBitsRead
		assert.EqualValuesf(t, remainingBitsAfter, reader.NonReadBits(), "%s: NonReadBits mismatch after read", name)
		assert.EqualValuesf(t, bytesToFit(reader.NonReadBits()), reader.NonReadBytes(), "%s: NonReadBytes mismatch after read", name)
	}

	// --- BOUNDARY/EOF CHECKS ---

	// 1. Attempting to read past the available bits should panic
	assert.Panicsf(t, func() {
		reader.Read(reader.NonReadBits() + 1)
	}, "%s: should panic when reading past EOF", name)

	// 2. Reading the padding bits (bits added to fill the last byte) should return 0
	//    The Writer ensures unused bits in the last byte are zeroed.
	zero := reader.Read(reader.NonReadBits())
	assert.EqualValuesf(t, uint(0), zero, "%s: padding bits must be zero", name)

	// 3. Final state check: nothing left to read
	assert.EqualValuesf(t, int(0), reader.NonReadBits(), "%s: should have 0 bits left", name)
	assert.EqualValuesf(t, int(0), reader.NonReadBytes(), "%s: should have 0 bytes left", name)
}

// TestBitArrayEmpty verifies that an empty test set produces an empty array.
func TestBitArrayEmpty(t *testing.T) {
	testBitArray(t, []testWord{}, "empty")
}

// TestBitArrayB0 verifies writing a single bit of value '0'.
func TestBitArrayB0(t *testing.T) {
	testBitArray(t, []testWord{
		{1, 0b0},
	}, "b0")
}

// TestBitArrayB1 verifies writing a single bit of value '1'.
func TestBitArrayB1(t *testing.T) {
	testBitArray(t, []testWord{
		{1, 0b1},
	}, "b1")
}

// TestBitArrayPattern01 verifies an alternating 9-bit pattern (010101010).
// This tests crossing a byte boundary (8 bits + 1 bit).
func TestBitArrayPattern01(t *testing.T) {
	testBitArray(t, []testWord{
		{9, 0b010101010},
	}, "b010101010")
}

// TestBitArrayPatternLong verifies a 17-bit pattern.
// This tests crossing multiple byte boundaries (8 + 8 + 1).
func TestBitArrayPatternLong(t *testing.T) {
	testBitArray(t, []testWord{
		{17, 0b01010101010101010},
	}, "b01010101010101010")
}

// TestBitArrayRand1 runs 50 random iterations of writing multiple 1-bit words.
// This effectively tests a stream of booleans.
func TestBitArrayRand1(t *testing.T) {
	r := rand.New(rand.NewSource(0))
	for i := 0; i < 50; i++ {
		testBitArray(t, genTestWords(r, 24, 1), fmt.Sprintf("1 bit, case#%d", i))
	}
}

// TestBitArrayRand8 runs 50 random iterations where words are up to 8 bits long.
// This tests standard byte-aligned and non-aligned writes mixed together.
func TestBitArrayRand8(t *testing.T) {
	r := rand.New(rand.NewSource(0))
	for i := 0; i < 50; i++ {
		testBitArray(t, genTestWords(r, 100, 8), fmt.Sprintf("8 bits, case#%d", i))
	}
}

// TestBitArrayRand17 runs 50 random iterations where words are up to 17 bits long.
// This tests writing values larger than a single byte (uint16+) into the stream.
func TestBitArrayRand17(t *testing.T) {
	r := rand.New(rand.NewSource(0))
	for i := 0; i < 50; i++ {
		testBitArray(t, genTestWords(r, 50, 17), fmt.Sprintf("17 bits, case#%d", i))
	}
}

// --- NEW TESTS ADDED BELOW ---

// TestBitArray_View ensures that the View() method allows peeking at bits
// without advancing the read pointer.
func TestBitArray_View(t *testing.T) {
	arr := Array{make([]byte, 0, 10)}
	writer := NewWriter(&arr)
	reader := NewReader(&arr)

	// Write two 8-bit patterns: 0xAA (10101010) and 0x55 (01010101)
	val1 := uint(0xAA)
	val2 := uint(0x55)
	writer.Write(8, val1)
	writer.Write(8, val2)

	// 1. View the first 8 bits
	viewVal1 := reader.View(8)
	assert.EqualValues(t, val1, viewVal1, "View() should return correct value")
	assert.Equal(t, 16, reader.NonReadBits(), "View() should not consume bits")

	// 2. Read the first 8 bits (should match what we just Viewed)
	readVal1 := reader.Read(8)
	assert.EqualValues(t, val1, readVal1, "Read() should match View() value")
	assert.Equal(t, 8, reader.NonReadBits(), "Read() should consume bits")

	// 3. View the next 8 bits
	viewVal2 := reader.View(8)
	assert.EqualValues(t, val2, viewVal2, "View() should return next value")

	// 4. Read the next 8 bits
	readVal2 := reader.Read(8)
	assert.EqualValues(t, val2, readVal2, "Read() should match View() value")
}

// TestBitArray_Boundaries explicitly targets byte boundaries to ensure off-by-one errors
// don't occur during writes that span across bytes.
func TestBitArray_Boundaries(t *testing.T) {
	tests := []struct {
		name  string
		words []testWord
	}{
		{
			name:  "Aligned Byte",
			words: []testWord{{8, 0xFF}},
		},
		{
			name:  "Byte + 4 bits",
			words: []testWord{{8, 0xFF}, {4, 0xA}},
		},
		{
			name:  "4 bits + Byte (Crossing boundary)",
			words: []testWord{{4, 0xA}, {8, 0xFF}},
		},
		{
			name:  "Exact 16 bits",
			words: []testWord{{16, 0xFFFF}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testBitArray(t, tc.words, tc.name)
		})
	}
}

// BenchmarkArray_write measures performance of writing fixed-size bit chunks.
func BenchmarkArray_write(b *testing.B) {
	for bits := 1; bits <= 9; bits++ {
		b.Run(fmt.Sprintf("%d bits", bits), func(b *testing.B) {
			b.ResetTimer()

			// Pre-allocate to avoid measuring allocation time
			arr := Array{make([]byte, 0, bytesToFit(bits*b.N))}
			writer := NewWriter(&arr)

			for i := 0; i < b.N; i++ {
				writer.Write(bits, 0xff)
			}
		})
	}
}

// BenchmarkArray_read measures performance of reading fixed-size bit chunks.
func BenchmarkArray_read(b *testing.B) {
	for bits := 1; bits <= 9; bits++ {
		b.Run(fmt.Sprintf("%d bits", bits), func(b *testing.B) {
			b.ResetTimer()

			// Prepare data
			arr := Array{make([]byte, bytesToFit(bits*b.N))}
			reader := NewReader(&arr)

			for i := 0; i < b.N; i++ {
				_ = reader.Read(bits)
			}
		})
	}
}
