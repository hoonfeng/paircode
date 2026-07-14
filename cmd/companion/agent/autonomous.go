// 自主模式：外层设计者 Loop（update_plan + delegate_task）→ 内层执行 Loop（全部工具）。
// 外层 LLM 是真正的设计者，通过工具调用控制计划、分派任务、调整策略。
// 所有逻辑在 agent 包内完成，bridge 只需一句调用。

package agent

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hoonfeng/paircode/pkg/codegraph"
)

// registerOuterExplorationTools 为外层设计者注册只读探索工具。
// 执行 agent（内层）拥有完整工具集，而外层（设计者）只需理解项目结构以制定计划，
// 因此只注册查询/搜索/读取类的只读工具，不注册任何写工具。
func registerOuterExplorationTools(r *Registry, root string) {
	// ── read_file：读取文件 ──
	r.Register(&Tool{
		Name:        "read_file",
		Description: "读取文件内容。path 为工作区内路径。可选 offset(起始行,1 基)+limit(行数)读片段；省略则读全文(超 2000 行只返回前 2000 行并提示用 offset/limit 翻页)。",
		Parameters:  objSchema(props{"path": strProp("文件路径（工作区内）"), "offset": intProp("可选：起始行号(1 基)"), "limit": intProp("可选：读取行数")}, "path"),
		ReadOnly:    true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			p, err := resolvePath(root, argStr(args, "path"))
			if err != nil {
				return "", err
			}
			data, err := os.ReadFile(p)
			if err != nil {
				return "", err
			}
			if strings.IndexByte(string(data), 0) >= 0 {
				return "", fmt.Errorf("「%s」是二进制文件，read_file 不支持读取二进制内容；请用 inspect_binary 工具查看", argStr(args, "path"))
			}
			offset, limit := argInt(args, "offset", 0), argInt(args, "limit", 0)
			if offset <= 0 && limit <= 0 {
				lines := strings.Split(string(data), "\n")
				if len(lines) > 2000 {
					return strings.Join(lines[:2000], "\n") + fmt.Sprintf("\n…[文件共 %d 行，仅显示前 2000；用 offset/limit 读其余]", len(lines)), nil
				}
				return string(data), nil
			}
			lines := strings.Split(string(data), "\n")
			start := offset - 1
			if start < 0 {
				start = 0
			}
			if start >= len(lines) {
				return "", fmt.Errorf("offset %d 超出文件行数 %d", offset, len(lines))
			}
			end := len(lines)
			if limit > 0 && start+limit < end {
				end = start + limit
			}
			return strings.Join(lines[start:end], "\n"), nil
		},
	})

	// ── list_files：列出目录 ──
	r.Register(&Tool{
		Name:        "list_files",
		Description: "列出目录下的文件/子目录（目录在前）。path 省略则列工作区根；pattern 可选（如 *.go）。",
		Parameters:  objSchema(props{"path": strProp("目录路径（省略=工作区根）"), "pattern": strProp("可选通配符过滤，如 *.go")}),
		ReadOnly:    true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			rel := argStr(args, "path")
			p := root
			if rel != "" {
				var err error
				if p, err = resolvePath(root, rel); err != nil {
					return "", err
				}
			}
			entries, err := os.ReadDir(p)
			if err != nil {
				return "", err
			}
			pattern := argStr(args, "pattern")
			sort.SliceStable(entries, func(i, j int) bool {
				if entries[i].IsDir() != entries[j].IsDir() {
					return entries[i].IsDir()
				}
				return entries[i].Name() < entries[j].Name()
			})
			var b strings.Builder
			for _, e := range entries {
				if pattern != "" && !e.IsDir() {
					if ok, _ := filepath.Match(pattern, e.Name()); !ok {
						continue
					}
				}
				if e.IsDir() {
					b.WriteString("[dir]  ")
				} else {
					b.WriteString("[file] ")
				}
				b.WriteString(e.Name())
				b.WriteString("\n")
			}
			return b.String(), nil
		},
	})

	// ── 搜索工具（search_content, search_files, find_files_by_pattern）──
	registerSearchTools(r, root)

	// ── 文件符号工具（find_symbol, get_file_symbols, 等）──
	registerFileSymbolTools(r, root)

	// ── 代码知识图谱只读工具（所有 codegraph 工具，不含 codegraph_build）──
	r.Register(&Tool{
		Name:        "codegraph_stats",
		Description: "查看代码知识图谱的统计信息：实体总数、关系总数、覆盖的文件数、包数、各类实体分布。",
		Parameters:  objSchema(props{}),
		ReadOnly:    true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			g, err := getCodeGraph(root)
			if err != nil {
				return "", err
			}
			stats := g.Stats()
			return codegraph.GraphStatsText(stats), nil
		},
	})
	r.Register(&Tool{
		Name: "codegraph_file_structure",
		Description: "获取指定文件的实体结构树（文件→函数/类型→方法/字段的层次结构）。" +
			"用于理解文件内部的组织结构。",
		Parameters: objSchema(props{
			"file": strProp("文件路径（工作区相对路径，如 'cmd/companion/main.go'）"),
		}, "file"),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			filePath := codegraph.NormalizeFilePath(root, argStr(args, "file"))
			g, err := getCodeGraph(root)
			if err != nil {
				return "", err
			}
			qe := codegraph.NewQueryEngine(g)
			nodes := qe.GetFileStructure(filePath)
			return codegraph.FileStructureText(nodes), nil
		},
	})
	r.Register(&Tool{
		Name:        "codegraph_function",
		Description: "按名称查找函数/方法的定义位置。支持函数名、包名.函数名、或接收者.方法名。返回文件路径、行号、签名等信息。",
		Parameters:  objSchema(props{"name": strProp("函数名（如 'main'、'ServeHTTP'、'foo.Bar'）")}, "name"),
		ReadOnly:    true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			name := argStr(args, "name")
			g, err := getCodeGraph(root)
			if err != nil {
				return "", err
			}
			qe := codegraph.NewQueryEngine(g)
			locs := qe.GetFunctionDefinition(name)
			return codegraph.FunctionDefinitionText(locs), nil
		},
	})
	r.Register(&Tool{
		Name:        "codegraph_class",
		Description: "获取类型（struct/interface）的完整层次结构：字段、方法、嵌入类型。支持结构体名或接口名。",
		Parameters:  objSchema(props{"name": strProp("类型名（如 'Server'、'Handler'）")}, "name"),
		ReadOnly:    true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			name := argStr(args, "name")
			g, err := getCodeGraph(root)
			if err != nil {
				return "", err
			}
			qe := codegraph.NewQueryEngine(g)
			h := qe.GetClassHierarchy(name)
			return codegraph.ClassHierarchyText(h), nil
		},
	})
	r.Register(&Tool{
		Name:        "codegraph_callers",
		Description: "查询哪些函数调用了指定的函数/方法。用于理解函数被使用的情况。返回调用者的文件路径和行号。",
		Parameters:  objSchema(props{"name": strProp("函数/方法名（如 'SendRequest'、'handler.Handle'）")}, "name"),
		ReadOnly:    true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			name := argStr(args, "name")
			g, err := getCodeGraph(root)
			if err != nil {
				return "", err
			}
			qe := codegraph.NewQueryEngine(g)
			calls := qe.GetCallers(name)
			return codegraph.CallInfoText(calls, "调用者"), nil
		},
	})
	r.Register(&Tool{
		Name:        "codegraph_callees",
		Description: "查询指定的函数/方法调用了哪些其他函数。用于理解函数的内部调用情况。返回被调用者的名称和调用位置。",
		Parameters:  objSchema(props{"name": strProp("函数/方法名（如 'handleRequest'）")}, "name"),
		ReadOnly:    true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			name := argStr(args, "name")
			g, err := getCodeGraph(root)
			if err != nil {
				return "", err
			}
			qe := codegraph.NewQueryEngine(g)
			calls := qe.GetCallees(name)
			return codegraph.CallInfoText(calls, "被调用者"), nil
		},
	})
	r.Register(&Tool{
		Name: "codegraph_impact",
		Description: "分析修改某个函数/类型/文件后可能影响的范围。" +
			"基于调用图进行可达性分析，返回受影响的文件、函数列表和传播路径。" +
			"用于回答「修改这个函数会影响哪些地方？」",
		Parameters: objSchema(props{
			"entity":   strProp("实体标识（函数名、类型名或文件路径，如 'SendRequest'、'cmd/main.go'）"),
			"maxDepth": intProp("可选：搜索深度（默认 10，限制传递链长度）"),
		}, "entity"),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			entityID := argStr(args, "entity")
			maxDepth := argInt(args, "max_depth", 10)
			g, err := getCodeGraph(root)
			if err != nil {
				return "", err
			}
			qe := codegraph.NewQueryEngine(g)
			result := qe.ImpactAnalysis(entityID, maxDepth)
			return codegraph.ImpactResultText(result), nil
		},
	})
	r.Register(&Tool{
		Name: "codegraph_search",
		Description: "在代码知识图谱中搜索实体（函数、类型、变量、文件等）。" +
			"支持按名称搜索和按类型过滤。返回匹配实体的位置、签名和相关度评分。" +
			"比 search_content 更精确，因为基于结构化理解而非纯文本匹配。",
		Parameters: objSchema(props{
			"query":      strProp("搜索关键词（函数名、类型名、变量名等）"),
			"scope":      strProp("可选：搜索范围，可选值: all(全部)/file(文件)/function(函数)/type(类型)/variable(变量)/package(包)，默认 all"),
			"maxResults": intProp("可选：最大返回数（默认 20）"),
		}, "query"),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			query := argStr(args, "query")
			scopeStr := argStr(args, "scope")
			maxResults := argInt(args, "max_results", 20)
			g, err := getCodeGraph(root)
			if err != nil {
				return "", err
			}

			scope := codegraph.SearchScope(scopeStr)

			se := codegraph.NewSearchEngine(g, root)
			resp := se.Search(codegraph.SearchRequest{
				Query:      query,
				Scope:      scope,
				MaxResults: maxResults,
			})
			return codegraph.FormatResults(resp), nil
		},
	})
	r.Register(&Tool{
		Name: "codegraph_git_history",
		Description: "查询 Git 提交历史，并关联到代码实体。可以查询最近提交、" +
			"影响某个文件的提交，或者某个实体的变更历史。" +
			"用于回答「这个函数是谁改的？」、「这个 bug 是哪次提交引入的？」",
		Parameters: objSchema(props{
			"file":  strProp("可选：文件路径，查询影响该文件的提交历史"),
			"count": intProp("可选：返回提交数（默认 20，最大 100）"),
		}),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			filePath := strings.TrimSpace(argStr(args, "file"))
			count := argInt(args, "count", 20)
			if count > 100 {
				count = 100
			}
			if count <= 0 {
				count = 20
			}
			gh := codegraph.NewGitHistory(root)
			var commits []codegraph.CommitInfo
			var err error
			if filePath != "" {
				filePath = codegraph.NormalizeFilePath(root, filePath)
				commits, err = gh.GetCommitsAffecting(filePath, count)
			} else {
				commits, err = gh.GetRecentCommits(count)
			}
			if err != nil {
				return "", fmt.Errorf("查询 Git 历史失败: %w", err)
			}
			return codegraph.GitHistoryText(commits), nil
		},
	})
	r.Register(&Tool{
		Name: "codegraph_entity_history",
		Description: "查询指定代码实体的完整变更历史：谁在什么时候修改了它，" +
			"以及对应的提交消息。实体可以是函数名、类型名或文件路径。",
		Parameters: objSchema(props{
			"entity": strProp("实体名（函数名、类型名或文件路径）"),
		}, "entity"),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			entityName := argStr(args, "entity")
			g, err := getCodeGraph(root)
			if err != nil {
				return "", err
			}
			entities := g.SearchEntities(entityName)
			if len(entities) == 0 {
				gh := codegraph.NewGitHistory(root)
				commits, err := gh.GetCommitsAffecting(entityName, 20)
				if err != nil {
					return "", fmt.Errorf("未找到实体且查询文件历史失败: %v", err)
				}
				return codegraph.GitHistoryText(commits), nil
			}
			e := entities[0]
			gh := codegraph.NewGitHistory(root)
			commits, err := gh.GetEntityHistory(g, e.ID)
			if err != nil {
				return "", err
			}
			if len(commits) == 0 {
				return fmt.Sprintf("实体 %s (%s:%d) 暂无变更记录", e.Name, e.FilePath, e.Line), nil
			}
			return codegraph.GitHistoryText(commits), nil
		},
	})

	// ── web 搜索工具（web_search, web_fetch）──
	registerWebTools(r)

	// ── 项目知识库只读工具 ──
	r.Register(&Tool{
		Name:        "project_info_list",
		Description: "列出知识库所有条目的【总览】（路径 + 标题 + 分级）。渐进式披露的总览层。",
		Parameters:  objSchema(props{}),
		ReadOnly:    true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			dir := projectInfoDir(root)
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
		Name:        "project_info_read",
		Description: "读取知识库某篇的全文（按路径，如 概览 / 模块-agent）。渐进式披露的细节层。",
		Parameters:  objSchema(props{"path": strProp("条目路径，不含 .md")}, "path"),
		ReadOnly:    true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			dir := projectInfoDir(root)
			rel := safeInfoPath(argStr(args, "path"))
			data, err := os.ReadFile(infoFilePath(dir, rel))
			if err != nil {
				if os.IsNotExist(err) {
					return "知识库不存在条目: " + argStr(args, "path"), nil
				}
				return "", err
			}
			return string(data), nil
		},
	})
	r.Register(&Tool{
		Name:        "project_info_search",
		Description: "按关键词搜索知识库（匹配路径/标题/正文），返回命中条目。",
		Parameters:  objSchema(props{"query": strProp("关键词")}, "query"),
		ReadOnly:    true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			dir := projectInfoDir(root)
			q := strings.ToLower(strings.TrimSpace(argStr(args, "query")))
			if q == "" {
				return "", fmt.Errorf("query 不能为空")
			}
			entries, err := os.ReadDir(dir)
			if err != nil {
				return "知识库为空", nil
			}
			var matched []string
			for _, e := range entries {
				if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
					continue
				}
				name := strings.TrimSuffix(e.Name(), ".md")
				data, _ := os.ReadFile(filepath.Join(dir, e.Name()))
				if strings.Contains(strings.ToLower(string(data)), q) || strings.Contains(strings.ToLower(name), q) {
					matched = append(matched, name)
				}
			}
			if len(matched) == 0 {
				return "未找到匹配的知识库条目", nil
			}
			return "匹配条目:\n" + strings.Join(matched, "\n"), nil
		},
	})
	r.Register(&Tool{
		Name:        "project_info_explore",
		Description: "返回项目目录结构概览（根目录关键文件、顶层目录及文件数）——构建知识库的起点；据此用 read_file 读关键文件分析，再 project_info_write 写入 概览/模块-*/决策-*。",
		Parameters:  objSchema(props{}),
		ReadOnly:    true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			return exploreProjectStructure(root), nil
		},
	})

	// ── 记忆工具只读子集（memory_read/memory_list/memory_search）──
	r.Register(&Tool{
		Name:        "memory_read",
		Description: "按 name 读取一条记忆的全文。",
		Parameters:  objSchema(props{"name": strProp("记忆名")}, "name"),
		ReadOnly:    true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			dir := memoryDir(root)
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
			dir := memoryDir(root)
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
			dir := memoryDir(root)
			q := strings.TrimSpace(argStr(args, "query"))
			if q == "" {
				return "", fmt.Errorf("query 不能为空")
			}
			return listMemories(dir, q), nil
		},
	})
}

