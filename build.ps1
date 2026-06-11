# goui 构建脚本 — 必须设置 CGO_ENABLED=1（依赖 goskia CGO 绑定）
$env:CGO_ENABLED = '1'
Write-Host "[INFO] CGO_ENABLED=$env:CGO_ENABLED" -ForegroundColor Cyan
Write-Host "[INFO] Building companion.exe ..." -ForegroundColor Cyan

go build -o companion.exe ./cmd/companion/
if ($LASTEXITCODE -ne 0) {
    Write-Host "[ERROR] 构建失败，请确保:" -ForegroundColor Red
    Write-Host "  1. GCC/MinGW 在 PATH 中（CGO 需要 C 编译器）"
    Write-Host "  2. F:\syproject\goskia 仓库存在"
    Write-Host "  3. libSkiaSharp.dll 在项目根目录"
    exit $LASTEXITCODE
}
Write-Host "[OK] companion.exe 构建成功" -ForegroundColor Green
