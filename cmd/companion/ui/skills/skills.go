// Package skillspanel 是 Skills 子系统的数据层与编辑对话框。
// 三级（系统内置只读 / 用户全局 / 项目 .pair）管理 SKILL.md。
//
//go:build windows

package skillspanel

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/user/gou-ide/cmd/companion/core"
	"github.com/user/gou-ide/cmd/companion/ui/mcp"
	"github.com/user/gou-ide/cmd/companion/ui"
	"github.com/user/goui/pkg/widget"
)

// Entry 一个 Skill（SKILL.md 的 frontmatter + 正文）。
type Entry struct {
	Name        string
	Description string
	Mode        string // auto / manual / always
	Globs       string
	Tools       string
	Content     string
}

// Levels 三级复用通用层级常量（system/user/project）。
var Levels = []struct{ ID, Name string }{
	{mcppanel.LevelSystem, "系统级"}, {mcppanel.LevelUser, "用户级"}, {mcppanel.LevelProject, "项目级"},
}

// rootFor 某级 skills 目录。
func rootFor(level string) string {
	switch level {
	case mcppanel.LevelSystem:
		return filepath.Join(core.ConfigDir(), "skills")
	case mcppanel.LevelUser:
		return filepath.Join(core.ConfigDir(), "user-skills")
	case mcppanel.LevelProject:
		return filepath.Join(core.Root(), ".pair", "skills")
	}
	return ""
}

