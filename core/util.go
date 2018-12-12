package core

import (
	"fmt"
	"io"
	"github.com/pkg/errors"
)

// 用来判断交易输出是不是空
func allzeros(b []byte) bool {
	for i := range b {
		if b[i]!=0 {
			return false
		}
	}
	return true
}

// 获取交易得梅克尔根值
func getMerkel(txs []*Tx) ([]byte) {
	mtr := make([][]byte, len(txs))
	for i := range txs {
		mtr[i] = txs[i].Hash.Hash[:]
	}
	var j, i2 int
	for siz:=len(txs); siz>1; siz=(siz+1)/2 {
		for i := 0; i < siz; i += 2 {
			if i+1 < siz-1 {
				i2 = i+1
			} else {
				i2 = siz-1
			}
			h := Sha2Sum(append(mtr[j+i], mtr[j+i2]...))
			mtr = append(mtr, h[:])
		}
		j += siz
	}
	return mtr[len(mtr)-1]
}


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

func ReadVarLen(b io.Reader) (res uint64, e error) {
	var buf [8]byte;
	var n int

	n, e = b.Read(buf[:1])
	if e != nil {
		println("ReadVLen1 error:", e.Error())
		return
	}

	if n != 1 {
		e = errors.New("Buffer empty")
		return
	}

	if buf[0] < 0xfd {
		res = uint64(buf[0])
		return
	}

	c := 2 << (2-(0xff-buf[0]));

	n, e = b.Read(buf[:c])
	if e != nil {
		println("ReadVLen1 error:", e.Error())
		return
	}
	if n != c {
		e = errors.New("Buffer too short")
		return
	}
	for i:=0; i<c; i++ {
		res |= (uint64(buf[i]) << uint64(8*i))
	}
	return
}


// Writes var_length field into the given writer
func WriteVarLen(b io.Writer, var_len uint32) {
	if var_len < 0xfd {
		b.Write([]byte{byte(var_len)})
		return
	}
	if var_len < 0x10000 {
		b.Write([]byte{0xfd, byte(var_len), byte(var_len>>8)})
		return
	}
	b.Write([]byte{0xfe, byte(var_len), byte(var_len>>8), byte(var_len>>16), byte(var_len>>24)})
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

