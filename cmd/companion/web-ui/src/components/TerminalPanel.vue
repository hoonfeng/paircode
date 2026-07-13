<template>
  <div class="terminal-panel">
    <!-- 终端标签栏 -->
    <div class="term-tabs">
      <button v-for="(tab, i) in tabs" :key="tab.id"
              :class="['term-tab', { active: i === activeIdx }]"
              @click="switchTab(i)"
              @mouseup.middle="closeTab(i)">
        <SvgIcon name="terminal" :size="12" />
        <span class="term-tab-label">{{ tab.label }}</span>
        <span v-if="tabs.length > 1" class="term-tab-close" @click.stop="closeTab(i)" title="关闭">×</span>
      </button>
      <button class="term-tab new-tab" @click="showNewDialog = true" title="新建终端">
        <SvgIcon name="plus" :size="12" />
      </button>
    </div>

    <!-- 终端容器 -->
    <div class="term-container">
      <div v-for="tab in tabs" :key="tab.id"
           :ref="el => setTermRef(tab.id, el)"
           :class="['term-xterm', { active: tab.id === activeId }]">
      </div>
    </div>

    <!-- 无终端时 -->
    <div v-if="tabs.length === 0" class="term-empty">
      <button class="term-create-btn" @click="showNewDialog = true">新建终端</button>
    </div>

    <!-- 新建终端对话框 -->
    <div v-if="showNewDialog" class="term-dialog-overlay" @click.self="showNewDialog = false">
      <div class="term-dialog">
        <h3>新建终端</h3>
        <div class="dialog-row">
          <label>Shell 类型：</label>
          <select v-model="newShell">
            <option value="cmd">CMD</option>
            <option value="powershell">PowerShell</option>
            <option value="gitbash">Git Bash</option>
          </select>
        </div>
        <div class="dialog-row">
          <label>工作目录：</label>
          <input v-model="newCwd" class="dialog-input" placeholder="默认工作区根目录" />
        </div>
        <div class="dialog-actions">
          <button class="btn-cancel" @click="showNewDialog = false">取消</button>
          <button class="btn-confirm" @click="createTerminal">创建</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, nextTick } from 'vue'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import '@xterm/xterm/css/xterm.css'
import SvgIcon from './SvgIcon.vue'

// ─── 状态 ──────────────────────────────────────────────────

const tabs = ref([])          // { id, label, shell, term, fitAddon, ws }
const activeIdx = ref(-1)
const showNewDialog = ref(false)
const newShell = ref('cmd')
const newCwd = ref('')

let termIdCounter = 0

const activeId = ref(null)

// DOM 引用映射：tabId → HTMLDivElement
const termRefs = {}

function setTermRef(tabId, el) {
  if (el) termRefs[tabId] = el
}

// ─── 创建终端 ──────────────────────────────────────────────

function createTerminal() {
  showNewDialog.value = false
  termIdCounter++
  const id = 'term-' + termIdCounter
  const shell = newShell.value
  const cwd = newCwd.value || ''

  const tab = {
    id,
    label: shellLabel(shell) + ' ' + termIdCounter,
    shell,
    cwd,
    term: null,
    fitAddon: null,
    ws: null,
  }
  tabs.value.push(tab)
  activeIdx.value = tabs.value.length - 1
  activeId.value = id

  nextTick(() => {
    initTerminal(tab)
  })
}

function shellLabel(shell) {
  switch (shell) {
    case 'powershell': return 'PowerShell'
    case 'gitbash': return 'Bash'
    default: return 'CMD'
  }
}

// ─── 初始化 xterm.js + WebSocket ──────────────────────────

