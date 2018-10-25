package qdb

import (
	"github.com/piotrnar/qdb"
	"io"
	"bytes"
	"github.com/piotrnar/gocoin/btc"
	"fmt"
	"github.com/OhBonsai/yoin/core"
)

const (
	UnwindBufferMaxHistory = 7*24*6  //分叉了存一个星期
)

type unwindDB struct {
	tdb *qdb.DB
	lastBlockHeight uint32
	lastBlockHash [32]byte
	defragCount uint64
}


func newUnwindDB(dir string) (db *unwindDB) {
	db = new(unwindDB)
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

func (db *unwindDB) del(height uint32) {
	db.tdb.Del(qdb.KeyType(height))
}


func (db *unwindDB) sync() {
	db.tdb.Sync()
}

func (db *unwindDB) nosync() {
	db.tdb.NoSync()
}

func (db *unwindDB) save() {
	db.tdb.Defrag()
}

func (db *unwindDB) close() {
	db.tdb.Close()
}

func (db *unwindDB) idle() bool {
	if db.tdb.Defrag() {
		db.defragCount++
		return true
	}
	return false
}

func unwindFromReader(f io.Reader, unsp *unspentDB) {
	for {
		po, to := readSpent(f)
		if po == nil {
			break
		}
		if to != nil {
			// record deleted - so add it
			unsp.add(po, to)
		} else {
			// record added - so delete it
			unsp.del(po)
		}
	}
}


func (db *unwindDB) undo(height uint32, unsp *unspentDB) {
	if height != db.lastBlockHeight {
		panic("Unexpected height")
	}

	v := db.tdb.Get(qdb.KeyType(height))
	if v == nil {
		panic("Unwind data not found")
	}

	unwindFromReader(bytes.NewReader(v[32:]), unsp)
	db.del(height)

	db.lastBlockHeight--
	v = db.tdb.Get(qdb.KeyType(db.lastBlockHeight))
	if v == nil {
		panic("Parent data not found")
	}
	copy(db.lastBlockHash[:], v[:32])
	return
}

func (db *unwindDB) commit(changes *core.BlockChangeEvent, blhash []byte) {
	if db.lastBlockHeight+1 != changes.Height {
		println(db.lastBlockHeight+1, changes.Height)
		panic("Unexpected height")
	}
	db.lastBlockHeight++
	copy(db.lastBlockHash[:], blhash[0:32])

	f := new(bytes.Buffer)
	f.Write(blhash[0:32])
	for k, _ := range changes.AddedTxs {
		writeSpent(f, &k, nil)
	}
	for k, v := range changes.DeletedTxs {
		writeSpent(f, &k, v)
	}
	db.tdb.Put(qdb.KeyType(changes.Height), f.Bytes())
	if changes.Height >= UnwindBufferMaxHistory {
		db.del(changes.Height-UnwindBufferMaxHistory)
	}
}


func (db *unwindDB) GetLastBlockHash() (val []byte) {
	if db.lastBlockHeight != 0 {
		val = make([]byte, 32)
		copy(val, db.lastBlockHash[:])
	}
	return
}


func (db *unwindDB) stats() (s string) {
	s = fmt.Sprintf("UNWIND: len:%d  last:%d  defrags:%d\n",
		db.tdb.Count(), db.lastBlockHeight, db.defragCount)
	s += "Last block: " + btc.NewUint256(db.lastBlockHash[:]).String() + "\n"
	return
}
