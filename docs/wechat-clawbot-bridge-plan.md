# 微信 ClawBot 桥（手机微信 ↔ PairCode）开发方案

> 状态：**待用户审阅** ｜ 日期：2026-09-15 ｜ 关联任务：t6（M1 桥原型）
>
> 需求：**手机微信**发消息 → PairCode 后端处理 → 回复回微信（明确不考虑 PC 微信）。
>
> 本方案基于一轮完整调研（微信侧通道 + PairCode 侧集成点均已核实/实测），
> 文末附「共识决策记录」与待确认点，请审阅。

---

## 1. 背景与调研结论

### 1.1 调研过程（摘要）

| 路线 | 结论 |
|---|---|
| PC 微信自动化（wxauto / wx4py / WeChatFerry） | ❌ 全死：wxauto 仅 3.9.x、wx4py 被微信 4.1.9+ 反制、WeChatFerry 停在 3.9.12、网页协议早已关闭 |
| iPad/Pad 协议（Gewechat / WeChatPadPro） | ❌ 高风险：Gewechat 已被微信官方打击公告震慑停运；WeChatPadPro 走付费私域模式 |
| **微信官方 ClawBot（⭐ 选定）** | ✅ **2026 年微信官方开放**：手机微信「我→设置→插件→ClawBot」，扫码绑定你自己的 AI Agent；**官方通道、零封号风险、无需公网** |

### 1.2 关键证据（全部一手核实）

