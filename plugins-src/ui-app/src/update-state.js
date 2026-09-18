// update-state.js — 在线更新的全局状态（状态栏徽标 / 更新弹窗共用一份）。
//
// 数据源：GET /api/update/status（内核接口，见 cmd/companion/update_api.go + internal/update）。
//   · 后端在启动 20s 后自动检查一次（设置项 app-update.autoCheck，默认开），此后按
//     checkIntervalHours；本模块**只读状态、不主动触发网络检查**——避免每个浏览器标签
//     各自去打 GitHub（未启用更新接口的旧版本实例会静默失败，不显示徽标）。
//   · 有活动任务（检查/下载/校验/解压/安装/重启）时轮询收紧到 3s，空闲时 30s；
//     都是本地 GET，开销可忽略。
//
// 角色划分：StatusBar 显示「新版本 vX 可用」徽标 → openUpdateModal() → UpdateModal 内渲染
// UpdateCard（检查 → 下载 → 校验 → 安装重启）；「关于」弹窗内也复用同一卡片。
import { ref, computed } from 'vue'
import api from './api.js'
import { showUpdate } from './ui-state.js'

// 最近一次 /api/update/status 的 state 对象（null = 尚未取到/接口不可用）
export const updateInfo = ref(null)

// 任务进行中的阶段（轮询需加密）
const ACTIVE = ['checking', 'downloading', 'verifying', 'extracting', 'applying', 'restarting']
// 需要显示全局徽标的阶段（含「已下载就绪」——此时点开就能装）
const BADGE_STAGES = ['available', 'downloading', 'verifying', 'extracting', 'applying', 'restarting', 'ready']

let timer = null

export async function refreshUpdateStatus() {
  try {
    const r = await api.apiGet('/update/status')
    updateInfo.value = (r && r.state) || null
  } catch {
    // 接口不可用（旧实例 / 未启用）：保持静默，不显示徽标
    updateInfo.value = null
  }
  return updateInfo.value
}

function schedule() {
  const active = ACTIVE.includes((updateInfo.value || {}).stage)
  timer = setTimeout(async () => {
    await refreshUpdateStatus()
    schedule()
  }, active ? 3000 : 30000)
}

// 启动轮询（幂等：StatusBar 挂载时调用）
export function startUpdateWatch() {
  if (timer) return
  refreshUpdateStatus().then(schedule)
}

export function stopUpdateWatch() {
  if (timer) {
    clearTimeout(timer)
    timer = null
  }
}

export function openUpdateModal() {
  showUpdate.value = true
}

// ─── 状态栏徽标派生状态 ───
export const updateAvailable = computed(() => {
  const s = updateInfo.value
  if (!s) return false
  if (s.hasUpdate) return true
  return BADGE_STAGES.includes(s.stage)
})

export const updateBadgeText = computed(() => {
  const s = updateInfo.value || {}
  const v = s.latest ? `v${s.latest}` : '新版本'
  switch (s.stage) {
    case 'checking': return '检查更新…'
    case 'downloading': return `下载更新 ${Math.round(Number((s.progress || {}).percent) || 0)}%`
    case 'verifying': return '校验更新包…'
    case 'extracting': return '解压更新包…'
    case 'applying': return '安装更新…'
    case 'restarting': return '正在重启…'
    case 'ready': return `${v} 已就绪`
    default: return `新版本 ${v} 可用`
  }
})

export const updateTitle = computed(() => `${updateBadgeText.value} — 点击打开软件更新`)
