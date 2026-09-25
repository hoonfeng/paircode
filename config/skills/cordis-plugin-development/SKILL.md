---
name: cordis-plugin-development
description: 编写/修改 cordis 动态插件（JS/TS，goja 沙箱）的完整指南：插件形态、ctx 能力全表、harness 用法、版本化工作流、常见错误排查。在 cordis(op=define) 写插件之前或 cordis(op=run) 失败后先加载本技能。
activation: auto
---

# cordis 插件开发指南

本环境支持动态插件：JS/TS 代码在 goja 沙箱中执行（对齐 deepseek-harness cordis-host-runner）。
插件代码只存在于进程内存（不落盘、跨重启不存续）；需要跨重启存续用磁盘插件包或 .pair/cordis.patch.json。

## 0. 铁律（先读这一节）

1. **自建插件 = 用户资产的持久变更**：`cordis(op=define)` 会把插件固化成磁盘插件包
   （`<InstallDir>/.pair/plugins/<name>/`，程序安装目录）——**对所有工作区生效、会出现在
   插件面板、重启自动装配**。因此：
   - **仅在用户明确要求/同意做插件时才 define**；用户没提插件的事，就不要建插件。
   - **优先复用**：先 `cordis(op=inspect)` 看已有插件、`cordis(op=services)` 看宿主服务；
     磁盘插件（tool-art / tool-rig / tool-design / tool-web / tool-harness / tool-memory …）
     与宿主工具能完成的事，一律直接调用，不要"为了省几次工具调用"另建插件代跑。
   - 临时验证用途：用完 `cordis(op=stop)` / `undefine` **并提醒用户删除插件面板条目**，
     否则会长期污染全局插件面（历史事故：agent 自建的探针插件被自动固化，用户不知情地
     多出一个插件、且其代码随后覆盖了同名磁盘插件的定义）。
2. **只读 op 不受此限**：inspect / services / query 随时可用，鼓励先查再写。
3. 磁盘插件包是插件的长期载体（真源在仓库 plugins-dist/ 或 .pair/plugins/）；
   动态插件适合"用户明确要的临时/实验能力"，不适合悄悄替用户沉淀资产。

★ 完整版用户文档：docs/plugin-development.md（ctx 全表/示例/坑）+ docs/go-core-capabilities.md（Go 内核能力清单）。
★ 写插件前先 cordis(op=services) 查精确签名；动手前可看磁盘插件现成范例（.pair/plugins/core-api、tool-memory、tool-system 等——tool-memory 是「单工具 + op 分派 + dynamicApproval」的现成范例）。

## 1. 插件形态（两种）

```js
// ① 对象形态（apply 必填）
return {
  name: 'my-plugin',
  inject: ['fs', 'web'],           // 可选：硬依赖服务（缺失→插件 waiting，服务出现自动激活）
  apply(ctx, config) {             // config 来自 cordis(op=run) 的 config 参数 / package.json "config"（无则 undefined）
    // 注册工具 / 监听事件 / 提供服务 ...
  }
}

// ② 函数形态（cordis 生态惯例 module.exports = function(ctx, config){}）
return function myPlugin(ctx, config) {
  // 函数名作插件名；匿名函数用 dyn id
}
// 函数形态可用静态属性声明硬依赖：
myPlugin.inject = ['fs']
```

## 2. ctx 能力面（★ 以源码 jsplugin.go buildContextObject 为准；下表全）

