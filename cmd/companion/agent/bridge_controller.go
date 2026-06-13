// BridgeController —— Agent 与系统之间的桥接层。
//
// 设计意图：
//   gou-ide 的 Agent 引擎（本包）是纯 Go 的独立引擎，不依赖 GUI（goui/Skia）。
//   BridgeController 抽象了 Agent 访问系统资源（文件系统、进程、配置、网络等）的统一接口，
//   支持两种模式：
//     - BridgeMode（默认，安全受限）：当前行为——路径限工作区内、命令限 120s 超时、只读/写审批门控
//     - TakeoverMode（全面接管）：Agent 获得直接控制系统资源的能力（路径不限、命令不限、配置可写）
//
// 未来独立管理 Agent：
//   本控制器设计为可脱离 GUI 独立运行。当拆出独立的管理 Agent 时，只需：
//     1. 创建 BridgeController 实例（无需 bridge.go 的 UI 绑定）
//     2. 注册系统工具到 Registry
//     3. 直接调用 loop.Run()
//   即获得一个独立的系统管理 Agent。

package agent

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// BridgeMode 桥接模式：决定 Agent 对系统资源的访问权限。
type BridgeMode string

const (
	// BridgeBridged 桥接模式（默认）：
	//   - 文件操作：resolvePath 限工作区内
	//   - 命令执行：120s 超时，工作区内
	//   - Lua 沙箱：仅 base/string/table/math
	//   - 配置：只读
	//   - 权限门：Gate 裁决（ask/allow/deny）
	BridgeBridged BridgeMode = "bridge"

	// BridgeTakeover 接管模式（全面管控）：
	//   - 文件操作：不限路径（可系统全局）
	//   - 命令执行：不限超时，不限目录
	//   - Lua 沙箱：可注册文件/进程/网络等特权 API
	//   - 配置：可读写
	//   - 权限门：仅重大操作需审批
	BridgeTakeover BridgeMode = "takeover"
)

// SystemCapability 系统能力标记（供 Agent 感知当前可用的功能）。
type SystemCapability string

const (
	CapFileSystem     SystemCapability = "filesystem"       // 文件系统
	CapProcess        SystemCapability = "process"          // 进程管理
	CapNetwork        SystemCapability = "network"          // 网络
	CapSystemConfig   SystemCapability = "system_config"    // 系统配置
	CapServiceControl SystemCapability = "service_control"  // 服务控制
	CapRegistry       SystemCapability = "registry"         // 注册表（Windows）
	CapEnvironment    SystemCapability = "environment"      // 环境变量
)

// BridgeController Agent 与系统之间的桥接控制器。
// 线程安全：方法级别的锁保护模式切换和审计日志。
type BridgeController struct {
	mu     sync.RWMutex
	mode   BridgeMode      // 当前模式
	root   string           // 工作区根目录
	log    []AuditEntry     // 审计日志
	capMap map[SystemCapability]bool // 当前可用能力集合

	// takeoverAuthorized 接管授权标记（仅 TakeoverMode 下为 true）
	takeoverAuthorized bool
	// takeoverApprovedAt 接管授权时间
	takeoverApprovedAt time.Time
	// takeoverApprovedBy 接管授权者（"user" / "config"）
	takeoverApprovedBy string
}

// AuditEntry 审计日志条目（记录所有桥接操作）。
type AuditEntry struct {
	Time    time.Time `json:"time"`
	Mode    string    `json:"mode"`
	Action  string    `json:"action"`
	Target  string    `json:"target,omitempty"`
	Result  string    `json:"result,omitempty"`
	Details string    `json:"details,omitempty"`
}

// NewBridgeController 创建桥接控制器，默认工作在 BridgeBridged 模式。
func NewBridgeController(root string) *BridgeController {
	bc := &BridgeController{
		mode:   BridgeBridged,
		root:   root,
		log:    make([]AuditEntry, 0, 64),
		capMap: make(map[SystemCapability]bool),
	}
	bc.syncCapabilities()
	return bc
}

// ─── 模式管理 ────────────────────────────────────────────────

// Mode 返回当前桥接模式。
func (bc *BridgeController) Mode() BridgeMode {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.mode
}

// ModeString 返回模式的中文描述。
func (bc *BridgeController) ModeString() string {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	switch bc.mode {
	case BridgeBridged:
		return "桥接模式（安全受限）"
	case BridgeTakeover:
		return "接管模式（全面管控）"
	default:
		return "未知模式"
	}
}

// SwitchToTakeover 切换到接管模式。
// authorizedBy 为授权者标识："user"（用户交互授权）或 "config"（配置预设）。
func (bc *BridgeController) SwitchToTakeover(authorizedBy string) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	if bc.mode == BridgeTakeover {
		return fmt.Errorf("已在接管模式")
	}

	bc.mode = BridgeTakeover
	bc.takeoverAuthorized = true
	bc.takeoverApprovedAt = time.Now()
	bc.takeoverApprovedBy = authorizedBy
	bc.syncCapabilities()

	bc.appendAudit("switch_takeover", "", "ok", "授权者: "+authorizedBy)
	return nil
}

