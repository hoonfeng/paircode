package agent

// 审核 Agent（多 Agent 编排之二）—— 忠实复刻参考 prompts/roles/reviewer.md + agents/reviewer.ts。
// 用审核模型(reviewModel, non-thinking)在执行前把关写类工具调用：安全/编码/结构/兼容；
// 关键文件删除直接驳回。companion 把它接进 Loop.Approve（AI 审核模式），驳回把建议回灌让执行 Agent 改道。

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// ReviewVerdict 审核裁决（复刻参考 reviewer.ts ReviewResult 的核心字段）。
type ReviewVerdict struct {
	Verdict     string   `json:"verdict"` // 通过 / 驳回 / 需要修改
	Confidence  float64  `json:"confidence"`
	Suggestions []string `json:"suggestions"`
	Summary     string   `json:"summary"`
}

// Approved 是否放行：仅「通过」放行（驳回/需要修改/解析失败都拦住）。
func (v ReviewVerdict) Approved() bool { return v.Verdict == "通过" }

// FeedbackText 驳回时回灌给执行 Agent 的反馈（结论 + 建议），让它据此改道。
func (v ReviewVerdict) FeedbackText() string {
	var b strings.Builder
	b.WriteString("审核未通过（" + orDefault(v.Verdict, "需要修改") + "）：" + orDefault(v.Summary, "存在风险，请改用更安全的方式"))
	for _, s := range v.Suggestions {
		if strings.TrimSpace(s) != "" {
			b.WriteString("\n- " + s)
		}
	}
	return b.String()
}

// Reviewer 审核 Agent。Provider 用审核模型（建议 non-thinking）。
type Reviewer struct {
	Provider     Provider
	SystemPrompt string // 角色系统提示（空=用内置默认）；宿主可从 config 加载覆盖（非硬编码）
}

// DefaultReviewerPrompt 审核角色提示（磁盘优先：config/roles/reviewer.md 覆盖；
// 缺失/不可读时回退内置 reviewerSystemPrompt）。★ t1 C1/C2 闭环：角色内容在磁盘，
// 不重编译即可覆盖审核角色设定。
func DefaultReviewerPrompt() string {
	if s := LoadRolePrompt("reviewer"); s != "" {
		return s
	}
	return reviewerSystemPrompt
}

// reviewerSystemPrompt 复刻参考 prompts/roles/reviewer.md（角色/四层审核/输出格式/决策标准/规则）。
const reviewerSystemPrompt = `# 角色
你是审核 Agent（Reviewer）——代码变更前的最后一道防线。审核代码变更、Shell 命令的安全性、正确性和一致性，拥有一票否决权。

` + AIIdentityAwareness + `# 审核层次
## Shell 命令
严格审查：破坏性操作（rm -rf、force push、hard reset、drop table、format、del /f /s 等）；路径穿越或访问系统关键目录；编码风险（Windows cmd.exe 中文 echo/type 可能乱码；PowerShell 未指定 -Encoding 可能改变编码）。
### 阻塞与后台运行（bash 专用）
检查 bash 是否在执行一个长期运行的命令（如 dev server、watch、go run 启动服务、npm run dev 等）。此类命令会阻塞 agent 循环 120s 后超时终止，严重影响 agent 正常运行。发现此类情况应驳回并建议改用 run_background。
反过来，run_background 用于长命令，短查询则用 bash。
## 文件操作（编码感知）
安全性：是否引入注入攻击、XSS、路径穿越等漏洞；编码处理：.bat/.cmd 中文须 GBK，.ps1 中文建议 UTF-8 BOM；结构完整性：是否破坏 JSON/XML/YAML 结构、删除关键配置；向后兼容：是否影响已有 API 接口、配置文件格式。
## 删除操作（关键文件保护）
package.json、go.mod、.env、CLAUDE.md、AGENTS.md、Dockerfile、.gitignore 等关键文件直接驳回。

# 输出格式
严格 JSON（不要额外文本）：
{"verdict":"通过"|"驳回"|"需要修改","confidence":0.0-1.0,"issues":[{"severity":"严重"|"重要"|"轻微","description":"问题描述","location":"文件:行号"}],"suggestions":["改进建议"],"summary":"审核结论一句话总结"}

# 决策标准
- 无安全隐患且变更合理，与当前任务上下文一致 → 通过
- Agent 正按照用户的任务要求修改文件/运行命令 → 除非有明确的安全风险，否则应放行
- 存在安全问题或触发关键文件保护 → 驳回
- 需调整但不严重 → 需要修改 + 具体建议
- 所有审核输出使用中文`

