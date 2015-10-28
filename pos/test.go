package pos

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/kwonalbert/spacecoin/util"
	"golang.org/x/crypto/sha3"
	"os"
)

var expHashes [][hashSize]byte = nil
var expMerkle [][hashSize]byte = nil
var expProof [][hashSize]byte = nil

func sampleGraph(pk []byte, index int) [][]int {
	adj := [][]int{[]int{0, 0, 0, 0, 0},
		[]int{1, 0, 0, 0, 0},
		[]int{1, 0, 0, 0, 0},
		[]int{1, 0, 1, 0, 0},
		[]int{0, 1, 0, 1, 0}}

	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, 4)
	val := append(pk, buf.Bytes()...)
	expHashes[4] = sha3.Sum256(val)

	buf = new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, 3)
	val = append(pk, buf.Bytes()...)
	val = append(val, expHashes[4][:]...)
	expHashes[3] = sha3.Sum256(val)

	binary.Write(buf, binary.BigEndian, 1)
	val = append(pk, buf.Bytes()...)
	val = append(pk, expHashes[4][:]...)
	expHashes[1] = sha3.Sum256(val)

	binary.Write(buf, binary.BigEndian, 2)
	val = append(buf.Bytes(), expHashes[3][:]...)
	val = append(pk, val...)
	expHashes[2] = sha3.Sum256(val)

	binary.Write(buf, binary.BigEndian, 0)
	val = buf.Bytes()
	val = append(val, expHashes[1][:]...)
	val = append(val, expHashes[2][:]...)
	val = append(val, expHashes[3][:]...)
	val = append(pk, val...)
	expHashes[0] = sha3.Sum256(val)

	for i := 0; i < len(adj); i++ {
		expMerkle[i+index] = expHashes[i]
	}

	val = append(expMerkle[14][:], expMerkle[15][:]...)
	val = append(pk, val...)
	expMerkle[7] = sha3.Sum256(val)

	val = append(expMerkle[12][:], expMerkle[13][:]...)
	val = append(pk, val...)
	expMerkle[6] = sha3.Sum256(val)

	val = append(expMerkle[10][:], expMerkle[11][:]...)
	val = append(pk, val...)
	expMerkle[5] = sha3.Sum256(val)

	val = append(expMerkle[8][:], expMerkle[9][:]...)
	val = append(pk, val...)
	expMerkle[4] = sha3.Sum256(val)

	val = append(expMerkle[6][:], expMerkle[7][:]...)
	val = append(pk, val...)
	expMerkle[3] = sha3.Sum256(val)

	val = append(expMerkle[4][:], expMerkle[5][:]...)
	val = append(pk, val...)
	expMerkle[2] = sha3.Sum256(val)

	val = append(expMerkle[2][:], expMerkle[3][:]...)
	val = append(pk, val...)
	expMerkle[1] = sha3.Sum256(val)

	//testing for node 1
	expProof[0] = expMerkle[8]
	expProof[1] = expMerkle[5]
	expProof[2] = expMerkle[3]

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
	wd, _ := os.Getwd()
	for i := range adj {
		for j := range adj[i] {
			if adj[i][j] == 0 {
				continue
			}
			// i points to j => i is parent of j
			err := os.Symlink(fmt.Sprintf("%s/%s/%d", wd, graph, i), fmt.Sprintf("%s/%d/%d", graph, j, i))
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

func Setup(pk []byte, size int, index int, graph string) {
	expHashes = make([][hashSize]byte, size)
	expMerkle = make([][hashSize]byte, 2*index)
	expProof = make([][hashSize]byte, util.Log2(index))
	adj := sampleGraph(pk, index)
	setupGraph(adj, graph)
}
