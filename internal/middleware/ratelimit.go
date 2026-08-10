package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/asnakech/asnakech-servers/internal/apperr"
	"github.com/asnakech/asnakech-servers/internal/response"
	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// IPRateLimiter limits requests per client IP using a token bucket.
type IPRateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
	r        rate.Limit
	b        int
}

func NewIPRateLimiter(requestsPerSecond float64, burst int) *IPRateLimiter {
	return &IPRateLimiter{
		limiters: make(map[string]*rate.Limiter),
		r:        rate.Limit(requestsPerSecond),
		b:        burst,
	}
}

func (l *IPRateLimiter) get(ip string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()

	lim, ok := l.limiters[ip]
	if !ok {
		lim = rate.NewLimiter(l.r, l.b)
		l.limiters[ip] = lim
	}
	return lim
}

// Limit returns middleware that rejects excess requests with 429.
func (l *IPRateLimiter) Limit() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !l.get(c.ClientIP()).Allow() {
			c.Header("Retry-After", "1")
			response.FailCode(c, http.StatusTooManyRequests, apperr.CodeRateLimited, "too many requests")
			c.Abort()
			return
		}
		c.Next()
	}
}

// CleanupPeriodically drops idle limiter entries (best-effort memory control).
func (l *IPRateLimiter) CleanupPeriodically(every time.Duration, stop <-chan struct{}) {
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			l.mu.Lock()
			l.limiters = make(map[string]*rate.Limiter)
			l.mu.Unlock()
		}
	}
}
