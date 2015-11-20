package block

import (
	"encoding/json"
	"github.com/kwonalbert/spacemint/pos"
)

const (
	Payment     = 0
	SpaceCommit = 1
	Punishment  = 2
)

type Transaction struct {
	t   int // transaction type
	tid int // unique transaction identifier

	// payment
	in  *In
	out *Out

	// spacecommit
	commit *pos.Commitment

	// punishment
	pk  []byte
	m   []byte
	j   int
	sig []byte
}

type In struct {
	tid int
	k   int    // indicating which benefactor
	sig []byte // signature of (transaction.tid, tid, k, out)
}

type Out struct {
	pk    []byte  // recipients pubkey
	coins float64 // amount of coin given
}

func (t *Transaction) MarshalBinary() ([]byte, error) {
	return json.Marshal(t)
}

func (t *Transaction) UnmarhsalBinary(data []byte) error {
	return json.Unmarshal(data, t)
}
