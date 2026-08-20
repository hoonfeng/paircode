<template>
  <div class="modal-overlay">
    <div class="modal-content market-modal">
      <div class="modal-header">
        <h2><SvgIcon name="package" :size="20" /> 市场</h2>
        <div class="market-tabs">
          <button :class="{ active: tab === 'all' }" @click="tab='all';doSearch()">全部</button>
            <button v-for="s in marketTabs" :key="s.kind"
                    :class="{ active: tab === s.kind }"
                    @click="tab=s.kind;doSearch()">{{ s.label }}</button>
          <button :class="{ active: tab === 'installed' }" @click="tab='installed';loadInstalled()">已安装</button>
        </div>
        <button class="modal-close" @click="$emit('close')">×</button>
      </div>
      <div class="modal-body">
        <!-- 搜索栏（非「已安装」tab 显示） -->
        <div v-if="tab !== 'installed'" class="market-search">
          <div class="search-icon"><SvgIcon name="search" :size="14" /></div>
          <input v-model="query" placeholder="搜索 MCP / 技能 / npm 插件…" @input="debounceSearch" class="search-input" />
          <button v-if="query" class="search-clear" @click="query='';doSearch()">×</button>
          <button class="market-refresh-btn" @click="refreshRemote" :disabled="refreshing" :title="refreshTip">
            <SvgIcon :name="refreshing ? 'cycle' : 'refresh'" :size="14" />
            {{ refreshing ? '刷新中…' : '刷新' }}
          </button>
        </div>

        <!-- 「已安装」tab 的操作栏 -->
        <div v-if="tab === 'installed'" class="installed-toolbar">
          <button class="btn-add-mcp" @click="showAddMCP = true"><SvgIcon name="plus" :size="14" /> 添加 MCP 服务器</button>
          <button class="btn-refresh" @click="loadInstalled"><SvgIcon name="refresh" :size="14" /> 刷新</button>
        </div>

        <!-- 添加 MCP 表单 -->
        <div v-if="showAddMCP" class="mcp-form">
          <div class="mcp-form-row"><label>名称</label><input v-model="mcpForm.name" placeholder="如 my-server" /></div>
          <div class="mcp-form-row"><label>命令</label><input v-model="mcpForm.command" placeholder="如 npx / uvx" /></div>
          <div class="mcp-form-row"><label>参数</label><input v-model="mcpForm.argsText" placeholder="如 -y @modelcontextprotocol/server-filesystem" /></div>
          <div class="mcp-form-row"><label>层级</label>
            <select v-model="mcpForm.level">
              <option value="user">用户级（全局）</option>
              <option value="project">工作区级</option>
            </select>
          </div>
          <div class="mcp-form-actions">
            <button class="btn-primary" @click="saveMCP" :disabled="!mcpForm.name || savingMCP">
              {{ savingMCP ? '保存中…' : '保存' }}
            </button>
            <button class="btn-secondary" @click="showAddMCP = false; resetMCPForm()">取消</button>
          </div>
          <div v-if="mcpError" class="mcp-form-error">{{ mcpError }}</div>
        </div>

        <!-- 编辑 MCP 表单 -->
        <div v-if="editingMCP" class="mcp-form">
          <div class="mcp-form-row"><label>名称</label><input v-model="editMCPForm.name" /></div>
          <div class="mcp-form-row"><label>命令</label><input v-model="editMCPForm.command" /></div>
          <div class="mcp-form-row"><label>参数</label><input v-model="editMCPForm.argsText" /></div>
          <div class="mcp-form-row"><label>层级</label>
            <select v-model="editMCPForm.level">
              <option value="user">用户级（全局）</option>
              <option value="project">工作区级</option>
            </select>
          </div>
          <div class="mcp-form-actions">
            <button class="btn-primary" @click="updateMCP" :disabled="!editMCPForm.name">保存</button>
            <button class="btn-secondary" @click="editingMCP = false">取消</button>
          </div>
        </div>

        <!-- 技能内容查看 -->
        <div v-if="viewingSkill" class="skill-viewer">
          <div class="skill-viewer-header">
            <strong>{{ viewingSkill.name }}</strong>
            <button class="modal-close" @click="viewingSkill = null">×</button>
          </div>
          <pre class="skill-viewer-content">{{ viewingSkill.content }}</pre>
        </div>

        <!-- 加载状态 -->
        <div v-if="loading" class="market-loading">
          <span class="dot-pulse"></span>
          <span>搜索中...</span>
        </div>

        <!-- 已安装列表 -->
        <div v-else-if="tab === 'installed'" class="market-list" ref="listRef">
          <!-- MCP 分组 -->
          <div v-if="installedMCPs.length > 0" class="installed-group">
            <div class="installed-group-title">MCP 服务器</div>
            <div v-for="item in installedMCPs" :key="'mcp-' + item.name + '-' + item.level" class="installed-item">
              <div class="ii-icon icon-mcp"><SvgIcon name="package" :size="18" /></div>
              <div class="ii-status-dot" :class="item.enabled === false ? 'dot-disabled' : (item._connected ? 'dot-connected' : 'dot-idle')" :title="item.enabled === false ? '已禁用' : (item._connected ? '已连接' : '未连接')"></div>
              <div class="ii-body">
                <div class="ii-name">{{ item.name }}</div>
                <div class="ii-desc">{{ item.command }} {{ (item.args || []).join(' ') }}</div>
                <span class="ii-badge">MCP · {{ item.level === 'project' ? '工作区级' : '用户级' }}</span>
              </div>
              <div class="ii-actions">
                <button class="ii-btn ii-toggle" :class="{ 'is-enabled': item.enabled !== false }" @click="toggleMCP(item)" :title="item.enabled === false ? '点击启用' : '点击禁用'">
                  {{ item.enabled === false ? '禁用' : '启用' }}
                </button>
                <button class="ii-btn ii-edit" @click="startEditMCP(item)" title="编辑">编辑</button>
                <button class="ii-btn ii-del" @click="delMCP(item)" title="删除">删除</button>
              </div>
            </div>
          </div>
          <!-- 技能分组 -->
          <div v-if="installedSkills.length > 0" class="installed-group">
            <div class="installed-group-title">技能</div>
            <div v-for="item in installedSkills" :key="'skill-' + item.name + '-' + item.level" class="installed-item">
              <div class="ii-icon icon-skill"><SvgIcon name="code" :size="18" /></div>
              <div class="ii-body">
                <div class="ii-name">{{ item.name }}</div>
                <div class="ii-desc">{{ item.description || '无描述' }}</div>
                <span class="ii-badge">
                  技能 ·
                  <span :class="'status-' + (item.status || 'on')">
                    {{ item.status === 'off' ? '关闭' : item.status === 'max' ? '始终激活' : '按需' }}
                  </span>
                  · {{ item.level === 'system' ? '用户级' : '工作区级' }}
                </span>
              </div>
              <div class="ii-actions">
                <select class="ss-status-select" :value="item.status || 'on'" @change="setSkillStatus(item, $event.target.value)" :title="statusTitle(item)">
                  <option value="off">关闭</option>
                  <option value="on">按需</option>
                  <option value="max">始终激活</option>
                </select>
                <button class="ii-btn ii-view" @click="viewSkill(item)" title="查看内容">查看</button>
                <button v-if="item.level !== 'system'" class="ii-btn ii-del" @click="delSkill(item)" title="删除">删除</button>
              </div>
            </div>
          </div>
          <div v-if="installedMCPs.length === 0 && installedSkills.length === 0" class="market-empty">
            <div class="me-icon"><SvgIcon name="package" :size="32" /></div>
            <div>暂无已安装的 MCP 服务器或技能</div>
            <div class="me-hint">切换到「全部」tab 搜索安装，或点击上方「添加 MCP 服务器」</div>
          </div>
        </div>

        <!-- 搜索/浏览列表 -->
        <div v-else class="market-list" ref="listRef">
          <div v-for="item in items" :key="item.id" class="market-item">
            <div class="mi-icon" :class="'icon-' + item.kind">
              <SvgIcon :name="item.kind === 'plugin' ? 'puzzle' : (item.kind === 'skill' ? 'code' : 'package')" :size="20" />
            </div>
            <div class="mi-body">
              <div class="mi-name">{{ item.name }}</div>
              <div class="mi-desc">{{ item.description }}</div>
              <div class="mi-meta">
                <span class="mi-type" :class="'type-' + item.kind">{{ item.kind === 'mcp' ? 'MCP' : item.kind === 'plugin' ? '插件' : '技能' }}</span>
                <span v-if="item.tags" class="mi-tags">
                  <span v-for="tag in item.tags" :key="tag" class="mi-tag">{{ tag }}</span>
                </span>
                <span v-if="item.installed" class="mi-installed"><SvgIcon name="check" :size="10" /> 已安装</span>
              </div>
            </div>
            <div v-if="!item.installed" class="mi-install-area">
              <button v-if="item.kind !== 'mcp'"
                      class="mi-install-btn"
                      @click="installItem(item, 'user')"
                      :disabled="installing === item.id">
                <SvgIcon v-if="installing === item.id" name="cycle" :size="12" />
                {{ installing === item.id ? '安装中…' : '安装' }}
              </button>
              <template v-else>
                <select v-model="item._installScope" class="mi-scope-select" @click.stop>
                  <option value="user">用户级</option>
                  <option value="project">工作区级</option>
                </select>
                <button class="mi-install-btn"
                        @click="installItem(item, item._installScope || 'user')"
                        :disabled="installing === item.id">
                  <SvgIcon v-if="installing === item.id" name="cycle" :size="12" />
                  {{ installing === item.id ? '安装中…' : '安装' }}
                </button>
              </template>
            </div>
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
        <span class="market-count">{{ tab === 'installed' ? (installedMCPs.length + installedSkills.length) : items.length }} 个条目</span>
        <span v-if="error" class="market-error">{{ error }}</span>
        <span class="market-tip">安装后下次对话生效</span>
        <button class="btn-secondary" @click="$emit('close')">关闭</button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import api from '../api.js'
