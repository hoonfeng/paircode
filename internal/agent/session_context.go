// session_context.go — 会话连贯性上下文构建器
// 核心思想：主动注入 > 被动工具。Agent 不应依赖自己"想起来"去调用 memory_* 工具，
// 而应在每次对话恢复时，由系统主动注入：任务进度、对话摘要、相关记忆、项目归属、工作区结构。
//
// 注入点：buildWebSystemDynamic() 和 Loop.Run() 的初始消息。

package agent

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
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
	RecentFileEdits        []string // 最近编辑的文件列表
	GitStatus              string   // Git 状态快照（最近提交 + 未提交改动）
	CodeGraphStats         string   // 代码图谱统计（实体数/覆盖率）
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
func extractRecentEdits(history []Message, roots []string) []string {
	seen := make(map[string]bool)
	var files []string

	// 从最近的助手消息中提取文件操作
	for i := len(history) - 1; i >= 0 && len(files) < 10; i-- {
		msg := history[i]
		if msg.Role != RoleAssistant {
			// 也检查工具结果消息
			if msg.Role == RoleTool && msg.Name != "" {
				// RoleTool 消息的 Name 字段可能包含工具名
			}
			continue
		}

		// 从 Content 中提取文件路径模式
		content := msg.Content
		for _, root := range roots {
			// 简单模式：找工作区内路径
			lines := strings.Split(content, "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				// 检查是否包含文件路径特征
				if strings.Contains(line, root) || strings.Contains(line, ".go") ||
					strings.Contains(line, ".ts") || strings.Contains(line, ".vue") ||
					strings.Contains(line, ".md") {
					// 提取可能的文件路径
					parts := strings.Fields(line)
					for _, part := range parts {
						part = strings.Trim(part, "`\"'[](),")
						if isFilePathLike(part) && !seen[part] {
							seen[part] = true
							files = append(files, part)
							if len(files) >= 10 {
								break
							}
						}
					}
				}
				if len(files) >= 10 {
					break
				}
			}
			if len(files) >= 10 {
				break
			}
		}
	}
	return files
}

// isFilePathLike 判断字符串是否像文件路径。
func isFilePathLike(s string) bool {
	if len(s) < 3 || len(s) > 200 {
		return false
	}
	// 包含常见代码文件扩展名
	codeExts := []string{".go", ".ts", ".js", ".vue", ".py", ".rs", ".java", ".cpp", ".c", ".h",
		".md", ".json", ".yaml", ".yml", ".toml", ".xml", ".html", ".css", ".scss", ".sql"}
	for _, ext := range codeExts {
		if strings.HasSuffix(strings.ToLower(s), ext) {
			return true
		}
	}
	// 包含路径分隔符
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

// buildCodeGraphStats 获取代码图谱统计信息。
func buildCodeGraphStats() string {
	cgGraphMu.RLock()
	g := cgGraph
	cgGraphMu.RUnlock()
	if g == nil {
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
	if len(history) == 0 {
		// 新对话：注入工作区结构概览 + 自动召回记忆 + Git 状态 + 代码图谱
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
		return b.String()
	}

	// 有历史：完整注入会话连贯性上下文
	sc := BuildSessionContext(convID, roots, currentTask, history, store)
	return sc.FormatForInjection()
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
