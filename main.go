package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

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

	// --- Inject Gin Honeypot Trap ---
	r.Use(middleware.Honeypot(db, geoip))

	// --- Public Routes ---
	r.Static("/static", "./static")
	r.GET("/", func(c *gin.Context) {
		c.Redirect(301, "/static/")
	})

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "honeypot": "armed"})
	})

	// --- Protected Advanced Training APIs ---
	api := r.Group("/api")
	api.Use(middleware.JWTAuthMiddleware())
	{
		api.GET("/threats/live", func(c *gin.Context) {
			limit := 50
			if limitParam := c.Query("limit"); limitParam != "" {
				if l, err := strconv.Atoi(limitParam); err == nil && l > 0 && l <= 1000 {
					limit = l
				}
			}

			attacks, err := db.GetRecentAttacks(limit)
			if err != nil {
				c.JSON(500, gin.H{"error": "failed to fetch threat log"})
				return
			}
			if attacks == nil {
				attacks = []server.AttackEvent{}
			}
			c.JSON(200, attacks)
		})

		api.GET("/threats/stats", func(c *gin.Context) {
			topCountries, err := db.GetTopCountries(10)
			if err != nil {
				c.JSON(500, gin.H{"error": "failed to fetch stats"})
				return
			}
			if topCountries == nil {
				topCountries = []server.CountryStat{}
			}
			c.JSON(200, gin.H{"top_countries": topCountries})
		})
	}

	// --- Start Gin ---
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	r.Run(":" + port)
}
