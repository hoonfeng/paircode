// tool-rig — 2D 角色创作域工具面（纯 goja 零依赖）
//
// 设计原则（对齐《创作域支持方案》§2.5 与**上游 .iki 格式契约**）：
//   · 真相源 = **真实 .iki 文档**（version 1），字段与 @ikijs/format 的 schema 逐项一致：
//       { version, name, canvas{width,height}, parameters[], textures?, parts[],
//         deformers?(matrix|warp), physics?(spring), physicsChains?(angular chain) }
//     因此本插件产出的 rig.model.iki.json **可直接被上游运行时装（@ikijs/engine, WebGL2）加载**，
//     不是"自家方言"（验收里用上游 loadIkiModel 做交叉校验）。
//   · 坐标口径照上游：模型空间原点 = 画布中心、**+y 向上**；rotation 单位度、**逆时针为正**。
//   · 「骨骼」= matrix deformer（pivot + transform + bindings 线性参数绑定）；「蒙皮」= part.deformer；
//     参数驱动走 binding.channel（translateX|translateY|rotate|scaleX|scaleY|opacity）——
//     部件的**线性绑定按上游语义**：参数值归一化到 0..1 后映射 [from,to]，
//     translate/rotate/scale **累加**到基变换，opacity **相乘**。
//   · 左右约定（照上游 / Live2D）：`*_L` = **角色自身左侧** = 屏幕 +x 侧。
//   · 输入侧：PSD 分层导入 —— 自研**最小 PSD 结构解析**（不读像素，只读图层名/位置/可见性/组），
//     二进制经 ctx.fs.readFileBase64 + 自研 base64 解码（沙箱无 Buffer，不依赖 atob 实现差异）。
//   · 产物全为**文本**：自包含可交互预览 HTML（自研 canvas 渲染器：deformer 树矩阵复合 +
//     线性绑定 + 弹簧/角链二级运动 + 参数滑块）/ 部件布局 SVG / 图层命名清单 JSON（交代美术）
//     / 旁挂校验报告 JSON。
//
// 沙箱能力：无 require / Buffer / Node API —— 一律走 ctx.fs（readFile / readFileBase64 / stat / writeFile）。

// ── 常量 ───────────────────────────────────────────────────
var IKI_VERSION = 1;                       // 上游 .iki 格式版本
var PROJECT_NAME = 'rig.model.iki.json';   // 默认工程文件（就是 .iki 文档）

var LIMIT_PARTS = 400;      // 部件数上限
var LIMIT_DEFORMERS = 240;  // deformer 数上限
var LIMIT_PARAMS = 64;      // 参数数上限
var LIMIT_PHYSICS = 48;     // 弹簧物理数上限
var LIMIT_CHAINS = 16;      // 角链物理数上限
var LIMIT_CHAIN_SEG = 16;   // 单链段数上限
var LIMIT_CANVAS = 8192;    // 画布边上限
var LIMIT_ORDER = 100000;   // 绘制顺序上限
var LIMIT_MESH_VERTS = 4096; // 单部件网格顶点上限
var LIMIT_PSD_BYTES = 32 << 20; // PSD 直接解析上限（32MB；更大的走图层清单导入）
var EPS = 1e-6;

// 标准参数表（**逐项对齐上游 StandardParameter**：id 一字不差，min/max/default 按上游语义）
var STD_PARAMS = [
  { id: 'ParamMouthOpenY', name: '嘴张开', min: 0, max: 1, default: 0, role: 'mouth' },
  { id: 'ParamMouthForm', name: '嘴形（笑/撇嘴）', min: -1, max: 1, default: 0, role: 'mouth' },
  { id: 'ParamEyeLOpen', name: '左眼开合', min: 0, max: 1, default: 1, role: 'eye_l' },
  { id: 'ParamEyeROpen', name: '右眼开合', min: 0, max: 1, default: 1, role: 'eye_r' },
  { id: 'ParamEyeBallX', name: '眼球左右', min: -1, max: 1, default: 0, role: 'eye' },
  { id: 'ParamEyeBallY', name: '眼球上下', min: -1, max: 1, default: 0, role: 'eye' },
  { id: 'ParamAngleX', name: '头部左右转', min: -30, max: 30, default: 0, role: 'head' },
  { id: 'ParamAngleY', name: '头部俯仰', min: -30, max: 30, default: 0, role: 'head' },
  { id: 'ParamAngleZ', name: '头部倾斜', min: -30, max: 30, default: 0, role: 'head' },
  { id: 'ParamBreath', name: '呼吸', min: 0, max: 1, default: 0, role: 'body' },
  { id: 'ParamBrowLY', name: '左眉上下', min: -1, max: 1, default: 0, role: 'brow_l' },
  { id: 'ParamBrowRY', name: '右眉上下', min: -1, max: 1, default: 0, role: 'brow_r' },
  { id: 'ParamBrowLAngle', name: '左眉倾斜', min: -1, max: 1, default: 0, role: 'brow_l' },
  { id: 'ParamBrowRAngle', name: '右眉倾斜', min: -1, max: 1, default: 0, role: 'brow_r' },
  // 两个物理输出参数（由弹簧/角链写，宿主不直接设 —— 上游注释亦如此）
  { id: 'ParamHairSwayX', name: '头发左右摆', min: -1, max: 1, default: 0, role: 'hair', physicsOut: true },
  { id: 'ParamHairSwayZ', name: '头发前后摆', min: -1, max: 1, default: 0, role: 'hair', physicsOut: true },
];

var STD_PARAM_IDS = (function () { var m = {}; for (var i = 0; i < STD_PARAMS.length; i++) m[STD_PARAMS[i].id] = true; return m; })();
var STD_PARAM_BY_ID = (function () { var m = {}; for (var i = 0; i < STD_PARAMS.length; i++) m[STD_PARAMS[i].id] = STD_PARAMS[i]; return m; })();

// 变换通道白名单（上游 IkiTransformChannel）
var CHANNELS = ['translateX', 'translateY', 'rotate', 'scaleX', 'scaleY', 'opacity'];
var MATRIX_CHANNELS = ['translateX', 'translateY', 'rotate', 'scaleX', 'scaleY'];

// 角色部位 role 表：`*_L` / `*_R` 是**角色自身**左右（屏幕 +x 侧 = 角色左）。
// 顺序即优先级（先匹配到的胜出）；每项 { role, re, bone, side }
//   bone = 自动绑定建议的 deformer id 词根（auto-rig 用它建树）
var ROLE_RULES = [
  { re: /(^|_)(front|back|side)?_?hair|ahoge|ponytail|twintail|braid|bang/, role: 'hair', bone: 'hair' },
  { re: /(^|_)head|(^|_)face($|_)|skull/, role: 'head', bone: 'head' },
  { re: /(^|_)eye(_|$)|eyeball|iris|pupil|eyewhite|eyelash|eye_lid/, role: 'eye', bone: 'head' },
  { re: /(^|_)brow|eyebrow/, role: 'brow', bone: 'head' },
  { re: /(^|_)mouth|lips|teeth|tongue|jaw|chin/, role: 'mouth', bone: 'head' },
  { re: /(^|_)ear($|_)|nose|cheek|blush/, role: 'face', bone: 'head' },
  { re: /(^|_)neck/, role: 'neck', bone: 'neck' },
  { re: /(^|_)(upper_?)?arm|shoulder|elbow|forearm|wrist/, role: 'arm', bone: 'arm' },
  { re: /(^|_)hand|finger|palm|thumb/, role: 'hand', bone: 'arm' },
  { re: /(^|_)(thigh|leg|knee|shin|calf|foot|toe|ankle|shoe|boot)/, role: 'leg', bone: 'leg' },
  { re: /(^|_)(cloth|skirt|dress|coat|cape|scarf|tie|ribbon|belt|sleeve|pants)/, role: 'cloth', bone: 'body' },
  { re: /(^|_)(tail|wing|horn|ear_?wing|accessor|weapon|sword|staff|prop)/, role: 'accessory', bone: 'body' },
  { re: /(^|_)(body|torso|trunk|chest|hip|waist|spine|back)/, role: 'body', bone: 'body' },
];

var HAIR_RE = ROLE_RULES[0].re;  // 头发（物理链候选）
var SWAY_ROLES = { hair: true, cloth: true, accessory: true }; // 二级运动候选部位

// ── 基础工具 ───────────────────────────────────────────────
function isNum(v) { return typeof v === 'number' && isFinite(v); }
function isInt(v) { return isNum(v) && Math.floor(v) === v; }
function isObj(v) { return v !== null && typeof v === 'object' && !(v instanceof Array); }
function isArr(v) { return Object.prototype.toString.call(v) === '[object Array]'; }
function clamp(v, lo, hi) { return v < lo ? lo : (v > hi ? hi : v); }
function nowIso() { return new Date().toISOString(); }
function num(v, def) {
  var n = (typeof v === 'number') ? v : parseFloat(v);
  return isNum(n) ? n : def;
}
function str(v, def) { return (v === undefined || v === null) ? def : String(v); }
function fmtNum(n) {
  var r = Math.round(num(n, 0) * 1000) / 1000;
  if (Math.abs(r) < 1e-9) r = 0;
  return String(r);
}
function deepClone(o) { return JSON.parse(JSON.stringify(o)); }
function copyObj(o) {
  if (!isObj(o)) return {};
  var r = {}, k;
  for (k in o) { if (Object.prototype.hasOwnProperty.call(o, k)) r[k] = o[k]; }
  return r;
}
function has(o, k) { return o !== null && o !== undefined && Object.prototype.hasOwnProperty.call(o, k); }

function argStr(args, key, def) {
  var v = args[key];
  if (v === undefined || v === null || v === '') return def;
  return String(v);
}
function argNum(args, key, def) {
  var v = args[key];
  if (v === undefined || v === null || v === '') return def;
  return num(v, def);
}
function argInt(args, key, def) {
  var v = args[key];
  if (v === undefined || v === null || v === '') return def;
  var n = (typeof v === 'number') ? Math.floor(v) : parseInt(v, 10);
  return isInt(n) ? n : def;
}
function argBool(args, key, def) {
  var v = args[key];
  if (v === undefined || v === null || v === '') return def;
  if (typeof v === 'boolean') return v;
  return v === 'true' || v === '1' || v === 1;
}
function slug(s) {
  return String(s === undefined || s === null ? '' : s)
    .toLowerCase()
    .replace(/[^\w]+/g, '_')
    .replace(/^_+|_+$/g, '');
}
function round3(n) { var r = Math.round(num(n, 0) * 1000) / 1000; return Math.abs(r) < 1e-9 ? 0 : r; }

// ── 二进制：base64 解码 + 字节读取器（沙箱无 Buffer，自研；不依赖 atob 的实现差异）──
var B64_CHARS = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/';
var B64_REV = (function () {
  var m = {};
  for (var i = 0; i < 64; i++) m[B64_CHARS.charAt(i)] = i;
  return m;
})();

// b64ToBytes：base64 文本 → Uint8Array（自动剥 data URI 前缀与空白；非法字符截断）
function b64ToBytes(s) {
  var clean = String(s === undefined || s === null ? '' : s).replace(/^data:[^,]*,/, '').replace(/[\s\r\n]+/g, '');
  // '=' 填充不计入有效字符
  var end = clean.length;
  while (end > 0 && clean.charAt(end - 1) === '=') end--;
  var out = new Uint8Array(Math.floor(end * 3 / 4) + 3);
  var oi = 0;
  for (var i = 0; i < end; i += 4) {
    var c0 = B64_REV[clean.charAt(i)];
    if (c0 === undefined) break;
    var c1 = i + 1 < end ? B64_REV[clean.charAt(i + 1)] : undefined;
    var c2 = i + 2 < end ? B64_REV[clean.charAt(i + 2)] : undefined;
    var c3 = i + 3 < end ? B64_REV[clean.charAt(i + 3)] : undefined;
    if (c1 === undefined) break;
    out[oi++] = (c0 << 2) | (c1 >> 4);
    if (c2 !== undefined) {
      out[oi++] = ((c1 & 15) << 4) | (c2 >> 2);
      if (c3 !== undefined) out[oi++] = ((c2 & 3) << 6) | c3;
    }
  }
  return out.subarray(0, oi);
}

// ByteReader：大端读取器（PSD 全大端）
function ByteReader(bytes) {
  this.b = bytes;
  this.p = 0;
  this.n = bytes.length;
}
ByteReader.prototype.need = function (k) {
  if (this.p + k > this.n) {
    throw new Error('数据在偏移 ' + this.p + ' 处截断：需要 ' + k + ' 字节，仅剩 ' + (this.n - this.p));
  }
  return this.p;
};
ByteReader.prototype.left = function () { return this.n - this.p; };
ByteReader.prototype.seek = function (to) {
  if (to < 0 || to > this.n) throw new Error('越界跳转: ' + to + '（数据长度 ' + this.n + '）');
  this.p = to;
};
ByteReader.prototype.skip = function (k) { this.need(k); this.p += k; };
ByteReader.prototype.u8 = function () { this.need(1); return this.b[this.p++]; };
ByteReader.prototype.i8 = function () { var v = this.u8(); return v > 127 ? v - 256 : v; };
ByteReader.prototype.u16 = function () { this.need(2); var v = (this.b[this.p] << 8) | this.b[this.p + 1]; this.p += 2; return v; };
ByteReader.prototype.i16 = function () { var v = this.u16(); return v > 32767 ? v - 65536 : v; };
ByteReader.prototype.u32 = function () {
  this.need(4);
  var b = this.b, p = this.p;
  this.p += 4;
  return ((b[p] * 16777216) + (b[p + 1] << 16) + (b[p + 2] << 8) + b[p + 3]);
};
ByteReader.prototype.i32 = function () {
  var v = this.u32();
  return v > 2147483647 ? v - 4294967296 : v;
};
ByteReader.prototype.bytes = function (k) { this.need(k); var r = this.b.subarray(this.p, this.p + k); this.p += k; return r; };
ByteReader.prototype.ascii = function (k) {
  var r = '', s = this.bytes(k);
  for (var i = 0; i < s.length; i++) r += String.fromCharCode(s[i]);
  return r;
};

// UTF-8 解码（自研，手写多字节；沙箱 TextDecoder 是 polyfill，行为差异不可控）
function utf8Of(bytes) {
  var out = '', i = 0, n = bytes.length;
  while (i < n) {
    var c = bytes[i++];
    if (c < 0x80) { out += String.fromCharCode(c); continue; }
    if (c >= 0xC2 && c < 0xE0 && i < n) {
      out += String.fromCharCode(((c & 0x1F) << 6) | (bytes[i++] & 0x3F));
      continue;
    }
    if (c >= 0xE0 && c < 0xF0 && i + 1 < n) {
      var c1 = bytes[i++], c2 = bytes[i++];
      out += String.fromCharCode(((c & 0x0F) << 12) | ((c1 & 0x3F) << 6) | (c2 & 0x3F));
      continue;
    }
    if (c >= 0xF0 && i + 2 < n) {
      var d1 = bytes[i++], d2 = bytes[i++], d3 = bytes[i++];
      var cp = ((c & 0x07) << 18) | ((d1 & 0x3F) << 12) | ((d2 & 0x3F) << 6) | (d3 & 0x3F);
      cp -= 0x10000;
      out += String.fromCharCode(0xD800 + (cp >> 10), 0xDC00 + (cp & 0x3FF));
      continue;
    }
    out += '\uFFFD';
  }
  return out;
}

// UTF-16BE 解码（PSD 的 'luni' Unicode 图层名）
function utf16beOf(bytes) {
  var out = '';
  for (var i = 0; i + 1 < bytes.length; i += 2) {
    var v = (bytes[i] << 8) | bytes[i + 1];
    if (v === 0) continue;
    out += String.fromCharCode(v);
  }
  return out;
}
// ── PSD 结构解析（只读结构，不读像素）────────────────────────
// 依据 Adobe Photoshop File Formats Specification（全大端）：
//   File Header(26B) → Color Mode Data(u32 len) → Image Resources(u32 len)
//   → Layer and Mask Info(u32 len) → Layer Info(u32 len)
//        [i16 layerCount][layer records][channel image data]
//      → Global Layer Mask Info(u32 len) → 附加图层信息
// 只解析：文件头 / 图层矩形 / 可见性 / 不透明度 / 混合模式 / 图层名（Pascal + 'luni' Unicode）
//         / 图层组（'lsct'：1=打开组 2=闭合组 3=组结束分隔符）
// 通道像素数据一律**跳过**（读长度后直接前进，不复制）—— 不解压 RLE、不做光栅化（那是外部链的事）。
//
// ★ 组归属算法（实测口径）：PSD 列表顺序自底向上；组的成员块位于其「组结束分隔符(lsct=3)」
//   之下（更小 index）且连续，组记录本身(lsct=1/2)紧跟在分隔符之上。
//   故：① 配对 (分隔符 k, 组记录 k+1)；② 组内容 = 从 k-1 向下收集，直到遇到另一个分隔符为止
//   （内层组的组记录会作为子项被收进父组，其自身内容由它自己的分隔符配对处理）。

var PSD_SIG = 0x38425053;      // '8BPS'
var PSD_SECTION_CLOSED = 2;
var PSD_SECTION_OPEN = 1;
var PSD_SECTION_DIVIDER = 3;   // 边界分隔符（组内容结束标记）
var PSD_FLAG_HIDDEN = 0x02;    // 图层 flags bit1 = 隐藏

// readPsdLayerRecord：单个图层记录（含图层名与 luni 附加块）
function readPsdLayerRecord(r) {
  var rec = {};
  rec.top = r.i32();
  rec.left = r.i32();
  rec.bottom = r.i32();
  rec.right = r.i32();
  var chCount = r.u16();
  if (chCount > 64) throw new Error('图层通道数异常: ' + chCount + '（PSD 规范上限 56）');
  rec.channels = [];
  var chBytes = 0;
  for (var c = 0; c < chCount; c++) {
    var cid = r.i16();
    var clen = r.u32();
    rec.channels.push({ id: cid, length: clen });
    chBytes += clen;
  }
  rec.channelBytes = chBytes;
  rec.blendSig = r.ascii(4);
  rec.blendKey = r.ascii(4);
  rec.opacity = r.u8();
  rec.clipping = r.u8();
  rec.flags = r.u8();
  r.u8(); // filler
  rec.visible = (rec.flags & PSD_FLAG_HIDDEN) === 0;
  rec.hidden = !rec.visible;
  var extraLen = r.u32();
  var extraEnd = r.p + extraLen;
  if (extraEnd > r.n) throw new Error('图层附加信息段越界（偏移 ' + r.p + ' 长度 ' + extraLen + ' > 文件 ' + r.n + '）');
  // ① 图层蒙版数据
  var maskLen = r.u32();
  r.skip(maskLen);
  // ② 混合范围
  var brLen = r.u32();
  r.skip(brLen);
  // ③ Pascal 图层名（长度含 1 字节长度字段，整体 4 字节对齐）
  var nameLen = r.u8();
  var nameBytes = r.bytes(nameLen);
  rec.name = utf8Of(nameBytes);
  var used = 1 + nameLen;
  var pad = (4 - (used % 4)) % 4;
  r.skip(pad);
  // ④ 附加信息块（'8BIM'/'8B64' + key + u32 长度；长度奇则补 1 字节）
  rec.unicodeName = '';
  rec.sectionType = 0;
  rec.sectionKeys = [];
  while (r.p + 12 <= extraEnd) {
    var blockStart = r.p;
    var sig = r.ascii(4);
    if (sig !== '8BIM' && sig !== '8B64') {
      // 非标准块：无法继续安全解析本层的附加块，跳到段尾
      break;
    }
    var key = r.ascii(4);
    var blen = r.u32();
    if (blockStart + 8 + blen > r.n) break; // 长度异常（越过文件尾）
    rec.sectionKeys.push(key);
    var dataStart = r.p;
    if (key === 'luni') {
      // ★ Unicode 图层名 = 4 字节字符数 + UTF-16BE 数据（长度前缀不能当字符，实测踩过）
      var uniChars = blen >= 4 ? r.u32() : 0;
      var take = Math.max(0, Math.min(blen - 4, uniChars * 2));
      rec.unicodeName = utf16beOf(r.bytes(take));
    } else if (key === 'lsct') {
      rec.sectionType = r.u32();
      if (blen >= 12 && r.p + 8 <= extraEnd) {
        r.skip(4);                    // 分节用的混合模式签名（'8BIM'）
        rec.sectionBlendKey = r.ascii(4);
      }
    } else if (key === 'lnsr') {
      rec.sectionName = utf8Of(r.bytes(blen)); // 分隔符图层的目标组名（可选）
    }
    // ★ 用块起点推进（读内容已前进过，绝不能再按 blen 跳一次 —— 曾经双重消耗导致后续块全被跳过）
    r.seek(dataStart + blen + (blen % 2));
  }
  r.seek(extraEnd);
  rec.displayName = rec.unicodeName || rec.name || '';
  return rec;
}

// parsePsd：解析 PSD 结构，返回 { header, layers(自底向上扁平)、roots(树)、warnings, stats }
function parsePsd(bytes) {
  var r = new ByteReader(bytes);
  var warnings = [];
  var header = {};
  if (r.u32() !== PSD_SIG) throw new Error('不是 PSD 文件（应以 8BPS 开头）');
  header.version = r.u16();
  if (header.version !== 1) {
    throw new Error('只支持 PSD（version 1）；读到 version=' + header.version + (header.version === 2 ? '（PSB 大文件格式：长度字段为 8 字节，本解析器不支持 —— 请在 Photoshop 里另存为 PSD）' : ''));
  }
  r.skip(6);
  header.channels = r.u16();
  header.height = r.u32();
  header.width = r.u32();
  header.depth = r.u16();
  header.colorMode = r.u16();
  if (header.width <= 0 || header.height <= 0 || header.width > 300000 || header.height > 300000) {
    throw new Error('PSD 画布尺寸异常: ' + header.width + 'x' + header.height);
  }
  var colorModeLen = r.u32();
  r.skip(colorModeLen);
  var resLen = r.u32();
  r.skip(resLen);

  var lmLen = r.u32();
  var recs = [];
  var lmLayers = 0;
  if (lmLen > 0 && r.left() >= 4) {
    var liLen = r.u32();
    var liEnd = r.p + liLen;
    var count = r.i16();
    var nLayers = count < 0 ? -count : count;
    if (count < 0) warnings.push('图层计数为负：表示文件含第一个 alpha 通道（已按绝对值 ' + nLayers + ' 处理）');
    lmLayers = nLayers;
    if (nLayers > 2000) throw new Error('图层数异常: ' + nLayers);
    for (var i = 0; i < nLayers; i++) recs.push(readPsdLayerRecord(r));
    // 通道图像数据：按记录里的长度顺序跳过（不解压）
    var dataStart = r.p;
    var skipped = 0;
    for (var j = 0; j < recs.length; j++) {
      var ch = recs[j].channels;
      for (var k = 0; k < ch.length; k++) {
        var take = Math.min(ch[k].length, Math.max(0, liEnd - r.p));
        r.skip(take);
        skipped += take;
      }
    }
    if (skipped !== recs.reduce(function (a, x) { return a + x.channelBytes; }, 0)) {
      warnings.push('通道数据长度与图层记录不一致（PSD 长度字段溢出保护已生效；不影响结构解析）');
    }
    if (r.p < liEnd) r.seek(liEnd);
    header.channelBytesSkipped = skipped;
    header.channelDataOffset = dataStart;
  } else {
    warnings.push('该 PSD 不含图层信息段（可能是拼合图像）—— 无分层可导入');
  }

  // ── 组树 ──
  // ★ 算法与 ag-psd（成熟开源实现）一致，并已用其写出的真实 PSD 交叉验证：
  //   列表顺序自底向上；**从列表末尾（最上层）向 index 0 遍历**：
  //     · 'lsct' 1/2（组记录）→ 入栈：其后（更靠下）的图层都归该组
  //     · 'lsct' 3（组结束分隔符）→ 出栈
  //     · 其余图层（含组记录自身）→ 挂到当前栈顶（根用 -1 表示）
  //   children 用 unshift 收集，得到自底向上顺序（= 绘制顺序）。
  var gi;
  for (gi = 0; gi < recs.length; gi++) {
    recs[gi].index = gi;
    recs[gi].children = [];
    recs[gi].parent = -1;
  }
  var rootKids = [];
  var stack = [-1];
  function pushChild(parentIdx, childIdx) {
    if (parentIdx < 0) rootKids.unshift(childIdx);
    else recs[parentIdx].children.unshift(childIdx);
  }
  for (gi = recs.length - 1; gi >= 0; gi--) {
    var Lg = recs[gi];
    if (Lg.sectionType === PSD_SECTION_OPEN || Lg.sectionType === PSD_SECTION_CLOSED) {
      Lg.isGroup = true;
      var par = stack[stack.length - 1];
      Lg.parent = par;
      pushChild(par, gi);
      stack.push(gi);
    } else if (Lg.sectionType === PSD_SECTION_DIVIDER) {
      Lg.parent = stack[stack.length - 1];
      if (stack.length > 1) stack.pop();
    } else {
      var par2 = stack[stack.length - 1];
      Lg.parent = par2;
      pushChild(par2, gi);
    }
  }
  // 深度 + 祖先可见性（Photoshop 语义：父组隐藏 → 整棵子树失效）
  function walk(idx, depth, ancVisible) {
    var L = recs[idx];
    L.depth = depth;
    L.effectiveVisible = ancVisible && L.visible;
    for (var i = 0; i < L.children.length; i++) walk(L.children[i], depth + 1, L.effectiveVisible);
  }
  var roots = rootKids.slice();
  for (var q = 0; q < roots.length; q++) walk(roots[q], 0, true);
  for (gi = 0; gi < recs.length; gi++) {
    if (recs[gi].depth === undefined) { recs[gi].depth = 0; recs[gi].effectiveVisible = false; }
  }
  // 绘制顺序：扁平链表自底向上（PSD 顺序即绘制顺序）
  var flat = [];
  for (var f = 0; f < recs.length; f++) flat.push(f);
  var leafCount = 0, groupCount = 0;
  for (var g = 0; g < recs.length; g++) {
    if (recs[g].isGroup) groupCount++;
    if (!recs[g].isGroup && recs[g].sectionType !== PSD_SECTION_DIVIDER) leafCount++;
  }
  return {
    header: header,
    layers: recs,          // 自底向上（index 小 = 靠后绘制）
    roots: roots,
    warnings: warnings,
    stats: { layerRecords: recs.length, leaves: leafCount, groups: groupCount },
  };
}