### 无条件注入成员
| 成员 | 签名 |
|---|---|
| ctx.get(name) / ctx.provide(name, obj) | 动态服务读写（provide 含函数可用；卸载自动撤销；新服务触发等待者激活） |
| ctx.on(name, fn) / ctx.emit(name, payload) | 事件订阅/广播（ui:/client: 前缀自动转发浏览器） |
| ctx.effect(fn) | 插件卸载清理 |
| ctx.timeout(fn, ms) / ctx.interval(fn, ms) | 定时器（返回 cancel；卸载自动清理） |
| ctx.tools.register({name,description,parameters,execute}) / list() | 工具注册/清单（同名冲突报错；list 只含对 cordis 可见的） |
| ctx.hostTool.exec(name, args) / .meta(name) / .names() | 宿主 Go 存档执行器（迁移模式：编排在插件、能力在宿主） |
| ctx.http.register(method, path, fn) | 旧形态接口注册（fn(req)→resp；"/*" 结尾前缀匹配；另有 list()） |
| ctx.webServer.register({kind:'exact'\|'prefix', path, handler}) | ★ 新接口推荐；Node 风格 handler(req,res)，不区分方法（自查 req.method）；兼容三参形态；req.query 是 RawQuery 字符串 |
| ctx.sse.register(path, (emit, params)=>cleanup) / ctx.ws.register(path, (conn, params)=>cleanup) | 实时推送/WebSocket（conn.send/close/onMessage） |
| ctx.kernel.routes()/installed()/total()/install([{key}]) | 内核路由表查询/挂载（卸载自动摘除；core-api 插件范例） |
| ctx.process.runBackground(cmd, cwd?)/readOutput(id)/kill(id) | 后台进程（全局单例跨轮次存活） |
| ctx.binary.exec(tool, args, {bin?, timeout?}) / .dir() | 独立插件二进制（<插件目录>/bin/<name>.exe；opts.bin 跨插件共用） |
| ctx.registerSettings({key,title,fields})/getSettings(key?)/setSettings(key, obj) | 插件配置（fields 含 name/label/type/default/hint/group/options/min/max/step/placeholder；setSettings 整对象覆盖） |
| ctx.systemPrompt.section({name,order,text}) / .variable({name,provider}) | 系统提示片段/动态变量 {{name}} |
| ctx.loopFactory.register(apply) | ★ 循环装配器**链**：多插件按注册顺序**依次叠加**（同插件名重复注册=替换该项；卸载自动摘除）。apply(opts)→overrides\|null；opts={system,stepBudget,toolCallBudget,maxToolBudgetSegments,maxContextTokens,autonomous,maxAutonomousMinutes,checkpointInterval,workspaceRoot,reviewMode}。⚠️ 旧语义「单槽位后注册整体覆盖」已废（会把先装载插件的装配参数全部吃掉） |
| ctx.loopFactory.registerLoop({...}) | JS 循环内核（宿主 Run 委托 JS 驱动；循环内可用 ctx.build(msgs, ephemeral)→callMsgs）；未注册→Go 默认循环 |
| ctx.loopFactory.registerHandoff({id?,onUserTurn?,onSegment?}) | JS 会话交接策略（用户输入/段续跑两个跨轮边界）；未注册或失败→Go 默认实现（agentloop 范例） |
| ctx.loopFactory.registerAutopilot({id?,decide}) | ★ 自主模式决策器：decide(req)→{action:'continue'\|'done',task?,assessment?}；工作 agent 自然结束时宿主调用（见 §2.1） |
| ctx.subagent.run(spec) / ctx.subagent.available() | ★ 子 agent 回合（**同步阻塞**）：spec={name,round,system,task,tools,extraTools,stepBudget,toolCallBudget,maxContextTokens,timeoutMs,resultTool,convId} → {content,segments,submitted,steps,toolCalls,promptTokens,outputTokens,durationMs,error,ended}（见 §2.1） |
| ctx.commands.register({name,description,handler}) / list() / run(name,args) | ★ slash 命令注册面（需 inject:['commands']；handler 同步、返回值须为字符串；卸载自动注销） |
| ctx.llmtrace.register(fn) / unregister(fn) | LLM 请求/响应完整追踪（fn(trace)→void；llm-trace 数据面） |
| ctx.hooks.register(event, fn) | 循环钩子：PreToolUse/PostToolUse/UserPromptSubmit/Stop；fn(payload)→{block,feedback}\|null（block 仅门事件生效，feedback 回灌 LLM） |
| ctx.activation.declare({command}) | 声明按需激活入口（用户执行该 slash 命令时激活本插件） |
| ctx.aiPresets.get(name)/list()、ctx.models.get(provider) | 预设表 / 服务商表**数据面**（Go 只给数据，展开决策在插件装配器） |
| ctx.providerFactory.register(name, factory) | provider 装配工厂（区别于 ctx.provider.register 的协议级实现） |
| ctx.prompts.provide({name,text}) / remove({name}) | 命名提示词片段提供/移除（跨插件共享） |
| ctx.handoff.{title,marker,enabled,thresholds,estimateTokens,ruleSummary,stripSystem,isHandoffText,fingerprint,keepForRelevance,parseRelevance} | 会话交接无状态能力（口径桥接 Go 单一真源，避免两侧漂移） |
| ctx.provider.register(name, impl) | ★ Provider 实现级插槽（2026-09-03 协议层重构）：注册服务商名的实现（impl(params)→Provider 实例）；同名覆盖返回还原函数（卸载自动回退） |
| ctx.toolset.registerTemplate({id,title,match?,generate}) | 工具集构建模板（generate 返回插件定义数组） |
| ctx.market.register({kind,source,name,desc})/unregister/list() | 市场源（skill/mcp/plugin） |
| ctx.registerClientMethod(method, fn) | host 半方法给浏览器 client 半远程调用（ui.invoke） |
| ctx.app.{workspaceRoot,root,folders,projectName,installDir,configDir,recentProjects,workspaceFolders} | 宿主信息（动态只读） |

