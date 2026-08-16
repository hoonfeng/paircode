// ═══════════════════════════════════════════════════════════════
// web-api — HTTP 接口插件化落地（ctx.http.register 首个磁盘插件用例）
//
// 背景（2026-08-16）：宿主已提供 HTTP 接口插件化扩展点（ctx.http.register →
// internal/agent/ext_routes.go ExtRouteMiddleware，插件路由在宿主 mux 之前拦截），
// 但一直无磁盘插件使用。本插件把该能力落到磁盘：注册 /api/ext/* 扩展路由，
// 证明「插件可以扩展 HTTP 接口」链路全通（浏览器/外部工具可直接 curl 消费）。
//
// 路由清单（6 条，全部同步实现——ctx.fs/ctx.web 方法同步返回，无需 await）：
//   GET /api/ext/status   宿主/插件状态（工作区根、工具数、服务名、时间）
//   GET /api/ext/fetch    web fetch 同源代理（?url=…，解决前端跨域）
//   GET /api/ext/fs/read  受限读文件（?path=相对工作区根，越界拦截）
//   GET /api/ext/fs/exists 文件存在检查
//   GET /api/ext/fs/list  受限目录列表
//   GET /api/ext/routes   本插件注册的路由清单（自描述）
//
// 设计边界：内置 /api/* 核心 API（chat/conversations/plugins/settings…）与
// Go 内核深度耦合（会话/agent/存储），保持宿主实现（框架协议）；插件扩展点
// 面向「新增接口」——如本插件的代理/受限访问类 API，前端与外部工具可复用。
// ═══════════════════════════════════════════════════════════════

function parseQuery(qs) {
  const out = {};
  if (!qs) return out;
  for (const part of String(qs).split('&')) {
    if (!part) continue;
    const i = part.indexOf('=');
    const k = i < 0 ? part : part.slice(0, i);
    const v = i < 0 ? '' : part.slice(i + 1);
    try { out[decodeURIComponent(k)] = decodeURIComponent(v); } catch (e) { out[k] = v; }
  }
  return out;
}

function json(status, obj) {
  return {
    status,
    body: JSON.stringify(obj),
    headers: { 'Content-Type': 'application/json; charset=utf-8' },
  };
}

return {
  name: 'web-api',
  purpose: 'HTTP 接口插件化落地（ctx.http.register 首个磁盘用例）——注册 /api/ext/* 扩展路由：宿主状态、web fetch 同源代理、受限文件访问、路由清单',
  inject: ['fs', 'web', 'tools', 'logger'],
  apply(ctx) {
    const root = (ctx.app && ctx.app.workspaceRoot) || ctx.workspaceRoot || '';
    const fsSvc = ctx.fs;
    const webSvc = ctx.web;
    const log = (msg) => ctx.logger('web-api').log(msg);
    const routes = [];

    // 1. 宿主/插件状态
    ctx.http.register('GET', '/api/ext/status', () => {
      let toolCount = 0;
      try { toolCount = (ctx.tools.list() || []).length; } catch (e) { /* 工具服务不可用时忽略 */ }
      return json(200, {
        ok: true,
        service: 'web-api',
        workspaceRoot: root,
        tools: toolCount,
        time: new Date().toISOString(),
      });
    });
    routes.push('GET /api/ext/status');

    // 2. web fetch 同源代理（GET ?url=…）
    ctx.http.register('GET', '/api/ext/fetch', (req) => {
      const q = parseQuery(req.query);
      const url = q.url || '';
      if (!url) return json(400, { ok: false, error: '缺少 url 查询参数' });
      try {
        const r = webSvc.fetch(url);
        return {
          status: r.ok ? 200 : 502,
          body: r.text || '',
          headers: { 'Content-Type': 'text/plain; charset=utf-8' },
        };
      } catch (e) {
        return json(502, { ok: false, error: String(e) });
      }
    });
    routes.push('GET /api/ext/fetch');

    // 3. 受限读文件（?path=相对工作区根）
    ctx.http.register('GET', '/api/ext/fs/read', (req) => {
      const q = parseQuery(req.query);
      const path = q.path || '';
      if (!path) return json(400, { ok: false, error: '缺少 path 参数' });
      try {
        const text = fsSvc.readFile(path);
        return { status: 200, body: text, headers: { 'Content-Type': 'text/plain; charset=utf-8' } };
      } catch (e) {
        return json(404, { ok: false, error: String(e) });
      }
    });
    routes.push('GET /api/ext/fs/read');

    // 4. 文件存在检查
    ctx.http.register('GET', '/api/ext/fs/exists', (req) => {
      const q = parseQuery(req.query);
      const path = q.path || '';
      if (!path) return json(400, { ok: false, error: '缺少 path 参数' });
      try {
        return json(200, { ok: true, exists: !!fsSvc.exists(path) });
      } catch (e) {
        return json(200, { ok: true, exists: false });
      }
    });
    routes.push('GET /api/ext/fs/exists');

    // 5. 受限目录列表
    ctx.http.register('GET', '/api/ext/fs/list', (req) => {
      const q = parseQuery(req.query);
      const path = q.path || '';
      try {
        const entries = fsSvc.readdir(path);
        return json(200, { ok: true, entries: entries || [] });
      } catch (e) {
        return json(404, { ok: false, error: String(e) });
      }
    });
    routes.push('GET /api/ext/fs/list');

    // 6. 路由清单（自描述）
    ctx.http.register('GET', '/api/ext/routes', () => json(200, { ok: true, service: 'web-api', routes }));
    routes.push('GET /api/ext/routes');

    log(`已注册 /api/ext/* 共 ${routes.length} 条 HTTP 路由（接口插件化落地）`);
  },
}
