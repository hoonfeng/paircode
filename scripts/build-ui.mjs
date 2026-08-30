// ═══════════════════════════════════════════════════════════════
// build-ui.mjs — DSH 兼容分布式 UI 区域插件包构建调度器（2026-08-16 按槽位细粒度拆分）
//
// ★ 目标：把「单一脚本硬编码 9 区域一次性构建」改为「可逐插件发现的分布式装配」。
//   构建目标不再写死在本脚本里，而是从每个 UI 区域插件的 package.json 的
//   `dsh.ui.build` 段（manifest + 构建目标 + 输出目录）解耦发现。
//
// 产物：每个区域一个独立 IIFE bundle + 独立 css，输出到该插件 manifest 声明的
//   `dsh.ui.build.outDir`（如 .pair/plugins/ui-<region>/assets/），运行时经
//   /plugins-assets/<id>/assets/<file>.js 加载。壳（web-ui dist）由 vite.config.js
//   独立构建（含 __PAIRCODE_CORE）。
//
// ★ external 共享核心（区域 bundle 不打包，运行时从 window.__PAIRCODE_CORE 取）：
//   vue → __PAIRCODE_CORE.Vue            （Vue 单例，reactive 互通）
//   ui-state.js → __PAIRCODE_CORE.uiState（同一全局状态）
//   api.js → __PAIRCODE_CORE.api
//   plugin-runtime.js → __PAIRCODE_CORE.pluginRuntime（同一槽位注册表）
//   agent-events.js → __PAIRCODE_CORE.agentEvents
//   app-actions.js → __PAIRCODE_CORE.actions
//
// 用法（cwd=仓库根；依赖从 plugins-src/ui-app/node_modules 解析）：
//   node scripts/build-ui.mjs                  # 构建全部发现的 UI 区域插件包
//   node scripts/build-ui.mjs --list           # 只列出发现的区域包（不构建）
//   node scripts/build-ui.mjs --region <id>    # 只构建单个区域（如 --region editor）
// ═══════════════════════════════════════════════════════════════
import { createRequire } from 'node:module'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { readdirSync, readFileSync, rmSync, existsSync } from 'node:fs'

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
const pluginsDir = path.join(repoRoot, '.pair', 'plugins')

// ─── 区域发现（分布式：从 manifest 解耦，不再硬编码 9 区域）──────────
// 每 UI 区域/功能插件包的 package.json 含 `dsh.ui`（manifest，见契约 §3.1）与
// `dsh.ui.build`（构建目标：entry/global/outDir/fileName，见契约 §4.1）。
// 任一满足「含 dsh.ui 且含 dsh.ui.build」的包即为一独立可构建的分布式 UI 区域插件。
function discoverRegions() {
  const regions = []
  if (!existsSync(pluginsDir)) return regions
  for (const dir of readdirSync(pluginsDir)) {
    const pkgPath = path.join(pluginsDir, dir, 'package.json')
    if (!existsSync(pkgPath)) continue
    let pkg
    try { pkg = JSON.parse(readFileSync(pkgPath, 'utf8')) } catch { continue }
    const ui = pkg && pkg.dsh && pkg.dsh.ui
    if (!ui || !ui.build) continue
    regions.push({
      id: pkg.name,               // == package.json name（唯一装配 key）
      entry: ui.build.entry,      // 相对 plugins-src/ui-app（如 src/ui-main-titlebar.js）
      global: ui.build.global,    // IIFE 全局名（如 UiTitlebar / GitPanel）
      outDir: ui.build.outDir,    // 相对仓库根（如 .pair/plugins/ui-titlebar/assets）
      fileName: ui.build.fileName,// 产物基底名（如 ui-titlebar / git-panel）
      manifest: ui,
      // ★ region 短名：优先 manifest.dsh.ui.build.region（显式），否则从包名剥离 ui- 前缀
      //   （editor ↔ ui-editor；git-api/marketplace 无前缀即包名）。
      region: (ui.build.region) || pkg.name.replace(/^ui-/, ''),
    })
  }
  return regions
}

// external 匹配（组件里的相对导入 + vue）——共享核心单例契约（区域 bundle 不打包）
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

// 惰性加载 vite（仅实际构建时需要；--list 不需要，保证发现层无构建依赖）
let _vite = null
function getVite() {
  if (_vite) return _vite
  const require = createRequire(path.join(uiRoot, 'package.json'))
  _vite = { build: require('vite').build, vuePlugin: require('@vitejs/plugin-vue').default }
  return _vite
}

async function buildRegion(r) {
  const { build, vuePlugin } = getVite()
  const outDir = path.join(repoRoot, r.outDir)
  const fname = r.fileName || ('ui-' + r.id)
  console.log(`\n═══ 构建 ${r.id} (${r.entry}) → ${r.outDir} ═══`)
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
            // 产物固定名：<fileName>.js / <fileName>.css
            entryFileNames: `${fname}.js`,
            assetFileNames: `${fname}[extname]`,
          },
        },
      },
    })
    console.log(`✓ ${r.id} → ${outDir}`)
    return true
  } catch (e) {
    console.error(`✗ ${r.id} 构建失败:`, e && e.message || e)
    return false
  }
}

async function main() {
  const regions = discoverRegions()
  const argv = process.argv.slice(2)
  const listOnly = argv.includes('--list')
  const regionArg = argv.find((a, i) => a === '--region' && argv[i + 1]) ? argv[argv.indexOf('--region') + 1] : ''

  if (regions.length === 0) {
    console.log('[build-ui] 未发现任何可构建的 UI 区域插件（需含 dsh.ui + dsh.ui.build）')
    process.exit(0)
  }

  console.log(`[build-ui] 发现 ${regions.length} 个分布式 UI 区域插件包：`)
  for (const r of regions) {
    console.log(`  · ${r.id.padEnd(16)} entry=${r.entry}  out=${r.outDir}  global=${r.global}`)
  }

  if (listOnly) {
    console.log('\n[build-ui] --list：仅列出，未构建。')
    process.exit(0)
  }

  // 单区域构建（独立可构建）：--region <id>（匹配短名 region 或包名 id）
  let targets = regions
  if (regionArg) {
    targets = regions.filter(r => r.id === regionArg || r.region === regionArg)
    if (targets.length === 0) {
      console.error(`[build-ui] --region ${regionArg} 未匹配到已发现的区域插件（可用：${regions.map(r => r.region).join(', ')}）`)
      process.exit(1)
    }
    console.log(`[build-ui] --region ${regionArg}：仅构建该区域（${targets.map(r => r.id).join(', ')}）。`)
  }

  let failed = 0
  for (const r of targets) {
    if (!(await buildRegion(r))) failed++
  }
  console.log(failed ? `\n完成（${failed} 个失败）` : `\n全部 ${targets.length} 个区域构建成功`)
  process.exit(failed ? 1 : 0)
}

main()
