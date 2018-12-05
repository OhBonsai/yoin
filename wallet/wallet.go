package main

import (
	"os"
	"fmt"
	"flag"
	"bytes"
	"bufio"
	"strconv"
	"strings"
	"math/big"
	"crypto/rand"
	"crypto/ecdsa"
	"encoding/hex"
	"github.com/OhBonsai/yoin/core"
)

type oneSendTo struct {
	addr *core.BtcAddr
	amount uint64
}

var (
	// Command line switches
	dump *bool = flag.Bool("l", false, "List public addressses from the wallet")
	noverify *bool = flag.Bool("q", false, "Do not verify keys while listing them")
	keycnt *uint = flag.Uint("n", 25, "Set the number of keys to be used")
	fee *float64 = flag.Float64("fee", 0.0005, "Transaction fee")
	send *string  = flag.String("send", "", "Send money to list of comma separated pairs: address=amount")
	change *string  = flag.String("change", "", "Send any change to this address (otherwise return to 1st input)")
	testnet *bool = flag.Bool("t", false, "Force work with testnet addresses")
	uncompressed *bool = flag.Bool("u", false, "Use uncompressed public keys")
	secfile *string  = flag.String("sec", "wallet.sec", "Read secret password (master seed) from this file")

	// set in load_balance():
	unspentOuts []*core.TxPrevOut
	unspentOutsLabel []string
	loadedTxs map[[32]byte] *core.Tx = make(map[[32]byte] *core.Tx)
	totBtc uint64

	verbyte, privver byte

	// set in make_wallet():
	priv_keys [][32]byte
	labels []string
	publ_addrs []*core.BtcAddr

	maxKeyVal *big.Int // used by verify_key

	// set in parse_spend():
	spendBtc, feeBtc, changeBtc uint64
	sendTo []oneSendTo

	curv *core.BitCurve = core.S256()
)


// Read a line from stdin
func getline() string {
	li, _, _ := bufio.NewReader(os.Stdin).ReadLine()
	return string(li)
}

// Input the password (that is the secret seed to your wallet)
func getpass() string {
	f, e := os.Open("wallet.sec")
	if e != nil {
		fmt.Print("Enter your wallet's seed password: ")
		pass := getline()
		if pass!="" && *dump {
			fmt.Print("Re-enter the seed password (to be sure): ")
			if pass!=getline() {
				println("The two passwords you entered do not match")
				os.Exit(1)
			}
		}
		return pass
	}
	le, _ := f.Seek(0, os.SEEK_END)
	buf := make([]byte, le)
	f.Seek(0, os.SEEK_SET)
	n, e := f.Read(buf[:])
	if e != nil {
		println("Reading secret file:", e.Error())
		os.Exit(1)
	}
	if int64(n)!=le {
		println("Something is wrong with the password file. Aborting.")
		os.Exit(1)
	}
	for i := range buf {
		if buf[i]<' ' || buf[i]>126 {
			fmt.Println("WARNING: Your secret contains non-printable characters\007")
			break
		}
	}
	return string(buf)
}

// get public key in bitcoin protocol format, from the give private key
func getPubKey(curv *core.BitCurve, priv_key []byte, compressed bool) (res []byte) {
	x, y := curv.ScalarBaseMult(priv_key)
	xd := x.Bytes()

	if len(xd)>32 {
		println("x is too long:", len(xd))
		os.Exit(2)
	}

	if !compressed {
		yd := y.Bytes()
		if len(yd)>32 {
			println("y is too long:", len(yd))
			os.Exit(2)
		}

		res = make([]byte, 65)
		res[0] = 4
		copy(res[1+32-len(xd):33], xd)
		copy(res[33+32-len(yd):65], yd)
	} else {
		res = make([]byte, 33)
		res[0] = 2+byte(y.Bit(0)) // 02 for even Y values, 03 for odd..
		copy(res[1+32-len(xd):33], xd)
	}

	return
}

