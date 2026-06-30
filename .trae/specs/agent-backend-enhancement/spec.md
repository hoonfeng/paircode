# Agent 后端增强规划

## 目标

参考 `f:\syproject\ref\adk-go`（Google ADK Go）的设计模式，**不引入其依赖**，对 gou-ide 的 agent 后端做四项增强：

1. **工具层增强**：解决文件编辑"找不到原文本"、glob 不支持 `**`、命中率低等问题。
2. **MCP 客户端替换**：用官方 `github.com/modelcontextprotocol/go-sdk` 替换自研手写 JSON-RPC。
3. **Skills 机制增强**：参考 ADK 三级渐进披露模式。
4. **多 agent 编排**：参考 ADK 的 transfer/delegation，**不做上下文隔离**，保证缓存前缀命中。

**范围**：仅 agent 后端，不动 UI 层。

---

## 设计原则

1. **不引入 adk-go**：go-sdk 是唯一新增依赖（独立轻量，不绑 Gemini）。
2. **不破坏现有特色**：保留 compress（上下文压缩）、circling（绕圈检测）、Approve（审批钩子）、token 用量统计、task_manager。
3. **子 agent 共享上下文**：子 agent 调用 LLM 时，messages 前缀 = 父 agent 完整历史 + 委托工具的 FunctionResponse。**不做 IsolationScope 过滤**（与 ADK task/single_turn 的关键差异）。这保证缓存前缀稳定，子 agent 不重复读取工作。
4. **渐进式改造**：每个阶段可独立验证、独立提交。

---

## 阶段一：工具层增强（修最痛问题）

### 1.1 修复文件编辑工具（edit_file / multi_edit）

**问题根因**（见 findings.md §5.1）：CRLF vs LF、无规范化、要求逐字节复述原文。

**改造方案**：
- **规范化匹配**：匹配前对文件内容和 old_string 都做 CRLF→LF 归一化（`\r\n` → `\n`，孤立 `\r` → `\n`）。命中后按**原始字节**替换（保持文件原有换行风格）。
- **空白容忍匹配**：精确匹配失败时，进入 fallback：
  - 按行 split，逐行 trim 后比较，找到唯一对应行段。
  - 行尾空白、行首缩进差异（tab vs 空格）容忍。
- **行号定位模式**：新增 `line_start`/`line_end` 可选参数，LLM 可选行号定位（更可靠）或 old_string 定位。
- **失败诊断**：匹配失败时返回当前文件内容的前后片段（带行号），帮助 LLM 下一轮纠正。
- **参考 ADK**：BeforeToolCallback 风格的预校验 + panic recovery。

**实现位置**：`agent/tools.go:157-254`，新增 `agent/edit_matcher.go` 封装匹配逻辑。

### 1.2 修复 glob 工具（支持 `**`）

**问题**（findings.md §5.2）：`filepath.Match`/`path.Match` 不支持 `**`。

**改造方案**：
- 新增 `agent/glob.go`：用 `filepath.WalkDir` 遍历 + 自实现 globstar 匹配（`**` 匹配任意层级目录）。
- 合并 `search_files` 与 `find_files_by_pattern` 为统一实现（消除行为不一致）。
- 修正 description：明确 `**` 语义。

**实现位置**：`agent/findfiles.go`、`agent/search.go`、新增 `agent/glob.go`。

### 1.3 工具 description 优化

- 精简 description，去除实现细节（如"基于 LSP 的 documentSymbol 能力"）。
- 明确参数格式、返回结构、何时调用。
- 参考 ADK ProcessRequest 模式：对动态资源类工具（如 skills），在 LLM 请求前注入可用资源列表 + 调用时机说明。

### 1.4 工具注册链路修复

- 审计 `RegisterDefaultTools`（tools.go:386-391），补齐 `registerFindSymbolTool`/`registerFileSymbolTools`/`registerFindFilesByPatternTool`/`registerGoBuildTools`/`registerRunTestTools`/`registerCodeFixTool`/`registerBridgeTools` 的调用。
- 确保所有工具都注册进 Registry，LLM 能看到。

