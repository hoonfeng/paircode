# API 文档

PairCode IDE 内置了一套完整的 HTTP REST API，供 Web 前端与后端核心功能交互。所有 API 地址均以 `/api` 开头，返回 JSON 格式数据。WebSocket 用于实时通信（AI 事件流推送和终端数据）。

> **安全提示**：所有 API 仅监听本地回环地址（127.0.0.1），默认不对外暴露，确保安全性。请勿将服务端口暴露到公网或局域网。

---

## 使用说明

### 请求格式
- 查询参数直接在 URL 中传递
- POST / PUT 请求体使用 `application/json` 格式
- 所有接口返回 JSON 对象或数组

### 响应格式
成功响应直接返回数据对象或数组。错误时返回如下格式：

```json
{ "error": "错误描述信息" }
```

---

## 一、服务健康检查

检查 IDE 后端服务是否正常运行。

```
GET /api/health
```

返回服务器状态、当前工作区路径和文件夹列表。

---

## 二、文件系统操作

浏览、读写和管理工作区内的文件与目录。

| 接口 | 说明 |
|------|------|
| `GET /api/fs/list?path={目录路径}` | 列出指定目录下的文件和子目录 |
| `GET /api/fs/read?path={文件路径}` | 读取指定文件的文本内容 |
| `POST /api/fs/write` | 将内容写入指定文件（覆盖写入，自动创建目录） |
| `GET /api/fs/search?q={关键词}&path={搜索路径}` | 在工作区文件中按关键词搜索 |
| `POST /api/fs/rename` | 重命名或移动文件 / 目录 |
| `POST /api/fs/delete` | 删除指定文件或目录（不可恢复） |
| `POST /api/fs/mkdir` | 在指定路径创建新目录 |
| `GET /api/fs/image?path={图片路径}` | 以 Base64 编码返回图片文件数据 |
| `GET /api/fs/hex?path={文件路径}&offset={偏移}&length={长度}` | 返回文件的十六进制字节内容 |
| `GET /api/fs/file-info?path={文件路径}` | 获取文件大小、修改时间和类型等信息 |
| `GET /api/fs/drives` | 列出系统上所有可用的磁盘驱动器 |

---

## 三、工作区管理

查看和切换当前打开的工作区。

```
GET /api/workspace
POST /api/workspace
```

- GET — 获取当前工作区根路径、文件夹列表和加载状态
- POST — 切换工作区、添加文件夹或创建新工作区

---

## 四、设置管理

读取和修改 IDE 的全局设置（AI 模型配置、主题、工作区列表等）。

```
GET /api/settings
PUT /api/settings
```

- GET — 返回所有设置项
- PUT — 保存更新后的设置

---

## 五、系统工具

查看系统信息和执行命令。

| 接口 | 说明 |
|------|------|
| `GET /api/system/info` | 返回主机名、操作系统、当前目录等系统信息 |
| `POST /api/system/exec` | 在工作区目录下执行一条命令并返回输出 |

---

## 六、AI 模型

查看可用的 AI 模型和提供商列表。

```
GET /api/models
```

返回当前配置中所有可用的 AI 服务商和对应的模型列表。

---

## 七、对话管理

管理 AI 对话会话，包括创建、发送消息、控制 AI 行为等。

### 对话列表与创建
| 接口 | 说明 |
|------|------|
| `GET /api/conversations?workspace={工作区路径}` | 返回指定工作区下的所有对话列表 |
| `POST /api/conversations` | 创建一个新的对话会话 |

### 单条对话操作
| 接口 | 说明 |
|------|------|
| `GET /api/conversations/{convId}` | 获取指定对话的元数据 |
| `PUT /api/conversations/{convId}` | 更新对话标题 |
| `DELETE /api/conversations/{convId}` | 删除指定对话及其所有消息 |

### 消息管理
| 接口 | 说明 |
|------|------|
| `GET /api/conversations/{convId}/messages?limit={数量}&before={偏移}` | 分页获取对话的消息列表 |
| `POST /api/conversations/{convId}/messages` | 向对话中添加一条消息 |
| `GET /api/conversations/{convId}/messages/count` | 返回指定对话的消息总数 |

### AI 交互
| 接口 | 说明 |
|------|------|
| `POST /api/chat/send` | 将用户消息发送给 AI 处理，回复通过 WebSocket 实时推送 |
| `POST /api/chat/stop?convId={会话ID}` | 中断 AI 正在进行的思考和工具调用 |
| `POST /api/chat/approve` | 批准或拒绝 AI 请求审批的操作（如写入文件） |
| `POST /api/chat/feedback` | 在 AI 执行过程中补充说明或纠正行为 |

---

## 八、指令与思想

管理 AI 的系统指令和行为指导原则。

