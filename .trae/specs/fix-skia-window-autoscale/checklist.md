# Checklist

- [x] `CreateWindow` 直接传入 `cfg.Width, cfg.Height`（不再除以显示器 contentScale）
- [x] `logicalW = cfg.Width, logicalH = cfg.Height`（CSS 像素，供 Layout 使用）
- [x] 兜底逻辑：`fbW == cfg.Width && gcsX > 1.0` 时 `fbW = cfg.Width × gcsX`
- [x] `a.width = fbW`（物理像素），`a.logicalW = cfg.Width`（CSS 像素）
- [x] `Centered` 居中使用 `cfg.Width/cfg.Height`（SetPos 和 GetVideoMode 都是 CSS 像素）
- [x] `SetSizeCallback` 中 `a.width = actualFbW`（物理像素），`a.logicalW = width`（CSS 像素）
- [x] `SetFramebufferSizeCallback` 中 `a.width = fbw`（物理像素），`a.logicalW = ww`（CSS 像素）
- [x] 编译通过：`CGO_ENABLED=1 go build -tags skia ./app/...` 和 `./examples/html_layout/...`
- [x] 运行验证：`cfg=880x900` 时 `fb=1100x1125, logical=880x900, cs=1.250, a.width=1100`
- [x] 窗口物理大小随 DPI 自动放大（1100×1125 = 880×1.25 × 900×1.25）
- [x] 事件 hitTest 区域与绘制区域对齐（元素起始位置 (0,0) 对应窗口左上角）
- [x] 临时诊断日志已删除
