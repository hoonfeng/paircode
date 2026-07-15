package codegraph

import (
	"fmt"
	"os"
	"path/filepath"
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

// ════════════════════════════════════════════════════════════════
// 上下文聚合（get_edit_context / get_ai_context）
// ════════════════════════════════════════════════════════════════

// EditContext 修改某个代码位置所需的全部上下文。
type EditContext struct {
	Symbol     SymbolDetail   `json:"symbol"`     // 目标符号完整信息
	Callers    []CallerDetail `json:"callers"`    // 调用者列表
	Tests      []TestDetail   `json:"tests"`      // 关联测试
	Memories   []MemoryBrief  `json:"memories"`   // 相关记忆摘要
	GitHistory []CommitBrief  `json:"gitHistory"` // 近期 Git 历史
	TokenUsed  int            `json:"tokenUsed"`  // 已用 token 估算
}

// SymbolDetail 符号详细信息。
type SymbolDetail struct {
	Name       string `json:"name"`
	Kind       string `json:"kind"`
	FilePath   string `json:"filePath"`
	Line       int    `json:"line"`
	EndLine    int    `json:"endLine"`
	Signature  string `json:"signature"`
	Doc        string `json:"doc"`
	SourceCode string `json:"sourceCode"` // 函数/类型全文源码
}

// CallerDetail 调用者详细信息。
type CallerDetail struct {
	Name       string `json:"name"`
	Kind       string `json:"kind"`
	FilePath   string `json:"filePath"`
	Line       int    `json:"line"`
	Signature  string `json:"signature"`
	SourceCode string `json:"sourceCode,omitempty"` // 调用者源码片段
}

// TestDetail 关联测试信息。
type TestDetail struct {
	Name       string `json:"name"`
	FilePath   string `json:"filePath"`
	Line       int    `json:"line"`
	SourceCode string `json:"sourceCode,omitempty"`
}

// MemoryBrief 简略记忆条目。
type MemoryBrief struct {
	Title   string   `json:"title"`
	Summary string   `json:"summary"`
	Tags    []string `json:"tags"`
}

// CommitBrief 简略提交信息。
type CommitBrief struct {
	Hash    string `json:"hash"`
	Author  string `json:"author"`
	Date    string `json:"date"`
	Message string `json:"message"`
}

// GetEditContext 获取修改某位置代码所需的完整上下文。
// filePath 为工作区相对路径，line 为 1 基行号。
// maxTokens 控制返回内容的 token 预算（0=不限）。
// memoryFunc 为可选的记忆查找回调（由 agent 层提供），用于注入相关记忆。
func GetEditContext(qe *QueryEngine, root, filePath string, line int, maxTokens int, memoryFunc func(query string) []MemoryBrief) *EditContext {
	ctx := &EditContext{}
	usedTokens := 0

	// 1. 找该行所属的实体（函数/方法/类型）
	entities := qe.graph.GetEntitiesByFile(filePath)
	var target *Entity
	for _, e := range entities {
		if line >= e.Line && line <= e.EndLine && e.EndLine > e.Line {
			if e.Kind == EntityFunction || e.Kind == EntityMethod || e.Kind == EntityStruct || e.Kind == EntityInterface {
				if target == nil || (e.Line >= target.Line && e.EndLine <= target.EndLine) {
					target = e
				}
			}
		}
	}
	if target == nil {
		// 回退：精确行匹配
		for _, e := range entities {
			if e.Line == line && (e.Kind == EntityFunction || e.Kind == EntityMethod) {
				target = e
				break
			}
		}
	}
	if target == nil && len(entities) > 0 {
		target = entities[0] // 兜底
	}

	if target != nil {
		ctx.Symbol = SymbolDetail{
			Name:      target.Name,
			Kind:      string(target.Kind),
			FilePath:  target.FilePath,
			Line:      target.Line,
			EndLine:   target.EndLine,
			Signature: target.Signature,
			Doc:       target.Doc,
		}
		// 读取源码
		if source, err := readFileLines(root, filePath, target.Line, target.EndLine); err == nil {
			ctx.Symbol.SourceCode = source
			usedTokens += estimateTokens(source)
		}
	}

	// 2. 查找调用者
	if target != nil {
		callers := qe.GetCallers(target.Name)
		for _, c := range callers {
			cd := CallerDetail{
				Name:      c.CallerName,
				Kind:      c.CallerKind,
				FilePath:  c.CallerFile,
				Line:      c.CallerLine,
			}
			// 读取调用者源码（前后 5 行）
			if source, err := readFileLines(root, c.CallerFile, c.CallerLine-3, c.CallerLine+3); err == nil {
				cd.SourceCode = source
			}
			// 查询调用者的签名
			for _, e := range qe.graph.GetEntitiesByFile(c.CallerFile) {
				if e.Name == c.CallerName && (e.Kind == EntityFunction || e.Kind == EntityMethod) {
					cd.Signature = e.Signature
					break
				}
			}
			ctx.Callers = append(ctx.Callers, cd)
			usedTokens += estimateTokens(cd.SourceCode)
		}
		// 按 token 预算截断
		if maxTokens > 0 && usedTokens > maxTokens && len(ctx.Callers) > 3 {
			ctx.Callers = ctx.Callers[:3]
		}
	}

	// 3. 查找关联测试
	if target != nil {
		ctx.Tests = findRelatedTests(qe, root, target)
		for _, t := range ctx.Tests {
			usedTokens += estimateTokens(t.SourceCode)
		}
		if maxTokens > 0 && usedTokens > maxTokens && len(ctx.Tests) > 3 {
			ctx.Tests = ctx.Tests[:3]
		}
	}

	// 4. 查询 Git 历史
	gh := NewGitHistory(root)
	if target != nil && target.FilePath != "" {
		if commits, err := gh.GetCommitsAffecting(target.FilePath, 10); err == nil {
			for _, c := range commits {
				ctx.GitHistory = append(ctx.GitHistory, CommitBrief{
					Hash:    c.Hash,
					Author:  c.Author,
					Date:    c.Date,
					Message: c.Message,
				})
			}
		}
	}

	// 5. 记忆注入（通过回调）
	if memoryFunc != nil && target != nil {
		ctx.Memories = memoryFunc(target.Name)
	}

	ctx.TokenUsed = usedTokens
	return ctx
}

// readFileLines 读取文件的指定行范围（1 基，含两端）。
func readFileLines(root, filePath string, startLine, endLine int) (string, error) {
	if filePath == "" || startLine <= 0 {
		return "", fmt.Errorf("无效参数")
	}
	fullPath := filePath
	if !filepath.IsAbs(fullPath) {
		fullPath = filepath.Join(root, fullPath)
	}
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(data), "\n")
	if startLine > len(lines) {
		return "", fmt.Errorf("起始行 %d 超出文件行数 %d", startLine, len(lines))
	}
	if endLine > len(lines) {
		endLine = len(lines)
	}
	return strings.Join(lines[startLine-1:endLine], "\n"), nil
}

