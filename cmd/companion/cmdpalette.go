// 命令面板（Ctrl+P 快速打开文件）—— VS Code 风格的文件快速跳转。
// 数据源：实时遍历 core.Folders 下文件（跳过 .git/node_modules 等）。
// 交互：输入过滤、上下键选择、Enter 打开、Esc/blur 关闭。
//
//go:build windows

package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/hoonfeng/gwui/component"
	"github.com/hoonfeng/gwui/dom"
	"github.com/hoonfeng/gwui/event"

	"github.com/hoonfeng/paircode/cmd/companion/core"
	editorpanel "github.com/hoonfeng/paircode/cmd/companion/ui/editor"
)

// 命令面板忽略的目录名（与搜索/探索工具一致，防止上下文爆炸）。
var cmdPaletteSkipDirs = map[string]bool{
	".git": true, "node_modules": true, ".vscode": true, ".idea": true,
	"dist": true, "build": true, "out": true, "bin": true, "obj": true,
	".next": true, ".nuxt": true, ".cache": true, "__pycache__": true,
	".pytest_cache": true, ".mypy_cache": true, "venv": true, ".venv": true,
	"target": true, ".gradle": true, ".mvn": true,
}

// 命令面板最大返回结果数（性能与可用性平衡）。
const cmdPaletteMaxResults = 50

// 命令面板最多扫描文件数（防止巨型仓库卡顿）。
const cmdPaletteMaxScan = 20000

// cmdPaletteFile 缓存的文件条目。
type cmdPaletteFile struct {
	path string // 绝对路径
	name string // 文件名（用于显示与匹配）
	rel  string // 相对工作区根的路径（用于显示）
}

var (
	cmdPaletteFiles   []cmdPaletteFile
	cmdPaletteLoaded  bool
	cmdPaletteMu      sync.Mutex
	cmdPaletteSelIdx  int
	cmdPaletteMatches []cmdPaletteFile
)

// loadCmdPaletteFiles 扫描工作区构建文件列表（带缓存）。
// 在 goroutine 中调用；扫描完成后由调用方触发 UI 刷新。
func loadCmdPaletteFiles() []cmdPaletteFile {
	cmdPaletteMu.Lock()
	defer cmdPaletteMu.Unlock()
	if cmdPaletteLoaded {
		return cmdPaletteFiles
	}
	var files []cmdPaletteFile
	scanCount := 0
	for _, root := range core.Folders {
		_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				name := d.Name()
				if path != root && cmdPaletteSkipDirs[name] {
					return filepath.SkipDir
				}
				return nil
			}
			scanCount++
			if scanCount > cmdPaletteMaxScan {
				return filepath.SkipDir
			}
			name := d.Name()
			rel, _ := filepath.Rel(root, path)
			files = append(files, cmdPaletteFile{path: path, name: name, rel: filepath.ToSlash(rel)})
			return nil
		})
	}
	// 按文件名排序（短路径优先），便于结果稳定
	sort.SliceStable(files, func(i, j int) bool {
		if len(files[i].rel) != len(files[j].rel) {
			return len(files[i].rel) < len(files[j].rel)
		}
		return files[i].rel < files[j].rel
	})
	cmdPaletteFiles = files
	cmdPaletteLoaded = true
	return files
}

// invalidateCmdPaletteFiles 工作区变更时调用，清空缓存以便下次重新扫描。
func invalidateCmdPaletteFiles() {
	cmdPaletteMu.Lock()
	cmdPaletteFiles = nil
	cmdPaletteLoaded = false
	cmdPaletteMu.Unlock()
}

// filterCmdPalette 根据查询过滤文件列表（支持模糊匹配：子串、首字母、路径段）。
func filterCmdPalette(query string, files []cmdPaletteFile) []cmdPaletteFile {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		// 空查询：返回前 N 项
		if len(files) > cmdPaletteMaxResults {
			return files[:cmdPaletteMaxResults]
		}
		return files
	}
	var matches []cmdPaletteFile
	for _, f := range files {
		if fuzzyMatch(q, strings.ToLower(f.name), strings.ToLower(f.rel)) {
			matches = append(matches, f)
			if len(matches) >= cmdPaletteMaxResults {
				break
			}
		}
	}
	return matches
}

