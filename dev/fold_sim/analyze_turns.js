// 分析 jsonl 中各 assistant 回合的段分布，选最完整的 run
const fs = require('fs');
const file = process.argv[2];
const lines = fs.readFileSync(file, 'utf8').split('\n').filter(l => l.trim());
const turns = [];
for (const l of lines) {
  try {
    const o = JSON.parse(l);
    const m = o.message || {};
    if (m.role !== 'assistant') continue;
    const segs = o.segments || [];
    const types = segs.map(s => s.type);
    let tc = types.filter(t => t === 'tool_call').length;
    let th = types.filter(t => t === 'thinking').length;
    let co = types.filter(t => t === 'content').length;
    turns.push({ idx: turns.length, total: segs.length, tc, th, co, types: types.join(',') });
  } catch (e) {}
}
turns.forEach(t => console.log(JSON.stringify(t)));
