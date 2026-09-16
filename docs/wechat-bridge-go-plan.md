# 微信桥 Go 插件化 —— 开发方案（v1.0 · 已确认）

> **状态：已确认，本文档为开发依据** ｜ 确认日期：2026-09-15 ｜ 前置：`docs/wechat-clawbot-bridge-plan.md`（M1 Node 原型已跑通）
>
> **用户拍板记录（本轮，全部确认）**：
> - ①插件形态与原有插件运行方式一致 ②二维码实时生成不存档 ③退出清理三层+内核小改「一并补上」
> - ④多账号「为以后多用户做准备，必须要稳」 ⑤主动发微信工具框架集成（可补丁增强） ⑥双路完整移植
> - ⑦凭据放工作区 + conv 沿用 + 全局单实例；无现成 WS 客户端 → **自研做一个**；其余拍板按建议：设置面板加 link 字段（A）、每账号独立会话、发语音挂账、桥内置 WS/QR vendor/端口 9097。

---

## 一、目标与范围

**目标**：微信（官方 ClawBot 协议）⇄ PairCode 双向桥，以「与现有插件 100% 同构」的 JS 插件壳 + Go 常驻进程形态交付；多账号（为多用户预留）、媒体收发、Agent 主动发送、退出零残留。

**不在本期**：微信群聊（协议无此概念）、语音消息发送（协议未见先例，挂账）、多用户权限体系（仅预留 `ownerUserId` 字段）。

**交付物**：
1. `.pair/plugins/wechat-bridge/` 插件包（package.json / index.js / client.js / bin/wechat-bridge.exe / README）；
2. `plugins-src/plugins/wechat-bridge/` Go 源码树（主 module 内）；
3. 内核小改 ×2（`internal/agent/shell.go` 退出清理、`cmd/companion/main.go` 钩子调用；前端 `SettingsModal.vue` 加 `link` 字段类型）；
4. `scripts/build-wx-bridge.mjs` 构建脚本；
5. 文档（按 doc-org 规范：插件文档 + API 契约）。

---

## 二、模块设计

### 2.1 Go 桥（`plugins-src/plugins/wechat-bridge/`，主 module 内）

> 原则：插件自持实现，不 import internal、不引第三方重依赖（与 tool-binary 范式一致）。仅 vendor 一个纯 Go QR 编码库进 `qr/`。

| 包 | 职责 | M1 对应 | 关键设计 |
|---|---|---|---|
| `main.go` | 入口：flag 解析、单实例、自守护、编排 | main.mjs | `--data-dir --paircode --port --dump` |
| `ctl/` | stdio 控制面：命令解析（行 JSON）/事件输出 | — | stdin goroutine + 命令队列；**EOF=退出** |
| `ilink/` | 协议客户端：构造头部、5 端点、超时/重试 | ilink.mjs | `Client.Do(method, path, body) (json, error)`；35s 长轮询超时视为空批 |
| `login/` | 扫码登录状态机（8 态）+ 二维码会话 | login.mjs | 内存持有 QR（不落盘）；30 次/60 分钟窗口；`scaned_but_redirect` 切 base_url |
| `account/` | 账号注册表/凭据/状态持久化；多实例编排 | main.mjs+store.mjs | 见 §2.1.1；每账号一个 `Runtime` 结构 |
| `bridge/` | 主循环：长轮询/队列/typing/分片/双路等待 | bridge.mjs+paircode.mjs | 每账号独立 goroutine 组；见 §2.1.2 |
| `stream/` | **自研 WS 客户端** + 事件订阅 + 流式切片 | ws.mjs+eventstream.mjs+stream.mjs | 见 §2.1.3 |
| `media/` | 媒体收发：CDN 下载/上传、AES-128-ECB、item 构造 | CowAgent 规格 | 见 §6.4 |
| `localhttp/` | 回环 HTTP 面（/send /status /qr …） | — | 127.0.0.1，端口自动避让，token 校验 |
| `store/` | 原子 JSON 读写 | store.mjs | `.tmp` + rename（Windows 兼容） |
| `qr/` | QR 编码（vendored）→ PNG 渲染 | 新增 | 候选 `rsc.io/qr` / `skip2/go-qrcode` |

#### 2.1.1 多账号模型（★ 必须稳）

