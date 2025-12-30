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
	fmt.Printf("- Method: %s\n", request.RequestLine.Method)
	fmt.Printf("- Target: %s\n", request.RequestLine.RequestTarget)
	fmt.Printf("- Version: %s\n", request.RequestLine.HttpVersion)
}
