package core

import "runtime"


const (
	BlockMapInitLen = 300e3
)

var useRoutines int = 3 * runtime.NumCPU() // use fewer times more go-routines to optimize an idle time
var taskDone chan bool

func init() {
	taskDone = make(chan bool, useRoutines)
}

