// tool-design — UI 设计创作域工具面（纯 goja 零依赖）
//
// 设计原则（对齐《创作域支持方案》「真相源必须文本」+ 技能 no-ai-colors / emoji-icons）：
//   · 真相源 ① 设计令牌 design.tokens.json —— 语义命名 + 标准色板值（Tailwind 公开标准值），
//     不是随机生成的"AI 味"配色；面板与校验器都从这里取值，禁止节点里散落裸色值。
//   · 真相源 ② 界面工程 design.project.json —— 屏幕（screen）→ 节点树（node）；
//     节点样式一律引用 token（`$color.bg` / `$space.md`），保证设计系统一致性（D2 判据）。
//   · 产物全部是**文本**：自包含 HTML 预览（绝对定位渲染，可截图核验）/ CSS 变量表 /
//     mermaid 图（本插件只**生成 mermaid 文本**，渲染交给宿主 Markdown —— 零依赖）。
//   · 节点几何由内置**确定性布局引擎**（flex 堆叠模型）计算；校验与 HTML 渲染共用同一份盒子，
//     因此「校验通过」与「截图所见」严格同源（不会出现"报告说没溢出、图上却压字"）。
//
// 本插件是磁盘 goja 轨插件：沙箱无 require / Buffer / Node API，一律走 ctx 服务（ctx.fs / ctx.logger）。
// 图标禁止用 emoji（技能 emoji-icons）：只能引用内置 ICONS 白名单（内联 SVG path，stroke 风格）。

// ── 常量 ───────────────────────────────────────────────────
var SCHEMA = 'paircode.design/1';

// 令牌分组（每组是 { 名: 值 } 扁平 map；引用语法 `$<组>.<名>`）：
//   color(颜色字符串) space/radius/fontSize/fontWeight/lineHeight(数字) shadow/font(字符串)
var TOKEN_GROUPS = ['color', 'space', 'radius', 'fontSize', 'fontWeight', 'lineHeight', 'shadow', 'font'];

// 容器类型（参与 flex 排布）与叶子类型（内容自撑尺寸）
var CONTAINER_TYPES = ['frame', 'row', 'col', 'card', 'list'];
var LEAF_TYPES = ['text', 'button', 'input', 'badge', 'icon', 'image', 'divider', 'spacer', 'checkbox'];
var NODE_TYPES = CONTAINER_TYPES.concat(LEAF_TYPES);

// 可交互叶子（触达区判据 D6 适用）：button / input / checkbox / 可点 icon
var INTERACTIVE_TYPES = ['button', 'input', 'checkbox'];

// 图标白名单（技能 emoji-icons：禁止 emoji，只认这些内联 SVG path；viewBox 0 0 24 24，stroke 风格）
var ICONS = {
  home: 'M3 10.5 12 3.5l9 7M5.5 9.5V20h13V9.5',
  // 滑杆式"设置"图标（原齿轮弧线版本在 20px 下渲染成模糊星芒 —— 实测截图发现，换简单几何）
  settings: 'M4 7.5h16M4 16.5h16M9.5 5v5M15.5 14v5',
  search: 'M10.5 17a6.5 6.5 0 100-13 6.5 6.5 0 000 13M15.5 15.5 20.5 20.5',
  user: 'M12 12a4 4 0 100-8 4 4 0 000 8M4.5 20c1.5-3.5 4.2-5 7.5-5s6 1.5 7.5 5',
  plus: 'M12 5v14M5 12h14',
  close: 'M6 6l12 12M18 6L6 18',
  check: 'M5 13l4.5 4.5L19 7',
  'chevron-right': 'M9.5 5.5 16 12l-6.5 6.5',
  'chevron-down': 'M5.5 9.5 12 16l6.5-6.5',
  bell: 'M6.5 17h11l-1.5-2.5V11a4 4 0 10-8 0v3.5L6.5 17M10 19.5a2 2 0 004 0',
  trash: 'M5 7.5h14M9 7.5V5h6v2.5M7.5 7.5 8.5 20h7l1-12.5M10.5 11v5.5M13.5 11v5.5',
  edit: 'M4.5 19.5h4l10-10-4-4-10 10v4M14.5 5.5l4 4',
  download: 'M12 4v11M7.5 11 12 15.5 16.5 11M5 19.5h14',
  upload: 'M12 15.5V4.5M7.5 9 12 4.5 16.5 9M5 19.5h14',
  refresh: 'M19 12a7 7 0 11-2.4-5.3M19 4.5V9h-4.5',
  play: 'M8 5.5v13l11-6.5z',
  pause: 'M9 5.5v13M15 5.5v13',
  folder: 'M4 6.5h5.5l2 2.5H20v9.5H4z',
  file: 'M6 3.5h7l5 5v12H6zM13 3.5v5h5',
  star: 'M12 4.5l2.4 5 5.1.7-3.7 3.6.9 5.1-4.7-2.5-4.7 2.5.9-5.1L4.5 10.2l5.1-.7z',
  heart: 'M12 19.5C6.5 15.5 4 12.8 4 9.9 4 7.7 5.7 6 7.9 6c1.6 0 3 .9 4.1 2.4C13.1 6.9 14.5 6 16.1 6 18.3 6 20 7.7 20 9.9c0 2.9-2.5 5.6-8 9.6z',
  lock: 'M7 10.5h10V20H7zM9 10.5V8a3 3 0 016 0v2.5',
  mail: 'M4 6.5h16v11H4zM4 7l8 6 8-6',
  clock: 'M12 20a8 8 0 100-16 8 8 0 000 16M12 8v4.5l3 2',
  grid: 'M4.5 4.5h6v6h-6zM13.5 4.5h6v6h-6zM4.5 13.5h6v6h-6zM13.5 13.5h6v6h-6z',
  list: 'M8 6.5h12M8 12h12M8 17.5h12M4.5 6.5h.01M4.5 12h.01M4.5 17.5h.01',
  info: 'M12 20a8 8 0 100-16 8 8 0 000 16M12 11v5.5M12 8v.01',
  warning: 'M12 4 21 19.5H3zM12 10v4M12 16.5v.01',
  error: 'M12 20a8 8 0 100-16 8 8 0 000 16M9 9l6 6M15 9l-6 6',
  success: 'M12 20a8 8 0 100-16 8 8 0 000 16M8 12.5l3 3 5-6',
  external: 'M14 4.5h5.5V10M19.5 4.5 11 13M17 14v5.5H4.5V7H10',
  eye: 'M2.5 12S6 6.5 12 6.5 21.5 12 21.5 12 18 17.5 12 17.5 2.5 12 2.5 12zM12 14.5a2.5 2.5 0 100-5 2.5 2.5 0 000 5',
  copy: 'M8.5 8.5h11V20h-11zM4.5 4.5h11v4M4.5 4.5V16h4',
  link: 'M9.5 14.5 14.5 9.5M8 12 5.5 14.5a4 4 0 005.5 5.5L13.5 17M10.5 7 13 4.5a4 4 0 015.5 5.5L16 12',
  more: 'M6 12h.01M12 12h.01M18 12h.01',
};

// CSS 命名色子集（解析用；与 tool-art 同口径，避免全表 148 项膨胀）
var NAMED_COLORS = {
  black: '#000000', silver: '#C0C0C0', gray: '#808080', grey: '#808080', white: '#FFFFFF',
  maroon: '#800000', red: '#FF0000', purple: '#800080', fuchsia: '#FF00FF', magenta: '#FF00FF',
  green: '#008000', lime: '#00FF00', olive: '#808000', yellow: '#FFFF00', navy: '#000080',
  blue: '#0000FF', teal: '#008080', aqua: '#00FFFF', cyan: '#00FFFF', orange: '#FFA500',
};

// 触达区下限（移动端 44 / 桌面 32；D6 判据按屏幕宽度自动选，可显式覆盖）
var TOUCH_MIN_MOBILE = 44;
var TOUCH_MIN_DESKTOP = 32;
var TOUCH_MOBILE_MAX_W = 480;

var LIMIT_SCREENS = 60;      // 屏幕数上限
var LIMIT_NODES = 1200;      // 单工程节点总数上限
var LIMIT_DEPTH = 12;        // 节点树深度上限
var LIMIT_DIM = 4096;        // 屏幕尺寸上限
var EPS = 1e-6;
var GRID_BASE = 4;           // 间距网格基准（D4：显式间距必须是 4 的倍数）

// ── 基础工具 ───────────────────────────────────────────────
function isNum(v) { return typeof v === 'number' && isFinite(v); }
function isInt(v) { return isNum(v) && Math.floor(v) === v; }
function clamp(v, lo, hi) { return v < lo ? lo : (v > hi ? hi : v); }
function nowIso() { return new Date().toISOString(); }
function num(v, def) {
  var n = typeof v === 'number' ? v : parseFloat(v);
  return isNum(n) ? n : def;
}
function str(v, def) { return (v === undefined || v === null) ? def : String(v); }
function fmtNum(n) {
  var r = Math.round(num(n, 0) * 1000) / 1000;
  return String(r);
}
function deepClone(o) { return JSON.parse(JSON.stringify(o)); }

function argStr(args, key, def) {
  var v = args[key];
  if (v === undefined || v === null || v === '') return def;
  return String(v);
}
function argNum2(args, key, def) {
  var v = args[key];
  if (v === undefined || v === null || v === '') return def;
  var n = typeof v === 'number' ? v : parseFloat(v);
  return isNum(n) ? n : def;
}
function argInt2(args, key, def) {
  var v = args[key];
  if (v === undefined || v === null || v === '') return def;
  var n = typeof v === 'number' ? v : parseInt(v, 10);
  return isNum(n) ? n : def;
}
function argBool(args, key, def) {
  var v = args[key];
  if (v === undefined || v === null || v === '') return def;
  if (typeof v === 'boolean') return v;
  return v === 'true' || v === '1' || v === 1;
}

// ── 颜色 ───────────────────────────────────────────────────
function parseColor(input) {
  if (input === null || input === undefined) return null;
  var s = String(input).trim();
  if (!s) return null;
  if (s.toLowerCase() === 'transparent' || s.toLowerCase() === 'none') return { r: 0, g: 0, b: 0, a: 0, none: true };
  var lower = s.toLowerCase();
  if (NAMED_COLORS[lower]) s = NAMED_COLORS[lower];
  if (s.charAt(0) === '#') {
    var h = s.slice(1);
    if (h.length === 3 || h.length === 4) {
      var out = { r: parseInt(h[0] + h[0], 16), g: parseInt(h[1] + h[1], 16), b: parseInt(h[2] + h[2], 16), a: 1, none: false };
      if (h.length === 4) out.a = clamp(parseInt(h[3] + h[3], 16) / 255, 0, 1);
      return out;
    }
    if (h.length === 6 || h.length === 8) {
      var o2 = { r: parseInt(h.slice(0, 2), 16), g: parseInt(h.slice(2, 4), 16), b: parseInt(h.slice(4, 6), 16), a: 1, none: false };
      if (h.length === 8) o2.a = clamp(parseInt(h.slice(6, 8), 16) / 255, 0, 1);
      if (isNum(o2.r) && isNum(o2.g) && isNum(o2.b)) return o2;
      return null;
    }
    return null;
  }
  var m = /^rgba?\(([^)]+)\)$/i.exec(s);
  if (m) {
    var ps = m[1].split(',');
    if (ps.length < 3) return null;
    var rr = parseFloat(ps[0]); var gg = parseFloat(ps[1]); var bb = parseFloat(ps[2]);
    var aa = ps.length > 3 ? parseFloat(ps[3]) : 1;
    if (!isNum(rr) || !isNum(gg) || !isNum(bb)) return null;
    return { r: clamp(Math.round(rr), 0, 255), g: clamp(Math.round(gg), 0, 255), b: clamp(Math.round(bb), 0, 255), a: clamp(isNum(aa) ? aa : 1, 0, 1), none: false };
  }
  return null;
}
function toHex(c) {
  function h2(n) { var t = Math.round(clamp(n, 0, 255)).toString(16).toUpperCase(); return t.length < 2 ? '0' + t : t; }
  return '#' + h2(c.r) + h2(c.g) + h2(c.b);
}
function colorCss(c) {
  if (!c) return 'transparent';
  if (c.none || c.a <= 0.001) return 'transparent';
  if (c.a >= 0.999) return toHex(c);
  return 'rgba(' + c.r + ',' + c.g + ',' + c.b + ',' + fmtNum(c.a) + ')';
}
function blend(fg, bg) {
  if (!fg) return bg;
  if (fg.none || fg.a <= 0.001) return bg;
  if (fg.a >= 0.999) return { r: fg.r, g: fg.g, b: fg.b, a: 1, none: false };
  var base = bg || { r: 255, g: 255, b: 255, a: 1 };
  return {
    r: Math.round(fg.r * fg.a + base.r * (1 - fg.a)),
    g: Math.round(fg.g * fg.a + base.g * (1 - fg.a)),
    b: Math.round(fg.b * fg.a + base.b * (1 - fg.a)),
    a: 1,
    none: false,
  };
}
function relLum(c) {
  function ch(v) { var s = v / 255; return s <= 0.03928 ? s / 12.92 : Math.pow((s + 0.055) / 1.055, 2.4); }
  return 0.2126 * ch(c.r) + 0.7152 * ch(c.g) + 0.0722 * ch(c.b);
}
function contrastRatio(a, b) {
  if (!a || !b) return null;
  var la = relLum(a); var lb = relLum(b);
  var hi = Math.max(la, lb); var lo = Math.min(la, lb);
  return (hi + 0.05) / (lo + 0.05);
}

// ── 设计令牌（真相源 ①） ────────────────────────────────────
// 默认值全部取自公开标准（Tailwind CSS 官方色板 / 常用 8pt 间距与字号阶梯），非随机生成。
function defaultTokens() {
  return {
    color: {
      bg: '#FFFFFF',
      surface: '#F9FAFB',
      'surface-2': '#F3F4F6',
      border: '#E5E7EB',
      fg: '#111827',
      muted: '#6B7280',
      subtle: '#9CA3AF',
      accent: '#2563EB',
      'accent-fg': '#FFFFFF',
      'accent-soft': '#EFF6FF',
      success: '#047857',
      warning: '#92400E',
      danger: '#B91C1C',
    },
    space: { none: 0, xs: 4, sm: 8, md: 12, lg: 16, xl: 24, '2xl': 32, '3xl': 48 },
    radius: { none: 0, sm: 4, md: 6, lg: 8, full: 999 },
    fontSize: { xs: 11, sm: 12, md: 14, lg: 16, xl: 20, '2xl': 28 },
    fontWeight: { normal: 400, medium: 500, semibold: 600 },
    lineHeight: { tight: 1.25, normal: 1.5, loose: 1.75 },
    shadow: {
      none: 'none',
      sm: '0 1px 2px rgba(17,24,39,0.06)',
      md: '0 4px 12px rgba(17,24,39,0.08)',
    },
    font: { sans: 'system-ui, -apple-system, "Segoe UI", "Microsoft YaHei", sans-serif' },
  };
}

// 令牌取值：`$color.bg` → 值；支持 `$space.md`；非引用原样返回
function lookupToken(tokens, ref) {
  if (ref === undefined || ref === null) return { found: false, value: undefined };
  var s = String(ref);
  if (s.charAt(0) !== '$') return { found: true, value: undefined, literal: true };
  var body = s.slice(1);
  var dot = body.indexOf('.');
  if (dot <= 0) return { found: false, value: undefined };
  var grp = body.slice(0, dot); var name = body.slice(dot + 1);
  if (TOKEN_GROUPS.indexOf(grp) < 0) return { found: false, value: undefined, group: grp, name: name };
  var g = tokens && tokens[grp];
  if (!g || !(name in g)) return { found: false, value: undefined, group: grp, name: name };
  return { found: true, value: g[name], group: grp, name: name };
}

// 解析「尺寸类」属性：数字（px）或 token 引用 → 数值；'fill'/'auto' 原样返回
function resolveSize(v, tokens) {
  if (v === undefined || v === null) return undefined;
  var s = String(v);
  if (s === 'fill' || s === 'auto') return s;
  var tok = lookupToken(tokens, s);
  if (String(s).charAt(0) === '$') {
    if (!tok.found) return 'BADREF';
    var n = num(tok.value, NaN);
    return isNum(n) ? n : 'BADREF';
  }
  var direct = num(s, NaN);
  return isNum(direct) ? direct : 'BADREF';
}

// 解析「颜色类」属性 → 颜色对象（悬空引用返回 'BADREF'）
function resolveColor(v, tokens) {
  if (v === undefined || v === null) return undefined;
  var s = String(v);
  if (s === 'none' || s === 'transparent') return { r: 0, g: 0, b: 0, a: 0, none: true };
  if (s.charAt(0) === '$') {
    var tok = lookupToken(tokens, s);
    if (!tok.found) return 'BADREF';
    var c = parseColor(tok.value);
    if (!c) return 'BADREF';
    return c;
  }
  var c2 = parseColor(s);
  return c2 || 'BADREF';
}

function tokenSummary(tokens) {
  var lines = [];
  var counted = 0;
  for (var i = 0; i < TOKEN_GROUPS.length; i++) {
    var g = TOKEN_GROUPS[i]; var obj = tokens[g] || {};
    var keys = Object.keys(obj);
    counted += keys.length;
    lines.push('  ' + g + '（' + keys.length + '）：' + keys.slice(0, 8).map(function (k) { return k + '=' + obj[k]; }).join('  ') + (keys.length > 8 ? '  …' : ''));
  }
  return '令牌共 ' + counted + ' 项\n' + lines.join('\n');
}

// ── 布局引擎（确定性 flex 堆叠模型） ─────────────────────────
// 规则（与 HTML 渲染严格同源，渲染用同一份盒子做绝对定位）：
//   · 容器（frame/row/col/card/list）：w 默认 fill、h 默认 auto（内容高）；pad 内缩；子节点间 gap；
//     row → 主轴水平，col/frame/card/list → 主轴垂直；cross 轴默认拉伸（子节点 w/h = 'fill' 时填满内容区）。
//   · 叶子：由内容估算尺寸（text 按字号×行高估算宽度换行；button/input/badge 为 padding + 内容）。
//   · 溢出与越界都会被记录（D8 判据），且渲染时可见 → 「校验=截图」。

function isCJK(code) {
  return (code >= 0x2e80 && code <= 0x9fff) || (code >= 0x3000 && code <= 0x303f) ||
    (code >= 0xac00 && code <= 0xd7af) || (code >= 0xf900 && code <= 0xfaff) ||
    (code >= 0xff00 && code <= 0xff60);
}
// 确定性文本宽度估算（em）：CJK 全宽，拉丁按字宽类别。仅供几何检查，不做像素级字体还原。
function textWidthEm(s) {
  var total = 0;
  for (var i = 0; i < s.length; i++) {
    var c = s.charCodeAt(i);
    if (isCJK(c)) { total += 1.0; continue; }
    if (c === 32) { total += 0.28; continue; }
    if (c >= 48 && c <= 57) { total += 0.58; continue; }
    if (c >= 65 && c <= 90) { total += 0.70; continue; }
    if (c >= 97 && c <= 122) { total += 0.57; continue; }
    if (c === 46 || c === 44 || c === 58 || c === 59 || c === 39 || c === 124 || c === 33) { total += 0.28; continue; }
    if (c === 95 || c === 45 || c === 47 || c === 92) { total += 0.45; continue; }
    total += 0.58;
  }
  return total;
}
function textWidthPx(s, fontSize, weight) {
  var em = textWidthEm(s);
  var wobble = num(weight, 400) >= 600 ? 1.03 : 1.0;
  return Math.round(em * fontSize * wobble * 1000) / 1000;
}
// 估算换行后的行数（按空白切分 + 逐词累积；CJK 逐字断行）
function wrapLineCount(text, maxW, fontSize, weight) {
  var t = String(text);
  if (maxW <= 0) return 1;
  var tokens = t.split(/(\s+)/).filter(function (x) { return x !== ''; });
  var lines = 1; var cur = 0;
  for (var i = 0; i < tokens.length; i++) {
    var piece = tokens[i];
    var w = textWidthPx(piece, fontSize, weight);
    if (cur > 0 && cur + w > maxW) { lines++; cur = w; } else { cur += w; }
    // 单词自身超宽 → 按字数折行
    while (cur > maxW && maxW > 0) { lines++; cur -= maxW; }
  }
  return lines;
}

function nodeDefaultPad(node) {
  var t = node.type;
  if (t === 'card') return '$space.lg';
  if (t === 'button' || t === 'input' || t === 'badge' || t === 'list' || t === 'checkbox') return '$space.sm';
  return '$space.none';
}

function measureLeaf(node, availW, tokens, problems, screenId) {
  var p = node.props || {};
  var bad = false;
  function badRef(what) {
    problems.push({ kind: 'badref', screen: screenId, node: node.id, detail: what + ' 引用了不存在的 token（node ' + node.id + '）' });
    bad = true;
  }
  if (node.type === 'text') {
    var fs = resolveSize(p.size !== undefined ? p.size : '$fontSize.md', tokens);
    if (fs === 'BADREF') { badRef('size'); fs = 14; }
    var fw = resolveSize(p.weight !== undefined ? p.weight : '$fontWeight.normal', tokens);
    if (fw === 'BADREF') { badRef('weight'); fw = 400; }
    var lh = resolveSize(p.leading !== undefined ? p.leading : '$lineHeight.normal', tokens);
    if (lh === 'BADREF') { badRef('leading'); lh = 1.5; }
    var txt = str(p.text, '');
    var w = resolveSize(p.w, tokens);
    if (w === 'BADREF') { badRef('w'); w = undefined; }
    var effW = (w === undefined || w === 'auto') ? availW : (w === 'fill' ? availW : w);
    var lines = num(p.maxLines, 0) > 0 ? Math.min(wrapLineCount(txt, effW, fs, fw), num(p.maxLines, 1)) : wrapLineCount(txt, effW, fs, fw);
    // ★ 安全余量 4%+2px：估算模型与真实字体渲染总有偏差，不留余量会裁掉末字
    //   （实测："Agent 回复" 渲染成 "Agent 回"、长句末字 "端。" 被裁）—— 盒子略宽也更保守（D8 更早发现溢出）
    var contentW = Math.min(textWidthPx(txt, fs, fw) * 1.04 + 2, effW) || effW;
    var h = Math.round(lines * fs * lh * 1000) / 1000;
    var outW = (w === undefined || w === 'auto') ? contentW : (w === 'fill' ? availW : w);
    return { w: Math.round(outW * 1000) / 1000, h: h, lines: lines, fontSize: fs, weight: fw, leading: lh };
  }
  if (node.type === 'button') {
    var bfs = resolveSize(p.size !== undefined ? p.size : '$fontSize.md', tokens);
    if (bfs === 'BADREF') { badRef('size'); bfs = 14; }
    var padX = resolveSize(p.padX !== undefined ? p.padX : '$space.lg', tokens);
    if (padX === 'BADREF') { badRef('padX'); padX = 16; }
    var padY = resolveSize(p.padY !== undefined ? p.padY : '$space.sm', tokens);
    if (padY === 'BADREF') { badRef('padY'); padY = 8; }
    var label = str(p.label, '');
    var bw = resolveSize(p.w, tokens);
    if (bw === 'BADREF') { badRef('w'); bw = undefined; }
    var bh = resolveSize(p.h, tokens);
    if (bh === 'BADREF') { badRef('h'); bh = undefined; }
    var autoW = Math.round((textWidthPx(label, bfs, 600) + padX * 2) * 1000) / 1000;
    var autoH = Math.round((bfs * 1.5 + padY * 2) * 1000) / 1000;
    return {
      w: bw === 'fill' ? availW : (bw === undefined ? (p.full === true ? availW : autoW) : bw),
      h: bh === 'fill' ? undefined : (bh === undefined ? autoH : bh),
      autoH: autoH, fontSize: bfs,
    };
  }
  if (node.type === 'input' || node.type === 'checkbox') {
    var ih = node.type === 'input'
      ? resolveSize(p.h !== undefined ? p.h : 36, tokens)
      : resolveSize(p.h !== undefined ? p.h : '$fontSize.lg', tokens);
    if (ih === 'BADREF') { badRef('h'); ih = node.type === 'input' ? 36 : 16; }
    var iw = resolveSize(p.w !== undefined ? p.w : (node.type === 'checkbox' ? ih : 'fill'), tokens);
    if (iw === 'BADREF') { badRef('w'); iw = 'fill'; }
    return {
      w: iw === 'fill' ? availW : (iw === undefined ? availW : iw),
      h: node.type === 'checkbox' ? ih : (ih === 'fill' ? 36 : ih),
      minTouch: true,
    };
  }
  if (node.type === 'badge') {
    var bs = resolveSize(p.size !== undefined ? p.size : '$fontSize.xs', tokens);
    if (bs === 'BADREF') { badRef('size'); bs = 11; }
    var bpx = resolveSize(p.padX !== undefined ? p.padX : '$space.sm', tokens);
    if (bpx === 'BADREF') { badRef('padX'); bpx = 8; }
    var bpy = resolveSize(p.padY !== undefined ? p.padY : '$space.xs', tokens);
    if (bpy === 'BADREF') { badRef('padY'); bpy = 4; }
    return {
      w: Math.round((textWidthPx(str(p.label, ''), bs, 500) + bpx * 2) * 1000) / 1000,
      h: Math.round((bs * 1.4 + bpy * 2) * 1000) / 1000,
      fontSize: bs,
    };
  }
  if (node.type === 'icon') {
    var isz = resolveSize(p.size !== undefined ? p.size : 20, tokens);
    if (isz === 'BADREF') { badRef('size'); isz = 20; }
    return { w: isz === 'fill' ? availW : isz, h: isz === 'fill' ? 20 : isz };
  }
  if (node.type === 'image') {
    var ratio = num(p.ratio, 0);
    var iw2 = resolveSize(p.w !== undefined ? p.w : 'fill', tokens);
    if (iw2 === 'BADREF') { badRef('w'); iw2 = 'fill'; }
    var wpx = iw2 === 'fill' || iw2 === undefined ? availW : iw2;
    var hpx = resolveSize(p.h, tokens);
    if (hpx === 'BADREF') { badRef('h'); hpx = undefined; }
    if (hpx === undefined) hpx = ratio > 0 ? Math.round(wpx / ratio * 1000) / 1000 : 120;
    return { w: wpx, h: hpx };
  }
  if (node.type === 'divider') {
    var dv = resolveSize(p.h !== undefined ? p.h : 1, tokens);
    if (dv === 'BADREF') { badRef('h'); dv = 1; }
    return { w: availW, h: dv };
  }
  if (node.type === 'spacer') {
    var sp = resolveSize(p.h !== undefined ? p.h : '$space.md', tokens);
    if (sp === 'BADREF') { badRef('h'); sp = 12; }
    return { w: availW, h: sp === 'fill' ? 0 : sp };
  }
  return { w: availW, h: 0 };
}

// 计算单个节点盒子并递归子节点。
// ctx2: {tokens, boxes, problems, screenId, depth}
function layoutNode(node, x, y, availW, availH, ctx2) {
  var tokens = ctx2.tokens;
  var problems = ctx2.problems;
  var p = node.props || {};
  var isContainer = CONTAINER_TYPES.indexOf(node.type) >= 0;

  // pad / gap
  var padRaw = p.pad !== undefined ? p.pad : nodeDefaultPad(node);
  var pad = resolveSize(padRaw, tokens);
  if (pad === 'BADREF') { problems.push({ kind: 'badref', screen: ctx2.screenId, node: node.id, detail: 'pad 引用不存在: ' + padRaw }); pad = 0; }
  if (pad === 'fill' || pad === 'auto' || pad === undefined) pad = 0;
  var gapRaw = p.gap !== undefined ? p.gap : (isContainer ? '$space.sm' : '$space.none');
  var gap = resolveSize(gapRaw, tokens);
  if (gap === 'BADREF') { problems.push({ kind: 'badref', screen: ctx2.screenId, node: node.id, detail: 'gap 引用不存在: ' + gapRaw }); gap = 0; }
  if (typeof gap !== 'number') gap = 0;

  var wRaw = resolveSize(p.w, tokens);
  if (wRaw === 'BADREF') { problems.push({ kind: 'badref', screen: ctx2.screenId, node: node.id, detail: 'w 引用不存在: ' + p.w }); wRaw = undefined; }
  var hRaw = resolveSize(p.h, tokens);
  if (hRaw === 'BADREF') { problems.push({ kind: 'badref', screen: ctx2.screenId, node: node.id, detail: 'h 引用不存在: ' + p.h }); hRaw = undefined; }

  var box = { id: node.id, x: x, y: y, w: 0, h: 0 };

  if (!isContainer) {
    var m = measureLeaf(node, availW, tokens, problems, ctx2.screenId);
    box.w = wRaw === undefined || wRaw === 'auto' ? m.w : (wRaw === 'fill' ? availW : wRaw);
    box.h = hRaw === undefined || hRaw === 'auto' ? m.h : (hRaw === 'fill' ? (availH || m.h) : hRaw);
    box.metrics = m;
    ctx2.boxes[node.id] = box;
    return box;
  }

  // 容器：主轴方向
  var dir = node.type === 'row' ? 'row' : 'col';
  var outerW = wRaw === undefined || wRaw === 'fill' || wRaw === 'auto' ? availW : wRaw;
  var outerH = (hRaw === undefined || hRaw === 'auto') ? null : (hRaw === 'fill' ? availH : hRaw);
  var innerW = Math.max(0, outerW - pad * 2);
  var innerH = outerH === null ? null : Math.max(0, outerH - pad * 2);

  var kids = node.children || [];
  var innerX = x + pad; var innerY = y + pad;
  var mainUsed = 0;
  var boxesTmp = [];

  for (var i = 0; i < kids.length; i++) {
    var kid = kids[i];
    var childAvailW = dir === 'row' ? innerW : innerW;
    var childAvailH = innerH === null ? 0 : innerH;
    var before = ctx2.problems.length;
    var kb = layoutNode(kid, dir === 'row' ? (innerX + mainUsed) : innerX, dir === 'row' ? innerY : (innerY + mainUsed), childAvailW, childAvailH, {
      tokens: tokens,
      boxes: ctx2.boxes,
      problems: problems,
      screenId: ctx2.screenId,
      depth: (ctx2.depth || 0) + 1,
    });
    // cross 轴拉伸：子节点未显式指定 size 且是容器或 fill → 填满交叉轴
    if (dir === 'col' && !(kid.props && (kid.props.w !== undefined && kid.props.w !== 'auto'))) {
      var stretch = CONTAINER_TYPES.indexOf(kid.type) >= 0 || kid.type === 'divider' || kid.type === 'input' || kid.type === 'image' || kid.type === 'spacer';
      if (stretch) {
        kb.w = innerW;
        kb.x = innerX;
        if (kb.metrics) kb.metrics.w = innerW;
      }
    }
    mainUsed += (dir === 'row' ? kb.w : kb.h) + (i < kids.length - 1 ? gap : 0);
    boxesTmp.push(kb);
    if (ctx2.problems.length > before) { /* 子节点问题已记录 */ }
  }

  var contentMain = mainUsed;
  var finalH = outerH;
  if (finalH === null) finalH = Math.round((dir === 'row' ? maxOf(boxesTmp.map(function (b) { return b.h; })) : contentMain) * 1000) / 1000 + pad * 2;
  var finalW = outerW;
  if (dir === 'row' && wRaw === undefined && node.type === 'row') {
    // 仅 row 未指定宽时按内容宽；frame/col/card/list 默认撑满可用宽度
    finalW = Math.round(contentMain * 1000) / 1000 + pad * 2;
  }

  box.w = Math.round(finalW * 1000) / 1000;
  box.h = Math.round(finalH * 1000) / 1000;
  box.pad = pad;
  box.gap = gap;
  box.dir = dir;
  box.children = boxesTmp.map(function (b) { return b.id; });
  ctx2.boxes[node.id] = box;

  // 主轴溢出（D8）：内容超出容器可用高/宽
  var availMain = dir === 'row' ? innerW : (innerH === null ? null : innerH);
  if (availMain !== null && contentMain > availMain + EPS) {
    problems.push({
      kind: 'overflow',
      screen: ctx2.screenId,
      node: node.id,
      detail: (dir === 'row' ? '横向' : '纵向') + '溢出：内容 ' + fmtNum(contentMain) + 'px > 可用 ' + fmtNum(availMain) + 'px（容器 ' + node.id + '）',
      amount: Math.round((contentMain - availMain) * 1000) / 1000,
    });
  }
  return box;
}
function maxOf(arr) {
  if (!arr || !arr.length) return 0;
  var m = -Infinity;
  for (var i = 0; i < arr.length; i++) if (arr[i] > m) m = arr[i];
  return m === -Infinity ? 0 : m;
}

function layoutScreen(screen, tokens) {
  var boxes = {};
  var problems = [];
  var sw = num(screen.width, 360); var sh = num(screen.height, 640);
  var root = screen.root;
  if (root) {
    var rootNode = deepClone(root);
    rootNode.props = rootNode.props || {};
    // 根节点默认撑满屏幕
    if (rootNode.props.w === undefined) rootNode.props.w = sw;
    if (rootNode.props.h === undefined) rootNode.props.h = sh;
    layoutNode(rootNode, 0, 0, sw, sh, { tokens: tokens, boxes: boxes, problems: problems, screenId: screen.id, depth: 0 });
  }
  // 越界检查（D8）
  var ids = Object.keys(boxes);
  for (var i = 0; i < ids.length; i++) {
    var b = boxes[ids[i]];
    if (b.x < -EPS || b.y < -EPS || b.x + b.w > sw + EPS || b.y + b.h > sh + EPS) {
      problems.push({
        kind: 'outside',
        screen: screen.id,
        node: ids[i],
        detail: '越出屏幕：' + ids[i] + ' @(' + fmtNum(b.x) + ',' + fmtNum(b.y) + ') ' + fmtNum(b.w) + '×' + fmtNum(b.h) + '（屏幕 ' + sw + '×' + sh + '）',
      });
    }
  }
  return { boxes: boxes, problems: problems };
}

// ── 工程模型（真相源 ②） ────────────────────────────────────
function emptyProject(title, tokensPath) {
  return {
    schema: SCHEMA,
    kind: 'project',
    meta: { title: title || '未命名设计', updatedAt: nowIso() },
    tokens: tokensPath || 'design.tokens.json',
    screens: [],
    flow: [],
    artifacts: { html: 'design.html', css: 'design.tokens.css', flow: 'design.flow.mmd' },
  };
}

function normalizeProject(obj) {
  if (!obj || typeof obj !== 'object') throw new Error('工程内容不是对象');
  var sc = str(obj.schema, '');
  if (sc && sc !== SCHEMA) throw new Error('工程 schema 不匹配：' + sc + '（期望 ' + SCHEMA + '）');
  var proj = obj;
  proj.schema = SCHEMA;
  proj.kind = 'project';
  proj.meta = proj.meta || {};
  if (!proj.meta.title) proj.meta.title = '未命名设计';
  if (!proj.tokens) proj.tokens = 'design.tokens.json';
  if (!proj.screens || !proj.screens.length) proj.screens = [];
  if (!proj.flow) proj.flow = [];
  proj.artifacts = proj.artifacts || { html: 'design.html', css: 'design.tokens.css', flow: 'design.flow.mmd' };
  for (var i = 0; i < proj.screens.length; i++) {
    var s = proj.screens[i] || {};
    s.id = str(s.id, 'screen' + (i + 1));
    s.name = str(s.name, s.id);
    s.width = num(s.width, 360);
    s.height = num(s.height, 640);
    if (!s.background) s.background = '$color.bg';
    proj.screens[i] = s;
  }
  return proj;
}

