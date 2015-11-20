package block

import (
	sign "crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"github.com/kwonalbert/spacemint/pos"
	"log"
	"os"
	"testing"
)

var chain *BlockChain = nil
var oldB *Block
var b *Block

func TestChain(t *testing.T) {
	chain.Add(oldB)
	chain.Add(b)
	b0, err := chain.Read(0)
	if err != nil {
		log.Fatal(err)
	}
	b1, err := chain.Read(1)
	if err != nil {
		log.Fatal(err)
	}
	log.Println(oldB)
	log.Println(b0)
	log.Println(b)
	log.Println(b1)
}

func TestMarshal(t *testing.T) {
	bin, _ := b.MarshalBinary()
	bPrime := new(Block)
	bPrime.UnmarshalBinary(bin)
	//log.Println("Marshal result:", b, bPrime)
}

func TestMain(m *testing.M) {
	os.Remove("block.chain")
	chain = NewBlockChain("block.chain")

	pk := []byte{1}
	pos.Setup(pk, 4, "../pos/graph")
	prover := pos.NewProver(pk, 4, "../pos/graph")
	commit := prover.Init()
	pos := PoS{
		Commit:    *commit,
		Challenge: []byte{2},
		Answer: Answer{
			Size:   4,
			Hashes: [][]byte{[]byte{3}, []byte{4}},
			Proofs: nil,
		},
		Quality: 1.3,
	}

	ts := make([]Transaction, 2)

	sk, _ := sign.GenerateKey(elliptic.P256(), rand.Reader)
	oldB = &Block{
		Id:    0,
		Hash:  Hash{},
		Trans: make([]Transaction, 3),
		Sig:   Signature{},
	}
	b = NewBlock(oldB, pos, ts, sk)
	os.Exit(m.Run())
}
