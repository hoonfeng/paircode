package codegraph

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"syscall"
)

// ── Git 历史集成（演化运维层） ──────────────────────────

// GitHistory 解析 Git 提交历史，将提交与代码实体关联。
// 通过调用 git CLI（与 agent 包中的 git 工具一致）获取提交信息。
type GitHistory struct {
	root string // 工作区根目录（Git 仓库根）
}

// NewGitHistory 创建 Git 历史分析器。
func NewGitHistory(root string) *GitHistory {
	return &GitHistory{root: root}
}

// CommitInfo 一次 Git 提交的结构化信息。
type CommitInfo struct {
	Hash     string   `json:"hash"`     // 提交哈希（短格式）
	Author   string   `json:"author"`   // 作者
	Date     string   `json:"date"`     // 提交日期
	Message  string   `json:"message"`  // 提交消息首行
	FullMsg  string   `json:"fullMsg"`  // 完整提交消息
	Files    []string `json:"files"`    // 变更的文件
	Added    []string `json:"added"`    // 新增文件
	Modified []string `json:"modified"` // 修改文件
	Deleted  []string `json:"deleted"`  // 删除文件
}

// GetRecentCommits 获取最近的 N 次提交。
func (gh *GitHistory) GetRecentCommits(count int) ([]CommitInfo, error) {
	if count <= 0 {
		count = 20
	}

	cmd := exec.Command("git",
		"-C", gh.root,
		"log",
		fmt.Sprintf("-%d", count),
		"--format=%H|%an|%ai|%s|%b",
		"--name-status",
	)
	if runtime.GOOS == "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	}
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git log 失败: %w", err)
	}

	return gh.parseCommits(string(out))
}

// GetCommitsAffecting 获取影响指定文件的所有提交。
func (gh *GitHistory) GetCommitsAffecting(filePath string, maxCount int) ([]CommitInfo, error) {
	if maxCount <= 0 {
		maxCount = 20
	}

	cmd := exec.Command("git",
		"-C", gh.root,
		"log",
		fmt.Sprintf("-%d", maxCount),
		"--format=%H|%an|%ai|%s|%b",
		"--name-status",
		"--", filePath,
	)
	if runtime.GOOS == "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	}
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git log -- %s 失败: %w", filePath, err)
	}

	return gh.parseCommits(string(out))
}

// GetCommitByHash 按哈希获取单个提交详情。
func (gh *GitHistory) GetCommitByHash(hash string) (*CommitInfo, error) {
	cmd := exec.Command("git",
		"-C", gh.root,
		"show",
		"--format=%H|%an|%ai|%s|%b",
		"--name-status",
		hash,
	)
	if runtime.GOOS == "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	}
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git show %s 失败: %w", hash, err)
	}

	commits, err := gh.parseCommits(string(out))
	if err != nil {
		return nil, err
	}
	if len(commits) == 0 {
		return nil, fmt.Errorf("未找到提交: %s", hash)
	}
	return &commits[0], nil
}

// ── 提交解析 ──────────────────────────────────────────

func (gh *GitHistory) parseCommits(output string) ([]CommitInfo, error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var commits []CommitInfo
	var current *CommitInfo

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 检查是否是提交头 (hash|author|date|subject|body)
		parts := strings.SplitN(line, "|", 5)
		if len(parts) >= 4 {
			// 保存上一个提交
			if current != nil {
				commits = append(commits, *current)
			}

			current = &CommitInfo{
				Hash:    shortHash(parts[0]),
				Author:  parts[1],
				Date:    parts[2],
				Message: parts[3],
			}
			if len(parts) >= 5 {
				current.FullMsg = strings.TrimSpace(parts[4])
			}
			continue
		}

		// 文件变更行 (A/M/D  filepath)
		if current != nil && (strings.HasPrefix(line, "A\t") ||
			strings.HasPrefix(line, "M\t") ||
			strings.HasPrefix(line, "D\t")) {
			status := line[0]
			filePath := strings.TrimSpace(line[2:])

			current.Files = append(current.Files, filePath)
			switch status {
			case 'A':
				current.Added = append(current.Added, filePath)
			case 'M':
				current.Modified = append(current.Modified, filePath)
			case 'D':
				current.Deleted = append(current.Deleted, filePath)
			}
		}
	}

	if current != nil {
		commits = append(commits, *current)
	}

	return commits, nil
}

// shortHash 截取短哈希（前 7 位）。
func shortHash(h string) string {
	if len(h) > 7 {
		return h[:7]
	}
	return h
}

// ── 图谱关联 ──────────────────────────────────────────

// BuildEvolutionGraph 将 Git 提交历史关联到图谱中。
// 为每个提交创建实体，并与变更文件中的实体建立关系。
func (gh *GitHistory) BuildEvolutionGraph(g *Graph, maxCommits int) error {
	commits, err := gh.GetRecentCommits(maxCommits)
	if err != nil {
		return err
	}

	for _, c := range commits {
		commitID := EntityID(EntityCommit, "", c.Hash)
		g.AddEntity(&Entity{
			ID:        commitID,
			Kind:      EntityCommit,
			Name:      c.Hash,
			FQN:       c.Hash,
			Signature: c.Message,
			Metadata: map[string]string{
				"author": c.Author,
				"date":   c.Date,
			},
		})

		// 关联到变更的文件实体
		for _, file := range c.Files {
			// 规范化文件路径
			file = filepath.ToSlash(file)
			// 查找该文件中的所有实体
			entities := g.GetEntitiesByFile(file)
			for _, e := range entities {
				g.AddRelation(&Relation{
					SourceID: commitID,
					TargetID: e.ID,
					Kind:     RelModifiedBy,
					Metadata: map[string]string{
						"file": file,
					},
				})
			}
		}
	}

	return nil
}

