// Must be required before express/pg so the instrumentations can patch them.
const { NodeSDK } = require("@opentelemetry/sdk-node")
const {
  OTLPTraceExporter,
} = require("@opentelemetry/exporter-trace-otlp-http")
const { HttpInstrumentation } = require("@opentelemetry/instrumentation-http")
const {
  ExpressInstrumentation,
} = require("@opentelemetry/instrumentation-express")
const { PgInstrumentation } = require("@opentelemetry/instrumentation-pg")

const sdk = new NodeSDK({
  serviceName: process.env.OTEL_SERVICE_NAME || "ticket-service",
  // Endpoint comes from OTEL_EXPORTER_OTLP_ENDPOINT (e.g. http://jaeger:4318)
  traceExporter: new OTLPTraceExporter(),
  instrumentations: [
    new HttpInstrumentation(),
    new ExpressInstrumentation(),
    new PgInstrumentation(),
  ],
})

sdk.start()

process.on("SIGTERM", () => {
  sdk.shutdown().finally(() => process.exit(0))
})
