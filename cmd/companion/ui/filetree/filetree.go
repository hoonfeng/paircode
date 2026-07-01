// 文件树面板 —— 左栏「文件」内容：读真实文件系统，懒加载、展开/折叠、按类型图标着色、
// 点击文件夹展开、点击文件选中（后续接编辑器）。VS Code 深色风。详见 AGENTS.md。
//
// GWui 版：使用 dom.Document 创建动态 UI，不再依赖 goui。
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

	"github.com/hoonfeng/gwui/dom"
	"github.com/hoonfeng/gwui/event"

	"github.com/hoonfeng/paircode/cmd/companion/core"
	editorpanel "github.com/hoonfeng/paircode/cmd/companion/ui/editor"
	"github.com/hoonfeng/paircode/cmd/companion/ui"
)

// ─── 颜色常量 ────────────────────────────────────────────────
var (
	colText     = ui.Text        // "#cccccc"
	colTextDim  = ui.TextDim     // "#8c8c8c"
	colTextMute = ui.TextMute    // "#6e6e6e"
	colSide     = ui.SideBg      // "#252526"
	colEditor   = ui.EditorBg    // "#1e1e1e"
	colBorder   = ui.Border      // "#2d2d2d"
	colAccent   = ui.Accent      // "#0e639c"
	colHover    = ui.HoverBg     // "#2a2d2e"
	colSelected = ui.ActiveBg    // "#094771"
	colWarning  = ui.Warning     // "#dcdcaa"
	colWhite    = "#ffffff"
)

// FileNode 文件树节点。字段公开，供外部访问。
type FileNode struct {
	Name     string
	Path     string
	IsDir    bool
	Children []*FileNode
	Expanded bool
	Loaded   bool
	ModTime  time.Time
}

func loadChildren(n *FileNode) {
	n.Loaded = true
	if fi, err := os.Stat(n.Path); err == nil {
		n.ModTime = fi.ModTime()
	}
	entries, err := os.ReadDir(n.Path)
	if err != nil {
		return
	}
	var dirs, files []*FileNode
	for _, e := range entries {
		c := &FileNode{Name: e.Name(), Path: filepath.Join(n.Path, e.Name()), IsDir: e.IsDir()}
		if c.IsDir {
			dirs = append(dirs, c)
		} else {
			files = append(files, c)
		}
	}
	byName := func(s []*FileNode) {
		sort.Slice(s, func(i, j int) bool { return strings.ToLower(s[i].Name) < strings.ToLower(s[j].Name) })
	}
	byName(dirs)
	byName(files)
	n.Children = append(dirs, files...)
}

func loadChildrenIfNeeded(n *FileNode) bool {
	if !n.Loaded {
		loadChildren(n)
		return true
	}
	fi, err := os.Stat(n.Path)
	if err != nil {
		return false
	}
	if fi.ModTime().Equal(n.ModTime) {
		return false
	}
	oldExpanded := map[string]bool{}
	collectExpanded(n, oldExpanded)
	n.Children = nil
	loadChildren(n)
	reExpand(n, oldExpanded)
	return true
}

// ─── 文件树面板（包级单例）─────────────────────────────────

var FileTree = &fileTreeState{}
var Panel = FileTree

// FileTreePanel 别名（bridge.go 引用）。
type fileTreePanel = fileTreeState

type fileTreeState struct {
	doc       *dom.Document
	rootEl    *dom.Element
	contentEl *dom.Element

	roots     []*FileNode
	active    string
}

// New 创建文件树面板。
func New(doc *dom.Document) *fileTreeState {
	FileTree.doc = doc

	// 加载 HTML 模板（资源目录 html/panels/filetree.html）
	ui.MustLoadPanelHTML(doc, "panels/filetree.html", nil)
	FileTree.rootEl = doc.GetElementByID("filetree-root")
	FileTree.contentEl = doc.GetElementByID("filetree-content")

	// 从临时父节点（body）中分离根元素
	ui.DetachRoot(FileTree.rootEl)

	// 文件树空白区域右键菜单
	if FileTree.contentEl != nil {
		on(FileTree.contentEl, event.ContextMenu, func(e event.Event) bool {
			if me, ok := e.(*event.MouseEvent); ok {
				// 优先冒泡到文件节点（节点自身的 ContextMenu 处理器会返回 true 并阻止传播）
				// 此处处理空白区域点击：事件必须未被文件节点消费
				if OnEmptyMenu != nil {
					OnEmptyMenu(float64(me.X), float64(me.Y))
				}
			}
			return true
		})
	}

	Panel = FileTree
	return FileTree
}

func (s *fileTreeState) Element() *dom.Element { return s.rootEl }

func (s *fileTreeState) Refresh() {
	s.renderAll()
}

