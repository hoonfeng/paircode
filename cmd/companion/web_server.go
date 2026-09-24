// HTTP 服务器核心（无 GWui 依赖，webonly 和桌面模式共用）。
//

package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"embed"
	"encoding/json"
	"fmt"
	"github.com/hoonfeng/paircode/pkg/executil"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/hoonfeng/paircode/internal/agent"
	"github.com/hoonfeng/paircode/internal/core"
	"github.com/hoonfeng/paircode/internal/server/handler"
	mcppanel "github.com/hoonfeng/paircode/internal/ui/mcp"
	"github.com/hoonfeng/paircode/internal/ui/skills"
	"github.com/hoonfeng/paircode/pkg/memory"
	"github.com/hoonfeng/paircode/pkg/summary"
)

// ─── 内嵌前端资源 ─────────────────────────────────────────────

//go:embed web-ui/dist
var webUIFiles embed.FS

// resolveWebDir 解析外部前端目录（磁盘优先，不重新编译改 UI）：
//  1. WEB_DIR 环境变量（显式指定，如开发时指向 web-ui/dist）
//  2. exe 旁 web/ 目录（存在则用；不存在则首次从 embed 解压产物）
//  3. 都不可用返回 ""（使用内嵌资源）
//
// ★ 用户改 UI 的路径：改 web-ui/src → npm run build → 拷贝/覆盖到外部目录 → 重启生效，
//
//	无需重新编译 Go；或直接编辑外部目录下的产物文件（index.html / assets/*.js）后重启。
func resolveWebDir() string {
	if dir := strings.TrimSpace(os.Getenv("WEB_DIR")); dir != "" {
		if st, err := os.Stat(dir); err == nil && st.IsDir() {
			return dir
		}
		log.Printf("[WebUI] WEB_DIR=%s 不存在，回退内嵌资源", dir)
		return ""
	}
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	// ★ 资源外置（主程序只保留框架）：优先 .pair/assets/runtime/web/（插件目录
	//   统一承载前端产物，可独立更新），其次 exe 旁 web/（兼容既有解压目录）。
	rtDir := filepath.Join(filepath.Dir(exe), ".pair", "assets", "runtime", "web")
	if st, err := os.Stat(rtDir); err == nil && st.IsDir() {
		return rtDir
	}
	dir := filepath.Join(filepath.Dir(exe), "web")
	if st, err := os.Stat(dir); err == nil && st.IsDir() {
		return dir
	}
	// 首次启动：从 embed 解压前端产物到 exe 旁 web/，之后用户可直接改该目录。
	subFS, err := fs.Sub(webUIFiles, "web-ui/dist")
	if err != nil {
		return ""
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return ""
	}
	ok := true
	_ = fs.WalkDir(subFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		target := filepath.Join(dir, filepath.FromSlash(path))
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, rerr := fs.ReadFile(subFS, path)
		if rerr != nil {
			ok = false
			return rerr
		}
		if werr := os.WriteFile(target, data, 0o644); werr != nil {
			ok = false
			return werr
		}
		return nil
	})
	if ok {
		return dir
	}
	_ = os.RemoveAll(dir)
	return ""
}

// webServer 是运行在 companion 内部的 HTTP 服务器。
type webServer struct {
	server    *http.Server
	port      int
	mu        sync.Mutex
	eventRing *eventRing // 全局事件环形缓冲（断连回放用）
}

// agentMgr 全局会话管理器：管理并行 agent 会话（Start/Stop/Subscribe 等）。
// web 层作为套壳，所有会话生命周期通过 agentMgr 管理。
var agentMgr = agent.NewSessionManager()

// ★ Node 桥会话管理器注入：Node 插件（npm cordis）经 ctx.store/ctx.loop 服务
//
//	读写会话消息/感知循环状态（数据落盘逻辑可插件化）。
func init() {
	agent.SetNodeBridgeManager(agentMgr)
	// ★ 2026-08-31 会话级模型路由：agent 包按会话解析 Provider 参数时，
	//   经此钩子读会话元数据里记录的 服务商/模型（切模型只改本会话）。
	//   ★ 2026-09-03 附带配置名 preset：装配按配置整套展开（含该配置 Key）。
	agent.SetConvModelLookup(func(convID, wsRoot string) (string, string, string) {
		store := agentMgr.StoreFor(wsRoot)
		if store == nil {
			return "", "", ""
		}
		meta, err := store.GetConversation(convID)
		if err != nil || meta == nil {
			return "", "", ""
		}
		return meta.Provider, meta.Model, meta.Preset
	})
	// ★ 2026-08-31 会话级审核模式桥：/api/tools/review?convId=… 读写会话级模式
	//   （会话元数据持久化 + 运行中 Loop 实时更新；未注入时 handler 回落工作区级）。
	agent.SetConvReviewBridge(
		func(convID, wsRoot string) (string, error) {
			store := agentMgr.StoreFor(wsRoot)
			if store == nil {
				return "", nil
			}
			return store.ConvReviewMode(convID), nil
		},
		func(convID, wsRoot, mode string) error {
			if store := agentMgr.StoreFor(wsRoot); store != nil {
				if err := store.SetConvReviewMode(convID, mode); err != nil {
					return err
				}
			}
			agentMgr.SetReviewMode(convID, mode)
			return nil
		},
	)
}

var ws *webServer

// ── 平台回调注册（由 webui_*.go 在启动时注册） ──
// 统一 API 层后，平台差异仅通过以下回调参数化：
//
//	buildProviderFn:   创建 LLM Provider（主模型）
//	buildSystemPromptFn:创建系统提示语
//	buildCompressorFn:  创建上下文压缩器（nil=规则式压缩）
type (
	buildProviderFn     func() agent.Provider
	buildSystemPromptFn func() string
	buildCompressorFn   func() agent.Compressor
)

var (
	webProvider     buildProviderFn
	webSystemPrompt buildSystemPromptFn
	webCompressor   buildCompressorFn
)

// findMessageStoreRoot 在所有工作区文件夹中查找第一个有对话数据目录的路径。
// 解决 bug：当 WorkspaceFolders 排序变化后，core.Root() 指向了没有对话数据的文件夹，
// 导致 MessageStore 读不到已有的 index.json，对话列表为空。
func findMessageStoreRoot() string {
	primary := core.Root()
	if primary == "" {
		return ""
	}
	// 遍历 core.Folders，找第一个已有对话数据的目录
	for _, f := range core.Folders {
		idxPath := filepath.Join(f, ".pair", "conversations", "index.json")
		if info, err := os.Stat(idxPath); err == nil && !info.IsDir() {
			return f
		}
		// 也检查旧格式 conversations.json（尚未迁移的）
		legacyPath := filepath.Join(f, ".pair", "conversations.json")
		if info, err := os.Stat(legacyPath); err == nil && !info.IsDir() {
			return f
		}
	}
	return primary
}

// startWebUI 在后台启动 Web UI 服务器。
func startWebUI(port int) {
	if ws != nil {
		return
	}
	ws = &webServer{
		port:      port,
		eventRing: newEventRing(1000), // 缓存最近 1000 个全局事件用于断连回放
	}
	// ★ 2026-09 会话唤醒投递：子 Agent 派生面已删除，只保留「向已存在会话投递
	//   一条输入把它唤醒跑一轮」的能力（JS 插件经 ctx.agents.followup 调用；
	//   agent-teams 插件 Web 面板「批准并运行」唤醒队长会话依赖它）。
	ws.installSessionWake()
	// ★ 2026-09 精简策略可见化：硬地板默认关闭（只用窗口比例阈值），
	//   想恢复绝对量保护设 PAIR_COMPACT_HARD_FLOOR=<token>。
	// ★ 2026-09-19：上下文窗口以服务商配置（models.json）为准（经装配器；settings 顶层值仅兜底）
	log.Printf("[compact] 精简策略：%s", agent.CompactPolicy(agent.ContextWindow(agent.ResolveProviderParams())))
	// ★ 钩子系统（t1 L2 闭环）：装载配置钩子（.pair/settings.json + ~/.pair/settings.json），
	//   与桌面端 Init 同一入口；无配置时全部 no-op。
	agent.InitLoopHooks()
	// 初始化 MessageStore（消息持久化的唯一权威），并迁移旧格式数据
	// ★ 先找已有对话数据的工作区目录（排序变化后 core.Root() 可能指向了没有对话数据的目录）
	root := findMessageStoreRoot()
	if root != "" {
		agentMgr.SetWorkspaceRoot(root)
		agent.SetCodeGraphDB(agentMgr.RawDB())
		// ★ 主项目根与共享 DB 成对设置：共享 DB 归属判定（cgStoreFor）依赖
		//   cgSharedRoot，缺省空串导致主项目恒走 JSONStore（307MB 全量读写）。
		agent.SetCodeGraphRoot(root)
		// ★ 会话桥注入（ask_user/task_create 插件化路由）：插件工具经
		//   ctx.hostTool.exec('ask_user'/'task_create') + _convID 路由回本会话，
		//   多会话并发按 convID 精确路由（WaitAnswer 读会话 askCh / 工作区查询）。
		agent.SetSessionBridge(&agent.SessionBridge{
			WaitAnswer: func(ctx context.Context, convID string) (string, error) {
				return agentMgr.WaitAnswer(ctx, convID)
			},
			// ★ Round3 ⑤：多问题回答数组（answers 路由）
			WaitAnswers: func(ctx context.Context, convID string) ([]agent.AskAnswer, error) {
				return agentMgr.WaitAnswers(ctx, convID)
			},
			GetWorkspaceRoot: func(convID string) string {
				return agentMgr.GetSessionWorkspaceRoot(convID)
			},
		})
		// 迁移旧 conversations.json + history_cache.json 到新格式
		convPath := filepath.Join(root, ".pair", "conversations.json")
		hcPath := filepath.Join(root, ".pair", "history_cache.json")
		if store := agentMgr.Store(); store != nil {
			if err := store.MigrateFromLegacy(convPath, hcPath); err != nil {
				fmt.Printf("[WebUI] 迁移旧对话数据失败（不阻塞启动）: %v\n", err)
			}
		}
		// ★ 启动时异步构建代码知识图谱（不阻塞 HTTP 服务器启动）
		go func() {
			if _, err := agent.EnsureCodeGraph(root); err != nil {
				log.Printf("[WebUI] 启动时构建代码图谱失败（不阻塞启动）: %v", err)
			} else {
				log.Printf("[WebUI] 代码知识图谱已就绪")
			}
		}()
	}
	if root := core.Root(); root != "" {
		memory.SetRoot(root)
	}
	// 初始化 Skills 资源目录（供 LoadAllSkills 使用）
	if root := core.Root(); root != "" {
		agent.SkillProjectDir = filepath.Join(root, ".pair", "skills")
	}
	if sysDir := filepath.Join(core.ConfigDir(), "skills"); sysDir != "" {
		agent.SkillSystemDir = sysDir
	}
	// ★ 2026-09-12 修复：全局技能目录（跨工作区共享；skill_write scope=global
	//   与「未开工作区时写入回落」的目标目录）。随安装目录固定，启动设置一次。
	agent.SkillGlobalDir = filepath.Join(core.InstallDir(), ".pair", "skills")
	// ★ 技能启停/状态覆盖在预热前应用（与 buildWebLoopOpts 保持一致）：
	//   若只在此后 buildWebLoopOpts 才设置，PromptCacheWarmer 预热缓存的是
	//   「全部技能」版本，运行时 overrides 生效变「部分技能」→ dynamic 段每次
	//   重建都不同 → system 消息前缀变化 → DeepSeek 缓存断裂（命中率掉到 ~50%）。
	agent.SkillEnabled = core.Settings.SkillEnabledOverrides
	agent.SkillStatusOverride = core.Settings.SkillStatusOverrides
	// 初始化 MCP 配置路径（供 agent/mcp_config.go 读写 mcp.json）
	agent.MCPUserConfigPath = filepath.Join(core.ConfigDir(), "mcp.json")
	if root := core.Root(); root != "" {
		agent.MCPProjectConfigPath = filepath.Join(root, ".pair", "mcp.json")
	}

	// ★ 启动时初始化参考注册表 + 全局插件宿主（web 模式唯一的 PluginHost）：
	//   ★ 插件是程序的扩展，存 <InstallDir>/.pair/plugins/（跨工作区生效）——与
	//     是否打开工作区无关，未打开工作区也必须初始化并装载全局插件
	//     （发布版启动即生效：UI 区域插件/工具插件不依赖工作区）；
	//     项目工具集（工作区 .pair/toolsets/）等有工作区时才装载。
	// ★ 旧 bin\.pair 一次性迁移必须在任何 .pair 读取（LoadAllToolsets/cordis.patch/
	//   asset store）之前执行（2026-09-09 InstallDir bin 上跳配套升级路径）。
	core.MigrateLegacyBinPairData()
	root = core.Root() // 可为空（未打开工作区）
	initReg := agent.NewRegistry()
	agent.RegisterHostFrameworkTools(initReg, root)
	// ★ harness 对齐（默认关闭——全量工具集；WB_HARNESS=1 开启时精简 pair 独有工具）；
	//   被禁用工具保留在注册表（前端可见可管理），内置工具集 builtin 可一键恢复
	if n := agent.ApplyHarnessToolFilter(initReg, nil); n > 0 {
		log.Printf("[WebUI] harness 对齐模式：禁用 %d 个 pair 独有工具（WB_FULL_TOOLS=1 恢复全量；工具集面板 builtin 分组开关启用）", n)
	}
	handler.SetToolsRegistry(initReg)
	log.Printf("[WebUI] 参考工具注册表已初始化（%d 个工具）", len(initReg.AllToolMeta()))

	// ★ 内置 HTTP 接口 → 内核路由表（接口插件化：能力注册，不挂载）。
	//   必须在磁盘插件装载（LoadAllToolsets → LoadGlobalPlugins 装 core-api）
	//   之前调用——core-api apply 时经 ctx.kernel.install 挂载这些接口。
	registerKernelAPIs(ws)

	// ★ 在线更新引擎（2026-09-19）：装配配置（安装目录 / config 目录 / 更新源）
	//   并启动后台自动检查（GitHub Releases 分发；实现见 internal/update，
	//   接口见 cmd/companion/update_api.go，设置项由 .pair/plugins/app-update 注册）。
	refreshUpdateEngine()
	startUpdateAutoCheck()

	// ★ 全局插件宿主：web 模式唯一的 PluginHost（浏览器插件面板 + cordis 工具共用）。
	//   与 AgentBase.Init 对齐：NewPluginHost + RegisterCordisTools + 内置插件 + cordis.patch.json。
	//   （框架能力 workspaceRoot 服务 + 内置工具集模板已内联 NewPluginHost，不占插件位）
	ph := agent.NewPluginHost(initReg, agentMgr.Store(), root)
	agent.RegisterCordisTools(initReg, ph, root)
	agent.RegisterToolsetTools(initReg, root, ph)
	// ★ 2026-09-11 场景创造（scenario_scan/scenario_create）：注册在宿主框架面——
	//   供 tool-scenario 磁盘插件 claimTool 存档（宿主能力库，ctx.hostTool 执行；
	//   须早于下方 LoadAllToolsets 装载插件）；不注册进会话 reg——会话可见性由
	//   /创造 命令按需激活控制。
	agent.RegisterScenarioTools(initReg, root, ph)
	if root != "" {
		// ★ 迁移旧版 builtin.json（内置组条目并入工作区工具集 default.json 后删除）；
		//   内置工具包与工作区工具集统一为一套逻辑
		agent.MigrateLegacyBuiltinJSON(ph, root)
	}
	// ★ 启动自动装载工具集（工作区 .pair/toolsets/ + 全局插件；未打开工作区只装全局插件）
	agent.LoadAllToolsets(ph, root)
	// ★ cordis.patch.json 静态插件装配：有工作区读工作区，否则读安装目录（全局）
	patchPath := filepath.Join(root, ".pair", "cordis.patch.json")
	if root == "" {
		patchPath = filepath.Join(core.InstallDir(), ".pair", "cordis.patch.json")
	}
	if err := ph.LoadCordisPatch(patchPath); err != nil {
		log.Printf("[WebUI] cordis.patch.json 装配失败（不阻塞启动）: %v", err)
	}
	handler.SetPluginHost(ph)
	agent.SetGlobalPluginHost(ph)
	log.Printf("[WebUI] 全局插件宿主已初始化（%d 个插件）", len(ph.List()))

	// 工作区文件夹变更时同步到 agent 路径解析
	core.OnSyncWorkspace = func(primaryChanged bool) {
		agent.WorkspaceRoots = core.Folders
		// ★ 清理已移除工作区的图谱缓存（释放内存图引用；后台构建回填前
		//   也会校验 root 是否仍活跃，不会复活已移除项目的缓存）。
		agent.PruneCodeGraphs(core.Folders)
		if primaryChanged {
			// 主工作区变更：同步数据库/技能/MCP/记忆路径到新工作区
			root := core.Root()
			if root != "" {
				agentMgr.SetWorkspaceRoot(root)
				agent.SetCodeGraphDB(agentMgr.RawDB())
				agent.SetCodeGraphRoot(root)
				agent.SkillProjectDir = filepath.Join(root, ".pair", "skills")
				agent.MCPProjectConfigPath = filepath.Join(root, ".pair", "mcp.json")
				memory.SetRoot(root)
			} else {
				// ★ 主工作区被移除：清共享 DB 引用（SetWorkspaceRoot 已关闭
				//   旧连接；codegraph 不再持句柄 → 删除工作区文件夹不失败）。
				agent.SetCodeGraphDB(nil)
				agent.SetCodeGraphRoot("")
			}
			// ★ 2026-09-20 插件宿主工作区根同步（修「插件不按工作区切路径 / 产物
			//   写进别的工作区」）：宿主 root 与各插件上下文根原先只在 NewPluginHost
			//   时快照，主工作区切换后不再更新 → 插件 ctx 服务（fs/bash/binary/…）
			//   仍按启动工作区解析路径。root 为空（无主工作区）时同步为空串，
			//   让插件内的路径解析显式报错，而不是静默沿用旧工作区。
			ph.SetWorkspaceRoot(root)
		}
	}
	mux := http.NewServeMux()

	// ★ 接口插件化（2026-08-16）：内置 /api/* 接口全部进内核路由表
	//   （internal/agent/kernel_api.go，能力层），路由挂载权交给 core-api
	//   磁盘插件（.pair/plugins/core-api/，apply 时 ctx.kernel.install）。
	//   mux 不再硬编码任何 /api/* 路由——插件经 ExtRouteMiddleware 优先拦截；
	//   插件停用 → 接口消失（接口随插件生命周期生灭）。
	registerKernelAPIs(ws)

	// chat 框架路由（WebSocket 端点 /ws、/api/terminal/ws 保留宿主；
	// chat/marketplace/memory 等 REST 接口已进内核表，见 registerKernelAPIs）
	registerExtraHandlers(mux, ws)

	// 启动事件持久化后台 worker：订阅 SubscribeAll，处理 token/历史持久化。
	// 与传输层（WebSocket）解耦——即使无前端连接，agent 事件仍会被持久化。
	ws.startEventPersistWorker()

	// ★ 预热 system prompt 编译缓存（启动时在后台线程中预热，不阻塞主流程）
	//   ★ 2026-08-31 按需激活：预热用空会话（按需插件段不注入——与「未激活」基线一致）。
	go agent.PromptCacheWarmer.WarmUp(func() string { return buildWebSystemPrompt("") })

	// ── 插件资产静态服务（全 UI 插件化）──
	// /plugins-assets/<name>/<file>：插件包（<InstallDir>/.pair/plugins/<name>/）
	// 内的静态资产（如 ui-app 插件的 UI bundle：plugins-assets/ui-app/ui-app.js）。
	// client 半经此 URL 加载 Vite 提前编译的产物（运行时零编译）。
	mux.HandleFunc("/plugins-assets/", func(w http.ResponseWriter, r *http.Request) {
		rest := strings.TrimPrefix(r.URL.Path, "/plugins-assets/")
		parts := strings.Split(rest, "/")
		if len(parts) < 2 || parts[0] == "" || strings.Contains(parts[0], "..") {
			http.NotFound(w, r)
			return
		}
		rel := filepath.Join(parts[1:]...)
		if rel == "" || strings.Contains(rel, "..") {
			http.NotFound(w, r)
			return
		}
		base := filepath.Clean(agent.GlobalPluginsPath())
		p := filepath.Join(base, parts[0], rel)
		if !strings.HasPrefix(p, base+string(os.PathSeparator)) {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		http.ServeFile(w, r, p)
	})

	// ── UI 插件 boot 图（外部兼容）──
	// GET /api/ui-boot：从磁盘 UI 插件包按 dsh.ui manifest 独立发现，输出 DSH
	// WebBootGraph 等价 boot 图（rev + entries[{id,url,rev,inject,immediately,external}]）。
	// 服务端装配（见 internal/agent/uiboot.go BuildUIBootGraph），非插件 ext 路由——
	// 作薄壳发现/装载单一入口，与 /plugins-assets/ 同为主机面路由。
	mux.HandleFunc("/api/ui-boot", handler.HandleUIBoot)

	// ── 静态文件 ──
	// ★ 一切皆插件：前端产物支持磁盘优先（WEB_DIR 环境变量 > exe 旁 web/ 目录），
	//   fallback 内嵌（//go:embed web-ui/dist）。
	//   用户改 UI 无需重新编译：npm run build 产物拷到 web/（或直接改 web/ 下产物）→ 重启生效。
	webDir := resolveWebDir()
	var fileServer http.Handler
	if webDir != "" {
		log.Printf("[WebUI] 使用外部前端目录: %s（改 UI 无需重新编译，重启生效）", webDir)
		fileServer = http.FileServer(http.Dir(webDir))
	} else {
		subFS, err := fs.Sub(webUIFiles, "web-ui/dist")
		if err != nil {
			log.Printf("[WebUI] 内嵌资源加载失败: %v", err)
			return
		}
		fileServer = http.FileServer(http.FS(subFS))
	}
	// 禁止浏览器缓存前端文件（避免 Edge 等浏览器加载旧版本 JS/CSS）
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		fileServer.ServeHTTP(w, r)
	}))

	ws.server = &http.Server{
		Addr:    fmt.Sprintf("0.0.0.0:%d", port),
		Handler: corsMiddleware(agent.ExtWSMiddleware(agent.ExtSSEMiddleware(agent.ExtRouteMiddleware(mux)))),
	}

	go func() {
		log.Printf("[WebUI] PairCode Web IDE 启动于 http://0.0.0.0:%d（局域网内其他设备可用本机 IP 访问）", port)
		if err := ws.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[WebUI] 服务器错误: %v", err)
		}
	}()
}

