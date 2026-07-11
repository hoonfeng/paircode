<template>
  <div class="git-panel">
    <!-- 加载中 -->
    <div v-if="loading && !hasData" class="git-loading">
      <SvgIcon name="refresh" :size="20" class="spinner" /><span>加载 Git 状态...</span>
    </div>

    <!-- 非 Git 仓库 -->
    <div v-else-if="!isRepo && hasData" class="git-empty">
      <SvgIcon name="source-control" :size="24" color="var(--text-muted)" />
      <span>非 Git 仓库</span>
      <span class="subtitle">此目录未初始化 Git</span>
      <button class="git-btn init-btn" @click="initRepo">初始化仓库</button>
    </div>

    <!-- Git 面板主体 -->
    <template v-else-if="isRepo">
      <!-- 仓库顶栏 -->
      <div class="git-repo-bar">
        <SvgIcon name="source-control" :size="14" color="var(--accent)" />
        <div class="git-branch-select" @click.stop="showBranchMenu = !showBranchMenu">
          <SvgIcon name="git-branch" :size="12" />
          <span class="branch-name">{{ currentBranch || '（无分支）' }}</span>
          <SvgIcon name="chevron-down" :size="10" />
        </div>
        <span v-if="ahead > 0" class="ahead-badge" title="领先上游">↑{{ ahead }}</span>
        <span v-if="behind > 0" class="behind-badge" title="落后上游">↓{{ behind }}</span>
        <div class="repo-actions">
          <button class="icon-btn" @click="refresh" title="刷新">
            <SvgIcon name="refresh" :size="13" :class="{ spinning: refreshing }" />
          </button>
        </div>
      </div>

      <!-- 分支管理菜单 -->
      <div v-if="showBranchMenu" class="branch-menu" @click.stop>
        <div class="branch-menu-header">
          <input v-model="branchFilter" placeholder="过滤分支..." class="branch-filter-input" />
        </div>
        <div class="branch-list">
          <div v-for="b in filteredBranches" :key="b"
               :class="['branch-item', { active: b === currentBranch }]"
               @click="switchBranch(b)">
            <SvgIcon :name="b === currentBranch ? 'check' : 'git-branch'" :size="12" />
            <span class="branch-item-name">{{ b }}</span>
            <button v-if="b !== currentBranch" class="branch-del-btn" @click.stop="deleteBranch(b)" title="删除分支">
              <SvgIcon name="close" :size="10" />
            </button>
          </div>
        </div>
        <div class="branch-menu-footer">
          <button class="git-btn" @click="showCreateBranch = true">新建分支</button>
          <button class="git-btn" @click="showBranchMenu = false">关闭</button>
        </div>
      </div>

      <!-- 操作栏 -->
      <div class="git-action-bar">
        <button class="git-btn action-btn" :disabled="!hasModified" @click="stageAll">
          <SvgIcon name="plus" :size="12" /> 全部暂存
        </button>
        <button class="git-btn action-btn commit-btn" :disabled="stagedCount === 0" @click="showCommitDialog = true">
          <SvgIcon name="check" :size="12" /> 提交
        </button>
        <div class="action-spacer"></div>
        <button class="icon-btn" @click="pull" title="拉取">
          <SvgIcon name="git-pull" :size="13" />
        </button>
        <button class="icon-btn" @click="showPushDialog = true" title="推送">
          <SvgIcon name="git-push" :size="13" />
        </button>
        <button class="icon-btn" @click="showStashPanel = !showStashPanel" title="暂存管理">
          <SvgIcon name="package" :size="13" />
        </button>
        <button class="icon-btn" @click="showIgnoreEditor = !showIgnoreEditor" title=".gitignore">
          <SvgIcon name="file-text" :size="13" />
        </button>
      </div>

      <!-- 变更区块 -->
      <div class="git-sections">
        <!-- 已暂存 -->
        <div class="section-block">
          <div class="section-header" @click="toggleCollapse('staged')">
            <SvgIcon :name="collapsed.staged ? 'chevron-right' : 'chevron-down'" :size="12" color="var(--accent)" />
            <span>已暂存 ({{ staged.length }})</span>
          </div>
          <div v-if="!collapsed.staged" class="section-items">
            <div v-for="item in staged" :key="item.path" class="file-row" @click="showFileDiff(item.path, true)">
              <span :class="'file-status staged'">{{ statusIcon(item.x) }}</span>
              <span class="file-path">{{ item.path }}</span>
              <div class="file-actions">
                <button class="row-btn" @click.stop="unstageFile(item.path)" title="取消暂存">
                  <SvgIcon name="minus" :size="12" />
                </button>
              </div>
            </div>
          </div>
        </div>

        <!-- 冲突 -->
        <div class="section-block">
          <div class="section-header conflict" @click="toggleCollapse('conflict')">
            <SvgIcon :name="collapsed.conflict ? 'chevron-right' : 'chevron-down'" :size="12" color="#f48771" />
            <span>冲突 ({{ conflict.length }})</span>
          </div>
          <div v-if="!collapsed.conflict" class="section-items">
            <div v-for="item in conflict" :key="item.path" class="file-row" @click="showFileDiff(item.path, false)">
              <span class="file-status conflict-st">!</span>
              <span class="file-path conflict-text">{{ item.path }}</span>
            </div>
          </div>
        </div>

        <!-- 已修改 -->
        <div class="section-block">
          <div class="section-header modified" @click="toggleCollapse('modified')">
            <SvgIcon :name="collapsed.modified ? 'chevron-right' : 'chevron-down'" :size="12" color="#dcdcaa" />
            <span>已修改 ({{ modified.length }})</span>
          </div>
          <div v-if="!collapsed.modified" class="section-items">
            <div v-for="item in modified" :key="item.path" class="file-row" @click="showFileDiff(item.path, false)">
              <span :class="'file-status modified-st'">{{ statusIcon(item.y) }}</span>
              <span class="file-path">{{ item.path }}</span>
              <div class="file-actions">
                <button class="row-btn" @click.stop="stageFile(item.path)" title="暂存">
                  <SvgIcon name="plus" :size="12" />
                </button>
                <button class="row-btn danger" @click.stop="discardFile(item.path)" title="丢弃">
                  <SvgIcon name="trash" :size="12" />
                </button>
              </div>
            </div>
          </div>
        </div>

        <!-- 未跟踪 -->
        <div class="section-block">
          <div class="section-header untracked" @click="toggleCollapse('untracked')">
            <SvgIcon :name="collapsed.untracked ? 'chevron-right' : 'chevron-down'" :size="12" color="var(--text-muted)" />
            <span>未跟踪 ({{ untracked.length }})</span>
          </div>
          <div v-if="!collapsed.untracked" class="section-items">
            <div v-for="item in untracked" :key="item.path" class="file-row" @click="showFileDiff(item.path, false)">
              <span class="file-status untracked-st">?</span>
              <span class="file-path untracked-text">{{ item.path }}</span>
              <div class="file-actions">
                <button class="row-btn" @click.stop="stageFile(item.path)" title="暂存">
                  <SvgIcon name="plus" :size="12" />
                </button>
              </div>
            </div>
          </div>
        </div>

        <!-- 工作区干净 -->
        <div v-if="totalChanges === 0 && commits.length > 0" class="clean-hint">
          <SvgIcon name="check" :size="14" color="var(--accent)" />
          <span>工作区干净</span>
        </div>
      </div>

      <!-- 提交历史 -->
      <div class="git-history">
        <div class="history-header" @click="toggleCollapse('history')">
          <SvgIcon :name="collapsed.history ? 'chevron-right' : 'chevron-down'" :size="12" color="var(--accent)" />
          <span>提交历史 ({{ commits.length }})</span>
          <button class="icon-btn" @click.stop="refreshCommits" title="刷新提交历史">
            <SvgIcon name="refresh" :size="11" />
          </button>
        </div>
        <div v-if="!collapsed.history" class="history-list">
          <div v-for="c in commits" :key="c.hash" class="commit-row" @dblclick="showCommitDetail(c)">
            <span class="commit-hash">{{ c.short }}</span>
            <span class="commit-msg">{{ c.msg }}</span>
            <span class="commit-date">{{ formatDate(c.date) }}</span>
          </div>
        </div>
      </div>
    </template>

    <!-- ≡≡≡ 对话框区域 ≡≡≡ -->

    <!-- 提交对话框 -->
    <Modal v-if="showCommitDialog" @close="showCommitDialog = false" :maxWidth="'420px'">
      <template #title>提交变更</template>
      <div class="form-layout">
        <input v-model="commitMsg" placeholder="提交信息（必填）" class="form-input" @keyup.enter="doCommit" />
        <textarea v-model="commitDesc" placeholder="详细描述（可选）" class="form-textarea" rows="3"></textarea>
        <div class="form-hint">{{ stagedCount }} 项已暂存</div>
        <div class="form-actions">
          <button class="git-btn" @click="showCommitDialog = false">取消</button>
          <button class="git-btn btn-primary" :disabled="!commitMsg.trim()" @click="doCommit">提交</button>
        </div>
      </div>
    </Modal>

    <!-- 推送对话框 -->
    <Modal v-if="showPushDialog" @close="showPushDialog = false" :maxWidth="'380px'">
      <template #title>推送</template>
      <div class="form-layout">
        <input v-model="pushRemote" placeholder="远程仓库（默认 origin）" class="form-input" />
        <input v-model="pushBranch" :placeholder="'分支（默认 ' + currentBranch + '）'" class="form-input" />
        <div class="form-actions">
          <button class="git-btn" @click="showPushDialog = false">取消</button>
          <button class="git-btn btn-primary" @click="doPush">推送</button>
        </div>
      </div>
    </Modal>

    <!-- 创建分支对话框 -->
    <Modal v-if="showCreateBranch" @close="showCreateBranch = false" :maxWidth="'360px'">
      <template #title>新建分支</template>
      <div class="form-layout">
        <input v-model="newBranchName" placeholder="分支名" class="form-input" @keyup.enter="createBranch" />
        <label class="form-checkbox">
          <input v-model="switchAfterCreate" type="checkbox" /> 创建后切换
        </label>
        <div class="form-actions">
          <button class="git-btn" @click="showCreateBranch = false">取消</button>
          <button class="git-btn btn-primary" :disabled="!newBranchName.trim()" @click="createBranch">创建</button>
        </div>
      </div>
    </Modal>

    <!-- 暂存管理面板 -->
    <Teleport to="body">
      <div v-if="showStashPanel" class="overlay" @click.self="showStashPanel = false">
        <div class="overlay-panel stash-panel">
          <div class="overlay-header">
            <span>暂存管理</span>
            <button class="icon-btn" @click="showStashPanel = false"><SvgIcon name="close" :size="14" /></button>
          </div>
          <div class="stash-form">
            <input v-model="stashMsg" placeholder="暂存备注（可选）" class="form-input" />
            <button class="git-btn btn-primary" @click="stashPush">暂存</button>
          </div>
          <div class="stash-list">
            <div v-for="s in stashes" :key="s.index" class="stash-item">
              <span class="stash-ref">{{ s.index }}</span>
              <span class="stash-msg">{{ s.msg }}</span>
              <div class="stash-actions">
                <button class="icon-btn" @click="stashPop(s.index)" title="弹出">
                  <SvgIcon name="undo" :size="12" />
                </button>
                <button class="icon-btn" @click="stashDrop(s.index)" title="删除">
                  <SvgIcon name="trash" :size="12" />
                </button>
              </div>
            </div>
            <div v-if="stashes.length === 0" class="stash-empty">没有暂存的更改</div>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- .gitignore 编辑器 -->
    <Teleport to="body">
      <div v-if="showIgnoreEditor" class="overlay" @click.self="showIgnoreEditor = false">
        <div class="overlay-panel ignore-panel">
          <div class="overlay-header">
            <span>.gitignore</span>
            <button class="icon-btn" @click="showIgnoreEditor = false"><SvgIcon name="close" :size="14" /></button>
          </div>
          <textarea v-model="ignoreContent" class="ignore-textarea" rows="12" spellcheck="false"></textarea>
          <div class="ignore-actions">
            <button class="git-btn" @click="showIgnoreEditor = false">取消</button>
            <button class="git-btn btn-primary" @click="saveIgnore">保存</button>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- 提交详情 -->
    <Modal v-if="showCommitDetailModal" @close="showCommitDetailModal = false" :maxWidth="'520px'">
      <template #title>{{ detailCommit?.short }} — {{ detailCommit?.msg?.substring(0, 40) }}</template>
      <div class="detail-content">
        <div class="detail-meta">
          <div><strong>作者：</strong>{{ detailCommit?.author }}</div>
          <div><strong>日期：</strong>{{ detailCommit?.date }}</div>
          <div><strong>哈希：</strong><code>{{ detailCommit?.hash }}</code></div>
        </div>
        <div class="detail-diff">
          <pre>{{ commitDiff || '加载中...' }}</pre>
        </div>
      </div>
      <div class="form-actions" style="padding: 8px 16px;">
        <button class="git-btn" @click="showCommitDetailModal = false">关闭</button>
        <button class="git-btn btn-primary" @click="copyHash(detailCommit?.hash)">复制哈希</button>
      </div>
    </Modal>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted, onUnmounted, watch } from 'vue'
