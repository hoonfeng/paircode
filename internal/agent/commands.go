// ═══════════════════════════════════════════════════════════════
// commands.go — 宿主 ctx.commands 命令面（Round3 ④.2 slash 命令）
//
// 对齐 DSH harness「slash 命令」语义：
//   - 插件经 ctx.commands.register({name, description, handler}) 注册命令；
//     插件卸载自动注销（按归属清理，无悬挂）
//   - HTTP 面：GET /api/commands（清单）+ POST /api/commands/run（执行；
//     结果以系统消息注入对话——前端输入框 "/" 菜单消费）
//   - 前端降级：无匹配命令时 "/" 输入原样发送（零破坏）
// ═══════════════════════════════════════════════════════════════

package agent

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/hoonfeng/paircode/goja"
)

// HostCommand 一条宿主命令（slash 命令注册表条目）。
type HostCommand struct {
	Name        string
	Description string
	Handler     func(ctx context.Context, args map[string]any) (string, error)
}

var (
	hostCmdMu      sync.RWMutex
	hostCommands   = map[string]*HostCommand{} // name → 命令
	hostCmdOwner   = map[string]string{}       // name → 归属插件（卸载自动注销）
	hostCmdOrder   []string                    // 注册顺序（稳定清单）
)

// RegisterHostCommand 注册宿主命令。owner 非空时按插件归属（同名重复注册覆盖；
// 插件卸载按 owner 批量注销）。
func RegisterHostCommand(name, description string, handler func(ctx context.Context, args map[string]any) (string, error), owner string) error {
	name = trimSlashCommand(name)
	if name == "" {
		return fmt.Errorf("命令名不能为空")
	}
	if handler == nil {
		return fmt.Errorf("命令 %s handler 不能为空", name)
	}
	hostCmdMu.Lock()
	defer hostCmdMu.Unlock()
	if _, exists := hostCommands[name]; !exists {
		hostCmdOrder = append(hostCmdOrder, name)
	}
	hostCommands[name] = &HostCommand{Name: name, Description: description, Handler: handler}
	hostCmdOwner[name] = owner
	return nil
}

// UnregisterHostCommands 注销某插件的全部命令（插件卸载时调用）。
func UnregisterHostCommands(owner string) {
	hostCmdMu.Lock()
	defer hostCmdMu.Unlock()
	var keep []string
	for _, n := range hostCmdOrder {
		if hostCmdOwner[n] == owner {
			delete(hostCommands, n)
			delete(hostCmdOwner, n)
			continue
		}
		keep = append(keep, n)
	}
	hostCmdOrder = keep
}

// ListHostCommands 命令清单（[{name, description, owner}]，注册顺序稳定）。
// ★ 2026-08-31：按需激活命令附加 onDemand/plugin 标（前端 slash 菜单可标注）。
func ListHostCommands() []map[string]any {
	hostCmdMu.RLock()
	defer hostCmdMu.RUnlock()
	onDemand := OnDemandCommandMapping() // 命令名 → 插件名（按需激活）
	out := make([]map[string]any, 0, len(hostCmdOrder))
	for _, n := range hostCmdOrder {
		if c, ok := hostCommands[n]; ok {
			item := map[string]any{
				"name":        c.Name,
				"description": c.Description,
				"owner":       hostCmdOwner[n],
			}
			if plugin, hit := onDemand[n]; hit {
				item["onDemand"] = true
				item["plugin"] = plugin
			}
			out = append(out, item)
		}
	}
	return out
}

// RunHostCommand 执行命令（args 由调用方解码；未知命令明确报错）。
func RunHostCommand(name string, args map[string]any) (string, error) {
	name = trimSlashCommand(name)
	hostCmdMu.RLock()
	c, ok := hostCommands[name]
	hostCmdMu.RUnlock()
	if !ok {
		return "", fmt.Errorf("未知命令 /%s（可用 GET /api/commands 查看清单）", name)
	}
	if args == nil {
		args = map[string]any{}
	}
	return c.Handler(context.Background(), args)
}

