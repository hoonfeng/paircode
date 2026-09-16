// Package media 微信媒体收发：CDN 下载/解密（收）与加密/上传（发）。
//
// 协议校准（2026-09-16 真机抓样 + 官方 @tencent-weixin/openclaw-weixin v2.4.8 对照）：
//   - 下载：优先 media.full_url；否则 <CDNBaseURL>/download?encrypted_query_param=…；
//   - 加解密：AES-128-ECB + PKCS7；密钥两种形态——
//     base64(16 字节原始 key) 或 base64(32 字符 hex 字符串)（在线都见过，都要兼容）；
//   - 上传：getuploadurl 预签名 → POST 密文（octet-stream）→ 响应头 x-encrypted-param；
//   - item 尺寸：image.mid_size / video.video_size = 密文尺寸；file.len = 明文尺寸；
//     media.aes_key = base64(32 字符 hex)（即对 hex 字符串再 base64）。
package media

import (
	"bytes"
	"crypto/aes"
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/hoonfeng/paircode/plugins-src/plugins/wechat-bridge/ilink"
)

// 常量。
const (
	// DefaultCDNBaseURL 微信 CDN 基础地址（上传/下载 URL 拼接 fallback）。
	DefaultCDNBaseURL = "https://novac2c.cdn.weixin.qq.com/c2c"
	// MaxMediaBytes 单文件大小上限（与官方一致，100MB）。
	MaxMediaBytes = 100 << 20
	// uploadMaxRetries CDN 上传失败重试次数（仅服务端错误/网络错误重试；4xx 直接失败）。
	uploadMaxRetries = 3
)

// getuploadurl media_type 取值（官方 proto UploadMediaType）。
const (
	UploadImage = 1
	UploadVideo = 2
	UploadFile  = 3
	UploadVoice = 4
)

// httpc 共享下载/上传客户端（复用连接；单请求超时 120s）。
var httpc = &http.Client{Timeout: 120 * time.Second}

// ─────────────────────────── 加解密 ───────────────────────────

// ParseAESKey 解析 aes_key（base64 字段）→ 16 字节 AES key。
// 兼容两种在线形态：base64(16 字节) 与 base64(32 字符 hex 字符串)。
func ParseAESKey(aesKeyBase64 string) ([]byte, error) {
	s := strings.TrimSpace(aesKeyBase64)
	if s == "" {
		return nil, fmt.Errorf("aes_key 为空")
	}
	decoded, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		if d2, e2 := base64.URLEncoding.DecodeString(s); e2 == nil {
			decoded = d2
		} else {
			return nil, fmt.Errorf("aes_key base64 解码失败: %v", err)
		}
	}
	if len(decoded) == 16 {
		return decoded, nil
	}
	if len(decoded) == 32 && isHexString(string(decoded)) {
		raw, err := hex.DecodeString(string(decoded))
		if err != nil {
			return nil, fmt.Errorf("aes_key hex 解码失败: %v", err)
		}
		return raw, nil
	}
	return nil, fmt.Errorf("aes_key 形态不符（base64 解码 %d 字节，期望 16 或 32 字符 hex）", len(decoded))
}

func isHexString(s string) bool {
	for _, c := range s {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F') {
			return false
		}
	}
	return len(s) > 0
}

// DecryptAESEcb AES-128-ECB 解密（剥 PKCS7 填充）。
func DecryptAESEcb(ciphertext, key []byte) ([]byte, error) {
	blk, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("AES key 非法: %w", err)
	}
	if len(ciphertext) == 0 || len(ciphertext)%blk.BlockSize() != 0 {
		return nil, fmt.Errorf("密文长度非法: %d", len(ciphertext))
	}
	out := make([]byte, len(ciphertext))
	for i := 0; i < len(ciphertext); i += blk.BlockSize() {
		blk.Decrypt(out[i:i+blk.BlockSize()], ciphertext[i:i+blk.BlockSize()])
	}
	return unpadPKCS7(out)
}

