package agent

// BES（Bugee Evolution System）—— 经验共享进化机制
// 双层存储：全局经验（跨项目共享）存安装目录 .pair/evolution/global/，
// 项目级经验（仅当前项目）存工作区 .pair/evolution/project/。
// 复刻参考 F:\syproject\伴随式codeagent\src\agent\evolution\

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/hoonfeng/paircode/cmd/companion/core"
)

// ── 类型定义 ───────────────────────────────────────────────

// CapsuleSignal 经验胶囊的触发信号（匹配条件）。
type CapsuleSignal struct {
	ErrorType    string   `json:"error_type"`
	ErrorPattern string   `json:"error_pattern"`
	ToolName     string   `json:"tool_name"`
	ContextTags  []string `json:"context_tags"`
}

// CapsuleSolution 经验胶囊的解决方案。
type CapsuleSolution struct {
	Summary    string   `json:"summary"`
	Steps      []string `json:"steps"`
	KeyChanges string   `json:"key_changes"`
}

// CapsuleValidation 经验胶囊的验证状态。
type CapsuleValidation struct {
	Status       string `json:"status"` // pending / verified / rejected
	SuccessCount int    `json:"success_count"`
}

// Capsule 一条经验胶囊。
type Capsule struct {
	ID         string            `json:"id"`
	Signal     CapsuleSignal     `json:"signal"`
	Solution   CapsuleSolution   `json:"solution"`
	Validation CapsuleValidation `json:"validation"`
	CreatedAt  string            `json:"created_at"`
	Scope      string            `json:"scope"` // "global" | "project"，默认 "global"
}

// Gene 一条技能基因（跨项目复用）。
type Gene struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	Languages   []string `json:"languages"`
	Frameworks  []string `json:"frameworks"`
	Tags        []string `json:"tags"`
	Body        string   `json:"body"`
	Examples    []string `json:"examples"`
	UsageCount  int      `json:"usage_count"`
	Scope       string   `json:"scope"` // "global" | "project"，默认 "global"
}

// EvolutionStatus 进化引擎状态。
type EvolutionStatus struct {
	CapsuleCount int    `json:"capsule_count"`
	GeneCount    int    `json:"gene_count"`
	Fingerprint  string `json:"fingerprint"`
}

// EvolutionEngine 进化引擎。
// 双层存储：全局经验跨项目共享，项目级经验仅当前项目可见。
// 存储统一走 AssetStore：安装目录/.pair/assets/global/{capsules|genes}/ 和 工作区/.pair/assets/project/{capsules|genes}/
type EvolutionEngine struct {
	mu                 sync.RWMutex
	root               string
	store              *AssetStore
	globalCapsulesDir  string // 安装目录 .pair/assets/global/capsules/
	globalGenesDir     string // 安装目录 .pair/assets/global/genes/
	projectCapsulesDir string // 工作区 .pair/assets/project/capsules/
	projectGenesDir    string // 工作区 .pair/assets/project/genes/
}

// NewEvolutionEngine 创建进化引擎。
// 全局经验存 InstallDir()/.pair/assets/global/，项目级经验存 root/.pair/assets/project/。
func NewEvolutionEngine(root string) *EvolutionEngine {
	store := NewAssetStore(root)
	gcd := store.EnsureDir("global", "capsules")
	ggd := store.EnsureDir("global", "genes")
	pcd := store.EnsureDir("project", "capsules")
	pgd := store.EnsureDir("project", "genes")
	return &EvolutionEngine{
		root:               root,
		store:              store,
		globalCapsulesDir:  gcd,
		globalGenesDir:     ggd,
		projectCapsulesDir: pcd,
		projectGenesDir:    pgd,
	}
}

// scopeCapsulesDir 根据作用域返回胶囊存储目录。
func (e *EvolutionEngine) scopeCapsulesDir(scope string) string {
	if scope == "project" {
		return e.projectCapsulesDir
	}
	return e.globalCapsulesDir
}

// scopeGenesDir 根据作用域返回基因存储目录。
func (e *EvolutionEngine) scopeGenesDir(scope string) string {
	if scope == "project" {
		return e.projectGenesDir
	}
	return e.globalGenesDir
}

