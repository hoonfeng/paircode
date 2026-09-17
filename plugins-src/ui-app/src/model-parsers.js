// ═══════════════════════════════════════════════════════════════
// model-parsers.js — 「识别到的文件」→ 可渲染几何（3D 模型面板的解析层）
//
// 面板点开任意识别到的文件时，前端**当场解析**并交给 ModelViewer 渲染
// （host 只回内容：文本原样 / 二进制 base64，见 tool-model 的 readArtifact）。
//
// 支持格式（与 tool-model 导出/可识别口径一致）：
//   · STL —— binary 与 ASCII 自动嗅探（3D 打印事实标准）
//   · OBJ —— Wavefront（v / f，含 v/vt/vn 与负索引、多边形扇形三角化）
//   · glTF 2.0 —— 自包含 JSON（buffer 走 base64 data URI）
//   · GLB —— 单文件二进制容器（12 字节头 + JSON 块 + BIN 块）
//
// 统一输出：
//   { parts: [{ id, name, color, pos: Float32Array, idx: Uint32Array }], note }
// 每三角独立顶点（flat shading，与插件导出侧同口径）——预览不做网格拓扑还原。
// ═══════════════════════════════════════════════════════════════

export const DEFAULT_COLOR = '#b0b4bd'

// base64 → Uint8Array（避免 atob 大字符串逐字符开销）
export function base64ToUint8(b64) {
  const bin = atob(String(b64 || ''))
  const out = new Uint8Array(bin.length)
  for (let i = 0; i < bin.length; i++) out[i] = bin.charCodeAt(i) & 0xff
  return out
}

function isBinarySTL(buf) {
  if (buf.byteLength < 84) return false
  const dv = new DataView(buf.buffer, buf.byteOffset, buf.byteLength)
  const n = dv.getUint32(80, true)
  if (84 + n * 50 === buf.byteLength) return true
  // 头 5 字节是 "solid" 且长度不匹配 → 视为 ASCII
  const head = String.fromCharCode(buf[0], buf[1], buf[2], buf[3], buf[4]).toLowerCase()
  return head !== 'solid'
}

// ── STL ────────────────────────────────────────────────────────
function stlFromTriples(triples) {
  const n = triples.length / 9
  const pos = new Float32Array(n * 9)
  const idx = new Uint32Array(n * 3)
  for (let i = 0; i < n * 9; i++) pos[i] = triples[i]
  for (let i = 0; i < n * 3; i++) idx[i] = i
  return { pos, idx }
}

export function parseSTLBinary(buf) {
  const dv = new DataView(buf.buffer, buf.byteOffset, buf.byteLength)
  const count = dv.getUint32(80, true)
  const pos = new Float32Array(count * 9)
  const idx = new Uint32Array(count * 3)
  let o = 84
  for (let t = 0; t < count; t++) {
    o += 12 // 面法线（渲染器按顶点重算，忽略）
    for (let v = 0; v < 3; v++) {
      pos[t * 9 + v * 3] = dv.getFloat32(o, true)
      pos[t * 9 + v * 3 + 1] = dv.getFloat32(o + 4, true)
      pos[t * 9 + v * 3 + 2] = dv.getFloat32(o + 8, true)
      o += 12
    }
    idx[t * 3] = t * 3
    idx[t * 3 + 1] = t * 3 + 1
    idx[t * 3 + 2] = t * 3 + 2
    o += 2 // 属性字节数
  }
  return { pos, idx }
}

export function parseSTLAscii(text) {
  const triples = []
  const re = /vertex\s+(-?[\d.eE+]+)\s+(-?[\d.eE+]+)\s+(-?[\d.eE+]+)/g
  let m
  while ((m = re.exec(text)) !== null) {
    triples.push(Number(m[1]), Number(m[2]), Number(m[3]))
  }
  const trimmed = triples.length - (triples.length % 9)
  return stlFromTriples(triples.slice(0, trimmed))
}

