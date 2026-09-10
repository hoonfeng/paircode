// ═══════════════════════════════════════════════════════════════
// deferred_tools.go — 工具延迟暴露（对齐 codex ToolExposure::Deferred + tool_search）
//
// 设计（2026-09-12，参考 ref/codex：core/src/tools/spec_plan.rs、
// tools/src/tool_search.rs、tools/src/tool_executor.rs 的 ToolExposure）：
//
//	codex 工具面分 Direct（初始暴露）/ Deferred（延迟暴露：注册但不进初始
//	模型可见列表，模型用 tool_search 搜索工具元数据，命中后于下一次模型
//	调用起暴露）。
//
//	本实现同构：
//	  - DeferredToolNames 为「低频按需工具」静态名单——这些工具默认不进
//	    LLM 工具面（Registry.Definitions / EnabledNames 过滤掉，除非本会话
//	    已发现）；工具自身 Enabled 状态不变（前端 /api/tools 可见、可管理，
//	    与 HarnessAlignedToolNames 的「禁用」语义互补）；
//	  - tool_search 工具（宿主注册，会话级 Registry 构建时装配）按关键词
//	    搜索 deferred 池（名/描述/用法/分类加权打分），命中后 MarkToolDiscovered
//	    提升——本会话内后续所有 LLM 调用都可见；
//	  - discovered 为 Registry 级状态（会话隔离：新会话新注册表重新隐藏；
//	    Copy/Subset 快照继承）；
//	  - Execute 对「未发现的 deferred 工具」的直接调用给出引导错误（防绕过）。
//
// ★ 与既有机制的关系：会话白名单收敛（工具集）控制 Enabled；harness 模式
// （WB_HARNESS=1）禁用非保留工具；deferred 只控制「进不进 LLM 工具面」。
// 名单一处维护（本文件），增删零成本。
// ═══════════════════════════════════════════════════════════════

package agent

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// DeferredToolNames 按需工具名单（静态；对齐 codex Deferred 暴露级别）。
//
// 选入标准：低频、按需、可被明确关键词检索（工具名/描述可搜索）——
// 保留 direct 的高频工具不在名单（harness 核心/任务/团队/知识库读写/管理类）。
var DeferredToolNames = map[string]bool{
	// ── 办公文档（tool-office，11）：低频按需 ──
	"csv_read": true, "csv_write": true, "json_to_table": true, "table_stats": true,
	"text_report": true, "word_read": true, "word_write": true, "read_xlsx": true,
	"write_xlsx": true, "read_pdf": true, "markdown_to_html": true,

	// ── 二进制/逆向（tool-binary，7；inspect_binary 保留 direct——read 的
	//    二进制引导直接引用它） ──
	"write_binary": true, "binary_strings": true, "binary_find": true,
	"binary_patch": true, "binary_info": true, "binary_hash": true, "binary_entropy": true,

	// ── 代码图谱低频扩展（tool-codegraph 23；保留 direct：explore/search/
	//    relations；build/stats 维护类动作转按需） ──
	"codegraph_file_structure": true, "codegraph_function": true, "codegraph_class": true,
	"codegraph_git_history": true, "codegraph_get_edit_context": true,
	"codegraph_find_related_tests": true, "codegraph_analyze_complexity": true,
	"codegraph_search_by_pattern": true,
	"codegraph_find_dead_code":    true, "codegraph_module_architecture": true,
	"codegraph_find_hot_paths": true, "codegraph_find_by_imports": true,
	"codegraph_get_detailed_symbol": true, "codegraph_find_dead_imports": true,
	"codegraph_search_by_error": true, "codegraph_index_markdown": true,
	"codegraph_search_docs": true, "codegraph_verify_design": true,
	"codegraph_pr_context": true, "codegraph_find_by_signature": true,
	"codegraph_semantic_search": true,
	"codegraph_build":           true, "codegraph_stats": true,

	// ── 资产/进化（tool-resource 并入后；低频） ──
	"asset_delete": true, "evolution_save_capsule": true,
	"evolution_save_gene": true, "evolution_status": true,

	// ── 资源/验证（低频维护动作） ──
	"resource_list": true, "resource_search": true, "resource_stats": true,
	"memory_verify": true, "project_info_verify": true,

	// ── 快照（tool-harness 并入后；回滚时按需） ──
	"restore_snapshot": true, "list_snapshots": true,

	// ── 市场 / MCP / 技能管理（配置类低频动作） ──
	"marketplace_search": true, "marketplace_install": true,
	"mcp_list": true, "mcp_add": true, "mcp_remove": true,
	"skill_write": true, "skill_delete": true,

	// ── 会话历史 / 工具元信息（tool-system 注册；低频查询） ──
	"history_search": true, "history_list": true, "history_count": true,
	"tool_stats": true, "progress_checker": true,

	// ── BUG 分析（tool-bug；专项排查场景按需） ──
	"bug_detect": true, "bug_fix": true,

	// ── 工具集管理（toolset_* 内核注册；配置类低频动作） ──
	"toolset_build": true, "toolset_edit": true, "toolset_export": true,
	"toolset_import": true, "toolset_list": true, "toolset_remove": true,
	"toolset_show": true,

	// ── 动态插件（cordis 单工具 op 分派，内核注册；插件编写/检视时按需） ──
	"cordis": true,
}

