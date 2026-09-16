// Package localhttp 实现微信桥的本地回环 HTTP 面（仅绑定 127.0.0.1）。
//
// 消费方：
//   - 插件壳（宿主 Node 进程）：状态查询与控制（再经壳的 /api/ext/wechat-bridge/* 代理给浏览器）；
//   - 浏览器（扫码页）：/qr 页面 + /qr/state + /qr.png 实时二维码（内存渲染、不落盘）；
//   - Agent 工具（wechat_send 等，P2/P3 接线）。
//
// 安全模型（方案决策⑦「端口+token 轻校验」）：
//   - 仅 127.0.0.1 绑定（外层 localhost，不对外）；
//   - 读操作放行；写操作（POST）在配置了 token 时校验 X-Bridge-Token；
//   - 写操作要求 Content-Type: application/json（防浏览器简单表单 CSRF）。
package localhttp

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/hoonfeng/paircode/plugins-src/plugins/wechat-bridge/login"
	"github.com/hoonfeng/paircode/plugins-src/plugins/wechat-bridge/qr"
)

//go:embed qr.html
var qrPageHTML string

// SessionLike 登录会话最小接口（由 login.Session 实现）。
type SessionLike interface {
	Snapshot() login.State
	QRContent() string
	SubmitVerifyCode(code string) bool
	RefreshQR()
}

// Options 依赖注入（main 接线；nil 表示该能力未启用，相关端点返回 501）。
type Options struct {
	Token      string                                         // 写操作校验 token（空=不校验，调试模式）
	Login      func() SessionLike                             // 当前登录会话（nil=无）
	Status     func() map[string]any                          // /status 数据（由 account.Manager 提供）
	Accounts   func() []map[string]any                        // 可选：/accounts 列表
	RemoveAcct func(id string) error                          // 可选：移除账号
	Relogin    func(id string) error                          // 可选：重新登录
	AddAcct    func(alias string) (string, error)             // 可选：新增账号
	Send       func(account, to, text string) error           // 可选：主动发送文本（Agent 工具 wechat_send）
	Contacts   func(account string) ([]map[string]any, error) // 可选：联系人列表（wechat_contacts）
	SendMedia  func(account, to, file, text string) error     // 可选：主动发送媒体（wechat_send 的 file 参数）
}

// Server 本地 HTTP 服务。
type Server struct {
	opts Options

	mu     sync.Mutex
	srv    *http.Server
	ln     net.Listener
	port   int
	rev    int
	lastQR string
}

// New 创建服务。
func New(opts Options) *Server { return &Server{opts: opts} }

// ListenLocal 在 127.0.0.1 上监听；端口冲突自动向后避让（最多 9 次）。
func ListenLocal(port int) (net.Listener, int, error) {
	if port <= 0 {
		port = 9097
	}
	var lastErr error
	for i := 0; i <= 9; i++ {
		ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port+i))
		if err == nil {
			return ln, ln.Addr().(*net.TCPAddr).Port, nil
		}
		lastErr = err
	}
	return nil, 0, fmt.Errorf("端口 %d~%d 均不可用: %v", port, port+9, lastErr)
}

// Start 启动服务并返回实际端口。
func (s *Server) Start(port int) (int, error) {
	mux := http.NewServeMux()
	s.routes(mux)

	ln, actual, err := ListenLocal(port)
	if err != nil {
		return 0, err
	}
	s.mu.Lock()
	s.ln = ln
	s.port = actual
	s.srv = &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	s.mu.Unlock()
	go func() { _ = s.srv.Serve(ln) }()
	return actual, nil
}

// Stop 关闭服务（幂等）。
func (s *Server) Stop() {
	s.mu.Lock()
	srv := s.srv
	s.mu.Unlock()
	if srv != nil {
		_ = srv.Close()
	}
}

// Port 返回实际监听端口（Start 后有效）。
func (s *Server) Port() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.port
}

// Handler 返回路由（测试用）。
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	s.routes(mux)
	return mux
}

// routes 装配全部路由（Start 与 Handler 共用，避免两处漂移）。
func (s *Server) routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("GET /status", s.handleStatus)
	mux.HandleFunc("GET /qr", s.handleQRPage)
	mux.HandleFunc("GET /qr.png", s.handleQRPng)
	mux.HandleFunc("GET /qr/state", s.handleQRState)
	mux.HandleFunc("POST /qr/verify", s.writeAuth(s.handleQRVerify))
	mux.HandleFunc("POST /qr/refresh", s.writeAuth(s.handleQRRefresh))
	mux.HandleFunc("GET /accounts", s.handleAccounts)
	mux.HandleFunc("POST /accounts", s.writeAuth(s.handleAccountAdd))
	mux.HandleFunc("DELETE /accounts", s.writeAuth(s.handleAccountRemove))
	mux.HandleFunc("POST /relogin", s.writeAuth(s.handleRelogin))
	mux.HandleFunc("POST /login/new", s.writeAuth(s.handleLoginNew))
	mux.HandleFunc("POST /send", s.writeAuth(s.handleSend))
	mux.HandleFunc("GET /contacts", s.handleContacts)
}

