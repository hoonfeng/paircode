# PairCode Build Script - GUI Panel + Web Server (same process)
# Usage: .\build.ps1 [-Pack] [-IconsOnly]

param(
    [switch]$Pack,
    [switch]$IconsOnly
)

$ErrorActionPreference = "Stop"
$ROOT = Split-Path -Parent $MyInvocation.MyCommand.Definition
Set-Location $ROOT

$env:CGO_ENABLED = '1'
$IconSrc = "assets\icon.png"
$SrcDir = "cmd\companion"
$DistDir = "release\PairCode"

Write-Host "[INFO] CGO_ENABLED=$env:CGO_ENABLED" -ForegroundColor Cyan

# ── preflight ──
Write-Host "`n[CHECK] Checking dependencies..." -ForegroundColor Cyan
$tools = @{ffmpeg = $false; windres = $false; go = $false}
Get-Command ffmpeg -ErrorAction SilentlyContinue | Out-Null; if ($?) { $tools.ffmpeg = $true }
Get-Command windres -ErrorAction SilentlyContinue | Out-Null; if ($?) { $tools.windres = $true }
Get-Command go -ErrorAction SilentlyContinue | Out-Null; if ($?) { $tools.go = $true }
foreach ($k in $tools.Keys) {
    if ($tools[$k]) { Write-Host "  [OK] $k" -ForegroundColor Green }
    else { Write-Host "  [FAIL] $k" -ForegroundColor Red; exit 1 }
}

# ═══════════════ Step 1: Icons ═══════════════
Write-Host "`n=== [1/4] Generate Windows icon (.ico) ===" -ForegroundColor Cyan
Write-Host "  -> scaling to multiple sizes..."
foreach ($s in @(16,32,48,64,128,256)) {
    $out = "$SrcDir\icon_$s.png"
    & ffmpeg -y -i $IconSrc -s "${s}x${s}" -update 1 $out 2>&1 | Out-Null
}
if (-not (Test-Path "$SrcDir\icon_256.png")) {
    Write-Host "[ERROR] Icon scaling failed" -ForegroundColor Red; exit 1
}

Write-Host "  -> packing into .ico..."
go run scripts\make_icon.go
if ($LASTEXITCODE -ne 0) { Write-Host "[ERROR] ICO packaging failed" -ForegroundColor Red; exit 1 }
Write-Host "[OK] icon.ico generated" -ForegroundColor Green

if ($IconsOnly) { Write-Host "`n[OK] Icons generated only (use -Pack or no switch for full build)" -ForegroundColor Green; return }

# ═══════════════ Step 2: Resource ═══════════════
Write-Host "`n=== [2/4] Compile resource (.rc -> .syso) ===" -ForegroundColor Cyan
windres -i "$SrcDir\companion.rc" -o "$SrcDir\companion.syso" -O coff
if ($LASTEXITCODE -ne 0) { Write-Host "[ERROR] Resource compilation failed" -ForegroundColor Red; exit 1 }
Write-Host "[OK] companion.syso generated" -ForegroundColor Green

# ═══════════════ Step 3: Frontend ═══════════════
Write-Host "`n=== [3/4] Build Web UI ===" -ForegroundColor Cyan
Push-Location "$SrcDir\web-ui"
npm run build
if ($LASTEXITCODE -ne 0) { Write-Host "[ERROR] Web UI build failed" -ForegroundColor Red; exit 1 }
Pop-Location
Write-Host "[OK] Web UI built" -ForegroundColor Green

# ═══════════════ Step 4: Go build ═══════════════
Write-Host "`n=== [4/4] Build companion.exe ===" -ForegroundColor Cyan
go build -o companion.exe ./cmd/companion/
if ($LASTEXITCODE -ne 0) { Write-Host "[ERROR] companion.exe build failed" -ForegroundColor Red; exit 1 }
Write-Host "[OK] companion.exe built (with icon + embedded Web UI)" -ForegroundColor Green

# ═══════════════ Optional: pack ═══════════════
if (-not $Pack) {
    Write-Host "`n=== Build complete ===" -ForegroundColor Green
    Write-Host "  companion.exe"
    Write-Host "`nTo pack release: .\build.ps1 -Pack"
    return
}

Write-Host "`n=== Packing release package ===" -ForegroundColor Cyan
Write-Host "  Output: $DistDir"

if (Test-Path $DistDir) { Remove-Item -Recurse -Force $DistDir }

$dirs = @(
    "bin\tesseract\tessdata","bin\tesseract\doc","bin\config",
    "config\skills","config\roles","config\philosophy",
    "assets","fonts","lib"
)
foreach ($d in $dirs) { New-Item -ItemType Directory -Path (Join-Path $DistDir $d) -Force | Out-Null }

# 1. main binary
if (Test-Path "companion.exe") { Copy-Item "companion.exe" "$DistDir\" }

# 2. tesseract
Write-Host "  -> bin\tesseract..."
if (Test-Path "bin\tesseract\*.exe") { Copy-Item "bin\tesseract\*.exe" "$DistDir\bin\tesseract\" }
if (Test-Path "bin\tesseract\*.dll") { Copy-Item "bin\tesseract\*.dll" "$DistDir\bin\tesseract\" }
if (Test-Path "bin\tesseract\tessdata") { Copy-Item -Recurse "bin\tesseract\tessdata\*" "$DistDir\bin\tesseract\tessdata\" }
if (Test-Path "bin\tesseract\doc") { Copy-Item -Recurse "bin\tesseract\doc\*" "$DistDir\bin\tesseract\doc\" }

# 3-4. bin config + headless-check
if (Test-Path "bin\config\models.json") { Copy-Item "bin\config\models.json" "$DistDir\bin\config\" }
if (Test-Path "bin\headless-check.js") { Copy-Item "bin\headless-check.js" "$DistDir\bin\" }

# 5. app config
Write-Host "  -> config..."
foreach ($f in @("models.json","settings.json","mcp.json")) {
    $src = "config\$f"
    if (Test-Path $src) { Copy-Item $src "$DistDir\config\" }
}
foreach ($d in @("skills","roles","philosophy")) {
    $src = "config\$d"
    if (Test-Path $src) { Copy-Item -Recurse "$src\*" "$DistDir\config\$d\" }
}

# 6. assets
Write-Host "  -> assets..."
foreach ($f in @("icon.svg","icon.png","icon64.png","icon128.png")) {
    $src = "assets\$f"
    if (Test-Path $src) { Copy-Item $src "$DistDir\assets\" }
}

# 7. fonts
Write-Host "  -> fonts..."
if (Test-Path "fonts\*.ttf") { Copy-Item "fonts\*.ttf" "$DistDir\fonts\" }

# 8. lib
Write-Host "  -> lib..."
if (Test-Path "lib\libSkiaSharp.dll") { Copy-Item "lib\libSkiaSharp.dll" "$DistDir\lib\" }

# summary
$count = (Get-ChildItem -Recurse $DistDir | Measure-Object).Count
$size = (Get-ChildItem -Recurse $DistDir | Measure-Object -Property Length -Sum).Sum
$sizeMB = [math]::Round($size / 1MB, 1)
Write-Host "`n[OK] Release package generated: $DistDir ($count files, $sizeMB MB)" -ForegroundColor Green