import SvgIcon from './SvgIcon.vue'

const emit = defineEmits(['close'])

const tab = ref('all')
const query = ref('')
const items = ref([])
const installing = ref('')
const loading = ref(false)
const refreshing = ref(false)
const error = ref('')
const refreshTip = ref('')
const listRef = ref(null)

// ── 市场源（插件化：磁盘插件 market-* 声明，动态 tab）──
const sources = ref([])
const marketTabs = computed(() => {
  const labelMap = { skill: '技能', mcp: 'MCP', plugin: '插件/工具集' }
  return (sources.value || []).map((s) => ({ kind: s.kind, label: labelMap[s.kind] || s.name || s.kind }))
})
async function loadSources() {
  try {
    const srcs = await api.apiGet('/marketplace/sources')
    sources.value = srcs || []
  } catch (e) {
    // 接口不可用（如 core-api 插件停用）→ 保持空 tab，搜索仍可经 kind 触发
  }
}

// ── 已安装管理 ──
const installedMCPs = ref([])
const installedSkills = ref([])
const showAddMCP = ref(false)
const savingMCP = ref(false)
const mcpError = ref('')
const editingMCP = ref(false)
const viewingSkill = ref(null)
const mcpForm = ref({ name: '', command: '', argsText: '', level: 'user' })
const editMCPForm = ref({ name: '', command: '', argsText: '', level: 'user', origName: '' })

