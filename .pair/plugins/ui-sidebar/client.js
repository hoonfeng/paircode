// ═══════════════════════════════════════════════════════════════
// ui-sidebar — client 半：(ui) => void
//
// 按槽位细粒度拆分（2026-08-16）：sidebar 区域独立插件包。
// 编译产物 assets/ui-sidebar.js（Vite lib IIFE，window.UiSidebar）——
// 提前编译，运行时仅加载执行；external 共享核心（Vue/状态/槽位注册表）
// 从 window.__PAIRCODE_CORE 取（壳 main.js 挂载）。
//
// render 返回 cleanup（app.unmount），槽位重渲染/插件卸载前宿主调用。
// ═══════════════════════════════════════════════════════════════
(ui) => {
  const GLOBAL = 'UiSidebar'
  const JS = 'plugins-assets/ui-sidebar/assets/ui-sidebar.js'
  const CSS = 'plugins-assets/ui-sidebar/assets/ui-sidebar.css'

  const register = () => {
    ui.registerSlot({
      slotId: 'sidebar',
      title: '左侧栏（ui-sidebar）',
      kind: 'single',
      render(el) {
        const mod = window[GLOBAL]
        if (!mod || typeof mod.mount !== 'function') {
          el.innerHTML = '<div style="padding:8px;font-size:12px;color:var(--text-muted)">sidebar bundle 未就绪</div>'
          return
        }
        try {
          return mod.mount(el)
        } catch (e) {
          console.warn('[ui-sidebar] mount 失败', e)
          el.innerHTML = '<div style="padding:8px;font-size:12px;color:var(--text-muted)">挂载失败: ' + (e && e.message || e) + '</div>'
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
      const msg = 'bundle 加载失败: ' + JS
      console.warn('[ui-sidebar]', msg)
      ui.reportFailure('render', msg)
    }
    document.head.appendChild(s)
  }
}
