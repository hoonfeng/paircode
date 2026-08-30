// ═══════════════════════════════════════════════════════════════
// ui-right-panel — client 半：(ui) => void
//
// 按槽位细粒度拆分（2026-08-16）：right-panel 区域独立插件包。
// 编译产物 assets/ui-right-panel.js（Vite lib IIFE，window.UiRightPanel）——
// 提前编译，运行时仅加载执行；external 共享核心（Vue/状态/槽位注册表）
// 从 window.__PAIRCODE_CORE 取（壳 main.js 挂载）。
//
// render 返回 cleanup（app.unmount），槽位重渲染/插件卸载前宿主调用。
// ═══════════════════════════════════════════════════════════════
(ui) => {
  const GLOBAL = 'UiRightPanel'
  const JS = 'plugins-assets/ui-right-panel/assets/ui-right-panel.js'
  const CSS = 'plugins-assets/ui-right-panel/assets/ui-right-panel.css'

  const register = () => {
    ui.registerSlot({
      // ★ 槽位名统一为 conversation（spec §5.2 反向对齐）：manifest dsh.ui.slot
      //   （ui-right-panel/package.json）、壳 host.main.children（ShellApp.vue
      //   useSingleSlot('conversation')）、本运行时 registerSlot slotId 三处一致，
      //   消除三处命名漂移（原 right-panel 名已弃）。
      slotId: 'conversation',
      title: '对话主视图（ui-right-panel）',
      kind: 'single',
      render(el) {
        const mod = window[GLOBAL]
        if (!mod || typeof mod.mount !== 'function') {
          el.innerHTML = '<div style="padding:8px;font-size:12px;color:var(--text-muted)">conversation bundle 未就绪</div>'
          return
        }
        try {
          return mod.mount(el)
        } catch (e) {
          console.warn('[ui-right-panel] mount 失败', e)
          el.innerHTML = '<div style="padding:8px;font-size:12px;color:var(--text-muted)">挂载失败: ' + (e && e.message || e) + '</div>'
        }
      },
    })
  }

  const link = document.createElement('link')
  link.rel = 'stylesheet'
  link.href = CSS
  document.head.appendChild(link)
  if (window[GLOBAL]) {
    register()
  } else {
    const s = document.createElement('script')
    s.src = JS
    s.onload = register
    s.onerror = () => {
      const msg = 'bundle 加载失败: ' + JS
      console.warn('[ui-right-panel]', msg)
      ui.reportFailure('render', msg)
    }
    document.head.appendChild(s)
  }
}