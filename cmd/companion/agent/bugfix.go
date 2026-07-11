package agent

// bugfix.go — 自动 BUG 修复循环（Analyze → Fix → Verify）。
// 集成到编排循环中：检测到构建/测试失败后，自动分析错误、定位代码、
// 生成修复提示、注入 agent 修复、重新验证。
// 支持文件级回滚（修复前备份修改过的文件）。

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"context"
)

// ── 类型定义 ───────────────────────────────────────────────

// FixAttemptStatus 修复尝试状态。
type FixAttemptStatus string

const (
	FixAttemptPending   FixAttemptStatus = "pending"    // 待执行
	FixAttemptRunning   FixAttemptStatus = "running"    // 执行中
	FixAttemptFixed     FixAttemptStatus = "fixed"      // 修复成功
	FixAttemptFailed    FixAttemptStatus = "failed"     // 修复失败
	FixAttemptRollback  FixAttemptStatus = "rollback"   // 已回滚
)

// BugFixResult 一次完整的 BUG 修复闭环结果。
type BugFixResult struct {
	Detected      *BugDetectResult  `json:"detected"`      // 检测结果
	Fixed         bool              `json:"fixed"`          // 是否修复成功
	FixedCount    int               `json:"fixedCount"`     // 修复的问题数
	Remaining     int               `json:"remaining"`      // 剩余问题数
	Attempts      int               `json:"attempts"`       // 修复尝试次数
	RolledBack    bool              `json:"rolledBack"`     // 是否已回滚
	FixSummary    string            `json:"fixSummary"`     // 修复摘要
	AgentTask     string            `json:"agentTask"`      // 生成的 agent 修复任务文本
	FinalDetect   *BugDetectResult  `json:"finalDetect"`    // 修复后的检测结果
	Duration      time.Duration     `json:"duration"`       // 总耗时
	backupDir     string            `json:"-"`              // 备份目录（内部使用，不序列化）
}

// FixRecord 修复记录（持久化到 .pair/tasks/fix-records/）。
type FixRecord struct {
	ID           string        `json:"id"`
	Time         string        `json:"time"`
	ConvID       string        `json:"convId,omitempty"`
	InitialState *BugDetectResult `json:"initialState"`
	FinalState   *BugDetectResult `json:"finalState"`
	Fixed        bool          `json:"fixed"`
	Attempts     int           `json:"attempts"`
	RolledBack   bool          `json:"rolledBack"`
	Summary      string        `json:"summary"`
	FilesChanged []string      `json:"filesChanged,omitempty"`
}

// ── 核心函数 ───────────────────────────────────────────────

// AutoDetectAndFix 自动检测并修复项目 BUG。
// 执行一次完整的：Detect → Analyze → Fix → Verify 循环。
// root 为项目根目录。
// maxAttempts 为最大修复尝试次数（默认 3）。
// fileBackups 为需要备份的文件列表（修复前备份，失败后恢复）。
//
// 返回修复结果，其中包含检测到的错误、生成的修复任务文本和最终验证结果。
// 注意：此函数不会直接修改文件（由 agent 通过工具修改），
// 但会备份 filesBackup 指定的文件以备回滚。
func AutoDetectAndFix(root string, maxAttempts int, convID string) *BugFixResult {
	start := time.Now()
	result := &BugFixResult{
		Attempts: 0,
	}

	if maxAttempts <= 0 {
		maxAttempts = 3
	}

	// ── Step 1：Detect ──
	result.Detected = DetectProjectErrors(root)
	if result.Detected.Success {
		result.Fixed = true
		result.FixSummary = "没有检测到错误"
		result.Duration = time.Since(start)
		return result
	}

	// ── Step 2：分析并生成修复任务 ──
	result.AgentTask = BuildDetailedFixPrompt(result.Detected, root)

	// ── Step 3：修复前备份 ──
	result.backupDir = createBackup(root, result.Detected)

	// 标记尚未验证
	result.Attempts = 1

	// ── Step 4：修复由调用方注入下一轮 loop ──

	// 记录修复记录
	saveFixRecord(root, result, convID)

	result.Duration = time.Since(start)
	return result
}

