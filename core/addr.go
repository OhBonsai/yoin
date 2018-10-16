package core


type BtcAddr struct {
	Version byte
	Hash160 [20]byte
	Checksum []byte
	Pubkey []byte
	Enc58str string
}