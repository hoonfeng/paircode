// ═══════════════════════════════════════════════════════════════
// tool-bridge — 桌面桥接（bridge_status/bridge_takeover/bridge_release/bridge_exec/bridge_register_system_tool）
//
// 生成来源（2026-08-16）：内置 Go 工具组 → 磁盘外置插件（tool_plugin_gen.go
// 自动生成，schema 完整外置拷贝）。api 声明在插件，execute 调 ctx.hostTool 复用宿主 Go 执行器（对齐 harness seam：编排在插件、能力在宿主）。
// 工具清单：bridge_status、bridge_takeover、bridge_lockdown、bridge_exec、bridge_register_system_tool
// ═══════════════════════════════════════════════════════════════
const tools = [
  {
    "name": "bridge_status",
    "description": "查看桥接系统状态：当前模式（桥接/接管）、可用系统能力、审计日志摘要。用于了解 Agent 当前对系统资源的访问权限。",
    "parameters": {
      "properties": {
        "detail": {
          "description": "可选：设为 'full' 查看完整审计日志",
          "type": "string"
        }
      },
      "type": "object"
    },
    "readOnly": true
  },
  {
    "name": "bridge_takeover",
    "description": "申请全面接管系统管理权限。接管后 Agent 获得完整的系统资源访问能力：不限路径的文件系统、不限进程的命令执行、系统配置读写、服务管理等。\n接管操作需要用户审批确认。完成管理任务后请用 bridge_lockdown 归还权限。\n注意：接管赋予 Agent 高权限，请谨慎使用并只执行必要的管理操作。",
    "parameters": {
      "properties": {
        "reason": {
          "description": "接管原因说明（必填，向用户解释为何需要全面权限）",
          "type": "string"
        }
      },
      "required": [
        "reason"
      ],
      "type": "object"
    },
    "requiresApproval": true
  },
  {
    "name": "bridge_lockdown",
    "description": "归还全面接管权限，切换回安全桥接模式。桥接模式下 Agent 只能访问工作区内资源，所有文件操作、命令执行均受安全约束。调用此工具后请确认 bridge_status 验证降级成功。",
    "parameters": {
      "properties": {},
      "type": "object"
    }
  },
  {
    "name": "bridge_exec",
    "description": "通过桥接执行系统命令。行为取决于当前桥接模式：\n- 桥接模式（默认）：限工作区内目录，120s 超时（同 run_command）\n- 接管模式：不限目录，不限超时（默认 5 分钟），可执行系统管理命令\n\n建议：日常开发用 run_command（标准模式），系统管理用 bridge_exec（接管模式）。",
    "parameters": {
      "properties": {
        "command": {
          "description": "要执行的命令",
          "type": "string"
        },
        "cwd": {
          "description": "可选工作目录（接管模式下不限工作区）",
          "type": "string"
        },
        "timeout": {
          "description": "可选超时秒数（桥接模式最大 120s，接管模式最大 600s）",
          "type": "string"
        }
      },
      "required": [
        "command"
      ],
      "type": "object"
    },
    "requiresApproval": true
  },
  {
    "name": "bridge_register_system_tool",
    "description": "【接管模式专用】注册一个系统管理工具到 Agent 工具集。创建的 Go 工具可以调用系统命令、读写系统文件（不限于工作区）。仅当处于接管模式时可用。创建的工具会注册到 Agent Registry 中，持续可用直到重启。\n\n注意：name 不能与现有工具重名，handler_code 是 Go 代码片段，使用 system.Run() 执行系统命令、system.ReadFile()/WriteFile() 访问文件。",
    "parameters": {
      "properties": {
        "description": {
          "description": "工具描述（必填）",
          "type": "string"
        },
        "handler_code": {
          "description": "Go 处理函数体。可用变量: args(map[string]any), system(*BridgeController), ctx(context.Context)。示例: `name := args[\"name\"].(string); out, _ := system.ExecCommand(ctx, \"sc query \"+name, \"\", 0); return out, nil`",
          "type": "string"
        },
        "name": {
          "description": "工具名（必填，唯一标识）",
          "type": "string"
        }
      },
      "required": [
        "name",
        "description",
        "handler_code"
      ],
      "type": "object"
    },
    "requiresApproval": true
  }
];

return {
  name: 'tool-bridge',
  purpose: '桌面桥接（bridge_status/bridge_takeover/bridge_release/bridge_exec/bridge_register_system_tool）（自动生成，迁移自内置 Go 工具组）',
  apply(ctx) {
    for (const t of tools) {
      ctx.tools.register({
        name: t.name,
        description: t.description,
        usageGuide: t.usageGuide,
        category: t.category,
        readOnly: t.readOnly,
        requiresApproval: t.requiresApproval,
        systemTool: t.systemTool,
        parameters: t.parameters,
        execute: (args) => ctx.hostTool.exec(t.name, args || {}),
      })
    }
  },
}