// SwitchToBridged 切换回桥接模式（安全沉降）。
func (bc *BridgeController) SwitchToBridged() error {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	if bc.mode == BridgeBridged {
		return fmt.Errorf("已在桥接模式")
	}

	bc.mode = BridgeBridged
	bc.takeoverAuthorized = false
	bc.syncCapabilities()

	bc.appendAudit("switch_bridged", "", "ok", "回到安全桥接模式")
	return nil
}

// IsTakeover 是否处于接管模式。
func (bc *BridgeController) IsTakeover() bool {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.mode == BridgeTakeover
}

// Capabilities 返回当前可用的系统能力列表。
func (bc *BridgeController) Capabilities() map[SystemCapability]bool {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	out := make(map[SystemCapability]bool, len(bc.capMap))
	for k, v := range bc.capMap {
		out[k] = v
	}
	return out
}

// CapabilitiesText 返回能力描述文本（供 Agent 系统提示）。
func (bc *BridgeController) CapabilitiesText() string {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	var caps []string
	for cap_, enabled := range bc.capMap {
		if enabled {
			switch cap_ {
			case CapFileSystem:
				caps = append(caps, "文件系统（全路径读写）")
			case CapProcess:
				caps = append(caps, "进程管理（任意命令执行）")
			case CapNetwork:
				caps = append(caps, "网络（HTTP/WS 请求）")
			case CapSystemConfig:
				caps = append(caps, "系统配置读写")
			case CapServiceControl:
				caps = append(caps, "服务管理")
			case CapRegistry:
				caps = append(caps, "Windows 注册表")
			case CapEnvironment:
				caps = append(caps, "环境变量管理")
			default:
				caps = append(caps, string(cap_))
			}
		}
	}
	if len(caps) == 0 {
		return "（无可用系统能力）"
	}
	return strings.Join(caps, "、")
}

// syncCapabilities 根据当前模式同步能力集合（调用前必须持有锁）。
func (bc *BridgeController) syncCapabilities() {
	bc.capMap = map[SystemCapability]bool{}
	switch bc.mode {
	case BridgeBridged:
		// 桥接模式：有限能力
		bc.capMap[CapFileSystem] = false // 受限：resolvePath 限工作区内
		bc.capMap[CapProcess] = false    // 受限：120s 超时 + 工作目录
		bc.capMap[CapNetwork] = true     // web_fetch/web_search 允许
		bc.capMap[CapSystemConfig] = false
		bc.capMap[CapServiceControl] = false
		bc.capMap[CapRegistry] = false
		bc.capMap[CapEnvironment] = false
	case BridgeTakeover:
		// 接管模式：全部能力
		bc.capMap[CapFileSystem] = true
		bc.capMap[CapProcess] = true
		bc.capMap[CapNetwork] = true
		bc.capMap[CapSystemConfig] = true
		bc.capMap[CapServiceControl] = true
		bc.capMap[CapRegistry] = true
		bc.capMap[CapEnvironment] = true
	}
}

// ─── 系统操作（统一桥接入口） ─────────────────────────────────

// ReadFile 读取文件（桥接模式受限，接管模式全路径）。
func (bc *BridgeController) ReadFile(path string) ([]byte, error) {
	bc.mu.RLock()
	mode := bc.mode
	bc.mu.RUnlock()

	resolved := path
	if mode == BridgeBridged {
		var err error
		resolved, err = bc.resolvePath(path)
		if err != nil {
			return nil, err
		}
	}

	data, err := os.ReadFile(resolved)
	bc.record("read_file", resolved, err)
	return data, err
}

// WriteFile 写入文件（桥接模式受限，接管模式全路径）。
func (bc *BridgeController) WriteFile(path string, content []byte, perm os.FileMode) error {
	bc.mu.RLock()
	mode := bc.mode
	bc.mu.RUnlock()

	resolved := path
	if mode == BridgeBridged {
		var err error
		resolved, err = bc.resolvePath(path)
		if err != nil {
			return err
		}
	}

	if err := os.MkdirAll(filepath.Dir(resolved), 0o755); err != nil {
		bc.record("write_file", resolved, err)
		return err
	}
	err := os.WriteFile(resolved, content, perm)
	bc.record("write_file", resolved, err)
	return err
}

