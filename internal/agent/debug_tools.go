package agent

// debug_* 工具集：AI 原生调试 — 基于脚本+日志+分析，语言无关。
//
// 设计原则：
//   传统调试器（DAP/gdb/lldb）绑定语言和协议。AI 真正擅长的是
//   「读代码 → 插日志 → 跑程序 → 分析输出」这个循环。
//   这套工具围绕 AI 的能力圈设计，支持任意语言。
//
// 工具列表：
//   debug_inject_log     — 在指定行后插入日志输出语句（自动识别语言）
//   debug_run_capture    — 运行程序并捕获完整输出（stdout+stderr+exit code）
//   debug_analyze_output  — 分析运行输出，提取错误/异常/堆栈等结构化信息
//   debug_parse_stack    — 解析堆栈轨迹（支持多种语言格式）
//   debug_cleanup_logs   — 移除之前注入的日志语句
//   debug_watch          — 监听文件变更自动重跑命令
//   debug_evaluate_session — 评估 agent 会话表现（离线评分，不浪费 token）

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ─── 语言检测 ─────────────────────────────────────────────

// langInfo 描述一种语言的日志注入方式。
type langInfo struct {
	PrintStmt func(indent, msg string) string // 生成日志语句
	Comment   string                          // 行注释前缀（仅用于检测）
}

var langDetect = map[string]langInfo{
	".go": {
		PrintStmt: func(indent, msg string) string {
			return indent + `fmt.Println("🪵 [DEBUG] ` + msg + `")`
		},
		Comment: "//",
	},
	".py": {
		PrintStmt: func(indent, msg string) string {
			return indent + `print("🪵 [DEBUG] ` + msg + `")`
		},
		Comment: "#",
	},
	".js": {
		PrintStmt: func(indent, msg string) string {
			return indent + `console.log("🪵 [DEBUG] ` + msg + `")`
		},
		Comment: "//",
	},
	".jsx": {
		PrintStmt: func(indent, msg string) string {
			return indent + `console.log("🪵 [DEBUG] ` + msg + `")`
		},
		Comment: "//",
	},
	".ts": {
		PrintStmt: func(indent, msg string) string {
			return indent + `console.log("🪵 [DEBUG] ` + msg + `")`
		},
		Comment: "//",
	},
	".tsx": {
		PrintStmt: func(indent, msg string) string {
			return indent + `console.log("🪵 [DEBUG] ` + msg + `")`
		},
		Comment: "//",
	},
	".vue": {
		PrintStmt: func(indent, msg string) string {
			return indent + `console.log("🪵 [DEBUG] ` + msg + `")`
		},
		Comment: "//",
	},
	".rs": {
		PrintStmt: func(indent, msg string) string {
			return indent + `println!("🪵 [DEBUG] ` + msg + `");`
		},
		Comment: "//",
	},
	".java": {
		PrintStmt: func(indent, msg string) string {
			return indent + `System.out.println("🪵 [DEBUG] ` + msg + `");`
		},
		Comment: "//",
	},
	".c": {
		PrintStmt: func(indent, msg string) string {
			return indent + `printf("🪵 [DEBUG] ` + msg + `\\n");`
		},
		Comment: "//",
	},
	".h": {
		PrintStmt: func(indent, msg string) string {
			return indent + `printf("🪵 [DEBUG] ` + msg + `\\n");`
		},
		Comment: "//",
	},
	".cpp": {
		PrintStmt: func(indent, msg string) string {
			return indent + `std::cout << "🪵 [DEBUG] ` + msg + `" << std::endl;`
		},
		Comment: "//",
	},
	".hpp": {
		PrintStmt: func(indent, msg string) string {
			return indent + `std::cout << "🪵 [DEBUG] ` + msg + `" << std::endl;`
		},
		Comment: "//",
	},
	".cc": {
		PrintStmt: func(indent, msg string) string {
			return indent + `std::cout << "🪵 [DEBUG] ` + msg + `" << std::endl;`
		},
		Comment: "//",
	},
	".cs": {
		PrintStmt: func(indent, msg string) string {
			return indent + `Console.WriteLine("🪵 [DEBUG] ` + msg + `");`
		},
		Comment: "//",
	},
	".rb": {
		PrintStmt: func(indent, msg string) string {
			return indent + `puts "🪵 [DEBUG] ` + msg + `"`
		},
		Comment: "#",
	},
	".php": {
		PrintStmt: func(indent, msg string) string {
			return indent + `echo "🪵 [DEBUG] ` + msg + `\\n";`
		},
		Comment: "//",
	},
	".swift": {
		PrintStmt: func(indent, msg string) string {
			return indent + `print("🪵 [DEBUG] ` + msg + `")`
		},
		Comment: "//",
	},
	".kt": {
		PrintStmt: func(indent, msg string) string {
			return indent + `println("🪵 [DEBUG] ` + msg + `")`
		},
		Comment: "//",
	},
	".sh": {
		PrintStmt: func(indent, msg string) string {
			return indent + `echo "🪵 [DEBUG] ` + msg + `"`
		},
		Comment: "#",
	},
	".bash": {
		PrintStmt: func(indent, msg string) string {
			return indent + `echo "🪵 [DEBUG] ` + msg + `"`
		},
		Comment: "#",
	},
	".lua": {
		PrintStmt: func(indent, msg string) string {
			return indent + `print("🪵 [DEBUG] ` + msg + `")`
		},
		Comment: "--",
	},
	".pl": {
		PrintStmt: func(indent, msg string) string {
			return indent + `print "🪵 [DEBUG] ` + msg + `\\n"`
		},
		Comment: "#",
	},
	".pm": {
		PrintStmt: func(indent, msg string) string {
			return indent + `print "🪵 [DEBUG] ` + msg + `\\n"`
		},
		Comment: "#",
	},
	".ex": {
		PrintStmt: func(indent, msg string) string {
			return indent + `IO.puts("🪵 [DEBUG] ` + msg + `")`
		},
		Comment: "#",
	},
	".exs": {
		PrintStmt: func(indent, msg string) string {
			return indent + `IO.puts("🪵 [DEBUG] ` + msg + `")`
		},
		Comment: "#",
	},
	".dart": {
		PrintStmt: func(indent, msg string) string {
			return indent + `print("🪵 [DEBUG] ` + msg + `");`
		},
		Comment: "//",
	},
}

// detectLang 根据文件名推断语言信息。
func detectLang(path string) (langInfo, string) {
	ext := strings.ToLower(filepath.Ext(path))
	if info, ok := langDetect[ext]; ok {
		return info, ext
	}
	// 后备：无扩展名或未知扩展名，返回 Go 风格（最通用）
	return langDetect[".go"], ".go"
}

