package pos

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	//"fmt"
	"github.com/kwonalbert/spacecoin/util"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"os"
	"runtime/pprof"
	"strconv"
)

var graphBase string = "%s.%s%d-%d"
var nodeBase string = "%s.%d-%d"

const (
	SO = 0
	SI = 1
)

type Graph struct {
	fn string
	db *leveldb.DB
}

type Node struct {
	I  int      // node id
	H  []byte   // hash at the file
	Ps []string // parent node files
}

func MarshalNode(n *Node) ([]byte, error) {
	buf := new(bytes.Buffer)
	enc := gob.NewEncoder(buf)
	err := enc.Encode(n)
	return buf.Bytes(), err
}

func UnmarshalNode(n *Node, data []byte) {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	dec.Decode(n)
}

func (n *Node) MarshalBinary() ([]byte, error) {
	return json.Marshal(n)
}

func (n *Node) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, n)
}

func (n *Node) UpdateParents(parents []string) {
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

	db, err := leveldb.OpenFile(fn, nil)
	if err != nil {
		panic(err)
	}

	g := &Graph{
		fn: fn,
		db: db,
	}

	opt := &opt.Options{
		WriteBuffer:        512 * opt.MiB,
		BlockCacheCapacity: 512 * opt.MiB,
	}

	g.XiGraph(index, "0", name, &count)

	return g
}

func (g *Graph) NewNode(nodeName string, id int, hash []byte, ps []string) {
	node := &Node{
		I:  id,
		H:  hash,
		Ps: ps,
	}

	g.WriteNode(node, nodeName)
}

func GraphName(graph, name string, index int, inst string) string {
	return util.ConcatStr(graph, ".", name,
		strconv.Itoa(index), "-", inst)
}

func NodeName(graphName string, id1, id2 int) string {
	return util.ConcatStr(graphName, ".",
		strconv.Itoa(id1), "-", strconv.Itoa(id2))
}

// Gets the node, and update the node.
// Otherwise, create a node
func (g *Graph) GetNode(nodeName string) *Node {
	node := new(Node)
	data, err := g.db.Get([]byte(nodeName), nil)
	if err == leveldb.ErrNotFound {
		return nil
	} else if err != nil {
		panic(err)
	}
	if data != nil {
		//UnmarshalNode(node, data)
		node.UnmarshalBinary(data)
		return node
	} else {
		return nil
	}
}

