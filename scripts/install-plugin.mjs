#!/usr/bin/env node
// install-plugin.mjs — 命令行安装 npm 上的 PairCode 插件（市场安装超时的替代路径）
//
// 背景：市场「安装」走内核 npm 拉取（internal/agent/npm_plugin.go，默认 registry.npmjs.org）。
//   实测本机 registry.npmjs.org 的 tarball 下载仅 ~3 KB/s（5.78 MB 的包 90s 只下了 274 KB），
//   前端/接口 30s 超时必然失败；registry.npmmirror.com 同包 1.6s 下完。
//   ⇒ 命令行安装（可指定镜像）是可靠落盘路径。
//
// 用法：
//   node scripts/install-plugin.mjs --pkg paircode-plugin-wechat-bridge
//   node scripts/install-plugin.mjs --pkg @paircode/tool-voice --registry https://registry.npmjs.org
//   node scripts/install-plugin.mjs --pkg xxx --install-dir "E:/Program Files (x86)/PairCode"
//   node scripts/install-plugin.mjs --pkg xxx --dry-run        # 只查版本与体积，不写盘
//
// 落盘口径与内核一致（internal/agent/toolset.go syncGlobalPluginPackage）：
//   <install-dir>/.pair/plugins/<name>/ ← 白名单整包（index.js / client.js / assets / lib / bin / README.md）
//   + package.json（保留原始字段，写 config.npm = {pkg, version} 供市场「更新」识别）
//   重启 PairCode 后由 LoadGlobalPlugins 自动装配（goja 轨）。
//
// 零第三方依赖（node 内置 fetch + zlib；tar 解析内置实现）。

import fs from 'node:fs';
import path from 'node:path';
import zlib from 'node:zlib';
import { createHash } from 'node:crypto';

// ─── 参数 ───────────────────────────────────────────────────────

function parseArgs(argv) {
  const out = { registry: 'https://registry.npmmirror.com', dryRun: false };
  for (let i = 0; i < argv.length; i++) {
    const a = argv[i];
    const next = () => {
      const v = argv[++i];
      if (v === undefined) fail(`参数 ${a} 缺值`);
      return v;
    };
    if (a === '--pkg' || a === '-p') out.pkg = next();
    else if (a === '--install-dir' || a === '-d') out.installDir = next();
    else if (a === '--registry' || a === '-r') out.registry = next().replace(/\/+$/, '');
    else if (a === '--version' || a === '-v') out.version = next();
    else if (a === '--dry-run') out.dryRun = true;
    else if (a === '--help' || a === '-h') out.help = true;
    else fail(`未知参数: ${a}（--help 查看用法）`);
  }
  return out;
}

function fail(msg) {
  console.error('✗ ' + msg);
  process.exit(1);
}

const USAGE = `
用法: node scripts/install-plugin.mjs --pkg <npm 包名> [选项]

选项:
  -p, --pkg <name>          npm 包名（必填，如 paircode-plugin-wechat-bridge / @paircode/tool-voice）
  -d, --install-dir <dir>   PairCode 安装目录（默认自动探测含 pair.exe 的常见安装路径）
  -r, --registry <url>      包源（默认 https://registry.npmmirror.com；国内 npmjs.org 极慢）
  -v, --version <ver>       指定版本（默认 latest）
      --dry-run             只查询版本/体积，不写盘
  -h, --help                显示本帮助
`;

// ─── 安装目录探测 ───────────────────────────────────────────────

function detectInstallDir() {
  const cands = [
    'E:/Program Files (x86)/PairCode',
    'E:/Program Files/PairCode',
    'C:/Program Files (x86)/PairCode',
    'C:/Program Files/PairCode',
    'D:/Program Files (x86)/PairCode',
    'D:/Program Files/PairCode',
  ];
  for (const c of cands) {
    try {
      if (fs.existsSync(path.join(c, 'pair.exe'))) return c;
    } catch { /* ignore */ }
  }
  return '';
}

// ─── tar 解析（USTAR + pax 路径覆盖）────────────────────────────