```
<workspace>/.pair/wechat-bridge/
├── accounts.json           # 账号注册表
├── accounts/<id>/
│   ├── credentials.json    # token/baseUrl/botId/ilinkUserId（对齐 M1 creds）
│   └── state.json          # getUpdatesBuf/processedIds/contextTokens/peerTyping
├── media/<accountId>/      # 媒体暂存（收：下载解密；发：待上传）
└── bridge.lock             # 单实例锁
```

`accounts.json`：
```json
{ "accounts": [ { "id":"bot_xxx", "alias":"工作号", "convId":"conv_wx_work",
  "workspaceRoot":"", "ownerUserId":"local", "enabled":true,
  "addedAt":"ISO", "lastMsgAt":"ISO" } ] }
```

**稳定性设计（逐条）**：
1. **隔离**：每账号独立 `Runtime{ token / 长轮询 goroutine / 消息队列(channel) / typing 票 / 媒体目录 / 错误状态 }`；一个账号的 panic 被 `recover` 拦截并标记该账号 `error` 态，**不影响其他账号与主进程**；
2. **状态机**：账号级 `login → running → stale(重试中) → offline(放弃)`；-14 重试 6 次（[2,4,8,16,32,60]s）失败 → 该账号转 `stale` 触发重登（重新扫码），其他账号照常；
3. **持久化**：凭据/状态每次变更原子落盘 → 任意重启免扫码续跑；
4. **幂等**：账号增删接口幂等（重复添加同 botId 返回已有条目）；
5. **资源限制**：每账号队列上限 256 条、媒体文件上限 100MB（超出拒绝并告警），防单账号耗尽全局；
6. **多用户预留**：`ownerUserId`（当前恒 `"local"`）+ 会话命名 `conv_wx_<alias>` 天然隔离；未来接入用户体系时按 owner 过滤/绑定，无需重构数据布局。

#### 2.1.2 桥主循环（每账号）

```
inbox 轮询(35s) → 消息解析(文本/媒体) → 队列 → worker(串行):
  投喂 POST /api/chat/send {msg, images?, ...} → 记 baseLen(JSONL 行数)
  → 双路等待：
     ├ WS 流式：eventstream 订阅 convId → stream.mjs 切片参数 → sendmessage 逐片
     └ JSONL 兜底：poll 1.5s 增量读 → 静默 15s 且无 tool_calls 判定完成
  两路竞速：WS done 先到 → 前缀校验；JSONL 先到 → 中断 WS 流按前缀补发
  → 存 contextTokens/processedIds 落盘 → 下一条
ask_user：检测到问询 → 中途通知（quiet 15s 触发）→ 用户回复继续等待
```

#### 2.1.3 自研 WS 客户端（stream/wsclient.go）

- **规格来源**：M1 `ws.mjs`（183 行，已实测跑通：握手 Sec-WebSocket-Key/Accept、客户端**掩码帧**、文本/Ping/Pong/Close、分片大小写）。
- **不引第三方库**（用户确认「没有现成的就做一个」）；风格对照项目既有手写 WS（`internal/agent/wsconn.go` 服务端帧层可参考逻辑，但**不复用代码**——它是服务端语义、无掩码）。
- 能力清单：`Dial(url) → WriteText/Close/WaitClose`、自动回 Pong（读 goroutine）、事件回调 `OnMessage/OnClose/OnError`。
- 重连：`eventstream` 层指数退避 1s→30s（照 eventstream.mjs）；连接重置退避=收到消息后。
- 自测：`--dump ws-probe` 连本机 9090 `/ws` 打印握手/心跳/一条事件即过半。

### 2.2 JS 壳（`.pair/plugins/wechat-bridge/`）

**package.json**（对齐 agent-teams）：
```json
{ "name": "wechat-bridge", "purpose": "微信 ClawBot 桥：手机微信 ⇄ PairCode 双向（多账号/媒体/主动发送）",
  "scope": "global", "type": "plugin", "version": "1.0.0", "main": "index.js", "client": "client.js" }
```

