package api

import (
	"net/http"
	"sync"
	"time"
)

// RateLimiter implements an in-memory per-key token bucket rate limiter.
// Keys are typically user IDs (authenticated) or IP addresses (public).
type RateLimiter struct {
	mu       sync.Mutex
	buckets  map[string]*bucket
	rate     int           // tokens added per interval
	burst    int           // max tokens in the bucket
	interval time.Duration // how often tokens are replenished
	cleanup  time.Duration // remove stale entries after this
}

type bucket struct {
	tokens   int
	lastSeen time.Time
}

// NewRateLimiter creates a rate limiter.
//
// rate: number of requests allowed per interval.
// burst: maximum burst capacity.
// interval: replenishment window (e.g. 1 * time.Minute).
func NewRateLimiter(rate, burst int, interval time.Duration) *RateLimiter {
	rl := &RateLimiter{
		buckets:  make(map[string]*bucket),
		rate:     rate,
		burst:    burst,
		interval: interval,
		cleanup:  10 * time.Minute,
	}
	go rl.cleanupLoop()
	return rl
}

// Allow checks whether a request from the given key is permitted.
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	b, exists := rl.buckets[key]
	now := time.Now()

	if !exists {
		rl.buckets[key] = &bucket{tokens: rl.burst - 1, lastSeen: now}
		return true
	}

	// Replenish tokens based on elapsed time.
	elapsed := now.Sub(b.lastSeen)
	tokensToAdd := int(elapsed/rl.interval) * rl.rate
	if tokensToAdd > 0 {
		b.tokens += tokensToAdd
		if b.tokens > rl.burst {
			b.tokens = rl.burst
		}
		b.lastSeen = now
	}

	if b.tokens <= 0 {
		return false
	}

	b.tokens--
	return true
}

// Middleware wraps an http.Handler with rate limiting.
// Uses the authenticated user ID if available, otherwise the remote IP.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key, _ := userFromContext(r.Context())
		if key == "" {
			key = r.RemoteAddr
		}

		if !rl.Allow(key) {
			w.Header().Set("Retry-After", "60")
			writeError(w, http.StatusTooManyRequests, "Rate limit exceeded. Try again later.")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// cleanupLoop periodically removes stale entries.
func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.cleanup)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		cutoff := time.Now().Add(-rl.cleanup)
		for key, b := range rl.buckets {
			if b.lastSeen.Before(cutoff) {
				delete(rl.buckets, key)
			}
		}
		rl.mu.Unlock()
	}
}
