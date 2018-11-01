package core

import "fmt"

func (ch *Chain) ProcessBlockTransaction(bl *Block, height uint32) (changes *BlockChangeEvent, e error) {
	changes = new(BlockChangeEvent)
	changes.Height = height
	changes.DeletedTxs = make(map[TxPrevOut]*TxOut)
	changes.AddedTxs = make(map[TxPrevOut]*TxOut)

	e = ch.commitTxs(bl. changes)
	return
}



func (ch *Chain) AcceptBlock(bl *Block) (e error) {
	prevBlk, ok := ch.BlockIndex[NewUint256(bl.PreBlockHash).BIdx()]
	if !ok {
		panic("THis should not happen")
	}


	// 生成一个节点
	cur := new(BlockTreeNode)
	cur.BlockHash = bl.Hash
	cur.Parent = prevBlk
	cur.Height = prevBlk.Height + 1
	cur.Bits = bl.DifficultyTarget
	cur.Timestamp = bl.CreateTime

	// 将此节点添加到树中
	ch.BlockIndexAccess.Lock()
	prevBlk.addChild(cur)
	ch.BlockIndex[cur.BlockHash.BIdx()] = cur
	ch.BlockIndexAccess.Unlock()


	// 如何接收到得块，刚好是当前最长链得儿子
	if ch.BlockTreeEnd == prevBlk {

		// 最长连
		if don(DBG_BLOCKS) {
			fmt.Printf("Adding block %s @ %d\n", cur.BlockHash.String(), cur.Height)
		}

		var changes *BlockChangeEvent
		changes, e = ch.ProcessBlockTransaction(bl, cur.Height)
		if e != nil {
			println("ProcessBlockTransactions", cur.BlockHash.String(), cur.Height, e.Error())
			ch.BlockIndexAccess.Lock()
			cur.Parent.delChild(cur)
			delete(ch.BlockIndex, cur.BlockHash.BIdx())
			ch.BlockIndexAccess.Unlock()
		} else {
			bl.Trusted = true
			ch.Blocks.AddBlock(cur.Height, bl)
			ch.Unspent.CommitBlockTxs(changes, bl.Hash.Hash[:])

			if !ch.DoNotSync {
				ch.Blocks.Sync()
				ch.Unspent.Sync()
			}

			ch.BlockTreeEnd = cur
		}
	} else {
		// 如果不是在最长连
		ch.Blocks.AddBlock(cur.Height, bl)
		if don(DBG_BLOCKS|DBG_ORPHAS) {
			fmt.Printf("Orphan block %s @ %d\n", cur.BlockHash.String(), cur.Height)
		}

		if cur.Height > ch.BlockTreeEnd.Height {
			ch.MoveToBlock(cur)
		}
	}

	return
}


func verify(sig []byte, prv []byte, i int, tx *Tx) {
	taskDone <- VerifyTxScript(sig, prv, i, tx)
}


func getUnspIndex(po *TxPrevOut) (idx [8]byte) {
	copy(idx[:], po.PreOutTxHash[:8])
	idx[0] ^= byte(po.OutIdxInTx)
	idx[1] ^= byte(po.OutIdxInTx>>8)
	idx[2] ^= byte(po.OutIdxInTx>>16)
	idx[3] ^= byte(po.OutIdxInTx>>32)
	return
}


func (ch *Chain) commitTxs (bl *Block, changes *BlockChangeEvent) (e error) {
	sumblockin := GetBlockReward(changes.Height)
	sublockout := uint64(0)

	if don(DBG_TX) {
		fmt.Printf("Commiting %d traactions\n", len(bl.Txs))
	}


	blUnsp := make(map[[32]byte] []*TxOut, len(bl.Txs))
	for i := range bl.Txs {
		outs := make([]*TxOut, len(bl.Txs[i].TxOuts))
		for j := range bl.Txs[i].TxOuts {
			bl.Txs[i].TxOuts[j].BlockHeight = changes.Height
			outs[j] = bl.Txs[i].TxOuts[j]
		}
		blUnsp[bl.Txs[i].Hash.Hash] = outs
	}


	for i := range bl.Txs {
		var txoutsum, txinsum uint64
		if don(DBG_TX) {
			fmt.Printf("tx %d/%d: \n", i+1, len(bl.Txs))
		}
	}
}