import SvgIcon from './SvgIcon.vue'
import Modal from './Modal.vue'
import api from '../api.js'

// ─── 状态 ─────────────────────────────────────────────────────
const loading = ref(false)
const refreshing = ref(false)
const hasData = ref(false)
const isRepo = ref(false)
const currentBranch = ref('')
const ahead = ref(0)
const behind = ref(0)
const staged = ref([])
const conflict = ref([])
const modified = ref([])
const untracked = ref([])
const branches = ref([])
const commits = ref([])
const error = ref('')

const collapsed = reactive({
  staged: false, conflict: false, modified: false, untracked: false, history: false,
})

const showBranchMenu = ref(false)
const branchFilter = ref('')
const showCommitDialog = ref(false)
const showPushDialog = ref(false)
const showCreateBranch = ref(false)
const showStashPanel = ref(false)
const showIgnoreEditor = ref(false)
const showCommitDetailModal = ref(false)
const detailCommit = ref(null)
const commitDiff = ref('')

const commitMsg = ref('')
const commitDesc = ref('')
const pushRemote = ref('origin')
const pushBranch = ref('')
const newBranchName = ref('')
const switchAfterCreate = ref(true)
const stashMsg = ref('')
const stashes = ref([])
const ignoreContent = ref('')

let refreshTimer = null

