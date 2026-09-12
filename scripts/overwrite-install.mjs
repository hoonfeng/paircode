#!/usr/bin/env node
/**
 * overwrite-install.mjs —— PairCode「完整覆盖安装」
 * =============================================================================
 * 做什么
 * ------
 * 把仓库最新构建产物（默认 release/PairCode，packager 的完整产出）**完整覆盖**到
 * 安装目录（默认自动探测，如 E:/Program Files (x86)/PairCode）：
 *
 *   ① 杀掉安装目录里正在运行的 pair.exe（脚本自身是 node 进程，不受影响，继续跑）
 *   ② 等待进程退出 / 端口释放
 *   ③ 备份安装目录（除保护项，可 --no-backup 跳过）
 *   ④ 清理安装目录 —— 除保护项外**全部删除**（避免旧文件残留导致插件/前端版本不一致）
 *   ⑤ 从源目录完整拷贝覆盖
 *   ⑥ 校验文件清单 + pair.exe md5 + 关键文件
 *   ⑦ 可选 --restart 重新拉起
 *
 * 保护项（默认）
 * --------------
 *   config/    ← 用户配置（settings.json 内含 API Key / mcp.json / models.json /
 *                ai-presets.json / roles / philosophy / skills）。源目录里的 config/
 *                只有脱敏后的 skills/（packager 的 stripSecrets 会清空密钥），
 *                一旦覆盖 = 配置全丢，所以**永不覆盖**。
 *   --protect <rel>  追加保护（可多次；支持 .pair/plugins/xxx 这类相对路径）
 *   --keep-logs      额外保护 logs/
 *
 * 用法
 * ----
 *   node scripts/overwrite-install.mjs                         # DRY-RUN：只打印计划（默认）
 *   node scripts/overwrite-install.mjs --apply                 # 执行完整覆盖安装
 *   node scripts/overwrite-install.mjs --apply --restart       # 覆盖后自动重启
 *   node scripts/overwrite-install.mjs --apply --protect .pair/plugins/moc3-tools
 *   node scripts/overwrite-install.mjs --apply --src release/PairCode \
 *        --install-dir "E:/Program Files (x86)/PairCode" --port 9090
 *
 * 参数
 * ----
 *   --apply              真正写入（不加则 dry-run）
 *   --src <dir>          源目录（默认 release/PairCode，须是完整安装结构）
 *   --install-dir <dir>  安装目录（默认：运行实例反查 > 常见安装路径探测）
 *   --protect <rel>      追加保护路径（可多次；逗号分隔亦可）
 *   --keep-logs          保护 logs/ 目录
 *   --restart            覆盖完成后重新启动 pair.exe
 *   --port <n>           重启端口（默认沿用被替换实例的监听端口，否则 9090）
 *   --no-backup          跳过备份（默认备份到 _temp/install-backup-<ts>/）
 *   --force              跳过预检硬失败（源不完整 / 无写权限）
 *
 * 重要前提
 * --------
 *   · 安装到 Program Files 需要**管理员权限**；脚本会先做写权限自检，不通过会明确提示。
 *   · 源目录必须是最新构建：改完代码先跑 packager.exe（或至少
 *     `node scripts/sync-plugins-to-bin.mjs` + go build），否则覆盖的是旧版本。
 *     packager 常用：`./packager.exe`（全流程）/ `./packager.exe --step build-go`（只编译）
 *   · 换 exe 必须关进程（Windows 不允许覆盖运行中的 exe）——本脚本负责这一步。
 *
 * 与 deploy-to-install.mjs 的区别
 * -------------------------------
 *   deploy-to-install.mjs：**增量**（只同步 pair.exe + agentloop 插件），不删旧文件，适合快速迭代。
 *   overwrite-install.mjs：**完整覆盖**（清空重建，config 除外），适合发布/版本对齐，
 *                          能消除旧文件残留（前端哈希产物、已移除插件等）。
 */
import fs from 'node:fs'
import path from 'node:path'
import os from 'node:os'
import crypto from 'node:crypto'
import { execFileSync, spawn } from 'node:child_process'
import { fileURLToPath } from 'node:url'

