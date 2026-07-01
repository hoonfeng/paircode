# Gio 渲染后端替换 Spec

## Why

当前 GWui 渲染后端硬绑定 goskia（Skia 的 cgo 绑定），存在严重的性能开销：

1. **cgo 边界开销**：每次绘图调用（DrawRect/DrawText/Save/Restore/ClipRect 等）都跨 cgo 边界，一帧布局树渲染触发数千次 cgo 调用
2. **GC 压力**：每个 skia 对象用 `runtime.SetFinalizer` 管理 C 资源，释放延迟；每次调用 `runtime.KeepAlive` 带来额外开销
3. **数据拷贝**：DrawText 把 string 转 []byte 传指针，字体数据用 `sk_data_new_with_copy` 拷贝
4. **GL 桥接三跳**：Skia 通过 C→Go→C 回调获取 GL 函数地址
5. **运行时依赖**：需要 libSkiaSharp.dll 在 PATH 或 exe 同目录，部署不便
6. **跨平台 IME 不一致**：Windows 有完整 IME（Win32 子类化），Linux/macOS 是空桩

Gio（gioui.org）在 Windows 上**完全零 cgo**（通过 syscall 调用 Win32/D3D11/libGLESv2），使用序列化 op.Ops 批量提交 GPU 命令，内置完整 IME 支持（Windows IMM + macOS NSTextInputClient），是更优的渲染后端选择。

## What Changes

### 渲染后端抽象层（新增 `backend/` 包）
- 新增 `backend/` 包定义后端无关的绘图接口（Surface/Canvas/Paint/Path/Font/Typeface/Image/GPUContext）
- `css.Value.Color` 类型从 `skia.Color` 改为后端无关的 `Color`（uint32 别名），消除 css 包对 skia 的硬依赖
- `render.Renderer` 改为依赖 `backend.` 接口而非具体 skia 类型

### Gio 后端实现（新增 `backend/gio/` 包）
- 实现 `backend.` 接口的 Gio 版本：将 Skia 立即模式调用转为 Gio 保留模式 op.Ops 录制
- 字体/文本：用 `gioui.org/text.Shaper` + `gioui.org/font/opentype` 替代 skia.Typeface/Font，重写 CJK/emoji 字形回退
- 路径/画笔：用 `gioui.org/op/clip.Path` + `gioui.org/op/paint.PaintOp` 替代 skia.Path/Paint
- 图片：用 `gioui.org/op/paint.ImageOp` 替代 skia.Image
- 滤镜：blur/grayscale 用 Gio 离屏渲染 + ImageOp 实现

### Skia 后端适配（新增 `backend/skia/` 包）
- 将现有 goskia 调用包装为 `backend.` 接口实现，保留向后兼容
- 通过构建标签或初始化注入选择后端

### 窗口管理重构（`app/` 包）
- **BREAKING**：删除 GLFW 依赖，改用 `gioui.org/app.Window` 管理窗口
- 窗口创建、尺寸变化、最大化/最小化/全屏、DPI 缩放全部委托 Gio
- 事件循环从 `glfw.WaitEventsTimeout` 改为 Gio 的 `window.Events()` channel 消费

### 事件系统适配（`app/` + `event/` 包）
- Gio 事件（`pointer.Event`/`key.Event`/`clipboard.Event`）转换为 GWui 的 `event.MouseEvent`/`event.KeyboardEvent` 等
- 保留 GWui 的 DOM 三阶段事件分发（捕获→目标→冒泡）

### IME 处理（`app/` 包）
- **BREAKING**：删除 `ime_windows.go`/`ime_linux.go`/`ime_darwin.go`/`ime_other.go` 的 Win32 子类化代码
- 改用 Gio 内置 IME：通过 `key.Editor` 接口接收 `key.EditEvent`/`key.CompositionEvent`/`key.SnippetEvent`
- `layout.Box.CompositionText`/`Composing`/`CursorScreenX/Y` 字段保留，由 Gio IME 事件驱动更新

### 离屏渲染（`gwui.go`）
- `gwui.Render(doc, w, h)` 从 `skia.NewRasterSurfaceN32Premul` 改为 Gio headless 模式（`gioui.org/app/headless`）

### 测试代码适配
- `render/render_test.go`：移除对 `skia.Color.R()` 等方法的直接调用，改用后端无关接口
- `css/css_test.go`：颜色比较从 `skia.Color` 改为 `backend.Color`
- `layout/layout_test.go`：无 skia 依赖，无需改动
- `examples/integration_test/main.go`：渲染调用路径不变（通过 `gwui.Render`），验证 PNG 输出
- 新增 `backend/gio/gio_test.go`：Gio 后端单元测试
- 新增 `backend/skia/skia_test.go`：Skia 后端适配测试

## Impact

