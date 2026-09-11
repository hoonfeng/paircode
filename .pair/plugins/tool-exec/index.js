// ═══════════════════════════════════════════════════════════════
// tool-exec — 会话式命令执行（exec_command / write_stdin / kill_process）
//
// ★ 2026-09 工具重构（对齐 codex exec_command/write_stdin 模型）：
//   一个 exec_command 覆盖「同步等待 + 长命令让出 + 会话复用」三态——
//   yield_time_ms 内完成即返回全量输出；超时仍在跑则返回 session_id，
//   后续用 write_stdin 轮询增量输出 / 写 stdin 交互，kill_process 终止。
//   取代旧 4 件套（run_background/read_output/kill_process/job_list——
//   异步轮询式，无同步等待、无 stdin、无增量语义）。
//
// 能力面：ctx.process（宿主 globalBG，跨轮次/跨 Registry 存活）
//   runCommand({command, cwd, yieldMs}) → {output, sessionId, running, exitCode, exitErr}
//   writeStdin({id, chars, yieldMs})    → {output(增量), running, exitCode, exitErr}
//   kill(id)
//
// 装配：.pair/plugins/ 启动扫描（LoadGlobalPlugins）→ define + load。
// 停用本插件（cordis(op=stop) tool-exec）即回收全部 3 个工具。
// ═══════════════════════════════════════════════════════════════

// 默认输出预算（token；1 token ≈ 4 字符折算——对齐 codex max_output_tokens 默认 10000）
const DEFAULT_MAX_TOKENS = 10000;
const DEFAULT_YIELD_MS = 10000;
const MIN_YIELD_MS = 250;
const MAX_YIELD_MS = 30000;

// 是否绝对路径（Windows 盘符 / Unix 根 / 反斜杠开头）。
function isAbsPath(p) {
  const s = String(p == null ? '' : p);
  return /^[a-zA-Z]:[\\/]/.test(s) || s.startsWith('/') || s.startsWith('\\');
}

// 项目路由：project 非空 → 目标项目（工作区另一根）下的路径。
//   · path 为绝对路径 → 原样返回（越界由宿主 resolve 拦截）
//   · project 为绝对路径 → 直接作为前缀拼接
//   · 其余 → 以 ../<project>/ 前缀拼相对路径（宿主多根归属检查通过）
function projPath(args, path) {
  const project = args.project;
  if (!project) return path;
  if (isAbsPath(path)) return path;
  const rel = String(path == null ? '' : path).replace(/^[\\/]+/, '');
  if (!rel) return path;
  const proj = String(project).replace(/[\\/]+$/, '');
  if (isAbsPath(proj)) return proj + '/' + rel;
  return '../' + proj.replace(/^[\\/]+/, '') + '/' + rel;
}

// yield 毫秒解析：缺省/非法 → def；否则钳制到 [250, 30000]
function clampYield(ms, def) {
  const v = Math.round(Number(ms || 0));
  if (!(v > 0)) return def;
  return Math.min(MAX_YIELD_MS, Math.max(MIN_YIELD_MS, v));
}

// 输出预算截断（保留尾部——构建/日志尾部信息量最大；前部给省略提示）
function capByTokens(text, maxTokens) {
  const budget = Math.max(1, Math.round(Number(maxTokens) > 0 ? Number(maxTokens) : DEFAULT_MAX_TOKENS)) * 4;
  if (!text || text.length <= budget) return text || '';
  return '…[前部输出已省略 ' + (text.length - budget) + ' 字符]\n' + text.slice(text.length - budget);
}

// 状态尾注（供模型感知会话生命周期）
function statusLine(running, sessionId, exitCode, exitErr, hint) {
  if (running) {
    const base = '[运行中 session_id=' + sessionId + ']';
    return hint ? '[运行中 session_id=' + sessionId + ' — 用 write_stdin 轮询输出/写 stdin，kill_process 停止]' : base;
  }
  const code = exitCode != null && Number(exitCode) >= 0 ? '退出码=' + exitCode : '退出码未知';
  return '[已结束 ' + code + (exitErr ? ' · ' + exitErr : '') + ']';
}

// ─── exec_command：执行命令（同步等待 → 超时转会话） ─────────
async function execCommand(ctx, args) {
  const command = String(args.command == null ? '' : args.command).trim();
  if (!command) throw new Error('command 不能为空');
  const opts = { command, yieldMs: clampYield(args.yield_time_ms, DEFAULT_YIELD_MS) };
  if (args.workdir != null && String(args.workdir) !== '') {
    opts.cwd = projPath(args, String(args.workdir));
  }
  const r = await ctx.process.runCommand(opts);
  let text = capByTokens(r.output || '', args.max_output_tokens);
  if (!text) text = '（无输出）';
  return text + '\n' + statusLine(r.running, r.sessionId, r.exitCode, r.exitErr, true);
}

