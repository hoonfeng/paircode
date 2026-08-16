package tsit

// node.go — 翻译 tree-sitter node.c，实现完整节点树遍历 API。
//
// 所有节点方法以 *Node 接收者实现。
// Node 不可变，通过 Tree 访问；Tree 中存储 internalNode 数组。

// ── 基础信息 ──────────────────────────────────────────

// Symbol 返回节点的数值类型 ID。
func (n *Node) Symbol() Symbol {
	if n == nil || n.tree == nil || n.id == 0 {
		return 0
	}
	idx := int(n.id) - 1
	if idx < 0 || idx >= len(n.tree.nodes) {
		return 0
	}
	return n.tree.nodes[idx].symbol
}

// GrammarType 返回节点在语法中的原始类型（忽略别名）。
func (n *Node) GrammarType() string {
	return n.NodeType() // Go 版简化实现
}

// GrammarSymbol 返回节点在语法中的原始符号（忽略别名）。
func (n *Node) GrammarSymbol() Symbol {
	return n.Symbol()
}

// ── 位置 ──────────────────────────────────────────────

// Range 返回节点的完整范围（字节偏移 + 行列）。
func (n *Node) Range() Range {
	return Range{
		StartPoint: n.StartPoint(),
		EndPoint:   n.EndPoint(),
		StartByte:  n.StartByte(),
		EndByte:    n.EndByte(),
	}
}

// ── 节点属性 ──────────────────────────────────────────

// IsMissing 检查节点是否由解析器为错误恢复而插入。
func (n *Node) IsMissing() bool {
	if n == nil || n.tree == nil {
		return false
	}
	idx := int(n.id) - 1
	if idx < 0 || idx >= len(n.tree.nodes) {
		return false
	}
	return n.tree.nodes[idx].isMissing
}

// IsExtra 检查节点是否为额外节点（如注释）。
func (n *Node) IsExtra() bool {
	if n == nil || n.tree == nil {
		return false
	}
	idx := int(n.id) - 1
	if idx < 0 || idx >= len(n.tree.nodes) {
		return false
	}
	return n.tree.nodes[idx].isExtra
}

// HasChanges 检查节点是否已被编辑。Go 版简化：总是返回 false（无增量编辑跟踪）。
func (n *Node) HasChanges() bool {
	return false
}

// DescendantCount 返回节点的后代数量（包括自身）。
func (n *Node) DescendantCount() uint32 {
	if n == nil || n.tree == nil {
		return 0
	}
	idx := int(n.id) - 1
	if idx < 0 || idx >= len(n.tree.nodes) {
		return 0
	}
	inner := n.tree.nodes[idx]
	count := uint32(1) // 自身
	for _, childIdx := range inner.children {
		if childIdx >= 0 && childIdx < len(n.tree.nodes) {
			child := &Node{id: unsafePointer(childIdx + 1), tree: n.tree}
			count += child.DescendantCount()
		}
	}
	return count
}

// ── 树遍历 ────────────────────────────────────────────

// Parent 返回父节点。根节点返回 NullNode。
func (n *Node) Parent() *Node {
	if n == nil || n.tree == nil || n.id == 0 {
		return nil
	}
	idx := int(n.id) - 1
	if idx < 0 || idx >= len(n.tree.nodes) {
		return nil
	}
	parentIdx := n.tree.nodes[idx].parent
	if parentIdx < 0 {
		return nil
	}
	return &Node{
		id:   unsafePointer(parentIdx + 1),
		tree: n.tree,
		context: [4]uint32{
			n.tree.nodes[parentIdx].startByte,
			n.tree.nodes[parentIdx].endByte,
			0, 0,
		},
	}
}

// Child 返回第 index 个子节点（含匿名节点）。
func (n *Node) Child(index int) *Node {
	if n == nil || n.tree == nil || index < 0 {
		return nil
	}
	idx := int(n.id) - 1
	if idx < 0 || idx >= len(n.tree.nodes) {
		return nil
	}
	children := n.tree.nodes[idx].children
	if index >= len(children) {
		return nil
	}
	childIdx := children[index]
	if childIdx < 0 || childIdx >= len(n.tree.nodes) {
		return nil
	}
	return &Node{
		id:   unsafePointer(childIdx + 1),
		tree: n.tree,
		context: [4]uint32{
			n.tree.nodes[childIdx].startByte,
			n.tree.nodes[childIdx].endByte,
			0, 0,
		},
	}
}