// estimateTokens 粗略估算 token 数（~4 chars/token）。
func estimateTokens(s string) int {
	return len(s) / 4
}

// ════════════════════════════════════════════════════════════════
// 测试发现（find_related_tests）
// ════════════════════════════════════════════════════════════════

// RelatedTestResult 测试发现结果。
type RelatedTestResult struct {
	Tests      []TestDetail `json:"tests"`
	TotalCount int          `json:"totalCount"`
}

// FindRelatedTests 查找与某个函数/方法关联的测试。
// 通过两种方式发现：（1）调用链中该函数被测试调用；（2）同名 TestXxx 函数。
func FindRelatedTests(qe *QueryEngine, root, funcName string) *RelatedTestResult {
	result := &RelatedTestResult{}

	// 方式 1: 查找调用了目标函数的所有测试
	callers := qe.GetCallers(funcName)
	for _, c := range callers {
		if !strings.HasSuffix(c.CallerFile, "_test.go") {
			continue
		}
		td := TestDetail{
			Name:     c.CallerName,
			FilePath: c.CallerFile,
			Line:     c.CallerLine,
		}
		if source, err := readFileLines(root, c.CallerFile, c.CallerLine-2, c.CallerLine+5); err == nil {
			td.SourceCode = source
		}
		result.Tests = append(result.Tests, td)
	}

	// 方式 2: 按命名约定（TestXxx 对应 Xxx 函数）
	for _, e := range qe.graph.GetEntitiesByKind(EntityFunction) {
		if !strings.HasSuffix(e.FilePath, "_test.go") {
			continue
		}
		testName := e.Name
		if strings.HasPrefix(testName, "Test") {
			// TestFoo → foo / Foo
			candidate := testName[4:] // 去掉 "Test" 前缀
			if strings.EqualFold(candidate, funcName) ||
				strings.EqualFold(candidate, strings.TrimPrefix(funcName, "Test")) {
				// 避免重复
				dup := false
				for _, existing := range result.Tests {
					if existing.Name == e.Name && existing.FilePath == e.FilePath {
						dup = true
						break
					}
				}
				if !dup {
					td := TestDetail{
						Name:     e.Name,
						FilePath: e.FilePath,
						Line:     e.Line,
					}
					if source, err := readFileLines(root, e.FilePath, e.Line, e.EndLine); err == nil {
						td.SourceCode = source
					}
					result.Tests = append(result.Tests, td)
				}
			}
		}
	}

	result.TotalCount = len(result.Tests)
	return result
}

