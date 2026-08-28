// ═══════════════════════════════════════════════════════════
// node_bridge.go — Node 运行时桥（真实 node 进程执行 npm cordis 插件）
//
// goja 沙箱只能执行无 npm 依赖的插件（非相对导入 mock 空模块）；
// 依赖 npm 生态（dotenv/axios 等）的 cordis 插件在真实 node 子进程中
// 运行：spawn `node bridge.js`（嵌入的 bridge_node.js，落盘到
// .pair/cordis/node/），stdin/stdout JSON Lines 双向通信：
//   - 插件 ctx.tools.register 注册的工具 → 进主 Registry（agent 可调用，
//     调用时转发 invoke 回 Node 执行）
//   - 插件 ctx.fs/web/bash 服务调用 → 转发 Go 侧对应工具
//     （read_file/write_file/list_files/web_fetch/run_command），
//     行为与 goja 插件一致（工作区根限制、输出格式同源）
//   - 崩溃自动重启（有限次）、退出清理子进程
//
// 桥目录结构（工作区 .pair/cordis/node/）：
//
//	bridge.js      ← 本文件 embed 内容落盘
//	plugins.json   ← {"plugins":["pkg@ver",...]} 要装载的插件（node_modules 安装）
//	node_modules/  ← npm install 的依赖（@cordisjs/core + 插件及其依赖）
//
// ═══════════════════════════════════════════════════════════
package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"github.com/hoonfeng/paircode/pkg/executil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	_ "embed"
)

//go:embed bridge_node.js
var bridgeNodeJS string

// bridgeNodeSource 返回 Node 桥脚本源码。★ 资源外置：优先读
// <exe 目录>/.pair/assets/runtime/bridge_node.js（可独立更新），缺失回退 embed。
func bridgeNodeSource() string {
	if s, ok := LoadRuntimeAssetString("bridge_node.js", bridgeNodeJS); ok {
		return s
	}
	return bridgeNodeJS
}

// bridgeResult Node 桥返回结果（invoke/service 响应）。
type bridgeResult struct {
	ok    bool
	data  string
	error string
}

// nodeBridge Node 桥进程管理器（全局单例：一个进程装载所有 node 型插件）。
type nodeBridge struct {
	mu       sync.Mutex
	cmd      *exec.Cmd
	stdin    *bufio.Writer
	pending  map[int64]chan bridgeResult
	seq      int64
	ph       *PluginHost
	dir      string // .pair/cordis/node
	ready    bool
	restarts int
	closed   bool
	toolsMu  sync.Mutex
	tools    map[string]*Tool // 已注册到宿主的桥工具（新宿主补注册用）
}

// globalNodeBridge 全局单例（对齐 GetGlobalPluginHost 模式）。
var globalNodeBridge *nodeBridge

// nodeBridgeManager 会话管理器引用（Node 插件 ctx.store / ctx.loop 服务后端）。
// cmd/companion 启动时经 SetNodeBridgeManager 注入（跨包解耦）。
var nodeBridgeManager *SessionManager

// SetNodeBridgeManager 注入会话管理器（web 层启动时调用一次）。
func SetNodeBridgeManager(m *SessionManager) { nodeBridgeManager = m }

// nodeBridgeDir 返回工作区 Node 桥目录（.pair/cordis/node）。
func nodeBridgeDir() string {
	root := npmPluginProjectRoot()
	if root == "" {
		root, _ = os.Getwd()
	}
	return filepath.Join(root, ".pair", "cordis", "node")
}

// nodeBridgeReady 判断 Node 桥是否在运行且就绪（含 node 型插件时）。
func nodeBridgeReady() bool {
	b := globalNodeBridge
	return b != nil && b.isReady()
}

func (b *nodeBridge) isReady() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.ready
}