// RunAutonomous 运行自主模式。
//
// planProv: 规划 LLM Provider（外层设计者 Loop 使用）
// innerLoop: 预配置的执行 Loop（已含全部工具 + OnEvent）
// task: 用户目标
//
// 架构：
//
//	外层 Loop（设计者 Agent）
//	  工具: update_plan, delegate_task, generate_commit_message
//	  职责: 分析 → 规划 → 逐项委托 → 评估 → 调整 → 直至完成
//	  ↓ delegate_task
//	内层 Loop（执行 Agent，复用调用方传入的 loop）
//	  工具: 全部执行工具（read_file, write_file, run_command, update_tasks...）
//	  职责: 执行具体子任务，返回结果给外层
func RunAutonomous(ctx context.Context, planProv Provider, innerLoop *Loop, task string) (string, error) {
	// 1. 构建外层注册表
	outerReg := NewRegistry()
	RegisterPlanOnlyTools(outerReg) // update_plan

	// 注册 delegate_task：handler 运行内层 Loop
	outerReg.Register(&Tool{
		Name: "delegate_task",
		Description: "把**一项具体子任务**委托给执行 agent（拥有完整工具集：读写文件、运行命令、搜索代码等）。" +
			"执行 agent 独立完成该子任务后返回结果，你根据结果决定下一步。\n\n" +
			"**重要：每次只委托计划中的一项子任务**，不要一次性委托整个大任务。\n" +
			"委托前应先调用 update_plan 将该项标记为 in_progress，完成后立即更新为 done 并标记下一项。",
		Parameters: objSchema(props{
			"task": strProp("要执行的子任务描述（计划中的一项），需清晰说明做什么、涉及哪些文件、预期产出"),
		}, "task"),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			subTask := argStr(args, "task")
			if subTask == "" {
				return "", fmt.Errorf("task 不能为空")
			}

			// 内层 Loop 运行子任务
			msgs, runErr := innerLoop.Run(ctx, subTask, nil)
			if runErr != nil && !errors.Is(runErr, ErrMaxIterations) {
				return "", runErr
			}

			// 提取最终输出
			output := ""
			for i := len(msgs) - 1; i >= 0; i-- {
				if msgs[i].Role == RoleAssistant && strings.TrimSpace(msgs[i].Content) != "" {
					output = msgs[i].Content
					break
				}
			}
			if output == "" {
				output = "(子任务未产出内容)"
			}
			// ★ 包装返回结果，防止外层 agent 误以为「整个任务已完成」。
			// 内层 agent 的输出（含完成报告）原样回灌会让外层 LLM 产生"全部任务已完成"的错觉，
			// 导致外层提前调用 generate_commit_message 并结束，不会继续推进剩余计划项。
			// 加上明确的「这是内层结果，外层据此决策」标记后，外层 agent 能清楚区分
			// 内层的执行结果和外层自身的决策职责，不会混淆。
			return fmt.Sprintf("【内层执行结果】\n%s\n\n---\n请根据以上内层的执行结果，决定下一步：\n1. **立即调用 update_plan 将当前项标记为 done**（无论是否还有剩余项，必须先更新计划状态）\n2. 如果还有剩余计划项，再调用 delegate_task 执行下一项\n3. 如果全部计划项都已是 done 状态，再调用 generate_commit_message 完成收尾", output), nil
		},
	})

	// 注册 generate_commit_message（外层也需要完成标记）
	outerReg.Register(&Tool{
		Name:        "generate_commit_message",
		Description: "全部任务完成后调用此工具记录提交信息，然后输出最终完成总结。",
		Parameters:  objSchema(props{"message": strProp("描述本次变更的句子")}, "message"),
		ReadOnly:    true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			msg := argStr(args, "message")
			return fmt.Sprintf("提交信息已记录: %s", msg), nil
		},
	})

	// ── 注册只读探索工具，让外层设计者能理解项目结构 ──
	root := ""
	if len(WorkspaceRoots) > 0 {
		root = WorkspaceRoots[0]
	}
	registerOuterExplorationTools(outerReg, root)

	// 2. 创建外层 Loop
	outer := &Loop{
		Provider:      planProv,
		Registry:      outerReg,
		System:        outerDesignerPrompt,
		MaxIterations: innerLoop.MaxIterations,
		OnEvent:       innerLoop.OnEvent, // 共用事件推送通道
	}

	// 3. 运行外层 Loop（设计者 agent 开始工作）
	msgs, err := outer.Run(ctx, task, nil)
	if err != nil && !errors.Is(err, ErrMaxIterations) {
		return "", err
	}

	// 兜底：外层 Loop 自然终止（LLM 输出文字未调工具）但计划未完成时，
	// 给一次续推机会，明确告诉它继续执行剩余项。
	if err == nil && planIncomplete(msgs) {
		outer.finishResult = nil
		outer.contentOnlyIters = 0
		msgs, err = outer.Run(ctx, "你制定的计划中还有未完成的项目，请继续执行。不要输出分析文字，直接调用 update_plan 更新状态并 delegate_task 推进下一项。", msgs)
		if err != nil && !errors.Is(err, ErrMaxIterations) {
			return "", err
		}
	}

	// 4. 提取最终输出
	output := ""
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == RoleAssistant && strings.TrimSpace(msgs[i].Content) != "" {
			output = msgs[i].Content
			break
		}
	}
	return output, nil
}

