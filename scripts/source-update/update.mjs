// update.mjs — PairCode 源码仓库全自动更新：预检 -> 快照 -> 合并上游 tag -> 重建 -> 成对部署。
// 用法：node scripts/source-update/update.mjs [--dry-run|--skip-deploy|--to <tag>|--force]
//       （建议经 run-update.cmd 分离运行——部署阶段会停 pair.exe 宿主）
// 退出码：0 成功/无需更新 | 1 预检 | 2 合并冲突 | 3 构建 | 4 清单 | 5 部署 | 6 已在运行
//
// 安全设计（对应审核关注点）：
//   · tag 白名单 assertSafeRef（拒 - 开头与特殊字符，防 git 选项解析/注入）；
//   · 路径参数 safeDir（拒 ..、要求绝对路径），且先校验 .git / pair.exe 存在；
//   · 子进程全部 spawnSync 数组传参（构建细节见 build.mjs，唯一 shell:true 系固定常量）；
//   · 本脚本不删除/不直接覆盖任何目标文件（唯一写入 = temp/source-update/ 运行目录）；
//   · 合并前先打 pre-merge-* 备份 tag，冲突自动 merge --abort 还原；
//   · 目标文件替换全部委托 deploy-pair-full.ps1（停服前备份+逐文件 sha 校验+失败回滚）。
import fs from 'node:fs';
import path from 'node:path';
import { spawnSync } from 'node:child_process';
import { DEFAULTS, SYNC_EXCLUDES, parseArgs, readLocalVersion, git, makeLogger,
  fetchGithubTags, pickLatestStable } from './lib.mjs';
import { buildFrontend, goChanged, buildExe, makeManifest } from './build.mjs';

const args = parseArgs(process.argv.slice(2));

function safeDir(p, what) {
  if (typeof p !== 'string' || !p.trim()) throw new Error(what + ' 不能为空');
  if (p.includes('..')) throw new Error(what + ' 不允许包含 .. : ' + p);
  const r = path.resolve(p);
  if (!path.isAbsolute(r)) throw new Error(what + ' 必须是绝对路径: ' + p);
  return r;
}
function safeRef(n) {
  if (!/^[A-Za-z0-9][A-Za-z0-9._-]*$/.test(String(n))) throw new Error('非法 tag 名（拒绝执行）: ' + JSON.stringify(n));
  return n;
}
let repoPath, installDir;
try {
  repoPath = safeDir(args['repo-path'] || DEFAULTS.repoPath, '--repo-path');
  installDir = safeDir(args['install-dir'] || DEFAULTS.installDir, '--install-dir');
} catch (e) { console.error('[source-update/update] ' + e.message); process.exit(1); }
const remote = args.remote || DEFAULTS.remote;
const repoSlug = args.repo || DEFAULTS.repo;
const dryRun = !!args['dry-run'], skipDeploy = !!args['skip-deploy'], force = !!args.force;
const toTag = typeof args.to === 'string' ? args.to : '';

const base = path.join(repoPath, 'temp', 'source-update');
const lockFile = path.join(base, 'lock.json');
const runId = (() => {
  const d = new Date(), p = (n) => String(n).padStart(2, '0');
  return d.getFullYear() + p(d.getMonth() + 1) + p(d.getDate()) + '-' + p(d.getHours()) + p(d.getMinutes()) + p(d.getSeconds());
})();
const runDir = path.join(base, 'run-' + runId);
fs.mkdirSync(runDir, { recursive: true });
const log = makeLogger(path.join(runDir, 'update.log'));
let stage = 'init';
const state = { tag: '', from: '', backupTag: '', exeBuilt: false, deployed: false, itemCount: 0 };

function fail(code, note) { const e = new Error(note); e.exitCode = code; e.stage = stage; return e; }

