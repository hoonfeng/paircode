<template>
  <div ref="wrapRef" class="mv" :class="{ 'mv-full': fill }">
    <canvas ref="cvRef" class="mv-canvas"></canvas>
    <div v-if="failMsg" class="mv-fail">{{ failMsg }}</div>
    <div class="mv-hint">拖拽=旋转 · 滚轮=缩放 · 右键拖拽=平移 · <code>W</code>=线框 · <code>R</code>=复位</div>
    <div class="mv-bar">
      <span class="mv-stat">{{ stat }}</span>
      <span class="mv-spacer"></span>
      <button class="mv-btn" :class="{ on: wire }" @click="toggleWire">{{ wire ? '线框 开' : '线框 关' }}</button>
      <button class="mv-btn" @click="resetCam">复位</button>
    </div>
  </div>
</template>

<script setup>
// ModelViewer — 3D 预览渲染器（自研 WebGL1，无外部依赖；与 tool-model 导出的
// 自包含预览页同一套渲染口径：轨道相机 / Lambert + 高光 / 线框 / 深色底）。
//
// 输入 props.parts：[{ id, name, color, pos, idx }] —— 来自 model-parsers 的解析结果
//   或 host 半 buildProject 的现场构建结果。渲染时按索引展开成**每三角独立顶点**
//   （flat shading，与导出侧同口径），展开后顶点顺序即索引顺序 → 直接用 drawArrays，
//   绕开 16 位索引上限（大模型无需扩展）。
//
// 交互：左键拖拽旋转 / 滚轮缩放 / 右键拖拽平移 / W 线框 / R 复位；相机自动框住包围盒。
// ═══════════════════════════════════════════════════════════════
import { ref, onMounted, onBeforeUnmount, watch } from 'vue'

const props = defineProps({
  parts: { type: Array, default: () => [] },
  fill: { type: Boolean, default: false },
})

const wrapRef = ref(null)
const cvRef = ref(null)
const failMsg = ref('')
const wire = ref(false)
const stat = ref('')

let gl = null
let prog = null
let loc = {}
let meshes = []          // [{ count, posBuf, nrmBuf, color, lineBuf, lineCount, use32 }]
let wireBuf = null
let cam = { theta: 0.62, phi: 1.06, dist: 120, target: [0, 0, 0] }
let raf = 0
let needs = true
let disposed = false
let drag = null

const VS = 'attribute vec3 aPos;attribute vec3 aNrm;uniform mat4 uProj,uView;varying vec3 vN,vW;' +
  'void main(){vN=aNrm;vW=aPos;gl_Position=uProj*uView*vec4(aPos,1.0);}'
const FS = 'precision mediump float;varying vec3 vN,vW;uniform vec3 uColor,uCam;uniform float uWire;' +
  'void main(){if(uWire>0.5){gl_FragColor=vec4(uColor*1.35,1.0);return;}' +
  'vec3 n=normalize(vN);vec3 l1=normalize(vec3(0.45,0.75,0.55)),l2=normalize(vec3(-0.6,-0.35,-0.7));' +
  'float d=max(dot(n,l1),0.0)*0.8+max(dot(n,l2),0.0)*0.22;vec3 v=normalize(uCam-vW);' +
  'vec3 h=normalize(l1+v);float s=pow(max(dot(n,h),0.0),36.0)*0.28;' +
  'vec3 amb=vec3(0.26,0.28,0.32);gl_FragColor=vec4(uColor*(amb+d)+vec3(s),1.0);}'

function compile(type, src) {
  const s = gl.createShader(type)
  gl.shaderSource(s, src)
  gl.compileShader(s)
  if (!gl.getShaderParameter(s, gl.COMPILE_STATUS)) {
    throw new Error('着色器编译失败：' + gl.getShaderInfoLog(s))
  }
  return s
}

