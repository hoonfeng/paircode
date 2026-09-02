# gou-ide「Prompt 发送链路」缓存机制分析

> 分析日期：2026-08-17
> 分析范围：`internal/agent`（缓存形状/统计/组装）+ `cmd/companion/web_server.go`（system prompt 组装入口）+ `session_context.go`（会话上下文 TTL 缓存）
> 适用模型：DeepSeek 系（`prompt_cache_hit_tokens` / `prompt_cache_miss_tokens`）及 OpenAI 兼容端点（`prompt_tokens_details.cached_tokens`）

---

## 1. 概述

gou-ide 的 prompt 发送链路围绕 **LLM 服务端 KV Cache（上下文缓存）** 做了多层设计：

- **静态/动态分界**：以 `CacheBoundary`（`<!--- CACHE_BOUNDARY --->`）为界，把 system prompt 拆成
  静态前缀（每次请求不变）与动态后缀（会话特定内容），最大化前缀复用；
- **本地缓存层**：system 静态前缀缓存、dynamic 30s TTL 缓存、会话上下文（resumeCtx）300s TTL 缓存，
  用于「输出稳定 → 前缀稳定 → provider 缓存连续命中」；
- **诊断能力**：`WB_CACHE_DIAG=1` 开启前缀形状快照比对（`CacheShape`），定位缓存断裂点；
- **统计能力**：`Usage` 双拼写归一化解析 + 会话级累计命中率（`sessionCache`）+ 磁盘持久化。

---

## 2. 相关源文件清单与职责

| 文件 | 职责 |
|------|------|
| `internal/agent/cache_shape.go` | 缓存形状核心：`PrefixShape` 快照、`CaptureShape`/`CompareShape` 前缀比对、`sessionCache` 会话累计统计、`promptCompileCache` 编译缓存（预留）、`PromptCacheWarmer` 预热器 |
| `internal/agent/loop.go` | `CacheBoundary` 常量（L51）、缓存诊断开关与全局状态（L53-60/L382）、system 消息注入（L414）、tools 获取与精简（L438）、LLM 调用与 usage 处理（L560-589）、`emitCacheShape`（L831）、`emitCacheUsage`（L873）、`buildCallContext`（L905）、`ComposeSystemPrompt`（L1447） |
| `internal/agent/types.go` | `Usage` 结构（L57-72）+ `UnmarshalJSON` 双拼写归一化（L79-98）、`PromptBreakdown` 构成估算结构（L100-109） |
| `internal/agent/session_context.go` | `resumeCtxCacheTTL`（L38）、`BuildResumeContext`（L967，300s TTL 缓存输出，稳定 dynamic 后缀） |
| `internal/agent/compress.go` | `EstimateBreakdown`（L327）prompt token 构成估算；压缩阈值常量（L21-26） |
| `internal/agent/history_condense.go` | `CondenseHistoryByPressure`（L50）token 压力触发压缩，小历史保持逐字节不变 |
| `internal/agent/cache_diag_test.go` | 缓存命中诊断测试：工具顺序稳定性 / UTF-8 截断 / Usage 兼容拼写 / 压缩触发 |
| `internal/agent/kv_cache_test.go` | KV 缓存相关测试（`CacheBoundary` 存在性、system 稳定性） |
| `internal/agent/storm_breaker_test.go` | `CaptureShape` 单元测试（system/tools 变化检测） |
| `cmd/companion/web_server.go` | system prompt 组装入口：`systemStaticPrefixCache`（L2213）、`buildSystemStaticPrefix`（L2230）、`buildWebSystemDynamicCache`（L2276）、`buildWebSystemDynamic`（L2284，30s TTL）、`buildWebSystemPrompt`（L2368）、`WarmUp` 调用（L329）、`buildWebLoopOpts`（L2414） |
| `internal/desktopbridge/desktopbridge.go` | 桌面端同链路：`BuildResumeContext` 调用（L452） |

---

## 3. 整体链路图

