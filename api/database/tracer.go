package database

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
)

const (
	traceKeyStart = "pgx_query_start"
	traceKeySQL   = "pgx_query_sql"
)

type DBTracer struct{}

func (t *DBTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {

	ctx = context.WithValue(ctx, traceKeyStart, time.Now())
	ctx = context.WithValue(ctx, traceKeySQL, data.SQL)

	return ctx
}

func (t *DBTracer) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryEndData) {

	start, okStart := ctx.Value(traceKeyStart).(time.Time)
	sqlCmd, okSQL := ctx.Value(traceKeySQL).(string)

	if !okStart || !okSQL {
		return
	}

	duration := time.Since(start)

	attrs := []any{
		slog.String("sql", sqlCmd),
		slog.Duration("duration", duration),
	}

	if data.Err != nil {
		attrs = append(attrs, slog.Any("error", data.Err))
		slog.Error("SQL Query Failed", attrs...)
	} else {
		slog.Info("SQL Query Executed", attrs...)
	}
}