// EncryptAESEcb AES-128-ECB 加密（PKCS7 填充）。
func EncryptAESEcb(plain, key []byte) ([]byte, error) {
	blk, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("AES key 非法: %w", err)
	}
	pt := padPKCS7(plain, blk.BlockSize())
	out := make([]byte, len(pt))
	for i := 0; i < len(pt); i += blk.BlockSize() {
		blk.Encrypt(out[i:i+blk.BlockSize()], pt[i:i+blk.BlockSize()])
	}
	return out, nil
}

// AESECBPaddedSize 计算 AES-128-ECB 密文尺寸（PKCS7 填充到 16 字节边界）。
func AESECBPaddedSize(n int64) int64 {
	return int64((n+16)/16) * 16
}

func padPKCS7(b []byte, blockSize int) []byte {
	n := blockSize - len(b)%blockSize
	out := make([]byte, len(b)+n)
	copy(out, b)
	for i := len(b); i < len(out); i++ {
		out[i] = byte(n)
	}
	return out
}

func unpadPKCS7(b []byte) ([]byte, error) {
	if len(b) == 0 {
		return nil, fmt.Errorf("明文为空")
	}
	n := int(b[len(b)-1])
	if n == 0 || n > 16 || n > len(b) {
		return nil, fmt.Errorf("PKCS7 填充非法（%d）", n)
	}
	for _, c := range b[len(b)-n:] {
		if int(c) != n {
			return nil, fmt.Errorf("PKCS7 填充字节不一致")
		}
	}
	return b[:len(b)-n], nil
}

// ─────────────────────────── 下载（收方向）───────────────────────────

// Fetch 下载（并按需解密）CDN 媒体。
// aesKeyBase64 为空时按明文下载（协议允许无加密的图）；非空时强制解密。
func Fetch(fullURL, encryptQueryParam, aesKeyBase64, cdnBaseURL string) ([]byte, error) {
	u := resolveDownloadURL(fullURL, encryptQueryParam, cdnBaseURL)
	data, err := httpGet(u)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(aesKeyBase64) == "" {
		return data, nil
	}
	key, err := ParseAESKey(aesKeyBase64)
	if err != nil {
		return nil, err
	}
	return DecryptAESEcb(data, key)
}

// resolveDownloadURL 下载 URL：full_url 优先；否则按 CDN 基址拼接。
func resolveDownloadURL(fullURL, encryptQueryParam, cdnBaseURL string) string {
	if f := strings.TrimSpace(fullURL); f != "" {
		return f
	}
	base := strings.TrimRight(strings.TrimSpace(cdnBaseURL), "/")
	if base == "" {
		base = DefaultCDNBaseURL
	}
	return base + "/download?encrypted_query_param=" + url.QueryEscape(encryptQueryParam)
}

func httpGet(u string) ([]byte, error) {
	resp, err := httpc.Get(u)
	if err != nil {
		return nil, fmt.Errorf("下载失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("下载 HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, MaxMediaBytes+1))
	if err != nil {
		return nil, fmt.Errorf("下载读取失败: %w", err)
	}
	if len(data) > MaxMediaBytes {
		return nil, fmt.Errorf("媒体超过 %dMB 上限", MaxMediaBytes>>20)
	}
	return data, nil
}

// ─────────────────────────── 收方向：落盘与描述 ───────────────────────────

// Inbound 收方向：下载/解密一个消息条目并保存到 dir；返回 (保存路径, 投喂描述, 错误)。
// note 无论成败都含可读说明（供投喂与日志）；成功时 err 为 nil。
func Inbound(dir string, item *ilink.MsgItem, cdnBaseURL string) (string, string, error) {
	if item == nil {
		return "", "（空条目）", fmt.Errorf("空消息条目")
	}
	switch item.Type {
	case 2:
		return inboundImage(dir, item, cdnBaseURL)
	case 3:
		return inboundVoice(dir, item, cdnBaseURL)
	case 4:
		return inboundFile(dir, item, cdnBaseURL)
	case 5:
		return inboundVideo(dir, item, cdnBaseURL)
	}
	return "", fmt.Sprintf("（未知媒体类型 %d）", item.Type), fmt.Errorf("未知媒体类型 %d", item.Type)
}

