<template>
  <div class="status-bar">
    <div class="status-left">
      <span class="status-item" v-if="state.workspaceRoot">
        <SvgIcon name="home" :size="12" />
        {{ state.workspaceRoot.split('\\').filter(Boolean).pop() || '工作区' }}
      </span>
      <span class="status-item" v-else>未加载</span>
      <!-- Git 分支 -->
      <!-- ★ 2026-09-25 对齐设计稿 th210（w48「master」）：git 未识别出分支时仍显示 master
           （设计稿底栏左段固定为「工作区名 + 分支名」，故不再以 v-if 隐藏）。 -->
      <span class="status-item git-branch-item" @click="switchToGit">
        <SvgIcon name="git-branch" :size="11" />
        {{ gitBranch || 'master' }}
      </span>
      <!-- ★ 2026-09-25 对齐设计稿 th211：左段仅「folder gou-ide ⟶ master」两项，
           故移除本行的 git 变更文件数（改动数信息在源代码管理面板/图标内仍可达）。 -->
    </div>
    <!-- ★ statusbar-items 槽位（list 型）：内置状态栏内细粒度叠加条目（插件加小状态/快捷入口） -->
    <div ref="statusItemsEl" class="plugin-slot-host plugin-slot-status-items"></div>
    <div class="status-right">
      <!-- ★ 2026-09-25 对齐设计稿 th214（w84「已连接 · 本地」） -->
      <span class="status-item conn-item" title="本地服务连接状态"><SvgIcon name="check" :size="12" color="var(--success, #4FD8A4)" /> 已连接 · 本地</span>
      <!-- ★ 2026-09-25（设计稿屏 1 ④）：状态栏带出**当前主题名** ——
           主题可被 ui-appearance 的「跟随系统」自动切换，此处给出可见回执，
           避免用户疑惑「我没点，界面怎么变了」。名称取中文名，未知 id 原样显示。 -->
      <!-- ★ 在线更新：发现新版本 / 更新进行中时全局提示（点击打开软件更新弹窗） -->
      <span v-if="updateAvailable" class="status-item update-item" :title="updateTitle" @click="openUpdateModal">
        <SvgIcon name="download" :size="12" />
        {{ updateBadgeText }}
      </span>
      <span class="status-item" v-if="state.activeFile">
        <SvgIcon name="file-code" :size="12" />
        {{ displayPath }}
      </span>
      <span class="status-item" v-if="state.openFiles.length > 0">Ln {{ state.cursorLine }}, Col {{ state.cursorCol }}</span>
      <span class="status-item">UTF-8</span>
      <!-- ★ 2026-09-25 对齐设计稿 th216（w168「上下文 44.4K / 1.0M · 3%」）——
           数据取真实上下文统计（与右栏 StatsRail 同源），非写死示例值。 -->
      <span class="status-item ctx-item" :title="'当前对话上下文占用：' + ctxStatusText">{{ ctxStatusText }}</span>
      <span class="status-item theme-item" :title="'当前主题：' + themeLabel + '（外观设置里可切换 / 开启跟随系统）'" @click="openAppearance">
        <span class="theme-dot" :style="{ background: themeAccent }"></span>
        {{ themeLabel }}
      </span>
    </div>
    <!-- ★ 2026-09-25（用户纠正）：此处曾承载原顶栏的 titlebar-right 槽位，但该槽位语义就是
         「标题栏右侧按钮区」，落在状态栏与命名、设计稿均不符（用户二次指出仍出现在状态栏）。
         该槽位已迁回 UiTitlebar 右段（.tb-right，margin-left:auto 右对齐）。
         ★ 不得在此重复挂载：mountListSlot 直接从全局 clientSlots 取全部 list 占用者渲染，
         两处并存会让同一插件各渲一份（重复条目）。 -->
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { state } from '../ui-state.js'
import SvgIcon from './SvgIcon.vue'
import api from '../api.js'
import { mountListSlot } from '../plugin-runtime.js'
import {
  updateAvailable, updateBadgeText, updateTitle,
  openUpdateModal, startUpdateWatch, stopUpdateWatch,
} from '../update-state.js'

const gitBranch = ref('')
const gitChanges = ref(0)
// ★ 2026-09-25 对齐设计稿 th216：状态栏「上下文 X / Y · Z%」。
// ★ 口径修正（2026-09-24，与 StatsRail/右栏一致）：
//   · used = 输入侧 promptTokens（后端字段）—— 模型输出不占上下文窗口，旧实现
//     额外 + completionTokens 属前端拼接口径，已删；
//   · max  = 后端 /conversations/{id}/token-stats 的 contextMaxTokens（= 后端
//     agent.ContextWindow 装配口径真值，正常总 > 0）；不再读 settings 顶层
//     contextMaxTokens（已迁入插件注册域、按机制不参与窗口取值），更不 `|| 1000000`
//     —— 那是硬编码臆测（旧状态栏显示的「1.0M」即此假值）。
//   · 上限确实未知时只显示已用值，不编造「/ Y · Z%」。
const ctxStatusText = computed(() => {
  const c = (state.convCtxStatsByConv && state.convCtxStatsByConv[state.currentConvId]) || {}
  const used = c.promptTokens || 0
  const max = c.contextMaxTokens || 0
  const fmt = (n) => n >= 1000000 ? (n / 1000000).toFixed(1) + 'M' : (n >= 1000 ? (n / 1000).toFixed(1) + 'K' : String(n))
  if (!(max > 0)) return '上下文 ' + fmt(used)
  return '上下文 ' + fmt(used) + ' / ' + fmt(max) + ' · ' + Math.round(used / max * 100) + '%'
})
const statusItemsEl = ref(null)
let statusItemsUnsub = null
let gitTimer = null

