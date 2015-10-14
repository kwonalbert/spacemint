package pos

import (
)

// log base 2
func log2(val int) int {
	r := 0;

	for ; val > 0; val = val >> 1 {
		r++;
	}
	return r
}