function initTerminal(tab) {
  const container = termRefs[tab.id]
  if (!container) {
    console.error('[Terminal] 未找到容器:', tab.id)
    return
  }

  // 创建 xterm
  const term = new Terminal({
    cursorBlink: true,
    cursorStyle: 'block',
    fontSize: 13,
    fontFamily: "'Consolas', 'Cascadia Code', 'Courier New', monospace",
    theme: {
      background: '#1e1e1e',
      foreground: '#d4d4d4',
      cursor: '#d4d4d4',
      selectionBackground: '#264f78',
      black: '#000000',
      red: '#cd3131',
      green: '#0dbc79',
      yellow: '#e5e510',
      blue: '#2472c8',
      magenta: '#bc3fbc',
      cyan: '#11a8cd',
      white: '#e5e5e5',
      brightBlack: '#666666',
      brightRed: '#f14c4c',
      brightGreen: '#23d18b',
      brightYellow: '#f5f543',
      brightBlue: '#3b8eea',
      brightMagenta: '#d670d6',
      brightCyan: '#29b8db',
      brightWhite: '#e5e5e5',
    },
    allowTransparency: false,
    scrollback: 5000,
    allowProposedApi: true,
  })

  const fitAddon = new FitAddon()
  term.loadAddon(fitAddon)

  // 挂载到 DOM
  term.open(container)
  fitAddon.fit()

  // 连接 WebSocket
  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:'
  const url = proto + '//' + location.host + '/api/terminal/ws'
  let ws
  try {
    ws = new WebSocket(url)
  } catch (e) {
    term.write('\r\n\x1b[31m[错误] WebSocket 连接失败: ' + e.message + '\x1b[0m\r\n')
    tab.term = term
    tab.fitAddon = fitAddon
    return
  }

  ws.binaryType = 'arraybuffer'

  ws.onopen = () => {
    // 发送 init 消息
    const initMsg = JSON.stringify({
      type: 'init',
      shell: tab.shell,
      cwd: tab.cwd || undefined,
      cols: term.cols,
      rows: term.rows,
    })
    ws.send(initMsg)
  }

  ws.onmessage = (ev) => {
    if (ev.data instanceof ArrayBuffer) {
      // 二进制帧 = PTY 输出（VT 序列）
      term.write(new Uint8Array(ev.data))
    } else {
      // 文本帧 = JSON 控制消息
      try {
        const msg = JSON.parse(ev.data)
        if (msg.type === 'error') {
          term.write('\r\n\x1b[31m[错误] ' + (msg.msg || '未知错误') + '\x1b[0m\r\n')
        } else if (msg.type === 'closed') {
          term.write('\r\n\x1b[33m[终端会话已关闭]\x1b[0m\r\n')
        }
      } catch {}
    }
  }

  ws.onclose = () => {
    try { term.write('\r\n\x1b[33m[连接已断开]\x1b[0m\r\n') } catch {}
  }

  ws.onerror = (e) => {
    term.write('\r\n\x1b[31m[WebSocket 错误]\x1b[0m\r\n')
  }

  // 键盘输入 → WebSocket 二进制帧
  term.onData((data) => {
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(data)
    }
  })

  // 尺寸变化 → 发送 resize + fit
  const resizeObserver = new ResizeObserver(() => {
    try {
      fitAddon.fit()
      if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({
          type: 'resize',
          cols: term.cols,
          rows: term.rows,
        }))
      }
    } catch {}
  })
  if (container.parentElement) {
    resizeObserver.observe(container.parentElement)
  }

  // 保存引用
  tab.term = term
  tab.fitAddon = fitAddon
  tab.ws = ws
  tab._resizeObserver = resizeObserver

  // 聚焦
  term.focus()
}

// ─── 切换 Tab ──────────────────────────────────────────────

function switchTab(i) {
  if (i < 0 || i >= tabs.value.length || i === activeIdx.value) return
  activeIdx.value = i
  activeId.value = tabs.value[i].id
  nextTick(() => {
    const tab = tabs.value[activeIdx.value]
    if (tab && tab.fitAddon) {
      try { tab.fitAddon.fit() } catch {}
    }
    if (tab && tab.term) {
      tab.term.focus()
    }
  })
}

// ─── 关闭 Tab ──────────────────────────────────────────────

function closeTab(i) {
  if (tabs.value.length <= 1) return
  const tab = tabs.value[i]
  if (!tab) return

  // 关闭 WebSocket
  if (tab.ws) {
    try {
      tab.ws.send(JSON.stringify({ type: 'close' }))
      tab.ws.close()
    } catch {}
  }

  // 清理 ResizeObserver
  if (tab._resizeObserver) {
    try { tab._resizeObserver.disconnect() } catch {}
  }

  // 销毁 xterm
  if (tab.term) {
    try { tab.term.dispose() } catch {}
  }

  tabs.value.splice(i, 1)
  if (activeIdx.value >= tabs.value.length) {
    activeIdx.value = tabs.value.length - 1
  }
  if (activeIdx.value >= 0) {
    activeId.value = tabs.value[activeIdx.value].id
  } else {
    activeId.value = null
  }
}

