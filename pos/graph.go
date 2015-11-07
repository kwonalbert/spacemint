package pos

import (
	"encoding/binary"
	//"fmt"
	"github.com/kwonalbert/spacecoin/util"
	"golang.org/x/crypto/sha3"
	"os"
	"runtime/pprof"
)

const nodeSize = hashSize

type Graph struct {
	pk   []byte
	fn   string
	db   *os.File
	log2 int64
	pow2 int64
	size int64
}

type Node struct {
	H []byte // hash at the file
}

func (n *Node) MarshalBinary() ([]byte, error) {
	return n.H, nil
}

func (n *Node) UnmarshalBinary(data []byte) error {
	n.H = data
	return nil
}

// Generate a new PoS graph of index
// Currently only supports the weaker PoS graph
// Note that this graph will have O(2^index) nodes
func NewGraph(index, size, pow2, log2 int64, fn string, pk []byte) *Graph {
	cpuprofile := "graph.prof"
	f, _ := os.Create(cpuprofile)
	pprof.StartCPUProfile(f)
	defer pprof.StopCPUProfile()

	// recursively generate graphs
	var count int64 = pow2

	db, err := os.Create(fn)
	if err != nil {
		panic(err)
	}

	g := &Graph{
		pk:   pk,
		fn:   fn,
		db:   db,
		log2: log2,
		size: size,
		pow2: pow2,
	}

	g.XiGraph(index, &count)

	return g
}

func (g *Graph) NewNode(id int64, hash []byte) {
	node := &Node{
		H: hash,
	}
	g.WriteNode(node, id)
}

// Gets the node, and update the node.
// Otherwise, create a node
func (g *Graph) GetNode(id int64) *Node {
	idx := g.bfsToPost(id)
	node := new(Node)
	data := make([]byte, nodeSize)
	num, err := g.db.ReadAt(data, idx*nodeSize)
	if err != nil || num != nodeSize {
		panic(err)
	}
	node.H = data
	return node
}

func (g *Graph) WriteNode(node *Node, id int64) {
	idx := g.bfsToPost(id)
	num, err := g.db.WriteAt(node.H, idx*nodeSize)
	if err != nil || num != nodeSize {
		panic(err)
	}
}

func (g *Graph) Close() {
	g.db.Close()
}

func (g *Graph) subtree(node int64) int64 {
	level := (g.log2 + 1) - util.Log2(node)
	return int64((1 << uint64(level)) - 1)
}

func (g *Graph) bfsToPost(node int64) int64 {
	if node == 1 {
		return 2*g.pow2 - 1
	} else if node%2 == 0 { //left child
		return g.bfsToPost(node/2) - g.subtree(node) - 1
	} else {
		return g.bfsToPost(node/2) - 1
	}
}

func numXi(index int64) int64 {
	return (1 << uint64(index)) * (index + 1) * index
}

func numButterfly(index int64) int64 {
	return 2 * (1 << uint64(index)) * index
}

func (g *Graph) ButterflyGraph(index int64, count *int64) {
	numLevel := 2 * index
	perLevel := int64(1 << uint64(index))
	begin := *count - perLevel // level 0 created outside
	// no parents at level 0
	var level, i int64
	for level = 1; level < numLevel; level++ {
		for i = 0; i < perLevel; i++ {
			var prev int64
			shift := index - level
			if level > numLevel/2 {
				shift = level - numLevel/2
			}
			if (i>>uint64(shift))&1 == 0 {
				prev = i + (1 << uint64(shift))
			} else {
				prev = i - (1 << uint64(shift))
			}
			parent0 := g.GetNode(begin + (level-1)*perLevel + prev)
			parent1 := g.GetNode(*count - perLevel)

			ph := append(parent0.H, parent1.H...)
			buf := make([]byte, hashSize)
			binary.PutVarint(buf, *count)
			val := append(g.pk, buf...)
			val = append(val, ph...)
			hash := sha3.Sum256(val)

			g.NewNode(*count, hash[:])
			*count++
		}
	}
}

