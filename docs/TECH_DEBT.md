# 技术债与已知限制（TECH_DEBT）

> 本文件记录**已明确定位但未修复**、或**判定为已知限制**的问题，避免反复烧预算。

## 已知限制 1：`.menu-btn` 内在宽度 40 vs 浏览器 55（chevron 溢出 7px）

**状态：永久停止投入**（连续 4 轮未收敛，投入产出失衡；按监督者第 9 轮指令退役）。

**现象**：帮助菜单按钮 `.menu-btn`（`display:inline-flex`）在 wb-ui 实测宽 **40**，浏览器 **55**；
其内部 chevron svg 右界 **152** 越过按钮右界 **145**（溢出 7px，视觉上图标贴边/出框）。

**对照数值**：

| 侧 | `button[0]` rect（同探针 / 同 1280×800 视口） | chevron svg |
|---|---|---|
| 引擎 | `105,8,`**`40`**`,24` | x = 141..**152**（> 按钮右界 145） |
| 浏览器 | `105,8,`**`55`**`,24` | x = 141..152（≤ 按钮右界 160 ✔） |

**已排除的路径（均有实测数据，非猜测）**：

1. **`intrinsicContentWidth(.menu-btn)` 返回 55**（正确的行 flex 求和：label 22 + caret/chevron 11 + gap 4 + padding 18）
   → 取值源正确，**不是它**。（本轮据此把 `inline-flex`/`inline-block` 包装层计入行内求和，改动保留：语义正确、全量回归绿。）
2. **`minContentWidth` 已改为「flex 行容器 = 子项 min-content 求和 + gap」**（`+49/-15`），
   但目标数值**仍是 40** → 该路径亦非决定性环节。
   `minContentWidth` 的 3 个调用点已定位：
   - `:925` 自身递归（已修）；
   - `:1345` **CSS-FLEXBOX §9.7 自动最小尺寸 clamp**（条件 `cs.OverflowX == style.OverflowVisible` 才取 `minContentWidth`）；
   - `:1527` 交叉轴 fit-content。

**下一步唯一入口（未验证，留给后续单独立项）**：在 `:1345` 打印
`it.minWidth / minContentWidth(it.box) / minBounds[i] / it.baseSize / it.targetSize / containerMainSize / freeSpace`，
判定是「`cs.OverflowX` 判定未通过导致 `minContentWidth` 从未被调用」还是「`it.baseSize` 早已是 40」。
**在此之前不要再改 `intrinsicContentWidth` / `minContentWidth` 或 flex 收缩阈值**（避免把 header title、
状态栏等其它 flex 场景改回归）。

## 已关闭（**非缺陷**）：输入区底行「5px 偏移」＝ 探针选择器口径差异

**终结判定（第 14 轮，双侧同源数据 · 同一页面 1280×800）**

原始输出（引擎，`state:*` 逐字段上报）：

```
state:obtn0Y   {"v":711}
state:barMargin {"v":"mt=0,mb=0,h=32.0px,rectY=711"}
state:barRect  {"v":"711,32"}
state:row1Rect {"v":"631,24"}
state:ciRect   {"v":"663,40"}
```

原始输出（浏览器，web_debug 同脚本）：

```
obtn0Y=706 | barMargin=mt=0px,mb=0px,h=32px,rectY=711 | wrapGap=disp=flex,gap=8px,rowGap=8px,padT=8px,padB=8px,rect=622,130
ciRect=663,40 | barRect=711,32 | row1Rect=631,24
```

逐字段对照：

| 字段 | 引擎 | 浏览器 | 差 |
|---|---|---|---|
| `.input-wrapper > *`（row1）rect | 631,24 | 631,24 | **0** |
| `.chat-input` rect | 663,40 | 663,40 | **0** |
| `.input-bottom-bar` rect | 711,32 | 711,32 | **0** |
| `.input-bottom-bar` margin | mt=0,mb=0,h=32 | mt=0px,mb=0px,h=32px | **0** |
| 首个 `.obtn` 的 y | 711（位于 bottom-bar 内） | 706（**另一个 `.obtn` 元素**） | 5（口径） |