function resetMCPForm() {
  mcpForm.value = { name: '', command: '', argsText: '', level: 'user' }
  mcpError.value = ''
}

async function loadInstalled() {
  loading.value = true
  error.value = ''
  try {
    const [mcpList, skillList] = await Promise.all([
      api.getMcpList('all'),
      api.getSkillsList()
    ])
    installedMCPs.value = mcpList || []
    installedSkills.value = skillList || []
  } catch (err) {
    error.value = '加载失败: ' + err.message
  } finally {
    loading.value = false
  }
}

async function saveMCP() {
  const f = mcpForm.value
  if (!f.name) return
  savingMCP.value = true
  mcpError.value = ''
  try {
    const args = f.argsText ? f.argsText.split(' ').filter(Boolean) : []
    await api.saveMcpItem({ action: 'save', name: f.name, command: f.command || 'npx', args, level: f.level })
    showAddMCP.value = false
    resetMCPForm()
    await loadInstalled()
  } catch (err) {
    mcpError.value = err.message
  } finally {
    savingMCP.value = false
  }
}

function startEditMCP(item) {
  editMCPForm.value = {
    origName: item.name,
    name: item.name,
    command: item.command,
    argsText: (item.args || []).join(' '),
    level: item.level,
  }
  editingMCP.value = true
}

