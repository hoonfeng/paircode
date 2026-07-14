package codegraph

import (
	"fmt"
	"sort"
	"strings"
)

// ── 查询引擎 ──────────────────────────────────────────

// QueryEngine 封装对图谱的结构化查询操作。
// 所有方法以 Graph 实例为操作对象，不修改图。
type QueryEngine struct {
	graph *Graph
}

// NewQueryEngine 创建基于指定图的查询引擎。
func NewQueryEngine(g *Graph) *QueryEngine {
	return &QueryEngine{graph: g}
}

// ── 文件结构 ──────────────────────────────────────────

// FileNode 文件结构树的节点。
type FileNode struct {
	Entity   *Entity    `json:"entity"`
	Children []FileNode `json:"children,omitempty"`
	Depth    int        `json:"depth"`
}

// GetFileStructure 返回指定文件的实体结构树（文件→函数/类型→方法/字段）。
func (qe *QueryEngine) GetFileStructure(filePath string) []FileNode {
	entities := qe.graph.GetEntitiesByFile(filePath)
	if len(entities) == 0 {
		return nil
	}

	// 按行号排序
	sort.Slice(entities, func(i, j int) bool {
		return entities[i].Line < entities[j].Line
	})

	// 构建层次树
	var roots []FileNode
	childMap := make(map[string][]*Entity) // 父实体ID → 子实体列表

	for _, e := range entities {
		// 找出此实体的父实体（包含它的实体）
		parents := qe.graph.GetPredecessors(e.ID, RelContains)
		if len(parents) == 0 {
			// 顶层实体
			roots = append(roots, FileNode{
				Entity: e,
				Depth:  0,
			})
		} else {
			for _, parent := range parents {
				childMap[parent.ID] = append(childMap[parent.ID], e)
			}
		}
	}

	// 递归填充子节点
	var fillChildren func(parents []FileNode) []FileNode
	fillChildren = func(nodes []FileNode) []FileNode {
		for i, node := range nodes {
			if children, ok := childMap[node.Entity.ID]; ok {
				sort.Slice(children, func(i, j int) bool {
					return children[i].Line < children[j].Line
				})
				for _, child := range children {
					node.Children = append(node.Children, FileNode{
						Entity: child,
						Depth:  node.Depth + 1,
					})
				}
				node.Children = fillChildren(node.Children)
				nodes[i] = node
			}
		}
		return nodes
	}

	return fillChildren(roots)
}

// ── 函数定义定位 ──────────────────────────────────────

// FuncLocation 函数定义位置。
type FuncLocation struct {
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	FilePath  string `json:"filePath"`
	Line      int    `json:"line"`
	EndLine   int    `json:"endLine"`
	Signature string `json:"signature"`
	Package   string `json:"package"`
	Receiver  string `json:"receiver,omitempty"`
}

// GetFunctionDefinition 按名称查找函数/方法的定义位置。
// name 可以是函数名、包名.函数名、或接收者.方法名。
func (qe *QueryEngine) GetFunctionDefinition(name string) []FuncLocation {
	entities := qe.graph.SearchEntities(name)
	var results []FuncLocation

	for _, e := range entities {
		if e.Kind != EntityFunction && e.Kind != EntityMethod {
			continue
		}
		// 精确匹配名称
		if !strings.EqualFold(e.Name, name) &&
			!strings.HasSuffix(strings.ToLower(e.FQN), strings.ToLower(name)) {
			continue
		}

		loc := FuncLocation{
			Name:      e.Name,
			Kind:      string(e.Kind),
			FilePath:  e.FilePath,
			Line:      e.Line,
			EndLine:   e.EndLine,
			Signature: e.Signature,
		}
		if e.Metadata != nil {
			loc.Receiver = e.Metadata["receiver"]
		}

		// 提取包名
		if e.FilePath != "" {
			entitiesInFile := qe.graph.GetEntitiesByFile(e.FilePath)
			for _, fe := range entitiesInFile {
				if fe.Kind == EntityPackage {
					loc.Package = fe.Name
					break
				}
			}
		}

		results = append(results, loc)
	}
	return results
}

// ── 类型层次 ──────────────────────────────────────────

// ClassHierarchy 类型层次结构。
type ClassHierarchy struct {
	Type       *Entity          `json:"type"`
	Methods    []FuncLocation   `json:"methods"`
	Fields     []*Entity        `json:"fields"`
	Embedded   []string         `json:"embedded"`   // 嵌入的类型
	Interfaces []string         `json:"interfaces"` // 实现的接口
}

