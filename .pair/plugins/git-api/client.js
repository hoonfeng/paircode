// git-api — client 半：Git 源代码管理面板（UI 与接口同插件）
//
// 编译产物 assets/git-panel.js（Vite lib IIFE，window.GitPanel）——
// 提前编译（node scripts/build-ui.mjs），运行时动态加载；
// external 共享核心（Vue/状态/api）从 window.__PAIRCODE_CORE 取。
// 面板注册进插件面板「客户端面板」区（PluginPanel）。
(ui) => {
  const GLOBAL = 'GitPanel'
  const JS = 'plugins-assets/git-api/assets/git-panel.js'
  const CSS = 'plugins-assets/git-api/assets/git-panel.css'

  const register = () => {
    ui.registerPanel({
      id: 'git-panel',
      title: 'Git 源代码管理',
      icon: 'source-control',
      render(el) {
        const mod = window[GLOBAL]
        if (!mod || typeof mod.mount !== 'function') {
          el.innerHTML = '<div style="padding:12px;font-size:12px;color:var(--text-muted)">Git 面板 bundle 未就绪</div>'
          return
        }
        try {
          return mod.mount(el)
        } catch (e) {
          console.warn('[git-api] Git 面板挂载失败', e)
          el.innerHTML = '<div style="padding:12px;font-size:12px;color:var(--text-muted)">挂载失败: ' + (e && e.message || e) + '</div>'
        }
      },
    })
  }

  if (window[GLOBAL]) {
    register()
  } else {
    const link = document.createElement('link')
    link.rel = 'stylesheet'
    link.href = CSS
    document.head.appendChild(link)
    const s = document.createElement('script')
    s.src = JS
    s.onload = register
    s.onerror = () => {
      const msg = 'Git 面板 bundle 加载失败: ' + JS
      console.warn('[git-api]', msg)
      ui.reportFailure('render', msg)
    }
    document.head.appendChild(s)
  }
}
