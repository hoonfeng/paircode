// ═══════════════════════════════════════════════════════════════
// tool-web — 网络工具（web_fetch/web_search + 截图/网页验证
// screenshot_desktop/screenshot_window/screenshot_area/web_debug）
//
// 迁移来源（2026-08-16）：内置 registerWebTools（internal/agent/web.go）
// → 磁盘外置插件。★ 2026-08-22 完整 JS 原生化：web_fetch（htmlToText
// 去标签 + 20000 截断）与 web_search（DuckDuckGo HTML 正则解析）实现全在
// 插件内（ctx.web.fetch），不再依赖 bin/tool-web.exe。
// ★ 2026-09 Round3 ③.4 插件瘦身合并：tool-screenshot（desktop/window/area）
// 与 tool-web-debug（web_debug）并入本插件——二者为 binary 型工具
// （execute 经 ctx.binary.exec → 内嵌内核 registerScreenshotTools/
// registerWebDebugTool 回退，插件不再独立存在）。
// ★ 2026-09-12 codex 精简轮：tool-vision（read_image）并入本插件（视觉/媒体
// 同域：联网+截图+看图），实现整体搬迁（__SUBMIT_IMAGE__ 标记协议不变，
// Loop 层 image_submit.go 解析）。
// ═══════════════════════════════════════════════════════════════

