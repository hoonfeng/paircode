# wb-ui 桌面端 JS 链/渲染链问题排查说明

> 背景：桌面版（wb-ui 引擎 + goja JS）与浏览器（V8/Blink）行为存在系统性差异。
> 前端代码相同，差异全部来自 wb-ui 引擎的 DOM/CSS/事件/字体实现不完整。
> 本文档记录排查方法论与已知修复，后续遇到"桌面版异常、浏览器正常"的问题
> 按此说明逐层排查，可大幅缩短定位时间。

---

## 一、问题表象 → 根因速查表（2026-08-07 更新）

| 表象 | 根因（wb-ui 引擎） | 修复 |
|---|---|---|
| 编辑区打开文件空白/半渲染 | CodeMirror 6 初始化抛错：`document.getSelection()` → `document.hasFocus()` → `dom.attributes.length` 三个 DOM API 缺失，`new EditorView()` 中途异常，Vue 挂载中断 | bindings/dom.go：getSelection 幂等分支补挂；hasFocus 扫描 focused 元素；Element.attributes 返回 NamedNodeMap |
| 打开文件编辑器布局错乱（cm-editor 0x0、内容画到编辑器外） | **布局引擎 Lookup 函数缺 TrimSpace**：CSS 值 `display:"flex "`（带尾随空格）→ `LookupDisplayType("flex ")` 不匹配 → default **inline** → cm-editor 行内布局塌缩。5 个 Lookup（display/position/white-space/overflow/text-overflow）全部受影响；bindings 侧 trim 了所以 JS getComputedStyle 正常（flex），布局侧 inline——两套不一致的根因 | style/computedstyle.go：5 个 Lookup 函数统一 `strings.TrimSpace(s)`。验证：cm-editor display=flex、position=relative、540.2 高正常布局 |
| 访问 `el.parentElement`/`el.children`/MutationObserver target 触发 Go panic | **typed nil 陷阱**：`ParentElement()` 返回 `(*dom.Element)(nil)` 赋给 `dom.Node` 接口后 `n == nil` 为 false → `wrapElement(nil)` → `makeDataset(nil)` panic → 渲染中断 | bindings/dom.go：nodeAccFn/arrElem/arrNode/nodeToJS 加 isNilNode/isNilEl typed-nil 检查；mutationRecordToJS 的 Target 强断言改 ok 模式 |
| 编辑输入字符破坏 CM6 结构（行 div 全部消失） | `setFocusedElementValue` 对 contenteditable 执行 `SetTextContent(全文)` → 抹掉 .cm-line/高亮 span；CM6 的 input 处理发现文本未变（全文替换文本相同）不重建结构 → 布局永久破坏 | bindings.InsertTextAtSelection（Selection range 处插文本节点）+ host.go：contenteditable 走光标插入 + input 事件；setFocusedElementValue 对 contenteditable 拒绝全文替换 |
| markdown 文本字体/颜色异常、乱码 | `getComputedStyle` 返回 `var(--font-code)` 字面量（级联 map 不解析 var()，自定义属性不继承） | bindings/dom.go：computedStyleFor 末尾 resolveVarInComputed + substituteVars（从 :root 收集 -- 变量替换） |
| 历史对话只加载最后 1 个 run | 滚轮/滚动条只更新引擎内偏移，不派发 `scroll` DOM 事件 → Vue @scroll 懒加载永不触发 | app/host.go：Host.dispatchScrollEvent 全路径派发（滚轮插值/滚动条/键盘） |
| 工作区切换慢（vs 浏览器） | ① DOM 事件派发走 goja `Call` 不 flush microtask，固定 120ms sleep 驱动 ② RebuildRenderTree 全量重建 + style 全量重扫 | ① host.go 智能等待（PendingTasks 排空即退）② frame.go style 指纹缓存跳过重扫（324→124ms） |
| desktop 只显示 1 个工作区 | ConfigDir() 用 exe 目录 → bin/ 下的 exe 读 bin/config 旧配置 | gou-ide core.ConfigDir：exe 在 bin/ 回退上级 config |

## 二、排查方法论（五步法）

### 1. 确认"浏览器正常"（排除前端问题）
- `web_debug` 打开 http://localhost:9090（或 9096）复现，截图 + console 错误
- 浏览器正常 → 问题 100% 在 wb-ui 引擎

