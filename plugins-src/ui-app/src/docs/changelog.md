# 更新日志

> 所有 PairCode IDE 的重要变更均记录在此文件中。

---

## 1.4.15 — 2026-09-02

### 修复
- **模型切换报「对话不存在 / 会话不存在」** — 根因：打包管线缺失 vite 壳构建步骤（pipeline 只跑 build-ui 区域插件 + robocopy 同步旧 dist），打包复用过期前端产物，致 `setConvModel` 调用参数错位（5 参新调用打在 4 参旧签名上，workspaceRoot 被传成配置名「硅基flash」）→ 后端按错位 workspaceRoot 路由隔离 store 查无会话报 400
- **packager.json pipeline 新增 `build-ui-frontend` 步骤** — 打包前显式构建 vite 壳（plugins-src/ui-app → .pair/assets/runtime/web），保证发布包前端产物始终来自新构建
- **后端跨 store 兜底防误报** — SessionManager 新增 `FindConversation`（指定 workspaceRoot store 查不到时遍历已打开 store 找回），GET/PUT 会话接口在参数缺失/错位时不再误报「不存在」，会话级模型切换落盘到会话真实所属工作区

### 改进
- 版本号整体提升至 v1.4.15（main.go 缺省 / packager.json / 前端 package.json）

---

## 1.2.1 — 2026-08-15

### 新增
- **按双层循环范式重写 Agent 核心** — 双层循环（turn/step 边界事件、inbox 双队列对齐 next-step/next-turn），消息组装与落盘对齐事件模型（agentloop 编号 ↔ 消息序列推导），系统提示精简为基础工具集模式（`WB_FULL_TOOLS=1` 恢复全量工具）
- **一切皆插件** — Go 插件框架 + goja JS 动态插件，goja 运行时完全内置（双仓库去除 replace），JS 插件沙箱支持 timer 服务（ctx.timeout/interval）与跨 goroutine 执行锁
- **内置 TS 编译器** — esbuild 纯 Go 转译（无 CGO/npm 依赖），TS 插件可直接加载（`cordis_define` 支持 js/ts/自动探测），多文件 TS bundle（Build stdin + mock 包）
- **工具全插件化** — 21 个内置功能插件（core/fs/git/web/shell/memory/task/project-info/codegraph/debug/vision/office/lsp 等），`cordis_inspect` 可见工具归属插件，Unload 可回收整组
- **多项目支持** — 工具 project 参数路由（文件类/搜索/Git 全套），codegraph 按项目独立建图与查询（非主项目用各自 JSONStore，天然隔离），memory/project-info 工具显式 project 参数化
- **工具集生态** — 模板插件化动态构建（`toolset_build` 按项目+需求自动组合工具并固化到工作区）、固化/导出/导入/市场发布（plugin 类型）、LLM 项目意图分析（语言无关，不固化任何语言模板）
- **插件生态 P0-P2** — 函数形态 + `apply(ctx, config)` + inject 服务 + VM 超时防护 + schema 校验 + 插件管理 UI（host/client 双半）+ client inspect provider
- **项目知识库树形化** — 树分支组织（目标/架构/实现/关键点/设计思想）+ AGENTS.md 分层 + .agents 路径兼容
- **历史注入对齐 harness** — 删除【历史轮次】前缀标注与 task 时间戳，系统提示补充多轮对话规则
- **ask_user 选项内输入** — 支持 single / multi / single-with-input / text 四态交互，修复参数名混淆导致选项不出现的问题
- **遗留五件套** — notes 写入同步 + read_image 工具 + run_code 嵌套 + prompt 注册中心 + 知识库过期检查修复

### 修复
- **移除未完成注入** — TOOL_OUTCOME_UNKNOWN / interrupted 机制移除，无 result 的 tool_call 以空占位维持配对契约，不再向模型注入「中断/未完成」语义
- **知识库过期验证误报** — 152 条假警告清零，159 条全绿

---

## 1.1.8 — 2026-08-11

