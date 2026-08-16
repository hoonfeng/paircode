// ═══════════════════════════════════════════════════════════════
// tool-web — 网络工具（web_fetch/web_search）
//
// 迁移来源（2026-08-16）：内置 registerWebTools（internal/agent/web.go）
// → 磁盘外置插件。web_fetch 调用实现在插件内（JS 编排 ctx.web.fetch）；
// web_search 走宿主执行器（SearXNG/DuckDuckGo 集成，二进制方案）。
// ═══════════════════════════════════════════════════════════════

// web_fetch：抓取网页转纯文本（JS 实现；ctx.web.fetch 返回 {ok,status,text}）
async function webFetch(ctx, args) {
  const url = String(args.url || '').trim()
  if (!/^https?:\/\//i.test(url)) throw new Error('仅支持 http(s) URL：' + url)
  const res = await ctx.web.fetch(url)
  if (!res.ok) throw new Error('HTTP ' + res.status + '：抓取 ' + url + ' 失败')
  const text = String(res.text || '')
  return text.length > 20000 ? text.slice(0, 20000) + '\n…[输出截断]' : text
}

const tools = [
  {
    name: 'web_fetch',
    description: '抓取一个 http(s) 网页并返回其纯文本内容（去除 HTML 标签，超长截断）。用于查阅在线文档、API 参考、网页。',
    usageGuide: '抓取 http(s) 网页并返回纯文本（去 HTML 标签）。用于查阅在线文档、API 参考、网页内容。拿到链接后再用这个读全文。比 run_command curl 更方便（自动去标签+编码处理+截断保护）。',
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
    usageGuide: '搜索网络，返回标题/链接/摘要。用于查文档、报错信息、库的用法、最新技术方案。拿到链接后可再用 web_fetch 读全文。比 run_command 手动搜索更高效（集成 SearXNG/DuckDuckGo）。',
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
  purpose: '网络工具（web_fetch/web_search）——迁移自内置 registerWebTools；web_fetch JS 实现（ctx.web.fetch）、web_search 统一宿主二进制',
  apply(ctx) {
    for (const t of tools) {
      const isFetch = t.name === 'web_fetch'
      ctx.tools.register({
        name: t.name,
        description: t.description,
        usageGuide: t.usageGuide,
        category: t.category,
        readOnly: t.readOnly,
        parameters: t.parameters,
        execute: (args) => (isFetch ? webFetch(ctx, args || {}) : ctx.binary.exec(t.name, args || {})),
      })
    }
    // 日志已省略（logger 需 inject 声明）
  },
}
