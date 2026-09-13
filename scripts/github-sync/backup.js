// 阶段1：生成差异清单并把「将被覆盖 / 将被删除」的文件完整备份
const { execSync } = require('child_process')
const fs = require('fs'), path = require('path')
const WT = 'E:\\paircode-master'
const GD = WT + '\\temp\\repo\\.git'
function git(args, allowFail){
  try{ return execSync('git --git-dir="'+GD+'" --work-tree="'+WT+'" '+args,{cwd:WT,encoding:'utf8',timeout:180000,stdio:['ignore','pipe','pipe'],maxBuffer:1<<26}).trim() }
  catch(e){ if(allowFail) return ''; throw new Error(String(e.stderr||e.message).slice(0,200)) }
}
// ── 1. 清单 A：与 master 有差异的文件（本地存在的部分）──
const diffAll = git('diff --name-only HEAD').split('\n').filter(Boolean)
const willOverwrite = diffAll.filter(f => fs.existsSync(path.join(WT, f)))
const willCreate    = diffAll.filter(f => !fs.existsSync(path.join(WT, f)))   // master 新增，本地缺失
// ── 2. 清单 B：本地未跟踪（master 没有）→ 对齐后应删除（temp/ 除外，是我们自己的临时区）──
const st = git('status --porcelain', true).split('\n').filter(Boolean)
const untracked = st.filter(l => l.slice(0,2) === '??').map(l => l.slice(3)).filter(p => p !== 'temp/' && !p.startsWith('temp/'))
const willDelete = []
for(const p of untracked){
  const abs = path.join(WT, p.replace(/\/$/,''))
  if(!fs.existsSync(abs)) continue
  if(fs.statSync(abs).isDirectory()){
    ;(function walk(d){ for(const e of fs.readdirSync(d,{withFileTypes:true})){ const q=path.join(d,e.name); if(e.isDirectory()) walk(q); else willDelete.push(path.relative(WT,q).replace(/\\/g,'/')) } })(abs)
    willDelete.push('__DIR__:'+p.replace(/\/$/,''))
  } else willDelete.push(p)
}
// ── 3. 备份 ──
const ts = new Date().toISOString().replace(/[-:T]/g,'').slice(0,12)      // YYYYMMDDHHmm
const BK = path.join(WT,'.pair','.local-changes-backup','sync-'+ts)
const backupList = [...new Set([...willOverwrite, ...willDelete.filter(x=>!x.startsWith('__DIR__:'))])]
let bytes = 0
for(const rel of backupList){
  const src = path.join(WT, rel.replace(/\//g,'\\'))
  if(!fs.existsSync(src)) continue
  const dst = path.join(BK, rel.replace(/\//g,'\\'))
  fs.mkdirSync(path.dirname(dst),{recursive:true})
  fs.copyFileSync(src, dst)
  bytes += fs.statSync(src).size
}
const manifest = [
  '# 同步前本地内容备份',
  '生成时间: '+new Date().toISOString(),
  '来源: E:\\paircode-master（GitHub hoonfeng/paircode 旧快照 + 本地未提交改动）',
  '对齐目标: master @ '+git('rev-parse master'),
  '',
  '## 备份文件数: '+backupList.length+'  总大小: '+(bytes/1048576).toFixed(2)+' MB',
  '',
  '## A. 与 master 有差异、已被 master 版本覆盖的文件 ('+willOverwrite.length+')',
  '其中以下 14 个是「git 历史中不存在」的本地独有内容（关键改动，勿丢）：',
  fs.existsSync(WT+'\\temp\\gh-sync\\local-only.txt') ? fs.readFileSync(WT+'\\temp\\gh-sync\\local-only.txt','utf8') : '',
  '',
  '## B. 本地未跟踪、对齐时被删除的旧残留文件 ('+willDelete.filter(x=>!x.startsWith('__DIR__:')).length+')',
  '（含被整目录删除的：'+willDelete.filter(x=>x.startsWith('__DIR__:')).map(x=>x.slice(8)).join(', ')+'）',
  '',
  '## C. master 新增、本地缺失、对齐后补齐的文件 ('+willCreate.length+')',
  willCreate.join('\n'),
  '',
  '## 恢复方法',
  '把本目录内容按相同相对路径覆盖回 E:\\paircode-master\\ 即可还原到同步前状态。',
  '（仅含被覆盖/被删除的文件；.pair 下会话、记忆等运行数据未受影响）',
].join('\n')
fs.writeFileSync(path.join(BK,'MANIFEST.md'), manifest, 'utf8')
fs.writeFileSync(WT+'\\temp\\gh-sync\\list-overwrite.txt', willOverwrite.join('\n'),'utf8')
fs.writeFileSync(WT+'\\temp\\gh-sync\\list-delete.txt', willDelete.join('\n'),'utf8')
fs.writeFileSync(WT+'\\temp\\gh-sync\\list-create.txt', willCreate.join('\n'),'utf8')
console.log('清单：覆盖 '+willOverwrite.length+' 个 / 删除 '+willDelete.filter(x=>!x.startsWith('__DIR__:')).length+' 个（'+willDelete.filter(x=>x.startsWith('__DIR__:')).length+' 个整目录）/ 补齐 '+willCreate.length+' 个')
console.log('备份目录: '+BK)
console.log('备份文件: '+backupList.length+' 个, '+(bytes/1048576).toFixed(2)+' MB')
