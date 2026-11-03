const { pool } = require("../db")
const { withSpan } = require("../tracer")

function findAll() {
  return withSpan("BookingRepository.findAll", {}, async () => {
    const { rows } = await pool.query("SELECT * FROM bookings ORDER BY id")
    return rows
  })
}

function findById(id) {
  return withSpan("BookingRepository.findById", { "booking.id": id }, async () => {
    const { rows } = await pool.query("SELECT * FROM bookings WHERE id = $1", [
      id,
    ])
    return rows[0]
  })
}

function insert({ ticketId, userName }) {
  return withSpan("BookingRepository.insert", { "ticket.id": ticketId }, async () => {
    const { rows } = await pool.query(
      "INSERT INTO bookings (ticket_id, user_name) VALUES ($1, $2) RETURNING *",
      [ticketId, userName],
    )
    return rows[0]
  })
}

module.exports = { findAll, findById, insert }
