// source-update — client 半：更新提示弹窗（纯 DOM；SVG 图标不用 emoji；主题变量跟随 IDE）。
//
// 行为：host 广播 'ui:source-update/state' 且 hasUpdate -> 居中弹窗
// （立即更新 / 稍后 / 忽略此版本，可手动关闭）；点「立即更新」调 host
// 分离启动更新脚本后，换右下角「更新中」气泡；一切异常输出到 console。
(ui) => {
  const PLUGIN = 'source-update';
  const S = { dialog: null, bubble: null, dialogKey: '', st: null };
  const esc = (s) => String(s == null ? '' : s).replace(/[&<>"']/g, (c) => ({
    '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;',
  }[c]));
  const SVG = {
    up: '<svg viewBox="0 0 16 16" width="13" height="13" fill="currentColor" aria-hidden="true"><path d="M8 2l5 5h-3v7H6V7H3l5-5z"/></svg>',
    close: '<svg viewBox="0 0 16 16" width="12" height="12" fill="currentColor" aria-hidden="true"><path d="M4.2 2.8 8 6.6l3.8-3.8 1.4 1.4L9.4 8l3.8 3.8-1.4 1.4L8 9.4l-3.8 3.8-1.4-1.4L6.6 8 2.8 4.2l1.4-1.4z"/></svg>',
    spin: '<svg viewBox="0 0 16 16" width="12" height="12" fill="currentColor" aria-hidden="true"><path d="M8 1a7 7 0 1 0 7 7h-1.6A5.4 5.4 0 1 1 8 2.6V1z"/></svg>',
  };
  const rm = (k) => { const e = S[k]; if (e && e.parentNode) e.parentNode.removeChild(e); S[k] = null; };

  const btn = (act, label, primary) =>
    '<button data-act="' + act + '" style="cursor:pointer;border-radius:6px;padding:6px 14px;font-size:12.5px;' +
    (primary
      ? 'background:var(--color-info,#3b82f6);border:1px solid var(--color-info,#3b82f6);color:#fff;'
      : 'background:transparent;border:1px solid var(--color-input-border,#555);color:var(--color-fg,#ddd);') +
    '">' + esc(label) + '</button>';

  function showDialog(st) {
    rm('dialog');
    const mask = document.createElement('div');
    mask.setAttribute('data-source-update', 'dialog');
    mask.style.cssText = 'position:fixed;inset:0;z-index:99999;display:flex;align-items:center;justify-content:center;' +
      'background:var(--color-scrim,rgba(0,0,0,.5));';
    mask.innerHTML =
      '<div style="width:min(480px,92vw);background:var(--color-surface,#252526);color:var(--color-fg,#ddd);' +
      'border:1px solid var(--color-input-border,#444);border-radius:10px;padding:18px 20px;' +
      'box-shadow:0 12px 40px rgba(0,0,0,.4);font-size:13px;line-height:1.6;">' +
      '<div style="display:flex;align-items:center;gap:8px;font-size:15px;font-weight:600;">' +
      SVG.up + '发现新版本 ' + esc(st.latest) +
      '<span data-act="close" title="关闭" style="margin-left:auto;cursor:pointer;opacity:.6;padding:2px 6px">' + SVG.close + '</span></div>' +
      '<div style="color:var(--color-subtle,#999);margin:8px 0 14px">' +
      '当前 ' + esc(st.current) + '，上游官方源码仓库有新版本。<br>' +
      '「立即更新」将自动执行：合并上游 → 重建前端 → 构建 exe（按需）→ 成对部署（失败自动回滚）。' +
      '部署阶段软件会重启一次，页面稍后自动恢复。</div>' +
      '<div style="display:flex;gap:8px;justify-content:flex-end;flex-wrap:wrap">' +
      btn('ignore', '忽略此版本', false) + btn('later', '稍后', false) + btn('update', '立即更新', true) +
      '</div></div>';
    mask.addEventListener('click', (ev) => {
      const t = ev.target && ev.target.closest ? ev.target.closest('[data-act]') : null;
      if (t) act(t.getAttribute('data-act'));
    });
    document.body.appendChild(mask);
    S.dialog = mask;
    if (window.console) console.log('[source-update] 发现新版本', st.latest, '(当前', st.current + ')');
  }

  function showBubble(st) {
    rm('bubble');
    const b = document.createElement('div');
    b.setAttribute('data-source-update', 'bubble');
    b.style.cssText = 'position:fixed;right:16px;bottom:16px;z-index:99999;max-width:340px;' +
      'background:var(--color-surface-2,#2d2d30);color:var(--color-fg,#ddd);' +
      'border:1px solid var(--color-input-border,#444);border-radius:8px;padding:10px 12px 10px 30px;' +
      'font-size:12px;line-height:1.5;box-shadow:0 6px 24px rgba(0,0,0,.35);';
    b.innerHTML =
      '<span style="position:absolute;left:10px;top:11px;opacity:.8">' + SVG.spin + '</span>' +
      '正在后台更新至 ' + esc((st && st.latest) || '') + '…' +
      '<div style="color:var(--color-subtle,#999);margin-top:4px">构建可能需要几分钟；部署阶段页面会短暂失联，稍后自动恢复。</div>' +
      '<span data-act="close" title="关闭" style="position:absolute;top:6px;right:8px;cursor:pointer;opacity:.6">' + SVG.close + '</span>';
    b.style.position = 'fixed';
    b.addEventListener('click', (ev) => {
      const t = ev.target && ev.target.closest ? ev.target.closest('[data-act="close"]') : null;
      if (t) rm('bubble');
    });
    document.body.appendChild(b);
    S.bubble = b;
  }

  async function act(a) {
    try {
      if (a === 'update') {
        const r = await ui.invoke(PLUGIN, 'startUpdate');
        rm('dialog');
        if (r && r.ok) showBubble(S.st || {});
        else if (window.console) console.error('[source-update] 启动更新失败:', r && r.msg);
      } else if (a === 'ignore') {
        await ui.invoke(PLUGIN, 'ignoreVersion');
        rm('dialog');
      } else { // later / close
        await ui.invoke(PLUGIN, 'later');
        rm('dialog');
      }
    } catch (e) {
      if (window.console) console.error('[source-update] 操作失败:', e);
    }
  }

  function apply(st) {
    S.st = st || {};
    if (st && st.hasUpdate && st.phase === 'available') {
      const key = String(st.latest || '') + '|' + String(st.current || '');
      if (S.dialogKey !== key) { S.dialogKey = key; showDialog(st); }
    } else if (st && st.phase === 'updating') {
      rm('dialog'); showBubble(st);
    } else {
      rm('dialog');
      if (!st || st.phase !== 'updating') rm('bubble');
    }
  }

  try { ui.on('ui:source-update/state', (st) => apply(st)); }
  catch (e) { if (window.console) console.error('[source-update] 事件订阅失败:', e); }
  try {
    ui.invoke(PLUGIN, 'getState').then(apply).catch((e) => {
      if (window.console) console.error('[source-update] getState 失败:', e);
    });
  } catch (e) { if (window.console) console.error('[source-update] invoke 不可用:', e); }
}