// stopWebUI 停止 Web 服务器。
func stopWebUI() {
	if ws != nil && ws.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		ws.server.Shutdown(ctx)
	}
}

// corsMiddleware 添加 CORS 头。
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ─── API 处理函数 ────────────────────────────────────────────

func (s *webServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	jsonResp(w, map[string]any{"status": "ok", "workspace": core.Root(), "folders": core.Folders})
}

// imageMime 根据扩展名返回图片 MIME 类型。
func imageMime(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	case ".webp":
		return "image/webp"
	case ".bmp":
		return "image/bmp"
	case ".ico":
		return "image/x-icon"
	}
	return ""
}

// handleFSImage 提供图片浏览（直接返回原始字节 + 正确 Content-Type）。
func (s *webServer) handleFSImage(w http.ResponseWriter, r *http.Request) {
	path := filepath.Clean(r.URL.Query().Get("path"))
	mime := imageMime(path)
	if mime == "" {
		jsonErr(w, "不支持的文件类型")
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		jsonErr(w, "读取文件失败")
		return
	}
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Cache-Control", "private, max-age=86400")
	w.Write(data)
}

func (s *webServer) handleWorkspace(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		jsonResp(w, map[string]any{
			"root":    core.Root(),
			"folders": core.Folders,
			"loaded":  core.Loaded,
		})
	case "POST":
		var req struct {
			Action      string   `json:"action"`
			Root        string   `json:"root"`
			Folders     []string `json:"folders"`
			Name        string   `json:"name"`
			Path        string   `json:"path"`
			ParentDir   string   `json:"parentDir"`
			Lang        string   `json:"lang"`
			DeleteFiles bool     `json:"deleteFiles"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonErr(w, err.Error())
			return
		}
		switch req.Action {
		case "create":
			if req.Name == "" && req.Root == "" {
				jsonErr(w, "需要 name 或 root 参数")
				return
			}
			root := req.Root
			if root == "" {
				home, _ := os.UserHomeDir()
				root = filepath.Join(home, "paircode-workspaces", req.Name)
			} else {
				root = filepath.Clean(root) // 归一化（防双反斜杠污染）
			}
			if err := os.MkdirAll(root, 0755); err != nil {
				jsonErr(w, "创建工作区失败: "+err.Error())
				return
			}
			core.Folders = []string{root}
			core.Settings.LastProject = root
			core.Settings.WorkspaceFolders = core.Folders
			// ★ 2026-08-22 同步 workspaceFolderLists（创建后刷新数据一致）
			core.SyncWorkspaceFolderList(root, core.Folders)
			core.Loaded = true
			core.Save()
			if core.OnSyncWorkspace != nil {
				core.OnSyncWorkspace(true)
			}
			jsonResp(w, map[string]any{"ok": true, "root": root})

		case "add-folder":
			if req.Path == "" {
				jsonErr(w, "需要 path 参数")
				return
			}
			req.Path = filepath.Clean(req.Path) // 归一化（防双反斜杠污染 → os.Stat 误判不存在 400）
			if _, err := os.Stat(req.Path); err != nil {
				jsonErr(w, "目录不存在: "+err.Error())
				return
			}
			for _, f := range core.Folders {
				if f == req.Path {
					jsonResp(w, map[string]any{"ok": true, "note": "already exists"})
					return
				}
			}
			core.Folders = append(core.Folders, req.Path)
			core.Settings.WorkspaceFolders = core.Folders
			// ★ 2026-08-22 同步 workspaceFolderLists（每工作区文件夹列表快照）：
			//   此前前端 saveWsList 只在明确事件时写该字段，add-folder 后若不写，
			//   刷新页面 loadWsList 从它恢复 folders 时新项目丢——从后端兜底同步，
			//   保证刷新/重启后 folders 完整。
			core.SyncWorkspaceFolderList(core.Root(), core.Folders)
			core.Save()
			if core.OnSyncWorkspace != nil {
				core.OnSyncWorkspace(true)
			}
			jsonResp(w, map[string]any{"ok": true, "folders": core.Folders})

		case "remove-folder":
			if req.Path == "" {
				jsonErr(w, "需要 path 参数")
				return
			}
			req.Path = filepath.Clean(req.Path) // 归一化（防双反斜杠污染）
			newFolders := make([]string, 0, len(core.Folders))
			for _, f := range core.Folders {
				if f != req.Path {
					newFolders = append(newFolders, f)
				}
			}
			core.Folders = newFolders
			core.Settings.WorkspaceFolders = core.Folders
			// ★ 2026-08-22 同步 workspaceFolderLists（同上：移除后刷新数据一致）
			core.SyncWorkspaceFolderList(core.Root(), core.Folders)
			core.Save()
			if core.OnSyncWorkspace != nil {
				core.OnSyncWorkspace(true)
			}
			jsonResp(w, map[string]any{"ok": true, "folders": core.Folders})

		case "new-project":
			parent := req.ParentDir
			if parent == "" {
				parent = core.Root()
				if parent == "" {
					jsonErr(w, "需要 parentDir 参数或先设置工作区")
					return
				}
			}
			if req.Name == "" {
				jsonErr(w, "需要 name 参数")
				return
			}
			projPath := filepath.Join(parent, req.Name)
			if err := os.MkdirAll(projPath, 0755); err != nil {
				jsonErr(w, "创建项目失败: "+err.Error())
				return
			}
			lang := req.Lang
			if lang == "" {
				lang = detectLang(req.Name)
			}
			genProjectTemplate(projPath, lang, req.Name)
			found := false
			for _, f := range core.Folders {
				if f == projPath {
					found = true
					break
				}
			}
			if !found {
				core.Folders = append(core.Folders, projPath)
				core.Settings.WorkspaceFolders = core.Folders
				// ★ 2026-08-22 同步 workspaceFolderLists（新建项目后刷新数据一致）
				core.SyncWorkspaceFolderList(core.Root(), core.Folders)
				core.Save()
			}
			jsonResp(w, map[string]any{"ok": true, "path": projPath, "lang": lang})

		case "delete":
			if req.Root == "" {
				jsonErr(w, "需要 root 参数（工作区路径）")
				return
			}
			root := filepath.Clean(req.Root) // 归一化（防双反斜杠污染）
			// ★ 2026-08-23 工作区隔离：删除前显式关闭该根的 store/DB 缓存句柄
			//   （多工作区缓存后不再由 SetWorkspaceRoot 代关——不关会导致
			//   Windows 删除/移动工作区文件夹时 pair.db 被占用而失败）。
			if agentMgr != nil {
				agentMgr.CloseWorkspaceDB(root)
			}
			// 1. 从 RecentProjects 中移除
			newProjects := make([]string, 0, len(core.Settings.RecentProjects))
			for _, p := range core.Settings.RecentProjects {
				if p != root {
					newProjects = append(newProjects, p)
				}
			}
			core.Settings.RecentProjects = newProjects

			// 2. 从 WorkspaceFolders 中移除
			newFolders := make([]string, 0, len(core.Settings.WorkspaceFolders))
			for _, f := range core.Settings.WorkspaceFolders {
				if f != root {
					newFolders = append(newFolders, f)
				}
			}
			core.Settings.WorkspaceFolders = newFolders

			// 3. 从 core.Folders 中移除
			core.Folders = newFolders

			// 4. 如果 LastProject 匹配，清空（避免 health 接口仍返回已删除的工作区）
			if core.Settings.LastProject == root {
				core.Settings.LastProject = ""
				if len(core.Folders) > 0 {
					core.Settings.LastProject = core.Folders[0]
				}
			}
			if core.Settings.LastProject == "" {
				core.Loaded = false
			}
			core.Save()

			// 5. 可选：删除工作区下的 .pair 目录（对话历史等）
			if req.DeleteFiles {
				pairDir := filepath.Join(root, ".pair")
				if stat, err := os.Stat(pairDir); err == nil && stat.IsDir() {
					if err := os.RemoveAll(pairDir); err != nil {
						log.Printf("[Workspace] 删除 %s 失败: %v", pairDir, err)
					} else {
						log.Printf("[Workspace] 已删除 %s", pairDir)
					}
				}
			}

			// 同步工作区变更
			if core.OnSyncWorkspace != nil {
				core.OnSyncWorkspace(true)
			}
			jsonResp(w, map[string]any{"ok": true})

		default:
			if req.Root != "" {
				req.Root = filepath.Clean(req.Root) // 归一化（switch/打开工作区：防双反斜杠污染）
				flds := make([]string, 0, len(req.Folders)+1)
				flds = append(flds, req.Root)
				for _, f := range req.Folders {
					if f != "" && f != req.Root {
						flds = append(flds, filepath.Clean(f))
					}
				}
				core.Folders = flds
				core.Settings.LastProject = req.Root
				core.Settings.WorkspaceFolders = core.Folders
				// ★ 2026-08-22 同步 workspaceFolderLists（切换后刷新数据一致）：
				//   该工作区的文件夹列表快照以切换时提交的 folders 为准。
				core.SyncWorkspaceFolderList(req.Root, core.Folders)
				core.Loaded = true
				core.Save()
				if core.OnSyncWorkspace != nil {
					core.OnSyncWorkspace(true)
				}
			}
			jsonResp(w, map[string]any{"ok": true})
		}
	default:
		jsonErr(w, "不支持的方法")
	}
}

func (s *webServer) handleSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		// ★ 2026-08-16：附带插件注册的配置段（schemas）供前端设置面板动态渲染
		jsonResp(w, map[string]any{
			"settings": core.Settings,
			"loaded":   core.Loaded,
			"schemas":  core.PluginSettingSchemas,
		})
	case "PUT":
		// convId 参数可选：来自工具栏切换时传当前对话 ID。
		// ★ 2026-08-31 会话级：带 convId 的提交（reviewMode 等）只作用于本会话——
		//   不再写全局 settings（此前写全局导致其他会话默认被改且重启丢失）。
		convId := r.URL.Query().Get("convId")
		var reqRoot string
		if rr := r.URL.Query().Get("root"); rr != "" {
			reqRoot = filepath.Clean(rr)
		} else {
			reqRoot = core.Root()
		}
		// 读取原始请求体
		body, err := io.ReadAll(r.Body)
		if err != nil {
			jsonErr(w, err.Error())
			return
		}
		// 解析为 map 判断哪些字段被提交
		var rawMap map[string]json.RawMessage
		if err := json.Unmarshal(body, &rawMap); err != nil {
			jsonErr(w, err.Error())
			return
		}

		// ★ 配置插件化（2026-08-19）嵌套格式支持：{ settings: {...}, pluginSettings: {...} }
		//   · settings 对象：AppSettings 顶层字段（配置插件 binding 字段收集），
		//     展开进 rawMap 参与下方反射增量 merge（类型精度保留）
		//   · pluginSettings 对象：插件命名空间值（ctx.setSettings 写入），整体合并
		if sRaw, ok := rawMap["settings"]; ok {
			var sMap map[string]json.RawMessage
			if err := json.Unmarshal(sRaw, &sMap); err == nil {
				for k, v := range sMap {
					rawMap[k] = v
				}
			}
		}
		if psRaw, ok := rawMap["pluginSettings"]; ok {
			var ps map[string]map[string]any
			if err := json.Unmarshal(psRaw, &ps); err == nil {
				if core.Settings.PluginSettings == nil {
					core.Settings.PluginSettings = map[string]map[string]any{}
				}
				for k, v := range ps {
					core.Settings.PluginSettings[k] = v
				}
			}
		}

		// ★ 2026-08-31 会话级审核模式：convId 非空时 reviewMode 从全局 merge
		//   中摘除——会话内切换只写本会话元数据（持久化）+ 实时更新当前 Loop，
		//   不污染全局默认（此前写全局导致其他会话默认被改、重启丢失选择）。
		convReviewMode := ""
		if convId != "" {
			if rawVal, ok := rawMap["reviewMode"]; ok {
				var newVal string
				if json.Unmarshal(rawVal, &newVal) == nil {
					convReviewMode = newVal
				}
				delete(rawMap, "reviewMode")
			}
		}

		// 增量 merge：只更新 rawMap 中存在的字段
		sv := reflect.ValueOf(&core.Settings).Elem()
		st := sv.Type()
		for i := 0; i < sv.NumField(); i++ {
			field := st.Field(i)
			jsonKey := strings.Split(field.Tag.Get("json"), ",")[0]
			if jsonKey == "" || jsonKey == "-" {
				continue
			}
			rawVal, ok := rawMap[jsonKey]
			if !ok {
				continue
			}
			// ★ 2026-09-22 修复：pluginSettings 已在上方按「段」合并（插件命名空间值），
			//   此处若再走反射整体替换，会把请求体里**未出现**的插件段整体清掉——
			//   实测：只提交 {pluginSettings:{autopilot:{...}}} → settings.json 里
			//   agentloop / generation 两段消失（用户配置静默丢失）。
			//   语义以「按段合并」为准（与 internal/server/handler/workspace.go 一致）。
			if jsonKey == "pluginSettings" {
				continue
			}
			// 用 json.Unmarshal 解析到字段类型，保留类型精度
			newVal := reflect.New(field.Type).Interface()
			if err := json.Unmarshal(rawVal, newVal); err == nil {
				sv.Field(i).Set(reflect.ValueOf(newVal).Elem())
			}
		}

		// ★ 路径归一化（2026-08-21）：旧页面（未含前端 normPath）提交的双反斜杠路径
		//   在此折叠为单反斜杠，防污染传播到配置（recentProjects/workspaceFolders/
		//   workspaceFolderLists 全量清理；前端新版本已自愈，此处后端兜底）。
		cleanPathList := func(ss []string) []string {
			out := make([]string, 0, len(ss))
			for _, p := range ss {
				if p != "" {
					out = append(out, filepath.Clean(p))
				}
			}
			return out
		}
		if core.Settings.RecentProjects != nil {
			core.Settings.RecentProjects = cleanPathList(core.Settings.RecentProjects)
		}
		if core.Settings.WorkspaceFolders != nil {
			core.Settings.WorkspaceFolders = cleanPathList(core.Settings.WorkspaceFolders)
		}
		if core.Settings.WorkspaceFolderLists != nil {
			lists := make(map[string][]string, len(core.Settings.WorkspaceFolderLists))
			for k, v := range core.Settings.WorkspaceFolderLists {
				lists[filepath.Clean(k)] = cleanPathList(v)
			}
			core.Settings.WorkspaceFolderLists = lists
		}

		core.Save()
		// ★ 2026-08-31 会话级审核模式：convId 非空 → 持久化到会话元数据 +
		//   实时更新当前 Loop（幂等：值未变不落盘）；convId 空 → 全局模式
		//   （已在 merge 写入 core.Settings.ReviewMode 并 Save，无需更新 Loop）。
		if convId != "" && convReviewMode != "" {
			if store := agentMgr.StoreFor(reqRoot); store != nil {
				if err := store.SetConvReviewMode(convId, convReviewMode); err != nil {
					log.Printf("[settings] 会话审核模式持久化失败 conv=%s: %v", convId, err)
				}
			}
			agentMgr.SetReviewMode(convId, convReviewMode)
		}
		// 同步工作区文件夹列表（确保 core.Folders 与 settings 一致）
		if _, ok := rawMap["workspaceFolders"]; ok && core.Settings.WorkspaceFolders != nil {
			core.Folders = core.Settings.WorkspaceFolders
			if core.OnSyncWorkspace != nil {
				core.OnSyncWorkspace(false)
			}
		}
		jsonResp(w, map[string]any{"ok": true})
	default:
		jsonErr(w, "不支持的方法")
	}
}

func (s *webServer) handleSysInfo(w http.ResponseWriter, r *http.Request) {
	host, _ := os.Hostname()
	cwd, _ := os.Getwd()
	jsonResp(w, map[string]any{
		"hostname":  host,
		"cwd":       cwd,
		"os":        "windows",
		"goos":      runtime.GOOS,
		"goarch":    runtime.GOARCH,
		"goversion": runtime.Version(),
		"workspace": core.Root(),
		"folders":   core.Folders,
		"version":   version,
	})
}

func (s *webServer) handleTasks(w http.ResponseWriter, r *http.Request) {
	root := core.Root()
	if root == "" {
		jsonResp(w, map[string]any{"tasks": []any{}})
		return
	}
	switch r.Method {
	case "GET":
		// 从 TaskManager 持久化目录 .pair/tasks/*.json 读取真实任务状态
		convID := r.URL.Query().Get("convId")
		type planStep struct {
			Step        string `json:"step"`
			Status      string `json:"status"`
			TaskID      string `json:"taskId"`
			Description string `json:"description"`
			CreatedAt   string `json:"created_at"`
		}
		tasksDir := filepath.Join(root, ".pair", "tasks")
		entries, err := os.ReadDir(tasksDir)
		if err != nil {
			jsonResp(w, map[string]any{"tasks": []any{}})
			return
		}
		plan := make([]planStep, 0)
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
				continue
			}
			data, err := os.ReadFile(filepath.Join(tasksDir, e.Name()))
			if err != nil {
				continue
			}
			var t struct {
				ID          string `json:"id"`
				Subject     string `json:"subject"`
				Description string `json:"description"`
				Status      string `json:"status"`
				ConvID      string `json:"convId"`
				CreatedAt   string `json:"created_at"`
			}
			if err := json.Unmarshal(data, &t); err != nil {
				continue
			}
			// 按对话过滤：convID 非空时只返回该对话的任务
			if convID != "" && t.ConvID != convID {
				continue
			}
			plan = append(plan, planStep{
				Step:        t.Subject,
				Status:      t.Status,
				TaskID:      t.ID,
				Description: t.Description,
				CreatedAt:   t.CreatedAt,
			})
		}
		// 按创建时间倒序排列（最新的在前）
		sort.Slice(plan, func(i, j int) bool { return plan[i].CreatedAt > plan[j].CreatedAt })
		jsonResp(w, map[string]any{"tasks": plan})
	default:
		jsonErr(w, "不支持的方法")
	}
}

// ─── 终端命令执行 API ──────────────────────────────────────────

func (s *webServer) handleExec(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonErr(w, "仅 POST")
		return
	}
	var req struct {
		Command string `json:"command"`
		Cwd     string `json:"cwd"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, err.Error())
		return
	}
	if req.Command == "" {
		jsonErr(w, "command 必填")
		return
	}
	currDir := req.Cwd
	if currDir == "" {
		currDir = core.Root()
		if currDir == "" {
			currDir, _ = os.Getwd()
		}
	}

	cmdName := "cmd.exe"
	args := []string{"/C", req.Command}

	cmd := exec.CommandContext(r.Context(), cmdName, args...)
	cmd.Dir = currDir
	// ★ 2026-08-19：cmd.exe 是 console 程序——无控制台父进程时会弹窗，显式隐藏。
	executil.HideWindow(cmd)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	result := map[string]any{
		"stdout":   stdout.String(),
		"stderr":   stderr.String(),
		"exitCode": 0,
		"cwd":      currDir,
	}
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result["exitCode"] = exitErr.ExitCode()
		} else {
			result["stderr"] = err.Error()
			result["exitCode"] = -1
		}
	}
	jsonResp(w, result)
}

