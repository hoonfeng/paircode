// v2：只列「本地独有内容」（master 对象库中没有 → 同步将丢失）
const { execSync } = require('child_process')
const fs = require('fs')
const GD = 'E:\\paircode-master\\temp\\repo\\.git'
const WT = 'E:\\paircode-master'
function g(args, allowFail){
  try{ return {ok:true,out:execSync('git --git-dir="'+GD+'" --work-tree="'+WT+'" '+args,{cwd:WT,encoding:'utf8',timeout:180000,stdio:['ignore','pipe','pipe'],maxBuffer:1<<26}).trim()} }
  catch(e){ return allowFail ? {ok:false,out:''} : {ok:false,out:String(e.stderr||e.message||'').trim().slice(0,120)} }
}
const files=g('diff --name-only HEAD').out.split('\n').filter(Boolean)
const localOnly=[], oldVer=[], absent=[]
for(const f of files){
  const abs=WT+'\\'+f.replace(/\//g,'\\')
  if(!fs.existsSync(abs)){ absent.push(f); continue }
  const h=g('hash-object --path="'+f+'" "'+abs+'"')
  if(!h.ok||!/^[0-9a-f]{40}$/.test(h.out)){ localOnly.push(f+'  [无法读取]'); continue }
  const ex=g('cat-file -e '+h.out, true)
  if(ex.ok) oldVer.push(f); else localOnly.push(f)
}
console.log('差异文件总数:',files.length,'| 本地旧版:',oldVer.length,'| 本地缺失(master新增):',absent.length,'| 本地独有内容:',localOnly.length)
console.log('\n===== B. 本地独有内容（git 历史中不存在 → 同步会覆盖/丢失）=====')
console.log(localOnly.length?localOnly.join('\n'):'（无）')
fs.writeFileSync(WT+'\\temp\\gh-sync\\local-only.txt', localOnly.join('\n'),'utf8')
fs.writeFileSync(WT+'\\temp\\gh-sync\\absent.txt', absent.join('\n'),'utf8')
fs.writeFileSync(WT+'\\temp\\gh-sync\\oldver.txt', oldVer.join('\n'),'utf8')
console.log('\n明细已写入 temp/gh-sync/{local-only,absent,oldver}.txt')
