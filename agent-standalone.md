# Agent 基座：独立 AI Agent 引擎

`cmd/companion/agent` 是一个**纯 Go 标准库**的 AI Agent 引擎（TAOR 循环），
零 GUI 依赖，可被任意 Go 应用嵌入使用。

## 30 秒快速集成

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/hoonfeng/paircode/cmd/companion/agent"
)

func main() {
	// 1. 创建工具注册表
	reg := agent.NewRegistry()
	agent.RegisterDefaultTools(reg, "/path/to/workspace")

	// 2. 配置 LLM Provider
	provider := &agent.OpenAIProvider{
		BaseURL: "https://api.openai.com/v1",
		APIKey:  "sk-xxx",
		Model:   "gpt-4o",
	}

	// 3. 创建并运行 Loop
	loop := &agent.Loop{
		Provider:      provider,
		Registry:      reg,
		System:        "你是一个编程助手",
		MaxIterations: 30,
		OnEvent: func(e agent.Event) {
			fmt.Printf("[%s] %s\n", e.Type, e.Content)
		},
	}

	msgs, err := loop.Run(context.Background(), "读取 hello.go 并修复里面的 BUG", nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("完成。共 %d 条消息\n", len(msgs))
}
```

## 核心架构

```
┌──────────────────────────────────────────────────┐
│  你的应用（Web / CLI / 桌面 / 移动端）            │
│  ┌────────────────────────────────────────────┐   │
│  │  Loop (TAOR 循环)                          │   │
│  │  think → act → observe → repeat           │   │
│  │  ┌────────┐ ┌──────────┐ ┌───────────┐    │   │
│  │  │Provider│ │ Registry │ │ Compressor│    │   │
│  │  │(LLM)   │ │(工具表)   │ │(压缩器)   │    │   │
│  │  └────────┘ └──────────┘ └───────────┘    │   │
│  └────────────────────────────────────────────┘   │
│  ┌────────────────────────────────────────────┐   │
│  │  SessionManager (会话管理)                  │   │
│  │  并行会话 / 事件广播 / 持久化               │   │
│  └────────────────────────────────────────────┘   │
│  ┌────────────────────────────────────────────┐   │
│  │  外围扩展（可选）                           │   │
│  │  MCP 服务器 / Skills 技能 / Lua 工具       │   │
│  │  记忆系统 / 项目知识库 / 代码图谱           │   │
│  └────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────┘
```

## 关键接口

### Provider（LLM 适配）
```go
type Provider interface {
    Chat(ctx context.Context, messages []Message, tools []ToolDefinition, onChunk func(Chunk)) (Message, error)
    Name() string
}
```
内置实现：`OpenAIProvider`（兼容 DeepSeek/Qwen/Moonshot 等）、`MockProvider`（测试用）。

### Registry（工具注册）
```go
reg := agent.NewRegistry()
reg.Register(&agent.Tool{
    Name:        "my_tool",
    Description: "我的工具",
    Parameters:  objSchema(props{"arg": strProp("参数")}, "arg"),
    Handler:     func(ctx context.Context, args map[string]any) (string, error) {
        return "结果", nil
    },
})
```

### Loop（核心编排器）
```go
loop := &agent.Loop{
    Provider:      provider,
    Registry:      reg,
    System:        "系统提示词",
    MaxIterations: 30,
    OnEvent:       func(e Event) { /* 流式推送 */ },
    Approve:       func(ctx context.Context, tc ToolCall) (bool, string) { return true, "" },
}
msgs, err := loop.Run(ctx, "用户输入", nil)
```

### 内置工具（RegisterDefaultTools）
- 文件操作：`read_file` / `edit_file` / `write_file` / `move_file` / `delete_file` / `list_files`
- 搜索：`search_content` / `search_files`
- Git：`git_status` / `git_diff` / `git_log` / `git_commit` / `git_push`
- Web：`web_fetch` / `web_search` / `web_debug`
- 代码：`codegraph_*` 全套
- 图像：`image_analyze` / `image_ocr`
- 调试：`debug_*` 全套
- 办公：`csv_*` / `word_*` / `read_pdf` / `read_xlsx` / `write_xlsx`
- BUG 检测：`bug_detect` / `bug_fix`
- 更多…

## 自主模式

```go
loopOpts := &agent.LoopOpts{
    Provider:      provider,
    Registry:      reg,
    System:        systemPrompt,
    Autonomous:    true,   // 启用自主模式
    PlanProvider:  planProvider, // 规划模型
    AutoReview:    true,   // AI 审核
    WorkspaceRoot: "/path/to/project",
}
sessionMgr := agent.NewSessionManager()
sessionMgr.Start(ctx, "conv-1", "修复登录页面 BUG", loopOpts)
```

## 最少依赖

Agent 核心仅依赖：
- Go 标准库（`net/http`、`encoding/json`、`os` 等）
- `pkg/codegraph`（代码知识图谱，纯 Go）
- `pkg/memory`（记忆系统，纯 Go）
- `pkg/summary`（摘要压缩，纯 Go）

零 CGO、零 GUI 框架、零外部 LLM SDK。
