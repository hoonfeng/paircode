// harness_tools —— 对齐 deepseek-harness 核心工具集的工具注册。
//
// 背景（2026-08-15）：用户要求按 deepseek-harness 设计重写 agent，并「暂时对齐他的工具集」
// 为下一步自举迭代（用 agent 开发 agent）做准备。harness 核心工具集：
//
//	tool-fs:             read / write / edit / read_image
//	tool-fs-search:      glob / grep
//	tool-str-replace:    str_replace_editor（view/create/str_replace/insert）
//	tool-bash:           bash（含 run_in_background）
//	tool-web:            web_search / web_fetch
//	code-mode:           run_code
//
// 本文件新增 harness 命名的工具；gou-ide 原有工具（read_file/write_file/edit_file/
// search_content/search_files/run_command/web_fetch/web_search 等）全部保留（前端/测试/
// 调用方兼容），新工具与旧工具共用同一执行体（handler 复用），不复制逻辑。
//
// 映射关系：
//
//	read           ↔ read_file（同 handler）
//	write          ↔ write_file
//	edit           ↔ edit_file
//	glob           ↔ search_files（harness glob 语义：路径通配，** 递归）
//	grep           ↔ search_content（harness grep 语义：正则全文搜索）
//	bash           ↔ run_command（harness bash 语义：shell 命令）
//	str_replace_editor  全新实现（view/create/str_replace/insert 四命令）
//	run_code            全新实现（执行一段代码，对齐 harness code-mode）
//	web_search / web_fetch 已有同名工具，保持
package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// RegisterHarnessTools 注册 deepseek-harness 命名的核心工具集。
// 必须在 RegisterDefaultTools 之后调用（handler 复用依赖旧工具已注册）。
func RegisterHarnessTools(r *Registry, root string) {
	registerHarnessAliases(r)
	registerStrReplaceEditor(r, root)
	registerRunCode(r, root)
}

// harnessAlias 复制旧工具注册为 harness 命名别名（同一 Handler，不复制逻辑）。
// 保留 ReadOnly/RequiresApproval 语义；Description/UsageGuide 对齐 harness 风格。
type harnessAlias struct {
	alias       string // 新工具名（harness 命名）
	source      string // 旧工具名（gou-ide 命名）
	description string // harness 风格描述
	usageGuide  string // harness 风格使用指南
	category    string
}

func registerHarnessAliases(r *Registry) {
	aliases := []harnessAlias{
		{
			alias: "read", source: "read_file",
			description: "读取文件内容（对齐 deepseek-harness read）。path 为工作区内路径；可选 offset(起始行,1 基)+limit(行数)读片段；省略则读全文(超 2000 行只返回前 2000 行并提示翻页)。",
			usageGuide:  "harness 标准读工具：读取文件内容。路径越界自动拦截，二进制自动拒绝（改用 inspect_binary）。大文件用 offset+limit 分页。",
			category:    "文件",
		},
		{
			alias: "write", source: "write_file",
			description: "把 content 完整写入 path（覆盖；父目录自动创建）。需审核批准。",
			usageGuide:  "harness 标准写工具：整文件写入（覆盖）。写类操作需人工确认。如需追加请先 read 再 write 覆盖。",
			category:    "文件",
		},
		{
			alias: "edit", source: "edit_file",
			description: "把文件中唯一一处 old_string 替换为 new_string（对齐 deepseek-harness edit）。内置智能匹配（CRLF 归一化+空白折叠）；匹配失败优先用 line_start/line_end 行号定位。",
			usageGuide:  "harness 标准编辑工具：小改动（≤5 行）用精确替换；大改动请用 write 写整段。替换前会自动快照。",
			category:    "文件",
		},
		{
			alias: "glob", source: "search_files",
			description: "按通配符递归查找文件，返回相对路径列表（对齐 deepseek-harness glob）。pattern 含 / 或 ** 时按路径模式（如 internal/**/*.go），否则匹配任意深度文件名（如 *.go）；path 限定子目录。",
			usageGuide:  "harness 标准 glob 工具：按路径模式发现文件。跳过 .git/node_modules 等目录。比 shell find 更精确（结构化、防撑爆）。",
			category:    "代码搜索",
		},
		{
			alias: "grep", source: "search_content",
			description: "在工作区内按正则搜索文件内容，返回「相对路径:行号: 行文本」（对齐 deepseek-harness grep）。pattern 为 RE2 正则；path 限定子目录；glob 按文件名过滤；case_insensitive 忽略大小写。",
			usageGuide:  "harness 标准 grep 工具：正则全文搜索。搜索函数/类型定义请优先用 codegraph_search（AST 级更精确）。",
			category:    "代码搜索",
		},
		{
			alias: "bash", source: "run_command",
			description: "同步执行一条 shell 命令并返回输出（对齐 deepseek-harness bash）。每次调用在独立 shell 中运行（无状态持久）。禁止用于长期进程（dev server/watch/tcp 监听）——请用 run_background。",
			usageGuide:  "harness 标准 bash 工具：执行命令（构建/测试/查询等短命令）。120s 超时自动终止。长期进程用 run_background/read_output/kill_process。",
			category:    "执行",
		},
	}
	for _, a := range aliases {
		src, ok := r.Get(a.source)
		if !ok {
			continue // 旧工具未注册（理论不发生，防御）
		}
		r.Register(&Tool{
			Name:             a.alias,
			Description:      a.description,
			UsageGuide:       a.usageGuide,
			Category:         a.category,
			Parameters:       src.Parameters,
			Handler:          src.Handler,
			ReadOnly:         src.ReadOnly,
			RequiresApproval: src.RequiresApproval,
		})
	}
}

