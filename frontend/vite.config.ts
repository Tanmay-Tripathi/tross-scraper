import path, { dirname } from "node:path";
import { fileURLToPath } from "node:url";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

const rootDir = dirname(fileURLToPath(import.meta.url));

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      "@": path.resolve(rootDir, "./src"),
    },
  },
  server: {
    port: 4204,
    strictPort: true,
  },
  preview: {
    port: 4204,
    strictPort: true,
  },
  build: {
    target: "es2020",
    sourcemap: false,
  },
});
