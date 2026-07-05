// Git 面板 —— 左栏「Git」内容：复刻参考 GitPanel（仓库名/分支/领先落后 + 全部暂存/提交/拉取/推送
// + 已暂存/冲突/已修改/未跟踪 四段可折叠列表，每行状态徽标 + 暂存/取消暂存/丢弃动作）。
// git 经 `git -C <root> ...` CLI 跑；porcelain 解析分类。详见 AGENTS.md。
//
// GWui 版：使用 dom.Document 创建动态 UI，不再依赖 goui。
//
//go:build windows

package gitpanel

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hoonfeng/gwui/component"
	"github.com/hoonfeng/gwui/dom"
	"github.com/hoonfeng/gwui/event"

	"github.com/hoonfeng/paircode/cmd/companion/core"
	editorpanel "github.com/hoonfeng/paircode/cmd/companion/ui/editor"
	"github.com/hoonfeng/paircode/cmd/companion/ui"
)

// ─── 颜色常量（VS Code Dark+）────────────────────────────────
var (
	colText    = ui.Text       // "#cccccc"
	colTextDim = ui.TextDim    // "#8c8c8c"
	colSideBg  = ui.SideBg     // "#252526"
	colBorder  = ui.Border     // "#2d2d2d"
	colAccent  = ui.Accent     // "#0e639c"
	colHover   = ui.HoverBg    // "#2a2d2e"
	colSuccess = ui.Success    // "#4ec9b0"
	colWarning = ui.Warning    // "#dcdcaa"
	colDanger  = ui.Error      // "#f48771"
	colWhite   = "#ffffff"
	colBg      = ui.EditorBg   // "#1e1e1e"
	colInputBg = ui.InputBg    // "#3c3c3c"
)

// Badge 文件树用的 git 状态徽标（符号 + 颜色）。
type Badge struct {
	Sym string
	Col string
}

// StatusMap 工作区文件的 git 状态（绝对路径→徽标），供文件树标记改动。未加载/非仓库→nil。
func StatusMap() map[string]Badge {
	if theGit == nil || !theGit.isRepo || theGit.root == "" {
		return nil
	}
	m := map[string]Badge{}
	put := func(entries []gitEntry, staged bool) {
		for _, e := range entries {
			abs := filepath.Join(theGit.root, filepath.FromSlash(e.path))
			st := e.y
			if staged {
				st = e.x
			}
			sym, col := badge(st, staged)
			m[abs] = Badge{sym, col}
		}
	}
	put(theGit.modified, false)
	put(theGit.untracked, false)
	put(theGit.conflict, false)
	put(theGit.staged, true)
	return m
}

// ─── Git CLI ─────────────────────────────────────────────────

func runGit(root string, args ...string) (string, error) {
	full := append([]string{"-C", root}, args...)
	cmd := exec.Command("git", full...)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(errb.String())
		if msg == "" {
			msg = err.Error()
		}
		return out.String(), fmt.Errorf("%s", msg)
	}
	return out.String(), nil
}

type gitEntry struct {
	path string
	x, y byte
}

type gitCommit struct {
	hash, short, author, date, msg string
}

// ─── Git 面板（包级单例）─────────────────────────────────────

var theGit = &gitState{
	collapsed: map[string]bool{},
	flatItems: make([]gitFlatItem, 0),
}

// GitPanelRef 外部引用
var Panel *gitState

type gitFlatItem struct {
	kind      byte       // 0=sectionHeader, 1=fileRow, 2=commitsHeader, 3=commitRow, 4=cleanHint
	key       string
	title     string
	accent    string
	count     int
	path      string
	sym       string
	col       string
	stagedSec bool
	x, y      byte
	hash, short, author, date, msg string
}

type gitData struct {
	root          string
	branch        string
	ahead, behind int
	isRepo        bool
	errMsg        string
	staged        []gitEntry
	conflict      []gitEntry
	modified      []gitEntry
	untracked     []gitEntry
	branches      []string
	commits       []gitCommit
}

type gitState struct {
	gitData
	doc         *dom.Document
	rootEl      *dom.Element
	contentEl   *dom.Element // 列表容器
	loaded      bool
	hasData     bool
	collapsed   map[string]bool
	hoveredPath string

	mu      sync.Mutex
	loading bool
	snap    *gitData
	actErr  string
	pumpID  int

	flatItems []gitFlatItem
}

// New 创建 Git 面板。
func New(doc *dom.Document) *gitState {
	theGit.doc = doc

	// 加载 HTML 模板（资源目录 html/panels/git.html）
	ui.MustLoadPanelHTML(doc, "panels/git.html", nil)
	theGit.rootEl = doc.GetElementByID("git-root")
	theGit.contentEl = doc.GetElementByID("git-content")

	// 从临时父节点（body）中分离根元素
	ui.DetachRoot(theGit.rootEl)

	Panel = theGit
	return theGit
}