// ─── str_replace_editor（对齐 tool-str-replace-editor）──────────────

// strReplaceEditorDesc 对齐 harness 的命令式编辑器描述。
const strReplaceEditorDesc = "Custom editing tool for viewing, creating and editing files（对齐 deepseek-harness str_replace_editor）\n" +
	"* `command` 必填：view / create / str_replace / insert\n" +
	"* `view` 显示文件内容（带行号）；path 为目录时列出非隐藏文件/目录最多 2 层\n" +
	"* `create` 创建新文件（path 已存在则报错）；内容在 `file_text`\n" +
	"* `str_replace` 把 `old_str` 替换为 `new_str`——old_str 必须精确匹配且唯一（含空白！不唯一则拒绝）\n" +
	"* `insert` 在 `insert_line` 之后插入 `new_str`\n" +
	"* `view` 支持 `view_range` 数组限定行范围（如 [11,12]，[-1] 到文件尾）\n" +
	"* 长输出会被截断并标记 `<response clipped>`"

func registerStrReplaceEditor(r *Registry, root string) {
	r.Register(&Tool{
		Name:        "str_replace_editor",
		Description: strReplaceEditorDesc,
		UsageGuide: "harness 标准命令式编辑器（Claude 系工具）：view 查看、create 创建、str_replace 精确替换（唯一匹配）、insert 行后插入。" +
			"与 edit_file 相比更适合『需要先查看行号、再精确替换』的流程；带行号输出方便后续定位。",
		Category: "文件",
		Parameters: objSchema(props{
			"command":      strProp("要执行的命令：view / create / str_replace / insert（必填）"),
			"path":         strProp("文件或目录路径（工作区内；view 支持目录，其他命令须为文件）"),
			"file_text":    strProp("create 命令的文件内容"),
			"insert_line":  intProp("insert 命令：在此行之后插入 new_str（1 基）"),
			"new_str":      strProp("str_replace 的替换新内容 / insert 的插入内容"),
			"old_str":      strProp("str_replace 的原文（须精确且唯一匹配）"),
			"view_range":   map[string]any{"type": "array", "items": map[string]any{"type": "integer"}, "description": "view 命令行范围，如 [11,12]；[-1] 表示到文件尾"},
		}, "command", "path"),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			command := argStr(args, "command")
			p, err := resolvePath(root, argStr(args, "path"))
			if err != nil {
				return "", err
			}
			switch command {
			case "view":
				return sreView(root, p, argIntSlice(args, "view_range"), ctx)
			case "create":
				return sreCreate(p, argStr(args, "file_text"), args, root)
			case "str_replace":
				return sreReplace(p, argStr(args, "old_str"), argStr(args, "new_str"), args, root)
			case "insert":
				return sreInsert(p, argInt(args, "insert_line", 0), argStr(args, "new_str"), args, root)
			default:
				return "", fmt.Errorf("无效 command %q：应为 view/create/str_replace/insert", command)
			}
		},
		RequiresApproval: true,
	})
}

