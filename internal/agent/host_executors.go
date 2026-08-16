// ═══════════════════════════════════════════════════════════════
// host_executors.go — 宿主工具执行器索引（对齐 harness「seam 服务」模式）
//
// 迁移背景（2026-08-16）：内置 Go 工具（builtin_plugins.go 的 20 组）迁移为
// 磁盘外置 JS 插件（.pair/plugins/tool-*）时，插件注册同名工具接管 agent 可见面；
// 原 Go 实现经「存档」登记到本索引，插件 execute 可经 ctx.hostTool(name, args)
// 调用宿主能力（对齐 harness：工具编排在插件、底层能力在宿主 seam 服务）。
//
// 本索引全局单例（跨 Registry/会话存活），与工具注册表（Registry）解耦：
//   - Registry 中的工具 = agent 可见面（由磁盘插件注册，可插拔）
//   - hostExecutors 中的执行器 = 宿主实现库（Go Handler，被插件引用）
//
// 生命周期：hostTool 服务随插件 ctx 注入；存档由 PluginHost 在插件覆盖宿主
// 工具时自动完成（claimTool）。进程退出即清空（无持久化需求）。
// ═══════════════════════════════════════════════════════════════

package agent

import (
	"context"
	"fmt"
	"sync"
)

// hostToolExecutor 宿主执行器：函数签名与 ToolHandler 一致。
type hostToolExecutor = ToolHandler

var (
	hostExecMu      sync.RWMutex
	hostExecutors   = map[string]hostToolExecutor{}   // 工具名 → 宿主执行器
	hostExecMeta    = map[string]*Tool{}              // 工具名 → 宿主元数据（schema 参考）
)

// ArchiveHostTool 存档宿主工具（执行器 + 元数据）。同名重复存档覆盖。
// 由 PluginHost 在插件接管宿主工具时调用；亦可供测试直接登记。
func ArchiveHostTool(t *Tool) {
	if t == nil || t.Name == "" {
		return
	}
	hostExecMu.Lock()
	defer hostExecMu.Unlock()
	if t.Handler != nil {
		hostExecutors[t.Name] = t.Handler
	}
	meta := *t // 浅拷贝（元数据只读引用）
	hostExecMeta[t.Name] = &meta
}

// ExecuteHostTool 执行宿主持有工具（ctx.hostTool 服务后端）。
// 未存档 → 明确错误（提示插件应改用 ctx.fs/ctx.bash 实现或检查装载顺序）。
func ExecuteHostTool(name string, args map[string]any) (string, error) {
	hostExecMu.RLock()
	fn, ok := hostExecutors[name]
	hostExecMu.RUnlock()
	if !ok {
		return "", fmt.Errorf("hostTool %q：宿主执行器不存在（该工具未由内置 Go 插件注册，或已被移除）", name)
	}
	return fn(context.Background(), args)
}

// HostToolMeta 取宿主工具元数据（schema 参考；JS 插件声明 schema 时可拉取对齐）。
func HostToolMeta(name string) (*Tool, bool) {
	hostExecMu.RLock()
	defer hostExecMu.RUnlock()
	t, ok := hostExecMeta[name]
	return t, ok
}

// HostToolNames 列出已存档的宿主工具名（诊断/清单用）。
func HostToolNames() []string {
	hostExecMu.RLock()
	defer hostExecMu.RUnlock()
	names := make([]string, 0, len(hostExecutors))
	for n := range hostExecutors {
		names = append(names, n)
	}
	return names
}
