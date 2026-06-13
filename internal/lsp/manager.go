package lsp

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// ServerSpec 声明如何启动一个语言服务器。
// Command 在 PATH 上解析（服务器从不捆绑）；InstallHint 在缺少时给出安装指引。
// Extensions 是此服务器处理的文件后缀（".go", ".rs"）——驱动文件→语言路由，
// 因此仅添加一项配置即可支持新语言而无需任何代码更改。
type ServerSpec struct {
	Command     string
	Args        []string
	Env         map[string]string
	LanguageID  string
	Extensions  []string
	InstallHint string
}

// Manager 管理一个会话中按需启动的语言服务器。
// 服务器在首次查询对应语言时启动并复用；生命周期由会话级 context 控制
// （由 Close 取消），而非单次对话。
type Manager struct {
	root     context.Context
	cancel   context.CancelFunc
	wsRoot   string
	specs    map[string]ServerSpec
	extIndex map[string]string // 文件扩展名 → 语言 key，由 specs 推导

	mu       sync.Mutex
	clients  map[string]*client
	starting map[string]chan struct{}
}

// NewManager 创建一个带默认语言服务器配置的管理器。
// specs 为 nil 时使用 DefaultSpecs()。
func NewManager(wsRoot string, specs map[string]ServerSpec) *Manager {
	if specs == nil {
		specs = DefaultSpecs()
	}
	root, cancel := context.WithCancel(context.Background())
	extIndex := map[string]string{}
	for lang, spec := range specs {
		for _, ext := range spec.Extensions {
			extIndex[strings.ToLower(ext)] = lang
		}
	}
	return &Manager{
		root:     root,
		cancel:   cancel,
		wsRoot:   wsRoot,
		specs:    specs,
		extIndex: extIndex,
		clients:  map[string]*client{},
		starting: map[string]chan struct{}{},
	}
}

// Close 关闭所有已启动的语言服务器。
func (m *Manager) Close() {
	m.mu.Lock()
	cs := make([]*client, 0, len(m.clients))
	for _, c := range m.clients {
		cs = append(cs, c)
	}
	m.clients = map[string]*client{}
	m.mu.Unlock()
	for _, c := range cs {
		c.close()
	}
	m.cancel()
}

// WsRoot 返回管理工作区根目录。
func (m *Manager) WsRoot() string { return m.wsRoot }

