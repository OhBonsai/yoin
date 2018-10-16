package core

import (
	"errors"
	"encoding/binary"
)


type Block struct {
	Version          uint32
	PreBlockHash     []byte     //  之前区块的hash
	MerkleRoot       []byte     //  交易的梅克尔树根节点
	CreateTime       uint32     //  取快创建时间

	DifficultyTarget uint32     // 难度系数
	Nonce            uint32     // 答案


	Txs              []*Tx

	Raw              []byte
	Hash             *Uint256   //  自己的Hash
	Trusted          bool
}


func DeserializeBlock(data []byte) (*Block, error){
	if len(data) < 81 {
		return nil,  errors.New("Block binary is too short...")
	}

	var b Block

	b.Hash = NewSha2Hash(data[:80])
	b.Raw = data

	b.Version = binary.LittleEndian.Uint32(data[0:4])
	b.PreBlockHash = data[4:36]
	b.MerkleRoot = data[36:68]
	b.CreateTime = binary.LittleEndian.Uint32(data[68:72])
	b.DifficultyTarget = binary.LittleEndian.Uint32(data[72:76])
	b.Nonce = binary.LittleEndian.Uint32(data[76:80])
	return &b, nil
}

func GetBlockReward(height uint32) (uint64) {
	return 50e8 >> (height / 210000)
}


func (b *Block) DecodeTxListFromRaw() (e error){
	offs := int(80)
	txCount, n := LoadVarLen(b.Raw[offs:])
	offs += n
	b.Txs = make([]*Tx, txCount)

	for i:=0; i< useRoutines; i++ {
		taskDone <- false
	}

	for i :=0; i < int(txCount); i++ {
		_ = <-taskDone // wait if we have too many goroutine already
		b.Txs[i], n = DeserializeTx(b.Raw[offs:])
		b.Txs[i].Size = uint32(n)
		go func(h **Uint256, b[]byte) {
			*h = NewSha2Hash(b)
			taskDone <- true
		}(&b.Txs[i].Hash, b.Raw[offs:offs+n])
		offs += n
	}

	for i:=0; i<useRoutines; i++ {
		_ = <-taskDone
	}
	return
}
