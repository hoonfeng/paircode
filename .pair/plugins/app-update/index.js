// app-update — 软件更新设置段（在线更新系统的配置面）。
//
// ═══════════════════════════════════════════════════════════════
// 职责边界（★ 只做配置注册，不含更新逻辑）：
//   - 更新引擎与接口在 Go 内核：internal/update（检查/下载/校验/解压/替换/回滚）
//     + /api/update/*（cmd/companion/update_api.go，内核路由表 key: update.*）；
//   - 设计文档：docs/online-update-design.md；
//   - 本插件只注册「设置面板 → 软件更新」配置段（纯 schema，前端设置面板自动渲染），
//     配置值落在 settings.json 的 pluginSettings['app-update']。
//
// 生效时机：内核每次处理 /api/update/* 前重新装配配置 → 保存设置后立即生效，无需重启。
// ═══════════════════════════════════════════════════════════════
return {
  name: 'app-update',
  purpose: '软件更新设置段（更新源/频道/镜像/自动检查/校验开关）——更新引擎与 /api/update/* 在 Go 内核',
  inject: ['logger'],
  apply(ctx) {
    const log = (m) => ctx.logger('app-update').log(m);

    ctx.registerSettings({
      key: 'app-update',
      title: '软件更新',
      fields: [
        {
          name: 'feedType',
          label: '更新源类型',
          type: 'select',
          options: ['github', 'custom'],
          default: 'github',
          hint: 'github = 官方 GitHub Releases；custom = 自建/内网清单',
        },
        {
          name: 'repo',
          label: 'GitHub 仓库',
          type: 'text',
          default: 'hoonfeng/paircode',
          placeholder: 'owner/repo',
        },
        {
          name: 'channel',
          label: '发布频道',
          type: 'select',
          options: ['stable', 'prerelease'],
          default: 'stable',
          hint: 'stable = 最新正式版；prerelease = 含预发布',
        },
        {
          name: 'feedURL',
          label: '自定义清单地址',
          type: 'text',
          placeholder: 'https://…/latest.json（或本地路径）',
          hint: '更新源类型选 custom 时生效；格式见 docs/online-update-design.md §2.2',
        },
        {
          name: 'mirrorPrefix',
          label: '下载镜像前缀',
          type: 'text',
          placeholder: 'https://ghproxy.net/',
          hint: 'github.com 直连不通时用镜像拼下载地址（api.github.com 端点不受影响）',
        },
        {
          name: 'autoCheck',
          label: '启动后自动检查',
          type: 'boolean',
          default: true,
          hint: '启动 20 秒后检查一次，仅提示不自动安装',
        },
        {
          name: 'checkIntervalHours',
          label: '自动检查间隔（小时）',
          type: 'number',
          default: 24,
          min: 1,
          max: 720,
        },
        {
          name: 'requireSha256',
          label: '强制 sha256 校验',
          type: 'boolean',
          default: true,
          hint: '校验不通过一律不安装；关闭后仅比对文件大小（不建议）',
        },
        {
          name: 'keepBackup',
          label: '保留旧版本备份',
          type: 'boolean',
          default: true,
          hint: '旧主程序保留为 <主程序>.old，便于手工回滚',
        },
      ],
    });

    const cfg = ctx.getSettings('app-update') || {};
    log(`软件更新设置段已注册（源=${cfg.feedType || 'github'}，频道=${cfg.channel || 'stable'}）`);
  },
}
