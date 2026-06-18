// 文件树面板 —— 左栏「文件」内容：读真实文件系统，懒加载、展开/折叠、按类型图标着色、
// 点击文件夹展开、点击文件选中（后续接编辑器）。VS Code 深色风。详见 AGENTS.md。
//
//go:build windows

package filetreepanel

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/hoonfeng/paircode/cmd/companion/core"
	"github.com/hoonfeng/paircode/cmd/companion/ui"
	"github.com/hoonfeng/paircode/cmd/companion/ui/editor"
	gitpanel "github.com/hoonfeng/paircode/cmd/companion/ui/git"
	"github.com/hoonfeng/goui/pkg/types"
	"github.com/hoonfeng/goui/pkg/widget"
)

// 文件类型图标(iconForFile→ui.FileIcon)+ 扩展名色已下沉到 ui 包(供文件树行 + 编辑器标签共用)。

// ─── 文件树模型 ───────────────────────────────────────────

type FileNode struct {
	name     string
	path     string
	isDir    bool
	children []*FileNode
	expanded bool
	loaded   bool      // 子节点是否已读
	modTime  time.Time // 目录修改时间（增量刷新用：mtime 未变则不再读盘）
}

// loadChildren 读目录子项：目录在前、各按名排序（不区分大小写）。同时记录目录 mtime。
// 失败置 loaded 防重试。
func loadChildren(n *FileNode) {
	n.loaded = true
	if fi, err := os.Stat(n.path); err == nil {
		n.modTime = fi.ModTime()
	}
	entries, err := os.ReadDir(n.path)
	if err != nil {
		return
	}
	var dirs, files []*FileNode
	for _, e := range entries {
		c := &FileNode{name: e.Name(), path: filepath.Join(n.path, e.Name()), isDir: e.IsDir()}
		if c.isDir {
			dirs = append(dirs, c)
		} else {
			files = append(files, c)
		}
	}
	byName := func(s []*FileNode) {
		sort.Slice(s, func(i, j int) bool { return strings.ToLower(s[i].name) < strings.ToLower(s[j].name) })
	}
	byName(dirs)
	byName(files)
	n.children = append(dirs, files...)
}

// loadChildrenIfNeeded 检查目录 mtime，有变化则重新加载子节点（保留已展开子目录的 expanded 态，
// 并递归加载其 children——否则 flattenFlat 展平时会丢失深层节点）。
// 返回 true 表示子节点列表有变化。mtime 未变则跳过读盘。
func loadChildrenIfNeeded(n *FileNode) bool {
	if !n.loaded {
		loadChildren(n)
		return true
	}
	fi, err := os.Stat(n.path)
	if err != nil {
		return false
	}
	if fi.ModTime().Equal(n.modTime) {
		return false // mtime 未变 → 子节点未变化
	}
	// mtime 变了：重读前备份旧子节点的展开态（包括深层）
	oldExpanded := map[string]bool{}
	collectExpanded(n, oldExpanded)
	n.children = nil
	loadChildren(n) // 会在内部更新 n.modTime
	// 递归恢复之前展开的目录（含深层展开的子目录）
	reExpand(n, oldExpanded)
	return true
}

// ─── 文件树面板（有状态，包级单例：跨 relayout 存活，同 ChatPanel）──

var FileTree = &fileTreeState{}

// FileTreePanel 文件树面板组件。
type FileTreePanel struct{ widget.StatefulWidget }

func (f *FileTreePanel) CreateState() widget.State { return FileTree }

type fileTreeState struct {
	widget.BaseState
	roots     []*FileNode               // 工作区各文件夹的根节点（VS Code 多根）
	active    string                    // 当前选中文件路径
	gitStatus map[string]gitpanel.Badge // 绝对路径→git 状态徽标（每次 Build 重建）
	// 多根拖拽排序（拖根文件夹的手柄重排；首个=Agent 主文件夹）
	dragPath         string  // 正在拖的根路径（""=未拖）
	dragLastY        float64 // 上次光标 Y（累积位移判定换位）
	dragStartPrimary string  // 拖拽开始时的主文件夹（结束时判主是否变→是否重建 agent）
	// 虚拟列表：扁平化的可见节点序列（Build 时重建）
	flatNodes []flatNode
	// 滚动位置缓存（跨重建保留，与 chat.go 同模式）
	cachedScrollOffset float64
	// 文件变更清单（底部可折叠面板）
	changesExpanded bool                     // true=展开显示变更列表
	changes         []gitpanel.FileChange    // 当前变更文件列表（Build 时填充）
}

