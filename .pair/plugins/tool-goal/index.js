// ═══════════════════════════════════════════════════════════════
// tool-goal — goal 工具（Round3 ③.1，对齐 goal 工具）
//
// 编排在插件、能力在宿主：schema/描述在插件，状态机与自动续轮在宿主
// （internal/agent/goal.go）。execute 经 ctx.hostTool.exec 路由回宿主
// 执行器（_convID 由宿主工具执行链自动注入，多会话并发不串）。
//
// 语义对齐：
//   - create_goal：objective 必填（直接给出，不做 LLM 推断）、max_goal_rounds 可选
//   - get_goal：返回 goal_id/revision/objective/phase/rounds/roundLimit/
//     blockerReason/armed
//   - update_goal：action ∈ {edit,pause,resume,complete,blocked}；revision 乐观锁
// ═══════════════════════════════════════════════════════════════
const tools = [
  {
    name: 'create_goal',
    description:
      '创建同会话完成目标（对齐 goal）。objective 必填（直接给出目标，不做推断）；max_goal_rounds 可选（自动续轮上限，默认 3）。创建后会话将在每轮结束后自动续轮推进，直到 update_goal complete/blocked 或达轮次上限。',
    parameters: {
      type: 'object',
      properties: {
        objective: { type: 'string', description: '目标描述（祈使句，直接给出，如「修复登录超时 bug」）' },
        max_goal_rounds: { type: 'integer', description: '可选：自动续轮上限（默认 3；0=不限——慎用，会无限续轮）' },
      },
      required: ['objective'],
    },
    systemTool: true,
  },
  {
    name: 'get_goal',
    description:
      '读取当前会话目标（goal_id/revision/objective/phase/rounds/roundLimit/blockerReason/armed）。无目标返回提示。',
    parameters: { type: 'object', properties: {}, required: [] },
    readOnly: true,
    systemTool: true,
  },
  {
    name: 'update_goal',
    description:
      '更新当前会话目标（对齐 goal update）。action ∈ {edit,pause,resume,complete,blocked}；revision 必传（乐观锁，冲突拒绝）。edit 可改 objective/max_goal_rounds；pause 停续轮、resume 重挂；complete 标记完成；blocked 标记阻塞（blocked_reason 必填说明）。',
    parameters: {
      type: 'object',
      properties: {
        goal_id: { type: 'string', description: '目标 ID（=会话 ID；get_goal 可查）' },
        revision: { type: 'integer', description: '当前 revision（get_goal 返回；冲突时拒绝）' },
        action: { type: 'string', description: 'edit / pause / resume / complete / blocked' },
        objective: { type: 'string', description: 'edit 用：新目标描述（可选）' },
        max_goal_rounds: { type: 'integer', description: 'edit 用：新自动续轮上限（可选）' },
        blocked_reason: { type: 'string', description: 'blocked 用：阻塞原因（必填）' },
      },
      required: ['goal_id', 'revision', 'action'],
    },
    systemTool: true,
  },
];

return {
  name: 'tool-goal',
  purpose: 'goal 机制工具面（create_goal/get_goal/update_goal，宿主 goal 状态机 + 自动续轮）',
  apply(ctx) {
    for (const t of tools) {
      ctx.tools.register({
        name: t.name,
        description: t.description,
        usageGuide: t.usageGuide,
        category: t.category,
        readOnly: t.readOnly,
        requiresApproval: t.requiresApproval,
        systemTool: t.systemTool,
        parameters: t.parameters,
        execute: (args) => ctx.hostTool.exec(t.name, args || {}),
      });
    }
  },
};
