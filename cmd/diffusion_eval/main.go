// cmd/diffusion_eval — 模型扩散生成思想（Diffusion of Thought）A/B 对比实验
//
// 用法:
//   go run ./cmd/diffusion_eval                          # 跑全部任务（baseline vs DoT）
//   go run ./cmd/diffusion_eval -arm baseline            # 只跑基线组
//   go run ./cmd/diffusion_eval -tasks 1,4,7             # 只跑指定任务
//   go run ./cmd/diffusion_eval -out report.json         # 结果输出到文件
//   go run ./cmd/diffusion_eval -max-iter 12             # 限制最大迭代数（默认 15）
//
// 配置: 复用 config/settings.json（APIKey/BaseURL/Model/MaxTokens 等）
//
// 对比指标:
//   - Token 消耗      : 总 prompt/completion/total（包装 Provider 拦截 Usage，含扩散开销）
//   - 结果质量        : Evaluator（LLM-as-Judge）四维评分 0-100
//   - 工具使用准确度  : 首步工具命中（vs 任务预定义期望工具）+ 失败工具调用率
//   - 工具使用成功率  : 1 - 失败工具调用 / 总工具调用
//   - 效率            : LLM 调用次数、迭代数、耗时
//   - 扩散成本        : DoT 组发散/收敛 token（DiffuseStats）

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/hoonfeng/paircode/internal/agent"
	"github.com/hoonfeng/paircode/internal/core"
)

// ─── usageRecorder: 包装 Provider 拦截 Usage（逐任务精确采集）────────

type usageRecorder struct {
	agent.Provider
	mu         sync.Mutex
	prompt     int
	completion int
	total      int
	calls      int
}

func (r *usageRecorder) Chat(ctx context.Context, messages []agent.Message, tools []agent.ToolDefinition, onChunk func(agent.Chunk)) (agent.Message, error) {
	r.mu.Lock()
	r.calls++
	r.mu.Unlock()
	return r.Provider.Chat(ctx, messages, tools, func(c agent.Chunk) {
		if c.Usage != nil {
			r.mu.Lock()
			r.prompt += c.Usage.PromptTokens
			r.completion += c.Usage.CompletionTokens
			if c.Usage.TotalTokens > 0 {
				r.total += c.Usage.TotalTokens
			} else {
				r.total += c.Usage.PromptTokens + c.Usage.CompletionTokens
			}
			r.mu.Unlock()
		}
		if onChunk != nil {
			onChunk(c)
		}
	})
}

func (r *usageRecorder) snapshot() (prompt, completion, total, calls int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.prompt, r.completion, r.total, r.calls
}

// ─── 实验任务定义 ─────────────────────────────────────────

type evalTask struct {
	ID           int    // 任务编号
	Name         string // 任务名
	Prompt       string // 发给 agent 的任务描述
	ExpectedTool string // 期望首步工具（前缀匹配，宽松判定）
}

var tasks = []evalTask{
	{1, "定位函数", "查找 internal/agent 包中 Loop 结构体的 Run 方法的定义文件和行号", "search"},
	{2, "代码搜索", "搜索项目里包含字符串 DiffusionThink 的所有 .go 文件，列出文件路径", "search_content"},
	{3, "读取文件", "读取 internal/agent/loop.go 文件的前 30 行内容", "read_file"},
	{4, "运行命令", "运行 go vet ./internal/agent 并报告是否有错误", "run_command"},
	{5, "文件统计", "列出 internal/agent 目录下所有以 _test.go 结尾的测试文件数量", "list_files"},
	{6, "Git 历史", "查看 gou-ide 项目最近 3 次 git 提交记录，列出提交信息", "git_log"},
	{7, "调用者分析", "找出 maybeCompact 函数被哪些函数调用，列出调用方", "search"},
	{8, "模块信息", "读取 go.mod 文件并说出 module 名称和 Go 版本", "read_file"},
	{9, "调用点搜索", "搜索代码中所有调用 getCodeGraph 函数的位置，列出文件路径", "search_content"},
	{10, "配置读取", "读取 config/settings.json 并报告 Model 字段的值", "read_file"},
}

// ─── 单 arm 单任务指标 ───────────────────────────────────

