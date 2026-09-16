// dump.go — 调试命令（--dump xxx）：执行后退出，不进入常驻模式。
//
// 用途：环境自检（selftest）与二维码渲染冒烟（qr）——桥开发/部署的快速验证入口。
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/hoonfeng/paircode/plugins-src/plugins/wechat-bridge/ilink"
	"github.com/hoonfeng/paircode/plugins-src/plugins/wechat-bridge/localhttp"
	"github.com/hoonfeng/paircode/plugins-src/plugins/wechat-bridge/login"
	"github.com/hoonfeng/paircode/plugins-src/plugins/wechat-bridge/qr"
	"github.com/hoonfeng/paircode/plugins-src/plugins/wechat-bridge/store"
	"github.com/hoonfeng/paircode/plugins-src/plugins/wechat-bridge/wsclient"
)

// runDump 执行调试命令，返回进程退出码。
func runDump(cmd string) int {
	switch cmd {
	case "selftest":
		return dumpSelftest()
	case "qr":
		return dumpQr()
	case "qr-fetch":
		return dumpQrFetch()
	case "login":
		return dumpLogin()
	case "ws-probe":
		return dumpWsProbe()
	default:
		fmt.Fprintf(os.Stderr, "未知 dump 命令: %s（可用: selftest|qr|qr-fetch|login|ws-probe）\n", cmd)
		return 2
	}
}

// dumpWsProbe 连接 PairCode /ws 事件流做连通冒烟：握手 + 收事件 N 秒 + 输出统计。
// 用法：--paircode http://127.0.0.1:9090 --dump ws-probe
func dumpWsProbe() int {
	base := *flagPaircode
	if base == "" {
		base = "http://127.0.0.1:9090"
	}
	wsURL := strings.TrimSuffix(base, "/") + "/ws"
	if strings.HasPrefix(wsURL, "http://") {
		wsURL = "ws://" + strings.TrimPrefix(wsURL, "http://")
	} else if strings.HasPrefix(wsURL, "https://") {
		fmt.Fprintln(os.Stderr, "wss（TLS）暂不支持")
		return 1
	}

	var mu sync.Mutex
	typeCount := map[string]int{}
	samples := []string{}
	msgCount := 0
	closed := make(chan struct{})

	cli, err := wsclient.Dial(wsURL, 10*time.Second, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "WS 连接失败 %s: %v\n", wsURL, err)
		return 1
	}
	cli.OnMessage = func(data []byte, isText bool) {
		if !isText {
			return
		}
		mu.Lock()
		defer mu.Unlock()
		msgCount++
		var m map[string]any
		if json.Unmarshal(data, &m) == nil {
			if t, ok := m["type"].(string); ok {
				typeCount[t]++
			} else {
				typeCount["(no type)"]++
			}
		} else {
			typeCount["(解析失败)"]++
		}
		if len(samples) < 3 {
			s := string(data)
			if len(s) > 160 {
				s = s[:160] + "…"
			}
			samples = append(samples, s)
		}
	}
	cli.OnClose = func() { close(closed) }

	fmt.Printf("已连接 %s，收集事件 8 秒…\n", wsURL)

	// 4 秒时发一次文本 ping（验证发送方向：客户端掩码帧 + 服务端兼容）
	go func() {
		time.Sleep(4 * time.Second)
		_ = cli.SendText(`{"type":"ping"}`)
	}()

	select {
	case <-closed:
		fmt.Println("（连接被对端关闭）")
	case <-time.After(8 * time.Second):
	}
	_ = cli.Close()
	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	data, _ := json.Marshal(map[string]any{
		"evt":       "ws-probe",
		"url":       wsURL,
		"msgCount":  msgCount,
		"typeCount": typeCount,
		"samples":   samples,
	})
	fmt.Println(string(data))
	if msgCount == 0 {
		return 1
	}
	return 0
}

// dumpLogin 交互式扫码登录：终端实时渲染二维码（内存渲染，不落盘），
// 成功后凭据保存到 <data-dir>/debug-creds.json（调试存档，不写入正式账号结构）。
// 需要验证码时直接在终端输入数字回车（stdin 交互）。
func dumpLogin() int {
	dir := *flagDataDir
	if dir == "" {
		dir = "."
	}
	fmt.Println("启动扫码登录…（手机微信 → 我 → 设置 → 插件 → ClawBot 扫码）")

	var mu sync.Mutex
	printedQR := ""
	lastStatus := ""
	sess := login.NewSession(ilink.DefaultBaseURL, nil, login.Options{
		OnStateChange: func(st login.State) {
			mu.Lock()
			defer mu.Unlock()
			if st.QRContent != "" && st.QRContent != printedQR {
				printedQR = st.QRContent
				if term, err := qr.TerminalText(st.QRContent); err == nil {
					fmt.Printf("\n%s\n", term)
				}
				if st.RefreshCount > 0 {
					fmt.Printf("（二维码已刷新 %d 次）\n", st.RefreshCount)
				}
			}
			line := st.Status
			if st.Err != "" {
				line += " — " + st.Err
			}
			if line != lastStatus {
				lastStatus = line
				fmt.Printf("[状态] %s\n", line)
			}
		},
	})
	sess.Start()

	// 验证码交互（如需要；直接输入数字回车）
	go func() {
		sc := bufio.NewScanner(os.Stdin)
		for sc.Scan() {
			code := strings.TrimSpace(sc.Text())
			if code == "" {
				continue
			}
			if sess.SubmitVerifyCode(code) {
				fmt.Println("已提交验证码，继续轮询…")
			} else {
				fmt.Println("（当前无需验证码输入，已忽略）")
			}
		}
	}()

	creds, err := sess.Wait()
	if err != nil {
		fmt.Fprintf(os.Stderr, "登录失败: %v\n", err)
		return 1
	}
	out := filepath.Join(dir, "debug-creds.json")
	if err := store.WriteJSONAtomic(out, creds); err != nil {
		fmt.Fprintf(os.Stderr, "保存凭据失败: %v\n", err)
		return 1
	}
	fmt.Printf("\n✅ 登录成功：bot_id=%s base_url=%s\n凭据已保存: %s\n", creds.BotID, creds.BaseURL, out)
	return 0
}

