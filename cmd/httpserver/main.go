package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"http1.1/internal/request"
	"http1.1/internal/response"
	"http1.1/internal/server"
)

func main() {
	// Start the server on port 8080
	srv, err := server.Serve(8080, handler)
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	log.Println("Server listening on http://localhost:8080")
	log.Println("Try:")
	log.Println("  curl http://localhost:8080/")
	log.Println("  curl http://localhost:8080/hello")
	log.Println("  curl -X POST http://localhost:8080/echo -d 'Hello, World!'")
	log.Println("  curl http://localhost:8080/chunked")

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("\nShutting down...")
	srv.Close()
}

// handler is the main request handler
func handler(w *response.Writer, r *request.Request) {
	log.Printf("%s %s %s", r.Method, r.Path, r.Version)

	// Route based on path
	switch r.Path {
	case "/":
		handleRoot(w, r)
	case "/hello":
		handleHello(w, r)
	case "/echo":
		handleEcho(w, r)
	case "/chunked":
		handleChunked(w, r)
	case "/json":
		handleJSON(w, r)
	default:
		handle404(w, r)
	}
}

// handleRoot serves the home page
func handleRoot(w *response.Writer, r *request.Request) {
	html := `<!DOCTYPE html>
<html>
<head><title>HTTP/1.1 Server</title></head>
<body>
	<h1>Welcome to the HTTP/1.1 Server!</h1>
	<p>Built from scratch to understand HTTP fundamentals.</p>
	<ul>
		<li><a href="/hello">Hello endpoint</a></li>
		<li><a href="/json">JSON endpoint</a></li>
		<li><a href="/chunked">Chunked response</a></li>
	</ul>
</body>
</html>`
	
	w.HTMLResponse(response.StatusOK, html)
}

// handleHello serves a simple greeting
func handleHello(w *response.Writer, r *request.Request) {
	w.TextResponse(response.StatusOK, "Hello, World!")
}

// handleEcho echoes back the request body
func handleEcho(w *response.Writer, r *request.Request) {
	if r.Method != "POST" {
		w.ErrorResponse(response.StatusBadRequest, "Only POST allowed")
		return
	}

	if len(r.Body) == 0 {
		w.TextResponse(response.StatusOK, "No body received")
		return
	}

	// Echo back what we received
	msg := fmt.Sprintf("You sent: %s", string(r.Body))
	w.TextResponse(response.StatusOK, msg)
}

// handleJSON serves JSON data
func handleJSON(w *response.Writer, r *request.Request) {
	json := `{
	"server": "HTTP/1.1 Server",
	"version": "1.0.0",
	"status": "running",
	"features": [
		"HTTP/1.1 persistent connections",
		"Chunked transfer encoding",
		"Request body parsing",
		"Multiple HTTP methods"
	]
}`
	
	w.JSONResponse(response.StatusOK, json)
}

// handleChunked demonstrates chunked transfer encoding
func handleChunked(w *response.Writer, r *request.Request) {
	// Start chunked response
	if err := w.ChunkedResponse(response.StatusOK, "text/plain"); err != nil {
		return
	}

	// Send multiple chunks
	chunks := []string{
		"This is the first chunk.\n",
		"Here comes the second chunk.\n",
		"And finally, the third chunk.\n",
	}

	for _, chunk := range chunks {
		if err := w.WriteChunk([]byte(chunk)); err != nil {
			return
		}
	}

	// Finish the chunked response
	w.FinishChunked()
}

// handle404 serves a 404 Not Found response
func handle404(w *response.Writer, r *request.Request) {
	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><title>404 Not Found</title></head>
<body>
	<h1>404 Not Found</h1>
	<p>The path <code>%s</code> was not found on this server.</p>
	<p><a href="/">Go back home</a></p>
</body>
</html>`, r.Path)
	
	w.HTMLResponse(response.StatusNotFound, html)
}