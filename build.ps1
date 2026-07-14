# PairCode 构建脚本 — 桌面 GUI 模式 + 系统托盘
$env:CGO_ENABLED = '0'
Write-Host "[INFO] CGO_ENABLED=$env:CGO_ENABLED" -ForegroundColor Cyan

Write-Host "`n=== 1. 构建桌面 GUI 主程序 (companion.exe) ===" -ForegroundColor Cyan
go build -tags desktop -ldflags="-H windowsgui" -o companion.exe ./cmd/companion/
if ($LASTEXITCODE -ne 0) {
    Write-Host "[ERROR] companion.exe 构建失败" -ForegroundColor Red
    exit $LASTEXITCODE
}
Write-Host "[OK] companion.exe 构建成功（桌面 GUI 模式，无控制台窗口）" -ForegroundColor Green

Write-Host "`n=== 2. 构建系统托盘启动器 (companion-tray.exe) ===" -ForegroundColor Cyan
go build -ldflags="-H windowsgui" -o companion-tray.exe ./cmd/companion-tray/
if ($LASTEXITCODE -ne 0) {
    Write-Host "[ERROR] companion-tray.exe 构建失败" -ForegroundColor Red
    exit $LASTEXITCODE
}
Write-Host "[OK] companion-tray.exe 构建成功（系统托盘，无控制台窗口）" -ForegroundColor Green

Write-Host "`n=== 全部构建成功 ===" -ForegroundColor Green
Write-Host "  companion.exe     (桌面 GUI — -tags desktop)" -ForegroundColor Green
Write-Host "  companion-tray.exe (系统托盘启动器)" -ForegroundColor Green
