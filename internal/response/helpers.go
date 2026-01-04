package response

import (
	"fmt"
	
	"http1.1/internal/headers"
)

// TextResponse writes a simple text response
func (w *Writer) TextResponse(code StatusCode, body string) error {
	if err := w.WriteStatusLine(code); err != nil {
		return err
	}

	h := headers.NewHeaders()
	h.Set("Content-Type", "text/plain")
	h.Set("Content-Length", fmt.Sprintf("%d", len(body)))
	h.Set("Connection", "close")

	if err := w.WriteHeaders(h); err != nil {
		return err
	}

	return w.WriteBody([]byte(body))
}

// HTMLResponse writes an HTML response
func (w *Writer) HTMLResponse(code StatusCode, body string) error {
	if err := w.WriteStatusLine(code); err != nil {
		return err
	}

	h := headers.NewHeaders()
	h.Set("Content-Type", "text/html; charset=utf-8")
	h.Set("Content-Length", fmt.Sprintf("%d", len(body)))

	if err := w.WriteHeaders(h); err != nil {
		return err
	}

	return w.WriteBody([]byte(body))
}

// JSONResponse writes a JSON response
func (w *Writer) JSONResponse(code StatusCode, body string) error {
	if err := w.WriteStatusLine(code); err != nil {
		return err
	}

	h := headers.NewHeaders()
	h.Set("Content-Type", "application/json")
	h.Set("Content-Length", fmt.Sprintf("%d", len(body)))

	if err := w.WriteHeaders(h); err != nil {
		return err
	}

	return w.WriteBody([]byte(body))
}

// ErrorResponse writes a standard error response
func (w *Writer) ErrorResponse(code StatusCode, message string) error {
	if message == "" {
		message = statusText[code]
	}

	body := fmt.Sprintf("Error %d: %s", code, message)
	return w.TextResponse(code, body)
}

// NoContentResponse writes a 204 No Content response
func (w *Writer) NoContentResponse() error {
	if err := w.WriteStatusLine(StatusNoContent); err != nil {
		return err
	}

	h := headers.NewHeaders()
	return w.WriteHeaders(h)
}

// ChunkedResponse starts a chunked response
func (w *Writer) ChunkedResponse(code StatusCode, contentType string) error {
	if err := w.WriteStatusLine(code); err != nil {
		return err
	}

	h := headers.NewHeaders()
	h.Set("Transfer-Encoding", "chunked")
	if contentType != "" {
		h.Set("Content-Type", contentType)
	}

	return w.WriteHeaders(h)
}