package headers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHeaderParse(t *testing.T) {
	// Test: Valid single header
	h := NewHeaders()
	data := []byte("Host: localhost:42069\r\n")
	n, done, err := h.Parse(data)
	require.NoError(t, err)
	val, ok := h.Get("host")
	assert.True(t, ok)
	assert.Equal(t, "localhost:42069", val)
	assert.Equal(t, 23, n)
	assert.False(t, done)

	// Test: Valid single header with extra whitespace
	h = NewHeaders()
	data = []byte("Host:   localhost:42069   \r\n")
	n, done, err = h.Parse(data)
	require.NoError(t, err)
	val, ok = h.Get("host")
	assert.True(t, ok)
	assert.Equal(t, "localhost:42069", val)
	assert.False(t, done)

	// Test: Duplicate headers (should store multiple values)
	h = NewHeaders()
	data = []byte("Set-Cookie: a=1\r\nSet-Cookie: b=2\r\n")
	n, done, err = h.Parse(data)
	require.NoError(t, err)
	values := h.GetAll("set-cookie")
	assert.Equal(t, []string{"a=1", "b=2"}, values)
	assert.False(t, done)

	// Test: Get returns first value for duplicate headers
	val, ok = h.Get("set-cookie")
	assert.True(t, ok)
	assert.Equal(t, "a=1", val)

	// Test: Empty line signals end of headers
	h = NewHeaders()
	data = []byte("\r\n")
	n, done, err = h.Parse(data)
	require.NoError(t, err)
	assert.Equal(t, 2, n)
	assert.True(t, done)

	// Test: Headers followed by empty line
	h = NewHeaders()
	data = []byte("Host: example.com\r\n\r\n")
	n, done, err = h.Parse(data)
	require.NoError(t, err)
	assert.Equal(t, 21, n)
	assert.True(t, done)

	// Test: Whitespace before colon (invalid)
	h = NewHeaders()
	data = []byte("Host : localhost\r\n")
	n, done, err = h.Parse(data)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "malformed")

	// Test: Whitespace in middle of name (invalid)
	h = NewHeaders()
	data = []byte("Ho st: localhost\r\n")
	n, done, err = h.Parse(data)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "malformed")

	// Test: Case insensitive storage
	h = NewHeaders()
	data = []byte("Content-Type: application/json\r\n")
	n, done, err = h.Parse(data)
	require.NoError(t, err)
	val, ok = h.Get("content-type")
	assert.True(t, ok)
	assert.Equal(t, "application/json", val)
	// Should also work with different case
	val, ok = h.Get("CONTENT-TYPE")
	assert.True(t, ok)
	assert.Equal(t, "application/json", val)

	// Test: Invalid character in header name
	h = NewHeaders()
	data = []byte("H©st: localhost\r\n")
	n, done, err = h.Parse(data)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid character")

	// Test: No colon in header
	h = NewHeaders()
	data = []byte("InvalidHeader\r\n")
	n, done, err = h.Parse(data)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "malformed")

	// Test: Obsolete line folding (should reject)
	h = NewHeaders()
	data = []byte("Host: example.com\r\n continued\r\n")
	n, done, err = h.Parse(data)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "line folding")

	// Test: Tab character starting line (obsolete line folding)
	h = NewHeaders()
	data = []byte("Host: example.com\r\n\tcontinued\r\n")
	n, done, err = h.Parse(data)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "line folding")

	// Test: Incomplete headers (no \r\n yet)
	h = NewHeaders()
	data = []byte("Host: example.com")
	n, done, err = h.Parse(data)
	require.NoError(t, err)
	assert.Equal(t, 0, n) // Nothing parsed yet
	assert.False(t, done)
	assert.Len(t, h.GetAll("host"), 0) // No headers stored

	// Test: Add method
	h = NewHeaders()
	h.Add("X-Custom", "value1")
	h.Add("X-Custom", "value2")
	values = h.GetAll("x-custom")
	assert.Equal(t, []string{"value1", "value2"}, values)

	// Test: Set method replaces values
	h = NewHeaders()
	h.Add("X-Custom", "value1")
	h.Add("X-Custom", "value2")
	h.Set("X-Custom", "new-value")
	values = h.GetAll("x-custom")
	assert.Equal(t, []string{"new-value"}, values)

	// Test: Get on non-existent header
	h = NewHeaders()
	val, ok = h.Get("non-existent")
	assert.False(t, ok)
	assert.Equal(t, "", val)

	// Test: Multiple headers in one parse
	h = NewHeaders()
	data = []byte("Host: example.com\r\nContent-Type: text/html\r\nContent-Length: 42\r\n")
	n, done, err = h.Parse(data)
	require.NoError(t, err)
	assert.False(t, done)
	val, _ = h.Get("host")
	assert.Equal(t, "example.com", val)
	val, _ = h.Get("content-type")
	assert.Equal(t, "text/html", val)
	val, _ = h.Get("content-length")
	assert.Equal(t, "42", val)

	// Test: Empty header value (allowed)
	h = NewHeaders()
	data = []byte("X-Empty:\r\n")
	n, done, err = h.Parse(data)
	require.NoError(t, err)
	val, ok = h.Get("x-empty")
	assert.True(t, ok)
	assert.Equal(t, "", val)
}