// psdLeafLayers：取可绘制图层（排除组记录与分隔符），按自底向上顺序。
function psdLeafLayers(psd) {
  var out = [];
  for (var i = 0; i < psd.layers.length; i++) {
    var L = psd.layers[i];
    if (L.isGroup || L.sectionType === PSD_SECTION_DIVIDER) continue;
    out.push(L);
  }
  return out;
}

// psdLayerGroupPath：图层的组路径（根 → 叶），用于命名与分类。
function psdGroupPath(psd, layer) {
  var path = [];
  var cur = layer.parent;
  var guard = 0;
  while (cur !== undefined && cur >= 0 && guard++ < 64) {
    path.unshift(psd.layers[cur].displayName);
    cur = psd.layers[cur].parent;
  }
  return path;
}

// psdRectToModel：PSD 矩形（左上原点、y 向下）→ 模型空间（画布中心原点、+y 向上）。
function psdRectToModel(rec, canvasW, canvasH) {
  var w = Math.max(0, rec.right - rec.left);
  var h = Math.max(0, rec.bottom - rec.top);
  return {
    x: round3((rec.left + rec.right) / 2 - canvasW / 2),
    y: round3(canvasH / 2 - (rec.top + rec.bottom) / 2),
    width: w,
    height: h,
  };
}
// ── 变换与渲染语义（逐行对齐上游 engine dist：affine.ts / deform.ts）──────────
//   · affine = [a, b, c, d, tx, ty]：x' = a·x + c·y + tx，y' = b·x + d·y + ty
//   · 参数绑定：t = (clamp(v) - min)/(max-min)；value = from + (to-from)·t；
//     translate/rotate/scale **累加**到基变换，opacity **相乘**
//   · part 世界矩阵 = (parent deformer world?) · T(x,y) · R(rot) · S(w·scaleX, h·scaleY)
//     （part 几何是 ±0.5 单位方 → 由 width/height 缩放；transform 是**相对其 deformer** 的局部变换）
//   · deformer 局部矩阵 = T(pivot) · T(x,y) · R(rot) · S(sx,sy) · T(-pivot)；world = parentWorld · local
function mA() { return [1, 0, 0, 1, 0, 0]; }

function mMul(a, b) {
  return [
    a[0] * b[0] + a[2] * b[1],
    a[1] * b[0] + a[3] * b[1],
    a[0] * b[2] + a[2] * b[3],
    a[1] * b[2] + a[3] * b[3],
    a[0] * b[4] + a[2] * b[5] + a[4],
    a[1] * b[4] + a[3] * b[5] + a[5],
  ];
}
function mTranslate(tx, ty) { return [1, 0, 0, 1, num(tx, 0), num(ty, 0)]; }
function mScale(sx, sy) { return [num(sx, 1), 0, 0, num(sy, 1), 0, 0]; }
function mRotate(deg) {
  var r = num(deg, 0) * Math.PI / 180;
  var c = Math.cos(r), s = Math.sin(r);
  return [c, s, -s, c, 0, 0];
}
function mApply(m, x, y) {
  return { x: m[0] * x + m[2] * y + m[4], y: m[1] * x + m[3] * y + m[5] };
}

// ParamStore：参数值存储（clamp 取值 + 归一化；default 先按 [min,max] 夹取，照上游语义）
function ParamStore(model) {
  this.defs = {};
  this.order = [];
  this.values = {};
  this.defaults = {};
  var ps = (model && model.parameters) || [];
  for (var i = 0; i < ps.length; i++) {
    var p = ps[i];
    if (!p || p.id === undefined) continue;
    this.defs[p.id] = p;
    this.order.push(p.id);
    var d = clamp(num(p.default, 0), num(p.min, 0), num(p.max, 1));
    this.defaults[p.id] = d;
    this.values[p.id] = d;
  }
}
ParamStore.prototype.hasParam = function (id) { return has(this.defs, id); };
ParamStore.prototype.def = function (id) { return this.defs[id] || null; };
ParamStore.prototype.get = function (id) {
  var p = this.defs[id];
  if (!p) return 0;
  var v = has(this.values, id) ? num(this.values[id], 0) : 0;
  return clamp(v, num(p.min, 0), num(p.max, 1));
};
ParamStore.prototype.normalized = function (id) {
  var p = this.defs[id];
  if (!p) return 0;
  var mn = num(p.min, 0), mx = num(p.max, 1);
  if (mx === mn) return 0;
  return (this.get(id) - mn) / (mx - mn);
};
ParamStore.prototype.set = function (id, v) { this.values[id] = num(v, 0); };
ParamStore.prototype.setDefault = function (id) { this.values[id] = has(this.defaults, id) ? this.defaults[id] : 0; };
ParamStore.prototype.apply = function (map) {
  if (!isObj(map)) return;
  for (var k in map) { if (Object.prototype.hasOwnProperty.call(map, k)) this.values[k] = num(map[k], 0); }
};
ParamStore.prototype.reset = function () {
  for (var k in this.defaults) { if (Object.prototype.hasOwnProperty.call(this.defaults, k)) this.values[k] = this.defaults[k]; }
};
ParamStore.prototype.snapshot = function () { return copyObj(this.values); };

// evaluateTransform：基变换 + 线性绑定（照上游 deform.ts evaluateTransform）
function evaluateTransform(transform, bindings, store) {
  var base = transform || {};
  var res = {
    x: num(base.x, 0),
    y: num(base.y, 0),
    rotation: num(base.rotation, 0),
    scaleX: has(base, 'scaleX') ? num(base.scaleX, 1) : 1,
    scaleY: has(base, 'scaleY') ? num(base.scaleY, 1) : 1,
    opacity: has(base, 'opacity') ? num(base.opacity, 1) : 1,
  };
  var bs = bindings || [];
  for (var i = 0; i < bs.length; i++) {
    var b = bs[i];
    var t = store.normalized(b.parameter);
    var v = num(b.from, 0) + (num(b.to, 0) - num(b.from, 0)) * t;
    if (b.channel === 'translateX') res.x += v;
    else if (b.channel === 'translateY') res.y += v;
    else if (b.channel === 'rotate') res.rotation += v;
    else if (b.channel === 'scaleX') res.scaleX += v;
    else if (b.channel === 'scaleY') res.scaleY += v;
    else if (b.channel === 'opacity') res.opacity *= v;
  }
  return res;
}

function deformerLocalMatrix(d, store) {
  var t = evaluateTransform(d.transform, d.bindings, store);
  var trs = mMul(mMul(mTranslate(t.x, t.y), mRotate(t.rotation)), mScale(t.scaleX, t.scaleY));
  var px = num(d.pivot && d.pivot.x, 0), py = num(d.pivot && d.pivot.y, 0);
  return mMul(mMul(mTranslate(px, py), trs), mTranslate(-px, -py));
}

// resolveDeformerWorlds：矩阵 deformer 世界矩阵（父在先，缓存 + 环检测）
function resolveDeformerWorlds(deformers, store) {
  var byId = {};
  var list = [];
  for (var i = 0; i < (deformers || []).length; i++) {
    var d = deformers[i];
    if (d.kind === undefined || d.kind === 'matrix') { byId[d.id] = d; list.push(d); }
  }
  var cache = {};
  function resolve(d, depth) {
    if (has(cache, d.id)) return cache[d.id];
    if (depth > 64) throw new Error('deformer 父链过深（疑似环）: ' + d.id);
    var local = deformerLocalMatrix(d, store);
    var world;
    if (d.parent === undefined || d.parent === null || d.parent === '') {
      world = local;
    } else {
      var par = byId[d.parent];
      if (!par) throw new Error('deformer 父节点未解析: "' + d.parent + '"（模型未规范化？）');
      world = mMul(resolve(par, depth + 1), local);
    }
    cache[d.id] = world;
    return world;
  }
  for (var k = 0; k < list.length; k++) resolve(list[k], 0);
  return cache;
}

// partWorldMatrix：部件世界矩阵（含 deformer 继承）
function partWorldMatrix(part, worlds, store) {
  var t = evaluateTransform(part.transform, part.bindings, store);
  var inner = mMul(mMul(mTranslate(t.x, t.y), mRotate(t.rotation)), mScale(num(part.width, 0) * t.scaleX, num(part.height, 0) * t.scaleY));
  if (part.deformer !== undefined && part.deformer !== null && part.deformer !== '') {
    var w = worlds[part.deformer];
    // ★ 上游语义（engine drawPart）：warp deformer 只有网格、没有矩阵 —— 其子部件的几何
    //   等于 partAffine（模型空间）+ grid 形变，**不继承任何 matrix 父链**（故这里查不到就返回 inner）。
    if (w) return mMul(w, inner);
  }
  return inner;
}

// warpKeyformOffsets：1D warp 在当前参数值下的关键帧偏移（线性插值；无 warp 返回 null）
function warpKeyformOffsets(warps, store) {
  if (!isArr(warps) || !warps.length) return null;
  var w = warps[0];
  if (!w || !isArr(w.keyforms) || !w.keyforms.length) return null;
  var v = store.get(w.parameter);
  var ks = w.keyforms;
  if (v <= num(ks[0].value, 0)) return ks[0].offsets;
  if (v >= num(ks[ks.length - 1].value, 0)) return ks[ks.length - 1].offsets;
  for (var i = 0; i < ks.length - 1; i++) {
    var a = ks[i], b = ks[i + 1];
    if (v >= num(a.value, 0) && v <= num(b.value, 0)) {
      var span = num(b.value, 0) - num(a.value, 0);
      var t = span === 0 ? 0 : (v - num(a.value, 0)) / span;
      var out = [];
      for (var k = 0; k < a.offsets.length; k++) {
        var av = num(a.offsets[k], 0), bv = num(b.offsets[k], 0);
        out[k] = av + (bv - av) * t;
      }
      return out;
    }
  }
  return null;
}

// gridSampleOffset：把 grid 控制点偏移双线性采样到模型空间点（与预览渲染同一公式 —— 校验与所见同源）
function gridSampleOffset(grid, offsets, pt) {
  var cols = Math.round(num(grid.cols, 1)), rows = Math.round(num(grid.rows, 1));
  if (cols < 1 || rows < 1 || !isArr(grid.points) || grid.points.length < (cols + 1) * (rows + 1) * 2) return { x: 0, y: 0 };
  var x0 = num(grid.points[0], 0), x1 = num(grid.points[cols * 2], 0);
  var y1 = num(grid.points[1], 0), y0 = num(grid.points[rows * (cols + 1) * 2 + 1], 0);
  var u = clamp((pt.x - x0) / ((x1 - x0) || 1), 0, 1);
  var v = clamp((pt.y - y0) / ((y1 - y0) || 1), 0, 1);
  var cx = u * cols, cy = (1 - v) * rows;
  var c0 = Math.min(Math.floor(cx), cols - 1), r0 = Math.min(Math.floor(cy), rows - 1);
  var fx = cx - c0, fy = cy - r0;
  function off(c, r) { var idx = (r * (cols + 1) + c) * 2; return { x: num(offsets[idx], 0), y: num(offsets[idx + 1], 0) }; }
  var o00 = off(c0, r0), o10 = off(c0 + 1, r0), o01 = off(c0, r0 + 1), o11 = off(c0 + 1, r0 + 1);
  return {
    x: (o00.x * (1 - fx) + o10.x * fx) * (1 - fy) + (o01.x * (1 - fx) + o11.x * fx) * fy,
    y: (o00.y * (1 - fx) + o10.y * fx) * (1 - fy) + (o01.y * (1 - fx) + o11.y * fx) * fy,
  };
}

// partQuad：部件四角（单位方 ±0.5 → 世界坐标；顺序 LT, RT, RB, LB）
//   传 model 时，warp 子部件的四角会再经 grid 形变 —— 与预览渲染同源。
function partQuad(part, worlds, store, model) {
  var m = partWorldMatrix(part, worlds, store);
  var pts = [mApply(m, -0.5, 0.5), mApply(m, 0.5, 0.5), mApply(m, 0.5, -0.5), mApply(m, -0.5, -0.5)];
  if (model && part.deformer) {
    var ds = model.deformers || [];
    for (var i = 0; i < ds.length; i++) {
      if (ds[i].id === part.deformer && ds[i].kind === 'warp') {
        var offs = warpKeyformOffsets(ds[i].warps, store);
        if (offs) {
          for (var k = 0; k < 4; k++) {
            var o = gridSampleOffset(ds[i].grid, offs, pts[k]);
            pts[k] = { x: pts[k].x + o.x, y: pts[k].y + o.y };
          }
        }
        break;
      }
    }
  }
  var t = evaluateTransform(part.transform, part.bindings, store);
  return { pts: pts, opacity: clamp(t.opacity, 0, 1), matrix: m };
}

function quadAABB(q) {
  var xs = [q.pts[0].x, q.pts[1].x, q.pts[2].x, q.pts[3].x];
  var ys = [q.pts[0].y, q.pts[1].y, q.pts[2].y, q.pts[3].y];
  return { x0: Math.min(xs[0], xs[1], xs[2], xs[3]), x1: Math.max(xs[0], xs[1], xs[2], xs[3]), y0: Math.min(ys[0], ys[1], ys[2], ys[3]), y1: Math.max(ys[0], ys[1], ys[2], ys[3]) };
}
// quadArea：四边形面积（拆两个三角形；用于重叠/退化判定）
function quadArea(q) {
  var p = q.pts;
  function tri(a, b, c) { return Math.abs((b.x - a.x) * (c.y - a.y) - (c.x - a.x) * (b.y - a.y)) / 2; }
  return tri(p[0], p[1], p[2]) + tri(p[0], p[2], p[3]);
}
// aabbOverlapRatio：两个 AABB 交叠面积 / 较小者面积（0 表示不叠）
function aabbOverlapRatio(a, b) {
  var w = Math.min(a.x1, b.x1) - Math.max(a.x0, b.x0);
  var h = Math.min(a.y1, b.y1) - Math.max(a.y0, b.y0);
  if (w <= 0 || h <= 0) return 0;
  var inter = w * h;
  var sa = (a.x1 - a.x0) * (a.y1 - a.y0);
  var sb = (b.x1 - b.x0) * (b.y1 - b.y0);
  var small = Math.min(sa, sb);
  return small <= EPS ? 0 : inter / small;
}

// ── 模型 IO / 规范化 ───────────────────────────────────────
function readTextFile(ctx, path, what) {
  if (!ctx.fs.exists(path)) throw new Error((what || '文件') + '不存在: ' + path);
  try { return ctx.fs.readFile(path); } catch (e) { throw new Error('读取' + (what || '文件') + '失败: ' + ((e && e.message) || e)); }
}
function readJsonFile(ctx, path, what) {
  var txt = readTextFile(ctx, path, what);
  try { return JSON.parse(txt); } catch (e) { throw new Error((what || '文件') + ' 不是合法 JSON（' + path + '）: ' + ((e && e.message) || e)); }
}
function writeJsonFile(ctx, path, obj) {
  ctx.fs.writeFile(path, JSON.stringify(obj, null, 2) + '\n');
}
function writeTextFile(ctx, path, text) {
  ctx.fs.writeFile(path, text);
}

function stdParameterDefs() {
  var out = [];
  for (var i = 0; i < STD_PARAMS.length; i++) {
    var s = STD_PARAMS[i];
    out.push({ id: s.id, name: s.name, min: s.min, max: s.max, default: s.default }); // 只写契约字段
  }
  return out;
}

// emptyModel：空角色模型（含 16 个标准参数；可选只带用到的参数）
function emptyModel(name, w, h, withStdParams) {
  var m = {
    version: IKI_VERSION,
    name: name || 'character',
    canvas: { width: w, height: h },
    parameters: withStdParams === false ? [] : stdParameterDefs(),
    parts: [],
  };
  return m;
}

// normalizeModel：补齐可选字段 + 数值化 + parts 按 order 稳定排序（不改用户意图）
function normalizeModel(raw) {
  if (!isObj(raw)) throw new Error('模型不是 JSON 对象');
  var m = deepClone(raw);
  m.version = isNum(m.version) ? m.version : IKI_VERSION;
  m.name = str(m.name, 'character');
  if (!isObj(m.canvas)) m.canvas = { width: 1000, height: 1000 };
  m.canvas.width = round3(num(m.canvas.width, 1000));
  m.canvas.height = round3(num(m.canvas.height, 1000));
  if (!isArr(m.parameters)) m.parameters = [];
  m.parameters = m.parameters.filter(function (p) { return isObj(p) && p.id !== undefined; }).map(function (p) {
    return {
      id: String(p.id),
      name: p.name === undefined ? undefined : String(p.name),
      min: round3(num(p.min, 0)),
      max: round3(num(p.max, 1)),
      default: round3(num(p.default, 0)),
    };
  });
  if (!isArr(m.parts)) m.parts = [];
  m.parts = m.parts.filter(isObj).map(function (p) {
    var o = {
      id: String(p.id === undefined ? 'part' : p.id),
      color: isArr(p.color) && p.color.length === 4 ? [num(p.color[0], 1), num(p.color[1], 1), num(p.color[2], 1), num(p.color[3], 1)] : [0.85, 0.85, 0.9, 1],
      width: round3(num(p.width, 0)),
      height: round3(num(p.height, 0)),
      transform: {
        x: round3(num(p.transform && p.transform.x, 0)),
        y: round3(num(p.transform && p.transform.y, 0)),
      },
      order: isNum(p.order) ? Math.round(p.order) : 0,
    };
    if (p.transform && has(p.transform, 'rotation')) o.transform.rotation = round3(num(p.transform.rotation, 0));
    if (p.transform && has(p.transform, 'scaleX')) o.transform.scaleX = round3(num(p.transform.scaleX, 1));
    if (p.transform && has(p.transform, 'scaleY')) o.transform.scaleY = round3(num(p.transform.scaleY, 1));
    if (p.transform && has(p.transform, 'opacity')) o.transform.opacity = round3(num(p.transform.opacity, 1));
    if (isArr(p.bindings) && p.bindings.length) {
      o.bindings = p.bindings.filter(isObj).map(function (b) {
        return { parameter: String(b.parameter), channel: String(b.channel), from: round3(num(b.from, 0)), to: round3(num(b.to, 0)) };
      });
    }
    if (p.deformer !== undefined && p.deformer !== null && p.deformer !== '') o.deformer = String(p.deformer);
    if (isObj(p.texture)) {
      o.texture = { index: Math.round(num(p.texture.index, 0)), uv: { x: num(p.texture.uv && p.texture.uv.x, 0), y: num(p.texture.uv && p.texture.uv.y, 0), width: num(p.texture.uv && p.texture.uv.width, 1), height: num(p.texture.uv && p.texture.uv.height, 1) } };
    }
    if (isObj(p.mesh)) {
      o.mesh = { vertices: (p.mesh.vertices || []).map(function (v) { return round3(num(v, 0)); }), uvs: (p.mesh.uvs || []).map(function (v) { return round3(num(v, 0)); }), indices: (p.mesh.indices || []).map(function (v) { return Math.round(num(v, 0)); }) };
    }
    if (isArr(p.warps) && p.warps.length) {
      o.warps = p.warps.filter(isObj).map(function (w) {
        return {
          parameter: String(w.parameter),
          keyforms: (w.keyforms || []).filter(isObj).map(function (k) { return { value: round3(num(k.value, 0)), offsets: (k.offsets || []).map(function (x) { return round3(num(x, 0)); }) }; }),
        };
      });
    }
    if (isObj(p.clip) && isArr(p.clip.masks)) o.clip = { masks: p.clip.masks.map(String) };
    return o;
  }).sort(function (a, b) { return a.order - b.order; });
  if (isArr(m.textures)) m.textures = m.textures.filter(isObj).map(function (t) { return { source: String(t.source) }; });
  if (isArr(m.deformers)) {
    m.deformers = m.deformers.filter(isObj).map(function (d) {
      if (d.kind === 'warp') {
        var g = d.grid || {};
        var o = { kind: 'warp', id: String(d.id), grid: { cols: Math.round(num(g.cols, 1)), rows: Math.round(num(g.rows, 1)), points: (g.points || []).map(function (v) { return round3(num(v, 0)); }) } };
        if (d.parent !== undefined && d.parent !== null && d.parent !== '') o.parent = String(d.parent);
        if (isArr(d.warps) && d.warps.length) {
          o.warps = d.warps.filter(isObj).map(function (w) {
            return { parameter: String(w.parameter), keyforms: (w.keyforms || []).filter(isObj).map(function (k) { return { value: round3(num(k.value, 0)), offsets: (k.offsets || []).map(function (x) { return round3(num(x, 0)); }) }; }) };
          });
        }
        if (isObj(d.warp2d)) {
          o.warp2d = {
            parameter: String(d.warp2d.parameter), parameterY: String(d.warp2d.parameterY),
            valuesX: (d.warp2d.valuesX || []).map(function (v) { return round3(num(v, 0)); }),
            valuesY: (d.warp2d.valuesY || []).map(function (v) { return round3(num(v, 0)); }),
            keyforms2d: (d.warp2d.keyforms2d || []).filter(isObj).map(function (k) { return { offsets: (k.offsets || []).map(function (x) { return round3(num(x, 0)); }) }; }),
          };
        }
        return o;
      }
      var mo = { id: String(d.id), pivot: { x: round3(num(d.pivot && d.pivot.x, 0)), y: round3(num(d.pivot && d.pivot.y, 0)) } };
      if (d.kind !== undefined && d.kind !== 'matrix') mo.kind = String(d.kind);
      if (d.parent !== undefined && d.parent !== null && d.parent !== '') mo.parent = String(d.parent);
      if (isObj(d.transform)) {
        mo.transform = { x: round3(num(d.transform.x, 0)), y: round3(num(d.transform.y, 0)) };
        if (has(d.transform, 'rotation')) mo.transform.rotation = round3(num(d.transform.rotation, 0));
        if (has(d.transform, 'scaleX')) mo.transform.scaleX = round3(num(d.transform.scaleX, 1));
        if (has(d.transform, 'scaleY')) mo.transform.scaleY = round3(num(d.transform.scaleY, 1));
      }
      if (isArr(d.bindings) && d.bindings.length) {
        mo.bindings = d.bindings.filter(isObj).map(function (b) { return { parameter: String(b.parameter), channel: String(b.channel), from: round3(num(b.from, 0)), to: round3(num(b.to, 0)) }; });
      }
      return mo;
    });
  }
  if (isArr(m.physics)) {
    m.physics = m.physics.filter(isObj).map(function (p) {
      return {
        id: String(p.id),
        input: { parameter: String(p.input && p.input.parameter), weight: round3(num(p.input && p.input.weight, 1)) },
        output: { parameter: String(p.output && p.output.parameter), scale: round3(num(p.output && p.output.scale, 1)) },
        mass: round3(num(p.mass, 1)), stiffness: round3(num(p.stiffness, 1)), damping: round3(num(p.damping, 0.1)),
      };
    });
  }
  if (isArr(m.physicsChains)) {
    m.physicsChains = m.physicsChains.filter(isObj).map(function (c) {
      var o = {
        id: String(c.id),
        anchorDeformer: String(c.anchorDeformer),
        gravity: { angle: round3(num(c.gravity && c.gravity.angle, -90)), strength: round3(num(c.gravity && c.gravity.strength, 1)) },
        segments: (c.segments || []).filter(isObj).map(function (s) {
          var so = { output: { parameter: String(s.output && s.output.parameter), scale: round3(num(s.output && s.output.scale, 1)) }, mass: round3(num(s.mass, 1)), stiffness: round3(num(s.stiffness, 1)), damping: round3(num(s.damping, 0.1)) };
          if (has(s, 'restAngle')) so.restAngle = round3(num(s.restAngle, 0));
          return so;
        }),
      };
      return o;
    });
  }
  return m;
}

function loadModel(ctx, path) {
  return normalizeModel(readJsonFile(ctx, path, '角色工程'));
}
function saveModel(ctx, path, model) {
  writeJsonFile(ctx, path, model);
}

// modelSummary：模型概要（工具输出/面板同源）
function modelSummary(model) {
  var orderGaps = 0, i;
  var orders = [];
  for (i = 0; i < model.parts.length; i++) orders.push(model.parts[i].order);
  orders.sort(function (a, b) { return a - b; });
  for (i = 0; i < orders.length; i++) { if (orders[i] !== i) { orderGaps = 1; break; } }
  var byRole = {};
  for (i = 0; i < model.parts.length; i++) {
    var r = classifyRole(model.parts[i].id, null).role;
    byRole[r] = (byRole[r] || 0) + 1;
  }
  return {
    name: model.name,
    canvas: model.canvas,
    parts: model.parts.length,
    deformers: (model.deformers || []).length,
    matrixDeformers: (model.deformers || []).filter(function (d) { return d.kind === undefined || d.kind === 'matrix'; }).length,
    warpDeformers: (model.deformers || []).filter(function (d) { return d.kind === 'warp'; }).length,
    parameters: model.parameters.length,
    physics: (model.physics || []).length,
    physicsChains: (model.physicsChains || []).length,
    textures: (model.textures || []).length,
    boundParts: model.parts.filter(function (p) { return p.deformer !== undefined; }).length,
    partsWithBindings: model.parts.filter(function (p) { return isArr(p.bindings) && p.bindings.length > 0; }).length,
    orderContiguous: !orderGaps,
    byRole: byRole,
  };
}
// ── 部件命名约定与角色分类（对齐上游 auto-rig 的 role layer 约定）──────────────
// `*_L` / `*_R` = **角色自身**左右（屏幕 +x 侧 = 角色左）；无后缀视为中轴部件。
// 分类结果用于：① 自动绑定（role → deformer 归属与参数绑定）② 命名规范校验 ③ 面板分组。
var SIDE_RE = /(?:^|_)(l|left)(?:_|$)|左/;
var SIDE_R_RE = /(?:^|_)(r|right)(?:_|$)|右/;
var EYEBALL_RE = /iris|pupil|eyeball|highlight|catchlight/;
var EYELID_RE = /eyelid|eye_lid|eyelash|lash|eyewhite|eye_white|eye($|_)/;

