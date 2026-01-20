package response

import (
	"fmt"

	"github.com/Brownie44l1/http1.1/internal/headers"
)

// TextResponse writes a simple text response
func (w *Writer) TextResponse(code StatusCode, body string) error {
	if err := w.WriteStatusLine(code); err != nil {
		return err
	}

	h := headers.NewHeaders()
	h.Set("Content-Type", "text/plain; charset=utf-8")
	h.Set("Content-Length", fmt.Sprintf("%d", len(body)))

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
	h.Set("Content-Type", "application/json; charset=utf-8")
	h.Set("Content-Length", fmt.Sprintf("%d", len(body)))

	if err := w.WriteHeaders(h); err != nil {
		return err
	}

	return w.WriteBody([]byte(body))
}

// ErrorResponse writes a standard error response
func (w *Writer) ErrorResponse(code StatusCode, message string) error {
	if message == "" {
		if text, ok := statusText[code]; ok {
			message = text
		} else {
			message = "Unknown Error"
		}
	}

	body := fmt.Sprintf("Error %d: %s\n", code, message)
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

// RedirectResponse writes a redirect response
func (w *Writer) RedirectResponse(code StatusCode, location string) error {
	if code != 301 && code != 302 && code != 303 && code != 307 && code != 308 {
		return fmt.Errorf("invalid redirect status code: %d", code)
	}

	if err := w.WriteStatusLine(code); err != nil {
		return err
	}

	h := headers.NewHeaders()
	h.Set("Location", location)
	h.Set("Content-Length", "0")

	if err := w.WriteHeaders(h); err != nil {
		return err
	}

	return w.WriteBody(nil)
}

// BytesResponse writes a response with arbitrary byte content
func (w *Writer) BytesResponse(code StatusCode, contentType string, data []byte) error {
	if err := w.WriteStatusLine(code); err != nil {
		return err
	}

	h := headers.NewHeaders()
	if contentType != "" {
		h.Set("Content-Type", contentType)
	}
	h.Set("Content-Length", fmt.Sprintf("%d", len(data)))

	if err := w.WriteHeaders(h); err != nil {
		return err
	}

	return w.WriteBody(data)
}