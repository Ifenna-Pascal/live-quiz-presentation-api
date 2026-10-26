package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter allows at most `limit` requests per `window` for each key
// (fixed window). It lives in memory, so each API instance counts on its own.
type RateLimiter struct {
	limit  int
	window time.Duration
	now    func() time.Time

	mu      sync.Mutex
	windows map[string]*rateWindow
	sweepAt time.Time
}

type rateWindow struct {
	start time.Time
	count int
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{limit: limit, window: window, now: time.Now, windows: map[string]*rateWindow{}}
}

// Allow records one request for key. When the limit is reached it returns
// false and how long until the window resets.
func (r *RateLimiter) Allow(key string) (bool, time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := r.now()

	// Drop finished windows now and then so the map cannot grow forever.
	if now.After(r.sweepAt) {
		for k, w := range r.windows {
			if now.Sub(w.start) >= r.window {
				delete(r.windows, k)
			}
		}
		r.sweepAt = now.Add(r.window)
	}

	w, ok := r.windows[key]
	if !ok || now.Sub(w.start) >= r.window {
		r.windows[key] = &rateWindow{start: now, count: 1}
		return true, 0
	}
	if w.count >= r.limit {
		return false, r.window - now.Sub(w.start)
	}
	w.count++
	return true, 0
}

// RateLimit is a gin middleware limiting requests per client IP. Limits are
// generous because a whole classroom can share one public IP.
func RateLimit(limit int, window time.Duration) gin.HandlerFunc {
	limiter := NewRateLimiter(limit, window)
	return func(c *gin.Context) {
		ok, retryAfter := limiter.Allow(c.ClientIP())
		if !ok {
			c.Header("Retry-After", strconv.Itoa(int(retryAfter.Seconds())+1))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "Too many requests. Please slow down and try again shortly."})
			return
		}
		c.Next()
	}
}
