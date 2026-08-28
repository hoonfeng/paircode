// ═══════════════════════════════════════════════════════════════
// tool-vision — 图片提交给 LLM 视觉识别（submit_image）
//
// ★ 定位（2026-08-22）：agent 工作（测试 UI/截图/测试产出图片）时，
//   图片只落在磁盘，LLM 看不到——agent 只能用本地工具（DOM 分析等）
//   猜测画面内容。submit_image 让 agent 显式把图片随下一轮 LLM 请求
//   一起发送（OpenAI 兼容 image_url 块），LLM 直接"看到"图片：
//   识别截图文字、分析界面布局、验证 UI 渲染、描述图表等。
//
// ★ 结果标记协议（__SUBMIT_IMAGE__:json）：
//   工具执行成功后返回「标记行 + 提示文本」；Loop 层（Go）在工具结果
//   处理处解析标记 → 读图 bytes → ImagePart → 挂 pendingImages →
//   buildCallContext 注入 user 消息（Images 字段）→ Provider.Chat
//   以 image_url 块发送。标记从结果文本剥离（净化后给 LLM 的文本不含
//   标记）。协议通用：任何工具（web_debug/screenshot 等）可在结果中
//   使用同一标记实现「产图提交」。
//
// ★ 防护：图片 ≤2MiB（DeepSeek 视觉接口建议）；路径限工作区内；
//   格式 png/jpg/jpeg/gif/webp。每轮最多注入 1 张（超出丢弃并提示）。
// ═══════════════════════════════════════════════════════════════
const IMG_EXTS = ['png', 'jpg', 'jpeg', 'gif', 'webp']

function mimeOf(ext) {
  return { png: 'image/png', jpg: 'image/jpeg', jpeg: 'image/jpeg', gif: 'image/gif', webp: 'image/webp' }[ext] || 'image/jpeg'
}

const tools = [
  {
    "name": "submit_image",
    "description": "把工作区内的图片文件（截图/界面图/测试产出图等）随下一轮 LLM 请求一起提交给模型做视觉识别：识别截图文字、分析界面布局、验证 UI 渲染效果、描述图表。适用于 LLM 需要\"看到\"图片的场景（比仅用本地工具分析更准确）。支持 PNG/JPG/JPEG/GIF/WebP，单图 ≤2MiB。",
    "usageGuide": "截图/测试/网页验证产出图片后，把图片路径提交给 LLM 看：submit_image(path=图片路径, prompt=关注的问题)。适用于：① web_debug/screenshot 截图后要 LLM 确认页面渲染效果；② 测试生成图表要 LLM 分析；③ 图片是问题描述的一部分（LLM 看图排错）。",
    "parameters": {
      "type": "object",
      "properties": {
        "path": {
          "type": "string",
          "description": "图片路径（工作区内绝对路径或相对路径），支持 PNG/JPG/JPEG/GIF/WebP，≤2MiB"
        },
        "prompt": {
          "type": "string",
          "description": "可选：想让 LLM 关注的问题（默认：描述图片内容、识别文字、分析布局）"
        }
      },
      "required": ["path"]
    },
    "readOnly": true
  }
];

return {
  name: 'tool-vision',
  purpose: '图片提交给 LLM 视觉识别（submit_image：把图片随上下文提交给当前模型，OpenAI 兼容 image_url 块协议）',
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
          const pathR = String((args && args.path) || '').trim()
          if (!pathR) return '错误：缺少 path 参数（图片路径）'
          const prompt = String((args && args.prompt) || '').trim()
          // ── 路径解析：绝对（含盘符或 / 开头）直接用；相对拼工作区根 ──
          let full = pathR
          if (!(full.includes(':') || full.startsWith('/') || full.startsWith('\\'))) {
            full = (root() + '/' + pathR).replace(/[\/]+/g, '/')
          }
          // ── 校验：存在 / 非目录 / 扩展名 / 大小 ──
          let st
          try { st = ctx.fs.stat(full) } catch (e) { return '错误：图片不存在或不可访问：' + pathR }
          if (!st || st.isDir) return '错误：path 是目录：' + pathR
          const ext = (full.split('.').pop() || '').toLowerCase()
          if (!IMG_EXTS.includes(ext)) return '错误：不支持的图片格式 .' + ext + '（支持 png/jpg/jpeg/gif/webp）'
          const LIMIT = 2 * 1024 * 1024
          if (st.size > LIMIT) return '错误：图片 ' + (st.size / 1048576).toFixed(1) + 'MiB 超过 2MiB 限制——请压缩后提交（截图为局部/缩小尺寸）'
          // ── 生成结果：标记行（供 Loop 解析）+ 描述文本（给 LLM 看） ──
          const mark = JSON.stringify({
            kind: 'submit_image',
            path: full,
            mime: mimeOf(ext),
            size: st.size,
            prompt,
          })
          const kb = (st.size / 1024).toFixed(1)
          return '__SUBMIT_IMAGE__:' + mark + '\n图片已提交给模型：' + pathR + '（' + kb + 'KB，' + mimeOf(ext) + '）。请按 prompt 查看并分析图片。'
        },
      })
    }
    log.info('submit_image 工具已注册（图片随上下文提交 LLM 视觉识别）')
    return { dispose: () => {} }
  },
}
