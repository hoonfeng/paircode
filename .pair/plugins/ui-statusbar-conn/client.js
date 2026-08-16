// ═══════════════════════════════════════════════════════════════
// ui-statusbar-conn — client 半：(ui) => void
//
// 迁移来源：web-ui StatusBar.vue 内置「连接状态」指示（connected 圆点 +
// 「已连接/断开」文案）。原实现位于内置模板右侧区；现迁移为磁盘插件，
// 经 statusbar-items 槽位（list 型叠加，状态栏中间区）装配渲染。
//
// 行为与原内置一致：
//   - 启动立即探测 /api/health，之后每 5s 轮询
//   - 监听 window 'ws-connection-change' 事件（App.vue 派发）实时刷新
// render 返回 cleanup：槽位重渲染/插件卸载前宿主调用（清定时器+事件）。
// ═══════════════════════════════════════════════════════════════
(ui) => {
  const SLOT = 'statusbar-items'

  ui.registerSlot({
    slotId: SLOT,
    title: '连接状态（迁移自内置 StatusBar）',
    kind: 'list',
    render(el) {
      el.innerHTML = [
        '<style>',
        '.ui-conn-item{display:inline-flex;align-items:center;gap:5px;font-size:11px;line-height:1;color:var(--status-text,#e5e7eb);white-space:nowrap}',
        '.ui-conn-dot{width:8px;height:8px;border-radius:50%;background:#9ca3af;transition:background .2s}',
        '.ui-conn-dot.on{background:#4ade80;box-shadow:0 0 4px rgba(74,222,128,.7)}',
        '</style>',
        '<span class="ui-conn-item"><span class="ui-conn-dot"></span><span class="ui-conn-text">检测中…</span></span>',
      ].join('')
      const dot = el.querySelector('.ui-conn-dot')
      const text = el.querySelector('.ui-conn-text')
      let timer = null
      let state = null // null=未知 true=已连接 false=断开

      const paint = () => {
        if (state === null) { dot.classList.remove('on'); text.textContent = '检测中…'; return }
        dot.classList.toggle('on', state)
        text.textContent = state ? '已连接' : '断开'
      }
      const check = async () => {
        let ok = false
        try {
          const r = await fetch('/api/health')
          ok = r.ok
        } catch { ok = false }
        state = ok
        paint()
      }
      const onWsChange = (e) => {
        const v = e && e.detail
        if (v && typeof v.connected === 'boolean') { state = v.connected; paint() }
      }

      check()
      timer = setInterval(check, 5000)
      window.addEventListener('ws-connection-change', onWsChange)
      paint()

      // cleanup：槽位重渲染/插件卸载前调用
      return () => {
        if (timer) { clearInterval(timer); timer = null }
        window.removeEventListener('ws-connection-change', onWsChange)
      }
    },
  })
}