```
┌──────────────────────────── 发送链路（每轮对话）────────────────────────────┐
│                                                                              │
│  handleChatSend (web_server.go)                                              │
│    │                                                                          │
│    ▼                                                                          │
│  buildWebLoopOpts (web_server.go:2414)                                        │
│    │                                                                          │
│    ├─► sys = buildWebSystemPrompt()  (:2472)                                  │
│    │     │                                                                     │
│    │     ├─► buildSystemStaticPrefix()  (:2230)  ──► systemStaticPrefixCache  │
│    │     │     └ key = roots|SystemInstructions|Philosophy (:2222)            │
│    │     ├─► buildWebSystemDynamic()    (:2284)  ──► buildWebSystemDynamicCache│
│    │     │     └ 30s TTL；skills/记忆/规则/知识库/项目环境                      │
│    │     ├─► 插件系统提示段 (PluginPromptSections) 追加 dynamic 尾部 (:2370)   │
│    │     └─► ComposeSystemPrompt(static, dynamic)  (loop.go:1447)             │
│    │           = static + CacheBoundary + dynamic   ← 唯一 boundary           │
│    │                                                                          │
│    ├─► history = store.LoadAll(convID)  (:2490)  保留 reasoning_content      │
│    ├─► resumeCtx = BuildResumeContext(...)  (:2529)  ──► 300s TTL 缓存        │
│    │     └ sys += "\n\n" + resumeCtx      （dynamic 尾部追加，不进静态前缀）   │
│    ├─► history = CondenseHistoryByPressure(...)  (:2541)  token 压力触发      │
│    └─► LoopOpts{ System: sys, Provider, ... }                                 │
│                                                                              │
│  Loop.Run (loop.go:375)                                                       │
│    │                                                                          │
│    ├─► msgs += {RoleSystem, l.System}  (:414)   ← system 消息                 │
│    ├─► tools = Registry.Definitions() + ApplyConciseToolDescriptions  (:438)  │
│    ├─► maybeCompact 兜底 (:460)                                               │
│    │                                                                          │
│    └─► 每轮迭代:                                                              │
│          ├─► callMsgs = buildCallContext(msgs)  (:532 / :905)                 │
│          │     └ 背景块(固定)插到当前任务前；动态日志追加末尾                    │
│          ├─► [WB_CACHE_DIAG] emitCacheShape(callMsgs, tools)  (:555)          │
│          ├─► Provider.Chat(ctx, callMsgs, tools, cb)  (:560)                  │
│          │     └ 流式 chunk：Content/Reasoning → UI                           │
│          └─► chunk.Usage 处理  (:570)                                         │
│                ├─► lastPromptTokens = PromptTokens  （驱动压缩判定）           │
│                ├─► EstimateBreakdown (仅 Provider 未返回时估算)               │
│                ├─► emit(EventUsage)  →  UI 侧栏统计                           │
│                ├─► [WB_CACHE_DIAG] emitCacheUsage → sessionCache 累计          │
│                └─► SaveTokenUsage(ForRoot) → 磁盘持久化                       │
│                                                                              │
└──────────────────────────────────────────────────────────────────────────────┘
```

---

## 4. 各环节详解

### 4.1 缓存键如何计算

存在 **4 套本地缓存键** + **1 套诊断形状**，职责各不相同：

| 缓存 | 位置 | 键 | 说明 |
|------|------|-----|------|
| `systemStaticPrefixCache`（单槽） | web_server.go:2216 | `roots \| SystemInstructions \| PhilosophyPrompt`（字符串拼接，L2222） | 静态前缀内容级缓存；键不变则前缀逐字节复用 |
| `buildWebSystemDynamicCache`（单槽） | web_server.go:2276 | 无键，30s TTL | dynamic 后缀；TTL 内复用同一输出 |
| `resumeCtxCache`（单槽） | session_context.go:28-38 | `convID \| roots`（L970） | 会话上下文输出；300s TTL 内复用 |
| `promptCompileCache`（map） | cache_shape.go:135-138 | `sha256(roots‖instructions‖philosophy)[:16]`（L175） | ★ 预留层，见 §6 |
| `PrefixShape`（诊断快照） | cache_shape.go:18-24 | `sha256(static+dynamic+tools)[:8]` 等 5 个哈希 | 只用于比对诊断，不用于复用 |

