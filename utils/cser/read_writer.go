/*
This file implements the Primitive Encoding Logic for the CSER protocol.
While binary.go handles the container format, this file handles the actual values:
Integers (U16, U32, U64): Uses a clever "Split Encoding". The number of bytes needed to store the integer is written to the Bit Stream,
while the actual bytes are written to the Byte Stream. This compresses small numbers efficiently.
Booleans: Written as single bits in the Bit Stream.
Arrays/Slices: Written as [Length][Data].
Canonical Enforcement: It strictly panics if data is not encoded in the most compact way possible (e.g., storing the number 5 using 8 bytes instead of 1 is illegal).
*/
package cser

import (
	"errors"
	"math/big"

	"github.com/rony4d/go-opera-asset/utils/bits"
	"github.com/rony4d/go-opera-asset/utils/fast"
)

// Standard errors for encoding validation.
var (
	ErrNonCanonicalEncoding = errors.New("non canonical encoding: data not packed minimally or unused bits non-zero")
	ErrMalformedEncoding    = errors.New("malformed encoding: structure invalid or truncated")
	ErrTooLargeAlloc        = errors.New("too large allocation: decoded size exceeds limits")
)

// MaxAlloc limits the size of byte slices to prevent OOM (Out of Memory) attacks during decoding.
const MaxAlloc = 100 * 1024

// Writer orchestrates writing to the two separate streams.
type Writer struct {
	BitsW  *bits.Writer // For booleans and length-prefixes
	BytesW *fast.Writer // For raw data bytes
}

// Reader orchestrates reading from the two separate streams.
type Reader struct {
	BitsR  *bits.Reader
	BytesR *fast.Reader
}

// NewWriter creates a ready-to-use CSER writer.
func NewWriter() *Writer {
	// Pre-allocate some space to avoid immediate re-allocations
	bbits := &bits.Array{Bytes: make([]byte, 0, 32)}
	bbytes := make([]byte, 0, 200)
	return &Writer{
		BitsW:  bits.NewWriter(bbits),
		BytesW: fast.NewWriter(bbytes),
	}
}

// ----------------------------------------------------------------------------
// Low-Level Encoding Primitives
// ----------------------------------------------------------------------------

// writeUint64Compact implements standard "Varint" encoding (Base-128).
// Used for the file-suffix length field, not for general integers.
// Logic: 7 bits of data per byte. MSB (0x80) is the "more bytes coming" flag.
func writeUint64Compact(bytesW *fast.Writer, v uint64) {
	for i := 0; ; i++ {
		chunk := v & 0b01111111 // Take lower 7 bits
		v = v >> 7              // Shift down
		if v == 0 {
			// This is the last byte. MSB is 0.
			chunk |= 0b10000000 // Wait, the code actually sets MSB for STOP?
			// Let's check the original code...
			// Original: if v==0 { chunk |= 0x80 } ...
			// Original Read: stop = (chunk & 0x80) != 0
			// This is REVERSE Varint logic: 1 means STOP, 0 means CONTINUE.
			chunk |= 0b10000000
		}
		bytesW.WriteByte(byte(chunk))
		if v == 0 {
			break
		}
	}
	return
}

// readUint64Compact decodes the reverse-logic varint.
func readUint64Compact(bytesR *fast.Reader) uint64 {
	v := uint64(0)
	stop := false
	for i := 0; !stop; i++ {
		chunk := uint64(bytesR.ReadByte())
		stop = (chunk & 0b10000000) != 0 // 1 means STOP
		word := chunk & 0b01111111       // Data bits
		v |= word << (i * 7)

		// Canonical Check: The last byte (highest significant) cannot be zero data
		// unless the number is actually zero.
		// Example: Encoding '5' as [5, 0(stop)] is illegal. Must be [5(stop)].
		if i > 0 && stop && word == 0 {
			panic(ErrNonCanonicalEncoding)
		}
	}
	return v
}

// writeUint64BitCompact writes an integer using Little Endian bytes.
// It writes exactly enough bytes to represent 'v', but at least 'minSize' bytes.
// Returns the number of bytes written.
func writeUint64BitCompact(bytesW *fast.Writer, v uint64, minSize int) (size int) {
	// Write until value is exhausted AND we met the minimum size requirement
	for size < minSize || v != 0 {
		bytesW.WriteByte(byte(v)) // Write lowest 8 bits
		size++
		v = v >> 8 // Shift down
	}
	return
}

// readUint64BitCompact reads 'size' bytes and reassembles the integer (Little Endian).
func readUint64BitCompact(bytesR *fast.Reader, size int) uint64 {
	var (
		v    uint64
		last byte
	)
	buf := bytesR.Read(size)
	for i, b := range buf {
		v |= uint64(b) << uint(8*i)
		last = b
	}

	// Canonical Check: The most significant byte cannot be zero.
	// If it is zero, it means we used more bytes than necessary (e.g. padding), which is forbidden.
	if size > 1 && last == 0 {
		panic(ErrNonCanonicalEncoding)
	}

	return v
}

// ----------------------------------------------------------------------------
// CSER Split-Stream Primitives (The "Side-Channel Length" pattern)
// ----------------------------------------------------------------------------

// readU64_bits is a generic helper for integers.
// 1. Reads the *byte length* from the Bit Stream.
// 2. Reads the *actual bytes* from the Byte Stream.
func (r *Reader) readU64_bits(minSize int, bitsForSize int) uint64 {
	// Read N bits to determine how many extra bytes to read beyond minSize.
	size := r.BitsR.Read(bitsForSize)
	size += uint(minSize)
	return readUint64BitCompact(r.BytesR, int(size))
}

