package pos

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"golang.org/x/crypto/sha3"
	"math/rand"
	"os"
)

func sampleGraph() [][]int{
	adj := [][]int{[]int{0, 0, 0, 0,},
		       []int{1, 0, 0, 0,},
		       []int{1, 0, 0, 0,},
           	       []int{1, 0, 1, 0,},}

	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, 3)
	expHashes[3] = sha3.Sum256(buf.Bytes())

	binary.Write(buf, binary.BigEndian, 1)
	expHashes[1] = sha3.Sum256(buf.Bytes())

	binary.Write(buf, binary.BigEndian, 2)
	val := buf.Bytes()
	val = append(val, expHashes[3][:]...)
	expHashes[2] = sha3.Sum256(val)

	binary.Write(buf, binary.BigEndian, 0)
	val = buf.Bytes()
	val = append(val, expHashes[1][:] ...)
	val = append(val, expHashes[2][:] ...)
	val = append(val, expHashes[3][:] ...)
	expHashes[0] = sha3.Sum256(val)


	for i := 0; i < 4; i++ {
		expMerkle[i+4] = expHashes[i]
	}

	val3 := append(expMerkle[6][:], expMerkle[7][:] ...)
	expMerkle[3] = sha3.Sum256(val3)

	val2 := append(expMerkle[4][:], expMerkle[5][:] ...)
	expMerkle[2] = sha3.Sum256(val2)

	val1 := append(expMerkle[2][:], expMerkle[3][:] ...)
	expMerkle[1] = sha3.Sum256(val1)

	return adj
}

func randomGraph(n int) [][]int{
	rand.Seed(47576409822)
	// generate a random DAG
	adj := make([][]int, n)
	for i := range adj {
		adj[i] = make([]int, n)
	}

	for i := range adj {
		for j := range adj[i] {
			// lower triangular matrix => DAG
			if j >= i {
				break
			}
			r := rand.Float64()
			if r < 0.5 {
				adj[i][j] = 1
			} else {
				adj[i][j] = 0
			}
		}
	}
	return adj
}

// NOTE: this is NOT the graph that should be used in the final version
//       we need to generate a correct graph for PoS
func setupGraph(adj [][]int, graph string) {
	os.RemoveAll(graph)
	os.Mkdir(graph, 0777)
	for i := range adj {
		os.Mkdir(fmt.Sprintf("%s/%d", graph, i), 0777)
	}
	for i := range adj {
		for j := range adj[i] {
			if adj[i][j] == 0 {
				continue
			}
			// i points to j => i is parent of j
			err := os.Symlink(fmt.Sprintf("%s/%d", graph, i), fmt.Sprintf("%s/%d/%d", graph, j, i))
			if err != nil {
				fmt.Println(err)
			}
		}
	}

	fmt.Println("Graph")
	for i := range adj {
		fmt.Println(adj[i])
	}
}

func setup(n int, graph string) *Prover{
	expHashes = make([][hashSize]byte, n)
	expMerkle = make([][hashSize]byte, 2*n)
	adj := sampleGraph()
	setupGraph(adj, graph)
	return NewProver(nil, n, graph)
}
