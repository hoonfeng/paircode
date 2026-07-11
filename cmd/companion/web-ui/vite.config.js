import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  base: './',
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    assetsInlineLimit: 8192,
  },
  server: {
    port: 5173,
    proxy: {
      '/api': 'http://localhost:9090'
    }
  }
})
