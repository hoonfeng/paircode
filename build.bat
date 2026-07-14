@echo off
REM PairCode Build Script - GUI Panel + Web Server (same process)
echo [INFO] CGO mode enabled (Skia GPU)

echo.
echo === Building companion.exe ===
go build -o companion.exe ./cmd/companion/
if %ERRORLEVEL% neq 0 (
    echo [ERROR] companion.exe build failed
    exit /b %ERRORLEVEL%
)
echo [OK] companion.exe built successfully

echo.
echo === All builds successful ===
echo companion.exe - GUI startup panel + Web IDE server
