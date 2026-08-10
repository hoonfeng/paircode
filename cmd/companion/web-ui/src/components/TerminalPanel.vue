<template>
  <div class="terminal-panel">
    <!-- 终端标签栏 -->
    <div class="term-tabs">
      <button v-for="(term, i) in terminals" :key="i"
              :class="['term-tab', { active: i === activeTermIdx }]"
              @click="switchTerm(i)"
              @mouseup.middle="closeTerm(i)">
        <SvgIcon name="terminal" :size="12" />
        <span class="term-tab-label">{{ term.label }}</span>
        <span class="term-tab-close" @click.stop="closeTerm(i)" title="关闭">×</span>
      </button>
      <button class="term-tab new-tab" @click="newTerminal" title="新建终端">
        <SvgIcon name="plus" :size="12" />
      </button>
      <!-- Shell 类型选择 -->
      <div class="term-shell-select" title="选择终端类型">
        <select v-model="defaultShell" @change="saveShellPreference">
          <option value="cmd">cmd</option>
          <option value="powershell">PowerShell</option>
          <option value="gitbash">Git Bash</option>
        </select>
      </div>
      <span class="term-tabs-filler"></span>
      <button class="term-tab term-panel-close" @click="$emit('close-panel')" title="关闭终端面板">
        <SvgIcon name="close" :size="12" />
      </button>
    </div>

    <!-- xterm.js 容器 -->
    <template v-if="terminals.length > 0">
      <div class="term-content" ref="contentRef">
        <div v-for="(term, i) in terminals" :key="term.id"
             :ref="el => setTermRef(i, el)"
             :class="['term-xterm-wrap', { 'term-hidden': i !== activeTermIdx }]">
        </div>
      </div>
    </template>

    <!-- 无终端时 -->
    <div v-else class="term-empty">
      <button class="term-create-btn" @click="newTerminal">新建终端</button>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted, onBeforeUnmount, nextTick, watch, inject } from 'vue'
import { state } from '../main.js'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import '@xterm/xterm/css/xterm.css'
import SvgIcon from './SvgIcon.vue'

defineEmits(['close-panel'])

const contentRef = ref(null)
const terminals = ref([])
const activeTermIdx = ref(-1)
const termRefs = reactive({})
let termCounter = 0
let resizeObserver = null

// ── 面板高度 watch 兜底（确定性修复：不依赖 ResizeObserver） ──
// App.vue provide('bottomPanelHeight')。面板拉伸（拖 panel-resizer）→
// bottomPanelHeight 变化 → 这里收到通知 → 等 Vue patch + 引擎布局稳定后
// 重新 fit 活动终端。即使 RO 回调因引擎时序/布局传导未触发，终端也会
// 跟随面板高度重算 rows（与浏览器对齐）。
const bottomPanelHeight = inject('bottomPanelHeight', null)
let panelResizeTimer = null
if (bottomPanelHeight) {
  watch(bottomPanelHeight, () => {
    if (panelResizeTimer) clearTimeout(panelResizeTimer)
    // 等 Vue patch + 引擎布局稳定后 fit。★ 引擎布局 dirty 传播在
    // 面板高度变化后可能需多帧才让 .term-xterm-wrap（absolute）拿到
    // 新高度——fit 一次可能读到旧值（rows 不变）。fitWithRetry 在
    // fit 后核对 rows 与容器是否匹配，不匹配则 150ms 重试（最多 40
    // 次 ≈ 6s），布局最终稳定后必成功（与浏览器对齐）。
    panelResizeTimer = setTimeout(() => {
      fitWithRetry(terminals.value[activeTermIdx.value], 40)
    }, 80)
  })
}