→ 三个关键容器几何**两侧逐位一致（0 差）**，引擎布局与浏览器相同；此前记录的「引擎 711 / 浏览器 706」
来自 `document.querySelector(".obtn")` 在浏览器侧**命中了另一个 `.obtn`**（引擎侧首个 `.obtn` 在 bottom-bar 内、y=711）。
**判定：非缺陷（探针口径差异）→ 支线关闭。**

**残留（读值层，无可见因果，按纪律不再追）**：引擎 `getComputedStyle(.input-wrapper)` 的
`display/gap/row-gap/padding-*` 读到 `undefined`/`0`，浏览器为 `flex`/`8px`/`8px`/`8px`；
但两容器 `rect` 完全一致 ⇒ 对布局与可见渲染无影响（与已关闭的 `.chat-input` 读值支线同源）。

## 已关闭支线：`.chat-input` 引擎侧 padding 读 0（读值层差异，**无可见因果**）

**终结判定（第 13 轮，依据 WB_COMPUTED_DEBUG 原始输出）**：

```
$ WB_COMPUTED_DEBUG=1 p1-win.exe -port 9098 ...      # 真实页面
[computed-dump] <div class="chat-input"> CACHE MISS: 走完整级联，将打印 matchedDecls
[computed-dump] <div class="chat-input"> matched decls=3
[computed-dump]   padding    = [{7 0 0 0 0 0 0 }] imp=false order=2
[computed-dump]   box-sizing = [{0 0 border-box 0 0 0 0 0 }] imp=false order=3
[computed-dump] <div class="chat-input"> CACHE HIT: cached padding-top="0" min-height=""
```

1. 真实页面该元素**只匹配到 3 条声明**，`padding` 的解析值为 **0**；而源码规则
   `RightPanel.vue:3031-3035` 含 **7 条**声明（border / background / height / min-height / font-size / padding / box-sizing）。
2. 把**同一规则集原样搬进纯 HTML** 后，引擎读到 `pt=8px pr=0 pb=8px pl=0 minH=40px fs=14px h=40px bs=border-box`
   （`webkit/computed_important_test.go` PASS）⇒ **引擎的选择器匹配/级联/`!important` 处理均正确**；
   差异来自**真实页面 dist CSS 的内容**（该规则在 dist 中未以同形选择器出现），**不是引擎缺陷**。
3. **可见影响：无**。该元素声明 `box-sizing:border-box` + `height:40px`，实测 `rectH` 两侧均为 **40**（`663..703`），
   padding 不改变总高 ⇒ 对可见布局零收益（底部行的 y 也不由它决定）。

**处置：支线关闭**，后续轮次不再追。（依赖 computed 的 JS 逻辑仍受「白名单补齐四向 padding/margin + border 简写展开」修复的正面影响。）

**未取到的证据（如实标注）**：A2 要求的「dist 中含 chat-input 的 CSS 规则原文」未取到 —— 探针新增的
`cssRules` 字段在该次运行未上报（evidence 未产出），因此**未指名**具体是哪条规则贡献了 `padding:0`；
上面第 2/3 条的判定**只依赖 A1 原始输出 + 最小用例对照**，不含推断。

**状态：可修缺陷，落点已收敛，待落地**（此前误作为「已知限制」归档，此处纠正）。

**对照数值**：

| 元素 | 引擎 y | 浏览器 y | 差 |
|---|---|---|---|
| `.obtn[0..2]`（输入区工具按钮） | 711 | 706 | +5 |
| 模型 `select[last]` | 713 | 708 | +5 |
| `.ibb-enter-hint` | 720 | 715 | +5 |
| `.input-wrapper`（整体） | 622,130 | 622,130 | ✔ 0 |
| `.chat-input` height | 40px | 40px | ✔ 0 |
| `.chat-input` padding-top/-bottom（computed） | **0 / 0** | **8px / 8px** | ✗ |

