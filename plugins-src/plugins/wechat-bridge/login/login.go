// Package login 实现扫码登录状态机（ilink get_bot_qrcode / get_qrcode_status）。
//
// 状态分支对齐官方实现 8 态：
//
//	wait / scaned / need_verifycode / expired / verify_code_blocked /
//	binded_redirect / scaned_but_redirect / confirmed
//
// 与 M1 Node 原型的两个关键差异（方案决策②⑧）：
//  1. 二维码只存内存、绝不落盘（M1 存 last-qr.png 为反例）——渲染由 localhttp /qr.png 实时出；
//  2. 验证码输入不走终端 stdin，而是由外部（壳 / stdio 命令 / HTTP）注入 SubmitVerifyCode。
package login

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/hoonfeng/paircode/plugins-src/plugins/wechat-bridge/ilink"
)

// Credentials 登录成功后的凭据（JSON 字段对齐 M1 creds，保证数据兼容）。
type Credentials struct {
	Token   string `json:"token"`
	BaseURL string `json:"base_url"`
	BotID   string `json:"bot_id"`
	UserID  string `json:"user_id"`
	SavedAt string `json:"saved_at,omitempty"`
}

// 对外状态常量。
const (
	StatusWait          = "wait"                // 等待扫码
	StatusScaned        = "scaned"              // 已扫码待确认
	StatusNeedVerify    = "need_verifycode"     // 需要验证码
	StatusVerifyBlocked = "verify_code_blocked" // 验证码多次错误
	StatusExpired       = "expired"             // 二维码过期（自动刷新）
	StatusRedirect      = "scaned_but_redirect" // IDC 重定向
	StatusBinded        = "binded_redirect"     // 已绑定过（需先解绑）
	StatusConfirmed     = "confirmed"           // 已确认（成功）
	StatusFailed        = "failed"              // 失败（终态）
	StatusStopped       = "stopped"             // 被停止（终态）
)

// 常见失败错误。
var (
	ErrStopped = errors.New("登录会话已停止")
	ErrBinded  = errors.New("该微信已绑定过此 Agent（binded_redirect）；如需更换，请先在手机微信 ClawBot 设置中解除绑定后重试")
	ErrTimeout = errors.New("二维码登录超时")
)

// State 对外状态快照（渲染/监控用；并发安全由 Session 保证）。
type State struct {
	Status       string    `json:"status"`
	Scanned      bool      `json:"scanned"`
	RefreshCount int       `json:"refreshCount"`
	Err          string    `json:"err,omitempty"`
	UpdatedAt    time.Time `json:"updatedAt"`
	QRContent    string    `json:"-"` // 二维码内容（内部渲染用，不出 JSON）
}

// Options 登录会话参数。
type Options struct {
	LoginTimeout  time.Duration // 总超时（默认 60 分钟；桥场景用户可随时扫）
	MaxRefreshes  int           // 二维码刷新上限（默认 30 次 ≈ 60 分钟窗口）
	PollTimeout   time.Duration // 单次状态轮询超时（默认 40s，含 35s 长轮询）
	VerifyWait    time.Duration // 验证码输入等待上限（默认 5 分钟）
	OnStateChange func(State)   // 状态变化回调（可选）
}

func (o *Options) applyDefaults() {
	if o.LoginTimeout <= 0 {
		o.LoginTimeout = 60 * time.Minute
	}
	if o.MaxRefreshes <= 0 {
		o.MaxRefreshes = 30
	}
	if o.PollTimeout <= 0 {
		o.PollTimeout = 40 * time.Second
	}
	if o.VerifyWait <= 0 {
		o.VerifyWait = 5 * time.Minute
	}
}

// Session 一次登录会话（内存持有二维码；线程安全）。
type Session struct {
	baseURL     string
	localTokens []string
	opts        Options

	ctx    context.Context
	cancel context.CancelFunc

	mu    sync.Mutex
	state State
	code  string // 当前 qrcode token
	creds *Credentials
	err   error

	verifyCh  chan string
	refreshCh chan struct{}
	doneCh    chan struct{}
	started   bool
}