function normalizeTokens(obj) {
  if (!obj || typeof obj !== 'object') throw new Error('令牌内容不是对象');
  var sc = str(obj.schema, '');
  if (sc && sc !== SCHEMA) throw new Error('令牌 schema 不匹配：' + sc + '（期望 ' + SCHEMA + '）');
  var t = obj;
  t.schema = SCHEMA;
  t.kind = 'tokens';
  t.meta = t.meta || {};
  if (!t.meta.title) t.meta.title = '设计令牌';
  for (var i = 0; i < TOKEN_GROUPS.length; i++) {
    var g = TOKEN_GROUPS[i];
    if (!t[g] || typeof t[g] !== 'object') t[g] = {};
  }
  return t;
}

function forEachNode(root, fn) {
  if (!root) return;
  fn(root);
  var kids = root.children || [];
  for (var i = 0; i < kids.length; i++) forEachNode(kids[i], fn);
}

function forEachScreenNode(proj, fn) {
  for (var i = 0; i < proj.screens.length; i++) {
    var s = proj.screens[i];
    if (s.root) forEachNode(s.root, function (n) { fn(n, s); });
  }
}

function countNodes(proj) {
  var n = 0;
  forEachScreenNode(proj, function () { n++; });
  return n;
}

// 结构树摘要（Agent 看的是文本 → 必须一眼看出层级与引用）
function nodeTreeLines(node, tokens, depth, out, maxDepth) {
  if (!node) return out;
  var p = node.props || {};
  var extra = [];
  if (node.type === 'text') extra.push(str(p.text, '').slice(0, 24));
  if (node.type === 'button') extra.push(str(p.label, ''));
  if (node.type === 'badge') extra.push(str(p.label, ''));
  if (node.type === 'icon') extra.push(str(p.name, ''));
  if (node.type === 'input') extra.push(str(p.label, '') || str(p.placeholder, ''));
  var size = [];
  if (p.w !== undefined) size.push('w=' + p.w);
  if (p.h !== undefined) size.push('h=' + p.h);
  if (p.pad !== undefined) size.push('pad=' + p.pad);
  if (p.gap !== undefined) size.push('gap=' + p.gap);
  out.push(new Array(depth + 1).join('  ') + '- ' + node.id + ' [' + node.type + '] ' +
    (extra.length ? '"' + extra[0] + '" ' : '') + (size.length ? '{' + size.join(' ') + '}' : ''));
  if (depth >= maxDepth) {
    if (node.children && node.children.length) out.push(new Array(depth + 2).join('  ') + '…(' + node.children.length + ' 个子节点)');
    return out;
  }
  var kids = node.children || [];
  for (var i = 0; i < kids.length; i++) nodeTreeLines(kids[i], tokens, depth + 1, out, maxDepth);
  return out;
}

function projectSummary(proj, path) {
  var lines = [];
  var total = countNodes(proj);
  lines.push('标题：' + proj.meta.title + '｜屏幕 ' + proj.screens.length + ' 个｜节点 ' + total + ' 个｜导航边 ' + (proj.flow || []).length + ' 条');
  lines.push('令牌：' + proj.tokens);
  lines.push('产物：' + JSON.stringify(proj.artifacts || {}));
  for (var i = 0; i < proj.screens.length; i++) {
    var s = proj.screens[i];
    var n = 0;
    if (s.root) forEachNode(s.root, function () { n++; });
    lines.push('');
    lines.push('屏幕 ' + s.id + '「' + s.name + '」' + num(s.width, 0) + '×' + num(s.height, 0) + '｜背景 ' + str(s.background, '$color.bg') + '｜节点 ' + n);
    lines.push(nodeTreeLines(s.root, null, 0, [], 3).join('\n'));
  }
  lines.push('');
  lines.push('文件：' + path);
  return lines.join('\n');
}

// ── 节点查找与目标选择 ─────────────────────────────────────
function findParentOf(root, id) {
  if (!root) return null;
  var kids = root.children || [];
  for (var i = 0; i < kids.length; i++) {
    if (kids[i].id === id) return { parent: root, index: i, list: kids, node: kids[i] };
    var deep = findParentOf(kids[i], id);
    if (deep) return deep;
  }
  return null;
}

function screenById(proj, id) {
  for (var i = 0; i < proj.screens.length; i++) if (proj.screens[i].id === id) return proj.screens[i];
  return null;
}

function latestNodeId(proj) {
  var max = 0;
  forEachScreenNode(proj, function (n) {
    // 兼容带屏幕前缀的 id（home_n6）：取尾部数字，避免新增节点编号与模板撞号
    var m = /(\d+)$/.exec(String(n.id));
    if (m) max = Math.max(max, parseInt(m[1], 10));
  });
  return max;
}

function allocNodeId(proj, reserved) {
  var used = {};
  forEachScreenNode(proj, function (n) { used[String(n.id)] = true; });
  // ★ reserved：本批次正在构建/复制、尚未挂进树的节点 id 也要占用，
  //   否则同一批次内多次分配会拿到同一个 id（实测踩过：duplicate 后 D1 报 id 重复）。
  if (reserved) for (var rk in reserved) used[rk] = true;
  var k = latestNodeId(proj) + 1;
  while (used['n' + k]) k++;
  var id = 'n' + k;
  if (reserved) reserved[id] = true;
  return id;
}

// 模板节点 id 加屏幕前缀：多个屏幕各用一套模板时，id 必须全局唯一（否则 D1 报重复、
// 目标选择出现歧义 —— 实测踩过）。前缀 = 屏幕 id + '_'，确定性且可读（home_n6）。
function prefixIds(node, prefix) {
  node.id = prefix + '_' + node.id;
  var kids = node.children || [];
  for (var i = 0; i < kids.length; i++) prefixIds(kids[i], prefix);
  return node;
}

function depthOf(proj, id) {
  var d = -1;
  for (var i = 0; i < proj.screens.length; i++) {
    var s = proj.screens[i];
    var stack = [[s.root, 0]];
    while (stack.length) {
      var it = stack.pop();
      if (!it[0]) continue;
      if (it[0].id === id) { d = Math.max(d, it[1]); }
      var kids = it[0].children || [];
      for (var k = 0; k < kids.length; k++) stack.push([kids[k], it[1] + 1]);
    }
  }
  return d;
}

// 选取目标节点：op.ids / op.id / op.screen / op.type / op.name（未指定目标直接报错，避免误改全部）
function pickTargets(proj, op, what) {
  var ids = op.ids;
  var single = op.id;
  var screenFilter = op.screen;
  var typeFilter = op.type;
  var nameFilter = op.name;
  var matched = [];
  var wanted = {};
  var hasExplicit = false;
  if (ids && ids.length) { hasExplicit = true; for (var a = 0; a < ids.length; a++) wanted[String(ids[a])] = true; }
  if (single) { hasExplicit = true; wanted[String(single)] = true; }
  for (var i = 0; i < proj.screens.length; i++) {
    var s = proj.screens[i];
    if (screenFilter && s.id !== screenFilter) continue;
    forEachNode(s.root, function (n) {
      if (hasExplicit && !wanted[String(n.id)]) return;
      if (!hasExplicit && typeFilter && n.type !== typeFilter) return;
      if (!hasExplicit && nameFilter && str((n.props || {}).name, '') !== nameFilter) return;
      matched.push({ screen: s, node: n });
    });
  }
  if (hasExplicit) {
    var foundIds = matched.map(function (m) { return m.node.id; });
    for (var kk in wanted) {
      if (foundIds.indexOf(kk) < 0) throw new Error(what + '：目标节点不存在: ' + kk);
    }
  } else if (!typeFilter && !nameFilter) {
    throw new Error(what + '：未指定目标（需给出 ids / id，或用 type/name + screen 过滤）—— 拒绝"无目标即改全部"');
  }
  if (!matched.length) throw new Error(what + '：没有匹配到任何节点');
  return matched;
}

// ── 内置模板（快速产出可看的设计；全部引用 token，天然满足 D2/D4/D5） ──
function tplMobile() {
  return {
    id: 'n1', type: 'frame', props: { w: 'fill', h: 'fill', bg: '$color.bg', pad: '$space.lg', gap: '$space.md' },
    children: [
      {
        id: 'n2', type: 'row', props: { gap: '$space.sm', justify: 'between' }, children: [
          { id: 'n3', type: 'text', props: { text: '今日任务', size: '$fontSize.xl', weight: '$fontWeight.semibold', color: '$color.fg' } },
          { id: 'n4', type: 'icon', props: { name: 'bell', size: 20, color: '$color.muted' } },
        ],
      },
      { id: 'n5', type: 'text', props: { text: '3 项待处理 · 1 项进行中', size: '$fontSize.sm', color: '$color.muted' } },
      {
        id: 'n6', type: 'card', props: { bg: '$color.surface', border: '$color.border', radius: '$radius.md', pad: '$space.lg', gap: '$space.sm' }, children: [
          { id: 'n7', type: 'text', props: { text: '矢量画板插件验收', size: '$fontSize.md', weight: '$fontWeight.medium', color: '$color.fg' } },
          { id: 'n8', type: 'text', props: { text: '面板端到端已通过，等待硬门槛回归', size: '$fontSize.sm', color: '$color.muted' } },
          {
            id: 'n9', type: 'row', props: { gap: '$space.sm' }, children: [
              { id: 'n10', type: 'badge', props: { label: '进行中', tone: 'info' } },
              { id: 'n11', type: 'badge', props: { label: '高优先', tone: 'warning' } },
            ],
          },
        ],
      },
      {
        id: 'n12', type: 'card', props: { bg: '$color.surface', border: '$color.border', radius: '$radius.md', pad: '$space.lg', gap: '$space.sm' }, children: [
          { id: 'n13', type: 'row', props: { gap: '$space.sm' }, children: [
            { id: 'n14', type: 'icon', props: { name: 'check', size: 18, color: '$color.success' } },
            { id: 'n15', type: 'text', props: { text: '设计令牌已冻结 v1', size: '$fontSize.md', color: '$color.fg' } },
          ] },
          { id: 'n16', type: 'text', props: { text: '色板、间距、字号阶梯全部来自 token', size: '$fontSize.sm', color: '$color.muted' } },
        ],
      },
      { id: 'n17', type: 'spacer', props: { h: '$space.md' } },
      { id: 'n18', type: 'button', props: { label: '新建任务', variant: 'primary', full: true, padY: '$space.md' }, children: [] },
    ],
  };
}

function tplPanel() {
  return {
    id: 'n1', type: 'frame', props: { w: 'fill', h: 'fill', bg: '$color.bg', pad: '$space.lg', gap: '$space.md' },
    children: [
      { id: 'n2', type: 'text', props: { text: '设计校验', size: '$fontSize.lg', weight: '$fontWeight.semibold', color: '$color.fg' } },
      { id: 'n3', type: 'text', props: { text: '9 项判据 · 与导出 HTML 同源', size: '$fontSize.sm', color: '$color.muted' } },
      { id: 'n4', type: 'divider', props: { color: '$color.border' } },
      {
        id: 'n5', type: 'row', props: { gap: '$space.sm' }, children: [
          { id: 'n6', type: 'icon', props: { name: 'success', size: 18, color: '$color.success' } },
          { id: 'n7', type: 'text', props: { text: 'D3 对比度达标', size: '$fontSize.sm', color: '$color.fg' } },
        ],
      },
      {
        id: 'n8', type: 'row', props: { gap: '$space.sm' }, children: [
          { id: 'n9', type: 'icon', props: { name: 'success', size: 18, color: '$color.success' } },
          { id: 'n10', type: 'text', props: { text: 'D4 间距落在 4px 网格', size: '$fontSize.sm', color: '$color.fg' } },
        ],
      },
      {
        id: 'n11', type: 'row', props: { gap: '$space.sm' }, children: [
          { id: 'n12', type: 'icon', props: { name: 'warning', size: 18, color: '$color.warning' } },
          { id: 'n13', type: 'text', props: { text: 'D6 触达区偏小（36px）', size: '$fontSize.sm', color: '$color.fg' } },
        ],
      },
      { id: 'n14', type: 'spacer', props: { h: '$space.sm' } },
      {
        id: 'n15', type: 'row', props: { gap: '$space.sm' }, children: [
          { id: 'n16', type: 'button', props: { label: '重新校验', variant: 'primary', padY: '$space.md' } },
          { id: 'n17', type: 'button', props: { label: '导出 HTML', variant: 'secondary', padY: '$space.md' } },
        ],
      },
    ],
  };
}

var TEMPLATES = { mobile: tplMobile, panel: tplPanel };

// ── 编辑命令（命令式 op，与面板操作同源） ─────────────────────
function applyOps(proj, tokens, ops) {
  var log = [];
  for (var i = 0; i < ops.length; i++) {
    var op = ops[i] || {};
    var name = str(op.op, '');
    if (!name) throw new Error('第 ' + (i + 1) + ' 条命令缺少 op 字段');

    if (name === 'project.rename') {
      proj.meta.title = str(op.title, proj.meta.title);
      log.push('重命名工程 → ' + proj.meta.title);
      continue;
    }

    if (name === 'token.set') {
      var grp = str(op.group, ''); var tname = str(op.name, '');
      if (TOKEN_GROUPS.indexOf(grp) < 0) throw new Error('未知令牌组: ' + grp + '（可用：' + TOKEN_GROUPS.join('/') + '）');
      if (!tname) throw new Error('token.set 缺少 name');
      if (op.value === undefined) throw new Error('token.set 缺少 value');
      tokens[grp] = tokens[grp] || {};
      var existed = tname in tokens[grp];
      validateTokenValue(grp, tname, op.value);
      tokens[grp][tname] = op.value;
      log.push('令牌 ' + grp + '.' + tname + ' = ' + op.value + (existed ? '（覆盖）' : '（新增）'));
      continue;
    }
    if (name === 'token.remove') {
      var g2 = str(op.group, ''); var n2 = str(op.name, '');
      if (TOKEN_GROUPS.indexOf(g2) < 0) throw new Error('未知令牌组: ' + g2);
      if (!tokens[g2] || !(n2 in tokens[g2])) throw new Error('令牌不存在: ' + g2 + '.' + n2);
      delete tokens[g2][n2];
      log.push('删除令牌 ' + g2 + '.' + n2);
      continue;
    }

    if (name === 'screen.add') {
      var sid = str(op.id, '');
      if (!sid) throw new Error('screen.add 缺少 id');
      if (/[^A-Za-z0-9_-]/.test(sid)) throw new Error('屏幕 id 只允许字母/数字/下划线/连字符: ' + sid);
      if (screenById(proj, sid)) throw new Error('屏幕 id 已存在: ' + sid);
      if (proj.screens.length >= LIMIT_SCREENS) throw new Error('屏幕数超过上限 ' + LIMIT_SCREENS);
      var sc2 = {
        id: sid,
        name: str(op.name, sid),
        width: num(op.width, 360),
        height: num(op.height, 640),
        background: str(op.background, '$color.bg'),
      };
      var tplName = str(op.template, 'none');
      if (tplName && tplName !== 'none') {
        if (!TEMPLATES[tplName]) throw new Error('未知模板: ' + tplName + '（可用：' + Object.keys(TEMPLATES).join('/') + '/none）');
        sc2.root = TEMPLATES[tplName]();
        prefixIds(sc2.root, sid);
        log.push('屏幕 ' + sid + ' 使用模板 ' + tplName);
      }
      proj.screens.push(sc2);
      log.push('新增屏幕 ' + sid + '「' + sc2.name + '」' + sc2.width + '×' + sc2.height + (sc2.root ? '' : '（空屏）'));
      continue;
    }
    if (name === 'screen.remove') {
      var rid = str(op.id, '');
      var idx = -1;
      for (var si = 0; si < proj.screens.length; si++) if (proj.screens[si].id === rid) idx = si;
      if (idx < 0) throw new Error('屏幕不存在: ' + rid);
      proj.screens.splice(idx, 1);
      proj.flow = (proj.flow || []).filter(function (e) { return e.from !== rid && e.to !== rid; });
      log.push('删除屏幕 ' + rid + '（同时清理其导航边）');
      continue;
    }
    if (name === 'screen.set') {
      var s2 = screenById(proj, str(op.id, ''));
      if (!s2) throw new Error('屏幕不存在: ' + op.id);
      var changed = [];
      if (op.name !== undefined) { s2.name = String(op.name); changed.push('name'); }
      if (op.width !== undefined) { s2.width = num(op.width, s2.width); changed.push('width=' + s2.width); }
      if (op.height !== undefined) { s2.height = num(op.height, s2.height); changed.push('height=' + s2.height); }
      if (op.background !== undefined) { s2.background = String(op.background); changed.push('background=' + s2.background); }
      if (op.template !== undefined) {
        if (op.template === 'none') { delete s2.root; changed.push('root=清空'); }
        else {
          if (!TEMPLATES[op.template]) throw new Error('未知模板: ' + op.template);
          s2.root = TEMPLATES[op.template]();
          prefixIds(s2.root, s2.id);
          changed.push('root=模板 ' + op.template);
        }
      }
      if (!changed.length) throw new Error('screen.set 未提供任何字段（name/width/height/background/template）');
      log.push('更新屏幕 ' + s2.id + '：' + changed.join('、'));
      continue;
    }

    if (name === 'node.add') {
      var scId = str(op.screen, '');
      var screen = scId ? screenById(proj, scId) : (proj.screens[0] || null);
      if (!screen) throw new Error('找不到目标屏幕（工程里还没有屏幕，先用 screen.add）');
      var defs = op.nodes && op.nodes.length ? op.nodes : (op.node ? [op.node] : null);
      var reservedNodes = {};
      if (!defs) throw new Error('node.add 缺少 node（单个）或 nodes（数组）');
      var parentId = str(op.parent, 'root');
      var container = null;
      if (parentId === 'root' || parentId === '') {
        if (!screen.root) {
          // 空屏：第一个节点被提升为根
          screen.root = null;
        }
        container = null;
      } else {
        var found = screen.root ? findParentOf(screen.root, parentId) : null;
        var target = (screen.root && screen.root.id === parentId) ? { node: screen.root } : (found ? { node: found.node } : null);
        if (!target) throw new Error('父节点不存在: ' + parentId + '（屏幕 ' + screen.id + '）');
        if (CONTAINER_TYPES.indexOf(target.node.type) < 0) throw new Error('父节点不是容器类型: ' + parentId + ' (' + target.node.type + ')');
        container = target.node;
      }
      for (var d = 0; d < defs.length; d++) {
        var built = buildNode(defs[d], proj, reservedNodes);
        if (!container) {
          if (screen.root) {
            // 已存在根 → 追加为根的最后一个子节点
            if (CONTAINER_TYPES.indexOf(screen.root.type) < 0) throw new Error('屏幕根不是容器，无法追加：请指定 parent');
            screen.root.children = screen.root.children || [];
            screen.root.children.push(built);
          } else {
            screen.root = built;
          }
        } else {
          container.children = container.children || [];
          var at = op.index === undefined ? container.children.length : clamp(num(op.index, container.children.length), 0, container.children.length);
          if (at === container.children.length) container.children.push(built); else container.children.splice(at, 0, built);
        }
        log.push('新增节点 ' + built.id + ' [' + built.type + ']' + (container ? ' → ' + container.id : ' → 屏幕根'));
      }
      continue;
    }

    if (name === 'node.set') {
      var targets = pickTargets(proj, op, 'node.set');
      var props = op.props && typeof op.props === 'object' ? op.props : null;
      if (!props) throw new Error('node.set 缺少 props 对象（要合并进节点的属性）');
      for (var ti = 0; ti < targets.length; ti++) {
        var tn = targets[ti].node;
        tn.props = tn.props || {};
        for (var pk in props) {
          if (props[pk] === null) delete tn.props[pk]; else tn.props[pk] = props[pk];
        }
      }
      log.push('更新 ' + targets.length + ' 个节点的属性：' + Object.keys(props).join('、'));
      continue;
    }
    if (name === 'node.text') {
      if (op.text === undefined) throw new Error('node.text 缺少 text');
      var tTargets = pickTargets(proj, op, 'node.text');
      for (var t2 = 0; t2 < tTargets.length; t2++) {
        var nn = tTargets[t2].node;
        nn.props = nn.props || {};
        if (nn.type === 'text') nn.props.text = String(op.text);
        else if (nn.type === 'button' || nn.type === 'badge') nn.props.label = String(op.text);
        else throw new Error('节点 ' + nn.id + '（' + nn.type + '）没有文案可改');
      }
      log.push('修改 ' + tTargets.length + ' 个节点文案 → ' + String(op.text).slice(0, 30));
      continue;
    }
    if (name === 'node.remove') {
      var rTargets = pickTargets(proj, op, 'node.remove');
      for (var ri = 0; ri < rTargets.length; ri++) {
        var rn = rTargets[ri].node;
        var rs = rTargets[ri].screen;
        if (rs.root && rs.root.id === rn.id) { delete rs.root; log.push('删除屏幕根节点 ' + rn.id); continue; }
        var loc = findParentOf(rs.root, rn.id);
        if (!loc) throw new Error('找不到节点位置: ' + rn.id);
        loc.list.splice(loc.index, 1);
        log.push('删除节点 ' + rn.id + '（含其子树）');
      }
      continue;
    }
    if (name === 'node.duplicate') {
      var dupId = str(op.id, '');
      if (!dupId) throw new Error('node.duplicate 缺少 id');
      var foundList = [];
      for (var dj = 0; dj < proj.screens.length; dj++) {
        var ds = proj.screens[dj];
        var dl = ds.root ? findParentOf(ds.root, dupId) : null;
        if (dl) foundList.push({ screen: ds, loc: dl });
      }
      if (!foundList.length) throw new Error('节点不存在: ' + dupId);
      var f0 = foundList[0];
      var copy = deepClone(f0.loc.node);
      renumber(copy, proj, {});
      f0.loc.list.splice(f0.loc.index + 1, 0, copy);
      log.push('复制节点 ' + dupId + ' → ' + copy.id + '（子树 ' + countSub(copy) + ' 节点，已整体重编号）');
      continue;
    }
    if (name === 'node.move') {
      var mvId = str(op.id, '');
      if (!mvId) throw new Error('node.move 缺少 id');
      var from = null;
      for (var mi = 0; mi < proj.screens.length; mi++) {
        var ms = proj.screens[mi];
        if (ms.root && ms.root.id === mvId) throw new Error('屏幕根节点不能移动');
        var ml = ms.root ? findParentOf(ms.root, mvId) : null;
        if (ml) { from = { screen: ms, loc: ml }; break; }
      }
      if (!from) throw new Error('节点不存在: ' + mvId);
      var toParentId = str(op.toParent, str(op.parent, ''));
      var mnode = from.loc.node;
      from.loc.list.splice(from.loc.index, 1);
      var placed = false;
      if (!toParentId || toParentId === 'root') {
        var rs2 = from.screen;
        if (!rs2.root) { rs2.root = mnode; placed = true; }
        else if (CONTAINER_TYPES.indexOf(rs2.root.type) >= 0) {
          rs2.root.children = rs2.root.children || [];
          rs2.root.children.push(mnode); placed = true;
        }
      } else {
        var tp = from.screen.root ? findParentOf(from.screen.root, toParentId) : null;
        var tpNode = (from.screen.root && from.screen.root.id === toParentId) ? from.screen.root : (tp ? tp.node : null);
        if (tpNode && CONTAINER_TYPES.indexOf(tpNode.type) >= 0) {
          tpNode.children = tpNode.children || [];
          var at2 = op.index === undefined ? tpNode.children.length : clamp(num(op.index, tpNode.children.length), 0, tpNode.children.length);
          tpNode.children.splice(at2, 0, mnode); placed = true;
        }
      }
      if (!placed) { from.loc.list.splice(from.loc.index, 0, mnode); throw new Error('目标父容器不存在或不是容器类型: ' + toParentId); }
      log.push('移动节点 ' + mvId + ' → ' + (toParentId || '屏幕根'));
      continue;
    }

    if (name === 'flow.add') {
      var f1 = str(op.from, ''); var t3 = str(op.to, '');
      if (!f1 || !t3) throw new Error('flow.add 需要 from 与 to（屏幕 id）');
      if (!screenById(proj, f1)) throw new Error('屏幕不存在: ' + f1);
      if (!screenById(proj, t3)) throw new Error('屏幕不存在: ' + t3);
      proj.flow = proj.flow || [];
      var dup = false;
      for (var fi = 0; fi < proj.flow.length; fi++) if (proj.flow[fi].from === f1 && proj.flow[fi].to === t3) dup = true;
      if (dup) { log.push('导航边已存在，跳过：' + f1 + ' → ' + t3); continue; }
      proj.flow.push({ from: f1, to: t3, label: str(op.label, '') });
      log.push('新增导航边 ' + f1 + ' → ' + t3 + (op.label ? '（' + op.label + '）' : ''));
      continue;
    }
    if (name === 'flow.remove') {
      var f2 = str(op.from, ''); var t4 = str(op.to, '');
      var before2 = (proj.flow || []).length;
      proj.flow = (proj.flow || []).filter(function (e) { return !(e.from === f2 && e.to === t4); });
      if ((proj.flow || []).length === before2) throw new Error('导航边不存在: ' + f2 + ' → ' + t4);
      log.push('删除导航边 ' + f2 + ' → ' + t4);
      continue;
    }
    if (name === 'flow.sync') {
      // 从节点交互（props.action=link:#屏幕id）推导导航边 —— 设计真相源里 flow（信息架构：屏幕之间可达）
      // 与 action:link（屏幕内谁触发跳转）是同一件事的两面，让前者自动跟随后者，两者不再各写一份。
      // 只增不删（保守：手写的边、系统/外部触发的边不会被误删），prune=true 才一并删掉「无交互支撑」的边。
      var wantEdges = []; var wantKey = {};
      for (var sw = 0; sw < proj.screens.length; sw++) {
        var scr0 = proj.screens[sw];
        if (!scr0.root) continue;
        var collect = (function (sid) {
          return function (nd) {
            var aa = nodeAction(nd);
            if (!aa || aa.verb !== 'link' || aa.target.charAt(0) !== '#') return;
            var tg = aa.target.slice(1);
            if (!screenById(proj, tg)) return;   // 目标屏幕不存在 → 由 D10 报错，这里不臆造边
            var kk = sid + '>' + tg;
            if (wantKey[kk]) return;
            wantKey[kk] = true;
            wantEdges.push({ from: sid, to: tg, nodeId: nd.id });
          };
        })(scr0.id);
        forEachNode(scr0.root, collect);
      }
      proj.flow = proj.flow || [];
      var haveKey = {};
      for (var hf2 = 0; hf2 < proj.flow.length; hf2++) haveKey[proj.flow[hf2].from + '>' + proj.flow[hf2].to] = true;
      var added2 = []; var existed2 = 0;
      for (var we2 = 0; we2 < wantEdges.length; we2++) {
        var wk2 = wantEdges[we2].from + '>' + wantEdges[we2].to;
        if (haveKey[wk2]) { existed2++; continue; }
        proj.flow.push({ from: wantEdges[we2].from, to: wantEdges[we2].to, label: str(op.label, '') });
        haveKey[wk2] = true;
        added2.push(wantEdges[we2].from + ' → ' + wantEdges[we2].to + '（触发节点 ' + wantEdges[we2].nodeId + '）');
      }
      var dropped2 = [];
      if (op.prune === true) {
        var kept2 = [];
        for (var pf3 = 0; pf3 < proj.flow.length; pf3++) {
          var pe3 = proj.flow[pf3];
          if (wantKey[pe3.from + '>' + pe3.to]) kept2.push(pe3);
          else dropped2.push(pe3.from + ' → ' + pe3.to);
        }
        proj.flow = kept2;
      }
      log.push('交互推导出 ' + wantEdges.length + ' 条屏幕跳转：新增 ' + added2.length + ' 条导航边，已存在 ' + existed2 + ' 条' +
        (op.prune === true ? ('，删除无交互支撑 ' + dropped2.length + ' 条') : '（未删任何边；prune=true 才删无交互支撑的边）'));
      for (var ai2 = 0; ai2 < added2.length; ai2++) log.push('  + ' + added2[ai2]);
      for (var di2 = 0; di2 < dropped2.length; di2++) log.push('  - ' + dropped2[di2]);
      continue;
    }

    throw new Error('未知 op: ' + name + '（可用：project.rename / token.set / token.remove / screen.add / screen.remove / screen.set / node.add / node.set / node.text / node.remove / node.duplicate / node.move / flow.add / flow.remove / flow.sync）');
  }
  proj.meta.updatedAt = nowIso();
  return log;
}

function countSub(node) {
  var n = 0;
  forEachNode(node, function () { n++; });
  return n;
}

// 构建节点（校验类型/字段；未给 id 用确定性编号）
function buildNode(def, proj, reserved) {
  if (!def || typeof def !== 'object') throw new Error('节点定义必须是对象');
  var type = str(def.type, '');
  if (NODE_TYPES.indexOf(type) < 0) throw new Error('未知节点类型: ' + type + '（可用：' + NODE_TYPES.join('/') + '）');
  var node = {
    id: str(def.id, allocNodeId(proj, reserved)),
    type: type,
    props: {},
  };
  if (typeof def.props === 'object' && def.props) {
    for (var k in def.props) node.props[k] = def.props[k];
  }
  var fields = NODE_FIELDS[type];
  for (var f = 0; f < fields.length; f++) {
    if (def[fields[f]] !== undefined) node.props[fields[f]] = def[fields[f]];
  }
  if (def.name !== undefined) node.props.name = def.name;
  if (def.children && def.children.length) {
    if (CONTAINER_TYPES.indexOf(type) < 0) throw new Error(type + ' 不能有子节点（只有 ' + CONTAINER_TYPES.join('/') + ' 是容器）');
    node.children = [];
    for (var c = 0; c < def.children.length; c++) node.children.push(buildNode(def.children[c], proj, reserved));
  }
  return node;
}

// 复制子树时整体重编号（保持结构、避免 id 冲突）
function renumber(node, proj, reserved) {
  node.id = allocNodeId(proj, reserved);
  var kids = node.children || [];
  for (var i = 0; i < kids.length; i++) renumber(kids[i], proj, reserved);
  return node;
}

// 各类型的「平铺字段」（允许直接在 node 定义里给字段，不必嵌 props 里）
var NODE_FIELDS = {
  frame: ['w', 'h', 'pad', 'gap', 'bg', 'radius', 'border', 'shadow', 'align', 'justify'],
  row: ['w', 'h', 'pad', 'gap', 'bg', 'radius', 'border', 'shadow', 'align', 'justify'],
  col: ['w', 'h', 'pad', 'gap', 'bg', 'radius', 'border', 'shadow', 'align', 'justify'],
  card: ['w', 'h', 'pad', 'gap', 'bg', 'radius', 'border', 'shadow', 'align', 'justify'],
  list: ['w', 'h', 'pad', 'gap', 'bg', 'radius', 'border', 'shadow', 'align', 'justify'],
  text: ['text', 'w', 'size', 'weight', 'leading', 'color', 'maxLines', 'align'],
  button: ['label', 'w', 'h', 'size', 'padX', 'padY', 'variant', 'full', 'disabled'],
  input: ['w', 'h', 'label', 'placeholder', 'value', 'disabled'],
  badge: ['label', 'tone', 'size', 'padX', 'padY'],
  icon: ['name', 'size', 'color'],
  image: ['w', 'h', 'ratio', 'alt', 'radius'],
  divider: ['h', 'color'],
  spacer: ['h'],
  checkbox: ['label', 'checked', 'w', 'h'],
};

// ── 内容字段（组件参数化的提升对象）─────────────────────────
// 每种类型「用户可见文案」所在的字段：抽成组件后这些字段提升为 props（不传则用设计原值兜底）。
// input 是特例：填了 value 用 value，否则用 placeholder（设计稿里常只填 placeholder）。
var CONTENT_FIELDS = {
  text: 'text', button: 'label', badge: 'label', checkbox: 'label', image: 'alt',
};
function contentField(node) {
  var t = String(node.type);
  if (t === 'input') return str((node.props || {}).value, '') ? 'value' : 'placeholder';
  return CONTENT_FIELDS[t] || '';
}
function contentValue(node) {
  var f = contentField(node);
  return f ? str((node.props || {})[f], '') : '';
}
// 插槽宿主：props.slot = true | '<名字>' → 抽成组件时该节点的**内容**由外部传入（外壳与样式留在组件里，
//   渲染为 <slot /> / {children}；屏幕引用处把它的子树写进组件标签内部）。
function isSlotHost(node) {
  var v = (node.props || {}).slot;
  return v === true || (typeof v === 'string' && v.length > 0);
}
function slotName(node) {
  var v = (node.props || {}).slot;
  return (typeof v === 'string') ? v : '';
}

// slotNameNormalize：插槽名归一化 —— Vue 的 `#name` 与 React 的 props 名共用一套写法
//   （React 属性名不能带 `-`，所以统一转 camelCase：'left-panel' → 'leftPanel'）
function slotNameNormalize(s) {
  var v = String(s == null ? '' : s).trim();
  if (!v) return '';
  var parts = v.split(/[^A-Za-z0-9]+/);
  var out = '';
  for (var i = 0; i < parts.length; i++) {
    var seg = parts[i];
    if (!seg) continue;
    out += out ? (seg.charAt(0).toUpperCase() + seg.slice(1)) : (seg.charAt(0).toLowerCase() + seg.slice(1));
  }
  if (!out) return '';
  if (/^[0-9]/.test(out)) out = 's' + out;
  return out;
}

