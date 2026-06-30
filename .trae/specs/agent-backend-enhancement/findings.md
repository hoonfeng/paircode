# 研究发现：adk-go 参考 + gou-ide 现状诊断

研究来源：`f:\syproject\ref\adk-go`（Google ADK Go v2）、`f:\syproject\gou-ide`（companion）。

---

## 一、adk-go 多 agent 编排：上下文共享机制（核心参考）

### 模型：单 session + 事件标记过滤
- 所有 agent 共享**同一个 session**（同一 sessionID、同一 events 列表、同一 state）。见 `agent/agent.go:358-396` 的 `invocationContext` 与 `WithICDelta`——`InvocationContextDelta` 只能覆盖 `UserContent/Branch/IsolationScope/Agent/Context`，**不能覆盖 session/artifacts/memory**。
- 上下文隔离是通过 `Branch` 和 `IsolationScope` 字段在构建 LLM prompt 时做**逻辑过滤**，非物理分离。见 `session/session.go:94-143` 的 Event 结构。

### 三种委托模式（`agent/llmagent/llmagent.go:350-360`）
| 模式 | 行为 | 缓存前缀 |
|---|---|---|
| ModeChat（transfer_to_agent）| 原样传递完整历史，子 agent 看到全部 events | **稳定，命中** |
| ModeTask（task_agent_tool）| 隔离到 `IsolationScope=fc.ID`，子 agent 只看自己 scope + 合成首条 user 消息 | 子 agent 前缀短，但父协调器前缀稳定 |
| ModeSingleTurn | 强制 `IncludeContents=none`，只看当前轮 | 无历史 |

### transfer_to_agent 实际执行（`internal/llminternal/base_flow.go:658-684`）
```go
nextAgent := f.agentToRun(ctx, ev.Actions.TransferToAgent)
nextStream = nextAgent.Run(ctx)  // 同一 ctx（同 session）
for ev, err := range nextStream { yield(ev, err) }  // 转发到同一 iterator
```
**关键**：转移后的 agent 用同一 ctx（同 session、同 invocationID），事件追加到同一 session。

### chat 协调器的 messages 构建（`internal/llminternal/contents_processor.go:76-203`）
- `buildContentsDefault`：协调器 `isolationScope=""`，只看 `IsolationScope=""` 的事件 → **完整历史 → 缓存前缀稳定**。
- task/single_turn 子 agent：通过 `buildTaskInputUserContent`（行 651-693）合成首条消息，父历史不直接出现。

### 对本项目的指导意义
**用户要求"不做上下文隔离"** → 应采用 **ModeChat 风格**（transfer + 完整历史共享），子 agent 看到完整 messages 前缀，缓存命中最高。这是与 ADK task/single_turn 模式的关键差异——ADK 那两个模式会隔离，本项目显式不做。

---

## 二、adk-go 工具实现（参考）

### Tool 接口分层
- 公开 `tool.Tool`（`tool/tool.go:38-46`）：仅 `Name()/Description()/IsLongRunning()`。
- 内部 `FunctionTool`（`internal/toolinternal/tool.go:28-32`）：`Declaration() *genai.FunctionDeclaration` + `Run(ctx, args any)(map[string]any, error)`。

### functiontool 泛型工厂（`tool/functiontool/function.go:72-120`）
```go
type Func[TArgs, TResults any] func(agent.Context, TArgs) (TResults, error)
```
- 强类型参数，`json` tag 反射自动生成 JSON Schema（`resolvedSchema[T]`，行 267-277）。
- `ConvertToWithJSONSchema` 在 Run 入口校验（`internal/typeutil/convert.go:47`），脏参数进不了业务逻辑。
- 自带 defer recover（行 187-191），单工具 panic 不崩 agent。

### 工具回调钩子（`agent/llmagent/llmagent.go:380-405`）
- `BeforeToolCallback`：Run 前执行，返回非 nil 可跳过实际工具，可原地改 args。
- `AfterToolCallback`：Run 后执行，可替换结果。
- `OnToolErrorCallback`：出错时执行，可替换错误。

### ProcessRequest 动态注入 instruction（`tool/loadartifactstool/load_artifacts_tool.go:130-152`）
工具自己在请求阶段往 system instruction 塞上下文（列出可用资源 + 调用时机说明）——这是提高 LLM 命中率的核心模式。