function detectSide(name) {
  var n = String(name || '').toLowerCase();
  if (SIDE_RE.test(n)) return 'L';
  if (SIDE_R_RE.test(n)) return 'R';
  return '';
}

// classifyRole：按命名（+组路径）判定部位 role。
function classifyRole(name, groupPath) {
  var n = String(name || '').toLowerCase();
  var side = detectSide(n);
  // 先看组路径（美术常在组上写部位）
  var hay = n;
  if (groupPath && groupPath.length) hay = groupPath.join('_').toLowerCase() + '_' + n;
  for (var i = 0; i < ROLE_RULES.length; i++) {
    var r = ROLE_RULES[i];
    if (r.re.test(hay)) {
      var role = r.role;
      if (role === 'eye' && EYEBALL_RE.test(n)) role = 'eyeball';
      if (role === 'eye' && EYELID_RE.test(n)) role = 'eyelid';
      if (role === 'eye' || role === 'eyeball' || role === 'eyelid') {
        return { role: role + (side ? '_' + side.toLowerCase() : ''), base: role, side: side, bone: r.bone };
      }
      if (role === 'brow' || role === 'arm' || role === 'hand' || role === 'leg') {
        return { role: role + (side ? '_' + side.toLowerCase() : ''), base: role, side: side, bone: r.bone };
      }
      return { role: role, base: role, side: side, bone: r.bone };
    }
  }
  return { role: 'unclassified', base: 'other', side: side, bone: '' };
}

// partRectAABB：一组部件的模型空间 AABB（用于推导 deformer pivot）
function partRectAABB(parts) {
  if (!parts || !parts.length) return null;
  var x0 = Infinity, y0 = Infinity, x1 = -Infinity, y1 = -Infinity;
  for (var i = 0; i < parts.length; i++) {
    var p = parts[i];
    var cx = num(p.transform && p.transform.x, 0), cy = num(p.transform && p.transform.y, 0);
    var w = num(p.width, 0), h = num(p.height, 0);
    x0 = Math.min(x0, cx - w / 2); x1 = Math.max(x1, cx + w / 2);
    y0 = Math.min(y0, cy - h / 2); y1 = Math.max(y1, cy + h / 2);
  }
  return { x0: x0, y0: y0, x1: x1, y1: y1, cx: (x0 + x1) / 2, cy: (y0 + y1) / 2, w: x1 - x0, h: y1 - y0 };
}

// 占位色：中性灰阶（无彩色），明度按部位分层 —— 仅用于无纹理时看结构，不是"设计配色"。
var ROLE_SHADE = {
  body: 0.62, head: 0.72, neck: 0.66, hair: 0.42, eyelid: 0.86, eyeball: 0.55,
  mouth: 0.78, face: 0.74, brow: 0.38, arm: 0.64, hand: 0.70, leg: 0.58,
  cloth: 0.50, accessory: 0.46, unclassified: 0.68,
};
function shadeFor(role) {
  var base = String(role || '').split('_')[0];
  var v = has(ROLE_SHADE, base) ? ROLE_SHADE[base] : 0.65;
  var l = clamp(v, 0.2, 0.92);
  return [round3(l), round3(l + 0.01 > 1 ? 1 : l + 0.01), round3(Math.min(1, l + 0.03)), 1];
}

// ── auto-rig：命名约定 → deformer 树 + 参数绑定 + 二级运动 ────────────────────
// 设计（对齐上游 .iki 的组合方式）：部件挂 matrix deformer（继承变换），
// 参数驱动分两级：**deformer 级**（头部/身体整体运动，挂在 deformer 的 bindings）
// 与**部件级**（眼睑/眼球/嘴/眉的局部形变，挂在 part.bindings）。
// 二级运动：头发 → physicsChains（多段角链，锚在头部）；衣摆/饰品 → physics（弹簧）。
// 无对应部件的 role 不生成任何节点（避免空 deformer 噪声）。
function ensureParam(model, id, def) {
  for (var i = 0; i < model.parameters.length; i++) {
    if (model.parameters[i].id === id) return false;
  }
  model.parameters.push({ id: id, name: def && def.name ? def.name : undefined, min: num(def && def.min, -1), max: num(def && def.max, 1), default: num(def && def.default, 0) });
  return true;
}

function buildAutoRig(model, opts) {
  opts = opts || {};
  var notes = [];
  var report = { byRole: {}, unclassified: [], notes: notes, partsBound: 0, partsWithParamBindings: 0 };
  var groups = {};
  var i, p;
  for (i = 0; i < model.parts.length; i++) {
    p = model.parts[i];
    var cls = classifyRole(p.id, p._groupPath || null);
    p._role = cls.role;
    p._side = cls.side;
    report.byRole[cls.role] = (report.byRole[cls.role] || 0) + 1;
    if (cls.role === 'unclassified') report.unclassified.push({ id: p.id, order: p.order, note: '命名未匹配部位词根 —— 已挂到 body，可用 rig_edit 的 part.rename / bind.set 修正' });
    if (!groups[cls.role]) groups[cls.role] = [];
    groups[cls.role].push(p);
  }
  function group(role) { return groups[role] || []; }
  function groupAny(roles) {
    var out = [];
    for (var k = 0; k < roles.length; k++) out = out.concat(group(roles[k]));
    return out;
  }

  var canvasW = num(model.canvas.width, 1000), canvasH = num(model.canvas.height, 1000);
  var defs = {};   // id → deformer 定义
  var order = [];  // 生成顺序（稳定性：同样输入 → 同样输出）
  // ★ deformer id 一律加 bone_ 前缀：.iki 里 part id 与 deformer id **共用同一命名空间**
  //   （上游 parseIkiModel 会报 deformers[i].id collides with parts[j].id —— 实测踩过：
  //    自动绑定生成的 head/neck/hair 与同名部件相撞）。前缀同时让"骨骼"语义更清楚。
  function boneId(x) { return 'bone_' + x; }
  var warpIds = {};
  for (var wi = 0; wi < (model.deformers || []).length; wi++) {
    if (model.deformers[wi].kind === 'warp') warpIds[model.deformers[wi].id] = true;
  }

  function addDeformer(id, parent, pivot) {
    var did = boneId(id);
    if (defs[did]) return defs[did];
    var d = { id: did, pivot: { x: round3(pivot.x), y: round3(pivot.y) } };
    if (parent) d.parent = boneId(parent);
    defs[did] = d;
    order.push(did);
    return d;
  }
  function bind(deformerId, param, channel, from, to) {
    var d = defs[boneId(deformerId)];
    if (!d) return;
    if (!d.bindings) d.bindings = [];
    for (var k = 0; k < d.bindings.length; k++) {
      if (d.bindings[k].parameter === param && d.bindings[k].channel === channel) return; // 幂等
    }
    d.bindings.push({ parameter: param, channel: channel, from: round3(from), to: round3(to) });
  }
  function attach(part, deformerId) {
    var d = defs[boneId(deformerId)];
    if (!d) return false;
    // 已挂在 warp deformer 上的部件保留手工挂载（warp 形变是美术精修成果，自动绑定不得覆盖）
    if (part.deformer && warpIds[part.deformer]) return false;
    part.deformer = boneId(deformerId);
    report.partsBound++;
    return true;
  }
  function partBind(part, param, channel, from, to) {
    if (!part.bindings) part.bindings = [];
    for (var k = 0; k < part.bindings.length; k++) {
      if (part.bindings[k].parameter === param && part.bindings[k].channel === channel) return;
    }
    part.bindings.push({ parameter: param, channel: channel, from: round3(from), to: round3(to) });
    report.partsWithParamBindings++;
    // ★ 上游语义：绑定值是**累加**到基变换上的（result.scaleY += value）。要让 from/to 表示
    //   最终缩放，必须把基变换的 scale 归零 —— 否则 1 + 0.2 会让"闭眼/张嘴"变成放大（实测抓到）。
    //   （位置类通道不用归零：x/y 本就是"基准位置 + 偏移"的语义。）
    if (channel === 'scaleX' || channel === 'scaleY') {
      if (!part.transform) part.transform = { x: 0, y: 0 };
      if (!num(part.transform[channel], 0)) part.transform[channel] = 0;
    }
  }
  function hasAny(roles) { return groupAny(roles).length > 0; }

  // ① 根 / 身体
  var bodyParts = groupAny(['body', 'unclassified']);
  var bodyBox = partRectAABB(bodyParts.length ? bodyParts : model.parts);
  var rootPivot = bodyBox ? { x: 0, y: 0 } : { x: 0, y: 0 };
  addDeformer('root', null, rootPivot);                       // 根：画布中心（模型空间原点）
  var bodyPivot = bodyBox ? { x: bodyBox.cx, y: bodyBox.cy } : { x: 0, y: 0 };
  addDeformer('body', 'root', bodyPivot);
  bind('body', 'ParamAngleX', 'translateX', 0, -num(opts.bodyFollowX, 6));
  bind('body', 'ParamBreath', 'scaleY', 0, num(opts.breathScale, 0.012));
  for (i = 0; i < bodyParts.length; i++) attach(bodyParts[i], 'body');

  // ② 脖子 / 头（头挂在身体上，pivot 取头部件包围盒底边中点——"颈部转轴"）
  var neckParts = group('neck'), headParts = group('head');
  var faceParts = groupAny(['face', 'eyelid', 'eyelid_l', 'eyelid_r', 'eyeball', 'eyeball_l', 'eyeball_r', 'brow', 'brow_l', 'brow_r', 'mouth']);
  var headAll = headParts.concat(faceParts);
  var headBox = partRectAABB(headAll);
  var neckBox = partRectAABB(neckParts);
  var headPivot = headBox ? { x: neckBox ? neckBox.cx : headBox.cx, y: (neckBox ? neckBox.cy + neckBox.h * 0.2 : headBox.y0) } : { x: 0, y: num(canvasH, 1000) * 0.25 };
  if (neckParts.length) {
    var neckPivot = neckBox ? { x: neckBox.cx, y: neckBox.y0 } : headPivot;
    addDeformer('neck', 'body', neckPivot);
    bind('neck', 'ParamAngleX', 'rotate', 0, num(opts.neckFollowX, 4));
    for (i = 0; i < neckParts.length; i++) attach(neckParts[i], 'neck');
    addDeformer('head', 'neck', headPivot);
  } else {
    addDeformer('head', 'body', headPivot);
  }
  bind('head', 'ParamAngleZ', 'rotate', -num(opts.headTilt, 8), num(opts.headTilt, 8));
  bind('head', 'ParamAngleX', 'translateX', num(opts.headTurnX, -20), num(opts.headTurnX, 20));
  bind('head', 'ParamAngleY', 'translateY', num(opts.headPitchY, -12), num(opts.headPitchY, 12));
  for (i = 0; i < headParts.length; i++) attach(headParts[i], 'head');
  for (i = 0; i < faceParts.length; i++) attach(faceParts[i], 'head');

  // ③ 面部局部绑定（部件级）
  var lids = groupAny(['eyelid_l', 'eyelid_r']), balls = groupAny(['eyeball_l', 'eyeball_r']);
  for (i = 0; i < lids.length; i++) {
    var lid = lids[i];
    var lidParam = lid._side === 'L' ? 'ParamEyeLOpen' : (lid._side === 'R' ? 'ParamEyeROpen' : 'ParamEyeLOpen');
    partBind(lid, lidParam, 'scaleY', num(opts.eyeClosedScale, 0.08), 1);
    if (lid._side === '') notes.push('眼睑部件 "' + lid.id + '" 未标 _L/_R —— 已按左眼参数绑定，请补命名（*_L / *_R = 角色自身左右）');
  }
  for (i = 0; i < balls.length; i++) {
    var ball = balls[i];
    // ★ 虹膜/瞳孔必须**与眼睑同参数一起闭合**：只绑眼白会在眨眼时留下"睁着的虹膜"
    //   （实测截图发现：眼白压扁了、虹膜还立着，看起来像翻白眼）。
    if (ball._side === 'L' || ball._side === 'R') {
      partBind(ball, ball._side === 'L' ? 'ParamEyeLOpen' : 'ParamEyeROpen', 'scaleY', num(opts.eyeClosedScale, 0.08), 1);
    }
    var dx = num(ball.width, 0) * num(opts.eyeballRange, 0.06);
    var dy = num(ball.height, 0) * num(opts.eyeballRange, 0.06);
    partBind(ball, 'ParamEyeBallX', 'translateX', -dx, dx);
    partBind(ball, 'ParamEyeBallY', 'translateY', -dy, dy);
  }
  var mouths = group('mouth');
  for (i = 0; i < mouths.length; i++) partBind(mouths[i], 'ParamMouthOpenY', 'scaleY', num(opts.mouthClosedScale, 0.12), 1);
  var brows = groupAny(['brow_l', 'brow_r', 'brow']);
  for (i = 0; i < brows.length; i++) {
    var br = brows[i];
    var byParam = br._side === 'R' ? 'ParamBrowRY' : 'ParamBrowLY';
    var baParam = br._side === 'R' ? 'ParamBrowRAngle' : 'ParamBrowLAngle';
    partBind(br, byParam, 'translateY', -num(br.height, 0) * 0.18, num(br.height, 0) * 0.18);
    partBind(br, baParam, 'rotate', -num(opts.browAngle, 6), num(opts.browAngle, 6));
  }

  // ④ 手臂 / 腿 / 手（各自 pivot 取"靠身体端"）
  var limbs = [
    { role: 'arm_l', id: 'arm_l', pivot: 'x0' }, { role: 'arm_r', id: 'arm_r', pivot: 'x1' },
    { role: 'hand_l', id: 'hand_l', piv: 'arm_l' }, { role: 'hand_r', id: 'hand_r', piv: 'arm_r' },
    { role: 'leg_l', id: 'leg_l', pivot: 'top' }, { role: 'leg_r', id: 'leg_r', pivot: 'top' },
  ];
  for (i = 0; i < limbs.length; i++) {
    var L = limbs[i];
    var ps = group(L.role);
    if (!ps.length) {
      // 手没有独立部件时，与手臂同名处理（手部件常直接叫 hand_L）
      continue;
    }
    var box = partRectAABB(ps);
    var pv;
    if (L.pivot === 'x0') pv = { x: box.x0, y: box.cy };
    else if (L.pivot === 'x1') pv = { x: box.x1, y: box.cy };
    else if (L.pivot === 'top') pv = { x: box.cx, y: box.y1 };
    else pv = { x: box.cx, y: box.cy };
    var parentId = (L.id === 'hand_l') ? 'arm_l' : (L.id === 'hand_r' ? 'arm_r' : 'body');
    addDeformer(L.id, defs[boneId(parentId)] ? parentId : 'body', pv);
    bind(L.id, 'ParamAngleZ', 'rotate', L.id.indexOf('_l') > 0 ? -num(opts.limbSway, 3) : num(opts.limbSway, 3), L.id.indexOf('_l') > 0 ? num(opts.limbSway, 3) : -num(opts.limbSway, 3));
    for (var m = 0; m < ps.length; m++) attach(ps[m], L.id);
  }

  // ⑤ 衣摆 / 饰品：弹簧（输入头部倾斜 → 输出自定义摆动参数）
  var clothParts = groupAny(['cloth', 'accessory']);
  if (clothParts.length) {
    var clothBox = partRectAABB(clothParts);
    ensureParam(model, 'ParamClothSwayX', { name: '衣物/饰品左右摆', min: -1, max: 1, default: 0 });
    addDeformer('cloth', 'body', { x: clothBox.cx, y: clothBox.y1 });
    bind('cloth', 'ParamClothSwayX', 'rotate', -num(opts.clothSway, 5), num(opts.clothSway, 5));
    for (i = 0; i < clothParts.length; i++) attach(clothParts[i], 'cloth');
    if (!model.physics) model.physics = [];
    // ★ 幂等：重复跑 bind.auto 不得堆积弹簧（否则模型每次都在变，破坏可 diff）
    for (i = 0; i < model.physics.length; i++) if (model.physics[i].id === 'ph_cloth') { model.physics.splice(i, 1); break; }
    model.physics.push({
      id: 'ph_cloth', input: { parameter: 'ParamAngleZ', weight: 1 },
      output: { parameter: 'ParamClothSwayX', scale: 1 },
      mass: 1, stiffness: 20, damping: 0.35,
    });
  }

  // ⑥ 头发：多段角链（锚在头部 deformer；输出标准参数 ParamHairSwayX/Z）
  var hairParts = group('hair');
  if (hairParts.length) {
    var hairBox = partRectAABB(hairParts);
    addDeformer('hair', 'head', { x: hairBox.cx, y: hairBox.y0 });
    bind('hair', 'ParamHairSwayX', 'rotate', -num(opts.hairSway, 6), num(opts.hairSway, 6));
    bind('hair', 'ParamHairSwayZ', 'scaleX', 0, num(opts.hairSwayZ, 0.04));
    for (i = 0; i < hairParts.length; i++) attach(hairParts[i], 'hair');
    if (!model.physicsChains) model.physicsChains = [];
    var segs = [];
    var segCount = clamp(argInt(opts, 'hairSegments', 2), 1, LIMIT_CHAIN_SEG);
    for (i = 0; i < segCount; i++) {
      segs.push({
        output: { parameter: i === 0 ? 'ParamHairSwayX' : 'ParamHairSwayZ', scale: i === 0 ? 1 : 0.7 },
        restAngle: 0, mass: 1, stiffness: i === 0 ? 14 : 10, damping: 0.3,
      });
    }
    for (i = 0; i < model.physicsChains.length; i++) if (model.physicsChains[i].id === 'chain_hair') { model.physicsChains.splice(i, 1); break; }
    model.physicsChains.push({
      id: 'chain_hair', anchorDeformer: boneId('head'),
      gravity: { angle: -90, strength: 1 },
      segments: segs,
    });
    if (segCount > 2) notes.push('角链段数 ' + segCount + ' 超过了标准输出参数个数（2）—— 多余段共享 ParamHairSwayZ，建议改为自定义参数');
  }

  // ⑦ 把生成的 deformer 并入模型（已存在的同 id 保留原定义，只补 bindings）
  if (!model.deformers) model.deformers = [];
  var existing = {};
  for (i = 0; i < model.deformers.length; i++) existing[model.deformers[i].id] = model.deformers[i];
  for (i = 0; i < order.length; i++) {
    var nd = defs[order[i]];
    var old = existing[nd.id];
    if (old) {
      if (old.pivot === undefined) old.pivot = nd.pivot;
      if (old.parent === undefined && nd.parent !== undefined) old.parent = nd.parent;
      if (nd.bindings) {
        if (!old.bindings) old.bindings = [];
        for (var bi = 0; bi < nd.bindings.length; bi++) {
          var dup = false;
          for (var bj = 0; bj < old.bindings.length; bj++) {
            if (old.bindings[bj].parameter === nd.bindings[bi].parameter && old.bindings[bj].channel === nd.bindings[bi].channel) dup = true;
          }
          if (!dup) old.bindings.push(nd.bindings[bi]);
        }
      }
    } else {
      model.deformers.push(nd);
    }
  }
  // 清理内部临时字段
  for (i = 0; i < model.parts.length; i++) { delete model.parts[i]._role; delete model.parts[i]._side; delete model.parts[i]._groupPath; }
  report.deformers = order.length;
  report.deformerIds = order;
  return { model: model, report: report };
}

// ── 输入侧：PSD / 图层清单 → parts ─────────────────────────
// PSD 坐标（左上原点、y 向下）→ 部件；两条导入路径共用同一转换，保证一致。
function layersToParts(layers, canvasW, canvasH, opts) {
  opts = opts || {};
  var parts = [];
  var used = {};
  var skippedHidden = 0, skippedEmpty = 0;
  var renameHints = [];
  var z = 0;
  for (var i = 0; i < layers.length; i++) {
    var L = layers[i];
    if (opts.visibleOnly !== false && L.effectiveVisible === false) { skippedHidden++; continue; }
    var rect = psdRectToModel(L, canvasW, canvasH);
    if (rect.width < 1 || rect.height < 1) { skippedEmpty++; continue; }
    var rawName = L.displayName || L.name || 'layer';
    var base = slug(rawName);
    if (!/^[a-z]/.test(base)) {
      // 中文/纯数字/纯符号名 → 给合规 id 并提示改名，否则 R2 命名判据会 FAIL 而用户不知道原因
      renameHints.push({ original: String(rawName), assigned: 'layer', note: '图层名不含 ASCII 词根 —— 已按 layer_N 命名；改成命名约定（小写词根 + _L/_R）才能自动绑定部位' });
      base = 'layer';
    }
    var id = base, n = 2;
    while (has(used, id)) { id = base + '_' + n; n++; }
    used[id] = true;
    var cls = classifyRole(id, L.groupPath || null);
    var part = {
      id: id,
      color: shadeFor(cls.role),
      width: rect.width,
      height: rect.height,
      transform: { x: rect.x, y: rect.y },
      order: z++,
    };
    if (opts.atlas) {
      part.texture = {
        index: argInt(opts, 'atlasIndex', 0),
        uv: { x: round3(L.left / opts.atlas.width), y: round3(L.top / opts.atlas.height), width: round3(rect.width / opts.atlas.width), height: round3(rect.height / opts.atlas.height) },
      };
    }
    parts.push(part);
  }
  return { parts: parts, skippedHidden: skippedHidden, skippedEmpty: skippedEmpty, renameHints: renameHints };
}

// importPsd：解析 PSD 字节 → parts（最小 PSD 结构解析，不读像素）
function importPsd(ctx, opts) {
  var path = argStr(opts, 'path', '');
  if (!path) throw new Error('缺少 src（PSD 源文件路径）：rig_import source=psd src=角色.psd');
  if (!ctx.fs.exists(path)) throw new Error('PSD 文件不存在: ' + path);
  var st = null;
  try { st = ctx.fs.stat(path); } catch (_) { st = null; }
  if (st && num(st.size, 0) > LIMIT_PSD_BYTES) {
    throw new Error('PSD 文件 ' + fmtNum(num(st.size, 0) / (1 << 20)) + 'MB 超过直接解析上限 ' + (LIMIT_PSD_BYTES >> 20) + 'MB（沙箱需整体读入内存）。' +
      '方案：① 在 Photoshop 里另存为较小的 PSD（去掉像素内容/拼合前先导出结构）；② 用「图层清单」路径导入 —— 把图层的名/位置/尺寸写成 JSON 后 rig_import source=manifest。');
  }
  var b64 = ctx.fs.readFileBase64(path);
  var bytes = b64ToBytes(b64);
  var psd = parsePsd(bytes);
  var leaves = psdLeafLayers(psd);
  var withPath = [];
  for (var i = 0; i < leaves.length; i++) {
    var L = leaves[i];
    L.groupPath = psdGroupPath(psd, L);
    withPath.push(L);
  }
  var conv = layersToParts(withPath, psd.header.width, psd.header.height, opts);
  return {
    psd: psd,
    parts: conv.parts,
    canvas: { width: psd.header.width, height: psd.header.height },
    skippedHidden: conv.skippedHidden,
    skippedEmpty: conv.skippedEmpty,
    renameHints: conv.renameHints,
    bytes: bytes.length,
  };
}

// importManifest：图层清单 JSON → parts（PSD 不可用/过大时的替代输入路径，格式与 PSD 解析输出同构）
function importManifest(ctx, opts) {
  var path = argStr(opts, 'path', '');
  if (!path) throw new Error('缺少 path（图层清单 JSON 路径）');
  var man = readJsonFile(ctx, path, '图层清单');
  if (!isArr(man.layers)) throw new Error('图层清单缺少 layers 数组：' + path);
  var canvasW = num(man.canvas && man.canvas.width, 0), canvasH = num(man.canvas && man.canvas.height, 0);
  if (!(canvasW > 0) || !(canvasH > 0)) throw new Error('图层清单缺少 canvas.width / canvas.height');
  var layers = [];
  for (var i = 0; i < man.layers.length; i++) {
    var L = man.layers[i];
    var left = num(L.x !== undefined ? L.x : L.left, 0);
    var top = num(L.y !== undefined ? L.y : L.top, 0);
    layers.push({
      left: left, top: top,
      right: left + num(L.width, 0), bottom: top + num(L.height, 0),
      displayName: str(L.name, 'layer'),
      effectiveVisible: L.visible === undefined ? true : !!L.visible,
      groupPath: isArr(L.group) ? L.group : (L.group ? String(L.group).split('/') : []),
    });
  }
  var conv = layersToParts(layers, canvasW, canvasH, opts);
  return { manifest: man, parts: conv.parts, canvas: { width: canvasW, height: canvasH }, skippedHidden: conv.skippedHidden, skippedEmpty: conv.skippedEmpty, renameHints: conv.renameHints };
}
// ── 工具① rig_model：角色工程（.iki 文档）的创建与查看 ─────────────────────
function rigModel(args, opts, ctx) {
  var path = argStr(args, 'path', PROJECT_NAME);
  var action = argStr(args, 'action', 'get');
  if (action === 'create') {
    var name = argStr(args, 'name', 'character');
    var cw = argInt(args, 'width', 0), ch = argInt(args, 'height', 0);
    var canvas = args.canvas || {};
    cw = argInt(args, 'width', argInt(canvas, 'width', 1000));
    ch = argInt(args, 'height', argInt(canvas, 'height', 1000));
    if (cw <= 0 || ch <= 0 || cw > LIMIT_CANVAS || ch > LIMIT_CANVAS) throw new Error('画布尺寸非法: ' + cw + 'x' + ch + '（1..' + LIMIT_CANVAS + '）');
    if (ctx.fs.exists(path) && !argBool(args, 'overwrite', false)) {
      return '工程已存在: ' + path + '（重建请 overwrite=true，查看请 action=get）';
    }
    var model = emptyModel(name, cw, ch, argBool(args, 'stdParams', true));
    // ★ 一次调用成型：create 时可带 parts=[…]（每项与 rig_add / rig_edit 的 part.add 同构）——
    //   手搭小部件集合时不必「先 create 再 N 次 rig_edit」；任一项非法整批不落盘。
    var added0 = applyPartSpecs(model, args.parts, 'parts');
    var out0 = normalizeModel(model);
    saveModel(ctx, path, out0);
    var resNew = {
      ok: true, action: 'created', path: path, name: out0.name,
      canvas: { width: cw, height: ch },
      params: out0.parameters.length, parts: out0.parts.length,
      summary: modelSummary(out0),
      next: 'rig_import source=psd src=<角色.psd>（自动分层 + 自动绑定），或 rig_add 批量加部件 / rig_edit 绑骨骼与参数（bind.auto / part.bind）',
    };
    if (added0.length) resNew.added = added0;
    return JSON.stringify(resNew, null, 2);
  }
  if (!ctx.fs.exists(path)) throw new Error('工程不存在: ' + path + '（先 action=create 或 rig_import）');
  var m = loadModel(ctx, path);
  if (action === 'get' || action === 'summary') {
    return JSON.stringify(modelSummary(m), null, 2);
  }
  if (action === 'rename') {
    var to = argStr(args, 'name', '');
    if (!to) throw new Error('rename 需要 name');
    m.name = to;
    saveModel(ctx, path, normalizeModel(m));
    return '已重命名: ' + to;
  }
  if (action === 'reset') {
    if (!argBool(args, 'force', false)) return 'reset 会清空 parts/deformers/physics（保留参数与画布）—— 确认请加 force=true';
    m.parts = [];
    m.deformers = [];
    m.physics = [];
    m.physicsChains = [];
    m.textures = [];
    saveModel(ctx, path, normalizeModel(m));
    return '已重置工程结构（参数 ' + m.parameters.length + ' 个保留）: ' + path;
  }
  throw new Error('未知 action: ' + action + '（可选 create / get / rename / reset）');
}

