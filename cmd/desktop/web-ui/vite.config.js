import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import path from 'path'
import { fileURLToPath } from 'url'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)

// ── 与 companion/web-ui 共享同一份前端源码 ──
// cmd/desktop/web-ui → cmd/companion/web-ui = ../../companion/web-ui
const SHARED_SRC = path.resolve(__dirname, '../../companion/web-ui')

export default defineConfig({
  root: SHARED_SRC,
  plugins: [vue()],
  base: './',
  build: {
    outDir: path.resolve(__dirname, 'dist'),
    emptyOutDir: true,
    minify: false,
    assetsInlineLimit: 8192,
    cssCodeSplit: false,
    rollupOptions: {
      output: {
        format: 'iife',
        inlineDynamicImports: true,
        entryFileNames: 'assets/[name]-[hash].js',
        assetFileNames: 'assets/[name]-[hash][extname]',
      },
    },
  },
  server: {
    port: 5174,
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
