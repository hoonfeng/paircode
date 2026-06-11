package agent

// 二进制 读/写/分析工具 —— read_file 只读文本，二进制走这里，避免把原始字节当文本灌进上下文撑爆。
//   · inspect_binary：大小 + 嗅探类型（magic bytes）+ 区段十六进制/ASCII 预览（hexdump 风格，有界）。
//   · write_binary：base64 → 文件。

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const maxReadFileSize = 10 << 20 // read_file 文本上限 10MB（更大用 offset/limit 或 search_content）

func registerBinaryTools(r *Registry, root string) {
	r.Register(&Tool{
		Name: "inspect_binary",
		Description: "分析二进制文件而不撑爆上下文：返回大小 + 嗅探类型（按 magic bytes）+ 指定区段的十六进制/ASCII 预览" +
			"（hexdump 风格）。读图片/可执行/压缩包/字体等二进制用它，别用 read_file。",
		Parameters: objSchema(props{
			"path":   strProp("文件路径（工作区内）"),
			"offset": intProp("可选：起始字节偏移（默认 0）"),
			"length": intProp("可选：预览字节数（默认 256，上限 4096）"),
		}, "path"),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			p, err := resolvePath(root, argStr(args, "path"))
			if err != nil {
				return "", err
			}
			fi, err := os.Stat(p)
			if err != nil {
				return "", err
			}
			if fi.IsDir() {
				return "", fmt.Errorf("「%s」是目录，用 list_files", argStr(args, "path"))
			}
			offset := int64(argInt(args, "offset", 0))
			if offset < 0 {
				offset = 0
			}
			length := clampInt(argInt(args, "length", 256), 256, 1, 4096)
			f, err := os.Open(p)
			if err != nil {
				return "", err
			}
			defer f.Close()
			head := make([]byte, 16) // 文件头（嗅探类型）
			hn, _ := f.Read(head)
			buf := make([]byte, length) // 预览区段
			n := 0
			if offset < fi.Size() {
				f.Seek(offset, 0)
				n, _ = f.Read(buf)
			}
			var b strings.Builder
			fmt.Fprintf(&b, "文件：%s\n大小：%d 字节\n类型：%s\n\n十六进制预览（偏移 %d，%d 字节）：\n%s",
				argStr(args, "path"), fi.Size(), detectFileType(head[:hn], fi.Name()), offset, n, hexDump(buf[:n], offset))
			return b.String(), nil
		},
	})

	r.Register(&Tool{
		Name:             "write_binary",
		Description:      "把 base64 编码的字节写入文件（path；覆盖；父目录自动创建）。用于写二进制内容。",
		Parameters:       objSchema(props{"path": strProp("文件路径"), "base64": strProp("base64 编码的字节")}, "path", "base64"),
		RequiresApproval: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			p, err := resolvePath(root, argStr(args, "path"))
			if err != nil {
				return "", err
			}
			raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(argStr(args, "base64")))
			if err != nil {
				return "", fmt.Errorf("base64 解码失败: %w", err)
			}
			if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
				return "", err
			}
			if err := os.WriteFile(p, raw, 0o644); err != nil {
				return "", err
			}
			return fmt.Sprintf("已写入 %s（%d 字节）", argStr(args, "path"), len(raw)), nil
		},
	})
}

// detectFileType 按 magic bytes 嗅探常见类型，回退扩展名。
func detectFileType(head []byte, name string) string {
	has := func(sig ...byte) bool {
		if len(head) < len(sig) {
			return false
		}
		for i, b := range sig {
			if head[i] != b {
				return false
			}
		}
		return true
	}
	switch {
	case has(0x89, 'P', 'N', 'G'):
		return "PNG 图片"
	case has(0xFF, 0xD8, 0xFF):
		return "JPEG 图片"
	case has('G', 'I', 'F', '8'):
		return "GIF 图片"
	case has('B', 'M'):
		return "BMP 图片"
	case has('R', 'I', 'F', 'F'):
		return "RIFF（WAV/AVI/WebP）"
	case has('%', 'P', 'D', 'F'):
		return "PDF 文档"
	case has('P', 'K', 0x03, 0x04):
		return "ZIP（或 jar/docx/xlsx 等）"
	case has(0x1F, 0x8B):
		return "GZIP 压缩"
	case has('7', 'z', 0xBC, 0xAF):
		return "7z 压缩"
	case has(0x7F, 'E', 'L', 'F'):
		return "ELF 可执行（Linux）"
	case has('M', 'Z'):
		return "PE 可执行（Windows .exe/.dll）"
	case has(0xCA, 0xFE, 0xBA, 0xBE):
		return "Java class / Mach-O"
	case has(0x00, 0x01, 0x00, 0x00):
		return "TTF 字体 / ICO"
	case has('O', 'T', 'T', 'O'):
		return "OTF 字体"
	}
	if ext := strings.ToLower(filepath.Ext(name)); ext != "" {
		return "未知二进制（扩展名 " + ext + "）"
	}
	return "未知二进制"
}

// hexDump 经典 hexdump：8 位偏移 + 16 字节十六进制（中分） + ASCII（不可见→.）。
func hexDump(data []byte, baseOffset int64) string {
	if len(data) == 0 {
		return "（空）\n"
	}
	var b strings.Builder
	for i := 0; i < len(data); i += 16 {
		fmt.Fprintf(&b, "%08x  ", baseOffset+int64(i))
		end := min(i+16, len(data))
		for j := i; j < i+16; j++ {
			if j < end {
				fmt.Fprintf(&b, "%02x ", data[j])
			} else {
				b.WriteString("   ")
			}
			if j == i+7 {
				b.WriteByte(' ')
			}
		}
		b.WriteString(" |")
		for j := i; j < end; j++ {
			if c := data[j]; c >= 0x20 && c < 0x7f {
				b.WriteByte(c)
			} else {
				b.WriteByte('.')
			}
		}
		b.WriteString("|\n")
	}
	return b.String()
}
