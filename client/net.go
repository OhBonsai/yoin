package client

import (
	"time"
	"github.com/OhBonsai/yoin/core"
	"net"
	"encoding/binary"
	"crypto/rand"
	"bytes"
	"fmt"
	"os"
	"strings"
	"github.com/pkg/errors"
)

const (
	Version = 7001
	UserAgent = "/Satoshi:0.8.1"

	Services = uint64(0x1)
	SendAddrsEvery = 15 * time.Minute
	AskAddrsEvery = 5 * time.Minute

	MaxInCons = 8
	MaxOutCons = 8
	MatTotCons = MaxInCons + MaxOutCons

	NoDataTimeout = 2 * time.Minute

	MaxBytesInSendBuffer = 16 * 1024
	NewBlocksAskDurations = 30 * time.Second
)

var (
	openCons map[uint64] * oneConnection = make(map[uint64]*oneConnection, MatTotCons)
	InConsActive, OutConsActive uint
	DefaultTcpPort uint16
	MyExternalAddr *core.NetAddr
	LastConnId uint32
)

type oneConnection struct {
	PeerAddr *onePeer
	ConnId uint32

	Broken bool
	BanIt bool

	Incomming bool
	*net.TCPConn

	ConnectedAt time.Time
	VerackReceived bool

	node struct{
		version uint32
		services uint64
		timestamp uint64
		height uint32
		agent string
	}

	recv struct{
		hdr [24]byte
		hdr_len int
		dat []byte
		datlen uint32
	}

	send struct{
		buf []byte
		sofar int
	}

	LoopCnt, TicksCnt uint
	BytesReceived, BytesSent uint64
	LastCmdRcvd, LastCmdSent string

	PendingInvs []*[36]byte
	NextAddrSent time.Time
	NextGetAddr time.Time
	LastDataGot time.Time
	LastBlocksFrom *core.BlockTreeNode
	NextBlocksAsk time.Time
}

func (c *oneConnection) SendRawMsg(cmd string, pl []byte) (e error) {
	if len(c.send.buf) > 1024*1024 {
		println(c.PeerAddr.Ip(), "WTF??", cmd, c.LastCmdSent)
		return
	}

	sbuf := make([]byte, 24+len(pl))

	c.LastCmdSent = cmd
	binary.LittleEndian.PutUint32(sbuf[0:4], Version)
	copy(sbuf[0:4], Magic[:])
	copy(sbuf[4:16], cmd)
	binary.LittleEndian.PutUint32(sbuf[16:20], uint32(len(pl)))
	sh := core.Sha2Sum(pl[:])
	copy(sbuf[20:24], sh[:4])
	copy(sbuf[24:], pl)

	c.send.buf = append(c.send.buf, sbuf...)
	return
}


func (c *oneConnection) DoS() {
	c.BanIt = true
	c.Broken = true
}


func putAddr(b *bytes.Buffer, a string) {
	var ip [4]byte
	var p uint16
	n, e := fmt.Sscanf(a, "%d.%d.%d.%d:%d", &ip[0], &ip[1], &ip[2], &ip[3], &p)
	if e != nil || n != 5 {
		println("Incorrect address:", a)
		os.Exit(1)
	}
	binary.Write(b, binary.LittleEndian, uint64(Services))

	// No Ip6 supported:
	b.Write([]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xFF, 0xFF})
	b.Write(ip[:])
	binary.Write(b, binary.BigEndian, uint16(p))
}

func (c *oneConnection) SendVersion() {
	b := bytes.NewBuffer([]byte{})

	binary.Write(b, binary.LittleEndian, uint32(Version))
	binary.Write(b, binary.LittleEndian, uint32(Services))
	binary.Write(b, binary.LittleEndian, uint64(time.Now().Unix()))

	putAddr(b, c.TCPConn.RemoteAddr().String())
	putAddr(b, c.TCPConn.LocalAddr().String())

	var nonce [8]byte
	rand.Read(nonce[:])
	b.Write(nonce[:])

	b.WriteByte(byte(len(UserAgent)))
	b.Write([]byte(UserAgent))

	binary.Write(b, binary.LittleEndian, uint32(LastBlock.Height))

	c.SendRawMsg("version", b.Bytes())
}