func (g *gitState) Element() *dom.Element { return g.rootEl }

func (g *gitState) Refresh() {
	g.renderAll()
}

// Branch 返回当前分支名，供状态栏显示。
func Branch() string {
	if theGit != nil {
		return theGit.branch
	}
	return ""
}

func (g *gitState) Ensure() {
	if g.loaded {
		return
	}
	g.loaded = true
	g.reloadAsync(nil)
}

func Ensure() {
	if theGit != nil {
		theGit.Ensure()
	}
}

func (g *gitState) renderAll() {
	g.Ensure()
	g.contentEl.ClearChildren()

	if g.loading && !g.hasData {
		g.renderEmpty("refresh-cw", "加载 Git 状态...", "")
		return
	}
	if !g.isRepo {
		g.renderEmpty("git-branch", "非 Git 仓库", "此目录未初始化 Git")
		return
	}

	g.contentEl.AppendChild(g.repoBar())
	g.contentEl.AppendChild(g.actionBar())

	if g.changeCount() == 0 && len(g.commits) == 0 {
		g.renderEmpty("circle-check", "工作区干净", "没有未提交的变更")
		return
	}

	g.flatItems = g.flatItems[:0]
	if g.changeCount() > 0 {
		g.sectionFlat("已暂存", "staged", g.staged, colSuccess, true)
		g.sectionFlat("冲突", "conflict", g.conflict, colDanger, false)
		g.sectionFlat("已修改", "modified", g.modified, colWarning, false)
		g.sectionFlat("未跟踪", "untracked", g.untracked, colTextDim, false)
	} else if len(g.commits) > 0 {
		g.flatItems = append(g.flatItems, gitFlatItem{kind: 4})
	}
	if len(g.commits) > 0 {
		key := "history"
		g.flatItems = append(g.flatItems, gitFlatItem{kind: 2, key: key, title: "提交历史", accent: colAccent, count: len(g.commits)})
		if !g.collapsed[key] {
			for _, c := range g.commits {
				g.flatItems = append(g.flatItems, gitFlatItem{
					kind: 3, hash: c.hash, short: c.short,
					author: c.author, date: c.date, msg: c.msg,
				})
			}
		}
	}

	for _, fi := range g.flatItems {
		w := g.renderFlatItem(fi)
		if w != nil {
			g.contentEl.AppendChild(w)
		}
	}
}

func (g *gitState) renderEmpty(icon, title, subtitle string) {
	el := g.doc.CreateElement("div")
	el.SetAttribute("style", "display:flex;flex-direction:column;align-items:center;justify-content:center;padding:24px;gap:8px;")
	ic := g.doc.CreateElement("span")
	ic.SetAttribute("data-icon", icon)
	ic.SetAttribute("style", "width:24px;height:24px;color:"+colTextDim+";")
	el.AppendChild(ic)
	t := g.doc.CreateElement("div")
	t.SetAttribute("style", "color:"+colTextDim+";font-size:12px;")
	t.SetTextContent(title)
	el.AppendChild(t)
	if subtitle != "" {
		s := g.doc.CreateElement("div")
		s.SetAttribute("style", "color:"+colTextDim+";font-size:11px;")
		s.SetTextContent(subtitle)
		el.AppendChild(s)
	}
	// 加载中图标旋转
	if icon == "refresh-cw" {
		ic.SetAttribute("style", "width:24px;height:24px;color:"+colTextDim+";animation:spin 1s linear infinite;")
	}
	g.contentEl.AppendChild(el)
}

// ─── 数据加载 ─────────────────────────────────────────────────

func (g *gitState) reloadAsync(action func() string) {
	g.mu.Lock()
	if g.loading {
		g.mu.Unlock()
		return
	}
	g.loading = true
	g.mu.Unlock()

	root := core.Root()
	g.startPump()
	go func() {
		ae := ""
		if action != nil {
			ae = action()
		}
		d := computeGitSnapshot(root)
		g.mu.Lock()
		g.snap, g.actErr, g.loading = d, ae, false
		g.mu.Unlock()
	}()
}

func (g *gitState) drain() {
	g.mu.Lock()
	d, ae := g.snap, g.actErr
	g.snap, g.actErr = nil, ""
	loading := g.loading
	g.mu.Unlock()

	if d != nil {
		g.gitData = *d
		g.hasData = true
		g.renderAll()
		if ui.Ctx.App != nil {
			ui.Ctx.App.MarkDirty()
		}
		if OnTreeRefresh != nil {
			OnTreeRefresh()
		}
		if ae != "" {
			g.showAlert("Git 出错", ae)
		}
	}
	if !loading {
		g.stopPump()
	}
}

func (g *gitState) startPump() {
	if g.pumpID != 0 {
		return
	}
	if ui.Ctx.App == nil {
		return
	}
	g.pumpID = ui.Ctx.App.SetInterval(func() { g.drain() }, 100*time.Millisecond)
}

