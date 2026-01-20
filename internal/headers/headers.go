package headers

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var (
	ErrDuplicateContentLength    = errors.New("duplicate Content-Length header")
	ErrConflictingContentLength  = errors.New("conflicting Content-Length values")
	ErrDuplicateHost             = errors.New("duplicate Host header")
	ErrDuplicateTransferEncoding = errors.New("duplicate Transfer-Encoding header")
	ErrBothChunkedAndLength      = errors.New("both Transfer-Encoding and Content-Length present")
	ErrObsoleteLineFolding       = errors.New("obsolete line folding not supported")
	ErrMalformedHeader           = errors.New("malformed header")
	ErrInvalidHeaderChar         = errors.New("invalid character in header name")
	ErrTooManyHeaders            = errors.New("too many headers")
	ErrHeaderTooLarge            = errors.New("header too large")
)

const (
	MaxHeaderLines = 100
	MaxHeaderSize  = 1 << 20 // 1MB
)

type Headers struct {
	headers map[string][]string

	// Track special headers for validation
	tracking *headerTracking
}

type headerTracking struct {
	seenHost             bool
	seenContentLength    bool
	contentLengthValue   int64
	seenTransferEncoding bool
	isChunked            bool
	headerCount          int
	totalSize            int
}

func NewHeaders() *Headers {
	return &Headers{
		headers:  make(map[string][]string),
		tracking: &headerTracking{},
	}
}

// Get returns the first value for a header
func (h *Headers) Get(key string) (string, bool) {
	values := h.headers[strings.ToLower(key)]
	if len(values) == 0 {
		return "", false
	}
	return values[0], true
}

// GetAll returns all values for a header
func (h *Headers) GetAll(key string) []string {
	return h.headers[strings.ToLower(key)]
}

// GetAllHeaders returns the internal map (for iteration)
func (h *Headers) GetAllHeaders() map[string][]string {
	return h.headers
}

// Set replaces all values for a header
func (h *Headers) Set(key, value string) {
	h.headers[strings.ToLower(key)] = []string{value}
}

// Add appends a value to a header (use carefully - validation bypassed)
func (h *Headers) Add(key, value string) {
	key = strings.ToLower(key)
	h.headers[key] = append(h.headers[key], value)
}

// Del removes a header
func (h *Headers) Del(key string) {
	delete(h.headers, strings.ToLower(key))
}

// IsChunked returns true if Transfer-Encoding: chunked
func (h *Headers) IsChunked() bool {
	return h.tracking.isChunked
}

// ContentLength returns the Content-Length value (-1 if not present)
func (h *Headers) ContentLength() int64 {
	if !h.tracking.seenContentLength {
		return -1
	}
	return h.tracking.contentLengthValue
}

// Parse parses headers from raw bytes with security validation
func (h *Headers) Parse(data []byte) (int, bool, error) {
	read := 0
	done := false

	for {
		// Check header count limit
		if h.tracking.headerCount >= MaxHeaderLines {
			return read, false, ErrTooManyHeaders
		}

		// Check total header size
		if h.tracking.totalSize >= MaxHeaderSize {
			return read, false, ErrHeaderTooLarge
		}

		idx := bytes.Index(data[read:], []byte("\r\n"))
		if idx == -1 {
			// Need more data
			break
		}

		if idx == 0 {
			// Empty line = end of headers
			done = true
			read += 2
			break
		}

		line := data[read : read+idx]
		h.tracking.totalSize += len(line)

		// Check for line folding (obsolete, reject it)
		if line[0] == ' ' || line[0] == '\t' {
			return read, false, ErrObsoleteLineFolding
		}

		name, value, err := parseHeader(line)
		if err != nil {
			return read, done, err
		}

		// Validate and store with security checks
		if err := h.addWithValidation(name, value); err != nil {
			return read, false, err
		}

		h.tracking.headerCount++
		read += idx + 2
	}

	// Final validation after all headers parsed
	if done {
		if err := h.validateFinal(); err != nil {
			return read, done, err
		}
	}

	return read, done, nil
}

// addWithValidation adds header with security validation
func (h *Headers) addWithValidation(name, value string) error {
	nameLower := strings.ToLower(name)

	switch nameLower {
	case "host":
		if h.tracking.seenHost {
			return ErrDuplicateHost
		}
		h.tracking.seenHost = true
		h.headers[nameLower] = []string{value}

	case "content-length":
		cl, err := strconv.ParseInt(value, 10, 64)
		if err != nil || cl < 0 {
			return fmt.Errorf("invalid Content-Length: %w", err)
		}

		if h.tracking.seenContentLength {
			if h.tracking.contentLengthValue != cl {
				return ErrConflictingContentLength
			}
			return nil
		}

		h.tracking.seenContentLength = true
		h.tracking.contentLengthValue = cl
		h.headers[nameLower] = []string{value}

	case "transfer-encoding":
		if h.tracking.seenTransferEncoding {
			return ErrDuplicateTransferEncoding
		}
		h.tracking.seenTransferEncoding = true

		if strings.ToLower(strings.TrimSpace(value)) == "chunked" {
			h.tracking.isChunked = true
		}
		h.headers[nameLower] = []string{value}

	default:
		h.headers[nameLower] = append(h.headers[nameLower], value)
	}

	return nil
}

func (h *Headers) validateFinal() error {
	if h.tracking.isChunked && h.tracking.seenContentLength {
		return ErrBothChunkedAndLength
	}

	return nil
}

func parseHeader(line []byte) (string, string, error) {
	before, after, ok := bytes.Cut(line, []byte{':'})
	if !ok {
		return "", "", ErrMalformedHeader
	}

	name := before
	value := after

	if bytes.ContainsAny(name, " \t") {
		return "", "", ErrMalformedHeader
	}

	if len(name) == 0 {
		return "", "", ErrMalformedHeader
	}

	for _, b := range name {
		if !isValidHeaderChar(b) {
			return "", "", fmt.Errorf("%w: %c", ErrInvalidHeaderChar, b)
		}
	}

	if bytes.ContainsAny(value, "\x00\r\n") {
		return "", "", fmt.Errorf("invalid characters in header value")
	}
	value = bytes.TrimSpace(value)
	return string(name), string(value), nil
}

func isValidHeaderChar(b byte) bool {
	return (b >= 'A' && b <= 'Z') ||
		(b >= 'a' && b <= 'z') ||
		(b >= '0' && b <= '9') ||
		b == '!' || b == '#' || b == '$' || b == '%' || b == '&' ||
		b == '\'' || b == '*' || b == '+' || b == '-' || b == '.' ||
		b == '^' || b == '_' || b == '`' || b == '|' || b == '~'
}
