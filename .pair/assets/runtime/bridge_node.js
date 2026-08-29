// ═══════════════════════════════════════════════════════════
// bridge_node.js — Node 运行时桥（真实 node 进程执行 npm cordis 插件）
//
// 用途：goja 沙箱只能执行无 npm 依赖的插件（非相对导入 mock 空模块）；
// 依赖 npm 生态（dotenv/axios 等）的 cordis 插件在真实 node 中运行，
// 通过本桥与 Go 主进程通信。插件可用 ctx.tools.register 注册工具
// （进 Go 工具表，agent 可直接调用）；ctx.fs/web/bash 服务转发 Go 侧
// 对应工具实现（复用 read_file/web_fetch/run_command 等，行为与 goja
// 插件一致）。
//
// ★ Round4（2026-09）：DSH 插件运行时（cordis4 轨）。
//   插件形态：npm 包，peer 依赖 @deepseek-ai/cordis ^4 + @deepseek-ai/dsh-*，
//   apply(ctx) 使用 cordis4 Context 语义（inject/get/effect/on）+ DSH 服务面
//   （ctx.agents / ctx.subagents / ctx.llm / ctx.systemPrompt / ctx.commands /
//   ctx.logger / ctx.get('webServer'|'workspaceRegistry')）。plugins.json 条目
//   runtime=="dsh" 时走本轨：import('@deepseek-ai/cordis') new Context() +
//   DSH 门面（decorateDshCtx）；其余条目沿用 cordis3 轨（@cordisjs/core）。
//   双轨各自独立 import，node_modules 平铺共存无冲突。
//
// 协议：stdin/stdout JSON Lines（每行一个 JSON 对象，不换行转义）。
//   Go → Node: {"t":"invoke","id":N,"tool":"xx","args":{...},"convId":"...","wsRoot":"..."}
//              {"t":"service","id":N,"svc":"fs|web|bash|...","method":"xx","args":{...}}
//              {"t":"ping","id":N}
//              {"t":"cmdrun","id":N,"name":"agent-teams","args":{rawInput,...}}
//              {"t":"event","name":"agent/status","payload":{...}}
//   Node → Go: {"t":"ready","plugins":[...],"tools":[...]}
//              {"t":"tool","plugin":"pkg","def":{name,description,parameters,...}}
//              {"t":"result","id":N,"ok":true,"data":"..."}
//              {"t":"result","id":N,"ok":false,"error":"..."}
//              {"t":"subscribe","plugin":"pkg","events":["agent/status",...]}
//              {"t":"log","level":"info","msg":"..."}
//
// 环境变量：CORDIS_BRIDGE_DIR（bridge 工作目录，含 plugins.json +
// node_modules）、CORDIS_WORKSPACE_ROOT（工作区根，暴露给 ctx.workspaceRoot）。
// ═══════════════════════════════════════════════════════════
'use strict';
const fs = require('node:fs');
const path = require('node:path');
const readline = require('node:readline');

const BRIDGE_DIR = process.env.CORDIS_BRIDGE_DIR || '.';
const PLUGINS_FILE = path.join(BRIDGE_DIR, 'plugins.json');
const WORKSPACE_ROOT = process.env.CORDIS_WORKSPACE_ROOT || '.';

function send(msg) {
  process.stdout.write(JSON.stringify(msg) + '\n');
}

// console → 统一写 stderr（不污染 stdout 协议）+ 上报 Go 侧日志
for (const m of ['log', 'info', 'warn', 'error']) {
  console[m] = (...a) => {
    const text = a.map((x) => (typeof x === 'string' ? x : safeString(x))).join(' ');
    try { process.stderr.write(text + '\n'); } catch (_) {}
    try { send({ t: 'log', level: m, msg: text }); } catch (_) {}
  };
}
function safeString(x) {
  try { return JSON.stringify(x); } catch (_) { return String(x); }
}

