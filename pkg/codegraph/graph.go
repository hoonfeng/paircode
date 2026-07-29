// Package codegraph 实现代码知识图谱，作为 Agent 内置的确定性记忆与推理引擎。
//
// 四层融合知识：
//   (1) 语法结构层 — 文件/包/类/函数/变量结构（Go AST）
//   (2) 依赖调用层 — 导入/调用/继承关系（静态分析）
//   (3) 语义文档层 — 注释/文档/嵌入向量（预留）
//   (4) 演化运维层 — Git历史/变更追溯（预留）
//
// 设计原则：
//   - 图谱是「地图」而非「全书」：只存结构和元信息，源码按需从文件系统读取。
//   - 确定性与 LLM 平衡：语法/依赖用确定性工具，语义标注用 LLM。
//   - 无外部 MCP 依赖：所有组件进程内运行。
//   - 纯 Go 标准库 + 可选轻量依赖。
package codegraph

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// ── 实体类型（节点类型） ──────────────────────────────────

// EntityKind 表示图谱中节点的种类。
type EntityKind string

const (
	// 语法结构层
	EntityProject    EntityKind = "project"     // 项目（根节点）
	EntityModule     EntityKind = "module"      // Go module
	EntityPackage    EntityKind = "package"     // Go package
	EntityFile       EntityKind = "file"        // 源文件
	EntityStruct     EntityKind = "struct"      // 结构体
	EntityInterface  EntityKind = "interface"   // 接口
	EntityFunction   EntityKind = "function"    // 函数
	EntityMethod     EntityKind = "method"      // 方法（结构体/接口的方法）
	EntityType       EntityKind = "type"        // 类型别名/定义
	EntityVariable   EntityKind = "variable"    // 全局变量
	EntityConstant   EntityKind = "constant"    // 常量
	EntityField      EntityKind = "field"       // 结构体字段
	EntityParameter  EntityKind = "parameter"   // 函数参数

	// 依赖调用层
	EntityImport     EntityKind = "import"      // 导入语句
	EntityCallSite   EntityKind = "call_site"   // 调用点

	// 语义文档层（预留）
	EntityComment    EntityKind = "comment"     // 注释块
	EntityDocSection EntityKind = "doc_section" // 文档段落

	// 演化运维层（预留）
	EntityCommit     EntityKind = "commit"      // Git 提交
)

// GraphStore 图谱持久化接口（JSON 和 SQLite 两种实现）。
type GraphStore interface {
	Save(g *Graph) error
	// SaveIncremental 增量保存：只更新 changedFiles 中变更文件的实体和关系，
	// 不涉及的文件保持不动。避免全量 DELETE+INSERT 的性能开销。
	// 如果 changedFiles 为空或 nil，行为同 Save()（全量）。
	SaveIncremental(g *Graph, changedFiles []string) error
	Load() (*Graph, error)
	SaveIndex(index map[string]time.Time) error
	LoadIndex() (map[string]time.Time, error)
	CachedGraph(maxAge int) *Graph
	Exists() bool
	Delete() error
}

// ── 关系类型（边类型） ──────────────────────────────────

// RelationKind 表示图谱中边的种类。
type RelationKind string

const (
	// 包含关系（层次结构）
	RelContains    RelationKind = "contains"    // 父包含子（如包→文件，文件→函数）
	RelBelongsTo   RelationKind = "belongs_to"  // 子隶属于父（反向语义）

	// 定义关系
	RelDefines     RelationKind = "defines"     // 定义（如文件→函数，类型→方法）

	// 依赖/调用关系
	RelCalls       RelationKind = "calls"       // 函数调用
	RelCalledBy    RelationKind = "called_by"   // 被调用（反向）
	RelImports     RelationKind = "imports"     // 文件导入包
	RelImportedBy  RelationKind = "imported_by" // 被导入（反向）
	RelDependsOn   RelationKind = "depends_on"  // 包依赖包

	// 类型关系
	RelInherits    RelationKind = "inherits"    // 继承/实现接口
	RelImplements  RelationKind = "implements"  // 实现接口
	RelEmbeds      RelationKind = "embeds"      // 嵌入类型

	// 语义关系（预留）
	RelDescribes   RelationKind = "describes"   // 文档描述实体

	// 演化关系（预留）
	RelIntroduced  RelationKind = "introduced"  // 提交引入实体
	RelModifiedBy  RelationKind = "modified_by" // 实体被提交修改
)

