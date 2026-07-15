package agent

// codegraph_tools.go — 代码知识图谱的 Agent 工具注册。
// 将 codegraph 包的构建、查询、搜索、影响分析、Git 历史等功能
// 注册为 Agent 可调用的工具，所有操作在 Agent 进程内完成（无 MCP 依赖）。
//
// 本文件放置在 cmd/companion/agent/ 中，以直接访问 agent 的 Registry 和工具类型。

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/hoonfeng/paircode/pkg/codegraph"
	"github.com/hoonfeng/paircode/pkg/memory"
)

// ── 全局状态 ──────────────────────────────────────────

var (
	cgGraph     *codegraph.Graph
	cgGraphMu   sync.RWMutex
	cgRoot      string
	cgInitOnce  sync.Once

	// cgDB 共享 SQLite 数据库连接，非空时 codegraph 使用 SQLiteStore 代替 JSONStore。
	cgDB *sql.DB
)

// SetCodeGraphDB 设置 codegraph 使用的共享数据库连接。
// 由 web_server.go 在 buildWebLoopOpts 中调用。
func SetCodeGraphDB(db *sql.DB) {
	cgDB = db
}

// ensureCodeGraph 确保图谱已初始化。首次调用时自动加载或构建。
func ensureCodeGraph(root string) (*codegraph.Graph, error) {
	cgRoot = root
	var loadErr error
	cgInitOnce.Do(func() {
		var built bool
		cgGraph, built, loadErr = codegraph.EnsureBuildIfNeeded(root)
		if loadErr == nil && built {
			// 刚构建完成
		}
	})
	if loadErr != nil {
		return nil, loadErr
	}
	return cgGraph, nil
}

// EnsureCodeGraph 公开包装器，供 web_server.go 调用。
func EnsureCodeGraph(root string) (*codegraph.Graph, error) {
	return ensureCodeGraph(root)
}

// getCodeGraph 获取当前图谱实例（确保已初始化）。
func getCodeGraph(root string) (*codegraph.Graph, error) {
	cgGraphMu.RLock()
	if cgGraph != nil {
		cgGraphMu.RUnlock()
		return cgGraph, nil
	}
	cgGraphMu.RUnlock()
	return ensureCodeGraph(root)
}

// resetCodeGraph 重置图谱（下次调用时重新构建）。
func resetCodeGraph() {
	cgGraphMu.Lock()
	cgGraph = nil
	cgInitOnce = sync.Once{}
	cgGraphMu.Unlock()
}

// ── 工具注册 ──────────────────────────────────────────

