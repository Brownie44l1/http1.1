package headers

import (
	"bytes"
	"fmt"
	"strings"
)

type Headers struct {
	Header map[string]string
}

var rn = []byte("\r\n")

func NewHeaders() Headers {
	return Headers{
		Header: make(map[string]string),
	}
}

func parseHeader(fieldLine []byte) (string, string, error) {
	parts := bytes.SplitN(fieldLine, []byte(":"), 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("malformed field line")
	}

	name := parts[0]
	value := bytes.TrimSpace(parts[1])

	if bytes.HasSuffix(name, []byte(" ")) {
		return "", "", fmt.Errorf("malformed field name")
	}

	for _, b := range name {
		if !isValidHeaderChar(b) {
			return "", "", fmt.Errorf("invalid character in field name")
		}
	}

	nameStr := strings.ToLower(string(name))

	return nameStr, string(value), nil
}

func (h *Headers) Parse(data []byte) (int, bool, error) {
	read := 0
	done := false

	for {
		idx := bytes.Index(data[read:], rn)
		if idx == -1 {
			break
		}

		if idx == 0 {
			done = true
			read += len(rn)
			break
		}

		name, value, err := parseHeader(data[read : read+idx])
		if err != nil {
			return read, done, err
		}

		read += idx + len(rn)
		
		lname := strings.ToLower(name)
		if existing, exists := h.Header[lname]; exists {
			h.Header[lname] = existing + ", " + value
		} else {
			h.Header[lname] = value
		}
	}

	return read, done, nil
}

func (h *Headers) Get(key string) (string, bool) {
	v, ok := h.Header[strings.ToLower(key)]
	return v, ok
}

func (h *Headers) Set(key, value string) {
	if h.Header == nil {
		h.Header = make(map[string]string)
	}
	h.Header[key] = value
}

func isValidHeaderChar(b byte) bool {
	return (b >= 'A' && b <= 'Z') ||
		(b >= 'a' && b <= 'z') ||
		(b >= '0' && b <= '9') ||
		b == '!' || b == '#' || b == '$' || b == '%' || b == '&' ||
		b == '\'' || b == '*' || b == '+' || b == '-' || b == '.' ||
		b == '^' || b == '_' || b == '`' || b == '|' || b == '~'
}
