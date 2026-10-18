<template>
  <div class="fx-root">
    <header class="fx-head">
      <h1>wb-ui 引擎 fixture · 按 IDE 实现复刻</h1>
      <div class="fx-status">插件列表 / 输入框 / 市场 · 引擎缺陷隔离验证</div>
    </header>

    <div class="fx-grid">
      <!-- ═══ A. 插件列表（复刻 PluginPanel）═══ -->
      <section class="fx-col">
        <h2>A · 插件列表（滚动 / 名称 / 徽章）</h2>
        <div class="pp-panel">
          <div class="pp-head">
            <span class="pp-title">插件（{{ plugins.length }}）</span>
            <span class="pp-sub">列表随滚动移动</span>
          </div>
          <div class="pp-list" id="pp-list" @scroll="recordListScroll">
            <div v-for="(p, i) in plugins" :key="p.name" class="pp-item" :id="'pp-item-' + i">
              <div class="pp-item-row">
                <div class="pp-row-main">
                  <span class="pp-state" :class="p.state"></span>
                  <span class="pp-name">{{ p.name }}</span>
                  <span class="pp-chevron">›</span>
                </div>
                <div class="pp-row-meta">
                  <span class="pp-src" :class="p.source">{{ p.srcLabel }}</span>
                  <span v-if="p.scope === 'global'" class="pp-badge">全局</span>
                  <span v-if="p.hasClient" class="pp-badge">UI</span>
                  <span v-if="p.tools" class="pp-count">{{ p.tools }} 工具</span>
                </div>
                <div v-if="p.hasClient" class="pp-row-ui">
                  <span class="pp-ui-label" :class="{ on: p.uiActive }">{{ p.uiActive ? 'UI 已启用' : 'UI 未启用' }}</span>
                  <label class="pp-switch">
                    <input type="checkbox" :checked="p.uiActive" @click.stop />
                    <span class="pp-switch-track"></span>
                  </label>
                </div>
              </div>
            </div>
          </div>
          <div class="pp-foot">
            <span>scrollTop：<b id="pp-scroll-value">{{ ppScrollTop }}</b>px</span>
            <button class="fx-btn" @click="scrollList(120)">滚 +120</button>
            <button class="fx-btn" @click="scrollList(0)">回顶</button>
          </div>
        </div>
      </section>

      <!-- ═══ B. 输入框（复刻 RightPanel 输入区）═══ -->
      <section class="fx-col">
        <h2>B · 输入框（contenteditable / placeholder / 光标）</h2>
        <div class="chat-input-area" id="chat-input-area">
          <div class="input-wrapper">
            <div class="input-row1">
              <span class="ir-pill ir-pill-model">claude-sonnet</span>
              <span class="ir-pill ir-pill-toolset">场景：基础</span>
            </div>
            <div class="chat-input" id="chat-input" ref="chatInputEl"
                 contenteditable="true"
                 data-placeholder="描述你要做的事，Agent 会自己选工具、跑验证…"
                 style="height: 80px"
                 @input="onInput"
                 @focus="onFocus"
                 @blur="onBlur"></div>
            <div class="input-bottom-bar">
              <div class="ibb-btns">
                <span class="obtn">▣</span>
                <span class="obtn">≡</span>
                <span class="obtn obtn-agent">⟳</span>
              </div>
              <span class="ibb-enter-hint">Enter 发送 · Shift+Enter 换行</span>
              <button class="send-btn" @click="noop">发送</button>
            </div>
          </div>
        </div>
        <div class="pp-foot">
          <span>focused：<b id="ci-focused">{{ ciFocused }}</b></span>
          <span>行高：<b id="ci-lh">{{ ciLh }}</b></span>
          <span>高度：<b id="ci-h">{{ ciH }}</b></span>
        </div>
      </section>

      <!-- ═══ C. 市场（复刻 MarketplacePanel 卡片列表）═══ -->
      <section class="fx-col">
        <h2>C · 市场卡片（层叠 / 滚动）</h2>
        <div class="market-body" id="market-body">
          <div class="market-search">
            <input class="search-input" placeholder="搜索 MCP / 技能 / npm 插件…" />
            <button class="fx-btn">刷新</button>
          </div>
          <div class="market-list" id="market-list" @scroll="recordMarketScroll">
            <div v-for="c in cards" :key="c.id" class="market-item">
              <div class="mi-head">
                <span class="mi-name">{{ c.name }}</span>
                <span class="mi-badge">{{ c.kind }}</span>
              </div>
              <div class="mi-desc">{{ c.desc }}</div>
              <div class="mi-foot">
                <span class="mi-src">{{ c.src }}</span>
                <button class="mi-btn">安装</button>
              </div>
            </div>
          </div>
          <div class="pp-foot">
            <span>scrollTop：<b id="market-scroll-value">{{ marketScrollTop }}</b>px</span>
            <button class="fx-btn" @click="scrollMarket(120)">滚 +120</button>
            <button class="fx-btn" @click="scrollMarket(0)">回顶</button>
          </div>
        </div>
      </section>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onBeforeUnmount } from 'vue'

