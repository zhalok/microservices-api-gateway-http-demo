package model

import "encoding/json"

// Ticket's price is a string to preserve NUMERIC precision.
type Ticket struct {
	ID     int    `json:"id"`
	Event  string `json:"event"`
	Price  string `json:"price"`
	Status string `json:"status"`
}

// UpdateParams is the domain-level update; ExpectedStatus makes it a compare-and-set.
type UpdateParams struct {
	Status, Event, Price *string
	ExpectedStatus       *string
}

// CreateTicketRequest is the POST /tickets body.
type CreateTicketRequest struct {
	Event string       `json:"event"`
	Price *json.Number `json:"price"`
}

// UpdateTicketRequest is the PATCH /tickets/{id} body.
type UpdateTicketRequest struct {
	Status         *string      `json:"status"`
	Event          *string      `json:"event"`
	Price          *json.Number `json:"price"`
	ExpectedStatus *string      `json:"expected_status"`
}

// ErrorResponse is the JSON shape of every error reply.
type ErrorResponse struct {
	Error string `json:"error"`
}
