package config

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/joho/godotenv"
)

type AppConfig struct {
	Port           string
	DBUser         string
	DBPassword     string
	DBHost         string
	DBPort         string
	DBName         string
	SupabaseURL    string
	JWTSecret      string
	ServiceRoleKey string
	BrevoAPIKey    string
	BrevoFromEmail string
}

func Load() (*AppConfig, error) {
	if err := godotenv.Load(); err != nil {
		slog.Warn("No .env file found or error loading it. Relying on system environment variables.")
	} else {
		slog.Info(".env file loaded successfully")
	}

	cfg := &AppConfig{
		Port: getEnv("PORT", "8080"),

		DBUser:     os.Getenv("DB_USER"),
		DBPassword: os.Getenv("DB_PASSWORD"),
		DBHost:     os.Getenv("DB_HOST"),
		DBPort:     os.Getenv("DB_PORT"),
		DBName:     os.Getenv("DB_NAME"),

		SupabaseURL:    os.Getenv("SUPABASE_URL"),
		JWTSecret:      os.Getenv("JWT_SECRET"),
		ServiceRoleKey: os.Getenv("SUPABASE_SERVICE_ROLE_KEY"),

		BrevoAPIKey:    os.Getenv("BREVO_API_KEY"),
		BrevoFromEmail: os.Getenv("BREVO_FROM_EMAIL"),
	}

	if cfg.SupabaseURL == "" {
		return nil, fmt.Errorf("SUPABASE_URL is required in environment")
	}
	if cfg.DBUser == "" || cfg.DBPassword == "" || cfg.DBHost == "" {
		return nil, fmt.Errorf("database credentials are incomplete in environment")
	}

	return cfg, nil
}

func (c *AppConfig) DatabaseDSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=require",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName,
	)
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
