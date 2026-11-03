const { Router } = require("express")
const controller = require("./controllers/bookingController")

// Express 4 doesn't catch rejected promises, so forward them to the error handler.
const wrap = (fn) => (req, res, next) => fn(req, res).catch(next)

const router = Router()

router.get("/health", (_req, res) => res.json({ status: "ok" }))

router.get("/bookings", wrap(controller.list))
router.get("/bookings/:id", wrap(controller.get))
router.post("/bookings", wrap(controller.create))

module.exports = router