### inject 声明服务（宿主内置 25 个：fs/web/bash/sse/ws/logger/timer/tools/events/store/handoff/app/workspaceRoot/kernel/market/process/mcp/skill/toolset/npm/plugins/agents/llm/http/commands）
声明后 ctx.xxx 可用；未声明访问 undefined。精确签名用 `cordis(op=services)` 查，要点：

- fs（工作区受限，越界拦截）/ web（GET，60s 超时 4MB）/ bash（120s 超时；git-bash 非 cmd）/ sse / ws
- logger(scope)→{log,info,warn,debug,error} / timer / kernel（内核路由表 install/routes/installed/total）/ market（市场源）
- process（runBackground/readOutput/kill）/ mcp（list/upsert/remove）/ skill（list/write/remove）
- toolset（list/save/remove/install；⚠️ 声明后会**覆盖**默认的 `ctx.toolset.registerTemplate` 形态，二选一）
- npm（install/uninstall/installed/checkUpdates/update）/ plugins（reloadDisk + console/env/platform/arch/version/cwd/exit/process/btoa/atob）
- agents（ready/followup：会话唤醒投递）/ commands（slash 命令面，见 ctx.commands）/ llm / http

另有静态服务：app（已无条件注入）、workspaceRoot（ctx.get('workspaceRoot')）、store（会话存储）。

## 2.1 自主模式与子 agent（★ 2026-09-21 新增，详见 docs §4.8）

「策略在插件、能力在宿主」的旗舰范例，宿主只给两个原语：

1. `ctx.loopFactory.registerAutopilot({id?, decide})`——工作 agent 自然结束时宿主回调：
   `req = {convId, workspaceRoot, round, maxRounds, objective, workerReport, workerTurns, recentHistory}`；
   返回 `{action:'continue'|'done', task?, assessment?}`（continue 时 task 必填；done/未返回/抛错 → 整轮结束）。
2. `ctx.subagent.run(spec)`——同步跑完一个**独立 agent 回合**（独立系统提示/工具白名单/来源标注），
   期间事件带 `AgentName` 实时下发前端（看板分区）。派发前先 `ctx.subagent.available()` 探测；
   子 agent 循环强制走 Go 路径、工具执行复用宿主管线（审批/预算/事件）。

