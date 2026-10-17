# wechat-bridge 插件修复指南

> 方向：**插件自身向现有机制靠拢（IDE 内核一行不改）**
> 本指南所有接口、字段、行为均经源码核实与实测，出处标在括号里。

---

## 零、总原则

**内核不新增任何为微信互联服务的东西。** 桥需要的每一样东西，插件都通过「自己声明配置 + 调用现有普通会话接口」拿到：

| 桥需要什么 | 现有机制里等价的东西 | 插件怎么拿 |
|---|---|---|
| 用哪个 AI 配置 / 模型 | 会话级模型三元组 `(provider, model, preset)` | 插件设置里声明 → 写进会话三元组 |
| 一个能投喂的会话 | **普通会话**（同一 `index.json`、同一 `.jsonl`、同一列表） | `POST /api/conversations` 建模 |
| 会话管理入口 | 普通会话的创建 / 列表 / 删除 | 插件在**标题栏注册面板**，调同源接口 |
| 投喂 | `POST /api/chat/send` | 沿用现状，只是 `convId` 换成面板管理的普通会话 |

**桥会话 = 普通会话**：用户能在 IDE 左侧会话列表看到它、点进去继续对话、改标题、删除。插件不做任何"影子会话"。

---

## 一、插件内声明配置（模拟「设置 → AI」的选择）

**位置**：`index.js` 顶部 `DEFAULT_SETTINGS`（现 `:35-41`）+ `ctx.registerSettings({fields})`（现 `:74-85`）。

**新增声明项**（与既有字段同格式，`{name,label,type,default,hint}`）：

```js
const DEFAULT_SETTINGS = {
  enabled: false,
  autoStart: false,          // ← 建议改默认：插件装回来不该自动拉起常驻桥
  defaultConvId: '',         // ← 建议空：改由面板「新建会话」决定
  aiPreset: '',              // ★ 新增：本插件使用的 AI 配置名（如 ds-vision）
  aiModel: '',               // ★ 新增：模型名（空 = 用该配置的 executeModel）
  aiProvider: '',            // ★ 新增：服务商（空 = 按配置整套展开）
  workspaceRoot: '',         // 现有：桥会话所属工作区
  paircodeUrl: '',
}

ctx.registerSettings({
  key: PLUGIN,
  title: '微信桥（wechat-bridge）',
  fields: [
    /* …现有 5 项保持不变… */
    { name: 'aiPreset',   label: 'AI 配置', type: 'text', default: '',
      hint: '桥会话使用的 AI 配置名（如 ds-vision）；留空＝沿用全局激活配置' },
    { name: 'aiModel',    label: '模型',    type: 'text', default: '',
      hint: '留空＝用该 AI 配置里的执行模型' },
    { name: 'aiProvider', label: '服务商',  type: 'text', default: '',
      hint: '留空＝按 AI 配置自动展开（含其 Key）' },
  ],
})
```

读取沿用现有写法（`index.js:88-93`）：

```js
function loadCfg() {
  let cur = {}; try { cur = ctx.getSettings(PLUGIN) || {} } catch (e) {}
  return Object.assign({}, DEFAULT_SETTINGS, cur)
}
```

**三项全空的语义必须是"不介入"**：桥回到现状行为（吃全局激活配置）。这样插件在旧环境可安全回退，也不会覆盖用户的其他会话选择。

---

## 二、标题栏功能面板（现成机制，加一块「会话」区）

插件**已经有**标题栏槽位注册与浮层面板，不用造新机制：

- 槽位注册：`ui.registerSlot({ slot: 'titlebar-right', … })`（`client.js:313`）
- 浮层面板 DOM：`.wxbridge-panel`（`client.js:277-299`）
- host → client 状态：`ctx.emit('ui:wechat-bridge/state', payload)`，client 侧 `ui.on(...)`（`client.js:304`）
- client → host 调用：`ui.invoke(PLUGIN, method, args)`（`client.js:98`）
- 浏览器侧可直接发同源请求：`fetch(...)`（`client.js:107` 已在用）