**index.js（host 半，`return {name, purpose, inject, apply(ctx)}`）**：
- `apply`：读设置 → 注册设置/schema、HTTP 接口、工具；`enabled && autoStart` → 拉起桥；
- 生命周期：`ctx.process.runBackground(bin, args)` → 3s `ctx.interval` 增量解析 stdout → 状态内存化 + `ctx.emit('client:wechat-bridge:state'…)`；`done && exitErr` → 退避重启（2s→60s，上限 5 次转告警）；`ctx.effect` 注册清理（stop 命令 → 2s 超时 kill）；
- HTTP（`ctx.webServer.register`，前缀 `/api/ext/wechat-bridge/`）：`/qr`(代理桥 HTML)、`/qr.png`(代理)、`/status`、`/send`、`/accounts`(+add/remove)、`/refresh`——浏览器侧全部走壳（桥端口对内隐藏）；
- 工具：`wechat_send` / `wechat_contacts`（同步 `ctx.web.post` 调桥本地 HTTP，见 §5.5）。

**client.js（client 半，`(ui) => {…}`）**：
- 标题栏图标（未读/状态角标，SVG）+ 面板：账号列表（别名/状态/最后消息/二维码入口/重登/移除）、「添加账号」、启停开关、连接状态；
- 数据：`ui.on('wechat-bridge:state')` 事件驱动刷新（2s 事件轮询由浏览器侧宿主机制带）；操作 `ui.invoke` 回 host；二维码**不内嵌**——面板里给「打开扫码页」链接（跳设置中的同一 URL，浏览器新标签实时刷新）；
- 纯 DOM + 主题 CSS 变量（照 agent-teams/client.js 规范，禁 emoji 图标）。

### 2.3 内核小改 ×3（精确落点，已核实）

| # | 文件 | 改动 | 依据 |
|---|---|---|---|
| 1 | `internal/agent/shell.go` | 新增 `func KillAllBackgroundProcesses()`：锁 `globalBG.mu` → 遍历 `procs` → 未 done 的 `killProcessTree(cmd.Process.Pid)`（复用 shell.go:511 既有函数）→ 清表 | globalBG 定义 shell.go:210（包级单例） |
| 2 | `cmd/companion/main.go` | 退出钩子（现 :58-68 仅 `agent.CloseAllMCPConnections()`）追加 `agent.KillAllBackgroundProcesses()` + 日志 | 已读现状 |
| 3 | `plugins-src/ui-app/src/components/SettingsModal.vue` | 字段渲染链（:38-82 v-if 串）新增 `link` 分支：`<a :href="abs(f.href||f.default)" target="_blank">{{ f.linkText \|\| '打开' }}</a>`（相对路径补 `location.origin`）；保存跳过列表（:234 `continue` 条件）加入 `'link'`（纯展示字段不落设置） | 已读现状 |

> 改动 #1/#2 即用户要求「子进程泄漏退出环节同类缺口一并补上」：覆盖**所有**后台进程（agent 的 dev server、其他插件后台进程、本桥），配合桥 stdin-EOF 自杀形成双保险。改动 #3 为「设置链接」方案 A（通用增强，所有插件受益）。

---

## 三、接口契约

### 3.1 stdio（壳 ↔ 桥）

**壳→桥（stdin，行 JSON）**：`status` / `stop` / `relogin {account}` / `verify_code {account, code}` / `accounts.remove {account}` / `qr.refresh {account}`

**桥→壳（stdout，行 JSON）**：
```json
{"evt":"hello","version":"1.0.0","pid":1234,"port":9097,"reused":false,"accounts":["bot_x"]}
{"evt":"status","accounts":[{"id":"bot_x","alias":"工作号","phase":"running","lastMsgAt":"…","queue":0}]}
{"evt":"account","account":"bot_x","phase":"login","qr":true,"verifyNeeded":false}
{"evt":"log","level":"info","msg":"…"}
{"evt":"error","account":"bot_x","msg":"…","fatal":false}
```
说明：`phase ∈ login|running|stale|offline`；`hello.port` 供壳拼桥本地 URL（工具调用用）。

### 3.2 桥本地 HTTP（127.0.0.1:<port>，回环；写操作带 `X-Bridge-Token`）

