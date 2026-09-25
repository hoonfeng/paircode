@echo off
REM ============================================================
REM PairCode source-update detached runner.
REM Encoding: pure ASCII on purpose -- safe under both the GBK/ANSI
REM and the UTF-8 console code pages (no non-ASCII bytes at all).
REM
REM Why detached: update.mjs ends with the deploy step, which stops
REM and restarts pair.exe. If this runner lived inside the host's
REM process tree, it could be taken down together with the host.
REM Caller example (from the plugin):
REM   cmd /c start "PairCode update" cmd /c "<repo>\scripts\source-update\run-update.cmd"
REM ============================================================
setlocal EnableExtensions
set "SD=%~dp0"
for %%I in ("%SD%..\..") do set "REPO=%%~fI"
set "LOGDIR=%REPO%\temp\source-update"
set "RUNNER=%SD%update.mjs"

if not exist "%RUNNER%" (
  echo [runner] FAILED: update.mjs not found: "%RUNNER%"
  exit /b 1
)
where node >nul 2>nul
if errorlevel 1 (
  echo [runner] FAILED: node not found in PATH
  exit /b 1
)

if not exist "%LOGDIR%" mkdir "%LOGDIR%"
if not exist "%LOGDIR%" (
  echo [runner] FAILED: cannot create log dir: "%LOGDIR%"
  exit /b 1
)

REM short delay (~2s): let the UI settle after the click before work starts
ping -n 3 127.0.0.1 >nul

cd /d "%REPO%"
if errorlevel 1 (
  echo [runner] FAILED: cannot cd to "%REPO%"
  exit /b 1
)

echo [runner] start %date% %time%>> "%LOGDIR%\runner.log"
node "%RUNNER%" >> "%LOGDIR%\runner.log" 2>&1
set "RC=%ERRORLEVEL%"
echo [runner] exit=%RC% %date% %time%>> "%LOGDIR%\runner.log"
if not "%RC%"=="0" echo [runner] update failed - see "%LOGDIR%\runner.log"
exit /b %RC%
