// 记忆工具：memory_write/read/list/search —— 跨会话持久记忆，存在工作区 .pair/memory/ 下，
// 每条一个 .md（frontmatter: name/type/description + 正文）。让 agent 记住项目知识/用户偏好/教训。

package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

func memoryDir(root string) string { return filepath.Join(root, ".pair", "memory") }

// RecallMemories 任务开始时自动召回相关项目记忆（关键词/CJK 二元组重叠打分，取前 max 条），
// 格式化为上下文段附在任务后——让记忆从「Agent 主动查」升级为「自动召回」。无匹配返回 ""。
func RecallMemories(root, task string, max int) string {
	dir := memoryDir(root)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	toks := taskTokens(task)
	if len(toks) == 0 {
		return ""
	}
	type hit struct {
		name, desc string
		score      int
	}
	var hits []hit
	for _, e := range entries {
		if e.IsDir() || !isMemFile(e.Name()) {
			continue
		}
		data, _ := os.ReadFile(filepath.Join(dir, e.Name()))
		text := strings.ToLower(string(data))
		score := 0
		for tok := range toks {
			if strings.Contains(text, tok) {
				score++
			}
		}
		if score > 0 {
			hits = append(hits, hit{strings.TrimSuffix(e.Name(), ".md"), frontmatterField(string(data), "description"), score})
		}
	}
	if len(hits) == 0 {
		return ""
	}
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].score > hits[j].score })
	if len(hits) > max {
		hits = hits[:max]
	}
	var b strings.Builder
	b.WriteString("\n\n# 相关项目记忆（自动召回，供参考）\n")
	for _, h := range hits {
		b.WriteString("- 「" + h.name + "」：" + h.desc + "\n")
	}
	b.WriteString("（需要细节用 memory_read 读全文。）")
	return b.String()
}

// similarMemory 找与给定文本最相关的已有记忆名（关键词/二元组重叠 ≥3，排除 exclude）。无明显相关→""。
// 供 memory_write 新建时提醒「已有相关记忆，优先更新而非新建」，防记忆碎片化。
func similarMemory(dir, exclude, text string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	toks := taskTokens(text)
	if len(toks) < 3 {
		return ""
	}
	best, bestScore := "", 2 // 需 ≥3 个 token 重叠才算「相关」，避免噪声误报
	for _, e := range entries {
		if e.IsDir() || !isMemFile(e.Name()) {
			continue
		}
		nm := strings.TrimSuffix(e.Name(), ".md")
		if nm == exclude {
			continue
		}
		data, _ := os.ReadFile(filepath.Join(dir, e.Name()))
		lower := strings.ToLower(string(data))
		score := 0
		for tok := range toks {
			if strings.Contains(lower, tok) {
				score++
			}
		}
		if score > bestScore {
			best, bestScore = nm, score
		}
	}
	return best
}

// taskTokens 从任务文本提取检索 token：ASCII 词（长度>1，去停用词）+ CJK 二元组（中文无空格，按相邻 2 字切）。
func taskTokens(s string) map[string]bool {
	toks := map[string]bool{}
	var ascii, cjk []rune
	flushAscii := func() {
		if len(ascii) > 1 && !memStopWords[string(ascii)] {
			toks[string(ascii)] = true
		}
		ascii = ascii[:0]
	}
	flushCJK := func() {
		switch {
		case len(cjk) == 1:
			toks[string(cjk)] = true
		case len(cjk) >= 2:
			for i := 0; i+1 < len(cjk); i++ {
				toks[string(cjk[i:i+2])] = true
			}
		}
		cjk = cjk[:0]
	}
	for _, r := range strings.ToLower(s) {
		switch {
		case isCJK(r):
			flushAscii()
			cjk = append(cjk, r)
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			flushCJK()
			ascii = append(ascii, r)
		default:
			flushAscii()
			flushCJK()
		}
	}
	flushAscii()
	flushCJK()
	return toks
}

