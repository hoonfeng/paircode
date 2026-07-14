@echo off
REM PairCode Build Script - GUI Panel + Web Server (same process)
set CGO_ENABLED=0

echo [INFO] CGO_ENABLED=%CGO_ENABLED%

echo.
echo === Building companion.exe ===
go build -o companion.exe ./cmd/companion/
if %ERRORLEVEL% neq 0 (
    echo [ERROR] companion.exe build failed
    exit /b %ERRORLEVEL%
)
echo [OK] companion.exe built successfully

echo.
echo === Done ===
echo companion.exe  (GUI startup panel + Web IDE server)
