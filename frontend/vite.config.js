import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import wails from "@wailsio/runtime/plugins/vite";
import tailwindcss from "@tailwindcss/vite";
import { resolve } from "path";

const root = import.meta.dirname;

export default defineConfig({
  plugins: [react(), wails("./bindings"), tailwindcss()],
  base: "./",
  resolve: {
    alias: {
      "@": resolve(root, "./src"),
      bindings: resolve(root, "./bindings"),
    },
  },
  build: {
    chunkSizeWarningLimit: 850,
  },
});
