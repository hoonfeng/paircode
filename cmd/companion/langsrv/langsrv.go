// 语言服务器注册表：语言(扩展名去点) → LSP server。companion 编辑器据此为各语言接代码智能
// （补全/诊断/转到定义/查找引用/大纲/悬停）。服务器不在 PATH 则该语言纯词法降级，不报错。

//go:build windows

package langsrv

import (
	"os/exec"
	"strings"
	"sync"

	"github.com/hoonfeng/goui/pkg/widget"
)

// lspServerDef 一种语言的语言服务器配置。
type lspServerDef struct {
	cmd    string   // 可执行名
	args   []string // 启动参数（如 --stdio）
	langID string   // LSP languageId
}

// lspServers 各语言的默认语言服务器（需用户本机装好对应 server）。
var lspServers = map[string]lspServerDef{
	"go":     {"gopls", nil, "go"},
	"ts":     {"typescript-language-server", []string{"--stdio"}, "typescript"},
	"tsx":    {"typescript-language-server", []string{"--stdio"}, "typescriptreact"},
	"js":     {"typescript-language-server", []string{"--stdio"}, "javascript"},
	"jsx":    {"typescript-language-server", []string{"--stdio"}, "javascriptreact"},
	"mjs":    {"typescript-language-server", []string{"--stdio"}, "javascript"},
	"cjs":    {"typescript-language-server", []string{"--stdio"}, "javascript"},
	"py":     {"pyright-langserver", []string{"--stdio"}, "python"},
	"pyi":    {"pyright-langserver", []string{"--stdio"}, "python"},
	"rs":     {"rust-analyzer", nil, "rust"},
	"c":      {"clangd", nil, "c"},
	"cpp":    {"clangd", nil, "cpp"},
	"cc":     {"clangd", nil, "cpp"},
	"cxx":    {"clangd", nil, "cpp"},
	"h":      {"clangd", nil, "cpp"},
	"hpp":    {"clangd", nil, "cpp"},
	"java":   {"jdtls", nil, "java"},
	"rb":     {"solargraph", []string{"stdio"}, "ruby"},
	"php":    {"intelephense", []string{"--stdio"}, "php"},
	"lua":    {"lua-language-server", nil, "lua"},
	"json":   {"vscode-json-language-server", []string{"--stdio"}, "json"},
	"jsonc":  {"vscode-json-language-server", []string{"--stdio"}, "jsonc"},
	"css":    {"vscode-css-language-server", []string{"--stdio"}, "css"},
	"scss":   {"vscode-css-language-server", []string{"--stdio"}, "scss"},
	"less":   {"vscode-css-language-server", []string{"--stdio"}, "less"},
	"html":   {"vscode-html-language-server", []string{"--stdio"}, "html"},
	"yaml":   {"yaml-language-server", []string{"--stdio"}, "yaml"},
	"yml":    {"yaml-language-server", []string{"--stdio"}, "yaml"},
	"vue":    {"vue-language-server", []string{"--stdio"}, "vue"},
	"svelte": {"svelteserver", []string{"--stdio"}, "svelte"},
	"swift":  {"sourcekit-lsp", nil, "swift"},
	"kt":     {"kotlin-language-server", nil, "kotlin"},
	"zig":    {"zls", nil, "zig"},
	"dart":   {"dart", []string{"language-server"}, "dart"},
	"sh":     {"bash-language-server", []string{"start"}, "shellscript"},
	"bash":   {"bash-language-server", []string{"start"}, "shellscript"},
	"toml":   {"taplo", []string{"lsp", "stdio"}, "toml"},
	"tf":     {"terraform-ls", []string{"serve"}, "terraform"},
	"ex":     {"elixir-ls", nil, "elixir"},
	"exs":    {"elixir-ls", nil, "elixir"},
	"cs":     {"omnisharp", []string{"-lsp"}, "csharp"},
}

var (
	lspPathMu    sync.Mutex
	lspPathCache = map[string]bool{} // server cmd → 是否在 PATH（缓存，免每次开文件都 LookPath）
)

