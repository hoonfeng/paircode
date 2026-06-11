package agent

// 任务评测评分 —— 忠实复刻参考 src/main/bench/evaluator.ts 的 LLM-as-Judge：
// 任务完成后，用评测模型按 4 维度（完成度/正确性/深度/效率）给执行质量打分（0-100）+ 优缺点 + 反馈。
//
// 注：参考的「评分驱动自动迭代改进配置」(evolution/auto-iteration 自我进化引擎) 不移植——那是独立大子系统；
// 此处只做「评分 + 反馈展示」这一可见、可用的核心。

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// EvalScores 4 个评分维度（满分 40/30/20/10，复刻参考评分维度）。
type EvalScores struct {
	Completion  int `json:"completion"`  // 完成度 0-40
	Correctness int `json:"correctness"` // 正确性 0-30
	Depth       int `json:"depth"`       // 深度 0-20
	Efficiency  int `json:"efficiency"`  // 效率 0-10
}

// Evaluation 一次任务评测结果（复刻参考 BenchEvaluation 核心字段）。
type Evaluation struct {
	Scores     EvalScores `json:"scores"`
	Total      int        `json:"total"`
	Strengths  []string   `json:"strengths"`
	Weaknesses []string   `json:"weaknesses"`
	Feedback   string     `json:"feedback"`
}

// Evaluator 评测 Agent。Provider 用评测模型（建议 = 执行模型，temperature 0.2）。
type Evaluator struct {
	Provider     Provider
	SystemPrompt string // 评分系统提示（空=用内置默认）；宿主可从 config 加载覆盖（非硬编码）
}

// DefaultJudgePrompt 内置默认评分提示（config/roles/judge.md 缺失时的回退）。
func DefaultJudgePrompt() string { return judgeSystemPrompt }

// judgeSystemPrompt 复刻参考 bench/evaluator.ts 的 JUDGE_SYSTEM_PROMPT（评分维度 + 输出格式）。
const judgeSystemPrompt = `你是一个严格的代码质量评审专家。请根据以下维度评估任务完成质量：

## 评分维度

1. **完成度 (0-40)**: 任务要求的所有阶段是否都完成？输出是否完整？
2. **正确性 (0-30)**: 技术方案是否正确？代码/分析是否存在错误？
3. **深度 (0-20)**: 分析是否深入？是否考虑了架构/设计/权衡？
4. **效率 (0-10)**: 输出是否简洁清晰？结构是否合理？

## 输出格式

你必须只输出 JSON（不要其他任何内容）：
{"scores":{"completion":N,"correctness":N,"depth":N,"efficiency":N},"total":N,"strengths":["...",...],"weaknesses":["...",...],"feedback":"..."}`

// Evaluate 对一次任务执行评分。summary 为执行摘要（见 SummarizeRun）。复刻参考 evaluateRun + callLLM。
func (e *Evaluator) Evaluate(ctx context.Context, task, summary string) (Evaluation, error) {
	msg := "任务: " + task + "\n\n## Agent 执行摘要\n" + summary + `

请根据评分标准评估这个 Agent 的执行质量。考虑以下方面：
- 任务是否完成？工具调用是否有效？
- 是否有错误或失败的工具调用？
- 执行的效率和路径是否合理？
- 最终结果的质量如何？`
	if r := []rune(msg); len(r) > 16000 { // 复刻参考 slice(0,16000)
		msg = string(r[:16000])
	}
	resp, err := e.Provider.Chat(ctx, []Message{
		{Role: RoleSystem, Content: orDefault(e.SystemPrompt, judgeSystemPrompt)},
		{Role: RoleUser, Content: msg},
	}, nil, nil)
	if err != nil {
		return Evaluation{}, err
	}
	return parseEvaluation(resp.Content), nil
}

// parseEvaluation 抽 JSON 评分（首 { 到末 }）。无 total 则取各维度之和；解析失败→0 分 + 失败说明（复刻参考）。
func parseEvaluation(raw string) Evaluation {
	i, j := strings.IndexByte(raw, '{'), strings.LastIndexByte(raw, '}')
	if i >= 0 && j > i {
		var ev Evaluation
		if err := json.Unmarshal([]byte(raw[i:j+1]), &ev); err == nil {
			if ev.Total == 0 {
				ev.Total = ev.Scores.Completion + ev.Scores.Correctness + ev.Scores.Depth + ev.Scores.Efficiency
			}
			return ev
		}
	}
	return Evaluation{Weaknesses: []string{"Judge 评分解析失败"}, Feedback: "评测模型输出无法解析为 JSON"}
}

// SummarizeRun 从本轮完整消息历史提炼执行摘要（工具调用/结果/错误计数 + 最近 10 次调用 + 最终答复）。
// 复刻参考 readAgentLog 的关键事件提取（companion 无独立日志文件，直接用 loop 返回的 msgs）。
func SummarizeRun(msgs []Message) string {
	toolCalls, toolResults, errs := 0, 0, 0
	var recent []string
	var finalText string
	for _, m := range msgs {
		switch m.Role {
		case RoleAssistant:
			toolCalls += len(m.ToolCalls)
			for _, tc := range m.ToolCalls {
				recent = append(recent, "["+tc.Function.Name+"] "+truncRunesAgent(tc.Function.Arguments, 200))
			}
			if strings.TrimSpace(m.Content) != "" {
				finalText = m.Content // 末次有正文的助手消息 ≈ 最终答复
			}
		case RoleTool:
			toolResults++
			if strings.HasPrefix(strings.TrimSpace(m.Content), "Error:") {
				errs++
			}
		}
	}
	if len(recent) > 10 {
		recent = recent[len(recent)-10:]
	}
	var b strings.Builder
	fmt.Fprintf(&b, "工具调用: %d 次\n工具结果: %d 次\n错误: %d 次\n", toolCalls, toolResults, errs)
	if len(recent) > 0 {
		b.WriteString("\n最近工具调用:\n  " + strings.Join(recent, "\n  ") + "\n")
	}
	if strings.TrimSpace(finalText) != "" {
		b.WriteString("\n最终答复:\n" + truncRunesAgent(finalText, 2000))
	}
	return b.String()
}

// HasToolActivity 本轮是否有工具调用（无→纯问答聊天，不值得评测）。
func HasToolActivity(msgs []Message) bool {
	for _, m := range msgs {
		if len(m.ToolCalls) > 0 {
			return true
		}
	}
	return false
}
