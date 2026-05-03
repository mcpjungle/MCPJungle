import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  base: "/ui/",
  plugins: [react()],
  build: {
    outDir: "../internal/ui/dist",
    emptyOutDir: false,
  },
  server: {
    host: "127.0.0.1",
    port: 5173,
  },
});