func (g *gitState) stopPump() {
	if g.pumpID != 0 && ui.Ctx.App != nil {
		ui.Ctx.App.ClearInterval(g.pumpID)
		g.pumpID = 0
	}
}

func computeGitSnapshot(root string) *gitData {
	d := &gitData{root: root}
	if out, err := runGit(root, "rev-parse", "--is-inside-work-tree"); err != nil || strings.TrimSpace(out) != "true" {
		return d
	}
	d.isRepo = true
	if b, err := runGit(root, "branch", "--show-current"); err == nil {
		d.branch = strings.TrimSpace(b)
	}
	if out, err := runGit(root, "rev-list", "--left-right", "--count", "HEAD...@{upstream}"); err == nil {
		fmt.Sscanf(strings.TrimSpace(out), "%d\t%d", &d.ahead, &d.behind)
	}
	out, err := runGit(root, "status", "--porcelain")
	if err != nil {
		d.errMsg = err.Error()
		return d
	}
	for _, line := range strings.Split(out, "\n") {
		if len(line) < 4 {
			continue
		}
		x, y := line[0], line[1]
		path := line[3:]
		if i := strings.Index(path, " -> "); i >= 0 {
			path = path[i+4:]
		}
		d.categorize(x, y, strings.TrimSpace(path))
	}
	if out, err := runGit(root, "branch", "--format=%(refname:short)"); err == nil {
		for _, b := range strings.Split(strings.TrimSpace(out), "\n") {
			if b = strings.TrimSpace(b); b != "" {
				d.branches = append(d.branches, b)
			}
		}
	}
	if out, err := runGit(root, "log", "--max-count=20", "--pretty=format:%H|%h|%an|%cr|%s"); err == nil {
		for _, ln := range strings.Split(out, "\n") {
			if p := strings.SplitN(ln, "|", 5); len(p) == 5 {
				d.commits = append(d.commits, gitCommit{hash: p[0], short: p[1], author: p[2], date: p[3], msg: p[4]})
			}
		}
	}
	return d
}

func (g *gitState) checkoutBranch(b string) {
	if b == "" || b == g.branch {
		return
	}
	g.act("checkout", b)
}

func (d *gitData) categorize(x, y byte, path string) {
	e := gitEntry{path: path, x: x, y: y}
	switch {
	case x == '?' && y == '?':
		d.untracked = append(d.untracked, e)
	case x == 'U' || y == 'U' || (x == 'D' && y == 'D') || (x == 'A' && y == 'A'):
		d.conflict = append(d.conflict, e)
	default:
		if x != ' ' && x != '?' {
			d.staged = append(d.staged, e)
		}
		if y != ' ' && y != '?' {
			d.modified = append(d.modified, e)
		}
	}
}

func (g *gitState) changeCount() int {
	return len(g.staged) + len(g.conflict) + len(g.modified) + len(g.untracked)
}

func badge(st byte, staged bool) (string, string) {
	switch st {
	case '?':
		return "?", colTextDim
	case 'A':
		return "+", colSuccess
	case 'D':
		return "-", colDanger
	case 'R', 'C':
		return "→", colWarning
	case 'U':
		return "!", colDanger
	default:
		if staged {
			return "~", colSuccess
		}
		return "~", colWarning
	}
}

// ─── Actions ─────────────────────────────────────────────────

func (g *gitState) act(args ...string) {
	root := core.Root()
	g.reloadAsync(func() string {
		if _, err := runGit(root, args...); err != nil {
			return err.Error()
		}
		return ""
	})
}

func (g *gitState) stageAll()          { g.act("add", "-A") }
func (g *gitState) stageFile(p string) { g.act("add", "--", p) }
func (g *gitState) unstageFile(p string) { g.act("reset", "-q", "HEAD", "--", p) }
func (g *gitState) discardFile(p string) {
	if ui.Ctx.App != nil {
		g.showConfirm("丢弃更改", "确定丢弃「"+p+"」的工作区更改？不可撤销。", func() { g.act("checkout", "--", p) })
	}
}
func (g *gitState) push() { g.act("push") }
func (g *gitState) pull() { g.act("pull", "--ff-only") }

