#!/usr/bin/env node
/**
 * bili-cdp.js — 用 CDP 驱动 B 站投稿页（零第三方依赖）
 *
 * WebSocket 客户端实现复用 scripts/cdp-verify.js 的自建思路（node 原生 http/net/crypto）。
 * 参数中任何以 @ 开头的值会被当作文件路径读取其内容（避免 shell 转义地狱，尤其含中文/引号的 JS）。
 *
 * 用法：
 *   node scripts/bili-cdp.js list                     列出所有 target
 *   node scripts/bili-cdp.js frames                   列出 frame 树（调试 iframe）
 *   node scripts/bili-cdp.js eval @probe.js           在页面执行 JS（返回 JSON）
 *   node scripts/bili-cdp.js cookies [domain]         列出 cookie（含 httpOnly，可判登录态）
 *   node scripts/bili-cdp.js setfile <sel> <abspath>  DOM.setFileInputFiles（绕过原生文件对话框）
 *   node scripts/bili-cdp.js click <sel>              点击元素
 *   node scripts/bili-cdp.js nav <url>                导航（等待加载完成）
 *   node scripts/bili-cdp.js shot <out.png>           页面截图
 *
 * 环境变量：
 *   CDP_PORT   调试端口（默认 9222）
 *   CDP_MATCH  target 匹配串（默认 "bilibili"，匹配 url 或 title）
 */
'use strict';

const http = require('http');
const net = require('net');
const crypto = require('crypto');
const fs = require('fs');
const path = require('path');

const PORT = Number(process.env.CDP_PORT || 9222);
const HOST = '127.0.0.1';
const MATCH = process.env.CDP_MATCH || 'bilibili';

// ───────────────────────── 极简 WebSocket（仅 text frame，源自 cdp-verify.js） ─────────────────────────
class WS {
  constructor() {
    this.buf = Buffer.alloc(0);
    this.frag = [];
    this.onMessage = () => {};
  }
  connect(wsUrl) {
    return new Promise((resolve, reject) => {
      const m = wsUrl.match(/^ws:\/\/([^:/]+):(\d+)(\/.*)$/);
      if (!m) return reject(new Error('bad ws url: ' + wsUrl));
      this.sock = net.connect(Number(m[2]), m[1], () => {
        const key = crypto.randomBytes(16).toString('base64');
        this.sock.write(
          `GET ${m[3]} HTTP/1.1\r\nHost: ${m[1]}:${m[2]}\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: ${key}\r\nSec-WebSocket-Version: 13\r\n\r\n`
        );
      });
      let handshaken = false;
      this.sock.on('data', (d) => {
        this.buf = Buffer.concat([this.buf, d]);
        if (!handshaken) {
          const idx = this.buf.indexOf('\r\n\r\n');
          if (idx < 0) return;
          const head = this.buf.slice(0, idx).toString();
          if (!/101/.test(head)) return reject(new Error('handshake failed: ' + head.split('\r\n')[0]));
          handshaken = true;
          this.buf = this.buf.slice(idx + 4);
          this._drain();
          resolve();
        } else this._drain();
      });
      this.sock.on('error', reject);
    });
  }
  _drain() {
    for (;;) {
      if (this.buf.length < 2) return;
      const b0 = this.buf[0];
      const b1 = this.buf[1];
      let len = b1 & 0x7f;
      let off = 2;
      if (len === 126) {
        if (this.buf.length < 4) return;
        len = this.buf.readUInt16BE(2);
        off = 4;
      } else if (len === 127) {
        if (this.buf.length < 10) return;
        len = Number(this.buf.readBigUInt64BE(2));
        off = 10;
      }
      const masked = (b1 & 0x80) !== 0;
      let mask = null;
      if (masked) {
        if (this.buf.length < off + 4) return;
        mask = this.buf.slice(off, off + 4);
        off += 4;
      }
      if (this.buf.length < off + len) return;
      let payload = this.buf.slice(off, off + len);
      if (masked) {
        const out = Buffer.alloc(len);
        for (let i = 0; i < len; i++) out[i] = payload[i] ^ mask[i & 3];
        payload = out;
      }
      this.buf = this.buf.slice(off + len);
      const opcode = b0 & 0x0f;
      if (opcode === 1) this.frag.push(payload);
      else if (opcode === 9) this._sendFrame(10, payload);
      else if (opcode === 8) {
        this.sock.end();
        return;
      }
      if (opcode === 8) return;
      if (b0 & 0x80 && this.frag.length) {
        const full = Buffer.concat(this.frag);
        this.frag = [];
        this.onMessage(full);
      }
    }
  }
  _sendFrame(opcode, payload) {
    const len = payload.length;
    let header;
    if (len < 126) header = Buffer.from([0x80 | opcode, 0x80 | len]);
    else if (len < 65536) {
      header = Buffer.alloc(4);
      header[0] = 0x80 | opcode;
      header[1] = 0x80 | 126;
      header.writeUInt16BE(len, 2);
    } else {
      header = Buffer.alloc(10);
      header[0] = 0x80 | opcode;
      header[1] = 0x80 | 127;
      header.writeBigUInt64BE(BigInt(len), 2);
    }
    const mask = crypto.randomBytes(4);
    const masked = Buffer.alloc(len);
    for (let i = 0; i < len; i++) masked[i] = payload[i] ^ mask[i & 3];
    this.sock.write(Buffer.concat([header, mask, masked]));
  }
  send(obj) {
    this._sendFrame(1, Buffer.from(JSON.stringify(obj)));
  }
}