// ── 实体（节点） ───────────────────────────────────────

// Entity 图谱中的一个节点，表示代码中的某个元素。
type Entity struct {
	ID        string            `json:"id"`        // 全局唯一标识 "pkg/file.go:FuncName"
	Kind      EntityKind        `json:"kind"`      // 实体类型
	Name      string            `json:"name"`      // 实体名称
	FQN       string            `json:"fqn"`       // 完全限定名（如 "github.com/foo/bar.FuncName"）
	FilePath  string            `json:"filePath"`  // 源文件路径（工作区相对路径）
	Line      int               `json:"line"`      // 定义行号（1基）
	EndLine   int               `json:"endLine"`   // 结束行号
	Signature string            `json:"signature"` // 简短签名（如 "func Foo(x int) string"）
	Doc       string            `json:"doc"`       // 文档注释
	Metadata  map[string]string `json:"metadata"`  // 额外元数据（如 receiver type for methods）
}

// EntityID 生成标准化实体标识。
func EntityID(kind EntityKind, pkg, name string) string {
	if pkg != "" {
		return fmt.Sprintf("%s:%s", pkg, name)
	}
	return fmt.Sprintf("%s:%s", kind, name)
}

// ── 关系（边） ─────────────────────────────────────────

// Relation 图谱中的一条边，表示两个实体之间的关系。
type Relation struct {
	ID       string            `json:"id"`       // 唯一标识
	SourceID string            `json:"sourceId"` // 源实体 ID
	TargetID string            `json:"targetId"` // 目标实体 ID
	Kind     RelationKind      `json:"kind"`     // 关系类型
	File     string            `json:"file"`     // 关系所在文件
	Line     int               `json:"line"`     // 关系所在行
	Metadata map[string]string `json:"metadata"` // 额外元数据
}

// RelationID 生成关系标识。
func RelationID(source, target string, kind RelationKind) string {
	return fmt.Sprintf("%s--%s--%s", source, kind, target)
}

// ── 图（Graph） ────────────────────────────────────────

// Graph 内存图结构，支持并发读写。
// 用 map 存储实体和关系，维护邻接表支持高效遍历。
type Graph struct {
	mu sync.RWMutex

	// 基本存储
	entities map[string]*Entity   // ID → Entity
	relations map[string]*Relation // ID → Relation

	// 索引：实体类型索引
	entitiesByKind map[EntityKind]map[string]*Entity

	// 邻接表（前向 + 反向）
	outEdges map[string]map[string][]string // 实体ID → (关系类型 → 目标实体ID列表)
	inEdges  map[string]map[string][]string // 实体ID → (关系类型 → 源实体ID列表)

	// 文件索引：文件路径 → 该文件中所有实体ID
	fileEntities map[string]map[string]bool

	// 名称索引：名称 → 实体ID列表（支持模糊查找）
	nameIndex map[string][]string

	// 统计
	stats GraphStats
}

// GraphStats 图谱统计信息。
type GraphStats struct {
	EntityCount  int            `json:"entityCount"`
	RelationCount int           `json:"relationCount"`
	KindCounts   map[string]int `json:"kindCounts"`   // 各类实体数量
	FileCount    int            `json:"fileCount"`     // 覆盖的文件数
	PackageCount int            `json:"packageCount"`  // 覆盖的包数
}