// GetClassHierarchy 返回指定类型（struct/interface）的完整层次结构。
func (qe *QueryEngine) GetClassHierarchy(typeName string) *ClassHierarchy {
	// 搜索类型实体
	entities := qe.graph.SearchEntities(typeName)
	for _, e := range entities {
		if e.Kind != EntityStruct && e.Kind != EntityInterface && e.Kind != EntityType {
			continue
		}
		if !strings.EqualFold(e.Name, typeName) &&
			!strings.EqualFold(e.FQN, typeName) &&
			!strings.HasSuffix(strings.ToLower(e.FQN), strings.ToLower("."+typeName)) {
			continue
		}

		h := &ClassHierarchy{
			Type: e,
		}

		// 该类型包含的字段
		for _, child := range qe.graph.GetSuccessors(e.ID, RelContains) {
			if child.Kind == EntityField {
				h.Fields = append(h.Fields, child)
			}
		}

		// 该类型定义的方法
		for _, child := range qe.graph.GetSuccessors(e.ID, RelDefines) {
			if child.Kind == EntityMethod {
				h.Methods = append(h.Methods, FuncLocation{
					Name:      child.Name,
					Kind:      string(child.Kind),
					FilePath:  child.FilePath,
					Line:      child.Line,
					EndLine:   child.EndLine,
					Signature: child.Signature,
					Receiver:  typeName,
				})
			}
		}

		return h
	}
	return nil
}

// ── 调用者/被调用者 ──────────────────────────────────

// CallInfo 调用关系信息。
type CallInfo struct {
	CallerName  string `json:"callerName"`  // 调用者名称
	CalleeName  string `json:"calleeName"`  // 被调用者名称
	CallerFile  string `json:"callerFile"`  // 调用者文件
	CallerLine  int    `json:"callerLine"`  // 调用所在行
	CallerKind  string `json:"callerKind"`  // 调用者类型
}

// GetCallers 返回调用指定函数的所有调用者。
// funcName 为函数或方法名（支持 FQN 匹配）。
func (qe *QueryEngine) GetCallers(funcName string) []CallInfo {
	var results []CallInfo

	// 找到所有调用该函数的调用点实体
	callSites := qe.graph.SearchEntities("")
	for _, cs := range callSites {
		if cs.Kind != EntityCallSite {
			continue
		}
		if cs.Metadata == nil {
			continue
		}
		callee := cs.Metadata["callee"]
		if !strings.EqualFold(callee, funcName) &&
			!strings.HasSuffix(strings.ToLower(callee), strings.ToLower(funcName)) {
			continue
		}

		// 找到调用该调用点的函数/方法
		callers := qe.graph.GetPredecessors(cs.ID, RelCalls)
		for _, caller := range callers {
			results = append(results, CallInfo{
				CallerName: caller.Name,
				CallerFile: caller.FilePath,
				CallerLine: cs.Line,
				CallerKind: string(caller.Kind),
				CalleeName: callee,
			})
		}
	}

	return results
}

// GetCallees 返回指定函数调用了哪些其他函数。
func (qe *QueryEngine) GetCallees(funcName string) []CallInfo {
	var results []CallInfo

	// 找到函数实体
	entities := qe.graph.SearchEntities(funcName)
	for _, e := range entities {
		if e.Kind != EntityFunction && e.Kind != EntityMethod {
			continue
		}
		if !strings.EqualFold(e.Name, funcName) &&
			!strings.HasSuffix(strings.ToLower(e.FQN), strings.ToLower(funcName)) {
			continue
		}

		// 找出所有从该函数发出的调用
		callSites := qe.graph.GetSuccessors(e.ID, RelCalls)
		for _, cs := range callSites {
			if cs.Kind != EntityCallSite || cs.Metadata == nil {
				continue
			}
			callee := cs.Metadata["callee"]
			results = append(results, CallInfo{
				CallerName: e.Name,
				CallerFile: e.FilePath,
				CallerLine: cs.Line,
				CallerKind: string(e.Kind),
				CalleeName: callee,
			})
		}
	}

	return results
}

// ── 影响分析 ──────────────────────────────────────────

// ImpactResult 影响分析结果。
type ImpactResult struct {
	StartEntity  *Entity       `json:"startEntity"`
	Paths        []ImpactPath  `json:"paths"`
	AffectedFiles []string      `json:"affectedFiles"`
	AffectedFuncs []string      `json:"affectedFuncs"`
	Summary      string         `json:"summary"`
}

