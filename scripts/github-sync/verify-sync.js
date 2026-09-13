// 阶段5：同步后验证 —— 文件到位/已清理/运行数据完好/备份完好/remote 可用
const { execFileSync } = require('child_process')
const fs = require('fs'), path = require('path')
const WT = process.env.SYNC_WT || 'E:\\paircode-master'
function git(args){
  try{ const o=execFileSync('git',args,{cwd:WT,encoding:'utf8',timeout:180000,maxBuffer:1<<26,stdio:['ignore','pipe','pipe']}); return {ok:true,out:(o||'').trim()} }
  catch(e){ return {ok:false,out:String(e.stderr||e.message||'').trim().slice(0,300)} }
}
const E = p => fs.existsSync(path.join(WT,p))
let pass=0, fail=0
function check(desc, cond, extra){ (cond?pass++:fail++); console.log((cond?'  ✓ ':'  ✗ ')+desc+(extra?'   '+extra:'')) }
console.log('===== 1) master 新增文件是否到位 =====')
for(const f of ['internal/agent/handoff.go','internal/agent/apply_patch.go','internal/agent/run_stats.go','internal/agent/image_pipeline.go','internal/agent/llmtrace.go','internal/agent/scenario_tools.go','internal/agent/tool_budget.go','internal/core/pair_migrate.go','scripts/overwrite-install.mjs','scripts/verify-release.mjs','dev/agent-teams-goal-snapshot.md','dev/tool_plugin_gen/main.go'])
  check(f, E(f))
console.log('\n===== 2) 旧残留是否已清理 =====')
for(const p of ['.pair/plugins/tool-core','.pair/plugins/tool-git','.pair/plugins/tool-vision','.pair/plugins/web-api','config/philosophy','internal/agent/parallel_tools.go','internal/agent/subagent_registry.go','cmd/companion/subagent_spawn.go','plugins-src/plugins/tool-debug','docs/project-knowledge.md'])
  check(p+' 已移除', !E(p))
console.log('\n===== 3) 前端源码已是 master 版本 =====')
const uiState = fs.readFileSync(path.join(WT,'plugins-src/ui-app/src/ui-state.js'),'utf8')
check('ui-state.js 不再含 setFocusMode（本地改动标识）', !uiState.includes('setFocusMode'))
check('ui-state.js 含 master 的 runStatsByConv 注释', uiState.includes('run-stats.json') || uiState.includes('后端'))
const appAct = fs.readFileSync(path.join(WT,'plugins-src/ui-app/src/app-actions.js'),'utf8')
check('app-actions.js 不再含 convListVisible', !appAct.includes('convListVisible'))
console.log('\n===== 4) 构建产物（.pair 下的壳与区域包）=====')
const webAssets = fs.readdirSync(path.join(WT,'.pair/assets/runtime/web/assets'))
console.log('  壳产物:', webAssets.join(', '))
const rpAssets = fs.readdirSync(path.join(WT,'.pair/plugins/ui-right-panel/assets'))
console.log('  ui-right-panel 产物:', rpAssets.join(', '))
console.log('\n===== 5) 本地运行数据是否完好（git 忽略区）=====')
for(const p of ['.pair/memory','.pair/conversations','.pair/tasks','.pair/project-info','.pair/pair.db','.pair/project.md','.pair/token-stats.json'])
  check(p, E(p))
console.log('\n===== 6) 备份是否完好 =====')
const bks = fs.readdirSync(path.join(WT,'.pair/.local-changes-backup'))
console.log('  备份目录:', bks.join(', '))
let bkFiles=0; (function w(d){ for(const e of fs.readdirSync(d,{withFileTypes:true})){ const p=path.join(d,e.name); if(e.isDirectory()) w(p); else bkFiles++ } })(path.join(WT,'.pair/.local-changes-backup',bks[bks.length-1]))
check('备份文件数 = 246（245 文件 + MANIFEST）', bkFiles===246, '实际 '+bkFiles)
console.log('\n===== 7) 与 master 的一致性 =====')
const st = git(['status','--porcelain'])
const lines = st.ok?st.out.split('\n').filter(Boolean):[]
const onlyTemp = lines.every(l=>l.includes('temp/'))
check('工作区无内容差异（仅剩未跟踪的 temp/）', onlyTemp, lines.length?lines.join(' | '):'（完全干净）')
const diff = git(['diff','--stat','HEAD'])
check('git diff HEAD 为空', !diff.out)
const head = git(['log','-1','--format=%H'])
check('HEAD == master 最新 e6c1e625', head.out.startsWith('e6c1e625'), head.out.slice(0,8))
console.log('\n===== 8) remote 可用性（走代理 fetch 探测）=====')
const fr = git(['ls-remote','--heads','origin','master'])
check('origin(代理) 可访问', fr.ok && /refs\/heads\/master/.test(fr.out), fr.ok?fr.out.split('\t')[0].slice(0,12):fr.out)
console.log('\n===== 9) 为 temp/ 加本地私有忽略（不改仓库文件）=====')
const exclPath = path.join(WT,'.git/info/exclude')
let excl = fs.readFileSync(exclPath,'utf8')
if(!excl.includes('temp/')){
  if(!excl.endsWith('\n')) excl+='\n'
  excl += '# 本工作区临时目录（同步工具产物，不入库）\ntemp/\n'
  fs.writeFileSync(exclPath, excl, 'utf8')
  console.log('  已追加 temp/ 到 .git/info/exclude')
} else console.log('  已存在 temp/ 忽略规则')
const st2 = git(['status','--porcelain'])
check('git status 完全干净', !st2.out, st2.out||'（干净）')
console.log('\n===== 结果: 通过 '+pass+' / 失败 '+fail+' =====')