诊断形状 `CaptureShape`（cache_shape.go:53）对同一份输入计算 **5 个哈希**：

```
PrefixShape {
  SystemHash   = sha256(静态前缀)[:8]   // CacheBoundary 之前 → 影响 provider 缓存
  DynamicHash  = sha256(动态后缀)[:8]   // CacheBoundary 之后 → 变化不影响前缀缓存
  ToolsHash    = sha256(归一化 tools JSON)[:8]  // 按 name+desc 排序后哈希（顺序无关）
  ToolsRawHash = sha256(原始 tools JSON)[:8]   // 原始顺序（诊断顺序稳定性）
  PrefixHash   = sha256(system+dynamic+tools)[:8]  // 整体指纹
}
```

`normalizeToolDefs`（cache_shape.go:71）按 `Function.Name`、`Function.Description` 字典序排序，
保证 **同一批工具无论装配顺序如何，归一化哈希一致**。

### 4.2 预热 WarmUp 在何时执行、缓存什么

- **时机**：`cmd/companion/web_server.go:329`，web server 启动时后台 goroutine 执行一次：
  ```go
  go agent.PromptCacheWarmer.WarmUp(buildWebSystemPrompt)
  ```
  不阻塞主流程。
- **单例保护**：`promptCacheWarmer.WarmUp`（cache_shape.go:201）带 `warmedUp` 标志 + mutex，
  多次调用只执行一次构建。
- **实际效果**：`buildWebSystemPrompt()` 内部触发的写缓存是
  `buildSystemStaticPrefix()` 的 `systemStaticPrefixCache`（写入静态前缀字符串）
  与 `buildWebSystemDynamic()` 的 `buildWebSystemDynamicCache`（写入 30s TTL 动态段）。
  首次真实请求时这两段字符串**无需重新拼接**，且输出与启动时逐字节一致 → provider 前缀稳定。

> ⚠️ 注意：`WarmUp` 注释写的是「触发 `CacheSystemPrompt` 写缓存」，但 `buildWebSystemPrompt`
> 主链路实际**并未调用** `CacheSystemPrompt`/`GetCachedSystemPrompt`——`promptCompileCache`
> 目前是未接入的预留层（见 §6）。

### 4.3 发送时 prompt 如何组装

**system 消息（静态段 + 动态段）**：

```
静态前缀（CacheBoundary 之前，buildSystemStaticPrefix, web_server.go:2230）
├─ persona 槽位：插件 deployment:persona 贡献段（无则默认身份段）
├─ rules 槽位：  插件 deployment:rules 贡献段（无则默认行为准则）
├─ DefaultSystemPromptWithOverrides(folders, persona, rules)  ← 工作区路径固定，放静态前缀
├─ 系统级指令（Settings.SystemInstructions）
├─ PhilosophyPrompt（哲学指导思想）
└─ SelfManagementPrompt（自我管理规则）
──────────────── CacheBoundary（loop.go:51）────────────────
动态后缀（buildWebSystemDynamic, web_server.go:2284 + BuildResumeContext）
├─ skills 技能列表（skills.Prompt()）
├─ 长期记忆（LongTermMemoryPrompt）
├─ 项目约定（ProjectRules：AGENTS.md / CLAUDE.md / .pair/rules.md）
├─ 项目知识库（ProjectKnowledge, 2500 字截断）
├─ 项目环境（各工作区 .pair/project.md）
├─ 插件系统提示段（PluginPromptSections，仅当插件贡献时）(web_server.go:2370)
└─ 会话连贯性上下文（BuildResumeContext，web_server.go:2529 追加，300s TTL 缓存）
```

要点：
- **唯一 boundary**：`ComposeSystemPrompt(static, dynamic)`（loop.go:1447）统一拼接
  `static + CacheBoundary + dynamic`，杜绝双边界/漏边界。
- **时间戳不在 system 内**：已移至用户消息（web_server.go:2327 注释），避免 system 每轮变化。
- **resumeCtx 不进静态前缀**：在 `buildWebLoopOpts` 里 `sys += "\n\n" + resumeCtx`
  （web_server.go:2531），追加在 CacheBoundary 之后 → 变化不影响前缀缓存。

