package agent

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
)

// ─── 缓存优化（参考 DeepSeek-Reasonix agent/cache_shape.go）──

// PrefixShape hashes the portions of the request prefix that influence
// provider-side prompt-cache reuse. Comparing snapshots across turns
// lets us explain *why* a cache miss happened.
type PrefixShape struct {
	SystemHash   string // 静态前缀（CacheBoundary 之前）——影响 provider 缓存
	DynamicHash  string // 动态后缀（CacheBoundary 之后）——变化不影响前缀缓存
	ToolsHash    string // 归一化（排序后）工具定义哈希
	ToolsRawHash string // 原始顺序工具定义哈希（诊断工具顺序稳定性）
	PrefixHash   string
}

// CacheDiagnostics reports what changed between two LLM calls' prefixes.
type CacheDiagnostics struct {
	PrefixHash     string   `json:"prefix_hash"`
	PrefixChanged  bool     `json:"prefix_changed"`
	DynamicChanged bool     `json:"dynamic_changed"`
	ChangeReasons  []string `json:"change_reasons,omitempty"`
	SystemHash     string   `json:"system_hash"`
	DynamicHash    string   `json:"dynamic_hash"`
	ToolsHash      string   `json:"tools_hash"`
}

func shortHash(v interface{}) string {
	b, _ := json.Marshal(v)
	h := sha256.Sum256(b)
	return fmt.Sprintf("%x", h[:8])
}

// splitAtBoundary 把完整 system prompt 拆成静态前缀（CacheBoundary 之前）与动态后缀（之后）。
// provider 端 prompt-cache 只按静态前缀匹配；动态后缀变化不影响缓存命中。
func splitAtBoundary(p string) (static, dynamic string) {
	if i := strings.Index(p, CacheBoundary); i >= 0 {
		return p[:i], p[i+len(CacheBoundary):]
	}
	return p, ""
}

// CaptureShape takes a snapshot of the current prefix state.
func CaptureShape(systemPrompt string, toolDefs []ToolDefinition) PrefixShape {
	normalized := normalizeToolDefs(toolDefs)
	toolsJSON, _ := json.Marshal(normalized)
	rawJSON, _ := json.Marshal(toolDefs)
	static, dynamic := splitAtBoundary(systemPrompt)
	return PrefixShape{
		SystemHash:   shortHash(static),
		DynamicHash:  shortHash(dynamic),
		ToolsHash:    shortHash(string(toolsJSON)),
		ToolsRawHash: shortHash(string(rawJSON)),
		PrefixHash:   shortHash(map[string]interface{}{
			"system":  static,
			"dynamic": dynamic,
			"tools":   string(toolsJSON),
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
// ★ 只有静态 system 与 tools 变化会导致 provider 缓存断裂（PrefixChanged=true）；
//   动态后缀（boundary 后）变化单独标记 DynamicChanged，不算断裂。
func CompareShape(prev, cur PrefixShape) CacheDiagnostics {
	reasons := []string{}
	if prev.SystemHash != "" && prev.SystemHash != cur.SystemHash {
		reasons = append(reasons, "system")
	}
	if prev.ToolsHash != "" && prev.ToolsHash != cur.ToolsHash {
		reasons = append(reasons, "tools")
	}
	dynChanged := prev.DynamicHash != "" && prev.DynamicHash != cur.DynamicHash
	return CacheDiagnostics{
		PrefixHash:     cur.PrefixHash,
		PrefixChanged:  len(reasons) > 0,
		DynamicChanged: dynChanged,
		ChangeReasons:  reasons,
		SystemHash:     cur.SystemHash,
		DynamicHash:    cur.DynamicHash,
		ToolsHash:      cur.ToolsHash,
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

// ─── system prompt 编译缓存（compileCache）──
// 参考伴随式codeagent PromptCompiler.compileCache + PromptCacheWarmer。
// 用于缓存已编译的 system prompt 字符串，避免每次请求重复拼接。

// promptCompileCache 按 cacheKey 缓存已编译的 system prompt。
// cacheKey 格式: "rootsHash|instructionsHash|philosophyHash"
var (
	promptCompileCacheMu sync.RWMutex
	promptCompileCache   = make(map[string]string)
)

// CacheSystemPrompt 缓存已编译的 system prompt。
// roots: 工作区根目录列表；instructions: 系统指令；philosophy: 哲学指导思想。
// 内部计算哈希后存缓存。
func CacheSystemPrompt(roots []string, instructions, philosophy string, prompt string) string {
	key := compileCacheKey(roots, instructions, philosophy)
	promptCompileCacheMu.Lock()
	promptCompileCache[key] = prompt
	promptCompileCacheMu.Unlock()
	return prompt
}

// GetCachedSystemPrompt 获取缓存的 system prompt。
func GetCachedSystemPrompt(roots []string, instructions, philosophy string) (string, bool) {
	key := compileCacheKey(roots, instructions, philosophy)
	promptCompileCacheMu.RLock()
	p, ok := promptCompileCache[key]
	promptCompileCacheMu.RUnlock()
	return p, ok
}

// ClearPromptCompileCache 清空编译缓存（用于测试）。
func ClearPromptCompileCache() {
	promptCompileCacheMu.Lock()
	promptCompileCache = make(map[string]string)
	promptCompileCacheMu.Unlock()
}

// PromptCompileCacheSize 返回编译缓存条目数。
func PromptCompileCacheSize() int {
	promptCompileCacheMu.RLock()
	n := len(promptCompileCache)
	promptCompileCacheMu.RUnlock()
	return n
}

func compileCacheKey(roots []string, instructions, philosophy string) string {
	h := sha256.New()
	for _, r := range roots {
		h.Write([]byte(r))
		h.Write([]byte{0})
	}
	h.Write([]byte(instructions))
	h.Write([]byte{0})
	h.Write([]byte(philosophy))
	return fmt.Sprintf("%x", h.Sum(nil)[:16])
}

// ─── PromptCacheWarmer system prompt 编译预热器 ──
// 参考伴随式codeagent src/agent/context/cache-warmer.ts
// 在 Agent 启动时预填充 system prompt 编译缓存，避免首次请求时的重复拼接开销。

// PromptCacheWarmer 预热器实例（单例）。
var PromptCacheWarmer = &promptCacheWarmer{}

type promptCacheWarmer struct {
	warmedUp bool
	mu       sync.Mutex
}

// WarmUp 预热编译缓存。传入当前构建 system prompt 的函数。
// 在 Agent 启动时调用一次即可。
func (w *promptCacheWarmer) WarmUp(buildFn func() string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.warmedUp {
		return
	}
	_ = buildFn() // 调用构建函数，触发 CacheSystemPrompt 写缓存
	w.warmedUp = true
}

// IsWarmedUp 返回预热状态。
func (w *promptCacheWarmer) IsWarmedUp() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.warmedUp
}

// Reset 重置预热状态（用于测试）。
func (w *promptCacheWarmer) Reset() {
	w.mu.Lock()
	w.warmedUp = false
	w.mu.Unlock()
}