// ── 工具② rig_import：PSD / 图层清单 → parts（+ 可选自动绑定）────────────────
function rigImport(args, opts, ctx) {
  var path = argStr(args, 'path', PROJECT_NAME);
  var source = argStr(args, 'source', 'psd').toLowerCase();
  var srcPath = argStr(args, 'src', '');
  var mode = argStr(args, 'mode', 'replace').toLowerCase();
  if (mode !== 'replace' && mode !== 'append') throw new Error('mode 只支持 replace / append');
  var canvasOverride = args.canvas || null;
  var imp;
  if (source === 'psd' || source === 'psb') {
    imp = importPsd(ctx, {
      path: srcPath || argStr(args, 'psd', ''),
      visibleOnly: argBool(args, 'visibleOnly', true),
      atlas: isObj(args.atlas) ? { width: num(args.atlas.width, 0), height: num(args.atlas.height, 0) } : null,
      atlasIndex: argInt(args, 'atlasIndex', 0),
    });
  } else if (source === 'manifest' || source === 'json') {
    imp = importManifest(ctx, {
      path: srcPath || argStr(args, 'manifest', ''),
      visibleOnly: argBool(args, 'visibleOnly', true),
      atlas: isObj(args.atlas) ? { width: num(args.atlas.width, 0), height: num(args.atlas.height, 0) } : null,
      atlasIndex: argInt(args, 'atlasIndex', 0),
    });
  } else {
    throw new Error('source 只支持 psd / manifest（图层清单 JSON）');
  }
  var cw = canvasOverride ? argInt(canvasOverride, 'width', imp.canvas.width) : imp.canvas.width;
  var ch = canvasOverride ? argInt(canvasOverride, 'height', imp.canvas.height) : imp.canvas.height;
  if (cw > LIMIT_CANVAS || ch > LIMIT_CANVAS) throw new Error('画布尺寸超过上限 ' + LIMIT_CANVAS + '：' + cw + 'x' + ch);
  if (imp.parts.length > LIMIT_PARTS) throw new Error('图层数 ' + imp.parts.length + ' 超过部件上限 ' + LIMIT_PARTS + '（请先合并图层）');

  var model;
  if (ctx.fs.exists(path) && mode === 'append') {
    model = loadModel(ctx, path);
  } else if (ctx.fs.exists(path) && mode === 'replace') {
    if (!argBool(args, 'overwrite', false) && !argBool(args, 'force', false)) {
      return '工程已存在: ' + path + '（导入会覆盖现有 parts —— 确认请加 overwrite=true，或改 mode=append）';
    }
    model = loadModel(ctx, path);
  } else {
    model = emptyModel(argStr(args, 'name', 'character'), cw, ch, argBool(args, 'stdParams', true));
  }
  model.canvas = { width: cw, height: ch };
  if (mode === 'replace') {
    model.parts = [];
    model.deformers = [];
    model.physics = [];
    model.physicsChains = [];
  }
  var baseOrder = 0;
  for (var i = 0; i < model.parts.length; i++) baseOrder = Math.max(baseOrder, model.parts[i].order + 1);
  var idUsed = {};
  for (i = 0; i < model.parts.length; i++) idUsed[model.parts[i].id] = true;
  var added = [];
  for (i = 0; i < imp.parts.length; i++) {
    var p = imp.parts[i];
    var pid = p.id, n = 2;
    while (has(idUsed, pid)) { pid = p.id + '_' + n; n++; }
    idUsed[pid] = true;
    p.id = pid;
    p.order = baseOrder + p.order;
    model.parts.push(p);
    added.push({ id: pid, w: p.width, h: p.height, x: p.transform.x, y: p.transform.y, order: p.order });
  }
  model.parts = normalizeModel(model).parts; // 排序 + 契约清理

  var report = { imported: added.length, mode: mode, canvas: model.canvas, parts: added, byRole: {}, warnings: [] };
  for (i = 0; i < model.parts.length; i++) {
    var cls = classifyRole(model.parts[i].id, null);
    report.byRole[cls.role] = (report.byRole[cls.role] || 0) + 1;
  }
  if (imp.psd) {
    report.psd = {
      layers: imp.psd.stats.layerRecords, leaves: imp.psd.stats.leaves, groups: imp.psd.stats.groups,
      channels: imp.psd.header.channels, depth: imp.psd.header.depth, colorMode: imp.psd.header.colorMode,
      channelBytesSkipped: imp.psd.header.channelBytesSkipped, bytesRead: imp.bytes,
    };
    report.warnings = report.warnings.concat(imp.psd.warnings);
  }
  if (imp.skippedHidden) report.warnings.push('跳过隐藏图层 ' + imp.skippedHidden + ' 个（visibleOnly=false 可保留）');
  if (imp.skippedEmpty) report.warnings.push('跳过空矩形图层 ' + imp.skippedEmpty + ' 个');
  if (imp.renameHints && imp.renameHints.length) {
    report.renameHints = imp.renameHints.slice(0, 12);
    report.warnings.push('有 ' + imp.renameHints.length + ' 个图层名不是 ASCII 词根（中文/纯数字等）—— 已按 layer_N 命名：' +
      imp.renameHints.slice(0, 5).map(function (h) { return '「' + h.original + '」'; }).join(' ') +
      (imp.renameHints.length > 5 ? ' …' : '') + '。改成命名约定（小写词根 + _L/_R）后自动绑定才能识别部位。');
  }

  var auto = argBool(args, 'autoRig', true);
  if (auto) {
    var built = buildAutoRig(model, { hairSegments: argInt(args, 'hairSegments', 2) });
    model = built.model;
    report.autoRig = built.report;
  }
  model = normalizeModel(model);
  saveModel(ctx, path, model);
  return JSON.stringify(report, null, 2);
}

// ── 工具③ rig_edit：命令式编辑（一次 ops 批量应用，全有或全无）──────────────
function findParts(model, spec) {
  var out = [];
  var ids = isArr(spec.ids) ? spec.ids.map(String) : null;
  var single = spec.id !== undefined && spec.id !== null && spec.id !== '' ? String(spec.id) : '';
  var idList = ids || (single ? [single] : null);
  var roleFilter = spec.role !== undefined && spec.role !== null && spec.role !== '' ? String(spec.role) : '';
  var nameFilter = spec.name !== undefined && spec.name !== null && spec.name !== '' ? String(spec.name) : '';
  for (var i = 0; i < model.parts.length; i++) {
    var p = model.parts[i];
    if (idList && idList.indexOf(p.id) < 0) continue;
    if (roleFilter && classifyRole(p.id, null).role !== roleFilter) continue;
    if (nameFilter && p.id.indexOf(nameFilter) < 0) continue;
    out.push(p);
  }
  if (idList && out.length !== idList.length) {
    var found = {}, k;
    for (k = 0; k < out.length; k++) found[out[k].id] = true;
    var missing = [];
    for (k = 0; k < idList.length; k++) if (!found[idList[k]]) missing.push(idList[k]);
    throw new Error('部件不存在: ' + missing.join(', '));
  }
  if (!idList && !roleFilter && !nameFilter) throw new Error('该 op 需要指定目标：ids / id / role / name（不指定会误改整个模型）');
  if (!out.length) throw new Error('没有匹配的部件（条件: ' + JSON.stringify(spec) + '）');
  return out;
}
function findDeformer(model, id) {
  var ds = model.deformers || [];
  for (var i = 0; i < ds.length; i++) if (ds[i].id === id) return ds[i];
  return null;
}
function findParam(model, id) {
  for (var i = 0; i < model.parameters.length; i++) if (model.parameters[i].id === id) return model.parameters[i];
  return null;
}
function checkParamExists(model, id, where) {
  if (!findParam(model, id)) throw new Error(where + ' 引用了未声明的参数 "' + id + '"（先用 op=param.add 声明）');
}
function checkChannel(channel, where) {
  if (CHANNELS.indexOf(String(channel)) < 0) throw new Error(where + ' 的 channel 非法: "' + channel + '"（可选 ' + CHANNELS.join(' / ') + '）');
}
function applyBind(list, b, where, matrixOnly) {
  checkParamExists(list._model, b.parameter, where);
  checkChannel(b.channel, where);
  if (matrixOnly && b.channel === 'opacity') throw new Error(where + '：deformer 上不支持 opacity 通道（矩阵无法表达透明度）');
  if (!isNum(num(b.from, NaN)) || !isNum(num(b.to, NaN))) throw new Error(where + ' 需要 from / to 数值');
  if (!list.arr) list.arr = [];
  for (var i = 0; i < list.arr.length; i++) {
    if (list.arr[i].parameter === b.parameter && list.arr[i].channel === b.channel) {
      list.arr[i].from = round3(num(b.from, 0));
      list.arr[i].to = round3(num(b.to, 0));
      return 'updated';
    }
  }
  list.arr.push({ parameter: String(b.parameter), channel: String(b.channel), from: round3(num(b.from, 0)), to: round3(num(b.to, 0)) });
  return 'added';
}