// ImpactAnalysis 分析修改某实体后的影响范围。
// entityID 可以是实体 ID 或函数名/文件名。
// maxDepth 搜索最大深度（默认 10）。
func (qe *QueryEngine) ImpactAnalysis(entityID string, maxDepth int) *ImpactResult {
	// 1. 尝试按 ID 查找实体
	entity := qe.graph.GetEntity(entityID)
	if entity == nil {
		// 退而搜索名称
		entities := qe.graph.SearchEntities(entityID)
		if len(entities) > 0 {
			entity = entities[0]
		}
	}
	if entity == nil {
		return &ImpactResult{
			Summary: fmt.Sprintf("未找到实体: %s", entityID),
		}
	}

	// 2. 查找影响路径
	paths := qe.graph.FindImpactPaths(entity.ID, maxDepth)

	// 3. 汇总受影响的文件和函数
	fileSet := make(map[string]bool)
	funcSet := make(map[string]bool)

	for _, path := range paths {
		for _, e := range path.Entities {
			if e == nil {
				continue
			}
			if e.FilePath != "" {
				fileSet[e.FilePath] = true
			}
			if e.Kind == EntityFunction || e.Kind == EntityMethod {
				funcSet[e.FQN] = true
			}
		}
	}
	// 排除自身
	delete(fileSet, entity.FilePath)

	affectedFiles := make([]string, 0, len(fileSet))
	for f := range fileSet {
		affectedFiles = append(affectedFiles, f)
	}
	sort.Strings(affectedFiles)

	affectedFuncs := make([]string, 0, len(funcSet))
	for f := range funcSet {
		affectedFuncs = append(affectedFuncs, f)
	}
	sort.Strings(affectedFuncs)

	// 4. 生成摘要
	summary := fmt.Sprintf("修改 %s (%s:%d) 可能影响 %d 个文件中的 %d 个函数/方法，共 %d 条传播路径",
		entity.Name, entity.FilePath, entity.Line,
		len(affectedFiles), len(affectedFuncs), len(paths))

	return &ImpactResult{
		StartEntity:   entity,
		Paths:         paths,
		AffectedFiles: affectedFiles,
		AffectedFuncs: affectedFuncs,
		Summary:       summary,
	}
}

// ── 代码搜索 ──────────────────────────────────────────

// SearchResult 搜索结果。
type SearchResult struct {
	Entity   *Entity `json:"entity"`
	Relevance float64 `json:"relevance"` // 相关度评分（0-1）
}

// Search 搜索代码实体，支持以下模式：
// - 实体名搜索（函数名/类型名/变量名）
// - 文件路径搜索
// - 符号类型过滤
// - 模糊匹配
func (qe *QueryEngine) Search(query string, kind EntityKind) []SearchResult {
	var results []SearchResult

	if kind != "" {
		// 按类型过滤搜索
		byKind := qe.graph.GetEntitiesByKind(kind)
		for _, e := range byKind {
			if matchesQuery(e, query) {
				results = append(results, SearchResult{
					Entity:    e,
					Relevance: calcRelevance(e, query),
				})
			}
		}
	} else {
		// 全量搜索
		entities := qe.graph.SearchEntities(query)
		for _, e := range entities {
			if matchesQuery(e, query) {
				results = append(results, SearchResult{
					Entity:    e,
					Relevance: calcRelevance(e, query),
				})
			}
		}
	}

	// 按相关度排序
	sort.Slice(results, func(i, j int) bool {
		return results[i].Relevance > results[j].Relevance
	})

	// 限制结果数
	if len(results) > 50 {
		results = results[:50]
	}

	return results
}

// matchesQuery 判断实体是否匹配查询。
func matchesQuery(e *Entity, query string) bool {
	q := strings.ToLower(query)

	// 名称匹配
	if strings.Contains(strings.ToLower(e.Name), q) {
		return true
	}
	// FQN 匹配
	if strings.Contains(strings.ToLower(e.FQN), q) {
		return true
	}
	// 签名匹配
	if strings.Contains(strings.ToLower(e.Signature), q) {
		return true
	}
	// 文件路径匹配
	if strings.Contains(strings.ToLower(e.FilePath), q) {
		return true
	}
	return false
}

// calcRelevance 计算相关度评分（简单规则）。
func calcRelevance(e *Entity, query string) float64 {
	q := strings.ToLower(query)
	name := strings.ToLower(e.Name)
	fqn := strings.ToLower(e.FQN)

	score := 0.0

	// 精确名称匹配最高分
	if name == q {
		score = 1.0
	} else if strings.HasPrefix(name, q) {
		score = 0.9
	} else if strings.Contains(name, q) {
		score = 0.7
	} else if strings.Contains(fqn, q) {
		score = 0.5
	} else if strings.Contains(strings.ToLower(e.FilePath), q) {
		score = 0.3
	}

	// 函数和方法优先于变量
	if e.Kind == EntityFunction || e.Kind == EntityMethod {
		score += 0.05
	}

	if score > 1.0 {
		score = 1.0
	}

	return score
}

