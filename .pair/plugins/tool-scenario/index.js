// ═══════════════════════════════════════════════════════════════
// tool-scenario — 场景创造（创造模式：/创造 <需求>）
//
// 定位：agent 自主根据需求创建「场景工具集」（场景 = 工具集 = 工具面白名单；
//   固化到安装目录 .pair/toolsets/ —— 内置场景/预设所在），创建后对话面板
//   「工具集」选择器选中即生效（工具面每轮对话重建，无需重启）。
//
// 形态（按需激活，对齐 agent-teams 模式；2026-09-11 用户需求）：
//   · /创造 <需求> 命令（ctx.activation.declare on-demand）——执行后本会话激活：
//     其工具（scenario_scan/scenario_create）注册进 agent（宿主按激活状态合并，
//     跨对话保持；未激活不进 agent 工具面）。
//   · 系统提示段（alwaysVisible）：引导 + 创造流程协议。
//   · 工具 schema 声明在插件、execute 经 ctx.hostTool 路由宿主 Go 实现
//     （internal/agent/scenario_tools.go，「编排在插件、能力在宿主」seam）。
// ═══════════════════════════════════════════════════════════════

const tools = [
  {
    name: 'scenario_scan',
    description:
      '盘点当前可用能力（插件与内置工具组清单），供创建「场景工具集」时组合决策：分节列出磁盘/动态插件（插件名 + 用途 + 工具名）与内置工具组（builtin:<组名> + 工具名）。detail=full 附每个工具的一行描述。先本工具盘点，再用 scenario_create 组合创建。',
    usageGuide:
      '创造模式第一步：盘点可用插件/内置组。组合单位 = 插件（scenario_create 的 plugins 写插件名）/ 内置组（写 "builtin:<组名>"）。',
    readOnly: true,
    parameters: {
      type: 'object',
      properties: {
        detail: { type: 'string', description: 'summary（默认）=插件/组+工具名；full=附工具描述' },
      },
      required: [],
    },
  },
  {
    name: 'scenario_create',
    description:
      '一次性创建「场景工具集」并固化到安装目录 .pair/toolsets/<name>.json（内置场景/预设同目录；创建后对话面板选择即生效）。name 场景名（中文或小写字母/数字/-/_；预设名保留不可用）；description 用途描述；plugins 组合清单，元素为：字符串（"tool-office" 整插件 / "builtin:core" 整内置组）或对象（{plugin:"tool-office",tools:["csv_read"]} 仅白名单工具 / {group:"core"}）。先用 scenario_scan 盘点可用插件/组再组合。场景 = 工具面白名单：选中该场景时仅清单内工具对 agent 可见。',
    usageGuide:
      '创造模式第二步：组合创建场景。先 scenario_scan 看清单；plugins 优先复用现有插件/组；创建后 toolset_show 验证、报告用户如何启用。',
    requiresApproval: true,
    parameters: {
      type: 'object',
      properties: {
        name: { type: 'string', description: '场景名（中文或小写字母/数字/-/_，如 "音视频"、"data-analysis"；预设名不可用）' },
        description: { type: 'string', description: '场景用途描述（对话面板展示；必填建议）' },
        plugins: {
          type: 'array',
          items: { type: ['string', 'object'] },
          description: '组合清单："插件名"/"builtin:组名"，或 {plugin:"名",tools:[...]}/{group:"组名"}',
        },
        overwrite: { type: 'string', description: '同名场景已存在时 true=覆盖重建（默认 false 报错）' },
      },
      required: ['name', 'plugins'],
    },
  },
]

