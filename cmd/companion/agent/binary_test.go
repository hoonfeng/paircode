package agent

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestInspectBinary 二进制分析：magic bytes 嗅探类型(PNG) + hexdump 预览。
func TestInspectBinary(t *testing.T) {
	root := t.TempDir()
	reg := NewRegistry()
	RegisterDefaultTools(reg, root)
	ctx := context.Background()
	png := append([]byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}, make([]byte, 20)...)
	os.WriteFile(filepath.Join(root, "img.png"), png, 0o644)

	out, err := reg.Execute(ctx, "inspect_binary", `{"path":"img.png"}`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "PNG 图片") {
		t.Errorf("应嗅探出 PNG：%q", out)
	}
	if !strings.Contains(out, "89 50 4e 47") {
		t.Errorf("hexdump 应含 magic bytes：%q", out)
	}
}

// TestReadFileBinaryGuard read_file 遇二进制报错并引导到 inspect_binary（不把字节灌进上下文）。
func TestReadFileBinaryGuard(t *testing.T) {
	root := t.TempDir()
	reg := NewRegistry()
	RegisterDefaultTools(reg, root)
	ctx := context.Background()
	os.WriteFile(filepath.Join(root, "blob.bin"), []byte{0x00, 0x01, 0x02, 0x00, 0xff}, 0o644)
	if _, err := reg.Execute(ctx, "read_file", `{"path":"blob.bin"}`); err == nil || !strings.Contains(err.Error(), "inspect_binary") {
		t.Errorf("读二进制应报错并引导 inspect_binary，得 %v", err)
	}
}

// TestWriteBinary base64 → 文件，字节往返一致。
func TestWriteBinary(t *testing.T) {
	root := t.TempDir()
	reg := NewRegistry()
	RegisterDefaultTools(reg, root)
	ctx := context.Background()
	raw := []byte{0xDE, 0xAD, 0xBE, 0xEF}
	b64 := base64.StdEncoding.EncodeToString(raw)
	if _, err := reg.Execute(ctx, "write_binary", `{"path":"x.bin","base64":"`+b64+`"}`); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(filepath.Join(root, "x.bin")); string(got) != string(raw) {
		t.Errorf("写回字节不符：%x", got)
	}
}

// TestBinaryRETools 逆向工具：strings/find(text+hex)/hash/entropy/patch(含越界守卫)。
func TestBinaryRETools(t *testing.T) {
	root := t.TempDir()
	reg := NewRegistry()
	RegisterDefaultTools(reg, root)
	ctx := context.Background()
	content := []byte("Hello")
	content = append(content, 0x00, 0x00, 'M', 'Z')
	content = append(content, []byte("world magic")...)
	os.WriteFile(filepath.Join(root, "s.bin"), content, 0o644)

	if str, _ := reg.Execute(ctx, "binary_strings", `{"path":"s.bin","min_length":4}`); !strings.Contains(str, "Hello") || !strings.Contains(str, "world") {
		t.Errorf("strings 应提取 Hello/world：%q", str)
	}
	if fnd, _ := reg.Execute(ctx, "binary_find", `{"path":"s.bin","text":"world"}`); !strings.Contains(fnd, "命中") {
		t.Errorf("find text 应命中：%q", fnd)
	}
	if fhex, _ := reg.Execute(ctx, "binary_find", `{"path":"s.bin","hex":"4d5a"}`); !strings.Contains(fhex, "命中") {
		t.Errorf("find hex(MZ) 应命中：%q", fhex)
	}
	if h, _ := reg.Execute(ctx, "binary_hash", `{"path":"s.bin"}`); !strings.Contains(h, "SHA256") {
		t.Errorf("hash 应含 SHA256：%q", h)
	}
	if e, _ := reg.Execute(ctx, "binary_entropy", `{"path":"s.bin"}`); !strings.Contains(e, "整体熵") {
		t.Errorf("entropy 应含整体熵：%q", e)
	}
	if _, err := reg.Execute(ctx, "binary_patch", `{"path":"s.bin","offset":0,"hex":"90"}`); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(filepath.Join(root, "s.bin")); got[0] != 0x90 {
		t.Errorf("patch 后偏移 0 应为 0x90，得 0x%x", got[0])
	}
	if _, err := reg.Execute(ctx, "binary_patch", `{"path":"s.bin","offset":99999,"hex":"90"}`); err == nil {
		t.Error("越界 patch 应报错")
	}
}

// TestExtraSkipDirs 用户配置的额外忽略目录（SetExtraSkipDirs）即时生效；清空后恢复。
func TestExtraSkipDirs(t *testing.T) {
	defer SetExtraSkipDirs(nil) // 复原全局态
	root := t.TempDir()
	reg := NewRegistry()
	RegisterDefaultTools(reg, root)
	ctx := context.Background()
	for _, d := range []string{"generated", "src"} {
		os.MkdirAll(filepath.Join(root, d), 0o755)
		os.WriteFile(filepath.Join(root, d, "f.txt"), []byte("needle 针"), 0o644)
	}
	if out, _ := reg.Execute(ctx, "search_content", `{"pattern":"needle"}`); !strings.Contains(out, "generated/") {
		t.Errorf("默认 generated 不在跳过表，应命中：%q", out)
	}
	SetExtraSkipDirs([]string{"generated"})
	out, _ := reg.Execute(ctx, "search_content", `{"pattern":"needle"}`)
	if strings.Contains(out, "generated/") {
		t.Errorf("配置忽略后不应命中 generated：%q", out)
	}
	if !strings.Contains(out, "src/") {
		t.Errorf("src 仍应命中：%q", out)
	}
}

// TestSearchSkipsExpandedDirs 扩展的跳过目录（依赖库/产物，如 target）不进搜索，src 正常命中。
func TestSearchSkipsExpandedDirs(t *testing.T) {
	root := t.TempDir()
	reg := NewRegistry()
	RegisterDefaultTools(reg, root)
	ctx := context.Background()
	for _, d := range []string{"target", "__pycache__", "src"} {
		os.MkdirAll(filepath.Join(root, d), 0o755)
		os.WriteFile(filepath.Join(root, d, "f.txt"), []byte("needle 针"), 0o644)
	}
	out, _ := reg.Execute(ctx, "search_content", `{"pattern":"needle"}`)
	if strings.Contains(out, "target/") || strings.Contains(out, "__pycache__/") {
		t.Errorf("应跳过依赖库/产物目录：%q", out)
	}
	if !strings.Contains(out, "src/f.txt") {
		t.Errorf("应搜到 src：%q", out)
	}
}
