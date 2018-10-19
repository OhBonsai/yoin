package core

import "sync"


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
	ch.UnSpent = NewUnspentDB(dbRootDir, rescan)
	ch.loadBlockIndex()

	if rescan {
		ch.BlockTreeEnd = ch.BlockTreeRoot
	} else {
		for i:=0; i<3 && ch.BlockTreeEnd.Height>0; i++ {
			ch.UnSpent.UndoBlockTransactions(ch.BlockTreeEnd.Height)
			ch.BlockTreeEnd = ch.BlockTreeEnd.Parent
		}
	}

	end, _ := ch.BlockTreeRoot.FindFarthestNode()
	if end.Height > ch.BlockTreeEnd.Height {
		ch.ParseTillBlock(end)
	}

	return
}

func NewBlockIndex(h []byte) (o [Uint256IdxLen]byte) {
	copy(o[:], h[:Uint256IdxLen])
	return
}



func (ch *Chain) loadBlockIndex() {
	ch.BlockIndex = make(map[[Uint256IdxLen]byte]*BlockTreeNode, BlockMapInitLen)
	ch.BlockTreeRoot = new(BlockTreeNode)
	ch.BlockTreeRoot.BlockHash = ch.Genesis
	ch.BlockTreeRoot.Bits = nProofOfWorkLimit
	ch.BlockIndex[NewBlockIndex(ch.Genesis.Hash[:])] = ch.BlockTreeRoot

}