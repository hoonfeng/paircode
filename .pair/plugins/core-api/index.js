// ═══════════════════════════════════════════════════════════════
// core-api — 内置 HTTP 接口装配器（接口插件化：Go 硬编码清零）
//
// 背景（2026-08-16）：内置 /api/* 接口（约 90 条）原本在 web_server.go
// mux.HandleFunc 硬编码挂载。接口插件化后：
//   - 实现留在 Go 内核路由表（internal/agent/kernel_api.go，经
//     cmd/companion/kernel_register.go 注册）——能力层，与工具 hostTool 对齐；
//   - 挂载权在本插件：apply 时 ctx.kernel.install(ROUTES) 把清单逐条挂到
//     插件 ext 路由表（ExtRouteMiddleware 优先于宿主 mux 拦截）。
//
// ★ 本文件顶部 ROUTES 数组就是「内置接口清单」——用户可直接增删条目
//   控制接口挂载（删掉某 key = 该接口消失；内核表全部 key 可用
//   ctx.kernel.routes() 查询，缺失的 key 会打印 missing 警告）。
//
// ★ 管理面自锁警告：/api/plugins* 与 /api/toolsets* 也由本插件装配——
//   停用/删除本插件会导致插件面板与工具集面板接口消失（与停用 ui-app
//   同理），属预期行为；恢复方式：重启程序（磁盘插件启动自动装载）。
// ═══════════════════════════════════════════════════════════════

// 内置接口清单（key = 内核路由表标识；增删此数组即增删接口挂载）
const ROUTES = [
  // 系统
  { key: 'health' },
  { key: 'system.info' },
  { key: 'system.exec' },
  // 文件系统
  { key: 'fs.drives' },
  { key: 'fs.list' },
  { key: 'fs.read' },
  { key: 'fs.write' },
  { key: 'fs.rename' },
  { key: 'fs.delete' },
  { key: 'fs.mkdir' },
  { key: 'fs.search' },
  { key: 'fs.image' },
  { key: 'fs.file-info' },
  { key: 'fs.hex' },
  // 工作区 / 设置
  { key: 'workspace' },
  { key: 'settings' },
  { key: 'ui-assembly' },
  // 对话
  { key: 'chat.send' },
  { key: 'chat.stop' },
  { key: 'chat.answer' },
  { key: 'chat.approve' },
  { key: 'chat.feedback' },
  { key: 'chat.rollback' },
  { key: 'chat.compact' },
  // 会话列表
  { key: 'conversations' },
  { key: 'conversations.byID' },
  // Tasks / Plan
  { key: 'tasks' },
  { key: 'taskplan' },
  // 模型 / 指令
  { key: 'models' },
  { key: 'instructions' },
  // 工具配置
  { key: 'tools' },
  { key: 'tools.review' },
  // MCP / Skills
  { key: 'mcp.list' },
  { key: 'mcp.save' },
  { key: 'skills.list' },
  { key: 'skills.read' },
  { key: 'skills.save' },
  { key: 'skills.delete' },
  // Token / Debug
  { key: 'tokens.stats' },
  { key: 'debug.logs' },
  { key: 'debug.logs.byID' },
  // Git
  { key: 'git.status' },
  { key: 'git.init' },
  { key: 'git.diff' },
  { key: 'git.add' },
  { key: 'git.reset' },
  { key: 'git.commit' },
  { key: 'git.log' },
  { key: 'git.log.alias' },
  { key: 'git.branch' },
  { key: 'git.checkout' },
  { key: 'git.stash' },
  { key: 'git.stash-list' },
  { key: 'git.ignore' },
  { key: 'git.discard' },
  { key: 'git.push' },
  { key: 'git.pull' },
  { key: 'git.remote' },
  // 市场
  { key: 'marketplace.search' },
  { key: 'marketplace.install' },
  { key: 'marketplace.uninstall' },
  { key: 'marketplace.refresh' },
  { key: 'marketplace.sources' },
  // 记忆
  { key: 'memory.search' },
  { key: 'memory.list' },
  { key: 'memory.rebuild' },
  // 插件管理（★ 停用本插件会使此组接口消失——管理面自锁，见文件头警告）
  { key: 'plugins' },
  { key: 'plugins.detail' },
  { key: 'plugins.action' },
  { key: 'plugins.define' },
  { key: 'plugins.event' },
  { key: 'plugins.invoke' },
  { key: 'plugins.client-failure' },
  { key: 'plugins.client-events' },
  { key: 'plugins.client-state' },
  { key: 'plugins.builtin' },
  { key: 'plugins.tool' },
  // 工具集
  { key: 'toolsets' },
  { key: 'toolsets.build' },
  { key: 'toolsets.export' },
  { key: 'toolsets.import' },
  { key: 'toolsets.remove' },
    { key: 'toolsets.edit' },
];

return {
  name: 'core-api',
  purpose: '内置 HTTP 接口装配器（接口插件化：Go 硬编码清零）——持有全部内置 /api/* 接口清单，apply 时 ctx.kernel.install 挂载到插件路由表；改本文件 ROUTES 数组可增删接口',
  inject: ['kernel', 'logger'],
  apply(ctx) {
    const log = (msg) => ctx.logger('core-api').log(msg);
    const list = ROUTES.map((r) => ({ key: r.key }));
    const res = ctx.kernel.install(list);

    // 自检：内核表全部 key 与清单对照（缺的打印警告，方便用户补全）
    const all = ctx.kernel.routes() || [];
    const want = new Set(ROUTES.map((r) => r.key));
    const extra = all.filter((r) => !want.has(r.key)).map((r) => r.key);
    const total = ctx.kernel.total ? ctx.kernel.total() : all.length;
    if (extra.length > 0) {
      log(`内核表有 ${extra.length} 个接口未在清单中（未挂载）: ${extra.join(', ')}`);
    }
    log(`已装配内置接口 ${res.installed}/${res.total}（内核表共 ${total}，缺失 ${res.missing}）`);
  },
}
