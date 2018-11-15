package core


import (
	"crypto/sha256"
	"golang.org/x/crypto/ripemd160"
)

// Return hash: SHA256(SHA256(data))
func Sha2Sum(b []byte) (out [32]byte){
	s := sha256.New()
	s.Write(b[:])

	tmp := s.Sum(nil)
	s.Reset()
	s.Write(tmp)
	copy(out[:], s.Sum(nil))
	return
}

// Return Hash: RIMP160(SHA256(data))
func Rimp160AfterSha256(in []byte) (res [20]byte){
	sha := sha256.New()
	sha.Write(in)

	rim := ripemd160.New()
	rim.Write(sha.Sum(nil)[:])
	copy(res[:], rim.Sum(nil))
	return
}
