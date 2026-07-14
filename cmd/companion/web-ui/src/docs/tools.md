# AI 工具文档

PairCode IDE 中的 AI 助手内置了丰富的工具，可以像你使用 IDE 一样操作文件、搜索代码、运行命令、管理版本。你只需要用自然语言告诉 AI 你想做什么，AI 会自动选择合适的工具来完成任务。

所有工具对 AI 完全开放，你无需记忆工具名称——只需描述需求，AI 会自动判断该用什么。

---

## 一、代码阅读与搜索

**浏览项目结构、搜索代码内容和定位符号定义，是 AI 理解你代码的基础能力。**

- `read_file` — 读取指定文件的内容，可按行号范围读取部分内容。
- `list_files` — 列出目录下的文件和子目录，可按通配符过滤。
- `search_content` — 按正则表达式在文件内容中搜索匹配的行。
- `search_files` — 按通配符模式在工作区内递归查找文件。
- `find_files_by_pattern` — 按 glob 模式查找文件，附带语言和大小信息。
- `find_symbol` — 搜索 Go 代码中函数、类型、结构体等符号的定义位置。
- `get_file_symbols` — 列出指定文件中所有检测到的符号。
- `find_symbol_usages` — 搜索指定符号在项目中所有被引用的位置。
- `list_exported_symbols` — 列出项目中所有导出的公开符号。
- `get_file_dependencies` — 获取指定文件的导入依赖和反向依赖情况。
- `check_impact` — 分析修改某个文件后可能会影响哪些其他文件。
- `find_circular_deps` — 检测项目中是否存在包之间的循环依赖。
- `find_entry_points` — 查找项目中的入口文件。
- `find_config_files` — 查找项目中的配置文件。
- `tool_stats` — 查看各工具的使用频率统计。

---

## 二、代码知识图谱 CodeGraph

**将项目代码的结构化信息构建成知识图谱，让 AI 能像"看"一样理解代码之间的调用关系。**

- `codegraph_build` — 构建或更新项目的代码知识图谱。
- `codegraph_stats` — 查看知识图谱的统计信息（实体数、关系数等）。
- `codegraph_file_structure` — 获取文件的实体结构树（函数→类型→方法）。
- `codegraph_function` — 按名称查找函数或方法的定义位置和签名。
- `codegraph_class` — 获取结构体或接口的完整层次结构（字段、方法、嵌入类型）。
- `codegraph_callers` — 查询哪些函数调用了指定的函数。
- `codegraph_callees` — 查询指定函数内部调用了哪些其他函数。
- `codegraph_impact` — 分析修改某个函数或类型后可能影响的范围。
- `codegraph_search` — 在知识图谱中按名称搜索代码实体。
- `codegraph_git_history` — 查询 Git 提交历史并关联到代码实体。
- `codegraph_entity_history` — 查询指定代码实体的完整变更历史。

---

## 三、文件操作

**读写和编辑工作区内的文件，是 AI 帮你写代码的主要方式。**

- `write_file` — 将内容写入指定文件（覆盖模式），自动创建父目录。
- `edit_file` — 精确替换文件中的一段文本，支持按行号定位。
- `move_file` — 将文件或目录移动到新位置，也可用于重命名。
- `delete_file` — 删除指定文件（不可恢复）。
- `restore_snapshot` — 将文件恢复到修改前的版本。
- `list_snapshots` — 查看某个文件的所有修改历史版本。

---

## 四、命令执行

**在工作区中运行命令，AI 也能用命令行来完成任务。**

- `run_command` — 执行一条 shell 命令并等待结果返回。
- `run_background` — 在后台启动一条长命令（如启动开发服务器）。
- `read_output` — 读取某个后台进程累积的输出内容。
- `kill_process` — 停止某个正在运行的后台进程。

---

## 五、网络与搜索

**AI 可以联网获取信息或搜索资料。**

- `web_fetch` — 抓取网页内容并提取纯文本。
- `web_search` — 通过搜索引擎检索网络信息，返回标题和摘要。

---

## 六、网页验证与截图

**AI 可以打开网页、截图并分析页面内容，用于验证前端效果。**

