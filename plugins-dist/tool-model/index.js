// tool-model — 3D/CAD 创作域工具面（纯 goja 零依赖）
//
// 设计原则（对齐《创作域支持方案》§4.2 与**公开格式契约**）：
//   · 工程真相源 = 本插件的参数化工程文档（默认 model.json，文本、可 diff、可 review）：
//       { format:"paircode.model/1", unit:"mm", up:"z", params[], parts[] }
//     部件 = 几何表达式树 + 变换 + 材质；所有几何运算在**三角网格**上做（BSP 布尔）。
//   · 坐标口径照 CAD 通行惯例（OpenSCAD / JSCAD / STL）：**右手系、+z 向上、单位毫米**；
//     primitive 的 center 默认在原点（几何中心），size 为**全尺寸**——与 JSCAD primitives 同口径，
//     故可用 @jscad/modeling 做逐项交叉验证（体积 / 包围盒 / 三角面数）。
//   · 产物真相源 = **公开格式，不是自家方言**：
//       glTF 2.0（Khronos 标准 ISO/IEC 12113；本插件产出**自包含** .gltf / .glb——buffer 走
//         base64 data URI；导出时按规范把工程的 z-up 转成 **y-up**；验收用官方 gltf-validator 校验）
//       STL（3D 打印事实标准，binary/ascii）/ OBJ（Wavefront）
//   · 参数化：数值字段可写**表达式**（引用 params，如 "wall*2"、"h/2"），自研安全求值器（不用 eval）。
//   · 预览产物 = 自包含 HTML：内嵌几何内核 + 自研 WebGL 渲染器（轨道相机 / 光照 / 线框）+
//     参数滑块（改参数 → 浏览器内实时重算几何，与 Agent 侧跑的是同一份内核代码）。
//
// 沙箱能力：无 require / Buffer / Node API —— 一律走 ctx.fs（readFile / writeFile / exists / stat）。
// 外部真值（**仅用于测试，不是运行时依赖**）：@jscad/modeling 2.13（CSG 交叉验证）、gltf-validator 2.0（官方）。

// ── 常量 ───────────────────────────────────────────────────
var FORMAT = 'paircode.model/1';
var PROJECT_NAME = 'model.json';
var GLTF_VERSION = '2.0';

var LIMIT_PARTS = 200;          // 部件数上限
var LIMIT_PARAMS = 64;          // 参数数上限
var LIMIT_TRIS_TOTAL = 400000;  // 全模型三角面上限
var LIMIT_TRIS_PART = 60000;    // 单部件三角面上限
var LIMIT_BOOL_INPUT = 30000;   // 单次布尔运算输入面上限（BSP 成本约束）
var LIMIT_SEGMENTS = 512;       // 曲面分段上限
var TWIST_LAYERS_DEFAULT = 12;  // 扭转挤出的默认分层数（口径同 JSCAD extrudeTwist）
var LIMIT_ARRAY = 256;          // 阵列数量上限
var LIMIT_OBJ_BYTES = 64 << 20; // 导入 STL/OBJ 上限（64MB）
var LIMIT_EXPR_DEPTH = 32;      // 表达式括号/递归深度上限
var LIMIT_EDGE_OPS = 64;        // 单次倒角/圆角处理的边数上限（裁剪体顶点枚举为 O(n³)）
var EDGE_ANGLE_DEFAULT = 30;    // 认定为"棱"的最小折角（度）：更平的相邻面视为同一曲面
var ARC_SEG_DEFAULT = 8;        // 圆角弧面的离散段数（每段一个切平面）
var EDGE_NEIGHBOR_HOPS = 6;     // 边的"邻域"跳数（判定斜面只切到边附近，见 edgeCutSafety）
var LIMIT_CONVEX_FACES = 4000;  // 走"H-rep 直接构造"的凸体面数上限（面平面去重是 O(F²)）
var LIMIT_CONVEX_PLANES = 140;  // H-rep 直接构造的平面数上限（顶点枚举 O(n³)）
var LIMIT_FILLET_EDGES = 8;     // 单次圆角的边数上限（圆角要布尔，见 meshFillet 的成本说明）
var LIMIT_RB_PLANES = 32;       // rolling-ball 圆角直接构造的凸体平面数上限（三平面枚举 O(n³)，且片数∝棱数×segs）

var EPS = 1e-9;
var EPS_AREA = 1e-10;           // 退化三角面面积阈值

// 焊接容差相对 eps 的比例（★ 实测确定，2026-09-17）：
//   T 缝消解（meshAbsorbToEnds + meshFillTJunctions）会暴露出"横向错开 1~1.4 eps"的成对顶点，
//   0.5×eps 的焊接容差合并不了它们 ⇒ 残留单边（真缺口）。实测扫描（60×40×6 板 − 4×∅6 柱 @24 段）：
//     0.5× → 真缺口 10、单边 11      1× → 真缺口 1、单边 3
//     1.5× → 真缺口 0、单边 0  ← 拐点   2×/3× → 同为 0（无进一步收益，仅多合并真实特征）
//   体积影响：1.5× 时相对变化 1.3e-6（可忽略）。故取 1.5。
var WELD_RATIO = 1.5;
// 空间分辨率（对齐 JSCAD maths.EPS = 1e-5，其注释为"空间分辨率 = 100 纳米"）
var EPS_SPATIAL = 1e-5;
var EPS_PLANE = 1e-6;           // BSP 平面判定（与 csg.js 同量级）
var WELD_TOL = 1e-6;            // 顶点焊接容差（相对模型尺度，见 meshWeld）

// 材质默认（glTF pbrMetallicRoughness 语义）
var DEFAULT_MATERIAL = { color: '#b0b4bd', metallic: 0.0, roughness: 0.75 };

// ── 基础工具 ───────────────────────────────────────────────
function fail(msg) { throw new Error(msg); }

function isNum(v) { return typeof v === 'number' && isFinite(v); }
function isStr(v) { return typeof v === 'string'; }
function isObj(v) { return !!v && typeof v === 'object' && !(v instanceof Array); }
function isArr(v) { return v instanceof Array; }

// 有限小数格式化：避免浮点噪声进入文本产物（1e-9 以下归零，最多 6 位有效小数）
function fmtNum(v, digits) {
  if (!isFinite(v)) return '0';
  var d = digits === undefined ? 6 : digits;
  var r = Number(v.toFixed(d));
  return String(r);
}
// 归一化 -0 / 极小值（几何计算里 -0 与 1e-17 会让产物不确定）
function cleanNum(v, digits) {
  if (!isFinite(v)) return 0;
  var d = digits === undefined ? 9 : digits;
  var r = Number(v.toFixed(d));
  return r === 0 ? 0 : r;
}

function toNumber(v, what) {
  if (isNum(v)) return v;
  if (isStr(v)) { var n = Number(v); if (isFinite(n)) return n; }
  fail((what || '值') + '必须是有限数字（收到 ' + JSON.stringify(v) + '）');
}
// 三维数值数组（[x,y,z] 或 {x,y,z}）
function toVec3(v, what, dflt) {
  if (v === undefined || v === null) return dflt ? dflt.slice() : [0, 0, 0];
  if (isArr(v)) {
    if (v.length < 3) fail((what || '向量') + '必须是长度 3 的数组');
    return [toNumber(v[0]), toNumber(v[1]), toNumber(v[2])];
  }
  if (isObj(v)) return [toNumber(v.x || 0), toNumber(v.y || 0), toNumber(v.z || 0)];
  if (isNum(v)) return [v, v, v];
  fail((what || '向量') + '必须是 [x,y,z] 或 {x,y,z}');
}
function vec3ToArr(v, what, dflt) {
  var a = toVec3(v, what, dflt);
  if (a[0] === 0 && a[1] === 0 && a[2] === 0 && dflt) return dflt.slice();
  return a;
}

// ── 数学：vec3 ─────────────────────────────────────────────
function vAdd(a, b) { return [a[0] + b[0], a[1] + b[1], a[2] + b[2]]; }
function vSub(a, b) { return [a[0] - b[0], a[1] - b[1], a[2] - b[2]]; }
function vMul(a, s) { return [a[0] * s, a[1] * s, a[2] * s]; }
function vDot(a, b) { return a[0] * b[0] + a[1] * b[1] + a[2] * b[2]; }
function vCross(a, b) {
  return [a[1] * b[2] - a[2] * b[1], a[2] * b[0] - a[0] * b[2], a[0] * b[1] - a[1] * b[0]];
}
function vLen(a) { return Math.sqrt(vDot(a, a)); }
function vDist(a, b) { return vLen(vSub(a, b)); }
function vNorm(a) {
  var l = vLen(a);
  return l > EPS ? [a[0] / l, a[1] / l, a[2] / l] : [0, 0, 0];
}
function vFinite(a) { return isFinite(a[0]) && isFinite(a[1]) && isFinite(a[2]); }
function vEq(a, b, tol) {
  var t = tol === undefined ? WELD_TOL : tol;
  return Math.abs(a[0] - b[0]) <= t && Math.abs(a[1] - b[1]) <= t && Math.abs(a[2] - b[2]) <= t;
}

// ── 数学：mat4（**行主序** 16 元素，点按列向量右乘：p' = M · p）──────
function m4id() { return [1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1]; }

function m4mul(a, b) {
  var r = new Array(16);
  for (var i = 0; i < 4; i++) {
    for (var j = 0; j < 4; j++) {
      r[i * 4 + j] = a[i * 4] * b[j] + a[i * 4 + 1] * b[4 + j] + a[i * 4 + 2] * b[8 + j] + a[i * 4 + 3] * b[12 + j];
    }
  }
  return r;
}

function m4applyPoint(m, p) {
  var x = p[0], y = p[1], z = p[2];
  return [
    m[0] * x + m[1] * y + m[2] * z + m[3],
    m[4] * x + m[5] * y + m[6] * z + m[7],
    m[8] * x + m[9] * y + m[10] * z + m[11]
  ];
}
function m4applyDir(m, p) {
  var x = p[0], y = p[1], z = p[2];
  return [
    m[0] * x + m[1] * y + m[2] * z,
    m[4] * x + m[5] * y + m[6] * z,
    m[8] * x + m[9] * y + m[10] * z
  ];
}

function m4translate(t) { return [1, 0, 0, t[0], 0, 1, 0, t[1], 0, 0, 1, t[2], 0, 0, 0, 1]; }
function m4scale(s) { return [s[0], 0, 0, 0, 0, s[1], 0, 0, 0, 0, s[2], 0, 0, 0, 0, 1]; }

function m4area3(m) {
  // 左上 3x3 的行列式（= 体积缩放因子）
  return m[0] * (m[5] * m[10] - m[6] * m[9])
    - m[1] * (m[4] * m[10] - m[6] * m[8])
    + m[2] * (m[4] * m[9] - m[5] * m[8]);
}
function m4det(m) { return m4area3(m); }

function m4invert(m) {
  var det = m4area3(m);
  if (Math.abs(det) < 1e-12) fail('变换矩阵不可逆（体积缩放为 0）');
  var a = m[0], b = m[1], c = m[2], d = m[4], e = m[5], f = m[6], g = m[8], h = m[9], i = m[10];
  // 左上 3x3 的逆（伴随矩阵 / det，行主序）
  var i0 = (e * i - f * h) / det, i1 = (c * h - b * i) / det, i2 = (b * f - c * e) / det;
  var i3 = (f * g - d * i) / det, i4 = (a * i - c * g) / det, i5 = (c * d - a * f) / det;
  var i6 = (d * h - e * g) / det, i7 = (b * g - a * h) / det, i8 = (a * e - b * d) / det;
  var t = [m[3], m[7], m[11]];
  return [
    i0, i1, i2, -(i0 * t[0] + i1 * t[1] + i2 * t[2]),
    i3, i4, i5, -(i3 * t[0] + i4 * t[1] + i5 * t[2]),
    i6, i7, i8, -(i6 * t[0] + i7 * t[1] + i8 * t[2]),
    0, 0, 0, 1
  ];
}

function m4rotAxis(axis, deg) {
  var a = vNorm(axis);
  if (vLen(a) < 0.5) fail('旋转轴不能为零向量');
  var r = deg * Math.PI / 180, c = Math.cos(r), s = Math.sin(r), t = 1 - c;
  var x = a[0], y = a[1], z = a[2];
  return [
    t * x * x + c, t * x * y - s * z, t * x * z + s * y, 0,
    t * x * y + s * z, t * y * y + c, t * y * z - s * x, 0,
    t * x * z - s * y, t * y * z + s * x, t * z * z + c, 0,
    0, 0, 0, 1
  ];
}

// 变换组合顺序（照 JSCAD transforms.rotate 语义）：先绕 X、再绕 Y、最后绕 Z（内旋）
// 矩阵 = Rz · Ry · Rx
function m4rotXYZ(r) {
  var m = m4id();
  if (r[2]) m = m4mul(m4rotAxis([0, 0, 1], r[2]), m);
  if (r[1]) m = m4mul(m4rotAxis([0, 1, 0], r[1]), m);
  if (r[0]) m = m4mul(m4rotAxis([1, 0, 0], r[0]), m);
  return m;
}

// 行主序 3x3 逆（用于法线矩阵）
function m3inv(m) {
  var a = m[0] * (m[4] * m[8] - m[5] * m[7]) - m[1] * (m[3] * m[8] - m[5] * m[6]) + m[2] * (m[3] * m[7] - m[4] * m[6]);
  if (Math.abs(a) < 1e-12) return null;
  return [
    (m[4] * m[8] - m[5] * m[7]) / a, (m[2] * m[7] - m[1] * m[8]) / a, (m[1] * m[5] - m[2] * m[4]) / a,
    (m[5] * m[6] - m[3] * m[8]) / a, (m[0] * m[8] - m[2] * m[6]) / a, (m[2] * m[3] - m[0] * m[5]) / a,
    (m[3] * m[7] - m[4] * m[6]) / a, (m[1] * m[6] - m[0] * m[7]) / a, (m[0] * m[4] - m[1] * m[3]) / a
  ];
}

// TRS → 矩阵（M = T · R · S，R = Rz·Ry·Rx）
function m4fromTRS(trs) {
  var t = trs && trs.translate ? toVec3(trs.translate, 'translate', [0, 0, 0]) : [0, 0, 0];
  var r = trs && trs.rotate ? toVec3(trs.rotate, 'rotate', [0, 0, 0]) : [0, 0, 0];
  var s = trs && trs.scale ? toVec3(trs.scale, 'scale', [1, 1, 1]) : [1, 1, 1];
  var m = m4mul(m4rotXYZ(r), m4scale(s));
  m[3] = t[0]; m[7] = t[1]; m[11] = t[2];
  return m;
}

// ── mesh：索引三角网格 ───────────────────────────────────────
// 结构：{ positions: [[x,y,z],...], indices: [[i0,i1,i2],...] }
// 约定：三角面顶点**逆时针（从外部看）= 外向法线**；有向体积 > 0 表示闭合且朝外。
function meshNew(positions, indices) {
  return { positions: positions || [], indices: indices || [] };
}
function meshClone(m) {
  var p = [], i = [], k;
  for (k = 0; k < m.positions.length; k++) p.push(m.positions[k].slice());
  for (k = 0; k < m.indices.length; k++) i.push(m.indices[k].slice());
  return { positions: p, indices: i };
}
function meshTriCount(m) { return m.indices.length; }
function meshVertCount(m) { return m.positions.length; }

function meshTransform(m, mat) {
  var out = meshNew([], []), k;
  for (k = 0; k < m.positions.length; k++) out.positions.push(m4applyPoint(mat, m.positions[k]));
  var flip = m4det(mat) < 0; // 镜像/负缩放会翻转绕向 → 反向索引保持"外向"
  for (k = 0; k < m.indices.length; k++) {
    var f = m.indices[k];
    out.indices.push(flip ? [f[0], f[2], f[1]] : [f[0], f[1], f[2]]);
  }
  return out;
}

// 合并多个 mesh（拼接顶点表 + 索引偏移）
function meshMerge(list) {
  var out = meshNew([], []), k, j;
  for (k = 0; k < list.length; k++) {
    var m = list[k], off = out.positions.length;
    for (j = 0; j < m.positions.length; j++) out.positions.push(m.positions[j].slice());
    for (j = 0; j < m.indices.length; j++) {
      var f = m.indices[j];
      out.indices.push([f[0] + off, f[1] + off, f[2] + off]);
    }
  }
  return out;
}

function meshTriangle(m, k) {
  var f = m.indices[k];
  return [m.positions[f[0]], m.positions[f[1]], m.positions[f[2]]];
}

// 有向体积（散度定理；闭合且外向为正）——口径与 JSCAD poly3.measureSignedVolume 一致
function meshVolume(m) {
  var vol = 0;
  for (var k = 0; k < m.indices.length; k++) {
    var t = meshTriangle(m, k);
    vol += vDot(t[0], vCross(t[1], t[2])) / 6;
  }
  return vol;
}
function meshArea(m) {
  var a = 0;
  for (var k = 0; k < m.indices.length; k++) {
    var t = meshTriangle(m, k);
    a += vLen(vCross(vSub(t[1], t[0]), vSub(t[2], t[0]))) / 2;
  }
  return a;
}
function meshBbox(m) {
  if (!m.positions.length) return [[0, 0, 0], [0, 0, 0]];
  var mn = m.positions[0].slice(), mx = m.positions[0].slice();
  for (var k = 1; k < m.positions.length; k++) {
    var p = m.positions[k];
    for (var c = 0; c < 3; c++) {
      if (p[c] < mn[c]) mn[c] = p[c];
      if (p[c] > mx[c]) mx[c] = p[c];
    }
  }
  return [mn, mx];
}
function meshSize(m) {
  var b = meshBbox(m);
  return [b[1][0] - b[0][0], b[1][1] - b[0][1], b[1][2] - b[0][2]];
}
function meshCenter(m) {
  var b = meshBbox(m);
  return [(b[0][0] + b[1][0]) / 2, (b[0][1] + b[1][1]) / 2, (b[0][2] + b[1][2]) / 2];
}
// 面积加权质心（比包围盒中心更符合"零件重心"直觉）
function meshCentroid(m) {
  var sum = [0, 0, 0], wsum = 0;
  for (var k = 0; k < m.indices.length; k++) {
    var t = meshTriangle(m, k);
    var a = vLen(vCross(vSub(t[1], t[0]), vSub(t[2], t[0]))) / 2;
    if (a <= 0) continue;
    sum = vAdd(sum, vMul(vAdd(vAdd(t[0], t[1]), t[2]), a / 3));
    wsum += a;
  }
  if (wsum <= EPS) return meshCenter(m);
  return vMul(sum, 1 / wsum);
}

// 顶点焊接：把空间上重合的顶点合并（BSP 分裂产生的重复顶点必须先焊接，
// 否则拓扑统计把"本应共享的边"算成边界边）。返回 { mesh, map }
function meshWeld(m, tol) {
  var t = tol === undefined ? WELD_TOL : tol;
  var reps = [], repIndex = [], map = [];
  var bucket = {};
  function key(x, y, z) { return x + ':' + y + ':' + z; }
  function cell(v) { return [Math.round(v[0] / t), Math.round(v[1] / t), Math.round(v[2] / t)]; }
  for (var k = 0; k < m.positions.length; k++) {
    var p = m.positions[k];
    var c = cell(p), found = -1;
    for (var dx = -1; dx <= 1 && found < 0; dx++) {
      for (var dy = -1; dy <= 1 && found < 0; dy++) {
        for (var dz = -1; dz <= 1 && found < 0; dz++) {
          var list = bucket[key(c[0] + dx, c[1] + dy, c[2] + dz)];
          if (!list) continue;
          for (var q = 0; q < list.length; q++) {
            var idx = list[q];
            if (vEq(reps[idx], p, t)) { found = idx; break; }
          }
        }
      }
    }
    if (found >= 0) { map.push(repIndex[found]); continue; }
    reps.push(p.slice());
    repIndex.push(reps.length - 1);
    var kk = key(c[0], c[1], c[2]);
    (bucket[kk] = bucket[kk] || []).push(reps.length - 1);
    map.push(reps.length - 1);
  }
  var out = meshNew(reps, []);
  for (var f = 0; f < m.indices.length; f++) {
    var tri = m.indices[f];
    var a = map[tri[0]], b = map[tri[1]], d = map[tri[2]];
    if (a === b || b === d || a === d) continue; // 焊接后退化
    out.indices.push([a, b, d]);
  }
  return { mesh: out, map: map };
}

// 清理：去掉零面积/重复索引三角面
function meshDropDegenerate(m, epsArea) {
  var lim = epsArea === undefined ? EPS_AREA : epsArea;
  var out = meshNew(m.positions, []);
  for (var k = 0; k < m.indices.length; k++) {
    var f = m.indices[k];
    if (f[0] === f[1] || f[1] === f[2] || f[0] === f[2]) continue;
    var t = meshTriangle(m, k);
    var a = vLen(vCross(vSub(t[1], t[0]), vSub(t[2], t[0]))) / 2;
    if (a <= lim) continue;
    out.indices.push([f[0], f[1], f[2]]);
  }
  return out;
}

// 去掉未被引用的孤立顶点（导出前必做：否则 glTF 顶点数虚高）
function meshCompact(m) {
  var used = {}, k;
  for (k = 0; k < m.indices.length; k++) {
    used[m.indices[k][0]] = 1; used[m.indices[k][1]] = 1; used[m.indices[k][2]] = 1;
  }
  var map = {}, pos = [], out = meshNew(pos, []);
  for (k = 0; k < m.positions.length; k++) {
    if (!used[k]) continue;
    map[k] = pos.length;
    pos.push(m.positions[k].slice());
  }
  for (k = 0; k < m.indices.length; k++) {
    var f = m.indices[k];
    out.indices.push([map[f[0]], map[f[1]], map[f[2]]]);
  }
  return out;
}

// 拓扑统计（水密性/流形性判据的数据源）：
//   边界边 = 只被 1 个面共享；非流形边 = 被 >2 个面共享；绕向不一致 = 同一条边被两个面以**相同方向**使用
function meshTopology(m, tol) {
  var welded = meshWeld(m, tol).mesh;
  var edges = {}, boundary = 0, nonManifold = 0, flipped = 0, edgeCount = 0;
  for (var k = 0; k < welded.indices.length; k++) {
    var f = welded.indices[k];
    for (var e = 0; e < 3; e++) {
      var a = f[e], b = f[(e + 1) % 3];
      var lo = a < b ? a : b, hi = a < b ? b : a;
      var key = lo + '_' + hi;
      var rec = edges[key];
      if (!rec) { rec = edges[key] = { count: 0, dirs: [] }; edgeCount++; }
      rec.count++;
      rec.dirs.push(a < b ? 1 : -1); // 相对于 (lo→hi) 的方向
    }
  }
  for (var kk in edges) {
    if (!edges.hasOwnProperty(kk)) continue;
    var r = edges[kk];
    if (r.count === 1) boundary++;
    else if (r.count > 2) nonManifold++;
    else if (r.dirs[0] === r.dirs[1]) flipped++; // 两面同向 → 绕向不一致（共边方向应相反）
  }
  var V = welded.positions.length, E = edgeCount, F = welded.indices.length;
  return {
    vertices: V, edges: E, faces: F,
    euler: V - E + F,
    boundaryEdges: boundary, nonManifoldEdges: nonManifold, flippedEdgePairs: flipped,
    watertight: boundary === 0 && nonManifold === 0,
    mesh: welded
  };
}

// 容差化水密判定（M4 判据的核心工具，也是诚实报告"缝"的手段）：
//   严格判据"每条边恰好被 2 个面以相反方向使用"对**三角网格的布尔输出**过于苛刻——
//   相邻面片的边分段方式可能不同（一条长边 vs 由多个顶点串成的短边链），几何上严丝合缝，
//   拓扑上却对不上。这里对每条"严格未配对的半边"再做一次**区间覆盖**判定：
//   若存在方向相反、落在该边上的若干半边，其投影区间并集完整覆盖 [0,1]，则视为已配对（T 缝）。
//   于是能区分：真正的开边界（缺口）vs 分段不同但几何闭合的 T 缝。
function meshWatertightReport(m, tol) {
  var eps = tol === undefined ? meshEpsilon(m) : tol;
  // 焊接容差须与 meshRepair 一致（见 WELD_RATIO）：判定必须看"生产实际交付的那个网格"，
  // 否则会出现"修复后仍报缺口"的口径错位（实测踩过：一边 0.5×eps 一边 1.5×eps ⇒ 数字对不上）。
  var welded = meshWeld(m, eps * WELD_RATIO).mesh;
  var pos = welded.positions, tris = welded.indices;
  var i, e, k;
  var half = [];
  for (i = 0; i < tris.length; i++) {
    for (e = 0; e < 3; e++) half.push([tris[i][e], tris[i][(e + 1) % 3]]);
  }
  // 有向半边集合：闭合网格里每个有向半边恰好出现一次，故"配对"= **反向半边存在**
  //（不是"同一有向边出现多次"——那是把非流形也算进来了）
  var has = {};
  for (k = 0; k < half.length; k++) has[half[k][0] + '_' + half[k][1]] = 1;
  var unpairedStrict = 0;
  for (k = 0; k < half.length; k++) {
    if (!has[half[k][1] + '_' + half[k][0]]) unpairedStrict++;
  }
  if (!half.length) {
    return { strictUnpaired: 0, tolerantUnpaired: 0, tJunctions: 0, watertightStrict: true, watertightTolerant: true, halvedEdges: 0 };
  }
  // 空间索引（格子 → 半边下标）
  var bb = meshBbox(welded);
  var diag = Math.max(vLen(vSub(bb[1], bb[0])), eps * 1000);
  var cs = diag / 48;
  var grid = {};
  function gk(c) { return c[0] + ',' + c[1] + ',' + c[2]; }
  function gc(p) { return [Math.floor(p[0] / cs), Math.floor(p[1] / cs), Math.floor(p[2] / cs)]; }
  for (k = 0; k < half.length; k++) {
    var pa0 = pos[half[k][0]], pb0 = pos[half[k][1]];
    var len0 = vDist(pa0, pb0);
    var steps0 = Math.max(1, Math.ceil(len0 / cs));
    var prev0 = null;
    for (var s0 = 0; s0 <= steps0; s0++) {
      var t0 = s0 / steps0;
      var cc = gc([pa0[0] + (pb0[0] - pa0[0]) * t0, pa0[1] + (pb0[1] - pa0[1]) * t0, pa0[2] + (pb0[2] - pa0[2]) * t0]);
      var kk2 = gk(cc);
      if (kk2 === prev0) continue;
      prev0 = kk2;
      (grid[kk2] = grid[kk2] || []).push(k);
    }
  }
  function candidates(a, b) {
    var pa = pos[a], pb = pos[b];
    var c0 = gc([Math.min(pa[0], pb[0]) - eps, Math.min(pa[1], pb[1]) - eps, Math.min(pa[2], pb[2]) - eps]);
    var c1 = gc([Math.max(pa[0], pb[0]) + eps, Math.max(pa[1], pb[1]) + eps, Math.max(pa[2], pb[2]) + eps]);
    var out = [], seen = {};
    for (var x = c0[0]; x <= c1[0]; x++) {
      for (var y = c0[1]; y <= c1[1]; y++) {
        for (var z = c0[2]; z <= c1[2]; z++) {
          var list = grid[gk([x, y, z])];
          if (!list) continue;
          for (var q = 0; q < list.length; q++) {
            if (seen[list[q]]) continue;
            seen[list[q]] = 1;
            out.push(list[q]);
          }
        }
      }
    }
    return out;
  }
  function onSeg(p, a, ab, L2) {
    var t = vDot(vSub(p, a), ab) / L2;
    if (t < -eps / Math.sqrt(L2) || t > 1 + eps / Math.sqrt(L2)) return null;
    var proj = [a[0] + ab[0] * t, a[1] + ab[1] * t, a[2] + ab[2] * t];
    if (vDist(proj, p) > eps) return null;
    return t;
  }
  var tolerantUnpaired = 0, tj = 0, gapLength = 0, gapMax = 0;
  for (k = 0; k < half.length; k++) {
    var a = half[k][0], b = half[k][1];
    if (has[b + '_' + a]) continue; // 已严格配对
    var pa = pos[a], pb = pos[b];
    var ab = vSub(pb, pa), L2 = vDot(ab, ab);
    if (L2 <= 0) { tolerantUnpaired++; continue; }
    var ivs = [], cand = candidates(a, b);
    for (var q2 = 0; q2 < cand.length; q2++) {
      var g = half[cand[q2]];
      if (g[0] === a && g[1] === b) continue;
      var pc = pos[g[0]], pd = pos[g[1]];
      if (vDot(vSub(pd, pc), ab) >= 0) continue; // 需要方向相反
      var tc = onSeg(pc, pa, ab, L2), td = onSeg(pd, pa, ab, L2);
      if (tc === null || td === null) continue;
      ivs.push([Math.min(tc, td), Math.max(tc, td)]);
    }
    ivs.sort(function (p1, p2) { return p1[0] - p2[0]; });
    var cur = 0;
    for (var w = 0; w < ivs.length; w++) {
      if (ivs[w][0] > cur + 1e-9) break;
      if (ivs[w][1] > cur) cur = ivs[w][1];
    }
    if (cur >= 1 - 1e-6) tj++;
    else {
      tolerantUnpaired++;
      // ★ 缝的**几何长度**（mm）：未覆盖的弧长。这是"可打印性"的直接度量 ——
      //   条数会被"同一条缝被切成很多段"放大，长度才是真实缺陷量（M4 判据用它更贴题）。
      var miss = (1 - cur) * Math.sqrt(L2);
      gapLength += miss;
      if (miss > gapMax) gapMax = miss;
    }
  }
  return {
    strictUnpaired: unpairedStrict,
    tolerantUnpaired: tolerantUnpaired,
    tJunctions: tj,
    gapLength: gapLength,     // 未覆盖缝总长（mm）
    gapMax: gapMax,           // 单条最长缝（mm）
    watertightStrict: unpairedStrict === 0,
    watertightTolerant: tolerantUnpaired === 0,
    // "可打印"口径：缝在数值噪声内（总长 ≤ eps/10，单条 ≤ eps/10）即视为闭合
    printableClosed: gapLength <= eps * 0.1 && gapMax <= eps * 0.1,
    halfEdges: half.length,
    epsilon: eps
  };
}

// 连通分量（按面共享顶点聚类）——多实体阵列/多部件是正常的，单体多连通则可疑
function meshComponents(m) {
  var parent = [];
  for (var i = 0; i < m.positions.length; i++) parent.push(i);
  function find(x) { while (parent[x] !== x) { parent[x] = parent[parent[x]]; x = parent[x]; } return x; }
  function uni(a, b) { var ra = find(a), rb = find(b); if (ra !== rb) parent[rb] = ra; }
  for (var k = 0; k < m.indices.length; k++) {
    var f = m.indices[k];
    uni(f[0], f[1]); uni(f[1], f[2]);
  }
  var roots = {}, count = 0;
  for (var j = 0; j < m.indices.length; j++) {
    var r = find(m.indices[j][0]);
    if (!roots[r]) { roots[r] = { tris: 0 }; count++; }
    roots[r].tris++;
  }
  return { count: count, roots: roots };
}

// 按包围盒对齐（align）：把实体移到指定方位（min/center/max × 3 轴）
function meshAlign(m, spec) {
  var b = meshBbox(m), size = meshSize(m), c = meshCenter(m);
  var t = [0, 0, 0];
  var modes = isStr(spec) ? [spec, spec, spec] : [spec[0], spec[1], spec[2]];
  for (var i = 0; i < 3; i++) {
    var md = modes[i];
    if (md === 'min') t[i] = -b[0][i];
    else if (md === 'max') t[i] = -b[1][i];
    else if (md === 'center' || md === undefined || md === null) t[i] = -c[i];
    else fail('align 方位只能是 min / center / max（收到 ' + md + '）');
  }
  if (size[0] <= 0 && size[1] <= 0 && size[2] <= 0) return meshClone(m);
  return meshTransform(m, m4translate(t));
}

// 统一"外向化"：闭合网格若整体朝内（有向体积 < 0）则翻转全部面。
// 生成器（挤出/旋转体/基本体）不必各自纠结绕向，这里一次性归一 —— 且是**可观测**的。
function meshOrientOutward(m) {
  var t = meshTopology(m);
  if (t.boundaryEdges === 0 && t.nonManifoldEdges === 0) {
    if (meshVolume(t.mesh) < 0) {
      var out = meshNew(t.mesh.positions, []);
      for (var k = 0; k < t.mesh.indices.length; k++) {
        var f = t.mesh.indices[k];
        out.indices.push([f[0], f[2], f[1]]);
      }
      return out;
    }
    return t.mesh;
  }
  // 非闭合：不翻转（体积判据无意义），但仍返回焊接后的网格
  return t.mesh;
}

function clampInt(v, lo, hi, what) {
  var n = Math.round(toNumber(v, what));
  if (n < lo) n = lo;
  if (n > hi) n = hi;
  return n;
}

// ── 2D 轮廓（extrude / revolve 的输入）────────────────────────
// 归一化为：逆时针、无相邻重复点的点集 [[x,y],...]（y 对 revolve 而言是 z 轴坐标）
function polyArea(pts) {
  var a = 0;
  for (var i = 0; i < pts.length; i++) {
    var p = pts[i], q = pts[(i + 1) % pts.length];
    a += p[0] * q[1] - q[0] * p[1];
  }
  return a / 2;
}

// 2D 轮廓的可选中心偏移（默认原点）：带孔挤出的偏心孔靠它定位
function profileCenter(spec) {
  if (spec.center === undefined || spec.center === null) return [0, 0];
  if (!isArr(spec.center) || spec.center.length < 2) {
    fail('轮廓的 center 必须是 [x,y]（收到 ' + JSON.stringify(spec.center) + '）');
  }
  return [toNumber(spec.center[0], 'center'), toNumber(spec.center[1], 'center')];
}

function profilePoints(spec, what) {
  var raw = [], k;
  if (isArr(spec)) {
    for (k = 0; k < spec.length; k++) {
      var p = spec[k];
      if (!isArr(p) || p.length < 2) fail((what || 'profile') + ' 的第 ' + k + ' 个点必须是 [x,y]');
      raw.push([toNumber(p[0]), toNumber(p[1])]);
    }
  } else if (isObj(spec)) {
    var type = spec.type || 'polygon';
    var seg;
    if (type === 'rect' || type === 'rectangle' || type === 'square') {
      // 2D 轮廓的 size 是 [w,h]（不是三维 [x,y,z]）
      var s2 = spec.size === undefined ? [2, 2]
        : (isNum(spec.size) ? [spec.size, spec.size]
          : (isArr(spec.size) && spec.size.length >= 2
            ? [toNumber(spec.size[0]), toNumber(spec.size[1])]
            : fail('rect 轮廓的 size 必须是数字或 [w,h]')));
      var w = Math.abs(s2[0]) / 2, h = Math.abs(s2[1]) / 2;
      if (!(w > 0) || !(h > 0)) fail('rect 轮廓的 size 两个分量都必须大于 0');
      var c0 = profileCenter(spec);
      raw = [[c0[0] - w, c0[1] - h], [c0[0] + w, c0[1] - h], [c0[0] + w, c0[1] + h], [c0[0] - w, c0[1] + h]];
    } else if (type === 'circle') {
      var r = toNumber(spec.radius, 'radius');
      if (!(r > 0)) fail('circle 的 radius 必须大于 0');
      seg = clampInt(spec.segments === undefined ? 32 : spec.segments, 3, LIMIT_SEGMENTS, 'segments');
      for (k = 0; k < seg; k++) {
        var th = 2 * Math.PI * k / seg;
        raw.push([r * Math.cos(th), r * Math.sin(th)]);
      }
    } else if (type === 'star') {
      var verts = clampInt(spec.vertices === undefined ? 5 : spec.vertices, 3, 64, 'vertices');
      var ro = toNumber(spec.radius, 'radius');
      var ri = toNumber(spec.innerRadius === undefined ? ro / 2 : spec.innerRadius, 'innerRadius');
      if (!(ro > 0) || !(ri > 0)) fail('star 的 radius / innerRadius 必须大于 0');
      for (k = 0; k < verts * 2; k++) {
        var rr = k % 2 === 0 ? ro : ri;
        var a2 = Math.PI * k / verts - Math.PI / 2;
        raw.push([rr * Math.cos(a2), rr * Math.sin(a2)]);
      }
    } else if (type === 'polygon' || type === 'points') {
      if (!isArr(spec.points)) fail('polygon 轮廓需要 points 数组');
      for (k = 0; k < spec.points.length; k++) {
        var q = spec.points[k];
        if (!isArr(q) || q.length < 2) fail('polygon.points[' + k + '] 必须是 [x,y]');
        raw.push([toNumber(q[0]), toNumber(q[1])]);
      }
    } else {
      fail('未知轮廓类型 ' + type + '（可用 rect / circle / star / polygon）');
    }
    // 可选中心偏移（circle/star/polygon；rect 上面已就地应用）：带孔挤出的偏心孔靠它定位
    if (!(type === 'rect' || type === 'rectangle' || type === 'square')) {
      var cOff = profileCenter(spec);
      for (k = 0; k < raw.length; k++) { raw[k][0] += cOff[0]; raw[k][1] += cOff[1]; }
    }
  } else {
    fail((what || 'profile') + '必须是点集 [[x,y],...] 或 {type,...}');
  }
  // 去相邻重复点（含首尾环回）
  var pts = [];
  for (k = 0; k < raw.length; k++) {
    var prev = pts.length ? pts[pts.length - 1] : null;
    if (prev && Math.abs(prev[0] - raw[k][0]) <= EPS && Math.abs(prev[1] - raw[k][1]) <= EPS) continue;
    pts.push(raw[k]);
  }
  while (pts.length > 1) {
    var f0 = pts[0], l0 = pts[pts.length - 1];
    if (Math.abs(f0[0] - l0[0]) <= EPS && Math.abs(f0[1] - l0[1]) <= EPS) pts.pop();
    else break;
  }
  if (pts.length < 3) fail((what || 'profile') + ' 至少需要 3 个不同顶点（当前 ' + pts.length + '）');
  if (polyArea(pts) < 0) pts.reverse(); // 统一逆时针
  return pts;
}

// 简单多边形耳切三角化（earcut 风格：顺序切耳 + 共线点清理 + 兜底；O(n²)，轮廓顶点数很小，够用且无依赖）
// 带孔轮廓会先经「桥接」变成带零宽通道的环（含重复顶点），顺序切耳 + 含边界遮挡判定对它同样成立。
// 轮廓三角化（无洞）：earcut 版；返回 [[i,j,k],...]（CCW）。
// ★ 两条路径分工（实测决定，勿随手合并）：
//   · 挤出 / 扭转挤出 / 扫掠 / 回转 的**端盖** → 用 earcut 版（本函数）。
//     收益：切不出反向三角形（旧实现实测 19/70 个负面积）；带孔挤出的 M4 从
//     「208 条真缺口」直接变成「全部部件严格水密」。
//   · I.6-1 / I.6-2 的**共面轮廓收尾** → 用 triangulatePolygonWithHolesLegacy（旧桥接+耳切）。
//     原因：布尔路径的面数/M4 对三角化切法很敏感 —— earcut 会 splitEarcut 分割自接触环，
//     产出的三角形邻接更碎，导致下一轮 meshToPolys 的共面合并变差：
//     实测 4 次串行 subtract 的 polyNodes 388→677、polyInput 1913→2502、
//     面数 2216→2797、M4 真缺口 893→1255（T 缝 221→313）。故布尔路径维持旧实现**零回归**。
// 两个 Legacy 函数因此**不是**死代码，而是布尔路径的专用实现。
function triangulatePolygon(pts) {
  var flat = ecFlatten(pts);
  return ecFinalize(flat, null, ecTriangulate(flat, null), '轮廓');
}
// 旧实现（桥接 + 耳切 + 取最大凸角强切）：仅供 I.6-1/I.6-2 的共面轮廓收尾使用。
function triangulatePolygonLegacy(pts) {
  var n = pts.length;
  var idx = [], k;
  for (k = 0; k < n; k++) idx.push(k);
  var tris = [], guard = 0;
  function cross2(o, a, b) {
    return (pts[a][0] - pts[o][0]) * (pts[b][1] - pts[o][1]) - (pts[a][1] - pts[o][1]) * (pts[b][0] - pts[o][0]);
  }
  // 在三角形内（**含边界**，同 earcut 的 pointInTriangle）：边界/共线点也算「挡住」，
  // 这样横跨桥接通道（零宽）的耳会被通道上的重复顶点挡住，不会切穿。
  function withinTri(a, b, c, p) {
    var d1 = (pts[b][0] - pts[a][0]) * (p[1] - pts[a][1]) - (pts[b][1] - pts[a][1]) * (p[0] - pts[a][0]);
    var d2 = (pts[c][0] - pts[b][0]) * (p[1] - pts[b][1]) - (pts[c][1] - pts[b][1]) * (p[0] - pts[b][0]);
    var d3 = (pts[a][0] - pts[c][0]) * (p[1] - pts[c][1]) - (pts[a][1] - pts[c][1]) * (p[0] - pts[c][0]);
    var neg = (d1 < -EPS) || (d2 < -EPS) || (d3 < -EPS);
    var pos = (d1 > EPS) || (d2 > EPS) || (d3 > EPS);
    return !(neg && pos);
  }
  // 剔除共线/重合的中间顶点（earcut 的 filterPoints）：不改变覆盖面积，可解开「一轮找不到耳」的死结
  function prune() {
    var removed = false, q;
    for (q = 0; q < idx.length && idx.length > 3; q++) {
      var pa = idx[(q + idx.length - 1) % idx.length], pb = idx[q], pc = idx[(q + 1) % idx.length];
      var dup = (pts[pa][0] === pts[pb][0] && pts[pa][1] === pts[pb][1]) ||
                (pts[pb][0] === pts[pc][0] && pts[pb][1] === pts[pc][1]);
      if (dup || Math.abs(cross2(pa, pb, pc)) <= EPS) { idx.splice(q, 1); removed = true; q--; }
    }
    return removed;
  }
  // 对角线 (a→c) 是否与环的某条边**真交叉**（共享端点不算）。自接触环（桥接通道使环在某些
  // 顶点处自我接触）上，仅凭"三角形内无凹顶点"不足以判耳：对角线可能切穿通道或跨到环的另
  // 一段，从而切出**重叠**三角形（体积仍对、拓扑坏）。earcut 用 cureLocalIntersections 兜底，
  // 这里改用"对角线可见性"从判定源头拦住。
  function diagClear(a, b, c) {
    var A = pts[a], C = pts[c], i2;
    for (i2 = 0; i2 < idx.length; i2++) {
      var t1 = idx[i2], t2 = idx[(i2 + 1) % idx.length];
      if (t1 === a || t1 === c || t2 === a || t2 === c || t1 === b || t2 === b) continue;
      if (segCross(A, C, pts[t1], pts[t2])) return false;
    }
    return true;
  }
  var i = 0, misses = 0, m, a, b, c, p;
  while (idx.length > 3) {
    if (++guard > n * n + 64) fail('轮廓三角化失败（顶点过多或形状自交）');
    a = idx[(i + idx.length - 1) % idx.length];
    b = idx[i];
    c = idx[(i + 1) % idx.length];
    var okEar = cross2(a, b, c) > EPS;      // 凸角才可能成耳（共线/凹角跳过）
    if (okEar) {
      for (m = 0; m < idx.length; m++) {
        if (m === i) continue;
        p = idx[m];
        if (p === a || p === b || p === c) continue;
        // 只有**凹**顶点可能挡住耳（凸顶点落在耳内不影响三角化）—— 对齐 earcut 的 isEar
        if (cross2(idx[(m + idx.length - 1) % idx.length], p, idx[(m + 1) % idx.length]) > EPS) continue;
        if (withinTri(a, b, c, pts[p])) { okEar = false; break; }
      }
    }
    if (okEar && !diagClear(a, b, c)) okEar = false;
    if (okEar) {
      tris.push([a, b, c]);
      idx.splice(i, 1);
      if (i >= idx.length) i = 0;
      misses = 0;
      continue;
    }
    i = (i + 1) % idx.length;
    if (++misses > idx.length) {
      // 一轮都没有耳：先清理共线/重合点重试；仍不行则取最大凸角强切（自交/近退化轮廓兜底）
      if (!prune()) {
        var cut = -1, bestAr = -Infinity, qq;
        for (qq = 0; qq < idx.length; qq++) {
          var a2 = idx[(qq + idx.length - 1) % idx.length], b2 = idx[qq], c2 = idx[(qq + 1) % idx.length];
          var ar = cross2(a2, b2, c2);
          if (ar > bestAr) { bestAr = ar; cut = qq; }
        }
        if (cut < 0) fail('轮廓三角化失败：多边形可能自交或全部顶点共线');
        tris.push([idx[(cut + idx.length - 1) % idx.length], idx[cut], idx[(cut + 1) % idx.length]]);
        idx.splice(cut, 1);
      }
      i = 0;
      misses = 0;
    }
  }
  tris.push([idx[0], idx[1], idx[2]]);
  var out = [];
  for (k = 0; k < tris.length; k++) {
    var t = tris[k];
    if (t[0] === t[1] || t[1] === t[2] || t[0] === t[2]) continue;          // 桥接造成的重复顶点
    if (Math.abs(cross2(t[0], t[1], t[2])) <= 2 * EPS) continue;            // 退化（零面积）三角形
    out.push(t);
  }
  return out;
}

// ══ 带洞轮廓三角化：mapbox/earcut 核心算法的等价移植（零依赖 / ES5 / ec 前缀避免命名冲突）══
// 为什么必须换掉旧实现（「桥接 + 对角线可见性硬拦 + 取最大凸角强切」）：
//   旧实现在**自接触环**（多个桥接通道彼此交叉，3 个以上洞必然出现）上会「一轮找不到耳」，
//   而它的兜底是「取最大凸角强切」—— 强切会切穿零宽通道、产出**反向（负面积）**三角形。
//   实测 40×30 板 + 2 圆洞 + 1 方洞（test/_temp/model-holes-e2e.cjs）：
//     旧实现切出 19 个负面积三角形（占 70 个的 27%），报错停在「第 51 个底盖三角形」；
//     桥接退化到「最近可见顶点」时通道横穿整个形状（(3,12)→(-20,-15)），是卡死的直接原因。
//   earcut 的兜底是**三级递进、逐级放宽**：
//     filterPoints（删重复/共线点） → cureLocalIntersections（局部自交就地切成三角形）
//     → splitEarcut（用有效对角线把环一分为二后递归）—— 这才是在自接触输入上稳的原因。
//   旧实现把这些兜底换成了「对角线可见性从源头拦住」，拦得过狠 ⇒ 卡死 ⇒ 强切 ⇒ 坏网格。
// 坐标约定（与全模块一致，y 向上）：外轮廓 CCW（有向面积 > 0）、洞 CW（< 0）⇒ 输出三角形 CCW。
//   earcut 的 area()/signedArea()/pointInTriangle() 只依赖「外环有向面积为正」这一**数学**约定
//   （与屏幕 y 轴朝向无关），与上面的输入约定同构，故此处**逐字照搬、不做任何符号翻转**。
// 省略的部分：z-order 曲线哈希（invSize 不启用 ⇒ 只走朴素 isEar 分支；数据量小、无收益）。
var EC_BUDGET = 200000;      // 三角化步数预算（防退化输入把递归拖死）
var ecSteps = 0;
function ecArea(p, q, r) {
  return (q.y - p.y) * (r.x - q.x) - (q.x - p.x) * (r.y - q.y);
}
function ecSign(v) { return v > 0 ? 1 : (v < 0 ? -1 : 0); }
function ecEquals(p1, p2) { return p1.x === p2.x && p1.y === p2.y; }
function ecOnSegment(p, q, r) {
  return q.x <= Math.max(p.x, r.x) && q.x >= Math.min(p.x, r.x) &&
         q.y <= Math.max(p.y, r.y) && q.y >= Math.min(p.y, r.y);
}
function ecIntersects(p1, q1, p2, q2) {
  var o1 = ecSign(ecArea(p1, q1, p2)), o2 = ecSign(ecArea(p1, q1, q2)),
      o3 = ecSign(ecArea(p2, q2, p1)), o4 = ecSign(ecArea(p2, q2, q1));
  if (o1 !== o2 && o3 !== o4) return true;
  if (o1 === 0 && ecOnSegment(p1, p2, q1)) return true;
  if (o2 === 0 && ecOnSegment(p1, q2, q1)) return true;
  if (o3 === 0 && ecOnSegment(p2, p1, q2)) return true;
  if (o4 === 0 && ecOnSegment(p2, q1, q2)) return true;
  return false;
}
function ecIntersectsPolygon(a, b) {
  var p = a;
  do {
    if (p.i !== a.i && p.next.i !== a.i && p.i !== b.i && p.next.i !== b.i &&
        ecIntersects(p, p.next, a, b)) return true;
    p = p.next;
  } while (p !== a);
  return false;
}
function ecLocallyInside(a, b) {
  return ecArea(a.prev, a, a.next) < 0 ?
    (ecArea(a, b, a.next) >= 0 && ecArea(a, a.prev, b) >= 0) :
    (ecArea(a, b, a.prev) < 0 || ecArea(a, a.next, b) < 0);
}
function ecMiddleInside(a, b) {
  var p = a, inside = false, px = (a.x + b.x) / 2, py = (a.y + b.y) / 2;
  do {
    if (((p.y > py) !== (p.next.y > py)) && p.next.y !== p.y &&
        (px < (p.next.x - p.x) * (py - p.y) / (p.next.y - p.y) + p.x)) inside = !inside;
    p = p.next;
  } while (p !== a);
  return inside;
}
function ecSectorContainsSector(m, p) {
  return ecArea(m.prev, m, p.prev) < 0 && ecArea(p.next, m, m.next) < 0;
}
function ecPointInTriangle(ax, ay, bx, by, cx, cy, px, py) {
  return (cx - px) * (ay - py) >= (ax - px) * (cy - py) &&
         (ax - px) * (by - py) >= (bx - px) * (ay - py) &&
         (bx - px) * (cy - py) >= (cx - px) * (by - py);
}
function ecIsValidDiagonal(a, b) {
  return a.next.i !== b.i && a.prev.i !== b.i && !ecIntersectsPolygon(a, b) &&
    ((ecLocallyInside(a, b) && ecLocallyInside(b, a) && ecMiddleInside(a, b) &&
      (ecArea(a.prev, a, b.prev) || ecArea(a, b.prev, b))) ||
     (ecEquals(a, b) && ecArea(a.prev, a, a.next) > 0 && ecArea(b.prev, b, b.next) > 0));
}
function ecSignedAreaFlat(data, start, end, dim) {
  var sum = 0, i, j;
  for (i = start, j = end - dim; i < end; i += dim) {
    sum += (data[j] - data[i]) * (data[i + 1] + data[j + 1]);
    j = i;
  }
  return sum;
}
function ecCreateNode(i, x, y) {
  return { i: i, x: x, y: y, prev: null, next: null, steiner: false };
}
function ecInsertNode(i, x, y, last) {
  var p = ecCreateNode(i, x, y);
  if (!last) { p.prev = p; p.next = p; }
  else { p.next = last.next; p.prev = last; last.next.prev = p; last.next = p; }
  return p;
}
function ecRemoveNode(p) {
  p.next.prev = p.prev;
  p.prev.next = p.next;
}
function ecLinkedList(data, start, end, dim, clockwise) {
  var last = null, i;
  if (clockwise === (ecSignedAreaFlat(data, start, end, dim) > 0)) {
    for (i = start; i < end; i += dim) last = ecInsertNode(i / dim | 0, data[i], data[i + 1], last);
  } else {
    for (i = end - dim; i >= start; i -= dim) last = ecInsertNode(i / dim | 0, data[i], data[i + 1], last);
  }
  if (last && ecEquals(last, last.next)) { ecRemoveNode(last); last = last.next; }
  return last;
}
function ecFilterPoints(start, end) {
  if (!start) return start;
  if (!end) end = start;
  var p = start, again;
  do {
    again = false;
    // ★ 只删**重复点**，**不删共线点**（earcut 原版会连共线点一起删）。
    //   原因：轮廓上的共线中间点虽然对"面积"无贡献，却是**侧壁的顶点**——
    //   删掉它会让底/顶盖的边界少一条边，与侧壁对不上（实测「板+2圆孔+1方洞」
    //   boundary=6、χ=-6，model-holes-test / model-twist-test 各挂 1 项）。
    //   代价：兜底时少了"删点解卡"这一手，但 cureLocalIntersections + splitEarcut 仍在。
    if (!p.steiner && ecEquals(p, p.next)) {
      ecRemoveNode(p);
      p = end = p.prev;
      if (p === p.next) break;
      again = true;
    } else p = p.next;
  } while (again || p !== end);
  return end;
}
function ecLeftmost(start) {
  var p = start, leftmost = start;
  do {
    if (p.x < leftmost.x || (p.x === leftmost.x && p.y < leftmost.y)) leftmost = p;
    p = p.next;
  } while (p !== start);
  return leftmost;
}
function ecCompareXYSlope(a, b) {
  var result = a.x - b.x, aSlope, bSlope;
  if (result === 0) {
    result = a.y - b.y;
    if (result === 0) {
      aSlope = (a.next.y - a.y) / (a.next.x - a.x);
      bSlope = (b.next.y - b.y) / (b.next.x - b.x);
      result = aSlope - bSlope;
    }
  }
  return result;
}
function ecIsEar(ear) {
  var a = ear.prev, b = ear, c = ear.next;
  if (ecArea(a, b, c) >= 0) return false;          // 凹角/共线 ⇒ 不可能是耳
  var ax = a.x, ay = a.y, bx = b.x, by = b.y, cx = c.x, cy = c.y;
  var x0 = Math.min(ax, bx, cx), y0 = Math.min(ay, by, cy),
      x1 = Math.max(ax, bx, cx), y1 = Math.max(ay, by, cy);
  var p = c.next;
  while (p !== a) {
    // 只有**凹**顶点落进耳里才挡住（凸顶点落在耳内不影响三角化），且先用包围盒预筛
    if (p.x >= x0 && p.x <= x1 && p.y >= y0 && p.y <= y1 &&
        ecPointInTriangle(ax, ay, bx, by, cx, cy, p.x, p.y) &&
        ecArea(p.prev, p, p.next) >= 0) return false;
    p = p.next;
  }
  return true;
}
// 兜底 2：局部自交就地修 —— 若 a→p→p.next→b 这段自我交叉且两侧局部可达，
// 就把 a/p/b 切一个三角形、删掉 p 与其后继，把交叉「消掉」而不是硬切。
function ecCureLocalIntersections(start, tris) {
  var p = start, a, b;
  do {
    a = p.prev; b = p.next.next;
    if (!ecEquals(a, b) && ecIntersects(a, p, p.next, b) &&
        ecLocallyInside(a, b) && ecLocallyInside(b, a)) {
      tris.push([a.i, p.i, b.i]);
      ecRemoveNode(p);
      ecRemoveNode(p.next);
      p = start = b;
    }
    p = p.next;
  } while (p !== start);
  return ecFilterPoints(p);
}
function ecSplitPolygon(a, b) {
  var a2 = ecCreateNode(a.i, a.x, a.y), b2 = ecCreateNode(b.i, b.x, b.y),
      an = a.next, bp = b.prev;
  a.next = b; b.prev = a;
  a2.next = an; an.prev = a2;
  b2.next = a2; a2.prev = b2;
  bp.next = b2; b2.prev = bp;
  return b2;
}
// 兜底 3：用一条「有效对角线」把环一分为二，两个子环各自递归三角化。
function ecSplitEarcut(start, tris) {
  var a = start, b, c;
  do {
    b = a.next.next;
    while (b !== a.prev) {
      if (a.i !== b.i && ecIsValidDiagonal(a, b)) {
        c = ecSplitPolygon(a, b);
        a = ecFilterPoints(a, a.next);
        c = ecFilterPoints(c, c.next);
        ecEarcutLinked(a, tris, 0);
        ecEarcutLinked(c, tris, 0);
        return;
      }
      b = b.next;
    }
    a = a.next;
  } while (a !== start);
}
function ecEarcutLinked(ear, tris, pass) {
  if (!ear) return;
  if (++ecSteps > EC_BUDGET) fail('轮廓三角化超出步数预算（轮廓过于退化或自交）');
  var stop = ear, prev, next;
  while (ear.prev !== ear.next) {
    prev = ear.prev; next = ear.next;
    if (ecIsEar(ear)) {
      tris.push([prev.i, ear.i, next.i]);
      ecRemoveNode(ear);
      ear = next.next;
      stop = next.next;
      continue;
    }
    ear = next;
    if (ear === stop) {                 // 一轮都没找到耳 ⇒ 三级兜底
      if (!pass) ecEarcutLinked(ecFilterPoints(ear), tris, 1);
      else if (pass === 1) ecEarcutLinked(ecCureLocalIntersections(ecFilterPoints(ear), tris), tris, 2);
      else if (pass === 2) ecSplitEarcut(ear, tris);
      break;
    }
  }
}
function ecFindHoleBridge(hole, outerNode) {
  var p = outerNode, hx = hole.x, hy = hole.y, qx = -Infinity, m = null, x;
  if (ecEquals(hole, p)) return p;
  do {
    if (ecEquals(hole, p.next)) return p.next;
    else if (hy <= p.y && hy >= p.next.y && p.next.y !== p.y) {
      x = p.x + (hy - p.y) * (p.next.x - p.x) / (p.next.y - p.y);
      if (x <= hx && x > qx) {
        qx = x;
        m = p.x < p.next.x ? p : p.next;
        if (x === hx) return m;         // 洞口正好触到外环边 ⇒ 取左端点
      }
    }
    p = p.next;
  } while (p !== outerNode);
  if (!m) return null;
  // 在「洞点 / 交点 / 端点」构成的三角形里找点：若有点落在里面，改取「射线切角最小」的点
  var stop = m, mx = m.x, my = m.y, tanMin = Infinity, tan;
  p = m;
  do {
    if (hx >= p.x && p.x >= mx && hx !== p.x &&
        ecPointInTriangle(hy < my ? hx : qx, hy, mx, my, hy < my ? qx : hx, hy, p.x, p.y)) {
      tan = Math.abs(hy - p.y) / (hx - p.x);
      if (ecLocallyInside(p, hole) &&
          (tan < tanMin || (tan === tanMin && (p.x > m.x || (p.x === m.x && ecSectorContainsSector(m, p)))))) {
        m = p; tanMin = tan;
      }
    }
    p = p.next;
  } while (p !== stop);
  return m;
}
function ecEliminateHole(hole, outerNode) {
  var bridge = ecFindHoleBridge(hole, outerNode);
  if (!bridge) return outerNode;
  var bridgeReverse = ecSplitPolygon(bridge, hole);
  ecFilterPoints(bridgeReverse, bridgeReverse.next);
  return ecFilterPoints(bridge, bridge.next);
}
function ecEliminateHoles(data, holeIndices, outerNode, dim) {
  var queue = [], i, len = holeIndices.length, start, end, list;
  for (i = 0; i < len; i++) {
    start = holeIndices[i] * dim;
    end = i < len - 1 ? holeIndices[i + 1] * dim : data.length;
    list = ecLinkedList(data, start, end, dim, false);
    if (list === list.next) list.steiner = true;
    queue.push(ecLeftmost(list));      // 按洞的**最左**点排序（配合 findHoleBridge 的向左射线）
  }
  queue.sort(ecCompareXYSlope);
  for (i = 0; i < queue.length; i++) outerNode = ecEliminateHole(queue[i], outerNode);
  return outerNode;
}
function ecTriangulate(data, holeIndices) {
  var hasHoles = holeIndices && holeIndices.length;
  var outerLen = hasHoles ? holeIndices[0] * 2 : data.length;
  var outerNode = ecLinkedList(data, 0, outerLen, 2, true);
  var tris = [];
  ecSteps = 0;
  if (!outerNode || outerNode.next === outerNode.prev) return tris;
  if (hasHoles) outerNode = ecEliminateHoles(data, holeIndices, outerNode, 2);
  ecEarcutLinked(outerNode, tris, 0);
  return tris;
}
// 三角化结果收尾：丢退化三角形、统一为 CCW、**校验覆盖面积**（绝不静默产出坏网格）。
// 期望面积 = 整条扁平点表的 signedArea（外轮廓 CCW 为正、各洞 CW 为负 ⇒ 相加正好是净面积）；
// 桥接通道正反各走一次、面积自相抵消，故对自接触环该式同样成立。
function ecTriArea2(flat, t) {
  var ax = flat[t[0] * 2], ay = flat[t[0] * 2 + 1],
      bx = flat[t[1] * 2], by = flat[t[1] * 2 + 1],
      cx = flat[t[2] * 2], cy = flat[t[2] * 2 + 1];
  return (bx - ax) * (cy - ay) - (by - ay) * (cx - ax);
}
function ecFinalize(flat, holeIndices, raw, what) {
  var kept = [], got2 = 0, k, t, q, a2, tol, src, s, e, nh = holeIndices ? holeIndices.length : 0;
  for (k = 0; k < raw.length; k++) {           // raw 是 [[i,j,k],...]（earcutLinked 逐面 push）
    src = raw[k];
    t = [src[0], src[1], src[2]];
    if (t[0] === t[1] || t[1] === t[2] || t[0] === t[2]) continue;
    a2 = ecTriArea2(flat, t);
    if (a2 === 0) continue;
    kept.push(t);
    got2 += a2;
  }
  // 期望面积必须**按环分别累加**：外轮廓 CCW 为正、各洞 CW 为负 ⇒ 相加即净面积。
  // （不能对整个点表一次性求 signedArea —— 那会把各环之间的「虚拟连线」也算进去。）
  var want2 = ecSignedAreaFlat(flat, 0, nh ? holeIndices[0] * 2 : flat.length, 2);
  for (k = 0; k < nh; k++) {
    s = holeIndices[k] * 2;
    e = k < nh - 1 ? holeIndices[k + 1] * 2 : flat.length;
    want2 += ecSignedAreaFlat(flat, s, e, 2);
  }
  tol = Math.max(1e-8, Math.abs(want2) * 1e-8);
  if (Math.abs(got2 - want2) > tol && Math.abs(-got2 - want2) <= tol) {
    for (k = 0; k < kept.length; k++) { q = kept[k]; kept[k] = [q[0], q[2], q[1]]; }
    got2 = -got2;
  }
  if (Math.abs(got2 - want2) > tol) {
    fail(what + '三角化覆盖面积与轮廓不符（得 ' + (got2 / 2) + '，应为 ' + (want2 / 2) + '）' +
      '：轮廓自交、洞与外轮廓相交，或洞之间相交');
  }
  // ★ 边界完整性校验：输出三角形的**边界边**（找不到反向边的有向边——与 M4 同一判据；
  //   "恰好出现一次"是错的，内部对角线也是各出现一次的一对反向边）
  //   必须与输入各环的边**一一对应**。少了 ⇒ 底/顶盖的边界比侧壁少顶点（earcut 的
  //   filterPoints 删共线点就曾造成这个：实测「板+2圆孔+1方洞」boundary=6、χ=-6，
  //   model-holes-test 与 model-twist-test 各挂 1 项）；多了 ⇒ 三角化切穿了轮廓。
  //   面积守恒抓不住这两类缺陷（重叠与缺失可以互相抵消），故必须单独校验。
  var edgeCnt = {}, nv = flat.length / 2, i2, j2, bs, be, bounds = [], bad = [], ekey, parts, isBnd = {};
  for (k = 0; k < kept.length; k++) {
    t = kept[k];
    edgeCnt[t[0] + '_' + t[1]] = (edgeCnt[t[0] + '_' + t[1]] || 0) + 1;
    edgeCnt[t[1] + '_' + t[2]] = (edgeCnt[t[1] + '_' + t[2]] || 0) + 1;
    edgeCnt[t[2] + '_' + t[0]] = (edgeCnt[t[2] + '_' + t[0]] || 0) + 1;
  }
  for (ekey in edgeCnt) {
    parts = ekey.split('_');
    if (!edgeCnt[parts[1] + '_' + parts[0]]) isBnd[ekey] = 1;
  }
  bounds.push([0, nh ? holeIndices[0] : nv]);
  for (k = 0; k < nh; k++) bounds.push([holeIndices[k], k + 1 < nh ? holeIndices[k + 1] : nv]);
  var wantEdge = {};
  for (k = 0; k < bounds.length; k++) {
    bs = bounds[k][0]; be = bounds[k][1];
    for (i2 = bs; i2 < be; i2++) {
      j2 = i2 + 1 < be ? i2 + 1 : bs;
      wantEdge[i2 + '_' + j2] = 1;
      if (!isBnd[i2 + '_' + j2]) bad.push('缺边 ' + i2 + '→' + j2);
    }
  }
  for (ekey in isBnd) if (!wantEdge[ekey]) bad.push('多边 ' + ekey);
  if (bad.length) {
    fail(what + '三角化的边界与输入轮廓不一致（' + bad.length + ' 处，如 ' + bad.slice(0, 4).join('、') +
      '）：三角化丢边或越界，会直接表现为材料不水密');
  }
  return kept;
}
function ecFlatten(pts) {
  var flat = [], k;
  for (k = 0; k < pts.length; k++) { flat.push(pts[k][0], pts[k][1]); }
  return flat;
}

// ── 带孔轮廓（earcut 的 eliminateHoles 思路：桥接边把洞并入外轮廓 → 复用上面的耳切）──
// 洞轮廓统一取**顺时针**（外轮廓是 CCW）⇒ 洞的侧壁法线自动朝内；桥接产生的退化三角形会被过滤。
function holesPoints(raw) {
  if (raw === undefined || raw === null) return [];
  if (!isArr(raw)) fail('holes 必须是数组：每个元素是一个闭合轮廓 [[x,y],...]（或 {type:"polygon",points:[...]}）');
  var list = [];
  for (var i = 0; i < raw.length; i++) {
    var pts = profilePoints(raw[i], 'holes[' + i + ']');
    if (polyArea(pts) > 0) pts.reverse();
    list.push(pts);
  }
  return list;
}
function pointInPoly(pt, poly) {
  var inside = false;
  for (var i = 0, j = poly.length - 1; i < poly.length; j = i++) {
    var yi = poly[i][1], yj = poly[j][1];
    if (((yi > pt[1]) !== (yj > pt[1])) &&
        (pt[0] < (poly[j][0] - poly[i][0]) * (pt[1] - yi) / (yj - yi) + poly[i][0])) inside = !inside;
  }
  return inside;
}
function segCross(a, b, c, d) {
  function ori(p, q, r) { return (q[0] - p[0]) * (r[1] - p[1]) - (q[1] - p[1]) * (r[0] - p[0]); }
  var d1 = ori(a, b, c), d2 = ori(a, b, d), d3 = ori(c, d, a), d4 = ori(c, d, b);
  return ((d1 > EPS && d2 < -EPS) || (d1 < -EPS && d2 > EPS)) && ((d3 > EPS && d4 < -EPS) || (d3 < -EPS && d4 > EPS));
}
// 洞是否完全落在外轮廓内（顶点全在内 + 两条环互不相交）
function ringInsideRing(inner, outer) {
  var i, j;
  for (i = 0; i < inner.length; i++) if (!pointInPoly(inner[i], outer)) return false;
  for (i = 0; i < inner.length; i++) {
    var i2 = (i + 1) % inner.length;
    for (j = 0; j < outer.length; j++) {
      var j2 = (j + 1) % outer.length;
      if (segCross(inner[i], inner[i2], outer[j], outer[j2])) return false;
    }
  }
  return true;
}
// 带孔轮廓的前置校验（挤出/扭转/扫掠共用）：洞须完全落在外轮廓内，且洞之间互不相交、互不嵌套。
// 「相交/嵌套的洞」会让桥接通道彼此穿越（环自交），底/顶盖三角化必然产生重叠面 ⇒ 提前挡住。
function validateHoles(pts, holes) {
  var hh, h2;
  for (hh = 0; hh < holes.length; hh++) {
    if (!ringInsideRing(holes[hh], pts)) {
      fail('holes[' + hh + '] 必须完全落在外轮廓内部且与外轮廓不相交（不支持洞越界/穿透）');
    }
  }
  for (hh = 0; hh < holes.length; hh++) {
    for (h2 = hh + 1; h2 < holes.length; h2++) {
      if (ringsCross(holes[hh], holes[h2]) || ringInsideRing(holes[hh], holes[h2]) || ringInsideRing(holes[h2], holes[hh])) {
        fail('holes[' + hh + '] 与 holes[' + h2 + '] 相交或嵌套：各洞必须互不相交、互不包含' +
          '（同一块实体的多个独立腔体；需要"环中环"请拆成两个部件或用布尔运算）');
      }
    }
  }
}
// 两环是否相交（只看边交叉；包含关系由 ringInsideRing 判）
function ringsCross(a, b) {
  var i, j;
  for (i = 0; i < a.length; i++) {
    var i2 = (i + 1) % a.length;
    for (j = 0; j < b.length; j++) {
      var j2 = (j + 1) % b.length;
      if (segCross(a[i], a[i2], b[j], b[j2])) return true;
    }
  }
  return false;
}
// 环中每个点的出现次数（外轮廓点 1 次；被桥接过的点 2 次 ⇒ 已被占用）
function ringCounts(ringIdx) {
  var c = {};
  for (var i = 0; i < ringIdx.length; i++) c[ringIdx[i]] = (c[ringIdx[i]] || 0) + 1;
  return c;
}
// 桥接点：洞最右点 M 向 +x 射线取最近交点所在边的端点；不可见则退化为「最近可见顶点」。
// 多个洞可能都想桥接到同一个外轮廓顶点（例：两洞最右点在同一水平线上，射线打到同一交点）——
// 同一顶点被反复桥接会让环出现多次自接触，耳切会切出重叠三角形（体积仍对、拓扑坏）。
// 故按「环中是否已被占用」优先挑未占用的顶点，其次才取更近的射线交点。
function findBridge(ringIdx, all, M, counts) {
  function busy(pos) { return !!(counts && counts[ringIdx[pos]] > 1); }
  function visible(pos) {
    var P = all[ringIdx[pos]];
    for (var i2 = 0; i2 < ringIdx.length; i2++) {
      var i3 = (i2 + 1) % ringIdx.length;
      if (i2 === pos || i3 === pos) continue;      // 与桥接点共享端点的边不算相交
      if (segCross(M, P, all[ringIdx[i2]], all[ringIdx[i3]])) return false;
    }
    return true;
  }
  var cands = [];
  for (var i = 0; i < ringIdx.length; i++) {
    var a = all[ringIdx[i]], b = all[ringIdx[(i + 1) % ringIdx.length]];
    if (Math.abs(a[1] - b[1]) <= EPS) {            // 水平边
      if (Math.abs(M[1] - a[1]) <= EPS && Math.max(a[0], b[0]) >= M[0] - EPS) {
        cands.push({ pos: (a[0] >= b[0]) ? i : (i + 1) % ringIdx.length, x: Math.min(a[0], b[0]) });
      }
      continue;
    }
    var lo = Math.min(a[1], b[1]), hi = Math.max(a[1], b[1]);
    if (M[1] < lo - EPS || M[1] > hi + EPS) continue;
    var t = (M[1] - a[1]) / (b[1] - a[1]);
    var x = a[0] + t * (b[0] - a[0]);
    if (x < M[0] - EPS) continue;                  // 交点在 M 左侧 → 不是右侧最近交点
    cands.push({ pos: (a[0] >= b[0]) ? i : (i + 1) % ringIdx.length, x: x });
  }
  cands.sort(function (u, v) {
    return (busy(u.pos) ? 1 : 0) - (busy(v.pos) ? 1 : 0) || u.x - v.x;
  });
  for (var z = 0; z < cands.length; z++) if (visible(cands[z].pos)) return cands[z].pos;
  var order = [];
  for (var q = 0; q < ringIdx.length; q++) order.push(q);
  order.sort(function (u, v) {
    return (busy(u) ? 1 : 0) - (busy(v) ? 1 : 0) || vDist(M, all[ringIdx[u]]) - vDist(M, all[ringIdx[v]]);
  });
  for (var z2 = 0; z2 < order.length; z2++) if (visible(order[z2])) return order[z2];
  fail('带孔挤出：洞无法桥接到外轮廓（洞可能在外轮廓之外、自交，或与外轮廓相交）');
  return -1;
}
// 带孔三角化：索引空间 = 外轮廓（0..outer.length-1）+ 依次各洞的点（与挤出时的顶点顺序一致）
// 带洞轮廓三角化：earcut 版（eliminateHoles：洞按最左点排序 + 向左射线桥接）+ 三级兜底耳切。
// 索引空间与调用方一致 = 外轮廓 0..n-1、随后各洞依次。用途见上面 triangulatePolygon 的说明。
function triangulatePolygonWithHoles(outer, holes) {
  var flat = ecFlatten(outer), hi = [], k, j;
  for (k = 0; k < holes.length; k++) {
    hi.push(flat.length / 2);
    for (j = 0; j < holes[k].length; j++) { flat.push(holes[k][j][0], holes[k][j][1]); }
  }
  return ecFinalize(flat, hi, ecTriangulate(flat, hi), '带孔轮廓');
}
// 旧实现（桥接 + 耳切）：仅供 I.6-1/I.6-2 的共面轮廓收尾（布尔路径）使用。
function triangulatePolygonWithHolesLegacy(outer, holes) {
  var all = [], offs = [], k, j;
  offs.push(all.length);
  for (k = 0; k < outer.length; k++) all.push(outer[k]);
  for (k = 0; k < holes.length; k++) {
    offs.push(all.length);
    for (j = 0; j < holes[k].length; j++) all.push(holes[k][j]);
  }
  // 洞的处理顺序 = 「最右点 x」降序：最靠右的洞先桥接到外轮廓，之后各洞只能桥接到已合并的环
  // （含先前洞的顶点）——这样先处理的洞的桥接射线不会横穿尚未处理的洞，减少通道交叉。
  var plan = [];
  for (k = 0; k < holes.length; k++) {
    var base0 = offs[k + 1], hlen0 = holes[k].length, mi0 = 0;
    for (j = 1; j < hlen0; j++) if (all[base0 + j][0] > all[base0 + mi0][0] + EPS) mi0 = j;
    plan.push({ base: base0, hlen: hlen0, mi: mi0, x: all[base0 + mi0][0] });
  }
  plan.sort(function (u, v) { return v.x - u.x; });
  var ringIdx = [];
  for (k = 0; k < outer.length; k++) ringIdx.push(k);
  for (var pi = 0; pi < plan.length; pi++) {
    var base = plan[pi].base, hlen = plan[pi].hlen, mi = plan[pi].mi;
    var bpos = findBridge(ringIdx, all, all[base + mi], ringCounts(ringIdx));
    var merged = [];
    for (j = 0; j <= bpos; j++) merged.push(ringIdx[j]);
    for (j = 0; j <= hlen; j++) merged.push(base + ((mi + j) % hlen));   // 洞循环（起点=终点=M）
    merged.push(ringIdx[bpos]);                                          // 桥接点重复一次 ⇒ 与洞点 M 构成零宽通道
    for (j = bpos + 1; j < ringIdx.length; j++) merged.push(ringIdx[j]);
    ringIdx = merged;
  }
  var pts = [];
  for (k = 0; k < ringIdx.length; k++) pts.push(all[ringIdx[k]]);
  var local = triangulatePolygonLegacy(pts);
  var out = [];
  for (k = 0; k < local.length; k++) {
    var t = local[k];
    var A = ringIdx[t[0]], B = ringIdx[t[1]], C = ringIdx[t[2]];
    if (A === B || B === C || A === C) continue;                        // 桥接造成的退化三角形
    var ar = ((all[B][0] - all[A][0]) * (all[C][1] - all[A][1]) -
              (all[B][1] - all[A][1]) * (all[C][0] - all[A][0])) / 2;
    if (Math.abs(ar) <= EPS) continue;
    // 反向三角形（有向面积为负）= 切穿了自接触环的假耳 ⇒ 覆盖会重叠成对出现、拓扑变坏。
    // 合法输入（洞互不相交、互不嵌套）下不会出现；这里明确报错，绝不静默产出坏网格。
    if (ar < 0) {
      fail('带孔挤出的底/顶盖三角化出现反向三角形：holes 或外轮廓存在自交/嵌套（各洞必须互不相交、互不包含）' +
        '——检查第 ' + k + ' 个底盖三角形 [' + A + ',' + B + ',' + C + '] 对应的轮廓');
    }
    out.push([A, B, C]);
  }
  return out;
}

// ── 基本体（口径对齐 JSCAD primitives：center 默认原点、size 为全尺寸）──
function shapeCuboid(args) {
  var size = toVec3(args.size !== undefined ? args.size : (args.sides !== undefined ? args.sides : 2), 'size', [2, 2, 2]);
  var c = toVec3(args.center, 'center', [0, 0, 0]);
  if (size[0] < 0 || size[1] < 0 || size[2] < 0) fail('cuboid 的 size 不能为负');
  if (!(size[0] > 0) || !(size[1] > 0) || !(size[2] > 0)) fail('cuboid 的 size 必须三个分量都大于 0');
  var pos = [], k;
  for (k = 0; k < 8; k++) {
    pos.push([
      c[0] + (size[0] / 2) * ((k & 1) ? 1 : -1),
      c[1] + (size[1] / 2) * ((k & 2) ? 1 : -1),
      c[2] + (size[2] / 2) * ((k & 4) ? 1 : -1)
    ]);
  }
  var quads = [[0, 4, 6, 2], [1, 3, 7, 5], [0, 1, 5, 4], [2, 6, 7, 3], [0, 2, 3, 1], [4, 5, 7, 6]];
  var idx = [];
  for (k = 0; k < quads.length; k++) {
    var q = quads[k];
    idx.push([q[0], q[1], q[2]]);
    idx.push([q[0], q[2], q[3]]);
  }
  return meshNew(pos, idx);
}

function shapeSphere(args) {
  var r = toNumber(args.radius === undefined ? 1 : args.radius, 'radius');
  if (!(r > 0)) fail('sphere 的 radius 必须大于 0');
  var seg = clampInt(args.segments === undefined ? 32 : args.segments, 3, LIMIT_SEGMENTS, 'segments');
  var rings = clampInt(args.rings === undefined ? Math.max(2, Math.round(seg / 2)) : args.rings, 2, LIMIT_SEGMENTS, 'rings');
  var c = toVec3(args.center, 'center', [0, 0, 0]);
  return sphereMesh(c, [r, r, r], seg, rings);
}

function shapeEllipsoid(args) {
  var rr = toVec3(args.radius === undefined ? 1 : args.radius, 'radius', [1, 1, 1]);
  if (!(rr[0] > 0) || !(rr[1] > 0) || !(rr[2] > 0)) fail('ellipsoid 的 radius 三个分量都必须大于 0');
  var seg = clampInt(args.segments === undefined ? 32 : args.segments, 3, LIMIT_SEGMENTS, 'segments');
  var rings = clampInt(args.rings === undefined ? Math.max(2, Math.round(seg / 2)) : args.rings, 2, LIMIT_SEGMENTS, 'rings');
  var c = toVec3(args.center, 'center', [0, 0, 0]);
  return sphereMesh(c, rr, seg, rings);
}

// 球/椭球：**极点沿 z 轴**（CAD 惯例；JSCAD 的 sphere 沿 y，体积与面数相同，不影响交叉验证）
function sphereMesh(c, rr, seg, rings) {
  var pos = [[c[0], c[1], c[2] + rr[2]]], idx = [], j, i;
  for (i = 1; i < rings; i++) {
    var th = Math.PI * i / rings;
    var sz = Math.cos(th), sr = Math.sin(th);
    for (j = 0; j < seg; j++) {
      var ph = 2 * Math.PI * j / seg;
      pos.push([c[0] + rr[0] * sr * Math.cos(ph), c[1] + rr[1] * sr * Math.sin(ph), c[2] + rr[2] * sz]);
    }
  }
  var south = pos.length;
  pos.push([c[0], c[1], c[2] - rr[2]]);
  for (j = 0; j < seg; j++) {
    idx.push([0, 1 + j, 1 + (j + 1) % seg]); // 北极扇
  }
  for (i = 0; i < rings - 2; i++) {
    var a0 = 1 + i * seg, a1 = 1 + (i + 1) * seg;
    for (j = 0; j < seg; j++) {
      var j2 = (j + 1) % seg;
      idx.push([a0 + j, a1 + j, a1 + j2]);
      idx.push([a0 + j, a1 + j2, a0 + j2]);
    }
  }
  var lastRing = 1 + (rings - 2) * seg;
  for (j = 0; j < seg; j++) {
    idx.push([south, lastRing + (j + 1) % seg, lastRing + j]);
  }
  return meshNew(pos, idx);
}

// 圆柱 / 圆台 / 圆锥（轴沿 z）——一个实现覆盖三种（JSCAD cylinder + cylinderElliptic 的标量特例）
function shapeFrustum(args) {
  var h = toNumber(args.height === undefined ? 2 : args.height, 'height');
  if (!(h > 0)) fail('cylinder/frustum 的 height 必须大于 0');
  var rb = toNumber(args.radiusBottom !== undefined ? args.radiusBottom : (args.radius === undefined ? 1 : args.radius), 'radiusBottom');
  var rtRaw = args.radiusTop !== undefined ? toNumber(args.radiusTop, 'radiusTop')
    : (args.type === 'cone' ? 0 : (args.radius === undefined ? 1 : toNumber(args.radius)));
  if (rb < 0 || rtRaw < 0) fail('半径不能为负');
  if (rb <= 0 && rtRaw <= 0) fail('radiusBottom 与 radiusTop 不能同时为 0');
  var seg = clampInt(args.segments === undefined ? 32 : args.segments, 3, LIMIT_SEGMENTS, 'segments');
  var c = toVec3(args.center, 'center', [0, 0, 0]);
  var zB = c[2] - h / 2, zT = c[2] + h / 2;
  var pos = [], idx = [], j;
  var botBase = -1, topBase = -1, apexTop = -1, apexBot = -1;
  if (rb > EPS) {
    botBase = 0;
    for (j = 0; j < seg; j++) {
      var t0 = 2 * Math.PI * j / seg;
      pos.push([c[0] + rb * Math.cos(t0), c[1] + rb * Math.sin(t0), zB]);
    }
  }
  if (rtRaw > EPS) {
    topBase = pos.length;
    for (j = 0; j < seg; j++) {
      var t1 = 2 * Math.PI * j / seg;
      pos.push([c[0] + rtRaw * Math.cos(t1), c[1] + rtRaw * Math.sin(t1), zT]);
    }
  } else {
    apexTop = pos.length;
    pos.push([c[0], c[1], zT]);
  }
  // 侧面
  for (j = 0; j < seg; j++) {
    var j2 = (j + 1) % seg;
    if (botBase >= 0 && topBase >= 0) {
      idx.push([botBase + j, botBase + j2, topBase + j2]);
      idx.push([botBase + j, topBase + j2, topBase + j]);
    } else if (botBase >= 0) {
      idx.push([botBase + j, botBase + j2, apexTop]);
    } else {
      idx.push([topBase + j2, topBase + j, apexBot]);
    }
  }
  // 底面 / 顶面（各自中心扇形三角化）
  if (botBase >= 0) {
    var cb = pos.length;
    pos.push([c[0], c[1], zB]);
    for (j = 0; j < seg; j++) idx.push([cb, botBase + (j + 1) % seg, botBase + j]);
  } else {
    var cb2 = pos.length;
    pos.push([c[0], c[1], zB]);
    for (j = 0; j < seg; j++) idx.push([cb2, topBase + j, topBase + (j + 1) % seg]);
  }
  if (topBase >= 0) {
    var ct = pos.length;
    pos.push([c[0], c[1], zT]);
    for (j = 0; j < seg; j++) idx.push([ct, topBase + j, topBase + (j + 1) % seg]);
  }
  return meshNew(pos, idx);
}

// 圆环（口径同 JSCAD：innerRadius = 管截面半径，outerRadius = 主半径 R；要求 inner < outer）
function shapeTorus(args) {
  var ri = toNumber(args.innerRadius === undefined ? 1 : args.innerRadius, 'innerRadius');
  var ro = toNumber(args.outerRadius === undefined ? 4 : args.outerRadius, 'outerRadius');
  if (!(ri > 0) || !(ro > 0)) fail('torus 的 innerRadius / outerRadius 必须大于 0');
  if (ri >= ro) fail('torus 的 innerRadius（管半径）必须小于 outerRadius（主半径）');
  var seg = clampInt(args.segments === undefined ? 32 : args.segments, 3, LIMIT_SEGMENTS, 'segments');
  var rings = clampInt(args.rings === undefined ? 32 : args.rings, 3, LIMIT_SEGMENTS, 'rings');
  var c = toVec3(args.center, 'center', [0, 0, 0]);
  var pos = [], idx = [], i, j;
  for (i = 0; i < seg; i++) {
    var u = 2 * Math.PI * i / seg;
    for (j = 0; j < rings; j++) {
      var v = 2 * Math.PI * j / rings;
      var rad = ro + ri * Math.cos(v);
      pos.push([c[0] + rad * Math.cos(u), c[1] + rad * Math.sin(u), c[2] + ri * Math.sin(v)]);
    }
  }
  for (i = 0; i < seg; i++) {
    var i2 = (i + 1) % seg;
    for (j = 0; j < rings; j++) {
      var j2 = (j + 1) % rings;
      var a0 = i * rings + j, a1 = i * rings + j2, b0 = i2 * rings + j, b1 = i2 * rings + j2;
      idx.push([a0, b0, b1]);
      idx.push([a0, b1, a1]);
    }
  }
  return meshNew(pos, idx);
}

function shapePolyhedron(args) {
  if (!isArr(args.points) || args.points.length < 4) fail('polyhedron 需要 points 数组（≥4 个顶点）');
  if (!isArr(args.faces) || !args.faces.length) fail('polyhedron 需要 faces 数组（面 → 顶点下标）');
  var pos = [], k;
  for (k = 0; k < args.points.length; k++) {
    var p = args.points[k];
    if (!isArr(p) || p.length < 3) fail('polyhedron.points[' + k + '] 必须是 [x,y,z]');
    pos.push([toNumber(p[0]), toNumber(p[1]), toNumber(p[2])]);
  }
  var idx = [];
  for (k = 0; k < args.faces.length; k++) {
    var f = args.faces[k];
    if (!isArr(f) || f.length < 3) fail('polyhedron.faces[' + k + '] 至少需要 3 个顶点下标');
    for (var m = 1; m < f.length - 1; m++) {
      var a = Math.round(toNumber(f[0])), b = Math.round(toNumber(f[m])), d = Math.round(toNumber(f[m + 1]));
      if (a < 0 || b < 0 || d < 0 || a >= pos.length || b >= pos.length || d >= pos.length) {
        fail('polyhedron.faces[' + k + '] 越界（顶点下标必须在 0..' + (pos.length - 1) + '）');
      }
      idx.push([a, b, d]);
    }
  }
  return meshNew(pos, idx);
}

// 扭转挤出的网格生成（shapeExtrude 的 twist 分支）：把轮廓沿 z 分 layers 层，
// 第 j 层绕 z 轴按高度**线性**旋转 twist·j/layers 度（口径同 JSCAD extrudeTwist：twist = 总扭转角）。
// 每层都是**刚体旋转** ⇒ 洞随外轮廓同步转、环的拓扑与绕向都不变（holes 的 2D 内嵌校核依然成立）。
// 侧面 = 相邻层同名顶点直连（直纹面），绕向与直线挤出的侧壁完全一致（外侧法线朝外）。
// 顶点表按「层优先」排列（不是直线挤出的「下块 + 上块」），故与 twist=0 的直线挤出路径分开实现、互不影响。
function extrudeTwistMesh(rings, starts, n, cap, h, zc, twistDeg, segArg) {
  var seg = clampInt(segArg === undefined || segArg === null ? TWIST_LAYERS_DEFAULT : segArg, 1, LIMIT_SEGMENTS, 'segments');
  var perLayer = twistDeg / seg;
  if (Math.abs(perLayer) >= 180) {
    fail('扭转挤出的每层扭转角必须小于 180°（twist=' + cleanNum(twistDeg, 4) + '° / segments=' + seg + ' = ' +
      cleanNum(perLayer, 4) + '°/层）：每层转过半圈以上时相邻层顶点会"走近路"反向跨接，侧壁自交。' +
      '请增大 segments（建议 ≥ ' + Math.ceil(Math.abs(twistDeg) / 90) + '）或减小 twist');
  }
  var tris = 2 * seg * n + 2 * cap.length;
  if (tris > LIMIT_TRIS_PART) {
    fail('扭转挤出分段过多（segments=' + seg + ' × 轮廓 ' + n + ' 点 ⇒ 约 ' + tris + ' 面，单部件上限 ' +
      LIMIT_TRIS_PART + '）——请降低 segments 或减少轮廓顶点');
  }
  var rad = twistDeg * Math.PI / 180;
  var pos = [], idx = [], j, k, i;
  for (j = 0; j <= seg; j++) {
    var th = rad * j / seg;
    var ct = Math.cos(th), st = Math.sin(th);
    var z = zc - h / 2 + h * j / seg;
    for (k = 0; k < rings.length; k++) {
      for (i = 0; i < rings[k].length; i++) {
        var p = rings[k][i];
        pos.push([p[0] * ct - p[1] * st, p[0] * st + p[1] * ct, z]);
      }
    }
  }
  function vid(i2, j2) { return j2 * n + i2; }
  // 端盖：与直线挤出一致（底盖反绕、顶盖正绕）；cap 的下标是「外轮廓 + 洞」拼接后的全局下标
  for (k = 0; k < cap.length; k++) {
    var f = cap[k];
    idx.push([vid(f[0], 0), vid(f[2], 0), vid(f[1], 0)]);
    idx.push([vid(f[0], seg), vid(f[1], seg), vid(f[2], seg)]);
  }
  // 侧壁：每层、每种环各出一圈（洞是顺时针环 → 法线朝腔内侧）
  for (j = 0; j < seg; j++) {
    for (k = 0; k < rings.length; k++) {
      var st0 = starts[k], len = rings[k].length;
      for (i = 0; i < len; i++) {
        var a = st0 + i, b = st0 + ((i + 1) % len);
        idx.push([vid(a, j), vid(b, j), vid(b, j + 1)]);
        idx.push([vid(a, j), vid(b, j + 1), vid(a, j + 1)]);
      }
    }
  }
  return meshNew(pos, idx);
}

// 挤出（沿 z，居中；口径同 JSCAD extrudeLinear：height 默认 1）
// twist 非 0 → 改为扭转挤出（见 extrudeTwistMesh）；其余情况与原先逐位一致。
function shapeExtrude(args) {
  var pts = profilePoints(args.profile, 'profile');
  var holes = holesPoints(args.holes);
  var h = toNumber(args.height === undefined ? 1 : args.height, 'height');
  if (!(Math.abs(h) > EPS)) fail('extrude 的 height 不能为 0');
  var zc = args.center === undefined ? 0 : toNumber(args.center, 'center');
  validateHoles(pts, holes);
  var rings = [pts].concat(holes);
  var starts = [], n = 0, k;
  for (k = 0; k < rings.length; k++) { starts.push(n); n += rings[k].length; }
  var cap = holes.length ? triangulatePolygonWithHoles(pts, holes) : triangulatePolygon(pts);
  // twist 非 0 → 扭转挤出（分层旋转）；缺省/为 0 → 走下面的直线挤出路径（逐位不变）
  var twistDeg = args.twist === undefined || args.twist === null ? 0 : toNumber(args.twist, 'twist');
  if (Math.abs(twistDeg) > EPS) return extrudeTwistMesh(rings, starts, n, cap, h, zc, twistDeg, args.segments);
  var pos = [], idx = [], j2;
  for (k = 0; k < rings.length; k++) {
    for (j2 = 0; j2 < rings[k].length; j2++) pos.push([rings[k][j2][0], rings[k][j2][1], zc - h / 2]);
  }
  for (k = 0; k < n; k++) pos.push([pos[k][0], pos[k][1], zc + h / 2]);
  for (k = 0; k < cap.length; k++) {
    var f = cap[k];
    idx.push([f[0], f[2], f[1]]);                 // 底盖（法线朝 -z）
    idx.push([n + f[0], n + f[1], n + f[2]]);     // 顶盖（法线朝 +z）
  }
  // 侧壁：每个闭合环各出一圈（洞是顺时针 → 法线朝洞内，即腔壁）
  for (k = 0; k < rings.length; k++) {
    var st = starts[k], len = rings[k].length;
    for (j2 = 0; j2 < len; j2++) {
      var a = st + j2, b = st + ((j2 + 1) % len);
      idx.push([a, b, n + b]);
      idx.push([a, n + b, n + a]);
    }
  }
  return meshNew(pos, idx);
}

// ── 扫掠（sweep）：2D 轮廓沿 3D 路径扫掠 ────────────────────────────
// 框架传播用「平行传输」（Rodrigues 旋转，等价于 RMF / double-reflection 框架）：
//   不用 Frenet 的 n = t'/|t'| —— 直线段曲率为零、法向无定义，拐点处法向还会突然翻转，
//   扫掠侧壁因此常自交。平行传输对任意路径都连续、无扭转突变。
// 闭合路径额外做 holonomy 修正：把首尾框架的角差按站数均匀摊掉（否则接缝处扭转错位）。
// 两线段最短距离（3D，标准 clamped 最近点算法；路径自交检测用）
function segSegDist(a0, a1, b0, b1) {
  var d1 = vSub(a1, a0), d2 = vSub(b1, b0), r = vSub(a0, b0);
  var a = vDot(d1, d1), e = vDot(d2, d2), f = vDot(d2, r), b = vDot(d1, d2), c = vDot(d1, r);
  var s = 0, t = 0;
  if (a <= EPS && e <= EPS) return vLen(r);
  if (a <= EPS) { t = Math.max(0, Math.min(1, f / e)); }
  else if (e <= EPS) { s = Math.max(0, Math.min(1, -c / a)); }
  else {
    var denom = a * e - b * b;
    s = denom > EPS ? Math.max(0, Math.min(1, (b * f - c * e) / denom)) : 0;
    t = (b * s + f) / e;
    if (t < 0) { t = 0; s = Math.max(0, Math.min(1, -c / a)); }
    else if (t > 1) { t = 1; s = Math.max(0, Math.min(1, (b - c) / a)); }
  }
  return vLen(vSub(vAdd(a0, vMul(d1, s)), vAdd(b0, vMul(d2, t))));
}

// 路径 → 每站框架 {p, t, n, b}（b = t × n ⇒ (n,b,t) 右手系，与挤出用的 (x,y,z) 同构）
function sweepFrames(path, closed, upHint) {
  var n = path.length, i, k;
  var tan = [];
  for (i = 0; i < n; i++) {
    var a, b;
    if (i === 0) a = closed ? vSub(path[0], path[n - 1]) : vSub(path[1], path[0]);
    else a = vSub(path[i], path[i - 1]);
    if (i === n - 1) b = closed ? vSub(path[0], path[n - 1]) : vSub(path[n - 1], path[n - 2]);
    else b = vSub(path[i + 1], path[i]);
    var t = vAdd(vNorm(a), vNorm(b));
    if (vLen(t) <= 1e-9) {
      fail('扫掠路径在第 ' + i + ' 站发生 180° 折返（前后两段方向相反）——请拆分路径，或改用两段扫掠');
    }
    tan.push(vNorm(t));
  }
  var nrm;
  if (upHint) {
    if (Math.abs(vDot(upHint, tan[0])) > 0.999) fail('扫掠的 up 参考方向与路径起始切线几乎平行——请换一个参考方向');
    nrm = vNorm(vSub(upHint, vMul(tan[0], vDot(upHint, tan[0]))));
  } else {
    // 取与起始切线最不平行的坐标轴（保证"沿 +z 直线扫掠"退化为标准挤出：n=+x、b=+y）
    var axes = [[1, 0, 0], [0, 1, 0], [0, 0, 1]], pick = axes[0], best = Infinity;
    for (k = 0; k < 3; k++) {
      var d = Math.abs(vDot(axes[k], tan[0]));
      if (d < best - 1e-12) { best = d; pick = axes[k]; }
    }
    nrm = vNorm(vSub(pick, vMul(tan[0], vDot(pick, tan[0]))));
  }
  var frames = [];
  for (i = 0; i < n; i++) {
    if (i > 0) {
      var ax = vCross(tan[i - 1], tan[i]), s = vLen(ax);
      if (s > 1e-12) {
        nrm = m4applyDir(m4rotAxis(vMul(ax, 1 / s), Math.atan2(s, vDot(tan[i - 1], tan[i])) * 180 / Math.PI), nrm);
      }
      nrm = vNorm(vSub(nrm, vMul(tan[i], vDot(nrm, tan[i]))));   // 消除数值漂移
    }
    frames.push({ p: path[i], t: tan[i], n: nrm, b: vCross(tan[i], nrm) });
  }
  if (closed && n > 2) {
    var n0 = frames[0].n;
    var ax1 = vCross(tan[n - 1], tan[0]), s1 = vLen(ax1), nf = frames[n - 1].n;
    if (s1 > 1e-12) {
      nf = m4applyDir(m4rotAxis(vMul(ax1, 1 / s1), Math.atan2(s1, vDot(tan[n - 1], tan[0])) * 180 / Math.PI), nf);
    }
    nf = vNorm(vSub(nf, vMul(tan[0], vDot(nf, tan[0]))));
    var phi = Math.atan2(vDot(vCross(nf, n0), tan[0]), vDot(nf, n0));
    if (Math.abs(phi) > 1e-12) {
      for (i = 0; i < n; i++) {
        var nn = vNorm(m4applyDir(m4rotAxis(tan[i], phi * i / n * 180 / Math.PI), frames[i].n));
        frames[i].n = nn;
        frames[i].b = vCross(tan[i], nn);
      }
    }
  }
  return frames;
}

// 扫掠自交校验：① 路径非相邻段相交；② 站点曲率半径小于轮廓外接半径（内凹侧侧壁穿透自身）
function sweepSelfCheck(path, closed, rmax) {
  var n = path.length, segCount = closed ? n : n - 1, i, j;
  for (i = 0; i < segCount; i++) {
    for (j = i + 1; j < segCount; j++) {
      if (closed ? (j - i === 1 || (i === 0 && j === segCount - 1)) : (j - i === 1)) continue;
      if (segSegDist(path[i], path[(i + 1) % n], path[j], path[(j + 1) % n]) <= EPS_SPATIAL) {
        fail('扫掠路径自交（第 ' + i + ' 段与第 ' + j + ' 段的最近距离 ≤ ' + EPS_SPATIAL +
          '）——请修改 path，或拆成多段分别扫掠');
      }
    }
  }
  for (i = 0; i < n; i++) {
    var ip = i === 0 ? (closed ? n - 1 : 0) : i - 1;
    var inx = i === n - 1 ? (closed ? 0 : n - 1) : i + 1;
    if (ip === i || inx === i) continue;
    var d1 = vSub(path[i], path[ip]), d2 = vSub(path[inx], path[i]);
    var l1 = vLen(d1), l2 = vLen(d2);
    if (l1 <= EPS || l2 <= EPS) continue;
    var ct = Math.max(-1, Math.min(1, vDot(vMul(d1, 1 / l1), vMul(d2, 1 / l2))));
    var dth = Math.acos(ct);
    if (dth <= 1e-9) continue;
    var rho = Math.min(l1, l2) / dth;
    if (rmax > rho) {
      fail('扫掠自交：路径第 ' + i + ' 站处曲率半径 ≈ ' + cleanNum(rho, 4) + ' 小于轮廓外接半径 ' + cleanNum(rmax, 4) +
        '（内凹侧的侧壁会穿过自身）——请加大该处拐角半径、减小轮廓，或在 path 上加密分站');
    }
  }
}

// 扫掠体：轮廓（含洞）沿 path 扫掠；closed 无端盖，否则两端封盖（首站朝 -t、末站朝 +t）
function shapeSweep(args) {
  var pts = profilePoints(args.profile, 'profile');
  var holes = holesPoints(args.holes);
  validateHoles(pts, holes);
  var path = args.along;   // 字段名 along：path 是工具层的工程文件路径，不能复用
  if (!isArr(path) || path.length < 2) fail('sweep 需要 along 数组（≥2 个 [x,y,z] 点；工具层的 path 是工程文件路径）');
  var closed = args.closed;
  if (closed === undefined || closed === null) closed = vDist(path[0], path[path.length - 1]) <= EPS_SPATIAL;
  closed = !!closed;
  if (closed && vDist(path[0], path[path.length - 1]) <= EPS_SPATIAL) path = path.slice(0, path.length - 1);
  if (path.length < 2) fail('sweep 的 along 去重后不足 2 个不同点');
  var k, j, i;
  var rmax = 0;
  for (k = 0; k < pts.length; k++) rmax = Math.max(rmax, Math.sqrt(pts[k][0] * pts[k][0] + pts[k][1] * pts[k][1]));
  sweepSelfCheck(path, closed, rmax);
  var frames = sweepFrames(path, closed, args.up);
  var twistDeg = args.twist === undefined || args.twist === null ? 0 : toNumber(args.twist, 'twist');
  var S = path.length, segCount = closed ? S : S - 1;
  if (Math.abs(twistDeg) > EPS && Math.abs(twistDeg / segCount) >= 180) {
    fail('扫掠的每段扭转角必须小于 180°（twist=' + cleanNum(twistDeg, 4) + '° / ' + segCount + ' 段 = ' +
      cleanNum(twistDeg / segCount, 4) + '°/段）——请加密 path 或减小 twist');
  }
  var scales = null;
  if (args.scale !== undefined && args.scale !== null) {
    scales = [];
    if (isArr(args.scale)) {
      if (args.scale.length !== S) fail('sweep 的 scale 数组长度必须等于路径站数（' + S + '）');
      for (k = 0; k < S; k++) scales.push(toNumber(args.scale[k], 'scale'));
    } else {
      var sv = toNumber(args.scale, 'scale');
      for (k = 0; k < S; k++) scales.push(sv);
    }
    for (k = 0; k < S; k++) if (!(scales[k] > 0)) fail('sweep 的 scale 必须大于 0');
  }
  var rings = [pts].concat(holes);
  var starts = [], nn = 0;
  for (k = 0; k < rings.length; k++) { starts.push(nn); nn += rings[k].length; }
  var cap = null;
  if (!closed) cap = holes.length ? triangulatePolygonWithHoles(pts, holes) : triangulatePolygon(pts);
  var tris = 2 * segCount * nn + (closed ? 0 : 2 * cap.length);
  if (tris > LIMIT_TRIS_PART) {
    fail('扫掠面数过多（' + segCount + ' 段 × 轮廓 ' + nn + ' 点 ⇒ 约 ' + tris + ' 面，单部件上限 ' + LIMIT_TRIS_PART +
      '）——请减少路径站数或轮廓顶点');
  }
  var pos = [], idx = [];
  for (i = 0; i < S; i++) {
    var fr = frames[i];
    var sc = scales ? scales[i] : 1;
    var ang = (twistDeg * Math.PI / 180) * (closed ? i / S : (S > 1 ? i / (S - 1) : 0));
    var ct = Math.cos(ang), st = Math.sin(ang);
    for (k = 0; k < rings.length; k++) {
      for (j = 0; j < rings[k].length; j++) {
        var pp = rings[k][j];
        var x = (pp[0] * ct - pp[1] * st) * sc, y = (pp[0] * st + pp[1] * ct) * sc;
        pos.push(vAdd(fr.p, vAdd(vMul(fr.n, x), vMul(fr.b, y))));
      }
    }
  }
  function vid(i2, j2) { return i2 * nn + j2; }   // i2 = 站号，j2 = 环内全局下标
  for (k = 0; k < segCount; k++) {
    var k2 = (k + 1) % S;
    for (i = 0; i < rings.length; i++) {
      var st0 = starts[i], len = rings[i].length;
      for (j = 0; j < len; j++) {
        var a = st0 + j, b = st0 + ((j + 1) % len);
        idx.push([vid(k, a), vid(k, b), vid(k2, b)]);
        idx.push([vid(k, a), vid(k2, b), vid(k2, a)]);
      }
    }
  }
  if (!closed && cap) {
    for (k = 0; k < cap.length; k++) {
      var f = cap[k];
      idx.push([vid(0, f[0]), vid(0, f[2]), vid(0, f[1])]);              // 首站盖（法线朝 -t）
      idx.push([vid(S - 1, f[0]), vid(S - 1, f[1]), vid(S - 1, f[2])]);  // 末站盖（法线朝 +t）
    }
  }
  return meshNew(pos, idx);
}

// 旋转体（profile = [[r,z],...] 子午线，绕 z 轴；r 必须 ≥ 0）
function shapeRevolve(args) {
  var prof = profilePoints(args.profile, 'profile');
  var k;
  for (k = 0; k < prof.length; k++) {
    if (prof[k][0] < -EPS) fail('revolve 的 profile 半径 r 不能为负（第 ' + k + ' 点 r=' + prof[k][0] + '）');
  }
  if (polyArea(prof) < 0) prof.reverse(); // profilePoints 已统一，此处于防万一
  var seg = clampInt(args.segments === undefined ? 32 : args.segments, 3, LIMIT_SEGMENTS, 'segments');
  var angle = args.angle === undefined ? 360 : toNumber(args.angle, 'angle');
  if (!(Math.abs(angle) > EPS) || Math.abs(angle) > 360.0000001) fail('revolve 的 angle 必须在 (0,360]');
  var full = Math.abs(angle - 360) < 1e-9;
  var steps = full ? seg : Math.max(1, Math.round(seg * Math.abs(angle) / 360));
  var nP = prof.length, pos = [], idx = [], i, j;
  // 顶点：i（profile 点）× j（角度步；非整圈时共 steps+1 列）
  var cols = full ? steps : steps + 1;
  for (j = 0; j < cols; j++) {
    var th = (full ? 2 * Math.PI * j / steps : (angle * Math.PI / 180) * j / steps);
    var ct = Math.cos(th), st = Math.sin(th);
    for (i = 0; i < nP; i++) pos.push([prof[i][0] * ct, prof[i][0] * st, prof[i][1]]);
  }
  function vid(i2, j2) { return (full ? (j2 % steps) : j2) * nP + i2; }
  // profile 视为**闭合轮廓**（与 extrude 的轮廓同语义）：末点自动连回首点
  for (j = 0; j < steps; j++) {
    for (i = 0; i < nP; i++) {
      var i2 = (i + 1) % nP;
      var a = vid(i, j), b = vid(i2, j), d = vid(i2, j + 1), e = vid(i, j + 1);
      var pa = pos[a], pb = pos[b], pd = pos[d], pe = pos[e];
      if (vDist(pa, pb) > EPS) { idx.push([a, b, d]); idx.push([a, d, e]); }
      else if (vDist(pa, pd) > EPS) { idx.push([a, d, e]); }
    }
  }
  if (!full) {
    // 端盖：把 profile 平面三角化，映射到起始/结束角
    var capTris = triangulatePolygon(prof);
    for (k = 0; k < capTris.length; k++) {
      var t = capTris[k];
      var s0 = t[0], s1 = t[1], s2 = t[2];
      idx.push([vid(s0, 0), vid(s2, 0), vid(s1, 0)]);
      idx.push([vid(s0, steps), vid(s1, steps), vid(s2, steps)]);
    }
  }
  return meshNew(pos, idx);
}

// 导入的三角网格（STL/OBJ 解析结果）
function shapeMesh(args) {
  if (!isArr(args.positions) || !isArr(args.indices)) fail('mesh 形状需要 positions 与 indices');
  var pos = [], idx = [], k;
  for (k = 0; k < args.positions.length; k++) {
    var p = args.positions[k];
    if (!isArr(p) || p.length < 3) fail('mesh.positions[' + k + '] 必须是 [x,y,z]');
    pos.push([toNumber(p[0]), toNumber(p[1]), toNumber(p[2])]);
  }
  for (k = 0; k < args.indices.length; k++) {
    var f = args.indices[k];
    if (!isArr(f) || f.length < 3) fail('mesh.indices[' + k + '] 必须是 [i,j,k]');
    idx.push([Math.round(toNumber(f[0])), Math.round(toNumber(f[1])), Math.round(toNumber(f[2]))]);
  }
  return meshNew(pos, idx);
}

// ── BSP 布尔（经典 csg.js 算法；三角形 → 平面切分 → 树裁剪）──────
// 参考：Evan Wallace csg.js（MIT）的 BSP 布尔算法与 JSCAD @jscad/modeling 的同源实现；
// 本实现为独立重写（三角形输入、迭代式建树/裁剪以避免深递归爆栈）。
var CF_COPLANAR = 0, CF_FRONT = 1, CF_BACK = 2, CF_SPANNING = 3;

function triPlane(t) {
  var n = vNorm(vCross(vSub(t[1], t[0]), vSub(t[2], t[0])));
  if (vLen(n) < 0.5) return null; // 退化三角形
  return { n: n, w: vDot(n, t[0]) };
}
function triFlip(t) { return [t[0], t[2], t[1]]; }

// 凸多边形按平面分裂（csg.js 的 splitPolygon 同语义）
function splitConvexByPlane(plane, verts) {
  var n = verts.length, types = [], i, hasF = false, hasB = false;
  for (i = 0; i < n; i++) {
    var d = vDot(plane.n, verts[i]) - plane.w;
    var ty = d < -EPS_PLANE ? CF_BACK : (d > EPS_PLANE ? CF_FRONT : CF_COPLANAR);
    types.push(ty);
    if (ty === CF_FRONT) hasF = true;
    if (ty === CF_BACK) hasB = true;
  }
  if (!hasF && !hasB) return { coplanar: verts, front: null, back: null };
  if (!hasB) return { coplanar: null, front: verts, back: null };
  if (!hasF) return { coplanar: null, front: null, back: verts };
  var fv = [], bv = [];
  for (i = 0; i < n; i++) {
    var vi = verts[i], vj = verts[(i + 1) % n];
    var ti = types[i], tj = types[(i + 1) % n];
    if (ti !== CF_BACK) fv.push(vi);
    if (ti !== CF_FRONT) bv.push(vi);
    if ((ti === CF_FRONT && tj === CF_BACK) || (ti === CF_BACK && tj === CF_FRONT)) {
      var den = vDot(plane.n, vSub(vj, vi));
      var s = Math.abs(den) > EPS ? (plane.w - vDot(plane.n, vi)) / den : 0.5;
      var cut = [vi[0] + s * (vj[0] - vi[0]), vi[1] + s * (vj[1] - vi[1]), vi[2] + s * (vj[2] - vi[2])];
      fv.push(cut); bv.push(cut);
    }
  }
  return { coplanar: null, front: fv.length >= 3 ? fv : null, back: bv.length >= 3 ? bv : null };
}

// 多边形（凸）扇形三角化
function fanTriangles(verts) {
  var out = [];
  for (var k = 1; k < verts.length - 1; k++) out.push([verts[0], verts[k], verts[k + 1]]);
  return out;
}

// 用平面分裂一个三角形 →
//   { coplanar: [tri...]（与平面共面，调用方决定归属：建树时存本节点、裁剪时按朝向分流）,
//     front: [tri...], back: [tri...] }
function splitTriangleByPlane(plane, tri, triPl) {
  var res = splitConvexByPlane(plane, tri);
  var out = { coplanar: [], front: [], back: [] };
  if (res.coplanar) { out.coplanar.push(tri); return out; }
  if (res.front) { var ft = fanTriangles(res.front); for (var k = 0; k < ft.length; k++) out.front.push(ft[k]); }
  if (res.back) { var bt = fanTriangles(res.back); for (var m = 0; m < bt.length; m++) out.back.push(bt[m]); }
  return out;
}

function bspNew() { return { plane: null, front: null, back: null, tris: [] }; }

// 建树（迭代：显式工作栈，避免深递归）
function bspBuild(root, tris) {
  if (!tris.length) return;
  var stack = [{ node: root, tris: tris }];
  while (stack.length) {
    var item = stack.pop(), node = item.node, list = item.tris, k;
    if (!list.length) continue;
    if (!node.plane) {
      node.plane = triPlane(list[0]);
      if (!node.plane) { // 首个三角形退化 → 找一个非退化的
        for (k = 0; k < list.length; k++) {
          node.plane = triPlane(list[k]);
          if (node.plane) break;
        }
      }
      if (!node.plane) continue; // 全部退化
    }
    var f = [], b = [];
    for (k = 0; k < list.length; k++) {
      var tri = list[k], pl = triPlane(tri);
      if (!pl) continue;
      var r = splitTriangleByPlane(node.plane, tri, pl);
      // 共面三角形 = 本节点的"表面"（csg.js 语义：直接存本节点，不再下推）
      for (var a = 0; a < r.coplanar.length; a++) node.tris.push(r.coplanar[a]);
      for (var c = 0; c < r.front.length; c++) f.push(r.front[c]);
      for (var d = 0; d < r.back.length; d++) b.push(r.back[d]);
    }
    if (f.length) { node.front = node.front || bspNew(); stack.push({ node: node.front, tris: f }); }
    if (b.length) { node.back = node.back || bspNew(); stack.push({ node: node.back, tris: b }); }
  }
}

// 用 BSP 树裁剪一组三角形（保留树"外部/正面"的部分；无 back 子树时 back 侧被丢弃）
function bspClipTriangles(node, tris) {
  var out = [], stack = [{ node: node, tris: tris }];
  while (stack.length) {
    var item = stack.pop(), nd = item.node, list = item.tris, k;
    if (!list.length) continue;
    if (!nd.plane) { for (k = 0; k < list.length; k++) out.push(list[k]); continue; }
    var f = [], b = [];
    for (k = 0; k < list.length; k++) {
      var tri = list[k], pl = triPlane(tri);
      if (!pl) continue;
      var r = splitTriangleByPlane(nd.plane, tri, pl);
      // 裁剪时共面三角形按朝向分流（csg.js 的 clipPolygons 传 coplanarFront=front / coplanarBack=back）
      for (var a = 0; a < r.coplanar.length; a++) {
        (vDot(nd.plane.n, pl.n) > 0 ? f : b).push(r.coplanar[a]);
      }
      for (var c = 0; c < r.front.length; c++) f.push(r.front[c]);
      for (var d = 0; d < r.back.length; d++) b.push(r.back[d]);
    }
    if (nd.front) stack.push({ node: nd.front, tris: f });
    else for (k = 0; k < f.length; k++) out.push(f[k]);
    if (nd.back) stack.push({ node: nd.back, tris: b });
    // 无 back 子树 → b 丢弃（正是 clipTo 语义）
  }
  return out;
}

function bspClipTo(node, other) {
  var stack = [node];
  while (stack.length) {
    var nd = stack.pop();
    if (nd.tris.length) nd.tris = bspClipTriangles(other, nd.tris);
    if (nd.front) stack.push(nd.front);
    if (nd.back) stack.push(nd.back);
  }
}

function bspInvert(node) {
  var stack = [node];
  while (stack.length) {
    var nd = stack.pop(), k;
    for (k = 0; k < nd.tris.length; k++) nd.tris[k] = triFlip(nd.tris[k]);
    if (nd.plane) nd.plane = { n: vMul(nd.plane.n, -1), w: -nd.plane.w };
    var tmp = nd.front; nd.front = nd.back; nd.back = tmp;
    if (nd.front) stack.push(nd.front);
    if (nd.back) stack.push(nd.back);
  }
}

function bspAllTris(node) {
  var out = [], stack = [node];
  while (stack.length) {
    var nd = stack.pop();
    for (var k = 0; k < nd.tris.length; k++) out.push(nd.tris[k]);
    if (nd.front) stack.push(nd.front);
    if (nd.back) stack.push(nd.back);
  }
  return out;
}

// ── I.6-2 原型：多边形 BSP（csg.js 的原生表示 —— BSP 全程保凸多边形，导出前才三角化）──
// 动机（I.6-1 的实测结论）：三角路径在**每次切割后立即扇形三角化**（splitTriangleByPlane），
//   相邻面片的边分段方式于是各自独立 ⇒ T 型接缝 + 浮点碎片；I.6-1 在三角域事后合并，碰上
//   「BSP 输出本身有重叠」的簇只能整簇回退。多边形表示下一条边对一条边，切割点由同一对端点
//   算出，T 缝从源头消失；且**同一个 BSP 节点的多边形天然共面** ⇒ 合并退化成「对消内部边 +
//   串边界环」，不需要 T 缝细分 / 噪声清理 / 容差匹配。
//   ★ 凸性不变量：初始多边形是三角形（凸），凸多边形被平面切成两块仍凸 ⇒ BSP 内多边形恒凸
//     ⇒ 未合并时扇形三角化即正确；合并后才可能出现凹，交给 earcut 版 triangulatePolygonWithHoles。
//   ★ 安全回退：任何节点合并失败都退化为「逐多边形扇形三角化」——凸性保证这一步**永远正确**，
//     所以多边形路径不存在"输出坏几何"的风险面（与三角路径的回退相比强得多）。
//   ★ 不改默认路径：meshBoolean 仍走三角路径，本族函数仅供 meshBooleanPoly 使用，可逐位对照。
function meshToPolys(m) {
  var out = [], k;
  for (k = 0; k < m.indices.length; k++) {
    var t = m.indices[k];
    out.push([m.positions[t[0]], m.positions[t[1]], m.positions[t[2]]]);
  }
  return out;
}

// 多边形所在平面（取前三个不共线顶点；凸多边形的标准做法）
function polyPlane(verts) {
  for (var i = 2; i < verts.length; i++) {
    var p = triPlane([verts[0], verts[1], verts[i]]);
    if (p) return p;
  }
  return null;
}

function polyFlip(verts) { return verts.slice().reverse(); }

function polyBspNew() { return { plane: null, front: null, back: null, polys: [] }; }

// 建树（迭代式工作栈，与三角版同构；差别只在**不**把切割结果三角化）
function polyBspBuild(root, polys) {
  if (!polys.length) return;
  var stack = [{ node: root, polys: polys }];
  while (stack.length) {
    var item = stack.pop(), node = item.node, list = item.polys, k;
    if (!list.length) continue;
    if (!node.plane) {
      for (k = 0; k < list.length; k++) { node.plane = polyPlane(list[k]); if (node.plane) break; }
      if (!node.plane) continue;                     // 整层退化（零面积多边形）
    }
    var f = [], b = [];
    for (k = 0; k < list.length; k++) {
      var r = splitConvexByPlane(node.plane, list[k]);
      if (r.coplanar) node.polys.push(r.coplanar);   // 共面 = 本节点表面（csg.js 语义：不再下推）
      if (r.front) f.push(r.front);
      if (r.back) b.push(r.back);
    }
    if (f.length) { node.front = node.front || polyBspNew(); stack.push({ node: node.front, polys: f }); }
    if (b.length) { node.back = node.back || polyBspNew(); stack.push({ node: node.back, polys: b }); }
  }
}

// 用 BSP 树裁剪一组多边形（保留树"外部/正面"的部分；无 back 子树时 back 侧被丢弃 = clipTo 语义）
function polyBspClip(node, polys) {
  var out = [], stack = [{ node: node, polys: polys }];
  while (stack.length) {
    var item = stack.pop(), nd = item.node, list = item.polys, k;
    if (!list.length) continue;
    if (!nd.plane) { for (k = 0; k < list.length; k++) out.push(list[k]); continue; }
    var f = [], b = [];
    for (k = 0; k < list.length; k++) {
      var poly = list[k], pl = polyPlane(poly);
      if (!pl) continue;
      var r = splitConvexByPlane(nd.plane, poly);
      // 裁剪时共面多边形按朝向分流（csg.js 的 clipPolygons：coplanarFront=front / coplanarBack=back）
      if (r.coplanar) (vDot(nd.plane.n, pl.n) > 0 ? f : b).push(r.coplanar);
      if (r.front) f.push(r.front);
      if (r.back) b.push(r.back);
    }
    if (nd.front) stack.push({ node: nd.front, polys: f });
    else for (k = 0; k < f.length; k++) out.push(f[k]);
    if (nd.back) stack.push({ node: nd.back, polys: b });
  }
  return out;
}

function polyBspClipTo(node, other) {
  var stack = [node];
  while (stack.length) {
    var nd = stack.pop();
    if (nd.polys.length) nd.polys = polyBspClip(other, nd.polys);
    if (nd.front) stack.push(nd.front);
    if (nd.back) stack.push(nd.back);
  }
}

function polyBspInvert(node) {
  var stack = [node];
  while (stack.length) {
    var nd = stack.pop(), k;
    for (k = 0; k < nd.polys.length; k++) nd.polys[k] = polyFlip(nd.polys[k]);
    if (nd.plane) nd.plane = { n: vMul(nd.plane.n, -1), w: -nd.plane.w };
    var tmp = nd.front; nd.front = nd.back; nd.back = tmp;
    if (nd.front) stack.push(nd.front);
    if (nd.back) stack.push(nd.back);
  }
}

// 摊平全部多边形（对应三角版的 bspAllTris；union/intersect 收尾要把 B 的面并入 A 重建）
function polyBspAllPolys(node) {
  var out = [], stack = [node];
  while (stack.length) {
    var nd = stack.pop(), k;
    for (k = 0; k < nd.polys.length; k++) out.push(nd.polys[k]);
    if (nd.front) stack.push(nd.front);
    if (nd.back) stack.push(nd.back);
  }
  return out;
}

// 按**节点**收集（I.6-2 的关键：同一节点的 polys 共面，正是要合并的单元；
//   三角版把结果摊平成一张三角形表，这个「共面分组」信息就丢了 —— I.6-1 只能事后自己再聚类）
function polyBspNodes(node) {
  var out = [], stack = [node];
  while (stack.length) {
    var nd = stack.pop();
    out.push(nd);
    if (nd.front) stack.push(nd.front);
    if (nd.back) stack.push(nd.back);
  }
  return out;
}

// 轻量"缝量"指标：严格未配对的**半边**数（不建空间索引，O(F)）。
//   用于 meshRepair 的**不劣化保护**：消解是启发式的，在"多簇共面合并"等复杂输入上可能帮倒忙
//   （实测：板 − 8 孔场景 10 → 30）。凡消解后该指标变差，就回退到消解前的网格。
function meshSeamScore(m) {
  var tris = m.indices, has = {}, k, e;
  for (k = 0; k < tris.length; k++) {
    var f = tris[k];
    for (e = 0; e < 3; e++) has[f[e] + '_' + f[(e + 1) % 3]] = 1;
  }
  var n = 0;
  for (k = 0; k < tris.length; k++) {
    var g = tris[k];
    for (e = 0; e < 3; e++) if (!has[g[(e + 1) % 3] + '_' + g[e]]) n++;
  }
  return n;
}

// ── T 缝消解之一：把"落在未配对边内部、且贴近该边端点"的顶点吸附到端点 ────────────
//   动机（实测，2026-09-17）：BSP 布尔的分割点是由**每次布尔各自的容差**算出来的，跨次错开
//   ~1 eps（实例：(-20.2330616,-15.1227675) 与 (-20.2334149,-15.1231208)，相距 1.4 eps）。
//   这类"微边"用 0.5×eps 焊接不掉 ⇒ 在半边的严格意义上找不到配对 ⇒ M4 报 758 条真缺口。
//   收敛把关：只处理**严格未配对**的有向边（约占 16%），且顶点必须确实落在该边上（垂距 ≤ eps）。
//   只有"顶点沿边方向距端点 ≤ eps"才吸附到端点 —— 否则它是真正的分割点，交给 meshFillTJunctions。
function meshAbsorbToEnds(m, eps) {
  var pos = m.positions, tris = m.indices;
  if (!tris.length || !pos.length) return m;
  var has = {}, f, e, k;
  for (k = 0; k < tris.length; k++) {
    f = tris[k];
    for (e = 0; e < 3; e++) has[f[e] + '_' + f[(e + 1) % 3]] = 1;
  }
  var tgt = [], bd = [];
  for (k = 0; k < pos.length; k++) { tgt.push(null); bd.push(Infinity); }
  var n = 0;
  for (k = 0; k < tris.length; k++) {
    f = tris[k];
    for (e = 0; e < 3; e++) {
      var a = f[e], b = f[(e + 1) % 3];
      if (has[b + '_' + a]) continue;                  // 已配对 ⇒ 不动
      var pa = pos[a], pb = pos[b], ab = vSub(pb, pa), L2 = vDot(ab, ab);
      if (!(L2 > 0)) continue;
      var L = Math.sqrt(L2);
      for (var v = 0; v < pos.length; v++) {
        if (v === a || v === b) continue;
        var pv = pos[v], w = vSub(pv, pa);
        var t = vDot(w, ab) / L2;
        if (t <= 0 || t >= 1) continue;
        var pr = [pa[0] + ab[0] * t, pa[1] + ab[1] * t, pa[2] + ab[2] * t];
        var dd = vDist(pr, pv);
        if (dd > eps || dd >= bd[v]) continue;
        var to = null;
        if (t * L <= eps) to = pa;
        else if ((1 - t) * L <= eps) to = pb;
        if (!to) continue;
        bd[v] = dd; tgt[v] = to; n++;
      }
    }
  }
  if (!n) return m;
  var out = [];
  for (k = 0; k < pos.length; k++) out.push(tgt[k] ? tgt[k].slice() : pos[k]);
  return meshNew(out, tris);
}

// ── T 缝消解之二：把"落在未配对边内部"的顶点插入该边（链式分割 + 从对顶点扇形）──────
//   ① 必须从**边的对顶点**出发逐条边分割并递归：对整条边界链扇形是错的 —— 链上
//      "插入点 + 该边两端点"三点共线，扇形会造出 [a,v1,b] 这样的零面积三角形，
//      它在对面留下同向重复边（实测 4 条非流形），还会把角三角形丢掉（实测 11 条单边）。
//   ② 插入点排序必须稳定（t 相同时按顶点索引）：否则同一组点在相邻三角形里顺序不同，
//      链式展开就会产出同向重复边 ⇒ 非流形。
//   ③ 本函数只读不写坐标；合并/去隐患由调用方的 meshWeld + meshDropDegenerate 完成。
function meshFillTJunctions(m, eps) {
  var pos = m.positions, tris = m.indices;
  if (!tris.length || !pos.length) return m;
  var has = {}, k, f, e;
  for (k = 0; k < tris.length; k++) {
    f = tris[k];
    for (e = 0; e < 3; e++) has[f[e] + '_' + f[(e + 1) % 3]] = 1;
  }
  var ins = {}, nE = 0, nI = 0;
  for (k = 0; k < tris.length; k++) {
    f = tris[k];
    for (e = 0; e < 3; e++) {
      var a = f[e], b = f[(e + 1) % 3];
      if (has[b + '_' + a]) continue;                  // 已配对 ⇒ 不切
      var key = a + '|' + b;
      if (ins[key]) continue;                          // 同一有向边只算一次
      nE++;
      var pa = pos[a], pb = pos[b], ab = vSub(pb, pa), L2 = vDot(ab, ab);
      if (!(L2 > 0)) { ins[key] = []; continue; }
      var list = [];
      for (var v = 0; v < pos.length; v++) {
        if (v === a || v === b) continue;
        var pv = pos[v], w = vSub(pv, pa);
        var t = vDot(w, ab) / L2;
        if (t <= 0 || t >= 1) continue;
        var pr = [pa[0] + ab[0] * t, pa[1] + ab[1] * t, pa[2] + ab[2] * t];
        if (vDist(pr, pv) > eps) continue;
        list.push([t, v]);
      }
      list.sort(function (x, y) { return (x[0] - y[0]) || (x[1] - y[1]); });
      ins[key] = list; nI += list.length;
    }
  }
  if (!nI) return m;
  var out = [];
  function buildTri(a, b, c) {
    var tri = [[a, b, c], [b, c, a], [c, a, b]];
    for (var i = 0; i < 3; i++) {
      var u = tri[i][0], w2 = tri[i][1], x = tri[i][2];
      var list = ins[u + '|' + w2];
      if (!list || !list.length) continue;
      var seq = [u];
      for (var q = 0; q < list.length; q++) {
        var iv = list[q][1];
        if (iv !== u && iv !== w2) seq.push(iv);
      }
      if (seq.length < 2) continue;
      seq.push(w2);
      for (var j = 0; j + 1 < seq.length; j++) buildTri(seq[j], seq[j + 1], x);
      return;                                          // 该子三角形已递归展开
    }
    if (a !== b && b !== c && a !== c) out.push([a, b, c]);
  }
  for (k = 0; k < tris.length; k++) buildTri(tris[k][0], tris[k][1], tris[k][2]);
  return meshNew(pos, out);
}

// 布尔结果修复：顶点吸附到统一网格 → 焊接 → T 缝消解 → 再焊接 → 去零面积面 → 去孤立顶点。
//   ★ 实测结论（有 JSCAD 对照，见测试基线）：BSP 布尔的输出在"严格边匹配"意义下**本来就不水密**——
//     同一交点在相邻面里由各自插值算出（坐标差约 1e-4 量级），相邻面片的边分段方式还可能不同
//     （T 型接缝）。JSCAD 的原始 subtract 输出同样如此（boundary=152，其 generalize 后才是 0）——
//     它靠的是**多边形**表示（一条边对一条边，天生没有 T 缝）。
//   本实现是三角网格表示，故用"吸附 + 焊接 + **T 缝消解**"把缝压到零，并把**残余缝量**如实交给
//     判据 M4 报告（不假称水密）；确实需要严格水密的 3D 打印场景走导出层的打印预处理。
//   ★ 历史与订正：早期试过"T 型接缝缝合"（把落在边内的顶点纳入三角形重新三角化）——实测有害：
//     上千条非流形边 + 三角形顶到 30 万级。2026-09-17 定位到当时的做法有两处硬伤，订正后成功：
//     ① **无收敛条件**：当时对全部边、所有落在其上的顶点都插（含大量本不该动的），这里只处理
//        **严格未配对**的有向边（约占 16%），且顶点垂距必须 ≤ eps；
//     ② **三角化方式错**：对整条边界链扇形会把"插入点 + 该边两端点"三点共线拼成零面积三角形，
//        在对面留下同向重复边（非流形），还丢掉角三角形（单边）。订正为**从对顶点出发逐边分割并递归**。
//     订正后（同上场景）：真缺口 758 → **0**、严格未配对 0、非流形 0、单边 0，面数 2102 → 2540
//     （+20.8%，而非 30 万级），体积相对变化 1.3e-6、连通分量保持 1。
function meshRepair(m, epsOverride) {
  var eps = epsOverride === undefined ? meshEpsilon(m) : epsOverride;
  var wt = eps * WELD_RATIO;               // 见 WELD_RATIO 注释（T 缝消解需要 >1×eps 的合并半径）
  var cur = meshSnap(m, eps);
  cur = meshWeld(cur, wt).mesh;
  cur = meshDropDegenerate(meshCompact(cur), EPS_AREA);
  var base = cur;
  var score0 = meshSeamScore(base);
  if (!score0) return { mesh: base, epsilon: eps };          // 已水密 ⇒ 不做消解（省时且零风险）
  cur = meshWeld(meshAbsorbToEnds(base, eps), wt).mesh;      // ① eps 级微边并入端点
  cur = meshWeld(meshFillTJunctions(cur, eps), wt).mesh;     // ② 真分割点切开未配对边
  cur = meshDropDegenerate(meshCompact(cur), EPS_AREA);
  // ★ 不劣化保护：消解是启发式的，在复杂输入上可能帮倒忙（实测 板−8孔 10→30）。变差即回退，
  //   保证 meshRepair 的输出在"缝量"上**单调不劣于**不做消解的结果。
  if (meshSeamScore(cur) > score0) return { mesh: base, epsilon: eps };
  return { mesh: cur, epsilon: eps };
}

// ── 共面三角形合并（I.6-1：BSP 碎片的根治手段）───────────────────────────────
// 动机：BSP 裁剪把每个面切成大量小三角形 —— 60×40×6 底板挖 4 个 ∅6 孔得 **5823 面**（理想 ~400），
//   碎片同时是 T 缝与导出体积膨胀（glTF 826 KB / STL 395 KB）的来源。
// 口径：**按平面分组 → 簇内边界环 → 环嵌套分类（外环/洞）→ 带孔重三角化 → 面积守恒校验**；
//   任一环节不通过则**整簇回退** —— 宁可留碎片，也绝不产出"看起来更少但其实破了"的网格。
//   · 只合并**同朝向**三角形（外法线一致）：薄壁的内外两面共面但朝向相反，不可合并。
//   · 共边判据 = **顶点索引相同**（调用前必须已焊接）；顶点落在边内部的 T 缝按投影参数分裂边消解。
//   · 平面内坐标 = 原始坐标 ÷ eps：顶点此前已吸附到 eps 格点 ⇒ 缩放后为整数格点，面积量级大、
//     共线/水平边判据都落在整数量级上（沿用 triangulatePolygonWithHoles 的 EPS 判据才稳）。
//   · 返回 { mesh, stats }：stats 供测试与诊断（合并/回退的簇数、面数前后）。
function meshMergeCoplanar(m, epsIn) {
  var eps = epsIn === undefined ? meshEpsilon(m) : epsIn;
  var src = m.indices, pos = m.positions;
  var stats = { clusters: 0, merged: 0, fellBack: 0, trisBefore: src.length, trisAfter: src.length };
  if (src.length < 2 || pos.length < 3) return { mesh: m, stats: stats };
  var k, j, i;
  // ① 三角形 → 平面（单位法线 n + 偏移 d = n·p）；退化面不参与合并
  var nrm = [], off = [];
  for (k = 0; k < src.length; k++) {
    var t = meshTriangle(m, k);
    var cr = vCross(vSub(t[1], t[0]), vSub(t[2], t[0]));
    var L = vLen(cr);
    if (!(L > EPS_AREA)) { nrm.push(null); off.push(0); continue; }
    var n = vMul(cr, 1 / L);
    nrm.push(n);
    off.push(vDot(n, t[0]));
  }
  // ② 分组：法线粗桶（1e-3 分辨率）→ 桶内按偏移聚类（容差 eps）。
  //    同一平面的法线出自同一批计算路径（差异 ~1e-16）⇒ 必然同桶；恰被桶边界拆开的只是
  //    "少合并一些"，不会合错（偏差落在保守方向）。
  var buckets = {};
  for (k = 0; k < src.length; k++) {
    if (!nrm[k]) continue;
    var bk = Math.round(nrm[k][0] * 1000) + ':' + Math.round(nrm[k][1] * 1000) + ':' + Math.round(nrm[k][2] * 1000);
    (buckets[bk] = buckets[bk] || []).push(k);
  }
  // ③ 逐簇合并；替换表：null = 原样保留 / [] = 已被吸收 / 列表 = 替代三角形
  var repl = [];
  for (k = 0; k < src.length; k++) repl.push(null);
  for (var bkey in buckets) {
    var list = buckets[bkey];
    if (list.length < 2) continue;
    var reps = [];
    for (j = 0; j < list.length; j++) {
      var found = -1;
      for (i = 0; i < reps.length; i++) {
        if (Math.abs(reps[i].d - off[list[j]]) <= eps) { found = i; break; }
      }
      if (found < 0) reps.push({ d: off[list[j]], tris: [list[j]] });
      else reps[found].tris.push(list[j]);
    }
    for (i = 0; i < reps.length; i++) {
      var cl = reps[i].tris;
      if (cl.length < 2) continue;
      stats.clusters++;
      var res = null;
      try { res = coplanarMergeCluster(m, cl, nrm[cl[0]], eps); } catch (e) { res = null; }
      if (res && res.length) {
        repl[cl[0]] = res;
        for (j = 1; j < cl.length; j++) repl[cl[j]] = [];
        stats.merged++;
      } else {
        stats.fellBack++;
      }
    }
  }
  // ④ 重建（顶点表原样复用 ⇒ 坐标逐位不变；末尾去掉孤立顶点）
  var outIdx = [];
  for (k = 0; k < src.length; k++) {
    var rr = repl[k];
    if (!rr) { outIdx.push(src[k].slice()); continue; }
    for (j = 0; j < rr.length; j++) outIdx.push(rr[j]);
  }
  stats.trisAfter = outIdx.length;
  if (stats.merged === 0) return { mesh: m, stats: stats };
  return { mesh: meshCompact(meshNew(pos, outIdx)), stats: stats };
}

// 单个共面簇 → 替代三角形（顶点索引三元组）；返回 null 表示**回退**（保持原碎片）
// 共面合并的回退原因记录（诊断用：回退不是错误，但必须能说清"为什么没合并"）
var CP_REASON = { last: '', counts: {} };
function cpRevert(reason) {
  CP_REASON.last = reason;
  CP_REASON.counts[reason] = (CP_REASON.counts[reason] || 0) + 1;
  return null;
}

// 三角形的有向键（从最小顶点索引起、按绕向展开）：绕向相反的重复面键不同
function triKeyOf(T) {
  var m = (T[0] < T[1]) ? ((T[0] < T[2]) ? 0 : 2) : ((T[1] < T[2]) ? 1 : 2);
  return T[m] + ':' + T[(m + 1) % 3] + ':' + T[(m + 2) % 3];
}

function coplanarMergeCluster(m, cl, n, eps) {
  var pos = m.positions, k, j;
  var inv = 1 / eps;
  // 平面内正交基（右手：u × v = n）⇒ 2D 投影是等距映射，长度/面积都守恒
  var ax = (Math.abs(n[0]) <= Math.abs(n[1]) && Math.abs(n[0]) <= Math.abs(n[2])) ? [1, 0, 0]
         : (Math.abs(n[1]) <= Math.abs(n[2]) ? [0, 1, 0] : [0, 0, 1]);
  var uu = vNorm(vCross(ax, n)), vv = vCross(n, uu);
  var cache = [];
  function to2(vi) {
    if (cache[vi]) return cache[vi];
    var p = pos[vi];
    var q = [vDot(uu, p) * inv, vDot(vv, p) * inv];
    cache[vi] = q;
    return q;
  }
  // ⓪ 簇内 T 缝细分（BSP 输出的固有形态：几何上闭合、分段不同 —— 顶点落在**三角形边**内部）。
  //    不先消解则边界环串不起来（实测最大簇有 160 处）。做法：落在某条边内部的顶点按投影参数
  //    插入该边，三角形随之 1→n+1 分裂（纯拓扑细分，覆盖与面积不变）。
  //    ★ 必须对**所有**三角形边做，不能只对边界边 —— 实测 T 缝的宿主边多数是内部边（两侧都有面）。
  var cur = [];
  for (k = 0; k < cl.length; k++) cur.push(m.indices[cl[k]].slice());
  for (var rds = 0; rds < 8; rds++) {
    var vset0 = {}, q, ch = false;
    for (k = 0; k < cur.length; k++) { vset0[cur[k][0]] = 1; vset0[cur[k][1]] = 1; vset0[cur[k][2]] = 1; }
    var vl = [], vk0;
    for (vk0 in vset0) vl.push(+vk0);
    // 顶点空间网格（格 = 2D bbox 的 1/64）：边只查自身 AABB 覆盖的格子 ⇒ 避免 O(E·V)
    var gx0 = 0, gy0 = 0, gx1 = 0, gy1 = 0;
    for (q = 0; q < vl.length; q++) {
      var qc = to2(vl[q]);
      if (q === 0) { gx0 = gx1 = qc[0]; gy0 = gy1 = qc[1]; }
      if (qc[0] < gx0) gx0 = qc[0];
      if (qc[0] > gx1) gx1 = qc[0];
      if (qc[1] < gy0) gy0 = qc[1];
      if (qc[1] > gy1) gy1 = qc[1];
    }
    var gs = Math.max(1, Math.max(gx1 - gx0, gy1 - gy0) / 64);
    var grid = {};
    for (q = 0; q < vl.length; q++) {
      var qd = to2(vl[q]);
      var gk = Math.floor((qd[0] - gx0) / gs) + ':' + Math.floor((qd[1] - gy0) / gs);
      (grid[gk] = grid[gk] || []).push(vl[q]);
    }
    var out2 = [];
    for (k = 0; k < cur.length; k++) {
      var Tc = cur[k], split = false;
      for (j = 0; j < 3; j++) {
        var ia = Tc[j], ib = Tc[(j + 1) % 3], ic = Tc[(j + 2) % 3];
        var A2 = to2(ia), B2 = to2(ib);
        var ex = B2[0] - A2[0], ey = B2[1] - A2[1];
        var L2s = ex * ex + ey * ey;
        if (!(L2s > 4)) continue;                       // 短边（≤ 2 格）不查
        var Ls = Math.sqrt(L2s);
        var bx0 = Math.min(A2[0], B2[0]) - 1, bx1 = Math.max(A2[0], B2[0]) + 1;
        var by0 = Math.min(A2[1], B2[1]) - 1, by1 = Math.max(A2[1], B2[1]) + 1;
        var ga0 = Math.floor((bx0 - gx0) / gs), ga1 = Math.floor((bx1 - gx0) / gs);
        var gb0 = Math.floor((by0 - gy0) / gs), gb1 = Math.floor((by1 - gy0) / gs);
        var hits0 = [], ga, gb;
        for (ga = ga0; ga <= ga1; ga++) {
          for (gb = gb0; gb <= gb1; gb++) {
            var cells = grid[ga + ':' + gb];
            if (!cells) continue;
            for (q = 0; q < cells.length; q++) {
              var pv = cells[q];
              if (pv === ia || pv === ib || pv === ic) continue;
              var P2 = to2(pv);
              var px = P2[0] - A2[0], py = P2[1] - A2[1];
              if (Math.abs(ex * py - ey * px) > Ls * 0.25) continue;   // 到直线距离 > eps/4 才不算 T 缝
              var tp = (px * ex + py * ey) / L2s;
              if (tp * Ls <= 1 || (1 - tp) * Ls <= 1) continue;   // 距端点 ≤ eps
              hits0.push({ p: pv, t: tp });
            }
          }
        }
        if (!hits0.length) continue;
        hits0.sort(function (u, w) { return u.t - w.t; });
        var prev0 = ia;
        for (q = 0; q < hits0.length; q++) { out2.push([prev0, hits0[q].p, ic]); prev0 = hits0[q].p; }
        out2.push([prev0, ib, ic]);
        split = true; ch = true;
        break;                                          // 一轮只细分一条边（其余留到下一轮）
      }
      if (!split) out2.push(Tc);
    }
    cur = out2;
    if (!ch) break;
    if (cur.length > cl.length * 8 + 64) return cpRevert('T 缝细分膨胀过大');   // 病态几何保护
  }
  // ① 有向边统计：同向边出现 >1 次 ⇒ 非流形。但若成因是「完全重复的三角形」（BSP 偶发产物）
  //    则先去重再判 —— 重复面会让边统计与覆盖面积同时错，去重后两处都自洽（面积校验仍兜底）。
  function countEdges(list) {
    var c = {}, kk, jj;
    for (kk = 0; kk < list.length; kk++) {
      for (jj = 0; jj < 3; jj++) {
        var key = list[kk][jj] + ':' + list[kk][(jj + 1) % 3];
        c[key] = (c[key] || 0) + 1;
      }
    }
    return c;
  }
  var cnt = countEdges(cur), ekey, hasDup = false;
  for (ekey in cnt) {
    if (cnt[ekey] > 1) { hasDup = true; break; }
  }
  if (hasDup) {
    // (a) 完全重复的三角形（BSP 偶发产物）去重
    var seenT = {}, uniq = [];
    for (k = 0; k < cur.length; k++) {
      var Tk = triKeyOf(cur[k]);
      if (seenT[Tk]) continue;
      seenT[Tk] = 1;
      uniq.push(cur[k]);
    }
    if (uniq.length !== cur.length) cur = uniq;
    // (b) 微碎片清理：面积 ≤ 4 格²（2D 缩放坐标）的三角形是 BSP 在孔/边附近浮点噪声的产物，
    //     正是"同一条短边被 5 个面共享"这类非流形的制造者。删除它们对覆盖率的影响在 1e-8 相对
    //     量级（由 ⑦ 的弱校验记账）；若清理后边界不闭合，后面各步仍会回退，不会产出坏网格。
    var keep = [];
    for (k = 0; k < cur.length; k++) {
      var Tm = cur[k];
      var m0 = to2(Tm[0]), m1 = to2(Tm[1]), m2 = to2(Tm[2]);
      var arm = (m1[0] - m0[0]) * (m2[1] - m0[1]) - (m1[1] - m0[1]) * (m2[0] - m0[0]);
      if (Math.abs(arm) / 2 > 4) keep.push(Tm);
    }
    if (keep.length !== cur.length) cur = keep;
    if (cur.length < 3) return cpRevert('清理微碎片后三角形不足');
    cnt = countEdges(cur);
    var still = false;
    for (ekey in cnt) { if (cnt[ekey] > 1) { still = true; break; } }
    if (still) {
      // (c) 清理后仍非流形 ⇒ 回退。曾试过「短边塌缩」（把 ≤3 格的边两端点并成一个代表）——
      //     它会**移动顶点、改变几何**（实测单孔场景体积偏差 4e-8），且并未真正解决非流形，已删除。
      return cpRevert('非流形：同向边出现多次（清理后仍在）');
    }
  }
  var edges = [];
  for (ekey in cnt) {
    var ab = ekey.split(':');
    if (cnt[ab[1] + ':' + ab[0]]) continue;            // 内部边成对出现 ⇒ 跳过
    edges.push([+ab[0], +ab[1]]);
  }
  if (edges.length < 3) return cpRevert('边界边不足 3 条');
  // ② T 缝消解：边界顶点落在某条边界边内部 ⇒ 按投影参数分裂该边（不消解则环串不起来）
  for (var rd = 0; rd < 4; rd++) {
    var vset = {}, changed = false, vk;
    for (k = 0; k < edges.length; k++) { vset[edges[k][0]] = 1; vset[edges[k][1]] = 1; }
    var vlist = [];
    for (vk in vset) vlist.push(+vk);
    var next = [];
    for (k = 0; k < edges.length; k++) {
      var a = edges[k][0], b = edges[k][1];
      var A = to2(a), B = to2(b);
      var abx = B[0] - A[0], aby = B[1] - A[1];
      var L2 = abx * abx + aby * aby;
      if (!(L2 > 1)) { next.push(edges[k]); continue; }   // 边长不足一个格距 ⇒ 不动
      var L = Math.sqrt(L2);
      var hits = [];
      for (j = 0; j < vlist.length; j++) {
        var p = vlist[j];
        if (p === a || p === b) continue;
        var Q = to2(p);
        var qx = Q[0] - A[0], qy = Q[1] - A[1];
        if (Math.abs(abx * qy - aby * qx) > L * 0.25) continue;   // 到直线距离 > eps/4
        var tt = (qx * abx + qy * aby) / L2;
        if (tt * L <= 1 || (1 - tt) * L <= 1) continue;   // 距端点 ≤ eps ⇒ 视为端点而非 T 缝
        hits.push({ p: p, t: tt });
      }
      if (!hits.length) { next.push(edges[k]); continue; }
      hits.sort(function (x, y) { return x.t - y.t; });
      var prev = a;
      for (j = 0; j < hits.length; j++) { next.push([prev, hits[j].p]); prev = hits[j].p; }
      next.push([prev, b]);
      changed = true;
    }
    edges = next;
    if (!changed) break;
  }
  // ③ 串环：**边界跟随**（平面图的面遍历）—— 允许顶点处出度 > 1（BSP 输出的边界可能在一点
  //    自接触/分叉）。在每个顶点处选「相对入边反向的最小左转」出边 ⇒ 沿覆盖域边界行走；
  //    规整簇（出/入度均为 1）下这是唯一选择，结果与"严格配对"完全一致。
  var outList = {}, inList = {}, ea, eb;
  for (k = 0; k < edges.length; k++) {
    ea = edges[k][0]; eb = edges[k][1];
    (outList[ea] = outList[ea] || []).push(eb);
    (inList[eb] = inList[eb] || []).push(ea);
  }
  function turnAt(P, from, to) {                 // 在 P 处从「P→from」转到「P→to」的转角 ∈ (-π, π]
    var a0 = Math.atan2(from[1] - P[1], from[0] - P[0]);
    var b0 = Math.atan2(to[1] - P[1], to[0] - P[0]);
    var t0 = b0 - a0;
    while (t0 <= -Math.PI) t0 += 2 * Math.PI;
    while (t0 > Math.PI) t0 -= 2 * Math.PI;
    return t0;
  }
  var usedE = {}, rings = [], totalVs = 0, usedN = 0;
  for (k = 0; k < edges.length; k++) {
    if (usedE[edges[k][0] + ':' + edges[k][1]]) continue;
    var ring = [], ra = edges[k][0], rb = edges[k][1], guard = 0, closed = false;
    while (true) {
      var ek2 = ra + ':' + rb;
      if (usedE[ek2]) break;
      usedE[ek2] = 1; usedN++;
      ring.push(ra);
      var outs = outList[rb] || [], pick = -1, bestT = Infinity, P3 = to2(rb), Q3 = to2(ra);
      var sA = edges[k][0], sB = edges[k][1];
      for (j = 0; j < outs.length; j++) {
        var cand = outs[j];
        // 起始边允许作为"闭合步"再被选中（它此刻已被标记为已用）
        if (usedE[rb + ':' + cand] && !(rb === sA && cand === sB)) continue;
        var t3 = turnAt(P3, Q3, to2(cand));
        if (t3 < bestT) { bestT = t3; pick = cand; }
      }
      if (pick < 0) break;
      ra = rb; rb = pick;
      if (++guard > edges.length + 2) return cpRevert('环遍历超步数');
      if (ra === sA && rb === sB) { closed = true; break; }
      if (usedE[ra + ':' + rb]) break;               // 撞上已用边但不是起始边 ⇒ 该簇拓扑不可靠
    }
    if (!closed || ring.length < 3) return cpRevert('边界跟随未闭合（该簇拓扑不可靠）');
    var a2 = 0;
    for (j = 0; j < ring.length; j++) {
      var q1 = to2(ring[j]), q2 = to2(ring[(j + 1) % ring.length]);
      a2 += q1[0] * q2[1] - q2[0] * q1[1];
    }
    totalVs += ring.length;
    // 零宽通道（T 缝/自接触）会走出面积为 0 的退化环：它不贡献任何区域，直接忽略
    //（否则会被当成"方向不对的洞"而让整簇回退）。
    if (Math.abs(a2) <= 2) continue;
    rings.push({ vs: ring, a2: a2 });
  }
  if (!rings.length || usedN !== edges.length) return cpRevert('存在未被使用的边界边');
  // ④ 环嵌套分类：depth 偶 = 外环（CCW ⇒ a2 > 0），奇 = 洞（CW ⇒ a2 < 0）；方向不符则回退
  var ring2 = [];
  for (k = 0; k < rings.length; k++) {
    var arr = [];
    for (j = 0; j < rings[k].vs.length; j++) arr.push(to2(rings[k].vs[j]));
    ring2.push(arr);
  }
  function inRing(ri, rj) {                     // ri 是否落在 rj 内（顶点多数表决）
    var pts = ring2[ri], ins = 0, q;
    for (q = 0; q < pts.length; q++) if (pointInPoly(pts[q], ring2[rj])) ins++;
    return ins * 2 > pts.length;
  }
  var parent = [], depth = [];
  for (k = 0; k < rings.length; k++) {
    var best = -1, bestA = Infinity;
    for (j = 0; j < rings.length; j++) {
      if (j === k || !inRing(k, j)) continue;
      var ar = Math.abs(rings[j].a2);
      if (ar < bestA) { bestA = ar; best = j; }
    }
    parent.push(best);
  }
  for (k = 0; k < rings.length; k++) {
    var dd = 0, up = parent[k], g2 = 0;
    while (up >= 0) { dd++; up = parent[up]; if (++g2 > rings.length) return cpRevert('环嵌套成环'); }
    depth.push(dd);
    if ((dd % 2 === 0) !== (rings[k].a2 > 0)) {      return cpRevert(dd % 2 === 0 ? '外环方向不是 CCW' : '洞方向不是 CW');
    }
  }
  // ⑤ 面积守恒基准 = 簇内三角形在平面内的**有向**面积之和（覆盖量的如实值，重叠/裂缝都会偏离）
  var aBefore = 0;
  for (k = 0; k < cur.length; k++) {
    var Ft = cur[k];
    var b0 = to2(Ft[0]), b1 = to2(Ft[1]), b2 = to2(Ft[2]);
    aBefore += ((b1[0] - b0[0]) * (b2[1] - b0[1]) - (b1[1] - b0[1]) * (b2[0] - b0[0])) / 2;
  }
  // ⑥ 每个外环 + 其直接子环（洞）→ 带孔三角化（索引口径：外环各点 + 依次各洞各点）
  var out = [];
  for (k = 0; k < rings.length; k++) {
    if (depth[k] % 2 !== 0) continue;
    var holesIdx = [], holesPts = [], z, h;
    for (j = 0; j < rings.length; j++) {
      if (parent[j] !== k) continue;
      var hp = [];
      for (z = 0; z < rings[j].vs.length; z++) hp.push(to2(rings[j].vs[z]));
      holesIdx.push(rings[j].vs);
      holesPts.push(hp);
    }
    var idxMap = rings[k].vs.slice();
    for (h = 0; h < holesIdx.length; h++) {
      for (z = 0; z < holesIdx[h].length; z++) idxMap.push(holesIdx[h][z]);
    }
    var loc = null;
    // ★ 共面轮廓收尾走**旧桥接+耳切**实现：布尔路径的面数/M4 对三角化切法敏感，
    //   换成 earcut 会让 polyNodes/polyInput 暴涨、M4 真缺口变多（见 triangulatePolygon 处实测）。
    try { loc = triangulatePolygonWithHolesLegacy(ring2[k], holesPts); } catch (e) { return cpRevert('带孔三角化抛错'); }
    var got = 0;
    for (j = 0; j < loc.length; j++) {
      var t3 = loc[j];
      var I = idxMap[t3[0]], J = idxMap[t3[1]], K = idxMap[t3[2]];
      if (I === undefined || J === undefined || K === undefined) return cpRevert('三角化输出索引越界');
      if (I === J || J === K || I === K) continue;
      var c0 = to2(I), c1 = to2(J), c2 = to2(K);
      var ar3 = ((c1[0] - c0[0]) * (c2[1] - c0[1]) - (c1[1] - c0[1]) * (c2[0] - c0[0])) / 2;
      if (!(ar3 > 0)) return cpRevert('三角化输出反向三角形');                        // 反向/退化 ⇒ 回退（绝不静默产出坏面）
      out.push([I, J, K]);
      got++;
    }
    if (!got) return cpRevert('三角化输出为空');
  }
  // ⑦ 双重守恒校验 + 面数确实减少，否则不值得替换
  //    ① 强：三角化输出必须**恰好铺满环区域**（同一次参数化下计算 ⇒ 只剩浮点误差）；
  //    ② 弱：环区域与原三角形覆盖一致 —— T 缝细分会把"距边 ≤ eps 的近似点"按真实位置插入，
  //       带来 O(eps·L) 量级的面积漂移（eps 是吸附格距，远小于几何尺度），用 1e-6 相对容差记账；
  //       超出即回退（说明原覆盖本就不自洽：有重叠或真缺口）。
  var aAfter = 0, aRings = 0;
  for (k = 0; k < out.length; k++) {
    var d0 = to2(out[k][0]), d1 = to2(out[k][1]), d2 = to2(out[k][2]);
    aAfter += ((d1[0] - d0[0]) * (d2[1] - d0[1]) - (d1[1] - d0[1]) * (d2[0] - d0[0])) / 2;
  }
  for (k = 0; k < rings.length; k++) aRings += rings[k].a2 / 2;
  if (Math.abs(aAfter - aRings) > Math.max(1, Math.abs(aRings) * 1e-9)) {
    return cpRevert('三角化未铺满环区域（面积不符）');
  }
  if (Math.abs(aRings - aBefore) > Math.max(1, Math.abs(aBefore) * 1e-5)) {
    return cpRevert('环面积与原覆盖不符（细分漂移过大）');
  }
  // 输出三角形**不得重叠**：每条有向边 (a,b) 与 (b,a) 的出现次数之和至多 2，且为 2 时必须
  // 一正一反。同向重复 = 同一区域被覆盖两次 ⇒ 会制造非流形边（实测单孔场景能造出 4 条）。
  var oc = {}, ekey2;
  for (k = 0; k < out.length; k++) {
    for (j = 0; j < 3; j++) {
      var ok1 = out[k][j] + ':' + out[k][(j + 1) % 3];
      oc[ok1] = (oc[ok1] || 0) + 1;
    }
  }
  for (ekey2 in oc) {
    var pq = ekey2.split(':');
    var fwd = oc[ekey2], bwd = oc[pq[1] + ':' + pq[0]] || 0;
    if (fwd + bwd > 2 || (fwd + bwd === 2 && (fwd !== 1 || bwd !== 1))) {
      return cpRevert('输出存在重叠/同向重复边（会产生非流形）');
    }
  }
  if (out.length >= cl.length) return cpRevert('面数未减少');
  return out;
}

function meshToTris(m) {
  var out = [];
  for (var k = 0; k < m.indices.length; k++) {
    var f = m.indices[k];
    out.push([m.positions[f[0]].slice(), m.positions[f[1]].slice(), m.positions[f[2]].slice()]);
  }
  return out;
}

// 三角形集合 → 索引网格（焊接 + 去退化 + 去孤立顶点）
function trisToMesh(tris, tol) {
  var positions = [], indices = [], k, m;
  for (k = 0; k < tris.length; k++) {
    var base = positions.length;
    positions.push(tris[k][0].slice(), tris[k][1].slice(), tris[k][2].slice());
    indices.push([base, base + 1, base + 2]);
  }
  var welded = meshWeld(meshNew(positions, indices), tol === undefined ? WELD_TOL : tol).mesh;
  return meshCompact(meshDropDegenerate(welded));
}

// 空间分辨率（对齐 JSCAD measureEpsilon / maths.EPS 的口径）：eps = EPS · 平均尺寸。
//   ★ 这是本插件"水密性"问题的根因所在：BSP 裁剪的交点由**各三角形各自插值**，
//     同一几何点在不同三角形里的坐标会有 ~1e-4 级的差异 —— 既不严格相等，也无法用
//     任意小的容差合并（容差小了合不上、大了会把真实几何合掉，实测都会让面积失真）。
//     唯一可靠的做法与 JSCAD generalize({snap:true}) 一致：把顶点**吸附到统一网格**，
//     吸附后近似重合的点变成完全相同的坐标，边才可能真正配对。
function meshEpsilon(m) {
  if (!m.positions.length) return EPS_SPATIAL;
  var bb = meshBbox(m);
  var total = (bb[1][0] - bb[0][0]) + (bb[1][1] - bb[0][1]) + (bb[1][2] - bb[0][2]);
  return Math.max(EPS_SPATIAL, EPS_SPATIAL * total / 3);
}

// 顶点吸附（照 JSCAD vec3.snap 语义：round(v/eps)·eps，"+0" 消除 -0）
function meshSnap(m, eps) {
  var pos = [], idx = [], k;
  for (k = 0; k < m.positions.length; k++) {
    var p = m.positions[k];
    pos.push([
      Math.round(p[0] / eps) * eps + 0,
      Math.round(p[1] / eps) * eps + 0,
      Math.round(p[2] / eps) * eps + 0
    ]);
  }
  for (k = 0; k < m.indices.length; k++) idx.push(m.indices[k].slice());
  return meshNew(pos, idx);
}

// 焊接/缝合容差统一取自空间分辨率（焊接取半格：吸附后同格点应严格重合）
function weldTolerance(m) { return meshEpsilon(m) * 0.5; }
function tjTolerance(m) { return meshEpsilon(m); }

// ── 布尔运算默认入口（union / subtract / intersect），输入输出都是索引网格 ──────
//   默认走 **多边形 BSP 路径**（I.6-2）：BSP 全程保多边形，只在节点收尾时三角化。
//   为什么默认切它（实测 2026-09-17，60×40×6 板 @24 段；详见 §I.6-2 与 meshBooleanPoly 注释）：
//     · 面数 −65~81%、边界边 −36~56%（4 孔：2975→768 面、1451→928 边界边）；
//     · 更快 3~6×（板 − 24 孔：9962 面/350ms → 2887 面/59ms）；
//     · 与三角路径几何守恒（体积/面积互差 < 1e-6）、非流形 0、反向边对 0、两次运行逐位一致；
//     · 边界场景（不相交 / 完全包含 / 完全重合 / 同尺寸 intersect）两路径结果逐位一致。
//     · 输出已是**合并饱和**态：再跑 meshMergeCoplanar 仅 ∩ 场景 −6 面（1%）、其余 merged=0。
//   回退：setMeshBooleanLegacy(true)，或插件 config 给 legacyMeshBoolean=true → 走 meshBooleanTrig。
//   ★ 已知差异（仅 1 处，判为可接受）：两实体**仅共面接触**时 poly 50 面/22 边界边 vs 三角 46 面/18 边界边
//     （面 +4、边界 +4）；两者都不水密（退化输入），体积一致、非流形 0。
//   statsOut 可选：传入则回填所走路径的诊断，并置 statsOut.path = 'poly' | 'trig' 便于区分。
var meshBoolLegacy = false;
function setMeshBooleanLegacy(on) { meshBoolLegacy = !!on; return meshBoolLegacy; }
function meshBooleanPath() { return meshBoolLegacy ? 'trig' : 'poly'; }

// 多边形路径的"量化顶点表"：polyNodeToTris 在各平面节点间共享它，
// 使同一几何顶点在板面/孔壁/侧面获得**同一个代表坐标**（详见 polyNodeToTris 注释）。
// 每次布尔运行开始前清空（生命周期 = 一次 meshBooleanPoly）。
var POLY_QUANT_COORD = null;
var POLY_QUANT_KEYS = [];        // 全局量化键列表（供环边跨节点对齐用）
function polyQuantBegin() { POLY_QUANT_COORD = {}; POLY_QUANT_KEYS = []; }

// 收集所有节点的全部多边形顶点到全局量化表（key 唯一）。供 polyNodeToTris 做环边对齐。
//   eps 必须与 polyNodeToTris 使用的量化格一致（调用方传入同一个 TOL），否则 key 对不上。
function polyQuantCollect(nodes, eps) {
  var inv = 1 / eps;
  for (var k = 0; k < nodes.length; k++) {
    var ps = nodes[k].polys;
    for (var j = 0; j < ps.length; j++) for (var i = 0; i < ps[j].length; i++) {
      var p = ps[j][i];
      var key = Math.round(p[0] * inv) + '|' + Math.round(p[1] * inv) + '|' + Math.round(p[2] * inv);
      if (!POLY_QUANT_COORD[key]) { POLY_QUANT_COORD[key] = p; POLY_QUANT_KEYS.push(key); }
    }
  }
  return POLY_QUANT_KEYS.length;
}

function meshBoolean(meshA, meshB, op, tol, statsOut) {
  if (statsOut) statsOut.path = meshBooleanPath();
  if (meshBoolLegacy) return meshBooleanTrig(meshA, meshB, op, tol, statsOut);
  return meshBooleanPoly(meshA, meshB, op, tol, statsOut);
}

// 三角路径布尔（I.6-1 及以前的生产路径）：BSP 只吃三角形 ⇒ 共面合并只能**事后按簇聚类**，回退严重
//   （4 孔场景整簇回退，面数停在 2975）。保留为 **meshBoolean 的 legacy 分支 + 测试基线**，不在默认链路上。
function meshBooleanTrig(meshA, meshB, op, tol, statsOut) {
  var repaired = meshBooleanRaw(meshA, meshB, op, tol);
  var mg = meshMergeCoplanar(repaired);
  // ★ 合并会**自己引入 T 缝**：它按簇重三角化，新边与邻居的分段方式可能又对不上
  //   （实测 60×40×6 板 − 柱：合并前 boundary 10 → 合并后 26）。这些缝是"合并的产物"，
  //   与本路径无关，但会留在交付网格里 ⇒ 合并后再走一次 meshRepair（含 T 缝消解）。
  //   代价：面数回到与"未合并的修复结果"相近的量级（消解会补回部分面）。
  var fixed = meshRepair(mg.mesh).mesh;
  if (statsOut) {
    statsOut.before = repaired.indices.length;
    statsOut.after = fixed.indices.length;
    statsOut.merge = mg.stats;
    statsOut.mergeRaw = mg.mesh.indices.length;
  }
  return fixed;
}

// ★ BSP 平面判定容差的动态设定（实测结论，2026-09-17；M4 真缺口的根因修复）
//   EPS_PLANE 是"把顶点归类为共面"的阈值，它必须与本次输入的空间分辨率同量级；否则会把
//   "近共面"误判成"跨平面"：同一个几何平面被拆进多个 BSP 节点，切割平面因此带有微小倾斜，
//   在离该多边形很远处切出 1e-2 量级的坐标偏差 ⇒ 顶点 z 层爆炸、反向半边找不到配对（M4 真缺口）。
//   实测（60×40×6 板 − 4×∅6 柱 @24 段，4 次串行 subtract；面积以 24 边形孔真值校核）：
//     EPS_PLANE=1e-6（原值）  顶点 z 层 39、面 2216、边界边 1114、非流形 0、真缺口 893
//     EPS_PLANE=2×eps        顶点 z 层  2、面 2099、边界边 1021、非流形 0、真缺口 817  ← 全面更优
//   取 2×meshEpsilon：meshEpsilon 就是 eps 网格的"一格"，平面判定比一格还窄必然来回翻面。
//   返回旧值供 finally 恢复 —— 布尔运行期的放大容差不得泄漏给 chamfer/圆角等其它几何运算。
function bspEnterPlaneEps(meshA, meshB) {
  var saved = EPS_PLANE;
  var e = 2 * meshEpsilon(meshMerge([meshA, meshB]));
  if (e > EPS_PLANE) EPS_PLANE = e;
  return saved;
}

// BSP 布尔的**原始输出**（只做 meshRepair、不做共面合并）—— 共面合并的回退基线/诊断对照组
function meshBooleanRaw(meshA, meshB, op, tol) {
  if (!(meshA.indices.length > 0) || !(meshB.indices.length > 0)) fail('布尔运算的两个实体都不能为空');
  if (meshA.indices.length > LIMIT_BOOL_INPUT || meshB.indices.length > LIMIT_BOOL_INPUT) {
    fail('布尔运算输入过大（' + meshA.indices.length + ' + ' + meshB.indices.length + ' 面，单次上' + LIMIT_BOOL_INPUT + '）——请降低 segments 或拆分部件');
  }
  var TOL = tol === undefined ? weldTolerance(meshMerge([meshA, meshB])) : tol;
  var savedPlaneEps = bspEnterPlaneEps(meshA, meshB);
  try {
  var A = bspNew(), B = bspNew();
  bspBuild(A, meshToTris(meshA));
  bspBuild(B, meshToTris(meshB));
  if (op === 'union') {
    bspClipTo(A, B); bspClipTo(B, A);
    bspInvert(B); bspClipTo(B, A); bspInvert(B);
    bspBuild(A, bspAllTris(B));
  } else if (op === 'subtract') {
    bspInvert(A); bspClipTo(A, B); bspClipTo(B, A);
    bspInvert(B); bspClipTo(B, A); bspInvert(B);
    bspBuild(A, bspAllTris(B)); bspInvert(A);
  } else if (op === 'intersect') {
    bspInvert(A); bspClipTo(B, A); bspInvert(B); bspClipTo(A, B); bspClipTo(B, A);
    bspBuild(A, bspAllTris(B)); bspInvert(A);
  } else {
    fail('未知布尔运算 ' + op + '（可用 union / subtract / intersect）');
  }
  var out = trisToMesh(bspAllTris(A), TOL);
  if (!out.indices.length) fail('布尔运算结果为空（' + op + '）——检查两个实体是否有交集/是否完全包含');
  // BSP 裁剪必然产生 T 型接缝 → 统一修复（水密性是可打印的前提，也是 M 判据的验收面）
  // 共面合并（I.6-1）在 meshBooleanTrig 里对修复结果做，本函数不含它。
  return meshRepair(out).mesh;
  } finally { EPS_PLANE = savedPlaneEps; }
}

// ── I.6-2：节点内共面多边形合并 → 带孔轮廓 → 三角化 ──────────────
// 合并本身只有两步（这正是多边形表示的优势；I.6-1 在三角域要 T 缝细分 / 噪声清理 / 容差匹配）：
//   ① 有向边对消：内部边（正反各出现一次）删掉；
//   ② 剩余边串环：顶点出度必须 ≤ 1（否则边界自接触 ⇒ 回退），环按 2D 有向面积分正（外环）/负（洞）。
// 顶点用 eps 网格量化键比对——同一次切割的坐标在数学上严格相同，量化只为吸收 1ulp 级抖动。
// 返回坐标三元组数组；任何一步不成立都 cpRevert（返回 null），由调用方退化为逐多边形扇形三角化。
function polyNodeToTris(node, eps) {
  var polys = node.polys, k, j;
  if (!polys.length) return [];
  var pl = node.plane || polyPlane(polys[0]);
  if (!pl) return [];
  // ★ 量化坐标表**跨节点共享**（每次布尔运行开始时清空，见 polyQuantBegin）：
  //   板面节点与孔壁节点里"同一个几何顶点"必须用同一个代表坐标，否则两侧环各取各的代表，
  //   边界错开 ~1 格 ⇒ 反向半边找不到配对（M4 真缺口）。实测（60×40×6 板 − 4 柱 @24 段）：
  //   按节点独立 → 边界边 1021 / 真缺口 817；跨节点共享 → 边界边 993 / 真缺口 791。
  var inv = 1 / eps, coord = POLY_QUANT_COORD || (POLY_QUANT_COORD = {});
  function K(p) {                                  // 顶点量化键 → 代表坐标（首次出现者）
    var key = Math.round(p[0] * inv) + '|' + Math.round(p[1] * inv) + '|' + Math.round(p[2] * inv);
    if (!coord[key]) coord[key] = p;
    return key;
  }
  // ① 有向边收集（同向重复 ⇒ 非流形 ⇒ 回退）
  var dir = {}, order = [];
  for (k = 0; k < polys.length; k++) {
    var R = polys[k], q = [];
    for (j = 0; j < R.length; j++) q.push(K(R[j]));
    for (j = 0; j < q.length; j++) {
      if (q[j] === q[(j + 1) % q.length]) return cpRevert('多边形 BSP 输出存在重复顶点（量化后重合）');
    }
    for (j = 0; j < q.length; j++) {
      var ek = q[j] + '>' + q[(j + 1) % q.length];
      if (dir[ek]) return cpRevert('多边形 BSP 输出存在同向重复边（非流形）');
      dir[ek] = 1; order.push(ek);
    }
  }
  // ② 对消内部边，顺带建立「顶点 → 下一条边」（出度 > 1 = 边界自接触 ⇒ 回退）
  var next = {}, keep = 0;
  for (k = 0; k < order.length; k++) {
    var e2 = order[k], sp = e2.indexOf('>');
    var a = e2.slice(0, sp), b = e2.slice(sp + 1);
    if (dir[b + '>' + a]) continue;                 // 有反向边 ⇒ 内部边，对消
    if (next[a] !== undefined) return cpRevert('多边形 BSP 输出边界在顶点处分支（无法串环）');
    next[a] = b; keep++;
  }
  if (!keep) return cpRevert('多边形 BSP 输出无边界环（多边形被完全对消）');
  // ③ 串环（沿 next 前进，必须正好回到起点）
  var seen = {}, rings = [], starts = [], sk;
  for (sk in next) starts.push(sk);
  for (k = 0; k < starts.length; k++) {
    if (seen[starts[k]]) continue;
    var ring = [], cur = starts[k], guard = 0;
    while (cur !== undefined && !seen[cur]) {
      if (++guard > keep + 8) return cpRevert('多边形 BSP 输出串环失败（环长度异常）');
      seen[cur] = 1; ring.push(cur); cur = next[cur];
    }
    if (cur !== starts[k]) return cpRevert('多边形 BSP 输出边界自接触（环未闭合回起点）');
    if (ring.length >= 3) rings.push(ring);
  }
  // ③′ 环边跨节点对齐（M4 真缺口收尾修复）：把"落在环边内部"的其他节点顶点插进环里。
  //   ★ 时机是关键：必须放在 ② 有向边对消**之后**。对消的判据是"反向边严格存在"，
  //     若在对消前插点会破坏对消 —— 实测（在多边形上插）引入 10 条非流形边、面数 +50%。
  //     插在环上时对消已完成，只延长顶点序列，不改变任何既有拓扑关系。
  //     相邻节点（T 缝的两侧）读同一张全局量化表，故两侧插点集合一致 ⇒ 三角化后边能配对。
  if (POLY_QUANT_KEYS.length) {
    for (k = 0; k < rings.length; k++) {
      var rg = rings[k], nrg = rg.length, grown = [], mm;
      for (j = 0; j < nrg; j++) {
        var ka2 = rg[j], kb2 = rg[(j + 1) % nrg];
        grown.push(ka2);
        var pa2 = coord[ka2], pb2 = coord[kb2];
        var eab = [pb2[0] - pa2[0], pb2[1] - pa2[1], pb2[2] - pa2[2]];
        var eL2 = eab[0] * eab[0] + eab[1] * eab[1] + eab[2] * eab[2];
        if (eL2 <= 4 * eps * eps) continue;
        var midk = [];
        for (mm = 0; mm < POLY_QUANT_KEYS.length; mm++) {
          var kq = POLY_QUANT_KEYS[mm];
          if (kq === ka2 || kq === kb2) continue;
          var pq = coord[kq];
          if (!pq) continue;
          if (Math.abs(vDot(pl.n, pq) - pl.w) > eps) continue;   // 平面过滤：T 缝两侧必共面
          var tq = ((pq[0] - pa2[0]) * eab[0] + (pq[1] - pa2[1]) * eab[1] + (pq[2] - pa2[2]) * eab[2]) / eL2;
          if (tq <= 0 || tq >= 1) continue;
          var qx = pa2[0] + eab[0] * tq - pq[0], qy = pa2[1] + eab[1] * tq - pq[1], qz = pa2[2] + eab[2] * tq - pq[2];
          if (qx * qx + qy * qy + qz * qz > 4 * eps * eps) continue;
          midk.push([tq, kq]);
        }
        if (midk.length) {
          midk.sort(function (u, v) { return u[0] - v[0]; });
          for (mm = 0; mm < midk.length; mm++) grown.push(midk[mm][1]);
        }
      }
      var cleaned = [];
      for (j = 0; j < grown.length; j++) {
        if (j && grown[j] === grown[j - 1]) continue;
        cleaned.push(grown[j]);
      }
      if (cleaned.length >= 3 && cleaned[0] === cleaned[cleaned.length - 1]) cleaned.pop();
      if (cleaned.length >= 3) rings[k] = cleaned;
    }
  }
  // ④ 平面内正交基（右手：u × v = n ⇒ 2D 绕向与 3D 绕向一致，面积符号可直接用来分内外环）
  var nrm = pl.n;
  var ax = (Math.abs(nrm[0]) <= Math.abs(nrm[1]) && Math.abs(nrm[0]) <= Math.abs(nrm[2])) ? [1, 0, 0]
         : (Math.abs(nrm[1]) <= Math.abs(nrm[2]) ? [0, 1, 0] : [0, 0, 1]);
  var uu = vNorm(vCross(ax, nrm)), vv2 = vCross(nrm, uu);
  function to2(key) {
    var p = coord[key];
    return [vDot(uu, p) * inv, vDot(vv2, p) * inv];   // 单位 = eps 格 ⇒ 面积阈值与几何尺度无关
  }
  var outers = [], holeRings = [];
  for (k = 0; k < rings.length; k++) {
    var pts2 = [];
    for (j = 0; j < rings[k].length; j++) pts2.push(to2(rings[k][j]));
    var ar = polyArea(pts2);
    if (Math.abs(ar) <= 4) continue;                  // 零面积 / 微碎片（≤ 4 格²）
    if (ar > 0) outers.push({ ring: rings[k], pts: pts2, area: ar, holes: [] });
    else holeRings.push({ ring: rings[k], pts: pts2 });
  }
  if (!outers.length) return cpRevert('多边形 BSP 输出无有效外环');
  // ⑤ 洞归属：落在哪个外环内（多个包含时取面积最小者 —— 嵌套时属于最内层）
  for (k = 0; k < holeRings.length; k++) {
    var best = -1, bestA = Infinity;
    for (j = 0; j < outers.length; j++) {
      if (outers[j].area < bestA && pointInPoly(holeRings[k].pts[0], outers[j].pts)) { bestA = outers[j].area; best = j; }
    }
    if (best < 0) return cpRevert('共面合并的洞环不在任何外环内（环嵌套异常）');
    outers[best].holes.push(holeRings[k]);
  }
  // ⑥ 带孔三角化（复用 I.7 的桥接 + 耳切 —— 合并后可能出现凹轮廓，扇形三角化在此不适用）
  var out = [];
  for (k = 0; k < outers.length; k++) {
    var O = outers[k], pts3 = [], hs = [], hp, m;
    for (j = 0; j < O.ring.length; j++) pts3.push(coord[O.ring[j]]);
    for (j = 0; j < O.holes.length; j++) {
      hp = [];
      for (m = 0; m < O.holes[j].ring.length; m++) {
        pts3.push(coord[O.holes[j].ring[m]]);
        hp.push(O.holes[j].pts[m]);
      }
      hs.push(hp);                                    // 洞环已是 CW（面积负）⇒ 与桥接方向约定一致
    }
    var local;
    // ★ 同上：I.6-2 共面轮廓收尾用旧实现，保证布尔路径零回归。
    try { local = triangulatePolygonWithHolesLegacy(O.pts, hs); }
    catch (e) { return cpRevert('共面轮廓三角化失败：' + ((e && e.message) ? e.message : e)); }
    for (j = 0; j < local.length; j++) {
      out.push([pts3[local[j][0]], pts3[local[j][1]], pts3[local[j][2]]]);
    }
  }
  return out;
}

// I.6-2 多边形路径布尔 —— **meshBoolean 的默认路径**（三角版保留为 meshBooleanTrig 供回退/对照）
//   收尾：按节点做共面合并 + 三角化。任一节点合并失败 ⇒ 该节点退化「逐多边形扇形三角化」
//   （凸性保证正确），因此**多边形路径不存在产出坏几何的风险面**，最差只是少合并一些面。
//   statsOut 回填：polyNodes/polyInput/polyReverted + beforeRepair/afterRepair。
//   ★ 实测（2026-09-17；60×40×6 板 − ∅6 柱 @24 段，4/8 孔为 4/8 个该柱）：
//       场景          三角路径 面/边界      多边形路径 面/边界     面数降幅
//       板 − 1 孔          771 / 317           222 / 214          71.2%
//       板 − 4 孔         2975 / 1451          768 / 928          74.2%   ← I.6-1 只能到 2975 且整簇回退
//       板 − 8 孔         7392 / 3226         1405 / 1787         81.0%
//       板 ∪ 4 柱         3300 / 1474         1148 / 952          65.2%
//       板 ∩ 4 柱         1100 / 1108          600 / 590          45.5%
//     两路径体积/面积互差 < 1e-6（最大 1.09e-6），非流形边 0、反向边对 0、两次运行逐位一致。
//     回退节点 2/106，原因固定为「边界在顶点处分支」—— 即共面多边形集合在某顶点处出度 > 1
//     （多边形只在该点汇聚），当前直接回退；补「最小左转」串环即可吃掉（面数还能再降一点）。
//   ★ 已知遗留（非本路径引入）：subtract 挖孔体积相对解析正多边形真值偏大约 0.85 mm³（≈0.5%），
//     而 intersect 与解析值完全吻合。该偏差在本次改动前即存在（三角路径数值一字未变），
//     属既有 BSP/meshRepair 行为，需另行排查（不水密网格的体积积分本身就有此量级误差）。
function meshBooleanPoly(meshA, meshB, op, tol, statsOut) {
  if (!(meshA.indices.length > 0) || !(meshB.indices.length > 0)) fail('布尔运算的两个实体都不能为空');
  if (meshA.indices.length > LIMIT_BOOL_INPUT || meshB.indices.length > LIMIT_BOOL_INPUT) {
    fail('布尔运算输入过大（' + meshA.indices.length + ' + ' + meshB.indices.length + ' 面，单次上' + LIMIT_BOOL_INPUT + '）——请降低 segments 或拆分部件');
  }
  var TOL = tol === undefined ? weldTolerance(meshMerge([meshA, meshB])) : tol;
  // 平面判定容差按输入尺度放大（见 bspEnterPlaneEps）；以下 BSP 段整体处于 try 内，
  // 保证任何出口（含 fail 抛错）都能恢复 EPS_PLANE。缩进未重排以保持改动最小。
  var savedPlaneEps = bspEnterPlaneEps(meshA, meshB);
  try {
  polyQuantBegin();                                 // 量化坐标表：本次布尔运行内跨节点共享
  var A = polyBspNew(), B = polyBspNew();
  polyBspBuild(A, meshToPolys(meshA));
  polyBspBuild(B, meshToPolys(meshB));
  if (op === 'union') {
    polyBspClipTo(A, B); polyBspClipTo(B, A);
    polyBspInvert(B); polyBspClipTo(B, A); polyBspInvert(B);
    polyBspBuild(A, polyBspAllPolys(B));
  } else if (op === 'subtract') {
    polyBspInvert(A); polyBspClipTo(A, B); polyBspClipTo(B, A);
    polyBspInvert(B); polyBspClipTo(B, A); polyBspInvert(B);
    polyBspBuild(A, polyBspAllPolys(B)); polyBspInvert(A);
  } else if (op === 'intersect') {
    polyBspInvert(A); polyBspClipTo(B, A); polyBspInvert(B); polyBspClipTo(A, B); polyBspClipTo(B, A);
    polyBspBuild(A, polyBspAllPolys(B)); polyBspInvert(A);
  } else {
    fail('未知布尔运算 ' + op + '（可用 union / subtract / intersect）');
  }
  var nodes = polyBspNodes(A), tris = [], npoly = 0, reverted = 0, ni, ti, pi, fi;
  // 收集全局量化顶点表（供 polyNodeToTris 做环边跨节点对齐；见其中 ③′ 步骤）
  var qkeys = polyQuantCollect(nodes, TOL);
  for (ni = 0; ni < nodes.length; ni++) {
    npoly += nodes[ni].polys.length;
    var t = polyNodeToTris(nodes[ni], TOL);
    if (!t) {                                        // 回退：逐多边形扇形三角化（凸 ⇒ 正确）
      reverted++;
      for (pi = 0; pi < nodes[ni].polys.length; pi++) {
        var ft = fanTriangles(nodes[ni].polys[pi]);
        for (fi = 0; fi < ft.length; fi++) tris.push(ft[fi]);
      }
      continue;
    }
    for (ti = 0; ti < t.length; ti++) tris.push(t[ti]);
  }
  var out = trisToMesh(tris, TOL);
  if (!out.indices.length) fail('布尔运算结果为空（' + op + '）——检查两个实体是否有交集/是否完全包含');
  var rep = meshRepair(out);
  if (statsOut) {
    statsOut.polyNodes = nodes.length;
    statsOut.polyInput = npoly;
    statsOut.polyReverted = reverted;
    statsOut.beforeRepair = out.indices.length;
    statsOut.afterRepair = rep.mesh.indices.length;
    statsOut.polyQuantKeys = qkeys;
  }
  return rep.mesh;
  } finally { EPS_PLANE = savedPlaneEps; }
}

// ── 倒角 / 圆角（网格级边重建：I.8.5 路线 A/B）────────────────
// 口径与理由：
//   · 圆角/倒角不是参数化特征，而是**网格级边重建** —— 这里实现可解析验证的那一半：
//       ① 边识别：焊接顶点 → 每条边的两个邻面 → 外法线夹角（折角）→ 凸边 / 凹边；
//       ② 倒角：每条凸边一个**斜面半空间**，结果 = 实体 ∩ 所有斜面内半空间。
//          半空间集合本身是凸的 ⇒ 用 H-rep → V-rep 直接构造裁剪体（精确、不靠布尔求交），
//          再由**一次**布尔得到结果（避免逐边布尔造成的面数爆炸）；
//       ③ 圆角：先按半径 r 倒角切掉棱部楔块，再把「圆柱片段」拼回去（路线 B 的布尔近似）。
//   · 斜面平移量为什么是 d·cos(θ/2)：要求斜面恰好经过「两个邻面内距棱 d」的那两点 ——
//     d 就是用户直觉的"倒角量"（面内距离），θ 是内二面角（凸边 θ < 180°）。
//   · 顶点处（三条以上棱交会）：chamfer 由半空间求交自然得到正确的小平面；圆角的顶点混合见文档 I.9.6
//     —— 凸体全棱走 rolling-ball（顶点处是解析球片），不再是"不做"。
//   · 凹边（内二面角 > 180°）不支持：凹边倒角要「填」材料而非「切」（另一套 union 逻辑）——
//     默认跳过并在诊断里报告条数，用户显式指定凹边时明确报错。

// 三角形里"不在边 a-b 上"的那个顶点
function triThirdVertex(tri, a, b) {
  for (var k = 0; k < 3; k++) { if (tri[k] !== a && tri[k] !== b) return tri[k]; }
  return null;
}

function fmtVec3(v) { return '(' + fmtNum(v[0], 2) + ',' + fmtNum(v[1], 2) + ',' + fmtNum(v[2], 2) + ')'; }

function meshDiagonal(m) {
  var bb = meshBbox(m);
  return vLen(vSub(bb[1], bb[0]));
}

// 点到线段的最短距离
function distPointSeg(p, a, b) {
  var ab = vSub(b, a), L2 = vDot(ab, ab);
  if (L2 <= EPS) return vDist(p, a);
  var t = vDot(vSub(p, a), ab) / L2;
  if (t < 0) t = 0; else if (t > 1) t = 1;
  return vDist(p, [a[0] + ab[0] * t, a[1] + ab[1] * t, a[2] + ab[2] * t]);
}

// 边表：每条内部边的两个邻面、外向法线、折角与凸凹。
//   ★ 凸凹判据（对任意闭合网格成立）：取本面「不在边上」的第三顶点 c，
//     看它相对**另一个邻面**平面的符号（面法线外向为正）：在内侧 ⇒ 凸边；在外侧 ⇒ 凹边。
//     两个邻面各算一次互为印证（网格已外向化；不一致时取绝对值大者以抗退化面）。
function meshEdgeTable(m, weldTol) {
  var welded = meshWeld(m, weldTol === undefined ? WELD_TOL : weldTol).mesh;
  var pos = welded.positions, tris = welded.indices, i, e, k;
  var nrm = [];
  for (i = 0; i < tris.length; i++) {
    var t3 = tris[i];   // ★ 注意：triPlane 收的是**坐标**三元组，不是索引三元组
    var tpl = triPlane([pos[t3[0]], pos[t3[1]], pos[t3[2]]]);
    nrm.push(tpl ? tpl.n : null);
  }
  var map = {}, order = [];
  for (i = 0; i < tris.length; i++) {
    for (e = 0; e < 3; e++) {
      var a = tris[i][e], b = tris[i][(e + 1) % 3];
      var lo = a < b ? a : b, hi = a < b ? b : a;
      var key = lo + '_' + hi;
      var rec = map[key];
      if (!rec) { rec = map[key] = { a: lo, b: hi, faces: [] }; order.push(key); }
      rec.faces.push(i);
    }
  }
  var out = [], skipped = { open: 0, nonManifold: 0, degenerate: 0 };
  for (k = 0; k < order.length; k++) {
    var r = map[order[k]];
    if (r.faces.length === 1) { skipped.open++; continue; }
    if (r.faces.length > 2) { skipped.nonManifold++; continue; }
    var f1 = r.faces[0], f2 = r.faces[1];
    var n1 = nrm[f1], n2 = nrm[f2];
    var c1 = triThirdVertex(tris[f1], r.a, r.b), c2 = triThirdVertex(tris[f2], r.a, r.b);
    if (!n1 || !n2 || c1 === null || c2 === null) { skipped.degenerate++; continue; }
    var pa = pos[r.a], pb = pos[r.b], len = vDist(pa, pb);
    if (len <= EPS) { skipped.degenerate++; continue; }
    var s1 = vDot(n2, vSub(pos[c1], pa));
    var s2 = vDot(n1, vSub(pos[c2], pa));
    var sgn = Math.abs(s1) >= Math.abs(s2) ? s1 : s2;
    var cosPhi = vDot(n1, n2);
    if (cosPhi > 1) cosPhi = 1; else if (cosPhi < -1) cosPhi = -1;
    var phi = Math.acos(cosPhi);
    // 凸凹判定要带相对容差：共面的相邻面（同一平面被切成多个三角形的对角线）sgn 理论上是 0，
    //   浮点误差会让它变成 +1e-16 —— 若严格按 sgn > 0 判凹，圆柱侧面的 6 条对角线会被误判成凹边，
    //   进而让 meshConvexPlanes 把**凸体**圆柱判成非凸（白走布尔路径，面数从百级涨到千级且不水密）。
    var convTol = Math.max(1e-12, 1e-9 * len);
    var convex = sgn <= convTol;
    out.push({
      a: r.a, b: r.b, pa: pa, pb: pb, len: len,
      dir: vMul(vSub(pb, pa), 1 / len),
      mid: [(pa[0] + pb[0]) / 2, (pa[1] + pb[1]) / 2, (pa[2] + pb[2]) / 2],
      n1: n1, n2: n2,
      convex: convex,
      angleDeg: phi * 180 / Math.PI,                                        // 折角（相邻面外法线夹角）：平面 = 0
      dihedralDeg: (convex ? Math.PI - phi : Math.PI + phi) * 180 / Math.PI  // 内二面角（凸 < 180，凹 > 180）
    });
  }
  return { mesh: welded, edges: out, skipped: skipped };
}

// 挑出要处理的棱：
//   edges 缺省 'convex'（折角 ≥ minAngle 的凸边）/ 坐标对数组 [[[x,y,z],[x,y,z]], …]
//   （按端点匹配，与顶点编号顺序无关 ⇒ 可复现、可精确指定单条棱）
function selectBevelEdges(tab, spec, E, what) {
  var minAngle = spec.minAngle === undefined ? EDGE_ANGLE_DEFAULT : evalNum(spec.minAngle, E, 'minAngle');
  if (!(minAngle >= 0) || minAngle > 179.9) fail(what + ' 的 minAngle 必须在 0~180 度之间（收到 ' + minAngle + '）');
  var want = spec.edges === undefined ? 'convex' : spec.edges;
  var tol = meshEpsilon(tab.mesh) * 2;
  var list = [], flat = 0, concave = 0, k, i, ed;
  if (isArr(want)) {
    var pairs = [], hitConcave = 0, hitFlat = 0;
    for (k = 0; k < want.length; k++) {
      var it = want[k];
      if (!isArr(it) || it.length !== 2) fail(what + ' 的 edges[] 必须是 [[x,y,z],[x,y,z]] 形式的坐标对（第 ' + k + ' 项不是）');
      pairs.push([evalVec3(it[0], E, 'edges[' + k + '][0]'), evalVec3(it[1], E, 'edges[' + k + '][1]')]);
    }
    if (!pairs.length) fail(what + ' 的 edges 数组不能为空');
    for (k = 0; k < tab.edges.length; k++) {
      ed = tab.edges[k];
      var hit = -1;
      for (i = 0; i < pairs.length; i++) {
        if ((vEq(ed.pa, pairs[i][0], tol) && vEq(ed.pb, pairs[i][1], tol)) ||
            (vEq(ed.pa, pairs[i][1], tol) && vEq(ed.pb, pairs[i][0], tol))) { hit = i; break; }
      }
      if (hit < 0) continue;
      pairs.splice(hit, 1);
      if (ed.angleDeg < minAngle - 1e-9) { hitFlat++; continue; }
      if (!ed.convex) { hitConcave++; continue; }
      list.push(ed);
    }
    if (hitConcave) fail(what + '：指定的 ' + hitConcave + ' 条边是**凹边**（内二面角 > 180°）—— 凹边的圆角/倒角要在凹处填材料，当前只支持凸边（切除）');
    if (hitFlat) fail(what + '：指定的 ' + hitFlat + ' 条边折角小于 minAngle=' + minAngle + '°（不属于棱；如确需处理请调小 minAngle）');
    if (pairs.length) fail(what + '：有 ' + pairs.length + ' 条指定边在网格里找不到（网格共 ' + tab.edges.length + ' 条内部边；端点须是焊接后的顶点坐标，可用 model_doc 的几何摘要核对）');
  } else {
    if (want !== 'convex') fail(what + ' 的 edges 只能是 "convex"（默认）或坐标对数组（收到 ' + JSON.stringify(want) + '；凹边未支持，故无 "all"）');
    for (k = 0; k < tab.edges.length; k++) {
      ed = tab.edges[k];
      if (ed.angleDeg < minAngle - 1e-9) { flat++; continue; }
      if (!ed.convex) { concave++; continue; }
      list.push(ed);
    }
  }
  if (!list.length) {
    fail(what + '：没有可处理的凸边（折角 ≥ ' + minAngle + '°；网格 ' + tab.edges.length + ' 条内部边，其中 ' + flat +
      ' 条折角过小、' + concave + ' 条为凹边）。曲面体（球/光滑回转体）属正常；要放宽请调小 minAngle');
  }
  if (list.length > LIMIT_EDGE_OPS) {
    fail(what + '：选中 ' + list.length + ' 条边，超过单次上限 ' + LIMIT_EDGE_OPS + ' 条（裁剪体顶点枚举 O(n³)）。' +
      '请用 edges 数组分批指定，或调大 minAngle 减少参与边数');
  }
  return { list: list, flat: flat, concave: concave, minAngle: minAngle };
}

// 一条凸边的倒角斜面（内半空间 n·x ≤ w）：
//   m = normalize(n1+n2)（两邻面外法线的角平分线），平移量 t = d·cos(θ/2)
//   ⇒ 斜面恰好经过「两个邻面内距棱 d」的点（d = 面内倒角量，θ = 内二面角）
function chamferPlaneOfEdge(edge, d) {
  var sum = vAdd(edge.n1, edge.n2), ln = vLen(sum);
  if (ln < 1e-9) return null;
  var m = vMul(sum, 1 / ln);
  var t = d * Math.cos(edge.dihedralDeg * Math.PI / 360);
  return { n: m, w: vDot(m, edge.mid) - t, t: t };
}

// 顶点 → 相邻面（边的邻域 BFS 用）
function vertexFaceAdjacency(nv, tris) {
  var adj = [], i, k, e;
  for (i = 0; i < nv; i++) adj.push([]);
  for (k = 0; k < tris.length; k++) for (e = 0; e < 3; e++) adj[tris[k][e]].push(k);
  return adj;
}

// 边的邻域顶点集（沿面邻接 BFS hops 跳）
function edgeNeighborhoodVertices(tris, adj, a, b, hops) {
  var vset = {}, seenF = {}, frontier = [a, b], h, k, kk, e;
  vset[a] = 1; vset[b] = 1;
  for (h = 0; h < hops; h++) {
    var next = [];
    for (k = 0; k < frontier.length; k++) {
      var faces = adj[frontier[k]];
      for (kk = 0; kk < faces.length; kk++) {
        var f = faces[kk];
        if (seenF[f]) continue;
        seenF[f] = 1;
        for (e = 0; e < 3; e++) {
          var v = tris[f][e];
          if (!vset[v]) { vset[v] = 1; next.push(v); }
        }
      }
    }
    frontier = next;
    if (!frontier.length) break;
  }
  return vset;
}

// 斜面是否只切到「这条边附近」—— 非凸实体（L 形、带孔件、镂空）的远处材料若落在切除侧，
//   说明这个无限延伸的半空间会误伤，必须拦下（否则静默产出错误几何）。两个判据：
//     ① 切除侧的顶点必须属于边的 N 跳邻域（"完全不相邻的特征被切"必然违规）；
//     ② 切除侧顶点到边的距离不超过 K = max(4d, 0.05·模型对角线)（邻域内但穿透薄壁等）。
function edgeCutSafety(tab, adj, edge, plane, d, diag) {
  var pos = tab.mesh.positions, i;
  var slack = Math.max(EPS_PLANE, diag * 1e-7);
  var neigh = adj ? edgeNeighborhoodVertices(tab.mesh.indices, adj, edge.a, edge.b, EDGE_NEIGHBOR_HOPS) : null;
  var farDist = 0, farAt = null;
  for (i = 0; i < pos.length; i++) {
    var v = pos[i];
    if (vDot(plane.n, v) <= plane.w + slack) continue;
    if (neigh && !neigh[i]) return { ok: false, reason: 'far', at: v, dist: distPointSeg(v, edge.pa, edge.pb) };
    var dist = distPointSeg(v, edge.pa, edge.pb);
    if (dist > farDist) { farDist = dist; farAt = v; }
  }
  var K = Math.max(4 * d, 0.05 * diag);
  if (farDist > K) return { ok: false, reason: 'dist', at: farAt, dist: farDist, limit: K };
  return { ok: true };
}

// 三平面交点（Cramer 法则）；近共面退化 ⇒ null
function planesIntersect(p1, p2, p3) {
  var bc = vCross(p2.n, p3.n);
  var den = vDot(p1.n, bc);
  if (Math.abs(den) < 1e-9) return null;
  var s = vAdd(vMul(bc, p1.w), vAdd(vMul(vCross(p3.n, p1.n), p2.w), vMul(vCross(p1.n, p2.n), p3.w)));
  var p = vMul(s, 1 / den);
  return vFinite(p) ? p : null;
}

// 半空间集合（n·x ≤ w，n 单位外向）→ 凸多面体网格：
//   顶点枚举（三平面交点 → 校验落在所有半空间内 → 去重）→ 逐平面收集共面顶点 → 面内角度排序 → 扇形三角化。
//   ★ 纯几何、不经布尔 ⇒ 对凸约束**解析精确**，且面数最少（无 BSP 碎片）—— 倒角的正确性正建立在此。
function halfspacesMesh(planes, eps) {
  var n = planes.length, i, j, k;
  if (n < 4) return null;
  var verts = [], seen = {};
  for (i = 0; i < n; i++) {
    for (j = i + 1; j < n; j++) {
      for (k = j + 1; k < n; k++) {
        var p = planesIntersect(planes[i], planes[j], planes[k]);
        if (!p) continue;
        var inside = true;
        for (var q = 0; q < n; q++) {
          if (vDot(planes[q].n, p) > planes[q].w + eps) { inside = false; break; }
        }
        if (!inside) continue;
        var key = Math.round(p[0] / eps) + '_' + Math.round(p[1] / eps) + '_' + Math.round(p[2] / eps);
        if (seen[key]) continue;
        seen[key] = 1;
        verts.push(p);
      }
    }
  }
  if (verts.length < 4) return null;
  var tris = [];
  for (i = 0; i < n; i++) {
    var nv = planes[i].n, w = planes[i].w, face = [];
    for (j = 0; j < verts.length; j++) if (Math.abs(vDot(nv, verts[j]) - w) <= eps) face.push(verts[j]);
    if (face.length < 3) continue;
    var helper = Math.abs(nv[0]) < 0.9 ? [1, 0, 0] : [0, 1, 0];
    var u = vNorm(vCross(nv, helper)), vv = vNorm(vCross(nv, u));
    var cx = 0, cy = 0, cz = 0;
    for (j = 0; j < face.length; j++) { cx += face[j][0]; cy += face[j][1]; cz += face[j][2]; }
    var ctr = [cx / face.length, cy / face.length, cz / face.length];
    var ring = [];
    for (j = 0; j < face.length; j++) {
      var rel = vSub(face[j], ctr);
      ring.push([Math.atan2(vDot(rel, vv), vDot(rel, u)), face[j]]);
    }
    ring.sort(function (A, B) { return A[0] - B[0]; });
    for (j = 1; j + 1 < ring.length; j++) tris.push([ring[0][1], ring[j][1], ring[j + 1][1]]);
  }
  if (!tris.length) return null;
  var mesh = trisToMesh(tris, eps);
  if (!mesh.indices.length) return null;
  return meshOrientOutward(mesh);
}

// 把模型包住的 6 个半空间（给无限半空间集合一个有限边界）
function boxHalfspaces(mesh, margin) {
  var bb = meshBbox(mesh);
  var lo = vSub(bb[0], [margin, margin, margin]), hi = vAdd(bb[1], [margin, margin, margin]);
  return [
    { n: [1, 0, 0], w: hi[0] }, { n: [-1, 0, 0], w: -lo[0] },
    { n: [0, 1, 0], w: hi[1] }, { n: [0, -1, 0], w: -lo[1] },
    { n: [0, 0, 1], w: hi[2] }, { n: [0, 0, -1], w: -lo[2] }
  ];
}

// 顶点枚举/共面判定容差（与模型尺度相关：数值误差 ~1e-13 相对，留 3 个量级余量）
function halfspaceEps(m) { return Math.max(1e-7, meshDiagonal(m) * 1e-9); }

function bboxOverlap(a, b) {
  for (var i = 0; i < 3; i++) if (a[1][i] < b[0][i] || b[1][i] < a[0][i]) return false;
  return true;
}

// 凸块分组：组内两两**包围盒不相交** ⇒ 可安全 meshMerge 成一个多体网格（免布尔）。
//   ★ 逐块布尔会让 BSP 把网格反复切碎（实测 12 个弧块逐块 union 把 12 面的立方体顶到 6 万面）；
//   分组后只需"组数"次布尔，组内直接拼接。包围盒相交但几何不相交的保守分到不同组（只多付一次布尔）。
function groupBlocksByBBox(blocks) {
  var groups = [], i, g, k;
  for (i = 0; i < blocks.length; i++) {
    var bb = meshBbox(blocks[i]);
    var placed = false;
    for (g = 0; g < groups.length && !placed; g++) {
      var ok = true;
      for (k = 0; k < groups[g].bbs.length; k++) {
        if (bboxOverlap(bb, groups[g].bbs[k])) { ok = false; break; }
      }
      if (ok) { groups[g].items.push(blocks[i]); groups[g].bbs.push(bb); placed = true; }
    }
    if (!placed) groups.push({ items: [blocks[i]], bbs: [bb] });
  }
  return groups;
}

// 倒角裁剪：把一组斜面半空间应用到实体上。凸实体走 H-rep 直接构造（精确 + 水密 + 无碎片），
//   非凸走布尔（盒子 ∩ 斜面 = 裁剪体，与实体求交）。圆角复用本函数做"先切棱部楔块"这一步。
function chamferCut(mesh, tab, cutPlanes, d, what) {
  var eps = halfspaceEps(tab.mesh), diag = meshDiagonal(tab.mesh);
  // 盒子只是给"无限半空间集合"一个有限边界，宜远大于实体：盒子平面若贴近实体表面，
  //   布尔路径会平白多切出大量碎片（非凸体实测 boundary 178 → 拉开后显著减少）。
  var box = boxHalfspaces(tab.mesh, Math.max(2 * d, 2 * diag));
  var convexPlanes = meshConvexPlanes(tab, eps);
  if (convexPlanes && convexPlanes.length + box.length + cutPlanes.length <= LIMIT_CONVEX_PLANES) {
    return { mesh: halfspacesMesh(convexPlanes.concat(box).concat(cutPlanes), eps), mode: 'direct' };
  }
  var cutter = halfspacesMesh(box.concat(cutPlanes), eps);
  if (!cutter) return { mesh: null, mode: 'boolean' };
  return { mesh: meshBoolean(mesh, cutter, 'intersect'), mode: 'boolean' };
}

// 凸实体的面平面集合（去重）；非凸 / 含开边界 / 面数过大 ⇒ null。
//   ★ 凸体倒角可以完全绕开 BSP 布尔：结果 = 面平面 ∩ 斜面，直接 H-rep 精确构造 ——
//     与布尔相比：解析精确、**天然水密**（布尔会把共面/近共面输入切出 T 缝）、面数最少。
//   凸性判据用"所有内部边都是凸边"（对闭合网格与凸多面体等价），复用边表 ⇒ O(E)，不需要 O(V·F)。
function meshConvexPlanes(tab, eps) {
  if (tab.skipped.open || tab.skipped.nonManifold || tab.skipped.degenerate) return null;
  var i, k, j;
  for (i = 0; i < tab.edges.length; i++) if (!tab.edges[i].convex) return null;
  var pos = tab.mesh.positions, tris = tab.mesh.indices;
  if (!tris.length || tris.length > LIMIT_CONVEX_FACES) return null;
  var planes = [];
  for (k = 0; k < tris.length; k++) {
    var t = tris[k];
    var pl = triPlane([pos[t[0]], pos[t[1]], pos[t[2]]]);
    if (!pl) continue;
    var dup = false;
    for (j = 0; j < planes.length; j++) {
      if (vDot(planes[j].n, pl.n) > 1 - 1e-9 && Math.abs(planes[j].w - pl.w) <= eps) { dup = true; break; }
    }
    if (!dup) planes.push(pl);
  }
  return planes.length >= 4 ? planes : null;
}

// 倒角：裁剪体 = 大盒子 ∩ 所有选中凸边的斜面内半空间（凸 ⇒ H-rep 精确构造）→ 与实体**一次**求交。
function meshChamfer(mesh, op, E, stats) {
  var what = (stats && stats.label) ? stats.label : '倒角';
  if (op.distance === undefined) fail(what + '（chamfer）需要 distance（面内倒角量）');
  var d = evalNum(op.distance, E, 'distance');
  if (!(d > 0)) fail(what + '（chamfer）的 distance 必须为正（收到 ' + d + '）');
  var tab = meshEdgeTable(mesh);
  var sel = selectBevelEdges(tab, op, E, what + '（chamfer）');
  var eps = halfspaceEps(tab.mesh), diag = meshDiagonal(tab.mesh);
  if (!(diag > 0)) fail(what + '：网格尺度为 0，无法倒角');
  var adj = vertexFaceAdjacency(tab.mesh.positions.length, tab.mesh.indices);
  var cutPlanes = [];
  var used = 0, bad = [], k;
  for (k = 0; k < sel.list.length; k++) {
    var ed = sel.list[k], pl = chamferPlaneOfEdge(ed, d);
    if (!pl) { bad.push(fmtVec3(ed.mid) + '（两邻面法线退化）'); continue; }
    var sf = edgeCutSafety(tab, adj, ed, pl, d, diag);
    if (!sf.ok) {
      bad.push(fmtVec3(ed.mid) + '（' + (sf.reason === 'far'
        ? '切到 ' + fmtNum(sf.dist, 3) + ' 外的非邻接特征'
        : '切除范围 ' + fmtNum(sf.dist, 3) + ' 超过边邻域上限 ' + fmtNum(sf.limit, 3)) + '）');
      continue;
    }
    cutPlanes.push({ n: pl.n, w: pl.w });
    used++;
  }
  if (bad.length) {
    fail(what + '：' + bad.length + ' 条边不能安全倒角 —— ' + bad.slice(0, 3).join('；') +
      '。原因：斜面是**无限延伸**的半空间，而这些边附近实体不是凸的（凹腔、镂空、薄壁或相邻特征过近），' +
      '继续切会误伤远处材料。可用 edges 数组只指定安全的棱，或让这些边远离其他特征');
  }
  var cut = chamferCut(mesh, tab, cutPlanes, d, what);
  var out = cut.mesh, mode = cut.mode;
  if (!out) fail(what + '：distance=' + d + ' 过大（斜面把整个实体切光了）——请减小 distance');
  var v0 = Math.abs(meshVolume(mesh)), v1 = Math.abs(meshVolume(out));
  if (v1 > v0 * (1 + 1e-9)) fail(what + '：结果体积反而变大（' + fmtNum(v1, 3) + ' > ' + fmtNum(v0, 3) + '），几何异常');
  if (v1 < v0 * 1e-6) fail(what + '：distance=' + d + ' 过大，几乎切光实体（剩余体积 ' + fmtNum(v1, 3) + '）');
  if (stats) {
    stats.edges = used;
    stats.skippedFlat = sel.flat;
    stats.skippedConcave = sel.concave;
    stats.cutRatio = (v0 - v1) / v0;
    stats.mode = mode;
  }
  return out;
}

// ── 凸体圆角：rolling-ball 的**开运算**构造 (P ⊖ Ball_r) ⊕ Ball_r ──────────────────────
// 圆角的几何本质是形态学**开运算**（先用球把实体磨小、再滚回去），对凸体它可以解析构造，不必做布尔：
//   ① P ⊖ Ball_r：把每个面平面沿法线内缩 r ⇒ 仍是 H-rep 凸体 P'；
//   ② P' ⊕ Ball_r：半径 r 的球绕 P' 边界滚一圈 ⇒ 边界恰好是三类面片的拼合：
//        平面片（P' 的面外移 r）／圆柱片（轴 = P' 的棱）／球片（球心 = P' 的顶点）。
//   ★ 三类片沿**同一条解析曲线**相接，所以拼接天然水密：棱 (i,j) 的方向 ⊥ n_i、n_j
//     ⇒ span(n_i,n_j) 正好是轴的正交补 ⇒ 圆柱片在端点的径向 = 球面上由 n_i、n_j 张成的**大圆弧**；
//     两侧用同一个 slerpDir 取点 ⇒ 顶点坐标逐位相同，不需任何容差兜底。
//   ★ 顶点球片就是 rolling-ball 的**顶点混合面**，因此"相邻棱交会"不再是多个弧块的交集 ——
//     这正是路线 B（布尔拼弧块）做不到、而文档 I.9.5 列为"下一步①"的那件事。
//   结果 ⊆ 原体（自证）：Q ∈ P' ⇒ 对任意面 m 有 n_m·Q ≤ w_m − r；片上的点 X = Q + r·d（|d| = 1）⇒
//     n_m·X ≤ w_m − r + r·n_m·d ≤ w_m。所以圆角只会"切料"，不会外扩。
// 适用：**仅凸体**（非凸要"凹边填角"，是另一套逻辑）。返回 {m[,blendVerts]} / {err} / null(不适用→交路线 B)。

// 单位向量球面线性插值（大圆弧）。端点显式返回 b/a ⇒ 与相邻片共用时坐标逐位一致（水密的关键）。
function slerpDir(a, b, t) {
  if (!(t > 0)) return a.slice();
  if (t >= 1) return b.slice();
  var c = vDot(a, b);
  if (c > 1) c = 1; else if (c < -1) c = -1;
  var om = Math.acos(c), s = Math.sin(om);
  if (s < 1e-12) return vNorm(vAdd(a, b));
  return vNorm(vAdd(vMul(a, Math.sin((1 - t) * om) / s), vMul(b, Math.sin(t * om) / s)));
}

// 一组面法线按"绕中心方向"排序（凸锥的球面多边形顶点顺序）
function sortDirsAround(planes, on) {
  var m = [0, 0, 0], k;
  for (k = 0; k < on.length; k++) m = vAdd(m, planes[on[k]].n);
  if (vLen(m) < 1e-12) return null;
  m = vNorm(m);
  var helper = Math.abs(m[0]) < 0.9 ? [1, 0, 0] : [0, 1, 0];
  var u = vNorm(vCross(m, helper)), vv = vNorm(vCross(m, u));
  var ang = [];
  for (k = 0; k < on.length; k++) {
    var d = planes[on[k]].n, rel = vSub(d, vMul(m, vDot(d, m)));
    ang.push([Math.atan2(vDot(rel, vv), vDot(rel, u)), d]);
  }
  ang.sort(function (A, B) { return A[0] - B[0]; });
  var out = [];
  for (k = 0; k < ang.length; k++) out.push(ang[k][1]);
  return out;
}

// 球面多边形 → 三角形。边界分段与圆柱片端线用同一个 slerpDir（逐位一致 ⇒ 接缝水密）。
//   ★ 只做"单极点扇形"精度不够：apex（重心方向）到边界的角距可达 50°+，三角形弦面会把球顶
//     削掉（实测 r=2 时顶点处偏差 ≈0.32，体积缺口 7.9e-5 且**不随 segments 收敛** —— 因为缺陷
//     来自"径向一刀切"，与边界分几段无关）。所以在每个扇形内再按径向分 rings 层同心环：
//     环 k 的点 = slerp(重心方向, 边界点, k/rings)，环间连成四边形 ⇒ 径向误差按 rings 收敛，
//     而边界点仍是 slerp(d0, d1, j/segs) —— 与圆柱片端线同源，水密性不受影响。
//   内接离散：顶点全落在名义半径的球面上，绝不超出（与弧面切平面的口径一致）。
function sphericalCapTris(center, dirs, r, segs, rings) {
  var m = [0, 0, 0], i, j, k;
  for (i = 0; i < dirs.length; i++) m = vAdd(m, dirs[i]);
  if (vLen(m) < 1e-12) return [];
  var G = vNorm(m);
  var fan = [];
  for (i = 0; i < dirs.length; i++) {
    var d0 = dirs[i], d1 = dirs[(i + 1) % dirs.length];
    var edge = [];
    for (j = 0; j <= segs; j++) edge.push(slerpDir(d0, d1, j / segs));
    var prev = [G];                                   // 环 0：退化成重心方向
    for (k = 1; k <= rings; k++) {
      var ring = [];
      for (j = 0; j <= segs; j++) ring.push(k === rings ? edge[j] : slerpDir(G, edge[j], k / rings));
      if (k === 1) {
        for (j = 0; j < segs; j++) fan.push([G, ring[j], ring[j + 1]]);
      } else {
        for (j = 0; j < segs; j++) {
          fan.push([prev[j], prev[j + 1], ring[j + 1]]);
          fan.push([prev[j], ring[j + 1], ring[j]]);
        }
      }
      prev = ring;
    }
  }
  var out = [], a;
  for (a = 0; a < fan.length; a++) {
    out.push([
      vAdd(center, vMul(fan[a][0], r)),
      vAdd(center, vMul(fan[a][1], r)),
      vAdd(center, vMul(fan[a][2], r))
    ]);
  }
  return out;
}

// 凸体 rolling-ball 圆角：见上方说明。tab 已由调用方建好（复用其凸性判定）。
function rollingBallFillet(tab, r, segs, eps) {
  var planes = meshConvexPlanes(tab, eps);
  if (!planes || planes.length > LIMIT_RB_PLANES) return null;
  var n = planes.length, i, j, k, a;
  // ① 内缩 r
  var inner = [];
  for (i = 0; i < n; i++) inner.push({ n: planes[i].n, w: planes[i].w - r });
  // ② P' 的顶点 = 三平面交点且满足所有内缩半空间；同时记录"通过它的面"（供棱/球片用）
  var verts = [], seen = {};
  for (i = 0; i < n; i++) {
    for (j = i + 1; j < n; j++) {
      for (k = j + 1; k < n; k++) {
        var p = planesIntersect(inner[i], inner[j], inner[k]);
        if (!p) continue;
        var ok = true, on = [];
        for (a = 0; a < n; a++) {
          var dv = vDot(inner[a].n, p) - inner[a].w;
          if (dv > eps) { ok = false; break; }
          if (dv >= -eps) on.push(a);
        }
        if (!ok || on.length < 3) continue;
        var vk = Math.round(p[0] / eps) + '_' + Math.round(p[1] / eps) + '_' + Math.round(p[2] / eps);
        if (seen[vk] !== undefined) {
          var ex = verts[seen[vk]];
          for (a = 0; a < on.length; a++) if (ex.on.indexOf(on[a]) < 0) ex.on.push(on[a]);
          continue;
        }
        seen[vk] = verts.length;
        verts.push({ p: p, on: on });
      }
    }
  }
  if (verts.length < 4) return { err: 'deg' };   // 内缩后实体消失 ⇒ 半径过大
  // ③ 每个面的顶点环（面内按角度排序，与 halfspacesMesh 同口径）
  var faces = [];
  for (i = 0; i < n; i++) {
    var vids = [];
    for (a = 0; a < verts.length; a++) if (verts[a].on.indexOf(i) >= 0) vids.push(a);
    if (vids.length < 3) continue;
    var nv = planes[i].n;
    var helper = Math.abs(nv[0]) < 0.9 ? [1, 0, 0] : [0, 1, 0];
    var u = vNorm(vCross(nv, helper)), vv = vNorm(vCross(nv, u));
    var acc = [0, 0, 0];
    for (a = 0; a < vids.length; a++) acc = vAdd(acc, verts[vids[a]].p);
    var ctr = vMul(acc, 1 / vids.length), ring = [];
    for (a = 0; a < vids.length; a++) {
      var rel = vSub(verts[vids[a]].p, ctr);
      ring.push([Math.atan2(vDot(rel, vv), vDot(rel, u)), vids[a]]);
    }
    ring.sort(function (A, B) { return A[0] - B[0]; });
    var ord = [];
    for (a = 0; a < ring.length; a++) ord.push(ring[a][1]);
    faces.push({ i: i, ring: ord });
  }
  // ④ P' 的棱：恰好被两个面共用的两个顶点（>2 个 ⇒ 退化，交给路线 B；相对面只有 0 个，跳过）
  var ribs = [];
  for (i = 0; i < n; i++) {
    for (j = i + 1; j < n; j++) {
      var sh = [];
      for (a = 0; a < verts.length; a++) {
        if (verts[a].on.indexOf(i) >= 0 && verts[a].on.indexOf(j) >= 0) sh.push(a);
      }
      if (sh.length === 2) ribs.push({ i: i, j: j, A: sh[0], B: sh[1] });
      else if (sh.length > 2) return null;
    }
  }
  var tris = [];
  // ★ 统一绕向：三类片的顶点顺序由各自的参数化决定（面内角序／弧段方向／轴方向），彼此不一致，
  //   而"整体有向体积翻转"只能翻**全部**面、修不了部分反向（实测 flipped=108、体积偏小 17.7%）。
  //   凸体的外法线判据很干净：任意片的法线都应背离内缩体内部 ⇒ 以内缩体质心为参考逐片定向。
  var refC = [0, 0, 0];
  for (a = 0; a < verts.length; a++) refC = vAdd(refC, verts[a].p);
  refC = vMul(refC, 1 / verts.length);
  var emit = function (p0, p1, p2) {
    var nn = vCross(vSub(p1, p0), vSub(p2, p0));
    var c2 = vMul(vAdd(vAdd(p0, p1), p2), 1 / 3);
    if (vDot(nn, vSub(c2, refC)) < 0) tris.push([p0, p2, p1]);
    else tris.push([p0, p1, p2]);
  };
  // ⑤-1 平面片：面环整体外移 r
  for (k = 0; k < faces.length; k++) {
    var f = faces[k], fn = planes[f.i].n, w2 = [];
    for (a = 0; a < f.ring.length; a++) w2.push(vAdd(verts[f.ring[a]].p, vMul(fn, r)));
    for (a = 1; a + 1 < w2.length; a++) emit(w2[0], w2[a], w2[a + 1]);
  }
  // ⑤-2 圆柱片：轴 = P' 的棱，径向 = 两个面法线之间的大圆弧（与球片边界同源 ⇒ 逐位相接）
  for (k = 0; k < ribs.length; k++) {
    var e = ribs[k], A = verts[e.A].p, B = verts[e.B].p;
    var n1 = planes[e.i].n, n2 = planes[e.j].n;
    for (a = 0; a < segs; a++) {
      var d0 = slerpDir(n1, n2, a / segs), d1 = slerpDir(n1, n2, (a + 1) / segs);
      var q00 = vAdd(A, vMul(d0, r)), q01 = vAdd(A, vMul(d1, r));
      var q10 = vAdd(B, vMul(d0, r)), q11 = vAdd(B, vMul(d1, r));
      emit(q00, q01, q11);
      emit(q00, q11, q10);
    }
  }
  // ⑤-3 球片（顶点混合面）：球心 = P' 的顶点，方向域 = 该顶点相邻面法线张成的凸锥。
  //   径向分层数与弧面段数同量级（√segs，2~6）：让"球面顶点处"与"圆柱面处"的离散精度相当。
  var blendRings = Math.ceil(Math.sqrt(segs));
  if (blendRings < 2) blendRings = 2; else if (blendRings > 6) blendRings = 6;
  var blendVerts = 0;
  for (a = 0; a < verts.length; a++) {
    if (verts[a].on.length < 3) continue;
    var dirs = sortDirsAround(planes, verts[a].on);
    if (!dirs) continue;
    var st = sphericalCapTris(verts[a].p, dirs, r, segs, blendRings);
    if (!st.length) continue;
    blendVerts++;
    for (k = 0; k < st.length; k++) emit(st[k][0], st[k][1], st[k][2]);
  }
  if (!tris.length) return null;
  var out = trisToMesh(tris, eps);
  if (!out.indices.length) return null;
  return { m: meshOrientOutward(out), blendVerts: blendVerts, pieces: faces.length + ribs.length + blendVerts };
}

// 选中集是否 = **全部**真实棱（rolling-ball 走"整体开运算"，只圆一部分棱会破坏其前提）。
//   ★ 判据不能用"convex 的边数"：共面边（同一平面被三角形切开的对角线）sgn ≈ 0，按凸凹判定也算 convex，
//     立方体 18 条内部边里 6 条对角线会被算进来 ⇒ 永远判否、rolling-ball 形同虚设。
//     正确口径 = 折角 ≥ minAngle 的边总数（那才是"棱"），与 sel.list 比数量即集合相等（list ⊆ 棱）。
function rbCoversAllConvexEdges(tab, sel) {
  var sharp = 0, i;
  for (i = 0; i < tab.edges.length; i++) {
    if (tab.edges[i].angleDeg >= sel.minAngle - 1e-9) sharp++;
  }
  return sharp > 0 && sel.list.length === sharp;
}

// 圆角弧块的半空间表示（凸 = 扇形柱）：轴 = 距两邻面各 r 的直线（在实体内部），半径 r，
//   角度范围 = 内二面角 θ，沿边范围 = 边本身；弧面用 segs 个切平面离散（顶点落在半径 r 的圆上）。
function filletArcHalfspaces(edge, r, segs) {
  var c = vDot(edge.n1, edge.n2);
  if (c <= -1 + 1e-9) return null;
  var o = vSub(edge.mid, vMul(vAdd(edge.n1, edge.n2), r / (1 + c)));   // 距两面各 r 的轴点
  var s = edge.dir, half = edge.len / 2;
  var nMid = vNorm(vAdd(edge.n1, edge.n2));
  var vv = vNorm(vCross(s, edge.n1));
  if (vDot(vv, nMid) < 0) vv = vMul(vv, -1);
  var w2 = vNorm(vCross(s, edge.n2));
  if (vDot(w2, nMid) < 0) w2 = vMul(w2, -1);
  var theta = edge.dihedralDeg * Math.PI / 180;
  var planes = [
    { n: vMul(s, -1), w: -(vDot(s, o) - half) },
    { n: s, w: vDot(s, o) + half },
    { n: vMul(vv, -1), w: -vDot(vv, o) },     // 过轴与切点的径向平面（n1 侧）
    { n: vMul(w2, -1), w: -vDot(w2, o) }      // 过轴与切点的径向平面（n2 侧）
  ];
  // ★ 再加一道「斜面内侧」的反向约束，把加回块从整个四分之一圆柱切成**弓形**（圆 ∩ 斜面外侧）：
  //   完整圆柱块与「已倒角的实体」只沿两条切点**线**相切（零体积接触）——BSP union 处理这种退化
  //   接触会留下大量未配对边；换成弓形后二者沿**斜面片**相碰（面接触），拓扑干净得多。
  //   几何等价性：结果 = 实体 ∩ ({斜面内侧} ∪ 弓形) = 实体 ∩ ({斜面内侧} ∪ 四分之一圆柱) —— 与直接
  //   union 整个圆柱块完全一致，只是接触方式从"线"变成"面"。
  var cp = chamferPlaneOfEdge(edge, r);
  if (cp) planes.push({ n: vMul(cp.n, -1), w: -cp.w });
  for (var i = 0; i < segs; i++) {
    var a0 = theta * i / segs, a1 = theta * (i + 1) / segs, am = (a0 + a1) / 2;
    var nr = vNorm(vAdd(vMul(edge.n1, Math.cos(am)), vMul(vv, Math.sin(am))));
    planes.push({ n: nr, w: vDot(nr, o) + r * Math.cos((a1 - a0) / 2) });
  }
  return { o: o, planes: planes };
}

// 圆角：先按 r 倒角切掉棱部楔块（复用 chamferCut），再把「圆柱片段（弧块）」拼回去 —— 路线 B 的布尔近似。
//   · 弧块**先合并成一个网格、再整体 union 一次**：逐块 union 会让 BSP 把整个网格反复切碎
//     （实测 12 条棱逐块 union 把 12 面的立方体顶到 5.4 万面）。
//     ★ 本路径**不做顶点混合**：相邻棱的弧块在顶点处相交会留下大量未配对边，故那里明确拒绝。
//   · 顶点处是多个弧块的交集（非解析过渡面）—— 这正是必须拒绝相邻棱的原因；凸体全棱改走 rolling-ball。
function meshFillet(mesh, op, E, stats) {
  var what = (stats && stats.label) ? stats.label : '圆角';
  if (op.radius === undefined && op.distance === undefined) fail(what + '（fillet）需要 radius');
  var r = evalNum(op.radius === undefined ? op.distance : op.radius, E, 'radius');
  if (!(r > 0)) fail(what + '（fillet）的 radius 必须为正（收到 ' + r + '）');
  var segs = op.segments === undefined ? ARC_SEG_DEFAULT : clampInt(evalNum(op.segments, E, 'segments'), 2, 64, what + ' 的 segments');
  var tab = meshEdgeTable(mesh);
  var sel = selectBevelEdges(tab, op, E, what + '（fillet）');
  // ① rolling-ball 精确路径（凸体 + 选中集 = 全部凸棱）：结果 = (P ⊖ Ball_r) ⊕ Ball_r，解析构造、
  //    天然水密、无布尔，顶点处是解析球片 ⇒ **相邻棱（含立方体 12 棱全圆角）也支持**。
  //    ★ 必须在下面的边数上限之前：那种上限是布尔路径的成本约束，rolling-ball 不受它限制。
  if (rbCoversAllConvexEdges(tab, sel)) {
    var rb = rollingBallFillet(tab, r, segs, halfspaceEps(tab.mesh));
    if (rb && rb.err) {
      fail(what + '：radius=' + r + ' 过大 —— 各个面沿法线内缩 ' + r + ' 后实体已经消失，请减小 radius');
    }
    if (rb && rb.m) {
      var vb0 = Math.abs(meshVolume(mesh)), vb1 = Math.abs(meshVolume(rb.m));
      if (vb1 > vb0 * (1 + 1e-9)) fail(what + '：结果体积反而变大（' + fmtNum(vb1, 3) + ' > ' + fmtNum(vb0, 3) + '），几何异常');
      if (vb1 < vb0 * 1e-6) fail(what + '：radius=' + r + ' 过大，几乎切光实体（剩余体积 ' + fmtNum(vb1, 3) + '）');
      if (stats) {
        stats.edges = sel.list.length;
        stats.skippedFlat = sel.flat;
        stats.skippedConcave = sel.concave;
        stats.cutRatio = (vb0 - vb1) / vb0;
        stats.mode = 'rolling-ball';
        stats.blendVerts = rb.blendVerts;
        stats.pieces = rb.pieces;
      }
      return rb.m;
    }
  }
  // ② 布尔近似路径（路线 B）：只圆部分棱、或非凸体，继续走下面。
  if (sel.list.length > LIMIT_FILLET_EDGES) {
    fail(what + '：选中 ' + sel.list.length + ' 条边，超过这条（布尔近似）路径的单次上限 ' + LIMIT_FILLET_EDGES + ' 条。' +
      '原因：它要「切棱部料 + 拼回圆柱片段」，走布尔；多条棱在顶点交会时相邻弧块彼此相交，' +
      'BSP 的三角形数会超线性膨胀（实测 12 条棱的立方体要 76 s / 5.2 万面，且水密性明显变差）。' +
      '若对象是**凸体**：不指定 edges（或指定全部凸棱）即走 rolling-ball 精确路径，条数不受此限；' +
      '否则请用 edges 数组分批处理（每批 ≤ ' + LIMIT_FILLET_EDGES + ' 条），或改用 chamfer（倒角走 H-rep，一次可处理 64 条且精确水密）');
  }
  // 这条（布尔近似）路径不支持相邻棱（共享顶点）：两条相邻棱的加回块在角部相交，BSP 会留下大量
  //   未配对边（实测 2 条相邻棱 boundary=206、「4 竖 + 1 横」boundary=554，而互不相邻的棱是 0）。
  //   凸体全棱已由上面的 rolling-ball 处理（顶点处是解析球片，12 条棱 boundary=0）；这里明确拒绝，
  //   不静默产出坏网格。chamfer 也没有这个限制：它只做半空间求交，顶点处天然正确且精确。
  var useCount = {}, shared = 0, kk;
  for (kk = 0; kk < sel.list.length; kk++) {
    useCount[sel.list[kk].a] = (useCount[sel.list[kk].a] || 0) + 1;
    useCount[sel.list[kk].b] = (useCount[sel.list[kk].b] || 0) + 1;
  }
  for (kk in useCount) if (useCount.hasOwnProperty(kk) && useCount[kk] > 1) shared++;
  if (shared) {
    fail(what + '：选中的棱里有 ' + shared + ' 个顶点被两条以上棱共用（相邻棱交会）。' +
      '这条（布尔近似）路径不做顶点混合，硬做会产出不水密网格（实测 2 条相邻棱即 206 条开边界）——' +
      '若对象是**凸体**：不指定 edges（或指定全部凸棱）即走 rolling-ball 精确路径，' +
      '顶点处是解析球片，相邻棱同样严格水密；非凸体请只选**互不相邻**的棱（如同方向的 4 条竖棱、异面棱），' +
      '或改用 chamfer（半空间求交，顶点处天然正确且精确水密）');
  }
  var eps = halfspaceEps(tab.mesh), diag = meshDiagonal(tab.mesh);
  if (!(diag > 0)) fail(what + '：网格尺度为 0，无法圆角');
  var adj = vertexFaceAdjacency(tab.mesh.positions.length, tab.mesh.indices);
  var cutPlanes = [], arcMeshes = [], used = 0, bad = [], k;
  for (k = 0; k < sel.list.length; k++) {
    var ed = sel.list[k];
    var cp = chamferPlaneOfEdge(ed, r);
    var arcSpec = filletArcHalfspaces(ed, r, segs);
    if (!cp || !arcSpec) { bad.push(fmtVec3(ed.mid) + '（两邻面法线退化）'); continue; }
    var sf = edgeCutSafety(tab, adj, ed, cp, r, diag);
    if (!sf.ok) {
      bad.push(fmtVec3(ed.mid) + '（' + (sf.reason === 'far'
        ? '切到 ' + fmtNum(sf.dist, 3) + ' 外的非邻接特征'
        : '切除范围 ' + fmtNum(sf.dist, 3) + ' 超过边邻域上限 ' + fmtNum(sf.limit, 3)) + '）');
      continue;
    }
    var arc = halfspacesMesh(arcSpec.planes, eps);
    if (!arc) { bad.push(fmtVec3(ed.mid) + '（半径 ' + r + ' 过大，弧块退化）'); continue; }
    cutPlanes.push({ n: cp.n, w: cp.w });
    arcMeshes.push(arc);
    used++;
  }
  if (bad.length) {
    fail(what + '：' + bad.length + ' 条边不能安全圆角 —— ' + bad.slice(0, 3).join('；') + '。原因同 chamfer（见其报错说明）');
  }
  var cut = chamferCut(mesh, tab, cutPlanes, r, what);
  if (!cut.mesh) fail(what + '：radius=' + r + ' 过大（把整个实体切光了）——请减小 radius');
  var cur = cut.mesh;
  // 弧块分组：组内不相交 ⇒ 直接拼接，只有跨组才付布尔代价（见 groupBlocksByBBox 的实测说明）
  var groups = groupBlocksByBBox(arcMeshes);
  for (k = 0; k < groups.length; k++) {
    var gm = groups[k].items.length > 1 ? meshMerge(groups[k].items) : groups[k].items[0];
    cur = meshBoolean(cur, gm, 'union');
  }
  var v0 = Math.abs(meshVolume(mesh)), v1 = Math.abs(meshVolume(cur));
  if (v1 > v0 * (1 + 1e-6)) fail(what + '：结果体积反而变大（' + fmtNum(v1, 3) + ' > ' + fmtNum(v0, 3) + '），几何异常');
  if (v1 < v0 * 1e-6) fail(what + '：radius=' + r + ' 过大，几乎切光实体（剩余体积 ' + fmtNum(v1, 3) + '）');
  if (stats) {
    stats.edges = used;
    stats.skippedFlat = sel.flat;
    stats.skippedConcave = sel.concave;
    stats.cutRatio = (v0 - v1) / v0;
    stats.mode = cut.mode;
  }
  return cur;
}

// 镜像矩阵（沿过原点的法向轴镜像；det = -1，meshTransform 会自动翻转绕向保持外向）
function m4mirror(axis) {
  var a = vNorm(axis);
  if (vLen(a) < 0.5) fail('镜像轴不能为零向量');
  var x = a[0], y = a[1], z = a[2];
  return [
    1 - 2 * x * x, -2 * x * y, -2 * x * z, 0,
    -2 * x * y, 1 - 2 * y * y, -2 * y * z, 0,
    -2 * x * z, -2 * y * z, 1 - 2 * z * z, 0,
    0, 0, 0, 1
  ];
}

// ── 表达式求值（参数化：改一个参数，几何自动重算）────────────────
// 支持：数字、参数名、+ - * / % ^、括号、一元负号、函数调用、常量 pi/e。
// 不用 eval / Function（沙箱安全、结果可预期、错误信息可操作）。
var EXPR_FUNCS = {
  abs: Math.abs, sqrt: Math.sqrt, sin: Math.sin, cos: Math.cos, tan: Math.tan,
  asin: Math.asin, acos: Math.acos, atan: Math.atan, atan2: Math.atan2,
  floor: Math.floor, ceil: Math.ceil, round: Math.round, pow: Math.pow, hypot: Math.sqrt,
  min: Math.min, max: Math.max,
  sign: Math.sign || function (v) { return v > 0 ? 1 : (v < 0 ? -1 : 0); }
};
var EXPR_CONSTS = { pi: Math.PI, PI: Math.PI, e: Math.E, tau: 2 * Math.PI, TAU: 2 * Math.PI };
var RE_NUMBER = /^[0-9]*\.?[0-9]+([eE][+-]?[0-9]+)?/;
var RE_IDENT = /^[A-Za-z_][A-Za-z0-9_]*/;

function pendingError() {
  var e = new Error('__pending__');
  e.pending = true;
  return e;
}

// env：已解析的参数值；known：参数 id → 定义（用于区分"稍后才解析"与"根本不存在"）
function evalMathExpr(src, env, known) {
  var s = String(src);
  if (s.length > 512) fail('表达式过长（>' + 512 + ' 字符）');
  var pos = 0, depth = 0;
  function skipWs() { while (pos < s.length && (s.charAt(pos) === ' ' || s.charAt(pos) === '\t')) pos++; }
  function parseExpr() {
    var v = parseTerm();
    for (;;) {
      skipWs();
      var c = s.charAt(pos);
      if (c === '+') { pos++; v += parseTerm(); }
      else if (c === '-') { pos++; v -= parseTerm(); }
      else return v;
    }
  }
  function parseTerm() {
    var v = parseFactor();
    for (;;) {
      skipWs();
      var c = s.charAt(pos);
      if (c === '*') { pos++; v *= parseFactor(); }
      else if (c === '/') {
        pos++;
        var d = parseFactor();
        if (Math.abs(d) < 1e-300) fail('表达式出现除以 0："' + src + '"');
        v /= d;
      } else if (c === '%') {
        pos++;
        var m = parseFactor();
        if (Math.abs(m) < 1e-300) fail('表达式出现对 0 取模："' + src + '"');
        v %= m;
      } else return v;
    }
  }
  function parseFactor() {
    var base = parseUnary();
    skipWs();
    if (s.charAt(pos) === '^') { pos++; return Math.pow(base, parseFactor()); } // 右结合
    return base;
  }
  function parseUnary() {
    skipWs();
    var c = s.charAt(pos);
    if (c === '-') { pos++; return -parseUnary(); }
    if (c === '+') { pos++; return parseUnary(); }
    return parsePrimary();
  }
  function parsePrimary() {
    skipWs();
    var c = s.charAt(pos);
    if (c === '(') {
      pos++;
      if (++depth > LIMIT_EXPR_DEPTH) fail('表达式嵌套过深（>' + LIMIT_EXPR_DEPTH + '）');
      var v = parseExpr();
      skipWs();
      if (s.charAt(pos) !== ')') fail('表达式缺少右括号："' + src + '"');
      pos++;
      depth--;
      return v;
    }
    var rest = s.slice(pos);
    var mn = RE_NUMBER.exec(rest);
    if (mn && mn[0]) {
      if (/^[0-9]/.test(c) || (c === '.' && mn[0].length > 1)) {
        pos += mn[0].length;
        return Number(mn[0]);
      }
    }
    var mi = RE_IDENT.exec(rest);
    if (mi && mi[0]) {
      var name = mi[0];
      pos += name.length;
      skipWs();
      if (s.charAt(pos) === '(') {
        pos++;
        var args = [];
        skipWs();
        if (s.charAt(pos) !== ')') {
          for (;;) {
            args.push(parseExpr());
            skipWs();
            if (s.charAt(pos) === ',') { pos++; continue; }
            break;
          }
        }
        skipWs();
        if (s.charAt(pos) !== ')') fail('函数调用缺少右括号："' + src + '"');
        pos++;
        var fn = EXPR_FUNCS[name];
        if (!fn) fail('未知函数 ' + name + '()（可用：' + objKeys(EXPR_FUNCS).join(', ') + '）');
        if (!args.length) fail('函数 ' + name + '() 缺少参数');
        return fn.apply(null, args);
      }
      if (env && env.hasOwnProperty(name)) return env[name];
      if (EXPR_CONSTS.hasOwnProperty(name)) return EXPR_CONSTS[name];
      if (known && known[name]) throw pendingError(); // 定义在别处、稍后解析
      fail('表达式引用了不存在的参数 ' + name + '（已定义参数：' + (known ? objKeys(known).join(', ') || '无' : '无') + '）');
    }
    fail('表达式无法解析："' + src + '"（位置 ' + pos + '）');
  }
  var out = parseExpr();
  skipWs();
  if (pos !== s.length) fail('表达式有无法解析的尾部："' + src + '"');
  if (!isFinite(out)) fail('表达式结果不是有限数："' + src + '"');
  return out;
}

function objKeys(o) {
  var out = [], k;
  for (k in o) if (o.hasOwnProperty(k)) out.push(k);
  return out;
}

// 求值上下文：E = { params: {id: 值}, known: {id: 定义} }
function evalNum(v, E, what) {
  if (isNum(v)) return v;
  if (isStr(v)) return evalMathExpr(v, E.params, E.known);
  fail((what || '数值') + '必须是数字或表达式字符串（收到 ' + JSON.stringify(v) + '）');
}
function evalVec3(v, E, what, dflt) {
  if (v === undefined || v === null) return dflt ? dflt.slice() : [0, 0, 0];
  if (isArr(v)) {
    if (v.length < 3) {
      if (v.length === 1 || v.length === 2) { // 允许 [x] / [x,y]（z 缺省 0 或 dflt）
        var out0 = dflt ? dflt.slice() : [0, 0, 0];
        for (var q = 0; q < v.length; q++) out0[q] = evalNum(v[q], E, what);
        return out0;
      }
      fail((what || '向量') + '必须是长度 3 的数组');
    }
    return [evalNum(v[0], E, what), evalNum(v[1], E, what), evalNum(v[2], E, what)];
  }
  if (isObj(v)) return [evalNum(v.x === undefined ? 0 : v.x, E, what), evalNum(v.y === undefined ? 0 : v.y, E, what), evalNum(v.z === undefined ? 0 : v.z, E, what)];
  var s = evalNum(v, E, what);
  if (dflt) return [s, s, s];
  return [s, s, s];
}
function evalInt(v, E, what, dflt) {
  if (v === undefined || v === null) return dflt;
  return Math.round(evalNum(v, E, what));
}

// ── 轮廓求值（extrude / revolve 的输入同样支持表达式）──────────
function evalProfile(spec, E) {
  if (isArr(spec)) {
    var pts = [];
    for (var k = 0; k < spec.length; k++) {
      var p = spec[k];
      if (!isArr(p) || p.length < 2) fail('profile 第 ' + k + ' 个点必须是 [x,y]');
      pts.push([evalNum(p[0], E, 'profile'), evalNum(p[1], E, 'profile')]);
    }
    return pts;
  }
  if (!isObj(spec)) fail('profile 必须是点集 [[x,y],...] 或 {type,...}');
  var o = { type: spec.type === undefined ? 'polygon' : spec.type };
  if (spec.size !== undefined) {
    o.size = isArr(spec.size) ? [evalNum(spec.size[0], E, 'size'), evalNum(spec.size[1], E, 'size')] : evalNum(spec.size, E, 'size');
  }
  if (spec.center !== undefined) {
    if (!isArr(spec.center) || spec.center.length < 2) {
      fail('轮廓 center 必须是 [x,y]（收到 ' + JSON.stringify(spec.center) + '）');
    }
    o.center = [evalNum(spec.center[0], E, 'center'), evalNum(spec.center[1], E, 'center')];
  }
  if (spec.radius !== undefined) o.radius = evalNum(spec.radius, E, 'radius');
  if (spec.innerRadius !== undefined) o.innerRadius = evalNum(spec.innerRadius, E, 'innerRadius');
  if (spec.vertices !== undefined) o.vertices = evalNum(spec.vertices, E, 'vertices');
  if (spec.segments !== undefined) o.segments = evalNum(spec.segments, E, 'segments');
  if (spec.points !== undefined) {
    if (!isArr(spec.points)) fail('profile.points 必须是点集');
    o.points = [];
    for (var m = 0; m < spec.points.length; m++) {
      o.points.push([evalNum(spec.points[m][0], E, 'points'), evalNum(spec.points[m][1], E, 'points')]);
    }
  }
  return o;
}

// 带孔挤出的 holes 解析：数组元素与 profile 同口径（点集或 {type,...}）
function evalHoles(spec, E) {
  if (spec === undefined || spec === null) return undefined;
  if (!isArr(spec)) fail('holes 必须是数组：每个元素是一个闭合轮廓（点集 [[x,y],...] 或 {type:"circle"|"rect"|"polygon",...}）');
  var out = [];
  for (var k = 0; k < spec.length; k++) out.push(evalProfile(spec[k], E));
  return out;
}
// 扫掠路径解析（支持表达式）：去相邻重合点（零长段没有切线，框架会退化）
// ★ 字段名用 along 而不是 path —— path 在工具参数里已经是"工程文件路径"，不能复用。
function evalPath(spec, E) {
  if (!isArr(spec) || spec.length < 2) fail('sweep 需要 along 数组（≥2 个 [x,y,z] 点；注意工具层的 path 是工程文件路径）');
  var out = [], k;
  for (k = 0; k < spec.length; k++) {
    var p = spec[k];
    if (!isArr(p) || p.length < 3) fail('sweep.along[' + k + '] 必须是 [x,y,z]');
    out.push([evalNum(p[0], E, 'path'), evalNum(p[1], E, 'path'), evalNum(p[2], E, 'path')]);
  }
  var clean = [];
  for (k = 0; k < out.length; k++) {
    if (clean.length && vDist(clean[clean.length - 1], out[k]) <= EPS_SPATIAL) continue;
    clean.push(out[k]);
  }
  if (clean.length < 2) fail('sweep.along 去相邻重合点后不足 2 个不同点（当前 ' + clean.length + ' 个）');
  return clean;
}

// ── 几何表达式 → 网格（形状树，支持嵌套布尔）────────────────────
var SHAPE_TYPES = ['cuboid', 'cube', 'box', 'sphere', 'ellipsoid', 'cylinder', 'cone', 'frustum',
  'torus', 'polyhedron', 'extrude', 'revolve', 'sweep', 'mesh', 'boolean', 'group'];

// 形状构建的统一出口：**外向化**（挤出 / 旋转体等生成器的绕向差异在这里一次性归一，
// 保证"有向体积 > 0 = 法线朝外"，下游的布尔裁剪与导出都依赖这个不变量）
function buildShape(shape, E, depth) {
  return meshOrientOutward(buildShapeInner(shape, E, depth));
}
function buildShapeInner(shape, E, depth) {
  if (!isObj(shape)) fail('几何形状必须是对象（如 {"type":"cuboid","size":[10,10,10]}）');
  var t = shape.type === undefined ? 'cuboid' : shape.type;
  var d = depth === undefined ? 0 : depth;
  if (d > 12) fail('几何嵌套过深（>12 层）');
  if (t === 'cuboid' || t === 'box' || t === 'cube') {
    var sz = shape.size === undefined ? (shape.sides === undefined ? (t === 'cube' ? 2 : [2, 2, 2]) : shape.sides) : shape.size;
    return shapeCuboid({
      size: t === 'cube' ? (isNum(sz) || isStr(sz) ? evalNum(sz, E, 'size') : evalVec3(sz, E, 'size', [2, 2, 2]))
        : evalVec3(sz, E, 'size', [2, 2, 2]),
      center: evalVec3(shape.center, E, 'center', [0, 0, 0])
    });
  }
  if (t === 'sphere') {
    return shapeSphere({
      radius: evalNum(shape.radius === undefined ? 1 : shape.radius, E, 'radius'),
      segments: evalInt(shape.segments, E, 'segments', 32),
      rings: shape.rings === undefined ? undefined : evalInt(shape.rings, E, 'rings', 0),
      center: evalVec3(shape.center, E, 'center', [0, 0, 0])
    });
  }
  if (t === 'ellipsoid') {
    return shapeEllipsoid({
      radius: evalVec3(shape.radius === undefined ? 1 : shape.radius, E, 'radius', [1, 1, 1]),
      segments: evalInt(shape.segments, E, 'segments', 32),
      rings: shape.rings === undefined ? undefined : evalInt(shape.rings, E, 'rings', 0),
      center: evalVec3(shape.center, E, 'center', [0, 0, 0])
    });
  }
  if (t === 'cylinder' || t === 'cone' || t === 'frustum') {
    var rb = shape.radiusBottom !== undefined ? shape.radiusBottom : shape.radius;
    var rt = shape.radiusTop;
    if (t === 'cone' && rt === undefined) rt = 0;
    return shapeFrustum({
      type: t === 'cone' ? 'cone' : 'cylinder',
      radius: rb === undefined ? 1 : evalNum(rb, E, 'radius'),
      radiusBottom: rb === undefined ? undefined : evalNum(rb, E, 'radiusBottom'),
      radiusTop: rt === undefined ? undefined : evalNum(rt, E, 'radiusTop'),
      height: evalNum(shape.height === undefined ? 2 : shape.height, E, 'height'),
      segments: evalInt(shape.segments, E, 'segments', 32),
      center: evalVec3(shape.center, E, 'center', [0, 0, 0])
    });
  }
  if (t === 'torus') {
    return shapeTorus({
      innerRadius: evalNum(shape.innerRadius === undefined ? 1 : shape.innerRadius, E, 'innerRadius'),
      outerRadius: evalNum(shape.outerRadius === undefined ? 4 : shape.outerRadius, E, 'outerRadius'),
      segments: evalInt(shape.segments, E, 'segments', 32),
      rings: evalInt(shape.rings, E, 'rings', 32),
      center: evalVec3(shape.center, E, 'center', [0, 0, 0])
    });
  }
  if (t === 'polyhedron') {
    var pts = [], i;
    if (!isArr(shape.points)) fail('polyhedron 需要 points 数组');
    for (i = 0; i < shape.points.length; i++) pts.push(evalVec3(shape.points[i], E, 'points'));
    return shapePolyhedron({ points: pts, faces: shape.faces });
  }
  if (t === 'extrude') {
    return shapeExtrude({
      profile: evalProfile(shape.profile, E),
      holes: evalHoles(shape.holes, E),
      height: shape.height === undefined ? 1 : evalNum(shape.height, E, 'height'),
      center: shape.center === undefined ? undefined : evalNum(shape.center, E, 'center'),
      // twist = 总扭转角（度），segments = 分层数（extrude 的 segments 与曲面的 segments 同名字段，
      // 但语义是「扭转方向的分层数」，与轮廓自身的 segments 各管一段，互不影响）
      twist: shape.twist === undefined ? undefined : evalNum(shape.twist, E, 'twist'),
      segments: shape.segments === undefined ? undefined : evalInt(shape.segments, E, 'segments', TWIST_LAYERS_DEFAULT)
    });
  }
  if (t === 'revolve') {
    return shapeRevolve({
      profile: evalProfile(shape.profile, E),
      segments: evalInt(shape.segments, E, 'segments', 32),
      angle: shape.angle === undefined ? 360 : evalNum(shape.angle, E, 'angle')
    });
  }
  if (t === 'sweep') {
    var swScale = null;
    if (shape.scale !== undefined && shape.scale !== null) {
      if (isArr(shape.scale)) {
        swScale = [];
        for (var si = 0; si < shape.scale.length; si++) swScale.push(evalNum(shape.scale[si], E, 'scale'));
      } else {
        swScale = evalNum(shape.scale, E, 'scale');
      }
    }
    return shapeSweep({
      profile: evalProfile(shape.profile, E),
      holes: evalHoles(shape.holes, E),
      along: evalPath(shape.along, E),
      closed: shape.closed,
      twist: shape.twist === undefined ? undefined : evalNum(shape.twist, E, 'twist'),
      scale: swScale,
      up: shape.up === undefined ? undefined : evalVec3(shape.up, E, 'up')
    });
  }
  if (t === 'mesh') return shapeMesh(shape);
  if (t === 'boolean') {
    var op = shape.op === undefined ? 'union' : shape.op;
    var list = [];
    if (shape.of !== undefined) {
      if (!isArr(shape.of) || shape.of.length < 2) fail('boolean.of 至少需要 2 个形状');
      for (var q = 0; q < shape.of.length; q++) list.push(buildShape(shape.of[q], E, d + 1));
    } else {
      if (!shape.a || !shape.b) fail('boolean 需要 a 与 b（或 of 数组）');
      list.push(buildShape(shape.a, E, d + 1));
      list.push(buildShape(shape.b, E, d + 1));
    }
    var acc = list[0];
    for (var w = 1; w < list.length; w++) acc = meshBoolean(acc, list[w], op);
    return acc;
  }
  if (t === 'group') {
    if (!isArr(shape.shapes) || !shape.shapes.length) fail('group 需要 shapes 数组');
    var gs = [];
    for (var z = 0; z < shape.shapes.length; z++) {
      var sub = shape.shapes[z];
      var ms = buildShape(isObj(sub) && sub.shape ? sub.shape : sub, E, d + 1);
      if (isObj(sub) && sub.transform) ms = meshTransform(ms, m4fromTRS(evalTRS(sub.transform, E)));
      gs.push(ms);
    }
    return meshMerge(gs);
  }
  fail('未知几何类型 ' + t + '（可用：' + SHAPE_TYPES.join(', ') + '）');
}

// 变换求值（translate / rotate / scale，数值均可为表达式）
function evalTRS(trs, E) {
  if (!isObj(trs)) fail('transform 必须是对象');
  return {
    translate: evalVec3(trs.translate, E, 'translate', [0, 0, 0]),
    rotate: evalVec3(trs.rotate, E, 'rotate', [0, 0, 0]),
    scale: evalVec3(trs.scale, E, 'scale', [1, 1, 1])
  };
}

// 轴向（'x'|'y'|'z' 或 [x,y,z]）
function evalAxis(v, E, what, dflt) {
  if (v === undefined || v === null) return dflt ? dflt.slice() : [0, 0, 1];
  if (isStr(v)) {
    var s = v.toLowerCase();
    if (s === 'x') return [1, 0, 0];
    if (s === 'y') return [0, 1, 0];
    if (s === 'z') return [0, 0, 1];
    fail((what || '轴') + '只能是 x / y / z 或 [x,y,z]');
  }
  return vNorm(evalVec3(v, E, what, [0, 0, 1]));
}

// 阵列（阵列出的实体是**多个独立实体**，不参与布尔 —— 与 CAD 惯例一致）
function applyRepeat(mesh, rep, E, what) {
  var mode = rep.mode === undefined ? 'linear' : rep.mode;
  var out = [], k;
  if (mode === 'mirror') {
    out.push(mesh);
    out.push(meshTransform(mesh, m4mirror(evalAxis(rep.axis, E, 'axis', [1, 0, 0]))));
    return meshMerge(out);
  }
  var count = clampInt(evalNum(rep.count === undefined ? 2 : rep.count, E, 'count'), 2, LIMIT_ARRAY, 'count');
  if (mode === 'linear') {
    var delta = evalVec3(rep.delta === undefined ? rep.spacing : rep.delta, E, 'delta', [0, 0, 0]);
    if (rep.spacing !== undefined && rep.delta === undefined && rep.axis !== undefined) {
      delta = vMul(evalAxis(rep.axis, E, 'axis', [1, 0, 0]), evalNum(rep.spacing, E, 'spacing'));
    }
    if (vLen(delta) <= EPS) fail((what || 'linear 阵列') + '的 delta 不能为零向量');
    for (k = 0; k < count; k++) out.push(meshTransform(mesh, m4translate(vMul(delta, k))));
    return meshMerge(out);
  }
  if (mode === 'circular') {
    var axV = evalAxis(rep.axis, E, 'axis', [0, 0, 1]);
    var radius = evalNum(rep.radius === undefined ? 0 : rep.radius, E, 'radius');
    var start = evalNum(rep.startAngle === undefined ? 0 : rep.startAngle, E, 'startAngle');
    var span = evalNum(rep.angle === undefined ? 360 : rep.angle, E, 'angle');
    var center = evalVec3(rep.center, E, 'center', [0, 0, 0]);
    // 轴的垂直基（起始角 0 的位置可预期：z 轴 → +x，y 轴 → +z，x 轴 → +y；
    // 角度递增方向 = u 转向 vv = 绕轴逆时针）
    var u;
    if (Math.abs(axV[2] - 1) < 1e-9) u = [1, 0, 0];
    else if (Math.abs(axV[1] - 1) < 1e-9) u = [0, 0, 1];
    else if (Math.abs(axV[0] - 1) < 1e-9) u = [0, 1, 0];
    else { var helper = Math.abs(axV[2]) < 0.9 ? [0, 0, 1] : [1, 0, 0]; u = vNorm(vCross(helper, axV)); }
    var vv = vNorm(vCross(axV, u));
    var full = Math.abs(span - 360) < 1e-9;
    for (k = 0; k < count; k++) {
      var ang = start + (full ? span * k / count : (count > 1 ? span * k / (count - 1) : 0));
      // 位置变换 = 绕轴（过 center）旋到该角度 · 先沿轴垂直基的 u 方向推出半径
      //   M = T(center) · R(axV, ang) · T(u·radius)
      var rot = m4rotAxis(axV, ang);
      var m = m4mul(m4translate(center), m4mul(rot, m4translate(vMul(u, radius))));
      out.push(meshTransform(mesh, m));
    }
    return meshMerge(out);
  }
  fail('未知阵列模式 ' + mode + '（可用 linear / circular / mirror）');
}

// 部件 → 网格（几何 + 布尔 ops + 变换 + 阵列）
function buildPartMesh(part, E, diag) {
  if (!isObj(part)) fail('部件必须是对象');
  var mesh = buildShape(part.shape, E, 0);
  var ops = part.ops === undefined ? [] : part.ops;
  if (!isArr(ops)) fail('部件的 ops 必须是数组');
  for (var k = 0; k < ops.length; k++) {
    var op = ops[k];
    if (!isObj(op)) fail('第 ' + k + ' 个 op 必须是对象');
    var name = op.op;
    if (name === 'union' || name === 'subtract' || name === 'intersect') {
      if (!op.shape) fail('op ' + name + ' 需要 shape');
      var other = buildShape(op.shape, E, 0);
      if (op.transform) other = meshTransform(other, m4fromTRS(evalTRS(op.transform, E)));
      mesh = meshBoolean(mesh, other, name);
    } else if (name === 'translate' || name === 'rotate' || name === 'scale' || name === 'transform') {
      mesh = meshTransform(mesh, m4fromTRS(evalTRS(name === 'transform' ? op : op, E)));
    } else if (name === 'mirror') {
      mesh = meshTransform(mesh, m4mirror(evalAxis(op.axis, E, 'axis', [1, 0, 0])));
    } else if (name === 'align') {
      mesh = meshAlign(mesh, op.to === undefined ? 'center' : op.to);
    } else if (name === 'snap') {
      mesh = meshSnap(mesh, evalNum(op.grid === undefined ? meshEpsilon(mesh) : op.grid, E, 'grid'));
    } else if (name === 'repair') {
      mesh = meshRepair(mesh).mesh;
    } else if (name === 'chamfer' || name === 'fillet') {
      // 圆角/倒角 = 网格级边重建（处理了几条凸边、跳过几条凹边，经 diag 如实上报）
      var st = null;
      if (diag) {
        st = {
          part: part.id === undefined ? '?' : part.id,
          kind: name,
          label: '部件 ' + (part.id === undefined ? '?' : part.id) + ' 的' + (name === 'chamfer' ? '倒角' : '圆角')
        };
        diag.push(st);
      }
      mesh = name === 'chamfer' ? meshChamfer(mesh, op, E, st) : meshFillet(mesh, op, E, st);
    } else {
      fail('未知 op ' + name + '（可用 union/subtract/intersect/translate/rotate/scale/mirror/align/snap/repair/chamfer/fillet）');
    }
  }
  if (part.transform) mesh = meshTransform(mesh, m4fromTRS(evalTRS(part.transform, E)));
  if (part.repeat) mesh = applyRepeat(mesh, part.repeat, E, '部件 ' + (part.id || '?') + ' 的阵列');
  return mesh;
}

// ── 二进制写出（沙箱无 Buffer：自研 IEEE754 + base64 编码）──────────
// 沙箱提供 ctx.fs.writeFileBase64（二进制安全），故 binary STL / GLB 都能产出。
// 单精度 IEEE754 手写而不依赖 TypedArray（沙箱兼容性优先）。
function f32Bytes(v) {
  var out = [0, 0, 0, 0];
  if (!isFinite(v)) {
    // ±Inf / NaN：按 float32 的 Inf 表示（几何里出现即为缺陷，由判据抓）
    out[2] = 0x80;
    out[3] = (v < 0 ? 0xFF : 0x7F);
    return out;
  }
  var sign = 0;
  if (v < 0) { sign = 0x80; v = -v; }
  if (v === 0) { out[3] = sign; return out; }
  var e = Math.floor(Math.log(v) / Math.LN2);
  var m = v / Math.pow(2, e);
  while (m >= 2) { m /= 2; e++; }
  while (m < 1) { m *= 2; e--; }
  var biased = e + 127;
  if (biased >= 255) { // 溢出 → Inf
    out[2] = 0x80; out[3] = sign | 0x7F; return out;
  }
  if (biased <= 0) { // 次正规数
    var sub = Math.round(v / Math.pow(2, -149));
    if (sub <= 0) { out[3] = sign; return out; }
    out[0] = sub & 255; out[1] = (sub >>> 8) & 255; out[2] = (sub >>> 16) & 15; out[3] = sign;
    return out;
  }
  var frac = Math.round((m - 1) * 8388608); // 2^23
  if (frac >= 8388608) { frac -= 8388608; biased++; if (biased >= 255) { out[2] = 0x80; out[3] = sign | 0x7F; return out; } }
  out[0] = frac & 255;
  out[1] = (frac >>> 8) & 255;
  out[2] = ((frac >>> 16) & 0x7F) | ((biased & 1) << 7);
  out[3] = sign | ((biased >>> 1) & 0x7F);
  return out;
}

var B64_CHARS = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/';
function b64EncodeBytes(bytes) {
  var out = [], i, len = bytes.length, CH = B64_CHARS;
  for (i = 0; i + 2 < len; i += 3) {
    var n = (bytes[i] << 16) | (bytes[i + 1] << 8) | bytes[i + 2];
    out.push(CH.charAt((n >>> 18) & 63), CH.charAt((n >>> 12) & 63), CH.charAt((n >>> 6) & 63), CH.charAt(n & 63));
  }
  var rem = len - i;
  if (rem === 1) {
    var n1 = bytes[i] << 16;
    out.push(CH.charAt((n1 >>> 18) & 63), CH.charAt((n1 >>> 12) & 63), '=', '=');
  } else if (rem === 2) {
    var n2 = (bytes[i] << 16) | (bytes[i + 1] << 8);
    out.push(CH.charAt((n2 >>> 18) & 63), CH.charAt((n2 >>> 12) & 63), CH.charAt((n2 >>> 6) & 63), '=');
  }
  return out.join('');
}

function byteWriter() {
  return {
    buf: [],
    u8: function (v) { this.buf.push(v & 255); },
    u16: function (v) { this.buf.push(v & 255, (v >>> 8) & 255); },
    u32: function (v) { this.buf.push(v & 255, (v >>> 8) & 255, (v >>> 16) & 255, (v >>> 24) & 255); },
    f32: function (v) { var b = f32Bytes(v); this.buf.push(b[0], b[1], b[2], b[3]); },
    bytes: function (arr) { for (var i = 0; i < arr.length; i++) this.buf.push(arr[i] & 255); },
    ascii: function (s) { for (var i = 0; i < s.length; i++) this.buf.push(s.charCodeAt(i) & 255); },
    pad: function (n) { for (var i = 0; i < n; i++) this.buf.push(0); },
    utf8: function (s) { this.bytes(utf8Bytes(s)); }
  };
}

// UTF-8 编码（沙箱无 TextEncoder；GLB 的 JSON 块与 glTF 文本都按 UTF-8 落字节）
function utf8Bytes(s) {
  var out = [];
  for (var i = 0; i < s.length; i++) {
    var c = s.charCodeAt(i);
    if (c < 0x80) out.push(c);
    else if (c < 0x800) out.push(0xC0 | (c >> 6), 0x80 | (c & 63));
    else if (c >= 0xD800 && c <= 0xDBFF && i + 1 < s.length) {
      var c2 = s.charCodeAt(i + 1);
      var cp = 0x10000 + ((c - 0xD800) << 10) + (c2 - 0xDC00);
      out.push(0xF0 | (cp >> 18), 0x80 | ((cp >> 12) & 63), 0x80 | ((cp >> 6) & 63), 0x80 | (cp & 63));
      i++;
    } else out.push(0xE0 | (c >> 12), 0x80 | ((c >> 6) & 63), 0x80 | (c & 63));
  }
  return out;
}

function triNormal(t) {
  var n = vNorm(vCross(vSub(t[1], t[0]), vSub(t[2], t[0])));
  return vFinite(n) && vLen(n) > 0.5 ? n : [0, 0, 1];
}

// ── STL（3D 打印事实标准）───────────────────────────────────
// binary：80 字节头（**不能以 "solid" 开头**，否则解析器会当 ASCII 读）+ uint32 面数 + 每面 50 字节
function stlBinaryBytes(mesh, name) {
  var w = byteWriter();
  var head = 'paircode tool-model | ' + (name || 'model') + ' | ' + mesh.indices.length + ' facets | unit=mm | z-up';
  var hb = [];
  for (var i = 0; i < 80; i++) hb.push(i < head.length ? (head.charCodeAt(i) & 0x7F) : 32);
  if (String.fromCharCode(hb[0], hb[1], hb[2], hb[3], hb[4]).toLowerCase() === 'solid') hb[0] = 0x50; // 'P'
  w.bytes(hb);
  w.u32(mesh.indices.length);
  for (var k = 0; k < mesh.indices.length; k++) {
    var t = meshTriangle(mesh, k), n = triNormal(t);
    w.f32(n[0]); w.f32(n[1]); w.f32(n[2]);
    for (var v = 0; v < 3; v++) { w.f32(t[v][0]); w.f32(t[v][1]); w.f32(t[v][2]); }
    w.u16(0);
  }
  return w.buf;
}

function stlAsciiText(mesh, name) {
  var nm = (name || 'model').replace(/[^A-Za-z0-9_\-]/g, '_');
  var out = ['solid ' + nm];
  for (var k = 0; k < mesh.indices.length; k++) {
    var t = meshTriangle(mesh, k), n = triNormal(t);
    out.push('  facet normal ' + fmtNum(n[0]) + ' ' + fmtNum(n[1]) + ' ' + fmtNum(n[2]));
    out.push('    outer loop');
    for (var v = 0; v < 3; v++) out.push('      vertex ' + fmtNum(t[v][0]) + ' ' + fmtNum(t[v][1]) + ' ' + fmtNum(t[v][2]));
    out.push('    endloop');
    out.push('  endfacet');
  }
  out.push('endsolid ' + nm);
  return out.join('\n') + '\n';
}

// ── OBJ（Wavefront，文本）──────────────────────────────────
function objText(meshes) {
  var out = ['# paircode tool-model 导出（单位 mm，坐标右手系 +z 向上）',
    '# ' + meshes.length + ' 个对象，' + meshes.reduce(function (s, p) { return s + p.mesh.indices.length; }, 0) + ' 个三角面'];
  var vOff = 1, nOff = 1, i, k;
  for (i = 0; i < meshes.length; i++) {
    var part = meshes[i], m = part.mesh;
    out.push('o ' + (part.id || ('part' + i)));
    for (k = 0; k < m.positions.length; k++) {
      out.push('v ' + fmtNum(m.positions[k][0]) + ' ' + fmtNum(m.positions[k][1]) + ' ' + fmtNum(m.positions[k][2]));
    }
    for (k = 0; k < m.indices.length; k++) {
      var t = meshTriangle(m, k), n = triNormal(t);
      out.push('vn ' + fmtNum(n[0]) + ' ' + fmtNum(n[1]) + ' ' + fmtNum(n[2]));
    }
    for (k = 0; k < m.indices.length; k++) {
      var f = m.indices[k], n2 = nOff + k;
      out.push('f ' + (vOff + f[0]) + '//' + n2 + ' ' + (vOff + f[1]) + '//' + n2 + ' ' + (vOff + f[2]) + '//' + n2);
    }
    vOff += m.positions.length;
    nOff += m.indices.length;
  }
  return out.join('\n') + '\n';
}

// ── glTF 2.0（Khronos 标准；导出时把工程的 z-up 转成规范要求的 **y-up**）──
// 变换：(x, y, z) → (x, z, −y)（绕 X 轴 −90°）
function toGltfVec(p) { return [p[0], p[2], -p[1]]; }
function hexToRgb01(hex) {
  var s = String(hex || '#b0b4bd').replace('#', '');
  if (s.length === 3) s = s.charAt(0) + s.charAt(0) + s.charAt(1) + s.charAt(1) + s.charAt(2) + s.charAt(2);
  if (!/^[0-9a-fA-F]{6}$/.test(s)) s = 'b0b4bd';
  return [parseInt(s.slice(0, 2), 16) / 255, parseInt(s.slice(2, 4), 16) / 255, parseInt(s.slice(4, 6), 16) / 255, 1];
}

// 返回 { json, binBytes }（binBytes 为拼接后的顶点/索引数据，供 data URI 或 GLB 使用）
function buildGltfData(parts, opts) {
  var o = opts || {};
  var bin = [], bufferViews = [], accessors = [], meshes = [], nodes = [], materials = [], matIndex = {};
  function pad4() { while (bin.length % 4 !== 0) bin.push(0); }
  for (var i = 0; i < parts.length; i++) {
    var part = parts[i], m = part.mesh;
    if (!m.indices.length) continue;
    var matKey = JSON.stringify(part.material || DEFAULT_MATERIAL);
    if (matIndex[matKey] === undefined) {
      matIndex[matKey] = materials.length;
      var mt = part.material || DEFAULT_MATERIAL;
      materials.push({
        name: (part.id || 'mat') + '_mat',
        doubleSided: false,
        pbrMetallicRoughness: {
          baseColorFactor: hexToRgb01(mt.color),
          metallicFactor: isNum(mt.metallic) ? mt.metallic : DEFAULT_MATERIAL.metallic,
          roughnessFactor: isNum(mt.roughness) ? mt.roughness : DEFAULT_MATERIAL.roughness
        }
      });
    }
    var mi = matIndex[matKey];
    // 顶点（POSITION + NORMAL：每三角独立顶点，法线为面法线 —— 与水密无关的展示网格）
    var min = [Infinity, Infinity, Infinity], max = [-Infinity, -Infinity, -Infinity];
    pad4();
    var posOff = bin.length, triCount = m.indices.length;
    for (var k = 0; k < triCount; k++) {
      var t = meshTriangle(m, k), n = toGltfVec(triNormal(t));
      for (var v = 0; v < 3; v++) {
        var gp = toGltfVec(t[v]);
        for (var c = 0; c < 3; c++) {
          var fb = f32Bytes(gp[c]);
          bin.push(fb[0], fb[1], fb[2], fb[3]);
          if (gp[c] < min[c]) min[c] = gp[c];
          if (gp[c] > max[c]) max[c] = gp[c];
        }
      }
    }
    var posLen = bin.length - posOff;
    bufferViews.push({ buffer: 0, byteOffset: posOff, byteLength: posLen, target: 34962 });
    var posAcc = accessors.length;
    accessors.push({ bufferView: bufferViews.length - 1, componentType: 5126, count: triCount * 3, type: 'VEC3', min: min, max: max });
    pad4();
    var nrmOff = bin.length;
    for (var k2 = 0; k2 < triCount; k2++) {
      var t2 = meshTriangle(m, k2), n2 = toGltfVec(triNormal(t2));
      var nb = [f32Bytes(n2[0]), f32Bytes(n2[1]), f32Bytes(n2[2])];
      for (var v2 = 0; v2 < 3; v2++) {
        bin.push(nb[0][0], nb[0][1], nb[0][2], nb[0][3], nb[1][0], nb[1][1], nb[1][2], nb[1][3], nb[2][0], nb[2][1], nb[2][2], nb[2][3]);
      }
    }
    var nrmLen = bin.length - nrmOff;
    bufferViews.push({ buffer: 0, byteOffset: nrmOff, byteLength: nrmLen, target: 34962 });
    var nrmAcc = accessors.length;
    accessors.push({ bufferView: bufferViews.length - 1, componentType: 5126, count: triCount * 3, type: 'VEC3' });
    // 索引（顶点数 < 65536 用 uint16，否则 uint32）
    pad4();
    var idxOff = bin.length, use16 = triCount * 3 < 65536;
    for (var k3 = 0; k3 < triCount * 3; k3++) {
      if (use16) bin.push(k3 & 255, (k3 >>> 8) & 255);
      else bin.push(k3 & 255, (k3 >>> 8) & 255, (k3 >>> 16) & 255, (k3 >>> 24) & 255);
    }
    var idxLen = bin.length - idxOff;
    bufferViews.push({ buffer: 0, byteOffset: idxOff, byteLength: idxLen, target: 34963 });
    var idxAcc = accessors.length;
    accessors.push({ bufferView: bufferViews.length - 1, componentType: use16 ? 5123 : 5125, count: triCount * 3, type: 'SCALAR' });
    var meshIdx = meshes.length;
    meshes.push({
      name: part.id || ('part' + i),
      primitives: [{ attributes: { POSITION: posAcc, NORMAL: nrmAcc }, indices: idxAcc, material: mi, mode: 4 }]
    });
    nodes.push({ name: part.id || ('part' + i), mesh: meshIdx });
  }
  if (!nodes.length) fail('没有可导出的几何（所有部件为空或不可见）');
  pad4();
  var json = {
    asset: { version: GLTF_VERSION, generator: 'paircode tool-model (goja)', copyright: '' },
    scene: 0,
    scenes: [{ name: (o.name || 'model'), nodes: nodes.map(function (_, i2) { return i2; }) }],
    nodes: nodes,
    meshes: meshes,
    materials: materials,
    accessors: accessors,
    bufferViews: bufferViews,
    buffers: [{ byteLength: bin.length, uri: 'data:application/octet-stream;base64,' + b64EncodeBytes(bin) }]
  };
  return { json: json, binBytes: bin, gltf: json };
}

function gltfText(parts, opts) {
  var d = buildGltfData(parts, opts);
  return JSON.stringify(d.json, null, 2) + '\n';
}

// GLB（单文件二进制容器：12 字节头 + JSON 块 + BIN 块，块按 4 字节对齐）
function glbBytes(parts, opts) {
  var d = buildGltfData(parts, opts);
  var jsonBytes = utf8Bytes(JSON.stringify(d.json));
  while (jsonBytes.length % 4 !== 0) jsonBytes.push(0x20); // JSON 块用空格补齐
  var binBytes = d.binBytes.slice();
  while (binBytes.length % 4 !== 0) binBytes.push(0);
  var total = 12 + 8 + jsonBytes.length + 8 + binBytes.length;
  var w = byteWriter();
  w.u32(0x46546C67); // 'glTF'
  w.u32(2);
  w.u32(total);
  w.u32(jsonBytes.length); w.u32(0x4E4F534A); w.bytes(jsonBytes); // 'JSON'
  w.u32(binBytes.length); w.u32(0x004E4942); w.bytes(binBytes);   // 'BIN\0'
  return w.buf;
}

// ── 工程文档（参数化 CAD 工程的文本真相源）──────────────────────
function emptyModelDoc(name) {
  return {
    format: FORMAT,
    name: name || 'model',
    unit: 'mm',
    up: 'z',
    note: '坐标：右手系 +z 向上（CAD/STL 惯例；导出 glTF 时按规范转 y-up）。长度单位 mm。',
    params: [],
    parts: []
  };
}

function newParam(id, value, name, min, max) {
  var p = { id: id, value: value };
  if (name !== undefined) p.name = name;
  if (min !== undefined) p.min = min;
  if (max !== undefined) p.max = max;
  return p;
}

// 参数依赖解析：参数值本身可以是表达式（引用其它参数），故按"迭代求值直到无进展"解析，
// 并区分"引用了尚未解析的参数"（pending，继续等下一轮）与"引用了不存在的参数"（立即报错）。
function resolveParams(model) {
  var list = model.params === undefined ? [] : model.params;
  if (!isArr(list)) fail('params 必须是数组');
  if (list.length > LIMIT_PARAMS) fail('参数数量超过上限 ' + LIMIT_PARAMS);
  var byId = {}, order = [], k;
  for (k = 0; k < list.length; k++) {
    var p = list[k];
    if (!isObj(p)) fail('params[' + k + '] 必须是对象');
    if (!isStr(p.id) || !p.id) fail('params[' + k + '] 缺少 id');
    if (!/^[A-Za-z_][A-Za-z0-9_]*$/.test(p.id)) fail('参数 id 必须以字母/下划线开头且只含字母数字下划线：' + p.id);
    if (byId[p.id]) fail('参数 id 重复：' + p.id);
    byId[p.id] = p;
    order.push(p.id);
  }
  var values = {}, done = {};
  function attempt() {
    var progress = false;
    for (var i = 0; i < order.length; i++) {
      var id = order[i];
      if (done[id]) continue;
      var p2 = byId[id];
      try {
        values[id] = isStr(p2.value) ? evalMathExpr(p2.value, values, byId) : toNumber(p2.value, '参数 ' + id + ' 的 value');
        done[id] = true;
        progress = true;
      } catch (e) {
        if (e && e.pending) continue; // 等下一轮
        throw e;
      }
    }
    return progress;
  }
  for (var round = 0; round <= order.length + 1; round++) {
    if (!attempt()) break;
  }
  var missing = [];
  for (k = 0; k < order.length; k++) if (!done[order[k]]) missing.push(order[k]);
  if (missing.length) fail('参数无法解析（循环引用或引用不存在的参数）：' + missing.join(', '));
  for (k = 0; k < order.length; k++) {
    if (!isFinite(values[order[k]])) fail('参数 ' + order[k] + ' 的值不是有限数');
  }
  return { values: values, known: byId, list: list, order: order };
}

function mergeMaterial(mat) {
  var src = isObj(mat) ? mat : {};
  return {
    color: isStr(src.color) ? src.color : DEFAULT_MATERIAL.color,
    metallic: isNum(src.metallic) ? src.metallic : (isNum(src.metalness) ? src.metalness : DEFAULT_MATERIAL.metallic),
    roughness: isNum(src.roughness) ? src.roughness : DEFAULT_MATERIAL.roughness
  };
}

// 全模型构建：参数解析 → 逐部件建网格（含 ops / 变换 / 阵列）→ 统计
function buildModelMeshes(model, opts) {
  var o = opts || {};
  if (!isObj(model)) fail('工程文档必须是对象');
  var partsRaw = model.parts === undefined ? [] : model.parts;
  if (!isArr(partsRaw)) fail('parts 必须是数组');
  if (partsRaw.length > LIMIT_PARTS) fail('部件数量超过上限 ' + LIMIT_PARTS);
  var rp = resolveParams(model);
  var E = { params: rp.values, known: rp.known };
  var ids = {}, parts = [], i, k;
  for (i = 0; i < partsRaw.length; i++) {
    var p = partsRaw[i];
    if (!isObj(p)) fail('parts[' + i + '] 必须是对象');
    if (!isStr(p.id) || !p.id) fail('parts[' + i + '] 缺少 id');
    if (ids[p.id]) fail('部件 id 重复：' + p.id);
    ids[p.id] = 1;
    if (o.skipInvisible !== false && p.visible === false) continue;
    var mesh = buildPartMesh(p, E, o.diag);
    if (mesh.indices.length > LIMIT_TRIS_PART) {
      fail('部件 ' + p.id + ' 三角面数 ' + mesh.indices.length + ' 超过单部件上限 ' + LIMIT_TRIS_PART + '（降低 segments 或拆部件）');
    }
    parts.push({
      id: p.id,
      name: p.name === undefined ? p.id : p.name,
      mesh: mesh,
      material: mergeMaterial(p.material),
      source: p,
      tris: mesh.indices.length,
      vertices: mesh.positions.length,
      volume: meshVolume(mesh),
      area: meshArea(mesh),
      bbox: meshBbox(mesh)
    });
  }
  var totalTris = 0;
  for (k = 0; k < parts.length; k++) totalTris += parts[k].tris;
  if (totalTris > LIMIT_TRIS_TOTAL) fail('全模型三角面数 ' + totalTris + ' 超过上限 ' + LIMIT_TRIS_TOTAL);
  var merged = parts.length ? meshMerge(parts.map(function (x) { return x.mesh; })) : meshNew([], []);
  return { E: E, params: rp, parts: parts, merged: merged, totalTris: totalTris };
}

// ── 9 项判据（M1–M9）────────────────────────────────────────
// M4 口径（实测结论，有 JSCAD 对照）：BSP 三角网格布尔的输出在"严格边匹配"意义下通常不水密，
// 故分三级报告：严格水密 / 只有 T 型接缝（几何闭合、分段不同）/ 存在真缺口。
var CHECK_META = [
  { id: 'M1', title: '工程契约（format / 单位 / 部件结构）', level: 'fail' },
  { id: 'M2', title: '参数与表达式', level: 'fail' },
  { id: 'M3', title: '网格完整性（索引 / 有限性 / 退化面）', level: 'fail' },
  { id: 'M4', title: '水密性（可 3D 打印）', level: 'warn' },
  { id: 'M5', title: '绕向一致与外向', level: 'fail' },
  { id: 'M6', title: '实体性（体积 / 包围盒 / 连通）', level: 'warn' },
  { id: 'M7', title: '变换与尺度', level: 'warn' },
  { id: 'M8', title: '导出契约（glTF / STL / OBJ 结构）', level: 'warn' },
  { id: 'M9', title: '产物确定且自包含', level: 'warn' }
];

function mkCheck(id, ok, detail, items, level) {
  var meta = null;
  for (var i = 0; i < CHECK_META.length; i++) if (CHECK_META[i].id === id) meta = CHECK_META[i];
  return {
    id: id,
    title: meta ? meta.title : id,
    ok: !!ok,
    level: level || (meta ? meta.level : 'warn'),
    detail: detail || '',
    items: items || []
  };
}

function checkM1(model) {
  var items = [];
  var parts = model.parts === undefined ? [] : model.parts;
  if (model.format !== FORMAT) items.push('format 必须是 ' + FORMAT + '（当前 ' + JSON.stringify(model.format) + '）');
  if (model.unit !== undefined && model.unit !== 'mm') items.push('unit 只支持 mm（当前 ' + model.unit + '）——其他单位请先换算');
  if (model.up !== undefined && model.up !== 'z') items.push('up 必须是 z（CAD 惯例；导出 glTF 时按规范转 y-up）');
  if (isStr(model.name) && model.name.length > 64) items.push('name 过长（>64 字符）');
  if (!isArr(parts)) items.push('parts 必须是数组');
  else {
    var seen = {};
    for (var k = 0; k < parts.length; k++) {
      var p = parts[k];
      if (!isObj(p)) { items.push('parts[' + k + '] 不是对象'); continue; }
      if (!isStr(p.id) || !p.id) items.push('parts[' + k + '] 缺少 id');
      else if (seen[p.id]) items.push('部件 id 重复：' + p.id);
      else seen[p.id] = 1;
      if (p.shape === undefined) items.push('部件 ' + p.id + ' 缺少 shape');
      else if (!isObj(p.shape)) items.push('部件 ' + p.id + ' 的 shape 必须是对象');
      else if (p.shape.type !== undefined && SHAPE_TYPES.indexOf(p.shape.type) < 0) items.push('部件 ' + p.id + ' 的 shape.type 未知：' + p.shape.type);
      if (p.ops !== undefined && !isArr(p.ops)) items.push('部件 ' + p.id + ' 的 ops 必须是数组');
      if (p.repeat !== undefined && (!isObj(p.repeat) || ['linear', 'circular', 'mirror'].indexOf(p.repeat.mode === undefined ? 'linear' : p.repeat.mode) < 0)) {
        items.push('部件 ' + p.id + ' 的 repeat.mode 必须是 linear / circular / mirror');
      }
    }
  }
  return mkCheck('M1', items.length === 0, items.length ? '发现 ' + items.length + ' 处结构问题' : ('结构合法（' + parts.length + ' 个部件）'), items);
}

function checkM2(model, rp) {
  var items = [], info = [];
  var list = rp.list;
  for (var k = 0; k < list.length; k++) {
    var p = list[k], v = rp.values[p.id];
    if (p.min !== undefined && v < p.min) items.push('参数 ' + p.id + ' = ' + fmtNum(v) + ' 小于声明下限 ' + p.min);
    if (p.max !== undefined && v > p.max) items.push('参数 ' + p.id + ' = ' + fmtNum(v) + ' 大于声明上限 ' + p.max);
  }
  // 统计参数引用次数（哪些参数真的在驱动几何）
  var text = JSON.stringify(model);
  for (k = 0; k < list.length; k++) {
    var re = new RegExp('"' + list[k].id + '"', 'g');
    var hits = 0, m2;
    while ((m2 = re.exec(text)) !== null) hits++;
    if (hits <= 1) info.push('参数 ' + list[k].id + ' 未被几何引用（只出现在定义处）');
  }
  return mkCheck('M2', items.length === 0,
    items.length ? '参数越界 ' + items.length + ' 处' : ('参数全部可解析（' + list.length + ' 项）' + (info.length ? '；' + info.length + ' 项未被引用' : '')),
    items.concat(info));
}

function checkM3(built) {
  var items = [];
  for (var k = 0; k < built.parts.length; k++) {
    var p = built.parts[k], m = p.mesh, nonfin = 0, bad = 0, degen = 0, i;
    for (i = 0; i < m.positions.length; i++) if (!vFinite(m.positions[i])) nonfin++;
    for (i = 0; i < m.indices.length; i++) {
      var f = m.indices[i];
      if (f[0] < 0 || f[1] < 0 || f[2] < 0 || f[0] >= m.positions.length || f[1] >= m.positions.length || f[2] >= m.positions.length) { bad++; continue; }
      var t = meshTriangle(m, i);
      if (vLen(vCross(vSub(t[1], t[0]), vSub(t[2], t[0]))) / 2 <= EPS_AREA) degen++;
    }
    if (!m.indices.length) items.push('部件 ' + p.id + '：几何为空（0 个三角面）');
    if (nonfin) items.push('部件 ' + p.id + '：' + nonfin + ' 个顶点坐标非有限数');
    if (bad) items.push('部件 ' + p.id + '：' + bad + ' 个三角面索引越界');
    if (degen) items.push('部件 ' + p.id + '：' + degen + ' 个退化（零面积）三角面');
  }
  return mkCheck('M3', items.length === 0,
    items.length ? '发现 ' + items.length + ' 处网格问题' : ('全部 ' + built.parts.length + ' 个部件网格完整，合计 ' + built.totalTris + ' 个三角面'), items);
}

function checkM4(built) {
  var items = [], gaps = 0, tjs = 0, strict = 0, hals = 0;
  for (var k = 0; k < built.parts.length; k++) {
    var p = built.parts[k];
    var r = meshWatertightReport(p.mesh);
    strict += r.strictUnpaired; tjs += r.tJunctions; gaps += r.tolerantUnpaired; hals += r.halfEdges;
    if (!r.watertightStrict && r.tolerantUnpaired > 0) {
      items.push('部件 ' + p.id + '：未配对半边 ' + r.strictUnpaired + '（T 缝 ' + r.tJunctions + ' / 真缺口 ' + r.tolerantUnpaired + '）—— 建议 op=repair 或提高 segments 后复验');
    }
  }
  var detail;
  if (strict === 0) detail = '全部部件严格水密（有向半边全部配对）';
  else if (gaps === 0) detail = '几何闭合；存在 ' + tjs + ' 条 T 型接缝（相邻面片分段不同）—— 主流切片器按容差处理，可直接打印';
  else detail = '存在 ' + gaps + ' 条真缺口（未配对半边 ' + strict + ' / ' + hals + '）—— 3D 打印前建议修复';
  return mkCheck('M4', gaps === 0, detail, items);
}

function checkM5(built) {
  var items = [], nmTotal = 0, flipTotal = 0, halfTotal = 0, k;
  for (k = 0; k < built.parts.length; k++) {
    var p = built.parts[k], t = meshTopology(p.mesh);
    nmTotal += t.nonManifoldEdges;
    flipTotal += t.flippedEdgePairs;
    halfTotal += t.faces * 3;
    if (p.volume < 0) items.push('部件 ' + p.id + '：有向体积为负（法线整体朝内）');
  }
  // 判失败的门槛：
  //   · 有向体积为负 → 面整体朝内（真问题）
  //   · 共边同向占比 > 1% → 绕向大面积不一致（真问题）
  // 少量（实测 <0.02%）共边同向与非流形边是连续布尔在碎片交接处的副产物，不影响体积与切片
  // 打印，故作为提示写入 detail —— 判据只报事实与比例，不夸大也不掩盖。
  var flipRatio = halfTotal ? flipTotal / halfTotal : 0;
  if (flipRatio > 0.01) {
    items.push('共边同向 ' + flipTotal + ' 对（占半边 ' + fmtNum(flipRatio * 100, 2) + '% > 1%）—— 绕向大面积不一致：检查负缩放或面被翻转');
  }
  var detail = items.length ? '发现 ' + items.length + ' 处绕向问题'
    : ('全部部件绕向一致且法线朝外（有向体积 > 0）' +
      (flipTotal || nmTotal ? '；另有 ' + flipTotal + ' 对共边同向 / ' + nmTotal + ' 条非流形边（占半边 ' + fmtNum(flipRatio * 100, 3) + '%，连续布尔累积，不影响体积与打印）' : ''));
  return mkCheck('M5', items.length === 0, detail, items);
}

function checkM6(built) {
  var items = [], note = [];
  for (var k = 0; k < built.parts.length; k++) {
    var p = built.parts[k], bb = p.bbox;
    if (!(p.volume > 0)) items.push('部件 ' + p.id + '：体积为 ' + fmtNum(p.volume) + ' mm³（应为正）');
    for (var c = 0; c < 3; c++) {
      if (!(bb[1][c] > bb[0][c])) items.push('部件 ' + p.id + '：包围盒在 ' + 'xyz'.charAt(c) + ' 方向无厚度');
    }
    var bbv = (bb[1][0] - bb[0][0]) * (bb[1][1] - bb[0][1]) * (bb[1][2] - bb[0][2]);
    if (bbv > 0 && p.volume / bbv > 1.0000001) {
      items.push('部件 ' + p.id + '：体积是包围盒的 ' + fmtNum(p.volume / bbv, 3) + ' 倍（>1 不可能）→ 疑似自交 / 面翻转');
    }
    var comp = meshComponents(p.mesh);
    if (comp.count > 1) note.push('部件 ' + p.id + ' 含 ' + comp.count + ' 个连通体（阵列或多实体属正常）');
  }
  return mkCheck('M6', items.length === 0,
    items.length ? '发现 ' + items.length + ' 处实体性问题' : ('全部部件体积为正且包围盒有效' + (note.length ? '（' + note.join('；') + '）' : '')),
    items.concat(note));
}

function checkM7(model, built) {
  var items = [], maxDiag = 0, minDiag = Infinity, k;
  for (k = 0; k < built.parts.length; k++) {
    var sz = meshSize(built.parts[k].mesh);
    var diag = Math.sqrt(sz[0] * sz[0] + sz[1] * sz[1] + sz[2] * sz[2]);
    if (diag > maxDiag) maxDiag = diag;
    if (diag > 0 && diag < minDiag) minDiag = diag;
    if (sz[0] <= 0 || sz[1] <= 0 || sz[2] <= 0) {
      items.push('部件 ' + built.parts[k].id + ' 尺寸为 ' + fmtNum(sz[0]) + '×' + fmtNum(sz[1]) + '×' + fmtNum(sz[2]) + ' mm（存在零厚度方向）');
    }
  }
  if (maxDiag > 100000) items.push('全模型对角线 ' + fmtNum(maxDiag, 1) + ' mm 过大（>100 m）—— 确认单位应为 mm');
  if (maxDiag > 0 && maxDiag < 0.05) items.push('全模型对角线 ' + fmtNum(maxDiag, 4) + ' mm 过小（<0.05 mm）—— 确认单位 / 参数');
  return mkCheck('M7', items.length === 0,
    items.length ? '发现 ' + items.length + ' 处尺度问题' : ('尺度合理：最大对角线 ' + fmtNum(maxDiag, 3) + ' mm' + (minDiag < Infinity ? '，最小 ' + fmtNum(minDiag, 3) + ' mm' : '')), items);
}

function checkM8(products) {
  var items = [];
  for (var k = 0; k < products.length; k++) {
    var pr = products[k];
    if (pr.kind === 'gltf' && pr.text) {
      var okJson = false, g = null;
      try { g = JSON.parse(pr.text); okJson = true; } catch (e) { items.push(pr.path + '：glTF JSON 无法解析'); }
      if (okJson) {
        if (!g.asset || g.asset.version !== '2.0') items.push(pr.path + '：asset.version 必须是 "2.0"');
        if (!g.buffers || !g.buffers.length) items.push(pr.path + '：缺少 buffers');
        else {
          for (var b = 0; b < g.buffers.length; b++) {
            if (!/^data:application\/octet-stream;base64,/.test(g.buffers[b].uri || '')) items.push(pr.path + '：buffer uri 必须是自包含的 base64 data URI');
          }
        }
        if (!g.accessors || !g.accessors.length) items.push(pr.path + '：缺少 accessors');
        else {
          // 只校验 primitives 真正引用的 POSITION accessor（NORMAL 同为 VEC3/5126，但规范不要求 min/max）
          var posIdx = {};
          if (g.meshes) {
            for (var mi = 0; mi < g.meshes.length; mi++) {
              var prims = g.meshes[mi].primitives || [];
              for (var pi = 0; pi < prims.length; pi++) {
                if (prims[pi].attributes && prims[pi].attributes.POSITION !== undefined) posIdx[prims[pi].attributes.POSITION] = 1;
              }
            }
          }
          for (var a = 0; a < g.accessors.length; a++) {
            var acc = g.accessors[a];
            if ([5120, 5121, 5122, 5123, 5125, 5126].indexOf(acc.componentType) < 0) items.push(pr.path + '：accessor[' + a + '] componentType 非法');
            if (acc.count === undefined || !(acc.count > 0)) items.push(pr.path + '：accessor[' + a + '] count 非法');
            if (posIdx[a]) {
              if (acc.type !== 'VEC3' || acc.componentType !== 5126) items.push(pr.path + '：POSITION accessor[' + a + '] 必须是 VEC3/float32');
              if (!(acc.min && acc.max)) items.push(pr.path + '：POSITION accessor[' + a + '] 缺少 min/max（规范要求）');
            }
          }
        }
        if (g.bufferViews) {
          for (var bv = 0; bv < g.bufferViews.length; bv++) {
            if ((g.bufferViews[bv].byteOffset || 0) % 4 !== 0) items.push(pr.path + '：bufferView[' + bv + '] byteOffset 未 4 字节对齐');
          }
        }
        if (!g.meshes || !g.meshes.length) items.push(pr.path + '：缺少 meshes');
        if (!g.scenes || !g.scenes.length) items.push(pr.path + '：缺少 scenes');
      }
    } else if (pr.kind === 'stlb' && pr.bytes) {
      // 面数从 STL 头里读（校验"头部声明"与"实际长度"自洽）
      var n = ((pr.bytes[80] | (pr.bytes[81] << 8) | (pr.bytes[82] << 16) | (pr.bytes[83] << 24)) >>> 0);
      if (pr.bytes.length !== 84 + 50 * n) items.push(pr.path + '：头部声明 ' + n + ' 个三角面，但长度 ' + pr.bytes.length + ' ≠ 84+50×' + n + '=' + (84 + 50 * n));
      if (pr.faces !== undefined && pr.faces !== n) items.push(pr.path + '：头部面数 ' + n + ' 与导出面数 ' + pr.faces + ' 不一致');
      var magic = String.fromCharCode(pr.bytes[0], pr.bytes[1], pr.bytes[2], pr.bytes[3], pr.bytes[4]).toLowerCase();
      if (magic === 'solid') items.push(pr.path + '：binary STL 头以 "solid" 开头会被解析器误判为 ASCII');
    } else if (pr.kind === 'stla' && pr.text) {
      if (!/^solid /.test(pr.text)) items.push(pr.path + '：ASCII STL 必须以 "solid " 开头');
      if (!/endsolid [A-Za-z0-9_\-]*\s*$/.test(pr.text)) items.push(pr.path + '：ASCII STL 缺少 endsolid 结尾');
      var facets = (pr.text.match(/facet normal/g) || []).length;
      if (pr.faces !== undefined && facets !== pr.faces) items.push(pr.path + '：facet 数 ' + facets + ' ≠ 三角面数 ' + pr.faces);
    } else if (pr.kind === 'obj' && pr.text) {
      var vc = (pr.text.match(/^v /gm) || []).length;
      var fc = (pr.text.match(/^f /gm) || []).length;
      if (pr.faces !== undefined && fc !== pr.faces) items.push(pr.path + '：f 行数 ' + fc + ' ≠ 三角面数 ' + pr.faces);
      var maxIdx = 0, lines = pr.text.split('\n');
      for (var li = 0; li < lines.length; li++) {
        if (lines[li].indexOf('f ') !== 0) continue;
        var segs = lines[li].slice(2).trim().split(/\s+/);
        for (var si = 0; si < segs.length; si++) {
          var iv = Number(segs[si].split('/')[0]);
          if (!(iv >= 1)) items.push(pr.path + '：OBJ 索引不是 1 基正整数（' + segs[si] + '）');
          if (iv > maxIdx) maxIdx = iv;
        }
      }
      if (maxIdx > vc) items.push(pr.path + '：OBJ 索引越界（最大 ' + maxIdx + ' > 顶点数 ' + vc + '）');
    }
  }
  return mkCheck('M8', items.length === 0, items.length ? '产物契约问题 ' + items.length + ' 处' : ('已校验 ' + products.length + ' 个产物（glTF / STL / OBJ 结构合法）'), items);
}

function checkM9(products, determinism) {
  var items = [];
  for (var k = 0; k < products.length; k++) {
    var pr = products[k];
    if (pr.text && /https?:\/\//.test(pr.text)) items.push(pr.path + '：含外部 URL 引用（产物必须自包含）');
  }
  if (determinism && determinism.checked && !determinism.same) items.push('两次构建的产物不一致（长度 ' + determinism.len1 + ' vs ' + determinism.len2 + '）—— 存在非确定性来源');
  return mkCheck('M9', items.length === 0,
    items.length ? '自包含 / 确定性检查未通过' : ('产物自包含' + (determinism && determinism.checked ? '，两次构建位级一致（' + determinism.len1 + ' 字节）' : '')), items);
}

function runChecks(model, built, opts) {
  var o = opts || {};
  var checks = [
    checkM1(model),
    checkM2(model, built.params),
    checkM3(built),
    checkM4(built),
    checkM5(built),
    checkM6(built),
    checkM7(model, built),
    checkM8(o.products || []),
    checkM9(o.products || [], o.determinism)
  ];
  var fails = 0, warns = 0, k;
  for (k = 0; k < checks.length; k++) {
    if (checks[k].ok) continue;
    if (checks[k].level === 'fail') fails++; else warns++;
  }
  return { checks: checks, fails: fails, warns: warns, ok: fails === 0 && warns === 0 };
}

// ── 自包含可交互预览 HTML（自研 WebGL 渲染器，无外部依赖）──────────
// 只依赖工具侧算好的几何数据（世界坐标已应用），故不需要在浏览器里重跑内核。
function previewHtml(built, model, opts) {
  var o = opts || {};
  var partsData = [], k, i;
  for (k = 0; k < built.parts.length; k++) {
    var p = built.parts[k], m = p.mesh;
    var pos = [], idx = [];
    for (i = 0; i < m.positions.length; i++) {
      pos.push(cleanNum(m.positions[i][0], 6), cleanNum(m.positions[i][1], 6), cleanNum(m.positions[i][2], 6));
    }
    for (i = 0; i < m.indices.length; i++) idx.push(m.indices[i][0], m.indices[i][1], m.indices[i][2]);
    partsData.push({
      id: p.id, name: p.name, color: p.material.color,
      pos: pos, idx: idx, tris: p.tris, vol: cleanNum(p.volume, 3)
    });
  }
  var bb = built.parts.length ? meshBbox(built.merged) : [[0, 0, 0], [0, 0, 0]];
  var paramRows = [];
  for (k = 0; k < built.params.order.length; k++) {
    var pid = built.params.order[k], pdef = built.params.known[pid];
    paramRows.push({
      id: pid, name: pdef.name === undefined ? pid : pdef.name, value: cleanNum(built.params.values[pid], 6),
      range: (pdef.min === undefined && pdef.max === undefined) ? null : (pdef.min + '..' + pdef.max)
    });
  }
  var data = {
    name: model.name || 'model',
    unit: model.unit || 'mm',
    up: model.up || 'z',
    bbox: [[cleanNum(bb[0][0], 6), cleanNum(bb[0][1], 6), cleanNum(bb[0][2], 6)], [cleanNum(bb[1][0], 6), cleanNum(bb[1][1], 6), cleanNum(bb[1][2], 6)]],
    totalTris: built.totalTris,
    totalVolume: cleanNum(built.parts.reduce(function (s, x) { return s + x.volume; }, 0), 3),
    parts: partsData,
    params: paramRows
  };
  var L = [];
  L.push('<!DOCTYPE html>');
  L.push('<html lang="zh-CN"><head><meta charset="utf-8">');
  L.push('<meta name="viewport" content="width=device-width,initial-scale=1">');
  L.push('<title>' + String(data.name).replace(/[<>&]/g, '') + ' · 3D 预览</title>');
  L.push('<style>');
  L.push('html,body{margin:0;height:100%;overflow:hidden;background:#14161a;color:#e6e8ec;');
  L.push('font:12px/1.5 ui-sans-serif,system-ui,"Segoe UI","Noto Sans SC",sans-serif}');
  L.push('#c{display:block;width:100vw;height:100vh}');
  L.push('#hud{position:fixed;left:12px;top:12px;max-width:330px;background:rgba(20,22,26,.82);');
  L.push('border:1px solid #2b3038;border-radius:10px;padding:10px 12px;backdrop-filter:blur(6px)}');
  L.push('#hud h1{margin:0 0 6px;font-size:13px;font-weight:600;letter-spacing:.02em}');
  L.push('.kv{display:flex;justify-content:space-between;gap:10px;padding:1px 0;color:#a9b0bd}');
  L.push('.kv b{color:#e6e8ec;font-weight:500;font-variant-numeric:tabular-nums}');
  L.push('.sec{margin-top:8px;padding-top:8px;border-top:1px solid #262b33}');
  L.push('.t{color:#7f8896;font-size:11px;margin-bottom:3px;text-transform:uppercase;letter-spacing:.06em}');
  L.push('.chip{display:inline-flex;align-items:center;gap:5px;margin:2px 6px 2px 0}');
  L.push('.dot{width:9px;height:9px;border-radius:2px;box-shadow:0 0 0 1px rgba(255,255,255,.14) inset}');
  L.push('.hint{position:fixed;right:12px;bottom:12px;color:#7f8896;text-align:right;font-size:11px}');
  L.push('code{color:#e6e8ec;background:#22262d;padding:1px 4px;border-radius:4px}');
  L.push('</style></head><body>');
  L.push('<canvas id="c"></canvas>');
  L.push('<div id="hud"></div>');
  L.push('<div class="hint">拖拽=旋转 · 滚轮=缩放 · 右键拖拽=平移 · <code>W</code>=线框 · <code>R</code>=复位</div>');
  L.push('<script>');
  L.push('var MODEL=' + JSON.stringify(data) + ';');
  L.push(PREVIEW_JS);
  L.push('</scr' + 'ipt></body></html>');
  return L.join('\n') + '\n';
}

// 预览页里的渲染器（内联字符串：WebGL1 + 轨道相机 + Lambert 光照 + 线框 + HUD）
var PREVIEW_JS = [
  '(function(){',
  'var D=MODEL, cv=document.getElementById("c"), hud=document.getElementById("hud");',
  'var gl=cv.getContext("webgl")||cv.getContext("experimental-webgl");',
  'if(!gl){hud.innerHTML="<h1>浏览器不支持 WebGL</h1><div class=kv>请用现代浏览器打开此文件</div>";return}',
  // ── 着色器
  'var VS=["attribute vec3 aPos;attribute vec3 aNrm;uniform mat4 uProj,uView;varying vec3 vN,vW;",',
  ' "void main(){vN=aNrm;vW=aPos;gl_Position=uProj*uView*vec4(aPos,1.0);}"].join("\\n");',
  'var FS=["precision mediump float;varying vec3 vN,vW;uniform vec3 uColor,uCam;uniform float uWire;",',
  ' "void main(){if(uWire>0.5){gl_FragColor=vec4(uColor*1.35,1.0);return;}",',
  ' "vec3 n=normalize(vN);vec3 l1=normalize(vec3(0.45,0.75,0.55)),l2=normalize(vec3(-0.6,-0.35,-0.7));",',
  ' "float d=max(dot(n,l1),0.0)*0.8+max(dot(n,l2),0.0)*0.22;vec3 v=normalize(uCam-vW);",',
  ' "vec3 h=normalize(l1+v);float s=pow(max(dot(n,h),0.0),36.0)*0.28;",',
  ' "vec3 amb=vec3(0.26,0.28,0.32);gl_FragColor=vec4(uColor*(amb+d)+vec3(s),1.0);}"].join("\\n");',
  'function sh(t,src){var s=gl.createShader(t);gl.shaderSource(s,src);gl.compileShader(s);',
  ' if(!gl.getShaderParameter(s,gl.COMPILE_STATUS)){var lg=gl.getShaderInfoLog(s);',
  '  hud.innerHTML="<h1>着色器编译失败</h1><div class=kv>"+String(lg).replace(/[<>]/g,"")+"</div>";',
  '  throw new Error(lg)}return s}',
  'var prog=gl.createProgram();gl.attachShader(prog,sh(gl.VERTEX_SHADER,VS));gl.attachShader(prog,sh(gl.FRAGMENT_SHADER,FS));',
  'gl.linkProgram(prog);gl.useProgram(prog);',
  'var aPos=gl.getAttribLocation(prog,"aPos"),aNrm=gl.getAttribLocation(prog,"aNrm");',
  'var uProj=gl.getUniformLocation(prog,"uProj"),uView=gl.getUniformLocation(prog,"uView"),',
  ' uColor=gl.getUniformLocation(prog,"uColor"),uCam=gl.getUniformLocation(prog,"uCam"),uWire=gl.getUniformLocation(prog,"uWire");',
  'gl.enableVertexAttribArray(aPos);gl.enableVertexAttribArray(aNrm);',
  'gl.enable(gl.DEPTH_TEST);gl.enable(gl.CULL_FACE);gl.cullFace(gl.BACK);',
  // ── 每部件：顶点缓冲（位置 + 面法线）+ 三角索引 + 线框索引
  'function hex(c){c=(c||"#b0b4bd").replace("#","");if(c.length===3)c=c[0]+c[0]+c[1]+c[1]+c[2]+c[2];',
  ' var n=parseInt(c,16);if(isNaN(n))n=0xb0b4bd;return [((n>>16)&255)/255,((n>>8)&255)/255,(n&255)/255]}',
  'var meshes=[];',
  'for(var i=0;i<D.parts.length;i++){var p=D.parts[i],pos=p.pos,idx=p.idx,nv=pos.length/3;',
  ' var nrm=new Float32Array(nv*3);',
  ' for(var f=0;f<idx.length;f+=3){var i0=idx[f]*3,i1=idx[f+1]*3,i2=idx[f+2]*3;',
  '  var ax=pos[i1]-pos[i0],ay=pos[i1+1]-pos[i0+1],az=pos[i1+2]-pos[i0+2];',
  '  var bx=pos[i2]-pos[i0],by=pos[i2+1]-pos[i0+1],bz=pos[i2+2]-pos[i0+2];',
  '  var nx=ay*bz-az*by,ny=az*bx-ax*bz,nz=ax*by-ay*bx;var ln=Math.sqrt(nx*nx+ny*ny+nz*nz)||1;nx/=ln;ny/=ln;nz/=ln;',
  '  var tri=[idx[f],idx[f+1],idx[f+2]];',
  '  for(var v=0;v<3;v++){var o=tri[v];nrm[o*3]=nx;nrm[o*3+1]=ny;nrm[o*3+2]=nz;}}',
  ' var posBuf=gl.createBuffer();gl.bindBuffer(gl.ARRAY_BUFFER,posBuf);gl.bufferData(gl.ARRAY_BUFFER,new Float32Array(pos),gl.STATIC_DRAW);',
  ' var nrmBuf=gl.createBuffer();gl.bindBuffer(gl.ARRAY_BUFFER,nrmBuf);gl.bufferData(gl.ARRAY_BUFFER,nrm,gl.STATIC_DRAW);',
  ' var use32=nv>65535;var ext=use32?gl.getExtension("OES_element_index_uint"):null;',
  ' if(use32&&!ext){console.warn("部件 "+p.id+" 顶点数 "+nv+" 超过 65535 且浏览器不支持 32 位索引，已跳过");continue}',
  ' var IdxArr=use32?Uint32Array:Uint16Array,GlType=use32?gl.UNSIGNED_INT:gl.UNSIGNED_SHORT;',
  ' var idxBuf=gl.createBuffer();gl.bindBuffer(gl.ELEMENT_ARRAY_BUFFER,idxBuf);',
  ' gl.bufferData(gl.ELEMENT_ARRAY_BUFFER,new IdxArr(idx),gl.STATIC_DRAW);',
  ' var lines=[];for(var t=0;t<idx.length;t+=3){lines.push(idx[t],idx[t+1],idx[t+1],idx[t+2],idx[t+2],idx[t])}',
  ' var wireBuf=gl.createBuffer();gl.bindBuffer(gl.ELEMENT_ARRAY_BUFFER,wireBuf);',
  ' gl.bufferData(gl.ELEMENT_ARRAY_BUFFER,new IdxArr(lines),gl.STATIC_DRAW);',
  ' meshes.push({pos:posBuf,nrm:nrmBuf,idx:idxBuf,wire:wireBuf,count:idx.length,lcount:lines.length,col:hex(p.color),id:p.id,name:p.name,flt:GlType});}',
  // ── 矩阵（列主序，与 WebGL 一致）
  'function mul(a,b){var o=new Float32Array(16);for(var c=0;c<4;c++)for(var r=0;r<4;r++){var s=0;for(var k=0;k<4;k++)s+=a[k*4+r]*b[c*4+k];o[c*4+r]=s}return o}',
  'function persp(fov,asp,n,f){var t=1/Math.tan(fov/2),o=new Float32Array(16);o[0]=t/asp;o[5]=t;o[10]=(f+n)/(n-f);o[11]=-1;o[14]=2*f*n/(n-f);return o}',
  'function look(eye,ctr,up){var z=norm(sub(eye,ctr)),x=norm(cross(up,z)),y=cross(z,x),o=new Float32Array(16);',
  ' o[0]=x[0];o[4]=x[1];o[8]=x[2];o[1]=y[0];o[5]=y[1];o[9]=y[2];o[2]=z[0];o[6]=z[1];o[10]=z[2];',
  ' o[12]=-dot(x,eye);o[13]=-dot(y,eye);o[14]=-dot(z,eye);o[15]=1;return o}',
  'function sub(a,b){return [a[0]-b[0],a[1]-b[1],a[2]-b[2]]}function dot(a,b){return a[0]*b[0]+a[1]*b[1]+a[2]*b[2]}',
  'function cross(a,b){return [a[1]*b[2]-a[2]*b[1],a[2]*b[0]-a[0]*b[2],a[0]*b[1]-a[1]*b[0]]}',
  'function norm(a){var l=Math.sqrt(dot(a,a))||1;return [a[0]/l,a[1]/l,a[2]/l]}',
  // ── 相机（轨道）
  'var lo=D.bbox[0],hi=D.bbox[1];',
  'var ctr=[(lo[0]+hi[0])/2,(lo[1]+hi[1])/2,(lo[2]+hi[2])/2];',
  'var size=Math.max(hi[0]-lo[0],hi[1]-lo[1],hi[2]-lo[2],0.001);',
  'var cam={th:0.9,ph:1.05,dist:size*2.6,target:ctr.slice()}, wire=false, drag=null;',
  'function eye(){return [cam.target[0]+cam.dist*Math.sin(cam.ph)*Math.cos(cam.th),',
  ' cam.target[1]+cam.dist*Math.sin(cam.ph)*Math.sin(cam.th),cam.target[2]+cam.dist*Math.cos(cam.ph)]}',
  // ── 交互
  'cv.addEventListener("contextmenu",function(e){e.preventDefault()});',
  'cv.addEventListener("mousedown",function(e){drag={x:e.clientX,y:e.clientY,btn:e.button};e.preventDefault()});',
  'window.addEventListener("mouseup",function(){drag=null});',
  'window.addEventListener("mousemove",function(e){if(!drag)return;var dx=e.clientX-drag.x,dy=e.clientY-drag.y;drag.x=e.clientX;drag.y=e.clientY;',
  ' if(drag.btn===0){cam.th-=dx*0.008;cam.ph-=dy*0.008;cam.ph=Math.max(0.02,Math.min(Math.PI-0.02,cam.ph))}',
  ' else{var e0=eye(),z=norm(sub(e0,cam.target)),x=norm(cross([0,0,1],z)),y=cross(z,x);',
  '  var s=cam.dist*0.0016;for(var k=0;k<3;k++)cam.target[k]+=(-x[k]*dx+y[k]*dy)*s;}});',
  'cv.addEventListener("wheel",function(e){e.preventDefault();cam.dist*=Math.exp((e.deltaY>0?1:-1)*0.12);',
  ' cam.dist=Math.max(size*0.15,Math.min(size*40,cam.dist))},{passive:false});',
  'window.addEventListener("keydown",function(e){var k=e.key.toLowerCase();',
  ' if(k==="w"){wire=!wire}if(k==="r"){cam.th=0.9;cam.ph=1.05;cam.dist=size*2.6;cam.target=ctr.slice()}});',
  // ── 渲染
  'function frame(){var dpr=Math.min(2,window.devicePixelRatio||1);',
  ' var w=Math.floor(cv.clientWidth*dpr),h=Math.floor(cv.clientHeight*dpr);',
  ' if(cv.width!==w||cv.height!==h){cv.width=w;cv.height=h}',
  ' gl.viewport(0,0,cv.width,cv.height);gl.clearColor(0.078,0.086,0.102,1);gl.clear(gl.COLOR_BUFFER_BIT|gl.DEPTH_BUFFER_BIT);',
  ' var ep=eye(),proj=persp(45*Math.PI/180,cv.width/Math.max(1,cv.height),Math.max(size*0.005,0.01),size*80);',
  ' var view=look(ep,cam.target,[0,0,1]);',
  ' gl.uniformMatrix4fv(uProj,false,proj);gl.uniformMatrix4fv(uView,false,view);gl.uniform3fv(uCam,new Float32Array(ep));',
  ' for(var i=0;i<meshes.length;i++){var m=meshes[i];',
  '  gl.bindBuffer(gl.ARRAY_BUFFER,m.pos);gl.vertexAttribPointer(aPos,3,gl.FLOAT,false,0,0);',
  '  gl.bindBuffer(gl.ARRAY_BUFFER,m.nrm);gl.vertexAttribPointer(aNrm,3,gl.FLOAT,false,0,0);',
  '  gl.uniform3fv(uColor,new Float32Array(m.col));',
  '  if(!wire){gl.uniform1f(uWire,0);gl.enable(gl.CULL_FACE);gl.bindBuffer(gl.ELEMENT_ARRAY_BUFFER,m.idx);',
  '   gl.drawElements(gl.TRIANGLES,m.count,m.flt,0)}',
  '  else{gl.uniform1f(uWire,1);gl.disable(gl.CULL_FACE);gl.bindBuffer(gl.ELEMENT_ARRAY_BUFFER,m.wire);',
  '   gl.drawElements(gl.LINES,m.lcount,m.flt,0);gl.enable(gl.CULL_FACE)}}',
  ' requestAnimationFrame(frame)}',
  // ── HUD
  'var nh=["<h1>"+esc(D.name)+" · 3D 预览</h1>",',
  ' "<div class=kv><span>单位 / 坐标</span><b>"+esc(D.unit)+" · "+esc(D.up)+"-up</b></div>",',
  ' "<div class=kv><span>部件 / 三角面</span><b>"+D.parts.length+" / "+D.totalTris.toLocaleString()+"</b></div>",',
  ' "<div class=kv><span>体积合计</span><b>"+fmt(D.totalVolume)+" mm³</b></div>",',
  ' "<div class=kv><span>包围盒</span><b>"+fmt(hi[0]-lo[0])+"×"+fmt(hi[1]-lo[1])+"×"+fmt(hi[2]-lo[2])+" mm</b></div>"];',
  'if(D.params&&D.params.length){nh.push("<div class=sec><div class=t>参数</div>");',
  ' for(var q=0;q<D.params.length;q++){var pr=D.params[q];',
  '  nh.push("<div class=kv><span>"+esc(pr.name)+"</span><b>"+fmt(pr.value)+(pr.range?" <span style=\'color:#7f8896\'>("+esc(pr.range)+")</span>":"")+"</b></div>")}nh.push("</div>")}',
  'nh.push("<div class=sec><div class=t>部件</div>");',
  'for(var z=0;z<D.parts.length;z++){var pt=D.parts[z];',
  ' nh.push("<span class=chip><span class=dot style=\'background:"+esc(pt.color)+"\'></span>"+esc(pt.id)+" <span style=\'color:#7f8896\'>"+pt.tris.toLocaleString()+" 面</span></span>")}',
  'nh.push("</div>");hud.innerHTML=nh.join("");',
  'function esc(s){return String(s).replace(/[&<>"]/g,function(c){return {"&":"&amp;","<":"&lt;",">":"&gt;","\\"":"&quot;"}[c]})}',
  'function fmt(v){v=Number(v);if(!isFinite(v))return "-";var a=Math.abs(v);',
  ' if(a>=1000)return v.toFixed(0);if(a>=10)return v.toFixed(2);if(a>=0.01)return v.toFixed(3);return v.toExponential(2)}',
  'if(!window.requestAnimationFrame)window.requestAnimationFrame=function(f){setTimeout(f,16)};',
  // 调试钩子（供 IDE/CI 的多视角截图与自检使用，不影响正常交互）
  'window.__MODEL_VIEW={',
  ' set:function(o){if(o.th!==undefined)cam.th=o.th;if(o.ph!==undefined)cam.ph=o.ph;',
  '  if(o.dist!==undefined)cam.dist=o.dist;if(o.wire!==undefined)wire=!!o.wire;',
  '  if(o.reset){cam.th=0.9;cam.ph=1.05;cam.dist=size*2.6;cam.target=ctr.slice()}},',
  ' get:function(){return {th:cam.th,ph:cam.ph,dist:cam.dist,wire:wire}},',
  ' stats:function(){return {meshes:meshes.length,tris:meshes.reduce(function(s,m){return s+m.count/3},0),',
  '  parts:meshes.map(function(m){return {id:m.id,verts:m.count/3}})}}};',
  'frame();',
  '})();'
].join('\n');

// ── 工具层辅助 ─────────────────────────────────────────────
function argStr(args, key, dflt) {
  var v = args ? args[key] : undefined;
  if (v === undefined || v === null) return dflt;
  return isStr(v) ? v : String(v);
}
function argNum(args, key, dflt) {
  var v = args ? args[key] : undefined;
  if (v === undefined || v === null || v === '') return dflt;
  return toNumber(v, key);
}
function argBool(args, key, dflt) {
  var v = args ? args[key] : undefined;
  if (v === undefined || v === null) return dflt;
  return !!v;
}
function argArr(args, key, dflt) {
  var v = args ? args[key] : undefined;
  if (v === undefined || v === null) return dflt;
  if (isArr(v)) return v;
  if (isStr(v)) return v.split(',').map(function (s) { return s.trim(); }).filter(function (s) { return s.length > 0; });
  fail(key + ' 必须是数组');
}

function readModelFile(ctx, path) {
  if (!ctx.fs.exists(path)) fail('工程不存在：' + path + '（先执行 model_doc action=new）');
  var txt;
  try { txt = ctx.fs.readFile(path); } catch (e) { fail('读取工程失败：' + path + ' — ' + ((e && e.message) || e)); }
  var doc;
  try { doc = JSON.parse(txt); } catch (e) { fail('工程 JSON 解析失败（' + path + '）：' + ((e && e.message) || e)); }
  if (!isObj(doc)) fail('工程根必须是对象：' + path);
  if (doc.format === undefined) doc.format = FORMAT;
  if (doc.params === undefined) doc.params = [];
  if (doc.parts === undefined) doc.parts = [];
  return { model: doc, path: path, chars: txt.length };
}
function writeModelFile(ctx, path, model) {
  ctx.fs.writeFile(path, JSON.stringify(model, null, 2) + '\n');
  // ★ 预览链路（2026-09-17）：工程写盘即登记为「识别到的文件」→ 面板实时刷新并可点开预览
  noteArtifact(ctx, path, 'project', 'project',
    (model.name || '未命名') + ' · ' + ((model.parts && model.parts.length) || 0) + ' 部件');
}

// 参数输入的两种写法都接受：[{id,value,min,max,name}] 或 {id: value}
function normalizeParamInput(input) {
  var out = [];
  if (isArr(input)) {
    for (var k = 0; k < input.length; k++) {
      var p = input[k];
      if (!isObj(p) || !p.id) fail('params[' + k + '] 必须是 {id, value, ...}');
      var np = newParam(p.id, p.value, p.name, p.min, p.max);
      out.push(np);
    }
  } else if (isObj(input)) {
    for (var id in input) {
      if (!input.hasOwnProperty(id)) continue;
      out.push(newParam(id, input[id]));
    }
  } else fail('params 必须是数组或映射对象');
  return out;
}

function paramTable(model) {
  var out = [], k;
  for (k = 0; k < model.params.length; k++) {
    var p = model.params[k];
    out.push({ id: p.id, name: p.name === undefined ? null : p.name, value: p.value, min: p.min, max: p.max });
  }
  return out;
}

function modelSummary(model, built) {
  var s = {
    name: model.name,
    unit: model.unit || 'mm',
    parts: built.parts.length,
    triangles: built.totalTris,
    params: model.params.length,
    volume: cleanNum(built.parts.reduce(function (a, x) { return a + x.volume; }, 0), 4),
    area: cleanNum(built.parts.reduce(function (a, x) { return a + x.area; }, 0), 4),
    bbox: built.parts.length ? meshBbox(built.merged).map(function (v) { return v.map(function (x) { return cleanNum(x, 4); }); }) : null,
    size: built.parts.length ? meshSize(built.merged).map(function (x) { return cleanNum(x, 4); }) : null
  };
  return s;
}

function partBrief(p) {
  return {
    id: p.id,
    name: p.name,
    type: p.source.shape ? (p.source.shape.type || 'cuboid') : 'cuboid',
    ops: p.source.ops ? p.source.ops.length : 0,
    repeat: p.source.repeat ? p.source.repeat.mode || 'linear' : null,
    triangles: p.tris,
    volume: cleanNum(p.volume, 4),
    size: meshSize(p.mesh).map(function (x) { return cleanNum(x, 3); }),
    bbox: p.bbox.map(function (v) { return v.map(function (x) { return cleanNum(x, 3); }); }),
    material: p.material,
    visible: p.source.visible !== false
  };
}

// ── model_doc：工程生命周期 ─────────────────────────────────
function modelDoc(args, unknown, ctx) {
  var action = (argStr(args, 'action', 'get') || 'get').toLowerCase();
  var path = argStr(args, 'path', PROJECT_NAME) || PROJECT_NAME;
  if (action === 'new' || action === 'create') {
    if (ctx.fs.exists(path) && !argBool(args, 'overwrite', false)) {
      var cur = readModelFile(ctx, path);
      var b0 = null, e0 = null;
      try { b0 = buildModelMeshes(cur.model, { skipInvisible: false }); } catch (e) { e0 = (e && e.message) || String(e); }
      return {
        ok: true, action: 'exists', path: path,
        summary: b0 ? modelSummary(cur.model, b0) : null,
        error: e0,
        hint: '工程已存在，未覆盖。要重建请传 overwrite=true'
      };
    }
    var doc = emptyModelDoc(argStr(args, 'name', 'model'));
    if (args.params) doc.params = normalizeParamInput(args.params);
    if (args.note) doc.note = argStr(args, 'note');
    var rp = resolveParams(doc);
    // ★ 支持 parts：一次调用直接「建工程 + 全部部件」（每项与 model_add 单件同构），
    //   省掉「先 new 再逐件 model_add」的往返；构造/试建失败则整批不落盘。
    var added0 = [];
    if (isArr(args.parts) && args.parts.length) {
      var E0 = { params: rp.values, known: rp.known };
      for (var q0 = 0; q0 < args.parts.length; q0++) {
        if (!isObj(args.parts[q0])) fail('parts[' + q0 + '] 必须是对象');
        var p0 = buildPartDef(args.parts[q0], doc);
        var m0 = buildPartMesh(p0, E0);
        doc.parts.push(p0);
        added0.push(partAddedBrief(p0, m0));
      }
    }
    writeModelFile(ctx, path, doc);
    var resNew = {
      ok: true, action: 'created', path: path, name: doc.name,
      params: paramTable(doc),
      parts: doc.parts.length,
      hint: '接着用 model_add（单件或 parts=[…] 批量）/ model_edit op=part.add 加几何，再用 model_export / model_verify'
    };
    if (added0.length) resNew.added = added0;
    return resNew;
  }
  var rm = readModelFile(ctx, path);
  var model = rm.model;
  if (action === 'get' || action === 'read') {
    var built = null, err = null;
    try { built = buildModelMeshes(model, { skipInvisible: false }); } catch (e) { err = (e && e.message) || String(e); }
    return {
      ok: true, action: 'get', path: path, chars: rm.chars,
      format: model.format, name: model.name, unit: model.unit, up: model.up,
      params: paramTable(model),
      parts: built ? built.parts.map(partBrief) : (model.parts || []).map(function (p) { return { id: p.id, shape: p.shape ? p.shape.type : null, error: '几何未能构建' }; }),
      summary: built ? modelSummary(model, built) : null,
      error: err
    };
  }
  if (action === 'rename') {
    model.name = argStr(args, 'name', model.name);
    writeModelFile(ctx, path, model);
    return { ok: true, action: 'rename', path: path, name: model.name };
  }
  if (action === 'param' || action === 'set-param' || action === 'param-set') {
    var changed = [];
    if (args.params) {
      var patch = isArr(args.params) ? args.params : args.params;
      if (isArr(patch)) {
        for (var k = 0; k < patch.length; k++) {
          var item = patch[k];
          if (!isObj(item) || !item.id) fail('params[' + k + '] 必须是 {id, value}');
          setParamValue(model, item.id, item.value);
          changed.push(item.id);
        }
      } else {
        for (var pid in patch) {
          if (!patch.hasOwnProperty(pid)) continue;
          setParamValue(model, pid, patch[pid]);
          changed.push(pid);
        }
      }
    }
    if (args.id !== undefined) { setParamValue(model, argStr(args, 'id'), args.value); changed.push(argStr(args, 'id')); }
    if (!changed.length) fail('未指定要修改的参数（用 params:{id:value} 或 id + value）');
    var rp2 = resolveParams(model);
    writeModelFile(ctx, path, model);
    return {
      ok: true, action: 'param', path: path, changed: changed,
      params: paramTable(model),
      values: rp2.values
    };
  }
  if (action === 'reset') {
    if (!argBool(args, 'force', false)) fail('reset 会清空全部部件（保留参数与画布设置）。确认请传 force=true');
    model.parts = [];
    writeModelFile(ctx, path, model);
    return { ok: true, action: 'reset', path: path, parts: 0, params: model.params.length };
  }
  if (action === 'set') {
    var fields = [];
    if (args.name !== undefined) { model.name = argStr(args, 'name'); fields.push('name'); }
    if (args.unit !== undefined) { model.unit = argStr(args, 'unit', 'mm'); fields.push('unit'); }
    if (args.up !== undefined) { model.up = argStr(args, 'up', 'z'); fields.push('up'); }
    if (args.note !== undefined) { model.note = argStr(args, 'note'); fields.push('note'); }
    if (!fields.length) fail('set 需要至少一个字段：name / unit / up / note');
    writeModelFile(ctx, path, model);
    return { ok: true, action: 'set', path: path, fields: fields };
  }
  fail('未知 action ' + action + '（可用 new / get / rename / param / set / reset）');
}

function setParamValue(model, id, value) {
  if (!id) fail('参数 id 不能为空');
  for (var k = 0; k < model.params.length; k++) {
    if (model.params[k].id === id) { model.params[k].value = value; return; }
  }
  fail('参数不存在：' + id + '（现有：' + model.params.map(function (p) { return p.id; }).join(', ') + '）');
}

// ── model_add：新增部件 ─────────────────────────────────────
// 一处「部件定义」→ part 对象。四条新增路径共用同一套构造口径，避免「批量走的是另一套构造」分叉：
//   ① model_add 单件（同层 id/type/size/…）    ② model_add parts=[{…},…] 批量项
//   ③ model_doc action=new 的 parts 项          ④ model_edit 的 {op:"part.add", …} 命令
// ★ id 唯一性校验基于 model.parts 当前内容 —— 批量时逐件入列后再构造下一件，同批内的重复 id 也能抓到；
//   失败一律抛错 → 调用方不落盘（事务性），工程保持原样。
function buildPartDef(defArgs, model) {
  var id = argStr(defArgs, 'id');
  if (!id) fail('新增部件必须提供 id');
  for (var k = 0; k < model.parts.length; k++) {
    if (model.parts[k].id === id) fail('部件 id 已存在：' + id + '（改名或先 part.remove）');
  }
  var part = { id: id };
  if (defArgs.from) { // 从已有部件复制几何与设置
    var src = null;
    for (var m = 0; m < model.parts.length; m++) if (model.parts[m].id === argStr(defArgs, 'from')) src = model.parts[m];
    if (!src) fail('from 指向的部件不存在：' + argStr(defArgs, 'from'));
    part = JSON.parse(JSON.stringify(src));
    part.id = id;
    if (defArgs.name !== undefined) part.name = argStr(defArgs, 'name');
    if (defArgs.transform) part.transform = defArgs.transform;
    if (defArgs.repeat) part.repeat = defArgs.repeat;
    if (defArgs.material) part.material = defArgs.material;
  } else {
    var shape = defArgs.shape;
    if (!shape) {
      var t = argStr(defArgs, 'type', 'cuboid');
      shape = { type: t };
      var keys = ['size', 'center', 'radius', 'radiusTop', 'radiusBottom', 'height', 'innerRadius', 'outerRadius',
        'segments', 'rings', 'profile', 'holes', 'twist', 'along', 'closed', 'scale', 'up', 'points', 'faces', 'angle', 'sides', 'op', 'of', 'a', 'b', 'shapes', 'positions', 'indices'];
      for (var i = 0; i < keys.length; i++) {
        if (defArgs[keys[i]] !== undefined) shape[keys[i]] = defArgs[keys[i]];
      }
    }
    part.shape = shape;
    if (defArgs.name !== undefined) part.name = argStr(defArgs, 'name');
    if (defArgs.transform) part.transform = defArgs.transform;
    if (defArgs.repeat) part.repeat = defArgs.repeat;
    if (defArgs.material) part.material = defArgs.material;
    if (defArgs.ops) part.ops = defArgs.ops;
  }
  if (defArgs.visible === false) part.visible = false;
  if (defArgs.ops && defArgs.from) {
    part.ops = (part.ops || []).concat(defArgs.ops);
  }
  return part;
}

// 新增部件的回报摘要（model_add 批量件 / model_doc new 共用；口径与旧单件返回一致）
function partAddedBrief(part, mesh) {
  return {
    id: part.id, type: part.shape ? part.shape.type : null,
    triangles: mesh.indices.length, volume: cleanNum(meshVolume(mesh), 4),
    size: meshSize(mesh).map(function (x) { return cleanNum(x, 3); })
  };
}

function modelAdd(args, unknown, ctx) {
  var path = argStr(args, 'path', PROJECT_NAME) || PROJECT_NAME;
  var rm = readModelFile(ctx, path);
  var model = rm.model;
  // ★ 批量：parts=[{…},{…}] 一次加任意多个部件（每项与单件同构：id/from/type/…/transform/repeat/material/ops）。
  //   单件写法（id/type/size/… 写在同层）保持兼容 —— 传了 parts 就用 parts，忽略同层单件字段。
  //   事务性：全部构造 + 试建通过才落盘一次；任一项失败整批不写（工程保持原样）。
  var defs = isArr(args.parts) ? args.parts : [args];
  if (!defs.length) fail('parts 数组为空（传 parts=[{…}] 批量，或直接给单件的 id/type/… 参数）');
  // 先试建（参数上下文用当前工程）——保证写进去的几何一定是能算出来的
  var rp = resolveParams(model);
  var E = { params: rp.values, known: rp.known };
  var added = [];
  for (var i = 0; i < defs.length; i++) {
    if (!isObj(defs[i])) fail('parts[' + i + '] 必须是对象');
    var part = buildPartDef(defs[i], model);
    var mesh = buildPartMesh(part, E);
    model.parts.push(part);   // 逐件入列：下一件的 id 唯一性校验才看得见（失败则整体不落盘）
    added.push(partAddedBrief(part, mesh));
  }
  writeModelFile(ctx, path, model);
  var res = { ok: true, action: 'added', path: path, added: added, parts: model.parts.length };
  if (added.length === 1) { res.id = added[0].id; res.part = added[0]; }  // 单件写法兼容旧返回体
  return res;
}

// ── model_edit：命令式编辑（一次一批，任一 op 失败则整批不落盘）──────
function findPart(model, id, opName) {
  for (var k = 0; k < model.parts.length; k++) if (model.parts[k].id === id) return model.parts[k];
  fail('部件不存在：' + id + '（' + opName + '；现有：' + model.parts.map(function (p) { return p.id; }).join(', ') + '）');
}
function findPartIndex(model, id) {
  for (var k = 0; k < model.parts.length; k++) if (model.parts[k].id === id) return k;
  return -1;
}

function applyOp(model, op, index) {
  var name = argStr(op, 'op');
  if (!name) fail('ops[' + index + '] 缺少 op');
  var id = argStr(op, 'id');
  var k;
  if (name === 'part.add') {
    // 批量加部件：与 model_add 同一套构造口径（from 复制 / 表达式尺寸 / 阵列 / 材质 / ops）——
    // 复杂模型可以在**一次 model_edit 调用**里用 N 条 part.add 成型，不必一件一次 model_add。
    // 部件定义推荐写在 part:{…}（与 model_add 的 parts 项同构）；也支持直接平铺在同一层。
    var defAdd = isObj(op.part) ? op.part : op;
    var pNew = buildPartDef(defAdd, model);
    model.parts.push(pNew);
    return '新增部件 ' + pNew.id + '（' + ((pNew.shape && pNew.shape.type) || 'cuboid') + '）';
  }
  if (name === 'part.remove' || name === 'part.delete') {
    var idx = findPartIndex(model, id);
    if (idx < 0) fail('部件不存在：' + id);
    model.parts.splice(idx, 1);
    return '移除部件 ' + id;
  }
  if (name === 'part.rename') {
    var to = argStr(op, 'to', argStr(op, 'name'));
    if (!to) fail('part.rename 需要 to');
    if (!/^[A-Za-z0-9_\-\.]+$/.test(to)) fail('部件 id 只允许字母数字下划线连字符点：' + to);
    if (findPartIndex(model, to) >= 0) fail('目标 id 已存在：' + to);
    findPart(model, id, name).id = to;
    return '重命名部件 ' + id + ' → ' + to;
  }
  if (name === 'part.set') {
    var p = findPart(model, id, name);
    var props = argArr(op, 'props') ? null : op.props;
    if (isObj(op.props)) {
      for (k in op.props) if (op.props.hasOwnProperty(k)) p[k] = op.props[k];
    } else fail('part.set 需要 props 对象');
    return '更新部件 ' + id + ' 的 ' + objKeys(op.props).join(', ');
  }
  if (name === 'part.shape') {
    var p2 = findPart(model, id, name);
    if (op.shape) p2.shape = op.shape;
    else {
      var shape = { type: argStr(op, 'type', 'cuboid') };
      var keys = ['size', 'center', 'radius', 'radiusTop', 'radiusBottom', 'height', 'innerRadius', 'outerRadius',
        'segments', 'rings', 'profile', 'holes', 'twist', 'along', 'closed', 'scale', 'up', 'points', 'faces', 'angle', 'sides', 'op', 'of', 'a', 'b', 'shapes'];
      for (k = 0; k < keys.length; k++) if (op[keys[k]] !== undefined) shape[keys[k]] = op[keys[k]];
      p2.shape = shape;
    }
    return '替换部件 ' + id + ' 的几何（' + (p2.shape.type || 'cuboid') + '）';
  }
  if (name === 'part.transform') {
    findPart(model, id, name).transform = op.transform === undefined ? {} : op.transform;
    return '设置部件 ' + id + ' 的变换';
  }
  if (name === 'part.repeat') {
    if (op.repeat === null || op.repeat === undefined) delete findPart(model, id, name).repeat;
    else findPart(model, id, name).repeat = op.repeat;
    return '设置部件 ' + id + ' 的阵列' + (op.repeat ? '（' + (op.repeat.mode || 'linear') + '）' : '（已清除）');
  }
  if (name === 'part.material') {
    findPart(model, id, name).material = op.material;
    return '设置部件 ' + id + ' 的材质';
  }
  if (name === 'part.hide' || name === 'part.show') {
    findPart(model, id, name).visible = (name === 'part.show');
    return (name === 'part.hide' ? '隐藏' : '显示') + '部件 ' + id;
  }
  if (name === 'op.add') {
    var p3 = findPart(model, id, name);
    if (!isObj(op.op2) && !isObj(op.opObj)) {
      // 允许 {"op":"op.add","id":..,"do":"subtract","shape":{...}}
      var doName = argStr(op, 'do', 'subtract');
      if (!op.shape) fail('op.add 需要 shape（或 do + shape）');
      p3.ops = p3.ops || [];
      p3.ops.push({ op: doName, shape: op.shape, transform: op.transform });
      return '给部件 ' + id + ' 追加 ' + doName + ' 运算';
    }
    p3.ops = p3.ops || [];
    p3.ops.push(op.op2 || op.opObj);
    return '给部件 ' + id + ' 追加运算';
  }
  if (name === 'op.remove') {
    var p4 = findPart(model, id, name);
    if (!p4.ops || !p4.ops.length) fail('部件 ' + id + ' 没有 ops');
    var oi = argNum(op, 'index', -1);
    if (oi < 0 || oi >= p4.ops.length) fail('op.remove 的 index 越界（0..' + (p4.ops.length - 1) + '）');
    p4.ops.splice(oi, 1);
    return '删除部件 ' + id + ' 的第 ' + oi + ' 个运算';
  }
  if (name === 'op.clear') {
    var p5 = findPart(model, id, name);
    var n = (p5.ops || []).length;
    p5.ops = [];
    return '清空部件 ' + id + ' 的 ' + n + ' 个运算';
  }
  if (name === 'param.add') {
    if (!id) fail('param.add 需要 id');
    for (k = 0; k < model.params.length; k++) if (model.params[k].id === id) fail('参数已存在：' + id);
    model.params.push(newParam(id, op.value === undefined ? 0 : op.value, op.name, op.min, op.max));
    return '新增参数 ' + id + ' = ' + JSON.stringify(op.value === undefined ? 0 : op.value);
  }
  if (name === 'param.set') {
    setParamValue(model, id, op.value);
    return '设置参数 ' + id + ' = ' + JSON.stringify(op.value);
  }
  if (name === 'param.remove') {
    var pi = -1;
    for (k = 0; k < model.params.length; k++) if (model.params[k].id === id) pi = k;
    if (pi < 0) fail('参数不存在：' + id);
    model.params.splice(pi, 1);
    return '删除参数 ' + id;
  }
  if (name === 'mesh.align') {
    var p6 = findPart(model, id, name);
    p6.ops = p6.ops || [];
    p6.ops.push({ op: 'align', to: op.to === undefined ? 'center' : op.to });
    return '对部件 ' + id + ' 追加对齐（' + (op.to || 'center') + '）';
  }
  if (name === 'mesh.snap') {
    var p7 = findPart(model, id, name);
    p7.ops = p7.ops || [];
    p7.ops.push({ op: 'snap', grid: op.grid });
    return '对部件 ' + id + ' 追加顶点吸附';
  }
  if (name === 'mesh.repair') {
    var p8 = findPart(model, id, name);
    p8.ops = p8.ops || [];
    p8.ops.push({ op: 'repair' });
    return '对部件 ' + id + ' 追加几何修复（吸附 + 焊接 + 去退化面）';
  }
  if (name === 'mesh.chamfer' || name === 'mesh.fillet') {
    var isFillet = (name === 'mesh.fillet');
    var pBevel = findPart(model, id, name);
    if (op.distance === undefined && !(isFillet && op.radius !== undefined)) {
      fail(name + ' 需要 ' + (isFillet ? 'radius（或 distance）' : 'distance') + '（面内尺寸，可写表达式）');
    }
    var bevel = { op: isFillet ? 'fillet' : 'chamfer' };
    if (isFillet) bevel.radius = op.radius === undefined ? op.distance : op.radius;
    else bevel.distance = op.distance;
    if (op.edges !== undefined) bevel.edges = op.edges;
    if (op.minAngle !== undefined) bevel.minAngle = op.minAngle;
    if (isFillet && op.segments !== undefined) bevel.segments = op.segments;
    pBevel.ops = pBevel.ops || [];
    pBevel.ops.push(bevel);
    return '对部件 ' + id + ' 追加' + (isFillet ? '圆角' : '倒角') + '（' +
      (isFillet ? 'radius=' : 'distance=') + JSON.stringify(isFillet ? bevel.radius : bevel.distance) + '，' +
      (isArr(op.edges) ? '指定 ' + op.edges.length + ' 条棱' : '全部凸边（折角 ≥ ' +
        (op.minAngle === undefined ? EDGE_ANGLE_DEFAULT : op.minAngle) + '°）') +
      (isFillet ? '，弧面离散 ' + (op.segments === undefined ? ARC_SEG_DEFAULT : op.segments) + ' 段' : '') + '）';
  }
  if (name === 'part.reorder') {
    var ids = argArr(op, 'ids', []);
    if (!ids.length) fail('part.reorder 需要 ids 数组');
    var rest = model.parts.filter(function (x) { return ids.indexOf(x.id) < 0; });
    var ordered = [];
    for (k = 0; k < ids.length; k++) ordered.push(findPart(model, ids[k], name));
    model.parts = ordered.concat(rest);
    return '调整部件顺序：' + ids.join(', ') + ' 置前';
  }
  fail('未知 op ' + name + '（可用 part.add/remove/rename/set/shape/transform/repeat/material/hide/show/reorder、' +
    'op.add/remove/clear、param.add/set/remove、mesh.align/snap/repair/chamfer/fillet）');
}

function modelEdit(args, unknown, ctx) {
  var path = argStr(args, 'path', PROJECT_NAME) || PROJECT_NAME;
  var rm = readModelFile(ctx, path);
  var model = rm.model;
  var ops = args.ops;
  if (ops === undefined && args.op) ops = [args];
  if (!isArr(ops) || !ops.length) fail('需要 ops 数组（或 op 名 + 同层参数构成单条命令）');
  var applied = [];
  for (var k = 0; k < ops.length; k++) {
    if (!isObj(ops[k])) fail('ops[' + k + '] 必须是对象');
    applied.push(applyOp(model, ops[k], k));
  }
  // 事务性：整批应用成功后才做"可解析 + 可构建"校验，然后落盘
  resolveParams(model);
  var diag = [];
  var built = buildModelMeshes(model, { skipInvisible: false, diag: diag });
  writeModelFile(ctx, path, model);
  var res = { ok: true, path: path, applied: applied, parts: built.parts.length, summary: modelSummary(model, built) };
  // 圆角/倒角是网格级操作，实际处理了几条棱（以及跳过的凹边/平边）在这里如实回报
  if (diag.length) res.meshOps = diag;
  return res;
}

// ── 导出产物渲染（同一函数既写文件，也用于确定性比对）──────────
var EXPORT_FORMATS = ['gltf', 'glb', 'stl', 'stl-ascii', 'obj', 'html', 'json'];

var B64_LOOKUP = (function () {
  var m = {}, i;
  for (i = 0; i < B64_CHARS.length; i++) m[B64_CHARS.charAt(i)] = i;
  m['='] = 0;
  return m;
})();

function b64DecodeBytes(str) {
  // ★ 必须先剥掉尾部 '=' 填充：否则主循环会把"含填充的末块"也当成完整 4 字符块，
  //   多解出 1 个字节（大文件下 M8 的 STL 长度校验正是这么抓到它的）
  var s = String(str).replace(/[^A-Za-z0-9+\/=]/g, '').replace(/=+$/, '');
  var out = [], i;
  for (i = 0; i + 3 < s.length; i += 4) {
    var n = (B64_LOOKUP[s.charAt(i)] << 18) | (B64_LOOKUP[s.charAt(i + 1)] << 12) | (B64_LOOKUP[s.charAt(i + 2)] << 6) | B64_LOOKUP[s.charAt(i + 3)];
    out.push((n >> 16) & 255, (n >> 8) & 255, n & 255);
  }
  var rem = s.length - i;
  if (rem === 2) {
    var n1 = (B64_LOOKUP[s.charAt(i)] << 18) | (B64_LOOKUP[s.charAt(i + 1)] << 12);
    out.push((n1 >> 16) & 255);
  } else if (rem === 3) {
    var n2 = (B64_LOOKUP[s.charAt(i)] << 18) | (B64_LOOKUP[s.charAt(i + 1)] << 12) | (B64_LOOKUP[s.charAt(i + 2)] << 6);
    out.push((n2 >> 16) & 255, (n2 >> 8) & 255);
  }
  return out;
}

function outName(projectPath, ext) {
  return String(projectPath).replace(/\.json$/i, '') + ext;
}

function renderProduct(format, built, model, name) {
  var parts = built.parts;
  var merged = parts.length ? meshMerge(parts.map(function (p) { return p.mesh; })) : meshNew([], []);
  var faces = merged.indices.length;
  if (format === 'gltf') return { text: gltfText(parts, { name: name }), faces: faces, ext: '.gltf', kind: 'gltf' };
  if (format === 'glb') return { bytes: glbBytes(parts, { name: name }), faces: faces, ext: '.glb', kind: 'glb' };
  if (format === 'stl') return { bytes: stlBinaryBytes(merged, name), faces: faces, ext: '.stl', kind: 'stlb' };
  if (format === 'stl-ascii') return { text: stlAsciiText(merged, name), faces: faces, ext: '.ascii.stl', kind: 'stla' };
  if (format === 'obj') return { text: objText(parts), faces: faces, ext: '.obj', kind: 'obj' };
  if (format === 'html') return { text: previewHtml(built, model), faces: faces, ext: '.preview.html', kind: 'html' };
  if (format === 'json') return { text: JSON.stringify(model, null, 2) + '\n', faces: faces, ext: '.json', kind: 'json' };
  fail('未知导出格式 ' + format + '（可用 ' + EXPORT_FORMATS.join(' / ') + ' / all）');
}

function compareProduct(a, b) {
  if (!!a.text !== !!b.text) return { checked: true, same: false, len1: 0, len2: 0 };
  if (a.text !== undefined) return { checked: true, same: a.text === b.text, len1: a.text.length, len2: b.text.length };
  var same = a.bytes.length === b.bytes.length;
  if (same) {
    for (var i = 0; i < a.bytes.length; i++) if (a.bytes[i] !== b.bytes[i]) { same = false; break; }
  }
  return { checked: true, same: same, len1: a.bytes.length, len2: b.bytes.length };
}

// ── model_export ───────────────────────────────────────────
function modelExport(args, unknown, ctx) {
  var path = argStr(args, 'path', PROJECT_NAME) || PROJECT_NAME;
  var rm = readModelFile(ctx, path);
  var model = rm.model;
  var format = (argStr(args, 'format', 'gltf') || 'gltf').toLowerCase();
  var filter = argArr(args, 'parts', null);
  var built = buildModelMeshes(model, { skipInvisible: true });
  if (filter && filter.length) {
    var kept = [], k;
    for (k = 0; k < built.parts.length; k++) if (filter.indexOf(built.parts[k].id) >= 0) kept.push(built.parts[k]);
    if (!kept.length) fail('parts 过滤后没有可导出的部件（' + filter.join(', ') + '）');
    built.parts = kept;
    built.totalTris = kept.reduce(function (s, x) { return s + x.tris; }, 0);
  }
  if (!built.parts.length) fail('没有可导出的几何（工程里还没有部件，或全部 visible=false）');
  var name = argStr(args, 'name', model.name || 'model');
  var out = argStr(args, 'out', null);
  var formats = format === 'all' ? ['gltf', 'stl', 'stl-ascii', 'obj', 'html'] : [format];
  if (format === 'all' && argBool(args, 'withGlb', false)) formats.push('glb');
  var files = [], products = [], k2;
  for (k2 = 0; k2 < formats.length; k2++) {
    var fmt = formats[k2];
    var r = renderProduct(fmt, built, model, name);
    var target = out || outName(rm.path, r.ext);
    if (r.bytes !== undefined) {
      ctx.fs.writeFileBase64(target, b64EncodeBytes(r.bytes));
      products.push({ path: target, kind: r.kind, bytes: r.bytes, faces: r.faces });
      files.push({ path: target, format: fmt, bytes: r.bytes.length, faces: r.faces });
      noteArtifact(ctx, target, artifactKindOf(target), 'export', fmt + ' · ' + r.faces + ' 面');
    } else {
      ctx.fs.writeFile(target, r.text);
      products.push({ path: target, kind: r.kind, text: r.text, faces: r.faces });
      files.push({ path: target, format: fmt, bytes: utf8Bytes(r.text).length, faces: r.faces });
      noteArtifact(ctx, target, artifactKindOf(target), 'export', fmt + ' · ' + r.faces + ' 面');
    }
  }
  // 确定性：同一份工程渲染两次必须位级一致（M9 的证据）
  var det = compareProduct(renderProduct(formats[0], built, model, name), renderProduct(formats[0], built, model, name));
  var res = runChecks(model, built, { products: products, determinism: det });
  return {
    ok: true, path: rm.path, format: format, files: files,
    triangles: built.totalTris, determinism: det,
    verify: {
      fails: res.fails, warns: res.warns,
      checks: res.checks.map(function (c) { return { id: c.id, ok: c.ok, level: c.level, detail: c.detail }; })
    },
    hint: format === 'html' ? '预览 HTML 自包含（无外部依赖），可双击打开；IDE 面板也会直接渲染它' : null
  };
}

// ── model_verify：9 项判据 + 旁挂报告 ────────────────────────
function collectProducts(ctx, projectPath) {
  var base = String(projectPath).replace(/\.json$/i, '');
  var cands = ['.gltf', '.glb', '.stl', '.ascii.stl', '.obj', '.preview.html'];
  var kinds = { '.gltf': 'gltf', '.glb': 'glb', '.stl': 'stlb', '.ascii.stl': 'stla', '.obj': 'obj', '.preview.html': 'html' };
  var list = [], notes = [], k;
  for (k = 0; k < cands.length; k++) {
    var p = base + cands[k], kind = kinds[cands[k]];
    if (!ctx.fs.exists(p)) continue;
    try {
      if (kind === 'gltf' || kind === 'stla' || kind === 'obj' || kind === 'html') {
        list.push({ path: p, kind: kind, text: ctx.fs.readFile(p) });
      } else {
        list.push({ path: p, kind: kind, bytes: b64DecodeBytes(ctx.fs.readFileBase64(p)) });
      }
    } catch (e) {
      notes.push('产物 ' + p + ' 读取失败：' + ((e && e.message) || e));
    }
  }
  return { list: list, notes: notes };
}

function modelVerify(args, unknown, ctx) {
  var path = argStr(args, 'path', PROJECT_NAME) || PROJECT_NAME;
  var rm = readModelFile(ctx, path);
  var model = rm.model;
  var built = buildModelMeshes(model, { skipInvisible: false });
  var coll = collectProducts(ctx, rm.path);
  var det = null;
  if (argBool(args, 'determinism', false)) {
    var nm = model.name || 'model';
    det = compareProduct(renderProduct('gltf', built, model, nm), renderProduct('gltf', built, model, nm));
  }
  var res = runChecks(model, built, { products: coll.list, determinism: det });
  var report = {
    format: 'paircode.model.verify/1',
    project: rm.path,
    name: model.name,
    unit: model.unit || 'mm',
    summary: modelSummary(model, built),
    fails: res.fails,
    warns: res.warns,
    ok: res.ok,
    products: coll.list.map(function (p) { return p.path; }),
    // 每部件明细（面板与报告共用；面数/体积/尺寸只有构建后才知道，故写进报告）
    parts: built.parts.map(function (p) {
      return {
        id: p.id,
        name: p.name,
        type: p.source.shape ? (p.source.shape.type || 'cuboid') : 'cuboid',
        ops: p.source.ops ? p.source.ops.length : 0,
        repeat: p.source.repeat ? (p.source.repeat.mode || 'linear') : null,
        triangles: p.tris,
        volume: cleanNum(p.volume, 4),
        size: meshSize(p.mesh).map(function (x) { return cleanNum(x, 3); }),
        visible: p.source.visible !== false
      };
    }),
    checks: res.checks
  };
  var reportPath = argStr(args, 'report', null) || outName(rm.path, '.verify.json');
  ctx.fs.writeFile(reportPath, JSON.stringify(report, null, 2) + '\n');
  noteArtifact(ctx, reportPath, 'verify', 'verify', res.fails + ' fail / ' + res.warns + ' warn');
  return {
    ok: res.ok, path: rm.path, report: reportPath,
    fails: res.fails, warns: res.warns,
    checks: res.checks.map(function (c) {
      return { id: c.id, ok: c.ok, level: c.level, title: c.title, detail: c.detail, items: c.items.slice(0, 6) };
    }),
    productNotes: coll.notes
  };
}

// ══ 「识别到的文件」注册表 + 实时预览链路（2026-09-17：UI 与工具合并进同一插件）══
// 用户口径：工具操作到**具体文件** → 面板里点这个文件 → 登记为「识别到的文件」→ **实时**预览。
// host 半为此维护两件事：
//   登记表 ARTIFACTS：工具每次写文件（建/改工程、导出各格式产物、复验报告）立刻登记，并
//   ctx.emit('ui:tool-model/artifacts') 广播 → 面板事件驱动刷新（不靠手点刷新，也不轮询竞争）。
// ★ 2026-09-18：**不再扫描工作区**（原 scanWorkspaceArtifacts 已删）。旧口径是「登记表 + 全工作区
//   递归扫描（深度 4 / 上限 200 条）」，后果是工作区里任意 *.json（package.json / 别的域的工程
//   JSON / 任意配置）都被列进 3D 面板的「文件」栏 —— 用户口径：这个列表只该有**本插件产物**。
//   现在列表只有两条来源：
//     ① 工具产出：写盘即登记（工程 / 各格式产物 / 复验报告）；
//     ② 面板手动登记：顶部路径框输入文件 → claimArtifact 登记 → 当场预览。
//   外部/历史文件一律不自动出现（要看就在面板里手动登记，或直接在编辑器里打开）。
// 解析与渲染全在前端（点开即画）：host 只回「内容 + 元信息」——文本原样，二进制 base64。
var ARTIFACTS = [];        // [{path, kind, source, bytes, mtime, at, note}]（最新在前）
var ARTIFACTS_MAX = 300;   // 登记表上限（防长会话无限增长）

// 识别口径（扩展名 → kind）；未识别返回 ''（面板不列为「识别到的文件」）
function artifactKindOf(path) {
  var n = String(path || '').toLowerCase();
  if (/\.preview\.html$/.test(n)) return 'preview-html';
  if (/\.glb$/.test(n)) return 'glb';
  if (/\.gltf$/.test(n)) return 'gltf';
  if (/\.stl$/.test(n)) return 'stl';
  if (/\.obj$/.test(n)) return 'obj';
  if (/\.verify\.json$/.test(n)) return 'verify';
  if (/\.json$/.test(n)) return 'json';
  return '';
}

// 类型 → 人话标签（面板展示）
function artifactKindLabel(kind) {
  var m = {
    'preview-html': '自包含 WebGL 预览（HTML）',
    glb: 'GLB（单文件二进制 glTF）',
    gltf: 'glTF 2.0（Khronos 标准）',
    stl: 'STL（3D 打印；binary/ASCII 自动识别）',
    obj: 'Wavefront OBJ',
    verify: '复验报告（判据 M1–M9）',
    json: 'JSON',
    project: 'CAD 工程（paircode.model/1）'
  };
  return m[kind] || kind || '未知';
}

// 面板可见的条目（统一形状）
function artifactEntry(ctx, path, kind, source, note) {
  var st = null;
  try { st = ctx.fs.stat(path); } catch (e) { st = null; }
  if (!st || st.isDir) return null;
  var k = kind || artifactKindOf(path);
  return {
    path: path, kind: k, label: artifactKindLabel(k),
    source: source || 'claim', bytes: st.size, mtime: st.mtime,
    note: note || null, at: Date.now()
  };
}

// 广播：面板 ui.on('ui:tool-model/artifacts') 收到即刷新（事件驱动，实时）
function emitArtifacts(ctx, what) {
  try {
    if (ctx && typeof ctx.emit === 'function') {
      ctx.emit('ui:tool-model/artifacts', { at: Date.now(), what: what || '', count: ARTIFACTS.length });
    }
  } catch (e) { /* 旧宿主无事件桥：面板仍有定时复核兜底 */ }
}

// 登记一个文件为「识别到的文件」（工具写文件后立即调用；path = 工作区相对路径）
function noteArtifact(ctx, path, kind, source, note) {
  if (!path) return null;
  var item = { path: path, kind: kind || artifactKindOf(path), source: source || 'claim', note: note || null, at: Date.now() };
  for (var i = ARTIFACTS.length - 1; i >= 0; i--) if (ARTIFACTS[i].path === path) ARTIFACTS.splice(i, 1);
  ARTIFACTS.unshift(item);
  if (ARTIFACTS.length > ARTIFACTS_MAX) ARTIFACTS.length = ARTIFACTS_MAX;
  emitArtifacts(ctx, path);
  return item;
}

// ── client 方法（浏览器面板 ui.invoke('tool-model', <名>, args)）──────────
// listArtifacts：识别到的文件清单 = 登记表（工具产出 + 面板手动登记，最新在前）
// ★ 2026-09-18 去扫描：不再递归工作区。args.scan 已废弃（传了也忽略），返回里 scanned 恒 false
//   （老面板靠它决定是否本地合并扫描结果，恒 false 即不再合并），scanRemoved 供面板识别新口径。
function listArtifactsForUI(ctx, args) {
  var items = [], seen = {}, i;
  for (i = 0; i < ARTIFACTS.length; i++) {
    var it = ARTIFACTS[i];
    var e = artifactEntry(ctx, it.path, it.kind, it.source, it.note);
    if (!e) continue;                 // 已被删除/消失 → 不进清单
    if (seen[e.path]) continue;
    seen[e.path] = 1;
    items.push(e);
  }
  return { items: items, registered: ARTIFACTS.length, scanned: false, scanRemoved: true, at: Date.now() };
}

// readArtifact：给面板内容（文本原样；二进制 base64）——渲染与解析在前端做
function readArtifactForUI(ctx, args) {
  var path = argStr(args || {}, 'path', '') || '';
  if (!path) fail('readArtifact：缺 path');
  var st;
  try { st = ctx.fs.stat(path); } catch (e) { fail('文件不存在：' + path); }
  if (!st || st.isDir) fail('不是文件：' + path);
  var kind = artifactKindOf(path);
  var out = { path: path, kind: kind, label: artifactKindLabel(kind), bytes: st.size, mtime: st.mtime };
  // 文本类直接读文本；.json/.verify.json 再按 format 区分「工程」与「报告」
  var textKinds = { gltf: 1, obj: 1, 'preview-html': 1, verify: 1, json: 1 };
  if (textKinds[kind]) {
    var txt;
    try { txt = ctx.fs.readFile(path); } catch (e2) { fail('读取失败：' + path + ' — ' + ((e2 && e2.message) || e2)); }
    if (kind === 'json' || kind === 'verify') {
      try {
        var doc = JSON.parse(txt);
        if (doc && doc.format === FORMAT) { out.kind = 'project'; out.label = artifactKindLabel('project'); }
      } catch (e3) { /* 非 JSON 就当纯文本展示 */ }
    }
    out.text = txt;
    return out;
  }
  // 二进制（.glb / .stl——binary 与 ASCII 都可能）：base64 回前端解码后嗅探
  out.base64 = ctx.fs.readFileBase64(path);
  return out;
}

// claimArtifact：面板点选文件 = 「登记为识别到的文件」（进登记表 + 广播，其它面板/会话也看到）
function claimArtifactForUI(ctx, args) {
  var path = argStr(args || {}, 'path', '') || '';
  if (!path) fail('claimArtifact：缺 path');
  if (!ctx.fs.exists(path)) fail('文件不存在：' + path);
  var item = noteArtifact(ctx, path, artifactKindOf(path), 'claim', '面板点选登记');
  return { ok: true, item: item, count: ARTIFACTS.length };
}

// statArtifact：轻量状态查询（面板实时复核用——不读内容、不扫盘，只看 size/mtime）
// ★ 为什么需要它：登记表只记「工具写过的路径」，「文件内容」随时可能被 Agent / 外部编辑器改掉，
//   表里的 bytes/mtime 就过期了；若面板只比对登记表条目，就永远发现不了内容变化 → 预览不刷新。
//   故对**当前选中文件**一律走本方法取实时 size/mtime，变了就重读重绘。
function statArtifactForUI(ctx, args) {
  var path = argStr(args || {}, 'path', '') || '';
  if (!path) fail('statArtifact：缺 path');
  var st = null;
  try { st = ctx.fs.stat(path); } catch (e) { st = null; }
  if (!st || st.isDir) return { path: path, exists: false };
  return { path: path, exists: true, bytes: st.size, mtime: st.mtime, kind: artifactKindOf(path) };
}

// buildProject：点选工程 JSON 即用插件**同一份内核**现场构建，把几何数据交给面板渲染
// （工程即模型：改参数/加部件后重新调用即得到新几何 —— 面板侧无需内置内核）
function buildProjectForUI(ctx, args) {
  var path = argStr(args || {}, 'path', PROJECT_NAME) || PROJECT_NAME;
  var rm = readModelFile(ctx, path);
  var built = buildModelMeshes(rm.model, { skipInvisible: true });
  var parts = [], i, j;
  for (i = 0; i < built.parts.length; i++) {
    var p = built.parts[i], m = p.mesh, pos = [], idx = [];
    for (j = 0; j < m.positions.length; j++) {
      pos.push(cleanNum(m.positions[j][0], 6), cleanNum(m.positions[j][1], 6), cleanNum(m.positions[j][2], 6));
    }
    for (j = 0; j < m.indices.length; j++) idx.push(m.indices[j][0], m.indices[j][1], m.indices[j][2]);
    parts.push({
      id: p.id, name: p.name,
      color: (p.material && p.material.color) || DEFAULT_MATERIAL.color,
      pos: pos, idx: idx, tris: p.tris, volume: cleanNum(p.volume, 3)
    });
  }
  var bb = built.parts.length ? meshBbox(built.merged) : [[0, 0, 0], [0, 0, 0]];
  var params = [];
  for (i = 0; i < built.params.order.length; i++) {
    var pid = built.params.order[i], pdef = built.params.known[pid];
    params.push({
      id: pid, name: pdef.name === undefined ? pid : pdef.name,
      value: cleanNum(built.params.values[pid], 6),
      range: (pdef.min === undefined && pdef.max === undefined) ? null : (pdef.min + '..' + pdef.max)
    });
  }
  return {
    path: path, name: rm.model.name || 'model', unit: rm.model.unit || 'mm', up: rm.model.up || 'z',
    totalTris: built.totalTris,
    totalVolume: cleanNum(built.parts.reduce(function (s, x) { return s + x.volume; }, 0), 3),
    bbox: [
      [cleanNum(bb[0][0], 6), cleanNum(bb[0][1], 6), cleanNum(bb[0][2], 6)],
      [cleanNum(bb[1][0], 6), cleanNum(bb[1][1], 6), cleanNum(bb[1][2], 6)]
    ],
    parts: parts, params: params
  };
}

// ── 工具定义与插件入口（双轨：goja 顶层 return + Node module.exports）──
var IMPLS = {
  model_doc: modelDoc,
  model_add: modelAdd,
  model_edit: modelEdit,
  model_export: modelExport,
  model_verify: modelVerify
};

var TOOL_DEFS = [
  {
    name: 'model_doc',
    description: '参数化 CAD 工程的创建与查看（真相源 = 工程 JSON，默认 ' + PROJECT_NAME + '）。工程坐标照 CAD 惯例：右手系、+z 向上、单位 mm；几何字段可用表达式引用参数（如 "wall*2"）。action=new 建工程（可带 params + parts：一次把参数与全部部件都建好）/ get 看盘点（部件、参数、体积、包围盒）/ rename / param 改参数值（改完全模型自动重算）/ set 改元信息 / reset 清空部件。',
    usageGuide: '起步：model_doc action=new name=bracket params={wall:3, holes:2} → model_add id=base type=cuboid size=["wall*10","wall*10",5] → model_edit op.add 挖孔 → model_export format=all → model_verify。★ 部件多时走批量：action=new parts=[{id:"base", type:"cuboid", size:[40,30,6]}, {id:"boss", type:"cylinder", radius:5, height:10}] 一次建好工程 + 全部部件（不必 new 完再逐件 add）。已有工程先 action=get 看盘点（会顺带报"几何能否构建"）。',
    category: '创作',
    parameters: {
      type: 'object',
      properties: {
        action: { type: 'string', description: 'new（建工程）/ get（默认，看盘点）/ rename / param（改参数值）/ set（改 name/unit/up/note）/ reset（清空部件，需 force）' },
        path: { type: 'string', description: '可选：工程路径（默认 <主项目根>/' + PROJECT_NAME + '；相对主项目根解析）' },
        name: { type: 'string', description: '可选：模型名（new / rename / set）' },
        params: { type: 'object', description: '可选（new）：参数定义 [{id,value,min,max,name}] 或 {id: value}；可选（param）：{id: 新值} 或 [{id,value}]' },
        parts: { type: 'array', description: '可选（new）：部件定义数组——每项与 model_add 单件同构（id/from/type/size/shape/transform/repeat/material/ops/visible），一次调用把工程与全部部件一起建好；任一项构造或试建失败则整批不落盘' },
        id: { type: 'string', description: '可选（param）：单个参数 id' },
        value: { description: '可选（param）：该参数的新值（数字或表达式字符串）' },
        note: { type: 'string', description: '可选：备注（new / set）' },
        unit: { type: 'string', description: '可选（set）：目前只支持 mm' },
        up: { type: 'string', description: '可选（set）：目前只支持 z' },
        overwrite: { type: 'boolean', description: '可选（new）：工程已存在时是否重建（默认 false，不覆盖）' },
        force: { type: 'boolean', description: '可选（reset）：确认清空部件' }
      }
    }
  },
  {
    name: 'model_add',
    description: '新增部件。★ 支持批量：parts=[{…},{…}] 一次调用加任意多个部件（比逐件调用少几十倍往返；事务性：任一项失败整批不落盘），单件则用同层字段。几何可用 type + 字段快捷构造（cuboid/cube/sphere/ellipsoid/cylinder/cone/frustum/torus/polyhedron/extrude/revolve/sweep/mesh；extrude 可带 holes 打孔、带 twist 做扭转挤出；sweep 让轮廓沿任意 3D 路径扫掠，可配 holes/twist/scale/closed/up），也可传完整 shape（含布尔树 boolean/group）。可选 transform（translate/rotate/scale）、ops（对本体做 subtract/union/intersect，或网格级的 chamfer 倒角 / fillet 圆角 / repair / snap / align）、repeat（linear/circular/mirror 阵列）、material（color/metallic/roughness）。所有数值字段都能写表达式。新增时立即试算几何，算不出来不会写入。',
    usageGuide: '例：{id:"base", type:"cuboid", size:[40,30,6]}；带挖孔：{id:"base", type:"cuboid", size:[40,30,6], ops:[{op:"subtract", shape:{type:"cylinder", radius:3, height:20}}]}；带孔挤出板（更少三角形、更精确）：{id:"plate", type:"extrude", profile:{type:"rect", size:[40,30]}, holes:[{type:"circle", radius:3, center:[12,0]},{type:"circle", radius:3, center:[-12,0]}], height:6}；扭转挤出（绞龙/麻花柱/螺旋齿轮坯）：{id:"auger", type:"extrude", profile:{type:"circle", radius:8, segments:24}, height:40, twist:180, segments:24}；带孔扭转（内螺旋槽）：在上面加 holes:[{type:"circle", radius:3, segments:16}]；扫掠弯管（轮廓沿 3D 路径走，可带孔做空心管；扫掠路径字段叫 along）：{id:"elbow", type:"sweep", profile:{type:"circle", radius:4, segments:24}, holes:[{type:"circle", radius:2.5, segments:16}], along:[[0,0,0],[0,0,20],[0,8,30],[0,24,36]]}；扫掠圆环（闭合路径无端盖）：{id:"ring", type:"sweep", profile:{type:"circle", radius:3, segments:24}, along:[[20,0,0],[0,20,0],[-20,0,0],[0,-20,0]], closed:true}；锥形扫掠：直线 along + scale:[1,2]（每站缩放，也可给标量）；扭转扫掠：along + twist:360（沿路径总扭转角）；直线阵列：{id:"rib", type:"cuboid", size:[2,30,8], repeat:{mode:"linear", count:6, delta:[6,0,0]}}；圆周阵列：{repeat:{mode:"circular", count:8, axis:"z", radius:15}}；表达式尺寸：{size:["wall*10","wall*10","thk"]}。★ 批量（复杂模型一次成型 —— 别一件一次调用）：parts=[{id:"base", type:"cuboid", size:[40,30,6]}, {id:"boss", type:"cylinder", radius:5, height:10, transform:{translate:[0,0,8]}}, {id:"rib", type:"cuboid", size:[2,30,8], repeat:{mode:"linear", count:6, delta:[6,0,0]}}]。',
    category: '创作',
    parameters: {
      type: 'object',
      properties: {
        parts: { type: 'array', description: '★ 批量：部件定义数组——每项与单件同构（id/from/type/size/shape/transform/repeat/material/ops/visible），一次调用加任意多个部件；任一项构造或试建失败则整批不落盘。传了 parts 就不用传同层单件字段（也不用传 id）' },
        id: { type: 'string', description: '部件 id（单件时必填、唯一；字母数字下划线连字符点。批量时写在 parts 各项里）' },
        path: { type: 'string', description: '可选：工程路径（默认 <主项目根>/' + PROJECT_NAME + '；相对主项目根解析）' },
        type: { type: 'string', description: '几何类型（默认 cuboid）' },
        size: { description: 'cuboid/cube 全尺寸（数字或 [x,y,z]）' },
        center: { description: '可选：几何中心（默认原点）' },
        radius: { description: 'sphere/cylinder/cone 半径，或 torus 主半径' },
        radiusTop: { description: '可选：frustum 顶半径（cone 默认 0）' },
        radiusBottom: { description: '可选：frustum 底半径' },
        height: { description: 'cylinder/cone/extrude 高度（默认 2 / 1）' },
        innerRadius: { description: 'torus 管截面半径（须小于 outerRadius）' },
        outerRadius: { description: 'torus 主半径（默认 4）' },
        segments: { description: '分段数：曲面形状是圆周分段（默认 32）；extrude 配 twist 用时是**扭转分层数**（默认 12，每层扭转角须 < 180°）' },
        rings: { description: '可选：环向分段' },
        profile: { description: 'extrude/revolve/sweep 轮廓：{type:"rect"|"circle"|"star"|"polygon",...} 或点集 [[x,y],...]（sweep 时贴在路径每一站的局部 xy 平面上，x → 法向、y → 副法向）' },
        holes: { type: 'array', description: '可选（仅 extrude）：孔轮廓数组，每项同 profile 口径（支持 center 偏心）。一步挤出带孔板（比 subtract 布尔更少三角形、更精确）；洞须完全落在外轮廓内且互不相交，越界会报错。与 twist 同用时洞随外形一起扭转（内螺旋槽）。' },
        twist: { description: '可选（extrude/sweep）：总扭转角（度，默认 0 = 不扭转）。extrude 时轮廓沿 z 分 segments 层、每层绕 z 轴按高度线性旋转 twist/segments（分层数默认 12）；sweep 时沿路径按站号等比分配。每层/每段扭转角须 < 180°。' },
        along: { type: 'array', description: 'sweep 的扫掠路径：[[x,y,z],...]（≥2 点，可写表达式；相邻重合点自动去掉）。★ 名字不叫 path，因为工具参数里的 path 是工程文件路径。框架用平行传输（RMF）传播——直线段不会退化、拐点不会翻转；路径自交、180° 折返、曲率半径小于轮廓外接半径都会明确报错。' },
        closed: { type: 'boolean', description: '可选（sweep）：路径是否闭合（默认自动：首尾点重合即视为闭合）。闭合时无端盖（圆环/弯管环）；非闭合时两端封盖。★ 首尾不重合但语义闭合的路径需显式 closed:true。' },
        scale: { description: '可选（sweep）：沿路径的缩放，标量（恒定）或与站数等长的数组（可写表达式）——用于锥形/渐缩扫掠。' },
        up: { type: 'array', description: '可选（sweep）：[x,y,z] 参考方向，决定起始站框架的朝向（默认取与起始切线最不平行的坐标轴）。要固定扭转姿态时显式指定。' },
        angle: { description: '可选：revolve 角度（默认 360）' },
        points: { description: 'polyhedron 顶点 [[x,y,z],...]' },
        faces: { description: 'polyhedron 面（顶点下标）' },
        shape: { type: 'object', description: '可选：完整几何表达式（优先）；布尔树 {type:"boolean", op:"subtract", a:{...}, b:{...}}' },
        transform: { type: 'object', description: '可选：{translate:[x,y,z], rotate:[rx,ry,rz]（度，先 X 后 Y 后 Z）, scale:[sx,sy,sz]}' },
        ops: { type: 'array', description: '可选：[{op:"subtract"|"union"|"intersect", shape:{...}, transform:{...}}]；也支持网格级后处理 {op:"chamfer", distance}（倒角）/ {op:"fillet", radius, edges?, minAngle?, segments?}（圆角）/ {op:"repair"} / {op:"snap", grid} / {op:"align", to}' },
        repeat: { type: 'object', description: '可选：{mode:"linear",count,delta} / {mode:"circular",count,axis,radius} / {mode:"mirror",axis}' },
        material: { type: 'object', description: '可选：{color:"#88aadd", metallic:0..1, roughness:0..1}' },
        name: { type: 'string', description: '可选：显示名' },
        from: { type: 'string', description: '可选：从已有部件复制（id 换成新 id）' },
        visible: { type: 'boolean', description: '可选：false 则不参与导出' }
      }
    }
  },
  {
    name: 'model_edit',
    description: '命令式编辑工程（一次 ops 批量应用；任一 op 失败则整批不落盘，工程保持原样）。支持：部件（part.add 新增 / part.remove/rename/set/shape/transform/repeat/material/hide/show/reorder）、几何运算（op.add/remove/clear）、参数（param.add/set/remove）、网格处理（mesh.align/snap/repair/chamfer 倒角/fillet 圆角）。倒角与圆角作用在**网格级**（不是参数化特征）：按相邻面折角识别棱，默认处理全部凸棱、也可用坐标对数组精确指定；凹边（内二面角 > 180°）需填材料、暂不支持，指定时会明确报错。圆角对**凸体**默认（不指定 edges = 全部棱）走 rolling-ball 开运算解析路径 —— 相邻棱、多棱交会的顶点同样严格水密；只选部分棱或非凸体才走布尔近似路径。',
    usageGuide: '例：[{op:"op.add", id:"base", do:"subtract", shape:{type:"cylinder", radius:3, height:20}}] / [{op:"part.transform", id:"base", transform:{translate:[0,0,10]}}] / [{op:"param.set", id:"wall", value:4}]（全模型自动重算）/ [{op:"part.rename", id:"base", to:"plate"}] / [{op:"mesh.repair", id:"base"}]；倒角：[{op:"mesh.chamfer", id:"base", distance:1.5}]（全部凸棱；只做几条棱传 edges:[[[x,y,z],[x,y,z]],…]）—— 尺寸大或棱多时用倒角（一次构造、精确且水密）；圆角：[{op:"mesh.fillet", id:"base", radius:2}]（全部棱；只做几条棱传 edges:[[[x,y,z],[x,y,z]],…]，可传 segments 控制弧面离散）—— **凸体全棱（含相邻棱，如立方体 12 棱）走 rolling-ball 解析路径，严格水密**；只选部分棱或非凸体则走布尔近似，那时单次只支持**互不相邻**的棱（相邻棱交会会明确报错）。★ 批量：一次调用可传任意多条 op —— 复杂模型用 N 条 part.add 一次成型。例：[{op:"part.add", part:{id:"base", type:"cuboid", size:[40,30,6]}}, {op:"part.add", part:{id:"boss", type:"cylinder", radius:5, height:10, transform:{translate:[0,0,8]}}}, {op:"param.set", id:"wall", value:4}]。改完跑 model_verify 复验。',
    category: '创作',
    parameters: {
      type: 'object',
      properties: {
        path: { type: 'string', description: '可选：工程路径（默认 <主项目根>/' + PROJECT_NAME + '；相对主项目根解析）' },
        ops: { type: 'array', description: '编辑命令数组（按顺序应用）' },
        op: { type: 'string', description: '可选：单条 op 名（与同层参数合成为一条命令）' },
        id: { type: 'string', description: '可选：目标部件 / 参数 id' },
        to: { type: 'string', description: '可选：part.rename 的新 id，或 mesh.align 的方位（min/center/max）' },
        props: { type: 'object', description: '可选（part.set）：要合并的属性' },
        part: { type: 'object', description: '可选（part.add）：部件定义对象，与 model_add 的 parts 项同构（id/from/type/size/shape/transform/repeat/material/ops/visible）——一次调用可用多条 part.add 批量建部件' },
        shape: { type: 'object', description: '可选（part.shape / op.add / part.add 平铺写法）：几何' },
        do: { type: 'string', description: '可选（op.add）：subtract（默认）/ union / intersect' },
        transform: { type: 'object', description: '可选（part.transform / op.add）' },
        repeat: { type: 'object', description: '可选（part.repeat；传 null 清除阵列）' },
        material: { type: 'object', description: '可选（part.material）' },
        ids: { type: 'array', description: '可选（part.reorder）：新的顺序（列出的置前）' },
        index: { type: 'number', description: '可选（op.remove）：要删除的运算下标' },
        value: { description: '可选（param.add/set）：参数值' },
        min: { description: '可选（param.add）：下限' },
        max: { description: '可选（param.add）：上限' },
        grid: { description: '可选（mesh.snap）：吸附网格（默认按模型尺度）' },
        distance: { description: '可选（mesh.chamfer）：倒角量（面内尺寸，必须为正；可写表达式）。check：具体值须小于相邻面尺寸，过大（把实体切光）会报错' },
        radius: { description: '可选（mesh.fillet）：圆角半径（必须为正；可写表达式）。须小于最薄处尺寸（凸体全棱要求各面沿法线内缩 r 后实体仍在）；凸体全棱走 rolling-ball 时可含相邻棱，布尔近似路径则要求同一顶点处的半径不相互重叠' },
        edges: { description: '可选（mesh.chamfer / mesh.fillet）：只处理指定棱，传坐标对数组 [[[x,y,z],[x,y,z]],…]（端点须是焊接后的顶点坐标；与顶点编号顺序无关，可复现）。缺省 = 全部折角 ≥ minAngle 的凸棱' },
        minAngle: { description: '可选（mesh.chamfer / mesh.fillet）：认定"棱"的最小折角（度，默认 30）——相邻面外法线夹角小于它的边视为同一曲面（如圆柱侧面的分段边）不处理' },
        segments: { description: '可选（mesh.fillet）：弧面离散段数（默认 8，2~64）。越大越接近真圆、面数越多' }
      }
    }
  },
  {
    name: 'model_export',
    description: '导出产物（全部是公开格式，非自家方言）：gltf（Khronos glTF 2.0，自包含 base64 buffer，按规范把 +z-up 转 y-up）/ glb（单文件二进制容器）/ stl（binary STL，float32，3D 打印）/ stl-ascii / obj（Wavefront）/ html（自包含可交互 WebGL 预览）/ json（规范化工程）/ all（gltf+stl+stl-ascii+obj+html）。导出后顺带跑 9 项判据并返回结论。',
    usageGuide: '常用：model_export format=all（模型 + 打印文件 + 预览一次拿齐）；要上 3D 打印机用 format=stl；要进引擎/网页用 format=gltf 或 glb；要在面板里看用 format=html。只导出部分：parts=["base","lid"]；自定义路径：out=dist/plate.stl。',
    category: '创作',
    parameters: {
      type: 'object',
      properties: {
        format: { type: 'string', description: 'gltf（默认）/ glb / stl / stl-ascii / obj / html / json / all' },
        path: { type: 'string', description: '可选：工程路径（默认 <主项目根>/' + PROJECT_NAME + '；相对主项目根解析）' },
        out: { type: 'string', description: '可选：输出路径（默认按工程名派生：<主项目根>/model.gltf / model.stl / model.preview.html …）' },
        name: { type: 'string', description: '可选：导出模型名（写入 glTF node / STL 头）' },
        parts: { type: 'array', description: '可选：只导出指定部件 id' },
        withGlb: { type: 'boolean', description: '可选（format=all）：是否同时导出 .glb（默认 false）' }
      }
    }
  },
  {
    name: 'model_verify',
    description: '9 项判据自检，报告写到旁挂文件（默认 <工程名>.verify.json）。M1 工程契约 / M2 参数与表达式 / M3 网格完整性 / M4 水密性（可 3D 打印；区分"T 型接缝"与"真缺口"）/ M5 绕向一致且法线朝外 / M6 实体性（体积、包围盒占比、连通体）/ M7 变换与尺度 / M8 导出契约（glTF 结构、binary STL 长度与头、OBJ 1 基索引）/ M9 产物确定且自包含。',
    usageGuide: '每次改完几何跑一次：model_verify（会顺带校验同目录已导出的 model.gltf / model.stl / model.obj / model.preview.html）。要额外验证"两次构建位级一致"传 determinism=true。fails>0 必须处理；warns 是提示（M4 的 T 缝、M7 的尺度可疑等）。',
    category: '创作',
    parameters: {
      type: 'object',
      properties: {
        path: { type: 'string', description: '可选：工程路径（默认 <主项目根>/' + PROJECT_NAME + '；相对主项目根解析）' },
        report: { type: 'string', description: '可选：报告输出路径（默认 <主项目根>/<工程名>.verify.json）' },
        determinism: { type: 'boolean', description: '可选：是否额外验证"两次构建产物位级一致"（默认 false）' }
      }
    }
  }
];

var PLUGIN = {
  name: 'tool-model',
  purpose: '3D/CAD 创作域工具面（纯 goja 零依赖）：参数化 CAD 工程（表达式驱动）→ BSP 实体布尔（union/subtract/intersect）→ 导出公开格式（glTF 2.0 / GLB / binary+ASCII STL / OBJ / 自包含 WebGL 预览 HTML）→ 9 项判据（含分级的水密性报告）',
  inject: ['fs', 'logger'],
  apply: function (ctx, config) {
    // 可选回退：config.legacyMeshBoolean = true → 布尔走三角路径（I.6-1 及以前）；默认走多边形路径（I.6-2）
    if (config && config.legacyMeshBoolean) setMeshBooleanLegacy(true);
    for (var i = 0; i < TOOL_DEFS.length; i++) {
      (function (t) {
        ctx.tools.register({
          name: t.name,
          description: t.description,
          usageGuide: t.usageGuide,
          category: t.category,
          readOnly: !!t.readOnly,
          parameters: t.parameters,
          execute: function (args) { return IMPLS[t.name](args || {}, {}, ctx); }
        });
      })(TOOL_DEFS[i]);
    }
    try {
      if (ctx && ctx.logger && typeof ctx.logger === 'function') {
        ctx.logger('tool-model').info('已注册 ' + TOOL_DEFS.length + ' 个工具（goja 轨 · 零依赖 · glTF 2.0 / STL 真相源）');
      }
    } catch (_) { /* ignore */ }
    // ★ UI 半（client 半 + 面板 bundle 与本文件同包）：面板经 ui.invoke('tool-model', <名>, args)
    //   调用下列 host 方法 —— 见上方「识别到的文件」注册表（工具写文件即登记并广播，
    //   面板事件驱动实时刷新；点选文件即 claimArtifact 登记，点工程即 buildProject 现场构建）。
    try {
      ctx.registerClientMethod('listArtifacts', function (args) { return listArtifactsForUI(ctx, args || {}); });
      ctx.registerClientMethod('readArtifact', function (args) { return readArtifactForUI(ctx, args || {}); });
      ctx.registerClientMethod('claimArtifact', function (args) { return claimArtifactForUI(ctx, args || {}); });
      ctx.registerClientMethod('statArtifact', function (args) { return statArtifactForUI(ctx, args || {}); });
      ctx.registerClientMethod('buildProject', function (args) { return buildProjectForUI(ctx, args || {}); });
    } catch (e2) {
      try { ctx.logger('tool-model').warn('registerClientMethod 不可用（旧宿主）：面板退回 /api/fs/read 只读路径'); } catch (_) { /* ignore */ }
    }
  }
};

if (typeof module !== 'undefined' && module.exports) module.exports = PLUGIN;
return PLUGIN;
