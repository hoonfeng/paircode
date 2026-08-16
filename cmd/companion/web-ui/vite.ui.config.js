import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { fileURLToPath } from 'node:url'

// ═══════════════════════════════════════════════════════════════
// vite.ui.config.js — UI bundle 构建（全 UI 插件化）
//
// 把整个 IDE UI（ui-main.js → App.vue + 全部组件 + 状态）提前编译为
// 单个 IIFE bundle（window.__PAIRCODE_UI），输出到
//   <repo>/.pair/plugins/ui-app/assets/ui-app.js
// 由 ui-app 插件 client 半运行时加载（零运行时编译）。
//
// 与壳构建（vite.config.js）分离：壳 = index.html + 装载器（极薄）；
// UI 实现全部在插件资产里，可独立替换/删除/版本化，不重编译 Go。
// ═══════════════════════════════════════════════════════════════
export default defineConfig({
  plugins: [vue()],
  base: './',
  // UI bundle 不拷贝 public（favicon 等由壳 index.html 提供）
  publicDir: false,
  build: {
    // 输出到插件包资产目录（cmd/companion/web-ui → 3 级到 repo 根 .pair/plugins/ui-app/assets）
    outDir: fileURLToPath(new URL('../../../.pair/plugins/ui-app/assets', import.meta.url)),
    emptyOutDir: false,
    // ★ 保持未压缩（wb-ui jsc 对压缩单行的性能灾难，见 vite.config.js 注释）
    minify: false,
    rollupOptions: {
      input: fileURLToPath(new URL('./src/ui-main.js', import.meta.url)),
      output: {
        // IIFE + 无动态 import chunk：wb-ui JS 引擎兼容 + client 半 loadScript 直接执行
        format: 'iife',
        entryFileNames: 'ui-app.js',
        inlineDynamicImports: true,
      },
    },
  },
})
