package codegraph

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ── 存储配置 ──────────────────────────────────────────

// DefaultStoreDir 默认图谱存储目录（相对于工作区 .pair/）。
const DefaultStoreDir = "codegraph"

// Store 管理图谱的持久化存储（JSON 文件）。
// 每份完整快照存储为一个 JSON 文件，支持原子写入。
type Store struct {
	root     string // 工作区根目录
	storeDir string // 存储子目录名
	mu       sync.Mutex

	// 缓存：最近写入的图实例
	cachedGraph *Graph
	cachedAt    time.Time

	// 配置
	AutoSave bool // 每次修改后自动保存
}

// NewStore 创建新的图谱存储，根目录为工作区根。
// storeDir 默认为 "codegraph"（即 .pair/codegraph/）。
func NewStore(root string) *Store {
	return &Store{
		root:     root,
		storeDir: DefaultStoreDir,
		AutoSave: true,
	}
}

// storePath 返回图谱存储的完整目录路径。
func (s *Store) storePath() string {
	return filepath.Join(s.root, ".pair", s.storeDir)
}

// graphFilePath 返回图谱数据文件路径。
func (s *Store) graphFilePath() string {
	return filepath.Join(s.storePath(), "graph.json")
}

// indexFilePath 返回增量索引文件路径。
func (s *Store) indexFilePath() string {
	return filepath.Join(s.storePath(), "index.json")

}

// ensureDir 确保存储目录存在。
func (s *Store) ensureDir() error {
	return os.MkdirAll(s.storePath(), 0755)
}

// ── 保存 ──────────────────────────────────────────────

// Save 将图谱保存到 JSON 文件。原子写入（写临时文件再重命名）。
func (s *Store) Save(g *Graph) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensureDir(); err != nil {
		return fmt.Errorf("创建存储目录失败: %w", err)
	}

	snapshot := g.ToSnapshot()
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化图谱失败: %w", err)
	}

	filePath := s.graphFilePath()
	tmpPath := filePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("写入临时文件失败: %w", err)
	}
	if err := os.Rename(tmpPath, filePath); err != nil {
		return fmt.Errorf("重命名文件失败: %w", err)
	}

	// 更新缓存
	s.cachedGraph = g
	s.cachedAt = time.Now()
	return nil
}

// SaveIndex 保存文件索引（记录每个文件的上次解析时间，用于增量更新）。
func (s *Store) SaveIndex(index map[string]time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensureDir(); err != nil {
		return err
	}

	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.indexFilePath(), data, 0644)
}

// ── 加载 ──────────────────────────────────────────────

// Load 从 JSON 文件加载图谱。如果文件不存在返回空图。
func (s *Store) Load() (*Graph, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	filePath := s.graphFilePath()
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return NewGraph(), nil
		}
		return nil, fmt.Errorf("读取图谱文件失败: %w", err)
	}

	var snapshot Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, fmt.Errorf("解析图谱文件失败: %w", err)
	}

	g := FromSnapshot(snapshot)
	s.cachedGraph = g
	s.cachedAt = time.Now()
	return g, nil
}

// LoadIndex 加载文件索引。
func (s *Store) LoadIndex() (map[string]time.Time, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.indexFilePath())
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]time.Time), nil
		}
		return nil, err
	}
	var index map[string]time.Time
	if err := json.Unmarshal(data, &index); err != nil {
		return make(map[string]time.Time), nil
	}
	return index, nil
}

// ── 缓存管理 ──────────────────────────────────────────

// CachedGraph 返回缓存的图（如果已加载且未过时）。
// maxAge 为最大缓存时长（秒），0 表示不限制。
func (s *Store) CachedGraph(maxAge int) *Graph {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cachedGraph == nil {
		return nil
	}
	if maxAge > 0 && time.Since(s.cachedAt) > time.Duration(maxAge)*time.Second {
		return nil
	}
	return s.cachedGraph
}

// ── 路径与清理 ────────────────────────────────────────

// Exists 检查图谱文件是否存在。
func (s *Store) Exists() bool {
	_, err := os.Stat(s.graphFilePath())
	return err == nil
}

// Delete 删除所有图谱数据。
func (s *Store) Delete() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	os.RemoveAll(s.storePath())
	s.cachedGraph = nil
	return nil
}

// ── 快照管理 ──────────────────────────────────────────

// SaveSnapshot 保存一个带时间戳的命名快照（用于备份/回滚）。
func (s *Store) SaveSnapshot(g *Graph, name string) error {
	if err := s.ensureDir(); err != nil {
		return err
	}

	snapshot := g.ToSnapshot()
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}

	ts := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("snapshot_%s_%s.json", name, ts)
	return os.WriteFile(filepath.Join(s.storePath(), filename), data, 0644)
}

// ListSnapshots 列出所有快照文件。
func (s *Store) ListSnapshots() ([]string, error) {
	if err := s.ensureDir(); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(s.storePath())
	if err != nil {
		return nil, err
	}

	var snapshots []string
	for _, e := range entries {
		if !e.IsDir() && len(e.Name()) > 9 && e.Name()[:9] == "snapshot_" {
			snapshots = append(snapshots, e.Name())
		}
	}
	return snapshots, nil
}
