package middleware

import (
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Sou-Daroh/http-server/server"
)

// ---- Logger ----------------------------------------------------------------

// LoggerConfig holds logger configuration.
type LoggerConfig struct {
	// LogFile is the path to write access logs. Empty means stdout only.
	LogFile string
}

// Logger returns middleware that logs every request.
// Output format: timestamp  METHOD  path  status  latency  remote_ip
func Logger(cfg LoggerConfig) server.MiddlewareFunc {
	var logFile *os.File

	if cfg.LogFile != "" {
		f, err := os.OpenFile(cfg.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "logger: could not open log file %q: %v\n", cfg.LogFile, err)
		} else {
			logFile = f
		}
	}

	return func(next server.HandlerFunc) server.HandlerFunc {
		return func(req *server.Request, res *server.ResponseWriter) {
			start := time.Now()
			next(req, res)
			elapsed := time.Since(start)

			// Format: 2026-03-15T10:23:45Z  GET  /index.html  200  1.2ms  127.0.0.1
			ip, _, _ := net.SplitHostPort(req.RemoteAddr)
			line := fmt.Sprintf("%s\t%s\t%s\t%d\t%s\t%s\n",
				time.Now().UTC().Format(time.RFC3339),
				req.Method,
				req.Path,
				res.StatusCode(),
				elapsed.Round(time.Microsecond),
				ip,
			)

			fmt.Print(line)

			if logFile != nil {
				logFile.WriteString(line)
			}
		}
	}
}

// ---- CORS ------------------------------------------------------------------

// CORSConfig holds CORS settings.
type CORSConfig struct {
	AllowedOrigins []string
	AllowedMethods []string
	AllowedHeaders []string
}

// CORS returns middleware that adds Cross-Origin Resource Sharing headers.
// Preflight OPTIONS requests are handled and short-circuited automatically.
func CORS(cfg CORSConfig) server.MiddlewareFunc {
	allowedOrigins := cfg.AllowedOrigins
	allowedMethods := strings.Join(cfg.AllowedMethods, ", ")
	allowedHeaders := strings.Join(cfg.AllowedHeaders, ", ")

	originAllowed := func(origin string) bool {
		for _, o := range allowedOrigins {
			if o == "*" || o == origin {
				return true
			}
		}
		return false
	}

	return func(next server.HandlerFunc) server.HandlerFunc {
		return func(req *server.Request, res *server.ResponseWriter) {
			origin := req.Header("Origin")

			if origin != "" && originAllowed(origin) {
				res.SetHeader("Access-Control-Allow-Origin", origin)
				res.SetHeader("Access-Control-Allow-Methods", allowedMethods)
				res.SetHeader("Access-Control-Allow-Headers", allowedHeaders)
				res.SetHeader("Vary", "Origin")
			}

			// Handle preflight
			if req.Method == "OPTIONS" {
				res.WriteHeader(204)
				return
			}

			next(req, res)
		}
	}
}

// ---- Rate Limiter ----------------------------------------------------------

// RateLimitConfig holds rate limiter settings.
type RateLimitConfig struct {
	// RequestsPerSecond is the sustained rate per IP.
	RequestsPerSecond float64
	// Burst is the maximum instantaneous spike above the sustained rate.
	Burst int
}

// bucket is a token bucket for a single IP.
type bucket struct {
	tokens    float64
	lastRefil time.Time
	mu        sync.Mutex
}

// RateLimit returns middleware that limits requests per IP using a token bucket.
// Clients that exceed the limit receive 429 Too Many Requests.
func RateLimit(cfg RateLimitConfig) server.MiddlewareFunc {
	var mu sync.Mutex
	buckets := make(map[string]*bucket)

	getBucket := func(ip string) *bucket {
		mu.Lock()
		defer mu.Unlock()
		b, ok := buckets[ip]
		if !ok {
			b = &bucket{
				tokens:    float64(cfg.Burst),
				lastRefil: time.Now(),
			}
			buckets[ip] = b
		}
		return b
	}

	allow := func(ip string) bool {
		b := getBucket(ip)
		b.mu.Lock()
		defer b.mu.Unlock()

		now := time.Now()
		elapsed := now.Sub(b.lastRefil).Seconds()
		b.lastRefil = now

		// Refill tokens based on elapsed time
		b.tokens += elapsed * cfg.RequestsPerSecond
		if b.tokens > float64(cfg.Burst) {
			b.tokens = float64(cfg.Burst)
		}

		if b.tokens < 1 {
			return false
		}

		b.tokens--
		return true
	}

	return func(next server.HandlerFunc) server.HandlerFunc {
		return func(req *server.Request, res *server.ResponseWriter) {
			ip, _, err := net.SplitHostPort(req.RemoteAddr)
			if err != nil {
				ip = req.RemoteAddr
			}

			if !allow(ip) {
				res.SetHeader("Retry-After", "1")
				res.WriteText(429, "Too Many Requests")
				return
			}

			next(req, res)
		}
	}
}