// criticalFiles 关键文件：删除直接驳回（复刻参考，按 companion 项目栈调整：go.mod/go.sum 取代 tsconfig 等）。
var criticalFiles = map[string]bool{
	"package.json": true, "go.mod": true, "go.sum": true, ".env": true,
	"claude.md": true, "agents.md": true, "dockerfile": true,
	"docker-compose.yml": true, ".gitignore": true,
}

// NeedsReview 是否需要审核：只读工具放行，仅写类（写/改/删/移/运行命令/后台命令）过审。
func NeedsReview(toolName string) bool {
	switch toolName {
	// ★ Round3：新名（write/edit/bash）补入；旧名保留历史消息兼容
	// ★ 2026-09 工具重构：exec_command 取代 bash（会话式；yield 超时自动转会话）
	// ★ 2026-09 Round5：apply_patch 取代 edit/multi_edit（旧名保留历史消息兼容）
	case "write", "apply_patch", "edit", "bash", "exec_command", "write_file", "edit_file", "multi_edit", "move_file", "delete_file", "run_command":
		return true
	}
	// run_background/kill_process 管理自己启动的后台进程，是安全的进程管理，无需审核。
	// llm_* MCP 工具由 MCP 服务器自行鉴权，不经过 agent 审核。
	return strings.HasPrefix(toolName, "git_") && !strings.Contains(toolName, "status") &&
		!strings.Contains(toolName, "diff") && !strings.Contains(toolName, "log") &&
		!strings.Contains(toolName, "show") && !strings.Contains(toolName, "blame")
}

// isSafeShellCommand 判断 shell 命令是否安全，无须审核。
// 安全命令：构建、测试、开发服务器、包安装、格式检查、查询、简单文件操作。
// 破坏性命令：删除/格式化/重置/强推/磁盘操作等才需审核。
func isSafeShellCommand(command string) bool {
	cmd := strings.TrimSpace(command)
	// 常见安全前缀
	safePrefixes := []string{
		"go build", "go test", "go vet", "go fmt", "go run", "go mod",
		"npm run", "npm test", "npm install", "npm ci", "npm audit", "npm start",
		"npx tsc", "npx vite", "npx webpack",
		"vite build", "tsc --noEmit",
		"git status", "git diff", "git log", "git show", "git blame",
		"dir", "ls ", "echo ", "type ", "cd ", "chcp",
		"pnpm", "yarn",
		"chdir", "node ", "python", "dotnet build", "cargo build", "cargo test",
	}
	for _, p := range safePrefixes {
		if strings.HasPrefix(cmd, p) {
			return true
		}
	}
	return false
}

