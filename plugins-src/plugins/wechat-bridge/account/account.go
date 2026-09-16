// Package account 管理多账号：注册表、凭据持久化、每账号运行时与登录编排。
//
// 数据布局（root = 桥 --data-dir，通常 <workspace>/.pair/wechat-bridge）：
//
//	accounts.json                     注册表（账号条目数组）
//	accounts/<id>/credentials.json    凭据（token/base_url/bot_id/user_id）
//	accounts/<id>/state.json          桥运行状态（get_updates_buf 等；P2 接入）
//
// 账号 id 规则：登录成功后 = ilink bot_id；登录前使用临时 id（tmp-xxxxxx），
// 登录成功时迁移到 bot_id 目录（同 bot 重复登录则合并覆盖既有账号凭据）。
//
// 稳定性设计（多账号「必须稳」，方案 §2.1.1）：
//   - 隔离：每账号独立 Runtime；goroutine 均带 recover（panic 不扩散）；
//   - 状态机：账号级 phase（login/running/stale/offline/error），单账号异常不影响其他；
//   - 原子持久化：凭据/注册表均原子写，任意重启不丢；
//   - 幂等：重复添加同别名自动去重 convId；删除不存在账号返回错误但不崩；
//   - 多用户预留：ownerUserId 字段（当前恒 "local"）。
package account

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/hoonfeng/paircode/plugins-src/plugins/wechat-bridge/ilink"
	"github.com/hoonfeng/paircode/plugins-src/plugins/wechat-bridge/login"
	"github.com/hoonfeng/paircode/plugins-src/plugins/wechat-bridge/store"
)

// 账号运行阶段。
const (
	PhaseLogin   = "login"   // 登录中（扫码）
	PhaseRunning = "running" // 已登录运行
	PhaseStale   = "stale"   // 会话失效重试中（P2 桥接）
	PhaseOffline = "offline" // 未登录/已停止
	PhaseError   = "error"   // 异常（panic/持久化失败等）
)

// Info 账号注册信息（accounts.json 条目）。
type Info struct {
	ID            string `json:"id"`                      // bot_id（登录前 tmp-xxxxxx）
	Alias         string `json:"alias,omitempty"`         // 别名（显示名）
	ConvID        string `json:"convId"`                  // PairCode 会话 id（conv_wx_<别名>）
	WorkspaceRoot string `json:"workspaceRoot,omitempty"` // 默认工作区（空=宿主主工作区）
	OwnerUserID   string `json:"ownerUserId,omitempty"`   // 多用户预留（"local"）
	Enabled       bool   `json:"enabled"`
	AddedAt       string `json:"addedAt,omitempty"`
	LastMsgAt     string `json:"lastMsgAt,omitempty"`
}

// Registry accounts.json 结构。
type Registry struct {
	Accounts []Info `json:"accounts"`
}

// Runtime 单账号运行时（隔离单元）。
type Runtime struct {
	mu      sync.Mutex
	info    Info
	dir     string
	creds   *login.Credentials
	sess    *login.Session
	phase   string
	lastErr string
}

// Phase 当前阶段。
func (r *Runtime) Phase() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.phase
}

// Creds 当前凭据（可能为 nil）。
func (r *Runtime) Creds() *login.Credentials {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.creds
}

// Info 账号信息。
func (r *Runtime) Info() Info {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.info
}

// Dir 账号数据目录（state.json 等所在）。
func (r *Runtime) Dir() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.dir
}

// snapshotMap 状态快照（/status 输出用）。
func (r *Runtime) snapshotMap() map[string]any {
	r.mu.Lock()
	defer r.mu.Unlock()
	m := map[string]any{
		"id":     r.info.ID,
		"alias":  r.info.Alias,
		"convId": r.info.ConvID,
		"phase":  r.phase,
	}
	if r.info.LastMsgAt != "" {
		m["lastMsgAt"] = r.info.LastMsgAt
	}
	if r.lastErr != "" {
		m["lastErr"] = r.lastErr
	}
	if r.sess != nil {
		st := r.sess.Snapshot()
		m["loginStatus"] = st.Status
		m["scanned"] = st.Scanned
		m["refreshCount"] = st.RefreshCount
		m["hasQR"] = st.QRContent != ""
		if st.Err != "" {
			m["loginErr"] = st.Err
		}
	}
	return m
}

// Manager 账号总管。
type Manager struct {
	mu        sync.Mutex
	root      string
	regFile   string
	runtimes  map[string]*Runtime
	onChange  func()
	onReady   func(id string) // 账号就绪（登录成功/凭据可用）
	onRemoved func(id string) // 账号移除
}

