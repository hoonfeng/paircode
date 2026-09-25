// ═══════════════════════════════════════════════════════════════
// check.mjs — 检查上游（hoonfeng/paircode）是否有新版本。
//
// 判定：GitHub 最新 stable tag（semver 最大、排除预发布）
//       vs 本地 cmd/companion/main.go 的 version。
//
// 用法（cwd 任意）：
//   node scripts/source-update/check.mjs                   # 人类可读
//   node scripts/source-update/check.mjs --json            # 单行 JSON（插件消费）
//   node scripts/source-update/check.mjs --current v1.6.8  # 覆盖本地版本（演练/测试）
//   node scripts/source-update/check.mjs --repo owner/repo --repo-path <path>
//
// 退出码：0 = 检查完成（无论有无更新）；1 = 检查失败（网络/参数）。
// JSON 字段：ok/current/currentSource/latest/commit/hasUpdate/error/checkedAt
// 可选环境变量：GITHUB_TOKEN / GH_TOKEN（提高 API 限额）
// ═══════════════════════════════════════════════════════════════
import { DEFAULTS, parseArgs, cmpVer, readLocalVersion, fetchGithubTags, pickLatestStable } from './lib.mjs';

const args = parseArgs(process.argv.slice(2));
const repoPath = args['repo-path'] || DEFAULTS.repoPath;
const repo = args.repo || DEFAULTS.repo;

const out = { ok: false, repo, checkedAt: new Date().toISOString() };

try {
  const local = args.current
    ? { version: String(args.current), source: '--current override' }
    : readLocalVersion(repoPath);
  out.current = local.version;
  out.currentSource = local.source;
  if (!local.version) throw new Error('无法读取本地版本（cmd/companion/main.go 缺失或格式变化）');

  const token = process.env.GITHUB_TOKEN || process.env.GH_TOKEN || '';
  const tags = await fetchGithubTags(repo, token);
  const latest = pickLatestStable(Array.isArray(tags) ? tags : []);
  if (!latest) throw new Error('上游没有可用的 stable tag');

  out.latest = latest.name;
  out.commit = (latest.commit && latest.commit.sha) || '';
  out.hasUpdate = cmpVer(latest.name, local.version) > 0;
  out.ok = true;

  if (args.json) {
    process.stdout.write(JSON.stringify(out) + '\n');
  } else {
    console.log('[source-update/check] 上游: ' + repo);
    console.log('  本地版本: ' + out.current + '  (' + out.currentSource + ')');
    console.log('  最新 tag: ' + out.latest + '  (' + String(out.commit).slice(0, 8) + ')');
    console.log('  结论: ' + (out.hasUpdate ? '★ 有更新 ' + out.current + ' -> ' + out.latest : '已是最新'));
  }
  process.exitCode = 0;
} catch (e) {
  out.error = String((e && e.message) || e);
  if (args.json) process.stdout.write(JSON.stringify(out) + '\n');
  else console.error('[source-update/check] 失败: ' + out.error);
  process.exitCode = 1;
}

// 兜底强退：unref 的 timer 不阻止正常退出；仅在 event loop 被意外挂住时兜底。
setTimeout(() => process.exit(process.exitCode || 0), 5000).unref();