**tools**：
- `tools := l.Registry.Definitions()`（loop.go:438）——只取 `Enabled=true` 的工具；
- `ApplyConciseToolDescriptions(tools)`（loop.go:440）精简工具描述文言文。
  ★ 注释明确「DeepSeek 缓存不覆盖 tools，精简直接降 miss」；
- `Definitions()`（tools.go:318）内部按 name 字典序排序（对齐 harness `orderTools`），
  任意装配顺序输出一致 → tools 前缀稳定；
- `trimToolDesc`（tools.go:348）rune 安全截断 ~120 字符，修复了原字节截断产生无效 UTF-8
  导致工具描述乱码的缺陷。

**历史消息**：
- `store.LoadAll` 加载原始历史，**保留 reasoning_content**（DeepSeek 要求工具调用轮次必须回传，L2504）；
- `CondenseHistoryByPressure`（web_server.go:2541 → history_condense.go:50）：
  token 估算占比 `< 45%` 且未触 120K 硬地板时**保持原始历史逐字节不变**（缓存可连续命中）；
  超阈值才 `CondenseHistory`（最近 1 轮完整 + 倒数第 2 轮半压缩 + 更早轮次摘要）。
  2026-08-17 由「按轮数强制压缩」改为 token 压力触发——修复了小对话每轮改写历史前缀导致命中率骤降的问题。

**每次迭代的组装**（`buildCallContext`, loop.go:905）：
- 背景块（staleMsg 记忆/知识库过期检查 + buildInjectionMessage 历史摘要/自主模式提示
  + `backgroundCtxMarker` 前缀的外部注入消息）**固定内容插到「最后一条 user（当前任务）」之前**，
  不打断 system+历史的前缀；
- 动态内容（执行日志 buildLogBlock、用户反馈、时间预算等）**追加末尾**，随迭代增长不影响前缀。

### 4.4 响应 usage 的 cache_hit_tokens / cache_miss_tokens 如何解析统计

**结构定义**（types.go:57-72）：

```go
type Usage struct {
    PromptTokens          int `json:"prompt_tokens"`
    CompletionTokens      int `json:"completion_tokens"`
    TotalTokens           int `json:"total_tokens"`
    PromptCacheHitTokens  int `json:"prompt_cache_hit_tokens,omitempty"`  // DeepSeek 专有
    PromptCacheMissTokens int `json:"prompt_cache_miss_tokens,omitempty"`
    PromptTokensDetails   *struct{ CachedTokens int `json:"cached_tokens"` } // OpenAI 兼容
    PromptBreakdown
}
```

**解析归一化**（`UnmarshalJSON`, types.go:79-98）：
1. 标准 JSON 反序列化；
2. 若 `PromptCacheHitTokens == 0` 但 `PromptTokensDetails.CachedTokens > 0` →
   把 OpenAI 兼容拼写归一化到 `PromptCacheHitTokens`（对齐 harness `mapUsage`，2026-08-17）；
3. 若 hit>0 且 miss 缺失（部分网关只给 cached_tokens）→ 用 `prompt_tokens - hit` 反推 miss，
   避免前端命中率分母失真。

**统计路径**（loop.go:570-589，流式 chunk 回调内）：
1. `l.lastPromptTokens = Usage.PromptTokens` → 驱动下轮压缩判定（实测用量）；
2. `PromptBreakdown` 估算：仅当 Provider 未返回 breakdown 时，
   用 `EstimateBreakdown(callMsgs, tools, PromptTokens)`（compress.go:327）按
   system/skills/mcp/tool/history/other 六类占比归一化到 prompt_tokens；
3. `emit(Event{Type: EventUsage, Usage})` → 前端侧栏渲染命中/未命中与构成占比条；
4. `WB_CACHE_DIAG=1` 时 `emitCacheUsage`（loop.go:873）：
   - `sessionCache.record(hit, miss)`（cache_shape.go:115，原子计数，进程级累计）；
   - 输出本轮率 `hit/(hit+miss)` 与累计率 `Σhit/Σ(hit+miss)` 到 stderr；
