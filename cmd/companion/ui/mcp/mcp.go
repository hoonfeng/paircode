// Package mcppanel 是 MCP 服务器配置的数据层与编辑对话框。
// 类型: mcpEntry / 三级 CRUD / 合并加载 / 添加编辑对话框。
//
//go:build windows

package mcppanel

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/user/gou-ide/cmd/companion/agent"
	"github.com/user/gou-ide/cmd/companion/core"
	"github.com/user/gou-ide/cmd/companion/ui"
	"github.com/user/goui/pkg/widget"
)

// Entry 一个 MCP 服务器配置。
type Entry struct {
	Name    string
	Command string
	Args    []string
	Env     map[string]string
}

// MCP 三级：系统（内置只读）/ 用户（全局）/ 项目（.pair）。
const (
	LevelSystem  = "system"
	LevelUser    = "user"
	LevelProject = "project"
)

var Levels = []struct{ ID, Name string }{
	{LevelSystem, "系统级"}, {LevelUser, "用户级"}, {LevelProject, "项目级"},
}

// systemDefaults 内置（系统级）MCP 服务器，只读。
func systemDefaults() []Entry {
	return []Entry{
		{Name: "filesystem", Command: "npx", Args: []string{"-y", "@modelcontextprotocol/server-filesystem"}},
		{Name: "git", Command: "uvx", Args: []string{"mcp-server-git"}},
		{Name: "fetch", Command: "uvx", Args: []string{"mcp-server-fetch"}},
		{Name: "memory", Command: "npx", Args: []string{"-y", "@modelcontextprotocol/server-memory"}},
	}
}

// pathFor 用户/项目级 mcp.json 路径（系统级无文件→""）。
func PathFor(level string) string {
	switch level {
	case LevelUser:
		return filepath.Join(core.ConfigDir(), "mcp.json")
	case LevelProject:
		return filepath.Join(core.Root(), ".pair", "mcp.json")
	}
	return ""
}

type mcpFile struct {
	MCPServers map[string]struct {
		Command string            `json:"command"`
		Args    []string          `json:"args"`
		Env     map[string]string `json:"env,omitempty"`
	} `json:"mcpServers"`
}

