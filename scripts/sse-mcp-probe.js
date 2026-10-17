#!/usr/bin/env node
// sse-mcp-probe.js — 一次性 MCP SSE 探测（短超时，不驻留）
// 用法: node scripts/sse-mcp-probe.js <SSE_URL> [timeoutMs]
const url = process.argv[2];
const TIMEOUT = Number(process.argv[3] || 8000);
if (!url) { console.error('usage: node scripts/sse-mcp-probe.js <url> [timeoutMs]'); process.exit(2); }

let msgUrl = null;
const pending = new Map();
let idSeq = 900;
let endpointSeen = false;

function finish(id, payload, kind) {
  const p = pending.get(id);
  if (!p) return false;
  pending.delete(id);
  clearTimeout(p.timer);
  console.log(`\n=== RESP id=${id} ${kind} ===`);
  console.log(JSON.stringify(payload).slice(0, 4000));
  p.resolve(true);
  return true;
}

async function main() {
  const res = await fetch(url, { headers: { Accept: 'text/event-stream' } });
  console.log('SSE status =', res.status);
  if (!res.ok || !res.body) { console.log('SSE 建立失败'); process.exit(1); }
  const reader = res.body.getReader();
  const dec = new TextDecoder();
  let buf = '';

  (async () => {
    for (;;) {
      const { value, done } = await reader.read();
      if (done) break;
      buf += dec.decode(value, { stream: true }).replace(/\r\n/g, '\n');
      let i;
      while ((i = buf.indexOf('\n\n')) >= 0) {
        const raw = buf.slice(0, i);
        buf = buf.slice(i + 2);
        let ev = 'message';
        const dataLines = [];
        for (const line of raw.split('\n')) {
          if (line.startsWith('event:')) ev = line.slice(6).trim();
          else if (line.startsWith('data:')) dataLines.push(line.slice(5).replace(/^ /, ''));
        }
        const data = dataLines.join('\n');
        if (ev === 'endpoint') {
          msgUrl = data.startsWith('http') ? data : new URL(url).origin + data;
          console.log('endpoint =', msgUrl);
          endpointSeen = true;
          continue;
        }
        if (!data) continue;
        let m;
        try { m = JSON.parse(data); } catch { console.log('RAW', data.slice(0, 200)); continue; }
        if (m.id !== undefined && (m.result !== undefined || m.error !== undefined)) {
          finish(m.id, m.error ? { error: m.error } : m.result, m.error ? 'ERROR' : 'RESULT');
        } else if (m.method === 'ping' && m.id !== undefined) {
          fetch(msgUrl, { method: 'POST', headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ jsonrpc: '2.0', id: m.id, result: {} }) }).catch(() => {});
          console.log('(auto-pong id=%s)', m.id);
        } else if (m.method) {
          console.log('NOTIFY', JSON.stringify(m).slice(0, 200));
        }
      }
    }
  })().catch(e => console.log('readLoop err', e.message));

  const t0 = Date.now();
  while (!endpointSeen && Date.now() - t0 < 6000) await new Promise(r => setTimeout(r, 100));
  if (!endpointSeen) { console.log('❌ 未收到 endpoint 事件'); process.exit(1); }

  async function call(method, params, label) {
    const id = ++idSeq;
    const body = { jsonrpc: '2.0', id, method };
    if (params !== undefined) body.params = params;
    return new Promise((resolve) => {
      const timer = setTimeout(() => {
        pending.delete(id);
        console.log(`\n⏱ TIMEOUT ${label} (id=${id}, ${TIMEOUT}ms) — 无回包`);
        resolve(false);
      }, TIMEOUT);
      pending.set(id, { timer, resolve });
      fetch(msgUrl, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) })
        .then(async (r) => { const t = await r.text(); console.log(`POST ${label} → HTTP ${r.status} ${t.slice(0, 60)}`); })
        .catch(e => console.log('POST err', e.message));
    });
  }

  console.log('\n--- initialize (timeout %dms) ---', TIMEOUT);
  const ok1 = await call('initialize', {
    protocolVersion: '2024-11-05', capabilities: {},
    clientInfo: { name: 'probe', version: '1.0' },
  }, 'initialize');

  console.log('\n--- tools/list ---');
  const ok2 = await call('tools/list', undefined, 'tools/list');

  console.log('\n=== 结论 ===');
  console.log('initialize 回包 :', ok1 ? '✅ 有' : '❌ 无');
  console.log('tools/list 回包 :', ok2 ? '✅ 有' : '❌ 无');
  process.exit(0);
}
main().catch(e => { console.error('FATAL', e.message); process.exit(1); });
