// filetreepanel 对外接口 —— 文件树面板。外部经 FileTree 单例(Refresh/SelectPath/Toggle/RebuildRoots)用；
// 对话框/右键菜单(在 main 耦合 chat/工作区/dialog)由 main 注入(同 editorpanel.On* 模式)。
//
//go:build windows

package filetreepanel

// 注入回调(都在 main 耦合 chat/工作区/dialog，故注入而非本包持有)。
var (
	OnNodeMenu   func(x, y float64, n *FileNode) // 文件/目录右键菜单(ctxmenu 的 fileNodeMenu)
	OnEmptyMenu  func(x, y float64)              // 空白处根菜单(ctxmenu 的 fileTreeEmptyMenu)
	OnRootMenu   func(x, y float64, path string) // 多根工作区里某根文件夹的右键菜单(ctxmenu 的 workspaceRootMenu)
	OnOpenFolder func()                          // 「打开文件夹」对话框
	OnAddFolder  func()                          // 「添加文件夹到工作区」对话框

	// OnWorkspaceChanged 根拖拽重排后持久化工作区(project.syncWorkspace；primaryChanged=主文件夹是否变→重建 agent)。
	OnWorkspaceChanged func(primaryChanged bool)
)

// Roots 当前各根节点（ctxmenu 算相对路径 / 遍历用）。
func (s *fileTreeState) Roots() []*FileNode { return s.roots }

// SetRoots 直接设根（测试用）。
func (s *fileTreeState) SetRoots(rs ...*FileNode) { s.roots = rs }

// Reset 复位文件树单例（测试用）。
func Reset() { FileTree = &fileTreeState{} }

// NewFileNode 造一个节点（测试用，不读盘）。
func NewFileNode(name, path string, isDir bool) *FileNode {
	return &FileNode{name: name, path: path, isDir: isDir}
}

// ─── FileNode 字段访问(ctxmenu / 测试用，避免导出内部字段名) ───

func (n *FileNode) Path() string   { return n.path }
func (n *FileNode) Name() string   { return n.name }
func (n *FileNode) IsDir() bool    { return n.isDir }
func (n *FileNode) Expanded() bool { return n.expanded }
