import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import path from "path";

// https://vite.dev/config/
export default defineConfig(({ mode }) => ({
  plugins: [react(), tailwindcss()],
  // Defense in depth for credential leaks: even if a console.log survives
  // review it is stripped from the production bundle. The eslint `no-console`
  // rule is the first line; this is the second.
  //
  // `pure` rather than `drop: ["console"]` so console.error/console.warn stay —
  // the error boundary needs them to report render crashes.
  esbuild:
    mode === "production"
      ? {
          pure: [
            "console.log",
            "console.debug",
            "console.info",
            "console.trace",
            "console.table",
            "console.dir",
          ],
          drop: ["debugger"] as "debugger"[],
        }
      : undefined,
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  server: {
    port: 3000,
    proxy: {
      "/api": {
        target: "http://localhost:8080",
        changeOrigin: true,
      },
    },
  },
}));