// NewGraph 创建一个新的空图。
func NewGraph() *Graph {
	return &Graph{
		entities:       make(map[string]*Entity),
		relations:      make(map[string]*Relation),
		entitiesByKind: make(map[EntityKind]map[string]*Entity),
		outEdges:       make(map[string]map[string][]string),
		inEdges:        make(map[string]map[string][]string),
		fileEntities:   make(map[string]map[string]bool),
		nameIndex:      make(map[string][]string),
		stats: GraphStats{
			KindCounts: make(map[string]int),
		},
	}
}

// ── 实体操作 ──────────────────────────────────────────

// AddEntity 添加一个实体到图中。如果 ID 已存在则更新。
func (g *Graph) AddEntity(e *Entity) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.addEntityLocked(e)
}

func (g *Graph) addEntityLocked(e *Entity) {
	if e.ID == "" {
		return
	}
	// 如果已存在，先移除旧的索引
	if old, ok := g.entities[e.ID]; ok {
		g.removeEntityIndexLocked(old)
	}
	g.entities[e.ID] = e

	// 按类型索引
	if g.entitiesByKind[e.Kind] == nil {
		g.entitiesByKind[e.Kind] = make(map[string]*Entity)
	}
	g.entitiesByKind[e.Kind][e.ID] = e

	// 按文件索引
	if e.FilePath != "" {
		if g.fileEntities[e.FilePath] == nil {
			g.fileEntities[e.FilePath] = make(map[string]bool)
		}
		g.fileEntities[e.FilePath][e.ID] = true
	}

	// 按名称索引
	name := strings.ToLower(e.Name)
	g.nameIndex[name] = append(g.nameIndex[name], e.ID)

	// 统计
	g.stats.EntityCount++
	g.stats.KindCounts[string(e.Kind)]++
}

func (g *Graph) removeEntityIndexLocked(e *Entity) {
	delete(g.entities, e.ID)
	if byKind, ok := g.entitiesByKind[e.Kind]; ok {
		delete(byKind, e.ID)
	}
	if e.FilePath != "" {
		if fe, ok := g.fileEntities[e.FilePath]; ok {
			delete(fe, e.ID)
			if len(fe) == 0 {
				delete(g.fileEntities, e.FilePath)
			}
		}
	}
	name := strings.ToLower(e.Name)
	if ids, ok := g.nameIndex[name]; ok {
		newIDs := make([]string, 0, len(ids)-1)
		for _, id := range ids {
			if id != e.ID {
				newIDs = append(newIDs, id)
			}
		}
		if len(newIDs) > 0 {
			g.nameIndex[name] = newIDs
		} else {
			delete(g.nameIndex, name)
		}
	}

	g.stats.EntityCount--
	g.stats.KindCounts[string(e.Kind)]--
}

// GetEntity 按 ID 获取实体。
func (g *Graph) GetEntity(id string) *Entity {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.entities[id]
}

// GetEntitiesByKind 按类型获取所有实体。
func (g *Graph) GetEntitiesByKind(kind EntityKind) []*Entity {
	g.mu.RLock()
	defer g.mu.RUnlock()
	byKind := g.entitiesByKind[kind]
	result := make([]*Entity, 0, len(byKind))
	for _, e := range byKind {
		result = append(result, e)
	}
	return result
}

// GetEntitiesByFile 获取指定文件中的所有实体。
func (g *Graph) GetEntitiesByFile(filePath string) []*Entity {
	g.mu.RLock()
	defer g.mu.RUnlock()
	ids := g.fileEntities[filePath]
	result := make([]*Entity, 0, len(ids))
	for id := range ids {
		if e := g.entities[id]; e != nil {
			result = append(result, e)
		}
	}
	return result
}

// SearchEntities 按名称关键词搜索实体（不区分大小写，子串匹配）。
func (g *Graph) SearchEntities(query string) []*Entity {
	g.mu.RLock()
	defer g.mu.RUnlock()
	q := strings.ToLower(query)
	matched := make(map[string]bool)
	var result []*Entity
	for name, ids := range g.nameIndex {
		if strings.Contains(name, q) {
			for _, id := range ids {
				if !matched[id] {
					matched[id] = true
					if e := g.entities[id]; e != nil {
						result = append(result, e)
					}
				}
			}
		}
	}
	return result
}

