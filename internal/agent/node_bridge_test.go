package agent

import (
	"bufio"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestBridgeNodeSourceExternalPriority 「外置资源优先」场景验证（Round4 repair
// t6 F1a）：
//  ① 外置旧版 vs 内嵌新版 → bridgeNodeSource() 返回外置内容（外部优先，F1a
//     遮蔽根因的机制本身）；
//  ② 外置缺失 → 回退内嵌（单文件分发兜底）；
//  ③ 同步回归：仓库跟踪的外置资源（.pair/assets/runtime/bridge_node.js）必须
//     与内嵌 bridge_node.js 同版本语义（归一化换行后逐字节一致）——防止
//     「外置旧版（12,329B，2e0f36ab 起）遮蔽内嵌新版（29,834B）」再次发生。
func TestBridgeNodeSourceExternalPriority(t *testing.T) {
	// ① 外部优先（遮蔽内嵌）
	extDir := t.TempDir()
	orig := runtimeAssetDirOverride
	runtimeAssetDirOverride = func() string { return extDir }
	defer func() { runtimeAssetDirOverride = orig }()
	marker := "// EXTERNAL_OLD_BRIDGE_MARKER\n"
	if err := os.WriteFile(filepath.Join(extDir, "bridge_node.js"), []byte(marker), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := bridgeNodeSource(); got != marker {
		t.Fatalf("外置资源应优先于内嵌（F1a 机制）：got len=%d want %q", len(got), marker)
	}
	// ② 外置缺失 → 回退内嵌
	if err := os.Remove(filepath.Join(extDir, "bridge_node.js")); err != nil {
		t.Fatal(err)
	}
	if got := bridgeNodeSource(); got != bridgeNodeJS {
		t.Fatalf("外置缺失应回退内嵌：got len=%d want len=%d", len(got), len(bridgeNodeJS))
	}
	// ③ 同步回归：仓库外置资源 == 内嵌（同版本语义）
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Skipf("未找到仓库根（go.mod 不在 %s）", root)
	}
	asset, err := os.ReadFile(filepath.Join(root, ".pair", "assets", "runtime", "bridge_node.js"))
	if err != nil {
		t.Fatalf("仓库外置 bridge_node.js 缺失——真实装载将用内嵌（不遮蔽）但外置资源链断裂: %v", err)
	}
	// 语义标记（内嵌必须含 Round4 cordis4/dsh 装载语义；外置与其一致）
	norm := func(b []byte) string { return strings.ReplaceAll(string(b), "\r\n", "\n") }
	for _, m := range []string{"@deepseek-ai/cordis", "decorateDshCtx", "spec.runtime === 'dsh'"} {
		if !strings.Contains(norm([]byte(bridgeNodeJS)), m) {
			t.Fatalf("内嵌 bridge_node.js 缺 Round4 语义标记 %q（版本回退？）", m)
		}
	}
	if norm(asset) != norm([]byte(bridgeNodeJS)) {
		t.Fatalf("仓库外置 bridge_node.js 与内嵌不一致（len %d vs %d）——外置旧版会遮蔽内嵌新版，真实装载用旧语义（F1a 回归）",
			len(asset), len(bridgeNodeJS))
	}
}

// TestMapBridgeService 服务映射：fs/web/bash → Go 侧工具。
func TestMapBridgeService(t *testing.T) {
	cases := []struct {
		svc, method string
		args        map[string]any
		wantTool    string
	}{
		{"fs", "read", map[string]any{"path": "a.go"}, "read"},
		{"fs", "write", map[string]any{"path": "a.go", "content": "x"}, "write"},
		{"fs", "list", map[string]any{"path": "."}, ""},
		{"web", "fetch", map[string]any{"url": "https://x.com"}, "web_fetch"},
		{"web", "search", map[string]any{"query": "go"}, "web_search"},
		{"bash", "exec", map[string]any{"command": "echo hi"}, "bash"},
	}
	for _, c := range cases {
		tool, mapped, direct, err := mapBridgeService(c.svc, c.method, c.args)
		if err != nil {
			t.Fatalf("%s.%s 报错: %v", c.svc, c.method, err)
		}
		if tool != c.wantTool {
			t.Fatalf("%s.%s 期望工具 %s 实际 %s", c.svc, c.method, c.wantTool, tool)
		}
		if c.method == "read" {
			if mapped["path"] != "a.go" {
				t.Fatalf("read 参数映射异常: %v", mapped)
			}
		}
		_ = direct
	}
	// fs.exists 直连处理器
	_, _, direct, err := mapBridgeService("fs", "exists", map[string]any{"path": "nope-not-exist"})
	if err != nil || direct == nil {
		t.Fatalf("fs.exists 应返回直连处理器: %v", err)
	}
	// fs.list 直连处理器（R2-9：无对应工具，改直连列目录）
	_, _, direct2, err := mapBridgeService("fs", "list", map[string]any{"path": "."})
	if err != nil || direct2 == nil {
		t.Fatalf("fs.list 应返回直连处理器: %v", err)
	}
	if _, _, _, err := mapBridgeService("fs", "chmod", nil); err == nil {
		t.Fatalf("未知方法应报错")
	}
}

// TestNodeBridgeProtocolPing spawn 真实 node 跑 bridge.js（空插件清单），
// 验证 JSON Lines 协议闭环：ready → ping → pong。
func TestNodeBridgeProtocolPing(t *testing.T) {
	nodePath, err := exec.LookPath("node")
	if err != nil {
		t.Skip("无 node 运行时")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "bridge.js"), []byte(bridgeNodeJS), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "plugins.json"), []byte(`{"plugins":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(nodePath, filepath.Join(dir, "bridge.js"))
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "CORDIS_BRIDGE_DIR="+dir)
	stdin, _ := cmd.StdinPipe()
	stdout, _ := cmd.StdoutPipe()
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer cmd.Process.Kill()

	scanner := bufio.NewScanner(stdout)
	gotReady := false
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) && !gotReady {
		if !scanner.Scan() {
			t.Fatalf("bridge 提前退出: %v", scanner.Err())
		}
		var msg map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			continue
		}
		if msg["t"] == "ready" {
			gotReady = true
		}
	}
	if !gotReady {
		t.Fatalf("未收到 ready")
	}

	// ping → pong
	if _, err := stdin.Write([]byte(`{"t":"ping","id":42}` + "\n")); err != nil {
		t.Fatal(err)
	}
	if !scanner.Scan() {
		t.Fatalf("ping 无响应")
	}
	var resp struct {
		T    string `json:"t"`
		ID   int64  `json:"id"`
		OK   bool   `json:"ok"`
		Data string `json:"data"`
	}
	if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
		t.Fatalf("响应解析失败: %v (%s)", err, scanner.Text())
	}
	if resp.T != "result" || resp.ID != 42 || !resp.OK || resp.Data != "pong" {
		t.Fatalf("ping 响应异常: %+v", resp)
	}
	t.Log("协议闭环 OK: ready → ping → pong")
}

// TestNodePluginNeedsNode 运行时依赖检测（Node 桥 vs goja 判定）。
func TestNodePluginNeedsNode(t *testing.T) {
	if !nodePluginNeedsNode(map[string]any{"dependencies": map[string]any{"dotenv": "^16.0.0"}}) {
		t.Fatalf("有 dependencies 应判 true")
	}
	if !nodePluginNeedsNode(map[string]any{"peerDependencies": map[string]any{"cordis": "^4.0.0-rc.7"}}) {
		t.Fatalf("peer cordis ^4 应判 true（goja 无 cordis4 API）")
	}
	if !nodePluginNeedsNode(map[string]any{"peerDependencies": map[string]any{"@cordisjs/core": "^4.0.0"}}) {
		t.Fatalf("peer @cordisjs/core ^4 应判 true")
	}
	// ★ Round4：外部插件形态（@deepseek-ai/cordis ^4 peer，无 dependencies）→ Node 桥 dsh 轨
	if !nodePluginNeedsNode(map[string]any{"peerDependencies": map[string]any{"@deepseek-ai/cordis": "^4.0.1-rc.1"}}) {
		t.Fatalf("peer @deepseek-ai/cordis ^4 应判 true（外部插件走 Node 桥 dsh 轨）")
	}
	if nodePluginNeedsNode(map[string]any{"dependencies": map[string]any{}}) {
		t.Fatalf("空 dependencies 应判 false")
	}
	if nodePluginNeedsNode(map[string]any{"peerDependencies": map[string]any{"cordis": "^3.5.0"}}) {
		t.Fatalf("peer cordis3 应判 false（goja 可跑）")
	}
	if nodePluginNeedsNode(map[string]any{"peerDependencies": map[string]any{"@cordisjs/plugin-webui": "^0.8.2"}}) {
		t.Fatalf("非 cordis peer 应判 false")
	}
	if nodePluginNeedsNode(map[string]any{}) {
		t.Fatalf("无依赖应判 false")
	}
}

// TestNodePluginRuntime 运行时轨判定（dsh / node / goja）。
func TestNodePluginRuntime(t *testing.T) {
	if got := nodePluginRuntime(map[string]any{"peerDependencies": map[string]any{"@deepseek-ai/cordis": "^4.0.1-rc.1"}}); got != "dsh" {
		t.Fatalf("@deepseek-ai/cordis peer 应判 dsh，实际 %q", got)
	}
	if got := nodePluginRuntime(map[string]any{"dependencies": map[string]any{"axios": "^1.0.0"}}); got != "node" {
		t.Fatalf("dependencies 非空应判 node，实际 %q", got)
	}
	if got := nodePluginRuntime(map[string]any{"peerDependencies": map[string]any{"@cordisjs/core": "^4.0.0"}}); got != "node" {
		t.Fatalf("@cordisjs/core ^4 peer 应判 node，实际 %q", got)
	}
	if got := nodePluginRuntime(map[string]any{"peerDependencies": map[string]any{"cordis": "^3.5.0"}}); got != "" {
		t.Fatalf("cordis3 peer 应判 goja（空），实际 %q", got)
	}
	if got := nodePluginRuntime(map[string]any{}); got != "" {
		t.Fatalf("无依赖应判 goja（空），实际 %q", got)
	}
	// 同时含 @deepseek-ai/cordis 与 dependencies → dsh 优先（cordis4 语义）
	if got := nodePluginRuntime(map[string]any{
		"dependencies":     map[string]any{"axios": "^1.0.0"},
		"peerDependencies": map[string]any{"@deepseek-ai/cordis": "^4.0.1"},
	}); got != "dsh" {
		t.Fatalf("dsh peer 应优先于 dependencies，实际 %q", got)
	}
}

// TestNodePluginsFile plugins.json 读写幂等 + 旧格式兼容。
func TestNodePluginsFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "plugins.json")
	if err := writeNodePluginsFile(p, &nodePluginsFile{Plugins: []nodePluginEntry{
		{Spec: "a@1", Runtime: "node"}, {Spec: "b@2", Runtime: "dsh"},
	}}); err != nil {
		t.Fatal(err)
	}
	doc, err := readNodePluginsFile(p)
	if err != nil || len(doc.Plugins) != 2 {
		t.Fatalf("plugins.json 往返失败: %v %+v", err, doc)
	}
	if doc.Plugins[0].Spec != "a@1" || doc.Plugins[0].Runtime != "node" {
		t.Fatalf("条目 0 异常: %+v", doc.Plugins[0])
	}
	if doc.Plugins[1].Spec != "b@2" || doc.Plugins[1].Runtime != "dsh" {
		t.Fatalf("条目 1（dsh runtime）异常: %+v", doc.Plugins[1])
	}
	if got := doc.Specs(); len(got) != 2 || got[0] != "a@1" || got[1] != "b@2" {
		t.Fatalf("Specs() 异常: %v", got)
	}
	// 旧格式（字符串数组）读取兼容 → 默认 runtime=node
	legacy := filepath.Join(t.TempDir(), "plugins.json")
	if err := os.WriteFile(legacy, []byte(`{"plugins":["c@3","d@4"]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	ldoc, err := readNodePluginsFile(legacy)
	if err != nil || len(ldoc.Plugins) != 2 || ldoc.Plugins[0].Spec != "c@3" || ldoc.Plugins[0].Runtime != "node" {
		t.Fatalf("旧格式兼容失败: %v %+v", err, ldoc)
	}
	if specMatchesPkg("cordis-plugin-android@0.0.7", "cordis-plugin-android") != true {
		t.Fatalf("specMatchesPkg 匹配失败")
	}
	if strings.TrimSpace("") != "" {
		t.Fatal("unreachable")
	}
}