// 路径归一化（posix 风格、去尾斜杠）—— 参数区/保护清单即刻要用，故置于最前
const norm = (p) => String(p).replace(/\\/g, '/').replace(/\/+$/, '')

// ───────────────────────────── 参数解析 ─────────────────────────────
const argv = process.argv.slice(2)
const has = (f) => argv.includes(f)
const argOf = (n) => { const i = argv.indexOf(n); return i >= 0 ? argv[i + 1] : null }
const argAll = (n) => {
  const out = []
  for (let i = 0; i < argv.length; i++) if (argv[i] === n && i + 1 < argv.length) out.push(argv[i + 1])
  return out
}

const APPLY = has('--apply')
const RESTART = has('--restart')
const NO_BACKUP = has('--no-backup')
const KEEP_LOGS = has('--keep-logs')
const FORCE = has('--force')
const PORT_ARG = argOf('--port') ? Number(argOf('--port')) : null
const SRC_ARG = argOf('--src')
const DIR_ARG = argOf('--install-dir')
const PROTECT_EXTRA = argAll('--protect').flatMap((v) => v.split(',')).map((s) => s.trim()).filter(Boolean)

// 仓库根：优先用脚本自身位置推断（scripts/ 的上一级），避免 cwd 不同导致路径错乱
const SELF_DIR = path.dirname(fileURLToPath(import.meta.url))
const REPO = argOf('--repo') || path.dirname(SELF_DIR)
const SRC = path.resolve(REPO, SRC_ARG || path.join('release', 'PairCode'))

const PROTECT_DEFAULT = ['config']
const PROTECT = (KEEP_LOGS ? [...PROTECT_DEFAULT, 'logs'] : PROTECT_DEFAULT)
  .concat(PROTECT_EXTRA)
  .map(norm)

const CANDIDATES = [
  'E:/Program Files (x86)/PairCode',
  'E:/Program Files/PairCode',
  'C:/Program Files (x86)/PairCode',
  'C:/Program Files/PairCode',
  path.join(os.homedir(), 'PairCode'),
]

// 必须存在的源关键文件（判断源是否为「完整安装结构」）
const REQUIRED_SRC = [
  'pair.exe',
  'assets/icon.png',
  '.pair/plugins/agentloop/index.js',
  '.pair/assets/runtime/web/index.html',
  'plugins-src/ui-app/package.json',
]

// ───────────────────────────── 工具函数 ─────────────────────────────
const sleep = (ms) => new Promise((r) => setTimeout(r, ms))
const abs = (p) => path.resolve(p)
const md5 = (p) => crypto.createHash('md5').update(fs.readFileSync(p)).digest('hex')
const mb = (n) => (n / 1048576).toFixed(1) + ' MB'
const exists = (p) => { try { return fs.existsSync(p) } catch { return false } }

/** 是否命中保护清单（rel 为相对安装目录/源目录的 posix 路径） */
function isProtected(rel) {
  const r = norm(rel)
  return PROTECT.some((p) => r === p || r.startsWith(p + '/'))
}

/** 保护清单中是否存在以 rel 为父路径的项（有 → 该目录不能整删，需深入清理） */
function hasProtectedDescendant(rel) {
  const r = norm(rel)
  return PROTECT.some((p) => p.startsWith(r + '/'))
}

/** 删除 dir 下所有非保护项（保护项及其父链保留），返回删除的文件数 */
function pruneExceptProtected(dir, relPrefix) {
  let n = 0
  for (const e of fs.readdirSync(dir, { withFileTypes: true })) {
    const rel = relPrefix ? `${relPrefix}/${e.name}` : e.name
    const full = path.join(dir, e.name)
    if (isProtected(rel)) continue                              // 保护子树整体跳过
    if (e.isDirectory()) {
      if (hasProtectedDescendant(rel)) n += pruneExceptProtected(full, rel)
      else { n += walk(full).length; rmrf(full) }
    } else {
      try { fs.rmSync(full, { force: true }); n++ } catch { /* 占用/权限 */ }
    }
  }
  return n
}

