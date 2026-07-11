<template>
  <div class="terminal-panel">
    <!-- 终端标签栏 -->
    <div class="term-tabs">
      <button v-for="(term, i) in terminals" :key="i"
              :class="['term-tab', { active: i === activeTermIdx }]"
              @click="activeTermIdx = i"
              @mouseup.middle="closeTerm(i)">
        <SvgIcon name="terminal" :size="12" />
        <span class="term-tab-label">{{ term.label }}</span>
        <span class="term-tab-close" @click.stop="closeTerm(i)" title="关闭">×</span>
      </button>
      <button class="term-tab new-tab" @click="newTerminal" title="新建终端">
        <SvgIcon name="plus" :size="12" />
      </button>
    </div>

    <!-- 当前终端内容 -->
    <div v-if="terminals.length > 0" class="term-content">
      <div class="term-output" ref="termRef">
        <div v-for="(line, j) in currentTerm.output" :key="j"
             :class="['term-line', line.type === 'stderr' ? 'term-err' : '']">
          <span class="term-text">{{ line.text }}</span>
        </div>
        <div v-if="currentTerm.executing" class="term-line term-executing">
          <span class="term-cursor">_</span>
        </div>
      </div>
      <div class="term-input-line">
        <span class="term-prompt-inline">{{ currentTerm.cwd }}&gt;</span>
        <input ref="inputRef"
               v-model="currentTerm.input"
               class="term-input"
               @keydown="onKeydown"
               placeholder="输入命令..."
               :disabled="currentTerm.executing" />
        <button class="term-btn-run" @click="runCommand" :disabled="!currentTerm.input.trim() || currentTerm.executing">
          <SvgIcon name="send" :size="12" />
        </button>
      </div>
    </div>

    <!-- 无终端时 -->
    <div v-else class="term-empty">
      <button class="term-create-btn" @click="newTerminal">新建终端</button>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, nextTick } from 'vue'
import api from '../api.js'
import SvgIcon from './SvgIcon.vue'

// ── 多终端实例 ──
const terminals = ref([])
const activeTermIdx = ref(-1)
const termRef = ref(null)
const inputRef = ref(null)
let termCounter = 0

const currentTerm = computed(() => {
  if (activeTermIdx.value < 0 || activeTermIdx.value >= terminals.value.length) {
    return { output: [], input: '', executing: false, cwd: 'C:\\', label: '终端', history: [], histIdx: -1 }
  }
  return terminals.value[activeTermIdx.value]
})

function createTerminal(cwd) {
  termCounter++
  return {
    label: '终端 ' + termCounter,
    cwd: cwd || 'C:\\',
    input: '',
    output: [],
    executing: false,
    history: [],
    histIdx: -1,
  }
}

function newTerminal() {
  const term = createTerminal(currentTerm.value?.cwd)
  terminals.value.push(term)
  activeTermIdx.value = terminals.value.length - 1
  term.output.push({ type: 'stdout', text: 'PairCode 终端 (Windows) — ' + term.label })
  term.output.push({ type: 'stdout', text: '' })
  saveTerminals()
  nextTick(() => scrollToBottom())
}

function closeTerm(i) {
  if (terminals.value.length <= 1) return
  terminals.value.splice(i, 1)
  if (activeTermIdx.value >= terminals.value.length) {
    activeTermIdx.value = terminals.value.length - 1
  }
  saveTerminals()
}

// ── 命令执行 ──
async function runCommand() {
  const term = terminals.value[activeTermIdx.value]
  if (!term || !term.input.trim() || term.executing) return
  const cmd = term.input.trim()
  term.output.push({ type: 'stdout', text: term.cwd + '> ' + cmd })
  term.history.unshift(cmd)
  term.histIdx = -1
  term.input = ''
  term.executing = true
  try {
    const res = await api.apiPost('/system/exec', { command: cmd, cwd: term.cwd })
    if (res.stdout) {
      for (const line of res.stdout.split('\n').filter(Boolean)) {
        term.output.push({ type: 'stdout', text: line })
      }
    }
    if (res.stderr) {
      for (const line of res.stderr.split('\n').filter(Boolean)) {
        term.output.push({ type: 'stderr', text: line })
      }
    }
    if (res.cwd) { term.cwd = res.cwd; saveTerminals() }
  } catch (err) {
    term.output.push({ type: 'stderr', text: '错误: ' + err.message })
  }
  term.executing = false
  scrollToBottom()
}

const onKeydown = (e) => {
  const term = terminals.value[activeTermIdx.value]
  if (!term) return
  if (e.key === 'Enter') { e.preventDefault(); runCommand(); return }
  if (e.key === 'ArrowUp') {
    e.preventDefault()
    if (term.history.length > 0) {
      term.histIdx = Math.min(term.histIdx + 1, term.history.length - 1)
      term.input = term.history[term.histIdx]
    }
    return
  }
  if (e.key === 'ArrowDown') {
    e.preventDefault()
    if (term.histIdx > 0) {
      term.histIdx--
      term.input = term.history[term.histIdx]
    } else {
      term.histIdx = -1
      term.input = ''
    }
    return
  }
}

