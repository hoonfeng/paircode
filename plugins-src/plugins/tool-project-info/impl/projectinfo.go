// 项目知识库（Project Knowledge Base）—— 与「项目记忆」区分的另一套存储（复刻参考 .Pair/project-info）：
//   · 记忆(.pair/memory)   = Agent 跨会话学到的事实/教训/偏好（增量、Agent 主导）。
//   · 知识库(.pair/project-info) = 项目的结构化理解（架构/模块职责/数据流/设计决策），
//     由用户触发「探索项目知识库」一次性构建，是【给用户看的】可浏览中文文档，并自动注入 Agent 上下文。
// 每篇 = .pair/project-info/<路径>.md（首行 # 标题）；按路径深度分级：概览/模块自动加载、细节按需读（渐进式披露）。

package impl

import (
	"context"
	"fmt"
	. "github.com/hoonfeng/paircode/plugins-src/plugins/tool-project-info/toolbin"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

func projectInfoDir(root string) string { return filepath.Join(root, ".pair", "project-info") }

// partsOf 取条目路径末段（文件名段）。
func partsOf(rel string) string {
	if i := strings.LastIndexByte(rel, '/'); i >= 0 {
		return rel[i+1:]
	}
	return rel
}

// infoBranches 知识库顶层分支（树形分叉）：目标/架构/实现/关键点/设计思想。
// 概览.md 为根条目（不属于分支）。新条目应归入某分支，便于人浏览理解。
var infoBranches = []string{"目标", "架构", "实现", "关键点", "设计思想"}

// isInfoBranch 判断顶层路径段是否为合法知识库分支。
func isInfoBranch(head string) bool {
	for _, b := range infoBranches {
		if head == b {
			return true
		}
	}
	return false
}

// agentsNotesDir 外部决策树目录（.agents/notes/）——模型后训练含外部数据，
// 会幻觉该路径；存在时作为知识库只读附加源（条目路径前缀 notes/）。
func agentsNotesDir(root string) string { return filepath.Join(root, ".agents", "notes") }

// safeInfoPath 规范化条目路径：去 .md、清理、禁路径穿越（..、绝对路径），允许 / 嵌套。
func safeInfoPath(p string) string {
	p = strings.TrimSuffix(strings.TrimSpace(p), ".md")
	p = strings.ReplaceAll(p, "\\", "/")
	p = path.Clean("/" + p) // 绝对化再清理，吃掉 ..
	return strings.Trim(p, "/")
}

func infoFilePath(dir, rel string) string { return filepath.Join(dir, filepath.FromSlash(rel)+".md") }

// infoLevel 按路径分级（树形知识库）：
//
//	overview = 根条目（概览.md，唯一，自动全量加载）
//	module   = 1 层（分支/条目，如 架构/模块-agent）——自动加载标题摘要
//	detail   = 2+ 层（分支/子类/条目）——按需读（渐进式披露）
func infoLevel(rel string) string {
	low := strings.ToLower(rel)
	switch {
	case low == "overview" || rel == "概览" || rel == "项目概览":
		return "overview"
	case strings.Count(rel, "/") >= 2:
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
// 附加源：工作区 .agents/notes/ 参考决策树存在时并入（路径前缀 notes/，兼容模型幻觉路径）。
// skip 为非 nil 时对每个候选条目调用：返回 true 表示跳过（去重：notes/ 已镜像到树的条目不重复列）。
func scanInfoEntries(dir string) []infoEntry {
	var out []infoEntry
	scanDir := func(base, prefix string, skip func(string) bool) {
		filepath.WalkDir(base, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(p, ".md") {
				return nil
			}
			rel, _ := filepath.Rel(base, p)
			rel = filepath.ToSlash(strings.TrimSuffix(rel, ".md"))
			if skip != nil && skip(rel) {
				return nil
			}
			if prefix != "" {
				rel = prefix + "/" + rel
			}
			data, _ := os.ReadFile(p)
			out = append(out, infoEntry{Path: rel, Title: firstHeading(string(data), rel), Level: infoLevel(rel), Content: string(data)})
			return nil
		})
	}
	scanDir(dir, "", nil)
	rootDir := filepath.Dir(filepath.Dir(dir)) // .pair/project-info → 项目根
	notes := agentsNotesDir(rootDir)
	if notes != dir {
		scanDir(notes, "notes", func(nrel string) bool {
			br, ok := notesToBranchRel(nrel)
			if !ok {
				return false
			}
			_, err := os.Stat(infoFilePath(dir, br)) // 树中已有镜像副本 → 跳过，避免重复
			return err == nil
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

// notesToBranchRel 把外部决策树路径（.agents/notes/ 相对路径）映射到知识库树分支路径。
// 模型后训练含外部数据会幻觉 notes 路径（implemented/architecture、implemented/feature、
// implemented/process、decision…）；project_info_write 写入 notes/ 前缀路径时自动归入树分支，
// 保证知识库仍是完整树。输入可带 notes/ 前缀（工具 path 参数）或纯相对（扫描去重用）。
// 映射：implemented/architecture→架构、implemented/feature→实现、implemented/decision→设计思想、
// decision→设计思想、其余 implemented/*→关键点、其余→实现；取末段为文件名。
func notesToBranchRel(n string) (string, bool) {
	n = strings.TrimPrefix(strings.TrimPrefix(n, "notes/"), "/")
	if n == "" {
		return "", false
	}
	segs := strings.Split(n, "/")
	leaf := segs[len(segs)-1]
	var branch string
	switch {
	case len(segs) >= 2 && segs[0] == "implemented" && segs[1] == "architecture":
		branch = "架构"
	case len(segs) >= 2 && segs[0] == "implemented" && (segs[1] == "decision" || segs[1] == "decisions"):
		branch = "设计思想"
	case len(segs) >= 2 && segs[0] == "implemented" && segs[1] == "feature":
		branch = "实现"
	case len(segs) >= 2 && segs[0] == "implemented":
		branch = "关键点" // process / 其他实施记录 → 修复记录
	case segs[0] == "decision" || segs[0] == "decisions":
		branch = "设计思想"
	case len(segs) >= 2 && segs[0] == "inbox":
		branch = "实现"
	default:
		branch = "实现"
	}
	return branch + "/" + leaf, true
}

// infoTree 构建知识库条目树（分支=目录，叶子=条目），返回缩进树文本。
// showLevel 时叶子带分级标记。条目按路径排序保证确定性。
// 叶子显示「标题（路径末段）」（末段与标题不同时），树形分支给出完整路径上下文。
func infoTree(entries []infoEntry, showLevel bool) string {
	type node struct {
		name     string
		children map[string]*node
		entry    *infoEntry
	}
	root := &node{children: map[string]*node{}}
	for i := range entries {
		e := &entries[i]
		parts := strings.Split(e.Path, "/")
		cur := root
		for _, seg := range parts[:len(parts)-1] {
			nxt, ok := cur.children[seg]
			if !ok {
				nxt = &node{name: seg, children: map[string]*node{}}
				cur.children[seg] = nxt
			}
			cur = nxt
		}
		leaf := parts[len(parts)-1]
		n, ok := cur.children[leaf]
		if !ok {
			n = &node{name: leaf, children: map[string]*node{}}
			cur.children[leaf] = n
		}
		n.entry = e
	}
	var b strings.Builder
	var walk func(n *node, prefix string)
	walk = func(n *node, prefix string) {
		keys := make([]string, 0, len(n.children))
		for k := range n.children {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for i, k := range keys {
			ch := n.children[k]
			last := i == len(keys)-1
			conn := "├── "
			nextPrefix := prefix + "│   "
			if last {
				conn = "└── "
				nextPrefix = prefix + "    "
			}
			if ch.entry != nil {
				mark := ""
				if showLevel {
					mark = " [" + ch.entry.Level + "]"
				}
				title := ch.entry.Title
				if leaf := partsOf(ch.entry.Path); leaf != "" && leaf != title {
					title += "（" + leaf + "）"
				}
				b.WriteString(prefix + conn + title + mark + "\n")
			} else {
				b.WriteString(prefix + conn + k + "/\n")
			}
			walk(ch, nextPrefix)
		}
	}
	walk(root, "")
	return b.String()
}

// ProjectKnowledge 自动加载知识库概览注入 Agent 上下文：概览篇给正文 + 其余篇树形目录
// （渐进式披露，细则用 project_info_read 读）。预算 maxChars。无知识库→""。
func ProjectKnowledge(root string, maxChars int) string {
	entries := scanInfoEntries(projectInfoDir(root))
	if len(entries) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n\n# 项目知识库（自动加载）\n本项目的结构化理解树：概览/目标/架构/实现/关键点/设计思想；细则用 project_info_read 读全文。\n")
	for _, e := range entries { // 概览篇给正文
		if e.Level == "overview" {
			b.WriteString("\n" + truncRunesAgent(strings.TrimSpace(e.Content), 1400) + "\n")
			break
		}
	}
	b.WriteString("\n## 知识库目录（树）\n")
	b.WriteString(infoTree(entries, false))
	return truncRunesAgent(b.String(), maxChars)
}

// registerProjectInfoTools 注册项目知识库工具（write/read/list/search/delete/explore）。
// 写类不需审批（用户触发探索时 Agent 自主逐模块写入，复刻参考 requiresApproval:false）。
func Register(r *Registry, root string) {
	// infoDirFromArgs 解析 project 参数 → 目标项目知识库目录（缺省 = 主项目）。
	// 多项目工作区中知识库按项目隔离（.pair/project-info/ 在各项目根下）。
	infoDirFromArgs := func(args map[string]any) (string, error) {
		projRoot, err := ProjRootFromArgs(root, args)
		if err != nil {
			return "", err
		}
		return projectInfoDir(projRoot), nil
	}

	r.Register(&Tool{
		Name:       "project_info_write",
		UsageGuide: "写入/更新项目知识库条目，跨会话复用。★知识库是树：顶层分支 = 目标/架构/实现/关键点/设计思想（根为 概览）——路径带分支前缀（如 架构/模块-agent / 设计思想/决策-渲染架构）。也可用外部风格路径 notes/implemented/architecture/x（自动归入树分支 架构/x 并镜像 .agents/notes/）。读完关键文件后立即写入，积累项目的结构化理解。比记在脑子里可靠（持久化+跨会话可见）。多项目工作区可用 project 参数指定目标项目。",
		Description: "写入/更新项目知识库的一篇（.pair/project-info/<路径>.md）——记录项目架构/模块职责/数据流/设计决策等结构化理解，" +
			"跨会话复用、你和用户都能看。★树形路径：顶层分支 目标/架构/实现/关键点/设计思想，根条目用 概览（如 架构/模块-agent / 设计思想/决策-渲染架构）；兼容外部 notes/ 前缀路径（自动映射分支+镜像 .agents/notes/）。",
		Parameters: ObjSchema(Props{
			"path":    StrProp("条目路径（中文，带顶层分支前缀：目标/架构/实现/关键点/设计思想，如 架构/模块-agent），不含 .md；用 / 嵌套为细节篇"),
			"content": StrProp("Markdown 正文（首行用 # 标题）"),
			"project": ProjectSchemaProp(),
		}, "path", "content"),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			dir, err := infoDirFromArgs(args)
			if err != nil {
				return "", err
			}
			rel := safeInfoPath(ArgStr(args, "path"))
			if rel == "" {
				return "", fmt.Errorf("path 不能为空")
			}
			// ★notes/ 前缀兼容：模型后训练会幻觉 .agents/notes/implemented/… 路径，
			// 写入时自动归入树分支（如 notes/implemented/architecture/x → 架构/x），
			// 并镜像一份到 .agents/notes/ 原路径（参考工具链/read 可读到）。
			branchRel, mirrorRel := rel, ""
			if strings.HasPrefix(rel, "notes/") {
				if br, ok := notesToBranchRel(rel); ok {
					branchRel = br
				}
				mirrorRel = strings.TrimPrefix(rel, "notes/")
			}
			fp := infoFilePath(dir, branchRel)
			if err := os.MkdirAll(filepath.Dir(fp), 0o755); err != nil {
				return "", err
			}
			_, statErr := os.Stat(fp)
			if err := os.WriteFile(fp, []byte(ArgStr(args, "content")), 0o644); err != nil {
				return "", err
			}
			if mirrorRel != "" { // 镜像：.agents/notes/<原相对路径>.md
				nfp := infoFilePath(agentsNotesDir(filepath.Dir(filepath.Dir(dir))), mirrorRel)
				if err := os.MkdirAll(filepath.Dir(nfp), 0o755); err != nil {
					return "", err
				}
				if err := os.WriteFile(nfp, []byte(ArgStr(args, "content")), 0o644); err != nil {
					return "", err
				}
			}
			head := branchRel
			if i := strings.IndexByte(head, '/'); i > 0 {
				head = head[:i]
			}
			hint := ""
			if !strings.HasPrefix(rel, "notes/") && head != "概览" && !isInfoBranch(head) {
				hint = "（提示：知识库是树，建议用顶层分支 目标/架构/实现/关键点/设计思想 开头，如 架构/" + branchRel + "）"
			}
			verb := "已写入知识库"
			if statErr == nil {
				verb = "已更新知识库"
			}
			if mirrorRel != "" {
				return verb + "：" + branchRel + "（notes/ 参考路径已镜像 .agents/notes/" + mirrorRel + "）", nil
			}
			return verb + "：" + branchRel + hint, nil
		},
	})

	r.Register(&Tool{
		Name:        "project_info_read",
		UsageGuide:  "读取知识库某篇全文。渐进式披露：先 project_info_list 看总览，再用此工具读具体细则。比翻目录更方便（自动解析路径+内容格式化）。",
		Description: "读取知识库某篇的全文（按路径，如 概览 / 模块-agent）。渐进式披露的细节层。",
		Parameters:  ObjSchema(Props{"path": StrProp("条目路径，不含 .md"), "project": ProjectSchemaProp()}, "path"),
		ReadOnly:    true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			dir, err := infoDirFromArgs(args)
			if err != nil {
				return "", err
			}
			rel := safeInfoPath(ArgStr(args, "path"))
			data, err := os.ReadFile(infoFilePath(dir, rel))
			if err != nil {
				return "", fmt.Errorf("无此知识库条目：%s（用 project_info_list 看全部）", rel)
			}
			return string(data), nil
		},
	})

	r.Register(&Tool{
		Name:        "project_info_list",
		UsageGuide:  "列出知识库所有条目的总览（路径+标题+分级）。新项目先调此工具查看已有哪些文档，避免重复写入。",
		Description: "列出知识库所有条目的【总览】（路径 + 标题 + 分级）。渐进式披露的总览层。",
		Parameters:  ObjSchema(Props{"project": ProjectSchemaProp()}),
		ReadOnly:    true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			dir, err := infoDirFromArgs(args)
			if err != nil {
				return "", err
			}
			entries := scanInfoEntries(dir)
			if len(entries) == 0 {
				return "（知识库为空。用 project_info_explore 起步、project_info_write 写入，或菜单「探索项目知识库」。）", nil
			}
			return infoTree(entries, true), nil
		},
	})

	r.Register(&Tool{
		Name:        "project_info_tree",
		UsageGuide:  "查看知识库完整树形结构（分支/子类/条目缩进树）。比 project_info_list 更直观：先看树定位条目，再 project_info_read 读全文。",
		Description: "返回知识库完整树形结构（缩进树：目标/架构/实现/关键点/设计思想 分支 + 条目）。人可读的树形导航。",
		Parameters:  ObjSchema(Props{"project": ProjectSchemaProp()}),
		ReadOnly:    true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			dir, err := infoDirFromArgs(args)
			if err != nil {
				return "", err
			}
			entries := scanInfoEntries(dir)
			if len(entries) == 0 {
				return "（知识库为空）", nil
			}
			return "# 项目知识库（树）\n" + infoTree(entries, false), nil
		},
	})

	r.Register(&Tool{
		Name:        "project_info_search",
		UsageGuide:  "按关键词搜索知识库（匹配路径/标题/正文）。想查某个模块/概念是否已有文档时优先用此工具。",
		Description: "按关键词搜索知识库（匹配路径/标题/正文），返回命中条目。",
		Parameters:  ObjSchema(Props{"query": StrProp("关键词"), "project": ProjectSchemaProp()}, "query"),
		ReadOnly:    true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			dir, err := infoDirFromArgs(args)
			if err != nil {
				return "", err
			}
			q := strings.ToLower(strings.TrimSpace(ArgStr(args, "query")))
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
		UsageGuide:  "删除知识库某篇（按路径）。知识库条目过时/错误时用此工具清理。删除前建议先 project_info_read 确认。",
		Description: "删除知识库某篇（按路径）。",
		Parameters:  ObjSchema(Props{"path": StrProp("条目路径，不含 .md"), "project": ProjectSchemaProp()}, "path"),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			dir, err := infoDirFromArgs(args)
			if err != nil {
				return "", err
			}
			rel := safeInfoPath(ArgStr(args, "path"))
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
		Name:       "project_info_explore",
		UsageGuide: "扫描项目目录结构概览——构建知识库的起点。新项目首次接触时先调此工具了解项目全貌，再用 read 读关键文件，最后 project_info_write 写入结构化理解。",
		Description: "返回项目目录结构概览（根目录关键文件、顶层目录及文件数）——构建知识库的起点；" +
			"据此用 read 读关键文件分析，再 project_info_write 写入 概览/模块-*/决策-*。",
		Parameters: ObjSchema(Props{}),
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
	b.WriteString("\n建议：用 read 读关键文件分析后，project_info_write 写入「概览」「模块-<名>」「决策-<主题>」等中文条目。")
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
		"2. 用 read 阅读关键文件（入口、核心模块、配置）。\n" +
		"3. 分析各模块的架构、职责、数据流与设计决策。\n" +
		"4. 用 project_info_write 把分析写入知识库，建议中文路径：概览（项目概览）、模块-<各模块>、决策-<设计决策>。\n" +
		"全程用中文，命名用中文。完成后简要汇报写了哪些条目。"
}

// truncRunesAgent 按 rune 截断并加省略号。
func truncRunesAgent(s string, n int) string {
	s = strings.TrimSpace(s)
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

func isSkipDir(name string) bool { return defaultSkipDirs[name] || extraSkipDirs[name] }

var defaultSkipDirs = map[string]bool{
	// VCS / 编辑器
	".git": true, ".svn": true, ".hg": true, ".idea": true, ".vscode": true,
	// 依赖库 / 模块库
	"node_modules": true, "bower_components": true, "jspm_packages": true, "vendor": true, "Pods": true,
	"venv": true, ".venv": true, "__pycache__": true, ".pytest_cache": true, ".mypy_cache": true, ".tox": true,
	// 构建产物
	"dist": true, "build": true, "out": true, "target": true,
	".next": true, ".nuxt": true, ".svelte-kit": true, ".output": true,
	// 缓存 / 覆盖率 / 基建
	".gradle": true, ".cache": true, ".turbo": true, "coverage": true, ".nyc_output": true, ".terraform": true,
	// 本项目自身数据 / 备份
	".pair": true, "源码备份": true,
}

var extraSkipDirs = map[string]bool{}
