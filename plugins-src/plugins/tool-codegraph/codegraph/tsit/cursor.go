package tsit

// cursor.go — 翻译 tree-sitter tree_cursor.c，实现高效树遍历游标。

// TreeCursor 提供在语法树中高效遍历的能力。
// 比通过 Node.Parent/Child/NextSibling 更高效。
type TreeCursor struct {
	stack []cursorEntry
	tree  *Tree
}

type cursorEntry struct {
	node     *Node
	childIdx int // 当前子节点索引
	depth    int
}

// NewTreeCursor 从给定节点创建游标。
func NewTreeCursor(node *Node) *TreeCursor {
	if node == nil || node.tree == nil {
		return &TreeCursor{}
	}
	return &TreeCursor{
		stack: []cursorEntry{{node: node, childIdx: -1, depth: 0}},
		tree:  node.tree,
	}
}

// CurrentNode 返回游标当前指向的节点。
func (c *TreeCursor) CurrentNode() *Node {
	if len(c.stack) == 0 {
		return nil
	}
	return c.stack[len(c.stack)-1].node
}

// CurrentFieldName 返回当前节点的字段名。
func (c *TreeCursor) CurrentFieldName() string {
	node := c.CurrentNode()
	if node == nil || node.tree == nil || node.tree.language == nil {
		return ""
	}
	entry := c.stack[len(c.stack)-1]
	return node.tree.language.FieldNameForChild(entry.node, entry.childIdx)
}

// GotoParent 移动到父节点。
func (c *TreeCursor) GotoParent() bool {
	if len(c.stack) <= 1 {
		return false
	}
	c.stack = c.stack[:len(c.stack)-1]
	return true
}

// GotoNextSibling 移动到下一个兄弟节点。
func (c *TreeCursor) GotoNextSibling() bool {
	if len(c.stack) == 0 {
		return false
	}
	entry := &c.stack[len(c.stack)-1]
	parent := entry.node.Parent()
	if parent == nil {
		return false
	}
	next := entry.node.NextSibling()
	if next == nil {
		return false
	}
	entry.node = next
	entry.childIdx++
	return true
}

// GotoFirstChild 移动到第一个子节点。
func (c *TreeCursor) GotoFirstChild() bool {
	node := c.CurrentNode()
	if node == nil {
		return false
	}
	child := node.Child(0)
	if child == nil {
		return false
	}
	c.stack = append(c.stack, cursorEntry{
		node:     child,
		childIdx: 0,
		depth:    len(c.stack),
	})
	return true
}

// Reset 重置游标到新节点。
func (c *TreeCursor) Reset(node *Node) {
	c.stack = []cursorEntry{{node: node, childIdx: -1, depth: 0}}
	if node != nil {
		c.tree = node.tree
	}
}

// CurrentDepth 返回当前节点相对于根节点的深度。
func (c *TreeCursor) CurrentDepth() uint32 {
	return uint32(len(c.stack) - 1)
}
