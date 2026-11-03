const { Router } = require("express")
const controller = require("./controllers/ticketController")

// Express 4 doesn't catch rejected promises, so forward them to the error handler.
const wrap = (fn) => (req, res, next) => fn(req, res).catch(next)

const router = Router()

router.get("/health", (_req, res) => res.json({ status: "ok" }))

// List tickets, optionally filtered: /tickets?status=available
router.get("/tickets", wrap(controller.list))
router.get("/tickets/:id", wrap(controller.get))
router.post("/tickets", wrap(controller.create))
router.patch("/tickets/:id", wrap(controller.update))

module.exports = router
