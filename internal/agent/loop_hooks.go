// loop_hooks.go — 钩子系统接入 Loop（t1 报告 L2 缺口闭环）
//
// ★ 背景（2026-09）：internal/hook（PreToolUse/PostToolUse/UserPromptSubmit/Stop
//   钩子 Runner/信任门，移植自 DeepSeek-Reasonix）生产代码零导入——「移植了但
//   从未接线」。本文件把钩子系统接入 agent 执行链路：
//
//   1. 配置钩子（host 能力）：项目 <root>/.pair/settings.json（trusted）+ 全局
//      ~/.pair/settings.json 的 "hooks" 配置，经 internal/hook.Load 装载；
//      PreToolUse 阻塞语义（exit 2 / 超时 → 工具被拦截并回灌反馈）。
//   2. 插件钩子（插件优先）：Go 插件经 RegisterLoopHook、JS 插件经
//      ctx.hooks.register(event, fn) 注册程序化钩子（与配置钩子同事件表，
//      按注册顺序执行；任一 block 即拦截）。
//
// 接线点：
//   - Registry.Execute：PreToolUse（执行前门）+ PostToolUse（执行后观察）——
//     覆盖 Go 循环与 JS 循环（agentloop）两条执行路径；
//   - Loop.Run：UserPromptSubmit（轮次开始门）+ Stop（轮次结束，defer 兜底）。
//
// 安全语义：未调用 SetLoopHookRoots（生产 web/desktop 启动时调用）且无插件
// 注册钩子时全部 no-op——测试与无配置环境零行为变化。
//
// 事件名与 internal/hook 对齐："PreToolUse" / "PostToolUse" /
// "UserPromptSubmit" / "Stop"。

package agent

import (
	"context"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hoonfeng/paircode/internal/core"
	"github.com/hoonfeng/paircode/internal/hook"
)

// ─── 配置钩子装载（host 能力）──────────────────────────────

var (
	loopHookMu     sync.RWMutex
	loopHookRoots  string              // 最近一次 SetLoopHookRoots 的 projectRoot（变化重载）
	loopHookTrust  bool                // 最近一次装载的项目钩子信任标记（变化重载）
	loopHookLoaded []hook.ResolvedHook // 已装载的配置钩子（project + global）
	loopHooksProg  []programmaticHook  // 插件注册的程序化钩子
	loopHookSeq    int64               // 程序化钩子注册序号（撤销匹配用）
)

// programmaticHook 插件注册的钩子（Go/JS 统一形态）。
type programmaticHook struct {
	id    int64
	event string
	// fn 接收 payload，返回拦截裁决。block=true → 事件门拦截（仅门事件有效）。
	fn func(payload map[string]any) (block bool, feedback string)
}

// SetLoopHookRoots 设置钩子配置根并装载配置钩子（生产入口：web/desktop 启动时
// 调用一次；projectRoot 变更自动重载）。未调用 = 不装载配置钩子（全部 no-op）。
// ★ 信任门控（t4 审查 M2 修复）：项目钩子（<root>/.pair/settings.json）默认
// **不装载**——web 监听全接口，打开恶意工作区不应自动执行其钩子 shell 命令；
// 全局钩子（~/.pair/settings.json）不受信任门影响（用户本机自配置）。
// 显式信任项目钩子：SetLoopHookRootsTrusted / InitLoopHooks 环境变量开关。
func SetLoopHookRoots(projectRoot, homeDir string) {
	setLoopHookRoots(projectRoot, homeDir, false)
}

// SetLoopHookRootsTrusted 显式信任门控版本：trustedProject=true 时装载项目钩子
// （测试/受控环境用；生产默认走 SetLoopHookRoots 的不信任语义）。
func SetLoopHookRootsTrusted(projectRoot, homeDir string, trustedProject bool) {
	setLoopHookRoots(projectRoot, homeDir, trustedProject)
}

func setLoopHookRoots(projectRoot, homeDir string, trustedProject bool) {
	loopHookMu.Lock()
	defer loopHookMu.Unlock()
	if projectRoot == loopHookRoots && trustedProject == loopHookTrust {
		return
	}
	loopHookRoots = projectRoot
	loopHookTrust = trustedProject
	// 项目钩子仅受信任时装载（信任门）；全局钩子恒装载
	loaded := hook.Load(hook.LoadOptions{ProjectRoot: projectRoot, HomeDir: homeDir, Trusted: trustedProject})
	if len(loaded) > 0 {
		log.Printf("[hooks] 已装载 %d 个配置钩子（project=%q home=%q trusted=%v）", len(loaded), projectRoot, homeDir, trustedProject)
	}
	loopHookLoaded = loaded
}

