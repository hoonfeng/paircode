#!/usr/bin/env node
// 准备 go:embed 目录：cmd/companion/web-ui/dist ← .pair/assets/runtime/web
//
// 为什么需要这一步：go:embed 在编译期读取磁盘，而 dist/ 被 .gitignore 忽略
// （前端产物由 npm 构建生成）。CI 只 checkout 仓库时 dist/ 不存在 → 编译报
// `pattern web-ui/dist: no matching files found`。
//
// 发布用的前端产物已随仓库入库在 .pair/assets/runtime/web/（运行时优先读取
// 该目录），因此直接复制即可，无需在 CI 重新跑 npm 构建。
//
// 幂等：dist/ 已存在时不覆盖（本地开发可能正在用 npm run build 的产物）。
import fs from 'node:fs'
import path from 'node:path'

const root = process.cwd()
const src = path.join(root, '.pair', 'assets', 'runtime', 'web')
const dst = path.join(root, 'cmd', 'companion', 'web-ui', 'dist')

if (!fs.existsSync(path.join(src, 'index.html'))) {
  console.error(`[prepare-embed] 缺少前端产物：${src}/index.html`)
  process.exit(1)
}
// 完整性校验：index.html 引用的本地资源必须都在。只入库 index.html 而漏掉
// assets/*.js（hash 产物）时，CI 打出的发布包 Web UI 会白屏——这里快速失败，
// 比等到用户装上才发现好。
{
  const html = fs.readFileSync(path.join(src, 'index.html'), 'utf8')
  const refs = [...html.matchAll(/(?:src|href)="(\.\/[^"]+)"/g)].map((m) => m[1])
  const missing = refs
    .map((r) => path.join(src, r))
    .filter((p) => !fs.existsSync(p))
  if (missing.length > 0) {
    console.error(
      '[prepare-embed] 前端产物不完整，缺少：\n' +
        missing.map((p) => '  ' + p).join('\n') +
        '\n请在 plugins-src/ui-app 执行 npm run build，并提交 .pair/assets/runtime/web/assets/ 下的产物。'
    )
    process.exit(1)
  }
}
if (fs.existsSync(dst)) {
  console.log(`[prepare-embed] 已存在，跳过：${dst}`)
  process.exit(0)
}
fs.mkdirSync(path.dirname(dst), { recursive: true })
fs.cpSync(src, dst, { recursive: true })
console.log(`[prepare-embed] 已复制 ${src} → ${dst}`)
