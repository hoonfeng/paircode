@echo off
REM PairCode 构建脚本 — Web-only 模式 + 系统托盘
set CGO_ENABLED=0
echo [INFO] CGO_ENABLED=%CGO_ENABLED%

echo.
echo === 1. 构建 Web 服务器 (companion.exe) ===
go build -o companion.exe ./cmd/companion/
if %ERRORLEVEL% neq 0 (
    echo [ERROR] companion.exe 构建失败
    exit /b %ERRORLEVEL%
)
echo [OK] companion.exe 构建成功

echo.
echo === 2. 构建系统托盘启动器 (companion-tray.exe) ===
go build -o companion-tray.exe ./cmd/companion-tray/
if %ERRORLEVEL% neq 0 (
    echo [ERROR] companion-tray.exe 构建失败
    exit /b %ERRORLEVEL%
)
echo [OK] companion-tray.exe 构建成功

echo.
echo === 全部构建成功 ===
echo   companion.exe     (Web IDE 服务器)
echo   companion-tray.exe (系统托盘启动器)
