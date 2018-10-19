package core

import "math/big"

var b58set = []byte("123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz")

func b58chr2int(chr byte) int {
	for i:=range b58set {
		if b58set[i]==chr {
			return i
		}
	}
	return -1
}


var bn0 *big.Int = big.NewInt(0)
var bn58 *big.Int = big.NewInt(58)

func Encodeb58(a []byte) (s string) {
	idx := len(a) * 138 / 100 + 1
	buf := make([]byte, idx)
	bn := big.NewInt(0).SetBytes(a)
	var mo *big.Int
	for bn.Cmp(bn0) != 0 {
		bn, mo = bn.DivMod(bn, bn58, new(big.Int))
		idx--
		buf[idx] = b58set[mo.Int64()]
	}
	for i := range a {
		if a[i]!=0 {
			break
		}
		idx--
		buf[idx] = b58set[0]
	}

	s = string(buf[idx:])

	return
}

func Decodeb58(s string) []byte {
	bn := big.NewInt(0)
	for i := range s {
		v := b58chr2int(byte(s[i]))
		if v < 0 {
			return nil
		}
		bn = bn.Mul(bn, bn58)
		bn = bn.Add(bn, big.NewInt(int64(v)))
	}
	return bn.Bytes()
}
