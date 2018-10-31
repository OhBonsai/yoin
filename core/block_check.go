package core

import (
	"errors"
	"time"
	"bytes"
)

func (ch *Chain) CheckBlock(b *Block) (
	e error,
	dos bool,       //假的，你不是个好区块
	maylater bool,  //你是不是比你爹来的早了点
) {
	// 大小限制
	if len(b.Raw) < 81 || len(b.Raw) > 1e6 {
		e = errors.New("CheckBlock() : size limits failed")
		dos = true
		return
	}

	// 创建时间小于二分钟
	if int64(b.CreateTime) > time.Now().Unix() + 2 * 60 * 60 {
		e = errors.New("CheckBlock() : block timestamp too far in the future")
		dos = true
		return
	}

  	// 链上有没有这个区块
	if prv, has := ch.BlockIndex[b.Hash.BIdx()]; has {
		if prv.Parent == nil {
			// This is genesis block
			prv.Timestamp = b.CreateTime
			prv.Bits = b.DifficultyTarget
			e = errors.New("CheckBlock: Genesis bock")
			return
		} else {
			e = errors.New("CheckBlock: "+b.Hash.String()+" already in")
			return
		}
	}

	prevBlock, ok := ch.BlockIndex[NewUint256(b.PreBlockHash).BIdx()]
	if !ok {
		// 找不到你爹
		e = errors.New("CheckBlock: "+b.Hash.String()+" parent not found")
		maylater = true
		return
	}

	// 工作量证明检验
	gnwr := GetNextWorkRequired(prevBlock, b.CreateTime)
	if b.DifficultyTarget != gnwr {
		println("AcceptBlock() : incorrect proof of work ", b.DifficultyTarget," at block", prevBlock.Height+1,
			" exp:", gnwr)
		if !testnet || ((prevBlock.Height+1)%2016)!=0 {
			e = errors.New("CheckBlock: incorrect proof of work")
			dos = true
			return
		}
	}

	// 把交易都反序列化出来
	e = b.DecodeTxListFromRaw()
	if e != nil {
		dos = true
		return
	}

	if !b.Trusted {
		// 首先你得有个奖励
		if len(b.Txs) == 0 || !b.Txs[0].isCoinBase() {
			e = errors.New("CheckBlock() : first tx is not coinbase: "+b.Hash.String())
			dos = true
			return
		}
		// 其次你只能有一个Coinbase
		for i:=1; i<len(b.Txs); i++ {
			if b.Txs[i].isCoinBase() {
				e = errors.New("CheckBlock() : more than one coinbase")
				dos = true
				return
			}
		}

		// 检查梅克尔根
		if !bytes.Equal(getMerkel(b.Txs), b.MerkleRoot) {
			e = errors.New("CheckBlock() : Merkle Root mismatch")
			dos = true
			return
		}

		// 每个交易都检查一下
		for i:=0; i<len(b.Txs); i++ {
			e = b.Txs[i].CheckTransaction()
			if e!=nil {
				e = errors.New("CheckBlock() : CheckTransaction failed\n"+e.Error())
				dos = true
				return
			}
		}
	}

	return
}