// ─── 注册入口 ─────────────────────────────────────────────

func registerDebugTools(r *Registry, root string) {
	registerInjectLog(r, root)
	registerRunCapture(r, root)
	registerAnalyzeOutput(r, root)
	registerParseStack(r, root)
	registerCleanupLogs(r, root)
	registerWatch(r, root)
	registerEvaluateSession(r, root)
}

// ─── 1. debug_inject_log — 注入日志 ─────────────────────

func registerInjectLog(r *Registry, root string) {
	r.Register(&Tool{
		Name:       "debug_inject_log",
		UsageGuide: "在代码指定行后插入日志输出语句（语言无关）。自动识别文件后缀选择 print/console.log/println 等。日志含 🪵 [DEBUG] 标记，后续可用 debug_cleanup_logs 清理。支持 Go/Python/JS/TS/Rust/Java/C++/C#/Ruby/PHP 等 20+ 语言。",
		Description: "在指定文件的指定行后插入日志输出语句。自动根据文件扩展名选择正确的日志语法。" +
			"插入的日志包含 🪵 [DEBUG] 标记，后续可用 debug_cleanup_logs 统一移除。" +
			"支持 Go / Python / JavaScript / TypeScript / Vue / Rust / Java / C / C++ / C# / Ruby / PHP / Swift / Kotlin / Shell / Lua / Perl / Elixir / Dart。" +
			"注意：每行插入一次，插入后行号会偏移，后续注入需考虑偏移。",
		Parameters: objSchema(props{
			"file":    strProp("文件路径（相对于工作区根，如 'src/main.go'）"),
			"lines":   intArrProp("行号数组（从 1 开始），在每行之后插入日志"),
			"message": strProp("可选：自定义日志消息（默认自动标注文件名+行号）"),
		}, "file", "lines"),
		RequiresApproval: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			filePath := argStr(args, "file")
			if filePath == "" {
				return "", fmt.Errorf("file 不能为空")
			}
			lines := argIntSlice(args, "lines")
			if len(lines) == 0 {
				return "", fmt.Errorf("lines 不能为空")
			}
			msg := argStr(args, "message")

			// 读取文件
			absPath := filepath.Join(root, filePath)
			data, err := os.ReadFile(absPath)
			if err != nil {
				return "", fmt.Errorf("读取文件失败: %w", err)
			}

			// 检测语言
			info, ext := detectLang(filePath)

			// 按行处理（从后往前，避免插入导致行号偏移）
			content := string(data)
			scanner := bufio.NewScanner(strings.NewReader(content))
			var origLines []string
			for scanner.Scan() {
				origLines = append(origLines, scanner.Text())
			}

			// 排序并去重行号
			sort.Ints(lines)
			var uniq []int
			for _, l := range lines {
				if l < 1 || l > len(origLines) {
					continue
				}
				if len(uniq) > 0 && uniq[len(uniq)-1] == l {
					continue
				}
				uniq = append(uniq, l)
			}
			if len(uniq) == 0 {
				return "", fmt.Errorf("所有行号超出文件范围（文件共 %d 行）", len(origLines))
			}

			// 从后往前插入
			injected := 0
			for i := len(uniq) - 1; i >= 0; i-- {
				ln := uniq[i] - 1 // 0 基
				origLine := origLines[ln]
				indent := extractIndent(origLine)

				logMsg := msg
				if logMsg == "" {
					logMsg = fmt.Sprintf("%s:%d", filePath, uniq[i])
				}

				stmt := info.PrintStmt(indent, logMsg)
				// 在目标行后插入
				after := make([]string, 0, len(origLines)+1)
				after = append(after, origLines[:ln+1]...)
				after = append(after, stmt)
				after = append(after, origLines[ln+1:]...)
				origLines = after
				injected++
			}

			result := strings.Join(origLines, "\n")
			if err := os.WriteFile(absPath, []byte(result), 0o644); err != nil {
				return "", fmt.Errorf("写入文件失败: %w", err)
			}

			return fmt.Sprintf("已在 %s 中注入 %d 条日志语句（%s 格式）\n"+
				"运行程序后可看到 🪵 [DEBUG] 标记的输出\n"+
				"完成后可用 debug_cleanup_logs 移除", filePath, injected, ext), nil
		},
	})
}

// extractIndent 提取行首空白（缩进）。
func extractIndent(line string) string {
	var i int
	for i < len(line) && (line[i] == ' ' || line[i] == '\t') {
		i++
	}
	return line[:i]
}

// ─── 2. debug_run_capture — 运行并捕获输出 ──────────────

func registerRunCapture(r *Registry, root string) {
	r.Register(&Tool{
		Name:       "debug_run_capture",
		UsageGuide: "运行程序并捕获完整输出（stdout+stderr+exit code+耗时）。比手动 bash 更专注于调试场景：输出无限、报告退出码、包含耗时。支持超时控制。",
		Description: "运行指定命令并捕获完整输出。适用于调试场景：" +
			"运行目标程序，捕获所有 stdout/stderr，报告退出码和执行耗时。" +
			"与 bash 不同：输出不截断、明确报告退出码、包含耗时统计。" +
			"配合 debug_inject_log 使用：注入日志 → 运行捕获 → 分析输出。",
		Parameters: objSchema(props{
			"command": strProp("要执行的命令（如 'python main.py' 或 'go run .' 或 'node app.js'）"),
			"cwd":     strProp("可选：工作目录（相对于工作区根，默认根目录）"),
			"timeout": intProp("可选：超时秒数（默认 60s，-1=不设超时）"),
		}, "command"),
		RequiresApproval: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			command := argStr(args, "command")
			if command == "" {
				return "", fmt.Errorf("command 不能为空")
			}
			cwdStr := argStr(args, "cwd")
			timeoutSec := argInt(args, "timeout", 60)

			dir := root
			if cwdStr != "" {
				dir = filepath.Join(root, cwdStr)
			}

			// 构造命令
			var c *exec.Cmd
			var cancel context.CancelFunc
			if timeoutSec > 0 {
				var ctx2 context.Context
				ctx2, cancel = context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
				c = newShellCommandContext(ctx2, command)
			} else {
				c = newShellCommand(command)
				cancel = func() {}
			}
			if cancel != nil {
				defer cancel()
			}
			c.Dir = dir

			startTime := time.Now()

			// 捕获输出
			output, err := c.CombinedOutput()
			duration := time.Since(startTime)

			// 构造结果
			exitCode := 0
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					exitCode = exitErr.ExitCode()
				} else {
					// 超时或其他错误
					exitCode = -1
				}
			}

			outputStr := decodeCmdOutput(output)
			if len(outputStr) > 50000 {
				outputStr = outputStr[:50000] + "\n... [输出截断，共 " + strconv.Itoa(len(output)) + " 字节]"
			}

			result := fmt.Sprintf("## debug_run_capture 结果\n\n"+
				"**命令**: `%s`\n"+
				"**工作目录**: %s\n"+
				"**退出码**: %d\n"+
				"**耗时**: %v\n"+
				"**状态**: ", command, dir, exitCode, duration.Round(time.Millisecond))

			if exitCode == 0 {
				result += "✅ 正常退出"
			} else if exitCode == -1 {
				result += "⏱️ 超时/异常"
			} else {
				result += "❌ 异常退出"
			}

			if len(output) > 0 {
				result += fmt.Sprintf("\n\n### 输出（%d 字节）\n```\n%s\n```", len(output), outputStr)
			} else {
				result += "\n\n（无输出）"
			}

			return result, nil
		},
	})
}