// applyOp：单条编辑命令 → 直接改 model（抛错则调用方整批放弃）
function applyOp(model, spec, index) {
  if (!isObj(spec)) throw new Error('第 ' + (index + 1) + ' 条 op 不是对象');
  var op = argStr(spec, 'op', '');
  if (!op) throw new Error('第 ' + (index + 1) + ' 条 op 缺少 op 字段');
  var i, j;

  // ── 部件 ──
  if (op === 'part.add') {
    var node = isObj(spec.part) ? spec.part : spec;
    var id = slug(argStr(node, 'id', argStr(node, 'name', 'part')));
    if (!id) throw new Error('part.add 需要 id');
    for (i = 0; i < model.parts.length; i++) if (model.parts[i].id === id) throw new Error('部件 id 已存在: ' + id);
    if (model.parts.length >= LIMIT_PARTS) throw new Error('部件数已达上限 ' + LIMIT_PARTS);
    var w = num(node.width, 0), h = num(node.height, 0);
    if (!(w > 0) || !(h > 0)) throw new Error('part.add 需要 width / height > 0');
    var np = {
      id: id,
      color: isArr(node.color) && node.color.length === 4 ? [num(node.color[0], 1), num(node.color[1], 1), num(node.color[2], 1), num(node.color[3], 1)] : shadeFor(classifyRole(id, null).role),
      width: round3(w), height: round3(h),
      transform: { x: round3(num(node.x, 0)), y: round3(num(node.y, 0)) },
      order: isNum(node.order) ? Math.round(node.order) : (model.parts.length ? Math.max.apply(null, model.parts.map(function (p) { return p.order; })) + 1 : 0),
    };
    if (isNum(num(node.rotation, NaN)) && has(node, 'rotation')) np.transform.rotation = round3(num(node.rotation, 0));
    ['scaleX', 'scaleY', 'opacity'].forEach(function (k) { if (has(node, k)) np.transform[k] = round3(num(node[k], k === 'opacity' ? 1 : 1)); });
    if (argStr(node, 'deformer', '')) {
      if (!findDeformer(model, argStr(node, 'deformer', ''))) throw new Error('part.add 引用了未知 deformer: ' + argStr(node, 'deformer', ''));
      np.deformer = argStr(node, 'deformer', '');
    }
    model.parts.push(np);
    return { op: op, detail: '新增部件 ' + id + '（' + np.width + 'x' + np.height + ' @ ' + np.transform.x + ',' + np.transform.y + '）' };
  }
  if (op === 'part.set') {
    var props = isObj(spec.props) ? spec.props : {};
    if (!Object.keys(props).length) throw new Error('part.set 需要 props');
    var targets = findParts(model, spec);
    var changed = 0;
    for (i = 0; i < targets.length; i++) {
      var p = targets[i];
      for (var key in props) {
        if (!Object.prototype.hasOwnProperty.call(props, key)) continue;
        var v = props[key];
        if (key === 'width' || key === 'height') {
          if (v === null) { p[key] = 0; changed++; continue; }
          if (!(num(v, -1) > 0)) throw new Error('part.set 的 ' + key + ' 必须 > 0');
          p[key] = round3(num(v, 0)); changed++;
        } else if (key === 'order') {
          p.order = Math.round(num(v, 0)); changed++;
        } else if (key === 'color') {
          if (!isArr(v) || v.length !== 4) throw new Error('part.set 的 color 需要 [r,g,b,a]（0..1）');
          p.color = [num(v[0], 1), num(v[1], 1), num(v[2], 1), num(v[3], 1)]; changed++;
        } else if (key === 'deformer') {
          if (v === null || v === '') { delete p.deformer; changed++; continue; }
          if (!findDeformer(model, String(v))) throw new Error('part.set 引用了未知 deformer: ' + v);
          p.deformer = String(v); changed++;
        } else if (key === 'x' || key === 'y' || key === 'rotation' || key === 'scaleX' || key === 'scaleY' || key === 'opacity') {
          if (!p.transform) p.transform = { x: 0, y: 0 };
          if (v === null) delete p.transform[key];
          else p.transform[key] = round3(num(v, key === 'opacity' || key.indexOf('scale') === 0 ? 1 : 0));
          changed++;
        } else {
          throw new Error('part.set 不支持的字段: ' + key + '（部件契约字段只有 id/color/width/height/order/transform{x,y,rotation,scaleX,scaleY,opacity}/deformer/bindings/texture/mesh/warps/clip）');
        }
      }
    }
    return { op: op, detail: '更新 ' + targets.length + ' 个部件（' + changed + ' 处字段）: ' + targets.map(function (x) { return x.id; }).join(', ') };
  }
  if (op === 'part.rename') {
    var to = slug(argStr(spec, 'to', ''));
    if (!to) throw new Error('part.rename 需要 to');
    var list = findParts(model, spec);
    if (list.length !== 1) throw new Error('part.rename 一次只能改一个部件（匹配到 ' + list.length + ' 个）');
    for (i = 0; i < model.parts.length; i++) if (model.parts[i].id === to) throw new Error('目标 id 已存在: ' + to);
    var old = list[0].id;
    list[0].id = to;
    return { op: op, detail: '重命名 ' + old + ' → ' + to };
  }
  if (op === 'part.remove') {
    var del = findParts(model, spec);
    var delIds = {};
    for (i = 0; i < del.length; i++) delIds[del[i].id] = true;
    model.parts = model.parts.filter(function (x) { return !delIds[x.id]; });
    // 清理 clip 引用
    for (i = 0; i < model.parts.length; i++) {
      var cp = model.parts[i];
      if (cp.clip && isArr(cp.clip.masks)) {
        cp.clip.masks = cp.clip.masks.filter(function (mid) { return !delIds[mid]; });
        if (!cp.clip.masks.length) delete cp.clip;
      }
    }
    return { op: op, detail: '删除 ' + del.length + ' 个部件: ' + del.map(function (x) { return x.id; }).join(', ') };
  }
  if (op === 'part.order') {
    var ordList = findParts(model, spec);
    if (ordList.length !== 1) throw new Error('part.order 一次只能排一个部件');
    var target = Math.round(num(spec.order, NaN));
    if (!isNum(target)) throw new Error('part.order 需要 order（目标序号，0 起）');
    var seq = model.parts.slice().sort(function (a, b) { return a.order - b.order; });
    var moving = ordList[0];
    var without = seq.filter(function (x) { return x.id !== moving.id; });
    var at = clamp(target, 0, without.length);
    without.splice(at, 0, moving);
    for (i = 0; i < without.length; i++) without[i].order = i;
    return { op: op, detail: '部件 ' + moving.id + ' 移到绘制序号 ' + at + '（共 ' + without.length + ' 层）' };
  }
  if (op === 'part.bind') {
    var bl = findParts(model, spec);
    var res = [];
    for (i = 0; i < bl.length; i++) {
      if (!bl[i].bindings) bl[i].bindings = [];
      var holder = { arr: bl[i].bindings, _model: model };
      res.push(applyBind({ arr: bl[i].bindings, _model: model }, { parameter: spec.parameter, channel: spec.channel, from: spec.from, to: spec.to }, 'part.bind(' + bl[i].id + ')', false));
    }
    return { op: op, detail: '部件绑定 ' + bl.length + ' 个（' + res.join('/') + '）: ' + bl.map(function (x) { return x.id; }).join(', ') };
  }
  if (op === 'part.unbind') {
    var ul = findParts(model, spec);
    var removed = 0, kept = 0;
    for (i = 0; i < ul.length; i++) {
      if (!isArr(ul[i].bindings)) continue;
      var before = ul[i].bindings.length;
      ul[i].bindings = ul[i].bindings.filter(function (b) {
        if (spec.parameter && b.parameter !== String(spec.parameter)) return true;
        if (spec.channel && b.channel !== String(spec.channel)) return true;
        return false;
      });
      removed += before - ul[i].bindings.length;
      kept += ul[i].bindings.length;
      if (!ul[i].bindings.length) delete ul[i].bindings;
    }
    return { op: op, detail: '移除绑定 ' + removed + ' 条（保留 ' + kept + '）' };
  }
  if (op === 'part.assign') {
    var al = findParts(model, spec);
    var depId = argStr(spec, 'deformer', '');
    if (!findDeformer(model, depId)) throw new Error('part.assign 引用了未知 deformer: ' + depId);
    for (i = 0; i < al.length; i++) al[i].deformer = depId;
    return { op: op, detail: '把 ' + al.length + ' 个部件挂到 deformer ' + depId + ': ' + al.map(function (x) { return x.id; }).join(', ') };
  }
  if (op === 'part.mesh') {
    var ml = findParts(model, spec);
    var cols = clamp(argInt(spec, 'cols', 2), 1, 64), rows = clamp(argInt(spec, 'rows', 2), 1, 64);
    var out = [];
    for (i = 0; i < ml.length; i++) {
      var mp = ml[i];
      var verts = [], uvs = [], idx = [];
      var uvBase = (mp.texture && mp.texture.uv) ? mp.texture.uv : { x: 0, y: 0, width: 1, height: 1 };
      for (var r = 0; r <= rows; r++) {
        for (var c = 0; c <= cols; c++) {
          verts.push(round3(-0.5 + c / cols));
          verts.push(round3(0.5 - r / rows));
          uvs.push(round3(uvBase.x + uvBase.width * (c / cols)));
          uvs.push(round3(uvBase.y + uvBase.height * (r / rows)));
        }
      }
      for (r = 0; r < rows; r++) {
        for (c = 0; c < cols; c++) {
          var a0 = r * (cols + 1) + c, b0 = a0 + 1, c0 = a0 + cols + 1, d0 = c0 + 1;
          idx.push(a0, b0, d0, a0, d0, c0);
        }
      }
      if (verts.length / 2 > LIMIT_MESH_VERTS) throw new Error('网格顶点数 ' + (verts.length / 2) + ' 超过上限 ' + LIMIT_MESH_VERTS + '（减少 cols/rows）');
      mp.mesh = { vertices: verts, uvs: uvs, indices: idx };
      out.push(mp.id + '(' + (cols + 1) + 'x' + (rows + 1) + ')');
    }
    return { op: op, detail: '生成网格: ' + out.join(', ') };
  }
  if (op === 'part.clip') {
    var cl = findParts(model, spec);
    var masks = isArr(spec.masks) ? spec.masks.map(String) : (argStr(spec, 'mask', '') ? [argStr(spec, 'mask', '')] : null);
    if (!masks) throw new Error('part.clip 需要 masks 数组（或单个 mask）');
    for (i = 0; i < masks.length; i++) {
      var mk = null;
      for (j = 0; j < model.parts.length; j++) if (model.parts[j].id === masks[i]) mk = model.parts[j];
      if (!mk) throw new Error('遮罩部件不存在: ' + masks[i]);
      if (!mk.mesh) throw new Error('遮罩部件 "' + masks[i] + '" 没有网格（clip 要求遮罩自身带 mesh，先用 op=part.mesh）');
      if (mk.clip) throw new Error('遮罩部件 "' + masks[i] + '" 自身已被裁剪（.iki 的遮罩是扁平的、不允许嵌套）');
    }
    for (i = 0; i < cl.length; i++) cl[i].clip = { masks: masks.slice() };
    return { op: op, detail: '设置裁剪: ' + cl.map(function (x) { return x.id; }).join(', ') + ' ← ' + masks.join('+') };
  }

  // ── deformer ──
  if (op === 'deformer.add' || op === 'deformer.set') {
    var did = argStr(spec, 'id', '');
    if (!did) throw new Error(op + ' 需要 id');
    if (op === 'deformer.add') {
      if (findDeformer(model, did)) throw new Error('deformer id 已存在: ' + did);
      if ((model.deformers || []).length >= LIMIT_DEFORMERS) throw new Error('deformer 数已达上限 ' + LIMIT_DEFORMERS);
      var dNew = { id: did, pivot: { x: round3(num(spec.pivot && spec.pivot.x, 0)), y: round3(num(spec.pivot && spec.pivot.y, 0)) } };
      var par0 = argStr(spec, 'parent', '');
      if (par0) {
        if (!findDeformer(model, par0)) throw new Error('deformer.add 的 parent 不存在: ' + par0);
        dNew.parent = par0;
      }
      if (!model.deformers) model.deformers = [];
      model.deformers.push(dNew);
      return { op: op, detail: '新增 deformer ' + did + '（pivot ' + dNew.pivot.x + ',' + dNew.pivot.y + (dNew.parent ? ' 父 ' + dNew.parent : '') + '）' };
    }
    var dSet = findDeformer(model, did);
    if (!dSet) throw new Error('deformer 不存在: ' + did);
    var dp = isObj(spec.props) ? spec.props : spec;
    var dchanged = [];
    if (dp.pivot && isObj(dp.pivot)) {
      dSet.pivot = { x: round3(num(dp.pivot.x, 0)), y: round3(num(dp.pivot.y, 0)) };
      dchanged.push('pivot');
    }
    if (has(dp, 'parent')) {
      var pv = argStr(dp, 'parent', '');
      if (pv === '') delete dSet.parent;
      else {
        if (pv === did) throw new Error('deformer 不能把自己当父节点');
        if (!findDeformer(model, pv)) throw new Error('父 deformer 不存在: ' + pv);
        // 环检测
        var walkId = pv, guard = 0;
        while (walkId && guard++ < 64) {
          if (walkId === did) throw new Error('设置该父节点会形成环（' + did + ' → … → ' + pv + '）');
          var wd = findDeformer(model, walkId);
          walkId = wd ? wd.parent : null;
        }
        dSet.parent = pv;
      }
      dchanged.push('parent');
    }
    if (dp.transform && isObj(dp.transform)) {
      dSet.transform = copyObj(dSet.transform || {});
      ['x', 'y', 'rotation', 'scaleX', 'scaleY'].forEach(function (k) { if (has(dp.transform, k)) dSet.transform[k] = round3(num(dp.transform[k], k.indexOf('scale') === 0 ? 1 : 0)); });
      dchanged.push('transform');
    } else {
      ['x', 'y', 'rotation', 'scaleX', 'scaleY'].forEach(function (k) {
        if (has(dp, k)) {
          dSet.transform = copyObj(dSet.transform || {});
          dSet.transform[k] = round3(num(dp[k], k.indexOf('scale') === 0 ? 1 : 0));
          if (dchanged.indexOf('transform') < 0) dchanged.push('transform');
        }
      });
    }
    return { op: op, detail: '更新 deformer ' + did + '（' + (dchanged.join(',') || '无变化') + '）' };
  }
  if (op === 'deformer.remove') {
    var rid = argStr(spec, 'id', '');
    var rd = findDeformer(model, rid);
    if (!rd) throw new Error('deformer 不存在: ' + rid);
    if (argBool(spec, 'recursive', false)) {
      var doomed = { };
      var collect = function (pid) {
        doomed[pid] = true;
        for (var k = 0; k < (model.deformers || []).length; k++) {
          if (model.deformers[k].parent === pid) collect(model.deformers[k].id);
        }
      };
      collect(rid);
      model.deformers = (model.deformers || []).filter(function (d) { return !doomed[d.id]; });
      for (i = 0; i < model.parts.length; i++) if (doomed[model.parts[i].deformer]) delete model.parts[i].deformer;
      for (i = 0; i < (model.physicsChains || []).length; i++) {
        model.physicsChains = (model.physicsChains || []).filter(function (c) { return !doomed[c.anchorDeformer]; });
      }
      return { op: op, detail: '递归删除 deformer ' + Object.keys(doomed).join(', ') };
    }
    for (i = 0; i < model.parts.length; i++) if (model.parts[i].deformer === rid) throw new Error('仍有部件挂在该 deformer 上（' + model.parts[i].id + '…）—— 先 part.assign 改挂，或用 recursive=true');
    for (i = 0; i < (model.deformers || []).length; i++) if (model.deformers[i].parent === rid) throw new Error('仍有子 deformer（' + model.deformers[i].id + '）—— 用 recursive=true');
    for (i = 0; i < (model.physicsChains || []).length; i++) if (model.physicsChains[i].anchorDeformer === rid) throw new Error('角链 ' + model.physicsChains[i].id + ' 锚在该 deformer 上');
    model.deformers = model.deformers.filter(function (d) { return d.id !== rid; });
    return { op: op, detail: '删除 deformer ' + rid };
  }
  if (op === 'deformer.bind') {
    var bd = findDeformer(model, argStr(spec, 'id', ''));
    if (!bd) throw new Error('deformer 不存在: ' + argStr(spec, 'id', ''));
    if (!bd.bindings) bd.bindings = [];
    var r = applyBind({ arr: bd.bindings, _model: model }, { parameter: spec.parameter, channel: spec.channel, from: spec.from, to: spec.to }, 'deformer.bind(' + bd.id + ')', true);
    return { op: op, detail: 'deformer ' + bd.id + ' 绑定 ' + spec.parameter + '→' + spec.channel + '（' + r + '）' };
  }
  if (op === 'deformer.unbind') {
    var ud = findDeformer(model, argStr(spec, 'id', ''));
    if (!ud) throw new Error('deformer 不存在: ' + argStr(spec, 'id', ''));
    var before2 = (ud.bindings || []).length;
    ud.bindings = (ud.bindings || []).filter(function (b) {
      if (spec.parameter && b.parameter !== String(spec.parameter)) return true;
      if (spec.channel && b.channel !== String(spec.channel)) return true;
      return false;
    });
    if (!ud.bindings.length) delete ud.bindings;
    return { op: op, detail: 'deformer ' + ud.id + ' 移除绑定 ' + (before2 - (ud.bindings || []).length) + ' 条' };
  }
  if (op === 'warp.add') {
    var warpId = argStr(spec, 'id', '');
    if (!warpId) throw new Error('warp.add 需要 id');
    if (findDeformer(model, warpId)) throw new Error('deformer id 已存在: ' + warpId);
    var wid = argStr(spec, 'id', '');
    var targets0 = isArr(spec.ids) || spec.id ? null : null;
    // 目标部件（决定网格范围）：显式给 parts 或按 role 找
    var wParts = null;
    if (isArr(spec.parts) && spec.parts.length) {
      wParts = findParts(model, { ids: spec.parts });
    } else if (argStr(spec, 'role', '')) {
      wParts = findParts(model, { role: argStr(spec, 'role', '') });
    } else {
      throw new Error('warp.add 需要 parts（部件 id 数组）或 role');
    }
    var wbox = partRectAABB(wParts);
    var cols = clamp(argInt(spec, 'cols', 2), 1, 16), rows = clamp(argInt(spec, 'rows', 2), 1, 16);
    var gridPoints = [];
    for (var rr = 0; rr <= rows; rr++) {
      for (var cc = 0; cc <= cols; cc++) {
        gridPoints.push(round3(wbox.x0 + wbox.w * (cc / cols)));
        gridPoints.push(round3(wbox.y1 - wbox.h * (rr / rows)));   // row0 = TOP（+y 向上）
      }
    }
    var wd = { kind: 'warp', id: warpId, parent: argStr(spec, 'parent', '') || undefined, grid: { cols: cols, rows: rows, points: gridPoints } };
    if (!wd.parent) delete wd.parent;
    if (wd.parent && !findDeformer(model, wd.parent)) throw new Error('warp.add 的 parent 不存在: ' + wd.parent);
    if (wd.parent) {
      var pd = findDeformer(model, wd.parent);
      if (pd && pd.kind === 'warp') throw new Error('warp deformer 的父节点必须是 matrix deformer（上游契约）');
    }
    var param = argStr(spec, 'parameter', '');
    var preset = argStr(spec, 'preset', 'bend');
    var amount = num(spec.amount, 6);
    if (param) {
      checkParamExists(model, param, 'warp.add');
      // ★ keyform 的 value 必须**严格递增**（上游校验会拒）：缺省取参数范围的 min/max，
      //   显式给出 from >= to 时直接报错 —— 不产出必被上游拒绝的文件。
      var pdef = findParam(model, param);
      var from = has(spec, 'from') ? num(spec.from, 0) : num(pdef.min, 0);
      var to = has(spec, 'to') ? num(spec.to, 0) : num(pdef.max, 1);
      if (!(to > from)) throw new Error('warp.add 需要 to > from（keyform value 必须严格递增；缺省时取参数范围的 min/max）');
      var n = gridPoints.length / 2;
      function offsetsAt(k) {
        var offs = [];
        for (var q = 0; q < n; q++) {
          var gx = (q % (cols + 1)) / cols;        // 0..1 横向位置
          var gy = Math.floor(q / (cols + 1)) / rows;
          var dx = 0, dy = 0;
          if (preset === 'bend') { dx = amount * k * Math.sin(Math.PI * gy) * (gx - 0.5) * 2; dy = 0; }
          else if (preset === 'sway') { dx = amount * k * gy; dy = 0; }
          else if (preset === 'bulge') { dx = amount * k * (gx - 0.5); dy = amount * k * Math.sin(Math.PI * gx) * 0.5; }
          else if (preset === 'none') { dx = 0; dy = 0; }
          else throw new Error('warp.add 的 preset 只支持 bend / sway / bulge / none');
          offs.push(round3(dx));
          offs.push(round3(dy));
        }
        return offs;
      }
      wd.warps = [{ parameter: param, keyforms: [{ value: from, offsets: offsetsAt(0) }, { value: to, offsets: offsetsAt(1) }] }];
    }
    if (!model.deformers) model.deformers = [];
    model.deformers.push(wd);
    for (i = 0; i < wParts.length; i++) {
      var wp = wParts[i];
      if (!wp.mesh) {
        // 上游契约：挂到 warp deformer 的部件必须自带 mesh —— 就地按 2x2 生成
        var verts2 = [], uvs2 = [], idx2 = [];
        var uvB = (wp.texture && wp.texture.uv) ? wp.texture.uv : { x: 0, y: 0, width: 1, height: 1 };
        for (var r2 = 0; r2 <= 2; r2++) for (var c2 = 0; c2 <= 2; c2++) {
          verts2.push(round3(-0.5 + c2 / 2), round3(0.5 - r2 / 2));
          uvs2.push(round3(uvB.x + uvB.width * (c2 / 2)), round3(uvB.y + uvB.height * (r2 / 2)));
        }
        for (r2 = 0; r2 < 2; r2++) for (c2 = 0; c2 < 2; c2++) {
          var A = r2 * 3 + c2, B = A + 1, C = A + 3, D = C + 1;
          idx2.push(A, B, D, A, D, C);
        }
        wp.mesh = { vertices: verts2, uvs: uvs2, indices: idx2 };
      }
      wp.deformer = warpId;
    }
    return { op: op, detail: '新增 warp deformer ' + warpId + '（grid ' + (cols + 1) + 'x' + (rows + 1) + (param ? '，1D warp ← ' + param + ' 预设 ' + preset : '，无形变') + '），挂载 ' + wParts.length + ' 个部件' };
  }

  // ── 参数 ──
  if (op === 'param.add') {
    var pid2 = argStr(spec, 'id', '');
    if (!pid2) throw new Error('param.add 需要 id');
    if (findParam(model, pid2)) throw new Error('参数已存在: ' + pid2);
    if (model.parameters.length >= LIMIT_PARAMS) throw new Error('参数数已达上限 ' + LIMIT_PARAMS);
    var mn = num(spec.min, 0), mx = num(spec.max, 1);
    if (!(mx > mn)) throw new Error('param.add 需要 max > min');
    var dv = num(spec.default, mn);
    if (dv < mn || dv > mx) throw new Error('default ' + dv + ' 不在 [' + mn + ',' + mx + '] 内（上游运行时会夹取到 min/max）');
    model.parameters.push({ id: pid2, name: argStr(spec, 'name', '') || undefined, min: round3(mn), max: round3(mx), default: round3(dv) });
    return { op: op, detail: '新增参数 ' + pid2 + '（' + mn + '..' + mx + ' 默认 ' + dv + '）' };
  }
  if (op === 'param.set') {
    var pid3 = argStr(spec, 'id', '');
    var pDef = findParam(model, pid3);
    if (!pDef) throw new Error('参数不存在: ' + pid3);
    var props2 = isObj(spec.props) ? spec.props : spec;
    ['min', 'max', 'default'].forEach(function (k) { if (has(props2, k)) pDef[k] = round3(num(props2[k], pDef[k])); });
    if (has(props2, 'name')) pDef.name = argStr(props2, 'name', '') || undefined;
    if (!(num(pDef.max, 1) > num(pDef.min, 0))) throw new Error('参数 ' + pid3 + ' 的 max 必须 > min');
    if (num(pDef.default, 0) < num(pDef.min, 0) || num(pDef.default, 0) > num(pDef.max, 1)) {
      throw new Error('参数 ' + pid3 + ' 的 default 落在 [' + pDef.min + ',' + pDef.max + '] 之外');
    }
    if (!STD_PARAM_IDS[pid3]) return { op: op, detail: '更新参数 ' + pid3 };
    return { op: op, detail: '更新参数 ' + pid3 + '（★ 这是上游标准参数，改了范围会破坏宿主（口型/眨眼/头部追踪）的 1:1 映射，建议保持默认）' };
  }
  if (op === 'param.remove') {
    var pid4 = argStr(spec, 'id', '');
    if (!findParam(model, pid4)) throw new Error('参数不存在: ' + pid4);
    for (i = 0; i < model.parts.length; i++) {
      var b2 = model.parts[i].bindings || [];
      for (j = 0; j < b2.length; j++) if (b2[j].parameter === pid4) throw new Error('参数 ' + pid4 + ' 仍被部件 ' + model.parts[i].id + ' 绑定');
      var w2 = model.parts[i].warps || [];
      for (j = 0; j < w2.length; j++) if (w2[j].parameter === pid4) throw new Error('参数 ' + pid4 + ' 仍被部件 ' + model.parts[i].id + ' 的 warp 使用');
    }
    for (i = 0; i < (model.deformers || []).length; i++) {
      var db = model.deformers[i].bindings || [];
      for (j = 0; j < db.length; j++) if (db[j].parameter === pid4) throw new Error('参数 ' + pid4 + ' 仍被 deformer ' + model.deformers[i].id + ' 绑定');
      var dw = model.deformers[i].warps || [];
      for (j = 0; j < dw.length; j++) if (dw[j].parameter === pid4) throw new Error('参数 ' + pid4 + ' 仍被 deformer ' + model.deformers[i].id + ' 使用');
    }
    for (i = 0; i < (model.physics || []).length; i++) {
      if (model.physics[i].input.parameter === pid4 || model.physics[i].output.parameter === pid4) throw new Error('参数 ' + pid4 + ' 仍被弹簧 ' + model.physics[i].id + ' 使用');
    }
    for (i = 0; i < (model.physicsChains || []).length; i++) {
      var seg2 = model.physicsChains[i].segments || [];
      for (j = 0; j < seg2.length; j++) if (seg2[j].output.parameter === pid4) throw new Error('参数 ' + pid4 + ' 仍被角链 ' + model.physicsChains[i].id + ' 使用');
    }
    model.parameters = model.parameters.filter(function (x) { return x.id !== pid4; });
    return { op: op, detail: '删除参数 ' + pid4 };
  }

  // ── 物理 ──
  if (op === 'physics.add') {
    if (!model.physics) model.physics = [];
    if (model.physics.length >= LIMIT_PHYSICS) throw new Error('弹簧数已达上限 ' + LIMIT_PHYSICS);
    var phId = argStr(spec, 'id', '');
    if (!phId) throw new Error('physics.add 需要 id');
    for (i = 0; i < model.physics.length; i++) if (model.physics[i].id === phId) throw new Error('弹簧 id 已存在: ' + phId);
    // ★ input/output 既支持字符串（参数 id）也支持对象（{parameter,weight}）——
    //   不能直接走 argStr（它会把对象 String() 成 "[object Object]"，导致 in==out 误报）。
    var inP = isObj(spec.input) ? String(spec.input.parameter === undefined ? '' : spec.input.parameter) : argStr(spec, 'input', '');
    var outP = isObj(spec.output) ? String(spec.output.parameter === undefined ? '' : spec.output.parameter) : argStr(spec, 'output', '');
    if (!inP || !outP) throw new Error('physics.add 需要 input 与 output（参数 id）');
    if (inP === outP) throw new Error('physics 的 input 与 output 不能是同一参数（上游契约：二级运动不得写宿主驱动的输入参数）');
    checkParamExists(model, inP, 'physics.add.input');
    checkParamExists(model, outP, 'physics.add.output');
    var mass = num(spec.mass, 1), stiff = num(spec.stiffness, 12), damp = num(spec.damping, 0.3);
    if (!(mass > 0)) throw new Error('mass 必须 > 0');
    if (!(stiff > 0)) throw new Error('stiffness 必须 > 0');
    if (damp < 0 || damp >= 1) throw new Error('damping 需在 [0,1)（>=1 为过阻尼，二级运动不会回摆）');
    model.physics.push({
      id: phId,
      input: { parameter: inP, weight: num(isObj(spec.input) ? spec.input.weight : spec.weight, 1) },
      output: { parameter: outP, scale: num(isObj(spec.output) ? spec.output.scale : spec.scale, 1) },
      mass: round3(mass), stiffness: round3(stiff), damping: round3(damp),
    });
    return { op: op, detail: '新增弹簧 ' + phId + '（' + inP + ' → ' + outP + '，mass ' + mass + ' stiffness ' + stiff + ' damping ' + damp + '）' };
  }
  if (op === 'physics.remove') {
    var phId2 = argStr(spec, 'id', '');
    var n2 = (model.physics || []).length;
    model.physics = (model.physics || []).filter(function (x) { return x.id !== phId2; });
    if (model.physics.length === n2) throw new Error('弹簧不存在: ' + phId2);
    return { op: op, detail: '删除弹簧 ' + phId2 };
  }
  if (op === 'chain.add') {
    if (!model.physicsChains) model.physicsChains = [];
    if (model.physicsChains.length >= LIMIT_CHAINS) throw new Error('角链数已达上限 ' + LIMIT_CHAINS);
    var chId = argStr(spec, 'id', '');
    if (!chId) throw new Error('chain.add 需要 id');
    var anchor = argStr(spec, 'anchorDeformer', argStr(spec, 'anchor', ''));
    var ad = findDeformer(model, anchor);
    if (!ad) throw new Error('chain.add 的 anchorDeformer 不存在: ' + anchor);
    if (ad.kind === 'warp') throw new Error('角链锚点必须是 matrix deformer（上游契约：角链不跟随 warp 形变）');
    var segs = isArr(spec.segments) ? spec.segments : null;
    if (!segs || !segs.length) throw new Error('chain.add 需要 segments 数组（root → tip）');
    if (segs.length > LIMIT_CHAIN_SEG) throw new Error('段数超过上限 ' + LIMIT_CHAIN_SEG);
    var normSegs = [];
    for (i = 0; i < segs.length; i++) {
      var sg = segs[i];
      var oP = argStr(sg, 'output', argStr(isObj(sg.output) ? sg.output : {}, 'parameter', ''));
      if (!oP) throw new Error('chain.add 第 ' + (i + 1) + ' 段缺少 output.parameter');
      checkParamExists(model, oP, 'chain.add.segments[' + i + ']');
      var st2 = num(isObj(sg) ? sg.stiffness : 0, 12), dp2 = num(isObj(sg) ? sg.damping : 0, 0.3);
      if (st2 < 0) throw new Error('段 ' + (i + 1) + ' 的 stiffness 必须 >= 0（0 = 纯重力悬垂）');
      if (dp2 < 0) throw new Error('段 ' + (i + 1) + ' 的 damping 必须 >= 0');
      var ns = { output: { parameter: oP, scale: num(isObj(sg.output) ? sg.output.scale : 1, 1) }, mass: num(sg.mass, 1), stiffness: round3(st2), damping: round3(dp2) };
      if (!(ns.mass > 0)) throw new Error('段 ' + (i + 1) + ' 的 mass 必须 > 0');
      if (has(sg, 'restAngle')) ns.restAngle = round3(num(sg.restAngle, 0));
      normSegs.push(ns);
    }
    model.physicsChains.push({
      id: chId, anchorDeformer: anchor,
      gravity: { angle: num(spec.gravity && spec.gravity.angle, -90), strength: num(spec.gravity && spec.gravity.strength, 1) },
      segments: normSegs,
    });
    return { op: op, detail: '新增角链 ' + chId + '（锚 ' + anchor + '，' + normSegs.length + ' 段）' };
  }
  if (op === 'chain.remove') {
    var chId2 = argStr(spec, 'id', '');
    var n3 = (model.physicsChains || []).length;
    model.physicsChains = (model.physicsChains || []).filter(function (x) { return x.id !== chId2; });
    if (model.physicsChains.length === n3) throw new Error('角链不存在: ' + chId2);
    return { op: op, detail: '删除角链 ' + chId2 };
  }

  // ── 自动绑定 ──
  if (op === 'bind.auto') {
    var built2 = buildAutoRig(model, { hairSegments: argInt(spec, 'hairSegments', 2) });
    var rep = built2.report;
    var detail = '自动绑定：挂载 ' + rep.partsBound + ' 个部件到 ' + rep.deformers + ' 个 deformer（' + rep.deformerIds.join(', ') + '），部件级参数绑定 ' + rep.partsWithParamBindings + ' 条';
    if (rep.unclassified.length) detail += '；未识别命名 ' + rep.unclassified.length + ' 个（已挂 body）';
    return { op: op, detail: detail, report: rep };
  }

  // ── 纹理 ──
  if (op === 'texture.set') {
    var src = argStr(spec, 'source', '');
    if (!src) throw new Error('texture.set 需要 source（data:image/... 数据 URI 或图集路径）');
    if (!/^data:image\//.test(src) && !argBool(spec, 'allowExternal', false)) {
      throw new Error('.iki v1 的 textures[].source 只接受 data:image/ 数据 URI（上游校验）；外部路径需 allowExternal=true 才写入 —— 注意这会降低可移植性');
    }
    if (!model.textures) model.textures = [];
    var idx3 = argInt(spec, 'index', -1);
    if (idx3 < 0) model.textures.push({ source: src });
    else {
      if (idx3 >= model.textures.length) throw new Error('纹理下标越界: ' + idx3);
      model.textures[idx3] = { source: src };
    }
    return { op: op, detail: '设置纹理（共 ' + model.textures.length + ' 张，' + src.length + ' 字符）' };
  }

  throw new Error('未知 op: ' + op + '（可用：part.add/set/rename/remove/order/bind/unbind/assign/mesh/clip、deformer.add/set/remove/bind/unbind、warp.add、param.add/set/remove、physics.add/remove、chain.add/remove、bind.auto、texture.set）');
}

function composeOpFromArgs(args) {
  var o = {};
  for (var k in args) {
    if (!Object.prototype.hasOwnProperty.call(args, k)) continue;
    if (k === 'path' || k === 'ops') continue;
    o[k] = args[k];
  }
  return o;
}

// ── 批量新增部件（rig_add 与 rig_model create 的 parts 共用）────
// 与 part.add 同一套构造/校验（都走 applyOp），差别只在「N 个部件一次调用」：
// 省掉手写 N 条 op 的样板。事务性：逐件入列（后一件才看得见前一件占用的 id），
// 全部通过后由调用方落盘一次；任一项失败直接抛出 → 调用方走不到 saveModel，工程保持原样。
function applyPartSpecs(model, specs, what) {
  if (specs === undefined || specs === null) return [];
  if (!isArr(specs)) throw new Error((what || 'parts') + ' 必须是数组');
  if (!specs.length) throw new Error((what || 'parts') + ' 是空数组（要么不传，要么至少给一个部件）');
  var added = [];
  for (var i = 0; i < specs.length; i++) {
    if (!isObj(specs[i])) throw new Error((what || 'parts') + '[' + i + '] 必须是对象');
    var r;
    try {
      r = applyOp(model, { op: 'part.add', part: specs[i] }, i);
    } catch (e) {
      throw new Error('第 ' + (i + 1) + ' 个部件' + (specs[i].id ? '（' + specs[i].id + '）' : '') +
        '新增失败，整批未写入任何改动：' + ((e && e.message) || e));
    }
    added.push(r && r.detail ? r.detail : ('新增部件 ' + specs[i].id));
  }
  return added;
}

// 单件写法：除 path/parts 外全部透传给 part.add（与 rig_edit op=part.add 的平铺写法一致）
function partDefFromArgs(args) {
  var def = {};
  for (var k in args) {
    if (!Object.prototype.hasOwnProperty.call(args, k)) continue;
    if (k === 'path' || k === 'parts') continue;
    def[k] = args[k];
  }
  return def;
}

function rigEdit(args, opts, ctx) {
  var path = argStr(args, 'path', PROJECT_NAME);
  if (!ctx.fs.exists(path)) throw new Error('角色工程不存在: ' + path + '（先 rig_model action=create，或 rig_import 从 PSD/清单导入）');
  var model = loadModel(ctx, path);
  var ops = [];
  if (isArr(args.ops)) ops = args.ops;
  else if (argStr(args, 'op', '')) ops = [composeOpFromArgs(args)];
  else throw new Error('缺少 ops（命令数组）或 op（单条命令名 + 同层参数）');
  if (!ops.length) throw new Error('ops 为空');
  var log = [], i;
  for (i = 0; i < ops.length; i++) {
    var r;
    try {
      r = applyOp(model, ops[i], i);
    } catch (e) {
      throw new Error('第 ' + (i + 1) + ' 条 op（' + ((ops[i] && ops[i].op) || '?') + '）失败，整个批次未写入任何改动：' + ((e && e.message) || e));
    }
    log.push(r);
  }
  var out = normalizeModel(model);
  saveModel(ctx, path, out);
  return JSON.stringify({
    ok: true, action: 'edited', path: path,
    applied: log.length,
    log: log,
    summary: modelSummary(out),
    next: 'rig_verify 校验 / rig_export format=html 出预览 / rig_add 批量加部件',
  }, null, 2);
}

// ── rig_add：批量新增部件（一次调用 N 个，事务式）────────────────
// 与 tool-model 的 model_add（parts=[…]）同一形态：单件写法兼容 + 数组批量 + 整批不落盘。
function rigAdd(args, opts, ctx) {
  var path = argStr(args, 'path', PROJECT_NAME);
  if (!ctx.fs.exists(path)) throw new Error('角色工程不存在: ' + path + '（先 rig_model action=create，或 rig_import 从 PSD/清单导入）');
  var model = loadModel(ctx, path);
  var specs;
  if (args.parts !== undefined) {
    specs = args.parts;                              // 批量写法：以 parts 为准
  } else {
    if (!argStr(args, 'id', '')) throw new Error('rig_add 需要 parts 数组（批量）；或给 id/width/height（单件）');
    specs = [partDefFromArgs(args)];                 // 单件写法
  }
  var added = applyPartSpecs(model, specs, 'parts');
  var out = normalizeModel(model);
  saveModel(ctx, path, out);
  return JSON.stringify({
    ok: true, action: 'added', path: path, added: added,
    parts: out.parts.length,
    summary: modelSummary(out),
    next: 'rig_edit 绑骨骼与参数（deformer.add / part.bind / bind.auto）/ rig_export 出预览 / rig_verify 校验',
  }, null, 2);
}
// ── 产物①：自包含可交互预览 HTML ───────────────────────────
// 渲染器忠实实现上游语义（affine/deform/physics/physicsChains —— 与插件侧同一套公式）：
//   · 无外链、无字体依赖（纹理走 data: URI 时内联；无纹理部件用占位色 + 描边）
//   · 参数滑块实时驱动（含物理：弹簧 / 多段角链，1/60s 半隐式欧拉）
//   · 内置"演示动画"开关（呼吸 / 眨眼）—— 那是**宿主驱动**的模拟，用于看二级运动是否跟得上
//   · 暴露 window.__RIG_DEBUG（quadOf/matrixOf/setParam/step）供自动化核验渲染几何
function renderPreviewHtml(model, opts) {
  opts = opts || {};
  var W = num(model.canvas.width, 1000), H = num(model.canvas.height, 1000);
  var data = JSON.stringify(model).replace(/</g, '\\u003c');
  var paramRows = '';
  var i;
  for (i = 0; i < model.parameters.length; i++) {
    var p = model.parameters[i];
    if (i >= 24) { paramRows += '<div class="more">…还有 ' + (model.parameters.length - 24) + ' 个参数（面板/拖拽由宿主提供，此处只放前 24 个）</div>'; break; }
    paramRows += '<label class="row"><span class="nm" title="' + esc(p.id) + '">' + esc(p.name || p.id) + '</span>'
      + '<input type="range" data-pid="' + esc(p.id) + '" min="' + num(p.min, 0) + '" max="' + num(p.max, 1) + '" step="' + stepFor(p) + '" value="' + clamp(num(p.default, 0), num(p.min, 0), num(p.max, 1)) + '">'
      + '<b class="val">' + fmtNum(clamp(num(p.default, 0), num(p.min, 0), num(p.max, 1))) + '</b></label>';
  }
  return '<!DOCTYPE html>\n<html lang="zh-CN"><head><meta charset="utf-8">'
    + '<meta name="viewport" content="width=device-width,initial-scale=1">'
    + '<title>' + esc(model.name) + ' — 角色预览</title><style>'
    + '*{box-sizing:border-box}body{margin:0;font:13px/1.5 -apple-system,BlinkMacSystemFont,"Segoe UI",system-ui,sans-serif;color:#1f1f1f;background:#fafafa}'
    + '.wrap{display:flex;gap:12px;align-items:flex-start;padding:12px}'
    + '.stage{flex:1;min-width:0}canvas{display:block;width:100%;height:auto;background:#fff;border:1px solid #d6d6d6;border-radius:6px}'
    + '.meta{margin-top:6px;color:#6b6b6b;font-size:11px}'
    + 'aside{width:260px;flex:none;border:1px solid #d6d6d6;border-radius:6px;background:#fff;padding:8px 10px;max-height:calc(100vh - 24px);overflow:auto}'
    + 'h2{margin:0 0 6px;font-size:12px;font-weight:600;color:#4a4a4a}'
    + '.row{display:grid;grid-template-columns:1fr 96px 42px;gap:6px;align-items:center;padding:2px 0}'
    + '.row .nm{overflow:hidden;text-overflow:ellipsis;white-space:nowrap;color:#3a3a3a;font-size:12px}'
    + '.row .val{font-weight:500;font-variant-numeric:tabular-nums;text-align:right}'
    + 'input[type=range]{width:100%;margin:0}'
    + '.sep{border-top:1px solid #e4e4e4;margin:8px 0}'
    + '.more,.meta2{color:#6b6b6b;font-size:11px}'
    + 'button{margin-right:6px;padding:2px 8px;font-size:11px;color:#3a3a3a;background:#fff;border:1px solid #d6d6d6;border-radius:4px;cursor:pointer}'
    + 'button:hover{background:#f1f1f1}'
    + '</style></head><body><div class="wrap">'
    + '<div class="stage"><canvas id="cv" width="' + W + '" height="' + H + '"></canvas>'
    + '<div class="meta">画布 ' + W + '×' + H + '（模型空间原点 = 画布中心，+y 向上）｜部件 ' + model.parts.length
    + ' ｜deformer ' + ((model.deformers || []).length) + ' ｜弹簧 ' + ((model.physics || []).length) + ' ｜角链 ' + ((model.physicsChains || []).length) + '</div></div>'
    + '<aside><h2>参数</h2>' + paramRows
    + '<div class="sep"></div><h2>播放</h2>'
    + '<button id="btnIdle">演示动画：开</button><button id="btnReset">重置参数</button>'
    + '<div class="meta2" id="hz">—</div></aside></div>'
    + '<script>var MODEL=' + data + ';' + PREVIEW_JS + '</script></body></html>';
}
function stepFor(p) {
  var span = num(p.max, 1) - num(p.min, 0);
  if (span <= 1) return '0.01';
  if (span <= 10) return '0.1';
  return '1';
}
function esc(s) {
  return String(s === undefined || s === null ? '' : s).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}

