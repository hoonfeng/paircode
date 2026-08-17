// ═══════════════════════════════════════════════════════════════
// ui-quick-exec — host 半
//
// 「快速执行」菜单（标题栏 titlebar-right 槽位）：
//   动态读取工作区配置 .pair/quick-exec.json 生成菜单，
//   点击菜单项在宿主执行配置的命令（工作区根为 cwd）。
//   菜单第一项「配置命令」打开前端弹窗，管理命令列表（增删改查+排序）。
//
// 设计要点：
//   · 纯 UI 插件——不注册任何 agent 工具（不触碰 ctx.tools），
//     命令对 agent 完全不可见，仅人机交互。
//   · 全局生效（package.json scope=global；含 client 半自动 global）。
//   · 配置按工作区隔离（ctx.fs 限定工作区根，.pair/quick-exec.json）。
// ═══════════════════════════════════════════════════════════════
return {
  name: 'ui-quick-exec',
  purpose: '快速执行菜单：工作区配置命令列表，点击即执行（纯 UI，agent 不可见）',
  inject: ['fs', 'bash', 'app'],
  apply(ctx) {
    const CFG = '.pair/quick-exec.json'

    // 读取命令列表（容错：缺文件/坏 JSON → 空列表）
    function loadCommands() {
      try {
        if (!ctx.fs.exists(CFG)) return []
        const data = JSON.parse(ctx.fs.readFile(CFG))
        if (data && Array.isArray(data.commands)) {
          return data.commands.filter((c) => c && typeof c.name === 'string' && typeof c.command === 'string')
        }
      } catch (e) {
        console.warn('[ui-quick-exec] 读取 ' + CFG + ' 失败（按空列表处理）: ' + (e && e.message || e))
      }
      return []
    }

    // 写命令列表
    function saveCommands(cmds) {
      const data = { version: 1, updatedAt: new Date().toISOString(), commands: cmds }
      ctx.fs.writeFile(CFG, JSON.stringify(data, null, 2))
      return { ok: true, count: cmds.length }
    }

    // ── client 半可调方法（ui.invoke('ui-quick-exec', method, args)）──

    // 取命令列表 + 工作区名（菜单每次打开时动态拉取 → 手工改配置/切换工作区即生效）
    ctx.registerClientMethod('getCommands', () => {
      const root = ctx.app.workspaceRoot || ''
      const base = root.replace(/[\\/]+$/, '')
      const wsName = base.split(/[\\/]/).pop() || base
      return { commands: loadCommands(), workspace: wsName, workspaceRoot: root }
    })

    // 保存命令列表（配置弹窗保存时调用）
    ctx.registerClientMethod('saveCommands', (args) => {
      const cmds = args && Array.isArray(args.commands) ? args.commands : []
      return saveCommands(cmds)
    })

    // 执行命令（独立二进制 ui-quick-exec.exe，工作区根 cwd，超时可配默认 600s）
    // ★ 2026-08-17：原走 ctx.bash.exec（宿主 runShellWithTimeout 硬编码 120s，
    // 打包类长命令超 120s 被强制 kill）→ 改经 ctx.binary.exec，超时由 timeoutMs
    // 精确控制（opts.timeout = timeoutMs+10s 作外层保护），完全不经宿主 120s 桥接。
    ctx.registerClientMethod('runCommand', (args) => {
      const command = args && typeof args.command === 'string' ? args.command.trim() : ''
      const timeoutMs = args && typeof args.timeoutMs === 'number' && args.timeoutMs > 0
        ? Math.round(args.timeoutMs)
        : 600000 // 默认 600s
      if (!command) {
        return { ok: false, command: '', output: '', error: '命令为空', exitCode: -1, timedOut: false, durationMs: 0, timeoutMs }
      }
      try {
        const res = ctx.binary.exec('run', { command, timeoutMs }, { timeout: timeoutMs + 10000 })
        let data = {}
        try { data = JSON.parse((res && res.text) || '{}') } catch (e) { /* 二进制返回非 JSON（理论上不会） */ }
        const timedOut = !!data.timedOut
        const exitCode = typeof data.exitCode === 'number' ? data.exitCode : 0
        let error = ''
        if (timedOut) error = '命令超时（超过 ' + Math.round(timeoutMs / 1000) + ' 秒，已强制结束）'
        else if (exitCode !== 0) error = '命令退出码: ' + exitCode
        return {
          ok: !timedOut && exitCode === 0,
          command: command,
          output: data.output || '',
          error: error,
          exitCode: exitCode,
          timedOut: timedOut,
          durationMs: typeof data.durationMs === 'number' ? data.durationMs : 0,
          timeoutMs: timeoutMs,
        }
      } catch (e) {
        const msg = (e && e.message) || String(e)
        return {
          ok: false,
          command: command,
          output: '',
          error: msg,
          exitCode: -1,
          timedOut: /超时/.test(msg),
          durationMs: 0,
          timeoutMs: timeoutMs,
        }
      }
    })

    console.log('[ui-quick-exec] host 半已装载（配置: ' + CFG + '）')
  },
}
