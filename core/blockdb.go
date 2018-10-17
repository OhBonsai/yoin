package core

import (
	"os"
	"sync"
)

type oneBl struct {
	fpos uint64 // where at the block is stored in blockchain.dat
	blen uint32 // how long the block is in blockchain.dat

	ipos int64  // where at the record is stored in blockchain.idx
	trusted bool
}

type BlockDB struct {
	dirname string
	blockMap map[[Uint256IdxLen]byte] oneBl

	blockdata *os.File  //
	blockidx *os.File
	mutex sync.Mutex
}


func NewBlockDB(dir string) (db *BlockDB) {
	db = new(BlockDB)
	db.dirname = dir

	if db.dirname!="" && db.dirname[len(db.dirname )-1]!='/' && db.dirname[len(db.dirname )-1]!='\\' {
		db.dirname += "/"
	}

	db.blockMap = make(map[[Uint256IdxLen]byte] oneBl)
	os.MkdirAll(db.dirname, 0770)
	db.blockdata, _ = os.OpenFile(db.dirname + "blockchain.dat", os.O_RDWR | os.O_CREATE, 0660)

	if db.blockdata == nil {
		panic("Can't not open blockchain.data")
	}

	db.blockidx, _ = os.OpenFile(db.dirname + "blockchain.idx", os.O_RDWR | os.O_CREATE, 0660)
	if db.blockidx == nil {
		panic("Can't not open blockchain.idx")
	}
	return
}

