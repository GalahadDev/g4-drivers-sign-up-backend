package database

import (
	"context"
	"log/slog"
	"os"
	"time"

	"g4-services/api/config"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var Pool *pgxpool.Pool

func InitDB(cfg *config.AppConfig) {
	dsn := cfg.DatabaseDSN()

	pgConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		slog.Error("Error parsing DB config", "error", err)
		os.Exit(1)
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
		slog.Error("Unable to create connection pool", "error", err)
		os.Exit(1)
	}

	if err := Pool.Ping(ctx); err != nil {
		slog.Error("Database ping failed", "error", err)
		os.Exit(1)
	}

	slog.Info("Database connected successfully")
}
