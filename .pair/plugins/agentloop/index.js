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
// ★ 配置注册化（2026-08-19）：不再使用 package.json "config" 字段——
//   改经 ctx.registerSettings(schema) 在框架注册配置段（前端设置面板
//   「插件配置」tab 动态渲染），值存 settings.json 的 pluginSettings.agentloop；
//   运行时 ctx.getSettings('agentloop') 读取，改完保存后重启生效。
//
// 生命周期：停用/删除本插件 → 装配器自动还原默认工厂（Loop 不受影响）。
// ═══════════════════════════════════════════════════════════════

return {
  name: 'agentloop',
  purpose: 'Agent 循环装配器（配置注册化：模型/迭代/审核/提示词追加）',
  inject: ['logger'],
  apply(ctx, config) {
    // ── 配置注册化：声明本插件需要的配置段（框架统一管理 + 前端动态渲染）──
    const reg = ctx.registerSettings({
      key: 'agentloop',
      title: 'Agent 循环',
      fields: [
        { name: 'systemAppend', label: '系统提示词追加', type: 'textarea', default: '', hint: '追加到系统提示词末尾（如行为规范/角色设定）' },
        { name: 'maxIterations', label: '最大迭代数', type: 'number', default: 0, hint: '0=不覆盖（默认 50）' },
        { name: 'maxContextTokens', label: '上下文 token 上限', type: 'number', default: 0, hint: '0=不覆盖' },
        { name: 'autonomous', label: '自主模式', type: 'checkbox', default: false, hint: '覆盖全局自主模式开关' },
        { name: 'maxAutonomousMinutes', label: '自主时间预算（分钟）', type: 'number', default: 0, hint: '0=不覆盖' },
        { name: 'checkpointInterval', label: '检查点间隔（迭代数）', type: 'number', default: 0, hint: '0=不覆盖' },
        { name: 'reviewMode', label: '审核模式', type: 'select', default: '', options: ['', 'auto', 'manual', 'off'], hint: '空=不覆盖（auto=AI审核/manual=手动审批/off=放行）' },
        { name: 'autoCommit', label: '完成自动提交', type: 'checkbox', default: false, hint: '覆盖完成自动提交开关' },
        { name: 'reviewBlacklist', label: '审核黑名单', type: 'text', default: '', hint: '逗号分隔工具名（命中需审核）' },
        { name: 'reviewWhitelist', label: '审核白名单', type: 'text', default: '', hint: '逗号分隔工具名（命中跳过审核，黑名单优先）' },
      ],
    });

    // ── 运行时读取配置（registerSettings 返回当前值=已存值合并默认）──
    const cfg = (reg && reg.value) || ctx.getSettings('agentloop') || {};

    ctx.loopFactory.register((opts) => {
      const over = {};
      if (typeof cfg.systemAppend === 'string' && cfg.systemAppend) {
        over.system = (opts.system || '') + '\n\n' + cfg.systemAppend;
      }
      if (cfg.maxIterations != null && Number(cfg.maxIterations) > 0) over.maxIterations = Number(cfg.maxIterations);
      if (cfg.maxContextTokens != null && Number(cfg.maxContextTokens) > 0) over.maxContextTokens = Number(cfg.maxContextTokens);
      if (typeof cfg.autonomous === 'boolean') over.autonomous = cfg.autonomous;
      if (cfg.maxAutonomousMinutes != null && Number(cfg.maxAutonomousMinutes) > 0) over.maxAutonomousMinutes = Number(cfg.maxAutonomousMinutes);
      if (cfg.checkpointInterval != null && Number(cfg.checkpointInterval) > 0) over.checkpointInterval = Number(cfg.checkpointInterval);
      if (typeof cfg.reviewMode === 'string' && cfg.reviewMode) over.reviewMode = cfg.reviewMode;
      if (typeof cfg.autoCommit === 'boolean') over.autoCommit = cfg.autoCommit;
      if (typeof cfg.reviewBlacklist === 'string' && cfg.reviewBlacklist) {
        over.reviewBlacklist = cfg.reviewBlacklist.split(/[,，]/).map(s => s.trim()).filter(Boolean);
      }
      if (typeof cfg.reviewWhitelist === 'string' && cfg.reviewWhitelist) {
        over.reviewWhitelist = cfg.reviewWhitelist.split(/[,，]/).map(s => s.trim()).filter(Boolean);
      }
      return over;
    });
    const log = ctx.logger('agentloop');
    log.info('已注册 Agent 循环装配器（配置注册化 schema=' + (reg ? reg.key : 'agentloop') + '；当前值=' + JSON.stringify(cfg) + '）');
  }
};
