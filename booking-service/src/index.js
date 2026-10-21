const express = require("express");
const { pool, init } = require("./db");

const TICKET_SERVICE_URL = process.env.TICKET_SERVICE_URL;

const app = express();
app.use(express.json());

app.use((req, res, next) => {
  const start = Date.now();
  res.on("finish", () => {
    console.log(`${req.method} ${req.originalUrl} ${res.statusCode} ${Date.now() - start}ms`);
  });
  next();
});

app.get("/health", (_req, res) => res.json({ status: "ok" }));

app.get("/bookings", async (_req, res) => {
  const { rows } = await pool.query("SELECT * FROM bookings ORDER BY id");
  res.json(rows);
});

app.get("/bookings/:id", async (req, res) => {
  const { rows } = await pool.query("SELECT * FROM bookings WHERE id = $1", [req.params.id]);
  if (!rows[0]) return res.status(404).json({ error: "booking not found" });
  res.json(rows[0]);
});

function patchTicket(id, body) {
  return fetch(`${TICKET_SERVICE_URL}/tickets/${id}`, {
    method: "PATCH",
    headers: { "content-type": "application/json" },
    body: JSON.stringify(body),
  });
}

app.post("/bookings", async (req, res) => {
  const { ticket_id, user_name } = req.body;
  if (!ticket_id || !user_name) {
    return res.status(400).json({ error: "ticket_id and user_name are required" });
  }

  // 1. check availability
  const check = await fetch(`${TICKET_SERVICE_URL}/tickets/${ticket_id}`);
  if (check.status === 404) return res.status(404).json({ error: "ticket not found" });
  if (!check.ok) return res.status(502).json({ error: "ticket service error" });
  const ticket = await check.json();
  if (ticket.status !== "available") {
    return res.status(409).json({ error: "ticket is already booked" });
  }

  // 2. mark as booked (compare-and-set guards against a concurrent booking)
  const update = await patchTicket(ticket_id, { status: "booked", expected_status: "available" });
  if (update.status === 409) return res.status(409).json({ error: "ticket is already booked" });
  if (!update.ok) return res.status(502).json({ error: "ticket service error" });

  // 3. create booking record; if that fails, release the ticket
  try {
    const { rows } = await pool.query(
      "INSERT INTO bookings (ticket_id, user_name) VALUES ($1, $2) RETURNING *",
      [ticket_id, user_name]
    );
    res.status(201).json(rows[0]);
  } catch (err) {
    console.error("booking insert failed, releasing ticket", err);
    await patchTicket(ticket_id, { status: "available" }).catch(() => {});
    res.status(500).json({ error: "could not create booking" });
  }
});

app.use((err, _req, res, _next) => {
  console.error(err);
  res.status(502).json({ error: "internal error" });
});

async function start() {
  for (let i = 0; i < 20; i++) {
    try { await init(); break; } catch (e) {
      if (i === 19) throw e;
      await new Promise((r) => setTimeout(r, 1000));
    }
  }
  const port = process.env.PORT || 3002;
  app.listen(port, () => console.log(`booking-service listening on ${port}`));
}
start();
