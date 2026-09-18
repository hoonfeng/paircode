<!--
  UpdateCard — 在线更新卡片（检查 → 下载 → 校验 → 安装重启）
  数据源：/api/update/*（内核接口，见 cmd/companion/update_api.go + internal/update）
  呈现：内嵌于「关于 PairCode」弹窗，横条 + 进度条 + 折叠的更新说明。
-->
<template>
  <div class="uc">
    <!-- 主行：状态 + 操作 -->
    <div class="uc-line">
      <SvgIcon :name="iconName" :size="14" :color="iconColor" />
      <span class="uc-title">软件更新</span>
      <span class="uc-cur">当前 {{ curVersion }}</span>
      <span v-if="hasUpdate" class="uc-new">→ 新版本 v{{ latest }}</span>
      <span class="uc-state" :class="{ 'uc-err': !!error }">{{ displayMessage }}</span>
      <span class="uc-gap"></span>

      <template v-if="confirming">
        <span class="uc-warn">确认安装 v{{ latest }} 并重启？</span>
        <button class="uc-btn uc-primary" @click="doApply(true)">确认安装并重启</button>
        <button class="uc-btn" @click="confirming = false">取消</button>
      </template>
      <template v-else>
        <button v-if="!isActive && stage !== 'ready'" class="uc-btn" :disabled="busy" @click="doCheck">
          <SvgIcon name="refresh" :size="12" /> {{ checkLabel }}
        </button>
        <button v-if="stage === 'available'" class="uc-btn uc-primary" @click="doDownload">下载并安装</button>
        <button v-if="isActive" class="uc-btn" @click="doCancel">取消</button>
        <template v-if="stage === 'ready'">
          <button class="uc-btn" @click="doDryRun">预览替换清单</button>
          <button class="uc-btn uc-primary" @click="confirming = true">立即安装并重启</button>
          <button class="uc-btn" @click="doApply(false)">仅安装，稍后重启</button>
        </template>
      </template>
    </div>

    <!-- 进度条 -->
    <div v-if="isActive" class="uc-prog">
      <div class="uc-bar"><div class="uc-fill" :style="{ width: percent.toFixed(1) + '%' }"></div></div>
      <span class="uc-pct">{{ percent.toFixed(1) }}% ｜ {{ mb(progress.downloaded) }} / {{ mb(progress.total) }} ｜ {{ speedMB }} MB/s</span>
    </div>

    <!-- 替换计划（预览） -->
    <div v-if="planInfo" class="uc-plan">{{ planInfo }}</div>

    <!-- 更新说明（折叠） -->
    <details v-if="hasUpdate && notes" class="uc-notes">
      <summary>更新说明（v{{ latest }}）</summary>
      <pre class="uc-pre">{{ notes }}</pre>
    </details>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import SvgIcon from './SvgIcon.vue'
import api from '../api.js'

const props = defineProps({
  currentVersion: { type: String, default: '' },
})

const stage = ref('idle')
const state = ref({})
const latest = ref('')
const hasUpdate = ref(false)
const notes = ref('')
const error = ref('')
const progress = ref({})
const percent = ref(0)
const busy = ref(false)
const confirming = ref(false)
const planInfo = ref('')
let timer = null

// 任务进行中的阶段（需轮询状态）
const ACTIVE = ['checking', 'downloading', 'verifying', 'extracting', 'applying', 'restarting']
const isActive = computed(() => ACTIVE.includes(stage.value))
// 版本号统一显示为 v 前缀形式（/api/system/info 回传带 v，/api/update/status 回传不带 v）
const curVersion = computed(() => {
  const v = String(props.currentVersion || '').trim()
  if (!v) return '—'
  return v[0] === 'v' || v[0] === 'V' ? v : 'v' + v
})
const checkLabel = computed(() => (stage.value === 'checking' ? '检查中…' : hasUpdate.value ? '重新检查' : '检查更新'))
const displayMessage = computed(() => error.value || state.value.message || '尚未检查更新')

const iconName = computed(() => {
  switch (stage.value) {
    case 'downloading': case 'verifying': case 'extracting': case 'applying': return 'package'
    case 'ready': return 'check-circle'
    case 'error': return 'shield-off'
    case 'available': return 'package'
    default: return 'refresh'
  }
})
const iconColor = computed(() => (stage.value === 'error' ? 'var(--danger, #e5534b)' : 'var(--accent)'))
const speedMB = computed(() => (((progress.value || {}).speedBps || 0) / 1048576).toFixed(2))

function mb(n) {
  const v = Number(n || 0)
  if (v >= 1073741824) return (v / 1073741824).toFixed(2) + ' GB'
  return (v / 1048576).toFixed(1) + ' MB'
}

function applyState(s) {
  if (!s) return
  state.value = s
  stage.value = s.stage || 'idle'
  latest.value = s.latest || ''
  hasUpdate.value = !!s.hasUpdate
  notes.value = s.notes || ''
  error.value = s.error || ''
  progress.value = s.progress || {}
  percent.value = Number(progress.value.percent || 0)
  if (isActive.value) ensurePoll()
  else stopPoll()
}

function ensurePoll() {
  if (timer) return
  timer = setInterval(async () => {
    try {
      const r = await api.apiGet('/update/status')
      applyState(r.state)
    } catch { /* 轮询失败不打断 UI */ }
  }, 900)
}
function stopPoll() {
  if (timer) {
    clearInterval(timer)
    timer = null
  }
}

