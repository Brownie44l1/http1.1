package main

import (
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
)

type Request struct {
	Method string
	Path string
	Body string
	Headers map[string]string
}

type Response struct {
	Conn net.Conn
	StatusCode int
	Body string
	Headers map[string]string
}

type Route struct {
	Method string
	Path string
	Handler func(*Request, *Response)
}

type Router struct {
    routes []Route
}

func NewRouter() *Router {
    return &Router{
        routes: make([]Route, 0),
    }
}

func NewResponse(conn net.Conn) *Response {
	return &Response{
		Conn: conn,
		StatusCode: 200,
		Headers: make(map[string]string),
	}
}

func (r *Response) Send(body string) {
    r.Body = body
	r.Write()
}

func (r *Response) Write() error {
	if r.Headers == nil {
		r.Headers = make(map[string]string)
	}

	bodyBytes := []byte(r.Body)

	if _, ok := r.Headers["Content-Type"]; !ok {
		r.Headers["Content-Type"] = "text/plain"
	}

	if _, ok := r.Headers["Content-Length"]; !ok {
		r.Headers["Content-Length"] = strconv.Itoa(len(bodyBytes))
	}

	statusText := http.StatusText(r.StatusCode)
	if statusText == "" {
		statusText = "Unknown Status"
	}

	var headerLines strings.Builder
	for key, value := range r.Headers {
		headerLines.WriteString(fmt.Sprintf("%s: %s\r\n", key, value))
	}

	response := fmt.Sprintf(
		"HTTP/1.1 %d %s\r\n%s\r\n%s",
		r.StatusCode,
		statusText,
		headerLines.String(),
		r.Body,
	)

	_, err := r.Conn.Write([]byte(response))
	return err
}

func (r *Router) Match(method, path string) (func(*Request, *Response), bool) {
    for _, route := range r.routes {
        if route.Method != method {
            continue  // Wrong HTTP method, skip
        }
        
        // Check if route pattern has wildcard
        if strings.HasSuffix(route.Path, "/*") {
            // Extract prefix (everything before /*)
            prefix := strings.TrimSuffix(route.Path, "/*")
            
            // Check if request path starts with this prefix
            if strings.HasPrefix(path, prefix) {
                return route.Handler, true
            }
        } else {
            // Exact match
            if route.Path == path {
                return route.Handler, true
            }
        }
    }
    return nil, false
}

func (r *Router) Add(method, path string, handler func(*Request, *Response)) {
    r.routes = append(r.routes, Route{
        Method:  method,
        Path:    path,
        Handler: handler,
    })
}