- `web_debug` — 在浏览器中打开网页，可输入文字、点击元素、检查控制台错误并截图。
- `headless_browser` — 获取 JavaScript 渲染后的页面文本内容，适合单页应用。
- `screenshot_desktop` — 截取整个桌面屏幕。
- `screenshot_window` — 按窗口标题截取指定窗口。
- `screenshot_area` — 按坐标截取屏幕的指定区域。
- `screenshot_webpage` — 截取指定 URL 的网页。

---

## 七、图像分析

**AI 可以"看"图片并理解其中的内容。**

- `image_analyze` — 分析图片中的颜色分布、色块区域和基本图形。
- `image_ocr` — 从图片中识别文字，支持中英文混合识别。

---

## 八、二进制分析

**查看和分析二进制文件的内容，用于逆向工程或文件格式分析。**

- `inspect_binary` — 分析二进制文件的大小、类型和十六进制预览。
- `write_binary` — 将 Base64 编码的内容写入二进制文件。
- `binary_strings` — 从二进制文件中提取可打印的字符串。
- `binary_find` — 在二进制文件中搜索指定的字节模式或文本。
- `binary_patch` — 在二进制文件的指定位置写入字节补丁。
- `binary_info` — 解析可执行文件的结构（架构、入口、节区、导入导出符号）。
- `binary_hash` — 计算文件的 MD5、SHA1、SHA256 哈希值。
- `binary_entropy` — 按块计算文件的香农熵，用于识别压缩或加密区域。

---

## 九、办公文档

**读写常见的办公文档格式，包括表格、文档和 PDF。**

- `csv_read` — 读取 CSV 或 TSV 文件并以表格形式展示。
- `csv_write` — 将数据写入 CSV 或 TSV 文件。
- `json_to_table` — 将 JSON 数组数据转为 Markdown 表格。
- `table_stats` — 对表格数据的数值列做统计（求和、均值、最大值等）。
- `text_report` — 按文件扩展名分组统计代码行数。
- `word_read` — 读取 Word 文档的文本内容。
- `word_write` — 根据 Markdown 内容生成 Word 文档。
- `read_xlsx` — 读取 Excel 文件的内容。
- `write_xlsx` — 创建 Excel 文件。
- `read_pdf` — 提取 PDF 文件的文本内容，扫描型 PDF 会自动进行 OCR 识别。
- `markdown_to_html` — 将 Markdown 文本转换为 HTML。

---

## 十、Git 版本控制

**在对话中完成 Git 操作，AI 可以帮你管理代码版本。**

- `git_status` — 查看工作区的 Git 状态（已修改、已暂存、未跟踪文件）。
- `git_diff` — 查看文件的变更内容。
- `git_log` — 查看最近的提交历史。
- `git_show` — 查看某次提交的详情和改动。
- `git_blame` — 逐行查看文件的最后修改人和提交信息。
- `git_add` — 将文件加入暂存区。
- `git_commit` — 提交已暂存的改动。
- `git_branch` — 列出、创建或删除分支。
- `git_checkout` — 切换分支或恢复文件的修改。
- `git_stash` — 将当前工作区的改动暂存起来，稍后恢复。

---

## 十一、调试器 DAP

**AI 可以启动调试会话，设置断点并检查程序运行状态。**

- `debug_start` — 启动 Go 程序的调试会话。
- `debug_stop` — 停止当前的调试会话。
- `debug_breakpoint` — 在指定文件的指定行设置断点。
- `debug_continue` — 从暂停状态继续执行程序。
- `debug_next` — 单步跳过，执行当前行但不进入函数内部。
- `debug_step_in` — 单步进入，进入函数调用内部。
- `debug_step_out` — 单步跳出，执行到当前函数返回。
- `debug_stack` — 查看当前线程的调用栈。
- `debug_variables` — 查看当前暂停点的变量值。
- `debug_evaluate` — 在暂停状态下求值表达式。
- `debug_status` — 查看当前调试会话的状态。

---

## 十二、项目知识库

**将项目架构、模块职责和设计决策记录下来，让 AI 跨会话了解你的项目。**

- `project_info_write` — 写入一条项目知识，如架构说明或设计决策。
- `project_info_read` — 读取某条项目知识的详细内容。
- `project_info_list` — 列出知识库的所有条目概览。
- `project_info_search` — 按关键词搜索知识库中的内容。
- `project_info_delete` — 删除某条项目知识。
- `project_info_explore` — 生成项目目录结构概览。
- `project_index` — 扫描并索引项目文件结构。
- `project_overview` — 获取项目的总体概览（目录、文件数、语言分布）。

