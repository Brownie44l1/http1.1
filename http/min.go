package http
/*

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


import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var metrics = &Metrics{}

func main() {
	router := NewRouter()

	router.Add("GET", "/home", handleHome)
	router.Add("POST", "/home", handlePost)
	router.Add("GET", "/static/*", serveStatic)
	router.Add("POST", "/api/test", apiTest)
	router.Add("GET", "/api/metrics", getMetrics)

	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		panic(err)
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}
		go handleConn(conn, router)
	}
}

func handleHome(req *Request, res *Response) {
	res.StatusCode = 200
	res.Send("Welcome home!")
}

func handlePost(req *Request, res *Response) {
	res.StatusCode = 200
	res.Send("You sent: " + req.Body)
}

func serveStatic(req *Request, res *Response) {
	prefix := "/static/"
	path := req.Path
	path = strings.TrimPrefix(path, prefix)
	path = "./public/" + path

	content, err := os.ReadFile(path)
	if err != nil {
		res.StatusCode = 404
		sendError(res.Conn, res.StatusCode, "File not found")
	}

	ext := filepath.Ext(path)

	switch ext {
	case ".html":
		res.Headers["Content-Type"] = "text/html"
	case ".css":
		res.Headers["Content-Type"] = "text/css"
	case ".js":
		res.Headers["Content-Type"] = "application/javascript"
	default:
		res.Headers["Content-Type"] = "application/octet-stream"
	}

	res.Send(string(content))
}

func apiTest(req *Request, res *Response) {
	// Example: Receive JSON
	var data map[string]string
	err := req.ParseJSON(&data)
	if err != nil {
		res.StatusCode = 400
		res.Send("Invalid JSON\n")
		return
	}

	// Example: Send JSON
	response := map[string]string{
		"message": "Received your data",
		"name":    data["name"],
	}
	res.JSON(response)
}

func getMetrics(req *Request, res *Response) {
    stats := metrics.GetStats()
    res.StatusCode = 200
    res.JSON(stats)
}

func handleConn(conn net.Conn, router *Router) {
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	req, err := parseRequest
	if err != nil {
        // Handle timeout or malformed request
        return
    }
	conn.SetWriteDeadline(time.Now().Add(30 * time.Second))

	startTime := time.Now()
	metrics.StartRequest()

	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('\n')

	if err != nil {
		return
	}

	handler, found := router.Match(method, path)
	if !found {
        sendError(conn, 404, "Not Found")
        
        duration := time.Since(startTime)
        metrics.EndRequest(duration, 404)
        return
    }

	handler(req, res)
	duration := time.Since(startTime)
    metrics.EndRequest(duration, res.StatusCode)
}

// Helper functions.
func sendError(conn net.Conn, code int, message string) {
	body := fmt.Sprintf("%d %s\n", code, message)

	response := fmt.Sprintf(
		"HTTP/1.1 %d %s\r\n"+
			"Content-Length: %d\r\n"+
			"Content-Type: text/plain\r\n"+
			"\r\n"+
			"%s",
		code,
		message,
		len(body),
		body,
	)

	conn.Write([]byte(response))
}

func (s *Server) handleConnection(conn net.Conn) {
    defer conn.Close()
    
    // Loop for multiple requests on same connection
    for {
        // Set deadline for each request
        conn.SetReadDeadline(time.Now().Add(30 * time.Second))
        
        request, err := ParseRequest(conn)
        if err != nil {
            // Connection closed or error - break loop
            return
        }
        
        response := NewResponse(conn)
        
        // Find and execute handler
        handler := s.router.Match(request.Method, request.Path)
        if handler != nil {
            handler(request, response)
        } else {
            response.StatusCode = 404
            response.SendText("Not Found")
        }
        
        // Check if we should keep connection alive
        connectionHeader := strings.ToLower(request.Headers["Connection"])
        
        // HTTP/1.1 defaults to keep-alive, HTTP/1.0 doesn't
        shouldKeepAlive := request.Version == "HTTP/1.1"
        
        if connectionHeader == "close" {
            shouldKeepAlive = false
        } else if connectionHeader == "keep-alive" {
            shouldKeepAlive = true
        }
        
        if !shouldKeepAlive {
            return // Exit loop, defer will close connection
        }
        
        // Continue to next request on this connection
    }
} */ 