// 预览运行时（内联进 HTML；语义与插件侧一致，见上方注释）
var PREVIEW_JS = [
  '(function(){',
  'var cv=document.getElementById("cv"),ctx=cv.getContext("2d");',
  'var W=MODEL.canvas.width,H=MODEL.canvas.height;',
  'function clamp(v,a,b){return v<a?a:(v>b?b:v)}',
  'function mul(a,b){return [a[0]*b[0]+a[2]*b[1],a[1]*b[0]+a[3]*b[1],a[0]*b[2]+a[2]*b[3],a[1]*b[2]+a[3]*b[3],a[0]*b[4]+a[2]*b[5]+a[4],a[1]*b[4]+a[3]*b[5]+a[5]]}',
  'function tr(tx,ty){return [1,0,0,1,tx,ty]}',
  'function sc(sx,sy){return [sx,0,0,sy,0,0]}',
  'function rot(d){var r=d*Math.PI/180,c=Math.cos(r),s=Math.sin(r);return [c,s,-s,c,0,0]}',
  'function applyM(m,x,y){return {x:m[0]*x+m[2]*y+m[4],y:m[1]*x+m[3]*y+m[5]}}',
  'var defs={},vals={},defaults={};',
  'MODEL.parameters.forEach(function(p){defs[p.id]=p;var d=clamp(+p.default,p.min,p.max);defaults[p.id]=d;vals[p.id]=d});',
  'function get(id){var p=defs[id];if(!p)return 0;return clamp(vals[id]===undefined?0:vals[id],p.min,p.max)}',
  'function norm(id){var p=defs[id];if(!p||p.max===p.min)return 0;return (get(id)-p.min)/(p.max-p.min)}',
  'function signed(id){var p=defs[id];if(!p)return 0;var rest=clamp(p.default,p.min,p.max);var den=Math.max(Math.abs(p.max-rest),Math.abs(rest-p.min));if(den===0)return 0;return clamp((get(id)-rest)/den,-1,1)}',
  'function evT(t,bs){var b=t||{};var r={x:+b.x||0,y:+b.y||0,rotation:+b.rotation||0,scaleX:b.scaleX===undefined?1:+b.scaleX,scaleY:b.scaleY===undefined?1:+b.scaleY,opacity:b.opacity===undefined?1:+b.opacity};',
  ' (bs||[]).forEach(function(bd){var v=(+bd.from||0)+((+bd.to||0)-(+bd.from||0))*norm(bd.parameter);',
  '  if(bd.channel==="translateX")r.x+=v;else if(bd.channel==="translateY")r.y+=v;else if(bd.channel==="rotate")r.rotation+=v;else if(bd.channel==="scaleX")r.scaleX+=v;else if(bd.channel==="scaleY")r.scaleY+=v;else if(bd.channel==="opacity")r.opacity*=v});return r}',
  'function dLocal(d){var t=evT(d.transform,d.bindings);var trs=mul(mul(tr(t.x,t.y),rot(t.rotation)),sc(t.scaleX,t.scaleY));var px=(d.pivot&&+d.pivot.x)||0,py=(d.pivot&&+d.pivot.y)||0;return mul(mul(tr(px,py),trs),tr(-px,-py))}',
  'var dById={},dList=[];(MODEL.deformers||[]).forEach(function(d){if(d.kind===undefined||d.kind==="matrix"){dById[d.id]=d;dList.push(d)}});',
  'var dCache={};',
  'function dWorld(d,depth){if(dCache[d.id])return dCache[d.id];if((depth||0)>64)throw new Error("deformer 父链过深");var l=dLocal(d);var w;if(!d.parent)w=l;else{var par=dById[d.parent];if(!par)throw new Error("未知 deformer 父节点 "+d.parent);w=mul(dWorld(par,(depth||0)+1),l)}dCache[d.id]=w;return w}',
  'function worlds(){dCache={};dList.forEach(function(d){dWorld(d,0)});return dCache}',
  'function partM(p,W_,S_){var t=evT(p.transform,p.bindings);var inner=mul(mul(tr(t.x,t.y),rot(t.rotation)),sc((+p.width||0)*t.scaleX,(+p.height||0)*t.scaleY));if(p.deformer){var w=W_[p.deformer];if(w)return mul(w,inner)/*warp deformer 无矩阵：不继承父链（上游语义）*/}return inner}',
  'function quadOf(id){var p=null;MODEL.parts.forEach(function(x){if(x.id===id)p=x});if(!p)return null;var m=partM(p,worlds(),vals);return [[-0.5,0.5],[0.5,0.5],[0.5,-0.5],[-0.5,-0.5]].map(function(q){return applyM(m,q[0],q[1])})}',
  'var imgs={},imgReady={};',
  '(function(){var ts=MODEL.textures||[];ts.forEach(function(t,i){if(!/^data:image\\//.test(t.source))return;var im=new Image();im.onload=function(){imgReady[i]=true};im.src=t.source;imgs[i]=im})})();',
  'function warpOffsets(p,keyforms){var ks=keyforms||[];if(!ks.length)return null;var v=get(ks[0].parameter);var arr=ks[0].keyforms;if(!arr||!arr.length)return null;',
  ' if(v<=arr[0].value)return arr[0].offsets;if(v>=arr[arr.length-1].value)return arr[arr.length-1].offsets;',
  ' for(var i=0;i<arr.length-1;i++){var a=arr[i],b=arr[i+1];if(v>=a.value&&v<=b.value){var t=(b.value-a.value)===0?0:(v-a.value)/(b.value-a.value);var out=[];for(var k=0;k<a.offsets.length;k++)out[k]=(a.offsets[k]||0)+(((b.offsets[k]||0)-(a.offsets[k]||0))*t);return out}}return null}',
  'function gridWarpOffsets(d){if(!d||d.kind!=="warp"||!d.warps||!d.warps.length)return null;var w=d.warps[0];var arr=w.keyforms;if(!arr||!arr.length)return null;var v=get(w.parameter);',
  ' if(v<=arr[0].value)return arr[0].offsets;if(v>=arr[arr.length-1].value)return arr[arr.length-1].offsets;',
  ' for(var i=0;i<arr.length-1;i++){var a=arr[i],b=arr[i+1];if(v>=a.value&&v<=b.value){var t=(b.value-a.value)===0?0:(v-a.value)/(b.value-a.value);var out=[];for(var k=0;k<a.offsets.length;k++)out[k]=(a.offsets[k]||0)+(((b.offsets[k]||0)-(a.offsets[k]||0))*t);return out}}return null}',
  'function gridSample(dg,offs,pt){var cols=dg.cols,rows=dg.rows;var x0=dg.points[0],x1=dg.points[cols*2];var y1=dg.points[1],y0=dg.points[rows*(cols+1)*2+1];',
  ' var u=(pt.x-x0)/((x1-x0)||1),v=(pt.y-y0)/((y1-y0)||1);u=clamp(u,0,1);v=clamp(v,0,1);',
  ' var cx=u*cols,cy=(1-v)*rows;var c0=Math.floor(cx),r0=Math.floor(cy);c0=Math.min(c0,cols-1);r0=Math.min(r0,rows-1);',
  ' var fx=cx-c0,fy=cy-r0;function off(c,r){var i=(c*(cols+1)+r)*2;return {x:offs[i]||0,y:offs[i+1]||0}}',
  ' var o00=off(c0,r0),o10=off(c0+1,r0),o01=off(c0,r0+1),o11=off(c0+1,r0+1);',
  ' return {x:(o00.x*(1-fx)+o10.x*fx)*(1-fy)+(o01.x*(1-fx)+o11.x*fx)*fy,y:(o00.y*(1-fx)+o10.y*fx)*(1-fy)+(o01.y*(1-fx)+o11.y*fx)*fy}}',
  'var FLIP=[1,0,0,-1,W/2,H/2];',
  'function draw(){ctx.setTransform(1,0,0,1,0,0);ctx.clearRect(0,0,W,H);ctx.fillStyle="#ffffff";ctx.fillRect(0,0,W,H);',
  ' var Ws=worlds();var parts=MODEL.parts.slice().sort(function(a,b){return a.order-b.order});',
  ' for(var i=0;i<parts.length;i++){var p=parts[i];var m=mul(FLIP,partM(p,Ws,vals));var t=evT(p.transform,p.bindings);var op=clamp(t.opacity,0,1);if(op<=0.003)continue;',
  '  var wd=null,boff=null;if(p.deformer){var dd=dById[p.deformer];if(dd&&dd.kind==="warp"){wd=dd;boff=gridWarpOffsets(dd)}}',
  '  var poff=warpOffsets(p,p.warps);',
  '  if(p.mesh&&(poff||(wd&&boff))){drawMesh(p,m,poff,wd,boff,op);continue}',
  '  ctx.save();ctx.setTransform(m[0],m[1],m[2],m[3],m[4],m[5]);ctx.globalAlpha=op;',
  '  var tex=p.texture?imgs[p.texture.index]:null;var ready=p.texture?imgReady[p.texture.index]:false;',
  '  if(tex&&ready){ctx.scale(1,-1);var uv=p.texture.uv;var iw=tex.naturalWidth,ih=tex.naturalHeight;ctx.drawImage(tex,uv.x*iw,uv.y*ih,uv.width*iw,uv.height*ih,-0.5,-0.5,1,1)}',
  '  else{var c=p.color||[0.8,0.8,0.8,1];ctx.fillStyle="rgba("+Math.round(c[0]*255)+","+Math.round(c[1]*255)+","+Math.round(c[2]*255)+","+(c[3]||1)+")";ctx.fillRect(-0.5,-0.5,1,1);',
  '   ctx.lineWidth=1.5/Math.max(1,Math.abs(m[0])+Math.abs(m[3]));ctx.strokeStyle="rgba(60,60,60,0.35)";ctx.strokeRect(-0.5,-0.5,1,1)}',
  '  ctx.restore();}',
  ' ctx.setTransform(1,0,0,1,0,0);}',
  'function drawMesh(p,m,poff,wd,boff,op){var vs=p.mesh.vertices;var pts=[];var n=vs.length/2;',
  ' for(var i=0;i<n;i++){var lx=vs[i*2],ly=vs[i*2+1];if(poff){lx+=(poff[i*2]||0);ly+=(poff[i*2+1]||0)}',
  '  var model=applyM(m2model(p),lx,ly);',
  '  var mp={x:model.x,y:model.y};',
  '  if(wd&&boff){var o=gridSample(wd.grid,boff,applyM(m2model(p),lx,ly));mp={x:mp.x+o.x,y:mp.y+o.y}}',
  '  pts.push(applyM(FLIP,mp.x,mp.y))}',
  ' ctx.save();ctx.globalAlpha=op;var c=p.color||[0.8,0.8,0.8,1];ctx.fillStyle="rgba("+Math.round(c[0]*255)+","+Math.round(c[1]*255)+","+Math.round(c[2]*255)+","+((c[3]||1)*0.85)+")";',
  ' ctx.beginPath();for(var k=0;k<p.mesh.indices.length;k+=3){var a=pts[p.mesh.indices[k]],b=pts[p.mesh.indices[k+1]],d=pts[p.mesh.indices[k+2]];ctx.moveTo(a.x,a.y);ctx.lineTo(b.x,b.y);ctx.lineTo(d.x,d.y);ctx.closePath()}',
  ' ctx.fill();ctx.lineWidth=0.6;ctx.strokeStyle="rgba(60,60,60,0.25)";ctx.stroke();ctx.restore()}',
  'function m2model(p){var t=evT(p.transform,p.bindings);var m0=mul(mul(tr(t.x,t.y),rot(t.rotation)),sc((+p.width||0)*t.scaleX,(+p.height||0)*t.scaleY));if(p.deformer){var w=dCache[p.deformer];if(w)m0=mul(w,m0)}return m0}',
  'var springs=(MODEL.physics||[]).map(function(r){return {rig:r,x:0,v:0}});',
  'var chains=(MODEL.physicsChains||[]).map(function(c){return {chain:c,seg:(c.segments||[]).map(function(s){return {angle:0,vel:0}})}});',
  'var DT=1/60,acc=0,last=0,idle=true,demoT=0;',
  'function anchorAngle(chain){var d=dById[chain.anchorDeformer];if(!d)return 0;var w=dCache[d.id]||mul(FLIP,mul(tr(0,0),dWorld(d,0)));return Math.atan2(w[1],w[0])}',
  'function stepPhysics(){springs.forEach(function(s){var r=s.rig;var tgt=signed(r.input.parameter)*(+r.input.weight||0);var a=((+r.stiffness||0)*(tgt-s.x)-(+r.damping||0)*s.v)/(+r.mass||1);s.v+=a*DT;s.x+=s.v*DT;',
  ' var op=defs[r.output.parameter];var od=op?clamp(+op.default,op.min,op.max):0;vals[r.output.parameter]=od+s.x*(+r.output.scale||0)});',
  ' chains.forEach(function(cd){var c=cd.chain;var ga=(c.gravity.angle||0)*Math.PI/180,st=+c.gravity.strength||0;var wa=anchorAngle(c);',
  '  for(var i=0;i<c.segments.length;i++){var s=c.segments[i],stt=cd.seg[i];wa+=(s.restAngle||0)*Math.PI/180+stt.angle;',
  '   var a=(-( +s.stiffness||0)*stt.angle-st*Math.sin(wa-ga)-(+s.damping||0)*stt.vel)/(+s.mass||1);stt.vel+=a*DT;stt.angle+=stt.vel*DT;',
  '   var op=defs[s.output.parameter];var od=op?clamp(+op.default,op.min,op.max):0;vals[s.output.parameter]=od+stt.angle*180/Math.PI*(+s.output.scale||0)}})}',
  'function demo(dt){if(!idle)return;demoT+=dt;vals["ParamBreath"]=(Math.sin(demoT*Math.PI*2/3.2)+1)/2;',
  ' var bp=4.0,ph=demoT%bp;var b=ph<0.12?0:(ph<0.22?1-(ph-0.12)/0.1:1);b=ph<0.12?1-b:(ph<0.22?b:1);',
  ' if(defs["ParamEyeLOpen"])vals["ParamEyeLOpen"]=b;if(defs["ParamEyeROpen"])vals["ParamEyeROpen"]=b}',
  'var frames=0,lastHz=0,hzT=0;',
  'function loop(now){if(!last)last=now;var dt=Math.min(0.05,(now-last)/1000);last=now;demo(dt);acc+=dt;var steps=0;while(acc>=DT&&steps<5){stepPhysics();acc-=DT;steps++}',
  ' draw();frames++;hzT+=dt;if(hzT>=1){document.getElementById("hz").textContent=frames+" fps ｜ 弹簧 "+springs.length+" ｜ 角链 "+chains.length;frames=0;hzT=0}',
  ' requestAnimationFrame(loop)}',
  'function clampAll(){MODEL.parameters.forEach(function(p){vals[p.id]=clamp(vals[p.id]===undefined?clamp(+p.default,p.min,p.max):vals[p.id],p.min,p.max)})}',
  'document.querySelectorAll("input[data-pid]").forEach(function(el){el.addEventListener("input",function(){vals[el.getAttribute("data-pid")]=parseFloat(el.value);',
  ' el.parentNode.querySelector(".val").textContent=(Math.round(parseFloat(el.value)*1000)/1000);clampAll()})});',
  'document.getElementById("btnIdle").addEventListener("click",function(){idle=!idle;this.textContent="演示动画："+(idle?"开":"关")});',
  'document.getElementById("btnReset").addEventListener("click",function(){MODEL.parameters.forEach(function(p){vals[p.id]=clamp(+p.default,p.min,p.max)});',
  ' document.querySelectorAll("input[data-pid]").forEach(function(el){el.value=vals[el.getAttribute("data-pid")];el.parentNode.querySelector(".val").textContent=vals[el.getAttribute("data-pid")]})});',
  'window.__RIG_DEBUG={quadOf:quadOf,setParam:function(id,v){vals[id]=v;clampAll();stepPhysics();draw()},getParam:function(id){return get(id)},step:function(n){for(var i=0;i<(n||1);i++)stepPhysics();draw()},draw:draw,worlds:worlds,parts:function(){return MODEL.parts.map(function(p){return {id:p.id,order:p.order,deformer:p.deformer}})}};',
  'requestAnimationFrame(loop);',
  '})();',
].join('\n');

// ── 产物②：部件布局 SVG（静态可截图，同 tool-art 口径：矢量 + 文本真相源）──────
function renderSpritesSvg(model, opts) {
  opts = opts || {};
  var W = num(model.canvas.width, 1000), H = num(model.canvas.height, 1000);
  var store = new ParamStore(model);
  var worlds = {};
  var mathOnly = true;
  try { worlds = resolveDeformerWorlds(model.deformers || [], store); } catch (e) { mathOnly = false; }
  var parts = model.parts.slice().sort(function (a, b) { return a.order - b.order; });
  var out = [];
  out.push('<svg xmlns="http://www.w3.org/2000/svg" width="' + W + '" height="' + H + '" viewBox="0 0 ' + W + ' ' + H + '" role="img" aria-label="' + esc(model.name) + ' 部件布局">');
  out.push('  <title>' + esc(model.name) + ' — 部件布局（' + parts.length + ' 个，绘制顺序自后向前）</title>');
  out.push('  <rect x="0" y="0" width="' + W + '" height="' + H + '" fill="#ffffff"/>');
  out.push('  <g stroke="#c9c9c9" stroke-width="1" stroke-dasharray="4 4"><line x1="' + (W / 2) + '" y1="0" x2="' + (W / 2) + '" y2="' + H + '"/><line x1="0" y1="' + (H / 2) + '" x2="' + W + '" y2="' + (H / 2) + '"/></g>');
  var i;
  for (i = 0; i < parts.length; i++) {
    var p = parts[i];
    var q;
    try {
      q = partQuad(p, worlds, store, model);
    } catch (e) {
      continue;
    }
    var c = p.color || [0.8, 0.8, 0.8, 1];
    var fill = 'rgb(' + Math.round(c[0] * 255) + ',' + Math.round(c[1] * 255) + ',' + Math.round(c[2] * 255) + ')';
    var pts = q.pts.map(function (pt) { return fmtNum(W / 2 + pt.x) + ',' + fmtNum(H / 2 - pt.y); }).join(' ');
    var cx = W / 2 + (q.pts[0].x + q.pts[2].x) / 2;
    var cy = H / 2 - (q.pts[0].y + q.pts[2].y) / 2;
    var cls = classifyRole(p.id, null);
    out.push('  <g data-part="' + esc(p.id) + '" data-role="' + esc(cls.role) + '" data-order="' + p.order + '"' + (p.deformer ? ' data-deformer="' + esc(p.deformer) + '"' : '') + '>');
    out.push('    <polygon points="' + pts + '" fill="' + fill + '" fill-opacity="' + fmtNum(clamp(num(c[3], 1) * 0.55, 0, 1)) + '" stroke="#3c3c3c" stroke-opacity="0.5" stroke-width="1"/>');
    // 标签只在**装得下**时画（字号 11 → 每字符约 6.2 模型单位）：否则小部件（如眼白）的标签
    // 会溢出到相邻部件上互相压字（实测截图发现 eye_white_L 与 iris_L 叠成一团）。
    var label = p.id;
    if (opts.labels !== false && num(p.width, 0) > 26 && num(p.height, 0) > 12 && label.length * 6.2 <= num(p.width, 0) * 0.94) {
      out.push('    <text x="' + fmtNum(cx) + '" y="' + fmtNum(cy) + '" font-family="ui-monospace,Consolas,monospace" font-size="11" fill="#1f1f1f" text-anchor="middle" dominant-baseline="middle">' + esc(p.id) + '</text>');
    }
    out.push('  </g>');
  }
  var defsIds = (model.deformers || []).map(function (d) { return d.id + (d.parent ? '←' + d.parent : ''); });
  out.push('  <text x="10" y="' + (H - 10) + '" font-family="ui-monospace,Consolas,monospace" font-size="11" fill="#6b6b6b">deformer: ' + esc(defsIds.join('  ') || '(无)') + (mathOnly ? '' : ' ⚠ deformer 树存在未解析父节点，已跳过相关部件') + '</text>');
  out.push('</svg>');
  return out.join('\n') + '\n';
}

// ── 产物③：图层命名清单（交代美术/后续接入：命名约定 + 位置 + 建议绑定）────────
function renderPartsManifest(model) {
  var store = new ParamStore(model);
  var rows = [];
  for (var i = 0; i < model.parts.length; i++) {
    var p = model.parts[i];
    var cls = classifyRole(p.id, null);
    var row = {
      order: p.order, id: p.id, role: cls.role, side: cls.side || null,
      rect: { x: fmtNum(num(p.transform && p.transform.x, 0)), y: fmtNum(num(p.transform && p.transform.y, 0)), width: p.width, height: p.height },
      deformer: p.deformer || null,
      bindings: (p.bindings || []).map(function (b) { return b.parameter + '→' + b.channel + '(' + b.from + '..' + b.to + ')'; }),
      hasMesh: !!p.mesh, hasTexture: !!p.texture, clipped: !!(p.clip && p.clip.masks),
    };
    rows.push(row);
  }
  var fixed = (model.deformers || []).filter(function (d) { return d.bindings && d.bindings.length; }).map(function (d) {
    return { id: d.id, parent: d.parent || null, bindings: d.bindings.map(function (b) { return b.parameter + '→' + b.channel + '(' + b.from + '..' + b.to + ')'; }) };
  });
  return {
    model: model.name,
    canvas: { width: model.canvas.width, height: model.canvas.height },
    coordinateNote: 'rect.x/y = 部件中心在**模型空间**的坐标（原点 = 画布中心，+y 向上）；PSD 像素坐标 = (x + 画布宽/2, 画布高/2 - y)。',
    namingConvention: {
      rule: '小写词根 + 可选 _L/_R 侧别后缀（*_L / *_R 指**角色自身**左右，屏幕 +x 侧 = 角色左）；序号用 _2/_3',
      roots: ['head/hair/face/eye/eyeball/eyelid/brow/mouth', 'neck/body/torso/chest/hip', 'arm/hand/shoulder/elbow', 'leg/thigh/knee/shin/foot', 'cloth/skirt/coat/cape/scarf', 'tail/wing/horn/accessory/prop'],
      example: 'hair_front_L_2 / eye_white_R / arm_upper_L / skirt_front_2',
    },
    parts: rows,
    deformerBindings: fixed,
    physics: (model.physics || []).map(function (p) { return p.id + ': ' + p.input.parameter + '×' + p.input.weight + ' → ' + p.output.parameter + '×' + p.output.scale; }),
    physicsChains: (model.physicsChains || []).map(function (c) { return c.id + ': 锚 ' + c.anchorDeformer + '，' + c.segments.length + ' 段 → ' + c.segments.map(function (s) { return s.output.parameter; }).join(' / '); }),
    parameters: model.parameters.map(function (p) { return { id: p.id, name: p.name || null, min: p.min, max: p.max, default: p.default, standard: !!STD_PARAM_IDS[p.id] }; }),
  };
}