// ensureNodeBridge 确保 Node 桥进程运行（首次启动；崩溃后自动重启）。
// ★ 每次调用都绑定最新宿主 ph（新 agent 实例有独立 registry，需补注册桥工具）。
func ensureNodeBridge(ph *PluginHost, dir string) (*nodeBridge, error) {
	if globalNodeBridge != nil && globalNodeBridge.isReady() {
		globalNodeBridge.bindHost(ph)
		return globalNodeBridge, nil
	}
	if globalNodeBridge != nil {
		globalNodeBridge.Close()
		globalNodeBridge = nil
	}
	nodePath, err := exec.LookPath("node")
	if err != nil {
		return nil, fmt.Errorf("未找到 node 运行时（需要 Node 18+）：%v", err)
	}
	b := &nodeBridge{ph: ph, dir: dir, pending: map[int64]chan bridgeResult{}, tools: map[string]*Tool{}}
	if err := b.start(nodePath); err != nil {
		return nil, err
	}
	globalNodeBridge = b
	return b, nil
}

// bindHost 绑定最新宿主：更新 ph 并把已注册的桥工具补注册到新宿主 registry。
func (b *nodeBridge) bindHost(ph *PluginHost) {
	if ph == nil {
		return
	}
	b.mu.Lock()
	b.ph = ph
	b.mu.Unlock()
	b.registerToolsTo(ph)
}

// registerToolsTo 把桥已注册工具补注册到指定宿主（新 agent 实例用）。
func (b *nodeBridge) registerToolsTo(ph *PluginHost) {
	if ph == nil {
		return
	}
	b.toolsMu.Lock()
	tools := make([]*Tool, 0, len(b.tools))
	for _, t := range b.tools {
		tools = append(tools, t)
	}
	b.toolsMu.Unlock()
	for _, t := range tools {
		if err := ph.Context().RegisterTool(t); err != nil {
			log.Printf("[node-bridge] 补注册工具 %s 到新宿主失败: %v", t.Name, err)
		}
		// ★ 2026-08-17：装载 ≠ agent 可用——Node 桥插件工具同样受工作区工具集
		//   可见性收敛（不在工具集白名单 → 对 agent 隐藏，cordis/前端仍可见）。
		ph.hideToolIfNotInToolset(t.Name)
	}
}

// start 启动 node 子进程并等待 ready。
func (b *nodeBridge) start(nodePath string) error {
	bridgeJS := filepath.Join(b.dir, "bridge.js")
	if err := os.MkdirAll(b.dir, 0o755); err != nil {
		return fmt.Errorf("创建桥目录失败: %v", err)
	}
	if err := os.WriteFile(bridgeJS, []byte(bridgeNodeSource()), 0o644); err != nil {
		return fmt.Errorf("写入 bridge.js 失败: %v", err)
	}

	cmd := exec.Command(nodePath, bridgeJS)
	cmd.Dir = b.dir
	// ★ 2026-08-19：node.exe 是 console 程序——父进程无控制台（后台/服务方式
	//   启动）时会自己弹出控制台窗口；这里显式隐藏，杜绝弹窗。
	if runtime.GOOS == "windows" {
		executil.HideWindow(cmd)
	}
	cmd.Env = append(os.Environ(),
		"CORDIS_BRIDGE_DIR="+b.dir,
		"CORDIS_WORKSPACE_ROOT="+npmPluginProjectRoot(),
	)
	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin 管道失败: %v", err)
	}
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout 管道失败: %v", err)
	}
	cmd.Stderr = os.Stderr // node 侧 console 原样透出

	b.mu.Lock()
	b.cmd = cmd
	b.stdin = bufio.NewWriter(stdinPipe)
	b.pending = map[int64]chan bridgeResult{}
	b.ready = false
	b.mu.Unlock()
	b.toolsMu.Lock()
	if b.tools == nil {
		b.tools = map[string]*Tool{}
	}
	b.toolsMu.Unlock()

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动 node 桥失败: %v", err)
	}
	go b.readLoop(stdoutPipe)

	// 等待 ready（最长 10s）
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if b.isReady() {
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	b.Close()
	return fmt.Errorf("node 桥启动超时（10s 未 ready）")
}

