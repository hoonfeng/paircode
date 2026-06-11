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
