package media

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hoonfeng/paircode/plugins-src/plugins/wechat-bridge/ilink"
)

// TestParseAESKey 双形态解析：base64(hex 字符串) 与 base64(16 字节)。
func TestParseAESKey(t *testing.T) {
	// 形态一：真实抓样——base64("374316ee3157585f0cb33865200bac79")
	k1, err := ParseAESKey("Mzc0MzE2ZWUzMTU3NTg1ZjBjYjMzODY1MjAwYmFjNzk=")
	if err != nil {
		t.Fatalf("hex 形态解析失败: %v", err)
	}
	if hex.EncodeToString(k1) != "374316ee3157585f0cb33865200bac79" {
		t.Fatalf("hex 形态解析结果错误: %x", k1)
	}

	// 形态二：base64(16 字节原始 key)
	raw := []byte("0123456789abcdef")
	k2, err := ParseAESKey(base64.StdEncoding.EncodeToString(raw))
	if err != nil {
		t.Fatalf("raw16 形态解析失败: %v", err)
	}
	if !bytes.Equal(k2, raw) {
		t.Fatalf("raw16 形态解析结果错误: %x", k2)
	}

	// 非法输入
	if _, err := ParseAESKey(""); err == nil {
		t.Fatal("空 key 应报错")
	}
	if _, err := ParseAESKey(base64.StdEncoding.EncodeToString([]byte("too-short"))); err == nil {
		t.Fatal("长度不符应报错")
	}
}

// TestAESECBPaddedSize 密文尺寸公式（对齐官方 aesEcbPaddedSize）。
func TestAESECBPaddedSize(t *testing.T) {
	cases := map[int64]int64{0: 16, 1: 16, 15: 16, 16: 32, 17: 32, 31: 32, 32: 48, 286442: 286448}
	for in, want := range cases {
		if got := AESECBPaddedSize(in); got != want {
			t.Fatalf("AESECBPaddedSize(%d)=%d，期望 %d", in, got, want)
		}
	}
}

// TestEncryptDecryptRoundTrip 加解密往返（含补齐边界）。
func TestEncryptDecryptRoundTrip(t *testing.T) {
	key := []byte("0123456789abcdef")
	for _, n := range []int{0, 1, 15, 16, 17, 4096} {
		plain := bytes.Repeat([]byte("wx"), n)
		cipher, err := EncryptAESEcb(plain, key)
		if err != nil {
			t.Fatalf("n=%d 加密失败: %v", n, err)
		}
		if int64(len(cipher)) != AESECBPaddedSize(int64(len(plain))) {
			t.Fatalf("n=%d 密文尺寸不符: %d", n, len(cipher))
		}
		back, err := DecryptAESEcb(cipher, key)
		if err != nil {
			t.Fatalf("n=%d 解密失败: %v", n, err)
		}
		if !bytes.Equal(back, plain) {
			t.Fatalf("n=%d 往返不一致", n)
		}
	}
	// 非法密文（长度非块对齐）
	if _, err := DecryptAESEcb([]byte("short"), key); err == nil {
		t.Fatal("非法密文应报错")
	}
}

// TestSniffExt 文件头嗅探。
func TestSniffExt(t *testing.T) {
	jpg := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0, 0}
	if ext, kind := SniffExt(jpg); ext != "jpg" || kind != "image" {
		t.Fatalf("jpg 嗅探错误: %s %s", ext, kind)
	}
	png := append([]byte("\x89PNG\r\n\x1a\n"), 0, 0)
	if ext, _ := SniffExt(png); ext != "png" {
		t.Fatalf("png 嗅探错误: %s", ext)
	}
	pdf := []byte("%PDF-1.7 xxx")
	if ext, _ := SniffExt(pdf); ext != "pdf" {
		t.Fatalf("pdf 嗅探错误: %s", ext)
	}
	silk := []byte("\x02#!SILK_V3\x00\x00")
	if ext, kind := SniffExt(silk); ext != "silk" || kind != "voice" {
		t.Fatalf("silk 嗅探错误: %s %s", ext, kind)
	}
	silk0 := []byte("#!SILK_V3\x00")
	if ext, _ := SniffExt(silk0); ext != "silk" {
		t.Fatalf("silk(无前缀头) 嗅探错误: %s", ext)
	}
	mp4 := append([]byte{0, 0, 0, 0x20}, []byte("ftypisom")...)
	if ext, kind := SniffExt(mp4); ext != "mp4" || kind != "video" {
		t.Fatalf("mp4 嗅探错误: %s %s", ext, kind)
	}
	if ext, _ := SniffExt([]byte("random")); ext != "" {
		t.Fatalf("未知数据不应识别: %s", ext)
	}
}

