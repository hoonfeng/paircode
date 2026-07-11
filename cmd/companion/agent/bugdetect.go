package agent

// bugdetect.go — 项目 BUG 自动检测引擎。
// 自动编译/测试/运行项目，检测并分析错误，提取文件位置和代码上下文。
// 核心能力：Detect → Analyze → Locate，为 Autofix 提供输入。

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ── 类型定义 ───────────────────────────────────────────────

// BugSeverity 错误严重级别。
type BugSeverity string

const (
	BugSeverityError   BugSeverity = "error"   // 编译错误、运行时 panic
	BugSeverityWarning BugSeverity = "warning" // 编译警告、lint 警告
	BugSeverityInfo    BugSeverity = "info"    // 建议改进
)

// BugType 错误类型。
type BugType string

const (
	BugTypeCompile  BugType = "compile"   // 编译错误
	BugTypeTest     BugType = "test"      // 测试失败
	BugTypePanic    BugType = "panic"     // 运行时 panic
	BugTypeLint     BugType = "lint"      // lint 警告
	BugTypeGoVet    BugType = "govet"     // go vet 问题
	BugTypeRuntime  BugType = "runtime"   // 运行时错误（非 panic）
	BugTypeUnknown  BugType = "unknown"   // 未知类型
)

// BugLocation 错误代码位置。
type BugLocation struct {
	File     string `json:"file"`     // 相对工作区的文件路径
	Line     int    `json:"line"`     // 行号（1 基）
	Column   int    `json:"column"`   // 列号（0=未指定）
	Function string `json:"function,omitempty"` // 函数名（从堆栈提取）
}

// BugSymptom 单条错误症状。
type BugSymptom struct {
	Type      BugType     `json:"type"`      // 错误类型
	Severity  BugSeverity `json:"severity"`  // 严重级别
	Message   string      `json:"message"`   // 错误消息全文
	Location  BugLocation `json:"location"`  // 代码位置
	Context   string      `json:"context,omitempty"`  // 错误附近的代码上下文（5 行前后）
	Suggestion string     `json:"suggestion,omitempty"` // 可能的修复建议（可选）
}

// BugDetectResult 检测结果。
type BugDetectResult struct {
	Success    bool          `json:"success"`    // 是否全部通过
	ErrorCount int           `json:"errorCount"` // 错误总数
	Symptoms   []BugSymptom  `json:"symptoms"`   // 所有检测到的症状
	BuildOutput string       `json:"buildOutput"` // 原始构建输出
	Duration   time.Duration `json:"duration"`   // 检测耗时
	Summary    string        `json:"summary"`    // 人类可读摘要
}

// ── 正则表达式 ─────────────────────────────────────────────

// Go 编译错误：file:line:col: message
// 或 file:line: message
var goCompileRe = regexp.MustCompile(`^(.+?\.go):(\d+)(?::(\d+))?:\s*(.+)$`)

// Go panic 堆栈：goroutine X [running]: 后的
// file(line) 和 file(line, +offset)
var goPanicFileRe = regexp.MustCompile(`^\s*(.+\.go):(\d+)(?:\s+\+0x[0-9a-f]+)?\s*$`)

// Go test FAIL 行：--- FAIL: TestName (秒)
var goTestFailRe = regexp.MustCompile(`^---\s+FAIL:\s+(.+)\s+\(`)

// Go test 函数调用栈：实际代码文件行
var goTestStackRe = regexp.MustCompile(`^\s*(.+\.go):(\d+)\s+\(0x[0-9a-f]+\)`)

// Go vet 警告
var goVetRe = regexp.MustCompile(`^(.+\.go):(\d+):(\d+)?:\s*(.+)$`)

// ── 核心函数 ───────────────────────────────────────────────

