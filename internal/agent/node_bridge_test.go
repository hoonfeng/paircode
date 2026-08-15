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

// TestMapBridgeService 服务映射：fs/web/bash → Go 侧工具。
func TestMapBridgeService(t *testing.T) {
	cases := []struct {
		svc, method string
		args        map[string]any
		wantTool    string
	}{
		{"fs", "read", map[string]any{"path": "a.go"}, "read_file"},
		{"fs", "write", map[string]any{"path": "a.go", "content": "x"}, "write_file"},
		{"fs", "list", map[string]any{"path": "."}, "list_files"},
		{"web", "fetch", map[string]any{"url": "https://x.com"}, "web_fetch"},
		{"web", "search", map[string]any{"query": "go"}, "web_search"},
		{"bash", "exec", map[string]any{"command": "echo hi"}, "run_command"},
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

// TestNodePluginsFile plugins.json 读写幂等。
func TestNodePluginsFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "plugins.json")
	if err := writeNodePluginsFile(p, &nodePluginsFile{Plugins: []string{"a@1", "b@2"}}); err != nil {
		t.Fatal(err)
	}
	doc, err := readNodePluginsFile(p)
	if err != nil || len(doc.Plugins) != 2 || doc.Plugins[0] != "a@1" {
		t.Fatalf("plugins.json 往返失败: %v %+v", err, doc)
	}
	if specMatchesPkg("cordis-plugin-android@0.0.7", "cordis-plugin-android") != true {
		t.Fatalf("specMatchesPkg 匹配失败")
	}
	if strings.TrimSpace("") != "" {
		t.Fatal("unreachable")
	}
}
