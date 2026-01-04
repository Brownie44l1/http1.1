package headers

import (
	"bytes"
	"fmt"
	"strings"
)

type Headers struct {
	headers map[string][]string
}

func NewHeaders() *Headers {
	return &Headers{
		headers: make(map[string][]string),
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

// Add appends a value to a header
func (h *Headers) Add(key, value string) {
	key = strings.ToLower(key)
	h.headers[key] = append(h.headers[key], value)
}

// Del removes a header
func (h *Headers) Del(key string) {
	delete(h.headers, strings.ToLower(key))
}

// Parse parses headers from raw bytes
func (h *Headers) Parse(data []byte) (int, bool, error) {
	read := 0
	done := false

	for {
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

		// Check for line folding (obsolete, reject it)
		if line[0] == ' ' || line[0] == '\t' {
			return read, false, fmt.Errorf("obsolete line folding not supported")
		}

		name, value, err := parseHeader(line)
		if err != nil {
			return read, done, err
		}

		// Always append - let caller decide how to handle duplicates
		h.Add(name, value)

		read += idx + 2
	}

	return read, done, nil
}

func parseHeader(line []byte) (string, string, error) {
	colonIdx := bytes.IndexByte(line, ':')
	if colonIdx == -1 {
		return "", "", fmt.Errorf("malformed header: no colon")
	}

	name := line[:colonIdx]
	value := line[colonIdx+1:]

	// Validate name has no whitespace
	if bytes.ContainsAny(name, " \t") {
		return "", "", fmt.Errorf("malformed header: whitespace in name")
	}

	// Validate name characters
	for _, b := range name {
		if !isValidHeaderChar(b) {
			return "", "", fmt.Errorf("invalid character in header name: %c", b)
		}
	}

	// Trim leading/trailing whitespace from value (allowed)
	value = bytes.TrimSpace(value)

	return strings.ToLower(string(name)), string(value), nil
}

func isValidHeaderChar(b byte) bool {
	return (b >= 'A' && b <= 'Z') ||
		(b >= 'a' && b <= 'z') ||
		(b >= '0' && b <= '9') ||
		b == '!' || b == '#' || b == '$' || b == '%' || b == '&' ||
		b == '\'' || b == '*' || b == '+' || b == '-' || b == '.' ||
		b == '^' || b == '_' || b == '`' || b == '|' || b == '~'
}