| 端点 | 方法 | 入参 | 出参 | 用途 |
|---|---|---|---|---|
| `/send` | POST | `{account?, to, text?, file?}` | `{ok, error?, sentAt}` | 工具同步发送（60s 超时） |
| `/status` | GET | `?account=` | 全账号/单账号状态 | 壳 & 扫码页轮询 |
| `/qr` | GET | `?account=` | HTML（内嵌 img，JS 轮询 /status） | 扫码页（**实时渲染**） |
| `/qr.png` | GET | `?account=` | image/png | 二维码图片（每次现渲染） |
| `/qr/refresh` | POST | `{account}` | `{ok}` | 手动重取 |
| `/login/new` | POST | `{alias?}` | `{accountId}` | 添加账号 |
| `/relogin` | POST | `{account}` | `{ok}` | 重新登录 |
| `/accounts` | GET/POST/DELETE | — | JSON | 账号管理 |
| `/contacts` | GET | `?account=` | `{contacts:[…]}` | 联系人表（工具/UI） |
| `/health` | GET | — | `{ok,pid}` | 心跳/单实例探测 |

### 3.3 设置 schema（`ctx.registerSettings`）

```js
{ key:'wechat-bridge', title:'微信桥（wechat-bridge）', fields:[
  { name:'enabled',      label:'启用微信桥',        type:'checkbox', default:false },
  { name:'autoStart',    label:'启动时自动运行',     type:'checkbox', default:false },
  { name:'defaultConvId',label:'默认会话 ID',       type:'text',     default:'conv_wx_clawbot' },
  { name:'workspaceRoot',label:'默认工作区（空=主）', type:'text',     default:'' },
  { name:'qrLink',       label:'扫码登录',          type:'link',     href:'/api/ext/wechat-bridge/qr', linkText:'打开扫码页' },
]}
```

### 3.4 Agent 工具 schema

```js
{ name:'wechat_send',
  description:'主动向微信联系人发送消息。to=联系人 id/别名；file=工作区内路径（自动判型：png/jpg→图片，mp4→视频，其他→文件）',
  parameters:{ type:'object', properties:{
    to:{type:'string'}, text:{type:'string'}, file:{type:'string'} }, required:['to'] },
  execute: async (args) => { /* ctx.web.post(http://127.0.0.1:<port>/send, {…, _token}) */ } }

{ name:'wechat_contacts', description:'列出微信联系人（id/别名/最后消息时间），供 wechat_send 选择收件人',
  parameters:{ type:'object', properties:{ account:{type:'string'} } } }
```
> 扩展约定：`/send` 入参为可扩展命令集（后续加 `voice/emoji/location/card` 只增字段与桥内实现，壳与工具契约不变）。

### 3.5 数据格式（关键字段）

```jsonc
// credentials.json（对齐 M1 login.mjs creds）
{ "token":"…", "baseUrl":"https://ilinkai.weixin.qq.com", "botId":"bot_…",
  "ilinkUserId":"…", "savedAt":"ISO" }
// state.json（对齐 M1 bridge.mjs:36-38）
{ "getUpdatesBuf":"…", "processedIds":["…"], "contextTokens":{"<contactId>":"…"},
  "peerTyping":false, "updatedAt":"ISO" }
```

---

## 四、关键流程（时序摘要）

1. **启动**：宿主装载插件 → `apply` 起桥 → 桥读账号 → 无凭据账号进登录态（内存 QR）→ `hello` 事件 → 壳缓存 port/状态 → UI 面板刷新。
2. **退出**：四条路径全部收口——正常停机（壳 stop→kill）/ 插件停用（ctx.effect）/ 宿主 Ctrl+C（main.go 钩子 → KillAllBackgroundProcesses + 桥 stdin EOF 自杀）/ 强杀（桥 stdin EOF 秒退 + 心跳兜底）。验收 = **任何路径下 10s 内无残留进程**。
3. **扫码**：设置/面板点链接 → 壳 `/api/ext/wechat-bridge/qr` → 代理桥 `/qr` → 页面实时二维码（自动过期重取/手动刷新）→ 扫码（verify 码走 stdin 命令）→ 凭据落盘 → 账号转 `running`。
4. **消息收/回**：手机发消息 → 长轮询 → 投喂 → WS 流式逐片回发（首片 20 字/1s 必发；后续 160 字/2.5s；上限 1500/句末优先）→ JSONL 兜底校验 → 处理完落 state。
5. **媒体**：收（CDN 下载→AES 解密→图片走 `images` 多模态投喂；其他落盘 `media/<account>/` + 路径附文本）；发（读工作区文件→随机 key→加密→getuploadurl→上传→item 发送）。
6. **主动发送**：Agent 调 `wechat_send` → 壳同步 `ctx.web.post` 调桥 `/send` → 桥上传/发送 → 结果返回工具。