// ─── 对话列表 API ────────────────────────────────────────────

func (s *webServer) handleConversations(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		wsFilter := r.URL.Query().Get("workspace")
		// ★ 2026-08-23 工作区隔离：列表按请求工作区路由 store（此前全局 Store()
		//   → 切换后查询旧工作区对话拿到的是新工作区存储，列表为空/串数据）。
		store := agentMgr.StoreFor(wsFilter)
		if store == nil {
			jsonResp(w, []agent.ConversationMeta{})
			return
		}
		metas, err := store.ListConversations(wsFilter)
		if err != nil {
			jsonErr(w, err.Error())
			return
		}
		if metas == nil {
			metas = []agent.ConversationMeta{}
		}
		jsonResp(w, metas)

	case "POST":
		var req struct {
			ID            string `json:"id"`
			Title         string `json:"title"`
			WorkspaceRoot string `json:"workspaceRoot"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonErr(w, err.Error())
			return
		}
		store := agentMgr.StoreFor(req.WorkspaceRoot)
		if store == nil {
			jsonErr(w, "消息存储未初始化")
			return
		}
		id := req.ID
		if id == "" {
			id = fmt.Sprintf("conv_%d", time.Now().UnixNano())
		}
		wsRoot := req.WorkspaceRoot
		if wsRoot == "" {
			wsRoot = core.Root()
		}
		title := req.Title
		if title == "" {
			title = "新对话 " + time.Now().Format("15:04")
		}
		if err := store.CreateConversation(id, title, wsRoot); err != nil {
			jsonErr(w, err.Error())
			return
		}
		jsonResp(w, map[string]any{"ok": true, "id": id, "title": title})

	default:
		jsonErr(w, "不支持的方法")
	}
}

// slimStoredMessages 响应瘦身：省略前端不消费的重字段（slim=false 原样返回）。
//
// 实测（2026-09-25，单会话 50 条合并消息 = 19.57MB 原始 JSON 响应）：
//
//	segments                     8.86M 字符（61%，前端渲染必需 → 保留）
//	message.reasoning_content    3.32M 字符（24%）★ 与 segments 的 thinking 段同内容
//	message.tool_calls           1.78M 字符（13%）★ 与 segments 的 tool_call 段重复
//	message.content              0.28M 字符（ 2%）
//
// 前端 apiLoadAndBuildConv 只读 message 的 role/content/images + segments
// （grep 全前端无 reasoning_content/tool_calls 消费点）→ 后两者可安全省略（约省 37%）。
//
// ★ 仅在 HTTP 响应层裁剪：Message.Reasoning 是 DeepSeek 工具调用轮次回传契约要求的
//
//	字段，磁盘 JSONL 必须完整保留（见 types.go 注释与 copyHistoryNoReasoning）；
//	displayMessages 每次从磁盘重新反序列化 → 此处清空不影响缓存与后续请求。
func slimStoredMessages(msgs []agent.StoredMessage, slim bool) []agent.StoredMessage {
	if !slim {
		return msgs
	}
	for i := range msgs {
		msgs[i].Message.Reasoning = ""
		msgs[i].Message.ToolCalls = nil
		// ★ 2026-09-25 性能（段级惰性加载）：单条合并消息可达 1.5MB，瓶颈在**单条内的
		//   超长段**而非条数（实测 50 条中前 30 条占 100% ⇒ 缩小 limit 无效）。
		//   仅裁「折叠态完全不消费」的字段，保证可见交互零变化：
		//     · thinking.content  —— 折叠态只渲染「思考…」标签（RightPanel 模板
		//       `v-if="!seg._collapsed"` 才输出 content）；消息折叠摘要 msgSummary
		//       也不用它（只用 tool_call 计数 + content 段前 60 字符）。
		//     · tool_call.argsRaw —— 仅在展开时渲染「参数」（同理由 v-if 守卫）；
		//       折叠行的摘要来自 Result（前 120 字符）与工具名，与本字段无关。
		//     · tool_call.result —— 折叠行只用它做三件事：`toolResultSummary`
		//       胶囊文案（前 120 字符）、未知工具的 summary（前 80 字符）、
		//       /错误|失败|error…/.test(result) 判错误色（**跑全文**）。
		//       前两者 400 字符预览足够；后者改由后端按全文预计算 `_err`
		//       （见 agent.Segment.Err）保证判定等价 —— 实测该会话有 156 段的
		//       错误关键词只落在 400 字符之后，只留预览会把它们误判成成功色。
		//       实测可裁 1069 段 / 2.79M 字符，裁后省约 2.36M 字符（占 result
		//       总量 2.89M 的 81.7%）→ 剩余体积的最大头。
		//       ★ finish_task 例外：其 result 会被前端转成正文 content 段
		//       （RightPanel apiLoadAndBuildConv）→ 必须保留全文。
		//   预览长度 400：实测 thinking p50=2261 / argsRaw p50=431 ⇒ 多数 thinking 段
		//   折叠时无需取全文，而 argsRaw 约半数完整保留，减少展开时的按需请求次数。
		segs := msgs[i].Segments
		for j := range segs {
			switch segs[j].Type {
			case "thinking":
				if n := len([]rune(segs[j].Content)); n > slimSegPreviewRunes {
					segs[j].Content = cutRunes(segs[j].Content, slimSegPreviewRunes)
					segs[j].TruncLen = n
				}
			case "tool_call":
				if n := len([]rune(segs[j].ArgsRaw)); n > slimSegPreviewRunes {
					segs[j].ArgsRaw = cutRunes(segs[j].ArgsRaw, slimSegPreviewRunes)
					segs[j].TruncLen = n
				}
				if segs[j].Name != "finish_task" {
					if n := len([]rune(segs[j].Result)); n > slimSegPreviewRunes {
						// ★ 错误色判定必须按**全文**（在裁断之前完成），否则关键词
						//   落在预览之外时会被漏判（前端只拿得到预览片段）。
						errFlag := toolResultErrRe.MatchString(segs[j].Result)
						segs[j].Result = cutRunes(segs[j].Result, slimSegPreviewRunes)
						segs[j].Err = &errFlag
						if segs[j].TruncLen == 0 {
							segs[j].TruncLen = n
						}
					}
				}
			}
		}
	}
	return msgs
}

// slimSegPreviewRunes 折叠段首包预览的字符数（见 slimStoredMessages）。
const slimSegPreviewRunes = 400

// toolResultErrRe 工具胶囊「错误色」的关键词正则，与前端 RightPanel.vue 的判定
// （/错误|失败|error|Error|✗|Exception/）逐词一致（无 flag 的字面量交替）。
// result 在 slim 响应中只回预览片段，该判定却必须作用于**全文** —— 故由
// slimStoredMessages 在裁断之前调用本正则预计算出 _err（见 agent.Segment.Err）。
var toolResultErrRe = regexp.MustCompile(`错误|失败|error|Error|✗|Exception`)

// cutRunes 按**字符**（非字节）截断，避免切断多字节字符产生乱码。
func cutRunes(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max])
}

func (s *webServer) handleConversationByID(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/conversations/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		jsonErr(w, "缺少对话 ID")
		return
	}
	id := parts[0]
	sub := ""
	if len(parts) >= 2 {
		sub = parts[1]
	}
	subSub := ""
	if len(parts) >= 3 {
		subSub = parts[2]
	}

	// ★ 2026-08-23 工作区隔离：按请求工作区根路由 store（前端对话加载/回放带
	//   workspaceRoot 参数；缺省=当前工作区）。运行中切换工作区后回放旧工作区
	//   对话不再落到新工作区存储。
	wsRoot := r.URL.Query().Get("workspaceRoot")
	store := agentMgr.StoreFor(wsRoot)
	if store == nil {
		jsonErr(w, "消息存储未初始化")
		return
	}

	switch r.Method {
	case "GET":
		switch {
		case sub == "messages" && subSub == "count":
			n, err := store.Count(id)
			if err != nil {
				jsonErr(w, err.Error())
				return
			}
			jsonResp(w, map[string]any{"count": n})

		case sub == "messages" && subSub == "segment":
			// ★ 2026-09-25 性能（折叠段惰性加载）：取单段全文。
			//   GET /api/conversations/<id>/messages/segment?idx=<消息 idx>&seg=<段序号>&limit=<窗口>
			//   必须走与 /messages **完全相同**的读取+合并管道并传同一 limit →
			//   合并后的 idx / 段序号才能与前端手中的对象一一对应（否则展开时会取错段）。
			//   本接口**不做 slim 裁断**（返回原始完整字段），故任何片段都能取回全文。
			var midx, segIdx, lim int
			fmt.Sscanf(r.URL.Query().Get("idx"), "%d", &midx)
			fmt.Sscanf(r.URL.Query().Get("seg"), "%d", &segIdx)
			lim = 50
			if l := r.URL.Query().Get("limit"); l != "" {
				fmt.Sscanf(l, "%d", &lim)
			}
			full, _, err := store.LoadLatestForDisplay(id, lim)
			if err != nil {
				jsonErr(w, err.Error())
				return
			}
			for _, m := range full {
				if m.Idx != midx {
					continue
				}
				if segIdx < 0 || segIdx >= len(m.Segments) {
					jsonErr(w, "段序号越界")
					return
				}
				s := m.Segments[segIdx]
				jsonResp(w, map[string]any{
					"type": s.Type, "content": s.Content, "argsRaw": s.ArgsRaw, "result": s.Result,
				})
				return
			}
			jsonErr(w, "未找到该消息")

		case sub == "messages":
			limit := 50
			if l := r.URL.Query().Get("limit"); l != "" {
				fmt.Sscanf(l, "%d", &limit)
			}
			// ★ 2026-09-25 性能（接口瘦身）：slim=1 裁剪前端不消费的重字段，
			//   实测该接口单次响应 19.57MB → 裁剪后约 12.3MB（−37%）。
			slim := r.URL.Query().Get("slim") == "1"
			beforeStr := r.URL.Query().Get("before")
			if beforeStr != "" {
				var before int
				fmt.Sscanf(beforeStr, "%d", &before)
				// ★ 2026-08-22 展示专用：全量合并后再按合并条数 limit
				//（此前按原始行 limit，长对话每轮只返回 1~5 条超大合并消息，加载慢）
				msgs, err := store.LoadBeforeForDisplay(id, before, limit)
				if err != nil {
					jsonErr(w, err.Error())
					return
				}
				if msgs == nil {
					msgs = []agent.StoredMessage{}
				}
				total, _ := store.Count(id)
				jsonResp(w, map[string]any{"messages": slimStoredMessages(msgs, slim), "total": total})
			} else {
				msgs, total, err := store.LoadLatestForDisplay(id, limit)
				if err != nil {
					jsonErr(w, err.Error())
					return
				}
				if msgs == nil {
					msgs = []agent.StoredMessage{}
				}
				jsonResp(w, map[string]any{"messages": slimStoredMessages(msgs, slim), "total": total})
			}

		case sub == "token-stats":
			// ★ 生效上下文窗口真值（后端装配口径，2026-09-24）：
			//   一律取服务商配置（models.json）的装配结果（模型级 > 服务商级），
			//   未配置时回退机制常量 core.DefaultContextWindow（见 agent.ContextWindow）。
			//   前端不得再读 settings 顶层 contextMaxTokens（该字段已迁入插件注册域、
			//   按机制不参与窗口取值），更不得硬编码兜底——旧前端 `|| 1000000`
			//   会显示与后端实际生效口径不一致的假上限（本次修复的臆测点）。
			ctxWindow := agent.ContextWindow(agent.ResolveProviderParamsForConv(id, wsRoot))
			meta, err := store.GetConversation(id)
			if err != nil {
				jsonErr(w, err.Error())
				return
			}
			if meta == nil || meta.CtxStats == nil {
				jsonResp(w, map[string]any{
					"promptTokens": 0, "completionTokens": 0, "totalTokens": 0,
					"cacheHitTokens": 0, "cacheMissTokens": 0,
					"contextMaxTokens": ctxWindow,
				})
				return
			}
			cs := meta.CtxStats
			m := map[string]any{
				"promptTokens":     cs.PromptTokens,
				"completionTokens": cs.CompletionTokens,
				"totalTokens":      cs.TotalTokens,
				"cacheHitTokens":   cs.PromptCacheHitTokens,
				"cacheMissTokens":  cs.PromptCacheMissTokens,
				"contextMaxTokens": ctxWindow,
			}
			if cs.PromptBreakdown.SystemTokens > 0 || cs.PromptBreakdown.SkillsTokens > 0 ||
				cs.PromptBreakdown.MCPTokens > 0 || cs.PromptBreakdown.ToolTokens > 0 {
				m["systemTokens"] = cs.SystemTokens
				m["skillsTokens"] = cs.SkillsTokens
				m["mcpTokens"] = cs.MCPTokens
				m["toolTokens"] = cs.ToolTokens
				m["historyTokens"] = cs.HistoryTokens
				m["otherTokens"] = cs.OtherTokens
			}
			jsonResp(w, m)

		case sub == "run-stats":
			// ★ 2026-09-12 后端运行统计（run_stats.go）：每会话「最近一次运行」的
			//   耗时 / 步数 / 工具调用（次数 + 累计耗时）/ LLM 调用（次数 + 生成耗时）/
			//   token 用量 / 输出速度（t/s 由后端派生：生成阶段耗时为分母）。
			//   运行中返回内存实时累加值；结束后为定格值（已落盘 .pair/run-stats.json，
			//   刷新页面 / 重启 IDE 后仍可读——前端不再自行计算与持久化）。
			statsRoot := wsRoot
			if statsRoot == "" {
				statsRoot = core.Root()
			}
			jsonResp(w, agent.RunStatsFor(statsRoot, id))

		case sub == "":
			meta, err := store.GetConversation(id)
			if err != nil {
				jsonErr(w, err.Error())
				return
			}
			if meta == nil {
				// ★ 兜底：workspaceRoot 参数缺失/错位（如旧前端把配置名传入）
				//   时 StoreFor 落空——跨已打开 store 找回会话，避免误报「对话不存在」。
				if m2, _ := agentMgr.FindConversation(id); m2 != nil {
					jsonResp(w, m2)
					return
				}
				jsonErr(w, "对话不存在")
				return
			}
			jsonResp(w, meta)

		default:
			jsonErr(w, "未知的子路径: "+sub)
		}

	case "PUT":
		var req struct {
			Title string `json:"title"`
			// ★ 2026-08-31 会话级模型：切模型只改本会话（provider+model 一起提交；
			//   两者均为空串 = 清除会话覆盖 → 回落全局默认配置）。
			// ★ 2026-09-03 preset=所选 AI 配置名（装配按配置整套展开，含该配置 Key）。
			Provider   *string `json:"provider,omitempty"`
			Model      *string `json:"model,omitempty"`
			Preset     *string `json:"preset,omitempty"`
			ClearModel bool    `json:"clearModel,omitempty"`
			// ★ 2026-09-04 会话级工具集（通用集合）：toolset 字段写会话元数据，
			//   agent 工具面按所选集合收敛；空串=清除（回落 default 集合）。
			Toolset *string `json:"toolset,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonErr(w, err.Error())
			return
		}
		if req.Title != "" {
			if err := store.UpdateTitle(id, req.Title); err != nil {
				jsonErr(w, err.Error())
				return
			}
		}
		if req.Toolset != nil {
			// ★ 跨 store 兜底（与 SetConvModel 同策略）
			saveStore := store
			if meta, _ := store.GetConversation(id); meta == nil {
				if m2, _ := agentMgr.FindConversation(id); m2 != nil {
					if ws := m2.WorkspaceRoot; ws != "" {
						saveStore = agentMgr.StoreFor(ws)
					}
				}
			}
			if err := saveStore.SetConvToolset(id, strings.TrimSpace(*req.Toolset)); err != nil {
				jsonErr(w, err.Error())
				return
			}
			log.Printf("[conv-toolset] 会话 %s 工具集设为 %q（仅本会话生效）", id, strings.TrimSpace(*req.Toolset))
		}
		if req.ClearModel {
			if err := store.SetConvModel(id, "", "", ""); err != nil {
				jsonErr(w, err.Error())
				return
			}
		} else if req.Provider != nil || req.Model != nil || req.Preset != nil {
			meta, _ := store.GetConversation(id)
			prov, model, preset := "", "", ""
			saveStore := store
			if meta != nil {
				prov, model, preset = meta.Provider, meta.Model, meta.Preset
			} else if m2, _ := agentMgr.FindConversation(id); m2 != nil {
				// ★ 兜底：workspaceRoot 参数缺失/错位（旧前端把配置名传入）时，
				//   跨 store 找回会话并落到其真实所属 store 写入，避免误报「会话不存在」。
				prov, model, preset = m2.Provider, m2.Model, m2.Preset
				if ws := m2.WorkspaceRoot; ws != "" {
					saveStore = agentMgr.StoreFor(ws)
				}
			}
			if req.Provider != nil {
				prov = strings.TrimSpace(*req.Provider)
			}
			if req.Model != nil {
				model = strings.TrimSpace(*req.Model)
			}
			if req.Preset != nil {
				preset = strings.TrimSpace(*req.Preset)
			}
			if err := saveStore.SetConvModel(id, prov, model, preset); err != nil {
				jsonErr(w, err.Error())
				return
			}
			log.Printf("[conv-model] 会话 %s 模型切换为 %s / %s（配置 %s，仅本会话生效）", id, prov, model, preset)
			// ★ 2026-09-03 策略：运行中不切换——会话进行期间保持原模型，
			//   本次对话结束后发起下一次对话时按新落盘配置装配（Loop 创建时
			//   ResolveProviderParamsForConv 重新解析，见 handleChatSend 装配链路）。
		}
		jsonResp(w, map[string]any{"ok": true})

	case "POST":
		if sub != "messages" {
			jsonErr(w, "请使用 /conversations/{id}/messages")
			return
		}
		var msg struct {
			Role     string          `json:"role"`
			Content  string          `json:"content"`
			Segments []agent.Segment `json:"segments"`
		}
		if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
			jsonErr(w, err.Error())
			return
		}
		if msg.Role == "" {
			msg.Role = "user"
		}
		agentMsg := agent.Message{Role: agent.Role(msg.Role), Content: msg.Content}
		if err := store.AppendMessage(id, agentMsg, msg.Segments); err != nil {
			jsonErr(w, err.Error())
			return
		}
		jsonResp(w, map[string]any{"ok": true})

	case "DELETE":
		if err := store.DeleteConversation(id); err != nil {
			jsonErr(w, err.Error())
			return
		}
		// 同步删除记忆索引（跨对话摘要索引）
		memory.Delete(id)
		jsonResp(w, map[string]any{"ok": true})

	default:
		jsonErr(w, "不支持的方法")
	}
}