// capsulePath 返回胶囊的完整文件路径。
func (e *EvolutionEngine) capsulePath(id, scope string) string {
	return filepath.Join(e.scopeCapsulesDir(scope), id+".json")
}

// genePath 返回基因的完整文件路径。
func (e *EvolutionEngine) genePath(id, scope string) string {
	return filepath.Join(e.scopeGenesDir(scope), id+".json")
}

// SaveCapsule 保存经验胶囊。
// scope 为空时默认为 "global"。
func (e *EvolutionEngine) SaveCapsule(capsule *Capsule) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	capsule.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	if capsule.Scope == "" {
		capsule.Scope = "global"
	}
	return e.store.WriteJSON(capsule.Scope, "capsules", capsule.ID, capsule)
}

// SearchCapsules 按条件搜索匹配的经验胶囊（合并全局 + 项目级结果）。
// 自动兼容旧路径（.pair/evolution/）中的胶囊数据。
func (e *EvolutionEngine) SearchCapsules(errorMessage string, toolName string, contextTags []string) []*Capsule {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// 搜索全局 + 项目级胶囊目录（含旧路径兼容）
	dirs := []string{e.globalCapsulesDir, e.projectCapsulesDir}
	// 旧路径兼容：检查 .pair/evolution/ 目录
	oldGlobalDir := filepath.Join(core.InstallDir(), ".pair", "evolution", "global", "capsules")
	oldProjectDir := filepath.Join(e.root, ".pair", "evolution", "project", "capsules")
	for _, oldDir := range []string{oldGlobalDir, oldProjectDir} {
		if oldDir != e.globalCapsulesDir && oldDir != e.projectCapsulesDir {
			if _, err := os.Stat(oldDir); err == nil {
				dirs = append(dirs, oldDir)
			}
		}
	}

	capsules := make([]*Capsule, 0)
	seen := make(map[string]bool) // ID 去重，项目级优先

	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
				continue
			}
			data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
			if err != nil {
				continue
			}
			var c Capsule
			if err := json.Unmarshal(data, &c); err != nil {
				continue
			}

			// 已存在同名胶囊则跳过（项目级覆盖全局）
			if seen[c.ID] {
				continue
			}

			// 匹配规则
			score := 0
			if toolName != "" && strings.Contains(strings.ToLower(c.Signal.ToolName), strings.ToLower(toolName)) {
				score += 3
			}
			if errorMessage != "" {
				if strings.Contains(strings.ToLower(errorMessage), strings.ToLower(c.Signal.ErrorPattern)) {
					score += 2
				}
				if strings.Contains(strings.ToLower(c.Signal.ErrorType), strings.ToLower(errorMessage)) {
					score += 1
				}
			}
			for _, tag := range contextTags {
				for _, ct := range c.Signal.ContextTags {
					if strings.EqualFold(tag, ct) {
						score++
					}
				}
			}

			if score > 0 {
				capsules = append(capsules, &c)
				seen[c.ID] = true
			}
		}
	}

	return capsules
}

// SaveGene 保存技能基因。
// scope 为空时默认为 "global"。
func (e *EvolutionEngine) SaveGene(gene *Gene) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if gene.Scope == "" {
		gene.Scope = "global"
	}
	return e.store.WriteJSON(gene.Scope, "genes", gene.ID, gene)
}