---

## 五、任务分解与里程碑

| # | 任务 | 产出/验收 | 依赖 |
|---|---|---|---|
| P0-1 | 媒体消息 raw 抓取（真机发图/文件/语音） | 实测 JSON 样本（校准 media 字段与 CDN 域） | 用户真机 |
| P0-2 | 第二微信扫码绑定探测（多账号可行性） | 结论+证据（支持→照设计；不支持→启用预案 §7） | 用户第二部手机 |
| P0-3 | QR vendor + PNG 渲染 PoC | 控制台输出可扫 PNG（`--dump qr`） | — |
| P1-1 | 桥骨架：main/ctl/store/config/单实例 | `--dump selftest` 通过 | P0-3 |
| P1-2 | ilink 客户端 | `--dump qr-fetch` 拿到二维码内容 | P1-1 |
| P1-3 | login 状态机 + 二维码会话 | `--dump login` 真机扫码→凭据落盘 | P1-2 |
| P1-4 | account 管理（多账号结构/持久化/隔离） | 两套假凭据并发启动互不影响 | P1-3 |
| P1-5 | localhttp（/send /status /qr …）+ 自守护（EOF/心跳） | `--dump` 各端点 curl 验证；关 stdin 秒退 | P1-1 |
| P2-1 | paircode 投喂 + JSONL 增量兜底 | `--dump reply` 拿到完整回复文本 | P1-3 |
| P2-2 | 自研 WS 客户端 + eventstream | `--dump ws-probe` 连本机 /ws 收事件 | P2-1 |
| P2-3 | 流式切片 + 主循环（typing/分片/双路竞速） | 手机端可见逐片流式回复 | P2-1/2-2 |
| P2-4 | media 收发全链路 | 手机发图→PairCode 多模态可见；agent 发图→手机收到 | P2-3、P0-1 |
| P2-5 | ask_user 中途通知 + -14 重登联动 | M1 对应行为复现 | P2-3 |
| P3-1 | 插件 index.js：装载/设置/schema | 插件出现在设置面板 | P1 |
| P3-2 | 生命周期：runBackground/监督重启/effect 清理 | 停用插件→进程消失 | P3-1 |
| P3-3 | 壳 HTTP 接口（代理/状态/send/accounts） | curl 全绿 | P3-1 |
| P3-4 | 工具注册 wechat_send/wechat_contacts | Agent 对话中主动发消息成功 | P3-3 |
| P3-5 | **内核小改 #1/#2**（KillAllBackgroundProcesses+钩子） | 启动 dev server 后 Ctrl+C→无残留 | 独立 |
| P3-6 | **内核小改 #3**（SettingsModal link 字段） | 设置面板出现可点链接 | 独立 |
| P4-1 | client.js 面板（账号/状态/控制/链接） | 浏览器可操作全流程 | P3 |
| P4-2 | 标题栏图标 + 事件联动 | 状态实时（≤2s） | P4-1 |
| P5-1 | 构建脚本 build-wx-bridge.mjs（含 exe 占用检测） | 一条命令出包 | P1 |
| P5-2 | 端到端回归（§6 验收清单） | 全项通过 | P2/P3/P4 |
| P5-3 | 文档落盘（doc-org：插件/API） | docs/ 分类归档 | P5-2 |

---

## 六、测试与验收计划

**单元**：登录状态机 8 态分支、切片参数（对照 M1 行为逐条）、AES-128-ECB 加解密 roundtrip（三种 aes_key 格式容错）、JSONL 增量解析（跨行/半行）、QR 渲染字节有效性。

**专项**：
- 退出清理 4 路径（正常停机/插件停用/宿主 Ctrl+C/任务管理器强杀）→ 均 10s 内无残留；
- 多账号并发（P0-2 结论支持时：2 账号各收各回互不串；不支持时：单账号 + 切换流程验证）；
- 媒体（4 类型收、3 类型发、超限拒绝）；
- 断网/断线（WS 重连 1s→30s、长轮询超时自愈）。

**端到端验收清单（P5-2）**：M1 七项（扫码/收消息/全链路/免扫码重启/-14 重登/防循环/3 轮对话）＋ 新增 5 项（多账号、媒体收发、退出清理、工具主动发、二维码实时无落盘）。

---

## 七、风险与预案

