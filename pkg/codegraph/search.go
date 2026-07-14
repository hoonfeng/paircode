package codegraph

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// ── 搜索引擎 ──────────────────────────────────────────

// SearchEngine 统一搜索引擎，整合图谱搜索与文件搜索。
type SearchEngine struct {
	graph *Graph
	root  string // 工作区根目录
}

// NewSearchEngine 创建基于指定图的搜索引擎。
func NewSearchEngine(g *Graph, root string) *SearchEngine {
	return &SearchEngine{graph: g, root: root}
}

// ── 搜索类型 ──────────────────────────────────────────

// SearchType 搜索类型。
type SearchType string

const (
	SearchExact  SearchType = "exact"   // 精确匹配
	SearchFuzzy  SearchType = "fuzzy"   // 模糊匹配
	SearchPrefix SearchType = "prefix"  // 前缀匹配
)

// SearchScope 搜索范围。
type SearchScope string

const (
	ScopeAll      SearchScope = "all"      // 全部
	ScopeFile     SearchScope = "file"     // 仅文件
	ScopeFunction SearchScope = "function" // 仅函数/方法
	ScopeType     SearchScope = "type"     // 仅类型
	ScopeVariable SearchScope = "variable" // 仅变量
	ScopePackage  SearchScope = "package"  // 仅包
)

// SearchRequest 搜索请求。
type SearchRequest struct {
	Query      string      // 搜索关键词
	Type       SearchType  // 搜索类型（默认 fuzzy）
	Scope      SearchScope // 搜索范围（默认 all）
	MaxResults int         // 最大返回数（默认 20）
	FileFilter string      // 可选：文件路径过滤（glob）
}

// SearchResponse 搜索响应。
type SearchResponse struct {
	Total   int         `json:"total"`   // 总匹配数
	Results []SearchHit `json:"results"` // 匹配结果
}

// SearchHit 一条搜索命中的详细信息。
type SearchHit struct {
	EntityID    string  `json:"entityId"`
	Name        string  `json:"name"`
	Kind        string  `json:"kind"`
	FilePath    string  `json:"filePath"`
	Line        int     `json:"line"`
	Signature   string  `json:"signature,omitempty"`
	FQN         string  `json:"fqn,omitempty"`
	Package     string  `json:"package,omitempty"`
	Relevance   float64 `json:"relevance"`
	MatchReason string  `json:"matchReason,omitempty"`
}

// ── 执行搜索 ──────────────────────────────────────────

// Search 执行一次搜索，返回排序后的结果。
func (se *SearchEngine) Search(req SearchRequest) *SearchResponse {
	if req.MaxResults <= 0 {
		req.MaxResults = 20
	}
	if req.Scope == "" {
		req.Scope = ScopeAll
	}
	if req.Type == "" {
		req.Type = SearchFuzzy
	}

	q := strings.ToLower(strings.TrimSpace(req.Query))
	if q == "" {
		return &SearchResponse{}
	}

	results := se.executeSearch(q, req)

	sort.Slice(results, func(i, j int) bool {
		return results[i].Relevance > results[j].Relevance
	})

	total := len(results)
	if len(results) > req.MaxResults {
		results = results[:req.MaxResults]
	}

	return &SearchResponse{
		Total:   total,
		Results: results,
	}
}

func (se *SearchEngine) executeSearch(query string, req SearchRequest) []SearchHit {
	var hits []SearchHit

	switch req.Scope {
	case ScopeFile:
		for _, e := range se.graph.GetEntitiesByKind(EntityFile) {
			if se.matchEntity(e, query, req.Type) {
				hits = append(hits, se.makeHit(e, query))
			}
		}
	case ScopeFunction:
		for _, e := range se.graph.GetEntitiesByKind(EntityFunction) {
			if se.matchEntity(e, query, req.Type) {
				hits = append(hits, se.makeHit(e, query))
			}
		}
		for _, e := range se.graph.GetEntitiesByKind(EntityMethod) {
			if se.matchEntity(e, query, req.Type) {
				hits = append(hits, se.makeHit(e, query))
			}
		}
	case ScopeType:
		for _, kind := range []EntityKind{EntityStruct, EntityInterface, EntityType} {
			for _, e := range se.graph.GetEntitiesByKind(kind) {
				if se.matchEntity(e, query, req.Type) {
					hits = append(hits, se.makeHit(e, query))
				}
			}
		}
	case ScopeVariable:
		for _, kind := range []EntityKind{EntityVariable, EntityConstant, EntityField} {
			for _, e := range se.graph.GetEntitiesByKind(kind) {
				if se.matchEntity(e, query, req.Type) {
					hits = append(hits, se.makeHit(e, query))
				}
			}
		}
	case ScopePackage:
		for _, e := range se.graph.GetEntitiesByKind(EntityPackage) {
			if se.matchEntity(e, query, req.Type) {
				hits = append(hits, se.makeHit(e, query))
			}
		}
	default: // ScopeAll
		entities := se.graph.SearchEntities(query)
		for _, e := range entities {
			if se.passFileFilter(e, req.FileFilter) {
				hits = append(hits, se.makeHit(e, query))
			}
		}
		// 额外搜索文件
		for _, e := range se.graph.GetEntitiesByKind(EntityFile) {
			if strings.Contains(strings.ToLower(e.Name), query) ||
				strings.Contains(strings.ToLower(e.FilePath), query) {
				if se.passFileFilter(e, req.FileFilter) {
					hits = append(hits, se.makeHit(e, query))
				}
			}
		}
	}

	return hits
}

