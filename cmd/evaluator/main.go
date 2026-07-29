// cmd/evaluator — 独立的 AI Agent 会话评分工具
//
// 用法:
//   go run ./cmd/evaluator -conv-id <conv_id> -root <workspace_root>
//   go run ./cmd/evaluator -root <workspace_root>               # 评估最近一次会话
//
// 环境变量:
//   BASE_URL  LLM API base URL（如 https://api.deepseek.com/v1）
//   API_KEY   LLM API key
//   MODEL     LLM 模型名（如 deepseek-chat）
//
// 评分是一个独立的小 Agent：读取执行日志 → 调 LLM 语义化评分 → 输出报告。
// 不依赖主 agent 的 Provider/Registry/Loop，完全独立运行。
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ─── 数据结构 ─────────────────────────────────────────────

// ExecutionEntry 与 agent 包一致（避免导入依赖，保持独立）。
type ExecutionEntry struct {
	Round   int    `json:"round"`
	Agent   string `json:"agent"`
	Phase   string `json:"phase"`
	Summary string `json:"summary"`
}

// ExecutionLog 执行日志。
type ExecutionLog struct {
	Entries []ExecutionEntry `json:"entries"`
	Round   int              `json:"round"`
}

// scoreRequest 传给 LLM 的评分请求体（流式关闭，简单调用）。
type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// scoringResult LLM 输出的 JSON 评分。
type scoringResult struct {
	CompletionScore   float64  `json:"completion_score"`
	EfficiencyScore   float64  `json:"efficiency_score"`
	ReliabilityScore  float64  `json:"reliability_score"`
	AdaptabilityScore float64  `json:"adaptability_score"`
	Analysis          string   `json:"analysis"`
	Suggestions       []string `json:"suggestions"`
}

// ─── CLI ────────────────────────────────────────────────

func main() {
	convID := flag.String("conv-id", "", "对话 ID（可选；省略则评估最近一次会话）")
	root := flag.String("root", ".", "工作区根目录")
	baseURL := flag.String("base-url", "", "LLM API base URL（默认从环境变量 BASE_URL 读）")
	apiKey := flag.String("api-key", "", "LLM API key（默认从环境变量 API_KEY 读）")
	model := flag.String("model", "", "LLM 模型名（默认从环境变量 MODEL 读）")
	help := flag.Bool("help", false, "显示帮助")
	flag.Parse()

	if *help {
		fmt.Println(`Agent 会话评分工具 — 独立 LLM 评分小 Agent

用法:
  evaluator -conv-id <conv_id> -root <workspace_root>

选项:
  -conv-id    对话 ID（省略则评估最近一次会话）
  -root       工作区根目录（默认当前目录）
  -base-url   LLM API base URL（默认 $BASE_URL）
  -api-key    LLM API key（默认 $API_KEY）
  -model      LLM 模型名（默认 $MODEL）

环境变量: BASE_URL, API_KEY, MODEL`)
		return
	}

	// 读取配置：CLI 参数 > 环境变量
	base := *baseURL
	if base == "" {
		base = os.Getenv("BASE_URL")
	}
	key := *apiKey
	if key == "" {
		key = os.Getenv("API_KEY")
	}
	mdl := *model
	if mdl == "" {
		mdl = os.Getenv("MODEL")
	}

	if base == "" || key == "" || mdl == "" {
		fmt.Fprintln(os.Stderr, "错误：请设置 BASE_URL / API_KEY / MODEL（通过环境变量或 -base-url / -api-key / -model 参数）")
		os.Exit(1)
	}

	// ── 加载执行日志 ──
	var log *ExecutionLog
	if *convID != "" {
		log = loadLog(*root, *convID)
	} else {
		log = findLatestLog(*root)
	}

	if log == nil || len(log.Entries) == 0 {
		fmt.Fprintln(os.Stderr, "未找到执行日志。")
		fmt.Fprintf(os.Stderr, "执行日志存放位置: %s\n", filepath.Join(*root, ".pair", "execution_logs"))
		os.Exit(1)
	}

	if *convID == "" && log != nil {
		// 尝试从文件名推断 convID
		latestFile := findLatestLogFile(*root)
		if latestFile != "" {
			*convID = strings.TrimSuffix(filepath.Base(latestFile), ".json")
		}
	}

	// ── 构建评分 prompt ──
	prompt := buildScorePrompt(log)

	// ── 调 LLM 评分 ──
	fmt.Fprintf(os.Stderr, "📊 评分中（%d 条日志条目）…\n", len(log.Entries))
	result, err := callLLM(base, key, mdl, prompt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "LLM 评分失败: %v\n", err)
		os.Exit(1)
	}

	// ── 输出评分报告 ──
	printReport(*convID, log, result)
}

