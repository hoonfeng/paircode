// 甄别差异文件中：哪些是「本地旧版」（master 历史中有对应内容，可直接更新）
//            哪些是「本地独有改动」（对象库中不存在，同步将丢失 → 必须提示）
const { execSync } = require('child_process')
const GD = 'E:\\paircode-master\\temp\\repo\\.git'
const WT = 'E:\\paircode-master'
function g(args){
  try{ return {ok:true,out:execSync('git --git-dir="'+GD+'" --work-tree="'+WT+'" '+args,{cwd:WT,encoding:'utf8',timeout:180000,stdio:['ignore','pipe','pipe'],maxBuffer:1<<26}).trim()} }
  catch(e){ return {ok:false,out:String(e.stderr||e.stdout||e.message||'').trim().slice(0,200)} }
}
const numstat=g('diff --name-only HEAD')
const files=numstat.out.split('\n').filter(Boolean)
console.log('待甄别差异文件数:',files.length)
const localOnly=[], oldVer=[], err=[]
for(const f of files){
  const h=g('hash-object --path="'+f+'" "'+WT+'\\'+f.replace(/\//g,'\\')+'"')
  if(!h.ok||!/^[0-9a-f]{40}$/.test(h.out)){ err.push(f+' ['+h.out.slice(0,60)+']'); continue }
  const ex=g('cat-file -e '+h.out)
  if(ex.ok) oldVer.push(f); else localOnly.push(f)
}
console.log('\n===== A. 本地旧版（历史中存在 → 可安全更新为 master）共 '+oldVer.length+' =====')
console.log(oldVer.join('\n'))
console.log('\n===== B. 本地独有内容（对象库中没有 → 同步会丢失！）共 '+localOnly.length+' =====')
console.log(localOnly.join('\n'))
if(err.length){ console.log('\n===== 读取异常 '+err.length+' ====='); console.log(err.join('\n')) }
console.log('\n===== C. D 盘运行实例是否含折叠实现 =====')
try{
  const r=execSync('findstr /s /m /c:"convSidebarVisible" "D:\\PairCode-1.1.2\\.pair\\*.js" "D:\\PairCode-1.1.2\\.pair\\*.html"',{encoding:'utf8',timeout:60000})
  console.log(r.trim()||'（D 盘无匹配）')
}catch(e){ console.log('D 盘检索：'+(String(e.stdout||'').trim()||'无匹配')) }
