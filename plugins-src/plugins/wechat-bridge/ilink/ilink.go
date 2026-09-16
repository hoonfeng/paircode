// Package ilink 实现 ilink bot 协议客户端（移植自 M1 ilink.mjs，含官方实现语义）。
//
// 协议要点：
//   - 接入点 https://ilinkai.weixin.qq.com，端点前缀 /ilink/bot/<action>；
//   - 请求体统一包 base_info.channel_version；鉴权头 Authorization: Bearer <token>；
//   - X-WECHAT-UIN：随机 uint32 十进制字符串的 base64（每次请求新生成）；
//   - getupdates 为 35s 长轮询：超时视为空批（ret=0, msgs=[]）而非错误；
//   - 长轮询/瞬时网络错误按退避重试；-14（token 暂失效）语义由上层（bridge）处理。
package ilink

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// 协议常量（对齐 M1 ilink.mjs）。
const (
	ChannelVersion = "2.0.0"
	ClientVersion  = "131072" // uint32 编码 major<<16|minor<<8|patch（2.0.0）
	DefaultBaseURL = "https://ilinkai.weixin.qq.com"
)

// httpClient 复用连接池（长轮询多，Keep-Alive 重要；超时按请求 context 控制）。
var httpClient = &http.Client{
	Transport: &http.Transport{
		MaxIdleConns:        8,
		MaxIdleConnsPerHost: 4,
		IdleConnTimeout:     90 * time.Second,
	},
}

// RandomWechatUin 生成 X-WECHAT-UIN 头值：随机 uint32 十进制字符串的 base64。
func RandomWechatUin() string {
	var b [4]byte
	_, _ = rand.Read(b[:])
	n := binary.BigEndian.Uint32(b[:])
	return base64.StdEncoding.EncodeToString([]byte(strconv.FormatUint(uint64(n), 10)))
}

// BuildHeaders 构造协议请求头（token 为空则不带 Authorization）。
func BuildHeaders(token string) map[string]string {
	h := map[string]string{
		"Content-Type":            "application/json",
		"AuthorizationType":       "ilink_bot_token",
		"X-WECHAT-UIN":            RandomWechatUin(),
		"iLink-App-Id":            "bot",
		"iLink-App-ClientVersion": ClientVersion,
	}
	if token != "" {
		h["Authorization"] = "Bearer " + token
	}
	return h
}

// DoJSON 执行单次 HTTP 请求并返回响应体；非 2xx 视为错误。
// 供登录流程（裸请求）与 Client（协议请求）共用。
func DoJSON(method, reqURL string, headers map[string]string, body any, timeout time.Duration) ([]byte, error) {
	return DoJSONCtx(context.Background(), method, reqURL, headers, body, timeout)
}

// DoJSONCtx 带外部 ctx 的 DoJSON（ctx 取消/超时立即中断请求，供登录会话停止用）。
func DoJSONCtx(ctx context.Context, method, reqURL string, headers map[string]string, body any, timeout time.Duration) ([]byte, error) {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(data)
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, method, reqURL, reader)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncate(string(data), 200))
	}
	return data, nil
}

// IsTimeout 判断错误是否为超时（context 截止或网络超时）。
func IsTimeout(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var ne net.Error
	return errors.As(err, &ne) && ne.Timeout()
}

// isTransient 瞬时网络错误（连接复位/拒绝/断开等，可安全重试）。
func isTransient(err error) bool {
	if err == nil || IsTimeout(err) {
		return false
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, net.ErrClosed) {
		return true
	}
	msg := strings.ToLower(err.Error())
	for _, k := range []string{"connection reset", "connection refused", "broken pipe", "network is unreachable", "unexpected eof", "eof"} {
		if strings.Contains(msg, k) {
			return true
		}
	}
	return false
}

// FlexString 兼容 JSON string/number 的字符串字段（message_id/seq 等 id 类字段）。
type FlexString string

