// tool-art — 矢量/图像创作工具面（纯 goja 零依赖）
//
// 设计原则（对齐《创作域支持方案》§195「真相源必须文本」）：
//   · **真相源 = 画板工程 JSON**（art.project.json：画板 / 图层 / 图元），全部可 diff；
//   · **SVG 是唯一文本产物**（零依赖生成、可回读 → V4 往返契约）；
//     PNG 光栅化属「服务端导出链」阶段（Node 桥 @resvg/resvg-js，MPL-2.0），
//     本插件不含二进制渲染 —— 面板提供浏览器侧 PNG 导出（canvas 绘制 SVG）；
//   · 图元几何一律**确定性数值**（最多 3 位小数、固定输出顺序、固定属性顺序）→ 位级可复现（V5）。
//
// 本插件是磁盘 goja 轨插件：沙箱内没有 require / Buffer / Node API，一律走 ctx 服务（ctx.fs）。
// 沙箱没有 canvas，无法测量字体 → 文本宽度用确定性估算模型（见 textWidth），仅供几何检查使用。
//
// 图元模型（扁平字段，便于 diff）：
//   { id, type, layer, z, name?, ...几何, fill, stroke, strokeWidth, opacity, transform? }
//   · 几何字段按类型取用：rect(x,y,w,h) circle(cx,cy,r) ellipse(cx,cy,rx,ry)
//     line(x1,y1,x2,y2) polyline/polygon(points[[x,y],…]) path(d) text(x,y,text,fontSize,fontWeight,anchor)
//   · transform 用 2D 仿射矩阵 [a,b,c,d,e,f]（仅非恒等时写入；导出为 matrix(...)，
//     与 SVG 语义完全等价 → 导入任意 transform 列表都能无损往返）

// ── 常量 ───────────────────────────────────────────────────
var SCHEMA = 'paircode.art/1';

var SHAPE_TYPES = ['rect', 'circle', 'ellipse', 'line', 'polyline', 'polygon', 'path', 'text'];

// 默认色板 —— 全部取自 Tailwind CSS 官方色板（公开标准值，非随机生成）：
// 中性灰阶 6 档 + 语义色 4 档（info / success / warning / error）。
// 刻意避开"AI 味"的高饱和紫蓝渐变；需要的色请从既有品牌/设计系统取值。
var PALETTE_DEFAULT = [
  '#FFFFFF', // White
  '#F9FAFB', // Gray 50
  '#E5E7EB', // Gray 200
  '#9CA3AF', // Gray 400
  '#4B5563', // Gray 600
  '#111827', // Gray 900
  '#2563EB', // Blue 600    （info / 主色）
  '#059669', // Emerald 600 （success）
  '#F59E0B', // Amber 500   （warning）
  '#EF4444', // Red 500     （error）
];

// CSS 命名色子集（HTML4 基础色 + 常用）：解析用，避免全表 148 项膨胀
var NAMED_COLORS = {
  black: '#000000', silver: '#C0C0C0', gray: '#808080', grey: '#808080', white: '#FFFFFF',
  maroon: '#800000', red: '#FF0000', purple: '#800080', fuchsia: '#FF00FF', magenta: '#FF00FF',
  green: '#008000', lime: '#00FF00', olive: '#808000', yellow: '#FFFF00', navy: '#000080',
  blue: '#0000FF', teal: '#008080', aqua: '#00FFFF', cyan: '#00FFFF', orange: '#FFA500',
};

var LIMIT_CANVAS = 8192;   // 画板最大边长（覆盖 4K+ 素材，防误建超大画布）
var LIMIT_SHAPES = 5000;   // 图元数量上限（超出视为工程异常）
var EPS = 1e-6;

// ── 基础工具 ───────────────────────────────────────────────
function isNum(v) { return typeof v === 'number' && isFinite(v); }
function isInt(v) { return isNum(v) && Math.floor(v) === v; }
function clamp(v, lo, hi) { return v < lo ? lo : (v > hi ? hi : v); }
function nowIso() { return new Date().toISOString(); }
function num(v, def) {
  var n = typeof v === 'number' ? v : parseFloat(v);
  return isNum(n) ? n : def;
}

// 数字 → 字符串：最多 3 位小数、去掉尾随 0（保证 SVG 产物位级可复现）
function fmtNum(n) {
  if (!isNum(n)) return '0';
  return String(Math.round(n * 1000) / 1000);
}

function xmlEscape(s) {
  return String(s === undefined || s === null ? '' : s)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&apos;');
}

function xmlUnescape(s) {
  return String(s === undefined || s === null ? '' : s)
    .replace(/&lt;/g, '<')
    .replace(/&gt;/g, '>')
    .replace(/&quot;/g, '"')
    .replace(/&apos;/g, "'")
    .replace(/&#(\d+);/g, function (_, d) { return String.fromCharCode(parseInt(d, 10)); })
    .replace(/&amp;/g, '&');
}

// 取对象自有键（固定顺序：插入序 —— 由构造顺序决定，保证导出可复现）
function keysOf(o) {
  var out = [];
  for (var k in o) if (Object.prototype.hasOwnProperty.call(o, k)) out.push(k);
  return out;
}

// ── 颜色：解析 / 序列化 / WCAG 对比度 ──────────────────────
// 返回 { r, g, b (0..255), a (0..1), none: bool, current: bool }；不可解析返回 null
function parseColor(input) {
  if (input === undefined || input === null) return null;
  var v = String(input).trim().toLowerCase();
  if (v === '') return null;
  if (v === 'none') return { r: 0, g: 0, b: 0, a: 0, none: true, current: false };
  if (v === 'transparent') return { r: 0, g: 0, b: 0, a: 0, none: false, current: false };
  if (v === 'currentcolor') return { r: 0, g: 0, b: 0, a: 1, none: false, current: true };
  if (v.charAt(0) === '#') {
    var hex = v.slice(1);
    if (!/^[0-9a-f]+$/.test(hex)) return null;
    if (hex.length === 3 || hex.length === 4) {
      var r3 = parseInt(hex.charAt(0) + hex.charAt(0), 16);
      var g3 = parseInt(hex.charAt(1) + hex.charAt(1), 16);
      var b3 = parseInt(hex.charAt(2) + hex.charAt(2), 16);
      var a3 = hex.length === 4 ? parseInt(hex.charAt(3) + hex.charAt(3), 16) / 255 : 1;
      return { r: r3, g: g3, b: b3, a: a3, none: false, current: false };
    }
    if (hex.length === 6 || hex.length === 8) {
      var r6 = parseInt(hex.slice(0, 2), 16);
      var g6 = parseInt(hex.slice(2, 4), 16);
      var b6 = parseInt(hex.slice(4, 6), 16);
      var a6 = hex.length === 8 ? parseInt(hex.slice(6, 8), 16) / 255 : 1;
      return { r: r6, g: g6, b: b6, a: a6, none: false, current: false };
    }
    return null;
  }
  var m = /^rgba?\(([^)]+)\)$/.exec(v);
  if (m) {
    var parts = m[1].split(/[\s,\/]+/).filter(function (s) { return s !== ''; });
    if (parts.length < 3) return null;
    var chan = function (s) {
      if (s.charAt(s.length - 1) === '%') return clamp(parseFloat(s) / 100 * 255, 0, 255);
      return clamp(parseFloat(s), 0, 255);
    };
    var alpha = 1;
    if (parts.length >= 4) {
      alpha = parts[3].charAt(parts[3].length - 1) === '%'
        ? clamp(parseFloat(parts[3]) / 100, 0, 1)
        : clamp(parseFloat(parts[3]), 0, 1);
    }
    var cc = [chan(parts[0]), chan(parts[1]), chan(parts[2])];
    if (!isNum(cc[0]) || !isNum(cc[1]) || !isNum(cc[2]) || !isNum(alpha)) return null;
    return { r: cc[0], g: cc[1], b: cc[2], a: alpha, none: false, current: false };
  }
  if (Object.prototype.hasOwnProperty.call(NAMED_COLORS, v)) return parseColor(NAMED_COLORS[v]);
  return null;
}

function colorToCss(c) {
  if (!c) return 'none';
  if (c.none) return 'none';
  if (c.current) return 'currentColor';
  if (c.a >= 1 - EPS) return 'rgb(' + Math.round(c.r) + ',' + Math.round(c.g) + ',' + Math.round(c.b) + ')';
  return 'rgba(' + Math.round(c.r) + ',' + Math.round(c.g) + ',' + Math.round(c.b) + ',' + fmtNum(c.a) + ')';
}

// WCAG 2.1 相对亮度
function relLuminance(c) {
  var f = function (v) {
    var x = v / 255;
    return x <= 0.03928 ? x / 12.92 : Math.pow((x + 0.055) / 1.055, 2.4);
  };
  return 0.2126 * f(c.r) + 0.7152 * f(c.g) + 0.0722 * f(c.b);
}

// WCAG 2.1 对比度（1..21）
function contrastRatio(fg, bg) {
  var a = relLuminance(fg);
  var b = relLuminance(bg);
  var hi = a > b ? a : b;
  var lo = a > b ? b : a;
  return (hi + 0.05) / (lo + 0.05);
}

// 半透明前景压在背景上的实际视觉色（对比度必须按合成后的颜色算）
function blendOver(top, bottom) {
  if (!top) return bottom;
  if (top.a >= 1 - EPS) return top;
  var a = clamp(top.a, 0, 1);
  return {
    r: top.r * a + bottom.r * (1 - a),
    g: top.g * a + bottom.g * (1 - a),
    b: top.b * a + bottom.b * (1 - a),
    a: 1, none: false, current: false,
  };
}

// ── 2D 仿射矩阵 [a,b,c,d,e,f] ─────────────────────────────
var M_ID = [1, 0, 0, 1, 0, 0];

function mMul(m, n) { // 结果 = m·n（点先受 n 作用，再受 m 作用）
  return [
    m[0] * n[0] + m[2] * n[1],
    m[1] * n[0] + m[3] * n[1],
    m[0] * n[2] + m[2] * n[3],
    m[1] * n[2] + m[3] * n[3],
    m[0] * n[4] + m[2] * n[5] + m[4],
    m[1] * n[4] + m[3] * n[5] + m[5],
  ];
}

function mApply(m, x, y) {
  return { x: m[0] * x + m[2] * y + m[4], y: m[1] * x + m[3] * y + m[5] };
}

function mIsIdentity(m) {
  if (!m) return true;
  for (var i = 0; i < 6; i++) {
    var expect = (i === 0 || i === 3) ? 1 : 0;
    if (Math.abs(num(m[i], expect) - expect) > EPS) return false;
  }
  return true;
}

function mNorm(m) { // 归一化：恒等 → null（工程里不写冗余字段）
  if (!m) return null;
  var arr = [];
  for (var i = 0; i < 6; i++) arr.push(num(m[i], (i === 0 || i === 3) ? 1 : 0));
  return mIsIdentity(arr) ? null : arr;
}

function mTranslate(tx, ty) { return [1, 0, 0, 1, tx, ty]; }

function mScaleAbout(sx, sy, cx, cy) {
  return mMul(mMul(mTranslate(cx, cy), [sx, 0, 0, sy, 0, 0]), mTranslate(-cx, -cy));
}

function mRotateAbout(deg, cx, cy) {
  var r = deg * Math.PI / 180;
  var c = Math.cos(r);
  var s = Math.sin(r);
  return mMul(mMul(mTranslate(cx, cy), [c, s, -s, c, 0, 0]), mTranslate(-cx, -cy));
}

// SVG transform 列表 → 矩阵（translate / scale / rotate / matrix / skewX / skewY）
function mParseTransform(str) {
  var m = M_ID.slice();
  if (!str) return m;
  var re = /([a-zA-Z]+)\s*\(([^)]*)\)/g;
  var hit;
  while ((hit = re.exec(String(str))) !== null) {
    var fn = hit[1].toLowerCase();
    var nums = [];
    var raw = hit[2].split(/[\s,]+/);
    for (var i = 0; i < raw.length; i++) {
      if (raw[i] === '') continue;
      var v = parseFloat(raw[i]);
      if (isNum(v)) nums.push(v);
    }
    var t = null;
    if (fn === 'translate') {
      t = mTranslate(num(nums[0], 0), nums.length > 1 ? num(nums[1], 0) : 0);
    } else if (fn === 'scale') {
      var sx = nums.length ? num(nums[0], 1) : 1;
      t = [sx, 0, 0, nums.length > 1 ? num(nums[1], sx) : sx, 0, 0];
    } else if (fn === 'rotate') {
      var r = num(nums[0], 0) * Math.PI / 180;
      var cs = Math.cos(r);
      var sn = Math.sin(r);
      var rm = [cs, sn, -sn, cs, 0, 0];
      if (nums.length >= 3) {
        t = mMul(mMul(mTranslate(num(nums[1], 0), num(nums[2], 0)), rm), mTranslate(-num(nums[1], 0), -num(nums[2], 0)));
      } else {
        t = rm;
      }
    } else if (fn === 'matrix' && nums.length >= 6) {
      t = [nums[0], nums[1], nums[2], nums[3], nums[4], nums[5]];
    } else if (fn === 'skewx') {
      t = [1, 0, Math.tan(num(nums[0], 0) * Math.PI / 180), 1, 0, 0];
    } else if (fn === 'skewy') {
      t = [1, Math.tan(num(nums[0], 0) * Math.PI / 180), 0, 1, 0, 0];
    }
    if (t) m = mMul(m, t);
  }
  return m;
}

function mToSvgAttr(m) {
  return 'matrix(' + fmtNum(m[0]) + ' ' + fmtNum(m[1]) + ' ' + fmtNum(m[2]) + ' ' +
    fmtNum(m[3]) + ' ' + fmtNum(m[4]) + ' ' + fmtNum(m[5]) + ')';
}

