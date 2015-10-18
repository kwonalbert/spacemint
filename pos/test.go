package pos

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/kwonalbert/spacecoin/util"
	"golang.org/x/crypto/sha3"
	"math/rand"
	"os"
)

var expHashes [][hashSize]byte = nil
var expMerkle [][hashSize]byte = nil
var expProof [][hashSize]byte = nil

func sampleGraph(pk []byte) [][]int{
	adj := [][]int{[]int{0, 0, 0, 0,},
		       []int{1, 0, 0, 0,},
		       []int{1, 0, 0, 0,},
           	       []int{1, 0, 1, 0,},}

	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, 3)
	val := append(pk, buf.Bytes() ...)
	expHashes[3] = sha3.Sum256(val)

	binary.Write(buf, binary.BigEndian, 1)
	val = append(pk, buf.Bytes() ...)
	expHashes[1] = sha3.Sum256(val)

	binary.Write(buf, binary.BigEndian, 2)
	val = append(buf.Bytes(), expHashes[3][:] ...)
	val = append(pk, val ...)
	expHashes[2] = sha3.Sum256(val)

	binary.Write(buf, binary.BigEndian, 0)
	val = buf.Bytes()
	val = append(val, expHashes[1][:] ...)
	val = append(val, expHashes[2][:] ...)
	val = append(val, expHashes[3][:] ...)
	val = append(pk, val ...)
	expHashes[0] = sha3.Sum256(val)


	for i := 0; i < 4; i++ {
		expMerkle[i+4] = expHashes[i]
	}

	val3 := append(expMerkle[6][:], expMerkle[7][:] ...)
	val3 = append(pk, val3 ...)
	expMerkle[3] = sha3.Sum256(val3)

	val2 := append(expMerkle[4][:], expMerkle[5][:] ...)
	val2 = append(pk, val2 ...)
	expMerkle[2] = sha3.Sum256(val2)

	val1 := append(expMerkle[2][:], expMerkle[3][:] ...)
	val1 = append(pk, val1 ...)
	expMerkle[1] = sha3.Sum256(val1)

	//testing for node 1
	expProof = make([][hashSize]byte, util.Log2(4)-1)
	expProof[0] = expMerkle[4]
	expProof[1] = expMerkle[3]

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

func setup(pk []byte, n int, graph string) {
	expHashes = make([][hashSize]byte, n)
	expMerkle = make([][hashSize]byte, 2*n)
	adj := sampleGraph(pk)
	setupGraph(adj, graph)
}
