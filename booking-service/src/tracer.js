const { trace, SpanStatusCode } = require("@opentelemetry/api")

const tracer = trace.getTracer("booking-service")

// Runs fn inside a new span that is a child of the currently active span.
// Expected 4xx errors are recorded but not marked as span failures.
function withSpan(name, attributes, fn) {
  return tracer.startActiveSpan(name, { attributes }, async (span) => {
    try {
      return await fn(span)
    } catch (err) {
      span.recordException(err)
      if (!err.status || err.status >= 500) {
        span.setStatus({ code: SpanStatusCode.ERROR, message: err.message })
      }
      throw err
    } finally {
      span.end()
    }
  })
}

module.exports = { withSpan }
