# API 文档

PairCode IDE 内置了一套 HTTP API，供前端界面与后端核心功能交互。所有 API 地址均以 `/api` 开头，返回 JSON 格式数据。

以下文档列出了所有可用的 API 接口及其用途，方便你了解 IDE 各功能模块背后的交互方式。

---

## 一、健康检查

**检查 IDE 后端服务是否正常运行。**

```
GET /api/health
```

返回服务器状态、当前工作区路径和文件夹列表。

---

## 二、文件系统操作

**浏览、读写和管理工作区内的文件与目录。**

### 列出目录内容

```
GET /api/fs/list?path={目录路径}
```

返回指定目录下的文件和子目录列表。

### 读取文件内容

```
GET /api/fs/read?path={文件路径}
```

返回指定文件的文本内容。

### 写入/保存文件

```
POST /api/fs/write
```

将内容写入指定路径的文件（覆盖写入），路径不存在时自动创建。

### 搜索文件和内容

```
GET /api/fs/search?q={关键词}&path={搜索路径}
```

在工作区文件中按关键词搜索，返回匹配的文件、行号和行内容。

### 重命名文件/目录

```
POST /api/fs/rename
```

将文件或目录从原路径移动到新路径（也可用于重命名）。

### 删除文件/目录

```
POST /api/fs/delete
```

删除指定文件或目录（不可恢复）。

### 创建目录

```
POST /api/fs/mkdir
```

在指定路径创建新目录。

### 获取图片文件

```
GET /api/fs/image?path={图片路径}
```

以 Base64 编码返回图片文件数据，用于在界面中显示图片。

### 获取文件十六进制内容

```
GET /api/fs/hex?path={文件路径}&offset={偏移}&length={长度}
```

以十六进制格式返回文件的指定字节范围，用于查看二进制文件。

### 获取文件元信息

```
GET /api/fs/file-info?path={文件路径}
```

返回文件的大小、修改时间和类型等基本信息。

### 列出系统驱动器

```
GET /api/fs/drives
```

列出 Windows 系统上所有可用的磁盘驱动器（如 C:、D:）。

---

## 三、工作区管理

**查看和切换当前打开的工作区。**

```
GET /api/workspace
POST /api/workspace
```

- **GET** — 获取当前工作区根路径、文件夹列表和加载状态。
- **POST** — 切换工作区、添加文件夹或创建新工作区。

---

## 四、设置管理

**读取和修改 IDE 的全局设置。**

```
GET /api/settings
PUT /api/settings
```

- **GET** — 返回所有设置项（AI 模型配置、主题、工作区列表等）。
- **PUT** — 保存更新后的设置。

---

## 五、系统工具

**查看系统信息和执行命令。**

### 获取系统信息

```
GET /api/system/info
```

返回主机名、操作系统、当前目录等系统信息。

### 执行系统命令

```
POST /api/system/exec
```

在工作区目录下执行一条命令，并返回命令的标准输出和错误输出。

---

## 六、AI 模型

**查看可用的 AI 模型和提供商列表。**

```
GET /api/models
```

返回当前配置中所有可用的 AI 服务商和对应的模型列表。

---

## 七、对话管理

**管理 AI 对话会话，包括创建、发送消息、控制 AI 行为等。**

### 列表/创建对话

```
GET /api/conversations?workspace={工作区路径}
POST /api/conversations
```

- **GET** — 返回指定工作区下的所有对话列表。
- **POST** — 创建一个新的对话会话。

### 获取/更新/删除单条对话

```
GET /api/conversations/{convId}
PUT /api/conversations/{convId}
DELETE /api/conversations/{convId}
```

- **GET** — 获取指定对话的元数据。
- **PUT** — 更新对话标题。
- **DELETE** — 删除指定对话及其所有消息。

### 获取/添加消息

```
GET /api/conversations/{convId}/messages?limit={数量}&before={偏移}
POST /api/conversations/{convId}/messages
```

- **GET** — 分页获取对话的消息列表，支持向前翻页。
- **POST** — 向对话中添加一条消息。

### 获取消息数量

```
GET /api/conversations/{convId}/messages/count
```

返回指定对话的消息总数。

### 发送消息给 AI

```
POST /api/chat/send
```

将用户消息发送给 AI 处理。AI 的回复通过 WebSocket 实时推送。

### 停止 AI 响应

```
POST /api/chat/stop?convId={会话ID}
```

中断 AI 正在进行的思考和工具调用。

### 批准 AI 操作

```
POST /api/chat/approve
```

当 AI 请求审批时（如写入文件），用户通过此接口批准或拒绝。

### 提交反馈

```
POST /api/chat/feedback
```

在 AI 执行过程中，用户可以随时补充说明或纠正 AI 的行为。

---

## 八、指令与思想

**管理 AI 的系统指令和行为指导原则。**

### 读取/更新系统指令

```
GET /api/instructions?scope={作用域}
PUT /api/instructions?scope={作用域}
```

- **GET** — 读取指定作用域（system 或 project）的指令内容。
- **PUT** — 保存或更新指令内容。

### 读取/更新"思想"配置

```
GET /api/philosophy
PUT /api/philosophy
```

- **GET** — 读取当前 AI 行为指导配置。
- **PUT** — 保存 AI 行为指导文本，用于影响 AI 的决策风格。

---

## 九、任务与规划

**管理多步骤任务和开发规划的创建与跟踪。**

### 任务列表/更新

