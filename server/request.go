package server

import (
	"bufio"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
)

// Request represents a parsed HTTP/1.1 request.
type Request struct {
	Method     string
	Path       string
	RawQuery   string
	Query      map[string]string
	Headers    map[string]string
	Body       []byte
	Params     map[string]string
	RemoteAddr string
	Proto      string
}

// ParseRequest reads from r and parses a single HTTP/1.1 request.
// It returns a populated Request or an error if the request is malformed.
//
// Parsing happens in two phases:
//  1. Header phase — reads line by line until the blank CRLF separator,
//     extracting the request line and all headers.
//  2. Body phase — reads exactly Content-Length bytes if a body is present.
func ParseRequest(r io.Reader, remoteAddr string) (*Request, error) {
	reader := bufio.NewReader(r)

	// --- Phase 1: Request line ---
	requestLine, err := readLine(reader)
	if err != nil {
		return nil, fmt.Errorf("reading request line: %w", err)
	}

	method, rawPath, proto, err := parseRequestLine(requestLine)
	if err != nil {
		return nil, err
	}

	// --- Phase 2: Headers ---
	headers, err := parseHeaders(reader)
	if err != nil {
		return nil, fmt.Errorf("parsing headers: %w", err)
	}

	// --- Phase 3: Parse path and query string ---
	path, query, err := parsePath(rawPath)
	if err != nil {
		return nil, fmt.Errorf("parsing path: %w", err)
	}

	// --- Phase 4: Body ---
	body, err := readBody(reader, headers)
	if err != nil {
		return nil, fmt.Errorf("reading body: %w", err)
	}

	return &Request{
		Method:     method,
		Path:       path,
		RawQuery:   query,
		Query:      parseQueryString(query),
		Headers:    headers,
		Body:       body,
		Params:     make(map[string]string),
		RemoteAddr: remoteAddr,
		Proto:      proto,
	}, nil
}

// parseRequestLine splits "METHOD /path HTTP/1.1" into its three parts.
func parseRequestLine(line string) (method, path, proto string, err error) {
	parts := strings.SplitN(line, " ", 3)
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("malformed request line: %q", line)
	}

	method = strings.ToUpper(parts[0])
	path = parts[1]
	proto = parts[2]

	if !isValidMethod(method) {
		return "", "", "", fmt.Errorf("invalid HTTP method: %q", method)
	}

	if !strings.HasPrefix(proto, "HTTP/") {
		return "", "", "", fmt.Errorf("invalid protocol: %q", proto)
	}

	if path == "" {
		return "", "", "", fmt.Errorf("empty request path")
	}

	return method, path, proto, nil
}

// parseHeaders reads header lines until the blank line separator.
// Returns headers as a lowercase-key map.
func parseHeaders(reader *bufio.Reader) (map[string]string, error) {
	headers := make(map[string]string)

	for {
		line, err := readLine(reader)
		if err != nil {
			return nil, err
		}

		// Blank line signals end of headers
		if line == "" {
			break
		}

		key, value, found := strings.Cut(line, ":")
		if !found {
			return nil, fmt.Errorf("malformed header line: %q", line)
		}

		// Normalize: lowercase key, trimmed value
		key = strings.ToLower(strings.TrimSpace(key))
		value = strings.TrimSpace(value)

		// Reject header injection attempts
		if strings.ContainsAny(value, "\r\n") {
			return nil, fmt.Errorf("header value contains illegal characters")
		}

		headers[key] = value
	}

	return headers, nil
}

// parsePath splits the raw request path into the path and raw query string.
func parsePath(raw string) (path, query string, err error) {
	u, err := url.ParseRequestURI(raw)
	if err != nil {
		return "", "", fmt.Errorf("invalid path %q: %w", raw, err)
	}
	return u.Path, u.RawQuery, nil
}

// parseQueryString parses a raw query string into a key-value map.
// Duplicate keys are overwritten by the last value.
func parseQueryString(raw string) map[string]string {
	result := make(map[string]string)
	if raw == "" {
		return result
	}

	values, err := url.ParseQuery(raw)
	if err != nil {
		return result
	}

	for k, v := range values {
		if len(v) > 0 {
			result[k] = v[0]
		}
	}

	return result
}

// readBody reads exactly Content-Length bytes from the reader if present.
// Returns nil for requests with no body.
func readBody(reader *bufio.Reader, headers map[string]string) ([]byte, error) {
	clStr, ok := headers["content-length"]
	if !ok {
		return nil, nil
	}

	contentLength, err := strconv.Atoi(clStr)
	if err != nil {
		return nil, fmt.Errorf("invalid content-length: %q", clStr)
	}

	if contentLength < 0 {
		return nil, fmt.Errorf("negative content-length: %d", contentLength)
	}

	if contentLength == 0 {
		return nil, nil
	}

	body := make([]byte, contentLength)
	_, err = io.ReadFull(reader, body)
	if err != nil {
		return nil, fmt.Errorf("reading body: %w", err)
	}

	return body, nil
}

// readLine reads a single CRLF-terminated line from the reader,
// stripping the trailing \r\n.
func readLine(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	// Strip CRLF or LF
	line = strings.TrimRight(line, "\r\n")
	return line, nil
}

// isValidMethod returns true for standard HTTP methods.
func isValidMethod(method string) bool {
	switch method {
	case "GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS", "CONNECT", "TRACE":
		return true
	}
	return false
}

// KeepAlive reports whether the request connection header requests keep-alive.
func (r *Request) KeepAlive() bool {
	conn := strings.ToLower(r.Headers["connection"])
	if conn == "close" {
		return false
	}
	// HTTP/1.1 defaults to keep-alive
	if r.Proto == "HTTP/1.1" {
		return true
	}
	// HTTP/1.0 requires explicit keep-alive
	return conn == "keep-alive"
}

// ContentType returns the Content-Type header value, or empty string.
func (r *Request) ContentType() string {
	return r.Headers["content-type"]
}

// Header returns the value of a named header (case-insensitive).
func (r *Request) Header(name string) string {
	return r.Headers[strings.ToLower(name)]
}
