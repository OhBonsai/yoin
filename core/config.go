package core

import "runtime"


const (
	MAX_BLOCK_SIZE = 1000000
	COIN = 1e8
	MAX_MONEY = 21000000 * COIN
	BlockMapInitLen = 300e3
)

var useRoutines int = 3 * runtime.NumCPU() // use fewer times more go-routines to optimize an idle time
var taskDone chan bool

func init() {
	taskDone = make(chan bool, useRoutines)
}