// fuzzyMatch 模糊匹配：支持文件名子串、路径子串、驼峰首字母。
func fuzzyMatch(q, name, rel string) bool {
	if strings.Contains(name, q) || strings.Contains(rel, q) {
		return true
	}
	// 首字母匹配（如 "ft" 匹配 "file_tree.go"）
	qRunes := []rune(q)
	idx := 0
	for _, r := range name {
		if idx < len(qRunes) && r == qRunes[idx] {
			idx++
		}
	}
	return idx == len(qRunes)
}

// initCommandPalette 初始化命令面板的输入/键盘事件处理。
// HTML 中的 <input> 由 uixml factory 自动包装为 component.Input 组件，
// 因此通过 doc.ComponentAtNode 获取组件并使用 OnChange 回调。
// 注：Input 组件消费 Enter（返回 true），Enter 通过 OnChange 的"值未变"启发式检测。
func initCommandPalette(doc *dom.Document) {
	if theApp == nil {
		return
	}
	palette := doc.GetElementByID("cmd-palette")
	input := doc.GetElementByID("cmd-input")
	list := doc.GetElementByID("cmd-list")
	if palette == nil || input == nil || list == nil {
		return
	}

	// 点击遮罩关闭（仅点击遮罩本身，非内容）
	theApp.AddEventListener(palette, event.MouseDown, func(e event.Event) bool {
		if e.Target() == palette {
			closeCommandPalette()
			return true
		}
		return false
	})

	// 获取 Input 组件（HTML <input> 由 factory 创建为 component.Input）
	inp, _ := doc.ComponentAtNode(input).(*component.Input)

	// OnChange：文本变化时过滤；Enter 时（值未变）打开选中文件
	var lastValue string
	onChangeHandler := func(v string) {
		if v == lastValue {
			// Enter 被按下（onChange 触发但值未变）→ 打开选中文件
			if cmdPaletteSelIdx >= 0 && cmdPaletteSelIdx < len(cmdPaletteMatches) {
				f := cmdPaletteMatches[cmdPaletteSelIdx]
				closeCommandPalette()
				editorpanel.Editor.Open(f.path)
			}
			return
		}
		lastValue = v
		cmdPaletteMatches = filterCmdPalette(v, loadCmdPaletteFiles())
		cmdPaletteSelIdx = 0
		renderCmdPaletteList(doc, list)
		theApp.MarkDirty()
	}
	if inp != nil {
		inp.OnChange(onChangeHandler)
	}

	// 键盘导航：Esc/Up/Down（这些键不被 Input 组件消费，会传播到监听器）
	theApp.AddEventListener(input, event.KeyDown, func(e event.Event) bool {
		ke, ok := e.(*event.KeyboardEvent)
		if !ok {
			return false
		}
		switch ke.Key {
		case event.CodeEscape:
			closeCommandPalette()
			return true
		case event.CodeDown:
			cmdPaletteSelIdx++
			if cmdPaletteSelIdx >= len(cmdPaletteMatches) {
				cmdPaletteSelIdx = len(cmdPaletteMatches) - 1
			}
			renderCmdPaletteList(doc, list)
			theApp.MarkDirty()
			return true
		case event.CodeUp:
			cmdPaletteSelIdx--
			if cmdPaletteSelIdx < 0 {
				cmdPaletteSelIdx = 0
			}
			renderCmdPaletteList(doc, list)
			theApp.MarkDirty()
			return true
		}
		return false
	})
}

