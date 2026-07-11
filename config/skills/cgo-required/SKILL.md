---
name: cgo-required
description: 编译 Go 项目时 CGO 必须开启（CGO_ENABLED=1），否则链接 C 依赖（Skia/SQLite 等）会报错
activation: auto
globs: "*.go Makefile build.bat build.ps1 Dockerfile *.yaml *.yml docker-compose*"
version: 1.0.0
---

# CGO 编译开关规则

## 原则

项目中使用 C 语言依赖（如 Skia、SQLite 的 CGo 绑定、libxml2 等）时，**编译时必须开启 CGO（CGO_ENABLED=1）**。CGO 默认在 Windows 和 Linux 上是启用的，但某些场景（交叉编译、CI 环境、部分 IDE 的 Run Configuration）可能会被关闭，导致链接失败。

本项目的核心 GUI 引擎依赖 Skia 图形库（通过 CGo 调用），因此 CGO 是**刚性要求**。

## 典型错误

编译时遇到以下错误，通常是 CGO 未开启：

```
# 错误 1：链接器找不到符号（最常见）
/tmp/go-build/xxx/_pkg_.a: undefined reference to 'sk_xxx_create'

# 错误 2：C 头文件找不到
fatal error: Skia.h: No such file or directory

# 错误 3：pure Go 编译失败（CGo 代码被忽略）
cannot use (variable of type *C.sk_type) as type *sk_type in assignment

# 错误 4：build 约束排除（//go:build cgo 标记）
build excluded by build constraints
```

## 应该怎么做

### 1. 本地开发编译

```powershell
# ✅ 正确：显式开启 CGO
$env:CGO_ENABLED=1
go build ./cmd/companion/...

# ✅ 或在一行中设置
$env:CGO_ENABLED=1; go build ./cmd/companion/...
```

```bash
# ✅ Linux/macOS
CGO_ENABLED=1 go build ./cmd/companion/...
```

### 2. 使用构建脚本（项目已提供）

项目根目录已有 `build.bat` 和 `build.ps1` 构建脚本，它们已正确设置 CGO_ENABLED=1：

```powershell
# ✅ 使用 PowerShell 构建脚本
./build.ps1

# ✅ 使用 CMD 构建脚本
build.bat
```

### 3. IDE / Editor 配置

#### VS Code（.vscode/launch.json）
```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Build Companion",
      "type": "go",
      "request": "launch",
      "mode": "exec",
      "program": "${workspaceFolder}/cmd/companion",
      "env": {
        "CGO_ENABLED": "1"
      },
      "buildFlags": "-tags=skia"
    }
  ]
}
```

#### GoLand / IntelliJ
- **Run → Edit Configurations**
- 在 Environment 中添加：`CGO_ENABLED=1`
- 在 Go tool arguments 中添加：`-tags=skia`

### 4. CI/CD 配置

#### GitHub Actions
```yaml
- name: Build
  env:
    CGO_ENABLED: '1'
  run: go build ./cmd/companion/...
```

#### 自定义 Dockerfile
```dockerfile
FROM golang:1.22-bookworm AS builder
ENV CGO_ENABLED=1
# 安装 C 依赖（Skia 需要）
RUN apt-get update && apt-get install -y --no-install-recommends \
    libx11-dev libxcursor-dev libxrandr-dev libxi-dev \
    libgl1-mesa-dev libxxf86vm-dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /companion ./cmd/companion
```

### 5. 检查 CGO 状态

```powershell
# 查看当前 CGO 是否开启
go env CGO_ENABLED
# 输出 '1' 表示开启，'0' 表示关闭

# 查看 CGO 依赖的编译器
go env CC
# 检查 C 编译器是否存在
```

### 6. Windows 特殊注意事项

Windows 上 CGO 需要安装 **MinGW-w64** 或 **TDM-GCC** 提供 GCC 编译器。

```powershell
# 验证 GCC 可用
gcc --version

# 如果未安装，推荐使用 MSYS2 安装
# https://www.msys2.org/
# pacman -S mingw-w64-x86_64-gcc
```

## 项目依赖检查清单

| 依赖 | 是否依赖 CGO | 备注 |
|------|:-----------:|------|
| `github.com/jeffail/gabs/v2` | ❌ 纯 Go | |
| `gioui.org` (Gio UI) | ✅ 需要 CGO | 底层调用 Skia/OpenGL |
| `github.com/xyproto/algernon` | ❌ 纯 Go | |
| `github.com/nickng/scribble` | ❌ 纯 Go | |
| 内部 LSP 模块（`internal/lsp`） | ❌ 纯 Go | JSON-RPC over TCP |
| `libSkiaSharp.dll` | ✅ 需要 CGO | 项目中已包含预编译 DLL |

## 快速诊断流程

遇到编译报错时，按以下步骤排查是否为 CGO 问题：

```
1. 运行 go env CGO_ENABLED
   ├→ 输出 '0' → 执行 $env:CGO_ENABLED=1 后重试编译
   └→ 输出 '1' → 继续排查

2. 检查 C 编译器是否存在
   ├→ gcc --version 失败 → 安装 MinGW-w64 / GCC
   └→ 正常 → 继续排查

3. 检查 CC 环境变量
   └→ go env CC 是否指向正确的编译器路径

4. 使用项目构建脚本
   └→ ./build.ps1 或 build.bat（已内置 CGO_ENABLED=1）
```

## 示例

### ❌ 错误做法（CGO 未开启）

```powershell
# ❌ 直接编译，依赖 CGO_ENABLED 默认值
go build ./cmd/companion/...

# ❌ 显式关闭 CGO
$env:CGO_ENABLED=0; go build ./cmd/companion/...

# ❌ 在禁用 CGO 的 CI 环境中编译
# 错误：undefined reference to 'sk_xxx'
```

### ✅ 正确做法（CGO 已开启）

```powershell
# ✅ 显式开启 CGO 后编译
$env:CGO_ENABLED=1
go build ./cmd/companion/...

# ✅ 使用项目构建脚本
./build.ps1

# ✅ 使用 CMD 脚本
build.bat

# ✅ 完整构建流程
$env:CGO_ENABLED=1
$env:CC="gcc"  # Windows: 指定 MinGW GCC
go build -v -x ./cmd/companion/...  # -v 显示进度，-x 显示执行的命令
```
