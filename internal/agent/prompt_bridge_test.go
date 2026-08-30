// prompt_bridge_test.go —— ctx.prompts 桥验证（JS 插件运行时注册提示词资产）。
package agent

import (
	"os"
	"strings"
	"testing"
)

// TestJSPromptBridge ctx.prompts.provide/remove 端到端：JS 插件运行时注册
// 提示词资产 → LoadPrompt 可见；插件卸载 → 资产按来源清理。
func TestJSPromptBridge(t *testing.T) {
	ResetPromptAssetsForTest()
	defer ResetPromptAssetsForTest()
	code := `
return {
  name: 'prompt-demo',
  apply(ctx) {
    ctx.prompts.provide({ name: 'reviewer', text: 'js-bridge:reviewer' });
    ctx.prompts.provide({ name: 'demo-note', text: 'js-bridge:demo-note' });
  }
}`
	host, _ := loadJSCodeForTest(t, code)
	if got := LoadPrompt("reviewer"); got != "js-bridge:reviewer" {
		t.Fatalf("ctx.prompts.provide 未生效：got %q", got)
	}
	if got := LoadPrompt("demo-note"); got != "js-bridge:demo-note" {
		t.Fatalf("ctx.prompts.provide 未生效：got %q", got)
	}
	// 插件卸载 → 资产清理（LoadPrompt 回落）
	if err := host.Unload("prompt-demo"); err != nil {
		t.Fatalf("Unload: %v", err)
	}
	if got := LoadPrompt("reviewer"); got != "" {
		t.Fatalf("插件卸载后资产应清理：got %q", got)
	}
}

// TestJSPromptBridgeConfig 磁盘插件包 prompts/ 目录（插件内置形态）端到端：
// 造包 → define+load → LoadPrompt 可见（磁盘资产注册）。
func TestJSPromptBridgeDiskAssets(t *testing.T) {
	ResetPromptAssetsForTest()
	defer ResetPromptAssetsForTest()
	pkgDir := t.TempDir() + "/asset-demo"
	if err := os.MkdirAll(pkgDir+"/prompts", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pkgDir+"/prompts/planner.md", []byte("disk:planner-prompt"), 0o644); err != nil {
		t.Fatal(err)
	}
	if n := ScanPluginPromptAssets(pkgDir, "asset-demo"); n != 1 {
		t.Fatalf("应注册 1 个资产，got %d", n)
	}
	if got := LoadPrompt("planner"); got != "disk:planner-prompt" {
		t.Fatalf("磁盘插件资产未生效：got %q", got)
	}
}

// TestJSPromptBridgeInterop 运行时注册与磁盘资产共存：运行时优先。
func TestJSPromptBridgeInterop(t *testing.T) {
	ResetPromptAssetsForTest()
	defer ResetPromptAssetsForTest()
	pkgDir := t.TempDir() + "/asset-demo2"
	if err := os.MkdirAll(pkgDir+"/prompts", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pkgDir+"/prompts/judge.md", []byte("disk:judge"), 0o644); err != nil {
		t.Fatal(err)
	}
	ScanPluginPromptAssets(pkgDir, "asset-demo2")
	ProvidePrompt("judge", "runtime:judge", "js:over")
	if got := LoadPrompt("judge"); got != "runtime:judge" {
		t.Fatalf("运行时注册应优先：got %q", got)
	}
	RemovePrompt("judge")
	if got := LoadPrompt("judge"); !strings.Contains(got, "disk:judge") {
		t.Fatalf("移除后回落磁盘资产：got %q", got)
	}
}