// ─── 3. debug_analyze_output — 分析输出 ────────────────

// outputAnalysis 结构化的输出分析结果。
type outputAnalysis struct {
	LineCount   int            `json:"line_count"`
	ErrorLines  []string       `json:"error_lines"`
	StackFrames []parsedFrame  `json:"stack_frames"`
	Warnings    []string       `json:"warnings"`
	HasPanic    bool           `json:"has_panic"`
	HasError    bool           `json:"has_error"`
	KeyPatterns map[string]int `json:"key_patterns"`
	Summary     string         `json:"summary"`
}

// parsedFrame 解析出的堆栈帧。
type parsedFrame struct {
	Function string `json:"function"`
	File     string `json:"file"`
	Line     int    `json:"line"`
	Col      int    `json:"col"`
	Lang     string `json:"lang"` // 推断的语言
}

func registerAnalyzeOutput(r *Registry, root string) {
	r.Register(&Tool{
		Name:       "debug_analyze_output",
		UsageGuide: "分析程序运行输出，提取错误行、堆栈帧、警告、异常模式。返回结构化分析结果，帮助 AI 快速定位问题。配合 debug_run_capture 使用。",
		Description: "分析程序运行输出文本，自动提取结构化信息：" +
			"错误行、堆栈帧（支持多语言格式）、警告信息、panic/异常检测。" +
			"返回按行组织的分析报告。不依赖任何调试器协议，纯文本分析。",
		Parameters: objSchema(props{
			"output": strProp("要分析的输出文本（从 debug_run_capture 的结果中提取）"),
		}, "output"),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			output := argStr(args, "output")
			if output == "" {
				return "", fmt.Errorf("output 不能为空")
			}

			analysis := analyzeOutputText(output)

			// 格式化报告
			var b strings.Builder
			b.WriteString("## 输出分析报告\n\n")

			// 概览
			b.WriteString("### 概览\n")
			fmt.Fprintf(&b, "- 总行数: %d\n", analysis.LineCount)
			b.WriteString(fmt.Sprintf("- 含错误: %v\n", analysis.HasError))
			b.WriteString(fmt.Sprintf("- 含 Panic: %v\n", analysis.HasPanic))
			b.WriteString(fmt.Sprintf("- 堆栈帧: %d\n", len(analysis.StackFrames)))
			b.WriteString(fmt.Sprintf("- 警告: %d\n", len(analysis.Warnings)))
			b.WriteString("\n")

			// 关键模式统计
			if len(analysis.KeyPatterns) > 0 {
				b.WriteString("### 关键模式\n\n")
				b.WriteString("| 模式 | 出现次数 |\n")
				b.WriteString("|------|---------|\n")
				// 排序
				keys := make([]string, 0, len(analysis.KeyPatterns))
				for k := range analysis.KeyPatterns {
					keys = append(keys, k)
				}
				sort.Slice(keys, func(i, j int) bool {
					return analysis.KeyPatterns[keys[i]] > analysis.KeyPatterns[keys[j]]
				})
				for _, k := range keys {
					fmt.Fprintf(&b, "| `%s` | %d |\n", k, analysis.KeyPatterns[k])
				}
				b.WriteString("\n")
			}

			// 堆栈帧
			if len(analysis.StackFrames) > 0 {
				b.WriteString("### 堆栈帧\n\n")
				b.WriteString("| # | 函数 | 文件 | 行 | 列 | 语言 |\n")
				b.WriteString("|---|------|------|----|----|----|\n")
				for i, f := range analysis.StackFrames {
					col := ""
					if f.Col > 0 {
						col = strconv.Itoa(f.Col)
					}
					fmt.Fprintf(&b, "| %d | `%s` | `%s` | %d | %s | %s |\n",
						i+1, f.Function, f.File, f.Line, col, f.Lang)
				}
				b.WriteString("\n")
			}

			// 错误行
			if len(analysis.ErrorLines) > 0 {
				b.WriteString("### 疑似错误行\n\n")
				b.WriteString("```\n")
				for _, l := range analysis.ErrorLines {
					b.WriteString(l + "\n")
				}
				b.WriteString("```\n\n")
			}

			// 警告
			if len(analysis.Warnings) > 0 {
				b.WriteString("### 警告\n\n")
				for _, w := range analysis.Warnings {
					b.WriteString(fmt.Sprintf("- %s\n", w))
				}
				b.WriteString("\n")
			}

			b.WriteString("### 摘要\n")
			b.WriteString(analysis.Summary)

			return b.String(), nil
		},
	})
}