// openCommandPalette 打开命令面板，异步扫描文件列表并显示。
func openCommandPalette() {
	if theDoc == nil || theApp == nil {
		return
	}
	palette := theDoc.GetElementByID("cmd-palette")
	input := theDoc.GetElementByID("cmd-input")
	list := theDoc.GetElementByID("cmd-list")
	if palette == nil || input == nil || list == nil {
		return
	}
	palette.SetAttribute("style", "")
	// 通过 Input 组件清空文本
	if inp, ok := theDoc.ComponentAtNode(input).(*component.Input); ok && inp != nil {
		inp.SetValue("")
	}
	// 清空结果，显示加载提示
	list.ClearChildren()
	loading := theDoc.CreateElement("div")
	loading.SetAttribute("class", "cmd-empty")
	loading.SetTextContent("正在扫描文件…")
	list.AppendChild(loading)
	cmdPaletteSelIdx = 0
	cmdPaletteMatches = nil
	theApp.MarkDirty()

	// 异步扫描，避免首帧阻塞
	go func() {
		files := loadCmdPaletteFiles()
		// 回到主线程更新 UI
		go func() {
			time.Sleep(10 * time.Millisecond)
			if theDoc == nil || theApp == nil {
				return
			}
			cmdPaletteMatches = filterCmdPalette("", files)
			cmdPaletteSelIdx = 0
			renderCmdPaletteList(theDoc, list)
			theApp.MarkDirty()
		}()
	}()
}

// closeCommandPalette 关闭命令面板。
func closeCommandPalette() {
	if theDoc == nil {
		return
	}
	palette := theDoc.GetElementByID("cmd-palette")
	if palette != nil {
		palette.SetAttribute("style", "display:none;")
	}
	if theApp != nil {
		theApp.MarkDirty()
	}
}

// renderCmdPaletteList 渲染命令面板结果列表。
func renderCmdPaletteList(doc *dom.Document, list *dom.Element) {
	if list == nil {
		return
	}
	list.ClearChildren()
	if len(cmdPaletteMatches) == 0 {
		empty := doc.CreateElement("div")
		empty.SetAttribute("class", "cmd-empty")
		empty.SetTextContent("未找到匹配文件")
		list.AppendChild(empty)
		return
	}
	for i, f := range cmdPaletteMatches {
		f := f
		idx := i
		item := doc.CreateElement("div")
		if i == cmdPaletteSelIdx {
			item.SetAttribute("class", "cmd-item selected")
		} else {
			item.SetAttribute("class", "cmd-item")
		}
		// 图标
		iconName := fileIconForCmdPalette(f.name)
		icon := doc.CreateElement("span")
		icon.SetAttribute("data-icon", iconName)
		icon.SetAttribute("class", "cmd-icon")
		item.AppendChild(icon)
		// 路径文本
		path := doc.CreateElement("span")
		path.SetAttribute("class", "cmd-path")
		path.SetTextContent(f.rel)
		item.AppendChild(path)
		// 点击打开
		if theApp != nil {
			theApp.AddEventListener(item, event.Click, func(e event.Event) bool {
				editorpanel.Editor.Open(f.path)
				closeCommandPalette()
				return true
			})
			theApp.AddEventListener(item, event.MouseMove, func(e event.Event) bool {
				if cmdPaletteSelIdx != idx {
					cmdPaletteSelIdx = idx
					renderCmdPaletteList(doc, list)
					if theApp != nil {
						theApp.MarkDirty()
					}
				}
				return false
			})
		}
		list.AppendChild(item)
	}
}

// fileIconForCmdPalette 根据扩展名返回图标名。
func fileIconForCmdPalette(name string) string {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".go", ".js", ".ts", ".jsx", ".tsx", ".py", ".rs", ".java", ".c", ".cpp", ".h", ".hpp":
		return "file-code"
	case ".html", ".htm":
		return "file-code"
	case ".css", ".scss", ".less":
		return "file-code"
	case ".json":
		return "file-code"
	case ".md":
		return "file-text"
	case ".png", ".jpg", ".jpeg", ".gif", ".ico", ".svg":
		return "file-image"
	case ".zip", ".tar", ".gz", ".7z", ".rar":
		return "file-archive"
	default:
		return "file"
	}
}
