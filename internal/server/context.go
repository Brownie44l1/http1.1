package server

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Brownie44l1/http-1/internal/request"
	"github.com/Brownie44l1/http-1/internal/response"
	net "github.com/Brownie44l1/socket-wrapper"
)

// Context provides a convenient interface for handling requests and responses
type Context struct {
	Request   *request.Request
	Response  *response.Writer
	Params    map[string]string // Path parameters (e.g., /users/:id)
	RequestID string            // ✅ Issue #8: Request ID for tracing

	// ✅ Issue #6: For connection hijacking (WebSockets)
	conn     net.Conn
	hijacked bool
}

// NewContext creates a new context
func NewContext(req *request.Request, resp *response.Writer, conn net.Conn) *Context {
	// ✅ Issue #8: Extract or generate request ID
	requestID, _ := req.Headers.Get("x-request-id")
	if requestID == "" {
		requestID = generateRequestID()
	}

	return &Context{
		Request:   req,
		Response:  resp,
		Params:    make(map[string]string),
		RequestID: requestID,
		conn:      conn,
		hijacked:  false,
	}
}

// Method returns the HTTP method
func (c *Context) Method() string {
	return c.Request.Method
}

// Path returns the request path
func (c *Context) Path() string {
	return c.Request.Path
}

// Header gets a request header value
func (c *Context) Header(key string) string {
	val, _ := c.Request.Headers.Get(key)
	return val
}

// SetParams sets path parameters (called by router)
func (c *Context) SetParams(params map[string]string) {
	c.Params = params
}

// Param gets a path parameter by name
func (c *Context) Param(name string) string {
	return c.Params[name]
}

// Query gets a query parameter (basic implementation)
func (c *Context) Query(key string) string {
	// Simple implementation - parse query from path
	path := c.Request.Path
	if idx := strings.Index(path, "?"); idx != -1 {
		query := path[idx+1:]
		pairs := strings.Split(query, "&")
		for _, pair := range pairs {
			kv := strings.SplitN(pair, "=", 2)
			if len(kv) == 2 && kv[0] == key {
				return kv[1]
			}
		}
	}
	return ""
}

// Body returns the request body as bytes
func (c *Context) Body() []byte {
	return c.Request.Body
}

// BodyString returns the request body as a string
func (c *Context) BodyString() string {
	return string(c.Request.Body)
}

// Response helpers

// Text sends a plain text response
func (c *Context) Text(code response.StatusCode, text string) error {
	return c.Response.TextResponse(code, text)
}

// HTML sends an HTML response
func (c *Context) HTML(code response.StatusCode, html string) error {
	return c.Response.HTMLResponse(code, html)
}

// JSON sends a JSON response
func (c *Context) JSON(code response.StatusCode, json string) error {
	return c.Response.JSONResponse(code, json)
}

// Error sends an error response
func (c *Context) Error(code response.StatusCode, message string) error {
	return c.Response.ErrorResponse(code, message)
}

// Redirect sends a redirect response
func (c *Context) Redirect(code response.StatusCode, location string) error {
	return c.Response.RedirectResponse(code, location)
}

// NoContent sends a 204 No Content response
func (c *Context) NoContent() error {
	return c.Response.NoContentResponse()
}

// Status sends just a status code with no body
func (c *Context) Status(code response.StatusCode) error {
	return c.Response.WriteStatusLine(code)
}

// String is a helper for formatting responses
func (c *Context) String(code response.StatusCode, format string, values ...interface{}) error {
	text := fmt.Sprintf(format, values...)
	return c.Text(code, text)
}

// ✅ Issue #6: Hijack takes over the underlying connection (for WebSockets)
func (c *Context) Hijack() (net.Conn, error) {
	if c.hijacked {
		return nil, errors.New("connection already hijacked")
	}

	if c.conn == nil {
		return nil, errors.New("no underlying connection")
	}

	c.hijacked = true
	return c.conn, nil
}

// IsHijacked returns true if the connection has been hijacked
func (c *Context) IsHijacked() bool {
	return c.hijacked
}

// IsWebSocketUpgrade checks if this is a WebSocket upgrade request
func (c *Context) IsWebSocketUpgrade() bool {
	upgrade := strings.ToLower(c.Header("Upgrade"))
	connection := strings.ToLower(c.Header("Connection"))

	return upgrade == "websocket" && strings.Contains(connection, "upgrade")
}

// GetClientIP returns the client IP address
func (c *Context) GetClientIP() string {
	// Check X-Forwarded-For header first
	if xff := c.Header("X-Forwarded-For"); xff != "" {
		// Take first IP
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}

	// Check X-Real-IP header
	if xri := c.Header("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	// Fall back to remote address from connection
	if c.conn != nil {
		addr := c.conn.RemoteAddr()
		// Strip port
		if idx := strings.LastIndex(addr, ":"); idx != -1 {
			return addr[:idx]
		}
		return addr
	}

	return ""
}

// generateRequestID generates a unique request ID
func generateRequestID() string {
	// Simple implementation - use timestamp + random
	// For production, use UUID or similar
	return fmt.Sprintf("req-%d", time.Now().UnixNano())
}
