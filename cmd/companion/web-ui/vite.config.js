import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  base: './',
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    assetsInlineLimit: 8192,
    // ★ 启动加速尝试：minify + es2015 实测 jsc 可解析但运行时帧率掉
    // ~100 倍（goja 对压缩单行的性能灾难）→ 回滚保持未压缩。启动慢的
    // 根因是 bundle 大，优化方向见「终端预加载」方案（不依赖 minify）。
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