/** 递归列出文件 → [{ rel, size }] */
function walk(root) {
  const out = []
  const rec = (d) => {
    let entries
    try { entries = fs.readdirSync(d, { withFileTypes: true }) } catch { return }
    for (const e of entries) {
      const p = path.join(d, e.name)
      try {
        if (e.isDirectory()) rec(p)
        else if (e.isFile()) out.push({ rel: norm(path.relative(root, p)), size: fs.statSync(p).size })
      } catch { /* 权限/占用：跳过 */ }
    }
  }
  if (exists(root)) rec(root)
  return out
}

function copyTree(src, dst) {
  const st = fs.statSync(src)
  if (st.isDirectory()) {
    fs.mkdirSync(dst, { recursive: true })
    for (const e of fs.readdirSync(src)) copyTree(path.join(src, e), path.join(dst, e))
  } else {
    fs.mkdirSync(path.dirname(dst), { recursive: true })
    fs.copyFileSync(src, dst)
  }
}

function rmrf(p) { fs.rmSync(p, { recursive: true, force: true }) }

function dirSize(root) { return walk(root).reduce((a, x) => a + x.size, 0) }

function canWrite(dir) {
  const t = path.join(dir, `.owrtest-${process.pid}-${Date.now()}`)
  try { fs.writeFileSync(t, 'x'); fs.rmSync(t, { force: true }); return true } catch { return false }
}

function isAdmin() {
  try {
    const out = execFileSync('powershell', ['-NoProfile', '-NonInteractive', '-Command',
      '([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)'],
      { encoding: 'utf8', windowsHide: true })
    return /true/i.test(out)
  } catch { return false }
}

// ───────────────────────── 进程 / 端口 ─────────────────────────
/** 列出所有 pair.exe 进程 → [{ pid, exe }] */
function listPairProcs() {
  if (process.platform !== 'win32') return []
  const ps = `Get-CimInstance Win32_Process -Filter "Name='pair.exe'" | Select-Object ProcessId,ExecutablePath | ConvertTo-Json -Compress`
  let out = ''
  try {
    out = execFileSync('powershell', ['-NoProfile', '-NonInteractive', '-Command', ps],
      { encoding: 'utf8', windowsHide: true })
  } catch { return [] }
  out = out.trim()
  if (!out) return []
  let j
  try { j = JSON.parse(out) } catch { return [] }
  return (Array.isArray(j) ? j : [j])
    .filter((x) => x && x.ProcessId)
    .map((x) => ({ pid: Number(x.ProcessId), exe: norm(x.ExecutablePath || '') }))
}

/** 某 PID 正在 LISTENING 的端口列表 */
function portsOf(pid) {
  try {
    const out = execFileSync('netstat', ['-ano'], { encoding: 'utf8' })
    return out.split(/\r?\n/)
      .filter((l) => /LISTENING/i.test(l) && l.trim().split(/\s+/).pop() === String(pid))
      .map((l) => { const m = (l.trim().split(/\s+/)[1] || '').match(/:(\d+)$/); return m ? Number(m[1]) : null })
      .filter(Boolean)
  } catch { return [] }
}

function isListening(port) {
  try {
    const out = execFileSync('netstat', ['-ano'], { encoding: 'utf8' })
    return out.split(/\r?\n/).some((l) => /LISTENING/i.test(l) && new RegExp(`[:.]${port}[\\s]`).test(l))
  } catch { return false }
}

function killPid(pid) {
  try {
    execFileSync('taskkill', ['/PID', String(pid), '/T', '/F'], { encoding: 'utf8', windowsHide: true })
    return true
  } catch (e) {
    return String(e.stdout || e.message || '').includes('成功') || String(e.stdout || '').includes('SUCCESS')
  }
}

/** 等待给定 PID 全部退出，且（若指定）端口不再监听 */
async function waitGone(pids, port, timeoutMs = 20000) {
  const t0 = Date.now()
  let last = ''
  while (Date.now() - t0 < timeoutMs) {
    const alive = listPairProcs().filter((p) => pids.includes(p.pid))
    const listen = port ? isListening(port) : false
    if (!alive.length && !listen) return { ok: true, waited: Date.now() - t0 }
    last = `仍在运行: PID ${alive.map((p) => p.pid).join(',') || '无'}${listen ? ` / 端口 ${port} 仍被占用` : ''}`
    await sleep(400)
  }
  return { ok: false, waited: Date.now() - t0, last }
}