// isDeferredToolName 查询工具是否属于按需（延迟暴露）名单。
func isDeferredToolName(name string) bool { return DeferredToolNames[name] }

// registryCtxKey Registry 注入 Context 的私有键（Execute 时注入当前会话
// 注册表，供 tool_search 等需要访问注册表状态的内置工具使用）。
type registryCtxKey struct{}

// withRegistry 把注册表注入 Context（nil ctx 兜底为 Background——部分调用方
// 传 nil ctx，context.WithValue 在 nil parent 上 panic）。
func withRegistry(ctx context.Context, r *Registry) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, registryCtxKey{}, r)
}

// registryFromContext 从 Context 取注册表（不存在返回 nil）。
func registryFromContext(ctx context.Context) *Registry {
	if r, ok := ctx.Value(registryCtxKey{}).(*Registry); ok {
		return r
	}
	return nil
}

// ─── tool_search 工具 ─────────────────────────────────────────

// toolSearchTokenSplit 查询分词（空格/中英文标点分隔；长度 <2 的片段丢弃）。
func toolSearchTokenSplit(query string) []string {
	fields := strings.FieldsFunc(strings.ToLower(query), func(r rune) bool {
		switch r {
		case ' ', '\t', '\n', '\r', ',', ';', '，', '；', '。', '、', '/', '|', ':', '：':
			return true
		}
		return false
	})
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		if len([]rune(f)) >= 2 {
			out = append(out, f)
		}
	}
	return out
}

// scoreDeferred 计算单个候选工具对查询的匹配分（简化 BM25 风格加权）。
// 权重：name 12 / name(空格化) 10 / description 5 / category 4 / usageGuide 3；
// 完整查询串子串命中（长查询、中文短语）：name +15 / desc/guide +8。
func scoreDeferred(t *Tool, tokens []string, fullQuery string) int {
	name := strings.ToLower(t.Name)
	nameSpaced := strings.ToLower(strings.ReplaceAll(t.Name, "_", " "))
	desc := strings.ToLower(t.Description)
	guide := strings.ToLower(t.UsageGuide)
	cat := strings.ToLower(t.Category)
	score := 0
	for _, tok := range tokens {
		if strings.Contains(name, tok) {
			score += 12
		} else if strings.Contains(nameSpaced, tok) {
			score += 10
		}
		if strings.Contains(desc, tok) {
			score += 5
		}
		if strings.Contains(cat, tok) {
			score += 4
		}
		if strings.Contains(guide, tok) {
			score += 3
		}
	}
	if len([]rune(fullQuery)) >= 2 {
		if strings.Contains(name, fullQuery) || strings.Contains(nameSpaced, fullQuery) {
			score += 15
		}
		if strings.Contains(desc, fullQuery) || strings.Contains(guide, fullQuery) {
			score += 8
		}
	}
	return score
}

// searchDeferredTools 在会话注册表的 deferred 池中搜索（按分数排序，返回
// 命中工具；limit <= 0 用默认 8，上限 20）。不修改状态——提升由调用方做。
func searchDeferredTools(r *Registry, query string, limit int) []*Tool {
	if limit <= 0 {
		limit = 8
	}
	if limit > 20 {
		limit = 20
	}
	fullQuery := strings.ToLower(strings.TrimSpace(query))
	tokens := toolSearchTokenSplit(query)
	if fullQuery == "" {
		return nil
	}
	r.mu.RLock()
	cands := make([]*Tool, 0, len(r.order))
	for _, name := range r.order {
		t := r.tools[name]
		if t == nil || !t.Enabled || !isDeferredToolName(name) {
			continue
		}
		cands = append(cands, t)
	}
	r.mu.RUnlock()
	type scored struct {
		t *Tool
		s int
	}
	hits := make([]scored, 0, len(cands))
	for _, t := range cands {
		if s := scoreDeferred(t, tokens, fullQuery); s > 0 {
			hits = append(hits, scored{t, s})
		}
	}
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].s != hits[j].s {
			return hits[i].s > hits[j].s
		}
		return hits[i].t.Name < hits[j].t.Name
	})
	if len(hits) > limit {
		hits = hits[:limit]
	}
	out := make([]*Tool, 0, len(hits))
	for _, h := range hits {
		out = append(out, h.t)
	}
	return out
}