- **Affected code**:
  - `f:\syproject\GWui\css\css.go` — `Value.Color` 类型从 `skia.Color` 改为 `backend.Color`
  - `f:\syproject\GWui\render\render.go` — Renderer 重写为依赖 backend 接口
  - `f:\syproject\GWui\render\render_overlay.go` / `render_svg.go` — 同步适配
  - `f:\syproject\GWui\app\app.go` — 删除 GLFW + skia GL，改用 Gio Window
  - `f:\syproject\GWui\app\window_api.go` — 改为 Gio Window 适配
  - `f:\syproject\GWui\app\ime_*.go` — 删除 Win32 子类化，改用 Gio IME
  - `f:\syproject\GWui\gwui.go` — 离屏渲染改用 Gio headless
  - `f:\syproject\GWui\go.mod` — 新增 gioui.org 依赖，goskia 改为可选
  - `f:\syproject\gou-ide\go.mod` — 间接依赖更新
  - 所有测试文件 — 颜色/渲染相关断言适配

- **Affected specs**: CSS 系统（颜色类型解耦）、布局系统（无变化）、渲染系统（后端抽象）、窗口管理（Gio 替代 GLFW）、事件系统（事件源适配）、IME（Gio 内置）

## ADDED Requirements

### Requirement: 渲染后端抽象层
系统 SHALL 提供后端无关的绘图接口层（`backend` 包），使渲染器不直接依赖具体图形库。

#### Scenario: 后端接口定义
- **WHEN** 开发者查看 `backend/` 包
- **THEN** 应找到 Surface、Canvas、Paint、Path、Font、Typeface、Image、GPUContext 等接口定义
- **AND** 这些接口不引用任何具体图形库（skia/gio）的类型

#### Scenario: 颜色类型解耦
- **WHEN** 查看 `css.Value` 结构体
- **THEN** `Color` 字段类型应为 `backend.Color`（uint32 别名），而非 `skia.Color`
- **AND** `backend.Color` 提供 `R()/G()/B()/A()` 方法返回 uint8

#### Scenario: 后端选择
- **WHEN** 应用启动时
- **THEN** 可通过构建标签（`-tags gio` / `-tags skia`）或初始化函数选择渲染后端
- **AND** 默认后端为 Gio

### Requirement: Gio 渲染后端实现
系统 SHALL 提供 Gio 后端实现（`backend/gio` 包），将 GWui 的绘图调用转为 Gio op.Ops。

#### Scenario: 立即模式转保留模式
- **WHEN** Renderer 调用 `canvas.DrawRect(rect, paint)`
- **THEN** Gio 后端将此调用录制为 op.Ops 序列（clip.Rect + paint.Fill）
- **AND** 帧结束时批量提交到 GPU

#### Scenario: 文本渲染与字形回退
- **WHEN** 渲染包含 CJK + emoji 的混合文本
- **THEN** Gio 后端使用 `text.Shaper` 进行文本 shaping
- **AND** 通过字体集合（FontCollection）实现字形回退
- **AND** 文本测量结果（宽度）与原 Skia 实现误差 < 2px

#### Scenario: 离屏渲染
- **WHEN** 调用 `gwui.Render(doc, w, h)` 生成 PNG
- **THEN** 使用 Gio headless 模式渲染
- **AND** 输出 PNG 字节流与 Skia 后端视觉一致（像素差异 < 5%）

### Requirement: Gio 窗口管理
系统 SHALL 使用 `gioui.org/app.Window` 替代 GLFW 进行窗口管理。

#### Scenario: 窗口创建
- **WHEN** 调用 `app.New(doc, Config{Title, Width, Height, ...})`
- **THEN** 创建 Gio `app.Window` 并配置标题、尺寸、最小尺寸、可缩放
- **AND** 窗口居中显示

#### Scenario: 尺寸变化处理
- **WHEN** 用户拖拽调整窗口大小
- **THEN** Gio 发送 `Config` 事件，app 读取新尺寸
- **AND** 触发 `event.Resize` 事件分发
- **AND** 重新布局并渲染

#### Scenario: 最大化/最小化/还原
- **WHEN** 用户点击最大化按钮或调用 `app.Maximize()`
- **THEN** 窗口切换到最大化状态
- **AND** 触发布局重计算（利用全屏空间）
- **WHEN** 调用 `app.Restore()`
- **THEN** 窗口恢复到之前尺寸

#### Scenario: DPI 缩放
- **WHEN** 窗口在不同 DPI 显示器间移动
- **THEN** Gio 发送 `Config` 事件含新 `unit.Metric`
- **AND** 渲染按新 DPI 缩放
- **AND** 触发 `OnDPIChange` 回调

### Requirement: Gio 事件适配
系统 SHALL 将 Gio 事件转换为 GWui 事件系统的事件类型。

