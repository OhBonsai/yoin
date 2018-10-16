package core

import "runtime"

var useRoutines int = 3 * runtime.NumCPU() // use fewer times more go-routines to optimize an idle time
var taskDone chan bool

func init() {
	taskDone = make(chan bool, useRoutines)
}