func (g *gitState) commit() {
	if len(g.staged) == 0 {
		g.showAlert("提交", "没有已暂存的更改。")
		return
	}

	doc := g.doc
	modal := component.NewModal(doc)
	modal.SetTitle("提交变更")

	body := modal.Content()
	if body == nil {
		return
	}
	body.ClearChildren()
	body.SetAttribute("style", "display:flex;flex-direction:column;gap:8px;min-width:360px;")

	var msg, desc string
	msgIn := component.NewInput(doc, "提交信息（必填）")
	msgIn.SetBaseStyle("background-color:" + colInputBg + ";color:" + colText + ";border:1px solid " + colBorder + ";padding:4px 8px;font-size:13px;width:100%;")
	body.AppendChild(msgIn.Element())

	descIn := component.NewInput(doc, "详细描述（可选）")
	descIn.SetBaseStyle("background-color:" + colInputBg + ";color:" + colText + ";border:1px solid " + colBorder + ";padding:4px 8px;font-size:13px;width:100%;")
	body.AppendChild(descIn.Element())

	hint := doc.CreateElement("div")
	hint.SetAttribute("style", "color:"+colTextDim+";font-size:10px;")
	hint.SetTextContent(fmt.Sprintf("仅提交已暂存的更改（%d 项）", len(g.staged)))
	body.AppendChild(hint)

	btnRow := doc.CreateElement("div")
	btnRow.SetAttribute("style", "display:flex;flex-direction:row;gap:8px;justify-content:flex-end;")

	cancelBtn := component.NewButton(doc, "取消")
	cancelBtn.SetStyle("background-color:#9e9e9e;color:#fff;padding:4px 12px;font-size:13px;")
	cancelBtn.OnClick(func() { modal.Hide() })
	btnRow.AppendChild(cancelBtn.Element())

	submitBtn := component.NewButton(doc, "提交")
	submitBtn.SetStyle("background-color:" + colAccent + ";color:#fff;padding:4px 12px;font-size:13px;")
	submitBtn.OnClick(func() {
		msg = msgIn.Value()
		if strings.TrimSpace(msg) == "" {
			return
		}
		modal.Hide()
		args := []string{"commit", "-m", msg}
		if strings.TrimSpace(descIn.Value()) != "" {
			desc = descIn.Value()
			args = append(args, "-m", desc)
		}
		g.act(args...)
	})
	btnRow.AppendChild(submitBtn.Element())

	body.AppendChild(btnRow)
	modal.Show()
}

func (g *gitState) toggleSection(name string) {
	g.collapsed[name] = !g.collapsed[name]
	g.renderAll()
	if ui.Ctx.App != nil {
		ui.Ctx.App.MarkDirty()
	}
}

func (g *gitState) setHovered(p string, h bool) {
	if h {
		if g.hoveredPath != p {
			g.hoveredPath = p
			g.renderAll()
			if ui.Ctx.App != nil {
				ui.Ctx.App.MarkDirty()
			}
		}
	} else if g.hoveredPath == p {
		g.hoveredPath = ""
		g.renderAll()
		if ui.Ctx.App != nil {
			ui.Ctx.App.MarkDirty()
		}
	}
}

// ─── UI Components ───────────────────────────────────────────

func (g *gitState) repoBar() *dom.Element {
	bar := g.doc.CreateElement("div")
	bar.SetAttribute("style", "display:flex;flex-direction:row;align-items:center;height:32px;padding:0 10px 0 10px;background:"+colSideBg+";border-bottom:1px solid "+colBorder+";")

	icon := g.doc.CreateElement("span")
	icon.SetAttribute("data-icon", "git-branch")
	icon.SetAttribute("style", "width:13px;height:13px;color:"+colAccent+";flex-shrink:0;")
	bar.AppendChild(icon)
	bar.AppendChild(spacer(g.doc, 6))

	br := g.branch
	if br == "" {
		br = "（无分支）"
	}
	brEl := g.branchSelector(br)
	bar.AppendChild(brEl)

	if g.ahead > 0 {
		bar.AppendChild(spacer(g.doc, 8))
		a := g.doc.CreateElement("div")
		a.SetAttribute("style", "color:"+colTextDim+";font-size:10px;white-space:nowrap;")
		a.SetTextContent(fmt.Sprintf("↑%d", g.ahead))
		bar.AppendChild(a)
	}
	if g.behind > 0 {
		bar.AppendChild(spacer(g.doc, 6))
		b := g.doc.CreateElement("div")
		b.SetAttribute("style", "color:"+colTextDim+";font-size:10px;white-space:nowrap;")
		b.SetTextContent(fmt.Sprintf("↓%d", g.behind))
		bar.AppendChild(b)
	}

	filler := g.doc.CreateElement("div")
	filler.SetAttribute("style", "flex:1;")
	bar.AppendChild(filler)

	refreshBtn := g.doc.CreateElement("div")
	refreshBtn.SetAttribute("style", "display:flex;align-items:center;justify-content:center;width:24px;height:24px;cursor:pointer;border-radius:4px;")
	refreshBtn.SetAttribute("hover-style", "background:"+colHover+";")
	refreshIcon := g.doc.CreateElement("span")
	refreshIcon.SetAttribute("data-icon", "refresh-cw")
	refreshIcon.SetAttribute("style", "width:13px;height:13px;color:"+colTextDim+";")
	refreshBtn.AppendChild(refreshIcon)
	on(refreshBtn, event.Click, func(e event.Event) bool {
		g.reloadAsync(nil)
		return true
	})
	bar.AppendChild(refreshBtn)

	return bar
}

