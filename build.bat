@echo off
REM goui 构建脚本 — 必须设置 CGO_ENABLED=1（依赖 goskia CGO 绑定）
set CGO_ENABLED=0
echo [INFO] CGO_ENABLED=%CGO_ENABLED%
echo [INFO] Building web.exe ...
go build -tags webonly -o web.exe ./cmd/companion/
if %ERRORLEVEL% neq 0 (
    echo [ERROR] 构建失败，请确保:
    echo  1. GCC/MinGW 在 PATH 中（CGO 需要 C 编译器）
    echo  2. F:\syproject\goskia 仓库存在
    echo  3. libSkiaSharp.dll 在项目根目录
    goto :EOF
)
echo [OK] web.exe 构建成功