// FindHostCommand 按名查命令（nil 表示不存在）。
func FindHostCommand(name string) *HostCommand {
	name = trimSlashCommand(name)
	hostCmdMu.RLock()
	defer hostCmdMu.RUnlock()
	return hostCommands[name]
}

// trimSlashCommand 归一化命令名（去前导 "/" 与空白）。
func trimSlashCommand(name string) string {
	n := 0
	for _, r := range name {
		if r == '/' || r == ' ' || r == '\t' {
			n++
			continue
		}
		break
	}
	return name[n:]
}

// commandNames 排序输出（测试断言用）。
func commandNames() []string {
	hostCmdMu.RLock()
	defer hostCmdMu.RUnlock()
	out := append([]string(nil), hostCmdOrder...)
	sort.Strings(out)
	return out
}

// ─── ctx.commands 服务（JS 插件面；inject: ['commands'] 声明） ─────

// buildCommandsService 构造 ctx.commands（slash 命令注册面）。
//   - register({name, description, handler})：注册命令（归属本插件，卸载自动注销）
//   - list()：命令清单
//   - run(name, args)：执行命令
func (p *jsPluginAdapter) buildCommandsService(pc *PluginContext) goja.Value {
	vm := p.vm
	c := vm.NewObject()

	// 插件归属名（卸载注销用）：优先 def.name，回落 def.id
	owner := ""
	if p.def != nil {
		owner = p.def.name
		if owner == "" {
			owner = p.def.id
		}
	}

	c.Set("register", func(call goja.FunctionCall) goja.Value {
		arg := call.Argument(0)
		if goja.IsUndefined(arg) || goja.IsNull(arg) {
			panic(vm.NewTypeError("ctx.commands.register: 需要一个对象 {name, description, handler}"))
		}
		obj, ok := arg.Export().(map[string]any)
		if !ok {
			panic(vm.NewTypeError("ctx.commands.register: 参数必须是对象"))
		}
		name := mapStr(obj, "name")
		desc := mapStr(obj, "description")
		if name == "" {
			panic(vm.NewTypeError("ctx.commands.register: name 不能为空"))
		}
		fnVal := arg.ToObject(vm).Get("handler")
		fn, ok := goja.AssertFunction(fnVal)
		if !ok {
			panic(vm.NewTypeError("ctx.commands.register: handler 必须是函数"))
		}
		err := RegisterHostCommand(name, desc, func(ctx context.Context, args map[string]any) (string, error) {
			var out string
			var hErr error
			// goja 非并发安全：经 VM 锁进入（timer/事件回调同构；见 withLock）
			p.withLock(func() {
				v, err := fn(goja.Undefined(), vm.ToValue(args))
				if err != nil {
					hErr = err
					return
				}
				out = fmt.Sprintf("%v", v.Export())
			})
			if hErr != nil {
				return "", fmt.Errorf("命令 /%s 执行失败: %w", name, hErr)
			}
			return out, nil
		}, owner)
		if err != nil {
			panic(vm.NewGoError(err))
		}
		return vm.ToValue(map[string]any{"ok": true, "name": name})
	})

	c.Set("list", func(call goja.FunctionCall) goja.Value {
		return vm.ToValue(ListHostCommands())
	})

	c.Set("run", func(call goja.FunctionCall) goja.Value {
		name := call.Argument(0).String()
		var args map[string]any
		if a := call.Argument(1); a != nil && !goja.IsUndefined(a) && !goja.IsNull(a) {
			if m, ok := a.Export().(map[string]any); ok {
				args = m
			}
		}
		out, err := RunHostCommand(name, args)
		if err != nil {
			panic(vm.NewGoError(fmt.Errorf("ctx.commands.run 失败: %v", err)))
		}
		return vm.ToValue(out)
	})

	return vm.ToValue(c)
}
