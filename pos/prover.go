package pos

import (
	//"fmt"
	"github.com/kwonalbert/spacecoin/util"
	"golang.org/x/crypto/sha3"
	"os"
	"runtime/pprof"
)

type Prover struct {
	pk    []byte
	graph *Graph // storage for all the graphs
	name  string

	commit []byte // root hash of the merkle tree

	index int64 // index of the graphy in the family; power of 2
	size  int64 // size of the graph
	pow2  int64 // next closest power of 2
	log2  int64 // next closest log
	empty map[int64]bool
}

type Commitment struct {
	Pk     []byte
	Commit []byte
}

func NewProver(pk []byte, index int64, name, graph string) *Prover {
	size := numXi(index)
	log2 := util.Log2(size) + 1
	pow2 := int64(1 << uint64(log2))
	if (1 << uint64(log2-1)) == size {
		log2--
		pow2 = 1 << uint64(log2)
	}

	g := NewGraph(index, size, pow2, name, graph, pk)

	empty := make(map[int64]bool)

	for i := size; util.Count(uint64(i+1)) == 0; i /= 2 {
		empty[i+1] = true
	}

	p := Prover{
		pk:    pk,
		graph: g,
		name:  name,

		index: index,
		size:  size,
		pow2:  pow2,
		log2:  log2,
		empty: empty,
	}
	return &p
}

// Generate a merkle tree of the hashes of the vertices
// return: root hash of the merkle tree
//         will also write out the merkle tree
func (p *Prover) Init() *Commitment {
	f, _ := os.Create("prover.cpu")
	pprof.StartCPUProfile(f)
	defer pprof.StopCPUProfile()

	// build the merkle tree in depth first fashion
	// root node is 1
	root := p.generateMerkle(1)
	p.commit = root

	// f2, _ := os.Create("prover.mem")
	// pprof.WriteHeapProfile(f2)
	// f2.Close()

	commit := &Commitment{
		Pk:     p.pk,
		Commit: root,
	}

	return commit
}

func (p *Prover) emptyMerkle(node int64) bool {
	//_, found := p.empty[node]
	return false
}

// Recursive function to generate merkle tree
// Should have at most O(lgn) hashes in memory at a time
// return: hash at node i
func (p *Prover) generateMerkle(node int64) []byte {
	if node >= p.pow2 { // real vertices
		node = node - p.pow2
		if node >= p.size {
			return make([]byte, hashSize)
		} else {
			n := p.graph.GetNode(node)
			return n.H
		}
	} else if !p.emptyMerkle(node) {
		hash1 := p.generateMerkle(node * 2)
		hash2 := p.generateMerkle(node*2 + 1)
		val := append(hash1[:], hash2[:]...)
		val = append(p.pk, val...)
		hash := sha3.Sum256(val)

		p.graph.NewNode(-1*node, hash[:])

		return hash[:]
	} else {
		return make([]byte, hashSize)
	}
}

// return: hash of node, and the lgN hashes to verify node
func (p *Prover) Open(node int64) ([]byte, [][]byte) {
	var hash []byte
	n := p.graph.GetNode(node)
	if n != nil {
		hash = n.H
	} else {
		hash = make([]byte, hashSize)
	}

	proof := make([][]byte, p.log2)
	count := 0
	for i := node + p.pow2; i > 1; i /= 2 { // root hash not needed, so >1
		var sib int64

		if i%2 == 0 { // need to send only the sibling
			sib = i + 1
		} else {
			sib = i - 1
		}

		proof[count] = make([]byte, hashSize)
		if sib >= p.pow2 {
			node = sib - p.pow2
			if node >= p.size {
				proof[count] = n.H
				count++
				continue
			}
		} else {
			node = -1 * sib
		}
		n := p.graph.GetNode(node)
		if n != nil {
			proof[count] = n.H
		}
		count++
	}
	return hash, proof
}

// Receives challenges from the verifier to prove PoS
// return: the hash values of the challenges, and the proof for each
func (p *Prover) ProveSpace(challenges []int64) ([][]byte, [][][]byte) {
	hashes := make([][]byte, len(challenges))
	proofs := make([][][]byte, len(challenges))
	for i := range challenges {
		hashes[i], proofs[i] = p.Open(challenges[i])
		//TODO: open parents also
	}
	return hashes, proofs
}
