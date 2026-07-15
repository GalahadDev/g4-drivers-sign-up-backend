package database

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"g4-services/api/config"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var Pool *pgxpool.Pool

// InitDB initializes the global connection pool. Returns an error instead of
// calling os.Exit so the caller controls shutdown (audit M5, matches InitAuth).
func InitDB(cfg *config.AppConfig) error {
	dsn := cfg.DatabaseDSN()

	pgConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return fmt.Errorf("parsing DB config: %w", err)
	}

	pgConfig.ConnConfig.Tracer = &DBTracer{}

	// Configuración Session Pooler
	pgConfig.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
	pgConfig.MaxConns = 10
	pgConfig.MinConns = 2
	pgConfig.MaxConnLifetime = 1 * time.Hour

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	Pool, err = pgxpool.NewWithConfig(ctx, pgConfig)
	if err != nil {
		return fmt.Errorf("creating connection pool: %w", err)
	}

	if err := Pool.Ping(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	slog.Info("Database connected successfully")
	return nil
}
