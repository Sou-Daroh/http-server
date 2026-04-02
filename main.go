package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/Sou-Daroh/http-server/middleware"
	"github.com/Sou-Daroh/http-server/server"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
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
	var upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	r.GET("/ws", func(c *gin.Context) {
		// Validate JWT from query string (WebSocket can't send headers)
		tokenStr := c.Query("token")
		if tokenStr == "" {
			c.JSON(401, gin.H{"error": "Missing token"})
			return
		}

		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return server.JWTSecret, nil
		})
		if err != nil || !token.Valid {
			c.JSON(401, gin.H{"error": "Invalid token"})
			return
		}

		// Upgrade HTTP → WebSocket
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("[WS] Upgrade failed: %v", err)
			return
		}

		client := &server.Client{
			Hub:  hub,
			Conn: conn,
			Send: make(chan []byte, 256),
		}
		hub.Register <- client

		go client.WritePump()
		go client.ReadPump()
	})

	// Dashboard Authentication Gateway
	r.POST("/api/login", func(c *gin.Context) {
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid auth payload format"})
			return
		}

		if db.VerifyAdmin(req.Username, req.Password) {
			token, err := server.GenerateJWT(req.Username)
			if err != nil {
				c.JSON(500, gin.H{"error": "Server failed to forge JWT token"})
				return
			}
			// Respond with the token so the Vue browser can cache it
			c.JSON(200, gin.H{"token": token})
			return
		}

		c.JSON(401, gin.H{"error": "Invalid admin credentials"})
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

		api.POST("/simulate", server.SimulateAttack(hub))
	}

	// --- Start Gin ---
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	r.Run(":" + port)
}
