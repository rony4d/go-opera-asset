package bits

// This package implements a low-level "Bit Stream" Reader and Writer.
// It allows you to write data that is not aligned to standard 8-bit byte boundaries.
//
// Use Case:
// - Compressing boolean flags (write 1 bit instead of 8).
// - Writing custom small integers (e.g., a 3-bit number).
// - This is a core component of the Custom Serialization (CSER) format

type (
	// Array is a container for the underlying byte slice that holds the bitstream.
	Array struct {
		Bytes []byte
	}

	// Writer allows writing variable numbers of bits into an Array.
	// It maintains the state of where the "cursor" currently is within the current byte.
	Writer struct {
		*Array
		bitOffset int // 0-7: The index of the next bit to write in the current byte (Bytes[last])
	}

	// Reader allows reading variable numbers of bits from an Array.
	// It tracks position both by byte index and bit offset within that byte.
	Reader struct {
		*Array
		byteOffset int // Index of the current byte in Bytes
		bitOffset  int // 0-7: Index of the next bit to read in Bytes[byteOffset]
	}
)

// NewWriter creates a new bitstream writer pointing to the given array.
func NewWriter(arr *Array) *Writer {
	return &Writer{
		Array: arr,
	}
}

// NewReader creates a new bitstream reader pointing to the given array.
func NewReader(arr *Array) *Reader {
	return &Reader{
		Array: arr,
	}
}

// byteBitsFree calculates how many bits are left in the current byte (8 - offset).
// Example: If bitOffset is 3 (we wrote bits 0,1,2), then 5 bits are free.
func (a *Writer) byteBitsFree() int {
	return 8 - a.bitOffset
}

// writeIntoLastByte merges the bits of 'v' into the current active byte using OR logic.
// It shifts 'v' left by the offset to place the bits in the correct free slots.
func (a *Writer) writeIntoLastByte(v uint) {
	// OR equals: Keep existing bits, set new ones to 1 if 'v' has 1s.
	a.Bytes[len(a.Bytes)-1] |= byte(v << a.bitOffset)
}

// zeroTopByteBits is a helper that clears the upper 'bits' of a value 'v'.
// This is used to isolate the specific chunk of bits we want to write when splitting across bytes.
func zeroTopByteBits(v uint, bits int) uint {
	// Create a mask. Example: if bits=3 (clear top 3), we want to keep bottom 5.
	// This implementation actually clears based on the shift logic used in Write.
	// Note: The naming is slightly confusing relative to standard "clear high bits".
	// It effectively masks 'v' to fit into the remaining space.
	mask := uint(0xff) >> bits
	return v & mask
}

// Write appends the lowest 'bits' count of integer 'v' into the bitstream.
// Example: Write(3, 5) -> writes binary '101' (3 bits).
func (a *Writer) Write(bits int, v uint) {
	// If we are at the start of a new byte (or array is empty), allocate a fresh zero byte.
	if a.bitOffset == 0 {
		a.Bytes = append(a.Bytes, byte(0))
	}

	free := a.byteBitsFree()

	// Case 1: The data fits entirely within the current byte.
	if bits <= free {
		toWrite := bits
		// Merge bits into the current byte
		a.writeIntoLastByte(v)

		// Update the cursor
		if toWrite == free {
			// We filled the byte exactly. Reset cursor to 0.
			// (Next write will trigger the append(0) block above)
			a.bitOffset = 0
		} else {
			// We still have space in this byte.
			a.bitOffset += toWrite
		}
	} else {
		// Case 2: The data spills over into the next byte.
		// Strategy: Write what fits now, then recursively write the rest.

		toWrite := free      // Fill the remaining space
		clear := a.bitOffset // Helper variable for masking

		// Write the lower 'toWrite' bits of 'v' into the current byte.
		a.writeIntoLastByte(zeroTopByteBits(v, clear))

		// Current byte is now full.
		a.bitOffset = 0

		// Recursively write the REMAINING bits (v shifted right by what we just wrote).
		a.Write(bits-toWrite, v>>toWrite)
	}
}

// byteBitsFree returns how many unread bits remain in the current byte being read.
func (a *Reader) byteBitsFree() int {
	return 8 - a.bitOffset
}

// Read extracts 'bits' count from the stream and returns them as an integer.
// It advances the cursor.
func (a *Reader) Read(bits int) (v uint) {
	// Branch optimization check
	if bits == 0 {
		return 0
	}

	free := a.byteBitsFree()

	// Case 1: All requested bits are inside the current byte.
	if bits <= free {
		toRead := bits
		// Calculate how many bits on the "right" (higher index) we need to ignore.
		// Example: Byte is [11100011]. bitOffset=0. bits=3.
		// We want [111]. We need to clear the top 5 bits.
		// Note: The implementation logic here assumes Little Endian bit ordering logic usually.
		// (Reading from LSB to MSB relative to how Write put them in).

		// clear = 8 - (start + len)
		clear := 8 - (a.bitOffset + toRead)

		// Mask out the high bits we don't want, then shift down to 0.
		v = zeroTopByteBits(uint(a.Bytes[a.byteOffset]), clear) >> a.bitOffset

		// Update cursor
		if toRead == free {
			a.bitOffset = 0
			a.byteOffset++
		} else {
			a.bitOffset += toRead
		}
	} else {
		// Case 2: The requested bits span across two bytes.
		// Strategy: Read what's left in current byte, then recursively read the rest.

		toRead := free

		// Read the remaining bits in this byte (shifted down).
		v = uint(a.Bytes[a.byteOffset]) >> a.bitOffset

		// Move to next byte
		a.bitOffset = 0
		a.byteOffset++

		// Recursively read the rest from the next byte(s).
		rest := a.Read(bits - toRead)

		// Combine the results.
		// The 'rest' (higher bits of the result) must be shifted LEFT to make room for the lower bits we just read.
		v |= rest << toRead
	}
	return
}

// View allows "peeking" at the next 'bits' without advancing the cursor state.
// It clones the reader, performs a read, and returns the result.
func (a *Reader) View(bits int) (v uint) {
	cp := *a   // Shallow copy of the struct (pointer to Array is shared, but offsets are copied)
	cpp := &cp // Use the copy
	return cpp.Read(bits)
}

// NonReadBytes returns the number of full unconsumed bytes remaining in the buffer.
func (a *Reader) NonReadBytes() int {
	return len(a.Bytes) - a.byteOffset
}

// NonReadBits calculates the total number of individual unread bits remaining.
func (a *Reader) NonReadBits() int {
	// Total bytes * 8, minus the bits we've already skipped in the current byte.
	return a.NonReadBytes()*8 - a.bitOffset
}
