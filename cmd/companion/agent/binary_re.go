package agent

// 逆向分析工具集 —— 在 inspect_binary/write_binary 之上丰富：
//   binary_strings 提取可打印字符串 · binary_find 查字节/文本模式 · binary_patch 按偏移打补丁
//   binary_info 解析 PE/ELF/Mach-O 结构(stdlib debug/*) · binary_hash 哈希识别 · binary_entropy 熵(查壳)
// 全 pure-Go（stdlib），只读类不撑爆上下文（结果均有界）。

import (
	"context"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"debug/elf"
	"debug/macho"
	"debug/pe"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"os"
	"strings"
)

const maxBinaryLoad = 64 << 20 // strings/find/entropy 读全文上限 64MB

func registerBinaryRETools(r *Registry, root string) {
	r.Register(&Tool{
		Name: "binary_strings",
		Description: "从二进制提取可打印字符串（ASCII + UTF-16LE，逆向找嵌入文本/URL/符号/提示语常用）。" +
			"min_length 最短长度(默认 4)；max_results(默认 200)。返回 偏移: 字符串。",
		Parameters: objSchema(props{
			"path":        strProp("文件路径"),
			"min_length":  intProp("最短字符串长度（默认 4）"),
			"max_results": intProp("结果上限（默认 200）"),
		}, "path"),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			data, err := readBinaryCapped(root, argStr(args, "path"))
			if err != nil {
				return "", err
			}
			minLen := clampInt(argInt(args, "min_length", 4), 4, 1, 1000)
			max := clampInt(argInt(args, "max_results", 200), 200, 1, 5000)
			return extractStrings(data, minLen, max), nil
		},
	})

	r.Register(&Tool{
		Name: "binary_find",
		Description: "在二进制里查找字节模式（hex 如 4d5a 或 'ff d8 ff'）或文本（text），返回命中字节偏移（十六进制）。" +
			"hex 与 text 二选一；max_results 默认 100。",
		Parameters: objSchema(props{
			"path":        strProp("文件路径"),
			"hex":         strProp("十六进制字节模式（与 text 二选一）"),
			"text":        strProp("文本模式（与 hex 二选一）"),
			"max_results": intProp("上限（默认 100）"),
		}, "path"),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			var needle []byte
			if h := strings.TrimSpace(argStr(args, "hex")); h != "" {
				b, err := hex.DecodeString(strings.ReplaceAll(strings.ReplaceAll(h, " ", ""), "0x", ""))
				if err != nil {
					return "", fmt.Errorf("hex 解析失败: %w", err)
				}
				needle = b
			} else if t := argStr(args, "text"); t != "" {
				needle = []byte(t)
			} else {
				return "", fmt.Errorf("需提供 hex 或 text")
			}
			if len(needle) == 0 {
				return "", fmt.Errorf("模式不能为空")
			}
			data, err := readBinaryCapped(root, argStr(args, "path"))
			if err != nil {
				return "", err
			}
			max := clampInt(argInt(args, "max_results", 100), 100, 1, 5000)
			var b strings.Builder
			n, off := 0, 0
			for {
				i := indexBytes(data[off:], needle)
				if i < 0 {
					break
				}
				fmt.Fprintf(&b, "0x%08x\n", off+i)
				off += i + 1
				if n++; n >= max {
					b.WriteString("[已达上限]\n")
					break
				}
			}
			if n == 0 {
				return "（未找到）", nil
			}
			return fmt.Sprintf("命中 %d 处：\n%s", n, b.String()), nil
		},
	})

	r.Register(&Tool{
		Name: "binary_patch",
		Description: "在指定字节偏移处覆盖写入字节（hex），逆向打补丁用（如把跳转改 9090=两个 NOP）。" +
			"offset 字节偏移(0 基)；hex 要写入的字节。仅覆盖、不改文件大小。",
		Parameters: objSchema(props{
			"path":   strProp("文件路径"),
			"offset": intProp("字节偏移（0 基）"),
			"hex":    strProp("要写入的字节（十六进制）"),
		}, "path", "offset", "hex"),
		RequiresApproval: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			p, err := resolvePath(root, argStr(args, "path"))
			if err != nil {
				return "", err
			}
			raw, err := hex.DecodeString(strings.ReplaceAll(strings.TrimSpace(argStr(args, "hex")), " ", ""))
			if err != nil {
				return "", fmt.Errorf("hex 解析失败: %w", err)
			}
			if len(raw) == 0 {
				return "", fmt.Errorf("hex 不能为空")
			}
			offset := int64(argInt(args, "offset", -1))
			if offset < 0 {
				return "", fmt.Errorf("offset 无效")
			}
			fi, err := os.Stat(p)
			if err != nil {
				return "", err
			}
			if offset+int64(len(raw)) > fi.Size() {
				return "", fmt.Errorf("超出文件末尾（偏移 %d + %d 字节 > 大小 %d）；binary_patch 只覆盖不扩容", offset, len(raw), fi.Size())
			}
			f, err := os.OpenFile(p, os.O_WRONLY, 0o644)
			if err != nil {
				return "", err
			}
			defer f.Close()
			if _, err := f.WriteAt(raw, offset); err != nil {
				return "", err
			}
			return fmt.Sprintf("已在偏移 0x%x 覆盖 %d 字节", offset, len(raw)), nil
		},
	})

	r.Register(&Tool{
		Name:        "binary_info",
		Description: "解析可执行文件结构（PE/ELF/Mach-O，stdlib 解析）：架构、入口、节区(名/大小/地址)、导入库与符号、导出符号——逆向起步。",
		Parameters:  objSchema(props{"path": strProp("文件路径")}, "path"),
		ReadOnly:    true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			p, err := resolvePath(root, argStr(args, "path"))
			if err != nil {
				return "", err
			}
			return analyzeExecutable(p), nil
		},
	})

	r.Register(&Tool{
		Name:        "binary_hash",
		Description: "计算文件 大小 + MD5 + SHA1 + SHA256（识别样本/校验完整性）。流式计算，不全量载入。",
		Parameters:  objSchema(props{"path": strProp("文件路径")}, "path"),
		ReadOnly:    true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			p, err := resolvePath(root, argStr(args, "path"))
			if err != nil {
				return "", err
			}
			f, err := os.Open(p)
			if err != nil {
				return "", err
			}
			defer f.Close()
			h5, h1, h256 := md5.New(), sha1.New(), sha256.New()
			n, err := io.Copy(io.MultiWriter(h5, h1, h256), f)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("大小：%d 字节\nMD5：%x\nSHA1：%x\nSHA256：%x", n, h5.Sum(nil), h1.Sum(nil), h256.Sum(nil)), nil
		},
	})

	r.Register(&Tool{
		Name:        "binary_entropy",
		Description: "按块计算香农熵（0~8）：高熵(>7.5)提示压缩/加密/加壳区段，逆向识别壳常用。chunk_size 默认 4096。",
		Parameters: objSchema(props{
			"path":       strProp("文件路径"),
			"chunk_size": intProp("块大小字节（默认 4096）"),
		}, "path"),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			data, err := readBinaryCapped(root, argStr(args, "path"))
			if err != nil {
				return "", err
			}
			chunk := clampInt(argInt(args, "chunk_size", 4096), 4096, 256, 1<<20)
			return entropyReport(data, chunk), nil
		},
	})
}