// ── 数据 ──
const plugins = ref([
  { name: 'tool-flow', state: 'on', source: 'repo', srcLabel: 'repo', scope: 'global', hasClient: false, tools: 6, uiActive: false },
  { name: 'tool-harness', state: 'on', source: 'repo', srcLabel: 'repo', scope: 'global', hasClient: false, tools: 8, uiActive: false },
  { name: 'cordis-plugin-loader-with-a-very-long-name-that-should-ellipsis', state: 'on', source: 'js', srcLabel: '动态', scope: 'project', hasClient: true, tools: 3, uiActive: true },
  { name: 'ui-app', state: 'on', source: 'js', srcLabel: '动态', scope: 'global', hasClient: true, tools: 2, uiActive: true },
  { name: 'agentloop', state: 'off', source: 'repo', srcLabel: 'repo', scope: 'global', hasClient: false, tools: 5, uiActive: false },
  { name: 'tool-art', state: 'on', source: 'repo', srcLabel: 'repo', scope: 'project', hasClient: false, tools: 12, uiActive: false },
  { name: 'tool-model', state: 'on', source: 'repo', srcLabel: 'repo', scope: 'project', hasClient: true, tools: 9, uiActive: true },
  { name: '@paircode/plugin-node-bridge-long-package-name', state: 'on', source: 'node-bridge', srcLabel: 'npm', scope: 'global', hasClient: false, tools: 4, uiActive: false },
  { name: 'tool-music', state: 'off', source: 'repo', srcLabel: 'repo', scope: 'project', hasClient: false, tools: 7, uiActive: false },
  { name: 'tool-rig', state: 'on', source: 'repo', srcLabel: 'repo', scope: 'project', hasClient: false, tools: 5, uiActive: false },
  { name: 'tool-voice', state: 'on', source: 'repo', srcLabel: 'repo', scope: 'global', hasClient: false, tools: 3, uiActive: false },
  { name: 'plugin-dev-helpers-and-templates', state: 'on', source: 'js', srcLabel: '动态', scope: 'project', hasClient: true, tools: 2, uiActive: false },
])