// sreView 实现 str_replace_editor 的 view 命令：带行号显示文件，或列目录（最多 2 层）。
func sreView(root, p string, viewRange []int, ctx context.Context) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	info, err := os.Stat(p)
	if err != nil {
		return "", fmt.Errorf("路径 %s 不存在：%w", displayPath(root, p), err)
	}
	if info.IsDir() {
		if len(viewRange) > 0 {
			return "", fmt.Errorf("path 指向目录时不允许 view_range")
		}
		rows, err := sreListDir(p, 0)
		if err != nil {
			return "", err
		}
		sort.Strings(rows)
		out := fmt.Sprintf("以下是 %s 中最多 2 层深度的文件/目录（排除隐藏项、node_modules）：\n%s\n",
			displayPath(root, p), strings.Join(rows, "\n"))
		return capOutput(out, 16000), nil
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return "", err
	}
	content := string(data)
	allLines := strings.Split(content, "\n")
	// 去掉末尾空行（split 产生的多余项）
	if len(allLines) > 0 && allLines[len(allLines)-1] == "" {
		allLines = allLines[:len(allLines)-1]
	}
	initial, final := 1, len(allLines)
	prompt := fmt.Sprintf("以下是 %s 的内容（共 %d 行）：", displayPath(root, p), len(allLines))
	if len(viewRange) == 2 {
		initial, final = viewRange[0], viewRange[1]
		if initial < 1 || initial > len(allLines) {
			return "", fmt.Errorf("view_range 首元素 %d 超出文件行范围 [1, %d]", initial, len(allLines))
		}
		if final > len(allLines) {
			return "", fmt.Errorf("view_range 次元素 %d 超出文件总行数 %d", final, len(allLines))
		}
		if final != -1 && final < initial {
			return "", fmt.Errorf("view_range 次元素 %d 应 >= 首元素 %d", final, initial)
		}
		if final == -1 {
			final = len(allLines)
		}
		prompt += fmt.Sprintf("（view_range=[%d, %d]）", initial, final)
	}
	lines := allLines[initial-1 : final]
	var b strings.Builder
	b.WriteString(prompt + ":\n")
	for i, line := range lines {
		fmt.Fprintf(&b, "%6d  %s\n", initial+i, line)
	}
	b.WriteString("\n")
	return capOutput(b.String(), 16000), nil
}

// sreListDir 递归列目录（深度 ≤ 2），排除隐藏项/node_modules/__pycache__。
func sreListDir(p string, depth int) ([]string, error) {
	entries, err := os.ReadDir(p)
	if err != nil {
		return nil, err
	}
	var rows []string
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") || name == "node_modules" || name == "__pycache__" {
			continue
		}
		rel := filepath.ToSlash(name)
		if e.IsDir() {
			rows = append(rows, "d\t"+rel)
			if depth < 2 {
				sub, err := sreListDir(filepath.Join(p, name), depth+1)
				if err != nil {
					continue
				}
				for _, s := range sub {
					// s 形如 "d\tsub" / "f\tfile"，拼接父目录前缀
					sep := strings.IndexByte(s, '\t')
					if sep < 0 {
						continue
					}
					rows = append(rows, s[:sep]+"\t"+filepath.ToSlash(filepath.Join(name, s[sep+1:])))
				}
			}
		} else {
			rows = append(rows, "f\t"+rel)
		}
	}
	return rows, nil
}

