// market-skill — 技能市场（市场插件化：skill 市场挂载）
//
// 2026-08-18：skill 市场作为独立磁盘插件声明挂载——
//   - apply 时 ctx.market.register({kind:'skill'}) 启用技能市场；
//   - 停用/删除本插件 → 技能市场从面板与搜索中消失（其余市场不受影响）。
// 搜索实现留在 Go 内核（source=github → GitHub 仓库 → skill 条目），
// 无预设数据——用户搜索（query 非空）才显示结果。
return {
  name: 'market-skill',
  purpose: '技能市场：GitHub 仓库搜索 → skill 条目（安装到 .pair/skills）',
  inject: ['market', 'logger'],
  apply(ctx) {
    ctx.market.register({
      kind: 'skill',
      source: 'github',
      name: '技能',
      desc: 'GitHub 仓库搜索 → 技能条目（安装到工作区 .pair/skills）',
    });
    ctx.logger('market-skill').log('技能市场已挂载');
  },
};
