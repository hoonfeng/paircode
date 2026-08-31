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
//     （read/write/glob/web_fetch/bash），
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
	// ★ Round4 repair（t6）：epoch 代数计数——Close() 递增，start() 携带
	//   启动时快照校验；epoch 过期（启动期间被 Close/替换）→ 放弃进程防
	//   「旧进程僵尸 + globalNodeBridge 被过期启动覆盖」竞态。
	epoch   int
	toolsMu sync.Mutex
	tools   map[string]*Tool // 已注册到宿主的桥工具（新宿主补注册用）
	// ★ t5 集成：桥工具归属名（toolName → "node-bridge:<插件名>"；claimTool
	//   冲突报错信息完整 + 无冲突时正确登记归属）。受 toolsMu 保护。
	toolOwner map[string]string
	// ★ 并存与切换（2026-08-31，取消同名工具覆盖）：受 toolsMu 保护。
	//   specs      pkgName → npm spec（ready 消息上报的已装载清单；面板显示版本）
	//   conflicts  toolName → 同名工具并存记录（repo 版 ↔ 外部版，active 标生效方）
	specs     map[string]string
	conflicts map[string]*bridgeToolConflict
	// ★ Round4：host 事件订阅白名单（插件 ctx.on 声明的事件名；按名转发防风暴）
	subs map[string]bool
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
// ★ t5 集成：按工具归属名（node-bridge:<插件名>）注册——冲突报错信息完整，
//   无冲突时正确登记归属（与 goja 插件 claimTool 语义一致）。
// ★ 并存（2026-08-31）：与 repo 移植版同名冲突时不再接管/停用对方，
//   记录并存关系（repo 优先生效，桥工具挂起，面板可切换）。
func (b *nodeBridge) registerToolsTo(ph *PluginHost) {
	if ph == nil {
		return
	}
	b.toolsMu.Lock()
	tools := make([]*Tool, 0, len(b.tools))
	owners := make([]string, 0, len(b.tools))
	for _, t := range b.tools {
		tools = append(tools, t)
		owners = append(owners, b.toolOwner[t.Name])
	}
	b.toolsMu.Unlock()
	for i, t := range tools {
		owner := owners[i]
		if owner == "" {
			owner = "node-bridge"
		}
		if err := ph.Context().forPlugin(owner).RegisterTool(t); err != nil {
			if !b.shadowConflictingTool(ph, owner, t) {
				log.Printf("[node-bridge] 补注册工具 %s 到新宿主失败: %v", t.Name, err)
			}
		}
		// ★ 2026-08-17：装载 ≠ agent 可用——Node 桥插件工具同样受工作区工具集
		//   可见性收敛（不在工具集白名单 → 对 agent 隐藏，cordis/前端仍可见）。
		ph.hideToolIfNotInToolset(t.Name)
	}
}

// isStale 判断 start 的 epoch 快照是否已过期（期间发生 Close/替换）。
func (b *nodeBridge) isStale(epoch int) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.epoch != epoch
}

