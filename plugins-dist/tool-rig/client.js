// tool-rig — client 半：注册「角色」主内容区视图并注入 bundle
//
// ★ 2026-09-17 合并：本插件原本拆成 tool-rig（工具面）+ ui-rig（UI 面）两个独立
//   发布包，同属一个创作域却分两处维护（版本/文档/发布各一份，且用户看不出「AI 用的
//   工具」和「面板」是同一件事）。现合并为**单包**：host 半 = .iki 角色工程/PSD 绑定
//   内核 + 5 个工具（index.js），client 半 = 本文件，UI bundle = assets/rig-panel.js。
//
// 编译产物 assets/rig-panel.js（Vite lib IIFE，window.RigPanel）由
// node scripts/build-ui.mjs --region rig 生成（manifest 的 dsh.ui.build 声明；
// 区域发现对「同包同时含 dsh.ui + dsh.ui.build」的包生效，与工具面共存无冲突）。
// external 共享核心（Vue/api 等）从 window.__PAIRCODE_CORE 取。
//
// ★ 2026-09-18：不再注册进「插件面板」（ui.registerPanel）—— 那是壳级逃生口的
//   管理/总览区（docs/plugin-development.md §12 分工表），六域是「编辑器式工作台」，
//   主内容区视图（ui.registerView）才是它的位置。此前两处同时注册：插件面板里多一个
//   重复入口，且两处各 mount 一份 Vue 实例（状态不同步）。现只保留 view。
//
// 与其余五域同构：只注册主内容区视图（不动壳、纯加法，卸载插件即消失）。
(ui) => {
  const GLOBAL = 'RigPanel'
  const JS = 'plugins-assets/tool-rig/assets/rig-panel.js'
  const CSS = 'plugins-assets/tool-rig/assets/rig-panel.css'

  // 样式注入（幂等）
  if (!document.querySelector('link[data-rig-panel-css]')) {
    const link = document.createElement('link')
    link.rel = 'stylesheet'
    link.href = CSS
    link.setAttribute('data-rig-panel-css', '1')
    document.head.appendChild(link)
  }

  // 注册视图（bundle 就绪时立即注册；否则先注入 script，onload 后注册）
  const register = () => {
    const bundle = window[GLOBAL]
    if (!bundle || typeof bundle.mount !== 'function') return false
    // ★ 主内容区视图（2026-09 ui.registerView）：主内容区 tab，默认后台打开（不抢对话
    //   主视图），可与对话并排。
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
    console.warn('[tool-rig]', msg)
    ui.reportFailure('render', msg)
  }
  document.head.appendChild(s)
}
