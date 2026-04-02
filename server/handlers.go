package server

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// WebsocketHandler upgrades the connection and registers the client to the hub
func WebsocketHandler(hub *Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := c.Query("token")
		if tokenStr == "" {
			c.JSON(401, gin.H{"error": "Missing token"})
			return
		}

		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return JWTSecret, nil
		})
		if err != nil || !token.Valid {
			c.JSON(401, gin.H{"error": "Invalid token"})
			return
		}

		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("[WS] Upgrade failed: %v", err)
			return
		}

		client := &Client{
			Hub:  hub,
			Conn: conn,
			Send: make(chan []byte, 256),
		}
		hub.Register <- client

		go client.WritePump()
		go client.ReadPump()
	}
}

// LoginHandler verifies admin credentials and returns a JWT
func LoginHandler(db *Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid auth payload format"})
			return
		}

		if db.VerifyAdmin(req.Username, req.Password) {
			token, err := GenerateJWT(req.Username)
			if err != nil {
				c.JSON(500, gin.H{"error": "Server failed to forge JWT token"})
				return
			}
			c.JSON(200, gin.H{"token": token})
			return
		}

		c.JSON(401, gin.H{"error": "Invalid admin credentials"})
	}
}

// ThreatsLiveHandler pulls the historical threat feed for the dashboard
func ThreatsLiveHandler(db *Database) gin.HandlerFunc {
	return func(c *gin.Context) {
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
			attacks = []AttackEvent{}
		}
		c.JSON(200, attacks)
	}
}

// ThreatsStatsHandler pulls the leaderboards
func ThreatsStatsHandler(db *Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		topCountries, err := db.GetTopCountries(10)
		if err != nil {
			c.JSON(500, gin.H{"error": "failed to fetch stats"})
			return
		}
		if topCountries == nil {
			topCountries = []CountryStat{}
		}
		c.JSON(200, gin.H{"top_countries": topCountries})
	}
}