#### Scenario: 鼠标事件
- **WHEN** Gio 发送 `pointer.Event`（Press/Release/Move/Drag）
- **THEN** 转换为 `event.MouseEvent`（含 X/Y/Button/CtrlKey/ShiftKey）
- **AND** 通过 DOM 三阶段分发（捕获→目标→冒泡）

#### Scenario: 键盘事件
- **WHEN** Gio 发送 `key.Event`（Press/Release）
- **THEN** 转换为 `event.KeyboardEvent`（含 Key/Char/Ctrl/Shift）
- **AND** 快捷键管理器（shortcutManager）检查匹配

#### Scenario: 滚轮事件
- **WHEN** Gio 发送 `pointer.Event`（Scroll）
- **THEN** 转换为 `event.WheelEvent`（含 DeltaY）
- **AND** 分发到鼠标悬停的滚动容器

### Requirement: Gio IME 集成
系统 SHALL 使用 Gio 内置 IME 支持替代 Win32 子类化代码。

#### Scenario: 中文输入法组合
- **WHEN** 用户在输入框中用中文输入法打字
- **THEN** Gio 发送 `key.CompositionEvent`（组合中文本）
- **AND** 更新 `layout.Box.CompositionText` 字段
- **AND** 渲染器绘制拼音预览 + 下划线

#### Scenario: 输入法确认
- **WHEN** 用户选择候选词确认输入
- **THEN** Gio 发送 `key.EditEvent`（确认文本）
- **AND** 文本写入 DOM 节点
- **AND** 清除 `CompositionText`

#### Scenario: IME 候选窗口定位
- **WHEN** 组合状态变化
- **THEN** 通过 Gio 的 `key.SnippetCmd` + `key.SelectionCmd` 更新光标位置
- **AND** Gio 自动定位候选窗口到光标处

#### Scenario: 跨平台 IME
- **WHEN** 在 Windows 上运行
- **THEN** IME 通过 Win32 IMM 工作（Gio 内部处理）
- **WHEN** 在 macOS 上运行
- **THEN** IME 通过 NSTextInputClient 工作（Gio 内部处理）

### Requirement: Skia 后端向后兼容
系统 SHALL 保留 Skia 后端作为可选实现，确保向后兼容。

#### Scenario: Skia 后端可用
- **WHEN** 使用 `-tags skia` 构建标签编译
- **THEN** 系统使用 goskia 后端渲染
- **AND** 所有现有功能正常工作

#### Scenario: 后端切换无代码改动
- **WHEN** 在 Gio 和 Skia 后端间切换
- **THEN** GWui 上层 API（dom/css/layout/event/component）无需修改
- **AND** gou-ide 应用代码无需修改

## MODIFIED Requirements

### Requirement: 渲染器架构
渲染器（`render.Renderer`）SHALL 依赖 `backend.` 接口而非具体 skia 类型。Renderer 结构体的 `surface`/`canvas`/`typeface`/`font` 等字段类型改为 backend 接口。`renderBox` 递归函数的绘图调用通过 backend 接口发出，具体后端实现负责录制/执行。

### Requirement: 应用窗口管理
`app.App` SHALL 使用 `gioui.org/app.Window` 替代 `*glfw.Window`。窗口创建、事件循环、回调注册全部委托 Gio。`app.Config` 结构体保持不变（Title/Width/Height/FontPath/Resizable/Centered/MinWidth/MinHeight），内部映射到 Gio `app.Option`。

### Requirement: IME 处理
IME 处理 SHALL 通过 Gio 的 `key.Editor` 接口实现，而非 Win32 子类化。删除 `ime_windows.go`/`ime_linux.go`/`ime_darwin.go`/`ime_other.go` 中的平台特定代码，改为统一的 Gio IME 事件处理。`layout.Box.CompositionText`/`Composing`/`CursorScreenX/Y` 字段保留。

## REMOVED Requirements

### Requirement: GLFW 窗口管理
**Reason**: Gio 内置跨平台窗口管理，GLFW 成为冗余依赖
**Migration**: 所有 GLFW 回调（SetFramebufferSizeCallback/SetCursorPosCallback 等）改为 Gio 事件 channel 消费

### Requirement: Win32 IME 子类化
**Reason**: Gio 内置 IME 支持（Windows IMM + macOS NSTextInputClient），Win32 子类化代码冗余
**Migration**: `subclassGLFWWindow`/`imeWndProc`/`handleIMEComposition` 等函数删除，IME 事件改为从 Gio `key.CompositionEvent`/`key.EditEvent` 获取

### Requirement: 手动 GL 上下文管理
**Reason**: Gio 自管理 GPU 上下文（D3D11/OpenGL/Metal/Vulkan），无需手动创建 GL context
**Migration**: `skia.NewGLInterface`/`skia.NewGLContext`/`skia.NewGPUSurfaceFromFBO` 调用删除，Gio 自动处理
