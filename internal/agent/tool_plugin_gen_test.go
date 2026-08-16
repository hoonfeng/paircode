package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGeneratedDiskPluginLoad 验证自动生成的磁盘工具插件（tool_plugin_gen.go 产物）
// 可被 goja 沙箱装载：读仓库 .pair/plugins/tool-git/index.js → define+load →
// 同名工具接管（宿主执行器存档）+ Registry 注册成功。
func TestGeneratedDiskPluginLoad(t *testing.T) {
	// 定位仓库根（测试运行于 internal/agent/）
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("定位仓库根失败: %v", err)
	}
	pluginFile := filepath.Join(repoRoot, ".pair", "plugins", "tool-git", "index.js")
	code, err := os.ReadFile(pluginFile)
	if err != nil {
		t.Skipf("生成的插件不存在（先跑 go run -tags toolsgen ./dev/tool_plugin_gen）: %v", err)
	}

	reg := NewRegistry()
	host := NewPluginHost(reg, nil, repoRoot)
	RegisterBuiltinPlugins(host)

	id, err := host.DefineJSCodeFull(string(code), "js", "tool-git 装载测试", "", "")
	if err != nil {
		t.Fatalf("define 失败: %v", err)
	}
	def, _ := host.GetJSDef(id)
	if def == nil {
		t.Fatalf("定义 %s 不存在", id)
	}
	if err := host.LoadJSDynamic(def); err != nil {
		t.Fatalf("装载失败: %v", err)
	}

	// ① 10 个 git_* 工具全部注册（插件接管）
	for _, name := range []string{"git_status", "git_diff", "git_log", "git_show", "git_blame",
		"git_add", "git_commit", "git_branch", "git_checkout", "git_stash"} {
		tool, ok := reg.Get(name)
		if !ok {
			t.Fatalf("%s 未注册", name)
		}
		if strings.TrimSpace(tool.Description) == "" {
			t.Fatalf("%s 描述为空", name)
		}
		if !strings.Contains(tool.Name, "git_") {
			t.Fatalf("%s 名称异常", name)
		}
	}
	// ② 宿主执行器已存档（供 ctx.hostTool 复用）
	if _, ok := HostToolMeta("git_status"); !ok {
		t.Fatal("git_status 宿主执行器未存档")
	}
	if _, ok := HostToolMeta("git_commit"); !ok {
		t.Fatal("git_commit 宿主执行器未存档（含 write 类工具也应存档）")
	}
	// ③ hostTool 链路可执行（非 git 目录返回说明信息而非崩溃/未知工具）
	if _, err := ExecuteHostTool("git_status", map[string]any{"project": repoRoot}); err != nil {
		t.Fatalf("ExecuteHostTool(git_status) 失败: %v", err)
	}
}

// TestGeneratedPluginPackagesComplete 生成的插件包必须含 package.json
// （LoadGlobalPlugins 按 package.json 扫描，缺则跳过装载）。
func TestGeneratedPluginPackagesComplete(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("定位仓库根失败: %v", err)
	}
	pluginsDir := filepath.Join(repoRoot, ".pair", "plugins")
	entries, err := os.ReadDir(pluginsDir)
	if err != nil {
		t.Skipf(".pair/plugins 不存在: %v", err)
	}
	for _, e := range entries {
		if !e.IsDir() || !strings.HasPrefix(e.Name(), "tool-") {
			continue
		}
		pkg := filepath.Join(pluginsDir, e.Name(), "package.json")
		if _, err := os.Stat(pkg); err != nil {
			t.Errorf("%s 缺 package.json（装载器会跳过）: %v", e.Name(), err)
		}
	}
}
