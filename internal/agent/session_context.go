// session_context.go — 会话连贯性上下文构建器
// 核心思想：主动注入 > 被动工具。Agent 不应依赖自己"想起来"去调用 memory_* 工具，
// 而应在每次对话恢复时，由系统主动注入：任务进度、对话摘要、相关记忆、项目归属、工作区结构。
//
// 注入点：buildWebSystemDynamic() 和 Loop.Run() 的初始消息。

package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// ── 会话连贯性上下文 TTL 缓存 ──────────────────────────────
// BuildResumeContext 每次新对话都会重建 system 动态后缀（含记忆召回/Git 状态/任务进度等），
// 同一对话连续轮次间的任何内容抖动都会让 KV 缓存前缀从 system 尾部断裂。
// 这里对输出做短 TTL 缓存（同 convID+roots，60s 内复用），显著降低断裂频率。

var (
	resumeCtxCacheMu   sync.Mutex
	resumeCtxCacheKey  string
	resumeCtxCacheVal  string
	resumeCtxCacheAt   time.Time
	resumeCtxCacheTTL  = 60 * time.Second
)

// ── 会话连贯性上下文 ─────────────────────────────────────────

// SessionContext 聚合了让 Agent 理解"当前在哪、在干什么、干过什么"的所有信息。
type SessionContext struct {
	ConversationSummary    string   // 对话摘要（来自 MessageStore.SetSummary）
	TaskProgress           string   // 当前任务进度（来自持久化的 update_tasks）
	RelevantMemories       string   // 自动召回的相关项目记忆
	ProjectHint            string   // 当前对话最可能相关的项目目录
	WorkspaceStructure     string   // 工作区结构概览
	LastActivity           string   // 上次活动时间/内容
	RecentFileEdits        []string // 最近编辑的文件列表（从工具调用精确提取）
	GitStatus              string   // Git 状态快照（最近提交 + 未提交改动）
	CodeGraphStats         string   // 代码图谱统计（实体数/覆盖率）
	BuildStatus            string   // 最近构建状态（编译成功/失败、二进制时间戳）
	KBStaleness            string   // 知识库过期警告（哪些条目引用了不存在的文件）
}

// BuildSessionContext 构建会话连贯性上下文。
// convID: 当前对话 ID
// workspaceRoots: 工作区所有根目录
// currentTask: 当前用户任务文本
// history: 已有历史消息（用于推断项目归属和最近活动）
// store: 消息存储（用于读取对话摘要和元数据）
func BuildSessionContext(convID string, workspaceRoots []string, currentTask string, history []Message, store MessageStoreReader) *SessionContext {
	sc := &SessionContext{}

	// 1. 对话摘要（从持久化摘要中读取）
	var convSummary string
	if store != nil {
		if meta, err := store.GetConversation(convID); err == nil && meta != nil && meta.Summary != "" {
			sc.ConversationSummary = meta.Summary
			convSummary = meta.Summary
		}
	}

	// 2. 任务进度（从持久化的 update_tasks 读取）
	sc.TaskProgress = buildTaskProgressContext(workspaceRoots)

	// 3. 自动召回相关项目记忆（增强：不仅基于 task，也基于对话摘要和历史最近消息）
	sc.RelevantMemories = buildEnhancedRecallContext(workspaceRoots, currentTask, convSummary, history)

	// 4. 项目归属推断（根据历史和当前任务）
	sc.ProjectHint = inferProject(workspaceRoots, currentTask, history)

	// 5. 工作区结构概览
	sc.WorkspaceStructure = buildWorkspaceStructureOverview(workspaceRoots)

	// 6. 最近活动
	sc.LastActivity = buildLastActivity(history, convID, store)

	// 7. 最近编辑的文件
	sc.RecentFileEdits = extractRecentEdits(history, workspaceRoots)

	// 8. Git 状态快照（感知项目变更）
	sc.GitStatus = buildGitStatus(workspaceRoots)

	// 9. 代码图谱统计
	sc.CodeGraphStats = buildCodeGraphStats()

	// 10. 构建状态（最近编译结果 + 二进制时间戳）
	sc.BuildStatus = buildBuildStatus(workspaceRoots)

	// 11. 知识库过期检测
	sc.KBStaleness = buildKBStaleness(workspaceRoots)

	return sc
}

