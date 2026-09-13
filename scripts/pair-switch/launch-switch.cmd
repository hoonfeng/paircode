@echo off
REM 脱离 agent 会话执行 pair.exe 替换（会话会因 9090 重启而中断，本脚本继续运行）
ping -n 9 127.0.0.1 >nul
powershell -NoProfile -ExecutionPolicy Bypass -File "E:\paircode-master\temp\build\switch-pair.ps1"
