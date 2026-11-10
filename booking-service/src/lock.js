const crypto = require("crypto")
const Redis = require("ioredis")
const { withSpan } = require("./tracer")
const { HttpError } = require("./errors")

const redis = new Redis(process.env.REDIS_URL || "redis://localhost:6379")
redis.on("error", (err) => console.error("redis error", err.message))

const LOCK_TTL_MS = Number(process.env.LOCK_TTL_MS || 30000) // must exceed the longest critical section
const LOCK_WAIT_MS = Number(process.env.LOCK_WAIT_MS || 15000) // how long a caller waits for the lock
const RETRY_MS = 50

// Delete the key only if we still own it (token matches), so a lock that
// expired and was re-acquired by someone else is never released by mistake.
const RELEASE_SCRIPT = `
if redis.call("get", KEYS[1]) == ARGV[1] then
  return redis.call("del", KEYS[1])
end
return 0`

const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms))

// Runs fn while holding the distributed lock `key`. Callers on any
// booking-service instance are serialized per key.
async function withLock(key, fn) {
  const token = crypto.randomUUID()
  const deadline = Date.now() + LOCK_WAIT_MS

  await withSpan("Lock.acquire", { "lock.key": key }, async () => {
    while (!(await redis.set(key, token, "PX", LOCK_TTL_MS, "NX"))) {
      if (Date.now() >= deadline)
        throw new HttpError(503, "could not acquire lock, try again")
      await sleep(RETRY_MS)
    }
  })

  try {
    return await fn()
  } finally {
    await redis.eval(RELEASE_SCRIPT, 1, key, token).catch((err) =>
      console.error("lock release failed", err.message)
    )
  }
}

module.exports = { withLock }