// FormatForInjection 将会话上下文格式化为可注入 LLM 的文本。
// 只在有实质内容时返回非空字符串（避免注入空白块浪费 token）。
func (sc *SessionContext) FormatForInjection() string {
	if sc == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n\n# ★ 会话连贯性上下文（系统主动注入）\n")
	b.WriteString("> 以下信息帮助你理解当前会话的状态：你在哪、在干什么、干过什么。\n")
	b.WriteString("> 请勿重复已完成的工作，直接基于此上下文继续推进任务。\n")

	hasContent := false

	if sc.ConversationSummary != "" {
		b.WriteString("\n## 对话摘要\n")
		b.WriteString(sc.ConversationSummary + "\n")
		hasContent = true
	}

	if sc.TaskProgress != "" {
		b.WriteString("\n## 任务进度（持久化）\n")
		b.WriteString(sc.TaskProgress + "\n")
		hasContent = true
	}

	if sc.ProjectHint != "" {
		b.WriteString("\n## 当前项目归属\n")
		b.WriteString(sc.ProjectHint + "\n")
		hasContent = true
	}

	if sc.RelevantMemories != "" {
		b.WriteString("\n## 相关项目记忆（自动召回）\n")
		b.WriteString(sc.RelevantMemories + "\n")
		hasContent = true
	}

	if sc.LastActivity != "" {
		b.WriteString("\n## 最近活动\n")
		b.WriteString(sc.LastActivity + "\n")
		hasContent = true
	}

	if sc.GitStatus != "" {
		b.WriteString("\n## Git 状态（项目变更感知）\n")
		b.WriteString(sc.GitStatus + "\n")
		hasContent = true
	}

	if sc.CodeGraphStats != "" {
		b.WriteString("\n## 代码图谱统计\n")
		b.WriteString(sc.CodeGraphStats + "\n")
		hasContent = true
	}

	if sc.BuildStatus != "" {
		b.WriteString("\n## 构建状态\n")
		b.WriteString(sc.BuildStatus + "\n")
		hasContent = true
	}

	if sc.KBStaleness != "" {
		b.WriteString("\n## ⚠️ 知识库过期警告\n")
		b.WriteString(sc.KBStaleness + "\n")
		hasContent = true
	}

	if len(sc.RecentFileEdits) > 0 {
		b.WriteString("\n## 本次对话已编辑的文件\n")
		for _, f := range sc.RecentFileEdits {
			b.WriteString(fmt.Sprintf("- %s\n", f))
		}
		hasContent = true
	}

	if sc.WorkspaceStructure != "" {
		b.WriteString("\n## 工作区结构概览\n")
		b.WriteString(sc.WorkspaceStructure + "\n")
		hasContent = true
	}

	if !hasContent {
		return ""
	}
	return b.String()
}

// ── 子构建器 ────────────────────────────────────────────────

// buildTaskProgressContext 从所有工作区的持久化任务中构建进度上下文。
func buildTaskProgressContext(roots []string) string {
	for _, root := range roots {
		tm := NewTaskManager(root)
		tasks := tm.List("")
		if len(tasks) == 0 {
			continue
		}
		summary := tm.GetSummary()
		if summary.Total == 0 {
			continue
		}

		var b strings.Builder
		bar := buildProgressBar(summary.Completed, summary.Total, 20)
		b.WriteString(fmt.Sprintf("项目 %s：共 %d 项（%d 完成，%d 进行中，%d 待执行）\n",
			filepath.Base(root), summary.Total, summary.Completed, summary.InProgress, summary.Pending))
		b.WriteString(fmt.Sprintf("进度: %s %d/%d (%.0f%%)\n", bar, summary.Completed, summary.Total,
			float64(summary.Completed)*100/float64(summary.Total)))

		// 列出进行中和待执行的任务
		ready := tm.GetReady()
		blocked := tm.GetBlocked()

		if len(ready) > 0 {
			b.WriteString("\n进行中/可执行:\n")
			for _, t := range ready {
				statusIcon := "▸"
				if t.Status == TaskInProgress {
					statusIcon = "◉"
				}
				b.WriteString(fmt.Sprintf("  %s [%s] %s\n", statusIcon, t.ID, t.Subject))
				if t.Description != "" {
					b.WriteString(fmt.Sprintf("    → %s\n", t.Description))
				}
			}
		}
		if len(blocked) > 0 {
			b.WriteString("\n阻塞中:\n")
			for _, bt := range blocked {
				blockers := make([]string, len(bt.BlockedBy))
				for i, bk := range bt.BlockedBy {
					blockers[i] = bk.Subject
				}
				b.WriteString(fmt.Sprintf("  ✗ [%s] %s ← 等待: %s\n", bt.Task.ID, bt.Task.Subject,
					strings.Join(blockers, ", ")))
			}
		}
		return b.String()
	}
	return ""
}

