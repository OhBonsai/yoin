package qdb

import (
	"github.com/piotrnar/qdb"
	"fmt"
	"github.com/OhBonsai/yoin/core"
	"encoding/binary"
	"errors"
)

//TODO 改用sqlite数据，虽然会性能差，但是更加容易理解，现在用的是一个简单的K,V数据库

const prevOutIdxLen = qdb.KeySize


type unspentDB struct {
	dir string
	tdb [0x100] *qdb.DB
	defragIndex int
	defragCount uint64
	nosyncinprogress bool
}


func newUnspentDB(dir string) (db *unspentDB) {
	db = new(unspentDB)
	db.dir = dir
	return
}


// 通过某个值获取到数据库文件的具体位置
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

// 从TxIn引用的哪个Txout的交易hash获取key值
func getUnspentIndex(po *core.TxPrevOut) (qdb.KeyType) {
	return qdb.KeyType(binary.LittleEndian.Uint64(po.PreOutTxHash[:8]) ^ uint64(po.OutIdxInTx))
}


// 通过hash值获取Txout
func (db *unspentDB) get(po *core.TxPrevOut) (res *core.TxOut, e error) {
	ind := getUnspentIndex(po)
	val := db.dbN(int(po.PreOutTxHash[31])).Get(ind)

	if val==nil {
		e = errors.New("Unspent not found")
		return
	}

	if len(val)<48 {
		panic(fmt.Sprint("unspent record too short:", len(val)))
	}

	res = new(core.TxOut)
	res.Value = binary.LittleEndian.Uint64(val[36:44])
	res.BlockHeight = binary.LittleEndian.Uint32(val[44:48])
	res.ScriptPubKey  = make([]byte, len(val)-48)
	copy(res.ScriptPubKey, val[48:])
	return
}

// 存入k,v
func (db *unspentDB) add(idx *core.TxPrevOut, valPk *core.TxOut) {
	v := make([]byte, 48+len(valPk.ScriptPubKey))
	copy(v[0:32], idx.PreOutTxHash[:])
	binary.LittleEndian.PutUint32(v[32:36], idx.OutIdxInTx)
	binary.LittleEndian.PutUint64(v[36:44], valPk.Value)
	binary.LittleEndian.PutUint32(v[44:48], valPk.BlockHeight)
	copy(v[48:], valPk.ScriptPubKey)
	ind := getUnspentIndex(idx)
	db.dbN(int(idx.PreOutTxHash[31])).Put(ind, v)
}


// 删除某个k
func (db *unspentDB) del(idx *core.TxPrevOut) {
	db.dbN(int(idx.PreOutTxHash[31])).Del(getUnspentIndex(idx))
}


// TODO 数据库行为， 暂时不清楚
func (db *unspentDB) idle() bool {
	for _ = range db.tdb {
		db.defragIndex++
		if db.defragIndex >= len(db.tdb) {
			db.defragIndex = 0
		}
		if db.tdb[db.defragIndex]!=nil && db.tdb[db.defragIndex].Defrag() {
			db.defragCount++
			//println(db.defragIndex, "defragmented")
			return true
		}
	}
	return false
}

// TODO
func bin2unspent(v []byte, a uint32) (nr core.SingleUnspentTx) {
	copy(nr.TxPrevOut.PreOutTxHash[:], v[0:32])
	nr.TxPrevOut.OutIdxInTx = binary.LittleEndian.Uint32(v[32:36])
	nr.Value = binary.LittleEndian.Uint64(v[36:44])
	nr.MinedAt = binary.LittleEndian.Uint32(v[44:48])
	nr.AskIndex = a
	return
}

// 拿到地址集合所有的unpsent
func (db *unspentDB) GetAllUnspent(addr []*core.BtcAddr, quick bool) (res core.AllUnspentTx) {
	if quick {
		addrs := make(map[uint64] uint32, len(addr))
		for i, _  := range addr {
			addrs[binary.LittleEndian.Uint64(addr[i].Hash160[0:8])] = uint32(i)
		}
		for i := range db.tdb {
			db.dbN(i).Browse(func(k qdb.KeyType, v []byte) bool {
				scr := v[48:]
				if len(scr)==25 && scr[0]==0x76 && scr[1]==0xa9 && scr[2]==0x14 && scr[23]==0x88 && scr[24]==0xac {
					if askidx, ok := addrs[binary.LittleEndian.Uint64(scr[3:3+8])]; ok {
						res = append(res, bin2unspent(v[:48], askidx))
					}
				}
				return true
			})
		}
	} else {
		for i := range db.tdb {
			db.dbN(i).Browse(func(k qdb.KeyType, v []byte) bool {
				for a := range addr {
					if addr[a].Owns(v[48:]) {
						res = append(res, bin2unspent(v[:48], uint32(a)))
					}
				}
				return true
			})
		}
	}
	return
}


func (db *unspentDB) commit(changes *core.BlockChangeEvent) {
	// Now ally the unspent changes
	for k, v := range changes.AddedTxs {
		db.add(&k, v)
	}
	for k, _ := range changes.DeletedTxs {
		db.del(&k)
	}
}


func (db *unspentDB) stats() (s string) {
	var cnt, sum uint64
	for i := range db.tdb {
		db.dbN(i).Browse(func(k qdb.KeyType, v []byte) bool {
			sum += binary.LittleEndian.Uint64(v[36:44])
			cnt++
			return true
		})
	}
	return fmt.Sprintf("UNSPENT: %.8f BTC in %d outputs. defrags:%d\n",
		float64(sum)/1e8, cnt, db.defragCount)
}


func (db *unspentDB) sync() {
	db.nosyncinprogress = false
	for i := range db.tdb {
		if db.tdb[i]!=nil {
			db.tdb[i].Sync()
		}
	}
}

func (db *unspentDB) nosync() {
	db.nosyncinprogress = true
	for i := range db.tdb {
		if db.tdb[i]!=nil {
			db.tdb[i].NoSync()
		}
	}
}

func (db *unspentDB) save() {
	for i := range db.tdb {
		if db.tdb[i]!=nil {
			db.tdb[i].Defrag()
		}
	}
}

func (db *unspentDB) close() {
	for i := range db.tdb {
		if db.tdb[i]!=nil {
			db.tdb[i].Close()
			db.tdb[i] = nil
		}
	}
}