| 接口 | 说明 |
|------|------|
| `GET /api/instructions?scope={作用域}` | 读取指定作用域的指令内容 |
| `PUT /api/instructions?scope={作用域}` | 保存或更新指令内容 |
| `GET /api/philosophy` | 读取当前 AI 行为指导配置 |
| `PUT /api/philosophy` | 保存 AI 行为指导文本 |

---

## 九、任务与规划

管理多步骤任务和开发规划的创建与跟踪。

| 接口 | 说明 |
|------|------|
| `GET /api/tasks?convId={会话ID}` | 获取指定对话的任务列表 |
| `POST /api/tasks` | 创建或更新任务状态 |
| `GET /api/taskplan?name={规划名}` | 读取指定名称的任务规划文档 |
| `POST /api/taskplan` | 创建、追加内容或标记完成任务规划 |

---

## 十、Git 版本控制

在对话中完成 Git 版本管理操作。

| 接口 | 说明 |
|------|------|
| `GET /api/git/status?path={仓库路径}` | 查看仓库状态（已修改、已暂存、未跟踪文件） |
| `GET /api/git/diff?path={仓库路径}&file={文件路径}` | 显示指定文件的变更内容 |
| `POST /api/git/add` | 将指定文件加入暂存区 |
| `POST /api/git/reset` | 将文件从暂存区移除 |
| `POST /api/git/commit` | 提交暂存区的变更 |
| `GET /api/git/log?path={仓库路径}&count={数量}&file={文件路径}` | 查看提交历史 |
| `POST /api/git/branch` | 创建、删除或切换分支 |
| `POST /api/git/checkout` | 切换到指定分支 |
| `POST /api/git/stash` | 将工作区临时修改暂存起来 |
| `GET /api/git/stash-list?path={仓库路径}` | 查看所有已暂存的工作记录 |
| `POST /api/git/ignore` | 向 .gitignore 中添加忽略规则 |
| `POST /api/git/discard` | 丢弃工作区的未暂存修改 |
| `POST /api/git/push` | 将本地提交推送到远程仓库 |
| `POST /api/git/pull` | 从远程仓库拉取最新变更 |
| `GET /api/git/remote?path={仓库路径}` | 查看远程仓库地址列表 |

---

## 十一、Skills 技能

管理工作流程模板，让 AI 在特定场景下更高效。

| 接口 | 说明 |
|------|------|
| `GET /api/skills/list` | 返回所有已安装的技能列表 |
| `GET /api/skills/read?name={技能名}` | 查看指定技能的详细内容 |
| `POST /api/skills/delete` | 从系统中移除指定技能 |

---

## 十二、MCP 扩展

管理 MCP（模型上下文协议）服务配置，扩展 AI 的能力边界。

| 接口 | 说明 |
|------|------|
| `GET /api/mcp/list?level={层级}` | 返回已配置的 MCP 服务列表 |
| `POST /api/mcp/save` | 新增、更新或删除一个 MCP 服务配置 |

---

## 十三、Token 统计

查看 AI 模型对话的 Token 消耗情况。

```
GET /api/tokens/stats?workspace={工作区}
```

返回指定工作区的 Token 用量统计（提示词、补全、总用量）。

---

## 十四、调试日志

查看 AI 运行过程中的调试日志，帮助排查问题。

| 接口 | 说明 |
|------|------|
| `GET /api/debug/logs` | 返回所有调试日志的摘要列表 |
| `GET /api/debug/logs/{日志ID}` | 返回指定日志的详细内容 |

---

## 十五、技能市场

浏览、安装和刷新技能市场中的扩展。

| 接口 | 说明 |
|------|------|
| `GET /api/marketplace/search?q={关键词}&kind={类型}` | 在市场搜索可安装的扩展 |
| `POST /api/marketplace/install` | 从市场安装指定的扩展 |
| `POST /api/marketplace/refresh` | 刷新市场的缓存列表 |

---

## 十六、记忆系统

管理 AI 的跨会话记忆，让 AI 记住你的偏好和历史决策。

| 接口 | 说明 |
|------|------|
| `GET /api/memory/search?q={关键词}` | 在 AI 记忆中搜索相关内容 |
| `GET /api/memory/list` | 列出所有保存的 AI 记忆 |
| `POST /api/memory/rebuild` | 重新构建记忆搜索索引 |

---

## 十七、WebSocket 实时通信

用于流式推送 AI 对话事件和终端数据。

| 连接地址 | 用途 |
|----------|------|
| `ws://host/ws` | AI 对话的事件推送（思考过程、工具调用、回复完成等），一个连接接收所有会话事件 |
| `ws://host/api/terminal/ws` | 内置终端的输入输出通道，支持多个终端标签页并发通信 |
