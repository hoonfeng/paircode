// ═══════════════════════════════════════════════════════════════
// lib.mjs — source-update 共用工具：
//   版本读取（cmd/companion/main.go / packager.json）、semver 比较、
//   git 封装、日志、默认参数与部署同步排除清单。
//
// 目录约定：scripts/source-update/ 下三个入口：
//   check.mjs  —— 检查上游更新（可独立运行；插件消费 --json）
//   update.mjs —— 全自动更新编排（合并 -> 重建 -> 清单 -> 成对部署）
//   run-update.cmd —— 分离启动器（供插件“立即更新”使用）
// ═══════════════════════════════════════════════════════════════
import { spawnSync } from 'node:child_process';
import crypto from 'node:crypto';
import fs from 'node:fs';
import https from 'node:https';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

export const SCRIPT_DIR = path.dirname(fileURLToPath(import.meta.url));
export const REPO_ROOT = path.resolve(SCRIPT_DIR, '..', '..');

export const DEFAULTS = {
  repo: 'hoonfeng/paircode', // 上游 GitHub 仓库（官方）
  remote: 'github',          // 本地 git remote 名（指向 hoonfeng/paircode）
  repoPath: REPO_ROOT,       // 源码仓库本地路径
  installDir: 'D:\\PairCode', // 安装目录（成对部署目标）
  baseUrl: 'http://127.0.0.1:9090',
};

// 部署同步的永久排除清单（相对仓库根的 .pair 路径，正斜杠）：
//   - wechat-bridge/package.json —— 安装侧含本机独有改动（SUSPECT），覆盖会丢
//   - app-update / autopilot —— 引入于 v1.6.5 及更早、有意未同步到本机的历史项
//   - ui-statusbar-conn/client.js.disabled、assets/runtime/README.md —— 同上
// ★ 注意：source-update 插件本身不在此列 —— 它就是设计为要同步到本机的。
export const SYNC_EXCLUDES = new Set([
  '.pair/plugins/wechat-bridge/package.json',
  '.pair/plugins/app-update/index.js',
  '.pair/plugins/app-update/package.json',
  '.pair/plugins/autopilot/client.js',
  '.pair/plugins/autopilot/index.js',
  '.pair/plugins/autopilot/package.json',
  '.pair/plugins/ui-statusbar-conn/client.js.disabled',
  '.pair/assets/runtime/README.md',
]);

// 差异扫描时跳过的目录/文件（运行时数据与产物，不参与比对、绝不覆盖）
export const SKIP_DIR = /(?:^|\/)(node_modules|\.git|bin|logs|data|state|\.cache|tmp|snapshots)(?:\/|$)/;
export const SKIP_FILE = /(\.bak(-.*)?|\.tmp|\.log|\.old|~)$/i;

// ─── 版本 ────────────────────────────────────────────────────

// 本地版本：优先 main.go 的 `var version = "vX.Y.Z"`（权威，带 v 前缀），
// 兜底 packager.json 的 version 字段。
export function readLocalVersion(repoPath = REPO_ROOT) {
  const mainGo = path.join(repoPath, 'cmd', 'companion', 'main.go');
  if (fs.existsSync(mainGo)) {
    const m = fs.readFileSync(mainGo, 'utf8').match(/^var version = "([^"]+)"/m);
    if (m) return { version: m[1], source: 'cmd/companion/main.go' };
  }
  const pj = path.join(repoPath, 'packager.json');
  if (fs.existsSync(pj)) {
    try {
      const v = JSON.parse(fs.readFileSync(pj, 'utf8')).version;
      if (v) return { version: 'v' + v, source: 'packager.json' };
    } catch { /* ignore */ }
  }
  return { version: '', source: '' };
}

// 解析 semver（容忍 v 前缀与 -pre 后缀）；失败返回 null。
export function parseVer(s) {
  const m = String(s || '').trim().replace(/^v/i, '').match(/^(\d+)\.(\d+)\.(\d+)(?:-([0-9A-Za-z.-]+))?/);
  if (!m) return null;
  return { major: +m[1], minor: +m[2], patch: +m[3], pre: m[4] || '' };
}

