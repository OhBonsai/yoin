package core

import (
	"math/big"
)


const (
	nTargetTimespan = 14 * 24 * 60 * 60 // two weeks
	nTargetSpacing = 10 * 60
	nInterval = nTargetTimespan / nTargetSpacing
	nProofOfWorkLimit = 0x1d00ffff
)

var (
	bnProofOfWorkLimit *big.Int
	testnet bool
)


