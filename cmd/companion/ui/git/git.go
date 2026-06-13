// Git 面板 —— 左栏「Git」内容：复刻参考 GitPanel（仓库名/分支/领先落后 + 全部暂存/提交/拉取/推送
// + 已暂存/冲突/已修改/未跟踪 四段可折叠列表，每行状态徽标 + 暂存/取消暂存/丢弃动作）。
// git 经 `git -C <root> ...` CLI 跑；porcelain 解析分类。详见 AGENTS.md。
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

	"github.com/hoonfeng/paircode/cmd/companion/core"
	"github.com/hoonfeng/paircode/cmd/companion/ui/editor"
	"github.com/hoonfeng/paircode/cmd/companion/ui"
	"github.com/hoonfeng/goui/pkg/animation"
	"github.com/hoonfeng/goui/pkg/types"
	"github.com/hoonfeng/goui/pkg/widget"
)

// Badge 文件树用的 git 状态徽标（符号 + 颜色）。
type Badge struct {
	sym string
	col types.Color
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
	put(theGit.staged, true) // 已暂存优先显示
	return m
}

// Git 状态色统一在 ui 包语义令牌（*ui.Success 新增/*ui.Warning 修改/*ui.Danger 删除/*ui.Accent 蓝/*ui.FgMuted 灰）；本文件改读 ui。

// runGit 在 root 下跑 git，返回 stdout；失败带 stderr。
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

// gitEntry 一个变更项。x/y 为 porcelain 的暂存/工作区状态字符。
type gitEntry struct {
	path string
	x, y byte
}

// gitCommit 一条提交记录（提交历史用）。
type gitCommit struct {
	hash, short, author, date, msg string
}

// gitFlatItem 扁平化的可见项，供 VirtualList 按需渲染（取代 ScrollView 全量创建 Widget）
type gitFlatItem struct {
	kind      byte       // 0=sectionHeader, 1=fileRow, 2=commitsHeader, 3=commitRow, 4=cleanHint
	key       string     // section key（header 折叠用）
	title     string     // section 标题
	accent    types.Color
	count     int        // 项目数（header 显示）
	// fileRow (kind=1)
	path      string
	sym       string
	col       types.Color
	stagedSec bool
	x, y      byte       // git 状态字符（判断动作按钮：未跟踪仅显「暂存」，其余显「暂存+丢弃」）
	// commitRow (kind=3)
	hash, short, author, date, msg string
}

// ─── Git 面板（有状态，包级单例）──────────────────────────────

var theGit = &gitState{collapsed: map[string]bool{}}

// GitPanel Git 面板组件。
type GitPanel struct{ widget.StatefulWidget }

func (g *GitPanel) CreateState() widget.State { return theGit }

// gitData git 读出的数据。可在 goroutine 内独立计算（不碰 UI），作快照跨线程交给 UI 线程应用。
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
	widget.BaseState
	gitData          // 当前显示的数据（promoted：g.branch / g.staged …）
	loaded      bool // 是否已触发首次加载
	hasData     bool // 是否已成功载入一次（加载中显旧数据而非闪「非仓库」）
	collapsed   map[string]bool
	hoveredPath string // 当前 hover 的文件行（hover 才显行内动作按钮）

	// 异步：git CLI 在 goroutine 跑，结果经帧泵在 UI 线程 drain（复刻 agent bridge 跨线程模式）。
	mu      sync.Mutex
	loading bool
	snap    *gitData // goroutine 算好的快照，待 UI 线程应用
	actErr  string   // 待提示的动作错误（drain 里 ShowAlert）
	pump    *animation.Controller

	flatItems []gitFlatItem // 扁平化可见项（Build 时重建，供 VirtualList 用）
}

func (g *gitState) ensure() {
	if g.loaded {
		return
	}
	g.loaded = true
	g.reloadAsync(nil)
}

