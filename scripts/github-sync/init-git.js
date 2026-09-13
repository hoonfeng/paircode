// 阶段4：把已克隆的仓库接入工作区，配置可访问的 remote（github.com 直连在本机不可达）
const { execFileSync } = require('child_process')
const fs = require('fs'), path = require('path')
const WT = process.env.SYNC_WT || 'E:\\paircode-master'
const SRC_GIT = path.join(WT,'temp','repo','.git')
const DST_GIT = path.join(WT,'.git')
const PROXY_URL = 'https://gh-proxy.com/https://github.com/hoonfeng/paircode.git'
const DIRECT_URL = 'https://github.com/hoonfeng/paircode.git'
function git(args){
  try{
    const out = execFileSync('git', args, { cwd:WT, encoding:'utf8', timeout:180000, maxBuffer:1<<26, stdio:['ignore','pipe','pipe'] })
    return { ok:true, status:0, stdout:(out||'').trim(), stderr:'' }
  }catch(e){
    return { ok:false, status:e.status??-1, stdout:String(e.stdout||'').trim(), stderr:String(e.stderr||e.message||'').trim().slice(0,300) }
  }
}
const log = []
if(fs.existsSync(DST_GIT)){ log.push('工作区已存在 .git，跳过移动'); }
else{
  fs.renameSync(SRC_GIT, DST_GIT)
  log.push('已把克隆仓库接入工作区：temp/repo/.git → .git')
}
// remote 配置：origin=代理（本机可达），github=直连（备用）
const steps = [
  ['remote','set-url','origin',PROXY_URL],
  ['config','remote.origin.fetch','+refs/heads/*:refs/remotes/origin/*'],
  ['branch','--set-upstream-to=origin/master','master'],
]
for(const s of steps){ const r=git(s); log.push('$ git '+s.join(' ')+'  → '+(r.ok?(r.stderr||'OK'):'失败: '+r.stderr)) }
if(!git(['remote']).stdout.split('\n').includes('github')){
  const r=git(['remote','add','github',DIRECT_URL]); log.push('$ git remote add github <直连地址>  → '+(r.ok?'OK':r.stderr))
}
git(['config','core.autocrlf','true'])
git(['config','core.longpaths','true'])
// 清理临时克隆的工作树（仓库已接入，无需两份副本）
try{ fs.rmSync(path.join(WT,'temp','repo'),{recursive:true,force:true}); log.push('已清理 temp/repo 临时工作树') }catch(e){ log.push('清理 temp/repo 失败: '+e.message) }
// 验证
const st = git(['status','--porcelain'])
const head = git(['log','-1','--format=%h %ad %s','--date=iso'])
const rem = git(['remote','-v'])
const br = git(['branch','-vv'])
console.log(log.join('\n'))
console.log('\n== 验证 ==')
console.log('git status 条目:', st.stdout?st.stdout.split('\n').filter(Boolean).length:0, st.stdout?('\n'+st.stdout.split('\n').slice(0,10).join('\n')):'（工作区干净）')
console.log('HEAD:', head.stdout)
console.log('分支:', br.stdout)
console.log('remote:\n'+rem.stdout)
fs.writeFileSync(path.join(WT,'temp','gh-sync','git-init-result.json'), JSON.stringify({log,status:st.stdout,head:head.stdout,remote:rem.stdout,branch:br.stdout},null,2),'utf8')
