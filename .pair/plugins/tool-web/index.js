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
  {
    name: 'screenshot_desktop',
    description: '截取整个桌面（所有显示器），保存为 PNG 图片到 screenshots/ 目录。返回文件路径、尺寸和截图时间。之后可用多模态模型（如 DeepSeek-VL）直接分析截图内容。',
    usageGuide: '截取整个桌面（所有显示器），保存为 PNG。用于查看当前桌面状态、验证 GUI 效果。比手动按 PrintScreen 更方便（自动保存到 screenshots/ + 文件名管理）。',
    parameters: {
      properties: {
        name: { description: '可选：自定义文件名（不含扩展名），默认自动生成时间戳名称', type: 'string' },
      },
      type: 'object',
    },
    readOnly: true,
  },
  {
    name: 'screenshot_window',
    description: '按窗口标题或标题子串截取特定窗口，保存为 PNG 图片到 screenshots/ 目录。返回文件路径、窗口尺寸和截图时间。如果多个窗口匹配同一标题子串，会列出所有匹配窗口供选择。',
    usageGuide: '按窗口标题截取特定窗口，保存为 PNG。比截图整个桌面更精确（只截目标窗口）。title 支持子串匹配不区分大小写。',
    parameters: {
      properties: {
        name: { description: '可选：自定义文件名（不含扩展名），默认自动生成', type: 'string' },
        title: { description: '窗口标题或标题子串（不区分大小写）。例如 "记事本"、"Chrome"、"Calculator"', type: 'string' },
      },
      required: ['title'],
      type: 'object',
    },
    readOnly: true,
  },
  {
    name: 'screenshot_area',
    description: '按坐标截取指定区域，保存为 PNG 图片到 screenshots/ 目录。区域坐标可以是绝对坐标（相对于桌面左上角），也可以是百分比（如 "10% 20% 50% 30%"）。返回文件路径、区域尺寸和截图时间。',
    usageGuide: '按坐标截取指定屏幕区域。left/top/right/bottom 支持像素或百分比（如 10%）。用于截取界面局部细节。',
    parameters: {
      properties: {
        bottom: { description: '下边界：像素值或百分比', type: 'string' },
        left: { description: '左边界：像素值或百分比（如 "10%"）', type: 'string' },
        name: { description: '可选：自定义文件名', type: 'string' },
        right: { description: '右边界：像素值或百分比', type: 'string' },
        top: { description: '上边界：像素值或百分比', type: 'string' },
      },
      required: ['left', 'top', 'right', 'bottom'],
      type: 'object',
    },
    readOnly: true,
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
]

return {
  name: 'tool-web',
  inject: ['web'], // ★ binary 是 ctx 附属对象（jsplugin.go ctxObj.Set("binary", …)），非宿主服务，不进 inject
  purpose: '网络工具（web_fetch/web_search JS 原生 + screenshot_desktop/window/area/web_debug 内嵌内核回退）——2026-09 并入 tool-screenshot/tool-web-debug',
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
