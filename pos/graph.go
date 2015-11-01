package pos

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	//"runtime/pprof"
)

var graphBase string = "%s/%s%d-%d"
var nodeBase string = "%s/%d-%d"
var symBase string = "%s/%d-%d"
var parentBase string = "%s%d-%d.%d-%d"
var countBase string = "%s/node%d"

const (
	SO = 0
	SI = 1
)

type Node struct {
	Id      int      // node id
	Hash    []byte   // hash at the file
	Parents []string // parent node files
}

func (n *Node) MarshalBinary() ([]byte, error) {
	return json.Marshal(n)
}

func (n *Node) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, n)
}

// Gets the node at nodeFile, and update the node.
// Otherwise, create a node
func GetNode(nodeFile string, id int, hash []byte, parents []string) *Node {
	node := new(Node)
	if nodeFile != "" {
		_, err := os.Stat(nodeFile)
		if err == nil { //file exists
			f, err := os.Open(nodeFile)
			if err != nil {
				panic(err)
			}
			b, err := ioutil.ReadAll(f)
			if err != nil {
				panic(err)
			}
			err = node.UnmarshalBinary(b)
			if err != nil {
				panic(err)
			}
			f.Close()
		}
	} else {
		node.Id = id
		node.Hash = hash
	}
	node.Parents = append(parents, node.Parents...)
	return node
}

func (n *Node) Write(nodeFile string) {
	b, err := n.MarshalBinary()
	if err != nil {
		panic(err)
	}
	f, err := os.Create(nodeFile)
	if err != nil {
		panic(err)
	}
	num, err := f.Write(b)
	if err != nil || num != len(b) {
		panic(err)
	}
	f.Close()
}

// Generate a new PoS graph of index
// Currently only supports the weaker PoS graph
// Note that this graph will have O(2^index) nodes
func NewGraph(index int, dir string) {
	// cpuprofile := "cpu.prof"
	// f, _ := os.Create(cpuprofile)
	// pprof.StartCPUProfile(f)
	// defer pprof.StopCPUProfile()

	// Be careful when calling this!
	os.RemoveAll(dir)
	os.Mkdir(dir, 0777)
	// recursively generate graphs
	count := 0
	XiGraph(index, 0, dir, &count)
}

func numXi(index int) int {
	return (1 << uint(index)) * (index + 1) * index
}

func numButterfly(index int) int {
	return 2 * (1 << uint(index)) * index
}

// Maps a node index (0 to O(2^N)) to a folder (a physical node)
func IndexToNode(node int, index int, inst int, dir string) string {
	sources := 1 << uint(index)
	firstButter := sources + numButterfly(index-1)
	firstXi := firstButter + numXi(index-1)
	secondXi := firstXi + numXi(index-1)
	secondButter := secondXi + numButterfly(index-1)
	sinks := secondButter + sources

	graphDir := fmt.Sprintf(graphBase, dir, "Xi", index, inst)

	if node < sources {
		return fmt.Sprintf(nodeBase, graphDir, SO, node)
	} else if node >= sources && node < firstButter {
		node = node - sources
		butterfly0 := fmt.Sprintf(graphBase, graphDir, "C", index-1, 0)
		level := node / (1 << uint(index-1))
		nodeNum := node % (1 << uint(index-1))
		return fmt.Sprintf(nodeBase, butterfly0, level, nodeNum)
	} else if node >= firstButter && node < firstXi {
		node = node - firstButter
		return IndexToNode(node, index-1, 0, graphDir)
	} else if node >= firstXi && node < secondXi {
		node = node - firstXi
		return IndexToNode(node, index-1, 1, graphDir)
	} else if node >= secondXi && node < secondButter {
		node = node - secondXi
		butterfly1 := fmt.Sprintf(graphBase, graphDir, "C", index-1, 1)
		level := node / (1 << uint(index-1))
		nodeNum := node % (1 << uint(index-1))
		return fmt.Sprintf(nodeBase, butterfly1, level, nodeNum)
	} else if node >= secondButter && node < sinks {
		node = node - secondButter
		return fmt.Sprintf(nodeBase, graphDir, SI, node)
	} else {
		return ""
	}
}

