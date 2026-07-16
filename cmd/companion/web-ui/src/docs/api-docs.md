# API 文档

PairCode IDE 内置了一套完整的 HTTP REST API + WebSocket 实时通信协议，供 Web 前端与后端核心功能交互，也**支持第三方开发者基于本 API 进行二次开发**。所有 API 地址均以 `/api` 开头，返回 JSON 格式数据。

> **安全提示**：所有 API 仅监听本地回环地址（127.0.0.1），默认不对外暴露。请勿将服务端口暴露到公网或局域网。

---

## 通用约定

### 请求格式
- 查询参数（GET）直接在 URL 中传递
- POST / PUT 请求体使用 `application/json`
- 无特殊说明时，Content-Type 为 `application/json`

### 响应格式
| 场景 | 格式 | 说明 |
|------|------|------|
| 成功 | JSON 对象 或 JSON 数组 | 直接返回业务数据 |
| 错误 | `{"error": "错误描述信息"}` | HTTP 状态码 4xx/5xx |

### 错误码惯例
| HTTP 状态码 | 含义 |
|-------------|------|
| 200 | 成功 |
| 400 | 参数错误 / 请求体错误 |
| 404 | 资源不存在 |
| 405 | 方法不允许（如 GET 用了 POST） |
| 500 | 服务器内部错误 |

---

## 一、服务健康检查

检查 IDE 后端服务是否正常运行。

```
GET /api/health
```