function initGL() {
  const cv = cvRef.value
  // preserveDrawingBuffer：画布内容随时可取（验证/导出截图用）。预览场景无连续重绘，
  // 性能代价可忽略；否则 toDataURL/drawImage 在帧外取到的是空缓冲。
  gl = cv.getContext('webgl', { preserveDrawingBuffer: true }) || cv.getContext('experimental-webgl')
  if (!gl) {
    failMsg.value = '当前浏览器/环境不支持 WebGL，无法渲染 3D 预览'
    return false
  }
  prog = gl.createProgram()
  gl.attachShader(prog, compile(gl.VERTEX_SHADER, VS))
  gl.attachShader(prog, compile(gl.FRAGMENT_SHADER, FS))
  gl.linkProgram(prog)
  gl.useProgram(prog)
  loc = {
    aPos: gl.getAttribLocation(prog, 'aPos'),
    aNrm: gl.getAttribLocation(prog, 'aNrm'),
    uProj: gl.getUniformLocation(prog, 'uProj'),
    uView: gl.getUniformLocation(prog, 'uView'),
    uColor: gl.getUniformLocation(prog, 'uColor'),
    uCam: gl.getUniformLocation(prog, 'uCam'),
    uWire: gl.getUniformLocation(prog, 'uWire'),
  }
  gl.enableVertexAttribArray(loc.aPos)
  gl.enableVertexAttribArray(loc.aNrm)
  gl.enable(gl.DEPTH_TEST)
  gl.enable(gl.CULL_FACE)
  gl.cullFace(gl.BACK)
  // 调试锚（自动化验证/排障用）：属性与 uniform 的位置是否有效、GL 是否有错误
  if (wrapRef.value) {
    wrapRef.value.dataset.uloc = 'aPos=' + loc.aPos + ',aNrm=' + loc.aNrm +
      ',uProj=' + (loc.uProj ? 1 : 0) + ',uView=' + (loc.uView ? 1 : 0) +
      ',uColor=' + (loc.uColor ? 1 : 0) + ',uWire=' + (loc.uWire ? 1 : 0) +
      ',linked=' + (gl.getProgramParameter(prog, gl.LINK_STATUS) ? 1 : 0)
    wrapRef.value.dataset.glerr = String(gl.getError())
  }
  return true
}

function hexToRgb(c) {
  let s = String(c || '#b0b4bd').replace('#', '')
  if (s.length === 3) s = s.charAt(0) + s.charAt(0) + s.charAt(1) + s.charAt(1) + s.charAt(2) + s.charAt(2)
  const n = parseInt(s, 16)
  if (!isFinite(n)) return [0.69, 0.71, 0.74]
  return [((n >> 16) & 255) / 255, ((n >> 8) & 255) / 255, (n & 255) / 255]
}

