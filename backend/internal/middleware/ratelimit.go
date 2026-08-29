package middleware

import (
	"context"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

var tokenBucketScript = redis.NewScript(`
local key = KEYS[1]
local capacity = tonumber(ARGV[1])
local refill_rate = tonumber(ARGV[2])
local now = tonumber(ARGV[3])
local requested = 1

local b = redis.call("HMGET", key, "tokens", "last_refill")
local tokens = tonumber(b[1])
local last_refill = tonumber(b[2])

if not tokens then
    tokens = capacity
    last_refill = now
end

local elapsed = math.max(0, now - last_refill)
tokens = math.min(capacity, tokens + (elapsed * refill_rate))

if tokens < requested then
    redis.call("HMSET", key, "tokens", tokens, "last_refill", now)
    redis.call("EXPIRE", key, math.ceil(capacity / refill_rate) * 2)
    return -1
else
    tokens = tokens - requested
    redis.call("HMSET", key, "tokens", tokens, "last_refill", now)
    redis.call("EXPIRE", key, math.ceil(capacity / refill_rate) * 2)
    return tokens
end
`)

// RateLimiter wraps a Redis client for distributed rate limiting.
type RateLimiter struct {
	rdb        *redis.Client
	maxReq     int
	window     time.Duration
	refillRate float64
	capacity   float64
}

// NewRateLimiter creates a new Redis-backed RateLimiter.
func NewRateLimiter(rdb *redis.Client, maxReq int, window time.Duration) *RateLimiter {
	if maxReq <= 0 {
		maxReq = 120
	}
	if window <= 0 {
		window = time.Minute
	}
	return &RateLimiter{
		rdb:        rdb,
		maxReq:     maxReq,
		window:     window,
		refillRate: float64(maxReq) / window.Seconds(),
		capacity:   float64(maxReq),
	}
}

// Allow checks if the given key is allowed to perform an action.
func (rl *RateLimiter) Allow(ctx context.Context, key string) (bool, error) {
	now := float64(time.Now().UnixNano()) / 1e9
	res, err := tokenBucketScript.Run(ctx, rl.rdb, []string{key}, rl.capacity, rl.refillRate, now).Result()
	if err != nil {
		return false, err
	}
	tokens := res.(int64)
	return tokens >= 0, nil
}

// Middleware returns an HTTP middleware that limits requests per IP.
func (rl *RateLimiter) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := extractIP(r)
			key := "ratelimit:ip:" + ip

			allowed, err := rl.Allow(r.Context(), key)
			if err != nil {
				// Fail open on Redis errors
				next.ServeHTTP(w, r)
				return
			}

			if !allowed {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", "1")
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(`{"error":"rate limit exceeded, please retry later"}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RateLimit is a convenience constructor for the IP-based rate limiting middleware.
func RateLimit(rdb *redis.Client, maxReq int, window time.Duration) func(http.Handler) http.Handler {
	rl := NewRateLimiter(rdb, maxReq, window)
	return rl.Middleware()
}

func extractIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			ip := strings.TrimSpace(parts[0])
			if ip != "" {
				return ip
			}
		}
	}

	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		return strings.TrimSpace(xrip)
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}

	return r.RemoteAddr
}
