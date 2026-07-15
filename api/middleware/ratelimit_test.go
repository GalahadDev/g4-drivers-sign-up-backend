package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/time/rate"
)

func TestIPRateLimiter_GetLimiter(t *testing.T) {
	ctx := t.Context()
	rl := NewIPRateLimiter(ctx, rate.Limit(1), 1)

	a := rl.GetLimiter("1.2.3.4")
	again := rl.GetLimiter("1.2.3.4")
	if a != again {
		t.Error("expected the same limiter instance for the same key")
	}

	other := rl.GetLimiter("5.6.7.8")
	if a == other {
		t.Error("expected different limiter instances for different keys")
	}
}

func TestRateLimitMiddleware(t *testing.T) {
	ctx := t.Context()
	// burst of 1: the first request passes, the second is throttled.
	rl := NewIPRateLimiter(ctx, rate.Limit(1), 1)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := RateLimitMiddleware(rl, next)

	newReq := func() *http.Request {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "9.9.9.9:1234"
		return req
	}

	first := httptest.NewRecorder()
	handler.ServeHTTP(first, newReq())
	if first.Code != http.StatusOK {
		t.Fatalf("first request: status = %d, want 200", first.Code)
	}

	second := httptest.NewRecorder()
	handler.ServeHTTP(second, newReq())
	if second.Code != http.StatusTooManyRequests {
		t.Errorf("second request: status = %d, want 429", second.Code)
	}
	if got := second.Header().Get("Retry-After"); got != "60" {
		t.Errorf("Retry-After = %q, want \"60\"", got)
	}
}

func TestRateLimitMiddleware_KeyedByUserID(t *testing.T) {
	ctx := t.Context()
	rl := NewIPRateLimiter(ctx, rate.Limit(1), 1)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := RateLimitMiddleware(rl, next)

	// Two different users sharing the same RemoteAddr must not throttle each other.
	for _, userID := range []string{"user-a", "user-b"} {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "10.0.0.1:1111"
		req = req.WithContext(context.WithValue(req.Context(), ContextKeyUserID, userID))

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("user %s: status = %d, want 200", userID, rec.Code)
		}
	}
}
