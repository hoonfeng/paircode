package tsit

// tree.go — 翻译 tree-sitter tree.c，实现语法树管理。

// Tree 是语法树，由 Parser 生成，通过 Node 访问。
// 存储 internalNode 数组，每个 Node 持有指向 Tree 的引用。
type Tree struct {
	root     *Node
	language Language
	nodes    []internalNode
	ranges   []Range
}

// RootNode 返回语法树的根节点。
func (t *Tree) RootNode() *Node {
	return t.root
}

// Language 返回解析此树使用的语言。
func (t *Tree) Language() Language {
	return t.language
}

// Copy 创建语法树的浅拷贝（线程安全）。
func (t *Tree) Copy() *Tree {
	if t == nil {
		return nil
	}
	nodes := make([]internalNode, len(t.nodes))
	copy(nodes, t.nodes)
	return &Tree{
		root:     t.root,
		language: t.language,
		nodes:    nodes,
		ranges:   t.ranges,
	}
}

// Edit 编辑语法树以匹配源码变更。
func (t *Tree) Edit(edit *InputEdit) {
	if t == nil || edit == nil {
		return
	}
	// 遍历所有节点，调整位置
	for i := range t.nodes {
		n := &t.nodes[i]

		// 如果节点在编辑点之前，无需调整
		if n.endByte <= edit.StartByte {
			continue
		}

		// 如果节点在编辑点之后
		if n.startByte >= edit.StartByte {
			diff := int32(edit.NewEndByte) - int32(edit.OldEndByte)
			if diff > 0 {
				n.startByte += uint32(diff)
				n.endByte += uint32(diff)
			} else if diff < 0 {
				n.startByte -= uint32(-diff)
				n.endByte -= uint32(-diff)
			}
			// 调整行列位置
			n.startPoint = adjustPoint(n.startPoint, edit)
			n.endPoint = adjustPoint(n.endPoint, edit)
		}
	}

	// 更新根节点上下文
	if t.root != nil {
		t.root.context[0] = t.nodes[0].startByte
		t.root.context[1] = t.nodes[0].endByte
	}
}

func adjustPoint(p Point, edit *InputEdit) Point {
	if edit.StartPoint.Row < p.Row || (edit.StartPoint.Row == p.Row && edit.StartPoint.Column <= p.Column) {
		// 节点在编辑点之后，偏移
		rowDiff := int32(edit.NewEndPoint.Row) - int32(edit.OldEndPoint.Row)
		colDiff := int32(edit.NewEndPoint.Column) - int32(edit.OldEndPoint.Column)
		if rowDiff > 0 || (rowDiff == 0 && colDiff > 0) {
			p.Row += uint32(rowDiff)
			if rowDiff == 0 {
				p.Column += uint32(colDiff)
			}
		}
	}
	return p
}

// NodeCount 返回树中的节点总数。
func (t *Tree) NodeCount() int {
	if t == nil {
		return 0
	}
	return len(t.nodes)
}