// 防重入锁（lock.json + pid 存活检测；陈旧锁自动清理）
function acquireLock() {
  fs.mkdirSync(base, { recursive: true });
  if (fs.existsSync(lockFile)) {
    let old = {}; try { old = JSON.parse(fs.readFileSync(lockFile, 'utf8')); } catch { /* ignore */ }
    let alive = false;
    if (old.pid) { try { process.kill(old.pid, 0); alive = true; } catch (e) { alive = e.code === 'EPERM'; } }
    if (alive) { log('已有更新在运行（pid=' + old.pid + '），本次退出。'); process.exitCode = 6; return false; }
    log('陈旧锁（pid=' + old.pid + '）已清理。');
  }
  fs.writeFileSync(lockFile, JSON.stringify({
    pid: process.pid, startedAt: new Date().toISOString(), target: toTag || '(latest)',
  }, null, 2), 'utf8');
  return true;
}
function releaseLock() { try { fs.unlinkSync(lockFile); } catch { /* ignore */ } }

// 终局：写 RESULT.txt / last-result.json（插件启动时读取展示）、释放锁、定退出码
function finish(res) {
  const payload = Object.assign({ runDir, finishedAt: new Date().toISOString(), pid: process.pid }, state, res);
  try {
    const t = JSON.stringify(payload, null, 2);
    fs.writeFileSync(path.join(runDir, 'last-result.json'), t, 'utf8');
    fs.writeFileSync(path.join(base, 'last-result.json'), t, 'utf8');
    fs.writeFileSync(path.join(runDir, 'RESULT.txt'),
      (res.ok ? 'SUCCESS' : 'FAILED:' + (res.stage || '?')) + '  ' + (res.note || ''), 'utf8');
  } catch { /* ignore */ }
  log('RESULT: ' + (res.ok ? 'SUCCESS' : 'FAILED:' + (res.stage || '?')) + '  ' + (res.note || ''));
  releaseLock();
  process.exitCode = res.ok ? 0 : (res.code || 1);
}

// ─── [1/7] 预检：工作区/分支/版本/目标 tag/是否已有更新 ────────
async function preflight() {
  stage = 'preflight';
  log('=== [1/7] 预检 ===');
  log('仓库: ' + repoPath + ' / 安装目录: ' + installDir);
  if (!fs.existsSync(path.join(repoPath, '.git'))) throw fail(1, '仓库路径不是 git 仓库');
  if (!fs.existsSync(path.join(installDir, 'pair.exe'))) throw fail(1, '安装目录缺少 pair.exe');
  const st = git(repoPath, ['status', '--porcelain']);
  if (st.code !== 0) throw fail(1, 'git status 失败: ' + st.err);
  if (st.out) {
    const lines = st.out.split(/\r?\n/);
    for (const l of lines.slice(0, 20)) log('  dirty: ' + l);
    throw fail(1, '工作区不干净（' + lines.length + ' 项未提交改动，拒绝合并）');
  }
  const br = git(repoPath, ['rev-parse', '--abbrev-ref', 'HEAD']);
  log('分支: ' + br.out);
  if (br.out !== 'master' && !force) throw fail(1, '当前分支不是 master（--force 可跳过）');
  const local = readLocalVersion(repoPath);
  state.from = local.version;
  log('本地版本: ' + local.version + '（' + local.source + '）');
  let tagName = toTag, tagSha = '';
  if (tagName) safeRef(tagName);
  else {
    log('查询上游最新 tag: ' + repoSlug);
    const tags = await fetchGithubTags(repoSlug, process.env.GITHUB_TOKEN || process.env.GH_TOKEN || '');
    const latest = pickLatestStable(Array.isArray(tags) ? tags : []);
    if (!latest) throw fail(1, '上游没有可用 stable tag');
    tagName = safeRef(latest.name);
    tagSha = (latest.commit && latest.commit.sha) || '';
  }
  state.tag = tagName;
  log('目标 tag: ' + tagName + (tagSha ? '（' + tagSha.slice(0, 8) + '）' : ''));
  if (git(repoPath, ['rev-parse', '-q', '--verify', 'refs/tags/' + tagName]).code !== 0) {
    log('本地无该 tag，fetch ' + remote + ' ...');
    const f = git(repoPath, ['fetch', remote, 'tag', tagName]);
    if (f.code !== 0) throw fail(1, 'git fetch 失败: ' + f.err);
  }
  const ahead = parseInt(git(repoPath, ['rev-list', '--count', 'HEAD..' + tagName]).out || '0', 10);
  log('未包含提交数: ' + ahead);
  if (ahead === 0 && !force) { finish({ ok: true, stage: 'noop', note: '已包含 ' + tagName + '，无需更新' }); return false; }
  return true;
}