const cards = ref([
  { id: 1, name: '@modelcontextprotocol/server-filesystem', kind: 'MCP', desc: '本地文件系统访问：读写、检索、目录遍历', src: 'npm · 12.4k 下载' },
  { id: 2, name: 'paircode-skill-web-search', kind: '技能', desc: '联网检索并整理来源，自动去重', src: 'builtin' },
  { id: 3, name: 'cordis-ui-sidebar-extra-tools-with-long-name', kind: '插件', desc: 'UI 插件：在侧栏叠加实用工具入口', src: 'npm · 3.1k 下载' },
  { id: 4, name: 'mcp-server-sqlite', kind: 'MCP', desc: 'SQLite 数据库查询与模式浏览', src: 'npm · 8.7k 下载' },
  { id: 5, name: 'paircode-skill-image-optimizer', kind: '技能', desc: '图片压缩与格式转换（PNG/WebP/JPEG）', src: 'builtin' },
  { id: 6, name: 'cordis-plugin-log-viewer', kind: '插件', desc: '在面板中实时查看后端日志', src: 'npm · 1.9k 下载' },
  { id: 7, name: 'mcp-server-github', kind: 'MCP', desc: 'GitHub 仓库、Issue、PR 查询', src: 'npm · 15.2k 下载' },
  { id: 8, name: 'paircode-skill-doc-gen', kind: '技能', desc: '从源码注释生成 API 文档', src: 'builtin' },
])

// ── 输入框诊断 ──
const chatInputEl = ref(null)
const ciFocused = ref('no')
const ciLh = ref('-')
const ciH = ref('-')

function onFocus() { ciFocused.value = 'yes'; measure() }
function onBlur() { ciFocused.value = 'no' }
function onInput() { measure() }
function measure() {
  const el = chatInputEl.value
  if (!el) return
  const cs = getComputedStyle(el)
  ciLh.value = cs.lineHeight
  ciH.value = Math.round(el.getBoundingClientRect().height) + 'px'
}

// ── 滚动诊断 ──
const ppScrollTop = ref(0)
const marketScrollTop = ref(0)
const scrollLog = []

function recordListScroll(e) {
  ppScrollTop.value = Math.round(e.target.scrollTop)
  scrollLog.push('list@' + Math.round(e.target.scrollTop))
  if (scrollLog.length > 6) scrollLog.shift()
}
function recordMarketScroll(e) {
  marketScrollTop.value = Math.round(e.target.scrollTop)
}
function scrollList(y) {
  const el = document.getElementById('pp-list')
  if (el) el.scrollTop = y
}
function scrollMarket(y) {
  const el = document.getElementById('market-list')
  if (el) el.scrollTop = y
}

// ── rAF 轮询 scrollTop（若引擎把 scrollTop 复位，能观测到回 0）──
let rafId = 0
let lastPp = 0
function tick() {
  const el = document.getElementById('pp-list')
  if (el) {
    const v = Math.round(el.scrollTop)
    if (v !== lastPp) { lastPp = v; ppScrollTop.value = v }
  }
  const m = document.getElementById('market-list')
  if (m) marketScrollTop.value = Math.round(m.scrollTop)
  rafId = requestAnimationFrame(tick)
}

onMounted(() => {
  rafId = requestAnimationFrame(tick)
  // 自动聚焦输入框（验证引擎 focus 后是否绘制 caret）
  setTimeout(() => {
    const el = chatInputEl.value
    if (el) { el.focus(); measure() }
  }, 300)
  window.__fx = { plugins, cards, ciFocused, ciLh, ciH, ppScrollTop, marketScrollTop, scrollLog }
})
onBeforeUnmount(() => cancelAnimationFrame(rafId))

function noop() {}
</script>

<style>
/* ═══════════════════════════════════════════════════════════
   主题变量：按 IDE 暗色主题复刻（opaque 背景是层叠缺陷复现的关键）
   ═══════════════════════════════════════════════════════════ */
:root {
  --bg-primary: #141416;
  --bg-secondary: #1b1b1e;
  --bg-tertiary: #232327;
  --color-surface-2: #1f1f23;
  --color-surface-3: #2a2a2f;
  --color-input-border: #3a3a40;
  --text-primary: #e8e8ea;
  --text-muted: #8b8b93;
  --text-secondary: #b4b4bc;
  --border-color: #2e2e34;
  --accent: #4a8cff;
  --focus-ring: rgba(74, 140, 255, 0.25);
  --shadow-sm: 0 1px 2px rgba(0,0,0,.3);
  --fs-sm: 12px;
  --mono-font: ui-monospace, monospace;
  color-scheme: dark;
}
* { box-sizing: border-box; }
html, body, #app { margin: 0; padding: 0; }
body { background: var(--bg-primary); color: var(--text-primary); font-family: system-ui, sans-serif; font-size: 13px; }

