// Package verify 提供记忆与知识库的自动验证与过期清理。
//
// 记忆和项目知识库随时间推移可能包含对已删除文件/符号的引用，
// 这些过时信息会误导 Agent。本包负责：
//   - 扫描条目中的文件路径引用，检查文件是否仍然存在
//   - 扫描条目中的代码符号引用，检查符号是否仍在 codegraph 中
//   - 标记过期条目并支持自动清理
package verify

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// ── 报告 ──────────────────────────────────────────────

// Report 一次验证的完整报告。
type Report struct {
	CheckedAt   string        `json:"checkedAt"`
	MemoryCount int           `json:"memoryCount"`
	KBCount     int           `json:"kbCount"`
	Stale       []StaleEntry  `json:"stale,omitempty"`
	OKCount     int           `json:"okCount"`
}

// StaleEntry 一条被标记为过时的条目。
type StaleEntry struct {
	Source string   `json:"source"` // "memory" 或 "kb"
	ID     string   `json:"id"`     // 记忆 ID 或知识库路径
	Title  string   `json:"title"`
	Issues []string `json:"issues"` // 过期原因列表
}

// Verifier 验证器，根据上下文检查条目有效性。
type Verifier struct {
	WorkspaceRoots []string // 所有工作区根目录
	CodeGraphCheck func(symbol string) bool // 可选：检查符号是否在 codegraph 中
}

// VerifyAll 对所有记忆和知识库执行验证。
func (v *Verifier) VerifyAll(memories []MemoryEntry, kbEntries []KBEntry) *Report {
	r := &Report{
		CheckedAt: time.Now().Format("2006-01-02 15:04:05"),
	}

	// 验证记忆
	for _, m := range memories {
		issues := v.checkEntry(m.Title, m.Summary, m.KeyPoints)
		if len(issues) > 0 {
			r.Stale = append(r.Stale, StaleEntry{
				Source: "memory",
				ID:     m.ID,
				Title:  m.Title,
				Issues: issues,
			})
		} else {
			r.OKCount++
		}
		r.MemoryCount++
	}

	// 验证知识库
	for _, kb := range kbEntries {
		issues := v.checkEntry(kb.Title, kb.Content, nil)
		if len(issues) > 0 {
			r.Stale = append(r.Stale, StaleEntry{
				Source: "kb",
				ID:     kb.Path,
				Title:  kb.Title,
				Issues: issues,
			})
		} else {
			r.OKCount++
		}
		r.KBCount++
	}

	sort.Slice(r.Stale, func(i, j int) bool {
		if r.Stale[i].Source != r.Stale[j].Source {
			return r.Stale[i].Source < r.Stale[j].Source
		}
		return r.Stale[i].Title < r.Stale[j].Title
	})

	return r
}

// checkEntry 检查单条文本中的引用是否有效。
func (v *Verifier) checkEntry(title, content string, keyPoints []string) []string {
	var issues []string

	text := title + "\n" + content
	for _, kp := range keyPoints {
		text += "\n" + kp
	}

	// ★ 历史记录类条目豁免文件存在性检查：修复记录/排查记录引用的是当时
	//   存在的文件，之后被清理/重构是正常现象，记录本身（决策过程）仍有价值；
	//   误报会刷屏会话上下文（曾有 150+ 条假警告）。仅对「当前指引类」条目
	//   （模块/决策/架构/工作流/重写方案等）做存在性校验——那些才会误导 Agent。
	if isHistoricalRecord(title) {
		return issues
	}

	// 1. 检查 Go 文件路径引用（如 path/file.go, pkg/foo/bar.go）
	fileRefs := extractFileRefs(text)
	for _, ref := range fileRefs {
		if !v.fileExists(ref) {
			issues = append(issues, fmt.Sprintf("引用的文件已不存在: %s", ref))
		}
	}

	// 2. 检查目录引用（如 cmd/companion/ 等常见目录模式）
	dirRefs := extractDirRefs(text)
	for _, ref := range dirRefs {
		if !v.dirExists(ref) {
			issues = append(issues, fmt.Sprintf("引用的目录已不存在: %s", ref))
		}
	}

	// 3. 检查过期时间：超过 60 天未更新的可提醒
	//（不强制标记为过时，但报告会包含）
	if age := extractAge(title, content); age > 60 {
		issues = append(issues, fmt.Sprintf("条目已 %d 天未更新，请注意是否仍然有效", age))
	}

	return issues
}

