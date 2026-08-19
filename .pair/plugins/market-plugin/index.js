// market-plugin — 插件市场（市场插件化：plugin 市场挂载）
//
// 2026-08-19：做自己的插件生态，借用 npm 分发——不再兼容 dsh/cordis 插件约定：
//   - apply 时 ctx.market.register({kind:'plugin'}) 启用插件市场；
//   - 停用/删除本插件 → 插件市场从面板与搜索中消失。
// 搜索/安装实现留在 Go 内核：
//   - source=npm-paircode → npm registry 搜索带 paircode 关键词的插件
//     （发布者 package.json keywords 含 "paircode" 即被收录——自己的生态）；
//   - 安装：无 npm 依赖 → goja 沙箱；有依赖 → Node 运行时桥（真实 node）。
// 无预设数据——用户搜索（query 非空）才显示结果。
return {
  name: 'market-plugin',
  purpose: '插件市场：npm registry 搜索 PairCode 插件（goja 沙箱 / Node 桥安装）',
  inject: ['market', 'logger'],
  apply(ctx) {
    ctx.market.register({
      kind: 'plugin',
      source: 'npm-paircode',
      name: '插件',
      desc: 'npm PairCode 插件（自己的生态，借 npm 分发）：goja 沙箱或 Node 运行时桥安装',
    });
    ctx.logger('market-plugin').log('插件市场已挂载');
  },
};