### 新增
- **OCR / 图色识别能力** — 图片文字识别（中英文混合）与颜色分布分析，工具配置持久化 + 前端工具面板（2026-08-04）
- **对话历史注入膨胀三层压缩** — 固定背景 / 动态日志 / 长时压缩三层方案，控制上下文体积（2026-08-04）
- **异常中断后继续未完成对话** — 中断后可直接继续，不丢上下文（2026-08-06）
- **后台进程跨轮存活** — run_background 进程不再因每轮重建注册表而丢失（全局单例 bgRegistry）（2026-08-11）
- **多项目工具** — Lua 工具 / 工具配置按项目加载 + project 参数路由（2026-08-11）
- **背景摘要注入位置修复** — 压缩摘要固定在 task 前注入（前缀稳定），动态日志追加末尾，KV 缓存零损失优化（2026-08-08）

### 修复
- 关闭 run 内自动压缩，改由外层时机控制（2026-08-05）
- 历史消息配对错乱 — 用户消息重复存储导致 tool 配对错乱（lastUser 锚点重组）
- 历史消息分段导致多气泡 — 连续 assistant 消息合并显示
- 多轮对话 user 后 tool 粘连 + OnBatchPersist 偏移 — 压缩后固定偏移失效，改 lastUser 锚点重组
- 归档双 bug — ①Windows 归档静默失效（句柄未关闭 + os.Rename 不能覆盖）→ 显式 Close + 三步法原子替换；②归档摘要孤立 assistant 消息污染 LLM 上下文 → 改 role=user +【历史归档】标注
- 多根路径解析 Bug — 优先匹配文件实际存在的根目录

---

## 1.1.6 — 2026-07-30

### 修复
- **修复编辑器 Ctrl+F 不生效** — CodeMirror `search()` 扩展注册的 `openSearchPanel` 与自定义搜索面板 keymap 冲突，使用 `Prec.high()` 确保自定义 handler 优先执行，Ctrl+F 正确唤出中文搜索面板
- **搜索面板图标全部换为 SVG** — Unicode 字符（▲▼↔×）和文本标签（Aa ·\* 全词）全部替换为内联 SVG 图标，与界面风格统一
- **修复前端 API 路径缺少前导斜杠导致 404** — `apiURL()` 拼接时对无前导斜杠的 path 自动补全，`/apitools/review` 修正为 `/api/tools/review`
- **修复 codegraph 增量构建仍全量重写 SQLite** — `SQLiteStore.Save()` 在增量模式下调用的 `RemoveFileEntities` 清理旧数据，不再 `DELETE FROM` 全表

### 改进
- **编辑器中文搜索面板** — 新建 `FindPanel.vue` 组件，替换 CodeMirror 默认英文搜索面板，支持查找/替换/大小写敏感/正则/全词匹配
- **codegraph 增量构建测试** — 新增 `TestSQLiteStoreIncrementalPreserves` 和 `TestSQLiteStoreIncrementalBuild` 验证增量构建与并行完整性

---

## 1.1.5 — 2026-07-29

### 新增
- **run_command 后台化** — `run_command` 改用后台启动+轮询模式，不再阻塞 Agent 循环，可被上下文取消中断，超时后 LLM 可选择等待或继续
- **审核配置改为工作区级** — 审核黑白名单从全局 settings.json 迁移到工作区 .pair/tools.json，不同工作区可独立配置，避免动态工具（Lua）在不同工作区间混淆
- **Lua 工具补齐 Tool 结构** — `buildLuaTool` 自动设置 UsageGuide/Category/Enabled 字段，与标准工具结构一致
- **工具配置弹窗合并** — 「启用开关」和「审核黑白名单」合并为同一「工具配置」弹窗，标签页切换，避免歧义
- **自主模式 Follow-up 持续驱动** — Agent 自然终止后，通过 `OnNextTask` 回调自动注入 follow-up 消息，无需手动触发「继续」
- **流式更新机制** — Registry 新增 `OnToolUpdate` 回调，工具执行中间结果实时推送给前端
- **工具 UsageGuide 全覆盖** — 全部 ~140 个工具添加 `UsageGuide` 使用指导，明确何时用、为何优于 `run_command`、常见误区
- **启动日志详细化** — 启动时输出版本号、Go 版本、平台架构、工作目录、各工作区文件夹路径

### 改进
- **工具体系升级**
  - `Tool` 结构体新增 `UsageGuide`、`Category`、`Enabled` 字段
  - `Registry` 新增 `EnabledDefinitions()` 按状态过滤工具定义
  - 新增 `AllToolMeta()` API 供前端展示工具开关列表
  - 工具使用指南文本动态注入系统提示，引导 LLM 优先使用专用工具
