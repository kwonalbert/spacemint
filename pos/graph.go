package pos

import (
	"fmt"
	"os"
	//"runtime/pprof"
)

var graphBase string = "%s/%s%d-%d"
var nodeBase string = "%s/%d-%d"
var symBase string = "%s/%s/%d-%d"
var parentBase string = "%s%d-%d.%d-%d"
var countBase string = "%s/node%d"

const (
	SO = 0
	SI = 1
)

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
	//return fmt.Sprintf("%s/%d", dir, node)
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
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	graphDir := fmt.Sprintf(graphBase, dir, name, index, inst)
	_, err = os.Stat(graphDir)
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
			node := fmt.Sprintf(nodeBase, graphDir, level, i)
			err := os.Mkdir(node, 0777)
			if err != nil {
				panic(err)
			}

			f, err := os.Create(fmt.Sprintf(countBase, node, *count))
			if err != nil {
				panic(err)
			}
			f.Close()
			*count += 1

			// no parents at level 0
			if level == 0 {
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
			os.Symlink(fmt.Sprintf("%s/%s/%s", wd, graphDir, prev1),
				fmt.Sprintf("%s/%s", node, prev1))
			os.Symlink(fmt.Sprintf("%s/%s/%s", wd, graphDir, prev2),
				fmt.Sprintf("%s/%s", node, prev2))
		}
	}
}

func XiGraph(index int, inst int, dir string, count *int) {
	if index == 1 {
		ButterflyGraph(index, inst, "Xi", dir, count)
		return
	}
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	graphDir := fmt.Sprintf(graphBase, dir, "Xi", index, inst)
	err = os.Mkdir(graphDir, 0777)
	if err != nil {
		panic(err)
	}

	for i := 0; i < int(1<<uint(index)); i++ {
		// "0" for sources
		node := fmt.Sprintf(nodeBase, graphDir, SO, i)
		err = os.Mkdir(node, 0777)
		if err != nil {
			panic(err)
		}
		f, err := os.Create(fmt.Sprintf(countBase, node, *count))
		if err != nil {
			panic(err)
		}
		f.Close()
		*count += 1
	}

	// recursively generate graphs
	ButterflyGraph(index-1, 0, "C", graphDir, count)
	XiGraph(index-1, 0, graphDir, count)
	XiGraph(index-1, 1, graphDir, count)
	ButterflyGraph(index-1, 1, "C", graphDir, count)

	for i := 0; i < int(1<<uint(index)); i++ {
		// "1" for sinks
		node := fmt.Sprintf(nodeBase, graphDir, SI, i)
		err = os.Mkdir(node, 0777)
		if err != nil {
			panic(err)
		}
		f, err := os.Create(fmt.Sprintf(countBase, node, *count))
		if err != nil {
			panic(err)
		}
		f.Close()
		*count += 1
	}

	curGraph := fmt.Sprintf("%s%d-%d", "Xi", index, inst)
	offset := int(1 << uint(index-1)) //2^(index-1)

	// sources to sources of first butterfly
	butterfly0 := fmt.Sprintf(graphBase, graphDir, "C", index-1, 0)
	for i := 0; i < offset; i++ {
		node := fmt.Sprintf(nodeBase, butterfly0, 0, i)
		parent0 := fmt.Sprintf(symBase, wd, graphDir, SO, i)
		parent1 := fmt.Sprintf(symBase, wd, graphDir, SO, i+offset)
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
		parent := fmt.Sprintf(symBase, wd, butterfly0, 2*(index-1)-1, i)
		ln := fmt.Sprintf("%s.%s", curGraph, "C")
		pn := fmt.Sprintf(parentBase, ln, index-1, 0, 2*(index-1)-1, i)
		err = os.Symlink(parent, fmt.Sprintf("%s/%s", node, pn))
		if err != nil {
			panic(err)
		}
	}

	// sinks of first xi to sources of second xi
	xi1 := fmt.Sprintf(graphBase, graphDir, "Xi", index-1, 1)
	for i := 0; i < offset; i++ {
		node := fmt.Sprintf(nodeBase, xi1, SO, i)
		parent := fmt.Sprintf(symBase, wd, xi0, SI, i)
		pn := fmt.Sprintf(parentBase, "Xi", index-1, 0, SI, i)
		if index-1 == 0 {
			parent = fmt.Sprintf(symBase, wd, xi0, SO, i)
			pn = fmt.Sprintf(parentBase, "Xi", index-1, 0, SO, i)
		}
		err = os.Symlink(parent, fmt.Sprintf("%s/%s", node, pn))
		if err != nil {
			panic(err)
		}
	}

	// sinks of second xi to sources of second butterfly
	butterfly1 := fmt.Sprintf(graphBase, graphDir, "C", index-1, 1)
	for i := 0; i < offset; i++ {
		node := fmt.Sprintf(nodeBase, butterfly1, 0, i)
		parent := fmt.Sprintf(symBase, wd, xi1, SI, i)
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
		parent := fmt.Sprintf(symBase, wd, butterfly1, 2*(index-1)-1, i)
		ln := fmt.Sprintf("%s.%s", curGraph, "C")
		pn := fmt.Sprintf(parentBase, ln, index-1, 1, 2*(index-1)-1, i)
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
		parent := fmt.Sprintf(symBase, wd, graphDir, SO, i)
		pn := fmt.Sprintf(parentBase, "Xi", index, 0, SO, i)
		err = os.Symlink(parent, fmt.Sprintf("%s/%s", node, pn))
		if err != nil {
			panic(err)
		}
	}
}
