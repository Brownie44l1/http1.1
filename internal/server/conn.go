package server

import (
	"log"
	"net"
	"time"

	"http1.1/internal/request"
	"http1.1/internal/response"
)

// serveConn handles all requests on a single connection
func (s *Server) serveConn(conn net.Conn) {
	defer conn.Close()

	for {
		// Set read deadline for this request
		conn.SetReadDeadline(time.Now().Add(s.ReadTimeout))
		
		req, err := request.RequestFromReader(conn)
		if err != nil {
			// Connection error - send 400 if possible, then close
			s.handleBadRequest(conn, err)
			return
		}

		// Create response writer for this request
		w := response.NewWriter(conn)

		// Handle the request with panic recovery
		s.handleRequest(w, req)

		// Decide if we should keep connection alive
		if shouldCloseConnection(req, w) {
			return
		}

		// Clear deadline for next request
		conn.SetReadDeadline(time.Time{})
	}
}

// handleRequest wraps handler call with panic recovery
func (s *Server) handleRequest(w *response.Writer, req *request.Request) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Handler panic: %v", r)
			s.handle500(w)
		}
	}()
	
	s.handler(w, req)
}

// handleBadRequest sends 400 response
func (s *Server) handleBadRequest(conn net.Conn, err error) {
	w := response.NewWriter(conn)
	// Use the new ErrorResponse helper
	w.ErrorResponse(response.StatusBadRequest, err.Error())
}

// handle500 sends 500 response
func (s *Server) handle500(w *response.Writer) {
	// Only send 500 if we haven't started writing response yet
	// (can't send status line twice)
	w.ErrorResponse(response.StatusInternalServerError, "Internal Server Error")
}