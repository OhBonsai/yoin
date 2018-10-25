package qdb

import (
	"io"
	"github.com/OhBonsai/yoin/core"
	"encoding/binary"
)

func writeSpent(f io.Writer, po *core.TxPrevOut, to *core.TxOut) {
	if to == nil {
		// add
		f.Write([]byte{1})
		f.Write(po.PreOutTxHash[:])
		binary.Write(f, binary.LittleEndian, uint32(po.OutIdxInTx))
	} else {
		// deleted
		f.Write([]byte{0})
		f.Write(po.PreOutTxHash[:])
		binary.Write(f, binary.LittleEndian, uint32(po.OutIdxInTx))
		binary.Write(f, binary.LittleEndian, uint64(to.Value))
		binary.Write(f, binary.LittleEndian, uint32(len(to.ScriptPubKey)))
		f.Write(to.ScriptPubKey[:])
	}
}


func readSpent(f io.Reader) (po *core.TxPrevOut, to *core.TxOut) {
	var buf [49] byte
	n, e := f.Read(buf[:37])
	if n!=37 || e!=nil || buf[0]>1 {
		return
	}
	po = new(core.TxPrevOut)
	copy(po.PreOutTxHash[:], buf[1:33])
	po.OutIdxInTx = binary.LittleEndian.Uint32(buf[33:37])
	if buf[0]==0 {
		n, e = f.Read(buf[37:49])
		if n!=12 || e!=nil {
			panic("Unexpected end of file")
		}
		to = new(core.TxOut)
		to.Value = binary.LittleEndian.Uint64(buf[37:45])
		to.ScriptPubKey = make([]byte, binary.LittleEndian.Uint32(buf[45:49]))
		f.Read(to.ScriptPubKey[:])
	}
	return
}