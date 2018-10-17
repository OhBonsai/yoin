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
	blockIndex map[[Uint256IdxLen]byte] oneBl

	blockdata *os.File  //
	blockindx *os.File
	mutex sync.Mutex
}