func inboundImage(dir string, item *ilink.MsgItem, cdnBaseURL string) (string, string, error) {
	img := item.ImageItem
	if img == nil || img.Media == nil {
		return "", "（图片条目缺 media）", fmt.Errorf("图片条目缺 media")
	}
	if img.Media.FullURL == "" && img.Media.EncryptQueryParam == "" {
		return "", "（图片条目缺下载地址）", fmt.Errorf("图片条目缺下载地址")
	}
	key := img.Media.AESKey
	if strings.TrimSpace(img.AESKey) != "" {
		// image_item.aeskey 为 hex 原始 key → 包装成 base64（ParseAESKey 双形态兼容）
		key = base64.StdEncoding.EncodeToString([]byte(img.AESKey))
	}
	data, err := Fetch(img.Media.FullURL, img.Media.EncryptQueryParam, key, cdnBaseURL)
	if err != nil {
		return "", "（图片接收失败：" + err.Error() + "）", err
	}
	ext, _ := SniffExt(data)
	if ext == "" {
		ext = "jpg"
	}
	name := fmt.Sprintf("%s-img-%s.%s", tsPrefix(), randSuffix(), ext)
	path, err := saveInbound(dir, name, data)
	if err != nil {
		return "", "（图片保存失败：" + err.Error() + "）", err
	}
	note := fmt.Sprintf("[图片] %s（%d 字节）—— 请用 read_image 工具查看图片内容", path, len(data))
	if img.MidSize > 0 && img.MidSize != int64(len(data)) {
		note += fmt.Sprintf("（注意：声明 mid_size=%d 与实际不符）", img.MidSize)
	}
	return path, note, nil
}

func inboundVoice(dir string, item *ilink.MsgItem, cdnBaseURL string) (string, string, error) {
	v := item.VoiceItem
	if v == nil || v.Media == nil {
		return "", "（语音条目缺 media）", fmt.Errorf("语音条目缺 media")
	}
	if strings.TrimSpace(v.Media.AESKey) == "" {
		return "", "（语音条目缺 aes_key）", fmt.Errorf("语音条目缺 aes_key")
	}
	data, err := Fetch(v.Media.FullURL, v.Media.EncryptQueryParam, v.Media.AESKey, cdnBaseURL)
	if err != nil {
		return "", "（语音接收失败：" + err.Error() + "）", err
	}
	name := fmt.Sprintf("%s-voice-%s.silk", tsPrefix(), randSuffix())
	path, err := saveInbound(dir, name, data)
	if err != nil {
		return "", "（语音保存失败：" + err.Error() + "）", err
	}
	note := fmt.Sprintf("[语音 %.1f 秒]", float64(v.Playtime)/1000)
	if strings.TrimSpace(v.Text) != "" {
		note += " 转写：" + v.Text
	} else {
		note += "（微信未附转写文本）"
	}
	note += fmt.Sprintf("（原始音频：%s）", path)
	return path, note, nil
}

func inboundFile(dir string, item *ilink.MsgItem, cdnBaseURL string) (string, string, error) {
	f := item.FileItem
	if f == nil || f.Media == nil {
		return "", "（文件条目缺 media）", fmt.Errorf("文件条目缺 media")
	}
	if strings.TrimSpace(f.Media.AESKey) == "" {
		return "", "（文件条目缺 aes_key）", fmt.Errorf("文件条目缺 aes_key")
	}
	data, err := Fetch(f.Media.FullURL, f.Media.EncryptQueryParam, f.Media.AESKey, cdnBaseURL)
	if err != nil {
		return "", "（文件接收失败：" + err.Error() + "）", err
	}
	orig := sanitizeName(f.FileName)
	if orig == "" {
		orig = "file.bin"
	}
	name := fmt.Sprintf("%s-%s", tsPrefix(), orig)
	path, err := saveInbound(dir, name, data)
	if err != nil {
		return "", "（文件保存失败：" + err.Error() + "）", err
	}
	note := fmt.Sprintf("[文件] %s（%d 字节）→ %s", orDash(f.FileName), len(data), path)
	note += verifyNotes(string(f.Len), f.MD5, data)
	return path, note, nil
}