func (s *fileTreeState) RefreshPath(absPath string) {
	if len(s.roots) == 0 {
		return
	}
	for _, r := range s.roots {
		if !isDescendant(r.Path, absPath) {
			continue
		}
		if refreshPathChain(r, absPath) {
			s.renderAll()
			if ui.Ctx.App != nil {
				ui.Ctx.App.MarkDirty()
			}
		}
		return
	}
	s.Refresh()
}

func (s *fileTreeState) SelectPath(p string) {
	s.active = p
	s.renderAll()
	if ui.Ctx.App != nil {
		ui.Ctx.App.MarkDirty()
	}
}

func (s *fileTreeState) RebuildRoots() {
	s.buildRoots()
	s.renderAll()
	if ui.Ctx.App != nil {
		ui.Ctx.App.MarkDirty()
	}
}

func (s *fileTreeState) Toggle(n *FileNode) {
	if !n.IsDir {
		return
	}
	if !n.Loaded {
		loadChildren(n)
	}
	n.Expanded = !n.Expanded
	s.renderAll()
	if ui.Ctx.App != nil {
		ui.Ctx.App.MarkDirty()
	}
}

// ─── 内部 ────────────────────────────────────────────────────

func (s *fileTreeState) ensure() {
	if len(s.roots) == 0 {
		s.buildRoots()
	}
}

func (s *fileTreeState) buildRoots() {
	exp := map[string]bool{}
	for _, r := range s.roots {
		collectExpanded(r, exp)
	}
	s.roots = nil
	for _, p := range core.Folders {
		r := &FileNode{Name: filepath.Base(p), Path: p, IsDir: true, Expanded: true}
		loadChildren(r)
		reExpand(r, exp)
		s.roots = append(s.roots, r)
	}
}

func (s *fileTreeState) renderAll() {
	s.ensure()
	s.contentEl.ClearChildren()

	if len(s.roots) == 0 {
		s.renderEmpty()
		return
	}

	// 工具栏
	s.contentEl.AppendChild(s.toolbar())

	// 树容器（可滚动）
	treeContainer := s.doc.CreateElement("div")
	treeContainer.SetAttribute("style", "flex:1;overflow-y:auto;")

	if len(s.roots) == 1 {
		s.renderNodes(s.roots[0].Children, 0, treeContainer)
	} else {
		for _, r := range s.roots {
			rootRow := s.rootRow(r)
			treeContainer.AppendChild(rootRow)
			if r.Expanded {
				s.renderNodes(r.Children, 1, treeContainer)
			}
		}
	}
	s.contentEl.AppendChild(treeContainer)
}

func (s *fileTreeState) renderEmpty() {
	el := s.doc.CreateElement("div")
	el.SetAttribute("style", "display:flex;flex-direction:column;align-items:stretch;padding:14px 16px;gap:8px;")
	t := s.doc.CreateElement("div")
	t.SetAttribute("style", "color:"+colTextDim+";font-size:12px;")
	t.SetTextContent("尚未打开文件夹")
	el.AppendChild(t)

	openBtn := s.doc.CreateElement("div")
	openBtn.SetAttribute("style", "display:inline-flex;align-items:center;justify-content:center;padding:8px 14px;background:"+colAccent+";color:#fff;cursor:pointer;font-size:12px;min-height:32px;border-radius:4px;")
	openBtn.SetAttribute("hover-style", "background:#1177bb;")
	openBtn.SetTextContent("打开文件夹")
	on(openBtn, event.Click, func(e event.Event) bool {
		if OnOpenFolder != nil {
			OnOpenFolder()
		}
		return true
	})
	el.AppendChild(openBtn)

	s.contentEl.AppendChild(el)
}

