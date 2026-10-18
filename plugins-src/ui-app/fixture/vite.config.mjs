import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// 独立 fixture：复刻 IDE 的插件列表 / 输入框 / 市场卡片，单独构建到 _temp/vue-fixture-dist
// 定位：隔离 wb-ui 引擎缺陷（caret 不显示、滚动复位、层叠/裁剪异常），不依赖后端 API。
export default defineConfig({
  root: new URL('.', import.meta.url).pathname.replace(/^\/([A-Za-z]:)/, '$1'),
  plugins: [vue()],
  build: {
    outDir: 'F:/syproject/gou-ide/_temp/vue-fixture-dist',
    emptyOutDir: true,
    target: 'es2020',
  },
  server: { port: 5199 },
})