// TestKindFromPath 扩展名 → 发送类型。
func TestKindFromPath(t *testing.T) {
	if k, mt, ok := KindFromPath("a/b/c.PNG"); k != "image" || mt != UploadImage || !ok {
		t.Fatalf("png 类型错误: %s %d %v", k, mt, ok)
	}
	if k, mt, _ := KindFromPath("video.mp4"); k != "video" || mt != UploadVideo {
		t.Fatalf("mp4 类型错误: %s %d", k, mt)
	}
	if k, mt, _ := KindFromPath("doc.pdf"); k != "file" || mt != UploadFile {
		t.Fatalf("pdf 类型错误: %s %d", k, mt)
	}
	if _, _, ok := KindFromPath("noext"); ok {
		t.Fatal("无扩展名应拒绝")
	}
}

// TestBuildItem 发送 item 构造（aes_key = base64(hex 字符串)）。
func TestBuildItem(t *testing.T) {
	up := &Uploaded{
		FileKey:            "aabb",
		DownloadParam:      "param-1",
		AESKeyHex:          "374316ee3157585f0cb33865200bac79",
		FileSize:           12345,
		FileSizeCiphertext: 12352,
	}
	it, err := BuildItem("image", up, "")
	if err != nil {
		t.Fatal(err)
	}
	if it["type"] != 2 {
		t.Fatalf("image type 错误: %v", it["type"])
	}
	img := it["image_item"].(map[string]any)
	m := img["media"].(map[string]any)
	if m["encrypt_query_param"] != "param-1" || m["encrypt_type"] != 1 {
		t.Fatalf("image media 错误: %v", m)
	}
	wantKey := base64.StdEncoding.EncodeToString([]byte("374316ee3157585f0cb33865200bac79"))
	if m["aes_key"] != wantKey {
		t.Fatalf("aes_key 编码错误: %v", m["aes_key"])
	}
	if img["mid_size"] != int64(12352) {
		t.Fatalf("mid_size 错误: %v", img["mid_size"])
	}

	fi, _ := BuildItem("file", up, "报告 v1.pdf")
	fItem := fi["file_item"].(map[string]any)
	if fItem["file_name"] != "报告 v1.pdf" || fItem["len"] != "12345" {
		t.Fatalf("file_item 错误: %v", fItem)
	}

	vi, _ := BuildItem("video", up, "")
	vItem := vi["video_item"].(map[string]any)
	if vi["type"] != 5 || vItem["video_size"] != int64(12352) {
		t.Fatalf("video_item 错误: %v", vItem)
	}

	if _, err := BuildItem("unknown", up, ""); err == nil {
		t.Fatal("未知类型应报错")
	}
}

// TestFetchDecrypt 下载+解密（httptest 模拟 CDN）。
func TestFetchDecrypt(t *testing.T) {
	key := []byte("0123456789abcdef")
	plain := []byte("hello weixin media 你好微信")
	cipher, _ := EncryptAESEcb(plain, key)
	keyB64 := base64.StdEncoding.EncodeToString([]byte(hex.EncodeToString(key)))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(cipher)
	}))
	defer srv.Close()

	got, err := Fetch(srv.URL+"/download?encrypted_query_param=x", "", keyB64, "")
	if err != nil {
		t.Fatalf("Fetch 失败: %v", err)
	}
	if !bytes.Equal(got, plain) {
		t.Fatal("Fetch 解密结果不一致")
	}

	// 明文下载（无 key）
	got2, err := Fetch(srv.URL+"/d", "", "", "")
	if err != nil || !bytes.Equal(got2, cipher) {
		t.Fatalf("明文下载失败: %v", err)
	}

	// full_url 缺省 → 按 cdnBaseURL 拼接
	got3, err := Fetch("", "x", keyB64, srv.URL)
	if err != nil || !bytes.Equal(got3, plain) {
		t.Fatalf("基址拼接下载失败: %v", err)
	}

	// HTTP 404
	if _, err := Fetch(srv.URL+"/404", "", "", ""); err == nil {
		// 我们的 handler 总返回 200，这里构造一个 404 服务
	}
	notFound := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer notFound.Close()
	if _, err := Fetch(notFound.URL, "", "", ""); err == nil {
		t.Fatal("404 应报错")
	}
}