// DefaultSpecs 返回按语言 key 映射的常规服务器配置。
// 命令在 PATH 上尝试；没有任何服务器随本包发行。
// 扩展名驱动文件路由，因此用户可以从配置覆盖任何条目或添加全新语言。
func DefaultSpecs() map[string]ServerSpec {
	return map[string]ServerSpec{
		"go":         {Command: "gopls", LanguageID: "go", Extensions: []string{".go"}, InstallHint: "go install golang.org/x/tools/gopls@latest"},
		"rust":       {Command: "rust-analyzer", LanguageID: "rust", Extensions: []string{".rs"}, InstallHint: "rustup component add rust-analyzer"},
		"typescript": {Command: "typescript-language-server", Args: []string{"--stdio"}, LanguageID: "typescript", Extensions: []string{".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs"}, InstallHint: "npm i -g typescript-language-server typescript"},
		"python":     {Command: "pyright-langserver", Args: []string{"--stdio"}, LanguageID: "python", Extensions: []string{".py", ".pyi"}, InstallHint: "npm i -g pyright"},
		"cpp":        {Command: "clangd", LanguageID: "cpp", Extensions: []string{".c", ".h", ".cc", ".cpp", ".cxx", ".hpp", ".hh", ".hxx"}, InstallHint: "install clangd (LLVM): apt install clangd / brew install llvm / scoop install llvm"},
		"csharp":     {Command: "csharp-ls", LanguageID: "csharp", Extensions: []string{".cs"}, InstallHint: "dotnet tool install --global csharp-ls"},
		"java":       {Command: "jdtls", LanguageID: "java", Extensions: []string{".java"}, InstallHint: "install eclipse.jdt.ls (jdtls): brew install jdtls / from the JDT-LS releases"},
		"ruby":       {Command: "ruby-lsp", LanguageID: "ruby", Extensions: []string{".rb"}, InstallHint: "gem install ruby-lsp"},
		"php":        {Command: "intelephense", Args: []string{"--stdio"}, LanguageID: "php", Extensions: []string{".php"}, InstallHint: "npm i -g intelephense"},
		"lua":        {Command: "lua-language-server", LanguageID: "lua", Extensions: []string{".lua"}, InstallHint: "install lua-language-server: brew install lua-language-server / scoop install lua-language-server"},
		"bash":       {Command: "bash-language-server", Args: []string{"start"}, LanguageID: "shellscript", Extensions: []string{".sh", ".bash"}, InstallHint: "npm i -g bash-language-server"},
		"zig":        {Command: "zls", LanguageID: "zig", Extensions: []string{".zig"}, InstallHint: "install zls (ziglang/zls) matching your zig version"},
		"kotlin":     {Command: "kotlin-language-server", LanguageID: "kotlin", Extensions: []string{".kt", ".kts"}, InstallHint: "install kotlin-language-server: brew install kotlin-language-server"},
		"swift":      {Command: "sourcekit-lsp", LanguageID: "swift", Extensions: []string{".swift"}, InstallHint: "ships with the Swift toolchain (swift.org/download)"},
		"haskell":    {Command: "haskell-language-server-wrapper", Args: []string{"--lsp"}, LanguageID: "haskell", Extensions: []string{".hs"}, InstallHint: "install via ghcup: ghcup install hls"},
	}
}

// notInstalledError 携带安装提示，工具可以将缺失的能力告知模型。
type notInstalledError struct {
	command string
	hint    string
}

func (e *notInstalledError) Error() string {
	return fmt.Sprintf("language server %q not found on PATH. Install it: %s", e.command, e.hint)
}

func (m *Manager) abs(p string) string {
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	return filepath.Join(m.wsRoot, p)
}

// resolve 返回文件对应语言的运行中客户端，首次使用时自动启动。
// 并发首次调用通过 starting gate 共享一次启动，避免启动重复服务器。
func (m *Manager) resolve(path string) (*client, error) {
	lang := m.extIndex[strings.ToLower(filepath.Ext(path))]
	if lang == "" {
		return nil, fmt.Errorf("no language server configured for %s", filepath.Ext(path))
	}
	spec, ok := m.specs[lang]
	if !ok || spec.Command == "" {
		return nil, fmt.Errorf("no language server configured for %s files", lang)
	}

	m.mu.Lock()
	if c := m.clients[lang]; c != nil {
		m.mu.Unlock()
		return c, nil
	}
	if ch := m.starting[lang]; ch != nil {
		m.mu.Unlock()
		<-ch
		return m.resolve(path)
	}
	ch := make(chan struct{})
	m.starting[lang] = ch
	m.mu.Unlock()

	c, err := m.spawn(lang, spec)

	m.mu.Lock()
	delete(m.starting, lang)
	if err == nil {
		m.clients[lang] = c
	}
	close(ch)
	m.mu.Unlock()
	return c, err
}

func (m *Manager) spawn(_ string, spec ServerSpec) (*client, error) {
	bin, err := exec.LookPath(spec.Command)
	if err != nil {
		return nil, &notInstalledError{command: spec.Command, hint: spec.InstallHint}
	}
	return startClient(m.root, bin, spec.Args, spec.Env, spec.LanguageID, m.wsRoot)
}