- **窗口管理** — `run_command` / `run_background` 均设置 `HideWindow=true`，不再弹出 cmd 窗口
- **信号监听移除** — main 函数移除信号监听，进程不会因子进程结束而自动退出

### 修复
- **debug_start 启动修复** — 拆解 `dlv dap` 启动流程，分别发送 Initialize 和 Launch 请求，兼容 dlv 最新版本

---

## 1.1.2 — 2026-07-21

### 新增
- **附件标签化** — 消息中的文件/代码/图片附件不再嵌入正文，改为独立药丸形标签显示在用户消息文字下方，视觉更清爽
- **粘贴长文本自动转临时附件** — 输入框粘贴超过 2000 字符的文本时，自动写入 `_temp/` 目录并作为附件挂载，避免大段代码/日志撑爆输入区

### 改进
- `addToChat` 瘦身：文件添加到对话不再预读文件内容（40KB 截断已无意义），仅传递路径引用
- 目录引用新增 `type:dir` 支持，提示 agent 使用 `list_files` 查看
- 选中代码添加对话现在保留代码内容尾注供 agent 直接参考（截断 3000 字）

### 修复
- 文件树 Shift/Ctrl 多选逻辑修复：范围选择改为基于同级节点列表，清除后重新选中

---

## 1.1.1 — 2026-07-21

### 新增
- **审核配置界面重设计** — 从纯文本输入改为工具卡片式交互：所有工具按类别分组（文件操作、命令执行、Git、网络、截图、图像、二进制、办公文档、CodeGraph、调试器、知识库、记忆、LSP、BUG检测、任务管理、扩展市场等），每个工具显示中文名称，点击切换三态（默认 → 黑名单 → 白名单），支持搜索过滤，配置更直观高效

### 修复
- **修复新对话空状态提示位置偏移** — "开始新的对话，发送消息即可与 AI 助手对话"提示及图标从左下角偏移修正为居中显示

### 改进
- 版本号统一升级至 1.1.1（前端 package.json、后端 main.go、打包配置）

---

## 1.1.0 — 2026-07-20

### 新增
- **自主模式原生终止** — 去掉 `finish_task` 强制结束机制，Agent 自然输出后直接结束循环，交互更流畅
- **Agent 性能优化（P0-P3 五轮）** — eventRing 环形缓冲器减少内存分配、进度可视化（阶段指示器+工具调用计数+耗时）、工具描述精简减少 Token 消耗、并行工具执行机制、预压缩上下文避免截断
- **会话连贯性增强** — 新对话开始时自动注入 Git 变更感知、代码图谱统计、工作区结构概览，Agent 无需从零分析项目

### 改进
- **ChatView 重构** — 消息渲染管线全面优化，新增交互超时保护、审核驳回追踪、折叠/展开状态持久化
- **审核配置 UI 优化** — 弹窗改为向上弹出（bottom:100%），防止被视口底部裁切
- **编辑工具 v2 升级** — 更精确的符号级定位，减少行号偏移问题
- **kill_process 增强** — 改为杀进程树，彻底清理子进程
- **自主模式架构重构** — ephemeralMsgs 隔离内层消息，长时压缩精准保留推理上下文

### 修复
- 修复 `planExpanded` / `tasksExpanded` / `currentPhase` 重复声明导致的运行时崩溃
- 修复 `currentTasks` 未声明导致前端 `undefined.length` 崩溃
- 修复自然终止代码缩进丢失导致逻辑在循环外不执行

---

## 1.0.20 — 2026-07-18

### 修复
- **修复消息排序** — `_idx` 统一取 `max(existing)+1`，解决历史消息加载后序号错乱
- **修复用户反馈消息合并** — 用户反馈正确合并到 agent 输出气泡中，不再产生额外用户消息气泡
- **修复消息发送双占位竞态** — `switchConv` 复用历史消息中最后一条 assistant 消息接收后续 WS 事件，避免两个 assistant 气泡
- **修复 WS 连接与历史加载竞态** — `processStatus` 事件正确处理连接状态转换

### 改进
- 审核配置弹窗改为向上弹出（`bottom:100%`），防止被视口底部裁切
- 移除压缩按钮，简化 UI

