// 命令行输出编码探测解码：防 Windows 乱码。
//
// Windows 下经 cmd /C 启动的子进程，其 stdout 编码不可控：
//   - Go/Node/新工具 → UTF-8（chcp 65001 后）；
//   - 旧工具/Python2/某些内建命令 → 系统 ANSI 码页（中文系统为 GBK）；
//   - 部分程序无视 chcp，始终按 OEM/ANSI 输出。
//
// 直接把字节 string(b) 会按 UTF-8 解释，GBK 字节流产生乱码（菱形问号）。
// 策略（Windows 中文环境主场景）：
//   1. 字节流是合法 UTF-8（含纯 ASCII）→ 原样按 UTF-8；
//   2. 否则按 GBK 解码（GBK 双字节高位 0x81~0xFE 通常构成非法 UTF-8 序列，
//      探测不会误伤合法 UTF-8 输出）；
//   3. GBK 解码失败（极小概率）→ 回退原始字节。

package agent

import (
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
)

// decodeCmdOutput 命令行输出字节流 → 正确编码字符串（UTF-8 优先，GBK 兜底）。
// 所有 shell/进程输出转字符串的统一入口：runShellWithTimeout（bash/run_command/
// run_background/read_output/ctx.bash.exec）、bridge_exec、debug_tools、git 工具、
// bugdetect 等——一处修复，全链路受益。
func decodeCmdOutput(b []byte) string {
	if utf8.Valid(b) {
		return string(b)
	}
	if s, err := simplifiedchinese.GBK.NewDecoder().String(string(b)); err == nil {
		return s
	}
	return string(b) // 解码失败回退原始字节（至少不更糟）
}