// dumpQrFetch 真实请求微信服务器获取登录二维码（协议连通性验证）。
func dumpQrFetch() int {
	resp, err := ilink.FetchQrCode(ilink.DefaultBaseURL, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "获取二维码失败: %v\n", err)
		return 1
	}
	if resp.Qrcode == "" {
		fmt.Fprintf(os.Stderr, "服务器未返回二维码: %+v\n", resp)
		return 1
	}
	img := resp.QrcodeImgContent
	if len(img) > 40 {
		img = img[:40] + "…"
	}
	fmt.Printf("qrcode=%s\nimg_content=%s (len=%d)\n", resp.Qrcode, img, len(resp.QrcodeImgContent))
	return 0
}

// dumpSelftest 自检：数据目录可写 / store 原子往返 / QR 渲染 / 本地端口可用。
// 输出单行 JSON：{"evt":"selftest","ok":true,"checks":[...]}
func dumpSelftest() int {
	checks := []string{}
	fail := func(name string, err error) {
		checks = append(checks, fmt.Sprintf("FAIL %s: %v", name, err))
	}

	dir := *flagDataDir
	if dir == "" {
		dir = filepath.Join(os.TempDir(), "wx-bridge-selftest")
		fmt.Fprintf(os.Stderr, "[selftest] 未指定 --data-dir，使用临时目录: %s\n", dir)
	}

	// 1) 数据目录可写
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fail("data-dir", err)
	} else {
		p := filepath.Join(dir, ".selftest")
		if err := os.WriteFile(p, []byte("ok"), 0o644); err != nil {
			fail("data-dir 写入", err)
		} else {
			_ = os.Remove(p)
			checks = append(checks, "data-dir: OK ("+dir+")")
		}
	}

	// 2) store 原子读写往返
	type probe struct {
		V int `json:"v"`
	}
	sf := filepath.Join(dir, ".selftest.json")
	if err := store.WriteJSONAtomic(sf, probe{V: 42}); err != nil {
		fail("store 写", err)
	} else {
		var got probe
		if !store.ReadJSON(sf, &got) || got.V != 42 {
			fail("store 读", fmt.Errorf("往返值不符: %+v", got))
		} else {
			checks = append(checks, "store: OK (原子读写往返)")
		}
		_ = os.Remove(sf)
	}

	// 3) QR 渲染（PNG 签名与尺寸）
	if png, err := qr.RenderPNG("selftest://wx-bridge", qr.DefaultScale); err != nil {
		fail("qr 渲染", err)
	} else if len(png) < 100 {
		fail("qr 渲染", fmt.Errorf("输出过小: %d 字节", len(png)))
	} else {
		checks = append(checks, fmt.Sprintf("qr: OK (%d 字节 PNG)", len(png)))
	}

	// 4) 本地端口可用性
	if ln, actual, err := localhttp.ListenLocal(*flagPort); err != nil {
		fail("端口", err)
	} else {
		_ = ln.Close()
		checks = append(checks, fmt.Sprintf("port: OK (实际可用 %d)", actual))
	}

	// 汇总
	ok := true
	for _, c := range checks {
		if strings.HasPrefix(c, "FAIL") {
			ok = false
		}
	}
	data, _ := json.Marshal(map[string]any{"evt": "selftest", "ok": ok, "checks": checks})
	fmt.Println(string(data))
	if ok {
		return 0
	}
	return 1
}

// dumpQr 渲染测试二维码 PNG 到数据目录（人工用手机扫码验证可读性）。
func dumpQr() int {
	dir := *flagDataDir
	if dir == "" {
		dir = "."
	}
	text := "https://ilinkai.weixin.qq.com/qr?smoke=wx-bridge"
	png, err := qr.RenderPNG(text, qr.DefaultScale)
	if err != nil {
		fmt.Fprintf(os.Stderr, "二维码渲染失败: %v\n", err)
		return 1
	}
	out := filepath.Join(dir, "debug-qr.png")
	if err := os.WriteFile(out, png, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "写文件失败: %v\n", err)
		return 1
	}
	fmt.Printf("二维码已生成: %s\n内容: %s\n", out, text)
	return 0
}
