package server

import (
	"fmt"
	"log"
	"net"
	"sync/atomic"
	"time"

	"http1.1/internal/request"
	"http1.1/internal/response"
)

type Server struct {
	handler     Handler
	listener    net.Listener
	closed      atomic.Bool
	ReadTimeout time.Duration
}

type Handler func(w *response.Writer, r *request.Request)

func Serve(port uint16, handler Handler) (*Server, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, err
	}

	s := &Server{
		handler:     handler,
		listener:    listener,
		ReadTimeout: 30 * time.Second,
	}

	go s.listen()
	return s, nil
}

func (s *Server) listen() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			if s.closed.Load() {
				return
			}
			log.Printf("Error accepting connection: %v\n", err)
			continue
		}

		// Just spawn the connection handler
		go s.serveConn(conn)
	}
}

func (s *Server) Close() error {
	s.closed.Store(true)
	return s.listener.Close()
}
