package server

import "sync"

// ✅ Issue #13: Buffer Pooling for Performance

// BufferPool manages reusable byte buffers
type BufferPool struct {
	small  sync.Pool // 4KB buffers
	medium sync.Pool // 32KB buffers
	large  sync.Pool // 128KB buffers
}

// Global buffer pool instance
var globalBufferPool = &BufferPool{
	small: sync.Pool{
		New: func() interface{} {
			buf := make([]byte, 4096) // 4KB
			return &buf
		},
	},
	medium: sync.Pool{
		New: func() interface{} {
			buf := make([]byte, 32768) // 32KB
			return &buf
		},
	},
	large: sync.Pool{
		New: func() interface{} {
			buf := make([]byte, 131072) // 128KB
			return &buf
		},
	},
}

// GetBuffer returns a buffer of at least the requested size
func GetBuffer(size int) []byte {
	if size <= 4096 {
		buf := globalBufferPool.small.Get().(*[]byte)
		return (*buf)[:size]
	} else if size <= 32768 {
		buf := globalBufferPool.medium.Get().(*[]byte)
		return (*buf)[:size]
	} else {
		buf := globalBufferPool.large.Get().(*[]byte)
		if len(*buf) < size {
			// Need bigger buffer
			newBuf := make([]byte, size)
			return newBuf
		}
		return (*buf)[:size]
	}
}

// PutBuffer returns a buffer to the pool
func PutBuffer(buf []byte) {
	capacity := cap(buf)
	
	if capacity == 4096 {
		fullBuf := buf[:4096]
		globalBufferPool.small.Put(&fullBuf)
	} else if capacity == 32768 {
		fullBuf := buf[:32768]
		globalBufferPool.medium.Put(&fullBuf)
	} else if capacity == 131072 {
		fullBuf := buf[:131072]
		globalBufferPool.large.Put(&fullBuf)
	}
	// Else: buffer is non-standard size, let GC handle it
}

// BufferedReader can be used in parser.go to reuse buffers
type BufferedReader struct {
	buf []byte
}

// NewBufferedReader creates a new buffered reader with pooled buffer
func NewBufferedReader() *BufferedReader {
	return &BufferedReader{
		buf: GetBuffer(4096),
	}
}

// Close returns the buffer to the pool
func (br *BufferedReader) Close() {
	if br.buf != nil {
		PutBuffer(br.buf)
		br.buf = nil
	}
}

// Buffer returns the internal buffer
func (br *BufferedReader) Buffer() []byte {
	return br.buf
}