// ─── 辅助 ────────────────────────────────────────────────────

func readBinaryCapped(root, relPath string) ([]byte, error) {
	p, err := resolvePath(root, relPath)
	if err != nil {
		return nil, err
	}
	fi, err := os.Stat(p)
	if err != nil {
		return nil, err
	}
	if fi.Size() > maxBinaryLoad {
		return nil, fmt.Errorf("文件过大（%d 字节，上限 %d MB）；用 inspect_binary 分段或缩小范围", fi.Size(), maxBinaryLoad>>20)
	}
	return os.ReadFile(p)
}

// indexBytes 在 hay 中找 needle 首次位置（-1=无）。
func indexBytes(hay, needle []byte) int {
	return strings.Index(string(hay), string(needle))
}

// extractStrings 提取 ASCII + UTF-16LE 可打印字符串（偏移: 字符串）。
func extractStrings(data []byte, minLen, max int) string {
	var b strings.Builder
	count := 0
	emit := func(off int, s string) bool {
		fmt.Fprintf(&b, "%08x: %s\n", off, s)
		count++
		return count >= max
	}
	// ASCII
	start := -1
	for i := 0; i <= len(data); i++ {
		if i < len(data) && data[i] >= 0x20 && data[i] < 0x7f {
			if start < 0 {
				start = i
			}
			continue
		}
		if start >= 0 && i-start >= minLen {
			if emit(start, string(data[start:i])) {
				return b.String() + "[已达上限]\n"
			}
		}
		start = -1
	}
	// UTF-16LE（可打印 ASCII + 0x00 交替）
	start = -1
	for i := 0; i+1 < len(data); i += 2 {
		if data[i] >= 0x20 && data[i] < 0x7f && data[i+1] == 0x00 {
			if start < 0 {
				start = i
			}
			continue
		}
		if start >= 0 && (i-start)/2 >= minLen {
			s := make([]byte, 0, (i-start)/2)
			for j := start; j < i; j += 2 {
				s = append(s, data[j])
			}
			if emit(start, "(U)"+string(s)) {
				return b.String() + "[已达上限]\n"
			}
		}
		start = -1
	}
	if count == 0 {
		return "（未提取到字符串）"
	}
	return b.String()
}

