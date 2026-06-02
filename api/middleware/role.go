package middleware

import (
	"log/slog"
	"net/http"
)

func AdminMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role, ok := r.Context().Value(ContextKeyUserRole).(string)
		if !ok || role == "" {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		if role != "admin" {
			slog.Warn("Access denied: user is not admin",
				"user_id", r.Context().Value(ContextKeyUserID),
				"role", role,
			)
			http.Error(w, "Forbidden: Admins only", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}
