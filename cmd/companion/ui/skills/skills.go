// Package skills 是 Skills 管理（.pair/skills/*.md）。
// 提供技能的读取/写入/删除（项目级 .pair/skills/）。
// UI 面板待后续迁移；当前提供数据层操作供 bridge/agenttools 使用。
//
//go:build windows

package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hoonfeng/paircode/cmd/companion/core"
	"github.com/hoonfeng/paircode/cmd/companion/ui/mcp"
)

// ─── 层级（复用 mcp.Level）──

// LevelDef 层级描述。
type LevelDef struct {
	ID   mcp.Level
	Name string
}

// Levels 所有层级（显示顺序）。技能仅项目级。
var Levels = []LevelDef{
	{mcp.LevelProject, "项目级"},
}

// Entry 技能条目。
type Entry struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Mode        string `json:"mode"`
	Content     string `json:"content"`
}

// ModeLabel 返回模式的中文标签。
func ModeLabel(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "auto":
		return "按需"
	case "always":
		return "始终"
	case "manual":
		return "手动"
	}
	return "按需"
}

// skillsDir 技能目录（项目级 .pair/skills/）。
func skillsDir(lv mcp.Level) string {
	if lv == mcp.LevelProject {
		return filepath.Join(core.Root(), ".pair", "skills")
	}
	return ""
}

// ReadLevel 读某层级的所有技能（按文件名排序）。
func ReadLevel(lv mcp.Level) []Entry {
	dir := skillsDir(lv)
	if dir == "" {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []Entry
	for _, ent := range entries {
		if ent.IsDir() || !strings.HasSuffix(ent.Name(), ".md") {
			continue
		}
		name := strings.TrimSuffix(ent.Name(), ".md")
		e := readSkillFile(filepath.Join(dir, ent.Name()), name)
		out = append(out, e)
	}
	return out
}

// readSkillFile 解析技能文件：首行 # 标题 → Name；后续行 → Content（Description 取首段）。
func readSkillFile(path, fallbackName string) Entry {
	data, err := os.ReadFile(path)
	if err != nil {
		return Entry{Name: fallbackName}
	}
	text := string(data)
	e := Entry{Name: fallbackName, Mode: "auto"}
	lines := strings.Split(text, "\n")
	if len(lines) > 0 && strings.HasPrefix(strings.TrimSpace(lines[0]), "#") {
		e.Name = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(lines[0]), "#"))
		lines = lines[1:]
	}
	body := strings.TrimSpace(strings.Join(lines, "\n"))
	e.Content = body
	// Description：取第一段非空文本（最多 120 字）
	for _, ln := range lines {
		if s := strings.TrimSpace(ln); s != "" && !strings.HasPrefix(s, "##") {
			if len([]rune(s)) > 120 {
				s = string([]rune(s)[:120]) + "…"
			}
			e.Description = s
			break
		}
	}
	return e
}

// Write 写入/更新技能（项目级 .pair/skills/<name>.md）。
func Write(lv mcp.Level, e Entry) error {
	dir := skillsDir(lv)
	if dir == "" {
		return fmt.Errorf("技能仅支持项目级")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	var sb strings.Builder
	sb.WriteString("# " + e.Name + "\n\n")
	if e.Description != "" {
		sb.WriteString(e.Description + "\n\n")
	}
	sb.WriteString(e.Content)
	path := filepath.Join(dir, e.Name+".md")
	return os.WriteFile(path, []byte(sb.String()), 0o644)
}

// Delete 删除技能。
func Delete(lv mcp.Level, name string) error {
	dir := skillsDir(lv)
	if dir == "" {
		return fmt.Errorf("技能仅支持项目级")
	}
	path := filepath.Join(dir, name+".md")
	return os.Remove(path)
}

// Prompt 返回可用技能的提示词（注入 Agent 系统提示）。
func Prompt() string {
	var sb strings.Builder
	skills := ReadLevel(mcp.LevelProject)
	if len(skills) == 0 {
		return ""
	}
	sb.WriteString("\n\n# 可用技能（.pair/skills，按需用 skill_read 取全文）\n")
	for _, s := range skills {
		desc := s.Description
		if desc == "" {
			desc = "（无描述）"
		}
		fmt.Fprintf(&sb, "- %s：%s\n", s.Name, desc)
	}
	return sb.String()
}
