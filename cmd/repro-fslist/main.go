// ═══════════════════════════════════════════════════════════════
// repro-fslist —— 「添加工作区目录列表返回空」问题复现诊断单例
//
// 背景：PairCode IDE 中「添加工作区」的目录浏览器与「项目文件树」均调用
//   GET /api/fs/list（fs-api 磁盘插件 .pair/plugins/fs-api/index.js）：
//   · browse=1（添加工作区/新建项目选目录）→ ctx.bash.exec 调 PowerShell
//     Get-ChildItem，命令失败时插件容错返回空数组【无任何报错】；
//   · 非 browse（项目展开文件树）→ ctx.fs.readdir（工作区受限服务），
//     失败时前端 try/catch 静默吞掉 → 树为空。
//
// 本程序 1:1 复刻上述两条链路的执行细节，在任意机器上独立运行，
// 输出诊断结论，用于定位「部分系统返回空」的环境差异。
//
// 使用（零第三方依赖，仅标准库）：
//   go run main.go [目录路径]        （路径缺省 = 当前目录）
//   go build main.go && main.exe "D:\某目录"   （编译后发给测试者更省事）
// 建议在 PowerShell / Windows Terminal 中运行（避免旧 cmd 控制台中文乱码）。
// ═══════════════════════════════════════════════════════════════
package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unicode/utf16"
	"unicode/utf8"
)

const (
	shellTimeout = 120 * time.Second // 与 ctx.bash.exec 默认超时一致
	b64chars     = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
)

// ─────────────────────────────────────────────────────────────
// 1. 环境探测（复刻 shell.go detectBash + LookPath）
// ─────────────────────────────────────────────────────────────

func detectBash() (bashPath, msysBin string) {
	for _, cand := range []string{
		`C:\Program Files\Git\usr\bin\bash.exe`,
		`C:\Program Files (x86)\Git\usr\bin\bash.exe`,
	} {
		if st, err := os.Stat(cand); err == nil && !st.IsDir() {
			return cand, filepath.Dir(cand)
		}
	}
	if p, err := exec.LookPath("bash"); err == nil {
		return p, ""
	}
	return "", ""
}

func hideWindow(c *exec.Cmd) {
	if runtime.GOOS == "windows" {
		if c.SysProcAttr == nil {
			c.SysProcAttr = &syscall.SysProcAttr{}
		}
		c.SysProcAttr.HideWindow = true
	}
}

// runShellWrapped 复刻 newShellCommand（shell.go）：
//   Git Bash 可用 → bash -c cmd（msys bin 前置 PATH）
//   否则        → cmd /C chcp 65001 >nul & cmd
// 返回 (stdout, stderr, exitErr)
func runShellWrapped(command, cwd string, timeout time.Duration) (string, string, string) {
	var c *exec.Cmd
	if bashPath, msysBin := detectBash(); bashPath != "" {
		c = exec.Command(bashPath, "-c", command)
		if msysBin != "" {
			c.Env = append(os.Environ(), "PATH="+msysBin+";"+os.Getenv("PATH"))
		}
	} else {
		c = exec.Command("cmd", "/C", "chcp 65001 >nul & "+command)
	}
	hideWindow(c)
	// 合并 stdout/stderr 会互相污染，这里分开捕获以便诊断
	var outBuf, errBuf strings.Builder
	c.Stdout = &outBuf
	c.Stderr = &errBuf
	c.Dir = cwd
	if err := c.Start(); err != nil {
		return "", "", fmt.Sprintf("启动失败: %v", err)
	}
	done := make(chan error, 1)
	go func() { done <- c.Wait() }()
	select {
	case err := <-done:
		if err != nil {
			return decodeText([]byte(outBuf.String())), decodeText([]byte(errBuf.String())), err.Error()
		}
	case <-time.After(timeout):
		_ = c.Process.Kill()
		<-done
		return decodeText([]byte(outBuf.String())), decodeText([]byte(errBuf.String())), "超时 " + timeout.String()
	}
	return decodeText([]byte(outBuf.String())), decodeText([]byte(errBuf.String())), ""
}