// planIncomplete 从消息列表中找到最后一个 update_plan 的工具结果，
// 检查计划是否还有待办项（完成数 < 总数）。
// update_plan handler 返回格式为 "...完成 X/Y 步"。
func planIncomplete(msgs []Message) bool {
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role != RoleTool {
			continue
		}
		c := msgs[i].Content
		if !strings.Contains(c, "完成 ") || !strings.Contains(c, "/") {
			continue
		}
		// 提取 "完成 X/Y" 中的 X 和 Y
		idx := strings.Index(c, "完成 ")
		if idx < 0 {
			continue
		}
		rest := c[idx+len("完成 "):]
		slash := strings.IndexByte(rest, '/')
		if slash < 0 {
			continue
		}
		doneStr := strings.TrimSpace(rest[:slash])
		tail := rest[slash+1:]
		space := strings.IndexByte(tail, ' ')
		if space < 0 {
			continue
		}
		totalStr := strings.TrimSpace(tail[:space])
		done, total := 0, 0
		if _, e1 := fmt.Sscanf(doneStr, "%d", &done); e1 != nil {
			continue
		}
		if _, e2 := fmt.Sscanf(totalStr, "%d", &total); e2 != nil || total <= 0 {
			continue
		}
		return done < total
	}
	return false
}

