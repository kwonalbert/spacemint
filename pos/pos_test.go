package pos

import (
	"crypto/rand"
	"flag"
	"fmt"
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
var index int64 = 3
var size int64 = 0
var beta int = 30
var graphDir string = "Xi"
var name string = "G"

func TestPoS(t *testing.T) {
	seed := make([]byte, 64)
	rand.Read(seed)
	challenges := verifier.SelectChallenges(seed)
	now := time.Now()
	hashes, parents, proofs, pProofs := prover.ProveSpace(challenges)
	fmt.Printf("Prove: %f\n", time.Since(now).Seconds())

	now = time.Now()
	if !verifier.VerifySpace(challenges, hashes, parents, proofs, pProofs) {
		log.Fatal("Verify space failed:", challenges)
	}
	fmt.Printf("Verify: %f\n", time.Since(now).Seconds())
}

func TestMain(m *testing.M) {
	size = numXi(index)
	pk = []byte{1}

	runtime.GOMAXPROCS(runtime.NumCPU())

	id := flag.Int("index", 1, "graph index")
	flag.Parse()
	index = int64(*id)

	graphDir = fmt.Sprintf("%s%d", graphDir, *id)
	//os.RemoveAll(graphDir)

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
