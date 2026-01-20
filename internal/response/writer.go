package response

import (
	"fmt"
	"io"
	"strconv"

	"github.com/Brownie44l1/http1.1/internal/headers"
)

// StatusCode represents HTTP status codes
type StatusCode int

const (
	StatusOK                  StatusCode = 200
	StatusCreated             StatusCode = 201
	StatusNoContent           StatusCode = 204
	StatusBadRequest          StatusCode = 400
	StatusNotFound            StatusCode = 404
	StatusInternalServerError StatusCode = 500
)

// statusText maps status codes to reason phrases
var statusText = map[StatusCode]string{
	StatusOK:                  "OK",
	StatusCreated:             "Created",
	StatusNoContent:           "No Content",
	StatusBadRequest:          "Bad Request",
	StatusNotFound:            "Not Found",
	StatusInternalServerError: "Internal Server Error",
}

// writerState tracks what's been written so far
type writerState int

const (
	stateStart writerState = iota
	stateStatusWritten
	stateHeadersWritten
	stateBodyWritten
)

// Writer writes HTTP responses to an io.Writer
type Writer struct {
	w             io.Writer
	state         writerState
	statusCode    StatusCode
	contentLength int64 // -1 means unknown
	isChunked     bool
	hadError      bool
}

// NewWriter creates a new response writer
func NewWriter(w io.Writer) *Writer {
	return &Writer{
		w:             w,
		state:         stateStart,
		contentLength: -1,
	}
}

// WriteStatusLine writes the HTTP status line
func (w *Writer) WriteStatusLine(code StatusCode) error {
	if w.state != stateStart {
		return fmt.Errorf("status line already written")
	}

	reason, ok := statusText[code]
	if !ok {
		reason = "Unknown"
	}

	statusLine := fmt.Sprintf("HTTP/1.1 %d %s\r\n", code, reason)
	_, err := w.w.Write([]byte(statusLine))
	if err != nil {
		w.hadError = true
		return err
	}

	w.statusCode = code
	w.state = stateStatusWritten
	return nil
}

// WriteHeaders writes all HTTP headers
func (w *Writer) WriteHeaders(h *headers.Headers) error {
	if w.state != stateStatusWritten {
		return fmt.Errorf("must write status line before headers")
	}

	// Track important headers for connection management
	if cl, ok := h.Get("content-length"); ok {
		if length, err := strconv.ParseInt(cl, 10, 64); err == nil {
			w.contentLength = length
		}
	}

	if te, ok := h.Get("transfer-encoding"); ok {
		if te == "chunked" {
			w.isChunked = true
		}
	}

	// Write all headers
	for key, values := range h.GetAllHeaders() {
		for _, value := range values {
			headerLine := fmt.Sprintf("%s: %s\r\n", key, value)
			_, err := w.w.Write([]byte(headerLine))
			if err != nil {
				w.hadError = true
				return err
			}
		}
	}

	// Write empty line to end headers
	_, err := w.w.Write([]byte("\r\n"))
	if err != nil {
		w.hadError = true
		return err
	}

	w.state = stateHeadersWritten
	return nil
}

// WriteBody writes the complete response body
func (w *Writer) WriteBody(data []byte) error {
	if w.state != stateHeadersWritten {
		return fmt.Errorf("must write headers before body")
	}

	if len(data) == 0 {
		w.state = stateBodyWritten
		return nil
	}

	_, err := w.w.Write(data)
	if err != nil {
		w.hadError = true
		return err
	}

	w.state = stateBodyWritten
	return nil
}

// WriteChunk writes a single chunk (for chunked transfer encoding)
func (w *Writer) WriteChunk(data []byte) error {
	if w.state != stateHeadersWritten && w.state != stateBodyWritten {
		return fmt.Errorf("must write headers before chunks")
	}

	if len(data) == 0 {
		return nil // Don't write empty chunks (except final)
	}

	// Write chunk size in hex
	chunkSize := fmt.Sprintf("%x\r\n", len(data))
	if _, err := w.w.Write([]byte(chunkSize)); err != nil {
		w.hadError = true
		return err
	}

	// Write chunk data
	if _, err := w.w.Write(data); err != nil {
		w.hadError = true
		return err
	}

	// Write trailing CRLF
	if _, err := w.w.Write([]byte("\r\n")); err != nil {
		w.hadError = true
		return err
	}

	w.state = stateBodyWritten
	return nil
}

// FinishChunked writes the final zero-length chunk
func (w *Writer) FinishChunked() error {
	if w.state != stateHeadersWritten && w.state != stateBodyWritten {
		return fmt.Errorf("must write headers before finishing chunks")
	}

	// Write final chunk: 0\r\n\r\n
	_, err := w.w.Write([]byte("0\r\n\r\n"))
	if err != nil {
		w.hadError = true
		return err
	}

	w.state = stateBodyWritten
	return nil
}

// WriteTrailers writes HTTP trailers (after chunked body)
func (w *Writer) WriteTrailers(h *headers.Headers) error {
	if w.state != stateBodyWritten {
		return fmt.Errorf("must write body before trailers")
	}

	// Write trailers just like headers
	for key, values := range h.GetAllHeaders() {
		for _, value := range values {
			line := fmt.Sprintf("%s: %s\r\n", key, value)
			if _, err := w.w.Write([]byte(line)); err != nil {
				w.hadError = true
				return err
			}
		}
	}

	// Final CRLF
	_, err := w.w.Write([]byte("\r\n"))
	if err != nil {
		w.hadError = true
	}
	return err
}

// State tracking methods for connection management

func (w *Writer) HadError() bool {
	return w.hadError
}

func (w *Writer) HasContentLength() bool {
	return w.contentLength >= 0
}

func (w *Writer) IsChunked() bool {
	return w.isChunked
}

func (w *Writer) StatusCode() StatusCode {
	return w.statusCode
}