### 1.5 工具回调钩子（参考 ADK BeforeToolCallback/AfterToolCallback）

在 `Registry` 上增加可选回调链：
- `BeforeTool`：预校验参数、审计日志、文件快照备份（编辑前）、拒绝危险操作。
- `AfterTool`：记录 diff、触发索引更新。
- `OnToolError`：找不到原文本时自动重试（带规范化）、工具超时反射。

**实现位置**：`agent/tools.go`（Registry 扩展）、`agent/loop.go`（调用点）。

### 1.6（可选）强类型工具定义辅助

参考 ADK functiontool，提供泛型辅助函数 `DefineTool[TArgs, TResults any]`，用 struct + json tag 自动生成 schema + 入口校验。**不强制改造现有工具**，作为新工具的推荐写法。

**实现位置**：新增 `agent/typed_tool.go`。

---

## 阶段二：MCP 客户端替换（官方 go-sdk）

### 2.1 引入依赖
`go get github.com/modelcontextprotocol/go-sdk`（独立轻量，不绑 Gemini）。

### 2.2 重写 mcp.go
- 用 `mcp.CommandTransport{Command: exec.Command(...)}` 替换手写 JSON-RPC（mcp.go:29-148）。
- 用 `mcp.Client` + `mcp.ClientSession` 替换 `mcpClient`。
- **保留** gou-ide 的 `MCPServerConfig`、`Registry.Register`、`Tool` 结构、`RequiresApproval` 语义。

### 2.3 自动重连（参考 connectionRefresher）
- 封装 `mcpClient` 持有 `*exec.Cmd` 保活。
- `withRetry`：call 失败时 Ping 检测，死了则 Close 旧 session + 重启进程 + 重建 session，重试一次。
- 可刷新错误：`mcp.ErrConnectionClosed`/`mcp.ErrSessionMissing`/`io.EOF`。

### 2.4 分页 + 结构化输出
- `ListTools` 按 cursor 分页（client.go:78-110 模式）。
- `CallTool` 解析 `StructuredContent`（优先）+ `TextContent`（兜底）+ `IsError`。

### 2.5 超时 + 并发
- 所有 call 传 `context.Context`，默认 30s 超时（可配）。
- 保留单 mutex 串行化（MCP 协议要求），但单次 call 有 context 超时兜底。

### 2.6 细粒度 HITL
- 把 `RequiresApproval` 从 `bool` 改为 `ApprovalProvider func(toolName string, args map[string]any) bool`（参考 ADK `RequireConfirmationProvider`）。
- 默认 provider：对 `delete`/`write`/`exec` 类工具返回 true，只读返回 false。
- 保留原 `RequiresApproval bool` 字段兼容（bool 视为静态 provider）。

**实现位置**：重写 `agent/mcp.go`、`agent/mcp_test.go`。

---

## 阶段三：Skills 机制增强（三级渐进披露）

### 3.1 统一存储格式
- 统一为**目录式**：`.pair/skills/<name>/SKILL.md` + `references/` + `assets/` + `scripts/`。
- 修复 `ui/skills/skills.go`（扁平 .md）与 `resource_manager.go`（目录式）的冲突：废弃扁平格式，迁移现有扁平 .md 到目录式（兼容读旧格式一个版本）。
- 目录名必须 == frontmatter.name（参考 ADK 校验）。

### 3.2 完整 frontmatter 解析
- 用 YAML 解析 frontmatter（`name`/`description` 必填，`allowed-tools`/`globs`/`activation` 可选）。
- 修复 `ui/skills/skills.go:84-109` 只解析首行 `# 标题` 的问题。
- 消费 `allowed-tools`（限制 skill 可调用的工具白名单）和 `globs`（按文件类型激活）。

### 3.3 三级加载模式（参考 ADK skilltoolset）
- **L1 常驻 system prompt**：所有启用 skill 的 frontmatter（name+description）拼进 system prompt（轻量）。
- **L2 load_skill 工具**：按需加载指定 skill 的完整 SKILL.md 正文。
- **L3 load_skill_resource 工具**：按需加载 `references/`/`assets/`/`scripts/` 子文件，单文件上限 10MiB。
- 替换现有 `skill_read` 为 `load_skill`，新增 `load_skill_resource`。

