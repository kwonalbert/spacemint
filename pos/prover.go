package pos

import (
	"encoding/binary"
	"fmt"
	"github.com/kwonalbert/spacecoin/util"
	"golang.org/x/crypto/sha3"
)

type Prover struct {
	pk    []byte
	graph *Graph // storage for all the graphs
	name  string

	commit []byte // root hash of the merkle tree

	index int // index of the graphy in the family; power of 2
	size  int // size of the graph
	pow2  int // next closest power of 2
	log2  int // next closest log
}

type Commitment struct {
	Pk     []byte
	Commit []byte
}

func NewProver(pk []byte, index int, name, graph string) *Prover {
	size := numXi(index)
	log2 := util.Log2(size) + 1
	pow2 := 1 << uint(log2)
	if (1 << uint(log2-1)) == size {
		log2--
		pow2 = 1 << uint(log2)
	}

	g := NewGraph(index, name, graph)

	p := Prover{
		pk:    pk,
		graph: g,
		name:  name,

		index: index,
		size:  size,
		pow2:  pow2,
		log2:  log2,
	}
	return &p
}

func (p *Prover) computeHash(nodeName string) []byte {
	node := p.graph.GetNode(nodeName, -1, nil, nil)
	if node.Hash != nil { // hash has been computed before
		return node.Hash
	} else {
		buf := make([]byte, hashSize)
		binary.PutVarint(buf, int64(node.Id))
		val := append(p.pk, buf...)
		var hash [hashSize]byte

		if len(node.Parents) == 0 { // source node
			hash = sha3.Sum256(val)
		} else {
			var ph []byte // parent hashes
			for _, parent := range node.Parents {
				ph = append(ph, p.computeHash(parent)...)
			}
			hashes := append(val, ph...)
			hash = sha3.Sum256(hashes)
		}
		node.Hash = hash[:]
		p.graph.Write(node, nodeName)
		err := p.graph.s.Flush()
		if err != nil {
			panic(err)
		}

		return hash[:]
	}
}

// Computes all the hashes of the vertices
func (p *Prover) Init() *Commitment {
	curGraph := fmt.Sprintf(graphBase, p.name, posName, p.index, 0)

	for i := 0; i < (1 << uint(p.index)); i++ {
		nodeFile := fmt.Sprintf(nodeBase, curGraph, SI, i)
		p.computeHash(nodeFile)
	}

	return p.Commit()
}

// Recursive function to generate merkle tree
// Should have at most O(lgn) hashes in memory at a time
// return: hash at node i
func (p *Prover) generateMerkle(node int) []byte {
	if node >= p.pow2 { // real vertices
		nodeName := IndexToNode(node-p.pow2, p.index, 0, p.name)
		n := p.graph.GetNode(nodeName, -1, nil, nil)
		if n.Hash == nil {
			return make([]byte, hashSize)
		} else {
			return n.Hash
		}
	} else {
		hash1 := p.generateMerkle(node * 2)
		hash2 := p.generateMerkle(node*2 + 1)
		val := append(hash1[:], hash2[:]...)
		val = append(p.pk, val...)
		hash := sha3.Sum256(val)

		nodeName := fmt.Sprintf("merkle/%d", node)
		n := p.graph.GetNode(nodeName, -1, hash[:], nil)
		p.graph.Write(n, nodeName)
		err := p.graph.s.Flush()
		if err != nil {
			panic(err)
		}

		return hash[:]
	}
}

// Generate a merkle tree of the hashes of the vertices
// return: root hash of the merkle tree
//         will also write out the merkle tree
func (p *Prover) Commit() *Commitment {
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
	var hash []byte
	nodeName := IndexToNode(node, p.index, 0, p.name)
	n := p.graph.GetNode(nodeName, -1, nil, nil)
	hash = n.Hash
	if hash == nil {
		hash = make([]byte, hashSize)
	}

	proof := make([][]byte, p.log2)
	count := 0
	for i := node + p.pow2; i > 1; i /= 2 { // root hash not needed, so >1
		var sib int

		if i%2 == 0 { // need to send only the sibling
			sib = i + 1
		} else {
			sib = i - 1
		}

		proof[count] = make([]byte, hashSize)
		if sib >= p.pow2 {
			nodeName = IndexToNode(sib-p.pow2, p.index, 0, p.name)
		} else {
			nodeName = fmt.Sprintf("merkle/%d", sib)
		}
		node := p.graph.GetNode(nodeName, -1, nil, nil)
		if node.Hash != nil {
			proof[count] = node.Hash
		}
		count++
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
		//TODO: open parents also
	}
	return hashes, proofs
}
