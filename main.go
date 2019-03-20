package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/qshuai/tcolor"
)

const (
	// address pair
	base58Address = "mkYnutdmyXbTMk1QNgp3hcQa1pgGhXVY1S"
	privkey = "cNRuBb89ZA9UskAp3GsNW8o4PPsRjD5wY1JnQWwe6iL8td2bDXN8"

	// 1 satoshi/byte default
	feeRate = 1

	defaultSignatureSize = 107

	defaultSequence = 0xffffffff
)

type utxo struct{
	amount int64
	txHash *chainhash.Hash
	index int
	pkScript []byte
}

func main() {
	pkScript, err := getPkScript(base58Address)
	if err != nil {
		fmt.Println(tcolor.WithColor(tcolor.Red, "Please check your bech32 address: "+err.Error()))
		os.Exit(1)
	}

	// parse utxo
	txHash, err := chainhash.NewHashFromStr("1cf2928ba8ad02199cf320af8f6322269cb6ed8057349e2ee91a015f9fb54ab1")
	if err != nil {
		panic(err)
	}

	var u utxo
	u.txHash = txHash
	u.pkScript = pkScript
	u.amount = 110000
	u.index = 0

	// parse privkey
	wif, err := btcutil.DecodeWIF(privkey)
	if err != nil {
		fmt.Println(tcolor.WithColor(tcolor.Red, "Privkey format error: "+err.Error()))
		os.Exit(1)
	}

	// payto a multisig address, format: address: public key : private key
	// n1DEmZxGSa8yHvfSAYPtnMhb7MUsrjtsAE: 0334a8eb36d3add6a717cd1cde34f4cadc9e11154edecf2eb6f9bc50eee93ecf4c: cMxgWK5GhBfKFaqKzzrKnvCZQC8bZoo8ECHxmdo1gRFWAjgBJitv
	// mnTSmXwSFLYkw2DBwVzkVwd6eKEa26EVAs: 033a7bbfe0c777f5d3d1d89cd1479add8746df0a010fb80baf192ab2cb247c25a1: cTNd5Tt2TmtKvTEWu1dmpzgNnKt97qcDt4acgdpWZb72Yyey9gNK
	// mpuNzGVWEUXtMHpNngjwDkmzKwMTyhNH9W: 0386dbc13a9933a37103f1eb479c5a362eec91537c1361d8a34877a5576b3df5eb: cQdj8ZPBxHNfjsgqA6HRfXWMQhDaJMSRhNFPr77GQpYtAXzwZMr9
	pk1, _ := hex.DecodeString("0334a8eb36d3add6a717cd1cde34f4cadc9e11154edecf2eb6f9bc50eee93ecf4c")
	pk2, _ := hex.DecodeString("033a7bbfe0c777f5d3d1d89cd1479add8746df0a010fb80baf192ab2cb247c25a1")
	pk3, _ := hex.DecodeString("0386dbc13a9933a37103f1eb479c5a362eec91537c1361d8a34877a5576b3df5eb")
	builder := txscript.NewScriptBuilder()
	script, err := builder.AddOp(txscript.OP_2).AddData(pk1).AddData(pk2).AddData(pk3).AddOp(txscript.OP_3).AddOp(txscript.OP_CHECKMULTISIG).Script()
	if err != nil {
		panic(err)
	}

	tx, err := assembleTx(u, script, pkScript, wif)
	if err != nil {
		fmt.Println(tcolor.WithColor(tcolor.Red, "Assemble transaction or sign error:"+err.Error()))
		os.Exit(1)
	}

	buf := bytes.NewBuffer(nil)
	err = tx.Serialize(buf)
	if err != nil {
		fmt.Println(tcolor.WithColor(tcolor.Red, "Transaction serialize error:"+err.Error()))
		os.Exit(1)
	}
	// output result
	fmt.Println("txhash:         ", tcolor.WithColor(tcolor.Green, tx.TxHash().String()))
	fmt.Println("raw transaction:", tcolor.WithColor(tcolor.Green, hex.EncodeToString(buf.Bytes())))
}

func getPkScript(address string) ([]byte, error) {
	addr, err := btcutil.DecodeAddress(address, &chaincfg.TestNet3Params)
	if err != nil {
		return nil, err
	}

	return txscript.PayToAddrScript(addr)
}

func assembleTx(u utxo, multisig []byte, pkScript []byte, wif *btcutil.WIF) (*wire.MsgTx, error) {
	var tx wire.MsgTx
	tx.Version = 1
	tx.LockTime = 0

	tx.TxOut = make([]*wire.TxOut, 2)
	// total: 1100000, spend: 100000, change: 900000, fee: 100000
	tx.TxOut[0] = &wire.TxOut{PkScript: pkScript, Value: 900000}
	tx.TxOut[1] = &wire.TxOut{PkScript: multisig, Value: 100000}

	txIn := wire.TxIn{
		PreviousOutPoint: *wire.NewOutPoint(u.txHash, uint32(u.index)),
		Sequence:         defaultSequence,
	}
	tx.TxIn = append(tx.TxIn, &txIn)

	// sign the transaction
	return sign(&tx, 1100000, pkScript, wif)
}
