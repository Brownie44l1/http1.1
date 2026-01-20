package response

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Brownie44l1/http1.1/internal/headers"
)

func TestSimpleResponse(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)

	err := w.WriteStatusLine(StatusOK)
	require.NoError(t, err)

	h := headers.NewHeaders()
	h.Set("Content-Type", "text/plain")
	h.Set("Content-Length", "5")

	err = w.WriteHeaders(h)
	require.NoError(t, err)

	err = w.WriteBody([]byte("Hello"))
	require.NoError(t, err)

	result := buf.String()
	assert.Contains(t, result, "HTTP/1.1 200 OK\r\n")
	assert.Contains(t, result, "content-type: text/plain\r\n")
	assert.Contains(t, result, "content-length: 5\r\n")
	assert.Contains(t, result, "\r\n\r\n")
	assert.Contains(t, result, "Hello")
}

func TestStatusCodes(t *testing.T) {
	tests := []struct {
		code   StatusCode
		reason string
	}{
		{StatusOK, "OK"},
		{StatusCreated, "Created"},
		{StatusBadRequest, "Bad Request"},
		{StatusNotFound, "Not Found"},
		{StatusInternalServerError, "Internal Server Error"},
	}

	for _, tt := range tests {
		var buf bytes.Buffer
		w := NewWriter(&buf)

		err := w.WriteStatusLine(tt.code)
		require.NoError(t, err)

		result := buf.String()
		expected := fmt.Sprintf("HTTP/1.1 %d %s", tt.code, tt.reason)
		assert.Contains(t, result, expected)
	}
}

func TestChunkedEncoding(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)

	err := w.WriteStatusLine(StatusOK)
	require.NoError(t, err)

	h := headers.NewHeaders()
	h.Set("Transfer-Encoding", "chunked")
	err = w.WriteHeaders(h)
	require.NoError(t, err)

	// Write first chunk
	err = w.WriteChunk([]byte("Hello"))
	require.NoError(t, err)

	// Write second chunk
	err = w.WriteChunk([]byte(", World"))
	require.NoError(t, err)

	// Finish chunked
	err = w.FinishChunked()
	require.NoError(t, err)

	result := buf.String()
	assert.Contains(t, result, "5\r\nHello\r\n")   // First chunk
	assert.Contains(t, result, "7\r\n, World\r\n") // Second chunk
	assert.Contains(t, result, "0\r\n\r\n")        // Final chunk
	assert.True(t, w.IsChunked())
}

func TestStateValidation(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)

	// Can't write headers before status
	h := headers.NewHeaders()
	err := w.WriteHeaders(h)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status line")

	// Write status
	err = w.WriteStatusLine(StatusOK)
	require.NoError(t, err)

	// Can't write status again
	err = w.WriteStatusLine(StatusOK)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already written")

	// Can't write body before headers
	err = w.WriteBody([]byte("test"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "headers")
}

func TestTextResponse(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)

	err := w.TextResponse(StatusOK, "Hello, World!")
	require.NoError(t, err)

	result := buf.String()
	assert.Contains(t, result, "200 OK")
	assert.Contains(t, result, "content-type: text/plain")
	assert.Contains(t, result, "content-length: 13")
	assert.Contains(t, result, "Hello, World!")
}

func TestHTMLResponse(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)

	html := "<html><body>Test</body></html>"
	err := w.HTMLResponse(StatusOK, html)
	require.NoError(t, err)

	result := buf.String()
	assert.Contains(t, result, "text/html")
	assert.Contains(t, result, html)
}

func TestJSONResponse(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)

	json := `{"status":"ok"}`
	err := w.JSONResponse(StatusOK, json)
	require.NoError(t, err)

	result := buf.String()
	assert.Contains(t, result, "application/json")
	assert.Contains(t, result, json)
}

func TestErrorResponse(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)

	err := w.ErrorResponse(StatusNotFound, "Page not found")
	require.NoError(t, err)

	result := buf.String()
	assert.Contains(t, result, "404")
	assert.Contains(t, result, "Page not found")
}

func TestNoContentResponse(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)

	err := w.NoContentResponse()
	require.NoError(t, err)

	result := buf.String()
	assert.Contains(t, result, "204 No Content")
	// Should have headers but no body
	assert.True(t, strings.HasSuffix(result, "\r\n\r\n"))
}

func TestContentLengthTracking(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)

	err := w.WriteStatusLine(StatusOK)
	require.NoError(t, err)

	h := headers.NewHeaders()
	h.Set("Content-Length", "42")
	err = w.WriteHeaders(h)
	require.NoError(t, err)

	assert.True(t, w.HasContentLength())
	assert.False(t, w.IsChunked())
}

func TestChunkedTracking(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)

	err := w.WriteStatusLine(StatusOK)
	require.NoError(t, err)

	h := headers.NewHeaders()
	h.Set("Transfer-Encoding", "chunked")
	err = w.WriteHeaders(h)
	require.NoError(t, err)

	assert.True(t, w.IsChunked())
	assert.False(t, w.HasContentLength())
}

func TestMultipleHeaderValues(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)

	err := w.WriteStatusLine(StatusOK)
	require.NoError(t, err)

	h := headers.NewHeaders()
	h.Add("Set-Cookie", "session=abc")
	h.Add("Set-Cookie", "user=xyz")
	err = w.WriteHeaders(h)
	require.NoError(t, err)

	result := buf.String()
	// Both Set-Cookie headers should be present
	assert.Contains(t, result, "set-cookie: session=abc")
	assert.Contains(t, result, "set-cookie: user=xyz")
}

func TestEmptyBody(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)

	err := w.WriteStatusLine(StatusOK)
	require.NoError(t, err)

	h := headers.NewHeaders()
	h.Set("Content-Length", "0")
	err = w.WriteHeaders(h)
	require.NoError(t, err)

	err = w.WriteBody([]byte{})
	require.NoError(t, err)

	result := buf.String()
	// Should end with headers, no body content
	assert.True(t, strings.HasSuffix(result, "\r\n\r\n"))
}

func TestErrorTracking(t *testing.T) {
	// Use a failing writer
	w := NewWriter(&failWriter{})

	assert.False(t, w.HadError())

	_ = w.WriteStatusLine(StatusOK)
	assert.True(t, w.HadError())
}

// failWriter always returns an error
type failWriter struct{}

func (f *failWriter) Write(p []byte) (int, error) {
	return 0, assert.AnError
}