// 展开：索引 → 每三角独立顶点（同时算出面法线）
function buildMeshes() {
  releaseMeshes()
  if (!gl) return
  const use32 = !!gl.getExtension('OES_element_index_uint')
  wireBuf = gl.createBuffer()
  const parts = props.parts || []
  let total = 0
  for (const p of parts) {
    const idx = p.idx || []
    const pos = p.pos || []
    const triCount = Math.floor(idx.length / 3)
    if (!triCount) continue
    const vp = new Float32Array(triCount * 9)
    const vn = new Float32Array(triCount * 9)
    const lines = new Uint32Array(triCount * 6)
    for (let t = 0; t < triCount; t++) {
      const i0 = idx[t * 3] * 3, i1 = idx[t * 3 + 1] * 3, i2 = idx[t * 3 + 2] * 3
      const ax = pos[i0], ay = pos[i0 + 1], az = pos[i0 + 2]
      const bx = pos[i1], by = pos[i1 + 1], bz = pos[i1 + 2]
      const cx = pos[i2], cy = pos[i2 + 1], cz = pos[i2 + 2]
      vp[t * 9] = ax; vp[t * 9 + 1] = ay; vp[t * 9 + 2] = az
      vp[t * 9 + 3] = bx; vp[t * 9 + 4] = by; vp[t * 9 + 5] = bz
      vp[t * 9 + 6] = cx; vp[t * 9 + 7] = cy; vp[t * 9 + 8] = cz
      let nx = (by - ay) * (cz - az) - (bz - az) * (cy - ay)
      let ny = (bz - az) * (cx - ax) - (bx - ax) * (cz - az)
      let nz = (bx - ax) * (cy - ay) - (by - ay) * (cx - ax)
      const len = Math.sqrt(nx * nx + ny * ny + nz * nz) || 1
      nx /= len; ny /= len; nz /= len
      for (let v = 0; v < 3; v++) {
        vn[t * 9 + v * 3] = nx; vn[t * 9 + v * 3 + 1] = ny; vn[t * 9 + v * 3 + 2] = nz
      }
      const base = t * 3
      lines[t * 6] = base; lines[t * 6 + 1] = base + 1
      lines[t * 6 + 2] = base + 1; lines[t * 6 + 3] = base + 2
      lines[t * 6 + 4] = base + 2; lines[t * 6 + 5] = base
    }
    const posBuf = gl.createBuffer()
    gl.bindBuffer(gl.ARRAY_BUFFER, posBuf)
    gl.bufferData(gl.ARRAY_BUFFER, vp, gl.STATIC_DRAW)
    const nrmBuf = gl.createBuffer()
    gl.bindBuffer(gl.ARRAY_BUFFER, nrmBuf)
    gl.bufferData(gl.ARRAY_BUFFER, vn, gl.STATIC_DRAW)
    const lineBuf = gl.createBuffer()
    gl.bindBuffer(gl.ELEMENT_ARRAY_BUFFER, lineBuf)
    const lineArr = (triCount * 3 > 65535 && !use32) ? new Uint16Array(lines.subarray(0, Math.min(lines.length, 65535 * 6))) : (use32 ? lines : new Uint16Array(lines))
    gl.bufferData(gl.ELEMENT_ARRAY_BUFFER, lineArr, gl.STATIC_DRAW)
    meshes.push({
      count: triCount * 3, posBuf, nrmBuf, color: hexToRgb(p.color),
      lineBuf, lineCount: lineArr.length, use32,
    })
    total += triCount
  }
  // 相机框住包围盒
  let mn = [Infinity, Infinity, Infinity], mx = [-Infinity, -Infinity, -Infinity]
  for (const p of props.parts || []) {
    const pos = p.pos || []
    for (let i = 0; i < pos.length; i += 3) {
      for (let c = 0; c < 3; c++) {
        const v = pos[i + c]
        if (v < mn[c]) mn[c] = v
        if (v > mx[c]) mx[c] = v
      }
    }
  }
  let radius = 50
  if (isFinite(mn[0])) {
    cam.target = [(mn[0] + mx[0]) / 2, (mn[1] + mx[1]) / 2, (mn[2] + mx[2]) / 2]
    radius = Math.max(1e-3, Math.sqrt(
      (mx[0] - mn[0]) ** 2 + (mx[1] - mn[1]) ** 2 + (mx[2] - mn[2]) ** 2))
    cam.dist = radius * 1.9
  } else {
    cam.target = [0, 0, 0]
  }
  cam.lastRadius = radius   // 复位用（框住包围盒的那个半径）
  stat.value = total.toLocaleString() + ' 面 · ' + meshes.length + ' 部件'
  if (wrapRef.value) {
    // 自动化验证用（web_debug 可读 DOM 断言渲染确实发生）
    wrapRef.value.dataset.tris = String(total)
    wrapRef.value.dataset.parts = String(meshes.length)
    wrapRef.value.dataset.webgl = gl ? '1' : '0'
    wrapRef.value.dataset.cam = 'dist=' + cam.dist.toFixed(2) +
      ',target=' + cam.target.map((v) => v.toFixed(2)).join('/') +
      ',theta=' + cam.theta.toFixed(3) + ',phi=' + cam.phi.toFixed(3) +
      ',radius=' + (cam.lastRadius === undefined ? '-' : Number(cam.lastRadius).toFixed(2))
  }
  needs = true
}

function releaseMeshes() {
  for (const m of meshes) {
    gl.deleteBuffer(m.posBuf)
    gl.deleteBuffer(m.nrmBuf)
    gl.deleteBuffer(m.lineBuf)
  }
  meshes = []
  if (wireBuf) { gl.deleteBuffer(wireBuf); wireBuf = null }
}