5. `SaveTokenUsage(ForRoot)` 持久化到磁盘（页面刷新后恢复统计）。

### 4.5 缓存诊断（WB_CACHE_DIAG=1）

开关：`os.Getenv("WB_CACHE_DIAG") == "1"`（loop.go:382）。启动后每轮输出：

| 输出点 | 位置 | 内容 |
|--------|------|------|
| `emitCacheShape` | loop.go:831 | 每轮 LLM 调用前 `CaptureShape(systemPromptFromMsgs(callMsgs), tools)`，与上一轮（`cacheDiagPrev`，跨 Loop/Run 全局共享，L56-60）比对：首轮输出 5 个哈希 + `tools_n/tools_bytes`；`PrefixChanged=true` 输出「★缓存断裂 reasons=[system/tools]」；dynamic 变化单独标记（不影响缓存） |
| dynamic 段诊断 | web_server.go:2331-2358 | `buildWebSystemDynamic` 各段（skills/memory/rules/knowledge/env）哈希 + 技能行级列表，定位 dynamic 变化源 |
| 插件段诊断 | web_server.go:2373 | 插件系统提示段哈希/长度 |
| usageGuide 诊断 | web_server.go:2477 | 工具使用指南哈希（工具状态/顺序漂移检测） |
| `emitCacheUsage` | loop.go:873 | 本轮/累计 hit、miss、命中率 |

---

## 5. 缓存失效原因分析

按「影响 provider 前缀缓存」的严重程度排序：

### 5.1 高危：system 静态前缀变化（`PrefixChanged=true`）
1. **`systemStaticPrefixKey` 变化**：工作区 roots、`SystemInstructions`、`PhilosophyPrompt`
   任一变化 → 静态前缀整体重建 → 所有后续请求前缀断裂。
2. **persona/rules 插件槽位变化**：`deployment:persona` / `deployment:rules` 插件段
   加载/卸载/修改 → `DefaultSystemPromptWithOverrides` 输出变化（无贡献时输出与默认逐字节一致，稳定）。
3. **程序升级**：`DefaultSystemPromptWithOverrides` / `SelfManagementPrompt` 内容变更。

### 5.2 高危：tools 变化
1. **工具启用/禁用状态变化**：`Definitions()` 只导出 `Enabled=true`（tools.go:324），
   状态变化直接改变 tools 数组；
2. **插件热重载 / 工具集变化 / MCP 服务器启停**：注册集合变化 → tools JSON 变化；
3. **描述截断漂移**：`trimToolDesc` 输出随描述文案变化（已修复 UTF-8 乱码问题，
   但描述内容本身变化仍会断缓存）。
> 注：DeepSeek 缓存不覆盖 tools 段——tools 变化主要影响**前缀位置**与 miss 量，
> tools 体积精简（`ApplyConciseToolDescriptions`）直接降低 miss 成本。

### 5.3 中危：dynamic 后缀变化（DeepSeek 按 system 消息整体前缀匹配，
boundary 后变化同样破坏 system 段的缓存命中）
1. **`resumeCtxCacheTTL`（300s）过期重建**：记忆召回 / Git 状态 / 任务进度 / 代码图谱统计等
   内容变化 → resumeCtx 输出变化（已从 60s 调大到 300s，2026-08-17，命中率 99%→50% 问题的修复）；
2. **`buildWebSystemDynamicCache`（30s TTL）过期重建**：skills 列表、知识库、
   项目环境（.pair/project.md）文件变化；
3. **插件系统提示段变化**：插件加载/卸载改变 `PluginPromptSections` 输出；
4. **`UsageGuideText` 变化**：工具状态/顺序漂移影响使用指南文本。

### 5.4 中危：历史消息变化
1. **`CondenseHistoryByPressure` 触发压缩**（估算占比 ≥45% 或 ≥120K 硬地板）：
   压缩/替换中段消息 → 消息数组从压缩位置断裂（压缩与 KV 前缀的根本矛盾，history_condense.go:79 注释）；
