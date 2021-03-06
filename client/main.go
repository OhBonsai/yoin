package client

import (
	"flag"
	"github.com/OhBonsai/yoin/core"
	"github.com/piotrnar/gocoin/core"
	"time"
	"sync"
)

const (
	PendingFifoLen = 2000
)

var (
	testnet *bool = flag.Bool("t", false, "Uset Testnet3")
	rescan *bool = flag.Bool("r", false, "Discard unspent outputs DB and rescan the blockchain")
	proxy *string = flag.String("c", "", "Connect to this host")
	server *bool = flag.Bool("l", false, "Enable TCP server (allow incomming connections)")
	datadir *string = flag.String("d", "", "Specify yoin's database root folder")

	nosync *bool = flag.Bool("nosync", false, "Init blockchain with syncing disabled (dangerous!)")
	maxul = flag.Uint("ul", 0, "Upload limit in KB/s (0 for no limit)")
	maxdl = flag.Uint("dl", 0, "Download limit in KB/s (0 for no limit)")


	GenesisBlokc *core.Uint256
	Magic [4]byte
	BlockChain *core.Chain
	AddrVersion byte

	exit_now bool
	dbg uint64
	beep bool

	LastBlock *core.BlockTreeNode
	LastBlockReceived time.Time

	mutex sync.Mutex
	uicmddone chan bool = make(chan bool, 1)
	netBlocks chan *blockRcvd = make(chan *blockRcvd, 300)
	uiChannel chan oneUiReq = make(chan oneUiReq, 1)

	pendingBlocks map[[core.Uint256IdxLen]byte] *core.Uint256 = make(map[[core.Uint256IdxLen]byte] *core.Uint256, 600)
	pendingFifo chan [core.Uint256IdxLen]byte = make(chan [core.Uint256IdxLen]byte, PendingFifoLen)

	cachedBlocks map[[core.Uint256IdxLen]byte] *core.Block = make(map[[core.Uint256IdxLen]byte] *core.Block)
	receivedBlocks map[[core.Uint256IdxLen]byte] int64 = make(map[[core.Uint256IdxLen]byte] int64, 300e3)

	MyWallet *oneWallet

	Counter map[string] uint64 = make(map[string]uint64)

	busy string

	TransactionsToSend map[[32]byte] []byte = make(map[[32]byte] []byte)

)


type blockRcvd struct {
	conn *o
}


func CountSafe(k string) {
	mutex.Lock()
	Counter[k]++
	mutex.Unlock()
}