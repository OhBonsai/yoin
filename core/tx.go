package core

import (
	"bytes"
	"encoding/binary"
	"crypto/sha256"
	"fmt"
	"errors"
)

// 签名hash类型
const (
	_ = iota
	SIGHASH_ALL                    // 整单全包，对输入输出负全责
	SIGHASH_NONE                   // 开了张支票，只对输入负责，输出给谁都行
	SIGHASH_SINGLE                 // 对某个输入输出负责
	SIGHASH_ANYONECANPAY = 0x80
)

var slowMode bool

type TxPrevOut struct {            // 输入引用那个未花费的输出
	PreOutTxHash  [32]byte           // 对应输出的交易hash
	OutIdxInTx    uint32             // 对应输出交易Output index， 从0开始
}

type TxIn struct {
	Input TxPrevOut
	ScriptSig []byte                 // 解锁脚本，基于栈，支持IF/ELSE
	Sequence uint32                  // 锁定时间/区块
}

type TxOut struct {
	Value uint64                      // 多少钱
	BlockHeight uint32                // TODO 为什么要存快高，是为了加速验证这个TXout嘛？
	ScriptPubKey []byte               // 加密脚本，答对了钱就是你的。
}

type Tx struct {
	Version uint32                    // 交易版本号，一直是1
	TxIns    []*TxIn                  // 输入集合
	TxOuts   []*TxOut                 // 输出集合
	LockTime uint32                   // 锁定时间/区块

	Size uint32                       // TODO 什么的大小
	Hash *Uint256                     //  本次交易的hash
}


// 编码交易
func (t *Tx) Serialize() ([]byte){
	var tmpBuf [9]byte
	buf := new(bytes.Buffer)

	// 版本
	binary.Write(buf, binary.LittleEndian, t.Version)

	// 交易输入数量
	buf.Write(tmpBuf[: StoreVarLen(tmpBuf[:], len(t.TxIns))])
	// 交易输入
	for i := range t.TxIns {
		buf.Write(t.TxIns[i].Input.PreOutTxHash[:])
		binary.Write(buf, binary.LittleEndian, t.TxIns[i].Input.OutIdxInTx)
		buf.Write(tmpBuf[: StoreVarLen(tmpBuf[:], len(t.TxIns[i].ScriptSig))])
		buf.Write(t.TxIns[i].ScriptSig[:])
		binary.Write(buf, binary.LittleEndian, t.TxIns[i].Sequence)
	}

	// 交易输出数量
	buf.Write(tmpBuf[: StoreVarLen(tmpBuf[:], len(t.TxOuts))])
	for i := range t.TxOuts {
		binary.Write(buf, binary.LittleEndian, t.TxOuts[i].Value)
		buf.Write(tmpBuf[:StoreVarLen(tmpBuf[:], len(t.TxOuts[i].ScriptPubKey))])
		buf.Write(t.TxOuts[i].ScriptPubKey[:])
	}

	// LockTime
	binary.Write(buf, binary.LittleEndian, t.LockTime)

	return buf.Bytes()
}

// 解码交易输出
func DeserializeTxOut(b []byte) (txOut *TxOut, offs int) {
	var l ,n int

	txOut = new(TxOut)
	txOut.Value = binary.LittleEndian.Uint64(b[:8])
	offs = 8

	l, n = LoadVarLen(b[offs:])
	offs += n

	txOut.ScriptPubKey = make([]byte, l)
	copy(txOut.ScriptPubKey[:], b[offs: offs + l])
	offs += l

	return
}

// 解码交易输入
func DeserializeTxIn(b []byte) (txIn *TxIn, offs int) {
    var l, n int

    txIn = new(TxIn)

    // 该输入对应输出交易HASH
    copy(txIn.Input.PreOutTxHash[:], b[:32])
    // 输出在交易的序号
    txIn.Input.OutIdxInTx = binary.LittleEndian.Uint32(b[32:36])
    offs = 32 + 4

    l, n = LoadVarLen(b[offs:])
    offs += n

    // 加密脚本
    txIn.ScriptSig = make([]byte, l)
    copy(txIn.ScriptSig[:], b[offs:offs+l])
    offs += l

    // Sequence
	txIn.Sequence = binary.LittleEndian.Uint32(b[offs:offs+4])
	offs += 4

	return
}

// 解码交易
func DeserializeTx(b []byte) (tx *Tx, offs int) {
	var l, n int

	tx = new(Tx)
	tx.Version = binary.LittleEndian.Uint32(b[0:4])
	offs = 4

	// TxIns
	l, n = LoadVarLen(b[offs:])
	offs += n
	tx.TxIns = make([]*TxIn, l)
	for i, _ := range tx.TxIns {
		tx.TxIns[i], n = DeserializeTxIn(b[offs:])
		offs += n
	}

	// TxOuts
	l, n = LoadVarLen(b[offs:])
	offs += n
	tx.TxOuts = make([]*TxOut, l)
	for i, _ := range tx.TxOuts {
		tx.TxOuts[i], n =  DeserializeTxOut(b[offs:])
		offs += n
	}

	tx.LockTime = binary.LittleEndian.Uint32(b[offs: offs+4])
	offs += 4
	return
}


