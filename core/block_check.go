package core

import (
	"errors"
	"time"
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
		e = errors.New("CheckBlock: "+bl.Hash.String()+" parent not found")
		maylater = true
		return
	}

	// 工作量证明检验
	gnwr := GetNwe

}