func (g *Graph) XiGraph(index int64, count *int64) {
	// recursively generate graphs
	// compute hashes along the way

	pow2index := int64(1 << uint64(index))

	// the first sources
	// if index == 1, then this will generate level 0 of the butterfly
	var i int64
	if *count == g.pow2 {
		for i = 0; i < pow2index; i++ {
			buf := make([]byte, hashSize)
			binary.PutVarint(buf, *count)
			val := append(g.pk, buf...)
			hash := sha3.Sum256(val)

			g.NewNode(*count, hash[:])
			*count++
		}
	}

	if index == 1 {
		g.ButterflyGraph(index, count)
		return
	}

	sources := *count - pow2index
	firstButter := sources + pow2index
	firstXi := firstButter + numButterfly(index-1)
	secondXi := firstXi + numXi(index-1)
	secondButter := secondXi + numXi(index-1)
	sinks := secondButter + numButterfly(index-1)
	pow2index_1 := int64(1 << uint64(index-1))

	// sources to sources of first butterfly
	// create sources of first butterly
	for i = 0; i < pow2index_1; i++ {
		parent0 := g.GetNode(sources + i)
		parent1 := g.GetNode(sources + i + pow2index_1)

		ph := append(parent0.H, parent1.H...)
		buf := make([]byte, hashSize)
		binary.PutVarint(buf, *count)
		val := append(g.pk, buf...)
		val = append(val, ph...)
		hash := sha3.Sum256(val)

		g.NewNode(*count, hash[:])
		*count++
	}

	g.ButterflyGraph(index-1, count)
	// sinks of first butterfly to sources of first xi graph
	for i = 0; i < pow2index_1; i++ {
		nodeId := firstXi + i
		// index is the last level; i.e., sinks
		parent := g.GetNode(firstButter - pow2index_1 + i)

		buf := make([]byte, hashSize)
		binary.PutVarint(buf, nodeId)
		val := append(g.pk, buf...)
		val = append(val, parent.H...)
		hash := sha3.Sum256(val)

		g.NewNode(nodeId, hash[:])
		*count++
	}

	g.XiGraph(index-1, count)
	// sinks of first xi to sources of second xi
	for i = 0; i < pow2index_1; i++ {
		nodeId := secondXi + i
		parent := g.GetNode(secondXi - pow2index_1 + i)

		buf := make([]byte, hashSize)
		binary.PutVarint(buf, nodeId)
		val := append(g.pk, buf...)
		val = append(val, parent.H...)
		hash := sha3.Sum256(val)

		g.NewNode(nodeId, hash[:])
		*count++
	}

	g.XiGraph(index-1, count)
	// sinks of second xi to sources of second butterfly
	for i = 0; i < pow2index_1; i++ {
		nodeId := secondButter + i
		parent := g.GetNode(secondButter - pow2index_1 + i)

		buf := make([]byte, hashSize)
		binary.PutVarint(buf, nodeId)
		val := append(g.pk, buf...)
		val = append(val, parent.H...)
		hash := sha3.Sum256(val)

		g.NewNode(nodeId, hash[:])
		*count++
	}

	// generate sinks
	// sinks of second butterfly to sinks
	// and sources to sinks directly
	g.ButterflyGraph(index-1, count)
	for i = 0; i < pow2index_1; i++ {
		nodeId0 := sinks + i
		nodeId1 := sinks + i + pow2index_1
		parent0 := g.GetNode(sinks - pow2index_1 + i)
		parent1_0 := g.GetNode(sources + i)
		parent1_1 := g.GetNode(sources + i + pow2index_1)

		ph := append(parent0.H, parent1_0.H...)
		buf := make([]byte, hashSize)
		binary.PutVarint(buf, nodeId0)
		val := append(g.pk, buf...)
		val = append(val, ph...)
		hash1 := sha3.Sum256(val)

		ph = append(parent0.H, parent1_1.H...)
		binary.PutVarint(buf, nodeId1)
		val = append(g.pk, buf...)
		val = append(val, ph...)
		hash2 := sha3.Sum256(val)

		g.NewNode(nodeId0, hash1[:])
		g.NewNode(nodeId1, hash2[:])
		*count += 2
	}
}
