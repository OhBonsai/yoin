package core


type BlockTreeNode struct {
	BlockHash *Uint256
	Height uint32
	Bits uint32
	Timestamp uint32
	ParentHash *Uint256
	Parent *BlockTreeNode
	Children []*BlockTreeNode
}