// ── 交互（节点 props.action：受控动作词汇表）────────────────────────
// 设计真相源只声明**交互意图**，不携带脚本 —— 自由 JS 无法校验，也不该由设计稿承担
// （产物里更不该内联，D9 明确禁止 on* / javascript:）。词汇表：
//   link:<目标>    跳转 → 产物渲染**真链接** <a href>（目标：https:// http:// mailto: tel: /站内路径 #屏幕id）
//   emit:<事件名>  交给宿主处理 → Vue emit（defineEmits）/ React 回调 prop（on<Name>）/ HTML 出 data-action
//   toggle         切换 → 仅 checkbox（原生交互，不产代码）
var ACTION_VERBS = ['link', 'emit', 'toggle'];
var LINK_PROTOCOLS = ['https://', 'http://', 'mailto:', 'tel:'];
var EMIT_NAME_RE = /^[A-Za-z][A-Za-z0-9]*$/;
// parseAction：非法时返回 { verb:'invalid', error }（codegen 告警、校验器 D10 FAIL —— 都不静默）
function parseAction(raw) {
  if (raw === undefined || raw === null) return null;
  var s = str(raw, '').trim();
  if (!s) return null;
  var ci = s.indexOf(':');
  var verb = (ci < 0 ? s : s.slice(0, ci)).trim().toLowerCase();
  var target = ci < 0 ? '' : s.slice(ci + 1).trim();
  function bad(msg) { return { verb: 'invalid', target: target, raw: s, error: msg }; }
  if (ACTION_VERBS.indexOf(verb) < 0) {
    return bad('未知动作「' + verb + '」：支持 link:<目标> / emit:<事件名> / toggle');
  }
  if (verb === 'toggle') {
    if (target) return bad('toggle 不接受目标（写法就是 toggle）');
    return { verb: 'toggle', target: '', raw: s };
  }
  if (!target) return bad(verb + ' 缺少目标：写法 ' + verb + ':<' + (verb === 'link' ? '目标' : '事件名') + '>');
  if (verb === 'emit') {
    if (!EMIT_NAME_RE.test(target)) return bad('事件名要以字母开头、只含字母数字：' + target);
    return { verb: 'emit', target: target, raw: s };
  }
  var okTarget = (target.charAt(0) === '/' || target.charAt(0) === '#');
  for (var i = 0; i < LINK_PROTOCOLS.length; i++) {
    if (target.toLowerCase().indexOf(LINK_PROTOCOLS[i]) === 0) okTarget = true;
  }
  if (!okTarget) return bad('链接目标不允许：' + target + '（支持 https:// http:// mailto: tel: /站内路径 #屏幕id）');
  return { verb: 'link', target: target, raw: s };
}
function nodeAction(node) { return parseAction((node.props || {}).action); }
// actionHref：link 目标 → 产物 href（#屏幕id → ./<屏幕>.html；屏幕即页面，与 html 产物同构）
function actionHref(target) {
  if (target.charAt(0) === '#') return './' + safeFileName(target.slice(1)) + '.html';
  return target;
}
// link 只对"可点元素"有意义：input/checkbox/divider/spacer 没有链接语义（校验器会报错）
var LINKABLE_TYPES = ['frame', 'row', 'col', 'card', 'list', 'text', 'button', 'badge', 'icon', 'image'];
// 自宽叶子：宽度由内容决定的元素。放进 col 容器时 CSS flex 默认 align-items: stretch 会把它们拉到满宽
//   （实测：near 组件里的 badge 渲染成一条贯穿整行的色带 —— 与设计预览的绝对定位不一致）→ 显式按内容宽。
var SELF_WIDTH_TYPES = { badge: 1, button: 1, icon: 1, checkbox: 1 };
// optsWithDir：给子树带上父容器方向（只读副本，不改共享 opts 的其它字段）
function optsWithDir(opts, dir) {
  if (!opts) return null;
  var o = {};
  for (var k in opts) o[k] = opts[k];
  o.parentDir = dir;
  return o;
}
// handlerName：emit 名 → React 回调 prop 名（submit → onSubmit）
function handlerName(evt) { return 'on' + evt.charAt(0).toUpperCase() + evt.slice(1); }
// collectActions：结构树里的 emit 事件（去重 + DFS 顺序 → script 段确定性）
function collectActions(tree) {
  var seen = {};
  var emits = [];
  function walk(t) {
    var a = t.action;
    if (a && a.verb === 'emit' && !seen[a.target]) { seen[a.target] = true; emits.push(a.target); }
    for (var i = 0; i < t.children.length; i++) walk(t.children[i]);
  }
  walk(tree);
  return emits;
}
// emitList：事件名数组 → Vue defineEmits 参数列表（'submit', 'close'）
function emitList(emits) {
  var parts = [];
  for (var i = 0; i < emits.length; i++) parts.push(jsStr(emits[i]));
  return parts.join(', ');
}

// resolveSlotNames：组件原型的插槽宿主 → 确定性插槽表 [{ nodeId, name, declared, hostType }]
//   规则：① props.slot 给名字 → 归一化后用；② 只有一个宿主且未命名 → default（默认插槽）；
//         ③ 多个宿主里未命名的按 DFS 序号命名 slot1/slot2… 并告警（产物必须确定，不能含糊）；
//         ④ 重名 / 非法名 / 保留名（React 的 children）/ 与文案 prop 撞名 → 直接报错（否则产物非法）。
function resolveSlotNames(proto, propNames, compName, warnings) {
  var hosts = [];
  forEachNode(proto, function (n) { if (isSlotHost(n)) hosts.push(n); });
  var slots = [];
  var used = {};
  var defaultUsed = false;
  for (var i = 0; i < hosts.length; i++) {
    var raw = slotName(hosts[i]);
    var nm = raw ? slotNameNormalize(raw) : '';
    if (raw && !nm) {
      throw new Error('组件 ' + compName + ' 的插槽名不合法（节点 ' + hosts[i].id + ' props.slot=' + JSON.stringify(raw) +
        '）：请用字母/数字，可用 - _ 分隔（如 "header" / "left-panel"）');
    }
    if (!nm) {
      // 第一个未命名宿主 → default（默认插槽）；其余未命名 → slot<DFS 位次>（确定性回退，不静默丢弃）
      nm = defaultUsed ? ('slot' + (i + 1)) : 'default';
      if (hosts.length > 1) {
        warnings.push('组件 ' + compName + ' 有 ' + hosts.length + ' 个插槽宿主，节点 ' + hosts[i].id +
          ' 没写名字（props.slot: true）→ 自动命名为 "' + nm + '"' +
          (nm === 'default' ? '（默认插槽）' : '') +
          '（想自定义：design_edit node.set props={"slot":"header"}）');
      }
    }
    if (nm === 'default') defaultUsed = true;
    if (nm === 'children') {
      throw new Error('组件 ' + compName + ' 的插槽名不能叫 "children"（React 组件里这是默认子内容的保留名）：换个名字');
    }
    if (used[nm]) {
      throw new Error('组件 ' + compName + ' 的插槽名重复："' + nm + '"（节点 ' + used[nm] + ' 与 ' + hosts[i].id +
        '）：请给其中一个换名字');
    }
    if (nm !== 'default' && propNames && propNames[nm]) {
      throw new Error('组件 ' + compName + ' 的插槽名与文案 prop 重名："' + nm +
        '"（React 里插槽与 prop 都是 props，会互相覆盖）：用 props.prop 给该文案换个 prop 名');
    }
    used[nm] = hosts[i].id;
    slots.push({ nodeId: String(hosts[i].id), name: nm, declared: !!raw, hostType: String(hosts[i].type) });
  }
  return slots;
}

// treeSlotName：组件产物里该插槽节点的名字（查 comp.slotNames；无表时回退 props.slot 归一化 / default）
function treeSlotName(node, opts) {
  var comp = (opts && opts.comp) || null;
  if (comp && comp.slotNames) {
    var nm = comp.slotNames[String(node.id)];
    if (nm) return nm;
  }
  var raw = slotNameNormalize(slotName(node));
  return raw || 'default';
}

// ── 产物渲染（绝对定位；与布局引擎共用同一份盒子 → 校验=截图） ──
function cssColor(v, tokens, fallback) {
  var c = resolveColor(v, tokens);
  if (c === 'BADREF' || c === undefined || !c) return fallback;
  return colorCss(c);
}
function cssNum(v, tokens, fallback) {
  var n = resolveSize(v, tokens);
  if (typeof n !== 'number') return fallback;
  return String(n) + 'px';
}
// 阴影类属性（令牌存的是字符串，如 '0 1px 2px rgba(...)'；不走颜色解析）
function cssShadow(v, tokens, fallback) {
  if (v === undefined || v === null || v === 'none') return fallback;
  var s = String(v);
  if (s.charAt(0) === '$') {
    var t = lookupToken(tokens, s);
    if (!t.found) return fallback;
    return String(t.value);
  }
  return s;
}

function variantColors(variant, tokens) {
  var k = str(variant, 'primary');
  if (k === 'secondary') {
    return {
      bg: cssColor('$color.surface', tokens, '#F9FAFB'),
      fg: cssColor('$color.fg', tokens, '#111827'),
      border: cssColor('$color.border', tokens, '#E5E7EB'),
    };
  }
  if (k === 'ghost') {
    return { bg: 'transparent', fg: cssColor('$color.accent', tokens, '#2563EB'), border: 'transparent' };
  }
  if (k === 'danger') {
    return { bg: cssColor('$color.danger', tokens, '#B91C1C'), fg: cssColor('$color.accent-fg', tokens, '#FFFFFF'), border: 'transparent' };
  }
  return { bg: cssColor('$color.accent', tokens, '#2563EB'), fg: cssColor('$color.accent-fg', tokens, '#FFFFFF'), border: 'transparent' };
}

function badgeColors(tone, tokens) {
  var bg = cssColor('$color.surface-2', tokens, '#F3F4F6');
  var k = str(tone, 'neutral');
  if (k === 'info') return { bg: cssColor('$color.accent-soft', tokens, '#DBEAFE'), fg: cssColor('$color.accent', tokens, '#2563EB') };
  if (k === 'success') return { bg: bg, fg: cssColor('$color.success', tokens, '#047857') };
  if (k === 'warning') return { bg: bg, fg: cssColor('$color.warning', tokens, '#B45309') };
  if (k === 'danger') return { bg: bg, fg: cssColor('$color.danger', tokens, '#B91C1C') };
  return { bg: bg, fg: cssColor('$color.muted', tokens, '#6B7280') };
}

function escHtml(s) {
  return String(s).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}

function iconSvg(name, size, color, tokens, strokeWidth) {
  var d = ICONS[name];
  if (!d) return '';
  return '<svg width="' + fmtNum(size) + '" height="' + fmtNum(size) + '" viewBox="0 0 24 24" fill="none" ' +
    'stroke="' + color + '" stroke-width="' + fmtNum(num(strokeWidth, 1.6)) + '" stroke-linecap="round" stroke-linejoin="round">' +
    '<path d="' + d + '"/></svg>';
}

function renderNodeHtml(node, boxes, tokens, out, inheritedBg) {
  var b = boxes[node.id];
  if (!b) return;
  var p = node.props || {};
  var ownBg = resolveColor(p.bg, tokens);
  if (ownBg === 'BADREF') ownBg = undefined;
  // 交互：预览是**静态截图式**产物（D9 禁脚本/内联事件/外链）→ 只声明动作类型，不落 URL 与处理器
  var pa = nodeAction(node);
  var effBg = (ownBg && !ownBg.none && ownBg.a > 0.001) ? ownBg : inheritedBg;
  var effBgCss = effBg ? colorCss(effBg) : 'transparent';
  var base = 'left:' + fmtNum(b.x) + 'px;top:' + fmtNum(b.y) + 'px;width:' + fmtNum(b.w) + 'px;height:' + fmtNum(b.h) + 'px;';
  var style = base;
  if (ownBg && ownBg.none !== true && ownBg.a > 0.001) style += 'background:' + colorCss(ownBg) + ';';
  if (p.border !== undefined && !(ownBg && ownBg.none)) style += 'border:1px solid ' + cssColor(p.border, tokens, 'transparent') + ';';
  else if (node.type === 'input') style += 'border:1px solid ' + cssColor('$color.border', tokens, '#E5E7EB') + ';background:' + cssColor('$color.bg', tokens, '#fff') + ';';
  if (p.radius !== undefined && p.radius !== 'none') style += 'border-radius:' + cssNum(p.radius, tokens, '0px') + ';';
  if (p.shadow !== undefined && p.shadow !== 'none') style += 'box-shadow:' + cssShadow(p.shadow, tokens, 'none') + ';';

  var inner = '';
  if (node.type === 'text') {
    var fs = num(b.metrics ? b.metrics.fontSize : 14, 14);
    var fw = num(b.metrics ? b.metrics.weight : 400, 400);
    var lh = num(b.metrics ? b.metrics.leading : 1.5, 1.5);
    style += 'font-size:' + fmtNum(fs) + 'px;font-weight:' + fmtNum(fw) + ';line-height:' + fmtNum(lh) + ';';
    style += 'color:' + cssColor(p.color !== undefined ? p.color : '$color.fg', tokens, '#111827') + ';';
    style += 'text-align:' + (str(p.align, 'left') === 'center' ? 'center' : (str(p.align, 'left') === 'right' ? 'right' : 'left')) + ';';
    style += 'overflow:hidden;overflow-wrap:break-word;';
    inner = escHtml(str(p.text, ''));
  } else if (node.type === 'button') {
    var vc = variantColors(p.variant, tokens);
    style += 'background:' + vc.bg + ';color:' + vc.fg + ';border:1px solid ' + vc.border + ';';
    style += 'font-size:' + fmtNum(num(b.metrics ? b.metrics.fontSize : 14, 14)) + 'px;font-weight:600;';
    style += 'border-radius:' + (p.radius !== undefined ? cssNum(p.radius, tokens, '6px') : '6px') + ';';
    style += 'display:flex;align-items:center;justify-content:center;';
    if (argBoolish(p.disabled)) style += 'opacity:0.5;';
    inner = escHtml(str(p.label, ''));
  } else if (node.type === 'input') {
    style += 'display:flex;align-items:center;padding:0 10px;font-size:13px;color:' + cssColor('$color.muted', tokens, '#6B7280') + ';border-radius:6px;';
    inner = escHtml(str(p.value, '') || str(p.placeholder, '') || str(p.label, ''));
  } else if (node.type === 'checkbox') {
    var bc = badgeColors('neutral', tokens);
    style += 'display:flex;align-items:center;gap:8px;font-size:13px;color:' + cssColor('$color.fg', tokens, '#111827') + ';';
    inner = '<span style="display:inline-block;width:14px;height:14px;border:1px solid ' + cssColor('$color.border', tokens, '#E5E7EB') + ';border-radius:3px;' +
      (p.checked === true ? ('background:' + cssColor('$color.accent', tokens, '#2563EB') + ';') : '') + 'flex:none"></span>' + escHtml(str(p.label, ''));
  } else if (node.type === 'badge') {
    var bcol = badgeColors(p.tone, tokens);
    style += 'background:' + bcol.bg + ';color:' + bcol.fg + ';border-radius:999px;';
    style += 'font-size:' + fmtNum(num(b.metrics ? b.metrics.fontSize : 11, 11)) + 'px;font-weight:500;';
    style += 'display:flex;align-items:center;justify-content:center;white-space:nowrap;';
    inner = escHtml(str(p.label, ''));
  } else if (node.type === 'icon') {
    var icol = cssColor(p.color !== undefined ? p.color : '$color.fg', tokens, '#111827');
    var isz = num(b.w, 20);
    inner = iconSvg(str(p.name, ''), isz, icol, tokens, 1.6);
    if (!inner) inner = '<span style="display:block;width:100%;height:100%;border:1px dashed ' + cssColor('$color.border', tokens, '#E5E7EB') + ';"></span>';
  } else if (node.type === 'image') {
    style += 'background:' + cssColor('$color.surface-2', tokens, '#F3F4F6') + ';border:1px solid ' + cssColor('$color.border', tokens, '#E5E7EB') + ';';
    style += 'border-radius:' + (p.radius !== undefined ? cssNum(p.radius, tokens, '6px') : '6px') + ';';
    style += 'display:flex;align-items:center;justify-content:center;font-size:11px;color:' + cssColor('$color.subtle', tokens, '#9CA3AF') + ';';
    inner = escHtml(str(p.alt, '图片'));
  } else if (node.type === 'divider') {
    style += 'background:' + cssColor(p.color !== undefined ? p.color : '$color.border', tokens, '#E5E7EB') + ';';
  } else {
    // 容器：仅背景/边框/圆角
  }
  out.push('<div class="nd"' + ((pa && pa.verb !== 'invalid') ? (' data-action="' + pa.verb + '"') : '') +
    ' style="' + style + '">' + inner + '</div>');
  var kids = node.children || [];
  for (var i = 0; i < kids.length; i++) renderNodeHtml(kids[i], boxes, tokens, out, effBg);
}
function argBoolish(v) { return v === true || v === 'true' || v === 1; }

function renderScreenHtml(screen, tokens, layout) {
  var bgCss = cssColor(screen.background, tokens, '#FFFFFF');
  var bgColor = resolveColor(screen.background, tokens);
  var bgForInherit = (bgColor === 'BADREF' || !bgColor) ? null : bgColor;
  var parts = [];
  parts.push('<div class="screen" data-screen="' + escHtml(screen.id) + '" style="width:' + fmtNum(num(screen.width, 360)) +
    'px;height:' + fmtNum(num(screen.height, 640)) + 'px;background:' + bgCss + ';">');
  if (screen.root) renderNodeHtml(screen.root, layout.boxes, tokens, parts, bgForInherit);
  parts.push('</div>');
  return parts.join('\n');
}

function renderProjectHtml(proj, tokens) {
  var famTok = lookupToken(tokens, '$font.sans');
  var family = famTok.found ? String(famTok.value) : 'system-ui, sans-serif';
  var css = [
    '*{box-sizing:border-box}',
    'body{margin:0;padding:24px;background:#E5E7EB;font-family:' + family + ';-webkit-font-smoothing:antialiased}',
    '.screen{position:relative;overflow:hidden;margin:0 auto 24px;box-shadow:0 4px 12px rgba(17,24,39,0.08)}',
    '.nd{position:absolute}',
  ].join('\n');
  var body = [];
  for (var i = 0; i < proj.screens.length; i++) {
    var s = proj.screens[i];
    var layout = layoutScreen(s, tokens);
    body.push(renderScreenHtml(s, tokens, layout));
  }
  return '<!DOCTYPE html>\n<html lang="zh-CN">\n<head>\n<meta charset="utf-8">\n<title>' + escHtml(proj.meta.title) +
    ' · design preview</title>\n<style>\n' + css + '\n</style>\n</head>\n<body>\n' + body.join('\n') + '\n</body>\n</html>\n';
}

// ★ 令牌值 → CSS 值：尺寸类令牌（space/radius/fontSize）在工程里存的是**裸数字**
//   （space.md = 12），直接写进自定义属性是 **无效 CSS**（`--space-md: 12` 不是长度）。
//   旧版本正是如此 —— design_export format=css 的产物拿进前端完全不生效。
//   此处统一补 px；无量纲组（fontWeight/lineHeight）与字符串组（color/shadow/font）原样。
function tokenValueCss(group, value) {
  if (group === 'space' || group === 'radius' || group === 'fontSize') {
    var n = num(value, NaN);
    return isNum(n) ? fmtNum(n) + 'px' : '0px';
  }
  return String(value);
}

function renderTokensCss(tokens) {
  var lines = ['/* 设计令牌 → CSS 自定义属性（由 tool-design 生成，勿手改；真相源为 design.tokens.json） */', ':root {'];
  for (var i = 0; i < TOKEN_GROUPS.length; i++) {
    var g = TOKEN_GROUPS[i];
    var obj = tokens[g] || {};
    var keys = Object.keys(obj);
    if (!keys.length) continue;
    lines.push('  /* ' + g + ' */');
    for (var k = 0; k < keys.length; k++) {
      var nm = keys[k];
      var val = obj[nm];
      var cssVal = tokenValueCss(g, val);
      lines.push('  --' + g.replace(/[A-Z]/g, function (m) { return '-' + m.toLowerCase(); }) + '-' + String(nm) + ': ' + cssVal + ';');
    }
  }
  lines.push('}');
  lines.push('');
  return lines.join('\n');
}

function mmId(s) { return String(s).replace(/[^A-Za-z0-9_]/g, '_'); }

function renderMermaid(proj, tokens) {
  var lines = ['flowchart TD'];
  var screenIds = {};
  for (var i = 0; i < proj.screens.length; i++) {
    var s = proj.screens[i];
    var sid = 's_' + mmId(s.id);
    screenIds[s.id] = sid;
    lines.push('  subgraph ' + sid + '["' + s.name + ' · ' + fmtNum(num(s.width, 0)) + '×' + fmtNum(num(s.height, 0)) + '"]');
    var kids = (s.root && s.root.children) ? s.root.children : [];
    if (!kids.length) lines.push('    ' + sid + '_empty["（空屏）"]');
    for (var k = 0; k < kids.length; k++) {
      var nd = kids[k];
      var label = nodeLabel(nd);
      lines.push('    ' + sid + '_' + mmId(nd.id) + '["' + label + '"]');
    }
    lines.push('  end');
  }
  for (var f = 0; f < (proj.flow || []).length; f++) {
    var e = proj.flow[f];
    if (!screenIds[e.from] || !screenIds[e.to]) continue;
    lines.push('  ' + screenIds[e.from] + ' ' + (e.label ? '-- "' + e.label + '" -->' : '-->') + ' ' + screenIds[e.to]);
  }
  lines.push('');
  return lines.join('\n');
}
function nodeLabel(node) {
  var p = node.props || {};
  var t = node.type;
  if (t === 'text') return (str(p.text, '') || 'text').slice(0, 18);
  if (t === 'button' || t === 'badge') return t + ': ' + str(p.label, '');
  if (t === 'icon') return 'icon: ' + str(p.name, '');
  return t + ' ' + node.id;
}

// ── 校验器（9 项判据） ─────────────────────────────────────
function tokColor(tokens, ref) {
  var r = resolveColor(ref, tokens);
  return (r === 'BADREF' || !r) ? null : r;
}
function variantColorObjs(variant, tokens) {
  var k = str(variant, 'primary');
  if (k === 'secondary') return { bg: tokColor(tokens, '$color.surface'), fg: tokColor(tokens, '$color.fg') };
  if (k === 'ghost') return { bg: null, fg: tokColor(tokens, '$color.accent') };
  if (k === 'danger') return { bg: tokColor(tokens, '$color.danger'), fg: tokColor(tokens, '$color.accent-fg') };
  return { bg: tokColor(tokens, '$color.accent'), fg: tokColor(tokens, '$color.accent-fg') };
}
function badgeColorObjs(tone, tokens) {
  var k = str(tone, 'neutral');
  var soft = tokColor(tokens, '$color.surface-2');
  if (k === 'info') return { bg: tokColor(tokens, '$color.accent-soft'), fg: tokColor(tokens, '$color.accent') };
  if (k === 'success') return { bg: soft, fg: tokColor(tokens, '$color.success') };
  if (k === 'warning') return { bg: soft, fg: tokColor(tokens, '$color.warning') };
  if (k === 'danger') return { bg: soft, fg: tokColor(tokens, '$color.danger') };
  // neutral 徽章：浅灰底上用正文色（用 muted 会低于 4.5:1，D3 会抓 —— 那是正确行为）
  return { bg: soft, fg: tokColor(tokens, '$color.fg') };
}

var EMOJI_RANGES = [[0x1f000, 0x1faff], [0x2600, 0x27bf], [0x2b00, 0x2bff], [0xfe00, 0xfe0f], [0x1f1e6, 0x1f1ff]];
// 注意：emoji 多数在 BMP 之外（U+1F300+），charCodeAt 只能取到代理对高位 —— 必须手工合成码点，
// 否则 🚀(U+1F680) 会漏检（实测踩过：D7 负向用例假通过）。
function findEmoji(s) {
  var t = String(s);
  for (var i = 0; i < t.length; i++) {
    var c = t.charCodeAt(i);
    var cp = c;
    if (c >= 0xd800 && c <= 0xdbff && i + 1 < t.length) {
      var c2 = t.charCodeAt(i + 1);
      if (c2 >= 0xdc00 && c2 <= 0xdfff) { cp = (c - 0xd800) * 0x400 + (c2 - 0xdc00) + 0x10000; i++; }
    }
    for (var r = 0; r < EMOJI_RANGES.length; r++) {
      if (cp >= EMOJI_RANGES[r][0] && cp <= EMOJI_RANGES[r][1]) {
        if (cp > 0xffff) {
          return String.fromCharCode(0xd800 + ((cp - 0x10000) >> 10), 0xdc00 + ((cp - 0x10000) & 0x3ff));
        }
        return t.charAt(i);
      }
    }
  }
  return '';
}

// 字段 → 期望令牌组（D2 抓"颜色当宽度用"这类错用）
var FIELD_TOKEN_GROUP = {
  bg: ['color'], border: ['color'], color: ['color'], fill: ['color'],
  w: ['space', 'radius'], h: ['space', 'radius'], pad: ['space', 'radius'], gap: ['space', 'radius'],
  padX: ['space', 'radius'], padY: ['space', 'radius'], radius: ['radius', 'space'],
  size: ['fontSize', 'fontWeight', 'space', 'radius'], weight: ['fontWeight'], leading: ['lineHeight'],
  shadow: ['shadow'],
};
var GRID_FIELDS = ['pad', 'gap', 'padX', 'padY'];

function verifyProject(proj, tokens, opts) {
  var o = opts || {};
  var checks = [];
  var minContrast = num(o.minContrast, 0);
  var errs1 = []; var errs2 = []; var errs3 = []; var errs4 = []; var errs5 = []; var errs6 = []; var errs7 = [];
  var errs10 = [];
  var actionStats = { count: 0, link: 0, emit: 0, toggle: 0 };
  var linkEdges = [];        // D11：节点声明的「跳屏幕」交互（from=所在屏幕，to=目标屏幕）
  var linkExternal = 0;      // D11：外链/站内路径链接（不参与导航图一致性）
  var flowNoTrigger = [];    // D11：有导航边但没有任何节点声明跳转（提示，不判 FAIL）
  var layouts = {};
  var allIds = {};
  var totalNodes = 0;
  var textCount = 0; var contrastWorst = null; var contrastWorstWho = '';
  var touchWorst = null; var touchWorstWho = '';
  var maxDepth = 0;
  var warn2 = [];

  if (!proj.screens.length) errs1.push('工程里没有任何屏幕（screen）');
  if (proj.screens.length > LIMIT_SCREENS) errs1.push('屏幕数超过上限 ' + LIMIT_SCREENS);
  var screenIds = {};

  for (var i = 0; i < proj.screens.length; i++) {
    var s = proj.screens[i];
    if (screenIds[s.id]) errs1.push('屏幕 id 重复: ' + s.id);
    screenIds[s.id] = true;
    if (!isInt(num(s.width, NaN)) || !isInt(num(s.height, NaN))) errs1.push('屏幕 ' + s.id + ' 尺寸不是整数：' + s.width + '×' + s.height);
    if (num(s.width, 0) < 1 || num(s.width, 0) > LIMIT_DIM || num(s.height, 0) < 1 || num(s.height, 0) > LIMIT_DIM) {
      errs1.push('屏幕 ' + s.id + ' 尺寸越界（1..' + LIMIT_DIM + '）：' + s.width + '×' + s.height);
    }
    var bgC = resolveColor(s.background, tokens);
    if (bgC === 'BADREF' || !bgC) errs1.push('屏幕 ' + s.id + ' 背景无法解析: ' + s.background);
    else if (String(s.background).charAt(0) === '$' && lookupToken(tokens, s.background).group !== 'color') {
      errs1.push('屏幕 ' + s.id + ' 背景引用了非颜色令牌: ' + s.background);
    }
    if (!s.root) errs1.push('屏幕 ' + s.id + ' 没有任何节点（先用 node.add 或 template）');
    layouts[s.id] = layoutScreen(s, tokens);
  }

  for (var fi = 0; fi < (proj.flow || []).length; fi++) {
    var e = proj.flow[fi];
    if (!screenIds[e.from]) errs1.push('导航边起点屏幕不存在: ' + e.from);
    if (!screenIds[e.to]) errs1.push('导航边终点屏幕不存在: ' + e.to);
  }

  // 逐节点检查（结构 / 令牌 / 对比度 / 网格 / 字号 / 触达区 / 图标 / emoji）
  function walk(node, screen, inheritedBg, depth) {
    totalNodes++;
    if (depth > maxDepth) maxDepth = depth;
    if (depth > LIMIT_DEPTH) errs1.push('节点树过深（>' + LIMIT_DEPTH + '）：' + node.id);
    if (!node.id) errs1.push('存在没有 id 的节点');
    else {
      if (allIds[node.id]) errs1.push('节点 id 重复: ' + node.id);
      allIds[node.id] = true;
    }
    if (NODE_TYPES.indexOf(node.type) < 0) { errs1.push('未知节点类型: ' + node.type + '（' + node.id + '）'); return; }
    var isC = CONTAINER_TYPES.indexOf(node.type) >= 0;
    if (!isC && node.children && node.children.length) errs1.push('非容器节点有子节点: ' + node.id + '（' + node.type + '）');
    var p = node.props || {};
    var box = layouts[screen.id].boxes[node.id];

    // 必填字段
    if (node.type === 'text' && !str(p.text, '').trim()) errs1.push('text 节点没有文案: ' + node.id);
    if ((node.type === 'button' || node.type === 'badge') && !str(p.label, '').trim()) errs1.push(node.type + ' 节点没有 label: ' + node.id);
    if (node.type === 'icon' && !str(p.name, '')) errs1.push('icon 节点没有 name: ' + node.id);
    if (node.type === 'image' && !str(p.alt, '')) warn2.push('image 节点缺 alt（无障碍）：' + node.id);
    if (node.type === 'button' && p.variant !== undefined && ['primary', 'secondary', 'ghost', 'danger'].indexOf(String(p.variant)) < 0) {
      errs1.push('button variant 非法: ' + p.variant + '（' + node.id + '）');
    }
    if (node.type === 'badge' && p.tone !== undefined && ['neutral', 'info', 'success', 'warning', 'danger'].indexOf(String(p.tone)) < 0) {
      errs1.push('badge tone 非法: ' + p.tone + '（' + node.id + '）');
    }

    // D2 令牌引用（组匹配 + 无悬空）
    var ownBg = null;
    for (var key in p) {
      var v = p[key];
      if (typeof v !== 'string' || v.charAt(0) !== '$') continue;
      var look = lookupToken(tokens, v);
      if (!look.found) { errs2.push(node.id + '.' + key + ' 引用了不存在的令牌: ' + v); continue; }
      var expect = FIELD_TOKEN_GROUP[key];
      if (expect && expect.indexOf(look.group) < 0) {
        errs2.push(node.id + '.' + key + ' 引用了语义不符的令牌: ' + v + '（该字段期望 ' + expect.join('/') + ' 组）');
      }
      if (key === 'bg') ownBg = tokColor(tokens, v);
    }
    // 颜色字段的字面值也必须合法
    ['bg', 'border', 'color', 'fill'].forEach(function (cf) {
      if (p[cf] === undefined) return;
      var cc = resolveColor(p[cf], tokens);
      if (cc === 'BADREF' || !cc) errs2.push(node.id + '.' + cf + ' 颜色无法解析: ' + p[cf]);
      if (cf === 'bg' && cc && cc !== 'BADREF') ownBg = cc;
    });
    // D2 尺寸字段字面值必须可解析
    ['w', 'h', 'pad', 'gap', 'padX', 'padY', 'radius'].forEach(function (sf) {
      if (p[sf] === undefined) return;
      var rv = resolveSize(p[sf], tokens);
      if (rv === 'BADREF' || (typeof rv === 'number' && rv < 0)) errs2.push(node.id + '.' + sf + ' 尺寸无法解析/为负: ' + p[sf]);
    });

    // D4 间距网格（显式数字）
    for (var gf = 0; gf < GRID_FIELDS.length; gf++) {
      var gk = GRID_FIELDS[gf];
      if (p[gk] === undefined) continue;
      if (typeof p[gk] === 'string' && p[gk].charAt(0) === '$') continue;
      var gv = num(p[gk], NaN);
      if (!isNum(gv)) continue;
      if (gv % GRID_BASE !== 0) errs4.push(node.id + '.' + gk + ' = ' + gv + ' 不在 ' + GRID_BASE + 'px 网格上');
    }

    // D5 字号阶梯（必须引用 token，不接受裸数字）
    if (['text', 'button', 'badge', 'checkbox'].indexOf(node.type) >= 0 && p.size !== undefined) {
      var sv = String(p.size);
      if (sv.charAt(0) !== '$') errs5.push(node.id + '.size = ' + p.size + ' 不是字号令牌（应用 $fontSize.*）');
      else if (lookupToken(tokens, sv).group !== 'fontSize') errs5.push(node.id + '.size 引用非字号令牌: ' + sv);
    }

    // D7 图标白名单 + emoji 禁令
    if (node.type === 'icon') {
      var iname = str(p.name, '');
      if (iname && !ICONS[iname]) errs7.push('图标不在白名单: ' + iname + '（' + node.id + '）—— 禁止 emoji 作图标，请用内置 SVG 图标名');
    }
    ['text', 'label', 'placeholder', 'value', 'alt'].forEach(function (tf) {
      var tv = p[tf];
      if (tv === undefined) return;
      var em = findEmoji(tv);
      if (em) errs7.push(node.id + '.' + tf + ' 含 emoji/符号图标「' + em + '」—— 按技能 emoji-icons 须改用 SVG 图标');
    });

    // D10 交互（props.action = 受控词汇表：link:<目标> / emit:<名字> / toggle）
    // 设计真相源不携带脚本 —— 这里只校验「意图是否合法、目标是否存在」，产物里的绑定由 codegen 生成
    var actV = nodeAction(node);
    if (actV) {
      actionStats.count++;
      if (actV.verb === 'invalid') {
        errs10.push(node.id + ' 的 props.action 不合法：「' + actV.raw + '」—— ' + actV.error);
      } else {
        actionStats[actV.verb]++;
        if (actV.verb === 'toggle' && node.type !== 'checkbox') {
          errs10.push(node.id + '（' + node.type + '）标了 toggle，但 toggle 只对 checkbox 有意义（其它类型请用 emit）');
        }
        if (actV.verb === 'link') {
          if (LINKABLE_TYPES.indexOf(node.type) < 0) {
            errs10.push(node.id + '（' + node.type + '）不能做链接：link 只对 ' + LINKABLE_TYPES.join('/') + ' 有意义');
          }
          if (actV.target.charAt(0) === '#') {
            if (!screenIds[actV.target.slice(1)]) {
              errs10.push(node.id + ' 的链接目标屏幕不存在：' + actV.target + '（工程里的屏幕：' +
                (proj.screens.map(function (x) { return x.id; }).join('、') || '无') + '）');
            } else {
              linkEdges.push({ from: screen.id, to: actV.target.slice(1), nodeId: node.id });
            }
          } else {
            linkExternal++;
          }
        }
      }
    }

    // 继承背景（供 D3）
    var effBg = (ownBg && !ownBg.none && ownBg.a > 0.001) ? ownBg : inheritedBg;

    // D3 对比度
    var fg = null; var bgForCheck = effBg; var fontSize = null; var fontWeight = null; var who = node.id;
    if (node.type === 'text') {
      fg = p.color !== undefined ? tokColor(tokens, p.color) : tokColor(tokens, '$color.fg');
      fontSize = resolveSize(p.size !== undefined ? p.size : '$fontSize.md', tokens);
      fontWeight = resolveSize(p.weight !== undefined ? p.weight : '$fontWeight.normal', tokens);
      textCount++;
    } else if (node.type === 'button') {
      var vo = variantColorObjs(p.variant, tokens);
      fg = vo.fg; bgForCheck = vo.bg || effBg;
      fontSize = resolveSize(p.size !== undefined ? p.size : '$fontSize.md', tokens);
      fontWeight = 600;
      textCount++;
    } else if (node.type === 'badge') {
      var bo = badgeColorObjs(p.tone, tokens);
      fg = bo.fg; bgForCheck = bo.bg || effBg;
      fontSize = resolveSize(p.size !== undefined ? p.size : '$fontSize.xs', tokens);
      fontWeight = 500;
      textCount++;
    } else if (node.type === 'input' || node.type === 'checkbox') {
      fg = node.type === 'input' ? tokColor(tokens, '$color.muted') : tokColor(tokens, '$color.fg');
      bgForCheck = node.type === 'input' ? tokColor(tokens, '$color.bg') : effBg;
      fontSize = 13;
      fontWeight = 400;
      textCount++;
    }
    if (fg && bgForCheck) {
      var ratio = contrastRatio(fg, bgForCheck);
      if (ratio !== null) {
        var fs2 = isNum(fontSize) ? fontSize : 14;
        var large = fs2 >= 24 || (fs2 >= 18.66 && num(fontWeight, 400) >= 700);
        var need = minContrast > 0 ? minContrast : (large ? 3.0 : 4.5);
        if (contrastWorst === null || ratio < contrastWorst) { contrastWorst = ratio; contrastWorstWho = who + '（' + fmtNum(fs2) + 'px）'; }
        if (ratio + 0.005 < need) {
          errs3.push(who + ' 对比度 ' + fmtNum(Math.round(ratio * 100) / 100) + ':1 < ' + need + ':1（' + fmtNum(fs2) + 'px ' +
            (large ? '大字' : '正文') + '，前景 ' + colorCss(fg) + ' / 背景 ' + colorCss(bgForCheck) + '）');
        }
      }
    }

    // D6 触达区
    if (INTERACTIVE_TYPES.indexOf(node.type) >= 0 && box) {
      var minSide = num(screen.width, 360) <= TOUCH_MOBILE_MAX_W ? TOUCH_MIN_MOBILE : TOUCH_MIN_DESKTOP;
      var side = Math.min(box.w, box.h);
      if (touchWorst === null || side < touchWorst) { touchWorst = side; touchWorstWho = node.id + '（' + node.type + ' ' + fmtNum(box.w) + '×' + fmtNum(box.h) + '）'; }
      if (side + EPS < minSide) {
        errs6.push(node.id + ' 触达区 ' + fmtNum(box.w) + '×' + fmtNum(box.h) + '，最短边 ' + fmtNum(side) + 'px < ' + minSide + 'px（屏幕 ' + screen.width + 'px 宽）');
      }
    }

    var kids = node.children || [];
    for (var k = 0; k < kids.length; k++) walk(kids[k], screen, effBg, depth + 1);
  }
  for (var si2 = 0; si2 < proj.screens.length; si2++) {
    if (proj.screens[si2].root) walk(proj.screens[si2].root, proj.screens[si2], tokColor(tokens, proj.screens[si2].background), 0);
  }
  if (totalNodes > LIMIT_NODES) errs1.push('节点总数超过上限 ' + LIMIT_NODES + '：' + totalNodes);

  // D8 布局（溢出 / 越界）
  var errs8 = [];
  for (var li = 0; li < proj.screens.length; li++) {
    var sid2 = proj.screens[li].id;
    var pr = layouts[sid2].problems;
    for (var pi = 0; pi < pr.length; pi++) {
      if (pr[pi].kind === 'overflow' || pr[pi].kind === 'outside') errs8.push(pr[pi].detail);
    }
  }

  // D9 确定性与产物自包含
  var errs9 = [];
  var html1 = '';
  var html2 = '';
  try {
    html1 = renderProjectHtml(proj, tokens);
    html2 = renderProjectHtml(proj, tokens);
  } catch (e) {
    errs9.push('渲染 HTML 失败：' + ((e && e.message) || e));
  }
  if (!errs9.length && html1 !== html2) errs9.push('两次渲染的 HTML 不一致（确定性破坏）');
  if (!errs9.length) {
    var lower = html1.toLowerCase();
    if (lower.indexOf('<script') >= 0) errs9.push('产物含 <script>（不允许）');
    if (/\son[a-z]+\s*=/.test(lower)) errs9.push('产物含内联事件属性（on*，不允许）');
    if (lower.indexOf('javascript:') >= 0) errs9.push('产物含 javascript: URL（不允许）');
    if (lower.indexOf('http://') >= 0 || lower.indexOf('https://') >= 0) errs9.push('产物含外部链接（必须自包含）');
    var css1 = renderTokensCss(tokens);
    if (css1 !== renderTokensCss(tokens)) errs9.push('CSS 变量产物不确定');
  }

  // D11 导航与交互一致（flow 导航图 ↔ 节点 props.action=link:#屏幕）
  //   两者表达同一件事的两面：flow 是信息架构层的「屏幕之间可达」声明，action:link 是「屏幕内哪个元素触发它」。
  //   ① 声明了跳转却没有对应导航边 → 信息架构漏声明（FAIL：架构图与实际行为不符，评审会看不出来）
  //   ② 有导航边但无任何节点声明跳转 → 提示（允许系统/外部触发的跳转存在，不判 FAIL）
  var errs11 = [];
  var flowSet = {};
  for (var fl2 = 0; fl2 < (proj.flow || []).length; fl2++) {
    flowSet[proj.flow[fl2].from + '>' + proj.flow[fl2].to] = true;
  }
  var linkSet = {};
  for (var le2 = 0; le2 < linkEdges.length; le2++) {
    var lk2 = linkEdges[le2].from + '>' + linkEdges[le2].to;
    linkSet[lk2] = true;
    if (!flowSet[lk2]) {
      errs11.push(linkEdges[le2].nodeId + '（屏幕 ' + linkEdges[le2].from + '）声明了 link:#' + linkEdges[le2].to +
        '，但导航图里没有 ' + linkEdges[le2].from + ' → ' + linkEdges[le2].to +
        ' 这条边（用 design_edit op=flow.sync 按交互补齐，或 op=flow.add 手写）');
    }
  }
  for (var fl3 = 0; fl3 < (proj.flow || []).length; fl3++) {
    var fe2 = proj.flow[fl3];
    if (!linkSet[fe2.from + '>' + fe2.to]) flowNoTrigger.push(fe2.from + ' → ' + fe2.to);
  }
  var d11Note = [];
  if (flowNoTrigger.length) {
    d11Note.push('提示（非错误）：以下导航边没有节点声明跳转 —— ' + flowNoTrigger.join('、') +
      '；若为系统/外部触发可忽略，否则请给触发元素加 props.action=link:#目标');
  }

  function mk(id, name, pass, metric, detail) {
    return { id: id, name: name, pass: !!pass, metric: metric || '', detail: (detail || []).slice(0, 8).join('；') };
  }
  checks.push(mk('D1', '结构与必填', errs1.length === 0, errs1.length ? ('问题 ' + errs1.length + ' 处') : (proj.screens.length + ' 屏 / ' + totalNodes + ' 节点 / 最深 ' + maxDepth), errs1));
  checks.push(mk('D2', '设计令牌引用有效', errs2.length === 0, errs2.length ? ('悬空/错用 ' + errs2.length + ' 处') : '全部引用可解析', errs2.concat(warn2)));
  checks.push(mk('D3', '文本对比度（WCAG AA）', errs3.length === 0,
    contrastWorst === null ? '无可检文本' : ('最低 ' + fmtNum(Math.round(contrastWorst * 100) / 100) + ':1 @ ' + contrastWorstWho + '（' + textCount + ' 处）'), errs3));
  checks.push(mk('D4', '间距落在 ' + GRID_BASE + 'px 网格', errs4.length === 0, errs4.length ? ('越网格 ' + errs4.length + ' 处') : ('基准 ' + GRID_BASE + 'px，全部对齐'), errs4));
  checks.push(mk('D5', '字号来自令牌阶梯', errs5.length === 0, errs5.length ? ('裸字号 ' + errs5.length + ' 处') : '全部走 $fontSize.*', errs5));
  checks.push(mk('D6', '触达区尺寸', errs6.length === 0,
    touchWorst === null ? '无可交互元素' : ('最短边 ' + fmtNum(touchWorst) + 'px @ ' + touchWorstWho), errs6));
  checks.push(mk('D7', '图标白名单 / 无 emoji', errs7.length === 0, errs7.length ? ('违规 ' + errs7.length + ' 处') : ('白名单 ' + Object.keys(ICONS).length + ' 个图标可用'), errs7));
  checks.push(mk('D8', '布局无溢出/越界', errs8.length === 0, errs8.length ? ('问题 ' + errs8.length + ' 处') : '与预览渲染同源检查通过', errs8));
  checks.push(mk('D9', '产物确定且自包含', errs9.length === 0, errs9.length ? ('问题 ' + errs9.length + ' 处') : ('HTML ' + html1.length + ' 字符，两次渲染位级一致、零外链'), errs9));
  checks.push(mk('D10', '交互声明合法（props.action）', errs10.length === 0,
    actionStats.count
      ? (actionStats.count + ' 处交互：link ' + actionStats.link + ' / emit ' + actionStats.emit + ' / toggle ' + actionStats.toggle)
      : '无交互声明（纯静态展示）', errs10));
  checks.push(mk('D11', '导航与交互一致（flow ↔ action:link）', errs11.length === 0,
    (linkEdges.length
      ? ('跳屏幕 ' + linkEdges.length + ' 处，全部有导航边')
      : '无跳屏幕交互') +
      (linkExternal ? ('；外链/站内链接 ' + linkExternal + ' 处（不入导航图）') : '') +
      (flowNoTrigger.length ? ('；' + flowNoTrigger.length + ' 条导航边无节点触发（' + flowNoTrigger.slice(0, 3).join('、') + (flowNoTrigger.length > 3 ? '…' : '') + '）') : ''),
    errs11.concat(d11Note)));

  var passAll = true;
  for (var ci = 0; ci < checks.length; ci++) if (!checks[ci].pass) passAll = false;
  return {
    checks: checks,
    pass: passAll,
    stats: { screens: proj.screens.length, nodes: totalNodes, textNodes: textCount, maxDepth: maxDepth },
    html: html1,
  };
}

