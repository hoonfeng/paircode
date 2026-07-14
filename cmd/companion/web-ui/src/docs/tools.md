# AI 工具文档

PairCode IDE 中的 AI 助手内置了丰富的工具，能够在编程场景中自动执行各种操作。以下是所有可用工具的说明。

---

## 代码阅读与搜索

### 读取文件 — `read_file`
读取指定文件的内容，支持通过行号范围读取部分内容。

### 搜索文件 — `search_files`
按通配符模式（如 `*.go`）在工作区内递归查找文件。

### 搜索内容 — `search_content`
按正则表达式在文件内容中搜索匹配的行。

### 查找符号 — `find_symbol`
搜索 Go 代码中的函数、类型、结构体、接口、变量等符号的定义位置。

### 获取文件符号 — `get_file_symbols`
列出指定文件中所有检测到的符号。

### 符号引用查询 — `find_symbol_usages`
搜索指定符号在项目中所有被引用的位置。

### 导出符号列表 — `list_exported_symbols`
列出项目中所有导出的符号。

### 文件依赖分析 — `get_file_dependencies`
获取指定 Go 文件的导入依赖和反向依赖。

### 循环依赖检测 — `find_circular_deps`
检测 Go 项目中的包循环依赖。

### 影响范围分析 — `check_impact`
分析修改某文件后会影响哪些其他文件。

### 代码知识图谱 (CodeGraph)
- **codegraph_search** — 在知识图谱中按名称搜索实体
- **codegraph_function** — 查找函数/方法的定义位置
- **codegraph_class** — 获取类型（struct/interface）的完整层次结构
- **codegraph_callers** — 查询哪些函数调用了指定函数
- **codegraph_callees** — 查询指定函数调用了哪些其他函数
- **codegraph_impact** — 分析修改某实体后的影响范围
- **codegraph_file_structure** — 获取文件的实体结构树
- **codegraph_entity_history** — 查询代码实体的变更历史

---

## 文件操作

### 写入文件 — `write_file`
将内容写入指定文件（覆盖模式），自动创建父目录。

### 编辑文件 — `edit_file`
精确替换文件中的一段文本（支持自动 CRLF 归一化、空白折叠匹配），也可按行号定位。

### 多处编辑 — `multi_edit`
对一个文件按顺序应用多处替换，原子操作（任一步失败则全部不写）。

### 移动/重命名文件 — `move_file`
移动文件或目录到新位置。

### 删除文件 — `delete_file`
删除指定文件（不可恢复）。

---

## 命令执行

### 执行命令 — `run_command`
在工作区中执行一条 shell 命令（同步，120s 超时），返回标准输出和错误输出。

### 后台运行 — `run_background`
在后台启动一条长命令（如 dev server），立即返回进程 ID。

### 读取后台输出 — `read_output`
读取某后台进程累积的输出。

### 终止进程 — `kill_process`
停止某后台进程。

---

## 网络与搜索

### 网页抓取 — `web_fetch`
抓取 HTTP/HTTPS 网页并返回纯文本内容（去除 HTML 标签）。

### 网页搜索 — `web_search`
通过搜索引擎检索网络内容，返回标题、链接和摘要。

---

## 网页验证与截图

### 一站式网页验证 — `web_debug`
在无头浏览器中打开 URL，可自动输入文字、点击元素、执行 JS，捕获控制台错误/警告，最后截图保存。

### 无头浏览器 — `headless_browser`
获取 JavaScript 渲染后的页面文本内容（适合 SPA）。

### 网页截图 — `screenshot_webpage`
打开指定 URL 的网页并截图。

### 桌面截图 — `screenshot_desktop`
截取整个桌面。

### 窗口截图 — `screenshot_window`
按窗口标题截取特定窗口。

### 区域截图 — `screenshot_area`
按坐标截取指定区域。

---

## 图像分析

### 图像分析 — `image_analyze`
分析图片中的颜色分布、色块区域和基本图形。

### 图像文字识别 — `image_ocr`
从图片中识别文字（OCR），支持中英文混合识别。

---

## 二进制分析

包括二进制信息解析、十六进制查看、字符串提取、模式搜索、熵分析、哈希计算、字节补丁等工具，用于逆向工程与二进制文件分析。

---

## 办公文档

### CSV 读写 — `csv_read` / `csv_write`
读取和写入 CSV/TSV 文件。

### Excel 读写 — `read_xlsx` / `write_xlsx`
读取和创建 Excel .xlsx 文件。

### Word 读写 — `word_read` / `word_write`
读取和生成 Word .docx 文件。

### PDF 读取 — `read_pdf`
提取 PDF 文件的文本内容，扫描型 PDF 自动进行 OCR 识别。

### Markdown 转 HTML — `markdown_to_html`
将 Markdown 文本转换为 HTML。

### JSON 转表格 — `json_to_table`
将 JSON 数组字符串转为 Markdown 表格。

### 表格统计分析 — `table_stats`
对表格数据的数值列做统计。

### 代码统计报告 — `text_report`
按文件扩展名分组统计代码行数。

---

## Git 操作

查看状态、Diff 对比、暂存、提交、推送、拉取、分支管理、日志查看等完整 Git 功能。

---

## 调试器 (DAP)

支持启动 Go 程序调试会话（基于 Delve），设置断点、单步执行、查看调用栈和变量值。

---

## 项目知识库

以结构化方式记录项目架构、模块职责、数据流、设计决策等知识，跨会话复用。

---

## 记忆系统

AI 助手可以记住用户偏好、历史决策和项目约束，跨会话持续积累知识。

---

## BUG 检测与修复

- **bug_detect** — 全量检测项目 BUG（自动运行 go vet → go build → go test）
- **bug_analyze** — 分析构建/测试输出，提取错误位置和代码上下文
- **bug_fix** — 自动检测 BUG 并生成修复方案

---

## 任务与规划

- **task_create** — 创建子任务
- **update_tasks** — 维护任务进度清单
- **update_plan** — 维护执行计划步骤
- **auto_task_complete** — 自动完成任务跟踪

---

## Skill / MCP

- **skill_list / skill_write / skill_delete** — 管理工作流技能
- **mcp_list / mcp_add / mcp_remove** — 管理 MCP 服务器
- **marketplace_search / marketplace_install** — 检索和安装市场扩展

## 其他工具

- **ask_user** — 向用户提问以获取关键决策
- **restore_snapshot / list_snapshots** — 文件修改历史管理与恢复
- **generate_commit_message** — 在任务完成时自动生成提交信息