function perspective(fovy, aspect, near, far) {
  const f = 1 / Math.tan(fovy / 2)
  const nf = 1 / (near - far)
  return new Float32Array([
    f / aspect, 0, 0, 0,
    0, f, 0, 0,
    0, 0, (far + near) * nf, -1,
    0, 0, 2 * far * near * nf, 0,
  ])
}
function lookAt(eye, center, up) {
  const zx = eye[0] - center[0], zy = eye[1] - center[1], zz = eye[2] - center[2]
  let zl = Math.sqrt(zx * zx + zy * zy + zz * zz) || 1
  const z = [zx / zl, zy / zl, zz / zl]
  let x = [up[1] * z[2] - up[2] * z[1], up[2] * z[0] - up[0] * z[2], up[0] * z[1] - up[1] * z[0]]
  let xl = Math.sqrt(x[0] * x[0] + x[1] * x[1] + x[2] * x[2]) || 1
  x = [x[0] / xl, x[1] / xl, x[2] / xl]
  const y = [
    z[1] * x[2] - z[2] * x[1],
    z[2] * x[0] - z[0] * x[2],
    z[0] * x[1] - z[1] * x[0],
  ]
  return new Float32Array([
    x[0], y[0], z[0], 0,
    x[1], y[1], z[1], 0,
    x[2], y[2], z[2], 0,
    -(x[0] * eye[0] + x[1] * eye[1] + x[2] * eye[2]),
    -(y[0] * eye[0] + y[1] * eye[1] + y[2] * eye[2]),
    -(z[0] * eye[0] + z[1] * eye[1] + z[2] * eye[2]),
    1,
  ])
}

function draw() {
  raf = 0
  if (disposed || !gl) return
  const cv = cvRef.value
  const dpr = Math.min(2, window.devicePixelRatio || 1)
  const w = Math.max(1, Math.floor(cv.clientWidth * dpr))
  const h = Math.max(1, Math.floor(cv.clientHeight * dpr))
  if (cv.width !== w || cv.height !== h) { cv.width = w; cv.height = h }
  gl.viewport(0, 0, w, h)
  gl.clearColor(0.078, 0.086, 0.102, 1)
  gl.clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
  if (!meshes.length) return

  // ★ 相机以 **+Z 为上**（与 CAD 工程口径一致：右手系 / z-up / mm）——用 y-up 会把水平
  //   底板"竖起来"，与工程语义不符。eye 用 z 为极轴的球坐标。
  const sp = Math.sin(cam.phi), cp = Math.cos(cam.phi)
  const eye = [
    cam.target[0] + cam.dist * sp * Math.cos(cam.theta),
    cam.target[1] + cam.dist * sp * Math.sin(cam.theta),
    cam.target[2] + cam.dist * cp,
  ]
  gl.uniformMatrix4fv(loc.uProj, false, perspective(Math.PI / 4, w / h, Math.max(0.01, cam.dist / 500), cam.dist * 6))
  gl.uniformMatrix4fv(loc.uView, false, lookAt(eye, cam.target, [0, 0, 1]))
  gl.uniform3f(loc.uCam, eye[0], eye[1], eye[2])
  // 一次性把实际生效的矩阵/视口写到 DOM（排障锚，成本仅第一次）
  if (wrapRef.value && !wrapRef.value.dataset.proj) {
    const p4 = (m) => Array.from(m).map((v) => Number(v).toFixed(3)).join(',')
    wrapRef.value.dataset.proj = p4(perspective(Math.PI / 4, w / h, Math.max(0.01, cam.dist / 500), cam.dist * 6))
    wrapRef.value.dataset.view = p4(lookAt(eye, cam.target, [0, 0, 1]))
    wrapRef.value.dataset.viewport = w + 'x' + h
  }

  for (const m of meshes) {
    gl.bindBuffer(gl.ARRAY_BUFFER, m.posBuf)
    gl.vertexAttribPointer(loc.aPos, 3, gl.FLOAT, false, 0, 0)
    gl.bindBuffer(gl.ARRAY_BUFFER, m.nrmBuf)
    gl.vertexAttribPointer(loc.aNrm, 3, gl.FLOAT, false, 0, 0)
    gl.uniform3f(loc.uColor, m.color[0], m.color[1], m.color[2])
    if (wire.value) {
      gl.uniform1f(loc.uWire, 1)
      gl.bindBuffer(gl.ELEMENT_ARRAY_BUFFER, m.lineBuf)
      gl.drawElements(gl.LINES, m.lineCount, m.use32 ? gl.UNSIGNED_INT : gl.UNSIGNED_SHORT, 0)
    } else {
      gl.uniform1f(loc.uWire, 0)
      gl.drawArrays(gl.TRIANGLES, 0, m.count)
    }
  }
  // 一次性自检：读回像素统计模型在屏幕上的实际范围（排障锚；readPixels 在 draw 同帧内有效）
  //   （画布像素 > 2.5M 时跳过，避免 4K 场景一次性读回几十 MB）
  if (wrapRef.value && !wrapRef.value.dataset.pixbbox && w * h <= 2500000) {
    try {
      const px = new Uint8Array(w * h * 4)
      gl.readPixels(0, 0, w, h, gl.RGBA, gl.UNSIGNED_BYTE, px)
      let mnx = 1e9, mxx = -1, mny = 1e9, mxy = -1, n = 0
      for (let i = 0; i < px.length; i += 4) {
        if (Math.abs(px[i] - 20) > 14 || Math.abs(px[i + 1] - 22) > 14 || Math.abs(px[i + 2] - 26) > 14) {
          const p = i / 4
          const x = p % w
          const y = (p / w) | 0        // GL 像素原点在左下
          if (x < mnx) mnx = x
          if (x > mxx) mxx = x
          if (y < mny) mny = y
          if (y > mxy) mxy = y
          n++
        }
      }
      wrapRef.value.dataset.pixbbox = JSON.stringify({ n, mnx, mxx, mny, mxy, w, h })
    } catch (e) { wrapRef.value.dataset.pixbbox = 'err:' + (e && e.message) }
  }
}