// 矩阵是否只含平移（用于圆/椭圆保持"可参数化"的判断）
function mIsTranslateOnly(m) {
  return Math.abs(m[0] - 1) < EPS && Math.abs(m[1]) < EPS && Math.abs(m[2]) < EPS && Math.abs(m[3] - 1) < EPS;
}

// ── 几何：包围盒 ───────────────────────────────────────────
// 文本宽度估算（沙箱无 canvas）：确定性模型 —— CJK/全角 = 1.0 em，其余 = 0.55 em，粗体 ×1.05。
// 不追求与浏览器字形完全一致，只服务几何检查（视口越界 / 对齐 / 分布）；
// 面板预览由浏览器渲染真实字形，两者不会互相污染。
function textWidth(text, fontSize, fontWeight) {
  var s = String(text === undefined || text === null ? '' : text);
  var units = 0;
  for (var i = 0; i < s.length; i++) {
    var c = s.charCodeAt(i);
    if (c >= 0xd800 && c <= 0xdbff && i + 1 < s.length) i++; // 代理对（emoji 等）算一次宽字符
    units += c > 0x2e80 ? 1 : 0.55;
  }
  var fw = String(fontWeight === undefined || fontWeight === null ? '400' : fontWeight);
  var bold = fw === 'bold' || fw === 'bolder' || num(fw, 400) >= 600;
  return units * num(fontSize, 16) * (bold ? 1.05 : 1);
}

// path d 的保守包围盒：收集所有绝对坐标点（含贝塞尔控制点）→ AABB
// 说明：控制点会让包围盒略大于真实曲线（保守方向：宁可报"可能越界"，不漏报）
function pathPoints(d) {
  var pts = [];
  if (!d) return pts;
  var tokens = String(d).match(/[a-zA-Z]|-?\d*\.?\d+(?:e[-+]?\d+)?/g) || [];
  var cx = 0;
  var cy = 0;
  var sx = 0;
  var sy = 0;
  var cmd = '';
  var i = 0;
  while (i < tokens.length) {
    var tk = tokens[i];
    if (/^[a-zA-Z]$/.test(tk)) {
      cmd = tk;
      i++;
      if (cmd === 'Z' || cmd === 'z') { cx = sx; cy = sy; continue; }
      continue;
    }
    var rel = cmd === cmd.toLowerCase();
    var take = function (n) {
      var out = [];
      for (var k = 0; k < n; k++) {
        var v = parseFloat(tokens[i + k]);
        out.push(isNum(v) ? v : 0);
      }
      i += n;
      return out;
    };
    var up = cmd.toUpperCase();
    if (up === 'M' || up === 'L' || up === 'T') {
      var p = take(2);
      cx = rel ? cx + p[0] : p[0];
      cy = rel ? cy + p[1] : p[1];
      if (up === 'M') { sx = cx; sy = cy; cmd = rel ? 'l' : 'L'; }
      pts.push([cx, cy]);
    } else if (up === 'H') {
      var hx = take(1)[0];
      cx = rel ? cx + hx : hx;
      pts.push([cx, cy]);
    } else if (up === 'V') {
      var vy = take(1)[0];
      cy = rel ? cy + vy : vy;
      pts.push([cx, cy]);
    } else if (up === 'C') {
      var c1 = take(2);
      var c2 = take(2);
      var c3 = take(2);
      var ax1 = rel ? cx + c1[0] : c1[0];
      var ay1 = rel ? cy + c1[1] : c1[1];
      var ax2 = rel ? cx + c2[0] : c2[0];
      var ay2 = rel ? cy + c2[1] : c2[1];
      var ax3 = rel ? cx + c3[0] : c3[0];
      var ay3 = rel ? cy + c3[1] : c3[1];
      pts.push([ax1, ay1], [ax2, ay2], [ax3, ay3]);
      cx = ax3; cy = ay3;
    } else if (up === 'S' || up === 'Q') {
      var q1 = take(2);
      var q2 = take(2);
      var qx1 = rel ? cx + q1[0] : q1[0];
      var qy1 = rel ? cy + q1[1] : q1[1];
      var qx2 = rel ? cx + q2[0] : q2[0];
      var qy2 = rel ? cy + q2[1] : q2[1];
      pts.push([qx1, qy1], [qx2, qy2]);
      cx = qx2; cy = qy2;
    } else if (up === 'A') {
      var ap = take(7);
      cx = rel ? cx + ap[5] : ap[5];
      cy = rel ? cy + ap[6] : ap[6];
      pts.push([cx, cy]);
    } else {
      i++; // 未知记号：跳过，避免死循环
    }
  }
  return pts;
}

function pathBBox(d) {
  var pts = pathPoints(d);
  if (!pts.length) return { x: 0, y: 0, w: 0, h: 0 };
  var minX = Infinity;
  var minY = Infinity;
  var maxX = -Infinity;
  var maxY = -Infinity;
  for (var i = 0; i < pts.length; i++) {
    if (pts[i][0] < minX) minX = pts[i][0];
    if (pts[i][0] > maxX) maxX = pts[i][0];
    if (pts[i][1] < minY) minY = pts[i][1];
    if (pts[i][1] > maxY) maxY = pts[i][1];
  }
  return { x: minX, y: minY, w: maxX - minX, h: maxY - minY };
}

// 图元本地包围盒（未应用 transform）
function localBBox(sh) {
  var t = sh.type;
  if (t === 'rect') return { x: num(sh.x, 0), y: num(sh.y, 0), w: num(sh.w, 0), h: num(sh.h, 0) };
  if (t === 'circle') {
    var r = num(sh.r, 0);
    return { x: num(sh.cx, 0) - r, y: num(sh.cy, 0) - r, w: 2 * r, h: 2 * r };
  }
  if (t === 'ellipse') {
    var rx = num(sh.rx, 0);
    var ry = num(sh.ry, 0);
    return { x: num(sh.cx, 0) - rx, y: num(sh.cy, 0) - ry, w: 2 * rx, h: 2 * ry };
  }
  if (t === 'line') {
    var x1 = num(sh.x1, 0);
    var y1 = num(sh.y1, 0);
    var x2 = num(sh.x2, 0);
    var y2 = num(sh.y2, 0);
    return { x: Math.min(x1, x2), y: Math.min(y1, y2), w: Math.abs(x2 - x1), h: Math.abs(y2 - y1) };
  }
  if (t === 'polyline' || t === 'polygon') {
    var pts = sh.points || [];
    if (!pts.length) return { x: 0, y: 0, w: 0, h: 0 };
    var minX = Infinity;
    var minY = Infinity;
    var maxX = -Infinity;
    var maxY = -Infinity;
    for (var i = 0; i < pts.length; i++) {
      var px = num(pts[i][0], 0);
      var py = num(pts[i][1], 0);
      if (px < minX) minX = px;
      if (px > maxX) maxX = px;
      if (py < minY) minY = py;
      if (py > maxY) maxY = py;
    }
    return { x: minX, y: minY, w: maxX - minX, h: maxY - minY };
  }
  if (t === 'path') return pathBBox(sh.d);
  if (t === 'text') {
    var fs = num(sh.fontSize, 16);
    var w = textWidth(sh.text, fs, sh.fontWeight);
    var anchor = sh.anchor || 'start';
    var x0 = anchor === 'middle' ? num(sh.x, 0) - w / 2 : (anchor === 'end' ? num(sh.x, 0) - w : num(sh.x, 0));
    // 基线近似：上升 0.8em、下降 0.2em → 覆盖字形实际占位
    return { x: x0, y: num(sh.y, 0) - fs * 0.8, w: w, h: fs };
  }
  return { x: 0, y: 0, w: 0, h: 0 };
}

// 采样点：四角足够（path 已含控制点、保守）；圆/椭圆用 16 点圆周采样（旋转后 AABB 才准确）
function bboxSamples(sh, b) {
  var t = sh.type;
  if (t === 'circle' || t === 'ellipse') {
    var rx = t === 'circle' ? num(sh.r, 0) : num(sh.rx, 0);
    var ry = t === 'circle' ? num(sh.r, 0) : num(sh.ry, 0);
    var out = [];
    for (var i = 0; i < 16; i++) {
      var a = Math.PI * 2 * i / 16;
      out.push([num(sh.cx, 0) + rx * Math.cos(a), num(sh.cy, 0) + ry * Math.sin(a)]);
    }
    return out;
  }
  return [[b.x, b.y], [b.x + b.w, b.y], [b.x + b.w, b.y + b.h], [b.x, b.y + b.h]];
}

// 应用 transform 后的轴对齐包围盒
function shapeBBox(sh) {
  var b = localBBox(sh);
  var m = sh.transform;
  if (!m || mIsIdentity(m)) return b;
  var pts = bboxSamples(sh, b);
  var minX = Infinity;
  var minY = Infinity;
  var maxX = -Infinity;
  var maxY = -Infinity;
  for (var i = 0; i < pts.length; i++) {
    var p = mApply(m, pts[i][0], pts[i][1]);
    if (p.x < minX) minX = p.x;
    if (p.x > maxX) maxX = p.x;
    if (p.y < minY) minY = p.y;
    if (p.y > maxY) maxY = p.y;
  }
  return { x: minX, y: minY, w: maxX - minX, h: maxY - minY };
}

function bboxUnion(a, b) {
  if (!a) return { x: b.x, y: b.y, w: b.w, h: b.h };
  var x = Math.min(a.x, b.x);
  var y = Math.min(a.y, b.y);
  var x2 = Math.max(a.x + a.w, b.x + b.w);
  var y2 = Math.max(a.y + a.h, b.y + b.h);
  return { x: x, y: y, w: x2 - x, h: y2 - y };
}

function bboxIntersectArea(a, b) {
  var w = Math.min(a.x + a.w, b.x + b.w) - Math.max(a.x, b.x);
  var h = Math.min(a.y + a.h, b.y + b.h) - Math.max(a.y, b.y);
  if (w <= 0 || h <= 0) return 0;
  return w * h;
}

// 是否"有重叠"（含退化为线的情形：面积可为 0，但区间相交即算可见）
// ★ 不能用面积判断：水平/垂直线的包围盒高度/宽度为 0，面积恒为 0，会被误判为"完全在画板外"。
function bboxOverlaps(a, b) {
  return Math.min(a.x + a.w, b.x + b.w) >= Math.max(a.x, b.x) - EPS &&
    Math.min(a.y + a.h, b.y + b.h) >= Math.max(a.y, b.y) - EPS;
}

// b 是否完全包含 a（含 EPS 容差）
function bboxContains(b, a, tol) {
  var t = num(tol, 0);
  return a.x >= b.x - t && a.y >= b.y - t &&
    a.x + a.w <= b.x + b.w + t && a.y + a.h <= b.y + b.h + t;
}

function bboxArea(b) { return Math.max(0, b.w) * Math.max(0, b.h); }

// ── SVG 生成（唯一文本产物） ───────────────────────────────
var SVG_NS = 'http://www.w3.org/2000/svg';

function attrsToStr(pairs) {
  var out = '';
  for (var i = 0; i < pairs.length; i++) {
    if (pairs[i][1] === undefined || pairs[i][1] === null) continue;
    out += ' ' + pairs[i][0] + '="' + pairs[i][1] + '"';
  }
  return out;
}

// 颜色值判定：'none'（不画）/ 'color'（可解析的实色）
function normColorOrRaw(v) {
  if (v === undefined || v === null) return 'none';
  var c = parseColor(v);
  if (!c) return String(v);
  return c.none ? 'none' : (c.a <= EPS ? 'none' : 'color');
}

// 归一化 path d：折叠多余空白（让同一几何的 d 串稳定，服务 V5 确定性）
function normPathD(d) {
  return String(d === undefined || d === null ? '' : d).replace(/\s+/g, ' ').trim();
}

// 图元 → SVG 元素文本。
// ★ 属性顺序固定（id → data-name → transform → 几何 → fill/stroke/stroke-width/opacity）：
//   这是 V5「两次导出位级一致」的前提，改动顺序会让 V5 失败。
function shapeToSvg(sh, pad) {
  var t = sh.type;
  var p = pad || '    ';
  var a = [['id', xmlEscape(sh.id)]];
  if (sh.name) a.push(['data-name', xmlEscape(sh.name)]);
  if (sh.transform && !mIsIdentity(sh.transform)) a.push(['transform', mToSvgAttr(sh.transform)]);

  var geom = [];
  if (t === 'rect') {
    geom.push(['x', fmtNum(num(sh.x, 0))], ['y', fmtNum(num(sh.y, 0))],
      ['width', fmtNum(Math.max(0, num(sh.w, 0)))], ['height', fmtNum(Math.max(0, num(sh.h, 0)))]);
    if (num(sh.rx, 0) > 0) geom.push(['rx', fmtNum(num(sh.rx, 0))]);
  } else if (t === 'circle') {
    geom.push(['cx', fmtNum(num(sh.cx, 0))], ['cy', fmtNum(num(sh.cy, 0))],
      ['r', fmtNum(Math.max(0, num(sh.r, 0)))]);
  } else if (t === 'ellipse') {
    geom.push(['cx', fmtNum(num(sh.cx, 0))], ['cy', fmtNum(num(sh.cy, 0))],
      ['rx', fmtNum(Math.max(0, num(sh.rx, 0)))], ['ry', fmtNum(Math.max(0, num(sh.ry, 0)))]);
  } else if (t === 'line') {
    geom.push(['x1', fmtNum(num(sh.x1, 0))], ['y1', fmtNum(num(sh.y1, 0))],
      ['x2', fmtNum(num(sh.x2, 0))], ['y2', fmtNum(num(sh.y2, 0))]);
  } else if (t === 'polyline' || t === 'polygon') {
    var pts = sh.points || [];
    var parts = [];
    for (var i = 0; i < pts.length; i++) {
      parts.push(fmtNum(num(pts[i][0], 0)) + ',' + fmtNum(num(pts[i][1], 0)));
    }
    geom.push(['points', parts.join(' ')]);
  } else if (t === 'path') {
    geom.push(['d', xmlEscape(normPathD(sh.d))]);
  } else if (t === 'text') {
    geom.push(['x', fmtNum(num(sh.x, 0))], ['y', fmtNum(num(sh.y, 0))]);
    geom.push(['font-size', fmtNum(num(sh.fontSize, 16))]);
    if (sh.fontWeight !== undefined && sh.fontWeight !== null && String(sh.fontWeight) !== '') {
      geom.push(['font-weight', xmlEscape(sh.fontWeight)]);
    }
    if (sh.anchor && sh.anchor !== 'start') geom.push(['text-anchor', xmlEscape(sh.anchor)]);
  }
  a = a.concat(geom);

  // line 无填充语义；其余图元总是显式写 fill/stroke（明确性优先，且保证往返一致）
  if (t !== 'line') {
    a.push(['fill', xmlEscape(sh.fill === undefined || sh.fill === null ? 'none' : sh.fill)]);
  }
  a.push(['stroke', xmlEscape(sh.stroke === undefined || sh.stroke === null ? 'none' : sh.stroke)]);
  if (num(sh.strokeWidth, 0) > 0 && normColorOrRaw(sh.stroke) === 'color') {
    a.push(['stroke-width', fmtNum(num(sh.strokeWidth, 1))]);
  }
  if (num(sh.opacity, 1) < 1 - EPS) a.push(['opacity', fmtNum(clamp(num(sh.opacity, 1), 0, 1))]);

  if (t === 'text') {
    return p + '<text' + attrsToStr(a) + '>' + xmlEscape(sh.text || '') + '</text>';
  }
  return p + '<' + t + attrsToStr(a) + '/>';
}