// buildRecallContext 从所有工作区自动召回相关记忆。
func buildRecallContext(roots []string, task string) string {
	return buildEnhancedRecallContext(roots, task, "", nil)
}

// buildEnhancedRecallContext 增强版记忆召回：不仅基于当前 task 文本，
// 还融合对话摘要和历史最近消息，大幅提高召回率（解决"继续"类短任务召回不到记忆的问题）。
func buildEnhancedRecallContext(roots []string, task, convSummary string, history []Message) string {
	// 聚合所有召回关键词
	var recallText strings.Builder
	recallText.WriteString(task)
	if convSummary != "" {
		recallText.WriteString(" " + convSummary)
	}
	// 取最近 3 条用户消息和最后一条助手消息作为补充上下文
	userCount := 0
	var lastAssistantContent string
	for i := len(history) - 1; i >= 0; i-- {
		m := history[i]
		switch m.Role {
		case RoleUser:
			if userCount < 3 {
				recallText.WriteString(" " + m.Content)
				userCount++
			}
		case RoleAssistant:
			if lastAssistantContent == "" && m.Content != "" {
				lastAssistantContent = m.Content
				recallText.WriteString(" " + sessionTruncateStr(m.Content, 200))
			}
		}
		if userCount >= 3 && lastAssistantContent != "" {
			break
		}
	}

	var allHits []string
	mergedText := recallText.String()
	for _, root := range roots {
		if result := RecallMemories(root, mergedText, 5); result != "" {
			allHits = append(allHits, result)
		}
	}
	if len(allHits) == 0 {
		return ""
	}
	return strings.Join(allHits, "\n")
}

// inferProject 推断当前对话最可能相关的项目目录。
// 策略：①分析历史消息中提到的文件路径 ②分析当前任务中提到的项目名 ③默认第一个工作区根。
func inferProject(roots []string, task string, history []Message) string {
	if len(roots) <= 1 {
		return "" // 单项目无需提示
	}

	// 统计每个根目录被引用的次数
	scores := make(map[string]int)
	for _, r := range roots {
		scores[r] = 0
	}

	// 从历史消息中统计文件路径引用
	for _, msg := range history {
		content := strings.ToLower(msg.Content)
		for _, r := range roots {
			base := strings.ToLower(filepath.Base(r))
			if strings.Contains(content, base) {
				scores[r] += 2
			}
			// 检查完整路径
			if strings.Contains(content, strings.ToLower(r)) {
				scores[r] += 3
			}
		}
	}

	// 从当前任务中统计
	taskLower := strings.ToLower(task)
	for _, r := range roots {
		base := strings.ToLower(filepath.Base(r))
		if strings.Contains(taskLower, base) {
			scores[r] += 5
		}
	}

	// 找最高分
	var best string
	bestScore := 0
	for _, r := range roots {
		if scores[r] > bestScore {
			best = r
			bestScore = scores[r]
		}
	}

	if bestScore == 0 {
		return fmt.Sprintf("多项目工作区。如任务未明确指定项目，默认操作主项目：%s", roots[0])
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("根据对话上下文推断，当前任务最可能关联的项目是：**%s**\n", filepath.Base(best)))
	b.WriteString(fmt.Sprintf("路径：%s\n", best))
	if len(roots) > 0 && best != roots[0] {
		b.WriteString(fmt.Sprintf("（主项目为 %s，当前焦点为 %s）\n", filepath.Base(roots[0]), filepath.Base(best)))
	}
	return b.String()
}

