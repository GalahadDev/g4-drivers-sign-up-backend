package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	httpSwagger "github.com/swaggo/http-swagger/v2"

	"g4-services/api/config"
	"g4-services/api/database"
	"g4-services/api/middleware"

	"g4-services/api/handlers/admin"
	"g4-services/api/handlers/dashboard"
	"g4-services/api/handlers/drivers"
	"g4-services/api/handlers/users"
	"g4-services/api/handlers/vision"
	visionService "g4-services/api/services/vision"

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

	// 4. Vision Service
	visionSvc, err := visionService.NewVisionService()
	if err != nil {
		slog.Error("Failed to create Vision Service", "error", err)
	} else {
		defer visionSvc.Close()
	}

	// Main Router
	mainMux := http.NewServeMux()

	// API Router
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

	visionLimit := func(h http.HandlerFunc) http.Handler {
		return middleware.RateLimitMiddleware(protected(h), 0.0833, 3)
	}

	// --- API ROUTES  ---

	// Drivers & User
	apiMux.Handle("POST /drivers/register", strictLimit(drivers.RegisterDriver))
	if visionSvc != nil {
		apiMux.Handle("POST /drivers/validate-photo", visionLimit(vision.ValidatePhoto(visionSvc)))
	}
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

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       180 * time.Second, // 3 minutes per user request
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	slog.Info("Server configuration applied", "read_timeout", "180s", "write_timeout", "30s")

	if err := server.ListenAndServe(); err != nil {
		slog.Error("Server failed", "error", err)
		os.Exit(1)
	}
}