// ★ fit 重试兜底：wb-ui 引擎（desktop）在面板/容器尺寸变化后，布局的
// dirty 传播可能延迟多帧才更新 absolute 容器高度（.term-xterm-wrap
// top:0 bottom:0 → 高度 = 父内容区高度）。fitAddon.fit() 读
// getComputedStyle(wrap).height 可能长期拿到旧值（引擎 computed style
// 兜底读渲染树 box，而 box 高度在增量布局后未更新）→ rows 不重算。
// 这里 fit 后用 getBoundingClientRect（box 缺失时强制重建渲染树，能拿
// 到最新几何）核对 rows，不匹配则手动按实际几何 resize + 延迟重试，
// 直到布局稳定后 rows 与容器匹配（与浏览器对齐）。
function fitWithRetry(term, remaining) {
  if (!term || !term.fitAddon || !term.xterm) return
  try {
    // 首选 fitAddon.fit()（浏览器/布局已稳定时一次成功）
    term.fitAddon.fit()
    const wrap = term.xterm.element ? term.xterm.element.parentElement : null
    if (wrap) {
      const wr = wrap.getBoundingClientRect()
      if (wr && wr.height > 0) {
        const dims = term.xterm._core && term.xterm._core._renderService && term.xterm._core._renderService.dimensions
        const cellW = dims && dims.css ? dims.css.cell.width : 7.146788990825688
        const cellH = dims && dims.css ? dims.css.cell.height : 15
        const padV = 8 // .xterm padding 4px*2（上下）
        const padH = 8
        const expectCols = Math.max(1, Math.floor((wr.width - padH) / cellW))
        const expectRows = Math.max(1, Math.floor((wr.height - padV) / cellH))
        if ((term.xterm.cols !== expectCols || term.xterm.rows !== expectRows) && remaining > 0) {
          // fit 读到旧布局值 → 手动按实际几何 resize
          if (term.xterm.cols !== expectCols || term.xterm.rows !== expectRows) {
            term.xterm.resize(expectCols, expectRows)
          }
          if (term.xterm.rows !== expectRows || term.xterm.cols !== expectCols) {
            setTimeout(() => fitWithRetry(term, remaining - 1), 150)
            return
          }
        }
      }
    }
    if (term.ws) term.ws.resize(term.xterm.cols, term.xterm.rows)
  } catch (e) {}
}

// ── Shell 类型选择（持久化到 localStorage） ──
const defaultShell = ref(localStorage.getItem('term-default-shell') || 'cmd')
function saveShellPreference() {
  localStorage.setItem('term-default-shell', defaultShell.value)
}

// ── WebSocket 基础 URL ──
const proto = location.protocol === 'https:' ? 'wss:' : 'ws:'
const wsUrl = proto + '//' + location.host + '/api/terminal/ws'

// ── xterm 主题（从 CSS 变量动态读取） ──
function getXtermTheme() {
  const s = getComputedStyle(document.documentElement)
  const bg = s.getPropertyValue('--term-bg').trim() || '#0d1117'
  const fg = s.getPropertyValue('--term-text').trim() || '#e6edf3'
  const accent = s.getPropertyValue('--accent').trim() || '#58a6ff'
  const border = s.getPropertyValue('--border-color').trim() || '#30363d'
  const muted = s.getPropertyValue('--text-muted').trim() || '#6e7681'
  const sec = s.getPropertyValue('--text-secondary').trim() || '#8b949e'
  const prompt = s.getPropertyValue('--term-prompt').trim() || '#6a9955'
  return {
    background: bg,
    foreground: fg,
    cursor: accent,
    cursorAccent: bg,
    selectionBackground: accent + '44',
    black: border,
    red: '#e57373',
    green: prompt,
    yellow: '#d4a74e',
    blue: accent,
    magenta: '#b39ddb',
    cyan: '#4dd0e1',
    white: sec,
    brightBlack: muted,
    brightRed: '#ef9a9a',
    brightGreen: '#81c784',
    brightYellow: '#ffd54f',
    brightBlue: accent,
    brightMagenta: '#ce93d8',
    brightCyan: '#80deea',
    brightWhite: fg,
  }
}

// ── 创建 xterm 实例 ──
function createXtermInstance(domEl) {
  const isDesktopMode = typeof window !== 'undefined' && !!window.__DESKTOP_MODE__
  const terminal = new Terminal({
    cursorBlink: true,
    // ★ 光标样式：浏览器终端（Edge 实测）是竖线光标（bar），用户要求
    // 与浏览器对齐。xterm bar 光标的闪烁动画默认用 box-shadow（引擎
    // 动画系统不驱动 → 静止不闪），因此在下方注入 .xterm-cursor-bar
    // 的覆盖样式（background-color 闪烁 + width:2px 竖线），引擎动画
    // 系统支持 background-color → 正常闪烁，视觉与浏览器一致。
    cursorStyle: 'bar',
    fontSize: 13,
    fontFamily: "'Consolas', 'Cascadia Code', 'JetBrains Mono', monospace",
    theme: getXtermTheme(),
    allowProposedApi: true,
    cols: 80,
    rows: 24,
    scrollback: 5000,
    convertEol: true,
    // ★ desktop（wb-ui 引擎）无 HTMLCanvasElement → xterm canvas 渲染器
    // 无法创建；DOM 渲染器（纯 div/span）是 xterm 官方选项，浏览器标准
    // 语义一致。web 版保持默认 canvas 渲染（性能更好）。
    rendererType: isDesktopMode ? 'dom' : 'auto',
  })

  const fitAddon = new FitAddon()
  terminal.loadAddon(fitAddon)

  terminal.open(domEl)
  fitAddon.fit()
  // ★ 调试：暴露实例供 WB_TERM_TEST 检查 buffer/渲染状态
  window.__lastTerm = terminal

  return { terminal, fitAddon }
}

