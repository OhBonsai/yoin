package core

import "fmt"

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
}


func (u *Uint256) String() (s string) {
	for i := 0; i<32; i++ {
		s+= fmt.Sprintf("%02x", u.Hash[31-i])
	}
	return
}
