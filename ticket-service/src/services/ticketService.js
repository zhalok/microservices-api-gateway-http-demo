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

function update(id, { status, event, price }) {
  return withSpan("TicketService.update", { "ticket.id": id }, async () => {
    if (status && !STATUSES.includes(status))
      throw new HttpError(400, "status must be 'available' or 'booked'")

    const ticket = await repo.update(id, { status, event, price })
    if (!ticket) throw new HttpError(404, "ticket not found")
    return ticket
  })
}

module.exports = { list, get, create, update }
