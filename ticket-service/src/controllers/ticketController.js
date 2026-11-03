const service = require("../services/ticketService")
const { withSpan } = require("../tracer")

async function list(req, res) {
  await withSpan("TicketController.list", {}, async () => {
    res.json(await service.list(req.query.status))
  })
}

async function get(req, res) {
  await withSpan("TicketController.get", {}, async () => {
    res.json(await service.get(req.params.id))
  })
}

async function create(req, res) {
  await withSpan("TicketController.create", {}, async () => {
    res.status(201).json(await service.create(req.body))
  })
}

async function update(req, res) {
  await withSpan("TicketController.update", {}, async () => {
    res.json(await service.update(req.params.id, req.body))
  })
}

module.exports = { list, get, create, update }