// sreCreate 实现 create 命令：创建新文件（path 已存在则拒绝）。
func sreCreate(p, fileText string, args map[string]any, root string) (string, error) {
	if _, err := os.Stat(p); err == nil {
		return "", fmt.Errorf("create 失败：%s 已存在。要修改请用 str_replace/insert 或 edit 工具", displayPath(root, p))
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(p, []byte(fileText), 0o644); err != nil {
		return "", err
	}
	if FileChangeCallback != nil {
		FileChangeCallback(argStr(args, "path"))
	}
	return fmt.Sprintf("文件 %s 已创建（%d 字节）", displayPath(root, p), len(fileText)), nil
}

// sreReplace 实现 str_replace 命令：old_str 精确且唯一匹配替换。
func sreReplace(p, oldStr, newStr string, args map[string]any, root string) (string, error) {
	if oldStr == "" {
		return "", fmt.Errorf("str_replace 命令需要非空 old_str")
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return "", err
	}
	content := string(data)
	n := strings.Count(content, oldStr)
	if n == 0 {
		return "", fmt.Errorf("str_replace 失败：old_str 在 %s 中未找到。请用 view 查看实际内容（注意空白/缩进要精确匹配）", displayPath(root, p))
	}
	if n > 1 {
		return "", fmt.Errorf("str_replace 失败：old_str 在 %s 中出现 %d 次，不唯一。请在 old_str 中加上更多上下文使其唯一", displayPath(root, p), n)
	}
	out := strings.Replace(content, oldStr, newStr, 1)
	if err := writeFileWithSnapshot(root, p, out); err != nil {
		return "", err
	}
	if FileChangeCallback != nil {
		FileChangeCallback(argStr(args, "path"))
	}
	return fmt.Sprintf("文件 %s 已编辑（str_replace 成功，替换 1 处）", displayPath(root, p)), nil
}

// sreInsert 实现 insert 命令：在 insert_line 之后插入 new_str。
func sreInsert(p string, insertLine int, newStr string, args map[string]any, root string) (string, error) {
	if insertLine < 1 {
		return "", fmt.Errorf("insert 命令需要 insert_line >= 1（1 基）")
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return "", err
	}
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	if insertLine > len(lines) {
		return "", fmt.Errorf("insert 失败：insert_line %d 超出文件总行数 %d", insertLine, len(lines))
	}
	var b strings.Builder
	for i, line := range lines {
		b.WriteString(line)
		if i < len(lines)-1 {
			b.WriteString("\n")
		}
		if i+1 == insertLine {
			b.WriteString(newStr)
			if !strings.HasSuffix(newStr, "\n") {
				b.WriteString("\n")
			}
		}
	}
	if err := writeFileWithSnapshot(root, p, b.String()); err != nil {
		return "", err
	}
	if FileChangeCallback != nil {
		FileChangeCallback(argStr(args, "path"))
	}
	return fmt.Sprintf("文件 %s 已编辑（在第 %d 行后插入 %d 字符）", displayPath(root, p), insertLine, len(newStr)), nil
}

// writeFileWithSnapshot 带快照写文件（复用全局快照机制）。
func writeFileWithSnapshot(root, p, content string) error {
	SnapshotBeforeWriteWithTracking(root, p)
	return os.WriteFile(p, []byte(content), 0o644)
}

// displayPath 显示工作区相对路径（稳定、可读）。
func displayPath(root, p string) string {
	rel, err := filepath.Rel(root, p)
	if err != nil {
		return filepath.ToSlash(p)
	}
	return filepath.ToSlash(rel)
}

// ─── run_code（对齐 code-mode）─────────────────────────────────

// registerRunCode 注册 run_code 工具：执行一段代码（对齐 harness Code Mode）。
// Go 侧简化版：把 code 写入临时文件，按语言执行并返回 stdout/stderr。
// 不嵌套工具调用（harness 的嵌套调度在自举阶段再补）。
func registerRunCode(r *Registry, root string) {
	r.Register(&Tool{
		Name: "run_code",
		Description: "执行一段代码并返回输出（对齐 deepseek-harness run_code / Code Mode）。" +
			"参数：code（必填，要执行的程序体）、language（可选，auto/go/python/node，默认 auto 按内容探测）、" +
			"description（可选，简短说明）。仅返回程序的 stdout/stderr 与退出状态。",
		UsageGuide: "harness 标准代码执行工具：快速验证算法/处理数据/调用本地库，不用写临时文件。" +
			"与 bash 的区别：run_code 直接执行代码片段（自动建临时文件），bash 执行 shell 命令。",
		Category: "执行",
		Parameters: objSchema(props{
			"code":        strProp("要执行的代码（必填）"),
			"language":    strProp("可选：auto（默认，按内容探测）/ go / python / node"),
			"description": strProp("可选：简短说明这段代码做什么"),
		}, "code"),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			code := argStr(args, "code")
			if strings.TrimSpace(code) == "" {
				return "", fmt.Errorf("code 不能为空")
			}
			lang := argStr(args, "language")
			if lang == "" || lang == "auto" {
				lang = detectCodeLang(code)
			}
			return runCodeSnippet(ctx, root, lang, code)
		},
	})
}

