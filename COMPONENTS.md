# goui 组件清单

goui 提供 ~75 个组件（对标 Element Plus，覆盖 ~94%），其中 **86 个可声明式使用**（JSON 配置 / `BuildFromSpec`）。下表按类别列出声明式 `type` 名与简述。

声明式用法：

```go
ui, _ := widget.LoadConfig([]byte(`{"type":"Button","text":"OK","events":{"click":"onOk"}}`), handlers)
```

样式走 `style`（CSS 式）、组件特有属性走 `props`、事件走 `events`、子组件走 `children`。自定义组件用 `widget.RegisterComponent("Name", factory)` 注册。

---

## 布局 / 容器
| type | 说明 |
|---|---|
| `Column` / `VBox` | 竖向 Flex |
| `Row` / `HBox` | 横向 Flex |
| `Div` / `Container` | 通用容器（支持 CSS 样式、`:hover`/`:focus`/`:active` 伪类、绝对定位） |
| `Section` / `Spacer` | 区块 / 弹性占位 |
| `Space` | 等距排列子项 |
| `Stack` / `Positioned` | 层叠 / 绝对定位 |
| `Splitter` | 可拖动分隔（左右） |
| `ScrollView` | 滚动容器（滚动条 / 回顶 / 吸顶） |
| `Card` | 卡片 |
| `Divider` / `Separator` | 分割线 |
| `Affix` | 吸顶固钉 |
| `Watermark` | 水印 |

## 文本 / 编辑器
| type | 说明 |
|---|---|
| `Text` / `H1`–`H4` / `P` / `Small` / `Label` | 文本与标题 |
| `Markdown` | Markdown 只读渲染 |
| `RichText` | 富文本 WYSIWYG（粗斜下划删除线 / 字号 / 颜色 / 对齐 / 图片 / 撤销重做 / 清除格式） |
| `CodeEditor` | 代码编辑器（语法高亮 / 多光标 / 折叠 / minimap / gopls） |

## 表单 / 输入
| type | 说明 |
|---|---|
| `Button` | 按钮 |
| `Input` / `Textarea` | 单行 / 多行输入 |
| `Checkbox` / `Switch` / `RadioButton` | 复选 / 开关 / 单选 |
| `Rate` / `Slider` / `InputNumber` | 评分 / 滑块 / 数字步进 |
| `Select` / `SelectV2` | 下拉（多选 / 搜索 / 清除；V2 虚拟列表） |
| `Cascader` / `TreeSelect` | 级联 / 树选择 |
| `DatePicker` / `DateTimePicker` / `TimePicker` / `TimeSelect` | 日期时间选择 |
| `ColorPicker` | 取色器（HSV） |
| `Autocomplete` / `Mention` | 自动补全 / @ 提及 |
| `InputTag` | 标签输入 |
| `Upload` | 上传 |
| `Form` | 表单（校验 / 联动 / 动态增删） |

## 数据展示
| type | 说明 |
|---|---|
| `Tag` / `Badge` / `Avatar` | 标签 / 角标 / 头像 |
| `Image` | 图片（PNG/JPEG/GIF/WebP/BMP/SVG，GIF 动画） |
| `ProgressBar` | 进度条 |
| `Table` | 表格（排序 / 多选 / 可展开 / 固定列 / 固定表头 / slot 渲染） |
| `TreeV2` | 树（虚拟列表） |
| `Collapse` | 折叠面板 |
| `Descriptions` / `Statistic` / `Timeline` | 描述列表 / 统计 / 时间线 |
| `Calendar` / `Carousel` / `Skeleton` | 月历 / 走马灯 / 骨架屏 |
| `Empty` | 空状态 |

## 导航
| type | 说明 |
|---|---|
| `Menu` | 菜单（级联子菜单） |
| `Tabs` / `Steps` / `Breadcrumb` | 标签页 / 步骤条 / 面包屑 |
| `Dropdown` / `Anchor` / `PageHeader` | 下拉菜单 / 锚点 / 页头 |

## 反馈
| type | 说明 |
|---|---|
| `Alert` | 警告提示 |
| `Dialog` / `Drawer` | 对话框 / 抽屉 |
| `Loading` / `Result` | 加载遮罩 / 结果页 |
| `Popover` / `Tooltip` | 气泡卡片 / 文字提示 |

## 其他
| type | 说明 |
|---|---|
| `ConfigProvider` | 局部主题作用域（子树差异化配色） |
| `StructEditor` / `CodeWorkbench` | 代码表格化编辑器 / 表格⇄代码切换 |

---

## 命令式 API（不走声明式，直接调用）

- 消息 / 通知 / 弹框：`MessageSuccess/Error/Warning/Info`、`NotifySuccess(...)`、`ShowAlert/ShowConfirm`、`ShowMessageWith/ShowNotificationWith`（可配位置 / 配色）。
- 模态：`ShowDialog/ShowDrawer/ShowLoading/HideLoading`。
- 浮层：`ShowTooltip/ShowPopover/ShowContextMenu`、`ShowTour(steps...)`。

## 不适合声明式的（需 Go 代码）

`VirtualList` / `InfiniteScroll`（`render func(index)` 回调）、`Backtop`（需运行时 `*ScrollView`）、`OverlayHost`（基建）——本质是 render-func / 运行时依赖型。

