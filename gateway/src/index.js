const crypto = require("crypto");
const express = require("express");
const jwt = require("jsonwebtoken");
const { createProxyMiddleware } = require("http-proxy-middleware");

const JWT_SECRET = process.env.JWT_SECRET;
const JWT_EXPIRES_IN = process.env.JWT_EXPIRES_IN || "1h";

// Demo user store: "alice:alice123,bob:bob123". Swap for a real user table in production.
const users = new Map(
  (process.env.AUTH_USERS || "alice:alice123,bob:bob123")
    .split(",")
    .map((pair) => pair.trim().split(":"))
);

const safeEqual = (a, b) => {
  const ha = crypto.createHash("sha256").update(String(a)).digest();
  const hb = crypto.createHash("sha256").update(String(b)).digest();
  return crypto.timingSafeEqual(ha, hb);
};

const app = express();

app.use((req, _res, next) => {
  console.log(`${req.method} ${req.originalUrl}`);
  next();
});

app.get("/health", (_req, res) => res.json({ status: "ok" }));

// Public: exchange credentials for a JWT.
app.post("/auth/login", express.json(), (req, res) => {
  const { username, password } = req.body || {};
  const expected = users.get(username);
  if (!username || !password || expected === undefined || !safeEqual(password, expected)) {
    return res.status(401).json({ error: "invalid username or password" });
  }
  const token = jwt.sign({ sub: username }, JWT_SECRET, { algorithm: "HS256", expiresIn: JWT_EXPIRES_IN });
  res.json({ token, username });
});

// Everything below requires `Authorization: Bearer <jwt>`.
function requireUser(req, res, next) {
  const [scheme, token] = (req.get("authorization") || "").split(" ");
  if (scheme !== "Bearer" || !token) return res.status(401).json({ error: "missing bearer token" });
  try {
    req.user = jwt.verify(token, JWT_SECRET, { algorithms: ["HS256"] });
    next();
  } catch {
    res.status(401).json({ error: "invalid or expired token" });
  }
}

app.get("/auth/me", requireUser, (req, res) => res.json({ username: req.user.sub }));

// Routing table: path prefix -> upstream service.
// Note: `app.use(path, ...)` strips the prefix, so we re-add it via pathRewrite.
const routes = [
  { prefix: "/tickets", target: process.env.TICKET_SERVICE_URL },
  { prefix: "/bookings", target: process.env.BOOKING_SERVICE_URL },
];

for (const { prefix, target } of routes) {
  app.use(
    prefix,
    requireUser,
    createProxyMiddleware({
      target,
      changeOrigin: true,
      pathRewrite: (path) => prefix + (path === "/" ? "" : path),
      on: {
        proxyReq: (proxyReq, req) => {
          // Internal services trust the gateway: drop the JWT, pass the verified user along.
          proxyReq.removeHeader("authorization");
          proxyReq.setHeader("x-user", req.user.sub);
        },
        error: (_err, _req, res) => {
          res.writeHead(502, { "content-type": "application/json" });
          res.end(JSON.stringify({ error: `upstream ${prefix} unavailable` }));
        },
      },
    })
  );
}

app.use((_req, res) => res.status(404).json({ error: "route not found" }));

if (!JWT_SECRET) throw new Error("JWT_SECRET is required");
const port = process.env.PORT || 8080;
app.listen(port, () => console.log(`gateway listening on ${port}`));