async function doCheck() {
  busy.value = true
  planInfo.value = ''
  stage.value = 'checking'
  error.value = ''
  try {
    const r = await api.apiGet('/update/check')
    applyState(r.state)
    if (r.error) error.value = r.error
  } catch (e) {
    error.value = '检查更新失败：' + (e?.message || e)
    stage.value = 'error'
  } finally {
    busy.value = false
  }
}

async function doDownload() {
  planInfo.value = ''
  try {
    const r = await api.apiPost('/update/download')
    applyState(r.state)
    if (r.error) error.value = r.error
    ensurePoll()
  } catch (e) {
    error.value = '下载失败：' + (e?.message || e)
  }
}

async function doCancel() {
  try {
    const r = await api.apiPost('/update/cancel')
    applyState(r.state)
  } catch { /* 忽略 */ }
}

async function doDryRun() {
  planInfo.value = '正在生成替换计划…'
  try {
    const r = await api.apiPost('/update/apply', { dryRun: true, restart: false })
    const p = (r.result || {}).plan || {}
    planInfo.value = `替换预览：将写 ${(p.write || []).length} 个文件（${mb(p.bytes)}），保护名单跳过 ${(p.skip || []).length} 个（config/ 与 .pair 用户数据），主程序 ${p.mainProgram}`
    if (r.error) planInfo.value = '预览失败：' + r.error
  } catch (e) {
    planInfo.value = '预览失败：' + (e?.message || e)
  }
}

async function doApply(restart) {
  confirming.value = false
  planInfo.value = restart ? '正在安装并重启，页面将在数秒后断开…' : '正在安装（不重启）…'
  try {
    const r = await api.apiPost('/update/apply', { dryRun: false, restart })
    applyState(r.state)
    if (r.error) {
      planInfo.value = '安装失败：' + r.error
      return
    }
    const p = (r.result || {}).plan || {}
    planInfo.value = restart
      ? `已安装 v${latest.value}，正在重启（备份：${p.backupPath || '-'}）`
      : `已安装 v${latest.value}（${(p.write || []).length} 个文件），重启后生效；旧版本备份：${p.backupPath || '-'}`
  } catch (e) {
    planInfo.value = '安装失败：' + (e?.message || e)
  }
}

onMounted(async () => {
  try {
    const r = await api.apiGet('/update/status')
    applyState(r.state)
  } catch { /* 首次未初始化不报错 */ }
})

onUnmounted(stopPoll)
</script>

<style scoped>
.uc {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 10px 12px;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 6px;
}
.uc-line {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  font-size: 12px;
}
.uc-title { font-weight: 600; color: var(--text-primary); }
.uc-cur { color: var(--text-secondary); }
.uc-new { color: var(--accent); font-weight: 600; }
.uc-state { color: var(--text-secondary); }
.uc-err { color: var(--danger, #e5534b); }
.uc-warn { color: var(--danger, #e5534b); font-weight: 600; }
.uc-gap { flex: 1; }
.uc-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  font-size: 12px;
  color: var(--text-primary);
  background: var(--bg-hover);
  border: 1px solid var(--border-color);
  border-radius: 4px;
  cursor: pointer;
}
.uc-btn:hover:not(:disabled) { border-color: var(--accent); }
.uc-btn:disabled { opacity: 0.55; cursor: default; }
.uc-primary {
  background: var(--accent);
  border-color: var(--accent);
  color: #fff;
}
.uc-prog { display: flex; align-items: center; gap: 10px; }
.uc-bar {
  flex: 1;
  height: 6px;
  background: var(--bg-hover);
  border-radius: 3px;
  overflow: hidden;
}
.uc-fill {
  height: 100%;
  background: var(--accent);
  transition: width 0.3s ease;
}
.uc-pct { font-size: 11px; color: var(--text-secondary); white-space: nowrap; }
.uc-plan { font-size: 11px; color: var(--text-secondary); line-height: 1.5; }
.uc-notes { font-size: 11px; color: var(--text-secondary); }
.uc-notes summary { cursor: pointer; color: var(--accent); }
.uc-pre {
  max-height: 160px;
  overflow: auto;
  margin: 6px 0 0;
  padding: 8px;
  background: var(--bg-primary, #1b1b1f);
  border: 1px solid var(--border-color);
  border-radius: 4px;
  font-size: 11px;
  line-height: 1.5;
  white-space: pre-wrap;
  word-break: break-word;
}
</style>
