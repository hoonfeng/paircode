//go:build windows

package marketplacepanel

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/user/gou-ide/cmd/companion/core"
	"github.com/user/gou-ide/cmd/companion/ui/mcp"
	"github.com/user/gou-ide/cmd/companion/ui/skills"
	"github.com/user/gou-ide/cmd/companion/ui"
	"github.com/user/goui/internal/animation"
	"github.com/user/goui/internal/types"
	"github.com/user/goui/internal/widget"
)

// 扩展市场（复刻参考 MarketplacePanel）：模态对话框内 三 Tab(MCP/技能/已安装) + 搜索 + 作用域(项目/全局) + 卡片。

type Panel struct{ widget.StatefulWidget }

func (m *Panel) CreateState() widget.State { return TheMarket }

// OpenDialog 打开扩展市场模态对话框。
func OpenDialog() {
	TheMarket = &marketState{tab: "mcp", global: true}
	var id int
	dlg := widget.NewDialog("扩展市场", &Panel{}).WithWidth(720).WithTransition("fade").WithFooter(
		&widget.Button{Text: "关闭", OnClick: func() { widget.HideOverlay(id) },
			Color: *ui.BgMuted, TextColor: *ui.Fg, FontSize: 12,
			MinWidth: 50, MinHeight: 24, Padding: types.EdgeInsetsLTRB(12, 2, 13, 3)},
	)
	id = widget.ShowDialog(dlg)
}

var TheMarket *marketState

type marketState struct {
	widget.BaseState
	tab    string // "mcp" / "skills" / "installed"
	search string
	global bool // 作用域：true=全局(用户级 mcp.json / ~) false=项目级(.pair)

	// 实时搜索（MCP→npm registry / 技能→GitHub 仓库；真接源，非静态列表）。跨线程：协程写 pending，帧泵 drain。
	mu        sync.Mutex
	pending   *searchRes
	results   []marketResult // 已展示的搜索结果
	resultFor string         // results 对应的查询（空=显示精选静态列表）
	searching bool
	searchErr string
	pump      *animation.Controller
}

// marketResult 统一搜索结果：npm 包(github=false)或 GitHub 仓库技能(github=true)。
type marketResult struct {
	name, version, desc, url string
	github                   bool
}

type searchRes struct {
	query string
	pkgs  []marketResult
	err   string
}


// MarketMCP 市场 MCP 条目（精选注册表）。
type MarketMCP struct {
	ID, Name, Desc, Category, Command string
	Args                              []string
	Env                               []string
}

// MarketSkill 市场技能条目（精选注册表）。
type MarketSkill struct {
	ID, Name, Desc, Mode, Content string
}

var MCPMarket = []MarketMCP{
	{ID: "filesystem", Name: "Filesystem", Desc: "文件系统读写与搜索", Category: "存储", Command: "npx", Args: []string{"-y", "@modelcontextprotocol/server-filesystem"}},
	{ID: "git", Name: "Git", Desc: "Git 仓库分析与管理", Category: "开发", Command: "uvx", Args: []string{"mcp-server-git"}},
	{ID: "fetch", Name: "Fetch", Desc: "HTTP 抓取网页内容", Category: "网络", Command: "uvx", Args: []string{"mcp-server-fetch"}},
	{ID: "memory", Name: "Memory", Desc: "持久化知识图谱", Category: "存储", Command: "npx", Args: []string{"-y", "@modelcontextprotocol/server-memory"}},
	{ID: "playwright", Name: "Playwright", Desc: "浏览器自动化", Category: "测试", Command: "npx", Args: []string{"-y", "@playwright/mcp"}},
}

var SkillMarket = []MarketSkill{
	{ID: "code-review", Name: "代码审查规范", Desc: "拉取代码变更、逐文件审查、汇总问题与建议", Mode: "auto", Content: "审查代码时关注：安全性、性能、可维护性……"},
	{ID: "commit-message", Name: "提交信息规范", Desc: "按 Conventional Commits 规范化提交信息", Mode: "auto", Content: "按 type(scope): description 格式编写……"},
}