**已确定（含本轮新增证据）**：

1. **`.chat-input` 的 padding 差异与 5px 无因果关系**：其 `rectH` 两侧均为 **40**（`663..703`），
   底行 y 只由该总高决定 —— 故「引擎 padding 读 0 / 浏览器 8px」不产生这 5px。
2. **真实落点 = `.chat-input` 与 `.input-bottom-bar` 之间的间距**：由两侧同源数据反推 ——
   `.chat-input` 底边两边同为 **703**，而底行首个 `.obtn` 的 y 为引擎 **711** / 浏览器 **706**
   ⇒ 引擎间距 **8px**、浏览器 **3px**（差 5px）。`select[last]`(713/708)、`.ibb-enter-hint`(720/715) 同步同差。
3. 下一步（单点，一次即可定死）：dump `.input-bottom-bar` 的 `margin-top` 与 `.input-wrapper` 的 `gap`/`row-gap`
   （引擎 computed 读取 + 布局值），确认是多出的 margin 还是 gap 项。

## 已知限制 3：`BoxContentSize` 深嵌套近似 O(n²)

200 层 `overflow:auto` 深嵌套下耗时达分钟级（用例已 `t.Skip` 并注明）。真实页面层化嵌套 ≤3 层，微秒级、无可感影响。
优化方向：同帧缓存每个 box 的内容尺寸，或做「内容是否可能超出」的廉价短路。

## 已闭环条目（对照留档）

## 已核定 **非引擎缺陷**：两条「前端替代性绕过」已消除（2026-09-27）

gou-ide 前端 `plugins-src/ui-app/src/components/AboutModal.vue` 曾有两处为绕开
「引擎缺陷」而降级的写法。经引擎级实测（wb-ui `dev/probes/mathfunc_probe` /
`dev/probes/nowrap_probe`，提交 `aea8f5f`），两条依据**均不成立**，前端已还原为
标准 CSS（gou-ide 提交 `cd2252f1`，产物 ui-modals.js/css 成对重建并同步）。

### ① 裸 CSS 数学函数（min/max/clamp）作为长度属性值 —— 引擎**原生支持**（不存在缺陷）

**原判断**：「引擎未实现 CSS min()，`max-height: min(680px, 88vh)` 该声明被整体丢弃
（实测弹窗一度撑到 998px）」→ 前端降级为单一值 `max-height: 88vh`。

**实测（`mathfunc_probe`，视口 1440×900；读真实 `offsetHeight` + `computed maxHeight`）**：

| 声明 | 引擎 offsetHeight | 浏览器期望 |
|---|---|---|
| `max-height: 88vh` | 792 | 792 ✔ |
| `max-height: min(680px, 88vh)` | **680** | 680 ✔ |
| `max-height: max(100px, 20vh)` | 180 | 180 ✔ |
| `max-height: clamp(100px, 50vh, 400px)` | 400 | 400 ✔ |
| `max-height: calc(100vh - 100px)` | 800 | 800 ✔ |

引擎链路（已读码确认）：`engine/style/resolver.go:2041`（`max-height` 分支）→
`parseLength`（:2677）→ `mathFuncInfoS`（:2736，把 calc/min/max/clamp/round 同等识别）
→ `css.CalcHasRelativeUnit` 为真时返回 `Length{Unit:"calc", CalcExpr:…}`，延迟到布局期
带真实 context 求值（`engine/layout/layoututil.go:88`）。
**故「引擎未实现 min()」不成立**（`IsCalcValue` 只认 calc 这一点只影响
`cs.CalcValues` 的冗余缓存，不阻断 `parseLength` 路径）。原降级系拿旧引擎二进制
得出的**假阴性**。

### ② content 恰等于文字宽时误折行竖排 —— **不存在**

