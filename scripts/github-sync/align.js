// 阶段3（安全版）：工作区对齐 master —— checkout 覆盖/补齐 + 删除本地未跟踪旧残留
// 安全设计：
//  · git 调用一律用 execFileSync 参数数组（无 shell 拼接，无注入面）
//  · 工作区根从环境变量 SYNC_WT 注入（缺省用默认值）
//  · 删除前有白名单校验（必须来自清单文件）、路径沙箱校验、dry-run（APPLY=1 才真删）
//  · 所有命令返回结构化结果 { ok, status, stdout, stderr }
const { execFileSync } = require('child_process')
const fs = require('fs'), path = require('path')
const WT = process.env.SYNC_WT || 'E:\\paircode-master'
const GD = path.join(WT, 'temp', 'repo', '.git')
const APPLY = process.env.APPLY === '1'
const LIST_DELETE = path.join(WT, 'temp', 'gh-sync', 'list-delete.txt')

function git(args, allowFail){
  const argv = ['--git-dir='+GD, '--work-tree='+WT, ...args]
  try{
    const out = execFileSync('git', argv, { cwd: WT, encoding:'utf8', timeout:300000, maxBuffer:1<<26, stdio:['ignore','pipe','pipe'] })
    return { ok:true, status:0, stdout:(out||'').trim(), stderr:'' }
  }catch(e){
    const r = { ok:false, status:e.status??-1, stdout:String(e.stdout||'').trim(), stderr:String(e.stderr||e.message||'').trim().slice(0,300) }
    if(allowFail) return r
    throw new Error('git 失败('+argv.join(' ')+'): '+r.stderr)
  }
}
// ── 路径沙箱 + 白名单校验 ──
function safeAbs(rel){
  if(typeof rel!=='string' || !rel || rel.includes('\0')) return null
  const norm = rel.replace(/\\/g,'/').replace(/\/+$/,'')
  if(norm.startsWith('/') || /^[a-zA-Z]:/.test(norm) || norm.split('/').includes('..')) return null  // 绝对路径/穿越拒绝
  const abs = path.resolve(WT, norm)
  if(abs !== WT && !abs.startsWith(WT + path.sep)) return null                                      // 沙箱校验
  if(/^temp(\/|$)/.test(norm)) return null                                                          // 绝不动临时区
  if(/^\.pair\/\.local-changes-backup(\/|$)/.test(norm)) return null                                // 绝不动备份
  return { abs, norm }
}
const raw = fs.readFileSync(LIST_DELETE,'utf8').split('\n').map(s=>s.trim()).filter(Boolean)
const WHITELIST = { files:new Set(), dirs:new Set() }
for(const l of raw){
  if(l.startsWith('__DIR__:')) WHITELIST.dirs.add(l.slice(8).replace(/\\/g,'/').replace(/\/+$/,''))
  else WHITELIST.files.add(l.replace(/\\/g,'/'))
}
const result = { mode: APPLY?'APPLY':'DRY-RUN', checkout:null, deletedFiles:[], deletedDirs:[], skipped:[], statusAfter:null }
// ── 1) checkout -f master ──
if(APPLY){
  const r = git(['checkout','-f','master','--','.'], true)
  result.checkout = { ok:r.ok, stderr:r.stderr }
}else{
  result.checkout = { ok:true, note:'DRY-RUN 未执行 checkout' }
}
// ── 2) 删除未跟踪残留（白名单 ∩ 沙箱）──
const planFiles=[], planDirs=[]
for(const rel of WHITELIST.files){
  if(!WHITELIST.files.has(rel)) continue
  const s = safeAbs(rel)
  if(!s){ result.skipped.push(rel+'  [越界/非法]'); continue }
  if(!fs.existsSync(s.abs)){ result.skipped.push(rel+'  [已不存在]'); continue }
  planFiles.push(s)
}
for(const rel of WHITELIST.dirs){
  const s = safeAbs(rel)
  if(!s){ result.skipped.push(rel+'  [越界/非法]'); continue }
  if(!fs.existsSync(s.abs)){ result.skipped.push(rel+'  [已不存在]'); continue }
  planDirs.push(s)
}
console.log('模式:', result.mode, '| 待删文件', planFiles.length, '| 待删目录', planDirs.length)
for(const s of planFiles) console.log('  [文件] '+s.norm)
for(const s of planDirs)  console.log('  [目录] '+s.norm)
if(result.skipped.length){ console.log('  跳过:'); result.skipped.forEach(x=>console.log('    '+x)) }
if(APPLY){
  for(const s of planFiles){ fs.rmSync(s.abs,{force:true}); result.deletedFiles.push(s.norm) }
  for(const s of planDirs){ fs.rmSync(s.abs,{recursive:true,force:true}); result.deletedDirs.push(s.norm) }
  const st = git(['status','--porcelain'], true)
  result.statusAfter = st.stdout
  console.log('\n删除完成: 文件', result.deletedFiles.length, '目录', result.deletedDirs.length)
  console.log('对齐后 status 条目数:', st.stdout?st.stdout.split('\n').filter(Boolean).length:0)
  console.log(st.stdout ? st.stdout.split('\n').slice(0,15).join('\n') : '（工作区与 master 完全一致）')
}
fs.writeFileSync(path.join(WT,'temp','gh-sync','align-result.json'), JSON.stringify(result,null,2),'utf8')
console.log('\n结果已写入 temp/gh-sync/align-result.json')
