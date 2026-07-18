<script>
import { h } from 'vue'

export default {
  render() {
    return h('div', { class: 'app-layout' }, [
      h('div', { class: 'activity-bar' }, [
        h('div', { class: 'activity-item active' }, '⚡'),
        h('div', { class: 'activity-item' }, '📁'),
        h('div', { class: 'activity-item' }, '🔍'),
        h('div', { class: 'activity-item' }, '⚙'),
      ]),
      h('div', { class: 'sidebar' }, [
        h('div', { class: 'sidebar-header' }, 'EXPLORER'),
        h('div', { class: 'sidebar-item' }, '📄 index.html'),
        h('div', { class: 'sidebar-item' }, '📄 main.js'),
        h('div', { class: 'sidebar-item' }, '📄 App.vue'),
        h('div', { class: 'sidebar-divider' }),
        h('div', { class: 'sidebar-header' }, 'OUTLINE'),
        h('div', { class: 'sidebar-item' }, '▶ ' + this.title),
      ]),
      h('div', { class: 'main-content' }, [
        h('div', { class: 'tab-bar' }, [
          h('div', { class: 'tab active' }, 'App.vue'),
          h('div', { class: 'tab' }, 'main.js'),
        ]),
        h('div', { class: 'editor' }, [
          h('div', { class: 'editor-title' }, this.msg),
          h('div', { class: 'editor-content' }, [
            h('h1', 'hello'),
            h('p', 'Vue 3 桌面端渲染成功！'),
            h('p', { class: 'status-badge' }, '✅ JSC ES6 引擎已完善'),
          ]),
        ]),
      ]),
      h('div', { class: 'status-bar' }, [
        h('span', { class: 'status-left' }, 'main.js'),
        h('span', { class: 'status-right' }, 'UTF-8 | JavaScript | Ln 1, Col 1'),
      ]),
    ])
  },
  data() {
    return {
      title: 'App',
      msg: 'PairCode IDE - Desktop'
    }
  }
}
</script>

<style scoped>
.app-layout { display: flex; width: 100%; height: 100%; background: var(--bg-primary,#0d1117); color: var(--text-primary,#e6edf3); font-family: var(--font-ui,system-ui,sans-serif); font-size: var(--font-size-base,13px); }
.activity-bar { display: flex; flex-direction: column; width: 48px; background: var(--activity-bar-bg,#0d1117); border-right: 1px solid var(--border-color,#30363d); padding-top: 8px; align-items: center; gap: 4px; }
.activity-item { width: 36px; height: 36px; display: flex; align-items: center; justify-content: center; border-radius: 4px; cursor: pointer; color: var(--text-muted,#6e7681); font-size: 16px; }
.activity-item.active { color: var(--accent,#58a6ff); background: var(--accent-bg,rgba(88,166,255,0.10)); }
.sidebar { width: 220px; background: var(--sidebar-bg,#161b22); border-right: 1px solid var(--border-color,#30363d); padding: 8px 0; overflow-y: auto; }
.sidebar-header { padding: 8px 16px 4px; font-size: 12px; color: var(--text-secondary,#8b949e); font-weight: 600; letter-spacing: 1px; }
.sidebar-item { padding: 4px 16px; cursor: pointer; color: var(--text-primary,#e6edf3); font-size: 13px; }
.sidebar-divider { height: 1px; margin: 8px 16px; background: var(--border-color,#30363d); }
.main-content { flex: 1; display: flex; flex-direction: column; overflow: hidden; }
.tab-bar { display: flex; background: var(--tab-inactive-bg,#161b22); border-bottom: 1px solid var(--border-color,#30363d); }
.tab { padding: 8px 20px; cursor: pointer; color: var(--text-secondary,#8b949e); font-size: 12px; border-right: 1px solid var(--border-color,#30363d); }
.tab.active { background: var(--tab-active-bg,#0d1117); color: var(--text-primary,#e6edf3); border-bottom: 2px solid var(--accent,#58a6ff); }
.editor { flex: 1; padding: 24px; overflow-y: auto; }
.editor-title { font-size: 15px; font-weight: 600; margin-bottom: 16px; color: var(--text-primary,#e6edf3); }
.editor-content h1 { font-size: 32px; margin-bottom: 8px; color: var(--accent,#58a6ff); }
.editor-content p { font-size: 13px; line-height: 1.6; color: var(--text-secondary,#8b949e); margin-bottom: 8px; }
.status-badge { display: inline-block; padding: 4px 12px; background: var(--accent-bg,rgba(88,166,255,0.10)); border: 1px solid var(--accent,#58a6ff); border-radius: 4px; color: var(--accent-light,#79c0ff); font-size: 12px; }
.status-bar { display: flex; justify-content: space-between; padding: 4px 12px; background: var(--bg-tertiary,#21262d); border-top: 1px solid var(--border-color,#30363d); color: var(--text-secondary,#8b949e); font-size: 12px; }
.status-left, .status-right { display: flex; align-items: center; gap: 8px; }
</style>
