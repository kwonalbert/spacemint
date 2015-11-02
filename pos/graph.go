package pos

import (
	"encoding/json"
	"fmt"
	"github.com/boltdb/bolt"
	//"runtime/pprof"
)

var graphBase string = "%s.%s%d-%d"
var nodeBase string = "%s.%d-%d"

const (
	SO = 0
	SI = 1
)

type Graph struct {
	fn string
	db *bolt.DB
}

type Node struct {
	I  int      // node id
	H  []byte   `json:",omitempty"` // hash at the file
	Ps []string `json:",omitempty"` // parent node files
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
	// cpuprofile := "cpu.prof"
	// f, _ := os.Create(cpuprofile)
	// pprof.StartCPUProfile(f)
	// defer pprof.StopCPUProfile()
	// recursively generate graphs
	count := 0

	db, err := bolt.Open(fn, 0600, nil)
	if err != nil {
		panic(err)
	}

	db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucket([]byte("nodes"))
		if err != nil {
			panic(err)
		}
		return nil
	})

	g := &Graph{
		fn: fn,
		db: db,
	}

	g.XiGraph(index, 0, name, &count)

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

// Gets the node, and update the node.
// Otherwise, create a node
func (g *Graph) GetNode(nodeName string) *Node {
	node := new(Node)
	var val []byte
	g.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("nodes"))
		val = b.Get([]byte(nodeName))
		return nil
	})
	if val != nil {
		node.UnmarshalBinary(val)
		return node
	} else {
		return nil
	}
}

func (g *Graph) WriteNode(n *Node, nodeName string) {
	b, err := n.MarshalBinary()
	if err != nil {
		panic(err)
	}
	g.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("nodes"))
		err := bucket.Put([]byte(nodeName), []byte(b))
		return err
	})
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
func IndexToNode(node, index, inst int, name string) string {
	sources := 1 << uint(index)
	firstButter := sources + numButterfly(index-1)
	firstXi := firstButter + numXi(index-1)
	secondXi := firstXi + numXi(index-1)
	secondButter := secondXi + numButterfly(index-1)
	sinks := secondButter + sources

	curGraph := fmt.Sprintf(graphBase, name, posName, index, inst)

	if node < sources {
		return fmt.Sprintf(nodeBase, curGraph, SO, node)
	} else if node >= sources && node < firstButter {
		node = node - sources
		butterfly0 := fmt.Sprintf(graphBase, curGraph, "C", index-1, 0)
		level := node / (1 << uint(index-1))
		nodeNum := node % (1 << uint(index-1))
		return fmt.Sprintf(nodeBase, butterfly0, level, nodeNum)
	} else if node >= firstButter && node < firstXi {
		node = node - firstButter
		return IndexToNode(node, index-1, 0, curGraph)
	} else if node >= firstXi && node < secondXi {
		node = node - firstXi
		return IndexToNode(node, index-1, 1, curGraph)
	} else if node >= secondXi && node < secondButter {
		node = node - secondXi
		butterfly1 := fmt.Sprintf(graphBase, curGraph, "C", index-1, 1)
		level := node / (1 << uint(index-1))
		nodeNum := node % (1 << uint(index-1))
		return fmt.Sprintf(nodeBase, butterfly1, level, nodeNum)
	} else if node >= secondButter && node < sinks {
		node = node - secondButter
		return fmt.Sprintf(nodeBase, curGraph, SI, node)
	} else {
		return ""
	}
}

func (g *Graph) ButterflyGraph(index, inst int, name, graph string, count *int) {
	curGraph := fmt.Sprintf(graphBase, graph, name, index, inst)
	numLevel := 2 * index
	for level := 0; level < numLevel; level++ {
		for i := 0; i < int(1<<uint(index)); i++ {
			// no parents at level 0
			nodeName := fmt.Sprintf(nodeBase, curGraph, level, i)
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
			prev1 := fmt.Sprintf("%d-%d", level-1, prev)
			prev2 := fmt.Sprintf("%d-%d", level-1, i)
			parent1 := fmt.Sprintf("%s.%s", curGraph, prev1)
			parent2 := fmt.Sprintf("%s.%s", curGraph, prev2)

			parents := []string{parent1, parent2}
			g.NewNode(nodeName, *count, nil, parents)
			*count++
		}
	}
}

