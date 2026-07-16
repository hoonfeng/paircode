// Package uiapi 提供后端代码与 UI 层之间的抽象接口。
// 后端代码（core/langsrv/bridge 等）需要显示消息/对话框时，通过本包的函数变量调用，
// 由 main 在启动时注入实际实现（GWui 版）。这样后端不直接依赖任何 UI 框架。
package uiapi

// MessageKind 消息类型（影响图标/颜色）。
type MessageKind int

const (
	KindInfo    MessageKind = iota // 信息提示
	KindSuccess                    // 成功提示
	KindWarning                    // 警告提示
	KindError                      // 错误提示
)

// 以下函数变量由 main 在启动时注入。未注入时为 no-op（后端测试不阻塞）。

var MessageFunc func(text string, kind MessageKind) // 显示 toast 消息

var ShowConfirmFunc func(title, body string, kind MessageKind, onConfirm func()) // 显示确认对话框

var ShowDialogFunc func(title string, width float32, content interface{}, footer interface{}) int // 显示自定义对话框，返回 overlay id

var HideOverlayFunc func(id int) // 隐藏浮层

var MarkDirtyFunc func() // 标记需要重绘

var RequestFrameFunc func() // 请求下一帧刷新

// MessageError 显示错误消息。
func MessageError(text string) {
	if MessageFunc != nil {
		MessageFunc(text, KindError)
	}
}

// MessageSuccess 显示成功消息。
func MessageSuccess(text string) {
	if MessageFunc != nil {
		MessageFunc(text, KindSuccess)
	}
}

// MessageInfo 显示信息消息。
func MessageInfo(text string) {
	if MessageFunc != nil {
		MessageFunc(text, KindInfo)
	}
}

// MessageWarning 显示警告消息。
func MessageWarning(text string) {
	if MessageFunc != nil {
		MessageFunc(text, KindWarning)
	}
}

// ShowConfirm 显示确认对话框。
func ShowConfirm(title, body string, kind MessageKind, onConfirm func()) {
	if ShowConfirmFunc != nil {
		ShowConfirmFunc(title, body, kind, onConfirm)
	}
}

// ShowDialog 显示自定义对话框。
func ShowDialog(title string, width float32, content interface{}, footer interface{}) int {
	if ShowDialogFunc != nil {
		return ShowDialogFunc(title, width, content, footer)
	}
	return 0
}

// HideOverlay 隐藏浮层。
func HideOverlay(id int) {
	if HideOverlayFunc != nil {
		HideOverlayFunc(id)
	}
}

// MarkDirty 标记需要重布局+重绘。
func MarkDirty() {
	if MarkDirtyFunc != nil {
		MarkDirtyFunc()
	}
}

// RequestFrame 请求下一帧刷新。
func RequestFrame() {
	if RequestFrameFunc != nil {
		RequestFrameFunc()
	}
}