// ── 匹配逻辑 ──────────────────────────────────────────

func (se *SearchEngine) matchEntity(e *Entity, query string, matchType SearchType) bool {
	switch matchType {
	case SearchExact:
		return strings.EqualFold(e.Name, query) ||
			strings.EqualFold(e.FQN, query) ||
			strings.EqualFold(e.FilePath, query)
	case SearchPrefix:
		return strings.HasPrefix(strings.ToLower(e.Name), query) ||
			strings.HasPrefix(strings.ToLower(e.FQN), query) ||
			strings.HasPrefix(strings.ToLower(e.FilePath), query)
	default:
		return strings.Contains(strings.ToLower(e.Name), query) ||
			strings.Contains(strings.ToLower(e.FQN), query) ||
			strings.Contains(strings.ToLower(e.Signature), query) ||
			strings.Contains(strings.ToLower(e.FilePath), query)
	}
}

func (se *SearchEngine) passFileFilter(e *Entity, filter string) bool {
	if filter == "" {
		return true
	}
	matched, err := filepath.Match(filter, e.FilePath)
	if err != nil {
		return true
	}
	return matched
}

// ── 结果构建 ──────────────────────────────────────────

func (se *SearchEngine) makeHit(e *Entity, query string) SearchHit {
	hit := SearchHit{
		EntityID:  e.ID,
		Name:      e.Name,
		Kind:      string(e.Kind),
		FilePath:  e.FilePath,
		Line:      e.Line,
		Signature: e.Signature,
		FQN:       e.FQN,
		Relevance: calcScore(e, query),
	}

	// 提取包名
	if e.FilePath != "" {
		for _, fe := range se.graph.GetEntitiesByFile(e.FilePath) {
			if fe.Kind == EntityPackage {
				hit.Package = fe.Name
				break
			}
		}
	}

	switch {
	case strings.EqualFold(e.Name, query):
		hit.MatchReason = "名称精确匹配"
	case strings.Contains(strings.ToLower(e.Name), strings.ToLower(query)):
		hit.MatchReason = "名称包含"
	case strings.Contains(strings.ToLower(e.FQN), strings.ToLower(query)):
		hit.MatchReason = "完全限定名匹配"
	case strings.Contains(strings.ToLower(e.Signature), strings.ToLower(query)):
		hit.MatchReason = "签名匹配"
	case strings.Contains(strings.ToLower(e.FilePath), strings.ToLower(query)):
		hit.MatchReason = "文件路径匹配"
	default:
		hit.MatchReason = "模糊匹配"
	}

	return hit
}

// calcScore 计算相关度评分。
func calcScore(e *Entity, query string) float64 {
	q := strings.ToLower(query)
	name := strings.ToLower(e.Name)
	fqn := strings.ToLower(e.FQN)
	sig := strings.ToLower(e.Signature)
	fp := strings.ToLower(e.FilePath)

	score := 0.0
	switch {
	case name == q:
		score = 1.0
	case strings.HasPrefix(name, q):
		score = 0.85
	case strings.Contains(name, q):
		score = 0.7
	case strings.Contains(fqn, q):
		score = 0.5
	case strings.Contains(sig, q):
		score = 0.4
	case strings.Contains(fp, q):
		score = 0.3
	}

	if e.Kind == EntityFunction || e.Kind == EntityMethod {
		score += 0.05
	}
	if score > 1.0 {
		score = 1.0
	}
	return score
}

// ── 格式化输出 ────────────────────────────────────────

// FormatResults 将搜索结果格式化为人类可读文本。
func FormatResults(resp *SearchResponse) string {
	if resp.Total == 0 {
		return "未找到匹配结果。"
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("共找到 %d 个匹配结果（显示前 %d 个）：\n\n",
		resp.Total, len(resp.Results)))

	for i, hit := range resp.Results {
		b.WriteString(fmt.Sprintf("%d. %s", i+1, hit.Name))
		if hit.Package != "" {
			b.WriteString(fmt.Sprintf(" [%s]", hit.Package))
		}
		b.WriteString(fmt.Sprintf(" (%s)", hit.Kind))
		b.WriteString(fmt.Sprintf("\n   文件: %s:%d", hit.FilePath, hit.Line))
		if hit.Signature != "" {
			b.WriteString(fmt.Sprintf("\n   签名: %s", hit.Signature))
		}
		b.WriteString(fmt.Sprintf("\n   匹配: %s (相关度: %.2f)", hit.MatchReason, hit.Relevance))
		b.WriteString("\n")
	}

	return b.String()
}