```
GET /api/tasks?convId={会话ID}
POST /api/tasks
```

- **GET** — 获取指定对话的任务列表。
- **POST** — 创建或更新任务状态。

### 任务规划获取/创建

```
GET /api/taskplan?name={规划名}
POST /api/taskplan
```

- **GET** — 读取指定名称的任务规划文档。
- **POST** — 创建、追加内容或标记完成任务规划。

---

## 十、Git 版本控制

**在对话中完成 Git 版本管理操作。**

### 查看仓库状态

```
GET /api/git/status?path={仓库路径}
```

返回当前仓库中已修改、已暂存和未跟踪的文件列表。

### 查看文件差异

```
GET /api/git/diff?path={仓库路径}&file={文件路径}
```

显示指定文件的变更内容（与最后一次提交的差异）。

### 暂存文件

```
POST /api/git/add
```

将指定文件加入暂存区，准备提交。

### 取消暂存

```
POST /api/git/reset
```

将文件从暂存区移除，保留工作区的修改。

### 提交变更

```
POST /api/git/commit
```

将暂存区的变更提交到本地仓库。

### 查看提交历史

```
GET /api/git/log?path={仓库路径}&count={数量}&file={文件路径}
```

查看仓库的提交记录，可限制数量和按文件过滤。也可以通过 `/api/git-log` 访问同一接口（避开广告拦截器）。

### 创建/切换分支

```
POST /api/git/branch
```

创建、删除或切换 Git 分支。

### 切换分支

```
POST /api/git/checkout
```

切换到指定分支。

### 暂存当前工作

```
POST /api/git/stash
```

将工作区的临时修改暂存起来，恢复干净的工作区。

### 查看暂存列表

```
GET /api/git/stash-list?path={仓库路径}
```

查看所有已暂存的工作记录。

### 更新 .gitignore

```
POST /api/git/ignore
```

向 .gitignore 文件中添加忽略规则。

### 丢弃更改

```
POST /api/git/discard
```

丢弃工作区中对指定文件的未暂存修改。

### 推送到远程

```
POST /api/git/push
```

将本地提交推送到远程仓库。

### 从远程拉取

```
POST /api/git/pull
```

从远程仓库拉取最新变更并合并到本地。

### 查看远程仓库

```
GET /api/git/remote?path={仓库路径}
```

查看当前仓库配置的远程仓库地址列表。

---

## 十一、Skills 技能

**管理工作流程模板，让 AI 在特定场景下更高效。**

### 技能列表

```
GET /api/skills/list
```

返回所有已安装的技能列表。

### 读取技能详情

```
GET /api/skills/read?name={技能名}
```

查看指定技能的详细内容。

### 删除技能

```
POST /api/skills/delete
```

从系统中移除指定技能。

---

## 十二、MCP 扩展

**管理 MCP（模型上下文协议）服务配置，扩展 AI 的能力边界。**

### MCP 服务列表

```
GET /api/mcp/list?level={层级}
```

返回已配置的 MCP 服务列表，可按层级（user/project）过滤。

### 保存 MCP 配置

```
POST /api/mcp/save
```

新增、更新或删除一个 MCP 服务配置。

---

## 十三、Token 统计

**查看 AI 模型对话的 Token 消耗情况。**

```
GET /api/tokens/stats?workspace={工作区}
```

返回指定工作区的 Token 用量统计（提示词、补全、总用量）。

---

## 十四、调试日志

**查看 AI 运行过程中的调试日志，帮助排查问题。**

### 查看日志列表

```
GET /api/debug/logs
```

返回所有调试日志的摘要列表。

### 查看单条日志

```
GET /api/debug/logs/{日志ID}
```

返回指定日志的详细内容。

---

## 十五、技能市场

**浏览、安装和刷新技能市场中的扩展。**

### 搜索技能市场

```
GET /api/marketplace/search?q={关键词}&kind={类型}
```

在技能市场中搜索可安装的 MCP 服务或技能模板。

### 安装技能

```
POST /api/marketplace/install
```

从市场安装指定的 MCP 服务或技能。

### 刷新市场列表

```
POST /api/marketplace/refresh
```

从远程刷新技能市场的缓存列表。

---

## 十六、记忆系统

**管理 AI 的跨会话记忆，让 AI 记住用户偏好和历史决策。**

### 搜索记忆

```
GET /api/memory/search?q={关键词}
```

在 AI 记忆中搜索相关内容。

### 列出记忆

```
GET /api/memory/list
```

列出所有保存的 AI 记忆。

### 重建记忆索引

```
POST /api/memory/rebuild
```

重新构建记忆搜索索引，用于数据恢复后同步。

---

## 十七、WebSocket

**实时通信通道，用于流式推送 AI 对话事件和终端数据。**

### AI 对话实时通信

```
ws://host/ws
```

所有 AI 对话的事件（思考过程、工具调用、回复完成等）通过此连接实时推送到前端。一个连接即可接收所有对话会话的事件。

### 终端实时通信

```
ws://host/api/terminal/ws
```

内置终端的输入输出通道，支持多个终端标签页的并发通信。

---

## 安全说明

- 所有 API 仅监听本地回环地址（127.0.0.1），默认不对外暴露。
- 文件操作路径限定在工作区目录范围内，防止越权访问。
- 命令执行在工作区目录下进行，受操作系统用户权限限制。
- 请勿将 IDE 的 API 端口暴露到公网或局域网。