async function updateMCP() {
  const f = editMCPForm.value
  if (!f.name) return
  try {
    if (f.origName !== f.name) {
      // 改名相当于先删旧再建新
      await api.saveMcpItem({ action: 'delete', name: f.origName, level: f.level })
    }
    const args = f.argsText ? f.argsText.split(' ').filter(Boolean) : []
    await api.saveMcpItem({ action: 'save', name: f.name, command: f.command || 'npx', args, level: f.level })
    editingMCP.value = false
    await loadInstalled()
  } catch (err) {
    error.value = '保存失败: ' + err.message
  }
}

async function delMCP(item) {
  if (!confirm(`确认删除 MCP 服务器「${item.name}」？`)) return
  try {
    await api.saveMcpItem({ action: 'delete', name: item.name, level: item.level })
    await loadInstalled()
  } catch (err) {
    error.value = '删除失败: ' + err.message
  }
}

async function viewSkill(item) {
  try {
    const data = await api.readSkill(item.name, item.level)
    viewingSkill.value = data || { name: item.name, content: '（内容读取失败）' }
  } catch (err) {
    viewingSkill.value = { name: item.name, content: '读取失败: ' + err.message }
  }
}

async function delSkill(item) {
  if (!confirm(`确认删除技能「${item.name}」？`)) return
  try {
    await api.deleteSkill(item.name)
    await loadInstalled()
  } catch (err) {
    error.value = '删除失败: ' + err.message
  }
}

function statusTitle(item) {
  if (item.status === 'off') return '已关闭：技能完全禁用，不加载'
  if (item.status === 'max') return '始终激活：技能常驻 system prompt'
  return '按需：根据关键词/文件匹配自动激活'
}

async function setSkillStatus(item, status) {
  try {
    await api.saveSkillStatus(item.name, item.level || 'project', status)
    item.status = status
    window.$toast?.(`技能「${item.name}」状态已设为 ${status === 'off' ? '关闭' : status === 'max' ? '始终激活' : '按需'}`, 'success')
  } catch (err) {
    error.value = '设置失败: ' + err.message
    window.$toast?.('设置失败: ' + err.message, 'error')
  }
}

// ── 市场搜索 ──

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

async function refreshRemote() {
  refreshing.value = true
  error.value = ''
  try {
    const result = await api.apiPost('/marketplace/refresh', {})
    refreshTip.value = result.status || '已刷新'
    window.$toast?.(result.message || '远程市场已刷新', 'success')
    await doSearch()
  } catch (err) {
    error.value = '刷新失败: ' + err.message
    window.$toast?.('刷新远程市场失败: ' + err.message, 'error')
  } finally {
    refreshing.value = false
    setTimeout(() => { refreshTip.value = '' }, 5000)
  }
}

async function installItem(item, scope) {
  installing.value = item.id
  error.value = ''
  try {
    const body = {
      id: item.id,
      kind: item.kind || '',
      command: item.command || '',
      args: item.args || [],
      source: item.source || '',
      description: item.description || '',
    }
    if (item.kind === 'mcp') {
      body.scope = scope || 'user'
    } else if (item.kind === 'plugin') {
      body.scope = 'project' // 插件/工具集默认装到工作区（npm 插件 → .pair/plugins/<name>/ 插件包目录）
    }
    const result = await api.apiPost('/marketplace/install', body)
    item.installed = true
    window.$toast?.(result.message || '安装成功', 'success')
  } catch (err) {
    error.value = '安装失败: ' + err.message
    window.$toast?.('安装失败: ' + err.message, 'error')
  } finally {
    installing.value = ''
  }
}