// VerifyAfterFix 修复后验证：再次运行检测，检查是否仍有错误。
// 如果修复后仍有错误，可继续调用此函数（递归的 verify 步骤）。
// 返回更新后的 BugFixResult。
func VerifyAfterFix(root string, result *BugFixResult) *BugFixResult {
	// 重新检测
	result.FinalDetect = DetectProjectErrors(root)

	if result.FinalDetect.Success {
		result.Fixed = true
		result.Remaining = 0
		result.FixedCount = result.Detected.ErrorCount
		result.FixSummary = fmt.Sprintf("✅ 全部修复成功（共 %d 个问题）", result.FixedCount)
		result.RolledBack = false
	} else {
		result.Remaining = result.FinalDetect.ErrorCount
		result.FixedCount = result.Detected.ErrorCount - result.Remaining

		// 如果修复后问题减少但未清零，标记部分修复
		if result.FixedCount > 0 {
			result.FixSummary = fmt.Sprintf("部分修复：已修复 %d 个，仍有 %d 个问题。",
				result.FixedCount, result.Remaining)
		}

		// 如果修复次数超过限制，回滚
		if result.Attempts >= 3 {
			if result.backupDir != "" {
				rollbackFiles(result.backupDir, root)
				result.RolledBack = true
				result.FixSummary = fmt.Sprintf("❌ 修复失败（尝试 %d 次），已回滚。剩余 %d 个问题。",
					result.Attempts, result.Remaining)
			} else {
				result.FixSummary = fmt.Sprintf("❌ 修复失败（尝试 %d 次），无备份无法回滚。剩余 %d 个问题。",
					result.Attempts, result.Remaining)
			}
		} else {
			result.FixSummary = fmt.Sprintf("修复中（尝试 %d/%d）：已修复 %d 个，仍有 %d 个问题。",
				result.Attempts, 3, result.FixedCount, result.Remaining)
		}
	}

	return result
}

// BuildDetailedFixPrompt 构建详细的修复提示（含错误位置、代码上下文和修复建议）。
// 此 Prompt 直接注入到 agent 的任务中，指导 agent 逐个修复。
func BuildDetailedFixPrompt(detected *BugDetectResult, root string) string {
	if detected == nil || detected.Success {
		return ""
	}

	var b strings.Builder
	b.WriteString("# 自动检测到项目 BUG\n\n")
	b.WriteString("项目构建/测试失败，请分析并修复以下问题。\n\n")

	// 错误摘要
	b.WriteString("## 错误摘要\n")
	b.WriteString(detected.Summary)
	b.WriteString("\n\n")

	// 详细错误列表
	b.WriteString("## 详细错误列表\n\n")

	for i, symptom := range detected.Symptoms {
		b.WriteString(fmt.Sprintf("### 错误 %d: %s\n\n", i+1, symptom.Message))
		if symptom.Location.File != "" {
			b.WriteString(fmt.Sprintf("- **文件**: `%s`\n", symptom.Location.File))
			b.WriteString(fmt.Sprintf("- **行号**: %d\n", symptom.Location.Line))
			if symptom.Location.Column > 0 {
				b.WriteString(fmt.Sprintf("- **列号**: %d\n", symptom.Location.Column))
			}
		}
		b.WriteString(fmt.Sprintf("- **类型**: %s\n", symptom.Type))
		b.WriteString(fmt.Sprintf("- **严重程度**: %s\n", symptom.Severity))

		// 代码上下文
		if symptom.Context != "" {
			b.WriteString("\n**代码上下文**:\n```\n")
			b.WriteString(symptom.Context)
			b.WriteString("```\n")
		}

		b.WriteString("\n")
	}

	// 修复指南
	b.WriteString("## 修复指南\n\n")
	b.WriteString("1. 使用 `read_file` 读取每个错误位置附近的代码\n")
	b.WriteString("2. 分析错误原因：是语法错误、类型不匹配、未定义标识符还是其他问题\n")
	b.WriteString("3. 使用 `edit_file` 或 `write_file` 修复\n")
	b.WriteString("4. 修改后使用 `go_build` 验证是否通过\n")
	b.WriteString("5. 如果仍有错误，继续修复\n")
	b.WriteString("6. 所有错误修复完成后，运行 `go_build` 确认全部通过，然后调用 `finish_task`\n\n")

	b.WriteString("请开始修复。")

	return b.String()
}

// ── 备份与回滚 ─────────────────────────────────────────────

// createBackup 备份检测到的错误所涉及的文件。
// 返回备份目录路径（空串=未备份）。
func createBackup(root string, result *BugDetectResult) string {
	if root == "" || result == nil {
		return ""
	}

	// 收集需要备份的文件
	backupFiles := make(map[string]bool)
	for _, symptom := range result.Symptoms {
		if symptom.Location.File != "" {
			absPath := symptom.Location.File
			if !filepath.IsAbs(absPath) {
				absPath = filepath.Join(root, absPath)
			}
			if _, err := os.Stat(absPath); err == nil {
				backupFiles[absPath] = true
			}
		}
	}

	if len(backupFiles) == 0 {
		return ""
	}

	// 创建备份目录
	backupDir := filepath.Join(root, ".pair", "tasks", "fix-backups",
		fmt.Sprintf("backup_%s", time.Now().Format("20060102_150405")))
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return ""
	}

	// 备份每个文件
	for f := range backupFiles {
		relPath, err := filepath.Rel(root, f)
		if err != nil {
			continue
		}
		destPath := filepath.Join(backupDir, relPath)
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			continue
		}
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		os.WriteFile(destPath, data, 0644)
	}

	return backupDir
}