// 图元是否归入某图层（孤儿图元——layer 指向不存在的层——统一并入第一个图层，避免产物丢内容）
function wantsLayer(proj, shapeLayer, layerId) {
  var layers = proj.layers || [];
  if (!layers.length) return layerId === undefined;
  var first = layers[0].id;
  var known = false;
  for (var i = 0; i < layers.length; i++) if (layers[i].id === shapeLayer) known = true;
  var effective = known ? shapeLayer : first;
  return effective === layerId;
}

// 图层内按 z 排序（z 相同按添加顺序 → 稳定）
function shapesOfLayerInZOrder(proj, layerId) {
  var list = [];
  var all = proj.shapes || [];
  for (var i = 0; i < all.length; i++) {
    if (!wantsLayer(proj, all[i].layer, layerId)) continue;
    list.push({ sh: all[i], idx: i });
  }
  list.sort(function (a, b) {
    var za = num(a.sh.z, 0);
    var zb = num(b.sh.z, 0);
    if (za !== zb) return za - zb;
    return a.idx - b.idx;
  });
  var out = [];
  for (var k = 0; k < list.length; k++) out.push(list[k].sh);
  return out;
}

function projectToSvg(proj) {
  var cv = proj.canvas || {};
  var w = num(cv.width, 800);
  var h = num(cv.height, 600);
  var vb = cv.viewBox || ('0 0 ' + fmtNum(w) + ' ' + fmtNum(h));
  var title = (proj.meta && proj.meta.title) || '未命名画板';
  var lines = [];
  lines.push('<?xml version="1.0" encoding="UTF-8"?>');
  lines.push('<svg xmlns="' + SVG_NS + '" width="' + fmtNum(w) + '" height="' + fmtNum(h) +
    '" viewBox="' + xmlEscape(vb) + '" role="img" aria-label="' + xmlEscape(title) + '">');
  lines.push('  <title>' + xmlEscape(title) + '</title>');
  lines.push('  <desc>' + xmlEscape('由 tool-art 生成；真相源 = art.project.json（改 SVG 不回流，请改工程）') + '</desc>');

  var bgRaw = cv.background === undefined || cv.background === null ? 'none' : cv.background;
  var bg = parseColor(bgRaw);
  if (bg && !bg.none && bg.a > EPS) {
    lines.push('  <rect id="bg" x="0" y="0" width="' + fmtNum(w) + '" height="' + fmtNum(h) +
      '" fill="' + xmlEscape(bgRaw) + '"/>');
  }

  var layers = proj.layers || [];
  for (var i = 0; i < layers.length; i++) {
    var ly = layers[i];
    var ga = [['id', xmlEscape(ly.id)]];
    if (ly.name) ga.push(['data-name', xmlEscape(ly.name)]);
    if (ly.visible === false) ga.push(['display', 'none']);
    var kids = shapesOfLayerInZOrder(proj, ly.id);
    if (!kids.length) {
      lines.push('  <g' + attrsToStr(ga) + '/>');
      continue;
    }
    lines.push('  <g' + attrsToStr(ga) + '>');
    for (var k = 0; k < kids.length; k++) lines.push(shapeToSvg(kids[k], '    '));
    lines.push('  </g>');
  }
  lines.push('</svg>');
  return lines.join('\n') + '\n';
}

// ── SVG 解析（导入） ───────────────────────────────────────
// 沙箱无 DOMParser → 自实现极简 XML 扫描器（元素树 + 属性 + 文本节点）。
// 只处理 SVG 需要的部分：声明/注释/DOCTYPE 剥离、单双引号属性、自闭合标签、实体反转义。
function parseXml(text) {
  var root = { tag: '#root', attrs: {}, children: [], text: '' };
  var stack = [root];
  var s = String(text === undefined || text === null ? '' : text);
  s = s.replace(/<\?[\s\S]*?\?>/g, '').replace(/<!--[\s\S]*?-->/g, '').replace(/<!DOCTYPE[^>]*>/gi, '');
  var pos = 0;
  while (pos < s.length) {
    var lt = s.indexOf('<', pos);
    if (lt < 0) {
      var tail = s.slice(pos);
      if (tail.trim()) stack[stack.length - 1].text += tail;
      break;
    }
    if (lt > pos) {
      var chunk = s.slice(pos, lt);
      if (chunk.trim()) stack[stack.length - 1].text += chunk;
    }
    var gt = s.indexOf('>', lt);
    if (gt < 0) break;
    var rawTag = s.slice(lt + 1, gt);
    pos = gt + 1;
    if (rawTag.charAt(0) === '/') {
      var closeName = rawTag.slice(1).trim().split(/\s/)[0];
      for (var i = stack.length - 1; i > 0; i--) {
        if (stack[i].tag === closeName) { stack.length = i; break; }
      }
      continue;
    }
    var selfClose = /\/\s*$/.test(rawTag);
    var body = selfClose ? rawTag.replace(/\/\s*$/, '') : rawTag;
    var m = /^([a-zA-Z_][\w:.-]*)/.exec(body);
    if (!m) continue;
    var attrs = {};
    var attrRe = /([\w:.-]+)\s*=\s*(?:"([^"]*)"|'([^']*)')/g;
    var am;
    while ((am = attrRe.exec(body)) !== null) {
      attrs[am[1]] = xmlUnescape(am[2] !== undefined ? am[2] : am[3]);
    }
    var node = { tag: m[1], attrs: attrs, children: [], text: '' };
    stack[stack.length - 1].children.push(node);
    if (!selfClose) stack.push(node);
  }
  return root;
}

// 展现属性 + style 内联声明（style 优先）
var PRESENTATION_ATTRS = ['fill', 'stroke', 'stroke-width', 'opacity', 'fill-opacity', 'stroke-opacity',
  'font-size', 'font-weight', 'text-anchor', 'font-family', 'display'];

function styleMap(node) {
  var map = {};
  for (var i = 0; i < PRESENTATION_ATTRS.length; i++) {
    var k = PRESENTATION_ATTRS[i];
    if (node.attrs[k] !== undefined) map[k] = node.attrs[k];
  }
  if (node.attrs.style) {
    var decls = String(node.attrs.style).split(';');
    for (var d = 0; d < decls.length; d++) {
      var idx = decls[d].indexOf(':');
      if (idx <= 0) continue;
      var key = decls[d].slice(0, idx).trim().toLowerCase();
      var val = decls[d].slice(idx + 1).trim();
      if (key && val) map[key] = val;
    }
  }
  return map;
}

function attrNum(attrs, name, def) {
  if (!attrs || attrs[name] === undefined || attrs[name] === null) return def;
  var v = parseFloat(attrs[name]);
  return isNum(v) ? v : def;
}

function pointsAttrToArr(str) {
  var raw = String(str || '').split(/[\s,]+/);
  var vals = [];
  for (var i = 0; i < raw.length; i++) {
    if (raw[i] === '') continue;
    var v = parseFloat(raw[i]);
    if (isNum(v)) vals.push(v);
  }
  var pts = [];
  for (var k = 0; k + 1 < vals.length; k += 2) pts.push([vals[k], vals[k + 1]]);
  return pts;
}

// 颜色值净化：剥离脚本注入面（javascript: / url(...)）；非法值原样保留交由校验器报错
function sanitizeColorValue(v, warn, where) {
  var s = String(v === undefined || v === null ? '' : v).trim();
  var low = s.toLowerCase();
  if (low.indexOf('javascript:') >= 0 || low.indexOf('url(') >= 0) {
    if (warn) warn.push('已丢弃不安全的 ' + where + ' 值: ' + s);
    return 'none';
  }
  return s === '' ? 'none' : s;
}

// 元素 → 图元（返回 null 表示该标签不属于 SVG 图元子集）
function nodeToShape(node, layerId, parentM, warn) {
  var type = node.tag.toLowerCase();
  if (SHAPE_TYPES.indexOf(type) < 0) return null;
  var st = styleMap(node);
  var m = mMul(parentM || M_ID, mParseTransform(node.attrs.transform));
  var sh = { id: node.attrs.id || '', type: type, layer: layerId, z: 0 };
  if (node.attrs['data-name']) sh.name = node.attrs['data-name'];

  if (type === 'rect') {
    sh.x = attrNum(node.attrs, 'x', 0);
    sh.y = attrNum(node.attrs, 'y', 0);
    sh.w = Math.max(0, attrNum(node.attrs, 'width', 0));
    sh.h = Math.max(0, attrNum(node.attrs, 'height', 0));
    var rx = attrNum(node.attrs, 'rx', 0);
    if (rx > 0) sh.rx = rx;
  } else if (type === 'circle') {
    sh.cx = attrNum(node.attrs, 'cx', 0);
    sh.cy = attrNum(node.attrs, 'cy', 0);
    sh.r = Math.max(0, attrNum(node.attrs, 'r', 0));
  } else if (type === 'ellipse') {
    sh.cx = attrNum(node.attrs, 'cx', 0);
    sh.cy = attrNum(node.attrs, 'cy', 0);
    sh.rx = Math.max(0, attrNum(node.attrs, 'rx', 0));
    sh.ry = Math.max(0, attrNum(node.attrs, 'ry', 0));
  } else if (type === 'line') {
    sh.x1 = attrNum(node.attrs, 'x1', 0);
    sh.y1 = attrNum(node.attrs, 'y1', 0);
    sh.x2 = attrNum(node.attrs, 'x2', 0);
    sh.y2 = attrNum(node.attrs, 'y2', 0);
  } else if (type === 'polyline' || type === 'polygon') {
    sh.points = pointsAttrToArr(node.attrs.points);
  } else if (type === 'path') {
    sh.d = normPathD(node.attrs.d);
  } else if (type === 'text') {
    sh.x = attrNum(node.attrs, 'x', 0);
    sh.y = attrNum(node.attrs, 'y', 0);
    sh.text = String(node.text || '').replace(/\s+/g, ' ').trim();
    sh.fontSize = num(parseFloat(st['font-size']), 16);
    if (st['font-weight']) sh.fontWeight = st['font-weight'];
    if (st['text-anchor'] === 'middle' || st['text-anchor'] === 'end') sh.anchor = st['text-anchor'];
  }

  var strokeRaw = st.stroke === undefined ? 'none' : st.stroke;
  if (type !== 'line') {
    sh.fill = sanitizeColorValue(st.fill === undefined ? '#000000' : st.fill, warn, 'fill');
  }
  sh.stroke = sanitizeColorValue(strokeRaw, warn, 'stroke');
  sh.strokeWidth = normColorOrRaw(strokeRaw) === 'none'
    ? 0
    : clamp(num(parseFloat(st['stroke-width']), 1), 0, LIMIT_CANVAS);
  sh.opacity = st.opacity === undefined ? 1 : clamp(num(parseFloat(st.opacity), 1), 0, 1);

  if (!mIsIdentity(m)) sh.transform = m;
  return sh;
}

function attrLength(v) {
  if (v === undefined || v === null) return 0;
  var s = String(v).trim();
  if (s.indexOf('%') >= 0) return 0;
  var n = parseFloat(s);
  return isNum(n) ? n : 0;
}

function vbNumbers(v) {
  if (!v) return null;
  var parts = String(v).split(/[\s,]+/).filter(function (x) { return x !== ''; });
  if (parts.length < 4) return null;
  var out = [];
  for (var i = 0; i < 4; i++) {
    var n = parseFloat(parts[i]);
    if (!isNum(n)) return null;
    out.push(n);
  }
  return out;
}

// 层内 z 归一化为 0..n-1（按现有 z 稳定排序）
function normalizeZ(proj) {
  var layers = proj.layers || [];
  for (var i = 0; i < layers.length; i++) {
    var list = [];
    for (var k = 0; k < (proj.shapes || []).length; k++) {
      if (wantsLayer(proj, proj.shapes[k].layer, layers[i].id)) list.push({ sh: proj.shapes[k], idx: k });
    }
    list.sort(function (a, b) {
      var za = num(a.sh.z, 0);
      var zb = num(b.sh.z, 0);
      if (za !== zb) return za - zb;
      return a.idx - b.idx;
    });
    for (var n = 0; n < list.length; n++) list[n].sh.z = n;
  }
}