// ─── 计算属性 ─────────────────────────────────────────────────
const hasModified = computed(() => modified.value.length + untracked.value.length > 0)
const stagedCount = computed(() => staged.value.length)
const totalChanges = computed(() => staged.value.length + conflict.value.length + modified.value.length + untracked.value.length)

const filteredBranches = computed(() => {
  if (!branchFilter.value) return branches.value
  const f = branchFilter.value.toLowerCase()
  return branches.value.filter(b => b.toLowerCase().includes(f))
})

// ─── 生命周期 ─────────────────────────────────────────────────
onMounted(() => {
  loadStatus()
  refreshTimer = setInterval(loadStatus, 30000)
  document.addEventListener('click', handleOutsideClick)
})

onUnmounted(() => {
  if (refreshTimer) clearInterval(refreshTimer)
  document.removeEventListener('click', handleOutsideClick)
})

function handleOutsideClick() {
  if (showBranchMenu.value) showBranchMenu.value = false
}

// ─── API ──────────────────────────────────────────────────────
async function loadStatus() {
  if (loading.value) return
  loading.value = true
  try {
    const res = await api.apiGet('/git/status')
    hasData.value = true
    isRepo.value = res.isRepo || false
    if (isRepo.value) {
      currentBranch.value = res.branch || ''
      ahead.value = res.ahead || 0
      behind.value = res.behind || 0
      staged.value = res.staged || []
      conflict.value = res.conflict || []
      modified.value = res.modified || []
      untracked.value = res.untracked || []
      branches.value = res.branches || []
      error.value = res.error || ''
    } else {
      error.value = res.error || '非 Git 仓库'
    }
    if (isRepo.value) {
      try {
        const log = await api.apiGet('/git/log', { count: 50 })
        commits.value = log || []
      } catch {}
    }
  } catch (err) {
    hasData.value = true
    error.value = err.message
    isRepo.value = false
  } finally {
    loading.value = false
    refreshing.value = false
  }
}

