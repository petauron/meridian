import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import { fileURLToPath } from "node:url";
import manifest from "./package.json" with { type: "json" };
import { reviewedCenterSource } from "./platform-source.mjs";

const centerSource = reviewedCenterSource();

export default defineConfig({
  plugins: [tailwindcss(), react()],
  resolve: {
    alias: { "@": centerSource },
    dedupe: ["react", "react-dom"],
  },
  build: {
    lib: {
      entry: fileURLToPath(new URL("./src/index.tsx", import.meta.url)),
      formats: ["es"],
      fileName: () => `ui-meridian-${manifest.version}.js`,
      cssFileName: `ui-meridian-${manifest.version}`,
    },
    rollupOptions: { output: { inlineDynamicImports: true } },
  },
});