var SKIPPED_HINTS = ['defs', 'use', 'image', 'tspan', 'filter', 'clipPath', 'mask',
  'linearGradient', 'radialGradient', 'pattern', 'symbol', 'marker', 'foreignObject'];

// SVG 文本 → 画板工程。返回 { project, warnings }
function svgToProject(text, opts) {
  var options = opts || {};
  var warn = [];
  var doc = parseXml(text);
  var svg = null;
  for (var i = 0; i < doc.children.length; i++) {
    if (doc.children[i].tag.toLowerCase() === 'svg') { svg = doc.children[i]; break; }
  }
  if (!svg) throw new Error('不是有效的 SVG：未找到 <svg> 根元素');

  var w = attrLength(svg.attrs.width);
  var h = attrLength(svg.attrs.height);
  var vb = vbNumbers(svg.attrs.viewBox);
  if ((!w || !h) && vb) { w = w || vb[2]; h = h || vb[3]; }
  w = Math.round(w || 800);
  h = Math.round(h || 600);

  var title = options.title || '';
  for (var t = 0; t < svg.children.length; t++) {
    if (svg.children[t].tag.toLowerCase() === 'title' && !title) {
      title = String(svg.children[t].text || '').trim();
    }
  }

  var proj = {
    schema: SCHEMA,
    meta: { title: title || '导入的画板', createdAt: nowIso() },
    canvas: {
      width: w,
      height: h,
      viewBox: svg.attrs.viewBox || ('0 0 ' + fmtNum(w) + ' ' + fmtNum(h)),
      background: 'none',
    },
    palette: PALETTE_DEFAULT.slice(),
    layers: [],
    shapes: [],
  };

  var defaultLayer = { id: 'l1', name: '图层 1', visible: true };
  var layerIndex = {};
  var shapeSeq = 0;
  var idUsed = {};

  function ensureLayer(id, name, visible) {
    if (!layerIndex[id]) {
      var ly = { id: id, name: name || id, visible: visible !== false };
      layerIndex[id] = ly;
      proj.layers.push(ly);
    }
    return layerIndex[id];
  }

  function nextId(preferred) {
    var base = preferred && /^[A-Za-z_][\w.-]*$/.test(preferred) ? preferred : ('s' + (++shapeSeq));
    var id = base;
    var n = 2;
    while (idUsed[id]) { id = base + '-' + n; n++; }
    idUsed[id] = true;
    return id;
  }

  function walk(node, layerId, parentM) {
    for (var c = 0; c < node.children.length; c++) {
      var child = node.children[c];
      var tag = child.tag.toLowerCase();
      if (tag === 'title' || tag === 'desc' || tag === 'metadata') continue;
      if (tag === 'script' || tag === 'style') {
        warn.push('已忽略 <' + tag + '> 元素（安全策略：不执行、不保留脚本）');
        continue;
      }
      if (tag === 'g') {
        var rawId = child.attrs.id || '';
        var gid = /^[A-Za-z_][\w.-]*$/.test(rawId) ? rawId : ('l' + (proj.layers.length + 1));
        ensureLayer(gid, child.attrs['data-name'] || rawId || gid,
          String(styleMap(child).display || '').toLowerCase() !== 'none');
        walk(child, gid, mMul(parentM, mParseTransform(child.attrs.transform)));
        continue;
      }
      var sh = nodeToShape(child, layerId, parentM, warn);
      if (!sh) {
        if (warn.length < 40) {
          var hint = SKIPPED_HINTS.indexOf(child.tag) >= 0 ? '（当前 SVG 子集不含该特性）' : '';
          warn.push('已跳过不支持的元素 <' + child.tag + '>' + hint);
        }
        continue;
      }
      // 背景板识别：id=bg 且铺满画板、无描边的矩形 → canvas.background（不计入图元）
      var isBg = child.attrs.id === 'bg' && sh.type === 'rect' &&
        Math.abs(sh.x) < EPS && Math.abs(sh.y) < EPS &&
        Math.abs(sh.w - proj.canvas.width) < 1 && Math.abs(sh.h - proj.canvas.height) < 1 &&
        normColorOrRaw(sh.stroke) === 'none';
      if (isBg && proj.canvas.background === 'none') {
        proj.canvas.background = sh.fill;
        continue;
      }
      sh.id = nextId(child.attrs.id);
      sh.z = proj.shapes.length;
      proj.shapes.push(sh);
    }
  }

  walk(svg, defaultLayer.id, M_ID);
  if (!proj.layers.length) proj.layers.push(defaultLayer);
  normalizeZ(proj);
  return { project: proj, warnings: warn };
}

// ── 工程模型 ───────────────────────────────────────────────
function emptyProject(title, w, h) {
  var width = clamp(Math.round(num(w, 800)), 1, LIMIT_CANVAS);
  var height = clamp(Math.round(num(h, 600)), 1, LIMIT_CANVAS);
  return {
    schema: SCHEMA,
    meta: { title: title || '未命名画板', createdAt: nowIso() },
    canvas: {
      width: width,
      height: height,
      viewBox: '0 0 ' + width + ' ' + height,
      background: '#FFFFFF',
    },
    palette: PALETTE_DEFAULT.slice(),
    layers: [{ id: 'l1', name: '图层 1', visible: true }],
    shapes: [],
  };
}

function normalizeProject(proj) {
  if (!proj || typeof proj !== 'object') throw new Error('工程内容不是对象');
  if (!proj.canvas) proj.canvas = { width: 800, height: 600, background: '#FFFFFF' };
  proj.canvas.width = clamp(Math.round(num(proj.canvas.width, 800)), 1, LIMIT_CANVAS);
  proj.canvas.height = clamp(Math.round(num(proj.canvas.height, 600)), 1, LIMIT_CANVAS);
  if (!proj.canvas.viewBox) proj.canvas.viewBox = '0 0 ' + proj.canvas.width + ' ' + proj.canvas.height;
  if (proj.canvas.background === undefined) proj.canvas.background = 'none';
  if (!proj.meta || typeof proj.meta !== 'object') proj.meta = { title: '未命名画板' };
  if (!proj.meta.createdAt) proj.meta.createdAt = nowIso();
  if (!proj.palette || !proj.palette.length) proj.palette = PALETTE_DEFAULT.slice();
  if (!proj.layers || !proj.layers.length) proj.layers = [{ id: 'l1', name: '图层 1', visible: true }];
  if (!proj.shapes) proj.shapes = [];
  for (var i = 0; i < proj.shapes.length; i++) {
    var sh = proj.shapes[i];
    if (sh.transform && mIsIdentity(sh.transform)) delete sh.transform;
    if (sh.opacity === undefined) sh.opacity = 1;
    if (sh.type !== 'line' && (sh.fill === undefined || sh.fill === null)) sh.fill = '#000000';
    if (sh.stroke === undefined || sh.stroke === null) sh.stroke = 'none';
    if (sh.strokeWidth === undefined || sh.strokeWidth === null) {
      sh.strokeWidth = normColorOrRaw(sh.stroke) === 'none' ? 0 : 1;
    }
    if (sh.z === undefined || sh.z === null) sh.z = i;
  }
  return proj;
}

function loadProject(ctx, path) {
  if (!ctx.fs.exists(path)) throw new Error('画板工程不存在: ' + path + '（先用 art_project mode=create 创建）');
  var txt = '';
  try {
    txt = ctx.fs.readFile(path);
  } catch (e) {
    throw new Error('读取工程失败: ' + ((e && e.message) ? e.message : e));
  }
  var obj;
  try {
    obj = JSON.parse(txt);
  } catch (e) {
    throw new Error('工程 JSON 解析失败（' + path + '）: ' + ((e && e.message) ? e.message : e));
  }
  return normalizeProject(obj);
}

function saveProject(ctx, path, proj) {
  ctx.fs.writeFile(path, JSON.stringify(proj, null, 2) + '\n');
}

// ── 图元查询与几何操作 ─────────────────────────────────────
function findShape(proj, idOrIndex) {
  var list = proj.shapes || [];
  for (var i = 0; i < list.length; i++) if (list[i].id === idOrIndex) return list[i];
  if (isInt(idOrIndex) && idOrIndex >= 0 && idOrIndex < list.length) return list[idOrIndex];
  var asInt = parseInt(idOrIndex, 10);
  if (isNum(asInt) && asInt >= 0 && asInt < list.length) return list[asInt];
  return null;
}

// 目标解析：ids / id / type / layer / name / all —— 空条件直接报错（防止误改全部）
function resolveTargets(proj, spec) {
  var s = spec || {};
  var out = [];
  var seen = {};
  function push(sh) { if (sh && !seen[sh.id]) { seen[sh.id] = true; out.push(sh); } }

  if (s.ids && s.ids.length) {
    for (var i = 0; i < s.ids.length; i++) {
      var found = findShape(proj, s.ids[i]);
      if (!found) throw new Error('图元不存在: ' + s.ids[i]);
      push(found);
    }
    return out;
  }
  if (s.id) {
    var one = findShape(proj, s.id);
    if (!one) throw new Error('图元不存在: ' + s.id);
    push(one);
    return out;
  }
  var hasFilter = s.type || s.layer || s.name || s.all;
  if (!hasFilter) {
    throw new Error('必须指定目标：ids / id / type / layer / name / all=true（避免误改全部图元）');
  }
  var list = proj.shapes || [];
  for (var k = 0; k < list.length; k++) {
    var sh = list[k];
    if (s.type && sh.type !== s.type) continue;
    if (s.layer && sh.layer !== s.layer) continue;
    if (s.name && String(sh.name || '') !== String(s.name)) continue;
    push(sh);
  }
  return out;
}

function layerIds(proj) {
  var out = [];
  for (var i = 0; i < (proj.layers || []).length; i++) out.push(proj.layers[i].id);
  return out;
}

function firstLayerId(proj) {
  var ids = layerIds(proj);
  return ids.length ? ids[0] : 'l1';
}

function newShapeId(proj) {
  var used = {};
  for (var i = 0; i < (proj.shapes || []).length; i++) used[proj.shapes[i].id] = true;
  var n = 1;
  while (used['s' + n]) n++;
  return 's' + n;
}

function shapeCenter(sh) {
  var b = shapeBBox(sh);
  return { x: b.x + b.w / 2, y: b.y + b.h / 2 };
}

// 平移：无 transform（或纯平移）的直接改几何字段（diff 友好）；否则累加矩阵平移
function moveShapeBy(sh, dx, dy) {
  var t = sh.transform;
  if (t && !mIsTranslateOnly(t)) {
    var m = t.slice();
    m[4] += dx;
    m[5] += dy;
    sh.transform = mNorm(m);
    return;
  }
  if (t) { // 纯平移：吸收进几何字段，避免矩阵残留
    dx += t[4];
    dy += t[5];
    delete sh.transform;
  }
  var type = sh.type;
  if (type === 'rect' || type === 'text') {
    sh.x = num(sh.x, 0) + dx;
    sh.y = num(sh.y, 0) + dy;
  } else if (type === 'circle' || type === 'ellipse') {
    sh.cx = num(sh.cx, 0) + dx;
    sh.cy = num(sh.cy, 0) + dy;
  } else if (type === 'line') {
    sh.x1 = num(sh.x1, 0) + dx;
    sh.y1 = num(sh.y1, 0) + dy;
    sh.x2 = num(sh.x2, 0) + dx;
    sh.y2 = num(sh.y2, 0) + dy;
  } else if (type === 'polyline' || type === 'polygon') {
    var pts = sh.points || [];
    var out = [];
    for (var i = 0; i < pts.length; i++) out.push([num(pts[i][0], 0) + dx, num(pts[i][1], 0) + dy]);
    sh.points = out;
  } else {
    // path：坐标内嵌于 d，无法逐点平移 → 退化为矩阵平移
    sh.transform = mNorm([1, 0, 0, 1, dx, dy]);
  }
}

// 改尺寸：about = 'nw'（默认，保持左上角）| 'center'（保持中心）
function resizeShape(sh, opts) {
  var o = opts || {};
  var about = o.about === 'center' ? 'center' : 'nw';
  var before = shapeBBox(sh);
  var type = sh.type;
  if (type === 'rect') {
    var w = o.w === undefined ? num(sh.w, 0) : Math.max(0, num(o.w, 0));
    var h = o.h === undefined ? num(sh.h, 0) : Math.max(0, num(o.h, 0));
    sh.w = w;
    sh.h = h;
  } else if (type === 'circle') {
    sh.r = o.r === undefined ? num(sh.r, 0) * num(o.scale, 1) : Math.max(0, num(o.r, 0));
  } else if (type === 'ellipse') {
    sh.rx = o.rx === undefined ? num(sh.rx, 0) * num(o.scale, 1) : Math.max(0, num(o.rx, 0));
    sh.ry = o.ry === undefined ? num(sh.ry, 0) * num(o.scale, 1) : Math.max(0, num(o.ry, 0));
  } else if (type === 'text') {
    sh.fontSize = Math.max(1, o.fontSize === undefined ? num(sh.fontSize, 16) * num(o.scale, 1) : num(o.fontSize, 16));
  } else {
    throw new Error('shape.resize 不支持 ' + type + '（path/polyline/polygon 请用 art_edit 的 shape.set 或重建）');
  }
  if (about === 'center') {
    var after = shapeBBox(sh);
    moveShapeBy(sh, (before.x + before.w / 2) - (after.x + after.w / 2),
      (before.y + before.h / 2) - (after.y + after.h / 2));
  }
  return before;
}

// 允许通过 shape.set 改的样式字段（id/type 等结构性字段不允许改）
var SETTABLE = ['fill', 'stroke', 'strokeWidth', 'opacity', 'name', 'layer', 'anchor',
  'fontWeight', 'fontSize', 'rx', 'visible', 'href', 'text'];