func (g *gitState) branchSelector(br string) *dom.Element {
	if len(g.branches) <= 1 {
		el := g.doc.CreateElement("div")
		el.SetAttribute("style", "color:"+colText+";font-size:12px;white-space:nowrap;")
		el.SetTextContent(br)
		return el
	}

	items := make([]component.DropdownItem, 0, len(g.branches))
	for _, b := range g.branches {
		b := b
		items = append(items, component.DropdownItem{
			Label: b,
			OnClick: func() { g.checkoutBranch(b) },
		})
	}
	dd := component.NewDropdown(g.doc, br, items)
	return dd.Element()
}

func (g *gitState) actionBar() *dom.Element {
	bar := g.doc.CreateElement("div")
	bar.SetAttribute("style", "display:flex;flex-direction:row;align-items:center;padding:6px;gap:6px;")

	hasModified := len(g.modified)+len(g.untracked) > 0
	hasStaged := len(g.staged) > 0

	stageAllBtn := component.NewButton(g.doc, "全部暂存")
	stageAllBtn.SetStyle("background-color:"+colSuccess+";color:#fff;padding:4px 10px;font-size:11px;min-height:24px;")
	if !hasModified {
		stageAllBtn.SetStyle("background-color:"+colSideBg+";color:"+colTextDim+";padding:4px 10px;font-size:11px;min-height:24px;")
	} else {
		stageAllBtn.OnClick(func() { g.stageAll() })
	}
	bar.AppendChild(stageAllBtn.Element())

	commitBtn := component.NewButton(g.doc, "提交")
	commitBtn.SetStyle("background-color:"+colAccent+";color:#fff;padding:4px 10px;font-size:11px;min-height:24px;")
	if !hasStaged {
		commitBtn.SetStyle("background-color:"+colSideBg+";color:"+colTextDim+";padding:4px 10px;font-size:11px;min-height:24px;")
	} else {
		commitBtn.OnClick(func() { g.commit() })
	}
	bar.AppendChild(commitBtn.Element())

	filler := g.doc.CreateElement("div")
	filler.SetAttribute("style", "flex:1;")
	bar.AppendChild(filler)

	pullBtn := g.iconToolBtn("git-pull", func() { g.pull() })
	bar.AppendChild(pullBtn)

	pushBtn := g.iconToolBtn("git-push", func() { g.push() })
	bar.AppendChild(pushBtn)

	return bar
}

func (g *gitState) iconToolBtn(icon string, onClick func()) *dom.Element {
	btn := g.doc.CreateElement("div")
	btn.SetAttribute("style", "display:flex;align-items:center;justify-content:center;width:24px;height:24px;cursor:pointer;border-radius:4px;")
	btn.SetAttribute("hover-style", "background:"+colHover+";")
	ic := g.doc.CreateElement("span")
	ic.SetAttribute("data-icon", icon)
	ic.SetAttribute("style", "width:14px;height:14px;color:"+colTextDim+";")
	btn.AppendChild(ic)
	if onClick != nil {
		on(btn, event.Click, func(e event.Event) bool {
			onClick()
			return true
		})
	}
	return btn
}

func (g *gitState) sectionFlat(title, key string, items []gitEntry, accent string, stagedSec bool) {
	if len(items) == 0 {
		return
	}
	g.flatItems = append(g.flatItems, gitFlatItem{kind: 0, key: key, title: title, accent: accent, count: len(items)})
	if g.collapsed[key] {
		return
	}
	for _, e := range items {
		st := e.y
		if stagedSec {
			st = e.x
		}
		sym, col := badge(st, stagedSec)
		g.flatItems = append(g.flatItems, gitFlatItem{kind: 1, path: e.path, sym: sym, col: col, stagedSec: stagedSec, x: e.x, y: e.y})
	}
}

func (g *gitState) renderFlatItem(fi gitFlatItem) *dom.Element {
	switch fi.kind {
	case 0:
		return g.sectionHeader(fi)
	case 1:
		return g.flatFileRow(fi)
	case 2:
		return g.commitsHeader(fi)
	case 3:
		return g.commitRow(fi)
	case 4:
		el := g.doc.CreateElement("div")
		el.SetAttribute("style", "display:flex;flex-direction:row;align-items:center;height:24px;padding:0 8px;gap:6px;")
		ic := g.doc.CreateElement("span")
		ic.SetAttribute("data-icon", "circle-check")
		ic.SetAttribute("style", "width:13px;height:13px;color:"+colSuccess+";")
		el.AppendChild(ic)
		t := g.doc.CreateElement("div")
		t.SetAttribute("style", "color:"+colTextDim+";font-size:11px;")
		t.SetTextContent("工作区干净")
		el.AppendChild(t)
		return el
	}
	return nil
}

