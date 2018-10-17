package core


const Uint256IdxLen = 6  // bigger number, larger memory needed , less collision happen


// hash字节处理工具，因为最后都是二进制，方便人来阅读，搞成小段模式。

type Uint256 struct {
	Hash [32]byte
}


func NewSha2Hash(data []byte) (res *Uint256) {
	res = new(Uint256)
	res.Hash = Sha2Sum(data[:])
}

