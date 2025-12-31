package request

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"http1.1/internal/headers"
)

type ParserState string

type Request struct {
	RequestLine RequestLine
	Headers     headers.Headers
	Body        []byte
	State       ParserState
}

type RequestLine struct {
	HttpVersion   string
	RequestTarget string
	Method        string
}

const (
	StateInit           ParserState = "Init"
	StateParsingHeaders ParserState = "ParsingHeaders"
	StateParsingBody    ParserState = "ParsingBody"
	StateDone           ParserState = "Done"
	StateError          ParserState = "Error"
)

var ErrMalformedRequestLine = fmt.Errorf("malformed request line")
var ErrUnsupportedMethod = fmt.Errorf("unsupported method")
var ErrUnsupportedVersion = fmt.Errorf("unsupported version")
var Seperator = "\r\n"

func newRequest() *Request {
	return &Request{ //error
		State:   StateInit,
		Headers: headers.NewHeaders(),
		Body:    nil,
	}
}

func (r *Request) parse(data []byte) (int, error) {
	read := 0

outer:
	for {
		switch r.State {
		case StateError:
			return 0, errors.New("request in error state")

		case StateInit:
			rl, n, err := ParseRequestLine(data)
			if err != nil {
				r.State = StateError
				return 0, err
			}

			if n == 0 {
				break outer
			}

			r.RequestLine = *rl
			read += n
			r.State = StateParsingHeaders

		case StateParsingHeaders:
			n, done, err := r.Headers.Parse(data[read:])
			if err != nil {
				r.State = StateError
				return 0, err
			}

			read += n

			if !done {
				return read, nil
			}

			if done {
				if cl, ok := r.Headers.Get("Content-Length"); ok {
					if cl == "0" {
						r.State = StateDone
					} else {
						r.State = StateParsingBody
					}
				} else {
					r.State = StateDone
				}
			}
			return read, nil

		case StateParsingBody:
			clStr, _ := r.Headers.Get("Content-Length")
			contentLength, err := strconv.Atoi(clStr)
			if err != nil {
				r.State = StateError
				return 0, err
			}

			remaining := contentLength - len(r.Body)
			if remaining <= 0 {
				r.State = StateDone
				break outer // Exit to outer loop
			}

			toRead := min(remaining, len(data[read:]))
			r.Body = append(r.Body, data[read:read+toRead]...)
			read += toRead

			if len(r.Body) > contentLength {
				r.State = StateError
				return 0, errors.New("body larger than Content-Length")
			}

			if len(r.Body) == contentLength {
				r.State = StateDone
				// DON'T return here - let it continue to the outer break
			} else {
				return read, nil // Only return if we need more data
			}

		case StateDone:
			break outer

		default:
			return 0, fmt.Errorf("unknown state")
		}
	}
	return read, nil
}

func (r *Request) done() bool {
	return r.State == StateDone || r.State == StateError
}

func RequestFromReader(reader io.Reader) (*Request, error) {
	request := newRequest()

	buf := make([]byte, 1024)
	bufLen := 0

	for {
		if request.done() {
			break
		}

		// Parse whatever is currently in the buffer
		if bufLen > 0 {
			readN, parseErr := request.parse(buf[:bufLen])
			if parseErr != nil {
				return nil, parseErr
			}

			copy(buf, buf[readN:bufLen])
			bufLen -= readN
			
			// If we made progress, continue parsing
			if readN > 0 {
				continue
			}
		}

		// Only read more data if we need it (buffer empty OR parse made no progress)
		n, err := reader.Read(buf[bufLen:])
		
		if err != nil && err != io.EOF {
			return nil, err
		}
		
		if n == 0 && err == io.EOF {
			if request.State == StateParsingBody {
				if cl, ok := request.Headers.Get("Content-Length"); ok {
					contentLength, _ := strconv.Atoi(cl)
					if len(request.Body) < contentLength {
						return nil, errors.New("unexpected EOF while reading body")
					}
				}
			}
			break
		}
		
		bufLen += n
	}

	if request.State != StateDone {
		return nil, errors.New("incomplete request")
	}

	return request, nil
}

func ParseRequestLine(b []byte) (*RequestLine, int, error) {
	idx := bytes.Index(b, []byte(Seperator))
	if idx == -1 {
		return nil, 0, nil
	}

	startLine := b[:idx]
	read := idx + len(Seperator)

	parts := bytes.Split(startLine, []byte(" "))
	if len(parts) != 3 {
		return nil, 0, ErrMalformedRequestLine
	}

	method := string(parts[0])
	target := string(parts[1])
	version := string(parts[2])

	if method != "GET" && method != "POST" {
		return nil, 0, ErrUnsupportedMethod
	}

	if !strings.HasPrefix(target, "/") {
		return nil, 0, ErrMalformedRequestLine
	}

	if version != "HTTP/1.1" {
		return nil, 0, ErrUnsupportedVersion
	}

	return &RequestLine{
		HttpVersion:   version,
		RequestTarget: target,
		Method:        method,
	}, read, nil
}