// ── 关系操作 ──────────────────────────────────────────

// AddRelation 添加一条关系到图中。如果 ID 已存在则更新。
func (g *Graph) AddRelation(r *Relation) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.addRelationLocked(r)
}

func (g *Graph) addRelationLocked(r *Relation) {
	if r.ID == "" {
		r.ID = RelationID(r.SourceID, r.TargetID, r.Kind)
	}
	g.relations[r.ID] = r

	// 前向边
	if g.outEdges[r.SourceID] == nil {
		g.outEdges[r.SourceID] = make(map[string][]string)
	}
	g.outEdges[r.SourceID][string(r.Kind)] = append(
		g.outEdges[r.SourceID][string(r.Kind)], r.TargetID)

	// 反向边
	if g.inEdges[r.TargetID] == nil {
		g.inEdges[r.TargetID] = make(map[string][]string)
	}
	g.inEdges[r.TargetID][string(r.Kind)] = append(
		g.inEdges[r.TargetID][string(r.Kind)], r.SourceID)

	g.stats.RelationCount++
}

// GetRelations 获取指定实体的所有关系。
// kind 为空则返回所有类型，否则只返回指定类型。
// direction 为 "out"（出边）、"in"（入边）或 "both"（默认）。
func (g *Graph) GetRelations(entityID string, kind RelationKind, direction string) []*Relation {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var result []*Relation
	addFromMap := func(edgeMap map[string]map[string][]string, sourceID string) {
		if edges, ok := edgeMap[sourceID]; ok {
			for rk, targets := range edges {
				if kind != "" && rk != string(kind) {
					continue
				}
				for _, targetID := range targets {
					rid := RelationID(sourceID, targetID, RelationKind(rk))
					if r, ok := g.relations[rid]; ok {
						result = append(result, r)
					}
				}
			}
		}
	}

	switch direction {
	case "out":
		addFromMap(g.outEdges, entityID)
	case "in":
		addFromMap(g.inEdges, entityID)
	default: // "both"
		addFromMap(g.outEdges, entityID)
		addFromMap(g.inEdges, entityID)
	}
	return result
}

// GetSuccessors 获取指定实体的前驱（出边指向的实体），可选按关系类型过滤。
func (g *Graph) GetSuccessors(entityID string, kind RelationKind) []*Entity {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if edges, ok := g.outEdges[entityID]; ok {
		var result []*Entity
		for rk, targets := range edges {
			if kind != "" && rk != string(kind) {
				continue
			}
			for _, targetID := range targets {
				if e := g.entities[targetID]; e != nil {
					result = append(result, e)
				}
			}
		}
		return result
	}
	return nil
}

// GetPredecessors 获取指定实体的后继（入边指向的实体），可选按关系类型过滤。
func (g *Graph) GetPredecessors(entityID string, kind RelationKind) []*Entity {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if edges, ok := g.inEdges[entityID]; ok {
		var result []*Entity
		for rk, sources := range edges {
			if kind != "" && rk != string(kind) {
				continue
			}
			for _, sourceID := range sources {
				if e := g.entities[sourceID]; e != nil {
					result = append(result, e)
				}
			}
		}
		return result
	}
	return nil
}

// ── 图遍历 ────────────────────────────────────────────