```js
ctx.loopFactory.registerAutopilot({
  id: 'autopilot',
  decide: async (req) => {
    let note = '';
    if (ctx.subagent.available()) {
      const r = ctx.subagent.run({
        name: 'supervisor', round: req.round,
        system: '你是任务监督者：核查产出是否满足目标，只回事实与缺口。',
        task: `目标：${req.objective}\n汇报：${req.workerReport}`,
        tools: ['read', 'grep', 'glob'], stepBudget: 20, toolCallBudget: 20, timeoutMs: 600000,
      });
      note = r.content || '';
    }
    if (req.round >= (req.maxRounds || 3)) return { action: 'done', assessment: note };
    return { action: 'continue', task: `继续推进：${req.objective}`, assessment: note };
  },
});
```

## 2.5 工作区根解析（路径纪律 ★「产物写进别的工作区」的坑）

插件内所有路径解析（ctx.fs / ctx.bash / ctx.binary / ctx.process / grep / glob / tree …）
按以下优先级取根（宿主统一入口 ctxServiceRoot，与 binary/bash 等服务同源）：

1. 当前**工具调用会话**的工作区根（agent 执行本插件的工具时自动绑定，含并发多会话隔离）
2. UI invoke 绑定根（浏览器 client 半 `ui.invoke` 发起时刻的当前主工作区）
3. **装载期会话根**（`cordis(op=run)` 装载本插件的那个会话的工作区）
4. 插件上下文根（随宿主主工作区切换**实时更新**）
5. 全局主工作区兜底；全部取不到 → 显式报错「工作区根为空」（绝不静默落到别的工作区）

纪律：

- 要"当前正在干活的会话"的工作区 → 用**相对路径**（`out/a.png`、`src/x.ts`），别写死绝对路径。
- 要"用户当前所见工作区"（与具体会话无关的 UI 操作）→ 读 `ctx.app.workspaceRoot`（实时 accessor）。
- 要"插件自己所在目录"（读 bundle 资源/写插件私有缓存）→ 用 `ctx.binary.dir()` 或插件目录语义，别拿工作区根凑。
- 不要在 `apply`（装载）里写工程产物：apply 只做注册/调度；落盘交给工具 execute（那才有精确的会话根）。
- 排查"产物跑到别的工作区"：看 `cordis(op=inspect) id=xxx` 的 diag，确认写入发生在哪个阶段
  （apply / timer 回调 / 工具 execute）——不同阶段的根不同，见上表。

## 3. 注册工具（harness / ctx.tools）

```js
apply(ctx) {
  ctx.tools.register({
    name: 'my_tool',
    description: '做什么（给 LLM 看的说明）',
    parameters: { type: 'object', properties: { arg1: { type: 'string', description: '...' } }, required: ['arg1'] },
    async execute(args) {
      // args 是调用参数；返回 {text} 或任意 JSON 值
      return { text: '结果' }
    }
  })
}
```

- schema 校验：type 限 string/number/integer/boolean/object/array/null；`$ref` 只允许 `#/` 内部引用（防逃逸）。
- ★ 执行超时由插件自身控制（2026-08-22 起宿主不再强加 30s）：工具 execute 默认**不限时**
  （阻塞型交互工具如 ask_user 靠会话层超时）；如需死循环护栏，在工具定义上声明
  `timeout: 秒数`（如 `timeout: 30`，>0 才启用 goja Interrupt 强制中断）。
- 工具同名冲突会被拒绝（不能覆盖宿主或他人插件工具）——换名或先 cordis(op=stop) 占用方。
- ★ agent 可见性由工具集决定（对勾=加入工具集；去掉勾=从工具集移除；加入才可用）。
- `harness.defineTool(tool)` 只校验不注册；`harness.registerTool` 同 ctx.tools.register；`harness.handle(method, fn)` 注册可调用方法（Go 侧 Invoke，同 registerClientMethod）。

### Provider 实现级插槽（2026-09-03 新增）

```js
apply(ctx) {
  ctx.provider.register('my-provider', (params) => ({ // impl(params)→Provider 实例
    async chat(session) { /* ... */ },              // Chat 能力（agentloop 调用）
  }))
}
```

