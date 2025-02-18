import { fileURLToPath, URL } from "node:url";

import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import vueDevTools from "vite-plugin-vue-devtools";

// https://vite.dev/config/
export default defineConfig({
  plugins: [vue(), vueDevTools()],

  resolve: {
    alias: {
      "@": fileURLToPath(new URL("./src", import.meta.url)),
    },
  },

  build: {
    // Tell Vite that it's okay to nuke our outDir on rebuild even though it's
    // outside our project root.
    emptyOutDir: true,

    rollupOptions: {
      output: {
        dir: "../client-dist",
      },
    },
  },

  server: {
    // When in dev mode proxy all API calls to the go server
    proxy: {
      "/api": {
        target: "http://localhost:12264",
        changeOrigin: true,
      },
    },
  },
});
