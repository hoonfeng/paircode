// skill_loader.go Skills 核心加载逻辑（供 UI 和 agent 共用）。
// 三级渐进披露：
//   L1 常驻 system prompt —— frontmatter 的 name+description（PromptSkills）
//   L2 load_skill 工具     —— 按需加载 SKILL.md 正文（Skill.Body）
//   L3 load_skill_resource —— 按需加载 references/assets/scripts 子文件（LoadSkillResource，沙箱校验）
//
// 存储格式：目录式 .pair/skills/<name>/SKILL.md（frontmatter）+ references/ assets/ scripts/。
// 旧扁平 .pair/skills/<name>.md 兼容读取（无 frontmatter，首行 # 标题）。
// 内置技能：config/skills/<name>/SKILL.md（system 级）。

package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ─── 全局配置（由 bridge 初始化，供 agenttools/resource_manager 共用）──

// SkillSystemDir 内置技能目录（config/skills/），由 bridge 启动时设置。
var SkillSystemDir string

// SkillProjectDir 项目级技能目录（.pair/skills/），由 bridge 启动时设置。
var SkillProjectDir string

// SkillEnabled 启用覆盖（键 "level::name" 如 "system::emoji-icons"，值 true=启用）。
// 键不存在时默认启用。由 bridge 从 settings.json 的 SkillEnabledOverrides 注入。
var SkillEnabled map[string]bool

// ─── 类型 ──

// SkillLevel 技能层级。
type SkillLevel string

const (
	LevelSystem  SkillLevel = "system"  // 内置（config/skills/）
	LevelProject SkillLevel = "project" // 用户项目级（.pair/skills/）
)

// Skill 一个技能条目（三级渐进披露）。
type Skill struct {
	Name         string     // frontmatter.name（目录名须一致，不一致用目录名兜底）
	Description  string     // frontmatter.description（L1 展示）
	Mode         string     // frontmatter.activation：auto/always/manual（默认 auto）
	Globs        string     // frontmatter.globs：按文件类型激活（空格分隔，如 "*.css *.tsx"）
	AllowedTools []string   // frontmatter.allowed-tools：工具白名单
	Version      string     // frontmatter.version
	Level        SkillLevel // 层级
	DirPath      string     // 技能目录绝对路径（目录式=技能目录；扁平=skills 目录）
	Source       string     // "dir"（目录式 SKILL.md）/ "flat"（旧扁平 .md）
	Body         string     // 正文（L2，LoadAll 时已填充）
}

// ─── 加载 ──

// LoadAllSkills 扫描 system + project 两级目录，合并并按 SkillEnabled 过滤。
// 用全局 SkillSystemDir/SkillProjectDir/SkillEnabled。
func LoadAllSkills() []Skill {
	return loadAllFrom(SkillSystemDir, SkillProjectDir, SkillEnabled)
}

// loadAllFrom 内部实现（可测试，传参不依赖全局）。
func loadAllFrom(systemDir, projectDir string, enabled map[string]bool) []Skill {
	var all []Skill
	all = append(all, loadSkillsFromDir(systemDir, LevelSystem, enabled)...)
	all = append(all, loadSkillsFromDir(projectDir, LevelProject, enabled)...)
	return all
}