- 业务层统一经 `CreateProvider(params)` 创建：注册表命中 → 插件实现（新协议无需改
  Go 内核）；未命中按协议路由内置实现（anthropic-messages / openai-responses /
  openai-completions）。
- 插件卸载自动还原（注册时登记 cleanup，还原后回退 OpenAI 实现）；
  参考：internal/agent/provider_impls.go、jsplugin_providerimpl.go。

## 4. 生命周期与事件

```js
apply(ctx) {
  const cancel = ctx.on('some:event', payload => { ... })  // 订阅事件
  ctx.effect(() => { /* 插件卸载时执行清理 */ })
  ctx.provide('mySvc', { hello() { return 'world' } })  // 提供服务；卸载自动撤销
}
```

回调（事件/timer/工具 execute）在 VM 锁保护下执行，可安全访问插件闭包变量；**不要在回调外持有 goja 值跨 goroutine 使用**。

## 5. 版本化工作流（修改插件）

```text
1. cordis(op=inspect) id=xxx             → 看版本链与当前状态
2. cordis(op=inspect) id=xxx version=vN  → 读当前源码与诊断（不要凭记忆臆测）
3. 修改代码后：cordis(op=define) pluginId=xxx code=...  → 追加新版本（existing append）
4. cordis(op=run) id=xxx                 → 装载最新版（restart：自动先停旧实例）
5. 回滚：cordis(op=run) id=dyn-<旧版本号>  → 指定精确版本装载
```

- 首次 cordis(op=define) 返回的 `dyn-<n>` 就是稳定 pluginId；后续追加版本保持同一 pluginId。
- cordis(op=undefine) 删除整个插件（定义 + 磁盘包）。

## 6. 内置 cordis 运行时（CordisApi）

沙箱全局 `CordisApi`（@cordisjs/core bundle），可建真 cordis app 跑生态插件协作：

```js
return async function(ctx) {
  const app = new CordisApi.api.Context()
  app.plugin(/* cordis 生态插件 */)
  await app.start()          // 触发 ready
  // ctx.set('service', impl) / ctx.get('service') —— cordis 3 用 set 而非 provide！
}
```

Node API（require/setTimeout/fetch/process 等）沙箱中**不可用**，调用即抛引导错误——一律走 ctx 服务。

## 7. client 半（UI 插件；含 client 自动 global 作用域）

```js
// client 半：(ui) => void
(ui) => {
  ui.on(event, fn)                       // 收 host 事件（ui:/client: 前缀）
  ui.emit(event, payload)                // 发事件回 host（host: 前缀给 ctx.on('host:xxx') 消费）
  ui.invoke(plugin, method, args?)       // 远程调用 host 半 ctx.registerClientMethod 注册方法（RPC）
  ui.reportFailure(phase, message)       // 失败上报（render/guard/boot）
  ui.registerPanel({id,title,icon,render,props})  // 自定义面板（render(el, ui)，el 为容器 DOM）
  ui.registerView({id,title,icon,order,open,render})  // 主内容区独立 tab（工作台型视图；懒挂载+保持）
  ui.registerSlot({slotId, render, ...})          // 占用预定义槽位（如 titlebar-right 状态徽标）
  ui.http.get(path) / ui.http.post(path, body)   // 受限后端 API
}
// UI 槽位：single 替换型 titlebar/activitybar/sidebar/editor/right-panel/chat/statusbar
//          list 叠加型 overlay/titlebar-right/activitybar/editor-toolbar/chat-tools/statusbar-items
```

**UI 插件实战要点（2026-09，详见 `docs/plugin-development.md` §12.1–§12.3）**：

1. **状态类插件双挂载**：`titlebar-right` 槽位放常驻**徽标**（可发现性）+ `registerView`
   放**完整视图**（深度信息），徽标点击用 `layout.openViewTab(viewId)` 切视图。