func (m *Manager) prepare(ctx context.Context, file string, line int, symbol string) (*client, string, Position, error) {
	path := m.abs(file)
	c, err := m.resolve(path)
	if err != nil {
		return nil, "", Position{}, err
	}
	uri := pathToURI(path)
	if err := c.ensureSynced(uri, path); err != nil {
		return nil, "", Position{}, err
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, "", Position{}, err
	}
	pos, err := locate(string(content), line, symbol, c.posEnc)
	if err != nil {
		return nil, "", Position{}, err
	}
	return c, uri, pos, nil
}

// Definition 查找符号定义的位置。
func (m *Manager) Definition(ctx context.Context, file string, line int, symbol string) (string, error) {
	c, uri, pos, err := m.prepare(ctx, file, line, symbol)
	if err != nil {
		return "", err
	}
	raw, err := c.query(ctx, "textDocument/definition", uri, pos)
	if err != nil {
		return indexingOr(err)
	}
	return m.formatLocations("definition", parseLocations(raw)), nil
}

// indexingOr 将持续的 ContentModified 转为可重试的提示，其他错误正常返回。
func indexingOr(err error) (string, error) {
	if isContentModified(err) {
		return "the language server is still indexing this workspace — run the query again in a few seconds", nil
	}
	return "", err
}

// References 查找符号的所有引用位置。
func (m *Manager) References(ctx context.Context, file string, line int, symbol string) (string, error) {
	c, uri, pos, err := m.prepare(ctx, file, line, symbol)
	if err != nil {
		return "", err
	}
	raw, err := c.references(ctx, uri, pos)
	if err != nil {
		return indexingOr(err)
	}
	return m.formatLocations("reference", parseLocations(raw)), nil
}

// Hover 显示符号的类型签名和文档。
func (m *Manager) Hover(ctx context.Context, file string, line int, symbol string) (string, error) {
	c, uri, pos, err := m.prepare(ctx, file, line, symbol)
	if err != nil {
		return "", err
	}
	raw, err := c.query(ctx, "textDocument/hover", uri, pos)
	if err != nil {
		return indexingOr(err)
	}
	h := parseHover(raw)
	if h == "" {
		return "no hover information", nil
	}
	return h, nil
}

// Diagnostics 返回文件的编译器/检查器诊断信息。
func (m *Manager) Diagnostics(ctx context.Context, file string) (string, error) {
	path := m.abs(file)
	c, err := m.resolve(path)
	if err != nil {
		return "", err
	}
	uri := pathToURI(path)
	if err := c.ensureSynced(uri, path); err != nil {
		return "", err
	}
	diags := c.waitDiagnostics(ctx, uri, c.docVersion(uri), 2*time.Second)
	return formatDiagnostics(m.rel(path), diags), nil
}

func (m *Manager) rel(path string) string {
	if r, err := filepath.Rel(m.wsRoot, path); err == nil && !strings.HasPrefix(r, "..") {
		return filepath.ToSlash(r)
	}
	return path
}

func (m *Manager) formatLocations(kind string, locs []Location) string {
	if len(locs) == 0 {
		return "no " + kind + " found"
	}
	sort.Slice(locs, func(i, j int) bool {
		if locs[i].URI != locs[j].URI {
			return locs[i].URI < locs[j].URI
		}
		return locs[i].Range.Start.Line < locs[j].Range.Start.Line
	})
	var b strings.Builder
	fmt.Fprintf(&b, "%d %s(s):\n", len(locs), kind)
	for _, l := range locs {
		p := uriToPath(l.URI)
		line := l.Range.Start.Line + 1
		fmt.Fprintf(&b, "%s:%d", m.rel(p), line)
		if snippet := readLine(p, l.Range.Start.Line); snippet != "" {
			fmt.Fprintf(&b, "  %s", snippet)
		}
		b.WriteByte('\n')
	}
	return strings.TrimRight(b.String(), "\n")
}

func readLine(path string, line0 int) string {
	content, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	lines := strings.Split(string(content), "\n")
	if line0 < 0 || line0 >= len(lines) {
		return ""
	}
	return strings.TrimSpace(lines[line0])
}