// BFS 广度优先遍历，从 startID 开始，maxDepth 为最大深度（0=不限）。
// visit 回调返回 false 停止当前分支探索。
func (g *Graph) BFS(startID string, maxDepth int, visit func(entity *Entity, depth int) bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	type node struct {
		id    string
		depth int
	}

	visited := make(map[string]bool)
	queue := []node{{id: startID, depth: 0}}
	visited[startID] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if maxDepth > 0 && current.depth >= maxDepth {
			continue
		}

		entity := g.entities[current.id]
		if entity == nil {
			continue
		}

		if !visit(entity, current.depth) {
			continue
		}

		if edges, ok := g.outEdges[current.id]; ok {
			for _, targets := range edges {
				for _, targetID := range targets {
					if !visited[targetID] {
						visited[targetID] = true
						queue = append(queue, node{id: targetID, depth: current.depth + 1})
					}
				}
			}
		}
	}
}

// DFS 深度优先遍历。
func (g *Graph) DFS(startID string, maxDepth int, visit func(entity *Entity, depth int) bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	visited := make(map[string]bool)
	g.dfsRecursive(startID, 0, maxDepth, visited, visit)
}

func (g *Graph) dfsRecursive(id string, depth, maxDepth int, visited map[string]bool, visit func(*Entity, int) bool) {
	if visited[id] || (maxDepth > 0 && depth >= maxDepth) {
		return
	}
	visited[id] = true

	entity := g.entities[id]
	if entity == nil {
		return
	}
	if !visit(entity, depth) {
		return
	}

	if edges, ok := g.outEdges[id]; ok {
		for _, targets := range edges {
			for _, targetID := range targets {
				g.dfsRecursive(targetID, depth+1, maxDepth, visited, visit)
			}
		}
	}
}

// ── 影响路径分析 ──────────────────────────────────────

// ImpactPath 一条从源到目标的影响路径。
type ImpactPath struct {
	Path     []string `json:"path"`     // 实体ID链
	Entities []*Entity `json:"-"`       // 对应的实体对象（导出时不序列化）
	Depth    int       `json:"depth"`   // 路径深度
}

// FindImpactPaths 查找从 startID 出发的所有影响路径（DFS 有限深度）。
// 用于回答「修改这个函数会影响哪些地方？」
// maxDepth 限制搜索深度（默认 10）。
func (g *Graph) FindImpactPaths(startID string, maxDepth int) []ImpactPath {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if maxDepth <= 0 {
		maxDepth = 10
	}

	var paths []ImpactPath
	visited := make(map[string]bool)
	currentPath := []string{startID}

	var dfs func(id string, depth int)
	dfs = func(id string, depth int) {
		if depth >= maxDepth {
			return
		}

		edges, ok := g.outEdges[id]
		if !ok {
			// 叶子节点：记录路径
			path := make([]string, len(currentPath))
			copy(path, currentPath)
			entities := make([]*Entity, len(path))
			for i, pid := range path {
				entities[i] = g.entities[pid]
			}
			paths = append(paths, ImpactPath{
				Path:     path,
				Entities: entities,
				Depth:    depth,
			})
			return
		}

		hasSuccessor := false
		for _, targets := range edges {
			for _, targetID := range targets {
				if visited[targetID] {
					continue
				}
				hasSuccessor = true
				visited[targetID] = true
				currentPath = append(currentPath, targetID)
				dfs(targetID, depth+1)
				currentPath = currentPath[:len(currentPath)-1]
				delete(visited, targetID)
			}
		}

		if !hasSuccessor {
			path := make([]string, len(currentPath))
			copy(path, currentPath)
			entities := make([]*Entity, len(path))
			for i, pid := range path {
				entities[i] = g.entities[pid]
			}
			paths = append(paths, ImpactPath{
				Path:     path,
				Entities: entities,
				Depth:    depth,
			})
		}
	}

	visited[startID] = true
	dfs(startID, 0)
	return paths
}

// ── 统计 ──────────────────────────────────────────────

// Stats 返回图统计信息。
func (g *Graph) Stats() GraphStats {
	g.mu.RLock()
	defer g.mu.RUnlock()

	cp := g.stats
	cp.KindCounts = make(map[string]int)
	for k, v := range g.stats.KindCounts {
		cp.KindCounts[k] = v
	}
	cp.FileCount = len(g.fileEntities)

	// 统计包数
	pkgs := make(map[string]bool)
	for _, e := range g.entities {
		if e.Kind == EntityPackage {
			pkgs[e.Name] = true
		}
	}
	cp.PackageCount = len(pkgs)

	return cp
}

