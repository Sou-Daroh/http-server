package middleware

import (
	"log"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	// StrikeCache temporarily records the number of times an IP hits the honeypot
	StrikeCache sync.Map

	// BannedCache acts as our high-speed Web Application Firewall blocklist
	BannedCache sync.Map
)

const MaxStrikes = 3

// RegisterStrike logs a malicious hit. If they pass the threshold, they are moved to the BannedCache.
func RegisterStrike(ip string) {
	// Increment strike counter
	current, _ := StrikeCache.LoadOrStore(ip, 0)
	strikes := current.(int) + 1
	StrikeCache.Store(ip, strikes)

	if strikes >= MaxStrikes {
		BannedCache.Store(ip, time.Now())
		log.Printf("[WAF BAN HAMMER] IP %s just hit %d strikes. Connection access completely severed.", ip, strikes)
	}
}

// BlacklistInterceptor is the very first middleware in the routing chain.
// It acts as a Web Application Firewall that instantly drops traffic.
func BlacklistInterceptor() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		ip = strings.Trim(ip, "[]")

		// High-speed O(1) cache lookup bypasses any heavy database latency
		if _, banned := BannedCache.Load(ip); banned {
			c.JSON(403, gin.H{
				"error":   "Forbidden",
				"message": "Your IP has been permanently blacklisted for malicious activity.",
			})
			c.Abort() // Completely tear down the connection loop
			return
		}

		c.Next()
	}
}