async function toggleMCP(item) {
  try {
    const result = await api.saveMcpItem({ action: 'toggle', name: item.name, level: item.level })
    item.enabled = result.enabled
    window.$toast?.(`MCP 服务器「${item.name}」${result.enabled ? '已启用' : '已禁用'}`, 'success')
  } catch (err) {
    error.value = '切换失败: ' + err.message
    window.$toast?.('切换失败: ' + err.message, 'error')
  }
}

async function uninstallItem(item) {
  error.value = ''
  try {
    const isNpm = (item.source || '').startsWith('npm:')
    if (isNpm) {
      // npm 插件：统一卸载接口（patch 移除 + 宿主卸载）
      await api.apiPost('/marketplace/uninstall', { id: item.id, kind: 'plugin', source: item.source })
    } else if (item.kind === 'mcp') {
      await api.saveMcpItem({ action: 'delete', name: item.id, level: 'user' })
    } else if (item.kind === 'skill') {
      await api.deleteSkill(item.id)
    } else if (item.kind === 'plugin') {
      await api.apiPost('/toolsets/remove', { name: item.id.replace(/^plugin-/, ''), scope: 'project' })
    }
    item.installed = false
    window.$toast?.('已卸载: ' + item.name, 'success')
  } catch (err) {
    error.value = '卸载失败: ' + err.message
    window.$toast?.('卸载失败: ' + err.message, 'error')
  }
}

onMounted(() => {
  loadSources()
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
  max-width: 800px;
  max-height: 80vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  box-shadow: 0 8px 32px rgba(0,0,0,0.3);
  height: 75vh;
  min-height: 400px;
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
  min-height: 0;
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
.market-refresh-btn {
  position: absolute;
  right: 50px;
  top: 50%;
  transform: translateY(-50%);
  background: var(--bg-tertiary);
  border: 1px solid var(--border-color);
  color: var(--text-secondary);
  padding: 4px 10px;
  border-radius: 4px;
  cursor: pointer;
  font-size: 12px;
  display: flex;
  align-items: center;
  gap: 4px;
  transition: all 0.15s;
}
.market-refresh-btn:hover { color: var(--text-primary); border-color: var(--accent); }
.market-refresh-btn:disabled { opacity: 0.5; cursor: not-allowed; }

/* ── 已安装工具栏 ── */
.installed-toolbar {
  display: flex;
  gap: 8px;
  padding: 10px 16px;
  border-bottom: 1px solid var(--border-color);
}
.btn-add-mcp {
  background: var(--accent);
  color: #fff;
  border: none;
  padding: 6px 14px;
  border-radius: 6px;
  cursor: pointer;
  font-size: 13px;
  display: flex;
  align-items: center;
  gap: 4px;
}
.btn-add-mcp:hover { filter: brightness(1.1); }
.btn-refresh {
  background: var(--bg-tertiary);
  border: 1px solid var(--border-color);
  color: var(--text-secondary);
  padding: 6px 14px;
  border-radius: 6px;
  cursor: pointer;
  font-size: 13px;
  display: flex;
  align-items: center;
  gap: 4px;
}
.btn-refresh:hover { color: var(--text-primary); }

/* ── MCP 表单 ── */
.mcp-form {
  padding: 12px 16px;
  border-bottom: 1px solid var(--border-color);
  background: var(--bg-tertiary);
}
.mcp-form-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 6px;
}
.mcp-form-row label { width: 50px; font-size: 13px; color: var(--text-secondary); flex-shrink: 0; }
.mcp-form-row input,
.mcp-form-row select {
  flex: 1;
  background: var(--input-bg);
  border: 1px solid var(--border-color);
  color: var(--text-primary);
  padding: 5px 8px;
  font-size: 13px;
  border-radius: 4px;
  outline: none;
}
.mcp-form-row input:focus { border-color: var(--accent); }
.mcp-form-actions { display: flex; gap: 8px; margin-top: 8px; }
.mcp-form-error { margin-top: 6px; font-size: 12px; color: #c03; }
.btn-primary {
  background: var(--accent); color: #fff; border: none;
  padding: 6px 16px; border-radius: 6px; cursor: pointer; font-size: 13px;
}
.btn-primary:disabled { opacity: 0.5; cursor: not-allowed; }
.btn-secondary {
  background: var(--bg-tertiary); border: 1px solid var(--border-color);
  color: var(--text-primary); padding: 6px 16px; cursor: pointer; border-radius: 6px; font-size: 13px;
}
.btn-secondary:hover { background: var(--bg-hover); }

/* ── 技能查看器 ── */
.skill-viewer {
  padding: 12px 16px;
  border-bottom: 1px solid var(--border-color);
  background: var(--bg-tertiary);
  max-height: 300px;
  overflow: auto;
}
.skill-viewer-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
  font-size: 14px;
  color: var(--text-primary);
}
.skill-viewer-header .modal-close {
  margin-left: auto;
  font-size: 18px;
  width: 24px;
  height: 24px;
}
.skill-viewer-content {
  font-family: var(--font-mono);
  font-size: 12px;
  color: var(--text-secondary);
  white-space: pre-wrap;
  word-break: break-all;
  line-height: 1.5;
  max-height: 240px;
  overflow: auto;
}

/* ── 已安装列表 ── */
.installed-group { margin-bottom: 4px; }
.installed-group-title {
  font-size: 11px;
  text-transform: uppercase;
  color: var(--text-muted);
  padding: 8px 12px 4px;
  letter-spacing: 0.5px;
}
.installed-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 10px;
  border-radius: 6px;
}
.installed-item:hover { background: var(--bg-hover); }
.ii-icon {
  width: 36px; height: 36px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}
