# API 文档

PairCode IDE 内置 HTTP API 服务，供前端界面与后端交互。所有 API 前缀为 `/api`。

---

## 安全说明

- **路径限制**：所有文件操作路径限定在工作区根目录下，禁止通过 `../` 等路径穿越访问上级目录
- **只读安全**：文件写入/删除等操作仅限工作区目录，不会影响到系统关键文件
- **命令执行**：命令执行在工作区目录下进行，受系统权限限制
- **认证机制**：API 仅接受本地回环地址（127.0.0.1 / localhost）访问，默认不暴露到局域网

---

## 健康检查

获取服务器状态与当前工作区信息。

```
GET /api/health
```

**返回示例：**
```json
{
  "workspace": "{当前工作区路径}",
  "folders": ["{工作区文件夹列表}"],
  "version": "1.0.0"
}
```

---

## 文件系统

### 列出目录

```
GET /api/fs/list?path={工作区内目录路径}
```

返回目录下的文件和子目录列表。

### 读取文件

```
GET /api/fs/read?path={工作区内文件路径}
```

返回文件文本内容。

### 写入文件

```
POST /api/fs/write
```

**请求体：**
```json
{
  "path": "{工作区内文件路径}",
  "content": "文件内容"
}
```

### 搜索文件内容

```
GET /api/fs/search?q={关键词}&path={搜索路径}
```

按正则表达式在工作区文件中搜索内容，返回匹配的文件、行号及行内容。

### 重命名/移动

```
POST /api/fs/rename
```

**请求体：**
```json
{
  "from": "源路径",
  "to": "目标路径"
}
```

### 删除文件

```
DELETE /api/fs/delete?path={文件路径}
```

### 创建目录

```
POST /api/fs/mkdir
```

**请求体：**
```json
{
  "path": "{目录路径}"
}
```

### 读取图片（Base64）

```
GET /api/fs/image?path={图片路径}
```

### 读取二进制预览

```
GET /api/fs/hex?path={文件路径}&offset={偏移}&length={长度}
```

### 获取文件信息

```
GET /api/fs/file-info?path={文件路径}
```

### 获取驱动器列表

```
GET /api/fs/drives
```

返回系统可用驱动器列表。

---

## 工作区管理

### 切换工作区

```
POST /api/workspace
```

**请求体：**
```json
{
  "action": "switch",
  "root": "{新工作区路径}",
  "folders": ["{工作区子文件夹列表}"]
}
```

---

## 设置

### 获取设置

```
GET /api/settings
```

返回当前所有设置项。

### 保存设置

```
PUT /api/settings
```

**请求体：** 完整的设置 JSON 对象。

---

## 系统信息

```
GET /api/system/info
```

返回主机名、操作系统、Go 版本、当前目录和工作区信息。

### 执行命令

```
POST /api/system/exec
```

**请求体：**
```json
{
  "command": "要执行的命令",
  "cwd": "{工作目录}"
}
```

---

## 对话管理

### 获取对话列表

```
GET /api/conversations?workspace={工作区路径}
```

### 获取对话消息

```
GET /api/conversations/{convId}/messages?limit={数量}&before={偏移}
```

### 发送消息

```
POST /api/chat/send
```

**请求体：**
```json
{
  "convId": "会话ID",
  "message": "用户消息内容",
  "autonomous": false
}
```

### 停止 AI 响应

```
POST /api/chat/stop?convId={会话ID}
```

### 用户回答

```
POST /api/chat/answer
```

**请求体：**
```json
{
  "convId": "会话ID",
  "answer": "用户回答内容"
}
```

### 审批写工具

```
POST /api/chat/approve
```

**请求体：**
```json
{
  "convId": "会话ID",
  "approved": true
}
```

### 运行时反馈

```
POST /api/chat/feedback
```

**请求体：**
```json
{
  "convId": "会话ID",
  "content": "反馈内容"
}
```

---

## 模型管理

### 获取模型列表

```
GET /api/models
```

---

## Git 操作

### 查看状态

```
GET /api/git/status?path={仓库路径}
```

### Diff 对比

```
GET /api/git/diff?path={仓库路径}&file={文件路径}
```

### 暂存文件

```
POST /api/git/add
```

**请求体：**
```json
{
  "path": "仓库路径",
  "files": ["文件列表"]
}
```

### 提交

```
POST /api/git/commit
```

**请求体：**
```json
{
  "path": "仓库路径",
  "message": "提交信息",
  "all": true
}
```

### 查看日志

```
GET /api/git/log?path={仓库路径}&count={数量}
```

### 分支管理

```
POST /api/git/branch
```

**请求体：**
```json
{
  "path": "仓库路径",
  "action": "create/delete/switch",
  "name": "分支名"
}
```

### 推送/拉取

```
POST /api/git/push
POST /api/git/pull
```

### 远程管理

```
GET /api/git/remote?path={仓库路径}
```

---

## Skills / MCP

### 获取 Skills 列表

```
GET /api/skills/list
```

### 读取 Skill

```
GET /api/skills/read?name={技能名}
```

### 删除 Skill

```
POST /api/skills/delete
```

### 获取 MCP 列表

```
GET /api/mcp/list?level={层级}
```

### 保存 MCP 配置

```
POST /api/mcp/save
```

---

## Token 统计

```
GET /api/tokens/stats?workspace={工作区}
```

---

## Debug 日志

```
GET /api/debug/logs
GET /api/debug/logs/{日志ID}
```

---

## WebSocket

```
ws://host/ws
```

用于实时推送 AI 对话事件和状态更新。所有对话会话的事件通过单一 WebSocket 连接推送，支持断线自动重连。
