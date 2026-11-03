package controller

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"ticket-service/internal/httperr"
	"ticket-service/internal/model"
	"ticket-service/internal/service"
	"ticket-service/internal/tracing"
)

type TicketController struct{ svc *service.TicketService }

func New(svc *service.TicketService) *TicketController { return &TicketController{svc} }

// Routes registers the HTTP routes on mux.
func (c *TicketController) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, 200, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /tickets", c.handle("TicketController.list", c.list))
	mux.HandleFunc("GET /tickets/{id}", c.handle("TicketController.get", c.get))
	mux.HandleFunc("POST /tickets", c.handle("TicketController.create", c.create))
	mux.HandleFunc("PATCH /tickets/{id}", c.handle("TicketController.update", c.update))
}

type result struct {
	status int
	body   any
}

type handler func(r *http.Request) (result, error)

// handle wraps h in a controller span and renders its result or error as JSON.
func (c *TicketController) handle(span string, h handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res, err := tracing.WithSpan(r.Context(), span, nil, func(ctx context.Context) (result, error) {
			return h(r.WithContext(ctx))
		})
		if err != nil {
			if st := httperr.Status(err); st < 500 {
				writeJSON(w, st, model.ErrorResponse{Error: err.Error()})
				return
			}
			log.Println(err)
			writeJSON(w, 500, model.ErrorResponse{Error: "internal error"})
			return
		}
		writeJSON(w, res.status, res.body)
	}
}

func (c *TicketController) list(r *http.Request) (result, error) {
	tickets, err := c.svc.List(r.Context(), r.URL.Query().Get("status"))
	return result{200, tickets}, err
}

func (c *TicketController) get(r *http.Request) (result, error) {
	id, err := pathID(r)
	if err != nil {
		return result{}, err
	}
	t, err := c.svc.Get(r.Context(), id)
	return result{200, t}, err
}

func (c *TicketController) create(r *http.Request) (result, error) {
	var body model.CreateTicketRequest
	if err := decode(r, &body); err != nil {
		return result{}, err
	}
	price := ""
	if body.Price != nil {
		price = body.Price.String()
	}
	t, err := c.svc.Create(r.Context(), body.Event, price)
	return result{201, t}, err
}

func (c *TicketController) update(r *http.Request) (result, error) {
	id, err := pathID(r)
	if err != nil {
		return result{}, err
	}
	var body model.UpdateTicketRequest
	if err := decode(r, &body); err != nil {
		return result{}, err
	}
	p := model.UpdateParams{Status: body.Status, Event: body.Event, ExpectedStatus: body.ExpectedStatus}
	if body.Price != nil {
		s := body.Price.String()
		p.Price = &s
	}
	t, err := c.svc.Update(r.Context(), id, p)
	return result{200, t}, err
}

func pathID(r *http.Request) (int, error) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id < 1 {
		return 0, httperr.New(400, "invalid ticket id")
	}
	return id, nil
}

func decode(r *http.Request, v any) error {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		return httperr.New(400, "invalid JSON body")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
