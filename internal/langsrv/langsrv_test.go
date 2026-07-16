//go:build windows

package langsrv

import (
	"os/exec"
	"testing"
)

// TestLspFor 语言服务器注册表解析：未注册→无 LSP；注册且在 PATH→解析；带参数项正确；结果缓存。
func TestLspFor(t *testing.T) {
	if _, _, _, ok := For("zzz-unknown-ext"); ok {
		t.Error("未注册语言不应有 LSP")
	}
	if _, err := exec.LookPath("gopls"); err == nil {
		cmd, _, langID, ok := For("go")
		if !ok || cmd != "gopls" || langID != "go" {
			t.Errorf("go 应解析到 gopls/go，得 cmd=%s langID=%s ok=%v", cmd, langID, ok)
		}
	}
	// 带 --stdio 的项（不依赖 server 是否安装，只查注册表内容）
	if d, ok := lspServers["ts"]; !ok || d.cmd != "typescript-language-server" || len(d.args) == 0 || d.langID != "typescript" {
		t.Error("ts 注册表项应为 typescript-language-server --stdio / typescript")
	}
	if d := lspServers["py"]; d.langID != "python" {
		t.Error("py 的 langID 应为 python")
	}
	if _, _, _, ok := For("zzz-unknown-ext"); ok {
		t.Error("缓存后仍应 !ok")
	}
}

// TestLspInstall 安装映射：常见服务器有明确安装命令。
func TestLspInstall(t *testing.T) {
	if d := lspInstall["gopls"]; d.cmd != "go" {
		t.Errorf("gopls 应 go install，得 %q", d.cmd)
	}
	if d, ok := lspInstall["typescript-language-server"]; !ok || d.cmd != "npm" {
		t.Error("tsserver 应 npm install")
	}
	if d, ok := lspInstall["rust-analyzer"]; !ok || d.cmd != "rustup" {
		t.Error("rust-analyzer 应 rustup component add")
	}
}
