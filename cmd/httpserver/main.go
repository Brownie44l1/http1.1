package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"http1.1/internal/headers"
	"http1.1/internal/request"
	"http1.1/internal/response"
	"http1.1/internal/server"
)

func main() {
	handler := func(w *response.Writer, r *request.Request) {
		var statusCode response.StatusCode
		var html string

		switch r.RequestLine.RequestTarget {
		case "/yourproblem":
			statusCode = response.StatusBadRequest
			html = `
			<html>
			<head>
				<title>400 Bad Request</title>
			</head>
			<body>
				<h1>Bad Request</h1>
				<p>Your request honestly kinda sucked.</p>
			</body>
			</html>`

		case "/myproblem":
			statusCode = response.StatusInternalServerError
			html = `
			<html>
			<head>
				<title>500 Internal Server Error</title>
			</head>
			<body>
				<h1>Internal Server Error</h1>
				<p>Okay, you know what? This one is on me.</p>
			</body>
			</html>`

		default:
			statusCode = response.StatusOk
			html = `
			<html>
			<head>
				<title>200 OK</title>
			</head>
			<body>
				<h1>Success!</h1>
				<p>Your request was an absolute banger.</p>
			</body>
			</html>`
		}

		// Write status line
		err := w.WriteStatusLine(statusCode)
		if err != nil {
			log.Printf("Error writing status: %v", err)
			return
		}

		// Create and customize headers
		h := headers.Headers{
			Header: map[string]string{
				"Content-Length": fmt.Sprintf("%d", len(html)),
				"Connection":     "close",
				"Content-Type":   "text/html",
			},
		}

		// Write headers
		err = w.WriteHeaders(h)
		if err != nil {
			log.Printf("Error writing headers: %v", err)
			return
		}

		// Write body
		_, err = w.WriteBody([]byte(html))
		if err != nil {
			log.Printf("Error writing body: %v", err)
		}
	}

	srv, err := server.Serve(8080, handler)
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	fmt.Println("Server running on :8080")

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\nShutting down server...")
	srv.Close()
}
