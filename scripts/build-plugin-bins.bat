@echo off
rem ===========================================================
rem build-plugin-bins.bat - build all standalone plugin binaries
rem   plugins-src/plugins/<name>/ -> .pair/plugins/<name>/bin/<name>.exe
rem   (plugin dirs are self-contained; run this after clone)
rem ===========================================================
setlocal enabledelayedexpansion
cd /d "%~dp0.."
set CGO_ENABLED=0

for /d %%d in (plugins-src\plugins\*) do (
  set NAME=%%~nxd
  if not exist "plugins-src\plugins\!NAME!\main.go" (
    echo [skip] !NAME! no main.go shared subpkg
  ) else (
    echo === build plugin bin: !NAME! ===
    go build -o ".pair\plugins\!NAME!\bin\!NAME!.exe" "./plugins-src/plugins/!NAME!" || echo [FAILED] !NAME!
  )
)

echo.
echo All done. Output: .pair\plugins\[name]\bin\ loaded by ctx.binary.exec, restart to apply
endlocal
