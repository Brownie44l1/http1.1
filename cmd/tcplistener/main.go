package main

import (
	"bufio"
	"fmt"
	"net"

	"http1.1/internal/request"
)

func main() {
	listener, err := net.Listen("tcp", ":42069")
	if err != nil {
		return
	}
	defer listener.Close()
	fmt.Println("Listening on port 42069...")

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Accept error:", err)
			continue
		}

		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)

	request, err := request.RequestFromReader(reader)
	if err != nil {
		fmt.Println("failed to ReadFromRequest")
		return
	}
	fmt.Println("Request Line")
	fmt.Printf("Method: %s\n", request.Method)
	fmt.Printf("Path: %s\n", request.Path)
	fmt.Printf("Version: %s\n", request.Version)

	for key, values := range request.Headers.GetAllHeaders() {
		for _, value := range values {
			fmt.Printf("%s: %s\n", key, value)
		}
	}
	fmt.Println("Body")
	fmt.Printf("%s\n", string(request.Body))

	body := "Hello from your HTTP server!\n"

	response := fmt.Sprintf(
		"HTTP/1.1 200 OK\r\n"+
			"Content-Length: %d\r\n"+
			"Content-Type: text/plain\r\n"+
			"Connection: close\r\n"+
			"\r\n"+
			"%s",
		len(body),
		body,
	)

	conn.Write([]byte(response))
}
