package request

import (
	"errors"
	"fmt"
	"io"
)

// Size limits (Issue #3 - DoS protection)
const (
	maxRequestLineSize = 8192         // 8KB for request line
	maxHeaderSize      = 1 << 20      // 1MB total headers
	maxBodySize        = 100 << 20    // 100MB body (can be configured)
	maxHeaderLines     = 1000         // Max number of header lines
	maxURILength       = 8192         // Max URI length
)

var (
	ErrRequestLineTooLarge = errors.New("request line too large")
	ErrHeaderTooLarge      = errors.New("headers too large")
	// ErrBodyTooLarge is removed - already declared in body.go
	ErrTooManyHeaders      = errors.New("too many header lines")
	ErrURITooLong          = errors.New("URI too long")
)

// parserState represents the current state of the request parser
type parserState int

const (
	stateRequestLine parserState = iota
	stateHeaders
	stateBody
	stateDone
)

// parser handles incremental parsing of HTTP requests
type parser struct {
	state       parserState
	buffer      []byte       // Accumulates data between reads
	chunkParser *chunkParser // For chunked encoding
	
	// Size tracking (Issue #3)
	totalBytesRead int64
	headerLines    int
	maxBodySize    int64
}

func newParser(maxBodySize int64) *parser {
	if maxBodySize <= 0 {
		maxBodySize = maxBodySize // This line looks like a bug - should probably be a default value
	}
	
	return &parser{
		state:       stateRequestLine,
		buffer:      make([]byte, 0, 4096), // Start with 4KB
		chunkParser: &chunkParser{},
		maxBodySize: maxBodySize,
	}
}

// parseFromReader reads from io.Reader and parses the request
func (p *parser) parseFromReader(reader io.Reader, req *Request, maxHeaderBytes int) error {
	if maxHeaderBytes <= 0 {
		maxHeaderBytes = maxHeaderSize
	}
	
	readBuf := make([]byte, 4096)

	for p.state != stateDone {
		// Try to parse what we have in buffer first
		if len(p.buffer) > 0 {
			consumed, err := p.parse(p.buffer, req, maxHeaderBytes)
			if err != nil {
				return err
			}

			// Remove consumed bytes from buffer
			if consumed > 0 {
				p.buffer = p.buffer[consumed:]
				continue // Try parsing again before reading more
			}
		}

		// ✅ Issue #3: Check size limits BEFORE reading more
		if p.state != stateBody && len(p.buffer) >= maxHeaderBytes {
			return ErrHeaderTooLarge
		}

		// Need more data - read from connection
		n, err := reader.Read(readBuf)
		if n > 0 {
			// ✅ Issue #3: Prevent buffer from growing unbounded
			if p.state != stateBody {
				// We're still reading headers
				if len(p.buffer)+n > maxHeaderBytes {
					return ErrHeaderTooLarge
				}
			}
			
			p.buffer = append(p.buffer, readBuf[:n]...)
			p.totalBytesRead += int64(n)
		}

		if err != nil {
			if err == io.EOF {
				// EOF is only okay if we're done parsing
				if p.state == stateDone {
					return nil
				}
				return errors.New("unexpected EOF")
			}
			return fmt.Errorf("read error: %w", err)
		}
	}

	return nil
}

// parse processes buffered data and advances the state machine
// Returns number of bytes consumed
func (p *parser) parse(data []byte, req *Request, maxHeaderBytes int) (int, error) {
	switch p.state {
	case stateRequestLine:
		return p.parseRequestLine(data, req)

	case stateHeaders:
		return p.parseHeaders(data, req, maxHeaderBytes)

	case stateBody:
		return p.parseBody(data, req)

	case stateDone:
		return 0, nil

	default:
		return 0, fmt.Errorf("invalid parser state: %d", p.state)
	}
}

func (p *parser) parseRequestLine(data []byte, req *Request) (int, error) {
	// ✅ Issue #3: Check request line size
	if len(data) > maxRequestLineSize {
		return 0, ErrRequestLineTooLarge
	}
	
	method, path, version, consumed, err := parseRequestLine(data)
	if err != nil {
		return 0, err
	}

	if consumed == 0 {
		// Need more data
		return 0, nil
	}

	// ✅ Issue #3: Validate URI length
	if len(path) > maxURILength {
		return 0, ErrURITooLong
	}

	req.Method = method
	req.Path = path
	req.Version = version

	p.state = stateHeaders
	return consumed, nil
}

// parseHeaders parses HTTP headers until empty line
func (p *parser) parseHeaders(data []byte, req *Request, maxHeaderBytes int) (int, error) {
	consumed, done, err := req.Headers.Parse(data)
	if err != nil {
		return 0, err
	}

	// ✅ Issue #3: Count header lines
	p.headerLines++
	if p.headerLines > maxHeaderLines {
		return 0, ErrTooManyHeaders
	}

	if !done {
		// Headers not complete yet, need more data
		return consumed, nil
	}

	// Headers complete - determine what comes next
	if req.IsChunked() {
		// Chunked body
		p.state = stateBody
		return consumed, nil
	}

	cl := req.ContentLength()
	if cl > 0 {
		// ✅ Issue #3: Validate body size against limit
		if cl > p.maxBodySize {
			return 0, ErrBodyTooLarge
		}
		
		// Fixed-length body
		p.state = stateBody
		return consumed, nil
	}

	// No body (GET request, or Content-Length: 0)
	p.state = stateDone
	return consumed, nil
}

// parseBody reads the request body based on Content-Length or chunked encoding
func (p *parser) parseBody(data []byte, req *Request) (int, error) {
	if req.IsChunked() {
		return p.parseChunkedBody(data, req)
	}
	return p.parseFixedBody(data, req)
}

// parseFixedBody reads body with known Content-Length
func (p *parser) parseFixedBody(data []byte, req *Request) (int, error) {
	cl := req.ContentLength()
	if cl < 0 {
		return 0, errors.New("missing Content-Length for body")
	}

	// ✅ Issue #3: Double-check size limit
	if cl > p.maxBodySize {
		return 0, ErrBodyTooLarge
	}

	remaining := int(cl) - len(req.Body)
	if remaining <= 0 {
		p.state = stateDone
		return 0, nil
	}

	// Read up to what we need
	toRead := min(remaining, len(data))
	
	// ✅ Issue #3: Check total body size as we accumulate
	if int64(len(req.Body)+toRead) > p.maxBodySize {
		return 0, ErrBodyTooLarge
	}
	
	req.Body = append(req.Body, data[:toRead]...)

	// Check if body is complete
	if len(req.Body) == int(cl) {
		p.state = stateDone
	}

	return toRead, nil
}

// parseChunkedBody reads Transfer-Encoding: chunked body
func (p *parser) parseChunkedBody(data []byte, req *Request) (int, error) {
	// ✅ Issue #3: Use maxBodySize for chunked encoding too
	consumed, done, err := parseChunkedIncremental(data, &req.Body, p.chunkParser, p.maxBodySize)
	if err != nil {
		return 0, err
	}

	if done {
		p.state = stateDone
	}

	return consumed, nil
}