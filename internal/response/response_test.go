package response

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"http1.1/internal/headers"
)

func TestChunkedBodyRawBytes(t *testing.T) {
	// Test: Check exact byte sequence for chunked encoding
	buf := &bytes.Buffer{}
	w := NewWriter(buf)
	
	err := w.WriteStatusLine(StatusOk)
	require.NoError(t, err)
	
	h := headers.Headers{
		Header: map[string]string{"Transfer-Encoding": "chunked"},
	}
	err = w.WriteHeaders(h)
	require.NoError(t, err)
	
	// Write a simple 4-byte chunk
	_, err = w.WriteChunkedBody([]byte("TEST"))
	require.NoError(t, err)
	
	_, err = w.WriteChunkedBodyDone()
	require.NoError(t, err)
	
	got := buf.String()
	t.Logf("Raw bytes: %q", got)
	
	// Should contain: "4\r\nTEST\r\n0\r\n"
	assert.Contains(t, got, "4\r\n")
	assert.Contains(t, got, "TEST\r\n")
	assert.Contains(t, got, "0\r\n")
}

func TestWriterStatusLine(t *testing.T) {
	// Test: 200 OK
	buf := &bytes.Buffer{}
	w := NewWriter(buf)
	err := w.WriteStatusLine(StatusOk)
	require.NoError(t, err)
	assert.Equal(t, "HTTP/1.1 200 OK\r\n", buf.String())

	// Test: 400 Bad Request
	buf = &bytes.Buffer{}
	w = NewWriter(buf)
	err = w.WriteStatusLine(StatusBadRequest)
	require.NoError(t, err)
	assert.Equal(t, "HTTP/1.1 400 Bad Request\r\n", buf.String())

	// Test: 500 Internal Server Error
	buf = &bytes.Buffer{}
	w = NewWriter(buf)
	err = w.WriteStatusLine(StatusInternalServerError)
	require.NoError(t, err)
	assert.Equal(t, "HTTP/1.1 500 Internal Server Error\r\n", buf.String())
}

func TestWriterHeaders(t *testing.T) {
	// Test: Write headers after status line
	buf := &bytes.Buffer{}
	w := NewWriter(buf)

	err := w.WriteStatusLine(StatusOk)
	require.NoError(t, err)

	h := headers.Headers{
		Header: map[string]string{
			"Content-Type":   "text/html",
			"Content-Length": "100",
			"Connection":     "close",
		},
	}

	err = w.WriteHeaders(h)
	require.NoError(t, err)

	got := buf.String()
	assert.Contains(t, got, "Content-Type: text/html")
	assert.Contains(t, got, "Content-Length: 100")
	assert.Contains(t, got, "Connection: close")
	assert.Contains(t, got, "\r\n\r\n") // Blank line after headers
}

func TestWriterBody(t *testing.T) {
	// Test: Write body after status and headers
	buf := &bytes.Buffer{}
	w := NewWriter(buf)

	err := w.WriteStatusLine(StatusOk)
	require.NoError(t, err)

	h := headers.Headers{
		Header: map[string]string{
			"Content-Type": "text/plain",
		},
	}
	err = w.WriteHeaders(h)
	require.NoError(t, err)

	body := []byte("Hello, World!")
	n, err := w.WriteBody(body)
	require.NoError(t, err)
	assert.Equal(t, len(body), n)

	got := buf.String()
	assert.Contains(t, got, "Hello, World!")
}

func TestWriterChunkedBody(t *testing.T) {
	// Test: Write chunked body with multiple chunks
	buf := &bytes.Buffer{}
	w := NewWriter(buf)

	err := w.WriteStatusLine(StatusOk)
	require.NoError(t, err)

	h := headers.Headers{
		Header: map[string]string{
			"Transfer-Encoding": "chunked",
		},
	}
	err = w.WriteHeaders(h)
	require.NoError(t, err)

	// Write first chunk: "Hello, " (7 bytes)
	chunk1 := []byte("Hello, ")
	n, err := w.WriteChunkedBody(chunk1)
	require.NoError(t, err)
	assert.Equal(t, len(chunk1), n)

	// Write second chunk: "World!" (6 bytes)
	chunk2 := []byte("World!")
	n, err = w.WriteChunkedBody(chunk2)
	require.NoError(t, err)
	assert.Equal(t, len(chunk2), n)

	// Write final chunk
	_, err = w.WriteChunkedBodyDone()
	require.NoError(t, err)

	got := buf.String()
	assert.Contains(t, got, "7\r\n")        // Chunk size for "Hello, "
	assert.Contains(t, got, "Hello, \r\n")  // First chunk data
	assert.Contains(t, got, "6\r\n")        // Chunk size for "World!"
	assert.Contains(t, got, "World!\r\n")   // Second chunk data
	assert.Contains(t, got, "0\r\n")        // Final zero chunk

	t.Logf("Complete chunked output:\n%s", got)
}