// detectCodeLang 按代码内容探测语言（auto 模式）。
func detectCodeLang(code string) string {
	trimmed := strings.TrimSpace(code)
	switch {
	case strings.HasPrefix(trimmed, "package main") || strings.HasPrefix(trimmed, "package "):
		return "go"
	case strings.HasPrefix(trimmed, "import ") || strings.HasPrefix(trimmed, "def ") ||
		strings.HasPrefix(trimmed, "print(") || strings.Contains(trimmed, "\nprint("):
		return "python"
	case strings.HasPrefix(trimmed, "const ") || strings.HasPrefix(trimmed, "let ") ||
		strings.HasPrefix(trimmed, "var ") || strings.HasPrefix(trimmed, "function ") ||
		strings.HasPrefix(trimmed, "console.log"):
		return "node"
	default:
		return "go" // 默认 Go（项目主语言，自举迭代最常用）
	}
}

// runCodeSnippet 执行代码片段：写临时文件 → 运行 → 清理。
func runCodeSnippet(ctx context.Context, root, lang, code string) (string, error) {
	tmp, err := os.MkdirTemp("", "run_code_*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmp)

	var file, cmd string
	switch lang {
	case "go":
		file = filepath.Join(tmp, "main.go")
		if err := os.WriteFile(file, []byte(code), 0o644); err != nil {
			return "", err
		}
		cmd = "go run " + file
	case "python":
		file = filepath.Join(tmp, "main.py")
		if err := os.WriteFile(file, []byte(code), 0o644); err != nil {
			return "", err
		}
		cmd = "python " + file
	case "node":
		file = filepath.Join(tmp, "main.js")
		if err := os.WriteFile(file, []byte(code), 0o644); err != nil {
			return "", err
		}
		cmd = "node " + file
	default:
		return "", fmt.Errorf("不支持的语言 %q（可用 go/python/node）", lang)
	}

	out, exitErr := runShellWithTimeout(ctx, cmd, tmp)
	res := capOutput(out, 16000)
	if exitErr != "" {
		res += "\n[退出: " + exitErr + "]"
	}
	return res, nil
}

// runShellWithTimeout 执行 shell 命令并返回输出（120s 超时 + ctx 取消）。
// 复用全局后台进程机制（不阻塞 loop 线程）。
func runShellWithTimeout(ctx context.Context, command, dir string) (string, string) {
	bg := globalBG
	id, err := bg.start(command, dir)
	if err != nil {
		return "", err.Error()
	}
	p := bg.get(id)
	if p == nil {
		return "", "内部错误：后台进程创建后丢失"
	}
	deadline := time.After(120 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			if p.cmd != nil && p.cmd.Process != nil {
				killProcessTree(p.cmd.Process.Pid)
			}
			out, _, _ := p.snapshot()
			return capOutput(out, 16000), ctx.Err().Error()
		case <-deadline:
			if p.cmd != nil && p.cmd.Process != nil {
				killProcessTree(p.cmd.Process.Pid)
			}
			out, _, _ := p.snapshot()
			return capOutput(out, 16000), "超时 120s 已终止"
		case <-ticker.C:
			out, done, exitErr := p.snapshot()
			if done {
				bg.mu.Lock()
				delete(bg.procs, id)
				bg.mu.Unlock()
				return capOutput(out, 16000), exitErr
			}
		}
	}
}
