// ═══════════════════════════════════════════════════════════════
// git-api — Git 接口插件化（17 条）+ Git 源代码管理面板（UI 入插件）
//
// 背景（2026-08-19）：Git 接口原实现全部在 Go 内核（web_server.go handleGit*），
// 经 core-api 路由表挂载。本插件把 17 条 Git 接口整体搬进插件：
//   - host 半：ctx.webServer.register 注册同名路径接口（ExtRouteMiddleware
//     优先于宿主 mux 拦截），用 ctx.bash 调 git 命令——core-api 已删除
//     对应内核 key，本插件即唯一实现；
//   - client 半：注册 Git 面板（ui.registerPanel），编译产物
//     assets/git-panel.js 承载 GitPanel 组件（UI 与接口同插件）。
//   - 生命周期：卸载插件 = Git 接口与 Git 面板同时消失（一体化）。
// ═══════════════════════════════════════════════════════════════
return {
  name: 'git-api',
  purpose: 'Git 接口插件化（17 条：status/init/diff/add/reset/commit/log/branch/checkout/stash/ignore/discard/push/pull/remote）+ Git 源代码管理面板',
  inject: ['bash', 'fs', 'logger'],
  apply(ctx) {
    // ── 响应助手（对齐 Go jsonResp/jsonErr）──
        // query 解析（req.query 是 RawQuery 字符串）
    const qp = (req, key) => {
      const q = req.query || ''
      const m = new RegExp('(?:^|[?&])' + key + '=([^&]*)').exec(q)
      return m ? decodeURIComponent(m[1].replace(/\+/g, ' ')) : ''
    }
const ok = (data) => ({ status: 200, headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(data) })
    const err = (msg) => ({ status: 400, headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ error: String(msg) }) })

    // ── git 执行助手（对齐 runGitInternal）──
    // cmd 形态：git -C <dir> -c core.quotepath=false <args...>（shell=cmd /C + chcp 65001）
    const q = (s) => '"' + String(s).replace(/"/g, '\\"') + '"'
    function gitRun(req, args) {
      const dir = qp(req, 'path') || ctx.workspaceRoot || (ctx.app && ctx.app.workspaceRoot) || ''
      if (!dir) return { fail: true, msg: '未设置工作区', out: '' }
      const cmd = 'git -C ' + q(dir) + ' -c core.quotepath=false ' + args.map(q).join(' ')
      let r
      try { r = ctx.bash.exec(cmd, dir) } catch (e) { return { fail: true, msg: String(e && e.message || e), out: '' } }
      const out = (r && r.output || '').trim()
      if (r && r.error && !out) return { fail: true, msg: r.error, out: '' }
      return { fail: false, msg: '', out }
    }
    const bodyOf = (req) => { try { return req.json ? req.json() : {} } catch (e) { return {} } }

    // ── 接口注册 ──
    const R = []
    const reg = (path, handler) => R.push([path, handler])
    // 注册入口（handler(req) → {status,body,headers}）
    function registerAll() {
      for (const [path, handler] of R) {
        try {
          ctx.webServer.register({ kind: 'exact', path, handler: (req, res) => {
            const out = handler(req)
            if (out === undefined) return
            res.writeHead(out.status, out.headers)
            res.end(out.body)
          }})
        } catch (e) {
          if (ctx.logger) ctx.logger('git-api').warn('注册失败 ' + path + ': ' + e)
        }
      }
    }

    // ── 1. status ──
    reg('/api/git/status', (req) => {
      const res = { branch: '', ahead: 0, behind: 0, isRepo: false, staged: [], conflict: [], modified: [], untracked: [], brances: [] }
      const check = gitRun(req, ['rev-parse', '--is-inside-work-tree'])
      if (check.fail || check.out !== 'true') { res.error = '非 Git 仓库或未设置工作区'; return ok(res) }
      res.isRepo = true
      const br = gitRun(req, ['branch', '--show-current'])
      if (!br.fail) res.branch = br.out
      const ab = gitRun(req, ['rev-list', '--left-right', '--count', 'HEAD...@{upstream}'])
      if (!ab.fail) {
        const m = ab.out.match(/(\d+)\s+(\d+)/)
        if (m) { res.ahead = parseInt(m[1]); res.behind = parseInt(m[2]) }
      }
      const st = gitRun(req, ['status', '--porcelain'])
      if (!st.fail) {
        for (const line of st.out.split('\n')) {
          if (line.length < 4) continue
          const x = line[0], y = line[1]
          let p = line.slice(3).trim()
          const i = p.indexOf(' -> ')
          if (i >= 0) p = p.slice(i + 4)
          const e = { path: p, x, y }
          if (x === '?' && y === '?') res.untracked.push(e)
          else if (x === 'U' || y === 'U' || (x === 'D' && y === 'D') || (x === 'A' && y === 'A')) res.conflict.push(e)
          else {
            if (x !== ' ' && x !== '?') res.staged.push(e)
            if (y !== ' ' && y !== '?') res.modified.push(e)
          }
        }
      }
      const bl = gitRun(req, ['branch', '--format=%(refname:short)'])
      if (!bl.fail) res.brances = bl.out.split('\n').map(s => s.trim()).filter(Boolean)
      return ok(res)
    })

    // ── 2. init（POST，返回 {output}）──
    reg('/api/git/init', (req) => {
      if (req.method !== 'POST') return err('仅 POST')
      const r = gitRun(req, ['init'])
      if (r.fail) return err(r.msg)
      return ok({ output: r.out })
    })

    // ── 3. diff（GET ?file=&staged= → {diff}）──
    reg('/api/git/diff', (req) => {
      const file = qp(req, 'file') || ''
      const staged = qp(req, 'staged') === 'true'
      const args = ['diff']
      if (staged) args.push('--cached')
      if (file) args.push('--', file)
      const r = gitRun(req, args)
      if (r.fail) return err(r.msg)
      return ok({ diff: r.out || '（无改动）' })
    })

    // ── 4. add（POST {files:[]}）──
    reg('/api/git/add', (req) => {
      if (req.method !== 'POST') return err('仅 POST')
      const b = bodyOf(req)
      const args = ['add']
      if (b.files && b.files.length) args.push('--', ...b.files)
      else args.push('-A')
      const r = gitRun(req, args)
      if (r.fail) return err(r.msg)
      return ok({ ok: true })
    })

    // ── 5. reset（POST {files:[]}）──
    reg('/api/git/reset', (req) => {
      if (req.method !== 'POST') return err('仅 POST')
      const b = bodyOf(req)
      const args = ['reset', '-q', 'HEAD', '--']
      if (b.files && b.files.length) args.push(...b.files)
      else args.push('.')
      const r = gitRun(req, args)
      if (r.fail) return err(r.msg)
      return ok({ ok: true })
    })

    // ── 6. commit（POST {message, all}）──
    reg('/api/git/commit', (req) => {
      if (req.method !== 'POST') return err('仅 POST')
      const b = bodyOf(req)
      if (!String(b.message || '').trim()) return err('提交信息不能为空')
      const args = ['commit', '-m', b.message]
      if (b.all) args.push('-a')
      const r = gitRun(req, args)
      if (r.fail) return err(r.msg)
      return ok({ ok: true, output: r.out })
    })

    // ── 7/8. log + log.alias（GET ?count=&file= → [{hash,short,author,date,msg}]）──
    const logHandler = (req) => {
      const count = qp(req, 'count') || '50'
      const file = qp(req, 'file') || ''
      const args = ['log', '--max-count=' + count, '--pretty=format:%H|%h|%an|%ai|%B']
      if (file) args.push('--', file)
      const r = gitRun(req, args)
      if (r.fail) return err(r.msg)
      const commits = []
      for (const line of r.out.split('\n')) {
        const t = line.trim()
        if (!t) continue
        const parts = t.split('|')
        if (parts.length >= 5) commits.push({ hash: parts[0], short: parts[1], author: parts[2], date: parts[3], msg: parts.slice(4).join('|') })
      }
      return ok(commits)
    }
    reg('/api/git/log', logHandler)
    reg('/api/git-log', logHandler)

    // ── 9. branch（POST {action:list|create|delete|switch|create-switch|rename, name, newName}）──
    reg('/api/git/branch', (req) => {
      if (req.method !== 'POST') return err('仅 POST')
      const b = bodyOf(req)
      switch (b.action) {
        case 'list': {
          const r = gitRun(req, ['branch', '--all', '--format=%(refname:short)'])
          if (r.fail) return err(r.msg)
          return ok(r.out.split('\n'))
        }
        case 'create': {
          if (!b.name) return err('分支名不能为空')
          const r = gitRun(req, ['branch', b.name])
          if (r.fail) return err(r.msg)
          return ok({ ok: true })
        }
        case 'delete': {
          if (!b.name) return err('分支名不能为空')
          const r = gitRun(req, ['branch', '-D', b.name])
          if (r.fail) return err(r.msg)
          return ok({ ok: true })
        }
        case 'switch': {
          if (!b.name) return err('分支名不能为空')
          const r = gitRun(req, ['checkout', b.name])
          if (r.fail) return err(r.msg)
          return ok({ ok: true })
        }
        case 'create-switch': {
          if (!b.name) return err('分支名不能为空')
          const r = gitRun(req, ['checkout', '-b', b.name])
          if (r.fail) return err(r.msg)
          return ok({ ok: true })
        }
        case 'rename': {
          if (!b.name || !b.newName) return err('name 和 newName 不能为空')
          const r = gitRun(req, ['branch', '-m', b.name, b.newName])
          if (r.fail) return err(r.msg)
          return ok({ ok: true })
        }
        default: return err('未知操作: ' + b.action)
      }
    })

    // ── 10. checkout（POST {branch, create}）──
    reg('/api/git/checkout', (req) => {
      if (req.method !== 'POST') return err('仅 POST')
      const b = bodyOf(req)
      const args = ['checkout']
      if (b.create) args.push('-b')
      args.push(b.branch)
      const r = gitRun(req, args)
      if (r.fail) return err(r.msg)
      return ok({ ok: true })
    })

    // ── 11. stash（POST {action:push|pop|apply|drop, message, index}）──
    reg('/api/git/stash', (req) => {
      if (req.method !== 'POST') return err('仅 POST')
      const b = bodyOf(req)
      const action = b.action || 'push'
      let args
      switch (action) {
        case 'push': args = ['stash', 'push']; if (String(b.message || '').trim()) args.push('-m', b.message); break
        case 'pop': args = ['stash', 'pop']; if (b.index) args.push(b.index); break
        case 'apply': args = ['stash', 'apply']; if (b.index) args.push(b.index); break
        case 'drop': args = ['stash', 'drop']; if (b.index) args.push(b.index); break
        default: return err('未知 stash 操作: ' + action)
      }
      const r = gitRun(req, args)
      if (r.fail) return err(r.msg)
      return ok({ ok: true, output: r.out })
    })

    // ── 12. stash-list（GET → [{index,ref,msg,date}]）──
    reg('/api/git/stash-list', (req) => {
      const r = gitRun(req, ['stash', 'list', '--format=%(stash): %(stashsubject) | %(stashdate:short)'])
      if (r.fail) return ok([])
      const stashes = []
      for (const line of r.out.split('\n')) {
        const t = line.trim()
        if (!t) continue
        const parts = t.split(': ')
        if (parts.length < 2) continue
        const ref = parts[0]
        const rest = parts.slice(1).join(': ')
        const detail = rest.split(' | ')
        const msg = detail[0], date = detail.length > 1 ? detail[1] : ''
        stashes.push({ index: ref, ref, msg, date })
      }
      return ok(stashes)
    })

    // ── 13. ignore（GET → {content,rules}；POST {content|append}）──
    reg('/api/git/ignore', (req) => {
      const dir = qp(req, 'path') || ctx.workspaceRoot || (ctx.app && ctx.app.workspaceRoot) || ''
      const ignorePath = dir ? dir.replace(/[\\/]+$/, '') + '/.gitignore' : ''
      if (req.method === 'POST') {
        const b = bodyOf(req)
        if (b.append) {
          try {
            if (ctx.fs.exists(ignorePath)) ctx.fs.appendFile(ignorePath, '\n' + b.append + '\n')
            else ctx.fs.writeFile(ignorePath, b.append + '\n')
            return ok({ ok: true, appended: b.append })
          } catch (e) { return err(String(e && e.message || e)) }
        }
        try {
          ctx.fs.writeFile(ignorePath, String(b.content || ''))
          return ok({ ok: true })
        } catch (e) { return err(String(e && e.message || e)) }
      }
      try {
        const content = ctx.fs.exists(ignorePath) ? ctx.fs.readFile(ignorePath) : ''
        return ok({ content, rules: content.split('\n') })
      } catch (e) { return ok({ content: '', rules: [] }) }
    })

    // ── 14. discard（POST {files:[]}）──
    reg('/api/git/discard', (req) => {
      if (req.method !== 'POST') return err('仅 POST')
      const b = bodyOf(req)
      if (!(b.files && b.files.length)) return err('请指定要丢弃的文件')
      const r = gitRun(req, ['checkout', '--', ...b.files])
      if (r.fail) return err(r.msg)
      return ok({ ok: true })
    })

    // ── 15. push（POST {remote, branch} → {ok, output}）──
    reg('/api/git/push', (req) => {
      if (req.method !== 'POST') return err('仅 POST')
      const b = bodyOf(req)
      const args = ['push']
      if (b.remote) args.push(b.remote)
      if (b.branch) args.push(b.branch)
      const r = gitRun(req, args)
      if (r.fail) return err(r.msg)
      return ok({ ok: true, output: r.out })
    })

    // ── 16. pull（POST {remote, branch, rebase} → {ok, output}）──
    reg('/api/git/pull', (req) => {
      if (req.method !== 'POST') return err('仅 POST')
      const b = bodyOf(req)
      const args = ['pull']
      args.push(b.rebase ? '--rebase' : '--ff-only')
      if (b.remote) args.push(b.remote)
      if (b.branch) args.push(b.branch)
      const r = gitRun(req, args)
      if (r.fail) return err(r.msg)
      return ok({ ok: true, output: r.out })
    })

    // ── 17. remote（GET → [{name,url}]；POST {action:add|remove|set-url, name, url}）──
    reg('/api/git/remote', (req) => {
      if (req.method === 'GET') {
        const r = gitRun(req, ['remote', '-v'])
        if (r.fail) return ok([])
        const remotes = []
        const seen = {}
        for (const line of r.out.split('\n')) {
          const parts = line.trim().split(/\s+/)
          if (parts.length >= 2) {
            const name = parts[0], url = parts[1]
            if (!seen[name]) { remotes.push({ name, url }); seen[name] = true }
          }
        }
        return ok(remotes)
      }
      if (req.method === 'POST') {
        const b = bodyOf(req)
        switch (b.action) {
          case 'add': {
            if (!b.name || !b.url) return err('name 和 url 不能为空')
            const r = gitRun(req, ['remote', 'add', b.name, b.url])
            if (r.fail) return err(r.msg)
            return ok({ ok: true })
          }
          case 'remove': {
            if (!b.name) return err('name 不能为空')
            const r = gitRun(req, ['remote', 'remove', b.name])
            if (r.fail) return err(r.msg)
            return ok({ ok: true })
          }
          case 'set-url': {
            if (!b.name || !b.url) return err('name 和 url 不能为空')
            const r = gitRun(req, ['remote', 'set-url', b.name, b.url])
            if (r.fail) return err(r.msg)
            return ok({ ok: true })
          }
          default: return err('未知 remote 操作: ' + b.action)
        }
      }
      return err('不支持的方法')
    })

    registerAll()
    if (ctx.logger) ctx.logger('git-api').info('17 条 Git 接口已注册')
    return { dispose: () => {} }
  },
}
