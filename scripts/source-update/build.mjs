// build.mjs — source-update 构建阶段：重建 UI 区域包 / vite 壳 / [按需] exe / 生成部署清单。
// 子进程一律 spawnSync 数组传参（shell:false）；唯一 shell:true 系固定常量 "npm run build"。
// 导出：buildFrontend / goChanged / buildExe / makeManifest（供 update.mjs 调用）。
import fs from 'node:fs';
import path from 'node:path';
import { spawnSync } from 'node:child_process';
import { SYNC_EXCLUDES, SKIP_DIR, SKIP_FILE, git, sha256File } from './lib.mjs';

// 统一执行器：数组传参，无字符串拼接（opts.shell 仅用于固定常量命令）
function run(cmd, cmdArgs, opts = {}) {
  if (opts.log) opts.log('$ ' + cmd + (cmdArgs.length ? ' ' + cmdArgs.join(' ') : '') + (opts.cwd ? '   (cwd=' + opts.cwd + ')' : ''));
  const r = spawnSync(cmd, cmdArgs, {
    cwd: opts.cwd, encoding: 'utf8', maxBuffer: 128 * 1024 * 1024,
    windowsHide: true, shell: !!opts.shell, env: opts.env || process.env,
  });
  const code = r.status === null ? -1 : r.status;
  const out = String(r.stdout || '').trim(), err = String(r.stderr || '').trim();
  if (opts.log) {
    for (const l of out.split(/\r?\n/)) if (l.trim()) opts.log('  | ' + l);
    for (const l of err.split(/\r?\n/)) if (l.trim()) opts.log('  ! ' + l);
    opts.log('  -> exit=' + code);
  }
  return { code, out, err };
}

// [4a/4b] 重建前端：UI 区域包（→.pair/plugins/ui-*/assets/）+ vite 壳（→.pair/assets/runtime/web）
export function buildFrontend(repoPath, log) {
  const ui = run(process.execPath, ['scripts/build-ui.mjs'], { cwd: repoPath, log });
  if (ui.code !== 0) throw new Error('node scripts/build-ui.mjs 失败');
  const web = run('npm run build', [], { cwd: path.join(repoPath, 'plugins-src', 'ui-app'), shell: true, log });
  if (web.code !== 0) throw new Error('npm run build（vite 壳）失败');
}

