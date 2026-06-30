// Package skills 是 Skills 管理（.pair/skills/）。
// 数据层委托 agent/skill_loader（目录式 SKILL.md + frontmatter + 三级渐进披露）。
// UI 面板待后续迁移；当前提供数据层操作供 bridge/agenttools 使用。
//
//go:build windows

package skills

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/hoonfeng/paircode/cmd/companion/agent"
	"github.com/hoonfeng/paircode/cmd/companion/core"
	"github.com/hoonfeng/paircode/cmd/companion/ui/mcp"
)

// ─── 层级（复用 mcp.Level）──

// LevelDef 层级描述。
type LevelDef struct {
	ID   mcp.Level
	Name string
}

// Levels 所有层级（显示顺序）。技能仅项目级（内置 system 级由 skill_loader 管理，UI 面板不展示）。
var Levels = []LevelDef{
	{mcp.LevelProject, "项目级"},
}

// Entry 技能条目（UI 面板/agenttools 兼容用，内部委托 agent.Skill）。
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

// ReadLevel 读某层级的所有技能（委托 agent.LoadAllSkills，按 level 过滤为项目级）。
func ReadLevel(lv mcp.Level) []Entry {
	if lv != mcp.LevelProject {
		return nil
	}
	all := agent.LoadAllSkills()
	var out []Entry
	for _, s := range all {
		if s.Level != agent.LevelProject {
			continue
		}
		out = append(out, Entry{
			Name:        s.Name,
			Description: s.Description,
			Mode:        s.Mode,
			Content:     s.Body,
		})
	}
	return out
}

// Write 写入/更新技能（目录式 .pair/skills/<name>/SKILL.md，含 frontmatter）。
func Write(lv mcp.Level, e Entry) error {
	if lv != mcp.LevelProject {
		return fmt.Errorf("技能仅支持项目级")
	}
	dir := filepath.Join(core.Root(), ".pair", "skills")
	return agent.WriteSkill(dir, agent.Skill{
		Name:        e.Name,
		Description: e.Description,
		Mode:        e.Mode,
		Body:        e.Content,
	})
}

// Delete 删除技能（整个目录）。
func Delete(lv mcp.Level, name string) error {
	if lv != mcp.LevelProject {
		return fmt.Errorf("技能仅支持项目级")
	}
	dir := filepath.Join(core.Root(), ".pair", "skills")
	return agent.DeleteSkill(dir, name)
}

// Prompt 返回所有启用技能的 L1 提示词（含内置 system 级 + 项目级，注入 system prompt）。
func Prompt() string {
	return agent.PromptSkills(agent.LoadAllSkills())
}
