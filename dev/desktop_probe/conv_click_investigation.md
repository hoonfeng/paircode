# conv-item 点击不切换 active —— 证据文档（线索累积）

> 项目: gou-ide (web-ui Vue 3 + wb-ui 引擎) + wb-ui
> 状态: ✅ **已修复**（根因=Text/Comment JS wrapper 无缓存导致 Vue removeFragment 引用相等失败）
> 修复提交: wb-ui bindings/dom.go — Element-only wrapper cache 扩展为全部节点类型
> 最后更新: 2026-08-05（修复后）

---

## 0. 一句话结论（修复版）

desktop 端（wb-ui 引擎）点击左侧对话列表 `.conv-item` 无法切换 active 的根因：

**wb-ui bindings 的 `wrapText`/`wrapComment` 每次调用都创建全新的 JS wrapper 对象（Element 有 `elementWrapperCache`，Text/Comment 没有）。**
Vue 3 渲染器的 `removeFragment()` 用 `cur !== anchor`（引用相等）终止遍历循环——
在 wb-ui 中每次 `nextSibling` 都返回新 wrapper，`cur` 永远不可能 `=== anchor`，
循环永不终止，一路遍历到 `undefined` 后崩溃 `Cannot read property 'nextSibling' of undefined`。
崩溃中断 patch → vnode 树残破（el 为 undefined）→ 点击更新再崩溃 → active 永不切换。

**修复**：把 Element-only 的 wrapper 缓存扩展为 `nodeWrapperCache`（`map[dom.Node]*jsc.JSObject`），
`wrapText`/`wrapComment` 与 `wrapElement` 一样返回缓存 wrapper → 引用相等恢复 → removeFragment 正常终止。

## 1. 修复验证结果（conv_click_probe_clean.go，零 hook）

```
修复前: dynEls 前 7 true 后 26 false（badCount=29）→ 点击无反应
修复后: dynEls 33 个全 true → 点击 #1 →
  {"i":1,"cls":"conv-item active",...}        ← active 类正确切换
  currentConvId: conv_1785848158762927100      ← 会话正确切换
  msgItems:2, msgCount:2                        ← 消息区同步加载
  bodyChildren:153（点击前后不变）              ← 无幽灵节点累积
```

- `badCount=76` 为**正常**：subTree 扫描把未激活的 v-if 分支（el=undefined 合法态）也计入，不影响功能。
- 剩余 T C T 锚点组为 Teleport to="body" 组件的**正常挂载**（浏览器同样存在），不再崩溃、不再累积。

---

## 2. 问题现象（修复前）

- desktop 端点击左侧对话列表第 2 个 `.conv-item`，期望 active 类切换到该项 + 消息区刷新，实际无反应。
- 通过 `dev/desktop_probe/conv_click_probe_clean.go`（零 hook 复现）确认：
  - 点击前 vnode 树已残破：`badCount=29`，`dynEls` = 前 7 个有 el、后 26 个 el 为 `undefined`。
  - `childrenEls: [false, false]`（subTree.children 的 el 也是 false）。
  - console.error 捕获：`Cannot read property 'nextSibling' of undefined`、`Cannot read property 'style' of undefined`、`Cannot convert undefined or null to object`。

## 3. 幽灵节点（body 顶层）

- `document.body.childNodes` 有 **201 个**（正常应为 3 个：`\n  `、`div#app`、`\n`）。
- 结构：`[T(3), DIV#app, T("\n\n\n"), (T,C,T)×66]`，共 198 个幽灵空节点。
- 模式 `T C T` = 空文本 + 空注释 + 空文本，共 66 组。
- **与 script 执行强相关**：移除 `<script type="module">` 后 body 恢复 3 节点（B-no-script 对照实验）。
- 这些节点 parent 都是 body，Go 侧指针完全一致（见 §5）。

## 4. 幽灵节点插入者：Vue Teleport（决定性证据）

通过 `BeforePageScripts` 钩子（在 Vue 执行前安装，链式包装 desktopbridge 的 hook）对 `document.body` 的实例方法 `insertBefore/appendChild/removeChild` 打点，捕获：

```
I 1:BODY <- 3:#text
I 1:BODY <- 3:#text
I 1:BODY <- 8:#comment before 3:#text
I 1:BODY <- 3:#text
I 1:BODY <- 3:#text
I 1:BODY <- 8:#comment before 3:#text
...（共 336 次 I + 186 次 R）
```