// ── 服务代理（ctx.fs/web/bash/agents/... → Go 侧 Registry）──
let seq = 0;
const pending = new Map(); // id → {resolve, reject, timer}
function svc(svcName, method, args, timeoutMs = 90000) {
  const id = ++seq;
  return new Promise((resolve, reject) => {
    const timer = setTimeout(() => {
      if (pending.delete(id)) reject(new Error(`服务 ${svcName}.${method} 调用超时（${timeoutMs}ms）`));
    }, timeoutMs);
    pending.set(id, { resolve, reject, timer });
    send({ t: 'service', id, svc: svcName, method, args: args || {}, plugin: currentPlugin || '' });
  });
}

// ── 工具注册表 ──
const tools = new Map(); // name → def（含 handler/run 执行函数；__svc 为服务型调用）

// ── 命令注册表（DSH ctx.commands；handler 由 Go 侧 cmdrun 触发）──
const commands = new Map(); // name → {name, description, handler}

// ── 事件监听表（DSH ctx.on；Go 侧按订阅名转发 host 事件）──
const eventListeners = new Map(); // name → Set<fn>

// 当前装载插件名（svc/subscribe 消息归属用）。
let currentPlugin = '';

function sendSubscriptions() {
  const names = [...eventListeners.keys()];
  if (names.length > 0) {
    send({ t: 'subscribe', plugin: currentPlugin || '', events: names });
  }
}

// ── 服务型插件工具暴露 ──
// 插件经 ctx.provide(name, obj) 提供服务对象（如 cordis-plugin-android 的
// ctx.provide('android', bridge)）——服务对象的方法就是可调用能力。
// 把每个方法暴露为一个工具 <service>_<method>，agent 可直接 function-call。
function sanitizeToolName(s) {
  return String(s).replace(/[^a-zA-Z0-9_]/g, '_').replace(/_+/g, '_').replace(/^_|_$/g, '');
}
function paramNames(fn) {
  let src;
  try { src = Function.prototype.toString.call(fn); } catch (_) { return []; }
  const m = src.match(/^[\s\S]*?\(\s*([^)]*)\)/);
  if (!m) return [];
  return m[1].split(',').map((s) => s.trim().split(/[=:]/)[0]).filter(Boolean);
}
function ownMethods(obj) {
  const names = new Set();
  let cur = obj;
  while (cur && cur !== Object.prototype && cur !== Function.prototype) {
    for (const n of Object.getOwnPropertyNames(cur)) {
      if (n === 'constructor') continue;
      try {
        if (typeof obj[n] === 'function') names.add(n);
      } catch (_) {}
    }
    cur = Object.getPrototypeOf(cur);
  }
  return [...names];
}
function exposeService(plugin, serviceName, obj) {
  const base = sanitizeToolName(serviceName);
  for (const method of ownMethods(obj)) {
    const tname = sanitizeToolName(`${base}_${method}`);
    if (!tname) continue;
    const names = paramNames(obj[method]);
    const parameters = {
      type: 'object',
      properties: Object.fromEntries(names.map((n) => [n, { type: 'string', description: `参数 ${n}` }])),
      additionalProperties: true,
    };
    const def = {
      name: tname,
      description: `插件服务 ${serviceName}.${method}()（插件 ctx.provide('${serviceName}') 暴露）${names.length ? '，参数：' + names.join(', ') : ''}`,
      parameters,
      category: 'plugin-service',
      __svc: { obj, method, names },
    };
    tools.set(tname, def); // 同服务名多插件：后 provide 覆盖（与 cordis 语义一致）
    send({
      t: 'tool', plugin,
      def: {
        name: tname,
        description: def.description,
        parameters,
        category: 'plugin-service',
      },
    });
  }
}

// ── 消息文本提取（DSH UserMessage → 纯文本；createUserMessage 形态）──
function textOfMessage(msg) {
  if (msg === undefined || msg === null) return '';
  if (typeof msg === 'string') return msg;
  if (Array.isArray(msg.content)) {
    return msg.content
      .filter((b) => b && typeof b === 'object' && b.type === 'text' && typeof b.text === 'string')
      .map((b) => b.text)
      .join('\n');
  }
  if (typeof msg.content === 'string') return msg.content;
  return '';
}