function cstr(buf) {
  const end = buf.indexOf(0);
  return buf.subarray(0, end === -1 ? buf.length : end).toString('utf8');
}

function parseTar(buf) {
  const out = [];
  let off = 0;
  let paxPath = null;
  while (off + 512 <= buf.length) {
    const hdr = buf.subarray(off, off + 512);
    let allZero = true;
    for (let i = 0; i < 512; i++) if (hdr[i] !== 0) { allZero = false; break; }
    if (allZero) break;
    const name = cstr(hdr.subarray(0, 100));
    const size = parseInt(cstr(hdr.subarray(124, 136)).trim() || '0', 8) || 0;
    const typeflag = String.fromCharCode(hdr[156] || 48);
    const prefix = cstr(hdr.subarray(345, 500));
    off += 512;
    const data = buf.subarray(off, off + size);
    off += Math.ceil(size / 512) * 512;
    if (typeflag === 'x' || typeflag === 'g') {
      const m = data.toString('utf8').match(/\d+ path=([^\n]+)\n/);
      if (m) paxPath = m[1];
      continue;
    }
    let full = prefix ? prefix + '/' + name : name;
    if (paxPath) { full = paxPath; paxPath = null; }
    out.push({ name: full, data, isDir: typeflag === '5' });
  }
  return out;
}

// ─── 主流程 ─────────────────────────────────────────────────────

const argv = parseArgs(process.argv.slice(2));
if (argv.help) { console.log(USAGE.trim()); process.exit(0); }
if (!argv.pkg) { console.log(USAGE.trim()); fail('缺 --pkg'); }

const registry = argv.registry;
const pkg = argv.pkg;

// 1) 查版本
const metaURL = argv.version
  ? `${registry}/${pkg.replace('/', '%2F')}/${argv.version}`
  : `${registry}/${pkg.replace('/', '%2F')}/latest`;
console.log(`→ 查询 ${metaURL}`);
let meta;
try {
  const r = await fetch(metaURL, { signal: AbortSignal.timeout(30_000) });
  if (!r.ok) fail(`查询失败：HTTP ${r.status}（包名或 registry 是否正确？）`);
  meta = await r.json();
} catch (e) {
  fail(`查询失败：${e.message}`);
}
if (!meta.dist?.tarball) fail(`registry 未返回 tarball（包 ${pkg} 不存在？）`);
console.log(`  包: ${meta.name}@${meta.version}`);
console.log(`  说明: ${meta.description || '（无）'}`);
console.log(`  体积: ${((meta.dist.unpackedSize || 0) / 1048576).toFixed(2)} MB（解包）`);
if (meta.main) console.log(`  main: ${meta.main}${meta.client ? `  client: ${meta.client}` : ''}`);

// 运行期依赖 ⇒ Node 桥轨插件（需要 node_modules，本脚本只落包体，会给出提示）
const runtimeDeps = Object.keys(meta.dependencies || {});
if (runtimeDeps.length) {
  console.log(`  ⚠ 声明运行期依赖（${runtimeDeps.join(', ')}）⇒ Node 桥轨插件：`);
  console.log('    本脚本按内核市场安装口径落盘整包，桥轨装载还需要 .pair/cordis/node 侧 npm install（重启时自动尝试）。');
}

if (argv.dryRun) {
  console.log('（--dry-run：未写盘）');
  process.exit(0);
}

// 2) 目标目录
const installDir = argv.installDir || detectInstallDir();
if (!installDir) fail('未探测到安装目录（含 pair.exe）——请用 --install-dir 指定');
if (!fs.existsSync(path.join(installDir, 'pair.exe'))) {
  console.log(`  ⚠ ${installDir} 下未见 pair.exe（仍继续，按 --install-dir 处理）`);
}
const diskName = pkg.startsWith('@') && pkg.includes('/') ? pkg.slice(pkg.indexOf('/') + 1) : pkg;
const dst = path.join(installDir, '.pair', 'plugins', diskName);
try {
  fs.mkdirSync(dst, { recursive: true });
} catch (e) {
  fail(`创建 ${dst} 失败：${e.message}（Program Files 需要管理员权限的终端）`);
}
console.log(`→ 安装目录: ${installDir}`);
console.log(`→ 插件目录: ${dst}`);

