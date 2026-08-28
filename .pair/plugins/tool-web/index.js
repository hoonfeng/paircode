// ═══════════════════════════════════════════════════════════════
// tool-web — 网络工具（web_fetch/web_search）
//
// 迁移来源（2026-08-16）：内置 registerWebTools（internal/agent/web.go）
// → 磁盘外置插件。★ 2026-08-22 完整 JS 原生化：web_fetch（htmlToText
// 去标签 + 20000 截断）与 web_search（DuckDuckGo HTML 正则解析）实现全在
// 插件内（ctx.web.fetch），不再依赖 bin/tool-web.exe。
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
  },
]

return {
  name: 'tool-web',
  inject: ['web'],
  purpose: '网络工具（web_fetch/web_search）——2026-08-22 完整 JS 原生化：htmlToText/DDG 正则解析全部在插件内（ctx.web.fetch），不再依赖独立二进制',
  apply(ctx) {
    for (const t of tools) {
      ctx.tools.register({
        name: t.name,
        description: t.description,
        usageGuide: t.usageGuide,
        category: t.category,
        readOnly: t.readOnly,
        parameters: t.parameters,
        execute: (args) => (t.name === 'web_fetch' ? webFetch(ctx, args || {}) : webSearch(ctx, args || {})),
      })
    }
  },
}
