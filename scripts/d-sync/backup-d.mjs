#!/usr/bin/env node
// 备份 D:\PairCode 的全部「同步候选」内容到 temp/sync-D-PairCode/backup-D/（只读源，不改动 D）
// 产出 MANIFEST.json + MANIFEST.md（逐文件 sha256/size/mtime）
import fs from 'node:fs'
import path from 'node:path'
import crypto from 'node:crypto'

const ROOT = 'E:/paircode-master'
const D = 'D:/PairCode'
const DEST = path.join(ROOT, 'temp/sync-D-PairCode/backup-D')
const SKIP = new Set(['node_modules', '.git'])
const SRC = ['.pair/plugins', '.pair/assets/runtime', 'plugins-src/ui-app', 'plugins-src/plugins', 'docs', 'config/skills', 'assets', 'scripts', 'README.md']

function walk(root, out = [], depth = 0) {
  if (depth > 14) return out
  let es; try { es = fs.readdirSync(root, { withFileTypes: true }) } catch { return out }
  for (const e of es) {
    if (SKIP.has(e.name)) continue
    const f = path.join(root, e.name)
    if (e.isDirectory()) walk(f, out, depth + 1); else out.push(f)
  }
  return out
}

const manifest = []
let totalBytes = 0, totalFiles = 0
fs.mkdirSync(DEST, { recursive: true })

for (const rel of SRC) {
  const src = path.join(D, rel)
  let st; try { st = fs.statSync(src) } catch { manifest.push({ rel, status: 'MISSING_IN_D' }); continue }
  const dst = path.join(DEST, rel)
  if (st.isFile()) {
    fs.mkdirSync(path.dirname(dst), { recursive: true })
    fs.copyFileSync(src, dst)
    const buf = fs.readFileSync(src)
    manifest.push({ rel, bytes: buf.length, sha256: crypto.createHash('sha256').update(buf).digest('hex'), mtime: st.mtime.toISOString() })
    totalBytes += buf.length; totalFiles++
    continue
  }
  for (const f of walk(src)) {
    const r = path.relative(D, f), d = path.join(DEST, r)
    fs.mkdirSync(path.dirname(d), { recursive: true })
    fs.copyFileSync(f, d)
    const buf = fs.readFileSync(f)
    manifest.push({ rel: r, bytes: buf.length, sha256: crypto.createHash('sha256').update(buf).digest('hex'), mtime: fs.statSync(f).mtime.toISOString() })
    totalBytes += buf.length; totalFiles++
  }
}

const meta = { createdAt: new Date().toISOString(), source: D, baseline: 'E:/paircode-master @ e6c1e625 (GitHub master)', dest: DEST, files: totalFiles, bytes: totalBytes, scope: SRC }
fs.writeFileSync(path.join(DEST, 'MANIFEST.json'), JSON.stringify({ ...meta, entries: manifest }, null, 2), 'utf8')

let md = `# D:\\PairCode 同步候选内容备份\n\n- 备份时间：${meta.createdAt}\n- 来源：\`${D}\`\n- 基准：${meta.baseline}\n- 范围：${SRC.map(s => '`' + s + '`').join(', ')}\n- 合计：**${totalFiles}** 个文件 / **${(totalBytes / 1024 / 1024).toFixed(2)} MB**\n\n## 恢复方式\n\n\`\`\`bash\n# 把备份原样拷回 D（会覆盖 D 上同名文件）\ncp -r "${DEST}/.pair" "${D}/"\ncp -r "${DEST}/plugins-src" "${D}/"\ncp -r "${DEST}/docs" "${D}/"\ncp -r "${DEST}/config" "${D}/"\ncp -r "${DEST}/assets" "${D}/"\ncp -r "${DEST}/scripts" "${D}/"\ncp "${DEST}/README.md" "${D}/README.md"\n\`\`\`\n\n## 文件清单（${manifest.length}）\n\n`
for (const e of manifest) md += e.status ? `- ⚠️ ${e.rel} (${e.status})\n` : `- \`${e.rel}\` — ${e.bytes} B — sha256:${e.sha256.slice(0, 12)}\n`
fs.writeFileSync(path.join(DEST, 'MANIFEST.md'), md, 'utf8')

console.log(`备份完成 → ${DEST}`)
console.log(`文件数=${totalFiles} 体积=${(totalBytes / 1024 / 1024).toFixed(2)} MB`)
const byGroup = {}
for (const e of manifest) { if (e.status) continue; const g = SRC.find(s => e.rel === s || e.rel.startsWith(s + '/') || e.rel.startsWith(s + '\\')) || '?'; byGroup[g] = (byGroup[g] || 0) + 1 }
console.log('分组计数:', JSON.stringify(byGroup, null, 1))
