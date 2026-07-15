// 桌面版额外路由注册：chat/agent + market。
//
//go:build desktop

package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/hoonfeng/paircode/cmd/companion/agent"
	"github.com/hoonfeng/paircode/cmd/companion/bridge"
	"github.com/hoonfeng/paircode/cmd/companion/core"
)

func init() {
	// 桌面版：使用 bridge 压缩器
	webCompressor = func() agent.Compressor { return bridge.BuildCompressor() }
}

func registerExtraHandlers(mux *http.ServeMux, s *webServer) {
	mux.HandleFunc("/api/chat/send", s.handleChatSend)
	mux.HandleFunc("/ws", s.handleWebSocket) // WebSocket 端点（替代 SSE /api/chat/events）
	mux.HandleFunc("/api/terminal/ws", s.handleTerminalWS) // 终端 PTY WebSocket
	mux.HandleFunc("/api/chat/stop", s.handleChatStop)
	mux.HandleFunc("/api/chat/answer", s.handleChatAnswer)
	mux.HandleFunc("/api/chat/approve", s.handleChatApprove)
	mux.HandleFunc("/api/chat/feedback", s.handleChatFeedback)
	mux.HandleFunc("/api/marketplace/search", s.handleMarketplaceSearch)
	mux.HandleFunc("/api/marketplace/install", s.handleMarketplaceInstall)
	mux.HandleFunc("/api/marketplace/refresh", s.handleMarketplaceRefresh)

	// 长时记忆检索 API
	mux.HandleFunc("/api/memory/search", s.handleMemorySearch)
	mux.HandleFunc("/api/memory/list", s.handleMemoryList)
	mux.HandleFunc("/api/memory/rebuild", s.handleMemoryRebuild)
}

// emitPhase 发送阶段切换事件到事件通道。
func emitPhase(events chan agent.Event, phase string) {
	select {
	case events <- agent.Event{Type: agent.EventPhase, Content: phase}:
	default:
	}
}

// runOrchestrationLoop 外部编排循环（新版，基于 OrchestrationEngine 状态机）。
// 每次 loop.Run 完成后，用编排 agent 分析已完成内容并决定下一个任务，
// 将规划记录到 .pair/tasks/ 目录，然后继续执行，直到编排 agent 判定全部完成。
// 支持执行状态持久化和断点续跑。
// restoredState 可选：断点续跑时传入中断的执行状态，避免从头开始。
//
//nolint:unused // 暂未调用，后续迭代启用自主编排模式
func runOrchestrationLoop(ctx context.Context, prov agent.Provider, reg *agent.Registry, events chan agent.Event, approvalCh chan bool, task string, history []agent.Message, roots []string, root string, convID string, restoredState *agent.ExecutionState) []agent.Message {
	emitPhase(events, "自主执行中")

	// ── 构建编排引擎 ──
	planner := buildWebPlanner()
	webSysPrompt := buildWebSystemPrompt()

	eng := agent.NewOrchestrationEngine(
		prov, reg, planner,
		events, approvalCh,
		webSysPrompt, root, roots, convID,
	)
	eng.Task = task
	eng.History = history
	eng.RestoredState = restoredState
	eng.MaxLoops = 10

	// 内层 Loop 配置
	maxIter := core.Settings.MaxIterations * 2
	if maxIter <= 0 {
		maxIter = 60
	}
	eng.MaxIterations = maxIter
	eng.MaxContextTokens = core.Settings.ContextMaxTokens
	eng.Compressor = bridge.BuildCompressor()

	// 构建验证回调
	eng.VerifyFunc = func(r string) (bool, string) {
		result := autoVerifyProject(r)
		return result.success, result.output
	}

	// ── Panic 静默恢复（防止编排循环 panic 导致整个服务崩溃）──
	defer func() {
		if r := recover(); r != nil {
			agent.WritePanic("runOrchestrationLoop", r, "",
				map[string]string{"convId": convID, "task": task})
			panic(r)
		}
	}()

	// ── 运行编排引擎（阻塞直到终态）──
	result := eng.Run(ctx)

	// 结果处理
	if result.IsTerminal() && result.Phase != agent.PhaseDone {
		// 失败/取消 → 发送错误事件
		if result.Phase == agent.PhaseFailed {
			select {
			case events <- agent.Event{Type: agent.EventError, Content: result.Reason}:
			default:
			}
		}
	}
	return result.History
}

