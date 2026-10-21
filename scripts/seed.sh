#!/usr/bin/env bash
# Resets and seeds the demo data. Writes straight to Postgres so it keeps working
# once the gateway requires auth. Usage: ./scripts/seed.sh  (stack must be running)
set -euo pipefail
cd "$(dirname "$0")/.."

# Clear bookings too, otherwise they'd point at tickets that no longer exist.
docker compose exec -T booking-db psql -q -U postgres -d bookingdb \
  -c "TRUNCATE bookings RESTART IDENTITY"

docker compose exec -T ticket-db psql -q -U postgres -d ticketdb <<'SQL'
TRUNCATE tickets RESTART IDENTITY;
INSERT INTO tickets (event, price)
SELECT event, price
FROM (VALUES
  ('Coldplay - Dhaka', 120.00, 10),
  ('Jazz Night',        45.50, 10),
  ('Tech Conf 2026',   200.00, 10)
) AS e(event, price, qty), generate_series(1, e.qty);
SQL

echo "Seeded: $(docker compose exec -T ticket-db psql -tA -U postgres -d ticketdb -c 'SELECT count(*) FROM tickets') tickets"
