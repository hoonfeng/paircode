// ═══════════════════════════════════════════════════════════════
// build-ui.mjs — 7 个 UI 区域插件包构建脚本（2026-08-16 按槽位细粒度拆分）
//
// 产物：每个区域一个独立 IIFE bundle + 独立 css，输出到
//   .pair/plugins/ui-<region>/assets/（插件包 client.js 经 /plugins-assets/
//   加载）。壳（web-ui dist）由 vite.config.js 独立构建（含 __PAIRCODE_CORE）。
//
// ★ external 共享核心（区域 bundle 不打包，运行时从 window.__PAIRCODE_CORE 取）：
//   vue → __PAIRCODE_CORE.Vue            （Vue 单例，reactive 互通）
//   ui-state.js → __PAIRCODE_CORE.uiState（同一全局状态）
//   api.js → __PAIRCODE_CORE.api
//   plugin-runtime.js → __PAIRCODE_CORE.pluginRuntime（同一槽位注册表）
//   agent-events.js → __PAIRCODE_CORE.agentEvents
//   app-actions.js → __PAIRCODE_CORE.actions
//
// 运行：node scripts/build-ui.mjs（cwd=仓库根；依赖从 web-ui/node_modules 解析）
// ═══════════════════════════════════════════════════════════════
import { createRequire } from 'node:module'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { readdirSync, rmSync, existsSync } from 'node:fs'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const repoRoot = path.resolve(__dirname, '..')

// ★ Round3 ⑥.4 防堆积：构建前清理 cmd/companion/web-ui/dist 的历史 index-*.js
//   bundle（14 个历史产物问题根因：壳构建输出名带 hash，未引用旧包越积越多；
//   dist 为 //go:embed 兜底目录，只保留当前 index.html 引用的产物——
//   sync-web-dist.mjs 同步时也会清理，此处双保险防「先 build 后 sync」顺序漏网）。
const distDir = path.join(repoRoot, 'cmd', 'companion', 'web-ui', 'dist')
const distAssets = path.join(distDir, 'assets')
if (existsSync(distAssets)) {
  let removed = 0
  for (const f of readdirSync(distAssets)) {
    if (/^index-.*\.js$/.test(f)) {
      rmSync(path.join(distAssets, f), { force: true })
      removed++
    }
  }
  if (removed > 0) console.log(`[build-ui] 预清理 ${removed} 个历史 index-*.js bundle（${distAssets}）`)
}

// ★ 前端源码位于项目根独立目录 plugins-src/ui-app/（2026-08-17 迁移：.pair/plugins/ui-app-src →
//   plugins-src/ui-app/，源码独立于插件目录；node_modules 为 junction 指向
//   cmd/companion/web-ui/node_modules）
const uiRoot = path.join(repoRoot, 'plugins-src', 'ui-app')
const require = createRequire(path.join(uiRoot, 'package.json'))
const { build } = require('vite')
const vuePlugin = require('@vitejs/plugin-vue').default

// ─── 区域清单 ──────────────────────────────────────────────
const regions = [
  { id: 'titlebar',    entry: 'src/ui-main-titlebar.js',    global: 'UiTitlebar' },
  { id: 'activitybar', entry: 'src/ui-main-activitybar.js', global: 'UiActivitybar' },
  { id: 'sidebar',     entry: 'src/ui-main-sidebar.js',     global: 'UiSidebar' },
  { id: 'editor',      entry: 'src/ui-main-editor.js',      global: 'UiEditor' },
  { id: 'right-panel', entry: 'src/ui-main-right-panel.js', global: 'UiRightPanel' },
  { id: 'statusbar',   entry: 'src/ui-main-statusbar.js',   global: 'UiStatusbar' },
  { id: 'modals',      entry: 'src/ui-main-modals.js',      global: 'UiModals' },
  // git-api 插件 Git 面板（接口+UI 一体化，输出到插件包 assets）
  { id: 'git-api', entry: 'src/ui-main-git.js', global: 'GitPanel', outDir: '.pair/plugins/git-api/assets', fileName: 'git-panel' },
  // marketplace 插件市场面板（市场功能全插件化，输出到插件包 assets）
  { id: 'marketplace', entry: 'src/ui-main-marketplace.js', global: 'MarketplacePanel', outDir: '.pair/plugins/marketplace/assets', fileName: 'marketplace-panel' },
]

// external 匹配（组件里的相对导入 + vue）
const CORE_MODULES = ['ui-state.js', 'api.js', 'plugin-runtime.js', 'agent-events.js', 'app-actions.js']
function isExternal(id) {
  if (id === 'vue') return true
  const name = String(id).split(/[\\/]/).pop()
  return CORE_MODULES.includes(name)
}
function globalFor(id) {
  if (id === 'vue') return 'window.__PAIRCODE_CORE.Vue'
  const name = String(id).split(/[\\/]/).pop()
  const map = {
    'ui-state.js': 'window.__PAIRCODE_CORE.uiState',
    'api.js': 'window.__PAIRCODE_CORE.api',
    'plugin-runtime.js': 'window.__PAIRCODE_CORE.pluginRuntime',
    'agent-events.js': 'window.__PAIRCODE_CORE.agentEvents',
    'app-actions.js': 'window.__PAIRCODE_CORE.actions',
  }
  return map[name]
}

let failed = 0
for (const r of regions) {
  const outDir = r.outDir ? path.join(repoRoot, r.outDir) : path.join(repoRoot, '.pair/plugins/ui-' + r.id, 'assets')
  const fname = r.fileName || ('ui-' + r.id)
  console.log(`\n═══ 构建 ui-${r.id} (${r.entry}) ═══`)
  try {
    await build({
      configFile: false,
      root: uiRoot,
      plugins: [vuePlugin()],
      logLevel: 'warn',
      build: {
        lib: {
          entry: path.join(uiRoot, r.entry),
          name: r.global,
          formats: ['iife'],
        },
        outDir,
        emptyOutDir: false,
        minify: 'esbuild',
        cssCodeSplit: false,
        assetsInlineLimit: 1000000, // 小资源内联（logo.svg 等）
        rollupOptions: {
          external: isExternal,
          output: {
            globals: globalFor,
            // 产物固定名：ui-<region>.js / ui-<region>.css
            entryFileNames: `${fname}.js`,
            assetFileNames: `${fname}[extname]`,
          },
        },
      },
    })
    console.log(`✓ ui-${r.id} → ${outDir}`)
  } catch (e) {
    failed++
    console.error(`✗ ui-${r.id} 构建失败:`, e && e.message || e)
  }
}
console.log(failed ? `\n完成（${failed} 个失败）` : `\n全部 ${regions.length} 个区域构建成功`)
process.exit(failed ? 1 : 0)