// start 启动 node 子进程并等待 ready。
func (b *nodeBridge) start(nodePath string) error {
	// ★ Round4 repair（t6）：启动前快照 epoch；启动期间若 Close()（并发
	//   ensureNodeBridge 替换/宿主重置）递增 epoch → 放弃本进程（kill），
	//   防止旧进程继续存活与过期启动覆盖 globalNodeBridge。
	b.mu.Lock()
	myEpoch := b.epoch
	b.mu.Unlock()
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
		if b.isStale(myEpoch) { // 启动期间被 Close/替换 → 放弃本进程
			_ = cmd.Process.Kill()
			_, _ = cmd.Process.Wait()
			return fmt.Errorf("node 桥启动被取消（epoch 已过期，桥已关闭/替换）")
		}
		if b.isReady() {
			// ★ Round4 repair（t6）：启动成功后清零崩溃重启计数
			//   （连续崩溃计数只统计「启动失败/就绪前退出」段）。
			b.mu.Lock()
			b.restarts = 0
			b.mu.Unlock()
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
			Events  []string        `json:"events"` // Round4：插件事件订阅清单
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
			b.noteLoadedSpecs(msg.Plugins) // ★ 并存展示：已装载 spec → 插件面板版本/状态
			log.Printf("[node-bridge] 就绪（插件 %v，工具 %v）", msg.Plugins, msg.Tool)
		case "tool":
			b.handleToolMsg(msg.Plugin, msg.Def)
		case "service":
			b.handleServiceMsg(msg.ID, msg.Svc, msg.Method, msg.Args, msg.Plugin)
		case "subscribe":
			b.handleSubscribeMsg(msg.Plugin, msg.Events)
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
	if b.toolOwner == nil {
		b.toolOwner = map[string]string{}
	}
	owner := "node-bridge:" + plugin
	b.toolOwner[def.Name] = owner
	b.toolsMu.Unlock()
	if err := ph.Context().forPlugin(owner).RegisterTool(tool); err != nil {
		if !b.shadowConflictingTool(ph, owner, tool) {
			log.Printf("[node-bridge] 注册工具 %s 失败: %v", def.Name, err)
		}
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
	// ★ Round4：外部插件工具需要调用方会话身份（exec.agent.id / session.header.cwd）
	payload, _ := json.Marshal(map[string]any{"t": "invoke", "id": id, "tool": tool, "args": args, "wsRoot": wsRoot, "convId": SessionConvID(ctx)})
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

// handleServiceMsg Node 插件请求服务（ctx.fs/web/bash + Round4 外部服务面）
// → 转发 Go 侧工具 / 直连处理器。
func (b *nodeBridge) handleServiceMsg(id int64, svcName, method string, argsRaw json.RawMessage, plugin string) {
	log.Printf("[node-bridge:diag] service id=%d svc=%s method=%s plugin=%s", id, svcName, method, plugin)
	args := map[string]any{}
	if len(argsRaw) > 0 {
		_ = json.Unmarshal(argsRaw, &args)
	}
	// ★ Round4 repair（t6）：先锁内取 ph 快照再传入下游——dshService 与
	//   subAgentSpecFromArgs 共用同一快照（杜绝与 bindHost 的 b.ph 写并发
	//   数据竞争；也保证同一请求内 agents.start 的 spec.WsRoot 回退与
	//   工具 Execute 落在同一宿主上）。
	b.mu.Lock()
	ph := b.ph
	b.mu.Unlock()
	// ★ Round4：外部服务面（agents/subagents/llm/systemPrompt/commands/logger）
	if handled, data, err := b.dshService(svcName, method, args, plugin, ph); handled {
		if err != nil {
			b.sendResult(id, false, "", err.Error())
		} else {
			b.sendResult(id, true, data, "")
		}
		return
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
	if ph != nil {
		result, err = ph.Context().Tools.Execute(context.Background(), toolName, string(argsJSON))
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

// ─── Round4 外部服务面（cordis4 轨插件 ctx.agents/subagents/llm/systemPrompt/
// commands 的门面后端）────────────────────────────────────────
// 直接映射现有 Go 能力（SubAgentRegistry / 模型目录 / PluginHost 段 / 命令表），
// 与 goja 轨 ctx.agents/ctx.llm（jsplugin_agents.go）同源，行为一致。
// 返回 (handled, data, err)：handled=false 表示非 外部服务（走 mapBridgeService）。

// handleSubscribeMsg 记录插件事件订阅白名单（agent/status 等按名转发）。
func (b *nodeBridge) handleSubscribeMsg(plugin string, events []string) {
	b.mu.Lock()
	if b.subs == nil {
		b.subs = map[string]bool{}
	}
	for _, e := range events {
		if e = strings.TrimSpace(e); e != "" {
			b.subs[e] = true
		}
	}
	b.mu.Unlock()
	log.Printf("[node-bridge] 插件 %s 订阅事件 %v", plugin, events)
}

// bridgeHasSubscribers 是否有插件订阅了指定事件名。
func (b *nodeBridge) bridgeHasSubscribers(name string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.subs != nil && b.subs[name]
}

// emitBridgeEvent 把宿主事件转发给 Node 桥（仅当有插件订阅；防协议风暴）。
// 订阅白名单外的事件零开销（一次锁 + map 查询）。
func emitBridgeEvent(name string, payload any) {
	b := globalNodeBridge
	if b == nil || !b.isReady() || !b.bridgeHasSubscribers(name) {
		return
	}
	line, err := json.Marshal(map[string]any{"t": "event", "name": name, "payload": payload})
	if err != nil {
		log.Printf("[node-bridge] 事件 %s 序列化失败: %v", name, err)
		return
	}
	if err := b.sendLine(line); err != nil {
		log.Printf("[node-bridge] 事件 %s 转发失败: %v", name, err)
	}
}

// dshService 外部服务面实现（Node 插件 ctx.agents/subagents/llm/systemPrompt/commands）。
// ★ Round4 repair（t6）：ph 由 handleServiceMsg 锁内快照传入（与
//   subAgentSpecFromArgs 共用同一宿主快照），调用方不得传 b.ph。
func (b *nodeBridge) dshService(svcName, method string, args map[string]any, plugin string, ph *PluginHost) (handled bool, data string, err error) {
	switch svcName {
	case "agents":
		convID := argStr(args, "convId")
		switch method {
		case "get":
			rec := SubAgentInfo(convID)
			if rec == nil {
				return true, "null", nil
			}
			return true, bridgeJSON(map[string]any{
				"convId": rec.ConvID, "wsRoot": rec.WsRoot, "state": rec.State,
				"label": rec.Label, "team": rec.Team, "member": rec.Member,
				"parentConvId": rec.ParentConv, "report": rec.Report,
			}), nil
		case "list":
			recs := ListSubAgents("", "")
			out := make([]any, 0, len(recs))
			for _, rec := range recs {
				out = append(out, map[string]any{
					"convId": rec.ConvID, "wsRoot": rec.WsRoot, "state": rec.State,
					"label": rec.Label, "team": rec.Team, "member": rec.Member,
					"parentConvId": rec.ParentConv, "report": rec.Report,
				})
			}
			return true, bridgeJSON(out), nil
		case "status":
			rec := SubAgentInfo(convID)
			if rec == nil {
				return true, "null", nil
			}
			return true, bridgeJSON(map[string]any{
				"convId": rec.ConvID, "wsRoot": rec.WsRoot, "state": rec.State,
				"label": rec.Label, "team": rec.Team, "member": rec.Member,
				"parentConvId": rec.ParentConv, "turns": rec.Turns, "report": rec.Report,
			}), nil
		case "running":
			if convID == "" {
				return true, "false", nil
			}
			if rec := SubAgentInfo(convID); rec != nil {
				return true, fmt.Sprint(rec.State == "running"), nil
			}
			if mgr := GlobalSessionManager(); mgr != nil {
				return true, fmt.Sprint(mgr.IsRunning(convID)), nil
			}
			return true, "false", nil
		case "ready":
			return true, fmt.Sprint(SubAgentSpawnerReady()), nil
		case "start":
			spec, serr := subAgentSpecFromArgs(args, ph)
			if serr != nil {
				return true, "", serr
			}
			rec, serr := SpawnSubAgent(spec)
			if serr != nil {
				return true, "", serr
			}
			return true, bridgeJSON(map[string]any{"convId": rec.ConvID, "state": rec.State, "label": rec.Label}), nil
		case "fork":
			spec, serr := subAgentSpecFromArgs(args, ph)
			if serr != nil {
				return true, "", serr
			}
			spec.ForkOf = argStr(args, "forkFrom")
			rec, serr := ForkSubAgent(spec)
			if serr != nil {
				return true, "", serr
			}
			return true, bridgeJSON(map[string]any{"convId": rec.ConvID, "state": rec.State}), nil
		case "followup", "inject", "steer":
			if convID == "" {
				return true, "", fmt.Errorf("agents.%s 缺少 convId", method)
			}
			text := argStr(args, "text")
			if text == "" {
				return true, "", fmt.Errorf("agents.%s 缺少 text", method)
			}
			queued, serr := FollowupSubAgent(convID, text)
			if serr != nil {
				return true, "", serr
			}
			return true, bridgeJSON(map[string]any{"ok": true, "queued": queued, "convId": convID}), nil
		case "cancel", "stop":
			if convID == "" {
				return true, "", fmt.Errorf("agents.%s 缺少 convId", method)
			}
			if serr := StopSubAgent(convID); serr != nil {
				return true, "", serr
			}
			return true, "true", nil
		case "report":
			if convID == "" {
				return true, "", fmt.Errorf("agents.report 缺少 convId")
			}
			if serr := ReportSubAgent(convID, argStr(args, "text")); serr != nil {
				return true, "", serr
			}
			return true, bridgeJSON(map[string]any{"ok": true, "convId": convID}), nil
		case "lastText":
			return true, SubAgentLastText(convID), nil
		}
		return true, "", fmt.Errorf("未知 agents 服务方法: %s", method)
	case "subagents":
		switch method {
		case "getProvider":
			if argStr(args, "name") == "spawn" {
				return true, bridgeJSON(map[string]any{
					"name": "spawn", "prepareContinuable": true,
					"capabilities": map[string]any{"persona": true, "toolFilter": true},
				}), nil
			}
			return true, "null", nil
		case "list":
			return true, bridgeJSON([]string{"spawn"}), nil
		case "startContinuable":
			spec := SubAgentSpec{
				Label:           argStr(args, "label"),
				Task:            argStr(args, "prompt"),
				System:          argStr(args, "persona"),
				ParentConv:      argStr(args, "parentConvId"),
				Provider:        argStr(args, "provider2"),
				Model:           argStr(args, "model"),
				DenyTools:       argStrSlice(args, "denyTools"),
				MaxIter:         0,
				ReasoningEffort: argStr(args, "reasoningEffort"),
			}
			if spec.WsRoot == "" {
				spec.WsRoot = npmPluginProjectRoot()
			}
			rec, serr := SpawnSubAgent(spec)
			if serr != nil {
				return true, "", serr
			}
			return true, bridgeJSON(map[string]any{"childId": rec.ConvID, "convId": rec.ConvID, "state": rec.State}), nil
		case "followup":
			childID := argStr(args, "childId")
			if childID == "" {
				return true, "", fmt.Errorf("subagents.followup 缺少 childId")
			}
			if _, serr := FollowupSubAgent(childID, argStr(args, "text")); serr != nil {
				return true, "", serr
			}
			return true, "true", nil
		case "interrupt":
			if serr := StopSubAgent(argStr(args, "childId")); serr != nil {
				return true, "", serr
			}
			return true, "true", nil
		}
		return true, "", fmt.Errorf("未知 subagents 服务方法: %s", method)
	case "llm":
		switch method {
		case "models":
			models := SubAgentModels()
			if models == nil {
				return true, "[]", nil
			}
			return true, bridgeJSON(models), nil
		case "current":
			cur := SubAgentCurrentModel()
			if cur == nil {
				return true, "{}", nil
			}
			return true, bridgeJSON(cur), nil
		case "listModels":
			provider := argStr(args, "provider")
			models := SubAgentModels()
			out := make([]any, 0)
			for _, m := range models {
				if provider != "" && m["provider"] != provider {
					continue
				}
				out = append(out, map[string]any{"id": m["model"], "name": m["label"]})
			}
			return true, bridgeJSON(out), nil
		case "resolveCallConfig":
			provider := argStr(args, "provider")
			model := argStr(args, "model")
			if provider == "" || model == "" {
				return true, "", fmt.Errorf("llm.resolveCallConfig 需要 provider 与 model")
			}
			return true, bridgeJSON(map[string]any{
				"provider": provider, "model": model,
				"reasoningEffort": argStr(args, "reasoningEffort"),
			}), nil
		}
		return true, "", fmt.Errorf("未知 llm 服务方法: %s", method)
	case "systemPrompt":
		switch method {
		case "section":
			name := argStr(args, "name")
			text := argStr(args, "text")
			if name == "" {
				return true, "", fmt.Errorf("systemPrompt.section 缺少 name")
			}
			ph := GetGlobalPluginHost()
			if ph == nil {
				return true, "", fmt.Errorf("systemPrompt.section: 插件宿主未就绪")
			}
			order := 100
			if v, ok := args["order"].(float64); ok {
				order = int(v)
			}
			ph.Context().AddSystemPromptSection(&PromptSection{Name: name, Order: order, Text: text})
			return true, "ok", nil
		}
		return true, "", fmt.Errorf("未知 systemPrompt 服务方法: %s", method)
	case "prompts":
		switch method {
		case "provide":
			name := argStr(args, "name")
			if name == "" {
				return true, "", fmt.Errorf("prompts.provide 缺少 name")
			}
			ProvidePrompt(name, argStr(args, "text"), "node:"+plugin)
			return true, "ok", nil
		case "remove":
			name := argStr(args, "name")
			if name == "" {
				return true, "", fmt.Errorf("prompts.remove 缺少 name")
			}
			RemovePrompt(name)
			return true, "ok", nil
		case "list":
			return true, bridgeJSON(PromptAssetsSnapshot()), nil
		}
		return true, "", fmt.Errorf("未知 prompts 服务方法: %s", method)
	case "commands":
		switch method {
		case "register":
			name := argStr(args, "name")
			if name == "" {
				return true, "", fmt.Errorf("commands.register 缺少 name")
			}
			owner := "node-bridge:" + plugin
			handler := func(ctx context.Context, cargs map[string]any) (string, error) {
				return runNodeCommand(name, cargs)
			}
			if serr := RegisterHostCommand(name, argStr(args, "description"), handler, owner); serr != nil {
				return true, "", serr
			}
			return true, "ok", nil
		case "unregister":
			UnregisterHostCommands("node-bridge:" + plugin)
			return true, "ok", nil
		case "list":
			return true, bridgeJSON(ListHostCommands()), nil
		case "run":
			name := argStr(args, "name")
			if name == "" {
				return true, "", fmt.Errorf("commands.run 缺少 name")
			}
			result, serr := RunHostCommand(name, args)
			if serr != nil {
				return true, "", serr
			}
			return true, result, nil
		}
		return true, "", fmt.Errorf("未知 commands 服务方法: %s", method)
	case "logger":
		// Node 侧 logger 已走 console 通道（t:log 上报），无需往返。
		return true, "ok", nil
	}
	return false, "", nil
}

// subAgentSpecFromArgs 从 agents.start/fork 服务参数构造 SubAgentSpec。
func subAgentSpecFromArgs(args map[string]any, ph *PluginHost) (SubAgentSpec, error) {
	spec := SubAgentSpec{
		ConvID:          argStr(args, "convId"),
		ParentConv:      argStr(args, "parentConvId"),
		Label:           argStr(args, "label"),
		Team:            argStr(args, "team"),
		Member:          argStr(args, "member"),
		Task:            argStr(args, "task"),
		System:          argStr(args, "system"),
		Model:           argStr(args, "model"),
		Provider:        argStr(args, "provider"),
		ReasoningEffort: argStr(args, "reasoningEffort"),
		WsRoot:          argStr(args, "wsRoot"),
		DenyTools:       argStrSlice(args, "denyTools"),
		MaxIter:         mapInt(args, "maxIterations"),
	}
	if spec.WsRoot == "" {
		if ph != nil && ph.Context() != nil && ph.Context().WorkspaceRoot != "" {
			spec.WsRoot = ph.Context().WorkspaceRoot
		} else {
			spec.WsRoot = npmPluginProjectRoot()
		}
	}
	if strings.TrimSpace(spec.Task) == "" {
		return spec, fmt.Errorf("agents.start 缺少 task（首轮输入）")
	}
	return spec, nil
}

// runNodeCommand 执行 Node 侧注册的插件命令 handler（cmdrun 消息，等待结果）。
func runNodeCommand(name string, args map[string]any) (string, error) {
	b := globalNodeBridge
	if b == nil || !b.isReady() {
		return "", fmt.Errorf("node 桥未就绪（命令 %s 不可用）", name)
	}
	b.mu.Lock()
	b.seq++
	id := b.seq
	ch := make(chan bridgeResult, 1)
	b.pending[id] = ch
	b.mu.Unlock()
	payload, _ := json.Marshal(map[string]any{"t": "cmdrun", "id": id, "name": name, "args": args})
	if err := b.sendLine(payload); err != nil {
		return "", fmt.Errorf("node 桥发送失败: %v", err)
	}
	select {
	case r := <-ch:
		if r.ok {
			return r.data, nil
		}
		return "", fmt.Errorf("插件命令 %s 执行失败: %s", name, r.error)
	case <-time.After(90 * time.Second):
		return "", fmt.Errorf("插件命令 %s 超时（90 秒）", name)
	}
}

// mustJSON 序列化（内部数据结构已知可序列化；失败回退空对象）。
func bridgeJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(b)
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
		// ★ 2026-09 Round2（R2-9）：旧工具名（read/write/bash/
		//   list_files）在宿主生产注册面已不存在（磁盘插件 tool-harness/tool-shell
		//   承载新名）——桥映射同步到 harness 命名；fs.list 无对应工具，改直连。
		// ★ t4 F3（2026-09 t5）：R2-7 后 read 输出形态为 外部行号块
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
			// R2-9：bash → bash（tool-harness 插件承载）
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
	// ★ Round4 repair（t6）：epoch 递增——运行中的 start（含重启路径）
	//   检测到过期即放弃进程，杜绝「Close 与 start 竞态 → 僵尸进程/
	//   过期启动覆盖 globalNodeBridge」。
	b.epoch++
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
	return doc.Specs()
}
