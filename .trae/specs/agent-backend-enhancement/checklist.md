# 验收检查清单

## 阶段一：工具层增强

### 文件编辑工具
- [ ] edit_file 能在 CRLF 文件上用 LF 的 old_string 命中
- [ ] edit_file 能容忍行尾空白、tab/space 缩进差异
- [ ] edit_file 支持 line_start/line_end 行号定位
- [ ] edit_file 匹配失败时返回带行号的上下文诊断（非裸"未找到"）
- [ ] multi_edit 顺序应用多处替换，每处都走新匹配逻辑
- [ ] 命中后按原始字节替换，保持文件原有换行风格（CRLF 文件替换后仍 CRLF）
- [ ] edit_matcher_test.go 全绿，覆盖 CRLF/空白/行号/多行/多次出现

### glob 工具
- [ ] `**/*.go` 能递归匹配子目录
- [ ] `src/**/auth*` 能部分递归匹配
- [ ] `*.go` 只匹配单层
- [ ] Windows 路径分隔符正确（`\` 和 `/` 都能处理）
- [ ] find_files_by_pattern 与 search_files 行为一致（或已合并）
- [ ] glob_test.go 全绿

### description 与注册
- [ ] 所有工具 description 无实现细节冗余
- [ ] 符号工具 description 标注"仅 Go"前置
- [ ] RegisterDefaultTools 调用所有 register* 函数
- [ ] 测试断言 Definitions() 包含全部预期工具名

### 回调钩子
- [ ] Registry 支持 BeforeTool/AfterTool/OnToolError 回调
- [ ] Loop.Run 在工具执行前后正确调用回调
- [ ] 编辑前自动文件快照备份（.pair/snapshots/）
- [ ] edit_file 失败时 OnToolError 自动规范化重试

---

## 阶段二：MCP 客户端替换

- [ ] `github.com/modelcontextprotocol/go-sdk` 引入成功，go build 通过
- [ ] mcp.go 用 mcp.CommandTransport + mcp.Client + mcp.ClientSession
- [ ] *exec.Cmd 被保活（不再用 `_` 丢弃）
- [ ] MCP server 进程退出后，下次调用自动重连成功
- [ ] call/callTool 有 context 超时（默认 30s），服务器 hang 不再死锁
- [ ] ListTools 支持 cursor 分页（工具数 > 100 时能拉全）
- [ ] CallTool 解析 StructuredContent（优先）+ TextContent（兜底）+ IsError
- [ ] 细粒度 HITL：ApprovalProvider 按工具名/参数决定是否审批
- [ ] mcp_test.go 全绿，覆盖重连/超时/分页/StructuredContent/HITL
- [ ] 现有 MCP 配置（MCPServerConfig）无需用户改动即可工作

---

## 阶段三：Skills 机制增强

- [ ] skill_loader.go 统一加载 .pair/skills/ + config/skills/
- [ ] 目录式 SKILL.md（`<name>/SKILL.md`）正确加载
- [ ] frontmatter 完整解析（name/description 必填校验）
- [ ] 旧扁平 `.pair/skills/<name>.md` 兼容读取
- [ ] 目录名 == frontmatter.name 校验
- [ ] settings.json SkillEnabledOverrides 过滤生效
- [ ] L1：所有启用 skill 的 name+description 在 system prompt 中
- [ ] L2：load_skill 工具能加载指定 skill 完整正文
- [ ] L3：load_skill_resource 工具能加载 references/assets/scripts 子文件
- [ ] 资源沙箱：load_skill_resource 拒绝路径穿越（如 `../etc/passwd`）
- [ ] 单文件 10MiB 上限生效
- [ ] config/skills/ 内置 skill 被加载进 prompt（修复原缺失）
- [ ] ui/skills/skills.go 与 resource_manager.go 用同一 skill_loader（消除割裂）
- [ ] frontmatter allowed-tools/globs 字段被消费
- [ ] skill_loader_test.go 全绿

---

## 阶段四：多 agent 编排

### 基础设施
- [ ] SubAgent/AgentTree 结构定义
- [ ] AgentTree.FindAgent(name) 正确查找
- [ ] Loop 持有 AgentTree/State/parentMsgs 字段

### 缓存前缀稳定性（核心约束）
- [ ] 子 Loop 调用 LLM 时，messages 前 N 条与父 Loop 上一次调用的前缀**逐字节一致**
- [ ] 子 system 提示作为追加 instruction（非替换）
- [ ] 不在子 agent 调用前插入额外 system 或重排历史
- [ ] 测试用 mock provider 断言 messages 前缀一致

### 委托工具
- [ ] delegate_task：多轮委托，子 agent 运行至 [FINAL]/finish_task
- [ ] delegate_task：子产出追加到父 []Message（不隔离）
- [ ] delegate_single_turn：单轮委托，结果作 FR 返回
- [ ] finish_task：子 agent 调用后触发 Loop 退出
- [ ] transfer_to_agent：控制权转移，目标 agent 接管同一 []Message
- [ ] 委托工具自动注册（基于 AgentTree）

### 共享状态
- [ ] State map 所有 agent 可读写
- [ ] 子 Loop 继承父 Loop 的 State 引用（非副本）

### 不做隔离验证
- [ ] 子 agent 能看到父 agent 的完整历史（包括之前其他子 agent 的产出）
- [ ] 无 IsolationScope/Branch 过滤逻辑

### 测试
- [ ] subagent_test.go + delegate_test.go 全绿
- [ ] 端到端：coordinator → planner → coder 委托链跑通

---

## 阶段五：集成验证

- [ ] `go build ./cmd/companion/...` 通过
- [ ] `go test ./cmd/companion/agent/...` 全绿
- [ ] 现有特色机制未破坏：
  - [ ] compress.go 上下文压缩正常
  - [ ] circling.go 绕圈检测正常
  - [ ] Approve 审批钩子正常
  - [ ] token 用量统计正常
  - [ ] task_manager 正常
- [ ] companion.exe 启动正常，现有 UI 功能不受影响（UI 层未改）

---

## 约束遵守确认

- [ ] 未引入 adk-go 依赖（仅 go-sdk）
- [ ] 子 agent 未做上下文隔离（共享 []Message）
- [ ] 未动 UI 层（skills UI/chat UI 等）
- [ ] 现有特色机制（compress/circling/Approve/token统计/task_manager）保留
- [ ] 未做 ADK session/artifact/memory/server/otel（明确排除项）
