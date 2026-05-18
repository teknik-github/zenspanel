package middleware

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RateLimit returns Gin middleware that enforces a sliding-window per-IP
// rate limit. Used in front of /api/v1/auth/login to slow down brute-force
// attempts. The store is in-memory (sync.Map of timestamps); good enough
// for single-server deployments. Multi-server deployments should use
// RateLimitRedis instead.
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

// rateLimitLua is the atomic sliding-window check executed inside Redis.
// Doing the prune+count+add in one Lua call avoids the ZCARD/ZADD race
// that would otherwise let two concurrent requests both think they were
// under the limit.
//
// KEYS[1]   = ratelimit:<ip>
// ARGV[1]   = now (unix nanoseconds, also used as unique member name)
// ARGV[2]   = window length in nanoseconds
// ARGV[3]   = max requests per window
// ARGV[4]   = window length in seconds (for EXPIRE)
//
// Returns {1, 0}            when allowed
// Returns {0, oldest_score} when denied — caller computes Retry-After.
const rateLimitLua = `
local key = KEYS[1]
local now = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local max = tonumber(ARGV[3])
local ttl = tonumber(ARGV[4])
local cutoff = now - window

redis.call('ZREMRANGEBYSCORE', key, '-inf', cutoff)
local count = redis.call('ZCARD', key)
if count >= max then
  local oldest = redis.call('ZRANGE', key, 0, 0, 'WITHSCORES')
  return {0, oldest[2]}
end
redis.call('ZADD', key, now, now)
redis.call('EXPIRE', key, ttl)
return {1, 0}
`

// RateLimitRedis is the cluster-friendly version of RateLimit. State lives
// in a Redis sorted set per client IP; the algorithm is the same sliding
// window, but every API instance behind a load balancer reads/writes the
// same counter so an attacker can't spread their attempts across servers
// to bypass the limit.
//
// Fail-open: if Redis returns an error (network blip, auth failure,
// timeout), we allow the request and log a warning. Locking out every
// login during a Redis outage would be a worse failure mode than briefly
// relaxing rate limiting.
func RateLimitRedis(rdb *redis.Client, max int, window time.Duration) gin.HandlerFunc {
	script := redis.NewScript(rateLimitLua)
	ttlSeconds := int(window / time.Second)
	if ttlSeconds < 1 {
		ttlSeconds = 1
	}

	return func(c *gin.Context) {
		ip := c.ClientIP()
		ctx, cancel := context.WithTimeout(c.Request.Context(), 200*time.Millisecond)
		defer cancel()

		now := time.Now().UnixNano()
		raw, err := script.Run(ctx, rdb,
			[]string{"ratelimit:" + ip},
			now, int64(window), max, ttlSeconds,
		).Result()
		if err != nil {
			log.Printf("WARN: RateLimitRedis failed for %s (%v) — allowing request", ip, err)
			c.Next()
			return
		}
		arr, ok := raw.([]interface{})
		if !ok || len(arr) < 2 {
			log.Printf("WARN: RateLimitRedis unexpected return %v — allowing", raw)
			c.Next()
			return
		}
		allowed, _ := arr[0].(int64)
		if allowed == 1 {
			c.Next()
			return
		}
		// Denied — second element is the oldest timestamp in the window.
		// Retry-After = oldest + window. We store nanoseconds, so convert.
		oldestNs, _ := strconv.ParseInt(toString(arr[1]), 10, 64)
		retryAt := time.Unix(0, oldestNs).Add(window)
		c.Header("Retry-After", retryAt.UTC().Format(http.TimeFormat))
		c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
			"error": "too many requests, slow down",
		})
	}
}

// toString coerces the various concrete types Redis Lua return values can
// take (string, []byte, int64) into a string for ParseInt. The driver
// has historically returned strings, but newer versions occasionally
// return []byte for binary-safe data.
func toString(v interface{}) string {
	switch x := v.(type) {
	case string:
		return x
	case []byte:
		return string(x)
	case int64:
		return strconv.FormatInt(x, 10)
	}
	return ""
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
