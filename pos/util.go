package pos

import (
)

const hashName = "hash"
const hashSize = 256/8

// log base 2
func log2(val int) int {
	r := 0;

	for ; val > 0; val = val >> 1 {
		r++;
	}
	return r
}