function setShapeProps(ctx, proj, sh, set) {
  var applied = [];
  for (var k in set) {
    if (!Object.prototype.hasOwnProperty.call(set, k)) continue;
    if (SETTABLE.indexOf(k) < 0) {
      throw new Error('shape.set 不允许修改字段: ' + k + '（可改：' + SETTABLE.join('/') + '）');
    }
    if (k === 'layer') {
      var ids = layerIds(proj);
      if (ids.indexOf(set[k]) < 0) throw new Error('图层不存在: ' + set[k] + '（现有：' + ids.join(', ') + '）');
    }
    if (k === 'fill' || k === 'stroke') {
      sh[k] = sanitizeColorValue(set[k], null, k);
    } else {
      sh[k] = set[k];
    }
    applied.push(k + '=' + set[k]);
  }
  if (set.stroke !== undefined && set.strokeWidth === undefined && normColorOrRaw(sh.stroke) !== 'none' &&
    num(sh.strokeWidth, 0) <= 0) {
    sh.strokeWidth = 1; // 由 none 改为实色描边时补默认线宽，避免"设了颜色却看不见"
  }
  return applied;
}

// ── 图元构造 ───────────────────────────────────────────────
var ADD_GEOM = {
  rect: ['x', 'y', 'w', 'h', 'rx'],
  circle: ['cx', 'cy', 'r'],
  ellipse: ['cx', 'cy', 'rx', 'ry'],
  line: ['x1', 'y1', 'x2', 'y2'],
  polyline: ['points'],
  polygon: ['points'],
  path: ['d'],
  text: ['x', 'y', 'text', 'fontSize', 'fontWeight', 'anchor'],
};

function makeShape(proj, spec) {
  var type = String(spec.type || '').toLowerCase();
  if (SHAPE_TYPES.indexOf(type) < 0) {
    throw new Error('未知图元类型: ' + spec.type + '（可用：' + SHAPE_TYPES.join(' / ') + '）');
  }
  var ids = layerIds(proj);
  var layer = spec.layer || firstLayerId(proj);
  if (ids.indexOf(layer) < 0) {
    throw new Error('图层不存在: ' + layer + '（现有：' + ids.join(', ') + '；可先 layer.add）');
  }
  var sh = { id: spec.id ? String(spec.id) : newShapeId(proj), type: type, layer: layer, z: 0 };
  if (spec.name) sh.name = String(spec.name);

  var fields = ADD_GEOM[type];
  for (var i = 0; i < fields.length; i++) {
    var f = fields[i];
    if (spec[f] === undefined || spec[f] === null) continue;
    if (f === 'points') {
      sh.points = typeof spec[f] === 'string' ? pointsAttrToArr(spec[f]) : spec[f];
    } else if (f === 'text' || f === 'anchor' || f === 'fontWeight' || f === 'd') {
      sh[f] = String(spec[f]);
    } else {
      sh[f] = num(spec[f], 0);
    }
  }
  if (type === 'text' && (sh.text === undefined || sh.text === '')) {
    throw new Error('text 图元必须提供 text 内容');
  }
  if (type === 'path' && !sh.d) throw new Error('path 图元必须提供 d（路径数据）');
  if ((type === 'polyline' || type === 'polygon') && (!sh.points || sh.points.length < 2)) {
    throw new Error(type + ' 至少需要 2 个点（points 形如 "0,0 40,0 40,30"）');
  }
  if (type === 'text' && sh.fontSize === undefined) sh.fontSize = 16;

  if (type !== 'line') sh.fill = sanitizeColorValue(spec.fill === undefined ? '#000000' : spec.fill, null, 'fill');
  sh.stroke = sanitizeColorValue(spec.stroke === undefined ? 'none' : spec.stroke, null, 'stroke');
  sh.strokeWidth = spec.strokeWidth === undefined
    ? (normColorOrRaw(sh.stroke) === 'none' ? 0 : 1)
    : clamp(num(spec.strokeWidth, 1), 0, LIMIT_CANVAS);
  sh.opacity = spec.opacity === undefined ? 1 : clamp(num(spec.opacity, 1), 0, 1);
  return sh;
}

// ── op 引擎（命令式编辑链，与 UI 操作同源） ────────────────
function applyOps(proj, ops) {
  var log = [];
  for (var i = 0; i < ops.length; i++) {
    var op = ops[i] || {};
    var name = String(op.op || '');
    if (name === 'shape.add') {
      var sh = makeShape(proj, op);
      var sameLayer = shapesOfLayerInZOrder(proj, sh.layer);
      sh.z = sameLayer.length;
      proj.shapes.push(sh);
      log.push('新增 ' + sh.type + ' ' + sh.id + '（图层 ' + sh.layer + '，z=' + sh.z + '）');
    } else if (name === 'shape.remove') {
      var dels = resolveTargets(proj, op);
      var keep = [];
      var removed = 0;
      var delIds = {};
      for (var a = 0; a < dels.length; a++) delIds[dels[a].id] = true;
      for (var b = 0; b < proj.shapes.length; b++) {
        if (delIds[proj.shapes[b].id]) { removed++; continue; }
        keep.push(proj.shapes[b]);
      }
      proj.shapes = keep;
      log.push('删除 ' + removed + ' 个图元');
    } else if (name === 'shape.duplicate') {
      var srcs = resolveTargets(proj, op);
      var dx = num(op.dx, 16);
      var dy = num(op.dy, 16);
      for (var d = 0; d < srcs.length; d++) {
        var copy = JSON.parse(JSON.stringify(srcs[d]));
        copy.id = op.newId ? String(op.newId) : newShapeId(proj);
        moveShapeBy(copy, dx, dy);
        copy.z = shapesOfLayerInZOrder(proj, copy.layer).length;
        proj.shapes.push(copy);
        log.push('复制 ' + srcs[d].id + ' → ' + copy.id + '（偏移 ' + fmtNum(dx) + ',' + fmtNum(dy) + '）');
      }
    } else if (name === 'shape.move') {
      var mv = resolveTargets(proj, op);
      for (var m = 0; m < mv.length; m++) {
        if (op.to && typeof op.to === 'object') {
          var box = shapeBBox(mv[m]);
          var tx = num(op.to.x, box.x);
          var ty = num(op.to.y, box.y);
          moveShapeBy(mv[m], tx - box.x, ty - box.y);
        } else {
          moveShapeBy(mv[m], num(op.dx, 0), num(op.dy, 0));
        }
      }
      log.push('平移 ' + mv.length + ' 个图元' + (op.to ? '（到指定坐标）' : '（dx=' + fmtNum(num(op.dx, 0)) + ', dy=' + fmtNum(num(op.dy, 0)) + '）'));
    } else if (name === 'shape.resize') {
      var rz = resolveTargets(proj, op);
      for (var r = 0; r < rz.length; r++) {
        resizeShape(rz[r], { w: op.w, h: op.h, r: op.r, rx: op.rx, ry: op.ry, scale: op.scale, fontSize: op.fontSize, about: op.about });
      }
      log.push('调整 ' + rz.length + ' 个图元尺寸（about=' + (op.about === 'center' ? 'center' : 'nw') + '）');
    } else if (name === 'shape.set') {
      var sts = resolveTargets(proj, op);
      if (!op.set || typeof op.set !== 'object') throw new Error('shape.set 需要 set 对象（如 {"fill":"#2563EB"}）');
      for (var s = 0; s < sts.length; s++) setShapeProps(null, proj, sts[s], op.set);
      log.push('修改 ' + sts.length + ' 个图元属性：' + keysOf(op.set).join(', '));
    } else if (name === 'shape.text') {
      var txs = resolveTargets(proj, op);
      if (op.text === undefined) throw new Error('shape.text 需要 text 参数');
      for (var t = 0; t < txs.length; t++) {
        if (txs[t].type !== 'text') throw new Error('shape.text 只能用于 text 图元（' + txs[t].id + ' 是 ' + txs[t].type + '）');
        txs[t].text = String(op.text);
      }
      log.push('更新 ' + txs.length + ' 个文本内容');
    } else if (name === 'shape.align') {
      log.push(applyAlign(proj, op));
    } else if (name === 'shape.distribute') {
      log.push(applyDistribute(proj, op));
    } else if (name === 'shape.z') {
      log.push(applyZOrder(proj, op));
    } else if (name === 'layer.add') {
      var lid = String(op.id || ('l' + (proj.layers.length + 1)));
      if (layerIds(proj).indexOf(lid) >= 0) throw new Error('图层已存在: ' + lid);
      proj.layers.push({ id: lid, name: op.name ? String(op.name) : ('图层 ' + (proj.layers.length + 1)), visible: op.visible === false ? false : true });
      log.push('新增图层 ' + lid + '（' + (op.name || '') + '）');
    } else if (name === 'layer.remove') {
      var rid = String(op.id || '');
      if (layerIds(proj).indexOf(rid) < 0) throw new Error('图层不存在: ' + rid);
      var has = shapesOfLayerInZOrder(proj, rid);
      if (has.length) {
        if (!op.moveTo) throw new Error('图层 ' + rid + ' 还有 ' + has.length + ' 个图元（传 moveTo 指定迁移目标层，或先删图元）');
        if (layerIds(proj).indexOf(op.moveTo) < 0) throw new Error('目标图层不存在: ' + op.moveTo);
        for (var q = 0; q < has.length; q++) has[q].layer = op.moveTo;
      }
      var rest = [];
      for (var l = 0; l < proj.layers.length; l++) if (proj.layers[l].id !== rid) rest.push(proj.layers[l]);
      if (!rest.length) throw new Error('不能删除最后一个图层');
      proj.layers = rest;
      log.push('删除图层 ' + rid + (has.length ? '（' + has.length + ' 个图元迁移至 ' + op.moveTo + '）' : ''));
    } else if (name === 'layer.rename') {
      var lyr = findLayer(proj, op.id);
      if (!lyr) throw new Error('图层不存在: ' + op.id);
      lyr.name = String(op.name === undefined ? lyr.name : op.name);
      log.push('图层 ' + lyr.id + ' 重命名为 ' + lyr.name);
    } else if (name === 'layer.visible') {
      var lv = findLayer(proj, op.id);
      if (!lv) throw new Error('图层不存在: ' + op.id);
      lv.visible = op.visible === undefined ? !lv.visible : !!op.visible;
      log.push('图层 ' + lv.id + ' → ' + (lv.visible ? '显示' : '隐藏'));
    } else if (name === 'layer.reorder') {
      var idx = num(op.index, -1);
      var cur = -1;
      for (var g = 0; g < proj.layers.length; g++) if (proj.layers[g].id === op.id) cur = g;
      if (cur < 0) throw new Error('图层不存在: ' + op.id);
      if (idx < 0 || idx >= proj.layers.length) throw new Error('index 越界（0..' + (proj.layers.length - 1) + '）');
      var moved = proj.layers.splice(cur, 1)[0];
      proj.layers.splice(idx, 0, moved);
      log.push('图层 ' + moved.id + ' 移到位置 ' + idx);
    } else if (name === 'set.canvas') {
      if (op.width !== undefined) proj.canvas.width = clamp(Math.round(num(op.width, proj.canvas.width)), 1, LIMIT_CANVAS);
      if (op.height !== undefined) proj.canvas.height = clamp(Math.round(num(op.height, proj.canvas.height)), 1, LIMIT_CANVAS);
      if (op.background !== undefined) proj.canvas.background = sanitizeColorValue(op.background, null, 'background');
      if (op.viewBox) proj.canvas.viewBox = String(op.viewBox);
      else proj.canvas.viewBox = '0 0 ' + proj.canvas.width + ' ' + proj.canvas.height;
      log.push('画布 → ' + proj.canvas.width + '×' + proj.canvas.height + '（背景 ' + proj.canvas.background + '）');
    } else if (name === 'set.palette') {
      if (!op.colors || !op.colors.length) throw new Error('set.palette 需要 colors 数组');
      var cols = [];
      for (var p = 0; p < op.colors.length; p++) {
        if (!parseColor(op.colors[p])) throw new Error('色板值无法解析: ' + op.colors[p]);
        cols.push(String(op.colors[p]));
      }
      proj.palette = cols;
      log.push('色板 → ' + cols.length + ' 色');
    } else if (name === 'project.rename') {
      proj.meta.title = String(op.title === undefined ? proj.meta.title : op.title);
      log.push('标题 → ' + proj.meta.title);
    } else {
      throw new Error('未知 op: ' + name + '（可用：shape.add/remove/duplicate/move/resize/set/text/align/distribute/z、' +
        'layer.add/remove/rename/visible/reorder、set.canvas/set.palette、project.rename）');
    }
  }
  normalizeZ(proj);
  return log;
}

function findLayer(proj, id) {
  for (var i = 0; i < (proj.layers || []).length; i++) if (proj.layers[i].id === id) return proj.layers[i];
  return null;
}

var ALIGN_KEYS = ['left', 'hcenter', 'right', 'top', 'vcenter', 'bottom'];