// ChildCount 返回子节点数量（含匿名节点）。
func (n *Node) ChildCount() uint32 {
	if n == nil || n.tree == nil {
		return 0
	}
	idx := int(n.id) - 1
	if idx < 0 || idx >= len(n.tree.nodes) {
		return 0
	}
	return uint32(len(n.tree.nodes[idx].children))
}

// NamedChild 返回第 index 个命名子节点。
func (n *Node) NamedChild(index int) *Node {
	if n == nil || n.tree == nil || index < 0 {
		return nil
	}
	idx := int(n.id) - 1
	if idx < 0 || idx >= len(n.tree.nodes) {
		return nil
	}
	namedCount := 0
	for _, childIdx := range n.tree.nodes[idx].children {
		if childIdx >= 0 && childIdx < len(n.tree.nodes) && n.tree.nodes[childIdx].isNamed {
			if namedCount == index {
				return &Node{
					id:   unsafePointer(childIdx + 1),
					tree: n.tree,
				}
			}
			namedCount++
		}
	}
	return nil
}

// NamedChildCount 返回命名子节点数量。
func (n *Node) NamedChildCount() uint32 {
	if n == nil || n.tree == nil {
		return 0
	}
	idx := int(n.id) - 1
	if idx < 0 || idx >= len(n.tree.nodes) {
		return 0
	}
	return uint32(n.tree.nodes[idx].namedChildCount)
}

// ChildByFieldName 按字段名返回子节点。
func (n *Node) ChildByFieldName(name string) *Node {
	if n == nil || n.tree == nil || name == "" {
		return nil
	}
	idx := int(n.id) - 1
	if idx < 0 || idx >= len(n.tree.nodes) {
		return nil
	}

	// 对每个子节点，检查字段名
	for i, childIdx := range n.tree.nodes[idx].children {
		fieldName := n.FieldNameForChild(i)
		if fieldName == name {
			if childIdx >= 0 && childIdx < len(n.tree.nodes) {
				return &Node{
					id:   unsafePointer(childIdx + 1),
					tree: n.tree,
				}
			}
		}
	}
	return nil
}

// ChildByFieldID 按字段 ID 返回子节点。
func (n *Node) ChildByFieldID(fieldID FieldId) *Node {
	if n.tree == nil || n.tree.language == nil {
		return nil
	}
	name := n.tree.language.FieldNameForID(fieldID)
	return n.ChildByFieldName(name)
}

// FieldNameForChild 返回第 index 个子节点的字段名。
func (n *Node) FieldNameForChild(index int) string {
	if n == nil || n.tree == nil || n.tree.language == nil {
		return ""
	}
	// 简化实现：由语言定义提供字段映射
	// 在 Go 语言中通过解析函数声明时的 AST 提取字段名
	if l, ok := n.tree.language.(*GoLanguage); ok {
		return l.FieldNameForChild(n, index)
	}
	return ""
}

// FieldNameForNamedChild 返回第 index 个命名子节点的字段名。
func (n *Node) FieldNameForNamedChild(index int) string {
	if n == nil || n.tree == nil {
		return ""
	}
	// 找到第 index 个命名子节点，再查字段名
	namedCount := 0
	idx := int(n.id) - 1
	if idx < 0 || idx >= len(n.tree.nodes) {
		return ""
	}
	for i, childIdx := range n.tree.nodes[idx].children {
		if childIdx >= 0 && childIdx < len(n.tree.nodes) && n.tree.nodes[childIdx].isNamed {
			if namedCount == index {
				return n.FieldNameForChild(i)
			}
			namedCount++
		}
	}
	return ""
}

// ── 兄弟节点 ──────────────────────────────────────────

// NextSibling 返回下一个兄弟节点（含匿名）。
func (n *Node) NextSibling() *Node {
	if n == nil || n.tree == nil {
		return nil
	}
	parent := n.Parent()
	if parent == nil {
		return nil
	}
	idx := int(n.id) - 1
	parentIdx := int(parent.id) - 1
	if parentIdx < 0 || parentIdx >= len(n.tree.nodes) {
		return nil
	}
	// 在父节点子节点列表中找到自己，取下一个
	found := false
	for _, childIdx := range n.tree.nodes[parentIdx].children {
		if found {
			if childIdx >= 0 && childIdx < len(n.tree.nodes) {
				return &Node{
					id:   unsafePointer(childIdx + 1),
					tree: n.tree,
				}
			}
			return nil
		}
		if int(childIdx) == idx {
			found = true
		}
	}
	return nil
}