func serverAvailable(cmd string) bool {
	lspPathMu.Lock()
	defer lspPathMu.Unlock()
	if lspPathCache[cmd] { // 只缓存 true；false 不缓存——装上后重新探测即可发现
		return true
	}
	if _, err := exec.LookPath(cmd); err == nil {
		lspPathCache[cmd] = true
		return true
	}
	return false
}

// lspFor 某语言可用的语言服务器（注册表里有且在 PATH 才 ok；否则纯词法降级）。
func For(lang string) (cmd string, args []string, langID string, ok bool) {
	d, found := lspServers[lang]
	if !found || !serverAvailable(d.cmd) {
		return "", nil, "", false
	}
	return d.cmd, d.args, d.langID, true
}

// lspInstall 各服务器的安装命令（仅给有明确 CLI 安装方式的；空 cmd=需手动装，不弹）。
var lspInstall = map[string]struct {
	cmd  string
	args []string
}{
	"gopls":                       {"go", []string{"install", "golang.org/x/tools/gopls@latest"}},
	"typescript-language-server":  {"npm", []string{"install", "-g", "typescript-language-server", "typescript"}},
	"pyright-langserver":          {"npm", []string{"install", "-g", "pyright"}},
	"vscode-json-language-server": {"npm", []string{"install", "-g", "vscode-langservers-extracted"}},
	"vscode-css-language-server":  {"npm", []string{"install", "-g", "vscode-langservers-extracted"}},
	"vscode-html-language-server": {"npm", []string{"install", "-g", "vscode-langservers-extracted"}},
	"bash-language-server":        {"npm", []string{"install", "-g", "bash-language-server"}},
	"yaml-language-server":        {"npm", []string{"install", "-g", "yaml-language-server"}},
	"intelephense":                {"npm", []string{"install", "-g", "intelephense"}},
	"vue-language-server":         {"npm", []string{"install", "-g", "@vue/language-server"}},
	"svelteserver":                {"npm", []string{"install", "-g", "svelte-language-server"}},
	"rust-analyzer":               {"rustup", []string{"component", "add", "rust-analyzer"}},
}

var (
	lspPromptMu sync.Mutex
	lspPrompted = map[string]bool{} // 已提示过安装的服务器（不重复弹）
)

// maybeOfferInstall 某语言的服务器缺失、有安装方式、且没提示过 → 弹确认；确认则后台安装。开文件时调。
func MaybeOfferInstall(lang string) {
	d, ok := lspServers[lang]
	if !ok || serverAvailable(d.cmd) {
		return
	}
	inst, ok := lspInstall[d.cmd]
	if !ok || inst.cmd == "" {
		return
	}
	lspPromptMu.Lock()
	if lspPrompted[d.cmd] {
		lspPromptMu.Unlock()
		return
	}
	lspPrompted[d.cmd] = true
	lspPromptMu.Unlock()

	cmdline := inst.cmd + " " + strings.Join(inst.args, " ")
	widget.ShowConfirm("安装语言服务器",
		"未检测到 "+d.cmd+"，是否自动安装以获得该语言的代码智能？\n将运行：\n"+cmdline,
		widget.MsgInfo,
		func() { installLSPServer(d.cmd, inst.cmd, inst.args) },
		nil)
}

// installLSPServer 后台运行安装命令；overlay 有锁，从协程提示安全。完成后清缓存使重新探测发现。
func installLSPServer(server, cmd string, args []string) {
	widget.MessageInfo("正在安装 " + server + " …（后台运行，完成后重开该文件即生效）")
	go func() {
		out, err := exec.Command(cmd, args...).CombinedOutput()
		if err != nil {
			tail := strings.TrimSpace(string(out))
			if r := []rune(tail); len(r) > 160 {
				tail = "…" + string(r[len(r)-160:])
			}
			widget.MessageError(server + " 安装失败：" + tail)
		} else {
			lspPathMu.Lock()
			delete(lspPathCache, cmd)
			lspPathMu.Unlock()
			widget.MessageSuccess(server + " 已安装，重开该文件即生效")
		}
		if widget.OnNeedsRepaint != nil {
			widget.OnNeedsRepaint() // 唤醒 UI 渲染 toast
		}
	}()
}
