// tool-voice — client 半：注册「人声」插件面板并注入 bundle
//
// ★ 2026-09-17 合并：本插件原本拆成 tool-voice（工具面，Node 桥）+ ui-voice（UI 面）
//   两个独立发布包，同属一个创作域却分两处维护（版本/文档/发布各一份，且用户看不出
//   「AI 用的工具」和「面板」是同一件事）。现合并为**单包**：host 半 = 纯 DSP 人声链
//   （导入/分析/编辑/渲染/自检，5 个工具 + lib/），client 半 = 本文件，
//   UI bundle = assets/voice-panel.js。
//
// 编译产物 assets/voice-panel.js（Vite lib IIFE，window.VoicePanel）由
// node scripts/build-ui.mjs --region voice 生成（manifest 的 dsh.ui.build 声明；
// 区域发现对「同包同时含 dsh.ui + dsh.ui.build」的包生效，与工具面共存无冲突）。
// external 共享核心（Vue/api 等）从 window.__PAIRCODE_CORE 取。
//
// 与其余五域（art/design/model/music/rig）同包面板同样走通用 ui.registerPanel
// （不动壳、纯加法，卸载插件即消失）。
(ui) => {
  const GLOBAL = 'VoicePanel'
  const JS = 'plugins-assets/tool-voice/assets/voice-panel.js'
  const CSS = 'plugins-assets/tool-voice/assets/voice-panel.css'

  // 样式注入（幂等）
  if (!document.querySelector('link[data-voice-panel-css]')) {
    const link = document.createElement('link')
    link.rel = 'stylesheet'
    link.href = CSS
    link.setAttribute('data-voice-panel-css', '1')
    document.head.appendChild(link)
  }

  // 注册面板（bundle 就绪时立即注册；否则先注入 script，onload 后注册）
  const register = () => {
    const bundle = window[GLOBAL]
    if (!bundle || typeof bundle.mount !== 'function') return false
    ui.registerPanel({
      id: 'voice-panel',
      title: '人声',
      icon: 'sparkles',
      render: (el) => bundle.mount(el),
    })
    // ★ 中间区域视图（2026-09 ui.registerView）：主内容区 tab，默认后台打开（不抢对话
    //   主视图），可与对话并排；与插件面板共用同一 bundle。
    ui.registerView({
      id: 'voice-view',
      title: '人声',
      icon: 'sparkles',
      order: 60,
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
      ui.reportFailure('render', '人声面板 bundle 未导出 mount：' + JS)
    }
  }
  s.onerror = () => {
    const msg = '人声面板 bundle 加载失败: ' + JS
    console.warn('[tool-voice]', msg)
    ui.reportFailure('render', msg)
  }
  document.head.appendChild(s)
}