// ★ 2026-08-31：规划文档 API（/api/taskplan）已随 plan 体系移除——
//   任务追踪统一由 task 工具 + /api/tasks 承担（前端无调用点，纯死接口）。

// ─── 模型列表 API ──────────────────────────────────────────

func (s *webServer) handleModels(w http.ResponseWriter, r *http.Request) {
	// 委托共享实现：GET 读取 / POST/PUT 全量保存 → 落盘安装目录 config/models.json
	handler.HandleModels(w, r)
}

// handleModelsRename 服务商改名（委托共享实现）：改 models.json 键 + 同步 AI 配置里的 provider 引用
func (s *webServer) handleModelsRename(w http.ResponseWriter, r *http.Request) {
	handler.HandleModelsRename(w, r)
}

// handleAiPresets AI 配置预设 API（委托共享实现）：GET 查询 / POST 保存-应用-删除 / PUT 全量保存
func (s *webServer) handleAiPresets(w http.ResponseWriter, r *http.Request) {
	handler.HandleAiPresets(w, r)
}

// ─── 指令 API ──────────────────────────────────────────────

func (s *webServer) handleInstructions(w http.ResponseWriter, r *http.Request) {
	scope := r.URL.Query().Get("scope")
	if scope == "" {
		scope = "system"
	}

	switch scope {
	case "project":
		root := core.Root()
		if root == "" {
			jsonErr(w, "未设置工作区")
			return
		}
		pairDir := filepath.Join(root, ".pair")
		os.MkdirAll(pairDir, 0755)
		instPath := filepath.Join(pairDir, "instructions.md")

		switch r.Method {
		case "GET":
			data, err := os.ReadFile(instPath)
			if err != nil {
				jsonResp(w, map[string]any{"content": "", "path": instPath})
				return
			}
			jsonResp(w, map[string]any{"content": string(data), "path": instPath})

		case "PUT":
			var req struct {
				Content string `json:"content"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				jsonErr(w, err.Error())
				return
			}
			if err := os.WriteFile(instPath, []byte(req.Content), 0644); err != nil {
				jsonErr(w, err.Error())
				return
			}
			jsonResp(w, map[string]any{"ok": true, "path": instPath})

		default:
			jsonErr(w, "不支持的方法")
		}

	default:
		switch r.Method {
		case "GET":
			jsonResp(w, map[string]any{"content": core.Settings.SystemInstructions})

		case "PUT":
			var req struct {
				Content string `json:"content"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				jsonErr(w, err.Error())
				return
			}
			core.Settings.SystemInstructions = req.Content
			core.Save()
			jsonResp(w, map[string]any{"ok": true})

		default:
			jsonErr(w, "不支持的方法")
		}
	}
}

// ─── MCP 列表 API ──────────────────────────────────────────

func (s *webServer) handleMCPList(w http.ResponseWriter, r *http.Request) {
	lv := r.URL.Query().Get("level")
	type mcpItem struct {
		Name    string   `json:"name"`
		Command string   `json:"command"`
		Args    []string `json:"args"`
		Level   string   `json:"level"`
		Enabled bool     `json:"enabled"`
	}
	out := make([]mcpItem, 0)

	if lv == "" || lv == "all" || lv == "user" {
		for _, e := range mcppanel.ReadLevel(mcppanel.LevelUser) {
			out = append(out, mcpItem{
				Name: e.Name, Command: e.Command, Args: e.Args,
				Level: "user", Enabled: mcppanel.Enabled(mcppanel.LevelUser, e.Name),
			})
		}
	}
	if lv == "" || lv == "all" || lv == "project" {
		for _, e := range mcppanel.ReadLevel(mcppanel.LevelProject) {
			out = append(out, mcpItem{
				Name: e.Name, Command: e.Command, Args: e.Args,
				Level: "project", Enabled: mcppanel.Enabled(mcppanel.LevelProject, e.Name),
			})
		}
	}
	jsonResp(w, out)
}

