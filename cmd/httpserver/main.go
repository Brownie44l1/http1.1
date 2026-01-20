package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Brownie44l1/http1.1/internal/response"
	"github.com/Brownie44l1/http1.1/internal/router"
	"github.com/Brownie44l1/http1.1/internal/server"
)

func main() {
	// Create router
	r := router.New()

	// Register routes
	r.GET("/", handleHome)
	r.GET("/hello", handleHello)
	r.GET("/users/:id", handleGetUser)
	r.POST("/users", handleCreateUser)
	r.GET("/api/status", handleStatus)

	// Create server with custom config
	config := &server.Config{
		Addr:           ":8080",
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   15 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1MB
	}

	srv := server.New(config, &RouterAdapter{router: r})

	// Start server in a goroutine
	go func() {
		log.Println("Starting HTTP server on :8080")
		if err := srv.ListenAndServe(); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

// RouterAdapter adapts router.Router to server.Handler
type RouterAdapter struct {
	router *router.Router
}

func (ra *RouterAdapter) ServeHTTP(ctx *server.Context) {
	route, params := ra.router.Match(ctx.Method(), ctx.Path())
	if route == nil {
		ctx.Error(response.StatusNotFound, "Not Found")
		return
	}

	// Set params on context
	ctx.Params = params

	// Call handler
	route.Handler(ctx)
}

// Handler functions

func handleHome(ctx interface{}) {
	c := ctx.(*server.Context)
	html := `<!DOCTYPE html>
<html>
<head>
    <title>HTTP/1.1 Server</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 800px; margin: 50px auto; padding: 20px; }
        h1 { color: #333; }
        .endpoints { background: #f5f5f5; padding: 20px; border-radius: 5px; }
        code { background: #e0e0e0; padding: 2px 5px; border-radius: 3px; }
    </style>
</head>
<body>
    <h1>Welcome to HTTP/1.1 Server!</h1>
    <p>This is a custom HTTP/1.1 server built from scratch in Go.</p>
    
    <div class="endpoints">
        <h2>Available Endpoints:</h2>
        <ul>
            <li><code>GET /</code> - This page</li>
            <li><code>GET /hello</code> - Simple hello message</li>
            <li><code>GET /users/:id</code> - Get user by ID</li>
            <li><code>POST /users</code> - Create new user</li>
            <li><code>GET /api/status</code> - JSON status response</li>
        </ul>
    </div>
    
    <h2>Try it out:</h2>
    <pre>
# Get this page
curl http://localhost:8080/

# Get hello message
curl http://localhost:8080/hello

# Get user by ID
curl http://localhost:8080/users/123

# Create user (POST)
curl -X POST http://localhost:8080/users -d '{"name":"John"}'

# Get JSON status
curl http://localhost:8080/api/status
    </pre>
</body>
</html>`

	c.HTML(response.StatusOK, html)
}

func handleHello(ctx interface{}) {
	c := ctx.(*server.Context)
	name := c.Query("name")
	if name == "" {
		name = "World"
	}
	c.String(response.StatusOK, "Hello, %s!", name)
}

func handleGetUser(ctx interface{}) {
	c := ctx.(*server.Context)
	userID := c.Param("id")

	// Simulate getting user from database
	json := `{
  "id": "` + userID + `",
  "name": "John Doe",
  "email": "john@example.com",
  "created_at": "2024-01-15T10:30:00Z"
}`

	c.JSON(response.StatusOK, json)
}

func handleCreateUser(ctx interface{}) {
	c := ctx.(*server.Context)
	body := c.BodyString()

	log.Printf("Creating user with data: %s", body)

	// Simulate creating user
	json := `{
  "success": true,
  "message": "User created successfully",
  "data": ` + body + `
}`

	c.JSON(response.StatusCreated, json)
}

func handleStatus(ctx interface{}) {
	c := ctx.(*server.Context)
	json := `{
  "status": "ok",
  "version": "1.0.0",
  "uptime": "24h",
  "requests_handled": 1000
}`

	c.JSON(response.StatusOK, json)
}
