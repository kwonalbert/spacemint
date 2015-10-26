package block

import (
	"crypto"
	"crypto/rand"
	"encoding/json"
	"github.com/kwonalbert/spacecoin/pos"
	"github.com/kwonalbert/spacecoin/util"
	"golang.org/x/crypto/sha3"
)

type Block struct {
	Id      int
	Hash    Hash
	Trans   []Transaction
	Sig     Signature
}

type Hash struct {
	Hash    []byte // hash of previous block
	Proof   PoS
}

type PoS struct {
	Commit          pos.Commitment
	Challenge       []byte   // this round's challenge
	Answer          Answer   // answer to the challenge and proof
	Quality         float64  // quality of the answer
}

type Answer struct {
	Size    int
	Hashes  [][]byte
	Proofs  [][][]byte
}

type Signature struct {
	Tsig    []byte // signature on transaction i
	Ssig    []byte // signature on signature i-1
}


func NewBlock(old *Block, prf PoS, ts []Transaction, signer crypto.Signer) *Block {
	oldH, err := old.Hash.MarshalBinary()
	if err != nil {
		panic(err)
	}
	prevHash := sha3.Sum256(oldH)
	h := Hash{
		Hash: prevHash[:],
		Proof: prf,
	}

	var tsBytes []byte
	for i := range ts {
		b, err := ts[i].MarshalBinary()
		if err != nil {
			panic(err)
		}
		tsBytes = append(tsBytes, b ...)
	}
	sigBytes := util.Concat([][]byte{old.Sig.Tsig, old.Sig.Ssig})

	tsig, err := signer.Sign(rand.Reader, tsBytes, crypto.SHA3_256)
	if err != nil {
		panic(err)
	}
	ssig, err := signer.Sign(rand.Reader, sigBytes, crypto.SHA3_256)
	if err != nil {
		panic(err)
	}
	sig := Signature{
		Tsig: tsig,
		Ssig: ssig,
	}

	b := Block{
		Id: old.Id + 1,
		Hash: h,
		Trans: ts,
		Sig: sig,
	}
	return &b
}


func (b *Block) MarshalBinary() ([]byte, error) {
	return json.Marshal(b)
}

func (b *Block) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, b)
}

func (h *Hash) MarshalBinary() ([]byte, error) {
	return json.Marshal(h)
}

func (h *Hash) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, h)
}
