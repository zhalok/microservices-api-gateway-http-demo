# Ticket Booking – Microservices + API Gateway demo

```
client ──> gateway :8080 ─┬─> ticket-service  :3001 ──> ticket-db  (Postgres)
                          └─> booking-service :3002 ──> booking-db (Postgres)
                                   └── HTTP ──> ticket-service
```

Each service owns its own Postgres database. Only the gateway is exposed to the host.

## Run
    docker compose up --build

## API (all via the gateway, http://localhost:8080)
| Method | Path | Notes |
|---|---|---|
| GET | /tickets[?status=available] | list tickets |
| GET | /tickets/:id | get one |
| POST | /tickets | `{event, price}` |
| PATCH | /tickets/:id | `{status?, event?, price?, expected_status?}` – `expected_status` makes it compare-and-set (409 on mismatch) |
| POST | /bookings | `{ticket_id, user_name}` |
| GET | /bookings, /bookings/:id | |

## Booking flow
1. `GET` ticket from ticket-service → 409 if not `available`
2. `PATCH` ticket to `booked` with `expected_status: available` (atomic, prevents double booking)
3. Insert booking row; if that fails, the ticket is released again

## Try it
    curl -XPOST localhost:8080/bookings -H 'content-type: application/json' \
      -d '{"ticket_id":1,"user_name":"alice"}'
