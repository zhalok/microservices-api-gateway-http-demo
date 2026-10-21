#!/usr/bin/env bash
# Deletes all tickets and bookings (tables are kept, ids restart at 1).
# Usage: ./scripts/cleanup.sh  (stack must be running). Run ./scripts/seed.sh afterwards to refill.
set -euo pipefail
cd "$(dirname "$0")/.."

docker compose exec -T booking-db psql -q -U postgres -d bookingdb \
  -c "TRUNCATE bookings RESTART IDENTITY"
docker compose exec -T ticket-db psql -q -U postgres -d ticketdb \
  -c "TRUNCATE tickets RESTART IDENTITY"

echo "Cleaned: all tickets and bookings removed"
