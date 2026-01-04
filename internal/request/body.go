package request

import (
	"bytes"
	"errors"
	"strconv"
)

// Chunked transfer encoding format:
// 
// <chunk-size in hex>\r\n
// <chunk-data>\r\n
// <chunk-size in hex>\r\n
// <chunk-data>\r\n
// 0\r\n
// \r\n

type chunkParser struct {
	state      chunkState
	chunkSize  int    // Current chunk size being read
	chunkRead  int    // Bytes read of current chunk
}

type chunkState int

const (
	chunkStateSize chunkState = iota  // Reading chunk size line
	chunkStateData                     // Reading chunk data
	chunkStateDataCRLF                 // Reading CRLF after chunk data
	chunkStateTrailer                  // Reading trailer headers (after last chunk)
	chunkStateDone                     // All chunks read
)

var (
	ErrInvalidChunkSize = errors.New("invalid chunk size")
	ErrChunkTooLarge    = errors.New("chunk size too large")
)

const maxChunkSize = 10 * 1024 * 1024 // 10MB max per chunk

// parseChunked parses chunked transfer encoding
// Returns: bytesConsumed, done, error
func parseChunked(data []byte, body *[]byte) (int, bool, error) {
	parser := &chunkParser{state: chunkStateSize}
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
				// Last chunk
				parser.state = chunkStateTrailer
			} else {
				parser.state = chunkStateData
				parser.chunkRead = 0
			}
			
		case chunkStateData:
			remaining := parser.chunkSize - parser.chunkRead
			available := len(data[consumed:])
			toRead := min(remaining, available)
			
			*body = append(*body, data[consumed:consumed+toRead]...)
			consumed += toRead
			parser.chunkRead += toRead
			
			if parser.chunkRead == parser.chunkSize {
				parser.state = chunkStateDataCRLF
			} else {
				// Need more data
				return consumed, false, nil
			}
			
		case chunkStateDataCRLF:
			if len(data[consumed:]) < 2 {
				// Need more data
				return consumed, false, nil
			}
			
			if data[consumed] != '\r' || data[consumed+1] != '\n' {
				return consumed, false, errors.New("missing CRLF after chunk data")
			}
			
			consumed += 2
			parser.state = chunkStateSize
			
		case chunkStateTrailer:
			// After the last chunk (0\r\n), there might be trailer headers
			// For now, we just look for the final \r\n
			if len(data[consumed:]) < 2 {
				return consumed, false, nil
			}
			
			if data[consumed] == '\r' && data[consumed+1] == '\n' {
				consumed += 2
				parser.state = chunkStateDone
				return consumed, true, nil
			}
			
			// There are trailer headers - skip them for now
			// TODO: Parse trailer headers properly
			idx := bytes.Index(data[consumed:], []byte("\r\n\r\n"))
			if idx == -1 {
				return consumed, false, nil
			}
			consumed += idx + 4
			parser.state = chunkStateDone
			return consumed, true, nil
			
		case chunkStateDone:
			return consumed, true, nil
		}
	}
	
	return consumed, false, nil
}

// parseChunkSize parses the chunk size line: SIZE\r\n
func (p *chunkParser) parseChunkSize(data []byte) (int, error) {
	idx := bytes.Index(data, crlf)
	if idx == -1 {
		// Need more data
		return 0, nil
	}
	
	sizeLine := data[:idx]
	
	// Chunk size might have extensions: SIZE;name=value
	// We ignore extensions for now
	parts := bytes.SplitN(sizeLine, []byte(";"), 2)
	sizeHex := string(bytes.TrimSpace(parts[0]))
	
	// Parse hex size
	size, err := strconv.ParseInt(sizeHex, 16, 64)
	if err != nil {
		return 0, ErrInvalidChunkSize
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

// parseInt64 safely parses an int64 from a string
func parseInt64(s string) (int64, error) {
	val, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return -1, err
	}
	if val < 0 {
		return -1, errors.New("negative value not allowed")
	}
	return val, nil
}