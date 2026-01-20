package server

import (
	"github.com/Brownie44l1/http1.1/internal/request"
	"github.com/Brownie44l1/http1.1/internal/response"
)

// shouldCloseConnection determines if connection should be closed after this request
func shouldCloseConnection(req *request.Request, w *response.Writer) bool {
	// If response had errors, close the connection
	if w.HadError() {
		return true
	}

	// HTTP/1.0 closes by default unless "Connection: keep-alive"
	if req.IsHTTP10() {
		return !req.WantsKeepAlive()
	}

	// HTTP/1.1 keeps alive by default unless "Connection: close"
	if req.WantsClose() {
		return true
	}

	// Close if response didn't have proper framing
	// (no Content-Length and not chunked = can't tell where response ends)
	if !w.HasContentLength() && !w.IsChunked() {
		return true
	}

	// If it's a HEAD request and has a body, that's an error
	if req.Method == "HEAD" && (w.HasContentLength() || w.IsChunked()) {
		// HEAD responses must not have a body, but having Content-Length is ok
		// Only close if there was actual body data written
	}

	// Keep connection alive
	return false
}