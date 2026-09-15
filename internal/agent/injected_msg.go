package agent

import "strings"

// injected_msg.go — 「系统注入消息」识别（2026-09-13 重构，取代原「背景上下文」概念）。
//
// 消息流中存在一类 RoleUser 消息：它不是用户输入，而是系统注入的上下文载体——
// 早期历史压缩摘要、会话交接视图，以及历史遗留（2026-09-04 前）的背景快照。
// 这类消息：
//   - 对 LLM 可见（作为历史/上下文的一部分）；
//   - 不构成「用户任务轮次」（turn/step 统计、任务锚点计算须跳过）；
//   - ★ 不作为「真实历史」返回给前端展示（前端展示 = 用户真实消息 + 助手回复 +
//     工具调用链路；注入物是系统产物，混在对话里语义不清、易被误读为真实指令）。
//
// ★ 历史遗留前缀必须继续识别：旧会话 JSONL 中已落盘大量含旧前缀的消息，
//   漏识别会让 turn 统计/锚点计算把它们当成用户任务轮次（数据兼容，非功能保留）。
//
// 识别入口统一在此：新增注入类型 = 加一个前缀常量 + 一处判断，避免字符串散落各处。

const (
	// injectedHistoryDigestPrefix 早期历史压缩摘要前缀。
	// 归档时写入主 JSONL 首行（供 LLM 感知早期历史被压缩），并对前端隐藏。
	injectedHistoryDigestPrefix = "【历史压缩】"

	// legacyBackgroundSnapshotPrefix 历史遗留的「背景上下文」快照前缀。
	// ★ 2026-09-04 快照同步链已停用、2026-09-13 实现整体移除——此常量仅用于
	//   识别旧数据（旧会话落盘中仍存在这类消息），不承担任何注入功能。
	//   背景上下文实现已全部移除，禁止恢复注入链。
	legacyBackgroundSnapshotPrefix = "【背景上下文·非当前任务】"

	// legacyHistoryArchivePrefix 历史遗留的「历史归档」统计摘要前缀。
	// ★ 2026-09-13 前的归档实现只写一行条数统计（「共 N 条消息：用户 x/助手 y/工具 z」），
	//   无任何内容摘要——即「没压缩就丢弃」。新版改为真压缩摘要（injectedHistoryDigestPrefix）。
	//   此常量仅用于识别旧会话里已落盘的旧摘要行：不识别它会把它当真实用户消息
	//   泄漏到前端展示（实测真实会话 conv_1789220736865534600 即如此）。
	legacyHistoryArchivePrefix = "【历史归档】"
)

// handoffTitle 会话交接视图标题（定义于 handoff.go）——注入消息识别用其一并覆盖。

// isInjectedUserMessage 判断一条消息是否为系统注入物（非用户真实输入）。
// 非 user 角色一律返回 false。
func isInjectedUserMessage(role Role, content string) bool {
	if role != RoleUser {
		return false
	}
	return hasPrefixInjected(content)
}

// hasPrefixInjected 仅按内容前缀判断（供已确认角色为 user 的调用点直接使用）。
func hasPrefixInjected(content string) bool {
	return strings.HasPrefix(content, injectedHistoryDigestPrefix) ||
		strings.HasPrefix(content, legacyHistoryArchivePrefix) ||
		strings.HasPrefix(content, handoffTitle) ||
		strings.HasPrefix(content, legacyBackgroundSnapshotPrefix)
}
