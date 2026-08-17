(function() {
  "use strict";
  var __vite_style__ = document.createElement("style");
  __vite_style__.textContent = "\n@keyframes stopPulse-faf69761 {\n0%, 100% { opacity: 0.6;\n}\n50% { opacity: 1;\n}\n}\n@keyframes stopRingPulse-faf69761 {\n0%, 100% { opacity: 0.3; transform: scale(1);\n}\n50% { opacity: 0.15; transform: scale(1.15);\n}\n}\n.stop-pulse[data-v-faf69761] {\n  animation: stopPulse-faf69761 1.2s ease-in-out infinite;\n}\n.stop-pulse-ring[data-v-faf69761] {\n  animation: stopRingPulse-faf69761 1.2s ease-in-out infinite;\n  fill: none;\n  transform-origin: center;\n}\n.svg-icon[data-v-faf69761] {\n  display: inline-block;\n  vertical-align: middle;\n  flex-shrink: 0;\n}\n\n.plugin-panel[data-v-5c15f6cc] {\n  height: 100%;\n  display: flex;\n  flex-direction: column;\n  overflow: hidden;\n  font-size: 13px;\n}\n.pp-header[data-v-5c15f6cc] {\n  display: flex;\n  align-items: center;\n  justify-content: space-between;\n  padding: 8px 10px;\n  border-bottom: 1px solid var(--border-color);\n  flex-shrink: 0;\n}\n.pp-title[data-v-5c15f6cc] {\n  display: flex;\n  align-items: center;\n  gap: 6px;\n  font-weight: 600;\n  font-size: 12px;\n  color: var(--text-primary);\n}\n.pp-actions[data-v-5c15f6cc] { display: flex; gap: 4px;\n}\n.pp-icon-btn[data-v-5c15f6cc] {\n  background: none; border: none; cursor: pointer;\n  color: var(--text-muted); padding: 2px 4px; border-radius: 3px;\n  display: flex; align-items: center;\n}\n.pp-icon-btn[data-v-5c15f6cc]:hover { background: var(--bg-hover); color: var(--text-primary);\n}\n\n/* 新建表单 */\n.pp-new[data-v-5c15f6cc] {\n  padding: 10px;\n  border-bottom: 1px solid var(--border-color);\n  display: flex;\n  flex-direction: column;\n  gap: 6px;\n  background: var(--bg-tertiary);\n  flex-shrink: 0;\n  max-height: 45%;\n  overflow: auto;\n}\n.pp-new-title[data-v-5c15f6cc] { font-size: 12px; font-weight: 600; color: var(--text-secondary);\n}\n.pp-input[data-v-5c15f6cc], .pp-textarea[data-v-5c15f6cc] {\n  background: var(--bg-primary);\n  border: 1px solid var(--border-color);\n  color: var(--text-primary);\n  border-radius: 4px;\n  padding: 5px 8px;\n  font-size: 12px;\n  width: 100%;\n  box-sizing: border-box;\n}\n.pp-textarea.code[data-v-5c15f6cc] {\n  font-family: var(--font-code);\n  font-size: 11px;\n  line-height: 1.5;\n  resize: vertical;\n}\n.pp-new-foot[data-v-5c15f6cc] { display: flex; align-items: center; gap: 8px;\n}\n.pp-lang[data-v-5c15f6cc] { width: auto; flex-shrink: 0;\n}\n.pp-check[data-v-5c15f6cc] { display: flex; align-items: center; gap: 4px; font-size: 11px; color: var(--text-secondary); white-space: nowrap;\n}\n.pp-new-msg[data-v-5c15f6cc] { font-size: 11px; color: var(--accent-light); word-break: break-all;\n}\n.pp-new-msg.err[data-v-5c15f6cc] { color: var(--error, #e06c75);\n}\n\n/* client 面板区 */\n.pp-client[data-v-5c15f6cc] {\n  border-bottom: 1px solid var(--border-color);\n  flex-shrink: 0;\n  display: flex;\n  flex-direction: column;\n  max-height: 40%;\n}\n/* UI 槽位区（Slot 系统） */\n.pp-slots[data-v-5c15f6cc] {\n  border-bottom: 1px solid var(--border-color);\n  flex-shrink: 0;\n  padding: 6px 8px;\n  display: flex;\n  flex-direction: column;\n  gap: 5px;\n  max-height: 30%;\n  overflow-y: auto;\n}\n.pp-slots-head[data-v-5c15f6cc] {\n  display: flex; align-items: center; gap: 6px;\n  font-size: 11px; color: var(--text-primary); font-weight: 600;\n}\n.pp-slots-head svg[data-v-5c15f6cc] { color: var(--accent);\n}\n.pp-slots-sub[data-v-5c15f6cc] { font-weight: 400; font-size: 10px; color: var(--text-muted);\n}\n.pp-slot-row[data-v-5c15f6cc] {\n  display: flex; align-items: center; justify-content: space-between; gap: 8px;\n  background: var(--bg-primary);\n  border: 1px solid var(--border-color);\n  border-left: 2px solid var(--accent);\n  border-radius: 6px; padding: 5px 8px;\n  transition: border-color .12s, background .12s;\n}\n.pp-slot-row[data-v-5c15f6cc]:hover {\n  border-color: color-mix(in srgb, var(--accent) 45%, var(--border-color));\n  background: var(--bg-hover);\n}\n.pp-slot-info[data-v-5c15f6cc] { display: flex; flex-direction: column; gap: 2px; min-width: 0;\n}\n.pp-slot-title-row[data-v-5c15f6cc] { display: flex; align-items: center; gap: 6px; min-width: 0;\n}\n.pp-slot-id[data-v-5c15f6cc] {\n  font-family: var(--font-mono, monospace); font-size: 11px; color: var(--accent-light);\n  text-transform: uppercase; letter-spacing: 0.5px; font-weight: 600;\n}\n.pp-slot-owner[data-v-5c15f6cc] { font-size: 10px; color: var(--accent); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; max-width: 180px;\n}\n.pp-slot-owner.builtin[data-v-5c15f6cc] { color: var(--text-muted);\n}\n.pp-slot-kind[data-v-5c15f6cc] {\n  font-size: 9px; border-radius: 4px; padding: 0 5px; align-self: flex-start;\n  line-height: 15px; flex-shrink: 0; font-weight: 600; letter-spacing: .3px;\n}\n.pp-slot-kind.kind-single[data-v-5c15f6cc] { color: var(--accent-light); background: color-mix(in srgb, var(--accent) 14%, transparent); border: 1px solid color-mix(in srgb, var(--accent) 35%, transparent);\n}\n.pp-slot-kind.kind-list[data-v-5c15f6cc] { color: #3fb950; background: rgba(63, 185, 80, .10); border: 1px solid rgba(63, 185, 80, .30);\n}\n.pp-slot-list[data-v-5c15f6cc] { display: flex; flex-direction: column; gap: 3px; align-items: flex-end; flex-shrink: 0;\n}\n.pp-slot-list-item[data-v-5c15f6cc] {\n  display: flex; align-items: center; gap: 4px; font-size: 10px; color: var(--text-secondary);\n  cursor: pointer; max-width: 220px; padding: 1px 4px; border-radius: 4px;\n  transition: background .1s, color .1s;\n}\n.pp-slot-list-item[data-v-5c15f6cc]:hover { color: var(--text-primary); background: var(--bg-hover);\n}\n.pp-slot-list-item input[type='checkbox'][data-v-5c15f6cc] { accent-color: var(--accent); margin: 0;\n}\n.pp-slot-list-item span[data-v-5c15f6cc] { overflow: hidden; text-overflow: ellipsis; white-space: nowrap;\n}\n.pp-slot-empty[data-v-5c15f6cc] { font-size: 10px; color: var(--text-muted);\n}\n.pp-slot-select[data-v-5c15f6cc] {\n  width: 170px; font-size: 11px; padding: 3px 6px;\n  background: var(--bg-secondary); color: var(--text-primary);\n  border: 1px solid var(--border-color); border-radius: 5px; flex-shrink: 0;\n  cursor: pointer; transition: border-color .12s, box-shadow .12s;\n}\n.pp-slot-select[data-v-5c15f6cc]:hover { border-color: var(--accent);\n}\n.pp-slot-select[data-v-5c15f6cc]:focus { outline: none; border-color: var(--accent); box-shadow: 0 0 0 2px var(--focus-ring);\n}\n.pp-client-tabs[data-v-5c15f6cc] {\n  display: flex;\n  gap: 2px;\n  padding: 4px 8px 0;\n  border-bottom: 1px solid var(--border-color);\n  overflow-x: auto;\n}\n.pp-client-tab[data-v-5c15f6cc] {\n  display: flex; align-items: center; gap: 4px;\n  padding: 4px 10px;\n  font-size: 11px;\n  color: var(--text-secondary);\n  cursor: pointer;\n  border: 1px solid transparent;\n  border-bottom: none;\n  border-radius: 4px 4px 0 0;\n  white-space: nowrap;\n}\n.pp-client-tab.active[data-v-5c15f6cc] {\n  background: var(--bg-primary);\n  color: var(--text-primary);\n  border-color: var(--border-color);\n}\n.pp-client-tab-title[data-v-5c15f6cc] { max-width: 120px; overflow: hidden; text-overflow: ellipsis;\n}\n.pp-client-body[data-v-5c15f6cc] {\n  min-height: 80px;\n  max-height: 200px;\n  overflow: auto;\n  padding: 6px 8px;\n  font-size: 12px;\n}\n\n/* 列表 */\n.pp-list[data-v-5c15f6cc] { flex: 1; overflow: auto; padding: 4px 0;\n}\n.pp-loading[data-v-5c15f6cc], .pp-empty[data-v-5c15f6cc] {\n  display: flex; flex-direction: column; align-items: center; gap: 6px;\n  padding: 24px 12px; color: var(--text-muted); font-size: 12px;\n}\n.pp-empty-sub[data-v-5c15f6cc] { font-size: 11px; color: var(--text-muted); text-align: center;\n}\n.pp-item[data-v-5c15f6cc] { border-bottom: 1px solid var(--border-color);\n}\n.pp-item-row[data-v-5c15f6cc] {\n  display: flex; align-items: center; gap: 6px;\n  padding: 7px 10px;\n  cursor: pointer;\n}\n.pp-item-row[data-v-5c15f6cc]:hover { background: var(--bg-hover);\n}\n.pp-state[data-v-5c15f6cc] {\n  width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0;\n}\n.pp-state.on[data-v-5c15f6cc] { background: #4caf50; box-shadow: 0 0 4px rgba(76, 175, 80, .6);\n}\n.pp-state.off[data-v-5c15f6cc] { background: var(--text-muted); opacity: .4;\n}\n.pp-name[data-v-5c15f6cc] {\n  flex: 1; font-weight: 500; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;\n}\n.pp-src[data-v-5c15f6cc] {\n  font-size: 9px; padding: 1px 5px; border-radius: 3px;\n  font-family: var(--font-code); text-transform: uppercase;\n}\n.pp-src.js[data-v-5c15f6cc] { background: rgba(240, 219, 79, .15); color: #e5c07b;\n}\n.pp-src.go[data-v-5c15f6cc] { background: rgba(0, 178, 255, .12); color: #61afef;\n}\n.pp-badge[data-v-5c15f6cc] {\n  font-size: 9px; padding: 1px 5px; border-radius: 3px;\n  background: rgba(198, 120, 221, .15); color: #c678dd;\n  flex-shrink: 0;\n}\n.pp-badge-warn[data-v-5c15f6cc] {\n  background: rgba(229, 192, 123, .18); color: #e5c07b;\n  cursor: help;\n}\n.pp-count[data-v-5c15f6cc] { font-size: 10px; color: var(--text-muted); flex-shrink: 0;\n}\n.pp-ui-label[data-v-5c15f6cc] { font-size: 10px; color: var(--text-muted); flex-shrink: 0;\n}\n.pp-ui-label.on[data-v-5c15f6cc] { color: var(--accent, #4c9aff);\n}\n.pp-chevron[data-v-5c15f6cc] { transition: transform .15s; flex-shrink: 0;\n}\n.pp-chevron.open[data-v-5c15f6cc] { transform: rotate(90deg);\n}\n.pp-detail[data-v-5c15f6cc] { padding: 4px 10px 10px 24px; background: var(--bg-tertiary);\n}\n.pp-d-purpose[data-v-5c15f6cc] { font-size: 12px; color: var(--text-secondary); margin-bottom: 4px;\n}\n.pp-d-line[data-v-5c15f6cc] { font-size: 11px; color: var(--text-muted); margin: 2px 0; word-break: break-all;\n}\n.pp-d-tools[data-v-5c15f6cc] { display: flex; flex-direction: column; gap: 1px; margin: 4px 0; padding: 4px 6px; border: 1px solid var(--border-color); border-radius: 4px; background: var(--bg-primary);\n}\n.pp-d-tools-title[data-v-5c15f6cc] { font-size: 10px; color: var(--text-muted); margin-bottom: 2px;\n}\n.pp-d-tool[data-v-5c15f6cc] { display: flex; align-items: center; justify-content: space-between; gap: 8px; padding: 1px 2px; border-radius: 3px;\n}\n.pp-d-tool[data-v-5c15f6cc]:hover { background: var(--bg-secondary);\n}\n.pp-d-tname[data-v-5c15f6cc] { font-family: var(--font-code); font-size: 11px; color: var(--text-primary); overflow: hidden; text-overflow: ellipsis; white-space: nowrap;\n}\n.pp-d-code[data-v-5c15f6cc] {\n  margin-top: 6px;\n  border: 1px solid var(--border-color);\n  border-radius: 4px;\n  overflow: hidden;\n}\n.pp-d-code-head[data-v-5c15f6cc] {\n  display: flex; align-items: center; justify-content: space-between;\n  padding: 3px 8px;\n  background: var(--bg-primary);\n  font-size: 10px; color: var(--text-secondary);\n  border-bottom: 1px solid var(--border-color);\n}\n.pp-d-code pre[data-v-5c15f6cc] {\n  margin: 0; padding: 6px 8px;\n  font-family: var(--font-code); font-size: 10px;\n  line-height: 1.5;\n  color: var(--text-secondary);\n  overflow: auto;\n  max-height: 160px;\n  white-space: pre-wrap;\n  word-break: break-all;\n}\n.pp-d-actions[data-v-5c15f6cc] { display: flex; gap: 6px; margin-top: 8px;\n}\n.pp-btn[data-v-5c15f6cc] {\n  background: var(--bg-primary);\n  border: 1px solid var(--border-color);\n  color: var(--text-secondary);\n  border-radius: 4px;\n  padding: 3px 10px;\n  font-size: 11px;\n  cursor: pointer;\n}\n.pp-btn[data-v-5c15f6cc]:hover { background: var(--bg-hover); color: var(--text-primary);\n}\n.pp-btn.primary[data-v-5c15f6cc] { border-color: var(--accent); color: var(--accent-light);\n}\n.pp-btn.danger[data-v-5c15f6cc] { border-color: #e06c75; color: #e06c75;\n}\n.pp-btn[data-v-5c15f6cc]:disabled { opacity: .5; cursor: not-allowed;\n}\n.spinner[data-v-5c15f6cc] { animation: pp-spin-5c15f6cc 1s linear infinite;\n}\n@keyframes pp-spin-5c15f6cc {\nto { transform: rotate(360deg);\n}\n}\n\n.app-root[data-v-ed63d1b3] {\n  display: grid;\n  /* ★ 列保护：编辑器列 minmax(340px,1fr) 保证主区不被右侧/侧边栏挤压；\n     右侧列 var(--right-w) 由 ui-right-panel 包（拖拽/初始化）同步，\n     宿主与 bundle 根宽度一致（无空余）。--right-w = rpw+205\n     (chat rpw + conv 200 + resizer/border 5)，rpw 上限 360 → 右侧 ≤565px，\n     1280 窗口编辑器 ≥ 1280-48-280-565 = 387px。 */\n  grid-template-columns: 48px auto minmax(340px, 1fr) var(--right-w, 525px);\n  grid-template-rows: 30px 1fr 22px;\n  width: 100%; height: 100%;\n  background: var(--bg-primary);\n  color: var(--text-primary);\n  overflow: hidden;\n  font-family: var(--font-ui);\n}\n/* ★ 桌面端面板独立模式：只渲染右侧面板，占满整个窗口 */\n.app-root.panel-only[data-v-ed63d1b3] {\n  grid-template-columns: 1fr;\n  grid-template-rows: 1fr;\n}\n.app-root.panel-only .right-container[data-v-ed63d1b3] {\n  grid-column: 1; grid-row: 1;\n  width: 100% !important;\n  height: 100%;\n}\n/* 整区替换槽位（single）宿主：与内置区域同 grid 位置/尺寸 */\n.plugin-area-titlebar[data-v-ed63d1b3] { grid-column: 1 / -1; grid-row: 1; height: 30px;\n}\n.plugin-area-activitybar[data-v-ed63d1b3] { grid-column: 1; grid-row: 2; width: 48px;\n}\n.plugin-area-sidebar[data-v-ed63d1b3] { grid-column: 2; grid-row: 2; height: 100%; overflow: hidden;\n}\n.main-area[data-v-ed63d1b3] {\n  grid-column: 3; grid-row: 2;\n  display: flex; flex-direction: column; min-width: 0; overflow: hidden;\n}\n.right-container[data-v-ed63d1b3] {\n  grid-column: 4; grid-row: 2;\n  display: flex; flex-direction: row; overflow: hidden; position: relative;\n}\n.right-container.focus-mode[data-v-ed63d1b3] { grid-column: 3 / -1;\n}\n.app-statusbar-host[data-v-ed63d1b3] { grid-column: 1 / -1; grid-row: 3; z-index: 30; height: 22px;\n}\n.plugin-slot-host[data-v-ed63d1b3] { height: 100%; overflow: hidden;\n}\n/* ★ 插件渲染的子元素必须撑满宿主（bundle 根 auto 宽度不随宿主 grid 拉伸——\n   focus-mode 下宿主被 grid 拉到 3/-1 全宽，子元素保持内容宽 → 右侧大片空余） */\n.plugin-slot-host.right-container[data-v-ed63d1b3] > * { width: 100%; min-width: 0;\n}\n/* modals 槽位：fixed 全屏浮层容器（不占 grid 格） */\n.modals-host[data-v-ed63d1b3] { position: fixed; inset: 0; z-index: 200; pointer-events: none;\n}\n.modals-host[data-v-ed63d1b3] > * { pointer-events: auto;\n}\n.modals-empty[data-v-ed63d1b3] { display: none;\n}\n/* 空态占位（区域插件未装配时显示） */\n.slot-empty[data-v-ed63d1b3] {\n  display: flex; flex-direction: row; gap: 8px;\n  align-items: center; justify-content: center;\n  color: var(--text-muted); font-size: 12px;\n  background: var(--bg-primary);\n  border: 1px dashed var(--border-color);\n  min-height: 0;\n}\n/* activitybar 是竖条（~48px 宽）：空态改纵向排列 */\n.plugin-area-activitybar.slot-empty[data-v-ed63d1b3] { flex-direction: column; gap: 4px; padding: 4px;\n}\n.plugin-area-activitybar.slot-empty .escape-link[data-v-ed63d1b3] { font-size: 11px; padding: 2px 8px;\n}\n/* 空态内的「打开插件面板」恢复入口（上下文感知注入：只在区域未装配时出现，\n   插件全正常时零干扰；与常驻逃生按钮互为双保险） */\n.escape-link[data-v-ed63d1b3] {\n  background: none; border: 1px solid var(--border-color);\n  color: var(--accent, #4f8cff); font-size: 12px;\n  padding: 3px 12px; border-radius: 4px; cursor: pointer;\n  opacity: .85; transition: opacity .15s;\n}\n.escape-link[data-v-ed63d1b3]:hover { opacity: 1; background: rgba(79,140,255,.12);\n}\n/* ─── 壳级逃生口：插件面板浮动入口 ───\n   常驻极小按钮位于左下角（状态栏上方）：右侧面板占 grid 第 4 列，\n   左下角处于第 1-2 列区域（activitybar/sidebar 底部），focusMode 全屏对话\n   也不覆盖——不再遮挡右侧对话输入区。半透明弱化，hover 全显。\n   点击打开浮动插件面板（Fixed 560px 居中）。 */\n.plugin-escape-btn[data-v-ed63d1b3] {\n  position: fixed; left: 6px; bottom: 26px; z-index: 300;\n  width: 22px; height: 22px; border-radius: 5px;\n  display: flex; align-items: center; justify-content: center;\n  background: var(--bg-elevated, #2a2d36); color: var(--text-muted);\n  border: 1px solid var(--border-color); cursor: pointer;\n  opacity: .3; transition: opacity .15s;\n}\n.plugin-escape-btn[data-v-ed63d1b3]:hover { opacity: 1; color: var(--accent, #4f8cff);\n}\n.plugin-escape-overlay[data-v-ed63d1b3] {\n  position: fixed; inset: 0; z-index: 400;\n  background: rgba(0,0,0,.45);\n  display: flex; align-items: center; justify-content: center;\n}\n.plugin-escape-panel[data-v-ed63d1b3] {\n  width: 560px; max-width: 92vw; height: 70vh; max-height: 640px;\n  background: var(--bg-primary); border: 1px solid var(--border-color);\n  border-radius: 10px; box-shadow: 0 8px 40px rgba(0,0,0,.5);\n  display: flex; flex-direction: column; overflow: hidden;\n}\n.plugin-escape-head[data-v-ed63d1b3] {\n  display: flex; align-items: center; justify-content: space-between;\n  padding: 6px 10px; font-size: 12px; color: var(--text-muted);\n  border-bottom: 1px solid var(--border-color);\n  background: var(--bg-elevated, #262932);\n}\n.plugin-escape-close[data-v-ed63d1b3] {\n  border: none; background: none; color: var(--text-muted);\n  cursor: pointer; font-size: 13px; padding: 2px 6px; border-radius: 4px;\n}\n.plugin-escape-close[data-v-ed63d1b3]:hover { background: rgba(255,255,255,.08); color: #fff;\n}\n.plugin-escape-body[data-v-ed63d1b3] { flex: 1; overflow: auto;\n}\n.plugin-escape-body .plugin-panel[data-v-ed63d1b3] { height: 100%; border: none;\n}\n/*$vite$:1*/";
  document.head.appendChild(__vite_style__);
  /**
  * @vue/shared v3.5.39
  * (c) 2018-present Yuxi (Evan) You and Vue contributors
  * @license MIT
  **/
  // @__NO_SIDE_EFFECTS__
  function makeMap(str) {
    const map = /* @__PURE__ */ Object.create(null);
    for (const key of str.split(",")) map[key] = 1;
    return (val) => val in map;
  }
  const EMPTY_OBJ = {};
  const EMPTY_ARR = [];
  const NOOP = () => {
  };
  const NO = () => false;
  const isOn = (key) => key.charCodeAt(0) === 111 && key.charCodeAt(1) === 110 && // uppercase letter
  (key.charCodeAt(2) > 122 || key.charCodeAt(2) < 97);
  const isModelListener = (key) => key.startsWith("onUpdate:");
  const extend = Object.assign;
  const remove = (arr, el) => {
    const i = arr.indexOf(el);
    if (i > -1) {
      arr.splice(i, 1);
    }
  };
  const hasOwnProperty$1 = Object.prototype.hasOwnProperty;
  const hasOwn = (val, key) => hasOwnProperty$1.call(val, key);
  const isArray = Array.isArray;
  const isMap = (val) => toTypeString(val) === "[object Map]";
  const isSet = (val) => toTypeString(val) === "[object Set]";
  const isDate = (val) => toTypeString(val) === "[object Date]";
  const isRegExp = (val) => toTypeString(val) === "[object RegExp]";
  const isFunction = (val) => typeof val === "function";
  const isString = (val) => typeof val === "string";
  const isSymbol = (val) => typeof val === "symbol";
  const isObject = (val) => val !== null && typeof val === "object";
  const isPromise = (val) => {
    return (isObject(val) || isFunction(val)) && isFunction(val.then) && isFunction(val.catch);
  };
  const objectToString = Object.prototype.toString;
  const toTypeString = (value) => objectToString.call(value);
  const toRawType = (value) => {
    return toTypeString(value).slice(8, -1);
  };
  const isPlainObject = (val) => toTypeString(val) === "[object Object]";
  const isIntegerKey = (key) => isString(key) && key !== "NaN" && key[0] !== "-" && "" + parseInt(key, 10) === key;
  const isReservedProp = /* @__PURE__ */ makeMap(
    // the leading comma is intentional so empty string "" is also included
    ",key,ref,ref_for,ref_key,onVnodeBeforeMount,onVnodeMounted,onVnodeBeforeUpdate,onVnodeUpdated,onVnodeBeforeUnmount,onVnodeUnmounted"
  );
  const cacheStringFunction = (fn) => {
    const cache = /* @__PURE__ */ Object.create(null);
    return ((str) => {
      const hit = cache[str];
      return hit || (cache[str] = fn(str));
    });
  };
  const camelizeRE = /-\w/g;
  const camelize = cacheStringFunction(
    (str) => {
      return str.replace(camelizeRE, (c) => c.slice(1).toUpperCase());
    }
  );
  const hyphenateRE = /\B([A-Z])/g;
  const hyphenate = cacheStringFunction(
    (str) => str.replace(hyphenateRE, "-$1").toLowerCase()
  );
  const capitalize = cacheStringFunction((str) => {
    return str.charAt(0).toUpperCase() + str.slice(1);
  });
  const toHandlerKey = cacheStringFunction(
    (str) => {
      const s = str ? `on${capitalize(str)}` : ``;
      return s;
    }
  );
  const hasChanged = (value, oldValue) => !Object.is(value, oldValue);
  const invokeArrayFns = (fns, ...arg) => {
    for (let i = 0; i < fns.length; i++) {
      fns[i](...arg);
    }
  };
  const def = (obj, key, value, writable = false) => {
    Object.defineProperty(obj, key, {
      configurable: true,
      enumerable: false,
      writable,
      value
    });
  };
  const looseToNumber = (val) => {
    const n = parseFloat(val);
    return isNaN(n) ? val : n;
  };
  const toNumber = (val) => {
    const n = isString(val) ? Number(val) : NaN;
    return isNaN(n) ? val : n;
  };
  let _globalThis;
  const getGlobalThis = () => {
    return _globalThis || (_globalThis = typeof globalThis !== "undefined" ? globalThis : typeof self !== "undefined" ? self : typeof window !== "undefined" ? window : typeof global !== "undefined" ? global : {});
  };
  const GLOBALS_ALLOWED = "Infinity,undefined,NaN,isFinite,isNaN,parseFloat,parseInt,decodeURI,decodeURIComponent,encodeURI,encodeURIComponent,Math,Number,Date,Array,Object,Boolean,String,RegExp,Map,Set,JSON,Intl,BigInt,console,Error,Symbol";
  const isGloballyAllowed = /* @__PURE__ */ makeMap(GLOBALS_ALLOWED);
  function normalizeStyle(value) {
    if (isArray(value)) {
      const res = {};
      for (let i = 0; i < value.length; i++) {
        const item = value[i];
        const normalized = isString(item) ? parseStringStyle(item) : normalizeStyle(item);
        if (normalized) {
          for (const key in normalized) {
            res[key] = normalized[key];
          }
        }
      }
      return res;
    } else if (isString(value) || isObject(value)) {
      return value;
    }
  }
  const listDelimiterRE = /;(?![^(]*\))/g;
  const propertyDelimiterRE = /:([^]+)/;
  const styleCommentRE = /\/\*[^]*?\*\//g;
  function parseStringStyle(cssText) {
    const ret = {};
    cssText.replace(styleCommentRE, "").split(listDelimiterRE).forEach((item) => {
      if (item) {
        const tmp = item.split(propertyDelimiterRE);
        tmp.length > 1 && (ret[tmp[0].trim()] = tmp[1].trim());
      }
    });
    return ret;
  }
  function normalizeClass(value) {
    let res = "";
    if (isString(value)) {
      res = value;
    } else if (isArray(value)) {
      for (let i = 0; i < value.length; i++) {
        const normalized = normalizeClass(value[i]);
        if (normalized) {
          res += normalized + " ";
        }
      }
    } else if (isObject(value)) {
      for (const name in value) {
        if (value[name]) {
          res += name + " ";
        }
      }
    }
    return res.trim();
  }
  function normalizeProps(props) {
    if (!props) return null;
    let { class: klass, style } = props;
    if (klass && !isString(klass)) {
      props.class = normalizeClass(klass);
    }
    if (style) {
      props.style = normalizeStyle(style);
    }
    return props;
  }
  const specialBooleanAttrs = `itemscope,allowfullscreen,formnovalidate,ismap,nomodule,novalidate,readonly`;
  const isSpecialBooleanAttr = /* @__PURE__ */ makeMap(specialBooleanAttrs);
  function includeBooleanAttr(value) {
    return !!value || value === "";
  }
  function looseCompareArrays(a, b) {
    if (a.length !== b.length) return false;
    let equal = true;
    for (let i = 0; equal && i < a.length; i++) {
      equal = looseEqual(a[i], b[i]);
    }
    return equal;
  }
  function looseEqual(a, b) {
    if (a === b) return true;
    let aValidType = isDate(a);
    let bValidType = isDate(b);
    if (aValidType || bValidType) {
      return aValidType && bValidType ? a.getTime() === b.getTime() : false;
    }
    aValidType = isSymbol(a);
    bValidType = isSymbol(b);
    if (aValidType || bValidType) {
      return a === b;
    }
    aValidType = isArray(a);
    bValidType = isArray(b);
    if (aValidType || bValidType) {
      return aValidType && bValidType ? looseCompareArrays(a, b) : false;
    }
    aValidType = isObject(a);
    bValidType = isObject(b);
    if (aValidType || bValidType) {
      if (!aValidType || !bValidType) {
        return false;
      }
      const aKeysCount = Object.keys(a).length;
      const bKeysCount = Object.keys(b).length;
      if (aKeysCount !== bKeysCount) {
        return false;
      }
      for (const key in a) {
        const aHasKey = a.hasOwnProperty(key);
        const bHasKey = b.hasOwnProperty(key);
        if (aHasKey && !bHasKey || !aHasKey && bHasKey || !looseEqual(a[key], b[key])) {
          return false;
        }
      }
    }
    return String(a) === String(b);
  }
  function looseIndexOf(arr, val) {
    return arr.findIndex((item) => looseEqual(item, val));
  }
  const isRef$1 = (val) => {
    return !!(val && val["__v_isRef"] === true);
  };
  const toDisplayString = (val) => {
    return isString(val) ? val : val == null ? "" : isArray(val) || isObject(val) && (val.toString === objectToString || !isFunction(val.toString)) ? isRef$1(val) ? toDisplayString(val.value) : JSON.stringify(val, replacer, 2) : String(val);
  };
  const replacer = (_key, val) => {
    if (isRef$1(val)) {
      return replacer(_key, val.value);
    } else if (isMap(val)) {
      return {
        [`Map(${val.size})`]: [...val.entries()].reduce(
          (entries, [key, val2], i) => {
            entries[stringifySymbol(key, i) + " =>"] = val2;
            return entries;
          },
          {}
        )
      };
    } else if (isSet(val)) {
      return {
        [`Set(${val.size})`]: [...val.values()].map((v) => stringifySymbol(v))
      };
    } else if (isSymbol(val)) {
      return stringifySymbol(val);
    } else if (isObject(val) && !isArray(val) && !isPlainObject(val)) {
      return String(val);
    }
    return val;
  };
  const stringifySymbol = (v, i = "") => {
    var _a;
    return (
      // Symbol.description in es2019+ so we need to cast here to pass
      // the lib: es2016 check
      isSymbol(v) ? `Symbol(${(_a = v.description) != null ? _a : i})` : v
    );
  };
  function normalizeCssVarValue(value) {
    if (value == null) {
      return "initial";
    }
    if (typeof value === "string") {
      return value === "" ? " " : value;
    }
    return String(value);
  }
  /**
  * @vue/reactivity v3.5.39
  * (c) 2018-present Yuxi (Evan) You and Vue contributors
  * @license MIT
  **/
  let activeEffectScope;
  class EffectScope {
    // TODO isolatedDeclarations "__v_skip"
    constructor(detached = false) {
      this.detached = detached;
      this._active = true;
      this._on = 0;
      this.effects = [];
      this.cleanups = [];
      this._isPaused = false;
      this._warnOnRun = true;
      this.__v_skip = true;
      if (!detached && activeEffectScope) {
        if (activeEffectScope.active) {
          this.parent = activeEffectScope;
          this.index = (activeEffectScope.scopes || (activeEffectScope.scopes = [])).push(
            this
          ) - 1;
        } else {
          this._active = false;
          this._warnOnRun = false;
        }
      }
    }
    get active() {
      return this._active;
    }
    pause() {
      if (this._active) {
        this._isPaused = true;
        let i, l;
        if (this.scopes) {
          for (i = 0, l = this.scopes.length; i < l; i++) {
            this.scopes[i].pause();
          }
        }
        for (i = 0, l = this.effects.length; i < l; i++) {
          this.effects[i].pause();
        }
      }
    }
    /**
     * Resumes the effect scope, including all child scopes and effects.
     */
    resume() {
      if (this._active) {
        if (this._isPaused) {
          this._isPaused = false;
          let i, l;
          if (this.scopes) {
            for (i = 0, l = this.scopes.length; i < l; i++) {
              this.scopes[i].resume();
            }
          }
          for (i = 0, l = this.effects.length; i < l; i++) {
            this.effects[i].resume();
          }
        }
      }
    }
    run(fn) {
      if (this._active) {
        const currentEffectScope = activeEffectScope;
        try {
          activeEffectScope = this;
          return fn();
        } finally {
          activeEffectScope = currentEffectScope;
        }
      }
    }
    /**
     * This should only be called on non-detached scopes
     * @internal
     */
    on() {
      if (++this._on === 1) {
        this.prevScope = activeEffectScope;
        activeEffectScope = this;
      }
    }
    /**
     * This should only be called on non-detached scopes
     * @internal
     */
    off() {
      if (this._on > 0 && --this._on === 0) {
        if (activeEffectScope === this) {
          activeEffectScope = this.prevScope;
        } else {
          let current = activeEffectScope;
          while (current) {
            if (current.prevScope === this) {
              current.prevScope = this.prevScope;
              break;
            }
            current = current.prevScope;
          }
        }
        this.prevScope = void 0;
      }
    }
    stop(fromParent) {
      if (this._active) {
        this._active = false;
        let i, l;
        for (i = 0, l = this.effects.length; i < l; i++) {
          this.effects[i].stop();
        }
        this.effects.length = 0;
        for (i = 0, l = this.cleanups.length; i < l; i++) {
          this.cleanups[i]();
        }
        this.cleanups.length = 0;
        if (this.scopes) {
          for (i = 0, l = this.scopes.length; i < l; i++) {
            this.scopes[i].stop(true);
          }
          this.scopes.length = 0;
        }
        if (!this.detached && this.parent && !fromParent) {
          const last = this.parent.scopes.pop();
          if (last && last !== this) {
            this.parent.scopes[this.index] = last;
            last.index = this.index;
          }
        }
        this.parent = void 0;
      }
    }
  }
  function effectScope(detached) {
    return new EffectScope(detached);
  }
  function getCurrentScope() {
    return activeEffectScope;
  }
  function onScopeDispose(fn, failSilently = false) {
    if (activeEffectScope) {
      activeEffectScope.cleanups.push(fn);
    }
  }
  let activeSub;
  const pausedQueueEffects = /* @__PURE__ */ new WeakSet();
  class ReactiveEffect {
    constructor(fn) {
      this.fn = fn;
      this.deps = void 0;
      this.depsTail = void 0;
      this.flags = 1 | 4;
      this.next = void 0;
      this.cleanup = void 0;
      this.scheduler = void 0;
      if (activeEffectScope) {
        if (activeEffectScope.active) {
          activeEffectScope.effects.push(this);
        } else {
          this.flags &= -2;
        }
      }
    }
    pause() {
      this.flags |= 64;
    }
    resume() {
      if (this.flags & 64) {
        this.flags &= -65;
        if (pausedQueueEffects.has(this)) {
          pausedQueueEffects.delete(this);
          this.trigger();
        }
      }
    }
    /**
     * @internal
     */
    notify() {
      if (this.flags & 2 && !(this.flags & 32)) {
        return;
      }
      if (!(this.flags & 8)) {
        batch(this);
      }
    }
    run() {
      if (!(this.flags & 1)) {
        return this.fn();
      }
      this.flags |= 2;
      cleanupEffect(this);
      prepareDeps(this);
      const prevEffect = activeSub;
      const prevShouldTrack = shouldTrack;
      activeSub = this;
      shouldTrack = true;
      try {
        return this.fn();
      } finally {
        cleanupDeps(this);
        activeSub = prevEffect;
        shouldTrack = prevShouldTrack;
        this.flags &= -3;
      }
    }
    stop() {
      if (this.flags & 1) {
        for (let link = this.deps; link; link = link.nextDep) {
          removeSub(link);
        }
        this.deps = this.depsTail = void 0;
        cleanupEffect(this);
        this.onStop && this.onStop();
        this.flags &= -2;
      }
    }
    trigger() {
      if (this.flags & 64) {
        pausedQueueEffects.add(this);
      } else if (this.scheduler) {
        this.scheduler();
      } else {
        this.runIfDirty();
      }
    }
    /**
     * @internal
     */
    runIfDirty() {
      if (isDirty(this)) {
        this.run();
      }
    }
    get dirty() {
      return isDirty(this);
    }
  }
  let batchDepth = 0;
  let batchedSub;
  let batchedComputed;
  function batch(sub, isComputed = false) {
    sub.flags |= 8;
    if (isComputed) {
      sub.next = batchedComputed;
      batchedComputed = sub;
      return;
    }
    sub.next = batchedSub;
    batchedSub = sub;
  }
  function startBatch() {
    batchDepth++;
  }
  function endBatch() {
    if (--batchDepth > 0) {
      return;
    }
    if (batchedComputed) {
      let e = batchedComputed;
      batchedComputed = void 0;
      while (e) {
        const next = e.next;
        e.next = void 0;
        e.flags &= -9;
        e = next;
      }
    }
    let error;
    while (batchedSub) {
      let e = batchedSub;
      batchedSub = void 0;
      while (e) {
        const next = e.next;
        e.next = void 0;
        e.flags &= -9;
        if (e.flags & 1) {
          try {
            ;
            e.trigger();
          } catch (err) {
            if (!error) error = err;
          }
        }
        e = next;
      }
    }
    if (error) throw error;
  }
  function prepareDeps(sub) {
    for (let link = sub.deps; link; link = link.nextDep) {
      link.version = -1;
      link.prevActiveLink = link.dep.activeLink;
      link.dep.activeLink = link;
    }
  }
  function cleanupDeps(sub) {
    let head;
    let tail = sub.depsTail;
    let link = tail;
    while (link) {
      const prev = link.prevDep;
      if (link.version === -1) {
        if (link === tail) tail = prev;
        removeSub(link);
        removeDep(link);
      } else {
        head = link;
      }
      link.dep.activeLink = link.prevActiveLink;
      link.prevActiveLink = void 0;
      link = prev;
    }
    sub.deps = head;
    sub.depsTail = tail;
  }
  function isDirty(sub) {
    for (let link = sub.deps; link; link = link.nextDep) {
      if (link.dep.version !== link.version || link.dep.computed && (refreshComputed(link.dep.computed) || link.dep.version !== link.version)) {
        return true;
      }
    }
    if (sub._dirty) {
      return true;
    }
    return false;
  }
  function refreshComputed(computed2) {
    if (computed2.flags & 4 && !(computed2.flags & 16)) {
      return;
    }
    computed2.flags &= -17;
    if (computed2.globalVersion === globalVersion) {
      return;
    }
    computed2.globalVersion = globalVersion;
    if (!computed2.isSSR && computed2.flags & 128 && (!computed2.deps && !computed2._dirty || !isDirty(computed2))) {
      return;
    }
    computed2.flags |= 2;
    const dep = computed2.dep;
    const prevSub = activeSub;
    const prevShouldTrack = shouldTrack;
    activeSub = computed2;
    shouldTrack = true;
    try {
      prepareDeps(computed2);
      const value = computed2.fn(computed2._value);
      if (dep.version === 0 || hasChanged(value, computed2._value)) {
        computed2.flags |= 128;
        computed2._value = value;
        dep.version++;
      }
    } catch (err) {
      dep.version++;
      throw err;
    } finally {
      activeSub = prevSub;
      shouldTrack = prevShouldTrack;
      cleanupDeps(computed2);
      computed2.flags &= -3;
    }
  }
  function removeSub(link, soft = false) {
    const { dep, prevSub, nextSub } = link;
    if (prevSub) {
      prevSub.nextSub = nextSub;
      link.prevSub = void 0;
    }
    if (nextSub) {
      nextSub.prevSub = prevSub;
      link.nextSub = void 0;
    }
    if (dep.subs === link) {
      dep.subs = prevSub;
      if (!prevSub && dep.computed) {
        dep.computed.flags &= -5;
        for (let l = dep.computed.deps; l; l = l.nextDep) {
          removeSub(l, true);
        }
      }
    }
    if (!soft && !--dep.sc && dep.map) {
      dep.map.delete(dep.key);
    }
  }
  function removeDep(link) {
    const { prevDep, nextDep } = link;
    if (prevDep) {
      prevDep.nextDep = nextDep;
      link.prevDep = void 0;
    }
    if (nextDep) {
      nextDep.prevDep = prevDep;
      link.nextDep = void 0;
    }
  }
  function effect(fn, options) {
    if (fn.effect instanceof ReactiveEffect) {
      fn = fn.effect.fn;
    }
    const e = new ReactiveEffect(fn);
    if (options) {
      extend(e, options);
    }
    try {
      e.run();
    } catch (err) {
      e.stop();
      throw err;
    }
    const runner = e.run.bind(e);
    runner.effect = e;
    return runner;
  }
  function stop(runner) {
    runner.effect.stop();
  }
  let shouldTrack = true;
  const trackStack = [];
  function pauseTracking() {
    trackStack.push(shouldTrack);
    shouldTrack = false;
  }
  function resetTracking() {
    const last = trackStack.pop();
    shouldTrack = last === void 0 ? true : last;
  }
  function cleanupEffect(e) {
    const { cleanup } = e;
    e.cleanup = void 0;
    if (cleanup) {
      const prevSub = activeSub;
      activeSub = void 0;
      try {
        cleanup();
      } finally {
        activeSub = prevSub;
      }
    }
  }
  let globalVersion = 0;
  class Link {
    constructor(sub, dep) {
      this.sub = sub;
      this.dep = dep;
      this.version = dep.version;
      this.nextDep = this.prevDep = this.nextSub = this.prevSub = this.prevActiveLink = void 0;
    }
  }
  class Dep {
    // TODO isolatedDeclarations "__v_skip"
    constructor(computed2) {
      this.computed = computed2;
      this.version = 0;
      this.activeLink = void 0;
      this.subs = void 0;
      this.map = void 0;
      this.key = void 0;
      this.sc = 0;
      this.__v_skip = true;
    }
    track(debugInfo) {
      if (!activeSub || !shouldTrack || activeSub === this.computed) {
        return;
      }
      let link = this.activeLink;
      if (link === void 0 || link.sub !== activeSub) {
        link = this.activeLink = new Link(activeSub, this);
        if (!activeSub.deps) {
          activeSub.deps = activeSub.depsTail = link;
        } else {
          link.prevDep = activeSub.depsTail;
          activeSub.depsTail.nextDep = link;
          activeSub.depsTail = link;
        }
        addSub(link);
      } else if (link.version === -1) {
        link.version = this.version;
        if (link.nextDep) {
          const next = link.nextDep;
          next.prevDep = link.prevDep;
          if (link.prevDep) {
            link.prevDep.nextDep = next;
          }
          link.prevDep = activeSub.depsTail;
          link.nextDep = void 0;
          activeSub.depsTail.nextDep = link;
          activeSub.depsTail = link;
          if (activeSub.deps === link) {
            activeSub.deps = next;
          }
        }
      }
      return link;
    }
    trigger(debugInfo) {
      this.version++;
      globalVersion++;
      this.notify(debugInfo);
    }
    notify(debugInfo) {
      startBatch();
      try {
        if (false) ;
        for (let link = this.subs; link; link = link.prevSub) {
          if (link.sub.notify()) {
            ;
            link.sub.dep.notify();
          }
        }
      } finally {
        endBatch();
      }
    }
  }
  function addSub(link) {
    link.dep.sc++;
    if (link.sub.flags & 4) {
      const computed2 = link.dep.computed;
      if (computed2 && !link.dep.subs) {
        computed2.flags |= 4 | 16;
        for (let l = computed2.deps; l; l = l.nextDep) {
          addSub(l);
        }
      }
      const currentTail = link.dep.subs;
      if (currentTail !== link) {
        link.prevSub = currentTail;
        if (currentTail) currentTail.nextSub = link;
      }
      link.dep.subs = link;
    }
  }
  const targetMap = /* @__PURE__ */ new WeakMap();
  const ITERATE_KEY = /* @__PURE__ */ Symbol(
    ""
  );
  const MAP_KEY_ITERATE_KEY = /* @__PURE__ */ Symbol(
    ""
  );
  const ARRAY_ITERATE_KEY = /* @__PURE__ */ Symbol(
    ""
  );
  function track(target, type, key) {
    if (shouldTrack && activeSub) {
      let depsMap = targetMap.get(target);
      if (!depsMap) {
        targetMap.set(target, depsMap = /* @__PURE__ */ new Map());
      }
      let dep = depsMap.get(key);
      if (!dep) {
        depsMap.set(key, dep = new Dep());
        dep.map = depsMap;
        dep.key = key;
      }
      {
        dep.track();
      }
    }
  }
  function trigger(target, type, key, newValue, oldValue, oldTarget) {
    const depsMap = targetMap.get(target);
    if (!depsMap) {
      globalVersion++;
      return;
    }
    const run = (dep) => {
      if (dep) {
        {
          dep.trigger();
        }
      }
    };
    startBatch();
    if (type === "clear") {
      depsMap.forEach(run);
    } else {
      const targetIsArray = isArray(target);
      const isArrayIndex = targetIsArray && isIntegerKey(key);
      if (targetIsArray && key === "length") {
        const newLength = Number(newValue);
        depsMap.forEach((dep, key2) => {
          if (key2 === "length" || key2 === ARRAY_ITERATE_KEY || !isSymbol(key2) && key2 >= newLength) {
            run(dep);
          }
        });
      } else {
        if (key !== void 0 || depsMap.has(void 0)) {
          run(depsMap.get(key));
        }
        if (isArrayIndex) {
          run(depsMap.get(ARRAY_ITERATE_KEY));
        }
        switch (type) {
          case "add":
            if (!targetIsArray) {
              run(depsMap.get(ITERATE_KEY));
              if (isMap(target)) {
                run(depsMap.get(MAP_KEY_ITERATE_KEY));
              }
            } else if (isArrayIndex) {
              run(depsMap.get("length"));
            }
            break;
          case "delete":
            if (!targetIsArray) {
              run(depsMap.get(ITERATE_KEY));
              if (isMap(target)) {
                run(depsMap.get(MAP_KEY_ITERATE_KEY));
              }
            }
            break;
          case "set":
            if (isMap(target)) {
              run(depsMap.get(ITERATE_KEY));
            }
            break;
        }
      }
    }
    endBatch();
  }
  function getDepFromReactive(object, key) {
    const depMap = targetMap.get(object);
    return depMap && depMap.get(key);
  }
  function reactiveReadArray(array) {
    const raw = /* @__PURE__ */ toRaw(array);
    if (raw === array) return raw;
    track(raw, "iterate", ARRAY_ITERATE_KEY);
    return /* @__PURE__ */ isShallow(array) ? raw : raw.map(toReactive);
  }
  function shallowReadArray(arr) {
    track(arr = /* @__PURE__ */ toRaw(arr), "iterate", ARRAY_ITERATE_KEY);
    return arr;
  }
  function toWrapped(target, item) {
    if (/* @__PURE__ */ isReadonly(target)) {
      return /* @__PURE__ */ isReactive(target) ? toReadonly(toReactive(item)) : toReadonly(item);
    }
    return toReactive(item);
  }
  const arrayInstrumentations = {
    __proto__: null,
    [Symbol.iterator]() {
      return iterator(this, Symbol.iterator, (item) => toWrapped(this, item));
    },
    concat(...args) {
      return reactiveReadArray(this).concat(
        ...args.map((x) => isArray(x) ? reactiveReadArray(x) : x)
      );
    },
    entries() {
      return iterator(this, "entries", (value) => {
        value[1] = toWrapped(this, value[1]);
        return value;
      });
    },
    every(fn, thisArg) {
      return apply(this, "every", fn, thisArg, void 0, arguments);
    },
    filter(fn, thisArg) {
      return apply(
        this,
        "filter",
        fn,
        thisArg,
        (v) => v.map((item) => toWrapped(this, item)),
        arguments
      );
    },
    find(fn, thisArg) {
      return apply(
        this,
        "find",
        fn,
        thisArg,
        (item) => toWrapped(this, item),
        arguments
      );
    },
    findIndex(fn, thisArg) {
      return apply(this, "findIndex", fn, thisArg, void 0, arguments);
    },
    findLast(fn, thisArg) {
      return apply(
        this,
        "findLast",
        fn,
        thisArg,
        (item) => toWrapped(this, item),
        arguments
      );
    },
    findLastIndex(fn, thisArg) {
      return apply(this, "findLastIndex", fn, thisArg, void 0, arguments);
    },
    // flat, flatMap could benefit from ARRAY_ITERATE but are not straight-forward to implement
    forEach(fn, thisArg) {
      return apply(this, "forEach", fn, thisArg, void 0, arguments);
    },
    includes(...args) {
      return searchProxy(this, "includes", args);
    },
    indexOf(...args) {
      return searchProxy(this, "indexOf", args);
    },
    join(separator) {
      return reactiveReadArray(this).join(separator);
    },
    // keys() iterator only reads `length`, no optimization required
    lastIndexOf(...args) {
      return searchProxy(this, "lastIndexOf", args);
    },
    map(fn, thisArg) {
      return apply(this, "map", fn, thisArg, void 0, arguments);
    },
    pop() {
      return noTracking(this, "pop");
    },
    push(...args) {
      return noTracking(this, "push", args);
    },
    reduce(fn, ...args) {
      return reduce(this, "reduce", fn, args);
    },
    reduceRight(fn, ...args) {
      return reduce(this, "reduceRight", fn, args);
    },
    shift() {
      return noTracking(this, "shift");
    },
    // slice could use ARRAY_ITERATE but also seems to beg for range tracking
    some(fn, thisArg) {
      return apply(this, "some", fn, thisArg, void 0, arguments);
    },
    splice(...args) {
      return noTracking(this, "splice", args);
    },
    toReversed() {
      return reactiveReadArray(this).toReversed();
    },
    toSorted(comparer) {
      return reactiveReadArray(this).toSorted(comparer);
    },
    toSpliced(...args) {
      return reactiveReadArray(this).toSpliced(...args);
    },
    unshift(...args) {
      return noTracking(this, "unshift", args);
    },
    values() {
      return iterator(this, "values", (item) => toWrapped(this, item));
    }
  };
  function iterator(self2, method, wrapValue) {
    const arr = shallowReadArray(self2);
    const iter = arr[method]();
    if (arr !== self2 && !/* @__PURE__ */ isShallow(self2)) {
      iter._next = iter.next;
      iter.next = () => {
        const result = iter._next();
        if (!result.done) {
          result.value = wrapValue(result.value);
        }
        return result;
      };
    }
    return iter;
  }
  const arrayProto = Array.prototype;
  function apply(self2, method, fn, thisArg, wrappedRetFn, args) {
    const arr = shallowReadArray(self2);
    const needsWrap = arr !== self2 && !/* @__PURE__ */ isShallow(self2);
    const methodFn = arr[method];
    if (methodFn !== arrayProto[method]) {
      const result2 = methodFn.apply(self2, args);
      return needsWrap ? toReactive(result2) : result2;
    }
    let wrappedFn = fn;
    if (arr !== self2) {
      if (needsWrap) {
        wrappedFn = function(item, index) {
          return fn.call(this, toWrapped(self2, item), index, self2);
        };
      } else if (fn.length > 2) {
        wrappedFn = function(item, index) {
          return fn.call(this, item, index, self2);
        };
      }
    }
    const result = methodFn.call(arr, wrappedFn, thisArg);
    return needsWrap && wrappedRetFn ? wrappedRetFn(result) : result;
  }
  function reduce(self2, method, fn, args) {
    const arr = shallowReadArray(self2);
    const needsWrap = arr !== self2 && !/* @__PURE__ */ isShallow(self2);
    let wrappedFn = fn;
    let wrapInitialAccumulator = false;
    if (arr !== self2) {
      if (needsWrap) {
        wrapInitialAccumulator = args.length === 0;
        wrappedFn = function(acc, item, index) {
          if (wrapInitialAccumulator) {
            wrapInitialAccumulator = false;
            acc = toWrapped(self2, acc);
          }
          return fn.call(this, acc, toWrapped(self2, item), index, self2);
        };
      } else if (fn.length > 3) {
        wrappedFn = function(acc, item, index) {
          return fn.call(this, acc, item, index, self2);
        };
      }
    }
    const result = arr[method](wrappedFn, ...args);
    return wrapInitialAccumulator ? toWrapped(self2, result) : result;
  }
  function searchProxy(self2, method, args) {
    const arr = /* @__PURE__ */ toRaw(self2);
    track(arr, "iterate", ARRAY_ITERATE_KEY);
    const res = arr[method](...args);
    if ((res === -1 || res === false) && /* @__PURE__ */ isProxy(args[0])) {
      args[0] = /* @__PURE__ */ toRaw(args[0]);
      return arr[method](...args);
    }
    return res;
  }
  function noTracking(self2, method, args = []) {
    pauseTracking();
    startBatch();
    const res = (/* @__PURE__ */ toRaw(self2))[method].apply(self2, args);
    endBatch();
    resetTracking();
    return res;
  }
  const isNonTrackableKeys = /* @__PURE__ */ makeMap(`__proto__,__v_isRef,__isVue`);
  const builtInSymbols = new Set(
    /* @__PURE__ */ Object.getOwnPropertyNames(Symbol).filter((key) => key !== "arguments" && key !== "caller").map((key) => Symbol[key]).filter(isSymbol)
  );
  function hasOwnProperty(key) {
    if (!isSymbol(key)) key = String(key);
    const obj = /* @__PURE__ */ toRaw(this);
    track(obj, "has", key);
    return obj.hasOwnProperty(key);
  }
  class BaseReactiveHandler {
    constructor(_isReadonly = false, _isShallow = false) {
      this._isReadonly = _isReadonly;
      this._isShallow = _isShallow;
    }
    get(target, key, receiver) {
      if (key === "__v_skip") return target["__v_skip"];
      const isReadonly2 = this._isReadonly, isShallow2 = this._isShallow;
      if (key === "__v_isReactive") {
        return !isReadonly2;
      } else if (key === "__v_isReadonly") {
        return isReadonly2;
      } else if (key === "__v_isShallow") {
        return isShallow2;
      } else if (key === "__v_raw") {
        if (receiver === (isReadonly2 ? isShallow2 ? shallowReadonlyMap : readonlyMap : isShallow2 ? shallowReactiveMap : reactiveMap).get(target) || // receiver is not the reactive proxy, but has the same prototype
        // this means the receiver is a user proxy of the reactive proxy
        Object.getPrototypeOf(target) === Object.getPrototypeOf(receiver)) {
          return target;
        }
        return;
      }
      const targetIsArray = isArray(target);
      if (!isReadonly2) {
        let fn;
        if (targetIsArray && (fn = arrayInstrumentations[key])) {
          return fn;
        }
        if (key === "hasOwnProperty") {
          return hasOwnProperty;
        }
      }
      const res = Reflect.get(
        target,
        key,
        // if this is a proxy wrapping a ref, return methods using the raw ref
        // as receiver so that we don't have to call `toRaw` on the ref in all
        // its class methods
        /* @__PURE__ */ isRef(target) ? target : receiver
      );
      if (isSymbol(key) ? builtInSymbols.has(key) : isNonTrackableKeys(key)) {
        return res;
      }
      if (!isReadonly2) {
        track(target, "get", key);
      }
      if (isShallow2) {
        return res;
      }
      if (/* @__PURE__ */ isRef(res)) {
        const value = targetIsArray && isIntegerKey(key) ? res : res.value;
        return isReadonly2 && isObject(value) ? /* @__PURE__ */ readonly(value) : value;
      }
      if (isObject(res)) {
        return isReadonly2 ? /* @__PURE__ */ readonly(res) : /* @__PURE__ */ reactive(res);
      }
      return res;
    }
  }
  class MutableReactiveHandler extends BaseReactiveHandler {
    constructor(isShallow2 = false) {
      super(false, isShallow2);
    }
    set(target, key, value, receiver) {
      let oldValue = target[key];
      const isArrayWithIntegerKey = isArray(target) && isIntegerKey(key);
      if (!this._isShallow) {
        const isOldValueReadonly = /* @__PURE__ */ isReadonly(oldValue);
        if (!/* @__PURE__ */ isShallow(value) && !/* @__PURE__ */ isReadonly(value)) {
          oldValue = /* @__PURE__ */ toRaw(oldValue);
          value = /* @__PURE__ */ toRaw(value);
        }
        if (!isArrayWithIntegerKey && /* @__PURE__ */ isRef(oldValue) && !/* @__PURE__ */ isRef(value)) {
          if (isOldValueReadonly) {
            return true;
          } else {
            oldValue.value = value;
            return true;
          }
        }
      }
      const hadKey = isArrayWithIntegerKey ? Number(key) < target.length : hasOwn(target, key);
      const result = Reflect.set(
        target,
        key,
        value,
        /* @__PURE__ */ isRef(target) ? target : receiver
      );
      if (target === /* @__PURE__ */ toRaw(receiver) && result) {
        if (!hadKey) {
          trigger(target, "add", key, value);
        } else if (hasChanged(value, oldValue)) {
          trigger(target, "set", key, value);
        }
      }
      return result;
    }
    deleteProperty(target, key) {
      const hadKey = hasOwn(target, key);
      target[key];
      const result = Reflect.deleteProperty(target, key);
      if (result && hadKey) {
        trigger(target, "delete", key, void 0);
      }
      return result;
    }
    has(target, key) {
      const result = Reflect.has(target, key);
      if (!isSymbol(key) || !builtInSymbols.has(key)) {
        track(target, "has", key);
      }
      return result;
    }
    ownKeys(target) {
      track(
        target,
        "iterate",
        isArray(target) ? "length" : ITERATE_KEY
      );
      return Reflect.ownKeys(target);
    }
  }
  class ReadonlyReactiveHandler extends BaseReactiveHandler {
    constructor(isShallow2 = false) {
      super(true, isShallow2);
    }
    set(target, key) {
      return true;
    }
    deleteProperty(target, key) {
      return true;
    }
  }
  const mutableHandlers = /* @__PURE__ */ new MutableReactiveHandler();
  const readonlyHandlers = /* @__PURE__ */ new ReadonlyReactiveHandler();
  const shallowReactiveHandlers = /* @__PURE__ */ new MutableReactiveHandler(true);
  const shallowReadonlyHandlers = /* @__PURE__ */ new ReadonlyReactiveHandler(true);
  const toShallow = (value) => value;
  const getProto = (v) => Reflect.getPrototypeOf(v);
  function createIterableMethod(method, isReadonly2, isShallow2) {
    return function(...args) {
      const target = this["__v_raw"];
      const rawTarget = /* @__PURE__ */ toRaw(target);
      const targetIsMap = isMap(rawTarget);
      const isPair = method === "entries" || method === Symbol.iterator && targetIsMap;
      const isKeyOnly = method === "keys" && targetIsMap;
      const innerIterator = target[method](...args);
      const wrap = isShallow2 ? toShallow : isReadonly2 ? toReadonly : toReactive;
      !isReadonly2 && track(
        rawTarget,
        "iterate",
        isKeyOnly ? MAP_KEY_ITERATE_KEY : ITERATE_KEY
      );
      return extend(
        // inheriting all iterator properties
        Object.create(innerIterator),
        {
          // iterator protocol
          next() {
            const { value, done } = innerIterator.next();
            return done ? { value, done } : {
              value: isPair ? [wrap(value[0]), wrap(value[1])] : wrap(value),
              done
            };
          }
        }
      );
    };
  }
  function createReadonlyMethod(type) {
    return function(...args) {
      return type === "delete" ? false : type === "clear" ? void 0 : this;
    };
  }
  function createInstrumentations(readonly2, shallow) {
    const instrumentations = {
      get(key) {
        const target = this["__v_raw"];
        const rawTarget = /* @__PURE__ */ toRaw(target);
        const rawKey = /* @__PURE__ */ toRaw(key);
        if (!readonly2) {
          if (hasChanged(key, rawKey)) {
            track(rawTarget, "get", key);
          }
          track(rawTarget, "get", rawKey);
        }
        const { has } = getProto(rawTarget);
        const wrap = shallow ? toShallow : readonly2 ? toReadonly : toReactive;
        if (has.call(rawTarget, key)) {
          return wrap(target.get(key));
        } else if (has.call(rawTarget, rawKey)) {
          return wrap(target.get(rawKey));
        } else if (target !== rawTarget) {
          target.get(key);
        }
      },
      get size() {
        const target = this["__v_raw"];
        !readonly2 && track(/* @__PURE__ */ toRaw(target), "iterate", ITERATE_KEY);
        return target.size;
      },
      has(key) {
        const target = this["__v_raw"];
        const rawTarget = /* @__PURE__ */ toRaw(target);
        const rawKey = /* @__PURE__ */ toRaw(key);
        if (!readonly2) {
          if (hasChanged(key, rawKey)) {
            track(rawTarget, "has", key);
          }
          track(rawTarget, "has", rawKey);
        }
        return key === rawKey ? target.has(key) : target.has(key) || target.has(rawKey);
      },
      forEach(callback, thisArg) {
        const observed = this;
        const target = observed["__v_raw"];
        const rawTarget = /* @__PURE__ */ toRaw(target);
        const wrap = shallow ? toShallow : readonly2 ? toReadonly : toReactive;
        !readonly2 && track(rawTarget, "iterate", ITERATE_KEY);
        return target.forEach((value, key) => {
          return callback.call(thisArg, wrap(value), wrap(key), observed);
        });
      }
    };
    extend(
      instrumentations,
      readonly2 ? {
        add: createReadonlyMethod("add"),
        set: createReadonlyMethod("set"),
        delete: createReadonlyMethod("delete"),
        clear: createReadonlyMethod("clear")
      } : {
        add(value) {
          const target = /* @__PURE__ */ toRaw(this);
          const proto = getProto(target);
          const rawValue = /* @__PURE__ */ toRaw(value);
          const valueToAdd = !shallow && !/* @__PURE__ */ isShallow(value) && !/* @__PURE__ */ isReadonly(value) ? rawValue : value;
          const hadKey = proto.has.call(target, valueToAdd) || hasChanged(value, valueToAdd) && proto.has.call(target, value) || hasChanged(rawValue, valueToAdd) && proto.has.call(target, rawValue);
          if (!hadKey) {
            target.add(valueToAdd);
            trigger(target, "add", valueToAdd, valueToAdd);
          }
          return this;
        },
        set(key, value) {
          if (!shallow && !/* @__PURE__ */ isShallow(value) && !/* @__PURE__ */ isReadonly(value)) {
            value = /* @__PURE__ */ toRaw(value);
          }
          const target = /* @__PURE__ */ toRaw(this);
          const { has, get } = getProto(target);
          let hadKey = has.call(target, key);
          if (!hadKey) {
            key = /* @__PURE__ */ toRaw(key);
            hadKey = has.call(target, key);
          }
          const oldValue = get.call(target, key);
          target.set(key, value);
          if (!hadKey) {
            trigger(target, "add", key, value);
          } else if (hasChanged(value, oldValue)) {
            trigger(target, "set", key, value);
          }
          return this;
        },
        delete(key) {
          const target = /* @__PURE__ */ toRaw(this);
          const { has, get } = getProto(target);
          let hadKey = has.call(target, key);
          if (!hadKey) {
            key = /* @__PURE__ */ toRaw(key);
            hadKey = has.call(target, key);
          }
          get ? get.call(target, key) : void 0;
          const result = target.delete(key);
          if (hadKey) {
            trigger(target, "delete", key, void 0);
          }
          return result;
        },
        clear() {
          const target = /* @__PURE__ */ toRaw(this);
          const hadItems = target.size !== 0;
          const result = target.clear();
          if (hadItems) {
            trigger(
              target,
              "clear",
              void 0,
              void 0
            );
          }
          return result;
        }
      }
    );
    const iteratorMethods = [
      "keys",
      "values",
      "entries",
      Symbol.iterator
    ];
    iteratorMethods.forEach((method) => {
      instrumentations[method] = createIterableMethod(method, readonly2, shallow);
    });
    return instrumentations;
  }
  function createInstrumentationGetter(isReadonly2, shallow) {
    const instrumentations = createInstrumentations(isReadonly2, shallow);
    return (target, key, receiver) => {
      if (key === "__v_isReactive") {
        return !isReadonly2;
      } else if (key === "__v_isReadonly") {
        return isReadonly2;
      } else if (key === "__v_raw") {
        return target;
      }
      return Reflect.get(
        hasOwn(instrumentations, key) && key in target ? instrumentations : target,
        key,
        receiver
      );
    };
  }
  const mutableCollectionHandlers = {
    get: /* @__PURE__ */ createInstrumentationGetter(false, false)
  };
  const shallowCollectionHandlers = {
    get: /* @__PURE__ */ createInstrumentationGetter(false, true)
  };
  const readonlyCollectionHandlers = {
    get: /* @__PURE__ */ createInstrumentationGetter(true, false)
  };
  const shallowReadonlyCollectionHandlers = {
    get: /* @__PURE__ */ createInstrumentationGetter(true, true)
  };
  const reactiveMap = /* @__PURE__ */ new WeakMap();
  const shallowReactiveMap = /* @__PURE__ */ new WeakMap();
  const readonlyMap = /* @__PURE__ */ new WeakMap();
  const shallowReadonlyMap = /* @__PURE__ */ new WeakMap();
  function targetTypeMap(rawType) {
    switch (rawType) {
      case "Object":
      case "Array":
        return 1;
      case "Map":
      case "Set":
      case "WeakMap":
      case "WeakSet":
        return 2;
      default:
        return 0;
    }
  }
  // @__NO_SIDE_EFFECTS__
  function reactive(target) {
    if (/* @__PURE__ */ isReadonly(target)) {
      return target;
    }
    return createReactiveObject(
      target,
      false,
      mutableHandlers,
      mutableCollectionHandlers,
      reactiveMap
    );
  }
  // @__NO_SIDE_EFFECTS__
  function shallowReactive(target) {
    return createReactiveObject(
      target,
      false,
      shallowReactiveHandlers,
      shallowCollectionHandlers,
      shallowReactiveMap
    );
  }
  // @__NO_SIDE_EFFECTS__
  function readonly(target) {
    return createReactiveObject(
      target,
      true,
      readonlyHandlers,
      readonlyCollectionHandlers,
      readonlyMap
    );
  }
  // @__NO_SIDE_EFFECTS__
  function shallowReadonly(target) {
    return createReactiveObject(
      target,
      true,
      shallowReadonlyHandlers,
      shallowReadonlyCollectionHandlers,
      shallowReadonlyMap
    );
  }
  function createReactiveObject(target, isReadonly2, baseHandlers, collectionHandlers, proxyMap) {
    if (!isObject(target)) {
      return target;
    }
    if (target["__v_raw"] && !(isReadonly2 && target["__v_isReactive"])) {
      return target;
    }
    if (target["__v_skip"] || !Object.isExtensible(target)) {
      return target;
    }
    const existingProxy = proxyMap.get(target);
    if (existingProxy) {
      return existingProxy;
    }
    const targetType = targetTypeMap(toRawType(target));
    if (targetType === 0) {
      return target;
    }
    const proxy = new Proxy(
      target,
      targetType === 2 ? collectionHandlers : baseHandlers
    );
    proxyMap.set(target, proxy);
    return proxy;
  }
  // @__NO_SIDE_EFFECTS__
  function isReactive(value) {
    if (/* @__PURE__ */ isReadonly(value)) {
      return /* @__PURE__ */ isReactive(value["__v_raw"]);
    }
    return !!(value && value["__v_isReactive"]);
  }
  // @__NO_SIDE_EFFECTS__
  function isReadonly(value) {
    return !!(value && value["__v_isReadonly"]);
  }
  // @__NO_SIDE_EFFECTS__
  function isShallow(value) {
    return !!(value && value["__v_isShallow"]);
  }
  // @__NO_SIDE_EFFECTS__
  function isProxy(value) {
    return value ? !!value["__v_raw"] : false;
  }
  // @__NO_SIDE_EFFECTS__
  function toRaw(observed) {
    const raw = observed && observed["__v_raw"];
    return raw ? /* @__PURE__ */ toRaw(raw) : observed;
  }
  function markRaw(value) {
    if (!hasOwn(value, "__v_skip") && Object.isExtensible(value)) {
      def(value, "__v_skip", true);
    }
    return value;
  }
  const toReactive = (value) => isObject(value) ? /* @__PURE__ */ reactive(value) : value;
  const toReadonly = (value) => isObject(value) ? /* @__PURE__ */ readonly(value) : value;
  // @__NO_SIDE_EFFECTS__
  function isRef(r) {
    return r ? r["__v_isRef"] === true : false;
  }
  // @__NO_SIDE_EFFECTS__
  function ref(value) {
    return createRef(value, false);
  }
  // @__NO_SIDE_EFFECTS__
  function shallowRef(value) {
    return createRef(value, true);
  }
  function createRef(rawValue, shallow) {
    if (/* @__PURE__ */ isRef(rawValue)) {
      return rawValue;
    }
    return new RefImpl(rawValue, shallow);
  }
  class RefImpl {
    constructor(value, isShallow2) {
      this.dep = new Dep();
      this["__v_isRef"] = true;
      this["__v_isShallow"] = false;
      this._rawValue = isShallow2 ? value : /* @__PURE__ */ toRaw(value);
      this._value = isShallow2 ? value : toReactive(value);
      this["__v_isShallow"] = isShallow2;
    }
    get value() {
      {
        this.dep.track();
      }
      return this._value;
    }
    set value(newValue) {
      const oldValue = this._rawValue;
      const useDirectValue = this["__v_isShallow"] || /* @__PURE__ */ isShallow(newValue) || /* @__PURE__ */ isReadonly(newValue);
      newValue = useDirectValue ? newValue : /* @__PURE__ */ toRaw(newValue);
      if (hasChanged(newValue, oldValue)) {
        this._rawValue = newValue;
        this._value = useDirectValue ? newValue : toReactive(newValue);
        {
          this.dep.trigger();
        }
      }
    }
  }
  function triggerRef(ref2) {
    if (ref2.dep) {
      {
        ref2.dep.trigger();
      }
    }
  }
  function unref(ref2) {
    return /* @__PURE__ */ isRef(ref2) ? ref2.value : ref2;
  }
  function toValue(source) {
    return isFunction(source) ? source() : unref(source);
  }
  const shallowUnwrapHandlers = {
    get: (target, key, receiver) => key === "__v_raw" ? target : unref(Reflect.get(target, key, receiver)),
    set: (target, key, value, receiver) => {
      const oldValue = target[key];
      if (/* @__PURE__ */ isRef(oldValue) && !/* @__PURE__ */ isRef(value)) {
        oldValue.value = value;
        return true;
      } else {
        return Reflect.set(target, key, value, receiver);
      }
    }
  };
  function proxyRefs(objectWithRefs) {
    return /* @__PURE__ */ isReactive(objectWithRefs) ? objectWithRefs : new Proxy(objectWithRefs, shallowUnwrapHandlers);
  }
  class CustomRefImpl {
    constructor(factory) {
      this["__v_isRef"] = true;
      this._value = void 0;
      const dep = this.dep = new Dep();
      const { get, set } = factory(dep.track.bind(dep), dep.trigger.bind(dep));
      this._get = get;
      this._set = set;
    }
    get value() {
      return this._value = this._get();
    }
    set value(newVal) {
      this._set(newVal);
    }
  }
  function customRef(factory) {
    return new CustomRefImpl(factory);
  }
  // @__NO_SIDE_EFFECTS__
  function toRefs(object) {
    const ret = isArray(object) ? new Array(object.length) : {};
    for (const key in object) {
      ret[key] = propertyToRef(object, key);
    }
    return ret;
  }
  class ObjectRefImpl {
    constructor(_object, key, _defaultValue) {
      this._object = _object;
      this._defaultValue = _defaultValue;
      this["__v_isRef"] = true;
      this._value = void 0;
      this._key = isSymbol(key) ? key : String(key);
      this._raw = /* @__PURE__ */ toRaw(_object);
      let shallow = true;
      let obj = _object;
      if (!isArray(_object) || isSymbol(this._key) || !isIntegerKey(this._key)) {
        do {
          shallow = !/* @__PURE__ */ isProxy(obj) || /* @__PURE__ */ isShallow(obj);
        } while (shallow && (obj = obj["__v_raw"]));
      }
      this._shallow = shallow;
    }
    get value() {
      let val = this._object[this._key];
      if (this._shallow) {
        val = unref(val);
      }
      return this._value = val === void 0 ? this._defaultValue : val;
    }
    set value(newVal) {
      if (this._shallow && /* @__PURE__ */ isRef(this._raw[this._key])) {
        const nestedRef = this._object[this._key];
        if (/* @__PURE__ */ isRef(nestedRef)) {
          nestedRef.value = newVal;
          return;
        }
      }
      this._object[this._key] = newVal;
    }
    get dep() {
      return getDepFromReactive(this._raw, this._key);
    }
  }
  class GetterRefImpl {
    constructor(_getter) {
      this._getter = _getter;
      this["__v_isRef"] = true;
      this["__v_isReadonly"] = true;
      this._value = void 0;
    }
    get value() {
      return this._value = this._getter();
    }
  }
  // @__NO_SIDE_EFFECTS__
  function toRef(source, key, defaultValue) {
    if (/* @__PURE__ */ isRef(source)) {
      return source;
    } else if (isFunction(source)) {
      return new GetterRefImpl(source);
    } else if (isObject(source) && arguments.length > 1) {
      return propertyToRef(source, key, defaultValue);
    } else {
      return /* @__PURE__ */ ref(source);
    }
  }
  function propertyToRef(source, key, defaultValue) {
    return new ObjectRefImpl(source, key, defaultValue);
  }
  class ComputedRefImpl {
    constructor(fn, setter, isSSR) {
      this.fn = fn;
      this.setter = setter;
      this._value = void 0;
      this.dep = new Dep(this);
      this.__v_isRef = true;
      this.deps = void 0;
      this.depsTail = void 0;
      this.flags = 16;
      this.globalVersion = globalVersion - 1;
      this.next = void 0;
      this.effect = this;
      this["__v_isReadonly"] = !setter;
      this.isSSR = isSSR;
    }
    /**
     * @internal
     */
    notify() {
      this.flags |= 16;
      if (!(this.flags & 8) && // avoid infinite self recursion
      activeSub !== this) {
        batch(this, true);
        return true;
      }
    }
    get value() {
      const link = this.dep.track();
      refreshComputed(this);
      if (link) {
        link.version = this.dep.version;
      }
      return this._value;
    }
    set value(newValue) {
      if (this.setter) {
        this.setter(newValue);
      }
    }
  }
  // @__NO_SIDE_EFFECTS__
  function computed$1(getterOrOptions, debugOptions, isSSR = false) {
    let getter;
    let setter;
    if (isFunction(getterOrOptions)) {
      getter = getterOrOptions;
    } else {
      getter = getterOrOptions.get;
      setter = getterOrOptions.set;
    }
    const cRef = new ComputedRefImpl(getter, setter, isSSR);
    return cRef;
  }
  const TrackOpTypes = {
    "GET": "get",
    "HAS": "has",
    "ITERATE": "iterate"
  };
  const TriggerOpTypes = {
    "SET": "set",
    "ADD": "add",
    "DELETE": "delete",
    "CLEAR": "clear"
  };
  const INITIAL_WATCHER_VALUE = {};
  const cleanupMap = /* @__PURE__ */ new WeakMap();
  let activeWatcher = void 0;
  function getCurrentWatcher() {
    return activeWatcher;
  }
  function onWatcherCleanup(cleanupFn, failSilently = false, owner = activeWatcher) {
    if (owner) {
      let cleanups = cleanupMap.get(owner);
      if (!cleanups) cleanupMap.set(owner, cleanups = []);
      cleanups.push(cleanupFn);
    }
  }
  function watch$1(source, cb, options = EMPTY_OBJ) {
    const { immediate, deep, once, scheduler, augmentJob, call } = options;
    const reactiveGetter = (source2) => {
      if (deep) return source2;
      if (/* @__PURE__ */ isShallow(source2) || deep === false || deep === 0)
        return traverse(source2, 1);
      return traverse(source2);
    };
    let effect2;
    let getter;
    let cleanup;
    let boundCleanup;
    let forceTrigger = false;
    let isMultiSource = false;
    if (/* @__PURE__ */ isRef(source)) {
      getter = () => source.value;
      forceTrigger = /* @__PURE__ */ isShallow(source);
    } else if (/* @__PURE__ */ isReactive(source)) {
      getter = () => reactiveGetter(source);
      forceTrigger = true;
    } else if (isArray(source)) {
      isMultiSource = true;
      forceTrigger = source.some((s) => /* @__PURE__ */ isReactive(s) || /* @__PURE__ */ isShallow(s));
      getter = () => source.map((s) => {
        if (/* @__PURE__ */ isRef(s)) {
          return s.value;
        } else if (/* @__PURE__ */ isReactive(s)) {
          return reactiveGetter(s);
        } else if (isFunction(s)) {
          return call ? call(s, 2) : s();
        } else ;
      });
    } else if (isFunction(source)) {
      if (cb) {
        getter = call ? () => call(source, 2) : source;
      } else {
        getter = () => {
          if (cleanup) {
            pauseTracking();
            try {
              cleanup();
            } finally {
              resetTracking();
            }
          }
          const currentEffect = activeWatcher;
          activeWatcher = effect2;
          try {
            return call ? call(source, 3, [boundCleanup]) : source(boundCleanup);
          } finally {
            activeWatcher = currentEffect;
          }
        };
      }
    } else {
      getter = NOOP;
    }
    if (cb && deep) {
      const baseGetter = getter;
      const depth = deep === true ? Infinity : deep;
      getter = () => traverse(baseGetter(), depth);
    }
    const scope = getCurrentScope();
    const watchHandle = () => {
      effect2.stop();
      if (scope && scope.active) {
        remove(scope.effects, effect2);
      }
    };
    if (once && cb) {
      const _cb = cb;
      cb = (...args) => {
        const res = _cb(...args);
        watchHandle();
        return res;
      };
    }
    let oldValue = isMultiSource ? new Array(source.length).fill(INITIAL_WATCHER_VALUE) : INITIAL_WATCHER_VALUE;
    const job = (immediateFirstRun) => {
      if (!(effect2.flags & 1) || !effect2.dirty && !immediateFirstRun) {
        return;
      }
      if (cb) {
        const newValue = effect2.run();
        if (immediateFirstRun || deep || forceTrigger || (isMultiSource ? newValue.some((v, i) => hasChanged(v, oldValue[i])) : hasChanged(newValue, oldValue))) {
          if (cleanup) {
            cleanup();
          }
          const currentWatcher = activeWatcher;
          activeWatcher = effect2;
          try {
            const args = [
              newValue,
              // pass undefined as the old value when it's changed for the first time
              oldValue === INITIAL_WATCHER_VALUE ? void 0 : isMultiSource && oldValue[0] === INITIAL_WATCHER_VALUE ? [] : oldValue,
              boundCleanup
            ];
            oldValue = newValue;
            call ? call(cb, 3, args) : (
              // @ts-expect-error
              cb(...args)
            );
          } finally {
            activeWatcher = currentWatcher;
          }
        }
      } else {
        effect2.run();
      }
    };
    if (augmentJob) {
      augmentJob(job);
    }
    effect2 = new ReactiveEffect(getter);
    effect2.scheduler = scheduler ? () => scheduler(job, false) : job;
    boundCleanup = (fn) => onWatcherCleanup(fn, false, effect2);
    cleanup = effect2.onStop = () => {
      const cleanups = cleanupMap.get(effect2);
      if (cleanups) {
        if (call) {
          call(cleanups, 4);
        } else {
          for (const cleanup2 of cleanups) cleanup2();
        }
        cleanupMap.delete(effect2);
      }
    };
    if (cb) {
      if (immediate) {
        job(true);
      } else {
        oldValue = effect2.run();
      }
    } else if (scheduler) {
      scheduler(job.bind(null, true), true);
    } else {
      effect2.run();
    }
    watchHandle.pause = effect2.pause.bind(effect2);
    watchHandle.resume = effect2.resume.bind(effect2);
    watchHandle.stop = watchHandle;
    return watchHandle;
  }
  function traverse(value, depth = Infinity, seen) {
    if (depth <= 0 || !isObject(value) || value["__v_skip"]) {
      return value;
    }
    seen = seen || /* @__PURE__ */ new Map();
    if ((seen.get(value) || 0) >= depth) {
      return value;
    }
    seen.set(value, depth);
    depth--;
    if (/* @__PURE__ */ isRef(value)) {
      traverse(value.value, depth, seen);
    } else if (isArray(value)) {
      for (let i = 0; i < value.length; i++) {
        traverse(value[i], depth, seen);
      }
    } else if (isSet(value) || isMap(value)) {
      value.forEach((v) => {
        traverse(v, depth, seen);
      });
    } else if (isPlainObject(value)) {
      for (const key in value) {
        traverse(value[key], depth, seen);
      }
      for (const key of Object.getOwnPropertySymbols(value)) {
        if (Object.prototype.propertyIsEnumerable.call(value, key)) {
          traverse(value[key], depth, seen);
        }
      }
    }
    return value;
  }
  /**
  * @vue/runtime-core v3.5.39
  * (c) 2018-present Yuxi (Evan) You and Vue contributors
  * @license MIT
  **/
  const stack = [];
  function pushWarningContext(vnode) {
    stack.push(vnode);
  }
  function popWarningContext() {
    stack.pop();
  }
  let isWarning = false;
  function warn$1(msg, ...args) {
    if (isWarning) return;
    isWarning = true;
    pauseTracking();
    const instance = stack.length ? stack[stack.length - 1].component : null;
    const appWarnHandler = instance && instance.appContext.config.warnHandler;
    const trace = getComponentTrace();
    if (appWarnHandler) {
      callWithErrorHandling(
        appWarnHandler,
        instance,
        11,
        [
          // eslint-disable-next-line no-restricted-syntax
          msg + args.map((a) => {
            var _a, _b;
            return (_b = (_a = a.toString) == null ? void 0 : _a.call(a)) != null ? _b : JSON.stringify(a);
          }).join(""),
          instance && instance.proxy,
          trace.map(
            ({ vnode }) => `at <${formatComponentName(instance, vnode.type)}>`
          ).join("\n"),
          trace
        ]
      );
    } else {
      const warnArgs = [`[Vue warn]: ${msg}`, ...args];
      if (trace.length && // avoid spamming console during tests
      true) {
        warnArgs.push(`
`, ...formatTrace(trace));
      }
      console.warn(...warnArgs);
    }
    resetTracking();
    isWarning = false;
  }
  function getComponentTrace() {
    let currentVNode = stack[stack.length - 1];
    if (!currentVNode) {
      return [];
    }
    const normalizedStack = [];
    while (currentVNode) {
      const last = normalizedStack[0];
      if (last && last.vnode === currentVNode) {
        last.recurseCount++;
      } else {
        normalizedStack.push({
          vnode: currentVNode,
          recurseCount: 0
        });
      }
      const parentInstance = currentVNode.component && currentVNode.component.parent;
      currentVNode = parentInstance && parentInstance.vnode;
    }
    return normalizedStack;
  }
  function formatTrace(trace) {
    const logs = [];
    trace.forEach((entry, i) => {
      logs.push(...i === 0 ? [] : [`
`], ...formatTraceEntry(entry));
    });
    return logs;
  }
  function formatTraceEntry({ vnode, recurseCount }) {
    const postfix = recurseCount > 0 ? `... (${recurseCount} recursive calls)` : ``;
    const isRoot = vnode.component ? vnode.component.parent == null : false;
    const open = ` at <${formatComponentName(
      vnode.component,
      vnode.type,
      isRoot
    )}`;
    const close = `>` + postfix;
    return vnode.props ? [open, ...formatProps(vnode.props), close] : [open + close];
  }
  function formatProps(props) {
    const res = [];
    const keys = Object.keys(props);
    keys.slice(0, 3).forEach((key) => {
      res.push(...formatProp(key, props[key]));
    });
    if (keys.length > 3) {
      res.push(` ...`);
    }
    return res;
  }
  function formatProp(key, value, raw) {
    if (isString(value)) {
      value = JSON.stringify(value);
      return raw ? value : [`${key}=${value}`];
    } else if (typeof value === "number" || typeof value === "boolean" || value == null) {
      return raw ? value : [`${key}=${value}`];
    } else if (/* @__PURE__ */ isRef(value)) {
      value = formatProp(key, /* @__PURE__ */ toRaw(value.value), true);
      return raw ? value : [`${key}=Ref<`, value, `>`];
    } else if (isFunction(value)) {
      return [`${key}=fn${value.name ? `<${value.name}>` : ``}`];
    } else {
      value = /* @__PURE__ */ toRaw(value);
      return raw ? value : [`${key}=`, value];
    }
  }
  function assertNumber(val, type) {
    return;
  }
  const ErrorCodes = {
    "SETUP_FUNCTION": 0,
    "0": "SETUP_FUNCTION",
    "RENDER_FUNCTION": 1,
    "1": "RENDER_FUNCTION",
    "NATIVE_EVENT_HANDLER": 5,
    "5": "NATIVE_EVENT_HANDLER",
    "COMPONENT_EVENT_HANDLER": 6,
    "6": "COMPONENT_EVENT_HANDLER",
    "VNODE_HOOK": 7,
    "7": "VNODE_HOOK",
    "DIRECTIVE_HOOK": 8,
    "8": "DIRECTIVE_HOOK",
    "TRANSITION_HOOK": 9,
    "9": "TRANSITION_HOOK",
    "APP_ERROR_HANDLER": 10,
    "10": "APP_ERROR_HANDLER",
    "APP_WARN_HANDLER": 11,
    "11": "APP_WARN_HANDLER",
    "FUNCTION_REF": 12,
    "12": "FUNCTION_REF",
    "ASYNC_COMPONENT_LOADER": 13,
    "13": "ASYNC_COMPONENT_LOADER",
    "SCHEDULER": 14,
    "14": "SCHEDULER",
    "COMPONENT_UPDATE": 15,
    "15": "COMPONENT_UPDATE",
    "APP_UNMOUNT_CLEANUP": 16,
    "16": "APP_UNMOUNT_CLEANUP"
  };
  const ErrorTypeStrings$1 = {
    ["sp"]: "serverPrefetch hook",
    ["bc"]: "beforeCreate hook",
    ["c"]: "created hook",
    ["bm"]: "beforeMount hook",
    ["m"]: "mounted hook",
    ["bu"]: "beforeUpdate hook",
    ["u"]: "updated",
    ["bum"]: "beforeUnmount hook",
    ["um"]: "unmounted hook",
    ["a"]: "activated hook",
    ["da"]: "deactivated hook",
    ["ec"]: "errorCaptured hook",
    ["rtc"]: "renderTracked hook",
    ["rtg"]: "renderTriggered hook",
    [0]: "setup function",
    [1]: "render function",
    [2]: "watcher getter",
    [3]: "watcher callback",
    [4]: "watcher cleanup function",
    [5]: "native event handler",
    [6]: "component event handler",
    [7]: "vnode hook",
    [8]: "directive hook",
    [9]: "transition hook",
    [10]: "app errorHandler",
    [11]: "app warnHandler",
    [12]: "ref function",
    [13]: "async component loader",
    [14]: "scheduler flush",
    [15]: "component update",
    [16]: "app unmount cleanup function"
  };
  function callWithErrorHandling(fn, instance, type, args) {
    try {
      return args ? fn(...args) : fn();
    } catch (err) {
      handleError(err, instance, type);
    }
  }
  function callWithAsyncErrorHandling(fn, instance, type, args) {
    if (isFunction(fn)) {
      const res = callWithErrorHandling(fn, instance, type, args);
      if (res && isPromise(res)) {
        res.catch((err) => {
          handleError(err, instance, type);
        });
      }
      return res;
    }
    if (isArray(fn)) {
      const values = [];
      for (let i = 0; i < fn.length; i++) {
        values.push(callWithAsyncErrorHandling(fn[i], instance, type, args));
      }
      return values;
    }
  }
  function handleError(err, instance, type, throwInDev = true) {
    const contextVNode = instance ? instance.vnode : null;
    const { errorHandler, throwUnhandledErrorInProduction } = instance && instance.appContext.config || EMPTY_OBJ;
    if (instance) {
      let cur = instance.parent;
      const exposedInstance = instance.proxy;
      const errorInfo = `https://vuejs.org/error-reference/#runtime-${type}`;
      while (cur) {
        const errorCapturedHooks = cur.ec;
        if (errorCapturedHooks) {
          for (let i = 0; i < errorCapturedHooks.length; i++) {
            if (errorCapturedHooks[i](err, exposedInstance, errorInfo) === false) {
              return;
            }
          }
        }
        cur = cur.parent;
      }
      if (errorHandler) {
        pauseTracking();
        callWithErrorHandling(errorHandler, null, 10, [
          err,
          exposedInstance,
          errorInfo
        ]);
        resetTracking();
        return;
      }
    }
    logError(err, type, contextVNode, throwInDev, throwUnhandledErrorInProduction);
  }
  function logError(err, type, contextVNode, throwInDev = true, throwInProd = false) {
    if (throwInProd) {
      throw err;
    } else {
      console.error(err);
    }
  }
  const queue = [];
  let flushIndex = -1;
  const pendingPostFlushCbs = [];
  let activePostFlushCbs = null;
  let postFlushIndex = 0;
  const resolvedPromise = /* @__PURE__ */ Promise.resolve();
  let currentFlushPromise = null;
  function nextTick(fn) {
    const p2 = currentFlushPromise || resolvedPromise;
    return fn ? p2.then(this ? fn.bind(this) : fn) : p2;
  }
  function findInsertionIndex(id) {
    let start = flushIndex + 1;
    let end = queue.length;
    while (start < end) {
      const middle = start + end >>> 1;
      const middleJob = queue[middle];
      const middleJobId = getId(middleJob);
      if (middleJobId < id || middleJobId === id && middleJob.flags & 2) {
        start = middle + 1;
      } else {
        end = middle;
      }
    }
    return start;
  }
  function queueJob(job) {
    if (!(job.flags & 1)) {
      const jobId = getId(job);
      const lastJob = queue[queue.length - 1];
      if (!lastJob || // fast path when the job id is larger than the tail
      !(job.flags & 2) && jobId >= getId(lastJob)) {
        queue.push(job);
      } else {
        queue.splice(findInsertionIndex(jobId), 0, job);
      }
      job.flags |= 1;
      queueFlush();
    }
  }
  function queueFlush() {
    if (!currentFlushPromise) {
      currentFlushPromise = resolvedPromise.then(flushJobs);
    }
  }
  function queuePostFlushCb(cb) {
    if (!isArray(cb)) {
      if (activePostFlushCbs && cb.id === -1) {
        activePostFlushCbs.splice(postFlushIndex + 1, 0, cb);
      } else if (!(cb.flags & 1)) {
        pendingPostFlushCbs.push(cb);
        cb.flags |= 1;
      }
    } else {
      pendingPostFlushCbs.push(...cb);
    }
    queueFlush();
  }
  function flushPreFlushCbs(instance, seen, i = flushIndex + 1) {
    for (; i < queue.length; i++) {
      const cb = queue[i];
      if (cb && cb.flags & 2) {
        if (instance && cb.id !== instance.uid) {
          continue;
        }
        queue.splice(i, 1);
        i--;
        if (cb.flags & 4) {
          cb.flags &= -2;
        }
        cb();
        if (!(cb.flags & 4)) {
          cb.flags &= -2;
        }
      }
    }
  }
  function flushPostFlushCbs(seen) {
    if (pendingPostFlushCbs.length) {
      const deduped = [...new Set(pendingPostFlushCbs)].sort(
        (a, b) => getId(a) - getId(b)
      );
      pendingPostFlushCbs.length = 0;
      if (activePostFlushCbs) {
        activePostFlushCbs.push(...deduped);
        return;
      }
      activePostFlushCbs = deduped;
      for (postFlushIndex = 0; postFlushIndex < activePostFlushCbs.length; postFlushIndex++) {
        const cb = activePostFlushCbs[postFlushIndex];
        if (cb.flags & 4) {
          cb.flags &= -2;
        }
        if (!(cb.flags & 8)) cb();
        cb.flags &= -2;
      }
      activePostFlushCbs = null;
      postFlushIndex = 0;
    }
  }
  const getId = (job) => job.id == null ? job.flags & 2 ? -1 : Infinity : job.id;
  function flushJobs(seen) {
    try {
      for (flushIndex = 0; flushIndex < queue.length; flushIndex++) {
        const job = queue[flushIndex];
        if (job && !(job.flags & 8)) {
          if (false) ;
          if (job.flags & 4) {
            job.flags &= ~1;
          }
          callWithErrorHandling(
            job,
            job.i,
            job.i ? 15 : 14
          );
          if (!(job.flags & 4)) {
            job.flags &= ~1;
          }
        }
      }
    } finally {
      for (; flushIndex < queue.length; flushIndex++) {
        const job = queue[flushIndex];
        if (job) {
          job.flags &= -2;
        }
      }
      flushIndex = -1;
      queue.length = 0;
      flushPostFlushCbs();
      currentFlushPromise = null;
      if (queue.length || pendingPostFlushCbs.length) {
        flushJobs();
      }
    }
  }
  let devtools$1;
  let buffer = [];
  function setDevtoolsHook$1(hook, target) {
    var _a, _b;
    devtools$1 = hook;
    if (devtools$1) {
      devtools$1.enabled = true;
      buffer.forEach(({ event, args }) => devtools$1.emit(event, ...args));
      buffer = [];
    } else if (
      // handle late devtools injection - only do this if we are in an actual
      // browser environment to avoid the timer handle stalling test runner exit
      // (#4815)
      typeof window !== "undefined" && // some envs mock window but not fully
      window.HTMLElement && // also exclude jsdom
      // eslint-disable-next-line no-restricted-syntax
      !((_b = (_a = window.navigator) == null ? void 0 : _a.userAgent) == null ? void 0 : _b.includes("jsdom"))
    ) {
      const replay = target.__VUE_DEVTOOLS_HOOK_REPLAY__ = target.__VUE_DEVTOOLS_HOOK_REPLAY__ || [];
      replay.push((newHook) => {
        setDevtoolsHook$1(newHook, target);
      });
      setTimeout(() => {
        if (!devtools$1) {
          target.__VUE_DEVTOOLS_HOOK_REPLAY__ = null;
          buffer = [];
        }
      }, 3e3);
    } else {
      buffer = [];
    }
  }
  let currentRenderingInstance = null;
  let currentScopeId = null;
  function setCurrentRenderingInstance(instance) {
    const prev = currentRenderingInstance;
    currentRenderingInstance = instance;
    currentScopeId = instance && instance.type.__scopeId || null;
    return prev;
  }
  function pushScopeId(id) {
    currentScopeId = id;
  }
  function popScopeId() {
    currentScopeId = null;
  }
  const withScopeId = (_id) => withCtx;
  function withCtx(fn, ctx = currentRenderingInstance, isNonScopedSlot) {
    if (!ctx) return fn;
    if (fn._n) {
      return fn;
    }
    const renderFnWithContext = (...args) => {
      if (renderFnWithContext._d) {
        setBlockTracking(-1);
      }
      const prevInstance = setCurrentRenderingInstance(ctx);
      let res;
      try {
        res = fn(...args);
      } finally {
        setCurrentRenderingInstance(prevInstance);
        if (renderFnWithContext._d) {
          setBlockTracking(1);
        }
      }
      return res;
    };
    renderFnWithContext._n = true;
    renderFnWithContext._c = true;
    renderFnWithContext._d = true;
    return renderFnWithContext;
  }
  function withDirectives(vnode, directives) {
    if (currentRenderingInstance === null) {
      return vnode;
    }
    const instance = getComponentPublicInstance(currentRenderingInstance);
    const bindings = vnode.dirs || (vnode.dirs = []);
    for (let i = 0; i < directives.length; i++) {
      let [dir, value, arg, modifiers = EMPTY_OBJ] = directives[i];
      if (dir) {
        if (isFunction(dir)) {
          dir = {
            mounted: dir,
            updated: dir
          };
        }
        if (dir.deep) {
          traverse(value);
        }
        bindings.push({
          dir,
          instance,
          value,
          oldValue: void 0,
          arg,
          modifiers
        });
      }
    }
    return vnode;
  }
  function invokeDirectiveHook(vnode, prevVNode, instance, name) {
    const bindings = vnode.dirs;
    const oldBindings = prevVNode && prevVNode.dirs;
    for (let i = 0; i < bindings.length; i++) {
      const binding = bindings[i];
      if (oldBindings) {
        binding.oldValue = oldBindings[i].value;
      }
      let hook = binding.dir[name];
      if (hook) {
        pauseTracking();
        callWithAsyncErrorHandling(hook, instance, 8, [
          vnode.el,
          binding,
          vnode,
          prevVNode
        ]);
        resetTracking();
      }
    }
  }
  function provide(key, value) {
    if (currentInstance) {
      let provides = currentInstance.provides;
      const parentProvides = currentInstance.parent && currentInstance.parent.provides;
      if (parentProvides === provides) {
        provides = currentInstance.provides = Object.create(parentProvides);
      }
      provides[key] = value;
    }
  }
  function inject(key, defaultValue, treatDefaultAsFactory = false) {
    const instance = getCurrentInstance();
    if (instance || currentApp) {
      let provides = currentApp ? currentApp._context.provides : instance ? instance.parent == null || instance.ce ? instance.vnode.appContext && instance.vnode.appContext.provides : instance.parent.provides : void 0;
      if (provides && key in provides) {
        return provides[key];
      } else if (arguments.length > 1) {
        return treatDefaultAsFactory && isFunction(defaultValue) ? defaultValue.call(instance && instance.proxy) : defaultValue;
      } else ;
    }
  }
  function hasInjectionContext() {
    return !!(getCurrentInstance() || currentApp);
  }
  const ssrContextKey = /* @__PURE__ */ Symbol.for("v-scx");
  const useSSRContext = () => {
    {
      const ctx = inject(ssrContextKey);
      return ctx;
    }
  };
  function watchEffect(effect2, options) {
    return doWatch(effect2, null, options);
  }
  function watchPostEffect(effect2, options) {
    return doWatch(
      effect2,
      null,
      { flush: "post" }
    );
  }
  function watchSyncEffect(effect2, options) {
    return doWatch(
      effect2,
      null,
      { flush: "sync" }
    );
  }
  function watch(source, cb, options) {
    return doWatch(source, cb, options);
  }
  function doWatch(source, cb, options = EMPTY_OBJ) {
    const { immediate, deep, flush, once } = options;
    const baseWatchOptions = extend({}, options);
    const runsImmediately = cb && immediate || !cb && flush !== "post";
    let ssrCleanup;
    if (isInSSRComponentSetup) {
      if (flush === "sync") {
        const ctx = useSSRContext();
        ssrCleanup = ctx.__watcherHandles || (ctx.__watcherHandles = []);
      } else if (!runsImmediately) {
        const watchStopHandle = () => {
        };
        watchStopHandle.stop = NOOP;
        watchStopHandle.resume = NOOP;
        watchStopHandle.pause = NOOP;
        return watchStopHandle;
      }
    }
    const instance = currentInstance;
    baseWatchOptions.call = (fn, type, args) => callWithAsyncErrorHandling(fn, instance, type, args);
    let isPre = false;
    if (flush === "post") {
      baseWatchOptions.scheduler = (job) => {
        queuePostRenderEffect(job, instance && instance.suspense);
      };
    } else if (flush !== "sync") {
      isPre = true;
      baseWatchOptions.scheduler = (job, isFirstRun) => {
        if (isFirstRun) {
          job();
        } else {
          queueJob(job);
        }
      };
    }
    baseWatchOptions.augmentJob = (job) => {
      if (cb) {
        job.flags |= 4;
      }
      if (isPre) {
        job.flags |= 2;
        if (instance) {
          job.id = instance.uid;
          job.i = instance;
        }
      }
    };
    const watchHandle = watch$1(source, cb, baseWatchOptions);
    if (isInSSRComponentSetup) {
      if (ssrCleanup) {
        ssrCleanup.push(watchHandle);
      } else if (runsImmediately) {
        watchHandle();
      }
    }
    return watchHandle;
  }
  function instanceWatch(source, value, options) {
    const publicThis = this.proxy;
    const getter = isString(source) ? source.includes(".") ? createPathGetter(publicThis, source) : () => publicThis[source] : source.bind(publicThis, publicThis);
    let cb;
    if (isFunction(value)) {
      cb = value;
    } else {
      cb = value.handler;
      options = value;
    }
    const reset = setCurrentInstance(this);
    const res = doWatch(getter, cb.bind(publicThis), options);
    reset();
    return res;
  }
  function createPathGetter(ctx, path) {
    const segments = path.split(".");
    return () => {
      let cur = ctx;
      for (let i = 0; i < segments.length && cur; i++) {
        cur = cur[segments[i]];
      }
      return cur;
    };
  }
  const pendingMounts = /* @__PURE__ */ new WeakMap();
  const TeleportEndKey = /* @__PURE__ */ Symbol("_vte");
  const isTeleport = (type) => type.__isTeleport;
  const isTeleportDisabled = (props) => props && (props.disabled || props.disabled === "");
  const isTeleportDeferred = (props) => props && (props.defer || props.defer === "");
  const isTargetSVG = (target) => typeof SVGElement !== "undefined" && target instanceof SVGElement;
  const isTargetMathML = (target) => typeof MathMLElement === "function" && target instanceof MathMLElement;
  const resolveTarget = (props, select) => {
    const targetSelector = props && props.to;
    if (isString(targetSelector)) {
      if (!select) {
        return null;
      } else {
        const target = select(targetSelector);
        return target;
      }
    } else {
      return targetSelector;
    }
  };
  const TeleportImpl = {
    name: "Teleport",
    __isTeleport: true,
    process(n1, n2, container, anchor, parentComponent, parentSuspense, namespace, slotScopeIds, optimized, internals) {
      const {
        mc: mountChildren,
        pc: patchChildren,
        pbc: patchBlockChildren,
        o: { insert, querySelector, createText, createComment, parentNode }
      } = internals;
      const disabled = isTeleportDisabled(n2.props);
      let { dynamicChildren } = n2;
      const mount = (vnode, container2, anchor2) => {
        if (vnode.shapeFlag & 16) {
          mountChildren(
            vnode.children,
            container2,
            anchor2,
            parentComponent,
            parentSuspense,
            namespace,
            slotScopeIds,
            optimized
          );
        }
      };
      const mountToTarget = (vnode = n2) => {
        const disabled2 = isTeleportDisabled(vnode.props);
        const target = vnode.target = resolveTarget(vnode.props, querySelector);
        const targetAnchor = prepareAnchor(target, vnode, createText, insert);
        if (target) {
          if (namespace !== "svg" && isTargetSVG(target)) {
            namespace = "svg";
          } else if (namespace !== "mathml" && isTargetMathML(target)) {
            namespace = "mathml";
          }
          if (parentComponent && parentComponent.isCE) {
            (parentComponent.ce._teleportTargets || (parentComponent.ce._teleportTargets = /* @__PURE__ */ new Set())).add(target);
          }
          if (!disabled2) {
            mount(vnode, target, targetAnchor);
            updateCssVars(vnode, false);
          }
        }
      };
      const queuePendingMount = (vnode) => {
        const mountJob = () => {
          if (pendingMounts.get(vnode) !== mountJob) return;
          pendingMounts.delete(vnode);
          if (isTeleportDisabled(vnode.props)) {
            const mountContainer = parentNode(vnode.el) || container;
            mount(vnode, mountContainer, vnode.anchor);
            updateCssVars(vnode, true);
          }
          mountToTarget(vnode);
        };
        pendingMounts.set(vnode, mountJob);
        queuePostRenderEffect(mountJob, parentSuspense);
      };
      if (n1 == null) {
        const placeholder = n2.el = createText("");
        const mainAnchor = n2.anchor = createText("");
        insert(placeholder, container, anchor);
        insert(mainAnchor, container, anchor);
        if (isTeleportDeferred(n2.props) || parentSuspense && parentSuspense.pendingBranch) {
          queuePendingMount(n2);
          return;
        }
        if (disabled) {
          mount(n2, container, mainAnchor);
          updateCssVars(n2, true);
        }
        mountToTarget();
      } else {
        n2.el = n1.el;
        const mainAnchor = n2.anchor = n1.anchor;
        const pendingMount = pendingMounts.get(n1);
        if (pendingMount) {
          pendingMount.flags |= 8;
          pendingMounts.delete(n1);
          queuePendingMount(n2);
          return;
        }
        n2.targetStart = n1.targetStart;
        const target = n2.target = n1.target;
        const targetAnchor = n2.targetAnchor = n1.targetAnchor;
        const wasDisabled = isTeleportDisabled(n1.props);
        const currentContainer = wasDisabled ? container : target;
        const currentAnchor = wasDisabled ? mainAnchor : targetAnchor;
        if (namespace === "svg" || isTargetSVG(target)) {
          namespace = "svg";
        } else if (namespace === "mathml" || isTargetMathML(target)) {
          namespace = "mathml";
        }
        if (dynamicChildren) {
          patchBlockChildren(
            n1.dynamicChildren,
            dynamicChildren,
            currentContainer,
            parentComponent,
            parentSuspense,
            namespace,
            slotScopeIds
          );
          traverseStaticChildren(n1, n2, true);
        } else if (!optimized) {
          patchChildren(
            n1,
            n2,
            currentContainer,
            currentAnchor,
            parentComponent,
            parentSuspense,
            namespace,
            slotScopeIds,
            false
          );
        }
        if (disabled) {
          if (!wasDisabled) {
            moveTeleport(
              n2,
              container,
              mainAnchor,
              internals,
              1
            );
          } else {
            if (n2.props && n1.props && n2.props.to !== n1.props.to) {
              n2.props.to = n1.props.to;
            }
          }
        } else {
          if ((n2.props && n2.props.to) !== (n1.props && n1.props.to)) {
            const nextTarget = resolveTarget(n2.props, querySelector);
            if (nextTarget) {
              n2.target = nextTarget;
              moveTeleport(
                n2,
                nextTarget,
                null,
                internals,
                0
              );
            }
          } else if (wasDisabled) {
            moveTeleport(
              n2,
              target,
              targetAnchor,
              internals,
              1
            );
          }
        }
        updateCssVars(n2, disabled);
      }
    },
    remove(vnode, parentComponent, parentSuspense, { um: unmount, o: { remove: hostRemove } }, doRemove) {
      const {
        shapeFlag,
        children,
        anchor,
        targetStart,
        targetAnchor,
        target,
        props
      } = vnode;
      const disabled = isTeleportDisabled(props);
      const shouldRemove = doRemove || !disabled;
      const pendingMount = pendingMounts.get(vnode);
      if (pendingMount) {
        pendingMount.flags |= 8;
        pendingMounts.delete(vnode);
      }
      if (target) {
        hostRemove(targetStart);
        hostRemove(targetAnchor);
      }
      doRemove && hostRemove(anchor);
      if (!pendingMount && (disabled || target) && shapeFlag & 16) {
        for (let i = 0; i < children.length; i++) {
          const child = children[i];
          unmount(
            child,
            parentComponent,
            parentSuspense,
            shouldRemove,
            !!child.dynamicChildren
          );
        }
      }
    },
    move: moveTeleport,
    hydrate: hydrateTeleport
  };
  function moveTeleport(vnode, container, parentAnchor, { o: { insert }, m: move }, moveType = 2) {
    if (moveType === 0) {
      insert(vnode.targetAnchor, container, parentAnchor);
    }
    const { el, anchor, shapeFlag, children, props } = vnode;
    const isReorder = moveType === 2;
    if (isReorder) {
      insert(el, container, parentAnchor);
    }
    if (!pendingMounts.has(vnode) && (!isReorder || isTeleportDisabled(props))) {
      if (shapeFlag & 16) {
        for (let i = 0; i < children.length; i++) {
          move(
            children[i],
            container,
            parentAnchor,
            2
          );
        }
      }
    }
    if (isReorder) {
      insert(anchor, container, parentAnchor);
    }
  }
  function hydrateTeleport(node, vnode, parentComponent, parentSuspense, slotScopeIds, optimized, {
    o: { nextSibling, parentNode, querySelector, insert, createText }
  }, hydrateChildren) {
    function hydrateAnchor(target2, targetNode) {
      let targetAnchor = targetNode;
      while (targetAnchor) {
        if (targetAnchor && targetAnchor.nodeType === 8) {
          if (targetAnchor.data === "teleport start anchor") {
            vnode.targetStart = targetAnchor;
          } else if (targetAnchor.data === "teleport anchor") {
            vnode.targetAnchor = targetAnchor;
            target2._lpa = vnode.targetAnchor && nextSibling(vnode.targetAnchor);
            break;
          }
        }
        targetAnchor = nextSibling(targetAnchor);
      }
    }
    function hydrateDisabledTeleport(node2, vnode2) {
      vnode2.anchor = hydrateChildren(
        nextSibling(node2),
        vnode2,
        parentNode(node2),
        parentComponent,
        parentSuspense,
        slotScopeIds,
        optimized
      );
    }
    const target = vnode.target = resolveTarget(
      vnode.props,
      querySelector
    );
    const disabled = isTeleportDisabled(vnode.props);
    if (target) {
      const targetNode = target._lpa || target.firstChild;
      if (vnode.shapeFlag & 16) {
        if (disabled) {
          hydrateDisabledTeleport(node, vnode);
          hydrateAnchor(target, targetNode);
          if (!vnode.targetAnchor) {
            prepareAnchor(
              target,
              vnode,
              createText,
              insert,
              // if target is the same as the main view, insert anchors before current node
              // to avoid hydrating mismatch
              parentNode(node) === target ? node : null
            );
          }
        } else {
          vnode.anchor = nextSibling(node);
          hydrateAnchor(target, targetNode);
          if (!vnode.targetAnchor) {
            prepareAnchor(target, vnode, createText, insert);
          }
          hydrateChildren(
            targetNode && nextSibling(targetNode),
            vnode,
            target,
            parentComponent,
            parentSuspense,
            slotScopeIds,
            optimized
          );
        }
      }
      updateCssVars(vnode, disabled);
    } else if (disabled) {
      if (vnode.shapeFlag & 16) {
        hydrateDisabledTeleport(node, vnode);
        vnode.targetStart = node;
        vnode.targetAnchor = nextSibling(node);
      }
    }
    return vnode.anchor && nextSibling(vnode.anchor);
  }
  const Teleport = TeleportImpl;
  function updateCssVars(vnode, isDisabled) {
    const ctx = vnode.ctx;
    if (ctx && ctx.ut) {
      let node, anchor;
      if (isDisabled) {
        node = vnode.el;
        anchor = vnode.anchor;
      } else {
        node = vnode.targetStart;
        anchor = vnode.targetAnchor;
      }
      while (node && node !== anchor) {
        if (node.nodeType === 1) node.setAttribute("data-v-owner", ctx.uid);
        node = node.nextSibling;
      }
      ctx.ut();
    }
  }
  function prepareAnchor(target, vnode, createText, insert, anchor = null) {
    const targetStart = vnode.targetStart = createText("");
    const targetAnchor = vnode.targetAnchor = createText("");
    targetStart[TeleportEndKey] = targetAnchor;
    if (target) {
      insert(targetStart, target, anchor);
      insert(targetAnchor, target, anchor);
    }
    return targetAnchor;
  }
  const leaveCbKey = /* @__PURE__ */ Symbol("_leaveCb");
  const enterCbKey$1 = /* @__PURE__ */ Symbol("_enterCb");
  function useTransitionState() {
    const state2 = {
      isMounted: false,
      isLeaving: false,
      isUnmounting: false,
      leavingVNodes: /* @__PURE__ */ new Map()
    };
    onMounted(() => {
      state2.isMounted = true;
    });
    onBeforeUnmount(() => {
      state2.isUnmounting = true;
    });
    return state2;
  }
  const TransitionHookValidator = [Function, Array];
  const BaseTransitionPropsValidators = {
    mode: String,
    appear: Boolean,
    persisted: Boolean,
    // enter
    onBeforeEnter: TransitionHookValidator,
    onEnter: TransitionHookValidator,
    onAfterEnter: TransitionHookValidator,
    onEnterCancelled: TransitionHookValidator,
    // leave
    onBeforeLeave: TransitionHookValidator,
    onLeave: TransitionHookValidator,
    onAfterLeave: TransitionHookValidator,
    onLeaveCancelled: TransitionHookValidator,
    // appear
    onBeforeAppear: TransitionHookValidator,
    onAppear: TransitionHookValidator,
    onAfterAppear: TransitionHookValidator,
    onAppearCancelled: TransitionHookValidator
  };
  const recursiveGetSubtree = (instance) => {
    const subTree = instance.subTree;
    return subTree.component ? recursiveGetSubtree(subTree.component) : subTree;
  };
  const BaseTransitionImpl = {
    name: `BaseTransition`,
    props: BaseTransitionPropsValidators,
    setup(props, { slots }) {
      const instance = getCurrentInstance();
      const state2 = useTransitionState();
      return () => {
        const children = slots.default && getTransitionRawChildren(slots.default(), true);
        const child = children && children.length ? findNonCommentChild(children) : (
          // Keep explicit default-slot conditionals on the same transition path
          // as regular v-if branches, which render a comment placeholder.
          instance.subTree ? createCommentVNode() : void 0
        );
        if (!child) {
          return;
        }
        const rawProps = /* @__PURE__ */ toRaw(props);
        const { mode } = rawProps;
        if (state2.isLeaving) {
          return emptyPlaceholder(child);
        }
        const innerChild = getInnerChild$1(child);
        if (!innerChild) {
          return emptyPlaceholder(child);
        }
        let enterHooks = resolveTransitionHooks(
          innerChild,
          rawProps,
          state2,
          instance,
          // #11061, ensure enterHooks is fresh after clone
          (hooks) => enterHooks = hooks
        );
        if (innerChild.type !== Comment) {
          setTransitionHooks(innerChild, enterHooks);
        }
        let oldInnerChild = instance.subTree && getInnerChild$1(instance.subTree);
        if (oldInnerChild && oldInnerChild.type !== Comment && !isSameVNodeType(oldInnerChild, innerChild) && recursiveGetSubtree(instance).type !== Comment) {
          let leavingHooks = resolveTransitionHooks(
            oldInnerChild,
            rawProps,
            state2,
            instance
          );
          setTransitionHooks(oldInnerChild, leavingHooks);
          if (mode === "out-in" && innerChild.type !== Comment) {
            state2.isLeaving = true;
            leavingHooks.afterLeave = () => {
              state2.isLeaving = false;
              if (!(instance.job.flags & 8)) {
                instance.update();
              }
              delete leavingHooks.afterLeave;
              oldInnerChild = void 0;
            };
            return emptyPlaceholder(child);
          } else if (mode === "in-out" && innerChild.type !== Comment) {
            leavingHooks.delayLeave = (el, earlyRemove, delayedLeave) => {
              const leavingVNodesCache = getLeavingNodesForType(
                state2,
                oldInnerChild
              );
              leavingVNodesCache[String(oldInnerChild.key)] = oldInnerChild;
              el[leaveCbKey] = () => {
                earlyRemove();
                el[leaveCbKey] = void 0;
                delete enterHooks.delayedLeave;
                oldInnerChild = void 0;
              };
              enterHooks.delayedLeave = () => {
                delayedLeave();
                delete enterHooks.delayedLeave;
                oldInnerChild = void 0;
              };
            };
          } else {
            oldInnerChild = void 0;
          }
        } else if (oldInnerChild) {
          oldInnerChild = void 0;
        }
        return child;
      };
    }
  };
  function findNonCommentChild(children) {
    let child = children[0];
    if (children.length > 1) {
      for (const c of children) {
        if (c.type !== Comment) {
          child = c;
          break;
        }
      }
    }
    return child;
  }
  const BaseTransition = BaseTransitionImpl;
  function getLeavingNodesForType(state2, vnode) {
    const { leavingVNodes } = state2;
    let leavingVNodesCache = leavingVNodes.get(vnode.type);
    if (!leavingVNodesCache) {
      leavingVNodesCache = /* @__PURE__ */ Object.create(null);
      leavingVNodes.set(vnode.type, leavingVNodesCache);
    }
    return leavingVNodesCache;
  }
  function resolveTransitionHooks(vnode, props, state2, instance, postClone) {
    const {
      appear,
      mode,
      persisted = false,
      onBeforeEnter,
      onEnter,
      onAfterEnter,
      onEnterCancelled,
      onBeforeLeave,
      onLeave,
      onAfterLeave,
      onLeaveCancelled,
      onBeforeAppear,
      onAppear,
      onAfterAppear,
      onAppearCancelled
    } = props;
    const key = String(vnode.key);
    const leavingVNodesCache = getLeavingNodesForType(state2, vnode);
    const callHook2 = (hook, args) => {
      hook && callWithAsyncErrorHandling(
        hook,
        instance,
        9,
        args
      );
    };
    const callAsyncHook = (hook, args) => {
      const done = args[1];
      callHook2(hook, args);
      if (isArray(hook)) {
        if (hook.every((hook2) => hook2.length <= 1)) done();
      } else if (hook.length <= 1) {
        done();
      }
    };
    const hooks = {
      mode,
      persisted,
      beforeEnter(el) {
        let hook = onBeforeEnter;
        if (!state2.isMounted) {
          if (appear) {
            hook = onBeforeAppear || onBeforeEnter;
          } else {
            return;
          }
        }
        if (el[leaveCbKey]) {
          el[leaveCbKey](
            true
            /* cancelled */
          );
        }
        const leavingVNode = leavingVNodesCache[key];
        if (leavingVNode && isSameVNodeType(vnode, leavingVNode) && leavingVNode.el[leaveCbKey]) {
          leavingVNode.el[leaveCbKey]();
        }
        callHook2(hook, [el]);
      },
      enter(el) {
        if (leavingVNodesCache[key] === vnode) return;
        let hook = onEnter;
        let afterHook = onAfterEnter;
        let cancelHook = onEnterCancelled;
        if (!state2.isMounted) {
          if (appear) {
            hook = onAppear || onEnter;
            afterHook = onAfterAppear || onAfterEnter;
            cancelHook = onAppearCancelled || onEnterCancelled;
          } else {
            return;
          }
        }
        let called = false;
        el[enterCbKey$1] = (cancelled) => {
          if (called) return;
          called = true;
          if (cancelled) {
            callHook2(cancelHook, [el]);
          } else {
            callHook2(afterHook, [el]);
          }
          if (hooks.delayedLeave) {
            hooks.delayedLeave();
          }
          el[enterCbKey$1] = void 0;
        };
        const done = el[enterCbKey$1].bind(null, false);
        if (hook) {
          callAsyncHook(hook, [el, done]);
        } else {
          done();
        }
      },
      leave(el, remove2) {
        const key2 = String(vnode.key);
        if (el[enterCbKey$1]) {
          el[enterCbKey$1](
            true
            /* cancelled */
          );
        }
        if (state2.isUnmounting) {
          return remove2();
        }
        callHook2(onBeforeLeave, [el]);
        let called = false;
        el[leaveCbKey] = (cancelled) => {
          if (called) return;
          called = true;
          remove2();
          if (cancelled) {
            callHook2(onLeaveCancelled, [el]);
          } else {
            callHook2(onAfterLeave, [el]);
          }
          el[leaveCbKey] = void 0;
          if (leavingVNodesCache[key2] === vnode) {
            delete leavingVNodesCache[key2];
          }
        };
        const done = el[leaveCbKey].bind(null, false);
        leavingVNodesCache[key2] = vnode;
        if (onLeave) {
          callAsyncHook(onLeave, [el, done]);
        } else {
          done();
        }
      },
      clone(vnode2) {
        const hooks2 = resolveTransitionHooks(
          vnode2,
          props,
          state2,
          instance,
          postClone
        );
        if (postClone) postClone(hooks2);
        return hooks2;
      }
    };
    return hooks;
  }
  function emptyPlaceholder(vnode) {
    if (isKeepAlive(vnode)) {
      vnode = cloneVNode(vnode);
      vnode.children = null;
      return vnode;
    }
  }
  function getInnerChild$1(vnode) {
    if (!isKeepAlive(vnode)) {
      if (isTeleport(vnode.type) && vnode.children) {
        return findNonCommentChild(vnode.children);
      }
      return vnode;
    }
    if (vnode.component) {
      return vnode.component.subTree;
    }
    const { shapeFlag, children } = vnode;
    if (children) {
      if (shapeFlag & 16) {
        return children[0];
      }
      if (shapeFlag & 32 && isFunction(children.default)) {
        return children.default();
      }
    }
  }
  function setTransitionHooks(vnode, hooks) {
    if (vnode.shapeFlag & 6 && vnode.component) {
      vnode.transition = hooks;
      setTransitionHooks(vnode.component.subTree, hooks);
    } else if (vnode.shapeFlag & 128) {
      vnode.ssContent.transition = hooks.clone(vnode.ssContent);
      vnode.ssFallback.transition = hooks.clone(vnode.ssFallback);
    } else {
      vnode.transition = hooks;
    }
  }
  function getTransitionRawChildren(children, keepComment = false, parentKey) {
    let ret = [];
    let keyedFragmentCount = 0;
    for (let i = 0; i < children.length; i++) {
      let child = children[i];
      const key = parentKey == null ? child.key : String(parentKey) + String(child.key != null ? child.key : i);
      if (child.type === Fragment) {
        if (child.patchFlag & 128) keyedFragmentCount++;
        ret = ret.concat(
          getTransitionRawChildren(child.children, keepComment, key)
        );
      } else if (keepComment || child.type !== Comment) {
        ret.push(key != null ? cloneVNode(child, { key }) : child);
      }
    }
    if (keyedFragmentCount > 1) {
      for (let i = 0; i < ret.length; i++) {
        ret[i].patchFlag = -2;
      }
    }
    return ret;
  }
  // @__NO_SIDE_EFFECTS__
  function defineComponent(options, extraOptions) {
    return isFunction(options) ? (
      // #8236: extend call and options.name access are considered side-effects
      // by Rollup, so we have to wrap it in a pure-annotated IIFE.
      /* @__PURE__ */ (() => extend({ name: options.name }, extraOptions, { setup: options }))()
    ) : options;
  }
  function useId() {
    const i = getCurrentInstance();
    if (i) {
      return (i.appContext.config.idPrefix || "v") + "-" + i.ids[0] + i.ids[1]++;
    }
    return "";
  }
  function markAsyncBoundary(instance) {
    instance.ids = [instance.ids[0] + instance.ids[2]++ + "-", 0, 0];
  }
  function useTemplateRef(key) {
    const i = getCurrentInstance();
    const r = /* @__PURE__ */ shallowRef(null);
    if (i) {
      const refs = i.refs === EMPTY_OBJ ? i.refs = {} : i.refs;
      {
        Object.defineProperty(refs, key, {
          enumerable: true,
          get: () => r.value,
          set: (val) => r.value = val
        });
      }
    }
    const ret = r;
    return ret;
  }
  function isTemplateRefKey(refs, key) {
    let desc;
    return !!((desc = Object.getOwnPropertyDescriptor(refs, key)) && !desc.configurable);
  }
  const pendingSetRefMap = /* @__PURE__ */ new WeakMap();
  function setRef(rawRef, oldRawRef, parentSuspense, vnode, isUnmount = false) {
    if (isArray(rawRef)) {
      rawRef.forEach(
        (r, i) => setRef(
          r,
          oldRawRef && (isArray(oldRawRef) ? oldRawRef[i] : oldRawRef),
          parentSuspense,
          vnode,
          isUnmount
        )
      );
      return;
    }
    if (isAsyncWrapper(vnode) && !isUnmount) {
      if (vnode.shapeFlag & 512 && vnode.type.__asyncResolved && vnode.component.subTree.component) {
        setRef(rawRef, oldRawRef, parentSuspense, vnode.component.subTree);
      }
      return;
    }
    const refValue = vnode.shapeFlag & 4 ? getComponentPublicInstance(vnode.component) : vnode.el;
    const value = isUnmount ? null : refValue;
    const { i: owner, r: ref3 } = rawRef;
    const oldRef = oldRawRef && oldRawRef.r;
    const refs = owner.refs === EMPTY_OBJ ? owner.refs = {} : owner.refs;
    const setupState = owner.setupState;
    const rawSetupState = /* @__PURE__ */ toRaw(setupState);
    const canSetSetupRef = setupState === EMPTY_OBJ ? NO : (key) => {
      if (isTemplateRefKey(refs, key)) {
        return false;
      }
      return hasOwn(rawSetupState, key);
    };
    const canSetRef = (ref22, key) => {
      if (key && isTemplateRefKey(refs, key)) {
        return false;
      }
      return true;
    };
    if (oldRef != null && oldRef !== ref3) {
      invalidatePendingSetRef(oldRawRef);
      if (isString(oldRef)) {
        refs[oldRef] = null;
        if (canSetSetupRef(oldRef)) {
          setupState[oldRef] = null;
        }
      } else if (/* @__PURE__ */ isRef(oldRef)) {
        const oldRawRefAtom = oldRawRef;
        if (canSetRef(oldRef, oldRawRefAtom.k)) {
          oldRef.value = null;
        }
        if (oldRawRefAtom.k) refs[oldRawRefAtom.k] = null;
      }
    }
    if (isFunction(ref3)) {
      pauseTracking();
      try {
        callWithErrorHandling(ref3, owner, 12, [value, refs]);
      } finally {
        resetTracking();
      }
    } else {
      const _isString = isString(ref3);
      const _isRef = /* @__PURE__ */ isRef(ref3);
      if (_isString || _isRef) {
        const doSet = () => {
          if (rawRef.f) {
            const existing = _isString ? canSetSetupRef(ref3) ? setupState[ref3] : refs[ref3] : canSetRef() || !rawRef.k ? ref3.value : refs[rawRef.k];
            if (isUnmount) {
              isArray(existing) && remove(existing, refValue);
            } else {
              if (!isArray(existing)) {
                if (_isString) {
                  refs[ref3] = [refValue];
                  if (canSetSetupRef(ref3)) {
                    setupState[ref3] = refs[ref3];
                  }
                } else {
                  const newVal = [refValue];
                  if (canSetRef(ref3, rawRef.k)) {
                    ref3.value = newVal;
                  }
                  if (rawRef.k) refs[rawRef.k] = newVal;
                }
              } else if (!existing.includes(refValue)) {
                existing.push(refValue);
              }
            }
          } else if (_isString) {
            refs[ref3] = value;
            if (canSetSetupRef(ref3)) {
              setupState[ref3] = value;
            }
          } else if (_isRef) {
            if (canSetRef(ref3, rawRef.k)) {
              ref3.value = value;
            }
            if (rawRef.k) refs[rawRef.k] = value;
          } else ;
        };
        if (value) {
          const job = () => {
            doSet();
            pendingSetRefMap.delete(rawRef);
          };
          job.id = -1;
          pendingSetRefMap.set(rawRef, job);
          queuePostRenderEffect(job, parentSuspense);
        } else {
          invalidatePendingSetRef(rawRef);
          doSet();
        }
      }
    }
  }
  function invalidatePendingSetRef(rawRef) {
    const pendingSetRef = pendingSetRefMap.get(rawRef);
    if (pendingSetRef) {
      pendingSetRef.flags |= 8;
      pendingSetRefMap.delete(rawRef);
    }
  }
  let hasLoggedMismatchError = false;
  const logMismatchError = () => {
    if (hasLoggedMismatchError) {
      return;
    }
    console.error("Hydration completed but contains mismatches.");
    hasLoggedMismatchError = true;
  };
  const isSVGContainer = (container) => container.namespaceURI.includes("svg") && container.tagName !== "foreignObject";
  const isMathMLContainer = (container) => container.namespaceURI.includes("MathML");
  const getContainerType = (container) => {
    if (container.nodeType !== 1) return void 0;
    if (isSVGContainer(container)) return "svg";
    if (isMathMLContainer(container)) return "mathml";
    return void 0;
  };
  const isComment = (node) => node.nodeType === 8;
  function createHydrationFunctions(rendererInternals) {
    const {
      mt: mountComponent,
      p: patch,
      o: {
        patchProp: patchProp2,
        createText,
        nextSibling,
        parentNode,
        remove: remove2,
        insert,
        createComment
      }
    } = rendererInternals;
    const hydrate2 = (vnode, container) => {
      if (!container.hasChildNodes()) {
        patch(null, vnode, container);
        flushPostFlushCbs();
        container._vnode = vnode;
        return;
      }
      hydrateNode(container.firstChild, vnode, null, null, null);
      flushPostFlushCbs();
      container._vnode = vnode;
    };
    const hydrateNode = (node, vnode, parentComponent, parentSuspense, slotScopeIds, optimized = false) => {
      optimized = optimized || !!vnode.dynamicChildren;
      const isFragmentStart = isComment(node) && node.data === "[";
      const onMismatch = () => handleMismatch(
        node,
        vnode,
        parentComponent,
        parentSuspense,
        slotScopeIds,
        isFragmentStart
      );
      const { type, ref: ref3, shapeFlag, patchFlag } = vnode;
      let domType = node.nodeType;
      vnode.el = node;
      if (patchFlag === -2) {
        optimized = false;
        vnode.dynamicChildren = null;
      }
      let nextNode = null;
      switch (type) {
        case Text:
          if (domType !== 3) {
            if (vnode.children === "") {
              insert(vnode.el = createText(""), parentNode(node), node);
              nextNode = node;
            } else {
              nextNode = onMismatch();
            }
          } else {
            if (node.data !== vnode.children) {
              logMismatchError();
              node.data = vnode.children;
            }
            nextNode = nextSibling(node);
          }
          break;
        case Comment:
          if (isTemplateNode(node)) {
            nextNode = nextSibling(node);
            replaceNode(
              vnode.el = node.content.firstChild,
              node,
              parentComponent
            );
          } else if (domType !== 8 || isFragmentStart) {
            nextNode = onMismatch();
          } else {
            nextNode = nextSibling(node);
          }
          break;
        case Static:
          if (isFragmentStart) {
            node = nextSibling(node);
            domType = node.nodeType;
          }
          if (domType === 1 || domType === 3) {
            nextNode = node;
            const needToAdoptContent = !vnode.children.length;
            for (let i = 0; i < vnode.staticCount; i++) {
              if (needToAdoptContent)
                vnode.children += nextNode.nodeType === 1 ? nextNode.outerHTML : nextNode.data;
              if (i === vnode.staticCount - 1) {
                vnode.anchor = nextNode;
              }
              nextNode = nextSibling(nextNode);
            }
            return isFragmentStart ? nextSibling(nextNode) : nextNode;
          } else {
            onMismatch();
          }
          break;
        case Fragment:
          if (!isFragmentStart) {
            nextNode = onMismatch();
          } else {
            nextNode = hydrateFragment(
              node,
              vnode,
              parentComponent,
              parentSuspense,
              slotScopeIds,
              optimized
            );
          }
          break;
        default:
          if (shapeFlag & 1) {
            if ((domType !== 1 || vnode.type.toLowerCase() !== node.tagName.toLowerCase()) && !isTemplateNode(node)) {
              nextNode = onMismatch();
            } else {
              nextNode = hydrateElement(
                node,
                vnode,
                parentComponent,
                parentSuspense,
                slotScopeIds,
                optimized
              );
            }
          } else if (shapeFlag & 6) {
            vnode.slotScopeIds = slotScopeIds;
            const container = parentNode(node);
            if (isFragmentStart) {
              nextNode = locateClosingAnchor(node);
            } else if (isComment(node) && node.data === "teleport start") {
              nextNode = locateClosingAnchor(node, node.data, "teleport end");
            } else {
              nextNode = nextSibling(node);
            }
            mountComponent(
              vnode,
              container,
              null,
              parentComponent,
              parentSuspense,
              getContainerType(container),
              optimized
            );
            if (isAsyncWrapper(vnode) && !vnode.type.__asyncResolved) {
              let subTree;
              if (isFragmentStart) {
                subTree = createVNode(Fragment);
                subTree.anchor = nextNode ? nextNode.previousSibling : container.lastChild;
              } else {
                subTree = node.nodeType === 3 ? createTextVNode("") : createVNode("div");
              }
              subTree.el = node;
              vnode.component.subTree = subTree;
            }
          } else if (shapeFlag & 64) {
            if (domType !== 8) {
              nextNode = onMismatch();
            } else {
              nextNode = vnode.type.hydrate(
                node,
                vnode,
                parentComponent,
                parentSuspense,
                slotScopeIds,
                optimized,
                rendererInternals,
                hydrateChildren
              );
            }
          } else if (shapeFlag & 128) {
            nextNode = vnode.type.hydrate(
              node,
              vnode,
              parentComponent,
              parentSuspense,
              getContainerType(parentNode(node)),
              slotScopeIds,
              optimized,
              rendererInternals,
              hydrateNode
            );
          } else ;
      }
      if (ref3 != null) {
        setRef(ref3, null, parentSuspense, vnode);
      }
      return nextNode;
    };
    const hydrateElement = (el, vnode, parentComponent, parentSuspense, slotScopeIds, optimized) => {
      optimized = optimized || !!vnode.dynamicChildren;
      const {
        type,
        dynamicProps,
        props,
        patchFlag,
        shapeFlag,
        dirs,
        transition
      } = vnode;
      const forcePatch = type === "input" || type === "option";
      const hasDynamicProps = !!dynamicProps;
      if (forcePatch || hasDynamicProps || patchFlag !== -1) {
        if (dirs) {
          invokeDirectiveHook(vnode, null, parentComponent, "created");
        }
        let needCallTransitionHooks = false;
        if (isTemplateNode(el)) {
          needCallTransitionHooks = needTransition(
            null,
            // no need check parentSuspense in hydration
            transition
          ) && parentComponent && parentComponent.vnode.props && parentComponent.vnode.props.appear;
          const content = el.content.firstChild;
          if (needCallTransitionHooks) {
            const cls = content.getAttribute("class");
            if (cls) content.$cls = cls;
            transition.beforeEnter(content);
          }
          replaceNode(content, el, parentComponent);
          vnode.el = el = content;
        }
        if (shapeFlag & 16 && // skip if element has innerHTML / textContent
        !(props && (props.innerHTML || props.textContent))) {
          let next = hydrateChildren(
            el.firstChild,
            vnode,
            el,
            parentComponent,
            parentSuspense,
            slotScopeIds,
            optimized
          );
          if (next && !isMismatchAllowed(
            el,
            1
            /* CHILDREN */
          )) {
            logMismatchError();
          }
          while (next) {
            const cur = next;
            next = next.nextSibling;
            remove2(cur);
          }
        } else if (shapeFlag & 8) {
          let clientText = vnode.children;
          if (clientText[0] === "\n" && (el.tagName === "PRE" || el.tagName === "TEXTAREA")) {
            clientText = clientText.slice(1);
          }
          const { textContent } = el;
          if (textContent !== clientText && // innerHTML normalize \r\n or \r into a single \n in the DOM
          textContent !== clientText.replace(/\r\n|\r/g, "\n")) {
            if (!isMismatchAllowed(
              el,
              0
              /* TEXT */
            )) {
              logMismatchError();
            }
            el.textContent = vnode.children;
          }
        }
        if (props) {
          if (forcePatch || hasDynamicProps || !optimized || patchFlag & (16 | 32)) {
            const isCustomElement = el.tagName.includes("-");
            for (const key in props) {
              if (forcePatch && (key.endsWith("value") || key === "indeterminate") || isOn(key) && !isReservedProp(key) || // force hydrate v-bind with .prop modifiers
              key[0] === "." || isCustomElement && !isReservedProp(key) || dynamicProps && dynamicProps.includes(key)) {
                patchProp2(el, key, null, props[key], void 0, parentComponent);
              }
            }
          } else if (props.onClick) {
            patchProp2(
              el,
              "onClick",
              null,
              props.onClick,
              void 0,
              parentComponent
            );
          } else if (patchFlag & 4 && /* @__PURE__ */ isReactive(props.style)) {
            for (const key in props.style) props.style[key];
          }
        }
        let vnodeHooks;
        if (vnodeHooks = props && props.onVnodeBeforeMount) {
          invokeVNodeHook(vnodeHooks, parentComponent, vnode);
        }
        if (dirs) {
          invokeDirectiveHook(vnode, null, parentComponent, "beforeMount");
        }
        if ((vnodeHooks = props && props.onVnodeMounted) || dirs || needCallTransitionHooks) {
          queueEffectWithSuspense(() => {
            vnodeHooks && invokeVNodeHook(vnodeHooks, parentComponent, vnode);
            needCallTransitionHooks && transition.enter(el);
            dirs && invokeDirectiveHook(vnode, null, parentComponent, "mounted");
          }, parentSuspense);
        }
      }
      return el.nextSibling;
    };
    const hydrateChildren = (node, parentVNode, container, parentComponent, parentSuspense, slotScopeIds, optimized) => {
      optimized = optimized || !!parentVNode.dynamicChildren;
      const children = parentVNode.children;
      const l = children.length;
      let hasCheckedMismatch = false;
      for (let i = 0; i < l; i++) {
        const vnode = optimized ? children[i] : children[i] = normalizeVNode(children[i]);
        const isText = vnode.type === Text;
        if (node) {
          if (isText && !optimized) {
            if (i + 1 < l && normalizeVNode(children[i + 1]).type === Text) {
              insert(
                createText(
                  node.data.slice(vnode.children.length)
                ),
                container,
                nextSibling(node)
              );
              node.data = vnode.children;
            }
          }
          node = hydrateNode(
            node,
            vnode,
            parentComponent,
            parentSuspense,
            slotScopeIds,
            optimized
          );
        } else if (isText && !vnode.children) {
          insert(vnode.el = createText(""), container);
        } else {
          if (!hasCheckedMismatch) {
            hasCheckedMismatch = true;
            if (!isMismatchAllowed(
              container,
              1
              /* CHILDREN */
            )) {
              logMismatchError();
            }
          }
          patch(
            null,
            vnode,
            container,
            null,
            parentComponent,
            parentSuspense,
            getContainerType(container),
            slotScopeIds
          );
        }
      }
      return node;
    };
    const hydrateFragment = (node, vnode, parentComponent, parentSuspense, slotScopeIds, optimized) => {
      const { slotScopeIds: fragmentSlotScopeIds } = vnode;
      if (fragmentSlotScopeIds) {
        slotScopeIds = slotScopeIds ? slotScopeIds.concat(fragmentSlotScopeIds) : fragmentSlotScopeIds;
      }
      const container = parentNode(node);
      const next = hydrateChildren(
        nextSibling(node),
        vnode,
        container,
        parentComponent,
        parentSuspense,
        slotScopeIds,
        optimized
      );
      if (next && isComment(next) && next.data === "]") {
        return nextSibling(vnode.anchor = next);
      } else {
        logMismatchError();
        insert(vnode.anchor = createComment(`]`), container, next);
        return next;
      }
    };
    const handleMismatch = (node, vnode, parentComponent, parentSuspense, slotScopeIds, isFragment) => {
      if (!isNodeMismatchAllowed(node, vnode)) {
        logMismatchError();
      }
      vnode.el = null;
      if (isFragment) {
        const end = locateClosingAnchor(node);
        while (true) {
          const next2 = nextSibling(node);
          if (next2 && next2 !== end) {
            remove2(next2);
          } else {
            break;
          }
        }
      }
      const next = nextSibling(node);
      const container = parentNode(node);
      remove2(node);
      patch(
        null,
        vnode,
        container,
        next,
        parentComponent,
        parentSuspense,
        getContainerType(container),
        slotScopeIds
      );
      if (parentComponent) {
        parentComponent.vnode.el = vnode.el;
        updateHOCHostEl(parentComponent, vnode.el);
      }
      return next;
    };
    const locateClosingAnchor = (node, open = "[", close = "]") => {
      let match = 0;
      while (node) {
        node = nextSibling(node);
        if (node && isComment(node)) {
          if (node.data === open) match++;
          if (node.data === close) {
            if (match === 0) {
              return nextSibling(node);
            } else {
              match--;
            }
          }
        }
      }
      return node;
    };
    const replaceNode = (newNode, oldNode, parentComponent) => {
      const parentNode2 = oldNode.parentNode;
      if (parentNode2) {
        parentNode2.replaceChild(newNode, oldNode);
      }
      let parent = parentComponent;
      while (parent) {
        if (parent.vnode.el === oldNode) {
          parent.vnode.el = parent.subTree.el = newNode;
        }
        parent = parent.parent;
      }
    };
    const isTemplateNode = (node) => {
      return node.nodeType === 1 && node.tagName === "TEMPLATE";
    };
    return [hydrate2, hydrateNode];
  }
  const allowMismatchAttr = "data-allow-mismatch";
  const MismatchTypeString = {
    [
      0
      /* TEXT */
    ]: "text",
    [
      1
      /* CHILDREN */
    ]: "children",
    [
      2
      /* CLASS */
    ]: "class",
    [
      3
      /* STYLE */
    ]: "style",
    [
      4
      /* ATTRIBUTE */
    ]: "attribute"
  };
  function isMismatchAllowed(el, allowedType) {
    if (allowedType === 0 || allowedType === 1) {
      while (el && !el.hasAttribute(allowMismatchAttr)) {
        el = el.parentElement;
      }
    }
    return isMismatchAllowedByAttr(
      el && el.getAttribute(allowMismatchAttr),
      allowedType
    );
  }
  function isMismatchAllowedByAttr(allowedAttr, allowedType) {
    if (allowedAttr == null) {
      return false;
    } else if (allowedAttr === "") {
      return true;
    } else {
      const list = allowedAttr.split(",");
      if (allowedType === 0 && list.includes("children")) {
        return true;
      }
      return list.includes(MismatchTypeString[allowedType]);
    }
  }
  function isNodeMismatchAllowed(node, vnode) {
    return isMismatchAllowed(
      node.parentElement,
      1
      /* CHILDREN */
    ) || isMismatchAllowedByNode(node) || isMismatchAllowedByVNode(vnode);
  }
  function isMismatchAllowedByNode(node) {
    return node.nodeType === 1 && isMismatchAllowedByAttr(
      node.getAttribute(allowMismatchAttr),
      1
      /* CHILDREN */
    );
  }
  function isMismatchAllowedByVNode({ props }) {
    const allowedAttr = props && props[allowMismatchAttr];
    return typeof allowedAttr === "string" && isMismatchAllowedByAttr(
      allowedAttr,
      1
      /* CHILDREN */
    );
  }
  const requestIdleCallback = getGlobalThis().requestIdleCallback || ((cb) => setTimeout(cb, 1));
  const cancelIdleCallback = getGlobalThis().cancelIdleCallback || ((id) => clearTimeout(id));
  const hydrateOnIdle = (timeout = 1e4) => (hydrate2) => {
    const id = requestIdleCallback(hydrate2, { timeout });
    return () => cancelIdleCallback(id);
  };
  function elementIsVisibleInViewport(el) {
    const { top, left, bottom, right } = el.getBoundingClientRect();
    const { innerHeight, innerWidth } = window;
    return (top > 0 && top < innerHeight || bottom > 0 && bottom < innerHeight) && (left > 0 && left < innerWidth || right > 0 && right < innerWidth);
  }
  const hydrateOnVisible = (opts) => (hydrate2, forEach) => {
    const ob = new IntersectionObserver((entries) => {
      for (const e of entries) {
        if (!e.isIntersecting) continue;
        ob.disconnect();
        hydrate2();
        break;
      }
    }, opts);
    forEach((el) => {
      if (!(el instanceof Element)) return;
      if (elementIsVisibleInViewport(el)) {
        hydrate2();
        ob.disconnect();
        return false;
      }
      ob.observe(el);
    });
    return () => ob.disconnect();
  };
  const hydrateOnMediaQuery = (query) => (hydrate2) => {
    if (query) {
      const mql = matchMedia(query);
      if (mql.matches) {
        hydrate2();
      } else {
        mql.addEventListener("change", hydrate2, { once: true });
        return () => mql.removeEventListener("change", hydrate2);
      }
    }
  };
  const hydrateOnInteraction = (interactions = []) => (hydrate2, forEach) => {
    if (isString(interactions)) interactions = [interactions];
    let hasHydrated = false;
    const doHydrate = (e) => {
      if (!hasHydrated) {
        hasHydrated = true;
        teardown();
        hydrate2();
        e.target.dispatchEvent(new e.constructor(e.type, e));
      }
    };
    const teardown = () => {
      forEach((el) => {
        for (const i of interactions) {
          el.removeEventListener(i, doHydrate);
        }
      });
    };
    forEach((el) => {
      for (const i of interactions) {
        el.addEventListener(i, doHydrate, { once: true });
      }
    });
    return teardown;
  };
  function forEachElement(node, cb) {
    if (isComment(node) && node.data === "[") {
      let depth = 1;
      let next = node.nextSibling;
      while (next) {
        if (next.nodeType === 1) {
          const result = cb(next);
          if (result === false) {
            break;
          }
        } else if (isComment(next)) {
          if (next.data === "]") {
            if (--depth === 0) break;
          } else if (next.data === "[") {
            depth++;
          }
        }
        next = next.nextSibling;
      }
    } else {
      cb(node);
    }
  }
  const isAsyncWrapper = (i) => !!i.type.__asyncLoader;
  // @__NO_SIDE_EFFECTS__
  function defineAsyncComponent(source) {
    if (isFunction(source)) {
      source = { loader: source };
    }
    const {
      loader,
      loadingComponent,
      errorComponent,
      delay = 200,
      hydrate: hydrateStrategy,
      timeout,
      // undefined = never times out
      suspensible = true,
      onError: userOnError
    } = source;
    let pendingRequest = null;
    let resolvedComp;
    let retries = 0;
    const retry = () => {
      retries++;
      pendingRequest = null;
      return load();
    };
    const load = () => {
      let thisRequest;
      return pendingRequest || (thisRequest = pendingRequest = loader().catch((err) => {
        err = err instanceof Error ? err : new Error(String(err));
        if (userOnError) {
          return new Promise((resolve2, reject) => {
            const userRetry = () => resolve2(retry());
            const userFail = () => reject(err);
            userOnError(err, userRetry, userFail, retries + 1);
          });
        } else {
          throw err;
        }
      }).then((comp) => {
        if (thisRequest !== pendingRequest && pendingRequest) {
          return pendingRequest;
        }
        if (comp && (comp.__esModule || comp[Symbol.toStringTag] === "Module")) {
          comp = comp.default;
        }
        resolvedComp = comp;
        return comp;
      }));
    };
    return /* @__PURE__ */ defineComponent({
      name: "AsyncComponentWrapper",
      __asyncLoader: load,
      __asyncHydrate(el, instance, hydrate2) {
        let patched = false;
        (instance.bu || (instance.bu = [])).push(() => patched = true);
        const performHydrate = () => {
          if (patched) {
            return;
          }
          hydrate2();
        };
        const doHydrate = hydrateStrategy ? () => {
          const teardown = hydrateStrategy(
            performHydrate,
            (cb) => forEachElement(el, cb)
          );
          if (teardown) {
            (instance.bum || (instance.bum = [])).push(teardown);
          }
        } : performHydrate;
        if (resolvedComp) {
          doHydrate();
        } else {
          load().then(() => !instance.isUnmounted && doHydrate());
        }
      },
      get __asyncResolved() {
        return resolvedComp;
      },
      setup() {
        const instance = currentInstance;
        markAsyncBoundary(instance);
        if (resolvedComp) {
          return () => createInnerComp(resolvedComp, instance);
        }
        const onError = (err) => {
          pendingRequest = null;
          handleError(
            err,
            instance,
            13,
            !errorComponent
          );
        };
        if (suspensible && instance.suspense || isInSSRComponentSetup) {
          return load().then((comp) => {
            return () => createInnerComp(comp, instance);
          }).catch((err) => {
            onError(err);
            return () => errorComponent ? createVNode(errorComponent, {
              error: err
            }) : null;
          });
        }
        const loaded = /* @__PURE__ */ ref(false);
        const error = /* @__PURE__ */ ref();
        const delayed = /* @__PURE__ */ ref(!!delay);
        let timeoutTimer;
        let delayTimer;
        onUnmounted(() => {
          if (timeoutTimer != null) clearTimeout(timeoutTimer);
          if (delayTimer != null) clearTimeout(delayTimer);
        });
        if (delay) {
          delayTimer = setTimeout(() => {
            if (instance.isUnmounted) return;
            delayed.value = false;
          }, delay);
        }
        if (timeout != null) {
          timeoutTimer = setTimeout(() => {
            if (instance.isUnmounted) return;
            if (!loaded.value && !error.value) {
              const err = new Error(
                `Async component timed out after ${timeout}ms.`
              );
              onError(err);
              error.value = err;
            }
          }, timeout);
        }
        load().then(() => {
          if (instance.isUnmounted) return;
          loaded.value = true;
          if (instance.parent && isKeepAlive(instance.parent.vnode)) {
            instance.parent.update();
          }
        }).catch((err) => {
          if (instance.isUnmounted) {
            pendingRequest = null;
            return;
          }
          onError(err);
          error.value = err;
        });
        return () => {
          if (loaded.value && resolvedComp) {
            return createInnerComp(resolvedComp, instance);
          } else if (error.value && errorComponent) {
            return createVNode(errorComponent, {
              error: error.value
            });
          } else if (loadingComponent && !delayed.value) {
            return createInnerComp(
              loadingComponent,
              instance
            );
          }
        };
      }
    });
  }
  function createInnerComp(comp, parent) {
    const { ref: ref22, props, children, ce } = parent.vnode;
    const vnode = createVNode(comp, props, children);
    vnode.ref = ref22;
    vnode.ce = ce;
    delete parent.vnode.ce;
    return vnode;
  }
  const isKeepAlive = (vnode) => vnode.type.__isKeepAlive;
  const KeepAliveImpl = {
    name: `KeepAlive`,
    // Marker for special handling inside the renderer. We are not using a ===
    // check directly on KeepAlive in the renderer, because importing it directly
    // would prevent it from being tree-shaken.
    __isKeepAlive: true,
    props: {
      include: [String, RegExp, Array],
      exclude: [String, RegExp, Array],
      max: [String, Number]
    },
    setup(props, { slots }) {
      const instance = getCurrentInstance();
      const sharedContext = instance.ctx;
      if (!sharedContext.renderer) {
        return () => {
          const children = slots.default && slots.default();
          return children && children.length === 1 ? children[0] : children;
        };
      }
      const cache = /* @__PURE__ */ new Map();
      const keys = /* @__PURE__ */ new Set();
      let current = null;
      const parentSuspense = instance.suspense;
      const {
        renderer: {
          p: patch,
          m: move,
          um: _unmount,
          o: { createElement }
        }
      } = sharedContext;
      const storageContainer = createElement("div");
      sharedContext.activate = (vnode, container, anchor, namespace, optimized) => {
        const instance2 = vnode.component;
        move(vnode, container, anchor, 0, parentSuspense);
        patch(
          instance2.vnode,
          vnode,
          container,
          anchor,
          instance2,
          parentSuspense,
          namespace,
          vnode.slotScopeIds,
          optimized
        );
        queuePostRenderEffect(() => {
          instance2.isDeactivated = false;
          if (instance2.a) {
            invokeArrayFns(instance2.a);
          }
          const vnodeHook = vnode.props && vnode.props.onVnodeMounted;
          if (vnodeHook) {
            invokeVNodeHook(vnodeHook, instance2.parent, vnode);
          }
        }, parentSuspense);
      };
      sharedContext.deactivate = (vnode) => {
        const instance2 = vnode.component;
        invalidateMount(instance2.m);
        invalidateMount(instance2.a);
        move(vnode, storageContainer, null, 1, parentSuspense);
        queuePostRenderEffect(() => {
          if (instance2.da) {
            invokeArrayFns(instance2.da);
          }
          const vnodeHook = vnode.props && vnode.props.onVnodeUnmounted;
          if (vnodeHook) {
            invokeVNodeHook(vnodeHook, instance2.parent, vnode);
          }
          instance2.isDeactivated = true;
        }, parentSuspense);
      };
      function unmount(vnode) {
        resetShapeFlag(vnode);
        _unmount(vnode, instance, parentSuspense, true);
      }
      function pruneCache(filter) {
        cache.forEach((vnode, key) => {
          const name = getComponentName(
            isAsyncWrapper(vnode) ? vnode.type.__asyncResolved || {} : vnode.type
          );
          if (name && !filter(name)) {
            pruneCacheEntry(key);
          }
        });
      }
      function pruneCacheEntry(key) {
        const cached = cache.get(key);
        if (cached && (!current || !isSameVNodeType(cached, current))) {
          unmount(cached);
        } else if (current) {
          resetShapeFlag(current);
        }
        cache.delete(key);
        keys.delete(key);
      }
      watch(
        () => [props.include, props.exclude],
        ([include, exclude]) => {
          include && pruneCache((name) => matches(include, name));
          exclude && pruneCache((name) => !matches(exclude, name));
        },
        // prune post-render after `current` has been updated
        { flush: "post", deep: true }
      );
      let pendingCacheKey = null;
      const cacheSubtree = () => {
        if (pendingCacheKey != null) {
          if (isSuspense(instance.subTree.type)) {
            queuePostRenderEffect(() => {
              cache.set(pendingCacheKey, getInnerChild(instance.subTree));
            }, instance.subTree.suspense);
          } else {
            cache.set(pendingCacheKey, getInnerChild(instance.subTree));
          }
        }
      };
      onMounted(cacheSubtree);
      onUpdated(cacheSubtree);
      onBeforeUnmount(() => {
        cache.forEach((cached) => {
          const { subTree, suspense } = instance;
          const vnode = getInnerChild(subTree);
          if (cached.type === vnode.type && cached.key === vnode.key) {
            resetShapeFlag(vnode);
            const da = vnode.component.da;
            da && queuePostRenderEffect(da, suspense);
            return;
          }
          unmount(cached);
        });
      });
      return () => {
        pendingCacheKey = null;
        if (!slots.default) {
          return current = null;
        }
        const children = slots.default();
        const rawVNode = children[0];
        if (children.length > 1) {
          current = null;
          return children;
        } else if (!isVNode(rawVNode) || !(rawVNode.shapeFlag & 4) && !(rawVNode.shapeFlag & 128)) {
          current = null;
          return rawVNode;
        }
        let vnode = getInnerChild(rawVNode);
        if (vnode.type === Comment) {
          current = null;
          return vnode;
        }
        const comp = vnode.type;
        const name = getComponentName(
          isAsyncWrapper(vnode) ? vnode.type.__asyncResolved || {} : comp
        );
        const { include, exclude, max } = props;
        if (include && (!name || !matches(include, name)) || exclude && name && matches(exclude, name)) {
          vnode.shapeFlag &= -257;
          current = vnode;
          return rawVNode;
        }
        const key = vnode.key == null ? comp : vnode.key;
        const cachedVNode = cache.get(key);
        if (vnode.el) {
          vnode = cloneVNode(vnode);
          if (rawVNode.shapeFlag & 128) {
            rawVNode.ssContent = vnode;
          }
        }
        pendingCacheKey = key;
        if (cachedVNode) {
          vnode.el = cachedVNode.el;
          vnode.component = cachedVNode.component;
          if (vnode.transition) {
            setTransitionHooks(vnode, vnode.transition);
          }
          vnode.shapeFlag |= 512;
          keys.delete(key);
          keys.add(key);
        } else {
          keys.add(key);
          if (max && keys.size > parseInt(max, 10)) {
            pruneCacheEntry(keys.values().next().value);
          }
        }
        vnode.shapeFlag |= 256;
        current = vnode;
        return isSuspense(rawVNode.type) ? rawVNode : vnode;
      };
    }
  };
  const KeepAlive = KeepAliveImpl;
  function matches(pattern, name) {
    if (isArray(pattern)) {
      return pattern.some((p2) => matches(p2, name));
    } else if (isString(pattern)) {
      return pattern.split(",").includes(name);
    } else if (isRegExp(pattern)) {
      pattern.lastIndex = 0;
      return pattern.test(name);
    }
    return false;
  }
  function onActivated(hook, target) {
    registerKeepAliveHook(hook, "a", target);
  }
  function onDeactivated(hook, target) {
    registerKeepAliveHook(hook, "da", target);
  }
  function registerKeepAliveHook(hook, type, target = currentInstance) {
    const wrappedHook = hook.__wdc || (hook.__wdc = () => {
      let current = target;
      while (current) {
        if (current.isDeactivated) {
          return;
        }
        current = current.parent;
      }
      return hook();
    });
    injectHook(type, wrappedHook, target);
    if (target) {
      let current = target.parent;
      while (current && current.parent) {
        if (isKeepAlive(current.parent.vnode)) {
          injectToKeepAliveRoot(wrappedHook, type, target, current);
        }
        current = current.parent;
      }
    }
  }
  function injectToKeepAliveRoot(hook, type, target, keepAliveRoot) {
    const injected = injectHook(
      type,
      hook,
      keepAliveRoot,
      true
      /* prepend */
    );
    onUnmounted(() => {
      remove(keepAliveRoot[type], injected);
    }, target);
  }
  function resetShapeFlag(vnode) {
    vnode.shapeFlag &= -257;
    vnode.shapeFlag &= -513;
  }
  function getInnerChild(vnode) {
    return vnode.shapeFlag & 128 ? vnode.ssContent : vnode;
  }
  function injectHook(type, hook, target = currentInstance, prepend = false) {
    if (target) {
      const hooks = target[type] || (target[type] = []);
      const wrappedHook = hook.__weh || (hook.__weh = (...args) => {
        pauseTracking();
        const reset = setCurrentInstance(target);
        const res = callWithAsyncErrorHandling(hook, target, type, args);
        reset();
        resetTracking();
        return res;
      });
      if (prepend) {
        hooks.unshift(wrappedHook);
      } else {
        hooks.push(wrappedHook);
      }
      return wrappedHook;
    }
  }
  const createHook = (lifecycle) => (hook, target = currentInstance) => {
    if (!isInSSRComponentSetup || lifecycle === "sp") {
      injectHook(lifecycle, (...args) => hook(...args), target);
    }
  };
  const onBeforeMount = createHook("bm");
  const onMounted = createHook("m");
  const onBeforeUpdate = createHook(
    "bu"
  );
  const onUpdated = createHook("u");
  const onBeforeUnmount = createHook(
    "bum"
  );
  const onUnmounted = createHook("um");
  const onServerPrefetch = createHook(
    "sp"
  );
  const onRenderTriggered = createHook("rtg");
  const onRenderTracked = createHook("rtc");
  function onErrorCaptured(hook, target = currentInstance) {
    injectHook("ec", hook, target);
  }
  const COMPONENTS = "components";
  const DIRECTIVES = "directives";
  function resolveComponent(name, maybeSelfReference) {
    return resolveAsset(COMPONENTS, name, true, maybeSelfReference) || name;
  }
  const NULL_DYNAMIC_COMPONENT = /* @__PURE__ */ Symbol.for("v-ndc");
  function resolveDynamicComponent(component) {
    if (isString(component)) {
      return resolveAsset(COMPONENTS, component, false) || component;
    } else {
      return component || NULL_DYNAMIC_COMPONENT;
    }
  }
  function resolveDirective(name) {
    return resolveAsset(DIRECTIVES, name);
  }
  function resolveAsset(type, name, warnMissing = true, maybeSelfReference = false) {
    const instance = currentRenderingInstance || currentInstance;
    if (instance) {
      const Component = instance.type;
      if (type === COMPONENTS) {
        const selfName = getComponentName(
          Component,
          false
        );
        if (selfName && (selfName === name || selfName === camelize(name) || selfName === capitalize(camelize(name)))) {
          return Component;
        }
      }
      const res = (
        // local registration
        // check instance[type] first which is resolved for options API
        resolve(instance[type] || Component[type], name) || // global registration
        resolve(instance.appContext[type], name)
      );
      if (!res && maybeSelfReference) {
        return Component;
      }
      return res;
    }
  }
  function resolve(registry, name) {
    return registry && (registry[name] || registry[camelize(name)] || registry[capitalize(camelize(name))]);
  }
  function renderList(source, renderItem, cache, index) {
    let ret;
    const cached = cache && cache[index];
    const sourceIsArray = isArray(source);
    if (sourceIsArray || isString(source)) {
      const sourceIsReactiveArray = sourceIsArray && /* @__PURE__ */ isReactive(source);
      let needsWrap = false;
      let isReadonlySource = false;
      if (sourceIsReactiveArray) {
        needsWrap = !/* @__PURE__ */ isShallow(source);
        isReadonlySource = /* @__PURE__ */ isReadonly(source);
        source = shallowReadArray(source);
      }
      ret = new Array(source.length);
      for (let i = 0, l = source.length; i < l; i++) {
        ret[i] = renderItem(
          needsWrap ? isReadonlySource ? toReadonly(toReactive(source[i])) : toReactive(source[i]) : source[i],
          i,
          void 0,
          cached && cached[i]
        );
      }
    } else if (typeof source === "number") {
      {
        ret = new Array(source);
        for (let i = 0; i < source; i++) {
          ret[i] = renderItem(i + 1, i, void 0, cached && cached[i]);
        }
      }
    } else if (isObject(source)) {
      if (source[Symbol.iterator]) {
        ret = Array.from(
          source,
          (item, i) => renderItem(item, i, void 0, cached && cached[i])
        );
      } else {
        const keys = Object.keys(source);
        ret = new Array(keys.length);
        for (let i = 0, l = keys.length; i < l; i++) {
          const key = keys[i];
          ret[i] = renderItem(source[key], key, i, cached && cached[i]);
        }
      }
    } else {
      ret = [];
    }
    if (cache) {
      cache[index] = ret;
    }
    return ret;
  }
  function createSlots(slots, dynamicSlots) {
    for (let i = 0; i < dynamicSlots.length; i++) {
      const slot = dynamicSlots[i];
      if (isArray(slot)) {
        for (let j = 0; j < slot.length; j++) {
          slots[slot[j].name] = slot[j].fn;
        }
      } else if (slot) {
        slots[slot.name] = slot.key ? (...args) => {
          const res = slot.fn(...args);
          if (res) res.key = slot.key;
          return res;
        } : slot.fn;
      }
    }
    return slots;
  }
  function renderSlot(slots, name, props = {}, fallback, noSlotted) {
    if (currentRenderingInstance.ce || currentRenderingInstance.parent && isAsyncWrapper(currentRenderingInstance.parent) && currentRenderingInstance.parent.ce) {
      const hasProps = Object.keys(props).length > 0;
      if (name !== "default") props.name = name;
      return openBlock(), createBlock(
        Fragment,
        null,
        [createVNode("slot", props, fallback && fallback())],
        hasProps ? -2 : 64
      );
    }
    let slot = slots[name];
    if (slot && slot._c) {
      slot._d = false;
    }
    openBlock();
    const validSlotContent = slot && ensureValidVNode(slot(props));
    const slotKey = props.key || // slot content array of a dynamic conditional slot may have a branch
    // key attached in the `createSlots` helper, respect that
    validSlotContent && validSlotContent.key;
    const rendered = createBlock(
      Fragment,
      {
        key: (slotKey && !isSymbol(slotKey) ? slotKey : `_${name}`) + // #7256 force differentiate fallback content from actual content
        (!validSlotContent && fallback ? "_fb" : "")
      },
      validSlotContent || (fallback ? fallback() : []),
      validSlotContent && slots._ === 1 ? 64 : -2
    );
    if (!noSlotted && rendered.scopeId) {
      rendered.slotScopeIds = [rendered.scopeId + "-s"];
    }
    if (slot && slot._c) {
      slot._d = true;
    }
    return rendered;
  }
  function ensureValidVNode(vnodes) {
    return vnodes.some((child) => {
      if (!isVNode(child)) return true;
      if (child.type === Comment) return false;
      if (child.type === Fragment && !ensureValidVNode(child.children))
        return false;
      return true;
    }) ? vnodes : null;
  }
  function toHandlers(obj, preserveCaseIfNecessary) {
    const ret = {};
    for (const key in obj) {
      ret[preserveCaseIfNecessary && /[A-Z]/.test(key) ? `on:${key}` : toHandlerKey(key)] = obj[key];
    }
    return ret;
  }
  const getPublicInstance = (i) => {
    if (!i) return null;
    if (isStatefulComponent(i)) return getComponentPublicInstance(i);
    return getPublicInstance(i.parent);
  };
  const publicPropertiesMap = (
    // Move PURE marker to new line to workaround compiler discarding it
    // due to type annotation
    /* @__PURE__ */ extend(/* @__PURE__ */ Object.create(null), {
      $: (i) => i,
      $el: (i) => i.vnode.el,
      $data: (i) => i.data,
      $props: (i) => i.props,
      $attrs: (i) => i.attrs,
      $slots: (i) => i.slots,
      $refs: (i) => i.refs,
      $parent: (i) => getPublicInstance(i.parent),
      $root: (i) => getPublicInstance(i.root),
      $host: (i) => i.ce,
      $emit: (i) => i.emit,
      $options: (i) => resolveMergedOptions(i),
      $forceUpdate: (i) => i.f || (i.f = () => {
        queueJob(i.update);
      }),
      $nextTick: (i) => i.n || (i.n = nextTick.bind(i.proxy)),
      $watch: (i) => instanceWatch.bind(i)
    })
  );
  const hasSetupBinding = (state2, key) => state2 !== EMPTY_OBJ && !state2.__isScriptSetup && hasOwn(state2, key);
  const PublicInstanceProxyHandlers = {
    get({ _: instance }, key) {
      if (key === "__v_skip") {
        return true;
      }
      const { ctx, setupState, data, props, accessCache, type, appContext } = instance;
      if (key[0] !== "$") {
        const n = accessCache[key];
        if (n !== void 0) {
          switch (n) {
            case 1:
              return setupState[key];
            case 2:
              return data[key];
            case 4:
              return ctx[key];
            case 3:
              return props[key];
          }
        } else if (hasSetupBinding(setupState, key)) {
          accessCache[key] = 1;
          return setupState[key];
        } else if (data !== EMPTY_OBJ && hasOwn(data, key)) {
          accessCache[key] = 2;
          return data[key];
        } else if (hasOwn(props, key)) {
          accessCache[key] = 3;
          return props[key];
        } else if (ctx !== EMPTY_OBJ && hasOwn(ctx, key)) {
          accessCache[key] = 4;
          return ctx[key];
        } else if (shouldCacheAccess) {
          accessCache[key] = 0;
        }
      }
      const publicGetter = publicPropertiesMap[key];
      let cssModule, globalProperties;
      if (publicGetter) {
        if (key === "$attrs") {
          track(instance.attrs, "get", "");
        }
        return publicGetter(instance);
      } else if (
        // css module (injected by vue-loader)
        (cssModule = type.__cssModules) && (cssModule = cssModule[key])
      ) {
        return cssModule;
      } else if (ctx !== EMPTY_OBJ && hasOwn(ctx, key)) {
        accessCache[key] = 4;
        return ctx[key];
      } else if (
        // global properties
        globalProperties = appContext.config.globalProperties, hasOwn(globalProperties, key)
      ) {
        {
          return globalProperties[key];
        }
      } else ;
    },
    set({ _: instance }, key, value) {
      const { data, setupState, ctx } = instance;
      if (hasSetupBinding(setupState, key)) {
        setupState[key] = value;
        return true;
      } else if (data !== EMPTY_OBJ && hasOwn(data, key)) {
        data[key] = value;
        return true;
      } else if (hasOwn(instance.props, key)) {
        return false;
      }
      if (key[0] === "$" && key.slice(1) in instance) {
        return false;
      } else {
        {
          ctx[key] = value;
        }
      }
      return true;
    },
    has({
      _: { data, setupState, accessCache, ctx, appContext, props, type }
    }, key) {
      let cssModules;
      return !!(accessCache[key] || data !== EMPTY_OBJ && key[0] !== "$" && hasOwn(data, key) || hasSetupBinding(setupState, key) || hasOwn(props, key) || hasOwn(ctx, key) || hasOwn(publicPropertiesMap, key) || hasOwn(appContext.config.globalProperties, key) || (cssModules = type.__cssModules) && cssModules[key]);
    },
    defineProperty(target, key, descriptor) {
      if (descriptor.get != null) {
        target._.accessCache[key] = 0;
      } else if (hasOwn(descriptor, "value")) {
        this.set(target, key, descriptor.value, null);
      }
      return Reflect.defineProperty(target, key, descriptor);
    }
  };
  const RuntimeCompiledPublicInstanceProxyHandlers = /* @__PURE__ */ extend({}, PublicInstanceProxyHandlers, {
    get(target, key) {
      if (key === Symbol.unscopables) {
        return;
      }
      return PublicInstanceProxyHandlers.get(target, key, target);
    },
    has(_, key) {
      const has = key[0] !== "_" && !isGloballyAllowed(key);
      return has;
    }
  });
  function defineProps() {
    return null;
  }
  function defineEmits() {
    return null;
  }
  function defineExpose(exposed) {
  }
  function defineOptions(options) {
  }
  function defineSlots() {
    return null;
  }
  function defineModel() {
  }
  function withDefaults(props, defaults) {
    return null;
  }
  function useSlots() {
    return getContext().slots;
  }
  function useAttrs() {
    return getContext().attrs;
  }
  function getContext(calledFunctionName) {
    const i = getCurrentInstance();
    return i.setupContext || (i.setupContext = createSetupContext(i));
  }
  function normalizePropsOrEmits(props) {
    return isArray(props) ? props.reduce(
      (normalized, p2) => (normalized[p2] = null, normalized),
      {}
    ) : props;
  }
  function mergeDefaults(raw, defaults) {
    const props = normalizePropsOrEmits(raw);
    for (const key in defaults) {
      if (key.startsWith("__skip")) continue;
      let opt = props[key];
      if (opt) {
        if (isArray(opt) || isFunction(opt)) {
          opt = props[key] = { type: opt, default: defaults[key] };
        } else {
          opt.default = defaults[key];
        }
      } else if (opt === null) {
        opt = props[key] = { default: defaults[key] };
      } else ;
      if (opt && defaults[`__skip_${key}`]) {
        opt.skipFactory = true;
      }
    }
    return props;
  }
  function mergeModels(a, b) {
    if (!a || !b) return a || b;
    if (isArray(a) && isArray(b)) return a.concat(b);
    return extend({}, normalizePropsOrEmits(a), normalizePropsOrEmits(b));
  }
  function createPropsRestProxy(props, excludedKeys) {
    const ret = {};
    for (const key in props) {
      if (!excludedKeys.includes(key)) {
        Object.defineProperty(ret, key, {
          enumerable: true,
          get: () => props[key]
        });
      }
    }
    return ret;
  }
  function withAsyncContext(getAwaitable) {
    const ctx = getCurrentInstance();
    const inSSRSetup = isInSSRComponentSetup;
    let awaitable = getAwaitable();
    unsetCurrentInstance();
    if (inSSRSetup) {
      setInSSRSetupState(false);
    }
    const restore = () => {
      setCurrentInstance(ctx);
      if (inSSRSetup) {
        setInSSRSetupState(true);
      }
    };
    const cleanup = () => {
      if (getCurrentInstance() !== ctx) ctx.scope.off();
      unsetCurrentInstance();
      if (inSSRSetup) {
        setInSSRSetupState(false);
      }
    };
    if (isPromise(awaitable)) {
      awaitable = awaitable.catch((e) => {
        restore();
        Promise.resolve().then(() => Promise.resolve().then(cleanup));
        throw e;
      });
    }
    return [
      awaitable,
      () => {
        restore();
        Promise.resolve().then(cleanup);
      }
    ];
  }
  let shouldCacheAccess = true;
  function applyOptions(instance) {
    const options = resolveMergedOptions(instance);
    const publicThis = instance.proxy;
    const ctx = instance.ctx;
    shouldCacheAccess = false;
    if (options.beforeCreate) {
      callHook$1(options.beforeCreate, instance, "bc");
    }
    const {
      // state
      data: dataOptions,
      computed: computedOptions,
      methods,
      watch: watchOptions,
      provide: provideOptions,
      inject: injectOptions,
      // lifecycle
      created,
      beforeMount,
      mounted,
      beforeUpdate,
      updated,
      activated,
      deactivated,
      beforeDestroy,
      beforeUnmount,
      destroyed,
      unmounted,
      render: render2,
      renderTracked,
      renderTriggered,
      errorCaptured,
      serverPrefetch,
      // public API
      expose,
      inheritAttrs,
      // assets
      components,
      directives,
      filters
    } = options;
    const checkDuplicateProperties = null;
    if (injectOptions) {
      resolveInjections(injectOptions, ctx, checkDuplicateProperties);
    }
    if (methods) {
      for (const key in methods) {
        const methodHandler = methods[key];
        if (isFunction(methodHandler)) {
          {
            ctx[key] = methodHandler.bind(publicThis);
          }
        }
      }
    }
    if (dataOptions) {
      const data = dataOptions.call(publicThis, publicThis);
      if (!isObject(data)) ;
      else {
        instance.data = /* @__PURE__ */ reactive(data);
      }
    }
    shouldCacheAccess = true;
    if (computedOptions) {
      for (const key in computedOptions) {
        const opt = computedOptions[key];
        const get = isFunction(opt) ? opt.bind(publicThis, publicThis) : isFunction(opt.get) ? opt.get.bind(publicThis, publicThis) : NOOP;
        const set = !isFunction(opt) && isFunction(opt.set) ? opt.set.bind(publicThis) : NOOP;
        const c = computed({
          get,
          set
        });
        Object.defineProperty(ctx, key, {
          enumerable: true,
          configurable: true,
          get: () => c.value,
          set: (v) => c.value = v
        });
      }
    }
    if (watchOptions) {
      for (const key in watchOptions) {
        createWatcher(watchOptions[key], ctx, publicThis, key);
      }
    }
    if (provideOptions) {
      const provides = isFunction(provideOptions) ? provideOptions.call(publicThis) : provideOptions;
      Reflect.ownKeys(provides).forEach((key) => {
        provide(key, provides[key]);
      });
    }
    if (created) {
      callHook$1(created, instance, "c");
    }
    function registerLifecycleHook(register, hook) {
      if (isArray(hook)) {
        hook.forEach((_hook) => register(_hook.bind(publicThis)));
      } else if (hook) {
        register(hook.bind(publicThis));
      }
    }
    registerLifecycleHook(onBeforeMount, beforeMount);
    registerLifecycleHook(onMounted, mounted);
    registerLifecycleHook(onBeforeUpdate, beforeUpdate);
    registerLifecycleHook(onUpdated, updated);
    registerLifecycleHook(onActivated, activated);
    registerLifecycleHook(onDeactivated, deactivated);
    registerLifecycleHook(onErrorCaptured, errorCaptured);
    registerLifecycleHook(onRenderTracked, renderTracked);
    registerLifecycleHook(onRenderTriggered, renderTriggered);
    registerLifecycleHook(onBeforeUnmount, beforeUnmount);
    registerLifecycleHook(onUnmounted, unmounted);
    registerLifecycleHook(onServerPrefetch, serverPrefetch);
    if (isArray(expose)) {
      if (expose.length) {
        const exposed = instance.exposed || (instance.exposed = {});
        expose.forEach((key) => {
          Object.defineProperty(exposed, key, {
            get: () => publicThis[key],
            set: (val) => publicThis[key] = val,
            enumerable: true
          });
        });
      } else if (!instance.exposed) {
        instance.exposed = {};
      }
    }
    if (render2 && instance.render === NOOP) {
      instance.render = render2;
    }
    if (inheritAttrs != null) {
      instance.inheritAttrs = inheritAttrs;
    }
    if (components) instance.components = components;
    if (directives) instance.directives = directives;
    if (serverPrefetch) {
      markAsyncBoundary(instance);
    }
  }
  function resolveInjections(injectOptions, ctx, checkDuplicateProperties = NOOP) {
    if (isArray(injectOptions)) {
      injectOptions = normalizeInject(injectOptions);
    }
    for (const key in injectOptions) {
      const opt = injectOptions[key];
      let injected;
      if (isObject(opt)) {
        if ("default" in opt) {
          injected = inject(
            opt.from || key,
            opt.default,
            true
          );
        } else {
          injected = inject(opt.from || key);
        }
      } else {
        injected = inject(opt);
      }
      if (/* @__PURE__ */ isRef(injected)) {
        Object.defineProperty(ctx, key, {
          enumerable: true,
          configurable: true,
          get: () => injected.value,
          set: (v) => injected.value = v
        });
      } else {
        ctx[key] = injected;
      }
    }
  }
  function callHook$1(hook, instance, type) {
    callWithAsyncErrorHandling(
      isArray(hook) ? hook.map((h2) => h2.bind(instance.proxy)) : hook.bind(instance.proxy),
      instance,
      type
    );
  }
  function createWatcher(raw, ctx, publicThis, key) {
    let getter = key.includes(".") ? createPathGetter(publicThis, key) : () => publicThis[key];
    if (isString(raw)) {
      const handler = ctx[raw];
      if (isFunction(handler)) {
        {
          watch(getter, handler);
        }
      }
    } else if (isFunction(raw)) {
      {
        watch(getter, raw.bind(publicThis));
      }
    } else if (isObject(raw)) {
      if (isArray(raw)) {
        raw.forEach((r) => createWatcher(r, ctx, publicThis, key));
      } else {
        const handler = isFunction(raw.handler) ? raw.handler.bind(publicThis) : ctx[raw.handler];
        if (isFunction(handler)) {
          watch(getter, handler, raw);
        }
      }
    } else ;
  }
  function resolveMergedOptions(instance) {
    const base = instance.type;
    const { mixins, extends: extendsOptions } = base;
    const {
      mixins: globalMixins,
      optionsCache: cache,
      config: { optionMergeStrategies }
    } = instance.appContext;
    const cached = cache.get(base);
    let resolved;
    if (cached) {
      resolved = cached;
    } else if (!globalMixins.length && !mixins && !extendsOptions) {
      {
        resolved = base;
      }
    } else {
      resolved = {};
      if (globalMixins.length) {
        globalMixins.forEach(
          (m) => mergeOptions(resolved, m, optionMergeStrategies, true)
        );
      }
      mergeOptions(resolved, base, optionMergeStrategies);
    }
    if (isObject(base)) {
      cache.set(base, resolved);
    }
    return resolved;
  }
  function mergeOptions(to, from, strats, asMixin = false) {
    const { mixins, extends: extendsOptions } = from;
    if (extendsOptions) {
      mergeOptions(to, extendsOptions, strats, true);
    }
    if (mixins) {
      mixins.forEach(
        (m) => mergeOptions(to, m, strats, true)
      );
    }
    for (const key in from) {
      if (asMixin && key === "expose") ;
      else {
        const strat = internalOptionMergeStrats[key] || strats && strats[key];
        to[key] = strat ? strat(to[key], from[key]) : from[key];
      }
    }
    return to;
  }
  const internalOptionMergeStrats = {
    data: mergeDataFn,
    props: mergeEmitsOrPropsOptions,
    emits: mergeEmitsOrPropsOptions,
    // objects
    methods: mergeObjectOptions,
    computed: mergeObjectOptions,
    // lifecycle
    beforeCreate: mergeAsArray,
    created: mergeAsArray,
    beforeMount: mergeAsArray,
    mounted: mergeAsArray,
    beforeUpdate: mergeAsArray,
    updated: mergeAsArray,
    beforeDestroy: mergeAsArray,
    beforeUnmount: mergeAsArray,
    destroyed: mergeAsArray,
    unmounted: mergeAsArray,
    activated: mergeAsArray,
    deactivated: mergeAsArray,
    errorCaptured: mergeAsArray,
    serverPrefetch: mergeAsArray,
    // assets
    components: mergeObjectOptions,
    directives: mergeObjectOptions,
    // watch
    watch: mergeWatchOptions,
    // provide / inject
    provide: mergeDataFn,
    inject: mergeInject
  };
  function mergeDataFn(to, from) {
    if (!from) {
      return to;
    }
    if (!to) {
      return from;
    }
    return function mergedDataFn() {
      return extend(
        isFunction(to) ? to.call(this, this) : to,
        isFunction(from) ? from.call(this, this) : from
      );
    };
  }
  function mergeInject(to, from) {
    return mergeObjectOptions(normalizeInject(to), normalizeInject(from));
  }
  function normalizeInject(raw) {
    if (isArray(raw)) {
      const res = {};
      for (let i = 0; i < raw.length; i++) {
        res[raw[i]] = raw[i];
      }
      return res;
    }
    return raw;
  }
  function mergeAsArray(to, from) {
    return to ? [...new Set([].concat(to, from))] : from;
  }
  function mergeObjectOptions(to, from) {
    return to ? extend(/* @__PURE__ */ Object.create(null), to, from) : from;
  }
  function mergeEmitsOrPropsOptions(to, from) {
    if (to) {
      if (isArray(to) && isArray(from)) {
        return [.../* @__PURE__ */ new Set([...to, ...from])];
      }
      return extend(
        /* @__PURE__ */ Object.create(null),
        normalizePropsOrEmits(to),
        normalizePropsOrEmits(from != null ? from : {})
      );
    } else {
      return from;
    }
  }
  function mergeWatchOptions(to, from) {
    if (!to) return from;
    if (!from) return to;
    const merged = extend(/* @__PURE__ */ Object.create(null), to);
    for (const key in from) {
      merged[key] = mergeAsArray(to[key], from[key]);
    }
    return merged;
  }
  function createAppContext() {
    return {
      app: null,
      config: {
        isNativeTag: NO,
        performance: false,
        globalProperties: {},
        optionMergeStrategies: {},
        errorHandler: void 0,
        warnHandler: void 0,
        compilerOptions: {}
      },
      mixins: [],
      components: {},
      directives: {},
      provides: /* @__PURE__ */ Object.create(null),
      optionsCache: /* @__PURE__ */ new WeakMap(),
      propsCache: /* @__PURE__ */ new WeakMap(),
      emitsCache: /* @__PURE__ */ new WeakMap()
    };
  }
  let uid$1 = 0;
  function createAppAPI(render2, hydrate2) {
    return function createApp2(rootComponent, rootProps = null) {
      if (!isFunction(rootComponent)) {
        rootComponent = extend({}, rootComponent);
      }
      if (rootProps != null && !isObject(rootProps)) {
        rootProps = null;
      }
      const context = createAppContext();
      const installedPlugins = /* @__PURE__ */ new WeakSet();
      const pluginCleanupFns = [];
      let isMounted = false;
      const app = context.app = {
        _uid: uid$1++,
        _component: rootComponent,
        _props: rootProps,
        _container: null,
        _context: context,
        _instance: null,
        version,
        get config() {
          return context.config;
        },
        set config(v) {
        },
        use(plugin, ...options) {
          if (installedPlugins.has(plugin)) ;
          else if (plugin && isFunction(plugin.install)) {
            installedPlugins.add(plugin);
            plugin.install(app, ...options);
          } else if (isFunction(plugin)) {
            installedPlugins.add(plugin);
            plugin(app, ...options);
          } else ;
          return app;
        },
        mixin(mixin) {
          {
            if (!context.mixins.includes(mixin)) {
              context.mixins.push(mixin);
            }
          }
          return app;
        },
        component(name, component) {
          if (!component) {
            return context.components[name];
          }
          context.components[name] = component;
          return app;
        },
        directive(name, directive) {
          if (!directive) {
            return context.directives[name];
          }
          context.directives[name] = directive;
          return app;
        },
        mount(rootContainer, isHydrate, namespace) {
          if (!isMounted) {
            const vnode = app._ceVNode || createVNode(rootComponent, rootProps);
            vnode.appContext = context;
            if (namespace === true) {
              namespace = "svg";
            } else if (namespace === false) {
              namespace = void 0;
            }
            if (isHydrate && hydrate2) {
              hydrate2(vnode, rootContainer);
            } else {
              render2(vnode, rootContainer, namespace);
            }
            isMounted = true;
            app._container = rootContainer;
            rootContainer.__vue_app__ = app;
            return getComponentPublicInstance(vnode.component);
          }
        },
        onUnmount(cleanupFn) {
          pluginCleanupFns.push(cleanupFn);
        },
        unmount() {
          if (isMounted) {
            callWithAsyncErrorHandling(
              pluginCleanupFns,
              app._instance,
              16
            );
            render2(null, app._container);
            delete app._container.__vue_app__;
          }
        },
        provide(key, value) {
          context.provides[key] = value;
          return app;
        },
        runWithContext(fn) {
          const lastApp = currentApp;
          currentApp = app;
          try {
            return fn();
          } finally {
            currentApp = lastApp;
          }
        }
      };
      return app;
    };
  }
  let currentApp = null;
  function useModel(props, name, options = EMPTY_OBJ) {
    const i = getCurrentInstance();
    const camelizedName = camelize(name);
    const hyphenatedName = hyphenate(name);
    const modifiers = getModelModifiers(props, camelizedName);
    const res = customRef((track2, trigger2) => {
      let localValue;
      let prevSetValue = EMPTY_OBJ;
      let prevEmittedValue;
      watchSyncEffect(() => {
        const propValue = props[camelizedName];
        if (hasChanged(localValue, propValue)) {
          localValue = propValue;
          trigger2();
        }
      });
      return {
        get() {
          track2();
          return options.get ? options.get(localValue) : localValue;
        },
        set(value) {
          const emittedValue = options.set ? options.set(value) : value;
          if (!hasChanged(emittedValue, localValue) && !(prevSetValue !== EMPTY_OBJ && hasChanged(value, prevSetValue))) {
            return;
          }
          const rawProps = i.vnode.props;
          const hasVModel = !!(rawProps && // check if parent has passed v-model
          (name in rawProps || camelizedName in rawProps || hyphenatedName in rawProps) && (`onUpdate:${name}` in rawProps || `onUpdate:${camelizedName}` in rawProps || `onUpdate:${hyphenatedName}` in rawProps));
          if (!hasVModel) {
            localValue = value;
            trigger2();
          }
          i.emit(`update:${name}`, emittedValue);
          if (hasChanged(value, prevSetValue) && (hasChanged(value, emittedValue) && !hasChanged(emittedValue, prevEmittedValue) || // #13524: browsers differ in when they flush microtasks between
          // event listeners. If a v-model listener emits an intermediate value
          // and a following listener restores the model to its previous prop
          // value before parent updates are flushed, the parent render can be
          // deduped as having no prop change. Force a local update so DOM state
          // such as an input's value is synchronized back to the current model.
          hasVModel && prevSetValue !== EMPTY_OBJ && !hasChanged(emittedValue, localValue))) {
            trigger2();
          }
          prevSetValue = value;
          prevEmittedValue = emittedValue;
        }
      };
    });
    res[Symbol.iterator] = () => {
      let i2 = 0;
      return {
        next() {
          if (i2 < 2) {
            return { value: i2++ ? modifiers || EMPTY_OBJ : res, done: false };
          } else {
            return { done: true };
          }
        }
      };
    };
    return res;
  }
  const getModelModifiers = (props, modelName) => {
    return modelName === "modelValue" || modelName === "model-value" ? props.modelModifiers : props[`${modelName}Modifiers`] || props[`${camelize(modelName)}Modifiers`] || props[`${hyphenate(modelName)}Modifiers`];
  };
  function emit(instance, event, ...rawArgs) {
    if (instance.isUnmounted) return;
    const props = instance.vnode.props || EMPTY_OBJ;
    let args = rawArgs;
    const isModelListener2 = event.startsWith("update:");
    const modifiers = isModelListener2 && getModelModifiers(props, event.slice(7));
    if (modifiers) {
      if (modifiers.trim) {
        args = rawArgs.map((a) => isString(a) ? a.trim() : a);
      }
      if (modifiers.number) {
        args = rawArgs.map(looseToNumber);
      }
    }
    let handlerName;
    let handler = props[handlerName = toHandlerKey(event)] || // also try camelCase event handler (#2249)
    props[handlerName = toHandlerKey(camelize(event))];
    if (!handler && isModelListener2) {
      handler = props[handlerName = toHandlerKey(hyphenate(event))];
    }
    if (handler) {
      callWithAsyncErrorHandling(
        handler,
        instance,
        6,
        args
      );
    }
    const onceHandler = props[handlerName + `Once`];
    if (onceHandler) {
      if (!instance.emitted) {
        instance.emitted = {};
      } else if (instance.emitted[handlerName]) {
        return;
      }
      instance.emitted[handlerName] = true;
      callWithAsyncErrorHandling(
        onceHandler,
        instance,
        6,
        args
      );
    }
  }
  const mixinEmitsCache = /* @__PURE__ */ new WeakMap();
  function normalizeEmitsOptions(comp, appContext, asMixin = false) {
    const cache = asMixin ? mixinEmitsCache : appContext.emitsCache;
    const cached = cache.get(comp);
    if (cached !== void 0) {
      return cached;
    }
    const raw = comp.emits;
    let normalized = {};
    let hasExtends = false;
    if (!isFunction(comp)) {
      const extendEmits = (raw2) => {
        const normalizedFromExtend = normalizeEmitsOptions(raw2, appContext, true);
        if (normalizedFromExtend) {
          hasExtends = true;
          extend(normalized, normalizedFromExtend);
        }
      };
      if (!asMixin && appContext.mixins.length) {
        appContext.mixins.forEach(extendEmits);
      }
      if (comp.extends) {
        extendEmits(comp.extends);
      }
      if (comp.mixins) {
        comp.mixins.forEach(extendEmits);
      }
    }
    if (!raw && !hasExtends) {
      if (isObject(comp)) {
        cache.set(comp, null);
      }
      return null;
    }
    if (isArray(raw)) {
      raw.forEach((key) => normalized[key] = null);
    } else {
      extend(normalized, raw);
    }
    if (isObject(comp)) {
      cache.set(comp, normalized);
    }
    return normalized;
  }
  function isEmitListener(options, key) {
    if (!options || !isOn(key)) {
      return false;
    }
    key = key.slice(2);
    key = key === "Once" ? key : key.replace(/Once$/, "");
    return hasOwn(options, key[0].toLowerCase() + key.slice(1)) || hasOwn(options, hyphenate(key)) || hasOwn(options, key);
  }
  function markAttrsAccessed() {
  }
  function renderComponentRoot(instance) {
    const {
      type: Component,
      vnode,
      proxy,
      withProxy,
      propsOptions: [propsOptions],
      slots,
      attrs,
      emit: emit2,
      render: render2,
      renderCache,
      props,
      data,
      setupState,
      ctx,
      inheritAttrs
    } = instance;
    const prev = setCurrentRenderingInstance(instance);
    let result;
    let fallthroughAttrs;
    try {
      if (vnode.shapeFlag & 4) {
        const proxyToUse = withProxy || proxy;
        const thisProxy = false ? new Proxy(proxyToUse, {
          get(target, key, receiver) {
            warn$1(
              `Property '${String(
                key
              )}' was accessed via 'this'. Avoid using 'this' in templates.`
            );
            return Reflect.get(target, key, receiver);
          }
        }) : proxyToUse;
        result = normalizeVNode(
          render2.call(
            thisProxy,
            proxyToUse,
            renderCache,
            false ? /* @__PURE__ */ shallowReadonly(props) : props,
            setupState,
            data,
            ctx
          )
        );
        fallthroughAttrs = attrs;
      } else {
        const render22 = Component;
        if (false) ;
        result = normalizeVNode(
          render22.length > 1 ? render22(
            false ? /* @__PURE__ */ shallowReadonly(props) : props,
            false ? {
              get attrs() {
                markAttrsAccessed();
                return /* @__PURE__ */ shallowReadonly(attrs);
              },
              slots,
              emit: emit2
            } : { attrs, slots, emit: emit2 }
          ) : render22(
            false ? /* @__PURE__ */ shallowReadonly(props) : props,
            null
          )
        );
        fallthroughAttrs = Component.props ? attrs : getFunctionalFallthrough(attrs);
      }
    } catch (err) {
      blockStack.length = 0;
      handleError(err, instance, 1);
      result = createVNode(Comment);
    }
    let root = result;
    if (fallthroughAttrs && inheritAttrs !== false) {
      const keys = Object.keys(fallthroughAttrs);
      const { shapeFlag } = root;
      if (keys.length) {
        if (shapeFlag & (1 | 6)) {
          if (propsOptions && keys.some(isModelListener)) {
            fallthroughAttrs = filterModelListeners(
              fallthroughAttrs,
              propsOptions
            );
          }
          root = cloneVNode(root, fallthroughAttrs, false, true);
        }
      }
    }
    if (vnode.dirs) {
      root = cloneVNode(root, null, false, true);
      root.dirs = root.dirs ? root.dirs.concat(vnode.dirs) : vnode.dirs;
    }
    if (vnode.transition) {
      setTransitionHooks(root, vnode.transition);
    }
    {
      result = root;
    }
    setCurrentRenderingInstance(prev);
    return result;
  }
  function filterSingleRoot(children, recurse = true) {
    let singleRoot;
    for (let i = 0; i < children.length; i++) {
      const child = children[i];
      if (isVNode(child)) {
        if (child.type !== Comment || child.children === "v-if") {
          if (singleRoot) {
            return;
          } else {
            singleRoot = child;
          }
        }
      } else {
        return;
      }
    }
    return singleRoot;
  }
  const getFunctionalFallthrough = (attrs) => {
    let res;
    for (const key in attrs) {
      if (key === "class" || key === "style" || isOn(key)) {
        (res || (res = {}))[key] = attrs[key];
      }
    }
    return res;
  };
  const filterModelListeners = (attrs, props) => {
    const res = {};
    for (const key in attrs) {
      if (!isModelListener(key) || !(key.slice(9) in props)) {
        res[key] = attrs[key];
      }
    }
    return res;
  };
  function shouldUpdateComponent(prevVNode, nextVNode, optimized) {
    const { props: prevProps, children: prevChildren, component } = prevVNode;
    const { props: nextProps, children: nextChildren, patchFlag } = nextVNode;
    const emits = component.emitsOptions;
    if (nextVNode.dirs || nextVNode.transition) {
      return true;
    }
    if (optimized && patchFlag >= 0) {
      if (patchFlag & 1024) {
        return true;
      }
      if (patchFlag & 16) {
        if (!prevProps) {
          return !!nextProps;
        }
        return hasPropsChanged(prevProps, nextProps, emits);
      } else if (patchFlag & 8) {
        const dynamicProps = nextVNode.dynamicProps;
        for (let i = 0; i < dynamicProps.length; i++) {
          const key = dynamicProps[i];
          if (hasPropValueChanged(nextProps, prevProps, key) && !isEmitListener(emits, key)) {
            return true;
          }
        }
      }
    } else {
      if (prevChildren || nextChildren) {
        if (!nextChildren || !nextChildren.$stable) {
          return true;
        }
      }
      if (prevProps === nextProps) {
        return false;
      }
      if (!prevProps) {
        return !!nextProps;
      }
      if (!nextProps) {
        return true;
      }
      return hasPropsChanged(prevProps, nextProps, emits);
    }
    return false;
  }
  function hasPropsChanged(prevProps, nextProps, emitsOptions) {
    const nextKeys = Object.keys(nextProps);
    if (nextKeys.length !== Object.keys(prevProps).length) {
      return true;
    }
    for (let i = 0; i < nextKeys.length; i++) {
      const key = nextKeys[i];
      if (hasPropValueChanged(nextProps, prevProps, key) && !isEmitListener(emitsOptions, key)) {
        return true;
      }
    }
    return false;
  }
  function hasPropValueChanged(nextProps, prevProps, key) {
    const nextProp = nextProps[key];
    const prevProp = prevProps[key];
    if (key === "style" && isObject(nextProp) && isObject(prevProp)) {
      return !looseEqual(nextProp, prevProp);
    }
    return nextProp !== prevProp;
  }
  function updateHOCHostEl({ vnode, parent, suspense }, el) {
    while (parent) {
      const root = parent.subTree;
      if (root.suspense && root.suspense.activeBranch === vnode) {
        root.suspense.vnode.el = root.el = el;
        vnode = root;
      }
      if (root === vnode) {
        (vnode = parent.vnode).el = el;
        parent = parent.parent;
      } else {
        break;
      }
    }
    if (suspense && suspense.activeBranch === vnode) {
      suspense.vnode.el = el;
    }
  }
  const internalObjectProto = {};
  const createInternalObject = () => Object.create(internalObjectProto);
  const isInternalObject = (obj) => Object.getPrototypeOf(obj) === internalObjectProto;
  function initProps(instance, rawProps, isStateful, isSSR = false) {
    const props = {};
    const attrs = createInternalObject();
    instance.propsDefaults = /* @__PURE__ */ Object.create(null);
    setFullProps(instance, rawProps, props, attrs);
    for (const key in instance.propsOptions[0]) {
      if (!(key in props)) {
        props[key] = void 0;
      }
    }
    if (isStateful) {
      instance.props = isSSR ? props : /* @__PURE__ */ shallowReactive(props);
    } else {
      if (!instance.type.props) {
        instance.props = attrs;
      } else {
        instance.props = props;
      }
    }
    instance.attrs = attrs;
  }
  function updateProps(instance, rawProps, rawPrevProps, optimized) {
    const {
      props,
      attrs,
      vnode: { patchFlag }
    } = instance;
    const rawCurrentProps = /* @__PURE__ */ toRaw(props);
    const [options] = instance.propsOptions;
    let hasAttrsChanged = false;
    if (
      // always force full diff in dev
      // - #1942 if hmr is enabled with sfc component
      // - vite#872 non-sfc component used by sfc component
      (optimized || patchFlag > 0) && !(patchFlag & 16)
    ) {
      if (patchFlag & 8) {
        const propsToUpdate = instance.vnode.dynamicProps;
        for (let i = 0; i < propsToUpdate.length; i++) {
          let key = propsToUpdate[i];
          if (isEmitListener(instance.emitsOptions, key)) {
            continue;
          }
          const value = rawProps[key];
          if (options) {
            if (hasOwn(attrs, key)) {
              if (value !== attrs[key]) {
                attrs[key] = value;
                hasAttrsChanged = true;
              }
            } else {
              const camelizedKey = camelize(key);
              props[camelizedKey] = resolvePropValue(
                options,
                rawCurrentProps,
                camelizedKey,
                value,
                instance,
                false
              );
            }
          } else {
            if (value !== attrs[key]) {
              attrs[key] = value;
              hasAttrsChanged = true;
            }
          }
        }
      }
    } else {
      if (setFullProps(instance, rawProps, props, attrs)) {
        hasAttrsChanged = true;
      }
      let kebabKey;
      for (const key in rawCurrentProps) {
        if (!rawProps || // for camelCase
        !hasOwn(rawProps, key) && // it's possible the original props was passed in as kebab-case
        // and converted to camelCase (#955)
        ((kebabKey = hyphenate(key)) === key || !hasOwn(rawProps, kebabKey))) {
          if (options) {
            if (rawPrevProps && // for camelCase
            (rawPrevProps[key] !== void 0 || // for kebab-case
            rawPrevProps[kebabKey] !== void 0)) {
              props[key] = resolvePropValue(
                options,
                rawCurrentProps,
                key,
                void 0,
                instance,
                true
              );
            }
          } else {
            delete props[key];
          }
        }
      }
      if (attrs !== rawCurrentProps) {
        for (const key in attrs) {
          if (!rawProps || !hasOwn(rawProps, key) && true) {
            delete attrs[key];
            hasAttrsChanged = true;
          }
        }
      }
    }
    if (hasAttrsChanged) {
      trigger(instance.attrs, "set", "");
    }
  }
  function setFullProps(instance, rawProps, props, attrs) {
    const [options, needCastKeys] = instance.propsOptions;
    let hasAttrsChanged = false;
    let rawCastValues;
    if (rawProps) {
      for (let key in rawProps) {
        if (isReservedProp(key)) {
          continue;
        }
        const value = rawProps[key];
        let camelKey;
        if (options && hasOwn(options, camelKey = camelize(key))) {
          if (!needCastKeys || !needCastKeys.includes(camelKey)) {
            props[camelKey] = value;
          } else {
            (rawCastValues || (rawCastValues = {}))[camelKey] = value;
          }
        } else if (!isEmitListener(instance.emitsOptions, key)) {
          if (!(key in attrs) || value !== attrs[key]) {
            attrs[key] = value;
            hasAttrsChanged = true;
          }
        }
      }
    }
    if (needCastKeys) {
      const rawCurrentProps = /* @__PURE__ */ toRaw(props);
      const castValues = rawCastValues || EMPTY_OBJ;
      for (let i = 0; i < needCastKeys.length; i++) {
        const key = needCastKeys[i];
        props[key] = resolvePropValue(
          options,
          rawCurrentProps,
          key,
          castValues[key],
          instance,
          !hasOwn(castValues, key)
        );
      }
    }
    return hasAttrsChanged;
  }
  function resolvePropValue(options, props, key, value, instance, isAbsent) {
    const opt = options[key];
    if (opt != null) {
      const hasDefault = hasOwn(opt, "default");
      if (hasDefault && value === void 0) {
        const defaultValue = opt.default;
        if (opt.type !== Function && !opt.skipFactory && isFunction(defaultValue)) {
          const { propsDefaults } = instance;
          if (key in propsDefaults) {
            value = propsDefaults[key];
          } else {
            const reset = setCurrentInstance(instance);
            value = propsDefaults[key] = defaultValue.call(
              null,
              props
            );
            reset();
          }
        } else {
          value = defaultValue;
        }
        if (instance.ce) {
          instance.ce._setProp(key, value);
        }
      }
      if (opt[
        0
        /* shouldCast */
      ]) {
        if (isAbsent && !hasDefault) {
          value = false;
        } else if (opt[
          1
          /* shouldCastTrue */
        ] && (value === "" || value === hyphenate(key))) {
          value = true;
        }
      }
    }
    return value;
  }
  const mixinPropsCache = /* @__PURE__ */ new WeakMap();
  function normalizePropsOptions(comp, appContext, asMixin = false) {
    const cache = asMixin ? mixinPropsCache : appContext.propsCache;
    const cached = cache.get(comp);
    if (cached) {
      return cached;
    }
    const raw = comp.props;
    const normalized = {};
    const needCastKeys = [];
    let hasExtends = false;
    if (!isFunction(comp)) {
      const extendProps = (raw2) => {
        hasExtends = true;
        const [props, keys] = normalizePropsOptions(raw2, appContext, true);
        extend(normalized, props);
        if (keys) needCastKeys.push(...keys);
      };
      if (!asMixin && appContext.mixins.length) {
        appContext.mixins.forEach(extendProps);
      }
      if (comp.extends) {
        extendProps(comp.extends);
      }
      if (comp.mixins) {
        comp.mixins.forEach(extendProps);
      }
    }
    if (!raw && !hasExtends) {
      if (isObject(comp)) {
        cache.set(comp, EMPTY_ARR);
      }
      return EMPTY_ARR;
    }
    if (isArray(raw)) {
      for (let i = 0; i < raw.length; i++) {
        const normalizedKey = camelize(raw[i]);
        if (validatePropName(normalizedKey)) {
          normalized[normalizedKey] = EMPTY_OBJ;
        }
      }
    } else if (raw) {
      for (const key in raw) {
        const normalizedKey = camelize(key);
        if (validatePropName(normalizedKey)) {
          const opt = raw[key];
          const prop = normalized[normalizedKey] = isArray(opt) || isFunction(opt) ? { type: opt } : extend({}, opt);
          const propType = prop.type;
          let shouldCast = false;
          let shouldCastTrue = true;
          if (isArray(propType)) {
            for (let index = 0; index < propType.length; ++index) {
              const type = propType[index];
              const typeName = isFunction(type) && type.name;
              if (typeName === "Boolean") {
                shouldCast = true;
                break;
              } else if (typeName === "String") {
                shouldCastTrue = false;
              }
            }
          } else {
            shouldCast = isFunction(propType) && propType.name === "Boolean";
          }
          prop[
            0
            /* shouldCast */
          ] = shouldCast;
          prop[
            1
            /* shouldCastTrue */
          ] = shouldCastTrue;
          if (shouldCast || hasOwn(prop, "default")) {
            needCastKeys.push(normalizedKey);
          }
        }
      }
    }
    const res = [normalized, needCastKeys];
    if (isObject(comp)) {
      cache.set(comp, res);
    }
    return res;
  }
  function validatePropName(key) {
    if (key[0] !== "$" && !isReservedProp(key)) {
      return true;
    }
    return false;
  }
  const isInternalKey = (key) => key === "_" || key === "_ctx" || key === "$stable";
  const normalizeSlotValue = (value) => isArray(value) ? value.map(normalizeVNode) : [normalizeVNode(value)];
  const normalizeSlot = (key, rawSlot, ctx) => {
    if (rawSlot._n) {
      return rawSlot;
    }
    const normalized = withCtx((...args) => {
      if (false) ;
      return normalizeSlotValue(rawSlot(...args));
    }, ctx);
    normalized._c = false;
    return normalized;
  };
  const normalizeObjectSlots = (rawSlots, slots, instance) => {
    const ctx = rawSlots._ctx;
    for (const key in rawSlots) {
      if (isInternalKey(key)) continue;
      const value = rawSlots[key];
      if (isFunction(value)) {
        slots[key] = normalizeSlot(key, value, ctx);
      } else if (value != null) {
        const normalized = normalizeSlotValue(value);
        slots[key] = () => normalized;
      }
    }
  };
  const normalizeVNodeSlots = (instance, children) => {
    const normalized = normalizeSlotValue(children);
    instance.slots.default = () => normalized;
  };
  const assignSlots = (slots, children, optimized) => {
    for (const key in children) {
      if (optimized || !isInternalKey(key)) {
        slots[key] = children[key];
      }
    }
  };
  const initSlots = (instance, children, optimized) => {
    const slots = instance.slots = createInternalObject();
    if (instance.vnode.shapeFlag & 32) {
      const type = children._;
      if (type) {
        assignSlots(slots, children, optimized);
        if (optimized) {
          def(slots, "_", type, true);
        }
      } else {
        normalizeObjectSlots(children, slots);
      }
    } else if (children) {
      normalizeVNodeSlots(instance, children);
    }
  };
  const updateSlots = (instance, children, optimized) => {
    const { vnode, slots } = instance;
    let needDeletionCheck = true;
    let deletionComparisonTarget = EMPTY_OBJ;
    if (vnode.shapeFlag & 32) {
      const type = children._;
      if (type) {
        if (optimized && type === 1) {
          needDeletionCheck = false;
        } else {
          assignSlots(slots, children, optimized);
        }
      } else {
        needDeletionCheck = !children.$stable;
        normalizeObjectSlots(children, slots);
      }
      deletionComparisonTarget = children;
    } else if (children) {
      normalizeVNodeSlots(instance, children);
      deletionComparisonTarget = { default: 1 };
    }
    if (needDeletionCheck) {
      for (const key in slots) {
        if (!isInternalKey(key) && deletionComparisonTarget[key] == null) {
          delete slots[key];
        }
      }
    }
  };
  const queuePostRenderEffect = queueEffectWithSuspense;
  function createRenderer(options) {
    return baseCreateRenderer(options);
  }
  function createHydrationRenderer(options) {
    return baseCreateRenderer(options, createHydrationFunctions);
  }
  function baseCreateRenderer(options, createHydrationFns) {
    const target = getGlobalThis();
    target.__VUE__ = true;
    const {
      insert: hostInsert,
      remove: hostRemove,
      patchProp: hostPatchProp,
      createElement: hostCreateElement,
      createText: hostCreateText,
      createComment: hostCreateComment,
      setText: hostSetText,
      setElementText: hostSetElementText,
      parentNode: hostParentNode,
      nextSibling: hostNextSibling,
      setScopeId: hostSetScopeId = NOOP,
      insertStaticContent: hostInsertStaticContent
    } = options;
    const patch = (n1, n2, container, anchor = null, parentComponent = null, parentSuspense = null, namespace = void 0, slotScopeIds = null, optimized = !!n2.dynamicChildren) => {
      if (n1 === n2) {
        return;
      }
      if (n1 && !isSameVNodeType(n1, n2)) {
        anchor = getNextHostNode(n1);
        unmount(n1, parentComponent, parentSuspense, true);
        n1 = null;
      }
      if (n2.patchFlag === -2) {
        optimized = false;
        n2.dynamicChildren = null;
      }
      const { type, ref: ref3, shapeFlag } = n2;
      switch (type) {
        case Text:
          processText(n1, n2, container, anchor);
          break;
        case Comment:
          processCommentNode(n1, n2, container, anchor);
          break;
        case Static:
          if (n1 == null) {
            mountStaticNode(n2, container, anchor, namespace);
          }
          break;
        case Fragment:
          processFragment(
            n1,
            n2,
            container,
            anchor,
            parentComponent,
            parentSuspense,
            namespace,
            slotScopeIds,
            optimized
          );
          break;
        default:
          if (shapeFlag & 1) {
            processElement(
              n1,
              n2,
              container,
              anchor,
              parentComponent,
              parentSuspense,
              namespace,
              slotScopeIds,
              optimized
            );
          } else if (shapeFlag & 6) {
            processComponent(
              n1,
              n2,
              container,
              anchor,
              parentComponent,
              parentSuspense,
              namespace,
              slotScopeIds,
              optimized
            );
          } else if (shapeFlag & 64) {
            type.process(
              n1,
              n2,
              container,
              anchor,
              parentComponent,
              parentSuspense,
              namespace,
              slotScopeIds,
              optimized,
              internals
            );
          } else if (shapeFlag & 128) {
            type.process(
              n1,
              n2,
              container,
              anchor,
              parentComponent,
              parentSuspense,
              namespace,
              slotScopeIds,
              optimized,
              internals
            );
          } else ;
      }
      if (ref3 != null && parentComponent) {
        setRef(ref3, n1 && n1.ref, parentSuspense, n2 || n1, !n2);
      } else if (ref3 == null && n1 && n1.ref != null) {
        setRef(n1.ref, null, parentSuspense, n1, true);
      }
    };
    const processText = (n1, n2, container, anchor) => {
      if (n1 == null) {
        hostInsert(
          n2.el = hostCreateText(n2.children),
          container,
          anchor
        );
      } else {
        const el = n2.el = n1.el;
        if (n2.children !== n1.children) {
          hostSetText(el, n2.children);
        }
      }
    };
    const processCommentNode = (n1, n2, container, anchor) => {
      if (n1 == null) {
        hostInsert(
          n2.el = hostCreateComment(n2.children || ""),
          container,
          anchor
        );
      } else {
        n2.el = n1.el;
      }
    };
    const mountStaticNode = (n2, container, anchor, namespace) => {
      [n2.el, n2.anchor] = hostInsertStaticContent(
        n2.children,
        container,
        anchor,
        namespace,
        n2.el,
        n2.anchor
      );
    };
    const moveStaticNode = ({ el, anchor }, container, nextSibling) => {
      let next;
      while (el && el !== anchor) {
        next = hostNextSibling(el);
        hostInsert(el, container, nextSibling);
        el = next;
      }
      hostInsert(anchor, container, nextSibling);
    };
    const removeStaticNode = ({ el, anchor }) => {
      let next;
      while (el && el !== anchor) {
        next = hostNextSibling(el);
        hostRemove(el);
        el = next;
      }
      hostRemove(anchor);
    };
    const processElement = (n1, n2, container, anchor, parentComponent, parentSuspense, namespace, slotScopeIds, optimized) => {
      if (n2.type === "svg") {
        namespace = "svg";
      } else if (n2.type === "math") {
        namespace = "mathml";
      }
      if (n1 == null) {
        mountElement(
          n2,
          container,
          anchor,
          parentComponent,
          parentSuspense,
          namespace,
          slotScopeIds,
          optimized
        );
      } else {
        const customElement = n1.el && n1.el._isVueCE ? n1.el : null;
        try {
          if (customElement) {
            customElement._beginPatch();
          }
          patchElement(
            n1,
            n2,
            parentComponent,
            parentSuspense,
            namespace,
            slotScopeIds,
            optimized
          );
        } finally {
          if (customElement) {
            customElement._endPatch();
          }
        }
      }
    };
    const mountElement = (vnode, container, anchor, parentComponent, parentSuspense, namespace, slotScopeIds, optimized) => {
      let el;
      let vnodeHook;
      const { props, shapeFlag, transition, dirs } = vnode;
      el = vnode.el = hostCreateElement(
        vnode.type,
        namespace,
        props && props.is,
        props
      );
      if (shapeFlag & 8) {
        hostSetElementText(el, vnode.children);
      } else if (shapeFlag & 16) {
        mountChildren(
          vnode.children,
          el,
          null,
          parentComponent,
          parentSuspense,
          resolveChildrenNamespace(vnode, namespace),
          slotScopeIds,
          optimized
        );
      }
      if (dirs) {
        invokeDirectiveHook(vnode, null, parentComponent, "created");
      }
      setScopeId(el, vnode, vnode.scopeId, slotScopeIds, parentComponent);
      if (props) {
        for (const key in props) {
          if (key !== "value" && !isReservedProp(key)) {
            hostPatchProp(el, key, null, props[key], namespace, parentComponent);
          }
        }
        if ("value" in props) {
          hostPatchProp(el, "value", null, props.value, namespace);
        }
        if (vnodeHook = props.onVnodeBeforeMount) {
          invokeVNodeHook(vnodeHook, parentComponent, vnode);
        }
      }
      if (dirs) {
        invokeDirectiveHook(vnode, null, parentComponent, "beforeMount");
      }
      const needCallTransitionHooks = needTransition(parentSuspense, transition);
      if (needCallTransitionHooks) {
        transition.beforeEnter(el);
      }
      hostInsert(el, container, anchor);
      if ((vnodeHook = props && props.onVnodeMounted) || needCallTransitionHooks || dirs) {
        queuePostRenderEffect(() => {
          try {
            vnodeHook && invokeVNodeHook(vnodeHook, parentComponent, vnode);
            needCallTransitionHooks && transition.enter(el);
            dirs && invokeDirectiveHook(vnode, null, parentComponent, "mounted");
          } finally {
          }
        }, parentSuspense);
      }
    };
    const setScopeId = (el, vnode, scopeId, slotScopeIds, parentComponent) => {
      if (scopeId) {
        hostSetScopeId(el, scopeId);
      }
      if (slotScopeIds) {
        for (let i = 0; i < slotScopeIds.length; i++) {
          hostSetScopeId(el, slotScopeIds[i]);
        }
      }
      if (parentComponent) {
        let subTree = parentComponent.subTree;
        if (vnode === subTree || isSuspense(subTree.type) && (subTree.ssContent === vnode || subTree.ssFallback === vnode)) {
          const parentVNode = parentComponent.vnode;
          setScopeId(
            el,
            parentVNode,
            parentVNode.scopeId,
            parentVNode.slotScopeIds,
            parentComponent.parent
          );
        }
      }
    };
    const mountChildren = (children, container, anchor, parentComponent, parentSuspense, namespace, slotScopeIds, optimized, start = 0) => {
      for (let i = start; i < children.length; i++) {
        const child = children[i] = optimized ? cloneIfMounted(children[i]) : normalizeVNode(children[i]);
        patch(
          null,
          child,
          container,
          anchor,
          parentComponent,
          parentSuspense,
          namespace,
          slotScopeIds,
          optimized
        );
      }
    };
    const patchElement = (n1, n2, parentComponent, parentSuspense, namespace, slotScopeIds, optimized) => {
      const el = n2.el = n1.el;
      let { patchFlag, dynamicChildren, dirs } = n2;
      patchFlag |= n1.patchFlag & 16;
      const oldProps = n1.props || EMPTY_OBJ;
      const newProps = n2.props || EMPTY_OBJ;
      let vnodeHook;
      parentComponent && toggleRecurse(parentComponent, false);
      if (vnodeHook = newProps.onVnodeBeforeUpdate) {
        invokeVNodeHook(vnodeHook, parentComponent, n2, n1);
      }
      if (dirs) {
        invokeDirectiveHook(n2, n1, parentComponent, "beforeUpdate");
      }
      parentComponent && toggleRecurse(parentComponent, true);
      if (
        // #6385 the old vnode may be a user-wrapped non-isomorphic block
        // Force full diff when block metadata is unstable.
        dynamicChildren && (!n1.dynamicChildren || n1.dynamicChildren.length !== dynamicChildren.length)
      ) {
        patchFlag = 0;
        optimized = false;
        dynamicChildren = null;
      }
      if (oldProps.innerHTML && newProps.innerHTML == null || oldProps.textContent && newProps.textContent == null) {
        hostSetElementText(el, "");
      }
      if (dynamicChildren) {
        patchBlockChildren(
          n1.dynamicChildren,
          dynamicChildren,
          el,
          parentComponent,
          parentSuspense,
          resolveChildrenNamespace(n2, namespace),
          slotScopeIds
        );
      } else if (!optimized) {
        patchChildren(
          n1,
          n2,
          el,
          null,
          parentComponent,
          parentSuspense,
          resolveChildrenNamespace(n2, namespace),
          slotScopeIds,
          false
        );
      }
      if (patchFlag > 0) {
        if (patchFlag & 16) {
          patchProps(el, oldProps, newProps, parentComponent, namespace);
        } else {
          if (patchFlag & 2) {
            if (oldProps.class !== newProps.class) {
              hostPatchProp(el, "class", null, newProps.class, namespace);
            }
          }
          if (patchFlag & 4) {
            hostPatchProp(el, "style", oldProps.style, newProps.style, namespace);
          }
          if (patchFlag & 8) {
            const propsToUpdate = n2.dynamicProps;
            for (let i = 0; i < propsToUpdate.length; i++) {
              const key = propsToUpdate[i];
              const prev = oldProps[key];
              const next = newProps[key];
              if (next !== prev || key === "value") {
                hostPatchProp(el, key, prev, next, namespace, parentComponent);
              }
            }
          }
        }
        if (patchFlag & 1) {
          if (n1.children !== n2.children) {
            hostSetElementText(el, n2.children);
          }
        }
      } else if (!optimized && dynamicChildren == null) {
        patchProps(el, oldProps, newProps, parentComponent, namespace);
      }
      if ((vnodeHook = newProps.onVnodeUpdated) || dirs) {
        queuePostRenderEffect(() => {
          vnodeHook && invokeVNodeHook(vnodeHook, parentComponent, n2, n1);
          dirs && invokeDirectiveHook(n2, n1, parentComponent, "updated");
        }, parentSuspense);
      }
    };
    const patchBlockChildren = (oldChildren, newChildren, fallbackContainer, parentComponent, parentSuspense, namespace, slotScopeIds) => {
      for (let i = 0; i < newChildren.length; i++) {
        const oldVNode = oldChildren[i];
        const newVNode = newChildren[i];
        const container = (
          // oldVNode may be an errored async setup() component inside Suspense
          // which will not have a mounted element
          oldVNode.el && // - In the case of a Fragment, we need to provide the actual parent
          // of the Fragment itself so it can move its children.
          (oldVNode.type === Fragment || // - In the case of different nodes, there is going to be a replacement
          // which also requires the correct parent container
          !isSameVNodeType(oldVNode, newVNode) || // - In the case of a component, it could contain anything.
          oldVNode.shapeFlag & (6 | 64 | 128)) ? hostParentNode(oldVNode.el) : (
            // In other cases, the parent container is not actually used so we
            // just pass the block element here to avoid a DOM parentNode call.
            fallbackContainer
          )
        );
        patch(
          oldVNode,
          newVNode,
          container,
          null,
          parentComponent,
          parentSuspense,
          namespace,
          slotScopeIds,
          true
        );
      }
    };
    const patchProps = (el, oldProps, newProps, parentComponent, namespace) => {
      if (oldProps !== newProps) {
        if (oldProps !== EMPTY_OBJ) {
          for (const key in oldProps) {
            if (!isReservedProp(key) && !(key in newProps)) {
              hostPatchProp(
                el,
                key,
                oldProps[key],
                null,
                namespace,
                parentComponent
              );
            }
          }
        }
        for (const key in newProps) {
          if (isReservedProp(key)) continue;
          const next = newProps[key];
          const prev = oldProps[key];
          if (next !== prev && key !== "value") {
            hostPatchProp(el, key, prev, next, namespace, parentComponent);
          }
        }
        if ("value" in newProps) {
          hostPatchProp(el, "value", oldProps.value, newProps.value, namespace);
        }
      }
    };
    const processFragment = (n1, n2, container, anchor, parentComponent, parentSuspense, namespace, slotScopeIds, optimized) => {
      const fragmentStartAnchor = n2.el = n1 ? n1.el : hostCreateText("");
      const fragmentEndAnchor = n2.anchor = n1 ? n1.anchor : hostCreateText("");
      let { patchFlag, dynamicChildren, slotScopeIds: fragmentSlotScopeIds } = n2;
      if (fragmentSlotScopeIds) {
        slotScopeIds = slotScopeIds ? slotScopeIds.concat(fragmentSlotScopeIds) : fragmentSlotScopeIds;
      }
      if (n1 == null) {
        hostInsert(fragmentStartAnchor, container, anchor);
        hostInsert(fragmentEndAnchor, container, anchor);
        mountChildren(
          // #10007
          // such fragment like `<></>` will be compiled into
          // a fragment which doesn't have a children.
          // In this case fallback to an empty array
          n2.children || [],
          container,
          fragmentEndAnchor,
          parentComponent,
          parentSuspense,
          namespace,
          slotScopeIds,
          optimized
        );
      } else {
        if (patchFlag > 0 && patchFlag & 64 && dynamicChildren && // #2715 the previous fragment could've been a BAILed one as a result
        // of renderSlot() with no valid children
        n1.dynamicChildren && n1.dynamicChildren.length === dynamicChildren.length) {
          patchBlockChildren(
            n1.dynamicChildren,
            dynamicChildren,
            container,
            parentComponent,
            parentSuspense,
            namespace,
            slotScopeIds
          );
          if (
            // #2080 if the stable fragment has a key, it's a <template v-for> that may
            //  get moved around. Make sure all root level vnodes inherit el.
            // #2134 or if it's a component root, it may also get moved around
            // as the component is being moved.
            n2.key != null || parentComponent && n2 === parentComponent.subTree
          ) {
            traverseStaticChildren(
              n1,
              n2,
              true
              /* shallow */
            );
          }
        } else {
          patchChildren(
            n1,
            n2,
            container,
            fragmentEndAnchor,
            parentComponent,
            parentSuspense,
            namespace,
            slotScopeIds,
            optimized
          );
        }
      }
    };
    const processComponent = (n1, n2, container, anchor, parentComponent, parentSuspense, namespace, slotScopeIds, optimized) => {
      n2.slotScopeIds = slotScopeIds;
      if (n1 == null) {
        if (n2.shapeFlag & 512) {
          parentComponent.ctx.activate(
            n2,
            container,
            anchor,
            namespace,
            optimized
          );
        } else {
          mountComponent(
            n2,
            container,
            anchor,
            parentComponent,
            parentSuspense,
            namespace,
            optimized
          );
        }
      } else {
        updateComponent(n1, n2, optimized);
      }
    };
    const mountComponent = (initialVNode, container, anchor, parentComponent, parentSuspense, namespace, optimized) => {
      const instance = initialVNode.component = createComponentInstance(
        initialVNode,
        parentComponent,
        parentSuspense
      );
      if (isKeepAlive(initialVNode)) {
        instance.ctx.renderer = internals;
      }
      {
        setupComponent(instance, false, optimized);
      }
      if (instance.asyncDep) {
        parentSuspense && parentSuspense.registerDep(instance, setupRenderEffect, optimized);
        if (!initialVNode.el) {
          const placeholder = instance.subTree = createVNode(Comment);
          processCommentNode(null, placeholder, container, anchor);
          initialVNode.placeholder = placeholder.el;
        }
      } else {
        setupRenderEffect(
          instance,
          initialVNode,
          container,
          anchor,
          parentSuspense,
          namespace,
          optimized
        );
      }
    };
    const updateComponent = (n1, n2, optimized) => {
      const instance = n2.component = n1.component;
      if (shouldUpdateComponent(n1, n2, optimized)) {
        if (instance.asyncDep && !instance.asyncResolved) {
          updateComponentPreRender(instance, n2, optimized);
          return;
        } else {
          instance.next = n2;
          instance.update();
        }
      } else {
        n2.el = n1.el;
        instance.vnode = n2;
      }
    };
    const setupRenderEffect = (instance, initialVNode, container, anchor, parentSuspense, namespace, optimized) => {
      const componentUpdateFn = () => {
        if (!instance.isMounted) {
          let vnodeHook;
          const { el, props } = initialVNode;
          const { bm, m, parent, root, type } = instance;
          const isAsyncWrapperVNode = isAsyncWrapper(initialVNode);
          toggleRecurse(instance, false);
          if (bm) {
            invokeArrayFns(bm);
          }
          if (!isAsyncWrapperVNode && (vnodeHook = props && props.onVnodeBeforeMount)) {
            invokeVNodeHook(vnodeHook, parent, initialVNode);
          }
          toggleRecurse(instance, true);
          if (el && hydrateNode) {
            const hydrateSubTree = () => {
              instance.subTree = renderComponentRoot(instance);
              hydrateNode(
                el,
                instance.subTree,
                instance,
                parentSuspense,
                null
              );
            };
            if (isAsyncWrapperVNode && type.__asyncHydrate) {
              type.__asyncHydrate(
                el,
                instance,
                hydrateSubTree
              );
            } else {
              hydrateSubTree();
            }
          } else {
            if (root.ce && root.ce._hasShadowRoot()) {
              root.ce._injectChildStyle(
                type,
                instance.parent ? instance.parent.type : void 0
              );
            }
            const subTree = instance.subTree = renderComponentRoot(instance);
            patch(
              null,
              subTree,
              container,
              anchor,
              instance,
              parentSuspense,
              namespace
            );
            initialVNode.el = subTree.el;
          }
          if (m) {
            queuePostRenderEffect(m, parentSuspense);
          }
          if (!isAsyncWrapperVNode && (vnodeHook = props && props.onVnodeMounted)) {
            const scopedInitialVNode = initialVNode;
            queuePostRenderEffect(
              () => invokeVNodeHook(vnodeHook, parent, scopedInitialVNode),
              parentSuspense
            );
          }
          if (initialVNode.shapeFlag & 256 || parent && isAsyncWrapper(parent.vnode) && parent.vnode.shapeFlag & 256) {
            instance.a && queuePostRenderEffect(instance.a, parentSuspense);
          }
          instance.isMounted = true;
          initialVNode = container = anchor = null;
        } else {
          let { next, bu, u, parent, vnode } = instance;
          {
            const nonHydratedAsyncRoot = locateNonHydratedAsyncRoot(instance);
            if (nonHydratedAsyncRoot) {
              if (next) {
                next.el = vnode.el;
                updateComponentPreRender(instance, next, optimized);
              }
              nonHydratedAsyncRoot.asyncDep.then(() => {
                queuePostRenderEffect(() => {
                  if (!instance.isUnmounted) update();
                }, parentSuspense);
              });
              return;
            }
          }
          let originNext = next;
          let vnodeHook;
          toggleRecurse(instance, false);
          if (next) {
            next.el = vnode.el;
            updateComponentPreRender(instance, next, optimized);
          } else {
            next = vnode;
          }
          if (bu) {
            invokeArrayFns(bu);
          }
          if (vnodeHook = next.props && next.props.onVnodeBeforeUpdate) {
            invokeVNodeHook(vnodeHook, parent, next, vnode);
          }
          toggleRecurse(instance, true);
          const nextTree = renderComponentRoot(instance);
          const prevTree = instance.subTree;
          instance.subTree = nextTree;
          patch(
            prevTree,
            nextTree,
            // parent may have changed if it's in a teleport
            hostParentNode(prevTree.el),
            // anchor may have changed if it's in a fragment
            getNextHostNode(prevTree),
            instance,
            parentSuspense,
            namespace
          );
          next.el = nextTree.el;
          if (originNext === null) {
            updateHOCHostEl(instance, nextTree.el);
          }
          if (u) {
            queuePostRenderEffect(u, parentSuspense);
          }
          if (vnodeHook = next.props && next.props.onVnodeUpdated) {
            queuePostRenderEffect(
              () => invokeVNodeHook(vnodeHook, parent, next, vnode),
              parentSuspense
            );
          }
        }
      };
      instance.scope.on();
      const effect2 = instance.effect = new ReactiveEffect(componentUpdateFn);
      instance.scope.off();
      const update = instance.update = effect2.run.bind(effect2);
      const job = instance.job = effect2.runIfDirty.bind(effect2);
      job.i = instance;
      job.id = instance.uid;
      effect2.scheduler = () => queueJob(job);
      toggleRecurse(instance, true);
      update();
    };
    const updateComponentPreRender = (instance, nextVNode, optimized) => {
      nextVNode.component = instance;
      const prevProps = instance.vnode.props;
      instance.vnode = nextVNode;
      instance.next = null;
      updateProps(instance, nextVNode.props, prevProps, optimized);
      updateSlots(instance, nextVNode.children, optimized);
      pauseTracking();
      flushPreFlushCbs(instance);
      resetTracking();
    };
    const patchChildren = (n1, n2, container, anchor, parentComponent, parentSuspense, namespace, slotScopeIds, optimized = false) => {
      const c1 = n1 && n1.children;
      const prevShapeFlag = n1 ? n1.shapeFlag : 0;
      const c2 = n2.children;
      const { patchFlag, shapeFlag } = n2;
      if (patchFlag > 0) {
        if (patchFlag & 128) {
          patchKeyedChildren(
            c1,
            c2,
            container,
            anchor,
            parentComponent,
            parentSuspense,
            namespace,
            slotScopeIds,
            optimized
          );
          return;
        } else if (patchFlag & 256) {
          patchUnkeyedChildren(
            c1,
            c2,
            container,
            anchor,
            parentComponent,
            parentSuspense,
            namespace,
            slotScopeIds,
            optimized
          );
          return;
        }
      }
      if (shapeFlag & 8) {
        if (prevShapeFlag & 16) {
          unmountChildren(c1, parentComponent, parentSuspense);
        }
        if (c2 !== c1) {
          hostSetElementText(container, c2);
        }
      } else {
        if (prevShapeFlag & 16) {
          if (shapeFlag & 16) {
            patchKeyedChildren(
              c1,
              c2,
              container,
              anchor,
              parentComponent,
              parentSuspense,
              namespace,
              slotScopeIds,
              optimized
            );
          } else {
            unmountChildren(c1, parentComponent, parentSuspense, true);
          }
        } else {
          if (prevShapeFlag & 8) {
            hostSetElementText(container, "");
          }
          if (shapeFlag & 16) {
            mountChildren(
              c2,
              container,
              anchor,
              parentComponent,
              parentSuspense,
              namespace,
              slotScopeIds,
              optimized
            );
          }
        }
      }
    };
    const patchUnkeyedChildren = (c1, c2, container, anchor, parentComponent, parentSuspense, namespace, slotScopeIds, optimized) => {
      c1 = c1 || EMPTY_ARR;
      c2 = c2 || EMPTY_ARR;
      const oldLength = c1.length;
      const newLength = c2.length;
      const commonLength = Math.min(oldLength, newLength);
      let i;
      for (i = 0; i < commonLength; i++) {
        const nextChild = c2[i] = optimized ? cloneIfMounted(c2[i]) : normalizeVNode(c2[i]);
        patch(
          c1[i],
          nextChild,
          container,
          null,
          parentComponent,
          parentSuspense,
          namespace,
          slotScopeIds,
          optimized
        );
      }
      if (oldLength > newLength) {
        unmountChildren(
          c1,
          parentComponent,
          parentSuspense,
          true,
          false,
          commonLength
        );
      } else {
        mountChildren(
          c2,
          container,
          anchor,
          parentComponent,
          parentSuspense,
          namespace,
          slotScopeIds,
          optimized,
          commonLength
        );
      }
    };
    const patchKeyedChildren = (c1, c2, container, parentAnchor, parentComponent, parentSuspense, namespace, slotScopeIds, optimized) => {
      let i = 0;
      const l2 = c2.length;
      let e1 = c1.length - 1;
      let e2 = l2 - 1;
      while (i <= e1 && i <= e2) {
        const n1 = c1[i];
        const n2 = c2[i] = optimized ? cloneIfMounted(c2[i]) : normalizeVNode(c2[i]);
        if (isSameVNodeType(n1, n2)) {
          patch(
            n1,
            n2,
            container,
            null,
            parentComponent,
            parentSuspense,
            namespace,
            slotScopeIds,
            optimized
          );
        } else {
          break;
        }
        i++;
      }
      while (i <= e1 && i <= e2) {
        const n1 = c1[e1];
        const n2 = c2[e2] = optimized ? cloneIfMounted(c2[e2]) : normalizeVNode(c2[e2]);
        if (isSameVNodeType(n1, n2)) {
          patch(
            n1,
            n2,
            container,
            null,
            parentComponent,
            parentSuspense,
            namespace,
            slotScopeIds,
            optimized
          );
        } else {
          break;
        }
        e1--;
        e2--;
      }
      if (i > e1) {
        if (i <= e2) {
          const nextPos = e2 + 1;
          const anchor = nextPos < l2 ? c2[nextPos].el : parentAnchor;
          while (i <= e2) {
            patch(
              null,
              c2[i] = optimized ? cloneIfMounted(c2[i]) : normalizeVNode(c2[i]),
              container,
              anchor,
              parentComponent,
              parentSuspense,
              namespace,
              slotScopeIds,
              optimized
            );
            i++;
          }
        }
      } else if (i > e2) {
        while (i <= e1) {
          unmount(c1[i], parentComponent, parentSuspense, true);
          i++;
        }
      } else {
        const s1 = i;
        const s2 = i;
        const keyToNewIndexMap = /* @__PURE__ */ new Map();
        for (i = s2; i <= e2; i++) {
          const nextChild = c2[i] = optimized ? cloneIfMounted(c2[i]) : normalizeVNode(c2[i]);
          if (nextChild.key != null) {
            keyToNewIndexMap.set(nextChild.key, i);
          }
        }
        let j;
        let patched = 0;
        const toBePatched = e2 - s2 + 1;
        let moved = false;
        let maxNewIndexSoFar = 0;
        const newIndexToOldIndexMap = new Array(toBePatched);
        for (i = 0; i < toBePatched; i++) newIndexToOldIndexMap[i] = 0;
        for (i = s1; i <= e1; i++) {
          const prevChild = c1[i];
          if (patched >= toBePatched) {
            unmount(prevChild, parentComponent, parentSuspense, true);
            continue;
          }
          let newIndex;
          if (prevChild.key != null) {
            newIndex = keyToNewIndexMap.get(prevChild.key);
          } else {
            for (j = s2; j <= e2; j++) {
              if (newIndexToOldIndexMap[j - s2] === 0 && isSameVNodeType(prevChild, c2[j])) {
                newIndex = j;
                break;
              }
            }
          }
          if (newIndex === void 0) {
            unmount(prevChild, parentComponent, parentSuspense, true);
          } else {
            newIndexToOldIndexMap[newIndex - s2] = i + 1;
            if (newIndex >= maxNewIndexSoFar) {
              maxNewIndexSoFar = newIndex;
            } else {
              moved = true;
            }
            patch(
              prevChild,
              c2[newIndex],
              container,
              null,
              parentComponent,
              parentSuspense,
              namespace,
              slotScopeIds,
              optimized
            );
            patched++;
          }
        }
        const increasingNewIndexSequence = moved ? getSequence(newIndexToOldIndexMap) : EMPTY_ARR;
        j = increasingNewIndexSequence.length - 1;
        for (i = toBePatched - 1; i >= 0; i--) {
          const nextIndex = s2 + i;
          const nextChild = c2[nextIndex];
          const anchorVNode = c2[nextIndex + 1];
          const anchor = nextIndex + 1 < l2 ? (
            // #13559, #14173 fallback to el placeholder for unresolved async component
            anchorVNode.el || resolveAsyncComponentPlaceholder(anchorVNode)
          ) : parentAnchor;
          if (newIndexToOldIndexMap[i] === 0) {
            patch(
              null,
              nextChild,
              container,
              anchor,
              parentComponent,
              parentSuspense,
              namespace,
              slotScopeIds,
              optimized
            );
          } else if (moved) {
            if (j < 0 || i !== increasingNewIndexSequence[j]) {
              move(nextChild, container, anchor, 2);
            } else {
              j--;
            }
          }
        }
      }
    };
    const move = (vnode, container, anchor, moveType, parentSuspense = null) => {
      const { el, type, transition, children, shapeFlag } = vnode;
      if (shapeFlag & 6) {
        move(vnode.component.subTree, container, anchor, moveType);
        return;
      }
      if (shapeFlag & 128) {
        vnode.suspense.move(container, anchor, moveType);
        return;
      }
      if (shapeFlag & 64) {
        type.move(vnode, container, anchor, internals);
        return;
      }
      if (type === Fragment) {
        hostInsert(el, container, anchor);
        for (let i = 0; i < children.length; i++) {
          move(children[i], container, anchor, moveType);
        }
        hostInsert(vnode.anchor, container, anchor);
        return;
      }
      if (type === Static) {
        moveStaticNode(vnode, container, anchor);
        return;
      }
      const needTransition2 = moveType !== 2 && shapeFlag & 1 && transition;
      if (needTransition2) {
        if (moveType === 0) {
          if (transition.persisted && !el[leaveCbKey]) {
            hostInsert(el, container, anchor);
          } else {
            transition.beforeEnter(el);
            hostInsert(el, container, anchor);
            queuePostRenderEffect(() => transition.enter(el), parentSuspense);
          }
        } else {
          const { leave, delayLeave, afterLeave } = transition;
          const remove22 = () => {
            if (vnode.ctx.isUnmounted) {
              hostRemove(el);
            } else {
              hostInsert(el, container, anchor);
            }
          };
          const performLeave = () => {
            const wasLeaving = el._isLeaving || !!el[leaveCbKey];
            if (el._isLeaving) {
              el[leaveCbKey](
                true
                /* cancelled */
              );
            }
            if (transition.persisted && !wasLeaving) {
              remove22();
            } else {
              leave(el, () => {
                remove22();
                afterLeave && afterLeave();
              });
            }
          };
          if (delayLeave) {
            delayLeave(el, remove22, performLeave);
          } else {
            performLeave();
          }
        }
      } else {
        hostInsert(el, container, anchor);
      }
    };
    const unmount = (vnode, parentComponent, parentSuspense, doRemove = false, optimized = false) => {
      const {
        type,
        props,
        ref: ref3,
        children,
        dynamicChildren,
        shapeFlag,
        patchFlag,
        dirs,
        cacheIndex,
        memo
      } = vnode;
      if (patchFlag === -2) {
        optimized = false;
      }
      if (ref3 != null) {
        pauseTracking();
        setRef(ref3, null, parentSuspense, vnode, true);
        resetTracking();
      }
      if (cacheIndex != null) {
        parentComponent.renderCache[cacheIndex] = void 0;
      }
      if (shapeFlag & 256) {
        parentComponent.ctx.deactivate(vnode);
        return;
      }
      const shouldInvokeDirs = shapeFlag & 1 && dirs;
      const shouldInvokeVnodeHook = !isAsyncWrapper(vnode);
      let vnodeHook;
      if (shouldInvokeVnodeHook && (vnodeHook = props && props.onVnodeBeforeUnmount)) {
        invokeVNodeHook(vnodeHook, parentComponent, vnode);
      }
      if (shapeFlag & 6) {
        unmountComponent(vnode.component, parentSuspense, doRemove);
      } else {
        if (shapeFlag & 128) {
          vnode.suspense.unmount(parentSuspense, doRemove);
          return;
        }
        if (shouldInvokeDirs) {
          invokeDirectiveHook(vnode, null, parentComponent, "beforeUnmount");
        }
        if (shapeFlag & 64) {
          vnode.type.remove(
            vnode,
            parentComponent,
            parentSuspense,
            internals,
            doRemove
          );
        } else if (dynamicChildren && // #5154
        // when v-once is used inside a block, setBlockTracking(-1) marks the
        // parent block with hasOnce: true
        // so that it doesn't take the fast path during unmount - otherwise
        // components nested in v-once are never unmounted.
        !dynamicChildren.hasOnce && // #1153: fast path should not be taken for non-stable (v-for) fragments
        (type !== Fragment || patchFlag > 0 && patchFlag & 64)) {
          unmountChildren(
            dynamicChildren,
            parentComponent,
            parentSuspense,
            false,
            true
          );
        } else if (type === Fragment && patchFlag & (128 | 256) || !optimized && shapeFlag & 16) {
          unmountChildren(children, parentComponent, parentSuspense);
        }
        if (doRemove) {
          remove2(vnode);
        }
      }
      const shouldInvalidateMemo = memo != null && cacheIndex == null;
      if (shouldInvokeVnodeHook && (vnodeHook = props && props.onVnodeUnmounted) || shouldInvokeDirs || shouldInvalidateMemo) {
        queuePostRenderEffect(() => {
          vnodeHook && invokeVNodeHook(vnodeHook, parentComponent, vnode);
          shouldInvokeDirs && invokeDirectiveHook(vnode, null, parentComponent, "unmounted");
          if (shouldInvalidateMemo) {
            vnode.el = null;
          }
        }, parentSuspense);
      }
    };
    const remove2 = (vnode) => {
      const { type, el, anchor, transition } = vnode;
      if (type === Fragment) {
        {
          removeFragment(el, anchor);
        }
        return;
      }
      if (type === Static) {
        removeStaticNode(vnode);
        return;
      }
      const performRemove = () => {
        hostRemove(el);
        if (transition && !transition.persisted && transition.afterLeave) {
          transition.afterLeave();
        }
      };
      if (vnode.shapeFlag & 1 && transition && !transition.persisted) {
        const { leave, delayLeave } = transition;
        const performLeave = () => leave(el, performRemove);
        if (delayLeave) {
          delayLeave(vnode.el, performRemove, performLeave);
        } else {
          performLeave();
        }
      } else {
        performRemove();
      }
    };
    const removeFragment = (cur, end) => {
      let next;
      while (cur !== end) {
        next = hostNextSibling(cur);
        hostRemove(cur);
        cur = next;
      }
      hostRemove(end);
    };
    const unmountComponent = (instance, parentSuspense, doRemove) => {
      const { bum, scope, job, subTree, um, m, a } = instance;
      invalidateMount(m);
      invalidateMount(a);
      if (bum) {
        invokeArrayFns(bum);
      }
      scope.stop();
      if (job) {
        job.flags |= 8;
        unmount(subTree, instance, parentSuspense, doRemove);
      }
      if (um) {
        queuePostRenderEffect(um, parentSuspense);
      }
      queuePostRenderEffect(() => {
        instance.isUnmounted = true;
      }, parentSuspense);
    };
    const unmountChildren = (children, parentComponent, parentSuspense, doRemove = false, optimized = false, start = 0) => {
      for (let i = start; i < children.length; i++) {
        unmount(children[i], parentComponent, parentSuspense, doRemove, optimized);
      }
    };
    const getNextHostNode = (vnode) => {
      if (vnode.shapeFlag & 6) {
        return getNextHostNode(vnode.component.subTree);
      }
      if (vnode.shapeFlag & 128) {
        return vnode.suspense.next();
      }
      const el = hostNextSibling(vnode.anchor || vnode.el);
      const teleportEnd = el && el[TeleportEndKey];
      return teleportEnd ? hostNextSibling(teleportEnd) : el;
    };
    let isFlushing = false;
    const render2 = (vnode, container, namespace) => {
      let instance;
      if (vnode == null) {
        if (container._vnode) {
          unmount(container._vnode, null, null, true);
          instance = container._vnode.component;
        }
      } else {
        patch(
          container._vnode || null,
          vnode,
          container,
          null,
          null,
          null,
          namespace
        );
      }
      container._vnode = vnode;
      if (!isFlushing) {
        isFlushing = true;
        flushPreFlushCbs(instance);
        flushPostFlushCbs();
        isFlushing = false;
      }
    };
    const internals = {
      p: patch,
      um: unmount,
      m: move,
      r: remove2,
      mt: mountComponent,
      mc: mountChildren,
      pc: patchChildren,
      pbc: patchBlockChildren,
      n: getNextHostNode,
      o: options
    };
    let hydrate2;
    let hydrateNode;
    if (createHydrationFns) {
      [hydrate2, hydrateNode] = createHydrationFns(
        internals
      );
    }
    return {
      render: render2,
      hydrate: hydrate2,
      createApp: createAppAPI(render2, hydrate2)
    };
  }
  function resolveChildrenNamespace({ type, props }, currentNamespace) {
    return currentNamespace === "svg" && type === "foreignObject" || currentNamespace === "mathml" && type === "annotation-xml" && props && props.encoding && props.encoding.includes("html") ? void 0 : currentNamespace;
  }
  function toggleRecurse({ effect: effect2, job }, allowed) {
    if (allowed) {
      effect2.flags |= 32;
      job.flags |= 4;
    } else {
      effect2.flags &= -33;
      job.flags &= -5;
    }
  }
  function needTransition(parentSuspense, transition) {
    return (!parentSuspense || parentSuspense && !parentSuspense.pendingBranch) && transition && !transition.persisted;
  }
  function traverseStaticChildren(n1, n2, shallow = false) {
    const ch1 = n1.children;
    const ch2 = n2.children;
    if (isArray(ch1) && isArray(ch2)) {
      for (let i = 0; i < ch1.length; i++) {
        const c1 = ch1[i];
        let c2 = ch2[i];
        if (c2.shapeFlag & 1 && !c2.dynamicChildren) {
          if (c2.patchFlag <= 0 || c2.patchFlag === 32) {
            c2 = ch2[i] = cloneIfMounted(ch2[i]);
            c2.el = c1.el;
          }
          if (!shallow && c2.patchFlag !== -2)
            traverseStaticChildren(c1, c2);
        }
        if (c2.type === Text) {
          if (c2.patchFlag === -1) {
            c2 = ch2[i] = cloneIfMounted(c2);
          }
          c2.el = c1.el;
        }
        if (c2.type === Comment && !c2.el) {
          c2.el = c1.el;
        }
      }
    }
  }
  function getSequence(arr) {
    const p2 = arr.slice();
    const result = [0];
    let i, j, u, v, c;
    const len = arr.length;
    for (i = 0; i < len; i++) {
      const arrI = arr[i];
      if (arrI !== 0) {
        j = result[result.length - 1];
        if (arr[j] < arrI) {
          p2[i] = j;
          result.push(i);
          continue;
        }
        u = 0;
        v = result.length - 1;
        while (u < v) {
          c = u + v >> 1;
          if (arr[result[c]] < arrI) {
            u = c + 1;
          } else {
            v = c;
          }
        }
        if (arrI < arr[result[u]]) {
          if (u > 0) {
            p2[i] = result[u - 1];
          }
          result[u] = i;
        }
      }
    }
    u = result.length;
    v = result[u - 1];
    while (u-- > 0) {
      result[u] = v;
      v = p2[v];
    }
    return result;
  }
  function locateNonHydratedAsyncRoot(instance) {
    const subComponent = instance.subTree.component;
    if (subComponent) {
      if (subComponent.asyncDep && !subComponent.asyncResolved) {
        return subComponent;
      } else {
        return locateNonHydratedAsyncRoot(subComponent);
      }
    }
  }
  function invalidateMount(hooks) {
    if (hooks) {
      for (let i = 0; i < hooks.length; i++)
        hooks[i].flags |= 8;
    }
  }
  function resolveAsyncComponentPlaceholder(anchorVnode) {
    if (anchorVnode.placeholder) {
      return anchorVnode.placeholder;
    }
    const instance = anchorVnode.component;
    if (instance) {
      return resolveAsyncComponentPlaceholder(instance.subTree);
    }
    return null;
  }
  const isSuspense = (type) => type.__isSuspense;
  let suspenseId = 0;
  const SuspenseImpl = {
    name: "Suspense",
    // In order to make Suspense tree-shakable, we need to avoid importing it
    // directly in the renderer. The renderer checks for the __isSuspense flag
    // on a vnode's type and calls the `process` method, passing in renderer
    // internals.
    __isSuspense: true,
    process(n1, n2, container, anchor, parentComponent, parentSuspense, namespace, slotScopeIds, optimized, rendererInternals) {
      if (n1 == null) {
        mountSuspense(
          n2,
          container,
          anchor,
          parentComponent,
          parentSuspense,
          namespace,
          slotScopeIds,
          optimized,
          rendererInternals
        );
      } else {
        if (parentSuspense && parentSuspense.deps > 0 && !n1.suspense.isInFallback) {
          n2.suspense = n1.suspense;
          n2.suspense.vnode = n2;
          n2.el = n1.el;
          return;
        }
        patchSuspense(
          n1,
          n2,
          container,
          anchor,
          parentComponent,
          namespace,
          slotScopeIds,
          optimized,
          rendererInternals
        );
      }
    },
    hydrate: hydrateSuspense,
    normalize: normalizeSuspenseChildren
  };
  const Suspense = SuspenseImpl;
  function triggerEvent(vnode, name) {
    const eventListener = vnode.props && vnode.props[name];
    if (isFunction(eventListener)) {
      eventListener();
    }
  }
  function mountSuspense(vnode, container, anchor, parentComponent, parentSuspense, namespace, slotScopeIds, optimized, rendererInternals) {
    const {
      p: patch,
      o: { createElement }
    } = rendererInternals;
    const hiddenContainer = createElement("div");
    const suspense = vnode.suspense = createSuspenseBoundary(
      vnode,
      parentSuspense,
      parentComponent,
      container,
      hiddenContainer,
      anchor,
      namespace,
      slotScopeIds,
      optimized,
      rendererInternals
    );
    patch(
      null,
      suspense.pendingBranch = vnode.ssContent,
      hiddenContainer,
      null,
      parentComponent,
      suspense,
      namespace,
      slotScopeIds
    );
    if (suspense.deps > 0) {
      triggerEvent(vnode, "onPending");
      triggerEvent(vnode, "onFallback");
      patch(
        null,
        vnode.ssFallback,
        container,
        anchor,
        parentComponent,
        null,
        // fallback tree will not have suspense context
        namespace,
        slotScopeIds
      );
      setActiveBranch(suspense, vnode.ssFallback);
    } else {
      suspense.resolve(false, true);
    }
  }
  function patchSuspense(n1, n2, container, anchor, parentComponent, namespace, slotScopeIds, optimized, { p: patch, um: unmount, o: { createElement } }) {
    const suspense = n2.suspense = n1.suspense;
    suspense.vnode = n2;
    n2.el = n1.el;
    const newBranch = n2.ssContent;
    const newFallback = n2.ssFallback;
    const { activeBranch, pendingBranch, isInFallback, isHydrating } = suspense;
    if (pendingBranch) {
      suspense.pendingBranch = newBranch;
      if (isSameVNodeType(pendingBranch, newBranch)) {
        patch(
          pendingBranch,
          newBranch,
          suspense.hiddenContainer,
          null,
          parentComponent,
          suspense,
          namespace,
          slotScopeIds,
          optimized
        );
        if (suspense.deps <= 0) {
          suspense.resolve();
        } else if (isInFallback) {
          if (!isHydrating) {
            patch(
              activeBranch,
              newFallback,
              container,
              anchor,
              parentComponent,
              null,
              // fallback tree will not have suspense context
              namespace,
              slotScopeIds,
              optimized
            );
            setActiveBranch(suspense, newFallback);
          }
        }
      } else {
        suspense.pendingId = suspenseId++;
        if (isHydrating) {
          suspense.isHydrating = false;
          suspense.activeBranch = pendingBranch;
        } else {
          unmount(pendingBranch, parentComponent, suspense);
        }
        suspense.deps = 0;
        suspense.effects.length = 0;
        suspense.hiddenContainer = createElement("div");
        if (isInFallback) {
          patch(
            null,
            newBranch,
            suspense.hiddenContainer,
            null,
            parentComponent,
            suspense,
            namespace,
            slotScopeIds,
            optimized
          );
          if (suspense.deps <= 0) {
            suspense.resolve();
          } else {
            patch(
              activeBranch,
              newFallback,
              container,
              anchor,
              parentComponent,
              null,
              // fallback tree will not have suspense context
              namespace,
              slotScopeIds,
              optimized
            );
            setActiveBranch(suspense, newFallback);
          }
        } else if (activeBranch && isSameVNodeType(activeBranch, newBranch)) {
          patch(
            activeBranch,
            newBranch,
            container,
            anchor,
            parentComponent,
            suspense,
            namespace,
            slotScopeIds,
            optimized
          );
          suspense.resolve(true);
        } else {
          patch(
            null,
            newBranch,
            suspense.hiddenContainer,
            null,
            parentComponent,
            suspense,
            namespace,
            slotScopeIds,
            optimized
          );
          if (suspense.deps <= 0) {
            suspense.resolve();
          }
        }
      }
    } else {
      if (activeBranch && isSameVNodeType(activeBranch, newBranch)) {
        patch(
          activeBranch,
          newBranch,
          container,
          anchor,
          parentComponent,
          suspense,
          namespace,
          slotScopeIds,
          optimized
        );
        setActiveBranch(suspense, newBranch);
      } else {
        triggerEvent(n2, "onPending");
        suspense.pendingBranch = newBranch;
        if (newBranch.shapeFlag & 512) {
          suspense.pendingId = newBranch.component.suspenseId;
        } else {
          suspense.pendingId = suspenseId++;
        }
        patch(
          null,
          newBranch,
          suspense.hiddenContainer,
          null,
          parentComponent,
          suspense,
          namespace,
          slotScopeIds,
          optimized
        );
        if (suspense.deps <= 0) {
          suspense.resolve();
        } else {
          const { timeout, pendingId } = suspense;
          if (timeout > 0) {
            setTimeout(() => {
              if (suspense.pendingId === pendingId) {
                suspense.fallback(newFallback);
              }
            }, timeout);
          } else if (timeout === 0) {
            suspense.fallback(newFallback);
          }
        }
      }
    }
  }
  function createSuspenseBoundary(vnode, parentSuspense, parentComponent, container, hiddenContainer, anchor, namespace, slotScopeIds, optimized, rendererInternals, isHydrating = false) {
    const {
      p: patch,
      m: move,
      um: unmount,
      n: next,
      o: { parentNode, remove: remove2 }
    } = rendererInternals;
    let parentSuspenseId;
    const isSuspensible = isVNodeSuspensible(vnode);
    if (isSuspensible) {
      if (parentSuspense && parentSuspense.pendingBranch) {
        parentSuspenseId = parentSuspense.pendingId;
        parentSuspense.deps++;
      }
    }
    const timeout = vnode.props ? toNumber(vnode.props.timeout) : void 0;
    const initialAnchor = anchor;
    const suspense = {
      vnode,
      parent: parentSuspense,
      parentComponent,
      namespace,
      container,
      hiddenContainer,
      deps: 0,
      pendingId: suspenseId++,
      timeout: typeof timeout === "number" ? timeout : -1,
      activeBranch: null,
      isFallbackMountPending: false,
      pendingBranch: null,
      isInFallback: !isHydrating,
      isHydrating,
      isUnmounted: false,
      effects: [],
      resolve(resume = false, sync = false) {
        const {
          vnode: vnode2,
          activeBranch,
          pendingBranch,
          pendingId,
          effects,
          parentComponent: parentComponent2,
          container: container2,
          isInFallback
        } = suspense;
        let delayEnter = false;
        if (suspense.isHydrating) {
          suspense.isHydrating = false;
        } else if (!resume) {
          delayEnter = activeBranch && pendingBranch.transition && pendingBranch.transition.mode === "out-in";
          let hasUpdatedAnchor = false;
          if (delayEnter) {
            activeBranch.transition.afterLeave = () => {
              if (pendingId === suspense.pendingId) {
                move(
                  pendingBranch,
                  container2,
                  anchor === initialAnchor && !hasUpdatedAnchor ? next(activeBranch) : anchor,
                  0
                );
                queuePostFlushCb(effects);
                if (isInFallback && vnode2.ssFallback) {
                  vnode2.ssFallback.el = null;
                }
              }
            };
          }
          if (activeBranch && !suspense.isFallbackMountPending) {
            if (parentNode(activeBranch.el) === container2) {
              anchor = next(activeBranch);
              hasUpdatedAnchor = true;
            }
            unmount(activeBranch, parentComponent2, suspense, true);
            if (!delayEnter && isInFallback && vnode2.ssFallback) {
              queuePostRenderEffect(() => vnode2.ssFallback.el = null, suspense);
            }
          }
          if (!delayEnter) {
            move(pendingBranch, container2, anchor, 0);
          }
        }
        suspense.isFallbackMountPending = false;
        setActiveBranch(suspense, pendingBranch);
        suspense.pendingBranch = null;
        suspense.isInFallback = false;
        let parent = suspense.parent;
        let hasUnresolvedAncestor = false;
        while (parent) {
          if (parent.pendingBranch) {
            parent.effects.push(...effects);
            hasUnresolvedAncestor = true;
            break;
          }
          parent = parent.parent;
        }
        if (!hasUnresolvedAncestor && !delayEnter) {
          queuePostFlushCb(effects);
        }
        suspense.effects = [];
        if (isSuspensible) {
          if (parentSuspense && parentSuspense.pendingBranch && parentSuspenseId === parentSuspense.pendingId) {
            parentSuspense.deps--;
            if (parentSuspense.deps === 0 && !sync) {
              parentSuspense.resolve();
            }
          }
        }
        triggerEvent(vnode2, "onResolve");
      },
      fallback(fallbackVNode) {
        if (!suspense.pendingBranch) {
          return;
        }
        const { vnode: vnode2, activeBranch, parentComponent: parentComponent2, container: container2, namespace: namespace2 } = suspense;
        triggerEvent(vnode2, "onFallback");
        const anchor2 = next(activeBranch);
        const mountFallback = () => {
          suspense.isFallbackMountPending = false;
          if (!suspense.isInFallback) {
            return;
          }
          patch(
            null,
            fallbackVNode,
            container2,
            anchor2,
            parentComponent2,
            null,
            // fallback tree will not have suspense context
            namespace2,
            slotScopeIds,
            optimized
          );
          setActiveBranch(suspense, fallbackVNode);
        };
        const delayEnter = fallbackVNode.transition && fallbackVNode.transition.mode === "out-in";
        if (delayEnter) {
          suspense.isFallbackMountPending = true;
          activeBranch.transition.afterLeave = mountFallback;
        }
        suspense.isInFallback = true;
        unmount(
          activeBranch,
          parentComponent2,
          null,
          // no suspense so unmount hooks fire now
          true
          // shouldRemove
        );
        if (!delayEnter) {
          mountFallback();
        }
      },
      move(container2, anchor2, type) {
        suspense.activeBranch && move(suspense.activeBranch, container2, anchor2, type);
        suspense.container = container2;
      },
      next() {
        return suspense.activeBranch && next(suspense.activeBranch);
      },
      registerDep(instance, setupRenderEffect, optimized2) {
        const isInPendingSuspense = !!suspense.pendingBranch;
        if (isInPendingSuspense) {
          suspense.deps++;
        }
        const hydratedEl = instance.vnode.el;
        instance.asyncDep.catch((err) => {
          handleError(err, instance, 0);
        }).then((asyncSetupResult) => {
          if (instance.isUnmounted || suspense.isUnmounted || suspense.pendingId !== instance.suspenseId) {
            return;
          }
          unsetCurrentInstance();
          instance.asyncResolved = true;
          const { vnode: vnode2 } = instance;
          handleSetupResult(instance, asyncSetupResult, false);
          if (hydratedEl) {
            vnode2.el = hydratedEl;
          }
          const placeholder = !hydratedEl && instance.subTree.el;
          setupRenderEffect(
            instance,
            vnode2,
            // component may have been moved before resolve.
            // if this is not a hydration, instance.subTree will be the comment
            // placeholder.
            parentNode(hydratedEl || instance.subTree.el),
            // anchor will not be used if this is hydration, so only need to
            // consider the comment placeholder case.
            hydratedEl ? null : next(instance.subTree),
            suspense,
            namespace,
            optimized2
          );
          if (placeholder) {
            vnode2.placeholder = null;
            remove2(placeholder);
          }
          updateHOCHostEl(instance, vnode2.el);
          if (isInPendingSuspense && --suspense.deps === 0) {
            suspense.resolve();
          }
        });
      },
      unmount(parentSuspense2, doRemove) {
        suspense.isUnmounted = true;
        if (suspense.activeBranch) {
          unmount(
            suspense.activeBranch,
            parentComponent,
            parentSuspense2,
            doRemove
          );
        }
        if (suspense.pendingBranch) {
          unmount(
            suspense.pendingBranch,
            parentComponent,
            parentSuspense2,
            doRemove
          );
        }
      }
    };
    return suspense;
  }
  function hydrateSuspense(node, vnode, parentComponent, parentSuspense, namespace, slotScopeIds, optimized, rendererInternals, hydrateNode) {
    const suspense = vnode.suspense = createSuspenseBoundary(
      vnode,
      parentSuspense,
      parentComponent,
      node.parentNode,
      // eslint-disable-next-line no-restricted-globals
      document.createElement("div"),
      null,
      namespace,
      slotScopeIds,
      optimized,
      rendererInternals,
      true
    );
    const result = hydrateNode(
      node,
      suspense.pendingBranch = vnode.ssContent,
      parentComponent,
      suspense,
      slotScopeIds,
      optimized
    );
    if (suspense.deps === 0) {
      suspense.resolve(false, true);
    }
    return result;
  }
  function normalizeSuspenseChildren(vnode) {
    const { shapeFlag, children } = vnode;
    const isSlotChildren = shapeFlag & 32;
    vnode.ssContent = normalizeSuspenseSlot(
      isSlotChildren ? children.default : children
    );
    vnode.ssFallback = isSlotChildren ? normalizeSuspenseSlot(children.fallback) : createVNode(Comment);
  }
  function normalizeSuspenseSlot(s) {
    let block;
    if (isFunction(s)) {
      const trackBlock = isBlockTreeEnabled && s._c;
      if (trackBlock) {
        s._d = false;
        openBlock();
      }
      s = s();
      if (trackBlock) {
        s._d = true;
        block = currentBlock;
        closeBlock();
      }
    }
    if (isArray(s)) {
      const singleChild = filterSingleRoot(s);
      s = singleChild;
    }
    s = normalizeVNode(s);
    if (block && !s.dynamicChildren) {
      s.dynamicChildren = block.filter((c) => c !== s);
    }
    return s;
  }
  function queueEffectWithSuspense(fn, suspense) {
    if (suspense && suspense.pendingBranch) {
      if (isArray(fn)) {
        suspense.effects.push(...fn);
      } else {
        suspense.effects.push(fn);
      }
    } else {
      queuePostFlushCb(fn);
    }
  }
  function setActiveBranch(suspense, branch) {
    suspense.activeBranch = branch;
    const { vnode, parentComponent } = suspense;
    let el = branch.el;
    while (!el && branch.component) {
      branch = branch.component.subTree;
      el = branch.el;
    }
    vnode.el = el;
    if (parentComponent && parentComponent.subTree === vnode) {
      parentComponent.vnode.el = el;
      updateHOCHostEl(parentComponent, el);
    }
  }
  function isVNodeSuspensible(vnode) {
    const suspensible = vnode.props && vnode.props.suspensible;
    return suspensible != null && suspensible !== false;
  }
  const Fragment = /* @__PURE__ */ Symbol.for("v-fgt");
  const Text = /* @__PURE__ */ Symbol.for("v-txt");
  const Comment = /* @__PURE__ */ Symbol.for("v-cmt");
  const Static = /* @__PURE__ */ Symbol.for("v-stc");
  const blockStack = [];
  let currentBlock = null;
  function openBlock(disableTracking = false) {
    blockStack.push(currentBlock = disableTracking ? null : []);
  }
  function closeBlock() {
    blockStack.pop();
    currentBlock = blockStack[blockStack.length - 1] || null;
  }
  let isBlockTreeEnabled = 1;
  function setBlockTracking(value, inVOnce = false) {
    isBlockTreeEnabled += value;
    if (value < 0 && currentBlock && inVOnce) {
      currentBlock.hasOnce = true;
    }
  }
  function setupBlock(vnode) {
    vnode.dynamicChildren = isBlockTreeEnabled > 0 ? currentBlock || EMPTY_ARR : null;
    closeBlock();
    if (isBlockTreeEnabled > 0 && currentBlock) {
      currentBlock.push(vnode);
    }
    return vnode;
  }
  function createElementBlock(type, props, children, patchFlag, dynamicProps, shapeFlag) {
    return setupBlock(
      createBaseVNode(
        type,
        props,
        children,
        patchFlag,
        dynamicProps,
        shapeFlag,
        true
      )
    );
  }
  function createBlock(type, props, children, patchFlag, dynamicProps) {
    return setupBlock(
      createVNode(
        type,
        props,
        children,
        patchFlag,
        dynamicProps,
        true
      )
    );
  }
  function isVNode(value) {
    return value ? value.__v_isVNode === true : false;
  }
  function isSameVNodeType(n1, n2) {
    return n1.type === n2.type && n1.key === n2.key;
  }
  function transformVNodeArgs(transformer) {
  }
  const normalizeKey = ({ key }) => key != null ? key : null;
  const normalizeRef = ({
    ref: ref3,
    ref_key,
    ref_for
  }) => {
    if (typeof ref3 === "number") {
      ref3 = "" + ref3;
    }
    return ref3 != null ? isString(ref3) || /* @__PURE__ */ isRef(ref3) || isFunction(ref3) ? { i: currentRenderingInstance, r: ref3, k: ref_key, f: !!ref_for } : ref3 : null;
  };
  function createBaseVNode(type, props = null, children = null, patchFlag = 0, dynamicProps = null, shapeFlag = type === Fragment ? 0 : 1, isBlockNode = false, needFullChildrenNormalization = false) {
    const vnode = {
      __v_isVNode: true,
      __v_skip: true,
      type,
      props,
      key: props && normalizeKey(props),
      ref: props && normalizeRef(props),
      scopeId: currentScopeId,
      slotScopeIds: null,
      children,
      component: null,
      suspense: null,
      ssContent: null,
      ssFallback: null,
      dirs: null,
      transition: null,
      el: null,
      anchor: null,
      target: null,
      targetStart: null,
      targetAnchor: null,
      staticCount: 0,
      shapeFlag,
      patchFlag,
      dynamicProps,
      dynamicChildren: null,
      appContext: null,
      ctx: currentRenderingInstance
    };
    if (needFullChildrenNormalization) {
      normalizeChildren(vnode, children);
      if (shapeFlag & 128) {
        type.normalize(vnode);
      }
    } else if (children) {
      vnode.shapeFlag |= isString(children) ? 8 : 16;
    }
    if (isBlockTreeEnabled > 0 && // avoid a block node from tracking itself
    !isBlockNode && // has current parent block
    currentBlock && // presence of a patch flag indicates this node needs patching on updates.
    // component nodes also should always be patched, because even if the
    // component doesn't need to update, it needs to persist the instance on to
    // the next vnode so that it can be properly unmounted later.
    (vnode.patchFlag > 0 || shapeFlag & 6) && // the EVENTS flag is only for hydration and if it is the only flag, the
    // vnode should not be considered dynamic due to handler caching.
    vnode.patchFlag !== 32) {
      currentBlock.push(vnode);
    }
    return vnode;
  }
  const createVNode = _createVNode;
  function _createVNode(type, props = null, children = null, patchFlag = 0, dynamicProps = null, isBlockNode = false) {
    if (!type || type === NULL_DYNAMIC_COMPONENT) {
      type = Comment;
    }
    if (isVNode(type)) {
      const cloned = cloneVNode(
        type,
        props,
        true
        /* mergeRef: true */
      );
      if (children) {
        normalizeChildren(cloned, children);
      }
      if (isBlockTreeEnabled > 0 && !isBlockNode && currentBlock) {
        if (cloned.shapeFlag & 6) {
          currentBlock[currentBlock.indexOf(type)] = cloned;
        } else {
          currentBlock.push(cloned);
        }
      }
      cloned.patchFlag = -2;
      return cloned;
    }
    if (isClassComponent(type)) {
      type = type.__vccOpts;
    }
    if (props) {
      props = guardReactiveProps(props);
      let { class: klass, style } = props;
      if (klass && !isString(klass)) {
        props.class = normalizeClass(klass);
      }
      if (isObject(style)) {
        if (/* @__PURE__ */ isProxy(style) && !isArray(style)) {
          style = extend({}, style);
        }
        props.style = normalizeStyle(style);
      }
    }
    const shapeFlag = isString(type) ? 1 : isSuspense(type) ? 128 : isTeleport(type) ? 64 : isObject(type) ? 4 : isFunction(type) ? 2 : 0;
    return createBaseVNode(
      type,
      props,
      children,
      patchFlag,
      dynamicProps,
      shapeFlag,
      isBlockNode,
      true
    );
  }
  function guardReactiveProps(props) {
    if (!props) return null;
    return /* @__PURE__ */ isProxy(props) || isInternalObject(props) ? extend({}, props) : props;
  }
  function cloneVNode(vnode, extraProps, mergeRef = false, cloneTransition = false) {
    const { props, ref: ref3, patchFlag, children, transition } = vnode;
    const mergedProps = extraProps ? mergeProps(props || {}, extraProps) : props;
    const cloned = {
      __v_isVNode: true,
      __v_skip: true,
      type: vnode.type,
      props: mergedProps,
      key: mergedProps && normalizeKey(mergedProps),
      ref: extraProps && extraProps.ref ? (
        // #2078 in the case of <component :is="vnode" ref="extra"/>
        // if the vnode itself already has a ref, cloneVNode will need to merge
        // the refs so the single vnode can be set on multiple refs
        mergeRef && ref3 ? isArray(ref3) ? ref3.concat(normalizeRef(extraProps)) : [ref3, normalizeRef(extraProps)] : normalizeRef(extraProps)
      ) : ref3,
      scopeId: vnode.scopeId,
      slotScopeIds: vnode.slotScopeIds,
      children,
      target: vnode.target,
      targetStart: vnode.targetStart,
      targetAnchor: vnode.targetAnchor,
      staticCount: vnode.staticCount,
      shapeFlag: vnode.shapeFlag,
      // if the vnode is cloned with extra props, we can no longer assume its
      // existing patch flag to be reliable and need to add the FULL_PROPS flag.
      // note: preserve flag for fragments since they use the flag for children
      // fast paths only.
      patchFlag: extraProps && vnode.type !== Fragment ? patchFlag === -1 ? 16 : patchFlag | 16 : patchFlag,
      dynamicProps: vnode.dynamicProps,
      dynamicChildren: vnode.dynamicChildren,
      appContext: vnode.appContext,
      dirs: vnode.dirs,
      transition,
      // These should technically only be non-null on mounted VNodes. However,
      // they *should* be copied for kept-alive vnodes. So we just always copy
      // them since them being non-null during a mount doesn't affect the logic as
      // they will simply be overwritten.
      component: vnode.component,
      suspense: vnode.suspense,
      ssContent: vnode.ssContent && cloneVNode(vnode.ssContent),
      ssFallback: vnode.ssFallback && cloneVNode(vnode.ssFallback),
      placeholder: vnode.placeholder,
      el: vnode.el,
      anchor: vnode.anchor,
      ctx: vnode.ctx,
      ce: vnode.ce
    };
    if (transition && cloneTransition) {
      setTransitionHooks(
        cloned,
        transition.clone(cloned)
      );
    }
    return cloned;
  }
  function createTextVNode(text = " ", flag = 0) {
    return createVNode(Text, null, text, flag);
  }
  function createStaticVNode(content, numberOfNodes) {
    const vnode = createVNode(Static, null, content);
    vnode.staticCount = numberOfNodes;
    return vnode;
  }
  function createCommentVNode(text = "", asBlock = false) {
    return asBlock ? (openBlock(), createBlock(Comment, null, text)) : createVNode(Comment, null, text);
  }
  function normalizeVNode(child) {
    if (child == null || typeof child === "boolean") {
      return createVNode(Comment);
    } else if (isArray(child)) {
      return createVNode(
        Fragment,
        null,
        // #3666, avoid reference pollution when reusing vnode
        child.slice()
      );
    } else if (isVNode(child)) {
      return cloneIfMounted(child);
    } else {
      return createVNode(Text, null, String(child));
    }
  }
  function cloneIfMounted(child) {
    return child.el === null && child.patchFlag !== -1 || child.memo ? child : cloneVNode(child);
  }
  function normalizeChildren(vnode, children) {
    let type = 0;
    const { shapeFlag } = vnode;
    if (children == null) {
      children = null;
    } else if (isArray(children)) {
      type = 16;
    } else if (typeof children === "object") {
      if (shapeFlag & (1 | 64)) {
        const slot = children.default;
        if (slot) {
          slot._c && (slot._d = false);
          normalizeChildren(vnode, slot());
          slot._c && (slot._d = true);
        }
        return;
      } else {
        type = 32;
        const slotFlag = children._;
        if (!slotFlag && !isInternalObject(children)) {
          children._ctx = currentRenderingInstance;
        } else if (slotFlag === 3 && currentRenderingInstance) {
          if (currentRenderingInstance.slots._ === 1) {
            children._ = 1;
          } else {
            children._ = 2;
            vnode.patchFlag |= 1024;
          }
        }
      }
    } else if (isFunction(children)) {
      if (shapeFlag & (1 | 64)) {
        normalizeChildren(vnode, { default: children });
        return;
      }
      children = { default: children, _ctx: currentRenderingInstance };
      type = 32;
    } else {
      children = String(children);
      if (shapeFlag & 64) {
        type = 16;
        children = [createTextVNode(children)];
      } else {
        type = 8;
      }
    }
    vnode.children = children;
    vnode.shapeFlag |= type;
  }
  function mergeProps(...args) {
    const ret = {};
    for (let i = 0; i < args.length; i++) {
      const toMerge = args[i];
      for (const key in toMerge) {
        if (key === "class") {
          if (ret.class !== toMerge.class) {
            ret.class = normalizeClass([ret.class, toMerge.class]);
          }
        } else if (key === "style") {
          ret.style = normalizeStyle([ret.style, toMerge.style]);
        } else if (isOn(key)) {
          const existing = ret[key];
          const incoming = toMerge[key];
          if (incoming && existing !== incoming && !(isArray(existing) && existing.includes(incoming))) {
            ret[key] = existing ? [].concat(existing, incoming) : incoming;
          } else if (incoming == null && existing == null && // mergeProps({ 'onUpdate:modelValue': undefined }) should not retain
          // the model listener.
          !isModelListener(key)) {
            ret[key] = incoming;
          }
        } else if (key !== "") {
          ret[key] = toMerge[key];
        }
      }
    }
    return ret;
  }
  function invokeVNodeHook(hook, instance, vnode, prevVNode = null) {
    callWithAsyncErrorHandling(hook, instance, 7, [
      vnode,
      prevVNode
    ]);
  }
  const emptyAppContext = createAppContext();
  let uid = 0;
  function createComponentInstance(vnode, parent, suspense) {
    const type = vnode.type;
    const appContext = (parent ? parent.appContext : vnode.appContext) || emptyAppContext;
    const instance = {
      uid: uid++,
      vnode,
      type,
      parent,
      appContext,
      root: null,
      // to be immediately set
      next: null,
      subTree: null,
      // will be set synchronously right after creation
      effect: null,
      update: null,
      // will be set synchronously right after creation
      job: null,
      scope: new EffectScope(
        true
        /* detached */
      ),
      render: null,
      proxy: null,
      exposed: null,
      exposeProxy: null,
      withProxy: null,
      provides: parent ? parent.provides : Object.create(appContext.provides),
      ids: parent ? parent.ids : ["", 0, 0],
      accessCache: null,
      renderCache: [],
      // local resolved assets
      components: null,
      directives: null,
      // resolved props and emits options
      propsOptions: normalizePropsOptions(type, appContext),
      emitsOptions: normalizeEmitsOptions(type, appContext),
      // emit
      emit: null,
      // to be set immediately
      emitted: null,
      // props default value
      propsDefaults: EMPTY_OBJ,
      // inheritAttrs
      inheritAttrs: type.inheritAttrs,
      // state
      ctx: EMPTY_OBJ,
      data: EMPTY_OBJ,
      props: EMPTY_OBJ,
      attrs: EMPTY_OBJ,
      slots: EMPTY_OBJ,
      refs: EMPTY_OBJ,
      setupState: EMPTY_OBJ,
      setupContext: null,
      // suspense related
      suspense,
      suspenseId: suspense ? suspense.pendingId : 0,
      asyncDep: null,
      asyncResolved: false,
      // lifecycle hooks
      // not using enums here because it results in computed properties
      isMounted: false,
      isUnmounted: false,
      isDeactivated: false,
      bc: null,
      c: null,
      bm: null,
      m: null,
      bu: null,
      u: null,
      um: null,
      bum: null,
      da: null,
      a: null,
      rtg: null,
      rtc: null,
      ec: null,
      sp: null
    };
    {
      instance.ctx = { _: instance };
    }
    instance.root = parent ? parent.root : instance;
    instance.emit = emit.bind(null, instance);
    if (vnode.ce) {
      vnode.ce(instance);
    }
    return instance;
  }
  let currentInstance = null;
  const getCurrentInstance = () => currentInstance || currentRenderingInstance;
  let internalSetCurrentInstance;
  let setInSSRSetupState;
  {
    const g = getGlobalThis();
    const registerGlobalSetter = (key, setter) => {
      let setters;
      if (!(setters = g[key])) setters = g[key] = [];
      setters.push(setter);
      return (v) => {
        if (setters.length > 1) setters.forEach((set) => set(v));
        else setters[0](v);
      };
    };
    internalSetCurrentInstance = registerGlobalSetter(
      `__VUE_INSTANCE_SETTERS__`,
      (v) => currentInstance = v
    );
    setInSSRSetupState = registerGlobalSetter(
      `__VUE_SSR_SETTERS__`,
      (v) => isInSSRComponentSetup = v
    );
  }
  const setCurrentInstance = (instance) => {
    const prev = currentInstance;
    internalSetCurrentInstance(instance);
    instance.scope.on();
    return () => {
      instance.scope.off();
      internalSetCurrentInstance(prev);
    };
  };
  const unsetCurrentInstance = () => {
    currentInstance && currentInstance.scope.off();
    internalSetCurrentInstance(null);
  };
  function isStatefulComponent(instance) {
    return instance.vnode.shapeFlag & 4;
  }
  let isInSSRComponentSetup = false;
  function setupComponent(instance, isSSR = false, optimized = false) {
    isSSR && setInSSRSetupState(isSSR);
    const { props, children } = instance.vnode;
    const isStateful = isStatefulComponent(instance);
    initProps(instance, props, isStateful, isSSR);
    initSlots(instance, children, optimized || isSSR);
    const setupResult = isStateful ? setupStatefulComponent(instance, isSSR) : void 0;
    isSSR && setInSSRSetupState(false);
    return setupResult;
  }
  function setupStatefulComponent(instance, isSSR) {
    const Component = instance.type;
    instance.accessCache = /* @__PURE__ */ Object.create(null);
    instance.proxy = new Proxy(instance.ctx, PublicInstanceProxyHandlers);
    const { setup } = Component;
    if (setup) {
      pauseTracking();
      const setupContext = instance.setupContext = setup.length > 1 ? createSetupContext(instance) : null;
      const reset = setCurrentInstance(instance);
      const setupResult = callWithErrorHandling(
        setup,
        instance,
        0,
        [
          instance.props,
          setupContext
        ]
      );
      const isAsyncSetup = isPromise(setupResult);
      resetTracking();
      reset();
      if ((isAsyncSetup || instance.sp) && !isAsyncWrapper(instance)) {
        markAsyncBoundary(instance);
      }
      if (isAsyncSetup) {
        setupResult.then(unsetCurrentInstance, unsetCurrentInstance);
        if (isSSR) {
          return setupResult.then((resolvedResult) => {
            handleSetupResult(instance, resolvedResult, isSSR);
          }).catch((e) => {
            handleError(e, instance, 0);
          });
        } else {
          instance.asyncDep = setupResult;
        }
      } else {
        handleSetupResult(instance, setupResult, isSSR);
      }
    } else {
      finishComponentSetup(instance, isSSR);
    }
  }
  function handleSetupResult(instance, setupResult, isSSR) {
    if (isFunction(setupResult)) {
      if (instance.type.__ssrInlineRender) {
        instance.ssrRender = setupResult;
      } else {
        instance.render = setupResult;
      }
    } else if (isObject(setupResult)) {
      instance.setupState = proxyRefs(setupResult);
    } else ;
    finishComponentSetup(instance, isSSR);
  }
  let compile$1;
  let installWithProxy;
  function registerRuntimeCompiler(_compile) {
    compile$1 = _compile;
    installWithProxy = (i) => {
      if (i.render._rc) {
        i.withProxy = new Proxy(i.ctx, RuntimeCompiledPublicInstanceProxyHandlers);
      }
    };
  }
  const isRuntimeOnly = () => !compile$1;
  function finishComponentSetup(instance, isSSR, skipOptions) {
    const Component = instance.type;
    if (!instance.render) {
      if (!isSSR && compile$1 && !Component.render) {
        const template = Component.template || resolveMergedOptions(instance).template;
        if (template) {
          const { isCustomElement, compilerOptions } = instance.appContext.config;
          const { delimiters, compilerOptions: componentCompilerOptions } = Component;
          const finalCompilerOptions = extend(
            extend(
              {
                isCustomElement,
                delimiters
              },
              compilerOptions
            ),
            componentCompilerOptions
          );
          Component.render = compile$1(template, finalCompilerOptions);
        }
      }
      instance.render = Component.render || NOOP;
      if (installWithProxy) {
        installWithProxy(instance);
      }
    }
    {
      const reset = setCurrentInstance(instance);
      pauseTracking();
      try {
        applyOptions(instance);
      } finally {
        resetTracking();
        reset();
      }
    }
  }
  const attrsProxyHandlers = {
    get(target, key) {
      track(target, "get", "");
      return target[key];
    }
  };
  function createSetupContext(instance) {
    const expose = (exposed) => {
      instance.exposed = exposed || {};
    };
    {
      return {
        attrs: new Proxy(instance.attrs, attrsProxyHandlers),
        slots: instance.slots,
        emit: instance.emit,
        expose
      };
    }
  }
  function getComponentPublicInstance(instance) {
    if (instance.exposed) {
      return instance.exposeProxy || (instance.exposeProxy = new Proxy(proxyRefs(markRaw(instance.exposed)), {
        get(target, key) {
          if (key in target) {
            return target[key];
          } else if (key in publicPropertiesMap) {
            return publicPropertiesMap[key](instance);
          }
        },
        has(target, key) {
          return key in target || key in publicPropertiesMap;
        }
      }));
    } else {
      return instance.proxy;
    }
  }
  const classifyRE = /(?:^|[-_])\w/g;
  const classify = (str) => str.replace(classifyRE, (c) => c.toUpperCase()).replace(/[-_]/g, "");
  function getComponentName(Component, includeInferred = true) {
    return isFunction(Component) ? Component.displayName || Component.name : Component.name || includeInferred && Component.__name;
  }
  function formatComponentName(instance, Component, isRoot = false) {
    let name = getComponentName(Component);
    if (!name && Component.__file) {
      const match = Component.__file.match(/([^/\\]+)\.\w+$/);
      if (match) {
        name = match[1];
      }
    }
    if (!name && instance) {
      const inferFromRegistry = (registry) => {
        for (const key in registry) {
          if (registry[key] === Component) {
            return key;
          }
        }
      };
      name = inferFromRegistry(instance.components) || instance.parent && inferFromRegistry(
        instance.parent.type.components
      ) || inferFromRegistry(instance.appContext.components);
    }
    return name ? classify(name) : isRoot ? `App` : `Anonymous`;
  }
  function isClassComponent(value) {
    return isFunction(value) && "__vccOpts" in value;
  }
  const computed = (getterOrOptions, debugOptions) => {
    const c = /* @__PURE__ */ computed$1(getterOrOptions, debugOptions, isInSSRComponentSetup);
    return c;
  };
  function h(type, propsOrChildren, children) {
    try {
      setBlockTracking(-1);
      const l = arguments.length;
      if (l === 2) {
        if (isObject(propsOrChildren) && !isArray(propsOrChildren)) {
          if (isVNode(propsOrChildren)) {
            return createVNode(type, null, [propsOrChildren]);
          }
          return createVNode(type, propsOrChildren);
        } else {
          return createVNode(type, null, propsOrChildren);
        }
      } else {
        if (l > 3) {
          children = Array.prototype.slice.call(arguments, 2);
        } else if (l === 3 && isVNode(children)) {
          children = [children];
        }
        return createVNode(type, propsOrChildren, children);
      }
    } finally {
      setBlockTracking(1);
    }
  }
  function initCustomFormatter() {
    {
      return;
    }
  }
  function withMemo(memo, render2, cache, index) {
    const cached = cache[index];
    if (cached && isMemoSame(cached, memo)) {
      return cached;
    }
    const ret = render2();
    ret.memo = memo.slice();
    ret.cacheIndex = index;
    return cache[index] = ret;
  }
  function isMemoSame(cached, memo) {
    const prev = cached.memo;
    if (prev.length != memo.length) {
      return false;
    }
    for (let i = 0; i < prev.length; i++) {
      if (hasChanged(prev[i], memo[i])) {
        return false;
      }
    }
    if (isBlockTreeEnabled > 0 && currentBlock) {
      currentBlock.push(cached);
    }
    return true;
  }
  const version = "3.5.39";
  const warn = NOOP;
  const ErrorTypeStrings = ErrorTypeStrings$1;
  const devtools = devtools$1;
  const setDevtoolsHook = setDevtoolsHook$1;
  const _ssrUtils = {
    createComponentInstance,
    setupComponent,
    renderComponentRoot,
    setCurrentRenderingInstance,
    isVNode,
    normalizeVNode,
    getComponentPublicInstance,
    ensureValidVNode,
    pushWarningContext,
    popWarningContext
  };
  const ssrUtils = _ssrUtils;
  const resolveFilter = null;
  const compatUtils = null;
  const DeprecationTypes = null;
  /**
  * @vue/runtime-dom v3.5.39
  * (c) 2018-present Yuxi (Evan) You and Vue contributors
  * @license MIT
  **/
  let policy = void 0;
  const tt = typeof window !== "undefined" && window.trustedTypes;
  if (tt) {
    try {
      policy = /* @__PURE__ */ tt.createPolicy("vue", {
        createHTML: (val) => val
      });
    } catch (e) {
    }
  }
  const unsafeToTrustedHTML = policy ? (val) => policy.createHTML(val) : (val) => val;
  const svgNS = "http://www.w3.org/2000/svg";
  const mathmlNS = "http://www.w3.org/1998/Math/MathML";
  const doc = typeof document !== "undefined" ? document : null;
  const templateContainer = doc && /* @__PURE__ */ doc.createElement("template");
  const nodeOps = {
    insert: (child, parent, anchor) => {
      parent.insertBefore(child, anchor || null);
    },
    remove: (child) => {
      const parent = child.parentNode;
      if (parent) {
        parent.removeChild(child);
      }
    },
    createElement: (tag, namespace, is, props) => {
      const el = namespace === "svg" ? doc.createElementNS(svgNS, tag) : namespace === "mathml" ? doc.createElementNS(mathmlNS, tag) : is ? doc.createElement(tag, { is }) : doc.createElement(tag);
      if (tag === "select" && props && props.multiple != null) {
        el.setAttribute("multiple", props.multiple);
      }
      return el;
    },
    createText: (text) => doc.createTextNode(text),
    createComment: (text) => doc.createComment(text),
    setText: (node, text) => {
      node.nodeValue = text;
    },
    setElementText: (el, text) => {
      el.textContent = text;
    },
    parentNode: (node) => node.parentNode,
    nextSibling: (node) => node.nextSibling,
    querySelector: (selector) => doc.querySelector(selector),
    setScopeId(el, id) {
      el.setAttribute(id, "");
    },
    // __UNSAFE__
    // Reason: innerHTML.
    // Static content here can only come from compiled templates.
    // As long as the user only uses trusted templates, this is safe.
    insertStaticContent(content, parent, anchor, namespace, start, end) {
      const before = anchor ? anchor.previousSibling : parent.lastChild;
      if (start && (start === end || start.nextSibling)) {
        while (true) {
          parent.insertBefore(start.cloneNode(true), anchor);
          if (start === end || !(start = start.nextSibling)) break;
        }
      } else {
        templateContainer.innerHTML = unsafeToTrustedHTML(
          namespace === "svg" ? `<svg>${content}</svg>` : namespace === "mathml" ? `<math>${content}</math>` : content
        );
        const template = templateContainer.content;
        if (namespace === "svg" || namespace === "mathml") {
          const wrapper = template.firstChild;
          while (wrapper.firstChild) {
            template.appendChild(wrapper.firstChild);
          }
          template.removeChild(wrapper);
        }
        parent.insertBefore(template, anchor);
      }
      return [
        // first
        before ? before.nextSibling : parent.firstChild,
        // last
        anchor ? anchor.previousSibling : parent.lastChild
      ];
    }
  };
  const TRANSITION = "transition";
  const ANIMATION = "animation";
  const vtcKey = /* @__PURE__ */ Symbol("_vtc");
  const DOMTransitionPropsValidators = {
    name: String,
    type: String,
    css: {
      type: Boolean,
      default: true
    },
    duration: [String, Number, Object],
    enterFromClass: String,
    enterActiveClass: String,
    enterToClass: String,
    appearFromClass: String,
    appearActiveClass: String,
    appearToClass: String,
    leaveFromClass: String,
    leaveActiveClass: String,
    leaveToClass: String
  };
  const TransitionPropsValidators = /* @__PURE__ */ extend(
    {},
    BaseTransitionPropsValidators,
    DOMTransitionPropsValidators
  );
  const decorate$1 = (t) => {
    t.displayName = "Transition";
    t.props = TransitionPropsValidators;
    return t;
  };
  const Transition = /* @__PURE__ */ decorate$1(
    (props, { slots }) => h(BaseTransition, resolveTransitionProps(props), slots)
  );
  const callHook = (hook, args = []) => {
    if (isArray(hook)) {
      hook.forEach((h2) => h2(...args));
    } else if (hook) {
      hook(...args);
    }
  };
  const hasExplicitCallback = (hook) => {
    return hook ? isArray(hook) ? hook.some((h2) => h2.length > 1) : hook.length > 1 : false;
  };
  function resolveTransitionProps(rawProps) {
    const baseProps = {};
    for (const key in rawProps) {
      if (!(key in DOMTransitionPropsValidators)) {
        baseProps[key] = rawProps[key];
      }
    }
    if (rawProps.css === false) {
      return baseProps;
    }
    const {
      name = "v",
      type,
      duration,
      enterFromClass = `${name}-enter-from`,
      enterActiveClass = `${name}-enter-active`,
      enterToClass = `${name}-enter-to`,
      appearFromClass = enterFromClass,
      appearActiveClass = enterActiveClass,
      appearToClass = enterToClass,
      leaveFromClass = `${name}-leave-from`,
      leaveActiveClass = `${name}-leave-active`,
      leaveToClass = `${name}-leave-to`
    } = rawProps;
    const durations = normalizeDuration(duration);
    const enterDuration = durations && durations[0];
    const leaveDuration = durations && durations[1];
    const {
      onBeforeEnter,
      onEnter,
      onEnterCancelled,
      onLeave,
      onLeaveCancelled,
      onBeforeAppear = onBeforeEnter,
      onAppear = onEnter,
      onAppearCancelled = onEnterCancelled
    } = baseProps;
    const finishEnter = (el, isAppear, done, isCancelled) => {
      el._enterCancelled = isCancelled;
      removeTransitionClass(el, isAppear ? appearToClass : enterToClass);
      removeTransitionClass(el, isAppear ? appearActiveClass : enterActiveClass);
      done && done();
    };
    const finishLeave = (el, done) => {
      el._isLeaving = false;
      removeTransitionClass(el, leaveFromClass);
      removeTransitionClass(el, leaveToClass);
      removeTransitionClass(el, leaveActiveClass);
      done && done();
    };
    const makeEnterHook = (isAppear) => {
      return (el, done) => {
        const hook = isAppear ? onAppear : onEnter;
        const resolve2 = () => finishEnter(el, isAppear, done);
        callHook(hook, [el, resolve2]);
        nextFrame(() => {
          removeTransitionClass(el, isAppear ? appearFromClass : enterFromClass);
          addTransitionClass(el, isAppear ? appearToClass : enterToClass);
          if (!hasExplicitCallback(hook)) {
            whenTransitionEnds(el, type, enterDuration, resolve2);
          }
        });
      };
    };
    return extend(baseProps, {
      onBeforeEnter(el) {
        callHook(onBeforeEnter, [el]);
        addTransitionClass(el, enterFromClass);
        addTransitionClass(el, enterActiveClass);
      },
      onBeforeAppear(el) {
        callHook(onBeforeAppear, [el]);
        addTransitionClass(el, appearFromClass);
        addTransitionClass(el, appearActiveClass);
      },
      onEnter: makeEnterHook(false),
      onAppear: makeEnterHook(true),
      onLeave(el, done) {
        el._isLeaving = true;
        const resolve2 = () => finishLeave(el, done);
        addTransitionClass(el, leaveFromClass);
        if (!el._enterCancelled) {
          forceReflow(el);
          addTransitionClass(el, leaveActiveClass);
        } else {
          addTransitionClass(el, leaveActiveClass);
          forceReflow(el);
        }
        nextFrame(() => {
          if (!el._isLeaving) {
            return;
          }
          removeTransitionClass(el, leaveFromClass);
          addTransitionClass(el, leaveToClass);
          if (!hasExplicitCallback(onLeave)) {
            whenTransitionEnds(el, type, leaveDuration, resolve2);
          }
        });
        callHook(onLeave, [el, resolve2]);
      },
      onEnterCancelled(el) {
        finishEnter(el, false, void 0, true);
        callHook(onEnterCancelled, [el]);
      },
      onAppearCancelled(el) {
        finishEnter(el, true, void 0, true);
        callHook(onAppearCancelled, [el]);
      },
      onLeaveCancelled(el) {
        finishLeave(el);
        callHook(onLeaveCancelled, [el]);
      }
    });
  }
  function normalizeDuration(duration) {
    if (duration == null) {
      return null;
    } else if (isObject(duration)) {
      return [NumberOf(duration.enter), NumberOf(duration.leave)];
    } else {
      const n = NumberOf(duration);
      return [n, n];
    }
  }
  function NumberOf(val) {
    const res = toNumber(val);
    return res;
  }
  function addTransitionClass(el, cls) {
    cls.split(/\s+/).forEach((c) => c && el.classList.add(c));
    (el[vtcKey] || (el[vtcKey] = /* @__PURE__ */ new Set())).add(cls);
  }
  function removeTransitionClass(el, cls) {
    cls.split(/\s+/).forEach((c) => c && el.classList.remove(c));
    const _vtc = el[vtcKey];
    if (_vtc) {
      _vtc.delete(cls);
      if (!_vtc.size) {
        el[vtcKey] = void 0;
      }
    }
  }
  function nextFrame(cb) {
    requestAnimationFrame(() => {
      requestAnimationFrame(cb);
    });
  }
  let endId = 0;
  function whenTransitionEnds(el, expectedType, explicitTimeout, resolve2) {
    const id = el._endId = ++endId;
    const resolveIfNotStale = () => {
      if (id === el._endId) {
        resolve2();
      }
    };
    if (explicitTimeout != null) {
      return setTimeout(resolveIfNotStale, explicitTimeout);
    }
    const { type, timeout, propCount } = getTransitionInfo(el, expectedType);
    if (!type) {
      return resolve2();
    }
    const endEvent = type + "end";
    let ended = 0;
    const end = () => {
      el.removeEventListener(endEvent, onEnd);
      resolveIfNotStale();
    };
    const onEnd = (e) => {
      if (e.target === el && ++ended >= propCount) {
        end();
      }
    };
    setTimeout(() => {
      if (ended < propCount) {
        end();
      }
    }, timeout + 1);
    el.addEventListener(endEvent, onEnd);
  }
  function getTransitionInfo(el, expectedType) {
    const styles = window.getComputedStyle(el);
    const getStyleProperties = (key) => (styles[key] || "").split(", ");
    const transitionDelays = getStyleProperties(`${TRANSITION}Delay`);
    const transitionDurations = getStyleProperties(`${TRANSITION}Duration`);
    const transitionTimeout = getTimeout(transitionDelays, transitionDurations);
    const animationDelays = getStyleProperties(`${ANIMATION}Delay`);
    const animationDurations = getStyleProperties(`${ANIMATION}Duration`);
    const animationTimeout = getTimeout(animationDelays, animationDurations);
    let type = null;
    let timeout = 0;
    let propCount = 0;
    if (expectedType === TRANSITION) {
      if (transitionTimeout > 0) {
        type = TRANSITION;
        timeout = transitionTimeout;
        propCount = transitionDurations.length;
      }
    } else if (expectedType === ANIMATION) {
      if (animationTimeout > 0) {
        type = ANIMATION;
        timeout = animationTimeout;
        propCount = animationDurations.length;
      }
    } else {
      timeout = Math.max(transitionTimeout, animationTimeout);
      type = timeout > 0 ? transitionTimeout > animationTimeout ? TRANSITION : ANIMATION : null;
      propCount = type ? type === TRANSITION ? transitionDurations.length : animationDurations.length : 0;
    }
    const hasTransform = type === TRANSITION && /\b(?:transform|all)(?:,|$)/.test(
      getStyleProperties(`${TRANSITION}Property`).toString()
    );
    return {
      type,
      timeout,
      propCount,
      hasTransform
    };
  }
  function getTimeout(delays, durations) {
    while (delays.length < durations.length) {
      delays = delays.concat(delays);
    }
    return Math.max(...durations.map((d, i) => toMs(d) + toMs(delays[i])));
  }
  function toMs(s) {
    if (s === "auto") return 0;
    return Number(s.slice(0, -1).replace(",", ".")) * 1e3;
  }
  function forceReflow(el) {
    const targetDocument = el ? el.ownerDocument : document;
    return targetDocument.body.offsetHeight;
  }
  function patchClass(el, value, isSVG) {
    const transitionClasses = el[vtcKey];
    if (transitionClasses) {
      value = (value ? [value, ...transitionClasses] : [...transitionClasses]).join(" ");
    }
    if (value == null) {
      el.removeAttribute("class");
    } else if (isSVG) {
      el.setAttribute("class", value);
    } else {
      el.className = value;
    }
  }
  const vShowOriginalDisplay = /* @__PURE__ */ Symbol("_vod");
  const vShowHidden = /* @__PURE__ */ Symbol("_vsh");
  const vShow = {
    // used for prop mismatch check during hydration
    name: "show",
    beforeMount(el, { value }, { transition }) {
      el[vShowOriginalDisplay] = el.style.display === "none" ? "" : el.style.display;
      if (transition && value) {
        transition.beforeEnter(el);
      } else {
        setDisplay(el, value);
      }
    },
    mounted(el, { value }, { transition }) {
      if (transition && value) {
        transition.enter(el);
      }
    },
    updated(el, { value, oldValue }, { transition }) {
      if (!value === !oldValue) return;
      if (transition) {
        if (value) {
          transition.beforeEnter(el);
          setDisplay(el, true);
          transition.enter(el);
        } else {
          transition.leave(el, () => {
            setDisplay(el, false);
          });
        }
      } else {
        setDisplay(el, value);
      }
    },
    beforeUnmount(el, { value }) {
      setDisplay(el, value);
    }
  };
  function setDisplay(el, value) {
    el.style.display = value ? el[vShowOriginalDisplay] : "none";
    el[vShowHidden] = !value;
  }
  function initVShowForSSR() {
    vShow.getSSRProps = ({ value }) => {
      if (!value) {
        return { style: { display: "none" } };
      }
    };
  }
  const CSS_VAR_TEXT = /* @__PURE__ */ Symbol("");
  function useCssVars(getter) {
    const instance = getCurrentInstance();
    if (!instance) {
      return;
    }
    const updateTeleports = instance.ut = (vars = getter(instance.proxy)) => {
      Array.from(
        document.querySelectorAll(`[data-v-owner="${instance.uid}"]`)
      ).forEach((node) => setVarsOnNode(node, vars));
    };
    const setVars = () => {
      const vars = getter(instance.proxy);
      if (instance.ce) {
        setVarsOnNode(instance.ce, vars);
      } else {
        setVarsOnVNode(instance.subTree, vars);
      }
      updateTeleports(vars);
    };
    onBeforeUpdate(() => {
      queuePostFlushCb(setVars);
    });
    onMounted(() => {
      watch(setVars, NOOP, { flush: "post" });
      const ob = new MutationObserver(setVars);
      ob.observe(instance.subTree.el.parentNode, { childList: true });
      onUnmounted(() => ob.disconnect());
    });
  }
  function setVarsOnVNode(vnode, vars) {
    if (vnode.shapeFlag & 128) {
      const suspense = vnode.suspense;
      vnode = suspense.activeBranch;
      if (suspense.pendingBranch && !suspense.isHydrating) {
        suspense.effects.push(() => {
          setVarsOnVNode(suspense.activeBranch, vars);
        });
      }
    }
    while (vnode.component) {
      vnode = vnode.component.subTree;
    }
    if (vnode.shapeFlag & 1 && vnode.el) {
      setVarsOnNode(vnode.el, vars);
    } else if (vnode.type === Fragment) {
      vnode.children.forEach((c) => setVarsOnVNode(c, vars));
    } else if (vnode.type === Static) {
      let { el, anchor } = vnode;
      while (el) {
        setVarsOnNode(el, vars);
        if (el === anchor) break;
        el = el.nextSibling;
      }
    }
  }
  function setVarsOnNode(el, vars) {
    if (el.nodeType === 1) {
      const style = el.style;
      let cssText = "";
      for (const key in vars) {
        const value = normalizeCssVarValue(vars[key]);
        style.setProperty(`--${key}`, value);
        cssText += `--${key}: ${value};`;
      }
      style[CSS_VAR_TEXT] = cssText;
    }
  }
  const displayRE = /(?:^|;)\s*display\s*:/;
  function patchStyle(el, prev, next) {
    const style = el.style;
    const isCssString = isString(next);
    let hasControlledDisplay = false;
    if (next && !isCssString) {
      if (prev) {
        if (!isString(prev)) {
          for (const key in prev) {
            if (next[key] == null) {
              setStyle(style, key, "");
            }
          }
        } else {
          for (const prevStyle of prev.split(";")) {
            const key = prevStyle.slice(0, prevStyle.indexOf(":")).trim();
            if (next[key] == null) {
              setStyle(style, key, "");
            }
          }
        }
      }
      for (const key in next) {
        if (key === "display") {
          hasControlledDisplay = true;
        }
        const value = next[key];
        if (value != null) {
          if (!shouldPreserveTextareaResizeStyle(
            el,
            key,
            !isString(prev) && prev ? prev[key] : void 0,
            value
          )) {
            setStyle(style, key, value);
          }
        } else {
          setStyle(style, key, "");
        }
      }
    } else {
      if (isCssString) {
        if (prev !== next) {
          const cssVarText = style[CSS_VAR_TEXT];
          if (cssVarText) {
            next += ";" + cssVarText;
          }
          style.cssText = next;
          hasControlledDisplay = displayRE.test(next);
        }
      } else if (prev) {
        el.removeAttribute("style");
      }
    }
    if (vShowOriginalDisplay in el) {
      el[vShowOriginalDisplay] = hasControlledDisplay ? style.display : "";
      if (el[vShowHidden]) {
        style.display = "none";
      }
    }
  }
  const importantRE = /\s*!important$/;
  function setStyle(style, name, val) {
    if (isArray(val)) {
      val.forEach((v) => setStyle(style, name, v));
    } else {
      if (val == null) val = "";
      if (name.startsWith("--")) {
        style.setProperty(name, val);
      } else {
        const prefixed = autoPrefix(style, name);
        if (importantRE.test(val)) {
          style.setProperty(
            hyphenate(prefixed),
            val.replace(importantRE, ""),
            "important"
          );
        } else {
          style[prefixed] = val;
        }
      }
    }
  }
  const prefixes = ["Webkit", "Moz", "ms"];
  const prefixCache = {};
  function autoPrefix(style, rawName) {
    const cached = prefixCache[rawName];
    if (cached) {
      return cached;
    }
    let name = camelize(rawName);
    if (name !== "filter" && name in style) {
      return prefixCache[rawName] = name;
    }
    name = capitalize(name);
    for (let i = 0; i < prefixes.length; i++) {
      const prefixed = prefixes[i] + name;
      if (prefixed in style) {
        return prefixCache[rawName] = prefixed;
      }
    }
    return rawName;
  }
  function shouldPreserveTextareaResizeStyle(el, key, prev, next) {
    return el.tagName === "TEXTAREA" && (key === "width" || key === "height") && isString(next) && prev === next;
  }
  const xlinkNS = "http://www.w3.org/1999/xlink";
  function patchAttr(el, key, value, isSVG, instance, isBoolean = isSpecialBooleanAttr(key)) {
    if (isSVG && key.startsWith("xlink:")) {
      if (value == null) {
        el.removeAttributeNS(xlinkNS, key.slice(6, key.length));
      } else {
        el.setAttributeNS(xlinkNS, key, value);
      }
    } else {
      if (value == null || isBoolean && !includeBooleanAttr(value)) {
        el.removeAttribute(key);
      } else {
        el.setAttribute(
          key,
          isBoolean ? "" : isSymbol(value) ? String(value) : value
        );
      }
    }
  }
  function patchDOMProp(el, key, value, parentComponent, attrName) {
    if (key === "innerHTML" || key === "textContent") {
      if (value != null) {
        el[key] = key === "innerHTML" ? unsafeToTrustedHTML(value) : value;
      }
      return;
    }
    const tag = el.tagName;
    if (key === "value" && tag !== "PROGRESS" && // custom elements may use _value internally
    !tag.includes("-")) {
      const oldValue = tag === "OPTION" ? el.getAttribute("value") || "" : el.value;
      const newValue = value == null ? (
        // #11647: value should be set as empty string for null and undefined,
        // but <input type="checkbox"> should be set as 'on'.
        el.type === "checkbox" ? "on" : ""
      ) : String(value);
      if (oldValue !== newValue || !("_value" in el)) {
        el.value = newValue;
      }
      if (value == null) {
        el.removeAttribute(key);
      }
      el._value = value;
      return;
    }
    let needRemove = false;
    if (value === "" || value == null) {
      const type = typeof el[key];
      if (type === "boolean") {
        value = includeBooleanAttr(value);
      } else if (value == null && type === "string") {
        value = "";
        needRemove = true;
      } else if (type === "number") {
        value = 0;
        needRemove = true;
      }
    }
    try {
      el[key] = value;
    } catch (e) {
    }
    needRemove && el.removeAttribute(attrName || key);
  }
  function addEventListener(el, event, handler, options) {
    el.addEventListener(event, handler, options);
  }
  function removeEventListener(el, event, handler, options) {
    el.removeEventListener(event, handler, options);
  }
  const veiKey = /* @__PURE__ */ Symbol("_vei");
  function patchEvent(el, rawName, prevValue, nextValue, instance = null) {
    const invokers = el[veiKey] || (el[veiKey] = {});
    const existingInvoker = invokers[rawName];
    if (nextValue && existingInvoker) {
      existingInvoker.value = nextValue;
    } else {
      const [name, options] = parseName(rawName);
      if (nextValue) {
        const invoker = invokers[rawName] = createInvoker(
          nextValue,
          instance
        );
        addEventListener(el, name, invoker, options);
      } else if (existingInvoker) {
        removeEventListener(el, name, existingInvoker, options);
        invokers[rawName] = void 0;
      }
    }
  }
  const optionsModifierRE = /(Once|Passive|Capture)$/;
  const optionsModifierEventRE = /^on:?(?:Once|Passive|Capture)$/;
  function parseName(name) {
    let options;
    let m;
    while ((m = name.match(optionsModifierRE)) && !optionsModifierEventRE.test(name)) {
      if (!options) options = {};
      name = name.slice(0, name.length - m[1].length);
      options[m[1].toLowerCase()] = true;
    }
    const event = name[2] === ":" ? name.slice(3) : hyphenate(name.slice(2));
    return [event, options];
  }
  let cachedNow = 0;
  const p = /* @__PURE__ */ Promise.resolve();
  const getNow = () => cachedNow || (p.then(() => cachedNow = 0), cachedNow = Date.now());
  function createInvoker(initialValue, instance) {
    const invoker = (e) => {
      if (!e._vts) {
        e._vts = Date.now();
      } else if (e._vts <= invoker.attached) {
        return;
      }
      const value = invoker.value;
      if (isArray(value)) {
        const originalStop = e.stopImmediatePropagation;
        e.stopImmediatePropagation = () => {
          originalStop.call(e);
          e._stopped = true;
        };
        const handlers = value.slice();
        const args = [e];
        for (let i = 0; i < handlers.length; i++) {
          if (e._stopped) {
            break;
          }
          const handler = handlers[i];
          if (handler) {
            callWithAsyncErrorHandling(
              handler,
              instance,
              5,
              args
            );
          }
        }
      } else {
        callWithAsyncErrorHandling(
          value,
          instance,
          5,
          [e]
        );
      }
    };
    invoker.value = initialValue;
    invoker.attached = getNow();
    return invoker;
  }
  const isNativeOn = (key) => key.charCodeAt(0) === 111 && key.charCodeAt(1) === 110 && // lowercase letter
  key.charCodeAt(2) > 96 && key.charCodeAt(2) < 123;
  const patchProp = (el, key, prevValue, nextValue, namespace, parentComponent) => {
    const isSVG = namespace === "svg";
    if (key === "class") {
      patchClass(el, nextValue, isSVG);
    } else if (key === "style") {
      patchStyle(el, prevValue, nextValue);
    } else if (isOn(key)) {
      if (!isModelListener(key)) {
        patchEvent(el, key, prevValue, nextValue, parentComponent);
      }
    } else if (key[0] === "." ? (key = key.slice(1), true) : key[0] === "^" ? (key = key.slice(1), false) : shouldSetAsProp(el, key, nextValue, isSVG)) {
      patchDOMProp(el, key, nextValue);
      if (!el.tagName.includes("-") && (key === "value" || key === "checked" || key === "selected")) {
        patchAttr(el, key, nextValue, isSVG, parentComponent, key !== "value");
      }
    } else if (
      // #11081 force set props for possible async custom element
      el._isVueCE && // #12408 check if it's declared prop or it's async custom element
      (shouldSetAsPropForVueCE(el, key) || // @ts-expect-error _def is private
      el._def.__asyncLoader && (/[A-Z]/.test(key) || !isString(nextValue)))
    ) {
      patchDOMProp(el, camelize(key), nextValue, parentComponent, key);
    } else {
      if (key === "true-value") {
        el._trueValue = nextValue;
      } else if (key === "false-value") {
        el._falseValue = nextValue;
      }
      patchAttr(el, key, nextValue, isSVG);
    }
  };
  function shouldSetAsProp(el, key, value, isSVG) {
    if (isSVG) {
      if (key === "innerHTML" || key === "textContent") {
        return true;
      }
      if (key in el && isNativeOn(key) && isFunction(value)) {
        return true;
      }
      return false;
    }
    if (key === "spellcheck" || key === "draggable" || key === "translate" || key === "autocorrect") {
      return false;
    }
    if (key === "sandbox" && el.tagName === "IFRAME") {
      return false;
    }
    if (key === "form") {
      return false;
    }
    if (key === "list" && el.tagName === "INPUT") {
      return false;
    }
    if (key === "type" && el.tagName === "TEXTAREA") {
      return false;
    }
    if (key === "width" || key === "height") {
      const tag = el.tagName;
      if (tag === "IMG" || tag === "VIDEO" || tag === "CANVAS" || tag === "SOURCE") {
        return false;
      }
    }
    if (isNativeOn(key) && isString(value)) {
      return false;
    }
    return key in el;
  }
  function shouldSetAsPropForVueCE(el, key) {
    const props = (
      // @ts-expect-error _def is private
      el._def.props
    );
    if (!props) {
      return false;
    }
    const camelKey = camelize(key);
    return Array.isArray(props) ? props.some((prop) => camelize(prop) === camelKey) : Object.keys(props).some((prop) => camelize(prop) === camelKey);
  }
  const REMOVAL = {};
  // @__NO_SIDE_EFFECTS__
  function defineCustomElement(options, extraOptions, _createApp) {
    let Comp = /* @__PURE__ */ defineComponent(options, extraOptions);
    if (isPlainObject(Comp)) Comp = extend({}, Comp, extraOptions);
    class VueCustomElement extends VueElement {
      constructor(initialProps) {
        super(Comp, initialProps, _createApp);
      }
    }
    VueCustomElement.def = Comp;
    return VueCustomElement;
  }
  const defineSSRCustomElement = (/* @__NO_SIDE_EFFECTS__ */ (options, extraOptions) => {
    return /* @__PURE__ */ defineCustomElement(options, extraOptions, createSSRApp);
  });
  const BaseClass = typeof HTMLElement !== "undefined" ? HTMLElement : class {
  };
  class VueElement extends BaseClass {
    constructor(_def, _props = {}, _createApp = createApp) {
      super();
      this._def = _def;
      this._props = _props;
      this._createApp = _createApp;
      this._isVueCE = true;
      this._instance = null;
      this._app = null;
      this._nonce = this._def.nonce;
      this._connected = false;
      this._resolved = false;
      this._patching = false;
      this._dirty = false;
      this._numberProps = null;
      this._styleChildren = /* @__PURE__ */ new WeakSet();
      this._styleAnchors = /* @__PURE__ */ new WeakMap();
      this._ob = null;
      if (this.shadowRoot && _createApp !== createApp) {
        this._root = this.shadowRoot;
      } else {
        if (_def.shadowRoot !== false) {
          this.attachShadow(
            extend({}, _def.shadowRootOptions, {
              mode: "open"
            })
          );
          this._root = this.shadowRoot;
        } else {
          this._root = this;
        }
      }
    }
    connectedCallback() {
      if (!this.isConnected) return;
      if (!this.shadowRoot && !this._resolved) {
        this._parseSlots();
      }
      this._connected = true;
      let parent = this;
      while (parent = parent && // #12479 should check assignedSlot first to get correct parent
      (parent.assignedSlot || parent.parentNode || parent.host)) {
        if (parent instanceof VueElement) {
          this._parent = parent;
          break;
        }
      }
      if (!this._instance) {
        if (this._resolved) {
          this._mount(this._def);
        } else {
          if (parent && parent._pendingResolve) {
            this._pendingResolve = parent._pendingResolve.then(() => {
              this._pendingResolve = void 0;
              this._resolveDef();
            });
          } else {
            this._resolveDef();
          }
        }
      }
    }
    _setParent(parent = this._parent) {
      if (parent) {
        this._instance.parent = parent._instance;
        this._inheritParentContext(parent);
      }
    }
    _inheritParentContext(parent = this._parent) {
      if (parent && this._app) {
        Object.setPrototypeOf(
          this._app._context.provides,
          parent._instance.provides
        );
      }
    }
    disconnectedCallback() {
      this._connected = false;
      nextTick(() => {
        if (!this._connected) {
          if (this._ob) {
            this._ob.disconnect();
            this._ob = null;
          }
          this._app && this._app.unmount();
          if (this._instance) this._instance.ce = void 0;
          this._app = this._instance = null;
          if (this._teleportTargets) {
            this._teleportTargets.clear();
            this._teleportTargets = void 0;
          }
        }
      });
    }
    _processMutations(mutations) {
      for (const m of mutations) {
        this._setAttr(m.attributeName);
      }
    }
    /**
     * resolve inner component definition (handle possible async component)
     */
    _resolveDef() {
      if (this._pendingResolve) {
        return;
      }
      for (let i = 0; i < this.attributes.length; i++) {
        this._setAttr(this.attributes[i].name);
      }
      this._ob = new MutationObserver(this._processMutations.bind(this));
      this._ob.observe(this, { attributes: true });
      const resolve2 = (def2, isAsync = false) => {
        this._resolved = true;
        this._pendingResolve = void 0;
        const { props, styles } = def2;
        let numberProps;
        if (props && !isArray(props)) {
          for (const key in props) {
            const opt = props[key];
            if (opt === Number || opt && opt.type === Number) {
              if (key in this._props) {
                this._props[key] = toNumber(this._props[key]);
              }
              (numberProps || (numberProps = /* @__PURE__ */ Object.create(null)))[camelize(key)] = true;
            }
          }
        }
        this._numberProps = numberProps;
        this._resolveProps(def2);
        if (this.shadowRoot) {
          this._applyStyles(styles);
        }
        this._mount(def2);
      };
      const asyncDef = this._def.__asyncLoader;
      if (asyncDef) {
        this._pendingResolve = asyncDef().then((def2) => {
          def2.configureApp = this._def.configureApp;
          resolve2(this._def = def2, true);
        });
      } else {
        resolve2(this._def);
      }
    }
    _mount(def2) {
      this._app = this._createApp(def2);
      this._inheritParentContext();
      if (def2.configureApp) {
        def2.configureApp(this._app);
      }
      this._app._ceVNode = this._createVNode();
      this._app.mount(this._root);
      const exposed = this._instance && this._instance.exposed;
      if (!exposed) return;
      for (const key in exposed) {
        if (!hasOwn(this, key)) {
          Object.defineProperty(this, key, {
            // unwrap ref to be consistent with public instance behavior
            get: () => unref(exposed[key])
          });
        }
      }
    }
    _resolveProps(def2) {
      const { props } = def2;
      const declaredPropKeys = isArray(props) ? props : Object.keys(props || {});
      for (const key of Object.keys(this)) {
        if (key[0] !== "_" && declaredPropKeys.includes(key)) {
          this._setProp(key, this[key]);
        }
      }
      for (const key of declaredPropKeys.map(camelize)) {
        Object.defineProperty(this, key, {
          get() {
            return this._getProp(key);
          },
          set(val) {
            this._setProp(key, val, true, !this._patching);
          }
        });
      }
    }
    _setAttr(key) {
      if (key.startsWith("data-v-")) return;
      const has = this.hasAttribute(key);
      let value = has ? this.getAttribute(key) : REMOVAL;
      const camelKey = camelize(key);
      if (has && this._numberProps && this._numberProps[camelKey]) {
        value = toNumber(value);
      }
      this._setProp(camelKey, value, false, true);
    }
    /**
     * @internal
     */
    _getProp(key) {
      return this._props[key];
    }
    /**
     * @internal
     */
    _setProp(key, val, shouldReflect = true, shouldUpdate = false) {
      if (val !== this._props[key]) {
        this._dirty = true;
        if (val === REMOVAL) {
          delete this._props[key];
        } else {
          this._props[key] = val;
          if (key === "key" && this._app) {
            this._app._ceVNode.key = val;
          }
        }
        if (shouldUpdate && this._instance) {
          this._update();
        }
        if (shouldReflect) {
          const ob = this._ob;
          if (ob) {
            this._processMutations(ob.takeRecords());
            ob.disconnect();
          }
          if (val === true) {
            this.setAttribute(hyphenate(key), "");
          } else if (typeof val === "string" || typeof val === "number") {
            this.setAttribute(hyphenate(key), val + "");
          } else if (!val) {
            this.removeAttribute(hyphenate(key));
          }
          ob && ob.observe(this, { attributes: true });
        }
      }
    }
    _update() {
      const vnode = this._createVNode();
      if (this._app) vnode.appContext = this._app._context;
      render(vnode, this._root);
    }
    _createVNode() {
      const baseProps = {};
      if (!this.shadowRoot) {
        baseProps.onVnodeMounted = baseProps.onVnodeUpdated = this._renderSlots.bind(this);
      }
      const vnode = createVNode(this._def, extend(baseProps, this._props));
      if (!this._instance) {
        vnode.ce = (instance) => {
          this._instance = instance;
          instance.ce = this;
          instance.isCE = true;
          const dispatch = (event, args) => {
            this.dispatchEvent(
              new CustomEvent(
                event,
                isPlainObject(args[0]) ? extend({ detail: args }, args[0]) : { detail: args }
              )
            );
          };
          instance.emit = (event, ...args) => {
            dispatch(event, args);
            if (hyphenate(event) !== event) {
              dispatch(hyphenate(event), args);
            }
          };
          this._setParent();
        };
      }
      return vnode;
    }
    _applyStyles(styles, owner, parentComp) {
      if (!styles) return;
      if (owner) {
        if (owner === this._def || this._styleChildren.has(owner)) {
          return;
        }
        this._styleChildren.add(owner);
      }
      const nonce = this._nonce;
      const root = this.shadowRoot;
      const insertionAnchor = parentComp ? this._getStyleAnchor(parentComp) || this._getStyleAnchor(this._def) : this._getRootStyleInsertionAnchor(root);
      let last = null;
      for (let i = styles.length - 1; i >= 0; i--) {
        const s = document.createElement("style");
        if (nonce) s.setAttribute("nonce", nonce);
        s.textContent = styles[i];
        root.insertBefore(s, last || insertionAnchor);
        last = s;
        if (i === 0) {
          if (!parentComp) this._styleAnchors.set(this._def, s);
          if (owner) this._styleAnchors.set(owner, s);
        }
      }
    }
    _getStyleAnchor(comp) {
      if (!comp) {
        return null;
      }
      const anchor = this._styleAnchors.get(comp);
      if (anchor && anchor.parentNode === this.shadowRoot) {
        return anchor;
      }
      if (anchor) {
        this._styleAnchors.delete(comp);
      }
      return null;
    }
    _getRootStyleInsertionAnchor(root) {
      for (let i = 0; i < root.childNodes.length; i++) {
        const node = root.childNodes[i];
        if (!(node instanceof HTMLStyleElement)) {
          return node;
        }
      }
      return null;
    }
    /**
     * Only called when shadowRoot is false
     */
    _parseSlots() {
      const slots = this._slots = {};
      let n;
      while (n = this.firstChild) {
        const slotName = n.nodeType === 1 && n.getAttribute("slot") || "default";
        (slots[slotName] || (slots[slotName] = [])).push(n);
        this.removeChild(n);
      }
    }
    /**
     * Only called when shadowRoot is false
     */
    _renderSlots() {
      const outlets = this._getSlots();
      const scopeId = this._instance.type.__scopeId;
      for (let i = 0; i < outlets.length; i++) {
        const o = outlets[i];
        const slotName = o.getAttribute("name") || "default";
        const content = this._slots[slotName];
        const parent = o.parentNode;
        if (content) {
          for (const n of content) {
            if (scopeId && n.nodeType === 1) {
              const id = scopeId + "-s";
              const walker = document.createTreeWalker(n, 1);
              n.setAttribute(id, "");
              let child;
              while (child = walker.nextNode()) {
                child.setAttribute(id, "");
              }
            }
            parent.insertBefore(n, o);
          }
        } else {
          while (o.firstChild) parent.insertBefore(o.firstChild, o);
        }
        parent.removeChild(o);
      }
    }
    /**
     * @internal
     */
    _getSlots() {
      const roots = [this];
      if (this._teleportTargets) {
        roots.push(...this._teleportTargets);
      }
      const slots = /* @__PURE__ */ new Set();
      for (const root of roots) {
        const found = root.querySelectorAll("slot");
        for (let i = 0; i < found.length; i++) {
          slots.add(found[i]);
        }
      }
      return Array.from(slots);
    }
    /**
     * @internal
     */
    _injectChildStyle(comp, parentComp) {
      this._applyStyles(comp.styles, comp, parentComp);
    }
    /**
     * @internal
     */
    _beginPatch() {
      this._patching = true;
      this._dirty = false;
    }
    /**
     * @internal
     */
    _endPatch() {
      this._patching = false;
      if (this._dirty && this._instance) {
        this._update();
      }
    }
    /**
     * @internal
     */
    _hasShadowRoot() {
      return this._def.shadowRoot !== false;
    }
    /**
     * @internal
     */
    _removeChildStyle(comp) {
    }
  }
  function useHost(caller) {
    const instance = getCurrentInstance();
    const el = instance && instance.ce;
    if (el) {
      return el;
    }
    return null;
  }
  function useShadowRoot() {
    const el = useHost();
    return el && el.shadowRoot;
  }
  function useCssModule(name = "$style") {
    {
      const instance = getCurrentInstance();
      if (!instance) {
        return EMPTY_OBJ;
      }
      const modules = instance.type.__cssModules;
      if (!modules) {
        return EMPTY_OBJ;
      }
      const mod = modules[name];
      if (!mod) {
        return EMPTY_OBJ;
      }
      return mod;
    }
  }
  const positionMap = /* @__PURE__ */ new WeakMap();
  const newPositionMap = /* @__PURE__ */ new WeakMap();
  const moveCbKey = /* @__PURE__ */ Symbol("_moveCb");
  const enterCbKey = /* @__PURE__ */ Symbol("_enterCb");
  const decorate = (t) => {
    delete t.props.mode;
    return t;
  };
  const TransitionGroupImpl = /* @__PURE__ */ decorate({
    name: "TransitionGroup",
    props: /* @__PURE__ */ extend({}, TransitionPropsValidators, {
      tag: String,
      moveClass: String
    }),
    setup(props, { slots }) {
      const instance = getCurrentInstance();
      const state2 = useTransitionState();
      let prevChildren;
      let children;
      onUpdated(() => {
        if (!prevChildren.length) {
          return;
        }
        const moveClass = props.moveClass || `${props.name || "v"}-move`;
        if (!hasCSSTransform(
          prevChildren[0].el,
          instance.vnode.el,
          moveClass
        )) {
          prevChildren = [];
          return;
        }
        prevChildren.forEach(callPendingCbs);
        prevChildren.forEach(recordPosition);
        const movedChildren = prevChildren.filter(applyTranslation);
        forceReflow(instance.vnode.el);
        movedChildren.forEach((c) => {
          const el = c.el;
          const style = el.style;
          addTransitionClass(el, moveClass);
          style.transform = style.webkitTransform = style.transitionDuration = "";
          const cb = el[moveCbKey] = (e) => {
            if (e && e.target !== el) {
              return;
            }
            if (!e || e.propertyName.endsWith("transform")) {
              el.removeEventListener("transitionend", cb);
              el[moveCbKey] = null;
              removeTransitionClass(el, moveClass);
            }
          };
          el.addEventListener("transitionend", cb);
        });
        prevChildren = [];
      });
      return () => {
        const rawProps = /* @__PURE__ */ toRaw(props);
        const cssTransitionProps = resolveTransitionProps(rawProps);
        let tag = rawProps.tag || Fragment;
        prevChildren = [];
        if (children) {
          for (let i = 0; i < children.length; i++) {
            const child = children[i];
            if (child.el && child.el instanceof Element && // Hidden v-show nodes have no previous layout box to animate from.
            !child.el[vShowHidden]) {
              prevChildren.push(child);
              setTransitionHooks(
                child,
                resolveTransitionHooks(
                  child,
                  cssTransitionProps,
                  state2,
                  instance
                )
              );
              positionMap.set(child, getPosition(child.el));
            }
          }
        }
        children = slots.default ? getTransitionRawChildren(slots.default()) : [];
        for (let i = 0; i < children.length; i++) {
          const child = children[i];
          if (child.key != null) {
            setTransitionHooks(
              child,
              resolveTransitionHooks(child, cssTransitionProps, state2, instance)
            );
          }
        }
        return createVNode(tag, null, children);
      };
    }
  });
  const TransitionGroup = TransitionGroupImpl;
  function callPendingCbs(c) {
    const el = c.el;
    if (el[moveCbKey]) {
      el[moveCbKey]();
    }
    if (el[enterCbKey]) {
      el[enterCbKey]();
    }
  }
  function recordPosition(c) {
    newPositionMap.set(c, getPosition(c.el));
  }
  function applyTranslation(c) {
    const oldPos = positionMap.get(c);
    const newPos = newPositionMap.get(c);
    const dx = oldPos.left - newPos.left;
    const dy = oldPos.top - newPos.top;
    if (dx || dy) {
      const el = c.el;
      const s = el.style;
      const rect = el.getBoundingClientRect();
      let scaleX = 1;
      let scaleY = 1;
      if (el.offsetWidth) scaleX = rect.width / el.offsetWidth;
      if (el.offsetHeight) scaleY = rect.height / el.offsetHeight;
      if (!Number.isFinite(scaleX) || scaleX === 0) scaleX = 1;
      if (!Number.isFinite(scaleY) || scaleY === 0) scaleY = 1;
      if (Math.abs(scaleX - 1) < 0.01) scaleX = 1;
      if (Math.abs(scaleY - 1) < 0.01) scaleY = 1;
      s.transform = s.webkitTransform = `translate(${dx / scaleX}px,${dy / scaleY}px)`;
      s.transitionDuration = "0s";
      return c;
    }
  }
  function getPosition(el) {
    const rect = el.getBoundingClientRect();
    return {
      left: rect.left,
      top: rect.top
    };
  }
  function hasCSSTransform(el, root, moveClass) {
    const clone = el.cloneNode();
    const _vtc = el[vtcKey];
    if (_vtc) {
      _vtc.forEach((cls) => {
        cls.split(/\s+/).forEach((c) => c && clone.classList.remove(c));
      });
    }
    moveClass.split(/\s+/).forEach((c) => c && clone.classList.add(c));
    clone.style.display = "none";
    const container = root.nodeType === 1 ? root : root.parentNode;
    container.appendChild(clone);
    const { hasTransform } = getTransitionInfo(clone);
    container.removeChild(clone);
    return hasTransform;
  }
  const getModelAssigner = (vnode) => {
    const fn = vnode.props["onUpdate:modelValue"] || false;
    return isArray(fn) ? (value) => invokeArrayFns(fn, value) : fn;
  };
  function onCompositionStart(e) {
    e.target.composing = true;
  }
  function onCompositionEnd(e) {
    const target = e.target;
    if (target.composing) {
      target.composing = false;
      target.dispatchEvent(new Event("input"));
    }
  }
  const assignKey = /* @__PURE__ */ Symbol("_assign");
  function castValue(value, trim, number) {
    if (trim) value = value.trim();
    if (number) value = looseToNumber(value);
    return value;
  }
  const vModelText = {
    created(el, { modifiers: { lazy, trim, number } }, vnode) {
      el[assignKey] = getModelAssigner(vnode);
      const castToNumber = number || vnode.props && vnode.props.type === "number";
      addEventListener(el, lazy ? "change" : "input", (e) => {
        if (e.target.composing) return;
        el[assignKey](castValue(el.value, trim, castToNumber));
      });
      if (trim || castToNumber) {
        addEventListener(el, "change", () => {
          el.value = castValue(el.value, trim, castToNumber);
        });
      }
      if (!lazy) {
        addEventListener(el, "compositionstart", onCompositionStart);
        addEventListener(el, "compositionend", onCompositionEnd);
        addEventListener(el, "change", onCompositionEnd);
      }
    },
    // set value on mounted so it's after min/max for type="range"
    mounted(el, { value }) {
      el.value = value == null ? "" : value;
    },
    beforeUpdate(el, { value, oldValue, modifiers: { lazy, trim, number } }, vnode) {
      el[assignKey] = getModelAssigner(vnode);
      if (el.composing) return;
      const elValue = (number || el.type === "number") && !/^0\d/.test(el.value) ? looseToNumber(el.value) : el.value;
      const newValue = value == null ? "" : value;
      if (elValue === newValue) {
        return;
      }
      const rootNode = el.getRootNode();
      if ((rootNode instanceof Document || rootNode instanceof ShadowRoot) && rootNode.activeElement === el && el.type !== "range") {
        if (lazy && value === oldValue) {
          return;
        }
        if (trim && el.value.trim() === newValue) {
          return;
        }
      }
      el.value = newValue;
    }
  };
  const vModelCheckbox = {
    // #4096 array checkboxes need to be deep traversed
    deep: true,
    created(el, _, vnode) {
      el[assignKey] = getModelAssigner(vnode);
      addEventListener(el, "change", () => {
        const modelValue = el._modelValue;
        const elementValue = getValue(el);
        const checked = el.checked;
        const assign = el[assignKey];
        if (isArray(modelValue)) {
          const index = looseIndexOf(modelValue, elementValue);
          const found = index !== -1;
          if (checked && !found) {
            assign(modelValue.concat(elementValue));
          } else if (!checked && found) {
            const filtered = [...modelValue];
            filtered.splice(index, 1);
            assign(filtered);
          }
        } else if (isSet(modelValue)) {
          const cloned = new Set(modelValue);
          if (checked) {
            cloned.add(elementValue);
          } else {
            cloned.delete(elementValue);
          }
          assign(cloned);
        } else {
          assign(getCheckboxValue(el, checked));
        }
      });
    },
    // set initial checked on mount to wait for true-value/false-value
    mounted: setChecked,
    beforeUpdate(el, binding, vnode) {
      el[assignKey] = getModelAssigner(vnode);
      setChecked(el, binding, vnode);
    }
  };
  function setChecked(el, { value, oldValue }, vnode) {
    el._modelValue = value;
    let checked;
    if (isArray(value)) {
      checked = looseIndexOf(value, vnode.props.value) > -1;
    } else if (isSet(value)) {
      checked = value.has(vnode.props.value);
    } else {
      if (value === oldValue) return;
      checked = looseEqual(value, getCheckboxValue(el, true));
    }
    if (el.checked !== checked) {
      el.checked = checked;
    }
  }
  const vModelRadio = {
    created(el, { value }, vnode) {
      el.checked = looseEqual(value, vnode.props.value);
      el[assignKey] = getModelAssigner(vnode);
      addEventListener(el, "change", () => {
        el[assignKey](getValue(el));
      });
    },
    beforeUpdate(el, { value, oldValue }, vnode) {
      el[assignKey] = getModelAssigner(vnode);
      if (value !== oldValue) {
        el.checked = looseEqual(value, vnode.props.value);
      }
    }
  };
  const vModelSelect = {
    // <select multiple> value need to be deep traversed
    deep: true,
    created(el, { value, modifiers: { number } }, vnode) {
      const isSetModel = isSet(value);
      addEventListener(el, "change", () => {
        const selectedVal = Array.prototype.filter.call(el.options, (o) => o.selected).map(
          (o) => number ? looseToNumber(getValue(o)) : getValue(o)
        );
        el[assignKey](
          el.multiple ? isSetModel ? new Set(selectedVal) : selectedVal : selectedVal[0]
        );
        el._assigning = true;
        nextTick(() => {
          el._assigning = false;
        });
      });
      el[assignKey] = getModelAssigner(vnode);
    },
    // set value in mounted & updated because <select> relies on its children
    // <option>s.
    mounted(el, { value }) {
      setSelected(el, value);
    },
    beforeUpdate(el, _binding, vnode) {
      el[assignKey] = getModelAssigner(vnode);
    },
    updated(el, { value }) {
      if (!el._assigning) {
        setSelected(el, value);
      }
    }
  };
  function setSelected(el, value) {
    const isMultiple = el.multiple;
    const isArrayValue = isArray(value);
    if (isMultiple && !isArrayValue && !isSet(value)) {
      return;
    }
    for (let i = 0, l = el.options.length; i < l; i++) {
      const option = el.options[i];
      const optionValue = getValue(option);
      if (isMultiple) {
        if (isArrayValue) {
          const optionType = typeof optionValue;
          if (optionType === "string" || optionType === "number") {
            option.selected = value.some((v) => String(v) === String(optionValue));
          } else {
            option.selected = looseIndexOf(value, optionValue) > -1;
          }
        } else {
          option.selected = value.has(optionValue);
        }
      } else if (looseEqual(getValue(option), value)) {
        if (el.selectedIndex !== i) el.selectedIndex = i;
        return;
      }
    }
    if (!isMultiple && el.selectedIndex !== -1) {
      el.selectedIndex = -1;
    }
  }
  function getValue(el) {
    return "_value" in el ? el._value : el.value;
  }
  function getCheckboxValue(el, checked) {
    const key = checked ? "_trueValue" : "_falseValue";
    return key in el ? el[key] : checked;
  }
  const vModelDynamic = {
    created(el, binding, vnode) {
      callModelHook(el, binding, vnode, null, "created");
    },
    mounted(el, binding, vnode) {
      callModelHook(el, binding, vnode, null, "mounted");
    },
    beforeUpdate(el, binding, vnode, prevVNode) {
      callModelHook(el, binding, vnode, prevVNode, "beforeUpdate");
    },
    updated(el, binding, vnode, prevVNode) {
      callModelHook(el, binding, vnode, prevVNode, "updated");
    }
  };
  function resolveDynamicModel(tagName, type) {
    switch (tagName) {
      case "SELECT":
        return vModelSelect;
      case "TEXTAREA":
        return vModelText;
      default:
        switch (type) {
          case "checkbox":
            return vModelCheckbox;
          case "radio":
            return vModelRadio;
          default:
            return vModelText;
        }
    }
  }
  function callModelHook(el, binding, vnode, prevVNode, hook) {
    const modelToUse = resolveDynamicModel(
      el.tagName,
      vnode.props && vnode.props.type
    );
    const fn = modelToUse[hook];
    fn && fn(el, binding, vnode, prevVNode);
  }
  function initVModelForSSR() {
    vModelText.getSSRProps = ({ value }) => ({ value });
    vModelRadio.getSSRProps = ({ value }, vnode) => {
      if (vnode.props && looseEqual(vnode.props.value, value)) {
        return { checked: true };
      }
    };
    vModelCheckbox.getSSRProps = ({ value }, vnode) => {
      if (isArray(value)) {
        if (vnode.props && looseIndexOf(value, vnode.props.value) > -1) {
          return { checked: true };
        }
      } else if (isSet(value)) {
        if (vnode.props && value.has(vnode.props.value)) {
          return { checked: true };
        }
      } else if (value) {
        return { checked: true };
      }
    };
    vModelDynamic.getSSRProps = (binding, vnode) => {
      if (typeof vnode.type !== "string") {
        return;
      }
      const modelToUse = resolveDynamicModel(
        // resolveDynamicModel expects an uppercase tag name, but vnode.type is lowercase
        vnode.type.toUpperCase(),
        vnode.props && vnode.props.type
      );
      if (modelToUse.getSSRProps) {
        return modelToUse.getSSRProps(binding, vnode);
      }
    };
  }
  const systemModifiers = ["ctrl", "shift", "alt", "meta"];
  const modifierGuards = {
    stop: (e) => e.stopPropagation(),
    prevent: (e) => e.preventDefault(),
    self: (e) => e.target !== e.currentTarget,
    ctrl: (e) => !e.ctrlKey,
    shift: (e) => !e.shiftKey,
    alt: (e) => !e.altKey,
    meta: (e) => !e.metaKey,
    left: (e) => "button" in e && e.button !== 0,
    middle: (e) => "button" in e && e.button !== 1,
    right: (e) => "button" in e && e.button !== 2,
    exact: (e, modifiers) => systemModifiers.some((m) => e[`${m}Key`] && !modifiers.includes(m))
  };
  const withModifiers = (fn, modifiers) => {
    if (!fn) return fn;
    const cache = fn._withMods || (fn._withMods = {});
    const cacheKey = modifiers.join(".");
    return cache[cacheKey] || (cache[cacheKey] = ((event, ...args) => {
      for (let i = 0; i < modifiers.length; i++) {
        const guard = modifierGuards[modifiers[i]];
        if (guard && guard(event, modifiers)) return;
      }
      return fn(event, ...args);
    }));
  };
  const keyNames = {
    esc: "escape",
    space: " ",
    up: "arrow-up",
    left: "arrow-left",
    right: "arrow-right",
    down: "arrow-down",
    delete: "backspace"
  };
  const withKeys = (fn, modifiers) => {
    const cache = fn._withKeys || (fn._withKeys = {});
    const cacheKey = modifiers.join(".");
    return cache[cacheKey] || (cache[cacheKey] = ((event) => {
      if (!("key" in event)) {
        return;
      }
      const eventKey = hyphenate(event.key);
      if (modifiers.some(
        (k) => k === eventKey || keyNames[k] === eventKey
      )) {
        return fn(event);
      }
    }));
  };
  const rendererOptions = /* @__PURE__ */ extend({ patchProp }, nodeOps);
  let renderer;
  let enabledHydration = false;
  function ensureRenderer() {
    return renderer || (renderer = createRenderer(rendererOptions));
  }
  function ensureHydrationRenderer() {
    renderer = enabledHydration ? renderer : createHydrationRenderer(rendererOptions);
    enabledHydration = true;
    return renderer;
  }
  const render = ((...args) => {
    ensureRenderer().render(...args);
  });
  const hydrate = ((...args) => {
    ensureHydrationRenderer().hydrate(...args);
  });
  const createApp = ((...args) => {
    const app = ensureRenderer().createApp(...args);
    const { mount } = app;
    app.mount = (containerOrSelector) => {
      const container = normalizeContainer(containerOrSelector);
      if (!container) return;
      const component = app._component;
      if (!isFunction(component) && !component.render && !component.template) {
        component.template = container.innerHTML;
      }
      if (container.nodeType === 1) {
        container.textContent = "";
      }
      const proxy = mount(container, false, resolveRootNamespace(container));
      if (container instanceof Element) {
        container.removeAttribute("v-cloak");
        container.setAttribute("data-v-app", "");
      }
      return proxy;
    };
    return app;
  });
  const createSSRApp = ((...args) => {
    const app = ensureHydrationRenderer().createApp(...args);
    const { mount } = app;
    app.mount = (containerOrSelector) => {
      const container = normalizeContainer(containerOrSelector);
      if (container) {
        return mount(container, true, resolveRootNamespace(container));
      }
    };
    return app;
  });
  function resolveRootNamespace(container) {
    if (container instanceof SVGElement) {
      return "svg";
    }
    if (typeof MathMLElement === "function" && container instanceof MathMLElement) {
      return "mathml";
    }
  }
  function normalizeContainer(container) {
    if (isString(container)) {
      const res = document.querySelector(container);
      return res;
    }
    return container;
  }
  let ssrDirectiveInitialized = false;
  const initDirectivesForSSR = () => {
    if (!ssrDirectiveInitialized) {
      ssrDirectiveInitialized = true;
      initVModelForSSR();
      initVShowForSSR();
    }
  };
  /**
  * vue v3.5.39
  * (c) 2018-present Yuxi (Evan) You and Vue contributors
  * @license MIT
  **/
  const compile = () => {
  };
  const Vue = /* @__PURE__ */ Object.freeze(/* @__PURE__ */ Object.defineProperty({
    __proto__: null,
    BaseTransition,
    BaseTransitionPropsValidators,
    Comment,
    DeprecationTypes,
    EffectScope,
    ErrorCodes,
    ErrorTypeStrings,
    Fragment,
    KeepAlive,
    ReactiveEffect,
    Static,
    Suspense,
    Teleport,
    Text,
    TrackOpTypes,
    Transition,
    TransitionGroup,
    TriggerOpTypes,
    VueElement,
    assertNumber,
    callWithAsyncErrorHandling,
    callWithErrorHandling,
    camelize,
    capitalize,
    cloneVNode,
    compatUtils,
    compile,
    computed,
    createApp,
    createBlock,
    createCommentVNode,
    createElementBlock,
    createElementVNode: createBaseVNode,
    createHydrationRenderer,
    createPropsRestProxy,
    createRenderer,
    createSSRApp,
    createSlots,
    createStaticVNode,
    createTextVNode,
    createVNode,
    customRef,
    defineAsyncComponent,
    defineComponent,
    defineCustomElement,
    defineEmits,
    defineExpose,
    defineModel,
    defineOptions,
    defineProps,
    defineSSRCustomElement,
    defineSlots,
    devtools,
    effect,
    effectScope,
    getCurrentInstance,
    getCurrentScope,
    getCurrentWatcher,
    getTransitionRawChildren,
    guardReactiveProps,
    h,
    handleError,
    hasInjectionContext,
    hydrate,
    hydrateOnIdle,
    hydrateOnInteraction,
    hydrateOnMediaQuery,
    hydrateOnVisible,
    initCustomFormatter,
    initDirectivesForSSR,
    inject,
    isMemoSame,
    isProxy,
    isReactive,
    isReadonly,
    isRef,
    isRuntimeOnly,
    isShallow,
    isVNode,
    markRaw,
    mergeDefaults,
    mergeModels,
    mergeProps,
    nextTick,
    nodeOps,
    normalizeClass,
    normalizeProps,
    normalizeStyle,
    onActivated,
    onBeforeMount,
    onBeforeUnmount,
    onBeforeUpdate,
    onDeactivated,
    onErrorCaptured,
    onMounted,
    onRenderTracked,
    onRenderTriggered,
    onScopeDispose,
    onServerPrefetch,
    onUnmounted,
    onUpdated,
    onWatcherCleanup,
    openBlock,
    patchProp,
    popScopeId,
    provide,
    proxyRefs,
    pushScopeId,
    queuePostFlushCb,
    reactive,
    readonly,
    ref,
    registerRuntimeCompiler,
    render,
    renderList,
    renderSlot,
    resolveComponent,
    resolveDirective,
    resolveDynamicComponent,
    resolveFilter,
    resolveTransitionHooks,
    setBlockTracking,
    setDevtoolsHook,
    setTransitionHooks,
    shallowReactive,
    shallowReadonly,
    shallowRef,
    ssrContextKey,
    ssrUtils,
    stop,
    toDisplayString,
    toHandlerKey,
    toHandlers,
    toRaw,
    toRef,
    toRefs,
    toValue,
    transformVNodeArgs,
    triggerRef,
    unref,
    useAttrs,
    useCssModule,
    useCssVars,
    useHost,
    useId,
    useModel,
    useSSRContext,
    useShadowRoot,
    useSlots,
    useTemplateRef,
    useTransitionState,
    vModelCheckbox,
    vModelDynamic,
    vModelRadio,
    vModelSelect,
    vModelText,
    vShow,
    version,
    warn,
    watch,
    watchEffect,
    watchPostEffect,
    watchSyncEffect,
    withAsyncContext,
    withCtx,
    withDefaults,
    withDirectives,
    withKeys,
    withMemo,
    withModifiers,
    withScopeId
  }, Symbol.toStringTag, { value: "Module" }));
  const BASE = "/api";
  function apiURL(path, params = {}) {
    if (!path.startsWith("/")) path = "/" + path;
    const u = new URL(BASE + path, location.origin);
    for (const [k, v] of Object.entries(params)) {
      if (v !== void 0 && v !== null && v !== "") u.searchParams.set(k, v);
    }
    return u.toString();
  }
  async function apiGet(path, params = {}) {
    const r = await fetch(apiURL(path, params));
    if (!r.ok) {
      const e = await r.json().catch(() => ({ error: r.statusText }));
      throw new Error(e.error || e.message || r.statusText);
    }
    return r.json();
  }
  async function apiPost(path, body = {}, params = {}) {
    const r = await fetch(apiURL(path, params), {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body)
    });
    if (!r.ok) {
      const e = await r.json().catch(() => ({ error: r.statusText }));
      throw new Error(e.error || e.message || r.statusText);
    }
    return r.json();
  }
  async function apiPut(path, body = {}) {
    const r = await fetch(apiURL(path), {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body)
    });
    if (!r.ok) {
      const e = await r.json().catch(() => ({ error: r.statusText }));
      throw new Error(e.error || e.message || r.statusText);
    }
    return r.json();
  }
  async function apiDelete(path) {
    const r = await fetch(apiURL(path), { method: "DELETE" });
    if (!r.ok) {
      const e = await r.json().catch(() => ({ error: r.statusText }));
      throw new Error(e.error || e.message || r.statusText);
    }
    return r.json();
  }
  let wsSocket = null;
  let wsReconnectTimer = null;
  let wsReconnectCount = 0;
  const WS_MAX_RECONNECT = 20;
  let wsCallbacks = null;
  let wsManuallyClosed = false;
  let wsPongTimer = null;
  let wsRunningConvs = null;
  function initWebSocket(callbacks) {
    wsCallbacks = callbacks;
    wsManuallyClosed = false;
    wsReconnectCount = 0;
    if (wsSocket && (wsSocket.readyState === WebSocket.OPEN || wsSocket.readyState === WebSocket.CONNECTING)) {
      return;
    }
    doWsConnect();
  }
  function doWsConnect() {
    const proto = location.protocol === "https:" ? "wss:" : "ws:";
    const url = proto + "//" + location.host + "/ws";
    try {
      wsSocket = new WebSocket(url);
    } catch (e) {
      scheduleWsReconnect("WebSocket 创建失败: " + (e.message || e));
      return;
    }
    wsSocket.onopen = () => {
      wsReconnectCount = 0;
      if (wsReconnectTimer) {
        clearTimeout(wsReconnectTimer);
        wsReconnectTimer = null;
      }
      window.dispatchEvent(new CustomEvent("ws-connection-change", { detail: { connected: true } }));
      if (wsPongTimer) clearTimeout(wsPongTimer);
      wsPongTimer = setTimeout(() => {
        console.warn("[WS] 45s 未收到 pong，触发重连");
        wsSocket.close();
      }, 45e3);
    };
    wsSocket.onmessage = (ev) => {
      var _a, _b, _c;
      if (wsPongTimer) {
        clearTimeout(wsPongTimer);
      }
      wsPongTimer = setTimeout(() => {
        if (wsRunningConvs && wsRunningConvs.size > 0) {
          console.warn("[WS] 45s 无业务消息但 agent 运行中，保持连接（等待后端 ping）");
          return;
        }
        console.warn("[WS] 45s 无消息，触发重连");
        if (wsSocket) wsSocket.close();
      }, 45e3);
      let data;
      try {
        data = JSON.parse(ev.data);
      } catch {
        return;
      }
      if (!data) return;
      if (data.type === "ping") {
        return;
      }
      if (data.type === "status" && data.runningConvs) {
        wsRunningConvs = new Set(data.runningConvs);
        (_a = wsCallbacks == null ? void 0 : wsCallbacks.onStatus) == null ? void 0 : _a.call(wsCallbacks, {
          runningConvs: data.runningConvs,
          runningByWorkspace: data.runningByWorkspace || {}
        });
        return;
      }
      const convId = data.convId;
      if (!convId) return;
      if (data.type === "done") {
        (_b = wsCallbacks == null ? void 0 : wsCallbacks.onDone) == null ? void 0 : _b.call(wsCallbacks, convId, data);
      } else {
        (_c = wsCallbacks == null ? void 0 : wsCallbacks.onEvent) == null ? void 0 : _c.call(wsCallbacks, convId, data);
      }
    };
    wsSocket.onclose = () => {
      wsSocket = null;
      window.dispatchEvent(new CustomEvent("ws-connection-change", { detail: { connected: false } }));
      if (!wsManuallyClosed) scheduleWsReconnect("连接已关闭");
    };
    wsSocket.onerror = () => {
    };
  }
  function scheduleWsReconnect(reason) {
    if (wsManuallyClosed) return;
    if (wsReconnectCount >= WS_MAX_RECONNECT) {
      console.warn("[WS] 重连已达上限:", reason);
      if (wsCallbacks == null ? void 0 : wsCallbacks.onDisconnected) {
        wsCallbacks.onDisconnected();
      }
      return;
    }
    wsReconnectCount++;
    const delay = Math.min(500 * Math.pow(1.5, wsReconnectCount - 1), 5e3);
    console.warn("[WS] " + reason + "，" + delay + "ms 后重连 (" + wsReconnectCount + "/" + WS_MAX_RECONNECT + ")");
    if (wsReconnectTimer) clearTimeout(wsReconnectTimer);
    wsReconnectTimer = setTimeout(() => {
      doWsConnect();
    }, delay);
  }
  function reconnectWebSocket() {
    if (wsSocket) {
      wsSocket.onclose = null;
      wsSocket.close();
      wsSocket = null;
    }
    if (wsReconnectTimer) {
      clearTimeout(wsReconnectTimer);
      wsReconnectTimer = null;
    }
    if (wsPongTimer) {
      clearTimeout(wsPongTimer);
      wsPongTimer = null;
    }
    wsReconnectCount = 0;
    wsManuallyClosed = false;
    doWsConnect();
  }
  function closeWebSocket() {
    wsManuallyClosed = true;
    if (wsReconnectTimer) {
      clearTimeout(wsReconnectTimer);
      wsReconnectTimer = null;
    }
    if (wsPongTimer) {
      clearTimeout(wsPongTimer);
      wsPongTimer = null;
    }
    wsCallbacks = null;
    wsReconnectCount = WS_MAX_RECONNECT;
    if (wsSocket) {
      wsSocket.onclose = null;
      wsSocket.close();
      wsSocket = null;
    }
  }
  function isWebSocketOpen() {
    return !!(wsSocket && wsSocket.readyState === WebSocket.OPEN);
  }
  async function waitForWebSocket(timeout = 3e3) {
    if (wsSocket && wsSocket.readyState === WebSocket.OPEN) return true;
    if (wsManuallyClosed) return false;
    if (!wsSocket || wsSocket.readyState === WebSocket.CLOSED) {
      wsReconnectCount = 0;
      doWsConnect();
    }
    const deadline = Date.now() + timeout;
    while (Date.now() < deadline) {
      if (wsSocket && wsSocket.readyState === WebSocket.OPEN) return true;
      await new Promise((r) => setTimeout(r, 100));
    }
    return !!(wsSocket && wsSocket.readyState === WebSocket.OPEN);
  }
  async function chatStart(convId, message, autonomous, workspaceRoot) {
    const body = { convId, message, autonomous };
    if (workspaceRoot) body.workspaceRoot = workspaceRoot;
    return apiPost("/chat/send", body);
  }
  async function chatStop(convId) {
    return apiPost("/chat/stop?convId=" + encodeURIComponent(convId), {});
  }
  async function answerChat(convId, answer) {
    return apiPost("/chat/answer", { convId, answer });
  }
  async function approveChat(convId, approved) {
    return apiPost("/chat/approve", { convId, approved });
  }
  async function sendFeedback(convId, content) {
    return apiPost("/chat/feedback", { convId, content });
  }
  async function chatRollback(convId, msgIdx) {
    return apiPost("/chat/rollback", { convId, msgIdx });
  }
  async function chatCompact(convId) {
    return apiPost("/chat/compact?convId=" + encodeURIComponent(convId), {});
  }
  async function getMessages(convId, { limit = 50, before = null } = {}) {
    const params = { limit };
    if (before !== null && before !== void 0) params.before = before;
    return apiGet("/conversations/" + encodeURIComponent(convId) + "/messages", params);
  }
  async function getMessagesCount(convId) {
    return apiGet("/conversations/" + encodeURIComponent(convId) + "/messages/count");
  }
  async function getModels() {
    return apiGet("/models");
  }
  async function getMcpList(level = "all") {
    return apiGet("/mcp/list", { level });
  }
  async function saveMcpItem({ action, name, command, args, level }) {
    return apiPost("/mcp/save", { action, name, command, args: args || [], level: level || "user" });
  }
  async function getSkillsList() {
    return apiGet("/skills/list");
  }
  async function readSkill(name, level) {
    const params = { name };
    if (level) params.level = level;
    return apiGet("/skills/read", params);
  }
  async function deleteSkill(name) {
    return apiPost("/skills/delete", { name });
  }
  async function saveSkillStatus(name, level, status) {
    return apiPost("/skills/save", { action: "set-status", name, level, status });
  }
  async function getInstructions(scope = "system") {
    return apiGet("/instructions", { scope });
  }
  async function saveInstructions(scope, content) {
    return apiPut("/instructions?scope=" + scope, { content });
  }
  async function getPhilosophy() {
    return apiGet("/philosophy");
  }
  async function savePhilosophy(data) {
    return apiPut("/philosophy", data);
  }
  const api = { apiGet, apiPost, apiPut, apiDelete, initWebSocket, reconnectWebSocket, closeWebSocket, isWebSocketOpen, waitForWebSocket, chatStart, answerChat, approveChat, sendFeedback, chatRollback, chatCompact, chatStop, getMessages, getMessagesCount, getModels, getMcpList, saveMcpItem, getSkillsList, readSkill, deleteSkill, saveSkillStatus, getInstructions, saveInstructions, getPhilosophy, savePhilosophy, listPlugins, getPluginDetail, pluginAction, definePlugin, pluginEmit, pluginClientEvents, pluginClientState, pluginInvoke, pluginClientFailure, builtinPlugins, pluginToolToggle, getToolsets, toolsetEdit };
  async function listPlugins() {
    return apiGet("/plugins");
  }
  async function builtinPlugins(data) {
    if (data) return apiPost("/plugins/builtin", data);
    return apiGet("/plugins/builtin");
  }
  async function pluginToolToggle(tool, enabled) {
    return apiPost("/plugins/tool", { tool, enabled });
  }
  async function getPluginDetail(id) {
    return apiGet("/plugins/detail", { id });
  }
  async function pluginAction(id, action) {
    return apiPost("/plugins/action", { id, action });
  }
  async function definePlugin(data) {
    return apiPost("/plugins/define", data);
  }
  async function pluginEmit(event, payload) {
    return apiPost("/plugins/event", { event, payload });
  }
  async function pluginClientEvents(since) {
    return apiGet("/plugins/client-events", { since });
  }
  async function pluginClientState(snapshot) {
    if (snapshot) return apiPost("/plugins/client-state", snapshot);
    return apiGet("/plugins/client-state");
  }
  async function pluginInvoke(plugin, method, args) {
    return apiPost("/plugins/invoke", { plugin, method, args: args === void 0 ? null : args });
  }
  async function pluginClientFailure(plugin, phase, message) {
    return apiPost("/plugins/client-failure", { plugin, phase, message });
  }
  async function getToolsets(name) {
    return name ? apiGet("/toolsets", { name }) : apiGet("/toolsets");
  }
  async function toolsetEdit(data) {
    return apiPost("/toolsets/edit", data);
  }
  const PERSIST_KEY = "paircode-ide-state";
  const dialogState = /* @__PURE__ */ reactive({
    show: false,
    type: "",
    // 'confirm' | 'prompt' | 'alert'
    title: "",
    message: "",
    confirmText: "确定",
    cancelText: "取消",
    inputValue: "",
    inputPlaceholder: "",
    checkboxLabel: "",
    // confirm 类型时可选 checkbox 文案
    checkboxValue: false,
    // confirm 类型时 checkbox 状态
    resolve: null,
    // Promise resolve 函数
    toasts: []
    // { id, message, type }
  });
  window.$confirm = (message, title = "确认", confirmText = "确定", cancelText = "取消") => {
    return new Promise((resolve2) => {
      dialogState.type = "confirm";
      dialogState.title = title;
      dialogState.message = message;
      dialogState.confirmText = confirmText;
      dialogState.cancelText = cancelText;
      dialogState.checkboxLabel = "";
      dialogState.checkboxValue = false;
      dialogState.show = true;
      dialogState.resolve = resolve2;
    });
  };
  window.$confirmWithCheckbox = (message, title = "确认", checkboxLabel = "", confirmText = "确定", cancelText = "取消") => {
    return new Promise((resolve2) => {
      dialogState.type = "confirm";
      dialogState.title = title;
      dialogState.message = message;
      dialogState.confirmText = confirmText;
      dialogState.cancelText = cancelText;
      dialogState.checkboxLabel = checkboxLabel;
      dialogState.checkboxValue = false;
      dialogState.show = true;
      dialogState.resolve = resolve2;
    });
  };
  window.$prompt = (message, defaultValue = "", title = "输入", confirmText = "确定", cancelText = "取消") => {
    return new Promise((resolve2) => {
      dialogState.type = "prompt";
      dialogState.title = title;
      dialogState.message = message;
      dialogState.inputValue = defaultValue;
      dialogState.inputPlaceholder = "";
      dialogState.confirmText = confirmText;
      dialogState.cancelText = cancelText;
      dialogState.show = true;
      dialogState.resolve = resolve2;
    });
  };
  window.$alert = (message, title = "提示") => {
    return new Promise((resolve2) => {
      dialogState.type = "alert";
      dialogState.title = title;
      dialogState.message = message;
      dialogState.show = true;
      dialogState.resolve = resolve2;
    });
  };
  window.$toast = (message, type = "info", duration = 3e3) => {
    const id = Date.now() + Math.random();
    dialogState.toasts.push({ id, message, type });
    setTimeout(() => {
      dialogState.toasts = dialogState.toasts.filter((t) => t.id !== id);
    }, duration);
  };
  const state = /* @__PURE__ */ reactive({
    activeActivity: "explorer",
    sidebarVisible: true,
    rightPanelVisible: true,
    bottomPanelVisible: true,
    bottomPanelTab: "terminal",
    workspaceRoot: "",
    workspaceFolders: [],
    workspaceName: "",
    wsList: /* @__PURE__ */ reactive([]),
    fileTree: [],
    expandedDirs: {},
    loadingDir: "",
    openFiles: [],
    activeFile: "",
    fileContents: {},
    fileSavedContent: {},
    // 磁盘上原始内容，用于准确判断是否修改
    fileDirty: {},
    cursorLine: 1,
    cursorCol: 1,
    conversations: [],
    currentConvId: "",
    messages: [],
    chatLoading: false,
    chatSessionId: "",
    agentRunning: false,
    // ── 多会话并行：按 convId 存储各对话的独立状态 ──
    messagesByConv: {},
    // { [convId]: [...] } 各对话消息数组
    loadingByConv: {},
    // { [convId]: boolean } 各对话加载状态
    agentRunningByConv: {},
    // { [convId]: boolean } 各对话 agent 运行状态
    approvalByConv: {},
    // { [convId]: { callId, tool, args, waiting } } 各对话审批状态
    phaseByConv: {},
    // { [convId]: string } 各对话当前阶段（自主模式）
    nudgeByConv: {},
    // { [convId]: string } 各对话 nudge 提示文本
    convCtxStatsByConv: {},
    // { [convId]: reactive({...}) } 各对话上下文 token 统计
    msgTotalByConv: {},
    // { [convId]: number } 各对话总消息数（懒加载判断是否还有更早消息）
    msgLoadedByConv: {},
    // { [convId]: number } 各对话已加载消息数
    runningByWorkspace: {},
    // { [wsRoot]: count } 各工作区运行中 agent 计数（供工作区列表显示脉冲点）
    wsTokenStatsByWs: {},
    // { [wsRoot]: { promptTokens, ... } } 各工作区 token 统计（隔离）
    settings: {},
    settingsLoaded: false,
    pluginSchemas: [],
    // 插件注册的配置段（ctx.registerSettings → GET /api/settings.schemas）
    searchResults: [],
    selectedFilePaths: [],
    // 文件树多选路径列表
    lastClickedFilePath: "",
    // 文件树最近点击（Shift范围选择用）
    tasks: [],
    notificationCount: 0,
    theme: "dark",
    focusMode: false
    // ★ 默认非专注：编辑器+对话区并排（右侧宽度可拖拽调整）；Ctrl+K 切换专注（隐藏编辑器）
  });
  const showSettings = /* @__PURE__ */ ref(false);
  const showSystem = /* @__PURE__ */ ref(false);
  const showSource = /* @__PURE__ */ ref(false);
  const showMarketplace = /* @__PURE__ */ ref(false);
  const showAbout = /* @__PURE__ */ ref(false);
  const showQuickSwitcher = /* @__PURE__ */ ref(false);
  const helpDocTarget = /* @__PURE__ */ ref("features");
  const showHelp = /* @__PURE__ */ ref(false);
  const showHelpWrapper = computed({
    get() {
      return showHelp.value;
    },
    set(v) {
      if (typeof v === "string") {
        helpDocTarget.value = v;
        showHelp.value = true;
      } else {
        showHelp.value = !!v;
        if (showHelp.value) helpDocTarget.value = "getting-started";
      }
    }
  });
  const bottomPanelHeight = /* @__PURE__ */ ref(180);
  const rightPanelWidth = /* @__PURE__ */ ref(320);
  const sidebarWidth = /* @__PURE__ */ ref(280);
  function loadPanelSize() {
    try {
      const d = JSON.parse(localStorage.getItem("paircode-panel-size") || "{}");
      if (d.rpw) {
        const v = parseFloat(d.rpw);
        rightPanelWidth.value = Number.isFinite(v) ? Math.max(0, Math.min(v, window.innerWidth - 593)) : 320;
      }
      if (d.bph) bottomPanelHeight.value = Math.max(120, Math.min(parseFloat(d.bph) || 180, 500));
    } catch {
    }
    try {
      const sw = localStorage.getItem("paircode-sidebar-width");
      if (sw) sidebarWidth.value = Math.min(Math.max(parseInt(sw, 10) || 280, 160), 480);
    } catch {
    }
  }
  function savePanelSize() {
    try {
      localStorage.setItem("paircode-panel-size", JSON.stringify({
        rpw: rightPanelWidth.value,
        bph: bottomPanelHeight.value
      }));
    } catch {
    }
    try {
      localStorage.setItem("paircode-sidebar-width", String(sidebarWidth.value));
    } catch {
    }
  }
  loadPanelSize();
  if (typeof window !== "undefined") window.__state = state;
  const FONT_CONFIG = {
    dark: {
      ui: ["Inter:400,500,600,700"],
      code: ["JetBrains Mono:400,500,600"],
      google: ["Inter", "JetBrains Mono"]
    },
    light: {
      ui: ["Inter:400,500,600,700"],
      code: ["JetBrains Mono:400,500,600"],
      google: ["Inter", "JetBrains Mono"]
    },
    warm: {
      ui: ["Inter:400,500,600,700"],
      code: ["JetBrains Mono:400,500,600"],
      google: ["Inter", "JetBrains Mono"]
    },
    night: {
      ui: ["Inter:400,500,600,700"],
      code: ["JetBrains Mono:400,500,600"],
      google: ["Inter", "JetBrains Mono"]
    }
  };
  let fontLinkEl = null;
  function loadThemeFonts(theme) {
    const cfg = FONT_CONFIG[theme] || FONT_CONFIG.dark;
    const families = [cfg.ui[0], cfg.code[0]].filter(Boolean).join("&family=");
    const href = "https://fonts.geekzu.org/css2?family=" + families + "&display=swap";
    if (fontLinkEl) {
      document.head.removeChild(fontLinkEl);
      fontLinkEl = null;
    }
    try {
      const link = document.createElement("link");
      link.rel = "stylesheet";
      link.href = href;
      link.onload = () => {
        fontLinkEl = link;
      };
      link.onerror = () => {
      };
      document.head.appendChild(link);
    } catch {
    }
  }
  function applyTheme(themeName) {
    const theme = themeName || state.theme || "dark";
    state.theme = theme;
    document.documentElement.classList.remove("theme-dark", "theme-light", "theme-warm", "theme-night");
    document.body.classList.remove("theme-dark", "theme-light", "theme-warm", "theme-night");
    const cls = "theme-" + theme;
    document.documentElement.classList.add(cls);
    document.body.classList.add(cls);
    loadThemeFonts(theme);
    savePersistentState();
  }
  function savePersistentState() {
    try {
      const data = {
        version: 1,
        activeActivity: state.activeActivity,
        sidebarVisible: state.sidebarVisible,
        rightPanelVisible: state.rightPanelVisible,
        bottomPanelVisible: state.bottomPanelVisible,
        bottomPanelTab: state.bottomPanelTab,
        theme: state.theme
        // focusMode 不持久化：专注模式是临时视图状态（Ctrl+K），跨会话记住
        // 会导致用户浏览器残留 true 时每次打开都隐藏编辑器（历史坑）。
      };
      localStorage.setItem(PERSIST_KEY, JSON.stringify(data));
    } catch (e) {
      console.warn("savePersistentState error:", e);
    }
  }
  function loadPersistentState() {
    try {
      const raw = localStorage.getItem(PERSIST_KEY);
      if (!raw) return;
      const data = JSON.parse(raw);
      if (!data || !data.version) return;
      if (typeof data.sidebarVisible === "boolean") state.sidebarVisible = data.sidebarVisible;
      if (typeof data.rightPanelVisible === "boolean") state.rightPanelVisible = data.rightPanelVisible;
      if (typeof data.bottomPanelVisible === "boolean") state.bottomPanelVisible = data.bottomPanelVisible;
      if (data.bottomPanelTab) state.bottomPanelTab = data.bottomPanelTab;
      if (data.theme) {
        if (["dark", "light", "warm", "night"].includes(data.theme)) {
          applyTheme(data.theme);
        }
      }
    } catch (e) {
    }
  }
  const uiState = /* @__PURE__ */ Object.freeze(/* @__PURE__ */ Object.defineProperty({
    __proto__: null,
    PERSIST_KEY,
    applyTheme,
    bottomPanelHeight,
    dialogState,
    helpDocTarget,
    loadPanelSize,
    loadPersistentState,
    rightPanelWidth,
    savePanelSize,
    savePersistentState,
    showAbout,
    showHelp,
    showHelpWrapper,
    showMarketplace,
    showQuickSwitcher,
    showSettings,
    showSource,
    showSystem,
    sidebarWidth,
    state
  }, Symbol.toStringTag, { value: "Module" }));
  const __registry = typeof window !== "undefined" ? window.__SLOT_REGISTRY = window.__SLOT_REGISTRY || { instances: [], clientSlots: [], clientPanels: [] } : { instances: [], clientSlots: [], clientPanels: [] };
  const instances = __registry.instances;
  let pollTimer = null;
  let lastSeq = 0;
  let pollInterval = 2e3;
  const clientPanels = __registry.clientPanels;
  let panelMountFn = null;
  const clientSlots = __registry.clientSlots;
  const slotOwnerKey = (id) => "paircode-slot-" + id;
  let slotMountFns = [];
  function setSlotMount(fn) {
    if (!fn) {
      slotMountFns = [];
      return () => {
      };
    }
    slotMountFns.push(fn);
    try {
      fn(clientSlots);
    } catch (e) {
      console.warn("[slot] 初始通知失败", e);
    }
    return () => {
      const i = slotMountFns.indexOf(fn);
      if (i >= 0) slotMountFns.splice(i, 1);
    };
  }
  function emitSlotChanged() {
    for (const fn of slotMountFns) {
      try {
        fn(clientSlots);
      } catch (e) {
        console.warn("[slot] 通知失败", e);
      }
    }
  }
  function overlayKey(slotId, pluginName) {
    return "slotOverlay:" + slotId + ":" + pluginName;
  }
  function isOverlayActive(slotId, pluginName) {
    try {
      const v = localStorage.getItem(overlayKey(slotId, pluginName));
      if (v === null) return true;
      return v === "1";
    } catch (e) {
      return false;
    }
  }
  function setOverlayActive(slotId, pluginName, on) {
    try {
      localStorage.setItem(overlayKey(slotId, pluginName), on ? "1" : "0");
    } catch (e) {
    }
    emitSlotChanged();
    persistAssembly();
  }
  function uiEnabledKey(pluginName) {
    return "slotUIEnabled:" + pluginName;
  }
  function isPluginUIEnabled(pluginName) {
    try {
      const v = localStorage.getItem(uiEnabledKey(pluginName));
      if (v === null) return true;
      return v === "1";
    } catch (e) {
      return true;
    }
  }
  function setPluginUIEnabled(pluginName, on) {
    try {
      localStorage.setItem(uiEnabledKey(pluginName), on ? "1" : "0");
    } catch (e) {
    }
    emitSlotChanged();
    persistAssembly();
  }
  function getSlotCandidates(slotId) {
    return clientSlots.filter((s) => s.slotId === slotId);
  }
  function getSlotOwner(slotId) {
    let v = "";
    let neverChosen = true;
    try {
      v = localStorage.getItem(slotOwnerKey(slotId)) || "";
      neverChosen = localStorage.getItem(slotOwnerKey(slotId)) === null;
    } catch (e) {
    }
    if (v && !clientSlots.some((s) => s.slotId === slotId && s.kind !== "list" && s.pluginName === v && isPluginUIEnabled(v))) {
      neverChosen = true;
      v = "";
    }
    if (!v) neverChosen = true;
    if (v) return v;
    if (neverChosen) {
      const cands = clientSlots.filter((s) => s.slotId === slotId && s.kind !== "list" && typeof s.render === "function" && isPluginUIEnabled(s.pluginName));
      if (cands.length === 1) return cands[0].pluginName;
    }
    return "";
  }
  function setSlotOwner(slotId, pluginName) {
    try {
      localStorage.setItem(slotOwnerKey(slotId), pluginName || "");
    } catch (e) {
    }
    emitSlotChanged();
    persistAssembly();
  }
  let assemblyTimer = null;
  function persistAssembly() {
    if (assemblyTimer) return;
    assemblyTimer = setTimeout(async () => {
      assemblyTimer = null;
      try {
        const slotOwner = {};
        const slotOverlay = {};
        const slotUIEnabled = {};
        for (const s of clientSlots) {
          const v = localStorage.getItem(slotOwnerKey(s.slotId));
          if (v) slotOwner[s.slotId] = v;
          if (s.kind === "list") {
            const ov = localStorage.getItem(overlayKey(s.slotId, s.pluginName));
            if (ov === "0") slotOverlay[s.slotId + ":" + s.pluginName] = false;
            else if (ov === "1") slotOverlay[s.slotId + ":" + s.pluginName] = true;
          }
          const ue = localStorage.getItem(uiEnabledKey(s.pluginName));
          if (ue === "0") slotUIEnabled[s.pluginName] = false;
          else if (ue === "1") slotUIEnabled[s.pluginName] = true;
        }
        const res = await fetch("/api/ui-assembly", {
          method: "PUT",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ slotOwner, slotOverlay, slotUIEnabled })
        });
        if (!res.ok) console.warn("[plugin-runtime] 装配状态落盘失败", res.status);
      } catch (e) {
      }
    }, 400);
  }
  async function loadAssemblyFile() {
    try {
      const res = await fetch("/api/ui-assembly");
      if (!res.ok) return;
      const data = await res.json();
      if (!data || typeof data !== "object") return;
      const owner = data.slotOwner || {};
      const overlay = data.slotOverlay || {};
      const enabled = data.slotUIEnabled || {};
      let changed = false;
      for (const [slotId, pname] of Object.entries(owner)) {
        if (pname && localStorage.getItem(slotOwnerKey(slotId)) !== pname) {
          localStorage.setItem(slotOwnerKey(slotId), pname);
          changed = true;
        }
      }
      for (const [k, v] of Object.entries(overlay)) {
        const key = "slotOverlay:" + k;
        const val = v ? "1" : "0";
        if (localStorage.getItem(key) !== val) {
          localStorage.setItem(key, val);
          changed = true;
        }
      }
      for (const [name, v] of Object.entries(enabled)) {
        const key = uiEnabledKey(name);
        const val = v ? "1" : "0";
        if (localStorage.getItem(key) !== val) {
          localStorage.setItem(key, val);
          changed = true;
        }
      }
      if (changed) emitSlotChanged();
      console.log("[plugin-runtime] 装配状态已从 .pair/ui-assembly.json 合并（" + Object.keys(owner).length + " owner / " + Object.keys(overlay).length + " overlay / " + Object.keys(enabled).length + " uiEnabled）");
    } catch (e) {
    }
  }
  function getSlotUI(slotId) {
    const owner = getSlotOwner(slotId);
    if (!owner) return null;
    const s = clientSlots.find((x) => x.slotId === slotId && x.pluginName === owner);
    if (!s || typeof s.render !== "function") return null;
    return { render: s.render, ui: getUIFor(owner), pluginName: owner };
  }
  function getSlotUIList(slotId) {
    return clientSlots.filter((s) => s.slotId === slotId && s.kind === "list" && typeof s.render === "function" && isPluginUIEnabled(s.pluginName)).map((s) => ({ render: s.render, ui: getUIFor(s.pluginName), pluginName: s.pluginName }));
  }
  function mountListSlot(hostRef, slotId, opts = {}) {
    const isActive = opts.isActive || ((n) => isOverlayActive(slotId, n));
    const cleanups = /* @__PURE__ */ new Map();
    function render2() {
      const host = hostRef && hostRef.value;
      if (!host) return;
      for (const [name, c] of cleanups) {
        try {
          c();
        } catch (e) {
          console.warn("[slot] " + name + " cleanup 失败", e);
        }
      }
      cleanups.clear();
      host.innerHTML = "";
      for (const s of getSlotUIList(slotId)) {
        if (!isActive(s.pluginName)) continue;
        const item = document.createElement("div");
        item.className = "plugin-slot-item plugin-slot-" + slotId + "-item";
        item.dataset.plugin = s.pluginName;
        host.appendChild(item);
        try {
          const ret = s.render(item, s.ui);
          if (typeof ret === "function") cleanups.set(s.pluginName, ret);
        } catch (e) {
          console.warn("[slot] " + slotId + " 渲染失败", e);
          item.innerHTML = '<span style="color:var(--text-muted);font-size:11px">插件条目渲染失败</span>';
        }
      }
    }
    return setSlotMount(() => {
      render2();
    });
  }
  function useSingleSlot(slotId) {
    const owner = /* @__PURE__ */ ref("");
    const hostRef = /* @__PURE__ */ ref(null);
    let cleanup = null;
    let unsub = null;
    let started = false;
    let rendered = false;
    function render2() {
      const host = hostRef.value;
      if (!host) return;
      if (typeof cleanup === "function") {
        try {
          cleanup();
        } catch (e) {
        }
        cleanup = null;
      }
      host.innerHTML = "";
      const s = getSlotUI(slotId);
      if (s && typeof s.render === "function") {
        try {
          const ret = s.render(host, s.ui);
          if (typeof ret === "function") cleanup = ret;
        } catch (e) {
          console.warn("[slot] " + slotId + " 渲染失败", e);
          host.innerHTML = '<div style="padding:8px;font-size:12px;color:var(--text-muted)">插件「' + slotId + "」渲染失败</div>";
        }
      }
    }
    function refresh() {
      const prev = owner.value;
      owner.value = getSlotOwner(slotId);
      if (rendered && owner.value === prev) return;
      if (typeof cleanup === "function") {
        try {
          cleanup();
        } catch (e) {
        }
        cleanup = null;
      }
      nextTick(() => {
        if (owner.value) {
          render2();
          rendered = true;
        }
      });
    }
    return {
      owner,
      hostRef,
      // ★ setup 顶层同步调用：初始化 owner（避免首帧先 mount 内置组件再切插件
      //   分支——复杂组件（CM6 编辑器等）的「mount 后立即卸载」会触发错误）。
      //   在组件 setup 里 useSingleSlot(...) 后立即调用。
      init() {
        owner.value = getSlotOwner(slotId);
      },
      start() {
        if (started) return;
        started = true;
        refresh();
        unsub = setSlotMount(refresh);
      },
      stop() {
        started = false;
        if (unsub) {
          unsub();
          unsub = null;
        }
        if (typeof cleanup === "function") {
          try {
            cleanup();
          } catch (e) {
          }
          cleanup = null;
        }
      }
    };
  }
  function setPanelMount(fn) {
    panelMountFn = fn;
    if (panelMountFn) panelMountFn(clientPanels);
  }
  function emitPanelChanged() {
    if (panelMountFn) panelMountFn(clientPanels);
  }
  function makeUI(inst) {
    const ui = {
      // 收 host → 浏览器事件（ui:/client: 前缀由后端转发）
      on(event, fn) {
        if (typeof fn !== "function") return;
        if (!inst.onHandlers.has(event)) inst.onHandlers.set(event, []);
        inst.onHandlers.get(event).push(fn);
        if (!inst.events.includes(event)) inst.events.push(event);
        return () => {
          const arr = inst.onHandlers.get(event) || [];
          const i = arr.indexOf(fn);
          if (i >= 0) arr.splice(i, 1);
          if ((inst.onHandlers.get(event) || []).length === 0) {
            inst.onHandlers.delete(event);
            const ei = inst.events.indexOf(event);
            if (ei >= 0) inst.events.splice(ei, 1);
          }
        };
      },
      // 发事件回 host（host: 前缀约定；host 插件 ctx.on 消费）
      async emit(event, payload) {
        try {
          await api.pluginEmit(event, payload);
        } catch (e) {
          console.warn("[plugin] emit 失败", event, e);
        }
      },
      // D11 invoke RPC：远程调用 host 半注册的方法（ctx.registerClientMethod）。
      // 返回 {ok, value} 或 {ok:false, error}；插件可再 await 值。
      async invoke(plugin, method, args) {
        const res = await api.pluginInvoke(plugin, method, args);
        if (!res || !res.ok) {
          throw new Error(res && res.error || "invoke 失败: " + method);
        }
        return res.value;
      },
      // D11 失败上报：client 半 render/guard/boot 失败 → 后端记诊断，Agent 经
      // cordis_inspect 发现修复（不中断 host 半运行）。
      reportFailure(phase, message) {
        const ph = phase === "guard" || phase === "boot" ? phase : "render";
        api.pluginClientFailure(inst.name, ph, String(message || "unknown error")).catch(() => {
        });
      },
      // 注册自定义面板（显示在插件面板 client 区）
      registerPanel(spec) {
        if (!spec || !spec.id || !spec.title) {
          console.warn("[plugin] registerPanel 需要 {id, title, render?}");
          return;
        }
        const existing = clientPanels.findIndex((p2) => p2.id === spec.id);
        const panel = {
          id: spec.id,
          title: spec.title,
          icon: spec.icon || "sparkles",
          render: typeof spec.render === "function" ? spec.render : null,
          // 轻量 Slot：props 声明面板数据契约（{field: type}），宿主/其他插件可注入
          props: spec.props && typeof spec.props === "object" ? { ...spec.props } : null,
          pluginName: inst.name
        };
        if (existing >= 0) clientPanels[existing] = panel;
        else clientPanels.push(panel);
        emitPanelChanged();
        return {
          // 更新面板内容（重新渲染）
          update() {
            emitPanelChanged();
          },
          // 移除面板
          remove() {
            const i = clientPanels.findIndex((p2) => p2.id === panel.id);
            if (i >= 0) {
              clientPanels.splice(i, 1);
              emitPanelChanged();
            }
          }
        };
      },
      // 注册 UI 槽位占用（Slot 系统：替换宿主预定义界面区域，如 'statusbar'/'chat'；
      // kind='list' 槽位为叠加型——多个占用者同时渲染，如 'overlay' 浮动层）。
      // 同插件重复注册同槽位 → 替换。single 槽位宿主按 getSlotOwner 决定激活哪个
      // 占用者；list 槽位宿主渲染全部占用者。激活后调 render(el, ui)；render 可
      // 返回 cleanup 函数（宿主下次重渲染前调用）。
      registerSlot(spec) {
        if (!spec || !spec.slotId || !spec.title) {
          console.warn("[plugin] registerSlot 需要 {slotId, title, render?}");
          return;
        }
        const idx = clientSlots.findIndex((s) => s.slotId === spec.slotId && s.pluginName === inst.name);
        const slot = {
          slotId: spec.slotId,
          pluginName: inst.name,
          title: spec.title,
          kind: spec.kind === "list" ? "list" : "single",
          render: typeof spec.render === "function" ? spec.render : null,
          defId: inst.defId
        };
        if (idx >= 0) clientSlots[idx] = slot;
        else clientSlots.push(slot);
        emitSlotChanged();
        return {
          update() {
            emitSlotChanged();
          },
          remove() {
            const i = clientSlots.findIndex((s) => s.slotId === spec.slotId && s.pluginName === inst.name);
            if (i >= 0) {
              clientSlots.splice(i, 1);
              emitSlotChanged();
            }
          }
        };
      },
      // 受限后端 API（相对路径）
      http: {
        get: (path, params) => api.apiGet(path, params),
        post: (path, body) => api.apiPost(path, body)
      },
      log: (...args) => console.log("[plugin:" + inst.name + "]", ...args)
    };
    return ui;
  }
  function loadClientHalf(source) {
    if (!source || !source.clientCode || !String(source.clientCode).trim()) return null;
    const code = String(source.clientCode);
    let fn;
    try {
      const t = code.trim().replace(/^(?:\s*(?:\/\/[^\r\n]*|\/\*[\s\S]*?\*\/)\r?\n?)+/, "").trim();
      const isFnExpr = /^\(?\s*(async\s+)?(\(?\s*ui\s*\)?\s*=>|function\s*\()/.test(t);
      fn = new Function("ui", '"use strict";\n' + (isFnExpr ? "return (" + code + ")(ui)" : code));
    } catch (e) {
      console.warn("[plugin] client 半语法错误", source.name, e);
      return null;
    }
    const inst = {
      name: source.name,
      defId: source.defId,
      source: source.source || "js",
      status: "loaded",
      events: [],
      onHandlers: /* @__PURE__ */ new Map()
    };
    inst.ui = makeUI(inst);
    try {
      fn(inst.ui);
    } catch (e) {
      console.warn("[plugin] client 半执行错误", source.name, e);
      inst.status = "error";
      inst.error = String(e && e.message || e);
      instances.push(inst);
      api.pluginClientFailure(source.name, "boot", inst.error).catch(() => {
      });
      emitPanelChanged();
      emitSlotChanged();
      return inst;
    }
    instances.push(inst);
    emitPanelChanged();
    emitSlotChanged();
    return inst;
  }
  function unloadClientHalf(nameOrDefId) {
    const i = instances.findIndex((inst) => inst.name === nameOrDefId || inst.defId === nameOrDefId);
    if (i >= 0) {
      const name = instances[i].name;
      instances.splice(i, 1);
      for (let j = clientPanels.length - 1; j >= 0; j--) {
        if (clientPanels[j].pluginName === name) {
          clientPanels.splice(j, 1);
        }
      }
      for (let j = clientSlots.length - 1; j >= 0; j--) {
        if (clientSlots[j].pluginName === name) {
          clientSlots.splice(j, 1);
        }
      }
      emitPanelChanged();
      emitSlotChanged();
    }
    reportState();
  }
  async function syncClientHalves(plugins) {
    if (!plugins || !Array.isArray(plugins)) return;
    const active = /* @__PURE__ */ new Set();
    for (const p2 of plugins) {
      if (p2.hasClient && p2.state === "running" && p2.clientApproved && p2.clientCode) {
        active.add(p2.name);
        const exists = instances.find((inst) => inst.name === p2.name);
        if (!exists || exists.status === "error") {
          if (exists) {
            const ei = instances.indexOf(exists);
            if (ei >= 0) instances.splice(ei, 1);
          }
          loadClientHalf({ name: p2.name, defId: p2.defId, clientCode: p2.clientCode });
        }
      }
    }
    for (let i = instances.length - 1; i >= 0; i--) {
      if (!active.has(instances[i].name)) {
        instances.splice(i, 1);
      }
    }
    const liveNames = new Set(instances.map((x) => x.name));
    for (let j = clientPanels.length - 1; j >= 0; j--) {
      if (!liveNames.has(clientPanels[j].pluginName)) {
        clientPanels.splice(j, 1);
      }
    }
    for (let j = clientSlots.length - 1; j >= 0; j--) {
      if (!liveNames.has(clientSlots[j].pluginName)) {
        clientSlots.splice(j, 1);
      }
    }
    emitPanelChanged();
    emitSlotChanged();
    reportState();
  }
  function dispatchHostEvent(ev) {
    for (const inst of instances) {
      const fns = inst.onHandlers.get(ev.name);
      if (!fns) continue;
      for (const fn of fns) {
        try {
          fn(ev.payload);
        } catch (e) {
          console.warn("[plugin] 事件处理错误", inst.name, ev.name, e);
        }
      }
    }
  }
  function startPolling() {
    if (pollTimer) return;
    pollTimer = setInterval(async () => {
      try {
        const res = await api.pluginClientEvents(lastSeq);
        if (res && Array.isArray(res.events)) {
          for (const ev of res.events) dispatchHostEvent(ev);
          if (typeof res.lastSeq === "number") lastSeq = res.lastSeq;
        }
      } catch (e) {
      }
      reportState();
    }, pollInterval);
  }
  function buildSnapshot() {
    const panels = clientPanels.map((p2) => p2.id);
    const plugins = instances.map((inst) => ({
      name: inst.name,
      status: inst.status,
      version: inst.defId || "",
      ...inst.error ? { error: inst.error } : {},
      ...inst.events && inst.events.length ? { events: [...inst.events] } : {}
    }));
    for (const p2 of plugins) {
      const mine = clientPanels.filter((cp) => cp.pluginName === p2.name).map((cp) => cp.id);
      if (mine.length) p2.panels = mine;
      const mineSlots = clientSlots.filter((cs) => cs.pluginName === p2.name).map((cs) => cs.slotId);
      if (mineSlots.length) p2.slots = mineSlots;
    }
    const slots = clientSlots.map((s) => s.slotId);
    return { plugins, ...panels.length ? { panels } : {}, ...slots.length ? { slots } : {} };
  }
  async function reportState() {
    try {
      await api.pluginClientState(buildSnapshot());
    } catch (e) {
    }
  }
  function stopPolling() {
    if (pollTimer) {
      clearInterval(pollTimer);
      pollTimer = null;
    }
  }
  function getInstances() {
    return instances.map((x) => ({ name: x.name, defId: x.defId, source: x.source }));
  }
  function getUIFor(pluginName) {
    const inst = instances.find((x) => x.name === pluginName);
    return inst ? inst.ui : void 0;
  }
  const pluginRuntime = {
    loadClientHalf,
    unloadClientHalf,
    syncClientHalves,
    startPolling,
    stopPolling,
    dispatchHostEvent,
    getInstances,
    setPanelMount,
    clientPanels,
    clientSlots,
    setSlotMount,
    getSlotCandidates,
    getSlotOwner,
    setSlotOwner,
    getSlotUI,
    getSlotUIList,
    emitSlotChanged,
    isOverlayActive,
    setOverlayActive,
    isPluginUIEnabled,
    setPluginUIEnabled,
    mountListSlot,
    loadAssemblyFile
  };
  if (typeof window !== "undefined") {
    window.__pluginRuntime = {
      instances: () => instances.map((i) => ({ name: i.name, status: i.status, error: i.error || "" })),
      clientSlots: () => clientSlots.map((s) => ({ slotId: s.slotId, pluginName: s.pluginName, title: s.title, hasRender: typeof s.render === "function" })),
      clientPanels: () => clientPanels.map((p2) => ({ id: p2.id, pluginName: p2.pluginName })),
      getSlotOwner
    };
  }
  const pluginRuntime$1 = /* @__PURE__ */ Object.freeze(/* @__PURE__ */ Object.defineProperty({
    __proto__: null,
    buildSnapshot,
    clientPanels,
    clientSlots,
    default: pluginRuntime,
    emitSlotChanged,
    getInstances,
    getSlotCandidates,
    getSlotOwner,
    getSlotUI,
    getSlotUIList,
    getUIFor,
    isOverlayActive,
    isPluginUIEnabled,
    loadAssemblyFile,
    loadClientHalf,
    mountListSlot,
    reportState,
    setOverlayActive,
    setPanelMount,
    setPluginUIEnabled,
    setSlotMount,
    setSlotOwner,
    startPolling,
    stopPolling,
    syncClientHalves,
    unloadClientHalf,
    useSingleSlot
  }, Symbol.toStringTag, { value: "Module" }));
  const runtimes = {};
  let msgKeyCounter = 0;
  function makeMsgKey() {
    return "msg_" + Date.now() + "_" + msgKeyCounter++;
  }
  function pushSegment(segs, type, initial) {
    const last = segs[segs.length - 1];
    if (last && last.type === type) return last;
    const seg = { type, content: "", ...initial };
    segs.push(seg);
    return seg;
  }
  let globalCtx = {};
  function setGlobalCtx(ctx) {
    globalCtx = ctx || {};
  }
  function startConvRuntime(convId, msgKey, lastUserText = "") {
    runtimes[convId] = {
      msgKey,
      finalContent: "",
      lastUserText
    };
  }
  function getConvRuntime(convId) {
    return runtimes[convId] || null;
  }
  function resetConvRuntime(convId) {
    delete runtimes[convId];
  }
  function findMsgByKey(msgs, key) {
    if (!msgs || !key) return null;
    for (const m of msgs) {
      if (m._key === key) return m;
    }
    return null;
  }
  function createAssistantPlaceholder(convId, key) {
    const msgs = state.messagesByConv[convId];
    if (!msgs) return "";
    if (!key) key = makeMsgKey();
    let nextIdx = msgs.length;
    for (const m of msgs) {
      if ((m._idx ?? 0) >= nextIdx) nextIdx = (m._idx ?? 0) + 1;
    }
    const assistantMsg = {
      role: "assistant",
      content: "",
      segments: [],
      toolCalls: [],
      _key: key,
      _idx: nextIdx,
      _time: (/* @__PURE__ */ new Date()).toLocaleTimeString("zh-CN", { hour: "2-digit", minute: "2-digit" }),
      _loading: true
    };
    msgs.push(assistantMsg);
    return key;
  }
  function processAgentEvent(convId, data) {
    if (!state.messagesByConv[convId]) state.messagesByConv[convId] = [];
    const msgs = state.messagesByConv[convId];
    let rt = runtimes[convId];
    if (!rt) {
      const lastLoading = [...msgs].reverse().find((m) => m.role === "assistant" && m._loading);
      if (lastLoading && lastLoading._key) {
        console.log("[AE] processAgentEvent 自动恢复 runtime conv=%s key=%s type=%s", convId, lastLoading._key, data.type);
        rt = { msgKey: lastLoading._key, finalContent: "", lastUserText: "" };
        runtimes[convId] = rt;
      } else if (data.type === "content" || data.type === "thinking" || data.type === "done" || data.type === "error") {
        console.log("[AE] processAgentEvent 创建临时占位 conv=%s type=%s", convId, data.type);
        const key = makeMsgKey();
        let phNextIdx = msgs.length;
        for (const m of msgs) {
          if ((m._idx ?? 0) >= phNextIdx) phNextIdx = (m._idx ?? 0) + 1;
        }
        const placeholder = {
          role: "assistant",
          content: "",
          segments: [],
          toolCalls: [],
          _key: key,
          _idx: phNextIdx,
          _time: (/* @__PURE__ */ new Date()).toLocaleTimeString("zh-CN", { hour: "2-digit", minute: "2-digit" }),
          _loading: false
        };
        msgs.push(placeholder);
        rt = { msgKey: key, finalContent: "", lastUserText: "" };
        runtimes[convId] = rt;
      } else {
        console.warn("[AE] processAgentEvent 丢弃(无 runtime 且无法恢复): conv=%s type=%s", convId, data.type);
        return;
      }
    }
    if (!msgs) {
      console.warn("[AE] processAgentEvent 丢弃: 无 messagesByConv conv=%s type=%s", convId, data.type);
      return;
    }
    const isCurrent = state.currentConvId === convId;
    let msg = findMsgByKey(msgs, rt.msgKey);
    if (!msg) {
      const lastAssistant = [...msgs].reverse().find((m) => m.role === "assistant");
      if (lastAssistant) {
        console.log("[AE] processAgentEvent fallback 到最后 assistant conv=%s type=%s", convId, data.type);
        msg = lastAssistant;
        if (lastAssistant._key) rt.msgKey = lastAssistant._key;
      }
    }
    if (!msg) {
      console.warn("[AE] processAgentEvent 丢弃: 无目标 msg conv=%s type=%s", convId, data.type);
      return;
    }
    msg._loading = false;
    if (data.type === "thinking") {
      const seg = pushSegment(msg.segments, "thinking", { _mode: "collapsed", _collapsed: true });
      seg.content += data.content || "";
    } else if (data.type === "content") {
      rt.finalContent += data.content || "";
      const seg = pushSegment(msg.segments, "content");
      seg.content += data.content || "";
    } else if (data.type === "tool_call") {
      const toolName = data.tool || data.name || "";
      if (toolName === "ask_user") {
        let question = "";
        let askType = "text";
        let options = [];
        try {
          const args = typeof data.args === "string" ? JSON.parse(data.args) : data.args;
          question = args.question || "（无问题内容）";
          askType = args.askType || args.type || "text";
          if (Array.isArray(args.options)) {
            options = args.options;
          }
        } catch {
        }
        msg.segments.push({
          type: "ask_user",
          question,
          askType,
          options,
          callId: data.callId || data.callID || "",
          answer: "",
          _answered: false
        });
      } else if (toolName === "update_plan") {
        try {
          const args = data.args ? typeof data.args === "string" ? JSON.parse(data.args) : data.args : {};
          if (Array.isArray(args.plan) && globalCtx.onPlanUpdate) globalCtx.onPlanUpdate(args.plan, convId);
        } catch {
        }
        msg.segments.push({
          type: "tool_call",
          name: toolName,
          callId: data.callId || data.callID || "",
          argsRaw: data.args ? typeof data.args === "string" ? data.args : JSON.stringify(data.args, null, 2) : "",
          result: "",
          _mode: "collapsed",
          _collapsed: false,
          _expanded: false
        });
      } else if (toolName === "task_create") {
        try {
          const args = data.args ? typeof data.args === "string" ? JSON.parse(data.args) : data.args : {};
          if (globalCtx.onTaskCreate) globalCtx.onTaskCreate({ step: args.subject || "(新建任务)", status: "pending", callId: data.callId || data.callID || "", _taskId: null, planStepIndex: args.plan_step_index ?? null }, convId);
        } catch {
        }
        msg.segments.push({
          type: "tool_call",
          name: toolName,
          callId: data.callId || data.callID || "",
          argsRaw: data.args ? typeof data.args === "string" ? data.args : JSON.stringify(data.args, null, 2) : "",
          result: "",
          _mode: "expanded",
          _expanded: true
        });
      } else if (toolName === "task_update") {
        try {
          const args = data.args ? typeof data.args === "string" ? JSON.parse(data.args) : data.args : {};
          if (globalCtx.onTaskUpdate) globalCtx.onTaskUpdate(args.id, args.status || "", args.subject || "", convId);
        } catch {
        }
        msg.segments.push({
          type: "tool_call",
          name: toolName,
          callId: data.callId || data.callID || "",
          argsRaw: data.args ? typeof data.args === "string" ? data.args : JSON.stringify(data.args, null, 2) : "",
          result: "",
          _mode: "expanded",
          _expanded: true
        });
      } else if (toolName === "update_tasks") {
        try {
          const args = data.args ? typeof data.args === "string" ? JSON.parse(data.args) : data.args : {};
          if (Array.isArray(args.tasks) && globalCtx.onTaskReplace) {
            const tasks = args.tasks.map((t) => ({
              step: t.subject || t.description || "(无标题)",
              status: t.status || "pending",
              _taskId: t.id || null,
              planStepIndex: t.plan_step_index ?? null
            }));
            globalCtx.onTaskReplace(tasks, convId);
          }
        } catch {
        }
        msg.segments.push({
          type: "tool_call",
          name: toolName,
          callId: data.callId || data.callID || "",
          argsRaw: data.args ? typeof data.args === "string" ? data.args : JSON.stringify(data.args, null, 2) : "",
          result: "",
          _mode: "collapsed",
          _collapsed: false,
          _expanded: false
        });
      } else {
        msg.segments.push({
          type: "tool_call",
          name: toolName,
          callId: data.callId || data.callID || "",
          argsRaw: data.args ? typeof data.args === "string" ? data.args : JSON.stringify(data.args, null, 2) : "",
          result: "",
          _mode: "expanded",
          _expanded: true
        });
      }
    } else if (data.type === "tool_result") {
      const callId = data.callId || data.callID || "";
      const toolName = data.tool || data.name || "";
      const fileTools = ["write_file", "edit_file", "multi_edit", "delete_file", "move_file"];
      if (fileTools.includes(toolName)) {
        window.dispatchEvent(new CustomEvent("refresh-tree"));
      }
      if (toolName === "task_create" && globalCtx.onTaskSetId) {
        const idMatch = (data.content || "").match(/ID:\s*`([^`]+)`/);
        if (idMatch) globalCtx.onTaskSetId(callId, idMatch[1], convId);
      }
      if (!msg || !msg.segments) return;
      let target = null;
      for (let i = msg.segments.length - 1; i >= 0; i--) {
        const seg = msg.segments[i];
        if (seg.type === "tool_call") {
          if (callId && seg.callId === callId) {
            target = seg;
            break;
          }
          if (!target) target = seg;
        }
      }
      if (target) {
        target.result = data.content || "";
        target._expanded = false;
      }
    } else if (data.type === "approval") {
      let parsedArgs = {};
      try {
        parsedArgs = JSON.parse(data.args || "{}");
      } catch {
      }
      state.approvalByConv[convId] = {
        callId: data.callId || data.callID || "",
        tool: data.tool || "",
        args: data.args || "",
        parsedArgs,
        // 结构化后的参数
        waiting: true
      };
    } else if (data.type === "error") {
      const errText = (data.content || "").trim();
      const seg = pushSegment(msg.segments, "content");
      seg.content += "**[错误]** " + errText;
      seg.content += "\n\n> ⚠️ 本次任务未完成。可直接在下方输入继续（沿用本对话上下文），或点击对话列表中的该项恢复。";
      msg._loading = false;
      state.loadingByConv[convId] = false;
      state.agentRunningByConv[convId] = false;
      if (isCurrent) {
        state.chatLoading = false;
        state.agentRunning = false;
        if (globalCtx.onPhaseEnd) globalCtx.onPhaseEnd(convId);
      }
      const localConv = state.conversations.find((c) => c.id === convId);
      if (localConv) localConv.interrupted = true;
      window.dispatchEvent(new Event("save-conversations"));
      if (globalCtx.loadWsTokenStats) globalCtx.loadWsTokenStats();
      delete runtimes[convId];
      if (isCurrent) {
        state.messages = msgs;
        if (globalCtx.scrollToBottom) globalCtx.scrollToBottom(convId);
      }
      return;
    } else if (data.type === "usage" && data.usage) {
      const u = data.usage;
      if (isCurrent) {
        const cs = getConvCtxStats(convId);
        cs.promptTokens = u.prompt_tokens || 0;
        cs.completionTokens = u.completion_tokens || 0;
        cs.cacheHitTokens = u.prompt_cache_hit_tokens || 0;
        cs.cacheMissTokens = u.prompt_cache_miss_tokens || 0;
        if (u.prompt_breakdown) {
          const pb = u.prompt_breakdown;
          cs.systemTokens = pb.system_tokens || 0;
          cs.skillsTokens = pb.skills_tokens || 0;
          cs.mcpTokens = pb.mcp_tokens || 0;
          cs.toolTokens = pb.tool_tokens || 0;
          cs.historyTokens = pb.history_tokens || 0;
          cs.otherTokens = pb.other_tokens || 0;
        }
        const wsRoot = state.workspaceRoot;
        if (wsRoot) {
          if (!state.wsTokenStatsByWs[wsRoot]) {
            state.wsTokenStatsByWs[wsRoot] = { totalTokens: 0, promptTokens: 0, completionTokens: 0, cacheHitTokens: 0, cacheMissTokens: 0, systemTokens: 0, skillsTokens: 0, mcpTokens: 0, toolTokens: 0, historyTokens: 0, otherTokens: 0 };
          }
          const wsStats = state.wsTokenStatsByWs[wsRoot];
          wsStats.promptTokens += u.prompt_tokens || 0;
          wsStats.completionTokens += u.completion_tokens || 0;
          wsStats.totalTokens += (u.prompt_tokens || 0) + (u.completion_tokens || 0);
          wsStats.cacheHitTokens += u.prompt_cache_hit_tokens || 0;
          wsStats.cacheMissTokens += u.prompt_cache_miss_tokens || 0;
          if (u.prompt_breakdown) {
            wsStats.systemTokens += u.prompt_breakdown.system_tokens || 0;
            wsStats.skillsTokens += u.prompt_breakdown.skills_tokens || 0;
            wsStats.mcpTokens += u.prompt_breakdown.mcp_tokens || 0;
            wsStats.toolTokens += u.prompt_breakdown.tool_tokens || 0;
            wsStats.historyTokens += u.prompt_breakdown.history_tokens || 0;
            wsStats.otherTokens += u.prompt_breakdown.other_tokens || 0;
          }
        }
      }
    } else if (data.type === "phase") {
      state.phaseByConv[convId] = data.content || "";
      if (isCurrent && globalCtx.onPhaseChange) globalCtx.onPhaseChange(convId);
    } else if (data.type === "notice") {
      if (isCurrent) {
        state.nudgeByConv[convId] = (data.content || "").replace(/\n/g, " ").slice(0, 120);
        if (globalCtx.onNudge) globalCtx.onNudge(convId);
      }
    } else if (data.type === "compacted") {
      msg.segments.push({ type: "content", content: "> 📦 上下文已压缩（中段老消息已摘要）" });
    } else if (data.type === "circling") {
      msg.segments.push({ type: "content", content: "> ⚠️ 检测到重复操作，已提示 Agent 换思路" });
    } else if (data.type === "evaluation") {
      msg.segments.push({ type: "content", content: "> 📊 任务评测：\n" + (data.content || "") });
    }
    if (isCurrent) {
      state.messages = msgs;
    }
    if (isCurrent && globalCtx.scrollToBottom) globalCtx.scrollToBottom(convId);
  }
  function processAgentDone(convId, data) {
    console.log("[AE] processAgentDone conv=%s hasRuntime=%s msgByConvLen=%d", convId, !!runtimes[convId], (state.messagesByConv[convId] || []).length);
    const rt = runtimes[convId];
    const msgs = state.messagesByConv[convId];
    if (msgs && rt) {
      const msg = findMsgByKey(msgs, rt.msgKey);
      if (msg) {
        msg.content = rt.finalContent;
        const hasEffectiveSeg = (msg.segments || []).some((seg) => {
          if (seg.type === "tool_call" || seg.type === "ask_user") return true;
          if (seg.type === "content" && seg.content && seg.content.trim()) return true;
          if (seg.type === "thinking" && seg.content && seg.content.trim()) return true;
          return false;
        });
        const isEmptyPlaceholder = !rt.finalContent && !hasEffectiveSeg && (!data || data.doneReason !== "stopped");
        if (isEmptyPlaceholder) {
          console.log("[AE] processAgentDone 空占位 conv=%s msgKey=%s 设为完成提示", convId, rt.msgKey);
          msg.content = "**[操作完成]**";
          pushSegment(msg.segments, "content").content = "**[操作完成]**";
        }
        if (data && data.doneReason === "stopped") {
          if (!rt.finalContent) {
            msg.content = "**[任务已终止]** " + (data.content || "用户终止了任务");
          } else {
            pushSegment(msg.segments, "content").content += "\n\n**[任务已终止]** " + (data.content || "用户终止了任务");
          }
        }
      }
    }
    if (msgs) {
      for (const m of msgs) {
        if (m._loading) m._loading = false;
      }
      if (!rt) {
        const lastAssistant = [...msgs].reverse().find((m) => m.role === "assistant");
        if (lastAssistant && !lastAssistant.content && (!lastAssistant.segments || lastAssistant.segments.length === 0)) {
          if (data && data.content) {
            lastAssistant.content = data.content;
          } else if (data && data.doneReason === "stopped") {
            lastAssistant.content = "**[任务已终止]** " + (data.content || "用户终止了任务");
          } else {
            lastAssistant.content = "**[操作完成]**";
          }
          console.log("[AE] processAgentDone 兜底回填 content conv=%s", convId);
        }
      }
    }
    state.loadingByConv[convId] = false;
    state.agentRunningByConv[convId] = false;
    const isCurrent = state.currentConvId === convId;
    if (isCurrent) {
      state.chatLoading = false;
      state.agentRunning = false;
      if (globalCtx.onPhaseEnd) globalCtx.onPhaseEnd(convId);
    }
    if (globalCtx.loadWsTokenStats) globalCtx.loadWsTokenStats();
    if (rt && rt.finalContent && globalCtx.saveConvMsg) {
      const savedMsg = findMsgByKey(msgs, rt.msgKey);
      const savedIdx = savedMsg ? savedMsg._idx : -1;
      globalCtx.saveConvMsg(convId, rt.finalContent, savedIdx);
    }
    const localConv = state.conversations.find((c) => c.id === convId);
    if (localConv) localConv.msgCount = (localConv.msgCount || 0) + 1;
    window.dispatchEvent(new Event("save-conversations"));
    if (isCurrent) {
      state.messages = msgs;
    }
    if (isCurrent && globalCtx.scrollToBottom) globalCtx.scrollToBottom(convId);
    delete runtimes[convId];
  }
  function processAgentDisconnect(convId, errMsg) {
    const rt = runtimes[convId];
    const msgs = state.messagesByConv[convId];
    if (msgs && rt) {
      const msg = findMsgByKey(msgs, rt.msgKey);
      if (msg) {
        msg._loading = false;
        pushSegment(msg.segments, "content").content += "**[连接中断]** " + errMsg;
        pushSegment(msg.segments, "content").content += "\n\n> ⚠️ 本次任务未完成。后端恢复后可继续本对话（进度已保存）。";
      }
      const localConv = state.conversations.find((c) => c.id === convId);
      if (localConv) localConv.interrupted = true;
      window.dispatchEvent(new Event("save-conversations"));
    }
    const isCurrent = state.currentConvId === convId;
    if (isCurrent) {
      state.chatLoading = false;
      state.agentRunning = false;
    }
  }
  function processAllDisconnected() {
    for (const convId of Object.keys(state.agentRunningByConv)) {
      const rt = runtimes[convId];
      const msgs = state.messagesByConv[convId];
      if (msgs && rt) {
        const msg = findMsgByKey(msgs, rt.msgKey);
        if (msg) {
          msg._loading = false;
          if (!msg.content) msg.content = "";
          pushSegment(msg.segments, "content").content += "**[连接中断]** 后端进程已关闭，请重新发送消息。";
          pushSegment(msg.segments, "content").content += "\n\n> ⚠️ 本次任务未完成。重启后端后在本对话继续即可（进度已保存）。";
        } else {
          for (const m of msgs) {
            if (m._loading) m._loading = false;
          }
        }
      } else if (msgs) {
        for (const m of msgs) {
          if (m._loading) m._loading = false;
        }
      }
      const localConv = state.conversations.find((c) => c.id === convId);
      if (localConv) localConv.interrupted = true;
      state.agentRunningByConv[convId] = false;
      state.loadingByConv[convId] = false;
      delete runtimes[convId];
    }
    if (Object.keys(state.agentRunningByConv).length > 0) {
      window.dispatchEvent(new Event("save-conversations"));
    }
    state.chatLoading = false;
    state.agentRunning = false;
  }
  function processStatus(payload) {
    const p2 = Array.isArray(payload) ? { runningConvs: payload, runningByWorkspace: {} } : payload || {};
    const runningConvs = p2.runningConvs || [];
    const runningByWorkspace = p2.runningByWorkspace || {};
    state.runningByWorkspace = { ...runningByWorkspace };
    const runningSet = new Set(runningConvs);
    for (const convId of runningSet) {
      state.agentRunningByConv[convId] = true;
      state.loadingByConv[convId] = true;
      const msgsArr = state.messagesByConv[convId];
      if (msgsArr && msgsArr.length > 0 && !runtimes[convId]) {
        const hasRealMsgs = msgsArr.some((m) => !m._loading);
        const lastLoading = [...msgsArr].reverse().find((m) => m._loading);
        if (hasRealMsgs && !lastLoading) {
          const key = makeMsgKey();
          let psNextIdx = msgsArr.length;
          for (const m of msgsArr) {
            if ((m._idx ?? 0) >= psNextIdx) psNextIdx = (m._idx ?? 0) + 1;
          }
          console.log("[AE] processStatus 兜底创建占位 conv=%s key=%s", convId, key);
          msgsArr.push({
            role: "assistant",
            content: "",
            segments: [],
            toolCalls: [],
            _key: key,
            _idx: psNextIdx,
            _time: (/* @__PURE__ */ new Date()).toLocaleTimeString("zh-CN", { hour: "2-digit", minute: "2-digit" }),
            _loading: true
          });
          runtimes[convId] = { msgKey: key, finalContent: "", lastUserText: "" };
          if (state.currentConvId === convId) {
            state.messages = msgsArr;
          }
        }
      }
    }
    for (const convId of Object.keys(state.agentRunningByConv)) {
      if (state.agentRunningByConv[convId] && !runningSet.has(convId)) {
        state.agentRunningByConv[convId] = false;
        state.loadingByConv[convId] = false;
        if (state.currentConvId === convId) {
          state.chatLoading = false;
          state.agentRunning = false;
        }
        const msgsArr = state.messagesByConv[convId];
        if (msgsArr) {
          for (const m of msgsArr) {
            if (m._loading) m._loading = false;
          }
        }
        delete runtimes[convId];
      }
    }
    for (const convId of Object.keys(state.loadingByConv)) {
      if (state.loadingByConv[convId] && !runningSet.has(convId)) {
        state.loadingByConv[convId] = false;
        const msgsArr = state.messagesByConv[convId];
        if (msgsArr) {
          for (const m of msgsArr) {
            if (m._loading) m._loading = false;
          }
        }
      }
    }
  }
  function getConvCtxStats(convId) {
    if (!state.convCtxStatsByConv[convId]) {
      state.convCtxStatsByConv[convId] = /* @__PURE__ */ reactive({
        promptTokens: 0,
        completionTokens: 0,
        cacheHitTokens: 0,
        cacheMissTokens: 0,
        systemTokens: 0,
        skillsTokens: 0,
        mcpTokens: 0,
        toolTokens: 0,
        historyTokens: 0,
        otherTokens: 0
      });
    }
    return state.convCtxStatsByConv[convId];
  }
  function resetConvCtxStats(convId) {
    if (state.convCtxStatsByConv[convId]) {
      Object.assign(state.convCtxStatsByConv[convId], {
        promptTokens: 0,
        completionTokens: 0,
        cacheHitTokens: 0,
        cacheMissTokens: 0,
        systemTokens: 0,
        skillsTokens: 0,
        mcpTokens: 0,
        toolTokens: 0,
        historyTokens: 0,
        otherTokens: 0
      });
    }
  }
  const agentEvents = /* @__PURE__ */ Object.freeze(/* @__PURE__ */ Object.defineProperty({
    __proto__: null,
    createAssistantPlaceholder,
    getConvCtxStats,
    getConvRuntime,
    processAgentDisconnect,
    processAgentDone,
    processAgentEvent,
    processAllDisconnected,
    processStatus,
    resetConvCtxStats,
    resetConvRuntime,
    setGlobalCtx,
    startConvRuntime
  }, Symbol.toStringTag, { value: "Module" }));
  async function loadWsList() {
    var _a, _b, _c;
    if (!state.wsList) state.wsList = /* @__PURE__ */ reactive([]);
    const wsList = state.wsList;
    wsList.length = 0;
    let loadedItems = [];
    try {
      const resp = await api.apiGet("/settings");
      const settings = resp.settings || resp;
      const projects = settings.recentProjects || [];
      const folderLists = settings.workspaceFolderLists || {};
      const seen = /* @__PURE__ */ new Set();
      for (const p2 of projects) {
        if (!p2 || seen.has(p2)) continue;
        seen.add(p2);
        const folders = ((_a = folderLists[p2]) == null ? void 0 : _a.length) > 0 ? [...folderLists[p2]] : [p2];
        loadedItems.push(/* @__PURE__ */ reactive({
          path: p2,
          name: p2.split(/[\\/]/).filter(Boolean).pop() || p2,
          folders: p2 === state.workspaceRoot && ((_b = state.workspaceFolders) == null ? void 0 : _b.length) > 0 ? [...state.workspaceFolders] : folders,
          notify: false
        }));
      }
    } catch {
    }
    if (state.workspaceRoot && !loadedItems.find((w) => w.path === state.workspaceRoot)) {
      loadedItems.push(/* @__PURE__ */ reactive({
        path: state.workspaceRoot,
        name: state.workspaceRoot.split(/[\\/]/).filter(Boolean).pop() || state.workspaceRoot,
        folders: ((_c = state.workspaceFolders) == null ? void 0 : _c.length) > 0 ? [...state.workspaceFolders] : [state.workspaceRoot],
        notify: false
      }));
    }
    state.wsList = loadedItems;
  }
  async function saveWsList() {
    var _a;
    const wsList = state.wsList || [];
    try {
      const resp = await api.apiGet("/settings");
      const settings = resp.settings || resp;
      settings.recentProjects = wsList.slice(0, 20).map((w) => w.path).filter(Boolean);
      settings.workspaceFolderLists = settings.workspaceFolderLists || {};
      for (const ws of wsList) {
        if (((_a = ws.folders) == null ? void 0 : _a.length) > 0) {
          settings.workspaceFolderLists[ws.path] = [...ws.folders];
        }
      }
      await api.apiPut("/settings", settings);
    } catch {
    }
  }
  function checkNotifications() {
    const wsList = state.wsList || [];
    for (const ws of wsList) {
      ws.notify = state.notificationCount > 0 && ws.path !== state.workspaceRoot;
    }
  }
  async function switchWorkspace(targetPath) {
    if (!targetPath || targetPath === state.workspaceRoot) return;
    const wsList = state.wsList || [];
    try {
      const targetWs = wsList.find((w) => w.path === targetPath);
      const folders = (targetWs == null ? void 0 : targetWs.folders) || [];
      await api.apiPost("/workspace", {
        action: "switch",
        root: targetPath,
        folders: folders.filter((f) => f !== targetPath)
      });
      state.workspaceRoot = targetPath;
      state.workspaceFolders = folders.length > 0 ? [...folders] : [targetPath];
      state.settings.workspaceFolders = [...state.workspaceFolders];
      state.workspaceName = targetPath.split(/[\\/]/).filter(Boolean).pop() || targetPath;
      document.title = "PairCode IDE - " + state.workspaceName;
      state.openFiles = [];
      state.activeFile = "";
      state.fileContents = {};
      await loadFileTree();
      try {
        const list = await api.apiGet("/conversations", { workspace: targetPath });
        state.conversations = list || [];
      } catch {
      }
      window.dispatchEvent(new CustomEvent("workspace-switched"));
      const ws = wsList.find((w) => w.path === targetPath);
      if (ws) ws.notify = false;
      state.notificationCount = 0;
      if (targetWs) {
        targetWs.folders = [...state.workspaceFolders];
      }
      if (!wsList.find((w) => w.path === targetPath)) {
        wsList.push(/* @__PURE__ */ reactive({ path: targetPath, name: state.workspaceName, folders: [...state.workspaceFolders], notify: false }));
      }
      await saveWsList();
      savePersistentState();
      api.reconnectWebSocket();
    } catch (err) {
      console.error("切换工作区失败:", err);
    }
  }
  async function loadConversationsForWorkspace(path) {
    state.conversations = [];
    state.currentConvId = "";
    state.messages = [];
    if (typeof path !== "string" || !path) return;
    try {
      const list = await api.apiGet("/conversations", { workspace: path });
      state.conversations = list || [];
    } catch (e) {
      console.warn("从后端加载对话消息失败:", e);
    }
  }
  function switchActivity(id) {
    if (id === "settings") {
      showSettings.value = true;
      return;
    }
    if (id === "system") {
      showSystem.value = true;
      return;
    }
    if (id === "chat") {
      state.rightPanelVisible = !state.rightPanelVisible;
      return;
    }
    if (id === "marketplace") {
      showMarketplace.value = true;
      return;
    }
    if (state.activeActivity === id) {
      state.sidebarVisible = !state.sidebarVisible;
    } else {
      state.activeActivity = id;
      state.sidebarVisible = true;
    }
  }
  const loadFileTree = async () => {
    const dirs = state.workspaceFolders.length > 0 ? [...state.workspaceFolders] : [];
    if (dirs.length === 0 && state.workspaceRoot) dirs.push(state.workspaceRoot);
    const seen = /* @__PURE__ */ new Set();
    const unique = dirs.filter((d) => {
      if (seen.has(d) || !d) return false;
      seen.add(d);
      return true;
    });
    state.fileTree = [];
    for (const d of unique) {
      if (!d) continue;
      try {
        const entries = await api.apiGet("/fs/list", { path: d });
        state.fileTree.push({ path: d, name: d.split("\\").filter(Boolean).pop() || d, children: entries || [], loaded: false });
      } catch {
      }
    }
  };
  function handleKeydown(e) {
    if (e.target.tagName === "INPUT" || e.target.tagName === "TEXTAREA") return;
    if (e.ctrlKey && e.key === "b") {
      e.preventDefault();
      state.sidebarVisible = !state.sidebarVisible;
    }
    if (e.ctrlKey && e.key === "`") {
      e.preventDefault();
      state.bottomPanelVisible = !state.bottomPanelVisible;
    }
    if (e.ctrlKey && e.shiftKey && e.key === "E") {
      e.preventDefault();
      state.activeActivity = "explorer";
      state.sidebarVisible = true;
    }
    if (e.ctrlKey && e.shiftKey && e.key === "F") {
      e.preventDefault();
      state.activeActivity = "search";
      state.sidebarVisible = true;
    }
    if (e.ctrlKey && e.shiftKey && e.key === "T") {
      e.preventDefault();
      state.rightPanelVisible = true;
    }
    if (e.ctrlKey && e.shiftKey && e.key === "C") {
      e.preventDefault();
      state.rightPanelVisible = !state.rightPanelVisible;
    }
    if (e.ctrlKey && e.key === "k") {
      e.preventDefault();
      state.focusMode = !state.focusMode;
    }
  }
  let _lastRefreshTime = 0;
  const MIN_TREE_REFRESH_INTERVAL = 3e3;
  let _savedTreeScrollTop = 0;
  function refreshTree() {
    const now = Date.now();
    if (now - _lastRefreshTime < MIN_TREE_REFRESH_INTERVAL) return;
    _lastRefreshTime = now;
    const scrollEl = document.querySelector(".project-section");
    if (scrollEl) _savedTreeScrollTop = scrollEl.scrollTop;
    for (const path of Object.keys(state.fileContents)) {
      if (state.openFiles.includes(path)) {
        if (!state.fileDirty[path]) {
          api.apiGet("/fs/read", { path }).then((data) => {
            const normalized = (data.content || "").replace(/\r\n/g, "\n");
            state.fileContents[path] = normalized;
            state.fileSavedContent[path] = normalized;
            state.fileDirty[path] = false;
          }).catch(() => {
          });
        }
      } else {
        delete state.fileContents[path];
        delete state.fileSavedContent[path];
        delete state.fileDirty[path];
      }
    }
    loadFileTree().then(() => {
      if (_savedTreeScrollTop > 0) {
        nextTick(() => {
          const c = document.querySelector(".project-section");
          if (c) c.scrollTop = _savedTreeScrollTop;
        });
      }
    });
  }
  let cleanupFns = [];
  let refreshTimer = null;
  function initAppGlobals() {
    loadPersistentState();
    api.initWebSocket({
      onStatus: (payload) => processStatus(payload),
      onEvent: (convId, data) => processAgentEvent(convId, data),
      onDone: (convId, data) => processAgentDone(convId, data),
      onDisconnected: () => processAllDisconnected()
    });
    (async () => {
      try {
        const sresp = await api.apiGet("/settings");
        if (sresp && sresp.settings) {
          state.settings = sresp.settings;
          state.settingsLoaded = true;
          state.pluginSchemas = sresp.schemas || [];
        }
      } catch {
      }
      try {
        const health = await api.apiGet("/health");
        if (health && health.workspace) {
          state.workspaceRoot = health.workspace;
          state.workspaceFolders = health.folders || [];
          state.workspaceName = health.workspace.split("\\").filter(Boolean).pop() || health.workspace;
          document.title = "PairCode IDE - " + state.workspaceName;
          await loadConversationsForWorkspace(health.workspace);
        }
      } catch {
      }
      await loadFileTree();
    })();
    document.addEventListener("keydown", handleKeydown);
    const handlers = {
      "refresh-tree": refreshTree,
      "switch-activity": (e) => {
        var _a;
        if ((_a = e.detail) == null ? void 0 : _a.id) switchActivity(e.detail.id);
      },
      "open-marketplace": () => {
        showMarketplace.value = true;
      },
      "open-settings": () => {
        showSettings.value = true;
      },
      "stop-agent": () => {
        window.dispatchEvent(new CustomEvent("agent-stop"));
      },
      "save-conversations": async () => {
        checkNotifications();
      },
      "open-workspace-dialog": () => {
        state.activeActivity = "explorer";
        state.sidebarVisible = true;
      },
      "switch-workspace": async (e) => {
        var _a;
        if ((_a = e.detail) == null ? void 0 : _a.path) await switchWorkspace(e.detail.path);
      }
    };
    for (const [ev, fn] of Object.entries(handlers)) window.addEventListener(ev, fn);
    refreshTimer = setInterval(() => {
      if (document.visibilityState === "visible") {
        _lastRefreshTime = 0;
        window.dispatchEvent(new CustomEvent("refresh-tree"));
      }
    }, 5e3);
    cleanupFns = [
      () => document.removeEventListener("keydown", handleKeydown),
      () => {
        for (const [ev, fn] of Object.entries(handlers)) window.removeEventListener(ev, fn);
      },
      () => {
        if (refreshTimer) {
          clearInterval(refreshTimer);
          refreshTimer = null;
        }
      }
    ];
  }
  function cleanupAppGlobals() {
    for (const fn of cleanupFns) {
      try {
        fn();
      } catch {
      }
    }
    cleanupFns = [];
    api.closeWebSocket();
  }
  function desktopPrefetch() {
    try {
      if (typeof go !== "undefined" && go.bridge_call && typeof window !== "undefined" && window.__DESKTOP_MODE__) {
        const _r = go.bridge_call("GET", "/api/settings", "", "");
        const _parsed = JSON.parse(_r);
        const _settings = _parsed.body ? JSON.parse(_parsed.body).settings : {};
        const _projects = _settings && _settings.recentProjects || [];
        const _folders = _settings && _settings.workspaceFolderLists || {};
        const _seen = /* @__PURE__ */ new Set();
        const _items = [];
        for (const _p of _projects) {
          if (!_p || _seen.has(_p)) continue;
          _seen.add(_p);
          const _fl = _folders[_p] && _folders[_p].length > 0 ? [..._folders[_p]] : [_p];
          _items.push(/* @__PURE__ */ reactive({ path: _p, name: _p.split(/[\\/]/).filter(Boolean).pop() || _p, folders: _fl, notify: false }));
        }
        if (_items.length > 0) state.wsList = _items;
        try {
          const _h = JSON.parse(go.bridge_call("GET", "/api/health", "", ""));
          const _health = JSON.parse(_h.body || "{}");
          if (_health.workspace) {
            state.workspaceRoot = _health.workspace;
            state.workspaceFolders = _health.folders || [];
            state.workspaceName = _health.workspace.split("\\").filter(Boolean).pop() || _health.workspace;
            const _c = JSON.parse(go.bridge_call("GET", "/api/conversations?workspace=" + encodeURIComponent(_health.workspace), "", ""));
            const _list = JSON.parse(_c.body || "[]");
            if (Array.isArray(_list) && _list.length > 0) state.conversations = _list;
          }
        } catch {
        }
      }
    } catch {
    }
  }
  const appActions = {
    loadWsList,
    saveWsList,
    checkNotifications,
    switchWorkspace,
    loadConversationsForWorkspace,
    switchActivity,
    loadFileTree,
    handleKeydown,
    initAppGlobals,
    cleanupAppGlobals,
    desktopPrefetch
  };
  const actions = /* @__PURE__ */ Object.freeze(/* @__PURE__ */ Object.defineProperty({
    __proto__: null,
    checkNotifications,
    cleanupAppGlobals,
    default: appActions,
    desktopPrefetch,
    handleKeydown,
    initAppGlobals,
    loadConversationsForWorkspace,
    loadFileTree,
    loadWsList,
    saveWsList,
    switchActivity,
    switchWorkspace
  }, Symbol.toStringTag, { value: "Module" }));
  const _export_sfc = (sfc, props) => {
    const target = sfc.__vccOpts || sfc;
    for (const [key, val] of props) {
      target[key] = val;
    }
    return target;
  };
  const _hoisted_1$2 = ["width", "height"];
  const _hoisted_2$2 = {
    key: 0,
    d: "M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"
  };
  const _hoisted_3$2 = {
    key: 7,
    d: "M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"
  };
  const _hoisted_4$2 = {
    key: 10,
    points: "9 6 15 12 9 18"
  };
  const _hoisted_5$2 = {
    key: 11,
    points: "6 9 12 15 18 9"
  };
  const _hoisted_6$2 = {
    key: 30,
    d: "M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.77-3.77a6 6 0 0 1-7.94 7.94l-6.91 6.91a2.12 2.12 0 0 1-3-3l6.91-6.91a6 6 0 0 1 7.94-7.94l-3.76 3.76z"
  };
  const _hoisted_7$2 = {
    key: 48,
    x1: "5",
    y1: "12",
    x2: "19",
    y2: "12"
  };
  const _hoisted_8$2 = {
    key: 52,
    d: "M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"
  };
  const _hoisted_9$2 = {
    key: 55,
    points: "20 6 9 17 4 12"
  };
  const _hoisted_10$1 = {
    key: 58,
    d: "M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"
  };
  const _hoisted_11$1 = {
    key: 68,
    d: "M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.77-3.77a6 6 0 0 1-7.94 7.94l-6.91 6.91a2.12 2.12 0 0 1-3-3l6.91-6.91a6 6 0 0 1 7.94-7.94l-3.76 3.76z"
  };
  const _hoisted_12$1 = {
    key: 70,
    points: "15 6 9 12 15 18"
  };
  const _sfc_main$2 = {
    __name: "SvgIcon",
    props: {
      name: { type: String, required: true },
      size: { type: Number, default: 16 }
    },
    setup(__props) {
      return (_ctx, _cache) => {
        return openBlock(), createElementBlock("svg", {
          class: "svg-icon",
          width: __props.size,
          height: __props.size,
          viewBox: "0 0 24 24",
          fill: "none",
          stroke: "currentColor",
          "stroke-width": "2",
          "stroke-linecap": "round",
          "stroke-linejoin": "round"
        }, [
          __props.name === "folder" ? (openBlock(), createElementBlock("path", _hoisted_2$2)) : __props.name === "folder-open" ? (openBlock(), createElementBlock(Fragment, { key: 1 }, [
            _cache[0] || (_cache[0] = createBaseVNode("path", { d: "M6 17l-3-9h18l-3 9H6z" }, null, -1)),
            _cache[1] || (_cache[1] = createBaseVNode("path", { d: "M4 8V5a2 2 0 0 1 2-2h5l2 3h7a2 2 0 0 1 2 2v3" }, null, -1))
          ], 64)) : __props.name === "file" ? (openBlock(), createElementBlock(Fragment, { key: 2 }, [
            _cache[2] || (_cache[2] = createBaseVNode("path", { d: "M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" }, null, -1)),
            _cache[3] || (_cache[3] = createBaseVNode("polyline", { points: "14 2 14 8 20 8" }, null, -1))
          ], 64)) : __props.name === "file-code" ? (openBlock(), createElementBlock(Fragment, { key: 3 }, [
            _cache[4] || (_cache[4] = createStaticVNode('<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" data-v-faf69761></path><polyline points="14 2 14 8 20 8" data-v-faf69761></polyline><line x1="10" y1="12" x2="8" y2="14" data-v-faf69761></line><line x1="10" y1="16" x2="8" y2="18" data-v-faf69761></line><line x1="14" y1="12" x2="16" y2="14" data-v-faf69761></line><line x1="14" y1="16" x2="16" y2="18" data-v-faf69761></line>', 6))
          ], 64)) : __props.name === "file-text" ? (openBlock(), createElementBlock(Fragment, { key: 4 }, [
            _cache[5] || (_cache[5] = createBaseVNode("path", { d: "M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" }, null, -1)),
            _cache[6] || (_cache[6] = createBaseVNode("polyline", { points: "14 2 14 8 20 8" }, null, -1)),
            _cache[7] || (_cache[7] = createBaseVNode("line", {
              x1: "9",
              y1: "13",
              x2: "15",
              y2: "13"
            }, null, -1)),
            _cache[8] || (_cache[8] = createBaseVNode("line", {
              x1: "9",
              y1: "17",
              x2: "15",
              y2: "17"
            }, null, -1))
          ], 64)) : __props.name === "search" ? (openBlock(), createElementBlock(Fragment, { key: 5 }, [
            _cache[9] || (_cache[9] = createBaseVNode("circle", {
              cx: "11",
              cy: "11",
              r: "8"
            }, null, -1)),
            _cache[10] || (_cache[10] = createBaseVNode("line", {
              x1: "21",
              y1: "21",
              x2: "16.65",
              y2: "16.65"
            }, null, -1))
          ], 64)) : __props.name === "terminal" ? (openBlock(), createElementBlock(Fragment, { key: 6 }, [
            _cache[11] || (_cache[11] = createBaseVNode("polyline", { points: "4 17 10 11 4 5" }, null, -1)),
            _cache[12] || (_cache[12] = createBaseVNode("line", {
              x1: "12",
              y1: "19",
              x2: "20",
              y2: "19"
            }, null, -1))
          ], 64)) : __props.name === "chat" ? (openBlock(), createElementBlock("path", _hoisted_3$2)) : __props.name === "settings" ? (openBlock(), createElementBlock(Fragment, { key: 8 }, [
            _cache[13] || (_cache[13] = createBaseVNode("circle", {
              cx: "12",
              cy: "12",
              r: "3"
            }, null, -1)),
            _cache[14] || (_cache[14] = createBaseVNode("path", { d: "M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z" }, null, -1))
          ], 64)) : __props.name === "home" ? (openBlock(), createElementBlock(Fragment, { key: 9 }, [
            _cache[15] || (_cache[15] = createBaseVNode("path", { d: "M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z" }, null, -1)),
            _cache[16] || (_cache[16] = createBaseVNode("polyline", { points: "9 22 9 12 15 12 15 22" }, null, -1))
          ], 64)) : __props.name === "chevron-right" ? (openBlock(), createElementBlock("polyline", _hoisted_4$2)) : __props.name === "chevron-down" ? (openBlock(), createElementBlock("polyline", _hoisted_5$2)) : __props.name === "plus" ? (openBlock(), createElementBlock(Fragment, { key: 12 }, [
            _cache[17] || (_cache[17] = createBaseVNode("line", {
              x1: "12",
              y1: "5",
              x2: "12",
              y2: "19"
            }, null, -1)),
            _cache[18] || (_cache[18] = createBaseVNode("line", {
              x1: "5",
              y1: "12",
              x2: "19",
              y2: "12"
            }, null, -1))
          ], 64)) : __props.name === "close" ? (openBlock(), createElementBlock(Fragment, { key: 13 }, [
            _cache[19] || (_cache[19] = createBaseVNode("line", {
              x1: "18",
              y1: "6",
              x2: "6",
              y2: "18"
            }, null, -1)),
            _cache[20] || (_cache[20] = createBaseVNode("line", {
              x1: "6",
              y1: "6",
              x2: "18",
              y2: "18"
            }, null, -1))
          ], 64)) : __props.name === "refresh" ? (openBlock(), createElementBlock(Fragment, { key: 14 }, [
            _cache[21] || (_cache[21] = createBaseVNode("polyline", { points: "23 4 23 10 17 10" }, null, -1)),
            _cache[22] || (_cache[22] = createBaseVNode("path", { d: "M20.49 15a9 9 0 1 1-2.12-9.36L23 10" }, null, -1))
          ], 64)) : __props.name === "drive" ? (openBlock(), createElementBlock(Fragment, { key: 15 }, [
            _cache[23] || (_cache[23] = createBaseVNode("line", {
              x1: "22",
              y1: "12",
              x2: "2",
              y2: "12"
            }, null, -1)),
            _cache[24] || (_cache[24] = createBaseVNode("path", { d: "M5.45 5.11L2 12v6a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2v-6l-3.45-6.89A2 2 0 0 0 16.76 4H7.24a2 2 0 0 0-1.79 1.11z" }, null, -1)),
            _cache[25] || (_cache[25] = createBaseVNode("line", {
              x1: "6",
              y1: "16",
              x2: "6.01",
              y2: "16"
            }, null, -1)),
            _cache[26] || (_cache[26] = createBaseVNode("line", {
              x1: "10",
              y1: "16",
              x2: "10.01",
              y2: "16"
            }, null, -1))
          ], 64)) : __props.name === "source-control" ? (openBlock(), createElementBlock(Fragment, { key: 16 }, [
            _cache[27] || (_cache[27] = createBaseVNode("line", {
              x1: "6",
              y1: "3",
              x2: "6",
              y2: "15"
            }, null, -1)),
            _cache[28] || (_cache[28] = createBaseVNode("circle", {
              cx: "18",
              cy: "6",
              r: "3"
            }, null, -1)),
            _cache[29] || (_cache[29] = createBaseVNode("circle", {
              cx: "6",
              cy: "18",
              r: "3"
            }, null, -1)),
            _cache[30] || (_cache[30] = createBaseVNode("path", { d: "M18 9a9 9 0 0 1-9 9" }, null, -1))
          ], 64)) : __props.name === "git-branch" ? (openBlock(), createElementBlock(Fragment, { key: 17 }, [
            _cache[31] || (_cache[31] = createBaseVNode("line", {
              x1: "6",
              y1: "3",
              x2: "6",
              y2: "15"
            }, null, -1)),
            _cache[32] || (_cache[32] = createBaseVNode("circle", {
              cx: "18",
              cy: "6",
              r: "3"
            }, null, -1)),
            _cache[33] || (_cache[33] = createBaseVNode("circle", {
              cx: "6",
              cy: "18",
              r: "3"
            }, null, -1)),
            _cache[34] || (_cache[34] = createBaseVNode("path", { d: "M18 9a9 9 0 0 1-9 9" }, null, -1))
          ], 64)) : __props.name === "git-pull" ? (openBlock(), createElementBlock(Fragment, { key: 18 }, [
            _cache[35] || (_cache[35] = createStaticVNode('<circle cx="18" cy="18" r="3" data-v-faf69761></circle><circle cx="6" cy="6" r="3" data-v-faf69761></circle><path d="M13 6h3a2 2 0 0 1 2 2v7" data-v-faf69761></path><line x1="6" y1="18" x2="6" y2="9" data-v-faf69761></line><polyline points="9 9 6 6 3 9" data-v-faf69761></polyline>', 5))
          ], 64)) : __props.name === "git-push" ? (openBlock(), createElementBlock(Fragment, { key: 19 }, [
            _cache[36] || (_cache[36] = createStaticVNode('<circle cx="18" cy="6" r="3" data-v-faf69761></circle><circle cx="6" cy="18" r="3" data-v-faf69761></circle><path d="M13 18h-2a2 2 0 0 1-2-2V9" data-v-faf69761></path><line x1="6" y1="6" x2="6" y2="15" data-v-faf69761></line><polyline points="9 15 6 18 3 15" data-v-faf69761></polyline>', 5))
          ], 64)) : __props.name === "output" ? (openBlock(), createElementBlock(Fragment, { key: 20 }, [
            _cache[37] || (_cache[37] = createBaseVNode("rect", {
              x: "2",
              y: "3",
              width: "20",
              height: "14",
              rx: "2",
              ry: "2"
            }, null, -1)),
            _cache[38] || (_cache[38] = createBaseVNode("line", {
              x1: "8",
              y1: "21",
              x2: "16",
              y2: "21"
            }, null, -1)),
            _cache[39] || (_cache[39] = createBaseVNode("line", {
              x1: "12",
              y1: "17",
              x2: "12",
              y2: "21"
            }, null, -1))
          ], 64)) : __props.name === "warning" ? (openBlock(), createElementBlock(Fragment, { key: 21 }, [
            _cache[40] || (_cache[40] = createBaseVNode("path", { d: "M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z" }, null, -1)),
            _cache[41] || (_cache[41] = createBaseVNode("line", {
              x1: "12",
              y1: "9",
              x2: "12",
              y2: "13"
            }, null, -1)),
            _cache[42] || (_cache[42] = createBaseVNode("line", {
              x1: "12",
              y1: "17",
              x2: "12.01",
              y2: "17"
            }, null, -1))
          ], 64)) : __props.name === "undo" ? (openBlock(), createElementBlock(Fragment, { key: 22 }, [
            _cache[43] || (_cache[43] = createBaseVNode("polyline", { points: "1 4 1 10 7 10" }, null, -1)),
            _cache[44] || (_cache[44] = createBaseVNode("path", { d: "M3.51 15a9 9 0 1 0 2.13-9.36L1 10" }, null, -1))
          ], 64)) : __props.name === "redo" ? (openBlock(), createElementBlock(Fragment, { key: 23 }, [
            _cache[45] || (_cache[45] = createBaseVNode("polyline", { points: "23 4 23 10 17 10" }, null, -1)),
            _cache[46] || (_cache[46] = createBaseVNode("path", { d: "M20.49 15a9 9 0 1 1-2.12-9.36L23 10" }, null, -1))
          ], 64)) : __props.name === "package" ? (openBlock(), createElementBlock(Fragment, { key: 24 }, [
            _cache[47] || (_cache[47] = createBaseVNode("line", {
              x1: "16.5",
              y1: "9.4",
              x2: "7.5",
              y2: "4.21"
            }, null, -1)),
            _cache[48] || (_cache[48] = createBaseVNode("path", { d: "M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z" }, null, -1)),
            _cache[49] || (_cache[49] = createBaseVNode("polyline", { points: "3.27 6.96 12 12.01 20.73 6.96" }, null, -1)),
            _cache[50] || (_cache[50] = createBaseVNode("line", {
              x1: "12",
              y1: "22.08",
              x2: "12",
              y2: "12"
            }, null, -1))
          ], 64)) : __props.name === "globe" ? (openBlock(), createElementBlock(Fragment, { key: 25 }, [
            _cache[51] || (_cache[51] = createBaseVNode("circle", {
              cx: "12",
              cy: "12",
              r: "10"
            }, null, -1)),
            _cache[52] || (_cache[52] = createBaseVNode("line", {
              x1: "2",
              y1: "12",
              x2: "22",
              y2: "12"
            }, null, -1)),
            _cache[53] || (_cache[53] = createBaseVNode("path", { d: "M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z" }, null, -1))
          ], 64)) : __props.name === "cycle" ? (openBlock(), createElementBlock(Fragment, { key: 26 }, [
            _cache[54] || (_cache[54] = createBaseVNode("polyline", { points: "23 4 23 10 17 10" }, null, -1)),
            _cache[55] || (_cache[55] = createBaseVNode("polyline", { points: "1 20 1 14 7 14" }, null, -1)),
            _cache[56] || (_cache[56] = createBaseVNode("path", { d: "M3.51 9a9 9 0 0 1 14.85-3.36L23 10" }, null, -1)),
            _cache[57] || (_cache[57] = createBaseVNode("path", { d: "M20.49 15a9 9 0 0 1-14.85 3.36L1 14" }, null, -1))
          ], 64)) : __props.name === "send" ? (openBlock(), createElementBlock(Fragment, { key: 27 }, [
            _cache[58] || (_cache[58] = createBaseVNode("line", {
              x1: "12",
              y1: "19",
              x2: "12",
              y2: "5"
            }, null, -1)),
            _cache[59] || (_cache[59] = createBaseVNode("polyline", { points: "5 12 12 5 19 12" }, null, -1))
          ], 64)) : __props.name === "send-plane" ? (openBlock(), createElementBlock(Fragment, { key: 28 }, [
            _cache[60] || (_cache[60] = createBaseVNode("line", {
              x1: "22",
              y1: "2",
              x2: "11",
              y2: "13"
            }, null, -1)),
            _cache[61] || (_cache[61] = createBaseVNode("polygon", { points: "22 2 15 22 11 13 2 9 22 2" }, null, -1))
          ], 64)) : __props.name === "stop-dot" ? (openBlock(), createElementBlock(Fragment, { key: 29 }, [
            _cache[62] || (_cache[62] = createBaseVNode("circle", {
              cx: "12",
              cy: "12",
              r: "6",
              class: "stop-pulse"
            }, null, -1)),
            _cache[63] || (_cache[63] = createBaseVNode("circle", {
              cx: "12",
              cy: "12",
              r: "10",
              class: "stop-pulse-ring"
            }, null, -1))
          ], 64)) : __props.name === "wrench" ? (openBlock(), createElementBlock("path", _hoisted_6$2)) : __props.name === "database" ? (openBlock(), createElementBlock(Fragment, { key: 31 }, [
            _cache[64] || (_cache[64] = createBaseVNode("ellipse", {
              cx: "12",
              cy: "5",
              rx: "9",
              ry: "3"
            }, null, -1)),
            _cache[65] || (_cache[65] = createBaseVNode("path", { d: "M21 12c0 1.66-4 3-9 3s-9-1.34-9-3" }, null, -1)),
            _cache[66] || (_cache[66] = createBaseVNode("path", { d: "M3 5v14c0 1.66 4 3 9 3s9-1.34 9-3V5" }, null, -1))
          ], 64)) : __props.name === "user" ? (openBlock(), createElementBlock(Fragment, { key: 32 }, [
            _cache[67] || (_cache[67] = createBaseVNode("path", { d: "M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2" }, null, -1)),
            _cache[68] || (_cache[68] = createBaseVNode("circle", {
              cx: "12",
              cy: "7",
              r: "4"
            }, null, -1))
          ], 64)) : __props.name === "info" ? (openBlock(), createElementBlock(Fragment, { key: 33 }, [
            _cache[69] || (_cache[69] = createBaseVNode("circle", {
              cx: "12",
              cy: "12",
              r: "10"
            }, null, -1)),
            _cache[70] || (_cache[70] = createBaseVNode("line", {
              x1: "12",
              y1: "16",
              x2: "12",
              y2: "12"
            }, null, -1)),
            _cache[71] || (_cache[71] = createBaseVNode("line", {
              x1: "12",
              y1: "8",
              x2: "12.01",
              y2: "8"
            }, null, -1))
          ], 64)) : __props.name === "lightbulb" ? (openBlock(), createElementBlock(Fragment, { key: 34 }, [
            _cache[72] || (_cache[72] = createBaseVNode("path", { d: "M9 18h6" }, null, -1)),
            _cache[73] || (_cache[73] = createBaseVNode("path", { d: "M10 22h4" }, null, -1)),
            _cache[74] || (_cache[74] = createBaseVNode("path", { d: "M15.09 14c.18-.98.65-1.74 1.41-2.5A4.65 4.65 0 0 0 18 8 6 6 0 0 0 6 8c0 1 .23 2.23 1.5 3.5A4.61 4.61 0 0 1 8.91 14" }, null, -1))
          ], 64)) : __props.name === "sparkles" ? (openBlock(), createElementBlock(Fragment, { key: 35 }, [
            _cache[75] || (_cache[75] = createStaticVNode('<path d="M13.5 4L15 8l4 .5L15 12l1.5 4-4-2-4 2L10 12l-4-3.5L10 8z" data-v-faf69761></path><line x1="3" y1="18" x2="3" y2="21" data-v-faf69761></line><line x1="21" y1="18" x2="21" y2="21" data-v-faf69761></line><line x1="7" y1="20" x2="11" y2="20" data-v-faf69761></line><line x1="17" y1="20" x2="19" y2="20" data-v-faf69761></line>', 5))
          ], 64)) : __props.name === "bot" ? (openBlock(), createElementBlock(Fragment, { key: 36 }, [
            _cache[76] || (_cache[76] = createStaticVNode('<rect x="3" y="11" width="18" height="10" rx="2" data-v-faf69761></rect><circle cx="12" cy="5" r="2" data-v-faf69761></circle><path d="M12 7v4" data-v-faf69761></path><line x1="8" y1="16" x2="8" y2="16" data-v-faf69761></line><line x1="16" y1="16" x2="16" y2="16" data-v-faf69761></line>', 5))
          ], 64)) : __props.name === "file-js" ? (openBlock(), createElementBlock(Fragment, { key: 37 }, [
            _cache[77] || (_cache[77] = createBaseVNode("path", { d: "M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" }, null, -1)),
            _cache[78] || (_cache[78] = createBaseVNode("polyline", { points: "14 2 14 8 20 8" }, null, -1)),
            _cache[79] || (_cache[79] = createBaseVNode("text", {
              x: "8",
              y: "17",
              "font-size": "9",
              fill: "currentColor",
              "font-weight": "bold",
              stroke: "none"
            }, "JS", -1))
          ], 64)) : __props.name === "file-ts" ? (openBlock(), createElementBlock(Fragment, { key: 38 }, [
            _cache[80] || (_cache[80] = createBaseVNode("path", { d: "M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" }, null, -1)),
            _cache[81] || (_cache[81] = createBaseVNode("polyline", { points: "14 2 14 8 20 8" }, null, -1)),
            _cache[82] || (_cache[82] = createBaseVNode("text", {
              x: "8",
              y: "17",
              "font-size": "9",
              fill: "currentColor",
              "font-weight": "bold",
              stroke: "none"
            }, "TS", -1))
          ], 64)) : __props.name === "file-go" ? (openBlock(), createElementBlock(Fragment, { key: 39 }, [
            _cache[83] || (_cache[83] = createBaseVNode("path", { d: "M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" }, null, -1)),
            _cache[84] || (_cache[84] = createBaseVNode("polyline", { points: "14 2 14 8 20 8" }, null, -1)),
            _cache[85] || (_cache[85] = createBaseVNode("text", {
              x: "9",
              y: "17",
              "font-size": "9",
              fill: "currentColor",
              "font-weight": "bold",
              stroke: "none"
            }, "Go", -1))
          ], 64)) : __props.name === "file-py" ? (openBlock(), createElementBlock(Fragment, { key: 40 }, [
            _cache[86] || (_cache[86] = createBaseVNode("path", { d: "M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" }, null, -1)),
            _cache[87] || (_cache[87] = createBaseVNode("polyline", { points: "14 2 14 8 20 8" }, null, -1)),
            _cache[88] || (_cache[88] = createBaseVNode("text", {
              x: "7",
              y: "17",
              "font-size": "9",
              fill: "currentColor",
              "font-weight": "bold",
              stroke: "none"
            }, "Py", -1))
          ], 64)) : __props.name === "file-java" ? (openBlock(), createElementBlock(Fragment, { key: 41 }, [
            _cache[89] || (_cache[89] = createBaseVNode("path", { d: "M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" }, null, -1)),
            _cache[90] || (_cache[90] = createBaseVNode("polyline", { points: "14 2 14 8 20 8" }, null, -1)),
            _cache[91] || (_cache[91] = createBaseVNode("text", {
              x: "6",
              y: "17",
              "font-size": "8",
              fill: "currentColor",
              "font-weight": "bold",
              stroke: "none"
            }, "Java", -1))
          ], 64)) : __props.name === "file-html" ? (openBlock(), createElementBlock(Fragment, { key: 42 }, [
            _cache[92] || (_cache[92] = createBaseVNode("path", { d: "M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" }, null, -1)),
            _cache[93] || (_cache[93] = createBaseVNode("polyline", { points: "14 2 14 8 20 8" }, null, -1)),
            _cache[94] || (_cache[94] = createBaseVNode("text", {
              x: "6",
              y: "17",
              "font-size": "9",
              fill: "currentColor",
              "font-weight": "bold",
              stroke: "none"
            }, "HTML", -1))
          ], 64)) : __props.name === "file-css" ? (openBlock(), createElementBlock(Fragment, { key: 43 }, [
            _cache[95] || (_cache[95] = createBaseVNode("path", { d: "M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" }, null, -1)),
            _cache[96] || (_cache[96] = createBaseVNode("polyline", { points: "14 2 14 8 20 8" }, null, -1)),
            _cache[97] || (_cache[97] = createBaseVNode("text", {
              x: "7",
              y: "17",
              "font-size": "9",
              fill: "currentColor",
              "font-weight": "bold",
              stroke: "none"
            }, "CSS", -1))
          ], 64)) : __props.name === "file-json" ? (openBlock(), createElementBlock(Fragment, { key: 44 }, [
            _cache[98] || (_cache[98] = createBaseVNode("path", { d: "M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" }, null, -1)),
            _cache[99] || (_cache[99] = createBaseVNode("polyline", { points: "14 2 14 8 20 8" }, null, -1)),
            _cache[100] || (_cache[100] = createBaseVNode("text", {
              x: "5",
              y: "17",
              "font-size": "9",
              fill: "currentColor",
              "font-weight": "bold",
              stroke: "none"
            }, "{ }", -1))
          ], 64)) : __props.name === "file-md" ? (openBlock(), createElementBlock(Fragment, { key: 45 }, [
            _cache[101] || (_cache[101] = createBaseVNode("path", { d: "M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" }, null, -1)),
            _cache[102] || (_cache[102] = createBaseVNode("polyline", { points: "14 2 14 8 20 8" }, null, -1)),
            _cache[103] || (_cache[103] = createBaseVNode("text", {
              x: "7",
              y: "17",
              "font-size": "9",
              fill: "currentColor",
              "font-weight": "bold",
              stroke: "none"
            }, "MD", -1))
          ], 64)) : __props.name === "file-vue" ? (openBlock(), createElementBlock(Fragment, { key: 46 }, [
            _cache[104] || (_cache[104] = createBaseVNode("path", { d: "M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" }, null, -1)),
            _cache[105] || (_cache[105] = createBaseVNode("polyline", { points: "14 2 14 8 20 8" }, null, -1)),
            _cache[106] || (_cache[106] = createBaseVNode("text", {
              x: "7",
              y: "17",
              "font-size": "9",
              fill: "currentColor",
              "font-weight": "bold",
              stroke: "none"
            }, "Vue", -1))
          ], 64)) : __props.name === "copy" ? (openBlock(), createElementBlock(Fragment, { key: 47 }, [
            _cache[107] || (_cache[107] = createBaseVNode("rect", {
              x: "9",
              y: "9",
              width: "13",
              height: "13",
              rx: "2",
              ry: "2"
            }, null, -1)),
            _cache[108] || (_cache[108] = createBaseVNode("path", { d: "M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" }, null, -1))
          ], 64)) : __props.name === "minus" ? (openBlock(), createElementBlock("line", _hoisted_7$2)) : __props.name === "edit" ? (openBlock(), createElementBlock(Fragment, { key: 49 }, [
            _cache[109] || (_cache[109] = createBaseVNode("path", { d: "M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7" }, null, -1)),
            _cache[110] || (_cache[110] = createBaseVNode("path", { d: "M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z" }, null, -1))
          ], 64)) : __props.name === "trash" ? (openBlock(), createElementBlock(Fragment, { key: 50 }, [
            _cache[111] || (_cache[111] = createBaseVNode("polyline", { points: "3 6 5 6 21 6" }, null, -1)),
            _cache[112] || (_cache[112] = createBaseVNode("path", { d: "M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" }, null, -1))
          ], 64)) : __props.name === "file-plus" ? (openBlock(), createElementBlock(Fragment, { key: 51 }, [
            _cache[113] || (_cache[113] = createBaseVNode("path", { d: "M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" }, null, -1)),
            _cache[114] || (_cache[114] = createBaseVNode("polyline", { points: "14 2 14 8 20 8" }, null, -1)),
            _cache[115] || (_cache[115] = createBaseVNode("line", {
              x1: "12",
              y1: "18",
              x2: "12",
              y2: "12"
            }, null, -1)),
            _cache[116] || (_cache[116] = createBaseVNode("line", {
              x1: "9",
              y1: "15",
              x2: "15",
              y2: "15"
            }, null, -1))
          ], 64)) : __props.name === "message-square" ? (openBlock(), createElementBlock("path", _hoisted_8$2)) : __props.name === "folder-plus" ? (openBlock(), createElementBlock(Fragment, { key: 53 }, [
            _cache[117] || (_cache[117] = createBaseVNode("path", { d: "M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2v3" }, null, -1)),
            _cache[118] || (_cache[118] = createBaseVNode("line", {
              x1: "12",
              y1: "11",
              x2: "12",
              y2: "17"
            }, null, -1)),
            _cache[119] || (_cache[119] = createBaseVNode("line", {
              x1: "9",
              y1: "14",
              x2: "15",
              y2: "14"
            }, null, -1))
          ], 64)) : __props.name === "brain" ? (openBlock(), createElementBlock(Fragment, { key: 54 }, [
            _cache[120] || (_cache[120] = createBaseVNode("path", { d: "M12 2a4 4 0 0 0-4 4v1a5 5 0 0 0-5 5v1a4 4 0 0 0 3 3.87V17a3 3 0 0 0 3 3h6a3 3 0 0 0 3-3v-.13A4 4 0 0 0 21 13v-1a5 5 0 0 0-5-5V6a4 4 0 0 0-4-4z" }, null, -1)),
            _cache[121] || (_cache[121] = createBaseVNode("path", { d: "M9 12v2" }, null, -1)),
            _cache[122] || (_cache[122] = createBaseVNode("path", { d: "M15 12v2" }, null, -1)),
            _cache[123] || (_cache[123] = createBaseVNode("path", { d: "M12 9v5" }, null, -1))
          ], 64)) : __props.name === "check" ? (openBlock(), createElementBlock("polyline", _hoisted_9$2)) : __props.name === "clock" ? (openBlock(), createElementBlock(Fragment, { key: 56 }, [
            _cache[124] || (_cache[124] = createBaseVNode("circle", {
              cx: "12",
              cy: "12",
              r: "10"
            }, null, -1)),
            _cache[125] || (_cache[125] = createBaseVNode("polyline", { points: "12 6 12 12 16 14" }, null, -1))
          ], 64)) : __props.name === "help" ? (openBlock(), createElementBlock(Fragment, { key: 57 }, [
            _cache[126] || (_cache[126] = createBaseVNode("circle", {
              cx: "12",
              cy: "12",
              r: "10"
            }, null, -1)),
            _cache[127] || (_cache[127] = createBaseVNode("path", { d: "M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3" }, null, -1)),
            _cache[128] || (_cache[128] = createBaseVNode("line", {
              x1: "12",
              y1: "17",
              x2: "12.01",
              y2: "17"
            }, null, -1))
          ], 64)) : __props.name === "shield" ? (openBlock(), createElementBlock("path", _hoisted_10$1)) : __props.name === "shield-off" ? (openBlock(), createElementBlock(Fragment, { key: 59 }, [
            _cache[129] || (_cache[129] = createBaseVNode("path", { d: "M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" }, null, -1)),
            _cache[130] || (_cache[130] = createBaseVNode("line", {
              x1: "4",
              y1: "4",
              x2: "20",
              y2: "20",
              stroke: "currentColor",
              "stroke-width": "2",
              "stroke-linecap": "round"
            }, null, -1))
          ], 64)) : __props.name === "code" ? (openBlock(), createElementBlock(Fragment, { key: 60 }, [
            _cache[131] || (_cache[131] = createBaseVNode("polyline", { points: "16 18 22 12 16 6" }, null, -1)),
            _cache[132] || (_cache[132] = createBaseVNode("polyline", { points: "8 6 2 12 8 18" }, null, -1))
          ], 64)) : __props.name === "list" ? (openBlock(), createElementBlock(Fragment, { key: 61 }, [
            _cache[133] || (_cache[133] = createStaticVNode('<line x1="8" y1="6" x2="21" y2="6" data-v-faf69761></line><line x1="8" y1="12" x2="21" y2="12" data-v-faf69761></line><line x1="8" y1="18" x2="21" y2="18" data-v-faf69761></line><line x1="3" y1="6" x2="3.01" y2="6" data-v-faf69761></line><line x1="3" y1="12" x2="3.01" y2="12" data-v-faf69761></line><line x1="3" y1="18" x2="3.01" y2="18" data-v-faf69761></line>', 6))
          ], 64)) : __props.name === "layers" ? (openBlock(), createElementBlock(Fragment, { key: 62 }, [
            _cache[134] || (_cache[134] = createBaseVNode("polygon", { points: "12 2 2 7 12 12 22 7 12 2" }, null, -1)),
            _cache[135] || (_cache[135] = createBaseVNode("polyline", { points: "2 17 12 22 22 17" }, null, -1)),
            _cache[136] || (_cache[136] = createBaseVNode("polyline", { points: "2 12 12 17 22 12" }, null, -1))
          ], 64)) : __props.name === "eye" ? (openBlock(), createElementBlock(Fragment, { key: 63 }, [
            _cache[137] || (_cache[137] = createBaseVNode("path", { d: "M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" }, null, -1)),
            _cache[138] || (_cache[138] = createBaseVNode("circle", {
              cx: "12",
              cy: "12",
              r: "3"
            }, null, -1))
          ], 64)) : __props.name === "eye-off" ? (openBlock(), createElementBlock(Fragment, { key: 64 }, [
            _cache[139] || (_cache[139] = createBaseVNode("path", { d: "M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94" }, null, -1)),
            _cache[140] || (_cache[140] = createBaseVNode("path", { d: "M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19" }, null, -1)),
            _cache[141] || (_cache[141] = createBaseVNode("line", {
              x1: "1",
              y1: "1",
              x2: "23",
              y2: "23"
            }, null, -1))
          ], 64)) : __props.name === "bug" ? (openBlock(), createElementBlock(Fragment, { key: 65 }, [
            _cache[142] || (_cache[142] = createStaticVNode('<rect x="8" y="2" width="8" height="4" rx="1" ry="1" data-v-faf69761></rect><path d="M20 12h-3a5 5 0 0 1-5 5 5 5 0 0 1-5-5H4" data-v-faf69761></path><path d="M4 8h16" data-v-faf69761></path><path d="M12 2v7" data-v-faf69761></path><path d="M9 17l-3 4" data-v-faf69761></path><path d="M15 17l3 4" data-v-faf69761></path>', 6))
          ], 64)) : __props.name === "check-circle" ? (openBlock(), createElementBlock(Fragment, { key: 66 }, [
            _cache[143] || (_cache[143] = createBaseVNode("path", { d: "M22 11.08V12a10 10 0 1 1-5.93-9.14" }, null, -1)),
            _cache[144] || (_cache[144] = createBaseVNode("polyline", { points: "22 4 12 14.01 9 11.01" }, null, -1))
          ], 64)) : __props.name === "book-open" ? (openBlock(), createElementBlock(Fragment, { key: 67 }, [
            _cache[145] || (_cache[145] = createBaseVNode("path", { d: "M2 3h6a4 4 0 0 1 4 4v14a3 3 0 0 0-3-3H2z" }, null, -1)),
            _cache[146] || (_cache[146] = createBaseVNode("path", { d: "M22 3h-6a4 4 0 0 0-4 4v14a3 3 0 0 1 3-3h7z" }, null, -1))
          ], 64)) : __props.name === "tool" ? (openBlock(), createElementBlock("path", _hoisted_11$1)) : __props.name === "keyboard" ? (openBlock(), createElementBlock(Fragment, { key: 69 }, [
            _cache[147] || (_cache[147] = createStaticVNode('<rect x="2" y="4" width="20" height="16" rx="2" ry="2" data-v-faf69761></rect><line x1="6" y1="8" x2="6.01" y2="8" data-v-faf69761></line><line x1="10" y1="8" x2="10.01" y2="8" data-v-faf69761></line><line x1="14" y1="8" x2="14.01" y2="8" data-v-faf69761></line><line x1="18" y1="8" x2="18.01" y2="8" data-v-faf69761></line><line x1="6" y1="12" x2="6.01" y2="12" data-v-faf69761></line><line x1="10" y1="12" x2="10.01" y2="12" data-v-faf69761></line><line x1="14" y1="12" x2="14.01" y2="12" data-v-faf69761></line><line x1="18" y1="12" x2="18.01" y2="12" data-v-faf69761></line><line x1="6" y1="16" x2="18" y2="16" data-v-faf69761></line>', 10))
          ], 64)) : __props.name === "chevron-left" ? (openBlock(), createElementBlock("polyline", _hoisted_12$1)) : __props.name === "grid" ? (openBlock(), createElementBlock(Fragment, { key: 71 }, [
            _cache[148] || (_cache[148] = createBaseVNode("rect", {
              x: "3",
              y: "3",
              width: "7",
              height: "7"
            }, null, -1)),
            _cache[149] || (_cache[149] = createBaseVNode("rect", {
              x: "14",
              y: "3",
              width: "7",
              height: "7"
            }, null, -1)),
            _cache[150] || (_cache[150] = createBaseVNode("rect", {
              x: "14",
              y: "14",
              width: "7",
              height: "7"
            }, null, -1)),
            _cache[151] || (_cache[151] = createBaseVNode("rect", {
              x: "3",
              y: "14",
              width: "7",
              height: "7"
            }, null, -1))
          ], 64)) : __props.name === "puzzle" ? (openBlock(), createElementBlock(Fragment, { key: 72 }, [
            _cache[152] || (_cache[152] = createBaseVNode("path", { d: "M4 7h3a2 2 0 0 1 4 0h9v9h-3a2 2 0 0 0-4 0H4z" }, null, -1)),
            _cache[153] || (_cache[153] = createBaseVNode("path", { d: "M11 7v9" }, null, -1))
          ], 64)) : (openBlock(), createElementBlock(Fragment, { key: 73 }, [
            _cache[154] || (_cache[154] = createBaseVNode("path", { d: "M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" }, null, -1)),
            _cache[155] || (_cache[155] = createBaseVNode("polyline", { points: "14 2 14 8 20 8" }, null, -1)),
            _cache[156] || (_cache[156] = createBaseVNode("line", {
              x1: "9",
              y1: "13",
              x2: "15",
              y2: "13"
            }, null, -1)),
            _cache[157] || (_cache[157] = createBaseVNode("line", {
              x1: "9",
              y1: "17",
              x2: "15",
              y2: "17"
            }, null, -1))
          ], 64))
        ], 8, _hoisted_1$2);
      };
    }
  };
  const SvgIcon = /* @__PURE__ */ _export_sfc(_sfc_main$2, [["__scopeId", "data-v-faf69761"]]);
  const _hoisted_1$1 = { class: "plugin-panel" };
  const _hoisted_2$1 = { class: "pp-header" };
  const _hoisted_3$1 = { class: "pp-title" };
  const _hoisted_4$1 = { class: "pp-actions" };
  const _hoisted_5$1 = {
    key: 0,
    class: "pp-toolset"
  };
  const _hoisted_6$1 = { class: "pp-ts-head" };
  const _hoisted_7$1 = { label: "工作区工具集（本项目）" };
  const _hoisted_8$1 = ["value"];
  const _hoisted_9$1 = ["value"];
  const _hoisted_10 = {
    key: 0,
    class: "pp-ts-body"
  };
  const _hoisted_11 = { class: "pp-ts-title" };
  const _hoisted_12 = { class: "pp-ts-scope" };
  const _hoisted_13 = { class: "pp-ts-prow" };
  const _hoisted_14 = { class: "pp-ts-pname" };
  const _hoisted_15 = ["onClick"];
  const _hoisted_16 = {
    key: 1,
    class: "pp-ts-muted"
  };
  const _hoisted_17 = {
    key: 0,
    class: "pp-ts-purpose"
  };
  const _hoisted_18 = {
    key: 1,
    class: "pp-ts-tools"
  };
  const _hoisted_19 = ["title"];
  const _hoisted_20 = ["checked", "onChange"];
  const _hoisted_21 = {
    key: 2,
    class: "pp-ts-muted"
  };
  const _hoisted_22 = { class: "pp-ts-add" };
  const _hoisted_23 = ["value"];
  const _hoisted_24 = ["disabled"];
  const _hoisted_25 = {
    key: 1,
    class: "pp-ts-empty"
  };
  const _hoisted_26 = {
    key: 1,
    class: "pp-new"
  };
  const _hoisted_27 = { class: "pp-new-foot" };
  const _hoisted_28 = { class: "pp-check" };
  const _hoisted_29 = ["disabled"];
  const _hoisted_30 = {
    key: 2,
    class: "pp-client"
  };
  const _hoisted_31 = { class: "pp-client-tabs" };
  const _hoisted_32 = ["onClick"];
  const _hoisted_33 = { class: "pp-client-tab-title" };
  const _hoisted_34 = {
    key: 3,
    class: "pp-slots"
  };
  const _hoisted_35 = ["title"];
  const _hoisted_36 = { class: "pp-slots-title" };
  const _hoisted_37 = { class: "pp-slot-info" };
  const _hoisted_38 = { class: "pp-slot-title-row" };
  const _hoisted_39 = { class: "pp-slot-id" };
  const _hoisted_40 = ["value", "onChange", "title"];
  const _hoisted_41 = { value: "" };
  const _hoisted_42 = ["value"];
  const _hoisted_43 = {
    key: 1,
    class: "pp-slot-list"
  };
  const _hoisted_44 = ["checked", "onChange"];
  const _hoisted_45 = {
    key: 0,
    class: "pp-slot-empty"
  };
  const _hoisted_46 = { class: "pp-list" };
  const _hoisted_47 = {
    key: 0,
    class: "pp-loading"
  };
  const _hoisted_48 = {
    key: 1,
    class: "pp-empty"
  };
  const _hoisted_49 = {
    key: 2,
    class: "pp-empty"
  };
  const _hoisted_50 = ["onClick"];
  const _hoisted_51 = ["title"];
  const _hoisted_52 = {
    key: 0,
    class: "pp-badge",
    title: "全局插件：跨工作区生效（UI 类），不属于任何工具集"
  };
  const _hoisted_53 = {
    key: 1,
    class: "pp-badge",
    title: "含 client 半（浏览器 UI，已批准装载）"
  };
  const _hoisted_54 = {
    key: 2,
    class: "pp-badge pp-badge-warn",
    title: "client 半待激活批准：在对话中用 cordis_run 装载该插件触发审批"
  };
  const _hoisted_55 = {
    key: 3,
    class: "pp-badge",
    title: "含 client 半（浏览器 UI；装载后需批准）"
  };
  const _hoisted_56 = ["title"];
  const _hoisted_57 = ["title"];
  const _hoisted_58 = ["checked", "onChange"];
  const _hoisted_59 = {
    key: 0,
    class: "pp-detail"
  };
  const _hoisted_60 = {
    key: 0,
    class: "pp-d-purpose"
  };
  const _hoisted_61 = {
    key: 1,
    class: "pp-d-line"
  };
  const _hoisted_62 = { key: 0 };
  const _hoisted_63 = {
    key: 2,
    class: "pp-d-line"
  };
  const _hoisted_64 = {
    key: 3,
    class: "pp-d-line"
  };
  const _hoisted_65 = {
    key: 4,
    class: "pp-d-tools"
  };
  const _hoisted_66 = { class: "pp-d-tools-title" };
  const _hoisted_67 = ["title"];
  const _hoisted_68 = ["title"];
  const _hoisted_69 = ["checked", "onChange"];
  const _hoisted_70 = {
    key: 5,
    class: "pp-d-code"
  };
  const _hoisted_71 = { class: "pp-d-code-head" };
  const _hoisted_72 = ["onClick"];
  const _hoisted_73 = { class: "pp-d-actions" };
  const _hoisted_74 = ["onClick"];
  const _hoisted_75 = ["onClick"];
  const _hoisted_76 = ["onClick"];
  const _sfc_main$1 = {
    __name: "PluginPanel",
    setup(__props) {
      const plugins = /* @__PURE__ */ ref([]);
      const loading = /* @__PURE__ */ ref(false);
      const refreshing = /* @__PURE__ */ ref(false);
      const loadError = /* @__PURE__ */ ref(false);
      const expanded = /* @__PURE__ */ reactive({});
      const showNew = /* @__PURE__ */ ref(false);
      const defining = /* @__PURE__ */ ref(false);
      const slotsOpen = /* @__PURE__ */ ref(false);
      const newMsg = /* @__PURE__ */ ref("");
      const newMsgErr = /* @__PURE__ */ ref(false);
      const activePanelId = /* @__PURE__ */ ref("");
      const clientPanelEl = /* @__PURE__ */ ref(null);
      const showToolset = /* @__PURE__ */ ref(false);
      const toolsetMetas = /* @__PURE__ */ ref([]);
      const tsName = /* @__PURE__ */ ref("");
      const tsDetail = /* @__PURE__ */ ref(null);
      const addPluginName = /* @__PURE__ */ ref("");
      const newForm = /* @__PURE__ */ reactive({ purpose: "", code: "", client: "", language: "", run: true });
      async function loadToolsets() {
        try {
          const list = await api.getToolsets() || [];
          toolsetMetas.value = list.filter((t) => t.scope !== "global");
        } catch (e) {
          toolsetMetas.value = [];
        }
        if (tsName.value && !toolsetMetas.value.some((t) => t.name === tsName.value)) tsName.value = "";
        if (tsName.value) await loadToolsetDetail();
        else tsDetail.value = null;
      }
      async function loadToolsetDetail() {
        if (!tsName.value) {
          tsDetail.value = null;
          return;
        }
        try {
          tsDetail.value = await api.getToolsets(tsName.value);
        } catch (e) {
          window.$toast && window.$toast("加载工具集失败: " + (e.message || e), "error");
        }
      }
      function pluginToolsOf(pname) {
        const p2 = plugins.value.find((x) => x.name === pname);
        return p2 && p2.tools || [];
      }
      function isToolDisabled(pl, t) {
        return (pl.disabledTools || []).includes(t);
      }
      async function edit(data) {
        try {
          const res = await api.toolsetEdit({ name: tsName.value, ...data });
          window.$toast && window.$toast(res && res.message || "操作成功", "info");
          await loadToolsetDetail();
          await refresh();
        } catch (e) {
          window.$toast && window.$toast(e.message || "操作失败", "error");
        }
      }
      function toggleTool(pl, t) {
        edit({ action: isToolDisabled(pl, t) ? "enable_tool" : "rm_tool", plugin_name: pl.name, tool: t });
      }
      async function doAddPlugin() {
        if (!addPluginName.value) return;
        await edit({ action: "add_plugin", plugin_name: addPluginName.value });
        addPluginName.value = "";
      }
      const addablePlugins = computed(() => {
        const inTs = new Set((tsDetail.value && tsDetail.value.plugins || []).map((p2) => p2.name));
        return plugins.value.filter((p2) => !inTs.has(p2.name));
      });
      watch(showToolset, (v) => {
        if (v) loadToolsets();
      });
      function fetchPluginsJSON() {
        return new Promise((resolve2, reject) => {
          const x = new XMLHttpRequest();
          x.open("GET", "/api/plugins", true);
          x.timeout = 8e3;
          x.onload = () => {
            if (x.status >= 200 && x.status < 300) {
              try {
                resolve2(JSON.parse(x.responseText));
              } catch (e) {
                reject(e);
              }
            } else reject(new Error("HTTP " + x.status));
          };
          x.onerror = () => reject(new Error("network error"));
          x.ontimeout = () => reject(new Error("timeout"));
          x.send();
        });
      }
      async function refresh() {
        refreshing.value = true;
        loading.value = true;
        loadError.value = false;
        try {
          let list = [];
          for (let attempt = 0; attempt < 2; attempt++) {
            try {
              const data = await fetchPluginsJSON();
              if (Array.isArray(data)) {
                list = data;
                break;
              }
            } catch (e) {
              list = [];
            }
          }
          plugins.value = list;
          if (!list.length) loadError.value = true;
          const detailTargets = plugins.value.filter((p2) => p2.hasClient && !p2.clientCode);
          await Promise.allSettled(detailTargets.map(async (p2) => {
            try {
              const d = await api.getPluginDetail(p2.name);
              if (d && d.clientCode) p2.clientCode = d.clientCode;
            } catch (e) {
            }
          }));
          await syncClientHalves(plugins.value);
        } catch (e) {
          console.warn("[plugin] 加载失败", e);
          loadError.value = true;
        } finally {
          loading.value = false;
          refreshing.value = false;
        }
      }
      async function toggleDetail(p2) {
        expanded[p2.name] = !expanded[p2.name];
        if (expanded[p2.name] && p2.hasClient && !p2.clientCode) {
          try {
            const d = await api.getPluginDetail(p2.name);
            if (d) Object.assign(p2, d);
          } catch (e) {
          }
        }
      }
      function pluginToolOn(p2, t) {
        return !(p2.toolStates && p2.toolStates[t] === false);
      }
      async function togglePluginTool(p2, t) {
        const target = !pluginToolOn(p2, t);
        try {
          const res = await api.pluginToolToggle(t, target);
          window.$toast && window.$toast(res && res.message || (target ? "已启用" : "已禁用") + " " + t, "info");
          if (!p2.toolStates) p2.toolStates = {};
          p2.toolStates[t] = target;
        } catch (e) {
          window.$toast && window.$toast(e.message || "操作失败", "error");
        }
      }
      function uiSlotsOf(pname) {
        return clientSlots.filter((s) => s.pluginName === pname);
      }
      function uiPluginActive(pname) {
        const slots = uiSlotsOf(pname);
        if (!slots.length) return false;
        return slots.some((s) => {
          if (s.kind === "list") return isOverlayActive(s.slotId, s.pluginName);
          return isPluginUIEnabled(s.pluginName);
        });
      }
      function toggleUiPlugin(p2, on) {
        const slots = uiSlotsOf(p2.name);
        for (const s of slots) {
          if (s.kind === "list") {
            setOverlayActive(s.slotId, s.pluginName, on);
          } else {
            setPluginUIEnabled(s.pluginName, on);
            if (on) setSlotOwner(s.slotId, s.pluginName);
          }
        }
        const recover = p2.name === "ui-sidebar" && !on ? "（已停用；恢复入口：右下角壳级按钮）" : "";
        window.$toast && window.$toast(on ? "已启用 " + p2.name + " 的 UI（" + slots.map((s) => s.slotId).join(", ") + "）" : "已停用 " + p2.name + " 的 UI（区域恢复空态）" + recover, on ? "info" : "warn");
        emitSlotChanged();
      }
      async function doAction(p2, action) {
        try {
          await api.pluginAction(p2.name, action);
          if (action === "undefine") {
            unloadClientHalf(p2.name);
            delete expanded[p2.name];
            plugins.value = plugins.value.filter((x) => x.name !== p2.name);
          } else {
            await refresh();
          }
          window.$toast && window.$toast(`${action === "start" ? "已启动" : action === "stop" ? "已停止" : "已删除"} ${p2.name}`, "info");
        } catch (e) {
          window.$toast && window.$toast(e.message || "操作失败", "error");
        }
      }
      async function doDefine() {
        defining.value = true;
        newMsg.value = "";
        newMsgErr.value = false;
        try {
          const res = await api.definePlugin({
            purpose: newForm.purpose,
            code: newForm.code,
            client: newForm.client || void 0,
            language: newForm.language || void 0,
            run: newForm.run
          });
          newMsg.value = `已定义 ${res.id}（${res.state}）`;
          showNew.value = false;
          await refresh();
        } catch (e) {
          newMsgErr.value = true;
          newMsg.value = e.message || "定义失败";
        } finally {
          defining.value = false;
        }
      }
      function copyText(t) {
        try {
          navigator.clipboard.writeText(t);
          window.$toast && window.$toast("已复制", "info");
        } catch (e) {
        }
      }
      function selectPanel(id) {
        activePanelId.value = id;
        renderActivePanel();
      }
      async function renderActivePanel() {
        await nextTick();
        const el = clientPanelEl.value;
        if (!el) return;
        el.innerHTML = "";
        const panel = clientPanels.find((p2) => p2.id === activePanelId.value);
        if (panel && panel.render) {
          try {
            panel.render(el, getUIFor(panel.pluginName));
          } catch (e) {
            console.warn("[plugin] 面板渲染错误", panel.id, e);
            el.innerHTML = '<div style="color:var(--text-muted);padding:8px;font-size:12px">面板渲染失败</div>';
          }
        }
      }
      function onPanelsChanged(panels) {
        if (!activePanelId.value || !panels.some((p2) => p2.id === activePanelId.value)) {
          activePanelId.value = panels.length ? panels[0].id : "";
        }
        renderActivePanel();
      }
      const slotGroups = /* @__PURE__ */ ref([]);
      let slotUnsub = null;
      function refreshSlots() {
        const keys = [...new Set(clientSlots.map((s) => s.slotId + "::" + s.kind))];
        slotGroups.value = keys.map((k) => {
          const [slotId, kind] = k.split("::");
          const candidates = getSlotCandidates(slotId).filter((c) => c.kind === kind);
          return { slotId, kind, owner: getSlotOwner(slotId), candidates, builtin: null };
        });
      }
      function overlayActive(slotId, pluginName) {
        return isOverlayActive(slotId, pluginName);
      }
      function toggleOverlay(slotId, pluginName, on) {
        setOverlayActive(slotId, pluginName, on);
      }
      function switchSlot(slotId, pluginName) {
        setSlotOwner(slotId, pluginName || "");
        refreshSlots();
      }
      onMounted(() => {
        setPanelMount(onPanelsChanged);
        slotUnsub = setSlotMount(refreshSlots);
        startPolling();
        refresh();
      });
      onUnmounted(() => {
        stopPolling();
        setPanelMount(null);
        if (slotUnsub) {
          slotUnsub();
          slotUnsub = null;
        }
      });
      return (_ctx, _cache) => {
        return openBlock(), createElementBlock("div", _hoisted_1$1, [
          createBaseVNode("div", _hoisted_2$1, [
            createBaseVNode("span", _hoisted_3$1, [
              createVNode(SvgIcon, {
                name: "puzzle",
                size: 14
              }),
              _cache[11] || (_cache[11] = createTextVNode(" 插件", -1))
            ]),
            createBaseVNode("div", _hoisted_4$1, [
              createBaseVNode("button", {
                class: "pp-icon-btn",
                onClick: refresh,
                title: "刷新"
              }, [
                createVNode(SvgIcon, {
                  name: "refresh",
                  size: 13,
                  class: normalizeClass({ spinning: refreshing.value })
                }, null, 8, ["class"])
              ]),
              createBaseVNode("button", {
                class: normalizeClass(["pp-icon-btn", { active: showToolset.value }]),
                onClick: _cache[0] || (_cache[0] = ($event) => showToolset.value = !showToolset.value),
                title: "工具集管理（插件化：加插件/删插件/摘工具）"
              }, [
                createVNode(SvgIcon, {
                  name: "layers",
                  size: 13
                })
              ], 2),
              createBaseVNode("button", {
                class: "pp-icon-btn",
                onClick: _cache[1] || (_cache[1] = ($event) => showNew.value = !showNew.value),
                title: "新建插件"
              }, [
                createVNode(SvgIcon, {
                  name: "plus",
                  size: 14
                })
              ])
            ])
          ]),
          showToolset.value ? (openBlock(), createElementBlock("div", _hoisted_5$1, [
            createBaseVNode("div", _hoisted_6$1, [
              withDirectives(createBaseVNode("select", {
                "onUpdate:modelValue": _cache[2] || (_cache[2] = ($event) => tsName.value = $event),
                class: "pp-input pp-lang",
                onChange: loadToolsetDetail
              }, [
                _cache[12] || (_cache[12] = createBaseVNode("option", { value: "" }, "选择工具集…", -1)),
                createBaseVNode("optgroup", _hoisted_7$1, [
                  (openBlock(true), createElementBlock(Fragment, null, renderList(toolsetMetas.value.filter((x) => x.scope !== "builtin"), (t) => {
                    return openBlock(), createElementBlock("option", {
                      key: t.name,
                      value: t.name
                    }, toDisplayString(t.name) + "（" + toDisplayString(t.pluginCount) + " 插件）", 9, _hoisted_8$1);
                  }), 128)),
                  (openBlock(true), createElementBlock(Fragment, null, renderList(toolsetMetas.value.filter((x) => x.scope === "builtin"), (t) => {
                    return openBlock(), createElementBlock("option", {
                      key: t.name,
                      value: t.name
                    }, toDisplayString(t.name) + "（" + toDisplayString(t.pluginCount) + " 插件·内置默认）", 9, _hoisted_9$1);
                  }), 128))
                ])
              ], 544), [
                [vModelSelect, tsName.value]
              ]),
              createBaseVNode("button", {
                class: "pp-btn",
                onClick: loadToolsets
              }, "刷新")
            ]),
            tsDetail.value ? (openBlock(), createElementBlock("div", _hoisted_10, [
              createBaseVNode("div", _hoisted_11, [
                createTextVNode(toDisplayString(tsDetail.value.name) + " ", 1),
                createBaseVNode("span", _hoisted_12, toDisplayString(tsDetail.value.project ? tsDetail.value.project + "·" : "") + toDisplayString(tsDetail.value.description || "工具集"), 1)
              ]),
              (openBlock(true), createElementBlock(Fragment, null, renderList(tsDetail.value.plugins, (pl) => {
                return openBlock(), createElementBlock("div", {
                  key: pl.name,
                  class: "pp-ts-plugin"
                }, [
                  createBaseVNode("div", _hoisted_13, [
                    createBaseVNode("span", _hoisted_14, toDisplayString(pl.name), 1),
                    tsDetail.value.scope !== "builtin" ? (openBlock(), createElementBlock("button", {
                      key: 0,
                      class: "pp-btn danger",
                      onClick: ($event) => edit({ action: "rm_plugin", plugin_name: pl.name })
                    }, "移出工具集", 8, _hoisted_15)) : (openBlock(), createElementBlock("span", _hoisted_16, "内置"))
                  ]),
                  pl.purpose ? (openBlock(), createElementBlock("div", _hoisted_17, toDisplayString(pl.purpose), 1)) : createCommentVNode("", true),
                  pluginToolsOf(pl.name).length ? (openBlock(), createElementBlock("div", _hoisted_18, [
                    (openBlock(true), createElementBlock(Fragment, null, renderList(pluginToolsOf(pl.name), (t) => {
                      return openBlock(), createElementBlock("label", {
                        key: t,
                        class: "pp-ts-tool",
                        title: isToolDisabled(pl, t) ? "已摘除（对 agent 不可见），点击恢复" : "点击摘除（插件保留、工具不可见）"
                      }, [
                        createBaseVNode("input", {
                          type: "checkbox",
                          checked: !isToolDisabled(pl, t),
                          onChange: ($event) => toggleTool(pl, t)
                        }, null, 40, _hoisted_20),
                        createBaseVNode("span", {
                          class: normalizeClass({ off: isToolDisabled(pl, t) })
                        }, toDisplayString(t), 3)
                      ], 8, _hoisted_19);
                    }), 128))
                  ])) : (openBlock(), createElementBlock("div", _hoisted_21, "（插件未运行或无工具）"))
                ]);
              }), 128)),
              createBaseVNode("div", _hoisted_22, [
                withDirectives(createBaseVNode("select", {
                  "onUpdate:modelValue": _cache[3] || (_cache[3] = ($event) => addPluginName.value = $event),
                  class: "pp-input pp-lang"
                }, [
                  _cache[13] || (_cache[13] = createBaseVNode("option", { value: "" }, "把宿主插件加入工具集…", -1)),
                  (openBlock(true), createElementBlock(Fragment, null, renderList(addablePlugins.value, (p2) => {
                    return openBlock(), createElementBlock("option", {
                      key: p2.name,
                      value: p2.name
                    }, [
                      createTextVNode(toDisplayString(p2.name), 1),
                      p2.tools && p2.tools.length ? (openBlock(), createElementBlock(Fragment, { key: 0 }, [
                        createTextVNode("（" + toDisplayString(p2.tools.length) + " 工具）", 1)
                      ], 64)) : createCommentVNode("", true)
                    ], 8, _hoisted_23);
                  }), 128))
                ], 512), [
                  [vModelSelect, addPluginName.value]
                ]),
                createBaseVNode("button", {
                  class: "pp-btn primary",
                  disabled: !addPluginName.value,
                  onClick: doAddPlugin
                }, "加入", 8, _hoisted_24)
              ])
            ])) : (openBlock(), createElementBlock("div", _hoisted_25, "选择上方工具集查看/编辑其插件与工具"))
          ])) : createCommentVNode("", true),
          showNew.value ? (openBlock(), createElementBlock("div", _hoisted_26, [
            _cache[16] || (_cache[16] = createBaseVNode("div", { class: "pp-new-title" }, "新建 JS 动态插件", -1)),
            withDirectives(createBaseVNode("input", {
              "onUpdate:modelValue": _cache[4] || (_cache[4] = ($event) => newForm.purpose = $event),
              placeholder: "用途说明（必填）",
              class: "pp-input"
            }, null, 512), [
              [vModelText, newForm.purpose]
            ]),
            withDirectives(createBaseVNode("textarea", {
              "onUpdate:modelValue": _cache[5] || (_cache[5] = ($event) => newForm.code = $event),
              placeholder: "host 半代码（必填）：(async () => { return { name, apply(ctx, config) } })()",
              class: "pp-textarea code",
              rows: "6"
            }, null, 512), [
              [vModelText, newForm.code]
            ]),
            withDirectives(createBaseVNode("textarea", {
              "onUpdate:modelValue": _cache[6] || (_cache[6] = ($event) => newForm.client = $event),
              placeholder: "client 半代码（可选）：(ui) => { ui.registerPanel({...}); ui.on('ui:xxx', fn) }",
              class: "pp-textarea code",
              rows: "4"
            }, null, 512), [
              [vModelText, newForm.client]
            ]),
            createBaseVNode("div", _hoisted_27, [
              withDirectives(createBaseVNode("select", {
                "onUpdate:modelValue": _cache[7] || (_cache[7] = ($event) => newForm.language = $event),
                class: "pp-input pp-lang"
              }, [..._cache[14] || (_cache[14] = [
                createBaseVNode("option", { value: "" }, "语言(自动)", -1),
                createBaseVNode("option", { value: "js" }, "js", -1),
                createBaseVNode("option", { value: "ts" }, "ts", -1)
              ])], 512), [
                [vModelSelect, newForm.language]
              ]),
              createBaseVNode("label", _hoisted_28, [
                withDirectives(createBaseVNode("input", {
                  type: "checkbox",
                  "onUpdate:modelValue": _cache[8] || (_cache[8] = ($event) => newForm.run = $event)
                }, null, 512), [
                  [vModelCheckbox, newForm.run]
                ]),
                _cache[15] || (_cache[15] = createTextVNode(" 定义后立即装载", -1))
              ]),
              createBaseVNode("button", {
                class: "pp-btn primary",
                disabled: defining.value || !newForm.purpose || !newForm.code,
                onClick: doDefine
              }, toDisplayString(defining.value ? "定义中…" : "定义"), 9, _hoisted_29)
            ]),
            newMsg.value ? (openBlock(), createElementBlock("div", {
              key: 0,
              class: normalizeClass(["pp-new-msg", { err: newMsgErr.value }])
            }, toDisplayString(newMsg.value), 3)) : createCommentVNode("", true)
          ])) : createCommentVNode("", true),
          unref(clientPanels).length > 0 ? (openBlock(), createElementBlock("div", _hoisted_30, [
            createBaseVNode("div", _hoisted_31, [
              (openBlock(true), createElementBlock(Fragment, null, renderList(unref(clientPanels), (p2) => {
                return openBlock(), createElementBlock("div", {
                  key: p2.id,
                  class: normalizeClass(["pp-client-tab", { active: activePanelId.value === p2.id }]),
                  onClick: ($event) => selectPanel(p2.id)
                }, [
                  createVNode(SvgIcon, {
                    name: p2.icon || "sparkles",
                    size: 12
                  }, null, 8, ["name"]),
                  createBaseVNode("span", _hoisted_33, toDisplayString(p2.title), 1)
                ], 10, _hoisted_32);
              }), 128))
            ]),
            createBaseVNode("div", {
              ref_key: "clientPanelEl",
              ref: clientPanelEl,
              class: "pp-client-body"
            }, null, 512)
          ])) : createCommentVNode("", true),
          slotGroups.value.length > 0 ? (openBlock(), createElementBlock("div", _hoisted_34, [
            createBaseVNode("div", {
              class: "pp-slots-head",
              onClick: _cache[9] || (_cache[9] = ($event) => slotsOpen.value = !slotsOpen.value),
              title: slotsOpen.value ? "点击收起 UI 槽位列表" : "点击展开 UI 槽位列表",
              style: { "cursor": "pointer" }
            }, [
              createBaseVNode("span", _hoisted_36, [
                createVNode(SvgIcon, {
                  name: "layers",
                  size: 13
                }),
                _cache[17] || (_cache[17] = createTextVNode(" UI 槽位", -1))
              ]),
              _cache[18] || (_cache[18] = createBaseVNode("span", { class: "pp-slots-sub" }, "插件可替换的界面区域", -1)),
              createVNode(SvgIcon, {
                name: "chevron-right",
                size: 11,
                class: normalizeClass(["pp-chevron", { open: slotsOpen.value }])
              }, null, 8, ["class"])
            ], 8, _hoisted_35),
            slotsOpen.value ? (openBlock(true), createElementBlock(Fragment, { key: 0 }, renderList(slotGroups.value, (g) => {
              return openBlock(), createElementBlock("div", {
                key: g.slotId + "::" + g.kind,
                class: "pp-slot-row"
              }, [
                createBaseVNode("div", _hoisted_37, [
                  createBaseVNode("div", _hoisted_38, [
                    createBaseVNode("span", _hoisted_39, toDisplayString(g.slotId), 1),
                    createBaseVNode("span", {
                      class: normalizeClass(["pp-slot-kind", g.kind === "list" ? "kind-list" : "kind-single"])
                    }, toDisplayString(g.kind === "list" ? "叠加" : "替换"), 3)
                  ]),
                  createBaseVNode("span", {
                    class: normalizeClass(["pp-slot-owner", { builtin: !g.owner && g.kind !== "list" }])
                  }, toDisplayString(g.kind === "list" ? g.candidates.length ? g.candidates.length + " 个叠加条目" : "（无叠加条目）" : g.owner ? g.owner : g.builtin ? "内置组件" : "（无宿主）"), 3)
                ]),
                g.kind !== "list" ? (openBlock(), createElementBlock("select", {
                  key: 0,
                  class: "pp-input pp-slot-select",
                  value: g.owner,
                  onChange: ($event) => switchSlot(g.slotId, $event.target.value),
                  title: "切换 " + g.slotId + " 区域的渲染者"
                }, [
                  createBaseVNode("option", _hoisted_41, toDisplayString(g.builtin ? "内置组件（默认）" : "（未占用）"), 1),
                  (openBlock(true), createElementBlock(Fragment, null, renderList(g.candidates, (c) => {
                    return openBlock(), createElementBlock("option", {
                      key: c.pluginName,
                      value: c.pluginName
                    }, toDisplayString(c.pluginName) + " · " + toDisplayString(c.title), 9, _hoisted_42);
                  }), 128))
                ], 40, _hoisted_40)) : (openBlock(), createElementBlock("div", _hoisted_43, [
                  (openBlock(true), createElementBlock(Fragment, null, renderList(g.candidates, (c) => {
                    return openBlock(), createElementBlock("label", {
                      key: c.pluginName,
                      class: "pp-slot-list-item"
                    }, [
                      createBaseVNode("input", {
                        type: "checkbox",
                        checked: overlayActive(g.slotId, c.pluginName),
                        onChange: ($event) => toggleOverlay(g.slotId, c.pluginName, $event.target.checked)
                      }, null, 40, _hoisted_44),
                      createBaseVNode("span", null, toDisplayString(c.pluginName) + " · " + toDisplayString(c.title), 1)
                    ]);
                  }), 128)),
                  !g.candidates.length ? (openBlock(), createElementBlock("span", _hoisted_45, "（无叠加条目）")) : createCommentVNode("", true)
                ]))
              ]);
            }), 128)) : createCommentVNode("", true)
          ])) : createCommentVNode("", true),
          createBaseVNode("div", _hoisted_46, [
            loading.value && plugins.value.length === 0 ? (openBlock(), createElementBlock("div", _hoisted_47, [
              createVNode(SvgIcon, {
                name: "refresh",
                size: 16,
                class: "spinner"
              }),
              _cache[19] || (_cache[19] = createBaseVNode("span", null, "加载插件…", -1))
            ])) : loadError.value ? (openBlock(), createElementBlock("div", _hoisted_48, [
              createVNode(SvgIcon, {
                name: "puzzle",
                size: 22,
                color: "var(--text-muted)"
              }),
              _cache[20] || (_cache[20] = createBaseVNode("span", null, "插件列表加载失败", -1)),
              createBaseVNode("button", {
                class: "pp-btn primary",
                onClick: refresh
              }, "重试")
            ])) : plugins.value.length === 0 && !loading.value ? (openBlock(), createElementBlock("div", _hoisted_49, [
              createVNode(SvgIcon, {
                name: "puzzle",
                size: 22,
                color: "var(--text-muted)"
              }),
              _cache[21] || (_cache[21] = createBaseVNode("span", null, "暂无插件", -1)),
              _cache[22] || (_cache[22] = createBaseVNode("span", { class: "pp-empty-sub" }, "点击上方 + 新建 JS 动态插件，或用对话 cordis_define 定义", -1))
            ])) : createCommentVNode("", true),
            (openBlock(true), createElementBlock(Fragment, null, renderList(plugins.value, (p2) => {
              return openBlock(), createElementBlock("div", {
                key: p2.name,
                class: "pp-item"
              }, [
                createBaseVNode("div", {
                  class: "pp-item-row",
                  onClick: ($event) => toggleDetail(p2)
                }, [
                  createBaseVNode("span", {
                    class: normalizeClass(["pp-state", p2.state === "running" ? "on" : "off"])
                  }, null, 2),
                  createBaseVNode("span", {
                    class: "pp-name",
                    title: p2.purpose || p2.name
                  }, toDisplayString(p2.name), 9, _hoisted_51),
                  createBaseVNode("span", {
                    class: normalizeClass(["pp-src", p2.source])
                  }, toDisplayString(p2.source), 3),
                  p2.scope === "global" ? (openBlock(), createElementBlock("span", _hoisted_52, "全局")) : createCommentVNode("", true),
                  p2.hasClient && p2.clientApproved ? (openBlock(), createElementBlock("span", _hoisted_53, "UI")) : p2.hasClient && p2.state === "running" ? (openBlock(), createElementBlock("span", _hoisted_54, "UI 待批准")) : p2.hasClient ? (openBlock(), createElementBlock("span", _hoisted_55, "UI")) : createCommentVNode("", true),
                  p2.tools && p2.tools.length ? (openBlock(), createElementBlock("span", {
                    key: 4,
                    class: "pp-count",
                    title: p2.tools.join(", ")
                  }, toDisplayString(p2.tools.length) + " 工具", 9, _hoisted_56)) : createCommentVNode("", true),
                  p2.hasClient && p2.clientApproved && uiSlotsOf(p2.name).length ? (openBlock(), createElementBlock(Fragment, { key: 5 }, [
                    createBaseVNode("span", {
                      class: normalizeClass(["pp-ui-label", { on: uiPluginActive(p2.name) }])
                    }, toDisplayString(uiPluginActive(p2.name) ? "UI 已启用" : "UI 未启用"), 3),
                    createBaseVNode("label", {
                      class: "pp-switch",
                      title: uiPluginActive(p2.name) ? "停用该插件的 UI（恢复内置界面）" : "启用该插件的 UI（替换对应界面区域）"
                    }, [
                      createBaseVNode("input", {
                        type: "checkbox",
                        checked: uiPluginActive(p2.name),
                        onChange: ($event) => toggleUiPlugin(p2, $event.target.checked),
                        onClick: _cache[10] || (_cache[10] = withModifiers(() => {
                        }, ["stop"]))
                      }, null, 40, _hoisted_58),
                      _cache[23] || (_cache[23] = createBaseVNode("span", { class: "pp-switch-track" }, null, -1))
                    ], 8, _hoisted_57)
                  ], 64)) : createCommentVNode("", true),
                  createVNode(SvgIcon, {
                    name: "chevron-right",
                    size: 12,
                    class: normalizeClass(["pp-chevron", { open: expanded[p2.name] }])
                  }, null, 8, ["class"])
                ], 8, _hoisted_50),
                expanded[p2.name] ? (openBlock(), createElementBlock("div", _hoisted_59, [
                  p2.purpose ? (openBlock(), createElementBlock("div", _hoisted_60, toDisplayString(p2.purpose), 1)) : createCommentVNode("", true),
                  p2.defId ? (openBlock(), createElementBlock("div", _hoisted_61, [
                    createTextVNode("定义: " + toDisplayString(p2.defId), 1),
                    p2.version ? (openBlock(), createElementBlock("span", _hoisted_62, " · " + toDisplayString(p2.version), 1)) : createCommentVNode("", true)
                  ])) : createCommentVNode("", true),
                  p2.provides && p2.provides.length ? (openBlock(), createElementBlock("div", _hoisted_63, "服务: " + toDisplayString(p2.provides.join(", ")), 1)) : createCommentVNode("", true),
                  p2.sections && p2.sections.length ? (openBlock(), createElementBlock("div", _hoisted_64, "提示片段: " + toDisplayString(p2.sections.join(", ")), 1)) : createCommentVNode("", true),
                  p2.tools && p2.tools.length ? (openBlock(), createElementBlock("div", _hoisted_65, [
                    createBaseVNode("div", _hoisted_66, "工具（" + toDisplayString(p2.tools.length) + "）· 开关控制 agent 可见性", 1),
                    (openBlock(true), createElementBlock(Fragment, null, renderList(p2.tools, (t) => {
                      return openBlock(), createElementBlock("div", {
                        key: t,
                        class: "pp-d-tool"
                      }, [
                        createBaseVNode("span", {
                          class: "pp-d-tname",
                          title: t
                        }, toDisplayString(t), 9, _hoisted_67),
                        createBaseVNode("label", {
                          class: "pp-switch",
                          title: pluginToolOn(p2, t) ? "对 agent 可见；点击禁用（不影响插件运行）" : "对 agent 不可见；点击启用"
                        }, [
                          createBaseVNode("input", {
                            type: "checkbox",
                            checked: pluginToolOn(p2, t),
                            onChange: ($event) => togglePluginTool(p2, t)
                          }, null, 40, _hoisted_69),
                          _cache[24] || (_cache[24] = createBaseVNode("span", { class: "pp-switch-track" }, null, -1))
                        ], 8, _hoisted_68)
                      ]);
                    }), 128))
                  ])) : createCommentVNode("", true),
                  p2.clientCode ? (openBlock(), createElementBlock("div", _hoisted_70, [
                    createBaseVNode("div", _hoisted_71, [
                      _cache[25] || (_cache[25] = createBaseVNode("span", null, "client 半源码", -1)),
                      createBaseVNode("button", {
                        class: "pp-icon-btn",
                        onClick: ($event) => copyText(p2.clientCode),
                        title: "复制"
                      }, [
                        createVNode(SvgIcon, {
                          name: "copy",
                          size: 11
                        })
                      ], 8, _hoisted_72)
                    ]),
                    createBaseVNode("pre", null, toDisplayString(p2.clientCode), 1)
                  ])) : createCommentVNode("", true),
                  createBaseVNode("div", _hoisted_73, [
                    p2.state === "running" ? (openBlock(), createElementBlock(Fragment, { key: 0 }, [
                      !(p2.hasClient && p2.clientApproved && uiSlotsOf(p2.name).length) ? (openBlock(), createElementBlock("button", {
                        key: 0,
                        class: "pp-btn",
                        title: "停止整个插件（其全部工具对 agent 不可见）；单工具请用上方工具开关",
                        onClick: ($event) => doAction(p2, "stop")
                      }, "停止插件", 8, _hoisted_74)) : createCommentVNode("", true)
                    ], 64)) : (openBlock(), createElementBlock("button", {
                      key: 1,
                      class: "pp-btn primary",
                      onClick: ($event) => doAction(p2, "start")
                    }, "启动插件", 8, _hoisted_75)),
                    p2.source === "js" ? (openBlock(), createElementBlock("button", {
                      key: 2,
                      class: "pp-btn danger",
                      onClick: ($event) => doAction(p2, "undefine")
                    }, "删除定义", 8, _hoisted_76)) : createCommentVNode("", true)
                  ])
                ])) : createCommentVNode("", true)
              ]);
            }), 128))
          ])
        ]);
      };
    }
  };
  const PluginPanel = /* @__PURE__ */ _export_sfc(_sfc_main$1, [["__scopeId", "data-v-5c15f6cc"]]);
  const _hoisted_1 = {
    key: 0,
    class: "slot-empty plugin-area-titlebar"
  };
  const _hoisted_2 = {
    key: 2,
    class: "slot-empty plugin-area-activitybar"
  };
  const _hoisted_3 = {
    key: 4,
    class: "slot-empty plugin-area-sidebar"
  };
  const _hoisted_4 = {
    key: 6,
    class: "slot-empty main-area"
  };
  const _hoisted_5 = {
    key: 10,
    class: "slot-empty app-statusbar-host"
  };
  const _hoisted_6 = {
    key: 12,
    class: "modals-empty"
  };
  const _hoisted_7 = { class: "plugin-escape-panel" };
  const _hoisted_8 = { class: "plugin-escape-head" };
  const _hoisted_9 = { class: "plugin-escape-body" };
  const _sfc_main = {
    __name: "ShellApp",
    setup(__props) {
      const panelMode = typeof window !== "undefined" && window.__DESKTOP_PANEL_MODE__ === true;
      const pluginsOpen = /* @__PURE__ */ ref(false);
      const slots = {
        titlebar: useSingleSlot("titlebar"),
        activitybar: useSingleSlot("activitybar"),
        sidebar: useSingleSlot("sidebar"),
        editor: useSingleSlot("editor"),
        rightPanel: useSingleSlot("right-panel"),
        statusbar: useSingleSlot("statusbar"),
        modals: useSingleSlot("modals")
      };
      for (const s of Object.values(slots)) s.init();
      onMounted(async () => {
        for (const s of Object.values(slots)) s.start();
        desktopPrefetch();
        initAppGlobals();
        loadWsList();
        try {
          await loadAssemblyFile();
          const list = await api.listPlugins() || [];
          for (const p2 of list) {
            if (p2.hasClient && !p2.clientCode) {
              try {
                const d = await api.getPluginDetail(p2.name);
                if (d && d.clientCode) p2.clientCode = d.clientCode;
              } catch (e) {
              }
            }
          }
          await syncClientHalves(list);
          startPolling();
        } catch (e) {
          console.warn("[shell] client 半装载失败", e);
        }
      });
      onUnmounted(() => {
        for (const s of Object.values(slots)) s.stop();
        stopPolling();
        cleanupAppGlobals();
      });
      return (_ctx, _cache) => {
        return openBlock(), createElementBlock(Fragment, null, [
          createBaseVNode("div", {
            class: normalizeClass(["app-root", { "panel-only": unref(panelMode) }])
          }, [
            !unref(panelMode) && !slots.titlebar.owner.value ? (openBlock(), createElementBlock("div", _hoisted_1, [
              _cache[9] || (_cache[9] = createBaseVNode("span", null, "标题栏未装配（ui-titlebar）", -1)),
              createBaseVNode("button", {
                class: "escape-link",
                onClick: _cache[0] || (_cache[0] = ($event) => pluginsOpen.value = true)
              }, "打开插件面板")
            ])) : !unref(panelMode) ? (openBlock(), createElementBlock("div", {
              key: 1,
              ref: slots.titlebar.hostRef,
              class: "plugin-slot-host plugin-area-titlebar"
            }, null, 512)) : createCommentVNode("", true),
            !unref(panelMode) && !slots.activitybar.owner.value ? (openBlock(), createElementBlock("div", _hoisted_2, [
              _cache[10] || (_cache[10] = createBaseVNode("span", null, "⦿", -1)),
              createBaseVNode("button", {
                class: "escape-link",
                onClick: _cache[1] || (_cache[1] = ($event) => pluginsOpen.value = true)
              }, "面板")
            ])) : !unref(panelMode) ? (openBlock(), createElementBlock("div", {
              key: 3,
              ref: slots.activitybar.hostRef,
              class: "plugin-slot-host plugin-area-activitybar"
            }, null, 512)) : createCommentVNode("", true),
            !unref(panelMode) && unref(state).sidebarVisible && !slots.sidebar.owner.value ? (openBlock(), createElementBlock("div", _hoisted_3, [
              _cache[11] || (_cache[11] = createBaseVNode("span", null, "侧栏未装配（ui-sidebar）", -1)),
              createBaseVNode("button", {
                class: "escape-link",
                onClick: _cache[2] || (_cache[2] = ($event) => pluginsOpen.value = true)
              }, "打开插件面板")
            ])) : !unref(panelMode) && unref(state).sidebarVisible ? (openBlock(), createElementBlock("div", {
              key: 5,
              ref: slots.sidebar.hostRef,
              class: "plugin-slot-host plugin-area-sidebar"
            }, null, 512)) : createCommentVNode("", true),
            !unref(panelMode) && !unref(state).focusMode && !slots.editor.owner.value ? (openBlock(), createElementBlock("div", _hoisted_4, [
              _cache[12] || (_cache[12] = createBaseVNode("span", null, "编辑器未装配（ui-editor）", -1)),
              createBaseVNode("button", {
                class: "escape-link",
                onClick: _cache[3] || (_cache[3] = ($event) => pluginsOpen.value = true)
              }, "打开插件面板")
            ])) : !unref(panelMode) && !unref(state).focusMode ? (openBlock(), createElementBlock("div", {
              key: 7,
              ref: slots.editor.hostRef,
              class: "plugin-slot-host main-area"
            }, null, 512)) : createCommentVNode("", true),
            (unref(state).rightPanelVisible || unref(panelMode)) && !slots.rightPanel.owner.value ? (openBlock(), createElementBlock("div", {
              key: 8,
              class: normalizeClass(["slot-empty right-container", { "focus-mode": unref(state).focusMode, "panel-only": unref(panelMode) }])
            }, [
              _cache[13] || (_cache[13] = createBaseVNode("span", null, "对话面板未装配（ui-right-panel）", -1)),
              createBaseVNode("button", {
                class: "escape-link",
                onClick: _cache[4] || (_cache[4] = ($event) => pluginsOpen.value = true)
              }, "打开插件面板")
            ], 2)) : unref(state).rightPanelVisible || unref(panelMode) ? (openBlock(), createElementBlock("div", {
              key: 9,
              ref: slots.rightPanel.hostRef,
              class: normalizeClass(["plugin-slot-host right-container", { "focus-mode": unref(state).focusMode, "panel-only": unref(panelMode) }])
            }, null, 2)) : createCommentVNode("", true),
            !unref(panelMode) && !slots.statusbar.owner.value ? (openBlock(), createElementBlock("div", _hoisted_5, [
              _cache[14] || (_cache[14] = createBaseVNode("span", null, "状态栏未装配（ui-statusbar）", -1)),
              createBaseVNode("button", {
                class: "escape-link",
                onClick: _cache[5] || (_cache[5] = ($event) => pluginsOpen.value = true)
              }, "打开插件面板")
            ])) : !unref(panelMode) ? (openBlock(), createElementBlock("div", {
              key: 11,
              ref: slots.statusbar.hostRef,
              class: "plugin-slot-host app-statusbar-host"
            }, null, 512)) : createCommentVNode("", true),
            !slots.modals.owner.value ? (openBlock(), createElementBlock("div", _hoisted_6)) : (openBlock(), createElementBlock("div", {
              key: 13,
              ref: slots.modals.hostRef,
              class: "plugin-slot-host modals-host"
            }, null, 512))
          ], 2),
          (openBlock(), createBlock(Teleport, { to: "body" }, [
            !unref(panelMode) ? (openBlock(), createElementBlock("button", {
              key: 0,
              class: "plugin-escape-btn",
              title: "插件面板（壳级入口，不受插件停用影响）",
              onClick: _cache[6] || (_cache[6] = ($event) => pluginsOpen.value = true)
            }, [..._cache[15] || (_cache[15] = [
              createBaseVNode("svg", {
                viewBox: "0 0 16 16",
                width: "15",
                height: "15",
                fill: "none",
                stroke: "currentColor",
                "stroke-width": "1.5"
              }, [
                createBaseVNode("rect", {
                  x: "2",
                  y: "6",
                  width: "12",
                  height: "8",
                  rx: "1.5"
                }),
                createBaseVNode("path", { d: "M5 6V4.5a3 3 0 0 1 6 0V6" })
              ], -1)
            ])])) : createCommentVNode("", true),
            pluginsOpen.value ? (openBlock(), createElementBlock("div", {
              key: 1,
              class: "plugin-escape-overlay",
              onClick: _cache[8] || (_cache[8] = withModifiers(($event) => pluginsOpen.value = false, ["self"]))
            }, [
              createBaseVNode("div", _hoisted_7, [
                createBaseVNode("div", _hoisted_8, [
                  _cache[16] || (_cache[16] = createBaseVNode("span", null, "插件面板（壳级入口）", -1)),
                  createBaseVNode("button", {
                    class: "plugin-escape-close",
                    title: "关闭",
                    onClick: _cache[7] || (_cache[7] = ($event) => pluginsOpen.value = false)
                  }, "✕")
                ]),
                createBaseVNode("div", _hoisted_9, [
                  createVNode(PluginPanel)
                ])
              ])
            ])) : createCommentVNode("", true)
          ]))
        ], 64);
      };
    }
  };
  const ShellApp = /* @__PURE__ */ _export_sfc(_sfc_main, [["__scopeId", "data-v-ed63d1b3"]]);
  window.__PAIRCODE_CORE = { Vue, api, uiState, pluginRuntime: pluginRuntime$1, agentEvents, actions };
  createApp(ShellApp).mount("#app");
})();
