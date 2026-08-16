// ═══════════════════════════════════════════════════════════════
// agentloop — Agent 循环装配器（LoopFactory 单槽位 JS 装配器）
//
// 背景（2026-08-16）：agent 循环核心（Loop 双层循环）在 Go 内核
// （internal/agent/loop.go），会话/持久化/事件协议深度耦合，无法 JS 化。
// 对齐 deepseek-harness 的 AgentLoop cordis 插件（装配期决定循环参数）：
// 本插件在装配期经 ctx.loopFactory.register 注册「参数级装配器」——
// 每次创建循环（会话 Start / 自闭环 Run 统一走 CreateLoop）时，
// apply(opts) 收到当前装配参数快照，返回同形状对象覆盖非空字段。
// 真正换内核留给宿主 Go（ReplaceLoopFactory）；本插件做「配置驱动」。
//
// 配置（package.json "config" 字段，改完重启生效；全部可省=默认行为不变）：
//   systemAppend          追加到系统提示词末尾的片段（如行为规范/角色设定）
//   maxIterations         最大迭代数覆盖
//   maxContextTokens      上下文 token 上限覆盖
//   autonomous            自主模式覆盖（bool）
//   maxAutonomousMinutes  自主时间预算（分钟）
//   checkpointInterval    检查点间隔（迭代数）
//   reviewMode            审核模式覆盖（"auto"/"manual"/"off"）
//   autoCommit            完成自动提交覆盖（bool）
//   reviewBlacklist       审核黑名单工具名数组
//   reviewWhitelist       审核白名单工具名数组
//
// 生命周期：停用/删除本插件 → 装配器自动还原默认工厂（Loop 不受影响）。
// ═══════════════════════════════════════════════════════════════

return {
  name: 'agentloop',
  purpose: 'Agent 循环装配器（LoopFactory 参数级装配：模型/迭代/审核/提示词追加）',
  inject: ['logger'],
  apply(ctx, config) {
    const cfg = config || {};
    ctx.loopFactory.register((opts) => {
      const over = {};
      if (typeof cfg.systemAppend === 'string' && cfg.systemAppend) {
        over.system = (opts.system || '') + '\n\n' + cfg.systemAppend;
      }
      if (cfg.maxIterations != null) over.maxIterations = Number(cfg.maxIterations);
      if (cfg.maxContextTokens != null) over.maxContextTokens = Number(cfg.maxContextTokens);
      if (typeof cfg.autonomous === 'boolean') over.autonomous = cfg.autonomous;
      if (cfg.maxAutonomousMinutes != null) over.maxAutonomousMinutes = Number(cfg.maxAutonomousMinutes);
      if (cfg.checkpointInterval != null) over.checkpointInterval = Number(cfg.checkpointInterval);
      if (typeof cfg.reviewMode === 'string' && cfg.reviewMode) over.reviewMode = cfg.reviewMode;
      if (typeof cfg.autoCommit === 'boolean') over.autoCommit = cfg.autoCommit;
      if (Array.isArray(cfg.reviewBlacklist)) over.reviewBlacklist = cfg.reviewBlacklist.filter(x => typeof x === 'string' && x);
      if (Array.isArray(cfg.reviewWhitelist)) over.reviewWhitelist = cfg.reviewWhitelist.filter(x => typeof x === 'string' && x);
      return over;
    });
    const log = ctx.logger('agentloop');
    log.info('已注册 Agent 循环装配器（LoopFactory 单槽位；config=' + JSON.stringify(cfg) + '）');
  }
};