// runPSDirect 直接参数化启动 powershell（不经 bash/cmd 包装、无 2>/dev/null 重定向）
func runPSDirect(b64 string, cwd string, timeout time.Duration) (string, string, string) {
	psPath, err := exec.LookPath("powershell")
	if err != nil {
		return "", "", "powershell 不在 PATH: " + err.Error()
	}
	c := exec.Command(psPath, "-NoProfile", "-NonInteractive", "-EncodedCommand", b64)
	hideWindow(c)
	var outBuf, errBuf strings.Builder
	c.Stdout = &outBuf
	c.Stderr = &errBuf
	c.Dir = cwd
	if err := c.Start(); err != nil {
		return "", "", "启动失败: " + err.Error()
	}
	done := make(chan error, 1)
	go func() { done <- c.Wait() }()
	select {
	case err := <-done:
		if err != nil {
			return outBuf.String(), errBuf.String(), err.Error()
		}
	case <-time.After(timeout):
		_ = c.Process.Kill()
		<-done
		return outBuf.String(), errBuf.String(), "超时 " + timeout.String()
	}
	return outBuf.String(), errBuf.String(), ""
}

// ─────────────────────────────────────────────────────────────
// 2. 复刻 fs-api 的 PowerShell 命令构造
// ─────────────────────────────────────────────────────────────

