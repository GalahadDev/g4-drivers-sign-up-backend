package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAdminMiddleware(t *testing.T) {
	tests := []struct {
		name       string
		setRole    bool
		role       string
		wantStatus int
		wantNext   bool
	}{
		{name: "no role in context", setRole: false, wantStatus: http.StatusForbidden, wantNext: false},
		{name: "empty role", setRole: true, role: "", wantStatus: http.StatusForbidden, wantNext: false},
		{name: "non-admin role", setRole: true, role: "driver", wantStatus: http.StatusForbidden, wantNext: false},
		{name: "admin role passes", setRole: true, role: "admin", wantStatus: http.StatusOK, wantNext: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nextCalled := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/admin", nil)
			if tt.setRole {
				ctx := context.WithValue(req.Context(), ContextKeyUserRole, tt.role)
				req = req.WithContext(ctx)
			}
			rec := httptest.NewRecorder()

			AdminMiddleware(next).ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if nextCalled != tt.wantNext {
				t.Errorf("next called = %v, want %v", nextCalled, tt.wantNext)
			}
		})
	}
}
