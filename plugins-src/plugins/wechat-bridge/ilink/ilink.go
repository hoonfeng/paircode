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
}

// NewClient 创建协议客户端。
func NewClient(baseURL, token string) *Client {
	return &Client{BaseURL: strings.TrimRight(baseURL, "/"), Token: token}
}

type postOpt struct {
	timeout         time.Duration
	retries         int
	tolerateTimeout bool
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
func (c *Client) SendText(toUserID, text, contextToken string) (*SendTextResp, error) {
	return post[SendTextResp](c, "ilink/bot/sendmessage", map[string]any{
		"msg": map[string]any{
			"from_user_id":  "",
			"to_user_id":    toUserID,
			"client_id":     randomHex(8),
			"message_type":  2,
			"message_state": 2,
			"item_list":     []map[string]any{{"type": 1, "text_item": map[string]any{"text": text}}},
			"context_token": contextToken,
		},
	}, postOpt{retries: 2})
}

// SendItem 发送单条结构化消息条目（媒体等；item = item_list 的单个元素）。
// 与 SendText 同构：message_type=2(BOT)/message_state=2(FINISH)，必须带对应用户的 context_token。
func (c *Client) SendItem(toUserID string, item map[string]any, contextToken string) (*SendTextResp, error) {
	return post[SendTextResp](c, "ilink/bot/sendmessage", map[string]any{
		"msg": map[string]any{
			"from_user_id":  "",
			"to_user_id":    toUserID,
			"client_id":     randomHex(8),
			"message_type":  2,
			"message_state": 2,
			"item_list":     []map[string]any{item},
			"context_token": contextToken,
		},
	}, postOpt{retries: 2})
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
