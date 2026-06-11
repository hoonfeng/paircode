// Package chatpanel 是对话面板（复刻参考 ChatPanel）。单例跨重建存活（同 editorpanel/filetreepanel 模式）。
//
//go:build windows

package chatpanel

import (
	"encoding/json"
	"strings"

	"github.com/hoonfeng/paircode/cmd/companion/ui/state"
	"github.com/hoonfeng/goui/pkg/widget"
)

type AgentBridge interface {
	IsRunning() bool
	RunningThread() *state.Thread
	Start(task string)
	Stop()
	ResetForNewRoot()
	ResolveAsk(answer string)
	ResolveApproval(callID string, ok bool)
}

// NewBridge 由 main 注入（components.go init），用于在 send/regenerate 时懒建 bridge。
var NewBridge func(cs *ChatState) AgentBridge

// TheState 是对话面板的持久状态（包级单例，跨 relayout 存活）。
var TheState = &ChatState{Store: state.NewChatStore(), AutoReview: true, HoveredMsg: -1, ShowThreads: true}

// ChatPanel 对话面板。
type ChatPanel struct{ widget.StatefulWidget }

func (c *ChatPanel) CreateState() widget.State { return TheState }

// Area 返回对话面板组件。
func Area() widget.Widget { return &ChatPanel{} }

// Reset 复位对话面板单例（测试用）。
func Reset() {
	TheState = &ChatState{Store: state.NewChatStore(), AutoReview: true, HoveredMsg: -1, ShowThreads: true}
}

// AttachmentContext 把附件内容拼成给 agent 的上下文段（chat_test.go 用）。
func AttachmentContext(atts []string) string { return attachmentContext(atts) }

// AttachmentNames 附件文件名逗号分隔（chat_test.go 用）。
func AttachmentNames(atts []string) string { return attachmentNames(atts) }

// MsgMatches 消息是否命中搜索词（测试用）。
func MsgMatches(m state.Message, q string) bool { return msgMatches(m, q) }

// ArgPreview 截取工具参数 JSON 为一行预览（agent_bridge.go 用）。
func ArgPreview(argsJSON string) string { return argPreview(argsJSON) }

// ResolveApprovalUI 把按钮点击路由到单例对话面板的 bridge（测试/UI 用）。
func ResolveApprovalUI(callID string, ok bool) {
	if TheState != nil && TheState.Bridge != nil {
		TheState.Bridge.ResolveApproval(callID, ok)
	}
}

// ParseAsk 从 ask_user 工具参数解析出问答卡数据（测试用）。
func ParseAsk(argsJSON string) *pendingAsk {
	var pa pendingAsk
	if err := json.Unmarshal([]byte(argsJSON), &pa); err != nil || strings.TrimSpace(pa.Question) == "" {
		return &pendingAsk{Question: "（Agent 提问，但问题为空）"}
	}
	return &pa
}
