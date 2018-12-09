package client

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
	"runtime"
	"github.com/OhBonsai/yoin/core"
	"sort"
	"strconv"
	"github.com/piotrnar/gocoin/btc"
)

type oneUiCmd struct {
	cmds    []string
	help    string
	sync    bool
	handler func(pars string)
}

type oneUiReq struct {
	param   string
	handler func(pars string)
}

var uiCmds []*oneUiCmd

func newUi(cmds string, sync bool, hn func(string), help string) {
	cs := strings.Split(cmds, " ")

	if len(cs[0]) > 0 {
		var c = new(oneUiCmd)
		for i := range cs {
			c.cmds = append(c.cmds, cs[i])
		}
		c.sync = sync
		c.help = help
		c.handler = hn
		uiCmds = append(uiCmds, c)
	} else {
		panic("empty command string")
	}
}

func readline() string {
	li, _, _ := bufio.NewReader(os.Stdin).ReadLine()
	return string(li)
}

func askYesNo(msg string) bool {
	for {
		fmt.Print(msg, " (y/n) : ")
		l := strings.ToLower(readline())
		if l == "y" {
			return true
		} else if l == "n" {
			return false
		}
	}
	return false
}

func doUserIf() {
	var prompt bool = true
	time.Sleep(5e8)
	for {
		if prompt {
			fmt.Print("> ")
		}
		prompt = true
		li := strings.Trim(readline(), " \n\t\r")
		if len(li) > 0 {
			cmdpar := strings.SplitN(li, " ", 2)
			cmd := cmdpar[0]
			param := ""
			if len(cmdpar)==2 {
				param = cmdpar[1]
			}
			found := false
			for i := range uiCmds {
				for j := range uiCmds[i].cmds {
					if cmd==uiCmds[i].cmds[j] {
						found = true
						if uiCmds[i].sync {
							mutex.Lock()
							if busy!="" {
								print("now busy with ", busy)
							}
							mutex.Unlock()
							println("...")
							sta := time.Now().UnixNano()
							uiChannel <- oneUiReq{param:param, handler:uiCmds[i].handler}
							go func() {
								_ = <- uicmddone
								sto := time.Now().UnixNano()
								fmt.Printf("Ready in %.3fs\n", float64(sto-sta)/1e9)
								fmt.Print("> ")
							}()
							prompt = false
						} else {
							uiCmds[i].handler(param)
						}
					}
				}
			}
			if !found {
				fmt.Printf("Unknown command '%s'. Type 'help' for help.\n", cmd)
			}
		}
	}
}

func showInfo(par string) {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	fmt.Println("Go version:", runtime.Version(), "   System Memory used:",
		ms.HeapAlloc>>20, "MB    NewBlockBeep:", beep)

	mutex.Lock()
	defer mutex.Unlock()
	// Main thread activity:
	if busy!="" {
		println("BlockChain thread is busy with", busy)
	} else {
		println("BlockChain thread is currently idle")
	}
}

func showLast(par string) {
	mutex.Lock()
	defer mutex.Unlock()
	fmt.Println("LastBlock:", LastBlock.BlockHash.String())
	fmt.Printf("  Height: %d @ %s,  Difficulty: %.1f\n", LastBlock.Height,
		time.Unix(int64(LastBlock.Timestamp), 0).Format("2006/01/02 15:04:05"),
		core.GetDifficulty(LastBlock.Bits))
	if !LastBlockReceived.IsZero() {
		fmt.Println("  Received", time.Now().Sub(LastBlockReceived), "ago")
	}
}

func showCounter(par string) {
	mutex.Lock()
	defer mutex.Unlock()
	fmt.Printf("BlocksCached: %d,   BlocksPending: %d/%d,   NetQueueSize: %d,   NetConns: %d\n",
		len(cachedBlocks), len(pendingBlocks), len(pendingFifo), len(netBlocks), len(openCons))

	ck := make([]string, len(Counter))
	i := 0
	for k, _ := range Counter {
		ck[i] = k
		i++
	}
	sort.Strings(ck)

	var li string
	for i := range ck {
		k := ck[i]
		v := Counter[k]
		s := fmt.Sprint(k, ": ", v)
		if len(li)+len(s) >= 80 {
			fmt.Println(li)
			li = ""
		} else if li!="" {
			li += ",   "
		}
		li += s
	}
	if li != "" {
		fmt.Println(li)
	}
}



func uiBeep(par string) {
	if par=="1" || par=="on" || par=="true" {
		beep = true
	} else if par=="0" || par=="off" || par=="false" {
		beep = false
	}
	fmt.Println("beep:", beep)
}


func uiDbg(par string) {
	v, e := strconv.ParseUint(par, 10, 32)
	if e == nil {
		dbg = v
	}
	fmt.Println("dbg:", dbg)
}


func showInvs(par string) {
	mutex.Lock()
	fmt.Println(len(pendingBlocks), "pending invs")
	for _, v := range pendingBlocks {
		fmt.Println(v.String())
	}
	mutex.Unlock()
}


func showCached(par string) {
	for _, v := range cachedBlocks {
		fmt.Printf(" * %s -> %s\n", v.Hash.String(), btc.NewUint256(v.Parent).String())
	}
}


func showHelp(par string) {
	fmt.Println("There following commands are supported:")
	for i := range uiCmds {
		fmt.Print("   ")
		for j := range uiCmds[i].cmds {
			if j>0 {
				fmt.Print(", ")
			}
			fmt.Print(uiCmds[i].cmds[j])
		}
		fmt.Println(" -", uiCmds[i].help)
	}
	fmt.Println("All the commands are case sensitive.")
}


func init() {
	newUi("help h ?", false, showHelp, "Shows this help")
	newUi("info i", false, showInfo, "Shows general info")
	newUi("last l", false, showLast, "Show last block info")
	newUi("counters c", false, showCounter, "Show counters")
	newUi("beep", false, uiBeep, "Control beep when a new block is received (use param 0 or 1)")
	newUi("dbg", false, uiDbg, "Control debugs (use numeric parameter)")
	newUi("cach", false, showCached, "Show blocks cached in memory")
	newUi("invs", false, showInvs, "Show pending block inv's (ones waiting for data)")
}