// flatNode VirtualList 中的扁平节点
type flatNode struct {
	node  *FileNode
	depth int
	root  *FileNode // 非 nil 表示是根行（多根模式下）
	rIdx  int       // 根索引（root 非 nil 时有效）
}

const rootRowH = 24.0 // 根文件夹行高（与 VirtualList ItemHeight 对齐）

func (s *fileTreeState) ensure() {
	if len(s.roots) == 0 {
		s.buildRoots()
	}
}

// buildRoots 据 core.Folders 构建各根（保留原展开态）；无 SetState。
func (s *fileTreeState) buildRoots() {
	exp := map[string]bool{} // 快照展开态
	for _, r := range s.roots {
		collectExpanded(r, exp)
	}
	s.roots = nil
	// 无打开的工作区 → roots 保持 nil（Build 显示「未打开文件夹」空态，而非退当前目录假装加载了项目）。
	for _, p := range core.Folders {
		r := &FileNode{name: filepath.Base(p), path: p, isDir: true, expanded: true}
		loadChildren(r)
		reExpand(r, exp)
		s.roots = append(s.roots, r)
	}
}

// rebuildRoots 工作区文件夹变化后重建 + 刷新（project.go syncWorkspace 调）。
func (s *fileTreeState) RebuildRoots() {
	s.buildRoots()
	s.SetState()
}

func (s *fileTreeState) Build(ctx widget.BuildContext) widget.Widget {
	s.ensure()
	if len(s.roots) == 0 { // 未打开工作区 → 空态（不是加载中），复刻 VS Code「尚未打开文件夹」
		return s.emptyState()
	}
	gitpanel.Ensure()                  // 触发 git 状态异步加载（完成后 git drain 会 refresh 文件树→徽标显现）
	s.gitStatus = gitpanel.StatusMap() // 据当前 git 状态标记改动文件（未加载则 nil）
	s.changes = gitpanel.ChangedFiles() // 获取变更文件列表（供底部变更清单用）
	// 重建扁平节点列表（只存指针，不创建 Widget）
	s.flatNodes = s.flatNodes[:0]
	if len(s.roots) == 1 {
		s.flattenFlat(s.roots[0].children, 0) // 单文件夹：直接显示内容
	} else {
		for idx, r := range s.roots { // 多根：每个文件夹作可折叠根节
			s.flatNodes = append(s.flatNodes, flatNode{node: r, depth: 0, root: r, rIdx: idx})
			if r.expanded {
				s.flattenFlat(r.children, 1)
			}
		}
	}
	itemCount := len(s.flatNodes)
	itemH := 24.0
	// 使用 VirtualList 只构建可见行
	virtualList := &widget.VirtualList{
		ItemCount:    itemCount,
		ItemHeight:   itemH,
		ScrollOffset: s.cachedScrollOffset,
		OnScroll:     func(so float64) { s.cachedScrollOffset = so },
		RenderItem: func(i int) widget.Widget {
			if i < 0 || i >= len(s.flatNodes) {
				return nil
			}
			fn := s.flatNodes[i]
			if fn.root != nil {
				return s.rootRow(fn.root, fn.rIdx)
			}
			return s.row(fn.node, fn.depth)
		},
	}
	panel := widget.Div(
		widget.Style{BackgroundColor: ui.ShellSide, FlexDirection: "column", AlignItems: "stretch"},
		s.toolbar(),
		ui.Expand(virtualList),
		s.changesBar(),
	)
	return &widget.ContextArea{ // 右键空白处：根目录菜单（行的右键已 StopPropagation，不会冒到这）
		SingleChildWidget: widget.SingleChildWidget{Child: panel},
		OnContextMenu: func(x, y float64) {
			if OnEmptyMenu != nil {
				OnEmptyMenu(x, y)
			}
		},
	}
}