// NewSession 创建登录会话（baseURL 为初始接入点；localTokens 为本地已绑定 token）。
func NewSession(baseURL string, localTokens []string, opts Options) *Session {
	opts.applyDefaults()
	ctx, cancel := context.WithCancel(context.Background())
	return &Session{
		baseURL:     strings.TrimRight(baseURL, "/"),
		localTokens: localTokens,
		opts:        opts,
		ctx:         ctx,
		cancel:      cancel,
		state:       State{Status: StatusWait, UpdatedAt: time.Now()},
		verifyCh:    make(chan string, 1),
		refreshCh:   make(chan struct{}, 1),
		doneCh:      make(chan struct{}),
	}
}

// Start 启动状态机（异步）。
func (s *Session) Start() {
	if s.started {
		return
	}
	s.started = true
	go s.run()
}

// Snapshot 当前状态快照。
func (s *Session) Snapshot() State {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state
}

// QRContent 当前二维码内容（为空表示暂无二维码）。
func (s *Session) QRContent() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state.QRContent
}

// SubmitVerifyCode 注入手机端显示的验证码（仅在 need_verifycode 阶段接受）。
// 采用 1 缓冲通道：刚进入等待窗口的提交不会丢失。
func (s *Session) SubmitVerifyCode(code string) bool {
	code = strings.TrimSpace(code)
	if code == "" {
		return false
	}
	select {
	case s.verifyCh <- code:
		return true
	default:
		return false
	}
}

// RefreshQR 请求手动刷新二维码（异步生效）。
func (s *Session) RefreshQR() {
	select {
	case s.refreshCh <- struct{}{}:
	default:
	}
}

// Stop 停止会话（幂等；状态机将尽快退出）。
func (s *Session) Stop() {
	s.cancel()
}

// Wait 阻塞直到会话结束，返回凭据或错误。
func (s *Session) Wait() (*Credentials, error) {
	<-s.doneCh
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.creds, s.err
}

// Done 返回会话结束信号通道。
func (s *Session) Done() <-chan struct{} { return s.doneCh }

// ─────────────────────────── 状态机 ───────────────────────────

func (s *Session) run() {
	defer close(s.doneCh)
	defer func() {
		if p := recover(); p != nil {
			s.finishFailed(fmt.Errorf("登录状态机内部错误(panic): %v", p))
		}
	}()
	if err := s.fetchQR(); err != nil {
		s.finishFailed(fmt.Errorf("获取二维码失败: %w", err))
		return
	}

	deadline := time.Now().Add(s.opts.LoginTimeout)
	pendingCode := ""

	for {
		if s.ctx.Err() != nil {
			s.finishStopped()
			return
		}
		if time.Now().After(deadline) {
			s.finishFailed(ErrTimeout)
			return
		}
		// 手动刷新请求
		select {
		case <-s.refreshCh:
			if !s.refreshQR("manual") {
				return
			}
		default:
		}

		status, raw := s.pollOnce(pendingCode)

		switch status {
		case StatusWait:
			// 继续

		case StatusScaned:
			pendingCode = ""
			s.setState(func(st *State) {
				st.Status = StatusScaned
				st.Scanned = true
				st.Err = ""
			})

		case StatusNeedVerify:
			s.setState(func(st *State) { st.Status = StatusNeedVerify })
			code, ok := s.awaitVerify()
			if !ok {
				if s.ctx.Err() != nil {
					s.finishStopped()
				} else {
					s.finishFailed(fmt.Errorf("等待验证码输入超时"))
				}
				return
			}
			pendingCode = code
			continue // 不等待 1s，立即再轮询

		case StatusVerifyBlocked:
			pendingCode = ""
			s.setState(func(st *State) {
				st.Status = StatusVerifyBlocked
				st.Err = "验证码多次输入错误，将自动刷新二维码"
			})
			if !s.refreshQR("verify_code_blocked") {
				return
			}

		case StatusExpired:
			pendingCode = ""
			if !s.refreshQR("expired") {
				return
			}

		case StatusRedirect:
			host := mapStr(raw, "redirect_host")
			if host != "" {
				s.mu.Lock()
				s.baseURL = "https://" + host
				s.mu.Unlock()
				s.setState(func(st *State) {
					st.Status = StatusRedirect
					st.Err = "登录 IDC 重定向 → " + host
				})
			}

		case StatusBinded:
			s.finishFailed(ErrBinded)
			return

		case StatusConfirmed:
			token := mapStr(raw, "bot_token")
			botID := mapStr(raw, "ilink_bot_id")
			if token == "" || botID == "" {
				s.finishFailed(fmt.Errorf("登录确认但缺少 token/bot_id"))
				return
			}
			base := mapStr(raw, "baseurl")
			if base == "" {
				base = s.currentBase()
			}
			creds := &Credentials{
				Token:   token,
				BaseURL: base,
				BotID:   botID,
				UserID:  mapStr(raw, "ilink_user_id"),
				SavedAt: time.Now().UTC().Format(time.RFC3339),
			}
			s.mu.Lock()
			s.creds = creds
			s.mu.Unlock()
			s.setState(func(st *State) {
				st.Status = StatusConfirmed
				st.Err = ""
				st.QRContent = ""
			})
			return

		case StatusStopped:
			s.finishStopped()
			return

		default:
			// 未知状态：忽略继续（协议韧性）
		}

		select {
		case <-time.After(1 * time.Second):
		case <-s.ctx.Done():
			s.finishStopped()
			return
		}
	}
}

