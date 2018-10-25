package qdb

import (
	"os"
	"github.com/OhBonsai/yoin/core"
)

type UnspentTDB struct {
	unspent *unspentDB
	unwind *unwindDB
}


func NewDb(dir string, init bool) core.UnspentDBInf{
	var db UnspentTDB

	if init {
		os.RemoveAll(dir+"unspent/")
		os.RemoveAll(dir+"unspent/unwind/")
	}

	db.unspent = newUnspentDB(dir+"unspent/")
	db.unwind = newUnwindDB(dir+"unspent/unwind/")

	return &db
}


func (db UnspentTDB) GetLastBlockHash() ([]byte) {
	return db.unwind.GetLastBlockHash()
}


func (db UnspentTDB) CommitBlockTxs(changes *core.BlockChangeEvent, blhash []byte) (e error) {
	// First the unwind data
	db.unwind.commit(changes, blhash)
	db.unspent.commit(changes)
	return
}

func (db UnspentTDB) GetStats() (s string) {
	s += db.unspent.stats()
	s += db.unwind.stats()
	return
}


// Flush all the data to files
func (db UnspentTDB) Sync() {
	db.unwind.sync()
	db.unspent.sync()
}

func (db UnspentTDB) NoSync() {
	db.unwind.nosync()
	db.unspent.nosync()
}


func (db UnspentTDB) Close() {
	db.unwind.close()
	db.unspent.close()
}


func (db UnspentTDB) Idle() {
	if !db.unspent.idle() {
		//println("No Unspent to defrag")
		db.unwind.idle()
	}
}


func (db UnspentTDB) Save() {
	db.unwind.save()
	db.unspent.save()
}

func (db UnspentTDB) UndoBlockTransactions(height uint32) {
	db.unwind.undo(height, db.unspent)
}


func (db UnspentTDB) UnspentGet(po *core.TxPrevOut) (res *core.TxOut, e error) {
	return db.unspent.get(po)
}

func (db UnspentTDB) GetAllUnspent(addr []*core.BtcAddr, quick bool) (res core.AllUnspentTx) {
	return db.unspent.GetAllUnspent(addr, quick)
}

func init() {
	core.NewUnspentDB = NewDb
}
