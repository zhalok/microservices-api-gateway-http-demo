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

// Init creates the schema and seeds a few tickets on first run.
func Init(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS tickets (
		  id SERIAL PRIMARY KEY,
		  event TEXT NOT NULL,
		  price NUMERIC(10, 2) NOT NULL,
		  status TEXT NOT NULL DEFAULT 'available' CHECK (status IN ('available', 'booked'))
		)`); err != nil {
		return err
	}
	var n int
	if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM tickets").Scan(&n); err != nil {
		return err
	}
	if n == 0 {
		_, err := pool.Exec(ctx, `
			INSERT INTO tickets (event, price) VALUES
			  ('Coldplay - Dhaka', 120.00),
			  ('Coldplay - Dhaka', 120.00),
			  ('Jazz Night', 45.50),
			  ('Jazz Night', 45.50),
			  ('Tech Conf 2026', 200.00)`)
		return err
	}
	return nil
}