async function refresh() {
  refreshing.value = true
  await loadStatus()
  if (showStashPanel.value) await loadStashes()
}

async function refreshCommits() {
  try {
    commits.value = await api.apiGet('/git/log', { count: 50 }) || []
  } catch {}
}

async function loadStashes() {
  try {
    stashes.value = await api.apiGet('/git/stash-list') || []
  } catch { stashes.value = [] }
}

async function stageAll() {
  try { await api.apiPost('/git/add', { files: [] }); await loadStatus(); window.$toast?.('已全部暂存', 'success') }
  catch (err) { window.$toast?.('暂存失败: ' + err.message, 'error') }
}

async function stageFile(path) {
  try { await api.apiPost('/git/add', { files: [path] }); await loadStatus() }
  catch (err) { window.$toast?.('暂存失败: ' + err.message, 'error') }
}

async function unstageFile(path) {
  try { await api.apiPost('/git/reset', { files: [path] }); await loadStatus() }
  catch (err) { window.$toast?.('取消暂存失败: ' + err.message, 'error') }
}

async function discardFile(path) {
  const ok = await window.$confirm?.(`确定丢弃「${path}」的工作区更改？不可撤销。`, '丢弃更改', '确定丢弃', '取消')
  if (!ok) return
  try { await api.apiPost('/git/discard', { files: [path] }); await loadStatus() }
  catch (err) { window.$toast?.('丢弃失败: ' + err.message, 'error') }
}

