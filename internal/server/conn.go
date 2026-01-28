package server

import (
	"io"
	"time"

	"github.com/Brownie44l1/http-1/internal/request"
	"github.com/Brownie44l1/http-1/internal/response"
	net "github.com/Brownie44l1/socket-wrapper"
)

// handleConnection processes a single TCP connection
func handleConnection(conn net.Conn, handler Handler, config *Config, metrics *Metrics, logger Logger, shuttingDown bool) {
	defer conn.Close()

	// ✅ Issue #4: Set initial read deadline BEFORE parsing
	if config.ReadTimeout > 0 {
		if err := conn.SetReadDeadline(time.Now().Add(config.ReadTimeout)); err != nil {
			logger.Error("failed to set read deadline", Field{"error", err})
			return
		}
	}

	// Process requests in a loop (for keep-alive)
	requestCount := 0
	maxRequestsPerConn := 1000 // Prevent infinite keep-alive

	for requestCount < maxRequestsPerConn {
		requestCount++

		// ✅ Issue #4: Reset deadline before each request
		if config.ReadTimeout > 0 {
			if err := conn.SetReadDeadline(time.Now().Add(config.ReadTimeout)); err != nil {
				logger.Debug("failed to reset read deadline", Field{"error", err})
				return
			}
		}

		// ✅ Issue #3: Pass config for size limits
		req, err := request.RequestFromReaderWithConfig(conn, config.MaxHeaderBytes, config.MaxRequestBodySize)
		if err != nil {
			// EOF and connection closed errors are normal for keep-alive
			if err == io.EOF {
				// Client closed connection - this is normal
				return
			}

			// Check for timeout errors
			if timeoutErr, ok := err.(*net.TimeoutError); ok && timeoutErr.Timeout() {
				// Timeout is normal for idle connections
				return
			}

			// ✅ Issue #3: Check for size limit errors
			if err == request.ErrHeaderTooLarge ||
				err == request.ErrBodyTooLarge ||
				err == request.ErrRequestLineTooLarge {
				// Send 413 or 400 response
				w := response.NewWriter(conn)
				w.ErrorResponse(response.StatusRequestEntityTooLarge, "Request too large")
				return
			}

			// For other errors, log and try to send error response
			logger.Error("error parsing request",
				Field{"error", err},
				Field{"request_count", requestCount},
			)

			w := response.NewWriter(conn)
			if err := w.ErrorResponse(response.StatusBadRequest, "Invalid request"); err != nil {
				logger.Debug("failed to send error response", Field{"error", err})
			}
			return
		}

		// ✅ Issue #11: Handle Expect: 100-continue
		if expectHeader, ok := req.Headers.Get("expect"); ok {
			if expectHeader == "100-continue" {
				w := response.NewWriter(conn)
				if err := w.ContinueResponse(); err != nil {
					logger.Error("failed to send 100-continue", Field{"error", err})
					return
				}
				w.Flush()
			}
		}

		// Create response writer
		w := response.NewWriter(conn)

		// ✅ Issue #6: Create context with connection for hijacking
		ctx := NewContext(req, w, conn)

		// ✅ Issue #18: Add Connection: close header if shutting down
		if shuttingDown {
			w.Headers().Set("Connection", "close")
		}

		// ✅ Issue #4: Set write deadline
		if config.WriteTimeout > 0 {
			if err := conn.SetWriteDeadline(time.Now().Add(config.WriteTimeout)); err != nil {
				logger.Debug("failed to set write deadline", Field{"error", err})
				return
			}
		}

		// Call the handler
		start := time.Now()
		handler.ServeHTTP(ctx)
		duration := time.Since(start)

		// ✅ Issue #16: Record metrics
		if metrics != nil {
			metrics.RecordRequest(int(w.StatusCode()), duration)
		}

		// ✅ Issue #6: Check if connection was hijacked
		if ctx.IsHijacked() {
			// Handler took over the connection, we're done
			return
		}

		// Check if we should keep the connection alive
		if !shouldKeepAlive(req, w, shuttingDown) {
			return
		}

		// ✅ Issue #4: Reset read deadline for next request with idle timeout
		if config.IdleTimeout > 0 {
			if err := conn.SetReadDeadline(time.Now().Add(config.IdleTimeout)); err != nil {
				logger.Debug("failed to set idle deadline", Field{"error", err})
				return
			}
		}
	}

	// Reached max requests per connection
	logger.Debug("max requests per connection reached",
		Field{"count", requestCount},
	)
}

// shouldKeepAlive determines if the connection should be kept alive
func shouldKeepAlive(req *request.Request, w *response.Writer, shuttingDown bool) bool {
	// ✅ Issue #18: Never keep alive if shutting down
	if shuttingDown {
		return false
	}

	// Don't keep alive if there was a write error
	if w.HadError() {
		return false
	}

	// Check response Connection header
	if connHeader, ok := w.Headers().Get("connection"); ok {
		if connHeader == "close" {
			return false
		}
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
