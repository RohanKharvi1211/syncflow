package middlewares

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter provides simple in-memory rate limiting
type RateLimiter struct {
	visitors map[string]*Visitor
	mu       sync.RWMutex
	rate     int           // requests per minute
	window   time.Duration // time window (default: 1 minute)
}

// Visitor tracks request count and last reset time for an IP
type Visitor struct {
	count     int
	resetTime time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(rate int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*Visitor),
		rate:     rate,
		window:   window,
	}

	// Cleanup old visitors every 5 minutes
	go rl.cleanup()

	return rl
}

// cleanup removes old visitor entries
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, visitor := range rl.visitors {
			if now.After(visitor.resetTime.Add(rl.window * 2)) {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// Allow checks if a request from the given IP should be allowed
func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	visitor, exists := rl.visitors[ip]

	if !exists {
		// First visit
		rl.visitors[ip] = &Visitor{
			count:     1,
			resetTime: now,
		}
		return true
	}

	// Reset counter if window has passed
	if now.After(visitor.resetTime.Add(rl.window)) {
		visitor.count = 1
		visitor.resetTime = now
		return true
	}

	// Increment counter
	visitor.count++

	// Check if limit exceeded
	if visitor.count > rl.rate {
		return false
	}

	return true
}

// Global rate limiter instance
var globalRateLimiter *RateLimiter

func init() {
	// Default: 60 requests per minute per IP
	// Adjust these values based on your needs
	globalRateLimiter = NewRateLimiter(60, time.Minute)
}

// RateLimit middleware limits requests per IP
func RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip rate limiting for health checks
		if c.Request.URL.Path == "/health" {
			c.Next()
			return
		}

		// Get client IP
		ip := c.ClientIP()

		// Allow the request if within rate limit
		if !globalRateLimiter.Allow(ip) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "Rate limit exceeded",
				"message": "Too many requests from this IP. Please try again later.",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

