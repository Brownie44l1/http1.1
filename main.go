package main

import (
	"bufio"
	"fmt"
	"net"

	"http1.1/http"
)

type Handler interface {
	ServeHTTP(req *http.Request, resp *http.Response)
}

type HandlerFunc func(req *http.Request, resp *http.Response)

func (f HandlerFunc) ServeHTTP(req *http.Request, resp *http.Response) {
	f(req, resp)
}

func main() {
	handler := HandlerFunc(func(req *http.Request, resp *http.Response) {
		body := fmt.Sprintf("Method: %s\nPath: %s\n", req.Method, req.Path)
		resp.StatusCode = 200
		resp.Headers["Content-Type"] = []string{"text/plain"}
		resp.Send([]byte(body))
	})

	ListenAndServe(":8080", handler)
}

func ListenAndServe(addr string, handler Handler) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Accept error:", err)
			continue
		}
		go handleConnection(conn, handler)
	}
}

func handleConnection(conn net.Conn, handler Handler) {
    defer conn.Close()
    reader := bufio.NewReader(conn)

    req, err := http.ParseRequest(reader)
    if err != nil {
        fmt.Println("Parse error:", err)
        resp := http.NewResponse(conn)
        resp.StatusCode = 400
        resp.Headers = map[string][]string{"Content-Type": {"text/plain"}}
        resp.Send([]byte("Bad Request"))
        return
    }

    // Create response and call handler
    resp := http.NewResponse(conn)
    handler.ServeHTTP(req, resp)
}