// a > b → 1，a < b → -1，相等 → 0；无法解析抛错。
export function cmpVer(a, b) {
  const pa = parseVer(a), pb = parseVer(b);
  if (!pa || !pb) throw new Error('无法解析版本号: ' + a + ' / ' + b);
  for (const k of ['major', 'minor', 'patch']) {
    if (pa[k] !== pb[k]) return pa[k] > pb[k] ? 1 : -1;
  }
  if (!pa.pre && pb.pre) return 1;   // 正式版 > 预发布
  if (pa.pre && !pb.pre) return -1;
  if (pa.pre === pb.pre) return 0;
  return pa.pre > pb.pre ? 1 : -1;
}

// ─── git ─────────────────────────────────────────────────────

export function git(repoPath, args, opts = {}) {
  const r = spawnSync('git', ['-C', repoPath, ...args], {
    encoding: 'utf8',
    windowsHide: true,
    maxBuffer: 64 * 1024 * 1024,
    ...opts,
  });
  return {
    code: r.status === null ? -1 : r.status,
    out: String(r.stdout || '').trim(),
    err: String(r.stderr || '').trim(),
  };
}

// ─── 日志 ─────────────────────────────────────────────────────

export function ts() {
  const d = new Date();
  const p = (n) => String(n).padStart(2, '0');
  return d.getFullYear() + '-' + p(d.getMonth() + 1) + '-' + p(d.getDate()) + ' ' +
    p(d.getHours()) + ':' + p(d.getMinutes()) + ':' + p(d.getSeconds());
}

export function makeLogger(logFile) {
  if (logFile) fs.mkdirSync(path.dirname(logFile), { recursive: true });
  return (msg) => {
    const line = ts() + '  ' + msg;
    console.log(line);
    if (logFile) { try { fs.appendFileSync(logFile, line + '\n', 'utf8'); } catch { /* ignore */ } }
  };
}

// ─── 小工具 ───────────────────────────────────────────────────

export function parseArgs(argv) {
  const out = {};
  for (let i = 0; i < argv.length; i++) {
    const a = argv[i];
    if (a.startsWith('--')) {
      const k = a.slice(2);
      const next = argv[i + 1];
      if (next !== undefined && !next.startsWith('--')) { out[k] = next; i++; }
      else out[k] = true;
    }
  }
  return out;
}

export function sha256File(p) {
  return crypto.createHash('sha256').update(fs.readFileSync(p)).digest('hex').toUpperCase();
}

// ─── GitHub API ───────────────────────────────────────────────
//
// ★ 不用全局 fetch(undici)：其 keep-alive socket 会挂住 event loop，
//   在 Windows + Node 24 下紧接着 process.exit() 会触发 libuv 断言
//   "Assertion failed: !(handle->flags & UV_HANDLE_CLOSING)" 并以 127 退出。
//   原生 https + agent:false = 每次新建连接、响应后立即关闭，干净退出。
export function fetchGithubTags(repo, token) {
  return new Promise((resolve, reject) => {
    const headers = {
      'User-Agent': 'PairCode-SourceUpdate/1.0',
      'Accept': 'application/vnd.github+json',
    };
    if (token) headers.Authorization = 'Bearer ' + token;
    const req = https.get(
      {
        hostname: 'api.github.com',
        path: '/repos/' + repo + '/tags?per_page=100',
        headers,
        agent: false,
        timeout: 20000,
      },
      (res) => {
        let data = '';
        res.setEncoding('utf8');
        res.on('data', (c) => { data += c; });
        res.on('end', () => {
          if (res.statusCode !== 200) {
            reject(new Error('GitHub API HTTP ' + res.statusCode + (data ? '：' + data.slice(0, 200) : '')));
            return;
          }
          try { resolve(JSON.parse(data)); }
          catch (e) { reject(new Error('JSON 解析失败: ' + e.message)); }
        });
      }
    );
    req.on('timeout', () => req.destroy(new Error('请求超时（20s）')));
    req.on('error', reject);
  });
}

// 从 tag 列表中选最新 stable（semver 最大、跳过预发布）；无则返回 null。
export function pickLatestStable(tags) {
  let best = null;
  for (const t of tags) {
    const name = t && t.name;
    if (!name) continue;
    const v = parseVer(name);
    if (!v || v.pre) continue;
    if (!best || cmpVer(name, best.name) > 0) best = t;
  }
  return best;
}
