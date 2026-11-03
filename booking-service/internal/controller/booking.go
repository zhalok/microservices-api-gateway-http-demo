package controller

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"booking-service/internal/httperr"
	"booking-service/internal/model"
	"booking-service/internal/service"
	"booking-service/internal/tracing"
)

type BookingController struct{ svc *service.BookingService }

func New(svc *service.BookingService) *BookingController { return &BookingController{svc} }

// Routes registers the HTTP routes on mux.
func (c *BookingController) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, 200, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /bookings", c.handle("BookingController.list", c.list))
	mux.HandleFunc("GET /bookings/{id}", c.handle("BookingController.get", c.get))
	mux.HandleFunc("POST /bookings", c.handle("BookingController.create", c.create))
}

type result struct {
	status int
	body   any
}

type handler func(r *http.Request) (result, error)

// handle wraps h in a controller span and renders its result or error as JSON.
func (c *BookingController) handle(span string, h handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res, err := tracing.WithSpan(r.Context(), span, nil, func(ctx context.Context) (result, error) {
			return h(r.WithContext(ctx))
		})
		if err != nil {
			var he *httperr.Error
			if !errors.As(err, &he) {
				he = httperr.New(500, "internal error")
			}
			if he.Status >= 500 {
				log.Println(err)
			}
			writeJSON(w, he.Status, model.ErrorResponse{Error: he.Message})
			return
		}
		writeJSON(w, res.status, res.body)
	}
}

func (c *BookingController) list(r *http.Request) (result, error) {
	bookings, err := c.svc.List(r.Context())
	return result{200, bookings}, err
}

func (c *BookingController) get(r *http.Request) (result, error) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		return result{}, httperr.New(404, "booking not found")
	}
	b, err := c.svc.Get(r.Context(), id)
	return result{200, b}, err
}

func (c *BookingController) create(r *http.Request) (result, error) {
	var req model.CreateBookingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return result{}, httperr.New(400, "ticket_id and user_name are required")
	}
	b, err := c.svc.Create(r.Context(), req)
	return result{201, b}, err
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
