package bench

import (
	"strings"
	"testing"

	"github.com/Sou-Daroh/http-server/server"
)

// BenchmarkRequestParse measures the throughput of the HTTP request parser.
func BenchmarkRequestParse(b *testing.B) {
	raw := "GET /hello?name=world&page=1 HTTP/1.1\r\nHost: localhost\r\nUser-Agent: bench\r\nAccept: */*\r\nConnection: keep-alive\r\n\r\n"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := server.ParseRequest(strings.NewReader(raw), "127.0.0.1:1234")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkRequestParseWithBody measures parsing a POST request with a JSON body.
func BenchmarkRequestParseWithBody(b *testing.B) {
	body := `{"username":"alice","password":"secret","remember":true}`
	raw := "POST /login HTTP/1.1\r\nHost: localhost\r\nContent-Type: application/json\r\nContent-Length: 55\r\n\r\n" + body

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := server.ParseRequest(strings.NewReader(raw), "127.0.0.1:1234")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkRouter measures route matching across a table of registered routes.
func BenchmarkRouter10Routes(b *testing.B) {
	r := server.NewRouter()
	noop := func(req *server.Request, res *server.ResponseWriter) {}

	r.GET("/", noop)
	r.GET("/users", noop)
	r.GET("/users/:id", noop)
	r.POST("/users", noop)
	r.PUT("/users/:id", noop)
	r.DELETE("/users/:id", noop)
	r.GET("/posts", noop)
	r.GET("/posts/:id", noop)
	r.GET("/health", noop)
	r.GET("/api/echo", noop)

	req := &server.Request{Method: "GET", Path: "/users/42", Params: make(map[string]string)}
	res := &server.ResponseWriter{}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req.Params = make(map[string]string)
		r.Dispatch(req, res)
	}
}

// BenchmarkRouterNoMatch measures the cost of a 404 lookup.
func BenchmarkRouterNoMatch(b *testing.B) {
	r := server.NewRouter()
	noop := func(req *server.Request, res *server.ResponseWriter) {}

	r.GET("/users", noop)
	r.GET("/posts", noop)
	r.GET("/health", noop)

	req := &server.Request{Method: "GET", Path: "/nonexistent", Params: make(map[string]string)}
	res := &server.ResponseWriter{}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req.Params = make(map[string]string)
		r.Dispatch(req, res)
	}
}
