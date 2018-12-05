package core

import (
	"fmt"
	"math/big"
)

const Uint256IdxLen = 6  // bigger number, larger memory needed , less collision happen


// hash字节处理工具，因为最后都是二进制，方便人来阅读，搞成小段模式。

type Uint256 struct {
	Hash [32]byte
}

func NewUint256(h []byte) (res *Uint256) {
	res = new(Uint256)
	copy(res.Hash[:], h[:])
	return
}

func NewSha2Hash(data []byte) (res *Uint256) {
	res = new(Uint256)
	res.Hash = Sha2Sum(data[:])
	return
}


func (u *Uint256) String() (s string) {
	for i := 0; i<32; i++ {
		s+= fmt.Sprintf("%02x", u.Hash[31-i])
	}
	return
}

func (u *Uint256) BIdx() [Uint256IdxLen]byte {
	return NewBlockIndex(u.Hash[:])
}


func (u *Uint256) BigInt() *big.Int {
	var buf [32]byte
	for i := range buf {
		buf[i] = u.Hash[31-i]
	}
	return new(big.Int).SetBytes(buf[:])
}

func NewUint256FromString(s string) (res *Uint256) {
	var v int
	res = new(Uint256)
	for i := 0; i<32; i++ {
		fmt.Sscanf(s[2*i:2*i+2], "%x", &v)
		res.Hash[31-i] = byte(v)
	}
	return
}