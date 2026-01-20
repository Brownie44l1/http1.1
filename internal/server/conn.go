package server

import (
	"io"
	"log"
	"net"
	"time"

	"github.com/Brownie44l1/http1.1/internal/request"
	"github.com/Brownie44l1/http1.1/internal/response"
)

// handleConnection processes a single TCP connection
func handleConnection(conn net.Conn, handler Handler, config *Config) {
	defer conn.Close()

	// Set initial read deadline
	if config.ReadTimeout > 0 {
		conn.SetReadDeadline(time.Now().Add(config.ReadTimeout))
	}

	// Process requests in a loop (for keep-alive)
	for {
		// Parse the request
		req, err := request.RequestFromReader(conn)
		if err != nil {
			// EOF and connection closed errors are normal for keep-alive
			if err == io.EOF {
				// Client closed connection - this is normal
				return
			}

			// Check for timeout or other connection errors
			if netErr, ok := err.(net.Error); ok {
				if netErr.Timeout() {
					// Timeout is normal for idle connections
					return
				}
			}

			// For other errors, log and try to send error response
			log.Printf("Error parsing request: %v", err)
			w := response.NewWriter(conn)
			w.ErrorResponse(response.StatusBadRequest, "Invalid request")
			return
		}

		// Create response writer
		w := response.NewWriter(conn)

		// Create context
		ctx := NewContext(req, w)

		// Set write deadline
		if config.WriteTimeout > 0 {
			conn.SetWriteDeadline(time.Now().Add(config.WriteTimeout))
		}

		// Call the handler
		handler.ServeHTTP(ctx)

		// Check if we should keep the connection alive
		if !shouldKeepAlive(req, w) {
			return
		}

		// Reset read deadline for next request
		if config.IdleTimeout > 0 {
			conn.SetReadDeadline(time.Now().Add(config.IdleTimeout))
		}
	}
}

// shouldKeepAlive determines if the connection should be kept alive
func shouldKeepAlive(req *request.Request, w *response.Writer) bool {
	// Don't keep alive if there was a write error
	if w.HadError() {
		return false
	}

	// HTTP/1.0 closes by default unless "Connection: keep-alive"
	if req.IsHTTP10() {
		conn, ok := req.Headers.Get("connection")
		return ok && conn == "keep-alive"
	}

	// HTTP/1.1 keeps alive by default unless "Connection: close"
	conn, ok := req.Headers.Get("connection")
	return !ok || conn != "close"
}