func (c *oneConnection) HandleError(e error) (error) {
	if nerr, ok := e.(net.Error); ok && nerr.Timeout() {
		return nil
	}

	if dbg > 0 {
		println("handleError:", e.Error())
	}
	c.recv.hdr_len = 0
	c.recv.dat = nil
	c.Broken = true
	return e
}

type BCmsg struct {
	cmd string
	pl []byte
}

func (c *oneConnection) FetchMessage() (*BCmsg) {
	var e error
	var n int

	c.TCPConn.SetReadDeadline(time.Now().Add(time.Microsecond))

	for c.recv.hdr_len < 24 {
		n, e = SockRead(c.TCPConn, c.recv.hdr[c.recv.hdr_len:24])
		c.recv.hdr_len += n
		if e != nil {
			c.HandleError(e)
			return nil
		}

		if c.recv.hdr_len >= 4 && !bytes.Equal(c.recv.hdr[:4], Magic[:]) {
			println("FetchMessage: Proto out of sync")
			c.Broken = true
			return nil
		}

		if c.Broken {
			return nil
		}
 	}

 	dlen := binary.LittleEndian.Uint32(c.recv.hdr[16:20])

 	if dlen > 0 {
 		if c.recv.dat == nil {
 			c.recv.dat = make([]byte, dlen)
 			c.recv.datlen = 0
		}
		for c.recv.datlen < dlen {
			n, e = SockRead(c.TCPConn, c.recv.dat[c.recv.datlen:])
			c.recv.datlen += uint32(n)
			if e != nil {
				c.HandleError(e)
				return nil
			}
			if c.Broken {
				return nil
			}
		}
	}

	sh := core.Sha2Sum(c.recv.dat)
	if !bytes.Equal(c.recv.hdr[20:24], sh[:4]) {
		println(c.PeerAddr.Ip(), "Msg checksum error")
		c.DoS()
		c.recv.hdr_len = 0
		c.recv.dat = nil
		c.Broken = true
		return nil
	}

	ret := new(BCmsg)
	ret.cmd = strings.TrimRight(string(c.recv.hdr[4:16]), "\000")
	ret.pl = c.recv.dat
	c.recv.dat = nil
	c.recv.hdr_len = 0

	c.BytesReceived += uint64(24 + len(ret.pl))
	return ret
}

func (c *oneConnection) AnnounceOwnAddr() {
	if MyExternalAddr == nil {
		return
	}

	var buf [31]byte
	c.NextAddrSent = time.Now().Add(SendAddrsEvery)
	buf[0] = 1
	binary.LittleEndian.PutUint32(buf[1:5], uint32(time.Now().Unix()))
	ipd := MyExternalAddr.Bytes()
	copy(buf[5:], ipd[:])

	c.SendRawMsg("addr", buf[:])
}


func (c *oneConnection) VerMsg(pl []byte) error {
	if len(pl) >= 46 {
		c.node.version = binary.LittleEndian.Uint32(pl[0:4])
		c.node.services = binary.LittleEndian.Uint64(pl[4:12])
		c.node.timestamp = binary.LittleEndian.Uint64(pl[12:20])

		if MyExternalAddr == nil {
			MyExternalAddr = core.NewNetAddr(pl[20:46])
			MyExternalAddr.Port = DefaultTcpPort
		}

		if len(pl) >= 86 {
			le, of := core.LoadVarLen(pl[80:])
			of += le
			if len(pl) >= of + 4 {
				c.node.height = binary.LittleEndian.Uint32(pl[of:of+4])
			}
		}
	} else {
		return errors.New("version message too short")
	}
	c.SendRawMsg("verack", []byte{})
	if c.Incomming {
		c.SendVersion()
	}
	return nil
}


func (c *oneConnection) GetBlocks(lastbl []byte) {
	if dbg > 0 {
		println("GetBlocks since", core.NewUint256(lastbl).String())
	}

	var b [4+1+32+32]byte
	binary.LittleEndian.PutUint32(b[0:4], Version)
	b[4] = 1
	copy(b[5:37], lastbl)
	c.SendRawMsg("getblocks", b[:])
}