func ButterflyGraph(index int, inst int, name, dir string, count *int) {
	graphDir := fmt.Sprintf(graphBase, dir, name, index, inst)
	_, err := os.Stat(graphDir)
	if err == nil { // already created graph
		return
	}
	err = os.Mkdir(graphDir, 0777)
	if err != nil {
		panic(err)
	}
	numLevel := 2 * index
	for level := 0; level < numLevel; level++ {
		for i := 0; i < int(1<<uint(index)); i++ {
			// no parents at level 0
			nodeFile := fmt.Sprintf(nodeBase, graphDir, level, i)
			if level == 0 {
				node := GetNode("", *count, nil, nil)
				node.Write(nodeFile)
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
			parent1 := fmt.Sprintf("%s/%s", graphDir, prev1)
			parent2 := fmt.Sprintf("%s/%s", graphDir, prev2)

			parents := []string{parent1, parent2}
			node := GetNode("", *count, nil, parents)
			node.Write(nodeFile)
			*count++
		}
	}
}

func XiGraph(index int, inst int, dir string, count *int) {
	if index == 1 {
		ButterflyGraph(index, inst, "Xi", dir, count)
		return
	}
	graphDir := fmt.Sprintf(graphBase, dir, "Xi", index, inst)
	err := os.Mkdir(graphDir, 0777)
	if err != nil {
		panic(err)
	}

	for i := 0; i < int(1<<uint(index)); i++ {
		// "SO" for sources
		nodeFile := fmt.Sprintf(nodeBase, graphDir, SO, i)
		node := GetNode("", *count, nil, nil)
		node.Write(nodeFile)
		*count++
	}

	// recursively generate graphs
	ButterflyGraph(index-1, 0, "C", graphDir, count)
	XiGraph(index-1, 0, graphDir, count)
	XiGraph(index-1, 1, graphDir, count)
	ButterflyGraph(index-1, 1, "C", graphDir, count)

	for i := 0; i < int(1<<uint(index)); i++ {
		// "SI" for sinks
		nodeFile := fmt.Sprintf(nodeBase, graphDir, SI, i)
		node := GetNode("", *count, nil, nil)
		node.Write(nodeFile)
		*count++
	}

	offset := int(1 << uint(index-1)) //2^(index-1)

	// sources to sources of first butterfly
	butterfly0 := fmt.Sprintf(graphBase, graphDir, "C", index-1, 0)
	for i := 0; i < offset; i++ {
		nodeFile := fmt.Sprintf(nodeBase, butterfly0, 0, i)
		parent0 := fmt.Sprintf(symBase, graphDir, SO, i)
		parent1 := fmt.Sprintf(symBase, graphDir, SO, i+offset)
		node := GetNode(nodeFile, -1, nil, []string{parent0, parent1})
		node.Write(nodeFile)
	}

	// sinks of first butterfly to sources of first xi graph
	xi0 := fmt.Sprintf(graphBase, graphDir, "Xi", index-1, 0)
	for i := 0; i < offset; i++ {
		nodeFile := fmt.Sprintf(nodeBase, xi0, SO, i)
		// index is the last level; i.e., sinks
		parent := fmt.Sprintf(symBase, butterfly0, 2*(index-1)-1, i)
		node := GetNode(nodeFile, -1, nil, []string{parent})
		node.Write(nodeFile)
	}

	// sinks of first xi to sources of second xi
	xi1 := fmt.Sprintf(graphBase, graphDir, "Xi", index-1, 1)
	for i := 0; i < offset; i++ {
		nodeFile := fmt.Sprintf(nodeBase, xi1, SO, i)
		parent := fmt.Sprintf(symBase, xi0, SI, i)
		if index-1 == 0 {
			parent = fmt.Sprintf(symBase, xi0, SO, i)
		}
		node := GetNode(nodeFile, -1, nil, []string{parent})
		node.Write(nodeFile)
	}

	// sinks of second xi to sources of second butterfly
	butterfly1 := fmt.Sprintf(graphBase, graphDir, "C", index-1, 1)
	for i := 0; i < offset; i++ {
		nodeFile := fmt.Sprintf(nodeBase, butterfly1, 0, i)
		parent := fmt.Sprintf(symBase, xi1, SI, i)
		node := GetNode(nodeFile, -1, nil, []string{parent})
		node.Write(nodeFile)
	}

	// sinks of second butterfly to sinks
	for i := 0; i < offset; i++ {
		nodeFile0 := fmt.Sprintf(nodeBase, graphDir, SI, i)
		nodeFile1 := fmt.Sprintf(nodeBase, graphDir, SI, i+offset)
		parent := fmt.Sprintf(symBase, butterfly1, 2*(index-1)-1, i)
		node0 := GetNode(nodeFile0, -1, nil, []string{parent})
		node1 := GetNode(nodeFile1, -1, nil, []string{parent})
		node0.Write(nodeFile0)
		node1.Write(nodeFile1)
	}

	// sources to sinks directly
	for i := 0; i < int(1<<uint(index)); i++ {
		nodeFile := fmt.Sprintf(nodeBase, graphDir, SI, i)
		parent := fmt.Sprintf(symBase, graphDir, SO, i)
		node := GetNode(nodeFile, -1, nil, []string{parent})
		node.Write(nodeFile)
	}
}