### 2. 用 probe 复现（真实 dist + desktopbridge）
probe 加载真实 `cmd/companion/web-ui/dist` + desktopbridge（真实 API handler），
UI 交互用 `dispatchEvent(new MouseEvent('click', ...))` 走 Vue @click 真实路径。

现有 probe（`dev/desktop_probe/`，单文件 `go run`）：
```bash
set CGO_ENABLED=1
go run ./dev/desktop_probe/editor_md_probe.go     # 编辑器(CM6) + markdown var() 诊断
go run ./dev/desktop_probe/editor_render_probe.go # 离屏渲染 → PNG（_editor_render.png 等）
go run ./dev/desktop_probe/history_scroll_probe.go # 历史对话懒加载
go run ./dev/desktop_probe/ws_switch_probe.go     # 工作区切换耗时构成
```
★ 每个 probe 都是独立 main（目录不可整体 build），必须单文件 go run。

### 3. 注入全局错误捕获（定位 JS 异常）
```js
window.__errs = [];
window.addEventListener('error', function(e){ window.__errs.push('error: ' + (e && (e.message || e.type))); }, true);
window.addEventListener('unhandledrejection', function(e){ window.__errs.push('rejection: ' + ...); }, true);
var _ce = console.error;
console.error = function(){
  window.__errs.push('console.error: ' + Array.prototype.slice.call(arguments).map(function(a){
    var m = typeof a === 'string' ? a : ((a && a.message) || String(a));
    if (a && a.stack) m += ' | STACK: ' + String(a.stack).split('\n').slice(0, 8).join(' <- ');
    return m;
  }).join(' | ').slice(0, 700));
  return _ce.apply(console, arguments);
};
```
★ **必须抓 stack**（goja 的 <eval>:行号对应 bundle 行号）：
- `findstr /n "函数名" dist/assets/index-*.js` 或 python 按名字找 bundle 代码
- 堆栈指向的 setAttrs/sync 等函数名即定位线索

### 4. 按缺失 API 补实现（bindings/dom.go）
错误 `Object has no member 'xxx'` → 该对象缺方法。排查链：
- **document 级**：`wrapDocument` 里应有全部 document API。注意幂等分支：
  `RegisterDOMBindings` 里 `if domBindingsMarker 存在 → wrapDocument 新建 docObj 就 return`。
  ★ 任何 docObj 级 API（getSelection 等）必须放 wrapDocument 内或幂等分支里补挂，
  否则每次 EvalJS/RunJS 刷新后丢失 → 库初始化抛错。
- **Element 级**：`wrapElement` 里补（attributes/classList/focus 等）。
- **window 级**：全局对象 g（首次注册时挂）。

常见第三方库 DOM 依赖清单（已踩坑）：
- CodeMirror 6：`document.getSelection()`、`document.hasFocus()`、`dom.attributes`（NamedNodeMap：length + 索引）、`contenteditable`、ResizeObserver（已有，contentRect 为 0 够用）、`getComputedStyle().fontFamily`
- marked/markdown-it（v-html）：无特殊依赖，但要 getComputedStyle 能解析 var()
- Vue 3：`Element.ownerDocument`、`template.content`、`nodeWrapperCache` 引用相等（===）、`dispatchEvent`、scoped data-v

### 5. var() / CSS 变量（markdown 乱码/字体异常高频根因）
两层都要检查：
- **getComputedStyle 层**（bindings/computedStyleFor）：级联 map 直接写原始声明值，
  不解析 var()。修复：resolveVarInComputed（收集 :root 的 -- 变量 + substituteVars 替换）。
  ★ 只收集 :root（本项目变量全在 :root）；祖先链全遍历会因每层全量级联计算死循环/超时，
  必须加 computedStyleDepth 深度保护。
- **paint 层**（style/resolver.go）：已支持（resolveCustomProperties + resolveVarInProperties），
  一般无需改。
- 字体链路：`FontFamily` 字符串 → `splitFontFamily` 拆列表 → resolveFamily 映射
  （monospace→consolas、CJK→雅黑）→ OS lookup（中文必须走 osLookup 才有 Skia fallback）。

## 三、本次修复详情（2026-08-07）

