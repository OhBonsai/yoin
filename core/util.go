package core

import (
	"fmt"
	"crypto/sha256"
	"golang.org/x/crypto/ripemd160"
)

// 变长二进制小端存到指定buffer中
// 这个算法可以更加极限一点，在NDS导航通讯协议中有更优秀的VARINT压缩逻辑
func StoreVarLen(b []byte, value int)  int {
	// 直接搞成无符号
	uValue := uint32(value)

	// 值的范围 [0, 0xfd), 直接一个byte就能存这个值
	if uValue < 0xfd {
		b[0] = byte(uValue)
		return 1
	}

	// 值的范围 [0xfd, 0xffff], 用三个byte值存
	if uValue <= 0xffff {
		b[0] = 0xfd
		b[1] = byte(uValue)
		b[2] = byte(uValue >> 8)
	}

	// 值的范围 (0xffff, 2^33-1), 五个byte
	b[0] = 0xfe
	b[1] = byte(uValue)
	b[2] = byte(uValue>>8)
	b[3] = byte(uValue>>16)
	b[4] = byte(uValue>>24)
	return 5
}

func LoadVarLen(b []byte) (len int, var_int_size int) {
	c := b[0]

	if c < 0xfd {
		return int(c), 1
	}

	// 0xfd->3 , 0xfe->5
	var_int_size = 1 + (2<<(2-(0xff-c)))

	var res uint64
	// 小端转uint64
	for i:=1; i<var_int_size; i++{
		res |= (uint64(b[i]) << uint64(8*(i-1)))
	}

	if res>0x7fffffff {
		panic("wow, this should never happen")
	}

	len = int(res)
	return len, var_int_size
}


// debug tool
func PrintByteSlice(b []byte) {
	for i, _ := range b{
		if i % 16 == 0 {
			fmt.Print("\n")
			fmt.Printf("%d ", i / 16)
		}

		v := fmt.Sprintf("%x", b[i])
		if len(v) == 1{
			v = "0"+v
		}
		
		fmt.Printf("%s ", v)
	}
}

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