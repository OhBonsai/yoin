package core

// hash字节处理工具，因为最后都是二进制，方便人来阅读，搞成小段模式。

type Uint256 struct {
	Hash [32]byte
}
