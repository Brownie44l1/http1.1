package request

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
)

type ParserState string

type Request struct {
	RequestLine RequestLine
	State       ParserState
}

type RequestLine struct {
	HttpVersion   string
	RequestTarget string
	Method        string
}

const (
	StateInit ParserState = "Init"
	StateDone ParserState = "Done"
	StateError ParserState = "Error"
)

var ErrMalformedRequestLine = fmt.Errorf("malformed request line")
var ErrUnsupportedMethod = fmt.Errorf("unsupported method")
var ErrUnsupportedVersion = fmt.Errorf("unsupported version")
var Seperator = "\r\n"

func newRequest() *Request {
	return &Request{ //error
		State: StateInit,
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
			r.State = StateDone

		case StateDone:
			break outer
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

	for !request.done() {
		n, err := reader.Read(buf[bufLen:])
		if err != nil {
			return nil, err
		}

		bufLen += n
		readN, err := request.parse(buf[:bufLen])
		if err != nil {
			return nil, err
		}

		copy(buf, buf[readN:bufLen])
		bufLen -= readN
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

	if method != "GET" {
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