// ── 创建 WebSocket 连接 ──
function createWebSocket(termState, xtermInstance, fitAddon) {
  let socket = null
  let reconnectTimer = null
  let reconnectCount = 0
  const MAX_RECONNECT = 3
  let closed = false

  function connect() {
    if (closed) return
    try {
      socket = new WebSocket(wsUrl)
    } catch {
      scheduleReconnect()
      return
    }

    let initSent = false

    socket.onopen = () => {
      reconnectCount = 0
      // 发送 init 消息
      const cols = xtermInstance.cols || 80
      const rows = xtermInstance.rows || 24
      const msg = JSON.stringify({
        type: 'init',
        shell: termState.shell || 'cmd',
        cwd: termState.cwd || '',
        cols,
        rows,
      })
      socket.send(msg)
      initSent = true
      termState.status = 'connected'
    }

    socket.onmessage = (ev) => {
      window.__termRecv = (window.__termRecv || 0) + 1
      window.__termRecvLast = (typeof ev.data === 'string') ? ev.data.slice(0, 40) : String(ev.data).slice(0, 40)
      // 二进制帧 = PTY 输出（浏览器 Blob）
      if (typeof Blob !== 'undefined' && ev.data instanceof Blob) {
        ev.data.arrayBuffer().then(buf => {
          const uint8 = new Uint8Array(buf)
          xtermInstance.write(uint8)
        })
        return
      }
      // ★ desktop（wb-ui 引擎）：无 Blob 构造器，PTY 输出以字符串
      // 推送（含 VT 转义序列，xterm.write 直接渲染）。控制消息
      // （ready/error/closed）由 desktop 桥接通过 onopen/onerror/onclose
      // 表达，不推送 JSON。
      if (typeof ev.data === 'string') {
        xtermInstance.write(ev.data)
        return
      }
      // 文本帧 = JSON 控制消息（浏览器模式兜底）
      try {
        const data = JSON.parse(ev.data)
        if (data.type === 'ready') {
          termState.status = 'ready'
          // 连接就绪后重新 fit
          try { fitAddon.fit() } catch {}
        } else if (data.type === 'error') {
          xtermInstance.writeln('\r\n\x1b[31m[终端错误] ' + (data.msg || '未知错误') + '\x1b[0m')
          termState.status = 'error'
        } else if (data.type === 'closed') {
          xtermInstance.writeln('\r\n\x1b[33m[终端会话已关闭]\x1b[0m')
          termState.status = 'closed'
          closed = true
        }
      } catch {}
    }

    socket.onerror = () => {
      xtermInstance.writeln('\r\n\x1b[31m[终端连接错误]\x1b[0m')
      termState.status = 'error'
    }

    socket.onclose = () => {
      termState.status = 'disconnected'
      socket = null
      if (!closed) {
        scheduleReconnect()
      }
    }
  }

  function scheduleReconnect() {
    if (closed || reconnectCount >= MAX_RECONNECT) return
    reconnectCount++
    const delay = Math.min(1000 * Math.pow(2, reconnectCount - 1), 8000)
    reconnectTimer = setTimeout(connect, delay)
  }

  // 键盘输入 → WebSocket 二进制帧（后端 opcode 0x2 才识别为键盘输入）
  const disposeData = xtermInstance.onData(data => {
    if (socket && socket.readyState === WebSocket.OPEN) {
      // 使用 TextEncoder 转为 UTF-8 字节序列发送二进制帧（而非文本帧）
      socket.send(new TextEncoder().encode(data))
    }
  })

  // 关闭连接
  function close() {
    closed = true
    disposeData.dispose()
    if (reconnectTimer) {
      clearTimeout(reconnectTimer)
      reconnectTimer = null
    }
    if (socket) {
      try {
        const msg = JSON.stringify({ type: 'close' })
        socket.send(msg)
      } catch {}
      socket.close()
      socket = null
    }
  }

  // 启动首次连接
  connect()

  return {
    close,
    get socket() { return socket },
    /**
     * 通知后端终端尺寸变化
     */
    resize(cols, rows) {
      if (socket && socket.readyState === WebSocket.OPEN) {
        socket.send(JSON.stringify({ type: 'resize', cols, rows }))
      }
    },
  }
}

