package middleware

import (
	"bytes"
	"io"
	"strings"
	"time"

	"github.com/Sou-Daroh/http-server/server"
	"github.com/gin-gonic/gin"
)

// Honeypot returns a Gin middleware that intercepts suspicious requests
// and logs them to PostgreSQL before they hit the real backend APIs.
func Honeypot(db *server.Database, geoip *server.GeoIP) gin.HandlerFunc {
	suspiciousStubs := []string{
		".env",
		"wp-admin",
		"wp-login",
		"config.php",
		"passwd",
		"mysql",
		"phpmyadmin",
		"setup.php",
		"actuator",
		"swagger-ui",
		"cmd=",
		"exec=",
		"jndi:", // Log4Shell footprint
	}

	isSuspicious := func(path, rawQuery string) bool {
		lowerPath := strings.ToLower(path)
		lowerQuery := strings.ToLower(rawQuery)
		for _, stub := range suspiciousStubs {
			if strings.Contains(lowerPath, stub) || strings.Contains(lowerQuery, stub) {
				return true
			}
		}
		return false
	}

	return func(c *gin.Context) {
		path := c.Request.URL.Path
		rawQuery := c.Request.URL.RawQuery

		// Never block legitimate API calls to the dashboard
		if strings.HasPrefix(path, "/api/threats") || path == "/" || strings.HasPrefix(path, "/static") || strings.HasPrefix(path, "/dashboard") {
			c.Next()
			return
		}

		// We treat EVERYTHING else as a potential threat probe for the honeypot
		if isSuspicious(path, rawQuery) {
			// Gin natively handles proper proxy header ip extraction
			ip := c.ClientIP()
			ip = strings.Trim(ip, "[]") // Sanitize IPv6 brackets for GeoIP 

			payload := c.Request.Method + " " + path
			if len(rawQuery) > 0 {
				payload += "?" + rawQuery
			}

			// We only parse the body if we've flagged an attack, ensuring we don't break downstream handlers
			bodyBytes, _ := io.ReadAll(c.Request.Body)
			if len(bodyBytes) > 0 {
				bodyStr := string(bodyBytes)
				if len(bodyStr) > 200 {
					bodyStr = bodyStr[:200] + "..." // Truncate giant malicious blobs
				}
				payload += "\n" + bodyStr
			}

			// Re-inject the body into Gin's context just in case
			c.Request.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			targetPath := path

			// Concurrency Metric: Fire off Goroutine so the Hacker connection isn't blocked by Postgres latency
			go func() {
				loc := geoip.Lookup(ip)

				db.LogAttack(server.AttackEvent{
					IP:        ip,
					Country:   loc.Country,
					City:      loc.City,
					Lat:       loc.Lat,
					Lon:       loc.Lon,
					Payload:   payload,
					Target:    targetPath,
					Timestamp: time.Now(),
				})
			}()

			// Trap the attacker by returning a false 200 OK
			c.JSON(200, gin.H{"status": "success", "message": "# Honeypot Access Granted"})
			c.Abort()
			return
		}

		// Allow normal safe traffic
		c.Next()
	}
}
