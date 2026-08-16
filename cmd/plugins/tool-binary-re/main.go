// ═══════════════════════════════════════════════════════════════
// tool-binary-re — 独立插件二进制（逆向分析工具集）
//
// ★ 二进制插件协议（宿主 ctx.binary.exec 调用）：
//   stdin  一行 JSON：{"tool":"binary_strings","args":{...},"root":"<工作区根>"}
//   stdout 一行 JSON：{"ok":true,"text":"..."} | {"ok":false,"error":"..."}
//   exit 0（协议错误 exit 2）
//
// 与内置 Go 实现（internal/agent/binary_re.go）行为一致，纯 stdlib 无外部依赖。
// 装配位置：.pair/plugins/tool-binary-re/bin/tool-binary-re.exe（插件目录自包含）。
// 资源（若未来需要）放 .pair/plugins/tool-binary-re/assets/，本进程可从
// os.Executable() 上级目录定位。
// ═══════════════════════════════════════════════════════════════
package main

import (
	"bufio"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"debug/elf"
	"debug/macho"
	"debug/pe"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
)

const maxBinaryLoad = 64 << 20 // strings/find/entropy 读全文上限 64MB

// request / response（JSON 协议）
type request struct {
	Tool string         `json:"tool"`
	Args map[string]any `json:"args"`
	Root string         `json:"root"`
}

type response struct {
	OK    bool   `json:"ok"`
	Text  string `json:"text,omitempty"`
	Error string `json:"error,omitempty"`
}

func main() {
	sc := bufio.NewScanner(os.Stdin)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	var line string
	for sc.Scan() {
		line = sc.Text()
		break
	}
	if err := sc.Err(); err != nil {
		respond(okResp("", fmt.Errorf("stdin 读取失败: %w", err)))
		os.Exit(2)
	}
	if strings.TrimSpace(line) == "" {
		respond(okResp("", fmt.Errorf("空请求")))
		os.Exit(2)
	}
	var req request
	if err := json.Unmarshal([]byte(line), &req); err != nil {
		respond(okResp("", fmt.Errorf("请求 JSON 解析失败: %w", err)))
		os.Exit(2)
	}
	if req.Root == "" {
		respond(okResp("", fmt.Errorf("缺少 root（工作区根）")))
		os.Exit(2)
	}
	text, err := dispatch(req.Tool, req.Args, req.Root)
	respond(okResp(text, err))
}

func okResp(text string, err error) response {
	if err != nil {
		return response{OK: false, Error: err.Error()}
	}
	return response{OK: true, Text: text}
}

func respond(r response) {
	b, _ := json.Marshal(r)
	fmt.Println(string(b))
}

// dispatch 按工具名分发（与内置 registerBinaryRETools 一一对应）。
func dispatch(tool string, args map[string]any, root string) (string, error) {
	switch tool {
	case "binary_strings":
		data, err := readBinaryCapped(root, argStr(args, "path"))
		if err != nil {
			return "", err
		}
		minLen := clampInt(argInt(args, "min_length", 4), 4, 1, 1000)
		max := clampInt(argInt(args, "max_results", 200), 200, 1, 5000)
		return extractStrings(data, minLen, max), nil
	case "binary_find":
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
	case "binary_patch":
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
	case "binary_info":
		p, err := resolvePath(root, argStr(args, "path"))
		if err != nil {
			return "", err
		}
		return analyzeExecutable(p), nil
	case "binary_hash":
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
	case "binary_entropy":
		data, err := readBinaryCapped(root, argStr(args, "path"))
		if err != nil {
			return "", err
		}
		chunk := clampInt(argInt(args, "chunk_size", 4096), 4096, 256, 1<<20)
		return entropyReport(data, chunk), nil
	}
	return "", fmt.Errorf("未知工具 %q（支持：binary_strings/binary_find/binary_patch/binary_info/binary_hash/binary_entropy）", tool)
}

// ─── 路径解析（与宿主 resolvePath 同语义：相对 root、越界拦截）───

func resolvePath(root, p string) (string, error) {
	if strings.TrimSpace(p) == "" {
		return "", fmt.Errorf("path 不能为空")
	}
	full := p
	if !filepath.IsAbs(full) {
		full = filepath.Join(root, full)
	}
	full = filepath.Clean(full)
	// 越界拦截：必须位于 root（或其子目录）内
	if !strings.EqualFold(full, root) && !strings.HasPrefix(strings.ToLower(full), strings.ToLower(root)+string(os.PathSeparator)) {
		return "", fmt.Errorf("路径越界（仅限工作区内）: %s", p)
	}
	return full, nil
}

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

// ─── 参数辅助 ────────────────────────────────────────────────

func argStr(args map[string]any, key string) string {
	if args == nil {
		return ""
	}
	if v, ok := args[key]; ok && v != nil {
		return fmt.Sprintf("%v", v)
	}
	return ""
}

func argInt(args map[string]any, key string, def int) int {
	if args == nil {
		return def
	}
	if v, ok := args[key]; ok && v != nil {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		case int64:
			return int(n)
		case string:
			var i int
			if _, err := fmt.Sscanf(n, "%d", &i); err == nil {
				return i
			}
		}
	}
	return def
}

func clampInt(v, def, min, max int) int {
	if v == def {
		return v
	}
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// ─── 分析与报告（与内置实现逐行对齐）──────────────────────────

func indexBytes(hay, needle []byte) int {
	return strings.Index(string(hay), string(needle))
}

func extractStrings(data []byte, minLen, max int) string {
	var b strings.Builder
	count := 0
	emit := func(off int, s string) bool {
		fmt.Fprintf(&b, "%08x: %s\n", off, s)
		count++
		return count >= max
	}
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
		flag := " [警告]高熵"
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
