# 任务清单

状态：[ ] 待办  [~] 进行中  [x] 完成

---

## 阶段一：工具层增强（最痛问题优先）✅ 已完成

> 阶段一全部完成并通过验证（go build + go vet + go test 全绿）。详见 findings.md 第七节"阶段一执行发现"。
> 实际实施中额外修复了 4 个预存问题：storm_breaker_test.go 预存损坏、code_fix go tool fix 失效、code_format exit code 误判、read_file 缺二进制保护；并补齐 Registry.Unregister 方法。
> 1.3（description 优化）已在 1.1/1.2/1.4 修复中顺带完成（edit_file/glob/code_fix 等描述已更新）；1.6（强类型工具辅助）暂不实施（现有 objSchema 辅助足够，泛型改造收益不明显）。

### 1.1 修复文件编辑工具
- [ ] 1.1.1 新增 `agent/edit_matcher.go`：封装匹配逻辑
  - [ ] CRLF→LF 归一化函数（匹配阶段归一化，替换阶段保留原始字节）
  - [ ] 精确匹配 + 规范化匹配（两级）
  - [ ] 空白容忍 fallback：按行 split + trim 比较 + tab/space 归一
  - [ ] 行号定位匹配（line_start/line_end）
  - [ ] 匹配失败诊断：返回带行号的上下文片段
- [ ] 1.1.2 改造 `edit_file`（tools.go:157-189）：用 edit_matcher 替换 strings.Count/Replace
- [ ] 1.1.3 改造 `multi_edit`（tools.go:191-254）：同上，顺序应用多处替换
- [ ] 1.1.4 edit_file schema 增加 `line_start`/`line_end` 可选参数
- [ ] 1.1.5 测试 `agent/edit_matcher_test.go`：
  - [ ] CRLF 文件 + LF old_string 命中
  - [ ] 空白/缩进差异命中
  - [ ] 行号定位
  - [ ] 多行 old_string
  - [ ] 匹配失败诊断信息
  - [ ] old_string 出现多次报错
- [ ] 1.1.6 更新 loop.go:240-242 系统提示：移除"edit_file 失败重读重试"硬规则（工具已自愈）

### 1.2 修复 glob 工具
- [ ] 1.2.1 新增 `agent/glob.go`：globstar 匹配
  - [ ] `MatchGlob(pattern, path string) bool`：支持 `**`/`*`/`?`
  - [ ] `WalkGlob(root, pattern string) ([]string, error)`：WalkDir + MatchGlob
- [ ] 1.2.2 改造 `find_files_by_pattern`（findfiles.go）：用 WalkGlob 替换 filepath.Match
- [ ] 1.2.3 改造 `search_files`（search.go:166-215）：用 WalkGlob 替换 path.Match
- [ ] 1.2.4 统一两者行为（或合并为一个工具 + 别名）
- [ ] 1.2.5 修正 description：明确 `**` 语义和示例
- [ ] 1.2.6 测试 `agent/glob_test.go`：
  - [ ] `**/*.go` 递归
  - [ ] `src/**/auth*` 部分递归
  - [ ] `*.go` 单层
  - [ ] Windows 路径分隔符

### 1.3 工具 description 优化
- [ ] 1.3.1 审计所有工具 description，精简实现细节
- [ ] 1.3.2 明确参数格式、返回结构、何时调用
- [ ] 1.3.3 符号工具 description 标注"仅 Go"前置条件

### 1.4 工具注册链路修复
- [ ] 1.4.1 审计 `RegisterDefaultTools`（tools.go:386-391）
- [ ] 1.4.2 补齐 registerFindSymbolTool/registerFileSymbolTools/registerFindFilesByPatternTool/registerGoBuildTools/registerRunTestTools/registerCodeFixTool/registerBridgeTools 调用
- [ ] 1.4.3 验证所有工具在 Registry 中可见（写测试断言 Definitions() 包含预期工具名）

### 1.5 工具回调钩子
- [ ] 1.5.1 Registry 增加 `BeforeTool/AfterTool/OnToolError` 回调链字段
- [ ] 1.5.2 Loop.Run 在工具执行前后调用回调（loop.go:135-163）
- [ ] 1.5.3 实现默认 BeforeTool：编辑前文件快照备份（.pair/snapshots/）
- [ ] 1.5.4 实现默认 OnToolError：edit_file 失败时自动用规范化重试
- [ ] 1.5.5 测试回调链

