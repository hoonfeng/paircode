import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// 桌面端 Vite 配置：
// - 开发模式下不 proxy API 到 Go 后端（前端通过 desktopBridge 直接调 Go）
// - 定义 __DESKTOP_MODE__ 供 SDK 检测环境
// - 构建结果嵌入 Go binary（与 companion 相同模式）
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
    assetsInlineLimit: 8192,
  },
  server: {
    port: 5174, // 与 companion 的 5173 错开，避免端口冲突
  },
  define: {
    __DESKTOP_MODE__: JSON.stringify(true),
  },
})
