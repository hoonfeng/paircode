const fs = require('fs');
const file = process.argv[2];
const lines = fs.readFileSync(file, 'utf8').split('\n').filter(l => l.trim());
const roles = {};
let toolCalls = 0, thinkingSegs = 0;
let lastAssistant = null;
for (const l of lines) {
  try {
    const o = JSON.parse(l);
    const m = o.message || {};
    const r = m.role;
    roles[r] = (roles[r] || 0) + 1;
    if (r === 'assistant') {
      const segs = o.segments || [];
      let tc = 0, th = 0;
      for (const s of segs) {
        if (s.type === 'tool_call') tc++;
        if (s.type === 'thinking') th++;
      }
      toolCalls += tc;
      thinkingSegs += th;
      lastAssistant = { line: l, toolCalls: tc, thinking: th, segCount: segs.length };
    }
  } catch (e) {}
}
console.log('total_lines:', lines.length);
console.log('roles:', JSON.stringify(roles));
console.log('total_tool_calls:', toolCalls, 'total_thinking:', thinkingSegs);
if (lastAssistant) {
  console.log('last_assistant: segs=' + lastAssistant.segCount + ' toolCalls=' + lastAssistant.toolCalls + ' thinking=' + lastAssistant.thinking);
  const o = JSON.parse(lastAssistant.line);
  console.log('last_assistant_summary:', JSON.stringify((o.message && o.message.content || '').slice(0, 120)));
  console.log('last_assistant_seg_types:', JSON.stringify((o.segments || []).map(s => s.type)));
}