// ── DSH Agent 句柄（ctx.agents.get / exec.agent）──
// 宿主会话 = 一个可续聊子 Agent（convID 标识）。句柄是轻量投影：
// 状态经 Go 服务查询；followup/inject/steer/cancel 转发 Go 侧对应能力。
function makeAgent(convId, wsRoot, status, currentRoute) {
  const route = currentRoute || null;
  const session = {
    header: {
      cwd: wsRoot || WORKSPACE_ROOT,
      parentSession: undefined,
      seedLength: 0,
    },
    events: [],
    requestHeader: () => (route && route.provider && route.model
      ? { config: { provider: route.provider, model: route.model, ...(route.reasoningEffort ? { reasoningEffort: route.reasoningEffort } : {}) } }
      : undefined),
    append: () => {}, // 会话事件面（agent-teams/team-*）：宿主不落会话事件，无副作用
  };
  const agent = {
    id: convId || '',
    status: status || 'ready',
    session,
    options: { provider: route?.provider, model: route?.model },
    followup: (msg) => svc('agents', 'followup', { convId, text: textOfMessage(msg) }),
    inject: (msg) => svc('agents', 'inject', { convId, text: textOfMessage(msg) }),
    steer: (msg) => svc('agents', 'steer', { convId, text: textOfMessage(msg) }),
    cancel: (reason, opts) => svc('agents', 'cancel', { convId, reason: reason || {}, keepInbox: !!(opts && opts.keepInbox) }),
    whenIdle: async () => {
      // 轮询宿主运行态直到非 running（未登记会话视为已 idle）。
      for (let i = 0; i < 1200; i++) {
        try {
          const r = await svc('agents', 'running', { convId });
          if (String(r).trim() !== 'true') return;
        } catch (_) { return; }
        await new Promise((res) => setTimeout(res, 500));
      }
    },
  };
  return agent;
}

// 缓存 DSH Agent 句柄（ctx.agents.get 同步读；list/status 事件/startContinuable 刷新）。
const agentCache = new Map(); // convId → agent
let currentRoute = null; // 宿主当前模型路由（llm.current 缓存）

async function refreshCurrentRoute() {
  try {
    const r = await svc('llm', 'current', {});
    if (r && typeof r === 'object') currentRoute = r;
  } catch (_) {}
}

async function refreshAgentCache() {
  try {
    const list = await svc('agents', 'list', {});
    if (Array.isArray(list)) {
      for (const rec of list) {
        if (rec && typeof rec === 'object' && rec.convId) {
          agentCache.set(rec.convId, makeAgent(rec.convId, rec.wsRoot, rec.state, currentRoute));
        }
      }
    }
  } catch (_) {}
}

