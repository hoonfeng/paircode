#!/usr/bin/env node
/**
 * sse-mcp-repl.js — 常驻 MCP SSE 客户端（由 Agent 通过 stdin 驱动）
 *
 * 背景：IDE 的 mcp_add 只支持 stdio 传输，而本服务是 SSE 传输（event: endpoint → /message?sessionId=…）。
 * 本脚本直连 SSE 端点，维持**同一个 MCP session**（浏览器状态因此连续），并把每次响应分行打印，
 * 便于 Agent 用 write_stdin 分步读取；工具返回的图片自动落盘，供 read_image 目视验证。
 *
 * 用法:
 *   node scripts/sse-mcp-repl.js <SSE_URL>
 *
 * stdin 逐行读 JSON 命令（每行一条）:
 *   {"method":"tools/list"}                              → 直接转发 JSON-RPC
 *   {"tool":"browser_navigate","args":{...}}             → 转为 tools/call
 *   {"tool":"browser_navigate","argsFile":"a.json"}      → args 从文件读（中文文案避开 stdin 编码风险）
 *   {"ping":true}                                        → 存活探测
 *   可选 "timeoutMs":N 覆盖单次请求超时（默认 MCP_TIMEOUT_MS 或 120000）
 *
 * stdout 约定:
 *   ### SSE_CONNECTED / ### ENDPOINT <url> / ### INITIALIZED
 *   ### CALL tool=… / ### REQ method=…
 *   ### IMAGE_SAVED <paths>
 *   ### RESP id=N kind=result|error len=…  → 后跟 JSON（超 8000 字符截断）→ ### END id=N
 */
'use strict';

const fs = require('fs');
const path = require('path');
const readline = require('readline');

const SSE_URL = process.argv[2];
if (!SSE_URL) {
  console.error('usage: node scripts/sse-mcp-repl.js <SSE_URL>');
  process.exit(2);
}

const ORIGIN = new URL(SSE_URL).origin;
const SHOT_DIR = path.resolve(process.cwd(), 'screenshots');
const DEFAULT_TIMEOUT = Number(process.env.MCP_TIMEOUT_MS || 120000);
const MAX_BODY = 8000;

let messageUrl = null;
// 从 1000 起编号：服务端自身会用 1,2,3… 主动发 ping/请求，错开可避免 JSON-RPC id 撞号
let requestSeq = 1000;
let shotSeq = 0;
const pending = new Map();
let resolveReady;
const readyPromise = new Promise((r) => {
  resolveReady = r;
});

function log(...a) {
  process.stdout.write(a.join(' ') + '\n');
}

/** 工具结果里的 image content 落盘，返回原结果（已剔除 base64） */
function extractImages(result) {
  if (!result || !Array.isArray(result.content)) return result;
  const saved = [];
  for (const c of result.content) {
    if (c && c.type === 'image' && typeof c.data === 'string') {
      fs.mkdirSync(SHOT_DIR, { recursive: true });
      const ext = /jpe?g/i.test(c.mimeType || '') ? 'jpg' : 'png';
      shotSeq += 1;
      const p = path.join(SHOT_DIR, `mcp-${Date.now()}-${shotSeq}.${ext}`);
      try {
        fs.writeFileSync(p, Buffer.from(c.data, 'base64'));
        saved.push(p);
      } catch (e) {
        log('### IMAGE_ERR ' + e.message);
      }
      delete c.data;
      c._savedTo = p;
    }
  }
  if (saved.length) log('### IMAGE_SAVED ' + saved.join(' '));
  return result;
}

function finish(id, payload, kind) {
  const p = pending.get(id);
  if (!p) return; // 服务端自发的通知式响应，忽略
  pending.delete(id);
  clearTimeout(p.timer);
  const json = JSON.stringify(payload);
  log(`### RESP id=${id} kind=${kind} len=${json.length}`);
  log(json.length > MAX_BODY ? json.slice(0, MAX_BODY) + `\n...[truncated ${json.length - MAX_BODY} chars]` : json);
  log('### END id=' + id);
  p.resolve();
}

/** 解析并处理一条 SSE 事件块 */
function handleEvent(raw) {
  let ev = 'message';
  const dataLines = [];
  for (const line of raw.split('\n')) {
    if (line.startsWith('event:')) ev = line.slice(6).trim();
    else if (line.startsWith('data:')) dataLines.push(line.slice(5).replace(/^ /, ''));
  }
  const data = dataLines.join('\n');

  if (ev === 'endpoint') {
    messageUrl = data.startsWith('http') ? data : ORIGIN + (data.startsWith('/') ? data : '/' + data);
    log('### ENDPOINT ' + messageUrl);
    if (resolveReady) {
      resolveReady(messageUrl);
      resolveReady = null;
    }
    return;
  }
  if (!data) return;

  let msg;
  try {
    msg = JSON.parse(data);
  } catch {
    log('### RAW ev=' + ev + ' ' + data.slice(0, 300));
    return;
  }
  if (msg.id !== undefined && (msg.result !== undefined || msg.error !== undefined)) {
    finish(msg.id, msg.error ? { error: msg.error } : extractImages(msg.result), msg.error ? 'error' : 'result');
  } else if (msg.method) {
    log('### NOTIFY ' + JSON.stringify(msg).slice(0, 500));
    // 服务端会主动 ping 保活；按 MCP 规范必须回空结果，否则连接可能被判失活
    if (msg.method === 'ping' && msg.id !== undefined) {
      post({ jsonrpc: '2.0', id: msg.id, result: {} })
        .then(() => log('### PING_REPLIED id=' + msg.id))
        .catch((e) => log('### PING_ERR ' + e.message));
    }
  }
}

