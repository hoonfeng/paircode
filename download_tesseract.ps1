# download_tesseract.ps1 — 自动下载部署 Tesseract OCR 便携版到 bin/tesseract/
#
# 用途：image_ocr 工具依赖 Tesseract OCR。本脚本自动完成：
#   1. 从 UB-Mannheim 下载官方 Windows 构建（含引擎与依赖 DLL）
#   2. 用 7-Zip 提取可移植文件（不依赖系统安装）
#   3. 部署到 <项目根>/bin/tesseract/（agent 自动识别使用）
#   4. 补充中文语言包 chi_sim.traineddata
#
# 用法：powershell -ExecutionPolicy Bypass -File download_tesseract.ps1
# 幂等：重复运行会覆盖为最新版本。

$ErrorActionPreference = 'Stop'
$Root = Split-Path -Parent $MyInvocation.MyCommand.Path
$DestDir = Join-Path $Root 'bin\tesseract'
$TempDir = Join-Path $Root '_temp\tess_dl'
$Version = '5.4.0.20240606'
$InstallerUrl = "https://github.com/UB-Mannheim/tesseract/releases/download/v$Version/tesseract-ocr-w64-setup-$Version.exe"
$ChiSimUrl = 'https://github.com/tesseract-ocr/tessdata/raw/main/chi_sim.traineddata'

function Find-7z {
    foreach ($p in @("$env:ProgramFiles\7-Zip\7z.exe", "${env:ProgramFiles(x86)}\7-Zip\7z.exe")) {
        if (Test-Path $p) { return $p }
    }
    $cmd = Get-Command 7z -ErrorAction SilentlyContinue
    if ($cmd) { return $cmd.Source }
    return $null
}

Write-Host "==> Tesseract $Version 便携版部署到 $DestDir"
New-Item -ItemType Directory -Force -Path $TempDir, $DestDir | Out-Null
$Installer = Join-Path $TempDir "tesseract-setup-$Version.exe"

if (-not (Test-Path $Installer)) {
    Write-Host "==> 下载安装包 ($Version)..."
    curl.exe -L --fail --connect-timeout 30 -o $Installer $InstallerUrl
    if ($LASTEXITCODE -ne 0) { throw "下载失败: $InstallerUrl" }
}

$7z = Find-7z
if (-not $7z) { throw "未找到 7-Zip，请先安装（https://www.7-zip.org/）或手动解压安装包到 $DestDir" }

$ExtractDir = Join-Path $TempDir "extract"
if (Test-Path $ExtractDir) { Remove-Item -Recurse -Force $ExtractDir }
Write-Host "==> 提取安装包..."
& $7z x $Installer "-o$ExtractDir" -y -bso0 -bsp0 | Out-Null

Write-Host "==> 复制运行文件到 bin/tesseract/..."
Get-ChildItem $ExtractDir -File | Where-Object { $_.Extension -in '.exe', '.dll' } | Copy-Item -Destination $DestDir -Force
Copy-Item -Path (Join-Path $ExtractDir 'tessdata') -Destination $DestDir -Recurse -Force

Write-Host "==> 移除训练工具（非运行必需）..."
@('lstmtraining.exe','text2image.exe','cntraining.exe','mftraining.exe','combine_tessdata.exe',
  'combine_lang_model.exe','dawg2wordlist.exe','ambiguous_words.exe','classifier_tester.exe',
  'unicharset_extractor.exe','merge_unicharsets.exe','set_unicharset_properties.exe',
  'shapeclustering.exe','wordlist2dawg.exe','lstmeval.exe','tesseract-uninstall.exe') | ForEach-Object {
    $p = Join-Path $DestDir $_
    if (Test-Path $p) { Remove-Item $p -Force }
}

$ChiSim = Join-Path $DestDir 'tessdata\chi_sim.traineddata'
if (-not (Test-Path $ChiSim)) {
    Write-Host "==> 下载中文语言包 chi_sim (~42MB)..."
    curl.exe -L --fail --connect-timeout 30 -o $ChiSim $ChiSimUrl
    if ($LASTEXITCODE -ne 0) { Write-Warning "chi_sim 下载失败，中文 OCR 不可用（英文 eng 仍可用）" }
}

Write-Host "==> 验证..."
& (Join-Path $DestDir 'tesseract.exe') --version
& (Join-Path $DestDir 'tesseract.exe') --list-langs
Write-Host "完成！agent 将自动识别 bin/tesseract/ 下的 Tesseract。"
