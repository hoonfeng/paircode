// ═══════════════════════════════════════════════════════════════
// tool-vision-llm — DeepSeek 视觉模型图片理解（双模式 + 配置面板注册）
//
// 依据 DeepSeek 官方文档（https://api-docs.deepseek.com/zh-cn/guides/vision）：
//   - 模型 deepseek-v4-flash-vision-exp 支持图片输入（OpenAI 兼容格式）
//   - content 为块数组：[{type:'text',text}, {type:'image_url',image_url:{url:'data:<mime>;base64,<b64>'}}]
//   - 图片仅限 user 消息；支持 PNG/JPEG/GIF/WebP；单图 ≤32MiB；每图 ≤384 token
//   - detail 可选：low(512×512 省 token)/high/original/auto
//
// ★ 配置面板注册（2026-08-21 v4）：ctx.registerSettings 注册「视觉理解（vision）」
//   配置段（apiKey/baseURL/model），前端设置面板动态渲染；execute 读面板配置
//   （ctx.getSettings 优先）+ 装载配置（config）兜底。
// ★ 双模式：
//   模式 A（纯 goja）：宿主有 ctx.fs.readFileBase64 + ctx.web.post → 直接 JS 调用，
//     天然跨平台（Windows/Linux/macOS），无外部进程。
//   模式 B（脚本回退）：旧宿主（无上述能力）→ PowerShell（Windows）/ python3（Linux）
//     脚本，参数 base64 内嵌零转义，临时脚本写 tmp/ 执行后删除。
// ═══════════════════════════════════════════════════════════════
return {
  name: 'tool-vision-llm',
  purpose: 'DeepSeek 视觉模型图片理解（deepseek-v4-flash-vision-exp）：描述图片/识别截图文字/分析图表界面。双模式（纯 goja / 脚本回退），跨平台，配置面板注册（apiKey/baseURL/model）',
  inject: ['fs', 'web', 'bash', 'logger'],
  apply(ctx, config) {
    const log = (ctx.logger ? ctx.logger('vision-llm') : { info() {}, warn() {}, error() {} })
    const root = () => ctx.workspaceRoot || (ctx.app && ctx.app.workspaceRoot) || ''

    // ── 配置面板注册（前端设置面板动态渲染 tab）──
    try {
      ctx.registerSettings({
        key: 'tool-vision-llm',
        title: '视觉理解（vision）',
        fields: [
          { name: 'apiKey', label: 'API Key', type: 'password', hint: 'DeepSeek 官方 API Key（或支持视觉模型的兼容服务商）' },
          { name: 'baseURL', label: 'Base URL', type: 'text', default: 'https://api.deepseek.com', hint: 'OpenAI 兼容端点，如 https://api.deepseek.com' },
          { name: 'model', label: '视觉模型', type: 'text', default: 'deepseek-v4-flash-vision-exp', hint: '视觉模型名（仅此模型接受图片）' },
        ],
      })
    } catch (e) { log.warn('registerSettings 失败: ' + e) }

    // ── base64（goja 无 Buffer；TextEncoder 全局可用）──
    const B64CH = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/'
    const b64encode = (s) => {
      const bytes = new TextEncoder().encode(String(s))
      let b64 = ''
      for (let i = 0; i < bytes.length; i += 3) {
        const b0 = bytes[i], b1 = bytes[i + 1], b2 = bytes[i + 2]
        b64 += B64CH[b0 >> 2]
        b64 += B64CH[((b0 & 3) << 4) | (b1 === undefined ? 0 : b1 >> 4)]
        if (b1 === undefined) { b64 += '=='; break }
        b64 += B64CH[((b1 & 15) << 2) | (b2 === undefined ? 0 : b2 >> 6)]
        if (b2 === undefined) { b64 += '='; break }
        b64 += B64CH[b2 & 63]
      }
      return b64
    }

    // ── 模式检测 ──
    const hasGoja = typeof ctx.fs.readFileBase64 === 'function' && typeof ctx.web.post === 'function'

    // ── 模式 B：PowerShell 脚本（Windows，参数 base64 内嵌）──
    const psScript = (path, prompt, key, base, model, detail) => {
      const b = b64encode
      return `$ProgressPreference='SilentlyContinue'
$path=[Text.Encoding]::UTF8.GetString([Convert]::FromBase64String('${b(path)}'))
$prompt=[Text.Encoding]::UTF8.GetString([Convert]::FromBase64String('${b(prompt)}'))
$key=[Text.Encoding]::UTF8.GetString([Convert]::FromBase64String('${b(key)}'))
$base=[Text.Encoding]::UTF8.GetString([Convert]::FromBase64String('${b(base)}'))
$model=[Text.Encoding]::UTF8.GetString([Convert]::FromBase64String('${b(model)}'))
$detail=[Text.Encoding]::UTF8.GetString([Convert]::FromBase64String('${b(detail)}'))
$ext=[IO.Path]::GetExtension($path).ToLower()
$mime=switch($ext){'.png'{'image/png'}'.jpg'{'image/jpeg'}'.jpeg'{'image/jpeg'}'.gif'{'image/gif'}'.webp'{'image/webp'}default{'image/jpeg'}}
if(-not [IO.File]::Exists($path)){Write-Output '__VISION_ERROR__ 图片不存在: '+$path;exit 1}
$bytes=[IO.File]::ReadAllBytes($path)
if($bytes.Length -gt 33554432){Write-Output '__VISION_ERROR__ 图片超过 32MiB 限制';exit 1}
$b64=[Convert]::ToBase64String($bytes)
if($detail){$img=@{url="$mime;base64,$b64";detail=$detail}}else{$img=@{url="$mime;base64,$b64"}}
$body=@{model=$model;messages=@(@{role='user';content=@(@{type='text';text=$prompt},@{type='image_url';image_url=$img})})}|ConvertTo-Json -Depth 12
try{
  $resp=Invoke-RestMethod -Uri ($base.TrimEnd('/')+'/chat/completions') -Method Post -Headers @{Authorization="Bearer $key"} -ContentType 'application/json' -Body $body -TimeoutSec 110
  if($resp.choices[0].message.content){Write-Output $resp.choices[0].message.content}else{Write-Output '__VISION_ERROR__ 模型未返回内容'}
}catch{Write-Output ('__VISION_ERROR__ '+$_.Exception.Message);exit 1}`
    }

    // ── 模式 B：python3 脚本（Linux，参数 base64 内嵌）──
    const pyScript = (path, prompt, key, base, model, detail) => {
      const b = b64encode
      return `import base64,json,urllib.request,sys,os
path=base64.b64decode('${b(path)}').decode()
prompt=base64.b64decode('${b(prompt)}').decode()
key=base64.b64decode('${b(key)}').decode()
base=base64.b64decode('${b(base)}').decode()
model=base64.b64decode('${b(model)}').decode()
detail=base64.b64decode('${b(detail)}').decode()
if not os.path.isfile(path):
  print('__VISION_ERROR__ 图片不存在: '+path);sys.exit(1)
ext=path.lower().rsplit('.',1)[-1] if '.' in path else ''
mime={'png':'image/png','jpg':'image/jpeg','jpeg':'image/jpeg','gif':'image/gif','webp':'image/webp'}.get(ext,'image/jpeg')
data=open(path,'rb').read()
if len(data)>33554432:
  print('__VISION_ERROR__ 图片超过 32MiB 限制');sys.exit(1)
b64=base64.b64encode(data).decode()
img={"url":"data:%s;base64,%s"%(mime,b64)}
if detail: img["detail"]=detail
body=json.dumps({"model":model,"messages":[{"role":"user","content":[{"type":"text","text":prompt},{"type":"image_url","image_url":img}]}]})
req=urllib.request.Request(base.rstrip('/')+'/chat/completions',data=body.encode(),headers={'Authorization':'Bearer '+key,'Content-Type':'application/json'})
try:
  resp=urllib.request.urlopen(req,timeout=110)
  out=json.loads(resp.read().decode())
  c=out.get('choices',[{}])[0].get('message',{}).get('content')
  print(c if c else '__VISION_ERROR__ 模型未返回内容')
except Exception as e:
  print('__VISION_ERROR__ '+str(e));sys.exit(1)`
    }

    // ── 模式 B 执行（脚本回退）──
    const execScript = (path, prompt, apiKey, baseURL, model, detail) => {
      let isLinux = false
      try { const u = ctx.bash.exec('uname -s'); isLinux = !(u && u.error) } catch (e) {}
      const script = isLinux ? pyScript(path, prompt, apiKey, baseURL, model, detail) : psScript(path, prompt, apiKey, baseURL, model, detail)
      const tmp = root() + '/tmp/vision_llm_' + Date.now() + (isLinux ? '.py' : '.ps1')
      try {
        ctx.fs.writeFile(tmp, script)
        const cmd = isLinux ? 'python3 "' + tmp + '"' : 'powershell -NoProfile -NonInteractive -File "' + tmp + '"'
        const r = ctx.bash.exec(cmd)
        const out = (r && r.output || '').trim()
        if (out.startsWith('__VISION_ERROR__')) return '视觉分析失败：' + out.slice(17)
        if (!out) return '视觉分析失败：无输出' + (r && r.error ? '（' + r.error + '）' : '')
        return out
      } catch (e) {
        return '视觉分析失败：' + String(e && e.message || e)
      } finally {
        try { ctx.fs.rm(tmp) } catch (e) {}
      }
    }

    // ── 模式 A 执行（纯 goja）──
    const execGoja = (path, prompt, apiKey, baseURL, model, detail) => {
      let b64 = ''
      try { b64 = ctx.fs.readFileBase64(path) } catch (e) { return '错误：读取图片失败：' + String(e && e.message || e) }
      if (!b64) return '错误：图片为空或不存在'
      const bytes = Math.floor(b64.length * 3 / 4)
      if (bytes > 32 * 1024 * 1024) return '错误：图片超过 32MiB 限制（当前约 ' + Math.round(bytes / 1048576) + ' MiB）'
      const ext = path.toLowerCase().split('.').pop()
      const mime = { png: 'image/png', jpg: 'image/jpeg', jpeg: 'image/jpeg', gif: 'image/gif', webp: 'image/webp' }[ext] || 'image/jpeg'
      const img = { url: 'data:' + mime + ';base64,' + b64 }
      if (detail) img.detail = detail
      const body = JSON.stringify({
        model,
        messages: [{ role: 'user', content: [{ type: 'text', text: prompt }, { type: 'image_url', image_url: img }] }]
      })
      try {
        const r = ctx.web.post(baseURL + '/chat/completions', { 'Content-Type': 'application/json', 'Authorization': 'Bearer ' + apiKey }, body)
        if (!r || !r.text) return '错误：API 无响应'
        let j = null
        try { j = JSON.parse(r.text) } catch (e) {}
        if (!r.ok) return '错误：API ' + r.status + '：' + (j && j.error && j.error.message || r.text.slice(0, 300))
        const content = j && j.choices && j.choices[0] && j.choices[0].message && j.choices[0].message.content
        if (!content) return '错误：模型未返回内容'
        return content
      } catch (e) {
        return '错误：API 调用失败：' + String(e && e.message || e)
      }
    }

    // ── 工具注册 ──
    ctx.tools.register({
      name: 'vision',
      description: '用 DeepSeek 视觉模型（deepseek-v4-flash-vision-exp）理解图片内容：描述图片内容、识别截图中的文字、分析图表/界面布局。输入图片路径（工作区内），返回模型的文字描述。支持 PNG/JPEG/GIF/WebP（≤32MiB）。跨平台（Windows/Linux/macOS）。API 配置在「设置 → 视觉理解（vision）」面板（apiKey/baseURL/model，默认 DeepSeek 官方）。',
      parameters: {
        type: 'object',
        properties: {
          path: { type: 'string', description: '图片路径（工作区内绝对路径或相对路径），支持 PNG/JPEG/GIF/WebP，≤32MiB' },
          prompt: { type: 'string', description: '可选：自定义问题（默认「请详细描述这张图片的内容，包括其中的文字、图表、界面元素等」）' },
          detail: { type: 'string', description: '可选：low（512×512 省 token）/ high / original / auto（默认 auto）' }
        },
        required: ['path']
      },
      execute: async (args) => {
        const path = String(args.path || '').trim()
        if (!path) return '错误：缺少 path 参数（图片路径）'
        const prompt = String(args.prompt || '').trim() || '请详细描述这张图片的内容，包括其中的文字、图表、界面元素等'
        const detail = String(args.detail || '').trim()
        // 配置读取：面板配置（ctx.getSettings）优先，装载配置（config）兜底
        let s = {}
        try { s = (typeof ctx.getSettings === 'function') ? (ctx.getSettings('tool-vision-llm') || {}) : {} } catch (e) {}
        const cfg = config || {}
        const apiKey = s.apiKey || cfg.apiKey || ''
        const baseURL = String(s.baseURL || cfg.baseURL || 'https://api.deepseek.com').replace(/\/+$/, '')
        const model = s.model || cfg.model || 'deepseek-v4-flash-vision-exp'
        if (!apiKey) return '错误：未配置 apiKey——请在「设置 → 视觉理解（vision）」面板填写 API Key'
        // 路径解析：绝对（含盘符或 / 开头）直接用，否则拼工作区根
        const full = (path.includes(':') || path.startsWith('/')) ? path : (root() + '/' + path).replace(/\\/g, '/')
        return hasGoja ? execGoja(full, prompt, apiKey, baseURL, model, detail) : execScript(full, prompt, apiKey, baseURL, model, detail)
      }
    })
    log.info('vision 工具已注册（模式 ' + (hasGoja ? 'A 纯 goja' : 'B 脚本回退') + '，配置面板已注册「视觉理解（vision）」）')
    return { dispose: () => {} }
  },
}