type armMetrics struct {
	TaskID         int            `json:"taskId"`
	TaskName       string         `json:"taskName"`
	Arm            string         `json:"arm"` // baseline / dot
	PromptTokens   int            `json:"promptTokens"`
	CompletionTok  int            `json:"completionTokens"`
	TotalTokens    int            `json:"totalTokens"`
	DiffuseTokens  int            `json:"diffuseTokens"` // DoT 组扩散开销
	LLMCalls       int            `json:"llmCalls"`
	Iterations     int            `json:"iterations"`
	ToolCalls      int            `json:"toolCalls"`
	FailedCalls    int            `json:"failedToolCalls"`
	FirstTool      string         `json:"firstTool"`
	ExpectedTool   string         `json:"expectedTool"`
	FirstToolHit   bool           `json:"firstToolHit"`
	DiffuseStats   *agent.DiffuseStats `json:"diffuseStats,omitempty"`
	EvalTotal      int            `json:"evalTotal"`     // 质量分 0-100
	EvalCompletion int            `json:"evalCompletion"`
	EvalCorrect    int            `json:"evalCorrectness"`
	EvalDepth      int            `json:"evalDepth"`
	EvalEfficiency int            `json:"evalEfficiency"`
	EvalTokens     int            `json:"evalTokens"` // 评估用 token（单列）
	DurationMs     int64          `json:"durationMs"`
	ToolSeq        []string       `json:"toolSeq"` // 工具调用序列（调试用）
	Err            string         `json:"err,omitempty"`
}

// ─── 主流程 ──────────────────────────────────────────────

func main() {
	armFlag := flag.String("arm", "both", "baseline / dot / both")
	tasksFlag := flag.String("tasks", "", "逗号分隔任务 ID（默认全部）")
	outFlag := flag.String("out", "", "结果 JSON 输出路径（默认 stdout）")
	maxIter := flag.Int("max-iter", 15, "最大迭代数")
	flag.Parse()

	if !core.Load() {
		fmt.Println("⚠️  config/settings.json 读取失败，使用默认配置（可能无 APIKey）")
	}
	if core.Settings.APIKey == "" || core.Settings.BaseURL == "" {
		fmt.Println("❌ 未配置 APIKey/BaseURL（config/settings.json），无法运行实验")
		os.Exit(1)
	}

	prov := &agent.OpenAIProvider{
		BaseURL:      core.Settings.BaseURL,
		APIKey:       core.Settings.APIKey,
		Model:        core.MainModel(),
		Temperature:  core.Temperature(),
		MaxTokens:    core.Settings.MaxTokens,
		ThinkingMode: core.Settings.ThinkingMode,
	}
	fmt.Printf("🧪 扩散思考 A/B 实验  model=%s maxIter=%d\n", core.MainModel(), *maxIter)

	// 选定任务
	selected := tasks
	if *tasksFlag != "" {
		ids := map[int]bool{}
		for _, s := range strings.Split(*tasksFlag, ",") {
			var id int
			fmt.Sscanf(strings.TrimSpace(s), "%d", &id)
			ids[id] = true
		}
		selected = nil
		for _, t := range tasks {
			if ids[t.ID] {
				selected = append(selected, t)
			}
		}
	}

	root, _ := os.Getwd()
	results := []armMetrics{}

	for _, t := range selected {
		for _, arm := range []string{"baseline", "dot"} {
			if *armFlag != "both" && *armFlag != arm {
				continue
			}
			fmt.Printf("▶ 任务 %d【%s】 arm=%s …\n", t.ID, t.Name, arm)
			m := runArm(context.Background(), prov, t, arm, root, *maxIter)
			results = append(results, m)
			printMetric(m)
		}
	}

	// 输出
	report := map[string]any{
		"generatedAt": time.Now().Format(time.RFC3339),
		"model":       core.MainModel(),
		"results":     results,
	}
	out := *outFlag
	if out == "" {
		b, _ := json.MarshalIndent(report, "", "  ")
		fmt.Println(string(b))
	} else {
		b, _ := json.MarshalIndent(report, "", "  ")
		os.WriteFile(out, b, 0644)
		fmt.Printf("✔ 结果已写入 %s\n", out)
	}
}