即每组的插入顺序：先插 2 个空文本（append），再把 1 个空注释插到第 2 个文本前 → 最终排列 `[T, C, T]`。

**调用栈**（Vue dist 内函数名，eval 行号）：

插入：
```
rec → insert (11857) → prepareAnchor (8141) → mountToTarget (7847) → process (7890)
rec → insert (11857) → processCommentNode (9909) → patch (9814) → mountChildren (10071)
```

移除（崩溃路径）：
```
rec → remove (11862) → remove (7994) → unmount (10901) → unmountComponent (10989)
rec → remove (11862) → performRemove (10953) → remove2 (10967) → unmount (10926)
```

**`prepareAnchor` / `mountToTarget` / `process` 是 Vue 3.4 Teleport 组件的源码函数名。**

## 5. DOM 树指针完整性：Go 侧无断裂（排除早期假设）

早期 JS 侧检查（`dom_integrity_probe.go`）报告大量 `sibBroken/firstChildBroken/lastChildBroken`，
**后来证明是误报**（与根因同源）：wb-ui bindings 的 `nodeToJS` 对 Text/Comment 每次创建新 wrapper（Element 有缓存），
JS 侧用 `===` 比较必然失败。

改用 **Go 侧直接遍历 nodeBase 字段**（`dom_wire_probe.go` 的 `goSideIntegrity`）后：

```
[go-integrity after-load] {nodes:1575 parentBr:0 sibBr:0 fcBr:0 lcBr:0 bodyChildren:153}
[go-integrity after-render] {nodes:1541 parentBr:0 sibBr:0 fcBr:0 lcBr:0 bodyChildren:154}
```

→ **DOM 树指针完全一致，DOM 树没有坏。** 幽灵节点是合法的 DOM 节点（parent=body），只是位置错误。

## 6. 渲染树 vs DOM

- `domElements=937`（GetElementsByTagName("*")）、`renderObjects=1030`（含文本/注释 RO，差值正常，非幽灵）。
- `renderObjects not in document: 0`（所有 RO 都可追溯回 document）。
- 渲染树结构本身健康。

## 7. 已排除的假设

| 假设 | 结论 |
|---|---|
| JS 侧 DOM 指针断裂（sibBroken 等） | ❌ wrapper 误报，Go 侧一致 |
| appendChild/insertBefore 指针维护 bug | ❌ node.go 双向链表逻辑正确，Go 侧无断裂 |
| HTML tokenizer 解析 script 内容产生幽灵节点 | ❌ dist index.html 无内联 script 内容，tokenizer 有完整 script data state |
| hook 未生效（prototype 级） | ⚠️ 是坑：bindings 用 `obj.Set` per-instance 方法，prototype hook 无效；改用 document.body 实例 hook + Go 侧 OnNodeInserted 包装 |
| 幽灵节点是 JS 直接 appendChild 产生 | ❌ body 实例 hook 显示插入是 **insertBefore**，且来自 Teleport prepareAnchor |
| insertBefore 失败静默回退 AppendChild | ⚠️ 存在（wrapElement 容错），但与本次崩溃无直接因果 |

## 8. dist bundle 证据

`cmd/companion/web-ui/dist/assets/index-DLupMJd9.js` 第 7819 行：

```js
name: "Teleport",
__isTeleport: true,
process(n1, n2, container, anchor2, ...) {
  ...
  const mountToTarget = (vnode = n2) => {
    const target = vnode.target = resolveTarget(vnode.props, querySelector);
    const targetAnchor = prepareAnchor(target, vnode, createText2, insert2);
    ...
  };
  if (n1 == null) {
    const placeholder = n2.el = createText2("");
    const mainAnchor = n2.anchor = createText2("");
    insert2(placeholder, container, anchor2);   // ← 往 container 插 2 个空文本
    insert2(mainAnchor, container, anchor2);
    ...
    mountToTarget();                            // ← prepareAnchor 往 body 插锚点
  }
```

**Teleport 挂载流程（与 domlog 完全吻合）**：
1. `n2.el = createText("")`（占位空文本）+ `n2.anchor = createText("")`（主锚点空文本）
2. `insert2(placeholder, container)` + `insert2(mainAnchor, container)`
3. `prepareAnchor`：往 target（body）插入 `targetStart`（空文本）+ `targetAnchor`（空文本/注释）
4. `mount(vnode, target, targetAnchor)` 挂载 Teleport 内容到 body

