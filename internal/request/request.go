package request

import (
	"GO-HTTPSVR/internal/headers"
	"bytes"
	"errors"
	"fmt"
	"io"
)


type parserState string
const (
	StateInit     parserState = "init"
	StateHeaders  parserState = "headers"
	StateDone     parserState = "done"
	StateError    parserState = "error"
)

type RequestLine struct {
	HttpVersion   string
	RequestTarget string
	Method        string
}

type Request struct {
	RequestLine RequestLine
	Headers	 	*headers.Headers
	state       parserState
}

var ErrorMalformedRequestLine = fmt.Errorf("malformed request-line!")
//var ErrorUnsupportedHttpVersion = fmt.Errorf("unsupported HTTP version!")
var ErrorRequestInErrorState = fmt.Errorf("request in error state")
var SEPARATOR = []byte("\r\n")

func newRequest() *Request {
	return &Request {
		state: StateInit,
		Headers: headers.NewHeaders(),
	}
}

func parseRequestLine(b []byte) (*RequestLine, int, error) {	
	idx := bytes.Index(b, SEPARATOR)
	if idx == -1 {
		return nil, 0, nil
	}

	startLine := b[:idx]
	read := idx+len(SEPARATOR)

	parts := bytes.Split(startLine, []byte(" "))
	if len(parts) != 3 {
		return nil, 0, ErrorMalformedRequestLine
	}

	httpParts := bytes.SplitN(parts[2], []byte("/"), 2)
	if len(httpParts) != 2 || string(httpParts[0]) != "HTTP" || string(httpParts[1]) != "1.1" {
		return nil, 0, ErrorMalformedRequestLine
	}

	rl := &RequestLine{
		Method:        string(parts[0]),
		RequestTarget: string(parts[1]),
		HttpVersion:   string(httpParts[1]),
	}

	return rl, read, nil
}

func (r *Request) parse(data []byte) (int, error) {
	read := 0
	outer:
		for  {
			currentData := data[read:]

			switch r.state {
				case StateError:
					return 0, ErrorRequestInErrorState

				case StateInit:
					rl, n, err := parseRequestLine(currentData)
					if err != nil {
						r.state = StateError
						return 0, err
					}

					if n == 0 {
						break outer
					}

					r.RequestLine = *rl
					read += n
					r.state = StateHeaders

				case StateHeaders:
					n, done, err := r.Headers.Parse(currentData)
					if err != nil {
						//r.state = StateError
						return 0, err
					}

					if n == 0 {
						break outer
					}

					read += n

					if done {
						r.state = StateDone
					}

				case StateDone:
					break outer
				
				default:
					panic(fmt.Sprintf("DEFAULT: We are in an unexpected state: %s\n", r.state))
			}
		}
	return read, nil
}

func (r *Request) done() bool {
	return r.state == StateDone || r.state == StateError
}

func (r *Request) error() bool {
	return r.state == StateError
}

func RequestFromReader(reader io.Reader) (*Request, error) {
	request := newRequest()

	// NOTE: buffer could get overrun... a header or the body that exceeds 4K
	buf := make([]byte, (4 * 1024))
	bufLen := 0
	for !request.done() {
		n, err := reader.Read(buf[bufLen:])
		if err != nil {
			return nil, errors.Join(fmt.Errorf("unable to read from reader"), err)
		}

		bufLen += n
		readN, err := request.parse(buf[:bufLen])
		if err != nil {
			return nil, err
		}

		copy (buf, buf[readN:bufLen])
		bufLen -= readN
	}
	return request, nil
}
