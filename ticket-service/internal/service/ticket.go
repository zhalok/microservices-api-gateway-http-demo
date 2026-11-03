package service

import (
	"context"

	"go.opentelemetry.io/otel/attribute"

	"ticket-service/internal/httperr"
	"ticket-service/internal/repository"
	"ticket-service/internal/tracing"
)

type TicketService struct{ repo *repository.TicketRepository }

func NewTicketService(repo *repository.TicketRepository) *TicketService {
	return &TicketService{repo}
}

func isValidStatus(s string) bool { return s == "available" || s == "booked" }

func (s *TicketService) List(ctx context.Context, status string) ([]repository.Ticket, error) {
	return tracing.WithSpan(ctx, "TicketService.list", []attribute.KeyValue{attribute.String("ticket.status", status)},
		func(ctx context.Context) ([]repository.Ticket, error) {
			return s.repo.FindAll(ctx, status)
		})
}

func (s *TicketService) Get(ctx context.Context, id int) (*repository.Ticket, error) {
	return tracing.WithSpan(ctx, "TicketService.get", []attribute.KeyValue{attribute.Int("ticket.id", id)},
		func(ctx context.Context) (*repository.Ticket, error) {
			t, err := s.repo.FindByID(ctx, id)
			if err != nil {
				return nil, err
			}
			if t == nil {
				return nil, httperr.New(404, "ticket not found")
			}
			return t, nil
		})
}

func (s *TicketService) Create(ctx context.Context, event, price string) (*repository.Ticket, error) {
	return tracing.WithSpan(ctx, "TicketService.create", nil,
		func(ctx context.Context) (*repository.Ticket, error) {
			if event == "" || price == "" {
				return nil, httperr.New(400, "event and price are required")
			}
			return s.repo.Insert(ctx, event, price)
		})
}

// Update with ExpectedStatus is a compare-and-set, so two concurrent bookings
// can't both flip the same ticket to 'booked'.
func (s *TicketService) Update(ctx context.Context, id int, p repository.UpdateParams) (*repository.Ticket, error) {
	return tracing.WithSpan(ctx, "TicketService.update", []attribute.KeyValue{attribute.Int("ticket.id", id)},
		func(ctx context.Context) (*repository.Ticket, error) {
			if p.Status != nil && !isValidStatus(*p.Status) {
				return nil, httperr.New(400, "status must be 'available' or 'booked'")
			}
			t, err := s.repo.Update(ctx, id, p)
			if err != nil {
				return nil, err
			}
			if t != nil {
				return t, nil
			}
			ok, err := s.repo.Exists(ctx, id)
			if err != nil {
				return nil, err
			}
			if !ok {
				return nil, httperr.New(404, "ticket not found")
			}
			expected := ""
			if p.ExpectedStatus != nil {
				expected = *p.ExpectedStatus
			}
			return nil, httperr.New(409, "ticket is not '%s'", expected)
		})
}
