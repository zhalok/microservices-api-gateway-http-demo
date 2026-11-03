package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/attribute"

	"booking-service/internal/model"
	"booking-service/internal/tracing"
)

type BookingRepository struct{ pool *pgxpool.Pool }

func NewBookingRepository(pool *pgxpool.Pool) *BookingRepository { return &BookingRepository{pool} }

const cols = "id, ticket_id, user_name, created_at"

func scan(row pgx.Row) (*model.Booking, error) {
	var b model.Booking
	if err := row.Scan(&b.ID, &b.TicketID, &b.UserName, &b.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &b, nil
}

func (r *BookingRepository) FindAll(ctx context.Context) ([]model.Booking, error) {
	return tracing.WithSpan(ctx, "BookingRepository.findAll", nil,
		func(ctx context.Context) ([]model.Booking, error) {
			rows, err := r.pool.Query(ctx, "SELECT "+cols+" FROM bookings ORDER BY id")
			if err != nil {
				return nil, err
			}
			defer rows.Close()
			bookings := []model.Booking{}
			for rows.Next() {
				var b model.Booking
				if err := rows.Scan(&b.ID, &b.TicketID, &b.UserName, &b.CreatedAt); err != nil {
					return nil, err
				}
				bookings = append(bookings, b)
			}
			return bookings, rows.Err()
		})
}

func (r *BookingRepository) FindByID(ctx context.Context, id int) (*model.Booking, error) {
	return tracing.WithSpan(ctx, "BookingRepository.findById", []attribute.KeyValue{attribute.Int("booking.id", id)},
		func(ctx context.Context) (*model.Booking, error) {
			return scan(r.pool.QueryRow(ctx, "SELECT "+cols+" FROM bookings WHERE id = $1", id))
		})
}

func (r *BookingRepository) Insert(ctx context.Context, ticketID int, userName string) (*model.Booking, error) {
	return tracing.WithSpan(ctx, "BookingRepository.insert", []attribute.KeyValue{attribute.Int("ticket.id", ticketID)},
		func(ctx context.Context) (*model.Booking, error) {
			return scan(r.pool.QueryRow(ctx,
				"INSERT INTO bookings (ticket_id, user_name) VALUES ($1, $2) RETURNING "+cols, ticketID, userName))
		})
}
