# wechat-bridge — 微信 ClawBot 桥（手机微信 ⇄ PairCode）

以「JS 插件壳 + Go 常驻桥」形态交付的双向桥：手机微信（官方 ClawBot 协议）里的
消息投喂进 PairCode 会话，**回合最终回复自动回发微信**；支持媒体收发与 Agent 主动发消息。

> **平台要求：Windows x64**（`bin/wxbridge.exe` 为 Go/CGO 构建的 Windows 二进制；
> 其他平台需自行从源码构建，见「开发」）。

## 结构

```
wechat-bridge/
├── package.json        # 插件元数据（main=index.js, client=client.js）
├── index.js            # host 半：拉起/监督桥进程（ctx.process.runBackground）、
│                       #   事件流解析、HTTP 代理（/api/ext/wechat-bridge/*）、
│                       #   Agent 工具 wechat_send / wechat_contacts
├── client.js           # client 半：标题栏「微信」按钮 + 管理面板（状态/账号/控制/日志）
├── bin/wxbridge.exe    # Go 常驻桥（源码 plugins-src/plugins/wechat-bridge/）
└── README.md
```

## 使用

1. 「设置 → 插件 → 微信桥（wechat-bridge）」：勾选 **启用微信桥**；
   如需随宿主启动自动运行，再勾选 **启动时自动运行**；
2. 标题栏「微信」按钮 → 面板「启动」；
3. 「打开扫码页」（或设置里的「扫码登录」链接）→ 手机微信扫码 → 账号转入运行态；
4. 手机微信给机器人发消息：消息进 PairCode 会话，**回合结束即自动回复最终结果**
   （过程文本不发出；回复带节流保护，失败自动兜底，不丢消息）。

数据目录：`<工作区>/.pair/wechat-bridge/`（账号凭据/状态/实例锁）。

## 回复通道

- **只发最终回复**：实时流式转发默认关闭（过程文本会全量外泄）；回合结束后
  done 事件即发，JSONL 文件轮询作兜底（含会话「交接」重建自愈）；
- **发送节流**：最小间隔 + 短/长窗口条数限制 + 限流（ret=-2）冷静期退避；
- **等待上限**：单轮回复等待 30 分钟（超时给出提示，不会静默卡死）。

## 媒体

- **接收**：对方发来的图片/视频/文件自动落盘并投喂会话；
- **发送**：`wechat_send` 工具 / 本地 HTTP `/send` 支持文本与 `file`
  （图片/视频/文档，经微信 CDN 加密上传；需对方先发过消息建立 context_token）。

## 开发

- 改 Go 桥：`plugins-src/plugins/wechat-bridge/`（主 module 子包，不 import internal）；
- 构建并更新本插件 `bin/`：`node scripts/build-wx-bridge.mjs`
- 同步到安装目录（宿主只装载安装目录的插件）：
  `node scripts/build-wx-bridge.mjs --sync --install-dir D:/PairCode`

## 退出清理（无残留进程）

- 正常停止（面板「停止」）：壳发 `{"cmd":"stop"}` → 桥优雅退出，1.5s 超时强杀兜底；
- 插件卸载/停用：`ctx.effect` 停桥；
- 宿主退出（Ctrl+C / SIGTERM）：内核 `KillAllBackgroundProcesses()` 终止全部后台进程；
- 桥自守护：stdin EOF（壳/宿主退出）→ 桥自行退出，防孤儿残留。

## 已知边界

- 扫码页为 302 重定向到桥的本地回环页面（插件路由响应体仅支持字符串，PNG 二进制
  无法经壳代理）；
- 主动发送需要对方先给桥发过消息（建立 context_token）；
- 实时流式转发为关闭状态（改 `config/config.go` 的 `StreamEnabled` 并重编译可恢复；
  不建议——会把运行过程文本全部发到微信）。
