// ═══════════════════════════════════════════════════════════════
// conv-filters.js — 会话可见性过滤（PC 端界面策略）
//
// 微信桥（wechat-bridge 插件）的会话使用 `conv_wx_` 前缀命名（如
// conv_wx_微信、conv_wx_clawbot）——这些会话由桥投喂/读取，属于「桥专用
// 会话」：与 PC 端日常对话互不干扰，不出现在 PC 端会话列表/界面中
// （桥走 JSONL 文件直读 + /api/chat/send 投喂，不经前端列表，不受影响）。
//
// 过滤点：所有「拉取 /conversations 列表」的前端位置统一调用
// visibleConversations()——包括工作区切换、刷新重连、goja 预取等路径。
// ═══════════════════════════════════════════════════════════════

// 桥专用会话的 ID 前缀（与 wechat-bridge 桥的账号会话命名约定对齐：
// 桥的 ConvID 形如 conv_wx_<账号名>）
export const BRIDGE_CONV_PREFIXES = ['conv_wx_']

// isBridgeConversation 判断是否为桥专用会话（不出现在 PC 端界面）。
export function isBridgeConversation(id) {
  if (!id || typeof id !== 'string') return false
  return BRIDGE_CONV_PREFIXES.some((p) => id.startsWith(p))
}

// visibleConversations 过滤出 PC 端可见的会话列表（入参可为任意值）。
export function visibleConversations(list) {
  if (!Array.isArray(list)) return []
  return list.filter((c) => !isBridgeConversation(c && c.id))
}
