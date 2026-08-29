package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type ipBucket struct {
	tokens     float64
	lastRefill time.Time
}

// RateLimit returns a middleware that limits requests to maxReq per window per client IP
// using an in-memory token bucket rate limiter.
func RateLimit(maxReq int, window time.Duration) func(http.Handler) http.Handler {
	if maxReq <= 0 {
		maxReq = 120
	}
	if window <= 0 {
		window = time.Minute
	}

	refillRate := float64(maxReq) / window.Seconds() // tokens per second
	capacity := float64(maxReq)

	var mu sync.Mutex
	buckets := make(map[string]*ipBucket)

	// Background sweeper to purge inactive IPs
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for now := range ticker.C {
			mu.Lock()
			for ip, b := range buckets {
				if now.Sub(b.lastRefill) > window*2 {
					delete(buckets, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := extractIP(r)

			now := time.Now()

			mu.Lock()
			b, exists := buckets[ip]
			if !exists {
				b = &ipBucket{
					tokens:     capacity - 1.0,
					lastRefill: now,
				}
				buckets[ip] = b
				mu.Unlock()
				next.ServeHTTP(w, r)
				return
			}

			// Refill tokens based on elapsed time
			elapsed := now.Sub(b.lastRefill).Seconds()
			b.tokens = b.tokens + (elapsed * refillRate)
			if b.tokens > capacity {
				b.tokens = capacity
			}
			b.lastRefill = now

			if b.tokens < 1.0 {
				mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", "1")
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(`{"error":"rate limit exceeded, please retry later"}`))
				return
			}

			b.tokens -= 1.0
			mu.Unlock()

			next.ServeHTTP(w, r)
		})
	}
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