function applyAlign(proj, op) {
  var to = String(op.to || 'left');
  if (ALIGN_KEYS.indexOf(to) < 0) {
    throw new Error('未知对齐方式: ' + to + '（可用：' + ALIGN_KEYS.join(' / ') + '）');
  }
  var targets = resolveTargets(proj, op);
  var refBox;
  if (op.ref && op.ref !== 'canvas') {
    var refShape = findShape(proj, op.ref);
    if (!refShape) throw new Error('参照图元不存在: ' + op.ref);
    refBox = shapeBBox(refShape);
  } else {
    refBox = { x: 0, y: 0, w: proj.canvas.width, h: proj.canvas.height };
  }
  for (var i = 0; i < targets.length; i++) {
    var b = shapeBBox(targets[i]);
    var dx = 0;
    var dy = 0;
    if (to === 'left') dx = refBox.x - b.x;
    else if (to === 'hcenter') dx = (refBox.x + refBox.w / 2) - (b.x + b.w / 2);
    else if (to === 'right') dx = (refBox.x + refBox.w) - (b.x + b.w);
    else if (to === 'top') dy = refBox.y - b.y;
    else if (to === 'vcenter') dy = (refBox.y + refBox.h / 2) - (b.y + b.h / 2);
    else if (to === 'bottom') dy = (refBox.y + refBox.h) - (b.y + b.h);
    moveShapeBy(targets[i], dx, dy);
  }
  return '对齐 ' + targets.length + ' 个图元 → ' + to + '（参照 ' + (op.ref && op.ref !== 'canvas' ? op.ref : '画板') + '）';
}

function applyDistribute(proj, op) {
  var axis = String(op.axis || 'h');
  if (axis !== 'h' && axis !== 'v') throw new Error('axis 只支持 h（水平）或 v（垂直）');
  var targets = resolveTargets(proj, op);
  if (targets.length < 3) throw new Error('等距分布至少需要 3 个图元（当前 ' + targets.length + ' 个）');
  var items = [];
  for (var i = 0; i < targets.length; i++) items.push({ sh: targets[i], b: shapeBBox(targets[i]) });
  items.sort(function (a, b) {
    return axis === 'h' ? (a.b.x - b.b.x) : (a.b.y - b.b.y);
  });
  var first = items[0].b;
  var last = items[items.length - 1].b;
  var span = axis === 'h' ? (last.x + last.w - first.x) : (last.y + last.h - first.y);
  var sum = 0;
  for (var k = 0; k < items.length; k++) sum += axis === 'h' ? items[k].b.w : items[k].b.h;
  var gap = (span - sum) / (items.length - 1);
  var cursor = axis === 'h' ? first.x : first.y;
  for (var n = 0; n < items.length; n++) {
    var box = items[n].b;
    if (n > 0) {
      var delta = cursor - (axis === 'h' ? box.x : box.y);
      moveShapeBy(items[n].sh, axis === 'h' ? delta : 0, axis === 'h' ? 0 : delta);
    }
    cursor += (axis === 'h' ? box.w : box.h) + gap;
  }
  return '等距分布 ' + items.length + ' 个图元（' + (axis === 'h' ? '水平' : '垂直') + '，间隙 ' + fmtNum(gap) + '）';
}

function applyZOrder(proj, op) {
  var targets = resolveTargets(proj, op);
  if (isInt(op.z)) {
    for (var i = 0; i < targets.length; i++) targets[i].z = op.z;
    return '设置 ' + targets.length + ' 个图元 z=' + op.z;
  }
  var mode = String(op.mode || '');
  if (['front', 'back', 'forward', 'backward'].indexOf(mode) < 0) {
    throw new Error('shape.z 需要 z（整数）或 mode=front/back/forward/backward');
  }
  for (var t = 0; t < targets.length; t++) {
    var sh = targets[t];
    var list = shapesOfLayerInZOrder(proj, sh.layer);
    var idx = list.indexOf(sh);
    if (mode === 'front') sh.z = list.length + 10;
    else if (mode === 'back') sh.z = -1 - idx;
    else if (mode === 'forward') sh.z = (idx + 1 < list.length ? num(list[idx + 1].z, 0) + 0.5 : num(sh.z, 0) + 1);
    else if (mode === 'backward') sh.z = (idx > 0 ? num(list[idx - 1].z, 0) - 0.5 : num(sh.z, 0) - 1);
  }
  normalizeZ(proj);
  return '调整 ' + targets.length + ' 个图元层级（' + mode + '）';
}

// ── 摘要 ───────────────────────────────────────────────────
function projectSummary(proj, path) {
  var lines = [];
  var cv = proj.canvas;
  lines.push('## 画板 ' + (path || ''));
  lines.push('- 标题: ' + ((proj.meta && proj.meta.title) || '未命名画板'));
  lines.push('- 画布: ' + cv.width + '×' + cv.height + '（viewBox ' + cv.viewBox + '，背景 ' + cv.background + '）');
  var ly = [];
  for (var i = 0; i < proj.layers.length; i++) {
    var n = shapesOfLayerInZOrder(proj, proj.layers[i].id).length;
    ly.push(proj.layers[i].id + (proj.layers[i].visible === false ? '(隐藏)' : '') + ':' + n);
  }
  lines.push('- 图层: ' + proj.layers.length + '（' + ly.join(' / ') + '）');
  var byType = {};
  var union = null;
  var colors = {};
  for (var s = 0; s < proj.shapes.length; s++) {
    var sh = proj.shapes[s];
    byType[sh.type] = (byType[sh.type] || 0) + 1;
    union = bboxUnion(union, shapeBBox(sh));
    if (sh.fill && normColorOrRaw(sh.fill) === 'color') colors[String(sh.fill)] = true;
    if (sh.stroke && normColorOrRaw(sh.stroke) === 'color') colors[String(sh.stroke)] = true;
  }
  var typeStr = [];
  for (var t in byType) if (Object.prototype.hasOwnProperty.call(byType, t)) typeStr.push(t + ' ' + byType[t]);
  lines.push('- 图元: ' + proj.shapes.length + (typeStr.length ? '（' + typeStr.join(', ') + '）' : ''));
  if (union) {
    var cover = clamp(bboxArea(union) / Math.max(1, cv.width * cv.height) * 100, 0, 100);
    lines.push('- 覆盖范围: x ' + fmtNum(union.x) + '..' + fmtNum(union.x + union.w) +
      '，y ' + fmtNum(union.y) + '..' + fmtNum(union.y + union.h) + '（占画板 ' + cover.toFixed(1) + '%）');
  }
  lines.push('- 用色: ' + keysOf(colors).length + ' 种' + (keysOf(colors).length ? '（' + keysOf(colors).join(', ') + '）' : ''));
  lines.push('- 色板: ' + (proj.palette || []).length + ' 色（可用 art_edit set.palette 替换）');
  return lines.join('\n');
}

// ── 校验器（7 项判据） ─────────────────────────────────────
var TOL_ROUND = 1e-3; // 导出精度（3 位小数）对应的往返容差

function near(a, b) { return Math.abs(num(a, 0) - num(b, 0)) <= TOL_ROUND; }

function shapeRequiredFields(sh) {
  var t = sh.type;
  if (t === 'rect') return ['x', 'y', 'w', 'h'];
  if (t === 'circle') return ['cx', 'cy', 'r'];
  if (t === 'ellipse') return ['cx', 'cy', 'rx', 'ry'];
  if (t === 'line') return ['x1', 'y1', 'x2', 'y2'];
  if (t === 'path') return ['d'];
  if (t === 'text') return ['x', 'y', 'text', 'fontSize'];
  if (t === 'polyline' || t === 'polygon') return ['points'];
  return [];
}

// 单图元结构检查 → 错误信息数组
function shapeStructErrors(sh) {
  var errs = [];
  if (SHAPE_TYPES.indexOf(sh.type) < 0) {
    errs.push(sh.id + ': 未知类型 ' + sh.type);
    return errs;
  }
  var fields = shapeRequiredFields(sh);
  for (var i = 0; i < fields.length; i++) {
    var f = fields[i];
    if (f === 'points') {
      var pts = sh.points;
      if (!pts || pts.length < 2) { errs.push(sh.id + ': points 至少需要 2 个点'); continue; }
      for (var k = 0; k < pts.length; k++) {
        if (!isNum(num(pts[k][0], NaN)) || !isNum(num(pts[k][1], NaN))) {
          errs.push(sh.id + ': points[' + k + '] 非法');
          break;
        }
      }
    } else if (f === 'd') {
      if (!sh.d) errs.push(sh.id + ': path 缺少 d');
    } else if (f === 'text') {
      if (sh.text === undefined || sh.text === null || String(sh.text) === '') errs.push(sh.id + ': text 内容为空');
    } else if (!isNum(num(sh[f], NaN))) {
      errs.push(sh.id + ': ' + f + ' 非有限数值');
    }
  }
  var pos = { rect: ['w', 'h'], circle: ['r'], ellipse: ['rx', 'ry'], text: ['fontSize'] };
  if (pos[sh.type]) {
    for (var p = 0; p < pos[sh.type].length; p++) {
      var key = pos[sh.type][p];
      if (isNum(num(sh[key], NaN)) && num(sh[key], 0) <= 0) errs.push(sh.id + ': ' + key + ' 必须 > 0');
    }
  }
  if (sh.fill !== undefined && parseColor(sh.fill) === null) errs.push(sh.id + ': fill 无法解析（' + sh.fill + '）');
  if (sh.stroke !== undefined && parseColor(sh.stroke) === null) errs.push(sh.id + ': stroke 无法解析（' + sh.stroke + '）');
  if (sh.transform && !mIsIdentity(sh.transform)) {
    for (var m = 0; m < 6; m++) {
      if (!isNum(num(sh.transform[m], NaN))) { errs.push(sh.id + ': transform 非法'); break; }
    }
  }
  return errs;
}

// 文本「背后」的颜色：按 (图层序, z) 升序找最后一个包住文本中心的实心图元；无则画板背景（缺省白）
function textBackdrop(proj, sh) {
  var center = shapeCenter(sh);
  var found = null;
  for (var li = 0; li < (proj.layers || []).length; li++) {
    if (proj.layers[li].visible === false) continue;
    var list = shapesOfLayerInZOrder(proj, proj.layers[li].id);
    for (var i = 0; i < list.length; i++) {
      var o = list[i];
      if (o === sh || o.type === 'text') continue;
      var f = parseColor(o.fill);
      if (!f || f.none || f.a <= 0.05) continue;
      var b = shapeBBox(o);
      if (center.x >= b.x && center.x <= b.x + b.w && center.y >= b.y && center.y <= b.y + b.h) {
        found = blendOver({ r: f.r, g: f.g, b: f.b, a: f.a * clamp(num(o.opacity, 1), 0, 1), none: false, current: false },
          found || { r: 255, g: 255, b: 255, a: 1, none: false, current: false });
      }
    }
  }
  if (found) return { color: found, from: 'shape' };
  var bg = parseColor(proj.canvas.background);
  if (bg && !bg.none && bg.a > 0.05) return { color: bg, from: 'canvas' };
  return { color: { r: 255, g: 255, b: 255, a: 1, none: false, current: false }, from: 'default' };
}

function isLargeText(sh) {
  var fs = num(sh.fontSize, 16);
  var fw = String(sh.fontWeight === undefined ? '400' : sh.fontWeight);
  var bold = fw === 'bold' || fw === 'bolder' || num(fw, 400) >= 600;
  return fs >= 24 || (bold && fs >= 18.66);
}

// 工程往返比较（导出 SVG → 再导入）→ 差异描述数组
function compareProjects(proj, back) {
  var diffs = [];
  if (back.canvas.width !== proj.canvas.width || back.canvas.height !== proj.canvas.height) {
    diffs.push('画布尺寸不一致: ' + proj.canvas.width + '×' + proj.canvas.height + ' → ' +
      back.canvas.width + '×' + back.canvas.height);
  }
  if (String(back.canvas.background) !== String(proj.canvas.background)) {
    diffs.push('背景不一致: ' + proj.canvas.background + ' → ' + back.canvas.background);
  }
  if ((back.layers || []).length !== (proj.layers || []).length) {
    diffs.push('图层数量不一致: ' + proj.layers.length + ' → ' + back.layers.length);
  }
  if (back.shapes.length !== proj.shapes.length) {
    diffs.push('图元数量不一致: ' + proj.shapes.length + ' → ' + back.shapes.length);
    return diffs; // 数量不同后逐项比没有意义
  }
  var LAYER_GEOM = {
    rect: ['x', 'y', 'w', 'h', 'rx'], circle: ['cx', 'cy', 'r'], ellipse: ['cx', 'cy', 'rx', 'ry'],
    line: ['x1', 'y1', 'x2', 'y2'], text: ['x', 'y', 'fontSize', 'text'],
  };
  for (var li = 0; li < proj.layers.length; li++) {
    var a = shapesOfLayerInZOrder(proj, proj.layers[li].id);
    var b = shapesOfLayerInZOrder(back, proj.layers[li].id);
    if (a.length !== b.length) {
      diffs.push('图层 ' + proj.layers[li].id + ' 图元数不一致: ' + a.length + ' → ' + b.length);
      continue;
    }
    for (var i = 0; i < a.length; i++) {
      var x = a[i];
      var y = b[i];
      if (x.id !== y.id) { diffs.push('第 ' + i + ' 项 id 不一致: ' + x.id + ' → ' + y.id); continue; }
      if (x.type !== y.type) { diffs.push(x.id + ': 类型不一致 ' + x.type + ' → ' + y.type); continue; }
      var keys = LAYER_GEOM[x.type] || [];
      for (var k = 0; k < keys.length; k++) {
        var key = keys[k];
        if (key === 'text') {
          if (String(x.text === undefined ? '' : x.text) !== String(y.text === undefined ? '' : y.text)) {
            diffs.push(x.id + ': text 内容不一致');
          }
        } else if ((x[key] === undefined) !== (y[key] === undefined)) {
          diffs.push(x.id + ': ' + key + ' 存在性不一致');
        } else if (x[key] !== undefined && !near(x[key], y[key])) {
          diffs.push(x.id + ': ' + key + ' ' + fmtNum(num(x[key], 0)) + ' → ' + fmtNum(num(y[key], 0)));
        }
      }
      if (x.type === 'polyline' || x.type === 'polygon') {
        var px = x.points || [];
        var py = y.points || [];
        if (px.length !== py.length) diffs.push(x.id + ': points 点数不一致');
        else {
          for (var p = 0; p < px.length; p++) {
            if (!near(px[p][0], py[p][0]) || !near(px[p][1], py[p][1])) {
              diffs.push(x.id + ': points[' + p + '] 不一致');
              break;
            }
          }
        }
      }
      if (x.type === 'path' && normPathD(x.d) !== normPathD(y.d)) diffs.push(x.id + ': path d 不一致');
      if (String(x.fill === undefined ? '' : x.fill) !== String(y.fill === undefined ? '' : y.fill)) {
        diffs.push(x.id + ': fill 不一致（' + x.fill + ' → ' + y.fill + '）');
      }
      if (String(x.stroke === undefined ? '' : x.stroke) !== String(y.stroke === undefined ? '' : y.stroke)) {
        diffs.push(x.id + ': stroke 不一致（' + x.stroke + ' → ' + y.stroke + '）');
      }
      if (!near(num(x.strokeWidth, 0), num(y.strokeWidth, 0))) diffs.push(x.id + ': strokeWidth 不一致');
      if (!near(num(x.opacity, 1), num(y.opacity, 1))) diffs.push(x.id + ': opacity 不一致');
      var mx = x.transform && !mIsIdentity(x.transform) ? x.transform : null;
      var my = y.transform && !mIsIdentity(y.transform) ? y.transform : null;
      if ((mx === null) !== (my === null)) {
        diffs.push(x.id + ': transform 存在性不一致');
      } else if (mx) {
        for (var mm = 0; mm < 6; mm++) {
          if (!near(mx[mm], my[mm])) { diffs.push(x.id + ': transform 不一致'); break; }
        }
      }
      if (diffs.length > 6) break;
    }
  }
  return diffs;
}