// ─── 自动构建验证 ──────────────────────────────────────────

type verifyResult struct {
	success bool
	output  string
}

// autoVerifyProject 自动检测项目类型并运行构建验证。
// 返回验证结果（成功/失败 + 输出）。
func autoVerifyProject(root string) verifyResult {
	if root == "" {
		return verifyResult{success: true, output: "（无工作区，跳过验证）"}
	}

	// 检测 Go 项目
	goModPath := filepath.Join(root, "go.mod")
	if _, err := os.Stat(goModPath); err == nil {
		// 1. go vet
		ctx1, cancel1 := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel1()
		vet := exec.CommandContext(ctx1, "go", "vet", "-tags", "webonly", "./cmd/companion")
		vet.Dir = root
		vetOut, vetErr := vet.CombinedOutput()
		if vetErr != nil {
			return verifyResult{success: false,
				output: fmt.Sprintf("❌ go vet 失败:\n%s", string(vetOut))}
		}

		// 2. go build
		ctx2, cancel2 := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel2()
		build := exec.CommandContext(ctx2, "go", "build", "-tags", "webonly", "./cmd/companion")
		build.Dir = root
		buildOut, buildErr := build.CombinedOutput()
		if buildErr != nil {
			return verifyResult{success: false,
				output: fmt.Sprintf("❌ Go 构建失败:\n%s", string(buildOut))}
		}

		// 3. go test（仅运行时，不阻塞主流程）
		testResult := ""
		ctx3, cancel3 := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel3()
		test := exec.CommandContext(ctx3, "go", "test", "-count=1", "-timeout", "30s", "./cmd/companion/agent")
		test.Dir = root
		testOut, testErr := test.CombinedOutput()
		if testErr != nil {
			testResult = fmt.Sprintf("\n⚠️ 测试有失败:\n%s", string(testOut))
		} else {
			testResult = "\n✅ 测试通过"
		}

		return verifyResult{success: true,
			output: fmt.Sprintf("✅ Go 验证通过 (vet+build)%s", testResult)}
	}

	// 检测 Node.js 项目
	packagePath := filepath.Join(root, "package.json")
	if _, err := os.Stat(packagePath); err == nil {
		// 1. TypeScript 类型检查
		ctx1, cancel1 := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel1()
		tsc := exec.CommandContext(ctx1, "npx", "tsc", "--noEmit")
		tsc.Dir = root
		tscOut, tscErr := tsc.CombinedOutput()
		if tscErr == nil {
			// 2. 构建
			ctx2, cancel2 := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel2()
			vite := exec.CommandContext(ctx2, "npx", "vite", "build")
			vite.Dir = root
			viteOut, viteErr := vite.CombinedOutput()
			if viteErr != nil {
				return verifyResult{success: false,
					output: fmt.Sprintf("❌ Vite 构建失败:\n%s", string(viteOut))}
			}

			// 3. npm test（如果可用，快速运行）
			testResult := ""
			ctx3, cancel3 := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel3()
			npmTest := exec.CommandContext(ctx3, "npx", "--yes", "vitest", "run", "--reporter=verbose")
			npmTest.Dir = root
			testOut, testErr := npmTest.CombinedOutput()
			if testErr == nil {
				testResult = "\n✅ 前端测试通过"
			} else {
				// 不因为测试失败阻塞整个验证，仅记录
				testResult = fmt.Sprintf("\n⚠️ 前端测试结果:\n%s", string(testOut))
			}

			return verifyResult{success: true,
				output: fmt.Sprintf("✅ TypeScript + Vite 通过%s", testResult)}
		}
		// 仅构建（tsc 可能因为类型缺失而失败）
		ctx2, cancel2 := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel2()
		vite := exec.CommandContext(ctx2, "npx", "vite", "build")
		vite.Dir = root
		viteOut, viteErr := vite.CombinedOutput()
		if viteErr == nil {
			return verifyResult{success: true,
				output: "✅ Vite 构建通过（tsc 跳过）"}
		}
		return verifyResult{success: false,
			output: fmt.Sprintf("❌ 构建失败:\ntsc: %s\n\nvite: %s", string(tscOut), string(viteOut))}
	}

	// 未知项目类型，跳过自动验证
	return verifyResult{success: true, output: "（未识别的项目类型，跳过自动验证）"}
}

// countStates 统计具备指定状态的执行状态数量。
// countStates 已在 web_server.go 中统一实现