// ───────────────────────────── 探测安装目录 ─────────────────────────────
function detectInstallDir() {
  if (DIR_ARG) return abs(DIR_ARG)
  // ① 运行中的实例（排除仓库内的开发实例）
  const repoN = norm(abs(REPO)).toLowerCase()
  for (const p of listPairProcs()) {
    const dir = p.exe ? p.exe.replace(/\/[^/]*$/, '') : ''
    if (!dir) continue
    if (norm(dir).toLowerCase().startsWith(repoN)) continue
    if (exists(path.join(dir, 'pair.exe'))) return dir
  }
  // ② 常见安装路径
  for (const c of CANDIDATES) if (exists(path.join(c, 'pair.exe'))) return c
  return null
}

// ───────────────────────────── 主流程 ─────────────────────────────
function banner(t) {
  console.log('\n' + '─'.repeat(72))
  console.log(t)
  console.log('─'.repeat(72))
}

banner('PairCode 完整覆盖安装' + (APPLY ? '【APPLY】' : '【DRY-RUN 预演】'))
console.log('仓库根        :', REPO)
console.log('源目录        :', SRC)
console.log('保护项        :', PROTECT.join(', ') || '(无)')

// ── 源检查 ──
if (!exists(SRC)) {
  console.error(`\n✗ 源目录不存在: ${SRC}`)
  console.error('  → 先构建：./packager.exe（完整流程）或指定 --src <dir>')
  process.exit(1)
}
const missingSrc = REQUIRED_SRC.filter((r) => !exists(path.join(SRC, r)))
if (missingSrc.length) {
  console.error(`\n✗ 源目录结构不完整，缺少: ${missingSrc.join(', ')}`)
  if (!FORCE) { console.error('  → 不是有效安装结构；确认 --src 或加 --force 强行继续'); process.exit(1) }
  console.warn('  ⚠ --force：忽略该问题继续')
}
const srcFiles = walk(SRC)
const srcBytes = srcFiles.reduce((a, x) => a + x.size, 0)
console.log(`源内容        : ${srcFiles.length} 个文件 / ${mb(srcBytes)}`)

// ── 安装目录 ──
const INSTALL = detectInstallDir()
if (!INSTALL) {
  console.error('\n✗ 未探测到安装目录（可用 --install-dir "E:/Program Files (x86)/PairCode" 指定）')
  process.exit(1)
}
console.log('安装目录      :', INSTALL)

if (!exists(INSTALL)) {
  console.error(`\n✗ 安装目录不存在: ${INSTALL}`)
  process.exit(1)
}

// 防呆：源与安装目录不能相同 / 互相包含
const nSrc = norm(abs(SRC)).toLowerCase(), nIns = norm(abs(INSTALL)).toLowerCase()
if (nSrc === nIns) { console.error('\n✗ 源目录与安装目录相同，拒绝执行'); process.exit(1) }
if (nIns.startsWith(nSrc + '/')) { console.error('\n✗ 安装目录位于源目录内部，拒绝执行'); process.exit(1) }

const writable = canWrite(INSTALL)
const admin = process.platform === 'win32' ? isAdmin() : true

// ── 进程 ──
const procs = listPairProcs()
const installN = norm(abs(INSTALL)).toLowerCase()
const targetProcs = procs.filter((p) => norm(p.exe).toLowerCase() === installN + '/pair.exe')
const otherProcs = procs.filter((p) => !targetProcs.includes(p))
const listenPorts = [...new Set(targetProcs.flatMap((p) => portsOf(p.pid)))]

console.log('\n运行中的 pair.exe:')
if (!procs.length) console.log('  (无)')
for (const p of targetProcs) console.log(`  → PID ${p.pid}  ${p.exe}   【本脚本将结束它】`)
for (const p of otherProcs) console.log(`  · PID ${p.pid}  ${p.exe}   【保留，不在覆盖范围】`)

// ── 差异 ──
const srcMap = new Map(srcFiles.filter((x) => !isProtected(x.rel)).map((x) => [x.rel, x.size]))
const dstFiles = walk(INSTALL)
const dstMap = new Map(dstFiles.filter((x) => !isProtected(x.rel)).map((x) => [x.rel, x.size]))

