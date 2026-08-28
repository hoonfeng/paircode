@echo off
rem ===========================================================
rem build-plugin-bins.bat — ★ 已废弃（2026-08-27）★
rem
rem 插件已全量 JS 化（.pair/plugins/<name>/index.js），执行走宿主内嵌
rem Go 内核（internal/agent/embedded_tools.go），不再需要独立二进制。
rem 旧二进制已归档 bin/legacy-plugin-bins/。
rem
rem 本脚本曾把 plugins-src/plugins/<name>/ 编译为 .pair/plugins/<name>/bin/
rem <name>.exe，会覆盖/遮蔽磁盘 JS 插件 —— 禁止再跑。
rem packager.json 管线中对应步骤已移除。新插件一律 JS 形态，勿走二进制。
rem ===========================================================
echo [DEPRECATED] build-plugin-bins.bat has been removed from the pipeline.
echo [DEPRECATED] Plugins are all JS-based now; do NOT build plugin binaries.
echo [DEPRECATED] See .pair/project-info for details.
exit /b 0
