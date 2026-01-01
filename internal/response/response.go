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

func (w *Writer) WriteChunkedBody(p []byte) (int, error) {
	if w.state != stateHeadersWritten && w.state != stateBodyWritten {
		return 0, fmt.Errorf("must write status line and headers before chunked body")
	}

	if len(p) == 0 {
		return 0, nil
	}

	// Write chunk size in hexadecimal
	chunkSize := fmt.Sprintf("%x\r\n", len(p))
	_, err := w.w.Write([]byte(chunkSize))
	if err != nil {
		return 0, err
	}

	// Write chunk data
	n, err := w.w.Write(p)
	if err != nil {
		return n, err
	}

	// Write trailing CRLF
	_, err = w.w.Write([]byte("\r\n"))
	if err != nil {
		return n, err
	}

	w.state = stateBodyWritten
	return n, nil
}

func (w *Writer) WriteChunkedBodyDone() (int, error) {
	if w.state != stateHeadersWritten && w.state != stateBodyWritten {
		return 0, fmt.Errorf("must write status line and headers before ending chunked body")
	}

	// Write the final zero-sized chunk
	n, err := w.w.Write([]byte("0\r\n\r\n"))
	if err != nil {
		return n, err
	}

	w.state = stateBodyWritten
	return n, nil
}

func (w *Writer) WriteTrailers(h headers.Headers) error {
	if w.state != stateBodyWritten {
		return fmt.Errorf("must write body before trailers")
	}

	// Write trailers just like headers
	for key, value := range h.Header {
		trailerLine := fmt.Sprintf("%s: %s\r\n", key, value)
		_, err := w.w.Write([]byte(trailerLine))
		if err != nil {
			return err
		}
	}

	// Write final CRLF to end the message
	_, err := w.w.Write([]byte("\r\n"))
	return err
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
