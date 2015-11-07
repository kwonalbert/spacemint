package pos

import (
	"crypto/rand"
	"flag"
	"fmt"
	"golang.org/x/crypto/sha3"
	"log"
	"os"
	"runtime"
	"testing"
	"time"
)

//exp* gets setup in test.go
var prover *Prover = nil
var verifier *Verifier = nil
var pk []byte
var index int64 = 15
var size int64 = 0
var beta int = 10
var graphDir string = "/media/storage/Xi"
var name string = "G"

func BenchmarkSha3(b *testing.B) {
	// run the Fib function b.N times
	buf := make([]byte, 2*hashSize)
	for n := 0; n < b.N; n++ {
		rand.Read(buf)
		sha3.Sum256(buf)
	}
}

func TestEmpty(t *testing.T) {

}

func TestPoS(t *testing.T) {
	seed := make([]byte, 64)
	rand.Read(seed)
	challenges := verifier.SelectChallenges(seed)
	now := time.Now()
	hashes, proofs := prover.ProveSpace(challenges)
	fmt.Printf("Prove: %f\n", time.Since(now).Seconds())

	now = time.Now()
	if !verifier.VerifySpace(challenges, hashes, proofs) {
		log.Fatal("Verify space failed:", challenges)
	}
	fmt.Printf("Verify: %f\n", time.Since(now).Seconds())
}

// //Sanity check using simple graph
// //[0 0 0 0 0]
// //[1 0 0 0 0]
// //[1 0 0 0 0]
// //[1 0 1 0 0]
// //[0 1 0 1 0]
// func TestComputeHash(t *testing.T) {
// 	hashes := make([][]byte, size)
// 	for i := range hashes {
// 		f, _ := os.Open(fmt.Sprintf("%s/%d/hash", graphDir, i))
// 		hashes[i] = make([]byte, hashSize)
// 		f.Read(hashes[i])
// 	}

// 	// var result [hashSize]byte

// 	// for i := range expHashes {
// 	// 	copy(result[:], hashes[i])
// 	// 	if expHashes[i] != result {
// 	// 		log.Fatal("Hash mismatch:", expHashes[i], result)
// 	// 	}

// 	// }
// }

// func TestMerkleTree(t *testing.T) {
// 	result := make([][hashSize]byte, 2*index)
// 	for i := 1; i < index; i++ {
// 		f, _ := os.Open(fmt.Sprintf("%s/merkle/%d", graphDir, i))
// 		buf := make([]byte, hashSize)
// 		f.Read(buf)
// 		copy(result[i][:], buf)
// 	}
// 	for i := 0; i < index; i++ {
// 		f, err := os.Open(fmt.Sprintf("%s/%d/hash", graphDir, i))
// 		if err == nil {
// 			buf := make([]byte, hashSize)
// 			f.Read(buf)
// 			copy(result[i+index][:], buf)
// 		} // if no such node exists, then just consider hash to be 0
// 	}

// 	for i := 2*index - 1; i > 0; i-- {
// 		if expMerkle[i] != result[i] {
// 			log.Fatal("Merkle node mismatch:", i, expMerkle[i], result[i])
// 		}
// 	}

// }

func TestMain(m *testing.M) {
	size = numXi(index)
	pk = []byte{1}

	runtime.GOMAXPROCS(runtime.NumCPU())

	id := flag.Int("index", 1, "graph index")
	flag.Parse()
	index = int64(*id)

	os.RemoveAll(graphDir)

	now := time.Now()
	prover = NewProver(pk, index, name, graphDir)
	fmt.Printf("%d. Graph gen: %fs\n", index, time.Since(now).Seconds())

	now = time.Now()
	commit := prover.Init()
	fmt.Printf("%d. Graph commit: %fs\n", index, time.Since(now).Seconds())

	root := commit.Commit
	verifier = NewVerifier(pk, index, beta, root)

	os.Exit(m.Run())
}