// flattenFlat 递归将可见节点加入 flatNodes（不创建 Widget）
func (s *fileTreeState) flattenFlat(nodes []*FileNode, depth int) {
	for _, n := range nodes {
		s.flatNodes = append(s.flatNodes, flatNode{node: n, depth: depth})
		if n.isDir && n.expanded {
			s.flattenFlat(n.children, depth+1)
		}
	}
}

// emptyState 未打开工作区时的占位（复刻 VS Code「尚未打开文件夹」）：顶栏 + 提示 + 「打开文件夹」按钮，
// 而不是退当前目录假装已加载项目。
func (s *fileTreeState) emptyState() widget.Widget {
	return widget.Div(
		widget.Style{BackgroundColor: ui.ShellSide, FlexDirection: "column", AlignItems: "stretch"},
		s.toolbar(),
		widget.Div(
			widget.Style{FlexDirection: "column", AlignItems: "stretch", Gap: 8,
				Padding: types.EdgeInsetsLTRB(14, 16, 14, 14)},
			ui.TextLine("尚未打开文件夹", *ui.ShellTextDim, 12),
			&widget.Button{
				SingleChildWidget: widget.SingleChildWidget{Child: ui.TextC("打开文件夹", *ui.White, 12)},
				OnClick: func() {
					if OnOpenFolder != nil {
						OnOpenFolder()
					}
				},
				Color: *ui.AccentStrong, MinHeight: 32, Padding: types.EdgeInsetsLTRB(14, 0, 14, 0),
			},
		),
	)
}

// toolbar 文件树头部：工作区图标 + 工作区名（醒目，让用户看到打开的是哪个项目）+ 打开文件夹 + 刷新。
func (s *fileTreeState) toolbar() widget.Widget {
	return widget.Div(
		widget.Style{Height: 30, Padding: types.EdgeInsetsLTRB(8, 0, 6, 0), FlexDirection: "row", AlignItems: "center",
			BackgroundColor: ui.ShellSide, BorderColor: ui.ShellBorder, BorderWidth: 1},
		widget.Lucide("folder", widget.IconSize(13), widget.IconColor(*ui.ShellText)),
		widget.Div(widget.Style{Width: 6}),
		ui.Expand(ui.TextLine(core.ProjectName(), *ui.ShellText, 12)), // 工作区名（单文件夹名 / 多根「工作区 (N)」）
		ui.ShellIconBtn("folder-plus", func() { // 添加文件夹到工作区（VS Code 多根）
			if OnAddFolder != nil {
				OnAddFolder()
			}
		}),
		ui.ShellIconBtn("refresh-cw", s.Refresh),
	)
}

// row 单行：整行可点（Clickable，铺满宽 + 选中/悬停高亮）+ 缩进 + 图标 + 文件名。
func (s *fileTreeState) row(n *FileNode, depth int) widget.Widget {
	icon, iconCol := ui.FileIcon(n.name, n.isDir, n.expanded)
	bg := types.Color{}
	if n.path == s.active {
		bg = *ui.FtSelected
	}
	indent := 8.0 + float64(depth)*14
	// git 状态：改动文件名变色 + 行尾状态符（M/?/+ 等）。
	nameCol := *ui.ShellText
	var trailing widget.Widget = widget.Div(widget.Style{})
	if gb, ok := s.gitStatus[n.path]; ok {
		nameCol = gb.Col()
		trailing = ui.TextC(gb.Sym(), gb.Col(), 11)
	}
	// 目录行：添加展开/折叠箭头 (chevron)
	var before widget.Widget
	if n.isDir {
		chev := "chevron-right"
		if n.expanded {
			chev = "chevron-down"
		}
		before = widget.Lucide(chev, widget.IconSize(12), widget.IconColor(*ui.ShellTextDim))
	} else {
		before = widget.Div(widget.Style{Width: 12}) // 对齐缩进
	}
	row := &widget.Clickable{
		SingleChildWidget: widget.SingleChildWidget{Child: widget.Div(
			widget.Style{Height: 24, FlexDirection: "row", AlignItems: "center",
				Padding: types.EdgeInsetsLTRB(indent, 0, 8, 0)},
			before,
			widget.Div(widget.Style{Width: 2}),
			widget.Lucide(icon, widget.IconSize(15), widget.IconColor(iconCol)),
			widget.Div(widget.Style{Width: 6}),
			ui.Expand(ui.TextLine(n.name, nameCol, 12.5)),
			trailing,
		)},
		OnClick:    func() { s.onClick(n) },
		Color:      bg,
		HoverColor: *ui.FtHover,
	}
	return &widget.ContextArea{ // 右键：文件/文件夹菜单（StopPropagation 不冒泡到空白区菜单）
		SingleChildWidget: widget.SingleChildWidget{Child: row},
		OnContextMenu: func(x, y float64) {
			if OnNodeMenu != nil {
				OnNodeMenu(x, y, n)
			}
		},
	}
}

