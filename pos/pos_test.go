package pos

import (
	"crypto/rand"
	"fmt"
	"log"
	"os"
	"testing"
)

//exp* gets setup in test.go
var prover *Prover = nil
var verifier *Verifier = nil
var pk []byte
var size int = 5
var index int = 8
var beta int = 2
var graphDir string = "graph"

// func TestButterflyGraph(t *testing.T) {
// 	butterflyDir := "butterfly"
// 	os.RemoveAll(butterflyDir)
// 	os.Mkdir(butterflyDir, 0777)
// 	ButterflyGraph(0, 0, "C", butterflyDir)
// }

// func TestXiGraph(t *testing.T) {
// 	xiDir := "Xi"
// 	os.RemoveAll(xiDir)
// 	os.Mkdir(xiDir, 0777)
// 	XiGraph(3, 0, xiDir)
// }

func TestPoS(t *testing.T) {
	seed := make([]byte, 64)
	rand.Read(seed)
	challenges := verifier.SelectChallenges(seed)
	hashes, proofs := prover.ProveSpace(challenges)
	if !verifier.VerifySpace(challenges, hashes, proofs) {
		log.Fatal("Verify space failed:", challenges)
	}
}

func TestOpenVerify(t *testing.T) {
	hash, proof := prover.Open(1)
	for i := range expProof {
		for j := range expProof[i] {
			if expProof[i][j] != proof[i][j] {
				log.Fatal("Open failed:", expProof[i], proof[i])
			}
		}
	}

	if !verifier.Verify(1, hash, proof) {
		log.Fatal("Verify failed:", hash, proof)
	}
}

//Sanity check using simple graph
//[0 0 0 0 0]
//[1 0 0 0 0]
//[1 0 0 0 0]
//[1 0 1 0 0]
//[0 1 0 1 0]
func TestComputeHash(t *testing.T) {
	hashes := make([][]byte, size)
	for i := range hashes {
		f, _ := os.Open(fmt.Sprintf("%s/%d/hash", graphDir, i))
		hashes[i] = make([]byte, hashSize)
		f.Read(hashes[i])
	}

	var result [hashSize]byte

	for i := range expHashes {
		copy(result[:], hashes[i])
		if expHashes[i] != result {
			log.Fatal("Hash mismatch:", expHashes[i], result)
		}

	}
}

func TestMerkleTree(t *testing.T) {
	result := make([][hashSize]byte, 2*index)
	for i := 1; i < index; i++ {
		f, _ := os.Open(fmt.Sprintf("%s/merkle/%d", graphDir, i))
		buf := make([]byte, hashSize)
		f.Read(buf)
		copy(result[i][:], buf)
	}
	for i := 0; i < index; i++ {
		f, err := os.Open(fmt.Sprintf("%s/%d/hash", graphDir, i))
		if err == nil {
			buf := make([]byte, hashSize)
			f.Read(buf)
			copy(result[i+index][:], buf)
		} // if no such node exists, then just consider hash to be 0
	}

	for i := 2*index - 1; i > 0; i-- {
		if expMerkle[i] != result[i] {
			log.Fatal("Merkle node mismatch:", i, expMerkle[i], result[i])
		}
	}

}

func TestMain(m *testing.M) {
	pk = []byte{1}
	Setup(pk, size, index, graphDir)
	prover = NewProver(pk, index, graphDir)
	commit := prover.Init()
	root := commit.Commit

	verifier = NewVerifier(pk, index, beta, root)
	os.Exit(m.Run())
}