// ───────────────────────── CDP HTTP 端点 ─────────────────────────
function httpReq(method, p) {
  return new Promise((resolve, reject) => {
    const req = http.request({ host: HOST, port: PORT, path: p, method }, (res) => {
      let d = '';
      res.on('data', (c) => (d += c));
      res.on('end', () => {
        try {
          resolve(JSON.parse(d));
        } catch (e) {
          reject(new Error('bad json from ' + p + ': ' + d.slice(0, 200)));
        }
      });
    });
    req.on('error', reject);
    req.end();
  });
}

// ───────────────────────── CDP 会话 ─────────────────────────
let ws = null;
let msgId = 0;
const pending = new Map();

function call(method, params = {}, timeoutMs = 40000) {
  const id = ++msgId;
  return new Promise((resolve, reject) => {
    const timer = setTimeout(() => {
      pending.delete(id);
      reject(new Error('CDP timeout: ' + method));
    }, timeoutMs);
    pending.set(id, { resolve, reject, timer });
    ws.send({ id, method, params });
  });
}

async function pickTarget(match) {
  const list = await httpReq('GET', '/json/list');
  const pages = list.filter((t) => t.type === 'page' && t.webSocketDebuggerUrl);
  const m = match || MATCH;
  const hit = pages.find((t) => (t.url || '').includes(m) || (t.title || '').includes(m));
  return hit || pages[0];
}

async function connect(match) {
  const target = await pickTarget(match);
  if (!target) throw new Error('未找到可连接的 page target（端口 ' + PORT + '）');
  console.error('[cdp] 目标: ' + (target.title || '') + ' | ' + (target.url || ''));
  ws = new WS();
  ws.onMessage = (buf) => {
    let msg;
    try {
      msg = JSON.parse(buf.toString());
    } catch {
      return;
    }
    if (msg.id && pending.has(msg.id)) {
      const p = pending.get(msg.id);
      pending.delete(msg.id);
      clearTimeout(p.timer);
      if (msg.error) p.reject(new Error('CDP error: ' + JSON.stringify(msg.error)));
      else p.resolve(msg.result);
    }
  };
  await ws.connect(target.webSocketDebuggerUrl);
  return target;
}

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

function readArg(v) {
  if (typeof v === 'string' && v.startsWith('@')) return fs.readFileSync(v.slice(1), 'utf8');
  return v;
}

