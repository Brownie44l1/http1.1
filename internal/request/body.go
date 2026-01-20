package request

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
)

type chunkParser struct {
	state       chunkState
	chunkSize   int
	chunkRead   int
	totalBodySize int64  // Track total
}

type chunkState int

const (
	chunkStateSize chunkState = iota
	chunkStateData
	chunkStateDataCRLF
	chunkStateTrailer
	chunkStateDone
)

var (
	ErrInvalidChunkSize     = errors.New("invalid chunk size")
	ErrChunkTooLarge        = errors.New("chunk size too large")
	ErrChunkSizeLineTooLong = errors.New("chunk size line too long")
	ErrBodyTooLarge         = errors.New("chunked body exceeds maximum size")
	ErrInvalidChunkFormat   = errors.New("invalid chunk format")
	crlf                    = []byte("\r\n")
)

const (
	maxChunkSize       = 10 * 1024 * 1024    // 10MB per chunk
	maxTotalBodySize   = 50 * 1024 * 1024    // 50MB total
	maxChunkSizeLine   = 1024                // 1KB for size line
)

// parseChunkedIncremental parses chunked data incrementally
// Parser state must be preserved across calls!
func parseChunkedIncremental(data []byte, body *[]byte, parser *chunkParser, maxBodySize int64) (int, bool, error) {
	consumed := 0
	
	for consumed < len(data) {
		switch parser.state {
		case chunkStateSize:
			n, err := parser.parseChunkSize(data[consumed:])
			if err != nil {
				return consumed, false, err
			}
			if n == 0 {
				// Need more data
				return consumed, false, nil
			}
			consumed += n
			
			if parser.chunkSize == 0 {
				// Last chunk (0\r\n)
				parser.state = chunkStateTrailer
			} else {
				parser.state = chunkStateData
				parser.chunkRead = 0
			}
			
		case chunkStateData:
			remaining := parser.chunkSize - parser.chunkRead
			available := len(data[consumed:])
			toRead := min(remaining, available)
			
			// Check total body size limit
			if parser.totalBodySize + int64(toRead) > maxBodySize {
				return consumed, false, ErrBodyTooLarge
			}
			
			*body = append(*body, data[consumed:consumed+toRead]...)
			consumed += toRead
			parser.chunkRead += toRead
			parser.totalBodySize += int64(toRead)
			
			if parser.chunkRead == parser.chunkSize {
				parser.state = chunkStateDataCRLF
			} else {
				// Need more data for chunk
				return consumed, false, nil
			}
			
		case chunkStateDataCRLF:
			if len(data[consumed:]) < 2 {
				// Need more data
				return consumed, false, nil
			}
			
			if data[consumed] != '\r' || data[consumed+1] != '\n' {
				return consumed, false, ErrInvalidChunkFormat
			}
			
			consumed += 2
			parser.state = chunkStateSize  // Next chunk
			
		case chunkStateTrailer:
			if len(data[consumed:]) < 2 {
				return consumed, false, nil
			}
			
			if data[consumed] == '\r' && data[consumed+1] == '\n' {
				consumed += 2
				parser.state = chunkStateDone
				return consumed, true, nil
			}
			
			idx := bytes.Index(data[consumed:], []byte("\r\n\r\n"))
			if idx == -1 {
				// Check if we've buffered too much without finding end
				if len(data[consumed:]) > maxChunkSizeLine {
					return consumed, false, errors.New("trailer headers too large")
				}
				// Need more data
				return consumed, false, nil
			}
			
			trailers := data[consumed:consumed+idx]
			if bytes.ContainsAny(trailers, "\x00") {
				return consumed, false, errors.New("null byte in trailer headers")
			}
			
			consumed += idx + 4  // Skip trailers + \r\n\r\n
			parser.state = chunkStateDone
			return consumed, true, nil
			
		case chunkStateDone:
			return consumed, true, nil
		}
	}
	
	return consumed, false, nil
}

// parseChunkSize parses the chunk size line: SIZE[;extensions]\r\n
func (p *chunkParser) parseChunkSize(data []byte) (int, error) {
	// Limit search to prevent DoS
	searchLimit := min(len(data), maxChunkSizeLine)
	
	idx := bytes.Index(data[:searchLimit], crlf)
	if idx == -1 {
		if len(data) >= maxChunkSizeLine {
			return 0, ErrChunkSizeLineTooLong
		}
		// Need more data
		return 0, nil
	}
	
	sizeLine := data[:idx]
	
	// Chunk size might have extensions: SIZE;name=value
	// We ignore extensions but validate format
	parts := bytes.SplitN(sizeLine, []byte(";"), 2)
	sizeHex := string(bytes.TrimSpace(parts[0]))
	
	if len(parts) > 1 {
		ext := parts[1]
		if bytes.ContainsAny(ext, "\r\n\x00") {
			return 0, errors.New("invalid characters in chunk extension")
		}
	}
	
	size, err := strconv.ParseInt(sizeHex, 16, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrInvalidChunkSize, err)
	}
	
	if size < 0 {
		return 0, ErrInvalidChunkSize
	}
	
	if size > maxChunkSize {
		return 0, ErrChunkTooLarge
	}
	
	p.chunkSize = int(size)
	return idx + 2, nil 
}