// ── decorateCtx（cordis3 轨）给插件 apply 的 ctx 挂 harness 门面 ──
// （对齐 deepseek-harness host-runner）：tools.register / fs / web / bash /
// workspaceRoot / store / loop。
function decorateCtx(ctx, plugin) {
  currentPlugin = plugin;
  // 拦截 ctx.provide：服务型插件（无 tools.register）也能被 agent 调用
  const origProvide = typeof ctx.provide === 'function' ? ctx.provide.bind(ctx) : null;
  if (origProvide) {
    ctx.provide = (name, obj, ...rest) => {
      const r = origProvide(name, obj, ...rest);
      try {
        if (obj && (typeof obj === 'object' || typeof obj === 'function')) {
          exposeService(plugin, name, obj);
        }
      } catch (e) {
        console.warn(`[bridge] 服务 ${name} 工具暴露失败:`, (e && e.message) || e);
      }
      return r;
    };
  }
  ctx.tools = {
    register(def) {
      if (!def || typeof def.name !== 'string' || !def.name) throw new Error('tool 缺少 name');
      if (tools.has(def.name)) throw new Error(`工具名冲突: ${def.name}（已被其他插件注册）`);
      const fn = def.handler || def.run || def.fn || def.execute || def.call;
      if (typeof fn !== 'function') throw new Error(`工具 ${def.name} 缺少执行函数（handler/run/fn）`);
      tools.set(def.name, def);
      send({
        t: 'tool', plugin,
        def: {
          name: def.name,
          description: def.description || '',
          parameters: def.parameters || def.params || def.schema || undefined,
          category: def.category || 'plugin',
          readOnly: !!def.readOnly,
        },
      });
      return def;
    },
    list() { return [...tools.keys()]; },
  };
  ctx.fs = {
    read: (p, opts) => svc('fs', 'read', Object.assign({ path: p }, opts || {})),
    write: (p, content) => svc('fs', 'write', { path: p, content }),
    list: (p, opts) => svc('fs', 'list', Object.assign({ path: p || '.' }, opts || {})),
    exists: (p) => svc('fs', 'exists', { path: p }),
    mkdir: (p) => svc('fs', 'mkdir', { path: p }),
    remove: (p) => svc('fs', 'remove', { path: p }),
  };
  ctx.web = {
    fetch: (url, opts) => svc('web', 'fetch', { url, opts }),
    search: (q) => svc('web', 'search', { query: q }),
  };
  ctx.bash = {
    exec: (command, opts) => svc('bash', 'exec', Object.assign({ command }, opts || {})),
  };
  // ★ 2026-08-27 Node 桥能力扩展：消息落盘（store）+ 循环状态/上下文快照（loop）
  //   ——Node 插件可读/写会话消息（数据落盘逻辑改变）、感知/干预 agentloop。
  ctx.store = {
    read: (convId, opts) => svc('store', 'read', Object.assign({ convId }, opts || {})),
    append: (convId, role, content, opts) => svc('store', 'append', Object.assign({ convId, role, content }, opts || {})),
  };
  ctx.loop = {
    info: () => svc('loop', 'info', {}),
    snapshot: (convId, opts) => svc('loop', 'snapshot', Object.assign({ convId }, opts || {})),
  };
  ctx.workspaceRoot = WORKSPACE_ROOT;
  return ctx;
}

