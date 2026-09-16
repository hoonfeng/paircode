# qr — 内嵌二维码编码器（vendored）

本目录是 [rsc.io/qr](https://rsc.io/qr) **v0.2.0** 的源码内嵌副本（含 LICENSE），
供微信 ClawBot 桥生成登录二维码使用。内嵌原因：桥保持「自持实现、零第三方依赖」
（与 tool-binary 插件范式一致），便于独立构建与分发。

## 文件映射

| 本地文件 | 上游文件 | 改动 |
|---|---|---|
| `qr.go` | `qr.go` | import 路径改为本 module；移除规范 import 注释 |
| `png.go` | `png.go` | 无（纯标准库） |
| `coding/qr.go` | `coding/qr.go` | import 路径改为本 module |
| `gf256/gf256.go` | `gf256/gf256.go` | 无 |
| `render.go` | —（新增） | 便捷一步渲染 |
| `LICENSE` | `LICENSE` | 无（BSD-3-Clause，Go Authors） |

## 用法

```go
png, err := qr.RenderPNG("https://…二维码内容…", 8) // 每模块 8 像素
```

- 纠错级别固定 **L**（与 M1 Node 原型实测可扫参数一致）；
- 输出自带 4 模块静区（quiet zone），可直接屏幕展示；
- 版本自动选择（v1~v40，长内容自动升版本）。

## 上游更新

如需升级：从 `https://goproxy.cn/rsc.io/qr/@v/<版本>.zip` 拉取，重做上述文件映射与
import 路径替换（`rsc.io/qr/coding` → 本 module 路径）。
