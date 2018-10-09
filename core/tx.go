package core

import "bytes"

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


func (t *Tx) Serializer() ([]byte){
	wr := new(bytes.Buffer)
	return wr.Bytes()
}