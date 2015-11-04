package pos

import (
	"encoding/binary"
	"fmt"
	"golang.org/x/crypto/sha3"
	"os"
	"runtime/pprof"
)

const nodeSize = hashSize

type Graph struct {
	pk     []byte
	fn     string
	db     *os.File
	merkle *os.File
	pow2   int
	size   int
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
func NewGraph(index, size, pow2 int, name, fn string, pk []byte) *Graph {
	cpuprofile := "graph.prof"
	f, _ := os.Create(cpuprofile)
	pprof.StartCPUProfile(f)
	defer pprof.StopCPUProfile()

	// recursively generate graphs
	count := 0

	db, err := os.Create(fn)
	if err != nil {
		panic(err)
	}

	merkle, err := os.Create(fmt.Sprintf("%s-merkle", fn))
	if err != nil {
		panic(err)
	}

	g := &Graph{
		pk:     pk,
		fn:     fn,
		db:     db,
		merkle: merkle,
		size:   size,
		pow2:   pow2,
	}

	g.XiGraph(index, &count)

	return g
}

func (g *Graph) NewNode(id int, hash []byte) {
	if id >= 0 {
		node := &Node{
			H: hash,
		}
		g.WriteNode(node, id)
	} else {
		num, err := g.merkle.WriteAt(hash, int64(-id)*hashSize)
		if err != nil || num != hashSize {
			panic(err)
		}
	}
}

// Gets the node, and update the node.
// Otherwise, create a node
func (g *Graph) GetNode(id int) *Node {
	node := new(Node)
	if id >= g.size {
		return nil
	} else if id >= 0 {
		data := make([]byte, nodeSize)
		num, err := g.db.ReadAt(data, int64(id)*nodeSize)
		if err != nil || num != nodeSize {
			panic(err)
		}
		node.UnmarshalBinary(data)
		return node
	} else {
		hash := make([]byte, hashSize)
		num, err := g.merkle.ReadAt(hash, int64(-id)*hashSize)
		if err != nil || num != hashSize {
			panic(err)
		}
		node.H = hash
		return node
	}
}

func (g *Graph) WriteNode(node *Node, id int) {
	b, err := node.MarshalBinary()
	if err != nil {
		panic(err)
	}
	num, err := g.db.WriteAt(b, int64(id)*nodeSize)
	if err != nil || num != nodeSize {
		panic(err)
	}
}

func (g *Graph) Close() {
	g.db.Close()
	g.merkle.Close()
}

func numXi(index int) int {
	return (1 << uint(index)) * (index + 1) * index
}

func numButterfly(index int) int {
	return 2 * (1 << uint(index)) * index
}

func (g *Graph) ButterflyGraph(index int, count *int) {
	numLevel := 2 * index
	perLevel := int(1 << uint(index))
	begin := *count - perLevel // level 0 created outside
	// no parents at level 0
	for level := 1; level < numLevel; level++ {
		for i := 0; i < perLevel; i++ {
			prev := 0
			shift := index - level
			if level > numLevel/2 {
				shift = level - numLevel/2
			}
			if (i>>uint(shift))&1 == 0 {
				prev = i + (1 << uint(shift))
			} else {
				prev = i - (1 << uint(shift))
			}
			parent0 := g.GetNode(begin + (level-1)*perLevel + prev)
			parent1 := g.GetNode(*count - perLevel)

			ph := append(parent0.H, parent1.H...)
			buf := make([]byte, hashSize)
			binary.PutVarint(buf, int64(*count))
			val := append(g.pk, buf...)
			val = append(val, ph...)
			hash := sha3.Sum256(val)

			g.NewNode(*count, hash[:])
			*count++
		}
	}
}

func (g *Graph) XiGraph(index int, count *int) {
	// recursively generate graphs
	// compute hashes along the way

	pow2index := 1 << uint(index)

	// the first sources
	// if index == 1, then this will generate level 0 of the butterfly
	for i := *count; i < pow2index; i++ {
		buf := make([]byte, hashSize)
		binary.PutVarint(buf, int64(*count))
		val := append(g.pk, buf...)
		hash := sha3.Sum256(val)

		g.NewNode(*count, hash[:])
		*count++
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
	pow2index_1 := int(1 << uint(index-1))

	// sources to sources of first butterfly
	// create sources of first butterly
	for i := 0; i < pow2index_1; i++ {
		parent0 := g.GetNode(sources + i)
		parent1 := g.GetNode(sources + i + pow2index_1)

		ph := append(parent0.H, parent1.H...)
		buf := make([]byte, hashSize)
		binary.PutVarint(buf, int64(*count))
		val := append(g.pk, buf...)
		val = append(val, ph...)
		hash := sha3.Sum256(val)

		g.NewNode(*count, hash[:])
		*count++
	}

	g.ButterflyGraph(index-1, count)
	// sinks of first butterfly to sources of first xi graph
	for i := 0; i < pow2index_1; i++ {
		nodeId := firstXi + i
		// index is the last level; i.e., sinks
		parent := g.GetNode(firstButter - pow2index_1 + i)

		buf := make([]byte, hashSize)
		binary.PutVarint(buf, int64(nodeId))
		val := append(g.pk, buf...)
		val = append(val, parent.H...)
		hash := sha3.Sum256(val)

		g.NewNode(nodeId, hash[:])
		*count++
	}

	g.XiGraph(index-1, count)
	// sinks of first xi to sources of second xi
	for i := 0; i < pow2index_1; i++ {
		nodeId := secondXi + i
		parent := g.GetNode(secondXi - pow2index_1 + i)

		buf := make([]byte, hashSize)
		binary.PutVarint(buf, int64(nodeId))
		val := append(g.pk, buf...)
		val = append(val, parent.H...)
		hash := sha3.Sum256(val)

		g.NewNode(nodeId, hash[:])
		*count++
	}

	g.XiGraph(index-1, count)
	// sinks of second xi to sources of second butterfly
	for i := 0; i < pow2index_1; i++ {
		nodeId := secondButter + i
		parent := g.GetNode(secondButter - pow2index_1 + i)

		buf := make([]byte, hashSize)
		binary.PutVarint(buf, int64(nodeId))
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
	for i := 0; i < pow2index_1; i++ {
		nodeId0 := sinks + i
		nodeId1 := sinks + i + pow2index_1
		parent0 := g.GetNode(sinks - pow2index_1 + i)
		parent1_0 := g.GetNode(sources + i)
		parent1_1 := g.GetNode(sources + i + pow2index_1)

		ph := append(parent0.H, parent1_0.H...)
		buf := make([]byte, hashSize)
		binary.PutVarint(buf, int64(nodeId0))
		val := append(g.pk, buf...)
		val = append(val, ph...)
		hash1 := sha3.Sum256(val)

		ph = append(parent0.H, parent1_1.H...)
		binary.PutVarint(buf, int64(nodeId1))
		val = append(g.pk, buf...)
		val = append(val, ph...)
		hash2 := sha3.Sum256(val)

		g.NewNode(nodeId0, hash1[:])
		g.NewNode(nodeId1, hash2[:])
		*count += 2
	}
}