const scrollToBottom = () => {
  nextTick(() => {
    if (termRef.value) termRef.value.scrollTop = termRef.value.scrollHeight
  })
}

// 保存/恢复终端列表
const TERM_KEY = 'paircode-terminals'
function saveTerminals() {
  try {
    const data = terminals.value.map(t => ({
      label: t.label,
      cwd: t.cwd,
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
    let restored = false
    for (const t of data) {
      termCounter++
      terminals.value.push({
        label: t.label || '终端 ' + termCounter,
        cwd: t.cwd || 'C:\\',
        input: '',
        output: [{ type: 'stdout', text: 'PairCode 终端 (Windows) — ' + (t.label || '终端 ' + termCounter) }],
        executing: false,
        history: [],
        histIdx: -1,
      })
    }
    if (terminals.value.length > 0) {
      activeTermIdx.value = 0
      restored = true
    }
    return restored
  } catch { return false }
}

// 监听终端切换目录事件
onMounted(() => {
  // 尝试恢复终端
  if (!loadTerminals()) {
    // 默认创建一个终端
    newTerminal()
  }
  api.apiGet('/system/info').then(info => {
    if (info.cwd && terminals.value[0]) {
      terminals.value[0].cwd = info.cwd
    }
  }).catch(e => console.warn('[终端] 获取系统信息失败:', e))
  // 监听终端路径切换
  window.addEventListener('terminal-cwd', (e) => {
    const cwd = e.detail?.cwd
    if (cwd) {
      const term = terminals.value[activeTermIdx.value]
      if (term) term.cwd = cwd
      saveTerminals()
    }
  })
})
</script>

<style scoped>
.terminal-panel {
  display: flex; flex-direction: column; height: 100%;
  background: var(--bg-primary); color: var(--text-primary); font-size: 13px;
}
/* ── 终端标签栏 ── */
.term-tabs {
  display: flex; align-items: stretch; background: var(--bg-tertiary);
  border-bottom: 1px solid var(--border-color); flex-shrink: 0; overflow-x: auto;
}
.term-tab {
  display: flex; align-items: center; gap: 4px; padding: 4px 10px;
  background: none; border: none; border-right: 1px solid var(--border-color);
  color: var(--text-secondary); font-size: 12px; cursor: pointer; white-space: nowrap;
}
.term-tab.active {
  background: var(--bg-primary); color: var(--text-primary);
  border-bottom: 1px solid var(--bg-primary); margin-bottom: -1px;
}
.term-tab:hover:not(.active) { background: var(--bg-hover); }
.term-tab-close { font-size: 12px; margin-left: 2px; opacity: 0.5; }
.term-tab-close:hover { opacity: 1; color: #c03; }
.term-tab.new-tab { padding: 4px 8px; }

/* ── 内容区 ── */
.term-content {
  flex: 1; display: flex; flex-direction: column; min-height: 0;
}
.term-output {
  flex: 1; overflow-y: auto; padding: 4px 8px;
  font-family: 'Consolas', 'Cascadia Code', monospace; font-size: 12px; line-height: 1.5;
}
.term-line { white-space: pre-wrap; word-break: break-all; margin: 1px 0; }
.term-text { color: var(--term-text); }
.term-err .term-text { color: #f48771; }
.term-executing { color: var(--term-prompt); }
.term-cursor { animation: blink 1s infinite; color: var(--term-text); }
@keyframes blink { 0%, 100% { opacity: 1; } 50% { opacity: 0; } }

.term-input-line {
  display: flex; align-items: center; gap: 4px; padding: 4px 8px;
  border-top: 1px solid var(--border-color); flex-shrink: 0;
  background: var(--bg-tertiary);
}
.term-prompt-inline { color: var(--term-prompt); font-family: var(--font-code); font-size: 12px; white-space: nowrap; flex-shrink: 0; margin-right: 4px; }
.term-input {
  flex: 1; background: transparent; border: none; color: var(--term-text);
  font-family: var(--font-code); font-size: 12px; outline: none; padding: 2px 0;
}
.term-input::placeholder { color: var(--text-muted); }
.term-btn-run {
  background: var(--bg-hover); border: 1px solid var(--border-color);
  color: var(--text-secondary); padding: 2px 8px; border-radius: 3px; cursor: pointer;
  display: flex; align-items: center;
}
.term-btn-run:hover { background: var(--accent); color: #000; }
.term-btn-run:disabled { opacity: 0.4; cursor: default; }

.term-empty {
  flex: 1; display: flex; align-items: center; justify-content: center;
}
.term-create-btn {
  background: var(--accent); color: #000; border: none;
  padding: 6px 16px; border-radius: 4px; cursor: pointer; font-size: 13px;
}
</style>
