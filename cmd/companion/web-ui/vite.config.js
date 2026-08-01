import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  base: './',
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    assetsInlineLimit: 8192,
    // wb-ui's JS engine (goja-based JSC) cannot parse minified code
    // reliably — keep the bundle readable so the desktop renderer can
    // execute the real companion frontend.
    minify: false,
    rollupOptions: {
      output: {
        // Single IIFE bundle (no dynamic-import chunks, no ES module
        // statements) — required for the wb-ui JS engine.
        format: 'iife',
        inlineDynamicImports: true,
      },
    },
  },
  server: {
    port: 5173,
    proxy: {
      '/api/terminal/ws': {
        target: 'ws://localhost:9090',
        ws: true,
      },
      '/api': 'http://localhost:9090',
      '/ws': {
        target: 'ws://localhost:9090',
        ws: true,
      },
    },
  },
})