// ── 增强查询方法 ──────────────────────────────────────

// FindTypeUsages 查找使用指定类型的所有位置。
func (qe *QueryEngine) FindTypeUsages(typeName string) []SearchHit {
	se := NewSearchEngine(qe.graph, "")
	req := SearchRequest{
		Query:      typeName,
		Scope:      ScopeVariable,
		MaxResults: 100,
	}
	resp := se.Search(req)
	return resp.Results
}

// ── 克隆检测 ──────────────────────────────────────────

// CloneGroup 一组相似的代码实体。
type CloneGroup struct {
	Entities   []*Entity `json:"entities"`
	Similarity float64   `json:"similarity"`
	Pattern    string    `json:"pattern"`
}

// DetectClones 检测结构相似的代码实体（基于签名/结构）。
func (qe *QueryEngine) DetectClones(threshold float64) []CloneGroup {
	entities := qe.graph.GetEntitiesByKind(EntityFunction)
	entities = append(entities, qe.graph.GetEntitiesByKind(EntityMethod)...)

	patternGroups := make(map[string][]*Entity)
	for _, e := range entities {
		pattern := normalizeSignature(e.Signature)
		if pattern != "" {
			patternGroups[pattern] = append(patternGroups[pattern], e)
		}
	}

	var groups []CloneGroup
	for pattern, ents := range patternGroups {
		if len(ents) >= 2 {
			groups = append(groups, CloneGroup{
				Entities:   ents,
				Similarity: calcGroupSimilarity(ents),
				Pattern:    pattern,
			})
		}
	}
	return groups
}

func normalizeSignature(sig string) string {
	sig = strings.TrimSpace(sig)
	if sig == "" {
		return ""
	}
	parts := strings.SplitN(sig, "(", 2)
	if len(parts) < 2 {
		return sig
	}
	return parts[0] + "(...)"
}

func calcGroupSimilarity(ents []*Entity) float64 {
	if len(ents) <= 1 {
		return 1.0
	}
	firstLines := ents[0].EndLine - ents[0].Line
	matchCount := 0
	for _, e := range ents[1:] {
		lines := e.EndLine - e.Line
		if abs(lines-firstLines) <= 3 {
			matchCount++
		}
	}
	if len(ents) <= 1 {
		return 0
	}
	return float64(matchCount) / float64(len(ents)-1)
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// ── 依赖图分析 ──────────────────────────────────────

// DependencyGraph 模块依赖图。
type DependencyGraph struct {
	Nodes []string    `json:"nodes"`
	Edges [][2]string `json:"edges"`
}

// BuildDependencyGraph 从图谱构建模块依赖图。
func (qe *QueryEngine) BuildDependencyGraph() *DependencyGraph {
	dg := &DependencyGraph{}
	pkgs := qe.graph.GetEntitiesByKind(EntityPackage)
	pkgSet := make(map[string]bool)
	for _, p := range pkgs {
		if !pkgSet[p.Name] {
			dg.Nodes = append(dg.Nodes, p.Name)
			pkgSet[p.Name] = true
		}
	}
	seen := make(map[string]bool)
	for _, p := range pkgs {
		rels := qe.graph.GetRelations(p.ID, RelDependsOn, "out")
		for _, r := range rels {
			target := qe.graph.GetEntity(r.TargetID)
			if target != nil {
				key := p.Name + "→" + target.Name
				if !seen[key] {
					dg.Edges = append(dg.Edges, [2]string{p.Name, target.Name})
					seen[key] = true
				}
			}
		}
	}
	return dg
}

// CloneReport 生成克隆检测报告。
func CloneReport(groups []CloneGroup) string {
	if len(groups) == 0 {
		return "未检测到代码克隆。"
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("代码克隆检测结果（共 %d 组）：\n\n", len(groups)))
	for i, g := range groups {
		b.WriteString(fmt.Sprintf("组 %d: %s (相似度: %.0f%%)\n", i+1, g.Pattern, g.Similarity*100))
		for _, e := range g.Entities {
			b.WriteString(fmt.Sprintf("  - %s (%s:%d)\n", e.Name, e.FilePath, e.Line))
		}
		b.WriteString("\n")
	}
	return b.String()
}

// DependencyReport 生成依赖图报告。
func DependencyReport(dg *DependencyGraph) string {
	if len(dg.Nodes) == 0 {
		return "依赖图为空。"
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("模块依赖图（%d 个包，%d 条依赖）：\n\n", len(dg.Nodes), len(dg.Edges)))
	for _, e := range dg.Edges {
		b.WriteString(fmt.Sprintf("  %s → %s\n", e[0], e[1]))
	}
	return b.String()
}