// toggle 展开/折叠目录（右键菜单用，等价点击目录）。
func (s *fileTreeState) Toggle(n *FileNode) {
	if !n.isDir {
		return
	}
	if !n.loaded {
		loadChildren(n)
	}
	n.expanded = !n.expanded
	s.SetState()
}

// selectPath 选中某路径（高亮）。
func (s *fileTreeState) SelectPath(p string) { s.active = p; s.SetState() }

func (s *fileTreeState) onClick(n *FileNode) {
	if n.isDir {
		if !n.loaded {
			loadChildren(n)
		}
		n.expanded = !n.expanded
	} else {
		s.active = n.path
		editorpanel.Editor.Open(n.path) // 在中列编辑区打开（全局 relayout 让编辑区重建读取）
	}
	s.SetState()
}

// rootRow 多根工作区里，每个文件夹的可折叠根行：大写名 + chevron + idx==0 金色星标(Agent 主文件夹)
// + 右侧拖拽手柄(按住上下拖重排，首个=主文件夹)。点行折叠/展开，右键菜单。
func (s *fileTreeState) rootRow(r *FileNode, idx int) widget.Widget {
	chev := "chevron-down"
	if !r.expanded {
		chev = "chevron-right"
	}
	trail, trailCol := "", types.Color{}
	if idx == 0 { // 主文件夹（Agent 首选）：金色星标
		trail, trailCol = "star", types.ColorFromRGB(229, 192, 123)
	}
	// 整行可拖（DragRow 自绘叶子）：点行折叠/展开；长按或拖动→重排（首个=主文件夹）；右键菜单。
	return &widget.DragRow{
		LeadIcon: chev, LeadColor: *ui.ShellTextDim,
		Icon: "folder", Text: strings.ToUpper(r.name), TextColor: *ui.ShellText, TextSize: 11,
		TrailIcon: trail, TrailColor: trailCol,
		Height: rootRowH, Indent: 6,
		Bg: *ui.ShellSide, HoverBg: *ui.FtHover, Active: r.path == s.dragPath,
		OnTap: func() { s.Toggle(r) },
		OnContext: func(x, y float64) {
			if OnRootMenu != nil {
				OnRootMenu(x, y, r.path)
			}
		},
		OnDragStart: func(y float64) { s.onRootDragStart(r.path, y) },
		OnDragMove:  func(y float64) { s.onRootDragMove(y) },
		OnDragEnd:   func() { s.onRootDragEnd() },
	}
}

// onRootDragStart 手柄按下开始拖某根文件夹。
func (s *fileTreeState) onRootDragStart(path string, y float64) {
	s.dragPath = path
	s.dragLastY = y
	s.dragStartPrimary = core.Root()
	s.SetState()
}

// onRootDragMove 拖动中：光标每移过一行高，就与相邻根实时换位（首个=主文件夹）。
func (s *fileTreeState) onRootDragMove(y float64) {
	if s.dragPath == "" {
		return
	}
	for y <= s.dragLastY-rootRowH { // 向上够一行高 → 上移
		i := core.IndexOfFolder(s.dragPath)
		if i <= 0 {
			break
		}
		s.swapRoots(i, i-1)
		s.dragLastY -= rootRowH
	}
	for y >= s.dragLastY+rootRowH { // 向下够一行高 → 下移
		i := core.IndexOfFolder(s.dragPath)
		if i < 0 || i >= len(core.Folders)-1 {
			break
		}
		s.swapRoots(i, i+1)
		s.dragLastY += rootRowH
	}
}

