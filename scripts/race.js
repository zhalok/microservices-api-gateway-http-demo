#!/usr/bin/env node
// Fires concurrent booking requests for alice and bob against the same ticket
// to demonstrate the race condition. Bob's request is delayed server-side
// (DELAYED_USER_NAME), so he reads the ticket as "available" before alice books it.
//
// Usage: node scripts/race.js [ticket_id]
// Env:   GATEWAY_URL (default http://localhost:8080)
//        USERS       (default "alice:alice123,bob:bob123")
//        TICKET_ID   (default 1; the CLI argument takes precedence)

const GATEWAY_URL = process.env.GATEWAY_URL || "http://localhost:8080"
const TICKET_ID = Number(process.argv[2] || process.env.TICKET_ID || 1)
const USERS = (process.env.USERS || "alice:alice123,bob:bob123")
  .split(",")
  .map((pair) => pair.trim().split(":"))

async function login(username, password) {
  const res = await fetch(`${GATEWAY_URL}/auth/login`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({ username, password }),
  })
  if (!res.ok) throw new Error(`login failed for ${username}: ${res.status}`)
  return (await res.json()).token
}

async function book(username, token, start) {
  const res = await fetch(`${GATEWAY_URL}/bookings`, {
    method: "POST",
    headers: {
      "content-type": "application/json",
      authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({ ticket_id: TICKET_ID, user_name: username }),
  })
  const body = await res.json().catch(() => null)
  return { username, status: res.status, ms: Date.now() - start, body }
}

async function main() {
  // Log in first so token fetching doesn't skew the timing of the race.
  const tokens = await Promise.all(
    USERS.map(async ([name, password]) => [name, await login(name, password)])
  )

  console.log(`Booking ticket ${TICKET_ID} concurrently as: ${USERS.map(([n]) => n).join(", ")}`)
  const start = Date.now()
  const results = await Promise.all(tokens.map(([name, token]) => book(name, token, start)))

  for (const r of results.sort((a, b) => a.ms - b.ms)) {
    console.log(`[+${r.ms}ms] ${r.username}: HTTP ${r.status} ${JSON.stringify(r.body)}`)
  }

  const wins = results.filter((r) => r.status >= 200 && r.status < 300)
  console.log(
    wins.length > 1
      ? `\nRACE CONDITION: ${wins.length} users booked the same ticket`
      : `\nOK: ${wins.length} booking succeeded`
  )
}

main().catch((err) => {
  console.error(err.message)
  process.exit(1)
})
