package response

import (
	"fmt"
	"io"

	"http1.1/internal/headers"
)

type StatusCode int

const (
	StatusOk                  StatusCode = 200
	StatusBadRequest          StatusCode = 400
	StatusInternalServerError StatusCode = 500
)

type writerState int

const (
	stateStart writerState = iota
	stateStatusWritten
	stateHeadersWritten
	stateBodyWritten
)

type Writer struct {
	w     io.Writer
	state writerState
}

func NewWriter(w io.Writer) *Writer {
	return &Writer{
		w:     w,
		state: stateStart,
	}
}

func (w *Writer) WriteStatusLine(statusCode StatusCode) error {
	if w.state != stateStart {
		return fmt.Errorf("status line already written")
	}
	var reasonPhase string

	switch statusCode {
	case StatusOk:
		reasonPhase = "OK"
	case StatusBadRequest:
		reasonPhase = "Bad Request"
	case StatusInternalServerError:
		reasonPhase = "Internal Server Error"
	default:
		reasonPhase = ""
	}

	statusLine := fmt.Sprintf("HTTP/1.1 %d %s\r\n", statusCode, reasonPhase)
	_, err := w.w.Write([]byte(statusLine))
	if err != nil {
		return err
	}

	w.state = stateStatusWritten
	return nil
}

func (w *Writer) WriteHeaders(headers headers.Headers) error {
	if w.state != stateStatusWritten {
		return fmt.Errorf("must write status line before headers")
	}
	for key, value := range headers.Header {
		headerLine := fmt.Sprintf("%s: %s\r\n", key, value)
		_, err := w.w.Write([]byte(headerLine))
		if err != nil {
			return err
		}
	}

	_, err := w.w.Write([]byte("\r\n"))
	if err != nil {
		return err
	}
	w.state = stateHeadersWritten
	return nil
}

func (w *Writer) WriteBody(p []byte) (int, error) {
	if w.state != stateHeadersWritten {
		return 0, fmt.Errorf("must write status line and headers before body")
	}

	n, err := w.w.Write(p)
	if err != nil {
		return n, err
	}

	w.state = stateBodyWritten
	return n, nil
}

// Legacy functions for GetDefaultHeaders
func GetDefaultHeaders(contentLen int) headers.Headers {
	return headers.Headers{
		Header: map[string]string{
			"Content-Length": fmt.Sprintf("%d", contentLen),
			"Connection":     "close",
			"Content-Type":   "text/plain",
		},
	}
}
