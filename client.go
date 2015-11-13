package main

import (
	"crypto"
	sign "crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/kwonalbert/spacecoin/block"
	"github.com/kwonalbert/spacecoin/pos"
	"github.com/kwonalbert/spacecoin/util"
	"golang.org/x/crypto/sha3"
	"log"
	"math"
	"math/big"
	//"net"
	"net/rpc"
	"time"
)

type Client struct {
	//client and system params
	sk   crypto.Signer    // signing secretkey
	pk   crypto.PublicKey // signing pubkey
	t    time.Duration    // how soon we add a block
	dist int              // how far to look back for challenge

	//round
	sols chan *block.Block // others' blocks

	//pos params
	index    int64
	prover   *pos.Prover
	verifier *pos.Verifier
	commit   pos.Commitment

	chain   *block.BlockChain
	clients []*rpc.Client
}

func NewClient(t time.Duration, dist, beta int, index int64, graph string) *Client {
	sk, err := sign.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		panic(err)
	}
	pk := sk.Public()
	pkBytes, err := json.Marshal(pk)
	if err != nil {
		panic(err)
	}

	prover := pos.NewProver(pkBytes, index, "Xi", graph)
	commit := prover.Init()
	verifier := pos.NewVerifier(pkBytes, index, beta, commit.Commit)

	c := Client{
		sk:   sk,
		pk:   pk,
		t:    t,
		dist: dist,

		sols: make(chan *block.Block, 100), // nomially say 100 answers per round..

		index:    index,
		prover:   prover,
		verifier: verifier,
		commit:   *commit,
	}
	return &c
}

func (c *Client) Sign(msg []byte) ([]byte, error) {
	return c.sk.Sign(rand.Reader, msg, crypto.SHA3_256)
}

func (c *Client) Mine(challenge []byte) *block.PoS {
	nodes := c.verifier.SelectChallenges(challenge)
	hashes, parents, proofs, pProofs := c.prover.ProveSpace(nodes)
	a := block.Answer{
		Size:    c.index,
		Hashes:  hashes,
		Parents: parents,
		Proofs:  proofs,
		PProofs: pProofs,
	}
	p := block.PoS{
		Commit:    c.commit,
		Challenge: challenge,
		Answer:    a,
		Quality:   c.Quality(challenge, a),
	}

	return &p
}

// Compute quality of the answer. Also builds a verifier
// return: quality in float64
func (c *Client) Quality(challenge []byte, a block.Answer) float64 {
	nodes := c.verifier.SelectChallenges(challenge)
	if !c.verifier.VerifySpace(nodes, a.Hashes, a.Parents, a.Proofs, a.PProofs) {
		return -1
	}

	all := util.Concat(a.Hashes)
	answerHash := sha3.Sum256(all)
	x := new(big.Float).SetInt(new(big.Int).SetBytes(answerHash[:]))
	num, _ := util.Root(x, a.Size).Float64()
	den := math.Exp2(float64(1<<8) / float64(a.Size))
	return num / den
}

// Generate challenge from older blocks
// return: challenge for next block []byte
func (c *Client) GenerateChallenge() []byte {
	var b *block.Block
	var err error
	if c.chain.LastBlock < c.dist {
		b, err = c.chain.Read(c.chain.LastBlock)
	} else {
		b, err = c.chain.Read(c.chain.LastBlock - (c.dist - 1))
	}
	if err != nil {
		panic(err)
	}
	bin, err := b.MarshalBinary()
	if err != nil {
		panic(err)
	}
	challenge := sha3.Sum256(bin)
	return challenge[:]
}

// Runs a round of the protocol
func (c *Client) round() {
	challenge := c.GenerateChallenge()
	prf := c.Mine(challenge)

	send := true

	for {
		select {
		case b := <-c.sols:
			// probably can't trust the others in final version..
			if b.Hash.Proof.Quality > prf.Quality {
				send = false
				break
			}
		default:
			break
		}
	}
	if send {
		old, err := c.chain.Read(c.chain.LastBlock)
		if err != nil {
			panic(err)
		}
		// TODO: where do transactions come from??
		b := block.NewBlock(old, *prf, nil, c.sk)
		for _, r := range c.clients {
			err := r.Call("Client.SendBlock", b, nil)
			if err != nil {
				panic(err)
			}
		}
	}
}

func main() {
	idx := flag.Int("index", 1, "graph index")
	name := flag.String("name", "Xi", "graph name")
	dir := flag.String("file", "/media/storage/Xi", "graph location")
	mode := flag.String("mode", "gen", "mode:[gen|commit]")
	flag.Parse()

	pk := []byte{1}
	beta := 30
	now := time.Now()
	prover := pos.NewProver(pk, int64(*idx), *name, *dir)
	if *mode == "gen" {
		fmt.Printf("%d. Graph gen: %fs\n", *idx, time.Since(now).Seconds())
	} else if *mode == "commit" {
		now = time.Now()
		prover.Init()
		fmt.Printf("%d. Graph commit: %fs\n", *idx, time.Since(now).Seconds())
	} else if *mode == "check" {
		commit := prover.PreInit()
		root := commit.Commit
		verifier := pos.NewVerifier(pk, int64(*idx), beta, root)

		seed := make([]byte, 64)
		rand.Read(seed)
		cs := verifier.SelectChallenges(seed)

		now = time.Now()
		hashes, parents, proofs, pProofs := prover.ProveSpace(cs)
		fmt.Printf("Prove: %f\n", time.Since(now).Seconds())

		now = time.Now()
		if !verifier.VerifySpace(cs, hashes, parents, proofs, pProofs) {
			log.Fatal("Verify space failed:", cs)
		}
		fmt.Printf("Verify: %f\n", time.Since(now).Seconds())
	}
}
