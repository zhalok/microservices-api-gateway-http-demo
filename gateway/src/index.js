const express = require("express");
const { createProxyMiddleware } = require("http-proxy-middleware");

const app = express();

app.use((req, _res, next) => {
  console.log(`${req.method} ${req.originalUrl}`);
  next();
});

app.get("/health", (_req, res) => res.json({ status: "ok" }));

// Routing table: path prefix -> upstream service.
// Note: `app.use(path, ...)` strips the prefix, so we re-add it via pathRewrite.
const routes = [
  { prefix: "/tickets", target: process.env.TICKET_SERVICE_URL },
  { prefix: "/bookings", target: process.env.BOOKING_SERVICE_URL },
];

for (const { prefix, target } of routes) {
  app.use(
    prefix,
    createProxyMiddleware({
      target,
      changeOrigin: true,
      pathRewrite: (path) => prefix + (path === "/" ? "" : path),
      on: {
        error: (_err, _req, res) => {
          res.writeHead(502, { "content-type": "application/json" });
          res.end(JSON.stringify({ error: `upstream ${prefix} unavailable` }));
        },
      },
    })
  );
}

app.use((_req, res) => res.status(404).json({ error: "route not found" }));

const port = process.env.PORT || 8080;
app.listen(port, () => console.log(`gateway listening on ${port}`));
