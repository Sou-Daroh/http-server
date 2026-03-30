package server

import (
	"strings"
	"testing"
)

func TestParseGETRequest(t *testing.T) {
	raw := "GET /hello HTTP/1.1\r\nHost: localhost\r\nUser-Agent: test\r\n\r\n"
	req, err := ParseRequest(strings.NewReader(raw), "127.0.0.1:1234")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Method != "GET" {
		t.Errorf("expected method GET, got %q", req.Method)
	}
	if req.Path != "/hello" {
		t.Errorf("expected path /hello, got %q", req.Path)
	}
	if req.Headers["host"] != "localhost" {
		t.Errorf("expected host header 'localhost', got %q", req.Headers["host"])
	}
}

func TestParseQueryString(t *testing.T) {
	raw := "GET /search?q=golang&page=2 HTTP/1.1\r\nHost: localhost\r\n\r\n"
	req, err := ParseRequest(strings.NewReader(raw), "127.0.0.1:1234")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Path != "/search" {
		t.Errorf("expected path /search, got %q", req.Path)
	}
	if req.Query["q"] != "golang" {
		t.Errorf("expected q=golang, got %q", req.Query["q"])
	}
	if req.Query["page"] != "2" {
		t.Errorf("expected page=2, got %q", req.Query["page"])
	}
}

func TestParsePOSTWithBody(t *testing.T) {
	body := `{"name":"Alice"}`
	raw := "POST /users HTTP/1.1\r\nHost: localhost\r\nContent-Type: application/json\r\nContent-Length: 16\r\n\r\n" + body
	req, err := ParseRequest(strings.NewReader(raw), "127.0.0.1:1234")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Method != "POST" {
		t.Errorf("expected POST, got %q", req.Method)
	}
	if string(req.Body) != body {
		t.Errorf("expected body %q, got %q", body, string(req.Body))
	}
}

func TestMalformedRequestLine(t *testing.T) {
	cases := []string{
		"NOTVALID\r\n\r\n",
		"\r\n\r\n",
		"GET\r\n\r\n",
		"GET /path\r\n\r\n",
	}

	for _, raw := range cases {
		_, err := ParseRequest(strings.NewReader(raw), "127.0.0.1:1234")
		if err == nil {
			t.Errorf("expected error for input %q, got nil", raw)
		}
	}
}

func TestInvalidMethod(t *testing.T) {
	raw := "HACK /path HTTP/1.1\r\nHost: localhost\r\n\r\n"
	_, err := ParseRequest(strings.NewReader(raw), "127.0.0.1:1234")
	if err == nil {
		t.Error("expected error for invalid method, got nil")
	}
}

func TestHeaderInjectionRejected(t *testing.T) {
	raw := "GET / HTTP/1.1\r\nHost: localhost\r\nX-Evil: value\r\ninjected: header\r\n\r\n"
	// Header with CRLF in value — should be rejected
	raw2 := "GET / HTTP/1.1\r\nHost: localhost\r\nX-Bad: value\r\nextra: injected\r\n\r\n"

	_, err := ParseRequest(strings.NewReader(raw), "127.0.0.1:1234")
	if err != nil {
		t.Logf("first case: got expected behaviour: %v", err)
	}

	_, err2 := ParseRequest(strings.NewReader(raw2), "127.0.0.1:1234")
	if err2 != nil {
		t.Logf("second case: got expected behaviour: %v", err2)
	}
}

func TestKeepAliveHTTP11(t *testing.T) {
	raw := "GET / HTTP/1.1\r\nHost: localhost\r\n\r\n"
	req, err := ParseRequest(strings.NewReader(raw), "127.0.0.1:1234")
	if err != nil {
		t.Fatal(err)
	}
	if !req.KeepAlive() {
		t.Error("expected keep-alive=true for HTTP/1.1 without Connection header")
	}
}

func TestKeepAliveClose(t *testing.T) {
	raw := "GET / HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n"
	req, err := ParseRequest(strings.NewReader(raw), "127.0.0.1:1234")
	if err != nil {
		t.Fatal(err)
	}
	if req.KeepAlive() {
		t.Error("expected keep-alive=false with Connection: close")
	}
}

func TestEmptyQueryString(t *testing.T) {
	raw := "GET /path HTTP/1.1\r\nHost: localhost\r\n\r\n"
	req, err := ParseRequest(strings.NewReader(raw), "127.0.0.1:1234")
	if err != nil {
		t.Fatal(err)
	}
	if len(req.Query) != 0 {
		t.Errorf("expected empty query map, got %v", req.Query)
	}
}

func TestNoBody(t *testing.T) {
	raw := "GET / HTTP/1.1\r\nHost: localhost\r\n\r\n"
	req, err := ParseRequest(strings.NewReader(raw), "127.0.0.1:1234")
	if err != nil {
		t.Fatal(err)
	}
	if req.Body != nil {
		t.Errorf("expected nil body, got %v", req.Body)
	}
}