const toDelete = [...dstMap.keys()].filter((k) => !srcMap.has(k)).sort()
const toAdd = [...srcMap.keys()].filter((k) => !dstMap.has(k)).sort()
const common = [...srcMap.keys()].filter((k) => dstMap.has(k)).sort()
const changed = common.filter((k) => {
  try { return md5(path.join(SRC, k)) !== md5(path.join(INSTALL, k)) } catch { return false }
})
const protectedFiles = dstFiles.filter((x) => isProtected(x.rel))

console.log(`\n安装目录现状  : ${dstFiles.length} 个文件（其中受保护 ${protectedFiles.length} 个，不参与覆盖）`)
console.log('差异概览:')
console.log(`  · 将删除（旧版残留 / 已移除项）: ${toDelete.length}`)
console.log(`  · 将新增（新版新增文件）      : ${toAdd.length}`)
console.log(`  · 将更新（内容不同）          : ${changed.length}`)
console.log(`  · 保持不变                    : ${common.length - changed.length}`)

// 风险项：源里没有、但看起来属于「用户资产」的文件
const pluginsKeep = [...new Set(toDelete.filter((r) => r.startsWith('.pair/plugins/')).map((r) => r.split('/').slice(0, 3).join('/')))]
const toolsetsExtra = toDelete.filter((r) => r.startsWith('.pair/toolsets/') && r.endsWith('.json'))
if (pluginsKeep.length || toolsetsExtra.length) {
  console.log('\n⚠ 以下项在源目录中不存在，将被删除 —— 若非程序残留，请用 --protect 保留：')
  for (const p of pluginsKeep) console.log(`    --protect ${p}`)
  // 内置预设会在新实例启动时按 .preset-seeded 自动重建；只有自建工具集会真正丢失
  const BUILTIN = ['基础', '调试', '办公', '全功能', '全栈开发', '计划讨论']
  const isBuiltin = (r) => BUILTIN.includes(path.basename(r, '.json'))
  const customTs = toolsetsExtra.filter((r) => !isBuiltin(r))
  const builtinTs = toolsetsExtra.filter(isBuiltin)
  if (customTs.length) {
    console.log(`    --protect .pair/toolsets   （含 ${customTs.length} 个**自建**工具集，删了不会重建：`)
    for (const r of customTs) console.log(`         ${r}`)
    console.log('       ）')
  }
  if (builtinTs.length) {
    console.log(`    （另 ${builtinTs.length} 个内置预设会自动重建，无需保护：${builtinTs.map((r) => path.basename(r, '.json')).join('、')}）`)
  }
  console.log('  提示：logs/ 等运行期数据也会在新实例启动时自动重建，无需保护。')
}

if (toDelete.length) {
  console.log('\n将删除的文件:')
  for (const r of toDelete.slice(0, 40)) console.log('  - ' + r)
  if (toDelete.length > 40) console.log(`  ...（其余 ${toDelete.length - 40} 个同类省略）`)
}
if (changed.length) {
  console.log('\n将更新的文件:')
  for (const r of changed.slice(0, 30)) console.log('  ~ ' + r)
  if (changed.length > 30) console.log(`  ...（其余 ${changed.length - 30} 个省略）`)
}

banner('预检')
console.log(`写权限        : ${writable ? '✓ 可写' : '✗ 不可写'}`)
console.log(`管理员权限    : ${admin ? '✓ 是' : '✗ 否'}`)
if (!writable) {
  console.log('\n✗ 安装目录不可写：Program Files 需要管理员权限。')
  console.log('  → 用「以管理员身份运行」的终端重跑本脚本' + (FORCE ? '（--force 已指定，仍将继续尝试）' : ''))
  if (!FORCE) process.exit(1)
}

if (!APPLY) {
  console.log('\n(DRY-RUN 结束，未写入任何文件)')
  console.log('确认无误后执行:')
  console.log(`  node scripts/overwrite-install.mjs --apply${RESTART ? ' --restart' : ''}`)
  process.exit(0)
}

// ───────────────────────────── 执行 ─────────────────────────────
const t0 = Date.now()

