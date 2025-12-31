package server

import (
	"fmt"
	"log"
	"net"
	"sync/atomic"

	"http1.1/internal/request"
	"http1.1/internal/response"
)

type Server struct {
	handler  Handler
	listener net.Listener
	closed   atomic.Bool
}

type Handler func(w *response.Writer, r *request.Request)

type HandlerError struct {
	StatusCode response.StatusCode
	Message    string
}

func Serve(port uint16, handler Handler) (*Server, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, err
	}

	s := &Server{
		handler:  handler,
		listener: listener,
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
		go s.handle(conn)
	}
}

func (s *Server) handle(conn net.Conn) {
	defer conn.Close()
	req, err := request.RequestFromReader(conn)

	if err != nil {
		w := response.NewWriter(conn)
		w.WriteStatusLine(response.StatusBadRequest)
		headers := response.GetDefaultHeaders(len(err.Error()))
		w.WriteHeaders(headers)
		w.WriteBody([]byte(err.Error()))
		return
	}

	w := response.NewWriter(conn)
	s.handler(w, req)
}

func (s *Server) Close() error {
	s.closed.Store(true)
	return s.listener.Close()
}
