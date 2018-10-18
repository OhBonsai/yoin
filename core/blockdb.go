package core

import (
	"os"
	"sync"
	"encoding/binary"
	"fmt"
	"errors"
)

const (
	BLOCK_TRUSTED = 0x01
	BLOCK_INVALID = 0x02
	IDX_DATA_SIZE = 92
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
		[76:80] - 32-bit block's bits  //TODO 答案还是难度？
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
	//TODO golang map key的长度会影响性能吗？
	blockIdxMap map[[Uint256IdxLen]byte] * oneBlockIdx // 区块内存hash表。这里用48bit作为key, 不好发生碰撞了把~

	dataFile *os.File
	idxFile *os.File

	mutex sync.Mutex // 读写锁
}


//  创建一个blockDB对象，并且初始化
func NewBlockDB(dirName string) (db *BlockDB) {
	db = new(BlockDB)
	db.dirName = dirName
	db.blockIdxMap = make(map[[Uint256IdxLen]byte] *oneBlockIdx)
	err := os.MkdirAll(db.dirName, 0770)
	if err != nil {
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
	var flagZ [4]byte // 后面三位预留出来的

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
		flagZ[0] |= BLOCK_INVALID
	}

	db.idxFile.Write(flagZ[:])
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

// 设置某个区块失效（在分支，该分支不是最长的那条）
func (db *BlockDB) InvalidBlock(hash []byte) {
	idx := hash2idx(hash[:])
	db.mutex.Lock()

	curBlockIndex, ok := db.blockIdxMap[idx]
	if !ok {
		db.mutex.Unlock()
		println("No such block")
		return
	}
    // 将该区块设置为失效
	println("mark", NewUint256(hash).String(), "as invalid")
	if curBlockIndex.trusted {
		panic("if block is trusted, how can be invalid")
	}
	db.setBlockFlag(curBlockIndex, BLOCK_INVALID)
	// 在map中删除该块的kv
	delete(db.blockIdxMap, idx)
	db.mutex.Unlock()
}

// 设置某个区块可靠
func (db *BlockDB) TrustBlock(hash []byte) {
	idx := hash2idx(hash[:])
	db.mutex.Lock()

	curBlockIndex, ok := db.blockIdxMap[idx]
	if !ok {
		db.mutex.Unlock()
		println("No such block")
		return
	}

	if !curBlockIndex.trusted {
		println("mark", NewUint256(hash).String(), "as trusted")
		db.setBlockFlag(curBlockIndex, BLOCK_TRUSTED)
	}
	db.mutex.Unlock()
}

// 在索引文件中修改某个区块是否可信
func (db *BlockDB) setBlockFlag(cur *oneBlockIdx, v byte){
	var b [1]byte

	// 现存储当前文件读到哪里了
	cur.trusted = true
	cpos, _ := db.idxFile.Seek(0, os.SEEK_CUR)

	// 读可信flag的哪个字节
	db.idxFile.ReadAt(b[:], cur.iPos)
	// 修改值
	b[0] |= v
	// 在那个字节写
	db.idxFile.WriteAt(b[:], cur.iPos)
	// 恢复文件读写位置
	db.idxFile.Seek(cpos, os.SEEK_SET)
}


// 一般来说对文件的写操作只更新page cache. 操作系统会找时间写到内存中。如果断电了数据就丢失了。系统提供了
// int fsync(int fd); 来保证写进硬盘了
func (db *BlockDB) Sync() {
	db.idxFile.Sync()
	db.dataFile.Sync()
}

func (db *BlockDB) Close() {
	db.idxFile.Close()
	db.dataFile.Close()
}

// 获取某个区块的二进制
func (db *BlockDB) GetBlock(hash *Uint256) (blockRaw []byte, trusted bool, e error) {
	db.mutex.Lock()
	blockIdx, ok := db.blockIdxMap[hash2idx(hash.Hash[:])]
	db.mutex.Unlock()
	if !ok {
		e = errors.New("Block not in index") // TODO 万一是HASH碰撞了怎么办？
		return
	}

	blockRaw = make([]byte, blockIdx.bLen)
	db.dataFile.Seek(int64(blockIdx.fPos), os.SEEK_SET)
	db.dataFile.Read(blockRaw[:])

	trusted = blockIdx.trusted
	return
}

// 初始化blockIdxMap
func (db *BlockDB) InitBlockIdxMap(
	ch *Chain,
	walk func(ch *Chain, hash, prv []byte, h,target, tim uint32),
)(e error){
	var b [IDX_DATA_SIZE]byte
	var maxFilePos int64

	tPos, _ := db.idxFile.Seek(0, os.SEEK_SET)
	for {
		_, e := db.idxFile.Read(b[:])
		if e != nil {
			break // 读不出来，估摸着读到底了
		}

		if (b[0]&BLOCK_INVALID) != 0 {  // 不可靠的在文件中还存着，读出来就丢弃
			println("Block #", binary.LittleEndian.Uint32(b[68:72]), "is invalid", b[0])
			continue
		}

		trusted := (b[0]&BLOCK_TRUSTED) != 0  // 为什么用与或判断，牛逼啊，CPU支持的厉害呀
		blockHash := b[4:36]
		blockPreHash := b[36:68]
		height := binary.LittleEndian.Uint32(b[68:72])
		timestamp := binary.LittleEndian.Uint32(b[72:76])
		bits := binary.LittleEndian.Uint32(b[76:80])
		filepos := binary.LittleEndian.Uint64(b[80:88])
		bocklen := binary.LittleEndian.Uint32(b[88:92])
		if int64(filepos)+int64(bocklen) > maxFilePos {
			maxFilePos = int64(filepos)+int64(bocklen)  //记录一下现在多少byte了
		}

		db.blockIdxMap[hash2idx(blockHash)] = &oneBlockIdx{
			filepos,
			bocklen,
			tPos,
			trusted,
		}

		walk(ch, blockHash, blockPreHash, height, bits, timestamp)
		tPos += 92
	}

	// 读写到文尾
	db.idxFile.Seek(tPos, os.SEEK_SET)
	db.dataFile.Seek(maxFilePos, os.SEEK_SET)
	return

}