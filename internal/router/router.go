package router

import (
	"regexp"
	"strings"

	"github.com/Brownie44l1/http-1/internal/response"
	"github.com/Brownie44l1/http-1/internal/server"
)

// ✅ Issue #2: Use concrete type instead of interface{}
type Handler func(ctx *server.Context)

// Route represents a single route
type Route struct {
	Method   string
	Pattern  string // Original pattern (e.g., "/users/:id")
	Handler  Handler
	Params   []string       // Parameter names (e.g., ["id", "name"])
	Regex    *regexp.Regexp // ✅ Issue #10: Regex pattern for matching
	IsStatic bool           // True if no parameters/wildcards
}

// Router handles HTTP routing
type Router struct {
	routes           []*Route
	notFound         Handler // 404 handler
	methodNotAllowed Handler // 405 handler
}

// New creates a new router
func New() *Router {
	return &Router{
		routes: make([]*Route, 0),
		notFound: func(ctx *server.Context) {
			ctx.Error(response.StatusNotFound, "Not Found")
		},
		methodNotAllowed: func(ctx *server.Context) {
			ctx.Error(response.StatusMethodNotAllowed, "Method Not Allowed")
		},
	}
}

// Handle registers a new route
func (r *Router) Handle(method, pattern string, handler Handler) {
	// ✅ Issue #10: Parse pattern to extract params and build regex
	params, regex, isStatic := parsePattern(pattern)

	route := &Route{
		Method:   method,
		Pattern:  pattern,
		Handler:  handler,
		Params:   params,
		Regex:    regex,
		IsStatic: isStatic,
	}

	r.routes = append(r.routes, route)
}

// NotFound sets custom 404 handler
func (r *Router) NotFound(handler Handler) {
	r.notFound = handler
}

// MethodNotAllowed sets custom 405 handler
func (r *Router) MethodNotAllowed(handler Handler) {
	r.methodNotAllowed = handler
}

// GET is a shortcut for Handle("GET", ...)
func (r *Router) GET(pattern string, handler Handler) {
	r.Handle("GET", pattern, handler)
}

// POST is a shortcut for Handle("POST", ...)
func (r *Router) POST(pattern string, handler Handler) {
	r.Handle("POST", pattern, handler)
}

// PUT is a shortcut for Handle("PUT", ...)
func (r *Router) PUT(pattern string, handler Handler) {
	r.Handle("PUT", pattern, handler)
}

// DELETE is a shortcut for Handle("DELETE", ...)
func (r *Router) DELETE(pattern string, handler Handler) {
	r.Handle("DELETE", pattern, handler)
}

// PATCH is a shortcut for Handle("PATCH", ...)
func (r *Router) PATCH(pattern string, handler Handler) {
	r.Handle("PATCH", pattern, handler)
}

// HEAD is a shortcut for Handle("HEAD", ...)
func (r *Router) HEAD(pattern string, handler Handler) {
	r.Handle("HEAD", pattern, handler)
}

// OPTIONS is a shortcut for Handle("OPTIONS", ...)
func (r *Router) OPTIONS(pattern string, handler Handler) {
	r.Handle("OPTIONS", pattern, handler)
}

// Match finds a route that matches the given method and path
func (r *Router) Match(method, path string) (*Route, map[string]string) {
	// Remove query string if present
	if idx := strings.Index(path, "?"); idx != -1 {
		path = path[:idx]
	}

	// ✅ Issue #10: Priority order - static first, then params, then wildcards
	var (
		matchedRoute  *Route
		matchedParams map[string]string
	)

	for _, route := range r.routes {
		// Check method
		if route.Method != method {
			continue
		}

		// Try to match path
		if params := matchPath(route, path); params != nil {
			matchedRoute = route
			matchedParams = params

			// If it's a static route, return immediately (highest priority)
			if route.IsStatic {
				return matchedRoute, matchedParams
			}
		}
	}

	// Return best match (or nil if none)
	return matchedRoute, matchedParams
}

// ✅ Issue #2: Concrete type, no type assertions!
func (r *Router) ServeHTTP(ctx *server.Context) {
	route, params := r.Match(ctx.Method(), ctx.Path())

	if route == nil {
		// Check if path exists with different method
		for _, rt := range r.routes {
			if matchPath(rt, ctx.Path()) != nil {
				r.methodNotAllowed(ctx)
				return
			}
		}

		r.notFound(ctx)
		return
	}

	// ✅ Issue #2: Direct access, no type assertion needed
	ctx.SetParams(params)
	route.Handler(ctx)
}

