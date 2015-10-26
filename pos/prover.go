package pos

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/kwonalbert/spacecoin/util"
	"golang.org/x/crypto/sha3"
	"io/ioutil"
	"os"
)

type Prover struct {
	pk    []byte
	size  int    // # of vertices in the grpah
	graph string // directory containing the vertices

	commit []byte // root hash of the merkle tree
}

type Commitment struct {
	Pk     []byte
	Commit []byte
}

func NewProver(pk []byte, size int, graph string) *Prover {
	p := Prover{
		pk:    pk,
		size:  size,
		graph: graph,
	}
	return &p
}

func (p *Prover) computeHash(node string) []byte {
	nodeDir := fmt.Sprintf("%s/%s", p.graph, node)
	hf := fmt.Sprintf("%s/%s", nodeDir, hashName)
	f, err := os.Open(hf)
	if err == nil { // hash has been computed before
		hash := make([]byte, hashSize)
		n, err := f.Read(hash)
		if err != nil || n != hashSize {
			panic(err)
		}
		f.Close()
		return hash
	} else {
		parents, err := ioutil.ReadDir(nodeDir)
		if err != nil {
			panic(err)
		}

		buf := new(bytes.Buffer)
		binary.Write(buf, binary.BigEndian, node)
		val := append(p.pk, buf.Bytes()...)
		var hash [hashSize]byte

		if len(parents) == 0 { // source node
			hash = sha3.Sum256(val)
		} else {
			var ph []byte // parent hashes
			for _, file := range parents {
				if file.Name() == "hash" {
					continue
				}
				ph = append(ph, p.computeHash(file.Name())...)
			}
			hashes := append(val, ph...)
			hash = sha3.Sum256(hashes)
		}

		f, err = os.Create(hf)
		if err != nil {
			panic(err)
		}
		n, err := f.Write(hash[:])
		if err != nil || n != hashSize {
			panic(err)
		}
		f.Close()
		return hash[:]
	}
}

// Computes all the hashes of the vertices
func (p *Prover) Init() *Commitment {
	nodes, err := ioutil.ReadDir(p.graph)
	if err != nil {
		panic(err)
	}

	for _, file := range nodes {
		node := file.Name()
		p.computeHash(node)
	}

	return p.Commit()
}

// Recursive function to generate merkle tree
// Should have at most O(lgn) hashes in memory at a time
// return: hash at node i
func (p *Prover) generateMerkle(node int) []byte {
	if node >= p.size { // real vertices
		hf := fmt.Sprintf("%s/%d/%s", p.graph, node-p.size, hashName)
		f, err := os.Open(hf)
		if err != nil {
			panic(err)
		}
		hash := make([]byte, hashSize)
		n, err := f.Read(hash)
		if err != nil || n != hashSize {
			panic(err)
		}
		f.Close()
		return hash
	} else {
		hash1 := p.generateMerkle(node * 2)
		hash2 := p.generateMerkle(node*2 + 1)
		val := append(hash1[:], hash2[:]...)
		val = append(p.pk, val...)
		hash := sha3.Sum256(val)
		f, err := os.Create(fmt.Sprintf("%s/%s/%d", p.graph, "merkle", node))
		if err != nil {
			panic(err)
		}
		n, err := f.Write(hash[:])
		if err != nil || n != hashSize {
			panic(err)
		}
		f.Close()
		return hash[:]
	}
}

// Generate a merkle tree of the hashes of the vertices
// return: root hash of the merkle tree
//         will also write out the merkle tree
func (p *Prover) Commit() *Commitment {
	folder := fmt.Sprintf("%s/%s", p.graph, "merkle")
	err := os.Mkdir(folder, 0777)
	if err != nil {
		panic(err)
	}

	// build the merkle tree in depth first fashion
	// root node is 1
	root := p.generateMerkle(1)
	p.commit = root

	commit := &Commitment{
		Pk:     p.pk,
		Commit: root,
	}

	return commit
}

// return: hash of node, and the lgN hashes to verify node
func (p *Prover) Open(node int) ([]byte, [][]byte) {
	hash := make([]byte, hashSize)
	f, err := os.Open(fmt.Sprintf("%s/%d/hash", p.graph, node))
	if err != nil {
		panic(err)
	}
	n, err := f.Read(hash)
	if err != nil || n != hashSize {
		panic(err)
	}
	f.Close()

	proof := make([][]byte, util.Log2(p.size)-1)
	count := 0
	for i := node + p.size; i > 1; i /= 2 { // root hash not needed, so >1
		proof[count] = make([]byte, hashSize)
		var sib int

		if i%2 == 0 { // need to send only the sibling
			sib = i + 1
		} else {
			sib = i - 1
		}
		if sib >= p.size {
			f, err = os.Open(fmt.Sprintf("%s/%d/hash", p.graph, sib-p.size))
		} else {
			f, err = os.Open(fmt.Sprintf("%s/%s/%d", p.graph, "merkle", sib))
		}
		if err != nil {
			panic(err)
		}
		n, err := f.Read(proof[count])
		if err != nil || n != hashSize {
			panic(err)
		}
		count++
		f.Close()
	}
	return hash, proof
}

// Receives challenges from the verifier to prove PoS
// return: the hash values of the challenges, and the proof for each
func (p *Prover) ProveSpace(challenges []int) ([][]byte, [][][]byte) {
	hashes := make([][]byte, len(challenges))
	proofs := make([][][]byte, len(challenges))
	for i := range challenges {
		hashes[i], proofs[i] = p.Open(challenges[i])
	}
	return hashes, proofs
}