// DetectProjectErrors 综合检测项目错误。
// 按优先级顺序：go vet → go build → go test（build 失败则跳过 test）。
// 返回所有检测到的错误症状。
// root 为项目根目录（含 go.mod 的项目）。
func DetectProjectErrors(root string) *BugDetectResult {
	start := time.Now()
	result := &BugDetectResult{
		Success:  true,
		Symptoms: make([]BugSymptom, 0),
	}

	if root == "" {
		result.Summary = "（无工作区，跳过检测）"
		result.Duration = time.Since(start)
		return result
	}

	// 检测是否为 Go 项目
	goModPath := filepath.Join(root, "go.mod")
	if _, err := os.Stat(goModPath); err != nil {
		result.Summary = "（非 Go 项目，跳过检测）"
		result.Duration = time.Since(start)
		return result
	}

	// 1. go vet（轻量静态分析）
	vetOutput := runGoVet(root)
	if vetOutput != "" {
		symptoms := AnalyzeVetOutput(vetOutput, root)
		result.Symptoms = append(result.Symptoms, symptoms...)
	}

	// 2. go build（完整编译）
	buildOutput := runGoBuild(root)
	result.BuildOutput = buildOutput
	if buildOutput != "" {
		symptoms := AnalyzeBuildOutput(buildOutput, root)
		result.Symptoms = append(result.Symptoms, symptoms...)
		result.Summary = buildSummary(result)
		result.Duration = time.Since(start)
		return result // ⚡ build 失败则短路，跳过 test（编译错误下的测试毫无意义）
	}

	// 3. go test（仅编译通过后运行，范围收窄到核心包）
	testOutput := runGoTest(root)
	if testOutput != "" {
		symptoms := AnalyzeTestOutput(testOutput, root)
		result.Symptoms = append(result.Symptoms, symptoms...)
	}

	// 更新结果状态
	result.ErrorCount = len(result.Symptoms)
	result.Success = result.ErrorCount == 0
	result.Duration = time.Since(start)
	result.Summary = buildSummary(result)

	return result
}

// AnalyzeBuildOutput 解析 go build 输出，提取所有错误症状。
// buildOutput 为完整的 go build 输出文本。
// root 为项目根目录。
func AnalyzeBuildOutput(buildOutput string, root string) []BugSymptom {
	if buildOutput == "" {
		return nil
	}

	var symptoms []BugSymptom
	scanner := bufio.NewScanner(strings.NewReader(buildOutput))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// 匹配 file:line:col: message 或 file:line: message 格式
		matches := goCompileRe.FindStringSubmatch(line)

		if matches != nil {
			file := matches[1]
			lineNum, _ := strconv.Atoi(matches[2])
			col := 0
			if len(matches) > 3 && matches[3] != "" {
				col, _ = strconv.Atoi(matches[3])
			}
			msg := matches[len(matches)-1]

			// 规范化路径
			relFile := normalizePath(file, root)

			// 获取代码上下文
			ctxLines := extractErrorContext(file, lineNum, 5)

			symptom := BugSymptom{
				Type:     BugTypeCompile,
				Severity: BugSeverityError,
				Message:  msg,
				Location: BugLocation{
					File:   relFile,
					Line:   lineNum,
					Column: col,
				},
				Context: ctxLines,
			}
			symptoms = append(symptoms, symptom)
		} else {
			// 未匹配到文件格式，作为附加信息
			// 检查是否包含 build 失败关键字
			lower := strings.ToLower(line)
			if strings.Contains(lower, "error") || strings.Contains(lower, "cannot") ||
				strings.Contains(lower, "undefined") || strings.Contains(lower, "unexpected") {
				symptoms = append(symptoms, BugSymptom{
					Type:     BugTypeCompile,
					Severity: BugSeverityError,
					Message:  line,
				})
			}
		}
	}

	return symptoms
}