## 主题 / 配色 / i18n

```go
widget.SetTheme(widget.DarkTheme())              // 深色模式
widget.SetPrimaryColor(green)                     // 换主色
widget.NewConfigProvider(x).WithTheme(dark)       // 局部差异化
i18n.SetLocale("en")                              // 切换语言（运行时全树刷新）
i18n.LoadLocaleFile("locales/")                   // 加载语言包
```

## CSS 伪类（:hover / :focus / :active）

`Div` / `Container` 支持状态样式：进入对应状态时，其中已设的视觉属性（背景 / 边框颜色+宽度 / 圆角 / 阴影 / 透明度）覆盖基础样式，优先级 `:hover < :focus < :active`。边框可「从无到有」，方便做 focus ring。

```go
widget.Div(widget.Style{
    BackgroundColor: types.ColorRef(235, 238, 245), BorderRadius: 8,
    Hover:  &widget.Style{BackgroundColor: types.ColorRef(64, 158, 255)},            // 悬停变蓝
    Focus:  &widget.Style{BorderColor: types.ColorRef(64, 158, 255), BorderWidth: 2}, // 聚焦描边
    Active: &widget.Style{BackgroundColor: types.ColorRef(48, 120, 200)},            // 按下变深
}, child)
```

JSON 声明式（嵌套对象）：

```json
{"type":"Div","style":{"backgroundColor":"#ebeef5","borderRadius":8,
  "hover":{"backgroundColor":"#409eff"},
  "focus":{"borderColor":"#409eff","borderWidth":2},
  "active":{"backgroundColor":"#3078c8"}}}
```

**组件级焦点态**也接入了同一套伪类：`Input`/`Textarea` 聚焦/悬停边框、`Button` 聚焦环，以及**触发器/选择类**（`Select`/`SelectV2`/`DatePicker`/`TimePicker`/`TreeSelect`/`Cascader`）和 `Checkbox`/`Radio` 的悬停/激活边框色，都可按组件覆盖——链式 `input.WithFocusColor(c)` / `select.SetHoverBorderColor(c)`，或样式 `Style{Focus: &Style{BorderColor: ...}}`，或 JSON `{"type":"Input","style":{"focus":{"borderColor":"#f00"}}}`。复用这些的组件（`TimeSelect`/`Autocomplete`/`Mention`/`DateTimePicker` 等）随之自动继承。

```go
widget.NewInput("用户名", onChange).
    WithFocusColor(types.ColorRef(64, 158, 255)).  // 聚焦蓝边
    WithHoverColor(types.ColorRef(192, 196, 204))  // 悬停灰边
```

## 窗口能力（Windows）

透明窗口、系统托盘、托盘菜单、多窗口——均为 Win32 实现，由 `app.Application` 提供：

```go
a := app.NewApplication()
a.SetRootWidget(root)

a.Ready = func() {
    // 系统托盘 + 右键菜单
    id := a.AddTray("提示文字", "icon.ico", func() { /* 左键单击 */ })
    a.SetTrayMenu(id, []app.TrayMenuItem{
        {ID: 1, Label: "打开面板"},
        {Separator: true},
        {ID: 9, Label: "退出"},
    }, func(cmd int) { /* 菜单点击 */ })

    // 多窗口：副窗口独立渲染管线
    a.OpenWindow(window.WindowConfig{Title: "面板", Width: 360, Height: 240}, panelRoot)
}

// 透明窗口：整窗 95% 不透明
a.Run(app.Config{Title: "App", Width: 640, Height: 440, Opacity: 0.95})
a.SetOpacity(0.7) // 运行时调整透明度
```

| 能力 | API |
|---|---|
| 透明窗口 | `Config.Opacity` / `app.SetOpacity(0~1)`（整窗 alpha） |
| 系统托盘 | `app.AddTray(tooltip, iconPath, onLeftClick)` / `SetTrayTooltip` / `RemoveTray` |
| 托盘气泡通知 | `app.ShowTrayBalloon(trayID, title, text, level)`（level: 0 信息 / 1 警告 / 2 错误；真·系统通知） |
| 托盘菜单 | `app.SetTrayMenu(id, []TrayMenuItem{...}, onSelect)` |
| 多窗口 | `app.OpenWindow(cfg, root)` → `*SubWindow`（独立 pipeline；`.Close()`） |
| 无边框 / 自绘标题栏 | `Config.Borderless=true` 隐藏系统标题栏；`app.SetTitleBar(height, rightExclude)` 声明命中区→系统接管拖动 / **双击最大化** / Aero Snap；`app.Minimize()` / `ToggleMaximize()` / `Close()` 接自绘按钮（`widget.WindowDragHandle` 为不配命中区时的简易拖动备选） |
| 圆角 / 阴影 | `app.EnableWindowEffects()`（DWM 投影阴影 + Win11 圆角；无边框窗口补回系统阴影，旧系统自动忽略圆角） |
| 副窗口同款 | `*SubWindow` 也有 `SetTitleBar` / `EnableEffects` / `Minimize` / `ToggleMaximize` / `IsMaximized`，可做无边框自绘标题栏 + 圆角阴影 |

完整示例见 `examples/windowfeatures`。副窗口定位为工具面板/第二视图（基础鼠标/键盘交互；复杂拖放/IME/跨窗口焦点以主窗口为主）。

