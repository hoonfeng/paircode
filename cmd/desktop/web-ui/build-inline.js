// build-inline.js — 构建后处理脚本
// 读取 Vite 构建产物，将 JS 和 CSS 内联到 index.html 中，
// 生成 wb-ui 兼容的自包含 HTML（wb-ui LoadHTML 不支持外部 script/link）。
//
// 用法：node build-inline.js

import { readFileSync, writeFileSync, existsSync } from 'fs'
import { resolve, dirname } from 'path'
import { fileURLToPath } from 'url'

const __dirname = dirname(fileURLToPath(import.meta.url))
const distDir = resolve(__dirname, 'dist')

// 读取 index.html
const htmlPath = resolve(distDir, 'index.html')
if (!existsSync(htmlPath)) {
  console.error('dist/index.html 不存在，请先运行 npm run build')
  process.exit(1)
}

let html = readFileSync(htmlPath, 'utf-8')

// 1. 内联 CSS
html = html.replace(
  /<link rel="stylesheet"[^>]*href="\.\/([^"]+)"[^>]*>/g,
  (_, cssPath) => {
    const fullPath = resolve(distDir, cssPath)
    if (!existsSync(fullPath)) {
      console.warn('CSS 文件不存在:', fullPath)
      return ''
    }
    const css = readFileSync(fullPath, 'utf-8')
    return `<style>\n${css}\n</style>`
  }
)

// 2. 内联 JS（type="module" 脚本）
html = html.replace(
  /<script type="module"[^>]*src="\.\/([^"]+)"[^>]*><\/script>/g,
  (_, jsPath) => {
    const fullPath = resolve(distDir, jsPath)
    if (!existsSync(fullPath)) {
      console.warn('JS 文件不存在:', fullPath)
      return ''
    }
    const js = readFileSync(fullPath, 'utf-8')

    // 移除 export/import 语句（内联后不再需要 module 语法）
    const inlined = js
      .replace(/^export\s+default\s+/gm, 'return ')
      .replace(/^import\s+.*?from\s+['"].*?['"];?\s*$/gm, '')
      .replace(/^export\s+\{\s*.*?\s*\};?\s*$/gm, '')
      .replace(/^export\s+/gm, '')

    return `<script>\n${inlined}\n</script>`
  }
)

// 3. 移除所有 type="module" 和 crossorigin 属性
html = html.replace(/\s+type="module"/g, '')
html = html.replace(/\s+crossorigin="[^"]*"/g, '')

// 4. 移除动态 import()（vite 的预加载脚本可能包含 import()）
// 保留基本的 import 处理
html = html.replace(/<script>\s*import\s+['"]/g, '<script>null && import \'')

// 写入
writeFileSync(htmlPath, html, 'utf-8')
console.log('内联完成:', htmlPath)
console.log('大小:', (html.length / 1024).toFixed(1), 'KB')