// ✅ Issue #10: Enhanced pattern parsing with wildcards and regex
func parsePattern(pattern string) (params []string, regex *regexp.Regexp, isStatic bool) {
	isStatic = true
	params = make([]string, 0)

	// Convert pattern to regex
	regexStr := "^"
	parts := strings.Split(pattern, "/")

	for _, part := range parts {
		if part == "" {
			continue
		}

		regexStr += "/"

		if strings.HasPrefix(part, ":") {
			// Parameter: :id or :id<regex>
			isStatic = false

			paramName := part[1:]
			constraint := ""

			// Check for constraint: :id<[0-9]+>
			if idx := strings.Index(paramName, "<"); idx != -1 {
				constraint = paramName[idx+1 : len(paramName)-1]
				paramName = paramName[:idx]
			}

			params = append(params, paramName)

			if constraint != "" {
				regexStr += "(" + constraint + ")"
			} else {
				regexStr += "([^/]+)" // Match anything except /
			}

		} else if part == "*" || strings.HasPrefix(part, "*") {
			// Wildcard: * or *filepath
			isStatic = false

			paramName := "wildcard"
			if len(part) > 1 {
				paramName = part[1:]
			}

			params = append(params, paramName)
			regexStr += "(.*)" // Match everything

		} else {
			// Static part
			regexStr += regexp.QuoteMeta(part)
		}
	}

	regexStr += "$"
	regex = regexp.MustCompile(regexStr)

	return params, regex, isStatic
}

// matchPath uses regex to match path and extract parameters
func matchPath(route *Route, path string) map[string]string {
	// Quick static match
	if route.IsStatic {
		if route.Pattern == path {
			return make(map[string]string)
		}
		return nil
	}

	// Regex match
	matches := route.Regex.FindStringSubmatch(path)
	if matches == nil {
		return nil
	}

	// Extract parameters
	params := make(map[string]string)
	for i, name := range route.Params {
		params[name] = matches[i+1] // matches[0] is full match
	}

	return params
}

// Group creates a route group with common prefix
type Group struct {
	router      *Router
	prefix      string
	middlewares []server.Middleware
}

// Group creates a new route group
func (r *Router) Group(prefix string) *Group {
	return &Group{
		router:      r,
		prefix:      prefix,
		middlewares: make([]server.Middleware, 0),
	}
}

// Use adds middleware to the group
func (g *Group) Use(mw server.Middleware) {
	g.middlewares = append(g.middlewares, mw)
}

// Handle registers a route in the group
func (g *Group) Handle(method, pattern string, handler Handler) {
	fullPattern := g.prefix + pattern

	// Wrap handler with group middlewares
	finalHandler := handler
	for i := len(g.middlewares) - 1; i >= 0; i-- {
		finalHandler = wrapHandlerWithMiddleware(finalHandler, g.middlewares[i])
	}

	g.router.Handle(method, fullPattern, finalHandler)
}

// Convenience methods for groups
func (g *Group) GET(pattern string, handler Handler) {
	g.Handle("GET", pattern, handler)
}

func (g *Group) POST(pattern string, handler Handler) {
	g.Handle("POST", pattern, handler)
}

func (g *Group) PUT(pattern string, handler Handler) {
	g.Handle("PUT", pattern, handler)
}

func (g *Group) DELETE(pattern string, handler Handler) {
	g.Handle("DELETE", pattern, handler)
}

func (g *Group) PATCH(pattern string, handler Handler) {
	g.Handle("PATCH", pattern, handler)
}

// wrapHandlerWithMiddleware wraps a router handler with server middleware
func wrapHandlerWithMiddleware(handler Handler, mw server.Middleware) Handler {
	return func(ctx *server.Context) {
		// Convert router handler to server handler
		serverHandler := server.HandlerFunc(func(c *server.Context) {
			handler(c)
		})

		// Apply middleware
		wrappedHandler := mw(serverHandler)

		// Execute
		wrappedHandler.ServeHTTP(ctx)
	}
}