// ExecCommand 执行系统命令（桥接模式受限，接管模式全放开）。
func (bc *BridgeController) ExecCommand(ctx context.Context, command, cwd string, timeout time.Duration) (string, error) {
	bc.mu.RLock()
	mode := bc.mode
	bc.mu.RUnlock()

	dir := cwd
	if mode == BridgeBridged {
		// 桥接模式：限工作区内目录
		if dir != "" {
			var err error
			dir, err = bc.resolvePath(cwd)
			if err != nil {
				return "", err
			}
		} else {
			dir = bc.root
		}
		if timeout <= 0 || timeout > 120*time.Second {
			timeout = 120 * time.Second
		}
	} else {
		// 接管模式：不限目录，不限超时
		if dir == "" {
			dir = bc.root
		}
		if timeout <= 0 {
			timeout = 300 * time.Second // 接管模式默认 5 分钟
		}
	}

	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	c := exec.CommandContext(cctx, "cmd", "/C", "chcp 65001 >nul & "+command)
	c.Dir = dir
	out, err := c.CombinedOutput()
	result := string(out)
	if cctx.Err() == context.DeadlineExceeded {
		result += "\n[超时 " + timeout.String() + " 已终止]"
	}

	bc.record("exec", command+" @ "+dir, err)
	return result, err
}

// resolvePath 解析路径（工作区内安全约束）。
// 先检查路径是否在 bc.root 下；若不在，再查是否在 WorkspaceRoots（工作区其他根目录）下。
func (bc *BridgeController) resolvePath(p string) (string, error) {
	full := p
	if !filepath.IsAbs(full) {
		full = filepath.Join(bc.root, full)
	}
	full = filepath.Clean(full)
	rel, err := filepath.Rel(bc.root, full)
	if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return full, nil
	}
	// 再查其他工作区根目录（多根工作区支持）
	for _, wr := range WorkspaceRoots {
		if wr == bc.root {
			continue
		}
		rel, err := filepath.Rel(wr, full)
		if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return full, nil
		}
	}
	return "", fmt.Errorf("路径越界（不允许访问工作区外）: %s", p)
}

// ─── 审计 ────────────────────────────────────────────────────

// record 记录一次操作到审计日志。
func (bc *BridgeController) record(action, target string, err error) {
	result := "ok"
	if err != nil {
		result = "err: " + err.Error()
	}
	bc.mu.Lock()
	bc.appendAudit(action, target, result, "")
	bc.mu.Unlock()
}

// appendAudit 追加审计条目（调用前必须持有锁）。
func (bc *BridgeController) appendAudit(action, target, result, details string) {
	bc.log = append(bc.log, AuditEntry{
		Time:    time.Now(),
		Mode:    string(bc.mode),
		Action:  action,
		Target:  target,
		Result:  result,
		Details: details,
	})
	// 保留最近 1000 条
	if len(bc.log) > 1000 {
		bc.log = bc.log[len(bc.log)-1000:]
	}
}

// AuditLog 返回审计日志副本。
func (bc *BridgeController) AuditLog() []AuditEntry {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	out := make([]AuditEntry, len(bc.log))
	copy(out, bc.log)
	return out
}

// AuditLogText 返回审计日志的文本摘要。
func (bc *BridgeController) AuditLogText(limit int) string {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	if len(bc.log) == 0 {
		return "（无审计记录）"
	}

	start := 0
	if limit > 0 && len(bc.log) > limit {
		start = len(bc.log) - limit
	}

	var b strings.Builder
	b.WriteString("## 桥接审计日志\n\n")
	b.WriteString("| 时间 | 模式 | 操作 | 目标 | 结果 |\n")
	b.WriteString("|------|------|------|------|------|\n")
	for _, e := range bc.log[start:] {
		b.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s |\n",
			e.Time.Format("15:04:05"),
			e.Mode,
			e.Action,
			e.Target,
			e.Result,
		))
	}
	b.WriteString(fmt.Sprintf("\n共 %d 条记录（显示最近 %d 条）\n", len(bc.log), limit))
	return b.String()
}

// ─── 全局默认实例（供 bridge.go 和工具共享） ──────────────────

var (
	defaultBridgeController   *BridgeController
	defaultBridgeControllerMu sync.Mutex
)

// GetBridgeController 获取或创建全局默认 BridgeController。
func GetBridgeController(root string) *BridgeController {
	defaultBridgeControllerMu.Lock()
	defer defaultBridgeControllerMu.Unlock()
	if defaultBridgeController == nil {
		defaultBridgeController = NewBridgeController(root)
	} else if root != "" && defaultBridgeController.root != root {
		// 工作区根变化时更新
		defaultBridgeController.mu.Lock()
		defaultBridgeController.root = root
		defaultBridgeController.mu.Unlock()
	}
	return defaultBridgeController
}

// ResetBridgeController 重置全局桥接控制器（工作区切换时）。
func ResetBridgeController(root string) {
	defaultBridgeControllerMu.Lock()
	defer defaultBridgeControllerMu.Unlock()
	defaultBridgeController = NewBridgeController(root)
}

// BridgeModeFromString 从字符串解析桥接模式。
func BridgeModeFromString(s string) BridgeMode {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "takeover", "接管":
		return BridgeTakeover
	default:
		return BridgeBridged
	}
}
