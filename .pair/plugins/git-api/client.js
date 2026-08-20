// git-api — client 半：Git 源代码管理面板 bundle 加载
//
// 编译产物 assets/git-panel.js（Vite lib IIFE，window.GitPanel）——
// 提前编译（node scripts/build-ui.mjs），运行时动态加载；
// external 共享核心（Vue/状态/api）从 window.__PAIRCODE_CORE 取。
//
// ★ 2026-08-20：Git 面板不再注册进插件面板「客户端面板」区（registerPanel 移除），
//   改为活动栏「源代码管理」图标 → 侧边栏独立面板（Sidebar 组件在
//   activeActivity==='source' 时取 window.GitPanel 动态挂载）。
//   本 client 半仅负责注入 bundle（script/css），插件停用即面板消失。
(ui) => {
  const GLOBAL = 'GitPanel'
  const JS = 'plugins-assets/git-api/assets/git-panel.js'
  const CSS = 'plugins-assets/git-api/assets/git-panel.css'

  // 注入样式（幂等）
  if (!document.querySelector('link[data-git-panel-css]')) {
    const link = document.createElement('link')
    link.rel = 'stylesheet'
    link.href = CSS
    link.setAttribute('data-git-panel-css', '1')
    document.head.appendChild(link)
  }

  // bundle 未就绪时注入 script（幂等：onload 后再注入同一 URL 无效）
  if (!window[GLOBAL]) {
    const s = document.createElement('script')
    s.src = JS
    s.onerror = () => {
      const msg = 'Git 面板 bundle 加载失败: ' + JS
      console.warn('[git-api]', msg)
      ui.reportFailure('render', msg)
    }
    document.head.appendChild(s)
  }
}
