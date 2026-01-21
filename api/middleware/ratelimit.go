package middleware

import (
	"log/slog"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// IPRateLimiter gestiona límites por IP o Usuario
type IPRateLimiter struct {
	ips map[string]*rate.Limiter
	mu  *sync.RWMutex
	r   rate.Limit
	b   int
}

// NewIPRateLimiter crea un limitador que permite r requests por segundo con ráfaga de b
func NewIPRateLimiter(r rate.Limit, b int) *IPRateLimiter {
	i := &IPRateLimiter{
		ips: make(map[string]*rate.Limiter),
		mu:  &sync.RWMutex{},
		r:   r,
		b:   b,
	}
	go i.cleanup()

	return i
}

// AddIP crea un limitador para una key si no existe
func (i *IPRateLimiter) GetLimiter(key string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()

	limiter, exists := i.ips[key]
	if !exists {
		limiter = rate.NewLimiter(i.r, i.b)
		i.ips[key] = limiter
	}

	return limiter
}

// cleanup elimina entradas viejas
func (i *IPRateLimiter) cleanup() {
	for {
		time.Sleep(1 * time.Hour)
		i.mu.Lock()
		slog.Info("Cleaning up rate limit map", "entries", len(i.ips))
		i.ips = make(map[string]*rate.Limiter)
		i.mu.Unlock()
	}
}

func RateLimitMiddleware(next http.Handler, rps float64, burst int) http.Handler {
	limiter := NewIPRateLimiter(rate.Limit(rps), burst)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		key := r.RemoteAddr

		if userID, ok := r.Context().Value("user_id").(string); ok {
			key = userID
		}

		l := limiter.GetLimiter(key)

		if !l.Allow() {
			slog.Warn("Rate limit exceeded", "key", key, "path", r.URL.Path)
			http.Error(w, "Too Many Requests - Rate Limit Exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}
