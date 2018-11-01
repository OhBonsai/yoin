package core


func nextBlock(ch *Chain, hash, prev []byte, height, bits, timestamp uint32) {
	bh := NewUint256(hash[:])

	// 如果已经在字典中存了的话。
	if have, ok := ch.BlockIndex[bh.BIdx()]; ok {
		println("nextBlock:", bh.String(), "- already in")
		have.Bits = bits
		have.Timestamp = timestamp
		return
	}

	v := new(BlockTreeNode)
	v.BlockHash = bh
	v.ParentHash = NewUint256(prev[:])
	v.Height = height
	v.Bits = bits
	v.Timestamp = timestamp
	ch.BlockIndex[v.BlockHash.BIdx()] = v
}


// 从磁盘中获取索引
func (ch *Chain)loadBlockIndex() {
	ch.BlockIndex = make(map[[Uint256IdxLen]byte]*BlockTreeNode, BlockMapInitLen)
	ch.BlockTreeRoot = new(BlockTreeNode)
	ch.BlockTreeRoot.BlockHash = ch.Genesis
	ch.BlockTreeRoot.Bits = nProofOfWorkLimit
	ch.BlockIndex[NewBlockIndex(ch.Genesis.Hash[:])] = ch.BlockTreeRoot

	ch.Blocks.InitBlockIdxMap(ch, nextBlock)
	tlb := ch.Unspent.GetLastBlockHash()

	for _, v := range ch.BlockIndex {
		if v==ch.BlockTreeRoot {
			// skip root block (should be only one)
			continue
		}
		par, ok := ch.BlockIndex[v.ParentHash.BIdx()]
		if !ok {
			panic(v.BlockHash.String()+" has no Parent "+v.ParentHash.String())
		}
		/*if par.Height+1 != v.Height {
			panic("height mismatch")
		}*/
		v.Parent = par
		v.Parent.addChild(v)
		v.ParentHash = nil // we wont need this anymore
	}
	if tlb == nil {
		//println("No last block - full rescan will be needed")
		ch.BlockTreeEnd = ch.BlockTreeRoot
		return
	} else {
		var ok bool
		ch.BlockTreeEnd, ok = ch.BlockIndex[NewUint256(tlb).BIdx()]
		if !ok {
			panic("Last Block Hash not found")
		}
	}

}