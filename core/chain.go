package core

import "sync"

type BlockTreeNode struct {
	BlockHash *Uint256
	Height uint32
	Bits uint32
	Timestamp uint32
	ParentHash *Uint256
	Parent *BlockTreeNode
	Children []*BlockTreeNode
}



type Chain struct {
	Blocks *BlockDB
	UnSpent UnspentDB

	BlockTreeRoot *BlockTreeNode
	BlockTreeEnd *BlockTreeNode
	Genesis *Uint256

	BlockIndexAccess sync.Mutex
	BlockIndex map[[Uint256IdxLen]byte] *BlockTreeNode

	DoNotSync bool
}


func NewChain(dbRootDir string, genesis *Uint256, rescan bool) (ch *Chain) {
	testnet = genesis.Hash[0] == 0x43

	ch = new(Chain)
	ch.Genesis = genesis
	ch.Blocks = NewBlockDB(dbRootDir)
	ch.UnSpent = NewUnspentDb(dbRootDir, rescan)

	ch.loadBlockIndex()


}


func (ch *Chain) loadBlockIndex() {
	ch.BlockIndex = make(map[[Uint256IdxLen]byte]*BlockTreeNode, BlockMapInitLen)
	ch.BlockTreeRoot = new(BlockTreeNode)
	ch.BlockTreeRoot.BlockHash = ch.Genesis
	ch.BlockTreeRoot.Bits = n
}