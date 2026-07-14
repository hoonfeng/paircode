@echo off
REM PairCode 构建脚本 — 桌面 GUI 模式 + 系统托盘
set CGO_ENABLED=0
echo [INFO] CGO_ENABLED=%CGO_ENABLED%

echo.
echo === 1. 构建桌面 GUI 主程序 (companion.exe) ===
go build -tags desktop -ldflags="-H windowsgui" -o companion.exe ./cmd/companion/
if %ERRORLEVEL% neq 0 (
    echo [ERROR] companion.exe 构建失败
    exit /b %ERRORLEVEL%
)
echo [OK] companion.exe 构建成功（桌面 GUI 模式，无控制台窗口）

echo.
echo === 2. 构建系统托盘启动器 (companion-tray.exe) ===
go build -ldflags="-H windowsgui" -o companion-tray.exe ./cmd/companion-tray/
if %ERRORLEVEL% neq 0 (
    echo [ERROR] companion-tray.exe 构建失败
    exit /b %ERRORLEVEL%
)
echo [OK] companion-tray.exe 构建成功（系统托盘，无控制台窗口）

echo.
echo === 全部构建成功 ===
echo   companion.exe     (桌面 GUI — -tags desktop)
echo   companion-tray.exe (系统托盘启动器)
