package server

import (
	"bytes"
	"strings"
	"testing"
)

// mockConn is a minimal net.Conn mock that captures written bytes.
type mockConn struct {
	buf bytes.Buffer
}

func (m *mockConn) Write(b []byte) (int, error)        { return m.buf.Write(b) }
func (m *mockConn) Read(b []byte) (int, error)         { return 0, nil }
func (m *mockConn) Close() error                       { return nil }
func (m *mockConn) LocalAddr() mockAddr                { return mockAddr{} }
func (m *mockConn) RemoteAddr() mockAddr               { return mockAddr{} }
func (m *mockConn) SetDeadline(t interface{}) error    { return nil }
func (m *mockConn) SetReadDeadline(t interface{}) error  { return nil }
func (m *mockConn) SetWriteDeadline(t interface{}) error { return nil }

type mockAddr struct{}

func (mockAddr) Network() string { return "tcp" }
func (mockAddr) String() string  { return "127.0.0.1:9999" }

func TestResponseWriterStatus(t *testing.T) {
	res := &ResponseWriter{
		statusCode: 200,
		headers:    make(map[string]string),
	}

	res.WriteHeader(404)
	if res.statusCode != 404 {
		t.Errorf("expected status 404, got %d", res.statusCode)
	}
}

func TestResponseWriterHeaders(t *testing.T) {
	res := &ResponseWriter{
		statusCode: 200,
		headers:    make(map[string]string),
	}

	res.SetHeader("Content-Type", "application/json")
	if res.headers["Content-Type"] != "application/json" {
		t.Error("expected Content-Type header to be set")
	}
}

func TestHeaderInjectionPrevented(t *testing.T) {
	res := &ResponseWriter{
		statusCode: 200,
		headers:    make(map[string]string),
	}

	// Value with CRLF — should be silently dropped
	res.SetHeader("X-Evil", "value\r\nInjected: header")
	if _, exists := res.headers["X-Evil"]; exists {
		t.Error("expected header with CRLF value to be rejected")
	}
}

func TestWriteBody(t *testing.T) {
	res := &ResponseWriter{
		statusCode: 200,
		headers:    make(map[string]string),
	}

	res.Write([]byte("Hello"))
	res.Write([]byte(", World"))

	if string(res.body) != "Hello, World" {
		t.Errorf("expected body 'Hello, World', got %q", string(res.body))
	}
}

func TestStatusText(t *testing.T) {
	cases := map[int]string{
		200: "OK",
		404: "Not Found",
		500: "Internal Server Error",
		405: "Method Not Allowed",
		429: "Too Many Requests",
	}

	for code, expected := range cases {
		got := statusText(code)
		if got != expected {
			t.Errorf("statusText(%d): expected %q, got %q", code, expected, got)
		}
	}
}

func TestWriteJSON(t *testing.T) {
	res := &ResponseWriter{
		statusCode: 200,
		headers:    make(map[string]string),
	}

	err := res.WriteJSON(200, map[string]string{"status": "ok"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.headers["Content-Type"] != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", res.headers["Content-Type"])
	}

	if !strings.Contains(string(res.body), `"status"`) {
		t.Errorf("expected JSON body, got %q", string(res.body))
	}
}

func TestBodySize(t *testing.T) {
	res := &ResponseWriter{
		statusCode: 200,
		headers:    make(map[string]string),
	}

	res.Write([]byte("12345"))
	if res.BodySize() != 5 {
		t.Errorf("expected body size 5, got %d", res.BodySize())
	}
}
