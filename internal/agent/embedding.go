package agent

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"
)

// ── EmbeddingBackend 接口 ─────────────────────────

// EmbeddingBackend 嵌入计算后端接口。
// 支持 ONNX Runtime 和空实现（回退关键词搜索）。
type EmbeddingBackend interface {
	// Embed 计算单条文本的嵌入向量。
	Embed(text string) ([]float32, error)
	// Available 后端是否可用。
	Available() bool
	// Dim 返回嵌入向量维度（0 表示未知）。
	Dim() int
}

// noopBackend 空实现——始终返回不可用。
type noopBackend struct{}

func (n *noopBackend) Embed(text string) ([]float32, error) {
	return nil, fmt.Errorf("无可用嵌入后端")
}
func (n *noopBackend) Available() bool { return false }
func (n *noopBackend) Dim() int        { return 0 }

var (
	globalEmbedBackend   EmbeddingBackend
	embedBackendInitOnce sync.Once
)

// GetEmbeddingBackend 获取全局嵌入后端（懒初始化）。
// 优先 ONNX Runtime，失败时返回 noopBackend。
func GetEmbeddingBackend(root string) EmbeddingBackend {
	embedBackendInitOnce.Do(func() {
		// 尝试 ONNX Runtime
		onnx, err := NewONNXBackend(root)
		if err == nil {
			globalEmbedBackend = onnx
			return
		}
		// ONNX 不可用 → noop
		globalEmbedBackend = &noopBackend{}
	})
	return globalEmbedBackend
}

// ── 向量运算 ──────────────────────────────────────

// CosineSimilarity 计算两个向量的余弦相似度。
func CosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		va, vb := float64(a[i]), float64(b[i])
		dot += va * vb
		normA += va * va
		normB += vb * vb
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

// Normalize 原地归一化向量为单位向量。
func Normalize(vec []float32) {
	var norm float64
	for _, v := range vec {
		norm += float64(v) * float64(v)
	}
	if norm == 0 {
		return
	}
	norm = math.Sqrt(norm)
	for i := range vec {
		vec[i] = float32(float64(vec[i]) / norm)
	}
}

// ── 嵌入缓存 ──────────────────────────────────────

// EmbeddingCache 嵌入向量的磁盘缓存（.pair/doc_embeddings.json）。
type EmbeddingCache struct {
	path    string
	mu      sync.RWMutex
	Vectors map[string][]float32 `json:"vectors"`
	Version int                  `json:"version"`
}

// LoadEmbeddingCache 从文件加载嵌入缓存。
func LoadEmbeddingCache(root string) *EmbeddingCache {
	path := filepath.Join(root, ".pair", "doc_embeddings.json")
	cache := &EmbeddingCache{
		path:    path,
		Vectors: make(map[string][]float32),
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return cache
	}
	_ = json.Unmarshal(data, &cache.Vectors)
	return cache
}

// Save 保存嵌入缓存到磁盘。
func (ec *EmbeddingCache) Save() error {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	dir := filepath.Dir(ec.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(ec.Vectors)
	if err != nil {
		return err
	}
	return os.WriteFile(ec.path, data, 0o644)
}

// Get 获取指定键的嵌入向量。
func (ec *EmbeddingCache) Get(key string) []float32 {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	return ec.Vectors[key]
}

// Set 设置指定键的嵌入向量。
func (ec *EmbeddingCache) Set(key string, vec []float32) {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	ec.Vectors[key] = vec
}

// MeanPool 对 BERT 输出的 last_hidden_state 做均值池化。
// hidden: [seqLen, dim] 的二维 flatten 数组。
// attentionMask: [seqLen] 的掩码（1=有效，0=填充）。
// 返回 [dim] 的池化向量。
func MeanPool(hidden []float32, attentionMask []int64, dim int) []float32 {
	seqLen := len(attentionMask)
	if seqLen == 0 || dim == 0 {
		return nil
	}
	result := make([]float32, dim)
	var maskSum float64
	for i := 0; i < seqLen; i++ {
		if attentionMask[i] == 0 {
			continue
		}
		maskSum++
		base := i * dim
		for j := 0; j < dim; j++ {
			result[j] += hidden[base+j]
		}
	}
	if maskSum > 0 {
		for j := 0; j < dim; j++ {
			result[j] = float32(float64(result[j]) / maskSum)
		}
	}
	return result
}
