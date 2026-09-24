// ═══════════════════════════════════════════════════════════════
// ui-appearance — client 半：(ui) => void
//
// ★ P2「主题 CSS 插件化」（2026-09-25）：
//   8 套主题的完整 CSS 已从壳 plugins-src/ui-app/index.html 迁出，落在本插件包内
//   （theme-<id>.css），经 /plugins-assets/ui-appearance/theme-<id>.css 取用。
//   壳只保留一份 :root 基准令牌兜底 —— 本插件未装配/加载失败时界面不裸奔
//   （视觉回落 Midnight，不会出现无样式的裸 HTML）。
//
// 本 client 半只做一件事：**按当前主题注入/切换 <link>**。
//
// 实现要点（零框架依赖，不改 binding / 存储层）：
//   · 主题 class 由壳 ui-state.js 的 applyTheme() 同时挂在 <html> 与 <body> 上；
//     故 MutationObserver 监听 <html> 的 class 变化即可，无需事件总线。
//   · 旧主题 id（dark / night / light / warm）经别名表映射到同色系的新主题文件，
//     对应 CSS 选择器同时含新旧两个 class（见各 theme-*.css 头部注释）→ 旧 id 行为不变。
//   · <link> 追加到 <head> 末尾，确保在壳内联 <style> 之后 → 同特异性下后者胜出，
//     从而覆盖 :root 兜底。
// ═══════════════════════════════════════════════════════════════
(ui) => {
  const BASE = '/plugins-assets/ui-appearance/'
  const LINK_ID = 'ui-appearance-theme-css'
  // class 名 → 主题文件 id（新 id 一一对应 + 旧 id 别名）
  const THEME_FILE = {
    'theme-midnight': 'midnight', 'theme-dark': 'midnight',
    'theme-graphite': 'graphite',
    'theme-obsidian': 'obsidian', 'theme-night': 'obsidian',
    'theme-aurora': 'aurora',
    'theme-daylight': 'daylight', 'theme-light': 'daylight',
    'theme-paper': 'paper',
    'theme-sand': 'sand', 'theme-warm': 'sand',
    'theme-slate': 'slate',
  }
  let cur = null

  function readThemeId() {
    const cls = document.documentElement.classList
    for (let i = 0; i < cls.length; i++) if (THEME_FILE[cls[i]]) return THEME_FILE[cls[i]]
    return null
  }

  function apply() {
    const id = readThemeId()
    // 壳未挂任何主题 class 时（插件先于 applyTheme 执行）不注入 → :root 兜底生效
    if (!id || id === cur) return
    cur = id
    let el = document.getElementById(LINK_ID)
    if (!el) {
      el = document.createElement('link')
      el.id = LINK_ID
      el.rel = 'stylesheet'
      document.head.appendChild(el)   // 追加到末尾：在壳内联 <style> 之后
    }
    el.href = BASE + 'theme-' + id + '.css'
    // ★ 主题 CSS 是**异步**加载的：在它就绪前，任何「创建时读取 CSS 变量」的组件
    //   （如 TerminalPanel 创建 xterm、VoicePanel/MusicPanel 建 canvas）都会读到壳
    //   :root 兜底值（Midnight 暗色）→ 亮色主题下出现深色块。
    //   故在 link 加载完成后派发事件，供这些组件再刷新一次。
    el.onload = function () {
      try {
        window.dispatchEvent(new CustomEvent('paircode:theme-css-loaded', { detail: { id: id } }))
      } catch (e) { /* 老浏览器无 CustomEvent：跳过，不影响主题生效 */ }
    }
  }

  apply()
  try {
    new MutationObserver(apply).observe(document.documentElement, { attributes: true, attributeFilter: ['class'] })
  } catch (e) { /* 老浏览器无 MutationObserver：仅应用一次，可接受 */ }

  // ══════════════════════════════════════════════════════════════════════
  // 应用背景（P3）：把设置写进壳声明的 --app-bg-* 语义槽位
  // ══════════════════════════════════════════════════════════════════════
  // 设置项走 AppSettings 顶层（binding 直连，见 index.js），故直接从 state.settings 读；
  // 壳只声明槽位与兜底值，本插件负责取值 —— 背景能力可独立演进，壳无需变更。
  // ★ 读取本插件的设置域。背景参数**不加 binding** → 由宿主存入 pluginSettings.<schemaKey>
  //   （见 index.js：key="appearance"）。而 theme 字段带 binding:"theme" → 存 AppSettings 顶层。
  //   两者并存：theme 属「平台既有设置」，背景属「本插件自有设置」。
  function settings() {
    try {
      const st = (window.__PAIRCODE_CORE && window.__PAIRCODE_CORE.uiState.state) || {}
      const ps = (st.settings && st.settings.pluginSettings) || {}
      return Object.assign({}, ps.appearance || {}, ps["ui-appearance"] || {})
    } catch (e) { return {} }
  }

  // ── 对比度工具（sRGB 相对亮度，与 WCAG 2.x 一致）──
  function srgb(c) { c = c / 255; return c <= 0.03928 ? c / 12.92 : Math.pow((c + 0.055) / 1.055, 2.4) }
  function lum(rgb) { return 0.2126 * srgb(rgb[0]) + 0.7152 * srgb(rgb[1]) + 0.0722 * srgb(rgb[2]) }
  function contrast(a, b) {
    const l1 = lum(a), l2 = lum(b)
    return (Math.max(l1, l2) + 0.05) / (Math.min(l1, l2) + 0.05)
  }
  function parseColor(str) {
    if (!str) return null
    const t = String(str).trim()
    let m = t.match(/^#([0-9a-f]{3,8})$/i)
    if (m) {
      let h = m[1]
      if (h.length === 3 || h.length === 4) h = h.slice(0, 3).split("").map(c => c + c).join("")
      return [parseInt(h.slice(0, 2), 16), parseInt(h.slice(2, 4), 16), parseInt(h.slice(4, 6), 16)]
    }
    m = t.match(/^rgba?\(([^)]+)\)$/i)
    if (m) {
      const p = m[1].split(/[,\s\/]+/).filter(Boolean).map(Number)
      if (p.length >= 3 && p.slice(0, 3).every(n => !isNaN(n))) return [p[0], p[1], p[2]]
    }
    return null
  }
  const mix = (a, b, t) => [a[0] + (b[0] - a[0]) * t, a[1] + (b[1] - a[1]) * t, a[2] + (b[2] - a[2]) * t]
  const css = rgb => "rgb(" + rgb.map(v => Math.round(v)).join(", ") + ")"

  // ── 对比度守护（本插件内）──
  // 背景图内容不可预知，故按**最坏情况**求解：分别假设图片为纯白/纯黑，
  // 取与文字色对比度更差的那个作为实际背景，再反推所需的主题色遮罩强度 scrimAlpha，
  // 使最终视觉背景与文字色的对比度仍 ≥ minRatio（正文 4.5，大字 3.0）。
  // 由于 scrimAlpha 越大越接近主题原底色（其自身对比度已达标），该函数单调，可二分求解。
  function computeScrim(fg, themedBg, imgColor, opacity, minRatio) {
    for (let a = 0; a <= 1.0001; a += 0.05) {
      const blended = mix(themedBg, imgColor, opacity)   // 图片按用户不透明度混入主题底色
      const out = mix(blended, themedBg, a)              // 再按 scrimAlpha 拉回主题底色
      if (contrast(fg, out) >= minRatio) return Math.min(1, Number(a.toFixed(2)))
    }
    return 1
  }

  // ── 强调色覆盖（设计稿屏 4）──
  // 只覆盖 accent 系列槽位，不动整套色板；「默认」= 撤除覆盖回到主题自带值。
  // 亮暗各一档：同一色相在亮底上需更深才有 4.5:1（与状态色双档规则同理）。
  const ACCENTS = {
    teal:    { dark: "#22B8A6", light: "#0A6E7A" },
    amber:   { dark: "#D8A03C", light: "#8A5A10" },
    crimson: { dark: "#D9534F", light: "#A32B28" },
    violet:  { dark: "#8B6CF0", light: "#5B3FC4" },
  }
  const ACCENT_VARS = ["--color-accent", "--color-accent-soft", "--color-accent-bg", "--color-accent-ring", "--color-accent-fg"]
  const DARK_THEME_IDS = ["midnight", "graphite", "obsidian", "aurora"]
  function isDarkNow() { return DARK_THEME_IDS.indexOf(readThemeId()) >= 0 }

  // ── 填充方式映射（设计稿屏 5）──
  const FIT = {
    cover:   { size: "cover",   position: "center center", repeat: "no-repeat" },
    contain: { size: "contain", position: "center center", repeat: "no-repeat" },
    repeat:  { size: "auto",    position: "left top",      repeat: "repeat" },
    center:  { size: "auto",    position: "center center", repeat: "no-repeat" },
  }

  const BG_VARS = ["--app-bg-image", "--app-bg-color", "--app-bg-opacity", "--app-bg-size", "--app-bg-position", "--app-bg-repeat", "--app-bg-blur", "--app-bg-scrim", "--app-surface-alpha"]
  function clearBg() { for (const v of BG_VARS) document.documentElement.style.removeProperty(v) }

  let lastBgKey = ""
  function applyBackground() {
    const st = settings()
    const src = st.appBgSource || "none"
    const opRaw = st.appBgOpacity === undefined || st.appBgOpacity === null ? 35 : Number(st.appBgOpacity)
    const opacity = Math.max(0, Math.min(1, (isNaN(opRaw) ? 35 : opRaw) / 100))
    const blur = Math.max(0, Math.min(40, Number(st.appBgBlur) || 0))
    const dim = Math.max(-50, Math.min(60, Number(st.appBgDim) || 0))
    const fit = FIT[st.appBgFit] || FIT.cover
    // ★ 每主题独立背景：开启后亮色主题用 appBgImageLight，暗色主题用 appBgImage
    const perTheme = !!st.appBgPerTheme
    const imgDark = String(st.appBgImage || "").trim()
    const imgLight = String(st.appBgImageLight || "").trim()
    const useImg = (perTheme && !isDarkNow() && imgLight) ? imgLight : imgDark
    const colorBg = String(st.appBgColor || "").trim()
    const key = [src, opacity, blur, dim, st.appBgFit || "", useImg, colorBg, perTheme].join("|")
    if (key === lastBgKey) return
    lastBgKey = key
    const root = document.documentElement
    if (src === "none" || opacity === 0) { clearBg(); return }

    const de = getComputedStyle(root)
    const fg = parseColor(de.getPropertyValue("--color-fg").trim()) || [231, 236, 245]
    const bg = parseColor(de.getPropertyValue("--color-bg").trim()) || [15, 20, 32]

    let image = "none"
    let flatColor = "transparent"
    if (src === "image") {
      if (useImg) image = "url(\"" + useImg.replace(/"/g, "%22") + "\")"
    } else if (src === "color") {
      flatColor = colorBg || "#0F1420"
    }
    if (src === "gradient" || (src === "image" && image === "none")) {
      // 主题渐变：用 accent 与 cat-teal 造柔和氛围，颜色取自令牌 → 跟随主题
      const a1 = de.getPropertyValue("--color-accent").trim() || "#6FA8FF"
      const a2 = de.getPropertyValue("--color-cat-teal").trim() || "#3FD0E0"
      image = "radial-gradient(1200px 600px at 15% 0%, " + a1 + " 0%, transparent 60%), radial-gradient(1000px 500px at 85% 100%, " + a2 + " 0%, transparent 55%)"
    }

    // ★ 守护：图片按最坏情况（纯白/纯黑）取更差者
    const white = [255, 255, 255], black = [0, 0, 0]
    const worst = contrast(fg, mix(bg, white, opacity)) <= contrast(fg, mix(bg, black, opacity)) ? white : black
    const scrimAlpha = computeScrim(fg, bg, worst, opacity, 4.5)

    root.style.setProperty("--app-bg-image", image)
    root.style.setProperty("--app-bg-color", flatColor)
    root.style.setProperty("--app-bg-opacity", String(opacity))
    root.style.setProperty("--app-bg-size", fit.size)
    root.style.setProperty("--app-bg-position", fit.position)
    root.style.setProperty("--app-bg-repeat", fit.repeat)
    // ★ 模糊与暗化/提亮共用一个 filter（壳：#app-bg { filter: var(--app-bg-blur) }）：
    //   brightness(<1) 压暗、>1 提亮；dim=0 → 1（无视觉影响，向后兼容）。
    root.style.setProperty("--app-bg-blur",
      (blur > 0 ? "blur(" + blur + "px) " : "") + "brightness(" + (1 - dim / 100).toFixed(3) + ")")
    // scrim 叠在图之上（壳的 #app-bg 里 #app-bg::after 承担），强度由守护算出
    root.style.setProperty("--app-bg-scrim", "rgba(" + bg.map(v => Math.round(v)).join(", ") + ", " + scrimAlpha + ")")
    // ★ 面板不透明度（玻璃化，设计稿屏 3/5）：壳把主要面板别名用 color-mix 绑定到该槽位
    //   （alpha=1 时结果等于原色 → 未启用背景时视觉零变化）。
    const sa = Math.max(0.4, Math.min(1, (Number(st.appSurfaceAlpha) || 100) / 100))
    root.style.setProperty("--app-surface-alpha", String(sa))
  }

  // ══ 强调色覆盖（设计稿屏 4）══
  function applyAccent() {
    const st = settings()
    const id = st.accentOverride || "default"
    const root = document.documentElement
    if (id === "default" || !ACCENTS[id]) {
      for (const v of ACCENT_VARS) root.style.removeProperty(v)
      return
    }
    const c = ACCENTS[id][isDarkNow() ? "dark" : "light"]
    const rgb = parseColor(c) || [111, 168, 255]
    const base = rgb.map(v => Math.round(v)).join(", ")
    root.style.setProperty("--color-accent", c)
    // accent-fg：按对比度选黑/白，保证按钮文字仍可读
    root.style.setProperty("--color-accent-fg", contrast([255, 255, 255], rgb) >= 4.5 ? "#ffffff" : "#0b0d10")
    root.style.setProperty("--color-accent-soft", "rgba(" + base + ", 0.16)")
    root.style.setProperty("--color-accent-bg", "rgba(" + base + ", 0.12)")
    root.style.setProperty("--color-accent-ring", "rgba(" + base + ", 0.45)")
  }

  // ══ 跟随系统（设计稿屏 4）══
  // 开启后按 prefers-color-scheme 在 Midnight / Daylight 间切换。与壳 applyTheme 同行为
  // （class 同时挂 html 与 body），并写回前端状态与持久化，避免设置面板显示与视觉不一致。
  function systemThemeID() {
    try { return window.matchMedia("(prefers-color-scheme: light)").matches ? "daylight" : "midnight" } catch (e) { return "midnight" }
  }
  function applyFollowSystem() {
    const st = settings()
    if (!st.themeFollowSystem) return
    const want = systemThemeID()
    if (readThemeId() === want) return
    document.documentElement.className = "theme-" + want
    document.body.className = "theme-" + want
    try {
      const s = window.__PAIRCODE_CORE && window.__PAIRCODE_CORE.uiState && window.__PAIRCODE_CORE.uiState.state
      if (s) s.theme = want
      const raw = localStorage.getItem("paircode-ide-state")
      if (raw) { const o = JSON.parse(raw) || {}; o.theme = want; localStorage.setItem("paircode-ide-state", JSON.stringify(o)) }
    } catch (e) { }
  }

  function tick() { applyFollowSystem(); applyBackground(); applyAccent() }

  tick()
  setInterval(tick, 1200)   // 设置面板改动后自动生效
  // 主题切换会改变 fg/bg 基准 → 重新计算守护强度
  window.addEventListener("paircode:theme-css-loaded", function () { applyBackground(); applyAccent() })
  try { new MutationObserver(function () { applyBackground(); applyAccent() }).observe(document.documentElement, { attributes: true, attributeFilter: ["class", "data-theme"] }) } catch (e) { }
  // 系统明暗偏好变化（跟随系统开启时即时切换）
  try { window.matchMedia("(prefers-color-scheme: dark)").addEventListener("change", function () { applyFollowSystem() }) } catch (e) { }
}
