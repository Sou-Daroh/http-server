package middleware

import (
	"strings"
	"time"

	"github.com/Sou-Daroh/http-server/server"
)

// Honeypot returns middleware that intercepts suspicious requests
// and logs them to the sqlite database before they hit the real backend.
func Honeypot(db *server.Database, geoip *server.GeoIP) server.MiddlewareFunc {
	// List of paths that bots and hackers actively scan for
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
		"jndi:", // Added Log4Shell JNDI injection footprint
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

	return func(next server.HandlerFunc) server.HandlerFunc {
		return func(req *server.Request, res *server.ResponseWriter) {
			// Never block legitimate API calls to the dashboard
			if strings.HasPrefix(req.Path, "/api/threats") || req.Path == "/" || strings.HasPrefix(req.Path, "/static") || strings.HasPrefix(req.Path, "/dashboard") {
				next(req, res)
				return
			}

			// We treat EVERYTHING else as a potential threat probe for the honeypot
			if isSuspicious(req.Path, req.RawQuery) {
				// Record the attack
				ip := extractIP(req.RemoteAddr)
                
				// Construct payload fingerprint
				payload := req.Method + " " + req.Path
				if len(req.RawQuery) > 0 {
					payload += "?" + req.RawQuery
				}
				
				if len(req.Body) > 0 {
					bodyStr := string(req.Body)
					if len(bodyStr) > 200 {
                         // Truncate massive binary payloads
						bodyStr = bodyStr[:200] + "..."
					}
					payload += "\n" + bodyStr
				}

				targetPath := req.Path

				// Fire and forget the geoip lookup and database save so we don't block the high-volume TCP socket
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

				// Return a fake positive response to keep them attacking
				res.WriteText(200, "# Honeypot Access Granted\n{\n  \"status\": \"success\"\n}")
				return
			}

			// Allow normal traffic to pass through
			next(req, res)
		}
	}
}

// extractIP pulls out the IP from net.Addr strings like "10.0.0.1:45312"
func extractIP(remoteAddr string) string {
	if idx := strings.LastIndex(remoteAddr, ":"); idx != -1 {
		// Assuming IPv4 format mostly, avoiding port. Real logic would use net.SplitHostPort safely
		return remoteAddr[:idx]
	}
	return remoteAddr
}
