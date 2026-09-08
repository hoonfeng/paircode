// ═══════════════════════════════════════════════════════════════
// tool-vision — read_image：把图片读给模型看（视觉识别）
//
// ★ 定位（2026-08-22 建，2026-09 对齐 dsh read_image）：
//   agent 工作（测试 UI/截图/测试产出图片）时，图片只落在磁盘，LLM 看不到——
//   agent 只能用本地工具（DOM 分析等）猜测画面内容。read_image 让 agent
//   显式把图片随下一轮 LLM 请求一起发送，LLM 直接"看到"图片：
//   识别截图文字、分析界面布局、验证 UI 渲染、描述图表等。
//
// ★ 结果标记协议（__SUBMIT_IMAGE__:json）：
//   工具执行成功后返回「标记行 + 可选提示文本」；Loop 层（Go，internal/agent/
//   image_submit.go）解析标记 → 准入检查 → 归一化（对齐 dsh attachment 管线：
//   2048²/4MiB）→ 内容寻址落盘 → 挂 pendingImages → buildCallContext 装配时
//   按路由预算投影（DeepSeek 640k 像素/1MiB）并生成 dsh 风格信封文本。
//   标记从结果文本剥离（净化后给 LLM 的文本不含标记）。
//   协议通用：任何工具（web_debug/screenshot 等）可在结果中使用同一标记实现「产图提交」。
//
// ★ 防护：路径限工作区内（Go 侧 resolvePath 二次校验）；准入 20MiB /
//   64M 像素 / 长边 8192；格式 png/jpg/jpeg/gif/webp（也接受无扩展名，按内容嗅探）；
//   路径去重 + 会话内总数上限；非多模态模型由 Go 侧门控禁用本工具。
// ═══════════════════════════════════════════════════════════════
const IMG_EXTS = ['png', 'jpg', 'jpeg', 'gif', 'webp']
const MAX_BYTES = 20 * 1024 * 1024 // 对齐 dsh 准入上限（DEFAULT_MAX_REQUEST_IMAGE_BYTES）

function mimeOf(ext) {
  return { png: 'image/png', jpg: 'image/jpeg', jpeg: 'image/jpeg', gif: 'image/gif', webp: 'image/webp' }[ext] || 'image/jpeg'
}

const tools = [
  {
    "name": "read_image",
    "description": "读取工作区内的 PNG/JPEG/WebP/GIF 图片文件并把图片本身交给模型看（随下一轮请求一起发送）：识别截图文字、分析界面布局、验证 UI 渲染效果、描述图表。路径可以没有扩展名（按文件内容识别格式），也可以直接传归一化附件路径。Harness 会在下次模型请求前校验并降采样过大的图片，所以直接用本工具即可，不要为了看图片而安装图像库或生成缩略图。独立文件可以小批并发读取。要求当前模型支持图片输入。",
    "usageGuide": "截图/测试/网页验证产出图片后，把图片路径交给 LLM 看：read_image(file_path=图片路径, prompt=关注的问题)。适用于：① web_debug/screenshot 截图后要 LLM 确认页面渲染效果；② 测试生成图表要 LLM 分析；③ 图片是问题描述的一部分（LLM 看图排错）。",
    "parameters": {
      "type": "object",
      "properties": {
        "file_path": {
          "type": "string",
          "description": "图片文件路径（工作区内绝对路径或相对路径），支持 PNG/JPEG/WebP/GIF"
        },
        "prompt": {
          "type": "string",
          "description": "可选：想让 LLM 关注的问题（默认：描述图片内容、识别文字、分析布局）"
        }
      },
      "required": ["file_path"]
    },
    "readOnly": true
  }
];

return {
  name: 'tool-vision',
  purpose: '图片读给模型看（read_image：把图片随上下文交给当前模型，OpenAI 兼容 image_url 块协议；结果标记 __SUBMIT_IMAGE__ 由 Loop 层解析注入）',
  inject: ['fs', 'logger'],
  apply(ctx) {
    const log = (ctx.logger ? ctx.logger('tool-vision') : { info() {}, warn() {}, error() {} })
    const root = () => ctx.workspaceRoot || ''

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
        execute: (args) => {
          // 参数：file_path（对齐 dsh）；兼容旧名 path
          const pathR = String((args && (args.file_path || args.path)) || '').trim()
          if (!pathR) return 'file_path must be a non-empty string'
          const prompt = String((args && args.prompt) || '').trim()
          // ── 路径解析：绝对（含盘符或 / 开头）直接用；相对拼工作区根 ──
          let full = pathR
          if (!(full.includes(':') || full.startsWith('/') || full.startsWith('\\'))) {
            full = (root() + '/' + pathR).replace(/[\/]+/g, '/')
          }
          // ── 校验：存在 / 非目录 / 扩展名 / 大小 ──
          let st
          try { st = ctx.fs.stat(full) } catch (e) { return '错误：图片不存在或不可访问：' + pathR }
          if (!st || st.isDir) return '错误：file_path 是目录：' + pathR
          const dot = full.lastIndexOf('.')
          const slash = Math.max(full.lastIndexOf('/'), full.lastIndexOf('\\'))
          const ext = dot > slash ? full.slice(dot + 1).toLowerCase() : ''
          // 无扩展名也接受（对齐 dsh：按文件内容识别格式）；有扩展名必须是支持的图片格式
          if (ext && !IMG_EXTS.includes(ext)) {
            return '错误：无法读取 "' + pathR + '"，.' + ext + ' 扩展名不是受支持的图片格式；read_image 接受 PNG/JPEG/WebP/GIF（无扩展名的文件同样按内容识别）'
          }
          if (st.size > MAX_BYTES) {
            return '错误：图片 ' + (st.size / 1048576).toFixed(1) + 'MiB 超过 ' + (MAX_BYTES / 1048576) + 'MiB 准入上限——请缩小尺寸或压缩后重试'
          }
          // ── 生成结果：标记行（供 Loop 解析）+ 可选关注点（信封由 Go 层生成） ──
          const mark = JSON.stringify({
            kind: 'read_image',
            path: full,
            mime: ext ? mimeOf(ext) : '',
            size: st.size,
            prompt,
          })
          return '__SUBMIT_IMAGE__:' + mark + '\n' + (prompt ? '关注点：' + prompt : '')
        },
      })
    }
    log.info('read_image 工具已注册（图片随上下文交给 LLM 视觉识别）')
    return { dispose: () => {} }
  },
}