// ── decorateDshCtx（cordis4 轨）DSH 服务面门面 ──
// 对齐 deepseek-harness 插件运行时：agents / subagents / llm / systemPrompt /
// commands / logger / tools / get('webServer'|'workspaceRegistry') / effect /
// on（host 事件订阅桥）/ inject（服务就绪即同步回调）。
function decorateDshCtx(ctx, plugin) {
  currentPlugin = plugin;
  ctx.workspaceRoot = WORKSPACE_ROOT;
  ctx.tools = {
    register(def) {
      if (!def || typeof def.name !== 'string' || !def.name) throw new Error('tool 缺少 name');
      if (tools.has(def.name)) throw new Error(`工具名冲突: ${def.name}（已被其他插件注册）`);
      const fn = def.handler || def.run || def.fn || def.execute || def.call;
      if (typeof fn !== 'function') throw new Error(`工具 ${def.name} 缺少执行函数（handler/run/fn/execute）`);
      tools.set(def.name, def);
      send({
        t: 'tool', plugin,
        def: {
          name: def.name,
          description: def.description || '',
          parameters: def.parameters || def.params || def.schema || undefined,
          category: def.category || 'plugin',
          readOnly: !!def.readOnly,
        },
      });
      return def;
    },
    list() { return [...tools.keys()]; },
  };
  ctx.fs = {
    read: (p, opts) => svc('fs', 'read', Object.assign({ path: p }, opts || {})),
    write: (p, content) => svc('fs', 'write', Object.assign({ path: p, content }, {})),
    list: (p, opts) => svc('fs', 'list', Object.assign({ path: p || '.' }, opts || {})),
    exists: (p) => svc('fs', 'exists', { path: p }),
    mkdir: (p) => svc('fs', 'mkdir', { path: p }),
    remove: (p) => svc('fs', 'remove', { path: p }),
  };
  ctx.web = {
    fetch: (url, opts) => svc('web', 'fetch', { url, opts }),
    search: (q) => svc('web', 'search', { query: q }),
  };
  ctx.bash = {
    exec: (command, opts) => svc('bash', 'exec', Object.assign({ command }, opts || {})),
  };
  ctx.store = {
    read: (convId, opts) => svc('store', 'read', Object.assign({ convId }, opts || {})),
    append: (convId, role, content, opts) => svc('store', 'append', Object.assign({ convId, role, content }, opts || {})),
  };
  ctx.loop = {
    info: () => svc('loop', 'info', {}),
    snapshot: (convId, opts) => svc('loop', 'snapshot', Object.assign({ convId }, opts || {})),
  };

  // logger：经 console 通道（已上报 Go 日志轨；不再走 svc 往返）
  const logTo = (level, args) => {
    const text = [...args].map((x) => (typeof x === 'string' ? x : safeString(x))).join(' ');
    try { (console[level] || console.log)(`[${plugin}] ${text}`); } catch (_) {}
  };
  ctx.logger = {
    info: (...a) => logTo('info', a),
    warn: (...a) => logTo('warn', a),
    error: (...a) => logTo('error', a),
    debug: (...a) => logTo('log', a),
    success: (...a) => logTo('info', a),
  };

  // systemPrompt：段注册 → Go 侧 PluginHost（组装进系统提示）
  ctx.systemPrompt = {
    section({ name, order, text }) {
      if (!name) throw new Error('systemPrompt.section 缺少 name');
      svc('systemPrompt', 'section', {
        name,
        order: typeof order === 'number' ? order : 100,
        text: typeof text === 'function' ? text() : String(text || ''),
      }).catch(() => {});
      return () => {};
    },
    variable({ name, provider }) {
      if (!name) throw new Error('systemPrompt.variable 缺少 name');
      // 变量求值需 Node → Go 回调通道；当前宿主无消费方，注册即忽略（保持 API 兼容）。
      if (typeof provider === 'function') {
        try { provider(); } catch (_) {}
      }
      return () => {};
    },
  };

  // commands：注册进宿主命令表；执行时 Go 侧发 cmdrun 回来调 handler
  ctx.commands = {
    register(def) {
      if (!def || !def.name) throw new Error('commands.register 缺少 name');
      if (commands.has(def.name)) throw new Error(`命令名冲突: ${def.name}`);
      commands.set(def.name, def);
      svc('commands', 'register', { name: def.name, description: def.description || '' }).catch(() => {});
      return () => {
        if (commands.get(def.name) === def) {
          commands.delete(def.name);
          svc('commands', 'unregister', { name: def.name }).catch(() => {});
        }
      };
    },
    list() { return [...commands.keys()]; },
  };

  // agents：宿主子 Agent 编排面（Go 侧 SubAgentRegistry）
  ctx.agents = {
    get(id) {
      return agentCache.get(id);
    },
  };

  // subagents：成员派生面（Go 侧 SpawnSubAgent/FollowupSubAgent/StopSubAgent）
  ctx.subagents = {
    getProvider(name) {
      if (name === 'spawn') {
        return {
          name: 'spawn',
          prepareContinuable: () => undefined,
          capabilities: { persona: true, toolFilter: true },
        };
      }
      return undefined;
    },
    list() { return ['spawn']; },
    async startContinuable({ provider, label, request, signal }) {
      const persona = request && request.persona;
      const prompt = request && request.prompt ? textOfMessage({ content: request.prompt }) : '';
      const deny = request && request.toolFilter && Array.isArray(request.toolFilter.deny) ? request.toolFilter.deny : undefined;
      const agentOptions = request && request.agentOptions ? request.agentOptions : {};
      const parentConvId = request && request.parent ? request.parent.id : '';
      const res = await svc('subagents', 'startContinuable', {
        provider: provider || 'spawn',
        label: label || '',
        prompt,
        persona,
        parentConvId,
        provider2: agentOptions.provider,
        model: agentOptions.model,
        maxDepth: request && request.maxDepth,
        denyTools: deny,
      }, 120000);
      const childId = res && (res.childId || res.convId);
      if (childId) {
        agentCache.set(childId, makeAgent(childId, request && request.parent ? request.parent.session.header.cwd : WORKSPACE_ROOT, 'running', currentRoute));
      }
      return { childId };
    },
    async followup(parent, childId, content, options) {
      await svc('subagents', 'followup', { childId, text: textOfMessage({ content }) }, 120000);
    },
    async interrupt(childId, reason) {
      await svc('subagents', 'interrupt', { childId, reason: reason || {} }).catch((e) => {
        ctx.logger.warn(`subagents.interrupt ${childId} 失败: ${String(e && e.message || e)}`);
      });
    },
    registerContinuableSetup(cb) {
      // 子会话模型选择运行时：宿主在 startContinuable 直接透传 provider/model，
      // 无需回调装配；保留注册面（调用方 dispose 语义）。
      return () => {};
    },
  };

  // llm：模型目录/调用配置解析（Go 侧模型目录）
  ctx.llm = {
    async listModels(provider) {
      const r = await svc('llm', 'listModels', { provider: provider || '' });
      return Array.isArray(r) ? r : [];
    },
    async resolveCallConfig(cfg, signal) {
      const r = await svc('llm', 'resolveCallConfig', cfg || {});
      if (r && typeof r === 'object' && (r.provider || r.model)) return r;
      return cfg || {};
    },
  };

  // get：可选服务面（webServer/workspaceRegistry 宿主未装配 → undefined，
  // 插件保持 tool-only 不阻塞装载——与 headless profile 语义一致）
  const provided = new Set(['tools', 'llm', 'subagents', 'systemPrompt', 'agents', 'commands', 'logger', 'fs', 'web', 'bash', 'store', 'loop']);
  const getService = (name) => {
    if (name === 'tools') return ctx.tools;
    if (name === 'llm') return ctx.llm;
    if (name === 'subagents') return ctx.subagents;
    if (name === 'systemPrompt') return ctx.systemPrompt;
    if (name === 'agents') return ctx.agents;
    if (name === 'commands') return ctx.commands;
    if (name === 'logger') return ctx.logger;
    if (name === 'fs') return ctx.fs;
    if (name === 'web') return ctx.web;
    if (name === 'bash') return ctx.bash;
    if (name === 'store') return ctx.store;
    if (name === 'loop') return ctx.loop;
    return undefined;
  };
  ctx.get = (name) => (provided.has(name) ? getService(name) : undefined);

  // effect：立即执行并收集清理回调（桥生命周期 = 进程生命周期）
  const effects = [];
  ctx.effect = (fn, label) => {
    try {
      const cleanup = fn();
      if (typeof cleanup === 'function') effects.push(cleanup);
    } catch (e) {
      console.error(`[bridge] effect(${label || '?'}) 执行失败:`, (e && e.stack) || e);
    }
    return () => {};
  };

  // on：host 事件订阅桥（Go 按订阅白名单转发 agent/* 等事件）
  ctx.on = (name, fn) => {
    if (typeof fn !== 'function') return () => {};
    if (!eventListeners.has(name)) eventListeners.set(name, new Set());
    eventListeners.get(name).add(fn);
    sendSubscriptions();
    return () => {
      const set = eventListeners.get(name);
      if (set) {
        set.delete(fn);
        if (set.size === 0) {
          eventListeners.delete(name);
          sendSubscriptions();
        }
      }
    };
  };

  // inject：全部依赖已就绪 → 同步回调（宿主服务静态装配，不 pend fiber）
  ctx.inject = (list, callback) => {
    const missing = (list || []).filter((name) => !provided.has(name));
    if (missing.length === 0 && typeof callback === 'function') {
      try {
        callback(ctx);
      } catch (e) {
        console.error(`[bridge] inject callback 失败:`, (e && e.stack) || e);
      }
    }
    return () => {};
  };

  // 异步预热：模型路由 + 成员句柄缓存
  void refreshCurrentRoute();
  void refreshAgentCache();
  return ctx;
}