// memStopWords 常见停用词（不参与召回打分，避免噪声匹配）。
var memStopWords = map[string]bool{
	"the": true, "and": true, "for": true, "with": true, "you": true, "this": true, "that": true,
	"请": true, "帮我": true, "一下": true, "这个": true, "那个": true, "怎么": true, "如何": true,
}

// safeMemName 把名字里的路径危险字符(/ \ : . 空格)换成 -，防路径穿越；保留 CJK 等其它字符。
func safeMemName(s string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case '/', '\\', ':', '.', ' ':
			return '-'
		}
		return r
	}, strings.TrimSpace(s))
}

func frontmatterField(text, key string) string {
	for _, ln := range strings.Split(text, "\n") {
		if strings.HasPrefix(ln, key+":") {
			return strings.TrimSpace(strings.TrimPrefix(ln, key+":"))
		}
	}
	return ""
}

// isMemFile 是否为一条记忆文件（.md 且非索引 MEMORY.md 本身）。
func isMemFile(name string) bool {
	return strings.HasSuffix(name, ".md") && name != "MEMORY.md"
}

func memIndexPath(dir string) string { return filepath.Join(dir, "MEMORY.md") }

// MEMORY.md 索引模式（复刻参考）：总览 + 细节的渐进式披露。本表是总览，各条细节见对应文件。
const memIndexHeader = "# 记忆索引（总览）\n\n" +
	"> 类型：user 用户偏好 / feedback 纠正与确认 / project 项目决策 / reference 外部资源\n" +
	"> 渐进式披露：先看本总览，需要细则再读对应条目文件（memory_read）。\n\n"

func memIndexLine(name, desc string) string {
	if desc != "" {
		return "- [" + name + "](" + name + ".md) — " + desc
	}
	return "- [" + name + "](" + name + ".md)"
}

// genMemIndex 扫描目录现生成「记忆总览」内容（只读，不写盘）；无记忆→""。
func genMemIndex(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	var lines []string
	for _, e := range entries {
		if e.IsDir() || !isMemFile(e.Name()) {
			continue
		}
		data, _ := os.ReadFile(filepath.Join(dir, e.Name()))
		lines = append(lines, memIndexLine(strings.TrimSuffix(e.Name(), ".md"), frontmatterField(string(data), "description")))
	}
	if len(lines) == 0 {
		return ""
	}
	return memIndexHeader + strings.Join(lines, "\n") + "\n"
}

// writeMemIndex 重建并写盘 MEMORY.md（记忆写入/删除后调，保持总览文件实时、人也可直接打开看）。
func writeMemIndex(dir string) {
	c := genMemIndex(dir)
	if c == "" {
		c = memIndexHeader
	}
	_ = os.WriteFile(memIndexPath(dir), []byte(c), 0o644)
}

// listMemories 列出 dir 下记忆（filter 非空则按关键词过滤名/摘要/正文）。
func listMemories(dir, filter string) string {
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) == 0 {
		return "（暂无记忆）"
	}
	filter = strings.ToLower(filter)
	var lines []string
	for _, e := range entries {
		if e.IsDir() || !isMemFile(e.Name()) {
			continue
		}
		data, _ := os.ReadFile(filepath.Join(dir, e.Name()))
		text := string(data)
		if filter != "" && !strings.Contains(strings.ToLower(text), filter) {
			continue
		}
		lines = append(lines, "- "+strings.TrimSuffix(e.Name(), ".md")+"："+frontmatterField(text, "description"))
	}
	if len(lines) == 0 {
		return "（无匹配记忆）"
	}
	return strings.Join(lines, "\n")
}

