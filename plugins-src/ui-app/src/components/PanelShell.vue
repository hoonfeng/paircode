<!--
  PanelShell.vue — 创作域面板统一外壳（六域共用：画板 / 设计 / 3D 模型 / 音乐 / 角色 / 人声）

  结构（七屏逐字一致，对应 design.project.json 的设计稿）：
    ┌ 顶栏 ─ 域图标 · 域名称 · 工程文件输入 · 模式徽章 · 动作 · 刷新
    ├ 指标条 ─ 关键数字横排（不必滚动去找）
    ├ 主体 ─ 左栏（分段切换 + 清单）+ 主区（预览与详情）
    └ 底栏 ─ 只读说明 · 真相源文件名

  为什么抽成外壳：六域面板此前各写各的、信息一律垂直堆叠，用户得滚很久才看完；
  统一外壳后「一眼看全 + 左栏可切换 + 主区给预览」，且六个域的形态一致、一眼可辨。

  样式说明：本组件的 <style> **故意不加 scoped** —— 插槽内容（左栏/主区）由各面板提供，
  scoped 样式作用不到插槽，故把 .pn-* 基础类做成共享类（前缀唯一，无外溢风险）；
  各面板仍可用自己的 scoped 样式补充局部细节。配色一律取设计系统变量，不另起一套。
-->
<template>
  <div class="pn">
    <div class="pn-head">
      <SvgIcon v-if="icon" :name="icon" :size="13" class="pn-head-icon" />
      <span class="pn-head-title">{{ title }}</span>
      <input
        v-if="file !== null && file !== undefined"
        :value="file"
        class="pn-head-file"
        :placeholder="placeholder"
        spellcheck="false"
        @input="$emit('update:file', $event.target.value)"
        @keyup.enter="$emit('reload')"
      />
      <span v-else class="pn-spacer"></span>
      <span v-if="badge" class="pn-badge" :class="'pn-badge-' + badgeTone">{{ badge }}</span>
      <slot name="head-actions"></slot>
      <button
        v-if="reloadable"
        class="pn-icon-btn"
        :disabled="loading"
        :title="reloadTitle"
        @click="$emit('reload')"
      >
        <SvgIcon name="refresh" :size="12" :class="{ 'pn-spin': loading }" />
      </button>
    </div>

    <div v-if="error" class="pn-msg pn-msg-err">{{ error }}</div>
    <div v-else-if="notice" class="pn-msg">{{ notice }}</div>

    <div v-if="!empty && metrics.length" class="pn-metrics">
      <div v-for="(m, i) in metrics" :key="i" class="pn-metric" :title="m.title || ''">
        <span class="pn-metric-k">{{ m.k }}</span>
        <b class="pn-metric-v" :class="{ 'pn-mono': m.mono }">{{ m.v }}</b>
      </div>
      <span class="pn-spacer"></span>
      <slot name="metrics-tail"></slot>
    </div>

    <div class="pn-body">
      <div v-if="empty" class="pn-empty"><div class="pn-empty-guide"><slot name="empty"></slot></div></div>
      <template v-else>
        <aside class="pn-side">
          <div v-if="$slots.segments" class="pn-seg"><slot name="segments"></slot></div>
          <div class="pn-scroll pn-side-scroll"><slot name="side"></slot></div>
        </aside>
        <div class="pn-main"><slot name="main"></slot></div>
      </template>
    </div>

    <div class="pn-foot">
      <slot name="foot"><span class="pn-foot-text">{{ footnote }}</span></slot>
      <span class="pn-spacer"></span>
      <span v-if="source" class="pn-foot-src" :title="source">{{ source }}</span>
    </div>
  </div>
</template>

<script setup>
import SvgIcon from './SvgIcon.vue'

defineProps({
  title: { type: String, required: true },
  icon: { type: String, default: '' },
  file: { type: String, default: null }, // 传了才显示文件名输入框（null = 该面板无工程文件）
  placeholder: { type: String, default: '' },
  badge: { type: String, default: '' },
  badgeTone: { type: String, default: 'muted' }, // muted | accent | ok | bad
  metrics: { type: Array, default: () => [] }, // [{k,v,mono?,title?}]
  loading: { type: Boolean, default: false },
  error: { type: String, default: '' },
  notice: { type: String, default: '' },
  empty: { type: Boolean, default: false },
  reloadable: { type: Boolean, default: true },
  reloadTitle: { type: String, default: '重新载入' },
  footnote: { type: String, default: '' },
  source: { type: String, default: '' },
})
defineEmits(['update:file', 'reload'])
</script>

<style>
/* ── 外壳 ── */
.pn {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
  font-size: 12px;
  color: var(--text-primary);
  box-sizing: border-box;
}
.pn-spacer { flex: 1; }