---

## 1.0.19 — 2026-07-17

### 修复
- **修复 Web 端文件树不显示** — `FileExplorer.vue` 的 `<script setup>` 编译后 JS 中存在变量暂时性死区（TDZ），导致 `setup()` 抛出 `Cannot access 'd' before initialization`，文件树组件挂载失败。重建前端并重新编译 `companion.exe` 嵌入新版 dist 后修复
- **修复后端 dist 嵌入路径不一致** — `cmd/companion/main.go` 通过 `//go:embed web-ui/dist` 引用 companion 目录下的副本，但此前构建脚本将 dist 输出到 `cmd/desktop/web-ui/dist/`，两者不同步导致嵌入的仍是旧版 JS。统一构建流程后将新版 dist 正确复制到 `cmd/companion/web-ui/dist/`

### 改进
- 统一更新版本号至 1.0.19（后端 main.go、两个前端的 package.json）

---

## 1.0.8 — 2026-07-17

### 新增
- **多项目工作区支持** — 系统提示自动遍历所有工作区根目录，读取各自 `.pair/project.md` 环境配置注入给 AI，跨项目协作时准确感知每个项目的编译方式、CGO 开关等信息
- **CodeGraph 多项目全量建图** — `codegraph_build` 支持对所有工作区项目建图并合并到同一个知识图谱（`rebuild=true`），跨项目符号搜索成为可能
- **阻塞命令自动拦截** — 新增 `isBlockingCommand` 检测，自动拦截 dev server、watch 模式、`go run .`、`npm run dev` 等长期进程命令，提示改用 `run_background`，避免阻塞 AI 循环

### 改进
- **审核放行逻辑优化** — `run_command` 阻塞命令不再自动放行，强制走 LLM 审核；`run_background` 保持安全命令自动放行
- **工具描述优化** — `run_command` 描述明确禁止长期进程并列出典型误用场景；`run_background` 强调作为长期进程首选工具
- **系统提示增强** — 「错误恢复」和「防止卡死」两处加入阻塞/后台区分铁律，降低误用 `run_command` 概率

---

## 1.0.7 — 2026-07-17

### 修复
- **修复刷新页面后 ask_user 提交造成额外气泡** — 页面刷新后 `switchConv` 复用历史消息中最后一条 assistant 消息接收后续 WS 事件，不再另建新占位，避免两个 assistant 气泡

### 改进
- 统一更新版本号至 1.0.7（前端 package.json、后端 main.go、打包脚本）

---

## 1.0.6 — 2026-07-17

### 修复
- **修复消息持久化比较口径不一致** — `PersistNewMessages` 中 `persistedCount` 使用 `countJSONLLines`（统计文件总行数含 System），与 `histNonSystemCount`（统计非 System 消息数）口径不同，导致含 tool_call 的 assistant 消息在工具执行前被误判为"已落盘"而跳过写入。阻塞工具（如 ask_user）的前端始终无响应。改用 `readJSONL` 精确统计非 System 消息数
- **修复对话/任务/执行计划 API 空实现** — `GET /api/conversations/{id}` 缺 agent 运行状态，`GET /api/tasks` 和 `GET /api/taskplan` 原返回对话列表（完全错误的 stub），改为返回真实数据

---

## 1.0.5 — 2026-07-17

### 改进
- **消息持久化重构** — `PersistNewMessages` 改为全量覆盖写 JSONL，消除 diff 计算的竞态问题；`MessageStore` 新增 `ReplaceHistory` 支持历史压缩；`MergeLastAssistantRun` 移除，各轮次独立存储以保留 reasoning 完整时序

### 修复
- **修复 send on closed channel panic** — 移除三处 `go func` 在无监听者时向 channel 发送导致的崩溃
- **修复 PersistNewMessages 上下文压缩后新消息丢失** — 全量替换模式确保压缩后的摘要消息不被覆盖
- **修复自动提交仅提交主工作区** — `doAutoCommit` 遍历所有工作区执行 git add + commit
- **修复 idx 空洞导致消息跳过持久化** — `PersistNewMessages` 内部不再跳过 System/User 消息，确保序号连续

---

## 1.0.4 — 2026-07-17