// ── 创建终端（xterm + WebSocket） ──
function createTerminal(cwd, shell) {
  termCounter++
  const id = 'term-' + termCounter + '-' + Date.now().toString(36)
  const s = shell || 'cmd'
  return reactive({
    id,
    label: s === 'powershell' ? 'PS ' + termCounter : (s === 'gitbash' ? 'Bash ' + termCounter : '终端 ' + termCounter),
    cwd: cwd || '',
    shell: s,
    status: 'init', // init / connected / ready / error / closed / disconnected
    xterm: null,
    fitAddon: null,
    ws: null,
    domEl: null,
  })
}

// ── 设置 DOM 引用并挂载 xterm ──
function setTermRef(idx, el) {
  if (!el) return
  const term = terminals.value[idx]
  if (!term) return
  if (term.domEl === el) return
  term.domEl = el

  // ★ 延迟创建 xterm：wb-ui 引擎（desktop）在 Vue mount 瞬间布局未稳定，
  //   此时 xterm 用 offsetHeight 测量字符尺寸会得到 NaN/0 → 缓存 NaN →
  //   style.height="NaNpx" → 行高异常、终端内容画到视口外（空白）。
  //   延迟 80ms 等主循环完成布局后再创建，测量即正常。
  //   （浏览器无此问题：DOM 插入后同步 reflow 可得真实尺寸。）
  setTimeout(() => {
    if (!term.domEl || term.xterm) return
    // 创建 xterm 实例
    const { terminal, fitAddon } = createXtermInstance(term.domEl)
    term.xterm = terminal
    term.fitAddon = fitAddon

    // 创建 WebSocket 连接
    const ws = createWebSocket(term, terminal, fitAddon)
    term.ws = ws

    // ★ 创建后立即 fit（引擎已修复：getComputedStyle 未声明 padding 返回
    // "0px" + canvas 精确测量 → fit 一次即得正确 cols/rows，不再需要
    // 200/800/2000ms 多档重试——重试让终端启动最坏等 2 秒才显示正确列数）。
    try {
      term.fitAddon.fit()
      // ★ 全量 refresh：xterm open 时首测 cellW 可能错误（引擎字体/布局
      // 未完全就绪 → 首次 measure fallback 8px/cell），初始加载的信息
      // （banner/prompt）若按旧 cellW 渲染，与聚焦重测后的行（7.146px）
      // 空格间距不一致。fit 已触发 resize→重测→重建，此处再显式全量
      // 刷新当前视口，保证初始行与后续行用同一 cellW（与浏览器对齐）。
      try { terminal.refresh(0, terminal.rows - 1) } catch (e) {}
      if (term.ws && term.ws.readyState === 1) {
        term.ws.resize(term.xterm.cols, term.xterm.rows)
      }
    } catch (e) {}

    // 如果该终端是当前活动终端，添加 ResizeObserver 监听
    if (idx === activeTermIdx.value) {
      try {
        observeActiveTerm(term.domEl, term)
      } catch (e) {}
    }
  }, 80)
}

// ── 监听活动终端的尺寸变化 ──
function observeActiveTerm(el, term) {
  if (resizeObserver) resizeObserver.disconnect()

  let resizeTimer = null
  resizeObserver = new ResizeObserver(() => {
    if (resizeTimer) clearTimeout(resizeTimer)
    resizeTimer = setTimeout(() => {
      fitWithRetry(term, 20)
    }, 100)
  })

  if (el) resizeObserver.observe(el)
}

