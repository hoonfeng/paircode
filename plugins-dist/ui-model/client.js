// ui-model — client 半：注册「3D 模型」插件面板并注入 bundle
//
// 编译产物 assets/model-panel.js（Vite lib IIFE，window.ModelPanel）由
// node scripts/build-ui.mjs --region model 生成（manifest 的 dsh.ui.build 声明）。
// external 共享核心（Vue/api 等）从 window.__PAIRCODE_CORE 取。
//
// 与 ui-rig / ui-design / ui-art / ui-music / ui-voice 同样走通用 ui.registerPanel（不动壳、纯加法，卸载插件即消失）。
(ui) => {
  const GLOBAL = 'ModelPanel'
  const JS = 'plugins-assets/ui-model/assets/model-panel.js'
  const CSS = 'plugins-assets/ui-model/assets/model-panel.css'

  // 样式注入（幂等）
  if (!document.querySelector('link[data-model-panel-css]')) {
    const link = document.createElement('link')
    link.rel = 'stylesheet'
    link.href = CSS
    link.setAttribute('data-model-panel-css', '1')
    document.head.appendChild(link)
  }

  const register = () => {
    const bundle = window[GLOBAL]
    if (!bundle || typeof bundle.mount !== 'function') return false
    ui.registerPanel({
      id: 'model-panel',
      title: '3D 模型',
      icon: 'box',
      render: (el) => bundle.mount(el),
    })
    // ★ 中间区域视图（2026-09 ui.registerView）：主内容区 tab，默认后台打开（不抢对话
    //   主视图），可与对话并排；与插件面板共用同一 bundle。
    ui.registerView({
      id: 'model-view',
      title: '3D 模型',
      icon: 'box',
      order: 30,
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
      ui.reportFailure('render', '3D 模型面板 bundle 未导出 mount：' + JS)
    }
  }
  s.onerror = () => {
    const msg = '3D 模型面板 bundle 加载失败: ' + JS
    console.warn('[ui-model]', msg)
    ui.reportFailure('render', msg)
  }
  document.head.appendChild(s)
}