async function doCommit() {
  if (!commitMsg.value.trim()) return
  try {
    await api.apiPost('/git/commit', { message: commitMsg.value, all: false })
    commitMsg.value = ''; commitDesc.value = ''
    showCommitDialog.value = false
    window.$toast?.('提交成功', 'success')
    await loadStatus()
  } catch (err) { window.$toast?.('提交失败: ' + err.message, 'error') }
}

async function switchBranch(name) {
  if (name === currentBranch.value) return
  try {
    await api.apiPost('/git/branch', { action: 'switch', name })
    showBranchMenu.value = false
    await loadStatus()
  } catch (err) { window.$toast?.('切换分支失败: ' + err.message, 'error') }
}

async function deleteBranch(name) {
  const ok = await window.$confirm?.(`确定删除分支「${name}」？`, '删除分支', '确定删除', '取消')
  if (!ok) return
  try { await api.apiPost('/git/branch', { action: 'delete', name }); await loadStatus() }
  catch (err) { window.$toast?.('删除分支失败: ' + err.message, 'error') }
}

async function createBranch() {
  if (!newBranchName.value.trim()) return
  try {
    if (switchAfterCreate.value) {
      await api.apiPost('/git/branch', { action: 'create-switch', name: newBranchName.value })
    } else {
      await api.apiPost('/git/branch', { action: 'create', name: newBranchName.value })
    }
    newBranchName.value = ''; showCreateBranch.value = false
    await loadStatus()
  } catch (err) { window.$toast?.('创建分支失败: ' + err.message, 'error') }
}

async function doPush() {
  try {
    const body = {}
    if (pushRemote.value && pushRemote.value !== 'origin') body.remote = pushRemote.value
    if (pushBranch.value) body.branch = pushBranch.value
    await api.apiPost('/git/push', body)
    showPushDialog.value = false
    window.$toast?.('推送成功', 'success')
    await loadStatus()
  } catch (err) { window.$toast?.('推送失败: ' + err.message, 'error') }
}

async function pull() {
  try {
    await api.apiPost('/git/pull', {})
    window.$toast?.('拉取成功', 'success')
    await loadStatus()
  } catch (err) { window.$toast?.('拉取失败: ' + err.message, 'error') }
}

async function initRepo() {
  try {
    const res = await api.apiPost('/system/exec', { command: 'git init' })
    if (res.exitCode === 0) {
      window.$toast?.('Git 仓库已初始化', 'success')
      await loadStatus()
    } else {
      window.$toast?.('初始化失败: ' + (res.stderr || '未知错误'), 'error')
    }
  } catch (err) { window.$toast?.('初始化失败: ' + err.message, 'error') }
}

async function stashPush() {
  try {
    await api.apiPost('/git/stash', { action: 'push', message: stashMsg.value })
    stashMsg.value = ''
    window.$toast?.('已暂存', 'success')
    await loadStatus(); await loadStashes()
  } catch (err) { window.$toast?.('暂存失败: ' + err.message, 'error') }
}

async function stashPop(index) {
  try {
    await api.apiPost('/git/stash', { action: 'pop', index })
    window.$toast?.('已弹出暂存', 'success')
    await loadStatus(); await loadStashes()
  } catch (err) { window.$toast?.('弹出失败: ' + err.message, 'error') }
}

