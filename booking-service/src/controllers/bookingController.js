const service = require("../services/bookingService")
const { withSpan } = require("../tracer")

async function list(_req, res) {
  await withSpan("BookingController.list", {}, async () => {
    res.json(await service.list())
  })
}

async function get(req, res) {
  await withSpan("BookingController.get", {}, async () => {
    res.json(await service.get(req.params.id))
  })
}

async function create(req, res) {
  await withSpan("BookingController.create", {}, async () => {
    res.status(201).json(await service.createWithLock(req.body))
  })
}

module.exports = { list, get, create }