// ─── 加载日志 ───────────────────────────────────────────

func logDir(root string) string {
	return filepath.Join(root, ".pair", "execution_logs")
}

func loadLog(root, convID string) *ExecutionLog {
	path := filepath.Join(logDir(root), convID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "读取日志失败 %s: %v\n", path, err)
		return nil
	}
	var log ExecutionLog
	if err := json.Unmarshal(data, &log); err != nil {
		fmt.Fprintf(os.Stderr, "解析日志失败: %v\n", err)
		return nil
	}
	return &log
}

func findLatestLogFile(root string) string {
	dir := logDir(root)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	var latest string
	var latestTime time.Time
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().After(latestTime) {
			latestTime = info.ModTime()
			latest = e.Name()
		}
	}
	if latest == "" {
		return ""
	}
	return filepath.Join(dir, latest)
}

func findLatestLog(root string) *ExecutionLog {
	path := findLatestLogFile(root)
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var log ExecutionLog
	if err := json.Unmarshal(data, &log); err != nil {
		return nil
	}
	return &log
}

// ─── 构建评分 Prompt ──────────────────────────────────

func buildScorePrompt(log *ExecutionLog) string {
	var sb strings.Builder

	sb.WriteString("## 执行日志\n\n")
	sb.WriteString("| 轮次 | Agent | 阶段 | 摘要 |\n")
	sb.WriteString("|------|-------|------|------|\n")
	for _, e := range log.Entries {
		summary := e.Summary
		if len(summary) > 200 {
			summary = summary[:200] + "…"
		}
		summary = strings.ReplaceAll(summary, "|", "｜")
		fmt.Fprintf(&sb, "| %d | %s | %s | %s |\n", e.Round, e.Agent, e.Phase, summary)
	}

	return sb.String()
}

// ─── 调 LLM ────────────────────────────────────────────