| 风险 | 预案 |
|---|---|
| 协议限制同 bot 多微信绑定（P0-2） | 降级 A：多「bot 实例池」（每用户独立凭据集并行，桥结构已按多账号建，仅登录来源不同）；降级 B：单账号+切换；如实报告实测 |
| 媒体字段与 CowAgent 参考有出入 | P0-1 真机 sample 为准，media 包留字段容错 |
| WS 客户端兼容性（握手/掩码/分片） | 对照 ws.mjs 已跑通规范逐条移植 + `--dump ws-probe` 自测；不行退 JSONL 兜底（功能不受损） |
| exe 更新被占用 | 更新流程强制「停桥→覆盖→重启」；build 脚本检测并提示 |
| 「卡 100%」复现 | M1 的 context_tokens 落盘/processedIds 防重放/quiet 判定已沿用；P0-1 抓 sample 时专项观察 |
| 网络受限（vendor 下载） | gh-proxy 已实测可用；QR 库源码内嵌入仓（含 LICENSE） |

---

## 八、变更记录

- v1.0（2026-09-15）：由「实施方案 v2（待确认）」升级——13 项决策全部拍板；内核小改落点核实到文件:行；补任务分解/测试/风险。
- v2（2026-09-15）：M2 修订（7 项要求调研 + 全部源码核实）。
- v1（2026-09-14）：M1 方案（Node 原型，`docs/wechat-clawbot-bridge-plan.md`）。

---

## 附录 A：M1 规格速查（Go 移植参数表）

| 项 | 值 | 来源 |
|---|---|---|
| 端点 | get_bot_qrcode?bot_type=3 / get_qrcode_status / getupdates / sendmessage / getconfig / sendtyping / getuploadurl | ilink.mjs |
| 头 | AuthorizationType / Authorization Bearer / X-WECHAT-UIN(base64 随机 uint32) / iLink-App-Id:bot / iLink-App-ClientVersion:131072 / channel_version 2.0.0 | ilink.mjs:14-24 |
| 长轮询 | 35s（超时=空批） | getUpdates |
| -14 重试 | 6 次，等待 [2,4,8,16,32,60]s | bridge.mjs:11 |
| 分片 | 4000 字符（段/行优先） | splitText |
| typing | 5s 保活，status 1/2 | bridge.mjs |
| 回复等待 | poll 1.5s / quiet 15s / timeout 30min | config.mjs |
| 流式切片 | 首片 20 字+1s 必发；后续 160 字+2.5s；上限 1500；句末优先 | stream.mjs |
| 双路 | WS 主 + JSONL 兜底 + done 前缀校验补发 | bridge.mjs:_awaitReply |
| 媒体 CDN | https://novac2c.cdn.weixin.qq.com（AES-128-ECB + PKCS7） | 方案文档 §3.1 |
| 媒体 item | 1=文本 2=图片 3=语音 4=文件 5=视频；media{encrypt_query_param, aes_key, encrypt_type:1} | CowAgent |
| 上传 | media_type 1=图/2=视频/3=文件；filekey/aeskey(hex)/rawsize/rawfilemd5/filesize；回参 x-encrypted-param | CowAgent |

## 附录 B：已核实机制索引（文件:行）

- 插件装载/热更新：`internal/plugins/toolset.go`（LoadGlobalPlugins）、`jsplugin.go`（define/load/cleanup）
- 进程能力：`jsplugin.go:940-1101`（runBackground/readOutput/writeStdin/kill）；`shell.go:210` globalBG（包级单例）、`:511` killProcessTree
- 工具注册：`jsplugin.go:543`；同步语义 `:3998`/`:2363`
- 退出链：`cmd/companion/main.go:58-68`（现仅清 MCP——本次补 KillAllBackgroundProcesses）
- 静态资产暴露面：`web_server.go:395`（/plugins-assets）→ 凭据不进插件目录
- 投喂：`web_server.go:2348` `/api/chat/send`；多模态 `:2360` `agent.ImagePart`
- JSONL：`loop.go:172` OnBatchPersist 步级写；文件全量重写→按 idx 增量
- 设置面板：`SettingsModal.vue:38-82` 字段渲染链、`:234` 保存跳过列表（新增 link 类型）
- 样板插件：`agent-teams`（settings+client+工具）、`tool-binary`（Go 源码树+bin/ 范式）