// HTML 实体解码（常见命名实体 + &#xHH;/&#DDD; 数字实体）。
function decodeEntities(s) {
  return s
    .replace(/&#x([0-9a-fA-F]+);/g, (m, h) => { try { return String.fromCodePoint(parseInt(h, 16)) } catch { return m } })
    .replace(/&#(\d+);/g, (m, d) => { try { return String.fromCodePoint(Number(d)) } catch { return m } })
    .replace(/&amp;/g, '&')
    .replace(/&lt;/g, '<')
    .replace(/&gt;/g, '>')
    .replace(/&quot;/g, '"')
    .replace(/&#39;/g, "'")
    .replace(/&nbsp;/g, ' ')
}

// htmlToText 把 HTML 粗略转可读纯文本：去 script/style、块标签→换行、去其余标签、解实体、压空白。
function htmlToText(s) {
  s = String(s)
    .replace(/<(script|style)\b[^>]*>[\s\S]*?<\/\1>/gi, ' ')
    .replace(/<\/(p|div|li|tr|h[1-6]|section|article|header|footer|ul|ol|table|blockquote)>|<br\s*\/?>/gi, '\n')
    .replace(/<[^>]+>/g, '')
  s = decodeEntities(s)
  const lines = s.split('\n').map(l => l.replace(/[ \t]{2,}/g, ' ').trim())
  return lines.join('\n').replace(/\n{3,}/g, '\n\n').trim()
}

// ─── 搜索 / 抓取 API 客户端（★ 2026-10-17 API 实例化：不写死服务商与地址）───
// 设计原则：**插件里不出现任何服务商名、默认域名或密钥**。搜索/抓取走哪套 API 完全由设置决定：
//   设置 →「网络工具」→「搜索/抓取 API 实例」选一条 AI 配置（ai-presets.json 里的命名实例），
//   插件按实例名经 ctx.aiPresets 解析 baseURL + apiKey；留空 = 跟随「当前激活配置」。
// 端点路径也可配（默认 /v1/tools/search、/v1/tools/fetch，DeepSeek 网页搜索/抓取协议的形态）——
// 换服务商、换网关、换路径只改设置，插件代码零改动。
// 未配置 Key（或关闭开关）→ 回退 DuckDuckGo / 原生抓取，保证零配置环境下工具不回归。
const DEFAULT_SEARCH_PATH = '/v1/tools/search'
const DEFAULT_FETCH_PATH = '/v1/tools/fetch'

// apiSettings 读本插件命名空间配置（registerSettings 写入的 tool-web 段）。
function apiSettings(ctx) {
  try {
    const s = ctx.getSettings ? ctx.getSettings('tool-web') : null
    return s && typeof s === 'object' ? s : {}
  } catch (e) { return {} }
}

// normBase 归一化基础地址：去尾斜杠、剥 LLM 协议端点后缀（/chat/completions 等）。
// ★ 保留 /v1：搜索路径本身以 /v1 开头，拼接时由 joinAPI 做版本段去重（不会出现 /v1/v1）。
function normBase(u) {
  let s = String(u || '').trim()
  if (!s) return ''
  if (!/^https?:\/\//i.test(s)) return ''
  s = s.replace(/\/+$/, '')
  s = s.replace(/\/chat\/completions$/i, '')
  s = s.replace(/\/completions$/i, '')
  s = s.replace(/\/responses$/i, '')
  s = s.replace(/\/messages$/i, '')
  return s
}

// joinAPI 基础地址 + 端点路径，版本段去重（base 以 /v1 结尾且 path 以 /v1/ 开头时只保留一个）。
function joinAPI(base, path) {
  let b = normBase(base)
  let p = String(path || '').trim()
  if (!p) return ''
  if (!p.startsWith('/')) p = '/' + p
  if (!b) return ''
  const bm = /^(.*)\/v\d+$/i.exec(b)
  const pm = /^\/v\d+\//i.exec(p)
  if (bm && pm) {
    const bver = b.slice(bm[1].length).toLowerCase()
    const pver = pm[0].slice(0, -1).toLowerCase()
    if (bver === pver) b = bm[1]
  }
  return b + p
}

// activePresetName 当前激活的 AI 配置名（settings.preset；读不到 = 空）。
function activePresetName(ctx) {
  try { return String((ctx.app && ctx.app.settings && ctx.app.settings.preset) || '') } catch (e) { return '' }
}

// resolveAPI 按设置解析本次调用使用的 API 实例；返回 null = 未配置 Key / 已关闭。
// 来源优先级（全部实时读取，改设置即时生效）：
//   ① 设置「搜索/抓取 API 实例」指定的 AI 配置（或留空时的当前激活配置）→ baseURL + apiKey
//   ② 实例地址为空 → 该实例所属服务商在 models.json 里的地址（ctx.models）
//   ③ 设置里的「地址 / Key 覆盖」字段（调试或跨服务商复用同一实例时用）
//   ④ 兼容旧设置字段 deepseekApiKey / deepseekBaseURL / deepseekSearch（老用户升级不被静默关闭）
// ★ 不再按「服务商名 / baseURL 含某域名」猜测用哪条配置——实例是显式配置的。
function resolveAPI(ctx) {
  const st = apiSettings(ctx)
  if (st.enabled === false || st.deepseekSearch === false) return null
  const presetName = String(st.apiPreset || '').trim() || activePresetName(ctx)
  let baseURL = String(st.apiBaseURL || st.deepseekBaseURL || '').trim()
  let apiKey = String(st.apiKey || st.deepseekApiKey || '').trim()
  let alias = presetName || '当前激活配置'
  if (presetName) {
    let p = null
    try { p = ctx.aiPresets && typeof ctx.aiPresets.get === 'function' ? ctx.aiPresets.get(presetName) : null } catch (e) { p = null }
    if (p) {
      if (!baseURL) baseURL = String(p.baseURL || '').trim()
      if (!apiKey) apiKey = String(p.apiKey || '').trim()
      // 实例只填了服务商、没填地址 → 取服务商条目里的默认地址（models.json）
      if (!baseURL && p.provider && ctx.models && typeof ctx.models.get === 'function') {
        let m = null
        try { m = ctx.models.get(String(p.provider)) } catch (e) { m = null }
        if (m && m.baseURL) baseURL = String(m.baseURL).trim()
      }
    }
  }
  baseURL = normBase(baseURL)
  if (!apiKey || !baseURL) return null
  return {
    apiKey: apiKey,
    baseURL: baseURL,
    searchPath: String(st.searchPath || '').trim() || DEFAULT_SEARCH_PATH,
    fetchPath: String(st.fetchPath || '').trim() || DEFAULT_FETCH_PATH,
    alias: alias,
  }
}

// apiErrText 把接口失败响应压成一行可读错误（含 HTTP 状态与响应体片段）。
function apiErrText(what, res) {
  const status = res && res.status != null ? res.status : '?'
  const body = String((res && res.text) || '').replace(/\s+/g, ' ').slice(0, 300)
  return '搜索/抓取 API ' + what + '接口返回 HTTP ' + status + (body ? '：' + body : '')
}

// apiFetch 调实例的抓取端点（默认 /v1/tools/fetch）：返回页面标题 + Markdown 正文（20000 截断）。
function apiFetch(ctx, cred, url) {
  const endpoint = joinAPI(cred.baseURL, cred.fetchPath)
  const res = ctx.web.post(endpoint, {
    'Content-Type': 'application/json',
    'Authorization': 'Bearer ' + cred.apiKey,
  }, JSON.stringify({ url: url }))
  if (!res || !res.ok) throw new Error(apiErrText('网页抓取', res) + '（端点 ' + endpoint + '，实例 ' + cred.alias + '）')
  const data = JSON.parse(String(res.text || '{}'))
  const title = String(data.title || '')
  const content = String(data.content || '')
  let out = '（抓取 API 实例：' + cred.alias + '）\nURL: ' + url + '\n'
  if (title) out += '标题：' + title + '\n'
  out += '\n' + content
  if (out.length > 20000) out = out.slice(0, 20000) + '\n…（内容已截断）'
  return out
}

// web_fetch：优先实例的抓取 API（Markdown 正文），失败/未配置回退原生抓取。
function webFetch(ctx, args) {
  const url = String(args.url || '').trim()
  if (!/^https?:\/\//i.test(url)) throw new Error('仅支持 http(s) URL：' + url)
  const cred = resolveAPI(ctx)
  if (cred) {
    try {
      return apiFetch(ctx, cred, url)
    } catch (e) {
      const note = '（抓取 API 调用失败，已回退原生抓取：' + String((e && e.message) || e) + '）\n'
      let res
      try {
        res = ctx.web.fetch(url)
      } catch (e2) {
        // 回退也失败 → 两条错误都告诉用户（只报原生错误会把「其实是 API 地址/Key 配错」藏起来）
        throw new Error(String((e && e.message) || e) + '；原生抓取回退也失败：' + String((e2 && e2.message) || e2))
      }
      const text = htmlToText(res.text || '')
      if (text.length > 20000) return note + 'URL: ' + url + '\nHTTP ' + res.status + '\n\n' + text.slice(0, 20000) + '\n…（内容已截断）'
      return note + 'URL: ' + url + '\nHTTP ' + res.status + '\n\n' + text
    }
  }
  const res = ctx.web.fetch(url)
  const text = htmlToText(res.text || '')
  if (text.length > 20000) return 'URL: ' + url + '\nHTTP ' + res.status + '\n\n' + text.slice(0, 20000) + '\n…（内容已截断）'
  return 'URL: ' + url + '\nHTTP ' + res.status + '\n\n' + text
}

// —— web_search：DuckDuckGo HTML 正则解析（与二进制实现等价，SearXNG 无配置链路）——

// decodeDDGHref 解出 DDG 跳转链接里的真实 URL（//duckduckgo.com/l/?uddg=ENCODED）。
function decodeDDGHref(href) {
  const i = href.indexOf('uddg=')
  if (i >= 0) {
    let v = href.slice(i + 5)
    const j = v.indexOf('&')
    if (j >= 0) v = v.slice(0, j)
    try { return decodeURIComponent(v) } catch { return v }
  }
  return href
}

// stripTags 去标签 + 解实体 + 压空白。
function stripTags(s) {
  return decodeEntities(String(s).replace(/<[^>]+>/g, '')).trim()
}

// parseDDGResults 从 DDG HTML 结果页粗略抽取 标题/链接/摘要（按出现顺序配对）。
function parseDDGResults(body) {
  const anchors = []
  const reA = /<a\b([^>]*class="result__a"[^>]*)>([\s\S]*?)<\/a>/gi
  let m
  while ((m = reA.exec(body)) !== null) anchors.push(m)
  const snips = []
  const reS = /class="result__snippet"[^>]*>([\s\S]*?)<\/a>/gi
  while ((m = reS.exec(body)) !== null) snips.push(m)
  const out = []
  for (let i = 0; i < anchors.length; i++) {
    const attrs = anchors[i][1]
    const hrefM = /href="([^"]+)"/.exec(attrs)
    const href = hrefM ? hrefM[1] : ''
    out.push({
      title: stripTags(anchors[i][2]),
      url: decodeDDGHref(href),
      snippet: i < snips.length ? stripTags(snips[i][1]) : '',
    })
  }
  return out
}

// formatAPIResults 把搜索 API 结果格式化成模型易读的清单（别名=实例名，便于用户判断走的哪套 API）。
function formatAPIResults(q, results, alias) {
  let out = '「' + q + '」搜索结果（API 实例：' + alias + '）：\n'
  results.forEach((r, i) => {
    out += '\n' + (i + 1) + '. ' + (r.title || '(无标题)') + '\n   ' + (r.url || '')
    const meta = []
    if (r.site_name) meta.push('站点：' + r.site_name)
    if (r.date) meta.push('日期：' + r.date)
    if (typeof r.rerank_score === 'number') meta.push('相关度：' + r.rerank_score)
    if (meta.length) out += '\n   ' + meta.join(' · ')
    const text = String(r.summary || r.snippet || '')
    if (text) out += '\n   ' + text.slice(0, 800)
  })
  return out
}

// apiSearch 调实例的搜索端点（默认 /v1/tools/search）：query/count/ori_question → 结构化结果列表。
function apiSearch(ctx, cred, args, q, count) {
  const body = { query: q, count: count, need_content: false }
  // ori_question：调用方给了原始意图就透传（文档建议，有助理解搜索意图）
  const ori = String(args.ori_question || '').trim()
  if (ori) body.ori_question = ori.slice(0, 128)
  const endpoint = joinAPI(cred.baseURL, cred.searchPath)
  const res = ctx.web.post(endpoint, {
    'Content-Type': 'application/json',
    'Authorization': 'Bearer ' + cred.apiKey,
  }, JSON.stringify(body))
  if (!res || !res.ok) throw new Error(apiErrText('网页搜索', res) + '（端点 ' + endpoint + '，实例 ' + cred.alias + '）')
  const data = JSON.parse(String(res.text || '{}'))
  const results = Array.isArray(data.results) ? data.results : []
  if (results.length === 0) return '「' + q + '」无搜索结果（API 实例：' + cred.alias + '）。'
  return formatAPIResults(q, results, cred.alias)
}

// ddgSearch DuckDuckGo 回退（无需 key；限流/改版时返回无结果提示而非崩溃）。
function ddgSearch(ctx, q) {
  const resp = ctx.web.fetch('https://html.duckduckgo.com/html/?q=' + encodeURIComponent(q))
  const body = String(resp.text || '')
  const results = parseDDGResults(body)
  if (results.length === 0) {
    return '「' + q + '」无搜索结果（HTTP ' + resp.status + '；可能被限流或页面改版）。'
  }
  let out = '「' + q + '」搜索结果：\n'
  results.slice(0, 8).forEach((r, i) => {
    out += '\n' + (i + 1) + '. ' + r.title + '\n   ' + r.url
    if (r.snippet) out += '\n   ' + r.snippet
  })
  return out
}

// web_search：优先实例的搜索 API（默认 /v1/tools/search），失败/未配置回退 DuckDuckGo。
function webSearch(ctx, args) {
  const q = String(args.query || '').trim()
  if (!q) throw new Error('query 不能为空')
  const n = Number(args.count)
  const count = n > 0 ? Math.max(1, Math.min(50, Math.floor(n))) : 8
  const cred = resolveAPI(ctx)
  if (cred) {
    try {
      return apiSearch(ctx, cred, args, q, count)
    } catch (e) {
      const note = '（搜索 API 调用失败，已回退 DuckDuckGo：' + String((e && e.message) || e) + '）\n'
      try {
        return note + ddgSearch(ctx, q)
      } catch (e2) {
        // 回退也失败 → 两条错误都告诉用户（只抛 API 错误会掩盖「网络本身不通」）
        throw new Error(String((e && e.message) || e) + '；DuckDuckGo 回退也失败：' + String((e2 && e2.message) || e2))
      }
    }
  }
  return ddgSearch(ctx, q)
}

// ─── read_image（2026-09-12 自 tool-vision 并入）───
// 把图片读给模型看（视觉识别）：结果标记 __SUBMIT_IMAGE__:json 由 Loop 层
// （internal/agent/image_submit.go）解析 → 准入/归一化 → 随下一轮 LLM 请求发送。
const IMG_EXTS = ['png', 'jpg', 'jpeg', 'gif', 'webp']
const MAX_IMAGE_BYTES = 20 * 1024 * 1024 // 对齐 dsh 准入上限（DEFAULT_MAX_REQUEST_IMAGE_BYTES）

function imageMimeOf(ext) {
  return { png: 'image/png', jpg: 'image/jpeg', jpeg: 'image/jpeg', gif: 'image/gif', webp: 'image/webp' }[ext] || 'image/jpeg'
}

// readImage：校验路径/扩展名/大小后生成标记行（图片信封由 Go 层生成）。
function readImage(ctx, args) {
  const pathR = String((args && (args.file_path || args.path)) || '').trim()
  if (!pathR) return 'file_path must be a non-empty string'
  const prompt = String((args && args.prompt) || '').trim()
  // 路径解析：绝对（含盘符或 / 开头）直接用；相对拼工作区根
  let full = pathR
  if (!(full.includes(':') || full.startsWith('/') || full.startsWith('\\'))) {
    full = ((ctx.workspaceRoot || '') + '/' + pathR).replace(/[\/]+/g, '/')
  }
  let st
  try { st = ctx.fs.stat(full) } catch (e) { return '错误：图片不存在或不可访问：' + pathR }
  if (!st || st.isDir) return '错误：file_path 是目录：' + pathR
  const dot = full.lastIndexOf('.')
  const slash = Math.max(full.lastIndexOf('/'), full.lastIndexOf('\\'))
  const ext = dot > slash ? full.slice(dot + 1).toLowerCase() : ''
  // 无扩展名也接受（对齐 dsh：按文件内容识别格式）；有扩展名必须是支持的图片格式
  if (ext && !IMG_EXTS.includes(ext)) {
    return '错误：无法读取 "' + pathR + '"，.' + ext + ' 扩展名不是受支持的图片格式；read_image 接受 PNG/JPEG/WebP/GIF（无扩展名的文件同样按内容识别）'
  }
  if (st.size > MAX_IMAGE_BYTES) {
    return '错误：图片 ' + (st.size / 1048576).toFixed(1) + 'MiB 超过 ' + (MAX_IMAGE_BYTES / 1048576) + 'MiB 准入上限——请缩小尺寸或压缩后重试'
  }
  const mark = JSON.stringify({ kind: 'read_image', path: full, mime: ext ? imageMimeOf(ext) : '', size: st.size, prompt })
  return '__SUBMIT_IMAGE__:' + mark + '\n' + (prompt ? '关注点：' + prompt : '')
}

// screenshotImpl：三合一截屏调度（2026-09-12 合并 screenshot_desktop/window/area）。
// 参数校验在 JS 层，执行直通内嵌内核原名（ctx.binary.exec → 无 exe 时回退
// registerScreenshotTools 的 3 个原名工具）。
function screenshotImpl(ctx, args) {
  const target = String((args && args.target) || '').trim().toLowerCase()
  const name = { desktop: 'screenshot_desktop', window: 'screenshot_window', area: 'screenshot_area' }[target]
  if (!name) return '错误：target 必须是 "desktop" | "window" | "area"（收到：' + target + '）'
  if (target === 'window' && !String((args && args.title) || '').trim()) return '错误：target="window" 需要 title（窗口标题或子串）'
  if (target === 'area') {
    for (const k of ['left', 'top', 'right', 'bottom']) {
      if (args[k] == null || String(args[k]).trim() === '') return '错误：target="area" 需要 ' + k + '（像素或百分比）'
    }
  }
  const na = {}
  for (const k of Object.keys(args || {})) { if (k !== 'target') na[k] = args[k] }
  return ctx.binary.exec(name, na)
}

const tools = [
  {
    name: 'web_fetch',
    description: '抓取一个 http(s) 网页并返回其正文（优先走设置中选定的 API 实例的抓取接口，返回标题 + Markdown 正文；未配置或调用失败时回退原生抓取并去 HTML 标签）。用于查阅在线文档、API 参考、网页。',
    usageGuide: '抓取 http(s) 网页返回正文。设置「网络工具 → 搜索/抓取 API 实例」选定了实例且配置了 Key 时，优先调用该实例的抓取接口（默认路径 /v1/tools/fetch，Markdown 正文、自动处理反爬/编码）；失败或未配置时回退原生抓取（去标签+20000 字截断）。拿到链接后再用这个读全文。',
    category: '网络',
    readOnly: true,
    parameters: {
      type: 'object',
      properties: {
        url: { type: 'string', description: '要抓取的网页 URL（必须 http:// 或 https://）' },
      },
      required: ['url'],
    },
    impl: webFetch,
  },
  {
    name: 'web_search',
    description: '搜索网络，返回前若干条 标题/链接/摘要/站点/日期/相关度（优先走设置中选定的 API 实例的搜索接口，未配置或调用失败时回退 DuckDuckGo）。查文档、报错、库用法、最新信息时用；拿到链接可再用 web_fetch 读全文。',
    usageGuide: '搜索网络，返回标题/链接/摘要/站点/日期/相关度。设置「网络工具 → 搜索/抓取 API 实例」选定实例且配置了 Key 时，走该实例的搜索接口（默认路径 /v1/tools/search，支持 site:、-site:、引号等语法，count 可选 1~50），否则回退 DuckDuckGo。用于查文档、报错信息、库的用法、最新技术方案；拿到链接后再用 web_fetch 读全文。',
    category: '网络',
    readOnly: true,
    parameters: {
      type: 'object',
      properties: {
        query: { type: 'string', description: '搜索关键词' },
        count: { type: 'integer', description: '可选：返回条数（1~50，默认 8；仅搜索 API 生效）' },
        ori_question: { type: 'string', description: '可选：触发搜索的用户原始问题（≤128 字符，帮助理解搜索意图；仅搜索 API 生效）' },
      },
      required: ['query'],
    },
    impl: webSearch,
  },
  // ── binary 型：execute 经 ctx.binary.exec → 内嵌内核回退（2026-09 并入）──
  // ★ 2026-09-12 codex 精简轮：screenshot_desktop/window/area 三工具合并为
  //   单工具 screenshot（target 参数切换形态，对齐 codex「一个工具+参数」）——
  //   内核回退仍按原名执行（ctx.binary.exec('screenshot_'+target)），
  //   registerScreenshotTools 与门控兼容不变。
  {
    name: 'screenshot',
    description: '截屏保存为 PNG 到 screenshots/ 目录（单工具三形态）：target="desktop"=整个桌面（所有显示器）；"window"=按窗口标题截取（需 title）；"area"=按坐标截取区域（需 left/top/right/bottom，支持像素或百分比）。返回文件路径、尺寸和截图时间；之后可用 read_image 把图片交给模型看。',
    usageGuide: '截屏工具（三形态）：screenshot(target="desktop") 截桌面；screenshot(target="window", title="Chrome") 截窗口；screenshot(target="area", left=..., top=..., right=..., bottom=...) 截区域（坐标可为百分比如 "10%"）。截屏后用 read_image 查看效果（验证 UI/桌面/GUI）。',
    category: '视觉',
    readOnly: true,
    parameters: {
      type: 'object',
      properties: {
        target: { type: 'string', description: '截屏形态："desktop"（整个桌面）| "window"（按窗口标题）| "area"（按坐标区域）' },
        title: { type: 'string', description: 'target="window" 必填：窗口标题或子串（不区分大小写），如 "记事本"、"Chrome"' },
        left: { type: 'string', description: 'target="area" 必填：左边界（像素或百分比，如 "10%"）' },
        top: { type: 'string', description: 'target="area" 必填：上边界' },
        right: { type: 'string', description: 'target="area" 必填：右边界' },
        bottom: { type: 'string', description: 'target="area" 必填：下边界' },
        name: { type: 'string', description: '可选：自定义文件名（不含扩展名），默认自动生成时间戳名称' },
      },
      required: ['target'],
    },
    impl: screenshotImpl,
  },
  {
    name: 'web_debug',
    description: '一站式网页验证工具：在无头浏览器中打开 URL，捕获控制台错误/警告、网络请求失败（404/500/CORS）、DOM 结构概览、元素查询（标签/样式/尺寸/可见性/属性）、可选输入文字、点击元素、执行 JS、提取页面可见文字，最后截图保存。用于验证前端改动是否正常工作（白屏、JS 异常、接口报错、样式错乱等）。截图保存到 screenshots/ 目录，返回文件路径可用多模态模型（如 DeepSeek-VL）进一步分析。注意：首次使用会自动下载 Chromium（约 150MB），后续复用缓存。',
    usageGuide: '一站式网页验证工具：在无头浏览器中打开 URL，检查控制台错误+网络请求失败+截图。支持交互操作（click_selector/type_selector+type_text）、JS 求值（eval）、文字提取(text_extract)、元素查询(element_query)。前端改动验证首选工具，比手动打开浏览器检查更全自动化。',
    parameters: {
      properties: {
        click_selector: { description: '可选：页面加载后点击的 CSS 选择器（如 \'#submit-btn\'）', type: 'string' },
        element_query: { description: '可选：CSS 选择器，查询匹配元素的详细信息（标签/类/样式/尺寸/可见性/属性/文本）', type: 'string' },
        eval: { description: '可选：在页面上执行的 JavaScript 表达式（如 \'document.title\' 或 \'JSON.stringify(window.appState)\'）', type: 'string' },
        screenshot: { description: '可选：是否截图（默认 true）。截图保存到 screenshots/ 目录', type: 'boolean' },
        text_extract: { description: '可选：提取页面可见纯文本内容（默认 false，内容过多时自动截断）', type: 'boolean' },
        timeout: { description: '可选：总超时毫秒数（默认 30s；wait 较大或页面较慢时建议显式调大，如 120000）', type: 'integer' },
        type_selector: { description: '可选：要输入文字的 input/textarea 的 CSS 选择器', type: 'string' },
        type_text: { description: '可选：要输入的文字内容（需配合 type_selector）', type: 'string' },
        url: { description: '要验证的网页 URL（如 http://localhost:9090）', type: 'string' },
        viewport_height: { description: '可选：视口高度（默认 900）', type: 'integer' },
        viewport_width: { description: '可选：视口宽度（默认 1280）', type: 'integer' },
        wait: { description: '可选：页面加载后等待毫秒数（默认 2000，给 JS 渲染和异步请求时间）', type: 'integer' },
      },
      required: ['url'],
      type: 'object',
    },
    readOnly: true,
  },
  {
    name: 'read_image',
    description: '读取工作区内的 PNG/JPEG/WebP/GIF 图片文件并把图片本身交给模型看（随下一轮请求一起发送）：识别截图文字、分析界面布局、验证 UI 渲染效果、描述图表。路径可以没有扩展名（按文件内容识别格式），也可以直接传归一化附件路径。Harness 会在下次模型请求前校验并降采样过大的图片，所以直接用本工具即可，不要为了看图片而安装图像库或生成缩略图。独立文件可以小批并发读取。要求当前模型支持图片输入。',
    usageGuide: '截图/测试/网页验证产出图片后，把图片路径交给 LLM 看：read_image(file_path=图片路径, prompt=关注的问题)。适用于：① web_debug/screenshot 截图后要 LLM 确认页面渲染效果；② 测试生成图表要 LLM 分析；③ 图片是问题描述的一部分（LLM 看图排错）。',
    category: '视觉',
    readOnly: true,
    parameters: {
      type: 'object',
      properties: {
        file_path: { type: 'string', description: '图片文件路径（工作区内；相对路径相对主项目根解析，跨项目请传绝对路径），支持 PNG/JPEG/WebP/GIF' },
        prompt: { type: 'string', description: '可选：想让 LLM 关注的问题（默认：描述图片内容、识别文字、分析布局）' },
      },
      required: ['file_path'],
    },
    impl: readImage,
  },
]

return {
  name: 'tool-web',
  inject: ['web', 'fs'], // ★ binary 是 ctx 附属对象（jsplugin.go ctxObj.Set("binary", …)），非宿主服务，不进 inject；fs：read_image 校验用
  purpose: '网络与视觉工具（web_fetch/web_search：按设置选定的 API 实例调用搜索/抓取接口 + DuckDuckGo/原生抓取回退；screenshot/web_debug 内嵌内核回退 + read_image 看图）——并入 tool-screenshot/tool-web-debug（2026-09）与 tool-vision（2026-09-12）；2026-10-17 接入搜索/抓取 API 并改为「API 实例可配」（不内置服务商/地址）',
  apply(ctx) {
    // 插件设置（设置面板「网络工具」页）：搜索/抓取 API 接入参数。
    // ★ 2026-10-17 API 实例化：不在插件里写死服务商/地址/Key——用户选一条 AI 配置（API 实例），
    //   插件按实例名经 ctx.aiPresets 解析；留空 = 跟随当前激活配置（见 resolveAPI）。
    try {
      if (typeof ctx.registerSettings === 'function') {
        ctx.registerSettings({
          key: 'tool-web',
          title: '网络工具',
          fields: [
            {
              name: 'apiPreset',
              label: '搜索/抓取 API 实例',
              type: 'select',
              optionsSource: 'presets',
              default: '',
              hint: '用哪套 AI 配置（API 实例）的地址与 Key 调搜索/抓取接口。留空=跟随「当前激活配置」。本插件不内置任何服务商——换实例只改这里，代码不变',
            },
            {
              name: 'apiBaseURL',
              label: 'API 地址覆盖（可选）',
              type: 'text',
              default: '',
              hint: '留空=用所选实例的地址；填写则覆盖。只填基础地址（如 https://host/v1），/chat/completions 等后缀会被忽略、版本段与端点路径自动去重',
            },
            {
              name: 'apiKey',
              label: 'API Key 覆盖（可选）',
              type: 'password',
              default: '',
              hint: '留空=用所选实例的 Key；仅当搜索/抓取接口用的 Key 与实例不同时才填',
            },
            {
              name: 'searchPath',
              label: '搜索端点路径',
              type: 'text',
              default: DEFAULT_SEARCH_PATH,
              hint: '相对 API 地址的路径，默认 ' + DEFAULT_SEARCH_PATH + '（DeepSeek 网页搜索协议形态）；换服务商/网关时改这里',
            },
            {
              name: 'fetchPath',
              label: '抓取端点路径',
              type: 'text',
              default: DEFAULT_FETCH_PATH,
              hint: '相对 API 地址的路径，默认 ' + DEFAULT_FETCH_PATH + '（DeepSeek 网页抓取协议形态）',
            },
            {
              name: 'enabled',
              label: '启用搜索/抓取 API',
              type: 'checkbox',
              default: true,
              hint: '关闭后 web_search 回退 DuckDuckGo、web_fetch 回退原生抓取',
            },
          ],
        })
      }
    } catch (e) {
      // 设置注册失败不阻断工具装载（老宿主没有 registerSettings 时静默跳过）
      try { ctx.logger && ctx.logger.warn && ctx.logger.warn('tool-web: 设置注册失败：' + String((e && e.message) || e)) } catch (e2) {}
    }
    for (const t of tools) {
      ctx.tools.register({
        name: t.name,
        description: t.description,
        usageGuide: t.usageGuide,
        category: t.category,
        readOnly: t.readOnly,
        parameters: t.parameters,
        // web_fetch/web_search：JS 原生化；screenshot_*/web_debug：binary 型
        //（ctx.binary.exec → 内嵌内核 registerScreenshotTools/registerWebDebugTool）
        execute: (args) => (t.impl ? t.impl(ctx, args || {}) : ctx.binary.exec(t.name, args || {})),
      })
    }
  },
}