// analyzeOutputText 分析输出文本并返回结构化结果。
func analyzeOutputText(output string) outputAnalysis {
	lines := strings.Split(output, "\n")
	analysis := outputAnalysis{
		LineCount:   len(lines),
		ErrorLines:  []string{},
		StackFrames: []parsedFrame{},
		Warnings:    []string{},
		KeyPatterns: map[string]int{},
	}

	lower := strings.ToLower(output)

	// 检测关键模式
	patterns := map[string]string{
		"error":            "error",
		"exception":        "exception",
		"panic":            "panic",
		"traceback":        "traceback",
		"failed":           "failed",
		"warning":          "warning",
		"warn":             "warn",
		"fatal":            "fatal",
		"undefined":        "undefined",
		"nil pointer":      "nil pointer",
		"segmentation":     "segmentation",
		"bus error":        "bus error",
		"assertion":        "assertion",
		"cannot find":      "cannot find",
		"module not found": "module not found",
	}
	for key, pattern := range patterns {
		count := strings.Count(lower, pattern)
		if count > 0 {
			analysis.KeyPatterns[key] = count
		}
	}

	analysis.HasPanic = strings.Contains(lower, "panic") || strings.Contains(lower, "fatal")
	analysis.HasError = strings.Contains(lower, "error") ||
		strings.Contains(lower, "exception") ||
		strings.Contains(lower, "failed") ||
		analysis.HasPanic

	// 逐行分析
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		lowerLine := strings.ToLower(trimmed)

		// 错误行
		if strings.Contains(lowerLine, "error") ||
			strings.Contains(lowerLine, "exception:") ||
			strings.Contains(lowerLine, "panic:") ||
			strings.Contains(lowerLine, "fatal:") ||
			strings.Contains(lowerLine, "traceback") {
			analysis.ErrorLines = append(analysis.ErrorLines, fmt.Sprintf("L%d: %s", i+1, trimmed))
		}

		// 警告
		if strings.HasPrefix(lowerLine, "warning") || strings.HasPrefix(lowerLine, "warn:") {
			analysis.Warnings = append(analysis.Warnings, trimmed)
		}

		// 堆栈帧 — 检测各种语言的格式
		if frames := tryParseStackFrame(trimmed); len(frames) > 0 {
			analysis.StackFrames = append(analysis.StackFrames, frames...)
		}
	}

	// 生成摘要
	analysis.Summary = buildAnalysisSummary(analysis)

	return analysis
}

// tryParseStackFrame 尝试从一行文本中解析堆栈帧。
func tryParseStackFrame(line string) []parsedFrame {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return nil
	}

	var frames []parsedFrame

	// Go 格式: goroutine 1 [running]: main.foo(...) /path/file.go:10 +0x100
	reGo := regexp.MustCompile(`^\s*([\w./]+)\(.*\)\s+(/\S+):(\d+)(?:\s+\+0x[0-9a-f]+)?\s*$`)
	if m := reGo.FindStringSubmatch(trimmed); len(m) >= 4 {
		line, _ := strconv.Atoi(m[3])
		frames = append(frames, parsedFrame{
			Function: m[1],
			File:     m[2],
			Line:     line,
			Lang:     "Go",
		})
		return frames
	}

	// Python 格式: File "file.py", line 10, in function
	rePy := regexp.MustCompile(`^\s*File\s+"([^"]+)",\s+line\s+(\d+)(?:,\s+in\s+(\S+))?\s*$`)
	if m := rePy.FindStringSubmatch(trimmed); len(m) >= 3 {
		line, _ := strconv.Atoi(m[2])
		fn := m[3]
		if fn == "" {
			fn = "<module>"
		}
		frames = append(frames, parsedFrame{
			Function: fn,
			File:     m[1],
			Line:     line,
			Lang:     "Python",
		})
		return frames
	}

	// JS/TS 格式: at function (/path/file.ts:10:5)
	reJS := regexp.MustCompile(`^\s*at\s+(?:(?:async\s+)?)?(\S+)\s+\(?(\S+?):(\d+):(\d+)\)?\s*$`)
	if m := reJS.FindStringSubmatch(trimmed); len(m) >= 5 {
		line, _ := strconv.Atoi(m[3])
		col, _ := strconv.Atoi(m[4])
		frames = append(frames, parsedFrame{
			Function: m[1],
			File:     m[2],
			Line:     line,
			Col:      col,
			Lang:     "JS/TS",
		})
		return frames
	}

	// JS/TS 简化格式: at /path/file.ts:10:5
	reJS2 := regexp.MustCompile(`^\s*at\s+(\S+?):(\d+):(\d+)\s*$`)
	if m := reJS2.FindStringSubmatch(trimmed); len(m) >= 4 {
		line, _ := strconv.Atoi(m[2])
		col, _ := strconv.Atoi(m[3])
		frames = append(frames, parsedFrame{
			Function: "<anonymous>",
			File:     m[1],
			Line:     line,
			Col:      col,
			Lang:     "JS/TS",
		})
		return frames
	}

	// Java 格式: at com.example.Foo.bar(Foo.java:10)
	reJava := regexp.MustCompile(`^\s*at\s+([\w.]+)\.(\w+)\(([\w.]+):(\d+)\)\s*$`)
	if m := reJava.FindStringSubmatch(trimmed); len(m) >= 5 {
		line, _ := strconv.Atoi(m[4])
		frames = append(frames, parsedFrame{
			Function: m[1] + "." + m[2],
			File:     m[3],
			Line:     line,
			Lang:     "Java",
		})
		return frames
	}

	// Rust 格式: at file:line:col
	reRs := regexp.MustCompile(`^\s*at\s+(\S+?):(\d+):(\d+)\s*$`)
	if m := reRs.FindStringSubmatch(trimmed); len(m) >= 4 {
		line, _ := strconv.Atoi(m[2])
		col, _ := strconv.Atoi(m[3])
		frames = append(frames, parsedFrame{
			File: m[1],
			Line: line,
			Col:  col,
			Lang: "Rust",
		})
		return frames
	}

	// Rust panic 格式: file:line:col
	reRs2 := regexp.MustCompile(`^\s*(\S+?):(\d+):(\d+)\s*$`)
	if m := reRs2.FindStringSubmatch(trimmed); len(m) >= 4 {
		line, _ := strconv.Atoi(m[2])
		col, _ := strconv.Atoi(m[3])
		// 文件需要看起来像源码路径
		if strings.Contains(m[1], ".") || strings.Contains(m[1], "/") || strings.Contains(m[1], `\`) {
			frames = append(frames, parsedFrame{
				File: m[1],
				Line: line,
				Col:  col,
				Lang: "Rust",
			})
		}
		return frames
	}

	// C# 格式: at Namespace.Class.Method() in file:line
	reCs := regexp.MustCompile(`^\s*at\s+(\S+)\(.*\)\s+in\s+(\S+):(\d+)\s*$`)
	if m := reCs.FindStringSubmatch(trimmed); len(m) >= 4 {
		line, _ := strconv.Atoi(m[3])
		frames = append(frames, parsedFrame{
			Function: m[1],
			File:     m[2],
			Line:     line,
			Lang:     "C#",
		})
		return frames
	}

	return nil
}

// buildAnalysisSummary 生成分析摘要。
func buildAnalysisSummary(analysis outputAnalysis) string {
	if analysis.LineCount == 0 {
		return "输出为空，程序可能未产生任何输出。"
	}

	parts := []string{}

	if analysis.HasPanic {
		parts = append(parts, "程序发生了 panic/崩溃。")
	} else if analysis.HasError {
		parts = append(parts, "输出中包含错误信息。")
	}

	if len(analysis.StackFrames) > 0 {
		parts = append(parts, fmt.Sprintf("检测到 %d 个堆栈帧，可追踪错误源头。", len(analysis.StackFrames)))
	}

	if analysis.HasError {
		// 找到第一个错误行的文件/行信息
		for _, e := range analysis.ErrorLines {
			parts = append(parts, fmt.Sprintf("关键错误: %s", e))
			break
		}
	}

	if len(parts) == 0 {
		parts = append(parts, "未检测到明显错误。输出中包含警告/日志信息，建议检查输出全文。")
	}

	return strings.Join(parts, " ")
}

// ─── 4. debug_parse_stack — 解析堆栈轨迹 ───────────────

func registerParseStack(r *Registry, root string) {
	r.Register(&Tool{
		Name:       "debug_parse_stack",
		UsageGuide: "解析堆栈轨迹文本为结构化数据（帧列表）。自动识别 Go/Python/JS/TS/Java/Rust/C# 等多种格式。返回函数名、文件、行号、列号。",
		Description: "解析堆栈轨迹文本，返回结构化的帧列表。自动识别多种语言的堆栈格式：" +
			"Go（goroutine）、Python（Traceback）、JS/TS（at）、Java（at）、Rust（at/panic）、C#（at ... in）。" +
			"结果包含函数名、源文件、行号、列号、语言类型。" +
			"可用于将运行错误链接回源代码位置。",
		Parameters: objSchema(props{
			"text": strProp("堆栈轨迹文本（从错误输出中提取）"),
		}, "text"),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			text := argStr(args, "text")
			if text == "" {
				return "", fmt.Errorf("text 不能为空")
			}

			lines := strings.Split(text, "\n")
			var allFrames []parsedFrame
			for _, line := range lines {
				if frames := tryParseStackFrame(line); len(frames) > 0 {
					allFrames = append(allFrames, frames...)
				}
			}

			if len(allFrames) == 0 {
				return "未能从输入中解析出堆栈帧。支持的格式：\n" +
						"- Go:     `main.foo() /path/file.go:10`\n" +
						"- Python: `File \"file.py\", line 10, in function`\n" +
						"- JS/TS:  `at function (/path/file.ts:10:5)`\n" +
						"- Java:   `at com.example.Foo.bar(Foo.java:10)`\n" +
						"- Rust:   `at /path/file.rs:10:5`\n" +
						"- C#:     `at Namespace.Class.Method() in file:10`",
					nil
			}

			var b strings.Builder
			b.WriteString(fmt.Sprintf("## 解析结果（%d 帧）\n\n", len(allFrames)))
			b.WriteString("| # | 函数 | 文件 | 行 | 列 | 语言 |\n")
			b.WriteString("|---|------|------|----|----|----|\n")
			for i, f := range allFrames {
				col := ""
				if f.Col > 0 {
					col = strconv.Itoa(f.Col)
				}
				fn := f.Function
				if fn == "" {
					fn = "(anonymous)"
				}
				fmt.Fprintf(&b, "| %d | `%s` | `%s` | %d | %s | %s |\n",
					i+1, fn, f.File, f.Line, col, f.Lang)
			}

			// 按文件分组提示
			b.WriteString("\n### 涉及的文件\n\n")
			files := map[string][]parsedFrame{}
			for _, f := range allFrames {
				if f.File != "" {
					files[f.File] = append(files[f.File], f)
				}
			}
			for file, frames := range files {
				b.WriteString(fmt.Sprintf("- `%s` (%d 帧)\n", file, len(frames)))
			}

			return b.String(), nil
		},
	})
}

