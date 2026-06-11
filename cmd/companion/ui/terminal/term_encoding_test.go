//go:build windows

package termpanel

import (
	"testing"

	"github.com/user/gou-ide/cmd/companion/core"
	"golang.org/x/text/encoding/simplifiedchinese"
)

// TestGBKStreamingDecode 在每个可能的切点把 GBK 字节分两段喂 decodeOutput，
// 验证流式拼接（残留尾字节跨帧补全）后等于原始 UTF-8——模拟 PTY 任意分包。
func TestGBKStreamingDecode(t *testing.T) {
	orig := "你好abc世界！（GBK 编码测试）"
	gbk, err := simplifiedchinese.GBK.NewEncoder().Bytes([]byte(orig))
	if err != nil {
		t.Fatal(err)
	}
	old := core.Settings.TermEncoding
	core.Settings.TermEncoding = "gbk"
	defer func() { core.Settings.TermEncoding = old }()

	for cut := 0; cut <= len(gbk); cut++ {
		ts := &terminalState{}
		out := append([]byte(nil), ts.decodeOutput(gbk[:cut])...)
		out = append(out, ts.decodeOutput(gbk[cut:])...)
		if string(out) != orig {
			t.Fatalf("cut=%d: 得 %q，期望 %q", cut, string(out), orig)
		}
	}
}

// TestDecodeOutputPassthrough 非 gbk 编码（auto/utf-8）原样透传，不动字节。
func TestDecodeOutputPassthrough(t *testing.T) {
	old := core.Settings.TermEncoding
	defer func() { core.Settings.TermEncoding = old }()
	raw := []byte("你好 UTF-8 直通")
	for _, enc := range []string{"auto", "utf-8", ""} {
		core.Settings.TermEncoding = enc
		ts := &terminalState{}
		if string(ts.decodeOutput(raw)) != string(raw) {
			t.Errorf("enc=%q 应原样透传", enc)
		}
	}
}