func load_others() {
	f, e := os.Open("others.sec")
	if e == nil {
		defer f.Close()
		td := bufio.NewReader(f)
		for {
			li, _, _ := td.ReadLine()
			if li == nil {
				break
			}
			pk := strings.SplitN(strings.Trim(string(li), " "), " ", 2)
			pkb := core.Decodeb58(pk[0])
			if pkb == nil {
				println("Decodeb58 failed:", pk[0])
				continue
			}

			if len(pkb)!=37 && len(pkb)!=38 {
				println(pk[0], "has wrong key", len(pkb))
				println(hex.EncodeToString(pkb))
				continue
			}

			if pkb[0]!=privver {
				println(pk[0], "has version", pkb[0], "while we expect", privver)
				if pkb[0]==0xef {
					fmt.Println("We guess you meant testnet, so switching to testnet mode...")
					privver = 0xef
					verbyte = 0x6f
				} else {
					continue
				}
			}

			var sh [32]byte
			var compr bool

			if len(pkb)==37 {
				// compressed key
				//println(pk[0], "is compressed")
				sh = core.Sha2Sum(pkb[0:33])
				if !bytes.Equal(sh[:4], pkb[33:37]) {
					println(pk[0], "checksum error")
					continue
				}
				compr = false
			} else {
				if pkb[33]!=1 {
					println("we only support compressed keys of length 38 bytes", pk[0])
					continue
				}

				sh = core.Sha2Sum(pkb[0:34])
				if !bytes.Equal(sh[:4], pkb[34:38]) {
					println(pk[0], "checksum error")
					continue
				}
				compr = true
			}

			var key [32]byte
			copy(key[:], pkb[1:33])
			priv_keys = append(priv_keys, key)
			publ_addrs = append(publ_addrs, core.NewAddrFromPubKey(getPubKey(curv, key[:], compr), verbyte))
			if len(pk)>1 {
				labels = append(labels, pk[1])
			} else {
				labels = append(labels, fmt.Sprint("Other ", len(priv_keys)))
			}
		}
		fmt.Println(len(priv_keys), "keys imported")
	} else {
		fmt.Println("You can also have some dumped (b58 encoded) priv keys in 'others.sec'")
	}
}


// Get the secret seed and generate "*keycnt" key pairs (both private and public)
func make_wallet() {
	if *testnet {
		verbyte = 0x6f
		privver = 0xef
	} else {
		// verbyte is be zero by definition
		privver = 0x80
	}
	load_others()

	pass := getpass()
	seed_key := core.Sha2Sum([]byte(pass))
	if pass!="" {
		fmt.Println("Generating", *keycnt, "keys, version", verbyte,"...")
		for i:=uint(0); i < *keycnt; i++ {
			seed_key = core.Sha2Sum(seed_key[:])
			priv_keys = append(priv_keys, seed_key)
			publ_addrs = append(publ_addrs, core.NewAddrFromPubKey(getPubKey(curv, seed_key[:], !*uncompressed), verbyte))
			labels = append(labels, fmt.Sprint("Auto ", i+1))
		}
		fmt.Println("Private keys re-generated")
	}
}

// Verify the secret key's range and al if a test message signed with it verifies OK
func verify_key(priv []byte, publ []byte) bool {
	const TestMessage = "Just some test message..."
	hash := core.Sha2Sum([]byte(TestMessage))

	pub_key, e := core.NewPublicKey(publ)
	if e != nil {
		println("NewPublicKey:", e.Error(), "\007")
		os.Exit(1)
	}

	var key ecdsa.PrivateKey
	key.D = new(big.Int).SetBytes(priv)
	key.PublicKey = pub_key.PublicKey

	if key.D.Cmp(big.NewInt(0)) == 0 {
		println("pubkey value is zero")
		return false
	}

	if key.D.Cmp(maxKeyVal) != -1 {
		println("pubkey value is too big", hex.EncodeToString(publ))
		return false
	}

	r, s, err := ecdsa.Sign(rand.Reader, &key, hash[:])
	if err != nil {
		println("ecdsa.Sign:", err.Error())
		return false
	}

	ok := ecdsa.Verify(&key.PublicKey, hash[:], r, s)
	if !ok {
		println("The key pair does not verify!")
		return false
	}
	return true
}

// Print all the piblic addresses
func dump_addrs() {
	maxKeyVal, _ = new(big.Int).SetString("FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364141", 16)
	f, _ := os.Create("wallet.txt")
	for i := range publ_addrs {
		if !*noverify && !verify_key(priv_keys[i][:], publ_addrs[i].PubKey) {
			println("Something wrong with key at index", i, " - abort!\007")
			os.Exit(1)
		}
		fmt.Println(publ_addrs[i].String(), labels[i])
		if f != nil {
			fmt.Fprintln(f, publ_addrs[i].String(), labels[i])
		}
	}
	if f != nil {
		f.Close()
		fmt.Println("You can find all the addresses in wallet.txt file")
	}
}