/* 顶栏 */
.pn-head {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 8px;
  flex: none;
  border-bottom: 1px solid var(--border-color);
}
.pn-head-icon { color: var(--text-muted); flex: none; }
.pn-head-title { font-weight: 600; flex: none; }
.pn-head-file {
  flex: 1;
  min-width: 0;
  padding: 3px 6px;
  font-size: 11px;
  font-family: ui-monospace, Consolas, monospace;
  color: var(--text-secondary);
  background: var(--bg-primary);
  border: 1px solid var(--border-color);
  border-radius: 4px;
}
.pn-icon-btn {
  display: flex;
  align-items: center;
  padding: 3px 6px;
  color: var(--text-secondary);
  background: transparent;
  border: 1px solid var(--border-color);
  border-radius: 4px;
  cursor: pointer;
  flex: none;
}
.pn-icon-btn:hover:not(:disabled) { background: var(--bg-hover); }
.pn-icon-btn:disabled { opacity: 0.5; cursor: default; }
.pn-spin { animation: pn-spin 1s linear infinite; }
@keyframes pn-spin { to { transform: rotate(360deg); } }

/* 消息条 */
.pn-msg { flex: none; padding: 6px 8px; color: var(--text-muted); line-height: 1.6; }
.pn-msg-err { color: var(--text-primary); border-left: 2px solid var(--accent); background: var(--bg-tertiary); }

/* 指标条 */
.pn-metrics {
  display: flex;
  align-items: baseline;
  gap: 16px;
  padding: 6px 8px;
  flex: none;
  border-bottom: 1px solid var(--border-color);
  overflow-x: auto;
}
.pn-metric { display: flex; align-items: baseline; gap: 5px; white-space: nowrap; flex: none; }
.pn-metric-k { color: var(--text-muted); font-size: 11px; }
.pn-metric-v { font-weight: 600; }

/* 主体：左栏 + 主区 */
.pn-body { flex: 1; display: flex; min-height: 0; }
.pn-side {
  width: 264px;
  flex: none;
  display: flex;
  flex-direction: column;
  min-height: 0;
  border-right: 1px solid var(--border-color);
}
.pn-seg { display: flex; gap: 2px; padding: 6px 6px 0; flex: none; }
.pn-scroll { flex: 1; min-height: 0; overflow-y: auto; overflow-x: hidden; }
.pn-side-scroll { padding: 8px 6px; }
.pn-main { flex: 1; min-width: 0; display: flex; flex-direction: column; min-height: 0; }
.pn-empty { flex: 1; display: flex; align-items: center; justify-content: center; padding: 16px; }
.pn-empty-guide { max-width: 420px; display: flex; flex-direction: column; gap: 10px; }
.pn-empty-title { font-size: 14px; font-weight: 600; color: var(--text-primary); }
.pn-empty-step { display: flex; gap: 8px; align-items: baseline; font-size: 11px; line-height: 1.6; }
.pn-empty-step > i {
  flex: none;
  width: 16px;
  height: 16px;
  font-style: normal;
  font-size: 10px;
  line-height: 16px;
  text-align: center;
  color: var(--text-muted);
  border: 1px solid var(--border-color);
  border-radius: 50%;
  margin-top: 1px;
}
.pn-empty-step > span { color: var(--text-secondary); min-width: 0; }

/* 底栏 */
.pn-foot {
  flex: none;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 5px 8px;
  border-top: 1px solid var(--border-color);
  font-size: 11px;
  color: var(--text-muted);
}
.pn-foot-text { line-height: 1.5; }
.pn-foot-src { font-family: ui-monospace, Consolas, monospace; flex: none; }

/* ── 共用类（供各面板插槽内容使用） ── */

/* 徽章 */
.pn-badge {
  flex: none;
  padding: 1px 7px;
  border-radius: 9px;
  font-size: 10px;
  line-height: 1.6;
  border: 1px solid var(--border-color);
  color: var(--text-secondary);
  white-space: nowrap;
}
.pn-badge-accent { color: var(--bg-primary); background: var(--accent); border-color: var(--accent); }
.pn-badge-ok { color: var(--bg-primary); background: var(--accent); border-color: var(--accent); }
.pn-badge-bad { color: var(--text-primary); background: var(--bg-hover); }

/* 左栏分段按钮 */
.pn-tab {
  flex: 1;
  padding: 4px 6px;
  font-size: 11px;
  color: var(--text-muted);
  background: transparent;
  border: 1px solid transparent;
  border-radius: 4px;
  cursor: pointer;
  white-space: nowrap;
}
.pn-tab:hover { background: var(--bg-hover); }
.pn-tab-on { color: var(--text-primary); background: var(--bg-tertiary); border-color: var(--border-color); }