func (s *FlexString) UnmarshalJSON(b []byte) error {
	if len(b) == 0 || string(b) == "null" {
		return nil
	}
	if b[0] == '"' {
		var v string
		if err := json.Unmarshal(b, &v); err != nil {
			return err
		}
		*s = FlexString(v)
		return nil
	}
	*s = FlexString(string(b))
	return nil
}

func (s FlexString) String() string { return string(s) }

// ─────────────────────────── 类型 ───────────────────────────

// QrCodeResp get_bot_qrcode 响应。
type QrCodeResp struct {
	Qrcode           string `json:"qrcode"`             // 二维码 token（状态轮询用）
	QrcodeImgContent string `json:"qrcode_img_content"` // 二维码内容（渲染用）
}

// GetUpdatesResp getupdates 响应。
type GetUpdatesResp struct {
	Ret           int               `json:"ret"`
	Errcode       int               `json:"errcode"`
	Errmsg        string            `json:"errmsg"`
	GetUpdatesBuf string            `json:"get_updates_buf"`
	Msgs          []json.RawMessage `json:"msgs"` // 逐条弱解析（坏条容错）
}

// RawMsg 上行原始消息（已知字段；媒体字段在 P2-4 抓样后补全）。
type RawMsg struct {
	MessageType  int        `json:"message_type"` // 1=USER
	MessageID    FlexString `json:"message_id"`
	Seq          FlexString `json:"seq"`
	FromUserID   string     `json:"from_user_id"`
	ContextToken string     `json:"context_token"`
	ItemList     []MsgItem  `json:"item_list"`
}

// MsgItem 消息条目。
type MsgItem struct {
	Type     int       `json:"type"` // 1=文本 2=图片 3=语音 4=文件 5=视频
	TextItem *TextItem `json:"text_item,omitempty"`
	RefMsg   *RefMsg   `json:"ref_msg,omitempty"`

	// 媒体条目（P2-4 抓样补全；类型 2/3/4/5）
	ImageItem *ImageItem `json:"image_item,omitempty"`
	VoiceItem *VoiceItem `json:"voice_item,omitempty"`
	FileItem  *FileItem  `json:"file_item,omitempty"`
	VideoItem *VideoItem `json:"video_item,omitempty"`
}

// CDNMedia CDN 媒体引用（收/发同构）。
// aes_key 为 base64，内层可能是 16 字节原始 key 或 32 字符 hex 字符串（两种都在线见过）。
type CDNMedia struct {
	EncryptQueryParam string `json:"encrypt_query_param,omitempty"`
	AESKey            string `json:"aes_key,omitempty"`
	EncryptType       int    `json:"encrypt_type,omitempty"`
	FullURL           string `json:"full_url,omitempty"`
}

// ImageItem 图片条目。
type ImageItem struct {
	Media       *CDNMedia `json:"media,omitempty"`
	ThumbMedia  *CDNMedia `json:"thumb_media,omitempty"`
	AESKey      string    `json:"aeskey,omitempty"` // 原始 AES key（hex 字符串），优先于 media.aes_key
	MidSize     int64     `json:"mid_size,omitempty"`
	ThumbSize   int64     `json:"thumb_size,omitempty"`
	HdSize      int64     `json:"hd_size,omitempty"`
	ThumbWidth  int       `json:"thumb_width,omitempty"`
	ThumbHeight int       `json:"thumb_height,omitempty"`
}

// VoiceItem 语音条目（含微信侧 ASR 转写 text）。
type VoiceItem struct {
	Media         *CDNMedia `json:"media,omitempty"`
	EncodeType    int       `json:"encode_type,omitempty"`
	BitsPerSample int       `json:"bits_per_sample,omitempty"`
	SampleRate    int       `json:"sample_rate,omitempty"`
	Playtime      int       `json:"playtime,omitempty"` // 毫秒
	Text          string    `json:"text,omitempty"`     // ASR 转写
}

// FileItem 文件条目。
type FileItem struct {
	Media    *CDNMedia  `json:"media,omitempty"`
	FileName string     `json:"file_name,omitempty"`
	MD5      string     `json:"md5,omitempty"`
	Len      FlexString `json:"len,omitempty"` // 明文字节数（协议为字符串）
}

