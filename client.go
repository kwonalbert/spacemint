package main

import (
	"crypto"
	sign "crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"github.com/kwonalbert/spacecoin/block"
	"github.com/kwonalbert/spacecoin/pos"
	"github.com/kwonalbert/spacecoin/util"
	"golang.org/x/crypto/sha3"
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
	size     int
	prover   *pos.Prover
	verifier *pos.Verifier
	commit   pos.Commitment

	chain   *block.BlockChain
	clients []*rpc.Client
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
	commit := prover.Init()
	verifier := pos.NewVerifier(pkBytes, size, beta, commit.Commit)

	c := Client{
		sk:   sk,
		pk:   pk,
		t:    t,
		dist: dist,

		sols: make(chan *block.Block, 100), // nomially say 100 answers per round..

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

func (c *Client) Mine(challenge []byte) *block.PoS {
	nodes := c.verifier.SelectChallenges(challenge)
	hashes, proofs := c.prover.ProveSpace(nodes)
	a := block.Answer{
		Size:   c.size,
		Hashes: hashes,
		Proofs: proofs,
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
	if !c.verifier.VerifySpace(nodes, a.Hashes, a.Proofs) {
		return -1
	}

	all := util.Concat(a.Hashes)
	for i := range a.Proofs {
		all = append(all, util.Concat(a.Proofs[i])...)
	}
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

}