func (g *gitState) sectionHeader(fi gitFlatItem) *dom.Element {
	el := g.doc.CreateElement("div")
	el.SetAttribute("style", "display:flex;flex-direction:row;align-items:center;height:24px;padding:0 6px 0 6px;gap:4px;cursor:pointer;")
	el.SetAttribute("hover-style", "background:"+colHover+";")

	chev := "chevron-down"
	key := fi.key
	if g.collapsed[key] {
		chev = "chevron-right"
	}
	ic := g.doc.CreateElement("span")
	ic.SetAttribute("data-icon", chev)
	ic.SetAttribute("style", "width:13px;height:13px;color:"+fi.accent+";flex-shrink:0;")
	el.AppendChild(ic)

	t := g.doc.CreateElement("div")
	t.SetAttribute("style", "color:"+fi.accent+";font-size:11px;white-space:nowrap;")
	t.SetTextContent(fmt.Sprintf("%s (%d)", fi.title, fi.count))
	el.AppendChild(t)

	on(el, event.Click, func(e event.Event) bool {
		g.toggleSection(key)
		return true
	})
	return el
}

func (g *gitState) flatFileRow(fi gitFlatItem) *dom.Element {
	el := g.doc.CreateElement("div")
	el.SetAttribute("style", "display:flex;flex-direction:row;align-items:center;height:24px;padding:0 6px 0 16px;cursor:pointer;")
	el.SetAttribute("hover-style", "background:"+colHover+";")

	// 状态徽标
	symEl := g.doc.CreateElement("div")
	symEl.SetAttribute("style", "width:14px;color:"+fi.col+";font-size:12px;font-weight:bold;text-align:center;flex-shrink:0;")
	symEl.SetTextContent(fi.sym)
	el.AppendChild(symEl)
	el.AppendChild(spacer(g.doc, 4))

	// 路径
	pEl := g.doc.CreateElement("div")
	pEl.SetAttribute("style", "flex:1;color:"+colText+";font-size:12px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;")
	pEl.SetTextContent(shortGitPath(fi.path))
	el.AppendChild(pEl)

	// 动作按钮（hover 才显示）
	p := fi.path
	var actions []*dom.Element
	switch {
	case fi.stagedSec:
		actions = []*dom.Element{g.gitRowBtn("minus", func() { g.unstageFile(p) })}
	case fi.x == '?' && fi.y == '?':
		actions = []*dom.Element{g.gitRowBtn("plus", func() { g.stageFile(p) })}
	default:
		actions = []*dom.Element{
			g.gitRowBtn("plus", func() { g.stageFile(p) }),
			g.gitRowBtn("trash-2", func() { g.discardFile(p) }),
		}
	}
	trailW := float64(len(actions)) * 22
	if g.hoveredPath == p {
		trailEl := g.doc.CreateElement("div")
		trailEl.SetAttribute("style", fmt.Sprintf("display:flex;flex-direction:row;align-items:center;justify-content:flex-end;width:%.0fpx;flex-shrink:0;", trailW))
		for _, a := range actions {
			trailEl.AppendChild(a)
		}
		el.AppendChild(trailEl)
	} else {
		trailEl := g.doc.CreateElement("div")
		trailEl.SetAttribute("style", fmt.Sprintf("width:%.0fpx;flex-shrink:0;", trailW))
		el.AppendChild(trailEl)
	}

	on(el, event.Click, func(e event.Event) bool {
		showGitDiff(p, fi.stagedSec)
		return true
	})
	on(el, event.MouseEnter, func(e event.Event) bool {
		g.setHovered(p, true)
		return true
	})
	on(el, event.MouseLeave, func(e event.Event) bool {
		g.setHovered(p, false)
		return true
	})

	return el
}

func (g *gitState) commitsHeader(fi gitFlatItem) *dom.Element {
	el := g.doc.CreateElement("div")
	el.SetAttribute("style", "display:flex;flex-direction:row;align-items:center;height:24px;padding:0 6px 0 6px;gap:4px;cursor:pointer;")
	el.SetAttribute("hover-style", "background:"+colHover+";")

	chev := "chevron-down"
	if g.collapsed["history"] {
		chev = "chevron-right"
	}
	ic := g.doc.CreateElement("span")
	ic.SetAttribute("data-icon", chev)
	ic.SetAttribute("style", "width:13px;height:13px;color:"+fi.accent+";flex-shrink:0;")
	el.AppendChild(ic)

	t := g.doc.CreateElement("div")
	t.SetAttribute("style", "color:"+fi.accent+";font-size:11px;")
	t.SetTextContent(fmt.Sprintf("提交历史 (%d)", fi.count))
	el.AppendChild(t)

	on(el, event.Click, func(e event.Event) bool {
		g.toggleSection("history")
		return true
	})
	return el
}