// readLoop 读取 Node 侧 stdout JSON Lines 并分发。
func (b *nodeBridge) readLoop(stdout interface{ Read([]byte) (int, error) }) {
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var msg struct {
			T       string          `json:"t"`
			ID      int64           `json:"id"`
			OK      bool            `json:"ok"`
			Data    string          `json:"data"`
			Err     string          `json:"error"`
			Level   string          `json:"level"`
			Text    string          `json:"msg"`
			Plugins []string        `json:"plugins"`
			Tool    string          `json:"tool"`
			Plugin  string          `json:"plugin"`
			Def     json.RawMessage `json:"def"`
			Svc     string          `json:"svc"`
			Method  string          `json:"method"`
			Args    json.RawMessage `json:"args"`
		}
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			log.Printf("[node-bridge] 协议解析失败: %v (line %.80s)", err, line)
			continue
		}
		switch msg.T {
		case "ready":
			b.mu.Lock()
			b.ready = true
			b.mu.Unlock()
			log.Printf("[node-bridge] 就绪（插件 %v，工具 %v）", msg.Plugins, msg.Tool)
		case "tool":
			b.handleToolMsg(msg.Plugin, msg.Def)
		case "service":
			b.handleServiceMsg(msg.ID, msg.Svc, msg.Method, msg.Args)
		case "result":
			b.mu.Lock()
			ch := b.pending[msg.ID]
			delete(b.pending, msg.ID)
			b.mu.Unlock()
			if ch != nil {
				ch <- bridgeResult{ok: msg.OK, data: msg.Data, error: msg.Err}
			}
		case "log":
			log.Printf("[node-bridge:%s] %s", msg.Level, msg.Text)
		}
	}
	// 进程退出/管道关闭
	b.mu.Lock()
	exited := b.closed
	b.ready = false
	b.mu.Unlock()
	if !exited {
		log.Printf("[node-bridge] 进程退出（异常），自动重启")
		b.restart()
	}
}

// restart 崩溃后重启（最多 3 次；每次间隔递增）。
func (b *nodeBridge) restart() {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return
	}
	b.restarts++
	n := b.restarts
	b.mu.Unlock()
	if n > 3 {
		log.Printf("[node-bridge] 重启超过 3 次，放弃（请检查 node 环境与插件兼容性）")
		return
	}
	time.Sleep(time.Duration(n) * 500 * time.Millisecond)
	nodePath, err := exec.LookPath("node")
	if err != nil {
		log.Printf("[node-bridge] 重启失败: %v", err)
		return
	}
	if err := b.start(nodePath); err != nil {
		log.Printf("[node-bridge] 重启失败: %v", err)
	}
}

// handleToolMsg 注册 Node 插件工具到宿主 Registry（调用转发 Node invoke）。
func (b *nodeBridge) handleToolMsg(plugin string, defRaw json.RawMessage) {
	var def struct {
		Name        string         `json:"name"`
		Description string         `json:"description"`
		Parameters  map[string]any `json:"parameters"`
		Category    string         `json:"category"`
		ReadOnly    bool           `json:"readOnly"`
	}
	if err := json.Unmarshal(defRaw, &def); err != nil || def.Name == "" {
		log.Printf("[node-bridge] 工具定义异常: %v", err)
		return
	}
	b.mu.Lock()
	ph := b.ph
	b.mu.Unlock()
	if ph == nil {
		log.Printf("[node-bridge] 宿主未就绪，忽略工具 %s", def.Name)
		return
	}
	tool := &Tool{
		Name:        def.Name,
		Description: def.Description,
		Parameters:  def.Parameters,
		Category:    def.Category,
		ReadOnly:    def.ReadOnly,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			return b.invokeTool(ctx, def.Name, args)
		},
	}
	b.toolsMu.Lock()
	b.tools[def.Name] = tool
	b.toolsMu.Unlock()
	if err := ph.Context().RegisterTool(tool); err != nil {
		log.Printf("[node-bridge] 注册工具 %s 失败: %v", def.Name, err)
	}
	// ★ 2026-08-17：装载 ≠ agent 可用——Node 桥插件工具同样受工作区工具集
	//   可见性收敛（不在工具集白名单 → 对 agent 隐藏，cordis/前端仍可见）。
	ph.hideToolIfNotInToolset(def.Name)
}