// ── JSON 序列化 ──────────────────────────────────────

// Snapshot 图谱快照，用于序列化/反序列化。
type Snapshot struct {
	Entities  []*Entity   `json:"entities"`
	Relations []*Relation `json:"relations"`
	Stats     GraphStats  `json:"stats"`
}

// ToSnapshot 将当前图导出为快照。
func (g *Graph) ToSnapshot() Snapshot {
	g.mu.RLock()
	defer g.mu.RUnlock()

	entities := make([]*Entity, 0, len(g.entities))
	for _, e := range g.entities {
		entities = append(entities, e)
	}
	sort.Slice(entities, func(i, j int) bool {
		return entities[i].ID < entities[j].ID
	})

	relations := make([]*Relation, 0, len(g.relations))
	for _, r := range g.relations {
		relations = append(relations, r)
	}
	sort.Slice(relations, func(i, j int) bool {
		return relations[i].ID < relations[j].ID
	})

	return Snapshot{
		Entities:  entities,
		Relations: relations,
		Stats:     g.stats,
	}
}

// FromSnapshot 从快照恢复图。
func FromSnapshot(sn Snapshot) *Graph {
	g := NewGraph()
	for _, e := range sn.Entities {
		g.addEntityLocked(e)
	}
	for _, r := range sn.Relations {
		g.addRelationLocked(r)
	}
	g.stats = sn.Stats
	g.stats.KindCounts = make(map[string]int)
	for k, v := range sn.Stats.KindCounts {
		g.stats.KindCounts[k] = v
	}
	g.stats.FileCount = len(g.fileEntities)
	return g
}

// Clear 清空图。
func (g *Graph) Clear() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.entities = make(map[string]*Entity)
	g.relations = make(map[string]*Relation)
	g.entitiesByKind = make(map[EntityKind]map[string]*Entity)
	g.outEdges = make(map[string]map[string][]string)
	g.inEdges = make(map[string]map[string][]string)
	g.fileEntities = make(map[string]map[string]bool)
	g.nameIndex = make(map[string][]string)
	g.stats = GraphStats{KindCounts: make(map[string]int)}
}

// RemoveFileEntities 移除指定文件的所有实体和关联的关系。
// 用于增量更新：重新解析文件前先清除旧数据。
func (g *Graph) RemoveFileEntities(filePath string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	ids := g.fileEntities[filePath]
	if len(ids) == 0 {
		return
	}

	// 收集该文件中所有实体的 ID
	entityIDs := make(map[string]bool)
	for id := range ids {
		entityIDs[id] = true
	}

	// 移除涉及这些实体的关系
	for rid, r := range g.relations {
		if entityIDs[r.SourceID] || entityIDs[r.TargetID] {
			delete(g.relations, rid)
			g.stats.RelationCount--
		}
	}

	// 移除实体
	for id := range entityIDs {
		if e, ok := g.entities[id]; ok {
			g.removeEntityIndexLocked(e)
		}
	}

	// 清理邻接表
	for srcID := range entityIDs {
		delete(g.outEdges, srcID)
		delete(g.inEdges, srcID)
	}
	for _, edges := range g.outEdges {
		for rk, targets := range edges {
			newTargets := make([]string, 0, len(targets))
			for _, t := range targets {
				if !entityIDs[t] {
					newTargets = append(newTargets, t)
				}
			}
			edges[rk] = newTargets
		}
	}
	for _, edges := range g.inEdges {
		for rk, sources := range edges {
			newSources := make([]string, 0, len(sources))
			for _, s := range sources {
				if !entityIDs[s] {
					newSources = append(newSources, s)
				}
			}
			edges[rk] = newSources
		}
	}
}
