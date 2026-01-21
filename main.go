package main

import (
	"log/slog"
	"net/http"
	"os"

	httpSwagger "github.com/swaggo/http-swagger/v2"

	"g4-services/api/config"
	"g4-services/api/database"
	"g4-services/api/middleware"

	"g4-services/api/handlers/admin"
	"g4-services/api/handlers/dashboard"
	"g4-services/api/handlers/drivers"
	"g4-services/api/handlers/users"

	_ "g4-services/docs"
)

// @title           G4 Drivers API
// @version         1.0
// @description     API for G4 Car Service Drivers Application.
// @termsOfService  http://swagger.io/terms/

// @contact.name    API Support
// @contact.email   samllachp@gmail.com

// @host            localhost:8080
// @BasePath        /api

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {

	// 1. Logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// 2. Config
	cfg, err := config.Load()
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	// 3. Init DB & Auth
	database.InitDB(cfg)
	defer database.Pool.Close()

	if err := middleware.InitAuth(cfg); err != nil {
		slog.Error("Failed to initialize Auth JWKS", "error", err)
		os.Exit(1)
	}

	// Router Principal
	mainMux := http.NewServeMux()

	// Router para la API (El Grupo)
	apiMux := http.NewServeMux()

	// --- MIDDLEWARE HELPERS ---
	protected := func(h http.HandlerFunc) http.Handler {
		return middleware.AuthMiddleware(h)
	}

	adminOnly := func(h http.HandlerFunc) http.Handler {
		return middleware.AuthMiddleware(middleware.AdminMiddleware(h))
	}

	strictLimit := func(h http.HandlerFunc) http.Handler {
		return middleware.RateLimitMiddleware(protected(h), 0.00055, 2)
	}

	mediumLimit := func(h http.HandlerFunc) http.Handler {
		return middleware.RateLimitMiddleware(protected(h), 0.083, 5)
	}

	standardLimit := func(h http.HandlerFunc) http.Handler {
		return middleware.RateLimitMiddleware(protected(h), 1, 10)
	}

	// --- RUTAS DE LA API  ---

	// Drivers & User
	apiMux.Handle("POST /drivers/register", strictLimit(drivers.RegisterDriver))
	apiMux.Handle("GET /user/dashboard", standardLimit(dashboard.GetMyDashboard))
	apiMux.Handle("PUT /user/profile", mediumLimit(users.UpdateMyProfile))
	apiMux.Handle("GET /user/me", standardLimit(users.GetMe))

	// Admin
	apiMux.Handle("GET /admin/users", adminOnly(admin.ListUsers))
	apiMux.Handle("GET /admin/stats", adminOnly(admin.GetGlobalStats))
	apiMux.Handle("GET /admin/user", adminOnly(admin.GetUserDetail))

	//  Health check
	mainMux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	mainMux.Handle("/api/", http.StripPrefix("/api", apiMux))

	mainMux.Handle("/swagger/", httpSwagger.WrapHandler)

	slog.Info("Server starting", "port", cfg.Port)

	handler := middleware.RequestLogger(middleware.CORSMiddleware(mainMux))

	if err := http.ListenAndServe(":"+cfg.Port, handler); err != nil {
		slog.Error("Server failed", "error", err)
		os.Exit(1)
	}
}