// ① 杀进程
banner('① 结束运行中的实例')
if (!targetProcs.length) {
  console.log('无目标实例在运行，跳过')
} else {
  for (const p of targetProcs) {
    const ok = killPid(p.pid)
    console.log(`${ok ? '✓' : '✗'} taskkill /PID ${p.pid} ${ok ? '已结束' : '失败（可能需要管理员权限）'}`)
  }
  const waited = await waitGone(targetProcs.map((p) => p.pid), listenPorts[0] ?? null)
  console.log(waited.ok
    ? `✓ 进程已退出，端口已释放（耗时 ${(waited.waited / 1000).toFixed(1)}s）`
    : `⚠ 等待超时：${waited.last}`)
  if (!waited.ok) {
    console.log('  → 继续尝试覆盖（exe 被占用时会失败并提示）')
  }
}

// ② 备份
const ts = new Date().toISOString().replace(/[:.]/g, '-').slice(0, 19)
let bakDir = null
if (!NO_BACKUP) {
  banner('② 备份安装目录（保护项除外）')
  bakDir = path.join(REPO, '_temp', `install-backup-${ts}`)
  fs.mkdirSync(bakDir, { recursive: true })
  const entries = fs.readdirSync(INSTALL).filter((e) => !isProtected(e))
  let n = 0, bytes = 0
  for (const e of entries) {
    const s = path.join(INSTALL, e)
    copyTree(s, path.join(bakDir, e))
    const st = fs.statSync(s)
    if (st.isDirectory()) { const f = walk(s); n += f.length; bytes += f.reduce((a, x) => a + x.size, 0) }
    else { n++; bytes += st.size }
  }
  console.log(`✓ 已备份 ${n} 个文件 / ${mb(bytes)} → ${path.relative(REPO, bakDir)}`)
  console.log(`  回滚：删除安装目录内容（保留 config）后，把备份内容拷回 ${INSTALL}`)
} else {
  banner('② 备份（已跳过 --no-backup）')
}

// ③ 清理
banner('③ 清理安装目录（保护项除外）')
let delTop = 0, delFiles = 0
for (const e of fs.readdirSync(INSTALL)) {
  const p = path.join(INSTALL, e)
  const isDir = fs.statSync(p).isDirectory()
  const cnt = isDir ? walk(p).length : 1
  if (isProtected(e)) {
    console.log(`  · 保留 ${e}${e === 'config' ? '（用户配置）' : ''}  (${cnt} 个文件)`)
    continue
  }
  try {
    if (isDir && hasProtectedDescendant(e)) {
      // 目录内含受保护后代（如 --protect .pair/plugins/moc3-tools）→ 只删未保护部分
      const removed = pruneExceptProtected(p, e)
      delTop++; delFiles += removed
      console.log(`  ✓ 清理 ${e}  (删除 ${removed} / 保留 ${cnt - removed} 个文件)`)
    } else {
      rmrf(p)
      delTop++; delFiles += cnt
      console.log(`  ✓ 删除 ${e}  (${cnt} 个文件)`)
    }
  } catch (err) {
    console.log(`  ✗ 删除 ${e} 失败: ${err.message}`)
    if (e === 'pair.exe') console.log('    → exe 仍被占用：确认实例已关闭或使用管理员权限')
  }
}

// ④ 覆盖
banner('④ 从源目录完整覆盖')
const srcTop = fs.readdirSync(SRC)
let cpTop = 0, cpFiles = 0
for (const e of srcTop) {
  if (isProtected(e)) { console.log(`  · 跳过 ${e}（受保护）`); continue }
  const s = path.join(SRC, e), d = path.join(INSTALL, e)
  try {
    copyTree(s, d)
    const cnt = fs.statSync(s).isDirectory() ? walk(s).length : 1
    cpTop++; cpFiles += cnt
    console.log(`  ✓ 覆盖 ${e}  (${cnt} 个文件)`)
  } catch (err) {
    console.log(`  ✗ 覆盖 ${e} 失败: ${err.message}`)
  }
}