// ─── 生命周期 ──────────────────────────────────────────────

onMounted(() => {
  // 默认创建一个终端
  createTerminal()
})

onUnmounted(() => {
  for (const tab of tabs.value) {
    if (tab.ws) {
      try {
        tab.ws.send(JSON.stringify({ type: 'close' }))
        tab.ws.close()
      } catch {}
    }
    if (tab._resizeObserver) {
      try { tab._resizeObserver.disconnect() } catch {}
    }
    if (tab.term) {
      try { tab.term.dispose() } catch {}
    }
  }
  tabs.value = []
})
</script>

<style scoped>
.terminal-panel {
  display: flex; flex-direction: column; height: 100%;
  background: #1e1e1e; color: #d4d4d4; font-size: 13px;
}

/* ── 终端标签栏 ── */
.term-tabs {
  display: flex; align-items: stretch; background: #2d2d2d;
  border-bottom: 1px solid #3c3c3c; flex-shrink: 0; overflow-x: auto;
}
.term-tab {
  display: flex; align-items: center; gap: 4px; padding: 4px 10px;
  background: none; border: none; border-right: 1px solid #3c3c3c;
  color: #999; font-size: 12px; cursor: pointer; white-space: nowrap;
}
.term-tab.active {
  background: #1e1e1e; color: #d4d4d4;
  border-bottom: 1px solid #1e1e1e; margin-bottom: -1px;
}
.term-tab:hover:not(.active) { background: #3c3c3c; }
.term-tab-close { font-size: 12px; margin-left: 2px; opacity: 0.5; color: #999; }
.term-tab-close:hover { opacity: 1; color: #e06c75; }
.term-tab.new-tab { padding: 4px 8px; }

/* ── 内容区 ── */
.term-container {
  flex: 1; display: flex; min-height: 0; position: relative;
}
.term-xterm {
  display: none; width: 100%; height: 100%;
}
.term-xterm.active {
  display: block;
}

/* 覆盖 xterm 默认背景以完全填充 */
.term-xterm :deep(.xterm) {
  height: 100%; padding: 0;
}
.term-xterm :deep(.xterm-viewport) {
  scrollbar-width: thin;
  scrollbar-color: #424242 #1e1e1e;
}

/* ── 空状态 ── */
.term-empty {
  flex: 1; display: flex; align-items: center; justify-content: center;
  background: #1e1e1e;
}
.term-create-btn {
  background: #007acc; color: #fff; border: none;
  padding: 6px 16px; border-radius: 4px; cursor: pointer; font-size: 13px;
}
.term-create-btn:hover { background: #0098ff; }

/* ── 新建终端对话框 ── */
.term-dialog-overlay {
  position: fixed; top: 0; left: 0; right: 0; bottom: 0;
  background: rgba(0,0,0,0.5); z-index: 1000;
  display: flex; align-items: center; justify-content: center;
}
.term-dialog {
  background: #252526; border: 1px solid #3c3c3c;
  border-radius: 8px; padding: 20px; min-width: 360px;
  color: #d4d4d4; box-shadow: 0 8px 32px rgba(0,0,0,0.5);
}
.term-dialog h3 {
  margin: 0 0 16px; font-size: 15px; font-weight: 600;
  color: #e0e0e0;
}
.dialog-row {
  display: flex; align-items: center; gap: 8px; margin-bottom: 12px;
}
.dialog-row label {
  font-size: 12px; color: #999; white-space: nowrap; min-width: 80px;
}
.dialog-row select, .dialog-input {
  flex: 1; background: #3c3c3c; border: 1px solid #555;
  color: #d4d4d4; padding: 4px 8px; border-radius: 3px;
  font-size: 12px; outline: none;
}
.dialog-row select:focus, .dialog-input:focus {
  border-color: #007acc;
}
.dialog-actions {
  display: flex; justify-content: flex-end; gap: 8px; margin-top: 16px;
}
.btn-cancel {
  background: #3c3c3c; color: #d4d4d4; border: 1px solid #555;
  padding: 4px 16px; border-radius: 3px; cursor: pointer; font-size: 12px;
}
.btn-confirm {
  background: #007acc; color: #fff; border: none;
  padding: 4px 16px; border-radius: 3px; cursor: pointer; font-size: 12px;
}
.btn-cancel:hover { background: #555; }
.btn-confirm:hover { background: #0098ff; }
</style>
