// subagent.go 多 agent 编排：SubAgent / AgentTree。
// 与 ADK 关键差异：**不做上下文隔离**——子 agent 共享父 []Message 前缀，
// 保证 LLM prompt cache 命中（前 N 条逐字节一致）。子 system 作追加 instruction 而非替换。
//
// 编排示例（coordinator 协调）：
//	父调 delegate_task(agent="planner", task="分析并给计划")
//	  子 agent 看到完整父历史 + task（缓存命中），产出计划后调 finish_task(result=计划)
//	  计划作为工具结果回到父 []Message
//	父调 delegate_task(agent="coder", task="按计划重构")
//	  coder 看到父历史（含 planner 计划）+ task，缓存命中

package agent

// SubAgent 子 agent 定义（编排树节点）。
type SubAgent struct {
	Name        string   // 唯一名（delegate_task 的 agent_name）
	Description string   // LLM 据此决定是否委托
	System      string   // 子 agent 专属系统提示（空=继承父上下文；非空作追加 instruction，不替换，保前缀稳定）
	Tools       []string // 工具白名单（工具名；空=继承父全部工具）
	MaxIter     int      // 最大迭代（0=用 Loop 默认 30）
}

// AgentTree agent 编排树。
// Root 为根协调器；Children 含 root 及所有子 agent（name→SubAgent）；Parent 记录父子关系。
type AgentTree struct {
	Root     *SubAgent
	Children map[string]*SubAgent // name → sub-agent（含 root）
	Parent   map[string]string    // child name → parent name（root 的父为 ""）
}

// NewAgentTree 构造编排树。root 为根协调器，children 挂到 root 下。
// root 可空（仅构造空树后用 Add 手动挂）。
func NewAgentTree(root *SubAgent, children ...*SubAgent) *AgentTree {
	t := &AgentTree{
		Children: map[string]*SubAgent{},
		Parent:   map[string]string{},
	}
	if root != nil {
		t.Root = root
		t.Children[root.Name] = root
		t.Parent[root.Name] = ""
	}
	for _, c := range children {
		t.Add(c, root.Name)
	}
	return t
}

// Add 添加子 agent，挂到 parentName 下。parentName 不存在时不挂（容错）。
func (t *AgentTree) Add(sa *SubAgent, parentName string) {
	if sa == nil {
		return
	}
	t.Children[sa.Name] = sa
	t.Parent[sa.Name] = parentName
}

// Find 按名查找 agent（含 root）。找不到返回 nil。
func (t *AgentTree) Find(name string) *SubAgent {
	return t.Children[name]
}

// ParentOf 返回 name 的父 agent 名（root 返回 ""，不存在返回 ""）。
func (t *AgentTree) ParentOf(name string) string {
	return t.Parent[name]
}

// SubNames 返回所有非 root 的子 agent 名（delegate_task 可用目标），顺序不稳定。
func (t *AgentTree) SubNames() []string {
	var names []string
	for n := range t.Children {
		if t.Root == nil || n != t.Root.Name {
			names = append(names, n)
		}
	}
	return names
}
