import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [
    vue(),
    {
      name: 'desktop-mode-inject',
      transformIndexHtml(html) {
        return {
          html,
          tags: [
            {
              tag: 'script',
              attrs: { type: 'text/javascript' },
              children: 'window.__DESKTOP_MODE__ = true;',
              injectTo: 'head-prepend',
            },
          ],
        }
      },
    },
  ],
  base: './',
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    target: 'es2015',
    cssCodeSplit: false,
    assetsInlineLimit: 8192,
    rollupOptions: {
      output: {
        format: 'iife',
        name: 'PairCodeIDE',
        inlineDynamicImports: true,
        entryFileNames: 'assets/bundle.js',
        chunkFileNames: 'assets/bundle.js',
        assetFileNames: 'assets/[name].[ext]',
      },
    },
  },
  server: {
    port: 5174,
  },
  define: {
    __DESKTOP_MODE__: JSON.stringify(true),
  },
})