// RegisterLoopHook 注册程序化钩子（Go 插件/宿主代码用；事件名见 hook.Events）。
// 返回撤销函数（插件卸载时调用）。
func RegisterLoopHook(event string, fn func(payload map[string]any) (bool, string)) (restore func()) {
	loopHookMu.Lock()
	loopHookSeq++
	id := loopHookSeq
	loopHooksProg = append(loopHooksProg, programmaticHook{id: id, event: event, fn: fn})
	loopHookMu.Unlock()
	return func() {
		loopHookMu.Lock()
		defer loopHookMu.Unlock()
		for i := range loopHooksProg {
			if loopHooksProg[i].id == id {
				loopHooksProg = append(loopHooksProg[:i], loopHooksProg[i+1:]...)
				return
			}
		}
	}
}

// ─── 事件触发 ───────────────────────────────────────────────

// loopHookCurrentTurn 当前活动 Loop 的 turn 号（t4 L2 修复：hook payload 透传
// 真实轮次）。Loop.openTurn/endTurn 维护；钩子触发点（Registry.Execute 等）
// 无 Loop 引用，经此全局取号。多 Loop 并发时取最近打开的轮次（尽力而为）。
var loopHookCurrentTurn atomic.Int64

// hookPayloadOf 组装标准 payload（对齐 internal/hook.Payload 的 JSON 形态）。
// turn<=0 时自动取当前活动 Loop 的轮次（原实现恒传 0，t4 L2 修复）。
func hookPayloadOf(event hook.Event, toolName, argsJSON, result, prompt, cwd string, turn int) hook.Payload {
	if turn <= 0 {
		turn = int(loopHookCurrentTurn.Load())
	}
	p := hook.Payload{
		Event: event,
		Cwd:   cwd,
		Turn:  turn,
	}
	switch event {
	case hook.PreToolUse, hook.PostToolUse:
		p.ToolName = toolName
		p.ToolArgs = []byte(argsJSON)
		p.ToolResult = result
	case hook.UserPromptSubmit:
		p.Prompt = prompt
	case hook.Stop:
		p.Message = "agent 轮次结束"
	}
	return p
}

// hookCwd 取钩子 payload 的 cwd（当前主工作区；空则沿用进程 cwd）。
func hookCwd() string {
	if r := core.Root(); r != "" {
		return r
	}
	return ""
}

// fireHookEvent 执行某事件的全部钩子（配置钩子 + 程序化钩子）。
// 返回是否被拦截（仅门事件）与拦截反馈。
func fireHookEvent(ctx context.Context, event hook.Event, payload hook.Payload) (blocked bool, feedback string) {
	loopHookMu.RLock()
	configs := append([]hook.ResolvedHook(nil), loopHookLoaded...)
	progs := append([]programmaticHook(nil), loopHooksProg...)
	loopHookMu.RUnlock()

	// ① 配置钩子（internal/hook Runner；门事件阻塞语义）
	report := hook.Run(ctx, payload, configs, nil)
	if report.Blocked {
		var msgs []string
		for _, o := range report.Outcomes {
			if o.Decision == hook.DecisionBlock {
				msgs = append(msgs, o.Stdout)
			}
		}
		fb := ""
		if len(msgs) > 0 {
			fb = msgs[0]
		}
		if fb == "" {
			fb = "钩子拦截（PreToolUse 退出码 2 或超时）"
		}
		return true, fb
	}

	// ② 程序化钩子（插件注册；按注册顺序，任一 block 即拦截）
	payloadMap := map[string]any{
		"event":      string(payload.Event),
		"cwd":        payload.Cwd,
		"turn":       payload.Turn,
		"toolName":   payload.ToolName,
		"toolArgs":   string(payload.ToolArgs),
		"toolResult": payload.ToolResult,
		"prompt":     payload.Prompt,
		"message":    payload.Message,
	}
	for _, h := range progs {
		if h.event != string(event) {
			continue
		}
		b, fb := h.fn(payloadMap)
		if b {
			if fb == "" {
				fb = "插件钩子拦截"
			}
			return true, fb
		}
	}
	return false, ""
}

