package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Sou-Daroh/http-server/middleware"
	"github.com/Sou-Daroh/http-server/server"
)

func main() {
	// --- CLI flags ---
	configPath := flag.String("config", "config.json", "Path to config file")
	portOverride := flag.Int("port", 0, "Override config port")
	flag.Parse()

	// --- Load config ---
	cfg, err := LoadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading config: %v\n", err)
		os.Exit(1)
	}

	if *portOverride != 0 {
		cfg.Port = *portOverride
	}

	// --- Build server ---
	s := server.New(server.Options{
		Host:         cfg.Host,
		Port:         cfg.Port,
		StaticDir:    cfg.StaticDir,
		ReadTimeout:  time.Duration(cfg.Timeout.Read) * time.Second,
		WriteTimeout: time.Duration(cfg.Timeout.Write) * time.Second,
		IdleTimeout:  time.Duration(cfg.Timeout.Idle) * time.Second,
	})

	// --- Register middleware ---
	s.Use(middleware.Logger(middleware.LoggerConfig{
		LogFile: cfg.LogFile,
	}))

	if cfg.CORS.Enabled {
		s.Use(middleware.CORS(middleware.CORSConfig{
			AllowedOrigins: cfg.CORS.AllowedOrigins,
			AllowedMethods: cfg.CORS.AllowedMethods,
			AllowedHeaders: cfg.CORS.AllowedHeaders,
		}))
	}

	if cfg.RateLimit.Enabled {
		s.Use(middleware.RateLimit(middleware.RateLimitConfig{
			RequestsPerSecond: float64(cfg.RateLimit.RequestsPerSecond),
			Burst:             cfg.RateLimit.Burst,
		}))
	}

	// --- Register routes ---
	router := s.Router()

	// Health check
	router.GET("/health", func(req *server.Request, res *server.ResponseWriter) {
		res.WriteJSON(200, map[string]string{
			"status": "ok",
		})
	})

	// Example API route
	router.GET("/api/hello", func(req *server.Request, res *server.ResponseWriter) {
		name := req.Query["name"]
		if name == "" {
			name = "World"
		}
		res.WriteJSON(200, map[string]string{
			"message": "Hello, " + name + "!",
		})
	})

	// Echo route — returns request info as JSON (useful for debugging)
	router.GET("/api/echo", func(req *server.Request, res *server.ResponseWriter) {
		res.WriteJSON(200, map[string]any{
			"method":      req.Method,
			"path":        req.Path,
			"query":       req.Query,
			"remote_addr": req.RemoteAddr,
			"headers":     req.Headers,
		})
	})

	// --- Graceful shutdown ---
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Listen for SIGINT / SIGTERM
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		fmt.Println("\nshutting down...")
		cancel()
	}()

	// --- Start ---
	if err := s.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("server stopped")
}