// loadSkillsFromDir 扫描单个层级目录：目录式 SKILL.md 优先，旧扁平 .md 兼容。
func loadSkillsFromDir(dir string, level SkillLevel, enabled map[string]bool) []Skill {
	if dir == "" {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []Skill
	for _, ent := range entries {
		var s Skill
		if ent.IsDir() {
			// 目录式：<name>/SKILL.md
			skillFile := filepath.Join(dir, ent.Name(), "SKILL.md")
			data, err := os.ReadFile(skillFile)
			if err != nil {
				continue
			}
			s = parseSkill(string(data), ent.Name(), level)
			s.DirPath = filepath.Join(dir, ent.Name())
			s.Source = "dir"
		} else if strings.HasSuffix(ent.Name(), ".md") {
			// 旧扁平：<name>.md（无 frontmatter，兼容读）
			skillFile := filepath.Join(dir, ent.Name())
			data, err := os.ReadFile(skillFile)
			if err != nil {
				continue
			}
			name := strings.TrimSuffix(ent.Name(), ".md")
			s = parseFlatSkill(string(data), name, level)
			s.DirPath = dir
			s.Source = "flat"
		} else {
			continue
		}
		// enabled 过滤：键 "level::name"，不存在默认启用
		if v, ok := enabled[string(level)+"::"+s.Name]; ok && !v {
			continue
		}
		out = append(out, s)
	}
	return out
}

// FindSkill 按名查找技能。找不到返回 nil。
func FindSkill(skills []Skill, name string) *Skill {
	for i := range skills {
		if skills[i].Name == name {
			return &skills[i]
		}
	}
	return nil
}

// ─── frontmatter 解析（不引入 yaml 库，自实现轻量解析）──

// parseFrontmatter 解析 YAML frontmatter（--- 边界）。
// 支持单行 `key: value`，去引号。不支持多行值/嵌套（skill 字段都是单行）。
// 返回 fields（key→value）和 body（frontmatter 之后的正文）。
func parseFrontmatter(text string) (map[string]string, string) {
	fields := map[string]string{}
	lines := strings.Split(text, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return fields, text // 无 frontmatter
	}
	end := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			end = i
			break
		}
		ln := lines[i]
		idx := strings.Index(ln, ":")
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(ln[:idx])
		val := strings.TrimSpace(ln[idx+1:])
		// 去首尾引号（单/双）
		if len(val) >= 2 {
			first, last := val[0], val[len(val)-1]
			if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
				val = val[1 : len(val)-1]
			}
		}
		fields[key] = val
	}
	var body string
	if end >= 0 && end+1 < len(lines) {
		body = strings.TrimSpace(strings.Join(lines[end+1:], "\n"))
	}
	return fields, body
}

// parseSkill 解析目录式 SKILL.md（含 frontmatter）。dirName 用于 name 兜底。
func parseSkill(text, dirName string, level SkillLevel) Skill {
	fields, body := parseFrontmatter(text)
	s := Skill{
		Name:        fields["name"],
		Description: fields["description"],
		Mode:        fields["activation"],
		Globs:       fields["globs"],
		Version:     fields["version"],
		Level:       level,
		Body:        body,
	}
	if s.Name == "" {
		s.Name = dirName // 回退到目录名
	}
	if s.Mode == "" {
		s.Mode = "auto"
	}
	if at := fields["allowed-tools"]; at != "" {
		s.AllowedTools = parseList(at)
	}
	return s
}

// parseFlatSkill 解析旧扁平 .md（首行 # 标题，无 frontmatter）。
func parseFlatSkill(text, fallbackName string, level SkillLevel) Skill {
	s := Skill{Name: fallbackName, Mode: "auto", Level: level, Source: "flat"}
	lines := strings.Split(text, "\n")
	if len(lines) > 0 && strings.HasPrefix(strings.TrimSpace(lines[0]), "#") {
		s.Name = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(lines[0]), "#"))
		lines = lines[1:]
	}
	s.Body = strings.TrimSpace(strings.Join(lines, "\n"))
	// Description：首段非空非 ## 行（最多 120 字）
	for _, ln := range lines {
		if t := strings.TrimSpace(ln); t != "" && !strings.HasPrefix(t, "##") {
			if len([]rune(t)) > 120 {
				t = string([]rune(t)[:120]) + "…"
			}
			s.Description = t
			break
		}
	}
	return s
}

