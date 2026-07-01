# Tasks

- [x] Task 1: 回退 CreateWindow 的 contentScale 除法
  - [x] 删除 `monCsX, monCsY := glfw.GetPrimaryMonitor().GetContentScale()` 及 `logicalCreateW/logicalCreateH` 中间变量
  - [x] `CreateWindow` 直接传入 `cfg.Width, cfg.Height`
  - [x] 更新注释：`cfg.Width/Height` 为 CSS 像素（逻辑像素），DPI 感知后帧缓冲自动放大

- [x] Task 2: 修正 logicalW/logicalH 与兜底逻辑
  - [x] `logicalW := cfg.Width; logicalH := cfg.Height`（恢复为 CSS 像素，与配置一致）
  - [x] 兜底判断改用 `cfg.Width`：`if fbW == cfg.Width && gcsX > 1.0 { fbW = int(float32(cfg.Width)*gcsX + 0.5) }`
  - [x] `csX := float32(fbW) / float32(logicalW)`（如 1100/880=1.25）

- [x] Task 3: 修正 Centered 居中逻辑
  - [x] `win.SetPos((mode.Width-cfg.Width)/2, (mode.Height-cfg.Height)/2)`（恢复用 cfg.Width，因为 SetPos 和 GetVideoMode 都是 CSS 像素）

- [x] Task 4: 验证 SetSizeCallback / SetFramebufferSizeCallback
  - [x] 确认两个回调中 `a.width = actualFbW`（物理像素）、`a.logicalW = width/ww`（CSS 像素）已正确（前一 会话已修复，本次不改）

- [x] Task 5: 编译验证
  - [x] `CGO_ENABLED=1 go build -tags skia ./app/...` 通过
  - [x] `CGO_ENABLED=1 go build -tags skia ./examples/html_layout/...` 通过

- [x] Task 6: 运行验证窗口物理大小自动放大
  - [x] 运行 `GWUI_AUTO_TEST=1 go run -tags skia ./examples/html_layout`
  - [x] 临时日志确认：`cfg=880x900 fb=1100x1125 logical=880x900 cs=1.250 a.width=1100`
  - [x] 窗口物理大小 = cfg.Width × contentScale（自动放大）
  - [x] 验证后删除临时日志

# Task Dependencies
- Task 2 依赖 Task 1（logicalW 计算依赖 CreateWindow 参数）
- Task 3 独立
- Task 5 依赖 Task 1-4
- Task 6 依赖 Task 5
