# 运行时资源目录（主程序只保留框架）

本目录存放 IDE 运行时资源（原内嵌于 companion.exe 的 //go:embed 资源）。
**外部文件优先加载**：修改/替换本目录文件 → 重启生效，无需重新编译 Go。

## 资源清单与来源

| 文件 | 说明 | 源（Go embed fallback） |
|---|---|---|
| `cordis.bundle.js` | @cordisjs/core 3.18.1 goja 沙箱运行时（IIFE bundle，全局挂 CordisApi） | `internal/agent/assets/cordis.bundle.js` |
| `bridge_node.js` | Node 运行时桥脚本（真实 node 进程执行 npm cordis 插件） | `internal/agent/bridge_node.js` |
| `ide_ref.html` | 调试参照收集器模板（Edge headless 加载真实前端） | `cmd/companion/web-ui/ide_ref.html` |
| `ide_ref_select.html` | select 下拉箭头浏览器标准参照 | `cmd/companion/web-ui/ide_ref_select.html` |
| `ide_ref_modal.html` | 设置/工具弹窗 modal 几何浏览器参照 | `cmd/companion/web-ui/ide_ref_modal.html` |
| `ide_ref_setmodal.html` | 设置面板独立参照 | `cmd/companion/web-ui/ide_ref_setmodal.html` |

## 加载机制

- 加载顺序：**本目录（.pair/assets/runtime/）→ Go embed 兜底**
- 实现：`internal/agent/runtime_assets.go`（LoadRuntimeAsset / LoadRuntimeAssetString）
- 消费点：jsplugin.go `cordisBundleSource()`、node_bridge.go `bridgeNodeSource()`、
  cmd/companion/web_server.go（ide_ref* 4 端点）
- 资源缺失时回退内嵌版本，单文件分发仍可运行

## 替换方法

1. 编辑本目录对应文件（如 `bridge_node.js` 可自由修改桥协议）
2. 重启 companion → 外部版本生效

## 重建 cordis.bundle.js（相关源码）

`cordis.bundle.js` 由 @cordisjs/core 3.18.1 经 esbuild bundle：
```bash
npm i @cordisjs/core@3.18.1
npx esbuild node_modules/@cordisjs/core/lib/index.mjs \
  --bundle --format=iife --platform=neutral \
  --global-name=CordisApi --outfile=.pair/assets/runtime/cordis.bundle.js
# 注意：--platform=neutral 无 require/process/Buffer（goja 沙箱限制）
```
详细记录见 `.pair/project-info/关键点/修复记录-cordis核心goja验证+trap对齐2026-08-15.md`。

## 磁盘插件（.pair/plugins/）

工具/UI 插件目录见 `.pair/plugins/`（tool-* 20 组工具 + ui-* 8 组 UI）。
生成器（复杂工具组迁移）：`internal/agent/tool_plugin_gen.go`（toolsgen tag）+
`dev/tool_plugin_gen/main.go`，重跑 `go run -tags toolsgen ./dev/tool_plugin_gen`。
