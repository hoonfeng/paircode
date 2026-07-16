// skill_loader 测试：覆盖目录式加载、frontmatter 解析、旧扁平兼容、
// 资源沙箱路径穿越拒绝、enabled 过滤、L1/L2/L3 三级渐进披露、写入/删除往返。

package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeTestFile 写测试文件辅助。
func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// ─── frontmatter 解析 ──

func TestParseFrontmatter_WithQuotedValues(t *testing.T) {
	text := "---\nname: \"emoji-icons\"\ndescription: '禁止用 emoji 当图标'\nactivation: auto\n---\n\n# 正文\n细节"
	fields, body := parseFrontmatter(text)
	if fields["name"] != "emoji-icons" {
		t.Errorf("name 去双引号失败: %q", fields["name"])
	}
	if fields["description"] != "禁止用 emoji 当图标" {
		t.Errorf("description 去单引号失败: %q", fields["description"])
	}
	if fields["activation"] != "auto" {
		t.Errorf("activation: %q", fields["activation"])
	}
	if !strings.HasPrefix(body, "# 正文") {
		t.Errorf("body 应从正文开始: %q", body)
	}
}

func TestParseFrontmatter_NoFrontmatter(t *testing.T) {
	text := "# 标题\n正文"
	fields, body := parseFrontmatter(text)
	if len(fields) != 0 {
		t.Errorf("无 frontmatter 时 fields 应为空: %v", fields)
	}
	if body != text {
		t.Errorf("无 frontmatter 时 body 应为原文")
	}
}

// ─── 目录式加载 ──

func TestLoadSkillsFromDir_DirStyle(t *testing.T) {
	dir := t.TempDir()
	skillMd := "---\nname: my-skill\ndescription: 测试技能\nactivation: always\nglobs: \"*.go\"\nversion: 1.2.0\n---\n\n# 我的技能\n正文细节"
	writeTestFile(t, filepath.Join(dir, "my-skill", "SKILL.md"), skillMd)

	skills := loadSkillsFromDir(dir, LevelProject, nil)
	if len(skills) != 1 {
		t.Fatalf("应加载 1 个技能，得 %d", len(skills))
	}
	s := skills[0]
	if s.Name != "my-skill" {
		t.Errorf("Name: %q", s.Name)
	}
	if s.Description != "测试技能" {
		t.Errorf("Description: %q", s.Description)
	}
	if s.Mode != "always" {
		t.Errorf("Mode: %q", s.Mode)
	}
	if s.Globs != "*.go" {
		t.Errorf("Globs: %q", s.Globs)
	}
	if s.Version != "1.2.0" {
		t.Errorf("Version: %q", s.Version)
	}
	if s.Level != LevelProject {
		t.Errorf("Level: %v", s.Level)
	}
	if s.Source != "dir" {
		t.Errorf("Source: %q", s.Source)
	}
	if !strings.Contains(s.Body, "正文细节") {
		t.Errorf("Body 缺正文: %q", s.Body)
	}
}

func TestLoadSkillsFromDir_NameFallbackToDirName(t *testing.T) {
	// frontmatter 缺 name 时回退到目录名
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "fallback-name", "SKILL.md"),
		"---\ndescription: 无 name 字段\n---\n正文")
	skills := loadSkillsFromDir(dir, LevelSystem, nil)
	if len(skills) != 1 || skills[0].Name != "fallback-name" {
		t.Fatalf("name 应回退到目录名，得 %+v", skills)
	}
}

// ─── 旧扁平兼容 ──

func TestLoadSkillsFromDir_FlatStyle(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "legacy.md"), "# 旧技能\n这是描述行\n\n## 细节\n正文")
	skills := loadSkillsFromDir(dir, LevelProject, nil)
	if len(skills) != 1 {
		t.Fatalf("应加载 1 个扁平技能，得 %d", len(skills))
	}
	s := skills[0]
	if s.Name != "旧技能" {
		t.Errorf("扁平式 name 应取首行 # 标题: %q", s.Name)
	}
	if s.Source != "flat" {
		t.Errorf("Source 应为 flat: %q", s.Source)
	}
	if !strings.Contains(s.Description, "描述行") {
		t.Errorf("Description 应取首段非空行: %q", s.Description)
	}
}

// ─── enabled 过滤 ──