func inboundVideo(dir string, item *ilink.MsgItem, cdnBaseURL string) (string, string, error) {
	v := item.VideoItem
	if v == nil || v.Media == nil {
		return "", "（视频条目缺 media）", fmt.Errorf("视频条目缺 media")
	}
	if strings.TrimSpace(v.Media.AESKey) == "" {
		return "", "（视频条目缺 aes_key）", fmt.Errorf("视频条目缺 aes_key")
	}
	data, err := Fetch(v.Media.FullURL, v.Media.EncryptQueryParam, v.Media.AESKey, cdnBaseURL)
	if err != nil {
		return "", "（视频接收失败：" + err.Error() + "）", err
	}
	name := fmt.Sprintf("%s-video-%s.mp4", tsPrefix(), randSuffix())
	path, err := saveInbound(dir, name, data)
	if err != nil {
		return "", "（视频保存失败：" + err.Error() + "）", err
	}
	note := fmt.Sprintf("[视频 %d 秒] %s（%d 字节）", v.PlayLength, path, len(data))
	if v.VideoSize > 0 && v.VideoSize != int64(len(data)) {
		note += fmt.Sprintf("（注意：声明 video_size=%d 与实际不符）", v.VideoSize)
	}
	if v.VideoMD5 != "" {
		sum := md5.Sum(data)
		if hex.EncodeToString(sum[:]) != v.VideoMD5 {
			note += "（警告：MD5 校验不一致）"
		}
	}
	return path, note, nil
}

// verifyNotes 文件长度/MD5 校验附注（协议声明 vs 实际）。
func verifyNotes(lenStr, md5Str string, data []byte) string {
	note := ""
	if s := strings.TrimSpace(lenStr); s != "" {
		if want, err := strconv.ParseInt(s, 10, 64); err == nil && want != int64(len(data)) {
			note += fmt.Sprintf("（警告：声明 %d 字节与实际不符）", want)
		}
	}
	if strings.TrimSpace(md5Str) != "" {
		sum := md5.Sum(data)
		if hex.EncodeToString(sum[:]) != strings.ToLower(strings.TrimSpace(md5Str)) {
			note += "（警告：MD5 校验不一致）"
		}
	}
	return note
}

func saveInbound(dir, name string, data []byte) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// ─────────────────────────── 上传（发方向）───────────────────────────

// Uploaded 上传结果（构造发送 item 所需）。
type Uploaded struct {
	FileKey            string // 16 字节随机（hex 32 字符）
	DownloadParam      string // → media.encrypt_query_param
	AESKeyHex          string // 32 字符 hex（发给对端时再 base64）
	FileSize           int64  // 明文字节数
	FileSizeCiphertext int64  // 密文字节数（PKCS7 后）
}