func (c *oneConnection) ProcessInv(pl []byte) {
	if len(pl) < 37 {
		println(c.PeerAddr.Ip(), "inv payload too short", len(pl))
		return
	}

	cnt, of := core.LoadVarLen(pl)

	if len(pl) != of + 36*cnt {
		println("inv payload length mismatch", len(pl), of, cnt)
	}

	var txs uint32
	for i:=0; i<cnt; i++{
		typ := binary.LittleEndian.Uint32(pl[of:of+4])
		if typ == 2 {
			InvsNotify(pl[of+4:of+36])
		} else {
			txs ++
		}
		of += 36
	}

	if dbg>1 {
		println(c.PeerAddr.Ip(), "ProcessInv:", cnt, "tot /", txs, "txs")
	}
	return

}


func NetSendInv(typ uint32, h []byte, fromConn *oneConnection) (cnt uint) {
	inv := new([36]byte)
	binary.LittleEndian.PutUint32(inv[0:4], typ)
	copy(inv[4:36], h)

	mutex.Lock()
	defer mutex.Unlock()

	for _, v := range openCons {
		if v != fromConn {
			if len(v.PendingInvs) < 500 {
				v.PendingInvs = append(v.PendingInvs, inv)
				cnt ++
			}
		}
	}
	return
}


func addInvBlockBranch(inv map[[32]byte] bool, bl *core.BlockTreeNode, stop *core.Uint256) {
	if len(inv)>=500 || bl.BlockHash.Equal(stop) {
		return
	}

	inv[bl.BlockHash.Hash] = true
	for i:= range bl.Children {
		if len(inv) >= 500 {
			return
		}
		addInvBlockBranch(inv, bl.Children[i], stop)
	}
}

func (c *oneConnection) ProcessGetBlocks(pl []byte) {
	b := bytes.NewReader(pl)

	var ver uint32
	e := binary.Read(b, binary.LittleEndian, &ver)

	if e != nil {
		println("ProcessGetBlocks:", e.Error(), c.PeerAddr.Ip())
		return
	}

	cnt, e := core.ReadVarLen(b)
	if e != nil {
		println("ProcessGetBlocks:", e.Error(), c.PeerAddr.Ip())
	}

	h2get := make([]*core.Uint256, cnt)
	var h [32]byte
	for i:=0; i<int(cnt); i++ {
		n, _ := b.Read(h[:])
		if n != 32 {
			println("getblocks too short", c.PeerAddr.Ip())
			return
		}

		h2get[i] = core.NewUint256(h[:])
		if dbg > 1 {
			println(c.PeerAddr.Ip(), "getbl", h2get[i].String())
		}
	}

	n, _ := b.Read(h[:])
	if n != 32 {
		println("getblocks does not have hash_stop", c.PeerAddr.Ip())
		return
	}

	hashstop := core.NewUint256(h[:])

	var maxheight uint32
	invs := make(map[[32]byte] bool, 500)
	for i := range h2get {
		BlockChain.BlockIndexAccess.Lock()
		if bl, ok := BlockChain.BlockIndex[h2get[i].BIdx()]; ok {
			if bl.Height > maxheight {
				maxheight = bl.Height
			}
			addInvBlockBranch(invs, bl, hashstop)
		}
		BlockChain.BlockIndexAccess.Unlock()
		if len(invs) >= 500 {
			break
		}
	}

	if len(invs) > 0 {
		inv := new(bytes.Buffer)
		core.WriteVarLen(inv, uint32(len(invs)))
		for k, _ := range invs {
			binary.Write(inv, binary.LittleEndian, uint32(2))
			inv.Write(k[:])
		}

		if dbg > 1 {
			fmt.Println(c.PeerAddr.Ip(), "getblocks", cnt, maxheight, " ...", len(invs), "invs in resp ->", len(inv.Bytes()))
		}

		CountSafe("GetblocksReplies")
		c.SendRawMsg("inv", inv.Bytes())

	}
}