// Review 审核一次写类工具调用。
func (r *Reviewer) Review(ctx context.Context, tc ToolCall) (ReviewVerdict, error) {
	var args map[string]any
	_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)
	name := tc.Function.Name

	// exec_command/bash/run_command 安全命令智能放行（无需 LLM 审核；旧名 bash 并存兼容）
	if (name == "run_command" || name == "bash" || name == "exec_command") && isSafeShellCommand(argStr(args, "command")) && !isBlockingCommand(argStr(args, "command")) {
		return ReviewVerdict{Verdict: "通过", Confidence: 1, Summary: "安全检查通过：普通构建/测试/查询命令"}, nil
	}
	if (name == "run_background" || name == "exec_command") && isSafeShellCommand(argStr(args, "command")) {
		return ReviewVerdict{Verdict: "通过", Confidence: 1, Summary: "安全检查通过：后台/长驻运行安全命令（yield 超时自动转会话）"}, nil
	}
	// kill_process 仅杀死自己启动的后台进程，是安全的进程管理操作，不需要 LLM 审核。
	if name == "kill_process" {
		return ReviewVerdict{Verdict: "通过", Confidence: 1, Summary: "安全检查通过：后台进程管理"}, nil
	}
	// move_file 仅在工作区内重命名/移动文件，不改变内容也不破坏文件结构，直接放行。
	if name == "move_file" {
		return ReviewVerdict{Verdict: "通过", Confidence: 1, Summary: "安全通过：文件移动"}, nil
	}

	path, _ := args["path"].(string)

	// ★ Round5：apply_patch 含 "*** Delete File:" 时做关键文件保护（确定性拦截；
	//   其余场景走 LLM 审核——patch 内容已随 reviewUserPrompt 提交）。
	if name == "apply_patch" {
		for _, ln := range strings.Split(argStr(args, "patch"), "\n") {
			ln = strings.TrimSpace(ln)
			if strings.HasPrefix(ln, "*** Delete File:") {
				dp := strings.TrimSpace(strings.TrimPrefix(ln, "*** Delete File:"))
				if criticalFiles[strings.ToLower(baseName(dp))] {
					return ReviewVerdict{Verdict: "驳回", Confidence: 1,
						Summary:     "驳回：删除关键文件 " + baseName(dp) + " 需人工确认",
						Suggestions: []string{"如确需删除，请手动操作并确认影响范围", "先检查依赖关系再执行删除"}}, nil
				}
			}
		}
	}

	// 关键文件保护：删除关键文件直接驳回
	if strings.Contains(name, "delete") {
		if criticalFiles[strings.ToLower(baseName(path))] {
			return ReviewVerdict{Verdict: "驳回", Confidence: 1,
				Summary:     "驳回：删除关键文件 " + baseName(path) + " 需人工确认",
				Suggestions: []string{"如确需删除，请手动操作并确认影响范围", "先检查依赖关系再执行删除"}}, nil
		}
		// 非关键文件删除（Agent 创建的临时/备份文件清理），直接放行。
		// 关键文件已在上方保护；其余文件 Agent 有权管理其创建的文件。
		return ReviewVerdict{Verdict: "通过", Confidence: 1, Summary: "安全通过：非关键文件删除"}, nil
	}

	resp, err := r.Provider.Chat(ctx, []Message{
		{Role: RoleSystem, Content: orDefault(r.SystemPrompt, reviewerSystemPrompt)},
		{Role: RoleUser, Content: reviewUserPrompt(name, args)},
	}, nil, nil)
	if err != nil {
		return ReviewVerdict{}, err
	}
	return parseVerdict(resp.Content), nil
}

