// ═══════════════════════════════════════════════════════════════
// tool-debug — 调试工具（debug_inject_log/run_capture/analyze_output/parse_stack/cleanup_logs/watch/evaluate_session）
//
// 生成来源（2026-08-16）：内置 Go 工具组 → 磁盘外置插件（tool_plugin_gen.go
// 自动生成，schema 完整外置拷贝）。api 声明在插件，execute 调 ctx.binary 复用统一宿主二进制（.pair/plugins/tool-binary/bin/，源码 cmd/plugins/tool-binary/，承载全部内置工具组实现）。
// 工具清单：debug_inject_log、debug_run_capture、debug_analyze_output、debug_parse_stack、debug_cleanup_logs、debug_watch、debug_evaluate_session
// ═══════════════════════════════════════════════════════════════
const tools = [
  {
    "name": "debug_inject_log",
    "description": "在指定文件的指定行后插入日志输出语句。自动根据文件扩展名选择正确的日志语法。插入的日志包含 🪵 [DEBUG] 标记，后续可用 debug_cleanup_logs 统一移除。支持 Go / Python / JavaScript / TypeScript / Vue / Rust / Java / C / C++ / C# / Ruby / PHP / Swift / Kotlin / Shell / Lua / Perl / Elixir / Dart。注意：每行插入一次，插入后行号会偏移，后续注入需考虑偏移。",
    "usageGuide": "在代码指定行后插入日志输出语句（语言无关）。自动识别文件后缀选择 print/console.log/println 等。日志含 🪵 [DEBUG] 标记，后续可用 debug_cleanup_logs 清理。支持 Go/Python/JS/TS/Rust/Java/C++/C#/Ruby/PHP 等 20+ 语言。",
    "parameters": {
      "properties": {
        "file": {
          "description": "文件路径（相对于工作区根，如 'src/main.go'）",
          "type": "string"
        },
        "lines": {
          "description": "行号数组（从 1 开始），在每行之后插入日志",
          "items": {
            "type": "integer"
          },
          "type": "array"
        },
        "message": {
          "description": "可选：自定义日志消息（默认自动标注文件名+行号）",
          "type": "string"
        }
      },
      "required": [
        "file",
        "lines"
      ],
      "type": "object"
    },
    "requiresApproval": true
  },
  {
    "name": "debug_run_capture",
    "description": "运行指定命令并捕获完整输出。适用于调试场景：运行目标程序，捕获所有 stdout/stderr，报告退出码和执行耗时。与 run_command 不同：输出不截断、明确报告退出码、包含耗时统计。配合 debug_inject_log 使用：注入日志 → 运行捕获 → 分析输出。",
    "usageGuide": "运行程序并捕获完整输出（stdout+stderr+exit code+耗时）。比手动 run_command 更专注于调试场景：输出无限、报告退出码、包含耗时。支持超时控制。",
    "parameters": {
      "properties": {
        "command": {
          "description": "要执行的命令（如 'python main.py' 或 'go run .' 或 'node app.js'）",
          "type": "string"
        },
        "cwd": {
          "description": "可选：工作目录（相对于工作区根，默认根目录）",
          "type": "string"
        },
        "timeout": {
          "description": "可选：超时秒数（默认 60s，-1=不设超时）",
          "type": "integer"
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
    "name": "debug_analyze_output",
    "description": "分析程序运行输出文本，自动提取结构化信息：错误行、堆栈帧（支持多语言格式）、警告信息、panic/异常检测。返回按行组织的分析报告。不依赖任何调试器协议，纯文本分析。",
    "usageGuide": "分析程序运行输出，提取错误行、堆栈帧、警告、异常模式。返回结构化分析结果，帮助 AI 快速定位问题。配合 debug_run_capture 使用。",
    "parameters": {
      "properties": {
        "output": {
          "description": "要分析的输出文本（从 debug_run_capture 的结果中提取）",
          "type": "string"
        }
      },
      "required": [
        "output"
      ],
      "type": "object"
    },
    "readOnly": true
  },
  {
    "name": "debug_parse_stack",
    "description": "解析堆栈轨迹文本，返回结构化的帧列表。自动识别多种语言的堆栈格式：Go（goroutine）、Python（Traceback）、JS/TS（at）、Java（at）、Rust（at/panic）、C#（at ... in）。结果包含函数名、源文件、行号、列号、语言类型。可用于将运行错误链接回源代码位置。",
    "usageGuide": "解析堆栈轨迹文本为结构化数据（帧列表）。自动识别 Go/Python/JS/TS/Java/Rust/C# 等多种格式。返回函数名、文件、行号、列号。",
    "parameters": {
      "properties": {
        "text": {
          "description": "堆栈轨迹文本（从错误输出中提取）",
          "type": "string"
        }
      },
      "required": [
        "text"
      ],
      "type": "object"
    },
    "readOnly": true
  },
  {
    "name": "debug_cleanup_logs",
    "description": "移除之前通过 debug_inject_log 注入的日志语句。扫描文件中包含 🪵 [DEBUG] 标记的行并删除。可指定单个文件，或省略 file 参数自动扫描工作区内所有被注入过的文件。注意：仅移除通过 debug_inject_log 注入的日志，不影响手写的日志代码。",
    "usageGuide": "移除之前通过 debug_inject_log 注入的日志语句（包含 🪵 [DEBUG] 标记的行）。可指定单个文件或全部清理。",
    "parameters": {
      "properties": {
        "file": {
          "description": "可选：要清理的文件路径（省略则扫描工作区内所有可能被注入的文件）",
          "type": "string"
        }
      },
      "type": "object"
    },
    "requiresApproval": true
  },
  {
    "name": "debug_watch",
    "description": "监听匹配 glob 模式的文件，变更后自动执行指定命令。用于「改代码→自动跑」的调试循环。内置 2 秒轮询，500ms 去抖动。stop=true 可停止指定 id 的监听器。用 list=true 查看所有活跃监听器。",
    "usageGuide": "监听文件变更并自动重跑命令。用于「改代码→自动跑」的调试循环。指定 glob 模式匹配文件，变更后自动执行命令。内置 2s 轮询+去抖动。stop=true 停止指定 watch。",
    "parameters": {
      "properties": {
        "command": {
          "description": "文件变更后要执行的命令（如 'go test ./...' 或 'python main.py'）",
          "type": "string"
        },
        "list": {
          "description": "可选：设为 true 列出所有活跃监听器",
          "type": "boolean"
        },
        "pattern": {
          "description": "文件匹配模式（如 '**/*.go' 或 'src/**/*.py'），相对于工作区根",
          "type": "string"
        },
        "stop": {
          "description": "可选：停止指定 id 的监听器",
          "type": "string"
        },
        "timeout": {
          "description": "可选：每次运行的超时秒数（默认 120s）",
          "type": "integer"
        }
      },
      "type": "object"
    },
    "requiresApproval": true
  },
  {
    "name": "debug_evaluate_session",
    "description": "评估 agent 会话的表现，生成结构化评分报告。基于已保存的执行日志（.pair/execution_logs/）和工具调用统计进行离线分析。四个评分维度：完成度（任务是否完成）、效率（工具调用合理度）、可靠性（工具成功率）、适应性（错误恢复能力）。评分是离线分析，不消耗 agent 运行时的 token。评分结果可用于自我迭代参考。\n\n如需更高质的语义化 LLM 评分，请使用独立评分工具：\n  go run ./cmd/evaluator -conv-id \u003cconv_id\u003e -root \u003cworkspace_root\u003e\n该工具是独立项目，不依赖 agent 运行时，通过环境变量 BASE_URL/API_KEY/MODEL 配置 LLM。",
    "usageGuide": "对 agent 会话进行离线评分评估（机械公式）。如需更高质的语义化评分，请运行独立评分工具：go run ./cmd/evaluator -root \u003cworkspace\u003e。评分是离线分析，不消耗 agent 运行时的 token。",
    "parameters": {
      "properties": {
        "conv_id": {
          "description": "可选：对话 ID（省略则评估最近一次会话）",
          "type": "string"
        }
      },
      "type": "object"
    },
    "readOnly": true
  }
];

return {
  name: 'tool-debug',
  purpose: '调试工具（debug_inject_log/run_capture/analyze_output/parse_stack/cleanup_logs/watch/evaluate_session）（自动生成，迁移自内置 Go 工具组）',
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
        execute: (args) => ctx.binary.exec(t.name, args || {}, {bin: 'tool-binary'}),
      })
    }
  },
}