### 关键：ADK Go 没有文件编辑工具参考实现
全仓库 grep `old_string|new_string|edit_file|apply_patch|str_replace` **零匹配**。最接近的是 `loadartifactstool`（只读 artifact）。文件操作走 `ctx.Artifacts()` 抽象，非裸文件系统。

---

## 三、adk-go skills 机制（混合加载模式）

### SKILL.md 格式（`tool/skilltoolset/skill/frontmatter.go:37-44`）
YAML frontmatter（`name`+`description` 必填，另有 `license/compatibility/metadata/allowed-tools`）+ Markdown 正文。严格校验未知字段，name 必须与目录名一致。

### 三级渐进披露（核心设计）
1. **L1 常驻 system prompt**：frontmatter（name+description）经 `ProcessRequest` 注入（`toolset.go:107-116`），轻量。
2. **L2 按需加载指令**：`load_skill` 工具加载完整 SKILL.md 正文（`internal/skilltool/load_skill.go:49-85`）。
3. **L3 按需加载资源**：`load_skill_resource` 工具加载 `references/`/`assets/`/`scripts/` 子文件，单文件上限 10MiB（`load_skill_resource.go:27`）。

### 资源沙箱（`filesystem_source.go:115-134`）
`LoadResource` 强制校验路径前缀必须是 `references/`/`assets/`/`scripts/`，防路径穿越。

### 目录结构
```
<skill-name>/         # 目录名 == frontmatter.name
  SKILL.md            # 必需
  references/         # 可选：附加文档
  assets/             # 可选：模板/资源
  scripts/            # 可选：可执行脚本
```

---

## 四、adk-go mcptoolset（基于官方 go-sdk）

### 关键：go-sdk 是独立轻量依赖
`github.com/modelcontextprotocol/go-sdk v1.4.1` **不依赖 genai/adk**（见 `client.go:17-28` import），可与 gou-ide 的 OpenAI 兼容协议栈共存。

### mcptoolset 优势（vs gou-ide 自研）
| 维度 | gou-ide mcp.go | adk-go mcptoolset |
|---|---|---|
| 协议 | 手写 JSON-RPC，硬编码 `2024-11-05` | 官方 go-sdk，自动协商 |
| 传输 | 仅 stdio | stdio/SSE/in-memory，接口抽象 |
| 重连 | 无，`*exec.Cmd` 被丢弃（mcp.go:215） | `connectionRefresher` 自动重连+Ping 去重（client.go:36-188） |
| 分页 | 一次拉完 | cursor 分页（client.go:78-110） |
| 超时 | call/callTool 无超时，hang 则死锁 | 由 context 控制 |
| 输出 | 只取 text | 支持 StructuredContent/OutputSchema/IsError |
| HITL | 一刀切全 RequiresApproval | 静态 + 动态 `RequireConfirmationProvider`（按工具/参数） |

### 替换路径（推荐档位 A+B）
- **A 最小替换**：手写 JSON-RPC 层 → go-sdk 的 `mcp.Client`+`mcp.CommandTransport`，保留 gou-ide 的 Registry/Tool 抽象。
- **B 增强**：加自动重连、分页、结构化输出、细粒度 HITL。
- **不做 C**（全面对齐 adk-go Toolset 接口）：会牵动 gou-ide 上下文压缩/审批钩子等特色机制。

---

## 五、gou-ide 现状问题诊断

### 5.1 文件编辑工具"找不到原文本"根因（最痛）
实现位置：`agent/tools.go:157-189`（edit_file）、`tools.go:191-254`（multi_edit）。

```go
// tools.go:175,183
switch strings.Count(string(data), old) {  // 字节级精确匹配
case 0: return "", fmt.Errorf("未找到 old_string，无法定位编辑点")
case 1: // ok
default: return "", fmt.Errorf("old_string 出现多次，不唯一")
}
out := strings.Replace(string(data), old, neu, 1)
```

**根因（逐条）**：
1. **CRLF vs LF**：`os.ReadFile` 读原始字节含 `\r\n`，LLM 产出的 old_string 只含 `\n`（tokenizer 规范化掉 `\r`）→ `strings.Count` 返回 0。
2. **无任何规范化**：不 trim、不折叠空白、不对齐缩进。LLM 漏一个空格/tab 即不匹配。
3. **要求逐字节复述原文**：违反 LLM tokenizer 特性。
4. **无行号/patch/fuzzy 兜底**：唯一匹配失败即报错，让 LLM 重读重试（loop.go:240-242 系统提示承认此病）。
5. **测试未覆盖 CRLF**：tools_test.go 用 `"hello WORLD"` 这种无换行简单字符串。

