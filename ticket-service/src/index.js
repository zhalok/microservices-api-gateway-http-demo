require("./tracing") // must come first so express/pg get instrumented

const express = require("express")
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
  res.status(500).json({ error: "internal error" })
})

async function start() {
  const port = process.env.PORT || 3001
  app.listen(port, () => console.log(`ticket-service listening on ${port}`))
}
start()