// ─── [2/7] 快照（备份 tag 先于任何仓库变更）────────────────────
function snapshot() {
  stage = 'snapshot';
  const short = git(repoPath, ['rev-parse', '--short=8', 'HEAD']).out;
  const bTag = 'pre-merge-' + state.tag.replace(/\./g, '') + '-' + short;
  state.backupTag = bTag;
  if (git(repoPath, ['rev-parse', '-q', '--verify', 'refs/tags/' + bTag]).code === 0) { log('备份 tag 已存在: ' + bTag); return; }
  const r = git(repoPath, ['tag', bTag, 'HEAD']);
  if (r.code !== 0) throw fail(1, '创建备份 tag 失败: ' + r.err);
  log('备份 tag: ' + bTag + '（回滚: git reset --hard ' + bTag + '）');
}

// ─── [3/7] 合并上游（冲突自动 abort 还原）─────────────────────
function merge() {
  stage = 'merge';
  const preHead = git(repoPath, ['rev-parse', 'HEAD']).out;
  const sha = git(repoPath, ['rev-parse', '--short=8', state.tag + '^{commit}']).out || state.tag;
  log('=== [3/7] 合并 ' + state.tag + ' ===');
  const m = git(repoPath, ['merge', '--no-ff', '-m', 'merge: 并入官方 ' + state.tag + '（' + sha + '）', state.tag]);
  if (m.code !== 0) {
    const ab = git(repoPath, ['merge', '--abort']);
    log('合并冲突，已 abort 还原: ' + (ab.code === 0 ? 'OK' : ab.err));
    throw fail(2, '合并冲突（已还原，未做任何部署）');
  }
  log('合并完成: ' + git(repoPath, ['log', '-1', '--oneline']).out);
  return preHead;
}

// ─── [6/7] 部署：替换动作全部在 deploy-pair-full.ps1 内完成 ────
function deploy(manifestFile) {
  stage = 'deploy';
  log('=== [6/7] 成对部署（将停服并重启；含停服前备份与失败自动回滚）===');
  const ps = path.join(repoPath, 'scripts', 'deploy-pair-full.ps1');
  const idx = path.join(repoPath, '.pair', 'assets', 'runtime', 'web', 'index.html');
  const shell = fs.existsSync(idx) ? (fs.readFileSync(idx, 'utf8').match(/index-[A-Za-z0-9_-]+\.js/) || [''])[0] : '';
  if (shell) log('新壳: ' + shell);
  const stats = DEFAULTS.baseUrl + '/api/conversations/__probe__/run-stats?workspaceRoot=' +
    encodeURIComponent('D:\\PairCodeData');
  const dargs = ['-NoProfile', '-ExecutionPolicy', 'Bypass', '-File', ps,
    '-Root', runDir, '-Manifest', manifestFile,
    '-ExePath', path.join(installDir, 'pair.exe'), '-InstallDir', installDir,
    '-RollbackRoot', path.parse(installDir).root, '-StatsUrl', stats];
  if (shell) dargs.push('-ExpectShell', shell);
  const started = Date.now();
  const r = spawnSync('powershell', dargs, {
    cwd: repoPath, encoding: 'utf8', maxBuffer: 128 * 1024 * 1024, windowsHide: true,
  });
  const code = r.status === null ? -1 : r.status;
  log('deploy-pair-full exit=' + code + '（' + ((Date.now() - started) / 1000).toFixed(1) + 's）');
  if (code === 90) throw fail(5, '部署失败：已自动回滚（见 ' + runDir + '\\deploy.log）');
  if (code !== 0) throw fail(5, '部署脚本 exit=' + code + '（见 ' + runDir + '\\deploy.log）');
  state.deployed = true;
}

