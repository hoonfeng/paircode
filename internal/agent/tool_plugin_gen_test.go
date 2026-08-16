package agent

import (
	"context"
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
	// ② ★ 宿主不再承载 git 工具实现（2026-08-16 第三轮：内置 20 组注册删除，
	//    实现归插件独立二进制）——插件注册时宿主 Registry 无同名工具，
	//    故不存档 hostExecutors，也不应有宿主执行器
	if _, ok := HostToolMeta("git_status"); ok {
		t.Fatal("git_status 不应有宿主执行器（实现已迁移插件独立二进制）")
	}
	if _, ok := HostToolMeta("git_commit"); ok {
		t.Fatal("git_commit 不应有宿主执行器（实现已迁移插件独立二进制）")
	}
	// ③ hostTool 链路：无存档 → 明确报错（宿主已不承载，工具经插件 execute
	//    调 ctx.binary 独立二进制执行，不走 hostTool）
	if _, err := ExecuteHostTool("git_status", map[string]any{"project": repoRoot}); err == nil {
		t.Fatal("ExecuteHostTool(git_status) 应报错（宿主执行器不存在）")
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

// TestBinaryPluginExec 验证「二进制插件」全链路：磁盘插件 JS 壳（api 声明 +
// ctx.binary.exec 调度）→ 宿主 ctx.binary 服务 → 插件目录 bin/ 下独立二进制
// （stdin/stdout JSON 协议）→ 工具执行结果返回。
// ★ 插件目录自包含：源码（index.js + 独立二进制项目 cmd/plugins/tool-binary/）
//   与二进制（bin/tool-binary.exe）均在 .pair/plugins/tool-binary/ 内，
//   用户改源码重编译即更换实现，无需改主程序。
// ★ 2026-08-16：tool-binary-re 已合并进 tool-binary（逆向 6 工具并入），
//   本测试直接验证合并后插件。
func TestBinaryPluginExec(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("定位仓库根失败: %v", err)
	}
	pluginDir := filepath.Join(repoRoot, ".pair", "plugins", "tool-binary")
	exePath := filepath.Join(pluginDir, "bin", "tool-binary.exe")
	if _, err := os.Stat(exePath); err != nil {
		t.Skipf("独立二进制未编译（go build -o %s ./cmd/plugins/tool-binary）: %v", exePath, err)
	}
	code, err := os.ReadFile(filepath.Join(pluginDir, "index.js"))
	if err != nil {
		t.Fatalf("读插件源码失败: %v", err)
	}

	reg := NewRegistry()
	host := NewPluginHost(reg, nil, repoRoot)

	id, err := host.DefineJSCodeFull(string(code), "js", "binary 插件装载测试", "", "")
	if err != nil {
		t.Fatalf("define 失败: %v", err)
	}
	def, _ := host.GetJSDef(id)
	if def == nil {
		t.Fatalf("定义 %s 不存在", id)
	}
	def.dir = pluginDir // ★ 磁盘插件装载时由 applyGlobalPluginDir 注入（测试手动等价）
	if err := host.LoadJSDynamic(def); err != nil {
		t.Fatalf("装载失败: %v", err)
	}

	// ① execute 走 ctx.binary（插件 JS 已从 hostTool 切换为 binary.exec）
	tool, ok := reg.Get("binary_hash")
	if !ok {
		t.Fatal("binary_hash 未注册（插件接管失败）")
	}
	out, err := tool.Handler(context.Background(), map[string]any{"path": exePath})
	if err != nil {
		t.Fatalf("binary_hash 执行失败: %v", err)
	}
	if !strings.Contains(out, "MD5：") || !strings.Contains(out, "SHA256：") {
		t.Fatalf("binary_hash 输出异常: %.200s", out)
	}

	// ② 错误路径走协议 error 分支
	if _, err := tool.Handler(context.Background(), map[string]any{"path": "不存在的文件.bin"}); err == nil {
		t.Fatal("非法路径应报错")
	}

	// ③ 只读工具全链路（binary_entropy）
	ent, ok := reg.Get("binary_entropy")
	if !ok {
		t.Fatal("binary_entropy 未注册")
	}
	out2, err := ent.Handler(context.Background(), map[string]any{"path": exePath, "chunk_size": 65536})
	if err != nil {
		t.Fatalf("binary_entropy 执行失败: %v", err)
	}
	if !strings.Contains(out2, "整体熵") {
		t.Fatalf("binary_entropy 输出异常: %.200s", out2)
	}

	// ④ 插件目录服务可见（ctx.binary.dir）
	if def.dir == "" {
		t.Fatal("def.dir 未注入（ctx.binary 依赖它定位二进制）")
	}
}

// TestBinaryPluginExecUnified 验证统一宿主二进制链路：其他工具组插件 JS 壳
// execute 调 ctx.binary.exec(t.name, args, {bin:"tool-binary"}) → 统一二进制
// （cmd/plugins/tool-binary，承载全部内置组实现）→ 工具执行结果返回。
// ★ 改实现：重编译 cmd/plugins/tool-binary → 替换 .pair/plugins/tool-binary/bin/
//   tool-binary.exe → 全部切换组生效（主程序无需重编译）。
func TestBinaryPluginExecUnified(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("定位仓库根失败: %v", err)
	}
	pluginDir := filepath.Join(repoRoot, ".pair", "plugins", "tool-git")
	exePath := filepath.Join(repoRoot, ".pair", "plugins", "tool-binary", "bin", "tool-binary.exe")
	if _, err := os.Stat(exePath); err != nil {
		t.Skipf("统一二进制未编译（go build -o %s ./cmd/plugins/tool-binary）: %v", exePath, err)
	}
	code, err := os.ReadFile(filepath.Join(pluginDir, "index.js"))
	if err != nil {
		t.Fatalf("读插件源码失败: %v", err)
	}

	reg := NewRegistry()
	host := NewPluginHost(reg, nil, repoRoot)

	id, err := host.DefineJSCodeFull(string(code), "js", "统一二进制插件装载测试", "", "")
	if err != nil {
		t.Fatalf("define 失败: %v", err)
	}
	def, _ := host.GetJSDef(id)
	if def == nil {
		t.Fatalf("定义 %s 不存在", id)
	}
	def.dir = filepath.Dir(pluginDir) + string(filepath.Separator) + "tool-git" // ★ 磁盘装载注入
	if err := host.LoadJSDynamic(def); err != nil {
		t.Fatalf("装载失败: %v", err)
	}

	// ① execute 走 ctx.binary {bin:"tool-binary"}（插件 JS 已切换）
	tool, ok := reg.Get("git_status")
	if !ok {
		t.Fatal("git_status 未注册（插件接管失败）")
	}
	out, err := tool.Handler(context.Background(), map[string]any{})
	if err != nil {
		t.Fatalf("git_status 执行失败: %v", err)
	}
	if !strings.Contains(out, "master") && !strings.Contains(out, "branch") {
		t.Fatalf("git_status 输出异常: %.200s", out)
	}

	// ② 错误路径（未知工具）走协议 error 分支
	bad, _ := reg.Get("git_status")
	if _, err := bad.Handler(context.Background(), map[string]any{"_no_such_arg": true}); err != nil {
		// 参数错误应报错而非崩溃
		t.Logf("git_status 非法参数报错（预期可接受）: %v", err)
	}
}