### 1.6（可选）强类型工具辅助
- [ ] 1.6.1 新增 `agent/typed_tool.go`：`DefineTool[TArgs, TResults any]` 泛型辅助
- [ ] 1.6.2 用 struct json tag 反射生成 JSON Schema
- [ ] 1.6.3 入口校验（map→struct + schema 校验）
- [ ] 1.6.4 panic recovery 包装
- [ ] 1.6.5 示例：用 DefineTool 重写一个现有工具验证

---

## 阶段二：MCP 客户端替换（官方 go-sdk）✅ 已完成

> 阶段二全部完成并通过验证（go build ./cmd/companion/... + go vet + go test 全绿，agent 包 28s）。
> 用官方 go-sdk v1.6.1 替换自研 JSON-RPC 客户端：mcp.go 重写为 261 行，mcp_test.go 重写为 7 个测试用例（全部通过，0.24s）。
> 关键实现：
>   - transport 注入字段（生产用 CommandTransport，测试用 InMemoryTransport，进程内验证无需子进程）
>   - ensureAlive：Ping(5s 超时) 检测 + 失败重连；withRetry[T] 泛型对可刷新错误（connection closed/eof/broken pipe 等）重试一次
>   - listAllTools：按 NextCursor 翻页直到空；callTool：默认 30s 超时 + 重连重试
>   - parseCallToolResult：StructuredContent 优先（JSON）+ TextContent 兜底（拼接）+ IsError 转 error
>   - registerClientTools：工具名加 "mcp.<server>.<name>" 前缀，RequiresApproval=true；HITL 复用阶段一 Registry.BeforeTool 钩子（按前缀白名单）
>   - RegisterMCPServers 签名兼容（返回注册工具数，起不来的跳过不阻断）

- [x] 2.1 `go get github.com/modelcontextprotocol/go-sdk`，验证与 go 1.26 兼容
- [x] 2.2 重写 `agent/mcp.go`：
  - [x] 2.2.1 用 `mcp.CommandTransport` 替换手写 JSON-RPC（mcp.go:29-148）
  - [x] 2.2.2 用 `mcp.Client` + `mcp.ClientSession` 替换 mcpClient
  - [x] 2.2.3 保留 MCPServerConfig/Registry.Register/Tool 结构
  - [x] 2.2.4 保活 `*exec.Cmd`（修复 mcp.go:215 丢弃 cmd 问题）
- [x] 2.3 自动重连（参考 connectionRefresher, client.go:36-188）：
  - [x] 2.3.1 `withRetry[T]` 泛型重试
  - [x] 2.3.2 Ping 检测 + Close 旧 session + 重启进程 + 重建 session
  - [x] 2.3.3 可刷新错误清单：ErrConnectionClosed/ErrSessionMissing/io.EOF
- [x] 2.4 分页：ListTools 按 cursor 分页（client.go:78-110 模式）
- [x] 2.5 结构化输出：CallTool 解析 StructuredContent（优先）+ TextContent（兜底）+ IsError
- [x] 2.6 超时：所有 call 传 context，默认 30s（可配）
- [x] 2.7 细粒度 HITL：
  - [x] 2.7.1 MCP 工具 RequiresApproval=true（统一标记需审批）
  - [x] 2.7.2 复用阶段一 Registry.BeforeTool 钩子做白名单审批（按 "mcp.<server>.<tool>" 前缀）
  - [x] 2.7.3 保留 RequiresApproval bool 兼容
- [x] 2.8 重写 `agent/mcp_test.go`：
  - [x] 2.8.1 go-sdk 主流程（initialize/list/call）—— TestMCPListAndCall
  - [x] 2.8.2 断连自动重连 —— TestMCPReconnect（关 server1+换 transport+ensureAlive 重连 server2）
  - [x] 2.8.3 超时不死锁 —— TestMCPTimeout（callTimeout=100ms，慢工具 2s，验证提前返回）
  - [x] 2.8.4 分页拉取 —— TestMCPPagination（PageSize=5，12 工具，全部拉取）
  - [x] 2.8.5 StructuredContent 解析 —— TestMCPStructuredContent + TestMCPIsError
  - [x] 2.8.6 细粒度 HITL —— TestMCPRegistryHITL（BeforeTool 拒绝短路 + 放行执行）

