// ═══════════════════════════════════════════════════════════════
// web-api — HTTP 接口插件化落地（ctx.webServer 对齐 harness dsh-host-webserver）
//
// 背景（2026-08-16）：宿主提供 HTTP 接口插件化扩展点，本插件把能力落到磁盘：
// 注册 /api/ext/* 扩展路由，证明「接口在插件中定义、处理逻辑在插件中、
// 服务能力走 ctx.fs/ctx.web/ctx.tools 等 Go 服务」链路全通。
//
// ★ 2026-08-18：注册形态对齐参考项目（ref/deepseek-harness）——ctx.webServer
//   register({kind, path, handler})，handler 为 Node 风格 (req, res)：
//   res.writeHead(status, headers) / res.end(body) / res.statusCode 属性赋值；
//   支持 async handler（返回值 Promise 同步 drain）与 req.json() 解析。
//   dsh 生态插件的 webServer 注册代码可直接兼容。
//
// 路由清单（5 条，全部 ctx.webServer 注册）：
//   GET  /api/ext/status        宿主/插件状态（工作区根、工具数、服务名、时间）
//   GET  /api/ext/fetch         web fetch 同源代理（?url=…）
//   GET  /api/ext/routes        全量已注册路由清单（ctx.http.list()）
//   GET  /api/ext/async/status  ★ async handler 示例（await 编排）
//   POST /api/ext/echo          ★ req.json() 示例（JSON 体解析回显）
//
// ★ 2026-09 Round2（R2-10）：/api/ext/fs/{read,exists,list} 三路由已删除——
//   与 fs-api 插件 /api/fs/{read,list} 重复（fs-api 更完整），实测零消费者。
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

// Node 风格响应助手（res 完全持有响应生命周期，对齐 harness handler 形态）
function send(res, status, obj) {
  res.writeHead(status, { 'Content-Type': 'application/json; charset=utf-8' });
  res.end(JSON.stringify(obj));
}

return {
  name: 'web-api',
  purpose: 'HTTP 接口插件化落地（ctx.webServer 对齐 harness）——注册 /api/ext/* 扩展路由：宿主状态、web fetch 同源代理、async handler 与 req.json() 示例（Round2 移除与 fs-api 重复的 /api/ext/fs/* 三路由）',
  inject: ['web', 'tools', 'logger'],
  apply(ctx) {
    const root = (ctx.app && ctx.app.workspaceRoot) || ctx.workspaceRoot || '';
    const webSvc = ctx.web;
    const log = (msg) => ctx.logger('web-api').log(msg);
    const routes = [];

    // 1. 宿主/插件状态（Node 风格 res）
    ctx.webServer.register({ kind: 'exact', path: '/api/ext/status', handler: (req, res) => {
      let toolCount = 0;
      try { toolCount = (ctx.tools.list() || []).length; } catch (e) { /* 忽略 */ }
      send(res, 200, {
        ok: true,
        service: 'web-api',
        workspaceRoot: root,
        tools: toolCount,
        time: new Date().toISOString(),
      });
    }});
    routes.push('GET /api/ext/status');

    // 2. web fetch 同源代理（GET ?url=…）
    ctx.webServer.register({ kind: 'exact', path: '/api/ext/fetch', handler: (req, res) => {
      const q = parseQuery(req.query);
      const url = q.url || '';
      if (!url) return send(res, 400, { ok: false, error: '缺少 url 查询参数' });
      try {
        const r = webSvc.fetch(url);
        res.writeHead(r.ok ? 200 : 502, { 'Content-Type': 'text/plain; charset=utf-8' });
        res.end(r.text || '');
      } catch (e) {
        send(res, 502, { ok: false, error: String(e) });
      }
    }});
    routes.push('GET /api/ext/fetch');

    // 6. 路由清单（自描述；ctx.http.list() 返回全量已注册路由，含内核安装条目）
    ctx.webServer.register({ kind: 'exact', path: '/api/ext/routes', handler: (req, res) => {
      const all = ctx.http.list() || [];
      send(res, 200, { ok: true, service: 'web-api', routes, all });
    }});
    routes.push('GET /api/ext/routes');

    // 7. ★ async handler 示例：async 函数 + await ctx.tools.list()（Go 服务）+
    //    await Promise.resolve（微任务链）——处理逻辑可异步编排。
    ctx.webServer.register({ kind: 'exact', path: '/api/ext/async/status', handler: async (req, res) => {
      const tools = await ctx.tools.list();
      const r2 = await Promise.resolve((ctx.app && ctx.app.workspaceRoot) || ctx.workspaceRoot || '');
      send(res, 200, {
        ok: true,
        async: true,
        service: 'web-api',
        workspaceRoot: r2,
        tools: (tools || []).length,
      });
    }});
    routes.push('GET /api/ext/async/status');

    // 8. req.json() 示例：POST JSON 体解析 → 回显（服务在 Go，逻辑在插件）
    ctx.webServer.register({ kind: 'exact', path: '/api/ext/echo', handler: (req, res) => {
      let data;
      try { data = req.json(); } catch (e) { return send(res, 400, { ok: false, error: String(e) }); }
      send(res, 200, { ok: true, method: req.method, path: req.path, received: data });
    }});
    routes.push('POST /api/ext/echo');

    log(`已注册 /api/ext/* 共 ${routes.length} 条 HTTP 路由（ctx.webServer 对齐 harness）`);
  },
}
