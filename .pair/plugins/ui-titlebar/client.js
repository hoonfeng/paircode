// ═══════════════════════════════════════════════════════════════
// ui-titlebar — client 半：(ui) => void
//
// 按槽位细粒度拆分（2026-08-16）：titlebar 区域独立插件包。
// 编译产物 assets/ui-titlebar.js（Vite lib IIFE，window.UiTitlebar）——
// 提前编译，运行时仅加载执行；external 共享核心（Vue/状态/槽位注册表）
// 从 window.__PAIRCODE_CORE 取（壳 main.js 挂载）。
//
// 时序：壳装载本 client 半 → loadScript bundle → registerSlot('titlebar')
// → 槽位表变化 → 壳 ShellApp 渲染插件到 titlebar 挂载点 → render 调
// window.UiTitlebar.mount(el)。
// render 返回 cleanup（app.unmount），槽位重渲染/插件卸载前宿主调用。
// ═══════════════════════════════════════════════════════════════
(ui) => {
  const GLOBAL = 'UiTitlebar'
  const JS = 'plugins-assets/ui-titlebar/assets/ui-titlebar.js'
  const CSS = 'plugins-assets/ui-titlebar/assets/ui-titlebar.css'

  const register = () => {
    ui.registerSlot({
      slotId: 'titlebar',
      title: '标题栏（ui-titlebar）',
      kind: 'single',
      render(el) {
        const mod = window[GLOBAL]
        if (!mod || typeof mod.mount !== 'function') {
          el.innerHTML = '<div style="padding:8px;font-size:12px;color:var(--text-muted)">titlebar bundle 未就绪</div>'
          return
        }
        try {
          return mod.mount(el)
        } catch (e) {
          console.warn('[ui-titlebar] mount 失败', e)
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
      console.warn('[ui-titlebar]', msg)
      ui.reportFailure('render', msg)
    }
    document.head.appendChild(s)
  }
}