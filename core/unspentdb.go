package core

import "github.com/piotrnar/qdb"

const prevOutIdxLen = qdb.KeySize

type unspentDb struct {
	dir string
	tdb [0x100] *qdb.DB
	defragIndex int
	defragCount uint64
	nosyncinprogress bool
}

func newUnspentDB(dir string) (db *unspentDb) {
	db = new(unspentDb)
	db.dir = dir
	return
}