// entropyReport 整体 + 逐块香农熵（块数上限 64，超则采样步进）。
func entropyReport(data []byte, chunk int) string {
	if len(data) == 0 {
		return "（空文件）"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "整体熵：%.2f / 8.0（%d 字节）\n", shannon(data), len(data))
	nChunks := (len(data) + chunk - 1) / chunk
	step := 1
	if nChunks > 64 {
		step = nChunks / 64
	}
	b.WriteString("逐块（偏移 熵；>7.5 疑压缩/加密/加壳）：\n")
	shown := 0
	for c := 0; c < nChunks; c += step {
		off := c * chunk
		end := min(off+chunk, len(data))
		e := shannon(data[off:end])
		flag := ""
		if e > 7.5 {
			flag = " ⚠高熵"
		}
		fmt.Fprintf(&b, "0x%08x: %.2f%s\n", off, e, flag)
		if shown++; shown >= 64 {
			break
		}
	}
	return b.String()
}

func shannon(data []byte) float64 {
	if len(data) == 0 {
		return 0
	}
	var freq [256]int
	for _, c := range data {
		freq[c]++
	}
	e := 0.0
	n := float64(len(data))
	for _, f := range freq {
		if f == 0 {
			continue
		}
		p := float64(f) / n
		e -= p * math.Log2(p)
	}
	return e
}

// analyzeExecutable 依次尝试 PE/ELF/Mach-O 解析，输出结构摘要（有界）。
func analyzeExecutable(p string) string {
	if pf, err := pe.Open(p); err == nil {
		defer pf.Close()
		return describePE(pf)
	}
	if ef, err := elf.Open(p); err == nil {
		defer ef.Close()
		return describeELF(ef)
	}
	if mf, err := macho.Open(p); err == nil {
		defer mf.Close()
		return describeMachO(mf)
	}
	return "无法识别为 PE/ELF/Mach-O（binary_info 仅支持这三种可执行格式；其它用 inspect_binary 看原始字节）"
}

func describePE(f *pe.File) string {
	var b strings.Builder
	arch := map[uint16]string{0x8664: "x86-64", 0x14c: "x86", 0xaa64: "ARM64", 0x1c0: "ARM"}[f.Machine]
	if arch == "" {
		arch = fmt.Sprintf("machine=0x%x", f.Machine)
	}
	fmt.Fprintf(&b, "格式：PE（Windows）  架构：%s  节区数：%d\n\n节区：\n", arch, len(f.Sections))
	for i, s := range f.Sections {
		if i >= 30 {
			b.WriteString("…\n")
			break
		}
		fmt.Fprintf(&b, "  %-10s 虚拟大小 %-8d 虚拟地址 0x%x\n", s.Name, s.VirtualSize, s.VirtualAddress)
	}
	if libs, err := f.ImportedLibraries(); err == nil && len(libs) > 0 {
		b.WriteString("\n导入库：" + strings.Join(libs, ", ") + "\n")
	}
	if syms, err := f.ImportedSymbols(); err == nil && len(syms) > 0 {
		b.WriteString("\n导入符号（前 50）：\n" + capList(syms, 50))
	}
	return b.String()
}

func describeELF(f *elf.File) string {
	var b strings.Builder
	fmt.Fprintf(&b, "格式：ELF（Unix）  架构：%s  类型：%s  入口：0x%x  节区数：%d\n\n节区：\n",
		f.Machine, f.Type, f.Entry, len(f.Sections))
	for i, s := range f.Sections {
		if i >= 30 {
			b.WriteString("…\n")
			break
		}
		fmt.Fprintf(&b, "  %-16s 大小 %-8d 地址 0x%x\n", s.Name, s.Size, s.Addr)
	}
	if libs, err := f.ImportedLibraries(); err == nil && len(libs) > 0 {
		b.WriteString("\n依赖库：" + strings.Join(libs, ", ") + "\n")
	}
	if syms, err := f.ImportedSymbols(); err == nil && len(syms) > 0 {
		names := make([]string, 0, len(syms))
		for _, s := range syms {
			names = append(names, s.Name)
		}
		b.WriteString("\n导入符号（前 50）：\n" + capList(names, 50))
	}
	return b.String()
}

func describeMachO(f *macho.File) string {
	var b strings.Builder
	fmt.Fprintf(&b, "格式：Mach-O（macOS）  架构：%s  类型：%s  节区数：%d\n\n节区：\n", f.Cpu, f.Type, len(f.Sections))
	for i, s := range f.Sections {
		if i >= 30 {
			b.WriteString("…\n")
			break
		}
		fmt.Fprintf(&b, "  %-16s 大小 %-8d 地址 0x%x\n", s.Seg+","+s.Name, s.Size, s.Addr)
	}
	if libs, err := f.ImportedLibraries(); err == nil && len(libs) > 0 {
		b.WriteString("\n依赖库：" + strings.Join(libs, ", ") + "\n")
	}
	if syms, err := f.ImportedSymbols(); err == nil && len(syms) > 0 {
		b.WriteString("\n导入符号（前 50）：\n" + capList(syms, 50))
	}
	return b.String()
}

func capList(items []string, max int) string {
	if len(items) > max {
		return strings.Join(items[:max], "\n") + fmt.Sprintf("\n…（共 %d 个）", len(items))
	}
	return strings.Join(items, "\n")
}
