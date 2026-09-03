# 内置工具 × 插件工具重合排查报告 + tool-debug 移除记录（Round4.5）

> 2026-09 · 用户指令：排查内置工具是否与插件工具有重合（不要弄重复了）；
> debug 相关工具插件若为纯命令行包装则移除（浪费上下文、无用处）。

## 一、结论先行

**工具面无"同名双注册"——不存在真重复。** 业务工具的可见面唯一注册源
= 磁盘插件（`.pair/plugins/tool-*`）；宿主进程不再注册任何业务工具。

「重复感」来自三层展示/文档的同源重叠（插件面板、工具列表、文档"内置组"描述），
不是运行时重复。

## 二、排查证据（架构定论）

1. **宿主不承载工具实现**（`internal/agent/builtin_plugins.go` 头注释，
   2026-08-16 第三轮起）：
   - `builtinPluginSpecs`（core/git/debug/codegraph…20 组）仅保留为
     **二进制实现库的组规格**（供 plugins-src 独立插件二进制经
     `RegisterToolGroups` 按组注册），宿主进程不 apply 本表。
   - 宿主唯一内置注册 = `RegisterHostFrameworkTools`：仅 SystemTool 框架组
     （update_tasks/tool_stats/history_*）。
2. **装配单源化**（`internal/agent/toolset.go` applyToolsetPlugin，R2-8）：
   磁盘插件与内嵌 code 双份时磁盘优先（内嵌降级 name-only 白名单声明）——
   DisabledTools 经可见性收敛，同名工具不存在两处注册。
3. **预设自愈**（preset_toolsets.go / toolset.go）：
   预设生成时 `diskPluginCodeAvailable(name)` 过滤缺失的 tool-* 插件；
   内置组声明未注册工具 → 运行时自动移除声明（日志可见）。

## 三、「重复感」来源与处置

| 来源 | 说明 | 处置 |
|---|---|---|
| 插件面板 34 个插件（20 个 tool-* 业务壳）与工具列表同源展示 | UI 上"插件=工具"一一对应易误读为重复 | 单源即正确形态，不改 |
| 文档/注释"内置 20 组"与磁盘 tool-* 对应 | 历史措辞 | 新文档按"磁盘插件=载体"描述 |
| 同一组工具出现在多个预设 | 预设=场景组合，属正常复用 | 无重复语义 |

## 四、语义冗余面判定 → 移除 tool-debug

`tool-debug` 是**纯命令行包装壳**：index.js 仅声明 7 个 debug_* 工具的
schema/描述，execute 一律 `ctx.binary.exec` 直通宿主内嵌内核
（embedded_tools.go registerDebugTools）——无任何组合/编排逻辑。
按用户指示移除（同类判断标准：壳内无编排逻辑、能力全在 Go 内核）。

**保留判断**：tool-git/codegraph/binary/bug/web/harness 等其他 tool-* 虽同
为薄壳，但分别承载高频核心能力（版本控制/图谱/逆向/缺陷闭环/联网/沙箱），
且部分为 JS 原生实现（有实际 JS 编排），不在本次范围。

## 五、移除全链同步清单（已验证）

| 同步点 | 动作 |
|---|---|
| `.pair/plugins/tool-debug/`（磁盘插件） | 移出 → `_temp/removed-plugins-202609/tool-debug-plugins/` |
| `.pair/publish/tool-debug/`（发布包） | 移出 → `…/tool-debug-publish/` |
| `plugins-src/plugins/tool-debug/`（独立二进制源码） | 移出 → `…/tool-debug-src/` |
| `internal/agent/tool_plugin_gen.go` genToolGroups | 删 tool-debug 组（+Round4.5 注释） |
| `internal/agent/preset_toolsets.go` presetModes | 全栈开发/调试两处引用摘除（+注释） |
| `.pair/toolsets/全栈开发.json`（18→17）/ 调试.json（12→11）/ 全功能.json（21→20） | 删 tool-debug 插件条目 |
| `internal/agent/embedded_tools.go` | 删 registerDebugTools 内嵌回退条目 |
| `internal/agent/embedded_tools_test.go` | 删 debug_watch/debug_evaluate_session 断言行 |
| `internal/agent/harness_filter_test.go` | 两处清单删 debug_inject_log |
| `internal/agent/tool_landing_test.go` | toolPluginModes 删 tool-debug 行 |
| `internal/agent/diskplugin_jsnative_test.go` | 删 TestToolDebugJSNative 函数 |
| `internal/agent/jsplugin.go` 注释 | tool-git/tool-debug/tool-bug → tool-git/tool-bug |
| `.pair/project.md` | 工具面现状段追加 Round4.5 记录 |

**保留**：`registerDebugTools` 内核实现（debug_tools.go）、builtinPluginSpecs
debug 组规格、debugtools_test.go（内核直测）——独立二进制/未来恢复复用；
release/ 归档与 docs 历史文档为历史快照，不动。

## 六、可恢复性

全部移除物备份于 `_temp/removed-plugins-202609/`（gitignore 覆盖）。
恢复路径：移回三个目录 → 重新生成插件（toolsgen 或手工）→ 预设引用按
v4 注释补回 —— 内核无需任何改动。