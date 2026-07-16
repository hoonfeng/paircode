// 独立 Agent 示例——展示如何将 agent 包作为基座嵌入任意 Go 应用。
// 运行: go run ./examples/standalone-agent/main.go
//
//go:build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/hoonfeng/paircode/cmd/companion/agent"
)

func main() {
	// 使用临时工作区
	workDir, _ := os.Getwd()
	tmpDir := filepath.Join(workDir, ".agent-example")
	os.MkdirAll(tmpDir, 0o755)
	defer os.RemoveAll(tmpDir)

	// 写入一个示例文件
	helloPath := filepath.Join(tmpDir, "hello.txt")
	os.WriteFile(helloPath, []byte("Hello Agent World!"), 0o644)

	// 1. 创建工具注册表并注册默认工具
	reg := agent.NewRegistry()
	agent.RegisterDefaultTools(reg, tmpDir)

	// 2. 使用 MockProvider（无网络，脚本化响应）
	//    实际使用时替换为 OpenAIProvider：
	//    provider = &agent.OpenAIProvider{
	//        BaseURL: "https://api.openai.com/v1",
	//        APIKey:  "sk-xxx",
	//        Model:   "gpt-4o",
	//    }
	provider := &agent.MockProvider{
		Responses: []agent.Message{
			{
				Role: agent.RoleAssistant,
				ToolCalls: []agent.ToolCall{{
					ID:   "c1",
					Type: "function",
					Function: agent.FunctionCall{
						Name:      "read_file",
						Arguments: `{"path":"hello.txt"}`,
					},
				}},
			},
			{
				Role:    agent.RoleAssistant,
				Content: "文件内容读到了：Hello Agent World!\n任务完成。",
			},
		},
	}

	// 3. 创建 Loop
	loop := &agent.Loop{
		Provider:      provider,
		Registry:      reg,
		System:        "你是文件助手。用工具完成任务后回复结果。",
		MaxIterations: 10,
		OnEvent: func(e agent.Event) {
			switch e.Type {
			case agent.EventToolCall:
				fmt.Printf("🔧 调用工具: %s(%s)\n", e.Tool, e.Args)
			case agent.EventToolResult:
				fmt.Printf("📦 工具结果: %s\n", e.Content)
			case agent.EventContent:
				fmt.Printf("💬 %s\n", e.Content)
			case agent.EventDone:
				fmt.Printf("✅ 完成: %s\n", e.Content)
			}
		},
	}

	// 4. 运行
	fmt.Println("🚀 启动 Agent...")
	msgs, err := loop.Run(context.Background(), "读取 hello.txt 的内容", nil)
	if err != nil {
		log.Fatalf("Agent 运行失败: %v", err)
	}

	fmt.Printf("\n📝 共 %d 条消息，最后一条:\n", len(msgs))
	if len(msgs) > 0 {
		last := msgs[len(msgs)-1]
		fmt.Printf("  [%s] %s\n", last.Role, last.Content)
	}
}