async function post(body) {
  await readyPromise;
  const res = await fetch(messageUrl, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  });
  const txt = await res.text();
  if (!res.ok) log(`### POST_ERR status=${res.status} body=${txt.slice(0, 300)}`);
  else if (txt.trim()) log(`### POST_ACK ${txt.slice(0, 300)}`);
}

function rpc(method, params, timeoutMs) {
  const id = requestSeq++;
  const body = { jsonrpc: '2.0', id, method };
  if (params !== undefined) body.params = params;

  return new Promise((resolve) => {
    const timer = setTimeout(() => {
      if (pending.delete(id)) log(`### TIMEOUT id=${id} method=${method} (${timeoutMs || DEFAULT_TIMEOUT}ms)`);
      resolve();
    }, timeoutMs || DEFAULT_TIMEOUT);
    pending.set(id, { timer, resolve });
    post(body).catch((e) => {
      if (pending.delete(id)) {
        clearTimeout(timer);
        log(`### POST_EXC id=${id} ${e.message}`);
        resolve();
      }
    });
  });
}

function notify(method, params) {
  return post({ jsonrpc: '2.0', method, ...(params !== undefined ? { params } : {}) });
}

async function streamSse() {
  const res = await fetch(SSE_URL, { headers: { Accept: 'text/event-stream' } });
  if (!res.ok || !res.body) {
    log('### SSE_ERR status=' + res.status);
    process.exit(1);
  }
  log('### SSE_CONNECTED status=' + res.status);
  const reader = res.body.getReader();
  const dec = new TextDecoder();
  let buf = '';
  for (;;) {
    const { value, done } = await reader.read();
    if (done) {
      log('### SSE_CLOSED');
      break;
    }
    buf += dec.decode(value, { stream: true }).replace(/\r\n/g, '\n');
    let idx;
    while ((idx = buf.indexOf('\n\n')) >= 0) {
      const raw = buf.slice(0, idx);
      buf = buf.slice(idx + 2);
      try {
        handleEvent(raw);
      } catch (e) {
        log('### EVENT_ERR ' + e.message);
      }
    }
  }
}

/** 单条命令处理（由串行队列调用，保证 click / wait / snapshot 的先后顺序） */
async function handleLine(t) {
  let cmd;
  try {
    cmd = JSON.parse(t);
  } catch {
    log('### BAD_JSON ' + t.slice(0, 200));
    return;
  }
  if (cmd.sleep) {
    log('### SLEEP ' + cmd.sleep + 'ms');
    await new Promise((r) => setTimeout(r, cmd.sleep));
    return;
  }
  if (cmd.ping) {
    log(`### PONG messageUrl=${messageUrl} pending=${pending.size}`);
    return;
  }
  if (cmd.tool) {
    let args = cmd.args;
    if (cmd.argsFile) args = JSON.parse(fs.readFileSync(cmd.argsFile, 'utf8'));
    log('### CALL tool=' + cmd.tool + ' args=' + JSON.stringify(args || {}).slice(0, 600));
    await rpc('tools/call', { name: cmd.tool, arguments: args || {} }, cmd.timeoutMs);
    return;
  }
  if (cmd.raw) {
    log('### RAW_SEND ' + JSON.stringify(cmd.raw).slice(0, 400));
    await post(cmd.raw);
    return;
  }
  if (cmd.method) {
    log('### REQ method=' + cmd.method + (cmd.params ? ' params=' + JSON.stringify(cmd.params).slice(0, 400) : ''));
    await rpc(cmd.method, cmd.params, cmd.timeoutMs);
    return;
  }
  log('### BAD_CMD ' + t.slice(0, 200));
}

function startRepl() {
  const rl = readline.createInterface({ input: process.stdin });
  // 关键：串行链——一条命令处理完才处理下一条，避免批量下发时乱序
  let chain = Promise.resolve();
  rl.on('line', (line) => {
    const t = line.trim();
    if (!t) return;
    chain = chain.then(() => handleLine(t)).catch((e) => log('### CMD_ERR ' + e.message));
  });
  rl.on('close', () => {
    log('### STDIN_CLOSED');
    process.exit(0);
  });
}

(async () => {
  startRepl();
  await streamSse();
})().catch((e) => {
  log('### FATAL ' + e.stack);
  process.exit(1);
});
