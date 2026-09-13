#!/usr/bin/env node
// upstream-pr.mjs —— PairCode 上游 PR 自动化脚本
//
// 用法：
//   node scripts/github-sync/upstream-pr.mjs --message "fix(xxx): 描述"   # 提交改动 + 推送 fork + 查/建上游 PR
//   node scripts/github-sync/upstream-pr.mjs                              # 无改动时：推送检查 + 查上游 PR 现状
//   node scripts/github-sync/upstream-pr.mjs --dry-run                    # 只预览将做的事，不做任何写操作
//   node scripts/github-sync/upstream-pr.mjs --no-push                    # 只提交本地，不推送不查 PR
//   可选：--title "PR 标题" --body "PR 正文"（仅创建新 PR 时生效）
//
// 功能：检查/提交本地改动 → 推送 fork（origin，SSH over 443）→ 查找/创建上游 PR（hoonfeng/paircode）
// 依赖：node >= 18；git 可执行；网络经 Watt Toolkit 时自动加载其 CA（请求级信任，绝不关闭证书校验）
//
// 安全说明（逐条对应合规要求）：
//   - 所有 git 调用均使用参数数组（execFileSync('git', [args...])），不拼接 shell 字符串，无命令注入面；
//     --message/--title/--body 的值仅作为数组元素传递（-m 的值、JSON body 字段），且 --message 以 '-' 开头会被拒绝
//   - 推送仅执行 `git push origin master`，不含任何 --force / --delete / 镜像等破坏性参数
//   - --dry-run 模式不执行任何写操作：commit / push / POST 创建全部跳过；
//     仅保留只读操作（本地 git status/log/rev-parse 与 GitHub GET 查询）
//   - 全部文件为 UTF-8；token 只从本地文件或环境变量读取，任何情况下不打印 token 值
//   - 错误处理：git 失败、网络失败、缺 token、origin 配置错误均给出明确提示并以非零码退出
import fs from 'node:fs';
import path from 'node:path';
import https from 'node:https';
import tls from 'node:tls';
import { execFileSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const REPO_ROOT = path.resolve(__dirname, '..', '..');
const UPSTREAM = 'hoonfeng/paircode'; // PR 目标仓库（上游）
const FORK_OWNER = 'xh290';           // 源仓库（fork）owner
const FORK_REPO = 'xh290/paircode';   // 源仓库（fork）全名
const BRANCH = 'master';

// ── 参数解析（argv 数组，无 shell 参与）──
const argv = process.argv.slice(2);
if (argv.includes('--help') || argv.includes('-h')) {
  console.log(fs.readFileSync(fileURLToPath(import.meta.url), 'utf8').split('\n').slice(1, 12).map((l) => l.replace(/^\/\/ ?/, '')).join('\n'));
  process.exit(0);
}
const getOpt = (name, def = null) => {
  const i = argv.indexOf('--' + name);
  if (i < 0) return def;
  const v = argv[i + 1];
  return v && !v.startsWith('--') ? v : true;
};
const DRY = argv.includes('--dry-run');
const NO_PUSH = argv.includes('--no-push');
const MESSAGE = getOpt('message');
const TITLE = getOpt('title');
const BODY = getOpt('body');

const log = (...a) => console.log(...a);
const die = (msg) => { console.error('\n[x] ' + msg); process.exit(1); };

// ── git 封装（一律数组传参，等价于 execFile，无 shell 拼接）──
function git(...args) {
  return execFileSync('git', args, { cwd: REPO_ROOT, encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'] }).trim();
}
function gitOk(...args) {
  try { return { ok: true, out: git(...args) }; } catch (e) { return { ok: false, err: (e.stderr || e.message || '').toString().trim() }; }
}

// ── CA 处理（请求级信任 Watt 根证书；绝不禁用证书校验，禁止 rejectUnauthorized:false）──
function loadCA() {
  const list = [...tls.rootCertificates];
  const candidates = [
    process.env.PAIRCODE_WATT_CA,
    path.join(REPO_ROOT, 'temp', 'hf', 'watt-ca.pem'),
    process.env.LOCALAPPDATA ? path.join(process.env.LOCALAPPDATA, 'Steam++', 'Plugins', 'Accelerator', 'SteamTools.Certificate.cer') : null,
  ].filter(Boolean);
  for (const c of candidates) {
    try {
      if (fs.existsSync(c)) { list.push(fs.readFileSync(c, 'utf8')); log(`  - 已加载自定义 CA（Watt Toolkit）: ${c}`); break; }
    } catch { /* ignore */ }
  }
  return list;
}
const CA = loadCA();

// ── GitHub API 封装（只走 api.github.com；请求级 ca 列表；30s 超时）──
function api(method, apiPath, { token, body } = {}) {
  return new Promise((resolve, reject) => {
    const payload = body ? JSON.stringify(body) : null;
    const req = https.request({
      hostname: 'api.github.com', path: apiPath, method, ca: CA, timeout: 30000,
      headers: {
        'User-Agent': 'paircode-upstream-pr/1.0',
        Accept: 'application/vnd.github+json',
        ...(token ? { Authorization: 'Bearer ' + token } : {}),
        ...(payload ? { 'Content-Type': 'application/json', 'Content-Length': Buffer.byteLength(payload) } : {}),
      },
    }, (res) => {
      let data = '';
      res.on('data', (d) => (data += d));
      res.on('end', () => {
        let json = null;
        try { json = data ? JSON.parse(data) : null; } catch { /* 非 JSON 时保留 raw 原文 */ }
        resolve({ status: res.statusCode, json, raw: data });
      });
    });
    req.on('timeout', () => req.destroy(new Error('API 请求超时（30s）')));
    req.on('error', reject);
    if (payload) req.write(payload);
    req.end();
  });
}

// ── token 读取（仅创建 PR 需要；查询与推送都不需要）──
function readToken() {
  if (process.env.GH_TOKEN) return process.env.GH_TOKEN.trim();
  for (const p of [
    path.join(REPO_ROOT, '.pair', 'secrets', 'github-token.txt'),
    path.join(REPO_ROOT, 'temp', 'hf', 'classic-token.txt'),
  ]) {
    try { if (fs.existsSync(p)) { const t = fs.readFileSync(p, 'utf8').trim(); if (t) return t; } } catch { /* ignore */ }
  }
  return null;
}

// ── 主流程 ──
async function main() {
  log('== PairCode 上游 PR 自动化 (upstream-pr.mjs) ==');
  log(`仓库: ${REPO_ROOT}`);
  if (DRY) log('模式: dry-run（只预览，不做写操作）');

  // 0) 环境校验：origin 必须是 fork
  const originUrl = gitOk('remote', 'get-url', 'origin');
  if (!originUrl.ok || !String(originUrl.out).includes(FORK_REPO)) {
    die('origin 远端不是 fork（' + FORK_REPO + '）：' + (originUrl.err || originUrl.out || '未配置')
      + '\n请先执行: git remote set-url origin git@github.com:' + FORK_REPO + '.git');
  }
  log(`  - origin: ${originUrl.out}`);

  // 1) 状态检查（+ 可选提交）
  log('\n[1/5] 检查本地状态...');
  const branch = git('rev-parse', '--abbrev-ref', 'HEAD');
  if (branch !== BRANCH) die(`当前分支为 ${branch}，脚本仅支持 ${BRANCH}`);

  const dirty = git('status', '--porcelain');
  const dirtyFiles = dirty ? dirty.split('\n').filter(Boolean) : [];
  let willCommit = false; // dry-run 下「将提交」标记（用于推送预览判断）
  if (dirtyFiles.length) {
    log(`  - 未提交改动 ${dirtyFiles.length} 处：`);
    dirtyFiles.slice(0, 20).forEach((l) => log('      ' + l));
    if (dirtyFiles.length > 20) log(`      ...（共 ${dirtyFiles.length} 处）`);
    if (MESSAGE === true) die('--message 缺值：请提供提交消息字符串');
    if (!MESSAGE) die('存在未提交改动。请加 --message "提交消息"（或先手动 git commit）');
    if (String(MESSAGE).startsWith('-')) die('--message 不能以 - 开头（避免被误解析为 git 选项）');
    log(`  - 提交消息: ${MESSAGE}`);
    if (DRY) { willCommit = true; log('  - [dry-run] 将执行: git add -A && git commit -m <消息>'); }
    else {
      try { git('add', '-A'); git('commit', '-m', String(MESSAGE)); log('  - 已提交 [ok]'); }
      catch (e) { die('提交失败：' + (e.stderr || e.message)); }
    }
  } else {
    log('  - 工作区干净');
  }

  const head = git('rev-parse', 'HEAD');
  const originRef = gitOk('rev-parse', 'origin/master');
  const unpushed = originRef.ok ? git('log', '--oneline', 'origin/master..HEAD').split('\n').filter(Boolean) : [];
  log(`  - 本地 HEAD: ${head.slice(0, 8)}${originRef.ok ? (unpushed.length ? `（领先 origin/master ${unpushed.length} 个提交）` : (willCommit ? '（dry-run：提交后将新增 1 个提交）' : '（与 origin/master 一致）')) : '（origin/master 引用不可用）'}`);
  unpushed.forEach((l) => log('      ' + l));

  // 2) 推送 fork（仅 `git push origin master`，绝无 --force）
  log('\n[2/5] 推送 fork（origin -> ' + FORK_REPO + '）...');
  if (NO_PUSH) {
    log('  - --no-push：跳过推送与 PR 查询');
    log('\n== 完成（仅本地提交）==');
    return;
  }
  if (!originRef.ok || unpushed.length || willCommit) {
    if (DRY) log('  - [dry-run] 将执行: git push origin master');
    else {
      try { git('push', 'origin', BRANCH); log('  - 推送完成 [ok]'); }
      catch (e) { die('推送失败：' + (e.stderr || e.message)); }
    }
  } else {
    log('  - 无需推送（无新提交）');
  }

  // 3) 核对远端
  log('\n[3/5] 核对远端 master...');
  if (DRY) log('  - [dry-run] 跳过');
  else {
    const r = gitOk('ls-remote', 'origin', `refs/heads/${BRANCH}`);
    const remoteSha = r.ok ? r.out.split(/\s+/)[0] : null;
    if (!remoteSha) log('  - 警告：ls-remote 失败，无法核对（继续）');
    else if (remoteSha === head) log(`  - 远端 master = 本地 HEAD = ${head.slice(0, 8)} [ok]`);
    else log(`  - 警告：远端 ${remoteSha.slice(0, 8)} != 本地 ${head.slice(0, 8)}（若上游有变化请先 git fetch 处理）`);
  }

  // 4) 查上游 open PR（只读 GET，不需要 token）
  log('\n[4/5] 查询上游 open PR...');
  const q = await api('GET', `/repos/${UPSTREAM}/pulls?head=${FORK_OWNER}:${BRANCH}&state=open`);
  if (q.status !== 200) die(`查询 PR 失败: HTTP ${q.status} ${q.raw.slice(0, 200)}`);
  const openPr = Array.isArray(q.json) ? q.json[0] : null;
  if (openPr) {
    log(`  - 已有 open PR #${openPr.number}: ${openPr.title}`);
    log(`    ${openPr.html_url}`);
    log('\n== 完成：已有 PR（新推送的提交已自动包含其中）==');
    return;
  }

  // 5) 创建 PR（dry-run 在此前已返回；POST 前再确认一次 DRY）
  log('\n[5/5] 无 open PR，准备创建...');
  const br = await api('GET', `/repos/${UPSTREAM}/branches/${BRANCH}`);
  const baseSha = br.status === 200 ? br.json.commit.sha : null;
  let commits = [];
  let shortstat = '';
  if (baseSha && gitOk('cat-file', '-e', baseSha).ok) {
    commits = git('log', '--pretty=format:%h %s', `${baseSha}..HEAD`).split('\n').filter(Boolean);
    shortstat = (gitOk('diff', '--shortstat', `${baseSha}..HEAD`).out || '').trim();
  } else if (baseSha) {
    log(`  - 提示：本地缺上游基准对象 ${baseSha.slice(0, 8)}，标题/正文将简化`);
    commits = git('log', '--pretty=format:%h %s', '-10', 'HEAD').split('\n').filter(Boolean);
  }
  const stripHash = (s) => s.replace(/^\S+\s+/, '');
  if (DRY && willCommit && !commits.length) log('  - [dry-run] 提示：实际运行时提交已发生，将列出新提交并据其生成标题/正文');
  const title = (TITLE !== true && TITLE) || (commits.length === 1 ? stripHash(commits[0]) : `同步 ${commits.length || ''} 个提交：${commits[0] ? stripHash(commits[0]) : '更新'}`);
  const body = (BODY !== true && BODY) || [
    '## 变更概览',
    '',
    `从 fork 同步本地开发进展${commits.length ? `，共 ${commits.length} 个提交` : ''}${shortstat ? `（${shortstat}）` : ''}：`,
    '',
    '### 提交列表',
    ...(commits.length ? commits.map((c) => '- ' + c) : ['- （见 PR commits 页）']),
    '',
    '---',
    `Fork 同步：https://github.com/${FORK_REPO}`,
    '（本 PR 由 scripts/github-sync/upstream-pr.mjs 自动创建）',
  ].join('\n');

  log(`  - 标题: ${title}`);
  log(`  - 正文预览:\n-----`);
  log(body.slice(0, 500) + (body.length > 500 ? '\n...' : ''));
  log('-----');

  if (DRY) { log('\n[dry-run] 将创建 PR（未执行）'); return; }

  const token = readToken();
  if (!token) {
    die([
      '未找到 API token（创建 PR 需要；查询不需要）。',
      '注意：本地提交与推送 fork 已完成（或无需推送），仅差「创建 PR」这一步。',
      '请创建 classic token（仅勾选 public_repo 即可）并保存到：',
      `  ${path.join(REPO_ROOT, '.pair', 'secrets', 'github-token.txt')}`,
      '（或临时放 temp/hf/classic-token.txt；或设环境变量 GH_TOKEN）',
      '详见 .pair/secrets/README.md',
    ].join('\n'));
  }

  const cr = await api('POST', `/repos/${UPSTREAM}/pulls`, {
    token,
    body: { title, head: `${FORK_OWNER}:${BRANCH}`, base: BRANCH, body },
  });
  if (cr.status === 201) {
    log(`\n== [ok] PR 创建成功 #${cr.json.number}: ${cr.json.html_url} ==`);
    return;
  }
  if (cr.status === 422) {
    // 可能已存在（竞态）：复核一次
    const q2 = await api('GET', `/repos/${UPSTREAM}/pulls?head=${FORK_OWNER}:${BRANCH}&state=open`);
    if (Array.isArray(q2.json) && q2.json[0]) {
      log(`\n== [ok] PR 已存在 #${q2.json[0].number}: ${q2.json[0].html_url} ==`);
      return;
    }
  }
  die(`创建 PR 失败: HTTP ${cr.status}\n${cr.raw.slice(0, 500)}`);
}

main().catch((e) => die(e && e.stack || String(e)));
