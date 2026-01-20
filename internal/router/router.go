package router

import (
	"strings"
)

// Handler is a function that handles HTTP requests
type Handler func(ctx interface{})

// Route represents a single route
type Route struct {
	Method  string
	Path    string
	Handler Handler
	Params  []string // Parameter names (e.g., ["id", "name"])
}

// Router handles HTTP routing
type Router struct {
	routes []*Route
}

// New creates a new router
func New() *Router {
	return &Router{
		routes: make([]*Route, 0),
	}
}

// Handle registers a new route
func (r *Router) Handle(method, path string, handler Handler) {
	// Extract parameter names from path
	params := extractParams(path)
	
	route := &Route{
		Method:  method,
		Path:    path,
		Handler: handler,
		Params:  params,
	}
	
	r.routes = append(r.routes, route)
}

// GET is a shortcut for Handle("GET", ...)
func (r *Router) GET(path string, handler Handler) {
	r.Handle("GET", path, handler)
}

// POST is a shortcut for Handle("POST", ...)
func (r *Router) POST(path string, handler Handler) {
	r.Handle("POST", path, handler)
}

// PUT is a shortcut for Handle("PUT", ...)
func (r *Router) PUT(path string, handler Handler) {
	r.Handle("PUT", path, handler)
}

// DELETE is a shortcut for Handle("DELETE", ...)
func (r *Router) DELETE(path string, handler Handler) {
	r.Handle("DELETE", path, handler)
}

// PATCH is a shortcut for Handle("PATCH", ...)
func (r *Router) PATCH(path string, handler Handler) {
	r.Handle("PATCH", path, handler)
}

// Match finds a route that matches the given method and path
func (r *Router) Match(method, path string) (*Route, map[string]string) {
	// Remove query string if present
	if idx := strings.Index(path, "?"); idx != -1 {
		path = path[:idx]
	}
	
	for _, route := range r.routes {
		// Check method
		if route.Method != method {
			continue
		}
		
		// Check if path matches (with parameter support)
		if params := matchPath(route.Path, path); params != nil {
			return route, params
		}
	}
	
	return nil, nil
}

// ServeHTTP implements the server.Handler interface
func (r *Router) ServeHTTP(ctx interface{}) {
	// Type assert to get the actual context
	// This is a simple implementation - in production you'd want better type safety
	type contextInterface interface {
		Method() string
		Path() string
		Error(code int, msg string) error
	}
	
	c, ok := ctx.(contextInterface)
	if !ok {
		return
	}
	
	route, params := r.Match(c.Method(), c.Path())
	if route == nil {
		c.Error(404, "Not Found")
		return
	}
	
	// Set params on context if it supports it
	type paramSetter interface {
		SetParams(map[string]string)
	}
	if ps, ok := ctx.(paramSetter); ok {
		ps.SetParams(params)
	}
	
	// Call the handler
	route.Handler(ctx)
}

// extractParams extracts parameter names from a path pattern
// Example: "/users/:id/posts/:postId" -> ["id", "postId"]
func extractParams(path string) []string {
	parts := strings.Split(path, "/")
	params := make([]string, 0)
	
	for _, part := range parts {
		if strings.HasPrefix(part, ":") {
			params = append(params, part[1:])
		}
	}
	
	return params
}

// matchPath checks if a request path matches a route pattern
// Returns parameter values if match, nil otherwise
func matchPath(pattern, path string) map[string]string {
	patternParts := strings.Split(pattern, "/")
	pathParts := strings.Split(path, "/")
	
	// Must have same number of parts
	if len(patternParts) != len(pathParts) {
		return nil
	}
	
	params := make(map[string]string)
	
	for i := 0; i < len(patternParts); i++ {
		patternPart := patternParts[i]
		pathPart := pathParts[i]
		
		if strings.HasPrefix(patternPart, ":") {
			// This is a parameter
			paramName := patternPart[1:]
			params[paramName] = pathPart
		} else if patternPart != pathPart {
			// Static parts must match exactly
			return nil
		}
	}
	
	return params
}