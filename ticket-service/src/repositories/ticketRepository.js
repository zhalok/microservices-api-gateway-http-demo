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

// Compare-and-set: only updates when the current status matches expectedStatus
// (if given). Returns the updated row, or undefined if nothing matched.
function update(id, { status, event, price, expectedStatus }) {
  return withSpan("TicketRepository.update", { "ticket.id": id }, async () => {
    const { rows } = await pool.query(
      `UPDATE tickets SET
         status = COALESCE($2, status),
         event  = COALESCE($3, event),
         price  = COALESCE($4, price)
       WHERE id = $1 AND ($5::text IS NULL OR status = $5)
       RETURNING *`,
      [
        id,
        status ?? null,
        event ?? null,
        price ?? null,
        expectedStatus ?? null,
      ],
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
