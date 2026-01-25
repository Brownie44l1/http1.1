package response

import (
	"fmt"
	"io"
	"strconv"

	"github.com/Brownie44l1/http/internal/headers"
)

// StatusCode represents HTTP status codes
type StatusCode int

const (
	// 1xx Informational
	StatusContinue StatusCode = 100 // ✅ Issue #11: 100-continue support
	
	// 2xx Success
	StatusOK                  StatusCode = 200
	StatusCreated             StatusCode = 201
	StatusAccepted            StatusCode = 202
	StatusNoContent           StatusCode = 204
	StatusPartialContent      StatusCode = 206 // ✅ Issue #11: Range support
	
	// 3xx Redirection
	StatusMovedPermanently    StatusCode = 301
	StatusFound               StatusCode = 302
	StatusSeeOther            StatusCode = 303
	StatusNotModified         StatusCode = 304 // ✅ Issue #11: ETag support
	StatusTemporaryRedirect   StatusCode = 307
	StatusPermanentRedirect   StatusCode = 308
	
	// 4xx Client Errors
	StatusBadRequest          StatusCode = 400
	StatusUnauthorized        StatusCode = 401
	StatusForbidden           StatusCode = 403
	StatusNotFound            StatusCode = 404
	StatusMethodNotAllowed    StatusCode = 405
	StatusNotAcceptable       StatusCode = 406
	StatusRequestTimeout      StatusCode = 408
	StatusConflict            StatusCode = 409
	StatusPreconditionFailed  StatusCode = 412 // ✅ Issue #11: For ETag
	StatusRequestEntityTooLarge StatusCode = 413
	StatusURITooLong          StatusCode = 414
	StatusUnsupportedMediaType StatusCode = 415
	StatusRequestedRangeNotSatisfiable StatusCode = 416 // ✅ Issue #11: Range
	StatusExpectationFailed   StatusCode = 417 // ✅ Issue #11: Expect
	StatusTooManyRequests     StatusCode = 429
	
	// 5xx Server Errors
	StatusInternalServerError StatusCode = 500
	StatusNotImplemented      StatusCode = 501
	StatusBadGateway          StatusCode = 502
	StatusServiceUnavailable  StatusCode = 503
	StatusGatewayTimeout      StatusCode = 504
)

// statusText maps status codes to reason phrases
var statusText = map[StatusCode]string{
	StatusContinue:            "Continue",
	StatusOK:                  "OK",
	StatusCreated:             "Created",
	StatusAccepted:            "Accepted",
	StatusNoContent:           "No Content",
	StatusPartialContent:      "Partial Content",
	StatusMovedPermanently:    "Moved Permanently",
	StatusFound:               "Found",
	StatusSeeOther:            "See Other",
	StatusNotModified:         "Not Modified",
	StatusTemporaryRedirect:   "Temporary Redirect",
	StatusPermanentRedirect:   "Permanent Redirect",
	StatusBadRequest:          "Bad Request",
	StatusUnauthorized:        "Unauthorized",
	StatusForbidden:           "Forbidden",
	StatusNotFound:            "Not Found",
	StatusMethodNotAllowed:    "Method Not Allowed",
	StatusNotAcceptable:       "Not Acceptable",
	StatusRequestTimeout:      "Request Timeout",
	StatusConflict:            "Conflict",
	StatusPreconditionFailed:  "Precondition Failed",
	StatusRequestEntityTooLarge: "Request Entity Too Large",
	StatusURITooLong:          "URI Too Long",
	StatusUnsupportedMediaType: "Unsupported Media Type",
	StatusRequestedRangeNotSatisfiable: "Requested Range Not Satisfiable",
	StatusExpectationFailed:   "Expectation Failed",
	StatusTooManyRequests:     "Too Many Requests",
	StatusInternalServerError: "Internal Server Error",
	StatusNotImplemented:      "Not Implemented",
	StatusBadGateway:          "Bad Gateway",
	StatusServiceUnavailable:  "Service Unavailable",
	StatusGatewayTimeout:      "Gateway Timeout",
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
	headers       *headers.Headers // Store headers before writing
}

// NewWriter creates a new response writer
func NewWriter(w io.Writer) *Writer {
	return &Writer{
		w:             w,
		state:         stateStart,
		contentLength: -1,
		headers:       headers.NewHeaders(),
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

	// Store headers
	w.headers = h

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
	if w.state != stateHeadersWritten && w.state != stateBodyWritten {
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

// ✅ Issue #5: Flush forces buffered data to be sent
func (w *Writer) Flush() error {
	// Check if underlying writer supports flushing
	if flusher, ok := w.w.(interface{ Flush() error }); ok {
		return flusher.Flush()
	}
	// If not, it's a no-op (data is already written)
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

// Helper methods for common responses

// TextResponse sends a plain text response
func (w *Writer) TextResponse(code StatusCode, text string) error {
	h := headers.NewHeaders()
	h.Set("Content-Type", "text/plain; charset=utf-8")
	h.Set("Content-Length", strconv.Itoa(len(text)))

	if err := w.WriteStatusLine(code); err != nil {
		return err
	}
	if err := w.WriteHeaders(h); err != nil {
		return err
	}
	return w.WriteBody([]byte(text))
}

// HTMLResponse sends an HTML response
func (w *Writer) HTMLResponse(code StatusCode, html string) error {
	h := headers.NewHeaders()
	h.Set("Content-Type", "text/html; charset=utf-8")
	h.Set("Content-Length", strconv.Itoa(len(html)))

	if err := w.WriteStatusLine(code); err != nil {
		return err
	}
	if err := w.WriteHeaders(h); err != nil {
		return err
	}
	return w.WriteBody([]byte(html))
}

// JSONResponse sends a JSON response
func (w *Writer) JSONResponse(code StatusCode, json string) error {
	h := headers.NewHeaders()
	h.Set("Content-Type", "application/json; charset=utf-8")
	h.Set("Content-Length", strconv.Itoa(len(json)))

	if err := w.WriteStatusLine(code); err != nil {
		return err
	}
	if err := w.WriteHeaders(h); err != nil {
		return err
	}
	return w.WriteBody([]byte(json))
}

// ErrorResponse sends an error response
func (w *Writer) ErrorResponse(code StatusCode, message string) error {
	return w.TextResponse(code, message)
}

// RedirectResponse sends a redirect response
func (w *Writer) RedirectResponse(code StatusCode, location string) error {
	h := headers.NewHeaders()
	h.Set("Location", location)
	h.Set("Content-Length", "0")

	if err := w.WriteStatusLine(code); err != nil {
		return err
	}
	return w.WriteHeaders(h)
}

// NoContentResponse sends a 204 No Content response
func (w *Writer) NoContentResponse() error {
	h := headers.NewHeaders()
	if err := w.WriteStatusLine(StatusNoContent); err != nil {
		return err
	}
	return w.WriteHeaders(h)
}

// ✅ Issue #11: ContinueResponse sends 100 Continue
func (w *Writer) ContinueResponse() error {
	// 100 Continue is special - doesn't change state
	statusLine := "HTTP/1.1 100 Continue\r\n\r\n"
	_, err := w.w.Write([]byte(statusLine))
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

func (w *Writer) Headers() *headers.Headers {
	return w.headers
}