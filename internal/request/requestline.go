package request

import (
	"bytes"
	"errors"
)

var (
	ErrMalformedRequestLine = errors.New("malformed request line")
	ErrInvalidMethod        = errors.New("invalid HTTP method")
	ErrInvalidPath          = errors.New("invalid request path")
	ErrUnsupportedVersion   = errors.New("unsupported HTTP version")
)

// parseRequestLine parses: METHOD PATH VERSION\r\n
// Returns: method, path, version, bytesConsumed, error
func parseRequestLine(data []byte) (string, string, string, int, error) {
	// Find end of line
	idx := bytes.Index(data, crlf)
	if idx == -1 {
		// Need more data
		return "", "", "", 0, nil
	}
	
	line := data[:idx]
	consumed := idx + 2 // +2 for \r\n
	
	// Split into parts: METHOD PATH VERSION
	parts := bytes.SplitN(line, []byte(" "), 3)
	if len(parts) != 3 {
		return "", "", "", 0, ErrMalformedRequestLine
	}
	
	method := string(parts[0])
	path := string(parts[1])
	version := string(parts[2])
	
	// Validate method
	if !isValidMethod(method) {
		return "", "", "", 0, ErrInvalidMethod
	}
	
	// Validate path
	if !isValidPath(path) {
		return "", "", "", 0, ErrInvalidPath
	}
	
	// Validate version
	if !isValidVersion(version) {
		return "", "", "", 0, ErrUnsupportedVersion
	}
	
	return method, path, version, consumed, nil
}

// isValidMethod checks if the HTTP method is supported
func isValidMethod(method string) bool {
	switch method {
	case "GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS":
		return true
	default:
		return false
	}
}

// isValidPath checks if the request path is valid
func isValidPath(path string) bool {
	// Must start with / (origin-form) or be * (for OPTIONS)
	if len(path) == 0 {
		return false
	}
	
	if path[0] == '/' {
		return true
	}
	
	// Allow "*" for OPTIONS * HTTP/1.1
	if path == "*" {
		return true
	}
	
	// Could be absolute-form: http://example.com/path
	// For now, we'll accept it but not fully validate
	return true
}

// isValidVersion checks if HTTP version is supported
func isValidVersion(version string) bool {
	return version == "HTTP/1.0" || version == "HTTP/1.1"
}