// ═══════════════════════════════════════════════════════════════
// fs-api — 文件系统接口插件化（10 条）
//
// 背景（2026-08-19）：fs.* 接口原实现全部在 Go 内核（web_server.go / handler 包），
// 本插件用 ctx.fs（工作区受限文件服务）+ ctx.bash 补齐能力，经
// ctx.webServer.register 注册同名路径接口（core-api 已删除对应内核 key）。
// ★ 2026-09-09：drives 与 list 的 browse 分支改用 Go 原生能力 ctx.fs.drives /
//   ctx.fs.listDir（os.Stat/os.ReadDir 实现）——此前用 fsutil/PowerShell 命令，
//   部分系统命令不可用/被限制时返回空数组且无报错（添加工作区目录列表为空
//   的根因）；现在失败明确报 400 + 错误消息。
// 说明：fs.image（原始字节流）保留内核（ctx.fs 仅文本），未在本插件实现。
// ═══════════════════════════════════════════════════════════════
return {
  name: 'fs-api',
  purpose: '文件系统接口插件化（10 条：drives/list/read/write/rename/delete/mkdir/search/file-info/hex；drives 与 list-browse 用 Go 原生 ctx.fs 能力）',
  inject: ['fs', 'bash', 'logger'],
  apply(ctx) {
        // query 解析（req.query 是 RawQuery 字符串）
    const qp = (req, key) => {
      const q = req.query || ''
      const m = new RegExp('(?:^|[?&])' + key + '=([^&]*)').exec(q)
      return m ? decodeURIComponent(m[1].replace(/\+/g, ' ')) : ''
    }
const ok = (data) => ({ status: 200, headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(data) })
    const err = (msg) => ({ status: 400, headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ error: String(msg) }) })
    const root = () => ctx.workspaceRoot || (ctx.app && ctx.app.workspaceRoot) || ''
    const q = (s) => '"' + String(s).replace(/"/g, '\\"') + '"'
    const bodyOf = (req) => { try { return req.json ? req.json() : {} } catch (e) { return {} } }

    // ctx.fs 的 path 解析：工作区内路径或绝对路径（越界拦截在内核）
    const fsPath = (p) => p || root()

    // ── drives：A-Z 盘符探测（Go 能力 ctx.fs.drives，os.Stat 原生实现）──
    // ★ 2026-09-09：此前用 fsutil fsinfo drives 命令，受限系统失败返回空。
    const fsDrives = () => {
      try {
        return ok(ctx.fs.drives() || [])
      } catch (e) { return err(String(e && e.message || e)) }
    }

    // ── list：目录列表 [{name,isDir,size,modTime}] ──
    // ★ browse=1：目录浏览器模式（添加工作区/新建项目选目录需要浏览全盘）。
    //   Go 原生能力 ctx.fs.listDir（os.ReadDir 实现，跨平台：Windows 盘根
    //   /Unix 根自动规范化）——此前用 PowerShell 命令，受限系统失败返回空且无报错。
    const fsList = (req) => {
      const p = fsPath(qp(req, 'path') || '')
      if (qp(req, 'browse') === '1') {
        try {
          // ★ 2026-09-09 改用 Go 原生能力 ctx.fs.listDir（os.ReadDir）：
          //   - 不受工作区限制（全盘浏览）；
          //   - 路径规范化（双反斜杠折叠/Windows 盘根/Unix 根）在 Go 侧处理；
          //   - 失败明确报 400 + 错误消息（此前 PowerShell 失败容错返回空数组
          //     且无报错——添加工作区目录列表为空的根因）。
          let dir = String(p || '').trim()
          if (!dir) return ok([])
          const entries = ctx.fs.listDir(dir) || []
          const out = entries.map(x => ({ name: x.name, isDir: !!x.isDir, size: x.size || 0, modTime: String(x.mtime || '').slice(0, 19).replace('T', ' ') }))
          return ok(out)
        } catch (e) { return err('browse 失败: ' + String(e && e.message || e)) }
      }
      try {
        const names = ctx.fs.readdir(p) || []
        const out = []
        for (const name of names) {
          let isDir = false, size = 0, modTime = ''
          try {
            const st = ctx.fs.stat(p.replace(/[\\/]+$/, '') + '/' + name)
            if (st) { isDir = !!st.isDir; size = st.size || 0; modTime = st.mtime || '' }
          } catch (e) {}
          out.push({ name, isDir, size, modTime: String(modTime).slice(0, 19).replace('T', ' ') })
        }
        return ok(out)
      } catch (e) { return err(String(e && e.message || e)) }
    }

    // ── read：{content, size, path} ──
    const fsRead = (req) => {
      const p = fsPath(qp(req, 'path') || '')
      try {
        const content = ctx.fs.readFile(p)
        return ok({ content, size: content ? content.length : 0, path: p })
      } catch (e) { return err(String(e && e.message || e)) }
    }

    // ── write：POST {path, content} ──
    const fsWrite = (req) => {
      if (req.method !== 'POST') return err('仅 POST')
      const b = bodyOf(req)
      if (!b.path) return err('path 必填')
      try { ctx.fs.writeFile(b.path, String(b.content == null ? '' : b.content)) }
      catch (e) { return err(String(e && e.message || e)) }
      return ok({ ok: true, path: b.path })
    }

    // ── rename：POST {from, to}（bash mv 或 fs 能力）──
    const fsRename = (req) => {
      if (req.method !== 'POST') return err('仅 POST')
      const b = bodyOf(req)
      if (!b.from || !b.to) return err('from/to 必填')
      try {
        // ctx.fs 无 rename，用 bash（工作区内）
        const r = ctx.bash.exec('mv -f ' + q(b.from) + ' ' + q(b.to))
        if (r && r.error) {
          // move 失败兜底 copy+del？直接报错
          return err(r.error)
        }
      } catch (e) { return err(String(e && e.message || e)) }
      return ok({ ok: true, from: b.from, to: b.to })
    }

    // ── delete：POST {path, recursive} ──
    const fsDelete = (req) => {
      if (req.method !== 'POST') return err('仅 POST')
      const b = bodyOf(req)
      if (!b.path) return err('path 必填')
      try { ctx.fs.rm(b.path, !!b.recursive) }
      catch (e) { return err(String(e && e.message || e)) }
      return ok({ ok: true, path: b.path })
    }

    // ── mkdir：POST {path} ──
    const fsMkdir = (req) => {
      if (req.method !== 'POST') return err('仅 POST')
      const b = bodyOf(req)
      if (!b.path) return err('path 必填')
      try { ctx.fs.mkdir(b.path, true) }
      catch (e) { return err(String(e && e.message || e)) }
      return ok({ ok: true, path: b.path })
    }

    // ── search：GET ?q=&path= → [{file,line,text}]（复刻内核逻辑：跳过目录/文本扩展名/内容匹配）──
    const SKIP = { '.git': 1, 'node_modules': 1, 'vendor': 1, '.pair': 1, '.trae': 1, '.dbg': 1, '.context': 1, '__pycache__': 1, '.venv': 1, 'venv': 1, 'bin': 1, 'obj': 1, '.vs': 1 }
    const TEXTS = {}
    for (const e of ['.go','.js','.ts','.vue','.html','.css','.scss','.json','.md','.yml','.yaml','.xml','.py','.java','.rs','.c','.h','.cpp','.hpp','.sh','.bat','.ps1','.env','.gitignore','.dockerfile','.sql','.rb','.php','.swift','.kt','.toml','.ini','.cfg','.conf','.txt','.log','.csv','.tsv','.svg','.svelte','.astro','.gradle','.cmake','.lua','.pl','.pm','.r','.dart','.scala','.zig','.nim','.hbs','.ejs']) TEXTS[e] = 1
    const fsSearch = (req) => {
      const query = qp(req, 'q') || ''
      if (!query) return err('q 参数必填')
      const searchPath = qp(req, 'path') || root()
      const results = []
      const seen = {}
      const MAX_RESULTS = 200
      const MAX_TEXT = 10 * 1024 * 1024
      const MAX_UNKNOWN = 512 * 1024
      try {
        const walk = (dir) => {
          let entries = []
          try { entries = ctx.fs.readdir(dir) || [] } catch (e) { return }
          for (const name of entries) {
            if (results.length >= MAX_RESULTS) return
            const full = dir.replace(/[\\/]+$/, '') + '/' + name
            let st = null
            try { st = ctx.fs.stat(full) } catch (e) {}
            if (!st) continue
            if (st.isDir) {
              if (SKIP[name.toLowerCase()]) continue
              walk(full)
              continue
            }
            const ext = (name.includes('.')) ? '.' + name.split('.').pop().toLowerCase() : ''
            const sizeLimit = TEXTS[ext] ? MAX_TEXT : MAX_UNKNOWN
            if (st.size > sizeLimit) continue
            let content = ''
            try { content = ctx.fs.readFile(full) } catch (e) { continue }
            if (!content) continue
            const lines = content.split('\n')
            for (let i = 0; i < lines.length; i++) {
              if (lines[i].includes(query)) {
                if (!seen[full]) {
                  seen[full] = true
                  results.push({ file: full, line: i + 1, text: lines[i].trim() })
                  if (results.length >= MAX_RESULTS) return
                }
                break
              }
            }
          }
        }
        walk(searchPath)
      } catch (e) { return err(String(e && e.message || e)) }
      return ok(results)
    }

    // ── file-info：GET ?path= → {type,size,isImage,isBinary,mimeType} ──
    const fsFileInfo = (req) => {
      const p = fsPath(qp(req, 'path') || '')
      let st = null
      try { st = ctx.fs.stat(p) } catch (e) { return err('文件不存在或无法访问') }
      if (!st) return err('文件不存在或无法访问')
      if (st.isDir) return ok({ type: 'directory', size: st.size || 0 })
      if (st.size > 100 * 1024 * 1024) return err('文件超过 100MB')
      // 简化：按扩展名粗判 mime；二进制探测用 bash findstr 或读取内容（仅文本可读）
      const ext = (p.includes('.')) ? '.' + p.split('.').pop().toLowerCase() : ''
      const IMG = { '.png': 'image/png', '.jpg': 'image/jpeg', '.jpeg': 'image/jpeg', '.gif': 'image/gif', '.svg': 'image/svg+xml', '.webp': 'image/webp', '.bmp': 'image/bmp', '.ico': 'image/x-icon' }
      const mime = IMG[ext] || ''
      const isImage = !!mime
      let isBinary = false
      try {
        const head = ctx.fs.readFile(p).slice(0, 1024)
        isBinary = /[\x00-\x08\x0e-\x1f]/.test(head)
      } catch (e) { isBinary = true }
      const fileType = isImage ? 'image' : (isBinary ? 'binary' : 'text')
      return ok({ type: fileType, size: st.size || 0, isImage, isBinary, mimeType: mime })
    }

    // ── hex：GET ?path=&offset=&length= → {hex,offset,fileSize,hasMore}（bash certutil 转 hex 后切片）──
    const fsHex = (req) => {
      const p = fsPath(qp(req, 'path') || '')
      let st = null
      try { st = ctx.fs.stat(p) } catch (e) { return err('文件不存在') }
      if (!st) return err('文件不存在')
      const fileSize = st.size || 0
      if (fileSize > 100 * 1024 * 1024) return err('文件超过 100MB')
      let offset = parseInt(qp(req, 'offset') || '0') || 0
      let length = parseInt(qp(req, 'length') || '256') || 256
      if (length > 4096) length = 4096
      if (offset > fileSize) offset = fileSize
      if (offset + length > fileSize) length = fileSize - offset
      if (length <= 0) return ok({ hex: '', offset, fileSize, hasMore: false })
      // 用 bash 读字节（certutil -encodehex 是唯一保字节的 cmd 工具，但慢；改用 powershell 读取）
      // 简化：try ctx.fs.readFile 文本 + 编码 hex（二进制会丢字节——非文本文件降级）
      try {
        const content = ctx.fs.readFile(p)
        const bytes = []
        for (let i = offset; i < offset + length && i < content.length; i++) {
          bytes.push(content.charCodeAt(i) & 0xFF)
        }
        // 构造 hex 行（16 字节/行，地址+hex+ascii）
        const lines = []
        const bpl = 16
        for (let i = 0; i < bytes.length; i += bpl) {
          const chunk = bytes.slice(i, i + bpl)
          const addr = (offset + i).toString(16).toUpperCase().padStart(8, '0')
          let hexPart = ''
          for (let j = 0; j < chunk.length; j++) {
            if (j > 0 && j % 8 === 0) hexPart += ' '
            hexPart += chunk[j].toString(16).toUpperCase().padStart(2, '0') + ' '
          }
          let ascii = chunk.map(b => (b >= 32 && b <= 126) ? String.fromCharCode(b) : '.').join('')
          lines.push(addr + '  ' + hexPart + ' ' + ascii)
        }
        return ok({ hex: lines.join('\n'), offset, fileSize, hasMore: offset + length < fileSize })
      } catch (e) { return err(String(e && e.message || e)) }
    }

    // ── 注册 ──
    const routes = [
      ['/api/fs/drives', fsDrives],
      ['/api/fs/list', fsList],
      ['/api/fs/read', fsRead],
      ['/api/fs/write', fsWrite],
      ['/api/fs/rename', fsRename],
      ['/api/fs/delete', fsDelete],
      ['/api/fs/mkdir', fsMkdir],
      ['/api/fs/search', fsSearch],
      ['/api/fs/file-info', fsFileInfo],
      ['/api/fs/hex', fsHex],
    ]
    for (const [path, handler] of routes) {
      try {
        ctx.webServer.register({ kind: 'exact', path, handler: (req, res) => {
          const out = handler(req)
          if (out === undefined) return
          res.writeHead(out.status, out.headers)
          res.end(out.body)
        }})
      } catch (e) {
        if (ctx.logger) ctx.logger('fs-api').warn('注册失败 ' + path + ': ' + e)
      }
    }
    if (ctx.logger) ctx.logger('fs-api').info('10 条 FS 接口已注册（image 留内核）')
    return { dispose: () => {} }
  },
}
