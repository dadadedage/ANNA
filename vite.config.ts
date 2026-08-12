import { defineConfig } from "vite";

export default defineConfig({
  // Anna serves the bundle beneath its own path, so assets cannot be root-relative.
  base: "./",
  build: {
    outDir: "bundle",
    emptyOutDir: true,
  },
});