// load the content of the "balance/" folder
func load_balance() {
	f, e := os.Open("balance/unspent.txt")
	if e != nil {
		println(e.Error())
		os.Exit(1)
	}
	rd := bufio.NewReader(f)
	for {
		l, _, e := rd.ReadLine()
		if len(l)==0 && e!=nil {
			break
		}
		if l[64]=='-' {
			txid := core.NewUint256FromString(string(l[:64]))
			rst := strings.SplitN(string(l[65:]), " ", 2)
			vout, _ := strconv.ParseUint(rst[0], 10, 32)
			uns := new(core.TxPrevOut)
			copy(uns.PreOutTxHash[:], txid.Hash[:])
			uns.OutIdxInTx = uint32(vout)
			lab := ""
			if len(rst)>1 {
				lab = rst[1]
			}

			if _, ok := loadedTxs[txid.Hash]; !ok {
				tf, _ := os.Open("balance/"+txid.String()+".tx")
				if tf != nil {
					siz, _ := tf.Seek(0, os.SEEK_END)
					tf.Seek(0, os.SEEK_SET)
					buf := make([]byte, siz)
					tf.Read(buf)
					tf.Close()
					th := core.Sha2Sum(buf)
					if bytes.Equal(th[:], txid.Hash[:]) {
						tx, _ := core.DeserializeTx(buf)
						if tx != nil {
							loadedTxs[txid.Hash] = tx
						} else {
							println("transaction is corrupt:", txid.String())
						}
					} else {
						println("transaction file is corrupt:", txid.String())
						os.Exit(1)
					}
				} else {
					println("transaction file not found:", txid.String())
					os.Exit(1)
				}
			}

			uo := UO(uns)
			fnd := false
			for j := range publ_addrs {
				if publ_addrs[j].Owns(uo.ScriptPubKey) {
					unspentOuts = append(unspentOuts, uns)
					unspentOutsLabel = append(unspentOutsLabel, lab)
					totBtc += UO(uns).Value
					fnd = true
					break
				}
			}

			if !fnd {
				fmt.Println(uns.String(), "does not belogn to your wallet - ignore it")
			}

		}
	}
	f.Close()
	fmt.Printf("You have %.8f BTC in %d unspent outputs\n", float64(totBtc)/1e8, len(unspentOuts))
}

// Get TxOut record, by the given TxPrevOut
func UO(uns *core.TxPrevOut) *core.TxOut {
	tx, _ := loadedTxs[uns.PreOutTxHash]
	return tx.TxOuts[uns.OutIdxInTx]
}


// parse the "-send ..." parameter
func parse_spend() {
	// No dump, so send money...
	outs := strings.Split(*send, ",")
	sendTo = make([]oneSendTo, len(outs))

	for i := range outs {
		tmp := strings.Split(strings.Trim(outs[i], " "), "=")
		if len(tmp)!=2 {
			println("The otputs must be in a format address1=amount1[,addressN=amountN]\007")
			os.Exit(1)
		}

		a, e := core.NewAddrFromString(tmp[0])
		if e != nil {
			println("NewAddrFromString:", e.Error(), "\007")
			os.Exit(1)
		}
		sendTo[i].addr = a

		am, e := strconv.ParseFloat(tmp[1], 64)
		if e != nil {
			println("ParseFloat:", e.Error(), "\007")
			os.Exit(1)
		}
		sendTo[i].amount = uint64(am*1e8)
		spendBtc += sendTo[i].amount
	}
	feeBtc = uint64(*fee*1e8)
}


// return the change addrress or nil if there will be no change
func get_change_addr() (chng *core.BtcAddr) {
	if *change!="" {
		var e error
		chng, e = core.NewAddrFromString(*change)
		if e != nil {
			println("Change address:", e.Error(), "\007")
			os.Exit(1)
		}
	}

	// If change address not specified, send it back to the first input
	uo := UO(unspentOuts[0])
	for j := range publ_addrs {
		if publ_addrs[j].Owns(uo.ScriptPubKey) {
			chng = publ_addrs[j]
			return
		}
	}

	fmt.Println("You do not own the address of the first input\007")
	os.Exit(1)
	return
}