// FindCommitByEntity 查找影响指定实体的提交。
func (gh *GitHistory) FindCommitByEntity(g *Graph, entityID string) []*Entity {
	commits := g.GetPredecessors(entityID, RelModifiedBy)
	return commits
}

// ── Git Blame ─────────────────────────────────────────

// BlameInfo 逐行 blame 信息。
type BlameInfo struct {
	Line    int    `json:"line"`
	Hash    string `json:"hash"`
	Author  string `json:"author"`
	Date    string `json:"date"`
	Content string `json:"content"`
}

// BlameFile 对指定文件执行 git blame。
func (gh *GitHistory) BlameFile(filePath string) ([]BlameInfo, error) {
	cmd := exec.Command("git",
		"-C", gh.root,
		"blame",
		"--porcelain",
		filePath,
	)
	if runtime.GOOS == "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	}
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git blame %s 失败: %w", filePath, err)
	}

	return gh.parseBlame(string(out))
}

func (gh *GitHistory) parseBlame(output string) ([]BlameInfo, error) {
	// --porcelain 输出格式复杂，简化处理：只提取行号和哈希
	lines := strings.Split(output, "\n")
	var results []BlameInfo
	lineNum := 1

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if line == "" {
			continue
		}

		// porcelain 格式：<hash> <src_line> <dst_line> <lines>
		parts := strings.Fields(line)
		if len(parts) >= 4 && len(parts[0]) == 40 {
			// 这是提交头行
			hash := shortHash(parts[0])
			// 跳过元数据行（以 tab 或特定前缀开头）
			for i+1 < len(lines) {
				next := lines[i+1]
				if next == "" || strings.HasPrefix(next, "\t") {
					break
				}
				i++
			}

			// 内容行（以 tab 开头）
			content := ""
			if i+1 < len(lines) && strings.HasPrefix(lines[i+1], "\t") {
				content = strings.TrimPrefix(lines[i+1], "\t")
				i++
			}

			results = append(results, BlameInfo{
				Line:    lineNum,
				Hash:    hash,
				Content: content,
			})
			lineNum++
		}
	}

	return results, nil
}

// ── 辅助 ──────────────────────────────────────────────

// runGit 执行 git 命令（类似 agent 中的 runGit）。
func runGit(root string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	if runtime.GOOS == "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrStr := strings.TrimSpace(stderr.String())
		if stderrStr != "" {
			return "", fmt.Errorf("git 错误: %s", stderrStr)
		}
		return "", fmt.Errorf("git 执行失败: %w", err)
	}

	return stdout.String(), nil
}

// ── 函数导出 ──────────────────────────────────────────

// GetCommitTimeline 按时间线返回提交历史（已排序，最新的在前）。
func (gh *GitHistory) GetCommitTimeline(maxCommits int) ([]CommitInfo, error) {
	commits, err := gh.GetRecentCommits(maxCommits)
	if err != nil {
		return nil, err
	}

	sort.Slice(commits, func(i, j int) bool {
		return commits[i].Date > commits[j].Date
	})

	return commits, nil
}

// GetEntityHistory 返回指定实体的变更历史。
// entityID 为实体 ID，通过查找关联该实体的提交获取历史。
func (gh *GitHistory) GetEntityHistory(g *Graph, entityID string) ([]CommitInfo, error) {
	entity := g.GetEntity(entityID)
	if entity == nil {
		return nil, fmt.Errorf("未找到实体: %s", entityID)
	}

	// 首先尝试从图中获取关联的提交
	commits := g.GetPredecessors(entityID, RelModifiedBy)
	if len(commits) > 0 {
		var result []CommitInfo
		for _, c := range commits {
			result = append(result, CommitInfo{
				Hash:    c.Name,
				Message: c.Signature,
			})
		}
		return result, nil
	}

	// 回退到 git log 查询文件历史
	if entity.FilePath != "" {
		return gh.GetCommitsAffecting(entity.FilePath, 20)
	}

	return nil, nil
}

// WhenIntroduced 返回指定实体被引入的时间（首次提交）。
func (gh *GitHistory) WhenIntroduced(filePath string) (*CommitInfo, error) {
	// git log --diff-filter=A --name-status --format=... -- <file>
	cmd := exec.Command("git",
		"-C", gh.root,
		"log",
		"--diff-filter=A",
		"--format=%H|%an|%ai|%s",
		"--name-status",
		"--", filePath,
	)
	if runtime.GOOS == "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	}
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("查询首次提交失败: %w", err)
	}

	commits, err := gh.parseCommits(string(out))
	if err != nil {
		return nil, err
	}
	if len(commits) == 0 {
		return nil, fmt.Errorf("未找到 %s 的引入提交", filePath)
	}
	return &commits[len(commits)-1], nil // 最后一条（最旧的）是引入
}
