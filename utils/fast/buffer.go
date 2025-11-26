package fast

// buffer.go provides a lightweight, non-thread-safe wrapper around byte slices.
//
// Purpose:
// - Standard Go `bytes.Buffer` or `bufio` can be overkill for simple, linear serialization tasks.
// - This package provides a "fast" path that simply appends to a slice (Writer) or increments an integer index (Reader).
// - It performs NO bounds checking errors (it will panic if you read past the end), which is faster but requires the caller to be careful (safe for internal, trusted serialization code).

type Reader struct {
	// buf is the underlying data source.
	buf []byte
	// offset tracks the current reading position (cursor).
	offset int
}

type Writer struct {
	// buf is the accumulating byte slice.
	buf []byte
}

// NewReader creates a Reader to consume the provided byte slice.
func NewReader(bb []byte) *Reader {
	return &Reader{
		buf:    bb,
		offset: 0,
	}
}

// NewWriter creates a Writer that appends to the provided initial slice.
// Often called with `make([]byte, 0, capacity)` to pre-allocate memory.
func NewWriter(bb []byte) *Writer {
	return &Writer{
		buf: bb,
	}
}

// WriteByte appends a single byte to the buffer.
// This is efficient as it uses Go's built-in append optimization.
func (b *Writer) WriteByte(v byte) {
	b.buf = append(b.buf, v)
}

// Write appends a slice of bytes (bulk write) to the buffer.
func (b *Writer) Write(v []byte) {
	b.buf = append(b.buf, v...)
}

// Read consumes and returns the next 'n' bytes from the buffer.
//
// WARNING: This function does NOT check if 'n' bytes are available.
// If (offset + n) > len(buf), this will panic with a runtime slice bounds out of range error.
// This design choice prioritizes speed over safety.
//
// Note: It returns a slice that *shares memory* with the original buffer.
// Modifying the returned slice will modify the original buffer.
func (b *Reader) Read(n int) []byte {
	// Slice slicing: buf[start : end]
	res := b.buf[b.offset : b.offset+n]
	b.offset += n
	return res
}

// ReadByte consumes and returns a single byte.
// WARNING: Panics if buffer is empty.
func (b *Reader) ReadByte() byte {
	res := b.buf[b.offset]
	b.offset++
	return res
}

// Position returns the current cursor index of the Reader.
// Useful for determining how many bytes have been consumed.
func (b *Reader) Position() int {
	return b.offset
}

// Bytes returns the entire underlying buffer of the Reader.
func (b *Reader) Bytes() []byte {
	return b.buf
}

// Bytes returns the accumulated content of the Writer.
func (b *Writer) Bytes() []byte {
	return b.buf
}

// Empty checks if the Reader has reached the end of the buffer.
// Returns true if there are no more bytes to read.
func (b *Reader) Empty() bool {
	return len(b.buf) == b.offset
}