// ─── MCP 保存/删除 API ────────────────────────────────────

func (s *webServer) handleMCPSave(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonErr(w, "仅 POST")
		return
	}
	var req struct {
		Action  string   `json:"action"`
		Name    string   `json:"name"`
		Command string   `json:"command"`
		Args    []string `json:"args"`
		Level   string   `json:"level"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, err.Error())
		return
	}
	lv := mcppanel.LevelUser
	if req.Level == "project" {
		lv = mcppanel.LevelProject
	}

	switch req.Action {
	case "delete":
		if err := mcppanel.Delete(lv, req.Name); err != nil {
			jsonErr(w, err.Error())
			return
		}
		jsonResp(w, map[string]any{"ok": true, "action": "deleted", "name": req.Name})
	case "toggle":
		old := mcppanel.Enabled(lv, req.Name)
		if err := mcppanel.SetEnabled(lv, req.Name, !old); err != nil {
			jsonErr(w, err.Error())
			return
		}
		jsonResp(w, map[string]any{"ok": true, "action": "toggled", "name": req.Name, "enabled": !old})
	default:
		if req.Name == "" || req.Command == "" {
			jsonErr(w, "name 和 command 必填")
			return
		}
		if err := mcppanel.Upsert(lv, mcppanel.Entry{
			Name: req.Name, Command: req.Command, Args: req.Args,
		}); err != nil {
			jsonErr(w, err.Error())
			return
		}
		jsonResp(w, map[string]any{"ok": true, "action": "saved", "name": req.Name})
	}
}

// ─── Token 统计 API（数据由 agent 自闭环持久化） ──────────────

func (s *webServer) handleTokensStats(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		// 从 agent 自闭环存储读取（优先使用 query 参数中的 workspaceRoot，兜底用 core.Root()）
		wsRoot := r.URL.Query().Get("workspaceRoot")
		if wsRoot == "" {
			wsRoot = core.Root()
		}
		stats := agent.ReadTokenStatsForRoot(wsRoot)
		if stats == nil {
			jsonResp(w, map[string]any{
				"promptTokens": 0, "completionTokens": 0, "totalTokens": 0,
				"cacheHitTokens": 0, "cacheMissTokens": 0,
				"systemTokens": 0, "skillsTokens": 0, "mcpTokens": 0,
				"toolTokens": 0, "historyTokens": 0, "otherTokens": 0,
			})
			return
		}
		jsonResp(w, map[string]any{
			"promptTokens":     stats.PromptTokens,
			"completionTokens": stats.CompletionTokens,
			"totalTokens":      stats.TotalTokens,
			"cacheHitTokens":   stats.CacheHitTokens,
			"cacheMissTokens":  stats.CacheMissTokens,
			"systemTokens":     stats.SystemTokens,
			"skillsTokens":     stats.SkillsTokens,
			"mcpTokens":        stats.MCPTokens,
			"toolTokens":       stats.ToolTokens,
			"historyTokens":    stats.HistoryTokens,
			"otherTokens":      stats.OtherTokens,
		})
	case "POST":
		jsonResp(w, map[string]any{"ok": true, "message": "Token 统计由 agent 自闭环管理，无需手动重置"})
	default:
		jsonErr(w, "不支持的方法")
	}

}

// ─── Skills HTTP API ──────────────────────────────────────
func (s *webServer) handleSkillsList(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		jsonErr(w, "仅 GET")
		return
	}
	type skillItem struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Mode        string `json:"mode"`
		Level       string `json:"level"`
		Status      string `json:"status"`
	}
	skills := agent.LoadAllSkills()
	out := make([]skillItem, 0, len(skills))
	for _, sk := range skills {
		origMode := sk.Mode
		key := string(sk.Level) + "::" + sk.Name
		status := "on"
		if s, ok := agent.SkillStatusOverride[key]; ok {
			status = s
		} else {
			if v, ok := agent.SkillEnabled[key]; ok && !v {
				status = "off"
			} else if origMode == "always" {
				status = "max"
			}
		}
		out = append(out, skillItem{
			Name: sk.Name, Description: sk.Description,
			Mode: origMode, Level: string(sk.Level),
			Status: status,
		})
	}

	jsonResp(w, out)
}

func (s *webServer) handleSkillsDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonErr(w, "仅 POST")
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, err.Error())
		return
	}
	if req.Name == "" {
		jsonErr(w, "name 必填")
		return
	}
	if err := agent.DeleteSkill(agent.SkillProjectDir, req.Name); err != nil {
		jsonErr(w, err.Error())
		return
	}
	jsonResp(w, map[string]any{"ok": true, "message": "已删除技能: " + req.Name})
}

func (s *webServer) handleSkillsSave(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonErr(w, "仅 POST")
		return
	}
	var req struct {
		Action string `json:"action"`
		Name   string `json:"name"`
		Level  string `json:"level"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, err.Error())
		return
	}
	if req.Name == "" {
		jsonErr(w, "name 必填")
		return
	}
	lv := agent.LevelProject
	if req.Level == "system" {
		lv = agent.LevelSystem
	}
	switch req.Action {
	case "set-status":
		if req.Status != "off" && req.Status != "on" && req.Status != "max" {
			jsonErr(w, "status 必须为 off/on/max")
			return
		}
		agent.SkillSetStatus(req.Name, lv, req.Status)
		core.Settings.SkillStatusOverrides = agent.SkillStatusOverride
		core.Settings.SkillEnabledOverrides = agent.SkillEnabled
		core.Save()
		jsonResp(w, map[string]any{"ok": true, "action": "set-status", "name": req.Name, "status": req.Status})
	default:
		jsonErr(w, "未知 action: "+req.Action)
	}
}

// ─── 辅助 ────────────────────────────────────────────────────

func jsonResp(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// handleSkillsRead 读取技能正文内容。
// 支持 level 查询参数：system（全局 config/skills/）或 project（工作区 .pair/skills/，默认）。
func (s *webServer) handleSkillsRead(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		jsonErr(w, "仅 GET")
		return
	}
	name := r.URL.Query().Get("name")
	if name == "" {
		jsonErr(w, "name 必填")
		return
	}
	// 白名单校验：仅允许字母、数字、下划线、中划线
	if !validSkillName(name) {
		jsonErr(w, "无效的键名")
		return
	}
	level := r.URL.Query().Get("level")
	var baseDir string
	switch level {
	case "system", "global":
		baseDir = agent.SkillSystemDir
	default:
		baseDir = agent.SkillProjectDir
	}
	skillPath := filepath.Join(baseDir, name, "SKILL.md")
	// 路径穿越防护：校验最终路径必须以 baseDir 开头
	absPath, err := filepath.Abs(skillPath)
	if err != nil || !strings.HasPrefix(absPath, filepath.Clean(baseDir)+string(filepath.Separator)) {
		jsonErr(w, "请求的资源不存在")
		return
	}
	data, err := os.ReadFile(absPath)
	if err != nil {
		// 默认(project)级别找不到时回退系统级
		if level == "" || level == "project" {
			if agent.SkillSystemDir != "" {
				fallback := filepath.Join(agent.SkillSystemDir, name, "SKILL.md")
				absFallback, fbErr := filepath.Abs(fallback)
				if fbErr == nil && strings.HasPrefix(absFallback, filepath.Clean(agent.SkillSystemDir)+string(filepath.Separator)) {
					data, fbErr = os.ReadFile(absFallback)
					if fbErr == nil {
						jsonResp(w, map[string]string{"name": name, "content": string(data), "level": "system"})
						return
					}
				}
			}
		}
		jsonErr(w, "请求的资源不存在")
		return
	}
	respLevel := level
	if respLevel == "" {
		respLevel = "project"
	}
	jsonResp(w, map[string]string{"name": name, "content": string(data), "level": respLevel})
}

// validSkillName 校验技能名仅含字母、数字、下划线、中划线。
func validSkillName(name string) bool {
	if name == "" {
		return false
	}
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-') {
			return false
		}
	}
	return true
}

