package main

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

	startTime := time.Now()
	metrics.StartRequest()

	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('\n')

	if err != nil {
		return
	}

	parts := strings.Split(strings.TrimSpace(line), " ")
	if len(parts) != 3 {
		fmt.Println("Error reading request")
		return
	}

	method := parts[0]
	target := parts[1]
	version := parts[2]

	path := target
	query := ""

	if idx := strings.Index(path, "?"); idx != -1 {
		path = target[:idx]
		query = target[idx+1:]
	}

	queryParam := make(map[string]string)
	for _, param := range strings.Split(query, "&") {
		kv := strings.SplitN(param, "=", 2)
		if len(kv) == 2 {
			queryParam[kv[0]] = kv[1]
		}
	}

	switch strings.ToUpper(method) {
	case "GET", "POST", "PUT", "DELETE":
		fmt.Printf("Success: Processing method: %s\n", method)
	default:
		sendError(conn, 405, "Invalid Method")
		return
	}

	if !strings.HasPrefix(path, "/") {
		sendError(conn, 404, "Bad request")
		return
	}

	if version != "HTTP/1.1" {
		sendError(conn, 505, "HTTP Version Not Supported")
		return
	}

	fmt.Printf("Request Line: %s %s %s\n", method, path, version)

	//Read header
	headers := make(map[string]string)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}

		line = strings.TrimSpace(line)
		if line == "" {
			break
		}

		kv := strings.SplitN(line, ":", 2)
		if len(kv) != 2 {
			sendError(conn, 400, "Malformed Header")
			return
		}
		headers[(kv[0])] = strings.TrimSpace(kv[1])
	}

	var bodyData string
	if cl, ok := headers["Content-Length"]; ok {
		lenght, err := strconv.Atoi(cl)
		if err == nil && lenght > 0 {
			buf := make([]byte, lenght)
			_, err := io.ReadFull(reader, buf)
			if err == nil {
				bodyData = string(buf)
			}
		}
	}

	req := &Request{
		Method:  method,
		Path:    path,
		Body:    bodyData,
		Headers: headers,
	}

	res := NewResponse(conn)

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