// outerDesignerPrompt 外层设计者 Agent 的系统提示语。
const outerDesignerPrompt = `你是项目设计者和总指挥。

# 你的角色
你是整个开发任务的**设计者和总指挥**，不是执行者。你的核心价值在于：
1. **理解目标** — 分析用户到底要什么
2. **探索项目** — 阅读源码、搜索代码、浏览目录，全面了解项目结构
3. **制定计划** — 设计出完整的执行方案
4. **分派任务** — 把具体工作委托给执行 agent
5. **评估调整** — 根据执行结果动态调整计划

# 核心工具
## 探索工具（用于理解项目）
- **read_file** — 读取文件内容，了解现有代码
- **list_files** — 列出目录结构，了解项目组织
- **search_content** — 搜索文件内容（正则匹配）
- **search_files** / **find_files_by_pattern** — 按通配符查找文件
- **codegraph_search** — 搜索代码实体（函数、类型、变量），比 search_content 更精确
- **codegraph_function** — 查找函数/方法的定义位置和签名
- **codegraph_class** — 查看 struct/interface 的完整层次结构
- **codegraph_callers** — 查询哪些函数调用了指定函数
- **codegraph_callees** — 查询指定函数调用了哪些其他函数
- **codegraph_impact** — 分析修改后的影响范围
- **codegraph_file_structure** — 查看文件内部结构
- **codegraph_stats** — 查看代码图谱统计
- **codegraph_git_history** — 查询 Git 提交历史
- **codegraph_entity_history** — 查看实体的变更历史
- **find_symbol** / **get_file_symbols** — 查找符号定义
- **project_info_list** / **project_info_read** / **project_info_search** — 项目知识库
- **project_info_explore** — 项目目录结构概览
- **memory_read** / **memory_list** / **memory_search** — 读写长时记忆（历史决策、项目约定、用户偏好）
- **web_search** / **web_fetch** — 搜索外部文档

## 规划与执行工具
- **update_plan** — 制定和更新执行计划（步骤清单），展示给用户看整体进度
- **delegate_task** — 把具体任务委托给执行 agent，等它完成后看结果
- **generate_commit_message** — 所有任务完成后调用，记录提交信息，然后输出总结

# 工作流程
1. **探索** — 先用探索工具理解项目结构和代码现状（读关键文件、搜索相关代码、查知识库）
2. **规划** — 基于掌握的信息，用 update_plan 列出完整的执行计划
3. **执行** — 逐项调用 delegate_task 执行，每完成一项就更新 plan 状态
4. **调整** — 根据每项的实际执行结果决定下一步：继续 / 调整计划 / 标记完成
5. **收尾** — 全部完成后调用 generate_commit_message，输出最终总结

# 重要原则
- **先探索再规划**：不要凭空列计划，先读文件、搜代码、查知识库，全面理解后再制定方案
- **执行 agent 有完整工具集**（读写文件、运行命令、搜索代码等），可以独立完成你委托的任务
- **你不需要做具体执行**，你的工作是设计、决策、协调
- **每次只委托一个任务**，等结果回来后再决定下一步
- **结果不理想时可以调整**后续计划或重新委托
- **保持计划可见**：每次更新 plan 状态，让用户看到整体进度

# ★ 关键：避免被系统误判为「已完成」
每次 delegate_task 返回结果后，请**立即**调用 update_plan 更新状态（将刚完成项标记为 done、下一项标记为 in_progress），
然后立即调用 delegate_task 执行下一项。
**不要输出分析文字/总结**——系统检测到「无工具调用+有正文」会认为任务已完成并终止循环，
导致剩余计划项无法执行。只有全部计划项都标记为 done 后，才输出最终总结。`
