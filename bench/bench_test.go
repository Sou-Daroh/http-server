package bench

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// BenchmarkGinRouter matches the throughput of Gin's routing engine 
// against our old custom TCP router for an accurate syllabus comparison.
func BenchmarkGinRouter10Routes(b *testing.B) {
	// Suppress Gin debug logging for accurate CPU measurements
	gin.SetMode(gin.ReleaseMode)
	
	r := gin.New()
	noop := func(c *gin.Context) {
		c.Status(200)
	}

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

	// Simulate hitting the nested parameter route
	req, _ := http.NewRequest("GET", "/users/42", nil)
	
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}
}

// BenchmarkGinJsonPayload measures Gin parsing a POST request seamlessly with a JSON payload
func BenchmarkGinJsonPayload(b *testing.B) {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	
	r.POST("/login", func(c *gin.Context) {
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
			Remember bool   `json:"remember"`
		}
		// Measure native binding
		if err := c.ShouldBindJSON(&req); err != nil {
			c.Status(400)
			return
		}
		c.Status(200)
	})

	body := []byte(`{"username":"alice","password":"secret","remember":true}`)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
	}
}
