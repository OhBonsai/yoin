package client

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
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
			fmt.Print(">")
		}
	}
}