## 9. 根因与崩溃机制（修复后确认）

1. Vue mount App → 渲染含 Teleport 的组件（GitPanel 2 个 + Modal 1 个 `<Teleport to="body">`）。
2. Teleport `process` → `mountToTarget` → `prepareAnchor` 往 **body** 插入锚点（T C T 组）→ 产生幽灵节点（浏览器同样行为，无害）。
3. 某组件卸载/更新 → `unmountComponent` → `remove` → **removeFragment**：
   `cur = vnode.el; next = cur.nextSibling; while (cur !== anchor) {...}`。
4. **wb-ui 的 wrapText/wrapComment 无 wrapper 缓存** → `cur`（每次 nextSibling 的新 wrapper）永远 `!== anchor`（Vue 持有的旧 wrapper）→ 循环永不终止 → `cur` 变 `undefined` → `cur.nextSibling` 崩溃。
5. 崩溃中断 patch → 部分 vnode 未挂载（29 个 el undefined）→ 点击 `.conv-item` → Vue `@click` 处理器更新组件 → 在残破 vnode 树上 patch → 再次崩溃 → active 不切换。

**修复**：`bindings/dom.go` 的 `elementWrapperCache`（`map[*dom.Element]`）扩展为 `nodeWrapperCache`（`map[dom.Node]`），
`wrapText`/`wrapComment` 与 `wrapElement` 同样返回缓存 wrapper。恢复 JS 引用相等语义（浏览器行为）。

## 10. 复现工具清单（dev/desktop_probe/）

| 工具 | 作用 | 关键输出 |
|---|---|---|
| `conv_click_probe_clean.go` | 零 hook 复现 + 点击验证（**主验证工具**） | dynEls 全 true、active 切换、msgItems 加载 |
| `conv_click_probe.go` | 深度诊断（hook update 全树扫描，修复后已无需） | — |
| `dom_integrity_probe.go` | JS 侧完整性检查（**wrapper 误报，勿再用**） | sibBroken 大量（假） |
| `dom_wire_probe.go` | Go 侧完整性 + body 实例 hook + 调用栈 | parentBr:0、`I BODY <- T/C`、prepareAnchor 栈 |
| `mount_stage_probe.go` | 分阶段快照 | 幽灵节点在 stage0 就存在 |
| `conv_click_investigation.md` | 本文档 | — |

## 11. 遗留观察项（不影响功能）

- body 顶层仍有约 50 组 `T C T` 锚点（Teleport 正常挂载），点击前后数量稳定（153 不变），无累积。
- `badCount=76` 含未激活 v-if 分支的 el=undefined（合法），非问题。
- wb-ui `wrapElement.insertBefore` 的「anchor detached 时静默回退 AppendChild」容错仍在——是设计取舍，可后续评估是否改为抛错对齐浏览器。

## 12. 环境备忘

- 运行 probe：`set CGO_ENABLED=1 && go run ./dev/desktop_probe/<tool>.go`
- wb-ui 修改位于 `F:\syproject\wb-ui`（Go module: wb-ui，被 gou-ide replace 引用）
- dist 重新构建：`cd cmd/companion/web-ui && npm run build`
- 浏览器参照：`set WEB_PORT=9096 && go run ./cmd/companion`（Edge 加载 http://localhost:9096）

---

## 1. 问题现象

- desktop 端点击左侧对话列表第 2 个 `.conv-item`，期望 active 类切换到该项 + 消息区刷新，实际无反应。
- 通过 `dev/desktop_probe/conv_click_probe_clean.go`（零 hook 复现）确认：
  - 点击前 vnode 树已残破：`badCount=29`，`dynEls` = 前 7 个有 el、后 26 个 el 为 `undefined`。
  - `childrenEls: [false, false]`（subTree.children 的 el 也是 false）。
  - console.error 捕获：`Cannot read property 'nextSibling' of undefined`、`Cannot read property 'style' of undefined`、`Cannot convert undefined or null to object`。

## 2. 幽灵节点（body 顶层）

- `document.body.childNodes` 有 **201 个**（正常应为 3 个：`\n  `、`div#app`、`\n`）。
- 结构：`[T(3), DIV#app, T("\n\n\n"), (T,C,T)×66]`，共 198 个幽灵空节点。
- 模式 `T C T` = 空文本 + 空注释 + 空文本，共 66 组。
- **与 script 执行强相关**：移除 `<script type="module">` 后 body 恢复 3 节点（B-no-script 对照实验）。
- 这些节点 parent 都是 body，Go 侧指针完全一致（见 §4）。