// registerCodeGraphTools 注册所有代码知识图谱相关工具。
// 由 RegisterDefaultTools 调用。
func registerCodeGraphTools(r *Registry, root string) {
	// ── 1. codegraph_build — 构建/重建图谱 ──
	r.Register(&Tool{
		Name: "codegraph_build",
		Description: "构建或重建代码知识图谱。解析项目所有 Go 源文件，" +
			"提取文件、包、函数、方法、结构体、接口、变量、常量等实体，" +
			"以及包含、定义、调用、导入等关系。支持增量更新（只重新解析变更的文件）。" +
			"参数 rebuild=true 强制全量重建。",
		Parameters: objSchema(props{
			"rebuild": boolProp("可选：强制全量重建（默认 false，增量更新）"),
		}),
		ReadOnly: false,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			rebuild := argBool(args, "rebuild")

			moduleName := codegraph.DetectModuleName(root)
			config := codegraph.DefaultBuildConfig(root)
			config.ModuleName = moduleName
			config.AutoSave = true

			builder := codegraph.NewBuilder(config)
			// 共享 DB 连接时使用 SQLiteStore（增量写入）
			if cgDB != nil {
				builder.SetStore(codegraph.NewSQLiteStore(root, cgDB))
			}

			var result *codegraph.BuildResult
			var err error

			if rebuild {
				builder.Graph().Clear()
				codegraph.SaveGraph(root, builder.Graph())
				result, err = builder.BuildFull()
			} else {
				result, err = builder.IncrementalBuild()
			}

			if err != nil {
				return "", fmt.Errorf("构建图谱失败: %w", err)
			}

			// 更新缓存
			cgGraphMu.Lock()
			cgGraph = builder.Graph()
			cgInitOnce = sync.Once{}
			cgRoot = root
			cgGraphMu.Unlock()

			return codegraph.BuildResultText(result), nil
		},
	})

	// ── 2. codegraph_stats — 图谱统计 ──
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

	// ── 3. codegraph_file_structure — 文件结构树 ──
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

	// ── 4. codegraph_function — 函数定义定位 ──
	r.Register(&Tool{
		Name: "codegraph_function",
		Description: "按名称查找函数/方法的定义位置。支持函数名、包名.函数名、或接收者.方法名。" +
			"返回文件路径、行号、签名等信息。",
		Parameters: objSchema(props{
			"name": strProp("函数名（如 'main'、'ServeHTTP'、'foo.Bar'）"),
		}, "name"),
		ReadOnly: true,
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

	// ── 5. codegraph_class — 类型层次 ──
	r.Register(&Tool{
		Name: "codegraph_class",
		Description: "获取类型（struct/interface）的完整层次结构：字段、方法、嵌入类型。" +
			"支持结构体名或接口名。",
		Parameters: objSchema(props{
			"name": strProp("类型名（如 'Server'、'Handler'）"),
		}, "name"),
		ReadOnly: true,
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

	// ── 6. codegraph_callers — 调用者查询 ──
	r.Register(&Tool{
		Name: "codegraph_callers",
		Description: "查询哪些函数调用了指定的函数/方法。用于理解函数被使用的情况。" +
			"返回调用者的文件路径和行号。",
		Parameters: objSchema(props{
			"name": strProp("函数/方法名（如 'SendRequest'、'handler.Handle'）"),
		}, "name"),
		ReadOnly: true,
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

	// ── 7. codegraph_callees — 被调用者查询 ──
	r.Register(&Tool{
		Name: "codegraph_callees",
		Description: "查询指定的函数/方法调用了哪些其他函数。用于理解函数的内部调用情况。" +
			"返回被调用者的名称和调用位置。",
		Parameters: objSchema(props{
			"name": strProp("函数/方法名（如 'handleRequest'）"),
		}, "name"),
		ReadOnly: true,
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

	// ── 8. codegraph_impact — 影响分析 ──
	r.Register(&Tool{
		Name: "codegraph_impact",
		Description: "分析修改某个函数/类型/文件后可能影响的范围。" +
			"基于调用图进行可达性分析，返回受影响的文件、函数列表和传播路径。" +
			"用于回答「修改这个函数会影响哪些地方？」",
		Parameters: objSchema(props{
			"entity":  strProp("实体标识（函数名、类型名或文件路径，如 'SendRequest'、'cmd/main.go'）"),
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

	// ── 9. codegraph_search — 代码搜索 ──
	r.Register(&Tool{
		Name: "codegraph_search",
		Description: "在代码知识图谱中搜索实体（函数、类型、变量、文件等）。" +
			"支持按名称搜索和按类型过滤。返回匹配实体的位置、签名和相关度评分。" +
			"比 search_content 更精确，因为基于结构化理解而非纯文本匹配。",
		Parameters: objSchema(props{
			"query": strProp("搜索关键词（函数名、类型名、变量名等）"),
			"scope": strProp("可选：搜索范围，可选值: all(全部)/file(文件)/function(函数)/type(类型)/variable(变量)/package(包)，默认 all"),
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

	// ── 10. codegraph_git_history — Git 提交历史 ──
	r.Register(&Tool{
		Name: "codegraph_git_history",
		Description: "查询 Git 提交历史，并关联到代码实体。可以查询最近提交、" +
			"影响某个文件的提交，或者某个实体的变更历史。" +
			"用于回答「这个函数是谁改的？」、「这个 bug 是哪次提交引入的？」",
		Parameters: objSchema(props{
			"file": strProp("可选：文件路径，查询影响该文件的提交历史"),
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

	// ── 11. codegraph_entity_history — 实体变更历史 ──
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

			// 尝试按实体名搜索
			entities := g.SearchEntities(entityName)
			if len(entities) == 0 {
				// 回退：按文件路径查询 git 历史
				gh := codegraph.NewGitHistory(root)
				commits, err := gh.GetCommitsAffecting(entityName, 20)
				if err != nil {
					return "", fmt.Errorf("未找到实体且查询文件历史失败: %v", err)
				}
				return codegraph.GitHistoryText(commits), nil
			}

			// 取第一个匹配实体的历史
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

	// ── 12. codegraph_get_edit_context — 编辑上下文聚合 ──
	r.Register(&Tool{
		Name: "codegraph_get_edit_context",
		Description: "获取修改某个代码位置所需的完整上下文。" +
			"一次调用返回：符号源码、调用者列表、关联测试、近期 Git 历史、相关记忆。" +
			"比分别调用多个工具更高效。参数 maxTokens 控制返回内容的 token 预算。",
		Parameters: objSchema(props{
			"file":      strProp("文件路径（工作区相对路径，如 'cmd/main.go'）"),
			"line":      intProp("行号（1 基，目标函数/类型所在行）"),
			"maxTokens": intProp("可选：token 预算上限（默认 4000，0 不限）"),
		}, "file", "line"),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			filePath := argStr(args, "file")
			line := argInt(args, "line", 1)
			maxTokens := argInt(args, "max_tokens", 4000)

			g, err := getCodeGraph(root)
			if err != nil {
				return "", err
			}
			qe := codegraph.NewQueryEngine(g)

			// 构建记忆回调函数
			memoryFunc := func(query string) []codegraph.MemoryBrief {
				// 从 pkg/memory 搜索相关记忆
				entries := memory.Search(query)
				var mems []codegraph.MemoryBrief
				for _, e := range entries {
					mems = append(mems, codegraph.MemoryBrief{
						Title:   e.Title,
						Summary: e.Summary,
						Tags:    e.Tags,
					})
				}
				return mems
			}

			ctxResult := codegraph.GetEditContext(qe, root, filePath, line, maxTokens, memoryFunc)
			return codegraph.EditContextText(ctxResult), nil
		},
	})

	// ── 13. codegraph_find_related_tests — 测试发现 ──
	r.Register(&Tool{
		Name: "codegraph_find_related_tests",
		Description: "查找与指定函数/方法关联的测试。发现方式：" +
			"（1）测试函数调用了目标函数；（2）命名约定匹配（TestXxx ↔ Xxx）。" +
			"返回测试文件路径、行号和源码片段。",
		Parameters: objSchema(props{
			"function": strProp("函数/方法名（如 'SendRequest'、'handler.Handle'）"),
		}, "function"),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			funcName := argStr(args, "function")
			g, err := getCodeGraph(root)
			if err != nil {
				return "", err
			}
			qe := codegraph.NewQueryEngine(g)
			result := codegraph.FindRelatedTests(qe, root, funcName)
			return codegraph.RelatedTestsText(result), nil
		},
	})

	// ── 14. codegraph_analyze_complexity — 圈复杂度分析 ──
	r.Register(&Tool{
		Name: "codegraph_analyze_complexity",
		Description: "测量代码圈复杂度，用于评估重构优先级。" +
			"返回每个函数的复杂度评分（1=最低）、等级（A-E）和行数。" +
			"复杂度 >10 建议考虑重构，>20 为高风险。file 指定文件分析单个文件，省略则分析所有函数。",
		Parameters: objSchema(props{
			"file": strProp("可选：文件路径（工作区相对路径），分析单个文件；省略则分析全部"),
		}),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			filePath := strings.TrimSpace(argStr(args, "file"))
			g, err := getCodeGraph(root)
			if err != nil {
				return "", err
			}
			qe := codegraph.NewQueryEngine(g)
			report := codegraph.AnalyzeComplexity(qe, root, filePath)
			return codegraph.ComplexityReportText(report), nil
		},
	})
}

// CodeGraphProjectSummary 从 codegraph 提取项目结构的关键信息，
// 供系统提示注入。返回 ~200 字的紧凑概览：
// 入口、核心包（目录级）、文件/函数统计。
// 比单纯的数字统计更有实际指引价值。
func CodeGraphProjectSummary(cg *codegraph.Graph, root string) string {
	stats := cg.Stats()
	if stats.FileCount == 0 {
		return ""
	}

	// 收集项目内部包（通过文件路径判断——相对路径不含外部 vendor）
	pkgDirs := make(map[string]struct{}) // 顶层目录
	mainFiles := make([]string, 0)
	totalInternal := 0

	for _, fe := range cg.GetEntitiesByKind(codegraph.EntityFile) {
		if fe.FilePath == "" {
			continue
		}
		totalInternal++
		// 提取顶层分类目录
		dir := filepath.Dir(fe.FilePath)
		parts := strings.SplitN(dir, string(filepath.Separator), 2)
		if len(parts) > 0 && parts[0] != "." && parts[0] != "" {
			topDir := parts[0]
			// 只统计有意义的内部分类目录
			if topDir != ".git" && topDir != "node_modules" && topDir != "vendor" {
				pkgDirs[topDir] = struct{}{}
			}
		}
	}

	// 找入口文件
	for _, fe := range cg.GetEntitiesByKind(codegraph.EntityFile) {
		if fe.Name == "main.go" || fe.Name == "main.ts" || fe.Name == "main.py" ||
			fe.Name == "index.js" || fe.Name == "app.go" || fe.Name == "Server.go" {
			mainFiles = append(mainFiles, fe.FilePath)
		}
	}
	// 也找 main 函数
	for _, fn := range cg.GetEntitiesByKind(codegraph.EntityFunction) {
		if fn.Name == "main" && fn.FilePath != "" {
			if !contains(mainFiles, fn.FilePath) {
				mainFiles = append(mainFiles, fn.FilePath)
			}
		}
	}

	// 排序顶层目录
	sortedDirs := make([]string, 0, len(pkgDirs))
	for d := range pkgDirs {
		sortedDirs = append(sortedDirs, d)
	}
	sort.Strings(sortedDirs)

	// 构建摘要
	var b strings.Builder
	if len(mainFiles) > 0 {
		b.WriteString("入口: ")
		for i, mf := range mainFiles {
			if i > 2 {
				b.WriteString("…")
				break
			}
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(mf)
		}
		b.WriteString("\n")
	}

	b.WriteString(fmt.Sprintf("核心模块: %s\n", strings.Join(sortedDirs, ", ")))
	b.WriteString(fmt.Sprintf("文件: %d | 函数: %d | 类型: %d | 包: %d",
		stats.FileCount,
		stats.KindCounts["function"]+stats.KindCounts["method"],
		stats.KindCounts["struct"],
		stats.PackageCount))

	result := b.String()
	if len(result) > 400 {
		result = result[:400] + "…"
	}
	return result
}