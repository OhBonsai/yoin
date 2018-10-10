package core

import (
	"bytes"
	"encoding/binary"
)

// 签名hash类型
const (
	_ = iota
	SIGHASH_ALL                    // 整单全包，对输入输出负全责
	SIGHASH_NONE                   // 开了张支票，只对输入负责，输出给谁都行
	SIGHASH_SINGLE                 // 对某个输入输出负责
	SIGHASH_ANYONECANPAY = 0x80
)

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