// ═══════════════════════════════════════════════════════════════
// tool-screenshot — 截图（screenshot_desktop/window/area/webpage）
//
// 生成来源（2026-08-16）：内置 Go 工具组 → 磁盘外置插件（tool_plugin_gen.go
// 自动生成，schema 完整外置拷贝）。api 声明在插件，execute 调 ctx.binary 复用统一宿主二进制（.pair/plugins/tool-binary/bin/，源码 cmd/plugins/tool-binary/，承载全部内置工具组实现）。
// 工具清单：screenshot_desktop、screenshot_window、screenshot_area
// ═══════════════════════════════════════════════════════════════
const tools = [
  {
    "name": "screenshot_desktop",
    "description": "截取整个桌面（所有显示器），保存为 PNG 图片到 screenshots/ 目录。返回文件路径、尺寸和截图时间。之后可用 image_analyze 分析截图中的颜色/色块/图形，或用 image_ocr 识别文字。",
    "usageGuide": "截取整个桌面（所有显示器），保存为 PNG。用于查看当前桌面状态、验证 GUI 效果。比手动按 PrintScreen 更方便（自动保存到 screenshots/ + 文件名管理）。",
    "parameters": {
      "properties": {
        "name": {
          "description": "可选：自定义文件名（不含扩展名），默认自动生成时间戳名称",
          "type": "string"
        }
      },
      "type": "object"
    },
    "readOnly": true
  },
  {
    "name": "screenshot_window",
    "description": "按窗口标题或标题子串截取特定窗口，保存为 PNG 图片到 screenshots/ 目录。返回文件路径、窗口尺寸和截图时间。如果多个窗口匹配同一标题子串，会列出所有匹配窗口供选择。",
    "usageGuide": "按窗口标题截取特定窗口，保存为 PNG。比截图整个桌面更精确（只截目标窗口）。title 支持子串匹配不区分大小写。",
    "parameters": {
      "properties": {
        "name": {
          "description": "可选：自定义文件名（不含扩展名），默认自动生成",
          "type": "string"
        },
        "title": {
          "description": "窗口标题或标题子串（不区分大小写）。例如 \"记事本\"、\"Chrome\"、\"Calculator\"",
          "type": "string"
        }
      },
      "required": [
        "title"
      ],
      "type": "object"
    },
    "readOnly": true
  },
  {
    "name": "screenshot_area",
    "description": "按坐标截取指定区域，保存为 PNG 图片到 screenshots/ 目录。区域坐标可以是绝对坐标（相对于桌面左上角），也可以是百分比（如 \"10% 20% 50% 30%\"）。返回文件路径、区域尺寸和截图时间。",
    "usageGuide": "按坐标截取指定屏幕区域。left/top/right/bottom 支持像素或百分比（如 10%）。用于截取界面局部细节。",
    "parameters": {
      "properties": {
        "bottom": {
          "description": "下边界：像素值或百分比",
          "type": "string"
        },
        "left": {
          "description": "左边界：像素值或百分比（如 \"10%\"）",
          "type": "string"
        },
        "name": {
          "description": "可选：自定义文件名",
          "type": "string"
        },
        "right": {
          "description": "右边界：像素值或百分比",
          "type": "string"
        },
        "top": {
          "description": "上边界：像素值或百分比",
          "type": "string"
        }
      },
      "required": [
        "left",
        "top",
        "right",
        "bottom"
      ],
      "type": "object"
    },
    "readOnly": true
  }
];

return {
  name: 'tool-screenshot',
  purpose: '截图（screenshot_desktop/window/area/webpage）（自动生成，迁移自内置 Go 工具组）',
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
        execute: (args) => ctx.binary.exec(t.name, args || {}),
      })
    }
  },
}
