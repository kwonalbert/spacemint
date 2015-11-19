package pos

import (
	//"fmt"
	"github.com/kwonalbert/spacecoin/util"
	"golang.org/x/crypto/sha3"
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

	g := NewGraph(index, size, pow2, log2, graph, pk)

	empty := make(map[int64]bool)

	// if not power of 2, then uneven merkle
	if util.Count(uint64(size)) != 1 {
		for i := pow2 + size; util.Count(uint64(i+1)) != 1; i /= 2 {
			empty[i+1] = true
		}
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
	// build the merkle tree in depth first fashion
	// root node is 1
	root := p.generateMerkle()
	p.commit = root

	commit := &Commitment{
		Pk:     p.pk,
		Commit: root,
	}

	return commit
}

// Read the commitment from pre-initialized graph
func (p *Prover) PreInit() *Commitment {
	node := p.graph.GetId(2*p.pow2 - 1)
	p.commit = node.H
	commit := &Commitment{
		Pk:     p.pk,
		Commit: node.H,
	}
	return commit
}

func (p *Prover) emptyMerkle(node int64) bool {
	_, found := p.empty[node]
	return found
}

// Iterative function to generate merkle tree
// Should have at most O(lgn) hashes in memory at a time
// return: the root hash
func (p *Prover) generateMerkle() []byte {
	var stack []int64
	var hashStack [][]byte

	cur := int64(1)
	count := int64(1)

	for count == 1 || len(stack) != 0 {
		empty := p.emptyMerkle(cur)
		for ; cur < 2*p.pow2 && !empty; cur *= 2 {
			if cur < p.pow2 { //right child
				stack = append(stack, 2*cur+1)
			}
			stack = append(stack, cur)
		}

		if empty {
			count += p.graph.subtree(cur)
			hashStack = append(hashStack, make([]byte, hashSize))
		}

		cur, stack = stack[len(stack)-1], stack[:len(stack)-1]

		if len(stack) > 0 && cur < p.pow2 &&
			(stack[len(stack)-1] == 2*cur+1) {
			stack = stack[:len(stack)-1]
			stack = append(stack, cur)
			cur = 2*cur + 1
			continue
		}

		if cur >= p.pow2 {
			if cur >= p.pow2+p.size {
				hashStack = append(hashStack, make([]byte, hashSize))
				count++
			} else {
				n := p.graph.GetId(count)
				count++
				hashStack = append(hashStack, n.H)
			}
		} else if !p.emptyMerkle(cur) {
			hash2 := hashStack[len(hashStack)-1]
			hashStack = hashStack[:len(hashStack)-1]
			hash1 := hashStack[len(hashStack)-1]
			hashStack = hashStack[:len(hashStack)-1]
			val := append(hash1[:], hash2[:]...)
			hash := sha3.Sum256(val)

			hashStack = append(hashStack, hash[:])

			p.graph.NewNodeById(count, hash[:])
			count++
		}
		cur = 2 * p.pow2
	}

	return hashStack[0]
}

// Open a node in the merkle tree
// return: hash of node, and the lgN hashes to verify node
func (p *Prover) Open(node int64) ([]byte, [][]byte) {
	var hash []byte
	n := p.graph.GetNode(node + p.pow2)
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

		if sib >= p.pow2+p.size || p.emptyMerkle(sib) {
			proof[count] = make([]byte, hashSize)
			count++
			continue
		}

		n := p.graph.GetNode(sib)
		proof[count] = n.H
		count++
	}
	return hash, proof
}

// Receives challenges from the verifier to prove PoS
// return: the hash values of the challenges, the parent hashes,
//         the proof for each, and the proof for the parents
func (p *Prover) ProveSpace(challenges []int64) ([][]byte, [][][]byte, [][][]byte, [][][][]byte) {
	hashes := make([][]byte, len(challenges))
	proofs := make([][][]byte, len(challenges))
	parents := make([][][]byte, len(challenges))
	pProofs := make([][][][]byte, len(challenges))
	for i := range challenges {
		hashes[i], proofs[i] = p.Open(challenges[i])
		ps := p.graph.GetParents(challenges[i], p.index)
		for _, parent := range ps {
			if parent != -1 {
				hash, proof := p.Open(parent)
				parents[i] = append(parents[i], hash)
				pProofs[i] = append(pProofs[i], proof)
			}
		}
	}
	return hashes, parents, proofs, pProofs
}