// NewManager 创建账号总管（root 为桥数据目录）。
func NewManager(root string) *Manager {
	return &Manager{
		root:     root,
		regFile:  filepath.Join(root, "accounts.json"),
		runtimes: map[string]*Runtime{},
	}
}

// SetOnChange 注册状态变化回调（main 接线到 ctl 事件总线）。
func (m *Manager) SetOnChange(f func()) {
	m.mu.Lock()
	m.onChange = f
	m.mu.Unlock()
}

// SetOnReady 注册「账号就绪」回调（登录成功；可能在任意 goroutine 中调用，
// 实现方不得反调 Manager 的持锁方法以外接口——宿主据此启动桥循环）。
func (m *Manager) SetOnReady(f func(id string)) {
	m.mu.Lock()
	m.onReady = f
	m.mu.Unlock()
}

// SetOnRemoved 注册「账号移除」回调（Remove 之后调用；宿主据此停桥循环）。
func (m *Manager) SetOnRemoved(f func(id string)) {
	m.mu.Lock()
	m.onRemoved = f
	m.mu.Unlock()
}

func (m *Manager) notifyReady(id string) {
	m.mu.Lock()
	cb := m.onReady
	m.mu.Unlock()
	if cb != nil {
		defer func() {
			if p := recover(); p != nil {
				// 宿主回调异常隔离：不破坏登录成功流程（桥未启动可由 StartAll 兜底）
			}
		}()
		cb(id)
	}
}

func (m *Manager) notifyRemoved(id string) {
	m.mu.Lock()
	cb := m.onRemoved
	m.mu.Unlock()
	if cb != nil {
		defer func() {
			if p := recover(); p != nil {
				// 宿主回调异常隔离：不影响移除流程返回
			}
		}()
		cb(id)
	}
}

func (m *Manager) notify() {
	m.mu.Lock()
	cb := m.onChange
	m.mu.Unlock()
	if cb != nil {
		cb()
	}
}

// Load 读取注册表与各账号凭据。
func (m *Manager) Load() error {
	var reg Registry
	_ = store.ReadJSON(m.regFile, &reg) // 文件不存在/损坏 → 空注册表
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, info := range reg.Accounts {
		rt := &Runtime{
			info: info,
			dir:  filepath.Join(m.root, "accounts", info.ID),
		}
		var creds login.Credentials
		if store.ReadJSON(filepath.Join(rt.dir, "credentials.json"), &creds) && creds.Token != "" {
			rt.creds = &creds
			rt.phase = PhaseRunning
		} else {
			rt.phase = PhaseOffline
		}
		m.runtimes[info.ID] = rt
	}
	return nil
}

// IDs 账号 id 列表（排序稳定）。
func (m *Manager) IDs() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	ids := make([]string, 0, len(m.runtimes))
	for id := range m.runtimes {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// Get 按 id 取运行时会话（不存在返回 nil）。
func (m *Manager) Get(id string) *Runtime {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.runtimes[id]
}

// StatusMap /status 数据（{"accounts": [...]}）。
func (m *Manager) StatusMap() map[string]any {
	snap := map[string]any{}
	m.mu.Lock()
	rts := make([]*Runtime, 0, len(m.runtimes))
	ids := make([]string, 0, len(m.runtimes))
	for id := range m.runtimes {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		rts = append(rts, m.runtimes[id])
	}
	m.mu.Unlock()

	arr := make([]map[string]any, 0, len(rts))
	for _, rt := range rts {
		arr = append(arr, rt.snapshotMap())
	}
	snap["accounts"] = arr
	return snap
}

// CurrentLoginSession 当前处于登录中的会话（nil=无）。
func (m *Manager) CurrentLoginSession() *login.Session {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, rt := range m.runtimes {
		rt.mu.Lock()
		s := rt.sess
		rt.mu.Unlock()
		if s != nil {
			return s
		}
	}
	return nil
}

// ─────────────────────────── 增删与登录 ───────────────────────────

// Add 新增账号并启动扫码登录；返回账号 id（登录前为临时 id）。
func (m *Manager) Add(alias string) (string, error) {
	if act := m.activeLoginID(""); act != "" {
		return "", fmt.Errorf("已有账号（%s）正在登录中，请先完成扫码或稍候再试", act)
	}
	alias = strings.TrimSpace(alias)
	if alias == "" {
		alias = "微信"
	}
	tmpID := "tmp-" + randomHex(6)
	info := Info{
		ID:          tmpID,
		Alias:       alias,
		ConvID:      m.uniqueConvID(alias),
		OwnerUserID: "local",
		Enabled:     true,
		AddedAt:     time.Now().UTC().Format(time.RFC3339),
	}
	rt := &Runtime{
		info:  info,
		dir:   filepath.Join(m.root, "accounts", tmpID),
		phase: PhaseLogin,
	}
	m.mu.Lock()
	m.runtimes[tmpID] = rt
	m.mu.Unlock()

	m.startLogin(rt, nil)
	m.notify()
	return tmpID, nil
}

// Remove 移除账号（停会话、删数据目录、更新注册表）。
func (m *Manager) Remove(id string) error {
	m.mu.Lock()
	rt, ok := m.runtimes[id]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("账号不存在: %s", id)
	}
	delete(m.runtimes, id)
	m.mu.Unlock()

	rt.mu.Lock()
	if rt.sess != nil {
		rt.sess.Stop()
		rt.sess = nil
	}
	dir := rt.dir
	rt.mu.Unlock()

	_ = os.RemoveAll(dir)
	m.saveRegistry()
	m.notify()
	m.notifyRemoved(id)
	return nil
}