// buildWorkspaceStructureOverview 生成工作区结构概览。
// 仅扫描顶层目录和关键文件，不深入子目录，控制在 2000 字符以内。
func buildWorkspaceStructureOverview(roots []string) string {
	if len(roots) == 0 {
		return ""
	}

	var b strings.Builder
	for _, root := range roots {
		name := filepath.Base(root)
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}

		b.WriteString(fmt.Sprintf("### %s (%s)\n", name, root))

		// 分类：目录 + 关键文件
		var dirs, keyFiles []string
		for _, e := range entries {
			n := e.Name()
			// 跳过隐藏目录和常见忽略目录
			if strings.HasPrefix(n, ".") || n == "node_modules" || n == "vendor" {
				continue
			}
			if e.IsDir() {
				dirs = append(dirs, n)
			} else {
				// 只列出关键文件（配置文件、入口文件、说明文档等）
				ext := strings.ToLower(filepath.Ext(n))
				switch ext {
				case ".md", ".mod", ".sum", ".json", ".yaml", ".yml", ".toml", ".go":
					if n == "go.mod" || n == "go.sum" || n == "go.work" || n == "package.json" ||
						n == "tsconfig.json" || n == "vite.config.ts" || n == "README.md" ||
						n == "Makefile" || n == "Dockerfile" || strings.HasSuffix(n, ".go") {
						keyFiles = append(keyFiles, n)
					}
				}
			}
		}

		sort.Strings(dirs)
		sort.Strings(keyFiles)

		dirStr := strings.Join(dirs, ", ")
		if len(dirStr) > 200 {
			dirStr = dirStr[:200] + "…"
		}
		b.WriteString(fmt.Sprintf("  目录: %s\n", dirStr))

		if len(keyFiles) > 0 {
			fileStr := strings.Join(keyFiles, ", ")
			if len(fileStr) > 200 {
				fileStr = fileStr[:200] + "…"
			}
			b.WriteString(fmt.Sprintf("  关键文件: %s\n", fileStr))
		}
		b.WriteString("\n")
	}

	result := b.String()
	if len(result) > 2500 {
		result = result[:2500] + "\n…（已截断）"
	}
	return result
}

// buildLastActivity 构建最近活动信息。
func buildLastActivity(history []Message, convID string, store MessageStoreReader) string {
	// 优先从对话元数据获取
	if store != nil {
		if meta, err := store.GetConversation(convID); err == nil && meta != nil {
			if meta.UpdatedAt != "" {
				return fmt.Sprintf("对话最后更新时间：%s", meta.UpdatedAt)
			}
		}
	}

	// 从历史消息推断
	if len(history) == 0 {
		return "新对话，无历史活动。"
	}

	// 统计消息数量和角色
	userMsgs, assistantMsgs := 0, 0
	var lastUserMsg, lastAssistantMsg string
	for i := len(history) - 1; i >= 0; i-- {
		m := history[i]
		switch m.Role {
		case RoleUser:
			if userMsgs == 0 {
				lastUserMsg = sessionTruncateStr(m.Content, 100)
			}
			userMsgs++
		case RoleAssistant:
			if assistantMsgs == 0 {
				lastAssistantMsg = sessionTruncateStr(m.Content, 100)
			}
			assistantMsgs++
		}
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("历史消息：%d 条用户消息，%d 条助手回复。\n", userMsgs, assistantMsgs))
	if lastUserMsg != "" {
		b.WriteString(fmt.Sprintf("最近用户消息：%s\n", lastUserMsg))
	}
	if lastAssistantMsg != "" {
		b.WriteString(fmt.Sprintf("最近助手回复：%s\n", lastAssistantMsg))
	}
	return b.String()
}

// extractRecentEdits 从历史消息中提取最近编辑的文件列表。
// fileModifyTools 会修改/操作文件的工具名集合。
var fileModifyTools = map[string]bool{
	"edit_file":    true,
	"write_file":   true,
	"multi_edit":   true,
	"move_file":    true,
	"delete_file":  true,
	"write_binary": true,
}

// toolPathParams 各工具的文件路径参数名（可能有多个）。
var toolPathParams = map[string][]string{
	"edit_file":    {"path"},
	"write_file":   {"path"},
	"multi_edit":   {"path"},
	"move_file":    {"from", "to"},
	"delete_file":  {"path"},
	"write_binary": {"path"},
}

// extractRecentEdits 从历史消息中提取最近编辑的文件列表。
// 优先级：①ToolCalls 中的文件参数（最精确）②RoleTool 结果中含路径的行 ③助手文本中的路径（回退）。
func extractRecentEdits(history []Message, roots []string) []string {
	seen := make(map[string]bool)
	var files []string
	maxFiles := 20

	// ── 第 1 遍：从 assistant 消息的 ToolCalls 中提取（精确） ──
	for i := len(history) - 1; i >= 0 && len(files) < maxFiles; i-- {
		msg := history[i]
		if msg.Role != RoleAssistant || len(msg.ToolCalls) == 0 {
			continue
		}
		for _, tc := range msg.ToolCalls {
			p := extractPathFromToolArgs(tc)
			if p != "" && !seen[p] {
				seen[p] = true
				files = append(files, p)
			}
		}
	}

	// ── 第 2 遍：从 RoleTool 消息的结果中提取 ──
	for i := len(history) - 1; i >= 0 && len(files) < maxFiles; i-- {
		msg := history[i]
		if msg.Role != RoleTool || msg.Name == "" {
			continue
		}
		// 只处理文件修改工具
		if !fileModifyTools[msg.Name] {
			continue
		}
		for _, p := range extractPathsFromText(msg.Content) {
			if p != "" && !seen[p] {
				seen[p] = true
				files = append(files, p)
			}
		}
	}

	// ── 第 3 遍：从助手文本内容中提取（回退） ──
	for i := len(history) - 1; i >= 0 && len(files) < maxFiles; i-- {
		msg := history[i]
		if msg.Role != RoleAssistant || msg.Content == "" {
			continue
		}
		for _, p := range extractPathsFromText(msg.Content) {
			if p != "" && !seen[p] {
				seen[p] = true
				files = append(files, p)
			}
		}
	}

	return files
}

