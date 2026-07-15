@echo off
title PairCode Build Script
setlocal enabledelayedexpansion

REM === PairCode Build Script ===
REM Usage: build.bat [pack]
REM   pack  - build + pack release (bin/config/fonts/lib/assets)
REM   (no arg) - build companion.exe only

set "ROOT=%~dp0"
pushd "%ROOT%"

set "CGO_ENABLED=1"
set "ICON_SRC=assets\icon.png"
set "SRC_DIR=cmd\companion"
set "DIST_DIR=release\PairCode"

REM ── preflight checks ──
echo [CHECK] Checking dependencies...

where ffmpeg >nul 2>nul
if %ERRORLEVEL% neq 0 (
    echo [ERROR] ffmpeg not found.
    echo        Install from: https://ffmpeg.org/download.html
    echo        Or via winget: winget install FFmpeg
    popd
    exit /b 1
) else (
    echo  [OK] ffmpeg
)

where windres >nul 2>nul
if %ERRORLEVEL% neq 0 (
    echo [ERROR] windres not found (part of MinGW-w64).
    echo        Install from: https://www.mingw-w64.org/
    echo        Or via winget: winget install mingw
    popd
    exit /b 1
) else (
    echo  [OK] windres
)

go version >nul 2>nul
if %ERRORLEVEL% neq 0 (
    echo [ERROR] Go not found.
    echo        Install from: https://go.dev/dl/
    popd
    exit /b 1
) else (
    for /f "tokens=3" %%g in ('go version') do echo  [OK] Go %%g
)

echo [INFO] CGO_ENABLED=%CGO_ENABLED%
echo.

REM ═══════════════ Step 1: Generate icons ═══════════════
echo === [1/4] Generate Windows icon (.ico) ===
echo  -> scaling to multiple sizes...
ffmpeg -y -i "%ICON_SRC%" -s 16x16 -update 1 "%SRC_DIR%\icon_16.png" >nul 2>&1
ffmpeg -y -i "%ICON_SRC%" -s 32x32 -update 1 "%SRC_DIR%\icon_32.png" >nul 2>&1
ffmpeg -y -i "%ICON_SRC%" -s 48x48 -update 1 "%SRC_DIR%\icon_48.png" >nul 2>&1
ffmpeg -y -i "%ICON_SRC%" -s 64x64 -update 1 "%SRC_DIR%\icon_64.png" >nul 2>&1
ffmpeg -y -i "%ICON_SRC%" -s 128x128 -update 1 "%SRC_DIR%\icon_128.png" >nul 2>&1
ffmpeg -y -i "%ICON_SRC%" -s 256x256 -update 1 "%SRC_DIR%\icon_256.png" >nul 2>&1

if not exist "%SRC_DIR%\icon_256.png" (
    echo [ERROR] Icon scaling failed. Check assets/icon.png exists.
    popd
    exit /b 1
)

echo  -> packing into .ico...
go run scripts\make_icon.go
if %ERRORLEVEL% neq 0 (
    echo [ERROR] ICO packaging failed.
    popd
    exit /b 1
)
echo [OK] icon.ico generated
echo.

REM ═══════════════ Step 2: Compile resource ═══════════════
echo === [2/4] Compile resource (.rc -> .syso) ===
windres -i "%SRC_DIR%\companion.rc" -o "%SRC_DIR%\companion.syso" -O coff
if %ERRORLEVEL% neq 0 (
    echo [ERROR] Resource compilation failed (windres).
    echo        Check that companion.rc and icon.ico exist in %SRC_DIR%\
    popd
    exit /b 1
)
echo [OK] companion.syso generated
echo.

REM ═══════════════ Step 3: Build frontend ═══════════════
echo === [3/4] Build Web UI ===
pushd "%SRC_DIR%\web-ui"
call npm run build
if %ERRORLEVEL% neq 0 (
    echo [ERROR] Web UI build failed.
    popd
    popd
    exit /b 1
)
popd
echo [OK] Web UI built
echo.

REM ═══════════════ Step 4: Build Go binary ═══════════════
echo === [4/4] Build companion.exe ===
echo  -> CGO_ENABLED=%CGO_ENABLED%
go build -o companion.exe ./cmd/companion/
if %ERRORLEVEL% neq 0 (
    echo [ERROR] companion.exe build failed.
    popd
    exit /b 1
)
echo [OK] companion.exe built (with icon + embedded Web UI)
echo.

REM ═══════════════ Optional: pack release ═══════════════
if /i "%1"=="pack" goto :PACKAGE

