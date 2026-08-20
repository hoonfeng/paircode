// marketplace — client 半：市场面板 bundle 加载
//
// 编译产物 assets/marketplace-panel.js（Vite lib IIFE，window.MarketplacePanel）——
// 提前编译（node scripts/build-ui.mjs），运行时动态加载；
// external 共享核心（Vue/状态/api）从 window.__PAIRCODE_CORE 取。
//
// ★ 2026-08-20：市场功能全插件化——活动栏「市场」图标 → 侧边栏市场面板
//   （Sidebar 组件在 activeActivity==='marketplace' 时取 window.MarketplacePanel
//   动态挂载，与 git-api 的 GitPanel 同模式）。
//   本 client 半仅负责注入 bundle（script/css），插件停用即面板消失。
(ui) => {
  const GLOBAL = 'MarketplacePanel'
  const JS = 'plugins-assets/marketplace/assets/marketplace-panel.js'
  const CSS = 'plugins-assets/marketplace/assets/marketplace-panel.css'

  // 注入样式（幂等）
  if (!document.querySelector('link[data-marketplace-panel-css]')) {
    const link = document.createElement('link')
    link.rel = 'stylesheet'
    link.href = CSS
    link.setAttribute('data-marketplace-panel-css', '1')
    document.head.appendChild(link)
  }

  // bundle 未就绪时注入 script（幂等：onload 后再注入同一 URL 无效）
  if (!window[GLOBAL]) {
    const s = document.createElement('script')
    s.src = JS
    s.onerror = () => {
      const msg = '市场面板 bundle 加载失败: ' + JS
      console.warn('[marketplace]', msg)
      ui.reportFailure('render', msg)
    }
    document.head.appendChild(s)
  }
}