// ⑤ 校验
banner('⑤ 校验')
const afterFiles = walk(INSTALL)
const afterMap = new Map(afterFiles.filter((x) => !isProtected(x.rel)).map((x) => [x.rel, x.size]))
const missAfter = [...srcMap.keys()].filter((k) => !afterMap.has(k))
const extraAfter = [...afterMap.keys()].filter((k) => !srcMap.has(k))
const diffAfter = [...srcMap.keys()].filter((k) => {
  if (!afterMap.has(k)) return false
  try { return md5(path.join(SRC, k)) !== md5(path.join(INSTALL, k)) } catch { return true }
})

const srcExe = path.join(SRC, 'pair.exe'), dstExe = path.join(INSTALL, 'pair.exe')
let exeMd5 = '✗ 缺失'
try {
  const a = md5(srcExe), b = md5(dstExe)
  exeMd5 = a === b ? `✓ 一致 (${a.slice(0, 12)}…)` : `✗ 不一致（源 ${a.slice(0, 8)} / 装 ${b.slice(0, 8)}）`
} catch (e) { exeMd5 = '✗ 读取失败: ' + e.message }

console.log(`文件总数      : ${afterFiles.length}（受保护 ${afterFiles.filter((x) => isProtected(x.rel)).length}）`)
console.log(`pair.exe md5  : ${exeMd5}`)
console.log(`源文件缺失    : ${missAfter.length ? '✗ ' + missAfter.slice(0, 10).join(', ') : '✓ 无'}`)
console.log(`多余残留      : ${extraAfter.length ? '⚠ ' + extraAfter.slice(0, 10).join(', ') : '✓ 无'}`)
console.log(`内容不一致    : ${diffAfter.length ? '✗ ' + diffAfter.slice(0, 10).join(', ') : '✓ 无'}`)
console.log('关键文件:')
for (const r of REQUIRED_SRC) console.log(`  ${exists(path.join(INSTALL, r)) ? '✓' : '✗'} ${r}`)
console.log('受保护项（应保持原样）:')
for (const p of PROTECT) {
  const full = path.join(INSTALL, p)
  console.log(`  ${exists(full) ? '✓ 存在' : '✗ 缺失'} ${p}  (${exists(full) ? walk(full).length : 0} 个文件)`)
}
const cfg = path.join(INSTALL, 'config', 'settings.json')
console.log(`  ${exists(cfg) ? '✓' : '✗'} config/settings.json${exists(cfg) ? `  ${md5(cfg).slice(0, 12)}…（未被覆盖）` : ''}`)

const ok = !missAfter.length && !diffAfter.length && /^✓/.test(exeMd5)

// ⑥ 重启
let restarted = null
if (RESTART) {
  banner('⑥ 重新启动')
  const port = PORT_ARG || listenPorts[0] || 9090
  try {
    const child = spawn(path.join(INSTALL, 'pair.exe'), [], {
      cwd: INSTALL,
      detached: true,
      stdio: 'ignore',
      windowsHide: true,
      env: { ...process.env, WEB_PORT: String(port) },
    })
    child.unref()
    restarted = { pid: child.pid, port }
    console.log(`✓ 已启动 pair.exe（PID ${child.pid}，端口 ${port}，cwd=${INSTALL}）`)
    // 等端口起来
    const t = Date.now()
    while (Date.now() - t < 15000 && !isListening(port)) await sleep(500)
    console.log(isListening(port) ? `✓ 端口 ${port} 已在监听` : `⚠ 15s 内未监听到端口 ${port}（可能启动较慢或启动失败，查 logs/）`)
  } catch (e) {
    console.log(`✗ 启动失败: ${e.message}`)
  }
}

banner(ok ? '完成' : '完成（校验有问题，见上）')
console.log(`耗时          : ${((Date.now() - t0) / 1000).toFixed(1)}s`)
console.log(`删除 ${delTop} 项/${delFiles} 文件，覆盖 ${cpTop} 项/${cpFiles} 文件`)
if (bakDir) console.log(`备份          : ${path.relative(REPO, bakDir)}`)
if (restarted) console.log(`已重启        : PID ${restarted.pid} / 端口 ${restarted.port} → http://127.0.0.1:${restarted.port}`)
else console.log('未重启        : 手动运行 ' + path.join(INSTALL, 'pair.exe'))
process.exit(ok ? 0 : 2)
