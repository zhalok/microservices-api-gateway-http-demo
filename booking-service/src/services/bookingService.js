const repo = require("../repositories/bookingRepository")
const tickets = require("../ticketClient")
const { withSpan } = require("../tracer")
const { HttpError } = require("../errors")
const { withLock } = require("../lock")

function list() {
  return withSpan("BookingService.list", {}, () => repo.findAll())
}

function get(id) {
  return withSpan("BookingService.get", { "booking.id": id }, async () => {
    const booking = await repo.findById(id)
    if (!booking) throw new HttpError(404, "booking not found")
    return booking
  })
}

function create({ ticket_id, user_name } = {}) {
  return withSpan("BookingService.create", { "ticket.id": ticket_id ?? "" }, async () => {
    if (!ticket_id || !user_name)
      throw new HttpError(400, "ticket_id and user_name are required")

    // 1. check availability
    const ticket = await tickets.getTicket(ticket_id)
    if (ticket.status !== "available")
      throw new HttpError(409, "ticket is already booked")

    // simulating delay to trigger a race condition inconsistency

    if (user_name === (process.env.DELAYED_USER_NAME || "bob"))
      await new Promise((resolve) => setTimeout(resolve, 5000))

    // 2. mark as booked (compare-and-set is off on purpose; createWithLock provides the serialization)
    const update = await tickets.patchTicket(ticket_id, {
      status: "booked",
    })
    if (update.status === 409)
      throw new HttpError(409, "ticket is already booked")
    if (!update.ok) throw new HttpError(502, "ticket service error")

    // 3. create booking record; if that fails, release the ticket
    try {
      return await repo.insert({ ticketId: ticket_id, userName: user_name })
    } catch (err) {
      console.error("booking insert failed, releasing ticket", err)
      await tickets.patchTicket(ticket_id, { status: "available" }).catch(() => {})
      throw err
    }
  })
}

// Serializes bookings of the same ticket across service instances by wrapping
// create in a distributed lock.
function createWithLock(body = {}) {
  // Without a ticket_id there is nothing to lock; create() rejects it with a 400.
  if (!body.ticket_id) return create(body)
  return withLock(`lock:ticket:${body.ticket_id}`, () => create(body))
}

module.exports = { list, get, create, createWithLock }
