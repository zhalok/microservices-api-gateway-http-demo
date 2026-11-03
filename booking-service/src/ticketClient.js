const { withSpan } = require("./tracer")
const { HttpError } = require("./errors")

const TICKET_SERVICE_URL = process.env.TICKET_SERVICE_URL

function getTicket(id) {
  return withSpan("TicketClient.getTicket", { "ticket.id": id }, async () => {
    const res = await fetch(`${TICKET_SERVICE_URL}/tickets/${id}`)
    if (res.status === 404) throw new HttpError(404, "ticket not found")
    if (!res.ok) throw new HttpError(502, "ticket service error")
    return res.json()
  })
}

// Returns the fetch Response so callers can distinguish 409 (compare-and-set lost).
function patchTicket(id, body) {
  return withSpan("TicketClient.patchTicket", { "ticket.id": id }, () =>
    fetch(`${TICKET_SERVICE_URL}/tickets/${id}`, {
      method: "PATCH",
      headers: { "content-type": "application/json" },
      body: JSON.stringify(body),
    }),
  )
}

module.exports = { getTicket, patchTicket }