// Upload 读文件 → getuploadurl 预签名 → CDN 加密上传 → 返回上传信息。
// mediaType 取 UploadImage/UploadVideo/UploadFile。
func Upload(api *ilink.Client, filePath, toUserID string, mediaType int, cdnBaseURL string) (*Uploaded, error) {
	plain, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("读取文件失败: %w", err)
	}
	if len(plain) == 0 {
		return nil, fmt.Errorf("文件为空: %s", filePath)
	}
	if len(plain) > MaxMediaBytes {
		return nil, fmt.Errorf("文件超过 %dMB 上限", MaxMediaBytes>>20)
	}

	rawsize := int64(len(plain))
	sum := md5.Sum(plain)
	rawmd5 := hex.EncodeToString(sum[:])
	filesize := AESECBPaddedSize(rawsize)
	filekey := randHex(16)
	aesRaw := make([]byte, 16)
	if _, err := rand.Read(aesRaw); err != nil {
		return nil, fmt.Errorf("随机密钥生成失败: %w", err)
	}
	aesKeyHex := hex.EncodeToString(aesRaw)

	resp, err := api.GetUploadUrl(map[string]any{
		"filekey":       filekey,
		"media_type":    mediaType,
		"to_user_id":    toUserID,
		"rawsize":       rawsize,
		"rawfilemd5":    rawmd5,
		"filesize":      filesize,
		"no_need_thumb": true,
		"aeskey":        aesKeyHex,
	})
	if err != nil {
		return nil, fmt.Errorf("getuploadurl 请求失败: %w", err)
	}
	if resp.Ret != 0 || resp.Errcode != 0 {
		return nil, fmt.Errorf("getuploadurl 返回异常 ret=%d errcode=%d errmsg=%s", resp.Ret, resp.Errcode, resp.Errmsg)
	}
	if strings.TrimSpace(resp.UploadFullURL) == "" && strings.TrimSpace(resp.UploadParam) == "" {
		return nil, fmt.Errorf("getuploadurl 未返回上传地址")
	}

	cipher, err := EncryptAESEcb(plain, aesRaw)
	if err != nil {
		return nil, err
	}
	param, err := uploadCipher(cipher, resp.UploadFullURL, resp.UploadParam, filekey, cdnBaseURL)
	if err != nil {
		return nil, err
	}
	return &Uploaded{
		FileKey:            filekey,
		DownloadParam:      param,
		AESKeyHex:          aesKeyHex,
		FileSize:           rawsize,
		FileSizeCiphertext: filesize,
	}, nil
}

// uploadCipher POST 密文到 CDN；返回响应头 x-encrypted-param（下载参数）。
// upload_full_url 优先；否则按 CDN 基址拼接 upload 地址。
func uploadCipher(cipher []byte, fullURL, uploadParam, filekey, cdnBaseURL string) (string, error) {
	u := strings.TrimSpace(fullURL)
	if u == "" {
		base := strings.TrimRight(strings.TrimSpace(cdnBaseURL), "/")
		if base == "" {
			base = DefaultCDNBaseURL
		}
		u = base + "/upload?encrypted_query_param=" + url.QueryEscape(uploadParam) +
			"&filekey=" + url.QueryEscape(filekey)
	}

	var lastErr error
	for attempt := 1; attempt <= uploadMaxRetries; attempt++ {
		req, err := http.NewRequest(http.MethodPost, u, bytes.NewReader(cipher))
		if err != nil {
			return "", fmt.Errorf("上传请求构造失败: %w", err)
		}
		req.Header.Set("Content-Type", "application/octet-stream")
		resp, err := httpc.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("上传失败: %w", err)
			continue
		}
		status := resp.StatusCode
		errMsg := resp.Header.Get("x-error-message")
		if status >= 400 && status < 500 {
			if errMsg == "" {
				b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
				errMsg = strings.TrimSpace(string(b))
			}
			resp.Body.Close()
			return "", fmt.Errorf("CDN 上传客户端错误 %d: %s", status, errMsg)
		}
		if status != http.StatusOK {
			io.Copy(io.Discard, io.LimitReader(resp.Body, 2048))
			resp.Body.Close()
			lastErr = fmt.Errorf("CDN 上传服务端错误 %d: %s", status, errMsg)
			continue
		}
		param := resp.Header.Get("x-encrypted-param")
		io.Copy(io.Discard, io.LimitReader(resp.Body, 2048))
		resp.Body.Close()
		if param == "" {
			lastErr = fmt.Errorf("CDN 上传响应缺 x-encrypted-param 头")
			continue
		}
		return param, nil
	}
	return "", lastErr
}

// ─────────────────────────── 发送 item 构造 ───────────────────────────