// ── 切换终端标签页 ──
function switchTerm(idx) {
  if (idx === activeTermIdx.value) return
  activeTermIdx.value = idx

  nextTick(() => {
    const term = terminals.value[idx]
    if (!term) return

    // 如果 xterm 还没挂载（延迟创建），等待 ref 回调
    if (!term.xterm && term.domEl) {
      const { terminal, fitAddon } = createXtermInstance(term.domEl)
      term.xterm = terminal
      term.fitAddon = fitAddon

      const ws = createWebSocket(term, terminal, fitAddon)
      term.ws = ws
    }

    // 激活后重新 fit
    if (term.fitAddon) {
      try { term.fitAddon.fit() } catch {}
    }
    // ★ 全量刷新当前视口（保证行 span 用最新 cellW，初始信息与聚焦后一致）
    if (term.xterm) {
      try { term.xterm.refresh(0, term.xterm.rows - 1) } catch {}
    }

    // 自动聚焦 xterm，用户可直接输入
    if (term.xterm) {
      try { term.xterm.focus() } catch {}
    }

    // 切换 ResizeObserver 到新活动终端
    if (term.domEl) {
      observeActiveTerm(term.domEl, term)
    }
  })
}

// ── 新建终端 ──
function newTerminal() {
  const prevTerm = activeTermIdx.value >= 0 ? terminals.value[activeTermIdx.value] : null
  const cwd = prevTerm?.cwd || ''
  const term = createTerminal(cwd, defaultShell.value)
  terminals.value.push(term)
  switchTerm(terminals.value.length - 1)
  saveTerminals()
}

// ── 关闭终端 ──
function closeTerm(i) {
  if (terminals.value.length <= 1) return
  const term = terminals.value[i]
  // 销毁 WebSocket 连接
  if (term.ws) {
    term.ws.close()
    term.ws = null
  }
  // 销毁 xterm 实例
  if (term.xterm) {
    term.xterm.dispose()
    term.xterm = null
    term.fitAddon = null
  }
  terminals.value.splice(i, 1)
  if (activeTermIdx.value >= terminals.value.length) {
    activeTermIdx.value = terminals.value.length - 1
  }
  saveTerminals()
  // 切换后重新 fit
  nextTick(() => {
    const t = terminals.value[activeTermIdx.value]
    if (t && t.fitAddon) {
      try { t.fitAddon.fit() } catch {}
    }
  })
}

// ── 保存/恢复终端列表 ──
const TERM_KEY = 'paircode-terminals'

function saveTerminals() {
  try {
    const data = terminals.value.map(t => ({
      label: t.label,
      cwd: t.cwd,
      shell: t.shell,
    }))
    localStorage.setItem(TERM_KEY, JSON.stringify(data))
  } catch {}
}

function loadTerminals() {
  try {
    const raw = localStorage.getItem(TERM_KEY)
    if (!raw) return false
    const data = JSON.parse(raw)
    if (!Array.isArray(data) || data.length === 0) return false
    for (const t of data) {
      termCounter++
      terminals.value.push(reactive({
        id: 'term-' + termCounter + '-' + Date.now().toString(36),
        label: t.label || '终端 ' + termCounter,
        cwd: t.cwd || '',
        shell: t.shell || 'cmd',
        status: 'init',
        xterm: null,
        fitAddon: null,
        ws: null,
        domEl: null,
      }))
    }
    if (terminals.value.length > 0) {
      activeTermIdx.value = 0
      return true
    }
    return false
  } catch { return false }
}

// ── 生命周期 ──
onMounted(() => {
  if (!loadTerminals()) {
    newTerminal()
  }
})

onBeforeUnmount(() => {
  // 关闭所有 WebSocket 和 xterm
  for (const term of terminals.value) {
    if (term.ws) {
      term.ws.close()
      term.ws = null
    }
    if (term.xterm) {
      term.xterm.dispose()
      term.xterm = null
    }
  }
  if (resizeObserver) {
    resizeObserver.disconnect()
    resizeObserver = null
  }
})

// ── 主题切换时刷新 xterm 配色 ──
watch(() => state.theme, () => {
  const theme = getXtermTheme()
  for (const term of terminals.value) {
    if (term.xterm && term.xterm.setOption) {
      term.xterm.setOption('theme', theme)
    }
  }
})
</script>

<style>
/* xterm.css 部分覆盖 — 容器自适应 */
.terminal-panel .xterm {
  height: 100%;
  padding: 4px;
}
.terminal-panel .xterm-viewport {
  overflow-y: auto !important;
  scrollbar-width: thin;
}
/* ★ wb-ui 引擎对 xterm DOM renderer 字符 span 的 height:100% 解析异常
   （100% 用了父宽度 572px 当高度）→ 每列 span 高 572 垂直堆叠 → 终端
   多列错位。强制 height auto（内容高 15px）恢复浏览器式行内排列。 */
