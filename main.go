package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/Sou-Daroh/http-server/middleware"
	"github.com/Sou-Daroh/http-server/server"
)

func main() {
	// --- CLI flags ---
	configPath := flag.String("config", "config.json", "Path to config file")
	portOverride := flag.Int("port", 0, "Override config port")
	dbPath := flag.String("db", "honeypot.db", "Path to sqlite database")
	geoipPath := flag.String("geoip", "GeoLite2-City.mmdb", "Path to MaxMind GeoIP database")
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

	// --- Initialize Intelligence Engines ---
	fmt.Println("Initializing Threat Intelligence Databases...")
	db, err := server.NewDatabase(*dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: failed to open sqlite database: %v\n", err)
		os.Exit(1)
	}

	geoip, err := server.NewGeoIP(*geoipPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: maxmind geoip db failed to load, falling back to api: %v\n", err)
	}

	// --- Build server ---
	s := server.New(server.Options{
		Host:         cfg.Host,
		Port:         cfg.Port,
		StaticDir:    cfg.StaticDir, // This will host our Vue.js SPA
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

	// --- Inject Honeypot Trap ---
	// Always place honeypot AFTER logger/cors but BEFORE routing standard requests
	s.Use(middleware.Honeypot(db, geoip))

	// --- Register routes ---
	router := s.Router()

	// System Health check
	router.GET("/health", func(req *server.Request, res *server.ResponseWriter) {
		res.WriteJSON(200, map[string]string{
			"status":   "ok",
			"honeypot": "armed",
		})
	})

	// --- Threat Intelligence Dashboard APIs ---

	// GET /api/threats/live - Returns the latest captured attacks
	router.GET("/api/threats/live", func(req *server.Request, res *server.ResponseWriter) {
		limit := 50
		if req.Query["limit"] != "" {
			if l, err := strconv.Atoi(req.Query["limit"]); err == nil && l > 0 && l <= 1000 {
				limit = l
			}
		}

		attacks, err := db.GetRecentAttacks(limit)
		if err != nil {
			res.WriteJSON(500, map[string]string{"error": "failed to fetch threat log"})
			return
		}

		if attacks == nil {
			attacks = []server.AttackEvent{} // Ensure JSON array instead of null
		}
		res.WriteJSON(200, attacks)
	})

	// GET /api/threats/stats - Returns leaderboards and metrics
	router.GET("/api/threats/stats", func(req *server.Request, res *server.ResponseWriter) {
		topCountries, err := db.GetTopCountries(10)
		if err != nil {
			res.WriteJSON(500, map[string]string{"error": "failed to fetch stats"})
			return
		}

		if topCountries == nil {
			topCountries = []server.CountryStat{}
		}

		res.WriteJSON(200, map[string]any{
			"top_countries": topCountries,
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
		fmt.Println("\nshutting down safely...")
		cancel()
	}()

	// --- Start ---
	if err := s.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("server stopped")
}
