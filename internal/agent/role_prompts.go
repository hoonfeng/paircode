// role_prompts.go — 角色提示「磁盘优先」加载（内容资产插件化，对齐「内容在磁盘」）
//
// ★ 缺口闭环（t1 报告 C1/C2，2026-09）：config/roles/*.md 随包发布（packager.json
//   dist.include）但运行时代码零读取——reviewer/planner/judge 提示 100% Go 硬编码、
//   不可被磁盘覆盖。本文件实现磁盘优先 loader：
//
//	config/roles/<name>.md 存在且非空 → 用之（内容在磁盘，不重编译即可覆盖角色）
//	缺失 / 不可读 / 空白       → 回退 Go 内置提示（reviewerSystemPrompt 等）
//
//   缓存语义：角色提示属静态内容资产，进程启动后首次读取即缓存（改文件需重启，
//   与 settings.json 等配置文件一致）；LoadRolePromptReset 仅供测试复位缓存。
//
// ★ 命名对齐：roles 目录文件名 reviewer.md / planner.md / judge.md 分别对应
//   DefaultReviewerPrompt / DefaultPlannerPrompt / DefaultJudgePrompt 的覆盖源。
//   explorer.md / verifier.md 暂未消费（预留内容资产，未来角色装配即读）。

package agent

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/hoonfeng/paircode/internal/core"
)

// rolePromptCache 角色提示内容缓存（name → 磁盘内容；空串 = 磁盘无覆盖）。
var (
	rolePromptCacheMu sync.RWMutex
	rolePromptCache   = map[string]string{}
	// rolePromptBaseDir 角色提示目录覆盖（测试 seam；空 = core.ConfigDir()/roles）。
	rolePromptBaseDir string
)

// rolePromptDir 返回角色提示目录（默认 <ConfigDir>/roles；测试可覆盖）。
func rolePromptDir() string {
	if rolePromptBaseDir != "" {
		return rolePromptBaseDir
	}
	return filepath.Join(core.ConfigDir(), "roles")
}

// SetRolePromptBaseDirForTest 覆盖角色提示目录（测试用；空串还原默认）。
func SetRolePromptBaseDirForTest(dir string) {
	rolePromptCacheMu.Lock()
	rolePromptBaseDir = dir
	rolePromptCache = map[string]string{}
	rolePromptCacheMu.Unlock()
}

// rolePromptFilePath 返回角色提示磁盘路径：<rolesDir>/<name>.md。
// rolesDir 与 core.Load 同源（exe 旁 config/，bin/ 子目录回退上级），
// 保证 web 端与桌面端读到同一份发布内容。
func rolePromptFilePath(name string) string {
	return filepath.Join(rolePromptDir(), name+".md")
}

// LoadRolePrompt 读取角色提示（磁盘优先，缺失回退 ""）。
// 返回 "" = 无磁盘覆盖（调用方用内置提示兜底）。
func LoadRolePrompt(name string) string {
	if strings.TrimSpace(name) == "" {
		return ""
	}
	rolePromptCacheMu.RLock()
	v, ok := rolePromptCache[name]
	rolePromptCacheMu.RUnlock()
	if ok {
		return v
	}
	// 缓存未命中：读磁盘（缺失/损坏视为无覆盖，缓存空串防重复 IO）
	v = ""
	if b, err := os.ReadFile(rolePromptFilePath(name)); err == nil {
		if s := strings.TrimSpace(string(b)); s != "" {
			v = s
		}
	}
	rolePromptCacheMu.Lock()
	rolePromptCache[name] = v
	rolePromptCacheMu.Unlock()
	return v
}

// LoadRolePromptReset 清空角色提示缓存（测试用；生产无热更新需求）。
func LoadRolePromptReset() {
	rolePromptCacheMu.Lock()
	rolePromptCache = map[string]string{}
	rolePromptCacheMu.Unlock()
}

// ─── Judge（评测角色）提示 ────────────────────────────────────
// ★ Round4 A2 处置：原 evaluator.go（LLM-as-Judge 评测 Agent，零生产调用）
//   已删除——DefaultJudgePrompt/judgeSystemPrompt 迁入本文件保留（plugin-round3
//   C1/C2 闭环：磁盘优先 loader 引用）。评分系统本体为 bench 归档，生产零装配。

// DefaultJudgePrompt 评测角色提示（磁盘优先：config/roles/judge.md 覆盖；
// 缺失/不可读时回退内置 judgeSystemPrompt）。
func DefaultJudgePrompt() string {
	if s := LoadRolePrompt("judge"); s != "" {
		return s
	}
	return judgeSystemPrompt
}

// judgeSystemPrompt 复刻参考 bench/evaluator.ts 的 JUDGE_SYSTEM_PROMPT（评分维度 + 输出格式）。
const judgeSystemPrompt = `你是一个严格的代码质量评审专家。请根据以下维度评估任务完成质量：

` + AIIdentityAwareness + `## 评分维度

1. **完成度 (0-40)**: 任务要求的所有阶段是否都完成？输出是否完整？
2. **正确性 (0-30)**: 技术方案是否正确？代码/分析是否存在错误？
3. **深度 (0-20)**: 分析是否深入？是否考虑了架构/设计/权衡？
4. **效率 (0-10)**: 输出是否简洁清晰？结构是否合理？

## 输出格式

你必须只输出 JSON（不要其他任何内容）：
{"scores":{"completion":N,"correctness":N,"depth":N,"efficiency":N},"total":N,"strengths":["...",...],"weaknesses":["...",...],"feedback":"..."}`
