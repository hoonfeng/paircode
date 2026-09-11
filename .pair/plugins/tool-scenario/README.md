# tool-scenario — 场景创造（创造模式）

> ★ 2026-09-11 新增。用户需求：「提供一个创造模式，自主根据需求创建场景工具集并保存到
> 安装目录下（就是内置场景所在）」。

## 一句话

在对话里执行 **`/创造 <需求>`** → 当前会话激活本插件（获得 `scenario_scan` /
`scenario_create` 工具）→ agent **自主**盘点现有能力、组合、创建「场景工具集」
并固化到 **安装目录 `.pair/toolsets/<name>.json`**（与内置场景 基础/调试/办公/… 同目录）——
创建后对话面板「工具集」选择器选中即生效。

## 形态（按需激活）

| 部分 | 说明 |
| --- | --- |
| `/创造 <需求>` 命令 | `ctx.activation.declare({command:'创造'})` 声明按需激活；执行后本会话激活 |
| 工具 | `scenario_scan`（盘点：插件/内置组/工具清单，`detail=full` 附描述）、`scenario_create`（组合创建 + 固化） |
| 系统提示段 | `alwaysVisible`（order 118）：引导 + 创造流程协议（未激活也可引导用户执行 /创造） |
| 工具实现 | **schema 在插件、执行在宿主**：`execute → ctx.hostTool.exec` 路由 `internal/agent/scenario_tools.go`（seam 同 tool-system） |

未激活时：工具不进 agent 工具面（宿主按会话激活状态合并）；已激活跨对话保持
（工具面每轮对话重建，其工具每轮重新合入——即时性）。

## 场景（工具集）语义

- 场景 = 工具集 = **工具面白名单**：会话选择某场景后，仅该场景声明的工具对 agent 可见。
- 组合单位：
  - 插件：`"tool-office"`（整插件）或 `{plugin:"tool-office",tools:["csv_read"]}`（白名单）；
  - 内置组：`"builtin:core"` / `{group:"core"}`（整组加入）。
- 场景名：中文或小写字母/数字/-/_；**预设名（基础/调试/办公/全栈开发/计划讨论/全功能）保留不可用**。
- 保存目录：`<InstallDir>/.pair/toolsets/`（安装目录，内置场景所在；随程序共享，跨工作区）。

## 宿主能力依赖

- `ctx.activation`（按需激活声明）、`ctx.commands`（/创造 命令）、`ctx.systemPrompt`（引导段）、
  `ctx.hostTool`（宿主执行器：scenario_scan/scenario_create Go 实现）。
- 宿主注册点：`web_server.go`（initReg）/ `agent.go`（AgentBase）——**在插件装载前注册**
  （claimTool 存档 hostExecutors；不注册进会话 reg——可见性由激活控制）。

## 验证

- Go：`go build ./...`；`internal/agent` 单测（scenario 工具注册/创建/校验）。
- 端到端：9097 测试实例 — `/创造` 激活 → `scenario_scan` 盘点 → `scenario_create`
  创建 → `.pair/toolsets/<name>.json` 落盘 → 清理。