// ★ 尺寸变化必须触发重绘：面板/视图从隐藏→可见、侧栏宽度变化、tab 切换都会改变
//   canvas 的 CSS 尺寸；若只在挂载时画一次，画面会在错误的视口/宽高比下被拉伸
//   （实测症状：模型被放大并偏到画面一角）。
let lastW = 0
let lastH = 0
function tick() {
  raf = 0
  if (disposed) return
  const cv = cvRef.value
  if (cv) {
    const w = cv.clientWidth
    const h = cv.clientHeight
    if (w !== lastW || h !== lastH) { lastW = w; lastH = h; needs = true }
  }
  if (needs) { needs = false; draw() }
  raf = requestAnimationFrame(tick)
}

function invalidate() { needs = true }

function onDown(e) {
  const right = e.button === 2
  drag = { x: e.clientX, y: e.clientY, right }
  e.preventDefault()
}
// 相机基向量（z-up）：d = target→eye 方向，r = 屏幕右，u = 屏幕上
function camBasis() {
  const sp = Math.sin(cam.phi), cp = Math.cos(cam.phi)
  const d = [sp * Math.cos(cam.theta), sp * Math.sin(cam.theta), cp]
  let r = [-d[1], d[0], 0]
  const rl = Math.sqrt(r[0] * r[0] + r[1] * r[1]) || 1
  r = [r[0] / rl, r[1] / rl, 0]
  const u = [d[1] * r[2] - d[2] * r[1], d[2] * r[0] - d[0] * r[2], d[0] * r[1] - d[1] * r[0]]
  return { r, u }
}
function onMove(e) {
  if (!drag) return
  const dx = e.clientX - drag.x, dy = e.clientY - drag.y
  drag.x = e.clientX
  drag.y = e.clientY
  if (drag.right) {
    // 平移：沿相机右/上方向移动目标点（z-up 基向量）
    const scale = cam.dist / 600
    const b = camBasis()
    cam.target[0] += (-dx * b.r[0] + dy * b.u[0]) * scale
    cam.target[1] += (-dx * b.r[1] + dy * b.u[1]) * scale
    cam.target[2] += (-dx * b.r[2] + dy * b.u[2]) * scale
  } else {
    cam.theta -= dx * 0.01
    // z 极轴：phi ∈ (0, π)（0=正上方俯视，π=正下方仰视）
    cam.phi = Math.max(0.15, Math.min(Math.PI - 0.15, cam.phi + dy * 0.01))
  }
  invalidate()
}
function onUp() { drag = null }
function onWheel(e) {
  e.preventDefault()
  cam.dist = Math.max(0.05, cam.dist * (e.deltaY > 0 ? 1.12 : 0.89))
  invalidate()
}
function onKey(e) {
  const k = String(e.key || '').toLowerCase()
  if (k === 'w') { wire.value = !wire.value; e.preventDefault() }
  if (k === 'r') resetCam()
}
function resetCam() {
  cam.theta = 0.62
  cam.phi = 1.06
  const r = cam.lastRadius
  cam.dist = r ? r * 1.9 : cam.dist
  invalidate()
}
function toggleWire() { wire.value = !wire.value }
function onCtx(e) { e.preventDefault() }