// VideoItem 视频条目。
type VideoItem struct {
	Media       *CDNMedia `json:"media,omitempty"`
	ThumbMedia  *CDNMedia `json:"thumb_media,omitempty"`
	VideoSize   int64     `json:"video_size,omitempty"`  // 明文字节数
	PlayLength  int       `json:"play_length,omitempty"` // 秒
	VideoMD5    string    `json:"video_md5,omitempty"`
	ThumbSize   int64     `json:"thumb_size,omitempty"`
	ThumbWidth  int       `json:"thumb_width,omitempty"`
	ThumbHeight int       `json:"thumb_height,omitempty"`
}

// TextItem 文本条目。
type TextItem struct {
	Text string `json:"text"`
}

// RefMsg 引用消息。
type RefMsg struct {
	Title       string   `json:"title"`
	MessageItem *MsgItem `json:"message_item,omitempty"`
}

// SendTextResp sendmessage 响应。
type SendTextResp struct {
	Ret     int    `json:"ret"`
	Errcode int    `json:"errcode"`
	Errmsg  string `json:"errmsg"`
}

// GetUploadUrlResp getuploadurl 响应（发媒体前的 CDN 预签名参数）。
type GetUploadUrlResp struct {
	Ret              int    `json:"ret"`
	Errcode          int    `json:"errcode"`
	Errmsg           string `json:"errmsg"`
	UploadParam      string `json:"upload_param"`
	ThumbUploadParam string `json:"thumb_upload_param"`
	UploadFullURL    string `json:"upload_full_url"`
}

// GetConfigResp getconfig 响应。
type GetConfigResp struct {
	Ret          int    `json:"ret"`
	Errcode      int    `json:"errcode"`
	Errmsg       string `json:"errmsg"`
	TypingTicket string `json:"typing_ticket"`
}

// SendTypingResp sendtyping 响应。
type SendTypingResp struct {
	Ret     int    `json:"ret"`
	Errcode int    `json:"errcode"`
	Errmsg  string `json:"errmsg"`
}

// ─────────────────────────── 登录裸请求 ───────────────────────────

// FetchQrCode 获取登录二维码（POST get_bot_qrcode?bot_type=3）。
// localTokenList 带入本地已绑定 token，服务器可识别「已绑定」场景（binded_redirect）。
func FetchQrCode(baseURL string, localTokenList []string) (*QrCodeResp, error) {
	return FetchQrCodeCtx(context.Background(), baseURL, localTokenList)
}

// FetchQrCodeCtx 带 ctx 的 FetchQrCode。
func FetchQrCodeCtx(ctx context.Context, baseURL string, localTokenList []string) (*QrCodeResp, error) {
	if localTokenList == nil {
		localTokenList = []string{}
	}
	headers := map[string]string{
		"Content-Type":            "application/json",
		"iLink-App-Id":            "bot",
		"iLink-App-ClientVersion": ClientVersion,
	}
	data, err := DoJSONCtx(ctx, "POST", strings.TrimRight(baseURL, "/")+"/ilink/bot/get_bot_qrcode?bot_type=3",
		headers, map[string]any{"local_token_list": localTokenList}, 15*time.Second)
	if err != nil {
		return nil, err
	}
	var resp QrCodeResp
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("二维码响应解析失败: %w", err)
	}
	return &resp, nil
}

// PollQrStatus 查询扫码状态（长轮询；超时/网络错误由调用方按 wait 处理）。
func PollQrStatus(baseURL, qrcode, verifyCode string, timeout time.Duration) (map[string]any, error) {
	return PollQrStatusCtx(context.Background(), baseURL, qrcode, verifyCode, timeout)
}

