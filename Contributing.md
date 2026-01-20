# Contributing to HTTP/1.1 Server

Thank you for your interest in contributing! This document provides guidelines for contributing to this project.

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/YOUR_USERNAME/http1.1.git`
3. Create a branch: `git checkout -b feature/your-feature-name`
4. Make your changes
5. Run tests: `make test`
6. Commit: `git commit -m "Add feature: your feature description"`
7. Push: `git push origin feature/your-feature-name`
8. Create a Pull Request

## Development Setup

```bash
# Clone repository
git clone https://github.com/Brownie44l1/http1.1.git
cd http1.1

# Install dependencies
make deps

# Run tests
make test

# Run the server
make run
```

## Code Style

- Follow standard Go conventions
- Run `gofmt` before committing: `make fmt`
- Run `go vet` before committing: `make vet`
- Write tests for new features
- Keep functions small and focused
- Add comments for exported functions

## Testing

- Write unit tests for all new code
- Maintain or improve code coverage
- Test error cases, not just happy paths
- Use table-driven tests where appropriate

Example:
```go
func TestRequestParsing(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    *Request
        wantErr bool
    }{
        {
            name:  "valid GET request",
            input: "GET / HTTP/1.1\r\n\r\n",
            want:  &Request{Method: "GET", Path: "/"},
        },
        // More test cases...
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := Parse(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
            }
            // Assert got == tt.want
        })
    }
}
```

## Pull Request Guidelines

- Create focused PRs (one feature/fix per PR)
- Write clear commit messages
- Update documentation if needed
- Add tests for new features
- Ensure all tests pass
- Reference related issues

### Commit Message Format

```
type(scope): short description

Longer description if needed

Fixes #123
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `test`: Adding tests
- `refactor`: Code refactoring
- `perf`: Performance improvements

## Areas for Contribution

### Good First Issues
- Add more tests
- Improve documentation
- Add code examples
- Fix typos

### Feature Ideas
- Middleware system
- Static file serving
- Template rendering
- WebSocket support
- Server-Sent Events
- Compression support
- Cookie handling
- CORS support

### Performance
- Optimize parsers
- Reduce allocations
- Improve concurrency
- Add benchmarks

## Questions?

Open an issue for:
- Bug reports
- Feature requests
- Questions about the code
- Design discussions

## License
By contributing, you agree that your contributions will be licensed under the MIT License.