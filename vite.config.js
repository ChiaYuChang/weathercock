import { build, defineConfig } from "vite";
import tailwindcss from "@tailwindcss/vite";

export default defineConfig({
    build: {
        outDir: "static/js",
        rollupOptions: {
            input: "src/js/main.js",
            output: {
                entryFileNames: "bundle.js",
            }
        }
    },
});