**原判断**：「wb-ui 引擎中 content 恰等于文字宽时会误折行竖排 —— AboutModal 关闭按钮
变两行的根因」（`6ab5ff36`，2026-08-10）→ 前端挂 `white-space: nowrap`。

**实测（`nowrap_probe`）**：复刻本弹窗 `fixed overlay → flex 列 → modal-footer flex 行`
结构（**故意不加 nowrap**），固定场景 + 对「关闭」文本宽（26.0000px）做 **15 档扫描**
（`floor/ceil` 及 ±0.01 / ±0.5 / ±1 / ±2px），以 `Range.getClientRects()` 的 top 分带
去重作为行数判据 —— **全部单行（textTopBands==1）**，`scrollW` 恒为 26 不触发折行。
**该缺陷不存在**，nowrap 已移除。

### 教训（防复发）

「引擎不支持 X / 引擎在 Y 情况下会出错」这类结论**必须用当前源码构建的引擎实测**
（可复跑探针落 `dev/probes/`），不能沿用旧二进制或旧轮次的观察：否则会把可用的标准
写法降级成 hack，形成长期技术债，且后人无法判断该 hack 是否仍必要。

## 待核定：其余前端「引擎兼容层」（已识别，**尚未逐项实测**，勿据此改动）

| 位置 | 注释要点 | 状态 |
|---|---|---|
| `PresetManager.vue:183` | 兼容「引擎 select change 事件缺失」→ 除 `@change` 外加 watch 双保险 | 待核定 |
| `TerminalPanel.vue:204` | desktop（wb-ui 引擎）无 `HTMLCanvasElement` → 换 xterm 渲染器 | 待核定（疑**真实**能力缺失） |
| `TerminalPanel.vue:269` | desktop 无 `Blob` 构造器 → PTY 输出走字符串 | 待核定（疑**真实**能力缺失） |
| `TerminalPanel.vue:655/665` | xterm span 的 `height:100%` 解析异常；光标闪烁改 `background-color` | 待核定 |
| `CodeEditor.vue:266/275` | CM6 HeightOracle 行高探测依赖 `createRange`，引擎内加兼容层 | 待核定 |

> 核定方法同 ①②：写可复跑探针（`dev/probes/`）跑当前引擎，先证伪/证实再决定
> 是「修引擎并去掉前端绕过」还是「确认能力缺失并保留」。

| 问题类别 | 项 | 引擎值 | 浏览器值 | 状态 |
|---|---|---|---|---|
| 内容加载 | `img.complete` / `naturalWidth` / `naturalHeight` | 修复前 `undefined` → 修复后 `true` / `>0` | `true` / `>0` | **已修**（`webkit/img_loading_state_test.go`，断言已收紧） |
| 内容加载 | `llm-trace` 插件 `apply` 因「无工作区」中断装载 | 修复前每次启动报错 → 修复后正常装载（延迟初始化） | — | **已修** |
| 事件响应 | `select` 点击展开 + 选中 | `popup open rows=7` + `change val=计划讨论` | 同 | **判定为非缺陷**（此前系探针盲区） |
| 事件响应 | 滚轮未上溯可滚祖先 | `.settings-content` content 235 < 可视 562；`.modal-content` 为 `overflow:hidden` | 同 | **判定为非缺陷**（引擎判定正确，探针口径误报） |
| 组件样式 | grid item `height:100%` 用错包含块 | 槽位高 800 → 修复后 28/40/732（与轨道一致） | 同 | **已修** |
| 组件样式 | flex 交叉轴 margin 丢失 | `.conv-footer-pill` x 49 → 57 | 57 | **已修** |
| 组件样式 | `select` 在 flex 上下文内在宽度塌陷 | 38px → 240px | 240px | **已修** |
| 事件响应 | 手写 contenteditable 输入丢弃字符 | `inputText` 空 → `"hello"`（len=5） | — | **已修** |