在面板里加一块「会话」区，内容三项：

1. **声明配置只读展示**：`AI 配置 / 模型 / 服务商 / 工作区`（取自 host 广播的 state），并标出"内核是否认定该配置可用"（见第六节自检）。
2. **「新建会话（按声明配置）」按钮** —— 见第三节。
3. **会话列表**：`GET /api/conversations?workspace=<工作区>`（`web_server.go:1103-1119`），每行给三个动作：`设为桥会话` / `归档` / `删除`。

---

## 三、新建会话 = 严格模拟现有实现（两步，顺序不可反）

现有前端"新建对话"是 `apiPost('/conversations', { title: '新对话' })`（`RightPanel.vue:1768`）；"选模型"是 `setConvModel` → `PUT /api/conversations/{id}`（`api.js:586-591`）。插件照抄这两步：

```js
// ws = cfg.workspaceRoot || ctx.app.workspaceRoot || ctx.app.root（为空时后端回落 core.Root()）
const BASE = ''                       // 同源，浏览器侧直接 fetch；host 侧用 paircode 探测地址
const qs   = '?workspaceRoot=' + encodeURIComponent(ws)

// ① 建普通会话（★ 后端幂等：同 id 重复建直接返回成功 — message_store.go:744-748）
await fetch(BASE + '/api/conversations', {
  method: 'POST', headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ id: convId, title: '微信 · ' + alias, workspaceRoot: ws }),
})                                    // → {ok:true,id,title}（web_server.go:1121-1152）

// ② 按插件声明的配置绑定「会话级模型」（与前端 setConvModel 完全同源）
await fetch(BASE + '/api/conversations/' + encodeURIComponent(convId) + qs, {
  method: 'PUT', headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ provider: cfg.aiProvider || '', model: cfg.aiModel || '', preset: cfg.aiPreset || '' }),
})                                    // → {ok:true}；日志 [conv-model] 会话 … 仅本会话生效
```

**为什么这样就够了**：内核装配时的判定是（`internal/agent/provider_factory.go:198-216`）——会话三元组非空 → 按会话级配置整套展开；全空 → 才回落全局激活配置。桥此前 400 正是因为它"全空"，只能吃全局。绑定后，投喂使用的就是**插件声明的这一套配置**，与用户在 UI 里手动选了模型等价。

**`convId` 建议**：`conv_wx_<账号别名>`（与现状一致），由面板创建；也可让用户自定义标题、id 固定，便于多账号隔离。

**放哪里执行**：装在 **client 半（面板逻辑）** 最省事——浏览器侧 `fetch` 天然支持 options（`client.js:107` 已在用）；host 半（goja）若要用 `ctx.web.fetch` 发带 body 的 POST/PUT，需按其实际签名确认（现有用法只有单参 `ctx.web.fetch(url)`，`index.js:387/396`），不支持就把这两步放 client 半、host 只接收结果。

---

## 四、归档会话 = 插件语义 + 现有手段（诚实说明）

**先说清事实**：现有机制**没有"会话归档"**这个概念——

- `ConversationMeta` 无 `archived` 字段（`message_store.go:174-204`）
- 会话列表 UI 只有「新建对话」和「删除对话」（`ConvSidebar.vue:9 / :30 / :43`）
- `DELETE /api/conversations/{id}` 是**物理删除**（删 `.jsonl` + index 条目，`web_server.go:1536-1544` + `message_store.go:1601-1634`）
- 现有实现里的"归档"二字只指**消息级归档区**（上下文压缩后的早期历史，Idx 为负，`RightPanel.vue:1282`），与"会话归档"无关

所以插件的"归档"只能在**插件自己的语义**里定义，两种落地（推荐 A）：

**A｜无损归档（推荐）**：插件侧维护 `archivedConvIds`（写在插件 settings 或本地 state）→ 该会话从面板"当前桥会话"里移出、桥不再向它投喂；**会话本体在 IDE 里仍是普通会话**，用户照常可见、可点开继续、可自行删除。不调任何写接口，零副作用。