.terminal-panel .xterm-rows div span {
  height: auto !important;
}
.terminal-panel .xterm-screen {
  width: 100% !important;
}
/* ★ bar 光标（浏览器标准竖线）：xterm 的 blink_bar 动画用 box-shadow
   （wb-ui 引擎不驱动 → 光标静止）。覆盖为 background-color 闪烁 +
   2px 竖线——引擎动画系统支持 background-color → 正常闪烁，视觉与
   浏览器终端一致（1 字符宽竖线、随字符闪烁）。
   ★ keyframes 用平台式（0/49.9% 蓝 + 50/100% 透明）：引擎对 CSS
   timing-function 的 step-end 按线性处理（不支持步骤语义），若 keyframes
   只写 0%/100% 蓝 + 50% 透明 → 线性插值 → 光标「渐隐渐现」（不干脆）。
   平台式使两段关键帧内插值恒值、只在 50% 瞬间跳变 → 干脆开关切换，
   与浏览器 step-end 视觉效果一致。 */
.terminal-panel .xterm-cursor.xterm-cursor-bar {
  box-shadow: none !important;
  background-color: #58a6ff !important;
  width: 1px !important;
  animation: wb-term-bar-blink 1s step-end infinite !important;
}
@keyframes wb-term-bar-blink {
  0%, 49.9% { background-color: #58a6ff; }
  50%, 100% { background-color: transparent; }
}
</style>

<style scoped>
.terminal-panel {
  display: flex; flex-direction: column; height: 100%;
  background: var(--term-bg); color: var(--term-text); font-size: 13px;
}
/* ── 终端标签栏 ── */
.term-tabs {
  display: flex; align-items: stretch; background: var(--bg-secondary);
  border-bottom: 1px solid var(--border-color); flex-shrink: 0; overflow-x: auto;
  /* ★ 高度对齐用户浏览器（真实 Edge 9090）：Edge 环境字体行高较大，
     tab 内容自然撑到 33px → xterm 容器 147px → 内容底部间隙 15px。
     引擎字体行高较小（内容 25px），min-height 28 只撑到 28 → xterm
     容器 151px → 间隙 19px（比用户浏览器大 4px）。统一 min-height
     33px 使引擎渲染 = 用户浏览器（tab 33px、间隙 15px）。 */
  min-height: 33px;
}
.term-tab {
  display: flex; align-items: center; gap: 4px; padding: 4px 10px;
  background: none; border: none; border-right: 1px solid var(--border-color);
  color: var(--text-secondary); font-size: 12px; cursor: pointer; white-space: nowrap;
  user-select: none;
}
.term-tab.active {
  background: var(--term-bg); color: var(--text-primary);
  border-bottom: 1px solid var(--term-bg); margin-bottom: -1px;
}
.term-tab:hover:not(.active) { background: var(--bg-hover); }
.term-tab-close { font-size: 12px; margin-left: 2px; opacity: 0.5; }
.term-tab-close:hover { opacity: 1; color: #e57373; }
.term-tab.new-tab { padding: 4px 8px; }
.term-shell-select select {
  font-size: 11px; background: transparent; color: var(--text-secondary);
  border: 1px solid var(--border-color); border-radius: 3px; padding: 1px 4px;
  margin: 0 4px; cursor: pointer; height: 22px;
}
.term-shell-select select:hover { border-color: var(--accent); }
.term-tabs-filler { flex: 1; }
.term-panel-close { opacity: 0.5; padding: 4px 8px; }
.term-panel-close:hover { opacity: 1; color: #e57373; }

/* ── 终端内容区 ── */
.term-content {
  flex: 1; display: flex; flex-direction: column; min-height: 0;
  position: relative; overflow: hidden;
}
.term-xterm-wrap {
  flex: 1; min-height: 0; position: absolute;
  top: 0; left: 0; right: 0; bottom: 0;
}
.term-xterm-wrap.term-hidden {
  visibility: hidden; pointer-events: none;
}

.term-empty {
  flex: 1; display: flex; align-items: center; justify-content: center;
}
.term-create-btn {
  background: #58a6ff; color: #000; border: none;
  padding: 6px 16px; border-radius: 4px; cursor: pointer; font-size: 13px;
}
.term-create-btn:hover { background: #79c0ff; }
</style>
