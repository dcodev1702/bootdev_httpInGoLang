package headers

import (
	"bytes"
	"fmt"
	"strings"
)

func isToken(str []byte) bool {

	for _, ch := range str {
		found := false
		
		if ch >= 'A' && ch <= 'Z' || 
		   ch >= 'a' && ch <= 'z' || 
		   ch >= '0' && ch <= '9' {
			found = true
		}

		switch ch {
			case '!', '#', '$', '%', '&', '\'', '*', '+', '-', '.', '^', '_', '`', '|', '~':
				found = true
		}
		
		if !found {
			return false
		}
	}
	return true
}

var rn = []byte("\r\n")

func parseHeader(fieldLine []byte) (string, string, error) {
	parts := bytes.SplitN(fieldLine, []byte(":"), 2)
	//slog.Info("parseHeader", "field-line", string(fieldLine))
	if len(parts) != 2 {
		return "", "", fmt.Errorf("parseHeader()::malformed field line")
	}

	name := parts[0]
	value := bytes.TrimSpace(parts[1])

	// Header field names must not contain spaces per RFC 9110/9112
	if bytes.HasSuffix(name, []byte(" ")) {
		return "", "", fmt.Errorf("parseHeader()::malformed field name")
	}

	return string(name), string(value), nil
}

type Headers struct {
	headers map[string]string
}

func NewHeaders() *Headers {
	return &Headers{
		headers: map[string]string{},
	}
}

func (h *Headers) Get(name string) (string, bool) {
	str, ok := h.headers[strings.ToLower(name)]
	return str, ok
}

func (h *Headers) Set(name, value string) {
	name = strings.ToLower(name)

	if v, ok := h.headers[name]; ok {
		// field-name is the same, append additional value
		h.headers[name] = fmt.Sprintf("%s,%s", v, value)
	}else {
		h.headers[name] = value
	}
}

func (h *Headers) ForEach(cb func(name, value string)) {
	for name, value := range h.headers {
		cb(name, value)
	}
}

// Implementation for parsing headers from the given data
func (h Headers) Parse(data []byte) (int, bool, error) {

	read := 0
	done := false

	for {
		// Find the next CRLF
		idx := bytes.Index(data[read:], rn)
		if idx == -1 {
			break
		}

		// Empty line indicates end of headers
		if idx == 0 {
			
			done = true
			read += len(rn)
			break
		}

		//fmt.Printf("header line: \"%s\"\n", string(data[read:idx]))
		//fmt.Printf("header line: (%d) - %d\n", read, idx)
		name, value, err := parseHeader(data[read:read+idx])
		if err != nil {
			return 0, false, err
		}

		if !isToken([]byte(name)) {
			return 0, false, fmt.Errorf("isToken()::malformed field name")
		}

		read += idx + len(rn)
		h.Set(name, value)
	}

	return read, done, nil
}
