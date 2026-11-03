const repo = require("../repositories/ticketRepository")
const { withSpan } = require("../tracer")
const { HttpError } = require("../errors")

const STATUSES = ["available", "booked"]

function list(status) {
  return withSpan("TicketService.list", { "ticket.status": status ?? "" }, () =>
    repo.findAll(status),
  )
}

function get(id) {
  return withSpan("TicketService.get", { "ticket.id": id }, async () => {
    const ticket = await repo.findById(id)
    if (!ticket) throw new HttpError(404, "ticket not found")
    return ticket
  })
}

function create({ event, price }) {
  return withSpan("TicketService.create", {}, () => {
    if (!event || price == null)
      throw new HttpError(400, "event and price are required")
    return repo.insert({ event, price })
  })
}

// `expected_status` makes the update a compare-and-set, so two concurrent
// bookings can't both flip the same ticket to 'booked'.
function update(id, { status, event, price, expected_status }) {
  return withSpan("TicketService.update", { "ticket.id": id }, async () => {
    if (status && !STATUSES.includes(status))
      throw new HttpError(400, "status must be 'available' or 'booked'")

    const ticket = await repo.update(id, {
      status,
      event,
      price,
      expectedStatus: expected_status,
    })
    if (ticket) return ticket

    if (!(await repo.exists(id))) throw new HttpError(404, "ticket not found")
    throw new HttpError(409, `ticket is not '${expected_status}'`)
  })
}

module.exports = { list, get, create, update }
