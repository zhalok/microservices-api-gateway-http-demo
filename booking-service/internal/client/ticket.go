package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"

	"booking-service/internal/httperr"
	"booking-service/internal/model"
	"booking-service/internal/tracing"
)

// TicketClient talks to ticket-service over HTTP. The otelhttp transport
// propagates the trace context, so ticket-service spans join the same trace.
type TicketClient struct {
	baseURL string
	http    *http.Client
}

func NewTicketClient(baseURL string) *TicketClient {
	return &TicketClient{
		baseURL: baseURL,
		http:    &http.Client{Timeout: 10 * time.Second, Transport: otelhttp.NewTransport(http.DefaultTransport)},
	}
}

func (c *TicketClient) do(ctx context.Context, method string, id int, body any) (*http.Response, error) {
	var rd *bytes.Reader
	if body != nil {
		buf, _ := json.Marshal(body)
		rd = bytes.NewReader(buf)
	} else {
		rd = bytes.NewReader(nil)
	}
	req, err := http.NewRequestWithContext(ctx, method, fmt.Sprintf("%s/tickets/%d", c.baseURL, id), rd)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return c.http.Do(req)
}

// Get returns 404 if the ticket doesn't exist, 502 on any other upstream failure.
func (c *TicketClient) Get(ctx context.Context, id int) (*model.Ticket, error) {
	return tracing.WithSpan(ctx, "TicketClient.get", []attribute.KeyValue{attribute.Int("ticket.id", id)},
		func(ctx context.Context) (*model.Ticket, error) {
			resp, err := c.do(ctx, http.MethodGet, id, nil)
			if err != nil {
				return nil, upstreamErr(err)
			}
			defer resp.Body.Close()
			if resp.StatusCode == 404 {
				return nil, httperr.New(404, "ticket not found")
			}
			var t model.Ticket
			if resp.StatusCode >= 300 || json.NewDecoder(resp.Body).Decode(&t) != nil {
				return nil, httperr.New(502, "ticket service error")
			}
			return &t, nil
		})
}

// Patch updates a ticket. A 409 from ticket-service (compare-and-set mismatch)
// is surfaced as a 409 error.
func (c *TicketClient) Patch(ctx context.Context, id int, body map[string]any) (struct{}, error) {
	return tracing.WithSpan(ctx, "TicketClient.patch", []attribute.KeyValue{attribute.Int("ticket.id", id)},
		func(ctx context.Context) (struct{}, error) {
			resp, err := c.do(ctx, http.MethodPatch, id, body)
			if err != nil {
				return struct{}{}, upstreamErr(err)
			}
			defer resp.Body.Close()
			switch {
			case resp.StatusCode == 409:
				return struct{}{}, httperr.New(409, "ticket is already booked")
			case resp.StatusCode >= 300:
				return struct{}{}, httperr.New(502, "ticket service error")
			}
			return struct{}{}, nil
		})
}

func upstreamErr(err error) error {
	log.Println("ticket service request failed:", err)
	return httperr.New(502, "ticket service error")
}
