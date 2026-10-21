import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// The dev server proxies API calls to the gateway, so the browser only talks to
// one origin and the gateway needs no CORS config.
const gateway = process.env.GATEWAY_URL || "http://localhost:8080";

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      "/tickets": gateway,
      "/bookings": gateway,
    },
  },
});