// ── 文件读写 ──────────────────────────────────────────────
function loadJson(ctx, path, what) {
  if (!ctx.fs.exists(path)) throw new Error(what + '不存在: ' + path);
  var txt = '';
  try { txt = ctx.fs.readFile(path); } catch (e) { throw new Error('读取' + what + '失败: ' + ((e && e.message) || e)); }
  var obj = null;
  try { obj = JSON.parse(txt); } catch (e) { throw new Error(what + ' JSON 解析失败（' + path + '）: ' + ((e && e.message) || e)); }
  return obj;
}
function loadProject(ctx, path) { return normalizeProject(loadJson(ctx, path, '设计工程')); }
function saveProject(ctx, path, proj) {
  proj.meta = proj.meta || {};
  proj.meta.updatedAt = nowIso();
  ctx.fs.writeFile(path, JSON.stringify(proj, null, 2) + '\n');
}
function loadTokensFile(ctx, path) { return normalizeTokens(loadJson(ctx, path, '设计令牌')); }
function saveTokens(ctx, path, tokens) {
  tokens.meta = tokens.meta || {};
  tokens.meta.updatedAt = nowIso();
  ctx.fs.writeFile(path, JSON.stringify(tokens, null, 2) + '\n');
}
function loadTokensFor(ctx, proj) {
  var p = str(proj.tokens, 'design.tokens.json');
  return loadTokensFile(ctx, p);
}

// 令牌类型的合法性（颜色必须可解析 / 数值组必须数字 / 文本组必须字符串）
function validateTokenValue(group, name, value) {
  if (group === 'color') {
    var c = parseColor(value);
    if (!c) throw new Error('令牌 ' + group + '.' + name + ' 不是合法颜色: ' + value);
    if (c.none) throw new Error('令牌 ' + group + '.' + name + ' 不能是 transparent/none（颜色令牌需为实色）');
  } else if (group === 'space' || group === 'radius' || group === 'fontSize' || group === 'fontWeight' || group === 'lineHeight') {
    var n = typeof value === 'number' ? value : parseFloat(value);
    if (!isNum(n)) throw new Error('令牌 ' + group + '.' + name + ' 必须是数字: ' + value);
    if (n < 0) throw new Error('令牌 ' + group + '.' + name + ' 不能为负: ' + value);
    if (group === 'space' && n % GRID_BASE !== 0) throw new Error('令牌 ' + group + '.' + name + ' 应为 ' + GRID_BASE + 'px 倍数（间距网格）: ' + value);
  } else {
    if (typeof value !== 'string' || !value) throw new Error('令牌 ' + group + '.' + name + ' 必须是非空字符串');
  }
}

// 颜色对比度矩阵（设计令牌自检；用于 mode=show）
function pad(s, n) {
  var t = String(s);
  while (t.length < n) t += ' ';
  return t;
}
function contrastMatrix(tokens) {
  var rows = [];
  var bgs = [];
  var colorNames = Object.keys(tokens.color || {});
  ['bg', 'surface'].forEach(function (b) {
    if (colorNames.indexOf(b) >= 0) bgs.push(b);
  });
  if (!bgs.length) return '（无 bg/surface 令牌，跳过对比度矩阵）';
  var header = '  前景 \\ 背景      ' + bgs.map(function (b) { return pad(b, 12); }).join('');
  rows.push(header);
  for (var i = 0; i < colorNames.length; i++) {
    var nm = colorNames[i];
    var fg = tokColor(tokens, '$color.' + nm);
    var cells = [];
    for (var b = 0; b < bgs.length; b++) {
      var bg = tokColor(tokens, '$color.' + bgs[b]);
      var r = contrastRatio(fg, bg);
      var txt = r === null ? '-' : (fmtNum(Math.round(r * 100) / 100) + ':1');
      if (r !== null && r + 0.005 < 4.5) txt += '!';
      cells.push(pad(txt, 12));
    }
    rows.push('  ' + pad(nm, 16) + cells.join(''));
  }
  rows.push('  （! = 低于 4.5:1 正文门槛；仅表示"当作正文色不够"，作大标题/装饰可用）');
  return rows.join('\n');
}

function verifyText(res, path, reportPath) {
  var lines = [];
  lines.push('设计校验：' + (res.pass ? '通过 ✓' : '未通过 ✗') + '（' + res.stats.screens + ' 屏 / ' + res.stats.nodes + ' 节点 / 文本 ' + res.stats.textNodes + ' 处）');
  for (var i = 0; i < res.checks.length; i++) {
    var c = res.checks[i];
    lines.push('  ' + (c.pass ? '✓' : '✗') + ' ' + pad(c.id, 4) + pad(c.name, 22) + c.metric);
  }
  var bad = res.checks.filter(function (c) { return !c.pass; });
  if (bad.length) {
    lines.push('');
    lines.push('不利项（前 12 条）：');
    var shown = 0;
    for (var b = 0; b < bad.length && shown < 12; b++) {
      var parts = bad[b].detail ? String(bad[b].detail).split('；') : [];
      for (var k = 0; k < parts.length && shown < 12; k++) {
        lines.push('  · [' + bad[b].id + '] ' + parts[k]);
        shown++;
      }
    }
  }
  lines.push('');
  lines.push('报告：' + reportPath + '（旁挂，不改工程本体）｜工程：' + path);
  if (res.pass) lines.push('下一步：design_export format=html 出预览（可截图核对）→ 视觉确认后再交付。');
  return lines.join('\n');
}

// ── 工具实现 ──────────────────────────────────────────────
function designTokens(args, exec, ctx) {
  var path = argStr(args, 'path', 'design.tokens.json');
  var mode = argStr(args, 'mode', 'show');
  if (mode === 'create') {
    if (ctx.fs.exists(path) && !argBool(args, 'overwrite', false)) {
      return '设计令牌已存在: ' + path + '（要重建请 overwrite=true；改单个令牌请 mode=update 或 design_edit 的 token.set）';
    }
    var t = defaultTokens();
    t.meta = { title: argStr(args, 'title', '设计令牌'), updatedAt: nowIso() };
    saveTokens(ctx, path, t);
    return '✓ 已创建设计令牌: ' + path + '\n\n' + tokenSummary(t) + '\n\n对比度矩阵（WCAG，越低越差）：\n' + contrastMatrix(t);
  }
  if (mode === 'show') {
    var t1 = loadTokensFile(ctx, path);
    return tokenSummary(t1) + '\n\n对比度矩阵（WCAG，越低越差）：\n' + contrastMatrix(t1) + '\n\n文件：' + path;
  }
  if (mode === 'update') {
    var t2 = loadTokensFile(ctx, path);
    var patch = args.tokens;
    if (!patch || typeof patch !== 'object') throw new Error('mode=update 需要 tokens 参数：{"color":{"accent":"#1D4ED8"}}');
    var log = [];
    var groups = Object.keys(patch);
    for (var gi = 0; gi < groups.length; gi++) {
      var g = groups[gi];
      if (TOKEN_GROUPS.indexOf(g) < 0) throw new Error('未知令牌组: ' + g + '（可用：' + TOKEN_GROUPS.join('/') + '）');
      var items = patch[g] || {};
      var names = Object.keys(items);
      for (var ni = 0; ni < names.length; ni++) {
        var nm = names[ni];
        var val = items[nm];
        if (val === null) {
          if (!(nm in (t2[g] || {}))) throw new Error('令牌不存在: ' + g + '.' + nm);
          delete t2[g][nm];
          log.push('删除 ' + g + '.' + nm);
          continue;
        }
        validateTokenValue(g, nm, val);
        var isNew = !(nm in (t2[g] || {}));
        t2[g] = t2[g] || {};
        t2[g][nm] = val;
        log.push((isNew ? '新增 ' : '覆盖 ') + g + '.' + nm + ' = ' + val);
      }
    }
    if (!log.length) throw new Error('tokens 参数里没有任何要改的令牌');
    saveTokens(ctx, path, t2);
    return '✓ 已更新设计令牌: ' + path + '\n\n' + log.join('\n') + '\n\n对比度矩阵：\n' + contrastMatrix(t2);
  }
  throw new Error('未知 mode: ' + mode + '（可用 create / show / update）');
}

function designProject(args, exec, ctx) {
  var path = argStr(args, 'path', 'design.project.json');
  var mode = argStr(args, 'mode', 'show');
  if (mode === 'create') {
    if (ctx.fs.exists(path) && !argBool(args, 'overwrite', false)) {
      return '设计工程已存在: ' + path + '（要重建请 overwrite=true；改内容请用 design_edit）';
    }
    var tokensPath = argStr(args, 'tokens', 'design.tokens.json');
    var proj = emptyProject(argStr(args, 'title', undefined), tokensPath);
    if (!ctx.fs.exists(tokensPath)) {
      var dt = defaultTokens();
      dt.meta = { title: '设计令牌', updatedAt: nowIso() };
      saveTokens(ctx, tokensPath, dt);
    }
    var screens = args.screens;
    if (screens && screens.length) {
      for (var i = 0; i < screens.length; i++) {
        var sd = screens[i] || {};
        var sid = str(sd.id, 'screen' + (i + 1));
        var tpl = str(sd.template, 'none');
        var sc = { id: sid, name: str(sd.name, sid), width: num(sd.width, 360), height: num(sd.height, 640), background: str(sd.background, '$color.bg') };
        if (tpl && tpl !== 'none') {
          if (!TEMPLATES[tpl]) throw new Error('未知模板: ' + tpl + '（可用：' + Object.keys(TEMPLATES).join('/') + '/none）');
          sc.root = TEMPLATES[tpl]();
          prefixIds(sc.root, sid);
        }
        proj.screens.push(sc);
      }
    }
    saveProject(ctx, path, proj);
    return '✓ 已创建设计工程: ' + path + '\n\n' + projectSummary(proj, path);
  }
  if (mode === 'show') {
    return projectSummary(loadProject(ctx, path), path);
  }
  if (mode === 'update') {
    var p2 = loadProject(ctx, path);
    var logs = [];
    if (args.title !== undefined) { p2.meta.title = String(args.title); logs.push('标题 → ' + p2.meta.title); }
    if (args.tokens !== undefined) { p2.tokens = String(args.tokens); logs.push('令牌文件 → ' + p2.tokens); }
    if (args.html !== undefined) { p2.artifacts = p2.artifacts || {}; p2.artifacts.html = String(args.html); logs.push('HTML 产物 → ' + p2.artifacts.html); }
    if (!logs.length) throw new Error('mode=update 未提供任何字段（title / tokens / html）');
    saveProject(ctx, path, p2);
    return '✓ 已更新设计工程: ' + path + '\n\n' + logs.join('\n') + '\n\n' + projectSummary(p2, path);
  }
  throw new Error('未知 mode: ' + mode + '（可用 create / show / update）');
}

function designEdit(args, exec, ctx) {
  var path = argStr(args, 'path', 'design.project.json');
  var proj = loadProject(ctx, path);
  var tokens = loadTokensFor(ctx, proj);
  var ops = args.ops && args.ops.length ? args.ops.slice() : [];
  if (args.op) {
    var single = {};
    single.op = args.op;
    for (var k in args) { if (k !== 'ops' && k !== 'op' && k !== 'path') single[k] = args[k]; }
    ops.push(single);
  }
  if (!ops.length) throw new Error('未提供任何编辑命令（ops 数组或 op + 参数）');
  var before = JSON.stringify(tokens);
  var log = applyOps(proj, tokens, ops);
  var tokensChanged = JSON.stringify(tokens) !== before;
  if (tokensChanged) saveTokens(ctx, str(proj.tokens, 'design.tokens.json'), tokens);
  saveProject(ctx, path, proj);
  return '✓ 已应用 ' + ops.length + ' 条命令\n\n' + log.join('\n') + '\n\n屏幕 ' + proj.screens.length +
    ' 个｜节点 ' + countNodes(proj) + ' 个' + (tokensChanged ? '｜令牌已更新' : '') +
    '\n下一步：design_verify 校验 → design_export format=html 出预览。';
}

function designExport(args, exec, ctx) {
  var path = argStr(args, 'path', 'design.project.json');
  var format = String(argStr(args, 'format', 'html')).toLowerCase();
  var proj = loadProject(ctx, path);
  var tokens = loadTokensFor(ctx, proj);
  var onlyScreen = argStr(args, 'screen', '');
  var overwrite = argBool(args, 'overwrite', false);
  proj.artifacts = proj.artifacts || {};
  if (format === 'html') {
    var target = proj;
    if (onlyScreen) {
      var foundScreen = screenById(proj, onlyScreen);
      if (!foundScreen) throw new Error('屏幕不存在: ' + onlyScreen);
      target = deepClone(proj);
      target.screens = target.screens.filter(function (s) { return s.id === onlyScreen; });
    }
    var html = renderProjectHtml(target, tokens);
    var out = argStr(args, 'out', str(proj.artifacts.html, 'design.html'));
    if (ctx.fs.exists(out) && !overwrite) return 'HTML 已存在: ' + out + '（覆盖请 overwrite=true）';
    ctx.fs.writeFile(out, html);
    return '✓ 已导出 HTML 预览: ' + out + '（' + html.length + ' 字符，' + target.screens.length + ' 屏，自包含：无外链、无脚本）\n' +
      '布局由本插件布局引擎计算并绝对定位（与 design_verify 的 D8 同源）→ 截图即所见。\n' +
      '建议：再用 design_verify 校验，并截图核对渲染。';
  }
  if (format === 'css') {
    var css = renderTokensCss(tokens);
    var outCss = argStr(args, 'out', str(proj.artifacts.css, 'design.tokens.css'));
    if (ctx.fs.exists(outCss) && !overwrite) return 'CSS 已存在: ' + outCss + '（覆盖请 overwrite=true）';
    ctx.fs.writeFile(outCss, css);
    var cnt = 0;
    for (var i = 0; i < TOKEN_GROUPS.length; i++) cnt += Object.keys(tokens[TOKEN_GROUPS[i]] || {}).length;
    return '✓ 已导出 CSS 变量: ' + outCss + '（' + cnt + ' 个自定义属性，' + css.length + ' 字符）';
  }
  if (format === 'mermaid') {
    var mm = renderMermaid(proj, tokens);
    var outMm = argStr(args, 'out', str(proj.artifacts.flow, 'design.flow.mmd'));
    if (ctx.fs.exists(outMm) && !overwrite) return 'mermaid 已存在: ' + outMm + '（覆盖请 overwrite=true）';
    ctx.fs.writeFile(outMm, mm);
    return '✓ 已导出 mermaid 流程图: ' + outMm + '（' + proj.screens.length + ' 个 subgraph / ' + (proj.flow || []).length + ' 条导航边）\n' +
      '本插件只生成 mermaid 文本（零依赖），渲染交给宿主 Markdown / mermaid 渲染器。';
  }
  throw new Error('未知 format: ' + format + '（可用 html / css / mermaid）');
}

// ═══════════════════════════════════════════════════════════════
// 前端代码生成（design_codegen）—— 把设计**落地成工程项目代码**
//
// 与 design_export(format=html) 的本质区别：
//   · 预览 = 绝对定位盒子（left/top/width/height 由布局引擎算出，截图即所见，供人核对）；
//   · 代码 = **流式布局**（同一套 flex 规则，用 CSS 表达）：容器 → display:flex +
//     flex-direction + gap + padding；尺寸默认由内容与父容器决定（不写死像素）；
//     语义标签（p / button / input / label / hr / svg）、语义 class（.d-<节点 id>）；
//     颜色/间距/圆角/字号/字重/行高/阴影一律走**令牌 CSS 变量**（var(--color-fg)），
//     不散落裸值 —— 改 design.tokens.json 重新生成即全站同步（D2 判据精神的落地）。
//
// 产物（targets 可多选，全部为文本、确定性）：
//   html  → <targetDir>/html/<屏幕 id>.html + <屏幕 id>.css（零依赖静态页，link 引用 tokens.css）
//   vue   → <targetDir>/vue/<屏幕 id PascalCase>.vue（SFC：template + style scoped）
//   react → <targetDir>/react/<屏幕 id PascalCase>.jsx + <...>.module.css
//   公共  → <targetDir>/tokens.css（令牌 → CSS 变量）+ <targetDir>/README.md（集成说明）
// 幂等：同工程两次生成位级一致（无时间戳/无随机/无绝对路径）；写盘前把已存在文件
//   备份为 <文件>.bak（可人工回滚）。
// ═══════════════════════════════════════════════════════════════

// refToVar：令牌引用 → CSS 变量（'$color.bg' → 'var(--color-bg)'；非引用 → null）
function refToVar(ref) {
  if (ref === undefined || ref === null) return null;
  var s = String(ref);
  if (s.charAt(0) !== '$') return null;
  var body = s.slice(1);
  var dot = body.indexOf('.');
  if (dot <= 0) return null;
  var grp = body.slice(0, dot);
  var name = body.slice(dot + 1);
  if (TOKEN_GROUPS.indexOf(grp) < 0) return null;
  var gvar = grp.replace(/[A-Z]/g, function (m) { return '-' + m.toLowerCase(); });
  return 'var(--' + gvar + '-' + name + ')';
}

// tokVarCss：令牌引用 → var(...)，但**引用必须真实存在**（悬空即返回 null：
//   宁可少一条声明，也不产出 var(--color-typo) 这种永远无效的变量；D2 判据会抓悬空引用）。
function tokVarCss(ref, tokens) {
  var rv = refToVar(ref);
  if (!rv) return null;
  return lookupToken(tokens, ref).found ? rv : null;
}

// cssSafe：裸值兜底清理（防把 `;`/`}` 带进声明造成规则逃逸）
function cssSafe(v) {
  return String(v).replace(/[;{}<>]/g, '').replace(/\/\*/g, '').trim();
}

// 尺寸类 → CSS 值（令牌 → var()；数字 → px；fill → 100%；auto / 空 → null）
function sizePropCss(v, tokens) {
  if (v === undefined || v === null) return null;
  var s = String(v);
  if (s === 'auto') return null;
  if (s === 'fill') return '100%';
  var rv = tokVarCss(s, tokens);
  if (rv) return rv;
  var n = num(s, NaN);
  return isNum(n) ? fmtNum(n) + 'px' : null;
}

// 间距类（pad/gap/padX/padY）→ CSS 值（0 省略；数字要求落在 4px 网格，否则按原值输出并提示）
function spacePropCss(v, tokens) {
  if (v === undefined || v === null) return null;
  var s = String(v);
  var tok = lookupToken(tokens, s);
  if (String(s).charAt(0) === '$') {
    if (!tok.found) return null;
    var tv = num(tok.value, NaN);
    if (!isNum(tv) || tv === 0) return null;
    return refToVar(s);
  }
  var n = num(s, NaN);
  if (!isNum(n) || n === 0) return null;
  return fmtNum(n) + 'px';
}

// 颜色类 → CSS 值（令牌 → var()；裸色值 → 规范化 hex；none → transparent）
function colorPropCss(v, tokens) {
  if (v === undefined || v === null) return null;
  var s = String(v);
  if (s === 'none' || s === 'transparent') return 'transparent';
  var rv = tokVarCss(s, tokens);
  if (rv) return rv;
  var c = parseColor(s);
  return c ? colorCss(c) : null;
}

// 阴影 → CSS 值
function shadowPropCss(v, tokens) {
  if (v === undefined || v === null) return null;
  var s = String(v);
  if (s === 'none') return 'none';
  var rv = tokVarCss(s, tokens);
  if (rv) return rv;
  return cssSafe(s);
}

// 字号/字重/行高 → CSS 值（默认值对齐 measureLeaf：$fontSize.md / normal / $lineHeight.normal）
function fontPropCss(v, tokens, defRef) {
  var s = (v === undefined || v === null) ? defRef : v;
  if (s === undefined || s === null) return null;
  return tokVarCss(s, tokens) || sizePropCss(s, tokens);
}

// 节点 → 语义 class（三级优先，尽量让落地代码可读、可维护）：
//   ① props.name 显式命名（最优）        → .d-<name>
//   ② 语义化 id（非模板自动编号）        → .d-<id>（如 hero-title）
//   ③ 模板自动编号 id（home-panel_n3）  → .d-<类型>-<序号>（.d-text-3，比 n3 可读）
//   冲突（同名/同 id）追加 -2/-3（DFS 顺序固定 → 结果确定）。
//   ③ 出现时会在返回里提示「建议给节点起语义 id」。
function codeClass(node, out) {
  var p = node.props || {};
  // ★ icon 的 props.name 是**图标名**（ICONS 白名单键，如 home/success），不是节点名 ——
  //   拿它当类名会得到 .d-home / .d-success（跨节点撞名 + 丢掉节点语义），必须排除。
  var nm = (node.type === 'icon') ? '' : str(p.name, '');
  var base;
  if (nm && /^[A-Za-z][A-Za-z0-9_-]*$/.test(nm)) {
    base = nm;
  } else {
    var id = String(node.id);
    var m = /^(.*)_n(\d+)$/.exec(id);
    if (m) {
      base = String(node.type) + '-' + m[2];
      if (out) {
        if (!out.autoIdIds) out.autoIdIds = {};
        out.autoIdIds[id] = true;
      }
    } else {
      base = id.replace(/[^A-Za-z0-9_-]/g, '-');
    }
  }
  if (!/^[A-Za-z_]/.test(base)) base = 'n-' + base;
  var cls = 'd-' + base;
  if (out) {
    if (!out.used) out.used = {};
    if (out.used[cls]) {
      var n = 2;
      while (out.used[cls + '-' + n]) n++;
      cls = cls + '-' + n;
    }
    out.used[cls] = true;
  }
  return cls;
}
function screenClass(screen) {
  var id = String(screen.id).replace(/[^A-Za-z0-9_-]/g, '-');
  if (!/^[A-Za-z_]/.test(id)) id = 's-' + id;
  return 'screen-' + id;
}
// 屏幕 id → 组件名（PascalCase；React/Vue 组件标识符）
function pascalCase(s) {
  var parts = String(s).split(/[^A-Za-z0-9]+/);
  var out = '';
  for (var i = 0; i < parts.length; i++) {
    var p = parts[i];
    if (!p) continue;
    out += p.charAt(0).toUpperCase() + p.slice(1);
  }
  if (!out) out = 'Screen';
  if (/^[0-9]/.test(out)) out = 'S' + out;
  return out;
}

// 按钮 variant / 徽章 tone → 令牌变量（与校验器 variantColorObjs/badgeColorObjs 同一套语义）
function variantVars(variant) {
  var k = str(variant, 'primary');
  if (k === 'secondary') return { bg: 'var(--color-surface)', fg: 'var(--color-fg)', border: 'var(--color-border)' };
  if (k === 'ghost') return { bg: 'transparent', fg: 'var(--color-accent)', border: 'transparent' };
  if (k === 'danger') return { bg: 'var(--color-danger)', fg: 'var(--color-accent-fg)', border: 'var(--color-danger)' };
  return { bg: 'var(--color-accent)', fg: 'var(--color-accent-fg)', border: 'var(--color-accent)' };
}
function toneVars(tone) {
  var k = str(tone, 'neutral');
  if (k === 'info') return { bg: 'var(--color-accent-soft)', fg: 'var(--color-accent)' };
  if (k === 'success') return { bg: 'var(--color-surface-2)', fg: 'var(--color-success)' };
  if (k === 'warning') return { bg: 'var(--color-surface-2)', fg: 'var(--color-warning)' };
  if (k === 'danger') return { bg: 'var(--color-surface-2)', fg: 'var(--color-danger)' };
  return { bg: 'var(--color-surface-2)', fg: 'var(--color-fg)' };
}
function nodeIsContainer(node) { return CONTAINER_TYPES.indexOf(node.type) >= 0; }

