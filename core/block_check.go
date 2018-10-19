package core

import (
	"errors"
	"time"
)

func (ch *Chain) CheckBlock(b *Block) (e error, dos bool, maylater bool) {
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

	//

}