// prepare a signed transaction
func make_signed_tx() {
	// Make an empty transaction
	tx := new(core.Tx)
	tx.Version = 1
	tx.LockTime = 0

	// Select as many inputs as we need to pay the full amount (with the fee)
	var btcsofar uint64
	var inpcnt uint
	for inpcnt=0; inpcnt<uint(len(unspentOuts)); inpcnt++ {
		uo := UO(unspentOuts[inpcnt])
		// add the input to our transaction:
		tin := new(core.TxIn)
		tin.Input = *unspentOuts[inpcnt]
		tin.Sequence = 0xffffffff
		tx.TxIns = append(tx.TxIns, tin)

		btcsofar += uo.Value
		if btcsofar >= spendBtc + feeBtc {
			break
		}
	}
	changeBtc = btcsofar - (spendBtc + feeBtc)
	fmt.Printf("Spending %d out of %d outputs...\n", inpcnt+1, len(unspentOuts))

	// Build transaction outputs:
	tx.TxOuts = make([]*core.TxOut, len(sendTo))
	for o := range sendTo {
		tx.TxOuts[o] = &core.TxOut{Value: sendTo[o].amount, ScriptPubKey: sendTo[o].addr.OutScript()}
	}

	if changeBtc > 0 {
		// Add one more output (with the change)
		tx.TxOuts = append(tx.TxOuts, &core.TxOut{Value: changeBtc, ScriptPubKey: get_change_addr().OutScript()})
	}

	//fmt.Println("Unsigned:", hex.EncodeToString(tx.Serialize()))

	for in := range tx.TxIns {
		uo := UO(unspentOuts[in])
		var found bool
		for j := range publ_addrs {
			if publ_addrs[j].Owns(uo.ScriptPubKey) {
				pub_key, e := core.NewPublicKey(publ_addrs[j].PubKey)
				if e != nil {
					println("NewPublicKey:", e.Error(), "\007")
					os.Exit(1)
				}

				// Load the key (private and public)
				var key ecdsa.PrivateKey
				key.D = new(big.Int).SetBytes(priv_keys[j][:])
				key.PublicKey = pub_key.PublicKey

				//Calculate proper transaction hash
				h := tx.SignatureHash(uo.ScriptPubKey, in, core.SIGHASH_ALL)
				//fmt.Println("SignatureHash:", core.NewUint256(h).String())

				// Sign
				r, s, err := ecdsa.Sign(rand.Reader, &key, h)
				if err != nil {
					println("Sign:", err.Error(), "\007")
					os.Exit(1)
				}
				rb := r.Bytes()
				sb := s.Bytes()

				if rb[0] >= 0x80 { // I thinnk this is needed, thought I am not quite sure... :P
					rb = append([]byte{0x00}, rb...)
				}

				if sb[0] >= 0x80 { // I thinnk this is needed, thought I am not quite sure... :P
					sb = append([]byte{0x00}, sb...)
				}

				// Output the signing result into a buffer, in format expected by bitcoin protocol
				busig := new(bytes.Buffer)
				busig.WriteByte(0x30)
				busig.WriteByte(byte(4+len(rb)+len(sb)))
				busig.WriteByte(0x02)
				busig.WriteByte(byte(len(rb)))
				busig.Write(rb)
				busig.WriteByte(0x02)
				busig.WriteByte(byte(len(sb)))
				busig.Write(sb)
				busig.WriteByte(0x01) // hash type

				// Output the signature and the public key into tx.ScriptSig
				buscr := new(bytes.Buffer)
				buscr.WriteByte(byte(busig.Len()))
				buscr.Write(busig.Bytes())

				buscr.WriteByte(byte(len(publ_addrs[j].PubKey)))
				buscr.Write(publ_addrs[j].PubKey)

				// assign:
				tx.TxIns[in].ScriptSig = buscr.Bytes()

				found = true
				break
			}
		}
		if !found {
			fmt.Println("You do not have private key for input number", hex.EncodeToString(uo.ScriptPubKey), "\007")
			os.Exit(1)
		}
	}

	rawtx := tx.Serialize()
	tx.Hash = core.NewSha2Hash(rawtx)

	hs := tx.Hash.String()
	fmt.Println(hs)

	f, _ := os.Create(hs[:8]+".txt")
	if f != nil {
		f.Write([]byte(hex.EncodeToString(rawtx)))
		f.Close()
		fmt.Println("Transaction data stored in", hs[:8]+".txt")
	}

	f, _ = os.Create("balance/unspent.txt")
	if f != nil {
		for j:=uint(0); j<uint(len(unspentOuts)); j++ {
			if j>inpcnt {
				fmt.Fprintln(f, unspentOuts[j], unspentOutsLabel[j])
			}
		}
		fmt.Println(inpcnt, "spent output(s) removed from 'balance/unspent.txt'")

		var addback int
		for out := range tx.TxOuts {
			for j := range publ_addrs {
				if publ_addrs[j].Owns(tx.TxOuts[out].ScriptPubKey) {
					fmt.Fprintf(f, "%s-%03d # %.8f / %s\n", tx.Hash.String(), out,
						float64(tx.TxOuts[out].Value)/1e8, publ_addrs[j].String())
					addback++
				}
			}
		}
		f.Close()
		if addback > 0 {
			f, _ = os.Create("balance/"+hs+".tx")
			if f != nil {
				f.Write(rawtx)
				f.Close()
			}
			fmt.Println(addback, "new output(s) appended to 'balance/unspent.txt'")
		}
	}
}


func main() {
	if flag.Lookup("h") != nil {
		flag.PrintDefaults()
		os.Exit(0)
	}
	flag.Parse()

	make_wallet()

	if *dump {
		dump_addrs()
		return
	}

	// If no dump, then it should be send money
	load_balance()

	if *send!="" {
		parse_spend()
		if spendBtc + feeBtc > totBtc {
			fmt.Println("You want to spend more than you own")
			return
		}
		make_signed_tx()
	}
}
