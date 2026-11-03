package service

import (
	"context"
	"log"
	"time"

	"go.opentelemetry.io/otel/attribute"

	"booking-service/internal/client"
	"booking-service/internal/httperr"
	"booking-service/internal/model"
	"booking-service/internal/repository"
	"booking-service/internal/tracing"
)

type BookingService struct {
	repo    *repository.BookingRepository
	tickets *client.TicketClient
}

func NewBookingService(repo *repository.BookingRepository, tickets *client.TicketClient) *BookingService {
	return &BookingService{repo, tickets}
}

func (s *BookingService) List(ctx context.Context) ([]model.Booking, error) {
	return tracing.WithSpan(ctx, "BookingService.list", nil, s.repo.FindAll)
}

func (s *BookingService) Get(ctx context.Context, id int) (*model.Booking, error) {
	return tracing.WithSpan(ctx, "BookingService.get", []attribute.KeyValue{attribute.Int("booking.id", id)},
		func(ctx context.Context) (*model.Booking, error) {
			b, err := s.repo.FindByID(ctx, id)
			if err != nil {
				return nil, err
			}
			if b == nil {
				return nil, httperr.New(404, "booking not found")
			}
			return b, nil
		})
}

func (s *BookingService) Create(ctx context.Context, req model.CreateBookingRequest) (*model.Booking, error) {
	return tracing.WithSpan(ctx, "BookingService.create", []attribute.KeyValue{attribute.Int("ticket.id", req.TicketID)},
		func(ctx context.Context) (*model.Booking, error) {
			if req.TicketID == 0 || req.UserName == "" {
				return nil, httperr.New(400, "ticket_id and user_name are required")
			}

			// 1. check availability
			ticket, err := s.tickets.Get(ctx, req.TicketID)
			if err != nil {
				return nil, err
			}
			if ticket.Status != "available" {
				return nil, httperr.New(409, "ticket is already booked")
			}

			// 2. mark as booked (compare-and-set guards against a concurrent booking)
			if _, err := s.tickets.Patch(ctx, req.TicketID, map[string]any{"status": "booked", "expected_status": "available"}); err != nil {
				return nil, err
			}

			// 3. create booking record; if that fails, release the ticket
			b, err := s.repo.Insert(ctx, req.TicketID, req.UserName)
			if err != nil {
				log.Println("booking insert failed, releasing ticket:", err)
				// Detach from the request's cancellation but keep its trace.
				relCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
				defer cancel()
				if _, perr := s.tickets.Patch(relCtx, req.TicketID, map[string]any{"status": "available"}); perr != nil {
					log.Println("could not release ticket:", perr)
				}
				return nil, httperr.New(500, "could not create booking")
			}
			return b, nil
		})
}
