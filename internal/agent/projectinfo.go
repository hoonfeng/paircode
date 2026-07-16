// 项目知识库（Project Knowledge Base）—— 与「项目记忆」区分的另一套存储（复刻参考 .Pair/project-info）：
//   · 记忆(.pair/memory)   = Agent 跨会话学到的事实/教训/偏好（增量、Agent 主导）。
//   · 知识库(.pair/project-info) = 项目的结构化理解（架构/模块职责/数据流/设计决策），
//     由用户触发「探索项目知识库」一次性构建，是【给用户看的】可浏览中文文档，并自动注入 Agent 上下文。
// 每篇 = .pair/project-info/<路径>.md（首行 # 标题）；按路径深度分级：概览/模块自动加载、细节按需读（渐进式披露）。

package agent

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

func projectInfoDir(root string) string { return filepath.Join(root, ".pair", "project-info") }

// safeInfoPath 规范化条目路径：去 .md、清理、禁路径穿越（..、绝对路径），允许 / 嵌套。
func safeInfoPath(p string) string {
	p = strings.TrimSuffix(strings.TrimSpace(p), ".md")
	p = strings.ReplaceAll(p, "\\", "/")
	p = path.Clean("/" + p) // 绝对化再清理，吃掉 ..
	return strings.Trim(p, "/")
}

func infoFilePath(dir, rel string) string { return filepath.Join(dir, filepath.FromSlash(rel)+".md") }

// infoLevel 按路径分级：概览(overview/概览) 始终加载；顶层其余=模块(自动加载)；嵌套(带/)=细节(按需读)。
func infoLevel(rel string) string {
	low := strings.ToLower(rel)
	switch {
	case low == "overview" || rel == "概览" || rel == "项目概览":
		return "overview"
	case strings.Contains(rel, "/"):
		return "detail"
	default:
		return "module"
	}
}

type infoEntry struct{ Path, Title, Level, Content string }

func firstHeading(md, fallback string) string {
	for _, ln := range strings.Split(md, "\n") {
		if s := strings.TrimSpace(ln); strings.HasPrefix(s, "# ") {
			return strings.TrimSpace(s[2:])
		}
	}
	return fallback
}

// scanInfoEntries 递归扫描知识库目录（.md），返回各条目（路径/标题/分级/正文）。
func scanInfoEntries(dir string) []infoEntry {
	var out []infoEntry
	filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".md") {
			return nil
		}
		rel, _ := filepath.Rel(dir, p)
		rel = filepath.ToSlash(strings.TrimSuffix(rel, ".md"))
		data, _ := os.ReadFile(p)
		out = append(out, infoEntry{Path: rel, Title: firstHeading(string(data), rel), Level: infoLevel(rel), Content: string(data)})
		return nil
	})
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

// ProjectKnowledge 自动加载知识库概览注入 Agent 上下文：概览篇给正文 + 其余篇列目录（渐进式披露，
// 细则用 project_info_read 读）。预算 maxChars。无知识库→""。
func ProjectKnowledge(root string, maxChars int) string {
	entries := scanInfoEntries(projectInfoDir(root))
	if len(entries) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n\n# 项目知识库（自动加载）\n本项目的结构化理解；需要某篇细则用 project_info_read 读全文。\n")
	for _, e := range entries { // 概览篇给正文
		if e.Level == "overview" {
			b.WriteString("\n" + truncRunesAgent(strings.TrimSpace(e.Content), 1400) + "\n")
			break
		}
	}
	b.WriteString("\n## 知识库目录\n")
	for _, e := range entries {
		if e.Level == "overview" {
			continue
		}
		b.WriteString("- [" + e.Title + "](" + e.Path + ")\n")
	}
	return truncRunesAgent(b.String(), maxChars)
}

