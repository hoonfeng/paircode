<template>
  <div v-if="waiting" class="approval-bar">
    <div class="approval-bar-icon"><SvgIcon name="shield" :size="14" /></div>
    <div class="approval-bar-info">
      <span class="approval-bar-tool">{{ toolLabel }}</span>
      <span class="approval-bar-args">{{ structuredSummary }}</span>
    </div>
    <div class="approval-bar-actions">
      <button class="approval-btn approval-btn-allow" @click="$emit('resolve', true)">允许</button>
      <button class="approval-btn approval-btn-deny" @click="$emit('resolve', false)">拒绝</button>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import SvgIcon from './SvgIcon.vue'

const props = defineProps({
  waiting: { type: Boolean, default: false },
  tool: { type: String, default: '' },
  args: { type: String, default: '' },
  parsedArgs: { type: Object, default: () => ({}) },
})
defineEmits(['resolve'])

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
  // edit_file / write_file
  if (p.path) {
    let desc = p.path
    if (p.line_start && p.line_end) {
      desc += `（第${p.line_start}-${p.line_end}行）`
    } else if (p.line_start) {
      desc += `（第${p.line_start}行）`
    }
    // 显示变更概览
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
  // multi_edit
  if (p.path && Array.isArray(p.edits) && p.edits.length > 0) {
    return `${p.path} | ${p.edits.length} 处修改`
  }
  // delete_file / move_file
  if (p.from) return `${p.from} → ${p.to || '(删除)'}`
  // run_command
  if (p.command) return p.command.slice(0, 120)
  return null
}

const structuredSummary = computed(() => {
  const desc = describeFileOp(props.parsedArgs)
  if (desc) return desc
  // fallback: 截断原始 JSON
  const s = props.args || ''
  return s.length > 80 ? s.slice(0, 80) + '…' : s
})
</script>

<style scoped>
.approval-bar {
  display: flex; align-items: center; gap: 6px;
  margin: 4px 8px; padding: 6px 10px;
  background: var(--bg-warning, #fff3cd); border: 1px solid var(--border-warning, #ffc107);
  border-radius: var(--border-radius); flex-shrink: 0;
}
.approval-bar-icon { flex-shrink: 0; color: #cc7b1e; }
.approval-bar-info { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 2px; }
.approval-bar-tool { font-size: 12px; font-weight: 600; color: var(--text-primary); }
.approval-bar-args { font-size: 11px; color: var(--text-secondary); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.approval-bar-actions { display: flex; gap: 4px; flex-shrink: 0; }
.approval-btn {
  padding: 4px 12px; border: none; border-radius: 3px; font-size: 12px; cursor: pointer;
  transition: opacity 0.15s;
}
.approval-btn:hover { opacity: 0.85; }
.approval-btn-allow { background: #2ea043; color: #fff; }
.approval-btn-deny { background: #da3633; color: #fff; }
</style>
