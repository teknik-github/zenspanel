package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimit returns Gin middleware that enforces a sliding-window per-IP
// rate limit. Used in front of /api/v1/auth/login to slow down brute-force
// attempts. The store is in-memory (sync.Map of timestamps); good enough
// for single-server deployments. Multi-server deployments should swap
// this out for a Redis-backed limiter.
//
// max — maximum requests per window.
// window — the sliding window length.
//
// On exceed, responds 429 with a Retry-After header pointing at the time
// the oldest counted request will fall out of the window.
func RateLimit(max int, window time.Duration) gin.HandlerFunc {
	limiter := newSlidingWindow(window)

	// Background pruner so the map doesn't grow unboundedly with one-shot
	// IPs. Pruning every window-length is enough — entries older than that
	// no longer affect any decision.
	go func() {
		t := time.NewTicker(window)
		defer t.Stop()
		for range t.C {
			limiter.prune()
		}
	}()

	return func(c *gin.Context) {
		ip := c.ClientIP()
		ok, retryAfter := limiter.allow(ip, max)
		if !ok {
			c.Header("Retry-After", retryAfter.UTC().Format(http.TimeFormat))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "too many requests, slow down",
			})
			return
		}
		c.Next()
	}
}

type slidingWindow struct {
	mu     sync.Mutex
	hits   map[string][]time.Time
	window time.Duration
}

func newSlidingWindow(window time.Duration) *slidingWindow {
	return &slidingWindow{
		hits:   make(map[string][]time.Time),
		window: window,
	}
}

// allow records a hit for key and returns whether it's within the limit.
// When the limit is exceeded, the second return value is the time at which
// the oldest counted hit will fall out of the window — clients can wait
// until that point to retry.
func (s *slidingWindow) allow(key string, max int) (bool, time.Time) {
	now := time.Now()
	cutoff := now.Add(-s.window)

	s.mu.Lock()
	defer s.mu.Unlock()

	pruned := s.hits[key][:0]
	for _, t := range s.hits[key] {
		if t.After(cutoff) {
			pruned = append(pruned, t)
		}
	}
	s.hits[key] = pruned

	if len(pruned) >= max {
		retry := pruned[0].Add(s.window)
		return false, retry
	}
	s.hits[key] = append(s.hits[key], now)
	return true, time.Time{}
}

// prune drops keys whose hits are all older than the window.
func (s *slidingWindow) prune() {
	cutoff := time.Now().Add(-s.window)
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, ts := range s.hits {
		// hits are append-only and ordered, so the last element is the
		// newest — if even that is older than cutoff, the whole slice is.
		if len(ts) == 0 || ts[len(ts)-1].Before(cutoff) {
			delete(s.hits, k)
		}
	}
}
