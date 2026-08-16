// ═══════════════════════════════════════════════════════════════
// tool-binary — 二进制读写 + 逆向分析（inspect_binary/write_binary/
//   binary_strings/find/patch/info/hash/entropy）
//
// 生成来源（2026-08-16）：内置 Go 工具组 → 磁盘外置插件（tool_plugin_gen.go
// 自动生成，schema 完整外置拷贝）。api 声明在插件，execute 调 ctx.binary 复用本插件目录 bin/ 下的独立二进制（源码 cmd/plugins/<name>/，改实现重编译即更换）。
// 2026-08-16：与 tool-binary-re 合并（逆向分析 6 工具并入本插件，删除独立插件）。
// 工具清单：inspect_binary、write_binary、binary_strings、binary_find、binary_patch、binary_info、binary_hash、binary_entropy
// ═══════════════════════════════════════════════════════════════
const tools = [
  {
    "name": "inspect_binary",
    "description": "分析二进制文件而不撑爆上下文：返回大小 + 嗅探类型（按 magic bytes）+ 指定区段的十六进制/ASCII 预览（hexdump 风格）。读图片/可执行/压缩包/字体等二进制用它，别用 read_file。",
    "usageGuide": "分析二进制文件：大小 + 类型嗅探（magic bytes）+ hexdump 预览。二进制文件（图片/可执行/压缩包/字体等）只能用此工具，不可用 read_file（read_file 会拒绝含 NULL 字节的文件）。比直接读原始字节安全（预览有界不撑爆上下文）。",
    "parameters": {
      "properties": {
        "length": {
          "description": "可选：预览字节数（默认 256，上限 4096）",
          "type": "integer"
        },
        "offset": {
          "description": "可选：起始字节偏移（默认 0）",
          "type": "integer"
        },
        "path": {
          "description": "文件路径（工作区内）",
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
    "name": "write_binary",
    "description": "把 base64 编码的字节写入文件（path；覆盖；父目录自动创建）。用于写二进制内容。",
    "usageGuide": "把 base64 编码的字节写入文件。用于写二进制内容（图片/字体/编译产物等）。需审核批准。比 write_file 更省 token（base64 比文本转义更紧凑）。",
    "parameters": {
      "properties": {
        "base64": {
          "description": "base64 编码的字节",
          "type": "string"
        },
        "path": {
          "description": "文件路径",
          "type": "string"
        }
      },
      "required": [
        "path",
        "base64"
      ],
      "type": "object"
    },
    "requiresApproval": true
  },
  {
    "name": "binary_strings",
    "description": "从二进制提取可打印字符串（ASCII + UTF-16LE，逆向找嵌入文本/URL/符号/提示语常用）。min_length 最短长度(默认 4)；max_results(默认 200)。返回 偏移: 字符串。",
    "usageGuide": "从二进制提取可打印字符串（ASCII + UTF-16LE）。逆向工程常用：找嵌入文本/URL/符号/提示语。比直接 search_content 更高效（跳过二进制结构直接取文本）。",
    "parameters": {
      "properties": {
        "max_results": {
          "description": "结果上限（默认 200）",
          "type": "integer"
        },
        "min_length": {
          "description": "最短字符串长度（默认 4）",
          "type": "integer"
        },
        "path": {
          "description": "文件路径",
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
    "name": "binary_find",
    "description": "在二进制里查找字节模式（hex 如 4d5a 或 'ff d8 ff'）或文本（text），返回命中字节偏移（十六进制）。hex 与 text 二选一；max_results 默认 100。",
    "usageGuide": "在二进制中按字节（hex）或文本（text）搜索模式，返回命中偏移。逆向分析用。比 search_content 更快（直接字节匹配无需文本解码）。",
    "parameters": {
      "properties": {
        "hex": {
          "description": "十六进制字节模式（与 text 二选一）",
          "type": "string"
        },
        "max_results": {
          "description": "上限（默认 100）",
          "type": "integer"
        },
        "path": {
          "description": "文件路径",
          "type": "string"
        },
        "text": {
          "description": "文本模式（与 hex 二选一）",
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
    "name": "binary_patch",
    "description": "在指定字节偏移处覆盖写入字节（hex），逆向打补丁用（如把跳转改 9090=两个 NOP）。offset 字节偏移(0 基)；hex 要写入的字节。仅覆盖、不改文件大小。",
    "usageGuide": "在指定字节偏移处覆盖写入字节（hex 编码），逆向打补丁用。仅覆盖不改文件大小。需审核批准。比手动 hex editor 更方便（自动处理偏移+hex 解析）。",
    "parameters": {
      "properties": {
        "hex": {
          "description": "要写入的字节（十六进制）",
          "type": "string"
        },
        "offset": {
          "description": "字节偏移（0 基）",
          "type": "integer"
        },
        "path": {
          "description": "文件路径",
          "type": "string"
        }
      },
      "required": [
        "path",
        "offset",
        "hex"
      ],
      "type": "object"
    },
    "requiresApproval": true
  },
  {
    "name": "binary_info",
    "description": "解析可执行文件结构（PE/ELF/Mach-O，stdlib 解析）：架构、入口、节区(名/大小/地址)、导入库与符号、导出符号——逆向起步。",
    "usageGuide": "解析可执行文件结构（PE/ELF/Mach-O）。查看区段/符号表/入口点等结构信息。比 objdump/readelf 更方便（纯 Go 实现无需外部工具）。",
    "parameters": {
      "properties": {
        "path": {
          "description": "文件路径",
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
    "name": "binary_hash",
    "description": "计算文件 大小 + MD5 + SHA1 + SHA256（识别样本/校验完整性）。流式计算，不全量载入。",
    "usageGuide": "计算文件哈希（MD5+SHA1+SHA256）。用于校验文件完整性、识别样本（从恶意软件到编译产物）。比 run_command certutil -hashfile 更方便（一次性出三种哈希）。",
    "parameters": {
      "properties": {
        "path": {
          "description": "文件路径",
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
    "name": "binary_entropy",
    "description": "按块计算香农熵（0~8）：高熵(>7.5)提示压缩/加密/加壳区段，逆向识别壳常用。chunk_size 默认 4096。",
    "usageGuide": "按块计算香农熵（0~8）。高熵(>7.5)提示压缩/加密/加壳。逆向分析查壳常用。比手动计算更快（分块扫描+视觉化结果）。",
    "parameters": {
      "properties": {
        "chunk_size": {
          "description": "块大小字节（默认 4096）",
          "type": "integer"
        },
        "path": {
          "description": "文件路径",
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
  name: 'tool-binary',
  purpose: '二进制读写 + 逆向分析（inspect_binary/write_binary/binary_strings/find/patch/info/hash/entropy，自动生成，迁移自内置 Go 工具组，含 tool-binary-re 合并）',
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
