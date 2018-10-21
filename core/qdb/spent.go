package qdb

import (
	"github.com/piotrnar/qdb"
	"io"
)


const (
	UnWindBufferMaxHistory = 7 * 24 * 6
)

type unwindDb struct {
	tdb *qdb.DB
	lastBlockHeight uint32
	lastBlockHash [32]byte
	defragCount uint64
}


func newUnwindDb(dir string) (db *unwindDb) {
	db = new(unwindDb)
	db.tdb, _ = qdb.NewDB(dir)

	db.lastBlockHeight = 0
	db.tdb.Browse(func(k qdb.KeyType, v []byte) bool {
		h := uint32(k)
		if h > db.lastBlockHeight {
			db.lastBlockHeight = h
			copy(db.lastBlockHash[:], v[:32])
		}
		return true
	})
	return
}


func unwindFromReader(f io.Reader, unsp *UnSpentDB)