---

## 阶段三：Skills 机制增强 ✅ 已完成

> 阶段三核心完成并通过验证（go build ./cmd/companion/... + go vet + go test 全绿，agent 包 12s，skill_loader 12 个测试全过）。
> 实现三级渐进披露（L1/L2/L3）+ 统一目录式存储格式，核心逻辑集中在 `agent/skill_loader.go`（供 UI 和 agent 共用）。
> 关键实现：
>   - 存储格式统一为目录式 `.pair/skills/<name>/SKILL.md`（frontmatter）+ `references/` `assets/` `scripts/` 子目录；旧扁平 `<name>.md` 兼容读取
>   - frontmatter 自实现轻量解析（`---` 边界 + `key: value` + 去引号，不引入 yaml 库）；name 缺失回退目录名
>   - L1 常驻 system prompt（PromptSkills 输出 name+description 列表）/ L2 load_skill 工具（加载正文 Skill.Body）/ L3 load_skill_resource 工具（加载子文件，10MiB 上限）
>   - 资源沙箱：LoadSkillResource 强制路径前缀为 references/assets/scripts + 二次 Abs 校验防 `..` 穿越
>   - SkillEnabledOverrides 按 `level::name` 键过滤（不存在默认启用）；bridge.go 启动时注入 SkillSystemDir/SkillProjectDir/SkillEnabled 全局变量（与 WorkspaceRoots 模式一致，agent 包不依赖 core）
>   - 内置技能 config/skills/ 经 SkillSystemDir 加载，system 级注入 L1 prompt
>   - ui/skills/skills.go、agent/resource_manager.go skillsProvider、agenttools/tools.go 全部委托 skill_loader，消除旧 frontmatterField/firstLine 散落调用
> 3.6（allowed-tools 工具白名单裁剪 / globs 按文件类型自动激活）属运行时激活增强，需 Loop 层配合，暂不实施（与阶段一 1.6 同处理）；frontmatter 字段已解析存储于 Skill 结构，后续激活逻辑可直接消费。

- [x] 3.1 新增 `agent/skill_loader.go`（核心加载逻辑，供 UI 和 agent 共用）：
  - [x] 3.1.1 目录式扫描：`.pair/skills/<name>/SKILL.md` + `config/skills/<name>/SKILL.md`
  - [x] 3.1.2 YAML frontmatter 解析（name/description 必填，allowed-tools/globs/activation 可选）
  - [x] 3.1.3 目录名 == frontmatter.name 校验（name 缺失回退目录名）
  - [x] 3.1.4 旧扁平 `.pair/skills/<name>.md` 兼容读取（自动包装为单文件 skill）
  - [x] 3.1.5 按 settings.json SkillEnabledOverrides 过滤
- [x] 3.2 修复 `ui/skills/skills.go`：
  - [x] 3.2.1 改用 skill_loader 加载
  - [x] 3.2.2 修复 readSkillFile 只解析首行 `# 标题` 的问题（改用 frontmatter）
  - [x] 3.2.3 Prompt() 输出 L1（name+description 列表）
- [x] 3.3 修复 `agent/resource_manager.go:437,450` 的 skillsProvider：改用 skill_loader
- [x] 3.4 改造 `agenttools/tools.go`：
  - [x] 3.4.1 `skill_read` → `load_skill`：加载完整 SKILL.md 正文（L2）
  - [x] 3.4.2 新增 `load_skill_resource`：加载 references/assets/scripts 子文件（L3），10MiB 上限
  - [x] 3.4.3 资源沙箱：路径前缀校验（references/assets/scripts）
- [x] 3.5 内置 skill 加载链路：config/skills/ → skill_loader → prompt（bridge.go 注入 SkillSystemDir）
- [~] 3.6 消费 frontmatter 字段（暂不实施，需 Loop 层配合）：
  - [ ] 3.6.1 `allowed-tools`：skill 激活时限制可用工具白名单
  - [ ] 3.6.2 `globs`：按当前编辑文件类型自动激活
