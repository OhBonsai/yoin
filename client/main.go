package client

import (
	"flag"
	"github.com/OhBonsai/yoin/core"
	"github.com/piotrnar/gocoin/btc"
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
	BlockChain *btc.Chain
	AddrVersion byte

	exit_now bool
	dbg uint64
	beep bool

	LastBlock *core.BlockTreeNode
	LastBlockReceived time.Time

	mutex sync.Mutex
	uicmddone chan bool = make(chan bool, 1)
	netBlocks chan *blockRcvd = make(chan *blockRcvd, 300)

)


type blockRcvd struct {
	conn *o
}