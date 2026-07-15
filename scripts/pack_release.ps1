# pack_release.ps1 — 打包发布包（使用已编译的 companion.exe）
$DistDir = "release\PairCode"
if (Test-Path $DistDir) { Remove-Item -Recurse -Force $DistDir }

$dirs = @(
    "bin\tesseract\tessdata","bin\config",
    "config\skills","config\roles","config\philosophy",
    "assets","fonts","lib"
)
foreach ($d in $dirs) { New-Item -ItemType Directory -Path (Join-Path $DistDir $d) -Force | Out-Null }

# 1. main binary
Copy-Item "companion.exe" "$DistDir\"

# 2. tesseract (only runtime files)
Write-Host "  -> bin\tesseract..."
foreach ($exe in @('tesseract.exe','tesseract-uninstall.exe','winpath.exe')) {
    $src = "bin\tesseract\$exe"
    if (Test-Path $src) { Copy-Item $src "$DistDir\bin\tesseract\" }
}
if (Test-Path "bin\tesseract\*.dll") { Copy-Item "bin\tesseract\*.dll" "$DistDir\bin\tesseract\" }
if (Test-Path "bin\tesseract\tessdata") { Copy-Item -Recurse "bin\tesseract\tessdata\*" "$DistDir\bin\tesseract\tessdata\" }

# 3-4. bin config + headless-check
if (Test-Path "bin\config\models.json") { Copy-Item "bin\config\models.json" "$DistDir\bin\config\" }
if (Test-Path "bin\headless-check.js") { Copy-Item "bin\headless-check.js" "$DistDir\bin\" }

# 5. app config
Write-Host "  -> config..."
if (Test-Path "config\models.json") { Copy-Item "config\models.json" "$DistDir\config\" }
if (Test-Path "config\settings.json") { Copy-Item "config\settings.json" "$DistDir\config\" }
if (Test-Path "config\mcp.json") { Copy-Item "config\mcp.json" "$DistDir\config\" }
if (Test-Path "config\skills") { Copy-Item -Recurse "config\skills\*" "$DistDir\config\skills\" }
if (Test-Path "config\roles") { Copy-Item -Recurse "config\roles\*" "$DistDir\config\roles\" }
if (Test-Path "config\philosophy") { Copy-Item -Recurse "config\philosophy\*" "$DistDir\config\philosophy\" }

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
Write-Host "`n[OK] Release package generated: $DistDir ($count files, $sizeMB MB)"