// pollOnce 一轮状态轮询。超时/网络错误按 wait 处理（对齐 M1 语义）。
func (s *Session) pollOnce(verifyCode string) (string, map[string]any) {
	m, err := ilink.PollQrStatusCtx(s.ctx, s.currentBase(), s.currentQR(), verifyCode, s.opts.PollTimeout)
	if err != nil {
		if s.ctx.Err() != nil {
			return StatusStopped, nil
		}
		return StatusWait, nil
	}
	st := mapStr(m, "status")
	if st == "" {
		st = StatusWait
	}
	return st, m
}

// fetchQR 拉取新二维码（内存持有；不落盘）。
func (s *Session) fetchQR() error {
	resp, err := ilink.FetchQrCodeCtx(s.ctx, s.currentBase(), s.localTokens)
	if err != nil {
		return err
	}
	if resp.Qrcode == "" {
		return fmt.Errorf("服务器未返回二维码")
	}
	s.mu.Lock()
	s.code = resp.Qrcode
	s.mu.Unlock()
	s.setState(func(st *State) {
		st.Status = StatusWait
		st.QRContent = resp.QrcodeImgContent
		st.Scanned = false
		st.Err = ""
	})
	return nil
}

// refreshQR 刷新二维码（计数 + 上限保护）。
func (s *Session) refreshQR(reason string) bool {
	s.mu.Lock()
	s.state.RefreshCount++
	count := s.state.RefreshCount
	blocked := count > s.opts.MaxRefreshes
	s.mu.Unlock()
	if blocked {
		s.finishFailed(fmt.Errorf("二维码多次过期/失败（%d 次，%s），登录中止", s.opts.MaxRefreshes, reason))
		return false
	}
	if err := s.fetchQR(); err != nil {
		s.finishFailed(fmt.Errorf("刷新二维码失败: %w", err))
		return false
	}
	return true
}

// awaitVerify 等待验证码注入。
func (s *Session) awaitVerify() (string, bool) {
	select {
	case code := <-s.verifyCh:
		return code, true
	case <-time.After(s.opts.VerifyWait):
		return "", false
	case <-s.ctx.Done():
		return "", false
	}
}

// ─────────────────────────── 终态与工具 ───────────────────────────

func (s *Session) finishFailed(err error) {
	s.mu.Lock()
	s.err = err
	s.mu.Unlock()
	s.setState(func(st *State) {
		st.Status = StatusFailed
		st.Err = err.Error()
		st.QRContent = ""
	})
}

func (s *Session) finishStopped() {
	s.mu.Lock()
	if s.err == nil {
		s.err = ErrStopped
	}
	s.mu.Unlock()
	s.setState(func(st *State) {
		if st.Status != StatusFailed {
			st.Status = StatusStopped
		}
		st.QRContent = ""
	})
}

// setState 修改状态并触发回调（回调在锁外调用，避免死锁）。
func (s *Session) setState(mut func(*State)) {
	s.mu.Lock()
	mut(&s.state)
	s.state.UpdatedAt = time.Now()
	cb := s.opts.OnStateChange
	st := s.state
	s.mu.Unlock()
	if cb != nil {
		cb(st)
	}
}

func (s *Session) currentBase() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.baseURL
}

func (s *Session) currentQR() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.code
}

func mapStr(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, _ := m[key].(string)
	return v
}