// ─── 主流程 ────────────────────────────────────────────────────
(async () => {
  try {
    log('═══ source-update 开始（run=' + runId + '）═══');
    if (!acquireLock()) return;
    if (!(await preflight())) return;
    if (dryRun) {
      stage = 'dry-run';
      log('[dry-run] 计划: 备份 tag -> git merge --no-ff ' + state.tag +
        ' -> build-ui.mjs + npm run build -> [Go 变化则]exe 构建 -> 清单（排除 ' + SYNC_EXCLUDES.size +
        ' 项）-> deploy-pair-full.ps1');
      finish({ ok: true, stage: 'dry-run', note: '预检通过（' + state.from + ' -> ' + state.tag + '），未执行变更' });
      return;
    }
    snapshot();
    const preHead = merge();
    stage = 'build';
    log('=== [4/7] 重建前端产物 ===');
    buildFrontend(repoPath, log);
    const gc = goChanged(repoPath, preHead, 'HEAD');
    log('合并文件数 ' + gc.count + '，Go 相关变化: ' + gc.changed);
    let exePath = '';
    if (gc.changed) exePath = buildExe(repoPath, state.tag, log);
    else log('Go 无变化，跳过 exe 构建（本次仅同步前端产物）');
    state.exeBuilt = !!exePath;
    stage = 'manifest';
    log('=== [5/7] 生成部署清单 ===');
    const mf = makeManifest({ repoPath, installDir, exePath, runDir, log });
    state.itemCount = mf.count;
    if (mf.count === 0) { finish({ ok: true, stage: 'noop', note: '无差异，无需部署' }); return; }
    if (skipDeploy) { finish({ ok: true, stage: 'skip-deploy', note: '构建与清单完成（未部署）' }); return; }
    deploy(mf.file);
    // ─── [7/7] 部署后：专注模式收纳探针（告警不阻断——收纳问题不致命，不拖垮升级流程）───
    stage = 'focus-probe';
    try {
      log('=== [7/7] 专注收纳探针（部署后自动检查，告警不阻断） ===');
      const { runFocusProbe } = await import('./focus-probe.mjs');
      const webPort = Number(new URL(DEFAULTS.baseUrl).port || 9090);
      const probe = await runFocusProbe({ webPort, log });
      state.focusProbe = { ok: probe.ok, note: probe.note, uncollected: probe.uncollected };
      log('专注收纳探针: ' + probe.note);
    } catch (e) {
      // 探针任何异常都不改变更新结果（含环境缺失：无 Edge/端口占用等）
      state.focusProbe = { ok: false, note: '探针异常（不阻断）: ' + (e && e.message || e) };
      log('专注收纳探针异常（不阻断部署）: ' + (e && e.message || e));
    }
    finish({ ok: true, stage: 'done', note: state.from + ' -> ' + state.tag + '，部署 ' + mf.count +
      ' 项' + (state.exeBuilt ? '（含新 exe）' : '（仅前端）') +
      (state.focusProbe ? '；专注探针 ' + state.focusProbe.note : '') });
  } catch (e) {
    finish({ ok: false, stage: e.stage || stage, note: e.message, code: e.exitCode || 1 });
  }
})();
// 兜底强退：unref 的 timer 不阻止正常退出；仅在 event loop 意外被挂住时兜底。
setTimeout(() => process.exit(process.exitCode || 0), 10000).unref();