### 3.4 资源沙箱
- `load_skill_resource` 强制校验路径前缀必须是 `references/`/`assets/`/`scripts/`，防路径穿越（参考 `filesystem_source.go:115-134`）。

### 3.5 内置 skill 加载
- 修复 `config/skills/` 下内置 skill 未被加载进 prompt 的问题（findings.md §5.3）。
- 加载链路：`config/skills/`（内置）+ `.pair/skills/`（用户）合并，按 `settings.json` 的 `SkillEnabledOverrides` 过滤。

**实现位置**：`ui/skills/skills.go`、`agent/resource_manager.go`、`agenttools/tools.go`、新增 `agent/skill_loader.go`（核心加载逻辑，供 UI 和 agent 共用）。

---

## 阶段四：多 agent 编排（参考 ADK，不引入，不做上下文隔离）

### 4.1 设计核心：共享 []Message，不做隔离

**与 ADK 的关键差异**：ADK 的 task/single_turn 模式用 IsolationScope 过滤子 agent 可见事件；本项目**显式不做隔离**，子 agent 看到父 agent 的完整 []Message 历史 + 委托工具的 FunctionResponse。

**缓存命中保证**：子 agent 调用 LLM 时，messages 前缀 = 父历史（前 N 条）+ 委托 FR。前 N 条与父 agent 调用时的前缀完全一致 → **prompt cache 命中**，节省 token，子 agent 不需重复读取工作文件。

### 4.2 agent 树结构
```go
type SubAgent struct {
    Name        string
    Description string  // LLM 据此决定是否委托
    System      string  // 子 agent 专属系统提示（可空，空则继承父）
    Tools       []string // 工具白名单（工具名；空=继承父全部工具）
    MaxIter     int
}

type AgentTree struct {
    Root     *SubAgent
    Children map[string]*SubAgent // name → sub-agent
    Parent   map[string]string    // child name → parent name
}
```

### 4.3 委托工具（参考 ADK task_agent_tool / single_turn_tool）

新增两个工具，注册到父 agent 的 Registry：

**delegate_task**（多轮委托，参考 ADK TaskAgentTool）：
- 参数：`agent_name`（目标子 agent）、`task`（任务描述）。
- 行为：创建子 Loop，传入父的 []Message 副本 + 合成 user 消息（task 内容），运行至子 agent 输出 `[FINAL]` 或 finish_task。
- **关键**：子 Loop 的 Provider 调用用父的完整 []Message 作前缀 → 缓存命中。
- 子 Loop 产出的 assistant/tool 消息**追加到父的 []Message**（不隔离），父协调器下一轮能看到子 agent 的工作。
- 子 Loop 复用父的 Registry（或按 Tools 白名单裁剪）。

**delegate_single_turn**（单轮委托，参考 ADK SingleTurnTool）：
- 参数：`agent_name`、`input`。
- 行为：同步执行子 agent 一轮，结果作为 FunctionResponse 返回。
- 同样共享父 []Message 前缀。

**finish_task**（参考 ADK FinishTaskTool）：
- 子 agent 调用，表示任务完成，参数 `result`。
- 触发子 Loop 退出。

### 4.4 transfer_to_agent（控制权转移，参考 ADK ModeChat）

**delegate_task** 是父主动委托子；**transfer_to_agent** 是当前 agent 把控制权完全交给另一个 agent（通常是兄弟或父）。
- 参数：`agent_name`。
- 行为：当前 Loop 退出，目标 agent 接管，继续用同一 []Message。
- 用于"这个问题该让 X agent 处理"的场景。

### 4.5 共享状态
- 全局 `State map[string]any`，所有 agent 可读写（参考 ADK session.State）。
- 通过 `Loop` 持有，子 Loop 继承父 Loop 的 State 引用。
- 用于跨 agent 传递中间结果（避免塞进 messages 撑爆上下文）。