async function invokeTool(toolName, args, convId, wsRoot) {
  const def = tools.get(toolName);
  if (!def) throw new Error(`桥接工具不存在: ${toolName}`);
  // 服务型调用（ctx.provide 暴露）：按参数名展开调用服务方法
  if (def.__svc) {
    const { obj, method, names } = def.__svc;
    let result;
    if (names.length > 0) {
      result = obj[method](...names.map((n) => (args || {})[n]));
    } else {
      result = obj[method]();
    }
    if (result && typeof result.then === 'function') result = await result;
    return stringifyResult(result);
  }
  const fn = def.handler || def.run || def.fn || def.execute || def.call;
  if (typeof fn !== 'function') throw new Error(`工具 ${toolName} 没有可执行函数`);
  // DSH 工具执行上下文：exec.agent = 调用会话句柄（宿主导航信息经 convId/wsRoot 注入）
  const exec = {
    agent: makeAgent(convId || '', wsRoot || WORKSPACE_ROOT, 'running', currentRoute),
    signal: new AbortController().signal,
  };
  if (convId) agentCache.set(convId, exec.agent);
  const result = await fn(args || {}, exec);
  return stringifyResult(result);
}

// stringifyResult 结果序列化（>2MB 截断防协议撑爆；截图类 base64 提示落盘）。
function stringifyResult(result) {
  if (typeof result === 'string') return result;
  if (result === undefined || result === null) return '';
  let s;
  try { s = JSON.stringify(result); } catch (_) { s = String(result); }
  if (s.length > 2 * 1024 * 1024) {
    console.warn('[bridge] 工具结果过大（' + s.length + ' 字符）已截断——大图建议经 ctx.fs 写入文件');
    return s.slice(0, 2 * 1024 * 1024) + '\n...[结果过大已截断]';
  }
  return s;
}