// PrevSibling 返回上一个兄弟节点（含匿名）。
func (n *Node) PrevSibling() *Node {
	if n == nil || n.tree == nil {
		return nil
	}
	parent := n.Parent()
	if parent == nil {
		return nil
	}
	idx := int(n.id) - 1
	parentIdx := int(parent.id) - 1
	if parentIdx < 0 || parentIdx >= len(n.tree.nodes) {
		return nil
	}
	var prev int = -1
	for _, childIdx := range n.tree.nodes[parentIdx].children {
		if int(childIdx) == idx {
			if prev >= 0 && prev < len(n.tree.nodes) {
				return &Node{
					id:   unsafePointer(prev + 1),
					tree: n.tree,
				}
			}
			return nil
		}
		prev = childIdx
	}
	return nil
}

// NextNamedSibling 返回下一个命名兄弟节点。
func (n *Node) NextNamedSibling() *Node {
	if n == nil || n.tree == nil {
		return nil
	}
	parent := n.Parent()
	if parent == nil {
		return nil
	}
	idx := int(n.id) - 1
	parentIdx := int(parent.id) - 1
	if parentIdx < 0 || parentIdx >= len(n.tree.nodes) {
		return nil
	}
	found := false
	for _, childIdx := range n.tree.nodes[parentIdx].children {
		if found {
			if childIdx >= 0 && childIdx < len(n.tree.nodes) && n.tree.nodes[childIdx].isNamed {
				return &Node{
					id:   unsafePointer(childIdx + 1),
					tree: n.tree,
				}
			}
		}
		if int(childIdx) == idx {
			found = true
		}
	}
	return nil
}

// PrevNamedSibling 返回上一个命名兄弟节点。
func (n *Node) PrevNamedSibling() *Node {
	if n == nil || n.tree == nil {
		return nil
	}
	parent := n.Parent()
	if parent == nil {
		return nil
	}
	idx := int(n.id) - 1
	parentIdx := int(parent.id) - 1
	if parentIdx < 0 || parentIdx >= len(n.tree.nodes) {
		return nil
	}
	var lastNamed int = -1
	for _, childIdx := range n.tree.nodes[parentIdx].children {
		if int(childIdx) == idx {
			if lastNamed >= 0 && lastNamed < len(n.tree.nodes) {
				return &Node{
					id:   unsafePointer(lastNamed + 1),
					tree: n.tree,
				}
			}
			return nil
		}
		if childIdx >= 0 && childIdx < len(n.tree.nodes) && n.tree.nodes[childIdx].isNamed {
			lastNamed = childIdx
		}
	}
	return nil
}

// ── 子节点查找 ────────────────────────────────────────

// FirstChildForByte 返回包含或起始于给定字节偏移的第一个子节点。
func (n *Node) FirstChildForByte(byte uint32) *Node {
	if n == nil || n.tree == nil {
		return nil
	}
	idx := int(n.id) - 1
	if idx < 0 || idx >= len(n.tree.nodes) {
		return nil
	}
	for _, childIdx := range n.tree.nodes[idx].children {
		if childIdx < 0 || childIdx >= len(n.tree.nodes) {
			continue
		}
		child := n.tree.nodes[childIdx]
		if child.startByte <= byte && child.endByte > byte {
			return &Node{
				id:   unsafePointer(childIdx + 1),
				tree: n.tree,
			}
		}
		if child.startByte >= byte {
			return &Node{
				id:   unsafePointer(childIdx + 1),
				tree: n.tree,
			}
		}
	}
	return nil
}

// FirstNamedChildForByte 返回包含或起始于给定字节偏移的第一个命名子节点。
func (n *Node) FirstNamedChildForByte(byte uint32) *Node {
	if n == nil || n.tree == nil {
		return nil
	}
	idx := int(n.id) - 1
	if idx < 0 || idx >= len(n.tree.nodes) {
		return nil
	}
	for _, childIdx := range n.tree.nodes[idx].children {
		if childIdx < 0 || childIdx >= len(n.tree.nodes) {
			continue
		}
		child := n.tree.nodes[childIdx]
		if !child.isNamed {
			continue
		}
		if child.startByte <= byte && child.endByte > byte {
			return &Node{
				id:   unsafePointer(childIdx + 1),
				tree: n.tree,
			}
		}
		if child.startByte >= byte {
			return &Node{
				id:   unsafePointer(childIdx + 1),
				tree: n.tree,
			}
		}
	}
	return nil
}

