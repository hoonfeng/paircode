return {
  name: 'merge-demo-verify',
  inject: ['fs', 'bash'],
  apply(ctx) {
    const path = 'tmp/merge_demo/demo.txt'
    const toEsc = s => s.split('').map(c => {
      const code = c.charCodeAt(0)
      return code > 127 ? '\\u' + code.toString(16).padStart(4, '0') : c
    }).join('')
    const after = ctx.fs.readFile(path)
    // 压成单行转义字符串，避免 bash 把换行拆成多条命令
    const oneLine = toEsc(after).split('\r').join('\\r').split('\n').join('\\n')
    const res = ctx.bash.exec('echo "' + oneLine + '"')
    const msg = 'fileContent=' + oneLine +
      ' | echo=' + res.output.trim() +
      ' | hasWorld=' + after.includes('world') +
      ' | hasShiJie=' + after.includes('世界') +
      (res.error ? ' | bashErr=' + res.error : '')
    throw new Error('MERGE_VERIFY_RESULT: ' + msg)
  }
}