// firePreToolUseHooks PreToolUse 门（工具执行前）。
// blocked=true 时调用方应拦截工具并把 feedback 回灌 LLM。
func firePreToolUseHooks(ctx context.Context, toolName, argsJSON string) (bool, string) {
	loopHookMu.RLock()
	noConfig := len(loopHookLoaded) == 0
	noProg := len(loopHooksProg) == 0
	loopHookMu.RUnlock()
	if noConfig && noProg {
		return false, ""
	}
	if ctx == nil {
		ctx = context.Background() // Registry.Execute 可能收到 nil ctx（防御）
	}
	return fireHookEvent(ctx, hook.PreToolUse, hookPayloadOf(hook.PreToolUse, toolName, argsJSON, "", "", hookCwd(), 0))
}

// firePostToolUseHooks PostToolUse 观察（工具执行后；不拦截）。
func firePostToolUseHooks(ctx context.Context, toolName, argsJSON, result string, toolErr error) {
	loopHookMu.RLock()
	noConfig := len(loopHookLoaded) == 0
	noProg := len(loopHooksProg) == 0
	loopHookMu.RUnlock()
	if noConfig && noProg {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if toolErr != nil {
		result = "Error: " + toolErr.Error()
	}
	_, _ = fireHookEvent(ctx, hook.PostToolUse, hookPayloadOf(hook.PostToolUse, toolName, argsJSON, result, "", hookCwd(), 0))
}

// fireUserPromptSubmitHooks UserPromptSubmit 门（轮次开始前）。
func fireUserPromptSubmitHooks(ctx context.Context, prompt string) (bool, string) {
	loopHookMu.RLock()
	noConfig := len(loopHookLoaded) == 0
	noProg := len(loopHooksProg) == 0
	loopHookMu.RUnlock()
	if noConfig && noProg {
		return false, ""
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return fireHookEvent(ctx, hook.UserPromptSubmit, hookPayloadOf(hook.UserPromptSubmit, "", "", "", prompt, hookCwd(), 0))
}

// fireStopHooks Stop 观察（轮次结束；defer 兜底所有返回路径）。
func fireStopHooks(ctx context.Context) {
	loopHookMu.RLock()
	noConfig := len(loopHookLoaded) == 0
	noProg := len(loopHooksProg) == 0
	loopHookMu.RUnlock()
	if noConfig && noProg {
		return
	}
	// Stop 钩子允许更宽的超时（内部 hook 默认 30s）；ctx 取消时不再等
	cctx, cancel := context.WithTimeout(ctx, 35*time.Second)
	defer cancel()
	_, _ = fireHookEvent(cctx, hook.Stop, hookPayloadOf(hook.Stop, "", "", "", "", hookCwd(), 0))
}

// ─── 诊断 ───────────────────────────────────────────────────

// LoopHooksStatus 钩子系统状态摘要（诊断/插件面板用）。
func LoopHooksStatus() map[string]any {
	loopHookMu.RLock()
	defer loopHookMu.RUnlock()
	configs := 0
	byEvent := map[string]int{}
	for _, h := range loopHookLoaded {
		configs++
		byEvent[string(h.Event)]++
	}
	prog := len(loopHooksProg)
	return map[string]any{
		"configHooks": configs,
		"pluginHooks": prog,
		"byEvent":     byEvent,
		"projectRoot": loopHookRoots,
		"trusted":     loopHookTrust,
	}
}

// homeDirOf 取用户主目录（钩子全局配置定位）。
func homeDirOf() string {
	if h, err := os.UserHomeDir(); err == nil {
		return h
	}
	return ""
}

// InitLoopHooks 生产入口便捷封装：以当前主工作区根 + 系统主目录装载配置钩子
// （web/desktop 启动时调用；未调用 = 不装载配置钩子，全部 no-op）。
// ★ 信任门控（t4 M2 修复）：项目钩子默认不装载；设置环境变量
// PAIRCODE_TRUST_PROJECT_HOOKS=1 显式信任当前工作区后装载（全局钩子恒装载）。
func InitLoopHooks() {
	trusted := os.Getenv("PAIRCODE_TRUST_PROJECT_HOOKS") == "1"
	SetLoopHookRootsTrusted(core.Root(), homeDirOf(), trusted)
}