// buildCodeTree：设计节点 → 代码结构树
//   { tag, cls, attrs: [[名,值|true]], text, svg:{path,size}, selfClose, children,
//     decls: [...], extra: [{ sel, decls: [...] }] }
// 规则与布局引擎（layoutNode）严格对齐：容器 flex + 默认 gap $space.sm、
//   col/frame/card/list 撑满交叉轴、row 未指定宽按内容宽、叶子尺寸由内容决定。
function buildCodeTree(node, tokens, out, opts) {
  var p = node.props || {};
  var t = node.type;
  var isC = nodeIsContainer(node);
  var cls = codeClass(node, out);
  var decls = [];
  var extra = [];
  var attrs = [];
  var text = '';
  var svg = null;
  var selfClose = false;
  var tag = 'div';
  // 组件参数化 / 插槽：只有**组件产物**构建时传 opts.comp（屏幕侧是使用方，保持字面量）
  //   ★ 两件事互相独立：插槽由 props.slot 标记决定，参数化由 opts.comp.params 决定
  var isSlot = !!(opts && opts.comp && isSlotHost(node));
  var textProp = '';
  // 交互：props.action（受控词汇表，见 parseAction）→ 绑定按目标形态生成（html/vue 进 attrs，react 走 JSX）
  var act = nodeAction(node);
  var actTarget = (opts && opts.target) ? opts.target : 'html';
  if (act && act.verb === 'invalid') {
    out.warnings.push('节点 ' + node.id + ' 的 props.action 不合法（' + act.raw + '）：' + act.error + ' —— 本次未生成交互绑定');
    act = null;
  }
  if (act && act.verb === 'toggle' && t !== 'checkbox') {
    out.warnings.push('节点 ' + node.id + ' 标了 toggle，但类型是 ' + t + '（toggle 只对 checkbox 有意义）→ 未生成交互');
    act = null;
  }
  if (opts && opts.comp && opts.comp.params && !isSlot) {
    var cf = contentField(node);
    if (cf) {
      var pn = propNameFor(node, opts.comp.used);
      if (pn) {
        textProp = pn;
        opts.comp.list.push({ name: pn, field: cf, def: str(p[cf], '') });
      }
    }
  }

  // ── 通用：背景 / 边框 / 圆角 / 阴影 ──
  var bgc = colorPropCss(p.bg, tokens);
  if (bgc) decls.push('background: ' + bgc);
  if (p.border !== undefined && String(p.border) !== 'none') {
    var bd = colorPropCss(p.border, tokens);
    if (bd) decls.push('border: 1px solid ' + bd);
  }
  if (p.radius !== undefined && String(p.radius) !== 'none') {
    var rc = tokVarCss(p.radius, tokens) || (isNum(num(String(p.radius), NaN)) ? fmtNum(num(String(p.radius), 0)) + 'px' : null);
    if (rc) decls.push('border-radius: ' + rc);
  }
  var shd = shadowPropCss(p.shadow, tokens);
  if (shd) decls.push('box-shadow: ' + shd);

  var wCss = sizePropCss(p.w, tokens);
  var hCss = sizePropCss(p.h, tokens);

  if (isC) {
    // ── 容器 ──
    decls.push('display: flex');
    decls.push('flex-direction: ' + (t === 'row' ? 'row' : 'column'));
    var gapRaw = p.gap !== undefined ? p.gap : '$space.sm'; // 与 layoutNode 默认一致
    var gapCss = spacePropCss(gapRaw, tokens);
    if (gapCss) decls.push('gap: ' + gapCss);
    var padRaw = p.pad !== undefined ? p.pad : nodeDefaultPad(node);
    var padCss = spacePropCss(padRaw, tokens);
    if (padCss) decls.push('padding: ' + padCss);
    if (wCss) decls.push('width: ' + wCss);
    if (hCss === '100%') { decls.push('flex: 1 1 auto'); decls.push('min-height: 0'); }
    else if (hCss) decls.push('height: ' + hCss);
    // row 未显式指定宽 → 与布局引擎一致：按内容宽 + 左对齐（避免被 col 父容器拉伸）
    if (t === 'row' && !wCss) { decls.push('width: fit-content'); decls.push('align-self: flex-start'); }
  } else if (t === 'text') {
    tag = 'p';
    decls.push('margin: 0');
    var fsz = fontPropCss(p.size, tokens, '$fontSize.md');
    if (fsz) decls.push('font-size: ' + fsz);
    var fwt = fontPropCss(p.weight, tokens, '$fontWeight.normal');
    if (fwt) decls.push('font-weight: ' + fwt);
    var flh = fontPropCss(p.leading, tokens, '$lineHeight.normal');
    if (flh) decls.push('line-height: ' + flh);
    decls.push('color: ' + (colorPropCss(p.color, tokens) || 'var(--color-fg)'));
    var al = str(p.align, 'left');
    if (al === 'center' || al === 'right') decls.push('text-align: ' + al);
    decls.push('overflow-wrap: break-word');
    if (num(p.maxLines, 0) > 0) {
      decls.push('display: -webkit-box');
      decls.push('-webkit-box-orient: vertical');
      decls.push('-webkit-line-clamp: ' + fmtNum(num(p.maxLines, 1)));
      decls.push('overflow: hidden');
    }
    if (wCss) decls.push('width: ' + wCss);
    text = str(p.text, '');
  } else if (t === 'button') {
    tag = 'button';
    var vv = variantVars(p.variant);
    var bPadX = spacePropCss(p.padX !== undefined ? p.padX : '$space.lg', tokens) || '16px';
    var bPadY = spacePropCss(p.padY !== undefined ? p.padY : '$space.sm', tokens) || '8px';
    decls.push('display: inline-flex');
    decls.push('align-items: center');
    decls.push('justify-content: center');
    decls.push('padding: ' + bPadY + ' ' + bPadX);
    decls.push('font-size: ' + (fontPropCss(p.size, tokens, '$fontSize.md') || '14px'));
    decls.push('font-weight: 600');
    decls.push('line-height: 1.5');
    decls.push('background: ' + vv.bg);
    decls.push('color: ' + vv.fg);
    decls.push('border: 1px solid ' + vv.border);
    decls.push('border-radius: ' + (tokVarCss(p.radius, tokens) || 'var(--radius-md)'));
    decls.push('cursor: pointer');
    if (argBoolish(p.disabled)) { decls.push('opacity: 0.5'); attrs.push(['disabled', true]); }
    if (p.full === true || String(p.w) === 'fill') decls.push('width: 100%');
    else if (wCss) decls.push('width: ' + wCss);
    if (hCss && hCss !== '100%') decls.push('height: ' + hCss);
    text = str(p.label, '');
  } else if (t === 'input') {
    tag = 'input';
    selfClose = true;
    var iH = hCss || '36px';
    decls.push('height: ' + iH);
    decls.push('padding: 0 ' + (spacePropCss(p.padX, tokens) || '10px'));
    decls.push('font-size: var(--font-size-md)');
    decls.push('color: var(--color-fg)');
    decls.push('background: var(--color-bg)');
    decls.push('border: 1px solid var(--color-border)');
    decls.push('border-radius: var(--radius-md)');
    decls.push('width: ' + (wCss || '100%'));
    attrs.push(['type', 'text']);
    var ph = str(p.placeholder, '') || str(p.label, '');
    if (ph) attrs.push(['placeholder', ph]);
    if (str(p.value, '')) attrs.push(['value', str(p.value, '')]);
    if (argBoolish(p.disabled)) { decls.push('opacity: 0.5'); attrs.push(['disabled', true]); }
  } else if (t === 'checkbox') {
    tag = 'label';
    decls.push('display: inline-flex');
    decls.push('align-items: center');
    decls.push('gap: var(--space-sm)');
    decls.push('font-size: var(--font-size-md)');
    decls.push('color: var(--color-fg)');
    decls.push('cursor: pointer');
    extra.push({ sel: '.' + cls + ' input', decls: ['width: 14px', 'height: 14px', 'margin: 0', 'accent-color: var(--color-accent)'] });
    attrs.push(['data-checked', p.checked === true ? 'true' : 'false']);
    text = str(p.label, '');
  } else if (t === 'badge') {
    tag = 'span';
    var tv = toneVars(p.tone);
    decls.push('display: inline-flex');
    decls.push('align-items: center');
    decls.push('padding: ' + (spacePropCss(p.padY !== undefined ? p.padY : '$space.xs', tokens) || '4px') +
      ' ' + (spacePropCss(p.padX !== undefined ? p.padX : '$space.sm', tokens) || '8px'));
    decls.push('font-size: ' + (fontPropCss(p.size, tokens, '$fontSize.xs') || '11px'));
    decls.push('font-weight: 500');
    decls.push('line-height: 1.4');
    decls.push('white-space: nowrap');
    decls.push('background: ' + tv.bg);
    decls.push('color: ' + tv.fg);
    decls.push('border-radius: var(--radius-full)');
    text = str(p.label, '');
  } else if (t === 'icon') {
    tag = 'span';
    var isz = num(p.size !== undefined ? p.size : 20, 20);
    if (String(p.size).charAt(0) === '$') {
      var sTok = lookupToken(tokens, p.size);
      isz = sTok.found ? num(sTok.value, 20) : 20;
    }
    decls.push('display: inline-flex');
    decls.push('color: ' + (colorPropCss(p.color, tokens) || 'var(--color-fg)'));
    decls.push('flex: none');
    var icName = str(p.name, '');
    if (ICONS[icName]) {
      svg = { path: ICONS[icName], size: isz, name: icName };
      attrs.push(['role', 'img']);
      attrs.push(['aria-label', icName]);
    } else {
      // 未在白名单（D7 会判失败）：代码里留一个可替换的占位，不臆造图形
      decls.push('width: ' + fmtNum(isz) + 'px');
      decls.push('height: ' + fmtNum(isz) + 'px');
      decls.push('border: 1px dashed var(--color-border)');
      decls.push('border-radius: var(--radius-sm)');
      if (icName) out.warnings.push('图标不在白名单（D7）：' + icName + '（节点 ' + node.id + '）已生成占位方块');
    }
  } else if (t === 'image') {
    tag = 'div';
    decls.push('width: ' + (wCss || '100%'));
    var ratio = num(p.ratio, 0);
    if (ratio > 0) decls.push('aspect-ratio: ' + cssSafe(String(ratio)));
    else decls.push('height: ' + (hCss || '120px'));
    decls.push('background: var(--color-surface-2)');
    decls.push('border: 1px solid var(--color-border)');
    decls.push('border-radius: ' + (tokVarCss(p.radius, tokens) || 'var(--radius-md)'));
    attrs.push(['role', 'img']);
    attrs.push(['aria-label', str(p.alt, '图片')]);
  } else if (t === 'divider') {
    tag = 'hr';
    selfClose = true;
    decls.push('border: none');
    decls.push('height: ' + (sizePropCss(p.h, tokens) || '1px'));
    decls.push('background: ' + (colorPropCss(p.color, tokens) || 'var(--color-border)'));
    decls.push('margin: 0');
    decls.push('width: 100%');
  } else if (t === 'spacer') {
    tag = 'div';
    decls.push('height: ' + (sizePropCss(p.h, tokens) || 'var(--space-md)'));
    if (hCss === '100%') { decls.pop(); decls.push('flex: 1 1 auto'); }
    attrs.push(['aria-hidden', 'true']);
  }

  // ── 交互绑定（link → 真链接；emit → 按目标形态出绑定）──
  if (act && act.verb === 'link') {
    if (LINKABLE_TYPES.indexOf(t) < 0) {
      out.warnings.push('节点 ' + node.id + '（' + t + '）不适合做链接（props.action=' + act.raw + '）→ 未生成链接；' +
        '请把 action 放在按钮/文本/卡片等可点元素上');
      act = null;
    } else {
      if (tag !== 'a') { decls.push('text-decoration: none'); decls.push('cursor: pointer'); }
      tag = 'a';
      attrs.push(['href', actionHref(act.target)]);
      if (act.target.charAt(0) === '#' && opts && opts.screenIds && opts.screenIds.indexOf(act.target.slice(1)) < 0) {
        out.warnings.push('节点 ' + node.id + ' 的链接目标屏幕不存在：' + act.target + '（用 design_project 看屏幕 id）');
      }
    }
  } else if (act && act.verb === 'emit') {
    // Vue：组件/屏幕都可用 defineEmits（屏幕是根组件，父级同样能监听）；HTML：静态页无脚本 → 声明式属性
    if (actTarget === 'vue') attrs.push(['@click', "emit('" + act.target + "')"]);
    else if (actTarget !== 'react') attrs.push(['data-action', 'emit:' + act.target]);
  }

  // col 容器里的自宽叶子按内容宽（否则被 flex 拉伸成满宽，与设计预览不一致；满宽按钮/显式 100% 宽度除外）
  if (SELF_WIDTH_TYPES[t] && opts && opts.parentDir === 'col') {
    var stretchW = (t === 'button') && (p.full === true || String(p.w) === 'fill' || String(wCss).indexOf('100%') >= 0);
    if (!stretchW) decls.push('align-self: flex-start');
  }

  var kids = [];
  var children = node.children || [];
  if (!isSlot) {
    var childOpts = optsWithDir(opts, isC ? (t === 'row' ? 'row' : 'col') : 'col');
    // 插槽宿主：内容由屏幕侧传入（渲染层输出 <slot /> / {children}），本子树不生成
    for (var i = 0; i < children.length; i++) {
      var ch = children[i];
      // 抽组件（design_codegen component=<节点 id>）：被抽走的子节点在原位只留一个组件引用
      //   （不再产出它的标记与 CSS —— 那些归组件自己的产物）
      if (opts && opts.refs && opts.refs[String(ch.id)]) {
        kids.push(buildRefNode(ch, tokens, out, opts.refs[String(ch.id)], opts));
        continue;
      }
      kids.push(buildCodeTree(ch, tokens, out, childOpts));
    }
  }
  return {
    id: node.id, type: t, tag: tag, cls: cls, attrs: attrs, text: text, svg: svg,
    selfClose: selfClose, children: kids, decls: decls, extra: extra,
    checked: p.checked === true, textProp: textProp, slot: isSlot,
    slotName: isSlot ? treeSlotName(node, opts) : '',
    action: act || null,
  };
}

// utf8Bytes：UTF-8 字节数（产物统计必须按字节，不能按字符数 —— 中文字符 3 字节）
function utf8Bytes(s) {
  var t = String(s), n = 0;
  for (var i = 0; i < t.length; i++) {
    var c = t.charCodeAt(i);
    if (c < 0x80) n += 1;
    else if (c < 0x800) n += 2;
    else if (c >= 0xd800 && c <= 0xdbff) { n += 4; i++; }
    else n += 3;
  }
  return n;
}

// argArr：数组/逗号或空格分隔字符串参数 → 字符串数组
function argArr(args, key) {
  var v = args ? args[key] : undefined;
  if (v === undefined || v === null) return [];
  if (Array.isArray(v)) {
    var out = [];
    for (var i = 0; i < v.length; i++) out.push(String(v[i]));
    return out;
  }
  var s = String(v).trim();
  if (!s) return [];
  return s.split(/[,\s]+/).filter(function (x) { return !!x; });
}

function safeFileName(s) {
  var v = String(s).replace(/[^A-Za-z0-9._-]/g, '-');
  if (!v) v = 'screen';
  return v;
}

// ── 结构树 → CSS 规则（DFS 顺序 → 产物确定） ────────────────
// refNode：组件引用节点（抽组件后原位只留 <HeroCard />，样式由被引组件自带）
//   attrs = 与组件默认值有差异的 prop（相同则省略）；groups = 插槽组（默认槽 + 具名槽）
//   children 是各插槽组子内容的**扁平并集**（CSS 规则与引用收集按它遍历，渲染按 groups 分组）
function refNode(info, attrs, groups) {
  var gs = groups || [];
  var kids = [];
  for (var gi = 0; gi < gs.length; gi++) {
    for (var gj = 0; gj < gs[gi].children.length; gj++) kids.push(gs[gi].children[gj]);
  }
  return {
    id: 'ref:' + info.name, type: 'component', ref: info.name, tag: info.name, cls: '',
    attrs: attrs || [], text: '', svg: null, selfClose: !kids.length, children: kids,
    groups: gs,
    decls: [], extra: [], checked: false, textProp: '', slot: false,
  };
}

// buildRefNode：屏幕里的被抽实例 → 组件引用节点
//   插槽子内容在**屏幕的类名表**里渲染（屏幕侧写实内容，组件侧留 <slot />），两套类名互不干扰
function buildRefNode(memberNode, tokens, out, info, opts) {
  var raw = slotGroupsOf(memberNode, info);
  var groups = [];
  for (var i = 0; i < raw.length; i++) {
    var kids = [];
    for (var j = 0; j < raw[i].children.length; j++) {
      // 插槽子内容按屏幕目标形态渲染（Vue 的 @click / HTML 的 data-action 取决于 target）
      kids.push(buildCodeTree(raw[i].children[j], tokens, out, {
        refs: null,
        target: opts && opts.target,
        screenIds: opts && opts.screenIds,
      }));
    }
    groups.push({ name: raw[i].name, children: kids });
  }
  var rattrs = refAttrsFor(memberNode, info);
  if (info.near && info.nearStyles) {
    // 近似组实例：内联覆盖配色变量（Vue/HTML 走静态 style；React 由 renderTreeJsx 转成对象形式）
    var nst = info.nearStyles[String(memberNode.id)];
    if (nst) rattrs.push(['style', nst]);
  }
  return refNode(info, rattrs, groups);
}

// collectRefs：结构树里用到的组件名（去重 + DFS 出现顺序 → import 顺序确定）
function collectRefs(tree) {
  var seen = {};
  var list = [];
  function walk(t) {
    if (t.ref && !seen[t.ref]) { seen[t.ref] = true; list.push(t.ref); }
    for (var i = 0; i < t.children.length; i++) walk(t.children[i]);
  }
  walk(tree);
  return list;
}

function treeCssRules(tree, lines) {
  if (tree.ref) {
    // 引用节点本身不产 CSS（样式在被引组件的样式文件里），但**插槽子内容**在屏幕的类名表里，要产
    for (var rc = 0; rc < tree.children.length; rc++) treeCssRules(tree.children[rc], lines);
    return;
  }
  lines.push('.' + tree.cls + ' {');
  for (var i = 0; i < tree.decls.length; i++) lines.push('  ' + tree.decls[i] + ';');
  lines.push('}');
  for (var e = 0; e < tree.extra.length; e++) {
    var ex = tree.extra[e];
    lines.push(ex.sel + ' {');
    for (var j = 0; j < ex.decls.length; j++) lines.push('  ' + ex.decls[j] + ';');
    lines.push('}');
  }
  for (var c = 0; c < tree.children.length; c++) treeCssRules(tree.children[c], lines);
}

// ── 结构树 → HTML / Vue 模板 与 JSX ─────────────────────────
function svgMarkup(svg, jsx) {
  if (jsx) {
    return '<svg width={' + fmtNum(svg.size) + '} height={' + fmtNum(svg.size) + '} viewBox="0 0 24 24" ' +
      'fill="none" stroke="currentColor" strokeWidth={1.6} strokeLinecap="round" strokeLinejoin="round" ' +
      'aria-hidden="true"><path d="' + svg.path + '" /></svg>';
  }
  return '<svg width="' + fmtNum(svg.size) + '" height="' + fmtNum(svg.size) + '" viewBox="0 0 24 24" ' +
    'fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" ' +
    'aria-hidden="true"><path d="' + svg.path + '"/></svg>';
}
// jsxText：JSX 里裸文本的 `{`/`}` 会当表达式 → 转 HTML 实体（JSX 支持实体）
function jsxText(s) {
  return String(s).replace(/\{/g, '&#123;').replace(/\}/g, '&#125;');
}
function indent(n) {
  var s = '';
  for (var i = 0; i < n; i++) s += '  ';
  return s;
}

