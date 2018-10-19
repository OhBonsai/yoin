package core


type BlockTreeNode struct {
	BlockHash *Uint256
	Height uint32
	Bits uint32
	Timestamp uint32
	ParentHash *Uint256
	Parent *BlockTreeNode
	Children []*BlockTreeNode  //链可以分岔的
}


// 从某个区块找到到最远的区块
func (n *BlockTreeNode) FindFarthestNode() (*BlockTreeNode, int) {
	if len(n.Children) == 0{
		return n, 0
	}

	res, depth := n.Children[0].FindFarthestNode()
	if len(n.Children) > 1 {
		for i:=1; i<len(n.Children); i++ {
			tRes, tDept := n.Children[i].FindFarthestNode()
			if tDept > depth {
				res, depth = tRes, tDept
			}
		}
	}
	return res, depth+1
}


// 找到指向某个节点的女儿节点。分岔的地方
func (n *BlockTreeNode) FindNextNodePathTo(to *BlockTreeNode) (*BlockTreeNode) {
	if to.Height <= n.Height {
		panic("End block is not higher then current")
	}

	if len(n.Children) == 0 { // 没儿子，怎么找到
		panic("Unknown path to block " + to.BlockHash.String() )
	}

	if n == to {  // 自己找自己，狗日的
		return nil
	}

	if len(n.Children) == 1 {
		return n.Children[0] // 只有一个女儿
	}

	// 从后往前走
	for {
		if to.Parent == n {
			return to
		}

		to = to.Parent
	}

	return nil
}


// 加女儿
func (n *BlockTreeNode) addChild(c *BlockTreeNode) {
	n.Children = append(n.Children, c)
}


// 打掉儿子
func (n *BlockTreeNode) delChild(c *BlockTreeNode) {
	tmpChildren := make([]*BlockTreeNode, len(n.Children) - 1)
	cursor := 0

	for _, child := range n.Children {
		if child != c {
			tmpChildren[cursor] = child
			cursor++
		}
	}

	if cursor != len(n.Children) - 1{
		panic("Child not found")
	}

	n.Children = tmpChildren
}