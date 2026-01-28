//go:build linux
// +build linux

package server

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	net "github.com/Brownie44l1/socket-wrapper"
)

// Config holds server configuration
type Config struct {
	Port               int           // Port to listen on
	ReadTimeout        time.Duration // Max time to read request
	WriteTimeout       time.Duration // Max time to write response
	IdleTimeout        time.Duration // Max time for keep-alive
	MaxHeaderBytes     int           // Max header size
	MaxRequestBodySize int64         // Max request body size
	MaxConns           int           // Max concurrent connections

	// Network-level config
	ReusePort   bool          // SO_REUSEPORT for multi-process
	DeferAccept time.Duration // TCP_DEFER_ACCEPT optimization

	// Request limits (DoS protection)
	MaxRequestsPerConn int           // Max requests per connection
	RequestTimeout     time.Duration // Total time for request including body
}

// DefaultConfig returns sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Port:               8080,
		ReadTimeout:        15 * time.Second,
		WriteTimeout:       15 * time.Second,
		IdleTimeout:        60 * time.Second,
		MaxHeaderBytes:     1 << 20,  // 1MB
		MaxRequestBodySize: 10 << 20, // 10MB
		MaxConns:           10000,
		ReusePort:          false,
		DeferAccept:        1 * time.Second, // Optimize for HTTP
		MaxRequestsPerConn: 1000,             // Prevent infinite keep-alive
		RequestTimeout:     30 * time.Second,
	}
}

type Server struct {
	config   *Config
	handler  Handler
	listener net.Listener
	metrics *Metrics
	logger  Logger
	mu       sync.RWMutex
	shutdown bool
	wg       sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc
	middlewares []Middleware
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

// Middleware wraps a handler
type Middleware func(Handler) Handler

// New creates a new server with the given configuration and handler
func New(config *Config, handler Handler) *Server {
	if config == nil {
		config = DefaultConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Create a simple default logger if none provided
	defaultLogger := &simpleLogger{}

	return &Server{
		config:      config,
		handler:     handler,
		ctx:         ctx,
		cancel:      cancel,
		middlewares: make([]Middleware, 0),
		metrics:     NewMetrics(),
		logger:      defaultLogger,
	}
}

// simpleLogger is a basic logger implementation
type simpleLogger struct{}

func (l *simpleLogger) Debug(msg string, fields ...Field) {}
func (l *simpleLogger) Info(msg string, fields ...Field)  {}
func (l *simpleLogger) Error(msg string, fields ...Field) {
	log.Printf("ERROR: %s", msg)
}
func (l *simpleLogger) Warn(msg string, fields ...Field) {
	log.Printf("WARN: %s", msg)
}

// SetLogger sets a custom logger for the server
func (s *Server) SetLogger(logger Logger) {
	s.logger = logger
}

// Use adds a middleware to the server
func (s *Server) Use(mw Middleware) {
	s.middlewares = append(s.middlewares, mw)
}

// buildHandler applies all middlewares to the handler
func (s *Server) buildHandler() Handler {
	handler := s.handler

	// Apply middlewares in reverse order
	for i := len(s.middlewares) - 1; i >= 0; i-- {
		handler = s.middlewares[i](handler)
	}

	return handler
}

// ListenAndServe starts the server using custom net library
func (s *Server) ListenAndServe() error {
	// Create network configuration using fluent API
	netConfig := net.DefaultConfig().
		WithPort(s.config.Port).
		WithReusePort(s.config.ReusePort).
		WithMaxConns(s.config.MaxConns).
		WithDeferAccept(s.config.DeferAccept)

	// Create listener using custom net library
	listener, err := net.Listen(netConfig)
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %w", s.config.Port, err)
	}

	s.listener = listener
	log.Printf("HTTP server listening on %s", listener.Addr())

	return s.serve()
}

// serve accepts connections and handles them
func (s *Server) serve() error {
	defer s.listener.Close()

	// Build final handler with middlewares
	finalHandler := s.buildHandler()

	for {
		// Check if we're shutting down
		select {
		case <-s.ctx.Done():
			log.Println("Server stopping accept loop")
			return nil
		default:
		}

		// Accept new connection with context
		conn, err := s.listener.AcceptContext(s.ctx)
		if err != nil {
			// Check for shutdown errors
			if s.isShuttingDown() {
				return nil
			}
			log.Printf("Accept error: %v", err)
			continue
		}

		// Track metrics
		s.metrics.ActiveConnections.Add(1)

		// Handle connection in goroutine
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			defer s.metrics.ActiveConnections.Add(-1)

			s.handleConn(conn, finalHandler)
		}()
	}
}

// isShuttingDown checks if the server is shutting down
func (s *Server) isShuttingDown() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.shutdown
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

	log.Println("HTTP server shutting down...")

	// Cancel context to signal all goroutines
	s.cancel()

	// Use the listener's graceful shutdown
	if s.listener != nil {
		// Extract timeout from context
		deadline, ok := ctx.Deadline()
		var timeout time.Duration
		if ok {
			timeout = time.Until(deadline)
		} else {
			timeout = 30 * time.Second
		}

		// Let listener handle graceful shutdown
		if err := s.listener.GracefulShutdown(timeout); err != nil {
			log.Printf("Graceful shutdown warning: %v", err)
		}
	}

	// Wait for existing handlers to finish
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("All HTTP connections closed")
		return nil
	case <-ctx.Done():
		return fmt.Errorf("shutdown timeout: %w", ctx.Err())
	}
}

// handleConn processes a single connection
func (s *Server) handleConn(conn net.Conn, handler Handler) {
	// Check if we're shutting down
	s.mu.RLock()
	shuttingDown := s.shutdown
	s.mu.RUnlock()

	handleConnection(conn, handler, s.config, s.metrics, s.logger, shuttingDown)
}

// Metrics returns server metrics
func (s *Server) Metrics() *Metrics {
	return s.metrics
}

// Stats returns listener statistics (if available)
func (s *Server) Stats() interface{} {
	if s.listener != nil {
		return s.listener.Stats()
	}
	return nil
}