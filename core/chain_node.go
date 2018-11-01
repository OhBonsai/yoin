package core

import (
	"time"
	"fmt"
)

type BlockTreeNode struct {
	BlockHash *Uint256
	Height uint32
	Bits uint32
	Timestamp uint32
	ParentHash *Uint256
	Parent *BlockTreeNode
	Children []*BlockTreeNode  //链可以分岔的
}


// 解析到某个区块
func (ch *Chain) ParseTillBlock(end *BlockTreeNode) {
	var b []byte
	var er error
	var trusted bool

	prvSync := ch.DoNotSync
	ch.DoNotSync = true

	if end.Height - ch.BlockTreeEnd.Height > 100 {
		ch.Unspent.NoSync()  // 非常安全了
	}

	prv := time.Now().UnixNano()
	for ch.BlockTreeEnd != end {
		cur := time.Now().UnixNano()
		if cur-prv >= 10e9 {
			fmt.Println("ParseTillBlock ...", ch.BlockTreeEnd.Height, "/", end.Height)
			prv = cur
		}
		nxt := ch.BlockTreeEnd.FindNextNodePathTo(end)
		if nxt == nil {
			break
		}

		b, trusted, er = ch.Blocks.GetBlock(nxt.BlockHash)
		if er != nil {
			panic("Db.BlockGet(): "+er.Error())
		}

		bl, er := DeserializeBlock(b)
		if er != nil {
			ch.DeleteBranch(nxt)
			break
		}

		bl.Trusted = trusted

		bl.DecodeTxListFromRaw()

		changes, er := ch.ProcessBlockTransactions(bl, nxt.Height)
		if er != nil {
			println("ProcessBlockTransactions", nxt.Height, er.Error())
			ch.DeleteBranch(nxt)
			break
		}
		ch.Blocks.TrustBlock(bl.Hash.Hash[:])
		if !ch.DoNotSync {
			ch.Blocks.Sync()
		}
		ch.Unspent.CommitBlockTxs(changes, bl.Hash.Hash[:])

		ch.BlockTreeEnd = nxt
	}
	ch.Unspent.Sync()

	if ch.BlockTreeEnd != end {
		end, _ = ch.BlockTreeRoot.FindFarthestNode()
		fmt.Println("ParseTillBlock failed - now go to", end.Height)
		ch.MoveToBlock(end)
	}
	ch.DoNotSync = prvSync
}


// 从某个区块找到到最远的区块
func (n *BlockTreeNode) FindFarthestNode() (*BlockTreeNode, int) {
	if len(n.Children) == 0{
		return n, 0
	}

	res, depth := n.Children[0].FindFarthestNode()
	if len(n.Children) > 1 {
		for i:=1; i<len(n.Children); i++ {
			tRes, tDept := n.Children[i].FindFarthestNode()
			if tDept > depth {
				res, depth = tRes, tDept
			}
		}
	}
	return res, depth+1
}


// 找到指向某个节点的女儿节点。分岔的地方
func (n *BlockTreeNode) FindNextNodePathTo(to *BlockTreeNode) (*BlockTreeNode) {
	if to.Height <= n.Height {
		panic("End block is not higher then current")
	}

	if len(n.Children) == 0 { // 没儿子，怎么找到
		panic("Unknown path to block " + to.BlockHash.String() )
	}

	if n == to {  // 自己找自己，狗日的
		return nil
	}

	if len(n.Children) == 1 {
		return n.Children[0] // 只有一个女儿
	}

	// 从后往前走
	for {
		if to.Parent == n {
			return to
		}

		to = to.Parent
	}

	return nil
}

//
func (ch *Chain)MoveToBlock(dst *BlockTreeNode) {
	fmt.Printf("MoveToBlock: %d -> %\n", ch.BlockTreeEnd.Height, dst.Height)

	cur := dst
	for cur.Height > ch.BlockTreeEnd.Height {
		cur = cur.Parent
	}

	// At this point both "ch.BlockTreeEnd" and "cur" should be at the same height
	for ch.BlockTreeEnd != cur {
		if don(DBG_ORPHAS) {
			fmt.Printf("->orph block %s @ %d\n", ch.BlockTreeEnd.BlockHash.String(),
				ch.BlockTreeEnd.Height)
		}
		ch.Unspent.UndoBlockTransactions(ch.BlockTreeEnd.Height)
		ch.BlockTreeEnd = ch.BlockTreeEnd.Parent
		cur = cur.Parent
	}
	fmt.Printf("Reached common node @ %d\n", ch.BlockTreeEnd.Height)
	ch.ParseTillBlock(dst)
}


func (cur *BlockTreeNode) delAllChildren() {
	for i := range cur.Children {
		cur.Children[i].delAllChildren()
	}
}

func (ch *Chain) DeleteBranch(cur *BlockTreeNode) {
	// first disconnect it from the Parent
	ch.BlockIndexAccess.Lock()
	delete(ch.BlockIndex, cur.BlockHash.BIdx())
	cur.Parent.delChild(cur)
	cur.delAllChildren()
	ch.BlockIndexAccess.Unlock()
	ch.Blocks.InvalidBlock(cur.BlockHash.Hash[:])
	if !ch.DoNotSync {
		ch.Blocks.Sync()
	}
}


// 加女儿
func (n *BlockTreeNode) addChild(c *BlockTreeNode) {
	n.Children = append(n.Children, c)
}


// 打掉儿子
func (n *BlockTreeNode) delChild(c *BlockTreeNode) {
	tmpChildren := make([]*BlockTreeNode, len(n.Children) - 1)
	cursor := 0

	for _, child := range n.Children {
		if child != c {
			tmpChildren[cursor] = child
			cursor++
		}
	}

	if cursor != len(n.Children) - 1{
		panic("Child not found")
	}

	n.Children = tmpChildren
}