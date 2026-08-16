<template>
  <div v-if="waiting" class="approval-bar">
    <div class="approval-bar-icon"><SvgIcon name="shield" :size="14" /></div>
    <div class="approval-bar-info">
      <span class="approval-bar-tool">{{ toolLabel }}</span>
      <span class="approval-bar-args">{{ structuredSummary }}</span>
      <!-- 回复输入：拒绝时可填写原因，允许时也可附加说明 -->
      <textarea
        v-if="showReply"
        class="approval-bar-reply"
        v-model="replyText"
        :placeholder="replyPlaceholder"
        rows="2"
        @keydown="onReplyKeydown"
      ></textarea>
    </div>
    <div class="approval-bar-actions">
      <button class="approval-btn approval-btn-allow" @click="resolve(true)">允许</button>
      <button class="approval-btn approval-btn-deny" @click="resolve(false)">拒绝</button>
      <button class="approval-btn approval-btn-toggle" @click="showReply = !showReply" :title="showReply ? '收起回复' : '填写回复'">
        <SvgIcon name="message-square" :size="12" />
      </button>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import SvgIcon from './SvgIcon.vue'

const props = defineProps({
  waiting: { type: Boolean, default: false },
  tool: { type: String, default: '' },
  args: { type: String, default: '' },
  parsedArgs: { type: Object, default: () => ({}) },
})
const emit = defineEmits(['resolve'])

const showReply = ref(false)
const replyText = ref('')

function resolve(approved) {
  emit('resolve', { approved, reply: replyText.value.trim() })
  replyText.value = ''
  showReply.value = false
}

function onReplyKeydown(e) {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    resolve(!e.metaKey && !e.ctrlKey) // Ctrl+Enter=拒绝, Enter=允许
  }
}

const replyPlaceholder = computed(() => {
  return '输入回复内容...（Enter 允许，Ctrl+Enter 拒绝）'
})

// 工具中文标签映射
const toolLabels = {
  'edit_file': '编辑文件',
  'multi_edit': '批量编辑文件',
  'write_file': '写入文件',
  'delete_file': '删除文件',
  'move_file': '移动/重命名文件',
  'run_command': '执行命令',
  'git_add': 'Git 暂存',
  'git_commit': 'Git 提交',
  'git_reset': 'Git 重置',
  'git_checkout': 'Git 检出',
  'git_branch': 'Git 分支',
  'git_stash': 'Git 贮藏',
  'code_fix': '简化代码',
  'code_format': '格式化代码',
  'go_build': 'Go 构建',
  'run_test': '运行测试',
  'go_run': '运行程序',
  'debug_start': '启动调试',
  'write_binary': '写入二进制',
}

const toolLabel = computed(() => {
  return toolLabels[props.tool] || props.tool || '未知操作'
})

// 从 args 解析参数的显示名（结构化展示）
function describeFileOp(args) {
  const p = props.parsedArgs
  if (p.path) {
    let desc = p.path
    if (p.line_start && p.line_end) {
      desc += `（第${p.line_start}-${p.line_end}行）`
    } else if (p.line_start) {
      desc += `（第${p.line_start}行）`
    }
    if (p.old_string && p.new_string) {
      const oldLen = p.old_string.length
      const newLen = p.new_string.length
      if (oldLen > 20 || newLen > 20) {
        desc += ` | ${oldLen}字节 → ${newLen}字节`
      } else {
        desc += ` | "${p.old_string.slice(0, 30)}" → "${p.new_string.slice(0, 30)}"`
      }
    } else if (p.content) {
      desc += ` | ${p.content.length} 字节`
    }
    return desc
  }
  if (p.path && Array.isArray(p.edits) && p.edits.length > 0) {
    return `${p.path} | ${p.edits.length} 处修改`
  }
  if (p.from) return `${p.from} → ${p.to || '(删除)'}`
  if (p.command) return p.command.slice(0, 120)
  return null
}

const structuredSummary = computed(() => {
  const desc = describeFileOp(props.parsedArgs)
  if (desc) return desc
  const s = props.args || ''
  return s.length > 80 ? s.slice(0, 80) + '…' : s
})
</script>

<style scoped>
.approval-bar {
  display: flex; align-items: flex-start; gap: 6px;
  margin: 4px 8px; padding: 6px 10px;
  background: var(--accent-bg);
  border: 1px solid var(--accent);
  border-radius: var(--border-radius); flex-shrink: 0;
}
.approval-bar-icon { flex-shrink: 0; color: var(--accent); padding-top: 2px; }
.approval-bar-info { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 2px; }
.approval-bar-tool { font-size: 12px; font-weight: 600; color: var(--text-primary); }
.approval-bar-args { font-size: 11px; color: var(--text-secondary); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.approval-bar-reply {
  margin-top: 4px; width: 100%; box-sizing: border-box;
  background: var(--input-bg); border: 1px solid var(--border-color);
  border-radius: 4px; color: var(--text-primary); font-size: 12px;
  padding: 4px 6px; resize: vertical; font-family: inherit;
  min-height: 40px; max-height: 120px;
}
.approval-bar-reply::placeholder { color: var(--text-muted); font-size: 11px; }
.approval-bar-actions { display: flex; gap: 4px; flex-shrink: 0; align-items: flex-start; }
.approval-btn {
  padding: 4px 10px; border: none; border-radius: 3px; font-size: 11px; cursor: pointer;
  transition: opacity 0.15s; white-space: nowrap;
}
.approval-btn:hover { opacity: 0.85; }
.approval-btn-allow { background: #2ea043; color: #fff; }
.approval-btn-deny { background: #da3633; color: #fff; }
.approval-btn-toggle { background: var(--bg-tertiary); color: var(--text-secondary); border: 1px solid var(--border-color); display: flex; align-items: center; gap: 2px; padding: 4px 6px; }
.approval-btn-toggle:hover { background: var(--bg-hover); color: var(--text-primary); }
</style>