func callLLM(baseURL, apiKey, model, logData string) (*scoringResult, error) {
	systemPrompt := `你是一个专业的 AI Agent 会话评估员。任务是评估一次 AI Agent 执行任务的表现。

请根据以下执行日志，从四个维度评分（0-100）：

1. **完成度** (completion_score): 任务是否达成目标？是否自然结束或明确完成？
2. **效率** (efficiency_score): 工具调用次数是否合理？轮次是否过多？有无冗余操作？
3. **可靠性** (reliability_score): 错误率是否过高？是否稳定完成任务？
4. **适应性** (adaptability_score): 遇到错误后能否有效恢复？有无重试行为？

评分标准：
- 90+ 优秀 | 80-89 良好 | 70-79 一般 | 60-69 需改进 | <60 差

请输出**严格合法的 JSON**（不带 markdown 代码块标记，纯 JSON 对象），格式：
{
  "completion_score": 85,
  "efficiency_score": 70,
  "reliability_score": 90,
  "adaptability_score": 65,
  "analysis": "一段简短的自然语言分析，描述总体表现",
  "suggestions": ["改进建议1", "改进建议2"]
}

只输出 JSON，不要附带其他文字。`

	userMsg := "请评估以下 agent 会话表现：\n\n" + logData

	req := chatRequest{
		Model: model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userMsg},
		},
		Stream: false,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := strings.TrimRight(baseURL, "/") + "/chat/completions"
	httpReq, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("LLM 请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("LLM HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(errBody)))
	}

	var chatResp chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("解析 LLM 响应失败: %w", err)
	}

	if chatResp.Error != nil && chatResp.Error.Message != "" {
		return nil, fmt.Errorf("LLM 返回错误: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("LLM 响应无 choices")
	}

	jsonStr := strings.TrimSpace(chatResp.Choices[0].Message.Content)
	// 清理可能的 markdown 代码块包裹
	if strings.HasPrefix(jsonStr, "```") {
		jsonStr = strings.TrimPrefix(jsonStr, "```json")
		jsonStr = strings.TrimPrefix(jsonStr, "```")
		if idx := strings.LastIndex(jsonStr, "```"); idx >= 0 {
			jsonStr = jsonStr[:idx]
		}
		jsonStr = strings.TrimSpace(jsonStr)
	}

	var sr scoringResult
	if err := json.Unmarshal([]byte(jsonStr), &sr); err != nil {
		return nil, fmt.Errorf("解析评分 JSON 失败: %w\n原始响应: %s", err, jsonStr)
	}

	// 钳制分值为 0-100
	clamp := func(v float64) float64 {
		if v < 0 {
			return 0
		}
		if v > 100 {
			return 100
		}
		return v
	}
	sr.CompletionScore = clamp(sr.CompletionScore)
	sr.EfficiencyScore = clamp(sr.EfficiencyScore)
	sr.ReliabilityScore = clamp(sr.ReliabilityScore)
	sr.AdaptabilityScore = clamp(sr.AdaptabilityScore)

	return &sr, nil
}

// ─── 输出报告 ───────────────────────────────────────────

func printReport(convID string, log *ExecutionLog, sr *scoringResult) {
	overall := sr.CompletionScore*0.35 + sr.EfficiencyScore*0.20 +
		sr.ReliabilityScore*0.30 + sr.AdaptabilityScore*0.15

	fmt.Println("## Agent 会话评分报告")
	fmt.Println()
	if convID != "" {
		fmt.Printf("**会话**: `%s`\n\n", convID)
	}

	fmt.Printf("**日志条目**: %d 条\n\n", len(log.Entries))

	// 评分卡片
	fmt.Println("| 维度 | 分数 | 评级 |")
	fmt.Println("|------|------|------|")
	dims := []struct {
		name  string
		score float64
	}{
		{"🎯 完成度", sr.CompletionScore},
		{"⚡ 效率", sr.EfficiencyScore},
		{"🔒 可靠性", sr.ReliabilityScore},
		{"🔄 适应性", sr.AdaptabilityScore},
		{"📊 总分", overall},
	}
	for _, d := range dims {
		fmt.Printf("| %s | **%.1f** | %s |\n", d.name, d.score, rating(d.score))
	}

	fmt.Println()
	fmt.Println("### 分析")
	fmt.Println(sr.Analysis)

	if len(sr.Suggestions) > 0 {
		fmt.Println()
		fmt.Println("### 改进建议")
		for _, s := range sr.Suggestions {
			fmt.Printf("- %s\n", s)
		}
	}

	fmt.Println()
	fmt.Println("---")
	fmt.Println("评分标准：90+ 优秀 | 80-89 良好 | 70-79 一般 | 60-69 需改进 | <60 差")
}

func rating(s float64) string {
	switch {
	case s >= 95:
		return "🏆 卓越"
	case s >= 90:
		return "🌟 优秀"
	case s >= 80:
		return "✅ 良好"
	case s >= 70:
		return "📈 一般"
	case s >= 60:
		return "⚠️ 需改进"
	default:
		return "❌ 差"
	}
}
