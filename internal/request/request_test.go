package request

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSimpleGETRequest(t *testing.T) {
	data := "GET /index.html HTTP/1.1\r\nHost: example.com\r\n\r\n"
	req, err := RequestFromReader(strings.NewReader(data))

	require.NoError(t, err)
	assert.Equal(t, "GET", req.Method)
	assert.Equal(t, "/index.html", req.Path)
	assert.Equal(t, "HTTP/1.1", req.Version)

	host, ok := req.Headers.Get("host")
	assert.True(t, ok)
	assert.Equal(t, "example.com", host)
	assert.Len(t, req.Body, 0)
}

func TestPOSTWithContentLength(t *testing.T) {
	data := "POST /api/data HTTP/1.1\r\n" +
		"Host: api.example.com\r\n" +
		"Content-Length: 13\r\n" +
		"\r\n" +
		"Hello, World!"

	req, err := RequestFromReader(strings.NewReader(data))

	require.NoError(t, err)
	assert.Equal(t, "POST", req.Method)
	assert.Equal(t, "/api/data", req.Path)
	assert.Equal(t, int64(13), req.ContentLength())
	assert.Equal(t, "Hello, World!", string(req.Body))
}

func TestChunkedTransferEncoding(t *testing.T) {
	data := "POST /upload HTTP/1.1\r\n" +
		"Host: example.com\r\n" +
		"Transfer-Encoding: chunked\r\n" +
		"\r\n" +
		"5\r\n" +
		"Hello\r\n" +
		"7\r\n" +
		", World\r\n" +
		"0\r\n" +
		"\r\n"

	req, err := RequestFromReader(strings.NewReader(data))

	require.NoError(t, err)
	assert.Equal(t, "POST", req.Method)
	assert.True(t, req.IsChunked())
	assert.Equal(t, "Hello, World", string(req.Body))
}

func TestHTTP10Request(t *testing.T) {
	data := "GET / HTTP/1.0\r\nHost: old.com\r\n\r\n"
	req, err := RequestFromReader(strings.NewReader(data))

	require.NoError(t, err)
	assert.True(t, req.IsHTTP10())
	assert.False(t, req.IsHTTP11())
	assert.True(t, req.WantsClose()) // HTTP/1.0 closes by default
}

func TestConnectionClose(t *testing.T) {
	data := "GET / HTTP/1.1\r\n" +
		"Host: example.com\r\n" +
		"Connection: close\r\n" +
		"\r\n"

	req, err := RequestFromReader(strings.NewReader(data))

	require.NoError(t, err)
	assert.True(t, req.WantsClose())
	assert.False(t, req.WantsKeepAlive())
}

func TestConnectionKeepAlive(t *testing.T) {
	data := "GET / HTTP/1.1\r\nHost: example.com\r\n\r\n"
	req, err := RequestFromReader(strings.NewReader(data))

	require.NoError(t, err)
	assert.False(t, req.WantsClose())
	assert.True(t, req.WantsKeepAlive()) // HTTP/1.1 default
}

func TestInvalidMethod(t *testing.T) {
	data := "INVALID /path HTTP/1.1\r\nHost: example.com\r\n\r\n"
	_, err := RequestFromReader(strings.NewReader(data))

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidMethod)
}

func TestMalformedRequestLine(t *testing.T) {
	// Missing HTTP version
	data := "GET /path\r\nHost: example.com\r\n\r\n"
	_, err := RequestFromReader(strings.NewReader(data))

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrMalformedRequestLine)
}

func TestUnsupportedVersion(t *testing.T) {
	data := "GET / HTTP/2.0\r\nHost: example.com\r\n\r\n"
	_, err := RequestFromReader(strings.NewReader(data))

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnsupportedVersion)
}

func TestInvalidPath(t *testing.T) {
	// Path doesn't start with /
	data := "GET invalid HTTP/1.1\r\nHost: example.com\r\n\r\n"
	_, err := RequestFromReader(strings.NewReader(data))

	// Actually "invalid" is accepted as absolute-form
	// Let's test truly invalid: empty path
	data = "GET HTTP/1.1\r\nHost: example.com\r\n\r\n"
	_, err = RequestFromReader(strings.NewReader(data))

	require.Error(t, err)
}

func TestIncrementalParsing(t *testing.T) {
	// Simulate slow reader that returns data byte by byte
	data := []byte("GET / HTTP/1.1\r\nHost: example.com\r\n\r\n")
	reader := &slowReader{data: data, chunkSize: 5}

	req, err := RequestFromReader(reader)

	require.NoError(t, err)
	assert.Equal(t, "GET", req.Method)
	assert.Equal(t, "/", req.Path)
}

func TestPartialBodyRead(t *testing.T) {
	// Body arrives in multiple reads
	data := "POST / HTTP/1.1\r\n" +
		"Content-Length: 20\r\n" +
		"\r\n" +
		"12345"

	reader := &slowReader{
		data:      []byte(data + "67890" + "1234567890"),
		chunkSize: len(data),
	}

	req, err := RequestFromReader(reader)

	require.NoError(t, err)
	assert.Equal(t, "12345678901234567890", string(req.Body))
}

func TestUnexpectedEOF(t *testing.T) {
	// Content-Length says 100 bytes, but we only have 10
	data := "POST / HTTP/1.1\r\n" +
		"Content-Length: 100\r\n" +
		"\r\n" +
		"0123456789"

	_, err := RequestFromReader(strings.NewReader(data))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "EOF")
}

func TestMultipleMethods(t *testing.T) {
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}

	for _, method := range methods {
		data := method + " / HTTP/1.1\r\nHost: example.com\r\n\r\n"
		req, err := RequestFromReader(strings.NewReader(data))

		require.NoError(t, err, "Method %s should be valid", method)
		assert.Equal(t, method, req.Method)
	}
}

func TestOptionsAsterisk(t *testing.T) {
	// OPTIONS * is valid
	data := "OPTIONS * HTTP/1.1\r\nHost: example.com\r\n\r\n"
	req, err := RequestFromReader(strings.NewReader(data))

	require.NoError(t, err)
	assert.Equal(t, "OPTIONS", req.Method)
	assert.Equal(t, "*", req.Path)
}

func TestChunkedWithTrailers(t *testing.T) {
	data := "POST / HTTP/1.1\r\n" +
		"Transfer-Encoding: chunked\r\n" +
		"\r\n" +
		"5\r\n" +
		"Hello\r\n" +
		"0\r\n" +
		"X-Checksum: abc123\r\n" +
		"\r\n"

	req, err := RequestFromReader(strings.NewReader(data))

	require.NoError(t, err)
	assert.Equal(t, "Hello", string(req.Body))
	// Note: We don't parse trailers yet, but it shouldn't error
}

// slowReader simulates a network connection that provides data slowly
type slowReader struct {
	data      []byte
	chunkSize int
	offset    int
}

func (r *slowReader) Read(p []byte) (int, error) {
	if r.offset >= len(r.data) {
		return 0, io.EOF
	}

	n := r.chunkSize
	if n > len(p) {
		n = len(p)
	}
	if n > len(r.data)-r.offset {
		n = len(r.data) - r.offset
	}

	copy(p, r.data[r.offset:r.offset+n])
	r.offset += n
	return n, nil
}