// reviewUserPrompt 构造送审 prompt（审核模型看到的唯一输入）。
//
// ★ 2026-09 修复（三方 agentloop 补丁 P2/P3 的宿主侧根治版）：原先只读
//
//	args["path"] / args["content"]（content 空再回退 old_string），而现代工具
//	与插件的参数名并非如此——write 用 file_path、edit 用 new_string / edits[]、
//	git_* 用 message / files / target / action —— 送审 prompt 于是成了
//	「路径空 + 内容空」，审核模型看不到真实变更，只能保守驳回
//	（git 类工具实测驳回率 100%）。现统一做「送审字段归一化」：
//	  path    ← path / file_path / target / file / files[] / 补丁文件头
//	  content ← content / new_string / edits[] / patch / old_string
//	  git_*   ← project / 仓库根兜底 path，关键参数汇总为 content
//	只读 args，不修改 tc——执行路径（loop.tools.run）零副作用。
func reviewUserPrompt(name string, args map[string]any) string {
	if name == "run_command" || name == "run_background" || name == "bash" || name == "exec_command" {
		cmd, _ := args["command"].(string)
		return "[审核：Shell 命令]\n命令：" + cmd + "\n\n请严格检查：\n" +
			"1. 破坏性操作（rm -rf、force push、hard reset、format、del /f /s 等）\n" +
			"2. 会修改项目外系统状态的命令\n3. 路径穿越或访问系统关键目录\n" +
			"4. 编码风险（cmd.exe 中文乱码 / PowerShell 未指定 -Encoding）\n" +
			"5. 【会话检查】长驻命令（dev server / watch / go run 服务 / npm run dev）由 exec_command 的 yield_time_ms 自动转会话（不阻塞）——如疑似长驻，确认是否合理\n\n以 JSON 格式输出审核结果。"
	}
	path := reviewPath(args)
	content := reviewContent(args)
	// git 类工具：参数是 project/message/files/target/action 等（无 path/content）
	// → path 用 project / 仓库根兜底，content 汇总关键参数，让审核器看到
	//   「操作类型 + 提交信息 + 目标对象」，而不是空串。
	if strings.HasPrefix(name, "git_") {
		if path == "" {
			if proj := strings.TrimSpace(argStr(args, "project")); proj != "" {
				path = "（项目）" + proj
			} else {
				path = "（git 仓库工作区）"
			}
		}
		if content == "" {
			content = gitArgsSummary(name, args)
		}
	}
	op := "写入（新建/覆盖）"
	switch {
	case name == "apply_patch":
		op = "编辑（补丁应用）"
	case strings.HasPrefix(name, "git_"):
		op = "Git 操作"
	case strings.Contains(name, "edit"):
		op = "编辑（字符串替换）"
	case strings.Contains(name, "delete"):
		op = "删除"
	case strings.Contains(name, "move"):
		op = "移动/重命名"
	}
	return "[审核：代码变更]\n文件：" + path + "\n操作：" + op + "\n内容预览：" + truncRunesAgent(content, 500) +
		"\n\n请严格检查：\n1. 安全性（注入/XSS/路径穿越）\n2. 编码处理（.bat/.cmd 须 GBK；.ps1 建议 UTF-8 BOM）\n" +
		"3. 结构完整性（JSON/XML/YAML 不被破坏）\n4. 向后兼容（不破坏已有 API/配置格式）\n5. 错误处理\n\n以 JSON 格式输出审核结果。"
}

// reviewPath 送审路径归一化（首个非空者胜出）：
// path → file_path → target → file → files[] → apply_patch 补丁里的首个文件头。
func reviewPath(args map[string]any) string {
	for _, key := range []string{"path", "file_path", "target", "file"} {
		if s := strings.TrimSpace(argStr(args, key)); s != "" {
			return s
		}
	}
	if files := reviewStrList(args, "files"); len(files) > 0 {
		return strings.Join(files, ", ")
	}
	return applyPatchFirstPath(argStr(args, "patch"))
}

// reviewContent 送审内容归一化（首个非空者胜出）：
// content → new_string → edits[]（各段新代码）→ patch（补丁正文）→ old_string（旧行为保留）。
func reviewContent(args map[string]any) string {
	if s := argStr(args, "content"); s != "" {
		return s
	}
	if s := argStr(args, "new_string"); s != "" {
		return s
	}
	if s := editsReviewText(args); s != "" {
		return s
	}
	if s := argStr(args, "patch"); s != "" {
		return s
	}
	return argStr(args, "old_string")
}

// applyPatchFirstPath 从 codex 补丁文本提取首个文件头路径
// （*** Add File: / *** Update File: / *** Delete File:）——补丁载荷里才有目标路径。
func applyPatchFirstPath(patchStr string) string {
	if patchStr == "" {
		return ""
	}
	for _, raw := range strings.Split(patchStr, "\n") {
		ln := strings.TrimSpace(raw)
		for _, pfx := range []string{"*** Add File:", "*** Update File:", "*** Delete File:"} {
			if strings.HasPrefix(ln, pfx) {
				if p := strings.TrimSpace(strings.TrimPrefix(ln, pfx)); p != "" {
					return p
				}
			}
		}
	}
	return ""
}

