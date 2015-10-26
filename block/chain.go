package block

import (
	"errors"
	"os"
)

// Defines and implements block chain

type BlockChain struct {
	chain           *os.File          // block chain (in a single file)
	//probably should write this out to disk too?
	seekIndex       map[int]int64     // maps nth block to a seek index
	LastBlock       int               // last block that was added
}

func NewBlockChain(fn string) *BlockChain {
	f, err := os.Create(fn)
	if err != nil {
		panic(err)
	}

	bc := BlockChain{
		chain: f,
		seekIndex: make(map[int]int64),
		LastBlock: -1,
	}
	return &bc
}

// Add a block to end of chain
func (bc *BlockChain) Add(b *Block) error {
	bin, err := b.MarshalBinary()
	if err != nil {
		return err
	}

	n, err := bc.chain.Write(bin)
	if err != nil {
		return err
	} else if n != len(bin) {
		return errors.New("Couldn't write the whole block to chain.")
	}

	bc.LastBlock++
	if bc.LastBlock == 0 {
		bc.seekIndex[bc.LastBlock] = 0
	}
	bc.seekIndex[bc.LastBlock+1] = bc.seekIndex[bc.LastBlock] + int64(len(bin))
	return nil
}

// Find and return the ith block
func (bc *BlockChain) Read(i int) (*Block, error) {
	idx := bc.seekIndex[i]
	next, ok := bc.seekIndex[i+1]
	if !ok { //last block
		stat, err := bc.chain.Stat()
		if err != nil {
			panic(err)
		}
		next = stat.Size()
	}

	bin := make([]byte, next-idx)
	n, err := bc.chain.ReadAt(bin, idx)
	if err != nil {
		return nil, err
	} else if n != len(bin) {
		return nil, errors.New("Couldn't read the whole block from chain.")
	}
	b := new(Block)
	err = b.UnmarshalBinary(bin)
	return b, err
}