onMounted(() => {
  if (!initGL()) return
  buildMeshes()
  const cv = cvRef.value
  cv.addEventListener('mousedown', onDown)
  window.addEventListener('mousemove', onMove)
  window.addEventListener('mouseup', onUp)
  cv.addEventListener('wheel', onWheel, { passive: false })
  cv.addEventListener('contextmenu', onCtx)
  cv.setAttribute('tabindex', '0')
  cv.addEventListener('keydown', onKey)
  raf = requestAnimationFrame(tick)
})

onBeforeUnmount(() => {
  disposed = true
  if (raf) cancelAnimationFrame(raf)
  const cv = cvRef.value
  if (cv) {
    cv.removeEventListener('mousedown', onDown)
    cv.removeEventListener('wheel', onWheel)
    cv.removeEventListener('contextmenu', onCtx)
    cv.removeEventListener('keydown', onKey)
  }
  window.removeEventListener('mousemove', onMove)
  window.removeEventListener('mouseup', onUp)
  if (gl) releaseMeshes()
})

watch(() => props.parts, () => { buildMeshes() }, { deep: false })
watch(wire, () => { invalidate() })
</script>

<style scoped>
/* 高度显式给定（不依赖父容器高度链）：侧栏面板用 320px；主内容区 tab 用更大高度，
   容器高度塌陷时也不会把 3D 画面压扁/拉伸。 */
.mv { position: relative; width: 100%; height: 320px; background: #14161a; border: 1px solid var(--border-color); border-radius: 4px; overflow: hidden; }
.mv-full { height: 100%; min-height: 340px; }
.mv-canvas { display: block; width: 100%; height: 100%; outline: none; }
.mv-hint { position: absolute; right: 8px; bottom: 6px; color: var(--text-muted); font-size: 10px; pointer-events: none; }
.mv-hint code { color: var(--text-secondary); }
.mv-bar { position: absolute; left: 8px; top: 6px; right: 8px; display: flex; align-items: center; gap: 6px; }
.mv-spacer { flex: 1; }
.mv-stat { color: var(--text-muted); font-size: 10px; font-variant-numeric: tabular-nums; }
.mv-btn {
  padding: 2px 8px; font-size: 10px; color: var(--text-secondary);
  background: rgba(20, 22, 26, 0.7); border: 1px solid var(--border-color); border-radius: 4px; cursor: pointer;
}
.mv-btn:hover { background: var(--bg-hover); }
.mv-btn.on { color: var(--text-primary); border-color: var(--text-muted); }
.mv-fail { position: absolute; inset: 0; display: flex; align-items: center; justify-content: center; padding: 12px; color: var(--text-muted); font-size: 12px; text-align: center; }
</style>