// TestInboundImage 收方向：图片下载解密落盘 + 描述。
func TestInboundImage(t *testing.T) {
	key := []byte("abcdefghijklmnop")
	plain := append([]byte{0xFF, 0xD8, 0xFF, 0xE0}, bytes.Repeat([]byte{1}, 64)...)
	cipher, _ := EncryptAESEcb(plain, key)
	keyB64 := base64.StdEncoding.EncodeToString([]byte(hex.EncodeToString(key)))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(cipher)
	}))
	defer srv.Close()

	dir := t.TempDir()
	item := &ilink.MsgItem{
		Type: 2,
		ImageItem: &ilink.ImageItem{
			Media:   &ilink.CDNMedia{FullURL: srv.URL + "/img", AESKey: keyB64},
			MidSize: int64(len(cipher)),
		},
	}
	path, note, err := Inbound(dir, item, "")
	if err != nil {
		t.Fatalf("Inbound 失败: %v", err)
	}
	if !strings.HasSuffix(path, ".jpg") {
		t.Fatalf("扩展名推断错误: %s", path)
	}
	saved, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(saved, plain) {
		t.Fatalf("落盘内容不一致: %v", err)
	}
	if !strings.Contains(note, "read_image") {
		t.Fatalf("描述应含 read_image 指引: %s", note)
	}
	if !strings.Contains(note, fmt.Sprintf("%d 字节", len(plain))) {
		t.Fatalf("描述应含字节数: %s", note)
	}

	// 校验不一致提示
	item.ImageItem.MidSize = 999
	_, note2, _ := Inbound(dir, item, "")
	if !strings.Contains(note2, "不符") {
		t.Fatalf("尺寸不符应有警告: %s", note2)
	}

	// image_item.aeskey（hex 形态）优先
	item2 := &ilink.MsgItem{
		Type: 2,
		ImageItem: &ilink.ImageItem{
			Media:   &ilink.CDNMedia{FullURL: srv.URL + "/img"},
			AESKey:  hex.EncodeToString(key),
			MidSize: int64(len(cipher)),
		},
	}
	_, _, err = Inbound(dir, item2, "")
	if err != nil {
		t.Fatalf("hex aeskey 形态失败: %v", err)
	}
}

