<template>
  <div class="modal-overlay" @click.self="$emit('close')">
    <div class="modal-content market-modal">
      <div class="modal-header">
        <h2><SvgIcon name="package" :size="20" /> 市场</h2>
        <div class="market-tabs">
          <button :class="{ active: tab === 'all' }" @click="tab='all';doSearch()">全部</button>
          <button :class="{ active: tab === 'mcp' }" @click="tab='mcp';doSearch()">MCP</button>
          <button :class="{ active: tab === 'skill' }" @click="tab='skill';doSearch()">技能</button>
        </div>
        <button class="modal-close" @click="$emit('close')">×</button>
      </div>
      <div class="modal-body">
        <div class="market-search">
          <div class="search-icon"><SvgIcon name="search" :size="14" /></div>
          <input v-model="query" placeholder="搜索 MCP 服务器或技能…" @input="debounceSearch" class="search-input" />
          <button v-if="query" class="search-clear" @click="query='';doSearch()">×</button>
        </div>

        <!-- 加载状态 -->
        <div v-if="loading" class="market-loading">
          <span class="dot-pulse"></span>
          <span>搜索中...</span>
        </div>

        <!-- 结果列表 -->
        <div v-else class="market-list" ref="listRef">
          <div v-for="item in items" :key="item.id" class="market-item">
            <div class="mi-icon" :class="'icon-' + item.kind">
              <SvgIcon :name="item.kind === 'skill' ? 'code' : 'package'" :size="20" />
            </div>
            <div class="mi-body">
              <div class="mi-name">{{ item.name }}</div>
              <div class="mi-desc">{{ item.description }}</div>
              <div class="mi-meta">
                <span class="mi-type" :class="'type-' + item.kind">{{ item.kind === 'mcp' ? 'MCP' : '技能' }}</span>
                <span v-if="item.tags" class="mi-tags">
                  <span v-for="tag in item.tags" :key="tag" class="mi-tag">{{ tag }}</span>
                </span>
                <span v-if="item.installed" class="mi-installed"><SvgIcon name="check" :size="10" /> 已安装</span>
              </div>
            </div>
            <button v-if="!item.installed"
                    class="mi-install-btn"
                    @click="installItem(item)"
                    :disabled="installing === item.id">
              <SvgIcon v-if="installing === item.id" name="cycle" :size="12" />
              {{ installing === item.id ? '安装中…' : '安装' }}
            </button>
            <button v-else class="mi-uninstall-btn" @click="uninstallItem(item)">
              <SvgIcon name="trash" :size="12" /> 卸载
            </button>
          </div>
          <div v-if="!loading && items.length === 0" class="market-empty">
            <div class="me-icon"><SvgIcon name="package" :size="32" /></div>
            <div v-if="query">未找到匹配 "{{ query }}" 的条目</div>
            <div v-else>市场中暂无可用条目</div>
            <div class="me-hint">试试其他关键词或分类</div>
          </div>
        </div>
      </div>
      <div class="modal-footer">
        <span class="market-count">共 {{ items.length }} 个条目</span>
        <span v-if="error" class="market-error">{{ error }}</span>
        <span class="market-tip">安装后下次对话生效</span>
        <button class="btn-secondary" @click="$emit('close')">关闭</button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import api from '../api.js'
import SvgIcon from './SvgIcon.vue'

const emit = defineEmits(['close'])

const tab = ref('all')
const query = ref('')
const items = ref([])
const installing = ref('')
const loading = ref(false)
const error = ref('')
const listRef = ref(null)

let debounceTimer = null
function debounceSearch() {
  clearTimeout(debounceTimer)
  loading.value = true
  debounceTimer = setTimeout(doSearch, 250)
}

async function doSearch() {
  loading.value = true
  error.value = ''
  try {
    const kind = tab.value === 'all' ? '' : tab.value
    const results = await api.apiGet('/marketplace/search', {
      q: query.value,
      kind: kind,
    })
    items.value = results || []
  } catch (err) {
    error.value = '搜索失败: ' + err.message
    items.value = []
  } finally {
    loading.value = false
  }
}

async function installItem(item) {
  installing.value = item.id
  error.value = ''
  try {
    const result = await api.apiPost('/marketplace/install', { id: item.id })
    item.installed = true
    window.$toast?.(result.message || '安装成功', 'success')
  } catch (err) {
    error.value = '安装失败: ' + err.message
    window.$toast?.('安装失败: ' + err.message, 'error')
  } finally {
    installing.value = ''
  }
}

async function uninstallItem(item) {
  // 卸载功能需要后端支持，目前简化：仅前端标记
  item.installed = false
  window.$toast?.('已标记为卸载状态', 'info')
}

onMounted(() => {
  doSearch()
})
</script>

<style scoped>
.modal-overlay {
  position: fixed;
  top: 0; left: 0; right: 0; bottom: 0;
  background: rgba(0,0,0,0.55);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  backdrop-filter: blur(4px);
}
.market-modal {
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 12px;
  width: 85vw;
  max-width: 720px;
  max-height: 80vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  box-shadow: 0 8px 32px rgba(0,0,0,0.3);
}

