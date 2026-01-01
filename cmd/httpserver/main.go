package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"http1.1/internal/headers"
	"http1.1/internal/request"
	"http1.1/internal/response"
	"http1.1/internal/server"
)

func main() {
	handler := func(w *response.Writer, r *request.Request) {
		// Check if this is a proxy request to httpbin
		if strings.HasPrefix(r.RequestLine.RequestTarget, "/httpbin/") {
			proxyHandler(w, r)
			return
		}

		// Original handler logic
		var statusCode response.StatusCode
		var html string

		switch r.RequestLine.RequestTarget {
		case "/video":
			videoData, err := os.ReadFile("assets/vim.mp4")
			if err != nil {
				log.Printf("Error reading video file: %v", err)
				statusCode = response.StatusInternalServerError
				html = `
				<html>
				<head>
					<title>500 Internal Server Error</title>
				</head>
				<body>
					<h1>Internal Server Error</h1>
					<p>Could not load video file.</p>
				</body>
				</html>`
			} else {
				statusCode = response.StatusOk

				// Write status line
				err = w.WriteStatusLine(statusCode)
				if err != nil {
					log.Printf("Error writing status: %v", err)
					return
				}

				// Create headers for video
				h := headers.Headers{
					Header: map[string]string{
						"Content-Length": fmt.Sprintf("%d", len(videoData)),
						"Connection":     "close",
						"Content-Type":   "video/mp4",
					},
				}

				// Write headers
				err = w.WriteHeaders(h)
				if err != nil {
					log.Printf("Error writing headers: %v", err)
					return
				}

				// Write video data
				_, err = w.WriteBody(videoData)
				if err != nil {
					log.Printf("Error writing body: %v", err)
				}
				return
			}

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

func proxyHandler(w *response.Writer, r *request.Request) {
	// Map /httpbin/x to https://httpbin.org/x
	path := strings.TrimPrefix(r.RequestLine.RequestTarget, "/httpbin")
	url := "https://httpbin.org" + path

	// Make request to httpbin.org
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Error proxying request: %v", err)
		w.WriteStatusLine(response.StatusInternalServerError)
		h := headers.Headers{
			Header: map[string]string{
				"Content-Type": "text/plain",
				"Connection":   "close",
			},
		}
		w.WriteHeaders(h)
		w.WriteBody([]byte("Error proxying request"))
		return
	}
	defer resp.Body.Close()

	// Write status line
	statusCode := response.StatusCode(resp.StatusCode)
	err = w.WriteStatusLine(statusCode)
	if err != nil {
		log.Printf("Error writing status: %v", err)
		return
	}

	// Prepare headers for chunked response
	h := headers.Headers{
		Header: make(map[string]string),
	}

	// Copy relevant headers from httpbin response, but skip Content-Length
	for key, values := range resp.Header {
		if key != "Content-Length" && len(values) > 0 {
			h.Header[key] = values[0]
		}
	}

	// Add chunked encoding and trailer announcement
	h.Header["Transfer-Encoding"] = "chunked"
	h.Header["Trailer"] = "X-Content-SHA256, X-Content-Length"

	// Write headers
	err = w.WriteHeaders(h)
	if err != nil {
		log.Printf("Error writing headers: %v", err)
		return
	}

	// Read and forward response in chunks
	buffer := make([]byte, 1024)
	var fullBody []byte
	hasher := sha256.New()

	for {
		n, err := resp.Body.Read(buffer)
		if n > 0 {
			chunk := buffer[:n]
			fmt.Printf("Read %d bytes\n", n)

			// Track full body for hash and length
			fullBody = append(fullBody, chunk...)
			hasher.Write(chunk)

			// Write chunk to client
			_, writeErr := w.WriteChunkedBody(chunk)
			if writeErr != nil {
				log.Printf("Error writing chunk: %v", writeErr)
				return
			}
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("Error reading response body: %v", err)
			break
		}
	}

	// Write final chunk (zero-sized)
	_, err = w.WriteChunkedBodyDone()
	if err != nil {
		log.Printf("Error writing final chunk: %v", err)
		return
	}

	// Calculate and write trailers
	hash := hasher.Sum(nil)
	trailers := headers.Headers{
		Header: map[string]string{
			"X-Content-SHA256": fmt.Sprintf("%x", hash),
			"X-Content-Length": fmt.Sprintf("%d", len(fullBody)),
		},
	}

	err = w.WriteTrailers(trailers)
	if err != nil {
		log.Printf("Error writing trailers: %v", err)
	}
}
