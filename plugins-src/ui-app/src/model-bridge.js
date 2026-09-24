// ═══════════════════════════════════════════════════════════════
// model-bridge.js — 「3D 模型」面板 ↔ tool-model（同包 host 半）的通道
//
// ★ 合并后的口径（2026-09-17）：UI 与工具在**同一个插件包**（plugins-dist/tool-model/）里，
//   面板经 ui.invoke('tool-model', <方法>, args) 直连 host 半注册的方法：
//     listArtifacts  识别到的文件清单（工具写文件即登记 + 面板手动登记；★ 2026-09-18 起不扫工作区）
//     readArtifact   文件内容（文本原样 / 二进制 base64）
//     claimArtifact  点选即登记为「识别到的文件」
//     buildProject   点工程即用**同一份内核**现场构建几何
//   host 半写文件时广播 ui:tool-model/artifacts → onArtifacts 订阅 → 面板事件驱动实时刷新。
//
// 降级（旧宿主无 registerClientMethod/invoke）：退回 HTTP 只读三件套
//   （model.json / model.preview.html / model.verify.json，走 /api/fs/read），
//   此时不支持手动登记与二进制格式——面板会明确提示原因，不静默装作能用。
// ═══════════════════════════════════════════════════════════════
import api from './api.js'
import { getUIFor } from './plugin-runtime.js'
import { state } from './ui-state.js'

export const PLUGIN = 'tool-model'

// 降级路径的固定三件套（与旧面板行为一致）
const LEGACY_ORDER = ['model.json', 'model.verify.json', 'model.preview.html']

const KIND_BY_EXT = [
  [/\.preview\.html$/i, 'preview-html'],
  [/\.glb$/i, 'glb'],
  [/\.gltf$/i, 'gltf'],
  [/\.stl$/i, 'stl'],
  [/\.obj$/i, 'obj'],
  [/\.verify\.json$/i, 'verify'],
  [/\.json$/i, 'json'],
]

export function kindOfPath(p) {
  const s = String(p || '')
  for (const [re, k] of KIND_BY_EXT) if (re.test(s)) return k
  return ''
}

export function absPath(rel) {
  const root = String((state && state.workspaceRoot) || '').replace(/[\\/]+$/, '')
  if (!root) return rel
  return root + '/' + String(rel).replace(/^[\\/]+/, '')
}

function resolveUI(uiArg) {
  if (uiArg) return uiArg
  try { return getUIFor(PLUGIN) || null } catch (_) { return null }
}

export function createBridge(uiArg) {
  const ui = resolveUI(uiArg)
  const canInvoke = !!(ui && typeof ui.invoke === 'function')
  const invoke = canInvoke ? (m, a) => ui.invoke(PLUGIN, m, a) : null

  async function httpRead(path) {
    const r = await api.apiGet('/fs/read', { path: absPath(path) })
    const text = (r && r.content) || ''
    return { path, kind: kindOfPath(path), text, bytes: text.length, mtime: '' }
  }

  return {
    mode: canInvoke ? 'invoke' : 'http',
    modeNote: canInvoke
      ? '已连通 tool-model（同包 host 半）：工具产出自动登记、顶部路径框可手动登记任意文件、任意格式实时预览（不扫描工作区）'
      : '旧宿主无插件调用通道：只读 model.json / model.verify.json / model.preview.html（样式预览等需较新宿主）',

    // 识别到的文件清单（登记表：工具产出 + 面板手动登记）。args 里若还带 scan 也无妨——
    // host 半 2026-09-18 起忽略它（不再扫工作区），返回 scanned=false / scanRemoved=true。
    async listArtifacts(args = {}) {
      if (canInvoke) return invoke('listArtifacts', args)
      const items = []
      for (const p of LEGACY_ORDER) {
        try {
          const a = await httpRead(p)
          items.push({ path: p, kind: a.kind, label: '', source: 'http', bytes: a.bytes, mtime: '', note: null })
        } catch (_) { /* 不存在 */ }
      }
      return { items, registered: 0, scanned: false, at: Date.now() }
    },

    async readArtifact(path) {
      if (canInvoke) return invoke('readArtifact', { path })
      return httpRead(path)
    },

    // 点选 = 登记为「识别到的文件」（降级模式仅前端高亮，无 host 登记）
    async claimArtifact(path) {
      if (!canInvoke) return { ok: false, item: null }
      return invoke('claimArtifact', { path })
    },

    // 选中文件的实时状态（size/mtime）——实时复核用；降级模式返回 null（改由 listArtifacts 比较）
    async statArtifact(path) {
      if (!canInvoke) return null
      try { return await invoke('statArtifact', { path }) } catch (_) { return null }
    },

    // 点工程 → 同一份内核现场构建（降级模式不支持：内核在 host 半）
    async buildProject(path) {
      if (!canInvoke) throw new Error('当前宿主不支持插件调用（buildProject）：升级 IDE 后可用')
      return invoke('buildProject', { path })
    },

    // host 半写文件即广播 → 取消订阅函数
    onArtifacts(fn) {
      if (ui && typeof ui.on === 'function') {
        try { return ui.on('ui:tool-model/artifacts', fn) } catch (_) { return () => {} }
      }
      return () => {}
    },
  }
}