// jsStr：JS 字符串字面量（单引号风格，贴近手写代码；转义 \ ' 与换行）
function jsStr(s) {
  return "'" + String(s).replace(/\\/g, '\\\\').replace(/'/g, "\\'").replace(/\r?\n/g, '\\n') + "'";
}
// jsxAttr：JSX 属性值里的 " / { / } 必须转义（裸花括号会被当表达式边界）
function jsxAttr(v) {
  return String(v).replace(/"/g, '&quot;').replace(/\{/g, '&#123;').replace(/\}/g, '&#125;');
}
// renderRefAttrs：组件引用上的属性（Vue/HTML 走 escHtml，JSX 走 jsxAttr）
function renderRefAttrs(attrs, jsx) {
  var s = '';
  for (var i = 0; i < attrs.length; i++) {
    var a = attrs[i];
    s += (a[1] === true) ? (' ' + a[0]) : (' ' + a[0] + '="' + (jsx ? jsxAttr(a[1]) : escHtml(a[1])) + '"');
  }
  return s;
}

// cssVarsJsx：'--c2-bg: #fff; --c5-fg: #000' → "{ '--c2-bg': '#fff', '--c5-fg': '#000' }"
//   React 的 style 只接受对象（CSS 变量的 key 必须带引号）
function cssVarsJsx(text) {
  var parts = String(text).split(';');
  var entries = [];
  for (var i = 0; i < parts.length; i++) {
    var p = parts[i].trim();
    if (!p) continue;
    var ci = p.indexOf(':');
    if (ci < 0) continue;
    entries.push(jsStr(p.slice(0, ci).trim()) + ': ' + jsStr(p.slice(ci + 1).trim()));
  }
  return '{ ' + entries.join(', ') + ' }';
}
// renderRefAttrsJsx：引用节点属性 → JSX（style 走对象形式：近似组的配色变量覆盖）
function renderRefAttrsJsx(attrs) {
  var styleText = '';
  var keep = [];
  for (var i = 0; i < attrs.length; i++) {
    if (attrs[i][0] === 'style' && typeof attrs[i][1] === 'string') styleText = attrs[i][1];
    else keep.push(attrs[i]);
  }
  var s = renderRefAttrs(keep, true);
  if (styleText) s += ' style={' + cssVarsJsx(styleText) + '}';
  return s;
}

// renderTreeMarkup：HTML 与 Vue 模板共用（差异仅自闭合写法：HTML 用 '>'，Vue 用 ' />'）
function renderTreeMarkup(tree, depth, lines, selfCloseStr) {
  var ind = indent(depth);
  if (tree.ref) {
    var rattr = renderRefAttrs(tree.attrs, false);
    if (!tree.children.length) { lines.push(ind + '<' + tree.ref + rattr + ' />'); return; }
    lines.push(ind + '<' + tree.ref + rattr + '>');
    var rgrp = tree.groups || [];
    if (!rgrp.length) {
      for (var rc0 = 0; rc0 < tree.children.length; rc0++) {
        renderTreeMarkup(tree.children[rc0], depth + 1, lines, selfCloseStr);
      }
    } else {
      // 插槽：默认槽直接写标签内；具名槽用 <template #name>（Vue 语法 —— HTML 目标不抽组件，走不到这里）
      for (var rg = 0; rg < rgrp.length; rg++) {
        var gp = rgrp[rg];
        if (gp.name === 'default') {
          for (var gc0 = 0; gc0 < gp.children.length; gc0++) {
            renderTreeMarkup(gp.children[gc0], depth + 1, lines, selfCloseStr);
          }
        } else {
          lines.push(ind + '  <template #' + gp.name + '>');
          for (var gc1 = 0; gc1 < gp.children.length; gc1++) {
            renderTreeMarkup(gp.children[gc1], depth + 2, lines, selfCloseStr);
          }
          lines.push(ind + '  </template>');
        }
      }
    }
    lines.push(ind + '</' + tree.ref + '>');
    return;
  }
  var attr = ' class="' + tree.cls + '"';
  for (var i = 0; i < tree.attrs.length; i++) {
    var a = tree.attrs[i];
    attr += (a[1] === true) ? (' ' + a[0]) : (' ' + a[0] + '="' + escHtml(a[1]) + '"');
  }
  var tag = tree.tag;
  // checkbox：label 内包 <input type="checkbox"> + 文本
  if (tree.type === 'checkbox') {
    lines.push(ind + '<' + tag + attr + '>');
    lines.push(ind + '  <input type="checkbox"' + (tree.checked ? ' checked' : '') + selfCloseStr);
    if (tree.textProp) lines.push(ind + '  <span>{{ ' + tree.textProp + ' }}</span>');
    else if (tree.text) lines.push(ind + '  <span>' + escHtml(tree.text) + '</span>');
    lines.push(ind + '</' + tag + '>');
    return;
  }
  if (tree.svg) {
    lines.push(ind + '<' + tag + attr + '>' + svgMarkup(tree.svg, false) + '</' + tag + '>');
    return;
  }
  if (tree.selfClose) {
    lines.push(ind + '<' + tag + attr + selfCloseStr);
    return;
  }
  if (tree.slot) {
    // 插槽宿主：内容由屏幕侧传入（子树已在屏幕侧渲染），组件里只留外壳 + 插槽口
    lines.push(ind + '<' + tag + attr + '>');
    lines.push(ind + '  ' + ((tree.slotName && tree.slotName !== 'default')
      ? ('<slot name="' + tree.slotName + '" />') : '<slot />'));
    lines.push(ind + '</' + tag + '>');
    return;
  }
  if (!tree.children.length && !tree.text) {
    lines.push(ind + '<' + tag + attr + '></' + tag + '>');
    return;
  }
  lines.push(ind + '<' + tag + attr + '>');
  if (tree.textProp) lines.push(ind + '  {{ ' + tree.textProp + ' }}');
  else if (tree.text) lines.push(ind + '  ' + escHtml(tree.text));
  for (var c = 0; c < tree.children.length; c++) renderTreeMarkup(tree.children[c], depth + 1, lines, selfCloseStr);
  lines.push(ind + '</' + tag + '>');
}

// renderTreeJsx：React（className 走 CSS Modules；SVG 属性 camelCase；checkbox 用 defaultChecked）
function renderTreeJsx(tree, depth, lines) {
  var ind = indent(depth);
  if (tree.ref) {
    var rattr0 = renderRefAttrsJsx(tree.attrs);
    var jgrp = tree.groups || [];
    var named = [];
    var defKids = [];
    for (var jg = 0; jg < jgrp.length; jg++) {
      if (jgrp[jg].name === 'default') defKids = defKids.concat(jgrp[jg].children);
      else named.push(jgrp[jg]);
    }
    if (!named.length) {
      if (!tree.children.length) { lines.push(ind + '<' + tree.ref + rattr0 + ' />'); return; }
      lines.push(ind + '<' + tree.ref + rattr0 + '>');
      for (var rc0 = 0; rc0 < tree.children.length; rc0++) renderTreeJsx(tree.children[rc0], depth + 1, lines);
      lines.push(ind + '</' + tree.ref + '>');
      return;
    }
    // 具名插槽：React 里插槽就是 props（header={<></>}）→ 走多行属性形式（Fragment 包裹多子节点）
    lines.push(ind + '<' + tree.ref);
    var attrsTxt = renderRefAttrsJsx(tree.attrs).trim();
    if (attrsTxt) lines.push(ind + '  ' + attrsTxt);
    for (var jn = 0; jn < named.length; jn++) {
      lines.push(ind + '  ' + named[jn].name + '={<>');
      for (var jc = 0; jc < named[jn].children.length; jc++) {
        renderTreeJsx(named[jn].children[jc], depth + 2, lines);
      }
      lines.push(ind + '  </>}');
    }
    if (!defKids.length) { lines.push(ind + '/>'); return; }
    lines.push(ind + '>');
    for (var dc = 0; dc < defKids.length; dc++) renderTreeJsx(defKids[dc], depth + 1, lines);
    lines.push(ind + '</' + tree.ref + '>');
    return;
  }
  var clsRefs = [];
  var clsParts = String(tree.cls).split(' ');
  for (var i = 0; i < clsParts.length; i++) clsRefs.push("styles['" + clsParts[i] + "']");
  var attr = ' className={' + clsRefs.join(" + ' ' + ") + '}';
  for (var j = 0; j < tree.attrs.length; j++) {
    var a = tree.attrs[j];
    attr += (a[1] === true) ? (' ' + a[0]) : (' ' + a[0] + '="' + String(a[1]).replace(/"/g, '&quot;') + '"');
  }
  // 交互：React 侧 emit → 回调 prop（on<Name>）；函数由组件 props 解构或屏幕内占位提供
  if (tree.action && tree.action.verb === 'emit') attr += ' onClick={' + handlerName(tree.action.target) + '}';
  var tag = tree.tag;
  if (tree.type === 'checkbox') {
    lines.push(ind + '<' + tag + attr + '>');
    lines.push(ind + '  <input type="checkbox"' + (tree.checked ? ' defaultChecked' : '') + ' />');
    if (tree.textProp) lines.push(ind + '  <span>{' + tree.textProp + '}</span>');
    else if (tree.text) lines.push(ind + '  <span>' + jsxText(tree.text) + '</span>');
    lines.push(ind + '</' + tag + '>');
    return;
  }
  if (tree.svg) {
    lines.push(ind + '<' + tag + attr + '>' + svgMarkup(tree.svg, true) + '</' + tag + '>');
    return;
  }
  if (tree.selfClose) {
    lines.push(ind + '<' + tag + attr + ' />');
    return;
  }
  if (tree.slot) {
    lines.push(ind + '<' + tag + attr + '>');
    lines.push(ind + '  {' + ((tree.slotName && tree.slotName !== 'default') ? tree.slotName : 'children') + '}');
    lines.push(ind + '</' + tag + '>');
    return;
  }
  if (!tree.children.length && !tree.text) {
    lines.push(ind + '<' + tag + attr + '></' + tag + '>');
    return;
  }
  lines.push(ind + '<' + tag + attr + '>');
  if (tree.textProp) lines.push(ind + '  {' + tree.textProp + '}');
  else if (tree.text) lines.push(ind + '  ' + jsxText(tree.text));
  for (var c = 0; c < tree.children.length; c++) renderTreeJsx(tree.children[c], depth + 1, lines);
  lines.push(ind + '</' + tag + '>');
}

// screenCodeTree：屏幕 → 结构树（屏幕容器 + 去尺寸的根节点树）
//   ★ 根节点的 w/h 在设计里等于画布尺寸（layoutScreen 的默认），代码里必须去掉——
//     否则生成的是写死 360×640 的死布局，而不是自适应屏幕。
function screenCodeTree(screen, tokens, out, opts) {
  // ★ 每次调用（每个屏幕 × 每个 target）都用**独立的类名去重表**：三个 target 各自从零
  //   分配同名后缀，产物才彼此一致 —— 否则 html 先占掉 .d-x，vue 里就成了 .d-x-2（实测
  //   踩过：Vue/React 产物类名多出 -2/-3 后缀，module.css 与 html 版对不上）。
  //   自动编号节点的统计按节点 id 跨 target 去重，提示里的数字才不虚高。
  if (!out.autoIdIds) out.autoIdIds = {};
  var local = { used: {}, autoIdIds: out.autoIdIds, warnings: out.warnings };
  var root = screen.root ? deepClone(screen.root) : null;
  var kids = [];
  if (root) {
    root.props = root.props || {};
    delete root.props.w;
    delete root.props.h;
    // 屏幕容器是 flex-direction: column → 根节点的父方向按 col 计（自宽叶子要按内容宽）
    kids.push(buildCodeTree(root, tokens, local, optsWithDir(opts, 'col')));
  } else {
    // ★ 空屏（screen.add 用默认 template='none' → 没有根节点）要显式提示：
    //   否则用户拿到一个只有空容器的组件，不知道是"设计为空"还是"生成失败"。
    out.warnings.push('屏幕 ' + screen.id + ' 是空屏（没有根节点）：生成的组件只有屏幕容器。' +
      '先用 design_edit node.add（不带 parent，首个节点会成为根）加内容再重新生成');
  }
  var decls = ['display: flex', 'flex-direction: column', 'width: 100%',
    'max-width: ' + fmtNum(num(screen.width, 360)) + 'px',
    'min-height: ' + fmtNum(num(screen.height, 640)) + 'px',
    'margin: 0 auto',
    'background: ' + (colorPropCss(screen.background, tokens) || 'var(--color-bg)')];
  return {
    tag: 'main', cls: screenClass(screen), attrs: [['data-screen', screen.id]], text: '', svg: null,
    selfClose: false, children: kids, decls: decls, extra: [], type: 'screen', checked: false,
  };
}

function screenCssText(screen, tokens, tree) {
  var lines = [
    '/* ' + screen.name + '（' + screen.id + '）—— 由 tool-design 从设计工程生成，勿手改；',
    '   真相源：设计工程 design.project.json + 设计令牌 design.tokens.json（改设计后重新生成）',
    '   布局：flex 流式（语义 class .d-<节点 id>），颜色/间距/字号/圆角全部走令牌变量 */',
    '',
    '*, *::before, *::after { box-sizing: border-box; }',
    '',
  ];
  treeCssRules(tree, lines);
  lines.push('');
  return lines.join('\n');
}

// ── 三种 target 的产物生成 ─────────────────────────────────
// HTML target 用 inline 模式（不传 opts.refs）：静态页没有组件机制，内联渲染才自包含可预览；
//   被抽子树的组件片段另出 html/components/（见 genComponentArtifacts）。
function genHtmlArtifacts(screen, tokens, targetDir, out) {
  // target 参与构建：link/emit 的绑定形态按目标形态生成（HTML 静态页 → 真链接 + data-action）
  var tree = screenCodeTree(screen, tokens, out, { target: 'html', screenIds: out.screenIds });
  var file = safeFileName(screen.id);
  var html = [];
  html.push('<!DOCTYPE html>');
  html.push('<html lang="zh-CN">');
  html.push('<head>');
  html.push('<meta charset="utf-8">');
  html.push('<meta name="viewport" content="width=device-width, initial-scale=1">');
  html.push('<title>' + escHtml(screen.name) + ' · ' + escHtml(str(out.title, '设计')) + '</title>');
  html.push('<link rel="stylesheet" href="../tokens.css">');
  html.push('<link rel="stylesheet" href="' + file + '.css">');
  html.push('</head>');
  html.push('<body>');
  var bodyLines = [];
  renderTreeMarkup(tree, 0, bodyLines, '>');
  for (var i = 0; i < bodyLines.length; i++) html.push(bodyLines[i]);
  html.push('</body>');
  html.push('</html>');
  html.push('');
  return [
    { rel: targetDir + '/html/' + file + '.html', content: html.join('\n'), kind: 'html' },
    { rel: targetDir + '/html/' + file + '.css', content: screenCssText(screen, tokens, tree), kind: 'css' },
  ];
}

// ═══ Tailwind 目标 —— 与 css 版同源同视觉，只是把「声明」换成 utility class ═══════
// 取舍：① 颜色/间距/圆角一律走**令牌变量任意值**（bg-[var(--color-surface)]）—— 令牌仍是唯一真相源，
//          产物不出现裸色值，改 design.tokens.json 后重新生成即全局同步；
//      ② 命中 Tailwind 默认 scale 的用语义 class（p-4 / gap-3 / rounded-lg / font-semibold），
//          否则用任意值兜底（p-[var(--space-md)]）—— 视觉与 css 版逐声明等价；
//      ③ 映射不了的声明**不静默丢**：收集成告警（输出 + codegen.report.json 的 warnings 都列）。
var TW_SPACE = {
  '0': '0', '1': 'px', '2': '0.5', '4': '1', '6': '1.5', '8': '2', '10': '2.5', '12': '3', '14': '3.5',
  '16': '4', '20': '5', '24': '6', '28': '7', '32': '8', '36': '9', '40': '10', '44': '11', '48': '12',
  '56': '14', '64': '16', '80': '20', '96': '24',
};
var TW_WEIGHT = { '400': 'font-normal', '500': 'font-medium', '600': 'font-semibold', '700': 'font-bold' };
var TW_WEIGHT_NAME = { normal: 'font-normal', medium: 'font-medium', semibold: 'font-semibold', bold: 'font-bold' };
var TW_RADIUS = {
  '0': 'rounded-none', '2': 'rounded-sm', '4': 'rounded', '6': 'rounded-md',
  '8': 'rounded-lg', '12': 'rounded-xl', '16': 'rounded-2xl', '24': 'rounded-3xl',
};
var TW_ALIGN = { 'flex-start': 'start', 'center': 'center', 'flex-end': 'end', 'stretch': 'stretch', 'baseline': 'baseline' };
var TW_JUSTIFY = {
  'flex-start': 'start', 'center': 'center', 'flex-end': 'end',
  'space-between': 'between', 'space-around': 'around', 'space-evenly': 'evenly',
};

// twArb：任意值语法（★ 值里的空格要写成下划线，否则类名被空格断开）。
// ★ 类型提示是**必须**的：值含 var()/calc()/clamp() 时 Tailwind 推断不出所属属性
//   （实测 text-[var(--font-size-sm)] 被当成 color → font-size 落回 16px，整屏文字全变大）⇒ 显式写 text-[length:…] / text-[color:…]。
function twArb(prefix, value, hint) {
  var v = String(value).trim();
  var body = v.replace(/\s+/g, '_');
  var ambiguous = (v.indexOf('var(') >= 0 || v.indexOf('calc(') >= 0 || v.indexOf('clamp(') >= 0);
  return prefix + '-[' + ((hint && ambiguous) ? (hint + ':' + body) : body) + ']';
}
// twSplitValues：按空格拆值，但**括号内不拆**（var(--space-md, 8px) 是一个值）
function twSplitValues(v) {
  var parts = []; var depth = 0; var cur = '';
  for (var i = 0; i < v.length; i++) {
    var ch = v.charAt(i);
    if (ch === '(') depth++;
    else if (ch === ')') depth--;
    if (ch === ' ' && depth <= 0) { if (cur) parts.push(cur); cur = ''; continue; }
    cur += ch;
  }
  if (cur) parts.push(cur);
  return parts;
}
function twSpaceOne(prefix, value) {
  var v = String(value).trim();
  var m = /^(\d+(?:\.\d+)?)px$/.exec(v);
  if (m && TW_SPACE[m[1]] !== undefined) return prefix + '-' + TW_SPACE[m[1]];
  if (v === '0') return prefix + '-0';
  if (v === 'auto') return prefix + '-auto';
  return twArb(prefix, v);
}
function twBox(prop, value) {      // padding / margin 简写 → 方向 class
  var b = prop === 'padding' ? 'p' : 'm';
  var parts = twSplitValues(String(value).trim());
  if (parts.length === 1) return [twSpaceOne(b, parts[0])];
  if (parts.length === 2) return [twSpaceOne(b + 'y', parts[0]), twSpaceOne(b + 'x', parts[1])];
  if (parts.length === 3) return [twSpaceOne(b + 't', parts[0]), twSpaceOne(b + 'x', parts[1]), twSpaceOne(b + 'b', parts[2])];
  return [twSpaceOne(b + 't', parts[0]), twSpaceOne(b + 'r', parts[1]), twSpaceOne(b + 'b', parts[2]), twSpaceOne(b + 'l', parts[3])];
}
function twForDecl(prop, value) {
  var v = String(value).trim();
  if (prop === 'display') {
    if (v === 'flex') return ['flex'];
    if (v === 'inline-flex') return ['inline-flex'];
    if (v === 'block') return ['block'];
    if (v === 'inline-block') return ['inline-block'];
    if (v === 'inline') return ['inline'];
    if (v === 'grid') return ['grid'];
    if (v === 'none') return ['hidden'];
    return [];
  }
  if (prop === 'flex-direction') return v === 'column' ? ['flex-col'] : (v === 'row' ? ['flex-row'] : []);
  if (prop === 'gap') return [twSpaceOne('gap', v)];
  if (prop === 'padding' || prop === 'margin') return twBox(prop, v);
  if (prop === 'width') {
    if (v === '100%') return ['w-full'];
    if (v === 'fit-content') return ['w-fit'];
    return [twSpaceOne('w', v)];
  }
  if (prop === 'max-width') {
    if (v === '100%') return ['max-w-full'];
    if (v === 'fit-content') return ['max-w-fit'];
    return [twArb('max-w', v)];
  }
  if (prop === 'min-width') return v === '100%' ? ['min-w-full'] : [twArb('min-w', v)];
  if (prop === 'height') return v === '100%' ? ['h-full'] : [twSpaceOne('h', v)];
  if (prop === 'min-height') return v === '0' ? ['min-h-0'] : [twArb('min-h', v)];
  if (prop === 'max-height') return v === '100%' ? ['max-h-full'] : [twArb('max-h', v)];
  if (prop === 'flex') {
    if (v === '1 1 auto') return ['flex-1'];
    if (v === '0 0 auto') return ['flex-none'];
    if (v === 'none') return ['flex-none'];
    if (v === 'auto') return ['flex-auto'];
    if (v === 'initial') return ['flex-initial'];
    return [];
  }
  if (prop === 'flex-wrap') {
    if (v === 'wrap') return ['flex-wrap'];
    if (v === 'nowrap') return ['flex-nowrap'];
    if (v === 'wrap-reverse') return ['flex-wrap-reverse'];
    return [];
  }
  if (prop === 'flex-grow') return (v === '1' || v === '1 1') ? ['grow'] : ['grow-0'];
  if (prop === 'flex-shrink') return v === '0' ? ['shrink-0'] : ['shrink'];
  if (prop === 'align-self') return TW_ALIGN[v] ? ['self-' + TW_ALIGN[v]] : [];
  if (prop === 'align-items') return TW_ALIGN[v] ? ['items-' + TW_ALIGN[v]] : [];
  if (prop === 'justify-content') return TW_JUSTIFY[v] ? ['justify-' + TW_JUSTIFY[v]] : [];
  if (prop === 'background' || prop === 'background-color') return [twArb('bg', v, 'color')];
  if (prop === 'color') return [twArb('text', v, 'color')];
  if (prop === 'border') {
    if (v === 'none') return ['border-0'];
    var bm = /^(\d+(?:\.\d+)?)px solid (.+)$/.exec(v);
    if (!bm) return [];
    return [(bm[1] === '1' ? 'border' : twArb('border', bm[1] + 'px', 'length')), twArb('border', bm[2], 'color')];
  }
  if (prop === 'border-radius') {
    var rm = /^(\d+(?:\.\d+)?)px$/.exec(v);
    if (rm && TW_RADIUS[rm[1]] !== undefined) return [TW_RADIUS[rm[1]]];
    return [twArb('rounded', v)];
  }
  if (prop === 'box-shadow') return [twArb('shadow', v)];
  if (prop === 'font-size') return [twArb('text', v, 'length')];
  if (prop === 'font-weight') {
    if (TW_WEIGHT[v]) return [TW_WEIGHT[v]];
    // 令牌变量（var(--font-weight-semibold)）→ 直接落语义类；Tailwind 的字重提示是 number（weight 不识别）
    var fwm = /^var\(--font-weight-([a-z-]+)\)$/.exec(v);
    if (fwm && TW_WEIGHT_NAME[fwm[1]]) return [TW_WEIGHT_NAME[fwm[1]]];
    return [twArb('font', v, 'number')];
  }
  if (prop === 'letter-spacing') return [twArb('tracking', v)];
  if (prop === 'text-transform') {
    if (v === 'uppercase') return ['uppercase'];
    if (v === 'lowercase') return ['lowercase'];
    if (v === 'capitalize') return ['capitalize'];
    if (v === 'none') return ['normal-case'];
    return [];
  }
  if (prop === 'text-decoration') return v === 'none' ? ['no-underline'] : (v === 'underline' ? ['underline'] : []);
  if (prop === 'cursor') {
    if (v === 'pointer') return ['cursor-pointer'];
    if (v === 'default') return ['cursor-default'];
    if (v === 'text') return ['cursor-text'];
    if (v === 'move') return ['cursor-move'];
    return [twArb('cursor', v)];
  }
  if (prop === 'line-height') return [twArb('leading', v)];
  if (prop === 'text-align') return (v === 'center' || v === 'left' || v === 'right') ? ['text-' + v] : [];
  if (prop === 'overflow') return v === 'hidden' ? ['overflow-hidden'] : (v === 'auto' ? ['overflow-auto'] : []);
  if (prop === 'overflow-wrap') return v === 'break-word' ? ['break-words'] : [];
  if (prop === 'white-space') return v === 'nowrap' ? ['whitespace-nowrap'] : [];
  if (prop === 'opacity') return [twArb('opacity', v)];
  if (prop === 'object-fit') return ['object-' + v];
  if (prop === 'aspect-ratio') {
    var av = v.replace(/\s+/g, '');
    if (av === '1/1' || av === '1') return ['aspect-square'];
    if (av === '16/9') return ['aspect-video'];
    if (av === 'auto') return ['aspect-auto'];
    return [twArb('aspect', av)];
  }
  if (prop === 'position') {
    if (v === 'relative') return ['relative'];
    if (v === 'absolute') return ['absolute'];
    return [];
  }
  return [];
}
function twApplyTree(tree, warnings, who) {
  var classes = [];
  var i;
  for (i = 0; i < tree.decls.length; i++) {
    var d = tree.decls[i];
    var ci = d.indexOf(':');
    if (ci < 0) continue;
    var prop = d.slice(0, ci).trim();
    var val = d.slice(ci + 1).trim();
    var mapped = twForDecl(prop, val);
    if (!mapped.length) {
      warnings.push('tailwind：节点 ' + who + ' 的声明未能映射为 utility class —— ' + prop + ': ' + val + '（请在项目样式里补上）');
      continue;
    }
    for (var j = 0; j < mapped.length; j++) if (classes.indexOf(mapped[j]) < 0) classes.push(mapped[j]);
  }
  if (tree.extra && tree.extra.length) {
    warnings.push('tailwind：节点 ' + who + ' 带 ' + tree.extra.length + ' 条派生规则（子选择器/伪类）—— utility class 表达不了，请改用 targets=["html"] 或自行补样式');
  }
  tree.tw = classes.join(' ');
  for (i = 0; i < tree.children.length; i++) twApplyTree(tree.children[i], warnings, tree.children[i].cls || '?');
}
function genTailwindArtifacts(screen, tokens, targetDir, out) {
  var tree = screenCodeTree(screen, tokens, out, { target: 'html', screenIds: out.screenIds });
  twApplyTree(tree, out.warnings, tree.cls);
  // class 换成 utility 串（tailwind 版不产 CSS 规则文件）；原语义 class 转 data-node 保留可定位性
  (function swap(t) {
    t.attrs.push(['data-node', t.cls]);
    t.cls = t.tw;
    for (var i = 0; i < t.children.length; i++) swap(t.children[i]);
  })(tree);
  var file = safeFileName(screen.id);
  var bodyLines = [];
  renderTreeMarkup(tree, 0, bodyLines, '>');
  var html = [];
  html.push('<!DOCTYPE html>');
  html.push('<html lang="zh-CN">');
  html.push('<head>');
  html.push('<meta charset="utf-8">');
  html.push('<meta name="viewport" content="width=device-width, initial-scale=1">');
  html.push('<title>' + escHtml(screen.name) + ' · ' + escHtml(str(out.title, '设计')) + '（Tailwind）</title>');
  html.push('<!-- 由 tool-design 从设计工程生成（targets 含 "tailwind"），勿手改；改设计请重新生成（幂等）。');
  html.push('     样式 = Tailwind utility class：颜色/间距/圆角走令牌变量任意值（bg-[var(--color-surface)]），');
  html.push('     故需引入 ../tokens.css，并让 Tailwind 的 content 覆盖 tailwind/ 目录。 -->');
  html.push('<link rel="stylesheet" href="../tokens.css">');
  html.push('</head>');
  html.push('<body>');
  for (var i = 0; i < bodyLines.length; i++) html.push(bodyLines[i]);
  html.push('</body>');
  html.push('</html>');
  html.push('');
  return [{ rel: targetDir + '/tailwind/' + file + '.html', content: html.join('\n'), kind: 'html' }];
}

function genTailwindReadme(targetDir) {
  var L = [];
  L.push('# Tailwind 产物说明');
  L.push('');
  L.push('由 tool-design 生成（`design_codegen targets=["tailwind"]`），勿手改 —— 改设计后重新生成（幂等、位级一致）。');
  L.push('');
  L.push('## 产物');
  L.push('- `' + targetDir + '/tailwind/<屏幕>.html`：class 全部是 Tailwind utility；节点原语义 class 保留在 `data-node` 属性上便于定位。');
  L.push('- 颜色 / 间距 / 字号 / 圆角一律走**令牌变量任意值**（`bg-[var(--color-surface)]`、`p-[var(--space-md)]`）—— 产物里没有裸色值。');
  L.push('');
  L.push('## 接入三步');
  L.push('1. 安装 Tailwind（v3+；任意值 `[…]` 与 `var()` 依赖 JIT 编译）。');
  L.push('2. `tailwind.config.js` 的 `content` 覆盖本目录：`"./' + targetDir + '/tailwind/**/*.html"`。');
  L.push('3. 引入 `' + targetDir + '/tokens.css`（设计令牌 → CSS 自定义属性），否则 `var(--color-*)` 全部落空。');
  L.push('');
  L.push('## 与 targets=["html"] 的关系');
  L.push('- 两者**逐声明等价**（同一棵代码树、同一套 flex 流式规则）：html 版把声明写成 `.d-<节点 id>` 规则，本版写成 utility class。');
  L.push('- 映射规则：命中 Tailwind 默认 scale → 语义 class（`p-4` / `gap-3` / `rounded-lg` / `font-semibold`）；');
  L.push('  其余 → 任意值兜底（`text-[13px]`、`w-[42%]`、`p-[var(--space-md)]`）；值内空格按 Tailwind 语法写成下划线。');
  L.push('');
  L.push('## 边界（不静默降级）');
  L.push('- 只生成 **HTML** 形态：不产出 Vue / React 的 Tailwind 版（组件与 scoped 样式另说）。');
  L.push('- 不参与组件抽取：`component=` 在 tailwind 产物下不生效（`tailwind/<屏>.html` 始终内联渲染）。');
  L.push('- 表达不了的声明（派生选择器 / 伪类 / 未知属性）会进 `codegen.report.json` 的 `warnings`，不会静默丢。');
  L.push('');
  return L.join('\n');
}

function genVueArtifact(screen, tokens, targetDir, out, opts) {
  var tree = screenCodeTree(screen, tokens, out, opts);
  var name = pascalCase(screen.id);
  var bodyLines = [];
  renderTreeMarkup(tree, 1, bodyLines, ' />');
  var refs = collectRefs(tree);
  var emits = collectActions(tree);
  var lines = [];
  lines.push('<!-- ' + escHtml(screen.name) + '（' + screen.id + '）—— 由 tool-design 从设计工程生成，勿手改；');
  lines.push('     真相源：design.project.json + design.tokens.json（改设计后重新生成，生成幂等）。');
  lines.push('     依赖：项目入口需引入 tokens.css（设计令牌 → CSS 自定义属性）-->');
  if (refs.length || emits.length) {
    lines.push('<script setup>');
    for (var r = 0; r < refs.length; r++) lines.push("import " + refs[r] + " from './components/" + refs[r] + ".vue'");
    if (emits.length) {
      if (refs.length) lines.push('');
      lines.push('// 交互：设计工程里 props.action="emit:<名字>" 的节点 → 屏幕作为根组件把事件交给父级监听');
      lines.push('const emit = defineEmits([' + emitList(emits) + '])');
    }
    lines.push('</script>');
    lines.push('');
  }
  lines.push('<template>');
  for (var i = 0; i < bodyLines.length; i++) lines.push(bodyLines[i]);
  lines.push('</template>');
  lines.push('');
  lines.push('<style scoped>');
  var cssLines = [
    '/* 布局：flex 流式；颜色/间距/字号/圆角全部走令牌变量（var(--color-fg) 等） */',
    '*, *::before, *::after { box-sizing: border-box; }',
    '',
  ];
  treeCssRules(tree, cssLines);
  for (var j = 0; j < cssLines.length; j++) lines.push(cssLines[j]);
  lines.push('</style>');
  lines.push('');
  return [{ rel: targetDir + '/vue/' + name + '.vue', content: lines.join('\n'), kind: 'vue' }];
}

function genReactArtifacts(screen, tokens, targetDir, out, opts) {
  var tree = screenCodeTree(screen, tokens, out, opts);
  var name = pascalCase(screen.id);
  var bodyLines = [];
  renderTreeJsx(tree, 1, bodyLines);
  var refs = collectRefs(tree);
  var emits = collectActions(tree);
  var lines = [];
  lines.push('// ' + screen.name + '（' + screen.id + '）—— 由 tool-design 从设计工程生成，勿手改；');
  lines.push('// 真相源：design.project.json + design.tokens.json（改设计后重新生成，生成幂等）。');
  lines.push("// 依赖：同目录 " + name + ".module.css + 项目入口引入 tokens.css（令牌 → CSS 变量）");
  lines.push("import styles from './" + name + ".module.css'");
  for (var rr = 0; rr < refs.length; rr++) lines.push("import " + refs[rr] + " from './components/" + refs[rr] + "'");
  lines.push('');
  lines.push('export default function ' + name + '() {');
  if (emits.length) {
    lines.push('  // 交互占位：设计工程里 props.action="emit:<名字>" 的节点 → 接上你的事件处理');
    for (var e1 = 0; e1 < emits.length; e1++) lines.push('  const ' + handlerName(emits[e1]) + ' = () => {}');
    lines.push('');
  }
  lines.push('  return (');
  for (var i = 0; i < bodyLines.length; i++) lines.push('  ' + bodyLines[i]);
  lines.push('  )');
  lines.push('}');
  lines.push('');
  return [
    { rel: targetDir + '/react/' + name + '.jsx', content: lines.join('\n'), kind: 'react' },
    { rel: targetDir + '/react/' + name + '.module.css', content: screenCssText(screen, tokens, tree), kind: 'css' },
  ];
}

// ── 抽组件（design_codegen component=<节点 id>） ─────────────
// componentName：节点 → 组件名（PascalCase）。命名来源与 codeClass 同一套优先级：
//   ① props.name（icon 除外 —— 那是图标名，不是节点名）② 语义 id ③ 类型-序号（模板自动编号 id）
function componentName(node) {
  var p = node.props || {};
  var nm = (node.type === 'icon') ? '' : str(p.name, '');
  var base;
  if (nm && /^[A-Za-z][A-Za-z0-9_-]*$/.test(nm)) {
    base = nm;
  } else {
    var id = String(node.id);
    var m = /^(.*)_n(\d+)$/.exec(id);
    base = m ? (String(node.type) + '-' + m[2]) : id;
  }
  return pascalCase(base);
}

// ── 组件参数化：内容字段 → props ────────────────────────────
// prop 命名：props.prop（显式指定名）> props.name > 节点 id（模板自动编号 c1_n7 → text-7）；
//   props.prop === false → 该节点内容不提升，保持字面量。
//   与 codeClass 同一套「语义优先」原则：屏幕上看到的属性名直接可读（<Card title="…" />）。
var PROP_RESERVED = { key: 1, ref: 1, class: 1, style: 1, children: 1, slot: 1, props: 1, on: 1 };
function camelCase(s) {
  var parts = String(s).split(/[^A-Za-z0-9]+/);
  var out = '';
  for (var i = 0; i < parts.length; i++) {
    var p = parts[i];
    if (!p) continue;
    if (!out) out = p.charAt(0).toLowerCase() + p.slice(1);
    else out += p.charAt(0).toUpperCase() + p.slice(1);
  }
  if (!out) out = 'text';
  if (/^[0-9]/.test(out)) out = 't' + out;
  // 保留字（Vue/React 特殊属性）不能当 prop 名 → 加 text 前缀（如 key → textKey）
  if (PROP_RESERVED[out.toLowerCase()]) out = 'text' + out.charAt(0).toUpperCase() + out.slice(1);
  return out;
}
function propNameFor(node, used) {
  var p = node.props || {};
  if (p.prop === false) return '';                 // 显式排除
  var nm = str(p.prop, '');
  if (!nm) {
    var vn = (node.type === 'icon') ? '' : str(p.name, '');
    if (vn) nm = vn;
    else {
      var id = String(node.id);
      var m = /^(.*)_n(\d+)$/.exec(id);
      nm = m ? ('text-' + m[2]) : id;              // 自动编号 id → text-7（模板序号，可读可循）
    }
  }
  var name = camelCase(nm);
  if (used[name]) {
    var n = 2;
    while (used[name + n]) n++;
    name = name + n;
  }
  used[name] = true;
  return name;
}

// collectCompProps：组件原型子树 → 内容节点序列 seq（含被排除项，供实例按位对齐）+ prop 清单 list
//   ★ 先序顺序必须与 buildCodeTree 分配 prop 名的顺序一致（父自身 → children），
//     屏幕实例传参才能按 seq 索引严格对齐（同构保证形状相同）。
function collectCompProps(node) {
  var used = {};
  var seq = [];
  var list = [];
  (function walk(n) {
    var f = contentField(n);
    if (f) {
      var nm = propNameFor(n, used);
      var def = str((n.props || {})[f], '');
      seq.push({ field: f, name: nm, def: def });
      if (nm) list.push({ name: nm, field: f, def: def });
    }
    if (isSlotHost(n)) return;                     // 插槽宿主：内容来自外部，子树不参与 prop 分配
    var cs = n.children || [];
    for (var i = 0; i < cs.length; i++) walk(cs[i]);
  })(node);
  return { seq: seq, list: list };
}

// 组件 props 上限：整屏被抽成组件时会出现几十个文案 prop（签名荒诞且没人会用）
//   → 超过上限退化为不参数化，并给出提示（拆小组件 / 显式 props: 'none'）。
var MAX_COMP_PROPS = 12;

// 结构指纹（同构判定）：节点类型 + 关键样式，**剔除** id / props.name / 内容字段 / 插槽标记。
//   键值用「长度:值」写法，避免值里出现分隔符造成假碰撞。
function styleFingerprint(node) {
  var p = node.props || {};
  var cf = contentField(node);
  var keys = [];
  for (var k in p) {
    if (k === 'name' || k === 'prop' || k === 'slot') continue;
    if (cf && k === cf) continue;
    var v = p[k];
    if (v === undefined || v === null || v === '') continue;
    var sv = (typeof v === 'object') ? JSON.stringify(v) : String(v);
    keys.push(k + '=' + sv.length + ':' + sv);
  }
  keys.sort();
  return keys.join('&');
}
function structFingerprint(node) {
  var s = String(node.type) + '{' + styleFingerprint(node) + '}';
  var cs = node.children || [];
  for (var i = 0; i < cs.length; i++) {
    var one = structFingerprint(cs[i]);
    s += '[' + one.length + ':' + one + ']';
  }
  return s;
}

// findDuplicateGroups：屏幕集合 → 同构重复组（可跨屏复用同一组件）
//   规则：出现 ≥ minCount 次、子树节点数 ≥ minNodes、跳过屏幕根（它是屏幕容器不是区块）；
//   大组优先，成员全部落在已选组子树内的候选被丢弃（避免抽出一堆碎片组件）。
function collectFingerprints(screenId, node, ancestors, index, buckets, counter) {
  var fp = structFingerprint(node);
  if (!index[fp]) {
    index[fp] = { fp: fp, members: [], first: counter.n++ };
    buckets.push(index[fp]);
  }
  index[fp].members.push({
    node: node, screenId: screenId, size: countSub(node), ancestors: ancestors.slice(0),
  });
  var kids = node.children || [];
  var next = ancestors.concat([String(node.id)]);
  for (var i = 0; i < kids.length; i++) collectFingerprints(screenId, kids[i], next, index, buckets, counter);
}
function findDuplicateGroups(screens, minCount, minNodes) {
  var index = {};
  var buckets = [];
  var counter = { n: 0 };
  for (var s = 0; s < screens.length; s++) {
    var scr = screens[s];
    if (!scr.root) continue;
    var tops = scr.root.children || [];
    for (var t = 0; t < tops.length; t++) {
      collectFingerprints(scr.id, tops[t], [String(scr.root.id)], index, buckets, counter);
    }
  }
  var groups = [];
  for (var b = 0; b < buckets.length; b++) {
    var bk = buckets[b];
    if (bk.members.length < minCount) continue;
    if (bk.members[0].size < minNodes) continue;
    groups.push(bk);
  }
  groups.sort(function (a, b2) {
    if (b2.members[0].size !== a.members[0].size) return b2.members[0].size - a.members[0].size;
    return a.first - b2.first;
  });
  var chosen = [];
  var covered = {};
  for (var g = 0; g < groups.length; g++) {
    var gr = groups[g];
    var allInside = gr.members.length > 0;
    for (var mi = 0; mi < gr.members.length && allInside; mi++) {
      var anc = gr.members[mi].ancestors;
      var inside = false;
      for (var ai = 0; ai < anc.length; ai++) if (covered[anc[ai]]) { inside = true; break; }
      if (!inside) allInside = false;
    }
    if (allInside) continue;
    chosen.push(gr);
    for (var mj = 0; mj < gr.members.length; mj++) covered[String(gr.members[mj].node.id)] = true;
  }
  return chosen;
}

// findNodeByKey：在单个屏幕里先按节点 id 找；找不到再按 props.name 找（同 name 多个 → 报错要 id）
function findNodeByKey(screen, key) {
  var byId = null;
  var byName = [];
  function walk(n) {
    if (byId) return;
    if (String(n.id) === key) { byId = n; return; }
    var nm = (n.type === 'icon') ? '' : str((n.props || {}).name, '');
    if (nm && nm === key) byName.push(n);
    var cs = n.children || [];
    for (var i = 0; i < cs.length; i++) walk(cs[i]);
  }
  if (screen.root) walk(screen.root);
  if (byId) return byId;
  if (byName.length > 1) {
    throw new Error('component=' + key + ' 在屏幕 ' + screen.id + ' 里匹配到多个同 props.name 的节点：请改用节点 id 指定');
  }
  return byName.length ? byName[0] : null;
}

// findNodeById：跨屏按 (屏幕 id, 节点 id) 查节点（近似组只给 screen/id，这里取回节点本体）
function findNodeById(screens, screenId, nodeId) {
  for (var i = 0; i < screens.length; i++) {
    if (String(screens[i].id) !== String(screenId)) continue;
    var hit = null;
    if (screens[i].root) {
      forEachNode(screens[i].root, function (n) { if (!hit && String(n.id) === String(nodeId)) hit = n; });
    }
    return hit;
  }
  return null;
}
// markCovered：标记节点及其整个子树（自动抽取时用于"大组优先、碎片不再单独抽"）
function markCovered(node, map) {
  map[String(node.id)] = true;
  var ks = node.children || [];
  for (var i = 0; i < ks.length; i++) markCovered(ks[i], map);
}

// resolveExtracts：component 参数 → 抽取清单 + 每屏「节点 id → 组件信息」映射
//   keys 支持：节点 id / props.name / 'auto'（自动识别同构重复区块 → 同一组件多实例复用）
//   一个组件 = 一组**同构实例**（显式指定时只有 1 个实例）；原型取组内第一个（先序 → 确定）。
//   返回 {
//     list: [{ key, name, screenId, node, members, auto, seq, props, paramOff, slots }],
//     byScreen: { <屏幕 id>: { <节点 id>: <组件信息> } }
//   }
//   同名组件自动追加序号；同一节点重复指定只抽一次（幂等：结果只取决于工程与参数）
function resolveExtracts(screens, keys, opts) {
  opts = opts || {};
  var list = [];
  var byScreen = {};
  var usedNames = {};
  var warnings = opts.warnings || [];

  function memberKey(screenId, nodeId) { return String(screenId) + '/' + String(nodeId); }
  function alreadyTaken(screenId, nodeId) {
    for (var i = 0; i < list.length; i++) {
      var ms = list[i].members;
      for (var m = 0; m < ms.length; m++) {
        if (memberKey(ms[m].screenId, ms[m].node.id) === memberKey(screenId, nodeId)) return true;
      }
    }
    return false;
  }

  // addGroup：一组同构实例 → 一个组件条目（原型 members[0]）
  function addGroup(key, members, isAuto, nearMode) {
    var proto = members[0].node;
    var name = componentName(proto);
    if (usedNames[name]) {
      var n2 = 2;
      while (usedNames[name + n2]) n2++;
      name = name + n2;
    }
    usedNames[name] = true;
    var cp = collectCompProps(proto);
    var paramOff = '';
    if (!opts.parametrize) paramOff = 'none';
    else if (cp.list.length > MAX_COMP_PROPS) {
      paramOff = 'tooMany';
      warnings.push('组件 ' + name + ' 的文案 prop 有 ' + cp.list.length + ' 个（超过上限 ' + MAX_COMP_PROPS +
        '）→ 本次未参数化（内容写死）。建议拆成更小的组件；想固定不参数化可显式传 props: "none"');
    }
    // 插槽表：支持**多个**插槽宿主（具名多插槽）—— 名字归一化/去重/保留名校验都在这里做完，
    //   组件产物与屏幕引用都按这张表对齐（顺序 = DFS，实例与原型同构 → 一一对应）
    var propList = paramOff ? [] : cp.list;
    var propNames = {};
    for (var pn = 0; pn < propList.length; pn++) propNames[propList[pn].name] = true;
    var slots = resolveSlotNames(proto, propNames, name, warnings);
    // 近似组：配色差异提升为 CSS 变量（组件默认值 = 原型实例的值，实例内联覆盖变量）
    var nearDiff = null;
    var nearStyles = null;
    if (nearMode) {
      nearDiff = nearDiffTable(members);
      nearStyles = {};
      for (var ns = 0; ns < members.length; ns++) {
        var st = nearStylesFor(members[ns].node, nearDiff, opts.tokens);
        if (st) nearStyles[String(members[ns].node.id)] = st;
      }
    }
    var info = {
      key: key, name: name, screenId: members[0].screenId, node: proto,
      members: members, auto: !!isAuto,
      seq: cp.seq, props: propList, paramOff: paramOff,
      slots: slots,
      near: !!nearMode, nearDiff: nearDiff, nearStyles: nearStyles,
      slotNodeId: slots.length ? slots[0].nodeId : '',
      slotLabel: slots.length ? slots[0].name : '',
    };
    list.push(info);
    for (var m2 = 0; m2 < members.length; m2++) {
      var sid = members[m2].screenId;
      if (!byScreen[sid]) byScreen[sid] = {};
      byScreen[sid][String(members[m2].node.id)] = info;
    }
    return info;
  }

  var explicit = [];
  var auto = false;
  var near = false;
  for (var k0 = 0; k0 < keys.length; k0++) {
    var kk = str(keys[k0], '').trim();
    if (!kk) continue;
    if (kk.toLowerCase() === 'auto') auto = true;
    else if (kk.toLowerCase() === 'near') near = true;
    else explicit.push(kk);
  }
  // ① 显式指定（节点 id / props.name）
  for (var i = 0; i < explicit.length; i++) {
    var key = explicit[i];
    var node = null;
    var screenId = '';
    for (var s = 0; s < screens.length; s++) {
      var hit = findNodeByKey(screens[s], key);
      if (hit) { node = hit; screenId = screens[s].id; break; }
    }
    if (!node) {
      throw new Error('component 指定的节点在当前屏幕里不存在：' + key +
        '（用 design_project 看屏幕与节点 id；节点 id 是 design_edit node.add 时传的 id）');
    }
    if (alreadyTaken(screenId, node.id)) continue;
    addGroup(key, [{ node: node, screenId: screenId }], false);
  }
  // ② auto：结构同构 + 关键样式一致 且出现 ≥2 次的区块（跨屏也算，组件天然可跨屏复用）
  if (auto) {
    var groups = findDuplicateGroups(screens, 2, 3);
    for (var g = 0; g < groups.length; g++) {
      var gr = groups[g];
      var skip = false;
      for (var gm = 0; gm < gr.members.length; gm++) {
        if (alreadyTaken(gr.members[gm].screenId, gr.members[gm].node.id)) { skip = true; break; }
      }
      if (skip) continue;
      addGroup('auto', gr.members, true);
    }
  }
  // ③ near：结构相同、**只有配色类字段不同**的区块 → 抽成同一组件 + 配色差异提升为 CSS 变量
  //    与 auto 的差别：auto 要求关键样式也一致；near 允许 bg/color/border/shadow 不同（差异进变量）
  if (near) {
    var ngroups = findNearGroups(screens, 2, 3);
    var chosenNodes = {};
    for (var lt = 0; lt < list.length; lt++) markCovered(list[lt].node, chosenNodes);
    var toneSkipped = 0;
    var slotSkipped = 0;
    var coveredSkipped = 0;
    for (var ng0 = 0; ng0 < ngroups.length; ng0++) {
      var grp0 = ngroups[ng0];
      var gmembers = [];
      var missing = false;
      for (var gm0 = 0; gm0 < grp0.members.length; gm0++) {
        var it0 = grp0.members[gm0];
        var nd0 = findNodeById(screens, it0.screen, it0.id);
        if (!nd0) { missing = true; break; }
        gmembers.push({ node: nd0, screenId: it0.screen });
      }
      if (missing || !gmembers.length) continue;
      var covered = false;
      for (var gc0 = 0; gc0 < gmembers.length; gc0++) {
        if (chosenNodes[String(gmembers[gc0].node.id)]) covered = true;
      }
      if (covered) { coveredSkipped++; continue; }
      var unsupported = '';
      for (var gf = 0; gf < grp0.diffKeys.length; gf++) {
        if (!NEAR_VAR_SUPPORT[grp0.diffKeys[gf]]) unsupported = grp0.diffKeys[gf];
      }
      if (unsupported) {
        if (unsupported === 'tone') toneSkipped++;
        continue;
      }
      var hasSlot = false;
      for (var gs0 = 0; gs0 < gmembers.length && !hasSlot; gs0++) {
        forEachNode(gmembers[gs0].node, function (n) { if (isSlotHost(n)) hasSlot = true; });
      }
      if (hasSlot) { slotSkipped++; continue; }
      for (var gm1 = 0; gm1 < gmembers.length; gm1++) markCovered(gmembers[gm1].node, chosenNodes);
      addGroup('near', gmembers, true, true);
    }
    if (toneSkipped) {
      warnings.push(toneSkipped + ' 组近似区块只有 tone（语义色）不同 → 未自动抽取：tone 同时决定前景与背景两条声明，' +
        '不是单一 CSS 声明可变量化的单位；把 tone 对齐成同一档，或分开用 component=<节点 id> 各抽各的');
    }
    if (slotSkipped) {
      warnings.push(slotSkipped + ' 组近似区块含插槽宿主 → 未自动抽取（插槽内容与配色变量的对齐无法保证）：' +
        '请用 component=<节点 id> 显式抽取');
    }
    opts.nearStats = { toneSkipped: toneSkipped, slotSkipped: slotSkipped, coveredSkipped: coveredSkipped };
  }
  return { list: list, byScreen: byScreen };
}

// ── 近似同构（结构相同、**只有颜色类字段不同**）─────────────────────────
// ★ 只报告、不抽取：颜色差异会让同一个组件的各实例与该组件默认值不一致（默认值只能有一份），
//   强行抽会把「设计上本就要区分的配色」压成一个组件 → 产物语义错。所以只提示差异字段，
//   由用户决定「对齐配色后再抽」还是「本来就是两种区块」。
var COLOR_KEYS = { bg: 1, color: 1, border: 1, shadow: 1, tone: 1 };
function colorSemantic(node) {
  var p = node.props || {};
  var out = {};
  for (var k in COLOR_KEYS) {
    var v = p[k];
    if (v === undefined || v === null || v === '') continue;
    out[k] = (typeof v === 'object') ? JSON.stringify(v) : String(v);
  }
  return out;
}
// 宽松指纹：颜色类字段归一化为 <color>（其余与严格指纹一致 → 组内差异只可能来自颜色）
function styleFingerprintLoose(node) {
  var p = node.props || {};
  var cf = contentField(node);
  var keys = [];
  for (var k in p) {
    if (k === 'name' || k === 'prop' || k === 'slot') continue;
    if (cf && k === cf) continue;
    var v = p[k];
    if (v === undefined || v === null || v === '') continue;
    var sv = COLOR_KEYS[k] ? '<color>' : ((typeof v === 'object') ? JSON.stringify(v) : String(v));
    keys.push(k + '=' + sv.length + ':' + sv);
  }
  keys.sort();
  return keys.join('&');
}
function structFingerprintLoose(node) {
  var s = String(node.type) + '{' + styleFingerprintLoose(node) + '}';
  var cs = node.children || [];
  for (var i = 0; i < cs.length; i++) {
    var one = structFingerprintLoose(cs[i]);
    s += '[' + one.length + ':' + one + ']';
  }
  return s;
}
// findNearGroups：宽松指纹分组 → 组内严格指纹**不唯一**（确实存在差异）且差异落在颜色字段 → 近似组
function findNearGroups(screens, minCount, minNodes) {
  var index = {};
  var buckets = [];
  var counter = { n: 0 };
  function walk(screenId, node) {
    var fp = structFingerprintLoose(node);
    if (!index[fp]) {
      index[fp] = { fp: fp, members: [], first: counter.n++ };
      buckets.push(index[fp]);
    }
    index[fp].members.push({
      node: node, screenId: screenId, size: countSub(node),
      strict: structFingerprint(node), colors: colorSemantic(node),
    });
    var kids = node.children || [];
    for (var i = 0; i < kids.length; i++) walk(screenId, kids[i]);
  }
  for (var s = 0; s < screens.length; s++) {
    var scr = screens[s];
    if (!scr.root) continue;
    var tops = scr.root.children || [];
    for (var t = 0; t < tops.length; t++) walk(scr.id, tops[t]);
  }
  var out = [];
  for (var b = 0; b < buckets.length; b++) {
    var bk = buckets[b];
    if (bk.members.length < minCount) continue;
    if (bk.members[0].size < minNodes) continue;
    var strictSet = {};
    var nStrict = 0;
    for (var m = 0; m < bk.members.length; m++) {
      if (!strictSet[bk.members[m].strict]) { strictSet[bk.members[m].strict] = true; nStrict++; }
    }
    if (nStrict < 2) continue;                       // 严格同构组 → 归 findDuplicateGroups 管
    // 差异字段：**逐节点**对齐比较（同构 → 先序对齐），取全树并集 ——
    //   只比组根节点自身会漏掉「差异在子节点上」的组（例如 badge 的 tone），那种组会被误判成
    //   「差异不在颜色上」而静默丢弃（实测踩过：子节点 tone 不同的两组既不报也不抽）。
    var dt = nearDiffTable(bk.members);
    var diffSet = {};
    var diffKeys = [];
    for (var sq in dt) {
      var fl = dt[sq].fields;
      for (var fj = 0; fj < fl.length; fj++) {
        if (!diffSet[fl[fj]]) { diffSet[fl[fj]] = true; diffKeys.push(fl[fj]); }
      }
    }
    if (!diffKeys.length) continue;                  // 差异不在颜色类字段上（设计上本就不同）→ 不报，避免噪音
    out.push({
      name: componentName(bk.members[0].node), type: String(bk.members[0].node.type),
      size: bk.members[0].size, count: bk.members.length, diffKeys: diffKeys, first: bk.first,
      members: bk.members.map(function (x) { return { screen: x.screenId, id: String(x.node.id) }; }),
    });
  }
  out.sort(function (a, b2) {
    if (b2.size !== a.size) return b2.size - a.size;
    return a.first - b2.first;
  });
  return out;
}

// ── 近似组自动抽取（component=near）：配色差异 → CSS 变量提升 ─────────────────
// 结构相同、只有颜色类字段不同的实例 → 抽成**同一个**组件，但配色差异不丢：
//   组件内声明写成 var(--c<先序序号>-<字段>, 原型值)，实例处用内联 style 覆盖变量值。
//   ⇒ 一份组件多实例复用，同时「设计上本来就要区分的配色」由实例自带（默认值 = 原型实例的值）。
//   v1 支持 bg / color / border / shadow；tone（badge 语义色，会同时改前景+背景）不支持 → 该组不抽并说明。
var NEAR_VAR_SUPPORT = { bg: 1, color: 1, border: 1, shadow: 1 };
var NEAR_DECL_PREFIX = { bg: 'background: ', color: 'color: ', border: 'border: 1px solid ', shadow: 'box-shadow: ' };
var NEAR_VAR_SUFFIX = { bg: 'bg', color: 'fg', border: 'border', shadow: 'shadow' };
function nearVarName(seq, field) { return '--c' + seq + '-' + (NEAR_VAR_SUFFIX[field] || field); }
// nodeIdBySeq：设计树里先序第 seq 个节点的 id（报告里要指回设计节点，不能只给序号）
function nodeIdBySeq(root, seq) {
  var idx = 0;
  var found = '';
  (function walk(n) {
    if (found) return;
    if (idx === seq) { found = String(n.id); idx++; return; }
    idx++;
    var ks = n.children || [];
    for (var i = 0; i < ks.length; i++) walk(ks[i]);
  })(root);
  return found;
}
// nearDiffTable：同构实例（先序对齐）→ { <先序序号>: { fields:[…], <字段>: <原型值> } }
function nearDiffTable(members) {
  var table = {};
  var idx = 0;
  function walk(nodes) {
    var proto = nodes[0];
    var fields = [];
    for (var f in COLOR_KEYS) {
      var seen = {};
      var nv = 0;
      var any = false;
      for (var i = 0; i < nodes.length; i++) {
        var v = (nodes[i].props || {})[f];
        if (v === undefined || v === null || v === '') continue;
        any = true;
        var sv = String(v);
        if (!seen[sv]) { seen[sv] = true; nv++; }
      }
      if (any && nv > 1) fields.push(f);
    }
    if (fields.length) {
      var entry = { fields: fields };
      for (var j = 0; j < fields.length; j++) entry[fields[j]] = str((proto.props || {})[fields[j]], '');
      table[idx] = entry;
    }
    idx++;
    var kids = proto.children || [];
    for (var c = 0; c < kids.length; c++) {
      var childNodes = [];
      for (var k = 0; k < nodes.length; k++) childNodes.push(((nodes[k].children) || [])[c]);
      walk(childNodes);
    }
  }
  walk(members.map(function (m) { return m.node; }));
  return table;
}
// nearWrapTree：组件树按差异表把命中的 CSS 声明包成 var()（保留原值为默认 → 不传变量即与原设计一致）
function nearWrapTree(tree, table) {
  var idx = 0;
  (function walk(t) {
    var seq = idx++;
    var entry = table[seq];
    if (entry) {
      for (var j = 0; j < entry.fields.length; j++) {
        var f = entry.fields[j];
        if (!NEAR_VAR_SUPPORT[f]) continue;
        var prefix = NEAR_DECL_PREFIX[f];
        var vn = nearVarName(seq, f);
        for (var d = 0; d < t.decls.length; d++) {
          if (t.decls[d].indexOf(prefix) === 0) {
            t.decls[d] = prefix + 'var(' + vn + ', ' + t.decls[d].slice(prefix.length) + ')';
          }
        }
      }
    }
    for (var i = 0; i < t.children.length; i++) walk(t.children[i]);
  })(tree);
}
// nearFieldCss：差异字段值 → CSS 量（与组件内声明用同一套解析，才可比、才能当变量值）
function nearFieldCss(field, v, tokens) {
  if (field === 'shadow') return shadowPropCss(v, tokens) || '';
  return colorPropCss(v, tokens) || '';
}
// nearStylesFor：实例树 → 该实例的内联变量覆盖（'--c2-bg: …; --c5-fg: …'；空串 = 与原型完全一致）
function nearStylesFor(memberNode, table, tokens) {
  var parts = [];
  var idx = 0;
  (function walk(n) {
    var seq = idx++;
    var entry = table[seq];
    if (entry) {
      for (var j = 0; j < entry.fields.length; j++) {
        var f = entry.fields[j];
        if (!NEAR_VAR_SUPPORT[f]) continue;
        var own = str((n.props || {})[f], '');
        if (own === entry[f]) continue;                      // 与原型一致 → 用组件默认值即可
        var cv = nearFieldCss(f, own, tokens);
        if (cv) parts.push(nearVarName(seq, f) + ': ' + cv);
      }
    }
    var ks = n.children || [];
    for (var i = 0; i < ks.length; i++) walk(ks[i]);
  })(memberNode);
  if (!parts.length) return '';
  return parts.join('; ');
}

// detectDuplicates：同构重复区块候选（只报告不抽取）—— 供 design_codegen detect=true 给建议
function detectDuplicates(screens, maxItems) {
  var groups = findDuplicateGroups(screens, 2, 3);
  var out = [];
  for (var g = 0; g < groups.length; g++) {
    var gr = groups[g];
    var members = [];
    for (var m = 0; m < gr.members.length; m++) {
      members.push({ screen: gr.members[m].screenId, id: String(gr.members[m].node.id) });
    }
    out.push({
      name: componentName(gr.members[0].node),
      type: String(gr.members[0].node.type),
      size: gr.members[0].size,
      count: gr.members.length,
      members: members,
      props: collectCompProps(gr.members[0].node).list.map(function (x) { return x.name; }),
    });
    if (out.length >= (maxItems || 20)) break;
  }
  return out;
}

// refAttrsFor：同构实例 → 与组件默认值**有差异**的 prop（相同则省略 → 屏幕产物保持干净）
//   ★ 按 seq 索引对齐（同构保证内容节点序列一致）；seq 里 name 为空的项是显式排除（props.prop=false）
function refAttrsFor(memberNode, comp) {
  var attrs = [];
  var idx = 0;
  (function walk(n) {
    var f = contentField(n);
    if (f) {
      var slot = (comp.seq || [])[idx];
      var val = str((n.props || {})[f], '');
      if (slot && slot.name && val !== slot.def) attrs.push([slot.name, val]);
      idx++;
    }
    if (isSlotHost(n)) return;                     // 插槽宿主内容由外部传入 → 不参与传参对齐
    var cs = n.children || [];
    for (var i = 0; i < cs.length; i++) walk(cs[i]);
  })(memberNode);
  return attrs;
}

// slotGroupsOf：同构实例 → 插槽组（与组件 slots 表**按 DFS 顺序**一一对应；名字取自组件定义）
//   每个组的 children 是实例里该插槽宿主的子树（在**屏幕**的类名表里渲染，作为引用处的插槽内容传入）
function slotGroupsOf(memberNode, comp) {
  var hosts = [];
  forEachNode(memberNode, function (n) { if (isSlotHost(n)) hosts.push(n); });
  var defs = (comp && comp.slots) || [];
  var groups = [];
  for (var i = 0; i < hosts.length; i++) {
    var def = defs[i];
    var nm = def ? def.name : (slotNameNormalize(slotName(hosts[i])) || 'default');
    groups.push({ name: nm, children: hosts[i].children || [] });
  }
  return groups;
}

// genComponentArtifacts：抽取的子树 → 独立组件产物
//   opts.refs = 同屏其它被抽节点（不含自身）→ 组件内部若含别的组件也用引用，避免自引用递归
//   组件内类名用**独立去重表**（与屏幕互不影响）：作用域是 scoped / CSS Modules，同名不冲突
function genComponentArtifacts(target, comp, tokens, targetDir, out, opts) {
  var compOut = { used: {}, autoIdIds: out.autoIdIds, warnings: out.warnings };
  // ★ 组件侧两件独立的事：① 内容字段参数化（comp.props 非空才开 —— props:"none" / 超上限时为空）
  //   ② 插槽占位（props.slot 标记的节点）。target=html 是静态片段（没有组件机制）：两者都不做
  var isComp = (target !== 'html');
  var paramsOn = isComp && !!(comp.props && comp.props.length);
  // 插槽节点 id → 名字（组件产物里据此写 <slot name="x" /> / {x}）
  //   React 侧签名要逐个声明插槽 prop（默认槽是 children，具名槽是 props）
  var compSlots = isComp ? (comp.slots || []) : [];
  var slotNames = {};
  var slotProps = [];
  for (var si = 0; si < compSlots.length; si++) {
    slotNames[compSlots[si].nodeId] = compSlots[si].name;
    slotProps.push(compSlots[si].name === 'default' ? 'children' : compSlots[si].name);
  }
  var compOpts = {
    refs: (opts && opts.refs) ? opts.refs : null,
    target: target,
    screenIds: out.screenIds,
    comp: isComp ? { params: paramsOn, used: {}, list: [], slotNames: slotNames } : null,
  };
  var tree = buildCodeTree(deepClone(comp.node), tokens, compOut, compOpts);
  // 近似组：把配色差异声明包成 CSS 变量（默认值 = 原型实例的值 → 不覆盖变量时与原设计一致）
  if (comp.near && comp.nearDiff) nearWrapTree(tree, comp.nearDiff);
  var compProps = (paramsOn && compOpts.comp.list) ? compOpts.comp.list : [];
  var name = comp.name;
  var note = '由 tool-design 从设计工程生成，勿手改；从屏幕 ' + comp.screenId + ' 的节点 ' +
    comp.node.id + ' 抽出（design_codegen component=' + comp.key + '）';
  if (comp.auto) {
    note = '由 tool-design 从设计工程生成，勿手改；' + comp.members.length +
      (comp.near
        ? ' 个**近似**区块共用（design_codegen component=near：结构一致、仅配色不同 → 配色差异提升为 CSS 变量，实例内联覆盖）'
        : ' 个同构区块共用（design_codegen component=auto 自动识别）') +
      '，原型：屏幕 ' + comp.screenId + ' 的节点 ' + comp.node.id;
  }
  if (target === 'vue') {
    var bodyLines = [];
    renderTreeMarkup(tree, 1, bodyLines, ' />');
    var refs = collectRefs(tree);
    var emits = collectActions(tree);
    var vlines = [];
    vlines.push('<!-- ' + name + ' —— ' + note + '。');
    vlines.push('     依赖：项目入口引入 tokens.css（令牌 → CSS 变量）-->');
    if (refs.length || compProps.length || emits.length) {
      vlines.push('<script setup>');
      for (var i = 0; i < refs.length; i++) vlines.push("import " + refs[i] + " from './" + refs[i] + ".vue'");
      if (compProps.length) {
        if (refs.length) vlines.push('');
        vlines.push('// 内容字段由设计工程提升为 props：默认值 = 设计原值，不传即与原设计一致');
        vlines.push('defineProps({');
        for (var pi = 0; pi < compProps.length; pi++) {
          vlines.push('  ' + compProps[pi].name + ': { type: String, default: ' +
            jsStr(compProps[pi].def) + ' },');
        }
        vlines.push('})');
      }
      if (emits.length) {
        if (refs.length || compProps.length) vlines.push('');
        vlines.push('// 交互：设计工程里 props.action="emit:<名字>" 的节点 → 组件对外发事件（父级 @"<名字>"）');
        vlines.push('const emit = defineEmits([' + emitList(emits) + '])');
      }
      vlines.push('</script>');
      vlines.push('');
    }
    vlines.push('<template>');
    for (var b = 0; b < bodyLines.length; b++) vlines.push(bodyLines[b]);
    vlines.push('</template>');
    vlines.push('');
    vlines.push('<style scoped>');
    var vcss = [
      '/* 组件 ' + name + '：布局 flex 流式；颜色/间距/字号/圆角全走令牌变量 */',
      '*, *::before, *::after { box-sizing: border-box; }',
      '',
    ];
    treeCssRules(tree, vcss);
    vcss.push('');
    for (var c1 = 0; c1 < vcss.length; c1++) vlines.push(vcss[c1]);
    vlines.push('</style>');
    vlines.push('');
    return [{ rel: targetDir + '/vue/components/' + name + '.vue', content: vlines.join('\n'), kind: 'vue-component' }];
  }
  if (target === 'react') {
    var jsxBody = [];
    renderTreeJsx(tree, 1, jsxBody);
    var rrefs = collectRefs(tree);
    var remits = collectActions(tree);
    var rl = [];
    rl.push('// ' + name + ' —— ' + note + '。');
    rl.push("// 依赖：同目录 " + name + ".module.css + 项目入口引入 tokens.css");
    rl.push("import styles from './" + name + ".module.css'");
    for (var j = 0; j < rrefs.length; j++) rl.push("import " + rrefs[j] + " from './" + rrefs[j] + "'");
    rl.push('');
    if (!compProps.length && !slotProps.length && !remits.length) {
      rl.push('export default function ' + name + '() {');
    } else {
      rl.push('export default function ' + name + '({');
      for (var pk = 0; pk < compProps.length; pk++) {
        rl.push('  ' + compProps[pk].name + ' = ' + jsStr(compProps[pk].def) + ',');
      }
      for (var sk = 0; sk < slotProps.length; sk++) rl.push('  ' + slotProps[sk] + ',');
      // 交互：设计工程 props.action="emit:<名字>" 的节点 → 组件回调 prop（默认空实现，可直接接上）
      for (var ek = 0; ek < remits.length; ek++) rl.push('  ' + handlerName(remits[ek]) + ' = () => {},');
      rl.push('}) {');
    }
    rl.push('  return (');
    for (var k2 = 0; k2 < jsxBody.length; k2++) rl.push('  ' + jsxBody[k2]);
    rl.push('  )');
    rl.push('}');
    rl.push('');
    var rcss = [
      '/* 组件 ' + name + '：由 tool-design 从设计工程生成，勿手改 */',
      '*, *::before, *::after { box-sizing: border-box; }',
      '',
    ];
    treeCssRules(tree, rcss);
    rcss.push('');
    return [
      { rel: targetDir + '/react/components/' + name + '.jsx', content: rl.join('\n'), kind: 'react-component' },
      { rel: targetDir + '/react/components/' + name + '.module.css', content: rcss.join('\n'), kind: 'css' },
    ];
  }
  // html：静态页没有组件机制 → 出可直接打开/粘贴的独立片段（自带 tokens.css 链接与本地 CSS）
  var hbody = [];
  renderTreeMarkup(tree, 1, hbody, '>');
  var hl = [];
  hl.push('<!DOCTYPE html>');
  hl.push('<html lang="zh-CN">');
  hl.push('<head>');
  hl.push('<meta charset="utf-8">');
  hl.push('<meta name="viewport" content="width=device-width, initial-scale=1">');
  hl.push('<title>' + escHtml(name) + ' · 组件片段（tool-design 生成）</title>');
  // ★ 组件片段在 <targetDir>/html/components/ 下 → 令牌在**上两级**（../.. 才到 targetDir）。
  //   写成 ../tokens.css 会 404（真浏览器实测踩过：片段样式全丢 → 白屏）。
  hl.push('<link rel="stylesheet" href="../../tokens.css">');
  hl.push('<link rel="stylesheet" href="' + name + '.css">');
  hl.push('</head>');
  hl.push('<body>');
  for (var hb = 0; hb < hbody.length; hb++) hl.push(hbody[hb]);
  hl.push('</body>');
  hl.push('</html>');
  hl.push('');
  var hcss = [
    '/* 组件 ' + name + '（HTML 片段）：屏幕 html/' + safeFileName(comp.screenId) + '.html 里为内联渲染，',
    '   本片段供其它页面复用（相对引用 ../../tokens.css） */',
    '*, *::before, *::after { box-sizing: border-box; }',
    '',
  ];
  treeCssRules(tree, hcss);
  hcss.push('');
  return [
    { rel: targetDir + '/html/components/' + name + '.html', content: hl.join('\n'), kind: 'html-component' },
    { rel: targetDir + '/html/components/' + name + '.css', content: hcss.join('\n'), kind: 'css' },
  ];
}

function genReadme(proj, tokens, targetDir, targets, screens, extracts) {
  var nComp = (extracts && extracts.length) ? extracts.length : 0;
  var lines = [];
  lines.push('# ' + str(proj.meta.title, '设计') + ' —— 前端代码（由 tool-design 生成）');
  lines.push('');
  lines.push('> 本目录由 `design_codegen` 生成，**请勿手改**：真相源是同目录上一级的');
  lines.push('> 设计工程（`' + str(proj.tokens, 'design.tokens.json') + '` 等）。改设计后用同样命令重新生成，生成幂等。');
  lines.push('');
  lines.push('## 目录');
  lines.push('');
  lines.push('```');
  lines.push(targetDir + '/');
  lines.push('├── tokens.css      # 设计令牌 → CSS 自定义属性（所有 target 共用，务必全局引入）');
  lines.push('├── README.md');
  if (targets.indexOf('html') >= 0) lines.push('├── html/           # 零依赖静态页（<屏幕>.html + <屏幕>.css）');
  if (targets.indexOf('vue') >= 0) lines.push('├── vue/            # Vue 3 单文件组件（<Screen>.vue，style scoped）');
  if (targets.indexOf('react') >= 0) lines.push('└── react/          # React 函数组件 + CSS Modules（<Screen>.jsx + .module.css）');
  if (nComp) lines.push('    （各 target 目录下另有 components/ —— 抽出的独立组件，共 ' + nComp + ' 个）');
  lines.push('```');
  lines.push('');
  lines.push('## 屏幕');
  lines.push('');
  for (var i = 0; i < screens.length; i++) {
    lines.push('- `' + screens[i].id + '`（' + screens[i].name + '）· ' +
      fmtNum(num(screens[i].width, 0)) + '×' + fmtNum(num(screens[i].height, 0)) + ' · 组件名 `' +
      pascalCase(screens[i].id) + '`');
  }
  lines.push('');
  if (nComp) {
    lines.push('## 组件（design_codegen component=… 抽出）');
    lines.push('');
    lines.push('下面这些子树已从屏幕里抽出为独立组件（同一组件可有多个**同构实例**）：Vue / React 的屏幕产物中原位是');
    lines.push('**组件引用**，改组件或改设计后重新生成即全站同步；HTML 没有组件机制，屏幕里保持内联渲染');
    lines.push('（静态页仍可独立打开），组件以 `html/components/` 下的片段形式另出，供其它页面复用。');
    lines.push('');
    lines.push('内容字段（文案 / 标签 / alt）已提升为 **props**：默认值 = 设计原值，不传即与原设计一致；');
    lines.push('屏幕引用只在**与默认值不同**时传参 —— 同构实例各自带自己的文案（如 `<TaskCard title="…" />`）。');
    lines.push('');
    for (var cc = 0; cc < extracts.length; cc++) {
      var e2 = extracts[cc];
      var inst = [];
      for (var ii = 0; ii < e2.members.length; ii++) {
        inst.push('`' + e2.members[ii].screenId + '/' + e2.members[ii].node.id + '`');
      }
      lines.push('- `' + e2.name + '` ← ' + (e2.auto ? ('自动识别的 ' + e2.members.length + ' 个同构实例：') : '屏幕 ') +
        inst.join('、') + '（' + e2.node.type + '）');
      if (e2.props && e2.props.length) {
        for (var pp = 0; pp < e2.props.length; pp++) {
          var defTxt = String(e2.props[pp].def).replace(/\s*\n\s*/g, ' ').replace(/`/g, "'");
          lines.push('  - prop `' + e2.props[pp].name + '` ← `' + e2.props[pp].field + '`，默认 `' + defTxt + '`');
        }
      } else if (e2.paramOff === 'tooMany') {
        lines.push('  - props：未参数化（文案 prop 超过 ' + MAX_COMP_PROPS + ' 个上限，内容已写死）');
      }
      if ((e2.slots || []).length) {
        for (var slz = 0; slz < (e2.slots || []).length; slz++) {
          var sld = e2.slots[slz];
          lines.push('  - 插槽 ' + (sld.name === 'default' ? '`默认`' : ('`' + sld.name + '`')) +
            '：设计节点 `' + sld.nodeId + '` 是插槽宿主 → 组件里留 ' +
            (sld.name === 'default' ? '`<slot />`' : ('`<slot name="' + sld.name + '" />`（React `{' + sld.name + '}`）')) +
            '；屏幕侧把该节点的子树写进引用处' +
            (sld.name === 'default' ? '标签内部' : '（Vue `<template #' + sld.name + '>`、React `' + sld.name + '={<></>}`）'));
        }
      }
      if (targets.indexOf('vue') >= 0) {
        lines.push('  - Vue：`vue/components/' + e2.name + '.vue`（屏幕已有 `<script setup>` import，模板里写 `<'
          + e2.name + ' />`）');
      }
      if (targets.indexOf('react') >= 0) {
        lines.push('  - React：`react/components/' + e2.name + '.jsx` + `.module.css`（屏幕已 import，JSX 里写 `<'
          + e2.name + ' />`）');
      }
      if (targets.indexOf('html') >= 0) {
        lines.push('  - HTML：`html/components/' + e2.name + '.html` + `.css`（片段，屏幕里是内联渲染）');
      }
    }
    lines.push('');
  }
  lines.push('## 集成');
  lines.push('');
  if (targets.indexOf('html') >= 0) {
    lines.push('- **静态页**：直接打开 `html/<屏幕>.html`（相对引用 `../tokens.css`，保持目录结构即可）。');
  }
  if (targets.indexOf('vue') >= 0) {
    lines.push('- **Vue**：把 `vue/*.vue` 放进 `src/components/`，在入口引入 `tokens.css`：');
    lines.push('  ```js');
    lines.push("  import './design-export/tokens.css'");
    lines.push('  ```');
  }
  if (targets.indexOf('react') >= 0) {
    lines.push('- **React**：把 `react/*.jsx` + `*.module.css` 放进 `src/components/`，同样全局引入 `tokens.css`。');
    lines.push('  CSS Modules 需要构建器支持（Vite / Next.js 开箱即用）。');
  }
  lines.push('');
  lines.push('## 约定');
  lines.push('');
  lines.push('- class 语义化：`.d-<设计节点 id>`（节点 id 在设计工程里保证唯一）+ 屏幕容器 `.screen-<屏幕 id>`。');
  lines.push('- 颜色 / 间距 / 字号 / 字重 / 行高 / 圆角 / 阴影 **一律引用令牌变量**（`var(--color-accent)`），');
  lines.push('  改 `design.tokens.json` 后重新生成即全站同步；代码里不出现散落色值。');
  lines.push('- 布局为 flex 流式（与设计器的布局引擎同一套规则），尺寸默认自适应，不写死画布像素。');
  lines.push('');
  return lines.join('\n');
}

// ── 产物指纹（防手改被静默覆盖） ────────────────────────────
// 沙箱无 crypto ⇒ 自实现确定性散列：FNV-1a（乘 16777619 展开为移位加法，纯 32 位、跨引擎一致）× 两个种子，
//   拼成 16 位十六进制 + 内容长度后缀。用途只是「同一份内容必得同一个指纹、改一个字符必变」——不做安全承诺。
function contentHash(s) {
  s = String(s);
  var h1 = 0x811c9dc5;
  var h2 = 0x9e3779b9;
  for (var i = 0; i < s.length; i++) {
    var c = s.charCodeAt(i);
    h1 = h1 ^ c;
    h1 = (h1 + ((h1 << 1) + (h1 << 4) + (h1 << 7) + (h1 << 8) + (h1 << 24))) >>> 0;
    h2 = h2 ^ (c + i);
    h2 = (h2 + ((h2 << 1) + (h2 << 4) + (h2 << 7) + (h2 << 8) + (h2 << 24))) >>> 0;
  }
  function hex8(n) { var t = (n >>> 0).toString(16); while (t.length < 8) t = '0' + t; return t; }
  return hex8(h1) + hex8(h2) + '-' + s.length;
}
var MANIFEST_NAME = 'codegen.manifest.json';

// loadManifest：读产物指纹清单（缺失/损坏都当作"无记录"，不阻断生成）
function loadManifest(ctx, dir) {
  var p = dir + '/' + MANIFEST_NAME;
  try {
    if (!ctx.fs.exists(p)) return { files: {} };
    var m = JSON.parse(ctx.fs.readFile(p));
    if (!m || typeof m !== 'object' || !m.files || typeof m.files !== 'object') return { files: {} };
    return m;
  } catch (e) { return { files: {} }; }
}

// ── 落地写入（覆盖前自动 .bak 备份） ────────────────────────
function writeArtifact(ctx, rel, content, report) {
  var existed = false;
  try { existed = ctx.fs.exists(rel); } catch (e) { existed = false; }
  if (existed) {
    var old = '';
    try { old = ctx.fs.readFile(rel); } catch (e2) { old = ''; }
    ctx.fs.writeFile(rel + '.bak', old);
    report.backups.push(rel + '.bak');
  }
  ctx.fs.writeFile(rel, content);
  report.files.push({ path: rel, bytes: utf8Bytes(content), backup: existed });
}

function humanBytes(n) {
  if (n < 1024) return n + ' B';
  return (Math.round(n / 102.4) / 10) + ' KB';
}

// ═══ design_codegen —— 设计 → 前端代码 → 落到项目 ─═══════════
function designCodegen(args, exec, ctx) {
  var path = argStr(args, 'path', 'design.project.json');
  var proj = loadProject(ctx, path);
  var tokens = loadTokensFor(ctx, proj);
  var targets = argArr(args, 'targets');
  if (!targets.length) targets = ['html'];
  var allowed = ['html', 'vue', 'react', 'tailwind'];
  for (var i = 0; i < targets.length; i++) {
    var tg = targets[i].toLowerCase();
    if (allowed.indexOf(tg) < 0) throw new Error('未知 target: ' + targets[i] + '（可用 html / vue / react / tailwind）');
    targets[i] = tg;
  }
  var targetDir = argStr(args, 'targetDir', 'design-export').replace(/\\/g, '/').replace(/\/+$/, '');
  if (!targetDir) throw new Error('targetDir 不能为空');
  if (targetDir.indexOf('..') >= 0) throw new Error('targetDir 不允许包含 ..（越界保护）：' + targetDir);
  var onlyScreen = argStr(args, 'screen', '');
  var dryRun = argBool(args, 'dryRun', false);
  var compKeys = argArr(args, 'component');
  var propsMode = str(argStr(args, 'props', 'auto'), 'auto').toLowerCase();
  if (propsMode !== 'auto' && propsMode !== 'none') {
    throw new Error('props 只支持 auto（默认：内容字段提升为组件 props）或 none（内容写死，不参数化）');
  }
  var detect = argBool(args, 'detect', false);

  var screens = [];
  for (var s = 0; s < proj.screens.length; s++) {
    if (onlyScreen && proj.screens[s].id !== onlyScreen) continue;
    screens.push(proj.screens[s]);
  }
  if (!screens.length) {
    throw new Error(onlyScreen ? ('屏幕不存在: ' + onlyScreen) : '工程里没有任何屏幕（先用 design_edit screen.add 加屏幕）');
  }

  var out = { warnings: [], title: str(proj.meta.title, '设计') };
  out.screenIds = proj.screens.map(function (x) { return String(x.id); });   // link:#屏幕id 的可达性判定
  var artifacts = [];
  // 抽组件：component=<节点 id> → 子树抽成独立组件，屏幕里原位换成组件引用；
  //   component=auto → 自动识别**同构重复区块**（跨屏也算），同一组件多实例复用
  var exOpts = { parametrize: propsMode === 'auto', warnings: out.warnings, tokens: tokens };
  var ex = resolveExtracts(screens, compKeys, exOpts);
  var extracts = ex.list;
  // auto 但一个都没识别到 → 明确报错（静默无操作最难排查）
  var autoRequested = false;
  var nearRequested = false;
  for (var ck = 0; ck < compKeys.length; ck++) {
    var lk = str(compKeys[ck], '').toLowerCase();
    if (lk === 'auto') autoRequested = true;
    if (lk === 'near') nearRequested = true;
  }
  if (nearRequested && !extracts.length) {
    var st0 = exOpts.nearStats || {};
    throw new Error('component=near 未识别到可自动抽取的近似组（判据：结构一致、仅 bg/color/border/shadow 不同、' +
      '出现 ≥2 次、子树 ≥3 节点、屏幕根不参与）—— 先用 detect=true 看候选' +
      (st0.toneSkipped ? ('；其中 ' + st0.toneSkipped + ' 组只因 tone（语义色）不同 → tone 同时决定前景与背景两条声明，' +
        '不能单变量化：把 tone 对齐成同一档，或分开用 component=<节点 id> 各抽各的') : '') +
      (st0.slotSkipped ? ('；其中 ' + st0.slotSkipped + ' 组含插槽宿主 → 请用 component=<节点 id> 显式抽取') : ''));
  }
  if (autoRequested && !extracts.length) {
    // 近似同构（结构相同、只有颜色不同）不自动抽取 → 失败信息里给出**可行动**的下一步，
    //   否则用户只会看到「没识别到」，不知道其实只差配色对齐
    var nearHint = '';
    var nearsHint = findNearGroups(screens, 2, 3);
    if (nearsHint.length) {
      nearHint = '；但检测到 ' + nearsHint.length + ' 组**近似**同构区块（结构相同、只有颜色类字段不同：' +
        nearsHint[0].name + ' 等 ' + nearsHint[0].count + ' 个，差异字段 ' + nearsHint[0].diffKeys.join('/') +
        '）—— 近似组不会自动抽取（颜色差异会与组件默认值冲突）：把配色对齐成同一套后再抽，' +
        '或把它们当两种区块分别用 component=<节点 id> 抽';
    }
    throw new Error('component=auto 未识别到同构重复区块（判据：结构 + 关键样式一致、出现 ≥2 次、子树 ≥3 个节点、' +
      '屏幕根不参与）—— 先用 detect=true 看候选，或用 component=<节点 id> 显式指定' + nearHint);
  }
  // detect：只报告同构重复区块候选（不抽取）—— 已在抽取清单里的组不再重复列出
  var dupCandidates = [];
  var nearCandidates = [];
  if (detect) {
    var cands = detectDuplicates(screens, 20);
    for (var dc0 = 0; dc0 < cands.length; dc0++) {
      var taken = false;
      for (var dm0 = 0; dm0 < cands[dc0].members.length; dm0++) {
        var mm0 = cands[dc0].members[dm0];
        if (ex.byScreen[mm0.screen] && ex.byScreen[mm0.screen][mm0.id]) { taken = true; break; }
      }
      if (!taken) dupCandidates.push(cands[dc0]);
    }
    // 近似组（结构相同、只有颜色不同）：只报告差异字段 + 为什么没抽
    var ncands = findNearGroups(screens, 2, 3);
    for (var nc0 = 0; nc0 < ncands.length; nc0++) {
      var ntaken = false;
      for (var nm0 = 0; nm0 < ncands[nc0].members.length; nm0++) {
        var nm1 = ncands[nc0].members[nm0];
        if (ex.byScreen[nm1.screen] && ex.byScreen[nm1.screen][nm1.id]) { ntaken = true; break; }
      }
      if (!ntaken) nearCandidates.push(ncands[nc0]);
    }
  }
  // 交互清单（报告 + 面板用）：受控 action → 每处交互的位置与语义
  var interactions = [];
  for (var si3 = 0; si3 < screens.length; si3++) {
    if (!screens[si3].root) continue;
    forEachNode(screens[si3].root, function (n) {
      var ia = nodeAction(n);
      if (!ia) return;
      // problem：语法/语义问题（报告与面板只认这一个字段 —— 合法但类型不适配也算问题）
      var ip = ia.error || '';
      if (!ip && ia.verb === 'toggle' && n.type !== 'checkbox') ip = 'toggle 只对 checkbox 有意义';
      if (!ip && ia.verb === 'link' && LINKABLE_TYPES.indexOf(n.type) < 0) {
        ip = 'link 只对 ' + LINKABLE_TYPES.join('/') + ' 有意义';
      }
      if (!ip && ia.verb === 'link' && ia.target.charAt(0) === '#' && out.screenIds.indexOf(ia.target.slice(1)) < 0) {
        ip = '链接目标屏幕不存在';
      }
      interactions.push({
        screen: screens[si3].id, id: String(n.id), type: String(n.type),
        action: ia.raw, verb: ia.verb, target: ia.target || '', error: ip,
      });
    });
  }
  if (extracts.length && targets.indexOf('html') >= 0 && targets.indexOf('vue') < 0 && targets.indexOf('react') < 0) {
    out.warnings.push('HTML 没有组件机制：屏幕保持内联渲染（静态页可独立预览），组件另出 html/components/ 片段；' +
      '要真正的组件引用，targets 请加上 vue 或 react');
  }
  if (extracts.length && targets.indexOf('tailwind') >= 0) {
    out.warnings.push('tailwind 产物不抽取组件（component= 只作用于 html / vue / react）：' + targetDir +
      '/tailwind/<屏幕>.html 始终内联渲染 —— 需要组件请用 vue / react 产物');
  }
  // 公共产物
  artifacts.push({ rel: targetDir + '/tokens.css', content: renderTokensCss(tokens), kind: 'tokens' });
  artifacts.push({ rel: targetDir + '/README.md', content: genReadme(proj, tokens, targetDir, targets, screens, extracts), kind: 'readme' });
  if (targets.indexOf('tailwind') >= 0) artifacts.push({ rel: targetDir + '/tailwind/README.md', content: genTailwindReadme(targetDir), kind: 'readme' });
  // 各屏幕 × 各 target
  for (var k = 0; k < screens.length; k++) {
    var refsK = ex.byScreen[screens[k].id] || null;
    if (targets.indexOf('html') >= 0) {
      var ha = genHtmlArtifacts(screens[k], tokens, targetDir, out);
      for (var a1 = 0; a1 < ha.length; a1++) artifacts.push(ha[a1]);
    }
    if (targets.indexOf('vue') >= 0) {
      var va = genVueArtifact(screens[k], tokens, targetDir, out, { refs: refsK, target: 'vue', screenIds: out.screenIds });
      for (var a2 = 0; a2 < va.length; a2++) artifacts.push(va[a2]);
    }
    if (targets.indexOf('react') >= 0) {
      var ra = genReactArtifacts(screens[k], tokens, targetDir, out, { refs: refsK, target: 'react', screenIds: out.screenIds });
      for (var a3 = 0; a3 < ra.length; a3++) artifacts.push(ra[a3]);
    }
    if (targets.indexOf('tailwind') >= 0) {
      var ta = genTailwindArtifacts(screens[k], tokens, targetDir, out);
      for (var a4 = 0; a4 < ta.length; a4++) artifacts.push(ta[a4]);
    }
  }
  // 抽出的组件（每个组件 × 每个 target）
  var compFiles = [];
  for (var ci = 0; ci < extracts.length; ci++) {
    var comp = extracts[ci];
    var innerRefs = {};
    for (var si = 0; si < extracts.length; si++) {
      var oth = extracts[si];
      // ★ 值必须是组件信息对象（含 props / 插槽元数据）：引用节点要据此传参与塞插槽内容
      if (oth.screenId === comp.screenId && oth.node.id !== comp.node.id) innerRefs[String(oth.node.id)] = oth;
    }
    var cArtifacts = [];
    if (targets.indexOf('html') >= 0) {
      var ca1 = genComponentArtifacts('html', comp, tokens, targetDir, out, { refs: innerRefs });
      for (var q1 = 0; q1 < ca1.length; q1++) cArtifacts.push(ca1[q1]);
    }
    if (targets.indexOf('vue') >= 0) {
      var ca2 = genComponentArtifacts('vue', comp, tokens, targetDir, out, { refs: innerRefs });
      for (var q2 = 0; q2 < ca2.length; q2++) cArtifacts.push(ca2[q2]);
    }
    if (targets.indexOf('react') >= 0) {
      var ca3 = genComponentArtifacts('react', comp, tokens, targetDir, out, { refs: innerRefs });
      for (var q3 = 0; q3 < ca3.length; q3++) cArtifacts.push(ca3[q3]);
    }
    for (var q4 = 0; q4 < cArtifacts.length; q4++) artifacts.push(cArtifacts[q4]);
    compFiles.push({
      id: comp.node.id, name: comp.name, screen: comp.screenId,
      auto: !!comp.auto,
      near: !!comp.near,
      nearVars: comp.near ? Object.keys(comp.nearDiff).map(function (k) {
        var e = comp.nearDiff[k];
        return {
          seq: Number(k), node: nodeIdBySeq(comp.node, Number(k)),
          fields: e.fields.map(function (f) { return { field: f, cssVar: nearVarName(Number(k), f) }; }),
        };
      }) : [],
      instances: comp.members.map(function (x) { return { screen: x.screenId, id: String(x.node.id) }; }),
      props: comp.props.map(function (x) { return { name: x.name, field: x.field, default: x.def }; }),
      paramOff: comp.paramOff || '',
      slot: comp.slots.length ? comp.slots[0].name : '',
      slotNode: comp.slots.length ? comp.slots[0].nodeId : '',
      slots: comp.slots.map(function (x) { return { name: x.name, node: x.nodeId }; }),
      files: cArtifacts.map(function (x) { return x.rel; }),
    });
  }

  var report = { files: [], backups: [] };
  var totalBytes = 0;
  for (var f = 0; f < artifacts.length; f++) totalBytes += utf8Bytes(artifacts[f].content);

  // 可维护性提示：模板自动编号的节点 → 类名只能退化为 .d-<类型>-<序号>
  var autoIds = Object.keys(out.autoIdIds || {});
  if (autoIds.length) {
    out.warnings.push(autoIds.length + ' 个节点仍是模板自动编号 id（类名退化为 .d-<类型>-<序号>，如 .d-text-3）：' +
      '给关键节点起语义 id 或 props.name（design_edit node.set / node.add 传 id），生成代码的类名会直接可读（如 .d-hero-title）');
  }

  // 产物指纹检查：若目标文件「与上次生成的内容不一致」→ 说明被人工改动过 ⇒ 默认拒绝覆盖（手写内容不该被静默冲掉）
  var allowOverwrite = argBool(args, 'overwrite', false);
  var prevManifest = loadManifest(ctx, targetDir);
  var changedFiles = [];
  var unmanagedFiles = [];
  for (var fp = 0; fp < artifacts.length; fp++) {
    var relF = artifacts[fp].rel;
    var existsF = false;
    try { existsF = ctx.fs.exists(relF); } catch (ef) { existsF = false; }
    if (!existsF) continue;
    var curF = '';
    try { curF = ctx.fs.readFile(relF); } catch (ef2) { curF = ''; }
    var recF = prevManifest.files ? prevManifest.files[relF] : null;
    if (recF && recF.hash) {
      if (recF.hash !== contentHash(curF)) changedFiles.push(relF);
    } else {
      unmanagedFiles.push(relF);   // 有文件无指纹记录：早期版本产物或人工创建 → 覆盖前告知，不阻断
    }
  }
  if (changedFiles.length && !allowOverwrite && !dryRun) {
    throw new Error('拒绝覆盖 ' + changedFiles.length + ' 个「已被人工改动」的产物（内容与上次生成不一致）：\n' +
      changedFiles.map(function (x) { return '  · ' + x; }).join('\n') +
      '\n本工具不会静默冲掉手写内容。三选一：① 把改动移走 / 换个文件名；② 用 targetDir 换落点；' +
      '③ 确实要覆盖 → 传 overwrite=true（覆盖前仍会写 .bak 备份，可回滚）。');
  }

  if (!dryRun) {
    for (var w = 0; w < artifacts.length; w++) {
      writeArtifact(ctx, artifacts[w].rel, artifacts[w].content, report);
    }
    // 产物指纹清单（先并入旧记录：单屏生成不会丢掉其它屏的指纹；键排序保证两次生成位级一致）
    var mf = { schema: SCHEMA, kind: 'design-codegen-manifest', ts: nowIso(), targetDir: targetDir, files: {} };
    for (var mk in prevManifest.files) mf.files[mk] = prevManifest.files[mk];
    for (var mi2 = 0; mi2 < artifacts.length; mi2++) {
      mf.files[artifacts[mi2].rel] = {
        hash: contentHash(artifacts[mi2].content),
        bytes: utf8Bytes(artifacts[mi2].content),
        ts: nowIso(),
      };
    }
    var sorted = {};
    var mfKeys = Object.keys(mf.files).sort();
    for (var ks = 0; ks < mfKeys.length; ks++) sorted[mfKeys[ks]] = mf.files[mfKeys[ks]];
    mf.files = sorted;
    ctx.fs.writeFile(targetDir + '/' + MANIFEST_NAME, JSON.stringify(mf, null, 2) + '\n');
    // 旁挂落地报告（供面板展示；本身也是产物，不参与 .bak 逻辑之外的统计）
    var rep = {
      schema: SCHEMA, kind: 'design-codegen', ts: nowIso(),
      project: path, tokens: str(proj.tokens, 'design.tokens.json'),
      targets: targets, targetDir: targetDir,
      screens: screens.map(function (x) { return x.id; }),
      components: compFiles,
      interactions: interactions,
      duplicates: dupCandidates,
      nearDuplicates: nearCandidates,
      propsMode: propsMode,
      files: report.files, backups: report.backups,
      changed: changedFiles, unmanaged: unmanagedFiles,
      warnings: out.warnings,
    };
    ctx.fs.writeFile(targetDir + '/codegen.report.json', JSON.stringify(rep, null, 2) + '\n');
  }

  var L = [];
  L.push((dryRun ? '（dryRun：未写盘）将生成 ' : '✓ 已生成 ') + artifacts.length + ' 个文件，共 ' + humanBytes(totalBytes) +
    '（targets: ' + targets.join(' / ') + '，屏幕 ' + screens.length + ' 个）');
  L.push('落点：' + targetDir + '/（相对主项目根；可用 targetDir 改到项目源码目录，如 src/components）');
  L.push('');
  L.push('产物清单：');
  for (var p2 = 0; p2 < artifacts.length; p2++) {
    L.push('  ' + (artifacts[p2].rel + '                                                                        ').slice(0, 52) +
      ' ' + humanBytes(utf8Bytes(artifacts[p2].content)));
  }
  if (compFiles.length) {
    L.push('');
    L.push('抽出组件（' + compFiles.length + ' 个；屏幕产物里原位已写成 <组件名 /> 引用）：');
    for (var g = 0; g < compFiles.length; g++) {
      var cf = compFiles[g];
      var where = cf.instances.map(function (x) { return x.screen + '/' + x.id; }).join('、');
      L.push('  · ' + cf.name + '  ← ' + (cf.auto ? ('自动识别 ' + cf.instances.length + ' 个同构区块')
        : ('屏幕 ' + cf.screen + ' 的节点 ' + cf.id)));
      if (cf.auto) L.push('      实例：' + where);
      if (cf.near && cf.nearVars.length) {
        L.push('      配色差异 → CSS 变量（实例内联覆盖，默认值 = 原型值）：' +
          cf.nearVars.map(function (v) {
            return v.fields.map(function (f) { return f.field + '→' + f.cssVar; }).join('/') + '（节点 ' + v.node + '）';
          }).join('；'));
      }
      if (cf.props.length) {
        var plist = [];
        for (var gp = 0; gp < cf.props.length; gp++) {
          plist.push(cf.props[gp].name + '（默认 "' + String(cf.props[gp].default).replace(/\s*\n\s*/g, ' ') + '"）');
        }
        L.push('      props：' + plist.join('、'));
      } else if (cf.paramOff === 'tooMany') {
        L.push('      props：未参数化（文案 prop 超过 ' + MAX_COMP_PROPS + ' 个上限，内容写死）');
      }
      if (cf.slot) {
        for (var ls = 0; ls < (cf.slots || []).length; ls++) {
          var sl2 = cf.slots[ls];
          L.push('      插槽 ' + (sl2.name === 'default' ? '默认（<slot />）' : ('"' + sl2.name + '"（<slot name="' + sl2.name + '" />）')) +
            ' ← 节点 ' + sl2.node + '：' +
            (sl2.name === 'default'
              ? '屏幕侧把该节点子树写进引用标签内部（React 走 children）'
              : ('屏幕侧写成 Vue `<template #' + sl2.name + '>…</template>` / React `' + sl2.name + '={<>…</>}`')));
        }
      }
      L.push('      ' + cf.files.join('\n      '));
    }
  }
  if (dupCandidates.length) {
    L.push('');
    L.push('检测到 ' + dupCandidates.length + ' 组同构重复区块（建议抽组件；传 component=auto 可自动抽取）：');
    for (var dc1 = 0; dc1 < dupCandidates.length; dc1++) {
      var cd = dupCandidates[dc1];
      var sites = [];
      for (var dm1 = 0; dm1 < cd.members.length; dm1++) sites.push(cd.members[dm1].screen + '/' + cd.members[dm1].id);
      L.push('  · ' + cd.name + '（' + cd.type + '，' + cd.size + ' 个节点）出现 ' + cd.count + ' 次：' + sites.join('、'));
    }
    L.push('  判据：结构（类型序列 + 子节点顺序）与关键样式完全一致，忽略节点 id/名称与文案（文案会提升为 props）');
  }
  if (nearCandidates.length) {
    L.push('');
    L.push('');
    L.push('另有 ' + nearCandidates.length + ' 组**近似**同构区块（结构相同、只有颜色类字段不同 —— 未自动抽取）：');
    for (var nd1 = 0; nd1 < nearCandidates.length; nd1++) {
      var nd = nearCandidates[nd1];
      var nsites = [];
      for (var nm2 = 0; nm2 < nd.members.length; nm2++) nsites.push(nd.members[nm2].screen + '/' + nd.members[nm2].id);
      L.push('  · ' + nd.name + '（' + nd.type + '，' + nd.size + ' 个节点）出现 ' + nd.count +
        ' 次：' + nsites.join('、') + ' —— 差异字段：' + nd.diffKeys.join('、'));
    }
    L.push('  为什么不自动抽：组件默认值只能有一份，配色不同的实例会被压成同一个外观（设计上本来要区分的配色就丢了）。');
    L.push('  想复用：① 把这几处配色对齐（改用同一个 token）后 component=auto 即可抽；② 或确认是有意区分 → 用 component=<节点 id> 各抽各的。');
  }
  if (interactions.length) {
    var ivc = { link: 0, emit: 0, toggle: 0, invalid: 0 };
    for (var ie = 0; ie < interactions.length; ie++) ivc[interactions[ie].verb]++;
    L.push('');
    L.push('交互（props.action，受控词汇表）：' + interactions.length + ' 处 —— link ' + ivc.link +
      ' / emit ' + ivc.emit + ' / toggle ' + ivc.toggle + (ivc.invalid ? (' / **非法 ' + ivc.invalid + '**') : ''));
    for (var i3 = 0; i3 < interactions.length && i3 < 12; i3++) {
      var it = interactions[i3];
      L.push('  · ' + it.screen + '/' + it.id + '（' + it.type + '）' + it.action + (it.error ? (' ← ' + it.error) : ''));
    }
    if (interactions.length > 12) L.push('  · …（其余 ' + (interactions.length - 12) + ' 处见 codegen.report.json）');
    L.push('  接线：link → 真 <a href>（#屏幕 → ./<屏幕>.html，接路由请自行改 href）；Vue emit → defineEmits（父级 @<名字>="…"）；');
    L.push('  React emit → on<名字> 回调 prop（占位空实现可直接替换）；HTML 静态页无脚本 → 交互只出 data-action，由宿主接线。');
  }
  if (changedFiles.length) {
    L.push('');
    L.push(dryRun
      ? ('（dryRun 预检）以下 ' + changedFiles.length + ' 个产物已被人工改动 —— 真正写盘会被**拒绝**（除非传 overwrite=true）：')
      : ('⚠ 已按 overwrite=true 覆盖 ' + changedFiles.length + ' 个「人工改动过」的产物（旧内容都在 .bak，可回滚）：'));
    for (var ch = 0; ch < changedFiles.length; ch++) L.push('  · ' + changedFiles[ch]);
  }
  if (unmanagedFiles.length) {
    L.push('');
    L.push('提示：' + unmanagedFiles.length + ' 个既有文件没有指纹记录（非本工具生成 / 早期版本产物）—— 已备份 .bak 后覆盖：');
    for (var um = 0; um < unmanagedFiles.length && um < 6; um++) L.push('  · ' + unmanagedFiles[um]);
    if (unmanagedFiles.length > 6) L.push('  · …（其余 ' + (unmanagedFiles.length - 6) + ' 个见 codegen.report.json 的 unmanaged）');
  }
  if (!dryRun) {
    L.push('');
    L.push('覆盖备份（.bak，可人工回滚）：' + (report.backups.length ? '\n  ' + report.backups.join('\n  ') : '无（均为新文件）'));
  }
  if (out.warnings.length) {
    L.push('');
    L.push('注意：');
    for (var w2 = 0; w2 < out.warnings.length; w2++) L.push('  · ' + out.warnings[w2]);
  }
  L.push('');
  L.push('怎么用：');
  L.push('  · 静态页：打开 ' + targetDir + '/html/<屏幕>.html（相对引用 ../tokens.css）；');
  L.push('  · Vue：把 ' + targetDir + '/vue/*.vue 拷进 src/components/，入口 import "' + targetDir + '/tokens.css"；');
  L.push('  · React：把 ' + targetDir + '/react/*.jsx + *.module.css 拷进 src/components/，同样引入 tokens.css。');
  if (compFiles.length) {
    L.push('  · 组件：被抽子树在屏幕里已是 <' + compFiles[0].name + ' /> 引用（Vue 屏幕自动补了 <script setup> import）；');
    L.push('    组件文件在 ' + targetDir + '/{vue,react,html}/components/ —— 连目录一起拷进项目即生效。');
    if (propsMode === 'auto') {
      L.push('  · 参数化：文案/标签/alt 已提升为 props（默认值 = 设计原值），屏幕只在**与默认值不同**时传参 ——' +
        '同构实例各自带自己的文案；想固定不变传 props: "none"。');
      L.push('    单个节点要排除/改名：design_edit node.set props={"prop":"title"}（或 false 表示不提升）。');
      L.push('    想让某节点内容由外部传入（插槽）：design_edit node.set props={"slot":true}（或 "slot":"名字"）。');
      L.push('    一个组件可以有**多个**插槽宿主（具名多插槽）：只有 1 个且没写名字 → 默认插槽（<slot /> / children）；');
      L.push('    多个则逐个起名字（如 "header" / "footer"），未命名的自动命名 slot1/slot2 并告警。');
    }
  }
  L.push('  令牌变量是唯一颜色/间距来源 —— 改 design.tokens.json 后重新生成即全站同步。');
  L.push('');
  L.push('下一步：design_verify 校验设计一致性（D2 悬空令牌 / D3 对比度 / D8 溢出），再用同样命令幂等重生成。');
  return L.join('\n');
}

function designVerify(args, exec, ctx) {
  var path = argStr(args, 'path', 'design.project.json');
  var proj = loadProject(ctx, path);
  var tokens = loadTokensFor(ctx, proj);
  var minContrast = argNum2(args, 'minContrast', 0);
  var res = verifyProject(proj, tokens, { minContrast: minContrast });
  var reportPath = argStr(args, 'report', 'design.verify.json');
  var report = {
    schema: SCHEMA,
    kind: 'design-verify',
    ts: nowIso(),
    project: path,
    tokens: str(proj.tokens, 'design.tokens.json'),
    pass: res.pass,
    stats: res.stats,
    checks: res.checks,
  };
  ctx.fs.writeFile(reportPath, JSON.stringify(report, null, 2) + '\n');
  return verifyText(res, path, reportPath);
}

// ── 工具声明 ──────────────────────────────────────────────
var TOOL_DEFS = [
  {
    name: 'design_tokens',
    description: '设计令牌（真相源 ①）的创建/查看/更新：颜色、间距、圆角、字号、字重、行高、阴影、字体族全部集中于此，语义命名（bg/fg/border/accent/muted…）。默认值取自公开标准（Tailwind 色板 + 4px 间距网格），不是随机生成的"AI 味"配色。界面节点只能引用令牌（$color.bg / $space.md）；校验器 D2 会抓悬空引用与语义错用（例如把颜色令牌当宽度）。更新时校验：颜色必须可解析、间距必须是 4px 倍数、数值组必须为非负数字。',
    usageGuide: '首次：design_tokens mode=create（生成 design.tokens.json，附带 WCAG 对比度矩阵）。改色：mode=update tokens={"color":{"accent":"#1D4ED8"}}（值传 null 表示删除）。查表：mode=show。色板请从既有品牌/设计系统取值，不要自创高饱和紫蓝渐变。',
    category: '创作',
    parameters: {
      type: 'object',
      properties: {
        path: { type: 'string', description: '令牌文件路径（默认 design.tokens.json；相对主项目根解析）' },
        mode: { type: 'string', description: 'create（新建）| show（默认，查看 + 对比度矩阵）| update（改令牌）' },
        title: { type: 'string', description: '可选（仅 create）：令牌文件标题' },
        tokens: { type: 'object', description: '仅 update：要合并的令牌 {"组":{"名":值}}（值为 null 表示删除该令牌）' },
        overwrite: { type: 'boolean', description: '可选（仅 create）：已存在时是否重建（默认 false）' },
      },
    },
  },
  {
    name: 'design_project',
    description: '界面工程（真相源 ②）的创建/查看/更新：屏幕（screen，含尺寸与背景）+ 节点树（node）。节点 13 类：容器 frame/row/col/card/list，叶子 text/button/input/badge/icon/image/divider/spacer/checkbox。节点样式一律引用令牌。内置模板 mobile（移动端任务卡）与 panel（插件面板式），可快速产出可看的设计。',
    usageGuide: '首次：design_project mode=create title=任务应用 screens=[{"id":"home","name":"首页","width":360,"height":640,"template":"mobile"}]（令牌文件不存在会自动创建）。之后用 design_edit 加/改节点，design_verify 校验，design_export format=html 出预览。',
    category: '创作',
    parameters: {
      type: 'object',
      properties: {
        path: { type: 'string', description: '工程文件路径（默认 design.project.json）' },
        mode: { type: 'string', description: 'create（新建）| show（默认，查看结构树）| update（改标题/令牌路径/产物路径）' },
        title: { type: 'string', description: '可选：标题' },
        tokens: { type: 'string', description: '可选：令牌文件路径（默认 design.tokens.json）' },
        screens: { type: 'array', description: '可选（仅 create）：屏幕定义 [{id,name,width,height,template}]，template 取 mobile/panel/none' },
        html: { type: 'string', description: '可选（仅 update）：HTML 产物路径（默认 design.html）' },
        overwrite: { type: 'boolean', description: '可选（仅 create）：已存在时是否重建（默认 false）' },
      },
    },
  },
  {
    name: 'design_edit',
    description: '向界面工程追加编辑命令（命令式 op，与面板操作同源）。支持：令牌（token.set / token.remove）、屏幕（screen.add/remove/set，可套模板）、节点（node.add/set/text/remove/duplicate/move）、导航（flow.add / flow.remove / flow.sync）、标题（project.rename）。目标选择支持 ids / id / type / name + screen 过滤（不指定目标直接报错，避免误改全部）。flow.sync 按节点 props.action=link:#屏幕 自动补齐导航边（只增不删、可反复执行；prune=true 才一并删掉无交互支撑的边）。',
    usageGuide: '示例：{"op":"node.add","screen":"home","parent":"n6","node":{"type":"text","text":"新增一行","color":"$color.muted","size":"$fontSize.sm"}} / {"op":"node.set","ids":["n7"],"props":{"weight":"$fontWeight.semibold"}} / {"op":"node.text","id":"n3","text":"本周任务"} / {"op":"token.set","group":"color","name":"accent","value":"#1D4ED8"} / {"op":"screen.add","id":"detail","name":"详情","width":360,"height":640,"template":"panel"} / {"op":"flow.add","from":"home","to":"detail","label":"点击卡片"}。交互（写在 props.action）：{"op":"node.set","ids":["btn1"],"props":{"action":"link:#detail"}}（跳转）/ {"action":"emit:save"}（交给宿主）/ {"action":"toggle"}（仅 checkbox）。改完用 design_verify 校验、design_export format=html 出预览。导航图跟不上交互（D11 报缺边）：{"op":"flow.sync"} 按 action:link 补齐导航边，幂等可重复执行。',
    category: '创作',
    parameters: {
      type: 'object',
      properties: {
        path: { type: 'string', description: '可选：工程路径（默认 design.project.json）' },
        ops: { type: 'array', description: '编辑命令数组（按顺序应用）' },
        op: { type: 'string', description: '可选：单条 op 名（与同层参数合成为一条命令）' },
        ids: { type: 'array', description: '可选：目标节点 id 数组' },
        id: { type: 'string', description: '可选：单个目标（节点 id / 屏幕 id，按 op 语义）' },
        screen: { type: 'string', description: '可选：限定屏幕（node.add 的落点屏幕 / 目标过滤）' },
        parent: { type: 'string', description: '可选（node.add）：父容器 id，默认 root（屏幕根）' },
        node: { type: 'object', description: '可选（node.add）：节点定义 {id?,type,props?,children?}' },
        props: { type: 'object', description: '可选（node.set）：要合并的属性（null 表示删除该属性）' },
        text: { type: 'string', description: '可选（node.text）：新文案' },
        group: { type: 'string', description: '可选（token.set/token.remove）：令牌组' },
        name: { type: 'string', description: '可选（token.set/token.remove / 目标过滤）：令牌名或节点 props.name' },
        value: { description: '可选（token.set）：令牌新值' },
      },
    },
  },
  {
    name: 'design_export',
    description: '导出设计产物（全部为文本，零依赖）：format=html 自包含 HTML 预览（内联样式、绝对定位，布局盒子与校验器同源 → 截图即所见）；format=css 令牌 → CSS 自定义属性（--color-accent 等）；format=mermaid 屏幕信息架构图（subgraph + 导航边；本插件只生成 mermaid 文本，渲染交给宿主）。',
    usageGuide: '出预览：design_export format=html（默认写 design.html）→ 截图核对渲染；单屏预览加 screen=home。交付前端：format=css → design.tokens.css。架构图：format=mermaid → design.flow.mmd（可在 Markdown 里渲染）。产物已存在需 overwrite=true。',
    category: '创作',
    parameters: {
      type: 'object',
      properties: {
        path: { type: 'string', description: '可选：工程路径（默认 design.project.json）' },
        format: { type: 'string', description: '产物格式：html（默认）| css | mermaid' },
        out: { type: 'string', description: '可选：输出路径（默认取工程 artifacts 或 design.html / design.tokens.css / design.flow.mmd）' },
        screen: { type: 'string', description: '可选（format=html）：只导出指定屏幕' },
        overwrite: { type: 'boolean', description: '可选：产物已存在时是否覆盖（默认 false）' },
      },
    },
  },
  {
    name: 'design_codegen',
    description: '把设计**落地成工程项目代码**（与 design_export 的「预览」不同）：从界面工程生成真实前端代码 —— HTML+CSS 静态页 / Vue 3 单文件组件（SFC）/ React 函数组件 + CSS Modules，并写入项目目录。生成的是**工程代码**而非截图式预览：flex 流式布局（与设计器布局引擎同一套规则）、语义标签与语义 class（.d-<节点 id>）、颜色/间距/字号/圆角/阴影一律引用设计令牌变量（var(--color-accent)）—— 改 design.tokens.json 重新生成即全站同步，代码里不散落裸值。也能把子树抽成**可复用组件**并**参数化**：component=<节点 id> 抽指定区块，component=auto 自动识别同构重复区块（跨屏也算，一个组件多实例复用）；抽出的组件把内容字段（文案/标签/alt）提升为 props（默认值 = 设计原值，不传即与原设计一致），屏幕引用只在**与默认值不同**时传参 —— 同构实例各自带自己的文案。props.slot 标记的节点在组件里变成插槽 —— 支持**多个具名插槽**（Vue 侧 `<slot name="header" />`、屏幕侧 `<template #header>`；React 侧具名 props `header={<></>}`），只有 1 个且未命名时是默认插槽（`<slot />` / children）。detect=true 只报告重复区块候选（不抽取）。写盘前把已存在文件自动备份为 .bak（可回滚），并维护**产物指纹清单**（codegen.manifest.json）：再次生成时若产物与上次生成的内容不一致（= 被人工改过）→ **默认拒绝覆盖**并报错（传 overwrite=true 才覆盖，覆盖前仍写 .bak），手写内容不会被静默冲掉；既有文件若没有指纹记录（早期版本产物/人工创建）则先提示、再备份覆盖；产物确定性（同工程两次生成位级一致）。交互由节点 props.action 声明（受控词汇表；设计真相源不携带脚本 —— 自由 JS 无法校验，静态产物也禁止 on*/javascript:）：link:<目标> 生成真链接 <a href>（#屏幕id → ./<屏幕>.html；/站内路径、https://、mailto:、tel: 原样）；emit:<事件名> 交给宿主 —— Vue 出 defineEmits + 内联 @click，组件与屏幕都可用（屏幕是根组件，由父级监听）、React 出 on<名字> 回调 prop（组件解构带空实现、屏幕内出可直接替换的占位函数）、HTML 静态页无脚本故只出 data-action 声明；toggle 仅 checkbox（原生交互，不产代码）。近似区块也能自动抽：component=near 把「结构一致、只有 bg/color/border/shadow 不同」的区块抽成一份组件，配色差异提升为 CSS 变量（组件内 var(--c<序号>-bg, 原型值)，实例处内联覆盖）—— 设计上本来要区分的配色不会丢；tone（语义色，同时决定前景与背景两条声明）差异不抽并说明原因。也能出 **Tailwind** 形态（targets 含 "tailwind"）：tailwind/<屏幕>.html 的 class 全部换成 Tailwind utility —— 颜色/间距/字号/圆角用**令牌变量任意值**（bg-[var(--color-surface)]、p-[var(--space-md)]，产物无裸色值），命中 Tailwind 默认 scale 时用语义 class（p-4 / gap-3 / rounded-lg / font-semibold）；与 css 版逐声明等价（同一棵代码树），映射不了的声明进 warnings 不静默丢；边界：只出 HTML 形态、不参与组件抽取。',
    usageGuide: '生成并落地：design_codegen targets=["html","vue"]（默认 html）→ 落 design-export/；要落进项目源码目录用 targetDir=src/components（默认 design-export，不碰既有源码树）。先看清单不写盘：dryRun=true。单屏：screen=home。抽组件：component=hero-card（节点 id 或 props.name，多个用逗号）→ 该子树落到 {vue,react,html}/components/<组件名>、屏幕里原位变成 <组件名 /> 引用（Vue 屏幕自动补 <script setup> import）。自动复用重复区块：先 detect=true 看候选（结构 + 关键样式同构、出现 ≥2 次），再 component=auto 一键抽成组件并让各实例复用。参数化默认开（props="auto"）：Vue 出 defineProps、React 出解构默认值，屏幕只传与默认值不同的文案；不想参数化传 props="none"。要排除某节点内容或改 prop 名：design_edit node.set props={"prop":"title"| false}；要让某节点内容由外部传入（插槽）：node.set props={"slot":true}；一个组件可标记**多个**插槽宿主，多个时逐个给名字（props={"slot":"header"}），未命名的自动命名 slot1/slot2 并告警。生成后建议跑 design_verify 校验设计一致性，再打开 html/<屏幕>.html 核对，或把 .vue/.jsx 拷进项目（务必全局引入生成的 tokens.css）。交互落地：link 已是真 <a href>；Vue 的 emit 由父级监听，React 的 on<名字> 回调（屏幕内是占位空实现）直接替换成你的处理函数；HTML 静态页的 data-action 交给宿主接线。近似复用：配色不同的同构区块交给 component=near（差异字段 → 变量名见 codegen.report.json 的 components[].nearVars）。要 Tailwind：targets=["html","tailwind"] → 额外落 tailwind/<屏幕>.html（class 全为 utility）+ tailwind/README.md（接入三步：装 Tailwind / content 覆盖该目录 / 引入 tokens.css）；tailwind 版与 css 版逐声明等价，命中默认 scale 用 p-4/gap-3/rounded-lg，否则任意值兜底，映射不了的声明进 warnings。产物被人工改动时的默认行为：拒绝覆盖（overwrite=true 强制，覆盖前写 .bak）；dryRun=true 会预检这类文件。产物指纹：生成会维护 codegen.manifest.json（每个产物的内容指纹），若某产物被人工改过，下次生成会**拒绝覆盖**并列出这些文件（三选一：移走改动 / 换 targetDir / 传 overwrite=true 强制，强制时仍写 .bak）；dryRun=true 可先预检（输出里标「（dryRun 预检）…会被拒绝」）。',
    category: '创作',
    parameters: {
      type: 'object',
      properties: {
        path: { type: 'string', description: '可选：工程路径（默认 design.project.json）' },
        targets: { type: 'array', description: '产物形态：html（默认，HTML + CSS 静态页）| vue（Vue 3 SFC，style scoped）| react（JSX + CSS Modules）| tailwind（HTML + Tailwind utility class：颜色/间距/圆角走令牌变量任意值）；可多选' },
        targetDir: { type: 'string', description: '可选：落地目录（默认 design-export，相对主项目根解析；同时写 tokens.css 与 README.md）' },
        screen: { type: 'string', description: '可选：只生成指定屏幕（默认全部屏幕）' },
        component: { type: 'array', description: '可选：把子树抽成独立组件并在屏幕产物里引用 —— 传 1..N 个节点 id / props.name（数组或逗号分隔），或传 "auto" 自动识别**同构重复区块**（结构 + 关键样式一致且出现 ≥2 次，跨屏也算；跳过屏幕根，大区块优先、被包含的碎片不再单独抽）→ 一个组件对应多个实例。组件名 PascalCase（取 props.name > 语义 id > 类型-序号），产物落 <targetDir>/{vue,react}/components/<组件名>.vue|.jsx(+.module.css) 与 html/components/<组件名>.html 片段：Vue 屏幕自动补 <script setup> import 并写 <组件名 />，React 屏幕 import 后写 <组件名 />，HTML 无组件机制故屏幕保持内联渲染（片段另出）。内容字段会被提升为 props（见 props 参数）' },
        props: { type: 'string', description: '可选：组件参数化模式 —— auto（默认）把抽出的组件里**内容字段**（text 的 text、button/badge/checkbox 的 label、input 的 value|placeholder、image 的 alt）提升为 props，默认值 = 设计原值，屏幕引用只在与原值不同时传参；none 表示内容写死（不生成 props）。单节点可用 node.set props={"prop":"名字"} 改名、props={"prop":false} 排除；props 超过 12 个时自动退化为不参数化并告警（大区块应拆小）' },
        detect: { type: 'boolean', description: '可选：只**报告**同构重复区块候选（不抽取、不改屏幕产物），供决定要不要 component=auto 抽成组件（默认 false）。同时报告「结构相同、只有颜色类字段不同」的**近似**组（附差异字段与为什么不自动抽）' },
        dryRun: { type: 'boolean', description: '可选：只列出将生成的文件、不写盘（默认 false）；同时预检「哪些产物已被人工改动」（真正写盘会被拒绝）' },
        overwrite: { type: 'boolean', description: '可选：产物与上次生成不一致（被人手改过）时是否仍覆盖（默认 false = 拒绝覆盖，保护手写内容；true 时覆盖前仍写 .bak）' },
      },
    },
  },
  {
    name: 'design_verify',
    description: '设计工程专项自检（11 项判据）：D1 结构与必填（屏幕/节点 id 唯一、类型合法、容器叶子约束、深度与规模上限）、D2 设计令牌引用有效（无悬空、语义组匹配、尺寸可解析）、D3 文本对比度（WCAG AA：正文 4.5:1、大字 3:1；按钮/徽章按其 variant/tone 的实际配色检查）、D4 间距落在 4px 网格、D5 字号来自令牌阶梯（禁止裸字号）、D6 触达区尺寸（移动 ≤480px 屏要求 44px，桌面 32px）、D7 图标白名单 + 无 emoji/符号图标（技能 emoji-icons）、D8 布局无溢出/越界（与 HTML 渲染同一份布局盒子）、D9 产物确定且自包含（两次渲染位级一致、无 script/on*/javascript:/外链）、D10 交互声明合法（props.action 受控词汇表：动词/目标/类型适配，link:#屏幕 必须真实存在，toggle 仅 checkbox）、D11 导航与交互一致（flow 导航图 ↔ 节点 props.action=link:#屏幕：声明了跳转却没有对应导航边 → FAIL；有导航边但无节点触发 → 提示）。任一 FAIL 说明设计与设计系统不一致或存在可用性缺陷，应视为构建失败。',
    usageGuide: '导出后必跑一遍；也可在 CI 里对工程文件跑（只读工程，报告旁挂 design.verify.json）。D3 抓"看不见的文字"、D6 抓"点不着的按钮"、D7 抓 emoji 当图标；D8 与预览渲染同源 —— 报告说没溢出，截图上就不会压字。可用 minContrast 放宽阈值，但建议保持 WCAG AA。D10 抓「声明了做不到的交互」（未知动词、把 input 当链接、指向不存在的屏幕、非 checkbox 标 toggle）；D11 抓「跳转声明与信息架构图不一致」（改了交互忘补导航边，架构图就与实现不符）—— design_edit op=flow.sync 一键对齐。',
    category: '创作',
    parameters: {
      type: 'object',
      properties: {
        path: { type: 'string', description: '可选：工程路径（默认 design.project.json）' },
        report: { type: 'string', description: '可选：旁挂校验报告路径（默认 design.verify.json）' },
        minContrast: { type: 'number', description: '可选：文本对比度统一阈值（默认按 WCAG：4.5 / 大字 3.0）' },
      },
    },
  },
];

var IMPLS = {
  design_tokens: designTokens,
  design_project: designProject,
  design_edit: designEdit,
  design_export: designExport,
  design_codegen: designCodegen,
  design_verify: designVerify,
};

// 双轨入口：goja 沙箱把本文件当函数体执行 → 顶层 return；Node 侧（测试/打包）走 module.exports。
var PLUGIN = {
  name: 'tool-design',
  purpose: 'UI 设计创作域工具面（纯 goja 零依赖）：设计令牌 + 界面工程双文本真相源，确定性布局引擎产出可截图 HTML 预览 / CSS 变量 / mermaid 图，11 项自检（对比度 / 网格 / 字号阶梯 / 触达区 / 图标白名单 / 布局溢出 / 交互合法性 / 导航一致）—— 可 diff、校验与渲染同源',
  inject: ['fs', 'logger'],
  apply: function (ctx) {
    for (var i = 0; i < TOOL_DEFS.length; i++) {
      (function (t) {
        ctx.tools.register({
          name: t.name,
          description: t.description,
          usageGuide: t.usageGuide,
          category: t.category,
          readOnly: !!t.readOnly,
          parameters: t.parameters,
          execute: function (args) { return IMPLS[t.name](args || {}, {}, ctx); },
        });
      })(TOOL_DEFS[i]);
    }
    try {
      if (ctx && typeof ctx.logger === 'function') {
        ctx.logger('tool-design').info('已注册 ' + TOOL_DEFS.length + ' 个工具（goja 轨 · 零依赖 · 令牌+工程双文本真相源）');
      }
    } catch (_) { /* ignore */ }
  },
};

if (typeof module !== 'undefined' && module.exports) module.exports = PLUGIN;
return PLUGIN;