func jsonErr(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func jsonStr(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// detectLang 根据项目名或目录内容推测语言。
func detectLang(name string) string {
	name = strings.ToLower(name)
	if strings.Contains(name, "go") || strings.HasSuffix(name, ".go") {
		return "go"
	}
	if strings.Contains(name, "rust") || strings.Contains(name, "rs") {
		return "rust"
	}
	if strings.Contains(name, "py") || strings.Contains(name, "python") {
		return "python"
	}
	if strings.Contains(name, "js") || strings.Contains(name, "node") || strings.Contains(name, "react") || strings.Contains(name, "vue") {
		return "javascript"
	}
	if strings.Contains(name, "ts") || strings.Contains(name, "typescript") {
		return "typescript"
	}
	if strings.Contains(name, "java") {
		return "java"
	}
	if strings.Contains(name, "html") || strings.Contains(name, "web") {
		return "html"
	}
	return "unknown"
}

// genProjectTemplate 在项目目录中生成初始模板文件。
func genProjectTemplate(projPath, lang, name string) {
	switch lang {
	case "go":
		modName := name
		if modName == "" || modName == "unknown" {
			modName = filepath.Base(projPath)
		}
		mainGo := fmt.Sprintf(`package main

import "fmt"

func main() {
	fmt.Println("Hello from %s!")
}
`, modName)
		os.WriteFile(filepath.Join(projPath, "main.go"), []byte(mainGo), 0644)
		os.WriteFile(filepath.Join(projPath, "go.mod"), []byte(fmt.Sprintf("module %s\n\ngo 1.24\n", modName)), 0644)

	case "python":
		os.WriteFile(filepath.Join(projPath, "main.py"), []byte(fmt.Sprintf(`# %s
def main():
    print("Hello from %s!")

if __name__ == "__main__":
    main()
`, name, name)), 0644)

	case "javascript", "typescript":
		os.WriteFile(filepath.Join(projPath, "index.js"), []byte(fmt.Sprintf(`// %s
console.log("Hello from %s!");
`, name, name)), 0644)

	case "html":
		os.WriteFile(filepath.Join(projPath, "index.html"), []byte(fmt.Sprintf(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>%s</title>
</head>
<body>
  <h1>Hello from %s!</h1>
</body>
</html>`, name, name)), 0644)

	case "java":
		className := "App"
		if name != "" && name != "unknown" {
			parts := strings.Split(name, "-")
			for i, p := range parts {
				if len(p) > 0 {
					parts[i] = strings.ToUpper(p[:1]) + p[1:]
				}
			}
			className = strings.Join(parts, "")
		}
		os.WriteFile(filepath.Join(projPath, className+".java"), []byte(fmt.Sprintf(`public class %s {
    public static void main(String[] args) {
        System.out.println("Hello from %s!");
    }
}
`, className, name)), 0644)
	}
}

// loadConversationHistory 从 MessageStore 加载对话的全部过往消息作为 LLM 上下文。
// 委托给 agentMgr.Store().LoadAll(convID)；store 未注入时返回 nil。
func (s *webServer) loadConversationHistory(convID string) []agent.Message {
	if convID == "" {
		return nil
	}
	store := agentMgr.Store()
	if store == nil {
		return nil
	}
	msgs, err := store.LoadAll(convID)
	if err != nil {
		return nil
	}
	return msgs
}

// ── 统一 API 层（平台无关，差异通过回调参数化） ─────────────────

// systemStaticPrefixCache 缓存 system prompt 静态前缀（CACHE_BOUNDARY 之前部分）。
// 键由 workspace roots hash + SystemInstructions 构成；
// 只要这些不变，静态前缀就无需重新拼接。
var systemStaticPrefixCache struct {
	mu     sync.Mutex
	key    string
	prefix string
}

func systemStaticPrefixKey() string {
	si := core.Settings.SystemInstructions
	roots := strings.Join(core.Folders, "|")
	return fmt.Sprintf("%s|%s", roots, si)
}

// buildSystemStaticPrefix 构建 system prompt 静态前缀（CACHE_BOUNDARY 之前）。
// 结果会被缓存，避免每次请求重复拼接字符串。
func buildSystemStaticPrefix() string {
	k := systemStaticPrefixKey()
	systemStaticPrefixCache.mu.Lock()
	defer systemStaticPrefixCache.mu.Unlock()
	if systemStaticPrefixCache.key == k && systemStaticPrefixCache.prefix != "" {
		return systemStaticPrefixCache.prefix
	}
	var b strings.Builder
	// ★ persona/rules 槽位（对齐 system-prompt 可替换 section）：
	//   插件贡献 name==deployment:persona 的段 → 替换默认 persona（身份段）；
	//   插件贡献 name==deployment:rules 的段 → 替换默认规则段（行为准则）。
	//   两者独立可组合；无贡献时输出与默认提示逐字节一致（静态前缀稳定）。
	persona, rules := "", ""
	if ph := handler.GetPluginHost(); ph != nil {
		if sec := ph.PersonaSection(); sec != nil {
			persona = sec.Text
		}
		if sec := ph.RulesSection(); sec != nil {
			rules = sec.Text
		}
	}
	b.WriteString(agent.DefaultSystemPromptWithOverrides(core.Folders, persona, rules)) // 工作区路径基本固定，放 system prompt 静态前缀可最大化 KV Cache 命中
	if si := strings.TrimSpace(core.Settings.SystemInstructions); si != "" {
		b.WriteString("\n\n# 系统级指令（务必遵守）\n" + si)
	}
	b.WriteString(agent.SelfManagementPrompt())
	// 注意：不在此追加 CacheBoundary——唯一 boundary 由 ComposeSystemPrompt 统一添加，
	// 确保 SystemInstructions 等静态内容全部位于 boundary 之前（同一静态前缀内）。
	systemStaticPrefixCache.key = k
	systemStaticPrefixCache.prefix = b.String()
	return systemStaticPrefixCache.prefix
}

// truncateStr 截断字符串到 maxRunes（诊断日志用）。
func truncateStr(s string, maxRunes int) string {
	r := []rune(s)
	if len(r) <= maxRunes {
		return s
	}
	return string(r[:maxRunes]) + "…"
}

// buildWebSystemDynamicCache 缓存 buildWebSystemDynamicBase 的输出（30s TTL）。
// skills 列表/项目约定等低频变化，缓存避免每次 Loop 重建重复扫描文件系统，
// 从而让 system 动态后缀在同一配置下保持稳定（减少 KV 缓存前缀断裂点漂移）。
// ★ 项目环境段不在其中——它走「会话级冻结」projectEnvSectionCached（见其注释）。
var buildWebSystemDynamicCache struct {
	mu  sync.Mutex
	ts  time.Time
	val string
}

// buildWebSystemDynamicBase 构建 system prompt 动态后缀的**基础段**（CACHE_BOUNDARY 之后）：
// skills 列表 + 项目约定（ProjectRules）。★ 2026-09-13 拆分：项目环境段曾在此，
// 因 agent 会按系统提示主动改写 .pair/project.md，内容一变即令 boundary 之后
// （含**整个历史**）在 provider 前缀缓存中全部 miss → 已改为会话级冻结独立承载
// （projectEnvSectionCached）。
func buildWebSystemDynamicBase() string {
	buildWebSystemDynamicCache.mu.Lock()
	defer buildWebSystemDynamicCache.mu.Unlock()
	if buildWebSystemDynamicCache.val != "" && time.Since(buildWebSystemDynamicCache.ts) < 30*time.Second {
		return buildWebSystemDynamicCache.val
	}

	var b strings.Builder
	root := core.Root()
	skillsSec := skills.Prompt()
	rulesSec := agent.ProjectRules(root)
	// ★ 2026-08-27 缓存优化 / 2026-09-13 治理：「记忆/知识库」曾从 system 动态后缀
	//   移入「背景上下文快照」（高频变化会断 system 前缀）——该快照链已整体移除
	//   （2026-09-04 停用 → 2026-09-13 删除，见 loop.go）。二者当前**无注入点**；
	//   如需恢复，走本函数（system 动态后缀，建议配会话级冻结，见 projectEnvSectionCached），
	//   勿恢复消息流快照注入链。
	b.WriteString(skillsSec)
	b.WriteString(rulesSec)

	// ★ 2026-09-13（S1 缓存前缀治理）：项目环境段已移出本函数 →
	//   buildProjectEnvSection() 现读 + projectEnvSectionCached(convID) 会话级冻结。
	//   原因：agent 会按系统提示主动更新 .pair/project.md（见 loop.go 的环境问题说明），
	//   该项目环境段位于 boundary 之后，内容一变即让整个历史一起 miss。
	// 时间戳已移至用户消息内（Loop.Run 中注入），保持系统提示词缓存前缀稳定。
	val := b.String()
	buildWebSystemDynamicCache.ts = time.Now()
	buildWebSystemDynamicCache.val = val
	// ★ 缓存诊断（WB_CACHE_DIAG=1）：输出 dynamic 各段哈希，定位导致前缀变化的内容源。
	//   DeepSeek 按 system 消息整体前缀匹配（boundary 后动态段变化同样破坏缓存命中）。
	if os.Getenv("WB_CACHE_DIAG") == "1" {
		secHash := func(s string) string {
			sum := sha256.Sum256([]byte(s))
			return fmt.Sprintf("%x", sum[:4])
		}
		log.Printf("[cache-diag] dynamic 基础段 hash skills=%s(%d) rules=%s(%d) total=%s len=%d",
			secHash(skillsSec), len(skillsSec),
			secHash(rulesSec), len(rulesSec), secHash(val), len(val))
		log.Printf("[cache-diag] 记忆/知识库已移入背景快照；项目环境见 [cache-diag] 项目环境行（会话级冻结）")
		// 技能行级：定位列表变化的具体技能（行格式 "- 名字：描述"）
		for _, skillLine := range strings.Split(skillsSec, "\n") {
			if strings.HasPrefix(skillLine, "- ") {
				name := skillLine[2:]
				if i := strings.Index(name, "："); i > 0 {
					name = name[:i]
				}
				log.Printf("[cache-diag]   skill %q len=%d", name, len(skillLine))
			}
		}
	}
	return val
}

// buildProjectEnvSection 现读各工作区根的 .pair/project.md，拼成「# 项目环境」段。
// ★ 调用方应优先用 projectEnvSectionCached（会话级冻结），仅在需要真实最新内容时直调本函数。
func buildProjectEnvSection() string {
	if len(core.Folders) == 0 {
		return ""
	}
	var b strings.Builder
	// ★ 项目环境：遍历所有工作区根目录，分别读取各自的 .pair/project.md
	b.WriteString("\n\n# 项目环境")
	for i, f := range core.Folders {
		projName := filepath.Base(f)
		if i == 0 {
			b.WriteString(fmt.Sprintf("\n\n### %s（主项目）\n", projName))
		} else {
			b.WriteString(fmt.Sprintf("\n\n### %s\n", projName))
		}
		b.WriteString(fmt.Sprintf("> 路径: %s\n", f))
		projEnv := agent.ReadProjectEnv(f)
		if projEnv != "" {
			// 去除 project.md 自带的 # 项目环境档案 顶栏标题，避免与 ### 嵌套层级混乱
			lines := strings.SplitN(projEnv, "\n", 2)
			if len(lines) > 1 && strings.HasPrefix(strings.TrimSpace(lines[0]), "# ") {
				b.WriteString(strings.TrimSpace(lines[1]) + "\n")
			} else {
				b.WriteString(projEnv + "\n")
			}
		} else {
			b.WriteString("（无环境配置）\n")
		}
	}
	return b.String()
}

// projectEnvSessionCache 项目环境段的**会话级冻结**缓存（convID → 段文本）。
// ★ 2026-09-13（S1 缓存前缀治理）：项目环境位于 system 动态后缀（boundary 之后），
// 而 agent 会按系统提示主动更新 .pair/project.md（「若问题未记录，在解决后更新」）——
// 内容一变，provider 前缀缓存从该点起失效：boundary 之后含**整个历史**全部 miss。
// 改为「会话内冻结」：本会话首次构建时读一次，此后逐字节不变（新内容下个会话生效）。
// 代价可忽略：agent 需要最新环境配置时可直接 read .pair/project.md（系统提示已如此引导）。
var projectEnvSessionCache = struct {
	mu   sync.Mutex
	m    map[string]string
	keys []string // FIFO：超上限逐出最旧会话
}{m: map[string]string{}}

// projectEnvCacheMaxSessions 会话级冻结条数上限（防长跑进程内存累积）。
const projectEnvCacheMaxSessions = 256

// projectEnvSectionCached 取项目环境段（会话级冻结）。convID 为空（预热/无会话）时用固定键，
// 同样冻结，保证同源请求逐字节一致。
func projectEnvSectionCached(convID string) string {
	key := convID
	if key == "" {
		key = "\x00no-session"
	}
	projectEnvSessionCache.mu.Lock()
	if s, ok := projectEnvSessionCache.m[key]; ok {
		projectEnvSessionCache.mu.Unlock()
		return s
	}
	projectEnvSessionCache.mu.Unlock()
	s := buildProjectEnvSection() // 读盘在锁外（避免慢 IO 持锁）
	projectEnvSessionCache.mu.Lock()
	defer projectEnvSessionCache.mu.Unlock()
	if old, ok := projectEnvSessionCache.m[key]; ok { // 并发首建：以先写入者为准
		return old
	}
	projectEnvSessionCache.m[key] = s
	projectEnvSessionCache.keys = append(projectEnvSessionCache.keys, key)
	if len(projectEnvSessionCache.keys) > projectEnvCacheMaxSessions {
		oldest := projectEnvSessionCache.keys[0]
		projectEnvSessionCache.keys = projectEnvSessionCache.keys[1:]
		delete(projectEnvSessionCache.m, oldest)
	}
	if os.Getenv("WB_CACHE_DIAG") == "1" {
		sum := sha256.Sum256([]byte(s))
		log.Printf("[cache-diag] 项目环境段 会话冻结 conv=%s hash=%x len=%d（本会话内不再变化）",
			convID, sum[:4], len(s))
	}
	return s
}

// buildWebSystemPrompt 构建完整系统提示词（桌面和 web 端共享）。
// 使用唯一的 CACHE_BOUNDARY 分隔静态前缀与动态后缀，最大化 LLM KV Cache 命中率。
// ★ 通过 ComposeSystemPrompt 统一添加 boundary，避免双边界/漏边界。
// ★ boundary 的真实语义（2026-09-13 校正）：它只是**本地的静态/动态分界标记**
//
//	（用于复用静态前缀拼接 + 诊断定位变化源），**provider 不认** —— 动态后缀一变，
//	boundary 之后的内容（含**整个历史**）在 provider 前缀缓存中全部 miss，仅静态前缀仍命中。
//	（dsh 的 `systemPromptUpdate: 'in-history'` 才是「变化内容不伤历史」的正解，见知识库
//	关键点-dsh缓存前缀语义对照与注入点审计（2026-09-13）。）
//
// ★ 插件贡献的系统提示段/变量（对齐 system-prompt 注册中心）并入动态侧：
//
//	插件段随加载/卸载变化，放 boundary 后至少保住静态前缀。
//
// ★ 2026-08-31 按需激活：convID 指定时，on-demand 插件（agent-teams 等）段
//
//	仅在本会话已激活时注入；convID 为空（如桌面端）时按需段一律隐藏。
func buildWebSystemPrompt(convID string) string {
	// ★ 2026-09-13（S1）：项目环境段走会话级冻结（convID 维度），不与 30s TTL 基础段共用缓存。
	envSec := projectEnvSectionCached(convID)
	dynamic := buildWebSystemDynamicBase() + envSec
	if ph := handler.GetPluginHost(); ph != nil {
		if secs, err := agent.PluginPromptSections(ph, convID); err == nil && secs != "" {
			// ★ 缓存诊断：插件段哈希/长度（dynamic 变化源定位）
			if os.Getenv("WB_CACHE_DIAG") == "1" {
				sum := sha256.Sum256([]byte(secs))
				log.Printf("[cache-diag] 插件段 hash=%x len=%d", sum[:4], len(secs))
			}
			dynamic += "\n\n# 插件系统提示（由插件贡献，遵循各自段内规则）\n" + secs
		}
	}
	sys := agent.ComposeSystemPrompt(buildSystemStaticPrefix(), dynamic)
	// ★ 缓存诊断（WB_CACHE_DIAG=1）：**每次请求**输出完整 system 哈希 + 项目环境段哈希。
	//   用途：端到端比对同一会话相邻请求的 system 是否逐字节一致（含 project.md 被改写后）。
	//   开诊断时每请求一行（与 llm-trace 同量级），默认关闭零开销。
	if os.Getenv("WB_CACHE_DIAG") == "1" {
		sum := sha256.Sum256([]byte(sys))
		envSum := sha256.Sum256([]byte(envSec))
		log.Printf("[cache-diag] system hash=%x len=%d env=%x conv=%s", sum[:4], len(sys), envSum[:4], convID)
	}
	return sys
}

// buildWebProvider 构建 LLM Provider（桌面和 web 端共享）。
// ★ 2026-08-21：改用 ResolveProviderParams() 获取最终参数（含 agentloop 装配器覆盖：
//
//	模型级温度/思考/输出/上下文/多模态），不再直接读 core.Settings 业务字段。
//
// ★ 2026-09（t1 S1 闭环）：经 agent.CreateProvider 创建——服务商名在实现注册表
//
//	（插件 ctx.provider.register）有实现 → 用插件实现；否则回退 OpenAI 兼容。
func buildWebProvider() agent.Provider {
	return buildWebProviderForConv("", "")
}

// buildWebProviderForConv 构建会话级 LLM Provider（★ 2026-08-31 会话级模型路由）：
// 会话元数据里记录了 服务商/模型 时以其为准，否则等价于全局默认（buildWebProvider）。
func buildWebProviderForConv(convID, wsRoot string) agent.Provider {
	cur := agent.ResolveProviderParamsForConv(convID, wsRoot)
	if cur.APIKey == "" || cur.BaseURL == "" {
		return nil
	}
	if cur.MaxTokens > 0 && cur.MaxTokens < 8192 {
		log.Printf("[WARN] maxTokens=%d 过小（<8192），可能导致思考/回复被截断。建议在设置中调大至 ≥8192", cur.MaxTokens)
	}
	return agent.CreateProvider(cur)
}

// countStates 统计具备指定状态的执行状态数量。
func countStates(states []*agent.ExecutionState, status agent.ExecStatus) int {
	n := 0
	for _, s := range states {
		if s.Status == status {
			n++
		}
	}
	return n
}

// buildWebLoopOpts 构建 agent.LoopOpts（统一版本，平台差异通过 webCompressor 回调）。
func (s *webServer) buildWebLoopOpts(convID, message string, autonomous bool, wsRoot string) agent.LoopOpts {
	// ★ 2026-08-31 会话级模型：本会话选定的模型优先（未选沿用全局默认配置）
	prov := buildWebProviderForConv(convID, wsRoot)

	// ★ 2026-08-23 工作区隔离（重大 BUG）：会话根由请求指定，不再用全局当前根
	//   core.Root()——前端为其他工作区发起会话时，工具注册根必须等于会话根；
	//   运行中切换全局工作区也只影响新会话，不改变已注册工具的根。
	if wsRoot == "" {
		wsRoot = core.Root()
	}
	root := wsRoot
	agent.WorkspaceRoots = core.Folders
	if root != "" {
		agent.SkillProjectDir = filepath.Join(root, ".pair", "skills")
	}
	if sysDir := filepath.Join(core.ConfigDir(), "skills"); sysDir != "" {
		agent.SkillSystemDir = sysDir
	}
	agent.SkillEnabled = core.Settings.SkillEnabledOverrides
	agent.SkillStatusOverride = core.Settings.SkillStatusOverrides
	// 初始化 MCP 配置路径（供 buildWebLoopOpts 内 mcppanel.LoadConfigs 使用）
	agent.MCPUserConfigPath = filepath.Join(core.ConfigDir(), "mcp.json")
	if root != "" {
		agent.MCPProjectConfigPath = filepath.Join(root, ".pair", "mcp.json")
	}
	reg := agent.NewRegistry()
	agent.RegisterHostFrameworkTools(reg, root)
	// ★ 插件系统：全局 PluginHost 的 cordis_* 工具（浏览器插件面板同源）
	if ph := handler.GetPluginHost(); ph != nil {
		agent.RegisterCordisTools(reg, ph, root)
		// ★ 工具集管理工具（toolset_build/list/export/import…，与插件宿主同源）
		agent.RegisterToolsetTools(reg, root, ph)
	}
	if cfgs := mcppanel.LoadConfigs(); len(cfgs) > 0 {
		agentCfgs := make([]agent.MCPServerConfig, len(cfgs))
		for i, c := range cfgs {
			agentCfgs[i] = agent.MCPServerConfig{Name: c.Name, Command: c.Command, Args: c.Args, Env: c.Env}
		}
		agent.RegisterMCPServers(reg, agentCfgs)
	}

	// ★ harness 对齐（默认关闭——全量工具集；WB_HARNESS=1 开启时精简 pair 独有工具）；
	//   插件注册的工具豁免（插件是内容，非 pair 独有编码能力）。
	//   ★ 2026-08-19 修复：pluginHost 直接获取，不依赖过滤回调（全量模式下
	//     ApplyHarnessToolFilter 提前 return 不触发回调 → pluginHost 恒 nil →
	//     MergePluginTools 跳过 → 插件工具不进会话 reg（agent 只剩管理工具）。
	pluginHost := handler.GetPluginHost()
	agent.ApplyHarnessToolFilter(reg, func(name string) bool {
		if pluginHost != nil {
			return pluginHost.IsPluginTool(name)
		}
		return false
	})
	// ★ 应用工作区工具集「内置工具包」状态（builtin.json 已加入的分组 → 组内工具启用）。
	//   会话级 reg 独立实例，需在过滤后显式恢复内置组启用状态（与启动 LoadAllToolsets 对齐）。
	agent.ApplyToolsetBuiltinState(reg, root)
	// ★ 合并插件注册的业务工具（Node 桥工具 + goja 插件 ctx.tools.register）：
	//   reg 是本次会话新建，需把插件宿主中插件注册的工具同步进来，
	//   agent 才能直接 function-call 调用（如 Node 桥插件的 hello_bridge）。
	//   ★ 2026-08-31 按需激活：按会话过滤（on-demand 插件未激活时工具隐藏）。
	if pluginHost != nil {
		agent.MergePluginToolsForConv(reg, pluginHost, convID)
	}
	agent.SetCodeGraphDB(agentMgr.RawDB())
	// ★ 与 SetCodeGraphDB 成对：buildWebLoopOpts 每次构建会话注册表时同步
	//   主项目根（共享 DB 归属判定），与 startWebUI 启动路径保持一致。
	agent.SetCodeGraphRoot(core.Root())
	agent.InitDebugLogger(root, 50)

	sys := buildWebSystemPrompt(convID)

	// ★ 注入工具使用指南（引导 LLM 优先使用专用工具而非 bash）
	if guide := reg.UsageGuideText(); guide != "" {
		// ★ 缓存诊断：UsageGuide 哈希（dynamic 变化源定位——工具状态/顺序漂移）
		if os.Getenv("WB_CACHE_DIAG") == "1" {
			sum := sha256.Sum256([]byte(guide))
			log.Printf("[cache-diag] usageGuide hash=%x len=%d", sum[:4], len(guide))
		}
		sys += "\n\n" + guide
	}

	// ★ 保存注册表引用，供 /api/tools 查询工具列表与状态
	handler.SetToolsRegistry(reg)

	var history []agent.Message
	var summaries []string

	var sumErr error
	if convID != "" {
		// ★ 2026-08-23 工作区隔离：会话历史加载按会话根路由 store（此前全局
		//   Store() 在切换工作区后可能读到其他工作区的存储）。
		if store := agentMgr.StoreFor(root); store != nil {
			raw, loadErr := store.LoadAll(convID)
			if loadErr != nil {
				log.Printf("[chat] LoadAll 失败 conv=%s err=%v", convID, loadErr)
			}
			if raw != nil {
				history = make([]agent.Message, len(raw))
				for i := range raw {
					history[i] = raw[i]
					// ★ 保留 Reasoning（思考链）不丢弃！DeepSeek 文档明确要求：
					//   进行工具调用的轮次，reasoning_content 在后续所有请求中必须回传。
					//   无工具调用的轮次，API 也会自动忽略 reasoning_content，保留无害。
					//   之前清空会导致 Agent 失去历史思考连贯性（变笨），
					//   且有工具调用的轮次缺失 reasoning_content 可能引发 API 400 错误。
				}
			}
			// 加载已持久化的压缩摘要（页面刷新后恢复上下文连续性）
			summaries, sumErr = store.LoadCompressedSummaries(convID)
			if sumErr != nil {
				log.Printf("[chat] LoadCompressedSummaries 失败 conv=%s err=%v", convID, sumErr)
			}
		}
	}
	// ★ 中断/未完成的 tool_call 不再注入「结果未知」提示（历史教训：TOOL_OUTCOME_UNKNOWN
	// 长文本会干扰模型判断、诱导核对/重试）。API 配对契约由 sanitizeToolPairing 兜底：
	// 缺结果的 tool_call 自动补空串占位（格式完整、内容零干扰）。

	// ★ 保存原始（未压缩）历史，供持久化使用——压缩版只给 LLM 上下文，不写回历史记录
	originalHistory := make([]agent.Message, len(history))
	copy(originalHistory, history)

	// ★★★ 会话连贯性上下文：ResumeContext / 背景上下文快照均已移除（2026-09-13）★★★
	// 历史：任务进度/对话摘要/记忆/项目归属经 BuildResumeContext → 背景上下文快照注入
	// （2026-08-23 工作区隔离 / 2026-09-03 KV 修复迁快照尾）。
	// 2026-09-04 停用（快照正文每轮必变 → 每轮追加新快照，实测累积 104 条 / 133 万字符 /
	// 占历史 20%+，上下文膨胀且稀释前缀缓存命中率），2026-09-13 实现整体删除。
	// 现状：会话连贯性上下文由**提交消息（handoff 交接视图）**承载（尾部追加语义，
	// 见 handoff.go）；任务进度由任务清单工具维护。★ 勿恢复快照注入链。

	// ★ 历史精简：跨轮次加载时只保留最近一轮完整交互细节，
	//   旧轮次压缩为 [用户消息, 助手最终报告]，丢弃中间 tool 输出。
	//   大幅减少上下文体积，同时保持语义连续性。
	// ★ 2026-08-17 对齐 compaction-basic：改为 token 压力触发——
	//   原实现按轮数（>2 轮历史）强制压缩，小对话也被改写历史前缀，
	//   导致 KV 缓存前缀每轮断裂、命中率骤降；现估算 token 占比，
	//   未达阈值（45% 窗口）保持原始历史逐字节不变（缓存可连续命中）。
	// ★ 2026-09-11 会话交接（handoff.go）：先做一次「判断 + 整理」——历史达阈值
	//   （窗口 30%/地板 24K token 或 ≥100 条）时，用一次 LLM 整理成「会话交接·提交消息」，
	//   替代全量历史注入（防上下文膨胀；复用/刷新见 handoff.go）；未达阈值/关闭时
	//   保持原逻辑（按 token 压力精简，未达压力阈值即逐字节原样，缓存连续命中）。
	handoffApplied := false
	// ★ 2026-09-19 上下文窗口以服务商配置（models.json）为准：装配器按「模型级 > 服务商级」
	//   取值，settings 顶层值仅在服务商/模型都未配置时兜底（此前这里直读 settings → 服务商配置不生效）。
	convParams := agent.ResolveProviderParamsForConv(convID, root)
	ctxWindow := agent.ContextWindow(convParams)
	if store := agentMgr.StoreFor(root); store != nil {
		// ★ 判官实例（B/C 语义复检）：独立轻量实例——non-thinking + 极小输出
		//   （只输出一个词），与主对话通道隔离；构建为纯参数装配（无网络），每轮现建。
		judge := agent.HandoffJudgeProvider(convParams)
		// ★ 2026-09-12 策略外置：整理由 agentloop 插件（registerHandoff.onUserTurn）实现，
		//   宿主只提供调用位置（会话装配前——插件无从自主介入）与能力（provider/judge/
		//   store/口径工具）；未注册或执行失败 → 回退 Go 默认实现（语义不变）。
		view, ok, hnotice := agent.HandoffUserTurnView(context.Background(), prov, judge, store,
			convID, root, history, message, ctxWindow)
		if ok {
			log.Printf("[handoff] conv=%s 已启用交接视图（历史 %d 条 → %d 条）", convID, len(history), len(view))
			history = view
			handoffApplied = true
			if hnotice == "" {
				hnotice = "已把此前对话整理为「会话交接·提交消息」（完整历史仍保存在会话记录中），后续基于交接要点继续"
			}
			agentMgr.PushNotice(convID, hnotice)
		}
	}
	if !handoffApplied {
		history = agent.CondenseHistoryByPressure(history, ctxWindow)
	}

	// ★ 2026-09-12：「最大迭代数」配置已移除（原此处读 core.Settings.MaxIterations，
	//   自主模式再 ×2 放大）——段内迭代安全上限改由 agent 侧按工具预算派生
	//   （internal/agent/tool_budget.go IterationLimit），段的收束由工具预算 +
	//   自动续跑（受 maxToolBudgetSegments 约束）负责，宿主不再传迭代数。

	return agent.LoopOpts{
		Provider: prov,
		Registry: reg,
		System:   sys,
		// ★ 2026-09-12 分段续跑配置化：单段工具调用预算 / 续跑段数上限由
		//   agentloop 插件注册配置（pluginSettings.agentloop）经装配器透传
		//   （见 .pair/plugins/agentloop/index.js 的 ctx.loopFactory.register）；
		//   宿主不设默认值——0 = agent 侧默认（120 次 / 20 段），
		//   负数预算 = 不限（归一化见 agent/tool_budget.go）。
		MaxContextTokens:    ctxWindow,
		Compressor:          webCompressor(),
		History:             history,         // 压缩版：供 LLM 上下文使用
		HistoryOriginal:     originalHistory, // 原始版：供持久化使用，防止压缩版写回历史记录
		CompressedSummaries: summaries,
		Autonomous:          autonomous,
		ConvID:              convID,    // 仅诊断用：缓存前缀诊断区分会话
	}
}

// handleChatSend 启动一次 agent 会话（非阻塞）。
func (s *webServer) handleChatSend(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonErr(w, "仅 POST")
		return
	}
	var req struct {
		Message       string            `json:"message"`
		SessionID     string            `json:"sessionId"`
		Autonomous    bool              `json:"autonomous"`
		ConvID        string            `json:"convId"`
		WorkspaceRoot string            `json:"workspaceRoot"`
		Images        []agent.ImagePart `json:"images,omitempty"` // ★ 2026-08-21 多模态：前端结构化发送的图片（base64 data URL / http(s) URL）
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, err.Error())
		return
	}
	if req.Message == "" {
		jsonErr(w, "消息不能为空")
		return
	}
	const maxMsgLen = 50000
	if len(req.Message) > maxMsgLen {
		req.Message = req.Message[:maxMsgLen] + "\n\n…（消息过长，已截断至 " + fmt.Sprint(maxMsgLen) + " 字符）"
	}
	if req.ConvID == "" {
		req.ConvID = fmt.Sprintf("conv_%d", time.Now().UnixNano())
	}
	if req.WorkspaceRoot == "" {
		req.WorkspaceRoot = core.Root()
	}

	// ★ 收到消息入口日志（排查「无响应」：确认后端确实收到并进入处理）
	log.Printf("[chat] 收到发送请求 conv=%s len=%d autonomous=%v ws=%s",
		req.ConvID, len(req.Message), req.Autonomous, req.WorkspaceRoot)

	// ★ 2026-09-04 修复：同步校验用会话感知判定（ConfiguredProviderForConv）。
	//   此前 ConfiguredProvider() 只看全局装配：会话已选 服务商/模型/配置 时，
	//   若全局激活配置未展开完整（如配置未指定模型），会误报「未配置 API key」。
	//   （用户实测：会话切换模型成功 + 装配日志 preset 有值 → 仍被同步校验拦截）
	if ok, missing := agent.ConfiguredProviderForConv(req.ConvID, req.WorkspaceRoot); !ok {
		hint := "未配置 API key。请在设置面板中配置 API Key 和模型。"
		switch missing {
		case "模型为空":
			hint = "模型未配置：当前配置未指定模型。请在对话面板选择模型，或在「设置 → AI」中为配置指定模型后再发送。"
		case "API Key 为空":
			hint = "API Key 未配置：请在「设置 → AI」中为当前配置填写 API Key。"
		case "API 地址为空":
			hint = "API 地址未配置：请在「设置 → AI」中为当前配置填写 API 地址。"
		}
		jsonErr(w, hint)
		return
	}

	if len(req.Images) > 0 {
		// ★ 2026-08-21 多模态：带图片的用户消息落盘（Images 随消息持久化，回放/历史加载保留）
		// ★ 2026-08-23 工作区隔离：按请求工作区根路由 store（切换后不在新工作区落盘）。
		if err := agentMgr.AppendPersistedUserMessageToWithImages(req.WorkspaceRoot, req.ConvID, req.Message, req.Images); err != nil {
			log.Printf("[chat] AppendPersistedUserMessageWithImages 失败 conv=%s err=%v", req.ConvID, err)
			jsonErr(w, "写入用户消息失败: "+err.Error())
			return
		}
	} else {
		if err := agentMgr.AppendPersistedUserMessageTo(req.WorkspaceRoot, req.ConvID, req.Message); err != nil {
			log.Printf("[chat] AppendPersistedUserMessage 失败 conv=%s err=%v", req.ConvID, err)
			jsonErr(w, "写入用户消息失败: "+err.Error())
			return
		}
	}
	// ★ 2026-09-10 slash 命令聊天入口兜底（双保险）：前端 '/' 菜单未触发（清单未加载 /
	//   菜单失配 / 直接输入带参文本）时，后端在此识别 "/命令" 消息：命中 on-demand 命令
	//   且本会话未激活 → 执行命令 + 激活插件 + 结果以系统消息注入（与 /api/commands/run
	//   语义一致，handler 收 args.args 子命令串）；已激活（前端菜单路径已执行过）→ 跳过
	//   防双执行双注入；未命中命令名 → 不拦截，原样照常发送（零破坏）。
	if strings.HasPrefix(req.Message, "/") {
		trimmed := strings.TrimSpace(req.Message)
		name, sub := "", ""
		if i := strings.IndexByte(trimmed, ' '); i >= 0 {
			name = trimmed[1:i]
			sub = strings.TrimSpace(trimmed[i+1:])
		} else {
			name = trimmed[1:]
		}
		if plugin := agent.OnDemandCommandMapping()[name]; plugin != "" && !agent.IsPluginActiveInConv(req.ConvID, plugin) {
			if output, err := agent.RunHostCommand(name, map[string]any{"args": sub}); err == nil {
				activated := agent.ActivateByCommand(req.ConvID, name)
				injected := output
				if activated != "" {
					log.Printf("[activation] 会话 %s 经 /%s 激活按需插件 %s（聊天入口兜底）", req.ConvID, name, activated)
					injected += "\n\n（插件 " + activated + " 已激活——其团队工具现已并入本会话，按系统提示中的 AgentTeams 协议开始执行）"
				}
				msg := agent.Message{Role: agent.RoleSystem, Content: "（命令 /" + name + " 执行结果）\n" + injected}
				// ★ 与用户消息落盘同路由（StoreFor 显式指定工作区）：AppendPersistedMessage
				//   走默认 store（会话未 Start 时可能是空/错路由），此处显式保证注入落盘。
				if store := agentMgr.StoreFor(req.WorkspaceRoot); store != nil {
					if err := store.AppendMessage(req.ConvID, msg, nil); err != nil {
						log.Printf("[commands] 聊天入口命令结果注入失败 conv=%s: %v", req.ConvID, err)
					}
				}
			} else {
				log.Printf("[commands] 聊天入口命令 /%s 执行失败（原样发送）: %v", name, err)
			}
		}
	}
	// ★ 2026-08-22 异步 Start：重活（opts 构建/插件合并/会话上下文注入/会话装配/历史加载）
	//   可能在 10s~76s 波动（实测日志），原同步等待会让前端 30s 超时显示「请求超时」。
	//   现改为 HTTP 立即返回「马上的消息」，Start 在后台 goroutine 执行；
	//   失败经 PushStartError → WS error 事件推送（前端按 convID 路由、清理 loading）。
	//   快速校验（消息空/未配置 Provider/用户消息落盘失败）保持在同步段，失败语义不变。
	// ★ 2026-09-08 启动链路抽为 launchConvRun：slash 命令激活（/agent-teams 等）需要
	//   同一套 opts 构建 + 审核解析 + Start，抽公共实现避免两处漂移。
	s.launchConvRun(req.ConvID, req.WorkspaceRoot, req.Message, req.Autonomous)
	jsonResp(w, map[string]any{"ok": true, "convId": req.ConvID})
}

// launchConvRun 后台启动一次 agent 会话运行（handleChatSend 与 slash 命令激活唤醒共用）：
// 构建 LoopOpts（工作区隔离 + 工具集白名单 + 多模态门控）→ 审核/规划 Provider 解析 →
// SessionManager.Start；失败经 PushStartError 推 WS error 事件（前端据此清理 loading）。
func (s *webServer) launchConvRun(convID, wsRoot, task string, autonomous bool) {
	go func() {
		// ★ 2026-08-23 工作区隔离：会话根（请求指定）贯穿 buildWebLoopOpts。
		opts := s.buildWebLoopOpts(convID, task, autonomous, wsRoot)
		// ★ 2026-08-21：Provider 判空——清空配置后 APIKey/BaseURL 为空 → buildWebProvider
		//   返回 nil → Loop.Provider=nil → agentloop 插件首次 loop.llm.chat 触发
		//   nil pointer panic（[session] Loop goroutine panic）。此处前置拦截给友好提示。
		if opts.Provider == nil {
			agentMgr.PushStartError(convID, "未配置 AI 服务商（APIKey/BaseURL 为空）：请先在「设置 → AI」中添加并应用 AI 配置，再发送消息")
			return
		}
		opts.WorkspaceRoot = wsRoot
		// ★ 2026-09-04 工具集模式改造：工具集已全局化（通用集合），agent 工具面
		//   按「会话选择的集合」收敛（ConversationMeta.Toolset；空=default 集合）
		//   ——不再是「工作区工具集并集」。precise 收敛：仅所选集合声明的工具可见。
		//   协议/管理工具（SystemTool + cordis_*/toolset_*）恒可用（函数内兜底）。
		if opts.Registry != nil {
			agent.ApplyConvToolsetWhitelist(handler.GetPluginHost(), opts.Registry, convID, wsRoot)
			// ★ 2026-09-09 多模态门控：非视觉模型禁用截图/看图工具（白名单之后执行，
			//   覆盖 SystemTool 恒可用项；多模态模型恢复启用防残留）
			agent.ApplyMultimodalToolGate(opts.Registry, opts.Provider)
		}

		// 先从全局设置取审核配置（默认值）
		opts.ReviewMode = core.Settings.ReviewMode
		opts.ReviewBlacklist = core.Settings.ReviewBlacklist
		opts.ReviewWhitelist = core.Settings.ReviewWhitelist
		// ★ 如果请求中指定了工作区根路径，从工作区配置覆盖审核配置
		if wsRoot != "" {
			wrMode, wrBlack, wrWhite := agent.LoadWorkspaceReviewConfig(wsRoot)
			if wrMode != "" && wrMode != "auto" {
				opts.ReviewMode = wrMode
			}
			if wrBlack != nil {
				opts.ReviewBlacklist = wrBlack
			}
			if wrWhite != nil {
				opts.ReviewWhitelist = wrWhite
			}
		}
		// ★ 2026-08-31 会话级审核模式最高优先：会话元数据记录的选择（持久化，
		//   重启/恢复会话仍生效）> 工作区配置 > 全局默认。
		if store := agentMgr.StoreFor(wsRoot); store != nil {
			if cm := store.ConvReviewMode(convID); cm != "" {
				opts.ReviewMode = cm
			}
		}
		// ★ 配置消费插件化：Review/Plan Provider 参数统一经装配点解析。
		// ★ 2026-08-31：按会话解析（审核/规划模型跟随本会话选定的模型）。
		cur := agent.ResolveProviderParamsForConv(convID, wsRoot)
		if opts.ReviewMode == "auto" && cur.ReviewModel != "" {
			pm := strings.TrimSpace(cur.PlanModel)
			if pm != "" && cur.BaseURL != "" && cur.APIKey != "" {
				// ★ t1 S1：实现级插件槽位（插件注册的 Provider 实现对新协议生效）
				rp := cur
				rp.Model = pm
				rp.Temperature = -1
				rp.ThinkingMode = "non-thinking"
				rp.MaxTokens = 0
				rp.Multimodal = false
				opts.ReviewProvider = agent.CreateProvider(rp)
			}
		}

		// ★ 2026-09-21 自主模式插件化：原 autonomous 分支创建的 PlanProvider（双层 Loop
		//   的规划模型）已随该架构删除——自主模式的「监督者」由插件决策器 + ctx.subagent
		//   能力派生，复用本会话 Provider（「跟随执行模型」）。监督轮数上限由插件经
		//   ctx.loopFactory.register 覆盖 maxSuperviseRounds（缺省内核 20）。

		// 使用分离的 context：setupCtx 用于 Start 方法本身的超时（避免在获取锁或建表时永久阻塞），
		// Loop 的运行由内部独立的 context 管理（Stop 可取消），不受此超时影响。
		setupCtx, setupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer setupCancel()
		if err := agentMgr.Start(setupCtx, convID, task, opts); err != nil {
			// ★ Start 失败日志（排查「无响应」：异步后经 WS error 事件推送，此处留档）
			log.Printf("[chat] Start 失败 conv=%s err=%v", convID, err)
			agentMgr.PushStartError(convID, err.Error())
			return
		}
		log.Printf("[chat] Start 成功 conv=%s（agent 循环已启动）", convID)
	}()
}

// handleChatStop 停止指定会话。
func (s *webServer) handleChatStop(w http.ResponseWriter, r *http.Request) {
	convID := r.URL.Query().Get("convId")
	if convID == "" {
		convID = r.URL.Query().Get("sessionId")
	}
	if convID == "" {
		jsonErr(w, "缺少 convId 参数")
		return
	}
	agentMgr.Stop(convID)
	jsonResp(w, map[string]any{"ok": true})
}

func (s *webServer) handleChatCompact(w http.ResponseWriter, r *http.Request) {
	convID := r.URL.Query().Get("convId")
	if convID == "" {
		jsonErr(w, "缺少 convId 参数")
		return
	}
	agentMgr.Compact(convID)
	jsonResp(w, map[string]any{"ok": true})
}

// handleChatAnswer 发送 ask_user 的用户回答。
// ★ Round3 ⑤：支持多问题 answers 数组 [{id, answer}]（与旧单问题 {answer} 双兼容）。
func (s *webServer) handleChatAnswer(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonErr(w, "仅 POST")
		return
	}
	var req struct {
		ConvID  string `json:"convId"`
		CallID  string `json:"callId"`
		Answer  string `json:"answer"`
		Answers []struct {
			ID     string `json:"id"`
			Answer string `json:"answer"`
		} `json:"answers"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, err.Error())
		return
	}
	if req.ConvID == "" {
		jsonErr(w, "convId 必填")
		return
	}
	if len(req.Answers) > 0 {
		answers := make([]agent.AskAnswer, 0, len(req.Answers))
		for _, a := range req.Answers {
			answers = append(answers, agent.AskAnswer{ID: a.ID, Answer: a.Answer})
		}
		if err := agentMgr.SendAnswers(req.ConvID, answers); err != nil {
			jsonErr(w, err.Error())
			return
		}
	} else {
		if err := agentMgr.SendAnswer(req.ConvID, req.Answer); err != nil {
			jsonErr(w, err.Error())
			return
		}
	}
	jsonResp(w, map[string]any{"ok": true})
}

// handleCommands GET /api/commands：slash 命令清单（前端 "/" 菜单提示）。
func (s *webServer) handleCommands(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		jsonErr(w, "仅 GET")
		return
	}
	jsonResp(w, map[string]any{"ok": true, "commands": agent.ListHostCommands()})
}

// handleCommandsRun POST /api/commands/run：执行 slash 命令；convID 提供时
// 结果以系统消息注入该会话（前端发送原样文本后 agent 可见命令输出）。
// ★ 2026-08-31 按需激活：命令命中 on-demand 插件（agent-teams 等）时，先激活
//
//	本会话再注入协议说明——agent 当轮即获得完整用法，且后续轮次工具/提示均可用。
func (s *webServer) handleCommandsRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonErr(w, "仅 POST")
		return
	}
	var req struct {
		Name          string         `json:"name"`
		Args          map[string]any `json:"args"`
		ConvID        string         `json:"convId"`
		WorkspaceRoot string         `json:"workspaceRoot"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, err.Error())
		return
	}
	if req.Name == "" {
		jsonErr(w, "name 必填")
		return
	}
	output, err := agent.RunHostCommand(req.Name, req.Args)
	if err != nil {
		jsonErr(w, err.Error())
		return
	}
	// ★ 按需激活：命令触发插件 → 本会话激活，工具立即可用。协议段已常驻
	//   （方案 B：alwaysVisible 段 = 引导+协议），此处仅提示解锁，不再重复注入全文。
	activated := agent.ActivateByCommand(req.ConvID, req.Name)
	if activated != "" {
		log.Printf("[activation] 会话 %s 经 /%s 激活按需插件 %s", req.ConvID, req.Name, activated)
		output += "\n\n（插件 " + activated + " 已激活——其团队工具现已并入本会话，按系统提示中的 AgentTeams 协议开始执行）"
	}
	// 结果注入对话（系统消息；持久化，刷新/续聊可见）
	if req.ConvID != "" {
		msg := agent.Message{Role: agent.RoleSystem, Content: "（命令 /" + req.Name + " 执行结果）\n" + output}
		if err := agentMgr.AppendPersistedMessage(req.ConvID, msg, nil); err != nil {
			log.Printf("[commands] 结果注入失败 conv=%s: %v", req.ConvID, err)
		}
	}
	// ★ 2026-09-08 激活后自动唤醒 agent（对齐 dsh src/command.ts 的 invocation.agent.followup）：
	//   此前命令只注入系统消息、不启动 agent —— 用户执行 /agent-teams 后界面毫无动作，
	//   被误判为「插件呼不出/没生效」。现在：激活成功且非纯查询子命令（status）→
	//   以命令原文 `/name <args>` 为 task 启动一次会话运行，队长当轮即按协议建队
	//   （launchConvRun 会重建工具面，agent-teams 工具同步可见）。
	// ★ 2026-09-11：已激活的按需命令重复执行也唤醒——命令是「会话动作入口」：
	//   /创造 <需求> 每次都要生成新指令、/agent-teams <目标> 每次都是新目标；
	//   仅首次激活唤醒会让二次执行「只有结果卡片、agent 不动」。
	if req.ConvID != "" {
		_, isOnDemandCmd := agent.OnDemandCommandMapping()[req.Name]
		if task := agentCommandTaskText(req.Name, req.Args); task != "" && (activated != "" || isOnDemandCmd) {
			wsRoot := req.WorkspaceRoot
			if wsRoot == "" {
				wsRoot = core.Root()
			}
			log.Printf("[activation] 会话 %s /%s（activated=%q）→ 自动唤醒 agent（task=%s）",
				req.ConvID, req.Name, activated, trimForLog(task, 80))
			s.launchConvRun(req.ConvID, wsRoot, task, false)
		}
	}
	jsonResp(w, map[string]any{"ok": true, "name": req.Name, "output": output})
}

// agentCommandTaskText 按需插件命令激活后传给 agent 的首轮 task 文本：
// `/name <args>` 原文（args 取 map 的 "args" 子命令串，如 `/agent-teams 实现 X`）；
// 纯查询子命令（status）返回 ""——不唤醒 agent，只回状态快照。
func agentCommandTaskText(name string, args map[string]any) string {
	sub := ""
	if args != nil {
		if v, ok := args["args"]; ok && v != nil {
			sub = strings.TrimSpace(fmt.Sprint(v))
		}
	}
	switch strings.ToLower(sub) {
	case "status":
		return ""
	}
	return strings.TrimSpace("/" + name + " " + sub)
}

// trimForLog 日志用截断（按 rune 计，避免切断多字节字符）。
func trimForLog(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

// handleChatApprove 发送审批结果。
func (s *webServer) handleChatApprove(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonErr(w, "仅 POST")
		return
	}
	var req struct {
		ConvID   string `json:"convId"`
		Approved bool   `json:"approved"`
		Reply    string `json:"reply"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, err.Error())
		return
	}
	if req.ConvID == "" {
		jsonErr(w, "convId 必填")
		return
	}
	if err := agentMgr.Approve(req.ConvID, req.Approved, req.Reply); err != nil {
		jsonErr(w, err.Error())
		return
	}
	jsonResp(w, map[string]any{"ok": true})
}