// ── plugins.json 条目解析（兼容字符串与 {spec, runtime} 对象）──
function readPluginSpecs() {
  let raw;
  try {
    raw = JSON.parse(fs.readFileSync(PLUGINS_FILE, 'utf8')).plugins || [];
  } catch (e) {
    console.error('[bridge] plugins.json 读取失败（桥接目录无插件配置）:', e.message);
    return [];
  }
  return raw.map((entry) => {
    if (typeof entry === 'string') return { spec: entry, runtime: 'node' };
    if (entry && typeof entry === 'object' && typeof entry.spec === 'string') {
      return { spec: entry.spec, runtime: entry.runtime || 'node' };
    }
    return null;
  }).filter(Boolean);
}

// ── 插件装载 ──
// runtime=="dsh" → cordis4（@deepseek-ai/cordis）+ DSH 门面；
// 其余 → cordis3（@cordisjs/core）既有路径。

// resolvePluginApply 从模块导出中解析 apply 函数（双轨共用）。
// ★ t5 集成修复：candidate 可能是「类/服务库」形态（如 @deepseek-ai/dsh-fs-local
//   导出 LocalFileSystem 类）——类经原型链取到 Function.prototype.apply 会以
//   诡异 TypeError 崩溃；这里仅接受：① 非 class 的函数（cordis 惯例 default=apply），
//   ② 对象 own property apply/default。类与空导出 → undefined（调用方友好跳过）。
function resolvePluginApply(candidate) {
  if (typeof candidate === 'function') {
    try {
      const src = Function.prototype.toString.call(candidate);
      if (/^\s*class\b/u.test(src)) return undefined; // 类构造器不是 apply 函数
    } catch (_) {}
    return candidate;
  }
  if (candidate && typeof candidate === 'object') {
    if (Object.hasOwn(candidate, 'apply') && typeof candidate.apply === 'function') {
      return candidate.apply;
    }
    if (typeof candidate.default === 'function') {
      return candidate.default;
    }
  }
  return undefined;
}

