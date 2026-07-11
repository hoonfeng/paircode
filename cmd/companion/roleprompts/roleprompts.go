// 角色提示 & 哲学的 config 加载 —— 落地「角色哲学指导从 config 加载、不硬编码」。
//   角色系统提示：config/roles/<id>.md（缺失则写入内置默认，供用户直接编辑）。
//   角色哲学：设置面板编辑的（settings.json）优先，否则 config/philosophy/<id>.txt 默认文件。
// 复刻参考 roles.ts「basePrompt(prompts/roles/*.md) + philosophy(philosophy/roles/*.txt)」的文件驱动结构。
//
//go:build windows

package roleprompts

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/hoonfeng/paircode/cmd/companion/agent"
	"github.com/hoonfeng/paircode/cmd/companion/core"
)

// ─── 哲学/角色数据（从 settings 包内联至此，消除 GWui 依赖）──

// PhilosophyEntry 一部经典哲学（思想 tab 可选）。
type PhilosophyEntry struct {
	ID   string
	Name string
}

// RoleEntry 一个角色（哲学小节标题用）。
type RoleEntry struct {
	ID   string
	Name string
}

// Philosophies 可选的经典哲学列表。
var Philosophies = []PhilosophyEntry{
	{"tao-te-ching", "《道德经》"},
	{"huangdi-yinfu-jing", "《黄帝阴符经》"},
	{"sunzi-bingfa", "《孙子兵法》"},
	{"lunyu", "《论语》"},
	{"yijing", "《易经》"},
	{"zhongyong", "《中庸》"},
	{"daxue", "《大学》"},
}

// RoleEntries 角色列表（哲学小节标题用）。
var RoleEntries = []RoleEntry{
	{"planner", "规划"},
	{"reviewer", "审核"},
	{"judge", "评测"},
	{"explorer", "探索"},
	{"verifier", "验证"},
	{"debugger", "调试"},
	{"executor", "执行"},
}

// Contains 检查字符串是否在切片中。
func Contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// ensureRolePrompts 启动时把内置默认角色提示落地到 config/roles/*.md（仅缺失时写），
// 让用户一开始就能在 config 里看到并编辑（规划/审核/评分；执行角色提示含工作区根、动态生成不落地）。
func Ensure() {
	LoadRolePrompt("planner", agent.DefaultPlannerPrompt())
	LoadRolePrompt("reviewer", agent.DefaultReviewerPrompt())
	LoadRolePrompt("judge", agent.DefaultJudgePrompt())
	LoadRolePrompt("explorer", DefaultExplorerPrompt)
	LoadRolePrompt("verifier", DefaultVerifierPrompt)
}

// loadRolePrompt 读 config/roles/<id>.md 作角色系统提示；不存在/为空 → 写入内置默认（方便用户编辑）再返回默认。
func LoadRolePrompt(id, def string) string {
	p := filepath.Join(core.ConfigDir(), "roles", id+".md")
	if data, err := os.ReadFile(p); err == nil {
		if s := strings.TrimSpace(string(data)); s != "" {
			return s
		}
	}
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	_ = os.WriteFile(p, []byte(def), 0o644) // 首次落地默认 → 之后从文件读，改文件即改角色提示（免重编）
	return def
}

// roleCustomPhilosophy 某角色自定义哲学——config 默认文件 + 设置面板编辑的，**互补叠加**（非替代）：
// config/philosophy/<id>.txt（shipped 默认）在前，用户在设置里加的在后，两者都注入。
func RoleCustomPhilosophy(roleID string) string {
	var parts []string
	if data, err := os.ReadFile(filepath.Join(core.ConfigDir(), "philosophy", roleID+".txt")); err == nil {
		if s := strings.TrimSpace(string(data)); s != "" {
			parts = append(parts, s)
		}
	}
	if c := strings.TrimSpace(core.Settings.PhilosophyRoles[roleID]); c != "" {
		parts = append(parts, c)
	}
	return strings.Join(parts, "\n\n")
}

// 探索/验证角色的内置默认提示（只读阶段用；复刻参考 explorer.md/verifier.md，按 companion 工具改写）。
// 与「角色哲学」互补：阶段 Loop 的 System = 此提示 + rolePhilosophy(角色)。可经 config/roles/*.md 覆盖。
const DefaultExplorerPrompt = `# 角色
你是探索 Agent（Explorer）——代码库导航与定位专家。职责是收集信息，绝不修改任何文件（写操作会被系统自动拦截）。

# 工作方式
- 先 list_files 看结构，再 search_content/search_files 定位，最后 read_file 细读关键文件。
- 串行按需：读一个→分析→再决定下一个；找到目标即汇报，不过度探索（最多约 6 轮）。

# 输出
简明汇报与任务相关的关键文件、结构、上下文要点（中文）。你的任务是导航定位，不深入实现细节、不动手改。`

const DefaultVerifierPrompt = `# 角色
你是验证 Agent（Verifier）——确认改动已正确生效。只读，绝不修改任何文件。

# 流程
1. read_file 读改动后的文件，确认修改已应用、内容正确。
2. 必要时 run_command 跑构建/测试，确认无报错。
3. 汇总：验证通过，或列出未通过项与遗留问题（中文）。`

// classicsPhilosophy 选中经典的共享指导（所有角色共用，每个 Agent 注入一次，避免重复）。
func classicsPhilosophy() string {
	if !core.Settings.PhilosophyEnabled {
		return ""
	}
	var names []string
	for _, c := range Philosophies {
		if Contains(core.Settings.PhilosophySelected, c.ID) {
			names = append(names, c.Name)
		}
	}
	if len(names) == 0 {
		return ""
	}
	return "\n\n# 指导思想\n以下经典的智慧贯穿你的分析、决策与行动：" + strings.Join(names, "、") + "。"
}

// roleSpecificPhilosophy 某角色专属哲学小节（config 默认 + 用户自定义，互补叠加；不含共享经典）。
func RoleSpecificPhilosophy(roleID string) string {
	if !core.Settings.PhilosophyEnabled {
		return ""
	}
	if c := RoleCustomPhilosophy(roleID); c != "" {
		return "\n\n## " + RoleDisplayName(roleID) + "哲学\n" + c
	}
	return ""
}

// rolePhilosophy 单角色完整哲学（共享经典 + 该角色专属）。用于独立角色 Agent（planner/reviewer/explorer/verifier）。
// 与角色「提示词」互补：各 Agent 的 System = 角色提示(config/roles/*.md) + rolePhilosophy(角色)，两者并存非替代。
func RolePhilosophy(roleID string) string {
	return classicsPhilosophy() + RoleSpecificPhilosophy(roleID)
}

// PhilosophyPrompt 主执行 Agent 的指导思想（共享经典，启用时）。bridge 注入主系统提示。
func PhilosophyPrompt() string {
	return classicsPhilosophy()
}

// roleDisplayName 角色显示名（哲学小节标题用）。
func RoleDisplayName(roleID string) string {
	for _, r := range RoleEntries {
		if r.ID == roleID {
			return r.Name
		}
	}
	return "通用 Agent"
}