// PollQrStatusCtx 带 ctx 的 PollQrStatus。
func PollQrStatusCtx(ctx context.Context, baseURL, qrcode, verifyCode string, timeout time.Duration) (map[string]any, error) {
	u := strings.TrimRight(baseURL, "/") + "/ilink/bot/get_qrcode_status?qrcode=" + url.QueryEscape(qrcode)
	if verifyCode != "" {
		u += "&verify_code=" + url.QueryEscape(verifyCode)
	}
	headers := map[string]string{
		"iLink-App-Id":            "bot",
		"iLink-App-ClientVersion": ClientVersion,
	}
	data, err := DoJSONCtx(ctx, "GET", u, headers, nil, timeout)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("状态响应解析失败: %w", err)
	}
	return m, nil
}

// ─────────────────────────── 协议客户端 ───────────────────────────

// Client 登录后的协议客户端（baseURL/token 可随重登/redirect 更换）。
type Client struct {
	BaseURL string
	Token   string

	// Logf 可选日志（桥注入；nil 静默）。
	Logf func(format string, args ...any)

	// ── 发送节流与限流退避（2026-09-16 新增，防打爆服务端发送频率限制）──
	// 背景：连续发送大量消息（如流式多片连发）触发服务端频率限制：
	// sendmessage 返回 ret=-2 prepare failed。2026-09-16 两轮真机实测：
	//   第一轮：约 11 分钟内累计 ~15 条触发；静默 15m25s 后恢复；
	//   第二轮：恢复后 2 分钟内累计 10 条再次触发；静默 15m50s、31m 仍失败
	//   （期间重试尝试会续期限流窗口——静默等待不重试才是正解）。
	// 对策：
	//   1) 发送端节流：最小间隔 + 短窗口（默认 2m/8 条）+ 长窗口（默认 15m/12 条）；
	//   2) -2 冷静期：触发后暂停发送，冷静期（首次 20m、翻倍至上限 60m）
	//      结束后重试；重试成功后阶梯重置。
	sendMu         sync.Mutex
	sendLast       time.Time     // 上次 sendmessage 发出时刻（最小间隔）
	sendWindow     []time.Time   // 滑动窗口内发送时刻（配额）
	sendLongWindow []time.Time   // 长窗口内发送时刻（配额；防 15 分钟级累计限流）
	limitUntil     time.Time     // -2 冷静期截止（期内所有发送等待）
	limitBackoff   time.Duration // 当前冷静期长度（成功发送后重置 0）
	stopCh         chan struct{} // Close() 关闭：中断等待中的节流/退避睡眠
	stopOnce       sync.Once

	// 节流参数（零值 = 默认；测试可注入小值）
	minSendInterval   time.Duration // 默认 3s
	sendWindowMax     int           // 默认 8
	sendWindowDur     time.Duration // 默认 120s
	sendLongWindowMax int           // 默认 12（长窗口条数上限）
	sendLongWindowDur time.Duration // 默认 15m（长窗口时长）
	limitBaseBackoff  time.Duration // 默认 20m
	limitMaxBackoff   time.Duration // 默认 60m
	limitMaxRetries   int           // 默认 2
}

// NewClient 创建协议客户端。
func NewClient(baseURL, token string) *Client {
	return &Client{BaseURL: strings.TrimRight(baseURL, "/"), Token: token, stopCh: make(chan struct{})}
}

type postOpt struct {
	timeout         time.Duration
	retries         int
	tolerateTimeout bool
}

// ─────────────────────────── 发送节流与限流退避 ───────────────────────────
//
// 背景（2026-09-16 真机实测）：连续发送大量消息（如流式 48 片连发，
// 桥内 500ms 间隔）触发服务端发送频率限制，sendmessage 返回 ret=-2
// prepare failed 且约 15 分钟内持续失败；期间每次重试都会续期限流窗口，
// 低频（≤数条/分钟）发送全量正常。因此发送端必须自限流并长退避。
//
// 设计：
//   - 节流：sendmessage 统一经 acquireSendSlot（最小间隔 + 滑动窗口配额）；
//   - 退避：-2 后暂停发送（noteSendLimited），冷静期结束自动重试（sendmsg）；
//   - 可取消：等待以 select + stopCh 实现（Client.Close 中断，桥停止不阻塞退出）；
//   - 并发安全：共享状态（sendLast/sendWindow/limitUntil/limitBackoff）一律在
//     sendMu 保护下读写；
//   - 兼容：SendText/SendItem 签名不变；节流参数零值 = 安全默认值（见
//     SendThrottleOptions 注释），不配置时行为为「自限流 + 长退避」。

