// ═══════════════════════════════════════════════════════════════
// tool-subagent — 子 Agent 编排工具（Round3 ③.2，对齐目标语义）
//
// 编排在插件、能力在宿主：execute 走 ctx.agents（宿主 subagent_registry +
// web 层注入的 Spawner）。subagent/subagent_fork 默认后台执行（run_in_background
// 语义 = 异步启动立即返回 convId；调用方可轮询 list_agents/status）。
//
//   subagent         后台委托（ctx.agents.start）
//   subagent_fork    以源会话快照派生（ctx.agents.fork；默认后台）
//   report           成员主动报告（写入 SubAgentRecord.Report）
//   list_agents      列出子 Agent（复用 ctx.agents.list）
//   interrupt_agent  中断子 Agent（复用 ctx.agents.stop）
//   send_message     向子 Agent 投递消息（复用 ctx.agents.followup）
// ═══════════════════════════════════════════════════════════════
const tools = [
  {
    name: 'subagent',
    description:
      '后台委托一个子 Agent 执行任务（独立会话、独立历史）。task 必填；label/team/member 标识归属；system 追加 persona；model/provider/reasoningEffort 可覆盖；denyTools 黑名单。返回 {convId, state}，之后用 list_agents/status 查结果、send_message 续聊、interrupt_agent 中断。',
    parameters: {
      type: 'object',
      properties: {
        task: { type: 'string', description: '委托任务（首轮输入，必填）' },
        label: { type: 'string', description: '展示名（如 team:member）' },
        team: { type: 'string', description: '团队/归属 ID' },
        member: { type: 'string', description: '成员名' },
        system: { type: 'string', description: 'persona：追加到系统提示' },
        model: { type: 'string', description: '模型覆盖（空=默认）' },
        provider: { type: 'string', description: '服务商覆盖（空=当前）' },
        reasoningEffort: { type: 'string', description: '思考档位覆盖（none~max）' },
        denyTools: { type: 'array', items: { type: 'string' }, description: '子 Agent 不可用的工具名' },
        maxIterations: { type: 'integer', description: '单轮最大迭代（<=0 用默认）' },
      },
      required: ['task'],
    },
  },
  {
    name: 'subagent_fork',
    description:
      '以源会话的消息快照派生一个子 Agent（新会话初始历史 = 源会话当前消息 + 任务）。forkFrom 必填（源会话 convID）。persona/模型/工具黑名单经 system/model/provider/denyTools 覆盖。默认后台执行，返回 {convId, forkOf, state}。',
    parameters: {
      type: 'object',
      properties: {
        task: { type: 'string', description: '派生任务（首轮输入，必填）' },
        forkFrom: { type: 'string', description: '源会话 convID（快照其当前消息作为初始历史，必填）' },
        label: { type: 'string', description: '展示名' },
        team: { type: 'string', description: '团队/归属 ID' },
        member: { type: 'string', description: '成员名' },
        system: { type: 'string', description: 'persona：追加到系统提示（覆盖源 persona）' },
        model: { type: 'string', description: '模型覆盖（空=默认）' },
        provider: { type: 'string', description: '服务商覆盖（空=当前）' },
        reasoningEffort: { type: 'string', description: '思考档位覆盖' },
        denyTools: { type: 'array', items: { type: 'string' }, description: '子 Agent 不可用的工具名' },
      },
      required: ['task', 'forkFrom'],
    },
  },
  {
    name: 'report',
    description:
      '子 Agent 主动向宿主报告进度/结论（写入本会话记录，队长 list_agents/status 可读）。text 必填。',
    parameters: {
      type: 'object',
      properties: {
        text: { type: 'string', description: '报告内容（进度/结论/阻塞说明）' },
      },
      required: ['text'],
    },
  },
  {
    name: 'list_agents',
    description:
      '列出全部子 Agent（可按 parentConvId/team 过滤），含 convId/state/turns/forkOf/report 等。',
    parameters: {
      type: 'object',
      properties: {
        parentConvId: { type: 'string', description: '按父会话过滤' },
        team: { type: 'string', description: '按团队过滤' },
      },
      required: [],
    },
    readOnly: true,
  },
  {
    name: 'interrupt_agent',
    description: '中断一个正在运行的子 Agent（convId 必填），其排队消息一并清空。',
    parameters: {
      type: 'object',
      properties: {
        convId: { type: 'string', description: '子 Agent 会话 ID' },
      },
      required: ['convId'],
    },
  },
  {
    name: 'send_message',
    description:
      '向子 Agent 投递一条消息（续聊）。空闲立即执行；运行中排队（轮次结束自动续发）。返回 {queued}。',
    parameters: {
      type: 'object',
      properties: {
        convId: { type: 'string', description: '子 Agent 会话 ID' },
        text: { type: 'string', description: '消息内容' },
      },
      required: ['convId', 'text'],
    },
  },
];

return {
  name: 'tool-subagent',
  purpose: '子 Agent 编排工具面（subagent/subagent_fork/report/list_agents/interrupt_agent/send_message，宿主 ctx.agents）',
  inject: ['agents'], // ctx.agents 按 inject 声明注入
  apply(ctx) {
    const host = {
      subagent: (args) => ctx.agents.start(args || {}),
      subagent_fork: (args) => ctx.agents.fork(args || {}),
      // report：本会话自身报告（_convID 由宿主工具执行链注入）
      report: (args) => ctx.agents.report((args && args._convID) || '', (args && args.text) || ''),
      list_agents: (args) => ctx.agents.list(args || {}),
      interrupt_agent: (args) => ctx.agents.stop((args && args.convId) || ''),
      send_message: (args) => ctx.agents.followup((args && args.convId) || '', (args && args.text) || ''),
    };
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
        execute: (args) => {
          const out = host[t.name](args || {});
          return JSON.stringify(out, null, 2);
        },
      });
    }
  },
};