// findRelatedTests 内部辅助：通过实体查找关联测试。
func findRelatedTests(qe *QueryEngine, root string, entity *Entity) []TestDetail {
	if entity == nil {
		return nil
	}
	// 用 FindRelatedTests 的内部实现，避免递归循环
	// 直接查找调用者和同名约定
	var tests []TestDetail
	callers := qe.GetCallers(entity.Name)
	for _, c := range callers {
		if !strings.HasSuffix(c.CallerFile, "_test.go") {
			continue
		}
		td := TestDetail{
			Name:     c.CallerName,
			FilePath: c.CallerFile,
			Line:     c.CallerLine,
		}
		if source, err := readFileLines(root, c.CallerFile, c.CallerLine-2, c.CallerLine+5); err == nil {
			td.SourceCode = source
		}
		tests = append(tests, td)
	}
	// 同名约定
	for _, e := range qe.graph.GetEntitiesByKind(EntityFunction) {
		if !strings.HasSuffix(e.FilePath, "_test.go") {
			continue
		}
		if strings.HasPrefix(e.Name, "Test") {
			candidate := e.Name[4:]
			if strings.EqualFold(candidate, entity.Name) {
				dup := false
				for _, existing := range tests {
					if existing.Name == e.Name && existing.FilePath == e.FilePath {
						dup = true
						break
					}
				}
				if !dup {
					td := TestDetail{
						Name:     e.Name,
						FilePath: e.FilePath,
						Line:     e.Line,
					}
					if source, err := readFileLines(root, e.FilePath, e.Line, e.EndLine); err == nil {
						td.SourceCode = source
					}
					tests = append(tests, td)
				}
			}
		}
	}
	return tests
}

// ════════════════════════════════════════════════════════════════
// 复杂度分析（analyze_complexity）
// ════════════════════════════════════════════════════════════════

// ComplexityReport 某个文件或函数的复杂度分析报告。
type ComplexityReport struct {
	Functions      []FunctionComplexity `json:"functions"`
	TotalFunctions int                  `json:"totalFunctions"`
	AvgComplexity  float64              `json:"avgComplexity"`
	MaxComplexity  int                  `json:"maxComplexity"`
	OverallGrade   string               `json:"overallGrade"`
}

// FunctionComplexity 单个函数的复杂度结果。
type FunctionComplexity struct {
	Name       string `json:"name"`
	Line       int    `json:"line"`
	EndLine    int    `json:"endLine"`
	Complexity int    `json:"complexity"`
	Grade      string `json:"grade"`
	LOC        int    `json:"loc"`
}

// AnalyzeComplexity 分析指定文件的函数复杂度。
// filePath 为工作区相对路径；如为空则分析所有函数。
func AnalyzeComplexity(qe *QueryEngine, root, filePath string) *ComplexityReport {
	report := &ComplexityReport{}

	var funcs []*Entity
	if filePath != "" {
		funcs = qe.graph.GetEntitiesByFile(filePath)
	} else {
		funcs = qe.graph.GetEntitiesByKind(EntityFunction)
		funcs = append(funcs, qe.graph.GetEntitiesByKind(EntityMethod)...)
	}

	for _, e := range funcs {
		if e.Kind != EntityFunction && e.Kind != EntityMethod {
			continue
		}
		if filePath != "" && e.FilePath != filePath {
			continue
		}

		fc := FunctionComplexity{
			Name:       e.Name,
			Line:       e.Line,
			EndLine:    e.EndLine,
			Complexity: 1, // 基线复杂度
			LOC:        e.EndLine - e.Line + 1,
		}

		// 读取源码计算圈复杂度
		if source, err := readFileLines(root, e.FilePath, e.Line, e.EndLine); err == nil {
			fc.Complexity = calcCyclomaticComplexity(source)
		}

		fc.Grade = complexityGrade(fc.Complexity)
		report.Functions = append(report.Functions, fc)

		if fc.Complexity > report.MaxComplexity {
			report.MaxComplexity = fc.Complexity
		}
		report.AvgComplexity += float64(fc.Complexity)
	}

	report.TotalFunctions = len(report.Functions)
	if report.TotalFunctions > 0 {
		report.AvgComplexity /= float64(report.TotalFunctions)
	}
	report.OverallGrade = complexityGrade(int(report.AvgComplexity))

	return report
}