// onRootDragEnd 结束拖拽：落盘新顺序；主文件夹变了才重建 agent。
func (s *fileTreeState) onRootDragEnd() {
	if s.dragPath == "" {
		return
	}
	s.dragPath = ""
	if OnWorkspaceChanged != nil { // 落盘新顺序(project.syncWorkspace 注入；主文件夹变了才重建 agent)
		OnWorkspaceChanged(core.Root() != s.dragStartPrimary)
	}
}

// swapRoots 拖拽中实时换两根（换 core.Folders + s.roots，保留展开态，不落盘——结束时统一落盘）。
func (s *fileTreeState) swapRoots(i, j int) {
	core.Folders[i], core.Folders[j] = core.Folders[j], core.Folders[i]
	if i < len(s.roots) && j < len(s.roots) {
		s.roots[i], s.roots[j] = s.roots[j], s.roots[i]
	}
	s.SetState()
}

// ─── 文件变更清单（底部可折叠面板）────────────────────────

// changesBar 渲染底部文件变更清单：折叠时仅标题行，展开时显示变更文件列表。
func (s *fileTreeState) changesBar() widget.Widget {
	changeCount := len(s.changes)
	if changeCount == 0 {
		return widget.Div(widget.Style{}) // 无变更时不显示
	}
	chev := "chevron-right"
	if s.changesExpanded {
		chev = "chevron-down"
	}
	titleColor := *ui.Warning // 有变更时橙色标题
	// 标题行（整行可点击展开/折叠）
	header := &widget.Clickable{
		SingleChildWidget: widget.SingleChildWidget{Child: widget.Div(
			widget.Style{
				Height: 24, Padding: types.EdgeInsetsLTRB(8, 0, 8, 0),
				FlexDirection: "row", AlignItems: "center",
				BorderColor: ui.ShellBorder, BorderWidth: 1,
			},
			widget.Lucide(chev, widget.IconSize(13), widget.IconColor(titleColor)),
			widget.Div(widget.Style{Width: 4}),
			widget.Lucide("circle-dot", widget.IconSize(12), widget.IconColor(titleColor)),
			widget.Div(widget.Style{Width: 4}),
			ui.Expand(ui.TextC(fmt.Sprintf("文件变更 (%d)", changeCount), titleColor, 11)),
		)},
		OnClick:    func() { s.changesExpanded = !s.changesExpanded; s.SetState() },
		HoverColor: *ui.FtHover,
	}
	if !s.changesExpanded {
		return header // 折叠：只显示标题行
	}
	// 展开：显示变更文件列表（按状态分组，内部滚动）
	var rows []widget.Widget
	// 已暂存
	for _, c := range s.changes {
		if !c.Staged {
			continue
		}
		rows = append(rows, s.changeRow(c))
	}
	// 已修改+冲突+未跟踪
	for _, c := range s.changes {
		if c.Staged {
			continue
		}
		rows = append(rows, s.changeRow(c))
	}
	// 限制最大高度，内部滚动
	maxH := 200.0
	if len(rows)*24 < int(maxH) {
		maxH = float64(len(rows) * 24)
	}
	list := widget.NewScrollView(
		ui.FlexCol(rows...),
	)
	scrollDiv := widget.Div(
		widget.Style{
			MaxHeight: maxH,
			BackgroundColor: ui.ShellSide,
		},
		list,
	)
	return widget.Div(
		widget.Style{FlexDirection: "column", AlignItems: "stretch"},
		header,
		scrollDiv,
	)
}