// SendThrottleOptions 发送节流参数（零值字段 = 默认值；桥从 config 注入）。
type SendThrottleOptions struct {
	MinInterval     time.Duration // sendmessage 最小间隔（默认 3s）
	WindowMax       int           // 短窗口内最大条数（默认 8）
	WindowDur       time.Duration // 短窗口时长（默认 120s）
	LongWindowMax   int           // 长窗口内最大条数（默认 12）
	LongWindowDur   time.Duration // 长窗口时长（默认 15m）
	LimitBase       time.Duration // ret=-2 首次冷静期（默认 20m）
	LimitMax        time.Duration // 冷静期上限（默认 60m）
	LimitMaxRetries int           // -2 最大重试次数（默认 2）
}

// SetSendThrottle 配置发送节流（bridge 构建客户端时注入；零值字段保持默认）。
func (c *Client) SetSendThrottle(opt SendThrottleOptions) {
	c.sendMu.Lock()
	defer c.sendMu.Unlock()
	if opt.MinInterval > 0 {
		c.minSendInterval = opt.MinInterval
	}
	if opt.WindowMax > 0 {
		c.sendWindowMax = opt.WindowMax
	}
	if opt.WindowDur > 0 {
		c.sendWindowDur = opt.WindowDur
	}
	if opt.LongWindowMax > 0 {
		c.sendLongWindowMax = opt.LongWindowMax
	}
	if opt.LongWindowDur > 0 {
		c.sendLongWindowDur = opt.LongWindowDur
	}
	if opt.LimitBase > 0 {
		c.limitBaseBackoff = opt.LimitBase
	}
	if opt.LimitMax > 0 {
		c.limitMaxBackoff = opt.LimitMax
	}
	if opt.LimitMaxRetries > 0 {
		c.limitMaxRetries = opt.LimitMaxRetries
	}
}

// Close 关闭客户端：中断所有等待中的节流/退避睡眠（桥停止时调用；幂等）。
func (c *Client) Close() {
	c.stopOnce.Do(func() {
		if c.stopCh != nil {
			close(c.stopCh)
		}
	})
}

func (c *Client) logf(format string, args ...any) {
	if c.Logf != nil {
		c.Logf(format, args...)
	}
}

// sendmsg 发送 sendmessage 请求（SendText/SendItem 共用）：节流 → 发送 →
// ret=-2 冷静期等待 + 重试。重试耗尽仍 -2 时返回原响应（上层日志/降级）。
func (c *Client) sendmsg(body map[string]any) (*SendTextResp, error) {
	maxRetries := c.limitMaxRetries
	if maxRetries <= 0 {
		maxRetries = 2
	}
	for attempt := 0; ; attempt++ {
		if err := c.acquireSendSlot(); err != nil {
			return nil, err
		}
		resp, err := post[SendTextResp](c, "ilink/bot/sendmessage", body, postOpt{retries: 2})
		if err != nil {
			return nil, err
		}
		if resp != nil && (resp.Ret == -2 || resp.Errcode == -2) {
			wait := c.noteSendLimited()
			c.logf("sendmessage 触发服务端限流（ret=-2 prepare failed），冷静 %s 后重试（第 %d/%d 次）",
				wait.Round(time.Second), attempt+1, maxRetries)
			if attempt+1 > maxRetries {
				return resp, nil // 重试耗尽：保留 -2 响应交由上层处理
			}
			continue
		}
		c.noteSendOK()
		return resp, nil
	}
}

