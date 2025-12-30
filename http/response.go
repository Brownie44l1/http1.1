package http

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
)

type Response struct {
	Conn       net.Conn
	StatusCode int
	Body       []byte
	Headers    map[string][]string
}

func NewResponse(conn net.Conn) *Response {
	return &Response{
		Conn:       conn,
		StatusCode: 200,
		Headers:    make(map[string][]string),
	}
}

func (r *Response) Send(body []byte) error {
	r.Body = body
	return r.Write()
}

func (r *Response) Write() error {
	if r.Headers == nil {
		r.Headers = make(map[string][]string)
	}

	if _, ok := r.Headers["Content-Type"]; !ok {
		r.Headers["Content-Type"] = append(r.Headers["Content-Type"], "text/plain")
	}

	if _, ok := r.Headers["Content-Length"]; !ok {
		r.Headers["Content-Length"] = append(r.Headers["Content-Length"], strconv.Itoa(len(r.Body)))
	}

	if _, ok := r.Headers["Connection"]; !ok {
		r.Headers["Connection"] = append(r.Headers["Connection"], "close")
	}

	statusText := http.StatusText(r.StatusCode)
	if statusText == "" {
		statusText = "Unknown Status"
	}

	var headerLines strings.Builder
	for key, values := range r.Headers {
		for _, value := range values {
			fmt.Fprintf(&headerLines, "%s: %s\r\n", key, value)
		}
	}

	response := fmt.Sprintf(
		"HTTP/1.1 %d %s\r\n%s\r\n",
		r.StatusCode,
		statusText,
		headerLines.String(),
	)

	if _, err := io.WriteString(r.Conn, response); err != nil {
		return err
	}

	totalWritten := 0
	for totalWritten < len(r.Body) {
		n, err := r.Conn.Write(r.Body[totalWritten:])
		if err != nil {
			return err
		}
		totalWritten += n
	}
	return nil
}