// reloadAsync 异步重读 git：可选先跑 action（add/commit/checkout…），再算快照；全程在 goroutine，
// 结果经帧泵在 UI 线程 drain 应用 → 大仓刷新/动作不冻 UI。action 返回错误串（空=成功）。
func (g *gitState) reloadAsync(action func() string) {
	g.mu.Lock()
	if g.loading { // 已在跑 → 跳过，避免并发 git
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

// drain 帧泵每帧（UI 线程）：有快照则应用 + 重排 + 文件树刷新 + 动作错误提示；goroutine 完成即停泵。
func (g *gitState) drain() {
	g.mu.Lock()
	d, ae := g.snap, g.actErr
	g.snap, g.actErr = nil, ""
	loading := g.loading
	g.mu.Unlock()

	if d != nil {
		g.gitData = *d
		g.hasData = true
		g.SetState()
		if OnTreeRefresh != nil { // 文件可能因动作新增/删除 → 刷新文件树(main 注入，破循环依赖)
			OnTreeRefresh()
		}
		if ae != "" {
			widget.ShowAlert("Git 出错", ae, widget.MsgWarning, nil)
		}
	}
	if !loading { // goroutine 已结束 → 停泵
		g.stopPump()
	}
}

func (g *gitState) startPump() {
	if g.pump != nil {
		return
	}
	p := animation.NewController(time.Second, animation.Linear)
	p.Repeat = true
	p.OnUpdate = func(float64) { g.drain() }
	g.pump = p
	p.Start()
}

func (g *gitState) stopPump() {
	if g.pump != nil {
		g.pump.Stop()
		g.pump = nil
	}
}

// computeGitSnapshot 在 goroutine 内同步跑 git CLI 算出数据快照（不碰 g/UI）。
func computeGitSnapshot(root string) *gitData {
	d := &gitData{root: root}
	if out, err := runGit(root, "rev-parse", "--is-inside-work-tree"); err != nil || strings.TrimSpace(out) != "true" {
		return d // isRepo=false
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
		if i := strings.Index(path, " -> "); i >= 0 { // 重命名：取新名
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

// checkoutBranch 切换分支（git checkout）。
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

// badge 据状态字符给徽标符号+色（复刻参考 ? + - → ~ !）。
func badge(st byte, staged bool) (string, types.Color) {
	switch st {
	case '?':
		return "?", *ui.FgMuted
	case 'A':
		return "+", *ui.Success
	case 'D':
		return "-", *ui.Danger
	case 'R', 'C':
		return "→", *ui.Warning
	case 'U':
		return "!", *ui.Danger
	default: // M 等
		if staged {
			return "~", *ui.Success
		}
		return "~", *ui.Warning
	}
}

// ─── 动作（git CLI；全程异步：动作 + 重读都在 goroutine，结果经帧泵应用）──────────────

func (g *gitState) act(args ...string) {
	root := core.Root()
	g.reloadAsync(func() string {
		if _, err := runGit(root, args...); err != nil {
			return err.Error()
		}
		return ""
	})
}

func (g *gitState) stageAll()            { g.act("add", "-A") }
func (g *gitState) stageFile(p string)   { g.act("add", "--", p) }
func (g *gitState) unstageFile(p string) { g.act("reset", "-q", "HEAD", "--", p) }
func (g *gitState) discardFile(p string) {
	widget.ShowConfirm("丢弃更改", "确定丢弃「"+p+"」的工作区更改？不可撤销。", widget.MsgWarning,
		func() { g.act("checkout", "--", p) }, nil)
}
func (g *gitState) push() { g.act("push") }
func (g *gitState) pull() { g.act("pull", "--ff-only") }

// commit 弹提交对话框（提交信息 必填 + 详细描述 可选，复刻参考 CommitDialog）。
func (g *gitState) commit() {
	if len(g.staged) == 0 {
		widget.ShowAlert("提交", "没有已暂存的更改。", widget.MsgWarning, nil)
		return
	}
	var msg, desc string
	msgIn := commitInput("提交信息（必填）", 2, func(t string) { msg = t })
	descIn := commitInput("详细描述（可选）", 3, func(t string) { desc = t })
	var id int
	body := widget.Div(
		widget.Style{Width: 400, FlexDirection: "column", AlignItems: "stretch"},
		msgIn,
		widget.Div(widget.Style{Height: 8}),
		descIn,
		widget.Div(widget.Style{Height: 6}),
		ui.TextC(fmt.Sprintf("仅提交已暂存的更改（%d 项）", len(g.staged)), *ui.FgMuted, 10),
	)
	dlg := widget.NewDialog("提交变更", body).WithWidth(440).WithFooter(
		ui.Btn("取消", func() { widget.HideOverlay(id) }),
		ui.PrimaryBtn("提交", func() {
			if strings.TrimSpace(msg) == "" {
				return
			}
			widget.HideOverlay(id)
			args := []string{"commit", "-m", msg}
			if strings.TrimSpace(desc) != "" {
				args = append(args, "-m", desc)
			}
			g.act(args...)
		}),
	)
	id = widget.ShowDialog(dlg)
}

// commitInput 提交对话框的多行输入框（深色）。
func commitInput(placeholder string, rows int, onChanged func(string)) *widget.Input {
	in := widget.NewInput(placeholder, onChanged)
	in.Multiline = true
	in.Rows = rows
	in.Color = *ui.Fg
	in.CursorColor = *ui.Fg
	in.BGColor = *ui.Bg
	in.BorderColor = *ui.Border
	in.FocusBorderColor = *ui.Accent
	in.HoverBorderColor = *ui.Border
	return in
}

func (g *gitState) toggleSection(name string) { g.collapsed[name] = !g.collapsed[name]; g.SetState() }

// ─── UI ───────────────────────────────────────────────────────

func (g *gitState) Build(ctx widget.BuildContext) widget.Widget {
	g.ensure()
	if g.loading && !g.hasData { // 首次加载中：显加载提示（已有数据则继续显旧数据，不闪）
		return ui.EmptyState("refresh-cw", "加载 Git 状态...", "")
	}
	if !g.isRepo {
		return ui.EmptyState("git-branch", "非 Git 仓库", "此目录未初始化 Git")
	}
	body := []widget.Widget{g.repoBar(), g.actionBar()}
	if g.changeCount() == 0 && len(g.commits) == 0 {
		body = append(body, ui.Expand(ui.EmptyState("circle-check", "工作区干净", "没有未提交的变更")))
	} else {
		g.flatItems = g.buildFlatItems()
		if len(g.flatItems) == 0 {
			body = append(body, ui.Expand(ui.EmptyState("circle-check", "工作区干净", "没有未提交的变更")))
		} else {
			body = append(body, ui.Expand(&widget.VirtualList{
				ItemCount:  len(g.flatItems),
				ItemHeight: 24,
				RenderItem: g.renderFlatItem,
			}))
		}
	}
	return widget.Div(
		widget.Style{BackgroundColor: ui.ShellSide, FlexDirection: "column", AlignItems: "stretch"},
		body,
	)
}

// buildFlatItems 据当前 gitData 和折叠态构建扁平可见项列表，供 VirtualList 按 index 渲染。
func (g *gitState) buildFlatItems() []gitFlatItem {
	var out []gitFlatItem
	if g.changeCount() > 0 {
		g.sectionFlat(&out, "已暂存", "staged", g.staged, *ui.Success, true)
		g.sectionFlat(&out, "冲突", "conflict", g.conflict, *ui.Danger, false)
		g.sectionFlat(&out, "已修改", "modified", g.modified, *ui.Warning, false)
		g.sectionFlat(&out, "未跟踪", "untracked", g.untracked, *ui.FgMuted, false)
	} else if len(g.commits) > 0 {
		out = append(out, gitFlatItem{kind: 4}) // cleanHint：工作区干净但有提交历史
	}
	// 提交历史段
	if len(g.commits) > 0 {
		key := "history"
		out = append(out, gitFlatItem{kind: 2, key: key, title: "提交历史", accent: *ui.Accent, count: len(g.commits)})
		if !g.collapsed[key] {
			for _, c := range g.commits {
				out = append(out, gitFlatItem{
					kind: 3, hash: c.hash, short: c.short,
					author: c.author, date: c.date, msg: c.msg,
				})
			}
		}
	}
	return out
}

// sectionFlat 把一个变更段追加到扁平可见项列表（取代 section 直接造 Widget）。
func (g *gitState) sectionFlat(out *[]gitFlatItem, title, key string, items []gitEntry, accent types.Color, stagedSec bool) {
	if len(items) == 0 {
		return
	}
	*out = append(*out, gitFlatItem{kind: 0, key: key, title: title, accent: accent, count: len(items)})
	if g.collapsed[key] {
		return
	}
	for _, e := range items {
		st := e.y
		if stagedSec {
			st = e.x
		}
		sym, col := badge(st, stagedSec)
		*out = append(*out, gitFlatItem{kind: 1, path: e.path, sym: sym, col: col, stagedSec: stagedSec, x: e.x, y: e.y})
	}
}

// renderFlatItem VirtualList 回调：按 index 渲染一个扁平项。
func (g *gitState) renderFlatItem(i int) widget.Widget {
	if i < 0 || i >= len(g.flatItems) {
		return nil
	}
	fi := g.flatItems[i]
	switch fi.kind {
	case 0: // sectionHeader
		chev := "chevron-down"
		key := fi.key
		if g.collapsed[key] {
			chev = "chevron-right"
		}
		return &widget.Clickable{
			SingleChildWidget: widget.SingleChildWidget{Child: widget.Div(
				widget.Style{Height: 24, Padding: types.EdgeInsetsLTRB(6, 0, 8, 0), FlexDirection: "row", AlignItems: "center"},
				widget.Lucide(chev, widget.IconSize(13), widget.IconColor(fi.accent)),
				widget.Div(widget.Style{Width: 4}),
				ui.TextC(fmt.Sprintf("%s (%d)", fi.title, fi.count), fi.accent, 11),
			)},
			OnClick:    func() { g.toggleSection(key) },
			HoverColor: *ui.FtHover,
		}
	case 1: // fileRow
		return g.flatFileRow(fi)
	case 2: // commitsHeader
		chev := "chevron-down"
		if g.collapsed["history"] {
			chev = "chevron-right"
		}
		return &widget.Clickable{
			SingleChildWidget: widget.SingleChildWidget{Child: widget.Div(
				widget.Style{Height: 24, Padding: types.EdgeInsetsLTRB(6, 0, 8, 0), FlexDirection: "row", AlignItems: "center"},
				widget.Lucide(chev, widget.IconSize(13), widget.IconColor(fi.accent)),
				widget.Div(widget.Style{Width: 4}),
				ui.TextC(fmt.Sprintf("提交历史 (%d)", fi.count), fi.accent, 11),
			)},
			OnClick:    func() { g.toggleSection("history") },
			HoverColor: *ui.FtHover,
		}
	case 3: // commitRow
		short, msg, date, full := fi.short, fi.msg, fi.date, fi.hash
		return &widget.Clickable{
			SingleChildWidget: widget.SingleChildWidget{Child: widget.Div(
				widget.Style{Height: 22, Padding: types.EdgeInsetsLTRB(16, 0, 6, 0), FlexDirection: "row", AlignItems: "center"},
				widget.Div(widget.Style{Width: 48, FlexDirection: "row", AlignItems: "center"}, ui.TextC(short, *ui.Accent, 10)),
				ui.Expand(ui.TextLine(msg, *ui.ShellText, 11)),
				widget.Div(widget.Style{Width: 6}),
				ui.TextC(date, *ui.ShellTextDim, 10),
			)},
			OnClick: func() {
				if widget.ClipboardWrite != nil {
					widget.ClipboardWrite(full)
				}
			},
			HoverColor: *ui.FtHover,
		}
	case 4: // cleanHint
		return widget.Div(
			widget.Style{Height: 24, Padding: types.EdgeInsetsLTRB(8, 0, 8, 0), FlexDirection: "row", AlignItems: "center"},
			widget.Lucide("circle-check", widget.IconSize(13), widget.IconColor(*ui.Success)),
			widget.Div(widget.Style{Width: 6}), ui.TextC("工作区干净", *ui.ShellTextDim, 11),
		)
	}
	return nil
}

// flatFileRow 从 gitFlatItem 渲染 git 文件行（取代 fileRow，但用 flatItem 的数据）。
func (g *gitState) flatFileRow(fi gitFlatItem) widget.Widget {
	p := fi.path
	var actions []widget.Widget
	switch {
	case fi.stagedSec:
		actions = []widget.Widget{gitRowBtn("minus", "取消暂存", func() { g.unstageFile(p) })}
	case fi.x == '?' && fi.y == '?':
		actions = []widget.Widget{gitRowBtn("plus", "暂存", func() { g.stageFile(p) })}
	default:
		actions = []widget.Widget{
			gitRowBtn("plus", "暂存", func() { g.stageFile(p) }),
			gitRowBtn("trash-2", "丢弃", func() { g.discardFile(p) }),
		}
	}
	trailW := float64(len(actions)) * 22
	trailing := widget.Div(widget.Style{Width: trailW, FlexDirection: "row", AlignItems: "center", JustifyContent: "flex-end"})
	if g.hoveredPath == p {
		trailing = widget.Div(widget.Style{Width: trailW, FlexDirection: "row", AlignItems: "center", JustifyContent: "flex-end"}, actions)
	}
	return &widget.Clickable{
		SingleChildWidget: widget.SingleChildWidget{Child: widget.Div(
			widget.Style{Height: 24, Padding: types.EdgeInsetsLTRB(16, 0, 6, 0), FlexDirection: "row", AlignItems: "center"},
			widget.Div(widget.Style{Width: 14, FlexDirection: "row", AlignItems: "center"}, ui.TextC(fi.sym, fi.col, 12)),
			widget.Div(widget.Style{Width: 4}),
			ui.Expand(ui.TextC(shortGitPath(p), *ui.ShellText, 12)),
			trailing,
		)},
		OnClick:       func() { showGitDiff(p, fi.stagedSec) },
		OnHoverChange: func(h bool) { g.setHovered(p, h) },
		HoverColor:    *ui.FtHover,
	}
}

// repoBar 顶部：分支名 + 领先/落后 + 刷新。
func (g *gitState) repoBar() widget.Widget {
	br := g.branch
	if br == "" {
		br = "（无分支）"
	}
	kids := []widget.Widget{
		widget.Lucide("git-branch", widget.IconSize(13), widget.IconColor(*ui.Accent)),
		widget.Div(widget.Style{Width: 6}),
		g.branchSelector(br),
	}
	if g.ahead > 0 {
		kids = append(kids, widget.Div(widget.Style{Width: 8}), ui.TextC(fmt.Sprintf("↑%d", g.ahead), *ui.ShellTextDim, 10))
	}
	if g.behind > 0 {
		kids = append(kids, widget.Div(widget.Style{Width: 6}), ui.TextC(fmt.Sprintf("↓%d", g.behind), *ui.ShellTextDim, 10))
	}
	kids = append(kids, ui.Expand(widget.Div(widget.Style{})), ui.ShellIconBtn("refresh-cw", func() { g.reloadAsync(nil) }))
	return widget.Div(
		widget.Style{Height: 32, Padding: types.EdgeInsetsLTRB(10, 0, 6, 0), BackgroundColor: ui.ShellSide,
			BorderColor: ui.ShellBorder, BorderWidth: 1, FlexDirection: "row", AlignItems: "center"},
		kids,
	)
}

// branchSelector 当前分支 + 下拉切换（>1 分支才下拉，否则纯标签）。
func (g *gitState) branchSelector(br string) widget.Widget {
	if len(g.branches) <= 1 {
		return ui.TextC(br, *ui.ShellText, 12)
	}
	items := make([]widget.DropdownItem, 0, len(g.branches))
	for _, b := range g.branches {
		items = append(items, widget.DropdownItem{Label: b, Command: b, Checked: b == g.branch})
	}
	trigger := &widget.Button{
		SingleChildWidget: widget.SingleChildWidget{Child: widget.Div(
			widget.Style{FlexDirection: "row", AlignItems: "center"},
			ui.TextC(br, *ui.ShellText, 12),
			widget.Div(widget.Style{Width: 3}),
			widget.Lucide("chevron-down", widget.IconSize(12), widget.IconColor(*ui.ShellTextDim)),
		)},
		Color: *ui.ShellSide, MinHeight: 22, Padding: types.EdgeInsetsLTRB(2, 0, 2, 0),
	}
	return widget.NewDropdown(trigger, items...).WithOnCommand(g.checkoutBranch).WithPlacement(widget.PlacementBottomStart)
}

// commitHistory 追加「提交历史」可折叠段 + 提交行。
func (g *gitState) commitHistory(out *[]widget.Widget) {
	if len(g.commits) == 0 {
		return
	}
	key := "history"
	chev := "chevron-down"
	if g.collapsed[key] {
		chev = "chevron-right"
	}
	*out = append(*out, &widget.Clickable{
		SingleChildWidget: widget.SingleChildWidget{Child: widget.Div(
			widget.Style{Height: 24, Padding: types.EdgeInsetsLTRB(6, 0, 8, 0), FlexDirection: "row", AlignItems: "center"},
			widget.Lucide(chev, widget.IconSize(13), widget.IconColor(*ui.Accent)),
			widget.Div(widget.Style{Width: 4}),
			ui.TextC(fmt.Sprintf("提交历史 (%d)", len(g.commits)), *ui.Accent, 11),
		)},
		OnClick:    func() { g.toggleSection(key) },
		HoverColor: *ui.FtHover,
	})
	if g.collapsed[key] {
		return
	}
	for _, c := range g.commits {
		*out = append(*out, g.commitRow(c))
	}
}

// commitRow 一条提交：短哈希(点击复制全哈希) + 主题 + 相对日期。
func (g *gitState) commitRow(c gitCommit) widget.Widget {
	full := c.hash
	return &widget.Clickable{
		SingleChildWidget: widget.SingleChildWidget{Child: widget.Div(
			widget.Style{Height: 22, Padding: types.EdgeInsetsLTRB(16, 0, 6, 0), FlexDirection: "row", AlignItems: "center"},
			widget.Div(widget.Style{Width: 48, FlexDirection: "row", AlignItems: "center"}, ui.TextC(c.short, *ui.Accent, 10)),
			ui.Expand(ui.TextLine(c.msg, *ui.ShellText, 11)), // 单行省略，避免长消息换行撑破固定行高→重叠
			widget.Div(widget.Style{Width: 6}),
			ui.TextC(c.date, *ui.ShellTextDim, 10),
		)},
		OnClick: func() {
			if widget.ClipboardWrite != nil {
				widget.ClipboardWrite(full)
			}
		},
		HoverColor: *ui.FtHover,
	}
}

// actionBar 全部暂存 / 提交 / 拉取 / 推送。
func (g *gitState) actionBar() widget.Widget {
	return widget.Div(
		widget.Style{Padding: types.EdgeInsets(6), FlexDirection: "row", AlignItems: "center"},
		gitBtn("全部暂存", *ui.Success, len(g.modified)+len(g.untracked) > 0, g.stageAll),
		widget.Div(widget.Style{Width: 6}),
		gitBtn("提交", *ui.Accent, len(g.staged) > 0, g.commit),
		ui.Expand(widget.Div(widget.Style{})),
		ui.ShellIconBtn("arrow-down-to-line", g.pull),
		ui.ShellIconBtn("arrow-up-from-line", g.push),
	)
}

// section 一段（标题 + 计数，可折叠）+ 文件行。
func (g *gitState) section(out *[]widget.Widget, title, key string, items []gitEntry, accent types.Color, stagedSec bool) {
	if len(items) == 0 {
		return
	}
	chev := "chevron-down"
	if g.collapsed[key] {
		chev = "chevron-right"
	}
	header := &widget.Clickable{
		SingleChildWidget: widget.SingleChildWidget{Child: widget.Div(
			widget.Style{Height: 24, Padding: types.EdgeInsetsLTRB(6, 0, 8, 0), FlexDirection: "row", AlignItems: "center"},
			widget.Lucide(chev, widget.IconSize(13), widget.IconColor(accent)),
			widget.Div(widget.Style{Width: 4}),
			ui.TextC(fmt.Sprintf("%s (%d)", title, len(items)), accent, 11),
		)},
		OnClick:    func() { g.toggleSection(key) },
		HoverColor: *ui.FtHover,
	}
	*out = append(*out, header)
	if g.collapsed[key] {
		return
	}
	for _, e := range items {
		*out = append(*out, g.fileRow(e, stagedSec))
	}
}

// fileRow 文件行：状态徽标 + 路径 + 动作按钮（暂存/取消暂存/丢弃）。
func (g *gitState) fileRow(e gitEntry, stagedSec bool) widget.Widget {
	st := e.x
	if !stagedSec {
		st = e.y
	}
	sym, col := badge(st, stagedSec)
	p := e.path
	// 行内动作按钮（hover 才显；非 hover 时用等宽空位占位，避免布局抖动）。
	var actions []widget.Widget
	switch {
	case stagedSec:
		actions = []widget.Widget{gitRowBtn("minus", "取消暂存", func() { g.unstageFile(p) })}
	case e.x == '?' && e.y == '?':
		actions = []widget.Widget{gitRowBtn("plus", "暂存", func() { g.stageFile(p) })}
	default:
		actions = []widget.Widget{
			gitRowBtn("plus", "暂存", func() { g.stageFile(p) }),
			gitRowBtn("trash-2", "丢弃", func() { g.discardFile(p) }),
		}
	}
	trailW := float64(len(actions)) * 22
	trailing := widget.Div(widget.Style{Width: trailW, FlexDirection: "row", AlignItems: "center", JustifyContent: "flex-end"})
	if g.hoveredPath == p {
		trailing = widget.Div(widget.Style{Width: trailW, FlexDirection: "row", AlignItems: "center", JustifyContent: "flex-end"}, actions)
	}
	return &widget.Clickable{
		SingleChildWidget: widget.SingleChildWidget{Child: widget.Div(
			widget.Style{Height: 24, Padding: types.EdgeInsetsLTRB(16, 0, 6, 0), FlexDirection: "row", AlignItems: "center"},
			widget.Div(widget.Style{Width: 14, FlexDirection: "row", AlignItems: "center"}, ui.TextC(sym, col, 12)),
			widget.Div(widget.Style{Width: 4}),
			ui.Expand(ui.TextC(shortGitPath(p), *ui.ShellText, 12)),
			trailing,
		)},
		OnClick:       func() { showGitDiff(p, stagedSec) }, // 点文件看差异（在编辑器打开走差异弹窗里的按钮）
		OnHoverChange: func(h bool) { g.setHovered(p, h) },
		HoverColor:    *ui.FtHover,
	}
}

// setHovered 记录/清除 hover 的文件行（仅变化时 SetState，避免无谓重排）。
func (g *gitState) setHovered(p string, h bool) {
	if h {
		if g.hoveredPath != p {
			g.hoveredPath = p
			g.SetState()
		}
	} else if g.hoveredPath == p {
		g.hoveredPath = ""
		g.SetState()
	}
}

// shortGitPath 取路径末段（目录前缀略灰，简化为只显文件名+父目录）。
func shortGitPath(p string) string {
	p = strings.ReplaceAll(p, "/", "\\")
	parts := strings.Split(p, "\\")
	if len(parts) <= 2 {
		return p
	}
	return ".../" + parts[len(parts)-2] + "/" + parts[len(parts)-1]
}

func openGitEntry(rel string) {
	abs := rel
	if theGit.root != "" {
		abs = theGit.root + "\\" + strings.ReplaceAll(rel, "/", "\\")
	}
	if line := firstChangedLine(theGit.root, rel); line > 0 {
		editorpanel.Editor.OpenAt(abs, line) // 跳到首个改动行
	} else {
		editorpanel.Editor.Open(abs)
	}
}

// showGitDiff 弹出文件的统一彩色差异（适配参考 DiffView：companion 无 Monaco → 统一格式 + 行着色）。
// staged=true 看已暂存的差异(--cached)，否则看工作区差异。
func showGitDiff(rel string, staged bool) {
	root := theGit.root
	args := []string{"diff"}
	if staged {
		args = append(args, "--cached")
	}
	args = append(args, "--", rel)
	out, err := runGit(root, args...)
	if err != nil {
		widget.ShowAlert("Git 出错", err.Error(), widget.MsgWarning, nil)
		return
	}
	if strings.TrimSpace(out) == "" {
		out = "（无文本差异：可能是新文件 / 二进制 / 仅模式变更）"
	}
	rows := make([]widget.Widget, 0, 64)
	for _, ln := range strings.Split(out, "\n") {
		rows = append(rows, diffLine(ln))
	}
	body := widget.Div(
		widget.Style{Width: 720, Height: 460},
		widget.NewScrollView(ui.FlexCol(rows...)),
	)
	var id int
	dlg := widget.NewDialog("差异 — "+rel, body).WithWidth(760).WithFooter(
		ui.Btn("在编辑器打开", func() { widget.HideOverlay(id); openGitEntry(rel) }),
		ui.PrimaryBtn("关闭", func() { widget.HideOverlay(id) }),
	)
	id = widget.ShowDialog(dlg)
}

// diffLine 一行统一 diff，按前缀着色：+绿 / -红 / @@蓝 / 文件头灰 / 上下文常色。
func diffLine(ln string) widget.Widget {
	col := *ui.ShellText
	switch {
	case strings.HasPrefix(ln, "+++"), strings.HasPrefix(ln, "---"), strings.HasPrefix(ln, "diff "), strings.HasPrefix(ln, "index "):
		col = *ui.ShellTextDim
	case strings.HasPrefix(ln, "@@"):
		col = *ui.Accent
	case strings.HasPrefix(ln, "+"):
		col = *ui.Success
	case strings.HasPrefix(ln, "-"):
		col = *ui.Danger
	}
	disp := strings.ReplaceAll(ln, "\t", "    ") // 等宽字体无 tab 字形会渲成□，展开为空格
	return widget.Div(
		widget.Style{Height: 16, Padding: types.EdgeInsetsLTRB(8, 0, 8, 0)},
		ui.Mono(disp, col, 12),
	)
}

// monoLabel 等宽文本已下沉到 ui.Mono（diff 对齐用）。

// firstChangedLine 取文件 git diff 首个 hunk 的新文件起始行（+N）；无改动/新文件返回 0（打开顶部）。
func firstChangedLine(root, rel string) int {
	out, err := runGit(root, "diff", "-U0", "HEAD", "--", rel)
	if err != nil || strings.TrimSpace(out) == "" {
		out, err = runGit(root, "diff", "-U0", "--cached", "--", rel) // 仅暂存的改动
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

// parseHunkNewLine 解析 hunk 头 "@@ -a,b +c,d @@" 的 c（新文件起始行）。解析失败返回 0。
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

// gitBtn 动作栏文本按钮（启用时着色，禁用置灰）。
func gitBtn(text string, color types.Color, enabled bool, onClick func()) widget.Widget {
	c := *ui.ShellTitle
	tc := *ui.ShellTextDim
	if enabled {
		c = color
		tc = *ui.White
	}
	b := &widget.Button{
		SingleChildWidget: widget.SingleChildWidget{Child: ui.TextC(text, tc, 11)},
		Color:             c,
		MinHeight:         24,
		Padding:           types.EdgeInsetsLTRB(10, 0, 10, 0),
	}
	if enabled {
		b.OnClick = onClick
	}
	return b
}

// gitRowBtn 行内小图标按钮。
func gitRowBtn(icon, _ string, onClick func()) widget.Widget {
	return &widget.Button{
		Icon: icon, IconSize: 13, TextColor: *ui.ShellTextDim,
		OnClick: onClick, Color: *ui.ShellSide, MinWidth: 22, MinHeight: 22,
	}
}

// gitMessage 居中空状态提示已下沉到 ui.EmptyState。
