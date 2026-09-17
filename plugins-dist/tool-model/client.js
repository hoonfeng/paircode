// tool-model — client 半：注册「3D 模型」插件面板并注入 bundle
//
// ★ 2026-09-17 合并：本插件原本拆成 tool-model（工具面）+ ui-model（UI 面）两个独立
//   发布包，同属一个创作域却分两处维护（版本/文档/发布各一份，且用户看不出「AI 用的
//   工具」和「面板」是同一件事）。现合并为**单包**：host 半 = 相/网格内核 + 5 个工具
//   （index.js），client 半 = 本文件，UI bundle = assets/model-panel.js。
//
// 编译产物 assets/model-panel.js（Vite lib IIFE，window.ModelPanel）由
// node scripts/build-ui.mjs --region model 生成（manifest 的 dsh.ui.build 声明；
// 区域发现对「同包同时含 dsh.ui + dsh.ui.build」的包生效，与工具面共存无冲突）。
// external 共享核心（Vue/api 等）从 window.__PAIRCODE_CORE 取。
//
// 与其余五域（art/design/music/rig/voice）同包面板同样走通用 ui.registerPanel（不动壳、纯加法，卸载插件即消失）。
//
// 数据面（★ 实时预览链路）：面板经 ui.invoke('tool-model', ...) 调 host 半的
// registerClientMethod：
//   · listArtifacts   → 「识别到的文件」清单（工具写文件即登记 + 工作区扫描，含 mtime/bytes）
//   · readArtifact    → 点选文件的内容（文本原样 / 二进制 base64）供面板即时解析渲染
//   · buildProject    → 点选工程 JSON 即用插件内核现场构建（几何数据回给面板渲染）
// host 半写文件时 ctx.emit('ui:tool-model/artifacts') 广播 → 面板事件驱动刷新（无需手点刷新）。
(ui) => {
  const GLOBAL = 'ModelPanel'
  const JS = 'plugins-assets/tool-model/assets/model-panel.js'
  const CSS = 'plugins-assets/tool-model/assets/model-panel.css'

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
      render: (el) => bundle.mount(el, { fill: false }),
    })
    // ★ 中间区域视图（2026-09 ui.registerView）：主内容区 tab，默认后台打开（不抢对话
    //   主视图），可与对话并排；与插件面板共用同一 bundle。
    ui.registerView({
      id: 'model-view',
      title: '3D 模型',
      icon: 'box',
      order: 30,
      open: true,
      render: (el) => bundle.mount(el, { fill: true }),
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
    console.warn('[tool-model]', msg)
    ui.reportFailure('render', msg)
  }
  document.head.appendChild(s)
}