// ─── write_stdin：向会话写 stdin / 轮询增量输出 ──────────────
async function writeStdin(ctx, args) {
  const id = Math.round(Number(args.session_id || 0));
  if (!(id > 0)) throw new Error('session_id 必填（正整数会话 id）');
  const chars = args.chars == null ? '' : String(args.chars);
  const r = await ctx.process.writeStdin({ id, chars, yieldMs: clampYield(args.yield_time_ms, 0) });
  let text = capByTokens(r.output || '', args.max_output_tokens);
  if (!text) text = chars ? '（已写入 stdin，无新输出）' : '（无新输出）';
  return text + '\n' + statusLine(r.running, id, r.exitCode, r.exitErr, false);
}

// ─── kill_process：终止会话 ─────────────────────────────────
async function killProcess(ctx, args) {
  const id = Math.round(Number(args.session_id || 0));
  if (!(id > 0)) throw new Error('session_id 必填（正整数会话 id）');
  await ctx.process.kill(id);
  return '已停止 session_id=' + id;
}

// ─── 工具声明 ───────────────────────────────────────────────
const tools = [
  {
    name: 'exec_command',
    description: '执行一条 shell 命令并返回输出（统一执行入口：短命令同步返回；长命令超时转会话，用 write_stdin 继续轮询/交互，kill_process 终止）。默认工作目录=主项目根；多项目工作区在其他项目下执行请传 workdir（相对主项目根）或 project（项目目录名）。',
    usageGuide: '统一 shell 执行。短查询（git/构建/测试/文件操作）直接传 command 同步拿结果；长进程（dev server/watch）等待 yield_time_ms 后返回 session_id，用 write_stdin 轮询输出、kill_process 停止。cwd 语义：workdir 相对「主项目根」解析，也可用 project 参数切到其他项目根，或直接给绝对路径。比 run_code 包装更直接，比异步工具链少一次往返。',
    category: '执行',
    parameters: {
      type: 'object',
      properties: {
        command: { type: 'string', description: '要执行的 shell 命令（bash 语法；无 bash 时 cmd 兜底）' },
        workdir: { type: 'string', description: '可选：工作目录（工作区内；相对主项目根解析，省略=主项目根；跨项目可传绝对路径或用 project 参数）' },
        yield_time_ms: { type: 'integer', description: '可选：等待毫秒（默认 10000，范围 250-30000）；超时仍在跑则返回 session_id' },
        max_output_tokens: { type: 'integer', description: '可选：输出 token 预算（默认 10000，超限保留尾部）' },
        project: { type: 'string', description: '可选：目标项目（工作区项目目录名如 ref，或相对主项目的路径/绝对路径）——命令工作目录切到该项目根。省略 = 主项目。' },
      },
      required: ['command'],
    },
    impl: execCommand,
  },
  {
    name: 'write_stdin',
    description: '向 exec_command 启动的会话进程写入 stdin（chars 空 = 仅轮询）并等待新输出（增量，自上次读取起）。',
    usageGuide: '会话续写：交互式进程（提示符/确认）用 chars 写入输入；长进程轮询增量输出用空 chars。等待 yield_time_ms 后返回新输出段；进程结束会带退出码。',
    category: '执行',
    readOnly: true,
    parameters: {
      type: 'object',
      properties: {
        session_id: { type: 'integer', description: '会话 id（exec_command 返回）' },
        chars: { type: 'string', description: '可选：写入 stdin 的字符（省略/空 = 仅轮询输出）' },
        yield_time_ms: { type: 'integer', description: '可选：等待毫秒（写入默认 250；轮询默认 5000）' },
        max_output_tokens: { type: 'integer', description: '可选：输出 token 预算（默认 10000）' },
      },
      required: ['session_id'],
    },
    impl: writeStdin,
  },
  {
    name: 'kill_process',
    description: '终止 exec_command 的会话进程（session_id）。',
    usageGuide: '停止长进程（dev server/watch/卡死命令）。仅限通过 exec_command 启动的会话；已结束的会话调用无害（幂等）。',
    category: '执行',
    parameters: {
      type: 'object',
      properties: {
        session_id: { type: 'integer', description: '会话 id（exec_command 返回）' },
      },
      required: ['session_id'],
    },
    impl: killProcess,
  },
];

return {
  name: 'tool-exec',
  purpose: '会话式命令执行（exec_command/write_stdin/kill_process）——对齐 codex unified exec 模型，取代 run_background/read_output/job_list',
  inject: ['process'],
  apply(ctx) {
    for (const t of tools) {
      ctx.tools.register({
        name: t.name,
        description: t.description,
        usageGuide: t.usageGuide,
        category: t.category,
        readOnly: t.readOnly,
        parameters: t.parameters,
        execute: (args) => t.impl(ctx, args || {}),
      });
    }
  },
};