// GetStatus 获取进化引擎状态（合并全局 + 项目级计数，含旧路径兼容）。
func (e *EvolutionEngine) GetStatus() EvolutionStatus {
	e.mu.RLock()
	defer e.mu.RUnlock()

	capsuleCount := 0
	geneCount := 0

	// 新路径
	dirs := []string{e.globalCapsulesDir, e.projectCapsulesDir}
	// 旧路径兼容
	oldGlobalCaps := filepath.Join(core.InstallDir(), ".pair", "evolution", "global", "capsules")
	oldProjectCaps := filepath.Join(e.root, ".pair", "evolution", "project", "capsules")
	if _, err := os.Stat(oldGlobalCaps); err == nil {
		dirs = append(dirs, oldGlobalCaps)
	}
	if _, err := os.Stat(oldProjectCaps); err == nil {
		dirs = append(dirs, oldProjectCaps)
	}

	for _, dir := range dirs {
		if entries, err := os.ReadDir(dir); err == nil {
			for _, entry := range entries {
				if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
					capsuleCount++
				}
			}
		}
	}

	geneDirs := []string{e.globalGenesDir, e.projectGenesDir}
	oldGlobalGenes := filepath.Join(core.InstallDir(), ".pair", "evolution", "global", "genes")
	oldProjectGenes := filepath.Join(e.root, ".pair", "evolution", "project", "genes")
	if _, err := os.Stat(oldGlobalGenes); err == nil {
		geneDirs = append(geneDirs, oldGlobalGenes)
	}
	if _, err := os.Stat(oldProjectGenes); err == nil {
		geneDirs = append(geneDirs, oldProjectGenes)
	}

	for _, dir := range geneDirs {
		if entries, err := os.ReadDir(dir); err == nil {
			for _, entry := range entries {
				if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
					geneCount++
				}
			}
		}
	}

	return EvolutionStatus{
		CapsuleCount: capsuleCount,
		GeneCount:    geneCount,
		Fingerprint:  fmt.Sprintf("%s-%d", e.root, time.Now().Unix()),
	}
}

// SearchGenes 搜索匹配的技能基因（合并全局 + 项目级结果，含旧路径兼容）。
// category 和 query 为空时返回全部基因。
func (e *EvolutionEngine) SearchGenes(query, category string, contextTags []string) []*Gene {
	e.mu.RLock()
	defer e.mu.RUnlock()

	dirs := []string{e.globalGenesDir, e.projectGenesDir}
	// 旧路径兼容
	oldGlobalGenes := filepath.Join(core.InstallDir(), ".pair", "evolution", "global", "genes")
	oldProjectGenes := filepath.Join(e.root, ".pair", "evolution", "project", "genes")
	for _, oldDir := range []string{oldGlobalGenes, oldProjectGenes} {
		if oldDir != e.globalGenesDir && oldDir != e.projectGenesDir {
			if _, err := os.Stat(oldDir); err == nil {
				dirs = append(dirs, oldDir)
			}
		}
	}

	genes := make([]*Gene, 0)
	seen := make(map[string]bool)

	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
				continue
			}
			data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
			if err != nil {
				continue
			}
			var g Gene
			if err := json.Unmarshal(data, &g); err != nil {
				continue
			}
			if seen[g.ID] {
				continue
			}

			// 匹配规则
			if query != "" && !strings.Contains(strings.ToLower(g.Name), strings.ToLower(query)) &&
				!strings.Contains(strings.ToLower(g.Description), strings.ToLower(query)) {
				continue
			}
			if category != "" && !strings.EqualFold(g.Category, category) {
				continue
			}
			if len(contextTags) > 0 {
				matched := false
				for _, tag := range contextTags {
					for _, gt := range g.Tags {
						if strings.EqualFold(tag, gt) {
							matched = true
							break
						}
					}
					if matched {
						break
					}
				}
				if !matched {
					continue
				}
			}

			genes = append(genes, &g)
			seen[g.ID] = true
		}
	}
	return genes
}

// GetProjectFingerprint 返回项目指纹。
func (e *EvolutionEngine) GetProjectFingerprint() string {
	return fmt.Sprintf("%s-%d", e.root, time.Now().Unix())
}

// ── 全局实例 ──────────────────────────────────────────────

var (
	globalEvolutionEngine   *EvolutionEngine
	globalEvolutionEngineMu sync.Mutex
)

// InitEvolutionEngine 初始化全局进化引擎。
func InitEvolutionEngine(root string) *EvolutionEngine {
	globalEvolutionEngineMu.Lock()
	defer globalEvolutionEngineMu.Unlock()
	globalEvolutionEngine = NewEvolutionEngine(root)
	return globalEvolutionEngine
}

// GetEvolutionEngine 获取全局进化引擎。
func GetEvolutionEngine() *EvolutionEngine {
	globalEvolutionEngineMu.Lock()
	defer globalEvolutionEngineMu.Unlock()
	return globalEvolutionEngine
}
