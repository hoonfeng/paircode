// market-plugin — dsh 插件市场（市场插件化：plugin 市场挂载）
//
// 2026-08-18：dsh 插件市场（npm cordis 插件生态）作为独立磁盘插件声明挂载——
//   - apply 时 ctx.market.register({kind:'plugin'}) 启用插件市场；
//   - 停用/删除本插件 → 插件市场从面板与搜索中消失。
// 搜索/安装实现留在 Go 内核：
//   - source=npm-cordis → npm registry 搜索 cordis 插件（dsh plugin 生态）；
//   - 安装：无 npm 依赖 → goja 沙箱；有依赖 → Node 运行时桥（真实 node）。
// 无预设数据——用户搜索（query 非空）才显示结果。
return {
  name: 'market-plugin',
  purpose: 'dsh 插件市场：npm registry 搜索 cordis 插件（goja 沙箱 / Node 桥安装）',
  inject: ['market', 'logger'],
  apply(ctx) {
    ctx.market.register({
      kind: 'plugin',
      source: 'npm-cordis',
      name: '插件',
      desc: 'npm cordis 插件（dsh 插件生态）：goja 沙箱或 Node 运行时桥安装',
    });
    ctx.logger('market-plugin').log('插件市场已挂载');
  },
};