// ─── 5. debug_cleanup_logs — 清理注入的日志 ────────────

func registerCleanupLogs(r *Registry, root string) {
	r.Register(&Tool{
		Name:       "debug_cleanup_logs",
		UsageGuide: "移除之前通过 debug_inject_log 注入的日志语句（包含 🪵 [DEBUG] 标记的行）。可指定单个文件或全部清理。",
		Description: "移除之前通过 debug_inject_log 注入的日志语句。" +
			"扫描文件中包含 🪵 [DEBUG] 标记的行并删除。" +
			"可指定单个文件，或省略 file 参数自动扫描工作区内所有被注入过的文件。" +
			"注意：仅移除通过 debug_inject_log 注入的日志，不影响手写的日志代码。",
		Parameters: objSchema(props{
			"file": strProp("可选：要清理的文件路径（省略则扫描工作区内所有可能被注入的文件）"),
		}),
		RequiresApproval: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			filePath := argStr(args, "file")

			if filePath != "" {
				return cleanupSingleFile(filePath, root)
			}
			return cleanupAllFiles(root)
		},
	})
}

// cleanupSingleFile 清理单个文件中的注入日志。
func cleanupSingleFile(filePath, root string) (string, error) {
	absPath := filepath.Join(root, filePath)
	data, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("读取文件失败: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	var cleaned []string
	removed := 0
	for _, line := range lines {
		if strings.Contains(line, "🪵 [DEBUG]") {
			removed++
			continue
		}
		cleaned = append(cleaned, line)
	}

	if removed == 0 {
		return fmt.Sprintf("%s 中没有找到注入的日志（🪵 [DEBUG] 标记）", filePath), nil
	}

	result := strings.Join(cleaned, "\n")
	if err := os.WriteFile(absPath, []byte(result), 0o644); err != nil {
		return "", fmt.Errorf("写入文件失败: %w", err)
	}

	return fmt.Sprintf("已从 %s 中移除 %d 条注入的日志语句", filePath, removed), nil
}

// cleanupAllFiles 扫描工作区并清理所有含注入日志的文件。
func cleanupAllFiles(root string) (string, error) {
	totalRemoved := 0
	cleanedFiles := 0

	// 扫描常见源文件
	exts := make([]string, 0, len(langDetect))
	for ext := range langDetect {
		exts = append(exts, ext)
	}

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // 跳过无法访问的目录
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "node_modules" || name == ".pair" ||
				name == "vendor" || name == "target" || name == "build" ||
				name == "dist" || name == "__pycache__" || name == ".next" {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		// 只处理已知扩展名
		known := false
		for _, e := range exts {
			if ext == e {
				known = true
				break
			}
		}
		if !known {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		lines := strings.Split(string(data), "\n")
		var cleaned []string
		removed := 0
		for _, line := range lines {
			if strings.Contains(line, "🪵 [DEBUG]") {
				removed++
				continue
			}
			cleaned = append(cleaned, line)
		}

		if removed > 0 {
			result := strings.Join(cleaned, "\n")
			if err := os.WriteFile(path, []byte(result), 0o644); err == nil {
				totalRemoved += removed
				cleanedFiles++
			}
		}
		return nil
	})

	if err != nil {
		return "", fmt.Errorf("扫描文件失败: %w", err)
	}

	if cleanedFiles == 0 {
		return "工作区中未找到注入的日志语句（🪵 [DEBUG] 标记）。可能已全部清理。", nil
	}

	return fmt.Sprintf("已清理 %d 个文件，共移除 %d 条注入的日志语句", cleanedFiles, totalRemoved), nil
}

// ─── 6. debug_watch — 文件监听+自动重跑 ────────────────

// watchProc 一个监听器实例。
type watchProc struct {
	id      int
	stopCh  chan struct{}
	root    string
	pattern string
	command string
	timeout int
	lastMod map[string]time.Time // 文件→最后修改时间
	output  string
	mu      sync.Mutex
}

var (
	watchMu     sync.Mutex
	watchProcs  = map[int]*watchProc{}
	watchNextID = 0
)

func registerWatch(r *Registry, root string) {
	r.Register(&Tool{
		Name:       "debug_watch",
		UsageGuide: "监听文件变更并自动重跑命令。用于「改代码→自动跑」的调试循环。指定 glob 模式匹配文件，变更后自动执行命令。内置 2s 轮询+去抖动。stop=true 停止指定 watch。",
		Description: "监听匹配 glob 模式的文件，变更后自动执行指定命令。" +
			"用于「改代码→自动跑」的调试循环。内置 2 秒轮询，500ms 去抖动。" +
			"stop=true 可停止指定 id 的监听器。用 list=true 查看所有活跃监听器。",
		Parameters: objSchema(props{
			"pattern": strProp("文件匹配模式（如 '**/*.go' 或 'src/**/*.py'），相对于工作区根"),
			"command": strProp("文件变更后要执行的命令（如 'go test ./...' 或 'python main.py'）"),
			"timeout": intProp("可选：每次运行的超时秒数（默认 120s）"),
			"stop":    strProp("可选：停止指定 id 的监听器"),
			"list":    boolProp("可选：设为 true 列出所有活跃监听器"),
		}),
		RequiresApproval: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			// 列出活跃监听器
			if argBool(args, "list") {
				watchMu.Lock()
				n := len(watchProcs)
				ids := make([]int, 0, n)
				for id := range watchProcs {
					ids = append(ids, id)
				}
				watchMu.Unlock()
				if n == 0 {
					return "当前没有活跃的文件监听器。", nil
				}
				var b strings.Builder
				b.WriteString(fmt.Sprintf("活跃的文件监听器（%d 个）:\n", n))
				for _, id := range ids {
					watchMu.Lock()
					wp := watchProcs[id]
					watchMu.Unlock()
					if wp != nil {
						fmt.Fprintf(&b, "  [%d] 模式: `%s` | 命令: `%s`\n", id, wp.pattern, wp.command)
					}
				}
				return b.String(), nil
			}

			// 停止指定监听器
			stopID := argStr(args, "stop")
			if stopID != "" {
				id, err := strconv.Atoi(stopID)
				if err != nil {
					return "", fmt.Errorf("无效的 id: %s", stopID)
				}
				watchMu.Lock()
				wp, ok := watchProcs[id]
				if ok {
					close(wp.stopCh)
					delete(watchProcs, id)
				}
				watchMu.Unlock()
				if !ok {
					return fmt.Sprintf("未找到 id=%d 的监听器", id), nil
				}
				return fmt.Sprintf("已停止文件监听器 [%d]", id), nil
			}

			// 启动新监听器
			pattern := argStr(args, "pattern")
			if pattern == "" {
				return "", fmt.Errorf("pattern 不能为空")
			}
			command := argStr(args, "command")
			if command == "" {
				return "", fmt.Errorf("command 不能为空")
			}
			timeoutSec := argInt(args, "timeout", 120)
			if timeoutSec <= 0 {
				timeoutSec = 120
			}

			watchMu.Lock()
			watchNextID++
			id := watchNextID
			wp := &watchProc{
				id:      id,
				stopCh:  make(chan struct{}),
				root:    root,
				pattern: pattern,
				command: command,
				timeout: timeoutSec,
				lastMod: map[string]time.Time{},
			}
			watchProcs[id] = wp
			watchMu.Unlock()

			// 后台启动轮询 goroutine
			go wp.run()

			return fmt.Sprintf("文件监听已启动 (id=%d)\n"+
				"模式: `%s`\n"+
				"命令: `%s`\n"+
				"每次变更后自动执行命令，结果写入 watch 日志\n"+
				"用 debug_watch list=true 查看状态\n"+
				"用 debug_watch stop=%d 停止", id, pattern, command, id), nil
		},
	})
}

