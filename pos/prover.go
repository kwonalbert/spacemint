package pos

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"golang.org/x/crypto/sha3"
	"io/ioutil"
	"os"
)

const hashName = "hash"
const hashSize = 256/8

type Prover struct {
	pk              []byte
	size            int    // # of vertices in the grpah
	graph           string // directory containing the vertices

	commitment      []byte // root hash of the merkle tree
}

func NewProver(pk []byte, size int, graph string) *Prover{
	p := Prover{
		pk:     pk,
		size:   size,
		graph:  graph,
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
		return hash
	} else {
		parents, err := ioutil.ReadDir(nodeDir)
		if err != nil {
			panic(err)
		}

		buf := new(bytes.Buffer)
		binary.Write(buf, binary.BigEndian, node)
		//probably should be val = append(buf.bytes(), p.pk ...)
		val := buf.Bytes()
		var hash [hashSize]byte

		if len(parents) == 0 { // source node
			hash = sha3.Sum256(val)
		} else {
			var ph []byte // parent hashes
			for _, file := range parents {
				if file.Name() == "hash" {
					continue
				}
				ph = append(ph, p.computeHash(file.Name()) ...)
			}
			hashes := append(val, ph ...)
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
		return hash[:]
	}
}

// Computes all the hashes of the vertices
func (p *Prover) InitGraph() {
	nodes, err := ioutil.ReadDir(p.graph)
	if err != nil {
		panic(err)
	}

	for _, file := range nodes {
		node := file.Name()
		p.computeHash(node)
	}

	p.Commit()
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
		return hash
	} else {
		hash1 := p.generateMerkle(node*2)
		hash2 := p.generateMerkle(node*2 + 1)
		val := append(hash1[:], hash2[:] ...)
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
func (p *Prover) Commit() []byte {
	folder := fmt.Sprintf("%s/%s", p.graph, "merkle")
	err := os.Mkdir(folder, 0777)
	if err != nil {
		panic(err)
	}

	// build the merkle tree in depth first fashion
	// root node is 1
	root := p.generateMerkle(1)
	p.commitment = root

	return root
}
