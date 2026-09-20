import react from "@vitejs/plugin-react";
import { defineConfig } from "vitest/config";

// The SPA is served under /xlearn by the gateway. Vite `base` makes the built
// asset URLs absolute under that prefix; the Router `basename` matches it. The
// gateway is base-path tolerant, so a running binary at :8080 answers /xlearn/*
// directly — hence the dev/preview proxy target below (ADR-0009 base-path model).
const API_PROXY = {
  "/xlearn/api": { target: "http://localhost:8080", changeOrigin: true },
};

export default defineConfig({
  base: "/xlearn/",
  plugins: [react()],
  server: { proxy: API_PROXY },
  preview: { proxy: API_PROXY },
  build: {
    outDir: "dist",
    sourcemap: false,
  },
  test: {
    environment: "jsdom",
    setupFiles: ["./src/test/setup.ts"],
    css: true,
    include: ["src/**/*.test.{ts,tsx}"],
  },
});