export function parseSTL(buf, textIfAscii) {
  if (isBinarySTL(buf)) return { geo: parseSTLBinary(buf), note: 'binary STL' }
  const text = textIfAscii !== undefined ? textIfAscii : new TextDecoder().decode(buf)
  return { geo: parseSTLAscii(text), note: 'ASCII STL' }
}

// ── OBJ ────────────────────────────────────────────────────────
export function parseOBJ(text) {
  const V = []
  const F = []
  const lines = String(text || '').split(/\r?\n/)
  for (const line of lines) {
    const s = line.trim()
    if (s === '' || s.charAt(0) === '#') continue
    if (s.charAt(0) === 'v' && (s.charAt(1) === ' ' || s.charAt(1) === '\t')) {
      const p = s.split(/\s+/)
      V.push(Number(p[1]) || 0, Number(p[2]) || 0, Number(p[3]) || 0)
      continue
    }
    if (s.charAt(0) === 'f' && (s.charAt(1) === ' ' || s.charAt(1) === '\t')) {
      const p = s.split(/\s+/).slice(1)
      const verts = []
      for (const tok of p) {
        const idxStr = tok.split('/')[0]
        if (idxStr === '') continue
        let i = parseInt(idxStr, 10)
        if (!isFinite(i)) continue
        i = i < 0 ? (V.length / 3 + i) : (i - 1) // OBJ 1 基；负数从尾部数
        verts.push(i)
      }
      // 多边形扇形三角化
      for (let k = 1; k + 1 < verts.length; k++) F.push(verts[0], verts[k], verts[k + 1])
    }
  }
  const pos = new Float32Array(V.length)
  for (let i = 0; i < V.length; i++) pos[i] = V[i]
  return { pos, idx: new Uint32Array(F) }
}

// ── glTF / GLB ─────────────────────────────────────────────────
const COMPONENT = { 5120: Int8Array, 5121: Uint8Array, 5122: Int16Array, 5123: Uint16Array, 5125: Uint32Array, 5126: Float32Array }
const NCOMP = { SCALAR: 1, VEC2: 2, VEC3: 3, VEC4: 4, MAT4: 16 }
const COMP_SIZE = { 5120: 1, 5121: 1, 5122: 2, 5123: 2, 5125: 4, 5126: 4 }

function gltfBuffers(json, binChunk) {
  const out = []
  const list = json.buffers || []
  for (let i = 0; i < list.length; i++) {
    const b = list[i]
    if (b.uri && b.uri.indexOf('data:') === 0) {
      const comma = b.uri.indexOf(',')
      out.push(base64ToUint8(b.uri.slice(comma + 1)))
    } else if (i === 0 && binChunk) {
      out.push(binChunk)
    } else {
      out.push(new Uint8Array(0))
    }
  }
  return out
}

function readAccessor(json, buffers, index) {
  const acc = (json.accessors || [])[index]
  if (!acc) return null
  const view = (json.bufferViews || [])[acc.bufferView]
  if (!view) return null
  const buf = buffers[view.buffer] || new Uint8Array(0)
  const Ctor = COMPONENT[acc.componentType] || Float32Array
  const n = NCOMP[acc.type] || 1
  const size = COMP_SIZE[acc.componentType] || 4
  const stride = view.byteStride || n * size
  const base = (view.byteOffset || 0) + (acc.byteOffset || 0)
  const out = new Ctor(acc.count * n)
  for (let i = 0; i < acc.count; i++) {
    const off = base + i * stride
    for (let c = 0; c < n; c++) {
      const p = off + c * size
      switch (acc.componentType) {
        case 5126: out[i * n + c] = new DataView(buf.buffer, buf.byteOffset + p, 4).getFloat32(0, true); break
        case 5125: out[i * n + c] = new DataView(buf.buffer, buf.byteOffset + p, 4).getUint32(0, true); break
        case 5123: out[i * n + c] = new DataView(buf.buffer, buf.byteOffset + p, 2).getUint16(0, true); break
        case 5122: out[i * n + c] = new DataView(buf.buffer, buf.byteOffset + p, 2).getInt16(0, true); break
        case 5121: out[i * n + c] = buf[p]; break
        case 5120: out[i * n + c] = new DataView(buf.buffer, buf.byteOffset + p, 1).getInt8(0); break
        default: out[i * n + c] = 0
      }
    }
  }
  return out
}