### 5.2 工具命中率低的原因
1. **glob 不支持 `**`**：`findfiles.go:23`/`search.go:78` description 承诺 `**` 递归，但实现用 `filepath.Match`/`path.Match`（只支持单层 `*`）→ LLM 传 `**/*.go` 匹配不到任何文件。
2. **两个 glob 工具行为不一致**：search_files（path.Match）vs find_files_by_pattern（filepath.Match），LLM 易选错。
3. **符号工具仅支持 Go**：find_symbol/list_exported_symbols 等在非 Go 文件上报错。
4. **LSP 工具每次重启**：get_file_symbols/find_symbol_usages 每次 newLSPClient，gopls 冷启动 30s+，无连接复用。
5. **搜索结果硬截断**：search_content max_results=200 + capOutput 16000，LLM 看到截断结果。
6. **description 偏长含实现细节**：稀释关键信息（pattern 格式、返回结构）。

### 5.3 skills 现状问题
- **存储路径不一致 bug**：`ui/skills/skills.go:54-56` 读扁平 `.pair/skills/*.md`；`resource_manager.go:437,450` 读目录式 `.pair/skills/<name>/SKILL.md`。两套割裂，Agent 写的扁平 .md 被 resource_manager 漏掉。
- **frontmatter 字段未被消费**：`config/skills/*/SKILL.md` 有 `activation/globs` 字段，但 `ui/skills/skills.go:84-109` 的 readSkillFile 只解析首行 `# 标题`，不解析 frontmatter。
- 只有 L1（prompt 注入名+描述）+ L2（skill_read 读全文），**无 L3 资源加载**。

### 5.4 MCP 自研版致命缺陷
- **call/callTool 无超时**（mcp.go:52-79,122-148）：`for { ReadBytes('\n') }` 无限阻塞，服务器 hang 则 agent 永久死锁，且 mu 锁被持有，该服务器所有后续调用全部死锁。
- **无重连，进程无保活**：mcp.go:215 用 `_` 接收 `*exec.Cmd`，进程退出后 io pipe 断裂，下次调用 EOF 报错。
- **无动态工具发现**：listTools 只在启动时调一次。
- **单 mutex 串行化**：同服务器工具无法并发。
- **非 JSON 行处理脆弱**：mcp.go:71 continue 可能死循环。

### 5.5 工具注册链路缺失嫌疑
`bridge.go:145-156` 调 `RegisterDefaultTools`，但 `tools.go:386-391` 的 RegisterDefaultTools 内部只调了 `registerSearchTools/registerGitTools/registerWebTools/registerPlanTool/registerShellTools/registerMemoryTools`。而 `registerFindSymbolTool`/`registerFileSymbolTools`/`registerFindFilesByPatternTool`/`registerGoBuildTools`/`registerRunTestTools`/`registerCodeFixTool`/`registerBridgeTools` **未在 RegisterDefaultTools 内调用**——部分工具可能根本未注册进 Registry，LLM 看不到。

---

## 六、关键约束（用户明确要求）

1. **不引入 adk-go 依赖**，只做参考。
2. **子 agent 不做上下文隔离**：保证缓存命中前缀稳定，节省 token，防止子 agent 重复读取工作。子 agent 共享父 agent 的 []Message 历史。
3. **暂不做 UI 层**，只做 agent 后端。
4. **替换 MCP 客户端**：用官方 go-sdk。
5. **增强 skills 机制**：参考 ADK 混合加载模式。
6. **工具增强**：解决文件编辑失败、命中率低问题。

---

## 七、阶段一执行发现（实施过程中暴露的预存问题与修复）

