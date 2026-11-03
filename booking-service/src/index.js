require("./tracing") // must come first so express/pg/fetch get instrumented

const express = require("express")
const { init } = require("./db")
const routes = require("./routes")

const app = express()
app.use(express.json())

app.use((req, res, next) => {
  const start = Date.now()
  res.on("finish", () => {
    console.log(
      `${req.method} ${req.originalUrl} ${res.statusCode} ${Date.now() - start}ms`,
    )
  })
  next()
})

app.use(routes)

app.use((err, _req, res, _next) => {
  if (err.status && err.status < 500)
    return res.status(err.status).json({ error: err.message })
  console.error(err)
  res.status(err.status || 500).json({ error: err.status === 502 ? err.message : "internal error" })
})

async function start() {
  await init()
  const port = process.env.PORT || 3002
  app.listen(port, () => console.log(`booking-service listening on ${port}`))
}
start()
