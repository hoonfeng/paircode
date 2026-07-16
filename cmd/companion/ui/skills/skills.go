// Package skills 是 Skills 管理的薄壳代理。
//
// 所有业务逻辑已迁入 agent/skill_loader.go。
// 本文件仅做初始化路径注入 + 旧 API 转发，并保留 ModeLabel 等纯格式化函数。
//
//go:build windows

package skills

import (
	"fmt"
	"strings"

	"github.com/hoonfeng/paircode/cmd/companion/agent"
)

// Level 配置层级（复用 agent 的 MCPLevel 概念，映射到 SkillLevel）。
type Level = agent.MCPLevel

const (
	LevelUser    Level = agent.MCPLevelUser    // 映射到 agent.LevelSystem（内置 config/skills）
	LevelProject Level = agent.MCPLevelProject // 映射到 agent.LevelProject（工作区 .pair/skills）
)

// LevelDef 层级描述。
type LevelDef struct {
	ID   Level
	Name string
}

// Levels 所有层级（显示顺序）。
// Levels 所有层级（显示顺序）。
var Levels = []LevelDef{
	{ID: LevelProject, Name: "工作区级"},
}
// Entry 技能条目（UI 面板兼容）。
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

// InitSkills 注入系统级和项目级技能目录到 agent 全局变量。
// 由 web_server.go 在启动时调用。
func InitSkills(systemDir, projectDir string) {
	agent.SkillSystemDir = systemDir
	agent.SkillProjectDir = projectDir
}

// ReadLevel 读某层级的所有技能（委托 agent.LoadAllSkills，按 level 过滤）。
func ReadLevel(lv Level) []Entry {
	all := agent.LoadAllSkills()
	var out []Entry
	for _, s := range all {
		want := agent.LevelProject
		if lv == LevelUser {
			want = agent.LevelSystem
		}
		if s.Level != want {
			continue
		}
		out = append(out, Entry{
			Name: s.Name, Description: s.Description,
			Mode: s.Mode, Content: s.Body,
		})
	}
	return out
}

// Write 写入/更新技能（目录式 .pair/skills/<name>/SKILL.md）。
// 仅支持工作区级；用户级（系统内置）为只读。
func Write(lv Level, e Entry) error {
	if lv == LevelUser {
		return fmt.Errorf("系统内置技能为只读，不可写入")
	}
	if agent.SkillProjectDir == "" {
		return fmt.Errorf("SkillProjectDir 未设置，请先调用 InitSkills")
	}
	return agent.WriteSkill(agent.SkillProjectDir, agent.Skill{
		Name: e.Name, Description: e.Description,
		Mode: e.Mode, Body: e.Content,
	})
}

// Delete 删除技能。
// 仅支持工作区级；用户级（系统内置）为只读。
func Delete(lv Level, name string) error {
	if lv == LevelUser {
		return fmt.Errorf("系统内置技能为只读，不可删除")
	}
	if agent.SkillProjectDir == "" {
		return fmt.Errorf("SkillProjectDir 未设置，请先调用 InitSkills")
	}
	return agent.DeleteSkill(agent.SkillProjectDir, name)
}

// Prompt 返回所有启用技能的 L1 提示词。
func Prompt() string {
	return agent.PromptSkills(agent.LoadAllSkills())
}
