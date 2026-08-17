// market-mcp — MCP 市场（市场插件化：mcp 市场挂载）
//
// 2026-08-18：MCP 市场作为独立磁盘插件声明挂载——
//   - apply 时 ctx.market.register({kind:'mcp'}) 启用 MCP 市场；
//   - 停用/删除本插件 → MCP 市场从面板与搜索中消失。
// 搜索实现留在 Go 内核（source=npm → npm registry 搜索 MCP 服务器），
// 无预设数据——用户搜索（query 非空）才显示结果。
return {
  name: 'market-mcp',
  purpose: 'MCP 市场：npm registry 搜索 MCP 服务器（安装到用户级/工作区级配置）',
  inject: ['market', 'logger'],
  apply(ctx) {
    ctx.market.register({
      kind: 'mcp',
      source: 'npm',
      name: 'MCP',
      desc: 'npm registry 搜索 MCP 服务器（npx 启动，安装到 MCP 配置）',
    });
    ctx.logger('market-mcp').log('MCP 市场已挂载');
  },
};