## 3. 幽灵节点插入者：Vue Teleport（决定性证据）

通过 `BeforePageScripts` 钩子（在 Vue 执行前安装，链式包装 desktopbridge 的 hook）对 `document.body` 的实例方法 `insertBefore/appendChild/removeChild` 打点，捕获：

```
I 1:BODY <- 3:#text
I 1:BODY <- 3:#text
I 1:BODY <- 8:#comment before 3:#text
I 1:BODY <- 3:#text
I 1:BODY <- 3:#text
I 1:BODY <- 8:#comment before 3:#text
...（共 336 次 I + 186 次 R）
```

即每组的插入顺序：先插 2 个空文本（append），再把 1 个空注释插到第 2 个文本前 → 最终排列 `[T, C, T]`。

**调用栈**（Vue dist 内函数名，eval 行号）：

插入：
```
rec → insert (11857) → prepareAnchor (8141) → mountToTarget (7847) → process (7890)
rec → insert (11857) → processCommentNode (9909) → patch (9814) → mountChildren (10071)
```

移除（崩溃路径）：
```
rec → remove (11862) → remove (7994) → unmount (10901) → unmountComponent (10989)
rec → remove (11862) → performRemove (10953) → remove2 (10967) → unmount (10926)
```

**`prepareAnchor` / `mountToTarget` / `process` 是 Vue 3.4 Teleport 组件的源码函数名。**

## 4. DOM 树指针完整性：Go 侧无断裂（排除早期假设）

早期 JS 侧检查（`dom_integrity_probe.go`）报告大量 `sibBroken/firstChildBroken/lastChildBroken`，
**后来证明是误报**：wb-ui bindings 的 `nodeToJS` 对 Text/Comment 每次创建新 wrapper（Element 有缓存），
JS 侧用 `===` 比较必然失败。

改用 **Go 侧直接遍历 nodeBase 字段**（`dom_wire_probe.go` 的 `goSideIntegrity`）后：

```
[go-integrity after-load] {nodes:1575 parentBr:0 sibBr:0 fcBr:0 lcBr:0 bodyChildren:153}
[go-integrity after-render] {nodes:1541 parentBr:0 sibBr:0 fcBr:0 lcBr:0 bodyChildren:154}
```

→ **DOM 树指针完全一致，DOM 树没有坏。** 幽灵节点是合法的 DOM 节点（parent=body），只是位置错误。

## 5. 渲染树 vs DOM

- `domElements=937`（GetElementsByTagName("*")）、`renderObjects=1030`（含文本/注释 RO，差值正常，非幽灵）。
- `renderObjects not in document: 0`（所有 RO 都可追溯回 document）。
- 渲染树结构本身健康。

## 6. 已排除的假设

| 假设 | 结论 |
|---|---|
| JS 侧 DOM 指针断裂（sibBroken 等） | ❌ wrapper 误报，Go 侧一致 |
| appendChild/insertBefore 指针维护 bug | ❌ node.go 双向链表逻辑正确，Go 侧无断裂 |
| HTML tokenizer 解析 script 内容产生幽灵节点 | ❌ dist index.html 无内联 script 内容，tokenizer 有完整 script data state |
| hook 未生效（prototype 级） | ⚠️ 是坑：bindings 用 `obj.Set` per-instance 方法，prototype hook 无效；改用 document.body 实例 hook + Go 侧 OnNodeInserted 包装 |
| 幽灵节点是 JS 直接 appendChild 产生 | ❌ body 实例 hook 显示插入是 **insertBefore**，且来自 Teleport prepareAnchor |

## 7. dist bundle 证据

`cmd/companion/web-ui/dist/assets/index-DLupMJd9.js` 第 7819 行：

```js
name: "Teleport",
__isTeleport: true,
process(n1, n2, container, anchor2, ...) {
  ...
  const mountToTarget = (vnode = n2) => {
    const target = vnode.target = resolveTarget(vnode.props, querySelector);
    const targetAnchor = prepareAnchor(target, vnode, createText2, insert2);
    ...
  };
  if (n1 == null) {
    const placeholder = n2.el = createText2("");
    const mainAnchor = n2.anchor = createText2("");
    insert2(placeholder, container, anchor2);   // ← 往 container 插 2 个空文本
    insert2(mainAnchor, container, anchor2);
    ...
    mountToTarget();                            // ← prepareAnchor 往 body 插锚点
  }
```