// 3) 下载 tarball
const t0 = Date.now();
console.log(`→ 下载 ${meta.dist.tarball}`);
let tgz;
try {
  const r = await fetch(meta.dist.tarball, { signal: AbortSignal.timeout(600_000) });
  if (!r.ok) fail(`下载失败：HTTP ${r.status}`);
  tgz = Buffer.from(await r.arrayBuffer());
} catch (e) {
  fail(`下载失败：${e.message}`);
}
const secs = ((Date.now() - t0) / 1000).toFixed(1);
console.log(`  完成: ${(tgz.length / 1048576).toFixed(2)} MB / ${secs}s（${(tgz.length / 1024 / Math.max(parseFloat(secs), 0.001)).toFixed(0)} KB/s）`);

// 4) 解包
let entries;
try {
  entries = parseTar(zlib.gunzipSync(tgz));
} catch (e) {
  fail(`解压失败：${e.message}（tarball 损坏？）`);
}
const PREFIX = 'package/';
const pkgEntries = entries
  .filter((e) => e.name === 'package' || e.name.startsWith(PREFIX))
  .map((e) => ({ ...e, rel: e.name === 'package' ? '' : e.name.slice(PREFIX.length) }))
  .filter((e) => e.rel !== '' || e.isDir);
if (!pkgEntries.length) fail('tarball 内无 package/ 内容');

const manifestEntry = pkgEntries.find((e) => e.rel === 'package.json');
let manifest = {};
if (manifestEntry) {
  try { manifest = JSON.parse(manifestEntry.data.toString('utf8')); } catch { /* ignore */ }
}

// 5) 白名单整包复制（与 syncGlobalPluginPackage 同口径）
const WHITELIST = ['index.js', 'client.js', 'assets', 'lib', 'bin', 'README.md'];
const copied = [];
const inWhitelist = (rel) => WHITELIST.some((w) => rel === w || rel.startsWith(w + '/'));
for (const w of WHITELIST) {
  fs.rmSync(path.join(dst, w), { recursive: true, force: true });
}
for (const e of pkgEntries) {
  if (!inWhitelist(e.rel)) continue;
  const target = path.join(dst, e.rel);
  if (!path.resolve(target).startsWith(path.resolve(dst))) fail(`tarball 含非法路径: ${e.name}`);
  if (e.isDir) { fs.mkdirSync(target, { recursive: true }); continue; }
  fs.mkdirSync(path.dirname(target), { recursive: true });
  fs.writeFileSync(target, e.data);
  copied.push(e.rel);
}
if (!copied.length) fail('包内无白名单内容（index.js/client.js/assets/lib/bin/README.md）');

// 6) package.json（保留原始字段 + 装载字段 + config.npm 元数据）
const out = { ...manifest };
out.name = diskName;
out.purpose = manifest.description || '';
if (!out.main) out.main = 'index.js';
if (!out.version) out.version = meta.version;
if (!out.type) out.type = 'plugin';
if (!out.scope) out.scope = 'global';
out.config = { ...(out.config || {}), npm: { pkg, version: meta.version } };
fs.writeFileSync(path.join(dst, 'package.json'), JSON.stringify(out, null, 2));

// 7) 结果
console.log('→ 已落盘:');
const walk = (dir, prefix = '') => {
  for (const e of fs.readdirSync(dir, { withFileTypes: true }).sort((a, b) => a.name.localeCompare(b.name))) {
    const p = path.join(dir, e.name);
    if (e.isDirectory()) walk(p, prefix + e.name + '/');
    else {
      const b = fs.readFileSync(p);
      console.log(`   ${prefix}${e.name}  ${b.length}B  md5=${createHash('md5').update(b).digest('hex')}`);
    }
  }
};
walk(dst);
console.log(`✓ 已安装「${pkg}」v${meta.version} → .pair/plugins/${diskName}/`);
console.log('  重启 PairCode 后由 LoadGlobalPlugins 自动装配（本机 9090 实例请自行重启，勿由脚本代劳）。');