// ── 工具④ rig_export ───────────────────────────────────────
function rigExport(args, opts, ctx) {
  var path = argStr(args, 'path', PROJECT_NAME);
  var model = loadModel(ctx, path);
  var format = argStr(args, 'format', 'html').toLowerCase();
  var overwrite = argBool(args, 'overwrite', false);
  var out = argStr(args, 'out', '');
  function target(defName) {
    var t = out || defName;
    if (ctx.fs.exists(t) && !overwrite) {
      throw new Error('产物已存在: ' + t + '（覆盖请 overwrite=true）');
    }
    return t;
  }
  if (format === 'html' || format === 'preview') {
    var t1 = target('rig.preview.html');
    var html = renderPreviewHtml(model, {});
    writeTextFile(ctx, t1, html);
    return '已导出可交互预览: ' + t1 + '（' + html.length + ' 字符；参数滑块 + 弹簧/角链二级运动 + 无外链）\n'
      + '说明：无纹理部件用占位灰阶 + 描边绘制（结构视图）；纹理走 data: URI 时直接绘制；网格部件按顶点绘制（warp 偏移已应用）。';
  }
  if (format === 'svg' || format === 'sprites') {
    var t2 = target('rig.sprites.svg');
    var svg = renderSpritesSvg(model, {});
    writeTextFile(ctx, t2, svg);
    return '已导出部件布局图: ' + t2 + '（' + svg.length + ' 字符；默认参数姿态、绘制顺序自后向前、含 deformer 树注记）';
  }
  if (format === 'parts' || format === 'manifest') {
    var t3 = target('rig.parts.json');
    var man = renderPartsManifest(model);
    writeJsonFile(ctx, t3, man);
    return '已导出图层命名清单: ' + t3 + '（' + man.parts.length + ' 个部件；含 role 分类、模型空间矩形、命名约定与建议绑定）\n'
      + '用途：交给美术按命名约定重导 PSD/图层 PNG；或作为「超大 PSD」走 rig_import source=manifest 的输入（把 rect 转回像素坐标即可）。';
  }
  if (format === 'iki' || format === 'model') {
    var t4 = target('rig.model.export.iki.json');
    writeJsonFile(ctx, t4, normalizeModel(model));
    return '已导出 .iki 文档副本: ' + t4 + '（version ' + IKI_VERSION + '，与工程文件同构；上游 @ikijs/engine 可直接加载）';
  }
  throw new Error('未知 format: ' + format + '（可选 html / svg / parts / iki）');
}
// ── 工具⑤ rig_verify：9 项判据（R1..R9）─────────────────────
// 口径（对齐《创作域支持方案》§2.5 与 §4.4 的验收要求：rig schema 校验 + 参数范围 + 部件重叠）：
//   R1 结构契约 ｜ R2 命名规范 ｜ R3 deformer 树 ｜ R4 蒙皮完整性 ｜ R5 参数与绑定
//   R6 物理语义 ｜ R7 部件重叠（拆层检查） ｜ R8 变换与网格 sanity ｜ R9 产物确定性且自包含
function emojiScan(s) {
  var t = String(s || ''), out = [];
  for (var i = 0; i < t.length; i++) {
    var c = t.charCodeAt(i), cp = c;
    if (c >= 0xD800 && c <= 0xDBFF && i + 1 < t.length) {
      var lo = t.charCodeAt(i + 1);
      if (lo >= 0xDC00 && lo <= 0xDFFF) { cp = 0x10000 + ((c - 0xD800) << 10) + (lo - 0xDC00); i++; }
    }
    if ((cp >= 0x1F300 && cp <= 0x1FAFF) || (cp >= 0x2600 && cp <= 0x27BF) || cp === 0xFE0F || (cp >= 0x1F000 && cp <= 0x1F2FF)) {
      out.push(String.fromCharCode(cp <= 0xFFFF ? cp : 0xD83D));
    }
  }
  return out;
}

