package qdb

import (
	"github.com/piotrnar/qdb"
	"fmt"
	"github.com/OhBonsai/yoin/core"
	"encoding/binary"
)

const prevOutIdxLen = qdb.KeySize


type unspentDB struct {
	dir string
	tdb [0x100] *qdb.DB
	defragIndex int
	defragCount uint64
	nosyncinprogress bool
}


func NewUnspentDB(dir string) (db *unspentDB) {
	db = new(unspentDB)
	db.dir = dir
	return
}


func (db *unspentDB) dbN(i int) (*qdb.DB) {
	if db.tdb[i] == nil {
		db.tdb[i], _ = qdb.NewDB(db.dir + fmt.Sprintf("%02x/", i))
		db.tdb[i].Load()
		if db.nosyncinprogress {
			db.tdb[i].NoSync()
		}
	}
	return db.tdb[i]
}

func getUnspentIndex(po *core.TxPrevOut) (qdb.KeyType) {
	return qdb.KeyType(binary.LittleEndian.Uint64(po.PreOutTxHash[:8]) ^ uint64(po.OutIdxInTx))
}