/* 按钮 */
.pn-btn {
  padding: 3px 10px;
  font-size: 11px;
  color: var(--text-secondary);
  background: transparent;
  border: 1px solid var(--border-color);
  border-radius: 4px;
  cursor: pointer;
  white-space: nowrap;
}
.pn-btn:hover:not(:disabled) { background: var(--bg-hover); color: var(--text-primary); }
.pn-btn:disabled { opacity: 0.5; cursor: default; }
.pn-btn-on { color: var(--text-primary); background: var(--bg-tertiary); }
.pn-row-gap { display: flex; align-items: center; gap: 6px; }

/* 区块卡 */
.pn-card {
  display: flex;
  flex-direction: column;
  gap: 5px;
  padding: 8px;
  background: var(--bg-tertiary);
  border: 1px solid var(--border-color);
  border-radius: 6px;
}
.pn-card + .pn-card { margin-top: 8px; }
.pn-card-title { display: flex; align-items: center; gap: 6px; font-weight: 600; color: var(--text-secondary); }
.pn-card-plain { background: transparent; border-color: transparent; padding: 0; }
.pn-card-plain + .pn-card-plain { margin-top: 12px; }

/* 键值行（左对齐紧凑，不两端拉开） */
.pn-kv { display: flex; gap: 8px; align-items: baseline; }
.pn-kv > span { color: var(--text-muted); flex: none; }
.pn-kv > b { font-weight: 500; min-width: 0; overflow-wrap: anywhere; }

/* 列表 */
.pn-list { display: flex; flex-direction: column; }
.pn-item {
  display: flex;
  align-items: center;
  gap: 6px;
  width: 100%;
  padding: 4px 6px;
  text-align: left;
  font-size: 11px;
  color: var(--text-secondary);
  background: transparent;
  border: none;
  border-radius: 4px;
  cursor: pointer;
}
.pn-item:hover { background: var(--bg-hover); }
.pn-item-on { color: var(--text-primary); background: var(--bg-hover); outline: 1px solid var(--accent); }
.pn-item-k { font-family: ui-monospace, Consolas, monospace; flex: none; }
.pn-item-v { margin-left: auto; color: var(--text-muted); flex: none; }

/* 文本工具类 */
.pn-dim { color: var(--text-muted); font-weight: 400; }
.pn-mini { font-size: 10px; }
.pn-mono { font-family: ui-monospace, Consolas, monospace; }
.pn-strong { color: var(--text-primary); font-weight: 600; }
.pn-warn { color: var(--text-primary); }
.pn-tip { color: var(--text-muted); line-height: 1.6; font-size: 11px; }
.pn-tip code {
  padding: 0 3px;
  font-family: ui-monospace, Consolas, monospace;
  font-size: 10px;
  background: var(--bg-primary);
  border: 1px solid var(--border-color);
  border-radius: 3px;
}

/* 校验行（点 + 名称 + 指标） */
.pn-check { display: flex; align-items: center; gap: 6px; font-size: 11px; }
.pn-check > b { flex: none; color: var(--text-secondary); }
.pn-check-name { flex: none; color: var(--text-secondary); }
.pn-check-metric { flex: 1; text-align: right; color: var(--text-muted); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.pn-dot { width: 7px; height: 7px; border-radius: 50%; flex: none; display: inline-block; background: var(--text-muted); }
.pn-dot-ok { background: var(--accent); }

/* 色块 */
.pn-swatch {
  display: inline-block;
  width: 10px;
  height: 10px;
  border: 1px solid var(--border-color);
  border-radius: 2px;
  vertical-align: middle;
  margin-right: 4px;
  flex: none;
}

/* 代码块 */
.pn-code {
  margin: 0;
  padding: 6px;
  font-family: ui-monospace, Consolas, monospace;
  font-size: 10px;
  line-height: 1.5;
  color: var(--text-secondary);
  background: var(--bg-primary);
  border: 1px solid var(--border-color);
  border-radius: 4px;
  overflow-x: auto;
  white-space: pre;
}

/* 表格（紧凑） */
.pn-table { width: 100%; border-collapse: collapse; font-size: 11px; }
.pn-table th { color: var(--text-muted); font-weight: 500; text-align: left; padding: 3px 4px; border-bottom: 1px solid var(--border-color); }
.pn-table td { padding: 3px 4px; border-bottom: 1px solid var(--bg-hover); }
.pn-table tbody tr { cursor: pointer; }
.pn-table tbody tr:hover { background: var(--bg-hover); }

/* 预览区 */
.pn-preview {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 120px;
  padding: 8px;
  background: var(--bg-primary);
  border: 1px solid var(--border-color);
  border-radius: 6px;
  overflow: hidden;
}
.pn-preview-scroll { overflow: auto; display: block; }
</style>
