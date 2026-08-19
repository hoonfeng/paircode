return {
  name: 'host-capability-probe',
  purpose: '宿主能力探测插件（ctx.app / ctx.fs.tree / webServer 注册）',
  inject: ['fs'],
  apply(ctx) {
    // 注册 /api/ext/probe：返回 ctx.app 全部能力 + 文件树
    if (ctx.webServer && ctx.webServer.register) {
      ctx.webServer.register({ kind: 'exact', path: '/api/ext/probe', handler: (req, res) => {
        const app = ctx.app || {}
        const out = {
          app: {
            workspaceRoot: app.workspaceRoot,
            root: app.root,
            folders: app.folders,
            projectName: app.projectName,
            installDir: app.installDir,
            configDir: app.configDir,
            recentProjects: app.recentProjects
          },
          tree: null
        }
        try {
          if (ctx.fs && ctx.fs.tree) {
            out.tree = ctx.fs.tree('.', 1)
            out.treeCount = out.tree ? out.tree.length : 0
          }
        } catch (e) { out.treeError = String(e) }
        const body = JSON.stringify(out)
        res.writeHead(200, { 'Content-Type': 'application/json; charset=utf-8' })
        res.end(body)
      }})
    }
  }
}
