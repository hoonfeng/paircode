// text_utils.go — 包内通用文本辅助。
//
// ★ 2026-09-21：truncStr 原定义在 autonomous_controller.go（旧自主模式辅助文件，
// 已随旧自主模式删除）——它被日志/通知/诊断多处使用，故迁到本文件独立保留。
package agent

// truncStr 截断字符串到 max 字节（超出加省略号）。
// 注：按字节截断（沿用原实现，用于日志/通知的粗略截断，不参与落盘或 LLM 输入）。
func truncStr(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
