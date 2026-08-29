package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestBridgeNodeResourceSync 防漂移守卫（历史回归 F1a，2026-08-29）：
// bridgeNodeSource() 外部资源优先（<exe 目录>/.pair/assets/runtime/bridge_node.js），
// 真实应用会加载外置脚本而非 embed——若外置为旧版（无 cordis4/DSH 分支），
// DSH 插件（runtime=dsh）装载必失败（旧脚本将 {spec,runtime} 对象 String() 化，
// 报 Cannot find package '[object Object]'），而 E2E 测试直写 embed 变量可全绿。
// 守卫：外置文件必须与内嵌版本完全一致。
func TestBridgeNodeResourceSync(t *testing.T) {
	// 内嵌版本必须含 DSH/cordis4 分支（运行时升级语义）
	for _, marker := range []string{`runtime=="dsh"`, "@deepseek-ai/cordis"} {
		if !strings.Contains(bridgeNodeJS, marker) {
			t.Fatalf("内嵌 bridge_node.js 缺少 DSH 分支标记 %q——运行时升级未落地", marker)
		}
	}
	// 外置资源（测试 cwd=internal/agent，仓库根为 ../..）必须与内嵌一致
	ext := filepath.Join("..", "..", ".pair", "assets", "runtime", "bridge_node.js")
	data, err := os.ReadFile(ext)
	if err != nil {
		t.Fatalf("读取外置桥资源失败: %v——请同步 internal/agent/bridge_node.js → .pair/assets/runtime/bridge_node.js", err)
	}
	if string(data) != bridgeNodeJS {
		t.Fatalf("外置桥资源与内嵌版本不一致（外置 %d 字节 vs 内嵌 %d 字节）——请同步 internal/agent/bridge_node.js → .pair/assets/runtime/bridge_node.js，否则真实应用将运行旧版桥脚本导致 DSH 插件装载失败（F1a）", len(data), len(bridgeNodeJS))
	}
}