func readFile(path string) []Entry {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var f mcpFile
	if json.Unmarshal(data, &f) != nil {
		return nil
	}
	var out []Entry
	for name, s := range f.MCPServers {
		out = append(out, Entry{Name: name, Command: s.Command, Args: s.Args, Env: s.Env})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// ReadLevel 读某一级的服务器（系统=内置，用户/项目=各自 mcp.json）。
func ReadLevel(level string) []Entry {
	if level == LevelSystem {
		return systemDefaults()
	}
	return readFile(PathFor(level))
}

func WriteFile(path string, es []Entry) error {
	m := map[string]any{}
	for _, e := range es {
		if e.Name == "" {
			continue
		}
		entry := map[string]any{"command": e.Command}
		if len(e.Args) > 0 {
			entry["args"] = e.Args
		}
		if len(e.Env) > 0 {
			entry["env"] = e.Env
		}
		m[e.Name] = entry
	}
	data, err := json.MarshalIndent(map[string]any{"mcpServers": m}, "", "  ")
	if err != nil {
		return err
	}
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	return os.WriteFile(path, data, 0o644)
}

// Upsert 新增/更新某一级的服务器（系统级只读→忽略）。
func Upsert(level string, e Entry) error {
	path := PathFor(level)
	if path == "" {
		return nil
	}
	es := readFile(path)
	found := false
	for i := range es {
		if es[i].Name == e.Name {
			es[i] = e
			found = true
			break
		}
	}
	if !found {
		es = append(es, e)
	}
	return WriteFile(path, es)
}

// Delete 删除某一级的服务器（系统级只读→忽略）。
func Delete(level, name string) error {
	path := PathFor(level)
	if path == "" {
		return nil
	}
	es := readFile(path)
	out := es[:0:0]
	for _, e := range es {
		if e.Name != name {
			out = append(out, e)
		}
	}
	return WriteFile(path, out)
}

// Enabled 是否启用：override 优先；默认系统级仅 filesystem 开、用户/项目级全开。
func Enabled(level, name string) bool {
	if v, ok := core.Settings.MCPEnabledOverrides[level+"::"+name]; ok {
		return v
	}
	if level == LevelSystem {
		return name == "filesystem"
	}
	return true
}

// SetEnabled 改某服务器启用态（存 override map）。
func SetEnabled(level, name string, on bool) {
	if core.Settings.MCPEnabledOverrides == nil {
		core.Settings.MCPEnabledOverrides = map[string]bool{}
	}
	core.Settings.MCPEnabledOverrides[level+"::"+name] = on
	core.Save()
}

// LoadConfigs 合并三级（项目>用户>系统，同名去重）、按启用过滤，供 agent 连接；自动连接关→不连。
func LoadConfigs() []agent.MCPServerConfig {
	if !core.Settings.AutoConnectMCP {
		return nil
	}
	var out []agent.MCPServerConfig
	seen := map[string]bool{}
	for _, lv := range []string{LevelProject, LevelUser, LevelSystem} {
		for _, e := range ReadLevel(lv) {
			if e.Command == "" || seen[e.Name] || !Enabled(lv, e.Name) {
				continue
			}
			seen[e.Name] = true
			out = append(out, agent.MCPServerConfig{Name: e.Name, Command: e.Command, Args: e.Args, Env: e.Env})
		}
	}
	return out
}

// ─── 编辑对话框 ─────────────────────────────────────────

var theEditor = &editorState{}

// EditorBody MCP 编辑对话框主体。
type EditorBody struct{ widget.StatefulWidget }

func (m *EditorBody) CreateState() widget.State { return theEditor }

type editorState struct {
	widget.BaseState
	level   string // 写入的层级（user/project；系统级不可编辑）
	orig    string // 原名（编辑时判断是否改名）
	name    string
	command string
	args    string // 空格分隔
	tok     int
}

// OpenEditor 打开「添加/编辑 MCP 服务器」对话框；保存写回该 level 的 mcp.json 后回调 onSaved 刷新。
func OpenEditor(level string, e Entry, onSaved func()) {
	theEditor.level = level
	theEditor.orig = e.Name
	theEditor.name = e.Name
	theEditor.command = e.Command
	theEditor.args = strings.Join(e.Args, " ")
	theEditor.tok++
	title := "添加 MCP 服务器"
	if e.Name != "" {
		title = "编辑 MCP 服务器"
	}
	var id int
	dlg := widget.NewDialog(title, &EditorBody{}).WithWidth(460).WithTransition("fade").WithFooter(
		ui.Btn("取消", func() { widget.HideOverlay(id) }),
		ui.PrimaryBtn("保存", func() {
			name := strings.TrimSpace(theEditor.name)
			if name == "" {
				widget.MessageWarning("请填写服务器名称")
				return
			}
			cmd := strings.TrimSpace(theEditor.command)
			if cmd == "" {
				widget.MessageWarning("请填写命令")
				return
			}
			if theEditor.orig != "" && theEditor.orig != name { // 改名→删旧增新
				_ = Delete(theEditor.level, theEditor.orig)
			}
			if err := Upsert(theEditor.level, Entry{Name: name, Command: cmd, Args: strings.Fields(theEditor.args)}); err != nil {
				widget.MessageError("保存失败：" + err.Error())
				return
			}
			if onSaved != nil {
				onSaved()
			}
			widget.MessageSuccess("已保存 MCP 服务器「" + name + "」（重开对话生效）")
			widget.HideOverlay(id)
		}),
	)
	id = widget.ShowDialog(dlg)
}

func (b *editorState) Build(ctx widget.BuildContext) widget.Widget {
	return widget.Div(widget.Style{Width: 428, FlexDirection: "column", AlignItems: "stretch"},
		ui.Field("名称", ui.Input("如 filesystem", b.name, b.tok, func(t string) { b.name = t })),
		ui.Field("命令", ui.Input("npx / uvx / node", b.command, b.tok, func(t string) { b.command = t })),
		ui.Field("参数（空格分隔）", ui.Input("-y @modelcontextprotocol/server-filesystem", b.args, b.tok, func(t string) { b.args = t })),
		widget.Div(widget.Style{Height: 4}),
		ui.TextC("环境变量 / 传输协议（streamable-http）等高级项可在 mcp.json 手编。", *ui.FgMuted, 10),
	)
}
