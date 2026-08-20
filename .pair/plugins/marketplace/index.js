// ═══════════════════════════════════════════════════════════════
// marketplace — 市场插件（市场功能全插件化：Go 内核零实现）
//
// 背景（2026-08-20）：市场功能原实现在 Go 内核（web_server.go 5 个 handler
// + kernel_register.go 5 条路由 + internal/ui/marketplace 包 + agent 市场 4 文件）。
// 现全部搬进本插件：
//   - host 半：ctx.webServer.register 挂 5 条 /api/marketplace/* 接口
//     （search/install/uninstall/sources/refresh），ctx.web.fetch 实时搜索
//     npm/GitHub，安装/卸载编排分派宿主通用能力 ctx.mcp / ctx.skill /
//     ctx.toolset / ctx.npm（Go 只留通用平台能力，无市场业务）；
//   - client 半：注入 assets/marketplace-panel.js（Vite 编译的市场面板，
//     UI 与接口同插件）；
//   - 市场源：ctx.market.register 三市场（skill/mcp/plugin）——替代旧的
//     market-skill / market-mcp / market-plugin 三个声明插件；
//   - agent 工具：ctx.tools.register marketplace_search / marketplace_install
//     （LLM 可用，替代 Go 内置同名工具）。
//   - 生命周期：停用/删除本插件 = 市场接口、UI、工具、市场源全部消失。
// ═══════════════════════════════════════════════════════════════
return {
  name: 'marketplace',
  purpose: '市场（全插件化）：npm/GitHub 实时搜索 + 安装/卸载 MCP/技能/工具集/npm 插件 + 市场面板 UI + agent 市场工具',
  inject: ['web', 'fs', 'market', 'tools', 'npm', 'mcp', 'skill', 'toolset', 'logger'],
  apply(ctx) {
    const log = (msg) => { if (ctx.logger) ctx.logger('marketplace').log(msg) }

    // ── 搜索实现（ctx.web.fetch，同步返回 {ok, status, text}）──
    const NPM_SEARCH = 'https://registry.npmjs.org/-/v1/search?text='
    const MAX = 20
    const shortName = (n) => { const p = String(n).split('/'); return p.length > 1 ? p[1] : p[0] }

    // npm registry 搜索 MCP 服务器（npx 启动）
    function searchNpmMCP(query) {
      const r = ctx.web.fetch(NPM_SEARCH + encodeURIComponent(query) + '&size=' + MAX)
      if (!r || !r.ok) return []
      let data = {}
      try { data = JSON.parse(r.text || '{}') } catch (e) { return [] }
      const out = []
      for (const obj of (data.objects || [])) {
        const p = obj.package || {}
        out.push({
          id: 'npm-' + p.name, kind: 'mcp', name: shortName(p.name),
          description: p.description || '',
          tags: ['mcp'].concat(p.keywords || []),
          source: (p.links && p.links.npm) || '',
          command: 'npx', args: ['-y', p.name],
        })
      }
      return out
    }

    // GitHub 仓库搜索 → skill 条目（按 stars 排序）
    function searchGitHubSkills(query) {
      const r = ctx.web.fetch('https://api.github.com/search/repositories?q=' + encodeURIComponent(query) + '&sort=stars&per_page=' + MAX)
      if (!r || !r.ok) return []
      let data = {}
      try { data = JSON.parse(r.text || '{}') } catch (e) { return [] }
      const out = []
      for (const it of (data.items || [])) {
        const tags = ['skill'].concat(it.topics || [])
        if (it.language) tags.push(it.language)
        out.push({
          id: 'gh-' + it.full_name, kind: 'skill', name: shortName(it.full_name),
          description: it.description || '', tags,
          source: it.html_url || '',
        })
      }
      return out
    }

    // npm PairCode 插件搜索（★ 官方命名约定精确匹配，2026-08-20 重名唯一化）：
    //   ① @paircode/<name> scope 包（唯一权威）
    //   ② paircode-plugin-<name> 裸名包（须带 paircode 关键词双重校验）
    // 无关包（paircode-terminal 等）被排除。
    function searchNpmPlugins(query) {
      const q = (String(query || '').trim() + ' paircode').trim()
      const r = ctx.web.fetch(NPM_SEARCH + encodeURIComponent(q) + '&size=' + MAX)
      if (!r || !r.ok) return []
      let data = {}
      try { data = JSON.parse(r.text || '{}') } catch (e) { return [] }
      const out = []
      for (const obj of (data.objects || [])) {
        const p = obj.package || {}
        const low = String(p.name || '').toLowerCase()
        if (low === 'cordis' || low === '@cordisjs/core' || low === '@cordisjs/plugin-loader') continue
        const isOfficial = low.startsWith('@paircode/') || low.startsWith('paircode-plugin-')
        if (!isOfficial) continue
        if (low.startsWith('paircode-plugin-')) {
          const kws = String(p.keywords || []).join(',').toLowerCase()
          if (!kws.includes('paircode')) continue
        }
        out.push({
          id: p.name, kind: 'plugin', name: shortName(p.name),
          description: p.description || '',
          tags: ['plugin'].concat(p.keywords || []),
          source: 'npm:' + p.name, version: p.version || '',
        })
      }
      return out
    }

    // ── 搜索结果瞬态缓存（agent 工具 marketplace_install <id> 用）──
    const cache = new Map() // id -> entry

    function searchAll(query, kind) {
      const q = String(query || '').trim()
      if (!q) return []
      const isAll = !kind || kind === 'all'
      const tasks = []
      if (isAll || kind === 'mcp') tasks.push(['mcp', searchNpmMCP])
      if (isAll || kind === 'skill') tasks.push(['skill', searchGitHubSkills])
      if (isAll || kind === 'plugin') tasks.push(['plugin', searchNpmPlugins])
      const out = []
      const seen = new Set()
      for (const [k, fn] of tasks) {
        for (const e of fn(q)) {
          if (!seen.has(e.id)) { seen.add(e.id); out.push(e) }
        }
      }
      cache.clear()
      for (const e of out) cache.set(e.id, e)
      return out
    }

    // ── 已安装检查 ──
    function isInstalled(e) {
      const id = e.id
      if (e.kind === 'mcp') return (ctx.mcp.list() || []).some(x => x.name === id)
      if (e.kind === 'skill') return (ctx.skill.list() || []).some(x => x.name === id)
      if (e.kind === 'plugin') {
        if (String(e.source || '').startsWith('npm:')) return ctx.npm.installed(id)
        const name = String(id).replace(/^plugin-/, '')
        return (ctx.toolset.list() || []).some(x => x.name === name)
      }
      return false
    }

    // ── 安装编排 ──
    function doInstall(body) {
      const id = String(body.id || '').trim()
      if (!id) throw new Error('id 必填')
      const kind = body.kind || ''
      const scope = body.scope === 'project' ? 'project' : 'user'
      const source = String(body.source || '')

      // npm 插件（source=npm:<pkg> 或搜索结果带 source）→ ctx.npm
      if (kind === 'plugin' && (source.startsWith('npm:') || (body.command && source))) {
        let pkg = source.startsWith('npm:') ? source.slice(4) : id
        // 剥离版本后缀（scoped 包 @scope/pkg@ver 的版本在最后一个 @ 后）
        const at = pkg.lastIndexOf('@')
        if (at > 0) pkg = pkg.slice(0, at)
        return ctx.npm.install(pkg || id)
      }
      if (kind === 'plugin') {
        // 工具集：Content = toolset 发布 JSON；否则按 id 查缓存/已有工具集
        if (body.content) {
          const pub = JSON.parse(body.content)
          const ts = (pub && pub.toolset) || {}
          if (!ts.name || !Array.isArray(ts.plugins) || ts.plugins.length === 0) throw new Error('插件条目缺 name/plugins')
          const msg = ctx.toolset.save({ name: ts.name, description: ts.description || '', plugins: ts.plugins, scope })
          return '✅ 已安装插件工具集「' + ts.name + '」（工作区，' + ts.plugins.length + ' 个插件）'
        }
        const cached = cache.get(id) || (body.command ? null : null)
        if (cached && cached.content) return doInstall(cached)
        const name = String(id).replace(/^plugin-/, '')
        const msg = ctx.toolset.save({ name, description: '', plugins: [], scope })
        return msg
      }
      if (kind === 'mcp') {
        const name = id
        const cmd = body.command || 'npx'
        const args = Array.isArray(body.args) ? body.args : []
        ctx.mcp.upsert({ name, command: cmd, args, level: scope })
        const lv = scope === 'project' ? '工作区级' : '用户级（全局）'
        return '✅ 已安装 MCP 服务器「' + (body.name || name) + '」（' + lv + '）'
      }
      if (kind === 'skill') {
        ctx.skill.write({ name: id, description: body.description || '', mode: body.activation || 'auto', content: body.content || '' })
        return '✅ 已安装技能「' + (body.name || id) + '」（工作区级 .pair/skills）'
      }
      throw new Error('未知条目类型: ' + kind)
    }

    // ── 卸载编排 ──
    function doUninstall(body) {
      const id = String(body.id || '').trim()
      if (!id) throw new Error('id 必填')
      const kind = body.kind || ''
      const source = String(body.source || '')
      if (source.startsWith('npm:')) {
        const pkg = source.slice(4)
        ctx.npm.uninstall(pkg)
        return '已卸载 npm 插件 ' + pkg
      }
      if (kind === 'mcp') {
        ctx.mcp.remove(id, 'user')
        ctx.mcp.remove(id, 'project')
        return '已卸载 MCP 服务器 ' + id
      }
      if (kind === 'skill') return ctx.skill.remove(id)
      if (kind === 'plugin') {
        const name = String(id).replace(/^plugin-/, '')
        return ctx.toolset.remove(name)
      }
      throw new Error('未知类型 ' + kind)
    }

    // ── HTTP 接口（ctx.webServer.register，Node 风格 handler）──
    const qp = (req, key) => {
      const q = req.query || ''
      const m = new RegExp('(?:^|[?&])' + key + '=([^&]*)').exec(q)
      return m ? decodeURIComponent(m[1].replace(/\+/g, ' ')) : ''
    }
    const ok = (data) => ({ status: 200, headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(data) })
    const err = (msg) => ({ status: 400, headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ error: String(msg) }) })
    const bodyOf = (req) => { try { return req.json ? req.json() : {} } catch (e) { return {} } }

    const R = []
    const reg = (path, handler) => R.push([path, handler])
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
          log('注册失败 ' + path + ': ' + e)
        }
      }
    }

    // 1. search（GET ?q= 或 ?query=&kind=）→ 搜索结果（含 installed 状态）
    reg('/api/marketplace/search', (req) => {
      if (req.method !== 'GET') return err('仅 GET')
      const query = qp(req, 'q') || qp(req, 'query')
      const kind = qp(req, 'kind') || '' // 空=全部市场
      const results = searchAll(query, kind)
      return ok(results.map(e => ({ ...e, installed: isInstalled(e) })))
    })

    // 2. install（POST {id, kind, command, args, scope, source, description}）
    reg('/api/marketplace/install', (req) => {
      if (req.method !== 'POST') return err('仅 POST')
      try {
        const msg = doInstall(bodyOf(req))
        return ok({ ok: true, message: msg })
      } catch (e) {
        return err(e && e.message || String(e))
      }
    })

    // 3. uninstall（POST {id, kind, source}）
    reg('/api/marketplace/uninstall', (req) => {
      if (req.method !== 'POST') return err('仅 POST')
      try {
        const msg = doUninstall(bodyOf(req))
        return ok({ ok: true, message: msg })
      } catch (e) {
        return err(e && e.message || String(e))
      }
    })

    // 4. refresh（POST）——实时搜索模式，无需拉取远程注册表
    reg('/api/marketplace/refresh', (req) => {
      if (req.method !== 'POST') return err('仅 POST')
      return ok({ ok: true, message: '市场为实时搜索模式（npm/GitHub），无需刷新' })
    })

    // 5. sources（GET）——已注册市场源（前端动态 tab）
    reg('/api/marketplace/sources', (req) => {
      if (req.method !== 'GET') return err('仅 GET')
      return ok(ctx.market.list())
    })

    registerAll()

    // ── 市场源注册（替代 market-skill/market-mcp/market-plugin 三插件）──
    ctx.market.register({ kind: 'mcp', source: 'npm', name: 'MCP', desc: 'npm registry 搜索 MCP 服务器（npx 启动，安装到 MCP 配置）' })
    ctx.market.register({ kind: 'plugin', source: 'npm-paircode', name: '插件', desc: 'npm PairCode 插件（自己的生态，借 npm 分发）：goja 沙箱或 Node 运行时桥安装' })
    ctx.market.register({ kind: 'skill', source: 'github', name: '技能', desc: 'GitHub 仓库搜索 → 技能条目（安装到工作区 .pair/skills）' })

    // ── agent 工具（LLM 可用，替代 Go 内置 marketplace_search/install）──
    const mObjSchema = (props, required) => ({ type: 'object', properties: props, required: Array.isArray(required) ? required : [required] })
    const mStrProp = (desc) => ({ type: 'string', description: desc })

    ctx.tools.register({
      name: 'marketplace_search',
      description: '在市场检索可安装的 MCP 服务器、技能与插件（★ 无预设数据——必须给 query 关键词，实时远程搜索 npm/GitHub 返回结果）。',
      readOnly: true,
      parameters: mObjSchema({ query: mStrProp('关键词（必填；无关键词返回空）'), kind: mStrProp('mcp/skill/plugin/all') }, 'query'),
      execute: async (_ctx, args) => {
        const query = String((args && args.query) || '').trim()
        const kind = (args && args.kind) || 'all'
        const results = searchAll(query, kind)
        if (results.length === 0) {
          if (!query) return '市场无预设数据——请提供搜索关键词（如「github」）实时检索 npm/GitHub。'
          return '未找到匹配的市场条目（远程实时搜索）。用 marketplace_install <id> 安装。'
        }
        let b = ''
        for (const e of results) {
          const inst = isInstalled(e) ? ' [已安装]' : ''
          b += '- [' + e.kind + '] ' + e.name + '（' + e.id + '）：' + (e.description || '') + inst + '\n'
        }
        return b + '\n共 ' + results.length + ' 个条目。用 marketplace_install <id> 安装。'
      },
    })
    ctx.tools.register({
      name: 'marketplace_install',
      description: '从市场按 id 安装一个 MCP、技能、工具集或 npm 插件。scope 可选 user/project。',
      parameters: mObjSchema({ id: mStrProp('条目 id'), scope: mStrProp('user/project') }, 'id'),
      execute: async (_ctx, args) => {
        const id = String((args && args.id) || '').trim()
        if (!id) return 'id 必填'
        const scope = (args && args.scope) === 'project' ? 'project' : 'user'
        const cached = cache.get(id)
        if (cached) {
          return doInstall({ ...cached, scope })
        }
        // 不在缓存：尝试按 id 规则分派（mcp/skill/plugin-<toolset>）
        if (id.startsWith('npm-')) return doInstall({ id, kind: 'mcp', command: 'npx', args: ['-y', id.slice(4)], scope })
        if (id.startsWith('gh-')) return doInstall({ id: id.slice(3), kind: 'skill', name: id.slice(3), scope })
        if (id.startsWith('plugin-')) return doInstall({ id, kind: 'plugin', scope })
        throw new Error('未找到市场条目 ' + id + '（请先 marketplace_search 搜索）')
      },
    })

    log('市场已挂载：5 接口 / 3 市场源 / 2 工具 / 面板 UI（client 半）')
  },
}
