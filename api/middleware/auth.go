package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"g4-services/api/config"

	"github.com/MicahParks/keyfunc/v2"
	"github.com/golang-jwt/jwt/v5"
)

var jwks *keyfunc.JWKS

func InitAuth(cfg *config.AppConfig) error {

	jwksURL := fmt.Sprintf("%s/auth/v1/.well-known/jwks.json", cfg.SupabaseURL)

	slog.Info("Initializing Auth", "jwks_url", jwksURL)

	options := keyfunc.Options{
		RefreshErrorHandler: func(err error) {
			slog.Error("Error refreshing JWKS", "error", err)
		},
		RefreshInterval:   time.Hour * 1,
		RefreshRateLimit:  time.Minute * 5,
		RefreshTimeout:    time.Second * 10,
		RefreshUnknownKID: true,
	}

	var err error
	jwks, err = keyfunc.Get(jwksURL, options)
	if err != nil {
		return fmt.Errorf("failed to load JWKS: %w", err)
	}

	return nil
}

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Missing Authorization Header", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			http.Error(w, "Invalid Token Format", http.StatusUnauthorized)
			return
		}

		token, err := jwt.Parse(tokenString, jwks.Keyfunc)
		if err != nil {
			slog.Warn("Token validation failed", "error", err)
			http.Error(w, "Invalid Token", http.StatusUnauthorized)
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			userID, ok := claims["sub"].(string)
			if !ok {
				http.Error(w, "Invalid Token Claims", http.StatusUnauthorized)
				return
			}

			// Try Auth Hook claim first, then fall back to app_metadata.role (standard Supabase).
			userRole, _ := claims["user_role"].(string)
			if userRole == "" {
				if appMeta, ok := claims["app_metadata"].(map[string]any); ok {
					userRole, _ = appMeta["role"].(string)
				}
			}
			if userRole == "" {
				userRole = "driver"
			}

			ctx := context.WithValue(r.Context(), ContextKeyUserID, userID)
			ctx = context.WithValue(ctx, ContextKeyUserRole, userRole)
			next.ServeHTTP(w, r.WithContext(ctx))
		} else {
			http.Error(w, "Invalid Token", http.StatusUnauthorized)
		}
	})
}