// normalizeBrowsePath 复刻 fs-api index.js browse 分支的路径规范化：
//   双反斜杠折叠 → 盘根保留末尾斜杠（F:\）→ 普通目录去尾部斜杠 → 单引号转义
func normalizeBrowsePath(p string) (string, error) {
	re := regexp.MustCompile(`\\{2,}`)
	dir := re.ReplaceAllString(p, `\`)
	if m := regexp.MustCompile(`^([a-zA-Z]):[\\/]?$`).FindStringSubmatch(dir); m != nil {
		dir = m[1] + ":\\"
	} else {
		dir = regexp.MustCompile(`[\\/]+$`).ReplaceAllString(dir, "")
	}
	if dir == "" {
		return "", fmt.Errorf("路径为空（fs-api 对空路径直接返回空数组）")
	}
	return dir, nil
}

// buildPSCommand 复刻 fs-api index.js 72-89 行：
//   1) 拼 PowerShell 脚本（Get-ChildItem -Force -LiteralPath）
//   2) 脚本 → UTF-16LE 字节 → base64（-EncodedCommand 传输，规避引号/中文编码）
func buildPSCommand(dir string) (b64 string, psScript string) {
	esc := strings.ReplaceAll(dir, "'", "''")
	ps := `$ProgressPreference='SilentlyContinue'; Get-ChildItem -Force -LiteralPath '` + esc + `' | ForEach-Object { [PSCustomObject]@{ n = $_.Name; d = $_.PSIsContainer; s = $_.Length } } | ConvertTo-Json -Compress`
	// UTF-16LE（与 JS charCodeAt 语义一致：utf16.Encode 处理代理对）
	u16 := utf16.Encode([]rune(ps))
	var bytes []byte
	for _, c := range u16 {
		bytes = append(bytes, byte(c&0xff), byte(c>>8))
	}
	return base64.StdEncoding.EncodeToString(bytes), ps
}

// parsePSOutput 复刻 fs-api 对 PowerShell 输出的处理：JSON 解析 → 条目数组
func parsePSOutput(text string) (entries []map[string]any, parseErr string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, "" // 输出为空 → 空目录（无报错）
	}
	var arr []map[string]any
	if err := json.Unmarshal([]byte(text), &arr); err != nil {
		var one map[string]any
		if err2 := json.Unmarshal([]byte(text), &one); err2 != nil {
			return nil, "JSON 解析失败: " + err.Error() + "（原文前 200 字节: " + preview(text, 200) + "）"
		}
		arr = []map[string]any{one}
	}
	return arr, ""
}

// ─────────────────────────────────────────────────────────────
// 3. 编码探测（复刻 encoding_detect.go：UTF-8 优先，非 UTF-8 → 提示 GBK/其他）
//    纯标准库不引 x/text：非 UTF-8 时给出字节分析提示
// ─────────────────────────────────────────────────────────────
func decodeText(b []byte) string {
	if utf8.Valid(b) {
		return string(b)
	}
	// 非 UTF-8：统计高位字节比例，粗略判断 GBK/UTF-16 可能
	high := 0
	for _, c := range b {
		if c >= 0x80 {
			high++
		}
	}
	if high > 0 && high%2 == 0 {
		return fmt.Sprintf("[非UTF-8 字节，可能 GBK/UTF-16，len=%d highBytes=%d] %s", len(b), high, preview(string(b), 120))
	}
	return fmt.Sprintf("[非UTF-8 字节 len=%d] %s", len(b), preview(string(b), 120))
}

func preview(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "...[截断]"
}

// ─────────────────────────────────────────────────────────────
// 4. 原生对照（模拟 ctx.fs.readdir 非 browse 分支 / os.ReadDir）
// ─────────────────────────────────────────────────────────────
func listNative(path string) ([]string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names, nil
}

// ─────────────────────────────────────────────────────────────
// 5. 附加探测：PowerShell 版本 / 执行策略 / 语言模式
// ─────────────────────────────────────────────────────────────
func psProbe(cwd string) {
	fmt.Println("\n[附加探测] PowerShell 环境能力")
	b64 := ""
	probe := func(script string) (string, bool) {
		u16 := utf16.Encode([]rune(script))
		var bytes []byte
		for _, c := range u16 {
			bytes = append(bytes, byte(c&0xff), byte(c>>8))
		}
		b64 = base64.StdEncoding.EncodeToString(bytes)
		out, _, errStr := runPSDirect(b64, cwd, 15*time.Second)
		if errStr != "" {
			return "", false
		}
		return strings.TrimSpace(out), true
	}

	if out, ok := probe(`$PSVersionTable.PSVersion.ToString()`); ok {
		fmt.Printf("  PowerShell 版本 : %s\n", out)
	} else {
		fmt.Printf("  PowerShell 版本 : 查询失败（被限制或不可用）\n")
	}
	if out, ok := probe(`(Get-ExecutionPolicy -Scope LocalMachine -ErrorAction SilentlyContinue); (Get-ExecutionPolicy -Scope CurrentUser -ErrorAction SilentlyContinue)`); ok {
		fmt.Printf("  执行策略        : %s\n", strings.ReplaceAll(out, "\n", " / "))
	} else {
		fmt.Printf("  执行策略        : 查询失败\n")
	}
	if out, ok := probe(`$ExecutionContext.SessionState.LanguageMode`); ok {
		fmt.Printf("  语言模式        : %s（ConstrainedLanguage = 受限，会拦截多数命令）\n", out)
	} else {
		fmt.Printf("  语言模式        : 查询失败（受限环境常见：连语言模式都查不到）\n")
	}
}

// ─────────────────────────────────────────────────────────────
// 6. 主流程
// ─────────────────────────────────────────────────────────────
func main() {
	fmt.Println("══════════════════════════════════════════════════════════════")
	fmt.Println(" repro-fslist —— 复现「添加工作区目录列表返回空」诊断工具")
	fmt.Println("══════════════════════════════════════════════════════════════")

	dir := "."
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		fmt.Printf("⚠ 路径解析失败: %v\n", err)
		abs = dir
	}
	abs = filepath.Clean(abs)
	// 模块内运行时 cwd：复刻 fs-api ctx.bash.exec（cwd = 工作区根）
	cwd, _ := os.Getwd()

	fmt.Printf("目标目录: %s\n", abs)
	fmt.Printf("程序 cwd: %s（模拟 ctx.bash.exec 的工作目录）\n\n", cwd)

	// ── 0. 环境信息 ──
	fmt.Println("── [1] 环境信息 ──")
	fmt.Printf("  GOOS/GOARCH   : %s/%s\n", runtime.GOOS, runtime.GOARCH)
	bashPath, msysBin := detectBash()
	if bashPath != "" {
		fmt.Printf("  Git Bash       : %s (msys bin: %s)\n", bashPath, msysBin)
		fmt.Println("  → 产品将走 [bash -c] 执行（2>/dev/null 语法合法）")
	} else {
		fmt.Println("  Git Bash       : ✗ 未找到")
		fmt.Println("  → 产品将走 [cmd /C chcp 65001 & ...] 兜底（★ 重点嫌疑：2>/dev/null 在 cmd 中非法）")
	}
	psPath, psErr := exec.LookPath("powershell")
	if psErr != nil {
		fmt.Printf("  PowerShell     : ✗ 不在 PATH（%v）→ 产品中 powershell 命令必然失败 → 返回空数组\n", psErr)
	} else {
		fmt.Printf("  PowerShell     : %s\n", psPath)
	}

	// ── 1. 目标目录存在性 ──
	fmt.Println("\n── [2] 目标目录存在性 ──")
	st, statErr := os.Stat(abs)
	if statErr != nil {
		fmt.Printf("  os.Stat: ✗ %v\n", statErr)
		fmt.Println("  → 目录不存在/无权限（产品 handleWorkspace add-folder 会报「目录不存在」；若 browse 直达则返回空数组）")
	} else if !st.IsDir() {
		fmt.Printf("  os.Stat: 不是目录（%s）\n", abs)
	} else {
		fmt.Printf("  os.Stat: ✓ 目录存在\n")
	}

	// ── 2. 链路 A：完整复刻 fs-api browse=1（shell 包装 + powershell + 2>/dev/null）──
	//    命令与 fs-api index.js 完全一致：'powershell ... -EncodedCommand <b64> 2>/dev/null'
	fmt.Println("\n── [3] 链路A：browse=1 完整复刻（shell 包装 + powershell -EncodedCommand + 2>/dev/null）──")
	normDir, normErr := normalizeBrowsePath(abs)
	if normErr != nil {
		fmt.Printf("  路径规范化失败: %v\n", normErr)
	} else {
		b64, ps := buildPSCommand(normDir)
		fmt.Printf("  规范化路径: %s\n", normDir)
		fmt.Printf("  PowerShell 脚本: %s\n", ps)
		cmd := "powershell -NoProfile -NonInteractive -EncodedCommand " + b64 + " 2>/dev/null"
		fmt.Printf("  完整命令: %s\n\n", cmd)
		out, errOut, exitErr := runShellWrapped(cmd, cwd, shellTimeout)
		fmt.Printf("  stdout(%d 字节): %s\n", len(out), preview(out, 240))
		if errOut != "" {
			fmt.Printf("  stderr(%d 字节): %s\n", len(errOut), preview(errOut, 240))
		}
		if exitErr != "" {
			fmt.Printf("  ★ exit错误: %s\n", exitErr)
			fmt.Println("  → 【与用户现象一致】fs-api 检测到 error 后容错 return ok([])：目录列表=空数组、无报错")
		} else {
			fmt.Printf("  exit: 0\n")
			entries, perr := parsePSOutput(out)
			if perr != "" {
				fmt.Printf("  ★ 输出解析: %s\n", perr)
			} else {
				fmt.Printf("  ✓ JSON 解析成功，目录条目数: %d\n", len(entries))
			}
		}
	}

	// ── 3. 链路 B：直接 powershell（绕过 shell 包装，定位包装层/PS 本体问题）──
	fmt.Println("\n── [4] 链路B：直接 powershell -EncodedCommand（参数化，无包装、无 2>/dev/null）──")
	if psErr != nil {
		fmt.Println("  跳过：powershell 不在 PATH")
	} else {
		b64, _ := buildPSCommand(normDir)
		out, errOut, exitErr := runPSDirect(b64, cwd, shellTimeout)
		fmt.Printf("  stdout(%d 字节): %s\n", len(out), preview(out, 240))
		if errOut != "" {
			fmt.Printf("  stderr(%d 字节): %s\n", len(errOut), preview(errOut, 240))
		}
		if exitErr != "" {
			fmt.Printf("  ★ exit错误: %s\n", exitErr)
		} else {
			fmt.Printf("  exit: 0\n")
			entries, perr := parsePSOutput(out)
			if perr != "" {
				fmt.Printf("  ★ 输出解析: %s\n", perr)
			} else {
				fmt.Printf("  ✓ JSON 解析成功，目录条目数: %d\n", len(entries))
			}
		}
	}

	// ── 4. 链路 C：Go 原生（模拟 ctx.fs.readdir 非 browse 分支）──
	fmt.Println("\n── [5] 链路C：Go 原生 os.ReadDir（模拟项目文件树非 browse 分支 ctx.fs.readdir）──")
	names, nErr := listNative(abs)
	if nErr != nil {
		fmt.Printf("  ★ os.ReadDir 失败: %v\n", nErr)
		fmt.Println("  → 目录本身不可读（权限/占用/特殊路径）→ 产品中 ctx.fs.readdir panic → 前端空 catch 吞错 → 文件树为空")
	} else {
		fmt.Printf("  ✓ os.ReadDir 成功，条目数: %d\n", len(names))
		for i, n := range names {
			if i >= 15 {
				fmt.Printf("    ...（共 %d 条，仅显示前 15 条）\n", len(names))
				break
			}
			fmt.Printf("    - %s\n", n)
		}
	}

	// ── 5. 附加探测 ──
	psProbe(cwd)

	// ── 6. 诊断结论 ──
	fmt.Println("\n══════════════════════════════════════════════════════════════")
	fmt.Println(" [诊断结论]")
	fmt.Println("══════════════════════════════════════════════════════════════")
	fmt.Println("请将本程序『完整输出』贴给开发排查。按以下组合定位：")
	fmt.Println()
	fmt.Println(" 1) 链路A 失败 + 链路B 成功  →  shell 包装层问题：")
	fmt.Println("    无 Git Bash 时产品走 cmd /C，『2>/dev/null』在 cmd 中非法（stderr 重定向失败），")
	fmt.Println("    整条命令 exit≠0 → fs-api 容错返回空数组。→ 修复方向：fs-api 按窗口兼容或改用参数化调用")
	fmt.Println()
	fmt.Println(" 2) 链路A/B 均失败 + 链路C 成功 →  PowerShell 环境问题：")
	fmt.Println("    powershell 不在 PATH / 被 AppLocker 拦截 / ConstrainedLanguage 受限语言模式 /")
	fmt.Println("    杀毒软件拦截 / 组策略限制执行 → 检查上方『附加探测』的版本/执行策略/语言模式")
	fmt.Println()
	fmt.Println(" 3) 链路A/B/C 全部成功          →  目录读取链路本身正常：")
	fmt.Println("    问题可能在前端或工作区根配置（/api/health 返回的 workspaceRoot/folders 为空、")
	fmt.Println("    或 workspaceFolderLists 快照与磁盘不符）→ 下一步排查 /api/health 与前端 loadFileTree")
	fmt.Println()
	fmt.Println(" 4) 链路C 也失败                →  目录本身不可读（权限/占位符/网络盘/符号链接死循环），")
	fmt.Println("    目录浏览与文件树都会表现为空")
	fmt.Println()
	fmt.Println(" 5) 链路B 成功但输出解析失败     →  输出编码问题（GBK/UTF-16 混入），检查 stdout 字节预览")
}