function rigVerify(args, opts, ctx) {
  var path = argStr(args, 'path', PROJECT_NAME);
  var reportPath = argStr(args, 'report', 'rig.verify.json');
  var overlapThreshold = clamp(num(args.overlapThreshold, 0.6), 0.05, 0.95);
  var model = loadModel(ctx, path);
  var checks = [];
  var i, j, k;

  // ── R1 结构契约 ──
  var f1 = [];
  if (num(model.version, 0) !== IKI_VERSION) f1.push({ level: 'fail', msg: '.iki version 必须是 ' + IKI_VERSION + '（上游契约 @ikijs/format）' });
  if (!String(model.name || '').length) f1.push({ level: 'fail', msg: 'name 为空' });
  if (!(num(model.canvas.width, 0) > 0) || !(num(model.canvas.height, 0) > 0)) f1.push({ level: 'fail', msg: 'canvas 尺寸非法' });
  if (num(model.canvas.width, 0) > LIMIT_CANVAS || num(model.canvas.height, 0) > LIMIT_CANVAS) f1.push({ level: 'fail', msg: 'canvas 超过上限 ' + LIMIT_CANVAS });
  if (model.parts.length === 0) f1.push({ level: 'fail', msg: '没有任何部件（drawable parts）' });
  if (model.parts.length > LIMIT_PARTS) f1.push({ level: 'fail', msg: '部件数 ' + model.parts.length + ' 超过上限 ' + LIMIT_PARTS });
  var seen = {}, orders = {};
  for (i = 0; i < model.parts.length; i++) {
    var p = model.parts[i];
    if (seen[p.id]) f1.push({ level: 'fail', msg: '部件 id 重复: ' + p.id });
    seen[p.id] = true;
    if (!(num(p.width, 0) > 0) || !(num(p.height, 0) > 0)) f1.push({ level: 'fail', msg: '部件 ' + p.id + ' 的 width/height 必须 > 0' });
    if (!isArr(p.color) || p.color.length !== 4) f1.push({ level: 'fail', msg: '部件 ' + p.id + ' 的 color 必须是 [r,g,b,a]' });
    else for (j = 0; j < 4; j++) if (!isNum(num(p.color[j], NaN)) || num(p.color[j], 0) < 0 || num(p.color[j], 0) > 1) f1.push({ level: 'fail', msg: '部件 ' + p.id + ' 的 color[' + j + '] 必须在 0..1' });
    if (!isObj(p.transform)) f1.push({ level: 'fail', msg: '部件 ' + p.id + ' 缺少 transform' });
    if (orders[p.order]) f1.push({ level: 'fail', msg: '绘制顺序 order 重复: ' + p.order + '（' + orders[p.order] + ' / ' + p.id + '）—— order 决定叠放层级，必须唯一' });
    orders[p.order] = p.id;
  }
  var sortedOrders = model.parts.map(function (x) { return x.order; }).sort(function (a, b) { return a - b; });
  for (i = 0; i < sortedOrders.length; i++) {
    if (sortedOrders[i] !== i) { f1.push({ level: 'warn', msg: 'order 不连续（' + sortedOrders.join(',') + '）—— 建议规范化成 0..n-1' }); break; }
  }
  var dSeen = {};
  for (i = 0; i < (model.deformers || []).length; i++) {
    var d0 = model.deformers[i];
    if (dSeen[d0.id]) f1.push({ level: 'fail', msg: 'deformer id 重复: ' + d0.id });
    dSeen[d0.id] = true;
    if (seen[d0.id]) f1.push({ level: 'fail', msg: 'deformer id 与部件 id 冲突: ' + d0.id + '（.iki 里 part 与 deformer 共用同一命名空间，上游 parseIkiModel 会拒绝 —— 建议给骨骼加 bone_ 前缀）' });
    if (d0.kind !== undefined && d0.kind !== 'matrix' && d0.kind !== 'warp') f1.push({ level: 'fail', msg: 'deformer ' + d0.id + ' 的 kind 非法: ' + d0.kind });
  }
  var paSeen = {};
  for (i = 0; i < model.parameters.length; i++) {
    if (paSeen[model.parameters[i].id]) f1.push({ level: 'fail', msg: '参数 id 重复: ' + model.parameters[i].id });
    paSeen[model.parameters[i].id] = true;
  }
  checks.push({ id: 'R1', name: '结构契约', findings: f1, metric: model.parts.length + ' 部件 / ' + (model.deformers || []).length + ' deformer / ' + model.parameters.length + ' 参数' });

  // ── R2 命名规范 ──
  var f2 = [], unclassified = [];
  var pairRoles = { eyelid: 1, eyeball: 1, brow: 1, arm: 1, hand: 1, leg: 1 };
  for (i = 0; i < model.parts.length; i++) {
    var p2 = model.parts[i];
    if (!/^[a-z][a-z0-9_]*$/.test(p2.id)) f2.push({ level: 'fail', msg: '部件名不合规「' + p2.id + '」：只允许小写字母/数字/下划线，且以字母开头' });
    if (p2.id.length > 48) f2.push({ level: 'warn', msg: '部件名过长: ' + p2.id });
    var em = emojiScan(p2.id);
    if (em.length) f2.push({ level: 'fail', msg: '部件名含 emoji/符号图标「' + p2.id + '」—— 命名只用语义词根（技能 emoji-icons）' });
    var cls = classifyRole(p2.id, null);
    if (cls.role === 'unclassified') unclassified.push(p2.id);
    var base = cls.role.split('_')[0];
    if (pairRoles[base] && !cls.side) f2.push({ level: 'warn', msg: '成对部件「' + p2.id + '」缺少侧别后缀（_L / _R = 角色自身左右）—— 自动绑定会把它当左/中轴处理' });
  }
  if (unclassified.length) f2.push({ level: 'warn', msg: '未识别命名 ' + unclassified.length + ' 个: ' + unclassified.slice(0, 8).join(', ') + (unclassified.length > 8 ? ' …' : '') + '（词根表见 rig.parts.json 的 namingConvention）' });
  checks.push({ id: 'R2', name: '部件命名规范', findings: f2, metric: model.parts.length + ' 个部件，未识别 ' + unclassified.length });

  // ── R3 deformer 树 ──
  var f3 = [];
  var dMap = {};
  for (i = 0; i < (model.deformers || []).length; i++) dMap[model.deformers[i].id] = model.deformers[i];
  for (i = 0; i < (model.deformers || []).length; i++) {
    var d = model.deformers[i];
    if (d.parent !== undefined) {
      if (!dMap[d.parent]) f3.push({ level: 'fail', msg: 'deformer ' + d.id + ' 的父节点不存在: ' + d.parent });
      else if (d.parent === d.id) f3.push({ level: 'fail', msg: 'deformer ' + d.id + ' 把自己当父节点' });
      else if (d.kind === 'warp' && dMap[d.parent].kind === 'warp') f3.push({ level: 'fail', msg: 'warp deformer ' + d.id + ' 的父节点是 warp（上游契约：父必须是 matrix deformer）' });
    }
    var walk = d, depth = 0, guard = 0;
    while (walk && walk.parent !== undefined && guard++ < 200) {
      depth++;
      walk = dMap[walk.parent];
      if (walk && walk.id === d.id) { f3.push({ level: 'fail', msg: 'deformer 树存在环（经过 ' + d.id + '）' }); break; }
    }
    if (depth > 8) f3.push({ level: 'fail', msg: 'deformer ' + d.id + ' 深度 ' + depth + ' 超过 8 级' });
    if (d.kind === 'warp') {
      var g = d.grid || {};
      if (!(num(g.cols, 0) >= 1) || !(num(g.rows, 0) >= 1)) f3.push({ level: 'fail', msg: 'warp deformer ' + d.id + ' 的 grid cols/rows 必须 >= 1' });
      var need = (num(g.cols, 0) + 1) * (num(g.rows, 0) + 1) * 2;
      if (!isArr(g.points) || g.points.length !== need) f3.push({ level: 'fail', msg: 'warp deformer ' + d.id + ' 的 grid.points 长度应为 ' + need + '（(cols+1)(rows+1)×2），实际 ' + ((g.points || []).length) });
      else {
        // 上游校验要求：rest 网格为规则轴对齐晶格（列 x 递增、行 y 递减）
        var cols = num(g.cols, 0), rows = num(g.rows, 0), bad = 0;
        for (var r = 0; r <= rows; r++) for (var c = 0; c <= cols; c++) {
          var idx = (r * (cols + 1) + c) * 2;
          if (c > 0 && !(g.points[idx] > g.points[((r * (cols + 1) + c - 1) * 2)])) bad++;
          if (r > 0 && !(g.points[idx + 1] < g.points[(((r - 1) * (cols + 1) + c) * 2) + 1])) bad++;
        }
        if (bad) f3.push({ level: 'fail', msg: 'warp deformer ' + d.id + ' 的 rest 网格不是规则晶格（列 x 必须严格递增、行 y 严格递减；上游校验会拒绝）' });
      }
      if (isArr(d.warps) && d.warp2d) f3.push({ level: 'fail', msg: 'warp deformer ' + d.id + ' 同时带了 warps 与 warp2d（上游契约：二选一）' });
      if (isArr(d.warps) && d.warps.length > 1) f3.push({ level: 'fail', msg: 'warp deformer ' + d.id + ' 有 ' + d.warps.length + ' 个 grid warp（上游 v1 只允许 1 个）' });
      var warps = d.warps || [];
      for (j = 0; j < warps.length; j++) {
        if (!paSeen[warps[j].parameter]) f3.push({ level: 'fail', msg: 'warp deformer ' + d.id + ' 的 warp 引用了未声明参数: ' + warps[j].parameter });
        var kfs = warps[j].keyforms || [];
        if (!kfs.length) f3.push({ level: 'fail', msg: 'warp deformer ' + d.id + ' 的 warp 没有 keyform' });
        for (k = 1; k < kfs.length; k++) if (!(num(kfs[k].value, 0) > num(kfs[k - 1].value, 0))) f3.push({ level: 'fail', msg: 'warp deformer ' + d.id + ' 的 keyform value 必须严格递增' });
        for (k = 0; k < kfs.length; k++) if (!isArr(kfs[k].offsets) || kfs[k].offsets.length !== ((d.grid.points || []).length)) f3.push({ level: 'fail', msg: 'warp deformer ' + d.id + ' 的 keyform offsets 长度必须等于 grid.points 长度' });
      }
    } else {
      if (!isObj(d.pivot)) f3.push({ level: 'fail', msg: 'matrix deformer ' + d.id + ' 缺少 pivot' });
    }
  }
  checks.push({ id: 'R3', name: 'deformer 树', findings: f3, metric: (model.deformers || []).length + ' 个节点' });

  // ── R4 蒙皮完整性 ──
  var f4 = [], unbound = [];
  for (i = 0; i < model.parts.length; i++) {
    var p4 = model.parts[i];
    if (p4.deformer === undefined) { unbound.push(p4.id); continue; }
    var dd = dMap[p4.deformer];
    if (!dd) f4.push({ level: 'fail', msg: '部件 ' + p4.id + ' 引用了不存在的 deformer: ' + p4.deformer });
    else if (dd.kind === 'warp' && !p4.mesh) f4.push({ level: 'fail', msg: '部件 ' + p4.id + ' 挂在 warp deformer 上但没有 mesh（上游契约：warp 子部件必须带网格）' });
  }
  if (unbound.length) f4.push({ level: 'fail', msg: '未挂 deformer 的部件 ' + unbound.length + ' 个: ' + unbound.slice(0, 8).join(', ') + (unbound.length > 8 ? ' …' : '') + '（跑 rig_edit op=bind.auto，或 op=part.assign 手动挂）' });
  var orphan = [];
  for (i = 0; i < (model.deformers || []).length; i++) {
    var did = model.deformers[i].id, used = false;
    for (j = 0; j < model.parts.length; j++) if (model.parts[j].deformer === did) used = true;
    for (j = 0; j < (model.deformers || []).length; j++) if (model.deformers[j].parent === did) used = true;
    for (j = 0; j < (model.physicsChains || []).length; j++) if (model.physicsChains[j].anchorDeformer === did) used = true;
    if (!used && model.deformers[i].kind !== 'warp') orphan.push(did);
  }
  if (orphan.length) f4.push({ level: 'warn', msg: '空 deformer（没有部件/子节点/角链挂载）: ' + orphan.join(', ') });
  checks.push({ id: 'R4', name: '蒙皮完整性', findings: f4, metric: (model.parts.length - unbound.length) + '/' + model.parts.length + ' 部件已挂载' });

  // ── R5 参数与绑定 ──
  var f5 = [];
  for (i = 0; i < model.parameters.length; i++) {
    var pa = model.parameters[i];
    if (!(num(pa.max, 0) > num(pa.min, 0))) f5.push({ level: 'fail', msg: '参数 ' + pa.id + ' 的 max 必须 > min（' + pa.min + '..' + pa.max + '）' });
    if (num(pa.default, 0) < num(pa.min, 0) || num(pa.default, 0) > num(pa.max, 1)) f5.push({ level: 'fail', msg: '参数 ' + pa.id + ' 的 default ' + pa.default + ' 不在 [' + pa.min + ',' + pa.max + '] 内' });
    var std = STD_PARAM_BY_ID[pa.id];
    if (std && (num(pa.min, 0) !== std.min || num(pa.max, 1) !== std.max)) {
      f5.push({ level: 'warn', msg: '标准参数 ' + pa.id + ' 的范围被改成了 ' + pa.min + '..' + pa.max + '（上游标准 ' + std.min + '..' + std.max + '）—— 会破坏宿主 1:1 映射（口型/眨眼/头部追踪）' });
    }
    if (!std && !/^Param[A-Z][A-Za-z0-9]*$/.test(pa.id)) f5.push({ level: 'warn', msg: '自定义参数 ' + pa.id + ' 不符合 ParamXxx 命名习惯（宿主无法识别，需自带驱动）' });
  }
  function checkBindings(list, where) {
    for (var q = 0; q < (list || []).length; q++) {
      var b = list[q];
      if (!paSeen[b.parameter]) f5.push({ level: 'fail', msg: where + ' 引用了未声明参数: ' + b.parameter });
      if (CHANNELS.indexOf(b.channel) < 0) f5.push({ level: 'fail', msg: where + ' 的 channel 非法: ' + b.channel });
      if (!isNum(num(b.from, NaN)) || !isNum(num(b.to, NaN))) f5.push({ level: 'fail', msg: where + ' 的 from/to 必须是数值' });
    }
  }
  for (i = 0; i < model.parts.length; i++) {
    checkBindings(model.parts[i].bindings, '部件 ' + model.parts[i].id);
    var ws = model.parts[i].warps || [];
    for (j = 0; j < ws.length; j++) {
      if (!paSeen[ws[j].parameter]) f5.push({ level: 'fail', msg: '部件 ' + model.parts[i].id + ' 的 warp 引用了未声明参数: ' + ws[j].parameter });
      var vlist = ws[j].keyforms || [];
      if (!vlist.length) f5.push({ level: 'fail', msg: '部件 ' + model.parts[i].id + ' 的 warp 没有 keyform' });
      if (model.parts[i].mesh) {
        for (k = 0; k < vlist.length; k++) {
          if ((vlist[k].offsets || []).length !== model.parts[i].mesh.vertices.length) f5.push({ level: 'fail', msg: '部件 ' + model.parts[i].id + ' 的 warp keyform offsets 长度必须等于 mesh 顶点数×2' });
        }
      } else f5.push({ level: 'fail', msg: '部件 ' + model.parts[i].id + ' 有 warps 但没有 mesh（上游契约：warps 需要 mesh）' });
      for (k = 1; k < vlist.length; k++) if (!(num(vlist[k].value, 0) > num(vlist[k - 1].value, 0))) f5.push({ level: 'fail', msg: '部件 ' + model.parts[i].id + ' 的 warp keyform value 必须严格递增' });
    }
    if (model.parts[i].clip && isArr(model.parts[i].clip.masks)) {
      for (j = 0; j < model.parts[i].clip.masks.length; j++) {
        var mid = model.parts[i].clip.masks[j], mpart = null;
        for (k = 0; k < model.parts.length; k++) if (model.parts[k].id === mid) mpart = model.parts[k];
        if (!mpart) f5.push({ level: 'fail', msg: '部件 ' + model.parts[i].id + ' 的遮罩不存在: ' + mid });
        else {
          if (!mpart.mesh) f5.push({ level: 'fail', msg: '遮罩部件 ' + mid + ' 没有 mesh（上游契约：遮罩必须带网格）' });
          if (mpart.clip) f5.push({ level: 'fail', msg: '遮罩部件 ' + mid + ' 自身被裁剪（上游契约：遮罩是扁平的）' });
        }
      }
    }
  }
  for (i = 0; i < (model.deformers || []).length; i++) {
    checkBindings(model.deformers[i].bindings, 'deformer ' + model.deformers[i].id);
    for (j = 0; j < (model.deformers[i].bindings || []).length; j++) {
      if (model.deformers[i].bindings[j].channel === 'opacity') f5.push({ level: 'fail', msg: 'deformer ' + model.deformers[i].id + ' 用了 opacity 通道（矩阵无法表达透明度，上游契约只允许矩阵通道）' });
    }
  }
  checks.push({ id: 'R5', name: '参数与绑定', findings: f5, metric: model.parameters.length + ' 参数，' + model.parts.filter(function (x) { return (x.bindings || []).length || (x.warps || []).length; }).length + ' 个部件有局部绑定' });

  // ── R6 物理语义 ──
  var f6 = [];
  var chainOuts = {};
  var springOuts = {};
  for (i = 0; i < (model.physics || []).length; i++) {
    var sp = model.physics[i];
    springOuts[sp.output.parameter] = (springOuts[sp.output.parameter] || 0) + 1;
    if (!paSeen[sp.input.parameter]) f6.push({ level: 'fail', msg: '弹簧 ' + sp.id + ' 的输入参数未声明: ' + sp.input.parameter });
    if (!paSeen[sp.output.parameter]) f6.push({ level: 'fail', msg: '弹簧 ' + sp.id + ' 的输出参数未声明: ' + sp.output.parameter });
    if (sp.input.parameter === sp.output.parameter) f6.push({ level: 'fail', msg: '弹簧 ' + sp.id + ' 的输入与输出是同一参数（上游契约：二级运动不得写宿主驱动参数）' });
    if (!(num(sp.mass, 0) > 0)) f6.push({ level: 'fail', msg: '弹簧 ' + sp.id + ' 的 mass 必须 > 0' });
    if (!(num(sp.stiffness, 0) > 0)) f6.push({ level: 'fail', msg: '弹簧 ' + sp.id + ' 的 stiffness 必须 > 0' });
    if (num(sp.damping, 0) < 0) f6.push({ level: 'fail', msg: '弹簧 ' + sp.id + ' 的 damping 必须 >= 0' });
    if (num(sp.damping, 0) >= Math.sqrt(4 * num(sp.mass, 1) * num(sp.stiffness, 1))) f6.push({ level: 'warn', msg: '弹簧 ' + sp.id + ' 处于过阻尼区（damping ' + sp.damping + ' ≥ 2√(mk)=' + round3(Math.sqrt(4 * num(sp.mass, 1) * num(sp.stiffness, 1))) + '）—— 二级运动不会回摆' });
    var outP = null;
    for (j = 0; j < model.parameters.length; j++) if (model.parameters[j].id === sp.output.parameter) outP = model.parameters[j];
    if (outP && !STD_PARAM_BY_ID[sp.output.parameter] && num(outP.default, 0) !== 0) f6.push({ level: 'warn', msg: '弹簧输出参数 ' + sp.output.parameter + ' 的默认值不是 0 —— 运行时会叠加在默认值上（静态姿态即偏移）' });
  }
  for (i = 0; i < (model.physicsChains || []).length; i++) {
    var ch = model.physicsChains[i];
    var anchor = dMap[ch.anchorDeformer];
    if (!anchor) f6.push({ level: 'fail', msg: '角链 ' + ch.id + ' 的锚 deformer 不存在: ' + ch.anchorDeformer });
    else if (anchor.kind === 'warp') f6.push({ level: 'fail', msg: '角链 ' + ch.id + ' 的锚是 warp deformer（上游契约：必须是 matrix）' });
    if (!isArr(ch.segments) || !ch.segments.length) f6.push({ level: 'fail', msg: '角链 ' + ch.id + ' 没有段' });
    if (num(ch.gravity && ch.gravity.strength, 0) < 0) f6.push({ level: 'fail', msg: '角链 ' + ch.id + ' 的 gravity.strength 必须 >= 0' });
    for (j = 0; j < (ch.segments || []).length; j++) {
      var sg = ch.segments[j];
      if (!paSeen[sg.output.parameter]) f6.push({ level: 'fail', msg: '角链 ' + ch.id + ' 第 ' + (j + 1) + ' 段的输出参数未声明: ' + sg.output.parameter });
      if (!(num(sg.mass, 0) > 0)) f6.push({ level: 'fail', msg: '角链 ' + ch.id + ' 第 ' + (j + 1) + ' 段的 mass 必须 > 0' });
      if (num(sg.stiffness, 0) < 0) f6.push({ level: 'fail', msg: '角链 ' + ch.id + ' 第 ' + (j + 1) + ' 段的 stiffness 必须 >= 0' });
      if (num(sg.damping, 0) < 0) f6.push({ level: 'fail', msg: '角链 ' + ch.id + ' 第 ' + (j + 1) + ' 段的 damping 必须 >= 0' });
      chainOuts[sg.output.parameter] = (chainOuts[sg.output.parameter] || 0) + 1;
    }
  }
  var multi = [];
  for (var kk in chainOuts) if (chainOuts[kk] > 1) multi.push(kk + '×' + chainOuts[kk]);
  if (multi.length) f6.push({ level: 'warn', msg: '多个链段写同一输出参数: ' + multi.join(', ') + '（宿主无法分辨来源，建议每段独立参数）' });
  var multiSpring = [];
  for (var sk in springOuts) if (springOuts[sk] > 1) multiSpring.push(sk + '×' + springOuts[sk]);
  if (multiSpring.length) f6.push({ level: 'warn', msg: '多个弹簧写同一输出参数: ' + multiSpring.join(', ') + '（两级二级运动叠加在同一条参数上，抖动会互相抵消/放大，建议各自独立参数）' });
  var cross = [];
  for (var ck in springOuts) if (chainOuts[ck]) cross.push(ck);
  if (cross.length) f6.push({ level: 'warn', msg: '同一输出参数既被弹簧写又被角链写: ' + cross.join(', ') });
  checks.push({ id: 'R6', name: '物理语义', findings: f6, metric: (model.physics || []).length + ' 弹簧 / ' + (model.physicsChains || []).length + ' 角链' });

  // ── R7 部件重叠（同部位拆层检查）──
  // ★ 口径（重要）：**重叠本身是正常的美术结构**（前发盖后发、睫毛盖眼白…），不能一律当缺陷。
  //   因此：① 两个同部位部件的包围盒**几乎完全重合**（位置+尺寸都在 5% 内）→ FAIL（多为重复导入/忘删）；
  //        ② 其余同部位重叠 → warn，提示确认是否需要裁切到边缘或用 clip 遮罩（拆层检查的人工复核线索）。
  //   阈值为"交叠面积 / 较小者面积"，用 overlapThreshold 调整（默认 0.6）。
  var f7 = [];
  var store7 = new ParamStore(model);
  var worlds7 = {};
  try { worlds7 = resolveDeformerWorlds(model.deformers || [], store7); } catch (e) { f7.push({ level: 'fail', msg: 'deformer 世界矩阵求解失败（先修 R3）: ' + ((e && e.message) || e) }); }
  var byRole = {};
  for (i = 0; i < model.parts.length; i++) {
    var pr = model.parts[i];
    var rl = classifyRole(pr.id, null).role.split('_')[0];
    if (!byRole[rl]) byRole[rl] = [];
    byRole[rl].push(pr);
  }
  var overlaps = [], exactDup = [];
  for (var rk in byRole) {
    var arr = byRole[rk];
    for (i = 0; i < arr.length; i++) {
      for (j = i + 1; j < arr.length; j++) {
        var qa, qb;
        try { qa = quadAABB(partQuad(arr[i], worlds7, store7, model)); qb = quadAABB(partQuad(arr[j], worlds7, store7, model)); } catch (e) { continue; }
        var ratio = aabbOverlapRatio(qa, qb);
        if (ratio <= overlapThreshold) continue;
        var wa = qa.x1 - qa.x0, ha = qa.y1 - qa.y0, wb = qb.x1 - qb.x0, hb = qb.y1 - qb.y0;
        var sameW = Math.abs(wa - wb) / Math.max(wa, wb, 1) < 0.05;
        var sameH = Math.abs(ha - hb) / Math.max(ha, hb, 1) < 0.05;
        var entry = { parts: arr[i].id + ' ↔ ' + arr[j].id, role: rk, ratio: round3(ratio) };
        if (ratio > 0.98 && sameW && sameH) exactDup.push(entry);
        else overlaps.push(entry);
      }
    }
  }
  exactDup.sort(function (a, b) { return b.ratio - a.ratio; });
  overlaps.sort(function (a, b) { return b.ratio - a.ratio; });
  for (i = 0; i < Math.min(exactDup.length, 6); i++) {
    f7.push({ level: 'fail', msg: '同部位「' + exactDup[i].role + '」两个部件位置与尺寸几乎完全重合：' + exactDup[i].parts + '（重叠 ' + Math.round(exactDup[i].ratio * 100) + '%）—— 多为重复导入或忘删图层（若确为有意叠放，请合并成一个部件）' });
  }
  if (exactDup.length > 6) f7.push({ level: 'warn', msg: '还有 ' + (exactDup.length - 6) + ' 组完全重合未逐条列出' });
  for (i = 0; i < Math.min(overlaps.length, 6); i++) {
    f7.push({ level: 'warn', msg: '同部位「' + overlaps[i].role + '」部件重叠 ' + Math.round(overlaps[i].ratio * 100) + '%：' + overlaps[i].parts + ' —— 前后层次重叠本身正常；若两者应各自独立活动，需把上层裁切到下层边缘或用 clip 遮罩' });
  }
  if (overlaps.length > 6) f7.push({ level: 'warn', msg: '还有 ' + (overlaps.length - 6) + ' 组同部位重叠未逐条列出' });
  checks.push({ id: 'R7', name: '部件重叠（拆层）', findings: f7, metric: '阈值 ' + overlapThreshold + '，完全重合 ' + exactDup.length + ' 组 / 一般重叠 ' + overlaps.length + ' 组' });

  // ── R8 变换与网格 sanity ──
  var f8 = [];
  var W8 = num(model.canvas.width, 1000), H8 = num(model.canvas.height, 1000);
  var outside = [];
  for (i = 0; i < model.parts.length; i++) {
    var p8 = model.parts[i];
    var box = { x0: num(p8.transform && p8.transform.x, 0) - num(p8.width, 0) / 2, x1: num(p8.transform && p8.transform.x, 0) + num(p8.width, 0) / 2, y0: num(p8.transform && p8.transform.y, 0) - num(p8.height, 0) / 2, y1: num(p8.transform && p8.transform.y, 0) + num(p8.height, 0) / 2 };
    // 模型空间：x ∈ [-W/2, W/2]、y ∈ [-H/2, H/2]（原点 = 画布中心），容差 20%
    if (box.x0 < -W8 * 0.6 || box.x1 > W8 * 0.6 || box.y0 < -H8 * 0.6 || box.y1 > H8 * 0.6) outside.push(p8.id);
    var sc8 = p8.transform && has(p8.transform, 'scaleX') ? num(p8.transform.scaleX, 1) : 1;
    var sc8y = p8.transform && has(p8.transform, 'scaleY') ? num(p8.transform.scaleY, 1) : 1;
    // ★ 基变换 scale 为 0 是**正常模式**（绑定值按上游语义累加到基变换，scale 通道由绑定提供）：
    //   只有"该通道没有绑定"时，缩放为 0 才意味着这个部件永远不会被看见。
    var bindCh = function (ch) {
      return (p8.bindings || []).some(function (b) { return b.channel === ch; });
    };
    if (Math.abs(sc8) < 1e-4 && !bindCh('scaleX')) f8.push({ level: 'warn', msg: '部件 ' + p8.id + ' 的 scaleX 接近 0 且该通道没有绑定（不会被看见）' });
    if (Math.abs(sc8y) < 1e-4 && !bindCh('scaleY')) f8.push({ level: 'warn', msg: '部件 ' + p8.id + ' 的 scaleY 接近 0 且该通道没有绑定（不会被看见）' });
    if (p8.mesh) {
      var mv = p8.mesh.vertices, mu = p8.mesh.uvs, mi = p8.mesh.indices;
      if (!isArr(mv) || mv.length < 6 || mv.length % 2 !== 0) f8.push({ level: 'fail', msg: '部件 ' + p8.id + ' 的 mesh.vertices 必须是偶数长度且 >= 6' });
      if (!isArr(mu) || mu.length !== (mv || []).length) f8.push({ level: 'fail', msg: '部件 ' + p8.id + ' 的 mesh.uvs 必须与 vertices 等长' });
      if (!isArr(mi) || mi.length % 3 !== 0 || !mi.length) f8.push({ level: 'fail', msg: '部件 ' + p8.id + ' 的 mesh.indices 长度必须是 3 的倍数' });
      else for (k = 0; k < mi.length; k++) if (mi[k] < 0 || mi[k] >= mv.length / 2) f8.push({ level: 'fail', msg: '部件 ' + p8.id + ' 的 mesh.indices 越界: ' + mi[k] });
    }
    var q8;
    try { q8 = partQuad(p8, worlds7, store7, model); } catch (e) { f8.push({ level: 'fail', msg: '部件 ' + p8.id + ' 变换求解失败: ' + ((e && e.message) || e) }); continue; }
    if (quadArea(q8) < 0.5) f8.push({ level: 'warn', msg: '部件 ' + p8.id + ' 的当前面积接近 0（退化四边形）' });
  }
  if (outside.length) f8.push({ level: 'warn', msg: '超出画布边界（含 20% 容差）的部件 ' + outside.length + ' 个: ' + outside.slice(0, 8).join(', ') + '（角色整体构图可能偏移）' });
  checks.push({ id: 'R8', name: '变换与网格', findings: f8, metric: model.parts.filter(function (x) { return !!x.mesh; }).length + ' 个网格部件' });

  // ── R9 产物确定性且自包含 ──
  var f9 = [];
  var h1 = renderPreviewHtml(model, {});
  var h2 = renderPreviewHtml(model, {});
  if (h1 !== h2) f9.push({ level: 'fail', msg: '预览 HTML 两次渲染不一致（含时间戳/随机数？产物必须可 diff）' });
  var s1 = renderSpritesSvg(model, {});
  var s2 = renderSpritesSvg(model, {});
  if (s1 !== s2) f9.push({ level: 'fail', msg: '部件布局 SVG 两次渲染不一致' });
  if (/<script[^>]+src=/i.test(h1)) f9.push({ level: 'fail', msg: '预览 HTML 引用了外部脚本（必须自包含）' });
  if (/<link[^>]+href=["']https?:/i.test(h1)) f9.push({ level: 'fail', msg: '预览 HTML 引用了外部样式' });
  if (/https?:\/\//.test(h1.replace(/xmlns="[^"]*"/g, ''))) {
    var ext = (h1.match(/https?:\/\/[^\s"'<>)]+/g) || []).filter(function (u) { return !/^https?:\/\/(www\.)?w3\.org/.test(u); });
    if (ext.length) f9.push({ level: 'warn', msg: '预览 HTML 含外部 URL ' + ext.length + ' 处（纹理应内联为 data: URI）: ' + ext.slice(0, 3).join(', ') });
  }
  if (/https?:\/\//.test(s1.replace(/xmlns="[^"]*"/g, ''))) f9.push({ level: 'warn', msg: '部件布局 SVG 含外部 URL' });
  checks.push({ id: 'R9', name: '产物确定且自包含', findings: f9, metric: 'HTML ' + h1.length + ' 字符 / SVG ' + s1.length + ' 字符' });

  // ── 汇总 ──
  var failed = 0, warned = 0;
  for (i = 0; i < checks.length; i++) {
    var fs = checks[i].findings || [], hasFail = false;
    for (j = 0; j < fs.length; j++) {
      if (fs[j].level === 'fail') hasFail = true;
      if (fs[j].level === 'warn') warned++;
    }
    checks[i].pass = !hasFail;
    checks[i].fails = fs.filter(function (x) { return x.level === 'fail'; }).length;
    checks[i].warns = fs.filter(function (x) { return x.level === 'warn'; }).length;
    if (!checks[i].pass) failed++;
  }
  var report = {
    schema: 'paircode.rig.verify/1',
    ts: nowIso(),
    model: path,
    name: model.name,
    passed: checks.length - failed,
    total: checks.length,
    allPassed: failed === 0,
    totalWarnings: warned,
    checks: checks,
  };
  if (reportPath) writeJsonFile(ctx, reportPath, report);
  return JSON.stringify(report, null, 2);
}
// ── 工具声明 ───────────────────────────────────────────────
var TOOL_DEFS = [
  {
    name: 'rig_model',
    description: '角色工程（.iki 文档）的创建与查看。真相源就是上游 .iki v1 格式（version/name/canvas/parameters/parts/deformers/physics/physicsChains），坐标口径照上游：模型空间原点 = 画布中心、+y 向上、rotation 逆时针为正。action=create 建工程（默认带 16 个上游标准参数；可一次带入整批部件 parts=[…]）/ get 看概要 / rename / reset（清空结构、保留参数与画布）。',
    usageGuide: '起步：rig_model action=create name=char canvas={width:1024,height:1024}（可带 parts=[{id:"body",width:400,height:600,order:0}] 一次把部件搭好）→ rig_import source=psd src=角色.psd（有分层素材时）→ rig_verify → rig_export format=html。已有工程先 action=get 看盘点（部件/deformer/参数/物理数量、按部位分布）；手搭/补部件用 rig_add。',
    category: '创作',
    parameters: {
      type: 'object',
      properties: {
        action: { type: 'string', description: 'create（默认）/ get / rename / reset' },
        path: { type: 'string', description: '可选：工程路径（默认 <主项目根>/rig.model.iki.json；相对主项目根解析）' },
        name: { type: 'string', description: '可选：角色名（create/rename）' },
        width: { type: 'number', description: '可选：画布宽（默认 1000）' },
        height: { type: 'number', description: '可选：画布高（默认 1000）' },
        canvas: { type: 'object', description: '可选：{width,height}（等价于 width/height）' },
        stdParams: { type: 'boolean', description: '可选（create）：是否写入 16 个上游标准参数（默认 true）' },
        parts: { type: 'array', description: '可选（create）：一次带入的部件数组（每项与 rig_add 的 parts 项 / rig_edit 的 part.add 同构：{id,width,height,x,y,order?,color?,deformer?}）；任一项非法则整批不落盘' },
        overwrite: { type: 'boolean', description: '可选（create）：已存在时是否重建（默认 false）' },
        force: { type: 'boolean', description: '可选（reset）：确认清空结构' },
      },
    },
  },
  {
    name: 'rig_import',
    description: '分层导入 → 部件（parts）。source=psd 直接解析 PSD 结构（自研最小解析器：只读图层名/矩形/可见性/组，不读像素、不解压 RLE；上限 32MB），source=manifest 走图层清单 JSON（超大 PSD 或外部工具导出时的替代路径）。导入后默认跑自动绑定（按命名约定建 deformer 树 + 头部/眼/嘴/眉参数绑定 + 头发角链 + 衣物弹簧）。',
    usageGuide: 'rig_import source=psd src=assets/char.psd → 自动分层 + 自动绑定；再 rig_verify 看 9 项判据。PSD 太大报错时改 source=manifest（清单字段：canvas{width,height} + layers[{name,x,y,width,height,visible,group}]，x/y 为 PSD 像素左上角）。已有工程追加图层用 mode=append；覆盖用 overwrite=true。atlas={width,height} 可同时写入纹理 UV。',
    category: '创作',
    parameters: {
      type: 'object',
      properties: {
        source: { type: 'string', description: 'psd（默认）| manifest（图层清单 JSON）' },
        src: { type: 'string', description: '源文件路径（PSD 或清单 JSON）' },
        path: { type: 'string', description: '可选：工程路径（默认 <主项目根>/rig.model.iki.json；相对主项目根解析）' },
        mode: { type: 'string', description: '可选：replace（默认，清空后导入）| append（追加）' },
        name: { type: 'string', description: '可选：工程不存在时新建的角色名' },
        visibleOnly: { type: 'boolean', description: '可选：跳过隐藏图层（默认 true）' },
        autoRig: { type: 'boolean', description: '可选：导入后自动绑定（默认 true）' },
        hairSegments: { type: 'number', description: '可选：头发角链段数（默认 2）' },
        atlas: { type: 'object', description: '可选：纹理图集尺寸 {width,height}（写入 part.texture.uv）' },
        atlasIndex: { type: 'number', description: '可选：纹理下标（默认 0）' },
        overwrite: { type: 'boolean', description: '可选：工程已存在时确认覆盖 parts' },
      },
      required: [],
    },
  },
  {
    name: 'rig_add',
    description: '批量新增部件到角色工程（一次调用任意多个，事务式）。parts=[{id:"hair_front",width:180,height:120,x:0,y:210,order:8},{id:"face",width:300,height:340,order:2}] —— 每项与 rig_edit 的 part.add 同构（id 必填、width/height > 0；可带 x/y/rotation/scaleX/scaleY/opacity/order/color[4]/deformer）。也支持单件写法（id/width/height 写在同一层）。全部校验通过才落盘一次，任一项失败整批不写入（工程保持原样）。',
    usageGuide: '手搭/补部件：rig_add parts=[{"id":"body","width":400,"height":600,"order":0},{"id":"face","width":300,"height":340,"x":0,"y":140,"order":2}]。id 会被 slug 规范化（小写 + _L/_R 侧别），重复 id 直接报错；order 不传自动排在最后。部件建好后用 rig_edit 绑骨骼与参数（deformer.add → part.assign / part.bind，或一键 bind.auto），再 rig_export 出预览、rig_verify 自检。有分层素材时优先 rig_import（自动分层 + 自动绑定）。',
    category: '创作',
    parameters: {
      type: 'object',
      properties: {
        path: { type: 'string', description: '可选：工程路径（默认 <主项目根>/rig.model.iki.json；相对主项目根解析）' },
        parts: { type: 'array', description: '部件定义数组（每项 {id,width,height,x?,y?,order?,color?,deformer?}；批量写法）' },
        id: { type: 'string', description: '可选（单件写法）：部件 id（必填；会自动 slug：小写 + _L/_R）' },
        width: { type: 'number', description: '可选（单件写法）：部件宽（模型空间单位，> 0）' },
        height: { type: 'number', description: '可选（单件写法）：部件高（> 0）' },
        x: { type: 'number', description: '可选（单件写法）：x（模型空间，原点 = 画布中心）' },
        y: { type: 'number', description: '可选（单件写法）：y（模型空间，+y 向上）' },
        order: { type: 'number', description: '可选（单件写法）：绘制顺序（不传自动排最后）' },
        deformer: { type: 'string', description: '可选（单件写法）：所属 deformer id（须已存在）' },
      },
    },
  },
  {
    name: 'rig_edit',
    description: '角色工程命令式编辑（一次 ops 批量应用，任一 op 失败则整批不落盘）。支持：部件（part.add/set/rename/remove/order/bind/unbind/assign/mesh/clip）、deformer（deformer.add/set/remove/bind/unbind、warp.add）、参数（param.add/set/remove）、物理（physics.add/remove、chain.add/remove）、自动绑定（bind.auto）、纹理（texture.set）。纯新增部件用 rig_add（parts 数组一次成型）更省。返回结构化结果（ok/applied/log/summary）。',
    usageGuide: '示例：{"op":"deformer.add","id":"head","parent":"body","pivot":{"x":0,"y":180}} / {"op":"part.assign","ids":["hair_front"],"deformer":"hair"} / {"op":"part.bind","id":"eye_white_L","parameter":"ParamEyeLOpen","channel":"scaleY","from":0.1,"to":1} / {"op":"warp.add","id":"warp_hair","parent":"head","parts":["hair_front"],"parameter":"ParamAngleZ","preset":"sway","amount":10} / {"op":"physics.add","id":"ph_skirt","input":"ParamAngleZ","output":"ParamClothSwayX","stiffness":18,"damping":0.35,"mass":1} / {"op":"bind.auto"}。参数引用必须先声明（param.add）；改完跑 rig_verify。 ★ 批量：一次调用可传任意多条 op（按顺序应用；任一 op 失败整批不落盘，错误信息里会点明第几条）—— 部件/骨骼多时一次调用成型（如 [{op:"part.add",...},{op:"part.add",...},{op:"chain.add",...}]），别一条一条调。',
    category: '创作',
    parameters: {
      type: 'object',
      properties: {
        path: { type: 'string', description: '可选：工程路径（默认 <主项目根>/rig.model.iki.json；相对主项目根解析）' },
        ops: { type: 'array', description: '编辑命令数组（按顺序应用）' },
        op: { type: 'string', description: '可选：单条 op 名（与同层参数合成为一条命令）' },
        ids: { type: 'array', description: '可选：目标部件 id 数组' },
        id: { type: 'string', description: '可选：单个目标（部件/deformer/参数/物理 id，按 op 语义）' },
        role: { type: 'string', description: '可选：按部位过滤目标（如 hair / eyelid_l / arm_r）' },
        name: { type: 'string', description: '可选：按名字子串过滤目标' },
        props: { type: 'object', description: '可选（part.set/deformer.set/param.set）：要合并的属性' },
        part: { type: 'object', description: '可选（part.add）：{id,width,height,x,y,order?,color?,deformer?}' },
        x: { type: 'number', description: '可选：部件/骨骼的 x（模型空间）' },
        y: { type: 'number', description: '可选：部件/骨骼的 y（模型空间，+y 向上）' },
        width: { type: 'number', description: '可选：部件宽（模型空间单位）' },
        height: { type: 'number', description: '可选：部件高' },
        order: { type: 'number', description: '可选：绘制顺序（或用 part.order 指定目标序号）' },
        parameter: { type: 'string', description: '可选：参数 id（绑定/warp/物理）' },
        channel: { type: 'string', description: '可选：translateX/translateY/rotate/scaleX/scaleY/opacity' },
        from: { description: '可选：绑定起点值（参数归一化 0 处）' },
        to: { description: '可选：绑定终点值（参数归一化 1 处）' },
        parent: { type: 'string', description: '可选：父 deformer id' },
        pivot: { type: 'object', description: '可选：deformer 转轴 {x,y}（模型空间）' },
        deformer: { type: 'string', description: '可选：目标 deformer id（part.assign/set）' },
        to_: { type: 'string', description: '（见 to）' },
        masks: { type: 'array', description: '可选（part.clip）：遮罩部件 id 数组' },
        mesh: { type: 'boolean', description: '（见 cols/rows，part.mesh 用）' },
        cols: { type: 'number', description: '可选：网格/变形网格列数' },
        rows: { type: 'number', description: '可选：网格/变形网格行数' },
        parts: { type: 'array', description: '可选（warp.add）：目标部件 id 数组' },
        preset: { type: 'string', description: '可选（warp.add）：bend / sway / bulge / none' },
        amount: { type: 'number', description: '可选（warp.add）：形变幅度' },
        min: { type: 'number', description: '可选（param.add/set）：参数下限' },
        max: { type: 'number', description: '可选（param.add/set）：参数上限' },
        default: { type: 'number', description: '可选（param.add/set）：参数默认值' },
        input: { description: '可选（physics.add）：{parameter,weight} 或直接参数 id' },
        output: { description: '可选（physics.add）：{parameter,scale} 或直接参数 id' },
        stiffness: { type: 'number', description: '可选：弹簧刚度（> 0）' },
        damping: { type: 'number', description: '可选：阻尼（[0,1) 区间最自然）' },
        mass: { type: 'number', description: '可选：质量（> 0）' },
        anchorDeformer: { type: 'string', description: '可选（chain.add）：角链锚 deformer（必须是 matrix）' },
        gravity: { type: 'object', description: '可选（chain.add）：{angle,strength}' },
        segments: { type: 'array', description: '可选（chain.add）：段数组 [{output,restAngle?,mass,stiffness,damping}]' },
        source: { type: 'string', description: '可选（texture.set）：data:image/... 数据 URI' },
      },
    },
  },
  {
    name: 'rig_export',
    description: '导出角色产物（全部文本、可 diff）：format=html 自包含可交互预览（参数滑块 + 弹簧/角链二级运动 + 演示动画，无外链、无脚本依赖；纹理 data URI 内联时直接绘制，否则占位灰阶 + 描边结构视图）；format=svg 部件布局图（默认参数姿态、绘制顺序自后向前）；format=parts 图层命名清单 JSON（role 分类 + 模型空间矩形 + 建议绑定，交给美术按约定重导图层）；format=iki 规范化 .iki 文档副本。',
    usageGuide: '出预览：rig_export format=html → 在面板/浏览器打开核对（可截图核验）；美术对接：format=parts → rig.parts.json；结构审查：format=svg；交上游引擎：format=iki。产物已存在需 overwrite=true。',
    category: '创作',
    parameters: {
      type: 'object',
      properties: {
        path: { type: 'string', description: '可选：工程路径（默认 <主项目根>/rig.model.iki.json；相对主项目根解析）' },
        format: { type: 'string', description: 'html（默认）| svg | parts | iki' },
        out: { type: 'string', description: '可选：输出路径（默认按 format 派生：<主项目根>/rig.preview.html / rig.sprites.svg / rig.parts.json / rig.model.export.iki.json；相对主项目根解析）' },
        overwrite: { type: 'boolean', description: '可选：产物已存在是否覆盖（默认 false）' },
      },
    },
  },
  {
    name: 'rig_verify',
    description: '角色工程专项自检（9 项判据）：R1 结构契约（.iki version/canvas/必填字段/id 唯一/order 唯一且连续）、R2 命名规范（小写词根 + _L/_R 侧别、禁 emoji、成对部件缺侧别）、R3 deformer 树（父存在/无环/深度/warp 父必须 matrix/网格晶格规则性）、R4 蒙皮完整性（部件是否都挂 deformer、warp 子部件是否有网格、空节点）、R5 参数与绑定（引用完整、min<max、default 在域内、通道合法、deformer 禁用 opacity、标准参数范围被改）、R6 物理语义（mass/stiffness/damping 范围、input≠output、锚必须是 matrix、多段共用输出参数）、R7 部件重叠（同部位高度重叠 = 拆层不完整，默认阈值 0.6）、R8 变换与网格（画布越界、退化四边形、mesh 顶点/UV/索引一致性）、R9 产物确定且自包含（两次渲染位级一致、无外链）。任一 R 项 FAIL 视为模型不达标。',
    usageGuide: '导入/编辑后必跑；报告旁挂 rig.verify.json（面板与 Agent 同源）。R2 抓命名、R4 抓"没绑上"，R7 抓拆层错误，R6 抓物理参数越界；R9 保证产物可 diff。阈值可调 overlapThreshold（0.05..0.95）。',
    category: '创作',
    parameters: {
      type: 'object',
      properties: {
        path: { type: 'string', description: '可选：工程路径（默认 <主项目根>/rig.model.iki.json；相对主项目根解析）' },
        report: { type: 'string', description: '可选：旁挂报告路径（默认 <主项目根>/rig.verify.json；传空串则不写）' },
        overlapThreshold: { type: 'number', description: '可选：同部位部件重叠比阈值（默认 0.6）' },
      },
    },
  },
];

var IMPLS = {
  rig_model: rigModel,
  rig_import: rigImport,
  rig_add: rigAdd,
  rig_edit: rigEdit,
  rig_export: rigExport,
  rig_verify: rigVerify,
};

// 双轨入口：goja 沙箱把本文件当函数体执行 → 顶层 return；Node 侧（测试/打包）走 module.exports。
var PLUGIN = {
  name: 'tool-rig',
  purpose: '2D 角色创作域工具面（纯 goja 零依赖）：真相源是上游 .iki v1 文档（可直接被 @ikijs/engine 加载）—— PSD 分层导入（自研结构解析）→ 命名约定自动绑定（deformer 树 + 参数绑定 + 弹簧/角链二级运动）→ 命令式编辑 → 自包含可交互预览 HTML / 布局 SVG / 图层命名清单 → 9 项判据自检',
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
      if (ctx && ctx.logger && typeof ctx.logger === 'function') {
        ctx.logger('tool-rig').info('已注册 ' + TOOL_DEFS.length + ' 个工具（goja 轨 · 零依赖 · .iki v1 真相源）');
      }
    } catch (_) { /* ignore */ }
  },
};

if (typeof module !== 'undefined' && module.exports) module.exports = PLUGIN;
return PLUGIN;
