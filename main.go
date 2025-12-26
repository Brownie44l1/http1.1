package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
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

	//Read request
	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('\n')

	if err != nil {
		return
	}

	//Parse request
	parts := strings.Fields(line)
	if len(parts) != 3 {
		log.Fatalln("Error reading request")
	}
	method := parts[0]
	path := parts[1]
	version := parts[2]

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
			return
		}
		headers[kv[0]] = strings.TrimSpace(kv[1])
	}

	//Response
	body := "Hello Server!\n"

	response := fmt.Sprintf(
		"HTTP/1.1 200 OK\r\n"+
			"Content-Length: %d\r\n"+
			"Content-Type: text/plain\r\n"+
			"\r\n"+
			"%s",
		len(body),
		body,
	)

	conn.Write([]byte(response))
}
