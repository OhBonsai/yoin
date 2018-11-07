package core

import (
	"fmt"
	"errors"
	"encoding/hex"
)

func (ch *Chain) ProcessBlockTransaction(bl *Block, height uint32) (changes *BlockChangeEvent, e error) {
	changes = new(BlockChangeEvent)
	changes.Height = height
	changes.DeletedTxs = make(map[TxPrevOut]*TxOut)
	changes.AddedTxs = make(map[TxPrevOut]*TxOut)

	e = ch.commitTxs(bl, changes)
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
	sumblockout := uint64(0)

	if don(DBG_TX) {
		fmt.Printf("Commiting %d traactions\n", len(bl.Txs))
	}

	// 将所有TXOUT都放到一个字典中
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
			fmt.Printf("tx %d/%d:]n", i+1, len(bl.Txs))
		}

		if i > 0 {
			scriptsOk := true
			for j:=0; j<useRoutines; j++ {
				taskDone <- true
			}

			for j:=0; j<len(bl.Txs[i].TxIns); j++ {
				inp := &bl.Txs[i].TxIns[j].Input
				if _, ok := changes.DeletedTxs[*inp]; ok {
					println("txin", inp.String(), "already spent in this block")
					e = errors.New("Input spent more then once in same block")
					break
				}

				if don(DBG_TX) {
					idx := getUnspIndex(inp)
					println(" idx", hex.EncodeToString(idx[:]))
				}

				tout := ch.PickUnspent(inp)

				if tout == nil {
					if don(DBG_TX) {
						println("PickUnspent failed")
					}
					t, ok := blUnsp[inp.PreOutTxHash]
					if !ok {
						e = errors.New("Unknown input")
						break
					}

					if inp.OutIdxInTx >= uint32(len(t)) {
						println("Vout too big", len(t), inp.String())
						e = errors.New("Vout too big")
						break
					}

					if t[inp.OutIdxInTx] == nil {
						println("Vout already spent", inp.String())
						e = errors.New("Vout already spent")
						break
					}

					tout = t[inp.OutIdxInTx]
					t[inp.OutIdxInTx] = nil
				}else{
					if don(DBG_TX) {
						println("PickUnspent OK")
					}
				}

				if !(<-taskDone) {
					println("VerifyScript error 1")
					scriptsOk = false
					break
				}

				if bl.Trusted {
					taskDone <- true
				} else {
					go func(i, j int, pks []byte) {
						taskDone <- VerifyTxScript(bl.Txs[i].TxIns[j].ScriptSig,
							pks, j, bl.Txs[i])
					}(i, j, tout.ScriptPubKey)
				}

				txinsum += tout.Value
				changes.DeletedTxs[*inp] = tout

				if don(DBG_TX) {
					fmt.Printf("  in %d: %.8f BTC @ %s\n", j+1, float64(tout.Value)/1e8,
						bl.Txs[i].TxIns[j].Input.String())
				}
			}

			if scriptsOk {
				scriptsOk = <- taskDone
			}

			for j:=1; j<useRoutines; j++ {
				if !(<-taskDone) {
					println("VerifyScript error 2")
					scriptsOk = false
				}
			}

			if !scriptsOk {
				return errors.New("VerifyScripts failed")
			}
		} else {
			if don(DBG_TX) {
				fmt.Printf("  mined %.8f\n", float64(sumblockin)/1e8)
			}
		}
		sumblockin += txinsum

		for j := range bl.Txs[i].TxOuts {
			if don(DBG_TX) {
				fmt.Printf("  out %d: %12.8f\n", j+1, float64(bl.Txs[i].TxOuts[j].Value)/1e8)
			}
			txoutsum += bl.Txs[i].TxOuts[j].Value
			txa := new(TxPrevOut)
			copy(txa.PreOutTxHash[:], bl.Txs[i].Hash.Hash[:])
			txa.OutIdxInTx = uint32(j)
			_, spent := changes.DeletedTxs[*txa]
			if spent {
				delete(changes.DeletedTxs, *txa)
			} else {
				changes.AddedTxs[*txa] = bl.Txs[i].TxOuts[j]
			}
		}
		sumblockout += txoutsum

		if don(DBG_TX) {
			fmt.Sprintf("  %12.8f -> %12.8f  (%.8f)\n",
				float64(txinsum)/1e8, float64(txoutsum)/1e8,
				float64(txinsum-txoutsum)/1e8)
		}

		if don(DBG_TX) && i>0 {
			fmt.Printf(" fee : %.8f\n", float64(txinsum-txoutsum)/1e8)
		}
		if i>0 && txoutsum > txinsum {
			return errors.New("More spent than at the input")
		}
		if e != nil {
			break // If any input fails, do not continue
		}
	}

	if sumblockin < sumblockout {
		return errors.New(fmt.Sprintf("Out:%d > In:%d", sumblockout, sumblockin))
	} else if don(DBG_WASTED) && sumblockin != sumblockout {
		fmt.Printf("%.8f BTC wasted in block %d\n", float64(sumblockin-sumblockout)/1e8, changes.Height)
	}

	return nil
}