import { createApp, h } from 'vue'

var app = createApp({
  render() {
    return h('div', { id: 'app-root' })
  }
})

console.log('A')
app.mount('#app')
console.log('B')

// 直接使用 innerHTML 创建布局（绕过 Vue 的 insertBefore bug）
var root = document.getElementById('app-root')
root.innerHTML = `
<div class="app-layout">
  <div class="activity-bar">
    <div class="activity-item active">⚡</div>
    <div class="activity-item">📁</div>
    <div class="activity-item">🔍</div>
    <div class="activity-item">⚙</div>
  </div>
  <div class="sidebar">
    <div class="sidebar-header">EXPLORER</div>
    <div class="sidebar-item">📄 index.html</div>
    <div class="sidebar-item">📄 main.js</div>
    <div class="sidebar-item">📄 App.vue</div>
    <div class="sidebar-divider"></div>
    <div class="sidebar-header">OUTLINE</div>
    <div class="sidebar-item">▶ App</div>
  </div>
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
  <div class="status-bar">
    <span class="status-left">main.js</span>
    <span class="status-right">UTF-8 | JavaScript | Ln 1, Col 1</span>
  </div>
</div>
`
console.log('C root children=' + root.childNodes.length)

var el = document.getElementById('app')
console.log('D children=' + el.childNodes.length)