// 对脚本进行签名
func (t *Tx) SignatureHash(scriptCode []byte, nIn int, hashType byte) ([]byte) {
	var tmpBuf [9]byte

	ht := hashType & 0x1f
	sha := sha256.New()

	binary.LittleEndian.PutUint32(tmpBuf[:4], t.Version)
	sha.Write(tmpBuf[:4])

	if (hashType & SIGHASH_ANYONECANPAY) != 0 {
		sha.Write([]byte{1}) // 只有一个输入

		// 构造一个输入的 待HASH字节
		sha.Write(t.TxIns[nIn].Input.PreOutTxHash[:])
		binary.LittleEndian.PutUint32(tmpBuf[:4], t.TxIns[nIn].Input.OutIdxInTx)

		sha.Write(tmpBuf[:4])
		sha.Write(tmpBuf[:StoreVarLen(tmpBuf[:], len(t.TxIns))])
		sha.Write(scriptCode[:])

		binary.LittleEndian.PutUint32(tmpBuf[:4], t.TxIns[nIn].Sequence)
	} else {
		sha.Write(tmpBuf[:StoreVarLen(tmpBuf[:], len(t.TxIns))])

		for i := range t.TxIns {
			sha.Write(t.TxIns[i].Input.PreOutTxHash[:])
			binary.LittleEndian.PutUint32(tmpBuf[:4], t.TxIns[i].Input.OutIdxInTx)
			sha.Write(tmpBuf[:4])

			if i == nIn {
				sha.Write(tmpBuf[:StoreVarLen(tmpBuf[:], len(scriptCode))])
				sha.Write(scriptCode[:])
			} else {
				sha.Write([]byte{0})
			}

			if (ht==SIGHASH_NONE || ht==SIGHASH_SINGLE) && i!=nIn {
				sha.Write([]byte{0, 0, 0, 0})
			} else {
				binary.LittleEndian.PutUint32(tmpBuf[:4], t.TxIns[i].Sequence)
				sha.Write(tmpBuf[:4])
			}
		}
	}

	if ht == SIGHASH_NONE {
		sha.Write([]byte{0})
	} else if ht == SIGHASH_SINGLE {
		nOut := nIn

		if nOut >= len(t.TxOuts) {
			fmt.Printf("ERROR: SignatureHash() : nOut=%d out of range\n", nOut);
			return nil
		}

		sha.Write(tmpBuf[:StoreVarLen(tmpBuf[:], nOut+1)])
		for i:=0; i<nOut; i++ {
			sha.Write([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0})
		}

		binary.LittleEndian.PutUint64(tmpBuf[:8], t.TxOuts[nOut].Value)
		sha.Write(tmpBuf[:8])
		sha.Write(tmpBuf[:StoreVarLen(tmpBuf[:], len(t.TxOuts))])
		sha.Write(t.TxOuts[nOut].ScriptPubKey[:])
	} else {
		sha.Write(tmpBuf[:StoreVarLen(tmpBuf[:], len(t.TxOuts))])

		for i := range t.TxOuts {
			binary.LittleEndian.PutUint64(tmpBuf[:8], t.TxOuts[i].Value)
			sha.Write(tmpBuf[:8])

			sha.Write(tmpBuf[:StoreVarLen(tmpBuf[:], len(t.TxOuts[i].ScriptPubKey))])
			sha.Write(t.TxOuts[i].ScriptPubKey[:])
		}
	}

	binary.LittleEndian.PutUint32(tmpBuf[:4], t.LockTime)
	sha.Write(tmpBuf[:4])
	sha.Write([]byte{hashType, 0, 0, 0})

	// TODO 搞不懂，需要打印二进制来看看
	tmp := sha.Sum(nil)
	sha.Reset()
	sha.Write(tmp)
	return sha.Sum(nil)
}

func (t *TxPrevOut)String() (s string) {
	for i := 0; i<32; i++ {
		s+= fmt.Sprintf("%02x", t.PreOutTxHash[31-i])
	}
	s+= fmt.Sprintf("-%03d", t.OutIdxInTx)
	return
}


// 检查一下交易有么有问题
func (tx *Tx) CheckTransaction() error {
	// 得有点东西
	if len(tx.TxIns)==0 {
		return errors.New("CheckTransaction() : vin empty")
	}
	if len(tx.TxOuts)==0 {
		return errors.New("CheckTransaction() : vout empty")
	}

	// 不能太大了
	if tx.Size > MAX_BLOCK_SIZE {
		return errors.New("CheckTransaction() : size limits failed")
	}

	if slowMode {
		var nValueOut uint64
		// 你钱转的太多了。。。不可能存在
		for i := range tx.TxOuts {
			if tx.TxOuts[i].Value > MAX_MONEY {
				return errors.New("CheckTransaction() : txout.nValue too high")
			}
			nValueOut += tx.TxOuts[i].Value
			if nValueOut > MAX_MONEY {
				return errors.New("CheckTransaction() : txout total out of range")
			}
		}

		// 是不是有重复得输入
		vInOutPoints := make(map[TxPrevOut]bool, len(tx.TxIns))
		for i := range tx.TxIns {
			_, present := vInOutPoints[tx.TxIns[i].Input]
			if present {
				return errors.New("CheckTransaction() : duplicate inputs")
			}
			vInOutPoints[tx.TxIns[i].Input] = true
		}
	}



}


//是不是奖励比
func (tx *Tx) isCoinBase() bool {
	return len(tx.TxIns) == 1 && tx.TxIns[0].Input
}

//TXOUT是不是空，用来判断是不是厨师交易
func (out TxPrevOut) IsNull() bool {
	return
}
