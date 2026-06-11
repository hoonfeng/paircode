// 编辑器会话持久化：把「当前工作区打开了哪些文件(标签路径)+激活的是哪个」存到
// 主文件夹/.pair/editor_session.json，重启或切换工作区后恢复。
// 只存路径——恢复时从磁盘读各文件「当前内容」（故重开即最新，不留旧快照）。
// 按工作区各存一份（文件落在该工作区主文件夹的 .pair/ 下），切工作区即各自恢复各自的标签。
//
//go:build windows

package editorpanel

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/user/gou-ide/cmd/companion/core"
)

type editorSession struct {
	Files  []string `json:"files"`  // 打开的文件绝对路径，按标签顺序
	Active int      `json:"active"` // 激活标签下标
}

func editorSessionPath() string {
	return filepath.Join(core.Root(), ".pair", "editor_session.json")
}

// persistSession 把当前打开的文件+激活项写入工作区会话文件。打开/关闭/切换标签后调用。
func (e *editorState) persistSession() {
	s := editorSession{Active: e.active}
	for _, t := range e.tabs {
		s.Files = append(s.Files, t.path)
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return
	}
	if os.MkdirAll(filepath.Join(core.Root(), ".pair"), 0o755) != nil {
		return
	}
	_ = os.WriteFile(editorSessionPath(), data, 0o644)
}

// restoreSession 从当前工作区会话文件恢复打开的文件，替换现有标签：
// 逐个从磁盘读「当前内容」，文件已删除/不可达则跳过。启动与切换工作区(Open Folder)时调用。
// 不回写（避免恢复时反复写盘）；调用方在运行期需自行 SetState 触发重绘。
func (e *editorState) RestoreSession() {
	e.tabs = nil
	e.active = 0
	if data, err := os.ReadFile(editorSessionPath()); err == nil {
		var s editorSession
		if json.Unmarshal(data, &s) == nil {
			for _, p := range s.Files {
				if fi, err := os.Stat(p); err != nil || fi.IsDir() {
					continue // 文件已不存在/成了目录→跳过
				}
				t := &editorTab{path: p}
				loadTabContent(t)
				e.tabs = append(e.tabs, t)
			}
			if s.Active >= 0 && s.Active < len(e.tabs) {
				e.active = s.Active
			}
		}
	}
	e.reload++ // 受控重载令牌：让 CodeEditor 同步到恢复后的激活标签内容
}
