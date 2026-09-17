// ui-art — client 半：注册「画板」插件面板并注入 bundle
//
// 编译产物 assets/art-panel.js（Vite lib IIFE，window.ArtPanel）由
// node scripts/build-ui.mjs --region art 生成（manifest 的 dsh.ui.build 声明）。
// external 共享核心（Vue/api 等）从 window.__PAIRCODE_CORE 取。
//
// 与 ui-voice / ui-music 同样走通用 ui.registerPanel（不动壳、纯加法，卸载插件即消失）。
(ui) => {
  const GLOBAL = 'ArtPanel'
  const JS = 'plugins-assets/ui-art/assets/art-panel.js'
  const CSS = 'plugins-assets/ui-art/assets/art-panel.css'

  // 样式注入（幂等）
  if (!document.querySelector('link[data-art-panel-css]')) {
    const link = document.createElement('link')
    link.rel = 'stylesheet'
    link.href = CSS
    link.setAttribute('data-art-panel-css', '1')
    document.head.appendChild(link)
  }

  // 注册面板（bundle 就绪时立即注册；否则先注入 script，onload 后注册）
  const register = () => {
    const bundle = window[GLOBAL]
    if (!bundle || typeof bundle.mount !== 'function') return false
    ui.registerPanel({
      id: 'art-panel',
      title: '画板',
      icon: 'grid',
      render: (el) => bundle.mount(el),
    })
    // ★ 中间区域视图（2026-09 ui.registerView）：与「对话 / 编辑器 / 市场 / 工具集」
    //   同级的主内容区 tab。open:true → 默认以「后台 tab」形式出现，不抢占对话主视图
    //   （mainTab 仍是 conversation），点 tab 才激活；可与对话并排（「并排对话」按钮）。
    //   与上面的插件面板共用同一 bundle（render 每次调用各建独立 Vue 实例）。
    ui.registerView({
      id: 'art-view',
      title: '画板',
      icon: 'grid',
      order: 10,
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
      ui.reportFailure('render', '画板面板 bundle 未导出 mount：' + JS)
    }
  }
  s.onerror = () => {
    const msg = '画板面板 bundle 加载失败: ' + JS
    console.warn('[ui-art]', msg)
    ui.reportFailure('render', msg)
  }
  document.head.appendChild(s)
}