### 修复 1：CodeMirror 6 编辑器无法初始化（编辑区异常）
- **现象**：打开文件编辑区空白/半渲染，console 报 `Object has no member 'getSelection'`
- **根因**：`RegisterDOMBindings` 幂等分支（每次 EvalJS/RunJS 前刷新 document）
  只 `wrapDocument` 后 return，而 `docObj.Set("getSelection")` 在幂等分支之外
  （首次注册才执行）→ 刷新后的 document 对象缺 getSelection。
  CM6 `EditorView` 构造调 `document.getSelection()` 抛错 → DOM 建到一半（有
  scroller/gutters 无 content）→ Vue 挂载中断。
- **修复链**（每补一个暴露下一个）：
  1. `document.getSelection()` → selObj 单例提前到幂等分支前创建 + 幂等分支补挂
  2. `document.hasFocus()` → wrapDocument 加 hasFocus（DocumentQuerySelectorAll("*")
     找 IsFocused 元素）
  3. `dom.attributes.length`（CM6 setAttrs）→ Element 加 attributes accessor
     （NamedNodeMap：arrayValue + {name,value}）
- **验证**：probe cmTree：errs=[]、hasEditor=true、lineCount=50、contentLen=1693

### 修复 2：markdown var() 未解析（字体/颜色异常、乱码）
- **现象**：markdown code/blockquote/a 的 computedStyle 返回 `var(--font-code)` 字面量
- **根因**：computedStyleFor 级联后把 `md.decl.ValueString()` 原样写 out，无 var() 替换；
  自定义属性（--x）不继承（浏览器语义：随级联继承）
- **修复**：computedStyleFor 末尾 resolveVarInComputed（:root 收集 -- 变量 +
  substituteVars 替换，嵌套 depth≤4）
- **验证**：code.ff 从 `var( --font-code )` → `"JetBrains Mono", "Cascadia Code", ...`；
  blockquote.color → `#8b949e`；a.color → `#79c0ff`

## 四、后续优化指引（按优先级）

1. **contenteditable 富文本编辑**（editing/ 包注释已预留）：
   CM6 依赖 contenteditable 输入。当前能显示/光标，但**输入/编辑可能不完整**
   （wb-ui 只支持 input/textarea 的键盘事件路径）。验证并补 contenteditable
   的 beforeinput/input 事件 + 选区编辑。
2. **ResizeObserver 真实现**：当前 observe 只微任务通知一次 contentRect(0,0,0,0)。
   CM6 布局依赖真实尺寸——若编辑器在容器 resize 后不重排，需要补尺寸回调。
3. **font-family 引号清理**：resolver 的 `strings.Trim(resolvedStr, \`"'\`)`
   只去首尾引号，中间引号（`"JetBrains Mono" , "Cascadia Code"`）仍保留——
   splitFontFamily 能处理但建议 resolver 统一清理。
4. **getComputedStyle 全量匹配性能**：每次级联全量遍历 CSS 规则匹配（Vue bundle
   几千条规则）。styleSheetCache 已缓存解析结果，但匹配仍每次全量——后续可加
   选择器索引（style/resolver 已有 sheetIndex 参考）。
5. **历史对话 markdown 渲染**：RightPanel 消息用 MarkdownRenderer（marked + mermaid）。
   mermaid 依赖 SVG 渲染，若图表显示异常单独排查 SVG 引擎。
6. **实机验证流程**：desktop 窗口 + 坐标点击受 Windows 前台锁/z-order 影响不稳定，
   优先用 probe 离屏渲染（wv.Render() → PNG）验证视觉；需实机时用
   PostMessage(WM_LBUTTONDOWN) 直发窗口消息（需写死 hwnd，见 _post2.ps1 思路）。

## 五、关键代码位置速查

| 文件 | 关注点 |
|---|---|
| wb-ui/bindings/dom.go | RegisterDOMBindings（幂等分支★）、wrapDocument（hasFocus★）、wrapElement（attributes★）、computedStyleFor + resolveVarInComputed + substituteVars |
| wb-ui/app/host.go | dispatchScrollEvent、handleClick 智能等待、imeFocusedEl |
| wb-ui/page/frame.go | styleFingerprint 缓存、RebuildRenderTree |
| wb-ui/style/resolver.go | resolveCustomProperties / resolveVarInProperties（paint 层 var） |
| wb-ui/platform/graphics/fontmgr.go | splitFontFamily、resolveFamily、osLookup（CJK fallback） |
| gou-ide/internal/core/settings.go | ConfigDir（bin/ 回退） |
| gou-ide/internal/desktopbridge/desktopbridge.go | fetch 拦截、WS_SWITCH_TIMING 耗时日志 |
