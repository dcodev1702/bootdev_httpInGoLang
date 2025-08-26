package request

import (
	"GO-HTTPSVR/internal/headers"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
)


type parserState string
const (
	StateInit     parserState = "init"
	StateHeaders  parserState = "headers"
	StateBody     parserState = "body"
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
	Body		string
	state       parserState
}

var ErrorMalformedRequestLine = fmt.Errorf("malformed request-line!")
//var ErrorUnsupportedHttpVersion = fmt.Errorf("unsupported HTTP version!")
var ErrorRequestInErrorState = fmt.Errorf("request in error state")
var SEPARATOR = []byte("\r\n")

func getIntHeaders(headers *headers.Headers, name string, defaultValue int) int {
	valueStr, exists := headers.Get(name)
	if !exists {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

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

func (r *Request) hasBody() bool {
	//TODO: When doing chuncked encoding, modify this method
	contentLength := getIntHeaders(r.Headers, "content-length", 0)
	return contentLength > 0
}

func (r *Request) parse(data []byte) (int, error) {
	read := 0
	dance:
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
						//r.state = StateDone
						break dance
					}

					r.RequestLine = *rl
					read += n
					r.state = StateHeaders

				case StateHeaders:
					n, done, err := r.Headers.Parse(currentData)
					if err != nil {
						r.state = StateError
						return 0, err
					}

					if n == 0 {
						//r.state = StateDone
						break dance
					}

					read += n

					// in the real world we would not get an EOF after reading data
					// therefore we would nicely transition to body, which would
					// allow us to then transition to done, if applicable
					// so intead, we're transitioning here. :(
					if done {
						if r.hasBody() {
							r.state = StateBody
						} else {
							r.state = StateDone
						}
					}

				case StateBody:
					lengthStr := getIntHeaders(r.Headers, "content-length", 0)

					if lengthStr == 0 {
						panic("chunked not implemented!")
					}

					remainingData := min(lengthStr - len(r.Body), len(currentData))
					r.Body += string(currentData[:remainingData])
					
					read += remainingData

					if len(r.Body) == lengthStr {
						r.state = StateDone
					}

				case StateDone:
					break dance
				
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

	buf := make([]byte, (4 * 1024))
	bufLen := 0
	for !request.done() {
		n, err := reader.Read(buf[bufLen:])
		
		if err != nil && err != io.EOF {
			return nil, errors.Join(fmt.Errorf("unable to read from reader"), err)
		}

		bufLen += n
		readN, parseErr := request.parse(buf[:bufLen])
		if parseErr != nil {
			return nil, parseErr
		}

		// Detect infinite loop condition
		if readN == 0 && n == 0 && bufLen > 0 {
			return nil, fmt.Errorf("parser stuck: buffer full but no progress made")
		}

		copy(buf, buf[readN:bufLen])
		bufLen -= readN

		if err == io.EOF {
			if request.state == StateBody {
				contentLength := getIntHeaders(request.Headers, "content-length", 0)
				if len(request.Body) < contentLength {
					return nil, fmt.Errorf("unexpected EOF: expected %d bytes in body, got %d", 
						contentLength, len(request.Body))
				}
			}
			if !request.done() {
				return nil, fmt.Errorf("unexpected EOF while parsing request")
			}
			break
		}
	}
	return request, nil
}


/*
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
*/