2. **手动压缩**（前端 CompactRequested → `maybeCompact`，loop.go:502）；
3. **跨轮次历史追加**：历史天然增长属于前缀连续命中，不算断裂；但若上一轮末尾消息被改写
   （如旧版按轮数压缩）则断裂。

### 5.5 其他
- **跨对话**：convID 不同 → resumeCtx 缓存键不同 → 每新对话重建 dynamic 后缀（合理，不可避免）；
- **`reasoning_content` 缺失/变化**：DeepSeek 要求工具调用轮次必须回传 reasoning_content，
  缺失可能引发 400 或上下文断裂（已修复为保留，L2504）；
- **双 boundary / 漏 boundary**：`ComposeSystemPrompt` 统一添加已规避；
  若绕过该函数直接拼接 system 则可能把可变内容放进静态前缀。

---

## 6. 关键发现与待办

1. **★ `promptCompileCache` 未接入主链路**：`CacheSystemPrompt` / `GetCachedSystemPrompt`
   / `ClearPromptCompileCache` / `PromptCompileCacheSize`（cache_shape.go:133-185）仅有定义，
   全仓库无外部调用（除 `WarmUp` 注释的间接描述）。`WarmUp` 实际预热的是
   `systemStaticPrefixCache` + `buildWebSystemDynamicCache` 两个局部缓存。
   → 建议：要么接入 `buildWebSystemPrompt`（增加一层 map 缓存，按 roots/instructions/philosophy
   键命中后免去字符串拼接），要么删除死代码并修正 `WarmUp` 注释，避免误导。

2. **缓存层均为单槽/小 map，无淘汰策略**：`systemStaticPrefixCache`/`buildWebSystemDynamicCache`/
   `resumeCtxCache` 各只有 1 个槽位，多项目切换时互相覆盖。对单项目开发影响不大，
   但多项目工作区频繁切换时，每次切回都需重建 dynamic（30s/300s TTL 内尚可复用）。

3. **`resumeCtxCache` 键不含 currentTask**：键为 `convID|roots`（session_context.go:970），
   同对话内任务切换时输出复用（TTL 内），可能注入过期任务进度——这是「缓存稳定性 vs 信息新鲜度」
   的权衡，300s TTL 是刻意选择。

4. **tools 排序一致性已由 `Definitions()` 字典序保证**（tools.go:335），
   但 `ApplyConciseToolDescriptions`（loop.go:440）的输出是否确定性、是否随配置变化，
   是 tools 前缀稳定的下一个关注点（cache_diag_test.go 已覆盖 Definitions 顺序稳定性）。

---

## 7. 结论

本项目的缓存设计是「**本地输出稳定化 + 服务端前缀复用**」双层结构：

- **本地层**（静态前缀缓存 / dynamic TTL / resumeCtx TTL / 压缩阈值）保证发送到
  provider 的 system、tools、历史**逐字节稳定**；
- **服务端层**（DeepSeek 上下文缓存）按公共前缀自动复用 KV，命中反馈在
  `usage.prompt_cache_hit_tokens`（或 OpenAI 兼容 `cached_tokens`）；
- **诊断层**（`WB_CACHE_DIAG` + `CacheShape`）把「缓存为什么断了」变成可观测的
  `[cache-diag]` 日志，配合 `cache_diag_test.go` 回归用例守住工具顺序/编码/压缩触发等边界。

已修复的历史断裂源（2026-08-17 批次）：resumeCtx TTL 60s→300s、按轮数压缩→token 压力触发、
tool 描述 UTF-8 截断、Definitions 字典序排序、Usage OpenAI 兼容拼写归一化。
遗留优化方向见 §6 与最终回复中的建议。

---

## 8. 2026-09-03 实测修复：命中率低的两大根因（跨轮首请求 0% → 98.2%）

> 背景：实测「命中率非常低」——单轮内迭代命中 93.7%，但**每轮对话的首请求恒为 0%**。
> 用 `WB_CACHE_DIAG=1` 抓跨轮日志定位出两个根因（均与「messages 第一条/system 与
> 完整输入前缀」的缓存匹配机制有关）：

