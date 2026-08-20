<template>
  <div class="pm-manager">
    <!-- 工具栏 -->
    <div class="mgm-toolbar">
      <span class="mgm-count">{{ names.length }} 个预设</span>
      <button class="mgm-btn mgm-primary" @click="saveCurrent">＋ 保存当前配置为预设</button>
    </div>

    <!-- 新增命名（保存当前配置 → 命名 → 入库） -->
    <div v-if="naming" class="mgm-edit">
      <div class="mgm-edit-title">保存当前配置为预设</div>
      <div class="mgm-field">
        <span class="mgm-field-label">预设名称（如「工作主力」「写作备用」，对话面板按此切换）</span>
        <input v-model="newName" placeholder="输入预设名称…" @keydown.enter="confirmSaveCurrent" />
      </div>
      <div class="mgm-field">
        <span class="mgm-field-label">将保存的配置（当前 AI 设置快照）</span>
        <div class="pm-snapshot">
          <div class="pm-snap-row"><span>服务商</span><b>{{ snap.provider || '—' }}</b></div>
          <div class="pm-snap-row"><span>执行模型</span><b>{{ snap.executeModel || '—' }}</b></div>
          <div class="pm-snap-row"><span>规划模型</span><b>{{ snap.planModel || '—' }}</b></div>
          <div class="pm-snap-row"><span>审核模型</span><b>{{ snap.reviewModel || '—' }}</b></div>
          <div class="pm-snap-row"><span>Base URL</span><b class="pm-snap-mono">{{ snap.baseURL || '—' }}</b></div>
          <div class="pm-snap-row"><span>API Key</span><b>{{ snap.apiKey ? (snap.apiKey.slice(0, 12) + '…') : '—' }}</b></div>
          <div class="pm-snap-row"><span>温度 / 思考 / 输出上限</span><b>{{ snap.temperature || '默认' }} / {{ snap.thinkingMode || '默认' }} / {{ snap.maxTokens || '默认' }}</b></div>
        </div>
      </div>
      <div class="mgm-edit-actions">
        <button class="mgm-btn mgm-primary" :disabled="saving" @click="confirmSaveCurrent">
          {{ saving ? '保存中…' : '保存预设' }}
        </button>
        <button class="mgm-btn" @click="naming = false">取消</button>
      </div>
    </div>

    <!-- 预设卡片列表 -->
    <div v-if="names.length" class="mgm-cards">
      <div v-for="n in names" :key="n" class="mgm-card" :class="{ 'pm-active': n === activePreset }">
        <div class="mgm-card-head">
          <span class="mgm-name" :title="n">{{ n }}<span v-if="n === activePreset" class="pm-active-badge">使用中</span></span>
          <div class="mgm-ops">
            <button class="mgm-btn mgm-small" :disabled="applying === n" @click="applyPreset(n)">
              {{ applying === n ? '应用中…' : '应用' }}
            </button>
            <button class="mgm-btn mgm-small" @click="startRename(n)">改名</button>
            <button class="mgm-btn mgm-small mgm-danger" @click="removePreset(n)">删除</button>
          </div>
        </div>
        <div class="pm-preview">
          <div class="pm-snap-row"><span>服务商</span><b>{{ (presets[n] || {}).provider || '—' }}</b></div>
          <div class="pm-snap-row"><span>执行模型</span><b>{{ (presets[n] || {}).executeModel || '—' }}</b></div>
          <div class="pm-snap-row"><span>规划 / 审核</span><b>{{ (presets[n] || {}).planModel || '—' }} / {{ (presets[n] || {}).reviewModel || '—' }}</b></div>
        </div>
      </div>
    </div>
    <div v-else-if="!naming" class="mgm-empty">暂无 AI 配置预设。先在上方配置好 AI（服务商 / 模型 / 参数），点「保存当前配置为预设」命名入库；对话面板即可从预设列表快速切换。</div>

    <div v-if="error" class="mgm-error">{{ error }}</div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import api from '../api.js'

// AI 配置预设管理面板：维护 config/ai-presets.json（GET /api/ai-presets，POST save/apply/delete/rename）。
// 预设 = 一份完整 AI 配置快照（服务商 + BaseURL + Key + 模型 + 参数）。
// 保存：抓当前 settings 快照 → 命名 → 入库；应用：预设整套写回 settings（对话面板同源切换）。
// ★ 服务商配置（models.json）不被预设改动——预设只存引用（provider 名）与覆盖字段。
const emit = defineEmits(['saved'])

const presets = ref({})        // { 预设名: AiPreset }
const names = computed(() => Object.keys(presets.value || {}))
const activePreset = ref('')   // settings.preset（当前使用中的预设名）
const naming = ref(false)
const newName = ref('')
const saving = ref(false)
const applying = ref('')
const error = ref('')
const snap = ref({})           // 保存时的当前配置快照（展示用）

function showError(msg) { error.value = msg; setTimeout(() => { if (error.value === msg) error.value = '' }, 4000) }

async function load() {
  try {
    const [p, st] = await Promise.all([
      api.getAiPresets().catch(() => ({ presets: {} })),
      api.apiGet('/settings').catch(() => ({ settings: {} })),
    ])
    presets.value = (p && p.presets) || {}
    activePreset.value = (st && st.settings && st.settings.preset) || ''
  } catch (e) {
    showError('加载预设失败: ' + (e.message || e))
  }
}

