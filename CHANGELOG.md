# Changelog

## v1.0.4 (2026-07-17)

### Bug Fixes
- **消息持久化核心修复**：`PersistNewMessages` 中 `persistedCount` 与 `histNonSystemCount` 口径不一致（总行数 vs 非 System 消息数），导致含 tool_call 的 assistant 消息在工具执行前未能真正写入磁盘。阻塞工具（如 ask_user）的前端提问框不会出现，表现为前端无响应。
- **git.go JSON tag 重复**：修复 `HandleGitLog` 中 `entry` 结构体四个字段共享同一 JSON tag 的 vet 警告。

### Features
- **对话状态 API**：`GET /api/conversations/{id}` 现在返回 `running`/`stopped`/`startedAt` 等 agent 运行状态，前端加载对话时可感知当前是否在执行中。
- **任务列表 API**：`GET /api/tasks?convId=xxx` 从 stub（返回对话列表）改为返回真实的任务列表和 `summary` 摘要，前端可展示子任务进度面板。
- **执行计划 API**：`GET /api/taskplan` 从 stub 改为返回 `.pair/tasks/*.md` 规划文档列表。
- **TaskManager**：新增 `ListByConvID(convID)` 方法，支持按对话过滤任务。
- **SessionManager**：新增 `GetStatus()` 方法，返回会话完整运行状态快照。

### 影响模块
- `internal/agent/message_store.go` — PersistNewMessages 计数器修复
- `internal/agent/session_manager.go` — 新增 GetStatus 方法
- `internal/agent/task_manager.go` — 新增 ListByConvID 方法
- `internal/server/handler/stub.go` — 3 个 API 从 stub 改为真实实现
- `internal/server/handler/git.go` — JSON tag 修复

### 验证
- go build (CGO_ENABLED=0/1) 编译通过
- go vet 无警告
- TestMessageStore 全部 14 个测试 PASS
