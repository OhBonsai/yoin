package core

import (
	"errors"
	"bytes"
	"fmt"
)

const (
	ADDRVER_BTC = 0x00
	ADDRVER_TESTNET = 0x6f
	ADDRVER_BONSAI = 0xff
)

type BtcAddr struct {
	Version byte
	Hash160 [20]byte

	CheckSum []byte
	PubKey []byte
	Enc58str string
}

func NewAddrFromString(s string) (a *BtcAddr, e error) {
	dec := Decodeb58(s)
	if dec == nil {
		e = errors.New("Cannot decode b58 string *"+ s +"*")
		return
	}

	if len(dec) < 25 {
		dec = append(bytes.Repeat([]byte{0}, 25-len(dec)), dec...)
	}

	if (len(dec)==25) {
		sh := Sha2Sum(dec[0:21])
		if !bytes.Equal(sh[:4], dec[21:25]) {
			e = errors.New("Address Checksum error")
		} else {
			a = new(BtcAddr)
			a.Version = dec[0]
			copy(a.Hash160[:], dec[1:21])
			a.CheckSum = make([]byte, 4)
			copy(a.CheckSum, dec[21:25])
			a.Enc58str = s
		}
	} else {
		e = errors.New(fmt.Sprintf("Unsupported hash length %d", len(dec)))
	}
	return
}


func NewAddrFromHash160(in []byte, ver byte) (a *BtcAddr) {
	a = new(BtcAddr)
	a.Version = ver
	copy(a.Hash160[:], in[:])
	return
}


func NewAddrFromPubKey(in []byte, ver byte) (a *BtcAddr) {
	a = new(BtcAddr)
	a.PubKey = make([]byte, len(in))
	copy(a.PubKey[:], in[:])
	a.Version = ver
	a.Hash160 = Rimp160AfterSha256(in)
	return
}

func NewAddrFromPkScript(scr []byte, ver byte) (*BtcAddr) {
	if len(scr)==25 && scr[0]==0x76 && scr[1]==0xa9 && scr[2]==0x14 && scr[23]==0x88 && scr[24]==0xac {
		return NewAddrFromHash160(scr[3:23], ver)
	} else if len(scr)==67 && scr[0]==0x41 && scr[1]==0x04 && scr[66]==0xac {
		return NewAddrFromPubKey(scr[1:66], ver)
	} else if len(scr)==23 && scr[0]==0xa9 && scr[1]==0x14 && scr[22]==0x87 {
		return NewAddrFromHash160(scr[2:22], ver)
	}
	return nil
}


func NewAddrFromDataWithSum(in []byte, ver byte) (a *BtcAddr, e error) {
	var ad [25]byte
	ad[0] = ver
	copy(ad[1:25], in[:])
	sh := Sha2Sum(ad[0:21])
	if !bytes.Equal(in[20:24], sh[:4]) {
		e = errors.New("Address Checksum error")
		return
	}

	copy(ad[21:25], sh[:4])

	a = new(BtcAddr)
	a.Version = ver
	copy(a.Hash160[:], in[:])

	a.CheckSum = make([]byte, 4)
	copy(a.CheckSum, sh[:4])
	return
}


func (a *BtcAddr) String() string {
	if a.Enc58str=="" {
		var ad [25]byte
		ad[0] = a.Version
		copy(ad[1:21], a.Hash160[:])
		if a.CheckSum==nil {
			sh := Sha2Sum(ad[0:21])
			a.CheckSum = make([]byte, 4)
			copy(a.CheckSum, sh[:4])
		}
		copy(ad[21:25], a.CheckSum[:])
		a.Enc58str = Encodeb58(ad[:])
	}
	return a.Enc58str
}

func (a *BtcAddr) Owns(scr []byte) (yes bool) {
	// The most common spend script
	if len(scr)==25 && scr[0]==0x76 && scr[1]==0xa9 && scr[2]==0x14 && scr[23]==0x88 && scr[24]==0xac {
		yes = bytes.Equal(scr[3:23], a.Hash160[:])
		return
	}

	// Spend script with an entire public key
	if len(scr)==67 && scr[0]==0x41 && scr[1]==0x04 && scr[66]==0xac {
		if a.PubKey == nil {
			h := Rimp160AfterSha256(scr[1:66])
			if h == a.Hash160 {
				a.PubKey = make([]byte, 65)
				copy(a.PubKey, scr[1:66])
				yes = true
			}
			return
		}
		yes = bytes.Equal(scr[1:34], a.PubKey[:33])
		return
	}

	// Spend script with a compressed public key
	if len(scr)==35 && scr[0]==0x21 && (scr[1]==0x02 || scr[1]==0x03) && scr[34]==0xac {
		if a.PubKey == nil {
			h := Rimp160AfterSha256(scr[1:34])
			if h == a.Hash160 {
				a.PubKey = make([]byte, 33)
				copy(a.PubKey, scr[1:34])
				yes = true
			}
			return
		}
		yes = bytes.Equal(scr[1:34], a.PubKey[:33])
		return
	}

	return
}


func (a *BtcAddr) OutScript() (res []byte) {
	res = make([]byte, 25)
	res[0] = 0x76
	res[1] = 0xa9
	res[2] = 20
	copy(res[3:23], a.Hash160[:])
	res[23] = 0x88
	res[24] = 0xac
	return
}