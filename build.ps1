# PairCode 构建脚本 — 启动面板 + Web 服务器
$env:CGO_ENABLED = '1'
Write-Host "[INFO] CGO_ENABLED=$env:CGO_ENABLED" -ForegroundColor Cyan

Write-Host "`n=== Building companion.exe ===" -ForegroundColor Cyan
go build -o companion.exe ./cmd/companion/
if ($LASTEXITCODE -ne 0) {
    Write-Host "[ERROR] companion.exe 构建失败" -ForegroundColor Red
    exit $LASTEXITCODE
}
Write-Host "[OK] companion.exe 构建成功（启动面板 + Web IDE 服务器）" -ForegroundColor Green

Write-Host "`n=== Done ===" -ForegroundColor Green
Write-Host "  companion.exe - GUI 启动面板 + Web IDE 服务器" -ForegroundColor Green