// run 启动轮询循环。
func (wp *watchProc) run() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	// 首次扫描建立基线
	wp.scanFiles()

	for {
		select {
		case <-wp.stopCh:
			return
		case <-ticker.C:
			changed := wp.scanFiles()
			if len(changed) > 0 {
				wp.execute(changed)
			}
		}
	}
}

// scanFiles 扫描匹配的文件，返回本次新增变更的文件列表。
func (wp *watchProc) scanFiles() []string {
	current := map[string]time.Time{}

	// 扫描匹配文件
	filepath.WalkDir(wp.root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			name := ""
			if d != nil {
				name = d.Name()
			}
			if name == ".git" || name == "node_modules" || name == ".pair" ||
				name == "vendor" || name == "target" || name == "build" ||
				name == "dist" || name == "__pycache__" {
				return filepath.SkipDir
			}
			return nil
		}
		// 简单通配匹配
		if matchGlob(wp.pattern, d.Name()) {
			info, err := d.Info()
			if err == nil {
				current[path] = info.ModTime()
			}
		}
		return nil
	})

	// 找变更
	var changed []string
	// 检查新增或修改
	for path, modTime := range current {
		if lastMod, ok := wp.lastMod[path]; !ok || modTime.After(lastMod) {
			changed = append(changed, path)
		}
	}
	// 检查删除
	for path := range wp.lastMod {
		if _, ok := current[path]; !ok {
			changed = append(changed, path+" (deleted)")
		}
	}

	wp.lastMod = current
	return changed
}

