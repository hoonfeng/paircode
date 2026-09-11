// tool-scenario 冒烟（Node 仿真宿主 ctx；不走 Go 服务）
// 用法: node smoke-test.js
// 覆盖: 装载 → 按需激活声明(/创造) → 工具注册(scenario_scan/scenario_create)
//       → 命令指令(含/不含需求) → 提示段(alwaysVisible) → 工具 execute → hostTool 路由
'use strict'
const fs = require('fs')
const path = require('path')

const registered = []
const commands = []
const sections = []
const activated = []
const hostCalls = []

const ctx = {
  logger: () => ({ log() {}, info() {}, warn() {}, error() {}, debug() {} }),
  activation: { declare: (o) => { activated.push(o); return { ok: true } } },
  commands: { register: (c) => commands.push(c) },
  systemPrompt: { section: (s) => sections.push(s) },
  tools: { register: (t) => registered.push(t) },
  hostTool: { exec: (name, args) => { hostCalls.push({ name, args }); return 'HOST:' + name } },
}

const src = fs.readFileSync(path.join(__dirname, 'index.js'), 'utf8')
const fn = new Function('ctx', 'process', 'console', src)
const plugin = fn(ctx, process, console)
console.log('插件:', plugin && plugin.name, '| inject:', JSON.stringify(plugin.inject))
plugin.apply(ctx)

const toolNames = registered.map((t) => t.name).sort()
console.log('工具(%d): %s', registered.length, toolNames.join(', '))
if (toolNames.join(',') !== 'scenario_create,scenario_scan') {
  console.error('工具面异常')
  process.exit(1)
}
if (!activated.some((a) => a.command === '创造')) {
  console.error('未声明按需激活 /创造')
  process.exit(1)
}
console.log('按需激活声明:', JSON.stringify(activated[0]))

const cmd = commands.find((c) => c.name === '创造')
if (!cmd) {
  console.error('未注册 /创造 命令')
  process.exit(1)
}
const out1 = cmd.handler({ args: '音视频处理场景' })
if (!/音视频处理场景/.test(out1) || !/scenario_create/.test(out1) || !/scenario_scan/.test(out1)) {
  console.error('命令指令异常: ' + out1)
  process.exit(1)
}
const out2 = cmd.handler({ args: '' })
if (!/未给出/.test(out2)) {
  console.error('空需求指令异常: ' + out2)
  process.exit(1)
}
console.log('命令指令 OK（含需求 %d 字 / 空需求提示）', out1.length)

const sec = sections.find((s) => s.name === 'tool-scenario')
if (!sec || !sec.alwaysVisible || sec.order !== 118) {
  console.error('提示段异常: ' + JSON.stringify(sec && { order: sec.order, alwaysVisible: sec.alwaysVisible }))
  process.exit(1)
}
console.log('提示段 OK（order=%d alwaysVisible=%s）', sec.order, sec.alwaysVisible)

const r = registered.find((t) => t.name === 'scenario_scan').execute({ detail: 'full' })
if (r !== 'HOST:scenario_scan' || hostCalls[0].name !== 'scenario_scan') {
  console.error('hostTool 路由异常: ' + r + ' / ' + JSON.stringify(hostCalls))
  process.exit(1)
}
const r2 = registered.find((t) => t.name === 'scenario_create').execute({ name: 'x' })
if (r2 !== 'HOST:scenario_create') {
  console.error('hostTool 路由异常(create): ' + r2)
  process.exit(1)
}
console.log('hostTool 路由 OK（scan/create → 宿主执行器）')

console.log('\n✅ tool-scenario 冒烟全部通过')