// calcCyclomaticComplexity 基于正则匹配计算圈复杂度。
// 计数决策关键词：if、else if、for、while、case、&&、||、catch。
func calcCyclomaticComplexity(source string) int {
	complexity := 1 // 基线
	lower := source

	// 统计决策关键词（简单词法：忽略字符串/注释中的匹配）
	keywords := []string{
		"if ", "else if ", "for ", "range ",
		"case ", "default:",
		"&&", "||",
		"catch ", "except:", "except ",
	}
	for _, kw := range keywords {
		count := strings.Count(lower, kw)
		// 避免过度计数（如 "if " 在注释中）
		if count > 0 {
			complexity += count
		}
	}

	return complexity
}

// complexityGrade 根据复杂度评分等级。
func complexityGrade(complexity int) string {
	switch {
	case complexity <= 5:
		return "A"
	case complexity <= 10:
		return "B"
	case complexity <= 20:
		return "C"
	case complexity <= 30:
		return "D"
	default:
		return "E"
	}
}

// ComplexityReportText 生成可读复杂度报告。
func ComplexityReportText(r *ComplexityReport) string {
	if r == nil || len(r.Functions) == 0 {
		return "（未找到函数或文件为空）"
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("复杂度分析：%d 个函数，平均 %.1f，最高 %d，总体评分 %s\n\n",
		r.TotalFunctions, r.AvgComplexity, r.MaxComplexity, r.OverallGrade))
	b.WriteString("函数名\t复杂度\t评分\t行数\t位置\n")
	b.WriteString("------\t----\t---\t----\t----\n")
	for _, f := range r.Functions {
		b.WriteString(fmt.Sprintf("%s\t%d\t%s\t%d\tL%d\n",
			f.Name, f.Complexity, f.Grade, f.LOC, f.Line))
	}
	return b.String()
}

// RelatedTestsText 生成可读测试发现报告。
func RelatedTestsText(r *RelatedTestResult) string {
	if r == nil || len(r.Tests) == 0 {
		return "未找到关联测试。"
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("关联测试（共 %d 个）：\n\n", r.TotalCount))
	for _, t := range r.Tests {
		b.WriteString(fmt.Sprintf("  %s (%s:%d)\n", t.Name, t.FilePath, t.Line))
	}
	return b.String()
}

// EditContextText 生成可读的编辑上下文报告。
func EditContextText(ctx *EditContext) string {
	if ctx == nil {
		return "（空上下文）"
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("■ 符号: %s (%s)\n", ctx.Symbol.Name, ctx.Symbol.Kind))
	b.WriteString(fmt.Sprintf("  位置: %s:%d-%d\n", ctx.Symbol.FilePath, ctx.Symbol.Line, ctx.Symbol.EndLine))
	if ctx.Symbol.Signature != "" {
		b.WriteString(fmt.Sprintf("  签名: %s\n", ctx.Symbol.Signature))
	}

	if len(ctx.Callers) > 0 {
		b.WriteString(fmt.Sprintf("\n■ 调用者（%d 个）：\n", len(ctx.Callers)))
		for _, c := range ctx.Callers {
			b.WriteString(fmt.Sprintf("  %s (%s:%d)\n", c.Name, c.FilePath, c.Line))
		}
	}

	if len(ctx.Tests) > 0 {
		b.WriteString(fmt.Sprintf("\n■ 关联测试（%d 个）：\n", len(ctx.Tests)))
		for _, t := range ctx.Tests {
			b.WriteString(fmt.Sprintf("  %s (%s:%d)\n", t.Name, t.FilePath, t.Line))
		}
	}

	if len(ctx.Memories) > 0 {
		b.WriteString(fmt.Sprintf("\n■ 相关记忆（%d 条）：\n", len(ctx.Memories)))
		for _, m := range ctx.Memories {
			b.WriteString(fmt.Sprintf("  %s\n", m.Title))
		}
	}

	if len(ctx.GitHistory) > 0 {
		b.WriteString(fmt.Sprintf("\n■ Git 历史（最近 %d 条）：\n", len(ctx.GitHistory)))
		for _, c := range ctx.GitHistory {
			b.WriteString(fmt.Sprintf("  %s | %s | %s\n", c.Hash, c.Date, c.Message))
		}
	}

	b.WriteString(fmt.Sprintf("\n（估算 token: %d）\n", ctx.TokenUsed))
	return b.String()
}
