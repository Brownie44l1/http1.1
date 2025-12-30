package http

import (
	"bufio"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
)

type Request struct {
	Method  string
	Path    string
	Body    string
	Version string
	Headers map[string][]string
	Query   map[string][]string
}

func ParseRequest(reader *bufio.Reader) (*Request, error) {
	r := &Request{}

	// Parse request line
	if err := r.parseRequestLine(reader); err != nil {
		return nil, err
	}

	// Parse headers
	if err := r.parseHeaders(reader); err != nil {
		return nil, err
	}

	// Parse body
	if err := r.parseBody(reader); err != nil {
		return nil, err
	}

	return r, nil
}

func (r *Request) parseRequestLine(reader *bufio.Reader) error {
	line, err := reader.ReadString('\n')
	if err != nil {
		return err
	}

	parts := strings.Split(strings.TrimSpace(line), " ")
	if len(parts) != 3 {
		return fmt.Errorf("invalid request line")
	}

	method := strings.ToUpper(parts[0])
	target := parts[1]
	version := parts[2]

	// Split path and query
	path := target
	query := ""

	if idx := strings.Index(target, "?"); idx != -1 {
		path = target[:idx]
		query = target[idx+1:]
	}

	// Validate method
	switch method {
	case "GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS":
		// Valid methods
	default:
		return fmt.Errorf("unsupported method: %s", method)
	}

	// Validate path
	if !strings.HasPrefix(path, "/") {
		return fmt.Errorf("invalid path: must start with /")
	}

	// Validate version
	if version != "HTTP/1.1" && version != "HTTP/1.0" {
		return fmt.Errorf("unsupported HTTP version: %s", version)
	}

	r.Method = method
	r.Path = path
	r.Version = version

	// Parse query string
	if query != "" {
		if err := r.parseQueryString(query); err != nil {
			return err
		}
	}

	return nil
}

func (r *Request) parseQueryString(query string) error {
	if r.Query == nil {
		r.Query = make(map[string][]string)
	}

	pairs := strings.Split(query, "&")
	for _, pair := range pairs {
		if pair == "" {
			continue
		}

		kv := strings.SplitN(pair, "=", 2)
		
		// Handle key without value (e.g., ?debug)
		var key, value string
		key, err := url.QueryUnescape(kv[0])
		if err != nil {
			// Skip malformed keys
			continue
		}

		if len(kv) == 2 {
			value, err = url.QueryUnescape(kv[1])
			if err != nil {
				// Skip malformed values
				continue
			}
		}

		r.Query[key] = append(r.Query[key], value)
	}

	return nil
}

func (r *Request) parseHeaders(reader *bufio.Reader) error {
	r.Headers = make(map[string][]string)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return err
		}

		line = strings.TrimRight(line, "\r\n")
		
		// Empty line marks end of headers
		if line == "" {
			break
		}

		kv := strings.SplitN(line, ":", 2)
		if len(kv) != 2 {
			return fmt.Errorf("malformed header: %s", line)
		}

		key := strings.TrimSpace(kv[0])
		value := strings.TrimSpace(kv[1])

		// Normalize header keys to lowercase for case-insensitive lookup
		key = strings.ToLower(key)

		r.Headers[key] = append(r.Headers[key], value)
	}
	
	return nil
}

func (r *Request) parseBody(reader *bufio.Reader) error {
	// Check for chunked transfer encoding first
	if encoding, ok := r.Headers["transfer-encoding"]; ok {
		// Check if any of the values contains "chunked"
		for _, enc := range encoding {
			if strings.ToLower(strings.TrimSpace(enc)) == "chunked" {
				bodyBytes, err := parseChunkedBody(reader)
				if err != nil {
					return err
				}
				r.Body = string(bodyBytes)
				return nil
			}
		}
	}

	// Check for Content-Length
	cl, ok := r.Headers["content-length"]
	if !ok || len(cl) == 0 {
		// No body
		return nil
	}

	length, err := strconv.Atoi(cl[0])
	if err != nil {
		return fmt.Errorf("invalid content-length: %s", cl[0])
	}

	if length < 0 {
		return fmt.Errorf("negative content-length: %d", length)
	}

	if length == 0 {
		// Empty body
		return nil
	}

	// Read exactly 'length' bytes
	buf := make([]byte, length)
	_, err = io.ReadFull(reader, buf)
	if err != nil {
		return fmt.Errorf("failed to read body: %w", err)
	}

	r.Body = string(buf)
	return nil
}

func parseChunkedBody(reader *bufio.Reader) ([]byte, error) {
	var body []byte

	for {
		// Read chunk size line
		sizeLine, err := reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("failed to read chunk size: %w", err)
		}

		// Parse hex size (chunk size can have extensions after semicolon)
		sizeStr := strings.TrimSpace(sizeLine)
		if idx := strings.Index(sizeStr, ";"); idx != -1 {
			sizeStr = sizeStr[:idx]
		}

		size, err := strconv.ParseInt(sizeStr, 16, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid chunk size '%s': %w", sizeStr, err)
		}

		if size < 0 {
			return nil, fmt.Errorf("negative chunk size: %d", size)
		}

		// Size 0 means last chunk
		if size == 0 {
			// Read trailing headers (if any) and final CRLF
			for {
				line, err := reader.ReadString('\n')
				if err != nil {
					return nil, fmt.Errorf("failed to read trailer: %w", err)
				}
				if strings.TrimSpace(line) == "" {
					break
				}
			}
			break
		}

		// Read chunk data
		chunk := make([]byte, size)
		_, err = io.ReadFull(reader, chunk)
		if err != nil {
			return nil, fmt.Errorf("failed to read chunk data: %w", err)
		}

		body = append(body, chunk...)

		// Read trailing CRLF after chunk data
		_, err = reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("failed to read chunk trailer: %w", err)
		}
	}

	return body, nil
}

// Helper methods for easier access

func (r *Request) GetHeader(key string) string {
	key = strings.ToLower(key)
	if values, ok := r.Headers[key]; ok && len(values) > 0 {
		return values[0]
	}
	return ""
}

func (r *Request) GetHeaders(key string) []string {
	key = strings.ToLower(key)
	return r.Headers[key]
}

func (r *Request) GetQuery(key string) string {
	if values, ok := r.Query[key]; ok && len(values) > 0 {
		return values[0]
	}
	return ""
}

func (r *Request) GetQueryValues(key string) []string {
	return r.Query[key]
}