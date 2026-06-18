// gitpanel 对外接口 —— Git 面板。filetree 经 Ensure/StatusMap/Badge 拿状态显徽标；
// main 经 IsRepo/Branch 显状态栏分支。git 动作改文件后刷新文件树经 OnTreeRefresh 注入(破循环依赖)。
//
//go:build windows

package gitpanel

import "github.com/hoonfeng/goui/pkg/types"

// OnTreeRefresh git 动作(丢弃/暂存等)改动文件后刷新文件树的回调，由 main 注入
// （破 git↔filetree 循环：git 不再直接引用文件树单例）。
var OnTreeRefresh func()

// Ensure 触发 git 状态异步加载（文件树 Build 时调用；完成后经 OnTreeRefresh 刷新徽标）。
func Ensure() { theGit.ensure() }

// IsRepo 当前工作区是否 git 仓库。
func IsRepo() bool { return theGit.isRepo }

// Branch 当前分支名（非仓库为空）。
func Branch() string { return theGit.branch }

// Col / Sym 文件状态徽标的颜色 / 符号（文件树行尾显示用）。
func (b Badge) Col() types.Color { return b.col }
func (b Badge) Sym() string      { return b.sym }

// FileChange 表示一个文件的 git 变更信息（供文件树变更清单用）。
type FileChange struct {
	Path   string       // 相对仓库根的路径（正斜杠）
	Status byte         // git 状态字符：?=未跟踪 M=修改 A=新增 D=删除 U=冲突
	Staged bool         // 是否已暂存
	Col    types.Color  // 状态徽标颜色
	Sym    string       // 状态徽标符号
}

// ChangedFiles 返回当前所有 git 变更文件的列表（已暂存+已修改+冲突+未跟踪）。
// 非仓库或无变更时返回 nil。
func ChangedFiles() []FileChange {
	if theGit == nil || !theGit.isRepo || theGit.root == "" {
		return nil
	}
	var out []FileChange
	add := func(entries []gitEntry, staged bool) {
		for _, e := range entries {
			st := e.y
			if staged {
				st = e.x
			}
			sym, col := badge(st, staged)
			out = append(out, FileChange{Path: e.path, Status: st, Staged: staged, Col: col, Sym: sym})
		}
	}
	add(theGit.staged, true)
	add(theGit.conflict, false)
	add(theGit.modified, false)
	add(theGit.untracked, false)
	return out
}

// ChangeCount 返回当前 git 变更文件总数。
func ChangeCount() int {
	if theGit == nil || !theGit.isRepo {
		return 0
	}
	return len(theGit.staged) + len(theGit.conflict) + len(theGit.modified) + len(theGit.untracked)
}
