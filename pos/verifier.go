package pos

import (
	"bytes"
	"encoding/binary"
	//"fmt"
	"golang.org/x/crypto/sha3"
)

type Verifier struct {
	pk    []byte // public key to verify the proof
	index int    // index of the graphy in the family
	beta  int    // number of challenges needed
	root  []byte // root hash
}

func NewVerifier(pk []byte, index int, beta int, root []byte) *Verifier {
	v := Verifier{
		pk:    pk,
		index: index,
		beta:  beta,
		root:  root,
	}
	return &v
}

//TODO: need to select based on some pseudorandomness/gamma function?
//      Note that these challenges are different from those of cryptocurrency
func (v *Verifier) SelectChallenges(seed []byte) []int {
	challenges := make([]int, v.beta)
	rands := make([]byte, v.beta*8)
	sha3.ShakeSum256(rands, seed) //PRNG
	for i := range challenges {
		buf := bytes.NewBuffer(rands[i*8 : (i+1)*8-1])
		val, err := binary.ReadVarint(buf)
		if err != nil {
			panic(err)
		}
		challenges[i] = int(val) % v.index
		if challenges[i] < 0 {
			challenges[i] = v.index + challenges[i]
		}
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
	for i := node + v.index; i > 1; i /= 2 {
		var val []byte
		if i%2 == 0 {
			val = append(curHash, proof[counter]...)
		} else {
			val = append(proof[counter], curHash...)
		}
		val = append(v.pk, val...)
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