func (g *Graph) XiGraph(index, inst int, graph string, count *int) {
	if index == 1 {
		g.ButterflyGraph(index, inst, posName, graph, count)
		return
	}
	curGraph := fmt.Sprintf(graphBase, graph, posName, index, inst)

	for i := 0; i < int(1<<uint(index)); i++ {
		// "SO" for sources
		nodeName := fmt.Sprintf(nodeBase, curGraph, SO, i)
		g.NewNode(nodeName, *count, nil, nil)
		*count++
	}

	// recursively generate graphs
	g.ButterflyGraph(index-1, 0, "C", curGraph, count)
	g.XiGraph(index-1, 0, curGraph, count)
	g.XiGraph(index-1, 1, curGraph, count)
	g.ButterflyGraph(index-1, 1, "C", curGraph, count)

	for i := 0; i < int(1<<uint(index)); i++ {
		// "SI" for sinks
		nodeName := fmt.Sprintf(nodeBase, curGraph, SI, i)
		g.NewNode(nodeName, *count, nil, nil)
		*count++
	}

	offset := int(1 << uint(index-1)) //2^(index-1)

	// sources to sources of first butterfly
	butterfly0 := fmt.Sprintf(graphBase, curGraph, "C", index-1, 0)
	for i := 0; i < offset; i++ {
		nodeName := fmt.Sprintf(nodeBase, butterfly0, 0, i)
		parent0 := fmt.Sprintf(nodeBase, curGraph, SO, i)
		parent1 := fmt.Sprintf(nodeBase, curGraph, SO, i+offset)
		node := g.GetNode(nodeName)
		node.UpdateParents([]string{parent0, parent1})
		g.WriteNode(node, nodeName)
	}

	// sinks of first butterfly to sources of first xi graph
	xi0 := fmt.Sprintf(graphBase, curGraph, posName, index-1, 0)
	for i := 0; i < offset; i++ {
		nodeName := fmt.Sprintf(nodeBase, xi0, SO, i)
		// index is the last level; i.e., sinks
		parent := fmt.Sprintf(nodeBase, butterfly0, 2*(index-1)-1, i)
		node := g.GetNode(nodeName)
		node.UpdateParents([]string{parent})
		g.WriteNode(node, nodeName)
	}

	// sinks of first xi to sources of second xi
	xi1 := fmt.Sprintf(graphBase, curGraph, posName, index-1, 1)
	for i := 0; i < offset; i++ {
		nodeName := fmt.Sprintf(nodeBase, xi1, SO, i)
		parent := fmt.Sprintf(nodeBase, xi0, SI, i)
		if index-1 == 0 {
			parent = fmt.Sprintf(nodeBase, xi0, SO, i)
		}
		node := g.GetNode(nodeName)
		node.UpdateParents([]string{parent})
		g.WriteNode(node, nodeName)
	}

	// sinks of second xi to sources of second butterfly
	butterfly1 := fmt.Sprintf(graphBase, curGraph, "C", index-1, 1)
	for i := 0; i < offset; i++ {
		nodeName := fmt.Sprintf(nodeBase, butterfly1, 0, i)
		parent := fmt.Sprintf(nodeBase, xi1, SI, i)
		node := g.GetNode(nodeName)
		node.UpdateParents([]string{parent})
		g.WriteNode(node, nodeName)
	}

	// sinks of second butterfly to sinks
	for i := 0; i < offset; i++ {
		nodeName0 := fmt.Sprintf(nodeBase, curGraph, SI, i)
		nodeName1 := fmt.Sprintf(nodeBase, curGraph, SI, i+offset)
		parent := fmt.Sprintf(nodeBase, butterfly1, 2*(index-1)-1, i)
		node0 := g.GetNode(nodeName0)
		node0.UpdateParents([]string{parent})
		node1 := g.GetNode(nodeName1)
		node1.UpdateParents([]string{parent})
		g.WriteNode(node0, nodeName0)
		g.WriteNode(node1, nodeName1)
	}

	// sources to sinks directly
	for i := 0; i < int(1<<uint(index)); i++ {
		nodeName := fmt.Sprintf(nodeBase, curGraph, SI, i)
		parent := fmt.Sprintf(nodeBase, curGraph, SO, i)
		node := g.GetNode(nodeName)
		node.UpdateParents([]string{parent})
		g.WriteNode(node, nodeName)
	}
}
