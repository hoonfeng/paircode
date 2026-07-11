// Package memory 提供长时记忆索引系统。
// 将已完成对话文档化并索引化，供 Agent 检索历史完成的任务。
// 记忆存储在 .pair/memory_index.json 中。
package memory

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Entry 一条已完成对话的结构化记忆。
type Entry struct {
	ID           string   `json:"id"`                    // 对话 ID
	Title        string   `json:"title"`                 // 对话标题
	Summary      string   `json:"summary"`               // 对话摘要
	CreatedAt    string   `json:"createdAt"`             // 创建时间
	UpdatedAt    string   `json:"updatedAt"`             // 最后消息时间
	MessageCount int      `json:"messageCount"`          // 消息总数
	Tags         []string `json:"tags,omitempty"`        // 自动提取的标签
	KeyPoints    []string `json:"keyPoints,omitempty"`   // 关键结论/成果
	CompletedAt  string   `json:"completedAt,omitempty"` // 完成时间
}

// Index 记忆索引文件结构。
type Index struct {
	Memories []Entry `json:"memories"`
}

var (
	store     Index
	storeMu   sync.Mutex
	storePath string
	storeInit bool
)

// SetRoot 设置工作区根目录，确定 memory_index.json 的存储路径。
// 由 web/GUI 各自在启动时调用。
func SetRoot(root string) {
	if root == "" {
		return
	}
	pairDir := filepath.Join(root, ".pair")
	os.MkdirAll(pairDir, 0755)
	storePath = filepath.Join(pairDir, "memory_index.json")
}

func load() {
	storeMu.Lock()
	defer storeMu.Unlock()
	if storeInit {
		return
	}
	storeInit = true
	store = Index{Memories: make([]Entry, 0)}
	if storePath == "" {
		return
	}
	data, err := os.ReadFile(storePath)
	if err != nil {
		return
	}
	json.Unmarshal(data, &store)
	if store.Memories == nil {
		store.Memories = make([]Entry, 0)
	}
}

func save() {
	if storePath == "" {
		return
	}
	data, _ := json.MarshalIndent(store, "", "  ")
	os.WriteFile(storePath, data, 0644)
}

// Upsert 插入或更新一条记忆条目（按 ID 匹配）。
func Upsert(entry Entry) {
	load()
	storeMu.Lock()
	defer storeMu.Unlock()
	for i, m := range store.Memories {
		if m.ID == entry.ID {
			store.Memories[i] = entry
			save()
			return
		}
	}
	store.Memories = append(store.Memories, entry)
	save()
}

// Delete 删除指定 ID 的记忆条目。
func Delete(id string) {
	load()
	storeMu.Lock()
	defer storeMu.Unlock()
	newMemories := make([]Entry, 0, len(store.Memories))
	for _, m := range store.Memories {
		if m.ID != id {
			newMemories = append(newMemories, m)
		}
	}
	store.Memories = newMemories
	save()
}

// Search 按关键词搜索记忆条目（模糊匹配标题/摘要/标签/关键点）。
// query 为空时返回所有条目。
func Search(query string) []Entry {
	load()
	storeMu.Lock()
	defer storeMu.Unlock()
	if query == "" {
		result := make([]Entry, len(store.Memories))
		copy(result, store.Memories)
		return result
	}
	q := strings.ToLower(query)
	results := make([]Entry, 0)
	for _, m := range store.Memories {
		if strings.Contains(strings.ToLower(m.Title), q) ||
			strings.Contains(strings.ToLower(m.Summary), q) {
			results = append(results, m)
			continue
		}
		for _, tag := range m.Tags {
			if strings.Contains(strings.ToLower(tag), q) {
				results = append(results, m)
				break
			}
		}
		for _, kp := range m.KeyPoints {
			if strings.Contains(strings.ToLower(kp), q) {
				results = append(results, m)
				break
			}
		}
	}
	return results
}

// List 返回所有记忆条目（按更新时间倒序）。
func List() []Entry {
	load()
	storeMu.Lock()
	defer storeMu.Unlock()
	memories := make([]Entry, len(store.Memories))
	copy(memories, store.Memories)
	// 倒序（最新的在前）
	reversed := make([]Entry, len(memories))
	for i, m := range memories {
		reversed[len(memories)-1-i] = m
	}
	return reversed
}

// Count 返回记忆条目总数。
func Count() int {
	load()
	storeMu.Lock()
	defer storeMu.Unlock()
	return len(store.Memories)
}

// ExtractTags 从对话消息文本中提取技术关键词标签。
func ExtractTags(messages []string) []string {
	tagSet := make(map[string]bool)
	techKeywords := map[string]string{
		"go": "Go", "golang": "Go", "rust": "Rust", "python": "Python",
		"javascript": "JavaScript", "typescript": "TypeScript", "vue": "Vue",
		"react": "React", "node": "Node.js", "sql": "SQL", "数据库": "数据库",
		"docker": "Docker", "git": "Git", "api": "API", "http": "HTTP",
		"json": "JSON", "css": "CSS", "html": "HTML", "linux": "Linux",
		"windows": "Windows", "测试": "测试", "部署": "部署", "性能": "性能",
		"安全": "安全", "优化": "优化", "修复": "修复", "重构": "重构",
		"前端": "前端", "后端": "后端", "全栈": "全栈",
	}
	content := strings.Join(messages, " ")
	lower := strings.ToLower(content)
	for kw, tag := range techKeywords {
		if strings.Contains(lower, kw) {
			tagSet[tag] = true
		}
	}
	tags := make([]string, 0, len(tagSet))
	for t := range tagSet {
		if len(tags) >= 8 {
			break
		}
		tags = append(tags, t)
	}
	return tags
}

// ExtractKeyPoints 从助手回复中提取关键结论点。
func ExtractKeyPoints(assistantMessages []string) []string {
	points := make([]string, 0)
	for _, content := range assistantMessages {
		markers := []string{"## 完成", "## 总结", "## 结果", "结论：", "结果：", "完成了"}
		for _, marker := range markers {
			if idx := strings.Index(content, marker); idx >= 0 {
				end := idx + 200
				if end > len(content) {
					end = len(content)
				}
				point := strings.TrimSpace(content[idx:end])
				if dot := strings.Index(point, "\n\n"); dot > 0 && dot < 150 {
					point = point[:dot]
				}
				points = append(points, truncateText(point, 150))
				break
			}
		}
		if len(points) >= 3 {
			break
		}
	}
	return points
}

func truncateText(s string, n int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) <= n {
		return string(r)
	}
	return string(r[:n]) + "…"
}
