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
	}
}

