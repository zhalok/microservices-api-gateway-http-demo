package model

import "time"

type Booking struct {
	ID        int       `json:"id"`
	TicketID  int       `json:"ticket_id"`
	UserName  string    `json:"user_name"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateBookingRequest is the POST /bookings body.
type CreateBookingRequest struct {
	TicketID int    `json:"ticket_id"`
	UserName string `json:"user_name"`
}

// Ticket is the subset of a ticket-service ticket that booking cares about.
type Ticket struct {
	Status string `json:"status"`
}

// ErrorResponse is the JSON shape of every error reply.
type ErrorResponse struct {
	Error string `json:"error"`
}