// editsReviewText 把 edits[] 各段新代码拼成送审文本（multi_edit 类工具的兜底；
// 无 new_string 的条目跳过）。
func editsReviewText(args map[string]any) string {
	raw, ok := args["edits"].([]any)
	if !ok || len(raw) == 0 {
		return ""
	}
	var b strings.Builder
	for i, item := range raw {
		m, _ := item.(map[string]any)
		if m == nil {
			continue
		}
		ns, _ := m["new_string"].(string)
		if ns == "" {
			continue
		}
		b.WriteString("#edit" + strconv.Itoa(i+1) + ":\n" + ns + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

// reviewStrList 取字符串列表参数（兼容 JSON 解出的 []any 与宿主侧 []string；非数组返回 nil）。
func reviewStrList(args map[string]any, key string) []string {
	switch v := args[key].(type) {
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, s)
			}
		}
		return out
	case []string:
		out := make([]string, 0, len(v))
		for _, s := range v {
			if strings.TrimSpace(s) != "" {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

// gitArgsSummary 汇总 git 类工具的关键参数（送审 content 兜底：git 操作无文件内容
// 变更，让审核器仍能判断「操作类型 + 提交信息 + 目标对象」）。
func gitArgsSummary(name string, args map[string]any) string {
	kv := make([]string, 0, 12)
	addVal := func(key string, val any) {
		switch v := val.(type) {
		case []any:
			ss := make([]string, 0, len(v))
			for _, it := range v {
				ss = append(ss, fmt.Sprint(it))
			}
			kv = append(kv, key+"="+strings.Join(ss, ", "))
		case []string:
			kv = append(kv, key+"="+strings.Join(v, ", "))
		case string:
			s := v
			if key == "message" {
				s = truncRunesAgent(s, 2000)
			}
			kv = append(kv, key+"="+s)
		case nil:
			kv = append(kv, key+"=null")
		default:
			kv = append(kv, key+"="+fmt.Sprint(v))
		}
	}
	for _, key := range []string{"files", "message", "commit", "target", "name", "file", "count", "all", "staged", "action"} {
		if v, ok := args[key]; ok {
			addVal(key, v)
		}
	}
	if s, ok := args["start"]; ok {
		if e, ok2 := args["end"]; ok2 {
			kv = append(kv, "lines="+fmt.Sprint(s)+"-"+fmt.Sprint(e))
		}
	}
	if v, ok := args["project"]; ok {
		addVal("project", v)
	}
	if len(kv) == 0 {
		kv = append(kv, "(无)")
	}
	return "git 操作「" + name + "」参数：" + strings.Join(kv, "; ")
}

// parseVerdict 抽 JSON 裁决（首 { 到末 }）。解析失败→「需要修改」（复刻参考 fallback：不放行、提示人工）。
func parseVerdict(content string) ReviewVerdict {
	i, j := strings.IndexByte(content, '{'), strings.LastIndexByte(content, '}')
	if i >= 0 && j > i {
		var v ReviewVerdict
		if err := json.Unmarshal([]byte(content[i:j+1]), &v); err == nil && v.Verdict != "" {
			return v
		}
	}
	return ReviewVerdict{Verdict: "需要修改", Confidence: 0.5,
		Summary: "无法自动解析审核结果，建议人工审查", Suggestions: []string{"请人工确认此操作的安全性"}}
}

// baseName 取路径末段（跨平台，切 / 和 \）。
func baseName(p string) string {
	p = strings.TrimRight(p, `/\`)
	if i := strings.LastIndexAny(p, `/\`); i >= 0 {
		return p[i+1:]
	}
	return p
}

func orDefault(s, def string) string {
	if strings.TrimSpace(s) == "" {
		return def
	}
	return s
}
