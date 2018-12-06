package client

import (
	"net"
	"sync"
	"strconv"
	"fmt"
	"time"
)

var (
	bwMutex sync.Mutex

	dlLastSec int64
	dlBytesSoFar int
	dlBytesPrvSec int
	dlBytesTotal uint64

	UploadLimit uint
	DownloadLimit uint


	ulLastSec int64
	ulBytesSoFar int
	ulBytesPrvSec int
	ulBytesTotal uint64
)


func setUploadMax(par string) {
	v, e := strconv.ParseUint(par, 10, 64)
	if e == nil {
		UploadLimit = uint(v << 10)
	}

	if UploadLimit!=0 {
		fmt.Printf("Current upload limit is %d KB/s\n", UploadLimit >> 10)
	} else {
		fmt.Println("The upload speed is not limited")
	}
}

func setDownloadMax(par string) {
	v, e := strconv.ParseUint(par, 10, 64)
	if e == nil {
		DownloadLimit = uint(v<<10)
	}
	if DownloadLimit != 0 {
		fmt.Printf("Current upload limit is %d KB/s\n", DownloadLimit>>10)
	} else {
		fmt.Println("The upload speed is not limited")
	}
}

func bwStats(par string) {
	bwMutex.Lock()
	defer bwMutex.Unlock()
	tickRecv()
	tickSent()
	fmt.Printf("Downloading at %d/%d KB/s, %d KB total\n",
		dlBytesPrvSec>>10, DownloadLimit>>10, dlBytesTotal>>10)
	fmt.Printf("Uploading at %d/%d KB/s, %d KB total\n",
		ulBytesPrvSec>>10, UploadLimit>>10, ulBytesTotal>>10)

}

func tickRecv() {
	now := time.Now().Unix()
	if now != dlLastSec {
		dlBytesPrvSec = dlBytesSoFar
		dlBytesSoFar = 0
		dlLastSec = now
	}
}

func tickSent() {
	now := time.Now().Unix()
	if now != ulLastSec {
		ulBytesPrvSec = ulBytesSoFar
		ulBytesSoFar = 0
		ulLastSec = now
	}
}


func SockRead(con *net.TCPConn, buf[]byte) (n int, err error) {
	var toread int
	bwMutex.Lock()
	defer bwMutex.Unlock()

}