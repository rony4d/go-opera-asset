package cser

import (
	"github.com/rony4d/go-opera-asset/utils/bits"
	"github.com/rony4d/go-opera-asset/utils/fast"
)

// binary.go provides the core functionality for CSER (Canonical Serialization).
// It implements the low-level bitstream operations and the higher-level serialization/deserialization functions.
//
// Use Case:
// - Serializing and deserializing data that is not aligned to standard 8-bit byte boundaries.
// - Writing custom small integers (e.g., a 3-bit number).
// - This is a core component of the Custom Serialization (CSER) format

// MarshalBinaryAdapter acts as a bridge between the high-level CSER serialization logic
// and the raw binary output required by Go's encoding interfaces.
//
// It sets up the two temporary buffers (Bits and Bytes), executes the user's
// serialization function, and then packs the results into a single byte slice.
func MarshalBinaryAdapter(marshalCser func(*Writer) error) ([]byte, error) {
	// 1. Create a CSER Writer which contains two internal buffers:
	//    - w.BitsW (for unaligned small bits)
	//    - w.BytesW (for aligned bytes)
	w := NewWriter()

	// 2. Run the provided serialization logic (callback).
	err := marshalCser(w)
	if err != nil {
		return nil, err
	}

	// 3. Merge the two buffers into one final byte slice.
	return binaryFromCSER(w.BitsW.Array, w.BytesW.Bytes())
}

// binaryFromCSER packs the "Body" (main bytes) and "Bits" (flags/small ints) into one raw slice.
//
// Layout on wire:
// [ Body Bytes ... ] + [ BitStream Bytes ... ] + [ REVERSED Varint(Len(BitStreamBytes)) ]
func binaryFromCSER(bbits *bits.Array, bbytes []byte) (raw []byte, err error) {
	// Start with the main body bytes
	bodyBytes := fast.NewWriter(bbytes)

	// Append the bit-stream bytes immediately after
	bodyBytes.Write(bbits.Bytes)

	// Calculate the size of the bit-stream portion.
	// We need to append this size to the very end so the reader knows where to split the data.
	sizeWriter := fast.NewWriter(make([]byte, 0, 4))

	// writeUint64Compact is a variable-length integer encoder (Varint).
	// (Note: This function is defined in another file in the cser package).
	writeUint64Compact(sizeWriter, uint64(len(bbits.Bytes)))

	// CRITICAL TRICK: The size varint is written in REVERSE order at the end of the buffer.
	// This allows the reader to scan backwards from the end of the file to decode the length.
	bodyBytes.Write(reversed(sizeWriter.Bytes()))

	return bodyBytes.Bytes(), nil
}

// binaryToCSER unpacks the raw binary blob back into separate "Bits" and "Bytes" streams.
// It works backwards from the end of the slice.
func binaryToCSER(raw []byte) (bbits *bits.Array, bbytes []byte, err error) {
	// 1. Read the Suffix to find out how big the BitStream is.
	//    We grab the last 9 bytes (max size of a 64-bit varint) and reverse them back to normal order.
	bitsSizeBuf := reversed(tail(raw, 9))

	// 2. Decode the Varint to get the actual length of the bit stream.
	bitsSizeReader := fast.NewReader(bitsSizeBuf)
	bitsSize := readUint64Compact(bitsSizeReader)

	// 3. Remove the Suffix from the raw data.
	//    raw now contains only [Body Bytes] + [BitStream Bytes]
	raw = raw[:len(raw)-bitsSizeReader.Position()]

	// Sanity Check: Ensure the declared bit stream size isn't larger than the remaining data.
	if uint64(len(raw)) < bitsSize {
		err = ErrMalformedEncoding
		return
	}

	// 4. Split the remaining raw data.
	//    The last `bitsSize` bytes go to the Bits Array.
	//    The preceding bytes go to the Body Bytes.
	bbits = &bits.Array{Bytes: raw[uint64(len(raw))-bitsSize:]}
	bbytes = raw[:uint64(len(raw))-bitsSize]
	return
}

// UnmarshalBinaryAdapter adapts the raw binary input to the CSER Reader interface.
// It splits the raw data and then runs the user's unmarshal function.
func UnmarshalBinaryAdapter(raw []byte, unmarshalCser func(reader *Reader) error) (err error) {
	// Safety catch for panics (common in fast serialization libraries that skip bounds checks)
	defer func() {
		if r := recover(); r != nil {
			err = ErrMalformedEncoding
		}
	}()

	// 1. Split the streams
	bbits, bbytes, err := binaryToCSER(raw)
	if err != nil {
		return err
	}

	// 2. Create the CSER Reader with the split streams
	bodyReader := &Reader{
		BitsR:  bits.NewReader(bbits),
		BytesR: fast.NewReader(bbytes),
	}

	// 3. Run the user's deserialization logic
	err = unmarshalCser(bodyReader)
	if err != nil {
		return err
	}

	// 4. Canonical Encoding Checks (Strict Mode)
	// Ensure that ALL data was consumed. If there are leftover bytes/bits, the encoding is invalid.

	// Check if there are unused bytes in the bitstream
	if bodyReader.BitsR.NonReadBytes() > 1 {
		return ErrNonCanonicalEncoding
	}

	// Check if there are unused bits in the final byte of the bitstream.
	// The protocol requires unused trailing bits to be zero.
	tail := bodyReader.BitsR.Read(bodyReader.BitsR.NonReadBits())
	if tail != 0 {
		return ErrNonCanonicalEncoding
	}

	// Check if there are unused bytes in the body stream
	if !bodyReader.BytesR.Empty() {
		return ErrNonCanonicalEncoding
	}

	return nil
}

// tail returns the last `cap` bytes of slice `b`.
// If `b` is smaller than `cap`, it returns the whole slice.
func tail(b []byte, cap int) []byte {
	if len(b) > cap {
		return b[len(b)-cap:]
	}
	return b
}

// reversed creates a NEW slice containing the bytes of `b` in reverse order.
// Used for writing/reading the suffix length field backwards.
func reversed(b []byte) []byte {
	reversed := make([]byte, len(b))
	for i, v := range b {
		reversed[len(b)-1-i] = v
	}
	return reversed
}
