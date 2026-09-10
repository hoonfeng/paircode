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

// web_fetch：抓取网页转纯文本（ctx.web.fetch 60s/4MB；20000 截断）
function webFetch(ctx, args) {
  const url = String(args.url || '').trim()
  if (!/^https?:\/\//i.test(url)) throw new Error('仅支持 http(s) URL：' + url)
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

// web_search：DuckDuckGo（无需 key；限流/改版时返回无结果提示而非崩溃）。
function webSearch(ctx, args) {
  const q = String(args.query || '').trim()
  if (!q) throw new Error('query 不能为空')
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
    description: '抓取一个 http(s) 网页并返回其纯文本内容（去除 HTML 标签，超长截断）。用于查阅在线文档、API 参考、网页。',
    usageGuide: '抓取 http(s) 网页并返回纯文本（去 HTML 标签）。用于查阅在线文档、API 参考、网页内容。拿到链接后再用这个读全文。比 bash curl 更方便（自动去标签+编码处理+截断保护）。',
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
    description: '搜索网络，返回前若干条 标题/链接/摘要（已配置 SearXNG 则优先用之，否则 DuckDuckGo）。查文档、报错、库用法、最新信息时用；拿到链接可再用 web_fetch 读全文。',
    usageGuide: '搜索网络，返回标题/链接/摘要。用于查文档、报错信息、库的用法、最新技术方案。拿到链接后可再用 web_fetch 读全文。比 bash 手动搜索更高效（集成 SearXNG/DuckDuckGo）。',
    category: '网络',
    readOnly: true,
    parameters: {
      type: 'object',
      properties: {
        query: { type: 'string', description: '搜索关键词' },
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
        file_path: { type: 'string', description: '图片文件路径（工作区内绝对路径或相对路径），支持 PNG/JPEG/WebP/GIF' },
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
  purpose: '网络与视觉工具（web_fetch/web_search JS 原生 + screenshot_desktop/window/area/web_debug 内嵌内核回退 + read_image 看图）——并入 tool-screenshot/tool-web-debug（2026-09）与 tool-vision（2026-09-12）',
  apply(ctx) {
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