1. 腾讯官方 npm 包 [`@tencent-weixin/openclaw-weixin`](https://www.npmjs.com/package/@tencent-weixin/openclaw-weixin)（MIT，v2.4.8，2026-09-01 仍在更新）——其 README 含官方《Backend API Protocol》文档，明确「接入自己后端的开发者需实现以下接口」→ **官方欢迎第三方自建后端**；
2. 45k★ 项目 CowAgent（原 chatgpt-on-wechat）已用纯 Python 实现同一协议（不依赖 OpenClaw 宿主）→ 协议可独立复刻；
3. **本机实测**：直接请求 `https://ilinkai.weixin.qq.com` 成功返回真实登录二维码（`get_bot_qrcode`），`getupdates` 返回标准协议响应（无 token 时 `{"errcode":-14}`）→ 协议在线、本机网络可达；
4. 用户手机微信（鸿蒙 + 微信 8.0.21）已确认存在 **ClawBot 入口卡片** ✅。

---

## 2. 目标与范围

### 2.1 目标（分阶段）

| 阶段 | 内容 | 交付物 |
|---|---|---|
| **M1 桥原型** | 全链路跑通：手机微信发消息 → 本机桥 → 投喂 PairCode agent → 取回复 → 发回微信 | `temp/wx-bridge/` Node.js 原型 + 真机验证记录 |
| **M2 插件化** | 固化进 PairCode 插件体系：设置面板（二维码/状态/配置）、随程序启动 | 磁盘插件包 + 桥进程（形态见 §4.2） |
| M3 增强（后续） | 媒体消息、多账号/多联系人会话隔离、输入状态（typing）、消息分片优化 | — |

### 2.2 本期明确不做

- ❌ PC 微信（用户明确排除）
- ❌ 群聊（ClawBot 协议仅支持单聊）
- ❌ 媒体消息收发（M1/M2 先文本；协议已具备，M3 补）
- ❌ 多租户/多用户并发隔离（先单账号；数据结构预留）
- ❌ 独立服务器部署（桥跑在本机，与 PairCode 同机）

---

## 3. 技术通道

### 3.1 微信侧：ClawBot / ilink bot 协议

**基础信息**

- API Base：`https://ilinkai.weixin.qq.com`（微信官方域名）
- 媒体 CDN：`https://novac2c.cdn.weixin.qq.com`（AES-128-ECB 加密传输）
- 协议形态：纯 HTTP JSON（`POST`）+ 长轮询（无 WebSocket、**无公网回调需求**）

**请求头（公共）**

```
Content-Type: application/json
AuthorizationType: ilink_bot_token
Authorization: Bearer <login-token>
X-WECHAT-UIN: <base64(random uint32)>          # 每次随机
iLink-App-Id: bot
iLink-App-ClientVersion: 131072                # 2.0.0 → major<<16|minor<<8|patch
```

请求体附加 `base_info.channel_version = "2.0.0"`。

**端点清单**

| 端点 | 用途 | 关键参数 |
|---|---|---|
| `ilink/bot/get_bot_qrcode?bot_type=3` | 获取登录二维码 | body `{local_token_list:[]}` → `{qrcode, qrcode_img_content}`（后者是二维码内容 URL） |
| `ilink/bot/get_qrcode_status` | 轮询扫码状态 | 以 `qrcode` id 轮询 → 确认后返回 **token** |
| `ilink/bot/getupdates` | **收消息（长轮询 35s）** | `{get_updates_buf}` → `{ret, msgs[], get_updates_buf, longpolling_timeout_ms}` |
| `ilink/bot/sendmessage` | **发消息** | `{msg:{to_user_id, context_token, client_id, message_type:2, message_state:2, item_list:[{type:1,text_item:{text}}]}}` |
| `ilink/bot/getconfig` | 取 typing ticket | `{ilink_user_id, context_token?}` |
| `ilink/bot/sendtyping` | 发送/取消「正在输入」 | `{ilink_user_id, typing_ticket, status(1/2)}` |
| `ilink/bot/getuploadurl` | 媒体上传预签名 | `{filekey, media_type, rawsize, rawfilemd5, filesize, ...}` |
| `ilink/bot/msg/notifystart/notifystop` | 消息状态通知（官方包存在，用途待细查） | — |

**消息结构（WeixinMessage）**

| 字段 | 说明 |
|---|---|
| `seq` / `message_id` | 序号 / 唯一 ID |
| `from_user_id` / `to_user_id` | 发送者 / 接收者 |
| `create_time_ms` | 时间戳（ms） |
| `session_id` | 会话 ID |
| `message_type` | `1`=USER，`2`=BOT |
| `message_state` | `0`=NEW，`1`=GENERATING，`2`=FINISH |
| `context_token` | **回复时必带**的会话上下文令牌 |
| `item_list` | 消息内容项数组 |

**item 类型**：`1`=文本(`text_item.text`)、`2`=图片、`3`=语音、`4`=文件、`5`=视频；媒体项含 `media {encrypt_query_param, aes_key, encrypt_type(1)}`（CDN 引用）。

**登录生命周期**

1. 获取二维码 → 渲染成图（`qrcode_img_content` 编码为二维码）→ 展示；
2. 用户手机微信扫码 + 确认授权 → 轮询拿 token → **凭据持久化**（本地 JSON）；
3. 会话过期（`errcode -14`）→ 自动重新发起扫码登录。

**参考实现（已落盘，供编码对照）**

- 官方实现（TS）：掘包解压于 `temp/wx-mobile/pkg/package/`（`dist/src/auth/login-qr.js`、`dist/src/api/api.js` 等）
- Python 版：`temp/wx-mobile/weixin_api.py` / `weixin_channel.py` / `weixin_message.py`（CowAgent）
- 官方协议文档：官方包 README（《Backend API Protocol》章节）

### 3.2 PairCode 侧：集成点（已核实源码）

**投喂（消息入）**

- `POST http://127.0.0.1:9090/api/chat/send`
- body：`{message, convId, workspaceRoot?, autonomous?, images?}`
- 行为：写入会话（`AppendPersistedUserMessageTo`）→ 启动/唤醒 agent 处理（与正常聊天同构）
- 错误：如「未配置 API key」等会返回错误文本（桥需捕获并透传微信）
- 注册位置：`cmd/companion/kernel_register.go:43`（kernel key `chat.send`）；实现 `cmd/companion/web_server.go handleChatSend`

**回复获取（消息出）——两种候选**

| 方案 | 机制 | 评估 |
|---|---|---|
| **A. 会话 JSONL tail（M1 主方案）** | 监听 `.pair/conversations/<convId>.jsonl`（每行 `{idx, message:{role, content, tool_calls?}, segments}`，实时追加）。投喂后追踪新增行：忽略 `role:tool` 与带 `tool_calls` 的中间轮，命中「`role:assistant` 且无 `tool_calls`」的最终回复 → 取 `content` 发回微信 | 简单、零侵入、离线可用；用「静默期（默认 8s 无新增）+ 最后一条为最终回复」双重判定保证完整性 |
| B. WS 事件流（`/ws` 全局事件） | 订阅宿主事件流拿更精确的「轮次完成」语义 | 更精确；M2 插件化后可用进程内事件（`ctx.on`）实现，更优 |

> M1 采用 A（最快跑通）；M2 评估 B（或「JSONL + 完成标记」方案）。回复获取是 M1 实施的第一步验证项。

**会话映射**

- 专用会话：桥为微信通道创建独立 PairCode 会话（`convId` 如 `conv_wx_clawbot`），与用户其他会话隔离；
- M3 扩展：按微信账号/联系人映射多会话（结构先预留）。

---

## 4. 总体架构

### 4.1 数据流

```
┌────────────────┐  1.发消息    ┌──────────────────────────┐
│   手机微信      │ ──────────→ │  微信官方服务器            │
│ （ClawBot 会话）│ ←────────── │  ilinkai.weixin.qq.com   │
└────────────────┘  4.收到回复   └────────────┬─────────────┘
                                             │ 2. getupdates 长轮询（拉消息）
                                             │ 4'. sendmessage（发回复）
                                 ┌───────────▼──────────────┐
                                 │  微信桥进程（本机）        │
                                 │  wechat-bridge（Node 原型）│
                                 └───────────┬──────────────┘
                                             │ 3a. 投喂 POST /api/chat/send
                                             │ 3b. tail 会话 JSONL 取回复
                                 ┌───────────▼──────────────┐
                                 │  PairCode 内核 + Agent     │
                                 │  （127.0.0.1:9090）        │
                                 └──────────────────────────┘
```

### 4.2 形态选择（分阶段）

**M1：独立 Node.js 进程**（`temp/wx-bridge/`）

- 选 Node.js 的理由：官方参考实现即 TS/JS（协议逻辑可直接对照搬运，风险最低）；本机已装 Node；`qrcode-terminal` 等轻量依赖成熟；
- 选「独立进程」的理由：先验证协议（最高风险项），不引入插件体系复杂度；出问题隔离好排查。

**M2：并入插件体系**（两个候选形态，M1 完成后再选）

| 形态 | 做法 | 优劣 |
|---|---|---|
| A. 磁盘插件包 + Node 桥进程 | 插件 `index.js` 管理页面/接口/生命周期（`ctx.http.register`、`ctx.registerSettings`、`ctx.process.runBackground` 管理桥进程），桥主体复用 M1 Node 代码（放插件 `assets/`） | 复用 M1 代码；Node 依赖需随包分发 |
| B. 磁盘插件包 + Go 独立二进制 | 桥用 Go 重写为 `bin/wechat-bridge.exe`（`plugins-src/plugins/wechat-bridge/`），插件 JS 壳调度（`ctx.binary.exec`） | 单文件自包含、与发行流程一致；需 Go 重写协议层 |

> 说明：纯 JS 插件（goja 沙箱）**不适合**本场景——沙箱无 Node API、`ctx.web` 仅支持 GET、长轮询（35s 级）与循环调度笨重；因此必须采用「独立进程/独立二进制」形态。

---

## 5. 数据设计（M1 原型）

| 数据 | 位置 | 格式 |
|---|---|---|
| 登录凭据 | `temp/wx-bridge/data/credentials.json` | `{token, base_url, saved_at, ...}`（原子写 + 权限收紧） |
| 消息游标 | （内存） | `get_updates_buf`（长轮询游标，断线重连时复用） |
| 处理状态 | `temp/wx-bridge/data/state.json` | 上次处理的消息 ID / 投喂时间戳（防重复处理） |
| 运行日志 | `temp/wx-bridge/logs/bridge-YYYYMMDD.log` | 滚动日志（收发/错误/重连） |
| 消息存档（可选） | `temp/wx-bridge/data/messages.jsonl` | 微信侧收发的原始消息（参考原文章设计） |

---

## 6. 实施步骤

### 6.1 M1（桥原型）细化

1. **回复获取机制验证**（第一步）：在 PairCode 里手动发一条消息，观察 `conv_*.jsonl` 的追加规律与「最终回复」的可判定性（静默期 + 无 tool_calls 启发式）；必要时加深对 `/ws` 的调研；
2. 搭建 `temp/wx-bridge/`（Node 工程：`package.json` + 模块拆分：`ilink.mjs` 协议层 / `paircode.mjs` 集成层 / `bridge.mjs` 主循环 / `login.mjs` 扫码登录）；
3. 协议层：二维码获取 + 轮询登录 + 凭据存取 + 自动重登（-14）；
4. 主循环：`getupdates` 长轮询 → 过滤（忽略 `message_type:2` BOT 消息/自己的发送）→ 投喂 `/api/chat/send` → tail JSONL 等待回复 → `sendmessage` 回发；
5. 边界处理：长文本分片（单条 ≤4000 字符）、投喂错误透传（如未配置 API key）、网络重试退避、断线重连；
6. **真机联调**：用户手机扫码 → 微信发消息 → 验证全链路 → 多轮对话/重启恢复测试。

### 6.2 M2（插件化）概要

1. 形态定稿（A/B 二选一）；
2. 磁盘插件包骨架：`package.json` + `index.js`（设置项：开关/状态/二维码获取/重新登录；HTTP 接口：`/api/ext/wechat-bridge/*`）+（能上就上）`client.js` 状态面板；
3. 桥进程生命周期管理（启动随插件、退出清理、崩溃重启）；
4. 回归验证 + 写入 `docs/plugin-*.md` 文档 + 知识库同步。

---

## 7. 验证方案与验收标准

### 7.1 M1 验收清单

- [ ] 二维码登录：终端展示二维码 → 手机微信扫码确认 → 登录取 token 成功
- [ ] 收消息：手机微信发「你好」→ 桥在 ≤2s 内拉到（日志可见）
- [ ] 全链路：消息投喂到专用会话 → agent 完成 → **回复（多行文本）完整发回手机微信**
- [ ] 凭据持久化：重启桥进程 → 免扫码自动上线
- [ ] 会话过期处理：模拟 -14 → 自动重新扫码流程
- [ ] 防循环：桥不响应 BOT 类型消息、不重复处理已发消息
- [ ] 连续 3 轮以上对话正常（含一条需要 agent 跑工具的长回复）

### 7.2 验证方法

- 桥日志 + PairCode 会话 JSONL 双向对照时间线；
- 微信侧实测（用户配合）截图为证；
- 异常注入：断网 30s、重启桥、重启 PairCode。

---

## 8. 风险与对策

| 风险 | 等级 | 对策 |
|---|---|---|
| 微信协议变更（更新后接口调整） | 中 | 关注官方包 CHANGELOG（v2.4.8 活跃）；`bot_agent` 声明 `PairCode/0.1` 便于官方归因；关键字段解析做防御（可选字段不崩） |
| 登录会话过期（-14） | 高（必然发生） | 自动重登：清凭据 → 重新出二维码 → 用户扫码；日志明确提示 |
| 回复获取误判（中间轮被当最终回复 / 回复被拆多发） | 中 | 静默期 + `tool_calls` 过滤 + 最小长度规则；M2 换事件机制根治 |
| 长任务等待（agent 跑几分钟） | 中 | 桥侧超时策略（如 10 分钟）+ 微信侧「处理中」提示（M3 用 sendTyping） |
| 一个微信绑定多 Agent 的共存问题（用户若也用 OpenClaw） | 低 | 实测确认；必要时在文档中说明解绑/切换方式 |
| Node 依赖/单文件分发（M2 形态） | 低 | 若走 B 形态（Go 二进制）即解决 |

---

## 9. 明确不做（重复强调）与后续挂账

- 不做 PC 微信；不做群聊；不做媒体消息（M1/M2）；不做多用户隔离（M1）；
- 挂账：消息加密存储、消息发送速率限制细化、白名单/黑名单、ClawBot 卡片内官方命令的兼容说明（写给用户的接入指引文档）。

---

## 10. 共识决策记录（Q1..Qn）

| # | 决策点 | 结论 | 状态 |
|---|---|---|---|
| Q1 | 手机微信通道选择 | **官方 ClawBot（ilink 协议）**；放弃 PC 微信、iPad 协议、公众号/企微回调等 | ✅ 已定（用户确认入口可用） |
| Q2 | 开发与文档顺序 | 先输出完整方案文档（本文件），用户审阅后再动手 | ✅ 已定（2026-09-15 用户选择） |
| Q3 | 桥形态 | M1 独立 Node 原型（temp/wx-bridge/）→ M2 插件化（磁盘插件包 + 进程/二进制） | ⏳ 待用户确认 |
| Q4 | 回复获取机制 | M1 用会话 JSONL tail（静默期+过滤）；M2 评估事件机制 | ⏳ 待用户确认 |
| Q5 | 消息范围 | 先纯文本；媒体/多账号挂 M3 | ⏳ 待用户确认 |
| Q6 | 会话映射 | 单专用会话（convId 固定），多联系人预留 | ⏳ 待用户确认 |

---

## 11. 参考资料

| 资料 | 位置 |
|---|---|
| 官方插件包（含《Backend API Protocol》） | `temp/wx-mobile/pkg/package/`（npm：`@tencent-weixin/openclaw-weixin@2.4.8`） |
| CowAgent 协议实现（Python） | `temp/wx-mobile/weixin_api.py` / `weixin_channel.py` / `weixin_message.py` |
| 接入教程（用户视角流程） | `temp/wx-mobile/runoob.html` |
| PairCode 插件开发文档 | `docs/plugin-development.md` |
| PairCode 内核接口注册表 | `cmd/companion/kernel_register.go`（chat.send 等 82 条） |
| 会话存储格式 | `.pair/conversations/<convId>.jsonl`（实测样例见本仓库） |
| 相关记忆 | `.pair/memory/`（「微信桥接可行性调研」） |