// parseList 解析 [a, b, c] 或 a, b, c 为 []string。
func parseList(s string) []string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]") {
		s = s[1 : len(s)-1]
	}
	if s == "" {
		return nil
	}
	var out []string
	for _, p := range strings.Split(s, ",") {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// ─── L3 资源加载（沙箱）──

// allowedResourceDirs 资源沙箱允许的子目录前缀。
var allowedResourceDirs = []string{"references", "assets", "scripts"}

// LoadSkillResource 加载技能子资源（L3）：references/ assets/ scripts/ 下的文件。
// relPath 必须位于这三个前缀目录内，防路径穿越。maxBytes>0 时限制大小。
func LoadSkillResource(s *Skill, relPath string, maxBytes int64) (string, error) {
	relPath = filepath.Clean(relPath)
	relPath = strings.TrimPrefix(relPath, string(filepath.Separator))
	// 沙箱：必须以允许的前缀开头
	ok := false
	for _, p := range allowedResourceDirs {
		if relPath == p || strings.HasPrefix(relPath, p+string(filepath.Separator)) {
			ok = true
			break
		}
	}
	if !ok {
		return "", fmt.Errorf("资源路径必须位于 references/ assets/ scripts/ 下，得 %q", relPath)
	}
	full := filepath.Join(s.DirPath, relPath)
	// 二次校验：解析后绝对路径必须在 DirPath 内（防 .. 穿越）
	dirAbs, err := filepath.Abs(s.DirPath)
	if err != nil {
		return "", err
	}
	abs, err := filepath.Abs(full)
	if err != nil {
		return "", err
	}
	if abs != dirAbs && !strings.HasPrefix(abs, dirAbs+string(filepath.Separator)) {
		return "", fmt.Errorf("路径穿越被拒: %q", relPath)
	}
	fi, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("资源不存在: %w", err)
	}
	if fi.IsDir() {
		return "", fmt.Errorf("路径是目录不是文件: %q", relPath)
	}
	if maxBytes > 0 && fi.Size() > maxBytes {
		return "", fmt.Errorf("资源过大: %d 字节（上限 %d）", fi.Size(), maxBytes)
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ─── 写入/删除（项目级目录式）──

// WriteSkill 写入项目级技能（目录式 <projectDir>/<name>/SKILL.md，含 frontmatter）。
func WriteSkill(projectDir string, s Skill) error {
	if s.Name == "" {
		return fmt.Errorf("技能名不能为空")
	}
	dir := filepath.Join(projectDir, s.Name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	var sb strings.Builder
	sb.WriteString("---\n")
	fmt.Fprintf(&sb, "name: %s\n", s.Name)
	if s.Description != "" {
		fmt.Fprintf(&sb, "description: %s\n", s.Description)
	}
	if s.Mode != "" {
		fmt.Fprintf(&sb, "activation: %s\n", s.Mode)
	}
	if s.Globs != "" {
		fmt.Fprintf(&sb, "globs: %q\n", s.Globs)
	}
	if len(s.AllowedTools) > 0 {
		fmt.Fprintf(&sb, "allowed-tools: [%s]\n", strings.Join(s.AllowedTools, ", "))
	}
	if s.Version != "" {
		fmt.Fprintf(&sb, "version: %s\n", s.Version)
	}
	sb.WriteString("---\n\n")
	sb.WriteString(s.Body)
	return os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(sb.String()), 0o644)
}

// DeleteSkill 删除项目级技能（整个目录）。
func DeleteSkill(projectDir, name string) error {
	return os.RemoveAll(filepath.Join(projectDir, name))
}

// ─── L1 提示词 ──

// PromptSkills 返回 L1 提示词（所有启用 skill 的 name+description 列表，注入 system prompt）。
func PromptSkills(skills []Skill) string {
	if len(skills) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("\n\n# 可用技能（按需用 load_skill 取全文，load_skill_resource 取子资源）\n")
	for _, s := range skills {
		desc := s.Description
		if desc == "" {
			desc = "（无描述）"
		}
		fmt.Fprintf(&sb, "- %s：%s\n", s.Name, desc)
	}
	return sb.String()
}