// AnalyzeTestOutput 解析 go test -v 输出，提取失败测试和堆栈信息。
func AnalyzeTestOutput(testOutput string, root string) []BugSymptom {
	if testOutput == "" {
		return nil
	}

	var symptoms []BugSymptom
	scanner := bufio.NewScanner(strings.NewReader(testOutput))
	var currentTest string
	var currentStack []string
	inStack := false
	seenTests := make(map[string]bool)

	for scanner.Scan() {
		line := scanner.Text()

		// 检测测试失败头部
		if matches := goTestFailRe.FindStringSubmatch(line); matches != nil {
			testName := matches[1]
			// 去重
			if seenTests[testName] {
				continue
			}
			seenTests[testName] = true
			currentTest = testName
			currentStack = nil
			inStack = true
			continue
		}

		// 如果在堆栈跟踪中
		if inStack {
			// 检测 panic
			if strings.HasPrefix(line, "panic:") || strings.HasPrefix(line, "fatal error:") {
				msg := strings.TrimSpace(line)
				if currentTest != "" {
					msg = fmt.Sprintf("测试 %s 中 %s", currentTest, msg)
				}
				symptom := BugSymptom{
					Type:     BugTypePanic,
					Severity: BugSeverityError,
					Message:  msg,
				}

				// 从堆栈提取文件位置
				if loc, found := extractLocationFromStack(currentStack, root); found {
					symptom.Location = loc
					ctxLines := extractErrorContext(
						filepath.Join(root, loc.File),
						loc.Line, 5,
					)
					symptom.Context = ctxLines
				}

				symptoms = append(symptoms, symptom)
				inStack = false
				continue
			}

			// 检测代码文件引用（堆栈行）
			if matches := goTestStackRe.FindStringSubmatch(line); matches != nil {
				file := matches[1]
				lineNum, _ := strconv.Atoi(matches[2])
				relFile := normalizePath(file, root)

				symptom := BugSymptom{
					Type:     BugTypeTest,
					Severity: BugSeverityError,
					Message:  fmt.Sprintf("测试失败: %s", currentTest),
					Location: BugLocation{
						File: relFile,
						Line: lineNum,
					},
				}
				ctxLines := extractErrorContext(file, lineNum, 5)
				symptom.Context = ctxLines

				symptoms = append(symptoms, symptom)
				currentStack = append(currentStack, line)
				inStack = false
				continue
			}

			// 空行可能表示堆栈结束
			if strings.TrimSpace(line) == "" && len(currentStack) > 0 {
				inStack = false
			} else {
				currentStack = append(currentStack, line)
			}
		}
	}

	// 如果没有提取到具体症状，但测试输出包含 FAIL
	if len(symptoms) == 0 && (strings.Contains(testOutput, "FAIL") || strings.Contains(testOutput, "panic")) {
		symptoms = append(symptoms, BugSymptom{
			Type:     BugTypeTest,
			Severity: BugSeverityError,
			Message:  "测试失败（原始输出见 BuildOutput）",
		})
	}

	return symptoms
}

// AnalyzeGoRunOutput 解析 go run 运行时输出，提取 panic 和错误堆栈。
func AnalyzeGoRunOutput(runOutput string, root string) []BugSymptom {
	if runOutput == "" {
		return nil
	}

	var symptoms []BugSymptom
	scanner := bufio.NewScanner(strings.NewReader(runOutput))
	var currentStack []string
	inPanic := false
	panicMsg := ""

	for scanner.Scan() {
		line := scanner.Text()

		// 检测 panic 起始
		if strings.HasPrefix(line, "panic:") || strings.HasPrefix(line, "fatal error:") {
			panicMsg = strings.TrimSpace(line)
			inPanic = true
			currentStack = nil
			continue
		}

		// 检测 goroutine 头部
		if inPanic && strings.HasPrefix(line, "goroutine ") {
			currentStack = append(currentStack, line)
			continue
		}

		// 检测 panic 堆栈中的文件引用
		if inPanic {
			if matches := goPanicFileRe.FindStringSubmatch(line); matches != nil {
				file := matches[1]
				lineNum, _ := strconv.Atoi(matches[2])
				relFile := normalizePath(file, root)

				symptom := BugSymptom{
					Type:     BugTypePanic,
					Severity: BugSeverityError,
					Message:  panicMsg,
					Location: BugLocation{
						File: relFile,
						Line: lineNum,
					},
				}
				ctxLines := extractErrorContext(file, lineNum, 5)
				symptom.Context = ctxLines

				symptoms = append(symptoms, symptom)
				currentStack = append(currentStack, line)
			} else if strings.HasPrefix(line, "\t") {
				currentStack = append(currentStack, line)
			} else if strings.TrimSpace(line) == "" {
				// 空行可能结束堆栈
				inPanic = false
			}
		}
	}

	return symptoms
}

// ExtractErrorContext 根据文件路径和行号，读取错误附近的代码上下文。
// contextLines 为前后行数。
func extractErrorContext(filePath string, lineNum int, contextLines int) string {
	if filePath == "" || lineNum <= 0 {
		return ""
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return ""
	}

	lines := strings.Split(string(data), "\n")
	totalLines := len(lines)

	start := lineNum - contextLines - 1
	if start < 0 {
		start = 0
	}
	end := lineNum + contextLines
	if end > totalLines {
		end = totalLines
	}

	var b strings.Builder
	for i := start; i < end; i++ {
		marker := " "
		if i+1 == lineNum {
			marker = "→"
		}
		b.WriteString(fmt.Sprintf("  %s %4d: %s\n", marker, i+1, lines[i]))
	}

	return b.String()
}

// ── 内部辅助 ───────────────────────────────────────────────

// runGoVet 执行 go vet -tags webonly ./cmd/companion，返回输出（空=成功）。
func runGoVet(root string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "vet", "-tags", "webonly", "./cmd/companion")
	cmd.Dir = root
	out, _ := cmd.CombinedOutput()
	return strings.TrimSpace(string(out))
}

