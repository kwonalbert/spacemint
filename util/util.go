package util

import (
	"bytes"
	"math/big"
)

// return: x^y
func Pow(x *big.Float, n int) *big.Float {
	res := new(big.Float).Copy(x)
	if n < 0 {
		res = res.Quo(big.NewFloat(1), res)
		n = -n
	} else if n == 0 {
		return big.NewFloat(1)
	}
	y := big.NewFloat(1)
	for i := n; i > 1; {
		if i%2 == 0 {
			i /= 2
		} else {
			y = y.Mul(res, y)
			i = (i - 1) / 2
		}
		res = res.Mul(res, res)
	}
	return res.Mul(res, y)
}

// Implements the nth root algorithm from
// https://en.wikipedia.org/wiki/Nth_root_algorithm
// return: nth root of x within some epsilon
func Root(x *big.Float, n int) *big.Float {
	guess := new(big.Float).Quo(x, big.NewFloat(float64(n)))
	diff := big.NewFloat(1)
	ep := big.NewFloat(0.00000001)
	abs := new(big.Float).Abs(diff)
	for abs.Cmp(ep) >= 0 {
		//fmt.Println(guess, abs)
		prev := Pow(guess, n-1)
		diff = new(big.Float).Quo(x, prev)
		diff = diff.Sub(diff, guess)
		diff = diff.Quo(diff, big.NewFloat(float64(n)))

		guess = guess.Add(guess, diff)
		abs = new(big.Float).Abs(diff)
	}
	return guess
}

// return: floor log base 2 of x
func Log2(x int64) int64 {
	var r int64 = 0
	for ; x > 1; x >>= 1 {
		r++
	}
	return r
}

func Concat(ls [][]byte) []byte {
	var res []byte
	for i := range ls {
		res = append(res, ls[i]...)
	}
	return res
}

func ConcatStr(ss ...string) string {
	buf := new(bytes.Buffer)
	for i := range ss {
		buf.WriteString(ss[i])
	}
	return buf.String()
}

//From hackers delight
func Count(x uint64) int {
	x = x - ((x >> 1) & 0x55555555)
	x = (x & 0x33333333) + ((x >> 2) & 0x33333333)
	x = (x + (x >> 4)) & 0x0F0F0F0F
	x = x + (x >> 8)
	x = x + (x >> 16)
	return int(x & 0x0000003F)
}
