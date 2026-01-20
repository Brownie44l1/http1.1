package server

import (
	"fmt"
	"strings"

	"github.com/Brownie44l1/http1.1/internal/request"
	"github.com/Brownie44l1/http1.1/internal/response"
)

// Context provides a convenient interface for handling requests and responses
type Context struct {
	Request  *request.Request
	Response *response.Writer
	Params   map[string]string // Path parameters (e.g., /users/:id)
}

// NewContext creates a new context
func NewContext(req *request.Request, resp *response.Writer) *Context {
	return &Context{
		Request:  req,
		Response: resp,
		Params:   make(map[string]string),
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

// Param gets a path parameter by name
func (c *Context) Param(name string) string {
	return c.Params[name]
}

// Query gets a query parameter (basic implementation)
// For full query parsing, you'd need to parse the URL query string
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