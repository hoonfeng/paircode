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
import { ref, reactive, computed, onMounted, onBeforeUnmount, nextTick, watch } from 'vue'
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
  const terminal = new Terminal({
    cursorBlink: true,
    cursorStyle: 'bar',
    fontSize: 13,
    fontFamily: "'Consolas', 'Cascadia Code', 'JetBrains Mono', monospace",
    theme: getXtermTheme(),
    allowProposedApi: true,
    cols: 80,
    rows: 24,
    scrollback: 5000,
    convertEol: true,
  })

  const fitAddon = new FitAddon()
  terminal.loadAddon(fitAddon)

  terminal.open(domEl)
  fitAddon.fit()

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
      // 二进制帧 = PTY 输出
      if (ev.data instanceof Blob) {
        ev.data.arrayBuffer().then(buf => {
          const uint8 = new Uint8Array(buf)
          xtermInstance.write(uint8)
        })
        return
      }
      // 文本帧 = JSON 控制消息
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

  // 键盘输入 → WebSocket 二进制
  const disposeData = xtermInstance.onData(data => {
    if (socket && socket.readyState === WebSocket.OPEN) {
      socket.send(data)
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

  // 创建 xterm 实例
  const { terminal, fitAddon } = createXtermInstance(el)
  term.xterm = terminal
  term.fitAddon = fitAddon

  // 创建 WebSocket 连接
  const ws = createWebSocket(term, terminal, fitAddon)
  term.ws = ws

  // 如果该终端是当前活动终端，添加 ResizeObserver 监听
  if (idx === activeTermIdx.value) {
    observeActiveTerm(el, term)
  }
}

// ── 监听活动终端的尺寸变化 ──
function observeActiveTerm(el, term) {
  if (resizeObserver) resizeObserver.disconnect()

  let resizeTimer = null
  resizeObserver = new ResizeObserver(() => {
    if (resizeTimer) clearTimeout(resizeTimer)
    resizeTimer = setTimeout(() => {
      if (term && term.fitAddon && term.xterm) {
        try {
          term.fitAddon.fit()
          if (term.ws) {
            term.ws.resize(term.xterm.cols, term.xterm.rows)
          }
        } catch {}
      }
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
.terminal-panel .xterm-screen {
  width: 100% !important;
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
  min-height: 28px;
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
