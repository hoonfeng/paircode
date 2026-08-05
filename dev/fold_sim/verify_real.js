// 验证 real_data.js 解析器：注入真实 jsonl 后输出结构
const fs = require('fs');
const path = require('path');

const realDataPath = path.join(__dirname, 'real_data.js');
const convPath = process.argv[2];

global.window = global;
eval(fs.readFileSync(realDataPath, 'utf8'));
// 第一次 eval 已定义 parseConversationJsonl 与 FALLBACK_COMBOS

const raw = fs.readFileSync(convPath, 'utf8');
global.__CONV_JSONL__ = raw;
// 重新触发数据装配（real_data.js 顶层代码在首次 eval 时执行过，
// 但 __CONV_JSONL__ 是后设的，需要再次执行装配逻辑）
// 直接手动调用：
let combos = null;
try { combos = global.parseConversationJsonl(raw); } catch (e) { console.log('parse err:', e.message); }
global.REAL_COMBOS = (combos && combos.length) ? combos : global.FALLBACK_COMBOS;

const cs = global.REAL_COMBOS;
console.log('combos:', cs.length);
console.log('has_user:', !!cs[0].user);
console.log('folded:', cs[0].assistant._folded);
console.log('summary:', JSON.stringify((cs[0].assistant._summary || '').slice(0, 100)));
console.log('segs:', cs[0].assistant.segments.map(s => s.type).join(','));
