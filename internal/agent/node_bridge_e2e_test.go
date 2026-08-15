package agent

// ═══════════════════════════════════════════════════════════
// node_bridge_e2e_test.go — Node 桥端到端验证
//
// 前置（本机已就绪）：
//   node >= 18；工作区 .pair/cordis/node/ 已 npm install 本地测试插件
//   local-test-plugin（源码 _tmp_bridge_plugin/，tools.register hello_bridge）。
//
// 链路：RegisterDefaultTools 建宿主工具表 → NewPluginHost →
// ensureNodeBridge 起 node 桥 → Node 侧装载插件并回传 hello_bridge 工具
// 定义（t:tool）→ Go 注册进 Registry → Execute 调用（invoke 转发 Node 执行；
// 插件内部 ctx.fs.read / ctx.bash.exec 经桥回发 service → Go 侧 read_file /
// run_command 执行 → 结果回传 Node）→ 组合结果回 Go。
// 环境不满足（无 node / 插件未安装）时自动 Skip，不影响 go test ./...。
// ═══════════════════════════════════════════════════════════

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNodeBridgeE2EHelloBridge(t *testing.T) {
	nodePath, err := exec.LookPath("node")
	if err != nil {
		t.Skip("无 node 运行时（需要 Node 18+）")
	}
	_ = nodePath

	// 工作区根：internal/agent → 上溯两级；校验 go.mod 兜底
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Skipf("未找到工作区根（go.mod 不在 %s）", root)
	}
	dir := filepath.Join(root, ".pair", "cordis", "node")
	if _, err := os.Stat(filepath.Join(dir, "node_modules", "local-test-plugin", "index.js")); err != nil {
		t.Skip("local-test-plugin 未安装到桥目录（需先 npm install local-test-plugin）")
	}

	// 宿主：核心工具 + 插件宿主（工作区根 = root）
	reg := NewRegistry()
	RegisterDefaultTools(reg, root)
	ph := NewPluginHost(reg, nil, root)
	SetGlobalPluginHost(ph)

	b, err := ensureNodeBridge(ph, dir)
	if err != nil {
		t.Fatalf("Node 桥启动失败: %v", err)
	}
	defer func() {
		b.Close()
		globalNodeBridge = nil
	}()

	// 等待 hello_bridge 注册进 Registry（Node ready → t:tool → handleToolMsg）
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		if _, ok := reg.Get("hello_bridge"); ok {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if _, ok := reg.Get("hello_bridge"); !ok {
		t.Fatalf("hello_bridge 未注册进 Registry（桥工具回传失败）")
	}

	// 调用工具（name=桥接测试）——内部 fs.read('go.mod') + bash.exec 走桥转发
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	out, err := reg.Execute(ctx, "hello_bridge", `{"name":"桥接测试"}`)
	if err != nil {
		t.Fatalf("hello_bridge 调用失败: %v", err)
	}
	t.Logf("hello_bridge 返回: %s", out)

	// 断言：name 回显 + go.mod 首行 + bash 转发标记
	if !strings.Contains(out, "桥接测试") {
		t.Errorf("返回缺少 name 回显: %q", out)
	}
	if !strings.Contains(out, "module github.com/hoonfeng/paircode") {
		t.Errorf("返回缺少 go.mod 首行（fs.read 转发失败）: %q", out)
	}
	if !strings.Contains(out, "BRIDGE_BASH_OK") {
		t.Errorf("返回缺少 bash 转发标记: %q", out)
	}
}