// ───────────────────────── 命令 ─────────────────────────
async function main() {
  const [cmd, ...rawArgs] = process.argv.slice(2);
  const args = rawArgs.map(readArg);

  if (!cmd || cmd === 'help') {
    console.log(fs.readFileSync(__filename, 'utf8').split('\n').slice(1, 20).join('\n').replace(/^ \* ?/gm, ''));
    return;
  }

  if (cmd === 'list') {
    const list = await httpReq('GET', '/json/list');
    for (const t of list) console.log(`${t.type}\t${(t.title || '').slice(0, 40)}\t${(t.url || '').slice(0, 90)}`);
    return;
  }

  await connect(args.cdpMatch && undefined);

  switch (cmd) {
    case 'frames': {
      const { frameTree } = await call('Page.getFrameTree');
      const walk = (f, d) => {
        console.log('  '.repeat(d) + `- ${f.frame.name || '(main)'} | ${(f.frame.url || '').slice(0, 100)}`);
        (f.childFrames || []).forEach((c) => walk(c, d + 1));
      };
      walk(frameTree, 0);
      break;
    }
    case 'eval': {
      const expr = args[0];
      const r = await call('Runtime.evaluate', {
        expression: expr,
        awaitPromise: true,
        returnByValue: true,
        userGesture: true,
      });
      if (r.exceptionDetails) {
        console.error('EVAL_EXCEPTION: ' + JSON.stringify(r.exceptionDetails).slice(0, 800));
        process.exitCode = 1;
      } else {
        console.log(JSON.stringify(r.result && 'value' in r.result ? r.result.value : r.result, null, 2));
      }
      break;
    }
    case 'cookies': {
      const filter = args[0];
      const { cookies } = await call('Storage.getCookies');
      const hit = filter ? cookies.filter((c) => (c.domain || '').includes(filter)) : cookies;
      console.log(`共 ${cookies.length} 条 cookie，匹配 "${filter || '*'}" 的 ${hit.length} 条：`);
      for (const c of hit) {
        console.log(
          `  ${c.domain}${c.path} | ${c.name}=${String(c.value).slice(0, 24)}${String(c.value).length > 24 ? '…' : ''} | httpOnly=${!!c.httpOnly} secure=${!!c.secure} expires=${new Date(c.expires * 1000).toISOString().slice(0, 19)}`
        );
      }
      break;
    }
    case 'setfile': {
      const [selector, filePath] = args;
      if (!fs.existsSync(filePath)) throw new Error('文件不存在: ' + filePath);
      const { root } = await call('DOM.getDocument', { depth: 1, pierce: true });
      const { nodeId } = await call('DOM.querySelector', { nodeId: root.nodeId, selector });
      if (!nodeId) throw new Error('未找到元素: ' + selector);
      await call('DOM.setFileInputFiles', { files: [path.resolve(filePath)], nodeId });
      console.log('OK 已把文件塞入 ' + selector + ' → ' + filePath);
      break;
    }
    case 'click': {
      const sel = args[0];
      const r = await call('Runtime.evaluate', {
        expression: `(() => { const el = document.querySelector(${JSON.stringify(sel)}); if (!el) return 'NOT_FOUND'; el.scrollIntoView({block:'center'}); el.click(); return 'CLICKED:' + (el.tagName||'') + ':' + (el.className||'').toString().slice(0,60); })()`,
        returnByValue: true,
        userGesture: true,
      });
      console.log(r.result.value);
      break;
    }
    case 'nav': {
      await call('Page.enable');
      await call('Page.navigate', { url: args[0] });
      await sleep(Number(process.env.CDP_NAV_WAIT || 6000));
      const r = await call('Runtime.evaluate', { expression: 'location.href + " | " + document.title', returnByValue: true });
      console.log(r.result.value);
      break;
    }
    case 'shot': {
      const out = path.resolve(args[0] || 'screenshots/cdp-shot.png');
      const { data } = await call('Page.captureScreenshot', { format: 'png', captureBeyondViewport: false });
      fs.mkdirSync(path.dirname(out), { recursive: true });
      fs.writeFileSync(out, Buffer.from(data, 'base64'));
      console.log('OK ' + out + ' (' + fs.statSync(out).size + ' bytes)');
      break;
    }
    default:
      throw new Error('未知命令: ' + cmd);
  }
}

main()
  .then(() => {
    if (ws && ws.sock) ws.sock.end();
    setTimeout(() => process.exit(process.exitCode || 0), 50);
  })
  .catch((e) => {
    console.error('FAILED: ' + e.message);
    if (ws && ws.sock) ws.sock.end();
    setTimeout(() => process.exit(1), 50);
  });