### 4.6 缓存前缀稳定性保证（核心约束）

- 子 agent 调用 LLM 时，messages 构造顺序：`[父 system（或子 system 覆盖）] + 父 []Message[0:N] + 合成 user 消息（委托 task）`。
- 父 []Message[0:N] 与父 agent 上一次调用的前缀**逐字节一致** → 命中 provider 的 prompt cache。
- **禁止**在子 agent 调用前插入额外 system 消息或重排历史（会破坏前缀）。
- 子 agent 的 system 提示若非空，作为**追加** instruction 而非替换（保持前缀稳定）。

### 4.7 编排示例
```
用户: "重构 auth 模块"
父 agent (coordinator):
  - 调 delegate_task(agent="planner", task="分析 auth 模块并给出重构计划")
    子 agent (planner) 看到完整父历史 + task，缓存命中，产出计划
    子 agent 调 finish_task(result="计划...")
    计划作为 FR 回到父 []Message
  - 父调 delegate_task(agent="coder", task="按计划重构 auth.go")
    子 agent (coder) 看到完整父历史（含 planner 的计划）+ task，缓存命中
  - 父调 delegate_task(agent="reviewer", task="审查重构结果")
```

**实现位置**：新增 `agent/subagent.go`（SubAgent/AgentTree）、`agent/delegate.go`（委托工具）、`agent/loop.go` 扩展（支持子 Loop 嵌套 + finish_task 退出信号）。

---

## 阶段五：测试与验证

### 5.1 文件编辑工具测试
- CRLF 文件编辑（old_string 用 LF）
- 空白/缩进差异匹配
- 行号定位模式
- 多行 old_string 匹配
- 匹配失败诊断信息

### 5.2 glob 测试
- `**/*.go` 递归匹配
- `src/**/auth*` 部分递归
- 大小写敏感（Windows）

### 5.3 MCP 测试
- go-sdk 替换后主流程
- 断连自动重连
- 超时不死锁
- 分页拉取
- StructuredContent 解析

### 5.4 skills 测试
- 目录式 SKILL.md 加载
- frontmatter 解析
- L1/L2/L3 三级加载
- 资源沙箱（路径穿越拒绝）
- 旧扁平格式兼容

### 5.5 多 agent 编排测试
- delegate_task 多轮委托
- delegate_single_turn 单轮委托
- finish_task 退出
- transfer_to_agent
- **缓存前缀命中验证**：子 agent 调用时 messages 前缀与父一致（可用 mock provider 断言 messages）

---

## 风险与缓解

| 风险 | 缓解 |
|---|---|
| 文件编辑规范化破坏文件原有换行风格 | 命中后按原始字节替换，仅匹配阶段归一化 |
| go-sdk 版本与 gou-ide go 1.26 兼容性 | go-sdk 要求 go 1.23+，兼容；先 `go get` 验证 |
| 子 agent 共享上下文导致上下文膨胀 | 复用现有 compress.go 压缩机制；子 agent 产出的 tool 消息可标记为中段可压缩 |
| 委托工具增加 LLM 选择复杂度 | description 明确"何时委托"；子 agent description 简洁；参考 ADK "Do NOT call in parallel" 提示 |
| skills 格式迁移破坏现有用户 skill | 兼容读旧扁平格式一个版本；提供迁移脚本 |

---

## 不做（明确排除）

- 不引入 adk-go 依赖（go-sdk 除外）。
- 不做 ADK 的 IsolationScope 上下文隔离（用户明确要求共享）。
- 不做 ADK 的 session/artifact/memory 服务抽象（gou-ide 用 []Message + .pair/tasks/ 已够）。
- 不做 ADK 的 server/REST/A2A 部署（桌面 IDE 不需要）。
- 不做 ADK 的 otel telemetry（gou-ide 有自己的 perf 日志）。
- 不动 UI 层（skills UI、chat UI 等本次不改）。
- 不强制改造所有现有工具为强类型（仅提供辅助函数，新工具推荐用）。