func registerMemoryTools(r *Registry, root string) {
	dir := memoryDir(root)

	r.Register(&Tool{
		Name: "memory_write",
		Description: "写入或【更新】一条持久记忆（跨会话保留在 .pair/memory/）。**先 memory_search/list 查有无相关记忆——" +
			"有则用其同名覆盖来更新（先 memory_read 读旧的、融合后写回），别为同一主题反复新建、造成碎片化**。" +
			"name 唯一标识；type: user(用户偏好)/feedback(纠正与确认的做法)/project(项目决策约束)/reference(外部资源指针)；description 一句话摘要；content 正文。",
		Parameters: objSchema(props{
			"name":        strProp("唯一名，用【简短中文】命名（如 数据库连接池配置）；更新已有记忆请用其原名"),
			"type":        strProp("user/feedback/project/reference"),
			"description": strProp("一句话摘要"),
			"content":     strProp("正文"),
		}, "name", "description", "content"),
		RequiresApproval: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			name := safeMemName(argStr(args, "name"))
			if name == "" {
				return "", fmt.Errorf("name 不能为空")
			}
			typ := argStr(args, "type")
			if typ == "" {
				typ = "project"
			}
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return "", err
			}
			path := filepath.Join(dir, name+".md")
			_, statErr := os.Stat(path)
			updating := statErr == nil // 同名已存在 → 这是更新
			desc, content := argStr(args, "description"), argStr(args, "content")
			body := fmt.Sprintf("---\nname: %s\ntype: %s\ndescription: %s\n---\n\n%s\n", name, typ, desc, content)
			if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
				return "", err
			}
			writeMemIndex(dir) // 同步维护 MEMORY.md 总览索引
			if updating {
				return "已更新记忆：" + name, nil
			}
			// 新建：若已有相关记忆，提醒优先更新它而非新建（防碎片化）。
			if sim := similarMemory(dir, name, name+" "+desc+" "+content); sim != "" {
				return "已新建记忆：" + name + "\n⚠ 已有相关记忆「" + sim +
					"」——若属同一主题，建议改用 memory_read 读它、融合后用「" + sim + "」更新，而非新建，避免记忆碎片化。", nil
			}
			return "已记忆：" + name, nil
		},
	})

	r.Register(&Tool{
		Name:             "memory_delete",
		Description:      "删除一条过时/错误的记忆（按 name）。保持记忆库精简准确，别让过时信息长期误导。",
		Parameters:       objSchema(props{"name": strProp("记忆名")}, "name"),
		RequiresApproval: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			name := safeMemName(argStr(args, "name"))
			path := filepath.Join(dir, name+".md")
			if _, err := os.Stat(path); err != nil {
				return "", fmt.Errorf("无此记忆: %s", name)
			}
			if err := os.Remove(path); err != nil {
				return "", err
			}
			writeMemIndex(dir) // 同步更新总览索引
			return "已删除记忆：" + name, nil
		},
	})

	r.Register(&Tool{
		Name:        "memory_read",
		Description: "按 name 读取一条记忆的全文。",
		Parameters:  objSchema(props{"name": strProp("记忆名")}, "name"),
		ReadOnly:    true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			name := safeMemName(argStr(args, "name"))
			data, err := os.ReadFile(filepath.Join(dir, name+".md"))
			if err != nil {
				return "", fmt.Errorf("无此记忆: %s", name)
			}
			return string(data), nil
		},
	})

	r.Register(&Tool{
		Name:        "memory_list",
		Description: "列出所有记忆的【总览】（名 + 摘要，渐进式披露的总览层）；要某条细则用 memory_read 读全文。",
		Parameters:  objSchema(props{}),
		ReadOnly:    true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			if c := genMemIndex(dir); c != "" {
				return c, nil
			}
			return "（暂无记忆）", nil
		},
	})

	r.Register(&Tool{
		Name:        "memory_search",
		Description: "按关键词搜索记忆（匹配名/摘要/正文），返回命中条目的名+摘要。",
		Parameters:  objSchema(props{"query": strProp("关键词")}, "query"),
		ReadOnly:    true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			q := strings.TrimSpace(argStr(args, "query"))
			if q == "" {
				return "", fmt.Errorf("query 不能为空")
			}
			return listMemories(dir, q), nil
		},
	})
}