.ii-body { flex: 1; min-width: 0; }
.ii-name { font-size: 13px; color: var(--text-primary); font-weight: 600; }
.ii-desc { font-size: 12px; color: var(--text-muted); margin-top: 2px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.ii-badge { font-size: 10px; color: var(--text-muted); background: var(--bg-tertiary); padding: 0 6px; border-radius: 3px; display: inline-block; margin-top: 2px; }
.ii-actions { display: flex; gap: 4px; flex-shrink: 0; }
.ii-btn {
  padding: 4px 10px;
  border-radius: 4px;
  font-size: 12px;
  cursor: pointer;
  border: 1px solid var(--border-color);
  background: var(--bg-tertiary);
  color: var(--text-secondary);
}
.ii-btn:hover { color: var(--text-primary); border-color: var(--accent); }
.ii-del:hover { color: #c03; border-color: #c03; background: rgba(204,0,51,0.08); }
.ii-toggle { min-width: 44px; }
.ii-toggle.is-enabled { color: #6a9955; border-color: #6a9955; background: rgba(106,153,85,0.08); }
.ii-toggle:not(.is-enabled) { color: var(--text-muted); border-color: var(--border-color); }

/* ── 技能状态选择器 ── */
.ss-status-select {
  background: var(--input-bg);
  border: 1px solid var(--border-color);
  color: var(--text-primary);
  font-size: 12px;
  padding: 3px 6px;
  border-radius: 4px;
  outline: none;
  cursor: pointer;
  min-width: 68px;
}
.ss-status-select:focus { border-color: var(--accent); }
.ss-status-select option { background: var(--bg-primary); color: var(--text-primary); }
.status-off { color: #c03; }
.status-max { color: #6a9955; }
.status-on { color: var(--text-secondary); }

/* ── 连接状态指示点 ── */
.ii-status-dot {
  width: 8px; height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
  margin-right: -4px;
}
.ii-status-dot.dot-connected { background: #6a9955; box-shadow: 0 0 4px rgba(106,153,85,0.5); }
.ii-status-dot.dot-idle { background: var(--text-muted); opacity: 0.4; }
.ii-status-dot.dot-disabled { background: #c03; opacity: 0.6; }

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
  min-height: 0;
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

/* ── 安装区域（含 scope 选择）── */
.mi-install-area { display: flex; align-items: center; gap: 4px; flex-shrink: 0; }
.mi-scope-select {
  background: var(--input-bg);
  border: 1px solid var(--border-color);
  color: var(--text-primary);
  font-size: 11px;
  padding: 4px 4px;
  border-radius: 4px;
  outline: none;
  cursor: pointer;
}
.mi-scope-select:focus { border-color: var(--accent); }
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