// isHistoricalRecord 判断标题是否属于历史记录类（修复/排查/改造/决策等过程记录）。
// 这些条目是「发生了什么」的历史事实，引用文件的存在性随时间自然失效，
// 不应视为过期。采用包含匹配（标题可能以文件名前缀开头，如 "agentloop 改造记录"）。
func isHistoricalRecord(title string) bool {
	for _, kw := range []string{
		"修复记录", "修复：", "修复", "排查记录", "历史", "评估报告", "异常报告",
		"性能", "体检", "验证", "冒烟", "升级",
		"改造记录", "实施记录", "重写方案", "决策", "方案", "设计",
	} {
		if strings.Contains(title, kw) {
			return true
		}
	}
	return false
}

// ── 文件引用提取 ──────────────────────────────────────

var (
	// goFilePathRE 匹配常见的 Go 文件路径引用
	goFilePathRE = regexp.MustCompile(`\b` +
		// 匹配如 `pkg/foo/bar.go`、`cmd/main.go`、`internal/agent/tools.go`
		`(?:cmd|internal|pkg|agent|config|scripts|web-ui)[/\w-]+\.(?:go|ts|vue|js|json|yaml|xml|md|css|html)` +
		`\b`)

	// dirPathRE 匹配目录引用（行内的 `cmd/companion/` 模式）
	dirPathRE = regexp.MustCompile(`\b(?:` +
		`cmd/[a-zA-Z0-9_/-]+|` +
		`internal/[a-zA-Z0-9_/-]+|` +
		`pkg/[a-zA-Z0-9_/-]+|` +
		`config/[a-zA-Z0-9_/-]+|` +
		`scripts/[a-zA-Z0-9_/-]+` +
		`)\b`)

	// fullPathRE 匹配含路径分隔符的相对路径（至少一层目录 + 文件扩展名）。
	// ★ 2026-08-15：要求至少含一个 `/`——排除裸文件名（知识库口语化引用
	//   "painter.go"、"host.go" 大量存在，当作路径检查全是误报）；
	//   也排除纯语言名被拆（如 tree-sitter 语言表 "js" 被误拆为 js/.js）。
	//   尾部 \b 防止长扩展名被短扩展名截断（.json 被 js 交替截成 .js）。
	fullPathRE = regexp.MustCompile(`(?:\w+/)[\w./-]*\.(?:go|ts|vue|jsx|json|yaml|xml|markdown|md|css|html|rs|py|java|rb|php|swift|kt|dart|lua|sh|sql|js)\b`)
)

// extractFileRefs 从文本中提取可能的文件路径引用。
func extractFileRefs(text string) []string {
	seen := map[string]bool{}
	var refs []string

	for _, loc := range fullPathRE.FindAllStringIndex(text, -1) {
		m := text[loc[0]:loc[1]]
		// 排除明显不是文件路径的匹配（如版本号 "1.0.7"、URL、import 路径）
		if looksLikeVersion(m) || looksLikeURL(m) || looksLikeImport(m) {
			continue
		}
		// ★ 排除 .pair/ 元数据目录引用：条目常用「{workspace}/.pair/xxx」描述
		//   设计路径（模板而非真实文件），且正则会拆掉前导点（.pair/ → pair/）
		if strings.HasPrefix(m, "pair/") || strings.HasPrefix(m, ".pair/") {
			continue
		}
		// ★ 排除点扩展名序列（语言扩展名表）：`.js/.jsx/.mjs/.ts/.tsx` 会拆出
		//   `js/.jsx/...`——若匹配串前一个字符是 `.`，说明源文本是点扩展名列表
		if loc[0] > 0 && text[loc[0]-1] == '.' {
			continue
		}
		clean := filepath.ToSlash(filepath.Clean(m))
		if !seen[clean] {
			seen[clean] = true
			refs = append(refs, clean)
		}
	}
	return refs
}