// ─────────────────────────── 基础端点 ───────────────────────────

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{
		"ok":   true,
		"pid":  os.Getpid(),
		"port": s.Port(),
		"ts":   time.Now().Unix(),
	})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	st := map[string]any{}
	if s.opts.Status != nil {
		st = s.opts.Status()
	}
	st["ok"] = true
	st["pid"] = os.Getpid()
	st["port"] = s.Port()
	writeJSON(w, 200, st)
}

// ─────────────────────────── 二维码（登录） ───────────────────────────

func (s *Server) currentSession() SessionLike {
	if s.opts.Login == nil {
		return nil
	}
	return s.opts.Login()
}

// handleQRPage 扫码页（内嵌 HTML；JS 轮询 /qr/state 实时更新）。
func (s *Server) handleQRPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = io.WriteString(w, qrPageHTML)
}

// handleQRPng 实时渲染当前二维码 PNG（内存渲染，每次现渲染；无码 404）。
func (s *Server) handleQRPng(w http.ResponseWriter, r *http.Request) {
	sess := s.currentSession()
	if sess == nil {
		http.Error(w, "no login session", http.StatusNotFound)
		return
	}
	content := sess.QRContent()
	if content == "" {
		http.Error(w, "no qr", http.StatusNotFound)
		return
	}
	png, err := qr.RenderPNG(content, qr.DefaultScale)
	if err != nil {
		http.Error(w, "qr render: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(png)
}

// handleQRState 扫码页轮询的状态 JSON。
func (s *Server) handleQRState(w http.ResponseWriter, r *http.Request) {
	sess := s.currentSession()
	if sess == nil {
		writeJSON(w, 200, map[string]any{"status": "none", "hasQR": false, "rev": 0})
		return
	}
	st := sess.Snapshot()
	writeJSON(w, 200, map[string]any{
		"status":       st.Status,
		"scanned":      st.Scanned,
		"refreshCount": st.RefreshCount,
		"err":          st.Err,
		"hasQR":        st.QRContent != "",
		"rev":          s.bumpRev(st.QRContent),
	})
}

// bumpRev 二维码内容变化时递增版本号（供前端 img 缓存失效）。
func (s *Server) bumpRev(content string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	if content != s.lastQR {
		s.lastQR = content
		s.rev++
	}
	return s.rev
}

func (s *Server) handleQRVerify(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Code string `json:"code"`
	}
	if !readJSONBody(w, r, &body) {
		return
	}
	sess := s.currentSession()
	if sess == nil {
		writeJSON(w, 409, errJSON("无登录会话"))
		return
	}
	if !sess.SubmitVerifyCode(body.Code) {
		writeJSON(w, 409, errJSON("当前不需要验证码输入"))
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}

func (s *Server) handleQRRefresh(w http.ResponseWriter, r *http.Request) {
	sess := s.currentSession()
	if sess == nil {
		writeJSON(w, 409, errJSON("无登录会话"))
		return
	}
	sess.RefreshQR()
	writeJSON(w, 200, map[string]any{"ok": true})
}

// ─────────────────────────── 账号 ───────────────────────────

func (s *Server) handleAccounts(w http.ResponseWriter, r *http.Request) {
	if s.opts.Accounts == nil {
		writeJSON(w, 501, errJSON("未接线"))
		return
	}
	writeJSON(w, 200, map[string]any{"accounts": s.opts.Accounts()})
}

func (s *Server) handleAccountAdd(w http.ResponseWriter, r *http.Request) {
	if s.opts.AddAcct == nil {
		writeJSON(w, 501, errJSON("未接线"))
		return
	}
	var body struct {
		Alias string `json:"alias"`
	}
	_ = json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&body)
	id, err := s.opts.AddAcct(strings.TrimSpace(body.Alias))
	if err != nil {
		writeJSON(w, 409, errJSON(err.Error()))
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true, "id": id})
}

