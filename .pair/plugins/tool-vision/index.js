// ═══════════════════════════════════════════════════════════════
// tool-vision — 图像视觉（image_analyze/image_ocr）
//
// 生成来源（2026-08-16）：内置 Go 工具组 → 磁盘外置插件（tool_plugin_gen.go
// 自动生成，schema 完整外置拷贝）。api 声明在插件，execute 调 ctx.binary 复用本插件目录 bin/ 下的独立二进制（源码 cmd/plugins/<name>/，改实现重编译即更换）。
// 工具清单：image_probe、read_image、image_analyze、image_ocr
// ═══════════════════════════════════════════════════════════════
const tools = [
  {
    "name": "image_probe",
    "description": "像素级精确扫描图片（UI 调试）。直接读取像素验证：坐标颜色、区域颜色/纯色性、指定颜色元素位置、边框、对比度。基于实际像素数据，可靠性高于 image_analyze 的启发式检测。rules 为 JSON 数组字符串，逐条返回检测报告（含通过/失败判定）。",
    "usageGuide": "像素级精确扫描图片（UI 调试专用）。与 image_analyze 的启发式分析不同，本工具直接读取像素做精确验证：查坐标颜色、查区域主色/是否含某色、搜索某颜色元素位置、检测边框、算对比度。rules 为 JSON 数组字符串，每条规则 {type, ...}。支持类型：\n  - pixel: 查询坐标颜色 {type:\"pixel\", x, y}\n  - region: 区域颜色统计 {type:\"region\", x1,y1,x2,y2, color?}(\"color\" 指定则校验该色是否出现在区域并返回占比)\n  - color_search: 在全图/区域找指定颜色元素 {type:\"color_search\", color, x1?,y1?,x2?,y2?, max?}\n  - border: 检测矩形边框 {type:\"border\", x1,y1,x2,y2, side:\"top|bottom|left|right\", expect_color?, min_thickness?}\n  - contrast: 计算两色 WCAG 对比度 {type:\"contrast\", color, color2}\n每条规则返回：检测值 + 是否通过（有期望时）。定位 UI bug 时用它精确验证可疑像素，不要仅靠 image_analyze 的猜测。",
    "parameters": {
      "properties": {
        "path": {
          "description": "图片路径（工作区内）",
          "type": "string"
        },
        "rules": {
          "description": "扫描规则 JSON 数组字符串，如 [{\"type\":\"pixel\",\"x\":10,\"y\":20},{\"type\":\"region\",\"x1\":0,\"y1\":0,\"x2\":100,\"y2\":50,\"color\":\"#FFFFFF\"}]",
          "type": "string"
        }
      },
      "required": [
        "path",
        "rules"
      ],
      "type": "object"
    },
    "readOnly": true
  },
  {
    "name": "read_image",
    "description": "读取图片（PNG/JPEG/GIF）返回 { path, width, height, mediaType, bytes(base64) }。对齐 harness tool-fs read_image；gou-ide 无 attachment 服务，base64 内联返回。图片 \u003e2MB 或非支持格式明确报错。",
    "usageGuide": "读取图片文件，返回宽高/格式元信息 + base64 内容。视觉模态的模型可直接理解图片内容；否则配合 image_analyze（颜色/色块分析）、image_ocr（文字识别）使用。支持 PNG/JPEG/GIF（≤2MB）。",
    "parameters": {
      "properties": {
        "file_path": {
          "description": "图片路径（工作区内）",
          "type": "string"
        }
      },
      "required": [
        "file_path"
      ],
      "type": "object"
    },
    "readOnly": true
  },
  {
    "name": "image_analyze",
    "description": "分析图片中的颜色分布、色块区域和基本图形。输入图片路径，返回按坐标块 (x1,y1)-(x2,y2) 描述的详细分析结果。⚠ 自动检测为启发式结果：细边框/渐变/阴影/小图标/抗锯齿边缘等细节可能检测不准，返回的色块坐标与图形分类仅供参考，不得作为 UI 布局正确性的决定性证据。支持 PNG / JPEG 格式。可用于分析 UI 界面布局、色块区域、颜色构成、图形检测等视觉分析任务。",
    "usageGuide": "分析图片中的颜色分布、色块区域和基本图形。用于理解 UI 截图、图表、图像内容。比肉眼更快（自动聚类颜色+区域检测）。⚠ 注意：颜色量化与区域检测是启发式算法，UI 细节（细边框、渐变、阴影、小图标）可能检测不准。定位 UI bug 时：色块坐标仅作参考，请结合截图人工核对，或针对具体坐标用像素级验证方式确认，不要仅凭分析结果断定布局错误。",
    "parameters": {
      "properties": {
        "detail": {
          "description": "可选：分析详细程度，\"high\"（详细）或 \"low\"（概览），默认 \"high\"",
          "type": "string"
        },
        "max_colors": {
          "description": "可选：最大颜色聚类数，默认 8，范围 1-32",
          "type": "integer"
        },
        "path": {
          "description": "图片路径（工作区内）",
          "type": "string"
        }
      },
      "required": [
        "path"
      ],
      "type": "object"
    },
    "readOnly": true
  },
  {
    "name": "image_ocr",
    "description": "从图片中识别文字（OCR）。返回识别出的文字内容、坐标位置 (x1,y1)-(x2,y2) 及置信度。⚠ 识别能力有限：小字号、低对比度、深色主题、抗锯齿文字等场景错误率较高，输出会标注每行置信度，置信度 \u003c60 的结果仅供参考，不得作为确定性结论。支持项目内嵌的 Tesseract 便携版（无需安装），也支持系统已安装的 Tesseract。支持 PNG / JPEG 格式。可用 lang 参数指定语言，如 \"chi_sim+eng\"（中英文混合）、\"eng\"（仅英文）。",
    "usageGuide": "从图片中识别文字（OCR）。支持中英文混合识别。截图后识别界面上的文字内容用此工具。⚠ 注意：OCR 结果可能有误（尤其小字号/低对比度/深色主题），输出含每行置信度。识别 UI 文本时请重点看置信度高的行；关键信息如按钮文案、报错内容，若置信度低或不确定，应重新截图放大后再识别，或用其他方式交叉验证（如查看对应源码/HTML），不要直接采信低置信度结果。",
    "parameters": {
      "properties": {
        "detail": {
          "description": "可选：是否返回详细的坐标信息，默认 true",
          "type": "boolean"
        },
        "lang": {
          "description": "可选：识别语言，如 \"chi_sim+eng\"（中英文）、\"eng\"（仅英文），默认 \"chi_sim+eng\"",
          "type": "string"
        },
        "path": {
          "description": "图片路径（工作区内）",
          "type": "string"
        }
      },
      "required": [
        "path"
      ],
      "type": "object"
    },
    "readOnly": true
  }
];

return {
  name: 'tool-vision',
  purpose: '图像视觉（image_analyze/image_ocr）（自动生成，迁移自内置 Go 工具组）',
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