// acquireSendSlot 发送节流闸门：冷静期 → 最小间隔 → 滑动窗口配额；
// 需要等待时以 select+stopCh 可取消睡眠实现，醒后循环重查（其他请求
// 可能在此期间更新冷静期/占用配额）。
func (c *Client) acquireSendSlot() error {
	interval := c.minSendInterval
	if interval <= 0 {
		interval = 3 * time.Second
	}
	winMax := c.sendWindowMax
	if winMax <= 0 {
		winMax = 8
	}
	winDur := c.sendWindowDur
	if winDur <= 0 {
		winDur = 120 * time.Second
	}
	longMax := c.sendLongWindowMax
	if longMax <= 0 {
		longMax = 12
	}
	longDur := c.sendLongWindowDur
	if longDur <= 0 {
		longDur = 15 * time.Minute
	}
	for {
		c.sendMu.Lock()
		now := time.Now()
		var wait time.Duration
		reason := ""
		switch {
		case now.Before(c.limitUntil):
			wait = time.Until(c.limitUntil)
			reason = "限流冷静期"
		default:
			if !c.sendLast.IsZero() {
				if d := interval - now.Sub(c.sendLast); d > wait {
					wait = d
					reason = "最小间隔"
				}
			}
			// 滑出窗口
			cut := 0
			for cut < len(c.sendWindow) && now.Sub(c.sendWindow[cut]) >= winDur {
				cut++
			}
			if cut > 0 {
				c.sendWindow = append(c.sendWindow[:0], c.sendWindow[cut:]...)
			}
			if len(c.sendWindow) >= winMax {
				if d := c.sendWindow[0].Add(winDur).Sub(now); d > wait {
					wait = d
					reason = "发送窗口配额"
				}
			}
			// 长窗口（防 15 分钟级累计限流）
			lcut := 0
			for lcut < len(c.sendLongWindow) && now.Sub(c.sendLongWindow[lcut]) >= longDur {
				lcut++
			}
			if lcut > 0 {
				c.sendLongWindow = append(c.sendLongWindow[:0], c.sendLongWindow[lcut:]...)
			}
			if len(c.sendLongWindow) >= longMax {
				if d := c.sendLongWindow[0].Add(longDur).Sub(now); d > wait {
					wait = d
					reason = "长窗口配额"
				}
			}
		}
		if wait <= 0 {
			c.sendLast = now
			c.sendWindow = append(c.sendWindow, now)
			c.sendLongWindow = append(c.sendLongWindow, now)
			c.sendMu.Unlock()
			return nil
		}
		ch := c.stopCh
		c.sendMu.Unlock()
		if wait > 3*time.Second {
			c.logf("发送节流：等待 %s（%s）", wait.Round(time.Second), reason)
		}
		timer := time.NewTimer(wait)
		select {
		case <-timer.C:
		case <-ch: // ch 为 nil（零值 Client）时永久阻塞：等价不可取消
			timer.Stop()
			return fmt.Errorf("客户端已停止：发送等待被取消")
		}
	}
}

// noteSendLimited 记录一次 ret=-2：进入/延长冷静期（首次 15m，此后翻倍至
// 上限 30m）。返回本次冷静期长度。
func (c *Client) noteSendLimited() time.Duration {
	c.sendMu.Lock()
	defer c.sendMu.Unlock()
	base := c.limitBaseBackoff
	if base <= 0 {
		base = 20 * time.Minute
	}
	maxB := c.limitMaxBackoff
	if maxB <= 0 {
		maxB = 60 * time.Minute
	}
	if c.limitBackoff <= 0 {
		c.limitBackoff = base
	} else {
		c.limitBackoff *= 2
		if c.limitBackoff > maxB {
			c.limitBackoff = maxB
		}
	}
	c.limitUntil = time.Now().Add(c.limitBackoff)
	return c.limitBackoff
}

// noteSendOK 发送成功：重置冷静期阶梯。
func (c *Client) noteSendOK() {
	c.sendMu.Lock()
	c.limitBackoff = 0
	c.sendMu.Unlock()
}