**Teleport 挂载流程（与 domlog 完全吻合）**：
1. `n2.el = createText("")`（占位空文本）+ `n2.anchor = createText("")`（主锚点空文本）
2. `insert2(placeholder, container)` + `insert2(mainAnchor, container)`
3. `prepareAnchor`：往 target（body）插入 `targetStart`（空文本）+ `targetAnchor`（空文本/注释）
4. `mount(vnode, target, targetAnchor)` 挂载 Teleport 内容到 body

## 8. 源码中涉及的 Teleport 组件（gou-ide web-ui 侧）

- `GitPanel.vue`：2 处 `<Teleport to="body">`（约 231、262 行）
- `Modal.vue`：1 处 `<Teleport to="body">`
- App 渲染链上初始即激活的 Teleport 数量决定了幽灵组数（66 组 T C T 待与组件树核对）

## 9. 崩溃机制（当前理解，待最后确认）

1. Vue mount App → 渲染含 Teleport 的组件（GitPanel/Modal 等）。
2. Teleport `process` → `mountToTarget` → `prepareAnchor` 往 **body** 插入锚点（T C T 组）。
3. 后续某次更新/卸载（v-if 切换、组件卸载）→ `unmountComponent` → `remove` → **removeFragment**：
   `cur = vnode.el; next = cur.nextSibling` → **el 为 undefined → 崩溃**。
4. 崩溃中断 patch → 部分 vnode 未挂载（29 个 el undefined）→ DOM 残留 detached 幽灵组。
5. 点击 `.conv-item` → Vue `@click` 处理器更新组件 → 在残破 vnode 树上 patch → 再次崩溃 → active 不切换。

**关键疑点**：`removeFragment` 的 `el` 为什么是 undefined？
- Teleport 卸载时其 el/anchor 是空文本，若这些空文本节点在 wb-ui 的 `insertBefore/removeChild` 下状态异常（如被提前移除、或 anchor2=null 时 insert 位置错误），el 会失效。
- 需要对比：浏览器中 Teleport 的 placeholder/mainAnchor 插入 container（#app），wb-ui 中是否插到了错误位置？domlog 只 hook 了 BODY，未 hook #app。

## 10. 复现工具清单（dev/desktop_probe/）

| 工具 | 作用 | 关键输出 |
|---|---|---|
| `conv_click_probe_clean.go` | 零 hook 复现点击不切换 | badCount=29、dynEls 前 7 true 后 26 false |
| `dom_integrity_probe.go` | JS 侧完整性检查（**有 wrapper 误报，勿再用**） | sibBroken 大量（假） |
| `dom_wire_probe.go` | Go 侧完整性 + body 实例 hook + 调用栈 | parentBr:0、`I BODY <- T/C`、prepareAnchor 栈 |
| `mount_stage_probe.go` | 分阶段快照 | 幽灵节点在 stage0 就存在 |
| `esm_check.go` / `set_probe.go` 等 | 早期探索 | — |

## 11. 下一步计划

1. **hook #app 的 insertBefore**，确认 Teleport 的 placeholder/mainAnchor 插入 container 时的行为（位置/顺序是否正确）。
2. **精确复现 Teleport 最小用例**：单独渲染 1 个 `<Teleport to="body">` 组件，观察挂载/卸载是否崩溃（对比浏览器）。
3. 定位 `removeFragment` 中 el undefined 的确切来源：
   - wb-ui `insertBefore(anchor2=null)` 是否等价浏览器 append？
   - wb-ui 对空文本节点（createText("")）的 insert 是否正常维护指针？
   - Vue `unmountComponent → remove` 时 vnode.el 引用的是否是 wb-ui 已移除的节点？
4. 修复后重跑 `conv_click_probe_clean.go` 验证 active 切换。

## 12. 环境备忘

- 运行 probe：`set CGO_ENABLED=1 && go run ./dev/desktop_probe/<tool>.go`
- wb-ui 修改位于 `F:\syproject\wb-ui`（Go module: wb-ui，被 gou-ide replace 引用）
- dist 重新构建：`cd cmd/companion/web-ui && npm run build`
- 浏览器参照：`set WEB_PORT=9096 && go run ./cmd/companion`（Edge 加载 http://localhost:9096）
