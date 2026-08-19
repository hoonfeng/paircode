return {
  name: 'merge-demo-replace',
  inject: ['fs', 'bash', 'logger'],
  apply(ctx) {
    const log = ctx.logger('merge-demo')
    const path = 'tmp/merge_demo/demo.txt'
    // 工具函数：非 ASCII 字符转 \uXXXX，保证 echo 输出 ASCII 安全（Windows cmd 中文编码坑）
    const toEsc = s => s.split('').map(c => {
      const code = c.charCodeAt(0)
      return code > 127 ? '\\u' + code.toString(16).padStart(4, '0') : c
    }).join('')

    // 1. 读取原始内容
    const content = ctx.fs.readFile(path)
    log.info('[1] 原始内容: ' + JSON.stringify(content))

    // 2. 替换 world -> 世界
    const replaced = content.split('world').join('世界')
    log.info('[2] 替换后: ' + JSON.stringify(replaced))

    // 3. 写回文件
    ctx.fs.writeFile(path, replaced)
    log.info('[3] 已写回 ' + path)

    // 4. 读回确认文件落盘
    const after = ctx.fs.readFile(path)
    log.info('[4] 写回后读回: ' + JSON.stringify(after))

    // 5. bash echo 确认替换成功（转义输出，防编码问题）
    const res = ctx.bash.exec('echo ' + toEsc(replaced))
    log.info('[5] bash echo 输出: ' + res.output.trim())
    if (res.error) log.error('bash stderr: ' + res.error)

    return { text: 'echo 输出: ' + res.output.trim() }
  }
}