### 新增
- **技能状态三级配置** — 技能可设为「关闭 / 按需加载 / 始终激活」三种模式，灵活控制 AI 行为
- **市场安装范围选择** — 安装 MCP 服务器或技能时，支持选择 user（全局）或 project（项目级）范围

### 改进
- **对话历史持久化增强** — 页面刷新后对话完整恢复，不再因浏览器关闭丢失上下文；后端全面接管消息状态管理，前端不再依赖本地缓存
- **消息展示优化** — 连续同一角色的消息自动合并显示（如多个 assistant 回复合并为一条），阅读更流畅
- **停止信号可靠性提升** — Agent 异常结束或用户主动停止时，前端能可靠收到停止信号并更新 UI 状态

### 修复
- 修复切换对话时 loading 状态卡死的问题（switchConv 提前放行占位消息）
- 修复消息历史顺序错乱和思考链（reasoning_content）丢失的严重问题
- 修复 MergeConsecutiveAssistants 跳过 RoleTool 消息导致工具调用结果不完整的问题

---

## 1.0.3 — 2026-07-17

### 改进
- **子进程窗口管理** — 所有后台子进程（Git 操作、BUG 检测编译/测试、Lua 工具执行、桥接命令）统一隐藏控制台窗口，避免黑框闪烁
- **会话持久化** — OnBatchPersist 回调从"每 5 轮"改为"每轮迭代"写盘，降低异常丢失风险
- **代码搜索提示修复** — codegraph 搜索无结果时正确显示查询内容而非空占位符

### 修复
- **PersistNewMessages idx 空洞 bug** — 修复因跳过 System/User 角色消息导致消息序号不连续、后续消息无法正确持久化的严重问题（db_store.go + db_adapter.go）

---

## 1.0.2 — 2026-07-16

### 改进
- **文档同步** — features.md 同步到最新版本，移除冗余的"版本信息与更新日志"章节

---

## 1.0.1 — 2026-07-11

### 新增
- **更新日志页面** — 帮助文档中新增更新日志页面，版本历史一目了然
- **WebSocket 协议文档** — API 文档补充完整 WebSocket 事件类型与负载定义
- **系统版本报告** — `/api/system/info` 现在返回 `version` 字段，前端"关于"面板同步显示

### 改进
- **API 文档全面重写** — 每个接口增加请求体 JSON Schema、响应示例和错误码说明，便于二次开发
- **帮助文档重构** — 文档归入"文档中心"分类，导航更清晰

---

## 1.0.0 — 2026-07-01

### 新增
- **AI 对话编程** — 用自然语言驱动 AI 读写文件、执行命令、管理 Git
- **自主 Agent 模式** — AI 自动分析项目、制定计划并执行多步骤任务
- **代码编辑器** — 内置多标签页编辑器，支持语法高亮、代码折叠、十六进制查看
- **文件管理** — 工作区目录树浏览、文件搜索、批量操作
- **Git 版本控制** — 对话驱动的 Git 操作（状态查看、暂存、提交、分支管理）
- **内置终端** — 浏览器中的终端面板，支持 AI 自动执行命令
- **对话历史管理** — 自动保存、回溯与继续历史对话
- **BUG 自动检测修复** — AI 扫描编译/测试问题并自动修复
- **Skills / MCP 扩展** — 可复用的工作流模板和模型上下文协议扩展
- **记忆系统** — AI 跨会话记住用户偏好和历史决策
- **任务与规划管理** — 复杂任务分解为可追踪的子步骤
- **Lua 自定义工具** — 通过 Lua 脚本创建自定义 AI 工具
- **代码知识图谱** — 函数调用关系、类型层次、影响范围分析
- **多模型支持** — 灵活切换 AI 模型后端（OpenAI / Claude 等）
- **主题系统** — 四套预设主题（暗色、白色、暖色、暗夜紫）
- **调试器** — 支持 Go 程序的断点、单步和变量查看
- **网页验证工具** — 自动打开 URL、截图、分析页面效果
- **办公文档处理** — 读取 Word / Excel / PDF 文件，支持 OCR

### 技术架构
- 后端使用 Go 语言，前端使用 Vue 3 + CodeMirror
- WebSocket 实时推送 AI 事件流
- 内嵌前端资源（go:embed），单二进制分发
- 纯本地运行，所有 API 仅监听本地回环地址
