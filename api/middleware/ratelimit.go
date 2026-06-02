package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type limiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type IPRateLimiter struct {
	entries map[string]*limiterEntry
	mu      sync.Mutex
	r       rate.Limit
	b       int
}

// NewIPRateLimiter creates a rate limiter. The cleanup goroutine runs until ctx is cancelled.
func NewIPRateLimiter(ctx context.Context, r rate.Limit, b int) *IPRateLimiter {
	rl := &IPRateLimiter{
		entries: make(map[string]*limiterEntry),
		r:       r,
		b:       b,
	}
	go rl.cleanup(ctx)
	return rl
}

func (rl *IPRateLimiter) GetLimiter(key string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	e, exists := rl.entries[key]
	if !exists {
		e = &limiterEntry{limiter: rate.NewLimiter(rl.r, rl.b)}
		rl.entries[key] = e
	}
	e.lastSeen = time.Now()
	return e.limiter
}

func (rl *IPRateLimiter) cleanup(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			rl.mu.Lock()
			const ttl = time.Hour
			for key, e := range rl.entries {
				if time.Since(e.lastSeen) > ttl {
					delete(rl.entries, key)
				}
			}
			slog.Info("Rate limiter cleanup done", "remaining_entries", len(rl.entries))
			rl.mu.Unlock()
		}
	}
}

// RateLimitMiddleware applies the given limiter to each request, keyed by user ID or remote IP.
func RateLimitMiddleware(limiter *IPRateLimiter, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.RemoteAddr
		if userID, ok := r.Context().Value(ContextKeyUserID).(string); ok && userID != "" {
			key = userID
		}

		if !limiter.GetLimiter(key).Allow() {
			slog.Warn("Rate limit exceeded", "key", key, "path", r.URL.Path)
			w.Header().Set("Retry-After", "60")
			http.Error(w, "Too Many Requests - Rate Limit Exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}