// extractDirRefs 从文本中提取目录引用。
func extractDirRefs(text string) []string {
	seen := map[string]bool{}
	var refs []string
	for _, loc := range dirPathRE.FindAllStringIndex(text, -1) {
		m := text[loc[0]:loc[1]]
		// ★ 排除实际是文件路径的匹配：`internal/agent/vision.go` 会被 dirPathRE
		//   拆出 `internal/agent/vision`（.go 不在字符类内）——若目录引用后紧跟
		//   扩展名（.go/.ts/...），说明它是文件路径前缀，跳过
		if loc[1] < len(text) && text[loc[1]] == '.' {
			continue
		}
		clean := filepath.ToSlash(filepath.Clean(m))
		if !seen[clean] {
			seen[clean] = true
			refs = append(refs, clean)
		}
	}
	return refs
}

// ── 存在性检查 ────────────────────────────────────────

func (v *Verifier) fileExists(relPath string) bool {
	for _, root := range v.WorkspaceRoots {
		full := filepath.Join(root, relPath)
		if fi, err := os.Stat(full); err == nil && !fi.IsDir() {
			return true
		}
		// 也尝试相对根目录（WorkspaceRoots[0]）
		if len(v.WorkspaceRoots) > 0 {
			full2 := filepath.Join(v.WorkspaceRoots[0], relPath)
			if fi, err := os.Stat(full2); err == nil && !fi.IsDir() {
				return true
			}
		}
	}
	return false
}

func (v *Verifier) dirExists(relPath string) bool {
	for _, root := range v.WorkspaceRoots {
		full := filepath.Join(root, relPath)
		if fi, err := os.Stat(full); err == nil && fi.IsDir() {
			return true
		}
	}
	return false
}

// ── 辅助判断 ──────────────────────────────────────────

var versionRE = regexp.MustCompile(`^v?\d+\.\d+\.\d+$`)

func looksLikeVersion(s string) bool {
	return versionRE.MatchString(s)
}

func looksLikeURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") || strings.Contains(s, "://")
}

var importPrefixes = []string{
	"github.com/", "golang.org/", "google.golang.org/", "gopkg.in/",
	"pkg.go.dev/", "npmjs.com/", "pypi.org/", "crates.io/",
}

func looksLikeImport(s string) bool {
	for _, p := range importPrefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

// extractAge 尝试从条目内容中提取"最后更新时间"信息，返回距今的天数。
// 如无法提取则返回 0（不过期）。
func extractAge(title, content string) int {
	// 尝试匹配常见的日期格式：2026-07-17, 2026/07/17
	dateRE := regexp.MustCompile(`(\d{4})[-/](\d{1,2})[-/](\d{1,2})`)
	allText := title + "\n" + content
	matches := dateRE.FindAllStringSubmatch(allText, -1)
	if len(matches) == 0 {
		return 0
	}

	// 取最后一个日期（最新的）
	last := matches[len(matches)-1]
	year, month, day := 0, 0, 0
	fmt.Sscanf(last[1], "%d", &year)
	fmt.Sscanf(last[2], "%d", &month)
	fmt.Sscanf(last[3], "%d", &day)
	date := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local)
	return int(time.Since(date).Hours() / 24)
}

// ── 外部类型映射 ──────────────────────────────────────

// MemoryEntry 验证时使用的记忆条目摘要。
type MemoryEntry struct {
	ID        string
	Title     string
	Summary   string
	KeyPoints []string
}

// KBEntry 验证时使用的知识库条目摘要。
type KBEntry struct {
	Path    string
	Title   string
	Content string
}