// extractPathFromToolArgs 从一次工具调用的 JSON 参数中提取文件路径。
func extractPathFromToolArgs(tc ToolCall) string {
	paramNames, ok := toolPathParams[tc.Function.Name]
	if !ok {
		return ""
	}
	var args map[string]any
	if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
		return ""
	}
	for _, pn := range paramNames {
		if v, ok := args[pn]; ok {
			if s, ok := v.(string); ok && s != "" {
				return s
			}
		}
	}
	return ""
}

// extractPathsFromText 从文本中提取所有像文件路径的字符串。
func extractPathsFromText(text string) []string {
	var paths []string
	seen := make(map[string]bool)
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// 用空格/标点分词
		parts := strings.FieldsFunc(line, func(r rune) bool {
			return r == ' ' || r == '`' || r == '"' || r == '\'' || r == '[' || r == ']' || r == '(' || r == ')' || r == ','
		})
		for _, part := range parts {
			part = strings.Trim(part, "`\"'[](),*")
			if isFilePathLike(part) && !seen[part] {
				seen[part] = true
				paths = append(paths, part)
			}
		}
	}
	return paths
}

// hasCJK 判断字符串是否含中文字符（中文自然语言短语不是文件路径）。
func hasCJK(s string) bool {
	for _, r := range s {
		if r >= '\u4e00' && r <= '\u9fff' {
			return true
		}
	}
	return false
}

// isTechnicalPhrase 判断是否为技术名词复合短语（≥2 段纯字母数字，如 HTML/CSS/JS、
// Proxy/Generator、from/to、WebCore/WebKit）——真实文件引用末段必带扩展名或符号，
// 纯字母数字 2 段以上多为文档列举的技术栈/术语，跳过防误报（漏报优于误报）。
func isTechnicalPhrase(s string) bool {
	if !strings.Contains(s, "/") {
		return false
	}
	segs := strings.Split(s, "/")
	if len(segs) < 2 {
		return false
	}
	for _, seg := range segs {
		if seg == "" {
			return false
		}
		for _, r := range seg {
			if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9') {
				return false
			}
		}
	}
	return true
}

// isAPIPhrase 判断是否为 API/对象名短语（含 . 但末段无代码扩展名）：
// EventTarget.AddEventListener/DispatchEvent、.pair/project-info 等。
// 文件级检测要求末段带代码扩展名（page/frame.go 保留、EventTarget… 跳过）。
func isAPIPhrase(s string) bool {
	if !strings.Contains(s, ".") {
		return false
	}
	segs := strings.Split(s, "/")
	last := segs[len(segs)-1]
	for _, ext := range codeFileExts {
		if strings.HasSuffix(strings.ToLower(last), ext) {
			return false // 末段带扩展名 → 真文件引用
		}
	}
	return true
}

// codeFileExts 代码文件扩展名（isFilePathLike 同源，避免重复声明）。
var codeFileExts = []string{".go", ".ts", ".js", ".vue", ".py", ".rs", ".java", ".cpp", ".c", ".h",
	".md", ".json", ".yaml", ".yml", ".toml", ".xml", ".html", ".css", ".scss", ".sql"}

// isFilePathLike 判断字符串是否像文件路径（含代码文件扩展名或路径分隔符）。
func isFilePathLike(s string) bool {
	if len(s) < 3 || len(s) > 200 {
		return false
	}
	codeExts := []string{".go", ".ts", ".js", ".vue", ".py", ".rs", ".java", ".cpp", ".c", ".h",
		".md", ".json", ".yaml", ".yml", ".toml", ".xml", ".html", ".css", ".scss", ".sql"}
	for _, ext := range codeExts {
		if strings.HasSuffix(strings.ToLower(s), ext) {
			return true
		}
	}
	if strings.Contains(s, "/") || strings.Contains(s, "\\") {
		return true
	}
	return false
}

// sessionTruncateStr 截断字符串到指定长度。
func sessionTruncateStr(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "…"
}

