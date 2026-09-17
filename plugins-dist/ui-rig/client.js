// ui-rig — client 半：注册「角色」插件面板并注入 bundle
//
// 编译产物 assets/rig-panel.js（Vite lib IIFE，window.RigPanel）由
// node scripts/build-ui.mjs --region rig 生成（manifest 的 dsh.ui.build 声明）。
// external 共享核心（Vue/api 等）从 window.__PAIRCODE_CORE 取。
//
// 与 ui-design / ui-art / ui-music / ui-voice 同样走通用 ui.registerPanel（不动壳、纯加法，卸载插件即消失）。
(ui) => {
  const GLOBAL = 'RigPanel'
  const JS = 'plugins-assets/ui-rig/assets/rig-panel.js'
  const CSS = 'plugins-assets/ui-rig/assets/rig-panel.css'

  // 样式注入（幂等）
  if (!document.querySelector('link[data-rig-panel-css]')) {
    const link = document.createElement('link')
    link.rel = 'stylesheet'
    link.href = CSS
    link.setAttribute('data-rig-panel-css', '1')
    document.head.appendChild(link)
  }

  // 注册面板（bundle 就绪时立即注册；否则先注入 script，onload 后注册）
  const register = () => {
    const bundle = window[GLOBAL]
    if (!bundle || typeof bundle.mount !== 'function') return false
    ui.registerPanel({
      id: 'rig-panel',
      title: '角色',
      icon: 'user',
      render: (el) => bundle.mount(el),
    })
    // ★ 中间区域视图（2026-09 ui.registerView）：主内容区 tab，默认后台打开（不抢对话
    //   主视图），可与对话并排；与插件面板共用同一 bundle。
    ui.registerView({
      id: 'rig-view',
      title: '角色',
      icon: 'user',
      order: 50,
      open: true,
      render: (el) => bundle.mount(el),
    })
    return true
  }

  if (register()) return
  const s = document.createElement('script')
  s.src = JS
  s.onload = () => {
    if (!register()) {
      ui.reportFailure('render', '角色面板 bundle 未导出 mount：' + JS)
    }
  }
  s.onerror = () => {
    const msg = '角色面板 bundle 加载失败: ' + JS
    console.warn('[ui-rig]', msg)
    ui.reportFailure('render', msg)
  }
  document.head.appendChild(s)
}