// npmSearch 查 npm registry（MCP 服务器多为 npm 包）。
func npmSearch(query string) ([]marketResult, error) {
	u := "https://registry.npmjs.org/-/v1/search?size=25&text=" + url.QueryEscape(query)
	resp, err := (&http.Client{Timeout: 12 * time.Second}).Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var data struct {
		Objects []struct {
			Package struct{ Name, Version, Description string } `json:"package"`
		} `json:"objects"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	out := make([]marketResult, 0, len(data.Objects))
	for _, o := range data.Objects {
		out = append(out, marketResult{name: o.Package.Name, version: o.Package.Version, desc: o.Package.Description})
	}
	return out, nil
}

// githubSearch 查 GitHub 仓库（技能，按 star 排），复刻参考 marketplace:search-github。
func githubSearch(query string) ([]marketResult, error) {
	u := "https://api.github.com/search/repositories?per_page=15&sort=stars&q=" + url.QueryEscape(query)
	req, _ := http.NewRequest("GET", u, nil)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	resp, err := (&http.Client{Timeout: 12 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var data struct {
		Items []struct {
			FullName    string `json:"full_name"`
			Description string `json:"description"`
			Stars       int    `json:"stargazers_count"`
			HTMLURL     string `json:"html_url"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	out := make([]marketResult, 0, len(data.Items))
	for _, it := range data.Items {
		out = append(out, marketResult{name: it.FullName, version: fmt.Sprintf("%d ★", it.Stars), desc: it.Description, url: it.HTMLURL, github: true})
	}
	return out, nil
}

// doSearch 触发联网搜索：MCP→npm、技能→GitHub（异步，帧泵刷 UI）；已安装 tab 仅重过滤本地。
func (s *marketState) doSearch() {
	q := strings.TrimSpace(s.search)
	if s.tab == "installed" {
		s.SetState()
		return
	}
	if q == "" { // 清空→回精选
		s.results, s.resultFor, s.searchErr = nil, "", ""
		s.SetState()
		return
	}
	s.searching = true
	if s.pump == nil {
		p := animation.NewController(time.Second, animation.Linear)
		p.Repeat = true
		p.OnUpdate = func(float64) { s.drain() }
		s.pump = p
		p.Start()
	}
	s.SetState()
	tab := s.tab
	go func() {
		var pkgs []marketResult
		var err error
		if tab == "skills" {
			pkgs, err = githubSearch(q)
		} else {
			pkgs, err = npmSearch(q)
		}
		es := ""
		if err != nil {
			es = err.Error()
		}
		s.mu.Lock()
		s.pending = &searchRes{query: q, pkgs: pkgs, err: es}
		s.mu.Unlock()
	}()
}

// drain 帧泵在 UI 线程调：把协程拿到的搜索结果应用到状态并停泵。
func (s *marketState) drain() {
	s.mu.Lock()
	p := s.pending
	s.pending = nil
	s.mu.Unlock()
	if p == nil {
		return
	}
	s.results, s.resultFor, s.searchErr, s.searching = p.pkgs, p.query, p.err, false
	if s.pump != nil {
		s.pump.Stop()
		s.pump = nil
	}
	s.SetState()
}

// installNpm 把一个 npm 包装成 MCP 服务器（npx -y <包名>）写进作用域配置。
func (s *marketState) installNpm(pkg string) {
	level := mcppanel.LevelProject
	if s.global {
		level = mcppanel.LevelUser
	}
	if err := mcppanel.Upsert(level, mcppanel.Entry{Name: pkg, Command: "npx", Args: []string{"-y", pkg}}); err != nil {
		widget.MessageError("安装失败：" + err.Error())
	} else {
		widget.MessageSuccess("已安装 " + pkg)
		s.SetState()
	}
}

func (s *marketState) Build(ctx widget.BuildContext) widget.Widget {
	return widget.Div(
		widget.Style{Height: 470, FlexDirection: "column", AlignItems: "stretch"},
		s.tabsRow(),
		s.searchRow(),
		s.scopeRow(),
		ui.Expand(widget.NewScrollView(widget.Div(
			widget.Style{FlexDirection: "column", AlignItems: "stretch", Gap: 6, Padding: types.EdgeInsetsLTRB(4, 8, 4, 8)},
			s.content()))),
	)
}

// tabsRow 三 Tab，激活态下方强调色下划线。
func (s *marketState) tabsRow() widget.Widget {
	tab := func(id, name, icon string) widget.Widget {
		tc := *ui.FgMuted
		var bg, underline *types.Color
		if s.tab == id {
			tc, bg, underline = *ui.Fg, ui.BgMuted, ui.AccentStrong
		}
		return &widget.Clickable{
			SingleChildWidget: widget.SingleChildWidget{Child: widget.Div(
				widget.Style{FlexDirection: "column", AlignItems: "stretch"},
				widget.Div(widget.Style{BackgroundColor: bg, FlexDirection: "row", AlignItems: "center", Gap: 5,
					Padding: types.EdgeInsetsLTRB(14, 8, 14, 7)},
					widget.Lucide(icon, widget.IconSize(12), widget.IconColor(tc)),
					ui.TextC(name, tc, 12)),
				widget.Div(widget.Style{Height: 2, BackgroundColor: underline}))},
			OnClick: func() { // 切 tab 清搜索与结果，各 tab 搜索独立
				s.tab, s.search, s.results, s.resultFor, s.searchErr = id, "", nil, "", ""
				s.SetState()
			},
		}
	}
	return widget.Div(widget.Style{FlexDirection: "row", AlignItems: "stretch", BorderColor: ui.Border, BorderWidth: 1},
		tab("mcp", "MCP", "puzzle"),
		tab("skills", "技能", "book-open"),
		tab("installed", "已安装", "check"),
	)
}

// searchRow 搜索框 + 搜索按钮。MCP tab 走真 npm 搜索（按钮触发）；技能/已安装本地过滤。
func (s *marketState) searchRow() widget.Widget {
	ph := map[string]string{"mcp": "搜索 npm 上的 MCP 服务器（回车/点搜索）…", "skills": "搜索 GitHub 技能仓库（回车/点搜索）…", "installed": "搜索已安装…"}[s.tab]
	in := widget.NewInput(ph, func(t string) {
		s.search = t
		if s.tab == "installed" { // 已安装本地实时过滤；MCP/技能联网，点「搜索」再发请求
			s.SetState()
		}
	})
	in.Text = s.search
	in.OnSubmit = func(string) { s.doSearch() } // 回车即搜
	in.Color = *ui.Fg
	in.PlaceholderColor = *ui.FgMuted
	in.CursorColor = *ui.Fg
	in.BGColor = *ui.BgMuted
	in.BorderColor = *ui.Border
	in.FocusBorderColor = *ui.Accent
	in.HoverBorderColor = *ui.Border
	row := []widget.Widget{ui.Expand(in)}
	if s.tab != "installed" { // MCP/技能有联网搜索按钮
		row = append(row, widget.Div(widget.Style{Width: 6}), pillBtn("搜索", *ui.AccentStrong, *ui.White, s.doSearch))
	}
	return widget.Div(widget.Style{Padding: types.EdgeInsetsLTRB(8, 8, 8, 4), FlexDirection: "row", AlignItems: "center"}, row)
}

// scopeRow 安装作用域：项目 / 全局 + 说明。
func (s *marketState) scopeRow() widget.Widget {
	btn := func(g bool, icon, name string) widget.Widget {
		bg, tc := ui.BgMuted, *ui.FgMuted
		if s.global == g {
			bg, tc = ui.AccentStrong, types.ColorFromRGB(255, 255, 255)
		}
		return &widget.Clickable{
			SingleChildWidget: widget.SingleChildWidget{Child: widget.Div(
				widget.Style{BackgroundColor: bg, BorderRadius: 4, Padding: types.EdgeInsetsLTRB(9, 3, 9, 3),
					FlexDirection: "row", AlignItems: "center", Gap: 4},
				widget.Lucide(icon, widget.IconSize(11), widget.IconColor(tc)),
				ui.TextC(name, tc, 11))},
			OnClick: func() { s.global = g; s.SetState() },
		}
	}
	help := "~/.pair · 所有项目可用"
	if !s.global {
		help = filepath.Base(core.Root()) + "/.pair · 仅当前项目"
	}
	return widget.Div(widget.Style{Padding: types.EdgeInsetsLTRB(8, 2, 8, 6), FlexDirection: "row", AlignItems: "center", Gap: 7},
		ui.TextC("安装到:", *ui.FgMuted, 11),
		btn(false, "folder-git-2", "项目"),
		btn(true, "globe", "全局"),
		widget.Div(widget.Style{Width: 4}),
		ui.TextC(help, *ui.FgMuted, 10),
	)
}

func (s *marketState) match(parts ...string) bool {
	if s.search == "" {
		return true
	}
	q := strings.ToLower(strings.TrimSpace(s.search))
	for _, p := range parts {
		if strings.Contains(strings.ToLower(p), q) {
			return true
		}
	}
	return false
}

func (s *marketState) content() widget.Widget {
	switch s.tab {
	case "installed":
		return s.installedContent()
	case "skills":
		if s.searching || s.resultFor != "" {
			return s.searchResults("GitHub")
		}
		return s.skillsFeatured()
	default:
		if s.searching || s.resultFor != "" {
			return s.searchResults("npm registry")
		}
		return s.mcpFeatured()
	}
}

// searchResults 联网搜索结果（MCP/技能共用）：加载中/错误/卡片；安装按结果类型分派。
func (s *marketState) searchResults(source string) widget.Widget {
	if s.searching {
		return widget.Div(widget.Style{Padding: types.EdgeInsetsLTRB(4, 22, 4, 4), FlexDirection: "row", AlignItems: "center", Gap: 6},
			widget.Lucide("refresh-cw", widget.IconSize(13), widget.IconColor(*ui.FgMuted)),
			ui.TextC("正在搜索 "+source+"…", *ui.FgMuted, 12))
	}
	if s.searchErr != "" {
		return cardList(nil, "搜索失败："+s.searchErr)
	}
	mcpHave, skHave := mcpInstalled(), skillInstalled()
	var kids []widget.Widget
	for _, r := range s.results {
		rr := r
		if rr.github {
			kids = append(kids, marketCard("book-open", rr.name, rr.version, rr.desc, skHave[skillNameFromRepo(rr.name)],
				func() { s.installGhSkill(rr.name, rr.desc, rr.url) }))
		} else {
			kids = append(kids, marketCard("puzzle", rr.name, rr.version, rr.desc, mcpHave[rr.name], func() { s.installNpm(rr.name) }))
		}
	}
	head := ui.TextC(fmt.Sprintf("%s · %d 个结果 · “%s”", source, len(s.results), s.resultFor), *ui.FgMuted, 10)
	return widget.Div(widget.Style{FlexDirection: "column", AlignItems: "stretch", Gap: 6},
		widget.Div(widget.Style{Padding: types.EdgeInsetsLTRB(2, 0, 2, 2)}, head),
		cardList(kids, source+" 上没有匹配结果，换个关键词试试"))
}

func (s *marketState) mcpFeatured() widget.Widget {
	have := mcpInstalled()
	var kids []widget.Widget
	for _, m := range MCPMarket {
		if !s.match(m.Name, m.Desc, m.Category) {
			continue
		}
		mm := m
		sub := mm.Desc
		if len(mm.Env) > 0 {
			sub += "（需配 " + strings.Join(mm.Env, "/") + "）"
		}
		kids = append(kids, marketCard("puzzle", mm.Name, mm.Category, sub, have[mm.ID], func() { s.install(mm.ID, mm.Name) }))
	}
	return featuredList("精选 MCP（输入关键词搜 npm 全库）", kids)
}

func (s *marketState) skillsFeatured() widget.Widget {
	have := skillInstalled()
	var kids []widget.Widget
	for _, sk := range SkillMarket {
		if !s.match(sk.Name, sk.Desc) {
			continue
		}
		ss := sk
		kids = append(kids, marketCard("book-open", ss.Name, ss.Mode, ss.Desc, have[ss.ID], func() { s.install(ss.ID, ss.Name) }))
	}
	return featuredList("精选技能（输入关键词搜 GitHub 仓库）", kids)
}

func featuredList(title string, kids []widget.Widget) widget.Widget {
	return widget.Div(widget.Style{FlexDirection: "column", AlignItems: "stretch", Gap: 6},
		widget.Div(widget.Style{Padding: types.EdgeInsetsLTRB(2, 0, 2, 4)}, ui.TextC(title, *ui.FgMuted, 10)),
		cardList(kids, "没有匹配的精选项"))
}

func mcpInstalled() map[string]bool {
	set := map[string]bool{}
	for _, lv := range []string{mcppanel.LevelUser, mcppanel.LevelProject} {
		for _, e := range mcppanel.ReadLevel(lv) {
			set[e.Name] = true
		}
	}
	return set
}

func skillInstalled() map[string]bool {
	set := map[string]bool{}
	for _, lv := range []string{mcppanel.LevelUser, mcppanel.LevelProject} {
		for _, e := range skillspanel.ReadLevel(lv) {
			set[e.Name] = true
		}
	}
	return set
}

func skillNameFromRepo(full string) string {
	if i := strings.LastIndex(full, "/"); i >= 0 {
		return full[i+1:]
	}
	return full
}

// installGhSkill 把一个 GitHub 仓库装成技能（引用仓库）写进作用域。
func (s *marketState) installGhSkill(repo, desc, repoURL string) {
	level := mcppanel.LevelProject
	if s.global {
		level = mcppanel.LevelUser
	}
	name := skillNameFromRepo(repo)
	content := "# " + name + "\n\n来源仓库：" + repoURL + "\n\n" + desc + "\n\n按该仓库的约定与用法工作（按需查阅其 README/SKILL.md）。"
	if err := skillspanel.Write(level, skillspanel.Entry{Name: name, Description: desc, Mode: "manual", Content: content}); err != nil {
		widget.MessageError("安装失败：" + err.Error())
	} else {
		widget.MessageSuccess("已安装技能 " + name)
		s.SetState()
	}
}

// installedContent 已安装：读用户/项目两级的 MCP 与技能，可移除。
func (s *marketState) installedContent() widget.Widget {
	var kids []widget.Widget
	for _, lv := range []string{mcppanel.LevelUser, mcppanel.LevelProject} {
		for _, e := range mcppanel.ReadLevel(lv) {
			if !s.match(e.Name, e.Command) {
				continue
			}
			name, level := e.Name, lv
			kids = append(kids, installedCard("puzzle", name, levelLabel(level), e.Command+" "+strings.Join(e.Args, " "),
				func() { _ = mcppanel.Delete(level, name); s.SetState() }))
		}
	}
	for _, lv := range []string{mcppanel.LevelUser, mcppanel.LevelProject} {
		for _, e := range skillspanel.ReadLevel(lv) {
			if !s.match(e.Name, e.Description) {
				continue
			}
			name, level := e.Name, lv
			kids = append(kids, installedCard("book-open", name, levelLabel(level), e.Description,
				func() { _ = skillspanel.Delete(level, name); s.SetState() }))
		}
	}
	return cardList(kids, "还没有安装任何扩展")
}

func (s *marketState) install(id, name string) {
	if _, err := InstallScoped(id, s.global); err != nil {
		widget.MessageError("安装失败：" + err.Error())
	} else {
		widget.MessageSuccess("已安装 " + name)
		s.SetState()
	}
}

// ── 通用卡片 / 工具 ──

func cardList(kids []widget.Widget, empty string) widget.Widget {
	if len(kids) == 0 {
		kids = []widget.Widget{widget.Div(widget.Style{Padding: types.EdgeInsetsLTRB(4, 20, 4, 4)}, ui.TextC(empty, *ui.FgMuted, 11))}
	}
	return widget.Div(widget.Style{FlexDirection: "column", AlignItems: "stretch", Gap: 6}, kids)
}

func marketCard(icon, name, badge, desc string, installed bool, onInstall func()) widget.Widget {
	ic := *ui.Accent
	if installed {
		ic = *ui.Success
	}
	var right widget.Widget
	if installed {
		right = widget.Div(widget.Style{FlexDirection: "row", AlignItems: "center", Gap: 3, BackgroundColor: ui.BgMuted,
			BorderRadius: 6, Padding: types.EdgeInsetsLTRB(10, 5, 12, 5)},
			widget.Lucide("check", widget.IconSize(12), widget.IconColor(*ui.Success)),
			ui.TextC("已安装", *ui.Success, 11))
	} else {
		right = pillBtn("安装", *ui.AccentStrong, *ui.White, onInstall)
	}
	return cardShell(icon, ic, name, badge, desc, right)
}

func installedCard(icon, name, badge, desc string, onRemove func()) widget.Widget {
	return cardShell(icon, *ui.Success, name, badge, desc, pillBtn("移除", *ui.BgMuted, *ui.Danger, onRemove))
}

// pillBtn 紧凑胶囊按钮（市场卡片安装/移除用）。
func pillBtn(text string, bg, tc types.Color, onClick func()) widget.Widget {
	return &widget.Button{
		SingleChildWidget: widget.SingleChildWidget{Child: ui.TextC(text, tc, 11)},
		OnClick:           onClick,
		Color:             bg,
		TextColor:         tc,
		MinHeight:         28,
		Padding:           types.EdgeInsetsLTRB(16, 0, 16, 0),
	}
}

func cardShell(icon string, iconColor types.Color, name, badge, desc string, right widget.Widget) widget.Widget {
	// 名字 expand 占满（宽度足够→Text 不被测量误差截末字），类别徽标靠右当标签（参考亦右上角标签）。
	head := []widget.Widget{ui.Expand(ui.TextC(name, *ui.Fg, 12))}
	if badge != "" {
		head = append(head, catBadge(badge))
	}
	return widget.Div(
		widget.Style{BackgroundColor: ui.BgSubtle, BorderColor: ui.Border, BorderWidth: 1, BorderRadius: 7,
			Padding: types.EdgeInsetsLTRB(11, 9, 11, 9), FlexDirection: "row", AlignItems: "center", Gap: 11},
		widget.Lucide(icon, widget.IconSize(18), widget.IconColor(iconColor)),
		ui.Expand(widget.Div(widget.Style{FlexDirection: "column", AlignItems: "stretch", Gap: 3},
			widget.Div(widget.Style{FlexDirection: "row", AlignItems: "center", Gap: 6}, head),
			ui.TextC(desc, *ui.FgMuted, 10))),
		right,
	)
}

func catBadge(text string) widget.Widget {
	// 尾随空格吃掉文本测量误差（末字被截那 1 字落到空格上），徽标文字完整。
	return widget.Div(widget.Style{BackgroundColor: ui.BgMuted, BorderRadius: 3, Padding: types.EdgeInsetsLTRB(6, 1, 7, 1)},
		ui.TextC(text+" ", *ui.FgMuted, 9))
}

func levelLabel(level string) string {
	if level == mcppanel.LevelUser {
		return "全局"
	}
	return "项目"
}

// InstallScoped 按作用域安装：global→用户级，否则项目级（写 mcp.json / .pair/skills）。
func InstallScoped(id string, global bool) (string, error) {
	level := mcppanel.LevelProject
	if global {
		level = mcppanel.LevelUser
	}
	for _, m := range MCPMarket {
		if m.ID == id {
			e := mcppanel.Entry{Name: m.ID, Command: m.Command, Args: m.Args}
			if len(m.Env) > 0 {
				e.Env = map[string]string{}
				for _, k := range m.Env {
					e.Env[k] = ""
				}
			}
			return "已安装 MCP " + m.Name, mcppanel.Upsert(level, e)
		}
	}
	for _, sk := range SkillMarket {
		if sk.ID == id {
			return "已安装技能 " + sk.Name, skillspanel.Write(level, skillspanel.Entry{Name: sk.ID, Description: sk.Desc, Mode: sk.Mode, Content: sk.Content})
		}
	}
	return "", fmt.Errorf("市场无此 id：%q", id)
}
