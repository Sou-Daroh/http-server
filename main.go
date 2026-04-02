package main

import (
	"fmt"
	"log"
	"os"

	"github.com/Sou-Daroh/http-server/middleware"
	"github.com/Sou-Daroh/http-server/server"
	"github.com/gin-gonic/gin"
)

func main() {
	fmt.Println("Initializing Postgres Threat Intelligence Engine...")

	// Read Docker DSN or fallback config
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		dsn = "postgres://admin:supersecretpassword@localhost:5432/honeypot?sslmode=disable"
	}

	db, err := server.NewDatabase(dsn)
	if err != nil {
		log.Fatalf("fatal: failed to open postgres database: %v", err)
	}

	geoip, err := server.NewGeoIP("GeoLite2-City.mmdb")
	if err != nil {
		fmt.Printf("warning: maxmind geoip db skipped, using api fallback\n")
	}

	// Initialize Gin Routing Engine
	r := gin.Default()

	// --- Inject Web Application Firewall (Ban Hammer) ---
	r.Use(middleware.BlacklistInterceptor())

	// Initialize WebSocket Hub (Go Concurrency Showcase)
	hub := server.NewHub()
	go hub.Run()

	// --- Inject Gin Honeypot Trap ---
	r.Use(middleware.Honeypot(db, geoip, hub))

	// --- Public Routes ---
	r.Static("/static", "./static")
	r.Static("/assets", "./static/assets") // Map Vue's Vite asset bundler paths dynamically
	r.GET("/", func(c *gin.Context) {
		c.Redirect(301, "/static/")
	})

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "honeypot": "armed"})
	})

	// --- WebSocket Real-Time Threat Feed ---
	r.GET("/ws", server.WebsocketHandler(hub))

	// Dashboard Authentication Gateway
	r.POST("/api/login", server.LoginHandler(db))

	// --- Protected Advanced Training APIs ---
	api := r.Group("/api")
	api.Use(middleware.JWTAuthMiddleware())
	{
		api.GET("/threats/live", server.ThreatsLiveHandler(db))
		api.GET("/threats/stats", server.ThreatsStatsHandler(db))
		api.POST("/simulate", server.SimulateAttack(hub))
	}

	// --- Start Gin ---
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	r.Run(":" + port)
}