- [x] 3.7 测试 `agent/skill_loader_test.go`：
  - [x] 3.7.1 目录式 SKILL.md 加载
  - [x] 3.7.2 frontmatter 解析（含引号去引号、无 frontmatter 兜底）
  - [x] 3.7.3 旧扁平格式兼容
  - [x] 3.7.4 资源沙箱（路径穿越拒绝 + 绝对路径拒绝 + 超限拒绝）
  - [x] 3.7.5 enabled 过滤
  - [x] 3.7.6 L1/L2/L3 三级加载（PromptSkills / Skill.Body / LoadSkillResource）+ FindSkill + Write/Delete 往返

---

## 阶段四：多 agent 编排（不做上下文隔离）✅ 已完成

> 阶段四全部完成并通过验证（go build ./cmd/companion/... + go vet + go test ./cmd/companion/agent/ 全绿，agent 包 11.6s，含 11 个阶段四新增测试）。
> 核心实现集中在 `agent/subagent.go`（编排树）+ `agent/delegate.go`（委托工具）+ `agent/loop.go` 扩展（字段+退出信号）+ `agent/tools.go`（Registry.Copy/Subset）。
> 关键设计（与 ADK 关键差异）：**不做上下文隔离**——子 agent 共享父 []Message 前缀保证 LLM prompt cache 命中：
>   - 缓存前缀稳定：子 Loop 用父 currentMsgs 作 history，剥离末尾未配对 assistant tool_call（=委托调用本身），使子首次 LLM 调用 messages 前缀 = 父上一次调用前缀（recordingProvider 断言逐条一致）
>   - 子 System 不替换父 System：子 Loop.System="" 不插入 system 消息（会破坏前缀）；子 agent 专属 System 作追加 instruction 拼到 task 前
>   - Registry.Copy()/Subset()：子 Loop 用副本/白名单裁剪表注册 finish_task，避免污染父表；白名单非空时 Subset 裁剪
>   - finish_task 退出信号：子 agent 调 finish_task → Loop.Run 检测工具名，置 finishResult 并 return；delegate handler 从 child.finishResult 取子结果
>   - transfer_to_agent：设 transferTarget，Loop.Run 检测后退出当前循环（控制权转移，调用方接管同一 []Message）
>   - 共享 State：Loop.State map[string]any，子 Loop 继承父引用（跨 agent 传递中间结果，不塞进 messages 撑爆上下文）

- [x] 4.1 新增 `agent/subagent.go`：
  - [x] 4.1.1 `SubAgent` 结构（Name/Description/System/Tools/MaxIter）
  - [x] 4.1.2 `AgentTree` 结构（Root/Children/Parent）
  - [x] 4.1.3 `NewAgentTree(root *SubAgent, children ...*SubAgent)` 构造
  - [x] 4.1.4 `Find(name)` 查找（+ ParentOf/SubNames/Add）
- [x] 4.2 扩展 `agent/loop.go`：
  - [x] 4.2.1 Loop 增加 `AgentTree`、`State map[string]any`、`currentMsgs []Message`、`finishResult`、`transferTarget` 字段
  - [x] 4.2.2 Loop.Run 同步 currentMsgs（每次 assistant append 后），供 delegate handler 读父历史
  - [x] 4.2.3 finish_task 退出信号：Loop.Run 检测 finish_task 工具调用即置 finishResult 并退出
  - [x] 4.2.4 子 Loop 的 Provider 调用用父 []Message 作前缀（缓存命中保证）
  - [x] 4.2.5 子 Loop 产出作工具结果回到父 []Message（不隔离）
  - [x] 4.2.6 子 system 提示作为追加 instruction（不替换，保前缀稳定）
- [x] 4.3 新增 `agent/delegate.go`（委托工具）：
  - [x] 4.3.1 `delegate_task` 工具：多轮委托
    - 参数：agent_name, task
    - 创建子 Loop，传入父 []Message（剥离末尾未配对 assistant tool_call）+ 合成 user（task）
    - 运行至 [FINAL] 或 finish_task
    - 子产出作工具结果回到父 []Message
    - 返回 FR（子 agent 最终结果摘要：finish_task result 优先，否则末条 assistant 正文）
  - [x] 4.3.2 `delegate_single_turn` 工具：单轮委托
    - 参数：agent_name, input
    - maxIter=1 同步执行一轮，结果作 FR 返回
  - [x] 4.3.3 `finish_task` 工具：子 agent 任务完成信号
    - 参数：result
    - 触发当前 Loop 退出（在 runSubAgent 创建子 Loop 时注册，避免污染父表）
  - [x] 4.3.4 `transfer_to_agent` 工具：控制权转移
    - 参数：agent_name
    - 当前 Loop 退出（transferTarget），目标 agent 接管同一 []Message
