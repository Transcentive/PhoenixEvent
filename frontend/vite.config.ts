import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import path from "path";

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "src"),
    },
  },
  server: {
    // 5174, so PhoenixEvent and The Mighty Queue (5173) can run side by side.
    port: 5174,
    strictPort: true,
    // Each organization has its own address (spec 4A.5):
    // lincolnpta.phoenixevent.localhost:5174, and phoenixevent.localhost:5174
    // is the general address.
    allowedHosts: [".phoenixevent.localhost"],
    // The same job nginx does on the droplet: one origin for the client and
    // the API. The Host header passes through unchanged, as nginx passes it,
    // because the API tells organizations apart by it.
    proxy: {
      "/api/v1": {
        target: "http://127.0.0.1:8082",
        changeOrigin: false,
      },
    },
  },
});