// ── Git 状态与代码图谱 ──────────────────────────────────────

// buildGitStatus 从所有工作区获取 Git 状态快照。
func buildGitStatus(roots []string) string {
	if len(roots) == 0 {
		return ""
	}
	var b strings.Builder
	for _, root := range roots {
		name := filepath.Base(root)
		logOutput, logErr := runGitCmd(root, "log", "--oneline", "-5", "--no-decorate")
		if logErr != nil {
			continue
		}
		logOutput = strings.TrimSpace(logOutput)
		statusOutput, _ := runGitCmd(root, "status", "--porcelain=v1", "--branch")
		statusOutput = strings.TrimSpace(statusOutput)
		if logOutput == "" && statusOutput == "" {
			continue
		}
		b.WriteString(fmt.Sprintf("### %s\n", name))
		if logOutput != "" {
			b.WriteString(fmt.Sprintf("最近提交:\n%s\n", logOutput))
		}
		if statusOutput != "" && !strings.HasPrefix(statusOutput, "##") {
			b.WriteString(fmt.Sprintf("未提交改动:\n%s\n", statusOutput))
		} else if strings.HasPrefix(statusOutput, "##") && !strings.Contains(statusOutput, "\n") {
			b.WriteString("工作区干净\n")
		} else if statusOutput != "" {
			b.WriteString(statusOutput + "\n")
		}
	}
	result := strings.TrimSpace(b.String())
	if len(result) > 2000 {
		result = result[:2000] + "\n…（已截断）"
	}
	return result
}

// runGitCmd 在指定目录运行 git 命令。
func runGitCmd(dir string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// buildBuildStatus 检查各工作区主项目的构建状态（二进制是否存在、最后构建时间）。
func buildBuildStatus(roots []string) string {
	if len(roots) == 0 {
		return ""
	}
	var b strings.Builder
	for _, root := range roots {
		name := filepath.Base(root)
		// 检查常见二进制输出位置
		candidates := []string{
			filepath.Join(root, "companion.exe"),
			filepath.Join(root, "release", "companion.exe"),
			filepath.Join(root, "bin", "companion.exe"),
		}
		for _, bin := range candidates {
			info, err := os.Stat(bin)
			if err == nil {
				b.WriteString(fmt.Sprintf("**%s** 二进制存在: `%s` (%s, 最后构建: %s)\n",
					name, filepath.Base(bin),
					formatFileSize(info.Size()),
					info.ModTime().Format("2006-01-02 15:04:05")))
				// 如果超过 24 小时，提示可能需要重建
				if time.Since(info.ModTime()) > 24*time.Hour {
					b.WriteString("  ⚠️ 二进制超过 24 小时未重新构建，如有代码变更建议 go build。\n")
				}
				break
			}
		}
		// 也检查 go build 缓存（go build 是否可用）
		_, goErr := exec.LookPath("go")
		if goErr != nil {
			b.WriteString(fmt.Sprintf("**%s**: ⚠️ go 命令不可用，无法编译。\n", name))
		}
	}
	result := strings.TrimSpace(b.String())
	if len(result) > 800 {
		result = result[:800] + "\n…（已截断）"
	}
	return result
}

// formatFileSize 格式化文件大小。
func formatFileSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

// buildKBStaleness 检测项目知识库条目是否过期（引用不存在的文件）。
// 仅当有过期条目时返回非空，避免无意义 token 消耗。
func buildKBStaleness(roots []string) string {
	if len(roots) == 0 {
		return ""
	}
	var allStale []string
	for _, root := range roots {
		pairDir := filepath.Join(root, ".pair", "project-info")
		entries, err := os.ReadDir(pairDir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}
			entryPath := filepath.Join(pairDir, entry.Name())
			data, err := os.ReadFile(entryPath)
			if err != nil {
				continue
			}
			// 快速扫描：检查是否引用了不存在的路径
			staleRefs := scanStaleRefs(string(data), root)
			if len(staleRefs) > 0 {
				name := strings.TrimSuffix(entry.Name(), ".md")
				// 只记录前 3 个过期引用
				preview := staleRefs
				if len(preview) > 3 {
					preview = preview[:3]
				}
				allStale = append(allStale, fmt.Sprintf("  - **%s**: %s（等 %d 处）",
					name, strings.Join(preview, "、"), len(staleRefs)))
			}
		}
	}
	if len(allStale) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("以下知识库条目引用了不存在的文件/目录，内容可能过时（共 %d 条）：\n", len(allStale)))
	// 最多列 10 条
	limit := len(allStale)
	if limit > 10 {
		limit = 10
	}
	for i := 0; i < limit; i++ {
		b.WriteString(allStale[i] + "\n")
	}
	if len(allStale) > 10 {
		b.WriteString(fmt.Sprintf("  … 还有 %d 条。建议运行 project_info_verify 查看全量，并更新或删除过时条目。\n", len(allStale)-10))
	}
	b.WriteString("过时条目用 project_info_delete 删除或用 project_info_write 更新。")
	return b.String()
}

