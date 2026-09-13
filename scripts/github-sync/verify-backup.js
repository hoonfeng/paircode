// 阶段2：校验备份完整性 —— 逐文件与工作区当前内容比对字节
const fs = require('fs'), path = require('path'), crypto = require('crypto')
const WT = 'E:\\paircode-master'
const BK = fs.readdirSync(path.join(WT,'.pair','.local-changes-backup')).map(d=>path.join(WT,'.pair','.local-changes-backup',d))
  .sort().pop()
console.log('校验备份目录:', BK)
const sha=p=>crypto.createHash('sha1').update(fs.readFileSync(p)).digest('hex')
let n=0, bad=[], bytes=0
;(function walk(d){ for(const e of fs.readdirSync(d,{withFileTypes:true})){ const p=path.join(d,e.name)
  if(e.isDirectory()){ walk(p); continue }
  if(e.name==='MANIFEST.md') continue
  n++; bytes+=fs.statSync(p).size
  const rel=path.relative(BK,p)
  const src=path.join(WT,rel)
  if(!fs.existsSync(src)){ bad.push(rel+'  [源文件已不存在]'); return }
  if(sha(p)!==sha(src)) bad.push(rel+'  [内容不一致]')
} })(BK)
console.log('备份文件数:', n, ' 总大小:', (bytes/1048576).toFixed(2), 'MB')
console.log('与源文件不一致条目:', bad.length)
if(bad.length) console.log(bad.slice(0,20).join('\n'))
console.log('\n== 关键前端改动是否已入备份 ==')
for(const f of ['plugins-src/ui-app/src/ui-state.js','plugins-src/ui-app/src/app-actions.js','plugins-src/ui-app/src/agent-events.js','plugins-src/ui-app/src/ShellApp.vue','plugins-src/ui-app/src/components/RightPanel.vue','plugins-src/ui-app/src/components/MenuBar.vue','plugins-src/ui-app/src/components/EditorArea.vue']){
  const p=path.join(BK,f.replace(/\//g,'\\'))
  const ok=fs.existsSync(p)
  const has=ok && fs.readFileSync(p,'utf8').includes('setFocusMode') || ok && fs.readFileSync(p,'utf8').includes('convListVisible')
  console.log('  '+f.padEnd(52)+(ok?'已备份':'★缺失')+(ok?(has?'  含本地改动标识':'  (无标识/正常)'):''))
}
console.log('\n== 旧残留整目录是否已备份 ==')
for(const d of ['.pair\\plugins\\tool-core','.pair\\plugins\\tool-git','.pair\\plugins\\tool-vision','config\\philosophy','plugins-src\\plugins\\tool-debug']){
  const p=path.join(BK,d)
  console.log('  '+d.padEnd(34)+(fs.existsSync(p)?'已备份('+fs.readdirSync(p).length+' 项)':'★缺失'))
}
console.log('\n== MANIFEST.md 前 12 行 ==')
console.log(fs.readFileSync(path.join(BK,'MANIFEST.md'),'utf8').split('\n').slice(0,12).join('\n'))