// handleChatFeedback 发送运行时反馈。
func (s *webServer) handleChatFeedback(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonErr(w, "仅 POST")
		return
	}
	var req struct {
		ConvID   string `json:"convId"`
		Feedback string `json:"feedback"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, err.Error())
		return
	}
	if req.ConvID == "" {
		jsonErr(w, "convId 必填")
		return
	}
	if err := agentMgr.SendFeedback(req.ConvID, req.Feedback); err != nil {
		jsonErr(w, err.Error())
		return
	}
	jsonResp(w, map[string]any{"ok": true})
}

// startEventPersistWorker 设置 OnDone 回调（compressor 由 webCompressor 回调提供）。
func (s *webServer) startEventPersistWorker() {
	agentMgr.OnDone = func(convID string) {
		go generateConversationSummary(convID, webCompressor())
	}
}

// ── 对话摘要生成 ──────────────────────────────────────────

// generateConversationSummary 用规则式生成对话摘要并保存到 MessageStore。
// 在对话完全结束后调用（goroutine 内异步）。prov 保留兼容但当前未使用（规则式摘要）。
func generateConversationSummary(convID string, prov agent.Provider) {
	if convID == "" {
		return
	}
	_ = prov

	store := agentMgr.Store()
	if store == nil {
		return
	}

	meta, err := store.GetConversation(convID)
	if err != nil || meta == nil {
		return
	}

	// 已有摘要 → 跳过（避免重复生成）
	if meta.Summary != "" {
		return
	}

	msgs, err := store.LoadAll(convID)
	if err != nil || len(msgs) == 0 {
		return
	}

	// 使用 pkg/summary 生成摘要
	summaryMsgs := make([]summary.Message, len(msgs))
	for i, m := range msgs {
		summaryMsgs[i] = summary.Message{Role: string(m.Role), Content: m.Content}
	}
	s := summary.Generate(summary.ConvInfo{
		ID: meta.ID, Title: meta.Title,
		CreatedAt: meta.CreatedAt, UpdatedAt: meta.UpdatedAt,
		Messages: summaryMsgs,
	})
	if s == "" {
		return
	}
	summaryAt := time.Now().Format("2006-01-02 15:04:05")
	store.SetSummary(convID, s, summaryAt)

	// 同步写入记忆索引
	assistantMsgs := make([]string, 0)
	allMsgs := make([]string, len(msgs))
	for i, m := range msgs {
		allMsgs[i] = m.Content
		if m.Role == agent.RoleAssistant {
			assistantMsgs = append(assistantMsgs, m.Content)
		}
	}
	memory.Upsert(memory.Entry{
		ID:           meta.ID,
		Title:        meta.Title,
		Summary:      s,
		CreatedAt:    meta.CreatedAt,
		UpdatedAt:    meta.UpdatedAt,
		MessageCount: len(msgs),
		Tags:         memory.ExtractTags(allMsgs),
		KeyPoints:    memory.ExtractKeyPoints(assistantMsgs),
		CompletedAt:  summaryAt,
	})
}

// ─── 记忆索引 API（委托 pkg/memory）─────────────────────────

func (s *webServer) handleMemorySearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	results := memory.Search(q)
	jsonResp(w, map[string]any{"results": results})
}

func (s *webServer) handleMemoryList(w http.ResponseWriter, r *http.Request) {
	memories := memory.List()
	jsonResp(w, map[string]any{"memories": memories})
}

func (s *webServer) handleMemoryRebuild(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonErr(w, "仅 POST")
		return
	}
	store := agentMgr.Store()
	if store == nil {
		jsonErr(w, "消息存储未初始化")
		return
	}
	metas, err := store.ListConversations(core.Root())
	if err != nil {
		jsonErr(w, err.Error())
		return
	}
	count := 0
	for _, meta := range metas {
		msgs, err := store.LoadAll(meta.ID)
		if err != nil || len(msgs) < 2 {
			continue
		}
		summaryStr := meta.Summary
		if summaryStr == "" {
			summaryMsgs := make([]summary.Message, len(msgs))
			for j, m := range msgs {
				summaryMsgs[j] = summary.Message{Role: string(m.Role), Content: m.Content}
			}
			summaryStr = summary.Generate(summary.ConvInfo{ID: meta.ID, Title: meta.Title, Messages: summaryMsgs})
		}
		assistantMsgs := make([]string, 0)
		allMsgs := make([]string, len(msgs))
		for j, m := range msgs {
			allMsgs[j] = m.Content
			if m.Role == agent.RoleAssistant {
				assistantMsgs = append(assistantMsgs, m.Content)
			}
		}
		memory.Upsert(memory.Entry{
			ID: meta.ID, Title: meta.Title, Summary: summaryStr,
			CreatedAt: meta.CreatedAt, UpdatedAt: meta.UpdatedAt,
			MessageCount: len(msgs),
			Tags:         memory.ExtractTags(allMsgs),
			KeyPoints:    memory.ExtractKeyPoints(assistantMsgs),
			CompletedAt:  meta.SummaryAt,
		})
		count++
	}
	jsonResp(w, map[string]any{"status": "ok", "count": count})
}

// ─── Debug 日志 API ──────────────────────────────────────────

func (s *webServer) handleDebugLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		jsonErr(w, "仅 GET")
		return
	}
	if agent.GlobalDebugLogger == nil {
		jsonResp(w, map[string]any{"logs": []any{}, "counts": map[string]int{}})
		return
	}

	level := r.URL.Query().Get("level")
	source := r.URL.Query().Get("source")
	limitStr := r.URL.Query().Get("limit")
	limit := 0
	if limitStr != "" {
		fmt.Sscanf(limitStr, "%d", &limit)
	}

	var logs []*agent.LogEntry
	if level != "" || source != "" {
		logs = agent.GlobalDebugLogger.FilterLogs(agent.LogLevel(level), source, limit)
	} else {
		logs = agent.GlobalDebugLogger.ListLogs(limit)
	}

	counts := agent.GlobalDebugLogger.CountLogs()

	jsonResp(w, map[string]any{
		"logs":   logs,
		"counts": counts,
	})
}

func (s *webServer) handleDebugLogByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		jsonErr(w, "仅 GET")
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/debug/logs/")
	if id == "" {
		jsonErr(w, "id 必填")
		return
	}
	if agent.GlobalDebugLogger == nil {
		jsonErr(w, "调试日志未初始化")
		return
	}
	entry := agent.GlobalDebugLogger.GetLog(id)
	if entry == nil {
		jsonErr(w, "日志不存在: "+id)
		return
	}
	jsonResp(w, entry)
}
