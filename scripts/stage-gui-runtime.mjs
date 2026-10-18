#!/usr/bin/env node
// 把当前平台的 libSkiaSharp 运行时库复制到 release/，供打包步骤收进发布包。
//
// GUI 主程序（pair.exe，-tags gui）由 cgo 链接 goskia 的导入库，运行时仍需
// 动态库在场：
//   - Windows: libSkiaSharp.dll 必须与 exe 同目录（或 PATH 中）；
//   - Linux:   rpath=$ORIGIN，libSkiaSharp.so 与可执行文件同目录；
//   - macOS:   rpath=@loader_path，libSkiaSharp.dylib 与可执行文件同目录。
//
// 库来源：goskia module 的 skia/lib/<os>_<arch>/（随 module 分发）。本脚本用
// `go list -m` 解析 module 目录，不依赖固定 GOPATH 布局。
//
// 失败不致命（发布包缺库时 GUI 无法启动，但服务模式仍可用）——调用方在
// pipeline 中配置了 ignoreError；这里尽量给出明确日志。
import { execFileSync } from 'node:child_process'
import fs from 'node:fs'
import path from 'node:path'

const root = process.cwd()
const releaseDir = path.join(root, 'release')

const osName = process.platform === 'win32' ? 'windows' : process.platform
const archName = process.arch === 'x64' ? 'amd64' : process.arch
const libDirName = `${osName}_${archName}`

const libNames =
  osName === 'windows' ? ['libSkiaSharp.dll'] :
  osName === 'darwin' ? ['libSkiaSharp.dylib'] :
  ['libSkiaSharp.so']

function goModDir(cwd) {
  try {
    const out = execFileSync('go', ['list', '-m', '-f', '{{.Dir}}', 'github.com/hoonfeng/goskia'], {
      cwd,
      encoding: 'utf8',
      stdio: ['ignore', 'pipe', 'ignore'],
      env: { ...process.env, GOWORK: 'off' },
    })
    return out.trim()
  } catch {
    return ''
  }
}

// 依次尝试：wb-ui 兄弟仓库（go.mod require goskia）→ 当前项目。
const moduleDirs = [
  goModDir(path.resolve(root, '..', 'wb-ui')),
  goModDir(root),
].filter(Boolean)

if (moduleDirs.length === 0) {
  console.log(`  ⚠ stage-gui-runtime: 未能解析 goskia module 目录（跳过 ${libDirName}）`)
  process.exit(0)
}

fs.mkdirSync(releaseDir, { recursive: true })

let copied = 0
for (const modDir of moduleDirs) {
  const srcDir = path.join(modDir, 'skia', 'lib', libDirName)
  for (const name of libNames) {
    const src = path.join(srcDir, name)
    if (!fs.existsSync(src)) continue
    const dst = path.join(releaseDir, name)
    // Windows 上目标文件若残留只读属性（或被索引/杀软短暂持有），
    // copyFileSync 直接覆盖会抛 EPERM；先删除再复制。
    try { fs.rmSync(dst, { force: true }) } catch {}
    fs.copyFileSync(src, dst)
    const mb = (fs.statSync(dst).size / 1024 / 1024).toFixed(1)
    console.log(`  → ${name} (${mb} MB) ← ${src}`)
    copied++
  }
  if (copied > 0) break
}

if (copied === 0) {
  console.log(`  ⚠ stage-gui-runtime: 未找到 ${libDirName} 的运行时库（${libNames.join(', ')}）`)
}
