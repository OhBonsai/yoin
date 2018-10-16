package core

import "fmt"

type SingleUnspentTx struct {
	TxPrevOut
	Value uint64
	AskIndex uint32
	MinedAt uint32
}

type AllUnspentTx []SingleUnspentTx


func (su *SingleUnspentTx) String() string {
	return fmt.Sprintf("%15.8f BTC from ", float64(su.Value)/1e8) + su.TxPrevOut.String()
}



type BlockChangeEvent struct {
	Height     uint32
	AddedTxs   map[TxPrevOut] *TxOut
	DeletedTxs map[TxPrevOut] *TxOut
}

type UnspentDB interface {
	CommitBlockTxs(*BlockChangeEvent, []byte) error
	UndoBlockTransactions(uint32)
	GetLastBlockHash() []byte

	UnspentGet(out *TxPrevOut) (*TxOut, error)
	GetAllUnspent(addr []*BtcAddr, quick bool) AllUnspentTx

	Idle()
	Save()
	Close()
	NoSync()
	Sync()
	GetStats()(string)
}