// TestUploadFlow 全流程：getuploadurl → 加密上传 → x-encrypted-param（httptest 双重模拟）。
func TestUploadFlow(t *testing.T) {
	fileContent := bytes.Repeat([]byte("wx-bridge-media-test-数据"), 100)
	tmp := filepath.Join(t.TempDir(), "pic.jpg")
	if err := os.WriteFile(tmp, fileContent, 0o644); err != nil {
		t.Fatal(err)
	}

	var srv *httptest.Server
	var aesKeySeen, filekeySeen, md5Seen string
	var rawsizeSeen, filesizeSeen float64
	uploadHit := 0

	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ilink/bot/getuploadurl":
			var req map[string]any
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Errorf("getuploadurl 请求解析失败: %v", err)
			}
			aesKeySeen, _ = req["aeskey"].(string)
			filekeySeen, _ = req["filekey"].(string)
			md5Seen, _ = req["rawfilemd5"].(string)
			rawsizeSeen, _ = req["rawsize"].(float64)
			filesizeSeen, _ = req["filesize"].(float64)
			if req["media_type"] != float64(UploadImage) || req["to_user_id"] != "user-1" || req["no_need_thumb"] != true {
				t.Errorf("getuploadurl 字段错误: %v", req)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ret": 0, "upload_full_url": srv.URL + "/cdn/upload"})
		case "/cdn/upload":
			uploadHit++
			body, _ := io.ReadAll(r.Body)
			if r.Header.Get("Content-Type") != "application/octet-stream" {
				t.Errorf("上传 Content-Type 错误: %s", r.Header.Get("Content-Type"))
			}
			key, err := hex.DecodeString(aesKeySeen)
			if err != nil {
				t.Errorf("aeskey 非法: %v", err)
			}
			plain, err := DecryptAESEcb(body, key)
			if err != nil {
				t.Errorf("服务器解密上传内容失败: %v", err)
			}
			if !bytes.Equal(plain, fileContent) {
				t.Error("上传内容与源文件不一致")
			}
			w.Header().Set("x-encrypted-param", "download-param-xyz")
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	api := ilink.NewClient(srv.URL, "test-token")
	up, err := Upload(api, tmp, "user-1", UploadImage, "")
	if err != nil {
		t.Fatalf("Upload 失败: %v", err)
	}
	if uploadHit != 1 {
		t.Fatalf("上传应命中 1 次，实际 %d", uploadHit)
	}
	if up.DownloadParam != "download-param-xyz" {
		t.Fatalf("downloadParam 错误: %s", up.DownloadParam)
	}
	if up.FileSize != int64(len(fileContent)) || up.FileSizeCiphertext != AESECBPaddedSize(int64(len(fileContent))) {
		t.Fatalf("尺寸字段错误: %+v", up)
	}
	if len(up.AESKeyHex) != 32 || len(filekeySeen) != 32 {
		t.Fatalf("key/filekey 应为 32 字符 hex: %s %s", up.AESKeyHex, filekeySeen)
	}
	if rawsizeSeen != float64(len(fileContent)) {
		t.Fatalf("rawsize 错误: %v", rawsizeSeen)
	}
	if filesizeSeen != float64(AESECBPaddedSize(int64(len(fileContent)))) {
		t.Fatalf("filesize 错误: %v", filesizeSeen)
	}
	if len(md5Seen) != 32 {
		t.Fatalf("rawfilemd5 应为 32 字符: %s", md5Seen)
	}

	// 上传失败（4xx）不重试
	failSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "getuploadurl") {
			_ = json.NewEncoder(w).Encode(map[string]any{"ret": 0, "upload_param": "p1"})
			return
		}
		w.Header().Set("x-error-message", "bad key")
		w.WriteHeader(400)
	}))
	defer failSrv.Close()
	api2 := ilink.NewClient(failSrv.URL, "t")
	if _, err := Upload(api2, tmp, "user-1", UploadImage, failSrv.URL); err == nil {
		t.Fatal("4xx 应直接失败")
	}
}

// TestInboundVoice 语音：落盘 + 转写文本进入描述。
func TestInboundVoice(t *testing.T) {
	key := []byte("abcdefghijklmnop")
	plain := []byte("\x02#!SILK_V3" + strings.Repeat("x", 100))
	cipher, _ := EncryptAESEcb(plain, key)
	keyB64 := base64.StdEncoding.EncodeToString([]byte(hex.EncodeToString(key)))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(cipher)
	}))
	defer srv.Close()

	dir := t.TempDir()
	item := &ilink.MsgItem{
		Type: 3,
		VoiceItem: &ilink.VoiceItem{
			Media:    &ilink.CDNMedia{FullURL: srv.URL + "/v", AESKey: keyB64},
			Playtime: 4215,
			Text:     "测试好了吗？给我看看",
		},
	}
	path, note, err := Inbound(dir, item, "")
	if err != nil {
		t.Fatalf("Inbound 失败: %v", err)
	}
	if !strings.HasSuffix(path, ".silk") {
		t.Fatalf("扩展名错误: %s", path)
	}
	if !strings.Contains(note, "测试好了吗？给我看看") || !strings.Contains(note, "4.2 秒") {
		t.Fatalf("语音描述错误: %s", note)
	}
}
