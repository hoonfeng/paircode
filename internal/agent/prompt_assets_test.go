// prompt_assets_test.go —— 提示词插件化（Prompt Assets）验证
//
// 覆盖：
//   - 同步守卫：插件包 prompts/ 默认资产（agentloop）与 Go 内置提示逐字节一致
//     （磁盘资产在生产静默生效，漂移会导致线上提示≠内置且无人察觉）
//   - 优先级：运行时注册 > 磁盘插件资产 > config/roles 旧式覆盖
//   - 插件配置 config.prompts 注册（插件+插件配置形态）
//   - 系统提示模板资产（{{ROOT_INFO}}/{{PRIMARY_ROOT}} 插值，未知变量原样保留）
//   - 卸载清理语义（RemovePromptSource）
package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// repoRootAbs 仓库根（测试从 internal/agent 包目录调用）。
func repoRootAbs(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

// TestPluginPromptAssetsSync 漂移守卫：agentloop 插件包 prompts/ 默认资产与
// Go 内置提示完全一致（含系统提示模板：{{ROOT_INFO}} 展开后 == 内置）。
func TestPluginPromptAssetsSync(t *testing.T) {
	ResetPromptAssetsForTest()
	defer ResetPromptAssetsForTest()
	promptsDir := filepath.Join(repoRootAbs(t), ".pair", "plugins", "agentloop", "prompts")

	// 角色资产：文本逐字节一致
	roleCases := []struct {
		file    string
		builtin string
		name    string
	}{
		{"reviewer.md", reviewerSystemPrompt, "reviewer"},
		{"planner.md", plannerSystemPrompt, "planner"},
		{"judge.md", judgeSystemPrompt, "judge"},
	}
	for _, c := range roleCases {
		data, err := os.ReadFile(filepath.Join(promptsDir, c.file))
		if err != nil {
			t.Errorf("agentloop/prompts/%s 缺失（提示词插件化资产不得删除）: %v", c.file, err)
			continue
		}
		if disk, builtin := strings.TrimSpace(string(data)), strings.TrimSpace(c.builtin); disk != builtin {
			t.Errorf("agentloop/prompts/%s 与内置 %sSystemPrompt 漂移（磁盘资产生产静默生效）——请同步两者\n  disk    len=%d\n  builtin len=%d",
				c.file, c.name, len(disk), len(builtin))
		}
	}

	// 系统提示资产：模板展开（{{ROOT_INFO}} 替换）后 == 内置全文
	roots := []string{filepath.Join("F:", "workspace-demo")}
	_, rootInfo := workspaceRoots(roots)
	sysCases := []struct {
		file    string
		builtin string
	}{
		{"system-harness.md", harnessSystemPrompt(roots)},
		{"system-full.md", fullSystemPrompt(roots)},
	}
	for _, c := range sysCases {
		data, err := os.ReadFile(filepath.Join(promptsDir, c.file))
		if err != nil {
			t.Errorf("agentloop/prompts/%s 缺失: %v", c.file, err)
			continue
		}
		expanded := strings.ReplaceAll(strings.TrimSpace(string(data)), "{{ROOT_INFO}}", rootInfo)
		if expanded != strings.TrimSpace(c.builtin) {
			t.Errorf("agentloop/prompts/%s 展开后与内置系统提示漂移\n  asset  len=%d\n  builtin len=%d",
				c.file, len(expanded), len(strings.TrimSpace(c.builtin)))
		}
	}
}

// TestPromptAssetsPriority 优先级：运行时注册 > 磁盘插件资产 > config/roles。
func TestPromptAssetsPriority(t *testing.T) {
	ResetPromptAssetsForTest()
	defer ResetPromptAssetsForTest()
	// 构造隔离的 config/roles 目录
	rolesDir := t.TempDir()
	SetRolePromptBaseDirForTest(rolesDir)
	defer SetRolePromptBaseDirForTest("")
	if err := os.WriteFile(filepath.Join(rolesDir, "reviewer.md"), []byte("config-roles:reviewer"), 0o644); err != nil {
		t.Fatal(err)
	}
	// 磁盘插件资产（模拟插件包 prompts/）
	pkgDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(pkgDir, "prompts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "prompts", "reviewer.md"), []byte("disk-asset:reviewer"), 0o644); err != nil {
		t.Fatal(err)
	}
	ScanPluginPromptAssets(pkgDir, "demo-plugin")

	if got := LoadPrompt("reviewer"); got != "disk-asset:reviewer" {
		t.Fatalf("磁盘插件资产应优先于 config/roles：got %q", got)
	}
	// 运行时注册覆盖磁盘资产
	ProvidePrompt("reviewer", "runtime:reviewer", "js:demo")
	if got := LoadPrompt("reviewer"); got != "runtime:reviewer" {
		t.Fatalf("运行时注册应优先于磁盘资产：got %q", got)
	}
	// 运行时移除后回落到磁盘资产
	RemovePrompt("reviewer")
	if got := LoadPrompt("reviewer"); got != "disk-asset:reviewer" {
		t.Fatalf("移除运行时注册后应回落磁盘资产：got %q", got)
	}
	// config/roles 仅兜底（磁盘资产缺席时）
	if got := LoadPrompt("nonexist"); got != "" {
		t.Fatalf("未知资产应返回空：got %q", got)
	}
}

// TestScanPluginPromptAssets 扫描语义：一级 .md、跳过隐藏/子目录/非 md、防重复。
func TestScanPluginPromptAssets(t *testing.T) {
	ResetPromptAssetsForTest()
	defer ResetPromptAssetsForTest()
	pkgDir := t.TempDir()
	proDir := filepath.Join(pkgDir, "prompts")
	if err := os.MkdirAll(proDir, 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(name, content string) {
		if err := os.WriteFile(filepath.Join(proDir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("reviewer.md", "# 审核")
	write("planner.md", "# 规划")
	write(".hidden.md", "隐藏不扫")
	write("readme.txt", "非 md 不扫")
	if err := os.MkdirAll(filepath.Join(proDir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	write(filepath.Join("sub", "embed.md"), "子目录不扫")
	if n := ScanPluginPromptAssets(pkgDir, "scan-demo"); n != 2 {
		t.Fatalf("应注册 2 个资产（并跳过隐藏/非 md/子目录），got %d", n)
	}
	if LoadPrompt("reviewer") != "# 审核" || LoadPrompt("planner") != "# 规划" {
		t.Fatal("扫描资产未生效")
	}
	if LoadPrompt(".hidden") != "" || LoadPrompt("readme") != "" || LoadPrompt("embed") != "" {
		t.Fatalf("不应注册隐藏/非 md/子目录资产")
	}
	// 防重复扫描：同一插件二次扫描不改变结果
	if n := ScanPluginPromptAssets(pkgDir, "scan-demo"); n != 0 {
		t.Fatalf("重复扫描应返回 0，got %d", n)
	}
}

// TestRegisterConfigPrompts 插件配置 config.prompts（name → text）注册。
func TestRegisterConfigPrompts(t *testing.T) {
	ResetPromptAssetsForTest()
	defer ResetPromptAssetsForTest()
	cfg := map[string]any{
		"prompts": map[string]any{
			"reviewer": "config:reviewer",
			"planner":  "config:planner",
			"bad":      123, // 非字符串忽略
		},
	}
	if n := registerConfigPrompts(cfg, "cfg-plugin"); n != 2 {
		t.Fatalf("应注册 2 个配置提示词，got %d", n)
	}
	if got := LoadPrompt("reviewer"); got != "config:reviewer" {
		t.Fatalf("config.prompts 未生效：got %q", got)
	}
	if LoadPrompt("bad") != "" {
		t.Fatal("非字符串配置应忽略")
	}
	// 卸载清理：按来源移除
	RemovePromptSource("cfg-plugin:config")
	if LoadPrompt("reviewer") != "" {
		t.Fatal("插件卸载后其 config.prompts 资产应被清理")
	}
}

// TestSystemPromptAssetOverride 系统提示模板资产：{{ROOT_INFO}}/{{PRIMARY_ROOT}}
// 插值 + 未知变量原样保留；无资产时回退内置。
func TestSystemPromptAssetOverride(t *testing.T) {
	ResetPromptAssetsForTest()
	defer ResetPromptAssetsForTest()
	roots := []string{"F:\\workspace-a", "F:\\workspace-b"}
	_, rootInfo := workspaceRoots(roots)

	// 无资产 → 内置（含 rootInfo 且不含模板头）
	got0 := harnessSystemPrompt(roots)
	if !strings.Contains(got0, rootInfo) || strings.Contains(got0, "{{ROOT_INFO}}") {
		t.Fatal("无资产时应为内置提示（rootInfo 已拼接、无模板占位符）")
	}
	// 模板资产生效
	ProvidePrompt("system-harness", "## 模板\n{{ROOT_INFO}}\n{{PRIMARY_ROOT}}\n{{UNKNOWN_VAR}}", "js:tmpl")
	got := harnessSystemPrompt(roots)
	if !strings.Contains(got, rootInfo) {
		t.Fatalf("{{ROOT_INFO}} 未插值：%q", got)
	}
	if !strings.Contains(got, "F:\\workspace-a") {
		t.Fatalf("{{PRIMARY_ROOT}} 未插值：%q", got)
	}
	if !strings.Contains(got, "{{UNKNOWN_VAR}}") {
		t.Fatal("未知变量应原样保留（宽松语义）")
	}
	// full 模板独立
	ProvidePrompt("system-full", "FULL {{ROOT_INFO}}", "js:tmpl")
	if got := fullSystemPrompt(roots); !strings.Contains(got, rootInfo) || !strings.HasPrefix(got, "FULL ") {
		t.Fatalf("system-full 模板未生效：%q", got)
	}
}

// TestResolvePromptVars 变量插值宽松语义。
func TestResolvePromptVars(t *testing.T) {
	got := ResolvePromptVars("a {{X}} b {{MISS}} c", map[string]string{"X": "1"})
	if got != "a 1 b {{MISS}} c" {
		t.Fatalf("插值异常：%q", got)
	}
	if got := ResolvePromptVars("", map[string]string{"X": "1"}); got != "" {
		t.Fatal("空文本应返回空")
	}
}

// TestRealAgentLoopPromptAssetsEffect 真实资产端到端：扫描 agentloop 插件包
// prompts/ 目录 → 系统提示/角色提示走模板资产分支 → 输出与内置展开逐字节一致。
func TestRealAgentLoopPromptAssetsEffect(t *testing.T) {
	ResetPromptAssetsForTest()
	pkgDir := filepath.Join(repoRootAbs(t), ".pair", "plugins", "agentloop")
	// 基线：无资产时的内置输出
	roots := []string{"F:\\workspace-demo"}
	baseSystem := harnessSystemPrompt(roots)
	baseFull := fullSystemPrompt(roots)
	baseReviewer := DefaultReviewerPrompt()
	basePlanner := DefaultPlannerPrompt()

	n := ScanPluginPromptAssets(pkgDir, "agentloop")
	defer ResetPromptAssetsForTest()
	if n < 5 {
		t.Fatalf("agentloop 插件包应注册 ≥5 个提示词资产，got %d", n)
	}
	// 系统提示：模板资产分支输出 == 内置（逐字节）
	if got := harnessSystemPrompt(roots); got != baseSystem {
		t.Fatalf("harness 系统提示资产与内置漂移\n got len=%d\nwant len=%d", len(got), len(baseSystem))
	}
	if got := fullSystemPrompt(roots); got != baseFull {
		t.Fatalf("full 系统提示资产与内置漂移")
	}
	// 角色提示：磁盘资产 == 内置
	if got := DefaultReviewerPrompt(); got != baseReviewer {
		t.Fatalf("reviewer 资产与内置漂移")
	}
	if got := DefaultPlannerPrompt(); got != basePlanner {
		t.Fatalf("planner 资产与内置漂移")
	}
}

// TestLoadRolePromptAssetsPlugin 角色提示经插件资产覆盖（运行时注册 > 配置 > 磁盘）。
func TestLoadRolePromptAssetsPlugin(t *testing.T) {
	ResetPromptAssetsForTest()
	defer ResetPromptAssetsForTest()
	// 无任何覆盖 → 内置
	if got := DefaultReviewerPrompt(); got != reviewerSystemPrompt {
		t.Fatalf("无覆盖应回退内置 reviewer")
	}
	// 插件运行时注册覆盖
	ProvidePrompt("reviewer", "# 自定义评审\n你是插件版 Reviewer。", "js:role-plugin")
	if got := DefaultReviewerPrompt(); !strings.Contains(got, "插件版 Reviewer") {
		t.Fatalf("插件运行时注册未生效：%q", got)
	}
	// 卸载清理后回落内置
	RemovePromptSource("js:role-plugin")
	if got := DefaultReviewerPrompt(); got != reviewerSystemPrompt {
		t.Fatal("清理后应回落内置 reviewer")
	}
}
