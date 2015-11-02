package pos

import (
	"bytes"
	"encoding/binary"
	//"fmt"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"os"
	"runtime/pprof"
)

type Graph struct {
	fn string
	db *leveldb.DB
}

type Node struct {
	H  []byte // hash at the file
	Ps []int  // parent node files
}

func (n *Node) MarshalBinary() ([]byte, error) {
	buf := new(bytes.Buffer)
	_, err := buf.Write(n.H)
	if err != nil {
		return nil, err
	}
	for _, p := range n.Ps {
		b := make([]byte, 8)
		binary.PutVarint(b, int64(p))
		_, err = buf.Write(b)
		if err != nil {
			return nil, nil
		}
	}
	return buf.Bytes(), nil
}

func (n *Node) UnmarshalBinary(data []byte) error {
	buf := bytes.NewBuffer(data)
	n.H = make([]byte, hashSize)
	n.Ps = make([]int, (len(data)-hashSize)/8)
	_, err := buf.Read(n.H)
	if err != nil {
		return err
	}
	for i := range n.Ps {
		parent, err := binary.ReadVarint(buf)
		if err != nil {
			return err
		}
		n.Ps[i] = int(parent)
	}
	return nil
}

func (n *Node) UpdateParents(parents []int) {
	n.Ps = append(n.Ps, parents...)
}

// Generate a new PoS graph of index
// Currently only supports the weaker PoS graph
// Note that this graph will have O(2^index) nodes
func NewGraph(index int, name, fn string) *Graph {
	cpuprofile := "graph.prof"
	f, _ := os.Create(cpuprofile)
	pprof.StartCPUProfile(f)
	defer pprof.StopCPUProfile()

	// recursively generate graphs
	count := 0

	opt := &opt.Options{
		WriteBuffer:        512 * opt.MiB,
		BlockCacheCapacity: 512 * opt.MiB,
	}

	db, err := leveldb.OpenFile(fn, opt)
	if err != nil {
		panic(err)
	}

	g := &Graph{
		fn: fn,
		db: db,
	}

	g.XiGraph(index, &count)

	return g
}

func (g *Graph) NewNode(id int, hash []byte, ps []int) {
	node := &Node{
		H:  hash,
		Ps: ps,
	}

	g.WriteNode(node, id)
}

// Gets the node, and update the node.
// Otherwise, create a node
func (g *Graph) GetNode(id int) *Node {
	node := new(Node)
	key := make([]byte, 8)
	binary.PutVarint(key, int64(id))
	data, err := g.db.Get(key, nil)
	if err == leveldb.ErrNotFound {
		return nil
	} else if err != nil {
		panic(err)
	}
	if data != nil {
		node.UnmarshalBinary(data)
		return node
	} else {
		return nil
	}
}

func (g *Graph) WriteNode(n *Node, id int) {
	b, err := n.MarshalBinary()
	if err != nil {
		panic(err)
	}
	key := make([]byte, 8)
	binary.PutVarint(key, int64(id))
	err = g.db.Put(key, b, nil)
	if err != nil {
		panic(err)
	}
}

func (g *Graph) Close() {
	g.db.Close()
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
				g.NewNode(*count, make([]byte, hashSize), nil)
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
			g.NewNode(*count, make([]byte, hashSize), parents)
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
		g.NewNode(*count, make([]byte, hashSize), nil)
		*count++
	}

	// recursively generate graphs
	g.ButterflyGraph(index-1, count)
	g.XiGraph(index-1, count)
	g.XiGraph(index-1, count)
	g.ButterflyGraph(index-1, count)

	for i := 0; i < pow2index; i++ {
		// "SI" for sinks
		g.NewNode(*count, make([]byte, hashSize), nil)
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
