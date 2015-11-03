package pos

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"runtime/pprof"
)

const nodeSize = hashSize + 2*8 + 1

type Graph struct {
	fn     string
	db     *os.File
	merkle *os.File
	pow2   int
	size   int
}

type Node struct {
	H  []byte // hash at the file
	Ps []int  // parent node files
}

func (n *Node) MarshalBinary() ([]byte, error) {
	var h byte
	if n.H == nil {
		n.H = make([]byte, hashSize)
		h = 0
	} else {
		h = 1
	}

	buf := new(bytes.Buffer)
	num, err := buf.Write(n.H)
	if err != nil || num != len(n.H) {
		return nil, err
	}

	for len(n.Ps) != 2 {
		n.Ps = append(n.Ps, -1)
	}

	for _, p := range n.Ps {
		b := make([]byte, 8)
		binary.PutVarint(b, int64(p))
		_, err = buf.Write(b)
		if err != nil {
			return nil, nil
		}
	}
	return append(buf.Bytes(), h), nil
}

func (n *Node) UnmarshalBinary(data []byte) error {
	n.H = data[:hashSize]
	n.Ps = make([]int, 2)
	if data[len(data)-1] == 0 {
		n.H = nil
	}
	count := 0
	for i := range n.Ps {
		start := hashSize + i*8
		end := hashSize + (i+1)*8
		parent, num := binary.Varint(data[start:end])
		if num < 0 {
			panic("couldn't read a parent")
		}
		if parent != -1 {
			n.Ps[count] = int(parent)
			count++
		}
	}
	n.Ps = n.Ps[:count]
	return nil
}

func (n *Node) UpdateParents(parents []int) {
	n.Ps = append(n.Ps, parents...)
}

// Generate a new PoS graph of index
// Currently only supports the weaker PoS graph
// Note that this graph will have O(2^index) nodes
func NewGraph(index, size, pow2 int, name, fn string) *Graph {
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
		fn:     fn,
		db:     db,
		merkle: merkle,
		size:   size,
		pow2:   pow2,
	}

	g.XiGraph(index, &count)

	return g
}

func (g *Graph) NewNode(id int, hash []byte, ps []int) {
	if id >= 0 {
		node := &Node{
			H:  hash,
			Ps: ps,
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
		if data != nil {
			node.UnmarshalBinary(data)
			return node
		} else {
			return nil
		}
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
	begin := *count
	numLevel := 2 * index
	perLevel := int(1 << uint(index))
	for level := 0; level < numLevel; level++ {
		for i := 0; i < perLevel; i++ {
			// no parents at level 0
			if level == 0 {
				g.NewNode(*count, nil, nil)
				*count++
				continue
			}
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
			parent1 := begin + (level-1)*perLevel + prev
			parent2 := *count - perLevel

			parents := []int{parent1, parent2}
			g.NewNode(*count, nil, parents)
			*count++
		}
	}
}

func (g *Graph) XiGraph(index int, count *int) {
	if index == 1 {
		g.ButterflyGraph(index, count)
		return
	}

	pow2index := 1 << uint(index)
	sources := *count
	firstButter := sources + pow2index
	firstXi := firstButter + numButterfly(index-1)
	secondXi := firstXi + numXi(index-1)
	secondButter := secondXi + numXi(index-1)
	sinks := secondButter + numButterfly(index-1)

	//graph generation
	for i := 0; i < pow2index; i++ {
		// "SO" for sources
		g.NewNode(*count, nil, nil)
		*count++
	}

	// recursively generate graphs
	g.ButterflyGraph(index-1, count)
	g.XiGraph(index-1, count)
	g.XiGraph(index-1, count)
	g.ButterflyGraph(index-1, count)

	for i := 0; i < pow2index; i++ {
		// "SI" for sinks
		g.NewNode(*count, nil, nil)
		*count++
	}

	pow2index_1 := int(1 << uint(index-1))

	// sources to sources of first butterfly
	for i := 0; i < pow2index_1; i++ {
		nodeId := firstButter + i
		parent0 := sources + i
		parent1 := sources + i + pow2index_1
		node := g.GetNode(nodeId)
		node.UpdateParents([]int{parent0, parent1})
		g.WriteNode(node, nodeId)
	}

	// sinks of first butterfly to sources of first xi graph
	for i := 0; i < pow2index_1; i++ {
		nodeId := firstXi + i
		// index is the last level; i.e., sinks
		parent := firstButter - pow2index_1 + i
		node := g.GetNode(nodeId)
		node.UpdateParents([]int{parent})
		g.WriteNode(node, nodeId)
	}

	// sinks of first xi to sources of second xi
	for i := 0; i < pow2index_1; i++ {
		nodeId := secondXi + i
		parent := secondXi - pow2index_1 + i
		node := g.GetNode(nodeId)
		node.UpdateParents([]int{parent})
		g.WriteNode(node, nodeId)
	}

	// sinks of second xi to sources of second butterfly
	for i := 0; i < pow2index_1; i++ {
		nodeId := secondButter + i
		parent := secondButter - pow2index_1 + i
		node := g.GetNode(nodeId)
		node.UpdateParents([]int{parent})
		g.WriteNode(node, nodeId)
	}

	// sinks of second butterfly to sinks
	for i := 0; i < pow2index_1; i++ {
		nodeId0 := sinks + i
		nodeId1 := sinks + i + pow2index_1
		parent := sinks - pow2index_1 + i
		node0 := g.GetNode(nodeId0)
		node0.UpdateParents([]int{parent})
		node1 := g.GetNode(nodeId1)
		node1.UpdateParents([]int{parent})
		g.WriteNode(node0, nodeId0)
		g.WriteNode(node1, nodeId1)
	}

	// sources to sinks directly
	for i := 0; i < int(1<<uint(index)); i++ {
		nodeId := sinks + i
		parent := sources + i
		node := g.GetNode(nodeId)
		node.UpdateParents([]int{parent})
		g.WriteNode(node, nodeId)
	}
}
