# PairCode 构建脚本 — Web-only 模式 + 系统托盘
$env:CGO_ENABLED = '0'
Write-Host "[INFO] CGO_ENABLED=$env:CGO_ENABLED" -ForegroundColor Cyan

Write-Host "`n=== 1. 构建 Web 服务器 (companion.exe) ===" -ForegroundColor Cyan
go build -o companion.exe ./cmd/companion/
if ($LASTEXITCODE -ne 0) {
    Write-Host "[ERROR] companion.exe 构建失败" -ForegroundColor Red
    exit $LASTEXITCODE
}
Write-Host "[OK] companion.exe 构建成功" -ForegroundColor Green

Write-Host "`n=== 2. 构建系统托盘启动器 (companion-tray.exe) ===" -ForegroundColor Cyan
go build -o companion-tray.exe ./cmd/companion-tray/
if ($LASTEXITCODE -ne 0) {
    Write-Host "[ERROR] companion-tray.exe 构建失败" -ForegroundColor Red
    exit $LASTEXITCODE
}
Write-Host "[OK] companion-tray.exe 构建成功" -ForegroundColor Green

Write-Host "`n=== 全部构建成功 ===" -ForegroundColor Green
Write-Host "  companion.exe     (Web IDE 服务器)" -ForegroundColor Green
Write-Host "  companion-tray.exe (系统托盘启动器)" -ForegroundColor Green
