// 分析：本地工作区 vs GitHub master 的差异性质（基线定位 / 本地独有文件来历 / 缺失文件来源）
const { execSync } = require('child_process')
const GD = 'E:\\paircode-master\\temp\\repo\\.git'
const WT = 'E:\\paircode-master'
function g(args){
  try{ return execSync('git --git-dir="'+GD+'" --work-tree="'+WT+'" '+args,{cwd:WT,encoding:'utf8',timeout:180000,stdio:['ignore','pipe','pipe'],maxBuffer:1<<26}).trim() }
  catch(e){ return 'ERR: '+String(e.stderr||e.stdout||e.message||'').trim().slice(0,200) }
}
console.log('===== 1) 本地内容在 master 历史中的定位 =====')
const probes=['.pair/plugins/agentloop/index.js','internal/agent/agentloop.go','cmd/companion/main.go','README.md','plugins-src/ui-app/src/agent-events.js','docs/plugin-development.md']
for(const f of probes){
  const sha=g('hash-object --path="'+f+'" "'+WT+'\\'+f.replace(/\//g,'\\')+'"')
  const w=g('log --all --format=%H -n 1 --find-object='+sha)
  if(/^[0-9a-f]{40}$/.test(w)){
    const cnt=g('rev-list --count '+w+'..master')
    const subj=g('log -1 --format=%s '+w)
    console.log(f.padEnd(42)+' 落后 '+cnt.padStart(4)+' 个提交  由于: '+subj.slice(0,46))
  } else console.log(f.padEnd(42)+' → 历史中无此内容（本地独有/已改）')
}
console.log('\n===== 2) 本地独有文件（master 树中没有）的来历 =====')
const locals=['internal/agent/parallel_tools.go','internal/agent/loop_parallel.go','cmd/companion/subagent_spawn.go','internal/agent/subagent_registry.go','internal/agent/subagent_sink.go','docs/project-knowledge.md','config/philosophy/tao-te-ching.txt','.pair/plugins/tool-core/index.js','.pair/plugins/tool-git/index.js','.pair/plugins/ui-quick-exec/index.js','plugins-src/plugins/tool-debug']
for(const f of locals){
  const h=g('log --all --oneline -n 3 -- "'+f+'"')
  const ret=g('log --all --oneline --diff-filter=D -n 1 -- "'+f+'"')
  const ins=g('ls-tree master -- "'+f+'"')
  console.log('· '+f)
  console.log('   master 树中存在: '+((ins&&/^\d+/.test(ins))?'是':'否')+'    历史: '+((h&&!h.startsWith('ERR')&&h)?h.replace(/\n/g,' | '):'（git 历史中从未出现）'))
  if(ret&&!ret.startsWith('ERR')&&ret) console.log('   曾删除于: '+ret)
}
console.log('\n===== 3) 本地缺失文件（master 有、本地无）的来源 =====')
const missing=['dev/agent-teams-goal-snapshot.md','dev/tool_plugin_gen/main.go','dev/v16-t1-frontend-dead-code-report.md','internal/agent/handoff.go','internal/agent/apply_patch.go','internal/agent/run_stats.go','internal/agent/image_pipeline.go','internal/agent/scenario_tools.go','internal/agent/llmtrace.go','scripts/overwrite-install.mjs','scripts/verify-release.mjs','docs/plugin-tools-overlap-audit.md','.pair/plugins/tool-bug/index.js','internal/core/pair_migrate.go']
for(const f of missing){
  const h=g('log -1 --format=%h|%ad|%s --date=short master -- "'+f+'"')
  console.log('· '+f.padEnd(44)+' ← '+(h.split('\n')[0]||'?'))
}
console.log('\n===== 4) master 时间线（判断落后幅度） =====')
console.log('HEAD      : '+g('log -1 --format=%h %ad %s --date=iso master'))
console.log('master~25 : '+g('log -1 --format=%h %ad %s --date=iso master~25'))
console.log('master~50 : '+g('log -1 --format=%h %ad %s --date=iso master~50'))
console.log('\n===== 5) 差异文件按主题归类（真实 diff） =====')
const numstat=g('diff --numstat HEAD').split('\n').filter(Boolean)
const byDir={}
for(const l of numstat){ const p=l.split('\t')[2]; if(!p) continue; const k=p.split('/').slice(0,2).join('/'); byDir[k]=(byDir[k]||0)+1 }
console.log(Object.entries(byDir).sort((a,b)=>b[1]-a[1]).map(([k,v])=>k+'='+v).join('\n'))