// mat4（列主序，glTF 口径）：从 node 的 matrix 或 TRS 构造
function mat4Identity() { return [1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1] }
function mat4Mul(a, b) {
  const o = new Array(16)
  for (let c = 0; c < 4; c++) {
    for (let r = 0; r < 4; r++) {
      o[c * 4 + r] = a[0 * 4 + r] * b[c * 4 + 0] + a[1 * 4 + r] * b[c * 4 + 1] + a[2 * 4 + r] * b[c * 4 + 2] + a[3 * 4 + r] * b[c * 4 + 3]
    }
  }
  return o
}
function mat4FromTRS(node) {
  if (node.matrix) return node.matrix.slice()
  const t = node.translation || [0, 0, 0]
  const r = node.rotation || [0, 0, 0, 1]
  const s = node.scale || [1, 1, 1]
  const [x, y, z, w] = r
  const x2 = x + x, y2 = y + y, z2 = z + z
  const xx = x * x2, xy = x * y2, xz = x * z2
  const yy = y * y2, yz = y * z2, zz = z * z2
  const wx = w * x2, wy = w * y2, wz = w * z2
  return [
    (1 - (yy + zz)) * s[0], (xy + wz) * s[0], (xz - wy) * s[0], 0,
    (xy - wz) * s[1], (1 - (xx + zz)) * s[1], (yz + wx) * s[1], 0,
    (xz + wy) * s[2], (yz - wx) * s[2], (1 - (xx + yy)) * s[2], 0,
    t[0], t[1], t[2], 1,
  ]
}
function applyMat4(m, x, y, z) {
  return [
    m[0] * x + m[4] * y + m[8] * z + m[12],
    m[1] * x + m[5] * y + m[9] * z + m[13],
    m[2] * x + m[6] * y + m[10] * z + m[14],
  ]
}
function colorOf(json, matIdx) {
  const mt = ((json.materials || [])[matIdx] || {}).pbrMetallicRoughness || {}
  const f = mt.baseColorFactor
  if (!f || f.length < 3) return DEFAULT_COLOR
  const h = (v) => Math.max(0, Math.min(255, Math.round(v * 255))).toString(16).padStart(2, '0')
  return '#' + h(f[0]) + h(f[1]) + h(f[2])
}

export function parseGLTF(json, binChunk) {
  const buffers = gltfBuffers(json, binChunk)
  const parts = []
  const scene = (json.scenes || [])[json.scene || 0] || {}
  const roots = scene.nodes || (json.nodes || []).map((_, i) => i)
  const walk = (nodeIdx, parentMat) => {
    const node = (json.nodes || [])[nodeIdx]
    if (!node) return
    const world = mat4Mul(parentMat, mat4FromTRS(node))
    if (node.mesh !== undefined) {
      const mesh = (json.meshes || [])[node.mesh]
      const prims = (mesh && mesh.primitives) || []
      for (let pi = 0; pi < prims.length; pi++) {
        const prim = prims[pi]
        const posAcc = readAccessor(json, buffers, prim.attributes && prim.attributes.POSITION)
        if (!posAcc) continue
        const idxAcc = prim.indices === undefined ? null : readAccessor(json, buffers, prim.indices)
        const vcount = posAcc.length / 3
        const pos = new Float32Array(vcount * 3)
        for (let i = 0; i < vcount; i++) {
          const p = applyMat4(world, posAcc[i * 3], posAcc[i * 3 + 1], posAcc[i * 3 + 2])
          // ★ glTF 按规范是 y-up（导出侧把工程的 z-up 转过了），这里转回工程口径 z-up：
          //   预览相机统一以 +Z 为上，否则同一模型在 STL（工程坐标）与 glTF 里朝向不一致。
          pos[i * 3] = p[0]; pos[i * 3 + 1] = -p[2]; pos[i * 3 + 2] = p[1]
        }
        let idx
        if (idxAcc) {
          idx = new Uint32Array(idxAcc.length)
          for (let i = 0; i < idxAcc.length; i++) idx[i] = idxAcc[i]
        } else {
          idx = new Uint32Array(vcount)
          for (let i = 0; i < vcount; i++) idx[i] = i
        }
        parts.push({
          id: (mesh && mesh.name) || ('mesh' + node.mesh),
          name: node.name || (mesh && mesh.name) || ('mesh' + node.mesh),
          color: colorOf(json, prim.material),
          pos, idx,
        })
      }
    }
    for (const child of node.children || []) walk(child, world)
  }
  for (const r of roots) walk(r, mat4Identity())
  return parts
}