// Relogin 为既有账号重新发起扫码登录（带旧 token 供服务器识别）。
func (m *Manager) Relogin(id string) error {
	m.mu.Lock()
	rt, ok := m.runtimes[id]
	m.mu.Unlock()
	if !ok {
		return fmt.Errorf("账号不存在: %s", id)
	}
	rt.mu.Lock()
	if rt.sess != nil {
		rt.mu.Unlock()
		return fmt.Errorf("该账号已在登录中")
	}
	local := []string{}
	if rt.creds != nil && rt.creds.Token != "" {
		local = []string{rt.creds.Token}
	}
	rt.mu.Unlock()
	if act := m.activeLoginID(id); act != "" {
		return fmt.Errorf("已有账号（%s）正在登录中，请先完成扫码或稍候再试", act)
	}
	m.startLogin(rt, local)
	return nil
}

// Bootstrap 启动编排：无账号 → 建默认账号并进入登录；有账号未登录 → 恢复登录。
func (m *Manager) Bootstrap() {
	m.mu.Lock()
	total := len(m.runtimes)
	var need []*Runtime
	for _, rt := range m.runtimes {
		rt.mu.Lock()
		if rt.sess == nil && (rt.phase == PhaseOffline || rt.creds == nil) {
			need = append(need, rt)
		}
		rt.mu.Unlock()
	}
	m.mu.Unlock()

	if total == 0 {
		_, _ = m.Add("微信")
		return
	}
	if len(need) > 0 {
		sort.Slice(need, func(i, j int) bool { return need[i].info.AddedAt < need[j].info.AddedAt })
		m.startLogin(need[0], nil)
	}
}

// StopAll 停止全部登录会话（退出流程用）。
func (m *Manager) StopAll() {
	m.mu.Lock()
	rts := make([]*Runtime, 0, len(m.runtimes))
	for _, rt := range m.runtimes {
		rts = append(rts, rt)
	}
	m.mu.Unlock()
	for _, rt := range rts {
		rt.mu.Lock()
		if rt.sess != nil {
			rt.sess.Stop()
		}
		rt.mu.Unlock()
	}
}

// ─────────────────────────── 内部 ───────────────────────────

// startLogin 启动账号的扫码登录会话（异步）。
func (m *Manager) startLogin(rt *Runtime, localTokens []string) {
	rt.mu.Lock()
	if rt.sess != nil {
		rt.mu.Unlock()
		return
	}
	rt.phase = PhaseLogin
	rt.lastErr = ""
	rt.mu.Unlock()

	sess := login.NewSession(ilink.DefaultBaseURL, localTokens, login.Options{
		OnStateChange: func(login.State) { m.notify() },
	})
	rt.mu.Lock()
	rt.sess = sess
	rt.mu.Unlock()
	sess.Start()

	go func() {
		defer func() {
			if p := recover(); p != nil {
				rt.mu.Lock()
				rt.phase = PhaseError
				rt.lastErr = fmt.Sprintf("登录任务内部错误(panic): %v", p)
				rt.mu.Unlock()
				m.notify()
			}
		}()
		creds, err := sess.Wait()
		rt.mu.Lock()
		rt.sess = nil
		rt.mu.Unlock()
		if err != nil {
			rt.mu.Lock()
			if rt.phase == PhaseLogin {
				rt.phase = PhaseOffline
				rt.lastErr = err.Error()
			}
			rt.mu.Unlock()
			m.notify()
			return
		}
		m.onLoginSuccess(rt, creds)
	}()
}

