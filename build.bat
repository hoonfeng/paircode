@echo off
chcp 65001 >nul
setlocal enabledelayedexpansion

echo ============================================
echo  PairCode IDE - Build ^& Package
echo ============================================

:: -- Version --
set VERSION=1.0.0
for /f "tokens=*" %%i in ('git log --oneline -1 2^>nul') do set GIT_HASH=%%i
if defined GIT_HASH set VERSION=%VERSION%-%GIT_HASH:~0,7%

echo Version: %VERSION%
echo.

:: -- Paths --
set ROOT=%~dp0
set DIST_DIR=%ROOT%release\PairCode
set RELEASE_DIR=%ROOT%release

:: -- Step 1: Build --
echo [1/4] Building companion.exe (with icon)...

cd /d "%ROOT%cmd\companion"
go build -ldflags="-s -w" -o "%ROOT%companion.exe" .
if %errorlevel% neq 0 (
    echo BUILD FAILED.
    pause
    exit /b %errorlevel%
)
echo Build OK

:: -- Step 2: Create dist dir --
echo [2/4] Creating distribution directory...

if exist "%DIST_DIR%" rmdir /S /Q "%DIST_DIR%" >nul 2>&1
mkdir "%DIST_DIR%" 2>nul || (
    echo Failed to create dist directory.
    pause
    exit /b 1
)
mkdir "%DIST_DIR%\bin" >nul 2>&1
mkdir "%DIST_DIR%\config" >nul 2>&1
mkdir "%DIST_DIR%\assets" >nul 2>&1
mkdir "%DIST_DIR%\fonts" >nul 2>&1
mkdir "%DIST_DIR%\lib" >nul 2>&1

:: -- Step 3: Copy files --
echo [3/4] Copying distribution files...

:: Main binary
copy /Y "%ROOT%companion.exe" "%DIST_DIR%\" >nul

:: Tesseract OCR
if exist "%ROOT%bin\tesseract\tesseract.exe" (
    mkdir "%DIST_DIR%\bin\tesseract\tessdata" >nul 2>&1
    copy "%ROOT%bin\tesseract\tesseract.exe" "%DIST_DIR%\bin\tesseract\" >nul 2>&1
    copy "%ROOT%bin\tesseract\tesseract-uninstall.exe" "%DIST_DIR%\bin\tesseract\" >nul 2>&1
    copy "%ROOT%bin\tesseract\winpath.exe" "%DIST_DIR%\bin\tesseract\" >nul 2>&1
    copy "%ROOT%bin\tesseract\*.dll" "%DIST_DIR%\bin\tesseract\" >nul 2>&1
    if exist "%ROOT%bin\tesseract\tessdata\*" (
        xcopy /E /I /Y "%ROOT%bin\tesseract\tessdata\*" "%DIST_DIR%\bin\tesseract\tessdata\" >nul 2>&1
    )
)

:: Config & models
if exist "%ROOT%bin\config\models.json" (
    mkdir "%DIST_DIR%\bin\config" >nul 2>&1
    copy "%ROOT%bin\config\models.json" "%DIST_DIR%\bin\config\" >nul 2>&1
)
if exist "%ROOT%bin\headless-check.js" copy "%ROOT%bin\headless-check.js" "%DIST_DIR%\bin\" >nul 2>&1

:: App config
if exist "%ROOT%config\models.json" copy "%ROOT%config\models.json" "%DIST_DIR%\config\" >nul 2>&1
if exist "%ROOT%config\settings.template.json" copy "%ROOT%config\settings.template.json" "%DIST_DIR%\config\settings.json" >nul 2>&1
if exist "%ROOT%config\mcp.json" copy "%ROOT%config\mcp.json" "%DIST_DIR%\config\" >nul 2>&1
if exist "%ROOT%config\skills\*" xcopy /E /I /Y "%ROOT%config\skills\*" "%DIST_DIR%\config\skills\" >nul 2>&1
if exist "%ROOT%config\roles\*" xcopy /E /I /Y "%ROOT%config\roles\*" "%DIST_DIR%\config\roles\" >nul 2>&1
if exist "%ROOT%config\philosophy\*" xcopy /E /I /Y "%ROOT%config\philosophy\*" "%DIST_DIR%\config\philosophy\" >nul 2>&1

:: Assets
if exist "%ROOT%assets\icon.svg" copy "%ROOT%assets\icon.svg" "%DIST_DIR%\assets\" >nul 2>&1
if exist "%ROOT%assets\icon.png" copy "%ROOT%assets\icon.png" "%DIST_DIR%\assets\" >nul 2>&1
if exist "%ROOT%assets\icon64.png" copy "%ROOT%assets\icon64.png" "%DIST_DIR%\assets\" >nul 2>&1
if exist "%ROOT%assets\icon128.png" copy "%ROOT%assets\icon128.png" "%DIST_DIR%\assets\" >nul 2>&1

:: Fonts
if exist "%ROOT%fonts\*.ttf" copy "%ROOT%fonts\*.ttf" "%DIST_DIR%\fonts\" >nul 2>&1

:: SkiaSharp
if exist "%ROOT%lib\libSkiaSharp.dll" copy "%ROOT%lib\libSkiaSharp.dll" "%DIST_DIR%\lib\" >nul 2>&1

echo Copy OK

:: -- Step 4: Zip --
echo [4/4] Creating ZIP package...

set ZIP_NAME=%RELEASE_DIR%\PairCode-%VERSION%.zip
if exist "%ZIP_NAME%" del "%ZIP_NAME%" >nul 2>&1

:: Generate a temporary PowerShell script for zipping (avoids encoding issues with inline PS)
set PS_SCRIPT=%TEMP%\paircode_zip_%RANDOM%.ps1
echo $src = '%DIST_DIR:\=\\%' > "%PS_SCRIPT%"
echo $dst = '%ZIP_NAME:\=\\%' >> "%PS_SCRIPT%"
echo try { >> "%PS_SCRIPT%"
echo   Compress-Archive -Path "$src\*" -DestinationPath $dst -Force -ErrorAction Stop >> "%PS_SCRIPT%"
echo   Write-Host 'ZIP_OK' >> "%PS_SCRIPT%"
echo } catch { >> "%PS_SCRIPT%"
echo   Write-Host 'ZIP_ERR:' $_ >> "%PS_SCRIPT%"
echo } >> "%PS_SCRIPT%"

powershell -NoProfile -ExecutionPolicy Bypass -File "%PS_SCRIPT%"
del "%PS_SCRIPT%" >nul 2>&1

if exist "%ZIP_NAME%" (
    for %%f in ("%ZIP_NAME%") do set ZIP_SIZE=%%~zf
    echo Zip OK!
    echo   Output: %ZIP_NAME%
    set /a ZIP_SIZE_MB=!ZIP_SIZE! / 1048576
    echo   Size: !ZIP_SIZE_MB! MB
) else (
    echo ZIP FAILED. Please manually compress: %DIST_DIR%
)

:: Cleanup temp build artifact
if exist "%ROOT%companion.exe" del "%ROOT%companion.exe" >nul 2>&1

echo.
echo ============================================
echo  Build Complete!
echo  Version: %VERSION%
echo  Output: %DIST_DIR%
echo  Package: %ZIP_NAME%
echo ============================================
endlocal
pause


