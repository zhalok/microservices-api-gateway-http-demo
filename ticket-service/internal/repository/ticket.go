package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/attribute"

	"ticket-service/internal/tracing"
)

// Ticket's price is a string to preserve NUMERIC precision (matches the JSON the Node service produced).
type Ticket struct {
	ID     int    `json:"id"`
	Event  string `json:"event"`
	Price  string `json:"price"`
	Status string `json:"status"`
}

type UpdateParams struct {
	Status, Event, Price *string
	ExpectedStatus       *string
}

type TicketRepository struct{ pool *pgxpool.Pool }

func NewTicketRepository(pool *pgxpool.Pool) *TicketRepository { return &TicketRepository{pool} }

const cols = "id, event, price::text, status"

func scan(row pgx.Row) (*Ticket, error) {
	var t Ticket
	if err := row.Scan(&t.ID, &t.Event, &t.Price, &t.Status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

func (r *TicketRepository) FindAll(ctx context.Context, status string) ([]Ticket, error) {
	return tracing.WithSpan(ctx, "TicketRepository.findAll", []attribute.KeyValue{attribute.String("ticket.status", status)},
		func(ctx context.Context) ([]Ticket, error) {
			var rows pgx.Rows
			var err error
			if status != "" {
				rows, err = r.pool.Query(ctx, "SELECT "+cols+" FROM tickets WHERE status = $1 ORDER BY id", status)
			} else {
				rows, err = r.pool.Query(ctx, "SELECT "+cols+" FROM tickets ORDER BY id")
			}
			if err != nil {
				return nil, err
			}
			defer rows.Close()
			tickets := []Ticket{}
			for rows.Next() {
				var t Ticket
				if err := rows.Scan(&t.ID, &t.Event, &t.Price, &t.Status); err != nil {
					return nil, err
				}
				tickets = append(tickets, t)
			}
			return tickets, rows.Err()
		})
}

func (r *TicketRepository) FindByID(ctx context.Context, id int) (*Ticket, error) {
	return tracing.WithSpan(ctx, "TicketRepository.findById", []attribute.KeyValue{attribute.Int("ticket.id", id)},
		func(ctx context.Context) (*Ticket, error) {
			return scan(r.pool.QueryRow(ctx, "SELECT "+cols+" FROM tickets WHERE id = $1", id))
		})
}

func (r *TicketRepository) Insert(ctx context.Context, event, price string) (*Ticket, error) {
	return tracing.WithSpan(ctx, "TicketRepository.insert", nil,
		func(ctx context.Context) (*Ticket, error) {
			return scan(r.pool.QueryRow(ctx,
				"INSERT INTO tickets (event, price) VALUES ($1, $2::numeric) RETURNING "+cols, event, price))
		})
}

// Update is a compare-and-set: it only applies when the current status matches
// ExpectedStatus (if given). Returns nil if nothing matched.
func (r *TicketRepository) Update(ctx context.Context, id int, p UpdateParams) (*Ticket, error) {
	return tracing.WithSpan(ctx, "TicketRepository.update", []attribute.KeyValue{attribute.Int("ticket.id", id)},
		func(ctx context.Context) (*Ticket, error) {
			return scan(r.pool.QueryRow(ctx, `
				UPDATE tickets SET
				  status = COALESCE($2, status),
				  event  = COALESCE($3, event),
				  price  = COALESCE($4::numeric, price)
				WHERE id = $1 AND ($5::text IS NULL OR status = $5)
				RETURNING `+cols, id, p.Status, p.Event, p.Price, p.ExpectedStatus))
		})
}

func (r *TicketRepository) Exists(ctx context.Context, id int) (bool, error) {
	return tracing.WithSpan(ctx, "TicketRepository.exists", []attribute.KeyValue{attribute.Int("ticket.id", id)},
		func(ctx context.Context) (bool, error) {
			var ok bool
			err := r.pool.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM tickets WHERE id = $1)", id).Scan(&ok)
			return ok, err
		})
}