/* ── 头部 ── */
.modal-header {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border-color);
}
.modal-header h2 {
  font-size: 16px;
  color: var(--text-primary);
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}
.market-tabs {
  display: flex;
  gap: 2px;
  background: var(--bg-tertiary);
  border-radius: 6px;
  padding: 2px;
}
.market-tabs button {
  background: none;
  border: none;
  color: var(--text-secondary);
  font-size: 13px;
  padding: 5px 14px;
  cursor: pointer;
  border-radius: 4px;
  transition: all 0.15s;
}
.market-tabs button:hover { color: var(--text-primary); }
.market-tabs button.active {
  color: var(--text-primary);
  background: var(--bg-primary);
  box-shadow: 0 1px 3px rgba(0,0,0,0.15);
}
.modal-close {
  margin-left: auto;
  background: none;
  border: none;
  color: var(--text-secondary);
  font-size: 22px;
  cursor: pointer;
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 4px;
}
.modal-close:hover { color: var(--text-primary); background: var(--bg-hover); }

/* ── 主体 ── */
.modal-body {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

/* ── 搜索框 ── */
.market-search {
  position: relative;
  padding: 12px 16px;
  border-bottom: 1px solid var(--border-color);
}
.search-icon {
  position: absolute;
  left: 26px;
  top: 50%;
  transform: translateY(-50%);
  color: var(--text-muted);
  pointer-events: none;
}
.search-input {
  width: 100%;
  background: var(--input-bg);
  border: 1px solid var(--border-color);
  color: var(--text-primary);
  padding: 8px 12px 8px 34px;
  font-size: 14px;
  outline: none;
  border-radius: 8px;
  transition: border-color 0.15s;
}
.search-input:focus {
  border-color: var(--accent);
  box-shadow: 0 0 0 2px var(--focus-ring);
}
.search-clear {
  position: absolute;
  right: 26px;
  top: 50%;
  transform: translateY(-50%);
  background: var(--bg-tertiary);
  border: none;
  color: var(--text-muted);
  width: 20px;
  height: 20px;
  border-radius: 50%;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
}
.search-clear:hover { color: var(--text-primary); background: var(--bg-hover); }

/* ── 加载状态 ── */
.market-loading {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 40px;
  color: var(--text-muted);
  font-size: 13px;
}
.dot-pulse {
  width: 8px; height: 8px;
  background: var(--accent);
  border-radius: 50%;
  animation: pulse 1s infinite;
}
@keyframes pulse {
  0%, 100% { opacity: 0.3; transform: scale(0.8); }
  50% { opacity: 1; transform: scale(1.2); }
}

/* ── 列表 ── */
.market-list {
  flex: 1;
  overflow-y: auto;
  padding: 8px 12px;
}
.market-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 10px;
  border-radius: 8px;
  cursor: default;
  transition: background 0.1s;
}
.market-item:hover { background: var(--bg-hover); }
.market-item + .market-item {
  border-top: 1px solid var(--border-color);
  margin-top: 0;
}

.mi-icon {
  width: 40px;
  height: 40px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  font-size: 20px;
}
.icon-mcp { background: rgba(126, 184, 218, 0.15); color: #7eb8da; }
.icon-skill { background: rgba(212, 167, 78, 0.15); color: #d4a74e; }

.mi-body { flex: 1; min-width: 0; }
.mi-name { font-size: 14px; color: var(--text-primary); font-weight: 600; }
.mi-desc { font-size: 12px; color: var(--text-muted); margin-top: 3px; line-height: 1.4; }
.mi-meta { display: flex; gap: 6px; margin-top: 5px; flex-wrap: wrap; align-items: center; }
.mi-type { font-size: 10px; padding: 1px 8px; border-radius: 10px; font-weight: 500; }
.type-mcp { background: rgba(126, 184, 218, 0.15); color: #7eb8da; }
.type-skill { background: rgba(212, 167, 78, 0.15); color: #d4a74e; }
.mi-tags { display: flex; gap: 3px; flex-wrap: wrap; }
.mi-tag { font-size: 10px; padding: 0 5px; border-radius: 3px; background: var(--bg-tertiary); color: var(--text-muted); }
.mi-installed { font-size: 11px; color: #6a9955; display: flex; align-items: center; gap: 2px; }

.mi-install-btn, .mi-uninstall-btn {
  flex-shrink: 0;
  padding: 6px 14px;
  border-radius: 6px;
  font-size: 13px;
  cursor: pointer;
  border: none;
  display: flex;
  align-items: center;
  gap: 4px;
  transition: all 0.15s;
}
.mi-install-btn {
  background: var(--accent);
  color: #fff;
}
.mi-install-btn:hover { filter: brightness(1.1); transform: translateY(-1px); }
.mi-install-btn:disabled { opacity: 0.5; cursor: not-allowed; transform: none; }
.mi-uninstall-btn {
  background: var(--bg-tertiary);
  border: 1px solid var(--border-color);
  color: var(--text-secondary);
}
.mi-uninstall-btn:hover { color: #c03; border-color: #c03; background: rgba(204, 0, 51, 0.08); }

/* ── 空状态 ── */
.market-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 20px;
  color: var(--text-muted);
  text-align: center;
}
.me-icon { margin-bottom: 12px; opacity: 0.25; font-size: 32px; }
.me-hint { font-size: 12px; margin-top: 6px; opacity: 0.6; }

/* ── 底部 ── */
.modal-footer {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 20px;
  border-top: 1px solid var(--border-color);
}
.market-count { font-size: 12px; color: var(--text-muted); }
.market-error { font-size: 12px; color: #c03; flex: 1; }
.market-tip { margin-left: auto; font-size: 11px; color: var(--text-muted); }
.btn-secondary {
  background: var(--bg-tertiary);
  border: 1px solid var(--border-color);
  color: var(--text-primary);
  padding: 6px 16px;
  cursor: pointer;
  border-radius: 6px;
}
.btn-secondary:hover { background: var(--bg-hover); }
</style>