// ReadLevel 读某级 skills 目录（目录缺失→nil）。
func ReadLevel(level string) []Entry {
	root := rootFor(level)
	if root == "" {
		return nil
	}
	ents, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	var out []Entry
	for _, e := range ents {
		if !e.IsDir() {
			continue
		}
		md, err := os.ReadFile(filepath.Join(root, e.Name(), "SKILL.md"))
		if err != nil {
			continue
		}
		s := parseMD(string(md))
		if s.Name == "" {
			s.Name = e.Name()
		}
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// parseMD 解析 SKILL.md：--- frontmatter --- 的 key: value + 正文。
func parseMD(md string) Entry {
	var s Entry
	md = strings.ReplaceAll(md, "\r\n", "\n")
	if strings.HasPrefix(md, "---\n") {
		if end := strings.Index(md[4:], "\n---"); end >= 0 {
			fm := md[4 : 4+end]
			s.Content = strings.TrimPrefix(md[4+end+4:], "\n")
			for _, line := range strings.Split(fm, "\n") {
				k, v, ok := strings.Cut(line, ":")
				if !ok {
					continue
				}
				k, v = strings.TrimSpace(k), strings.TrimSpace(v)
				switch k {
				case "name":
					s.Name = v
				case "description":
					s.Description = v
				case "mode", "activation":
					s.Mode = v
				case "globs":
					s.Globs = v
				case "tools", "allowed-tools":
					s.Tools = v
				}
			}
			return s
		}
	}
	s.Content = md
	return s
}

// Write 写某级 <名>/SKILL.md（带 frontmatter）。系统级只读→忽略。
func Write(level string, s Entry) error {
	if strings.TrimSpace(s.Name) == "" {
		return fmt.Errorf("名称必填")
	}
	root := rootFor(level)
	if root == "" {
		return nil
	}
	dir := filepath.Join(root, s.Name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString("name: " + s.Name + "\n")
	if s.Description != "" {
		b.WriteString("description: " + s.Description + "\n")
	}
	if s.Mode != "" {
		b.WriteString("mode: " + s.Mode + "\n")
	}
	if s.Globs != "" {
		b.WriteString("globs: " + s.Globs + "\n")
	}
	if s.Tools != "" {
		b.WriteString("tools: " + s.Tools + "\n")
	}
	b.WriteString("---\n\n")
	b.WriteString(s.Content)
	return os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(b.String()), 0o644)
}

// Delete 删除某级的 skill 目录。系统级只读→忽略。
func Delete(level, name string) error {
	root := rootFor(level)
	if root == "" {
		return nil
	}
	return os.RemoveAll(filepath.Join(root, name))
}

// Enabled 某 skill 是否启用：override 优先，默认全开。
func Enabled(level, name string) bool {
	if v, ok := core.Settings.SkillEnabledOverrides[level+"::"+name]; ok {
		return v
	}
	return true
}

// SetEnabled 改 skill 启用态（存 override map）。
func SetEnabled(level, name string, on bool) {
	if core.Settings.SkillEnabledOverrides == nil {
		core.Settings.SkillEnabledOverrides = map[string]bool{}
	}
	core.Settings.SkillEnabledOverrides[level+"::"+name] = on
	core.Save()
}

// ModeLabel 激活模式中文名。
func ModeLabel(m string) string {
	switch m {
	case "auto":
		return "自动"
	case "manual":
		return "手动"
	case "always":
		return "始终"
	}
	return m
}

// Prompt 合并三级（项目>用户>系统同名去重、按启用过滤）拼进 agent 系统提示。
func Prompt() string {
	var body strings.Builder
	seen := map[string]bool{}
	for _, lv := range []string{mcppanel.LevelProject, mcppanel.LevelUser, mcppanel.LevelSystem} {
		for _, s := range ReadLevel(lv) {
			if seen[s.Name] || !Enabled(lv, s.Name) {
				continue
			}
			seen[s.Name] = true
			switch s.Mode {
			case "manual":
				continue
			case "always":
				body.WriteString("\n\n## 技能：" + s.Name + "（始终遵循）\n")
				if s.Description != "" {
					body.WriteString(s.Description + "\n")
				}
				body.WriteString(strings.TrimSpace(s.Content))
			default:
				body.WriteString("\n- 「" + s.Name + "」：" + s.Description)
			}
		}
	}
	if body.Len() == 0 {
		return ""
	}
	return "\n\n# 可用技能（Skills）\n以下技能可用——标「始终遵循」的务必遵循；其余先看名/描述，判断相关后用 skill_read 读全文再采用：" + body.String()
}

// ─── 编辑对话框 ─────────────────────────────────────────

var theEditor = &editorState{}

// EditorBody Skill 编辑对话框主体。
type EditorBody struct{ widget.StatefulWidget }

func (m *EditorBody) CreateState() widget.State { return theEditor }

type editorState struct {
	widget.BaseState
	level   string
	orig    string
	name    string
	mode    string
	desc    string
	globs   string
	tools   string
	content string
	tok     int
}

// OpenEditor 打开「添加/编辑 Skill」对话框。
func OpenEditor(level string, s Entry, onSaved func()) {
	theEditor.level = level
	theEditor.orig = s.Name
	theEditor.name = s.Name
	theEditor.mode = s.Mode
	if theEditor.mode == "" {
		theEditor.mode = "auto"
	}
	theEditor.desc = s.Description
	theEditor.globs = s.Globs
	theEditor.tools = s.Tools
	theEditor.content = s.Content
	theEditor.tok++
	title := "添加 Skill"
	if s.Name != "" {
		title = "编辑 Skill"
	}
	var id int
	dlg := widget.NewDialog(title, &EditorBody{}).WithWidth(520).WithTransition("fade").WithFooter(
		ui.Btn("取消", func() { widget.HideOverlay(id) }),
		ui.PrimaryBtn("保存", func() {
			name := strings.TrimSpace(theEditor.name)
			if name == "" {
				widget.MessageWarning("请填写 Skill 名称")
				return
			}
			if theEditor.orig != "" && theEditor.orig != name {
				_ = Delete(theEditor.level, theEditor.orig)
			}
			if err := Write(theEditor.level, Entry{Name: name, Mode: theEditor.mode, Description: theEditor.desc,
				Globs: theEditor.globs, Tools: theEditor.tools, Content: theEditor.content}); err != nil {
				widget.MessageError("保存失败：" + err.Error())
				return
			}
			if onSaved != nil {
				onSaved()
			}
			widget.MessageSuccess("已保存 Skill「" + name + "」")
			widget.HideOverlay(id)
		}),
	)
	id = widget.ShowDialog(dlg)
}

func (b *editorState) Build(ctx widget.BuildContext) widget.Widget {
	return widget.Div(widget.Style{Width: 488, FlexDirection: "column", AlignItems: "stretch"},
		ui.Field("Skill 名称", ui.Input("如 my-custom-skill", b.name, b.tok, func(t string) { b.name = t })),
		ui.Field("激活模式", ui.Select(b.mode, []widget.SelectOption{
			{Label: "自动", Value: "auto"}, {Label: "手动", Value: "manual"}, {Label: "始终", Value: "always"},
		}, 460, func(v string) { b.mode = v; b.SetState() })),
		ui.Field("描述", ui.Input("简短描述此 Skill 的功能", b.desc, b.tok, func(t string) { b.desc = t })),
		ui.Field("文件匹配 (globs)", ui.Input("*.ts *.tsx", b.globs, b.tok, func(t string) { b.globs = t })),
		ui.Field("允许的工具", ui.Input("Read Write Edit", b.tools, b.tok, func(t string) { b.tools = t })),
		ui.Field("SKILL.md 内容", ui.Textarea("# 技能说明\n描述触发条件与操作步骤…", b.content, 8, b.tok, func(t string) { b.content = t })),
	)
}