// onLoginSuccess 登录成功：写凭据、迁移临时 id → bot_id（或合并到既有账号）。
func (m *Manager) onLoginSuccess(rt *Runtime, creds *login.Credentials) {
	newID := creds.BotID
	if newID == "" {
		newID = rt.info.ID
	}
	newDir := filepath.Join(m.root, "accounts", newID)

	// 1) 凭据落盘（先落盘再改内存，保证崩溃时一致性）
	if err := store.WriteJSONAtomic(filepath.Join(newDir, "credentials.json"), creds); err != nil {
		rt.mu.Lock()
		rt.phase = PhaseError
		rt.lastErr = "保存凭据失败: " + err.Error()
		rt.mu.Unlock()
		m.notify()
		return
	}

	m.mu.Lock()
	oldID := rt.info.ID
	if _, alive := m.runtimes[oldID]; !alive {
		// 账号已被移除：丢弃结果（已落盘的凭据随 Remove 清理流程处理）
		m.mu.Unlock()
		_ = os.RemoveAll(newDir)
		return
	}
	if existing, ok := m.runtimes[newID]; ok && existing != rt {
		// 合并：同一 bot 重新登录 → 覆盖既有账号
		delete(m.runtimes, oldID)
		m.mu.Unlock()

		rt.mu.Lock()
		oldDir := rt.dir
		rt.mu.Unlock()
		existing.mu.Lock()
		existing.creds = creds
		existing.phase = PhaseRunning
		existing.lastErr = ""
		existing.mu.Unlock()

		if filepath.Clean(oldDir) != filepath.Clean(existing.dir) {
			_ = os.RemoveAll(oldDir)
		}
		m.saveRegistry()
		m.notify()
		// 合并场景（同一 bot 重登）：通知就绪 id = 既有账号 id（newID）。
		// 顺序：先持久化/广播，再通知宿主启动桥（消费者读到的磁盘状态已一致）。
		m.notifyReady(newID)
		return
	}
	// 常规迁移
	if oldID != newID {
		delete(m.runtimes, oldID)
		m.runtimes[newID] = rt
	}
	m.mu.Unlock()

	rt.mu.Lock()
	oldDir := rt.dir
	rt.dir = newDir
	rt.info.ID = newID
	rt.creds = creds
	rt.phase = PhaseRunning
	rt.lastErr = ""
	rt.mu.Unlock()
	if filepath.Clean(oldDir) != filepath.Clean(newDir) {
		_ = os.RemoveAll(oldDir) // 临时空目录
	}
	m.saveRegistry()
	m.notify()
	// 临时 id → bot_id 迁移完成后通知宿主启动桥（newID 在函数开头赋值：
	// creds.BotID，为空则回退旧 id）。
	m.notifyReady(newID)
}

// saveRegistry 持久化注册表。
func (m *Manager) saveRegistry() {
	m.mu.Lock()
	infos := make([]Info, 0, len(m.runtimes))
	for _, rt := range m.runtimes {
		rt.mu.Lock()
		infos = append(infos, rt.info)
		rt.mu.Unlock()
	}
	m.mu.Unlock()
	sort.Slice(infos, func(i, j int) bool { return infos[i].AddedAt < infos[j].AddedAt })
	_ = store.WriteJSONAtomic(m.regFile, Registry{Accounts: infos})
}

// activeLoginID 返回正在登录的账号 id（except 除外；"" = 无）。
func (m *Manager) activeLoginID(except string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, rt := range m.runtimes {
		if id == except {
			continue
		}
		if rt.Phase() == PhaseLogin {
			return id
		}
	}
	return ""
}

// uniqueConvID 生成去重的会话 id：conv_wx_<别名 slug>（重复加 -2/-3…）。
func (m *Manager) uniqueConvID(alias string) string {
	base := "conv_wx_" + slug(alias)
	m.mu.Lock()
	defer m.mu.Unlock()
	taken := map[string]bool{}
	for _, rt := range m.runtimes {
		rt.mu.Lock()
		taken[rt.info.ConvID] = true
		rt.mu.Unlock()
	}
	if !taken[base] {
		return base
	}
	for i := 2; ; i++ {
		cand := fmt.Sprintf("%s-%d", base, i)
		if !taken[cand] {
			return cand
		}
	}
}

// slug 过滤文件名非法字符（保留中文等原样字符）。
func slug(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r < 0x20 || strings.ContainsRune(`\/:*?"<>|`, r) || r == ' ':
			b.WriteRune('-')
		default:
			b.WriteRune(r)
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		out = "wx"
	}
	return out
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