func TestLoadAllSkills_EnabledFilter(t *testing.T) {
	sysDir := t.TempDir()
	projDir := t.TempDir()
	writeTestFile(t, filepath.Join(sysDir, "builtin", "SKILL.md"),
		"---\nname: builtin\ndescription: 内置\n---\n正文")
	writeTestFile(t, filepath.Join(projDir, "user-skill", "SKILL.md"),
		"---\nname: user-skill\ndescription: 用户\n---\n正文")

	// 禁用 system::builtin
	enabled := map[string]bool{"system::builtin": false}
	skills := loadAllFrom(sysDir, projDir, enabled)
	if len(skills) != 1 {
		t.Fatalf("禁用内置后应剩 1 个，得 %d", len(skills))
	}
	if skills[0].Name != "user-skill" {
		t.Errorf("应剩 user-skill，得 %q", skills[0].Name)
	}

	// 全启用（enabled 为 nil）
	skills = loadAllFrom(sysDir, projDir, nil)
	if len(skills) != 2 {
		t.Errorf("nil enabled 应全启用，得 %d", len(skills))
	}
}

// ─── L3 资源沙箱 ──

func TestLoadSkillResource_Valid(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "ref-skill", "SKILL.md"),
		"---\nname: ref-skill\ndescription: 带资源\n---\n正文")
	writeTestFile(t, filepath.Join(dir, "ref-skill", "references", "faq.md"), "FAQ 内容")
	writeTestFile(t, filepath.Join(dir, "ref-skill", "scripts", "run.sh"), "echo hi")

	skills := loadSkillsFromDir(dir, LevelProject, nil)
	s := skills[0]

	got, err := LoadSkillResource(&s, "references/faq.md", 0)
	if err != nil {
		t.Fatalf("合法资源加载失败: %v", err)
	}
	if got != "FAQ 内容" {
		t.Errorf("资源内容: %q", got)
	}

	got, err = LoadSkillResource(&s, "scripts/run.sh", 0)
	if err != nil {
		t.Fatalf("scripts 资源加载失败: %v", err)
	}
	if !strings.Contains(got, "echo") {
		t.Errorf("scripts 内容: %q", got)
	}
}

func TestLoadSkillResource_PathTraversalRejected(t *testing.T) {
	dir := t.TempDir()
	// 在技能目录外放一个敏感文件
	writeTestFile(t, filepath.Join(dir, "secret.txt"), "密码")
	writeTestFile(t, filepath.Join(dir, "s", "SKILL.md"),
		"---\nname: s\ndescription: x\n---\n正文")

	skills := loadSkillsFromDir(dir, LevelProject, nil)
	s := skills[0]

	// 非法前缀（直接读 SKILL.md 同级）
	if _, err := LoadSkillResource(&s, "SKILL.md", 0); err == nil {
		t.Error("读 SKILL.md 应被拒（非 references/assets/scripts 前缀）")
	}
	// 路径穿越：references/../secret.txt
	if _, err := LoadSkillResource(&s, "references/../secret.txt", 0); err == nil {
		t.Error("路径穿越 references/../secret.txt 应被拒")
	}
	// 绝对路径
	if _, err := LoadSkillResource(&s, "/etc/passwd", 0); err == nil {
		t.Error("绝对路径应被拒")
	}
}

func TestLoadSkillResource_SizeLimit(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "s", "SKILL.md"),
		"---\nname: s\ndescription: x\n---\n正文")
	big := strings.Repeat("x", 1000)
	writeTestFile(t, filepath.Join(dir, "s", "references", "big.txt"), big)

	skills := loadSkillsFromDir(dir, LevelProject, nil)
	s := skills[0]
	if _, err := LoadSkillResource(&s, "references/big.txt", 100); err == nil {
		t.Error("超 sizeLimit 应被拒")
	}
	// 不超限应成功
	if _, err := LoadSkillResource(&s, "references/big.txt", 2000); err != nil {
		t.Errorf("未超限应成功: %v", err)
	}
}

// ─── L1 提示词 ──

func TestPromptSkills(t *testing.T) {
	skills := []Skill{
		{Name: "a", Description: "技能A"},
		{Name: "b", Description: ""},
	}
	prompt := PromptSkills(skills)
	if !strings.Contains(prompt, "load_skill") {
		t.Error("L1 提示词应提及 load_skill")
	}
	if !strings.Contains(prompt, "a：技能A") {
		t.Error("L1 应含 a 的名+描述")
	}
	if !strings.Contains(prompt, "b：（无描述）") {
		t.Error("空描述应用兜底文案")
	}

	if PromptSkills(nil) != "" {
		t.Error("空技能列表应返回空串")
	}
}

// ─── L2 正文（Body）已在前述加载测试覆盖，此处补 FindSkill ──

func TestFindSkill(t *testing.T) {
	skills := []Skill{{Name: "x"}, {Name: "y"}}
	if FindSkill(skills, "y") == nil {
		t.Error("应找到 y")
	}
	if FindSkill(skills, "z") != nil {
		t.Error("不应找到 z")
	}
}

// ─── 写入/删除往返 ──

