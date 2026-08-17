// ═══════════════════════════════════════════════════════════════
// ui-quick-exec — client 半：(ui) => void
//
// 标题栏 titlebar-right 槽位渲染「快速执行」按钮：
//   · 点击展开下拉菜单（每次打开动态 invoke getCommands 拉取
//     工作区配置 .pair/quick-exec.json 的命令列表 → 改配置/切工作区即生效）
//   · 菜单第一项「配置命令」→ 打开命令管理弹窗（增删改查 + 上移/下移排序）
//   · 点击命令项 → ui.invoke runCommand 在宿主执行（工作区根 cwd）→ 结果弹窗
// 纯 DOM 实现（不依赖 Vue bundle），CSS 变量跟随 IDE 主题。
// ═══════════════════════════════════════════════════════════════
(ui) => {
  const PLUGIN = 'ui-quick-exec'
  const SLOT = 'titlebar-right'

  // ─── SVG 图标（禁止 emoji，全部内联 SVG）─────────────────
  const SVG = {
    bolt: '<svg viewBox="0 0 16 16" width="11" height="11" fill="currentColor" aria-hidden="true"><path d="M8.5 1 3.2 9h4l-.7 6L12 7H8l.5-6z"/></svg>',
    caret: '<svg viewBox="0 0 16 16" width="8" height="8" fill="currentColor" aria-hidden="true"><path d="M4 6l4 4 4-4H4z"/></svg>',
    gear: '<svg viewBox="0 0 16 16" width="12" height="12" fill="currentColor" aria-hidden="true"><path d="M7 1h2l.4 1.8c.5.2 1 .5 1.5.9l1.7-.6 1 1.7-1.3 1.3c.1.4.2.9.2 1.4s-.1 1-.2 1.4l1.3 1.3-1 1.7-1.7-.6c-.5.4-1 .7-1.5.9L9 15H7l-.4-1.8a5.5 5.5 0 0 1-1.5-.9l-1.7.6-1-1.7 1.3-1.3c-.1-.4-.2-.9-.2-1.4s.1-1 .2-1.4L2.1 5.3l1-1.7 1.7.6c.5-.4 1-.7 1.5-.9L7 1zm1 4.5A2.5 2.5 0 1 0 8 10.5 2.5 2.5 0 0 0 8 5.5z"/></svg>',
    play: '<svg viewBox="0 0 16 16" width="11" height="11" fill="currentColor" aria-hidden="true"><path d="M4 2l10 6-10 6V2z"/></svg>',
    close: '<svg viewBox="0 0 16 16" width="12" height="12" fill="currentColor" aria-hidden="true"><path d="M4.2 2.8 8 6.6l3.8-3.8 1.4 1.4L9.4 8l3.8 3.8-1.4 1.4L8 9.4l-3.8 3.8-1.4-1.4L6.6 8 2.8 4.2l1.4-1.4z"/></svg>',
    empty: '<svg viewBox="0 0 16 16" width="20" height="20" fill="currentColor" aria-hidden="true"><path d="M2 3h12v10H2V3zm2 2v2h8V5H4zm0 4v2h5V9H4z"/></svg>',
  }

  // 全局样式（一次注入，幂等）
  const STYLE_ID = 'qexec-style'
  if (!document.getElementById(STYLE_ID)) {
    const st = document.createElement('style')
    st.id = STYLE_ID
    st.textContent = `
.qexec-btn{display:inline-flex;align-items:center;gap:4px;height:22px;padding:0 8px;font-size:11px;color:var(--text-secondary,#c9d1d9);background:var(--bg-tertiary,#21262d);border:1px solid var(--border-color,#30363d);border-radius:4px;cursor:pointer;white-space:nowrap}
.qexec-btn:hover{color:var(--text-primary,#e6edf3);background:var(--bg-hover,#2d333b);border-color:var(--accent-color,#4f8cff)}
.qexec-btn.busy{opacity:.6;pointer-events:none}
.qexec-caret{margin-left:2px;opacity:.7}
.qexec-menu{position:fixed;z-index:9000;min-width:280px;max-width:420px;background:var(--bg-secondary,#1c2128);border:1px solid var(--border-color,#30363d);border-radius:8px;box-shadow:0 8px 28px rgba(0,0,0,.45);overflow:hidden;font-size:12px;color:var(--text-primary,#e6edf3)}
.qexec-menu-head{display:flex;align-items:center;justify-content:space-between;padding:7px 10px;font-size:11px;color:var(--text-muted,#8b949e);border-bottom:1px solid var(--border-color,#30363d);background:var(--bg-tertiary,#21262d)}
.qexec-menu-head .qexec-ws{font-weight:600;color:var(--text-secondary,#c9d1d9);max-width:180px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
.qexec-cfg-item{display:flex;align-items:center;gap:7px;padding:8px 10px;cursor:pointer;color:var(--text-primary,#e6edf3)}
.qexec-cfg-item:hover{background:var(--bg-hover,#2d333b)}
.qexec-cfg-item svg{color:var(--accent-color,#4f8cff)}
.qexec-sep{height:1px;background:var(--border-color,#30363d);margin:2px 0}
.qexec-list{max-height:calc(60vh - 100px);overflow-y:auto}
.qexec-item{display:flex;align-items:center;gap:8px;padding:7px 10px;cursor:pointer;border-left:2px solid transparent}
.qexec-item:hover{background:var(--bg-hover,#2d333b);border-left-color:var(--accent-color,#4f8cff)}
.qexec-item-main{flex:1;min-width:0}
.qexec-item-name{display:block;font-size:12px;color:var(--text-primary,#e6edf3);white-space:nowrap;overflow:hidden;text-overflow:ellipsis}
.qexec-item-cmd{display:block;font-size:11px;color:var(--text-muted,#8b949e);font-family:Consolas,Menlo,monospace;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}
.qexec-item-run{display:flex;align-items:center;padding:3px;color:var(--text-muted,#8b949e);border-radius:3px;flex-shrink:0}
.qexec-item:hover .qexec-item-run{color:var(--accent-color,#4f8cff)}
.qexec-item-run:hover{background:rgba(79,140,255,.15)}
.qexec-empty{padding:18px 10px;text-align:center;color:var(--text-muted,#8b949e);font-size:12px}
.qexec-empty svg{display:block;margin:0 auto 6px;opacity:.5}
.qexec-empty a{color:var(--accent-color,#4f8cff);cursor:pointer;text-decoration:underline}
.qexec-menu-foot{padding:6px 10px;font-size:10px;color:var(--text-muted,#8b949e);border-top:1px solid var(--border-color,#30363d);display:flex;justify-content:space-between;align-items:center}
.qexec-busy{display:inline-flex;align-items:center;gap:5px;color:var(--accent-color,#4f8cff)}
.qexec-spin{width:9px;height:9px;border:2px solid rgba(79,140,255,.3);border-top-color:var(--accent-color,#4f8cff);border-radius:50%;animation:qexec-rot .8s linear infinite}
@keyframes qexec-rot{to{transform:rotate(360deg)}}
.qexec-overlay{position:fixed;inset:0;z-index:9500;background:rgba(0,0,0,.45);display:flex;align-items:flex-start;justify-content:center;padding:9vh 16px 16px}
.qexec-modal{width:100%;max-width:560px;background:var(--bg-secondary,#1c2128);border:1px solid var(--border-color,#30363d);border-radius:10px;box-shadow:0 12px 40px rgba(0,0,0,.5);overflow:hidden;display:flex;flex-direction:column;max-height:80vh}
.qexec-modal-head{display:flex;align-items:center;justify-content:space-between;padding:9px 12px;background:var(--bg-tertiary,#21262d);border-bottom:1px solid var(--border-color,#30363d);font-size:13px;font-weight:600;color:var(--text-primary,#e6edf3)}
.qexec-ico-btn{display:flex;align-items:center;justify-content:center;width:22px;height:22px;border:none;background:none;color:var(--text-muted,#8b949e);cursor:pointer;border-radius:4px;font-size:14px;line-height:1;padding:0}
.qexec-ico-btn:hover{color:var(--text-primary,#e6edf3);background:var(--bg-hover,#2d333b)}
.qexec-modal-body{padding:10px 12px;overflow-y:auto;flex:1}
.qexec-hint{font-size:11px;color:var(--text-muted,#8b949e);margin:0 0 8px;line-height:1.5}
.qexec-cfg-rows{display:flex;flex-direction:column;gap:6px}
.qexec-row{display:flex;align-items:center;gap:8px;padding:6px 8px;border:1px solid var(--border-color,#30363d);border-radius:6px;background:var(--bg-tertiary,#21262d)}
.qexec-row.editing{border-color:var(--accent-color,#4f8cff)}
.qexec-row-main{flex:1;min-width:0}
.qexec-row-name{display:block;font-size:12px;color:var(--text-primary,#e6edf3);white-space:nowrap;overflow:hidden;text-overflow:ellipsis}
.qexec-row-cmd{display:block;font-size:11px;color:var(--text-muted,#8b949e);font-family:Consolas,Menlo,monospace;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}
.qexec-row-ops{display:flex;gap:2px;flex-shrink:0}
.qexec-op{display:flex;align-items:center;justify-content:center;min-width:20px;height:20px;padding:0 4px;font-size:11px;color:var(--text-muted,#8b949e);background:none;border:none;border-radius:3px;cursor:pointer}
.qexec-op:hover{color:var(--text-primary,#e6edf3);background:var(--bg-hover,#2d333b)}
.qexec-op.danger:hover{color:#f85149;background:rgba(248,81,73,.15)}
.qexec-add-btn{margin-top:8px;width:100%;padding:6px;font-size:12px;color:var(--accent-color,#4f8cff);background:none;border:1px dashed var(--border-color,#30363d);border-radius:6px;cursor:pointer}
.qexec-add-btn:hover{background:rgba(79,140,255,.08);border-color:var(--accent-color,#4f8cff)}
.qexec-edit{margin-top:8px;padding:8px;border:1px solid var(--accent-color,#4f8cff);border-radius:6px;background:var(--bg-tertiary,#21262d);display:flex;flex-direction:column;gap:6px}
.qexec-edit input{width:100%;box-sizing:border-box;padding:5px 8px;font-size:12px;color:var(--text-primary,#e6edf3);background:var(--bg-primary,#161b22);border:1px solid var(--border-color,#30363d);border-radius:4px;outline:none}
.qexec-edit input:focus{border-color:var(--accent-color,#4f8cff)}
.qexec-edit label{font-size:10px;color:var(--text-muted,#8b949e)}
.qexec-edit-ops{display:flex;gap:6px;justify-content:flex-end}
.qexec-btn2{padding:4px 12px;font-size:12px;border-radius:5px;cursor:pointer;border:1px solid var(--border-color,#30363d);background:var(--bg-tertiary,#21262d);color:var(--text-secondary,#c9d1d9)}
.qexec-btn2:hover{background:var(--bg-hover,#2d333b);color:var(--text-primary,#e6edf3)}
.qexec-btn2.primary{border-color:var(--accent-color,#4f8cff);background:var(--accent-color,#4f8cff);color:#fff}
.qexec-btn2.primary:hover{filter:brightness(1.1)}
.qexec-btn2:disabled{opacity:.5;cursor:not-allowed}
.qexec-modal-foot{display:flex;align-items:center;justify-content:flex-end;gap:8px;padding:9px 12px;border-top:1px solid var(--border-color,#30363d);background:var(--bg-tertiary,#21262d)}
.qexec-msg{flex:1;font-size:11px;color:#f85149}
.qexec-msg.ok{color:#3fb950}
.qexec-res-body{padding:12px;overflow-y:auto}
.qexec-res-name{font-size:13px;font-weight:600;color:var(--text-primary,#e6edf3);margin-bottom:4px}
.qexec-res-cmd{display:block;font-size:11px;color:var(--text-muted,#8b949e);font-family:Consolas,Menlo,monospace;background:var(--bg-tertiary,#21262d);border:1px solid var(--border-color,#30363d);border-radius:5px;padding:5px 8px;margin-bottom:8px;white-space:pre-wrap;word-break:break-all}
.qexec-res-status{display:inline-flex;align-items:center;gap:6px;font-size:11px;font-weight:600;padding:2px 8px;border-radius:10px;margin-bottom:8px}
.qexec-res-status.ok{color:#3fb950;background:rgba(63,185,80,.12)}
.qexec-res-status.err{color:#f85149;background:rgba(248,81,73,.12)}
.qexec-pre{margin:0 0 8px;max-height:260px;overflow:auto;padding:8px 10px;font-size:11px;line-height:1.5;font-family:Consolas,Menlo,monospace;white-space:pre-wrap;word-break:break-all;background:var(--bg-primary,#161b22);border:1px solid var(--border-color,#30363d);border-radius:6px;color:var(--text-primary,#e6edf3)}
.qexec-pre.err{border-color:rgba(248,81,73,.5);color:#f85149}
.qexec-empty-pre{color:var(--text-muted,#8b949e);font-style:italic}
`
    document.head.appendChild(st)
  }

  ui.registerSlot({
    slotId: SLOT,
    title: '快速执行（工作区配置命令菜单）',
    kind: 'list',
    render(el) {
      // ── 状态 ──
      let menuOpen = false
      let busy = false
      let cmds = []
      let wsName = ''
      let cfgOpen = false
      let cfgCmds = []
      let editing = null // { idx, name, command }；idx<0 = 新增模式
      let cfgMsg = ''
      let cfgMsgOk = false
      let saving = false
      let resOpen = false
      let resData = null
      let menuEl = null
      let cfgEl = null
      let resEl = null
      let nameInput = null
      let cmdInput = null

      const esc = (s) => String(s == null ? '' : s)
        .replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;').replace(/'/g, '&#39;')

      // ── 按钮骨架 ──
      el.innerHTML = '<div class="qexec">' +
        '<button type="button" class="qexec-btn" title="快速执行（工作区配置命令）">' +
        SVG.bolt + '<span>快速执行</span><span class="qexec-caret">' + SVG.caret + '</span>' +
        '</button></div>'
      const btn = el.querySelector('.qexec-btn')

      // ── 菜单 ──
      const ensureMenu = () => {
        if (!menuEl) {
          menuEl = document.createElement('div')
          menuEl.className = 'qexec-menu'
          menuEl.addEventListener('click', onMenuClick)
          document.body.appendChild(menuEl)
        }
        return menuEl
      }
      const closeMenu = () => {
        menuOpen = false
        if (menuEl) { menuEl.style.display = 'none' }
        btn.classList.remove('busy')
      }
      async function openMenu() {
        btn.classList.add('busy')
        try {
          const r = await ui.invoke(PLUGIN, 'getCommands')
          cmds = (r && Array.isArray(r.commands)) ? r.commands : []
          wsName = (r && r.workspace) || ''
        } catch (e) {
          cmds = []
          wsName = ''
          console.warn('[ui-quick-exec] 拉取命令失败', e)
        }
        btn.classList.remove('busy')
        menuOpen = true
        renderMenu()
      }
      function renderMenu() {
        const m = ensureMenu()
        const rect = btn.getBoundingClientRect()
        m.style.top = (rect.bottom + 6) + 'px'
        m.style.right = Math.max(8, window.innerWidth - rect.right) + 'px'
        m.style.display = 'block'
        const listHtml = cmds.length
          ? cmds.map((c, i) =>
              '<div class="qexec-item" data-act="run" data-idx="' + i + '">' +
                '<div class="qexec-item-main">' +
                  '<span class="qexec-item-name">' + esc(c.name) + '</span>' +
                  '<span class="qexec-item-cmd">' + esc(c.command) + '</span>' +
                '</div>' +
                '<span class="qexec-item-run" title="执行">' + SVG.play + '</span>' +
              '</div>').join('')
          : '<div class="qexec-empty">' + SVG.empty +
              '<div>暂无命令</div>' +
              '<div>点击「<a data-act="config">配置命令</a>」添加</div>' +
            '</div>'
        m.innerHTML =
          '<div class="qexec-menu-head">' +
            '<span class="qexec-ws" title="' + esc(wsName) + '">' + esc(wsName || '工作区') + '</span>' +
            '<span>' + cmds.length + ' 条命令</span>' +
          '</div>' +
          '<div class="qexec-cfg-item" data-act="config">' + SVG.gear + '<span>配置命令</span></div>' +
          '<div class="qexec-sep"></div>' +
          '<div class="qexec-list">' + listHtml + '</div>' +
          '<div class="qexec-menu-foot">' +
            (busy ? '<span class="qexec-busy"><span class="qexec-spin"></span>执行中…</span>'
                 : '<span>命令在工作区根目录执行</span>') +
          '</div>'
      }
      function onMenuClick(e) {
        const t = e.target.closest('[data-act]')
        if (!t) return
        const act = t.getAttribute('data-act')
        e.stopPropagation()
        if (act === 'config') { closeMenu(); openCfg() }
        else if (act === 'run') { runCmd(parseInt(t.getAttribute('data-idx'), 10)) }
      }

      // ── 执行命令 ──
      async function runCmd(idx) {
        const c = cmds[idx]
        if (!c || busy) return
        busy = true
        renderMenu()
        try {
          const r = await ui.invoke(PLUGIN, 'runCommand', { command: c.command })
          resData = {
            name: c.name, command: c.command,
            output: (r && r.output) || '', error: (r && r.error) || '', ok: !!(r && r.ok),
          }
        } catch (e) {
          resData = { name: c.name, command: c.command, output: '', error: String((e && e.message) || e), ok: false }
        }
        busy = false
        closeMenu()
        openRes()
      }

      // ── 配置弹窗 ──
      function openCfg() {
        cfgOpen = true
        cfgCmds = cmds.map((c) => ({ name: c.name, command: c.command }))
        editing = null
        cfgMsg = ''
        cfgMsgOk = false
        renderCfg()
      }
      function closeCfg() {
        cfgOpen = false
        if (cfgEl) { cfgEl.remove(); cfgEl = null }
        editing = null
      }
      function renderCfg() {
        const ov = ensureCfg()
        const listHtml = cfgCmds.length
          ? cfgCmds.map((c, i) =>
              '<div class="qexec-row' + (editing && editing.idx === i ? ' editing' : '') + '">' +
                '<div class="qexec-row-main">' +
                  '<span class="qexec-row-name">' + esc(c.name) + '</span>' +
                  '<span class="qexec-row-cmd">' + esc(c.command) + '</span>' +
                '</div>' +
                '<div class="qexec-row-ops">' +
                  '<button type="button" class="qexec-op" data-op="up" data-idx="' + i + '" title="上移" ' + (i === 0 ? 'disabled' : '') + '>↑</button>' +
                  '<button type="button" class="qexec-op" data-op="down" data-idx="' + i + '" title="下移" ' + (i === cfgCmds.length - 1 ? 'disabled' : '') + '>↓</button>' +
                  '<button type="button" class="qexec-op" data-op="edit" data-idx="' + i + '" title="编辑">编辑</button>' +
                  '<button type="button" class="qexec-op danger" data-op="del" data-idx="' + i + '" title="删除">删除</button>' +
                '</div>' +
              '</div>').join('')
          : '<div class="qexec-empty">' + SVG.empty + '<div>暂无命令，点击下方「新增命令」添加</div></div>'
        const editHtml = editing
          ? '<div class="qexec-edit">' +
              '<div><label>命令名称</label><input type="text" class="qexec-in-name" placeholder="如：构建" value="' + esc(editing.name) + '" /></div>' +
              '<div><label>执行命令</label><input type="text" class="qexec-in-cmd" placeholder="如：go build ./cmd/companion" value="' + esc(editing.command) + '" /></div>' +
              '<div class="qexec-edit-ops">' +
                '<button type="button" class="qexec-btn2" data-act="edit-cancel">取消</button>' +
                '<button type="button" class="qexec-btn2 primary" data-act="edit-ok">确定</button>' +
              '</div>' +
            '</div>'
          : ''
        ov.querySelector('.qexec-body').innerHTML =
          '<p class="qexec-hint">命令保存在当前工作区 <code>.pair/quick-exec.json</code>，重启/刷新后仍保留；菜单每次打开时自动加载。</p>' +
          '<div class="qexec-cfg-rows">' + listHtml + '</div>' +
          editHtml +
          '<button type="button" class="qexec-add-btn" data-act="add">+ 新增命令</button>'
        const foot = ov.querySelector('.qexec-foot')
        foot.querySelector('.qexec-msg').textContent = cfgMsg
        foot.querySelector('.qexec-msg').className = 'qexec-msg' + (cfgMsgOk ? ' ok' : '')
        foot.querySelector('[data-act="save"]').disabled = saving
        // 编辑表单取值引用
        nameInput = ov.querySelector('.qexec-in-name')
        cmdInput = ov.querySelector('.qexec-in-cmd')
      }
      const ensureCfg = () => {
        if (!cfgEl) {
          cfgEl = document.createElement('div')
          cfgEl.className = 'qexec-overlay'
          cfgEl.innerHTML =
            '<div class="qexec-modal">' +
              '<div class="qexec-modal-head">' +
                '<span>配置命令 — 快速执行</span>' +
                '<button type="button" class="qexec-ico-btn" data-act="close" title="关闭">' + SVG.close + '</button>' +
              '</div>' +
              '<div class="qexec-modal-body qexec-body"></div>' +
              '<div class="qexec-modal-foot qexec-foot">' +
                '<span class="qexec-msg"></span>' +
                '<button type="button" class="qexec-btn2" data-act="cancel">取消</button>' +
                '<button type="button" class="qexec-btn2 primary" data-act="save">保存</button>' +
              '</div>' +
            '</div>'
          cfgEl.addEventListener('click', onCfgClick)
          document.body.appendChild(cfgEl)
        }
        return cfgEl
      }
      function onCfgClick(e) {
        const t = e.target.closest('[data-act],[data-op]')
        if (!t) return
        const act = t.getAttribute('data-act')
        const op = t.getAttribute('data-op')
        const idx = parseInt(t.getAttribute('data-idx'), 10)
        if (act === 'close' || act === 'cancel') { closeCfg(); return }
        if (act === 'add') { editing = { idx: -1, name: '', command: '' }; cfgMsg = ''; cfgMsgOk = false; renderCfg(); focusEdit() }
        else if (act === 'edit-ok') { applyEdit() }
        else if (act === 'edit-cancel') { editing = null; renderCfg() }
        else if (act === 'save') { saveCfg() }
        else if (op === 'edit') { editing = { idx, name: cfgCmds[idx].name, command: cfgCmds[idx].command }; cfgMsg = ''; cfgMsgOk = false; renderCfg(); focusEdit() }
        else if (op === 'del') { cfgCmds.splice(idx, 1); if (editing && editing.idx === idx) editing = null; cfgMsg = ''; cfgMsgOk = false; renderCfg() }
        else if (op === 'up' && idx > 0) { const t2 = cfgCmds[idx]; cfgCmds[idx] = cfgCmds[idx - 1]; cfgCmds[idx - 1] = t2; if (editing && editing.idx === idx) editing.idx = idx - 1; renderCfg() }
        else if (op === 'down' && idx < cfgCmds.length - 1) { const t2 = cfgCmds[idx]; cfgCmds[idx] = cfgCmds[idx + 1]; cfgCmds[idx + 1] = t2; if (editing && editing.idx === idx) editing.idx = idx + 1; renderCfg() }
      }
      const focusEdit = () => {
        if (nameInput) nameInput.focus()
      }
      function applyEdit() {
        const name = nameInput ? nameInput.value.trim() : ''
        const command = cmdInput ? cmdInput.value.trim() : ''
        if (!name || !command) { cfgMsg = '名称与命令均不能为空'; cfgMsgOk = false; renderCfg(); return }
        if (editing.idx < 0) cfgCmds.push({ name, command })
        else cfgCmds[editing.idx] = { name, command }
        editing = null
        cfgMsg = ''; cfgMsgOk = false
        renderCfg()
      }
      async function saveCfg() {
        saving = true
        renderCfg()
        try {
          const clean = cfgCmds.filter((c) => c && c.name && c.command)
          await ui.invoke(PLUGIN, 'saveCommands', { commands: clean })
          cmds = clean.map((c) => ({ name: c.name, command: c.command }))
          cfgMsg = '已保存 ' + clean.length + ' 条命令'; cfgMsgOk = true
          saving = false
          renderCfg()
          setTimeout(() => { closeCfg() }, 500)
        } catch (e) {
          saving = false
          cfgMsg = '保存失败: ' + String((e && e.message) || e); cfgMsgOk = false
          renderCfg()
        }
      }

      // ── 结果弹窗 ──
      function openRes() {
        resOpen = true
        renderRes()
      }
      function closeRes() {
        resOpen = false
        if (resEl) { resEl.remove(); resEl = null }
      }
      function renderRes() {
        if (!resData) return
        const ov = ensureRes()
        const outHtml = resData.output
          ? '<pre class="qexec-pre">' + esc(resData.output) + '</pre>'
          : '<pre class="qexec-pre qexec-empty-pre">（无输出）</pre>'
        const errHtml = resData.error
          ? '<pre class="qexec-pre err">' + esc(resData.error) + '</pre>' : ''
        ov.querySelector('.qexec-res-body').innerHTML =
          '<div class="qexec-res-name">' + esc(resData.name) + '</div>' +
          '<code class="qexec-res-cmd">' + esc(resData.command) + '</code>' +
          '<span class="qexec-res-status ' + (resData.ok ? 'ok' : 'err') + '">' +
            (resData.ok ? '执行成功' : '执行失败') + '</span>' +
          outHtml + errHtml
      }
      const ensureRes = () => {
        if (!resEl) {
          resEl = document.createElement('div')
          resEl.className = 'qexec-overlay'
          resEl.innerHTML =
            '<div class="qexec-modal">' +
              '<div class="qexec-modal-head">' +
                '<span>执行结果</span>' +
                '<button type="button" class="qexec-ico-btn" data-act="close" title="关闭">' + SVG.close + '</button>' +
              '</div>' +
              '<div class="qexec-modal-body qexec-res-body"></div>' +
            '</div>'
          resEl.addEventListener('click', (e) => {
            const t = e.target.closest('[data-act]')
            if (t && t.getAttribute('data-act') === 'close') closeRes()
          })
          document.body.appendChild(resEl)
        }
        return resEl
      }

      // ── 全局事件 ──
      const onDocMouse = (e) => {
        // 点按钮自身由 btn click 处理；点菜单内不关；其余关闭菜单
        if (menuOpen && menuEl && !menuEl.contains(e.target) && !btn.contains(e.target)) closeMenu()
      }
      const onKey = (e) => {
        if (e.key === 'Escape') {
          if (resOpen) closeRes()
          else if (cfgOpen) closeCfg()
          else if (menuOpen) closeMenu()
        }
      }
      btn.addEventListener('click', (e) => {
        e.stopPropagation()
        if (menuOpen) closeMenu()
        else openMenu()
      })
      document.addEventListener('mousedown', onDocMouse, true)
      document.addEventListener('keydown', onKey)
      window.addEventListener('resize', closeMenu)

      // ── cleanup：槽位重渲染/插件卸载前调用 ──
      return () => {
        document.removeEventListener('mousedown', onDocMouse, true)
        document.removeEventListener('keydown', onKey)
        window.removeEventListener('resize', closeMenu)
        if (menuEl) { menuEl.remove(); menuEl = null }
        if (cfgEl) { cfgEl.remove(); cfgEl = null }
        if (resEl) { resEl.remove(); resEl = null }
        el.innerHTML = ''
      }
    },
  })
}