**B｜彻底删除（可选，需二次确认）**：调 `DELETE /api/conversations/{id}`——与 UI 的 `×` 同源，但**历史不可恢复**，面板必须弹确认（文案注明"将删除该会话全部历史"）。

**新增会话 id 冲突**：面板"新建"若沿用同名 id，`POST` 幂等只会复用旧会话（不会清历史），要在 UI 上明示"已存在同名会话，将直接复用"。

---

## 五、桥投喂改为面板管理的普通会话

- `convId` 来源：面板当前选中的会话（host 侧 state 广播，或写回 `settings.activeConvId`），**不再硬依赖 `defaultConvId` / 全局激活配置**。
- 投喂调用**不变**：`POST /api/chat/send`，body `{message, convId, workspaceRoot}`（`paircode/paircode.go:112-148`）；改的只是 `convId`/`workspaceRoot` 指向插件声明的那个普通会话。
- 桥启动时如发现声明的会话尚未创建，**先按第三节两步建好再投喂**（`POST` 幂等，可反复调用，无副作用）。

---

## 六、插件侧自检（把问题挡在微信之外）

面板/启动时用现有只读接口做两项校验：

1. `GET /api/ai-presets` → 确认 `cfg.aiPreset` 存在；取该配置的执行模型，确认 `cfg.aiModel` 或配置自带模型非空。
2. `GET /api/conversations/{convId}?workspaceRoot=<ws>` → 确认会话存在且 `provider/model/preset` 与声明一致（`web_server.go:1411-1427`）。

任一不满足 → 面板红灯 + 写明原因（如"声明的 AI 配置不存在""该配置未指定模型"），**不要让内核的 400 文案原样发到微信联系人**（建议桥侧统一归类为"AI 配置无效，请联系管理员"）。

---

## 七、验证步骤

1. **建会话**：面板「新建会话（按声明配置）」→ `GET /api/conversations` 出现该会话；`GET /api/conversations/{id}` 能看到 `provider/model/preset` 已写入；IDE 日志出现 `[conv-model] 会话 … 仅本会话生效`。
2. **确认是普通会话**：IDE 左侧列表能看到它、点进去继续对话正常、拖动/重命名/删除等既有行为不变。
3. **桥投喂**：微信发一条 → 桥投喂该会话 → `POST /api/chat/send` 返回 **200 `{"ok":true}`**（不再 400），回复回到微信。
4. **归档**：面板归档 → 桥不再向该会话投喂；IDE 里该会话仍可见（若选方案 B 则确认已被删除）。
5. **回退保护**：把插件三项配置清空 → 桥行为回到现状（吃全局激活配置），不报错、不影响其他会话。

---

## 八、避坑清单

1. **绝不改内核**，也**不要**把模型塞进 `POST /api/chat/send`（该接口不接受 provider/model，`paircode.go:114-118` 可证）。
2. **不要写全局配置**：`settings.json` 的全局 preset/AI 配置不是插件该动的东西——那会污染用户所有对话，也正是这次故障的放大器。
3. **绑定前会话必须存在**：`SetConvModel` 对不存在的会话报 `会话 %s 不存在`（`message_store.go:1777`），所以顺序必须是"先 `POST` 建会话（幂等）→ 再 `PUT` 绑定"。
4. **`workspaceRoot` 三处必须一致**：建会话、绑模型、投喂（桥的 `workspaceRoot`）——三者不一致会落到不同 store，表现为"会话莫名不存在"。
5. **`preset` 必须真实可用**：配置名不存在或缺执行模型时，会话级装配同样报错。插件的价值就是把这件事在面板上说清楚。
6. **设置改动要经保存生效**：`ctx.getSettings` 读到的是落盘值，磁盘上直接改文件不等于运行时生效，必须走设置面板保存（或等价的保存路径）。
7. **`autoStart` 建议默认 `false`**，避免插件被装回来自动拉起常驻桥。