// writeU64_bits is the inverse.
// 1. Writes the bytes to Byte Stream.
// 2. Calculates length used.
// 3. Writes the length-offset to the Bit Stream.
func (w *Writer) writeU64_bits(minSize int, bitsForSize int, v uint64) {
	size := writeUint64BitCompact(w.BytesW, v, minSize)
	// Store (ActualLength - MinLength) in the bit stream
	w.BitsW.Write(bitsForSize, uint(size-minSize))
}

// U8 writes a single byte directly (no length prefix needed).
func (w *Writer) U8(v uint8) {
	w.BytesW.WriteByte(v)
}
func (r *Reader) U8() uint8 {
	return r.BytesR.ReadByte()
}

// U16 writes a uint16.
// Uses 1 bit for length in Bit Stream.
// Length can be 1 (min) or 2.
func (w *Writer) U16(v uint16) {
	w.writeU64_bits(1, 1, uint64(v))
}
func (r *Reader) U16() uint16 {
	v64 := r.readU64_bits(1, 1)
	return uint16(v64)
}

// U32 writes a uint32.
// Uses 2 bits for length in Bit Stream (offsets 0-3).
// Length can be 1..4 bytes.
func (w *Writer) U32(v uint32) {
	w.writeU64_bits(1, 2, uint64(v))
}
func (r *Reader) U32() uint32 {
	v64 := r.readU64_bits(1, 2)
	return uint32(v64)
}

// U64 writes a uint64.
// Uses 3 bits for length in Bit Stream (offsets 0-7).
// Length can be 1..8 bytes.
func (w *Writer) U64(v uint64) {
	w.writeU64_bits(1, 3, v)
}
func (r *Reader) U64() uint64 {
	return r.readU64_bits(1, 3)
}

// VarUint is an alias for U64 encoding logic (used for map sizes etc).
func (r *Reader) VarUint() uint64 {
	return r.readU64_bits(1, 3)
}
func (w *Writer) VarUint(v uint64) {
	w.writeU64_bits(1, 3, v)
}

// I64 writes a signed int64.
// Format: [Sign Bit in BitStream] + [Absolute Value as U64]
func (w *Writer) I64(v int64) {
	w.Bool(v < 0) // Sign bit
	if v < 0 {
		w.U64(uint64(-v))
	} else {
		w.U64(uint64(v))
	}
}
func (r *Reader) I64() int64 {
	neg := r.Bool()
	abs := r.U64()

	// Canonical Check: Negative Zero is illegal.
	if neg && abs == 0 {
		panic(ErrNonCanonicalEncoding)
	}
	if neg {
		return -int64(abs)
	}
	return int64(abs)
}

// U56 is used for slice lengths (limiting to 56 bits / 7 bytes).
// Uses 3 bits for length (0-7), minSize=0.
func (w *Writer) U56(v uint64) {
	const max = 1<<(8*7) - 1
	if v > max {
		panic("Value too big")
	}
	w.writeU64_bits(0, 3, v)
}
func (r *Reader) U56() uint64 {
	return r.readU64_bits(0, 3)
}

// Bool writes a single bit to the Bit Stream.
func (w *Writer) Bool(v bool) {
	u8 := uint(0)
	if v {
		u8 = 1
	}
	w.BitsW.Write(1, u8)
}
func (r *Reader) Bool() bool {
	u8 := r.BitsR.Read(1)
	return u8 != 0
}

// FixedBytes reads/writes a fixed amount of raw bytes to the Byte Stream.
func (w *Writer) FixedBytes(v []byte) {
	w.BytesW.Write(v)
}
func (r *Reader) FixedBytes(v []byte) {
	buf := r.BytesR.Read(len(v))
	copy(v, buf)
}

// SliceBytes handles variable length byte arrays.
// Format: [Length as U56] + [Raw Bytes]
func (w *Writer) SliceBytes(v []byte) {
	w.U56(uint64(len(v))) // Prefix with length
	w.FixedBytes(v)
}
func (r *Reader) SliceBytes(maxLen int) []byte {
	size := r.U56()
	if size > uint64(maxLen) {
		panic(ErrTooLargeAlloc)
	}
	buf := make([]byte, size)
	r.FixedBytes(buf)
	return buf
}

// PaddedBytes returns a slice with length of the slice is at least n bytes.
func PaddedBytes(b []byte, n int) []byte {
	if len(b) >= n {
		return b
	}
	padding := make([]byte, n-len(b))
	return append(padding, b...)
}

// BigInt handles arbitrary precision integers.
// Format: Serialized as a byte slice of the magnitude. Sign is not handled here?
// Correction: The code uses `SliceBytes(v.Bytes())`.
// `v.Bytes()` in Go returns absolute value as Big-Endian bytes.
// This implementation loses the sign for BigInts unless handled externally!
// (Usually used for Prices/Amounts which are always positive).
func (w *Writer) BigInt(v *big.Int) {
	bigBytes := []byte{}
	if v.Sign() != 0 {
		bigBytes = v.Bytes()
	}
	w.SliceBytes(bigBytes)
}

func (r *Reader) BigInt() *big.Int {
	buf := r.SliceBytes(512) // Limit max big int size
	if len(buf) == 0 {
		return new(big.Int)
	}
	return new(big.Int).SetBytes(buf)
}
