package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// IPLimiter holds a per-IP token bucket limiter.
type IPLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
}

func NewIPLimiter() *IPLimiter {
	return &IPLimiter{limiters: make(map[string]*rate.Limiter)}
}

// get returns (or creates) a limiter for the given IP.
// Allows 5 requests per minute with a burst of 5.
func (l *IPLimiter) get(ip string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()
	if lim, ok := l.limiters[ip]; ok {
		return lim
	}
	lim := rate.NewLimiter(rate.Every(time.Minute/5), 5)
	l.limiters[ip] = lim
	return lim
}

func RateLimitMiddleware(l *IPLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !l.get(c.ClientIP()).Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests, please try again later"})
			return
		}
		c.Next()
	}
}
