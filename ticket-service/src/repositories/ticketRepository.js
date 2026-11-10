const { pool } = require("../db")
const { withSpan } = require("../tracer")

function findAll(status) {
  return withSpan("TicketRepository.findAll", { "ticket.status": status ?? "" }, async () => {
    const { rows } = status
      ? await pool.query(
          "SELECT * FROM tickets WHERE status = $1 ORDER BY id",
          [status],
        )
      : await pool.query("SELECT * FROM tickets ORDER BY id")
    return rows
  })
}

function findById(id) {
  return withSpan("TicketRepository.findById", { "ticket.id": id }, async () => {
    const { rows } = await pool.query("SELECT * FROM tickets WHERE id = $1", [
      id,
    ])
    return rows[0]
  })
}

function insert({ event, price }) {
  return withSpan("TicketRepository.insert", {}, async () => {
    const { rows } = await pool.query(
      "INSERT INTO tickets (event, price) VALUES ($1, $2) RETURNING *",
      [event, price],
    )
    return rows[0]
  })
}

// Unconditional update (no status check, on purpose, to simulate a race).
// Returns the updated row, or undefined if the ticket doesn't exist.
function update(id, { status, event, price }) {
  return withSpan("TicketRepository.update", { "ticket.id": id }, async () => {
    const { rows } = await pool.query(
      `UPDATE tickets SET
         status = COALESCE($2, status),
         event  = COALESCE($3, event),
         price  = COALESCE($4, price)
       WHERE id = $1
       RETURNING *`,
      [id, status ?? null, event ?? null, price ?? null],
    )
    return rows[0]
  })
}

function exists(id) {
  return withSpan("TicketRepository.exists", { "ticket.id": id }, async () => {
    const { rowCount } = await pool.query("SELECT 1 FROM tickets WHERE id = $1", [
      id,
    ])
    return rowCount > 0
  })
}

module.exports = { findAll, findById, insert, update, exists }