// runGoBuild 执行 go build -tags webonly ./cmd/companion，返回输出（空=成功）。
func runGoBuild(root string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "build", "-tags", "webonly", "./cmd/companion")
	cmd.Dir = root
	out, _ := cmd.CombinedOutput()
	return strings.TrimSpace(string(out))
}

// runGoTest 执行 go test ./cmd/companion/agent，返回输出（空=成功）。
func runGoTest(root string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "test", "-count=1", "-timeout", "30s", "./cmd/companion/agent")
	cmd.Dir = root
	out, _ := cmd.CombinedOutput()
	output := strings.TrimSpace(string(out))
	// 仅当有 FAIL 或 panic 时才返回输出（只关注失败）
	if strings.Contains(output, "FAIL") || strings.Contains(output, "panic:") {
		return output
	}
	return ""
}

// normalizePath 将绝对路径规范化为项目相对路径。
func normalizePath(absPath string, root string) string {
	if root == "" {
		return absPath
	}
	// 如果是绝对路径，转相对
	if filepath.IsAbs(absPath) {
		rel, err := filepath.Rel(root, absPath)
		if err == nil {
			return filepath.ToSlash(rel)
		}
	}
	return filepath.ToSlash(absPath)
}

// extractLocationFromStack 从堆栈跟踪中提取首个文件位置。
func extractLocationFromStack(stack []string, root string) (BugLocation, bool) {
	for _, line := range stack {
		if matches := goPanicFileRe.FindStringSubmatch(line); matches != nil {
			file := matches[1]
			lineNum, _ := strconv.Atoi(matches[2])
			return BugLocation{
				File: normalizePath(file, root),
				Line: lineNum,
			}, true
		}
	}
	return BugLocation{}, false
}

// buildSummary 构建人类可读的检测摘要。
func buildSummary(result *BugDetectResult) string {
	if result.Success {
		return "✅ 全部通过：编译、测试均无错误"
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("❌ 发现 %d 个问题:\n\n", result.ErrorCount))

	// 按类型分组统计
	typeCount := make(map[BugType]int)
	for _, s := range result.Symptoms {
		typeCount[s.Type]++
	}

	for t, count := range typeCount {
		switch t {
		case BugTypeCompile:
			b.WriteString(fmt.Sprintf("  - 编译错误: %d 个\n", count))
		case BugTypeTest:
			b.WriteString(fmt.Sprintf("  - 测试失败: %d 个\n", count))
		case BugTypePanic:
			b.WriteString(fmt.Sprintf("  - 运行时 panic: %d 个\n", count))
		case BugTypeGoVet:
			b.WriteString(fmt.Sprintf("  - go vet 警告: %d 个\n", count))
		default:
			b.WriteString(fmt.Sprintf("  - 其他问题: %d 个\n", count))
		}
	}

	// 列出具体位置（最多 5 条）
	shown := 0
	for _, s := range result.Symptoms {
		if shown >= 5 {
			b.WriteString(fmt.Sprintf("  ... 还有 %d 个问题\n", result.ErrorCount-shown))
			break
		}
		if s.Location.File != "" {
			b.WriteString(fmt.Sprintf("  - %s:%d: %s\n", s.Location.File, s.Location.Line, truncateString(s.Message, 80)))
		} else {
			b.WriteString(fmt.Sprintf("  - %s\n", truncateString(s.Message, 80)))
		}
		shown++
	}

	return b.String()
}

// truncateString 截断字符串到最大长度。
func truncateString(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "..."
}

// AnalyzeVetOutput 解析 go vet 输出。
func AnalyzeVetOutput(vetOutput string, root string) []BugSymptom {
	if vetOutput == "" {
		return nil
	}

	var symptoms []BugSymptom
	scanner := bufio.NewScanner(strings.NewReader(vetOutput))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// go vet 输出格式：file:line:col: message
		// 或 file:line: message
		matches := goVetRe.FindStringSubmatch(line)
		if matches != nil {
			file := matches[1]
			lineNum, _ := strconv.Atoi(matches[2])
			col := 0
			if len(matches) > 3 && matches[3] != "" {
				col, _ = strconv.Atoi(matches[3])
			}
			msg := matches[len(matches)-1]

			relFile := normalizePath(file, root)
			ctxLines := extractErrorContext(file, lineNum, 3)

			symptom := BugSymptom{
				Type:     BugTypeGoVet,
				Severity: BugSeverityWarning,
				Message:  msg,
				Location: BugLocation{
					File:   relFile,
					Line:   lineNum,
					Column: col,
				},
				Context: ctxLines,
			}
			symptoms = append(symptoms, symptom)
		}
	}

	return symptoms
}

