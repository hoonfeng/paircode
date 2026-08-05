// chat-utils.js: 聊天消息渲染共享工具
// 供 RightPanel.vue 使用（ChatView.vue 已废弃删除）

import { computed } from 'vue'

// ─── 消息分组：将平铺的 user/assistant 消息按用户消息分组 ──
// 每组 { user, assistant }，assistant 可能为 null。
// ★ 保持对原消息对象的引用，WS 流式写入自动反映到 combo 内。
export function useMessageCombos(messages) {
  return computed(() => {
    const msgs = [...(messages || [])].sort((a, b) => (a._idx ?? 0) - (b._idx ?? 0))
    const combos = []
    let current = null
    let pendingFeedback = null
    for (const msg of msgs) {
      if (msg.role === 'user') {
        if (isFeedback(msg)) {
          pendingFeedback = msg
          continue
        }
        pendingFeedback = null
        current = { user: msg, assistant: null }
        combos.push(current)
      } else if (msg.role === 'assistant') {
        if (pendingFeedback && current) {
          if (!current.assistant) current.assistant = msg
          if (!current.assistant._feedbacks) current.assistant._feedbacks = []
          current.assistant._feedbacks.push({
            content: cleanMsgContent(pendingFeedback),
            replyMsg: msg
          })
          pendingFeedback = null
          continue
        }
        if (current && current.assistant === null) {
          current.assistant = msg
        } else {
          current = { user: null, assistant: msg }
          combos.push(current)
        }
        pendingFeedback = null
      }
    }
    return combos
  })
}

// ─── 消息辅助函数 ──

export function cleanMsgContent(msg) {
  if (!msg || !msg.content) return ''
  return msg.content.replace(/^【用户反馈】\s*/, '')
}

export function isFeedback(msg) {
  if (!msg || !msg.content) return false
  return msg.content.startsWith('【用户反馈】')
}

export function isDelegation(msg) {
  if (!msg) return false
  return !!(msg._delegation || (msg.content && msg.content.startsWith('委派')))
}

export function delegationAgent(msg) {
  return msg?._delegation || ''
}

export function isSystemMsg(msg) {
  if (!msg) return false
  return msg.role === 'system' ||
    (msg.role === 'user' && msg.content === '' && msg._noPrefix)
}

export function msgSummary(assistant) {
  if (!assistant) return ''
  if (typeof assistant._summary === 'string') return assistant._summary
  const content = assistant.content || ''
  if (!content) return ''
  return content.replace(/\n/g, ' ').slice(0, 50)
}

// ─── 工具渲染辅助 ──

function safeParse(json) {
  if (!json) return {}
  try { return JSON.parse(json) } catch { return {} }
}

export function toolMeta(seg) {
  const name = seg.name || ''
  const args = safeParse(seg.argsRaw)
  if (/^read_file\b/.test(name)) return { icon: 'file-text', title: '读取文件', detail: args.path || '', summary: '已读取' }
  if (/^write_file\b/.test(name)) return { icon: 'file-plus', title: '写入文件', detail: args.path || '', summary: '已写入' }
  if (/^edit_file|multi_edit\b/.test(name)) return { icon: 'edit', title: '编辑文件', detail: args.path || '', summary: '已编辑' }
  if (/^run_command\b/.test(name)) return { icon: 'terminal', title: '执行命令', detail: '$ ' + (args.command || '').slice(0, 60), summary: '已完成' }
  if (/^run_test\b/.test(name)) return { icon: 'check', title: '运行测试', detail: args.package_path || '', summary: '已完成' }
  if (/^search_content|search_files|find_symbol\b/.test(name)) return { icon: 'search', title: '搜索', detail: (args.pattern || args.symbol || '').slice(0, 60), summary: '已搜索' }
  if (/^web_search|web_fetch|web_debug\b/.test(name)) return { icon: 'globe', title: '网络', detail: (args.query || args.url || '').slice(0, 60), summary: '已完成' }
  if (/^git_\b/.test(name)) return { icon: 'source-control', title: 'Git', detail: args.file || '', summary: '已完成' }
  if (/^go_build|go_run|bug_detect|bug_fix\b/.test(name)) return { icon: 'terminal', title: name, detail: args.path || '', summary: '已完成' }
  if (/^screenshot_|image_|web_debug\b/.test(name)) return { icon: 'image', title: name, detail: '', summary: '已完成' }
  if (/^task_create|task_update|update_tasks|finish_task|generate_commit\b/.test(name)) return { icon: 'list', title: name, detail: '', summary: '已完成' }
  return { icon: 'zap', title: name, detail: '', summary: '' }
}

export function toolResultSummary(seg) {
  if (!seg.result) return ''
  const r = seg.result
  // 错误信息
  if (/^Error:|err\b|failed|失败\b/i.test(r)) {
    const short = r.split('\n')[0] || r
    return short.slice(0, 80)
  }
  // 成功标记
  const lines = r.split('\n').filter(l => l.trim())
  if (lines.length === 0) return ''
  return lines[0].slice(0, 80)
}

export function isTerminalTool(seg) {
  const name = seg.name || ''
  return /^run_command|run_test|go_build|go_run\b/.test(name)
}

export function formatTerminalCommand(seg) {
  if (!seg.argsRaw) return ''
  const args = safeParse(seg.argsRaw)
  return args.command || ''
}

// ─── 连续 assistant 消息合并（历史加载用）──

export function mergeConsecutiveAssistant(msgs) {
  const result = []
  for (const msg of msgs) {
    if (msg.role === 'assistant' && result.length > 0) {
      const last = result[result.length - 1]
      if (last.role === 'assistant') {
        if (msg.segments && msg.segments.length > 0) {
          if (!last.segments) last.segments = []
          for (const seg of msg.segments) {
            if (seg.type === 'content' && seg.content) {
              const lastContent = [...last.segments].reverse().find(s => s.type === 'content')
              if (lastContent && lastContent.content.endsWith(seg.content)) continue
              last.segments.push(seg)
            } else {
              last.segments.push(seg)
            }
          }
        }
        continue
      }
    }
    result.push(msg)
  }
  return result
}
