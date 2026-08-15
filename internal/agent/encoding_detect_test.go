// encoding_detect 单测：UTF-8 原样 / GBK 解码 / 混合回退。
package agent

import (
	"strings"
	"testing"

	"golang.org/x/text/encoding/simplifiedchinese"
)

func TestDecodeCmdOutput(t *testing.T) {
	// 1. 纯 ASCII：原样
	if s := decodeCmdOutput([]byte("hello world\n")); s != "hello world\n" {
		t.Errorf("ASCII 应原样，实际 %q", s)
	}

	// 2. 合法 UTF-8 中文：原样
	utf8Str := "中文输出：构建成功 ✓\n"
	if s := decodeCmdOutput([]byte(utf8Str)); s != utf8Str {
		t.Errorf("UTF-8 应原样，实际 %q", s)
	}

	// 3. GBK 字节流：解码为正确中文
	gbk, err := simplifiedchinese.GBK.NewEncoder().String("中文乱码测试")
	if err != nil {
		t.Fatalf("GBK 编码失败: %v", err)
	}
	if s := decodeCmdOutput([]byte(gbk)); s != "中文乱码测试" {
		t.Errorf("GBK 应解码为中文，实际 %q", s)
	}

	// 4. GBK + ASCII 混合：整体解码后英文保留
	gbkAll, _ := simplifiedchinese.GBK.NewEncoder().String("警告：磁盘空间不足")
	gbkMixedOut := "warning: " + gbkAll
	if s := decodeCmdOutput([]byte(gbkMixedOut)); !strings.Contains(s, "warning: ") || !strings.Contains(s, "磁盘空间不足") {
		t.Errorf("GBK+ASCII 混合应都保留，实际 %q", s)
	}

	// 5. 非法字节（非 UTF-8 且 GBK 解码也失败）：回退原始字节不 panic
	bad := []byte{0xFF, 0xFE, 0x00, 0x81}
	s := decodeCmdOutput(bad)
	if s == "" {
		t.Error("回退不应为空串")
	}
}
