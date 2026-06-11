package agent

// 审核 Agent（多 Agent 编排之二）—— 忠实复刻参考 prompts/roles/reviewer.md + agents/reviewer.ts。
// 用审核模型(reviewModel, non-thinking)在执行前把关写类工具调用：安全/编码/结构/兼容；
// 关键文件删除直接驳回。companion 把它接进 Loop.Approve（AI 审核模式），驳回把建议回灌让执行 Agent 改道。

import (
	"context"
	"encoding/json"
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

// DefaultReviewerPrompt 内置默认审核角色提示（config/roles/reviewer.md 缺失时的回退）。
func DefaultReviewerPrompt() string { return reviewerSystemPrompt }

// reviewerSystemPrompt 复刻参考 prompts/roles/reviewer.md（角色/四层审核/输出格式/决策标准/规则）。
const reviewerSystemPrompt = `# 角色
你是审核 Agent（Reviewer）——代码变更前的最后一道防线。审核代码变更、Shell 命令的安全性、正确性和一致性，拥有一票否决权。

# 审核层次
## Shell 命令
严格审查：破坏性操作（rm -rf、force push、hard reset、drop table、format、del /f /s 等）；路径穿越或访问系统关键目录；编码风险（Windows cmd.exe 中文 echo/type 可能乱码；PowerShell 未指定 -Encoding 可能改变编码）。
## 文件操作（编码感知）
安全性：是否引入注入攻击、XSS、路径穿越等漏洞；编码处理：.bat/.cmd 中文须 GBK，.ps1 中文建议 UTF-8 BOM；结构完整性：是否破坏 JSON/XML/YAML 结构、删除关键配置；向后兼容：是否影响已有 API 接口、配置文件格式。
## 删除操作（关键文件保护）
package.json、go.mod、.env、CLAUDE.md、AGENTS.md、Dockerfile、.gitignore 等关键文件直接驳回。

# 输出格式
严格 JSON（不要额外文本）：
{"verdict":"通过"|"驳回"|"需要修改","confidence":0.0-1.0,"issues":[{"severity":"严重"|"重要"|"轻微","description":"问题描述","location":"文件:行号"}],"suggestions":["改进建议"],"summary":"审核结论一句话总结"}

# 决策标准
- 无安全隐患且变更合理 → 通过
- 存在安全问题或触发关键文件保护 → 驳回
- 需调整但不严重 → 需要修改 + 具体建议
- 所有审核输出使用中文`

// criticalFiles 关键文件：删除直接驳回（复刻参考，按 companion 项目栈调整：go.mod/go.sum 取代 tsconfig 等）。
var criticalFiles = map[string]bool{
	"package.json": true, "go.mod": true, "go.sum": true, ".env": true,
	"claude.md": true, "agents.md": true, "dockerfile": true,
	"docker-compose.yml": true, ".gitignore": true,
}

// NeedsReview 是否需要审核：只读工具放行，仅写类（写/改/删/移/运行命令）过审。
func NeedsReview(toolName string) bool {
	switch toolName {
	case "write_file", "edit_file", "multi_edit", "move_file", "delete_file", "run_command", "run_background":
		return true
	}
	return strings.HasPrefix(toolName, "git_") && !strings.Contains(toolName, "status") &&
		!strings.Contains(toolName, "diff") && !strings.Contains(toolName, "log") &&
		!strings.Contains(toolName, "show") && !strings.Contains(toolName, "blame")
}

// Review 审核一次写类工具调用。关键文件删除→直接驳回（无需 LLM）；否则交审核模型判（JSON 裁决）。
func (r *Reviewer) Review(ctx context.Context, tc ToolCall) (ReviewVerdict, error) {
	var args map[string]any
	_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)
	name := tc.Function.Name
	path, _ := args["path"].(string)

	// 关键文件保护（复刻参考 审核删除操作）：删除关键文件直接驳回，省一次 LLM 调用。
	if strings.Contains(name, "delete") && criticalFiles[strings.ToLower(baseName(path))] {
		return ReviewVerdict{Verdict: "驳回", Confidence: 1,
			Summary:     "驳回：删除关键文件 " + baseName(path) + " 需人工确认",
			Suggestions: []string{"如确需删除，请手动操作并确认影响范围", "先检查依赖关系再执行删除"}}, nil
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

// reviewUserPrompt 据工具构造审核提示（复刻参考 审核命令 / 审核文件操作）。
func reviewUserPrompt(name string, args map[string]any) string {
	if name == "run_command" || name == "run_background" {
		cmd, _ := args["command"].(string)
		return "[审核：Shell 命令]\n命令：" + cmd + "\n\n请严格检查：\n" +
			"1. 破坏性操作（rm -rf、force push、hard reset、format、del /f /s 等）\n" +
			"2. 会修改项目外系统状态的命令\n3. 路径穿越或访问系统关键目录\n" +
			"4. 编码风险（cmd.exe 中文乱码 / PowerShell 未指定 -Encoding）\n\n以 JSON 格式输出审核结果。"
	}
	path, _ := args["path"].(string)
	content, _ := args["content"].(string)
	if content == "" {
		content, _ = args["old_string"].(string)
	}
	op := "写入（新建/覆盖）"
	if strings.Contains(name, "edit") {
		op = "编辑（字符串替换）"
	} else if strings.Contains(name, "delete") {
		op = "删除"
	} else if strings.Contains(name, "move") {
		op = "移动/重命名"
	}
	return "[审核：代码变更]\n文件：" + path + "\n操作：" + op + "\n内容预览：" + truncRunesAgent(content, 500) +
		"\n\n请严格检查：\n1. 安全性（注入/XSS/路径穿越）\n2. 编码处理（.bat/.cmd 须 GBK；.ps1 建议 UTF-8 BOM）\n" +
		"3. 结构完整性（JSON/XML/YAML 不被破坏）\n4. 向后兼容（不破坏已有 API/配置格式）\n5. 错误处理\n\n以 JSON 格式输出审核结果。"
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
