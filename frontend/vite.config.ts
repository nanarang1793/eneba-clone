/// <reference types="vitest/config" />
import { defineConfig, loadEnv } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

// In dev, /api is proxied to the Go server so the browser sees one origin
// (same setup as production, where Vercel rewrites /api to the backend).
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, "..", "VITE_");
  return {
    plugins: [react(), tailwindcss()],
    // `npm test` (Vitest): give tests an in-memory localStorage.
    test: { setupFiles: ["./src/test/setup.ts"] },
    server: {
      port: 5173,
      proxy: {
        "/api": env.VITE_API_PROXY_TARGET || "http://localhost:8080",
      },
      // In Docker on Windows, file-change events don't cross into the container,
      // so Vite has to poll for changes instead (set by docker-compose.yml).
      watch: env.VITE_USE_POLLING ? { usePolling: true, interval: 300 } : undefined,
    },
  };
});