### 7.1 storm_breaker_test.go 预存损坏（阻断全包测试编译）
- **现象**：`go test ./cmd/companion/agent/...` 编译失败，报 `undefined: RetryInfo/sendWithRetry/AuthError/WithRetryNotify/IsConnReset/retryableStatus/extractSystemPrompt/backoffDelay/maxBackoff`。
- **根因**：storm_breaker_test.go 引用了一组 Provider 重试机制符号，但这些符号在 agent 包**从未实现**（grep 确认 llm.go 只有 parseSSE，cache_shape.go 有 CaptureShape/CompareShape/sessionCache）。该测试文件像是从某含 Provider 重试的版本复制来，但实现从未落地。
- **处理**：重写 storm_breaker_test.go，移除引用未定义符号的测试（TestSendWithRetry_*/TestIsConnReset/TestRetryableStatus/TestRetryInfoJSON/TestExtractSystemPrompt/TestBackoffDelay），保留已实现符号的测试（Storm Breaker/RepeatSuccessGuard/EvidenceLedger/TruncateToolOutput/FinalReadiness/CaptureShape/parseSSE/canonicalToolArgs/sessionCache）。Provider 重试机制属阶段二/四范围，实现后应新建独立测试文件。

### 7.2 code_fix 工具失效（go tool fix -go flag 不支持）
- **现象**：code_fix 所有测试失败，报 `flag provided but not defined: -go`。
- **根因**：现代 Go 的 `go tool fix` 已变为 analysis 工具（`fix.exe unit.cfg`），不再支持 `-go`/`-diff`/直接传文件路径。fixer.go 的 `go tool fix -go <ver> <path>` 完全无法工作。
- **修复**：改用 `gofmt -s`（简化语法：合并冗余类型、简化复合字面量）替代。预览模式 `gofmt -s -d`，apply 模式 `gofmt -s -l`(忽略 exit code 收集) + `gofmt -s -w`(写入)。移除死代码 detectGoVersion 函数和冗余 LookPath("go") 检查。
- **语义变化**：code_fix 从"修复旧 API 模式"变为"简化语法"，与 code_format（gofmt 格式化）区分在于 `-s` 标志。

### 7.3 code_format 预览模式 exit code 误判
- **现象**：gofmt -d 有差异时部分版本 exit 1，formatter.go 误判为错误，测试 TestCodeFormatPreview 失败。
- **修复**：预览模式仅在"无 diff 且 err 非 nil"才报错；apply 模式拆为 `gofmt -l`(忽略 exit code 收集需改文件) + `gofmt -w`(写入)。

### 7.4 Registry.Unregister 方法缺失 + 测试本身漏调
- **现象**：TestRegistryUnregister 失败（Register 后 Get 仍返回 ok=true）。
- **根因双重**：① Registry 无 Unregister 方法（Lua 热重载需要）；② 测试本身漏调 `reg.Unregister("x")`，直接 Register 后就断言 Get 返回 false。
- **修复**：tools.go 新增 `Registry.Unregister(name)`（删 tools map + order 切片，并发安全）；luatool_test.go 补上 `reg.Unregister("x")` 调用。

### 7.5 read_file 缺二进制保护
- **现象**：TestReadFileBinaryGuard 失败——read_file 读含 NULL 字节的二进制文件不报错，把字节流灌进上下文。
- **修复**：read_file handler 在 os.ReadFile 后加 `strings.IndexByte(string(data), 0) >= 0` 检测，命中则报错引导 inspect_binary。

### 7.6 TestRegistryDefinitions 硬编码计数过时
- **现象**：原断言 `len(defs) != 45`，但注释的 45 组成（19+git7+memory5+project_info6+binary2+RE6）既不含 debug/web/plan/shell/task/find_*/go_*/code_* 等已注册工具，补齐 4 组注册后总数远超 45。
- **修复**：改为下限断言 `>= 50` + 关键工具名断言（覆盖各注册组），避免增减工具就破坏测试。

### 7.7 RegisterDefaultTools 注册链路补齐（最终状态）
RegisterDefaultTools 现调用全部 19 个 register* 函数：核心 8（read/write/edit/multi_edit/list/run_command/move/delete）+ search/git/web/plan/shell/memory + find_files_by_pattern/find_symbol/file_symbols/task/go_build/run_test/go_run/code_fix/code_format + project_info(6)/binary(2)/binary_re(6)/debug(11)。evolution_tools/tool_stats 由 bridge.go 单独注册（management tools 范畴）。

### 7.8 阶段一验证结果
- `go build ./cmd/companion/...` 通过
- `go vet ./cmd/companion/agent/...` 无警告
- `go test ./cmd/companion/agent/...` 全绿（含 edit_matcher 20 测试、glob 2 测试、registry_hooks 6 测试、code_fix 8 测试、code_format 6 测试、binary 3 测试、debug 11 测试、project_info 测试等）
