# THIRD_PARTY_NOTICES

本文件登记 gou-ide 运行时依赖/借用的第三方组件与许可证义务（随 Round4
JS 运行时升级专项新增维护；历史组件见各模块头注与 go.mod）。

> 全部登记组件均为 MIT 许可证：无传染性义务；保留版权头注即满足要求。
> 运行时以 npm 依赖形式装入 `.pair/cordis/node/node_modules/`（npm install
> 自带 license 字段）；本文件为显式声明汇总。

| 组件 | 版本 | 许可证 | 用途 | 版权/来源 |
|---|---|---|---|---|
| goja（wb-ui/goja fork） | fork of dop251/goja | MIT | goja 沙箱运行时（既有，保留随包 LICENSE） | Copyright 2016 Dmitry Panov；2012 Robert Krimen |
| goja_nodejs（候选 A，未实施） | latest（未引入） | MIT | 若未来实施 fs/path/process 等借用需登记（本轮 P2） | dop251/goja_nodejs |
| @cordisjs/core（cordis3） | 3.18.1（桥安装） | MIT | 现有 Node 桥 cordis3 Context | cordis 项目（2022 友好凉拌/koishi） |
| @deepseek-ai/cordis（cordis4） | ^4.0.1 | MIT | Round4 DSH 插件装载 Context（`@deepseek-ai/cordis` peer 显式安装） | deepseek-harness（2026 DeepSeek） |
| @deepseek-ai/dsh-agent / dsh-llm / dsh-session / dsh-subagent / dsh-tools / dsh-system-prompt / dsh-commands / dsh-workspace | 0.1.0-rc.x | MIT | DSH 服务面 peer 依赖（npm 安装） | deepseek-harness |
| @deepseek-ai/schemastery | ^3.18.1-rc.1 | MIT | DSH 插件 Config schema 运行时 | deepseek-harness |
| @nanmicoder/dsh-agent-teams | 0.1.14 | MIT | Round4 参考插件直跑验证（用户可自行安装） | Copyright 2026 程序员阿江（Relakkes） |
| cordis.bundle.js（embed） | @cordisjs/core 3.18.1 bundle | MIT | goja 内 cordis 运行时（既有，保留 assets 头注） | cordis 项目 |

## 借用代码注意事项

1. **wb-ui/goja**：fork 补丁（cb1a573 协作锁）保留原 LICENSE 与头注；本仓不复制其源码。
2. **DSH 插件生态**（@deepseek-ai/*）：仅作为 Node 桥运行时依赖被 `bridge_node.js`
   的 cordis4 装载分支 `import()`，不复制源码；peer 由 `npmInstallDshPeers`
   （internal/agent/node_plugins.go）显式安装并记录于桥目录 package.json。
3. **参考插件 @nanmicoder/dsh-agent-teams**：仅用于装载运行验证；其团队状态
   落盘 `<workspace>/.agent-teams/`，本仓不包含其源码副本。

## 维护约定

- 新增 npm 运行时依赖 → 在本表追加一行（组件/版本/许可证/用途/来源）。
- 候选 A（goja_nodejs 借用）若未来实施 → 补登本表并在引入处保留版权头注。