// deferredPoolSummary 未命中时的域概览提示（静态引导，帮助模型换词再搜）。
func deferredPoolSummary(count int) string {
	return fmt.Sprintf("当前会话有 %d 个「按需工具」未出现在默认工具面中，覆盖域：办公文档（csv/xlsx/word/pdf/表格/报告）、"+
		"二进制与逆向（binary_*/write_binary）、代码图谱扩展（codegraph_* 依赖分析/死代码/文档索引/图谱构建）、"+
		"资产与进化（asset_*/evolution_*）、资源与验证（resource_*/memory_verify/project_info_verify）、"+
		"快照（list_snapshots/restore_snapshot）、市场与 MCP、技能管理（skill_write/delete）、"+
		"会话历史（history_*）、工具元信息（tool_stats/progress_checker）、BUG 分析（bug_detect/bug_fix）、"+
		"工具集管理（toolset_*）、动态插件（cordis_*）。"+
		"换更具体的动作词或工具名（如 \"csv\"、\"死代码\"、\"恢复快照\"、\"历史\"、\"工具集\"）再搜。", count)
}

// toolSearchHandler tool_search 执行体：搜索 deferred 池 → 命中则提升
// （本会话可见，下一次 LLM 调用生效）→ 返回命中清单文本。
func toolSearchHandler(ctx context.Context, args map[string]any) (string, error) {
	r := registryFromContext(ctx)
	if r == nil {
		return "", fmt.Errorf("tool_search 需要会话注册表上下文（未注入）")
	}
	query := strings.TrimSpace(argStr(args, "query"))
	if query == "" {
		return "", fmt.Errorf("query 不能为空（描述你需要的工具能力/动作，如 \"csv 表格汇总\"、\"找死代码\"、\"恢复快照\"）")
	}
	limit := argInt(args, "limit", 0)
	hits := searchDeferredTools(r, query, limit)
	if len(hits) == 0 {
		// 池子规模（提示用）
		pool := 0
		for _, n := range r.Names() {
			if isDeferredToolName(n) {
				if t, ok := r.Get(n); ok && t.Enabled {
					pool++
				}
			}
		}
		return "未找到匹配的按需工具（query=" + query + "）。\n" + deferredPoolSummary(pool), nil
	}
	var b strings.Builder
	fmt.Fprintf(&b, "找到 %d 个按需工具（已在本会话启用，下一轮可直接调用）：\n", len(hits))
	for _, t := range hits {
		r.MarkToolDiscovered(t.Name)
		desc := trimToolDesc(t.Description)
		fmt.Fprintf(&b, "- %s：%s\n", t.Name, desc)
	}
	b.WriteString("（这些工具已加入本会话工具面；如还需要其他能力，可继续搜索。）")
	return b.String(), nil
}

// RegisterToolSearchTool 注册 tool_search 工具（宿主框架工具，会话级
// Registry 构建时经 RegisterHostFrameworkTools 装配）。
// SystemTool=false：对 LLM 可见且前端工具面板可管理（与 update_tasks 等
// 系统桥工具不同——本工具是 agent 的常规能力）。
func RegisterToolSearchTool(r *Registry, root string) {
	if r == nil {
		return
	}
	_ = root
	r.Register(&Tool{
		Name: "tool_search",
		Description: "按需工具搜索：默认工具面只含常用工具（低频工具为「按需」，不预先列出）。" +
			"需要办公文档（csv/xlsx/word/pdf）、二进制逆向、代码图谱扩展分析、资产/进化、资源验证、快照、市场/MCP/技能管理、" +
			"会话历史/BUG 分析/工具集管理/动态插件（cordis）等能力时，" +
			"先用本工具搜索——命中的工具会加入本会话工具面，下一次调用即可直接使用。",
		UsageGuide: "延迟暴露工具发现（对齐 codex tool_search）。query 用具体动作词或工具名（如 \"csv 汇总\"、\"死代码\"、\"恢复快照\"、\"pdf\"、\"历史\"、\"工具集\"）；" +
			"命中后无需再搜——下一轮直接调用命中工具。未命中时会返回可用域概览，换词再搜。",
		Category: "工具",
		ReadOnly: true,
		Parameters: objSchema(props{
			"query": strProp("搜索关键词（动作词/工具名/文件类型，如 \"csv\"、\"xlsx\"、\"死代码\"、\"恢复快照\"）"),
			"limit": intProp("可选：返回条数上限（默认 8，最大 20）"),
		}, "query"),
		Handler: toolSearchHandler,
	})
}
