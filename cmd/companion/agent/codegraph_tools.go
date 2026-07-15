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
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
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

	// ── 15. codegraph_search_by_pattern — 正则模式搜索 ──
	r.Register(&Tool{
		Name: "codegraph_search_by_pattern",
		Description: "用正则表达式在代码实体的名称、签名、文档注释中搜索。" +
			"比 codegraph_search 更精确，支持 scope 过滤（name/signature/docstring/any）。" +
			"支持按实体类型过滤（function/method/struct/interface/variable）。",
		Parameters: objSchema(props{
			"pattern":    strProp("正则表达式，如 'unwrap\\(\\)'、'SELECT .* FROM'、'TODO'"),
			"scope":      strProp("可选：搜索范围，any(默认)/name/signature/docstring"),
			"entityKind": strProp("可选：实体类型过滤，如 function/method/struct/interface/variable"),
			"maxResults": intProp("可选：最大返回数（默认 50）"),
		}, "pattern"),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			g, err := getCodeGraph(root)
			if err != nil {
				return "", err
			}
			qe := codegraph.NewQueryEngine(g)
			req := codegraph.PatternSearchRequest{
				Pattern:    argStr(args, "pattern"),
				Scope:      argStr(args, "scope"),
				MaxResults: argInt(args, "max_results", 50),
			}
			if kind := argStr(args, "entity_kind"); kind != "" {
				req.EntityKind = codegraph.EntityKind(kind)
			}
			hits := qe.SearchByPattern(req)
			return codegraph.PatternSearchText(hits), nil
		},
	})

	// ── 16. codegraph_trace_call_chain — 调用链追踪 ──
	r.Register(&Tool{
		Name: "codegraph_trace_call_chain",
		Description: "追踪函数/方法的调用链。" +
			"支持 callers（反向追踪谁调用了它）、callees（正向追踪它调用了谁）、both（双向）。" +
			"maxDepth 控制追踪深度（默认 5）。返回树形调用链。",
		Parameters: objSchema(props{
			"function":  strProp("函数/方法名（如 'SendRequest'、'handler.Handle'）"),
			"direction": strProp("可选：callers(反向)/callees(正向)/both(双向)，默认 callers"),
			"maxDepth":  intProp("可选：最大深度（默认 5）"),
		}, "function"),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			funcName := argStr(args, "function")
			direction := argStr(args, "direction")
			maxDepth := argInt(args, "max_depth", 5)
			g, err := getCodeGraph(root)
			if err != nil {
				return "", err
			}
			qe := codegraph.NewQueryEngine(g)
			nodes := qe.TraceCallChain(funcName, direction, maxDepth)
			return codegraph.CallChainText(nodes), nil
		},
	})

	// ── 17. codegraph_find_dead_code — 死代码检测 ──
	r.Register(&Tool{
		Name: "codegraph_find_dead_code",
		Description: "检测项目中疑似没有被调用的函数、类型、变量。" +
			"判定方式：函数无 incoming RelCalls 边 + 无其他引用。注意：Go 反射和接口分发可能误报，结果仅供参考。",
		Parameters: objSchema(props{}),
		ReadOnly:   true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			g, err := getCodeGraph(root)
			if err != nil {
				return "", err
			}
			qe := codegraph.NewQueryEngine(g)
			result := qe.FindDeadCode()
			return codegraph.DeadCodeText(result), nil
		},
	})

	// ── 18. codegraph_module_architecture — 模块架构分析 ──
	r.Register(&Tool{
		Name: "codegraph_module_architecture",
		Description: "获取一个目录/模块的架构概览。" +
			"返回：文件数、函数数、导出函数列表、类型列表、外部依赖、" +
			"内部依赖、复杂度热点。用于快速理解一个模块的职责和结构。",
		Parameters: objSchema(props{
			"path": strProp("目录路径（工作区相对路径，如 'cmd/companion/agent'）"),
		}, "path"),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			dirPath := argStr(args, "path")
			g, err := getCodeGraph(root)
			if err != nil {
				return "", err
			}
			qe := codegraph.NewQueryEngine(g)
			arch := qe.GetModuleArchitecture(root, dirPath)
			return codegraph.ModuleArchitectureText(arch), nil
		},
	})

	// ── 19. codegraph_find_entry_points — 入口点发现 ──
	r.Register(&Tool{
		Name: "codegraph_find_entry_points",
		Description: "发现应用程序入口点和执行起点。返回 main 函数、HTTP 处理器、CLI 命令。用于理解应用架构。",
		Parameters: objSchema(props{"entryType": strProp("可选：main/http_handler/cli_command/all，默认 all"), "limit": intProp("可选：最大返回数（默认 50）")}),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			g, err := getCodeGraph(root)
			if err != nil {
				return "", err
			}
			entryType := argStr(args, "entry_type")
			limit := argInt(args, "limit", 50)
			type ep struct{ name, kind, file string; line int }
			var entries []ep
			if entryType == "" || entryType == "all" || entryType == "main" {
				for _, fn := range g.GetEntitiesByKind(codegraph.EntityFunction) {
					if fn.Name == "main" && fn.FilePath != "" {
						entries = append(entries, ep{fn.Name, "main", fn.FilePath, fn.Line})
					}
				}
			}
			if entryType == "" || entryType == "all" || entryType == "http_handler" {
				patterns := []string{"HandleFunc", "Handle", "ServeHTTP", "router.GET", "echo.GET", "gin.GET", "http.Handle"}
				for _, fn := range g.GetEntitiesByKind(codegraph.EntityFunction) {
					for _, p := range patterns {
						if strings.Contains(fn.Signature, p) || strings.Contains(fn.Name, p) {
							entries = append(entries, ep{fn.Name, "http_handler", fn.FilePath, fn.Line})
							break
						}
					}
				}
				for _, fn := range g.GetEntitiesByKind(codegraph.EntityMethod) {
					if strings.Contains(fn.Signature, "ServeHTTP") {
						entries = append(entries, ep{fn.Name, "http_handler", fn.FilePath, fn.Line})
					}
				}
			}
			if entryType == "" || entryType == "all" || entryType == "cli_command" {
				cliPats := []string{"cobra", "Execute", "RunE", "flag.Parse"}
				for _, fn := range g.GetEntitiesByKind(codegraph.EntityFunction) {
					for _, p := range cliPats {
						if strings.Contains(fn.Signature, p) || strings.Contains(fn.Name, p) {
							entries = append(entries, ep{fn.Name, "cli_command", fn.FilePath, fn.Line})
							break
						}
					}
				}
			}
			seen := map[string]bool{}
			var uniq []ep
			for _, e := range entries {
				k := fmt.Sprintf("%s:%d:%s", e.file, e.line, e.name)
				if !seen[k] {
					seen[k] = true
					uniq = append(uniq, e)
				}
			}
			sort.Slice(uniq, func(i, j int) bool {
				if uniq[i].kind != uniq[j].kind {
					return uniq[i].kind < uniq[j].kind
				}
				return uniq[i].file < uniq[j].file
			})
			if len(uniq) > limit {
				uniq = uniq[:limit]
			}
			if len(uniq) == 0 {
				return "未发现已知模式的入口点。", nil
			}
			var b strings.Builder
			b.WriteString(fmt.Sprintf("发现 %d 个入口点：\n\n", len(uniq)))
			for _, e := range uniq {
				b.WriteString(fmt.Sprintf("  [%s] %s (%s:%d)\n", e.kind, e.name, e.file, e.line))
			}
			return b.String(), nil
		},
	})

	// ── 20. codegraph_find_hot_paths — 热路径发现 ──
	r.Register(&Tool{
		Name: "codegraph_find_hot_paths",
		Description: "查找项目中最常被调用的函数，按调用者数量排序。用于识别性能瓶颈和理解核心函数。",
		Parameters: objSchema(props{"limit": intProp("可选：最大返回数（默认 20）")}),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			g, err := getCodeGraph(root)
			if err != nil {
				return "", err
			}
			limit := argInt(args, "limit", 20)
			if limit <= 0 || limit > 100 {
				limit = 20
			}
			all := g.GetEntitiesByKind(codegraph.EntityFunction)
			all = append(all, g.GetEntitiesByKind(codegraph.EntityMethod)...)
			type hf struct{ name, file string; callers int }
			var list []hf
			for _, fn := range all {
				n := len(g.GetPredecessors(fn.ID, codegraph.RelCalls))
				if n > 0 {
					list = append(list, hf{fn.Name, fn.FilePath, n})
				}
			}
			sort.Slice(list, func(i, j int) bool { return list[i].callers > list[j].callers })
			if len(list) > limit {
				list = list[:limit]
			}
			if len(list) == 0 {
				return "未发现被调用的函数。", nil
			}
			var b strings.Builder
			b.WriteString(fmt.Sprintf("最热函数（按调用者数排序，共 %d 条）：\n\n", len(list)))
			b.WriteString(fmt.Sprintf("%-4s %-5s %-30s %s\n", "排名", "调用数", "函数名", "文件"))
			b.WriteString(strings.Repeat("─", 80) + "\n")
			for i, h := range list {
				b.WriteString(fmt.Sprintf("%-4d %-5d %-30s %s\n", i+1, h.callers, h.name, h.file))
			}
			return b.String(), nil
		},
	})

	// ── 21. codegraph_find_by_imports — 按导入查找文件 ──
	r.Register(&Tool{
		Name: "codegraph_find_by_imports",
		Description: "查找所有导入指定模块/包的文件。matchMode 支持 exact/prefix/contains/fuzzy，默认 contains。",
		Parameters: objSchema(props{"moduleName": strProp("模块/包名"), "matchMode": strProp("可选：exact|prefix|contains|fuzzy，默认 contains"), "limit": intProp("可选：最大返回数（默认 50）")}, "moduleName"),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			g, err := getCodeGraph(root)
			if err != nil {
				return "", err
			}
			moduleName := strings.TrimSpace(argStr(args, "module_name"))
			matchMode := argStr(args, "match_mode")
			limit := argInt(args, "limit", 50)
			if moduleName == "" {
				return "", fmt.Errorf("moduleName 不能为空")
			}
			if matchMode == "" {
				matchMode = "contains"
			}
			fileImports := map[string][]string{}
			for _, file := range g.GetEntitiesByKind(codegraph.EntityFile) {
				if file.FilePath == "" {
					continue
				}
				for _, imp := range g.GetSuccessors(file.ID, codegraph.RelImports) {
					impName := imp.Name
					if impName == "" {
						impName = imp.FQN
					}
					match := false
					switch matchMode {
					case "exact":
						match = strings.EqualFold(impName, moduleName)
					case "prefix":
						match = strings.HasPrefix(strings.ToLower(impName), strings.ToLower(moduleName))
					default:
						match = strings.Contains(strings.ToLower(impName), strings.ToLower(moduleName))
					}
					if match {
						fileImports[file.FilePath] = append(fileImports[file.FilePath], impName)
					}
				}
			}
			if len(fileImports) == 0 {
				return fmt.Sprintf("未找到导入「%s」的文件。", moduleName), nil
			}
			files := make([]string, 0, len(fileImports))
			for f := range fileImports {
				files = append(files, f)
			}
			sort.Strings(files)
			if len(files) > limit {
				files = files[:limit]
			}
			var b strings.Builder
			b.WriteString(fmt.Sprintf("导入「%s」的文件（共 %d 个）：\n\n", moduleName, len(fileImports)))
			for _, f := range files {
				b.WriteString(fmt.Sprintf("  %s\n", f))
				b.WriteString(fmt.Sprintf("    └ %s\n", fileImports[f][0]))
				if len(fileImports[f]) > 1 {
					b.WriteString(fmt.Sprintf("    └ ... 共 %d 个匹配导入\n", len(fileImports[f])))
				}
			}
			if len(files) < len(fileImports) {
				b.WriteString(fmt.Sprintf("\n... 还有 %d 个文件未显示。", len(fileImports)-len(files)))
			}
			return b.String(), nil
		},
	})

	// ── 22. codegraph_get_detailed_symbol — 符号详情一体化 ──
	r.Register(&Tool{
		Name: "codegraph_get_detailed_symbol",
		Description: "获取指定代码符号的详细上下文（源码+调用者+被调用者+元信息），一次调用获取完整符号画像。",
		Parameters: objSchema(props{
			"query":          strProp("符号名（函数名、类型名等）"),
			"includeSource":  boolProp("可选：包含源码正文（默认 true）"),
			"includeCallers": boolProp("可选：包含调用者列表（默认 true）"),
		}, "query"),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			query := strings.TrimSpace(argStr(args, "query"))
			if query == "" {
				return "", fmt.Errorf("query 不能为空")
			}
			g, err := getCodeGraph(root)
			if err != nil {
				return "", err
			}
			qe := codegraph.NewQueryEngine(g)
			entities := g.SearchEntities(query)
			if len(entities) == 0 {
				return fmt.Sprintf("未找到匹配「%s」的代码符号。", query), nil
			}
			if len(entities) > 5 {
				entities = entities[:5]
			}
			var b strings.Builder
			for i, e := range entities {
				if i > 0 {
					b.WriteString("\n" + strings.Repeat("─", 60) + "\n\n")
				}
				b.WriteString(fmt.Sprintf("## %s (%s)\n", e.Name, string(e.Kind)))
				b.WriteString(fmt.Sprintf("- **文件**: %s:%d-%d\n", e.FilePath, e.Line, e.EndLine))
				b.WriteString(fmt.Sprintf("- **FQN**: %s\n", e.FQN))
				if e.Signature != "" {
					b.WriteString(fmt.Sprintf("- **签名**: %s\n", e.Signature))
				}
				if argBool(args, "include_callers") || true {
					callers := qe.GetCallers(e.Name)
					b.WriteString(fmt.Sprintf("- **调用者**: %d 个\n", len(callers)))
					for j := 0; j < minInt(10, len(callers)); j++ {
						c := callers[j]
						b.WriteString(fmt.Sprintf("  · %s (%s:%d)\n", c.CallerName, c.CallerFile, c.CallerLine))
					}
					callees := qe.GetCallees(e.Name)
					b.WriteString(fmt.Sprintf("- **被调用者**: %d 个\n", len(callees)))
					for j := 0; j < minInt(10, len(callees)); j++ {
						c := callees[j]
						b.WriteString(fmt.Sprintf("  · %s (%s:%d)\n", c.CalleeName, c.CallerFile, c.CallerLine))
					}
				}
				if (argBool(args, "include_source") || true) && e.FilePath != "" {
					data, rErr := os.ReadFile(filepath.Join(root, e.FilePath))
					if rErr == nil {
						lines := strings.Split(string(data), "\n")
						start := max(0, e.Line-1)
						end := e.EndLine
						if end <= start {
							end = start + 20
						}
						end = min(len(lines), end)
						end = min(start+80, end)
						if start < end {
							b.WriteString(fmt.Sprintf("\n### 源码（%s:%d-%d）\n", e.FilePath, start+1, end))
							b.WriteString("```go\n")
							for ln := start; ln < end; ln++ {
								b.WriteString(fmt.Sprintf("%d\t%s\n", ln+1, lines[ln]))
							}
							b.WriteString("```\n")
						}
					}
				}
			}
			return b.String(), nil
		},
	})

	// ── 23. codegraph_find_dead_imports — 死导入检测 ──
	r.Register(&Tool{
		Name: "codegraph_find_dead_imports",
		Description: "查找文件中已导入但从未使用的模块（未使用的 import 语句）。",
		Parameters: objSchema(props{
			"file":  strProp("可选：仅分析指定文件；省略则分析全部"),
			"limit": intProp("可选：最大返回数（默认 50）"),
		}),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			g, err := getCodeGraph(root)
			if err != nil {
				return "", err
			}
			fileFilter := strings.TrimSpace(argStr(args, "file"))
			limit := argInt(args, "limit", 50)
			type deadImport struct{ file, imp string }
			var dead []deadImport
			for _, fe := range g.GetEntitiesByKind(codegraph.EntityFile) {
				if fe.FilePath == "" {
					continue
				}
				if fileFilter != "" && !strings.Contains(fe.FilePath, fileFilter) {
					continue
				}
				imports := g.GetSuccessors(fe.ID, codegraph.RelImports)
				if len(imports) == 0 {
					continue
				}
				fileEntities := g.GetEntitiesByFile(fe.FilePath)
				for _, imp := range imports {
					impName := imp.Name
					if impName == "" {
						impName = imp.FQN
					}
					lastSeg := impName
					if idx := strings.LastIndex(impName, "/"); idx >= 0 {
						lastSeg = impName[idx+1:]
					}
					if lastSeg == "" {
						lastSeg = impName
					}
					used := false
					for _, fe2 := range fileEntities {
						if fe2.Kind == codegraph.EntityFunction || fe2.Kind == codegraph.EntityMethod {
							if strings.Contains(fe2.FQN, lastSeg) || strings.Contains(fe2.Signature, lastSeg) {
								used = true
								break
							}
						}
						if fe2.Kind == codegraph.EntityType || fe2.Kind == codegraph.EntityStruct || fe2.Kind == codegraph.EntityInterface {
							if strings.Contains(fe2.Signature, lastSeg) {
								used = true
								break
							}
						}
					}
					if !used {
						dead = append(dead, deadImport{fe.FilePath, impName})
					}
				}
			}
			if len(dead) == 0 {
				return "未发现死导入（或图谱未收录所有文件）。", nil
			}
			sort.Slice(dead, func(i, j int) bool {
				if dead[i].file != dead[j].file { return dead[i].file < dead[j].file }
				return dead[i].imp < dead[j].imp
			})
			if len(dead) > limit { dead = dead[:limit] }
			byFile := map[string][]string{}
			for _, d := range dead { byFile[d.file] = append(byFile[d.file], d.imp) }
			var b strings.Builder
			b.WriteString(fmt.Sprintf("发现 %d 个可能未使用的导入（在 %d 个文件中）：\n\n", len(dead), len(byFile)))
			for f, imps := range byFile {
				b.WriteString(fmt.Sprintf("  %s\n", f))
				for _, imp := range imps { b.WriteString(fmt.Sprintf("    └ %s\n", imp)) }
			}
			return b.String(), nil
		},
	})

	// ── 24. codegraph_search_by_error — 错误模式搜索 ──
	r.Register(&Tool{
		Name: "codegraph_search_by_error",
		Description: "查找项目中抛出或处理错误的函数。扫描函数签名中的错误模式" +
			"（errors.New/fmt.Errorf/panic/if err != nil 等）。",
		Parameters: objSchema(props{
			"mode":      strProp("可选：throws(产生错误)/catches(处理错误)/any(全部)，默认 any"),
			"errorType": strProp("可选：指定错误类型名过滤，如 \"ErrNotFound\""),
			"limit":     intProp("可选：最大返回数（默认 50）"),
		}),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			g2, err := getCodeGraph(root)
			if err != nil { return "", err }
			mode := argStr(args, "mode"); errorType := strings.TrimSpace(argStr(args, "error_type"))
			limit := argInt(args, "limit", 50)
			allFuncs := g2.GetEntitiesByKind(codegraph.EntityFunction)
			allFuncs = append(allFuncs, g2.GetEntitiesByKind(codegraph.EntityMethod)...)
			type hit struct{ name, file, sig string; line int; patterns []string }
			var results []hit
			for _, fn := range allFuncs {
				var matched []string; sig := fn.Signature + " " + fn.Doc
				if mode == "" || mode == "any" || mode == "throws" {
					for _, p := range []string{"errors.New", "fmt.Errorf", "panic(", "return.*err"} {
						if matchedP, _ := regexp.MatchString(p, sig); matchedP { matched = append(matched, p) }
					}
				}
				if mode == "" || mode == "any" || mode == "catches" {
					for _, p := range []string{"if err != nil", "if err == nil", "catch(", "except "} {
						if strings.Contains(sig, p) { matched = append(matched, p) }
					}
				}
				if errorType != "" {
					if !strings.Contains(fn.Signature, errorType) && !strings.Contains(fn.FQN, errorType) { continue }
					matched = append(matched, "err:"+errorType)
				}
				if len(matched) > 0 { results = append(results, hit{fn.Name, fn.FilePath, fn.Signature, fn.Line, matched}) }
			}
			if len(results) == 0 { return "未找到错误匹配。", nil }
			sort.Slice(results, func(i, j int) bool {
				if results[i].file != results[j].file { return results[i].file < results[j].file }
				return results[i].name < results[j].name
			})
			if len(results) > limit { results = results[:limit] }
			var b strings.Builder
			b.WriteString(fmt.Sprintf("匹配 %d 个函数（显示 %d 个）：\n\n", len(results), minInt(len(results), limit)))
			for _, h := range results {
				b.WriteString(fmt.Sprintf("  %s (%s:%d)\n", h.name, h.file, h.line))
				b.WriteString(fmt.Sprintf("    模式: %s\n", strings.Join(h.patterns, ", ")))
			}
			return b.String(), nil
		},
	})

	// ── 25-26. codegraph_index_markdown + codegraph_search_docs — 文档索引与搜索 ──
	r.Register(&Tool{
		Name: "codegraph_index_markdown",
		Description: "索引项目中的 Markdown 文档文件，按标题分段建立搜索索引。" +
			"让后续的 codegraph_search_docs 能检索到相关内容。",
		Parameters: objSchema(props{
			"path": strProp("可选：指定 Markdown 文件路径；省略则扫描工作区所有 .md 文件"),
		}),
		ReadOnly: false,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			g3, err := getCodeGraph(root)
			if err != nil { return "", err }
			mdPath := strings.TrimSpace(argStr(args, "path"))
			var mdFiles []string
			if mdPath != "" { mdFiles = append(mdFiles, mdPath) } else {
				skipDirs := map[string]bool{".git": true, "node_modules": true, ".pair": true, "vendor": true}
				filepath.WalkDir(root, func(p string, d fs.DirEntry, werr error) error {
					if werr != nil { return nil }
					if d.IsDir() {
						if skipDirs[d.Name()] { return fs.SkipDir }; return nil
					}
					if strings.HasSuffix(strings.ToLower(p), ".md") {
						rel, _ := filepath.Rel(root, p); mdFiles = append(mdFiles, rel)
					}
					return nil
				})
			}
			indexed := 0
			for _, mf := range mdFiles {
				fullPath := filepath.Join(root, mf)
				data, rErr := os.ReadFile(fullPath)
				if rErr != nil { continue }
				sections := splitMarkdownSections(string(data))
				for i, sec := range sections {
					g3.AddEntity(&codegraph.Entity{
						ID: fmt.Sprintf("doc:%s#%d", mf, i), Kind: codegraph.EntityDocSection,
						Name: fmt.Sprintf("%s - %s", mf, sec.heading), FilePath: mf, Line: sec.line, Doc: sec.body,
					})
				}
				indexed++
			}
			return fmt.Sprintf("已索引 %d 个 Markdown 文件，共 %d 个文档节。使用 codegraph_search_docs 搜索。", indexed, len(mdFiles)), nil
		},
	})

	r.Register(&Tool{
		Name: "codegraph_search_docs",
		Description: "搜索已索引的项目文档（通过 codegraph_index_markdown 预先索引）。" +
			"支持关键词查询，返回匹配的文档章节（含标题和原文片段）。",
		Parameters: objSchema(props{"query": strProp("搜索关键词"), "limit": intProp("可选：最大返回数（默认 5）")}, "query"),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			g4, err := getCodeGraph(root)
			if err != nil { return "", err }
			query := strings.TrimSpace(argStr(args, "query")); limit := argInt(args, "limit", 5)
			if query == "" { return "", fmt.Errorf("query 不能为空") }
			sections := g4.GetEntitiesByKind(codegraph.EntityDocSection)
			type match struct{ entity *codegraph.Entity; score int }
			var matches []match
			q := strings.ToLower(query)
			for _, sec := range sections {
				score := 0; name := strings.ToLower(sec.Name); doc := strings.ToLower(sec.Doc)
				for _, w := range strings.Fields(q) {
					if len(w) < 2 { continue }
					if strings.Contains(name, w) { score += 3 }
					if strings.Contains(doc, w) { score++ }
				}
				if score > 0 { matches = append(matches, match{sec, score}) }
			}
			sort.Slice(matches, func(i, j int) bool { return matches[i].score > matches[j].score })
			if len(matches) > limit { matches = matches[:limit] }
			if len(matches) == 0 { return "未找到匹配的文档章节。请先运行 codegraph_index_markdown 索引文档。", nil }
			var b strings.Builder
			b.WriteString(fmt.Sprintf("找到 %d 个相关文档节：\n\n", len(matches)))
			for _, m := range matches {
				b.WriteString(fmt.Sprintf("### %s\n", m.entity.Name))
				b.WriteString(fmt.Sprintf("文件: %s (行 %d)\n\n", m.entity.FilePath, m.entity.Line))
				doc := m.entity.Doc
				if len([]rune(doc)) > 500 { doc = string([]rune(doc)[:500]) + "…" }
				b.WriteString(doc + "\n\n---\n\n")
			}
			return b.String(), nil
		},
	})

	// ── 27. codegraph_verify_design — 设计文档验证 ──
	r.Register(&Tool{
		Name: "codegraph_verify_design",
		Description: "对照设计文档检查代码实现是否存在。从文档中提取反引号包裹的标识符" +
			"（如 UserService、authenticate()），并在项目代码中搜索确认是否存在。",
		Parameters: objSchema(props{"docFile": strProp("Markdown 设计文档路径（如 'docs/ARCHITECTURE.md'）")}, "docFile"),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			docFile := strings.TrimSpace(argStr(args, "doc_file"))
			if docFile == "" { return "", fmt.Errorf("docFile 不能为空") }
			fullPath := filepath.Join(root, docFile)
			data, rErr := os.ReadFile(fullPath)
			if rErr != nil { return "", fmt.Errorf("读取文档失败: %w", rErr) }
			content := string(data)
			identRe := regexp.MustCompile("`([^`]+)`")
			idents := identRe.FindAllStringSubmatch(content, -1)
			unique := map[string]bool{}
			var symbols []string
			for _, m := range idents {
				s := strings.TrimSpace(m[1])
				if s != "" && len(s) > 1 && !unique[s] { unique[s] = true; symbols = append(symbols, s) }
			}
			if len(symbols) == 0 { return "文档中未发现反引号包裹的代码标识符。", nil }
			g5, err := getCodeGraph(root)
			if err != nil { return "", err }
			var found, notFound []string
			for _, sym := range symbols {
				if len(g5.SearchEntities(sym)) > 0 { found = append(found, sym) } else { notFound = append(notFound, sym) }
			}
			var b strings.Builder
			pct := 0
			if len(symbols) > 0 { pct = len(found) * 100 / len(symbols) }
			b.WriteString(fmt.Sprintf("设计文档「%s」验证结果：\n", docFile))
			b.WriteString(fmt.Sprintf("共提取 %d 个标识符，代码中存在 %d 个 (%d%%)，缺失 %d 个\n\n", len(symbols), len(found), pct, len(notFound)))
			if len(notFound) > 0 {
				b.WriteString("⚠️ 以下标识符在项目代码中未找到：\n")
				for _, s := range notFound { b.WriteString(fmt.Sprintf("  · `%s`\n", s)) }
				b.WriteString("\n可能原因：文档过期、命名变更、或描述了未实现的计划功能。\n")
			}
			if len(found) > 0 {
				b.WriteString("✅ 以下标识符已在代码中确认：\n")
				maxShow := minInt(20, len(found))
				for i := 0; i < maxShow; i++ { b.WriteString(fmt.Sprintf("  · `%s`\n", found[i])) }
				if len(found) > maxShow { b.WriteString(fmt.Sprintf("  · ... 还有 %d 个\n", len(found)-maxShow)) }
			}
			return b.String(), nil
		},
	})

	// ── 28. codegraph_pr_context — PR 影响分析 ──
	r.Register(&Tool{
		Name: "codegraph_pr_context",
		Description: "分析当前分支与目标分支之间的代码变更影响范围。运行 git diff 找到变更文件，" +
			"然后对每个变更函数分析调用者和影响范围。用于 PR 审查时评估负面影响。",
		Parameters: objSchema(props{
			"baseBranch": strProp("可选：基准分支名（默认 'main'）"),
			"format":     strProp("可选：输出格式 json/markdown，默认 markdown"),
		}),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			baseBranch := strings.TrimSpace(argStr(args, "base_branch"))
			if baseBranch == "" { baseBranch = "main" }
			// 执行 git diff
			diffOut, diffErr := runGit(ctx, root, "diff", "--stat", baseBranch+"...HEAD")
			if diffErr != nil {
				return "", fmt.Errorf("git diff 失败: %w", diffErr)
			}
			if strings.TrimSpace(diffOut) == "" || diffOut == "(无输出)" {
				return "当前分支与 " + baseBranch + " 无差异。", nil
			}
			// 获取变更文件列表
			nameOut, _ := runGit(ctx, root, "diff", "--name-only", baseBranch+"...HEAD")
			files := strings.Fields(strings.TrimSpace(nameOut))
			g6, err := getCodeGraph(root)
			if err != nil {
				return "无法加载 codegraph，仅显示文件变更：\n" + diffOut, nil
			}
			var b strings.Builder
			b.WriteString(fmt.Sprintf("## PR 影响分析（vs %s）\n\n", baseBranch))
			b.WriteString("### 变更文件\n```\n" + diffOut + "```\n\n")
			totalAffected := 0
			for _, f := range files {
				f = strings.TrimSpace(f)
				if f == "" { continue }
				fe := g6.GetEntitiesByFile(f)
				if len(fe) > 0 {
					callers := map[string]bool{}
					for _, e := range fe {
						if e.Kind == codegraph.EntityFunction || e.Kind == codegraph.EntityMethod {
							ci := codegraph.NewQueryEngine(g6).GetCallers(e.Name)
							for _, c := range ci { callers[c.CallerName+"@"+c.CallerFile] = true }
						}
					}
					if len(callers) > 0 {
						b.WriteString(fmt.Sprintf("- **%s** — 影响 %d 个外部调用者\n", f, len(callers)))
						for c := range callers {
							if totalAffected < 20 { b.WriteString(fmt.Sprintf("  · %s\n", c)) }
						}
						totalAffected += len(callers)
					} else {
						b.WriteString(fmt.Sprintf("- %s （无外部调用者）\n", f))
					}
				} else {
					b.WriteString(fmt.Sprintf("- %s （图谱中无此文件）\n", f))
				}
			}
			return b.String(), nil
		},
	})

}


// splitMarkdownSections 将 Markdown 内容按标题分割为多个文档节。
type mdSection struct {
	heading string
	body    string
	line    int
}

func splitMarkdownSections(content string) []mdSection {
	lines := strings.Split(content, "\n")
	var sections []mdSection
	var cur *mdSection
	headingRe := regexp.MustCompile("^#{1,4}\\s+(.+)$")
	for i, line := range lines {
		m := headingRe.FindStringSubmatch(line)
		if m != nil {
			if cur != nil {
				sections = append(sections, *cur)
			}
			cur = &mdSection{heading: m[1], line: i + 1}
		} else if cur != nil {
			if cur.body != "" {
				cur.body += "\n"
			}
			cur.body += line
		}
	}
	if cur != nil {
		sections = append(sections, *cur)
	}
	return sections
}

// CodeGraphProjectSummary 从 codegraph 提取项目结构的关键信息，
// 供系统提示注入。返回 ~200 字的紧凑概览：
// 入口、核心包（目录级）、文件/函数统计。
func CodeGraphProjectSummary(cg *codegraph.Graph, root string) string {
	stats := cg.Stats()
	if stats.FileCount == 0 {
		return ""
	}
	// 收集项目内部包（通过文件路径判断——相对路径不含外部 vendor）
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