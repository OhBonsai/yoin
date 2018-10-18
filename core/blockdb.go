package core

import (
	"os"
	"sync"
	"encoding/binary"
	"fmt"
)

const (
	BLOCK_TRUSTED = 0x01
	BLOCK_INVALID = 0x02
)

/*
	有两个文件，一个区块文件， 一个是区块索引文件
	blockchain.dat - 区块文件
	blockchain.idx - 索引文件
		[0] - flags:
			bit(0) - "trusted" flag - this block's scripts have been verified
			bit(1) - "invalid" flag - this block's scripts have failed
		[1:3] - reserved
		[4:36]  - 256-bit block hash
		[36:68] - 256-bit block's Parent hash
		[68:72] - 32-bit block height (genesis is 0)
		[72:76] - 32-bit block's timestamp
		[76:80] - 32-bit block's bits
		[80:88] - 64-bit block pos in block chain.dat file
		[88:92] - 32-bit block length in bytes
 */

type oneBlockIdx struct {
	fPos uint64 // 该区块在区块文件中哪个位置
	bLen uint32 // 区块长度

	iPos int64  // 该索引在索引的哪个位置
	trusted bool // 区块是否可信
}


type BlockDB struct {
	dirName string // 文件所在文件夹
	blockIdxMap map[[Uint256IdxLen]byte] * oneBlockIdx // 区块内存hash表。这里用48bit作为key, 不好发生碰撞了把~

	dataFile *os.File
	idxFile *os.File

	mutex sync.Mutex // 读写锁
}


//  创建一个blockDB对象，并且初始化
func newBlockDB(dirName string) (db *BlockDB) {
	db = new(BlockDB)
	db.dirName = dirName
	db.blockIdxMap = make(map[[Uint256IdxLen]byte] *oneBlockIdx)
	error := os.MkdirAll(db.dirName, 0770)
	if error != nil {
		panic("wrong dir")
	}

	db.dataFile, _ = os.OpenFile(db.dirName+"blockchain.dat", os.O_RDWR|os.O_CREATE, 0660)
	if db.dataFile == nil {
		panic("Cannot open blockchain.dat")
	}

	db.idxFile, _ = os.OpenFile(db.dirName+"blockchain.idx", os.O_RDWR|os.O_CREATE, 0660)
	if db.idxFile == nil {
		panic("Cannot open blockchain.idx")
	}
	return
}

// 查看数据库中有多少区块
func (db *BlockDB) GetStats() (s string) {
	db.mutex.Lock()
	s +=  fmt.Sprintf("BlockDB: %d blocks\n", len(db.blockIdxMap))
	db.mutex.Unlock()
	return
}

// 取Hash值的Uint256IdxLen（6）作为索引key
func hash2idx(h []byte) (idx [Uint256IdxLen]byte)  {
	copy(idx[:], h[:Uint256IdxLen])
	return
}

// 增加一个区块
func (db *BlockDB) AddBlock(height uint32, b *Block) (e error) {
	var pos int64
	var flagz [4]byte  // 后面三位预留出来的

	pos, e = db.dataFile.Seek(0, os.SEEK_END) // 文件尾部
	if e != nil {
		panic(e.Error())
	}

	_, e = db.dataFile.Write(b.Raw[:])
	if e != nil {
		panic(e.Error())
	}

	iPos, _ := db.idxFile.Seek(0, os.SEEK_CUR) // 不断的写，cur=end

	if b.Trusted {
		flagz[0] |= BLOCK_INVALID
	}

	db.idxFile.Write(flagz[:])
	db.idxFile.Write(b.Hash.Hash[0:32])
	db.idxFile.Write(b.Raw[4:36])
	binary.Write(db.idxFile, binary.LittleEndian, uint32(height))
	binary.Write(db.idxFile, binary.LittleEndian, uint32(b.CreateTime))
	binary.Write(db.idxFile, binary.LittleEndian, uint32(b.DifficultyTarget))
	binary.Write(db.idxFile, binary.LittleEndian, uint64(pos))
	binary.Write(db.idxFile, binary.LittleEndian, uint32(len(b.Raw[:])))

	db.mutex.Lock()
	db.blockIdxMap[hash2idx(b.Hash.Hash[:])] = &oneBlockIdx{
		fPos:uint64(pos),
		bLen:uint32(len(b.Raw[:])),
		iPos:iPos,
		trusted:b.Trusted,
	}
	db.mutex.Unlock()
	return
}