// Go 相关文件是否有变化（合并区间），决定是否需要重建 exe
export function goChanged(repoPath, fromRef, toRef) {
  const files = git(repoPath, ['diff', '--name-only', fromRef + '..' + toRef]).out.split(/\r?\n/).filter(Boolean);
  const changed = files.some((f) => f === 'go.mod' || f === 'go.sum' || /\.go$/.test(f) ||
    /^cmd\//.test(f) || /^internal\//.test(f) || /^pkg\//.test(f));
  return { changed, count: files.length };
}

function findWindres() {
  if (process.env.WINDRES && fs.existsSync(process.env.WINDRES)) return process.env.WINDRES;
  if (fs.existsSync('D:\\mingw64\\bin\\windres.exe')) return 'D:\\mingw64\\bin\\windres.exe';
  const w = spawnSync('windres', ['--version'], { encoding: 'utf8', windowsHide: true });
  return w.status === 0 ? 'windres' : '';
}

// [4c] exe 构建：资源 syso 缺失才生成（避免部署无图标 exe）；命名 pair-deploy-<tag 去点>.exe
export function buildExe(repoPath, tag, log) {
  const dir = path.join(repoPath, 'cmd', 'companion');
  const syso = ['rsrc_windows_amd64.syso', 'companion.syso'].map((n) => path.join(dir, n));
  if (!syso.some((p) => fs.existsSync(p))) {
    log('资源 syso 缺失，生成中...');
    if (!fs.existsSync(path.join(dir, 'icon.ico'))) {
      const py = run('python', ['-c',
        "from PIL import Image; im=Image.open('icon_256.png').convert('RGBA'); im.save('icon.ico', sizes=[(16,16),(24,24),(32,32),(48,48),(64,64),(128,128),(256,256)])"],
        { cwd: dir, log });
      if (py.code !== 0) throw new Error('生成 icon.ico 失败（需 python + Pillow）');
    }
    const wr = findWindres();
    if (!wr) throw new Error('windres 未找到（D:\\mingw64\\bin\\windres.exe 或 PATH / WINDRES 环境变量）');
    const w = run(wr, ['-I.', '--input=companion.rc', '--output=rsrc_windows_amd64.syso',
      '--output-format=coff', '--target=pe-x86-64'], { cwd: dir, log });
    if (w.code !== 0) throw new Error('windres 编译资源失败（中止以避免部署无图标 exe）');
  } else {
    log('资源 syso 已存在，跳过 rsrc 生成');
  }
  const exePath = path.join(repoPath, 'temp', 'build', 'pair-deploy-' + tag.replace(/\./g, '') + '.exe');
  fs.mkdirSync(path.dirname(exePath), { recursive: true });
  const gb = run('go', ['build', '-o', exePath, './cmd/companion'], {
    cwd: repoPath, log, env: Object.assign({}, process.env, { CGO_ENABLED: '1' }),
  });
  if (gb.code !== 0) throw new Error('go build 失败');
  log('exe 产出: ' + exePath + '  (' + (fs.statSync(exePath).size / 1024 / 1024).toFixed(1) + ' MB)');
  return exePath;
}

// [5] 部署清单：仓库 .pair/{plugins,assets} 与安装目录的全量差异（SYNC_EXCLUDES 除外）+ exe 项
//     仅安装侧独有的文件绝不触碰；本函数只读文件+写 runDir 下清单，不修改安装目录。
function walk(root, sub) {
  const out = new Map();
  const top = path.join(root, sub);
  if (!fs.existsSync(top)) return out;
  const stack = [top];
  while (stack.length) {
    const d = stack.pop();
    let es; try { es = fs.readdirSync(d, { withFileTypes: true }); } catch { continue; }
    for (const e of es) {
      const p = path.join(d, e.name);
      const rel = path.relative(top, p).split(path.sep).join('/');
      if (e.isDirectory()) { if (!SKIP_DIR.test(rel + '/')) stack.push(p); }
      else if (!SKIP_FILE.test(e.name)) out.set(rel, p);
    }
  }
  return out;
}
function classify(rel) {
  if (rel.endsWith('.exe')) return 'exe';
  if (rel.endsWith('index.html')) return 'shell';
  return rel.startsWith('.pair/plugins/') ? 'plugin' : 'asset';
}
export function makeManifest({ repoPath, installDir, exePath, runDir, log }) {
  const items = [];
  let same = 0, diff = 0, onlyRepo = 0, onlyD = 0, excluded = 0;
  for (const sub of ['.pair/plugins', '.pair/assets']) {
    const r = walk(repoPath, sub), d = walk(installDir, sub);
    for (const k of [...new Set([...r.keys(), ...d.keys()])].sort()) {
      const rel = sub + '/' + k;
      if (r.has(k) && d.has(k)) {
        if (sha256File(r.get(k)) === sha256File(d.get(k))) { same++; continue; }
        diff++;
      } else if (r.has(k)) { onlyRepo++; } else { onlyD++; continue; }
      if (SYNC_EXCLUDES.has(rel)) { excluded++; continue; }
      const src = r.get(k);
      items.push({ src, dst: path.join(installDir, sub, k), kind: classify(rel),
        srcSha: sha256File(src), srcSize: fs.statSync(src).size });
    }
  }
  if (exePath && fs.existsSync(exePath)) {
    items.push({ src: exePath, dst: path.join(installDir, 'pair.exe'), kind: 'exe',
      srcSha: sha256File(exePath), srcSize: fs.statSync(exePath).size });
  }
  const file = path.join(runDir, 'deploy-manifest.json');
  fs.writeFileSync(file, JSON.stringify(items, null, 2), 'utf8');
  log('差异: SAME=' + same + ' DIFF=' + diff + ' 仅仓库=' + onlyRepo + ' 仅D=' + onlyD + '（排除 ' + excluded + '）');
  log('清单: ' + file + '  共 ' + items.length + ' 项');
  for (const it of items) log('  [' + it.kind + '] ' + it.dst);
  return { file, count: items.length };
}