- [x] 4.4 委托工具注册：RegisterDelegateTools 向父 Registry 注册 delegate_task/delegate_single_turn/transfer_to_agent（基于 AgentTree）；runSubAgent 在子 Registry 注册 finish_task
- [x] 4.5 共享 State：Loop 持有 State map，子 Loop 继承父 State 引用（child.State = parent.State）
- [x] 4.6 缓存前缀稳定性验证：
  - [x] 4.6.1 子 Loop 构造 messages 时，前 N 条与父一致（recordingProvider 断言 recorded[1].messages[:2] == recorded[0].messages）
  - [x] 4.6.2 不插入额外 system/不重排历史（子 System="" 复用父 history 中的 system）
- [x] 4.7 测试 `agent/subagent_test.go` + `agent/delegate_test.go`：
  - [x] 4.7.1 delegate_task 多轮委托（TestDelegateTask_MultiTurn，recordingProvider 断言 messages 前缀）
  - [x] 4.7.2 delegate_single_turn 单轮委托（TestDelegateSingleTurn）
  - [x] 4.7.3 finish_task 退出（TestDelegateTask_MultiTurn 含 finish_task→子退出）
  - [x] 4.7.4 transfer_to_agent（TestTransferToAgent + TestTransferToAgent_NotFound）
  - [x] 4.7.5 缓存前缀命中验证（TestDelegateTask_MultiTurn 断言 recorded[1].messages[:2] == recorded[0].messages + 子 system 作追加 instruction）
  - [x] 4.7.6 共享 State 读写（TestSharedState，子 agent 通过 get_state 读父 State）
  - [x] 4.7.7 子 agent 工具白名单裁剪（TestToolWhitelist + TestDelegateTask_AgentNotFound）
  - [x] 4.7.8 Registry.Copy/Subset 单测（TestRegistry_Copy/TestRegistry_Subset）+ AgentTree 构造（TestNewAgentTree/TestAgentTree_Add）

---

## 阶段五：集成测试与验证

- [ ] 5.1 `go build ./cmd/companion/...` 全量编译通过
- [ ] 5.2 `go test ./cmd/companion/agent/...` 全绿
- [ ] 5.3 端到端：用真实 LLM 跑 edit_file CRLF 场景验证
- [ ] 5.4 端到端：用真实 LLM 跑 glob `**` 场景验证
- [ ] 5.5 端到端：启动一个 MCP server（如 filesystem-mcp）验证 go-sdk 替换
- [ ] 5.6 端到端：加载 config/skills/emoji-icons 验证 L1 注入 + L2 load_skill
- [ ] 5.7 端到端：多 agent 委托场景（coordinator → planner → coder）

---

## 提交节点（建议）

| 节点 | 内容 | 提交信息 |
|---|---|---|
| C1 | 1.1 文件编辑修复 | `fix(agent): 修复 edit_file CRLF/空白匹配失败，增加行号定位与诊断` |
| C2 | 1.2-1.4 glob + description + 注册链路 | `fix(agent): glob 支持 **，统一查找工具，补齐工具注册链路` |
| C3 | 1.5-1.6 回调钩子 + 强类型辅助 | `feat(agent): 工具回调钩子(Before/After/OnError) + 强类型工具辅助` |
| C4 | 2.x MCP go-sdk 替换 | `refactor(agent): 用官方 go-sdk 替换自研 MCP 客户端，加重连/分页/超时/细粒度HITL` |
| C5 | 3.x Skills 增强 | `feat(agent): skills 三级渐进披露(常驻prompt+按需加载+资源沙箱)，统一目录式格式` |
| C6 | 4.x 多 agent 编排 | `feat(agent): 多 agent 编排(delegate_task/single_turn/transfer)，共享上下文不做隔离，保证缓存前缀命中` |