function verifyProject(ctx, proj, opts) {
  var o = opts || {};
  var checks = [];
  var cv = proj.canvas;

  // V1 画板
  var vb = vbNumbers(cv.viewBox);
  var v1pb = isInt(num(cv.width, 0)) && num(cv.width, 0) >= 1 && num(cv.width, 0) <= LIMIT_CANVAS &&
    isInt(num(cv.height, 0)) && num(cv.height, 0) >= 1 && num(cv.height, 0) <= LIMIT_CANVAS;
  var bgc = parseColor(cv.background);
  var v1 = v1pb && !!vb && vb[2] > 0 && vb[3] > 0 && bgc !== null;
  var v1m = cv.width + '×' + cv.height + '；viewBox ' + cv.viewBox + '（' + (vb ? '合法' : '非法') +
    '）；背景 ' + cv.background + '（' + (bgc === null ? '无法解析' : '可解析') + '）';
  checks.push({ id: 'V1', name: '画板合法（尺寸 / viewBox / 背景色）', pass: v1, metric: v1m });

  // V2 图元结构
  var errsAll = [];
  var seenId = {};
  var dupIds = 0;
  for (var i = 0; i < proj.shapes.length; i++) {
    var sh = proj.shapes[i];
    if (seenId[sh.id]) dupIds++;
    seenId[sh.id] = true;
    if (!sh.id) errsAll.push('存在空 id');
    if (layerIds(proj).indexOf(sh.layer) < 0) errsAll.push(sh.id + ': 图层不存在（' + sh.layer + '）');
    var es = shapeStructErrors(sh);
    for (var e = 0; e < es.length; e++) errsAll.push(es[e]);
  }
  if (dupIds) errsAll.push('重复 id ' + dupIds + ' 个');
  if (proj.shapes.length > LIMIT_SHAPES) errsAll.push('图元数 ' + proj.shapes.length + ' 超过上限 ' + LIMIT_SHAPES);
  var v2 = errsAll.length === 0;
  var v2m = proj.shapes.length + ' 个图元；结构问题 ' + errsAll.length + ' 处' +
    (errsAll.length ? '｜' + errsAll.slice(0, 3).join('｜') : '');
  checks.push({ id: 'V2', name: '图元结构合法（类型/必填字段/id 唯一/layer 存在）', pass: v2, metric: v2m });

  // V3 视口可见性
  var canvasBox = { x: 0, y: 0, w: cv.width, h: cv.height };
  var outside = [];
  var partial = 0;
  var visible = 0;
  for (var v = 0; v < proj.shapes.length; v++) {
    var box = shapeBBox(proj.shapes[v]);
    var layerDef = findLayer(proj, proj.shapes[v].layer);
    var layerVisible = !layerDef || layerDef.visible !== false;
    if (!layerVisible) continue;
    visible++;
    if (!bboxOverlaps(box, canvasBox)) outside.push(proj.shapes[v].id);
    else if (!bboxContains(canvasBox, box, TOL_ROUND)) partial++;
  }
  var v3 = outside.length === 0 && visible > 0;
  var v3m = proj.shapes.length === 0 ? '画板为空（尚无任何图元）'
    : ('可见图元 ' + visible + '/' + proj.shapes.length + '；完全在画板外 ' + outside.length +
      (outside.length ? '（' + outside.slice(0, 3).join(', ') + '）' : '') + '；部分越界 ' + partial + '（提示项）');
  checks.push({ id: 'V3', name: '视口可见性（无图元完全落在画板外）', pass: v3, metric: v3m });

  // V4 SVG 往返一致
  var svgText = projectToSvg(proj);
  var back = svgToProject(svgText, {}).project;
  var diffs = compareProjects(proj, back);
  var v4 = diffs.length === 0;
  var v4m = v4 ? ('SVG ' + svgText.length + ' 字符，往返 ' + proj.shapes.length + ' 个图元逐字段一致（容差 ' + TOL_ROUND + '）')
    : ('差异 ' + diffs.length + ' 处｜' + diffs.slice(0, 3).join('｜'));
  checks.push({ id: 'V4', name: 'SVG 往返一致（导出再导入，几何/样式/id 逐字段比对）', pass: v4, metric: v4m });

  // V5 确定性
  var svg2 = projectToSvg(proj);
  var v5 = svgText === svg2;
  checks.push({
    id: 'V5',
    name: '确定性（两次导出位级一致）',
    pass: v5,
    metric: v5 ? ('两次导出同为 ' + svgText.length + ' 字符，逐字节一致') : '两次导出不一致（存在非确定性来源）',
  });

  // V6 颜色与对比度
  var minContrast = o.minContrast === undefined || o.minContrast === null ? null : num(o.minContrast, 4.5);
  var badColors = [];
  var worst = null;
  var lowList = [];
  var textCount = 0;
  for (var c = 0; c < proj.shapes.length; c++) {
    var s6 = proj.shapes[c];
    if (s6.fill !== undefined && parseColor(s6.fill) === null) badColors.push(s6.id + '.fill');
    if (s6.stroke !== undefined && parseColor(s6.stroke) === null) badColors.push(s6.id + '.stroke');
    if (s6.type !== 'text') continue;
    textCount++;
    var fgRaw = parseColor(s6.fill === undefined ? '#000000' : s6.fill);
    var backdrop = textBackdrop(proj, s6);
    var threshold = minContrast === null ? (isLargeText(s6) ? 3.0 : 4.5) : minContrast;
    if (!fgRaw || fgRaw.none) {
      lowList.push({ id: s6.id, ratio: 0, threshold: threshold, note: 'fill 为 none（文字不可见）' });
      continue;
    }
    var fg = blendOver({ r: fgRaw.r, g: fgRaw.g, b: fgRaw.b, a: fgRaw.a * clamp(num(s6.opacity, 1), 0, 1), none: false, current: false },
      backdrop.color);
    var ratio = contrastRatio(fg, backdrop.color);
    if (worst === null || ratio < worst) worst = ratio;
    if (ratio < threshold) lowList.push({ id: s6.id, ratio: ratio, threshold: threshold, note: '' });
  }
  lowList.sort(function (a, b2) { return a.ratio - b2.ratio; });
  var v6 = badColors.length === 0 && lowList.length === 0;
  var v6m = '文本 ' + textCount + ' 个；颜色非法 ' + badColors.length +
    (badColors.length ? '（' + badColors.slice(0, 3).join(', ') + '）' : '') +
    '；最低对比度 ' + (worst === null ? 'n/a' : worst.toFixed(2) + ':1') +
    (lowList.length ? '｜低于阈值 ' + lowList.length + ' 处：' +
      lowList.slice(0, 3).map(function (z) { return z.id + ' ' + z.ratio.toFixed(2) + ':1<' + z.threshold + (z.note ? ' ' + z.note : ''); }).join('｜') : '');
  checks.push({ id: 'V6', name: '颜色合法 + 文本对比度达标（WCAG AA：4.5:1，大字 3:1）', pass: v6, metric: v6m });

  // V7 图层 / 层级 / 安全
  var lerrs = [];
  var lseen = {};
  for (var l = 0; l < proj.layers.length; l++) {
    var ly = proj.layers[l];
    if (!ly.id) lerrs.push('存在空图层 id');
    if (lseen[ly.id]) lerrs.push('图层 id 重复: ' + ly.id);
    lseen[ly.id] = true;
    var zs = {};
    var kids = shapesOfLayerInZOrder(proj, ly.id);
    for (var q = 0; q < kids.length; q++) {
      var z = num(kids[q].z, 0);
      if (zs[z]) lerrs.push('图层 ' + ly.id + ' 内 z 重复: ' + z);
      zs[z] = true;
    }
  }
  var secFlags = [];
  if (/<script/i.test(svgText)) secFlags.push('<script>');
  if (/\son[a-z]+\s*=/i.test(svgText)) secFlags.push('on* 事件属性');
  if (/(?:href|xlink:href)\s*=\s*["']?\s*javascript:/i.test(svgText)) secFlags.push('javascript: URL');
  var v7 = lerrs.length === 0 && secFlags.length === 0;
  var v7m = '图层 ' + proj.layers.length + '（id 唯一' + (lerrs.length ? '：' + lerrs.slice(0, 2).join('｜') : '') +
    '）；SVG 脚本面 ' + (secFlags.length ? secFlags.join(', ') : '无');
  checks.push({ id: 'V7', name: '图层与层级合法 + 产物无脚本面', pass: v7, metric: v7m });

  var passAll = true;
  for (var k2 = 0; k2 < checks.length; k2++) if (!checks[k2].pass) passAll = false;

  var lines = [];
  lines.push('## 画板校验（' + (passAll ? '✅ 全部通过' : '❌ 存在失败项') + '）');
  lines.push('');
  for (var p2 = 0; p2 < checks.length; p2++) {
    lines.push('  ' + (checks[p2].pass ? '✅' : '❌') + ' ' + checks[p2].id + ' ' + checks[p2].name);
    lines.push('      指标: ' + checks[p2].metric);
  }
  return { checks: checks, pass: passAll, report: lines.join('\n') };
}

// ── 工具实现 ───────────────────────────────────────────────
function argStr(args, key, def) {
  var v = args[key];
  if (v === undefined || v === null || v === '') return def;
  return String(v);
}

function argInt(args, key, def) {
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

function artProject(args, exec, ctx) {
  var path = argStr(args, 'path', 'art.project.json');
  var mode = argStr(args, 'mode', 'show');
  if (mode === 'create') {
    if (ctx.fs.exists(path) && !argBool(args, 'overwrite', false)) {
      return '画板工程已存在: ' + path + '（要重建请传 overwrite=true；改内容请用 art_edit）';
    }
    var proj = emptyProject(argStr(args, 'title', undefined), args.width, args.height);
    if (args.background !== undefined) proj.canvas.background = sanitizeColorValue(args.background, null, 'background');
    if (args.palette && args.palette.length) {
      var cols = [];
      for (var i = 0; i < args.palette.length; i++) {
        if (!parseColor(args.palette[i])) throw new Error('色板值无法解析: ' + args.palette[i]);
        cols.push(String(args.palette[i]));
      }
      proj.palette = cols;
    }
    if (args.layers && args.layers.length) {
      var lys = [];
      for (var k = 0; k < args.layers.length; k++) {
        var ly = args.layers[k] || {};
        lys.push({
          id: ly.id || ('l' + (k + 1)),
          name: ly.name || ('图层 ' + (k + 1)),
          visible: ly.visible === false ? false : true,
        });
      }
      proj.layers = lys;
    }
    saveProject(ctx, path, proj);
    return '✅ 已创建画板工程: ' + path + '\n\n' + projectSummary(proj, path);
  }
  if (mode === 'show') {
    var p1 = loadProject(ctx, path);
    return projectSummary(p1, path);
  }
  if (mode === 'update') {
    var p2 = loadProject(ctx, path);
    var ops = [];
    if (args.title) ops.push({ op: 'project.rename', title: args.title });
    if (args.width !== undefined || args.height !== undefined || args.background !== undefined) {
      ops.push({ op: 'set.canvas', width: args.width, height: args.height, background: args.background });
    }
    if (!ops.length) return '未提供任何要更新的字段（title / width / height / background 至少一个）';
    var log = applyOps(p2, ops);
    saveProject(ctx, path, p2);
    return '✅ 已更新画板: ' + path + '\n\n' + log.join('\n') + '\n\n' + projectSummary(p2, path);
  }
  throw new Error('未知 mode: ' + mode + '（可用 create / show / update）');
}

function artEdit(args, exec, ctx) {
  var path = argStr(args, 'path', 'art.project.json');
  var proj = loadProject(ctx, path);
  var ops = args.ops;
  if (args.op) {
    ops = [{
      op: args.op, ids: args.ids, id: args.id, type: args.type, layer: args.layer, name: args.name,
      all: args.all, ref: args.ref, newId: args.newId, moveTo: args.moveTo,
      x: args.x, y: args.y, w: args.w, h: args.h, r: args.r, cx: args.cx, cy: args.cy,
      rx: args.rx, ry: args.ry, x1: args.x1, y1: args.y1, x2: args.x2, y2: args.y2,
      points: args.points, d: args.d, text: args.text, fontSize: args.fontSize,
      fontWeight: args.fontWeight, anchor: args.anchor, fill: args.fill, stroke: args.stroke,
      strokeWidth: args.strokeWidth, opacity: args.opacity, dx: args.dx, dy: args.dy, to: args.to,
      scale: args.scale, about: args.about, set: args.set, axis: args.axis, z: args.z, mode: args.mode,
      index: args.index, visible: args.visible, background: args.background, width: args.width,
      height: args.height, colors: args.colors, title: args.title, viewBox: args.viewBox,
    }];
  }
  if (!ops || !ops.length) throw new Error('需要 ops 数组（或 op + 同层参数）');
  var log = applyOps(proj, ops);
  saveProject(ctx, path, proj);
  return '✅ 已应用 ' + ops.length + ' 条编辑命令到 ' + path + '\n\n' + log.join('\n') +
    '\n\n' + projectSummary(proj, path);
}

function artImport(args, exec, ctx) {
  var path = argStr(args, 'path', undefined);
  if (!path) throw new Error('需要 path（要导入的 SVG 文件）');
  var out = argStr(args, 'out', 'art.project.json');
  if (ctx.fs.exists(out) && !argBool(args, 'overwrite', false)) {
    return '目标工程已存在: ' + out + '（要覆盖请传 overwrite=true，或换 out 路径）';
  }
  var text = ctx.fs.readFile(path);
  var res = svgToProject(text, { title: argStr(args, 'title', undefined) });
  saveProject(ctx, out, res.project);
  var tail = '';
  if (res.warnings.length) {
    tail = '\n\n⚠️ 导入提示（' + res.warnings.length + ' 条）:\n  - ' + res.warnings.slice(0, 8).join('\n  - ');
  }
  return '✅ 已导入 SVG → ' + out + '\n来源: ' + path + '（' + String(text).length + ' 字符，' +
    res.project.shapes.length + ' 个图元）' + tail + '\n\n' + projectSummary(res.project, out);
}

function artExport(args, exec, ctx) {
  var path = argStr(args, 'path', 'art.project.json');
  var proj = loadProject(ctx, path);
  var fmt = argStr(args, 'format', 'svg');
  var out = argStr(args, 'out', '');
  if (fmt === 'png') {
    // 如实说明边界（不假装能做到）：沙箱无渲染器，PNG 由浏览器或后续 Node 桥导出
    return '⚠️ PNG 光栅化不在本插件能力内（goja 沙箱没有渲染器）。两条可行路径：\n' +
      '  1) 在「画板」面板点「导出 PNG」——浏览器 canvas 绘制 SVG 后落盘（零依赖，推荐）；\n' +
      '  2) 需要服务端批量出图时，走「服务端导出链」阶段（Node 桥 @resvg/resvg-js，MPL-2.0）。\n' +
      '现在也可直接导出矢量：art_export format=svg（SVG 无损、可再编辑、可被 read_image 视觉核对）。';
  }
  if (fmt !== 'svg') {
    throw new Error('未知 format: ' + fmt + '（本插件可用 svg；png 见返回说明）');
  }
  if (!out) out = 'art.svg';
  var svg = projectToSvg(proj);
  ctx.fs.writeFile(out, svg);
  proj.artifacts = proj.artifacts || {};
  proj.artifacts.svg = out;
  saveProject(ctx, path, proj);
  return '✅ 已导出 SVG → ' + out + '\n' + svg.length + ' 字符，' + proj.shapes.length + ' 个图元，' +
    proj.layers.length + ' 个图层（工程 ' + path + ' 已登记产物）\n' +
    '提示：可用 read_image 直接「看」产物核对渲染；PNG 请在「画板」面板导出。';
}

function artVerify(args, exec, ctx) {
  var path = argStr(args, 'path', 'art.project.json');
  var proj = loadProject(ctx, path);
  var res = verifyProject(ctx, proj, args);
  // ★ 旁挂校验报告（供「画板」面板展示）：不写进工程 JSON —— 工程是真相源，必须保持纯净可 diff
  var reportPath = argStr(args, 'report', 'art.verify.json');
  var reportErr = '';
  try {
    ctx.fs.writeFile(reportPath, JSON.stringify({ ts: nowIso(), project: path, pass: res.pass, checks: res.checks }, null, 2));
  } catch (e) {
    reportErr = '\n⚠️ 校验报告写入失败（' + reportPath + '）: ' + ((e && e.message) ? e.message : e) + '（不影响校验结论）';
  }
  var tail = res.pass ? '\n\n工程: ' + path + '（全部判据通过，可交付）' : '\n\n工程: ' + path + '（存在失败项——修完再导出）';
  return res.report + tail + '\n校验报告: ' + reportPath + '（供 UI 面板展示）' + reportErr;
}

// ── 工具声明 ───────────────────────────────────────────────
var TOOL_DEFS = [
  {
    name: 'art_project',
    description: '矢量画板工程管理（文本真相源 art.project.json）：创建/查看/更新画布尺寸、背景色、色板、图层。图元几何用绝对坐标 + 可选 2D 仿射矩阵 transform；SVG 是唯一文本产物（零依赖生成、可回读）。PNG 光栅化不在沙箱内（见 art_export）。',
    usageGuide: '创作第一步：mode=create 建画板（width/height/background，可带 layers=[{id,name}]）→ art_edit 加图元（shape.add）与调整（move/resize/set/align/distribute/z）→ art_export 出 SVG → art_verify 校验。mode=show 只看摘要，mode=update 改画布属性。已有 SVG 素材请用 art_import 导入。',
    category: '创作',
    parameters: {
      type: 'object',
      properties: {
        path: { type: 'string', description: '工程文件路径（默认 art.project.json；相对主项目根解析）' },
        mode: { type: 'string', description: 'create（新建）| show（默认，查看摘要）| update（改画布属性）' },
        title: { type: 'string', description: '可选：画板标题' },
        width: { type: 'integer', description: '可选：画布宽（像素，1..8192，默认 800）' },
        height: { type: 'integer', description: '可选：画布高（像素，1..8192，默认 600）' },
        background: { type: 'string', description: '可选：背景色（#RRGGBB / rgb() / 命名色 / none）' },
        palette: { type: 'array', description: '可选：色板（颜色字符串数组，默认 Tailwind 标准 10 色）' },
        layers: { type: 'array', description: '可选（仅 create）：图层定义 [{id,name,visible}]' },
        overwrite: { type: 'boolean', description: '可选（仅 create）：已存在时是否重建（默认 false）' },
      },
    },
  },
  {
    name: 'art_edit',
    description: '向画板工程追加编辑命令（命令式 op，与面板操作同源）。支持：图元（shape.add/remove/duplicate/move/resize/set/text/z）、排版（shape.align 六向对齐 + shape.distribute 等距分布）、图层（layer.add/remove/rename/visible/reorder）、画布与色板（set.canvas/set.palette）、标题（project.rename）。图元类型：rect/circle/ellipse/line/polyline/polygon/path/text。目标选择支持 ids / id / type / layer / name / all 过滤（不指定目标直接报错，避免误改全部）。',
    usageGuide: '示例：{"op":"shape.add","type":"rect","x":40,"y":40,"w":320,"h":180,"fill":"#2563EB","rx":8} / {"op":"shape.add","type":"text","x":40,"y":80,"text":"标题","fontSize":24,"fontWeight":"600","fill":"#111827"} / {"op":"shape.align","type":"rect","to":"hcenter"} / {"op":"shape.distribute","ids":["s1","s2","s3"],"axis":"h"} / move：{"op":"shape.move","ids":["s1"],"to":{"x":100,"y":60}} 或 dx/dy 相对偏移。改完用 art_verify 校验、art_export 出 SVG。',
    category: '创作',
    parameters: {
      type: 'object',
      properties: {
        path: { type: 'string', description: '可选：工程路径（默认 art.project.json）' },
        ops: { type: 'array', description: '编辑命令数组（按顺序应用）' },
        op: { type: 'string', description: '可选：单条 op 名（与同层参数合成为一条命令）' },
        ids: { type: 'array', description: '可选：目标图元 id 数组' },
        id: { type: 'string', description: '可选：单个目标（图元 id 或图层 id，按 op 语义）' },
        type: { type: 'string', description: '可选：按图元类型过滤目标 / shape.add 的图元类型' },
        fill: { type: 'string', description: '可选：填充色（none 表示不填充）' },
        stroke: { type: 'string', description: '可选：描边色' },
      },
    },
  },
  {
    name: 'art_import',
    description: '把 SVG 文本导入为画板工程（可 diff 的真相源）。解析 SVG 子集：svg 根尺寸/viewBox、g 分组（→ 图层，含 display:none → 隐藏图层）、rect/circle/ellipse/line/polyline/polygon/path/text、fill/stroke/stroke-width/opacity/font-size/font-weight/text-anchor、transform（translate/scale/rotate/matrix/skew 全支持，归一为矩阵）、style 内联声明。安全：<script>/<style> 一律丢弃，javascript: / url() 颜色值被剥离。不支持的元素（defs/use/image/gradient 等）会列入提示但不影响导入。',
    usageGuide: '把设计工具导出的 SVG 变成可编辑工程：art_import path=logo.svg out=art.project.json（out 已存在需 overwrite=true）→ 之后用 art_edit 改 → art_export 出 SVG → art_verify 校验。导入后如提示"部分越界"，多为原素材画布尺寸与内容不符，可用 set.canvas 调整。',
    category: '创作',
    parameters: {
      type: 'object',
      properties: {
        path: { type: 'string', description: '要导入的 SVG 文件路径（相对主项目根解析）' },
        out: { type: 'string', description: '可选：输出的工程路径（默认 art.project.json）' },
        title: { type: 'string', description: '可选：画板标题（默认取 SVG 的 <title>）' },
        overwrite: { type: 'boolean', description: '可选：目标工程已存在时是否覆盖（默认 false）' },
      },
      required: ['path'],
    },
  },
  {
    name: 'art_export',
    description: '由画板工程导出产物。format=svg（默认）：零依赖生成标准 SVG（含 width/height/viewBox/title/desc、按图层分组 <g>、固定属性顺序）。PNG 光栅化不在 goja 沙箱能力内（无渲染器）——本工具会返回两条可行路径（浏览器 canvas 导出 / 服务端 Node 桥导出链），而不是假装完成。',
    usageGuide: '导出前先 art_verify（V4 往返 / V5 确定性是核心契约）。SVG 产物可被 read_image 直接「看」来核对渲染，也可交回设计工具继续编辑（但改 SVG 不会自动回流工程——真相源始终是艺术工程 JSON）。',
    category: '创作',
    parameters: {
      type: 'object',
      properties: {
        path: { type: 'string', description: '可选：工程路径（默认 art.project.json）' },
        format: { type: 'string', description: '产物格式：svg（默认，矢量文本产物）；png 会返回替代路径说明（沙箱无渲染器）' },
        out: { type: 'string', description: '可选：输出路径（默认 art.svg）' },
      },
    },
  },
  {
    name: 'art_verify',
    description: '画板工程专项自检（7 项判据）：V1 画板合法（尺寸/viewBox/背景可解析）、V2 图元结构合法（类型/必填字段/正尺寸/id 唯一/layer 存在）、V3 视口可见性（无图元完全落在画板外；部分越界只作提示）、V4 SVG 往返一致（导出 SVG 再导入，几何/样式/id 逐字段比对，容差 1e-3）、V5 确定性（两次导出位级一致）、V6 颜色合法 + 文本对比度达标（WCAG AA，大字 3:1；背景取文本下方的实心图元或画板底色）、V7 图层与层级合法 + 产物无脚本面（无 <script>/on* 事件/javascript: URL）。任一 FAIL 说明产物与真相源不一致或存在设计缺陷，应视为构建失败。',
    usageGuide: '导出前跑一遍最省事；也可在 CI 里对工程文件跑（只读工程）。V4/V5 是本插件的核心契约；V6 用于挡住"看不见的文字"（低对比度），可用 minContrast 放宽阈值但建议保持 AA。结果同时写成旁挂报告（默认 art.verify.json）供「画板」面板展示——写报告不改工程本体。',
    category: '创作',
    parameters: {
      type: 'object',
      properties: {
        path: { type: 'string', description: '可选：工程路径（默认 art.project.json）' },
        report: { type: 'string', description: '可选：旁挂校验报告路径（默认 art.verify.json）' },
        minContrast: { type: 'number', description: '可选：文本对比度统一阈值（默认按 WCAG：4.5，大字 3.0）' },
      },
    },
  },
];

var IMPLS = {
  art_project: artProject,
  art_edit: artEdit,
  art_import: artImport,
  art_export: artExport,
  art_verify: artVerify,
};

// 双轨入口：goja 沙箱把本文件当函数体执行 → 顶层 return；
// Node 侧（测试/打包）走 module.exports。
var PLUGIN = {
  name: 'tool-art',
  purpose: '矢量/图像创作工具面（纯 goja 零依赖）：画板工程 JSON 文本真相源 + SVG 生成/解析/回读 + 六向对齐与等距分布 + 7 项自检（含 WCAG 对比度）—— 可 diff、产物可回读、位级可复现',
  // ★ logger 必须显式 inject：宿主按注入表装配 ctx 服务（jsplugin.go:1839 case "logger"），
  //   未声明时 ctx.logger 为 undefined —— 装载日志会静默丢失。
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
      // ★ 宿主 logger 是「服务工厂」：ctx.logger(scope) → {log,info,warn,…}
      if (ctx && typeof ctx.logger === 'function') {
        ctx.logger('tool-art').info('已注册 ' + TOOL_DEFS.length + ' 个工具（goja 轨 · 零依赖 · SVG 文本真相源）');
      }
    } catch (_) { /* ignore */ }
  },
};

if (typeof module !== 'undefined' && module.exports) module.exports = PLUGIN;
return PLUGIN;
