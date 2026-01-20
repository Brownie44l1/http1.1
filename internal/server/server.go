package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

// Config holds server configuration
type Config struct {
	Addr              string        // Address to listen on (e.g., ":8080")
	ReadTimeout       time.Duration // Max time to read request
	WriteTimeout      time.Duration // Max time to write response
	IdleTimeout       time.Duration // Max time for keep-alive
	MaxHeaderBytes    int           // Max header size
	MaxRequestBodySize int64        // Max request body size
}

// DefaultConfig returns sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Addr:              ":8080",
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1MB
		MaxRequestBodySize: 10 << 20, // 10MB
	}
}

// Server represents an HTTP/1.1 server
type Server struct {
	config   *Config
	handler  Handler
	listener net.Listener
	
	mu       sync.Mutex
	shutdown bool
	wg       sync.WaitGroup
	
	ctx    context.Context
	cancel context.CancelFunc
}

// Handler processes HTTP requests
type Handler interface {
	ServeHTTP(ctx *Context)
}

// HandlerFunc type adapter
type HandlerFunc func(ctx *Context)

func (f HandlerFunc) ServeHTTP(ctx *Context) {
	f(ctx)
}

// New creates a new server with the given configuration and handler
func New(config *Config, handler Handler) *Server {
	if config == nil {
		config = DefaultConfig()
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	return &Server{
		config:  config,
		handler: handler,
		ctx:     ctx,
		cancel:  cancel,
	}
}

// ListenAndServe starts the server
func (s *Server) ListenAndServe() error {
	addr := s.config.Addr
	if addr == "" {
		addr = ":8080"
	}
	
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}
	
	s.listener = listener
	log.Printf("Server listening on %s", addr)
	
	return s.serve()
}

// serve accepts connections and handles them
func (s *Server) serve() error {
	defer s.listener.Close()
	
	for {
		// Check if we're shutting down
		s.mu.Lock()
		if s.shutdown {
			s.mu.Unlock()
			break
		}
		s.mu.Unlock()
		
		// Accept new connection
		conn, err := s.listener.Accept()
		if err != nil {
			// Check if this is due to shutdown
			select {
			case <-s.ctx.Done():
				return nil
			default:
				log.Printf("Accept error: %v", err)
				continue
			}
		}
		
		// Handle connection in goroutine
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.handleConn(conn)
		}()
	}
	
	// Wait for all connections to finish
	s.wg.Wait()
	return nil
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	if s.shutdown {
		s.mu.Unlock()
		return fmt.Errorf("server already shutdown")
	}
	s.shutdown = true
	s.mu.Unlock()
	
	log.Println("Server shutting down...")
	
	// Cancel context to signal all goroutines
	s.cancel()
	
	// Close listener to stop accepting new connections
	if s.listener != nil {
		s.listener.Close()
	}
	
	// Wait for existing connections with timeout
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		log.Println("All connections closed")
		return nil
	case <-ctx.Done():
		return fmt.Errorf("shutdown timeout: %w", ctx.Err())
	}
}

// handleConn processes a single connection (implemented in conn.go)
func (s *Server) handleConn(conn net.Conn) {
	handleConnection(conn, s.handler, s.config)
}