async function stashDrop(index) {
  const ok = await window.$confirm?.(`确定删除暂存 ${index}？`, '删除暂存', '确定', '取消')
  if (!ok) return
  try { await api.apiPost('/git/stash', { action: 'drop', index }); await loadStashes() }
  catch (err) { window.$toast?.('删除失败: ' + err.message, 'error') }
}

async function saveIgnore() {
  try {
    await api.apiPost('/git/ignore', { content: ignoreContent.value })
    showIgnoreEditor.value = false
    window.$toast?.('.gitignore 已保存', 'success')
  } catch (err) { window.$toast?.('保存失败: ' + err.message, 'error') }
}

async function loadIgnore() {
  try {
    const res = await api.apiGet('/git/ignore')
    ignoreContent.value = res.content || ''
  } catch { ignoreContent.value = '' }
}

async function showFileDiff(path, staged) {
  try {
    const res = await api.apiGet('/git/diff', { file: path, staged: staged ? 'true' : 'false' })
    const content = res.diff || '（无差异）'
    window.$alert?.(content, '差异 — ' + path)
  } catch (err) {
    window.$toast?.('无法加载差异: ' + err.message, 'error')
  }
}

async function showCommitDetail(c) {
  detailCommit.value = c
  commitDiff.value = ''
  showCommitDetailModal.value = true
  try {
    const res = await api.apiGet('/system/exec', { command: 'git show --stat ' + c.hash })
    commitDiff.value = res.stdout || '（无输出）'
  } catch { commitDiff.value = '（无法加载详情）' }
}

async function copyHash(hash) {
  if (!hash) return
  try {
    await navigator.clipboard.writeText(hash)
    window.$toast?.('已复制', 'success')
  } catch {}
}

// ─── 辅助 ─────────────────────────────────────────────────────
function toggleCollapse(key) { collapsed[key] = !collapsed[key] }
function formatDate(d) { return d ? d.substring(0, 10) : '' }
function statusIcon(s) {
  if (s === 'M' || s === 'm') return '~'
  if (s === 'A' || s === 'a') return '+'
  if (s === 'D' || s === 'd') return '-'
  if (s === 'R' || s === 'r') return '→'
  if (s === '?' || s === '!') return s
  return '~'
}

watch(showStashPanel, v => { if (v) loadStashes() })
watch(showIgnoreEditor, v => { if (v) loadIgnore() })
</script>

<style scoped>
.git-panel { display: flex; flex-direction: column; height: 100%; font-size: 12px; overflow: hidden; color: var(--text-primary); }

/* 加载与空状态 */
.git-loading, .git-empty {
  display: flex; flex-direction: column; align-items: center; justify-content: center;
  padding: 24px; gap: 8px; color: var(--text-muted); flex: 1;
}
.git-empty .subtitle { font-size: 11px; color: var(--text-muted); }
.spinner { animation: spin 1s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }

