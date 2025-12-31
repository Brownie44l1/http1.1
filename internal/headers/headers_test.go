package headers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHeaderParse(t *testing.T) {
    // Test: Valid single header
    headers := NewHeaders()
    data := []byte("Host: localhost:42069\r\n")
    n, done, err := headers.Parse(data)
    require.NoError(t, err)
    assert.Equal(t, "localhost:42069", headers.Header["host"])
    assert.Equal(t, 23, n)
    assert.False(t, done)
    
    // Test: Valid single header with extra whitespace
    headers = NewHeaders()
    data = []byte("Host:   localhost:42069   \r\n")
    n, done, err = headers.Parse(data)
    require.NoError(t, err)
    assert.Equal(t, "localhost:42069", headers.Header["host"])
    assert.False(t, done)
    
    // Test: Valid 2 headers with existing headers
    headers = NewHeaders()
    headers.Header["host"] = "example.com"
    data = []byte("Host: localhost:42069\r\n")
    n, done, err = headers.Parse(data)
    require.NoError(t, err)
    assert.Equal(t, "example.com, localhost:42069", headers.Header["host"])
    assert.False(t, done)
    
    // Test: Valid done
    headers = NewHeaders()
    data = []byte("\r\n")
    n, done, err = headers.Parse(data)
    require.NoError(t, err)
    assert.Equal(t, 2, n)
    assert.True(t, done)
    
    // Test: Invalid spacing header
    headers = NewHeaders()
    data = []byte("Host : localhost:42069\r\n")
    n, done, err = headers.Parse(data)
    require.Error(t, err)
    assert.Equal(t, 0, n)
    assert.False(t, done)
    
    // Test: Capital letters in key
    headers = NewHeaders()
    data = []byte("Content-Type: application/json\r\n")
    n, done, err = headers.Parse(data)
    require.NoError(t, err)
    assert.Equal(t, "application/json", headers.Header["content-type"])
    
    // Test: Invalid character in key
    headers = NewHeaders()
    data = []byte("H©st: localhost:42069\r\n")
    n, done, err = headers.Parse(data)
    require.Error(t, err)
}