# gou-ide 后端对接分析 & UI 缺失清单

> 2026-06-24

## 一、架构总览

```
main.go
  ├── shell 布局（标题栏/面板/分隔条/状态栏）
  ├── injectUIAPI()        → uiapi 抽象层注入
  ├── injectCoreCallbacks() → core 状态回调注入
  ├── injectPanelCallbacks() → 面板间回调注入
  ├── registerShortcuts()  → 快捷键注册
  │
  ├── ui/       → 面板组件（6个）
  │   ├── chat/chat.go     → 聊天面板 ←→ AgentBridge
  │   ├── editor/editor.go → 代码编辑器 ←→ LSP/Debug
  │   ├── filetree/        → 文件树 ←→ core.Folders
  │   ├── terminal/        → 终端面板 ←→ PTY
  │   ├── git/             → Git 面板 ←→ exec
  │   ├── search/          → 搜索面板
  │   ├── settings/        → 设置面板
  │   └── ctxmenu/         → 右键菜单
  │
  ├── uiapi/   → 后端→UI 抽象接口
  ├── bridge/  → Agent 桥接层
  ├── agent/   → AI Agent 核心
  ├── core/    → 项目/配置/状态管理
  └── langsrv/ → LSP 集成
```

## 二、后端对接状态

### ✅ 已对接

| 接口 | 实现 | 说明 |
|------|------|------|
| `uiapi.MarkDirty` | `theApp.MarkDirty()` | 标记重绘 |
| `uiapi.RequestFrame` | `theApp.RequestFrame()` | 请求帧刷新 |
| `core.OnSyncWorkspace` | 文件树重建 + 面板可见性 | 工作区同步 |
| `core.OnCloseProject` | 面板隐藏 + MarkDirty | 关闭项目 |
| `core.OnPickFolder` | `ctxmenupanel.PickFolder()` | 文件夹选择对话框 |
| `injectPanelCallbacks` | 文件树/编辑器右键菜单全部注入 | 面板间联动 |

### ❌ 未对接（TODO）

| 接口 | 现状 | 影响 |
|------|------|------|
| `uiapi.MessageFunc` | 仅 `log.Println` | Toast 通知不可见 |
| `uiapi.ShowConfirmFunc` | 直接 `onConfirm()` | 确认对话框跳过 |
| `uiapi.ShowDialogFunc` | `nil` | 自定义弹窗不可用 |
| `uiapi.HideOverlayFunc` | `nil` | 弹窗无法关闭 |
| `core.OnShowPrompt` | TODO | 输入对话框不可用 |
| `core.OnShowManager` | TODO | 工作区管理器不可用 |

## 三、UI 缺失功能清单

### 🔴 P0 — 核心缺失（需立即补）

| 功能 | 组件 | 文件 | 说明 |
|------|------|------|------|
| **Toast 通知** | `component.Toast` | `main.go:647` | uiapi.MessageFunc 未连接 Toast |
| **确认对话框** | `component.Modal` | `main.go:651` | uiapi.ShowConfirmFunc 未连接 Modal |
| **输入对话框** | `component.Modal + Input` | `main.go:716` | core.OnShowPrompt 未实现 |
| **浮层管理** | `dom.Overlay` | `uiapi/msg.go` | ShowDialogFunc/HideOverlayFunc 为 nil |

### 🟠 P1 — 交互缺失

| 功能 | 说明 |
|------|------|
| **工作区管理器** | core.OnShowManager 未实现 |
| **设置面板接入** | settingspanel 已创建但未注入 main |
| **搜索面板接入** | searchpanel 已创建但未注入 main |
| **Git 面板接入** | 已创建，左面板切换未接入 |

### 🟡 P2 — 体验优化

| 功能 | 说明 |
|------|------|
| **状态栏更新** | 状态栏已创建但无动态内容 |
| **快捷键显示** | 菜单中无快捷键提示 |
| **面板拖拽切换** | 左/右/底面板内容互相切换 |
| **欢迎页** | 无工作区时显示欢迎页 |
| **编辑器标签右键** | 关闭/保存/复制路径等 |

## 四、修复计划

### 第1步：uiapi 实现（P0）

```go
// main.go — injectUIAPI()
uiapi.MessageFunc = func(text string, kind uiapi.MessageKind) {
    toast := component.NewToast(theDoc)
    switch kind {
    case KindInfo: toast.ShowInfo(text)
    case KindSuccess: toast.ShowSuccess(text)
    case KindWarning: toast.ShowWarning(text)
    case KindError: toast.ShowError(text)
    }
}
uiapi.ShowConfirmFunc = func(title, body string, kind uiapi.MessageKind, onConfirm func()) {
    modal := component.NewModal(theDoc, title, body)
    modal.SetOnConfirm(onConfirm)
    modal.Show()
}
uiapi.ShowDialogFunc = func(title string, w float32, content, footer interface{}) int { ... }
```

### 第2步：输入对话框（P0）

```go
core.OnShowPrompt = func(title, initial string, onOk func(string)) {
    // 弹出 Modal 内含 Input
    modal := component.NewModal(theDoc, title, "")
    input := component.NewInput(theDoc, initial)
    // ... 确认时调用 onOk(input.Value())
}
```

### 第3步：面板接入（P1）

将 settings/search/git 面板接入 Shell 的左/右面板切换逻辑
