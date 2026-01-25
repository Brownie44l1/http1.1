package request

import (
	"io"
	"strconv"

	"github.com/Brownie44l1/http/internal/headers"
)

// Request represents a parsed HTTP request
type Request struct {
	Method  string
	Path    string
	Version string
	Headers *headers.Headers
	Body    []byte

	parser *parser
}

// NewRequest creates a new Request with initialized fields
func NewRequest() *Request {
	return &Request{
		Headers: headers.NewHeaders(),
		Body:    make([]byte, 0),
	}
}

func RequestFromReaderWithConfig(reader io.Reader, maxHeaderBytes int, maxBodySize int64) (*Request, error) {
	req := NewRequest()
	parser := newParser(maxBodySize)
	
	err := parser.parseFromReader(reader, req, maxHeaderBytes)
	if err != nil {
		return nil, err
	}
	
	return req, nil
}

// IsHTTP10 returns true if this is an HTTP/1.0 request
func (r *Request) IsHTTP10() bool {
	return r.Version == "HTTP/1.0"
}

// IsHTTP11 returns true if this is an HTTP/1.1 request
func (r *Request) IsHTTP11() bool {
	return r.Version == "HTTP/1.1"
}

// WantsClose returns true if the client wants to close the connection
// HTTP/1.0: true unless "Connection: keep-alive"
// HTTP/1.1: true only if "Connection: close"
func (r *Request) WantsClose() bool {
	conn, ok := r.Headers.Get("connection")
	if !ok {
		// HTTP/1.0 closes by default
		return r.IsHTTP10()
	}

	// Explicit "Connection: close"
	return conn == "close"
}

// WantsKeepAlive returns true if the client wants persistent connection
func (r *Request) WantsKeepAlive() bool {
	return !r.WantsClose()
}

// ContentLength returns the Content-Length header value, or -1 if not present
func (r *Request) ContentLength() int64 {
	cl, ok := r.Headers.Get("content-length")
	if !ok {
		return -1
	}

	length, err := parseInt64(cl)
	if err != nil {
		return -1
	}

	return length
}

// IsChunked returns true if Transfer-Encoding: chunked
func (r *Request) IsChunked() bool {
	te, ok := r.Headers.Get("transfer-encoding")
	if !ok {
		return false
	}
	return te == "chunked"
}

// parseInt64 parses a string to int64
func parseInt64(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

func RequestFromReader(reader io.Reader) (*Request, error) {
	return RequestFromReaderWithConfig(reader, 1<<20, 10<<20)
}