// ═══════════════════════════════════════════════════════════════
// tool-shell — 后台进程工具（run_background/read_output/kill_process
// + R2-7 命名别名 job_output/job_list/job_kill）
//
// 迁移来源（2026-08-16）：内置 registerShellTools（internal/agent/shell.go）
// → 磁盘外置插件。★ 调用实现在插件内（JS 编排 ctx.process 宿主服务），
// 不依赖 hostTool——演示「工具实现完全插件化」的形态。
// ctx.process 由宿主提供（globalBG 全局单例，跨 agent 轮次存活）。
// ★ 2026-09 Round2 R2-7：新增 job_output/job_list/job_kill 三个约定命名
//   别名（同一 ctx.process 能力面，原名保留兼容；job_list 用 ctx.process.list）。
// ═══════════════════════════════════════════════════════════════

// run_background：后台启动长命令，返回进程 id
async function runBackground(ctx, args) {
  const command = String(args.command || '').trim()
  if (!command) throw new Error('command 不能为空')
  const { id } = await ctx.process.runBackground(command, args.cwd || '')
  return `已后台启动 id=${id}。用 read_output(id=${id}) 看输出、kill_process(id=${id}) 停止。`
}

// read_output：读取后台进程累积输出与状态
async function readOutput(ctx, args) {
  const { output, done, exitErr, status } = await ctx.process.readOutput(Number(args.id))
  let line = `[${status}]`
  if (done && exitErr) line += `（${exitErr}）`
  const capped = output.length > 16000 ? output.slice(0, 16000) + '\n…[输出截断]' : output
  return `${line}\n${capped}`
}

// kill_process：停止后台进程
async function killProcess(ctx, args) {
  await ctx.process.kill(Number(args.id))
  return `已停止 id=${args.id}`
}

// job_list：列出全部后台进程（job_list 对齐，R2-7）
async function jobList(ctx, args) {
  const jobs = await ctx.process.list()
  if (!jobs || jobs.length === 0) return '（无后台进程）'
  return jobs.map(j => `- id=${j.id} 状态=${j.status}${j.error ? ' 错误=' + j.error : ''}`).join('\n')
}

const tools = [
  {
    name: 'run_background',
    description: '在后台启动一条长命令，不阻塞 agent 循环（推荐用于 dev server、watch 模式、调试服务等）。返回进程 id，随后用 read_output 读输出、kill_process 停止。如果命令会长期运行或保持监听状态，优先用此工具。短查询请用 bash。',
    usageGuide: '后台启动一条长命令，不阻塞 agent 循环。用于 dev server、npm run dev/watch 模式、调试服务、TCP 监听——这些场景只能用此工具，不可用 bash。返回进程 id，之后用 read_output/kill_process 控制。',
    category: '执行',
    parameters: {
      type: 'object',
      properties: {
        command: { type: 'string', description: '要后台执行的命令' },
        cwd: { type: 'string', description: '可选工作目录（工作区内）' },
        project: { type: 'string', description: '可选：目标项目。省略 = 主项目。' },
      },
      required: ['command'],
    },
  },
  {
    name: 'read_output',
    description: '读取某后台进程（id）累积的输出与运行状态（运行中/已结束）。',
    usageGuide: '读取后台进程的累积输出与运行状态。需先用 run_background 启动进程获得 id。比直接看终端更方便（自动截断保护+状态标记运行中/已结束）。',
    category: '执行',
    readOnly: true,
    parameters: {
      type: 'object',
      properties: {
        id: { type: 'integer', description: '进程 id' },
      },
      required: ['id'],
    },
  },
  {
    name: 'kill_process',
    description: '停止某后台进程（id）。只能杀死通过 run_background 启动的进程，无法操作外部进程。',
    usageGuide: '停止某后台进程（仅限通过 run_background 启动的）。进程跑偏/卡死/已不需要时用此工具停止。比 taskkill / pid 更方便（通过 id 直接操作）。',
    category: '执行',
    parameters: {
      type: 'object',
      properties: {
        id: { type: 'integer', description: '进程 id' },
      },
      required: ['id'],
    },
  },
  // ── R2-7 命名别名（job_output/job_list/job_kill，原名保留兼容）──
  {
    name: 'job_output',
    description: '读取某后台任务（id）累积的输出与运行状态（job_output 别名，语义同 read_output）。',
    usageGuide: '读取后台任务累积输出与状态（约定命名）。需先用 run_background 启动获得 id。',
    category: '执行',
    readOnly: true,
    parameters: {
      type: 'object',
      properties: {
        id: { type: 'integer', description: '任务 id' },
      },
      required: ['id'],
    },
  },
  {
    name: 'job_list',
    description: '列出全部后台任务（id + 状态 running/done/error）（job_list 对齐）。',
    usageGuide: '列出全部后台任务（id+状态）。配合 job_output/job_kill 管理后台任务。',
    category: '执行',
    readOnly: true,
    parameters: {
      type: 'object',
      properties: {},
    },
  },
  {
    name: 'job_kill',
    description: '停止某后台任务（id）（job_kill 别名，语义同 kill_process）。',
    usageGuide: '停止后台任务（约定命名）。仅限通过 run_background 启动的进程。',
    category: '执行',
    parameters: {
      type: 'object',
      properties: {
        id: { type: 'integer', description: '任务 id' },
      },
      required: ['id'],
    },
  },
]

const impls = {
  run_background: runBackground,
  read_output: readOutput,
  kill_process: killProcess,
  job_output: readOutput,
  job_list: jobList,
  job_kill: killProcess,
}

return {
  name: 'tool-shell',
  purpose: '后台进程工具（run_background/read_output/kill_process + 命名别名 job_output/job_list/job_kill）——迁移自内置 registerShellTools，调用实现（JS 编排 ctx.process）完全在插件内',
  apply(ctx) {
    for (const t of tools) {
      ctx.tools.register({
        name: t.name,
        description: t.description,
        usageGuide: t.usageGuide,
        category: t.category,
        readOnly: t.readOnly,
        parameters: t.parameters,
        execute: (args) => impls[t.name](ctx, args || {}),
      })
    }
    // 日志已省略（logger 需 inject 声明）
  },
}