/* 仓库顶栏 */
.git-repo-bar {
  display: flex; align-items: center; height: 32px; padding: 0 8px;
  background: var(--bg-tertiary); border-bottom: 1px solid var(--border-color); gap: 6px;
  position: relative; flex-shrink: 0;
}
.git-branch-select {
  display: flex; align-items: center; gap: 4px; cursor: pointer;
  padding: 2px 6px; border-radius: 3px; font-size: 12px; color: var(--text-primary);
}
.git-branch-select:hover { background: var(--bg-hover); }
.branch-name { max-width: 100px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.ahead-badge { color: #4ec9b0; font-size: 10px; }
.behind-badge { color: #dcdcaa; font-size: 10px; }
.repo-actions { margin-left: auto; display: flex; gap: 2px; }
.icon-btn {
  display: inline-flex; align-items: center; justify-content: center;
  width: 24px; height: 24px; cursor: pointer; border: none; background: none;
  color: var(--text-secondary); border-radius: 3px;
}
.icon-btn:hover { background: var(--bg-hover); color: var(--text-primary); }
.spinning { animation: spin 1s linear infinite; }

/* 分支菜单 */
.branch-menu {
  position: absolute; top: 32px; left: 8px; z-index: 100;
  width: calc(100% - 16px); max-width: 300px;
  background: var(--bg-primary); border: 1px solid var(--border-color);
  border-radius: 6px; box-shadow: 0 4px 16px rgba(0,0,0,0.3);
  max-height: 300px; display: flex; flex-direction: column;
}
.branch-filter-input {
  width: 100%; padding: 4px 8px; font-size: 12px;
}
.branch-filter-input:focus { border-color: var(--accent); }
.branch-list { flex: 1; overflow-y: auto; }
.branch-item {
  display: flex; align-items: center; gap: 6px; padding: 6px 10px;
  cursor: pointer; font-size: 12px; color: var(--text-secondary);
}
.branch-item:hover { background: var(--bg-hover); color: var(--text-primary); }
.branch-item.active { color: var(--accent); font-weight: 600; }
.branch-item-name { flex: 1; overflow: hidden; text-overflow: ellipsis; }
.branch-del-btn {
  display: flex; align-items: center; justify-content: center;
  width: 18px; height: 18px; border: none; background: none;
  cursor: pointer; color: var(--text-muted); border-radius: 3px;
}
.branch-del-btn:hover { background: rgba(244,135,113,0.2); color: #f48771; }
.branch-menu-footer {
  display: flex; gap: 4px; padding: 8px; border-top: 1px solid var(--border-color);
  justify-content: flex-end;
}

/* 操作栏 */
.git-action-bar {
  display: flex; align-items: center; padding: 6px; gap: 4px; flex-wrap: wrap; flex-shrink: 0;
}
.git-btn {
  display: inline-flex; align-items: center; gap: 4px;
  padding: 4px 10px; font-size: 11px; cursor: pointer;
  background: var(--bg-tertiary); border: 1px solid var(--border-color);
  color: var(--text-primary); border-radius: 3px; white-space: nowrap;
}
.git-btn:hover:not(:disabled) { background: var(--bg-hover); }
.git-btn:disabled { opacity: 0.4; cursor: default; }
.action-btn { font-size: 11px; min-height: 24px; }
.btn-primary { background: var(--accent); color: #fff; border-color: var(--accent); }
.btn-primary:hover:not(:disabled) { filter: brightness(1.1); }
.action-spacer { flex: 1; }
.init-btn { margin-top: 8px; background: var(--accent); color: #fff; border: none; padding: 6px 16px; }

/* 变更区块 */
.git-sections { flex: 1; overflow-y: auto; border-top: 1px solid var(--border-color); }
.section-block { border-bottom: 1px solid var(--border-color); }
.section-header {
  display: flex; align-items: center; gap: 4px; padding: 5px 8px;
  cursor: pointer; font-size: 11px; color: var(--accent); user-select: none;
}
.section-header:hover { background: var(--bg-hover); }
.section-header.conflict { color: #f48771; }
.section-header.modified { color: #dcdcaa; }
.section-header.untracked { color: var(--text-muted); }
.section-items { max-height: 300px; overflow-y: auto; }
.file-row {
  display: flex; align-items: center; gap: 4px; padding: 3px 8px 3px 20px;
  cursor: pointer; font-size: 12px;
}
.file-row:hover { background: var(--bg-hover); }
.file-status {
  width: 14px; text-align: center; font-size: 12px; font-weight: bold; flex-shrink: 0;
}
.file-status.staged { color: var(--accent); }
.file-status.modified-st { color: #dcdcaa; }
.file-status.untracked-st { color: var(--text-muted); }
.file-status.conflict-st { color: #f48771; }
.file-path { flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.file-path.conflict-text { color: #f48771; }
.file-path.untracked-text { color: var(--text-muted); }
.file-actions { display: flex; gap: 2px; opacity: 0; }
.file-row:hover .file-actions { opacity: 1; }
.row-btn {
  display: inline-flex; align-items: center; justify-content: center;
  width: 20px; height: 20px; cursor: pointer; border: none; background: none;
  color: var(--text-muted); border-radius: 3px;
}
.row-btn:hover { background: var(--bg-hover); color: var(--accent); }
.row-btn.danger:hover { color: #f48771; }
.clean-hint {
  display: flex; align-items: center; gap: 6px; padding: 8px 12px;
  color: var(--text-muted); font-size: 11px;
}

/* 提交历史 */
.git-history { border-top: 1px solid var(--border-color); flex-shrink: 0; }
.history-header {
  display: flex; align-items: center; gap: 4px; padding: 5px 8px;
  cursor: pointer; font-size: 11px; color: var(--accent);
}
.history-header:hover { background: var(--bg-hover); }
.history-list { max-height: 160px; overflow-y: auto; }
.commit-row {
  display: flex; align-items: center; gap: 6px; padding: 3px 8px 3px 20px;
  cursor: pointer; font-size: 11px;
}
.commit-row:hover { background: var(--bg-hover); }
.commit-hash { color: var(--accent); width: 48px; flex-shrink: 0; font-family: monospace; font-size: 10px; }
.commit-msg { flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.commit-date { color: var(--text-muted); font-size: 10px; flex-shrink: 0; }

/* 对话框表单 */
.form-layout { padding: 16px; display: flex; flex-direction: column; gap: 8px; }
.form-input, .form-textarea {
  width: 100%; padding: 6px 8px; font-size: 13px;
  background: var(--input-bg); border: 1px solid var(--border-color);
  color: var(--text-primary); border-radius: 3px; outline: none;
}
.form-input:focus, .form-textarea:focus { border-color: var(--accent); }
.form-textarea { resize: vertical; font-family: inherit; }
.form-hint { color: var(--text-muted); font-size: 10px; }
.form-actions { display: flex; gap: 8px; justify-content: flex-end; }
.form-checkbox { display: flex; align-items: center; gap: 6px; font-size: 12px; color: var(--text-secondary); cursor: pointer; }

/* 暂存面板 */
.overlay {
  position: fixed; top: 0; left: 0; right: 0; bottom: 0;
  background: rgba(0,0,0,0.4); z-index: 1000;
  display: flex; align-items: flex-start; justify-content: center; padding-top: 80px;
}
.overlay-panel {
  background: var(--bg-primary); border: 1px solid var(--border-color);
  border-radius: 8px; display: flex; flex-direction: column; box-shadow: 0 8px 32px rgba(0,0,0,0.4);
}
.stash-panel { width: 480px; max-height: 400px; }
.overlay-header {
  display: flex; align-items: center; justify-content: space-between;
  padding: 10px 12px; border-bottom: 1px solid var(--border-color);
  font-size: 13px; font-weight: 600;
}
.stash-form { display: flex; gap: 8px; padding: 8px 12px; align-items: center; }
.stash-form .form-input { flex: 1; }
.stash-list { flex: 1; overflow-y: auto; }
.stash-item {
  display: flex; align-items: center; gap: 8px; padding: 6px 12px;
  border-bottom: 1px solid var(--border-color); font-size: 12px;
}
.stash-ref { color: var(--accent); font-family: monospace; font-size: 11px; width: 70px; flex-shrink: 0; }
.stash-msg { flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.stash-actions { display: flex; gap: 2px; }
.stash-empty { padding: 20px; text-align: center; color: var(--text-muted); font-size: 12px; }

/* .gitignore 编辑器 */
.ignore-panel { width: 500px; max-height: 500px; }
.ignore-textarea {
  width: 100%; padding: 8px 12px; font-family: monospace; font-size: 12px;
  background: var(--input-bg); border: none; color: var(--text-primary);
  resize: vertical; outline: none; line-height: 1.5;
}
.ignore-actions { display: flex; gap: 8px; padding: 8px 12px; justify-content: flex-end; border-top: 1px solid var(--border-color); }

/* 提交详情 */
.detail-content { padding: 12px 16px; max-height: 400px; overflow-y: auto; }
.detail-meta { margin-bottom: 12px; font-size: 12px; color: var(--text-secondary); line-height: 1.8; }
.detail-meta code { background: var(--bg-tertiary); padding: 1px 6px; border-radius: 3px; font-size: 11px; }
.detail-diff {
  background: var(--bg-tertiary); border-radius: 4px; padding: 12px;
  font-family: monospace; font-size: 11px; line-height: 1.5;
  overflow-x: auto; color: var(--text-primary); white-space: pre-wrap;
  max-height: 300px; overflow-y: auto;
}

/* 滚动条 */
.git-sections::-webkit-scrollbar,
.history-list::-webkit-scrollbar,
.stash-list::-webkit-scrollbar,
.detail-content::-webkit-scrollbar { width: 4px; }
.git-sections::-webkit-scrollbar-thumb,
.history-list::-webkit-scrollbar-thumb,
.stash-list::-webkit-scrollbar-thumb,
.detail-content::-webkit-scrollbar-thumb { background: var(--scrollbar-thumb); border-radius: 2px; }
</style>