// ── 后裔节点查找 ──────────────────────────────────────

// DescendantForByteRange 返回覆盖给定字节范围的最小节点。
func (n *Node) DescendantForByteRange(start, end uint32) *Node {
	if n == nil || n.tree == nil {
		return nil
	}
	// 递归查找
	best := n
	for {
		found := false
		for i := uint32(0); i < best.ChildCount(); i++ {
			child := best.Child(int(i))
			if child == nil {
				continue
			}
			cs := child.StartByte()
			ce := child.EndByte()
			if cs <= start && ce >= end {
				best = child
				found = true
				break
			}
		}
		if !found {
			break
		}
	}
	return best
}

// DescendantForPointRange 返回覆盖给定行列范围的最小节点。
func (n *Node) DescendantForPointRange(start, end Point) *Node {
	if n == nil || n.tree == nil {
		return nil
	}
	best := n
	for {
		found := false
		for i := uint32(0); i < best.ChildCount(); i++ {
			child := best.Child(int(i))
			if child == nil {
				continue
			}
			sp := child.StartPoint()
			ep := child.EndPoint()
			if pointLE(sp, start) && pointGE(ep, end) {
				best = child
				found = true
				break
			}
		}
		if !found {
			break
		}
	}
	return best
}

// NamedDescendantForByteRange 返回覆盖给定字节范围的最小命名节点。
func (n *Node) NamedDescendantForByteRange(start, end uint32) *Node {
	if n == nil || n.tree == nil {
		return nil
	}
	best := n
	for {
		found := false
		for i := uint32(0); i < best.NamedChildCount(); i++ {
			child := best.NamedChild(int(i))
			if child == nil {
				continue
			}
			cs := child.StartByte()
			ce := child.EndByte()
			if cs <= start && ce >= end {
				best = child
				found = true
				break
			}
		}
		if !found {
			break
		}
	}
	return best
}

// NamedDescendantForPointRange 返回覆盖给定行列范围的最小命名节点。
func (n *Node) NamedDescendantForPointRange(start, end Point) *Node {
	if n == nil || n.tree == nil {
		return nil
	}
	best := n
	for {
		found := false
		for i := uint32(0); i < best.NamedChildCount(); i++ {
			child := best.NamedChild(int(i))
			if child == nil {
				continue
			}
			sp := child.StartPoint()
			ep := child.EndPoint()
			if pointLE(sp, start) && pointGE(ep, end) {
				best = child
				found = true
				break
			}
		}
		if !found {
			break
		}
	}
	return best
}

func pointLE(a, b Point) bool {
	return a.Row < b.Row || (a.Row == b.Row && a.Column <= b.Column)
}

func pointGE(a, b Point) bool {
	return a.Row > b.Row || (a.Row == b.Row && a.Column >= b.Column)
}

// ── 其他 ──────────────────────────────────────────────

// ChildWithDescendant 返回指定后裔节点的直接子祖先。
func (n *Node) ChildWithDescendant(descendant *Node) *Node {
	if n == nil || descendant == nil || n.tree == nil {
		return nil
	}
	if n.id == descendant.id {
		return n
	}
	// 沿祖先链上溯，找到 n 的子节点
	path := []*Node{descendant}
	for current := descendant.Parent(); current != nil && current.id != n.id; current = current.Parent() {
		path = append(path, current)
	}
	if len(path) == 0 {
		return nil
	}
	return path[len(path)-1]
}

// Equal 检查两个节点是否相同。
func (n *Node) Equal(other *Node) bool {
	if n == nil || other == nil {
		return n == other
	}
	return n.id == other.id && n.tree == other.tree
}

// ── 解析状态 ──────────────────────────────────────────

// ParseState 返回此节点的解析状态 ID。
func (n *Node) ParseState() StateId {
	return 0 // Go 版简化
}

// NextParseState 返回此节点之后的解析状态 ID。
func (n *Node) NextParseState() StateId {
	return 0 // Go 版简化
}

// ── 辅助 ──────────────────────────────────────────────

// Content 返回节点的源码文本。
func (n *Node) Content(source []byte) string {
	if n == nil || source == nil {
		return ""
	}
	start := n.StartByte()
	end := n.EndByte()
	if start >= uint32(len(source)) || end > uint32(len(source)) || start > end {
		return ""
	}
	return string(source[start:end])
}