async function loadPlugins() {
  const specs = readPluginSpecs();
  const loaded = [];
  for (const spec of specs) {
    const at = String(spec.spec).lastIndexOf('@');
    const pkgName = at > 0 ? String(spec.spec).slice(0, at) : String(spec.spec);
    const ver = at > 0 ? String(spec.spec).slice(at + 1) : '';
    try {
      let applyFn;
      if (spec.runtime === 'dsh') {
        // ★ Round4：cordis4 + DSH 服务面装载分支
        const { Context } = await import('@deepseek-ai/cordis');
        const mod = await import(pkgName);
        const candidate = mod.default || mod;
        applyFn = resolvePluginApply(candidate);
        if (typeof applyFn !== 'function') {
          console.error(`[bridge] 插件 ${pkgName} 无 apply 函数（导出形态不支持：类/服务库/空导出），跳过`);
          continue;
        }
        const ctx = decorateDshCtx(new Context(), pkgName);
        await applyFn(ctx, {});
      } else {
        const { Context } = await import('@cordisjs/core');
        const mod = await import(pkgName);
        const candidate = mod.default || mod;
        applyFn = resolvePluginApply(candidate);
        if (typeof applyFn !== 'function') {
          console.error(`[bridge] 插件 ${pkgName} 无 apply 函数（导出形态不支持：类/服务库/空导出），跳过`);
          continue;
        }
        const ctx = decorateCtx(new Context(), pkgName);
        const wrapped = (sub) => applyFn(decorateCtx(sub || ctx, pkgName));
        const pluginObj = (typeof candidate === 'object' && candidate && Object.hasOwn(candidate, 'apply'))
          ? candidate
          : { apply: wrapped, name: pkgName };
        if (typeof ctx.plugin === 'function') {
          ctx.plugin(pluginObj);
        } else {
          await wrapped(ctx);
        }
      }
      loaded.push(spec.spec);
      console.log(`[bridge] 已装载插件 ${pkgName}${ver ? '@' + ver : ''}（runtime=${spec.runtime}）`);
    } catch (e) {
      console.error(`[bridge] 插件 ${pkgName} 装载失败:`, e && e.stack || e);
    }
  }
  return loaded;
}

(async () => {
  const loaded = await loadPlugins();
  send({ t: 'ready', plugins: loaded, tools: [...tools.keys()] });

  const rl = readline.createInterface({ input: process.stdin, crlfDelay: Infinity });
  rl.on('line', async (line) => {
    let msg;
    try { msg = JSON.parse(line); } catch (_) { return; }
    try {
      if (msg.t === 'invoke') {
        try {
          const data = await invokeTool(msg.tool, msg.args, msg.convId, msg.wsRoot);
          send({ t: 'result', id: msg.id, ok: true, data });
        } catch (e) {
          send({ t: 'result', id: msg.id, ok: false, error: String((e && e.stack) || e) });
        }
      } else if (msg.t === 'ping') {
        send({ t: 'result', id: msg.id, ok: true, data: 'pong' });
      } else if (msg.t === 'cmdrun') {
        // Go 侧执行宿主命令 → 回 Node 调用插件命令 handler（DSH ctx.commands）
        try {
          const def = commands.get(msg.name);
          if (!def) throw new Error(`命令未注册: ${msg.name}`);
          const args = msg.args || {};
          const invocation = {
            rawInput: String(args.rawInput || ''),
            agent: makeAgent(String(args.convId || ''), String(args.wsRoot || WORKSPACE_ROOT), 'running', currentRoute),
          };
          const result = await def.handler(invocation);
          send({ t: 'result', id: msg.id, ok: true, data: stringifyResult(result) });
        } catch (e) {
          send({ t: 'result', id: msg.id, ok: false, error: String((e && e.stack) || e) });
        }
      } else if (msg.t === 'event') {
        // Go → Node host 事件（agent/status 等）：按订阅名分发
        const set = eventListeners.get(msg.name);
        if (set) {
          for (const fn of [...set]) {
            try { await fn(msg.payload); } catch (e) {
              console.error(`[bridge] 事件 ${msg.name} handler 异常:`, (e && e.stack) || e);
            }
          }
        }
      } else if (msg.t === 'result') {
        // Go → Node 的 service 响应：resolve 对应服务 promise（fs/web/bash 转发结果）
        const p = pending.get(msg.id);
        if (p) {
          pending.delete(msg.id);
          clearTimeout(p.timer);
          if (msg.ok) p.resolve(msg.data);
          else p.reject(new Error(String(msg.error || '服务调用失败')));
        }
      }
    } catch (e) {
      send({ t: 'log', level: 'error', msg: '协议错误: ' + ((e && e.stack) || e) });
    }
  });
  rl.on('close', () => process.exit(0));
})();