2. **运行态契约**：前端「正在运行」唯一真相是 `window.__state.agentRunningByConv[convId]`
   （后端 WS **status 帧**写入）；**事实数据**（轮次/记录）一律走持久化接口，运行态只控
   视觉提示 —— 否则事件一丢界面就不自洽。后端已修「自主模式收尾后运行态残留」
   （`EventStatusRefresh`：会话真正结束时补推一次 status）。
3. **验证**：非默认端口 + 独立二进制 + **只改会话级 preset**；状态类断言必须 **2s 高密度
   采样**（运行态是瞬态，5s 采样会漏掉整个运行窗口得出错误结论），并落盘采样序列 + 截图。

## 8. 常见错误排查

| 现象 | 原因 | 处理 |
|---|---|---|
| cordis(op=run) 报"求值失败/语法错误" | JS/TS 语法或顶层异常 | 修代码 → define append → run；可用 TS 类型注解（内置编译器转译） |
| 插件进入 waiting | inject 服务未就绪 | 等提供服务方运行；或改用 ctx.get 判 undefined（可选依赖） |
| 工具注册报同名冲突 | 工具名被宿主/他人占用 | 换工具名，或先 cordis(op=stop) 占用方 |
| apply 失败（diag 可见） | 运行期异常（如调用不存在的方法） | cordis(op=inspect) id=xxx 看 diag/lastError 定位阶段 |
| 插件 stop 后工具还在 | 未走 Unload 回收 | cordis(op=stop) 正确回收；自己注册的全局资源用 ctx.effect 清理 |
| req.query 不是对象 | RawQuery 字符串 | 自行 URLSearchParams 解析 |
| ctx.bash 报 move 不存在 | 执行器是 git-bash | 用 mv/cp；中文输出注意编码 |
| ctx.fs 越界 / web.fetch 只 GET | 设计约束 | 工作区外走内核接口；POST 走 ctx.http 反向或 bash curl |
| 插件写出的文件跑到别的工作区 | 写入发生在无会话绑定的上下文（apply / timer / 事件回调）且落到了旧快照根 | 2026-09-20 起宿主已修：根随主工作区实时更新 + 装载期绑定发起会话根（见 §2.5）。仍异常时把落盘移进工具 execute，或用相对路径 + `ctx.app.workspaceRoot` 自校验 |
| 插件 HTTP 接口返回 200 但 body 为空 | handler 返回了裸对象（宿主只认字符串 body） | 显式返回字符串（`JSON.stringify(data)`，或 `{status, body, headers}` 里 body 也须为字符串） |
| UI 文本被当 HTML 解析 / 排版错乱 / 内容消失 | client 半用了 innerHTML | 模型与用户文本一律 textContent（内含 `<` `>` `&`） |
| 两个插件的装配参数互相"吃掉" | 按旧的单槽位覆盖语义理解 ctx.loopFactory.register | 2026-09-21 起为装配器**链**（按注册顺序叠加，仅同插件名替换该项） |
| ctx.subagent.run 报"无会话运行环境" | 在装载期 / 无会话回调里派发 | 先 `ctx.subagent.available()` 判，不可用则降级为纯文本判断 |
| 子 agent 回合的思考/输出变成英文 | `spec.system` 是**完整替换**——不带内核「语言锁定（中文）」铁律，也不经插件装配器链 | 在 `spec.system` 首部自带语言约束（任务书里再重申一次更稳）；参考 `.pair/plugins/autopilot/index.js` 的 `LANGUAGE_LOCK` |
| 声明 inject:['toolset'] 后 registerTemplate 变 undefined | 同键被工具集服务覆盖 | 二选一：要模板注册就别 inject toolset |

## 9. 数据纪律

- 工具 execute 返回结果有体积上限：不要返回完整大文件/大数组，只回摘要或路径引用。
- console.log 输出会写透宿主日志（带 `[cordis:id]` 标签），避免刷屏。
- 插件行为应与 bash 同等信任级别（沙箱隔离全局，但不是安全边界）。