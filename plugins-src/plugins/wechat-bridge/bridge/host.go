// host.go — 桥总管：为注册表中每个已登录账号启动/停止桥循环，共享事件流。
//
// 生命周期接线（main 负责）：
//
//	manager.SetOnReady(host.StartAccount)    // 登录成功 → 启动桥
//	manager.SetOnRemoved(host.StopAccount)   // 移除账号 → 停桥
//	host.Start()                             // 启动事件流 + 全部 running 账号
//	host.StopAll()                           // 退出流程
package bridge

import (
	"fmt"
	"sync"
	"time"

	"github.com/hoonfeng/paircode/plugins-src/plugins/wechat-bridge/account"
	"github.com/hoonfeng/paircode/plugins-src/plugins/wechat-bridge/config"
)

// Host 多账号桥循环总管。
type Host struct {
	cfg config.Config
	mgr *account.Manager
	es  *EventStream
	log logf

	mu      sync.Mutex
	bridges map[string]*Bridge
}

// NewHost 创建桥总管（含事件流实例；不自动启动）。
func NewHost(cfg config.Config, mgr *account.Manager, log logf) *Host {
	h := &Host{
		cfg:     cfg,
		mgr:     mgr,
		log:     log,
		bridges: map[string]*Bridge{},
	}
	h.es = NewEventStream(wsURL(cfg.PairCodeURL), 10*time.Second, log)
	return h
}

// EventStream 暴露事件流实例（调试/探活用）。
func (h *Host) EventStream() *EventStream { return h.es }

// Start 启动事件流并按注册表现状启动全部已登录账号的桥循环。
func (h *Host) Start() {
	h.es.Start()
	h.StartAll()
}

// StartAll 为 phase=running 且有凭据的账号启动桥（幂等）。
func (h *Host) StartAll() {
	for _, id := range h.mgr.IDs() {
		rt := h.mgr.Get(id)
		if rt != nil && rt.Phase() == account.PhaseRunning && rt.Creds() != nil {
			h.StartAccount(id)
		}
	}
}

// StartAccount 启动单账号桥循环（幂等；账号不存在/未就绪时静默忽略）。
func (h *Host) StartAccount(id string) {
	rt := h.mgr.Get(id)
	if rt == nil {
		return
	}
	creds := rt.Creds()
	if creds == nil || creds.Token == "" {
		return
	}
	h.mu.Lock()
	if _, exists := h.bridges[id]; exists {
		h.mu.Unlock()
		return
	}
	b := New(h.cfg, rt.Info(), rt.Dir(), creds, h.es, h.log, h.handleStale)
	h.bridges[id] = b
	h.mu.Unlock()
	b.Start()
}

// StopAccount 停止单账号桥循环（幂等）。
func (h *Host) StopAccount(id string) {
	h.mu.Lock()
	b := h.bridges[id]
	delete(h.bridges, id)
	h.mu.Unlock()
	if b != nil {
		b.Stop()
	}
}

// StopAll 停止全部桥循环与事件流（退出流程）。
func (h *Host) StopAll() {
	h.mu.Lock()
	bridges := make([]*Bridge, 0, len(h.bridges))
	for _, b := range h.bridges {
		bridges = append(bridges, b)
	}
	h.bridges = map[string]*Bridge{}
	h.mu.Unlock()
	for _, b := range bridges {
		b.Stop()
	}
	h.es.Stop()
}

// ActiveCount 当前运行中的桥数量（/status 展示用）。
func (h *Host) ActiveCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.bridges)
}

// pickBridge 取指定账号的桥；accountID 为空时取唯一运行中的桥（多桥时报错由调用方处理）。
func (h *Host) pickBridge(accountID string) *Bridge {
	h.mu.Lock()
	defer h.mu.Unlock()
	if accountID != "" {
		return h.bridges[accountID]
	}
	if len(h.bridges) == 1 {
		for _, b := range h.bridges {
			return b
		}
	}
	return nil
}

// SendTo 主动发送文本（account 空 = 唯一运行中的桥）。供本地 HTTP /send 调用。
func (h *Host) SendTo(accountID, userID, text string) error {
	if accountID == "" {
		h.mu.Lock()
		n := len(h.bridges)
		h.mu.Unlock()
		if n > 1 {
			return fmt.Errorf("存在多个运行中的账号，请指定 account")
		}
	}
	b := h.pickBridge(accountID)
	if b == nil {
		if accountID != "" {
			return fmt.Errorf("账号 %s 未运行（不存在或未登录）", accountID)
		}
		return fmt.Errorf("没有运行中的账号桥")
	}
	return b.SendTo(userID, text)
}

// Contacts 已知联系人（account 空 = 唯一运行中的桥）。
func (h *Host) Contacts(accountID string) ([]map[string]any, error) {
	if accountID == "" {
		h.mu.Lock()
		n := len(h.bridges)
		h.mu.Unlock()
		if n > 1 {
			return nil, fmt.Errorf("存在多个运行中的账号，请指定 account")
		}
	}
	b := h.pickBridge(accountID)
	if b == nil {
		if accountID != "" {
			return nil, fmt.Errorf("账号 %s 未运行（不存在或未登录）", accountID)
		}
		return nil, fmt.Errorf("没有运行中的账号桥")
	}
	return b.Contacts(), nil
}

// SendMedia 主动发送媒体文件（account 空 = 唯一运行中的桥）。供本地 HTTP /send 的 file 分支调用。
func (h *Host) SendMedia(accountID, userID, filePath, caption string) error {
	if accountID == "" {
		h.mu.Lock()
		n := len(h.bridges)
		h.mu.Unlock()
		if n > 1 {
			return fmt.Errorf("存在多个运行中的账号，请指定 account")
		}
	}
	b := h.pickBridge(accountID)
	if b == nil {
		if accountID != "" {
			return fmt.Errorf("账号 %s 未运行（不存在或未登录）", accountID)
		}
		return fmt.Errorf("没有运行中的账号桥")
	}
	return b.SendMediaTo(userID, filePath, caption)
}

// handleStale -14 持续失效：停旧桥并请求重新登录（用户扫码成功后
// OnReady 回调重新 StartAccount，桥以新凭据重建）。
func (h *Host) handleStale(id string) {
	h.log("账号 %s 会话失效，发起重新登录…", id)
	h.StopAccount(id)
	if err := h.mgr.Relogin(id); err != nil {
		h.log("重新登录发起失败: %v（可稍后在设置面板手动重新登录）", err)
	}
}
