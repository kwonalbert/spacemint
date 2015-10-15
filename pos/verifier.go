package pos

import (
	"golang.org/x/crypto/sha3"
)

type Verifier struct {
	size            int
	root            []byte //root hash
}

func NewVerifier(size int, root []byte) *Verifier {
	v := Verifier{
		root:   root,
		size:   size,
	}
	return &v
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
