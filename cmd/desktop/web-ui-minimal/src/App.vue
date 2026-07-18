<template>
  <div class="app-layout">
    <!-- 活动栏 -->
    <div class="activity-bar">
      <div class="activity-item active">⚡</div>
      <div class="activity-item">📁</div>
      <div class="activity-item">🔍</div>
      <div class="activity-item">⚙</div>
    </div>

    <!-- 侧边栏 -->
    <div class="sidebar">
      <div class="sidebar-header">EXPLORER</div>
      <div class="sidebar-item">📄 index.html</div>
      <div class="sidebar-item">📄 main.js</div>
      <div class="sidebar-item">📄 App.vue</div>
      <div class="sidebar-divider"></div>
      <div class="sidebar-header">OUTLINE</div>
      <div class="sidebar-item">▶ App</div>
    </div>

    <!-- 主内容区 -->
    <div class="main-content">
      <div class="tab-bar">
        <div class="tab active">App.vue</div>
        <div class="tab">main.js</div>
      </div>
      <div class="editor">
        <div class="editor-title">PairCode IDE - Desktop</div>
        <div class="editor-content">
          <h1>hello</h1>
          <p>Vue 3 桌面端渲染成功！</p>
          <p class="status-badge">✅ JSC ES6 引擎已完善</p>
        </div>
      </div>
    </div>

    <!-- 状态栏 -->
    <div class="status-bar">
      <span class="status-left">main.js</span>
      <span class="status-right">UTF-8 | JavaScript | Ln 1, Col 1</span>
    </div>
  </div>
</template>

<style scoped>
.app-layout {
  display: flex;
  width: 100%;
  height: 100%;
  background: var(--bg-primary, #0d1117);
  color: var(--text-primary, #e6edf3);
  font-family: var(--font-ui, system-ui, sans-serif);
  font-size: var(--font-size-base, 13px);
}

/* 活动栏 */
.activity-bar {
  display: flex;
  flex-direction: column;
  width: 48px;
  background: var(--activity-bar-bg, #0d1117);
  border-right: 1px solid var(--border-color, #30363d);
  padding-top: 8px;
  align-items: center;
  gap: 4px;
}
.activity-item {
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--border-radius, 4px);
  cursor: pointer;
  color: var(--text-muted, #6e7681);
  font-size: 16px;
}
.activity-item.active {
  color: var(--accent, #58a6ff);
  background: var(--accent-bg, rgba(88, 166, 255, 0.10));
}
.activity-item:hover {
  background: var(--bg-hover, #1c2333);
}

/* 侧边栏 */
.sidebar {
  width: 220px;
  background: var(--sidebar-bg, #161b22);
  border-right: 1px solid var(--border-color, #30363d);
  padding: 8px 0;
  overflow-y: auto;
}
.sidebar-header {
  padding: 8px 16px 4px;
  font-size: var(--font-size-sm, 12px);
  color: var(--text-secondary, #8b949e);
  font-weight: 600;
  letter-spacing: 1px;
}
.sidebar-item {
  padding: 4px 16px;
  cursor: pointer;
  color: var(--text-primary, #e6edf3);
  font-size: var(--font-size-base, 13px);
}
.sidebar-item:hover {
  background: var(--bg-hover, #1c2333);
}
.sidebar-divider {
  height: 1px;
  margin: 8px 16px;
  background: var(--border-color, #30363d);
}

/* 主内容 */
.main-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.tab-bar {
  display: flex;
  background: var(--tab-inactive-bg, #161b22);
  border-bottom: 1px solid var(--border-color, #30363d);
}
.tab {
  padding: 8px 20px;
  cursor: pointer;
  color: var(--text-secondary, #8b949e);
  font-size: var(--font-size-sm, 12px);
  border-right: 1px solid var(--border-color, #30363d);
}
.tab.active {
  background: var(--tab-active-bg, #0d1117);
  color: var(--text-primary, #e6edf3);
  border-bottom: 2px solid var(--accent, #58a6ff);
}

/* 编辑器 */
.editor {
  flex: 1;
  padding: 24px;
  overflow-y: auto;
}
.editor-title {
  font-size: var(--font-size-lg, 15px);
  font-weight: 600;
  margin-bottom: 16px;
  color: var(--text-primary, #e6edf3);
}
.editor-content h1 {
  font-size: 32px;
  margin-bottom: 8px;
  color: var(--accent, #58a6ff);
}
.editor-content p {
  font-size: var(--font-size-base, 13px);
  line-height: 1.6;
  color: var(--text-secondary, #8b949e);
  margin-bottom: 8px;
}
.status-badge {
  display: inline-block;
  padding: 4px 12px;
  background: var(--accent-bg, rgba(88, 166, 255, 0.10));
  border: 1px solid var(--accent, #58a6ff);
  border-radius: var(--border-radius, 4px);
  color: var(--accent-light, #79c0ff);
  font-size: var(--font-size-sm, 12px);
}

/* 状态栏 */
.status-bar {
  display: flex;
  justify-content: space-between;
  padding: 4px 12px;
  background: var(--bg-tertiary, #21262d);
  border-top: 1px solid var(--border-color, #30363d);
  color: var(--text-secondary, #8b949e);
  font-size: var(--font-size-sm, 12px);
}
.status-left, .status-right {
  display: flex;
  align-items: center;
  gap: 8px;
}
</style>