### 8.1 根因 A：会话连贯性上下文（resumeCtx）拼入 system 尾部

- **现象**：`buildWebLoopOpts`（web_server.go）把 `BuildResumeContext()` 输出
  （任务进度/对话摘要/Git 状态/代码图谱统计等，**每轮内容必变**）直接
  `sys += "\n\n" + resumeCtx` 追加到 system 消息尾部。
- **机制**：system 是 messages **第一条**（输入前缀开头的第一个 token 块）。
  DeepSeek 按「完整输入公共前缀」匹配缓存——system 尾部变化 → 前缀在第一条
  消息内部断裂 → system 之后的历史/任务**全部 miss**（命中仅剩静态前缀部分）。
  注释「resumeCtx 在 CACHE_BOUNDARY 之后不影响前缀」是**错误认知**：
  boundary 之后仍是 system 消息的一部分，只要 system 内容有变化，前缀就断。
- **修复**：resumeCtx 迁入「背景上下文快照」（消息流尾部 append-only）：
  `LoopOpts.ResumeContext` → `Loop.ResumeContext` → `buildSnapshotContent` ① 段
  （Go 回退）/ `snapshot.parts().resume` + JS `buildSnapshotText` ① 段（JS 循环）→
  `syncContextSnapshot` 注入消息流。变化只断快照之后，system+历史前缀单调延展。
  goal 段（含 Rounds 每轮递增）同样从 `loop.System +=`（session_manager
  Start/续轮两处）改挂 `loop.ResumeContext +=`。

### 8.2 根因 B：首步极简工具面（StagedTools）——tools 集合跨轮切换

- **现象**：每次对话（每个 Run 的 turn 都从 1 开始）首个 LLM 请求注入极简面
  （8 个工具），后续请求恢复全量面（54 个工具）。
- **机制**：DeepSeek 缓存匹配**完整输入前缀（含工具定义序列化部分）**。
  上一轮最后请求 = 54 工具；本轮首请求 = 8 工具 → 工具集合不同 → 前缀从头断
  → 首请求 0% 命中；第 2 请求恢复 54 工具才命中（实测 step2 起 87.5%）。
- **修复（砍掉极简机制）**：极简工具面整体移除——`StagedTools/StagedToolGroups/
  FilterStagedTools/tools_staging.go`、agentloop 插件 `stagedToolGroups` 配置、
  装配透传（jsplugin_loopfactory）、测试全部删除；统一全量工具面。
  注：极简面 91.7% 首步选对率 vs 全量 87.5% 的实测优势在缓存成本面前不值
  （每轮首请求全 miss 的 token 损失远大于首步选对率差）。

### 8.3 实测结果（DeepSeek-V4-Flash，siliconflow，WB_CACHE_DIAG=1）

| 场景 | 修复前 | 修复后 |
|------|--------|--------|
| 单轮内迭代 | 93.7%（step≥2） | 96.1%~98.2% |
| **跨轮首请求** | **0%（tools 8 vs 54 + resumeCtx 变化）** | **98.2%**（tools 54 恒定 + system 稳定 + 快照 append） |
| 前缀断裂日志 | 无（当时未判 tools 面；含 tools 变化断链） | 0 条「★缓存断裂」 |
| [cache-diag] system | 每轮首请求后 system hash 稳定 | 恒 8aa596a2（多轮跨重启一致） |

### 8.4 遗留观察

1. **tools 变化仍会断前缀**：工具启用/禁用、插件装载/卸载会改变 tools 集合
   （54 → N）→ 下次请求前缀从头断。这是「配置变更」级断裂，正常且不可免，
   但工具状态应保持稳定（避免运行时频繁开关）。
2. **CompressedSummaries 追加**（压缩触发）也会在快照处断尾——快照机制已
   保证前缀到压缩点为止连续，属可接受损失。
3. **快照体积**：每轮 resume 变化追加一条快照，历史中快照会累积（每轮 1 条）。
   旧快照保留是前缀连续的前提；若会话极长（>50 轮），可考虑「快照段落整体
   作为压缩候选」——但压缩会断前缀，权衡后暂不处理。