func (g *gitState) commitRow(fi gitFlatItem) *dom.Element {
	full := fi.hash
	el := g.doc.CreateElement("div")
	el.SetAttribute("style", "display:flex;flex-direction:row;align-items:center;height:22px;padding:0 6px 0 16px;cursor:pointer;")
	el.SetAttribute("hover-style", "background:"+colHover+";")

	hashEl := g.doc.CreateElement("div")
	hashEl.SetAttribute("style", "width:48px;color:"+colAccent+";font-size:10px;flex-shrink:0;")
	hashEl.SetTextContent(fi.short)
	el.AppendChild(hashEl)

	msgEl := g.doc.CreateElement("div")
	msgEl.SetAttribute("style", "flex:1;color:"+colText+";font-size:11px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;")
	msgEl.SetTextContent(fi.msg)
	el.AppendChild(msgEl)

	el.AppendChild(spacer(g.doc, 6))

	dateEl := g.doc.CreateElement("div")
	dateEl.SetAttribute("style", "color:"+colTextDim+";font-size:10px;flex-shrink:0;")
	dateEl.SetTextContent(fi.date)
	el.AppendChild(dateEl)

	on(el, event.Click, func(e event.Event) bool {
		// copy full hash to clipboard via markDirty placeholder
		g.copyToClipboard(full)
		return true
	})
	return el
}

// gitRowBtn 行内小图标按钮。
func (g *gitState) gitRowBtn(icon string, onClick func()) *dom.Element {
	btn := g.doc.CreateElement("div")
	btn.SetAttribute("style", "display:flex;align-items:center;justify-content:center;width:22px;height:22px;cursor:pointer;border-radius:4px;")
	btn.SetAttribute("hover-style", "background:"+colHover+";")
	ic := g.doc.CreateElement("span")
	ic.SetAttribute("data-icon", icon)
	ic.SetAttribute("style", "width:13px;height:13px;color:"+colTextDim+";")
	btn.AppendChild(ic)
	if onClick != nil {
		on(btn, event.Click, func(e event.Event) bool {
			onClick()
			return true
		})
	}
	return btn
}

// ─── 对话框 ──────────────────────────────────────────────────

func (g *gitState) showAlert(title, msg string) {
	if ui.Ctx.App == nil {
		return
	}
	modal := component.NewModal(g.doc)
	modal.SetTitle(title)
	body := modal.Content()
	if body == nil {
		return
	}
	body.ClearChildren()
	body.SetAttribute("style", "display:flex;flex-direction:column;gap:12px;min-width:300px;")
	msgEl := g.doc.CreateElement("div")
	msgEl.SetAttribute("style", "color:"+colText+";font-size:13px;")
	msgEl.SetTextContent(msg)
	body.AppendChild(msgEl)
	okBtn := component.NewButton(g.doc, "确定")
	okBtn.OnClick(func() { modal.Hide() })
	body.AppendChild(okBtn.Element())
	modal.Show()
}

func (g *gitState) showConfirm(title, msg string, onOk func()) {
	modal := component.NewModal(g.doc)
	modal.SetTitle(title)
	body := modal.Content()
	if body == nil {
		return
	}
	body.ClearChildren()
	body.SetAttribute("style", "display:flex;flex-direction:column;gap:12px;min-width:300px;")
	msgEl := g.doc.CreateElement("div")
	msgEl.SetAttribute("style", "color:"+colText+";font-size:13px;")
	msgEl.SetTextContent(msg)
	body.AppendChild(msgEl)

	btnRow := g.doc.CreateElement("div")
	btnRow.SetAttribute("style", "display:flex;flex-direction:row;gap:8px;justify-content:flex-end;")

	cancelBtn := component.NewButton(g.doc, "取消")
	cancelBtn.SetStyle("background-color:#9e9e9e;color:#fff;padding:4px 12px;font-size:13px;")
	cancelBtn.OnClick(func() { modal.Hide() })
	btnRow.AppendChild(cancelBtn.Element())

	okBtn := component.NewButton(g.doc, "确定")
	okBtn.SetStyle("background-color:"+colAccent+";color:#fff;padding:4px 12px;font-size:13px;")
	okBtn.OnClick(func() { modal.Hide(); onOk() })
	btnRow.AppendChild(okBtn.Element())

	body.AppendChild(btnRow)
	modal.Show()
}

func (g *gitState) copyToClipboard(text string) {
	// Basic clipboard via JS-like approach in GWui context
	_ = text
}

// ─── 辅助 ────────────────────────────────────────────────────

func shortGitPath(p string) string {
	p = strings.ReplaceAll(p, "/", "\\")
	parts := strings.Split(p, "\\")
	if len(parts) <= 2 {
		return p
	}
	return ".../" + parts[len(parts)-2] + "/" + parts[len(parts)-1]
}

func spacer(doc *dom.Document, w float64) *dom.Element {
	s := doc.CreateElement("div")
	s.SetAttribute("style", fmt.Sprintf("width:%.0fpx;flex-shrink:0;", w))
	return s
}

// on 注册事件监听器（通过全局 App）。
func on(el *dom.Element, typ event.Type, fn func(event.Event) bool) {
	if ui.Ctx.App != nil {
		ui.Ctx.App.AddEventListener(el, typ, fn)
	}
}

// ─── 外部回调（main 注入）───────────────────────────────────