func (s *fileTreeState) toolbar() *dom.Element {
	bar := s.doc.CreateElement("div")
	bar.SetAttribute("style", "display:flex;flex-direction:row;align-items:center;height:30px;padding:0 8px 0 8px;background:"+colSide+";border-bottom:1px solid "+colBorder+";flex-shrink:0;")

	folderIcon := s.doc.CreateElement("span")
	folderIcon.SetAttribute("data-icon", "folder")
	folderIcon.SetAttribute("style", "width:16px;height:16px;color:"+colText+";flex-shrink:0;")
	bar.AppendChild(folderIcon)
	bar.AppendChild(spacer(s.doc, 6))

	nameEl := s.doc.CreateElement("div")
	nameEl.SetAttribute("style", "flex:1;color:"+colText+";font-size:13px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;")
	nameEl.SetTextContent(core.ProjectName())
	bar.AppendChild(nameEl)

	addBtn := s.doc.CreateElement("div")
	addBtn.SetAttribute("style", "display:flex;align-items:center;justify-content:center;width:22px;height:22px;cursor:pointer;border-radius:4px;")
	addBtn.SetAttribute("hover-style", "background:"+colHover+";")
	addIcon := s.doc.CreateElement("span")
	addIcon.SetAttribute("data-icon", "folder-plus")
	addIcon.SetAttribute("style", "width:16px;height:16px;color:"+colTextDim+";")
	addBtn.AppendChild(addIcon)
	on(addBtn, event.Click, func(e event.Event) bool {
		if OnAddFolder != nil {
			OnAddFolder()
		}
		return true
	})
	bar.AppendChild(addBtn)

	refBtn := s.doc.CreateElement("div")
	refBtn.SetAttribute("style", "display:flex;align-items:center;justify-content:center;width:22px;height:22px;cursor:pointer;border-radius:4px;")
	refBtn.SetAttribute("hover-style", "background:"+colHover+";")
	refIcon := s.doc.CreateElement("span")
	refIcon.SetAttribute("data-icon", "refresh-cw")
	refIcon.SetAttribute("style", "width:16px;height:16px;color:"+colTextDim+";")
	refBtn.AppendChild(refIcon)
	on(refBtn, event.Click, func(e event.Event) bool {
		s.Refresh()
		return true
	})
	bar.AppendChild(refBtn)

	return bar
}

func (s *fileTreeState) renderNodes(nodes []*FileNode, depth int, parent *dom.Element) {
	for _, n := range nodes {
		parent.AppendChild(s.row(n, depth))
		if n.IsDir && n.Expanded {
			s.renderNodes(n.Children, depth+1, parent)
		}
	}
}

func (s *fileTreeState) row(n *FileNode, depth int) *dom.Element {
	icon, iconCol := fileIcon(n.Name, n.IsDir, n.Expanded)
	bg := ""
	if n.Path == s.active {
		bg = colSelected
	}
	indent := 8.0 + float64(depth)*14

	nameCol := colText

	row := s.doc.CreateElement("div")
	row.SetAttribute("style", fmt.Sprintf("display:flex;flex-direction:row;align-items:center;height:24px;padding:0 8px 0 %.0fpx;cursor:pointer;background:%s;", indent, bg))
	row.SetAttribute("hover-style", "background:"+colHover+";")

	// 展开/折叠箭头
	if n.IsDir {
		chev := "chevron-right"
		if n.Expanded {
			chev = "chevron-down"
		}
		chevIc := s.doc.CreateElement("span")
		chevIc.SetAttribute("data-icon", chev)
		chevIc.SetAttribute("style", "width:12px;height:12px;color:"+colTextDim+";flex-shrink:0;")
		row.AppendChild(chevIc)
	} else {
		row.AppendChild(spacer(s.doc, 12))
	}

	row.AppendChild(spacer(s.doc, 2))

	// 文件类型图标
	ic := s.doc.CreateElement("span")
	ic.SetAttribute("data-icon", icon)
	ic.SetAttribute("style", fmt.Sprintf("width:16px;height:16px;color:%s;flex-shrink:0;", iconCol))
	row.AppendChild(ic)

	row.AppendChild(spacer(s.doc, 6))

	// 文件名
	nameEl := s.doc.CreateElement("div")
	nameEl.SetAttribute("style", fmt.Sprintf("flex:1;color:%s;font-size:13px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;", nameCol))
	nameEl.SetTextContent(n.Name)
	row.AppendChild(nameEl)

	nodePtr := n
	on(row, event.Click, func(e event.Event) bool {
		s.onClick(nodePtr)
		return true
	})
	// 右键菜单：ContextMenu 事件由 app_skia.go 在 mouse up 时派发（ButtonRight → ContextMenu）
	on(row, event.ContextMenu, func(e event.Event) bool {
		if me, ok := e.(*event.MouseEvent); ok && OnNodeMenu != nil {
			OnNodeMenu(float64(me.X), float64(me.Y), nodePtr)
		}
		return true
	})

	return row
}

func (s *fileTreeState) onClick(n *FileNode) {
	if n.IsDir {
		if !n.Loaded {
			loadChildren(n)
		}
		n.Expanded = !n.Expanded
	} else {
		s.active = n.Path
		editorpanel.Editor.Open(n.Path)
	}
	s.renderAll()
	if ui.Ctx.App != nil {
		ui.Ctx.App.MarkDirty()
	}
}