func (s *Server) handleAccountRemove(w http.ResponseWriter, r *http.Request) {
	if s.opts.RemoveAcct == nil {
		writeJSON(w, 501, errJSON("未接线"))
		return
	}
	id := r.URL.Query().Get("id")
	if id == "" {
		writeJSON(w, 400, errJSON("缺少 id"))
		return
	}
	if err := s.opts.RemoveAcct(id); err != nil {
		writeJSON(w, 409, errJSON(err.Error()))
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}

// handleRelogin 重新登录（POST {account}）：清除旧凭据并进入扫码流程。
func (s *Server) handleRelogin(w http.ResponseWriter, r *http.Request) {
	if s.opts.Relogin == nil {
		writeJSON(w, 501, errJSON("未接线"))
		return
	}
	var body struct {
		Account string `json:"account"`
	}
	_ = json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&body)
	acc := strings.TrimSpace(body.Account)
	if acc == "" {
		writeJSON(w, 400, errJSON("缺少 account"))
		return
	}
	if err := s.opts.Relogin(acc); err != nil {
		writeJSON(w, 409, errJSON(err.Error()))
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}

// handleLoginNew 新增账号（POST {alias?}）→ 进入扫码流程。
func (s *Server) handleLoginNew(w http.ResponseWriter, r *http.Request) {
	if s.opts.AddAcct == nil {
		writeJSON(w, 501, errJSON("未接线"))
		return
	}
	var body struct {
		Alias string `json:"alias"`
	}
	_ = json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&body)
	id, err := s.opts.AddAcct(strings.TrimSpace(body.Alias))
	if err != nil {
		writeJSON(w, 409, errJSON(err.Error()))
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true, "id": id})
}

// handleSend 主动发送文本（POST {account?, to, text, file?}）。
// file（媒体）为后续扩展字段：当前明确拒绝，避免调用方误以为已发送。
func (s *Server) handleSend(w http.ResponseWriter, r *http.Request) {
	if s.opts.Send == nil {
		writeJSON(w, 501, errJSON("未接线"))
		return
	}
	var body struct {
		Account string `json:"account"`
		To      string `json:"to"`
		Text    string `json:"text"`
		File    string `json:"file"`
	}
	if !readJSONBody(w, r, &body) {
		return
	}
	if body.File != "" {
		if s.opts.SendMedia == nil {
			writeJSON(w, 501, errJSON("媒体发送未接线"))
			return
		}
		if strings.TrimSpace(body.To) == "" {
			writeJSON(w, 400, errJSON("缺少 to（收件人）"))
			return
		}
		if err := s.opts.SendMedia(strings.TrimSpace(body.Account), strings.TrimSpace(body.To),
			strings.TrimSpace(body.File), strings.TrimSpace(body.Text)); err != nil {
			writeJSON(w, 409, errJSON(err.Error()))
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true, "sentAt": time.Now().Format(time.RFC3339), "file": body.File})
		return
	}
	if strings.TrimSpace(body.To) == "" {
		writeJSON(w, 400, errJSON("缺少 to（收件人）"))
		return
	}
	if strings.TrimSpace(body.Text) == "" {
		writeJSON(w, 400, errJSON("缺少 text（消息内容）"))
		return
	}
	if err := s.opts.Send(strings.TrimSpace(body.Account), strings.TrimSpace(body.To), body.Text); err != nil {
		writeJSON(w, 409, errJSON(err.Error()))
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true, "sentAt": time.Now().Format(time.RFC3339)})
}

// handleContacts 联系人列表（GET ?account=）。
func (s *Server) handleContacts(w http.ResponseWriter, r *http.Request) {
	if s.opts.Contacts == nil {
		writeJSON(w, 501, errJSON("未接线"))
		return
	}
	list, err := s.opts.Contacts(strings.TrimSpace(r.URL.Query().Get("account")))
	if err != nil {
		writeJSON(w, 409, errJSON(err.Error()))
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true, "contacts": list})
}

// ─────────────────────────── 中间件与工具 ───────────────────────────

// writeAuth 写操作防护：token 校验（配置了才校验）+ JSON Content-Type。
func (s *Server) writeAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.opts.Token != "" && r.Header.Get("X-Bridge-Token") != s.opts.Token {
			writeJSON(w, 401, errJSON("unauthorized"))
			return
		}
		if ct := r.Header.Get("Content-Type"); !strings.Contains(ct, "application/json") {
			writeJSON(w, 415, errJSON("Content-Type 必须是 application/json"))
			return
		}
		next(w, r)
	}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func errJSON(msg string) map[string]any {
	return map[string]any{"ok": false, "error": msg}
}

func readJSONBody(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(v); err != nil {
		writeJSON(w, 400, errJSON("请求体解析失败: "+err.Error()))
		return false
	}
	return true
}
