package server

import (
	"fmt"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/Brownie44l1/http-1/internal/response"
)

// ✅ Issue #7: Middleware Support
// Note: Middleware type is declared in server.go to avoid duplication

// LoggingMiddleware logs all requests
func LoggingMiddleware(logger Logger) Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(ctx *Context) {
			start := time.Now()

			// Call next handler
			next.ServeHTTP(ctx)

			duration := time.Since(start)

			// ✅ Issue #22: Don't log sensitive headers
			logger.Info("request handled",
				Field{"method", ctx.Method()},
				Field{"path", ctx.Path()},
				Field{"status", ctx.Response.StatusCode()},
				Field{"duration_ms", duration.Milliseconds()},
				Field{"request_id", ctx.RequestID},
				Field{"client_ip", ctx.GetClientIP()},
			)
		})
	}
}

// RecoveryMiddleware recovers from panics
func RecoveryMiddleware(logger Logger) Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(ctx *Context) {
			defer func() {
				if err := recover(); err != nil {
					logger.Error("panic recovered",
						Field{"error", err},
						Field{"stack", string(debug.Stack())},
						Field{"request_id", ctx.RequestID},
						Field{"path", ctx.Path()},
					)

					ctx.Error(response.StatusInternalServerError, "Internal Server Error")
				}
			}()

			next.ServeHTTP(ctx)
		})
	}
}

// RateLimiter implements a simple token bucket rate limiter per IP
type RateLimiter struct {
	mu       sync.RWMutex
	buckets  map[string]*bucket
	rate     int           // requests per window
	window   time.Duration // time window
	cleanupInterval time.Duration
}

type bucket struct {
	tokens    int
	lastReset time.Time
}

// NewRateLimiter creates a new rate limiter
// rate: number of requests allowed per window
// window: time window (e.g., 1 minute)
func NewRateLimiter(rate int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		buckets:         make(map[string]*bucket),
		rate:            rate,
		window:          window,
		cleanupInterval: window * 2,
	}

	// Start cleanup goroutine to remove old entries
	go rl.cleanup()

	return rl
}

// Allow checks if a request from the given IP should be allowed
func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	b, exists := rl.buckets[ip]
	if !exists {
		rl.buckets[ip] = &bucket{
			tokens:    rl.rate - 1,
			lastReset: now,
		}
		return true
	}

	// Reset bucket if window has passed
	if now.Sub(b.lastReset) >= rl.window {
		b.tokens = rl.rate - 1
		b.lastReset = now
		return true
	}

	// Check if tokens available
	if b.tokens > 0 {
		b.tokens--
		return true
	}

	return false
}

// cleanup removes old bucket entries periodically
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.cleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, b := range rl.buckets {
			if now.Sub(b.lastReset) > rl.window*2 {
				delete(rl.buckets, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// ✅ Issue #20: Rate Limiting Middleware
func RateLimitMiddleware(limiter *RateLimiter) Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(ctx *Context) {
			ip := ctx.GetClientIP()

			if !limiter.Allow(ip) {
				ctx.Error(response.StatusTooManyRequests, "Rate limit exceeded")
				return
			}

			next.ServeHTTP(ctx)
		})
	}
}

// ✅ Issue #21: CORS Middleware
func CORSMiddleware(config CORSConfig) Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(ctx *Context) {
			origin := ctx.Header("Origin")

			// Check if origin is allowed
			if isAllowedOrigin(origin, config.AllowedOrigins) {
				// Set CORS headers
				ctx.Response.Headers().Set("Access-Control-Allow-Origin", origin)
				ctx.Response.Headers().Set("Access-Control-Allow-Methods", strings.Join(config.AllowedMethods, ", "))
				ctx.Response.Headers().Set("Access-Control-Allow-Headers", strings.Join(config.AllowedHeaders, ", "))

				if config.AllowCredentials {
					ctx.Response.Headers().Set("Access-Control-Allow-Credentials", "true")
				}

				if config.MaxAge > 0 {
					ctx.Response.Headers().Set("Access-Control-Max-Age", fmt.Sprintf("%d", int(config.MaxAge.Seconds())))
				}
			}

			// Handle preflight request
			if ctx.Method() == "OPTIONS" {
				ctx.NoContent()
				return
			}

			next.ServeHTTP(ctx)
		})
	}
}

// CORSConfig configures CORS middleware
type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	AllowCredentials bool
	MaxAge           time.Duration
}

// DefaultCORSConfig returns a permissive CORS config (for development)
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization", "X-Requested-With"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}
}

func isAllowedOrigin(origin string, allowed []string) bool {
	if len(allowed) == 0 {
		return false
	}

	for _, allowedOrigin := range allowed {
		if allowedOrigin == "*" {
			return true
		}
		if allowedOrigin == origin {
			return true
		}
	}

	return false
}

// TimeoutMiddleware enforces a timeout on request handling
func TimeoutMiddleware(timeout time.Duration) Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(ctx *Context) {
			done := make(chan struct{})

			go func() {
				next.ServeHTTP(ctx)
				close(done)
			}()

			select {
			case <-done:
				// Request completed normally
			case <-time.After(timeout):
				// Timeout - but we can't stop the handler
				// Just log it
				ctx.Error(response.StatusRequestTimeout, "Request timeout")
			}
		})
	}
}

// RequestIDMiddleware adds a unique ID to each request
func RequestIDMiddleware() Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(ctx *Context) {
			// Request ID is already set in context.go
			// Just add it to response headers for tracking
			ctx.Response.Headers().Set("X-Request-ID", ctx.RequestID)

			next.ServeHTTP(ctx)
		})
	}
}

// MetricsMiddleware records request metrics
func MetricsMiddleware(metrics *Metrics) Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(ctx *Context) {
			start := time.Now()

			next.ServeHTTP(ctx)

			duration := time.Since(start)
			statusCode := int(ctx.Response.StatusCode())

			metrics.RecordRequest(statusCode, duration)
		})
	}
}

// CompressionMiddleware adds gzip compression (placeholder)
// ✅ Issue #11: Compression support (simplified)
func CompressionMiddleware() Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(ctx *Context) {
			acceptEncoding := ctx.Header("Accept-Encoding")

			// Check if client accepts gzip
			if strings.Contains(acceptEncoding, "gzip") {
				// TODO: Wrap response writer with gzip writer
				// For now, just pass through
			}

			next.ServeHTTP(ctx)
		})
	}
}