## Why This Split?

**server.go** - "I manage the server lifecycle"
- When asked: "Start listening" or "Shut down"

**conn.go** - "I handle one connection's entire lifetime"
- When asked: "Process all requests on this TCP connection"

**persistence.go** - "I decide when to close connections"
- When asked: "Should we keep this connection open?"

## The Flow Now:
```
server.listen() → accept connection
    ↓
server.serveConn(conn) → loop for requests
    ↓
    ├─ request.RequestFromReader() → parse
    ├─ server.handleRequest() → your handler (with panic recovery)
    └─ shouldCloseConnection() → keep-alive decision (persistence.go)
         ↓
    continue loop OR return (closes connection)

```

- Clean separation. Each file has one job. When you add features:

- Rate limiting? → Add to conn.go (per-connection tracking)
- Graceful shutdown? → Modify server.go (track active connections)
- Connection pooling metrics? → Add to persistence.go (log close reasons)