export function parseGLB(buf) {
  const dv = new DataView(buf.buffer, buf.byteOffset, buf.byteLength)
  if (dv.getUint32(0, true) !== 0x46546c67) throw new Error('不是 GLB（magic 不匹配）')
  const total = dv.getUint32(8, true)
  let o = 12
  let json = null
  let bin = null
  while (o + 8 <= Math.min(total, buf.byteLength)) {
    const len = dv.getUint32(o, true)
    const type = dv.getUint32(o + 4, true)
    const start = o + 8
    if (type === 0x4e4f534a) json = JSON.parse(new TextDecoder().decode(buf.subarray(start, start + len)))
    else if (type === 0x004e4942) bin = buf.subarray(start, start + len)
    o = start + len + ((4 - (len % 4)) % 4)
  }
  if (!json) throw new Error('GLB 缺 JSON 块')
  return parseGLTF(json, bin)
}

// ── 统一入口：按面板给的 kind + 内容解析 ─────────────────────────
// artifact: { path, kind, text?, base64? }（tool-model readArtifact 的返回）
export function parseArtifact(artifact) {
  const kind = artifact && artifact.kind
  const name = String((artifact && artifact.path) || '').split(/[\\/]/).pop()
  if (kind === 'gltf') {
    const json = JSON.parse(artifact.text)
    return { parts: parseGLTF(json), note: 'glTF 2.0 自包含（base64 buffer）', format: 'glTF 2.0' }
  }
  if (kind === 'glb') {
    const buf = base64ToUint8(artifact.base64)
    return { parts: parseGLB(buf), note: 'GLB 单文件二进制容器', format: 'GLB' }
  }
  if (kind === 'stl') {
    const buf = base64ToUint8(artifact.base64)
    const ascii = isBinarySTL(buf) ? undefined : new TextDecoder().decode(buf)
    const r = parseSTL(buf, ascii)
    return { parts: [{ id: name, name, color: DEFAULT_COLOR, pos: r.geo.pos, idx: r.geo.idx }], note: r.note, format: 'STL' }
  }
  if (kind === 'obj') {
    const g = parseOBJ(artifact.text)
    return { parts: [{ id: name, name, color: DEFAULT_COLOR, pos: g.pos, idx: g.idx }], note: 'Wavefront OBJ', format: 'OBJ' }
  }
  throw new Error('该文件类型不做几何预览：' + (kind || '未知'))
}

// 包围盒（面板显示尺寸用）
export function partsBbox(parts) {
  let mn = [Infinity, Infinity, Infinity]
  let mx = [-Infinity, -Infinity, -Infinity]
  for (const p of parts) {
    for (let i = 0; i < p.pos.length; i += 3) {
      for (let c = 0; c < 3; c++) {
        const v = p.pos[i + c]
        if (v < mn[c]) mn[c] = v
        if (v > mx[c]) mx[c] = v
      }
    }
  }
  if (!isFinite(mn[0])) return null
  return { min: mn, max: mx, size: [mx[0] - mn[0], mx[1] - mn[1], mx[2] - mn[2]] }
}

export function partsTriCount(parts) {
  let n = 0
  for (const p of parts) n += p.idx.length / 3
  return n
}