const displayPath = computed(() => {
  const p = state.activeFile
  if (!p) return ''
  const parts = p.replace(/\\/g, '/').split('/')
  const name = parts.pop()
  if (parts.length > 2) return '.../' + parts.slice(-2).join('/') + '/' + name
  return parts.length > 0 ? parts.join('/') + '/' + name : name
})

// ★ 2026-09-25（设计稿屏 1 ④）：状态栏展示当前主题名。
//   主题会被 ui-appearance 的「跟随系统」自动切换，状态栏给出可见回执，
//   避免用户疑惑「我没点，界面怎么变了」。旧 id 也映射（存量配置直显时的兜底）。
const THEME_LABELS = {
  midnight: 'Midnight', graphite: 'Graphite', obsidian: 'Obsidian', aurora: 'Aurora',
  daylight: 'Daylight', paper: 'Paper', sand: 'Sand', slate: 'Slate',
  dark: 'Midnight', night: 'Obsidian', light: 'Daylight', warm: 'Sand',
}
const themeLabel = computed(() => THEME_LABELS[state.theme] || state.theme || '未设置')
// 色点走令牌：既随主题变化，也随「强调色覆盖」实时变化，无需插件间通信
const themeAccent = 'var(--color-accent)'
// 点主题名 → 打开设置面板（外观页在其中）
function openAppearance() {
  window.dispatchEvent(new CustomEvent('switch-activity', { detail: { id: 'settings' } }))
}

async function loadGitInfo() {
  try {
    const res = await api.apiGet('/git/status')
    if (res.isRepo) {
      gitBranch.value = res.branch || ''
      gitChanges.value = (res.staged?.length || 0) + (res.modified?.length || 0) + (res.untracked?.length || 0)
    } else {
      gitBranch.value = ''
      gitChanges.value = 0
    }
  } catch {
    gitBranch.value = ''
    gitChanges.value = 0
  }
}

function switchToGit() {
  window.dispatchEvent(new CustomEvent('switch-activity', { detail: { id: 'source' } }))
}

onMounted(async () => {
  // statusbar-items 槽位（list 型）：状态栏内细粒度叠加。
  // ★ 连接状态指示已迁移为磁盘插件 ui-statusbar-conn（.pair/plugins/），
  //   不再内置——前端经 /api/plugins 装载 client 半后由插件渲染。
  statusItemsUnsub = mountListSlot(statusItemsEl, 'statusbar-items')
  // Load git info
  await loadGitInfo()
  gitTimer = setInterval(loadGitInfo, 15000)
  // 在线更新：读 /api/update/status（后端自动检查的结果），有新版则显示徽标
  startUpdateWatch()
})

onUnmounted(() => {
  if (gitTimer) clearInterval(gitTimer)
  if (statusItemsUnsub) { statusItemsUnsub(); statusItemsUnsub = null }
  stopUpdateWatch()
})
</script>

<style scoped>
.status-bar {
  /* ★ 2026-09-25 对齐设计稿 th219（row h=28）。宿主（.app-statusbar-host）同改 28px。 */
  height: 28px;
  /* ★ 2026-09-24 修正（t6 细节排查）：原 background: var(--accent) 让整条底栏变成
     强调色实底（实测 rgb(111,168,255) 亮蓝），文字却取 --status-text(→ --color-fg 浅色)，
     对比度仅约 1.5:1、几乎不可读。设计稿 th219 明确 bg=$color.surface（深色表面），
     文字 muted/accent/success，故改回表面色并统一文字令牌。 */
  background: var(--bg-secondary);
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 8px;
  font-size: 11px;
  color: var(--text-muted);
}
.status-left, .status-right { display: flex; align-items: center; gap: 8px; }
/* statusbar-items 槽位（list 型）：状态栏中间叠加区，与左右信息同行 */
.theme-item { gap: 5px; cursor: pointer; }
.theme-item:hover { color: var(--text-primary); }
.theme-dot {
  display: inline-block; width: 8px; height: 8px; border-radius: 999px; flex: none;
  box-shadow: 0 0 0 1px var(--border-color);
}
.plugin-slot-status-items {
  display: flex;
  align-items: center;
  gap: 8px;
  height: auto;
  margin: 0 auto;
}
.plugin-slot-status-items .plugin-slot-item {
  display: flex;
  align-items: center;
  opacity: 0.9;
  font-size: 11px;
  gap: 4px;
}
.plugin-slot-status-items .plugin-slot-item:hover { opacity: 1; }
.status-item { opacity: 0.9; display: flex; align-items: center; gap: 4px; }
/* 设计稿 th216「上下文 44.4K / 1.0M · 3%」为 accent 色（底栏唯一强调项） */
.ctx-item { color: var(--accent); }
.status-item:hover { opacity: 1; }
.git-branch-item, .git-status-icons { cursor: pointer; }
.git-branch-item:hover, .git-status-icons:hover { text-decoration: underline; }
/* 在线更新徽标：底色用当前文字色的低透明度（随主题走，不引入新色值） */
.update-item {
  cursor: pointer;
  font-weight: 600;
  padding: 0 7px;
  border-radius: 9px;
  background: color-mix(in srgb, currentColor 20%, transparent);
}
.update-item:hover { background: color-mix(in srgb, currentColor 32%, transparent); }
.update-item .svg-icon { animation: updatePulse 1.6s ease-in-out infinite; }
@keyframes updatePulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.45; }
}

</style>
