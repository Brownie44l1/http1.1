package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
)

func main() {
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		panic(err)
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}
		go handleConn(conn)
	}
}

func handleConn(conn net.Conn) {
	defer conn.Close()

	//Read & Parse request
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

	//Response
	if method == "GET" && path == "/home" {
		sendResponse(conn, 200, "OK", "You sent: "+bodyData+"\n")
	} else if method == "POST" && path == "/home" {
		sendResponse(conn, 200, "OK", "You sent: "+bodyData+"\n")
	} else {
		sendError(conn, 404, "Not available yet!")
	}
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

func sendResponse(conn net.Conn, code int, status string, body string) {
	response := fmt.Sprintf(
		"HTTP/1.1 %d %s\r\n"+
			"Content-Length: %d\r\n"+
			"Content-Type: text/plain\r\n"+
			"\r\n"+
			"%s",
		code,
		status,
		len(body),
		body,
	)

	conn.Write([]byte(response))
}
