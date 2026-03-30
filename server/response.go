package server

import (
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

// ResponseWriter builds and writes an HTTP/1.1 response to a net.Conn.
type ResponseWriter struct {
	conn       net.Conn
	statusCode int
	headers    map[string]string
	body       []byte
	written    bool
}

// NewResponseWriter creates a ResponseWriter for the given connection.
func NewResponseWriter(conn net.Conn) *ResponseWriter {
	return &ResponseWriter{
		conn:       conn,
		statusCode: 200,
		headers:    make(map[string]string),
	}
}

// WriteHeader sets the HTTP status code.
// Must be called before Write or Flush if you want a non-200 status.
func (rw *ResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
}

// SetHeader sets a response header key/value pair.
// Silently drops values containing CRLF to prevent header injection.
func (rw *ResponseWriter) SetHeader(key, value string) {
	if strings.ContainsAny(value, "\r\n") {
		return
	}
	rw.headers[key] = value
}

// Write appends bytes to the response body buffer.
func (rw *ResponseWriter) Write(data []byte) {
	rw.body = append(rw.body, data...)
}

// WriteString appends a string to the response body buffer.
func (rw *ResponseWriter) WriteString(s string) {
	rw.Write([]byte(s))
}

// WriteJSON sets Content-Type to application/json, marshals v,
// and writes the result as the response body.
func (rw *ResponseWriter) WriteJSON(code int, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	rw.WriteHeader(code)
	rw.SetHeader("Content-Type", "application/json")
	rw.Write(data)
	return nil
}

// WriteText sets Content-Type to text/plain and writes the string as body.
func (rw *ResponseWriter) WriteText(code int, text string) {
	rw.WriteHeader(code)
	rw.SetHeader("Content-Type", "text/plain; charset=utf-8")
	rw.WriteString(text)
}

// WriteHTML sets Content-Type to text/html and writes the string as body.
func (rw *ResponseWriter) WriteHTML(code int, html string) {
	rw.WriteHeader(code)
	rw.SetHeader("Content-Type", "text/html; charset=utf-8")
	rw.WriteString(html)
}

// Flush serializes and sends the complete HTTP response to the connection.
// It can only be called once — subsequent calls are no-ops.
func (rw *ResponseWriter) Flush(keepAlive bool) error {
	if rw.written {
		return nil
	}
	rw.written = true

	// Auto-set Content-Length
	rw.headers["Content-Length"] = strconv.Itoa(len(rw.body))

	// Connection header
	if keepAlive {
		rw.headers["Connection"] = "keep-alive"
	} else {
		rw.headers["Connection"] = "close"
	}

	// Date header
	rw.headers["Date"] = time.Now().UTC().Format(time.RFC1123)

	// Server header
	rw.headers["Server"] = "http-server/0.6"

	// Build response
	var sb strings.Builder

	// Status line
	sb.WriteString(fmt.Sprintf("HTTP/1.1 %d %s\r\n", rw.statusCode, statusText(rw.statusCode)))

	// Headers
	for k, v := range rw.headers {
		sb.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}

	// Blank line
	sb.WriteString("\r\n")

	// Write headers
	_, err := fmt.Fprint(rw.conn, sb.String())
	if err != nil {
		return err
	}

	// Write body
	if len(rw.body) > 0 {
		_, err = rw.conn.Write(rw.body)
		if err != nil {
			return err
		}
	}

	return nil
}

// StatusCode returns the current status code.
func (rw *ResponseWriter) StatusCode() int {
	return rw.statusCode
}

// BodySize returns the number of bytes in the body buffer.
func (rw *ResponseWriter) BodySize() int {
	return len(rw.body)
}

// statusText returns the standard reason phrase for an HTTP status code.
func statusText(code int) string {
	switch code {
	case 100:
		return "Continue"
	case 200:
		return "OK"
	case 201:
		return "Created"
	case 204:
		return "No Content"
	case 301:
		return "Moved Permanently"
	case 302:
		return "Found"
	case 304:
		return "Not Modified"
	case 400:
		return "Bad Request"
	case 401:
		return "Unauthorized"
	case 403:
		return "Forbidden"
	case 404:
		return "Not Found"
	case 405:
		return "Method Not Allowed"
	case 409:
		return "Conflict"
	case 413:
		return "Payload Too Large"
	case 429:
		return "Too Many Requests"
	case 500:
		return "Internal Server Error"
	case 501:
		return "Not Implemented"
	case 503:
		return "Service Unavailable"
	default:
		return "Unknown"
	}
}