// ── 便捷函数 ───────────────────────────────────────────────

// BugSymptomToMessage 将错误症状转换为系统提示消息（用于注入 agent 任务）。
func BugSymptomToMessage(symptom BugSymptom) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("【%s】%s\n", symptom.Type, symptom.Message))
	if symptom.Location.File != "" {
		b.WriteString(fmt.Sprintf("位置: %s:%d", symptom.Location.File, symptom.Location.Line))
		if symptom.Location.Column > 0 {
			b.WriteString(fmt.Sprintf(":%d", symptom.Location.Column))
		}
		b.WriteString("\n")
	}
	if symptom.Context != "" {
		b.WriteString(fmt.Sprintf("上下文:\n%s\n", symptom.Context))
	}
	return b.String()
}

// BuildFixPrompt 构建修复提示（用于交给 agent 修复的完整 Prompt）。
func BuildFixPrompt(result *BugDetectResult) string {
	if result.Success {
		return ""
	}

	var b strings.Builder
	b.WriteString("项目检测到以下问题需要修复:\n\n")
	b.WriteString(result.Summary)
	b.WriteString("\n\n请按以下步骤处理:\n")
	b.WriteString("1. 分析每个错误的原因\n")
	b.WriteString("2. 定位到对应代码文件\n")
	b.WriteString("3. 逐个修复\n")
	b.WriteString("4. 重新构建验证\n\n")

	// 列出具体错误
	b.WriteString("### 详细错误列表\n\n")
	for i, s := range result.Symptoms {
		b.WriteString(fmt.Sprintf("#### [%d] %s\n", i+1, s.Message))
		if s.Location.File != "" {
			b.WriteString(fmt.Sprintf("- 文件: `%s:%d`\n", s.Location.File, s.Location.Line))
		}
		if s.Context != "" {
			b.WriteString(fmt.Sprintf("- 代码上下文:\n```\n%s```\n", s.Context))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n修复完成后，调用 finish_task 工具提交结果。")
	return b.String()
}

// ── 工具注册 ───────────────────────────────────────────────

// registerBugDetectTools 注册 BUG 检测相关工具到注册表。
// 包括：bug_detect（全量检测）、bug_analyze（分析构建输出）、bug_fix_prompt（生成修复提示）。
func registerBugDetectTools(r *Registry, root string) {
	// bug_analyze — 分析构建/测试输出，提取错误位置
	r.Register(&Tool{
		Name: "bug_analyze",
		Description: "分析构建/测试/运行输出，提取错误位置和代码上下文。接受 output（构建输出文本），" +
			"output_type（build/test/run），返回解析后的错误列表（含文件路径、行号、消息和代码上下文）。" +
			"由 Detector 在自动检测时调用，也可供 agent 手动分析构建日志。",
		Parameters: map[string]any{
			"type": "object",
			"properties": props{
				"output":      strProp("构建/测试/运行的完整输出文本"),
				"output_type": strProp("输出类型：build（编译输出）/ test（测试输出）/ run（运行时输出），默认 build"),
			},
			"required": []string{"output"},
		},
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			output := argStr(args, "output")
			outputType := argStr(args, "output_type")
			if outputType == "" {
				outputType = "build"
			}

			var symptoms []BugSymptom
			switch outputType {
			case "test":
				symptoms = AnalyzeTestOutput(output, root)
			case "run":
				symptoms = AnalyzeGoRunOutput(output, root)
			default:
				symptoms = AnalyzeBuildOutput(output, root)
				// 如果构建输出没有解析到内容，尝试 vet 分析
				if len(symptoms) == 0 {
					symptoms = AnalyzeVetOutput(output, root)
				}
			}

			if len(symptoms) == 0 {
				return "未检测到错误症状。（可能输出中不包含可识别的错误格式）", nil
			}

			var b strings.Builder
			b.WriteString(fmt.Sprintf("检测到 %d 个问题:\n\n", len(symptoms)))
			for i, s := range symptoms {
				b.WriteString(fmt.Sprintf("[%d] ", i+1))
				b.WriteString(BugSymptomToMessage(s))
				b.WriteString("\n")
			}
			return b.String(), nil
		},
	})
}