func (s *fileTreeState) rootRow(r *FileNode) *dom.Element {
	chev := "chevron-down"
	if !r.Expanded {
		chev = "chevron-right"
	}

	row := s.doc.CreateElement("div")
	row.SetAttribute("style", "display:flex;flex-direction:row;align-items:center;height:24px;padding:0 8px 0 6px;cursor:pointer;")
	row.SetAttribute("hover-style", "background:"+colHover+";")

	chevIc := s.doc.CreateElement("span")
	chevIc.SetAttribute("data-icon", chev)
	chevIc.SetAttribute("style", "width:12px;height:12px;color:"+colTextDim+";flex-shrink:0;")
	row.AppendChild(chevIc)

	row.AppendChild(spacer(s.doc, 4))

	folderIc := s.doc.CreateElement("span")
	folderIc.SetAttribute("data-icon", "folder")
	folderIc.SetAttribute("style", "width:16px;height:16px;color:"+colAccent+";flex-shrink:0;")
	row.AppendChild(folderIc)

	row.AppendChild(spacer(s.doc, 4))

	nameEl := s.doc.CreateElement("div")
	nameEl.SetAttribute("style", "flex:1;color:"+colText+";font-size:11px;text-transform:uppercase;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;")
	nameEl.SetTextContent(r.Name)
	row.AppendChild(nameEl)

	on(row, event.Click, func(e event.Event) bool {
		s.Toggle(r)
		return true
	})
	// 右键菜单
	on(row, event.ContextMenu, func(e event.Event) bool {
		if me, ok := e.(*event.MouseEvent); ok && OnRootMenu != nil {
			OnRootMenu(float64(me.X), float64(me.Y), r.Path)
		}
		return true
	})

	return row
}

// ─── 增量刷新 ──────────────────────────────────────────────

func refreshNode(n *FileNode) bool {
	if !n.IsDir || !n.Loaded {
		return false
	}
	changed := loadChildrenIfNeeded(n)
	for _, c := range n.Children {
		if c.IsDir && c.Expanded {
			if refreshNode(c) {
				changed = true
			}
		}
	}
	return changed
}

func refreshPathChain(n *FileNode, absPath string) bool {
	if !n.IsDir || !n.Loaded {
		return false
	}
	changed := loadChildrenIfNeeded(n)
	if n.Path == absPath {
		return changed
	}
	for _, c := range n.Children {
		if c.IsDir && isDescendant(c.Path, absPath) {
			if refreshPathChain(c, absPath) {
				changed = true
			}
			return changed
		}
	}
	return changed
}

func isDescendant(parent, child string) bool {
	parent = filepath.Clean(parent)
	child = filepath.Clean(child)
	if parent == child {
		return true
	}
	rel, err := filepath.Rel(parent, child)
	return err == nil && !strings.HasPrefix(rel, "..")
}

func collectExpanded(n *FileNode, exp map[string]bool) {
	for _, c := range n.Children {
		if c.IsDir && c.Expanded {
			exp[c.Path] = true
			collectExpanded(c, exp)
		}
	}
}

func reExpand(n *FileNode, exp map[string]bool) {
	for _, c := range n.Children {
		if c.IsDir && exp[c.Path] {
			loadChildren(c)
			c.Expanded = true
			reExpand(c, exp)
		}
	}
}

// ─── 工具 ────────────────────────────────────────────────────

func fileIcon(name string, isDir, expanded bool) (string, string) {
	if isDir {
		if expanded {
			return "folder-open", colAccent
		}
		return "folder", colAccent
	}
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".go":
		return "file-code", "#00ADD8"
	case ".js", ".ts", ".jsx", ".tsx":
		return "file-code", "#3178C6"
	case ".py":
		return "file-code", "#3572A5"
	case ".rs":
		return "file-code", "#DEA584"
	case ".html", ".htm":
		return "file-code", "#E44D26"
	case ".css", ".scss", ".less":
		return "file-code", "#563D7C"
	case ".json":
		return "file-code", colTextMute
	case ".md":
		return "file-text", colAccent
	case ".png", ".jpg", ".jpeg", ".gif", ".ico", ".svg":
		return "file-image", "#CBBE6E"
	case ".zip", ".tar", ".gz", ".7z", ".rar":
		return "file-archive", "#CBCB6E"
	default:
		return "file", colTextDim
	}
}

func spacer(doc *dom.Document, w float64) *dom.Element {
	s := doc.CreateElement("div")
	s.SetAttribute("style", fmt.Sprintf("width:%.0fpx;flex-shrink:0;", w))
	return s
}

func on(el *dom.Element, typ event.Type, fn func(event.Event) bool) {
	if ui.Ctx.App != nil {
		ui.Ctx.App.AddEventListener(el, typ, fn)
	}
}

// ─── 外部回调 ────────────────────────────────────────────────

var (
	OnNodeMenu         func(x, y float64, n *FileNode)
	OnEmptyMenu        func(x, y float64)
	OnRootMenu         func(x, y float64, path string)
	OnOpenFolder       func()
	OnAddFolder        func()
	OnWorkspaceChanged func(primaryChanged bool)
)
