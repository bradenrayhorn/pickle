import { defineConfig } from "vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";

const basepath = new URL(import.meta.url).pathname.replace(
  "/vite.config.ts",
  "",
);

// https://vite.dev/config/
export default defineConfig({
  plugins: [svelte()],
  resolve: {
    alias: {
      "@wails": `${basepath}/wailsjs/go`,
      $lib: `${basepath}/src/lib`,
    },
  },
});
