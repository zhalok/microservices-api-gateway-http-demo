package db

import (
	"context"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5/pgxpool"
)

func Connect(ctx context.Context, url string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, err
	}
	cfg.ConnConfig.Tracer = otelpgx.NewTracer()
	return pgxpool.NewWithConfig(ctx, cfg)
}

// Init creates the schema.
func Init(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS bookings (
		  id SERIAL PRIMARY KEY,
		  ticket_id INTEGER NOT NULL UNIQUE,
		  user_name TEXT NOT NULL,
		  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`)
	return err
}