// changeRow 变更清单中的一行文件。
func (s *fileTreeState) changeRow(c gitpanel.FileChange) widget.Widget {
	return &widget.Clickable{
		SingleChildWidget: widget.SingleChildWidget{Child: widget.Div(
			widget.Style{
				Height: 24, Padding: types.EdgeInsetsLTRB(24, 0, 8, 0),
				FlexDirection: "row", AlignItems: "center",
			},
			widget.Div(widget.Style{Width: 14, FlexDirection: "row", AlignItems: "center"},
				ui.TextC(c.Sym, c.Col, 12)),
			widget.Div(widget.Style{Width: 4}),
			ui.Expand(ui.TextC(c.Path, *ui.ShellText, 11.5)),
		)},
		OnClick: func() {
			abs := filepath.Join(core.Root(), filepath.FromSlash(c.Path))
			s.active = abs
			editorpanel.Editor.Open(abs)
			s.SetState()
		},
		HoverColor: *ui.FtHover,
	}
}

// ═══════════════════════════════════════════════════════════
// 增量刷新：基于 mtime 缓存，避免每次 Refresh 都重建整棵树
// ═══════════════════════════════════════════════════════════

// Refresh 增量刷新文件树：只重新读 mtime 变化的目录，保留树结构和展开态。
// 不再像旧版那样销毁整棵树重建，大幅减少大项目中的磁盘 IO。
func (s *fileTreeState) Refresh() {
	if len(s.roots) == 0 {
		return
	}
	changed := false
	for _, r := range s.roots {
		if refreshNode(r) {
			changed = true
		}
	}
	if changed {
		s.SetState()
	}
}

// RefreshPath 已知某文件/目录路径发生变化时，只刷新包含该路径的目录链。
// 比全量 Refresh 更高效——避免遍历未变化的子树。
func (s *fileTreeState) RefreshPath(absPath string) {
	if len(s.roots) == 0 {
		return
	}
	for _, r := range s.roots {
		if !isDescendant(r.path, absPath) {
			continue
		}
		if refreshPathChain(r, absPath) {
			s.SetState()
		}
		return
	}
	// 未找到包含该路径的根——可能是新文件，fallback 到全量刷新
	s.Refresh()
}

// refreshNode 递归增量刷新节点 n 及其已展开的子目录。
// 对每个目录先检查 mtime，未变则跳过整个子树。返回是否有变化。
func refreshNode(n *FileNode) bool {
	if !n.isDir || !n.loaded {
		return false
	}
	changed := loadChildrenIfNeeded(n)
	// 对已展开的子目录递归刷新
	for _, c := range n.children {
		if c.isDir && c.expanded {
			if refreshNode(c) {
				changed = true
			}
		}
	}
	return changed
}

// refreshPathChain 从节点 n 开始，沿 absPath 的目录链向下刷新。
// 只刷新路径链上的目录，不遍历无关子树。
func refreshPathChain(n *FileNode, absPath string) bool {
	if !n.isDir || !n.loaded {
		return false
	}
	changed := loadChildrenIfNeeded(n)
	if n.path == absPath {
		return changed // 已到目标路径
	}
	// 找包含 absPath 的子目录，沿链刷新
	for _, c := range n.children {
		if c.isDir && isDescendant(c.path, absPath) {
			if refreshPathChain(c, absPath) {
				changed = true
			}
			return changed
		}
	}
	return changed
}

// isDescendant 检查 child 路径是否在 parent 目录之下（含等于）。
func isDescendant(parent, child string) bool {
	parent = filepath.Clean(parent)
	child = filepath.Clean(child)
	if parent == child {
		return true
	}
	rel, err := filepath.Rel(parent, child)
	return err == nil && !strings.HasPrefix(rel, "..")
}

// ─── 全量重建（工作区变化时用）────────────────────────────

func collectExpanded(n *FileNode, exp map[string]bool) {
	for _, c := range n.children {
		if c.isDir && c.expanded {
			exp[c.path] = true
			collectExpanded(c, exp)
		}
	}
}

func reExpand(n *FileNode, exp map[string]bool) {
	for _, c := range n.children {
		if c.isDir && exp[c.path] {
			loadChildren(c)
			c.expanded = true
			reExpand(c, exp)
		}
	}
}

// ftIconBtn 文件树工具条图标按钮已下沉到 ui.ShellIconBtn。