.fx-root { padding: 12px; }
.fx-head { display: flex; align-items: baseline; gap: 12px; margin-bottom: 10px; }
.fx-head h1 { font-size: 15px; margin: 0; }
.fx-status { font-size: 11px; color: var(--text-muted); }
.fx-grid { display: flex; gap: 12px; align-items: flex-start; }
.fx-col { flex: 1 1 0; min-width: 0; background: var(--bg-secondary); border: 1px solid var(--border-color); border-radius: 8px; padding: 10px; }
.fx-col h2 { font-size: 12px; margin: 0 0 8px 0; color: var(--text-secondary); font-weight: 600; }
.fx-btn { font-size: 11px; padding: 2px 8px; border-radius: 4px; border: 1px solid var(--border-color); background: var(--bg-tertiary); color: var(--text-secondary); cursor: pointer; }

/* ═══ A. 插件列表（复刻 PluginPanel）═══ */
.pp-panel { display: flex; flex-direction: column; height: 420px; background: var(--bg-secondary); border: 1px solid var(--border-color); border-radius: 8px; overflow: hidden; }
.pp-head { display: flex; align-items: baseline; gap: 8px; padding: 8px 10px; border-bottom: 1px solid var(--border-color); flex-shrink: 0; }
.pp-title { font-size: 13px; font-weight: 600; }
.pp-sub { font-size: 11px; color: var(--text-muted); }
.pp-list { flex: 1; overflow-y: auto; min-height: 0; }
.pp-item { border-bottom: 1px solid var(--border-color); }
.pp-item-row { padding: 8px 10px; }
/* ★ 复刻 2026-10-18 GUI 兼容改造：名称/徽章块级两行，不依赖 flex-wrap */
.pp-row-main { display: flex; align-items: center; gap: 6px; }
.pp-state { width: 7px; height: 7px; border-radius: 50%; background: #666; flex-shrink: 0; }
.pp-state.on { background: #4caf50; }
.pp-name { font-size: 13px; color: var(--text-primary); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; flex: 1 1 0; min-width: 96px; }
.pp-chevron { color: var(--text-muted); font-size: 12px; flex-shrink: 0; }
.pp-row-meta { display: flex; align-items: center; gap: 6px; margin-top: 4px; }
.pp-src { font-size: 10px; color: var(--text-muted); border: 1px solid var(--border-color); border-radius: 4px; padding: 0 4px; }
.pp-src.js { color: #4a8cff; border-color: rgba(74,140,255,.35); }
.pp-badge { font-size: 10px; color: #b4b4bc; background: var(--color-surface-3); border-radius: 4px; padding: 0 5px; }
.pp-count { font-size: 10px; color: var(--text-muted); }
.pp-row-ui { display: flex; align-items: center; gap: 8px; margin-top: 4px; }
.pp-ui-label { font-size: 11px; color: var(--text-muted); }
.pp-ui-label.on { color: #4caf50; }
.pp-switch { position: relative; display: inline-flex; width: 28px; height: 16px; }
.pp-switch input { opacity: 0; width: 0; height: 0; }
.pp-switch-track { position: absolute; inset: 0; border-radius: 8px; background: var(--color-surface-3); }
.pp-switch-track::after { content: ''; position: absolute; top: 2px; left: 2px; width: 12px; height: 12px; border-radius: 50%; background: #aaa; }
.pp-foot { display: flex; align-items: center; gap: 8px; padding: 6px 10px; border-top: 1px solid var(--border-color); font-size: 11px; color: var(--text-muted); flex-shrink: 0; }
.pp-foot b { color: var(--accent); }

/* ═══ B. 输入框（复刻 RightPanel 输入区）═══ */
.chat-input-area { display: flex; flex-direction: column; flex-shrink: 0; padding: 0 8px 8px 8px; background: var(--bg-secondary); border: 1px solid var(--border-color); border-radius: 8px; }
.input-wrapper { position: relative; display: flex; flex-direction: column; background: var(--color-surface-2); border: 1px solid var(--color-input-border); border-radius: 12px; padding: 8px; gap: 8px; width: 100%; box-sizing: border-box; box-shadow: var(--shadow-sm); }
.input-row1 { display: flex; align-items: center; gap: 6px; }
.ir-pill { font-size: 10px; padding: 1px 8px; border-radius: 10px; }
.ir-pill-model { background: var(--color-surface-3); color: var(--accent); }
.ir-pill-toolset { background: var(--color-surface-2); color: var(--text-muted); border: 1px solid var(--border-color); }
/* ★ 复刻真实 chat-input 全部关键属性：line-height:1.6 / min-height:80px / overflow-y:auto / pre-wrap */
.chat-input { display: block; width: 100%; background: transparent; border: none; color: var(--text-primary); padding: 16px 18px 8px 18px; border-radius: 0; font-size: 14px; resize: none; outline: none; min-height: 80px; font-family: inherit; line-height: 1.6; box-sizing: border-box; overflow-y: auto; white-space: pre-wrap; word-break: break-word; cursor: text; }
.chat-input:empty::before { content: attr(data-placeholder); color: var(--text-muted); pointer-events: none; }
.input-bottom-bar { display: flex; align-items: center; justify-content: space-between; gap: 6px; padding: 0 10px 10px 10px; }
.ibb-btns { display: flex; align-items: center; gap: 4px; }
.obtn { display: inline-flex; align-items: center; justify-content: center; width: 32px; height: 32px; border-radius: 8px; background: var(--color-surface-3); color: var(--text-secondary); font-size: 13px; }
.obtn-agent { color: #e0a030; }
.ibb-enter-hint { font-size: 11px; color: var(--text-muted); }
.send-btn { height: 32px; padding: 0 18px; border-radius: 8px; border: none; background: var(--accent); color: #fff; font-size: 13px; }

/* ═══ C. 市场（复刻 MarketplacePanel）═══ */
.market-body { display: flex; flex-direction: column; height: 420px; background: var(--bg-secondary); border: 1px solid var(--border-color); border-radius: 8px; overflow: hidden; }
.market-search { display: flex; align-items: center; gap: 6px; padding: 8px 10px; border-bottom: 1px solid var(--border-color); flex-shrink: 0; }
.search-input { flex: 1; height: 26px; border-radius: 6px; border: 1px solid var(--color-input-border); background: var(--bg-tertiary); color: var(--text-primary); font-size: 12px; padding: 0 8px; }
.market-list { flex: 1; overflow-y: auto; min-height: 0; padding: 8px; }
.market-item { display: flex; flex-direction: column; gap: 4px; padding: 8px 10px; border: 1px solid var(--border-color); border-radius: 8px; background: var(--bg-tertiary); margin-bottom: 6px; }
.mi-head { display: flex; align-items: center; gap: 6px; }
.mi-name { font-size: 13px; color: var(--text-primary); font-weight: 500; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; flex: 1 1 0; }
.mi-badge { font-size: 10px; color: #b4b4bc; background: var(--color-surface-3); border-radius: 4px; padding: 0 5px; flex-shrink: 0; }
.mi-desc { font-size: 11px; color: var(--text-muted); line-height: 1.5; }
.mi-foot { display: flex; align-items: center; justify-content: space-between; }
.mi-src { font-size: 10px; color: var(--text-muted); }
.mi-btn { height: 22px; padding: 0 10px; border-radius: 6px; border: 1px solid var(--border-color); background: var(--color-surface-3); color: var(--text-secondary); font-size: 11px; }
</style>