// registerProjectInfoTools 注册项目知识库工具（write/read/list/search/delete/explore）。
// 写类不需审批（用户触发探索时 Agent 自主逐模块写入，复刻参考 requiresApproval:false）。
func registerProjectInfoTools(r *Registry, root string) {
	dir := projectInfoDir(root)

	r.Register(&Tool{
		Name: "project_info_write",
		Description: "写入/更新项目知识库的一篇（.pair/project-info/<路径>.md）——记录项目架构/模块职责/数据流/设计决策等结构化理解，" +
			"跨会话复用、你和用户都能看。路径用中文（如 概览 / 模块-agent / 决策-渲染架构）。",
		Parameters: objSchema(props{
			"path":    strProp("条目路径（中文，如 概览 / 模块-agent），不含 .md；用 / 可嵌套为细节篇"),
			"content": strProp("Markdown 正文（首行用 # 标题）"),
		}, "path", "content"),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			rel := safeInfoPath(argStr(args, "path"))
			if rel == "" {
				return "", fmt.Errorf("path 不能为空")
			}
			fp := infoFilePath(dir, rel)
			if err := os.MkdirAll(filepath.Dir(fp), 0o755); err != nil {
				return "", err
			}
			_, statErr := os.Stat(fp)
			if err := os.WriteFile(fp, []byte(argStr(args, "content")), 0o644); err != nil {
				return "", err
			}
			if statErr == nil {
				return "已更新知识库：" + rel, nil
			}
			return "已写入知识库：" + rel, nil
		},
	})

	r.Register(&Tool{
		Name:        "project_info_read",
		Description: "读取知识库某篇的全文（按路径，如 概览 / 模块-agent）。渐进式披露的细节层。",
		Parameters:  objSchema(props{"path": strProp("条目路径，不含 .md")}, "path"),
		ReadOnly:    true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			rel := safeInfoPath(argStr(args, "path"))
			data, err := os.ReadFile(infoFilePath(dir, rel))
			if err != nil {
				return "", fmt.Errorf("无此知识库条目：%s（用 project_info_list 看全部）", rel)
			}
			return string(data), nil
		},
	})

	r.Register(&Tool{
		Name:        "project_info_list",
		Description: "列出知识库所有条目的【总览】（路径 + 标题 + 分级）。渐进式披露的总览层。",
		Parameters:  objSchema(props{}),
		ReadOnly:    true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			entries := scanInfoEntries(dir)
			if len(entries) == 0 {
				return "（知识库为空。用 project_info_explore 起步、project_info_write 写入，或菜单「探索项目知识库」。）", nil
			}
			var b strings.Builder
			for _, e := range entries {
				fmt.Fprintf(&b, "- [%s] %s（%s）\n", e.Level, e.Title, e.Path)
			}
			return b.String(), nil
		},
	})

	r.Register(&Tool{
		Name:        "project_info_search",
		Description: "按关键词搜索知识库（匹配路径/标题/正文），返回命中条目。",
		Parameters:  objSchema(props{"query": strProp("关键词")}, "query"),
		ReadOnly:    true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			q := strings.ToLower(strings.TrimSpace(argStr(args, "query")))
			if q == "" {
				return "", fmt.Errorf("query 不能为空")
			}
			var lines []string
			for _, e := range scanInfoEntries(dir) {
				if strings.Contains(strings.ToLower(e.Path+e.Title+e.Content), q) {
					lines = append(lines, "- "+e.Title+"（"+e.Path+"）")
				}
			}
			if len(lines) == 0 {
				return "（无匹配条目）", nil
			}
			return strings.Join(lines, "\n"), nil
		},
	})

	r.Register(&Tool{
		Name:        "project_info_delete",
		Description: "删除知识库某篇（按路径）。",
		Parameters:  objSchema(props{"path": strProp("条目路径，不含 .md")}, "path"),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			rel := safeInfoPath(argStr(args, "path"))
			fp := infoFilePath(dir, rel)
			if _, err := os.Stat(fp); err != nil {
				return "", fmt.Errorf("无此知识库条目：%s", rel)
			}
			if err := os.Remove(fp); err != nil {
				return "", err
			}
			return "已删除知识库条目：" + rel, nil
		},
	})

	r.Register(&Tool{
		Name: "project_info_explore",
		Description: "返回项目目录结构概览（根目录关键文件、顶层目录及文件数）——构建知识库的起点；" +
			"据此用 read_file 读关键文件分析，再 project_info_write 写入 概览/模块-*/决策-*。",
		Parameters: objSchema(props{}),
		ReadOnly:   true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			return exploreProjectStructure(root), nil
		},
	})
}

func infoKeyFile(n string) bool {
	low := strings.ToLower(n)
	switch {
	case strings.HasPrefix(low, "readme"), strings.HasPrefix(low, "makefile"):
		return true
	}
	switch low {
	case "go.mod", "package.json", "cargo.toml", "pyproject.toml", "pom.xml",
		"main.go", "agents.md", "claude.md", "go.sum", "tsconfig.json":
		return true
	}
	return false
}

// exploreProjectStructure 轻量项目结构概览：根目录关键文件 + 顶层目录及文件数（供 Agent 起步分析）。
func exploreProjectStructure(root string) string {
	entries, err := os.ReadDir(root)
	if err != nil {
		return "无法读取项目根目录：" + err.Error()
	}
	var b strings.Builder
	b.WriteString("# 项目结构概览（供分析后写入知识库）\n\n## 根目录关键文件\n")
	for _, e := range entries {
		if !e.IsDir() && infoKeyFile(e.Name()) {
			b.WriteString("- " + e.Name() + "\n")
		}
	}
	b.WriteString("\n## 顶层目录（约略文件数）\n")
	for _, e := range entries {
		if !e.IsDir() || isSkipDir(e.Name()) || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		b.WriteString(fmt.Sprintf("- %s/（约 %d 文件）\n", e.Name(), countDirFiles(filepath.Join(root, e.Name()))))
	}
	b.WriteString("\n建议：用 read_file 读关键文件分析后，project_info_write 写入「概览」「模块-<名>」「决策-<主题>」等中文条目。")
	return b.String()
}

// countDirFiles 数目录下文件（递归，跳过依赖/产物目录，上限 2000 防卡）。
func countDirFiles(dir string) int {
	n := 0
	filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if isSkipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if n++; n >= 2000 {
			return filepath.SkipAll
		}
		return nil
	})
	return n
}

// ExploreKnowledgeTask 用户触发「探索项目知识库」时发给 Agent 的任务（复刻参考 project:explore 的 task）。
func ExploreKnowledgeTask() string {
	return "探索本项目并构建【项目知识库】：\n" +
		"1. 先调用 project_info_explore 获取项目结构概览。\n" +
		"2. 用 read_file 阅读关键文件（入口、核心模块、配置）。\n" +
		"3. 分析各模块的架构、职责、数据流与设计决策。\n" +
		"4. 用 project_info_write 把分析写入知识库，建议中文路径：概览（项目概览）、模块-<各模块>、决策-<设计决策>。\n" +
		"全程用中文，命名用中文。完成后简要汇报写了哪些条目。"
}
