.PHONY: help build run test clean fmt vet bench

# Default target
help:
	@echo "HTTP/1.1 Server - Makefile Commands"
	@echo ""
	@echo "  make build    - Build the server binary"
	@echo "  make run      - Run the example server"
	@echo "  make test     - Run all tests"
	@echo "  make bench    - Run benchmarks"
	@echo "  make fmt      - Format code"
	@echo "  make vet      - Run go vet"
	@echo "  make clean    - Clean build artifacts"
	@echo ""

# Build the server
build:
	@echo "Building server..."
	@mkdir -p bin
	@go build -o bin/httpserver cmd/httpserver/main.go
	@echo "✓ Built to bin/httpserver"

# Run the example server
run: build
	@echo "Starting HTTP/1.1 server on :8080..."
	@./bin/httpserver

# Run all tests
test:
	@echo "Running tests..."
	@go test ./... -v

# Run tests with coverage
coverage:
	@echo "Running tests with coverage..."
	@go test ./... -cover -coverprofile=coverage.out
	@go tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report: coverage.html"

# Run benchmarks
bench:
	@echo "Running benchmarks..."
	@go test ./... -bench=. -benchmem

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...
	@echo "✓ Code formatted"

# Run go vet
vet:
	@echo "Running go vet..."
	@go vet ./...
	@echo "✓ Vet passed"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html
	@echo "✓ Cleaned"

# Install dependencies
deps:
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy
	@echo "✓ Dependencies ready"

# Run all checks (fmt, vet, test)
check: fmt vet test
	@echo "✓ All checks passed"