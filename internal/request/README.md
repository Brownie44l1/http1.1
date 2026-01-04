### The flow
```
RequestFromReader()
    ↓
parser.parseFromReader()
    ↓
    ├─ stateRequestLine → requestline.go:parseRequestLine()
    ├─ stateHeaders     → headers.Parse()
    └─ stateBody        → {
           ├─ chunked? → body.go:parseChunked()
           └─ fixed?   → parser.go:parseFixedBody()
       }
    ↓
Request (complete)
```