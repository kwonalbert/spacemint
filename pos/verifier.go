package pos

import (
	"crypto/rand"
	"golang.org/x/crypto/sha3"
	"math/big"
)

type Verifier struct {
	size            int    // size of the graph
	beta            int    // number of challenges needed
	root            []byte // root hash
}

func NewVerifier(size int, beta int, root []byte) *Verifier {
	v := Verifier{
		root:   root,
		beta:   beta,
		size:   size,
	}
	return &v
}

//TODO: need to select based on some pseudorandomness/gamma function?
func (v *Verifier) SelectChallenges() []int {
	challenges := make([]int, v.beta)
	size := big.NewInt(int64(v.size))
	for i := range challenges {
		r, err := rand.Int(rand.Reader, size)
		if err != nil {
			panic(err)
		}
		challenges[i] = int(r.Int64())
	}
	return challenges
}

func (v *Verifier) VerifySpace(challenges []int, hashes [][]byte, proofs [][][]byte) bool {
	for i := range challenges {
		if !v.Verify(challenges[i], hashes[i], proofs[i]) {
			return false
		}
	}
	return true
}

func (v *Verifier) Verify(node int, hash []byte, proof [][]byte) bool {
	curHash := hash
	counter := 0
	for i := node+v.size; i > 1; i /= 2 {
		var val []byte
		if i % 2 == 0 {
			val = append(curHash, proof[counter] ...)
		} else {
			val = append(proof[counter], curHash ...)
		}
		hash := sha3.Sum256(val)
		curHash = hash[:]
		counter++
	}

	for i := range v.root {
		if v.root[i] != curHash[i] {
			return false
		}
	}

	return len(v.root) == len(curHash)
}