function saveCurrent() {
  const s = (window && window.__PAIRCODE_CORE && window.__PAIRCODE_CORE.uiState && window.__PAIRCODE_CORE.uiState.state
    && window.__PAIRCODE_CORE.uiState.state.settings) || {}
  snap.value = {
    provider: s.provider || '',
    baseURL: s.baseURL || '',
    apiKey: s.apiKey || '',
    executeModel: s.executeModel || s.model || '',
    planModel: s.planModel || '',
    reviewModel: s.reviewModel || '',
    temperature: s.temperature || '',
    thinkingMode: s.thinkingMode || '',
    maxTokens: s.maxTokens || 0,
    contextMaxTokens: s.contextMaxTokens || 0,
  }
  newName.value = ''
  naming.value = true
}

async function confirmSaveCurrent() {
  const name = newName.value.trim()
  if (!name) { showError('请输入预设名称'); return }
  saving.value = true
  error.value = ''
  try {
    // 服务端 save 未传 preset 时自动抓取当前 settings 快照（与前端同源）
    const r = await api.saveAiPreset('save', name)
    if (r && r.ok) {
      presets.value = r.presets || presets.value
      naming.value = false
      emit('saved')
    } else {
      showError((r && r.error) || '保存失败')
    }
  } catch (e) {
    showError('保存失败: ' + (e.message || e))
  } finally {
    saving.value = false
  }
}

async function applyPreset(name) {
  applying.value = name
  error.value = ''
  try {
    const r = await api.saveAiPreset('apply', name)
    if (r && r.ok) {
      activePreset.value = name
      emit('saved')   // 通知父级刷新（settings 已整套写回）
    } else {
      showError((r && r.error) || '应用失败')
    }
  } catch (e) {
    showError('应用失败: ' + (e.message || e))
  } finally {
    applying.value = ''
  }
}

async function removePreset(name) {
  if (!confirm('删除预设「' + name + '」？')) return
  error.value = ''
  try {
    const r = await api.saveAiPreset('delete', name)
    if (r && r.ok) {
      presets.value = r.presets || presets.value
      if (activePreset.value === name) activePreset.value = ''
      emit('saved')
    } else {
      showError((r && r.error) || '删除失败')
    }
  } catch (e) {
    showError('删除失败: ' + (e.message || e))
  }
}

async function startRename(name) {
  const nn = prompt('新名称：', name)
  if (!nn || nn === name) return
  error.value = ''
  try {
    const map = { ...(presets.value || {}) }
    map[nn] = map[name]
    delete map[name]
    const r = await api.saveAiPreset('rename', nn, null) // action=rename 需要 presets 全量
    // rename 走全量接口：先 PUT 全量（含改名后集合）
    const r2 = await api.saveAiPresets(map)
    if (r2 && r2.ok) {
      presets.value = map
      if (activePreset.value === name) activePreset.value = nn
      emit('saved')
    } else {
      showError((r2 && r2.error) || '改名失败')
    }
  } catch (e) {
    showError('改名失败: ' + (e.message || e))
  }
}

onMounted(load)
defineExpose({ load })
</script>

<style scoped>
/* 复用模型组面板样式前缀（mgm-*）以保证整体一致；预设专属样式 pm-* */
.pm-manager { font-size: 13px; }
.mgm-toolbar { display: flex; align-items: center; justify-content: space-between; margin-bottom: 8px; }
.mgm-count { color: var(--txt-dim, #8a8f98); font-size: 12px; }
.mgm-btn { padding: 4px 10px; border: 1px solid var(--bd, #333); border-radius: 4px; background: transparent; color: var(--txt, #ccc); cursor: pointer; font-size: 12px; }
.mgm-btn:hover { background: var(--bd-hover, #2a2a2a); }
.mgm-primary { background: var(--accent, #3b82f6); border-color: var(--accent, #3b82f6); color: #fff; }
.mgm-primary:hover { background: var(--accent-hover, #2f6fe0); }
.mgm-danger { color: #e05555; border-color: #5a3333; }
.mgm-small { padding: 2px 8px; font-size: 12px; }
.mgm-edit { border: 1px solid var(--bd, #333); border-radius: 6px; padding: 12px; margin-bottom: 12px; background: var(--bg-2, #1a1a1f); }
.mgm-edit-title { font-weight: 600; margin-bottom: 10px; }
.mgm-field { margin-bottom: 10px; }
.mgm-field-label { display: block; font-size: 12px; color: var(--txt-dim, #8a8f98); margin-bottom: 6px; }
.mgm-field input[type="text"] { width: 100%; padding: 6px 8px; border: 1px solid var(--bd, #333); border-radius: 4px; background: var(--bg, #111); color: var(--txt, #ccc); box-sizing: border-box; }
.mgm-edit-actions { display: flex; gap: 8px; }
.mgm-cards { display: flex; flex-direction: column; gap: 8px; }
.mgm-card { border: 1px solid var(--bd, #333); border-radius: 6px; padding: 10px 12px; }
.mgm-card.pm-active { border-color: var(--accent, #3b82f6); }
.mgm-card-head { display: flex; align-items: center; justify-content: space-between; margin-bottom: 6px; }
.mgm-name { font-weight: 600; }
.pm-active-badge { margin-left: 8px; font-size: 11px; color: var(--accent, #3b82f6); border: 1px solid var(--accent, #3b82f6); border-radius: 3px; padding: 0 4px; }
.mgm-ops { display: flex; gap: 6px; }
.mgm-empty { color: var(--txt-dim, #8a8f98); padding: 12px 0; font-size: 12px; }
.mgm-error { color: #e05555; margin-top: 8px; font-size: 12px; }
.pm-snapshot, .pm-preview { border: 1px dashed var(--bd, #333); border-radius: 4px; padding: 8px; display: flex; flex-direction: column; gap: 4px; }
.pm-snap-row { display: flex; justify-content: space-between; font-size: 12px; }
.pm-snap-row span { color: var(--txt-dim, #8a8f98); }
.pm-snap-mono { font-family: monospace; font-size: 11px; word-break: break-all; }
</style>