// KindFromPath 按扩展名判定发送媒体类型：image / video / file（+ getuploadurl media_type）。
func KindFromPath(p string) (kind string, mediaType int, ok bool) {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(p), "."))
	switch ext {
	case "jpg", "jpeg", "png", "gif", "webp", "bmp":
		return "image", UploadImage, true
	case "mp4", "mov", "m4v", "avi", "mkv", "webm":
		return "video", UploadVideo, true
	case "":
		return "", 0, false
	default:
		return "file", UploadFile, true
	}
}

// BuildItem 构造发送 item（kind: image/video/file）；fileName 仅 file 类型使用。
func BuildItem(kind string, up *Uploaded, fileName string) (map[string]any, error) {
	mediaRef := map[string]any{
		"encrypt_query_param": up.DownloadParam,
		"aes_key":             base64.StdEncoding.EncodeToString([]byte(up.AESKeyHex)),
		"encrypt_type":        1,
	}
	switch kind {
	case "image":
		return map[string]any{
			"type": 2,
			"image_item": map[string]any{
				"media":    mediaRef,
				"mid_size": up.FileSizeCiphertext,
			},
		}, nil
	case "video":
		return map[string]any{
			"type": 5,
			"video_item": map[string]any{
				"media":      mediaRef,
				"video_size": up.FileSizeCiphertext,
			},
		}, nil
	case "file":
		name := sanitizeName(fileName)
		if name == "" {
			name = "file.bin"
		}
		return map[string]any{
			"type": 4,
			"file_item": map[string]any{
				"media":     mediaRef,
				"file_name": name,
				"len":       strconv.FormatInt(up.FileSize, 10),
			},
		}, nil
	}
	return nil, fmt.Errorf("不支持的媒体类型: %s", kind)
}

// ─────────────────────────── 嗅探与工具 ───────────────────────────

// SniffExt 按文件头（magic）推测扩展名与类别（无法识别时返回空串）。
func SniffExt(b []byte) (ext, kind string) {
	switch {
	case len(b) >= 3 && b[0] == 0xFF && b[1] == 0xD8 && b[2] == 0xFF:
		return "jpg", "image"
	case len(b) >= 8 && bytes.HasPrefix(b, []byte("\x89PNG\r\n\x1a\n")):
		return "png", "image"
	case len(b) >= 6 && (bytes.HasPrefix(b, []byte("GIF87a")) || bytes.HasPrefix(b, []byte("GIF89a"))):
		return "gif", "image"
	case len(b) >= 12 && bytes.HasPrefix(b, []byte("RIFF")) && bytes.Equal(b[8:12], []byte("WEBP")):
		return "webp", "image"
	case len(b) >= 4 && bytes.HasPrefix(b, []byte("%PDF")):
		return "pdf", "file"
	case len(b) >= 10 && (bytes.HasPrefix(b, []byte("#!SILK_V3")) || bytes.HasPrefix(b[1:], []byte("#!SILK_V3"))):
		return "silk", "voice"
	case len(b) >= 5 && bytes.HasPrefix(b, []byte("#!AMR")):
		return "amr", "voice"
	case len(b) >= 8 && bytes.Equal(b[4:8], []byte("ftyp")):
		return "mp4", "video"
	case len(b) >= 4 && bytes.HasPrefix(b, []byte("PK\x03\x04")):
		return "zip", "file"
	case len(b) >= 8 && bytes.HasPrefix(b, []byte("\x1f\x8b")):
		return "gz", "file"
	}
	return "", ""
}

// sanitizeName 过滤文件名非法字符并防路径穿越（Windows 兼容）。
func sanitizeName(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "..", "_")
	return strings.Map(func(ch rune) rune {
		switch ch {
		case '\\', '/', ':', '*', '?', '"', '<', '>', '|':
			return '_'
		}
		return ch
	}, s)
}

func tsPrefix() string {
	return time.Now().Format("20060102-150405")
}

func randSuffix() string {
	b := make([]byte, 2)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func randHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func orDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "-"
	}
	return s
}
