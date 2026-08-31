---
name: cordis-plugin-development
description: 编写/修改 cordis 动态插件（JS/TS，goja 沙箱）的完整指南：插件形态、ctx 服务签名、harness 用法、版本化工作流、常见错误排查。在 cordis_define 写插件之前或 cordis_run 失败后先加载本技能。
---

# cordis 插件开发指南

本环境支持动态插件：JS/TS 代码在 goja 沙箱中执行（对齐 cordis-host-runner 运行模型）。
插件代码只存在于进程内存（不落盘、跨重启不存续）；需要跨重启存续用 `.pair/cordis.patch.json`。

## 1. 插件形态（两种）

```js
// ① 对象形态（apply 必填）
return {
  name: 'my-plugin',
  inject: ['fs', 'web'],           // 可选：硬依赖服务（缺失→插件 waiting，服务出现自动激活）
  apply(ctx, config) {             // config 来自 cordis_run 的 config 参数（无则 undefined）
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

## 2. ctx 服务签名（写插件前先用 cordis_service_list / cordis_inspect_query 查精确签名）

| 服务 | 声明后访问 | 签名 |
|---|---|---|
| fs | `ctx.fs` | `readFile(path)→string` / `writeFile(path, content)` / `appendFile` / `exists(path)→bool` / `readdir(path)→[]string` / `stat(path)→{name,size,isDir,mtime}` / `mkdir(path, recursive?)` / `rm(path, recursive?)`。路径相对工作区根，越界拦截 |
| web | `ctx.web` | `fetch(url)→{ok, status, text}`（GET，60s 超时，4MB 上限） |
| bash | `ctx.bash` | `exec(cmd, cwd?)→{output, error}`（120s 超时） |
| logger | `ctx.logger(scope)` | `{log, info, warn, debug, error}`（带插件标签写透宿主 stdout） |
| timer | `ctx.timer` | `timeout(fn, ms)→cancel` / `interval(fn, ms)→cancel`（卸载自动清理） |
| tools | `ctx.tools` | `register({name, description, parameters, execute})` / `list()` |
| events | `ctx.on(name, fn)` / `ctx.emit(name, payload)` | `ui:`/`client:` 前缀事件转发浏览器 |
| app | `ctx.app` | `workspaceRoot` |
| store | `ctx.store` | 会话存储 |

动态服务：插件 A `ctx.provide('svc', obj)` → 插件 B `ctx.get('svc')`（无则 undefined，可选服务用此法；硬依赖才用 inject 等待）。

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
- 工具同名冲突会被拒绝（不能覆盖宿主或他人插件工具）——换名或先 cordis_stop 占用方。
- `harness.defineTool(tool)` 只校验不注册；`harness.handle(method, fn)` 注册可被调用的方法（Go 侧 Invoke）。

## 4. 生命周期与事件

```js
apply(ctx) {
  const cancel = ctx.on('some:event', payload => { ... })  // 订阅事件
  // 注意：on 的 cancel 会在插件卸载时自动调用，一般无需手动
  ctx.effect(() => { /* 插件卸载时执行清理（关闭连接、释放资源） */ })
  ctx.provide('mySvc', { hello() { return 'world' } })  // 提供服务；卸载自动撤销
}
```

回调（事件/timer/工具 execute）在 VM 锁保护下执行，可安全访问插件闭包变量；**不要在回调外持有 goja 值跨 goroutine 使用**。

## 5. 版本化工作流（修改插件）

```text
1. cordis_inspect id=xxx             → 看版本链与当前状态
2. cordis_inspect id=xxx version=vN  → 读当前源码与诊断（不要凭记忆臆测）
3. 修改代码后：cordis_define pluginId=xxx code=...  → 追加新版本（existing append）
4. cordis_run id=xxx                 → 装载最新版（restart：自动先停旧实例）
5. 回滚：cordis_run id=dyn-<旧版本号>  → 指定精确版本装载
```

- 首次 cordis_define 返回的 `dyn-<n>` 就是稳定 pluginId；后续追加版本保持同一 pluginId。
- cordis_undefine 删除整个插件（全部版本）。

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

## 7. 常见错误排查

| 现象 | 原因 | 处理 |
|---|---|---|
| cordis_run 报"求值失败/语法错误" | JS/TS 语法或顶层异常 | 修代码 → define append → run；可用 TS 类型注解（内置编译器转译） |
| 插件进入 waiting | inject 服务未就绪 | 等提供服务方运行；或改用 ctx.get 判 undefined（可选依赖） |
| 工具注册报同名冲突 | 工具名被宿主/他人占用 | 换工具名，或先 cordis_stop 占用方 |
| apply 失败（diag 可见） | 运行期异常（如调用不存在的方法） | cordis_inspect id=xxx 看 diag/lastError 定位阶段 |
| 插件 stop 后工具还在 | 未走 Unload 回收 | cordis_stop 正确回收；自己注册的全局资源用 ctx.effect 清理 |

## 8. 数据纪律

- 工具 execute 返回结果有体积上限：不要返回完整大文件/大数组，只回摘要或路径引用。
- console.log 输出会写透宿主日志（带 `[cordis:id]` 标签），避免刷屏。
- 插件行为应与 bash 同等信任级别（沙箱隔离全局，但不是安全边界）。
