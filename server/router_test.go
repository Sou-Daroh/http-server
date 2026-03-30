package server

import (
	"testing"
)

func TestExactRouteMatch(t *testing.T) {
	r := NewRouter()
	called := false

	r.GET("/hello", func(req *Request, res *ResponseWriter) {
		called = true
	})

	req := &Request{Method: "GET", Path: "/hello", Params: make(map[string]string)}
	res := &ResponseWriter{statusCode: 200, headers: make(map[string]string)}
	r.Dispatch(req, res)

	if !called {
		t.Error("expected handler to be called")
	}
}

func TestPathParameter(t *testing.T) {
	r := NewRouter()
	var capturedID string

	r.GET("/users/:id", func(req *Request, res *ResponseWriter) {
		capturedID = req.Params["id"]
	})

	req := &Request{Method: "GET", Path: "/users/42", Params: make(map[string]string)}
	res := &ResponseWriter{statusCode: 200, headers: make(map[string]string)}
	r.Dispatch(req, res)

	if capturedID != "42" {
		t.Errorf("expected id=42, got %q", capturedID)
	}
}

func TestMultiplePathParameters(t *testing.T) {
	r := NewRouter()
	var year, slug string

	r.GET("/posts/:year/:slug", func(req *Request, res *ResponseWriter) {
		year = req.Params["year"]
		slug = req.Params["slug"]
	})

	req := &Request{Method: "GET", Path: "/posts/2026/my-post", Params: make(map[string]string)}
	res := &ResponseWriter{statusCode: 200, headers: make(map[string]string)}
	r.Dispatch(req, res)

	if year != "2026" {
		t.Errorf("expected year=2026, got %q", year)
	}
	if slug != "my-post" {
		t.Errorf("expected slug=my-post, got %q", slug)
	}
}

func TestNotFound(t *testing.T) {
	r := NewRouter()

	req := &Request{Method: "GET", Path: "/nonexistent", Params: make(map[string]string)}
	res := &ResponseWriter{statusCode: 200, headers: make(map[string]string)}
	r.Dispatch(req, res)

	if res.statusCode != 404 {
		t.Errorf("expected 404, got %d", res.statusCode)
	}
}

func TestMethodNotAllowed(t *testing.T) {
	r := NewRouter()
	r.GET("/resource", func(req *Request, res *ResponseWriter) {})

	req := &Request{Method: "POST", Path: "/resource", Params: make(map[string]string)}
	res := &ResponseWriter{statusCode: 200, headers: make(map[string]string)}
	r.Dispatch(req, res)

	if res.statusCode != 405 {
		t.Errorf("expected 405, got %d", res.statusCode)
	}
	if res.headers["Allow"] == "" {
		t.Error("expected Allow header to be set on 405 response")
	}
}

func TestAllHTTPMethods(t *testing.T) {
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}

	for _, method := range methods {
		r := NewRouter()
		called := false

		r.Handle(method, "/test", func(req *Request, res *ResponseWriter) {
			called = true
		})

		req := &Request{Method: method, Path: "/test", Params: make(map[string]string)}
		res := &ResponseWriter{statusCode: 200, headers: make(map[string]string)}
		r.Dispatch(req, res)

		if !called {
			t.Errorf("handler not called for method %s", method)
		}
	}
}

func TestSplitPath(t *testing.T) {
	cases := []struct {
		input    string
		expected []string
	}{
		{"/", []string{}},
		{"/hello", []string{"hello"}},
		{"/users/42", []string{"users", "42"}},
		{"/a/b/c/", []string{"a", "b", "c"}},
		{"", []string{}},
	}

	for _, c := range cases {
		result := splitPath(c.input)
		if len(result) != len(c.expected) {
			t.Errorf("splitPath(%q): expected %v, got %v", c.input, c.expected, result)
			continue
		}
		for i := range result {
			if result[i] != c.expected[i] {
				t.Errorf("splitPath(%q)[%d]: expected %q, got %q", c.input, i, c.expected[i], result[i])
			}
		}
	}
}
