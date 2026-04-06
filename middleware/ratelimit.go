package middleware

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// IPLimiter holds a per-IP token bucket limiter.
type IPLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
	r        rate.Limit
	burst    int
}

// NewIPLimiter creates a limiter allowing r tokens/sec with the given burst size.
// Pass rate.Inf as r to disable limiting entirely (useful for load testing).
func NewIPLimiter(r rate.Limit, burst int) *IPLimiter {
	return &IPLimiter{
		limiters: make(map[string]*rate.Limiter),
		r:        r,
		burst:    burst,
	}
}

// get returns (or creates) a limiter for the given IP.
func (l *IPLimiter) get(ip string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()
	if lim, ok := l.limiters[ip]; ok {
		return lim
	}
	lim := rate.NewLimiter(l.r, l.burst)
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