// post 带 base_info 包装、超时与退避重试的 POST。
func post[T any](c *Client, endpoint string, body map[string]any, opt postOpt) (*T, error) {
	if opt.timeout <= 0 {
		opt.timeout = 15 * time.Second
	}
	body["base_info"] = map[string]any{"channel_version": ChannelVersion}
	reqURL := c.BaseURL + "/" + endpoint
	for attempt := 0; ; attempt++ {
		data, err := DoJSON("POST", reqURL, BuildHeaders(c.Token), body, opt.timeout)
		if err == nil {
			var t T
			if uerr := json.Unmarshal(data, &t); uerr != nil {
				return nil, fmt.Errorf("响应解析失败(%s): %w", endpoint, uerr)
			}
			return &t, nil
		}
		if IsTimeout(err) {
			if opt.tolerateTimeout {
				var zero T
				return &zero, nil // 长轮询超时 = 空批
			}
			return nil, fmt.Errorf("请求超时: %s", endpoint)
		}
		if isTransient(err) && attempt < opt.retries {
			time.Sleep(time.Duration(1000*(1<<attempt)) * time.Millisecond) // 1s, 2s, 4s…
			continue
		}
		return nil, err
	}
}

// GetUpdates 收消息（长轮询）。超时返回零值（ret=0, msgs=nil）语义为空批。
func (c *Client) GetUpdates(buf string, longPollSec int) (*GetUpdatesResp, error) {
	return post[GetUpdatesResp](c, "ilink/bot/getupdates",
		map[string]any{"get_updates_buf": buf},
		postOpt{timeout: time.Duration(longPollSec+5) * time.Second, tolerateTimeout: true})
}

// SendText 发送文本（回复必须带对应用户消息的 context_token）。
// msg.client_id 为随机 16 hex；message_type=2(BOT)/message_state=2(FINISH)。
// 统一经 sendmsg：发送节流 + ret=-2 冷静期退避（2026-09-16 新增）。
func (c *Client) SendText(toUserID, text, contextToken string) (*SendTextResp, error) {
	return c.sendmsg(map[string]any{
		"msg": map[string]any{
			"from_user_id":  "",
			"to_user_id":    toUserID,
			"client_id":     randomHex(8),
			"message_type":  2,
			"message_state": 2,
			"item_list":     []map[string]any{{"type": 1, "text_item": map[string]any{"text": text}}},
			"context_token": contextToken,
		},
	})
}

// SendItem 发送单条结构化消息条目（媒体等；item = item_list 的单个元素）。
// 与 SendText 同构：message_type=2(BOT)/message_state=2(FINISH)，必须带对应用户的 context_token。
// 统一经 sendmsg：发送节流 + ret=-2 冷静期退避（2026-09-16 新增）。
func (c *Client) SendItem(toUserID string, item map[string]any, contextToken string) (*SendTextResp, error) {
	return c.sendmsg(map[string]any{
		"msg": map[string]any{
			"from_user_id":  "",
			"to_user_id":    toUserID,
			"client_id":     randomHex(8),
			"message_type":  2,
			"message_state": 2,
			"item_list":     []map[string]any{item},
			"context_token": contextToken,
		},
	})
}

// GetUploadUrl 获取 CDN 上传预签名（发媒体；请求字段见 media 包 Upload）。
func (c *Client) GetUploadUrl(req map[string]any) (*GetUploadUrlResp, error) {
	return post[GetUploadUrlResp](c, "ilink/bot/getuploadurl", req, postOpt{timeout: 20 * time.Second, retries: 2})
}

// GetConfig 获取配置（typing_ticket 等）。
func (c *Client) GetConfig(ilinkUserID, contextToken string) (*GetConfigResp, error) {
	return post[GetConfigResp](c, "ilink/bot/getconfig", map[string]any{
		"ilink_user_id": ilinkUserID,
		"context_token": contextToken,
	}, postOpt{timeout: 10 * time.Second})
}

// SendTyping 发送「正在输入」状态（status 1=开始，2=停止）。
func (c *Client) SendTyping(ilinkUserID, typingTicket string, status int) (*SendTypingResp, error) {
	return post[SendTypingResp](c, "ilink/bot/sendtyping", map[string]any{
		"ilink_user_id": ilinkUserID,
		"typing_ticket": typingTicket,
		"status":        status,
	}, postOpt{timeout: 10 * time.Second})
}

// ─────────────────────────── 工具 ───────────────────────────

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