// scanStaleRefs 扫描文本中引用的文件/目录路径，返回不存在的引用。
// 跳过：裸文件名（无目录分隔符，可能在任意子目录）、已知其他项目的路径前缀、
// 自然语言/技术名词短语（含中文、≥3 段纯字母数字复合如 HTML/CSS/JS、glob 通配符）。
func scanStaleRefs(text, workspaceRoot string) []string {
	var refs []string
	seen := make(map[string]bool)
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		parts := strings.FieldsFunc(line, func(r rune) bool {
			// ASCII 空白/引号/括号/逗号/星号 + 中文标点（防止「page/frame.go（不存在）」粘连）
			return r == ' ' || r == '`' || r == '"' || r == '\'' || r == '[' || r == ']' ||
				r == '(' || r == ')' || r == ',' || r == '*' ||
				r == '\u3000' || r == '\uff08' || r == '\uff09' || r == '\u3001' ||
				r == '\uff0c' || r == '\uff1a' || r == '\u3010' || r == '\u3011'
		})
		for _, part := range parts {
			part = strings.Trim(part, "`\"'[](),*")
			if !isFilePathLike(part) || seen[part] {
				continue
			}
			// 跳过裸文件名（无目录分隔符）：它们可能在任意子目录中，不做误报
			if !strings.Contains(part, "/") && !strings.Contains(part, "\\") {
				continue
			}
			// 自然语言/技术名词短语不是文件路径（防误报）：
			//   - 含中文：知识库中文文案中的「项目目标/愿景/里程碑」「架构/模块-x」等
			//   - glob 通配（* ?）：模式不是具体路径
			//   - 尾斜杠：目录引用痕迹（loader/、editor/、cache/）——目录级引用误报率
			//     远高于价值（术语常带尾斜杠），放弃目录检测、只保留文件级检测
			//   - ≥2 段纯字母数字复合：HTML/CSS/JS、Proxy/Generator、from/to、WebCore/WebKit
			//   - 含 . 但末段无代码扩展名：API 名（EventTarget.AddEventListener/DispatchEvent、
			//     .pair/project-info 目录）——文件级检测要求末段带扩展名
			if hasCJK(part) || strings.ContainsAny(part, "*?") ||
				strings.HasSuffix(part, "/") || strings.HasSuffix(part, "\\") ||
				isTechnicalPhrase(part) || isAPIPhrase(part) ||
				strings.HasSuffix(strings.ToLower(part), ".h") { // C/C++ 头文件：Go 工作区不可能有，必为对标文档参考
				continue
			}
			// 跳过已知属于 wb-ui 等外部项目的路径
			if strings.HasPrefix(part, "jsc/") || strings.HasPrefix(part, "wb-ui/") ||
				strings.HasPrefix(part, "skia/") || strings.HasPrefix(part, "goui/") {
				continue
			}
			// 跳过已明确移除的旧文件/目录 + 外部参考项目路径
			// （loader/cache/、platform/network/ 等是 WebKit C++ 参考架构路径，不是本工作区文件）
			skipPrefixes := []string{
				"cmd/companion/webui_desktop", "cmd/companion/bridge/",
				"webui_desktop.go", "webui_webonly.go", "conversations.js",
				"build.sh", "pair/conversations", "pair/memory_index",
				"loader/cache", "loader/", "cache/",
				"platform/network", "platform/graphics", "platform/text",
				"WebCore/", "WebKit/Source",
			}
			skip := false
			for _, sp := range skipPrefixes {
				if strings.HasPrefix(part, sp) || strings.HasPrefix(strings.ToLower(part), strings.ToLower(sp)) {
					skip = true
					break
				}
			}
			if skip {
				continue
			}
			// 规范化路径并检查
			cleanPath := part
			if !filepath.IsAbs(cleanPath) {
				cleanPath = filepath.Join(workspaceRoot, cleanPath)
			}
			if _, err := os.Stat(cleanPath); os.IsNotExist(err) {
				seen[part] = true
				refs = append(refs, part)
				if len(refs) >= 10 {
					return refs
				}
			}
		}
	}
	return refs
}

