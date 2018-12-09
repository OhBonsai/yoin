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

func init(){
	newUi("bw", false, bwStats, "show network bandwidth statistics")
	newUi("ulimit ul", false, setUploadMax, "Set maximum upload speed. The value is in KB/second - 0 for unlimited")
	newUi("dlimit dl", false, setDownloadMax, "Set maximum download speed. The value is in KB/second - 0 for unlimited")}

func tickRecv() {
	now := time.Now().Unix()
	if now != dlLastSec {
		dlBytesPrvSec = dlBytesSoFar
		dlBytesSoFar = 0
		dlLastSec = now
	}
}

func countRcvd(n int) {
	bwMutex.Lock()
	tickRecv()
	dlBytesSoFar += n
	dlBytesTotal += uint64(n)
	bwMutex.Unlock()
}


func SockRead(con *net.TCPConn, buf[]byte) (n int, err error) {
	var toread int
	bwMutex.Lock()
	defer bwMutex.Unlock()

	if DownloadLimit == 0 {
		toread = len(buf)
	} else {
		toread = int(DownloadLimit) - dlBytesSoFar
		if toread > len(buf) {
			toread = len(buf)
		} else if toread < 0 {
			toread = 0
		}
	}

	dlBytesSoFar += toread
	dlBytesTotal += uint64(toread)
	bwMutex.Unlock()

	if toread > 0 {
		n, err = con.Read(buf[:toread])
	}
	return

}


func tickSent() {
	now := time.Now().Unix()
	if now != ulLastSec {
		ulBytesPrvSec = ulBytesSoFar
		ulBytesSoFar = 0
		ulLastSec = now
	}
}

func countSent(n int) {
	bwMutex.Lock()
	tickSent()
	ulBytesSoFar += n
	ulBytesTotal += uint64(n)
	bwMutex.Unlock()
}


func SockWrite(con *net.TCPConn, buf []byte) (n int, e error) {
	var tosend int
	bwMutex.Lock()
	tickSent()
	if UploadLimit == 0 {
		tosend = len(buf)
	}else {
		tosend = int(UploadLimit) - ulBytesSoFar
		if tosend > len(buf) {
			tosend = len(buf)
		} else if tosend < 0 {
			tosend = 0
		}
	}

	ulBytesSoFar += tosend
	ulBytesTotal += uint64(tosend)
	bwMutex.Unlock()

	if tosend > 0 {
		con.SetWriteDeadline(time.Now().Add(10 * time.Millisecond))
		n, e = con.Write(buf[:tosend])
		if e != nil {
			if nerr, ok := e.(net.Error); ok && nerr.Timeout() {
				e = nil
			}
		}
	}
	return
}