**响应示例：**
```json
{
  "status": "ok",
  "workspace": "F:/projects/my-app",
  "folders": ["F:/projects/my-app"]
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| status | string | 固定 `"ok"` |
| workspace | string | 当前工作区路径 |
| folders | string[] | 工作区包含的文件夹列表 |

---

## 二、文件系统操作

浏览、读写和管理工作区内的文件与目录。

### 2.1 列出目录

```
GET /api/fs/list?path={目录路径}
```

**参数：**
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| path | string | 否 | 目录路径，省略时返回工作区根目录 |

**响应示例：**
```json
[
  {"name": "src", "isDir": true, "size": 4096, "modTime": "2026-07-11T10:00:00Z"},
  {"name": "main.go", "isDir": false, "size": 2048, "modTime": "2026-07-11T09:30:00Z"}
]
```

| 字段 | 类型 | 说明 |
|------|------|------|
| name | string | 文件/目录名 |
| isDir | boolean | 是否为目录 |
| size | number | 文件大小（字节） |
| modTime | string | 最后修改时间（ISO 8601） |

---

### 2.2 读取文件

```
GET /api/fs/read?path={文件路径}
```

**参数：**
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| path | string | 是 | 文件路径 |

**响应：** 返回文件文本内容（字符串）。

---

### 2.3 写入文件

```
POST /api/fs/write
```

**请求体：**
```json
{
  "path": "src/main.go",
  "content": "package main\n\nfunc main() {\n\tprintln(\"hello\")\n}\n"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| path | string | 是 | 文件路径（相对于工作区或绝对路径） |
| content | string | 是 | 文件内容（覆盖写入，自动创建目录） |

**响应：** 空对象 `{}`（200）。

---

### 2.4 搜索文件内容

```
GET /api/fs/search?q={关键词}&path={搜索路径}
```

**参数：**
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| q | string | 是 | 搜索关键词 |
| path | string | 否 | 搜索目录，省略时使用工作区根目录 |

**响应示例：**
```json
[
  {"file": "src/main.go", "line": 15, "text": "func handleRequest(w http.ResponseWriter, r *http.Request) {"},
  {"file": "src/utils.go", "line": 42, "text": "// handleRequest 处理 HTTP 请求"}
]
```

| 字段 | 类型 | 说明 |
|------|------|------|
| file | string | 文件相对路径 |
| line | number | 行号 |
| text | string | 匹配行的内容 |

**自动忽略的目录：** `.git`、`node_modules`、`vendor`、`.pair`、`__pycache__`、`bin` 等。**仅搜索文本文件扩展名**（`.go` `.js` `.ts` `.vue` `.html` `.css` `.json` `.md` `.py` `.rs` `.java` 等 50+ 种）。

---

### 2.5 重命名/移动文件

```
POST /api/fs/rename
```

**请求体：**
```json
{
  "oldPath": "src/old.go",
  "newPath": "src/new.go"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| oldPath | string | 是 | 原路径 |
| newPath | string | 是 | 新路径 |

**响应：** `{"success": true}`

---

### 2.6 删除文件/目录

```
POST /api/fs/delete
```

**请求体：**
```json
{
  "path": "src/temp.go"
}
```

> ⚠️ 不可恢复，递归删除目录及其所有内容。

**响应：** `{"success": true}`

---

### 2.7 创建目录

```
POST /api/fs/mkdir
```

**请求体：**
```json
{
  "path": "src/new-folder"
}
```

**响应：** `{"success": true}`

---

### 2.8 获取图片数据

```
GET /api/fs/image?path={图片路径}
```

**参数：** `path` — 图片文件路径（支持 PNG / JPEG）

**响应：** Base64 编码的图片数据字符串（不含 `data:image/...` 前缀）。

**响应头：** `Content-Type: text/plain; charset=utf-8`

---

### 2.9 获取文件信息

```
GET /api/fs/file-info?path={文件路径}
```

**响应示例：**
```json
{
  "name": "main.go",
  "path": "F:/projects/my-app/src/main.go",
  "size": 2048,
  "modTime": "2026-07-11T09:30:00Z",
  "isDir": false
}
```

---

### 2.10 十六进制查看

```
GET /api/fs/hex?path={文件路径}&offset={偏移}&length={长度}
```

**参数：**
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| path | string | 是 | 文件路径 |
| offset | number | 否 | 起始字节偏移（默认 0） |
| length | number | 否 | 读取字节数（默认 512，最大 4096） |

**响应示例：**
```json
{
  "hex": "4d5a90000300000004000000ffff0000b80000000000000040",
  "text": "MZ.............@",
  "offset": 0,
  "length": 32
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| hex | string | 十六进制字符串 |
| text | string | ASCII 可打印字符（不可打印的替换为 `.`） |
| offset | number | 起始偏移 |
| length | number | 返回的字节数 |

---

### 2.11 列出磁盘驱动器

```
GET /api/fs/drives
```

**响应示例：**
```json
["C:\\", "D:\\", "E:\\"]
```

---

## 三、工作区管理

### 3.1 获取当前工作区

```
GET /api/workspace
```

**响应示例：**
```json
{
  "root": "F:/projects/my-app",
  "folders": ["F:/projects/my-app"],
  "loaded": true
}
```

### 3.2 切换/设置工作区

```
POST /api/workspace
```

**请求体（切换工作区）：**
```json
{
  "path": "F:/projects/another-project"
}
```

**请求体（添加文件夹）：**
```json
{
  "addFolder": "F:/projects/shared-lib"
}
```

**请求体（创建新工作区）：**
```json
{
  "create": "F:/projects/new-project"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| path | string | 按场景 | 切换工作区到指定路径 |
| addFolder | string | 按场景 | 在当前工作区添加文件夹 |
| create | string | 按场景 | 创建新目录并切换为其工作区 |

**响应：** 返回更新后的工作区信息（同 GET 响应格式）。

---

## 四、设置管理

### 4.1 读取设置

```
GET /api/settings
```

**响应示例：**
```json
{
  "apiKey": "sk-xxx",
  "baseURL": "https://api.openai.com/v1",
  "model": "gpt-4",
  "theme": "dark",
  "planModel": "claude-3-opus",
  "autoReview": true,
  "autoCommit": false,
  "maxTokens": 4096,
  "temperature": 0.7,
  "thinkingMode": "non-thinking",
  "recentWorkspaces": ["F:/projects/app1", "F:/projects/app2"]
}
```

### 4.2 保存设置

```
PUT /api/settings
```

**请求体：** 与 GET 返回格式相同，只需传入要修改的字段。

**响应：** `{"success": true}`

---

## 五、系统工具

### 5.1 系统信息

```
GET /api/system/info
```

**响应示例：**
```json
{
  "hostname": "DESKTOP-ABC123",
  "cwd": "F:/projects/my-app",
  "os": "windows",
  "goos": "windows",
  "workspace": "F:/projects/my-app",
  "folders": ["F:/projects/my-app"],
  "version": "v1.0.0"
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| hostname | string | 主机名 |
| cwd | string | 当前工作目录 |
| os | string | 操作系统名称 |
| goos | string | Go 平台标识 |
| workspace | string | IDE 工作区根路径 |
| folders | string[] | 工作区文件夹列表 |
| version | string | IDE 版本号（由打包器注入） |

### 5.2 执行命令

```
POST /api/system/exec
```

**请求体：**
```json
{
  "command": "go build ./cmd/app",
  "timeout": 30
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| command | string | 是 | 要执行的命令 |
| timeout | number | 否 | 超时秒数（默认 30，最大 120） |

**响应示例：**
```json
{
  "stdout": "# github.com/foo/app\nsrc/main.go:42: undefined: x\n",
  "stderr": "",
  "exitCode": 2
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| stdout | string | 标准输出 |
| stderr | string | 标准错误 |
| exitCode | number | 退出码（0 = 成功） |

> **安全限制：** 命令在工作区目录下执行；禁止交互式命令（如 `vim`）。

---

## 六、AI 模型

### 获取可用模型列表

```
GET /api/models
```

**响应示例：**
```json
{
  "providers": [
    {
      "name": "openai",
      "models": ["gpt-4", "gpt-4-turbo", "gpt-3.5-turbo"]
    },
    {
      "name": "claude",
      "models": ["claude-3-opus", "claude-3-sonnet", "claude-3-haiku"]
    }
  ],
  "current": {
    "provider": "openai",
    "model": "gpt-4"
  }
}
```

---

## 七、对话管理

### 7.1 对话列表

```
GET /api/conversations?workspace={工作区路径}
```

**参数：**
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| workspace | string | 否 | 工作区路径，省略时使用当前工作区 |

**响应示例：**
```json
[
  {
    "id": "conv_1741680000000",
    "title": "修复登录页面样式",
    "createdAt": "2026-07-11T10:00:00Z",
    "messageCount": 12,
    "workspace": "F:/projects/my-app"
  }
]
```

### 7.2 创建对话

```
POST /api/conversations
```

**请求体：**
```json
{
  "title": "新对话",
  "workspace": "F:/projects/my-app"
}
```

**响应：** 返回创建的对话对象（同 GET 列表中的格式）。

### 7.3 获取对话详情

```
GET /api/conversations/{convId}
```

**响应：** 返回单个对话的元数据对象。

### 7.4 更新对话

```
PUT /api/conversations/{convId}
```

**请求体：**
```json
{
  "title": "新的标题"
}
```

**响应：** `{"success": true}`

### 7.5 删除对话

```
DELETE /api/conversations/{convId}
```

**响应：** `{"success": true}`（同时删除该对话的所有消息）。

### 7.6 获取消息列表

```
GET /api/conversations/{convId}/messages?limit={数量}&before={偏移}
```

**参数：**
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| limit | number | 否 | 返回消息条数（默认 50） |
| before | number | 否 | 偏移量（跳过前 N 条，用于分页） |

**响应示例：**
```json
[
  {
    "role": "user",
    "content": "帮我写一个 HTTP 服务",
    "createdAt": "2026-07-11T10:00:00Z"
  },
  {
    "role": "assistant",
    "content": "好的，我来创建一个简单的 HTTP 服务...",
    "createdAt": "2026-07-11T10:00:05Z"
  }
]
```

| 字段 | 类型 | 说明 |
|------|------|------|
| role | string | `"user"` \| `"assistant"` \| `"system"` |
| content | string | 消息内容 |
| createdAt | string | 创建时间（ISO 8601） |

### 7.7 添加消息

```
POST /api/conversations/{convId}/messages
```

**请求体：**
```json
{
  "role": "user",
  "content": "继续上一个话题"
}
```

### 7.8 消息总数

```
GET /api/conversations/{convId}/messages/count
```

**响应：** `{"count": 42}`

### 7.9 发送消息给 AI（非阻塞）

```
POST /api/chat/send
```

**请求体：**
```json
{
  "message": "帮我创建一个 Go HTTP 服务",
  "sessionId": "sess_xxx",
  "convId": "conv_1741680000000",
  "autonomous": false,
  "workspaceRoot": "F:/projects/my-app"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| message | string | 是 | 用户消息内容（最长 50000 字符） |
| sessionId | string | 否 | 会话 ID |
| convId | string | 否 | 对话 ID（自动生成若留空） |
| autonomous | boolean | 否 | 是否启用自主模式（默认 false） |
| workspaceRoot | string | 否 | 工作区路径（默认当前工作区） |

**响应：** `{"sessionId": "sess_xxx", "convId": "conv_1741680000000"}`

AI 的回复不在此响应的 Body 中返回，而是通过 **WebSocket 实时推送**事件流（见第十七章）。

**前置条件：** 必须先配置 API Key 和模型。

---

### 7.10 停止 AI 响应

```
POST /api/chat/stop?convId={会话ID}
```

**参数：** `convId` — 要停止的对话 ID。

**响应：** `{"success": true}`

---

### 7.11 审批操作

```
POST /api/chat/approve
```

**请求体：**
```json
{
  "convId": "conv_xxx",
  "approved": true,
  "feedback": "请把函数名改为驼峰命名法"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| convId | string | 是 | 对话 ID |
| approved | boolean | 是 | 批准（true）或拒绝（false） |
| feedback | string | 否 | 拒绝时的反馈/纠正建议 |

---

### 7.12 发送反馈

```
POST /api/chat/feedback
```

**请求体：**
```json
{
  "convId": "conv_xxx",
  "content": "请改用更简洁的实现方式"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| convId | string | 是 | 对话 ID |
| content | string | 是 | 反馈/纠正内容 |

**工作原理：** 在 AI 下次 LLM 调用前，将反馈内容作为 `[User]` 消息注入本轮上下文，让 AI 在下一次回复中响应用户的补充或纠正。

---

## 八、指令与思想

### 8.1 读取指令

```
GET /api/instructions?scope={作用域}
```

**参数：** `scope` — 指令作用域（如 `"system"`、`"user"`）。

**响应：** 返回指令文本内容（字符串）。

### 8.2 保存指令

```
PUT /api/instructions?scope={作用域}
```

**请求体：** 纯文本字符串（指令内容）。

**响应：** `{"success": true}`

### 8.3 读取行为指导

```
GET /api/philosophy
```

**响应：** 返回 AI 行为指导配置文本。

### 8.4 保存行为指导

```
PUT /api/philosophy
```

**请求体：** 纯文本字符串。

**响应：** `{"success": true}`

---

## 九、任务与规划

### 9.1 获取任务列表

```
GET /api/tasks?convId={会话ID}
```

**响应示例：**
```json
[
  {
    "id": "task_1",
    "subject": "创建 HTTP 服务文件",
    "status": "completed",
    "description": "在 src/server.go 创建 HTTP 服务"
  },
  {
    "id": "task_2",
    "subject": "添加路由处理",
    "status": "in_progress"
  }
]
```

### 9.2 创建/更新任务

```
POST /api/tasks
```

**请求体：**
```json
{
  "convId": "conv_xxx",
  "tasks": [
    {"id": "task_1", "subject": "创建 HTTP 服务", "status": "in_progress"}
  ]
}
```

### 9.3 读取任务规划

```
GET /api/taskplan?name={规划名}
```

### 9.4 保存任务规划

```
POST /api/taskplan
```

**请求体：**
```json
{
  "name": "refactor-auth",
  "content": "## 重构计划\n1. 提取认证中间件\n2. 添加 JWT 支持",
  "action": "append"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | 规划名称 |
| content | string | 是 | 规划内容（Markdown） |
| action | string | 否 | `"append"`（追加）\| `"create"`（创建）\| `"done"`（标记完成），默认 `"append"` |

---

## 十、Git 版本控制

所有 Git API 均在**当前工作区目录**（或指定仓库路径）下执行。

### 10.1 仓库状态

```
GET /api/git/status?path={仓库路径}
```

**参数：**
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| path | string | 否 | 仓库路径（默认当前工作区） |

**响应示例：**
```json
{
  "branch": "main",
  "changes": [
    {"path": "src/main.go", "status": "M", "staged": false},
    {"path": "src/utils.go", "status": "M", "staged": true}
  ],
  "untracked": ["src/new.go"],
  "ahead": 1,
  "behind": 0
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| branch | string | 当前分支名 |
| changes[].path | string | 变更文件路径 |
| changes[].status | string | 状态码：`M`(修改) `A`(新增) `D`(删除) `R`(重命名) |
| changes[].staged | boolean | 是否已暂存 |
| untracked | string[] | 未跟踪文件列表 |
| ahead | number | 领先远程的提交数 |
| behind | number | 落后远程的提交数 |

### 10.2 查看差异

```
GET /api/git/diff?path={仓库路径}&file={文件路径}
```

**参数：**
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| path | string | 否 | 仓库路径 |
| file | string | 否 | 指定文件（省略则返回所有变更的 diff） |

**响应：** 返回 diff 文本（字符串）。

### 10.3 暂存文件

```
POST /api/git/add
```

**请求体：**
```json
{
  "path": "F:/projects/my-app",
  "files": ["src/main.go", "src/utils.go"]
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| path | string | 否 | 仓库路径（默认工作区） |
| files | string[] | 否 | 要暂存的文件列表（省略则暂存全部） |

**响应：** `{"success": true}`

### 10.4 取消暂存

```
POST /api/git/reset
```

**请求体：** 格式同 `git/add`。

### 10.5 提交

```
POST /api/git/commit
```

**请求体：**
```json
{
  "path": "F:/projects/my-app",
  "message": "feat: 添加用户认证模块"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| path | string | 否 | 仓库路径 |
| message | string | 是 | 提交信息 |

**响应：**
```json
{
  "success": true,
  "hash": "a1b2c3d4e5f6..."
}
```

### 10.6 查看提交历史

```
GET /api/git/log?path={仓库路径}&count={数量}&file={文件路径}
```

**参数：**
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| path | string | 否 | 仓库路径 |
| count | number | 否 | 返回条数（默认 15） |
| file | string | 否 | 限定某文件的提交历史 |

**响应示例：**
```json
[
  {
    "hash": "a1b2c3d",
    "author": "user",
    "date": "2026-07-11 10:00:00",
    "message": "feat: 添加用户认证模块"
  }
]
```

### 10.7 分支管理

```
POST /api/git/branch
```

| 操作 | 请求体 | 说明 |
|------|--------|------|
| 创建 | `{"path":"...","name":"feature-x","action":"create"}` | 创建新分支 |
| 删除 | `{"path":"...","name":"feature-x","action":"delete"}` | 删除分支 |
| 列表 | `{"path":"...","action":"list"}` | 列出所有分支 |
| 切换 | `{"path":"...","name":"feature-x","action":"checkout"}` | 切换分支 |

**响应：** 列表操作返回 `["main", "feature-x", ...]`，其他返回 `{"success": true}`。

### 10.8 切换分支

```
POST /api/git/checkout
```

**请求体：**
```json
{
  "path": "F:/projects/my-app",
  "branch": "feature-x",
  "file": "src/main.go"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| branch | string | 按场景 | 切换到的分支名 |
| file | string | 按场景 | 恢复指定文件到 HEAD（branch 和 file 二选一） |

### 10.9 贮藏

```
POST /api/git/stash
```

**请求体：**
```json
{
  "path": "F:/projects/my-app",
  "action": "push",
  "message": "暂存当前 WIP"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| path | string | 否 | 仓库路径 |
| action | string | 否 | `"push"`(贮藏,默认) \| `"pop"`(恢复) \| `"drop"`(丢弃) |
| message | string | 否 | 贮藏备注 |

### 10.10 查看贮藏列表

```
GET /api/git/stash-list?path={仓库路径}
```

**响应示例：**
```json
[
  {"index": 0, "message": "暂存当前 WIP"},
  {"index": 1, "message": "On feature-x: 临时保存"}
]
```

### 10.11 添加忽略规则

```
POST /api/git/ignore
```

**请求体：**
```json
{
  "path": "F:/projects/my-app",
  "patterns": ["*.log", ".env", "build/"]
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| patterns | string[] | 是 | 要添加到 `.gitignore` 的规则列表 |

### 10.12 丢弃修改

```
POST /api/git/discard
```

**请求体：**
```json
{
  "path": "F:/projects/my-app",
  "files": ["src/main.go"]
}
```

> ⚠️ 不可恢复！丢弃工作区未暂存的修改。

### 10.13 推送

```
POST /api/git/push
```

**请求体：**
```json
{
  "path": "F:/projects/my-app",
  "remote": "origin",
  "branch": "main"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| remote | string | 否 | 远程名（默认 `"origin"`） |
| branch | string | 否 | 分支名（默认当前分支） |

### 10.14 拉取

```
POST /api/git/pull
```

**请求体：**
```json
{
  "path": "F:/projects/my-app",
  "remote": "origin",
  "branch": "main"
}
```

### 10.15 查看远程仓库

```
GET /api/git/remote?path={仓库路径}
```

**响应示例：**
```json
{
  "origin": "https://github.com/user/repo.git"
}
```

---

## 十一、Skills 技能

### 11.1 技能列表

```
GET /api/skills/list
```

**响应示例：**
```json
[
  {
    "name": "code-review",
    "description": "代码审查工作流",
    "mode": "auto",
    "version": "1.0"
  }
]
```

### 11.2 读取技能

```
GET /api/skills/read?name={技能名}
```

**响应：** 返回技能的完整 Markdown 内容。

### 11.3 删除技能

```
POST /api/skills/delete
```

**请求体：**
```json
{
  "name": "code-review"
}
```

**响应：** `{"success": true}`

---

## 十二、MCP 扩展

### 12.1 MCP 列表

```
GET /api/mcp/list?level={层级}
```

**参数：**
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| level | string | 否 | 层级过滤（如 `"user"`、`"project"`） |

### 12.2 MCP 保存

```
POST /api/mcp/save
```

**请求体：**
```json
{
  "name": "my-db",
  "command": "node",
  "args": ["mcp-server-db/index.js"],
  "scope": "project"
}
```

此接口同时支持新增、更新和删除（删除时传 `"action": "delete"`）。

---

## 十三、Token 统计

### 获取 Token 用量

```
GET /api/tokens/stats?workspace={工作区}
```

**响应示例：**
```json
{
  "workspace": "F:/projects/my-app",
  "promptTokens": 125000,
  "completionTokens": 45000,
  "totalTokens": 170000,
  "cost": 0.85
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| promptTokens | number | 提示词 Token 数 |
| completionTokens | number | 补全 Token 数 |
| totalTokens | number | 总 Token 数 |
| cost | number | 估算费用（美元） |

---

## 十四、调试日志

### 14.1 日志列表

```
GET /api/debug/logs
```

**响应示例：**
```json
[
  {"id": "log_001", "time": "2026-07-11T10:00:00Z", "session": "sess_xxx", "summary": "工具调用: read_file src/main.go"}
]
```

### 14.2 日志详情

```
GET /api/debug/logs/{日志ID}
```

**响应：** 返回指定日志的完整内容。

---

## 十五、技能市场

### 15.1 搜索市场

```
GET /api/marketplace/search?q={关键词}&kind={类型}
```

**参数：**
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| q | string | 否 | 搜索关键词 |
| kind | string | 否 | 类型（`"mcp"`、`"skill"`、`"all"`） |

### 15.2 安装扩展

```
POST /api/marketplace/install
```

**请求体：**
```json
{
  "id": "skill-code-review",
  "scope": "project"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | 扩展 ID |
| scope | string | 否 | 安装范围（`"user"`、`"project"`） |

### 15.3 刷新市场缓存

```
POST /api/marketplace/refresh
```

**响应：** `{"success": true}`

---

## 十六、记忆系统

### 16.1 搜索记忆

```
GET /api/memory/search?q={关键词}
```

**响应示例：**
```json
[
  {"name": "项目编码规范", "description": "使用驼峰命名法", "type": "project", "content": "..."}
]
```

### 16.2 记忆列表

```
GET /api/memory/list
```

### 16.3 重建索引

```
POST /api/memory/rebuild
```

---

## 十七、WebSocket 实时通信协议

PairCode IDE 使用 **WebSocket** 替代传统 SSE，实现双向实时通信。支持两种 WebSocket 端点：

### 17.1 AI 事件推送

```
ws://127.0.0.1:{port}/ws
```

**用途：** 接收 AI 对话的事件流（思考过程、工具调用、回复内容、错误等）。

**协议：** 纯文本帧（JSON），**服务端单向推送**，客户端无需发送任何消息。

#### 事件类型总表

| 事件类型 | 说明 | 前端展示 |
|---------|------|---------|
| `thinking` | LLM 思考链增量 | 流式显示思考过程（斜体/灰色） |
| `content` | LLM 正文回复增量 | 流式显示正文内容 |
| `tool_call` | AI 即将执行某工具 | 显示工具调用卡片（工具名+参数） |
| `tool_result` | 工具执行结果返回 | 显示结果摘要 |
| `usage` | Token 用量统计 | 更新 Token 计数器 |
| `approval` | 请求用户审批写类操作 | 显示审批对话框（含工具名、参数、文件路径） |
| `error` | 出错或触发止损 | 显示错误信息 |
| `done` | 本次 AI 回复完成 | 关闭加载状态 |
| `compacted` | 上下文已压缩（旧消息被摘要替换） | 显示一条素色提示 |
| `evaluation` | 自主模式任务评分 | 显示评分卡 |
| `circling` | 检测到 AI 重复绕圈 | 显示"换思路"提示 |
| `notice` | 后台任务通知 | 显示一条素色提示 |
| `phase` | 自主模式阶段切换 | 显示阶段指示器（规划/执行/评测） |
| `final` | 单轮委托完成（delegate 用） | 同 done |

#### 事件 JSON 格式

```json
{
  "type": "thinking",
  "content": "我来分析一下这个需求...",
  "tool": "",
  "args": "",
  "callId": "",
  "agentName": "",
  "usage": null,
  "doneReason": ""
}
```

| 字段 | 类型 | 必含 | 说明 |
|------|------|------|------|
| type | string | 是 | 事件类型（见上表） |
| content | string | 按场景 | thinking/content/error/final 时携带文本内容 |
| tool | string | 按场景 | tool_call/tool_result 时携带工具名（如 `"read_file"`、`"edit_file"`） |
| args | string | 按场景 | tool_call 时携带工具参数的 JSON 字符串 |
| callId | string | 按场景 | 工具调用 ID，用于关联 tool_call → tool_result |
| agentName | string | 按场景 | 事件来源 Agent 名。空串=主 Agent，非空=子 Agent（用于区分） |
| usage | object | 按场景 | EventUsage 时携带 token 用量：`{promptTokens:N, completionTokens:N, totalTokens:N}` |
| doneReason | string | 按场景 | EventDone 时携带完成原因（如 `"completed"`、`"stopped"`、`"error"`） |

#### 典型事件序列

```
→ {type:"thinking", content:"我来分析一下..."}
→ {type:"tool_call", tool:"read_file", args:"{\"path\":\"main.go\"}", callId:"call_1"}
→ {type:"tool_result", tool:"read_file", content:"文件内容...", callId:"call_1"}
→ {type:"thinking", content:"看到文件结构了，接下来..."}
→ {type:"tool_call", tool:"edit_file", args:"{\"path\":\"main.go\",\"content\":\"...\"}", callId:"call_2"}
→ {type:"approval", tool:"edit_file", args:"{\"path\":\"main.go\"}", callId:"call_2"}
   （等待用户审批 → 调用 POST /api/chat/approve）
→ {type:"tool_result", tool:"edit_file", content:"文件已更新", callId:"call_2"}
→ {type:"content", content:"已完成修改，以下是改动内容..."}
→ {type:"usage", content:"", usage:{promptTokens:1200, completionTokens:350, totalTokens:1550}}
→ {type:"done", doneReason:"completed"}
```

> **重要：** WebSocket 连接为全局单连接，推送**所有**会话的事件。事件中的 `convId` 字段（若存在）用于区分不同对话。前端需根据 `convId` 路由到对应的对话面板。

---

### 17.2 终端 WebSocket

```
ws://127.0.0.1:{port}/api/terminal/ws
```

**用途：** 内置终端的双向输入输出通道，每连接对应一个 PTY 终端会话。

#### 协议规则

| 帧类型 | 方向 | 说明 |
|--------|------|------|
| 文本帧 (JSON) | 客户端→服务端 | 控制消息 |
| 文本帧 (JSON) | 服务端→客户端 | 状态通知 |
| 二进制帧 | 双向 | 原始 PTY I/O 字节流（含 VT 转义序列，由 xterm.js 渲染） |

#### 控制消息格式

**客户端 → 服务端（初始化）：**
```json
{"type": "init", "shell": "cmd", "cwd": "F:/projects/my-app"}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| type | string | 是 | 固定 `"init"` |
| shell | string | 是 | Shell 名：`"cmd"` \| `"powershell"` \| `"gitbash"`（白名单限制） |
| cwd | string | 是 | 工作目录（禁止穿越出工作区） |

**客户端 → 服务端（调整大小）：**
```json
{"type": "resize", "cols": 120, "rows": 30}
```

**服务端 → 客户端：**
```json
{"type": "ready"}
{"type": "error", "msg": "shell 不在白名单中"}
{"type": "closed"}
```

#### 安全措施

- Shell 白名单：仅允许 `cmd`、`powershell`、`gitbash`
- `cwd` 路径校验：禁止穿越出工作区
- PTY 关闭时强制终止子进程
- 并发 PTY 会话数限制：最多 16 个

---

## 附录：API 索引速查

### 基础 API
| 方法 | 端点 | 用途 |
|------|------|------|
| GET | `/api/health` | 健康检查 |
| GET | `/api/system/info` | 系统信息+版本号 |
| POST | `/api/system/exec` | 执行命令 |

### 文件系统 (10 个)
| 方法 | 端点 | 用途 |
|------|------|------|
| GET | `/api/fs/list` | 列出目录 |
| GET | `/api/fs/read` | 读取文件 |
| POST | `/api/fs/write` | 写入文件 |
| GET | `/api/fs/search` | 搜索内容 |
| POST | `/api/fs/rename` | 重命名 |
| POST | `/api/fs/delete` | 删除 |
| POST | `/api/fs/mkdir` | 创建目录 |
| GET | `/api/fs/image` | 图片 Base64 |
| GET | `/api/fs/file-info` | 文件信息 |
| GET | `/api/fs/hex` | 十六进制查看 |
| GET | `/api/fs/drives` | 磁盘驱动器列表 |

### 工作区 & 设置
| 方法 | 端点 | 用途 |
|------|------|------|
| GET/POST | `/api/workspace` | 工作区管理 |
| GET/PUT | `/api/settings` | 设置管理 |

### AI 对话 (6 个)
| 方法 | 端点 | 用途 |
|------|------|------|
| POST | `/api/chat/send` | 发送消息给 AI |
| POST | `/api/chat/stop` | 停止 AI 回复 |
| POST | `/api/chat/approve` | 审批操作 |
| POST | `/api/chat/feedback` | 发送反馈 |
| GET | `/api/models` | 可用模型列表 |

### 对话管理 (7 个)
| 方法 | 端点 | 用途 |
|------|------|------|
| GET | `/api/conversations` | 对话列表 |
| POST | `/api/conversations` | 创建对话 |
| GET | `/api/conversations/{id}` | 对话详情 |
| PUT | `/api/conversations/{id}` | 更新对话 |
| DELETE | `/api/conversations/{id}` | 删除对话 |
| GET | `/api/conversations/{id}/messages` | 消息列表 |
| POST | `/api/conversations/{id}/messages` | 添加消息 |
| GET | `/api/conversations/{id}/messages/count` | 消息总数 |

### Git (15 个)
| 方法 | 端点 | 用途 |
|------|------|------|
| GET | `/api/git/status` | 仓库状态 |
| GET | `/api/git/diff` | 查看差异 |
| POST | `/api/git/add` | 暂存 |
| POST | `/api/git/reset` | 取消暂存 |
| POST | `/api/git/commit` | 提交 |
| GET | `/api/git/log` | 提交历史 |
| POST | `/api/git/branch` | 分支管理 |
| POST | `/api/git/checkout` | 切换分支/恢复文件 |
| POST | `/api/git/stash` | 贮藏 |
| GET | `/api/git/stash-list` | 贮藏列表 |
| POST | `/api/git/ignore` | 添加忽略规则 |
| POST | `/api/git/discard` | 丢弃修改 |
| POST | `/api/git/push` | 推送 |
| POST | `/api/git/pull` | 拉取 |
| GET | `/api/git/remote` | 远程仓库地址 |

### 扩展 & 系统
| 方法 | 端点 | 用途 |
|------|------|------|
| GET | `/api/skills/list` | 技能列表 |
| GET | `/api/skills/read` | 读取技能 |
| POST | `/api/skills/delete` | 删除技能 |
| GET | `/api/mcp/list` | MCP 列表 |
| POST | `/api/mcp/save` | MCP 保存 |
| GET | `/api/tokens/stats` | Token 统计 |
| GET | `/api/debug/logs` | 调试日志列表 |
| GET | `/api/debug/logs/{id}` | 调试日志详情 |
| GET | `/api/memory/search` | 搜索记忆 |
| GET | `/api/memory/list` | 记忆列表 |
| POST | `/api/memory/rebuild` | 重建索引 |
| GET | `/api/marketplace/search` | 市场搜索 |
| POST | `/api/marketplace/install` | 安装扩展 |
| POST | `/api/marketplace/refresh` | 刷新缓存 |
| GET/PUT | `/api/instructions` | 指令管理 |
| GET/PUT | `/api/philosophy` | 行为指导 |
| GET/POST | `/api/tasks` | 任务管理 |
| GET/POST | `/api/taskplan` | 规划管理 |

---

### WebSocket 端点
| 端点 | 用途 |
|------|------|
| `ws://host/ws` | AI 事件流推送（思考/工具/结果/完成） |
| `ws://host/api/terminal/ws` | PTY 终端双向 I/O |
