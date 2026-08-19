return {
  name: 'merge-demo-verify2',
  inject: ['fs', 'bash'],
  pluginId: 'dyn-40',
  apply(ctx) {
    const path = 'tmp/merge_demo/demo2.txt'
    const toEsc = s => s.split('').map(c => {
      const code = c.charCodeAt(0)
      return code > 127 ? '\\u' + code.toString(16).padStart(4, '0') : c
    }).join('')

    if (!ctx.fs.exists(path)) {
      throw new Error('MERGE_VERIFY2: fileNotFound=' + path)
    }

    // read 读取当前内容
    const after = ctx.fs.readFile(path)
    const oneLine = toEsc(after).split('\r').join('\\r').split('\n').join('\\n')

    // 替换目标验证：是否已把 beta 替换为 BETA
    const hasBeta = after.includes('beta')
    const hasBETA = after.includes('BETA')

    // edit 幂等兜底：若仍有小写 beta 则替换写回
    let changed = false
    if (hasBeta) {
      ctx.fs.writeFile(path, after.split('beta').join('BETA'))
      changed = true
    }
    const final = ctx.fs.readFile(path)
    const finalLine = toEsc(final).split('\r').join('\\r').split('\n').join('\\n')

    // bash echo 验证（转义输出，防编码问题）
    const res = ctx.bash.exec('echo "' + finalLine + '"')
    const msg = 'fileContent=' + oneLine +
      ' | hasBeta=' + hasBeta +
      ' | hasBETA=' + hasBETA +
      ' | changed=' + changed +
      ' | final=' + finalLine +
      ' | echo=' + res.output.trim() +
      ' | echoOk=' + (res.output.trim() === finalLine) +
      (res.error ? ' | bashErr=' + res.error : '')
    throw new Error('MERGE_VERIFY2_RESULT: ' + msg)
  }
}