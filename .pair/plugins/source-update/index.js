// source-update — host 半：源码仓库更新检查与一键更新。
//
// 机制（更新引擎在仓库脚本，本插件只做调度与交互）：
//   · 检查：启动 30s 后 + 每 N 小时，调 <repo>/scripts/source-update/check.mjs --json
//     （GitHub 最新 stable tag vs 本地 main.go 版本；以输出 JSON 的 ok 字段判成败）。
//     结果广播 'ui:source-update/state'。
//   · 提示：client 半（client.js）收到广播后弹窗（立即更新/稍后/忽略此版本）。
//   · 更新：用户确认后分离启动 <repo>/scripts/source-update/run-update.cmd
//     （独立进程树跑 update.mjs：合并 -> 重建前端 -> [按需]exe -> 成对部署 -> 重启；
//      部署阶段会停本宿主，属预期——页面稍后自动重连）。
//
// 与 app-update（官方 Releases 二进制更新）分工不同、互不影响：本插件走
// “源码合并 + 本机自构建”流程，app-update 走官方发行包替换。
return {
  name: 'source-update',
  purpose: '源码仓库更新检查与一键更新（上游 tag 比对 -> 弹窗提示 -> 全自动合并/构建/成对部署）',
  inject: ['logger', 'bash', 'timer'],
  apply(ctx) {
    const log = (m) => ctx.logger('source-update').log(m);
    // shell 引号包装（路径可能含空格/中文；参照 git-api 插件惯例）
    const q = (s) => '"' + String(s).replace(/"/g, '\\"') + '"';

    const DEFAULTS = {
      enabled: true,
      repo: 'hoonfeng/paircode',
      repoPath: 'D:\\PairCodeData\\开发源码\\paircode-master',
      checkOnStart: true,
      checkDelaySec: 30,
      intervalHours: 24,
      ignored: '',
    };
    const cfg = () => {
      let raw = {};
      try { raw = (ctx.getSettings && ctx.getSettings('source-update')) || {}; } catch (e) { raw = {}; }
      return Object.assign({}, DEFAULTS, raw);
    };

    const state = {
      phase: 'idle', // idle | checking | available | up-to-date | updating | error
      current: '', latest: '', commit: '', hasUpdate: false,
      ignored: '', error: '', at: '',
    };
    const scriptsDir = () => cfg().repoPath + '\\scripts\\source-update';
    const broadcast = () => { try { ctx.emit('ui:source-update/state', Object.assign({}, state)); } catch (e) { /* ignore */ } };

    // ── 检查（调 check.mjs --json）──
    function runCheck(trigger) {
      const c = cfg();
      if (!c.enabled) { log('检查已禁用（设置 → 源码更新 → 启用检查）'); return state; }
      state.phase = 'checking'; state.error = ''; state.at = new Date().toISOString(); broadcast();
      let payload = null;
      try {
        const cmd = 'node ' + q(scriptsDir() + '\\check.mjs') + ' --json --repo ' + q(c.repo) + ' --repo-path ' + q(c.repoPath);
        const r = ctx.bash.exec(cmd, c.repoPath);
        const out = ((r && r.output) || '').trim();
        const line = out.split(/\r?\n/).filter((l) => l.trim()).pop() || '';
        payload = JSON.parse(line);
      } catch (e) {
        payload = { ok: false, error: String((e && e.message) || e) };
      }
      if (!payload || !payload.ok) {
        state.phase = 'error'; state.error = (payload && payload.error) || '检查失败（无输出）'; broadcast();
        log('检查失败[' + trigger + ']: ' + state.error);
        return state;
      }
      state.current = payload.current || ''; state.latest = payload.latest || '';
      state.commit = payload.commit || '';
      state.ignored = c.ignored || '';
      state.hasUpdate = !!payload.hasUpdate && state.latest !== state.ignored;
      state.phase = state.hasUpdate ? 'available' : 'up-to-date';
      state.error = ''; broadcast();
      log('检查[' + trigger + '] ' + state.current + ' -> ' + state.latest + '，有更新=' + state.hasUpdate +
        (state.latest === state.ignored ? '（已忽略该版本）' : ''));
      return state;
    }

    // ── client 方法（浏览器 ui.invoke('source-update', <method>)）──
    ctx.registerClientMethod('getState', () => Object.assign({}, state));
    ctx.registerClientMethod('checkNow', () => runCheck('manual'));
    ctx.registerClientMethod('later', () => { state.phase = 'idle'; broadcast(); return { ok: true }; });
    ctx.registerClientMethod('ignoreVersion', () => {
      const c = cfg();
      try { ctx.setSettings('source-update', Object.assign({}, c, { ignored: state.latest })); }
      catch (e) { log('写入忽略版本失败: ' + String((e && e.message) || e)); }
      state.ignored = state.latest; state.hasUpdate = false; state.phase = 'idle'; broadcast();
      log('已忽略版本 ' + state.latest + '（设置 → 源码更新 → 忽略的版本 可清除）');
      return { ok: true };
    });
    ctx.registerClientMethod('startUpdate', () => {
      // 分离启动：独立进程树运行 run-update.cmd（cmd start）。
      // 部署阶段会停本宿主，属预期；结果写 <repo>/temp/source-update/last-result.json。
      const cmdline = 'cmd //c start "PairCode update" cmd //c ' + q(scriptsDir() + '\\run-update.cmd');
      try {
        ctx.bash.exec(cmdline, cfg().repoPath);
        state.phase = 'updating'; broadcast();
        log('已分离启动更新进程（run-update.cmd）');
        return { ok: true };
      } catch (e) {
        const msg = String((e && e.message) || e);
        log('启动更新失败: ' + msg);
        return { ok: false, msg: msg };
      }
    });

    // ── 设置段（设置 → 源码更新）──
    ctx.registerSettings({
      key: 'source-update',
      title: '源码更新',
      fields: [
        { name: 'enabled', label: '启用检查', type: 'boolean', default: true, hint: '启动与周期检查上游源码仓库的新版本' },
        { name: 'repo', label: '上游仓库', type: 'text', default: 'hoonfeng/paircode', hint: 'owner/repo（官方源码仓库）' },
        { name: 'repoPath', label: '本地源码仓库路径', type: 'text', default: 'D:\\PairCodeData\\开发源码\\paircode-master', hint: '含 scripts/source-update/ 的仓库根' },
        { name: 'checkOnStart', label: '启动后自动检查', type: 'boolean', default: true, hint: '启动 30 秒后检查一次；有更新时弹窗提示，不自动安装' },
        { name: 'checkDelaySec', label: '启动检查延迟（秒）', type: 'number', default: 30, min: 5, max: 600 },
        { name: 'intervalHours', label: '周期检查间隔（小时）', type: 'number', default: 24, min: 1, max: 720 },
        { name: 'ignored', label: '忽略的版本', type: 'text', placeholder: 'v1.6.10', hint: '该版本不再提示；清空即恢复提示' },
      ],
    });

    // ── 调度：启动延时检查 + 周期检查 ──
    const c0 = cfg();
    if (c0.checkOnStart) {
      const delay = Math.max(5, c0.checkDelaySec || 30);
      ctx.timer.timeout(() => runCheck('startup'), delay * 1000);
      log('启动检查已排程（' + delay + 's 后）');
    }
    const cancelInterval = ctx.timer.interval(() => runCheck('interval'),
      Math.max(1, c0.intervalHours || 24) * 3600 * 1000);
    ctx.effect(() => { try { cancelInterval(); } catch (e) { /* ignore */ } });

    log('已就绪（上游=' + c0.repo + '，仓库=' + c0.repoPath + '）');
  },
}