// runArm 跑一个任务的一个 arm。
func runArm(ctx context.Context, prov agent.Provider, t evalTask, arm string, root string, maxIter int) armMetrics {
	m := armMetrics{TaskID: t.ID, TaskName: t.Name, Arm: arm, ExpectedTool: t.ExpectedTool}
	start := time.Now()

	rec := &usageRecorder{Provider: prov}
	reg := agent.NewRegistry()
	agent.RegisterDefaultTools(reg, root)
	// ★ 实验环境剔除 codegraph/LSP 重工具：避免首次调用触发全量索引构建（污染耗时与公平性），
	//   且两 arm 使用同一精简工具集（搜索/读取/运行/git 等轻量只读工具）。
	for _, name := range reg.Definitions() {
		n := name.Function.Name
		if strings.HasPrefix(n, "codegraph_") || strings.HasPrefix(n, "lsp_") {
			reg.Unregister(n)
		}
	}

	loop := &agent.Loop{
		Provider:      rec,
		Registry:      reg,
		System:        buildEvalSystem(),
		MaxIterations: maxIter,
	}
	if arm == "dot" {
		loop.DiffusionThink = agent.DiffusionThinkOpts{Enabled: true, Candidates: 3}
	}

	runCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	msgs, err := loop.Run(runCtx, t.Prompt, nil)
	m.DurationMs = time.Since(start).Milliseconds()
	if err != nil {
		m.Err = err.Error()
	}

	m.PromptTokens, m.CompletionTok, m.TotalTokens, m.LLMCalls = rec.snapshot()
	m.Iterations = countAssistantMsgs(msgs)
	m.ToolCalls, m.FailedCalls, m.FirstTool, m.ToolSeq = analyzeTools(msgs)
	m.FirstToolHit = m.FirstTool != "" && strings.Contains(m.FirstTool, m.ExpectedTool)
	if loop.DiffuseStats != nil {
		m.DiffuseStats = loop.DiffuseStats
		m.DiffuseTokens = loop.DiffuseStats.TotalTokens
	}

	// ── LLM-as-Judge 质量评估（独立 recorder 单列评估 token）──
	evalRec := &usageRecorder{Provider: prov}
	ev := &agent.Evaluator{Provider: evalRec}
	eval, eerr := ev.Evaluate(runCtx, t.Prompt, agent.SummarizeRun(msgs))
	if eerr == nil {
		m.EvalTotal = eval.Total
		m.EvalCompletion = eval.Scores.Completion
		m.EvalCorrect = eval.Scores.Correctness
		m.EvalDepth = eval.Scores.Depth
		m.EvalEfficiency = eval.Scores.Efficiency
	}
	_, _, m.EvalTokens, _ = evalRec.snapshot()

	return m
}

// buildEvalSystem 实验用精简系统提示（不含产品级冗长规则，公平对比）。
func buildEvalSystem() string {
	return `你是 PairCode CodeAgent（实验评估环境）。请高效、准确地完成用户任务：
1. 优先使用专用工具（搜索/读取/运行/代码图谱等），避免无效调用。
2. 工具失败时分析原因并换一种方式重试，最多 2 次。
3. 完成任务后给出简洁的最终答复（含关键结论）。`
}

// ─── 指标计算 ───────────────────────────────────────────

func countAssistantMsgs(msgs []agent.Message) int {
	n := 0
	for _, m := range msgs {
		if m.Role == agent.RoleAssistant {
			n++
		}
	}
	return n
}

// analyzeTools 统计工具调用序列：总数、失败数、首个工具名。
func analyzeTools(msgs []agent.Message) (total, failed int, first string, seq []string) {
	for _, m := range msgs {
		if m.Role == agent.RoleAssistant {
			for _, tc := range m.ToolCalls {
				total++
				if first == "" {
					first = tc.Function.Name
				}
				seq = append(seq, tc.Function.Name)
			}
		}
		if m.Role == agent.RoleTool && strings.HasPrefix(strings.TrimSpace(m.Content), "Error:") {
			failed++
		}
	}
	return
}

// printMetric 控制台单行输出。
func printMetric(m armMetrics) {
	first := m.FirstTool
	if first == "" {
		first = "（无工具）"
	}
	hit := "✗"
	if m.FirstToolHit {
		hit = "✓"
	}
	succRate := 100.0
	if m.ToolCalls > 0 {
		succRate = float64(m.ToolCalls-m.FailedCalls) / float64(m.ToolCalls) * 100
	}
	fmt.Printf("   %s: tokens=%d(llm×%d) 迭代=%d 工具=%d(失败%d) 首步=%s[%s] 质量=%d 成功率=%.0f%% 耗时=%.1fs\n",
		m.Arm, m.TotalTokens, m.LLMCalls, m.Iterations, m.ToolCalls, m.FailedCalls, first, hit,
		m.EvalTotal, succRate, float64(m.DurationMs)/1000)
	if m.Arm == "dot" && m.DiffuseStats != nil {
		fmt.Printf("      └ 扩散: 候选=%d 发散=%d+收敛=%d tokens=%d 建议首步=%s\n",
			m.DiffuseStats.Candidates, m.DiffuseStats.DivergenceTokens, m.DiffuseStats.ConvergenceTokens,
			m.DiffuseStats.TotalTokens, m.DiffuseStats.SuggestedTool)
	}
	_ = filepath.Join // keep import
}