// rollbackFiles 从备份目录恢复所有文件。
func rollbackFiles(backupDir string, root string) {
	if backupDir == "" || root == "" {
		return
	}

	filepath.Walk(backupDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(backupDir, path)
		if err != nil {
			return nil
		}
		destPath := filepath.Join(root, rel)
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		os.WriteFile(destPath, data, 0644)
		return nil
	})
}

// ── 记录持久化 ─────────────────────────────────────────────

// saveFixRecord 保存修复记录到 .pair/tasks/fix-records/。
func saveFixRecord(root string, result *BugFixResult, convID string) {
	if root == "" || result == nil {
		return
	}

	dir := filepath.Join(root, ".pair", "tasks", "fix-records")
	os.MkdirAll(dir, 0755)

	record := &FixRecord{
		ID:           fmt.Sprintf("fix_%s", time.Now().Format("20060102_150405")),
		Time:         time.Now().Format("2006-01-02 15:04:05"),
		ConvID:       convID,
		InitialState: result.Detected,
		FinalState:   result.FinalDetect,
		Fixed:        result.Fixed,
		Attempts:     result.Attempts,
		RolledBack:   result.RolledBack,
		Summary:      result.FixSummary,
	}

	data, _ := json.MarshalIndent(record, "", "  ")
	os.WriteFile(filepath.Join(dir, record.ID+".json"), data, 0644)
}

// ── 便捷函数 ───────────────────────────────────────────────

// IsBugFixTask 检查任务文本是否包含 BUG 修复指令。
func IsBugFixTask(task string) bool {
	lower := strings.ToLower(task)
	return strings.Contains(lower, "自动检测到项目 bug") ||
		strings.Contains(lower, "构建失败") ||
		strings.Contains(lower, "需要修复")
}

// FormatBuildErrorForAgent 将构建错误格式化为 agent 可操作的修复任务。
// 格式化内容：原始输出 + 解析后的错误列表 + 每个错误的代码位置与上下文。
func FormatBuildErrorForAgent(buildOutput string, root string) string {
	// 先解析
	symptoms := AnalyzeBuildOutput(buildOutput, root)
	if len(symptoms) == 0 {
		// 尝试测试输出
		symptoms = AnalyzeTestOutput(buildOutput, root)
	}
	if len(symptoms) == 0 {
		// 直接返回原始输出
		return fmt.Sprintf("项目构建/验证失败，需要修复:\n\n%s", buildOutput)
	}

	result := &BugDetectResult{
		Success:     false,
		ErrorCount:  len(symptoms),
		Symptoms:    symptoms,
		BuildOutput: buildOutput,
	}

	return BuildDetailedFixPrompt(result, root)
}

// ── 工具注册 ───────────────────────────────────────────────

func registerBugFixTools(r *Registry, root string) {
	// bug_detect — 全量项目 BUG 检测
	r.Register(&Tool{
		Name: "bug_detect",
		Description: "全量检测项目中是否存在 BUG。自动运行 go vet → go build → go test，" +
			"输出解析后的错误列表（含文件路径、行号、错误消息和代码上下文）。" +
			"用于自动发现编译/测试/运行时的 BUG。集成在自主模式的编排循环中。",
		Parameters: objSchema(props{},
		),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			result := DetectProjectErrors(root)
			if result.Success {
				return "✅ 项目检测通过，未发现错误。", nil
			}
			return result.Summary, nil
		},
	})

	// bug_fix — 生成 BUG 修复任务（检测 + 生成修复提示）
	r.Register(&Tool{
		Name: "bug_fix",
		Description: "自动检测项目 BUG（编译/测试/运行时错误），生成详细的修复任务文本。" +
			"返回包含错误位置、代码上下文和修复指南的完整修复任务。" +
			"可用于自主模式中在 loop 之间自动检测并修复项目问题。",
		Parameters: objSchema(props{
			"max_attempts": intProp("可选：最大修复尝试次数，默认 3"),
		}),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			result := AutoDetectAndFix(root, argInt(args, "max_attempts", 3), "")
			if result.Fixed {
				return "✅ 项目检测通过，无需修复。", nil
			}
			return result.AgentTask, nil
		},
	})
}

// ── 嵌入到 RegisterDefaultTools ──────────────────────────

// RegisterBugTools 注册所有 BUG 检测与修复工具。
// 由 RegisterDefaultTools 调用。
func RegisterBugTools(r *Registry, root string) {
	registerBugDetectTools(r, root)
	registerBugFixTools(r, root)
}

// ── 辅助 ──────────────────────────────────────────────────

