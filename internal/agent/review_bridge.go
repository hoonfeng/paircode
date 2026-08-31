package agent

// review_bridge.go — 会话级审核模式桥（★ 2026-08-31）
//
// 背景：前端工具栏切换审核模式（cycleReviewMode → PUT /tools/review）此前
// 只写工作区配置（SaveWorkspaceReviewConfig），无会话概念——同工作区所有
// 会话共享一个审核模式，且用户选择不随会话隔离。
// 现在：handler 支持 convId 会话级读写——会话元数据记录（ConversationMeta.
// ReviewMode，持久化）→ 建 loop 时最高优先（见 web_server buildWebLoopOpts）。
// 会话未设置时回落工作区配置（原路径不变）。
//
// 桥由 web 层启动时注入（web_server.go init()，同 SetConvModelLookup 模式）：
// agent 包不直接依赖 SessionManager，保持与 provider_factory 相同的解耦。

import "sync"

var (
	convReviewMu     sync.RWMutex
	convReviewGetFn  func(convID, wsRoot string) (string, error)
	convReviewApplyFn func(convID, wsRoot, mode string) error
)

// SetConvReviewBridge 注入会话级审核模式读写桥（web 层启动时调用一次）。
func SetConvReviewBridge(
	getFn func(convID, wsRoot string) (string, error),
	applyFn func(convID, wsRoot, mode string) error,
) {
	convReviewMu.Lock()
	defer convReviewMu.Unlock()
	convReviewGetFn = getFn
	convReviewApplyFn = applyFn
}

// LookupConvReview 查询会话级审核模式（""=未设置；未注入桥时返回 ""）。
func LookupConvReview(convID, wsRoot string) (string, error) {
	if convID == "" {
		return "", nil
	}
	convReviewMu.RLock()
	fn := convReviewGetFn
	convReviewMu.RUnlock()
	if fn == nil {
		return "", nil
	}
	return fn(convID, wsRoot)
}

// ApplyConvReview 应用会话级审核模式：持久化（会话元数据）+ 实时更新运行中 Loop。
// 模式非空时生效；未注入桥时返回 nil（等价未设置，回落工作区/全局）。
func ApplyConvReview(convID, wsRoot, mode string) error {
	if convID == "" || mode == "" {
		return nil
	}
	convReviewMu.RLock()
	fn := convReviewApplyFn
	convReviewMu.RUnlock()
	if fn == nil {
		return nil
	}
	return fn(convID, wsRoot, mode)
}
