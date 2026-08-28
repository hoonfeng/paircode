package agent

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestNativeDepHint 原生依赖提示命中与空。
func TestNativeDepHint(t *testing.T) {
	h := nativeDepHint(map[string]any{
		"dependencies": map[string]any{"better-sqlite3": "^9.0.0", "axios": "^1.0.0"},
	})
	if !strings.Contains(h, "better-sqlite3") || strings.Contains(h, "axios") {
		t.Fatalf("应只提示原生模块: %s", h)
	}
	if !strings.Contains(h, "sql.js") {
		t.Fatalf("应给出开源替代提示: %s", h)
	}
	if nativeDepHint(map[string]any{"dependencies": map[string]any{"axios": "^1.0.0"}}) != "" {
		t.Fatal("纯 JS 依赖不应提示")
	}
	if nativeDepHint(map[string]any{}) != "" {
		t.Fatal("空依赖不应提示")
	}
}

// TestNodeBridgeServiceTools 服务型插件（ctx.provide）工具暴露：
// 本地测试插件提供 demo 服务对象（hello/add 方法）→ 桥上报
// demo_hello/demo_add 工具 → invoke 转发调用服务方法。
func TestNodeBridgeServiceTools(t *testing.T) {
	nodePath, err := exec.LookPath("node")
	if err != nil {
		t.Skip("无 node 运行时")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "bridge.js"), []byte(bridgeNodeJS), 0o644); err != nil {
		t.Fatal(err)
	}
	// 复用真实桥目录的 node_modules（@cordisjs/core + cosmokit 传递依赖；
	// 需先 npm install 任一 node 插件才有）
	realNM := filepath.Join("..", "..", ".pair", "cordis", "node", "node_modules")
	if fi, err := os.Stat(realNM); err != nil || !fi.IsDir() {
		t.Skipf("无桥 node_modules（需先 npm install 任一 node 插件）: %v", err)
	}
	if err := copyDir(realNM, filepath.Join(dir, "node_modules")); err != nil {
		t.Fatal(err)
	}
	// 本地测试插件（真实 npm 包布局：node_modules/demo-service-plugin/）
	modDir := filepath.Join(dir, "node_modules", "demo-service-plugin")
	if err := os.MkdirAll(modDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modDir, "package.json"),
		[]byte(`{"name":"demo-service-plugin","version":"1.0.0","type":"module","main":"index.js"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	pluginSrc := `export default (ctx) => {
  const svc = {
    hello(name) { return 'hello ' + name; },
    add(a, b) { return Number(a) + Number(b); },
    async runtime() { return { ok: true, connected: false }; },
  };
  ctx.provide('demo', svc);
};`
	if err := os.WriteFile(filepath.Join(modDir, "index.js"), []byte(pluginSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "plugins.json"), []byte(`{"plugins":["demo-service-plugin"]}`), 0o644); err != nil {
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
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	toolsSeen := map[string]bool{}
	deadline := time.Now().Add(12 * time.Second)
	gotReady := false
	for time.Now().Before(deadline) {
		if !scanner.Scan() {
			t.Fatalf("bridge 提前退出: %v", scanner.Err())
		}
		var msg map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			continue
		}
		switch msg["t"] {
		case "tool":
			if def, ok := msg["def"].(map[string]any); ok {
				toolsSeen[fmt.Sprint(def["name"])] = true
			}
		case "ready":
			gotReady = true
		}
		if gotReady && len(toolsSeen) >= 3 {
			break
		}
	}
	if !gotReady {
		t.Fatalf("未收到 ready")
	}
	for _, want := range []string{"demo_hello", "demo_add", "demo_runtime"} {
		if !toolsSeen[want] {
			t.Fatalf("服务工具 %s 未上报（已见: %v）", want, toolsSeen)
		}
	}

	invoke := func(tool string, args map[string]any) (string, error) {
		if _, err := stdin.Write([]byte(fmt.Sprintf(`{"t":"invoke","id":7,"tool":%q,"args":%s}`+"\n", tool, mustJSON(args)))); err != nil {
			return "", err
		}
		if !scanner.Scan() {
			return "", fmt.Errorf("无响应")
		}
		var resp struct {
			T    string `json:"t"`
			OK   bool   `json:"ok"`
			Data string `json:"data"`
			Err  string `json:"error"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
			return "", fmt.Errorf("响应解析失败: %v (%s)", err, scanner.Text())
		}
		if resp.T != "result" {
			return "", fmt.Errorf("非 result 消息: %s", scanner.Text())
		}
		if !resp.OK {
			return "", fmt.Errorf("invoke 失败: %s", resp.Err)
		}
		return resp.Data, nil
	}

	if got, err := invoke("demo_hello", map[string]any{"name": "桥"}); err != nil || got != "hello 桥" {
		t.Fatalf("demo_hello 期望 'hello 桥' 实际 %q err=%v", got, err)
	}
	if got, err := invoke("demo_add", map[string]any{"a": 1, "b": 2}); err != nil || got != "3" {
		t.Fatalf("demo_add 期望 3 实际 %q err=%v", got, err)
	}
	if got, err := invoke("demo_runtime", nil); err != nil || !strings.Contains(got, `"ok":true`) {
		t.Fatalf("demo_runtime 期望含 ok:true 实际 %q err=%v", got, err)
	}
	t.Log("服务型插件工具暴露 + 调用 OK: demo_hello/demo_add/demo_runtime")
}

func mustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// TestMapBridgeStoreLoop 验证 Node 桥扩展服务（store/loop）映射与参数校验。
func TestMapBridgeStoreLoop(t *testing.T) {
	// store.read：convId 必须
	if _, _, _, err := mapBridgeService("store", "read", map[string]any{}); err == nil {
		t.Fatal("store.read 缺 convId 应报错")
	}
	// store.append：缺 role 报错
	if _, _, _, err := mapBridgeService("store", "append", map[string]any{"convId": "c1"}); err == nil {
		t.Fatal("store.append 缺 role 应报错")
	}
	// store.read 映射（nodeBridgeManager nil → 执行时报错，映射本身成功）
	tool, margs, direct, err := mapBridgeService("store", "read", map[string]any{"convId": "c1"})
	if err != nil || tool != "" || direct == nil {
		t.Fatalf("store.read 映射异常: tool=%q err=%v", tool, err)
	}
	if _, derr := direct(); derr == nil {
		t.Fatal("无会话管理器时应报错")
	}
	_ = margs
	// loop.info 映射
	if _, _, direct, err := mapBridgeService("loop", "info", map[string]any{}); err != nil || direct == nil {
		t.Fatalf("loop.info 映射异常: %v", err)
	}
	// loop.snapshot 缺 convId 报错
	if _, _, _, err := mapBridgeService("loop", "snapshot", map[string]any{}); err == nil {
		t.Fatal("loop.snapshot 缺 convId 应报错")
	}
	// 未知方法
	if _, _, _, err := mapBridgeService("store", "bogus", map[string]any{"convId": "c1"}); err == nil {
		t.Fatal("未知 method 应报错")
	}
	if _, _, _, err := mapBridgeService("loop", "bogus", map[string]any{}); err == nil {
		t.Fatal("未知 loop method 应报错")
	}
}

// TestNodeBridgeManagerSet 验证管理器注入。
func TestNodeBridgeManagerSet(t *testing.T) {
	defer func() { nodeBridgeManager = nil }()
	SetNodeBridgeManager(nil) // 幂等
	if nodeBridgeManager != nil {
		t.Fatal("set nil 应生效")
	}
	// 序列化 sanity（消息角色映射到 store.read 输出结构）
	b, _ := json.Marshal(map[string]any{"role": "user", "content": "hi"})
	if string(b) != `{"content":"hi","role":"user"}` {
		t.Fatalf("序列化异常: %s", b)
	}
}
