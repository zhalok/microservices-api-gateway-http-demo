import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// The browser calls the gateway directly (see VITE_API_URL in src/api.js); the gateway allows
// the dev (5173) and preview (4173) origins via CORS.
export default defineConfig({
  plugins: [react()],
  server: { port: 5173 },
});
