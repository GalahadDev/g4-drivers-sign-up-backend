package middleware

import (
	"log/slog"
	"net/http"

	"g4-services/api/database"
)

func AdminMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		userID, ok := r.Context().Value("user_id").(string)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var role string
		err := database.Pool.QueryRow(r.Context(), "SELECT role FROM profiles WHERE id = $1", userID).Scan(&role)

		if err != nil {
			slog.Error("Error checking admin role", "error", err)
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		if role != "admin" {
			slog.Warn("Access denied: User is not admin", "user_id", userID)
			http.Error(w, "Forbidden: Admins only", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}
