package main

import (
	"encoding/json"
	"crypto"
	sign "crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"github.com/kwonalbert/spacecoin/block"
	"github.com/kwonalbert/spacecoin/pos"
	"github.com/kwonalbert/spacecoin/util"
	"golang.org/x/crypto/sha3"
	"math"
	"math/big"
	"time"
)

type Client struct {
	sk              crypto.Signer     // signing secretkey
	pk              crypto.PublicKey  // signing pubkey
	t               time.Duration     // how soon we add a block
	dist            int               // how far to look back for challenge

	//pos params
	size            int
	prover          *pos.Prover
	verifier        *pos.Verifier
	commit          pos.Commitment
}

func NewClient(t time.Duration, dist int, size, beta int, graph string) *Client {
	sk, err := sign.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		panic(err)
	}
	pk := sk.Public()
	pkBytes, err := json.Marshal(pk)
	if err != nil {
		panic(err)
	}

	prover := pos.NewProver(pkBytes, size, graph)
	commit := prover.InitGraph()
	verifier := pos.NewVerifier(pkBytes, size, beta, commit.Commit)

	c := Client{
		sk:       sk,
		pk:       pk,
		t:        t,
		dist:     dist,

		size:     size,
		prover:   prover,
		verifier: verifier,
		commit:   *commit,
	}
	return &c
}

func (c *Client) Sign(msg []byte) ([]byte, error) {
	return c.sk.Sign(rand.Reader, msg, crypto.SHA3_256)
}

func (c *Client) Mine(challenge []byte) *block.PoS{
	nodes := c.verifier.SelectChallenges(challenge)
	hashes, proofs := c.prover.ProveSpace(nodes)
	a := block.Answer{
		Size:   c.size,
		Hashes: hashes,
		Proofs: proofs,
	}
	p := block.PoS{
		Commit: c.commit,
		Challenge: challenge,
		Answer: a,
		Quality: c.Quality(challenge, a),
	}

	return &p
}

// Compute quality of the answer. Also builds a verifier
// return: quality in float64
func (c Client) Quality(challenge []byte, a block.Answer) float64 {
	nodes := c.verifier.SelectChallenges(challenge)
	if !c.verifier.VerifySpace(nodes, a.Hashes, a.Proofs) {
		return -1
	}

	all := util.Concat(a.Hashes)
	for i := range a.Proofs {
		all = append(all, util.Concat(a.Proofs[i]) ...)
	}
	answerHash := sha3.Sum256(all)
	x := new(big.Float).SetInt(new(big.Int).SetBytes(answerHash[:]))
	num, _ := util.Root(x, a.Size).Float64()
	den := math.Exp2(float64(1 << 8)/float64(a.Size))
	return num/den
}


func main() {

}