// invokeTool 调用 Node 侧插件工具（发送 invoke 消息并等待结果）。
// ★ 2026-08-23 工作区隔离：payload 附带会话绑定的工作区根（ctx 链提取），
//   Node 侧插件可用（ctx.fs 等沙箱服务按根解析）；无会话时为空。
func (b *nodeBridge) invokeTool(ctx context.Context, tool string, args map[string]any) (string, error) {
	if !b.isReady() {
		return "", fmt.Errorf("node 桥未就绪（工具 %s 不可用）", tool)
	}
	b.mu.Lock()
	b.seq++
	id := b.seq
	ch := make(chan bridgeResult, 1)
	b.pending[id] = ch
	b.mu.Unlock()

	wsRoot := SessionWorkspaceRoot(ctx)
	payload, _ := json.Marshal(map[string]any{"t": "invoke", "id": id, "tool": tool, "args": args, "wsRoot": wsRoot})
	if err := b.sendLine(payload); err != nil {
		return "", fmt.Errorf("node 桥发送失败: %v", err)
	}
	select {
	case r := <-ch:
		if r.ok {
			return r.data, nil
		}
		return "", fmt.Errorf("node 插件工具 %s 执行失败: %s", tool, r.error)
	case <-time.After(3 * time.Minute):
		return "", fmt.Errorf("node 桥调用 %s 超时（3 分钟）", tool)
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

// handleServiceMsg Node 插件请求服务（ctx.fs/web/bash）→ 转发 Go 侧工具。
func (b *nodeBridge) handleServiceMsg(id int64, svcName, method string, argsRaw json.RawMessage) {
	log.Printf("[node-bridge:diag] service id=%d svc=%s method=%s args=%s", id, svcName, method, string(argsRaw))
	args := map[string]any{}
	if len(argsRaw) > 0 {
		_ = json.Unmarshal(argsRaw, &args)
	}
	toolName, mappedArgs, direct, err := mapBridgeService(svcName, method, args)
	log.Printf("[node-bridge:diag] service id=%d → tool=%s err=%v", id, toolName, err)
	if err == nil && direct != nil {
		data, derr := direct()
		if derr != nil {
			err = derr
		} else {
			b.sendResult(id, true, data, "")
			return
		}
	}
	if err != nil {
		b.sendResult(id, false, "", err.Error())
		return
	}
	argsJSON, _ := json.Marshal(mappedArgs)
	var result string
	if b.ph != nil {
		result, err = b.ph.Context().Tools.Execute(context.Background(), toolName, string(argsJSON))
		log.Printf("[node-bridge:diag] service id=%d Execute(%s) → result=%q err=%v", id, toolName, result, err)
	} else {
		err = fmt.Errorf("宿主未就绪")
	}
	if err != nil {
		b.sendResult(id, false, "", err.Error())
		return
	}
	b.sendResult(id, true, result, "")
}

// sendResult 回发 service/invoke 结果。
func (b *nodeBridge) sendResult(id int64, ok bool, data, errMsg string) {
	payload, _ := json.Marshal(map[string]any{"t": "result", "id": id, "ok": ok, "data": data, "error": errMsg})
	if err := b.sendLine(payload); err != nil {
		log.Printf("[node-bridge] 回发结果失败: %v", err)
	}
}

// mapBridgeService 把 (svc, method, args) 映射到 Go 侧工具。
// 返回 (工具名, 映射后参数, 直连处理器, 错误)。
func mapBridgeService(svcName, method string, args map[string]any) (string, map[string]any, func() (string, error), error) {
	// ★ 2026-08-27 Node 桥能力扩展：ctx.store（消息读写落盘）+ ctx.loop（循环状态/上下文快照）
	//   ——Node 插件可参与上下文处理与数据落盘逻辑（改变 agentloop 行为的基础）。
	switch svcName {
	case "store":
		convID := argStr(args, "convId")
		if convID == "" {
			return "", nil, nil, fmt.Errorf("store.%s 缺少 convId", method)
		}
		root := argStr(args, "workspaceRoot")
		if root == "" {
			root = npmPluginProjectRoot()
		}
		switch method {
		case "read":
			return "", nil, func() (string, error) {
				if nodeBridgeManager == nil {
					return "", fmt.Errorf("会话管理器未注入（host 未就绪）")
				}
				st := nodeBridgeManager.StoreFor(root)
				if st == nil {
					return "", fmt.Errorf("工作区 %s 无会话存储", root)
				}
				msgs, err := st.LoadAll(convID)
				if err != nil {
					return "", err
				}
				out := make([]map[string]any, 0, len(msgs))
				for _, m := range msgs {
					out = append(out, map[string]any{"role": m.Role, "content": m.Content, "name": m.Name, "toolCallId": m.ToolCallID})
				}
				b, _ := json.Marshal(out)
				return string(b), nil
			}, nil
		case "append":
			role := argStr(args, "role")
			content := argStr(args, "content")
			if role == "" {
				return "", nil, nil, fmt.Errorf("store.append 缺少 role")
			}
			return "", nil, func() (string, error) {
				if nodeBridgeManager == nil {
					return "", fmt.Errorf("会话管理器未注入（host 未就绪）")
				}
				st := nodeBridgeManager.StoreFor(root)
				if st == nil {
					return "", fmt.Errorf("工作区 %s 无会话存储", root)
				}
				msg := Message{Role: Role(role), Content: content}
				if err := st.AppendMessage(convID, msg, nil); err != nil {
					return "", err
				}
				return "ok", nil
			}, nil
		}
		return "", nil, nil, fmt.Errorf("未知 store 服务方法: %s", method)
	case "loop":
		switch method {
		case "info":
			return "", nil, func() (string, error) {
				if nodeBridgeManager == nil {
					return "", fmt.Errorf("会话管理器未注入（host 未就绪）")
				}
				running := nodeBridgeManager.ListRunning()
				b, _ := json.Marshal(map[string]any{"running": running, "active": fmt.Sprint(running)})
				return string(b), nil
			}, nil
		case "snapshot":
			convID := argStr(args, "convId")
			if convID == "" {
				return "", nil, nil, fmt.Errorf("loop.snapshot 缺少 convId")
			}
			root := argStr(args, "workspaceRoot")
			if root == "" {
				root = npmPluginProjectRoot()
			}
			return "", nil, func() (string, error) {
				if nodeBridgeManager == nil {
					return "", fmt.Errorf("会话管理器未注入（host 未就绪）")
				}
				hist := nodeBridgeManager.GetCurrentHistory(convID)
				summaries := nodeBridgeManager.GetCurrentCompressedSummaries(convID)
				running := nodeBridgeManager.IsRunning(convID)
				out := make([]map[string]any, 0, len(hist))
				for _, m := range hist {
					out = append(out, map[string]any{"role": m.Role, "content": m.Content})
				}
				b, _ := json.Marshal(map[string]any{
					"convId": convID, "running": running,
					"messages": out, "summaries": summaries,
				})
				return string(b), nil
			}, nil
		}
		return "", nil, nil, fmt.Errorf("未知 loop 服务方法: %s", method)
	}
	switch svcName {
	case "fs":
		p := argStr(args, "path")
		switch method {
		// ★ 2026-09 Round2（R2-9）：旧工具名（read_file/write_file/run_command/
		//   list_files）在宿主生产注册面已不存在（磁盘插件 tool-harness/tool-shell
		//   承载新名）——桥映射同步到 harness 命名；fs.list 无对应工具，改直连。
		// ★ t4 F3（2026-09 t5）：R2-7 后 read 输出形态为 DSH 行号块
		//   （<path>/<type>/<content> + "N: text" + total footer）——npm 插件
		//   ctx.fs.read 拿到的不是原始文件文本；当前仓库无 ctx.fs.read 消费者
		//   （仅注释示例），若第三方解析型插件出现需在桥接层剥离包装（前端专项）。
		case "read":
			return "read", map[string]any{"path": p, "offset": args["offset"], "limit": args["limit"]}, nil, nil
		case "write":
			return "write", map[string]any{"path": p, "content": args["content"]}, nil, nil
		case "list":
			return "", nil, func() (string, error) {
				root := npmPluginProjectRoot()
				abs, err := resolvePath(root, p)
				if err != nil {
					return "", err
				}
				entries, err := os.ReadDir(abs)
				if err != nil {
					return "", err
				}
				names := make([]string, 0, len(entries))
				for _, e := range entries {
					names = append(names, e.Name())
				}
				b, _ := json.Marshal(names)
				return string(b), nil
			}, nil
		case "exists":
			return "", nil, func() (string, error) {
				root := npmPluginProjectRoot()
				abs, err := resolvePath(root, p)
				if err != nil {
					return "", err
				}
				if _, err := os.Stat(abs); err == nil {
					return "true", nil
				}
				return "false", nil
			}, nil
		case "mkdir":
			return "", nil, func() (string, error) {
				root := npmPluginProjectRoot()
				abs, err := resolvePath(root, p)
				if err != nil {
					return "", err
				}
				if err := os.MkdirAll(abs, 0o755); err != nil {
					return "", err
				}
				return "ok", nil
			}, nil
		case "remove":
			return "", nil, func() (string, error) {
				root := npmPluginProjectRoot()
				abs, err := resolvePath(root, p)
				if err != nil {
					return "", err
				}
				if err := os.RemoveAll(abs); err != nil {
					return "", err
				}
				return "ok", nil
			}, nil
		}
		return "", nil, nil, fmt.Errorf("未知 fs 服务方法: %s", method)
	case "web":
		switch method {
		case "fetch":
			url := argStr(args, "url")
			if url == "" {
				return "", nil, nil, fmt.Errorf("web.fetch 缺少 url")
			}
			return "web_fetch", map[string]any{"url": url}, nil, nil
		case "search":
			return "web_search", map[string]any{"query": argStr(args, "query")}, nil, nil
		}
		return "", nil, nil, fmt.Errorf("未知 web 服务方法: %s", method)
	case "bash":
		switch method {
		case "exec":
			// R2-9：run_command → bash（tool-harness 插件承载）
			return "bash", map[string]any{"command": argStr(args, "command")}, nil, nil
		}
		return "", nil, nil, fmt.Errorf("未知 bash 服务方法: %s", method)
	}
	return "", nil, nil, fmt.Errorf("未知服务: %s", svcName)
}

// sendLine 写一行 JSON 到 node stdin（并发安全）。
func (b *nodeBridge) sendLine(payload []byte) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.stdin == nil {
		return fmt.Errorf("node 桥未启动")
	}
	if _, err := b.stdin.Write(payload); err != nil {
		return err
	}
	if err := b.stdin.WriteByte('\n'); err != nil {
		return err
	}
	return b.stdin.Flush()
}

// Close 关闭 Node 桥进程（卸载工具、kill 子进程）。
func (b *nodeBridge) Close() {
	b.mu.Lock()
	b.closed = true
	b.ready = false
	cmd := b.cmd
	b.cmd = nil
	b.stdin = nil
	b.mu.Unlock()
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}
}

// closeGlobalNodeBridge 进程退出/宿主重置时清理（companion 退出钩子调用）。
func closeGlobalNodeBridge() {
	if globalNodeBridge != nil {
		globalNodeBridge.Close()
		globalNodeBridge = nil
	}
}

// bridgeLoadedPlugins 返回桥当前装载的插件列表（plugins.json 内容，无桥则空）。
func bridgeLoadedPlugins() []string {
	b := globalNodeBridge
	if b == nil {
		return nil
	}
	doc, err := readNodePluginsFile(filepath.Join(b.dir, "plugins.json"))
	if err != nil {
		return nil
	}
	return doc.Plugins
}
