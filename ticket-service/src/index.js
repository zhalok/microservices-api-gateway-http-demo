const express = require("express");
const { pool, init } = require("./db");

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

// List tickets, optionally filtered: /tickets?status=available
app.get("/tickets", async (req, res) => {
  const { status } = req.query;
  const { rows } = status
    ? await pool.query("SELECT * FROM tickets WHERE status = $1 ORDER BY id", [status])
    : await pool.query("SELECT * FROM tickets ORDER BY id");
  res.json(rows);
});

app.get("/tickets/:id", async (req, res) => {
  const { rows } = await pool.query("SELECT * FROM tickets WHERE id = $1", [req.params.id]);
  if (!rows[0]) return res.status(404).json({ error: "ticket not found" });
  res.json(rows[0]);
});

app.post("/tickets", async (req, res) => {
  const { event, price } = req.body;
  if (!event || price == null) return res.status(400).json({ error: "event and price are required" });
  const { rows } = await pool.query(
    "INSERT INTO tickets (event, price) VALUES ($1, $2) RETURNING *",
    [event, price]
  );
  res.status(201).json(rows[0]);
});

// Update a ticket. `expected_status` makes the update a compare-and-set, so two
// concurrent bookings can't both flip the same ticket to 'booked'.
app.patch("/tickets/:id", async (req, res) => {
  const { status, event, price, expected_status } = req.body;
  if (status && !["available", "booked"].includes(status)) {
    return res.status(400).json({ error: "status must be 'available' or 'booked'" });
  }
  const { rows } = await pool.query(
    `UPDATE tickets SET
       status = COALESCE($2, status),
       event  = COALESCE($3, event),
       price  = COALESCE($4, price)
     WHERE id = $1 AND ($5::text IS NULL OR status = $5)
     RETURNING *`,
    [req.params.id, status ?? null, event ?? null, price ?? null, expected_status ?? null]
  );
  if (rows[0]) return res.json(rows[0]);

  const exists = await pool.query("SELECT 1 FROM tickets WHERE id = $1", [req.params.id]);
  if (!exists.rowCount) return res.status(404).json({ error: "ticket not found" });
  res.status(409).json({ error: `ticket is not '${expected_status}'` });
});

app.use((err, _req, res, _next) => {
  console.error(err);
  res.status(500).json({ error: "internal error" });
});

async function start() {
  for (let i = 0; i < 20; i++) {
    try { await init(); break; } catch (e) {
      if (i === 19) throw e;
      await new Promise((r) => setTimeout(r, 1000));
    }
  }
  const port = process.env.PORT || 3001;
  app.listen(port, () => console.log(`ticket-service listening on ${port}`));
}
start();
