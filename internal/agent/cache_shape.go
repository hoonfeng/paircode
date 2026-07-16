package agent

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"sync/atomic"
)

// ─── 缓存优化（参考 DeepSeek-Reasonix agent/cache_shape.go）──

// PrefixShape hashes the portions of the request prefix that influence
// provider-side prompt-cache reuse. Comparing snapshots across turns
// lets us explain *why* a cache miss happened.
type PrefixShape struct {
	SystemHash string
	ToolsHash  string
	PrefixHash string
}

// CacheDiagnostics reports what changed between two LLM calls' prefixes.
type CacheDiagnostics struct {
	PrefixHash    string   `json:"prefix_hash"`
	PrefixChanged bool     `json:"prefix_changed"`
	ChangeReasons []string `json:"change_reasons,omitempty"`
	SystemHash    string   `json:"system_hash"`
	ToolsHash     string   `json:"tools_hash"`
}

func shortHash(v interface{}) string {
	b, _ := json.Marshal(v)
	h := sha256.Sum256(b)
	return fmt.Sprintf("%x", h[:8])
}

// CaptureShape takes a snapshot of the current prefix state.
func CaptureShape(systemPrompt string, toolDefs []ToolDefinition) PrefixShape {
	normalized := normalizeToolDefs(toolDefs)
	toolsJSON, _ := json.Marshal(normalized)
	return PrefixShape{
		SystemHash: shortHash(systemPrompt),
		ToolsHash:  shortHash(string(toolsJSON)),
		PrefixHash: shortHash(map[string]interface{}{
			"system": systemPrompt,
			"tools":  string(toolsJSON),
		}),
	}
}

func normalizeToolDefs(defs []ToolDefinition) []ToolDefinition {
	out := make([]ToolDefinition, len(defs))
	copy(out, defs)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Function.Name != out[j].Function.Name {
			return out[i].Function.Name < out[j].Function.Name
		}
		return out[i].Function.Description < out[j].Function.Description
	})
	return out
}

// CompareShape returns diagnostics describing what changed between two shapes.
func CompareShape(prev, cur PrefixShape) CacheDiagnostics {
	reasons := []string{}
	if prev.SystemHash != "" && prev.SystemHash != cur.SystemHash {
		reasons = append(reasons, "system")
	}
	if prev.ToolsHash != "" && prev.ToolsHash != cur.ToolsHash {
		reasons = append(reasons, "tools")
	}
	return CacheDiagnostics{
		PrefixHash:    cur.PrefixHash,
		PrefixChanged: len(reasons) > 0,
		ChangeReasons: reasons,
		SystemHash:    cur.SystemHash,
		ToolsHash:     cur.ToolsHash,
	}
}

// ─── 会话级缓存累积 ──

// sessionCache 累积整个会话的缓存命中/未命中 token 数。
// Loop 在每次 LLM 调用后记录 usage，前端可用 Σhit/Σ(hit+miss) 展示聚合命中率。
type sessionCache struct {
	hit  atomic.Int64
	miss atomic.Int64
}

func (sc *sessionCache) record(hit, miss int) {
	if hit > 0 {
		sc.hit.Add(int64(hit))
	}
	if miss > 0 {
		sc.miss.Add(int64(miss))
	}
}

// Snapshot 返回当前会话的累积缓存命中/未命中。
func (sc *sessionCache) Snapshot() (hit, miss int) {
	return int(sc.hit.Load()), int(sc.miss.Load())
}
