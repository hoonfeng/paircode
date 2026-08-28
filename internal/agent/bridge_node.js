// ═══════════════════════════════════════════════════════════
// bridge_node.js — Node 运行时桥（真实 node 进程执行 npm cordis 插件）
//
// 用途：goja 沙箱只能执行无 npm 依赖的插件；依赖 npm 生态（dotenv/
// axios/undici 等）的 cordis 插件在真实 node 中运行，通过本桥与 Go
// 主进程通信。插件可用 ctx.tools.register 注册工具（进 Go 工具表，
// agent 可直接调用）；ctx.fs/web/bash 服务转发 Go 侧对应工具实现
// （复用 read_file/web_fetch/run_command 等，行为与 goja 插件一致）。
//
// 协议：stdin/stdout JSON Lines（每行一个 JSON 对象，不换行转义）。
//   Go → Node: {"t":"invoke","id":N,"tool":"xx","args":{...}}
//              {"t":"service","id":N,"svc":"fs|web|bash","method":"xx","args":{...}}
//              {"t":"ping","id":N}
//   Node → Go: {"t":"ready","plugins":[...],"tools":[...]}
//              {"t":"tool","plugin":"pkg","def":{name,description,parameters,...}}
//              {"t":"result","id":N,"ok":true,"data":"..."}
//              {"t":"result","id":N,"ok":false,"error":"..."}
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

// ── 服务代理（ctx.fs/web/bash → Go 侧 Registry）──
let seq = 0;
const pending = new Map(); // id → {resolve, reject, timer}
function svc(svcName, method, args, timeoutMs = 90000) {
  const id = ++seq;
  return new Promise((resolve, reject) => {
    const timer = setTimeout(() => {
      if (pending.delete(id)) reject(new Error(`服务 ${svcName}.${method} 调用超时（${timeoutMs}ms）`));
    }, timeoutMs);
    pending.set(id, { resolve, reject, timer });
    send({ t: 'service', id, svc: svcName, method, args: args || {} });
  });
}

// ── 工具注册表 ──
const tools = new Map(); // name → def（含 handler/run 执行函数；__svc 为服务型调用）

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

// decorateCtx 给插件 apply 的 ctx 挂 harness 门面（对齐 deepseek-harness
// host-runner）：tools.register / fs / web / bash / workspaceRoot。
function decorateCtx(ctx, plugin) {
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
  ctx.workspaceRoot = process.env.CORDIS_WORKSPACE_ROOT || '.';
  return ctx;
}

async function invokeTool(toolName, args) {
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
  const result = await fn(args || {});
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

// ── 插件装载（真实 cordis 运行时）──
async function loadPlugins() {
  let specs = [];
  try {
    specs = (JSON.parse(fs.readFileSync(PLUGINS_FILE, 'utf8')).plugins || []);
  } catch (e) {
    console.error('[bridge] plugins.json 读取失败（桥接目录无插件配置）:', e.message);
    return [];
  }
  const loaded = [];
  for (const spec of specs) {
    const at = String(spec).lastIndexOf('@');
    const pkgName = at > 0 ? String(spec).slice(0, at) : String(spec);
    const ver = at > 0 ? String(spec).slice(at + 1) : '';
    try {
      const { Context } = await import('@cordisjs/core');
      const mod = await import(pkgName);
      const candidate = mod.default || mod;
      const applyFn = typeof candidate === 'function'
        ? candidate
        : (candidate && (candidate.apply || candidate.default));
      if (typeof applyFn !== 'function') {
        console.error(`[bridge] 插件 ${pkgName} 无 apply 函数（导出形态不支持），跳过`);
        continue;
      }
      const ctx = decorateCtx(new Context(), pkgName);
      const wrapped = (sub) => applyFn(decorateCtx(sub || ctx, pkgName));
      const pluginObj = (typeof candidate === 'object' && candidate && candidate.apply)
        ? candidate
        : { apply: wrapped, name: pkgName };
      if (typeof ctx.plugin === 'function') {
        ctx.plugin(pluginObj);
      } else {
        await wrapped(ctx);
      }
      loaded.push(spec);
      console.log(`[bridge] 已装载插件 ${pkgName}${ver ? '@' + ver : ''}`);
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
          const data = await invokeTool(msg.tool, msg.args);
          send({ t: 'result', id: msg.id, ok: true, data });
        } catch (e) {
          send({ t: 'result', id: msg.id, ok: false, error: String((e && e.stack) || e) });
        }
      } else if (msg.t === 'ping') {
        send({ t: 'result', id: msg.id, ok: true, data: 'pong' });
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
