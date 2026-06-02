package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
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

	// Server context — cancelled on shutdown to stop rate limiter goroutines
	serverCtx, serverCancel := context.WithCancel(context.Background())
	defer serverCancel()

	// Rate limiters — one instance per tier, shared across all requests on that tier
	strictLimiter := middleware.NewIPRateLimiter(serverCtx, 0.00055, 2)
	mediumLimiter := middleware.NewIPRateLimiter(serverCtx, 0.083, 5)
	standardLimiter := middleware.NewIPRateLimiter(serverCtx, 1, 10)
	visionLimiter := middleware.NewIPRateLimiter(serverCtx, 0.0833, 3)
	docLimiter := middleware.NewIPRateLimiter(serverCtx, 0.5, 10)

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
		return middleware.RateLimitMiddleware(strictLimiter, protected(h))
	}

	mediumLimit := func(h http.HandlerFunc) http.Handler {
		return middleware.RateLimitMiddleware(mediumLimiter, protected(h))
	}

	standardLimit := func(h http.HandlerFunc) http.Handler {
		return middleware.RateLimitMiddleware(standardLimiter, protected(h))
	}

	visionLimit := func(h http.HandlerFunc) http.Handler {
		return middleware.RateLimitMiddleware(visionLimiter, protected(h))
	}

	docLimit := func(h http.HandlerFunc) http.Handler {
		return middleware.RateLimitMiddleware(docLimiter, protected(h))
	}

	// --- API ROUTES  ---

	// Drivers & User
	apiMux.Handle("POST /drivers/register", strictLimit(drivers.RegisterDriver(cfg)))
	if visionSvc != nil {
		apiMux.Handle("POST /drivers/validate-photo", visionLimit(vision.ValidatePhoto(visionSvc)))
		apiMux.Handle("POST /drivers/validate-document", docLimit(vision.ValidateDocument(visionSvc)))
	}
	apiMux.Handle("GET /user/dashboard", standardLimit(dashboard.GetMyDashboard))
	apiMux.Handle("PUT /user/profile", mediumLimit(users.UpdateMyProfile(cfg)))
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
		ReadTimeout:       180 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	slog.Info("Server configuration applied", "read_timeout", "180s", "write_timeout", "30s")

	// Start server in background
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server failed", "error", err)
			os.Exit(1)
		}
	}()

	// Block until OS signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down server gracefully...")
	serverCancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("Server shutdown error", "error", err)
	}
	slog.Info("Server stopped")
}