---

## 十三、记忆系统

**AI 可以记住你的偏好、历史决策和项目约束，跨对话持续积累。**

- `memory_write` — 写入一条持久记忆，AI 在后续对话中会自动参考。
- `memory_read` — 读取某条记忆的详细内容。
- `memory_search` — 按关键词搜索已有记忆。
- `memory_list` — 列出所有历史记忆的摘要。
- `memory_delete` — 删除一条过时的记忆。
- `memory_count` — 查询记忆库中的总条目数。

---

## 十四、BUG 检测与修复

**AI 可以自动发现代码中的问题并给出修复方案。**

- `bug_analyze` — 分析构建或测试的输出，提取错误位置和上下文。
- `bug_detect` — 全量检测项目中的 BUG，自动运行编译和测试检查。
- `bug_fix` — 自动检测 BUG 并生成修复方案，支持多次迭代修复。

---

## 十五、任务与规划

**AI 可以追踪任务进度和执行计划，确保复杂的多步骤任务有条不紊。**

- `task_create` — 创建一个新的子任务并跟踪其状态。
- `update_tasks` — 更新任务清单中各项任务的进度状态。
- `update_plan` — 维护和更新执行计划的步骤清单。
- `generate_commit_message` — 在任务全部完成后生成提交信息。

---

## 十六、Skill / MCP 管理

**管理和扩展 AI 的能力——技能是工作流模板，MCP 是标准化的工具扩展协议。**

- `skill_list` — 列出所有可用的技能及其激活模式。
- `load_skill` — 加载某个技能的完整内容供 AI 使用。
- `load_skill_resource` — 加载技能的附加资源文件。
- `skill_write` — 创建或更新一个技能模板。
- `skill_delete` — 删除一个项目级技能。
- `mcp_list` — 列出已配置的 MCP 服务器。
- `mcp_add` — 新增一个 MCP 服务器扩展。
- `mcp_remove` — 删除一个 MCP 服务器配置。

---

## 十七、市场

**浏览和安装来自公共市场的技能和 MCP 扩展。**

- `marketplace_search` — 在市场检索可安装的 MCP 服务器或技能。
- `marketplace_install` — 从市场安装指定的扩展。

---

## 十八、Lua 工具管理

**自定义和扩展 AI 的工具集——通过 Lua 脚本创建属于自己的工具。**

- `lua_tool_list` — 列出所有已创建的 Lua 自定义工具。
- `lua_tool_create` — 创建一个新的 Lua 自定义工具。
- `lua_tool_update` — 更新现有 Lua 工具的代码或参数。
- `lua_tool_delete` — 删除一个 Lua 自定义工具。

---

## 十九、桥接模式

**控制 AI 对系统资源的访问权限——安全模式与完全接管模式之间切换。**

- `bridge_status` — 查看当前的桥接模式和系统访问权限状态。
- `bridge_takeover` — 切换为接管模式，获得完整的系统资源访问能力。
- `bridge_lockdown` — 归还权限，切换回安全的桥接模式。
- `bridge_exec` — 通过桥接执行系统命令，行为取决于当前模式。
- `bridge_register_system_tool` — 在接管模式下注册一个系统管理工具。

---

## 二十、其他

**辅助性工具，在特定场景下帮助 AI 更好地与用户协作。**

- `ask_user` — 向用户提问以获取关键决策或澄清需求。
- `delegate_task` — 将复杂任务委托给子 AI 独立完成。
- `delegate_single_turn` — 让子 AI 执行一次简单的查询就返回结果。
- `transfer_to_agent` — 将对话控制权转移给另一个 AI 处理。
- `asset_list` — 列出已保存的智能资产（经验胶囊和技能基因）。
- `asset_search` — 搜索已保存的智能资产。
- `asset_delete` — 删除指定的智能资产。
- `evolution_save_capsule` — 将修复经验保存为经验胶囊，供日后复用。
- `evolution_search_capsules` — 根据错误信息检索历史修复方案。
- `evolution_save_gene` — 记录跨项目复用的编程最佳实践。
- `evolution_status` — 查看进化系统的运行状态和积累的资产数量。
- `resource_list` — 列出所有可用的资源文件。
- `resource_search` — 搜索资源文件。
- `resource_stats` — 查看资源使用统计。