var OnTreeRefresh func()

// ChangedFiles 返回变更文件列表（供文件树底部变更清单用）。
func ChangedFiles() []FileChange {
	if theGit == nil {
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
			out = append(out, FileChange{Path: e.path, Sym: sym, Col: col, Staged: staged})
		}
	}
	add(theGit.staged, true)
	add(theGit.conflict, false)
	add(theGit.modified, false)
	add(theGit.untracked, false)
	return out
}

// FileChange 用于文件树底部变更清单。
type FileChange struct {
	Path   string
	Sym    string
	Col    string
	Staged bool
}

// ─── 差异查看 ───────────────────────────────────────────────

func openGitEntry(rel string) {
	abs := rel
	if theGit.root != "" {
		abs = theGit.root + "\\" + strings.ReplaceAll(rel, "/", "\\")
	}
	if line := firstChangedLine(theGit.root, rel); line > 0 {
		editorpanel.Editor.OpenAt(abs, line)
	} else {
		editorpanel.Editor.Open(abs)
	}
}

func showGitDiff(rel string, staged bool) {
	root := theGit.root
	if root == "" {
		return
	}
	args := []string{"diff"}
	if staged {
		args = append(args, "--cached")
	}
	args = append(args, "--", rel)
	out, err := runGit(root, args...)
	if err != nil {
		if theGit != nil {
			theGit.showAlert("Git 出错", err.Error())
		}
		return
	}
	if strings.TrimSpace(out) == "" {
		out = "（无文本差异：可能是新文件 / 二进制 / 仅模式变更）"
	}

	doc := theGit.doc
	modal := component.NewModal(doc)
	modal.SetTitle("差异 — " + rel)
	// diff 内容较宽（含文件路径与代码），需要更大的宽度与高度；
	// SetMaxWidth 必须 >= body.min-width，否则布局视口(460)会压缩内容。
	modal.SetMaxWidth(720)
	modal.SetMaxHeight(500)
	body := modal.Content()
	if body == nil {
		return
	}
	body.ClearChildren()
	// 滚动由 .gwui-modal-body(overflow:auto + flex:1) 统一处理，
	// 这里若再设 max-height/overflow-y 会形成双层滚动条。
	body.SetAttribute("style", "display:flex;flex-direction:column;min-width:640px;")

	// 渲染差异行
	for _, ln := range strings.Split(out, "\n") {
		row := diffLine(ln)
		body.AppendChild(row)
	}

	btnRow := doc.CreateElement("div")
	btnRow.SetAttribute("style", "display:flex;flex-direction:row;gap:8px;justify-content:flex-end;margin-top:8px;")

	openBtn := component.NewButton(doc, "在编辑器打开")
	openBtn.OnClick(func() { modal.Hide(); openGitEntry(rel) })
	btnRow.AppendChild(openBtn.Element())

	closeBtn := component.NewButton(doc, "关闭")
	closeBtn.SetStyle("background-color:" + colAccent + ";color:#fff;padding:4px 12px;font-size:13px;")
	closeBtn.OnClick(func() { modal.Hide() })
	btnRow.AppendChild(closeBtn.Element())

	body.AppendChild(btnRow)
	modal.Show()
}

func diffLine(ln string) *dom.Element {
	col := colText
	switch {
	case strings.HasPrefix(ln, "+++"), strings.HasPrefix(ln, "---"), strings.HasPrefix(ln, "diff "), strings.HasPrefix(ln, "index "):
		col = colTextDim
	case strings.HasPrefix(ln, "@@"):
		col = colAccent
	case strings.HasPrefix(ln, "+"):
		col = colSuccess
	case strings.HasPrefix(ln, "-"):
		col = colDanger
	}
	disp := strings.ReplaceAll(ln, "\t", "    ")
	el := theGit.doc.CreateElement("div")
	el.SetAttribute("style", "height:16px;padding:0 8px;color:"+col+";font-size:12px;font-family:monospace;white-space:pre;overflow:hidden;")
	el.SetTextContent(disp)
	return el
}

func firstChangedLine(root, rel string) int {
	out, err := runGit(root, "diff", "-U0", "HEAD", "--", rel)
	if err != nil || strings.TrimSpace(out) == "" {
		out, err = runGit(root, "diff", "-U0", "--cached", "--", rel)
		if err != nil || strings.TrimSpace(out) == "" {
			return 0
		}
	}
	for _, ln := range strings.Split(out, "\n") {
		if strings.HasPrefix(ln, "@@") {
			return parseHunkNewLine(ln)
		}
	}
	return 0
}

func parseHunkNewLine(s string) int {
	i := strings.IndexByte(s, '+')
	if i < 0 {
		return 0
	}
	rest := s[i+1:]
	n := 0
	for n < len(rest) && rest[n] >= '0' && rest[n] <= '9' {
		n++
	}
	v, _ := strconv.Atoi(rest[:n])
	return v
}