echo === Build complete ===
echo   companion.exe
echo.
echo To pack release: build.bat pack
popd
goto :EOF

REM ═══════════════ Release packaging ═══════════════
:PACKAGE
echo.
echo === Packing release package ===
echo Output: %DIST_DIR%

if exist "%DIST_DIR%" rmdir /s /q "%DIST_DIR%"

REM create directory structure via PowerShell
powershell -NoProfile -Command "& {
    $d = '%DIST_DIR%';
    @(
        'bin\tesseract\tessdata','bin\tesseract\doc','bin\config',
        'config\skills','config\roles','config\philosophy',
        'assets','fonts','lib'
    ) | ForEach-Object {
        New-Item -ItemType Directory -Path (Join-Path $d $_) -Force | Out-Null
    }
    Write-Host '  directory structure created'
}"

REM 1. main executable
if exist "companion.exe" (
    copy /Y "companion.exe" "%DIST_DIR%\" >nul
    echo  -> companion.exe
) else (
    echo [WARN] companion.exe not found, skipping
)

REM 2. tesseract OCR engine
echo  -> bin\tesseract...
if exist "bin\tesseract\*.exe" copy "bin\tesseract\*.exe" "%DIST_DIR%\bin\tesseract\" >nul 2>&1
if exist "bin\tesseract\*.dll" copy "bin\tesseract\*.dll" "%DIST_DIR%\bin\tesseract\" >nul 2>&1
if exist "bin\tesseract\tessdata\*" xcopy /E /I /Y "bin\tesseract\tessdata\*" "%DIST_DIR%\bin\tesseract\tessdata\" >nul 2>&1
if exist "bin\tesseract\doc\*" xcopy /E /I /Y "bin\tesseract\doc\*" "%DIST_DIR%\bin\tesseract\doc\" >nul 2>&1

REM 3. bin config
if exist "bin\config\models.json" copy "bin\config\models.json" "%DIST_DIR%\bin\config\" >nul 2>&1

REM 4. headless-check.js
if exist "bin\headless-check.js" copy "bin\headless-check.js" "%DIST_DIR%\bin\" >nul 2>&1

REM 5. app config
echo  -> config...
if exist "config\models.json" copy "config\models.json" "%DIST_DIR%\config\" >nul 2>&1
if exist "config\settings.json" copy "config\settings.json" "%DIST_DIR%\config\" >nul 2>&1
if exist "config\mcp.json" copy "config\mcp.json" "%DIST_DIR%\config\" >nul 2>&1
if exist "config\skills\*" xcopy /E /I /Y "config\skills\*" "%DIST_DIR%\config\skills\" >nul 2>&1
if exist "config\roles\*" xcopy /E /I /Y "config\roles\*" "%DIST_DIR%\config\roles\" >nul 2>&1
if exist "config\philosophy\*" xcopy /E /I /Y "config\philosophy\*" "%DIST_DIR%\config\philosophy\" >nul 2>&1

REM 6. assets (runtime icons/logo)
echo  -> assets...
if exist "assets\icon.svg" copy "assets\icon.svg" "%DIST_DIR%\assets\" >nul 2>&1
if exist "assets\icon.png" copy "assets\icon.png" "%DIST_DIR%\assets\" >nul 2>&1
if exist "assets\icon64.png" copy "assets\icon64.png" "%DIST_DIR%\assets\" >nul 2>&1
if exist "assets\icon128.png" copy "assets\icon128.png" "%DIST_DIR%\assets\" >nul 2>&1

REM 7. fonts (Chinese fonts for rendering)
echo  -> fonts...
if exist "fonts\*.ttf" copy "fonts\*.ttf" "%DIST_DIR%\fonts\" >nul 2>&1

REM 8. lib (SkiaSharp)
echo  -> lib...
if exist "lib\libSkiaSharp.dll" copy "lib\libSkiaSharp.dll" "%DIST_DIR%\lib\" >nul 2>&1

REM summary
echo.
powershell -NoProfile -Command "& {
    $dir = '%DIST_DIR%';
    $count = (Get-ChildItem -Recurse $dir | Measure-Object).Count;
    $size = (Get-ChildItem -Recurse $dir | Measure-Object -Property Length -Sum).Sum;
    $sizeMB = [math]::Round($size / 1MB, 1);
    Write-Host ('[OK] Release package generated: ' + $dir + ' (' + $count + ' files, ' + $sizeMB + ' MB)');
}"
popd
goto :EOF