func TestWriteAndDeleteSkill(t *testing.T) {
	dir := t.TempDir()
	s := Skill{
		Name:        "written",
		Description: "写入测试",
		Mode:        "always",
		Body:        "这是正文",
	}
	if err := WriteSkill(dir, s); err != nil {
		t.Fatalf("WriteSkill 失败: %v", err)
	}
	// 应能加载回来
	skills := loadSkillsFromDir(dir, LevelProject, nil)
	if len(skills) != 1 || skills[0].Name != "written" {
		t.Fatalf("写入后加载失败: %+v", skills)
	}
	if skills[0].Description != "写入测试" || skills[0].Mode != "always" {
		t.Errorf("字段不符: %+v", skills[0])
	}
	if !strings.Contains(skills[0].Body, "这是正文") {
		t.Errorf("正文不符: %q", skills[0].Body)
	}
	// 删除
	if err := DeleteSkill(dir, "written"); err != nil {
		t.Fatalf("DeleteSkill 失败: %v", err)
	}
	if skills = loadSkillsFromDir(dir, LevelProject, nil); len(skills) != 0 {
		t.Errorf("删除后应无技能，得 %d", len(skills))
	}
}

// ─── 3.6 激活匹配 + allowed-tools 软提示 ──

func TestActiveSkills_Always(t *testing.T) {
	skills := []Skill{
		{Name: "always-one", Mode: "always"},
		{Name: "manual-one", Mode: "manual"},
	}
	got := ActiveSkills(skills, "", nil)
	if len(got) != 1 || got[0].Name != "always-one" {
		t.Errorf("always 应始终激活，manual 不激活，得 %+v", got)
	}
}

func TestActiveSkills_AutoByKeyword(t *testing.T) {
	skills := []Skill{
		{Name: "emoji-icons", Description: "禁止使用 Emoji 作为图标", Mode: "auto"},
		{Name: "unrelated", Description: "无关技能", Mode: "auto"},
	}
	// taskDesc 含 "emoji" 关键词 → 激活 emoji-icons
	got := ActiveSkills(skills, "检查 emoji 使用情况", nil)
	if len(got) != 1 || got[0].Name != "emoji-icons" {
		t.Errorf("关键词匹配应激活 emoji-icons，得 %+v", got)
	}
	// 无关任务不激活 auto 技能
	got = ActiveSkills(skills, "完全无关的任务", nil)
	if len(got) != 0 {
		t.Errorf("无关任务不应激活 auto 技能，得 %+v", got)
	}
}

func TestActiveSkills_AutoByGlobs(t *testing.T) {
	skills := []Skill{
		{Name: "css-rule", Description: "CSS 相关", Mode: "auto", Globs: "*.css *.scss"},
		{Name: "no-glob", Description: "无 glob", Mode: "auto"},
	}
	// 当前编辑 .css 文件 → 激活 css-rule
	got := ActiveSkills(skills, "完全无关的任务描述", []string{"src/style.css"})
	if len(got) != 1 || got[0].Name != "css-rule" {
		t.Errorf("globs 匹配 .css 应激活 css-rule，得 %+v", got)
	}
	// .go 文件不匹配
	got = ActiveSkills(skills, "无关描述", []string{"main.go"})
	if len(got) != 0 {
		t.Errorf(".go 文件不应激活 css-rule，得 %+v", got)
	}
}

func TestActiveSkills_ManualNotAuto(t *testing.T) {
	skills := []Skill{
		{Name: "manual-one", Description: "手动", Mode: "manual", Globs: "*.go"},
	}
	// manual 模式即使 globs 匹配也不自动激活
	got := ActiveSkills(skills, "manual-one", []string{"main.go"})
	if len(got) != 0 {
		t.Errorf("manual 模式不应自动激活，得 %+v", got)
	}
}

func TestActiveSkills_DefaultAuto(t *testing.T) {
	// Mode 空 → 视作 auto
	skills := []Skill{
		{Name: "default-auto", Description: "测试", Mode: ""},
	}
	got := ActiveSkills(skills, "default-auto", nil)
	if len(got) != 1 {
		t.Errorf("空 Mode 应视作 auto，关键词匹配应激活，得 %+v", got)
	}
}

func TestSkillBodyWithTools(t *testing.T) {
	// 无 allowed-tools：返回原正文
	s := Skill{Body: "正文内容", AllowedTools: nil}
	if got := SkillBodyWithTools(s); got != "正文内容" {
		t.Errorf("无 allowed-tools 应返回原正文，得 %q", got)
	}
	// 有 allowed-tools：末尾追加推荐工具集
	s.AllowedTools = []string{"read_file", "write_file"}
	got := SkillBodyWithTools(s)
	if !strings.Contains(got, "正文内容") {
		t.Errorf("应含原正文，得 %q", got)
	}
	if !strings.Contains(got, "推荐工具集") {
		t.Errorf("应含推荐工具集提示，得 %q", got)
	}
	if !strings.Contains(got, "read_file, write_file") {
		t.Errorf("应含工具列表，得 %q", got)
	}
}