// creationGuide：创造模式流程指导（/创造 命令的激活指令主体）。
function creationGuide(requirement) {
  const lines = [
    '用户请求创建新的「场景工具集」（创造模式）。你是创造者：分析需求 → 盘点现有能力 → 组合 → 创建并保存（自主执行，不要反复确认）。',
    '流程：',
    '1. 分析需求需要的能力域（3-8 个能力点：文件/命令/文档表格/图谱/网页/记忆/知识库/二进制/图像等）。',
    '2. 调用 scenario_scan 盘点可用插件与内置组（需要更细工具描述时 detail=full）。',
    '3. 组合：每个能力点映射到具体插件 / builtin:<组名>；优先复用现有工具，缺口如实说明。',
    '4. 调用 scenario_create 创建：name（中文短名词，如「音视频」）、description、plugins 清单。',
    '5. 调用 toolset_show <name> 验证已固化（.pair/toolsets/<name>.json，安装目录 = 内置场景所在）。',
    '6. 向用户报告：场景名、组合了哪些能力、如何在对话面板选择启用（一句话即可）。',
  ]
  if (requirement === '') {
    lines.push('需求未给出——先问用户这个场景要覆盖什么能力（或从最近对话上下文推断后向用户确认）。')
  } else {
    lines.push('需求：' + requirement)
  }
  return lines.join('\n')
}

return {
  name: 'tool-scenario',
  inject: ['logger', 'commands'],
  purpose:
    '场景创造（创造模式）：/创造 <需求> 按需激活——scenario_scan 盘点能力、scenario_create 组合创建场景工具集并固化到安装目录 .pair/toolsets/',
  apply(ctx) {
    const log = ctx.logger('tool-scenario')

    // ── 按需激活：/创造 触发本会话激活（其工具随之注册进 agent；未激活不占工具面）──
    try {
      const r = ctx.activation.declare({ command: '创造' })
      if (r && r.ok) log.info('已声明按需激活：/创造 触发本会话激活')
    } catch (e) {
      log.warn('ctx.activation.declare 失败（宿主不支持按需激活，插件按常驻注入）: ' + (e && e.message || e))
    }

    // ── 工具注册（schema 在插件、执行在宿主 ctx.hostTool）──
    for (const t of tools) {
      ctx.tools.register({
        name: t.name,
        description: t.description,
        usageGuide: t.usageGuide,
        readOnly: t.readOnly,
        requiresApproval: t.requiresApproval,
        parameters: t.parameters,
        execute: (args) => ctx.hostTool.exec(t.name, args || {}),
      })
    }

    // ── slash 命令 /创造 ──
    try {
      ctx.commands.register({
        name: '创造',
        description: '创造模式：/创造 <需求> 自主创建场景工具集（保存到安装目录 .pair/toolsets/）',
        handler: (args) => {
          try {
            const raw = String((args && args.args) || '').trim()
            return creationGuide(raw)
          } catch (e) {
            return '创造模式指令生成失败: ' + (e && e.message || e)
          }
        },
      })
      log.info('slash 命令 /创造 已注册')
    } catch (e) {
      log.warn('ctx.commands 注册失败（宿主不支持 commands 面，命令不可用）: ' + (e && e.message || e))
    }

    // ── 系统提示段（常驻引导：本模式是什么、没激活时怎么提示用户）──
    ctx.systemPrompt.section({
      name: 'tool-scenario',
      order: 118,
      alwaysVisible: true,
      text: [
        '# 场景创造（创造模式）',
        '',
        '「场景」= 工具集（工具面白名单；对话面板「工具集」选择器切换）。',
        '本模式用于：根据需求自主创建新的场景工具集，并保存到安装目录 .pair/toolsets/（内置场景/预设所在）。',
        '',
        '- 激活：用户在对话执行 /创造 <需求> → 本会话获得 scenario_scan / scenario_create 工具（未激活时不占工具面）。',
        '- 流程：scenario_scan 盘点现有插件与内置组 → 组合（优先复用现有工具）→ scenario_create 创建 → toolset_show 验证 → 报告用户如何启用。',
        '- 若你的工具面中没有 scenario_* 工具：创造模式未激活——引导用户执行 /创造 <需求>（激活后工具即可用）。',
        '- 场景名：中文或小写字母/数字/-/_；预设名（基础/调试/办公/…）保留不可用；保存位置 .pair/toolsets/<name>.json。',
      ].join('\n'),
    })

    log.info('tool-scenario 插件已装载（创造模式：/创造 <需求> → scenario_scan/scenario_create）')
  },
}