// execute 执行命令并记录输出。
func (wp *watchProc) execute(changed []string) {
	// 输出变更文件
	var b strings.Builder
	b.WriteString(fmt.Sprintf("[watch %d] 检测到 %d 个文件变更:\n", wp.id, len(changed)))
	now := time.Now().Format("15:04:05")
	for _, f := range changed {
		b.WriteString(fmt.Sprintf("  - %s\n", f))
	}
	b.WriteString(fmt.Sprintf("[%s] 开始执行: %s\n", now, wp.command))

	// 执行命令
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(wp.timeout)*time.Second)
	defer cancel()
	c := newShellCommandContext(ctx, wp.command)
	c.Dir = wp.root

	startTime := time.Now()
	output, err := c.CombinedOutput()
	elapsed := time.Since(startTime).Round(time.Millisecond)

	b.WriteString(fmt.Sprintf("[%s] 完成（%v）\n", time.Now().Format("15:04:05"), elapsed))
	if err != nil {
		b.WriteString(fmt.Sprintf("退出码: %d\n", c.ProcessState.ExitCode()))
	}
	if len(output) > 0 {
		b.WriteString("```\n")
		out := decodeCmdOutput(output)
		b.WriteString(out)
		if !strings.HasSuffix(out, "\n") {
			b.WriteString("\n")
		}
		b.WriteString("```\n")
	}

	wp.mu.Lock()
	wp.output += b.String()
	if len(wp.output) > 50000 {
		wp.output = wp.output[len(wp.output)-40000:]
	}
	wp.mu.Unlock()
}

// ─── 7. debug_evaluate_session — 会话评分评估 ──────────

// SessionScore 会话评分结果。
type SessionScore struct {
	SessionID  string             `json:"session_id"`
	Task       string             `json:"task"`
	Completed  bool               `json:"completed"`
	Rounds     int                `json:"rounds"`
	ToolCalls  int                `json:"tool_calls"`
	ToolStats  []ToolStatsSummary `json:"tool_stats"`
	ErrorCount int                `json:"error_count"`
	PanicCount int                `json:"panic_count"`

	// 各维度评分（0-100）
	CompletionScore   float64 `json:"completion_score"`   // 任务完成度
	EfficiencyScore   float64 `json:"efficiency_score"`   // 效率（工具调用数/轮次）
	ReliabilityScore  float64 `json:"reliability_score"`  // 可靠性（工具成功率）
	AdaptabilityScore float64 `json:"adaptability_score"` // 适应性（错误恢复率）
	OverallScore      float64 `json:"overall_score"`      // 总分
}

func registerEvaluateSession(r *Registry, root string) {
	r.Register(&Tool{
		Name:       "debug_evaluate_session",
		UsageGuide: "对 agent 会话进行离线评分评估（机械公式）。如需更高质的语义化评分，请运行独立评分工具：go run ./cmd/evaluator -root <workspace>。评分是离线分析，不消耗 agent 运行时的 token。",
		Description: "评估 agent 会话的表现，生成结构化评分报告。" +
			"基于已保存的执行日志（.pair/execution_logs/）和工具调用统计进行离线分析。" +
			"四个评分维度：完成度（任务是否完成）、效率（工具调用合理度）、" +
			"可靠性（工具成功率）、适应性（错误恢复能力）。" +
			"评分是离线分析，不消耗 agent 运行时的 token。" +
			"评分结果可用于自我迭代参考。\n\n" +
			"如需更高质的语义化 LLM 评分，请使用独立评分工具：\n" +
			"  go run ./cmd/evaluator -conv-id <conv_id> -root <workspace_root>\n" +
			"该工具是独立项目，不依赖 agent 运行时，通过环境变量 BASE_URL/API_KEY/MODEL 配置 LLM。",
		Parameters: objSchema(props{
			"conv_id": strProp("可选：对话 ID（省略则评估最近一次会话）"),
		}),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			convID := argStr(args, "conv_id")

			// 加载执行日志
			var log *ExecutionLog
			if convID != "" {
				log = LoadExecutionLog(root, convID)
			} else {
				// 尝试找到最近的日志
				log = findLatestExecutionLog(root)
			}

			if log == nil || len(log.Entries) == 0 {
				return "未找到执行日志。请先运行一次 agent 会话后再评估。\n" +
					"执行日志存放在 .pair/execution_logs/ 目录。", nil
			}

			// 加载工具统计
			ts := GetToolStats()

			// 计算评分
			score := evaluateSession(log, ts)

			// 格式化报告
			var b strings.Builder
			b.WriteString("## Agent 会话评分报告\n\n")

			if convID != "" {
				fmt.Fprintf(&b, "**会话**: `%s`\n\n", convID)
			}

			// 基本统计
			b.WriteString("### 基本统计\n\n")
			fmt.Fprintf(&b, "- 轮次: %d\n", score.Rounds)
			fmt.Fprintf(&b, "- 工具调用: %d\n", score.ToolCalls)
			fmt.Fprintf(&b, "- 错误数: %d\n", score.ErrorCount)
			fmt.Fprintf(&b, "- Panic 数: %d\n", score.PanicCount)
			b.WriteString("\n")

			// 评分卡片
			b.WriteString("### 评分\n\n")
			b.WriteString("| 维度 | 分数 | 评级 | 说明 |\n")
			b.WriteString("|------|------|------|------|\n")

			dims := []struct {
				name  string
				score float64
				desc  string
			}{
				{"🎯 完成度", score.CompletionScore, "任务是否完成、会话是否自然结束"},
				{"⚡ 效率", score.EfficiencyScore, "工具调用数量是否合理、轮次是否过多"},
				{"🔒 可靠性", score.ReliabilityScore, "工具调用成功率、错误率"},
				{"🔄 适应性", score.AdaptabilityScore, "错误后的恢复能力、重试成功率"},
				{"📊 总分", score.OverallScore, "加权综合评分"},
			}

			for _, d := range dims {
				rating := scoreRating(d.score)
				fmt.Fprintf(&b, "| %s | **%.1f** | %s | %s |\n", d.name, d.score, rating, d.desc)
			}
			b.WriteString("\n")

			// 工具统计
			if len(score.ToolStats) > 0 {
				b.WriteString("### 工具调用统计\n\n")
				b.WriteString("| 工具名 | 调用 | 成功 | 失败 | 成功率 |\n")
				b.WriteString("|--------|------|------|------|--------|\n")
				for _, s := range score.ToolStats {
					fmt.Fprintf(&b, "| `%s` | %d | %d | %d | %s |\n",
						s.Name, s.Calls, s.Success, s.Failures, s.Rate)
				}
				b.WriteString("\n")
			}

			// 改进建议
			b.WriteString("### 改进建议\n\n")
			suggestions := generateSuggestions(score)
			if len(suggestions) > 0 {
				for _, s := range suggestions {
					fmt.Fprintf(&b, "- %s\n", s)
				}
			} else {
				b.WriteString("当前表现良好，无明显改进点。\n")
			}

			b.WriteString("\n---\n")
			b.WriteString("评分标准：90+ 优秀 | 80-89 良好 | 70-79 一般 | 60-69 需要改进 | <60 差")

			return b.String(), nil
		},
	})
}

