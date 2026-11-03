package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Booking struct {
	ID        int       `json:"id"`
	TicketID  int       `json:"ticket_id"`
	UserName  string    `json:"user_name"`
	CreatedAt time.Time `json:"created_at"`
}

const bookingCols = "id, ticket_id, user_name, created_at"

func scanBooking(row pgx.Row) (*Booking, error) {
	var b Booking
	if err := row.Scan(&b.ID, &b.TicketID, &b.UserName, &b.CreatedAt); err != nil {
		return nil, err
	}
	return &b, nil
}

type server struct {
	pool             *pgxpool.Pool
	ticketServiceURL string
	client           *http.Client
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func (s *server) list(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pool.Query(r.Context(), "SELECT "+bookingCols+" FROM bookings ORDER BY id")
	if err != nil {
		log.Println(err)
		writeErr(w, 502, "internal error")
		return
	}
	defer rows.Close()
	bookings := []Booking{}
	for rows.Next() {
		var b Booking
		if err := rows.Scan(&b.ID, &b.TicketID, &b.UserName, &b.CreatedAt); err != nil {
			log.Println(err)
			writeErr(w, 502, "internal error")
			return
		}
		bookings = append(bookings, b)
	}
	writeJSON(w, 200, bookings)
}

func (s *server) get(w http.ResponseWriter, r *http.Request) {
	var id int
	if _, err := fmt.Sscanf(r.PathValue("id"), "%d", &id); err != nil {
		writeErr(w, 404, "booking not found")
		return
	}
	b, err := scanBooking(s.pool.QueryRow(r.Context(), "SELECT "+bookingCols+" FROM bookings WHERE id = $1", id))
	if errors.Is(err, pgx.ErrNoRows) {
		writeErr(w, 404, "booking not found")
		return
	}
	if err != nil {
		log.Println(err)
		writeErr(w, 502, "internal error")
		return
	}
	writeJSON(w, 200, b)
}

func (s *server) patchTicket(ctx context.Context, id int, body map[string]any) (*http.Response, error) {
	buf, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, fmt.Sprintf("%s/tickets/%d", s.ticketServiceURL, id), bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return s.client.Do(req)
}

func (s *server) create(w http.ResponseWriter, r *http.Request) {
	var in struct {
		TicketID int    `json:"ticket_id"`
		UserName string `json:"user_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.TicketID == 0 || in.UserName == "" {
		writeErr(w, 400, "ticket_id and user_name are required")
		return
	}
	ctx := r.Context()

	// 1. check availability
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/tickets/%d", s.ticketServiceURL, in.TicketID), nil)
	check, err := s.client.Do(req)
	if err != nil {
		log.Println(err)
		writeErr(w, 502, "ticket service error")
		return
	}
	defer check.Body.Close()
	if check.StatusCode == 404 {
		writeErr(w, 404, "ticket not found")
		return
	}
	if check.StatusCode >= 300 {
		writeErr(w, 502, "ticket service error")
		return
	}
	var ticket struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(check.Body).Decode(&ticket); err != nil {
		writeErr(w, 502, "ticket service error")
		return
	}
	if ticket.Status != "available" {
		writeErr(w, 409, "ticket is already booked")
		return
	}

	// 2. mark as booked (compare-and-set guards against a concurrent booking)
	update, err := s.patchTicket(ctx, in.TicketID, map[string]any{"status": "booked", "expected_status": "available"})
	if err != nil {
		log.Println(err)
		writeErr(w, 502, "ticket service error")
		return
	}
	update.Body.Close()
	if update.StatusCode == 409 {
		writeErr(w, 409, "ticket is already booked")
		return
	}
	if update.StatusCode >= 300 {
		writeErr(w, 502, "ticket service error")
		return
	}

	// 3. create booking record; if that fails, release the ticket
	b, err := scanBooking(s.pool.QueryRow(ctx,
		"INSERT INTO bookings (ticket_id, user_name) VALUES ($1, $2) RETURNING "+bookingCols, in.TicketID, in.UserName))
	if err != nil {
		log.Println("booking insert failed, releasing ticket:", err)
		// Use a fresh context: the request may be the thing that was cancelled.
		relCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if resp, err := s.patchTicket(relCtx, in.TicketID, map[string]any{"status": "available"}); err == nil {
			resp.Body.Close()
		}
		writeErr(w, 500, "could not create booking")
		return
	}
	writeJSON(w, 201, b)
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: 200}
		next.ServeHTTP(rec, r)
		log.Printf("%s %s %d %dms", r.Method, r.URL.RequestURI(), rec.status, time.Since(start).Milliseconds())
	})
}

func main() {
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()
	if _, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS bookings (
		  id SERIAL PRIMARY KEY,
		  ticket_id INTEGER NOT NULL UNIQUE,
		  user_name TEXT NOT NULL,
		  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`); err != nil {
		log.Fatalf("db init: %v", err)
	}

	s := &server{
		pool:             pool,
		ticketServiceURL: os.Getenv("TICKET_SERVICE_URL"),
		client:           &http.Client{Timeout: 10 * time.Second},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, 200, map[string]string{"status": "ok"}) })
	mux.HandleFunc("GET /bookings", s.list)
	mux.HandleFunc("GET /bookings/{id}", s.get)
	mux.HandleFunc("POST /bookings", s.create)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3002"
	}
	log.Printf("booking-service listening on %s", port)
	log.Fatal(http.ListenAndServe(":"+port, logRequests(mux)))
}