func (g *Graph) WriteNode(n *Node, nodeName string) {
	//b, err := MarshalNode(n)
	b, err := n.MarshalBinary()
	if err != nil {
		panic(err)
	}
	err = g.db.Put([]byte(nodeName), b, nil)
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

// Maps a node index (0 to O(2^N)) to a folder (a physical node)
func IndexToNode(node, index int, inst, name string) string {
	sources := 1 << uint(index)
	firstButter := sources + numButterfly(index-1)
	firstXi := firstButter + numXi(index-1)
	secondXi := firstXi + numXi(index-1)
	secondButter := secondXi + numButterfly(index-1)
	sinks := secondButter + sources

	curGraph := GraphName(name, posName, index, inst)

	if node < sources {
		return NodeName(curGraph, SO, node)
	} else if node >= sources && node < firstButter {
		node = node - sources
		butterfly0 := GraphName(curGraph, "C", index-1, "0")
		level := node / (1 << uint(index-1))
		nodeNum := node % (1 << uint(index-1))
		return NodeName(butterfly0, level, nodeNum)
	} else if node >= firstButter && node < firstXi {
		node = node - firstButter
		return IndexToNode(node, index-1, "0", curGraph)
	} else if node >= firstXi && node < secondXi {
		node = node - firstXi
		return IndexToNode(node, index-1, "1", curGraph)
	} else if node >= secondXi && node < secondButter {
		node = node - secondXi
		butterfly1 := GraphName(curGraph, "C", index-1, "1")
		level := node / (1 << uint(index-1))
		nodeNum := node % (1 << uint(index-1))
		return NodeName(butterfly1, level, nodeNum)
	} else if node >= secondButter && node < sinks {
		node = node - secondButter
		return NodeName(curGraph, SI, node)
	} else {
		return ""
	}
}

func (g *Graph) ButterflyGraph(index int, inst, name, graph string, count *int) {
	curGraph := GraphName(graph, name, index, inst)
	numLevel := 2 * index
	for level := 0; level < numLevel; level++ {
		for i := 0; i < int(1<<uint(index)); i++ {
			// no parents at level 0
			nodeName := NodeName(curGraph, level, i)
			if level == 0 {
				g.NewNode(nodeName, *count, nil, nil)
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
			parent1 := NodeName(curGraph, level-1, prev)
			parent2 := NodeName(curGraph, level-1, i)

			parents := []string{parent1, parent2}
			g.NewNode(nodeName, *count, nil, parents)
			*count++
		}
	}
}

func (g *Graph) XiGraph(index int, inst, graph string, count *int) {
	if index == 1 {
		g.ButterflyGraph(index, inst, posName, graph, count)
		return
	}
	curGraph := GraphName(graph, posName, index, inst)

	for i := 0; i < int(1<<uint(index)); i++ {
		// "SO" for sources
		nodeName := NodeName(curGraph, SO, i)
		g.NewNode(nodeName, *count, nil, nil)
		*count++
	}

	// recursively generate graphs
	g.ButterflyGraph(index-1, "0", "C", curGraph, count)
	g.XiGraph(index-1, "0", curGraph, count)
	g.XiGraph(index-1, "1", curGraph, count)
	g.ButterflyGraph(index-1, "1", "C", curGraph, count)

	for i := 0; i < int(1<<uint(index)); i++ {
		// "SI" for sinks
		nodeName := NodeName(curGraph, SI, i)
		g.NewNode(nodeName, *count, nil, nil)
		*count++
	}

	offset := int(1 << uint(index-1)) //2^(index-1)

	// sources to sources of first butterfly
	butterfly0 := GraphName(curGraph, "C", index-1, "0")
	for i := 0; i < offset; i++ {
		nodeName := NodeName(butterfly0, 0, i)
		parent0 := NodeName(curGraph, SO, i)
		parent1 := NodeName(curGraph, SO, i+offset)
		node := g.GetNode(nodeName)
		node.UpdateParents([]string{parent0, parent1})
		g.WriteNode(node, nodeName)
	}

	// sinks of first butterfly to sources of first xi graph
	xi0 := GraphName(curGraph, posName, index-1, "0")
	for i := 0; i < offset; i++ {
		nodeName := NodeName(xi0, SO, i)
		// index is the last level; i.e., sinks
		parent := NodeName(butterfly0, 2*(index-1)-1, i)
		node := g.GetNode(nodeName)
		node.UpdateParents([]string{parent})
		g.WriteNode(node, nodeName)
	}

	// sinks of first xi to sources of second xi
	xi1 := GraphName(curGraph, posName, index-1, "1")
	for i := 0; i < offset; i++ {
		nodeName := NodeName(xi1, SO, i)
		parent := NodeName(xi0, SI, i)
		if index-1 == 0 {
			parent = NodeName(xi0, SO, i)
		}
		node := g.GetNode(nodeName)
		node.UpdateParents([]string{parent})
		g.WriteNode(node, nodeName)
	}

	// sinks of second xi to sources of second butterfly
	butterfly1 := GraphName(curGraph, "C", index-1, "1")
	for i := 0; i < offset; i++ {
		nodeName := NodeName(butterfly1, 0, i)
		parent := NodeName(xi1, SI, i)
		node := g.GetNode(nodeName)
		node.UpdateParents([]string{parent})
		g.WriteNode(node, nodeName)
	}

	// sinks of second butterfly to sinks
	for i := 0; i < offset; i++ {
		nodeName0 := NodeName(curGraph, SI, i)
		nodeName1 := NodeName(curGraph, SI, i+offset)
		parent := NodeName(butterfly1, 2*(index-1)-1, i)
		node0 := g.GetNode(nodeName0)
		node0.UpdateParents([]string{parent})
		node1 := g.GetNode(nodeName1)
		node1.UpdateParents([]string{parent})
		g.WriteNode(node0, nodeName0)
		g.WriteNode(node1, nodeName1)
	}

	// sources to sinks directly
	for i := 0; i < int(1<<uint(index)); i++ {
		nodeName := NodeName(curGraph, SI, i)
		parent := NodeName(curGraph, SO, i)
		node := g.GetNode(nodeName)
		node.UpdateParents([]string{parent})
		g.WriteNode(node, nodeName)
	}
}
