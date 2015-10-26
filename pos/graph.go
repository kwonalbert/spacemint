package pos

import (
	"fmt"
	"os"
)

var graphBase string = "%s/%s%d-%d"
var nodeBase string = "%s/%d-%d"
var parentBase string = "%s%d-%d.%d-%d"

const (
	SO = 0
	SI = 1
)

// Generate a new PoS graph of index
// Currently only supports the weaker PoS graph
// Note that this graph will have O(2^index) nodes
func NewGraph(index int, dir string) {
	os.Mkdir(dir, 0666)
	// recursively generate graphs

}

func ButterflyGraph(index int, inst int, name, dir string) {
	graphDir := fmt.Sprintf(graphBase, dir, name, index, inst)
	_, err := os.Stat(graphDir)
	if err == nil { // already created graph
		return
	}
	err = os.Mkdir(graphDir, 0777)
	if err != nil {
		panic(err)
	}
	for level := 0; level < index+1; level++ {
		for i := 0; i < int(1<<uint(index)); i++ {
			node := fmt.Sprintf(nodeBase, graphDir, level, i)
			os.Mkdir(node, 0777)
			// no parents at level 0
			if level == 0 {
				continue
			}
			prev := 0
			if (i>>uint(level-1))&1 == 0 {
				prev = i + (1 << uint(level-1))
			} else {
				prev = i - (1 << uint(level-1))
			}
			prev1 := fmt.Sprintf("%d-%d", level-1, prev)
			prev2 := fmt.Sprintf("%d-%d", level-1, i)
			os.Symlink(fmt.Sprintf("%s/%s", graphDir, prev1),
				fmt.Sprintf("%s/%s", node, prev1))
			os.Symlink(fmt.Sprintf("%s/%s", graphDir, prev2),
				fmt.Sprintf("%s/%s", node, prev2))
		}
	}
}

func XiGraph(index int, inst int, dir string) {
	if index == 0 {
		ButterflyGraph(index, inst, "Xi", dir)
		return
	}
	graphDir := fmt.Sprintf(graphBase, dir, "Xi", index, inst)
	err := os.Mkdir(graphDir, 0777)
	if err != nil {
		panic(err)
	}

	// generate the two butterfly graphs on top and bottom
	ButterflyGraph(index-1, 0, "C", graphDir)
	ButterflyGraph(index-1, 1, "C", graphDir)

	// recursively generate XI graphs
	XiGraph(index-1, 0, graphDir)
	XiGraph(index-1, 1, graphDir)

	for i := 0; i < int(1<<uint(index)); i++ {
		// "0" for sources
		node := fmt.Sprintf(nodeBase, graphDir, SO, i)
		err = os.Mkdir(node, 0777)
		if err != nil {
			panic(err)
		}
		// "1" for sinks
		node = fmt.Sprintf(nodeBase, graphDir, SI, i)
		err = os.Mkdir(node, 0777)
		if err != nil {
			panic(err)
		}
	}

	offset := int(1 << uint(index-1)) //2^(index-1)

	curGraph := fmt.Sprintf("%s%d-%d", "Xi", index, inst)

	// sources to sources of first butterfly
	butterfly0 := fmt.Sprintf(graphBase, graphDir, "C", index-1, 0)
	for i := 0; i < offset; i++ {
		node := fmt.Sprintf(nodeBase, butterfly0, 0, i)
		parent0 := fmt.Sprintf(nodeBase, graphDir, SO, i)
		parent1 := fmt.Sprintf(nodeBase, graphDir, SO, i+offset)
		pn0 := fmt.Sprintf(parentBase, "Xi", index, 0, SO, i)
		pn1 := fmt.Sprintf(parentBase, "Xi", index, 0, SO, i+offset)
		err = os.Symlink(parent0, fmt.Sprintf("%s/%s", node, pn0))
		if err != nil {
			panic(err)
		}
		err = os.Symlink(parent1, fmt.Sprintf("%s/%s", node, pn1))
		if err != nil {
			panic(err)
		}
	}

	// sinks of first butterfly to sources of first xi graph
	xi0 := fmt.Sprintf(graphBase, graphDir, "Xi", index-1, 0)
	for i := 0; i < offset; i++ {
		node := fmt.Sprintf(nodeBase, xi0, SO, i)
		// index is the last level; i.e., sinks
		parent := fmt.Sprintf(nodeBase, butterfly0, index-1, i)
		ln := fmt.Sprintf("%s.%s", curGraph, "C")
		pn := fmt.Sprintf(parentBase, ln, index-1, 0, index-1, i)
		err = os.Symlink(parent, fmt.Sprintf("%s/%s", node, pn))
		if err != nil {
			panic(err)
		}
	}

	// sinks of first xi to sources of second xi
	xi1 := fmt.Sprintf(graphBase, graphDir, "Xi", index-1, 1)
	for i := 0; i < offset; i++ {
		node := fmt.Sprintf(nodeBase, xi1, SO, i)
		parent := fmt.Sprintf(nodeBase, xi0, SI, i)
		pn := fmt.Sprintf(parentBase, "Xi", index-1, 0, SI, i)
		err = os.Symlink(parent, fmt.Sprintf("%s/%s", node, pn))
		if err != nil {
			panic(err)
		}
	}

	// sinks of second xi to sources of second butterfly
	butterfly1 := fmt.Sprintf(graphBase, graphDir, "C", index-1, 1)
	for i := 0; i < offset; i++ {
		node := fmt.Sprintf(nodeBase, butterfly1, 0, i)
		parent := fmt.Sprintf(nodeBase, xi1, SI, i)
		pn := fmt.Sprintf(parentBase, "Xi", index-1, 1, SI, i)
		err = os.Symlink(parent, fmt.Sprintf("%s/%s", node, pn))
		if err != nil {
			panic(err)
		}
	}

	// sinks of second butterfly to sinks
	for i := 0; i < offset; i++ {
		node0 := fmt.Sprintf(nodeBase, graphDir, SI, i)
		node1 := fmt.Sprintf(nodeBase, graphDir, SI, i+offset)
		parent := fmt.Sprintf(nodeBase, butterfly1, index-1, i)
		ln := fmt.Sprintf("%s.%s", curGraph, "C")
		pn := fmt.Sprintf(parentBase, ln, index-1, 1, index-1, i)
		err = os.Symlink(parent, fmt.Sprintf("%s/%s", node0, pn))
		if err != nil {
			panic(err)
		}
		err = os.Symlink(parent, fmt.Sprintf("%s/%s", node1, pn))
		if err != nil {
			panic(err)
		}
	}

	// sources to sinks directly
	for i := 0; i < int(1<<uint(index)); i++ {
		node := fmt.Sprintf(nodeBase, graphDir, SI, i)
		parent := fmt.Sprintf(nodeBase, graphDir, SO, i)
		pn := fmt.Sprintf(parentBase, "Xi", index, 0, SO, i)
		err = os.Symlink(parent, fmt.Sprintf("%s/%s", node, pn))
		if err != nil {
			panic(err)
		}
	}
}