func TestWriterChunkedBodyWithHexSizes(t *testing.T) {
	// Test: Verify hex encoding of chunk sizes
	buf := &bytes.Buffer{}
	w := NewWriter(buf)

	err := w.WriteStatusLine(StatusOk)
	require.NoError(t, err)

	h := headers.Headers{
		Header: map[string]string{
			"Transfer-Encoding": "chunked",
		},
	}
	err = w.WriteHeaders(h)
	require.NoError(t, err)

	// Write a 255 byte chunk (ff in hex)
	chunk := make([]byte, 255)
	for i := range chunk {
		chunk[i] = 'A'
	}
	_, err = w.WriteChunkedBody(chunk)
	require.NoError(t, err)

	_, err = w.WriteChunkedBodyDone()
	require.NoError(t, err)

	got := buf.String()
	assert.Contains(t, got, "ff\r\n") // 255 in hex is 'ff'
	t.Logf("Hex size in output: %s", got)
}

func TestWriterTrailers(t *testing.T) {
	// Test: Write trailers after chunked body
	buf := &bytes.Buffer{}
	w := NewWriter(buf)

	err := w.WriteStatusLine(StatusOk)
	require.NoError(t, err)

	h := headers.Headers{
		Header: map[string]string{
			"Transfer-Encoding": "chunked",
			"Trailer":           "X-Content-SHA256, X-Content-Length",
		},
	}
	err = w.WriteHeaders(h)
	require.NoError(t, err)

	// Write chunked body
	data := []byte("Test data for hashing")
	_, err = w.WriteChunkedBody(data)
	require.NoError(t, err)

	_, err = w.WriteChunkedBodyDone()
	require.NoError(t, err)

	// Calculate hash
	hash := sha256.Sum256(data)
	hashStr := fmt.Sprintf("%x", hash)

	// Write trailers
	trailers := headers.Headers{
		Header: map[string]string{
			"X-Content-SHA256": hashStr,
			"X-Content-Length": fmt.Sprintf("%d", len(data)),
		},
	}

	err = w.WriteTrailers(trailers)
	require.NoError(t, err)

	got := buf.String()
	assert.Contains(t, got, "X-Content-SHA256: "+hashStr)
	assert.Contains(t, got, "X-Content-Length: 21")

	t.Logf("Complete output with trailers:\n%s", got)
}

func TestWriterStateValidation(t *testing.T) {
	// Test: Cannot write headers before status
	buf := &bytes.Buffer{}
	w := NewWriter(buf)
	h := headers.Headers{Header: map[string]string{"Content-Type": "text/plain"}}
	err := w.WriteHeaders(h)
	assert.Error(t, err)

	// Test: Cannot write body before headers
	buf = &bytes.Buffer{}
	w = NewWriter(buf)
	err = w.WriteStatusLine(StatusOk)
	require.NoError(t, err)
	_, err = w.WriteBody([]byte("test"))
	assert.Error(t, err)

	// Test: Cannot write trailers before body
	buf = &bytes.Buffer{}
	w = NewWriter(buf)
	err = w.WriteStatusLine(StatusOk)
	require.NoError(t, err)
	h = headers.Headers{Header: map[string]string{"Content-Type": "text/plain"}}
	err = w.WriteHeaders(h)
	require.NoError(t, err)
	trailers := headers.Headers{Header: map[string]string{"X-Test": "value"}}
	err = w.WriteTrailers(trailers)
	assert.Error(t, err)
}

func TestCompleteChunkedResponseWithTrailers(t *testing.T) {
	// Test: Complete end-to-end chunked response with trailers
	buf := &bytes.Buffer{}
	w := NewWriter(buf)

	err := w.WriteStatusLine(StatusOk)
	require.NoError(t, err)

	h := headers.Headers{
		Header: map[string]string{
			"Transfer-Encoding": "chunked",
			"Content-Type":      "application/json",
			"Trailer":           "X-Content-SHA256, X-Content-Length",
		},
	}
	err = w.WriteHeaders(h)
	require.NoError(t, err)

	// Write chunks
	fullBody := []byte{}
	chunks := []string{
		`{"id": 0}`,
		`{"id": 1}`,
		`{"id": 2}`,
	}

	for _, chunk := range chunks {
		data := []byte(chunk + "\n")
		fullBody = append(fullBody, data...)
		_, err = w.WriteChunkedBody(data)
		require.NoError(t, err)
	}

	_, err = w.WriteChunkedBodyDone()
	require.NoError(t, err)

	// Calculate hash and write trailers
	hash := sha256.Sum256(fullBody)
	trailers := headers.Headers{
		Header: map[string]string{
			"X-Content-SHA256": fmt.Sprintf("%x", hash),
			"X-Content-Length": fmt.Sprintf("%d", len(fullBody)),
		},
	}
	err = w.WriteTrailers(trailers)
	require.NoError(t, err)

	got := buf.String()

	// Verify structure
	assert.Contains(t, got, "HTTP/1.1 200 OK\r\n")
	assert.Contains(t, got, "Transfer-Encoding: chunked")
	assert.Contains(t, got, "0\r\n") // Final zero chunk
	assert.Contains(t, got, "X-Content-SHA256:")
	assert.Contains(t, got, "X-Content-Length: 30")

	t.Logf("Complete response:\n%s", got)
}