// evaluateSession 根据执行日志和工具统计计算评分。
func evaluateSession(log *ExecutionLog, ts *ToolStatsRecorder) SessionScore {
	score := SessionScore{
		Rounds: len(log.Entries),
	}

	// 分析日志内容
	hasFinishTask := false
	errorCount := 0
	panicCount := 0
	toolCallCount := 0
	totalPhases := len(log.Entries)

	for _, e := range log.Entries {
		summary := strings.ToLower(e.Summary)
		if strings.Contains(summary, "任务完成") || strings.Contains(summary, "generate_commit") {
			hasFinishTask = true
		}
		if strings.Contains(summary, "error") || strings.Contains(summary, "失败") {
			errorCount++
		}
		if strings.Contains(summary, "panic") {
			panicCount++
		}
		if strings.Contains(summary, "工具调用") || strings.Contains(summary, "执行") {
			toolCallCount++
		}
	}

	score.ErrorCount = errorCount
	score.PanicCount = panicCount
	score.Completed = hasFinishTask
	score.ToolCalls = toolCallCount

	// 加载工具统计
	if ts != nil {
		score.ToolStats = ts.Summary(0)
		totalCalls := ts.TotalCalls()
		if totalCalls > 0 {
			score.ToolCalls = totalCalls
		}

		// 计算工具成功率
		var totalSuccess, totalFail int
		for _, s := range score.ToolStats {
			totalSuccess += s.Success
			totalFail += s.Failures
		}
		totalAttempts := totalSuccess + totalFail

		// 完成度评分（0-100）
		if hasFinishTask || totalPhases >= 3 {
			score.CompletionScore = 80.0
			if hasFinishTask {
				score.CompletionScore = 100.0
			}
			// 如果错误很多，扣分
			if errorCount > 5 {
				score.CompletionScore -= 10
			}
		} else {
			score.CompletionScore = float64(totalPhases) / 10.0 * 100.0
			if score.CompletionScore > 50 {
				score.CompletionScore = 50
			}
		}

		// 效率评分（工具调用 / 轮次）
		if totalPhases > 0 {
			ratio := float64(totalCalls) / float64(totalPhases)
			if ratio <= 3 {
				score.EfficiencyScore = 90
			} else if ratio <= 5 {
				score.EfficiencyScore = 70
			} else if ratio <= 10 {
				score.EfficiencyScore = 50
			} else {
				score.EfficiencyScore = 30
			}
		} else {
			score.EfficiencyScore = 50
		}
		// 少量工具调用可能表示效率高
		if totalCalls <= 2 && hasFinishTask {
			score.EfficiencyScore = 100
		}

		// 可靠性评分（工具成功率）
		if totalAttempts > 0 {
			rate := float64(totalSuccess) / float64(totalAttempts) * 100
			score.ReliabilityScore = rate * 1.0
			// 扣分：panic
			score.ReliabilityScore -= float64(panicCount) * 10
			if score.ReliabilityScore < 0 {
				score.ReliabilityScore = 0
			}
			if score.ReliabilityScore > 100 {
				score.ReliabilityScore = 100
			}
		} else {
			score.ReliabilityScore = 80
		}

		// 适应性评分（错误恢复能力）
		if errorCount > 0 && totalCalls > 0 {
			recoveryRate := float64(totalCalls-errorCount) / float64(totalCalls) * 100
			score.AdaptabilityScore = recoveryRate
		} else if totalCalls > 0 {
			score.AdaptabilityScore = 90
		} else {
			score.AdaptabilityScore = 80
		}
	} else {
		// 无工具统计时的默认评分
		score.CompletionScore = 50
		score.EfficiencyScore = 50
		score.ReliabilityScore = 50
		score.AdaptabilityScore = 50
	}

	// 总分 = 加权平均
	score.OverallScore = score.CompletionScore*0.35 +
		score.EfficiencyScore*0.20 +
		score.ReliabilityScore*0.30 +
		score.AdaptabilityScore*0.15

	return score
}

// scoreRating 根据分数返回评级文字。
func scoreRating(s float64) string {
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

// generateSuggestions 根据评分生成改进建议。
func generateSuggestions(s SessionScore) []string {
	var suggestions []string

	if !s.Completed {
		suggestions = append(suggestions, "任务未完成。检查任务目标、是否存在报错或卡死。")
	}
	if s.EfficiencyScore < 60 {
		suggestions = append(suggestions, "工具调用过多/轮次过长。考虑用更少的工具调用完成任务，善用并行执行。")
	}
	if s.ReliabilityScore < 70 {
		suggestions = append(suggestions, "工具调用失败率高。检查高频失败工具的原因，可能需要调整调用方式或创建新工具。")
	}
	if s.AdaptabilityScore < 60 {
		suggestions = append(suggestions, "错误恢复能力弱。发生错误后未能有效恢复，建议增加错误处理和重试逻辑。")
	}
	if s.PanicCount > 0 {
		suggestions = append(suggestions, fmt.Sprintf("发生了 %d 次 panic，需要排查根本原因。", s.PanicCount))
	}
	if s.ErrorCount > 5 {
		suggestions = append(suggestions, fmt.Sprintf("错误次数较多（%d 次），建议检查高频错误模式。", s.ErrorCount))
	}

	return suggestions
}

// findLatestExecutionLog 查找最近的执行日志文件。
func findLatestExecutionLog(root string) *ExecutionLog {
	dir := executionLogDir(root)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
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
		return nil
	}

	convID := strings.TrimSuffix(latest, ".json")
	return LoadExecutionLog(root, convID)
}

// ─── 辅助函数 ─────────────────────────────────────────────

// intArrProp 生成整数数组类型的 JSON Schema 属性。
func intArrProp(desc string) map[string]any {
	return map[string]any{
		"type":        "array",
		"description": desc,
		"items":       map[string]any{"type": "integer"},
	}
}

// argIntSlice 从参数中提取整数切片。
func argIntSlice(args map[string]any, key string) []int {
	raw, ok := args[key].([]any)
	if !ok {
		return nil
	}
	out := make([]int, 0, len(raw))
	for _, v := range raw {
		switch n := v.(type) {
		case float64:
			out = append(out, int(n))
		case int:
			out = append(out, n)
		}
	}
	return out
}