// buildCodeGraphStats 获取代码图谱统计信息。
func buildCodeGraphStats() string {
	root := WorkspaceRoots[0]
	if root == "" {
		return ""
	}
	g, err := getCodeGraph(root)
	if err != nil || g == nil {
		return ""
	}
	stats := g.Stats()
	if stats.FileCount == 0 && stats.EntityCount == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("代码图谱已覆盖 %d 个文件、%d 个包。\n", stats.FileCount, stats.PackageCount))
	if stats.EntityCount > 0 {
		b.WriteString(fmt.Sprintf("实体总数：%d", stats.EntityCount))
		// 按类别展开
		if len(stats.KindCounts) > 0 {
			parts := make([]string, 0, len(stats.KindCounts))
			for k, v := range stats.KindCounts {
				parts = append(parts, fmt.Sprintf("%s %d", k, v))
			}
			b.WriteString("（" + strings.Join(parts, "、") + "）")
		}
		b.WriteString("\n")
	}
	if stats.RelationCount > 0 {
		b.WriteString(fmt.Sprintf("关系总数：%d。\n", stats.RelationCount))
	}
	b.WriteString("如果项目文件有变更，用 codegraph_build 重建图谱获取最新结构。")
	return b.String()
}

// ── 注入入口 ────────────────────────────────────────────────
// BuildResumeContext 为对话恢复构建主动注入的上下文块。
// 应在 buildWebSystemDynamic() 中调用，作为系统提示的动态后缀。
// 仅当对话有历史（非首次对话）时返回非空。
func BuildResumeContext(convID, currentTask string, history []Message, store MessageStoreReader, roots []string) string {
	// ★ TTL 缓存：同一对话（convID+roots）在 TTL 内复用输出，
	//   避免每轮新对话重建 system 动态后缀导致 KV 缓存前缀从尾部频繁断裂。
	cacheKey := fmt.Sprintf("%s|%s", convID, strings.Join(roots, "|"))
	resumeCtxCacheMu.Lock()
	if resumeCtxCacheKey == cacheKey && time.Since(resumeCtxCacheAt) < resumeCtxCacheTTL {
		val := resumeCtxCacheVal
		resumeCtxCacheMu.Unlock()
		return val
	}
	resumeCtxCacheMu.Unlock()

	var result string
	if len(history) == 0 {
		// 新对话：注入工作区结构概览 + 自动召回记忆 + Git 状态 + 代码图谱 + 构建状态
		var b strings.Builder
		if ws := buildWorkspaceStructureOverview(roots); ws != "" {
			b.WriteString("\n\n# 工作区结构\n")
			b.WriteString(ws)
		}
		if recall := buildRecallContext(roots, currentTask); recall != "" {
			b.WriteString("\n\n# 相关项目记忆（自动召回）\n")
			b.WriteString(recall)
			b.WriteString("\n（需要细节用 memory_read 读全文。）")
		}
		if gs := buildGitStatus(roots); gs != "" {
			b.WriteString("\n\n# Git 状态\n")
			b.WriteString(gs)
		}
		if cgs := buildCodeGraphStats(); cgs != "" {
			b.WriteString("\n\n# 代码图谱\n")
			b.WriteString(cgs)
		}
		if bs := buildBuildStatus(roots); bs != "" {
			b.WriteString("\n\n# 构建状态\n")
			b.WriteString(bs)
		}
		if kb := buildKBStaleness(roots); kb != "" {
			b.WriteString("\n\n# ⚠️ 知识库过期警告\n")
			b.WriteString(kb)
		}
		result = b.String()
	} else {
		// 有历史：完整注入会话连贯性上下文
		sc := BuildSessionContext(convID, roots, currentTask, history, store)
		result = sc.FormatForInjection()
	}

	// 写缓存
	resumeCtxCacheMu.Lock()
	resumeCtxCacheKey = cacheKey
	resumeCtxCacheVal = result
	resumeCtxCacheAt = time.Now()
	resumeCtxCacheMu.Unlock()
	return result
}

// ── 类型定义 ────────────────────────────────────────────────

// MessageStoreReader 消息存储只读接口（避免循环依赖）。
type MessageStoreReader interface {
	GetConversation(convID string) (*ConversationMeta, error)
	LoadAll(convID string) ([]Message, error)
}

// formatTimeAgo 格式化相对时间。
func formatTimeAgo(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "刚刚"
	case d < time.Hour:
		return fmt.Sprintf("%d 分钟前", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%d 小时前", int(d.Hours()))
	default:
		return fmt.Sprintf("%d 天前", int(d.Hours()/24))
	}
}
